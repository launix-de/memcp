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
				declaration := declarations["stream_emit"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
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
				ctx.SyncDesc(&d1)
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
				lbl0 := ctx.ReserveLabel()
				_ = lbl0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				d4 = JITPrepareScmerGoArg(ctx, d4)
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
				d7.NoHeapPointer = false
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
				ctx.SyncDesc(&d7)
				if d7.Loc == LocRegPair || d7.Loc == LocStackPair || d7.Loc == LocInputPair {
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
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				ctx.Coverage.NativeCalls++
				declaration := declarations["stream_window_reduce"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
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
				declaration := declarations["window_mut"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
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
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d86 JITValueDesc
				_ = d86
				var d87 JITValueDesc
				_ = d87
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
				var d90 JITValueDesc
				_ = d90
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
				var d99 JITValueDesc
				_ = d99
				var d100 JITValueDesc
				_ = d100
				var d102 JITValueDesc
				_ = d102
				var d103 JITValueDesc
				_ = d103
				var d104 JITValueDesc
				_ = d104
				var d105 JITValueDesc
				_ = d105
				var d106 JITValueDesc
				_ = d106
				var d163 JITValueDesc
				_ = d163
				var d164 JITValueDesc
				_ = d164
				var d165 JITValueDesc
				_ = d165
				var d225 JITValueDesc
				_ = d225
				var d226 JITValueDesc
				_ = d226
				var d227 JITValueDesc
				_ = d227
				var d228 JITValueDesc
				_ = d228
				var d231 JITValueDesc
				_ = d231
				var d294 JITValueDesc
				_ = d294
				var d295 JITValueDesc
				_ = d295
				var d296 JITValueDesc
				_ = d296
				var d364 JITValueDesc
				_ = d364
				var d365 JITValueDesc
				_ = d365
				var d366 JITValueDesc
				_ = d366
				var d367 JITValueDesc
				_ = d367
				var d368 JITValueDesc
				_ = d368
				var d369 JITValueDesc
				_ = d369
				var d370 JITValueDesc
				_ = d370
				var d371 JITValueDesc
				_ = d371
				var d447 JITValueDesc
				_ = d447
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
				var d454 JITValueDesc
				_ = d454
				var d455 JITValueDesc
				_ = d455
				var d456 JITValueDesc
				_ = d456
				var d457 JITValueDesc
				_ = d457
				var d458 JITValueDesc
				_ = d458
				var d459 JITValueDesc
				_ = d459
				var d461 JITValueDesc
				_ = d461
				var d462 JITValueDesc
				_ = d462
				var d463 JITValueDesc
				_ = d463
				var d464 JITValueDesc
				_ = d464
				var d465 JITValueDesc
				_ = d465
				var d466 JITValueDesc
				_ = d466
				var d467 JITValueDesc
				_ = d467
				var d468 JITValueDesc
				_ = d468
				var d469 JITValueDesc
				_ = d469
				var d470 JITValueDesc
				_ = d470
				var d471 JITValueDesc
				_ = d471
				var d472 JITValueDesc
				_ = d472
				var d473 JITValueDesc
				_ = d473
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [14]BBDescriptor
				bbs[7].PhiBase = int32(phiBase0) + int32(0)
				bbs[7].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 3}}, Count: 1})
				defer ctx.ReleaseRegisterHomes(registerHomes1)
				var r0 Reg
				phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK2 {
					r0 = registerHomes1.Registers[0]
				}
				var d3 JITValueDesc
				if phiHomeOK2 {
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
					ctx.BindReg(r0, &d3)
				} else {
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				}
				_ = d3
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbpos_0_11 := int32(-1)
				_ = bbpos_0_11
				lbl12 := ctx.ReserveLabel()
				_ = lbl12
				bbpos_0_12 := int32(-1)
				_ = bbpos_0_12
				lbl13 := ctx.ReserveLabel()
				_ = lbl13
				bbpos_0_13 := int32(-1)
				_ = bbpos_0_13
				lbl14 := ctx.ReserveLabel()
				_ = lbl14
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					d4 = args[0]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Type == tagSlice {
						d5 = jitKnownSliceHeader(ctx, &d4)
					} else {
						d5 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d4}, 3)
					}
					ctx.BindReg(d5.Reg, &d5)
					ctx.BindReg(d5.Reg2, &d5)
					ctx.BindReg(d5.Reg3, &d5)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					d6 = args[1]
					d6.ID = 0
					ctx.StabilizeDescForControlFlow(&d6)
					d7 = args[2]
					d7.ID = 0
					var d8 JITValueDesc
					if d7.Type == tagSlice {
						d8 = jitKnownSliceHeader(ctx, &d7)
					} else {
						d8 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d7}, 3)
					}
					ctx.BindReg(d8.Reg, &d8)
					ctx.BindReg(d8.Reg2, &d8)
					ctx.BindReg(d8.Reg3, &d8)
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.FreeDesc(&d7)
					var d9 JITValueDesc
					if d5.SliceSizeKnown {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.StackOff))}
					} else if d5.Loc == LocStackTriple {
						d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d9)
					var d10 JITValueDesc
					if d9.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d9.Imm.Int() < 3)}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d9.Reg, 3)
						ctx.EmitSetcc(r1, CondSignedLess)
						d10 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d10)
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
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d11.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					ctx.EmitJmp(lbl16)
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl3)
					ps14 := PhiState{General: true}
					ps14.OverlayValues = make([]JITValueDesc, 12)
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
					ps15.OverlayValues[3] = d3
					ps15.OverlayValues[4] = d4
					ps15.OverlayValues[5] = d5
					ps15.OverlayValues[6] = d6
					ps15.OverlayValues[7] = d7
					ps15.OverlayValues[8] = d8
					ps15.OverlayValues[9] = d9
					ps15.OverlayValues[10] = d10
					ps15.OverlayValues[11] = d11
					snap16 := d3
					snap17 := d4
					snap18 := d5
					snap19 := d6
					snap20 := d7
					snap21 := d8
					snap22 := d9
					snap23 := d10
					snap24 := d11
					alloc25 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps15)
					}
					ctx.RestoreAllocState(alloc25)
					d3 = snap16
					d4 = snap17
					d5 = snap18
					d6 = snap19
					d7 = snap20
					d8 = snap21
					d9 = snap22
					d10 = snap23
					d11 = snap24
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d28 = ctx.EmitSliceElementAddress(&d5, &d26, 16)
					ctx.EnsureDesc(&d28)
					r2 := ctx.AllocRegExcept(d28.Reg)
					ctx.EmitMovRegMem(r2, d28.Reg, 8)
					ctx.EmitMovRegMem(d28.Reg, d28.Reg, 0)
					d27 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d28.Reg, Reg2: r2}
					ctx.BindReg(d28.Reg, &d27)
					ctx.BindReg(r2, &d27)
					var d29 JITValueDesc
					if d27.Loc == LocImm {
						d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d27.Imm.Int())}
					} else if d27.Type == tagInt && d27.Loc == LocRegPair {
						ctx.FreeReg(d27.Reg)
						d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg2}
						ctx.BindReg(d27.Reg2, &d29)
						ctx.BindReg(d27.Reg2, &d29)
					} else if d27.Type == tagInt && d27.Loc == LocReg {
						d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg}
						ctx.BindReg(d27.Reg, &d29)
						ctx.BindReg(d27.Reg, &d29)
					} else {
						d29 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d27}, 1)
						d29.Type = tagInt
						ctx.BindReg(d29.Reg, &d29)
					}
					ctx.FreeDesc(&d27)
					ctx.EnsureDesc(&d29)
					ctx.EnsureDesc(&d29)
					ctx.StabilizeDescForControlFlow(&d29)
					d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					d33 = ctx.EmitSliceElementAddress(&d5, &d31, 16)
					ctx.EnsureDesc(&d33)
					r3 := ctx.AllocRegExcept(d33.Reg)
					ctx.EmitMovRegMem(r3, d33.Reg, 8)
					ctx.EmitMovRegMem(d33.Reg, d33.Reg, 0)
					d32 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d33.Reg, Reg2: r3}
					ctx.BindReg(d33.Reg, &d32)
					ctx.BindReg(r3, &d32)
					var d34 JITValueDesc
					if d32.Loc == LocImm {
						d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d32.Imm.Int())}
					} else if d32.Type == tagInt && d32.Loc == LocRegPair {
						ctx.FreeReg(d32.Reg)
						d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg2}
						ctx.BindReg(d32.Reg2, &d34)
						ctx.BindReg(d32.Reg2, &d34)
					} else if d32.Type == tagInt && d32.Loc == LocReg {
						d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg}
						ctx.BindReg(d32.Reg, &d34)
						ctx.BindReg(d32.Reg, &d34)
					} else {
						d34 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d32}, 1)
						d34.Type = tagInt
						ctx.BindReg(d34.Reg, &d34)
					}
					ctx.FreeDesc(&d32)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d34)
					ctx.StabilizeDescForControlFlow(&d34)
					d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					d38 = ctx.EmitSliceElementAddress(&d5, &d36, 16)
					ctx.EnsureDesc(&d38)
					r4 := ctx.AllocRegExcept(d38.Reg)
					ctx.EmitMovRegMem(r4, d38.Reg, 8)
					ctx.EmitMovRegMem(d38.Reg, d38.Reg, 0)
					d37 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d38.Reg, Reg2: r4}
					ctx.BindReg(d38.Reg, &d37)
					ctx.BindReg(r4, &d37)
					var d39 JITValueDesc
					if d37.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d37.Imm.Int())}
					} else if d37.Type == tagInt && d37.Loc == LocRegPair {
						ctx.FreeReg(d37.Reg)
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d37.Reg2}
						ctx.BindReg(d37.Reg2, &d39)
						ctx.BindReg(d37.Reg2, &d39)
					} else if d37.Type == tagInt && d37.Loc == LocReg {
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d37.Reg}
						ctx.BindReg(d37.Reg, &d39)
						ctx.BindReg(d37.Reg, &d39)
					} else {
						d39 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d37}, 1)
						d39.Type = tagInt
						ctx.BindReg(d39.Reg, &d39)
					}
					ctx.FreeDesc(&d37)
					ctx.EnsureDesc(&d39)
					ctx.EnsureDesc(&d39)
					ctx.StabilizeDescForControlFlow(&d39)
					d41 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(3)}
					var d42 JITValueDesc
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
						d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2}
						ctx.BindReg(d5.Reg2, &d42)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d41)
					ctx.EnsureDesc(&d42)
					var d44 JITValueDesc
					if d42.Loc == LocImm && d41.Loc == LocImm {
						d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d42.Imm.Int() - d41.Imm.Int())}
					} else {
						r5 := ctx.AllocReg()
						if d42.Loc == LocImm {
							ctx.EmitMovRegImm64(r5, uint64(d42.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r5, d42.Reg)
						}
						if d41.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d41.Imm.Int()))
							ctx.EmitSubInt64(r5, RegR11)
						} else {
							ctx.EmitSubInt64(r5, d41.Reg)
						}
						d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d44)
					}
					var d45 JITValueDesc
					r6 := ctx.EmitSliceDataAfterLow(&d5, &d41, 16)
					d45 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
					ctx.BindReg(r6, &d45)
					ctx.BindReg(r6, &d45)
					var d46 JITValueDesc
					var r7 Reg
					var r8 Reg
					ctx.SyncDesc(&d45)
					ctx.EnsureDesc(&d45)
					if d45.Loc == LocImm {
						r7 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, uint64(d45.Imm.Int()))
					} else {
						r7 = d45.Reg
					}
					ctx.ProtectReg(r7)
					ctx.SyncDesc(&d44)
					ctx.EnsureDesc(&d44)
					if d44.Loc == LocImm {
						r8 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r8, uint64(d44.Imm.Int()))
					} else {
						r8 = d44.Reg
					}
					ctx.ProtectReg(r8)
					r9 := ctx.EmitSliceCapAfterLow(&d5, &d41, r7, r8)
					ctx.UnprotectReg(r8)
					ctx.UnprotectReg(r7)
					d46 = JITValueDesc{Loc: LocRegTriple, Reg: r7, Reg2: r8, Reg3: r9}
					ctx.BindReg(r7, &d46)
					ctx.BindReg(r8, &d46)
					ctx.BindReg(r9, &d46)
					ctx.BindReg(r7, &d46)
					ctx.BindReg(r8, &d46)
					ctx.BindReg(r9, &d46)
					ctx.StabilizeDescForControlFlow(&d46)
					ctx.EnsureDesc(&d39)
					var d47 JITValueDesc
					if d39.Loc == LocImm {
						d47 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d39.Imm.Int() <= 0)}
					} else {
						r10 := ctx.AllocRegExcept(d39.Reg)
						ctx.EmitCmpRegImm32(d39.Reg, 0)
						ctx.EmitSetcc(r10, CondSignedLessOrEqual)
						d47 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
						ctx.BindReg(r10, &d47)
					}
					d48 = d47
					ctx.EnsureDesc(&d48)
					if d48.Loc != LocImm && d48.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d48.Loc == LocImm {
						if d48.Imm.Bool() {
							if ps.General {
							}
							ps49 := PhiState{General: ps.General}
							ps49.OverlayValues = make([]JITValueDesc, 49)
							ps49.OverlayValues[3] = d3
							ps49.OverlayValues[4] = d4
							ps49.OverlayValues[5] = d5
							ps49.OverlayValues[6] = d6
							ps49.OverlayValues[7] = d7
							ps49.OverlayValues[8] = d8
							ps49.OverlayValues[9] = d9
							ps49.OverlayValues[10] = d10
							ps49.OverlayValues[11] = d11
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
							ps49.OverlayValues[47] = d47
							ps49.OverlayValues[48] = d48
							return bbs[3].RenderPS(ps49)
						}
						if ps.General {
						}
						ps50 := PhiState{General: ps.General}
						ps50.OverlayValues = make([]JITValueDesc, 49)
						ps50.OverlayValues[3] = d3
						ps50.OverlayValues[4] = d4
						ps50.OverlayValues[5] = d5
						ps50.OverlayValues[6] = d6
						ps50.OverlayValues[7] = d7
						ps50.OverlayValues[8] = d8
						ps50.OverlayValues[9] = d9
						ps50.OverlayValues[10] = d10
						ps50.OverlayValues[11] = d11
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
						ps50.OverlayValues[47] = d47
						ps50.OverlayValues[48] = d48
						return bbs[6].RenderPS(ps50)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d48.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					ctx.EmitJmp(lbl18)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl7)
					ps51 := PhiState{General: true}
					ps51.OverlayValues = make([]JITValueDesc, 49)
					ps51.OverlayValues[3] = d3
					ps51.OverlayValues[4] = d4
					ps51.OverlayValues[5] = d5
					ps51.OverlayValues[6] = d6
					ps51.OverlayValues[7] = d7
					ps51.OverlayValues[8] = d8
					ps51.OverlayValues[9] = d9
					ps51.OverlayValues[10] = d10
					ps51.OverlayValues[11] = d11
					ps51.OverlayValues[26] = d26
					ps51.OverlayValues[27] = d27
					ps51.OverlayValues[28] = d28
					ps51.OverlayValues[29] = d29
					ps51.OverlayValues[30] = d30
					ps51.OverlayValues[31] = d31
					ps51.OverlayValues[32] = d32
					ps51.OverlayValues[33] = d33
					ps51.OverlayValues[34] = d34
					ps51.OverlayValues[35] = d35
					ps51.OverlayValues[36] = d36
					ps51.OverlayValues[37] = d37
					ps51.OverlayValues[38] = d38
					ps51.OverlayValues[39] = d39
					ps51.OverlayValues[40] = d40
					ps51.OverlayValues[41] = d41
					ps51.OverlayValues[42] = d42
					ps51.OverlayValues[43] = d43
					ps51.OverlayValues[44] = d44
					ps51.OverlayValues[45] = d45
					ps51.OverlayValues[46] = d46
					ps51.OverlayValues[47] = d47
					ps51.OverlayValues[48] = d48
					ps52 := PhiState{General: true}
					ps52.OverlayValues = make([]JITValueDesc, 49)
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[4] = d4
					ps52.OverlayValues[5] = d5
					ps52.OverlayValues[6] = d6
					ps52.OverlayValues[7] = d7
					ps52.OverlayValues[8] = d8
					ps52.OverlayValues[9] = d9
					ps52.OverlayValues[10] = d10
					ps52.OverlayValues[11] = d11
					ps52.OverlayValues[26] = d26
					ps52.OverlayValues[27] = d27
					ps52.OverlayValues[28] = d28
					ps52.OverlayValues[29] = d29
					ps52.OverlayValues[30] = d30
					ps52.OverlayValues[31] = d31
					ps52.OverlayValues[32] = d32
					ps52.OverlayValues[33] = d33
					ps52.OverlayValues[34] = d34
					ps52.OverlayValues[35] = d35
					ps52.OverlayValues[36] = d36
					ps52.OverlayValues[37] = d37
					ps52.OverlayValues[38] = d38
					ps52.OverlayValues[39] = d39
					ps52.OverlayValues[40] = d40
					ps52.OverlayValues[41] = d41
					ps52.OverlayValues[42] = d42
					ps52.OverlayValues[43] = d43
					ps52.OverlayValues[44] = d44
					ps52.OverlayValues[45] = d45
					ps52.OverlayValues[46] = d46
					ps52.OverlayValues[47] = d47
					ps52.OverlayValues[48] = d48
					snap53 := d3
					snap54 := d4
					snap55 := d5
					snap56 := d6
					snap57 := d7
					snap58 := d8
					snap59 := d9
					snap60 := d10
					snap61 := d11
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
					snap83 := d47
					snap84 := d48
					alloc85 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps52)
					}
					ctx.RestoreAllocState(alloc85)
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d6 = snap56
					d7 = snap57
					d8 = snap58
					d9 = snap59
					d10 = snap60
					d11 = snap61
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
					d47 = snap83
					d48 = snap84
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps51)
					}
					return result
					ctx.FreeDesc(&d47)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d39)
					var d86 JITValueDesc
					ctx.EnsureDesc(&d46)
					if d46.Loc == LocRegPair || d46.Loc == LocRegTriple {
						d86 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg2}
						ctx.BindReg(d46.Reg2, &d86)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d46)
					ctx.EnsureDesc(&d39)
					ctx.EnsureDesc(&d86)
					var d88 JITValueDesc
					if d86.Loc == LocImm && d39.Loc == LocImm {
						d88 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d86.Imm.Int() - d39.Imm.Int())}
					} else {
						r11 := ctx.AllocReg()
						if d86.Loc == LocImm {
							ctx.EmitMovRegImm64(r11, uint64(d86.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r11, d86.Reg)
						}
						if d39.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d39.Imm.Int()))
							ctx.EmitSubInt64(r11, RegR11)
						} else {
							ctx.EmitSubInt64(r11, d39.Reg)
						}
						d88 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
						ctx.BindReg(r11, &d88)
					}
					var d89 JITValueDesc
					r12 := ctx.EmitSliceDataAfterLow(&d46, &d39, 16)
					d89 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
					ctx.BindReg(r12, &d89)
					ctx.BindReg(r12, &d89)
					var d90 JITValueDesc
					var r13 Reg
					var r14 Reg
					ctx.SyncDesc(&d89)
					ctx.EnsureDesc(&d89)
					if d89.Loc == LocImm {
						r13 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r13, uint64(d89.Imm.Int()))
					} else {
						r13 = d89.Reg
					}
					ctx.ProtectReg(r13)
					ctx.SyncDesc(&d88)
					ctx.EnsureDesc(&d88)
					if d88.Loc == LocImm {
						r14 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r14, uint64(d88.Imm.Int()))
					} else {
						r14 = d88.Reg
					}
					ctx.ProtectReg(r14)
					r15 := ctx.EmitSliceCapAfterLow(&d46, &d39, r13, r14)
					ctx.UnprotectReg(r14)
					ctx.UnprotectReg(r13)
					d90 = JITValueDesc{Loc: LocRegTriple, Reg: r13, Reg2: r14, Reg3: r15}
					ctx.BindReg(r13, &d90)
					ctx.BindReg(r14, &d90)
					ctx.BindReg(r15, &d90)
					ctx.BindReg(r13, &d90)
					ctx.BindReg(r14, &d90)
					ctx.BindReg(r15, &d90)
					ctx.EnsureDesc(&d46)
					ctx.EnsureDesc(&d90)
					ctx.EnsureDesc(&d46)
					ctx.EnsureDesc(&d90)
					callResults91 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d46, d90}, []uint8{1}, []uint8{0})
					d92 = callResults91[0]
					d92.Type = tagInt
					var d93 JITValueDesc
					if d46.SliceSizeKnown {
						d93 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d46.KnownSliceLen))}
					} else if d46.Loc == LocImm {
						d93 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d46.StackOff))}
					} else if d46.Loc == LocStackTriple {
						d93 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d46.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d46)
						if d46.Loc == LocRegPair || d46.Loc == LocRegTriple {
							d93 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg2, ID: 0}
						} else if d46.Loc == LocReg {
							d93 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d93)
					ctx.EnsureDesc(&d39)
					ctx.EnsureDescsTogether(&d93, &d39)
					var d94 JITValueDesc
					if d93.Loc == LocImm && d39.Loc == LocImm {
						d94 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d93.Imm.Int() - d39.Imm.Int())}
					} else if d39.Loc == LocImm && d39.Imm.Int() == 0 {
						r16 := ctx.AllocRegExcept(d93.Reg)
						ctx.EmitMovRegReg(r16, d93.Reg)
						d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
						ctx.BindReg(r16, &d94)
					} else if d93.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d39.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d93.Imm.Int()))
						ctx.EmitSubInt64(scratch, d39.Reg)
						d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d94)
					} else if d39.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d93.Reg)
						ctx.EmitMovRegReg(scratch, d93.Reg)
						if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d39.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d39.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d94)
					} else {
						r17 := ctx.AllocRegExcept(d93.Reg, d39.Reg)
						ctx.EmitMovRegReg(r17, d93.Reg)
						ctx.EmitSubInt64(r17, d39.Reg)
						d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
						ctx.BindReg(r17, &d94)
					}
					if d94.Loc == LocReg && d93.Loc == LocReg && d94.Reg == d93.Reg {
						ctx.TransferReg(d93.Reg)
						d93.Loc = LocNone
					}
					ctx.FreeDesc(&d93)
					ctx.EnsureDesc(&d94)
					var d95 JITValueDesc
					ctx.EnsureDesc(&d46)
					if d46.Loc == LocRegPair || d46.Loc == LocRegTriple {
						d95 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg2}
						ctx.BindReg(d46.Reg2, &d95)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d46)
					ctx.EnsureDesc(&d94)
					ctx.EnsureDesc(&d95)
					var d97 JITValueDesc
					if d95.Loc == LocImm && d94.Loc == LocImm {
						d97 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d95.Imm.Int() - d94.Imm.Int())}
					} else {
						r18 := ctx.AllocReg()
						if d95.Loc == LocImm {
							ctx.EmitMovRegImm64(r18, uint64(d95.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r18, d95.Reg)
						}
						if d94.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d94.Imm.Int()))
							ctx.EmitSubInt64(r18, RegR11)
						} else {
							ctx.EmitSubInt64(r18, d94.Reg)
						}
						d97 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d97)
					}
					var d98 JITValueDesc
					r19 := ctx.EmitSliceDataAfterLow(&d46, &d94, 16)
					d98 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
					ctx.BindReg(r19, &d98)
					ctx.BindReg(r19, &d98)
					var d99 JITValueDesc
					var r20 Reg
					var r21 Reg
					ctx.SyncDesc(&d98)
					ctx.EnsureDesc(&d98)
					if d98.Loc == LocImm {
						r20 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r20, uint64(d98.Imm.Int()))
					} else {
						r20 = d98.Reg
					}
					ctx.ProtectReg(r20)
					ctx.SyncDesc(&d97)
					ctx.EnsureDesc(&d97)
					if d97.Loc == LocImm {
						r21 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r21, uint64(d97.Imm.Int()))
					} else {
						r21 = d97.Reg
					}
					ctx.ProtectReg(r21)
					r22 := ctx.EmitSliceCapAfterLow(&d46, &d94, r20, r21)
					ctx.UnprotectReg(r21)
					ctx.UnprotectReg(r20)
					d99 = JITValueDesc{Loc: LocRegTriple, Reg: r20, Reg2: r21, Reg3: r22}
					ctx.BindReg(r20, &d99)
					ctx.BindReg(r21, &d99)
					ctx.BindReg(r22, &d99)
					ctx.BindReg(r20, &d99)
					ctx.BindReg(r21, &d99)
					ctx.BindReg(r22, &d99)
					ctx.StabilizeDescForControlFlow(&d99)
					ctx.FreeDesc(&d94)
					var d100 JITValueDesc
					if d99.SliceSizeKnown {
						d100 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d99.KnownSliceLen))}
					} else if d99.Loc == LocImm {
						d100 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d99.StackOff))}
					} else if d99.Loc == LocStackTriple {
						d100 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d99.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d99)
						if d99.Loc == LocRegPair || d99.Loc == LocRegTriple {
							d100 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d99.Reg2, ID: 0}
						} else if d99.Loc == LocReg {
							d100 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d99.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d100)
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[7].PhiBase)+int32(0))
						}
					}
					ps101 := PhiState{General: ps.General}
					ps101.OverlayValues = make([]JITValueDesc, 101)
					ps101.OverlayValues[3] = d3
					ps101.OverlayValues[4] = d4
					ps101.OverlayValues[5] = d5
					ps101.OverlayValues[6] = d6
					ps101.OverlayValues[7] = d7
					ps101.OverlayValues[8] = d8
					ps101.OverlayValues[9] = d9
					ps101.OverlayValues[10] = d10
					ps101.OverlayValues[11] = d11
					ps101.OverlayValues[26] = d26
					ps101.OverlayValues[27] = d27
					ps101.OverlayValues[28] = d28
					ps101.OverlayValues[29] = d29
					ps101.OverlayValues[30] = d30
					ps101.OverlayValues[31] = d31
					ps101.OverlayValues[32] = d32
					ps101.OverlayValues[33] = d33
					ps101.OverlayValues[34] = d34
					ps101.OverlayValues[35] = d35
					ps101.OverlayValues[36] = d36
					ps101.OverlayValues[37] = d37
					ps101.OverlayValues[38] = d38
					ps101.OverlayValues[39] = d39
					ps101.OverlayValues[40] = d40
					ps101.OverlayValues[41] = d41
					ps101.OverlayValues[42] = d42
					ps101.OverlayValues[43] = d43
					ps101.OverlayValues[44] = d44
					ps101.OverlayValues[45] = d45
					ps101.OverlayValues[46] = d46
					ps101.OverlayValues[47] = d47
					ps101.OverlayValues[48] = d48
					ps101.OverlayValues[86] = d86
					ps101.OverlayValues[87] = d87
					ps101.OverlayValues[88] = d88
					ps101.OverlayValues[89] = d89
					ps101.OverlayValues[90] = d90
					ps101.OverlayValues[92] = d92
					ps101.OverlayValues[93] = d93
					ps101.OverlayValues[94] = d94
					ps101.OverlayValues[95] = d95
					ps101.OverlayValues[96] = d96
					ps101.OverlayValues[97] = d97
					ps101.OverlayValues[98] = d98
					ps101.OverlayValues[99] = d99
					ps101.OverlayValues[100] = d100
					ps101.PhiValues = make([]JITValueDesc, 1)
					d102 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps101.PhiValues[0] = d102
					if ps101.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps101)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
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
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
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
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					ctx.ReclaimUntrackedRegs()
					var d103 JITValueDesc
					if d46.SliceSizeKnown {
						d103 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d46.KnownSliceLen))}
					} else if d46.Loc == LocImm {
						d103 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d46.StackOff))}
					} else if d46.Loc == LocStackTriple {
						d103 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d46.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d46)
						if d46.Loc == LocRegPair || d46.Loc == LocRegTriple {
							d103 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg2, ID: 0}
						} else if d46.Loc == LocReg {
							d103 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d103)
					ctx.EnsureDesc(&d39)
					var d104 JITValueDesc
					if d103.Loc == LocImm && d39.Loc == LocImm {
						d104 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d103.Imm.Int() % d39.Imm.Int())}
					} else {
						d104 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntRem), []JITValueDesc{d103, d39}, 1)
					}
					if d104.Loc == LocReg && d103.Loc == LocReg && d104.Reg == d103.Reg {
						ctx.TransferReg(d103.Reg)
						d103.Loc = LocNone
					}
					ctx.FreeDesc(&d103)
					ctx.FreeDesc(&d39)
					ctx.EnsureDesc(&d104)
					var d105 JITValueDesc
					if d104.Loc == LocImm {
						d105 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d104.Imm.Int() != 0)}
					} else {
						r23 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d104.Reg, 0)
						ctx.EmitSetcc(r23, CondNotEqual)
						d105 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
						ctx.BindReg(r23, &d105)
					}
					ctx.FreeDesc(&d104)
					d106 = d105
					ctx.EnsureDesc(&d106)
					if d106.Loc != LocImm && d106.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d106.Loc == LocImm {
						if d106.Imm.Bool() {
							if ps.General {
							}
							ps107 := PhiState{General: ps.General}
							ps107.OverlayValues = make([]JITValueDesc, 107)
							ps107.OverlayValues[3] = d3
							ps107.OverlayValues[4] = d4
							ps107.OverlayValues[5] = d5
							ps107.OverlayValues[6] = d6
							ps107.OverlayValues[7] = d7
							ps107.OverlayValues[8] = d8
							ps107.OverlayValues[9] = d9
							ps107.OverlayValues[10] = d10
							ps107.OverlayValues[11] = d11
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
							ps107.OverlayValues[47] = d47
							ps107.OverlayValues[48] = d48
							ps107.OverlayValues[86] = d86
							ps107.OverlayValues[87] = d87
							ps107.OverlayValues[88] = d88
							ps107.OverlayValues[89] = d89
							ps107.OverlayValues[90] = d90
							ps107.OverlayValues[92] = d92
							ps107.OverlayValues[93] = d93
							ps107.OverlayValues[94] = d94
							ps107.OverlayValues[95] = d95
							ps107.OverlayValues[96] = d96
							ps107.OverlayValues[97] = d97
							ps107.OverlayValues[98] = d98
							ps107.OverlayValues[99] = d99
							ps107.OverlayValues[100] = d100
							ps107.OverlayValues[102] = d102
							ps107.OverlayValues[103] = d103
							ps107.OverlayValues[104] = d104
							ps107.OverlayValues[105] = d105
							ps107.OverlayValues[106] = d106
							return bbs[3].RenderPS(ps107)
						}
						if ps.General {
						}
						ps108 := PhiState{General: ps.General}
						ps108.OverlayValues = make([]JITValueDesc, 107)
						ps108.OverlayValues[3] = d3
						ps108.OverlayValues[4] = d4
						ps108.OverlayValues[5] = d5
						ps108.OverlayValues[6] = d6
						ps108.OverlayValues[7] = d7
						ps108.OverlayValues[8] = d8
						ps108.OverlayValues[9] = d9
						ps108.OverlayValues[10] = d10
						ps108.OverlayValues[11] = d11
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
						ps108.OverlayValues[47] = d47
						ps108.OverlayValues[48] = d48
						ps108.OverlayValues[86] = d86
						ps108.OverlayValues[87] = d87
						ps108.OverlayValues[88] = d88
						ps108.OverlayValues[89] = d89
						ps108.OverlayValues[90] = d90
						ps108.OverlayValues[92] = d92
						ps108.OverlayValues[93] = d93
						ps108.OverlayValues[94] = d94
						ps108.OverlayValues[95] = d95
						ps108.OverlayValues[96] = d96
						ps108.OverlayValues[97] = d97
						ps108.OverlayValues[98] = d98
						ps108.OverlayValues[99] = d99
						ps108.OverlayValues[100] = d100
						ps108.OverlayValues[102] = d102
						ps108.OverlayValues[103] = d103
						ps108.OverlayValues[104] = d104
						ps108.OverlayValues[105] = d105
						ps108.OverlayValues[106] = d106
						return bbs[4].RenderPS(ps108)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d106.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl19)
					ctx.EmitJmp(lbl20)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl5)
					ps109 := PhiState{General: true}
					ps109.OverlayValues = make([]JITValueDesc, 107)
					ps109.OverlayValues[3] = d3
					ps109.OverlayValues[4] = d4
					ps109.OverlayValues[5] = d5
					ps109.OverlayValues[6] = d6
					ps109.OverlayValues[7] = d7
					ps109.OverlayValues[8] = d8
					ps109.OverlayValues[9] = d9
					ps109.OverlayValues[10] = d10
					ps109.OverlayValues[11] = d11
					ps109.OverlayValues[26] = d26
					ps109.OverlayValues[27] = d27
					ps109.OverlayValues[28] = d28
					ps109.OverlayValues[29] = d29
					ps109.OverlayValues[30] = d30
					ps109.OverlayValues[31] = d31
					ps109.OverlayValues[32] = d32
					ps109.OverlayValues[33] = d33
					ps109.OverlayValues[34] = d34
					ps109.OverlayValues[35] = d35
					ps109.OverlayValues[36] = d36
					ps109.OverlayValues[37] = d37
					ps109.OverlayValues[38] = d38
					ps109.OverlayValues[39] = d39
					ps109.OverlayValues[40] = d40
					ps109.OverlayValues[41] = d41
					ps109.OverlayValues[42] = d42
					ps109.OverlayValues[43] = d43
					ps109.OverlayValues[44] = d44
					ps109.OverlayValues[45] = d45
					ps109.OverlayValues[46] = d46
					ps109.OverlayValues[47] = d47
					ps109.OverlayValues[48] = d48
					ps109.OverlayValues[86] = d86
					ps109.OverlayValues[87] = d87
					ps109.OverlayValues[88] = d88
					ps109.OverlayValues[89] = d89
					ps109.OverlayValues[90] = d90
					ps109.OverlayValues[92] = d92
					ps109.OverlayValues[93] = d93
					ps109.OverlayValues[94] = d94
					ps109.OverlayValues[95] = d95
					ps109.OverlayValues[96] = d96
					ps109.OverlayValues[97] = d97
					ps109.OverlayValues[98] = d98
					ps109.OverlayValues[99] = d99
					ps109.OverlayValues[100] = d100
					ps109.OverlayValues[102] = d102
					ps109.OverlayValues[103] = d103
					ps109.OverlayValues[104] = d104
					ps109.OverlayValues[105] = d105
					ps109.OverlayValues[106] = d106
					ps110 := PhiState{General: true}
					ps110.OverlayValues = make([]JITValueDesc, 107)
					ps110.OverlayValues[3] = d3
					ps110.OverlayValues[4] = d4
					ps110.OverlayValues[5] = d5
					ps110.OverlayValues[6] = d6
					ps110.OverlayValues[7] = d7
					ps110.OverlayValues[8] = d8
					ps110.OverlayValues[9] = d9
					ps110.OverlayValues[10] = d10
					ps110.OverlayValues[11] = d11
					ps110.OverlayValues[26] = d26
					ps110.OverlayValues[27] = d27
					ps110.OverlayValues[28] = d28
					ps110.OverlayValues[29] = d29
					ps110.OverlayValues[30] = d30
					ps110.OverlayValues[31] = d31
					ps110.OverlayValues[32] = d32
					ps110.OverlayValues[33] = d33
					ps110.OverlayValues[34] = d34
					ps110.OverlayValues[35] = d35
					ps110.OverlayValues[36] = d36
					ps110.OverlayValues[37] = d37
					ps110.OverlayValues[38] = d38
					ps110.OverlayValues[39] = d39
					ps110.OverlayValues[40] = d40
					ps110.OverlayValues[41] = d41
					ps110.OverlayValues[42] = d42
					ps110.OverlayValues[43] = d43
					ps110.OverlayValues[44] = d44
					ps110.OverlayValues[45] = d45
					ps110.OverlayValues[46] = d46
					ps110.OverlayValues[47] = d47
					ps110.OverlayValues[48] = d48
					ps110.OverlayValues[86] = d86
					ps110.OverlayValues[87] = d87
					ps110.OverlayValues[88] = d88
					ps110.OverlayValues[89] = d89
					ps110.OverlayValues[90] = d90
					ps110.OverlayValues[92] = d92
					ps110.OverlayValues[93] = d93
					ps110.OverlayValues[94] = d94
					ps110.OverlayValues[95] = d95
					ps110.OverlayValues[96] = d96
					ps110.OverlayValues[97] = d97
					ps110.OverlayValues[98] = d98
					ps110.OverlayValues[99] = d99
					ps110.OverlayValues[100] = d100
					ps110.OverlayValues[102] = d102
					ps110.OverlayValues[103] = d103
					ps110.OverlayValues[104] = d104
					ps110.OverlayValues[105] = d105
					ps110.OverlayValues[106] = d106
					snap111 := d3
					snap112 := d4
					snap113 := d5
					snap114 := d6
					snap115 := d7
					snap116 := d8
					snap117 := d9
					snap118 := d10
					snap119 := d11
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
					snap141 := d47
					snap142 := d48
					snap143 := d86
					snap144 := d87
					snap145 := d88
					snap146 := d89
					snap147 := d90
					snap148 := d92
					snap149 := d93
					snap150 := d94
					snap151 := d95
					snap152 := d96
					snap153 := d97
					snap154 := d98
					snap155 := d99
					snap156 := d100
					snap157 := d102
					snap158 := d103
					snap159 := d104
					snap160 := d105
					snap161 := d106
					alloc162 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps110)
					}
					ctx.RestoreAllocState(alloc162)
					d3 = snap111
					d4 = snap112
					d5 = snap113
					d6 = snap114
					d7 = snap115
					d8 = snap116
					d9 = snap117
					d10 = snap118
					d11 = snap119
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
					d47 = snap141
					d48 = snap142
					d86 = snap143
					d87 = snap144
					d88 = snap145
					d89 = snap146
					d90 = snap147
					d92 = snap148
					d93 = snap149
					d94 = snap150
					d95 = snap151
					d96 = snap152
					d97 = snap153
					d98 = snap154
					d99 = snap155
					d100 = snap156
					d102 = snap157
					d103 = snap158
					d104 = snap159
					d105 = snap160
					d106 = snap161
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps109)
					}
					return result
					ctx.FreeDesc(&d105)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
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
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
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
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
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
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					ctx.ReclaimUntrackedRegs()
					var d163 JITValueDesc
					if d46.SliceSizeKnown {
						d163 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d46.KnownSliceLen))}
					} else if d46.Loc == LocImm {
						d163 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d46.StackOff))}
					} else if d46.Loc == LocStackTriple {
						d163 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d46.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d46)
						if d46.Loc == LocRegPair || d46.Loc == LocRegTriple {
							d163 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg2, ID: 0}
						} else if d46.Loc == LocReg {
							d163 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d163)
					var d164 JITValueDesc
					if d163.Loc == LocImm {
						d164 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d163.Imm.Int() == 0)}
					} else {
						r24 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d163.Reg, 0)
						ctx.EmitSetcc(r24, CondEqual)
						d164 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
						ctx.BindReg(r24, &d164)
					}
					ctx.FreeDesc(&d163)
					d165 = d164
					ctx.EnsureDesc(&d165)
					if d165.Loc != LocImm && d165.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d165.Loc == LocImm {
						if d165.Imm.Bool() {
							if ps.General {
							}
							ps166 := PhiState{General: ps.General}
							ps166.OverlayValues = make([]JITValueDesc, 166)
							ps166.OverlayValues[3] = d3
							ps166.OverlayValues[4] = d4
							ps166.OverlayValues[5] = d5
							ps166.OverlayValues[6] = d6
							ps166.OverlayValues[7] = d7
							ps166.OverlayValues[8] = d8
							ps166.OverlayValues[9] = d9
							ps166.OverlayValues[10] = d10
							ps166.OverlayValues[11] = d11
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
							ps166.OverlayValues[47] = d47
							ps166.OverlayValues[48] = d48
							ps166.OverlayValues[86] = d86
							ps166.OverlayValues[87] = d87
							ps166.OverlayValues[88] = d88
							ps166.OverlayValues[89] = d89
							ps166.OverlayValues[90] = d90
							ps166.OverlayValues[92] = d92
							ps166.OverlayValues[93] = d93
							ps166.OverlayValues[94] = d94
							ps166.OverlayValues[95] = d95
							ps166.OverlayValues[96] = d96
							ps166.OverlayValues[97] = d97
							ps166.OverlayValues[98] = d98
							ps166.OverlayValues[99] = d99
							ps166.OverlayValues[100] = d100
							ps166.OverlayValues[102] = d102
							ps166.OverlayValues[103] = d103
							ps166.OverlayValues[104] = d104
							ps166.OverlayValues[105] = d105
							ps166.OverlayValues[106] = d106
							ps166.OverlayValues[163] = d163
							ps166.OverlayValues[164] = d164
							ps166.OverlayValues[165] = d165
							return bbs[3].RenderPS(ps166)
						}
						if ps.General {
						}
						ps167 := PhiState{General: ps.General}
						ps167.OverlayValues = make([]JITValueDesc, 166)
						ps167.OverlayValues[3] = d3
						ps167.OverlayValues[4] = d4
						ps167.OverlayValues[5] = d5
						ps167.OverlayValues[6] = d6
						ps167.OverlayValues[7] = d7
						ps167.OverlayValues[8] = d8
						ps167.OverlayValues[9] = d9
						ps167.OverlayValues[10] = d10
						ps167.OverlayValues[11] = d11
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
						ps167.OverlayValues[47] = d47
						ps167.OverlayValues[48] = d48
						ps167.OverlayValues[86] = d86
						ps167.OverlayValues[87] = d87
						ps167.OverlayValues[88] = d88
						ps167.OverlayValues[89] = d89
						ps167.OverlayValues[90] = d90
						ps167.OverlayValues[92] = d92
						ps167.OverlayValues[93] = d93
						ps167.OverlayValues[94] = d94
						ps167.OverlayValues[95] = d95
						ps167.OverlayValues[96] = d96
						ps167.OverlayValues[97] = d97
						ps167.OverlayValues[98] = d98
						ps167.OverlayValues[99] = d99
						ps167.OverlayValues[100] = d100
						ps167.OverlayValues[102] = d102
						ps167.OverlayValues[103] = d103
						ps167.OverlayValues[104] = d104
						ps167.OverlayValues[105] = d105
						ps167.OverlayValues[106] = d106
						ps167.OverlayValues[163] = d163
						ps167.OverlayValues[164] = d164
						ps167.OverlayValues[165] = d165
						return bbs[5].RenderPS(ps167)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl21 := ctx.ReserveLabel()
					lbl22 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d165.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl21)
					ctx.EmitJmp(lbl22)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl6)
					ps168 := PhiState{General: true}
					ps168.OverlayValues = make([]JITValueDesc, 166)
					ps168.OverlayValues[3] = d3
					ps168.OverlayValues[4] = d4
					ps168.OverlayValues[5] = d5
					ps168.OverlayValues[6] = d6
					ps168.OverlayValues[7] = d7
					ps168.OverlayValues[8] = d8
					ps168.OverlayValues[9] = d9
					ps168.OverlayValues[10] = d10
					ps168.OverlayValues[11] = d11
					ps168.OverlayValues[26] = d26
					ps168.OverlayValues[27] = d27
					ps168.OverlayValues[28] = d28
					ps168.OverlayValues[29] = d29
					ps168.OverlayValues[30] = d30
					ps168.OverlayValues[31] = d31
					ps168.OverlayValues[32] = d32
					ps168.OverlayValues[33] = d33
					ps168.OverlayValues[34] = d34
					ps168.OverlayValues[35] = d35
					ps168.OverlayValues[36] = d36
					ps168.OverlayValues[37] = d37
					ps168.OverlayValues[38] = d38
					ps168.OverlayValues[39] = d39
					ps168.OverlayValues[40] = d40
					ps168.OverlayValues[41] = d41
					ps168.OverlayValues[42] = d42
					ps168.OverlayValues[43] = d43
					ps168.OverlayValues[44] = d44
					ps168.OverlayValues[45] = d45
					ps168.OverlayValues[46] = d46
					ps168.OverlayValues[47] = d47
					ps168.OverlayValues[48] = d48
					ps168.OverlayValues[86] = d86
					ps168.OverlayValues[87] = d87
					ps168.OverlayValues[88] = d88
					ps168.OverlayValues[89] = d89
					ps168.OverlayValues[90] = d90
					ps168.OverlayValues[92] = d92
					ps168.OverlayValues[93] = d93
					ps168.OverlayValues[94] = d94
					ps168.OverlayValues[95] = d95
					ps168.OverlayValues[96] = d96
					ps168.OverlayValues[97] = d97
					ps168.OverlayValues[98] = d98
					ps168.OverlayValues[99] = d99
					ps168.OverlayValues[100] = d100
					ps168.OverlayValues[102] = d102
					ps168.OverlayValues[103] = d103
					ps168.OverlayValues[104] = d104
					ps168.OverlayValues[105] = d105
					ps168.OverlayValues[106] = d106
					ps168.OverlayValues[163] = d163
					ps168.OverlayValues[164] = d164
					ps168.OverlayValues[165] = d165
					ps169 := PhiState{General: true}
					ps169.OverlayValues = make([]JITValueDesc, 166)
					ps169.OverlayValues[3] = d3
					ps169.OverlayValues[4] = d4
					ps169.OverlayValues[5] = d5
					ps169.OverlayValues[6] = d6
					ps169.OverlayValues[7] = d7
					ps169.OverlayValues[8] = d8
					ps169.OverlayValues[9] = d9
					ps169.OverlayValues[10] = d10
					ps169.OverlayValues[11] = d11
					ps169.OverlayValues[26] = d26
					ps169.OverlayValues[27] = d27
					ps169.OverlayValues[28] = d28
					ps169.OverlayValues[29] = d29
					ps169.OverlayValues[30] = d30
					ps169.OverlayValues[31] = d31
					ps169.OverlayValues[32] = d32
					ps169.OverlayValues[33] = d33
					ps169.OverlayValues[34] = d34
					ps169.OverlayValues[35] = d35
					ps169.OverlayValues[36] = d36
					ps169.OverlayValues[37] = d37
					ps169.OverlayValues[38] = d38
					ps169.OverlayValues[39] = d39
					ps169.OverlayValues[40] = d40
					ps169.OverlayValues[41] = d41
					ps169.OverlayValues[42] = d42
					ps169.OverlayValues[43] = d43
					ps169.OverlayValues[44] = d44
					ps169.OverlayValues[45] = d45
					ps169.OverlayValues[46] = d46
					ps169.OverlayValues[47] = d47
					ps169.OverlayValues[48] = d48
					ps169.OverlayValues[86] = d86
					ps169.OverlayValues[87] = d87
					ps169.OverlayValues[88] = d88
					ps169.OverlayValues[89] = d89
					ps169.OverlayValues[90] = d90
					ps169.OverlayValues[92] = d92
					ps169.OverlayValues[93] = d93
					ps169.OverlayValues[94] = d94
					ps169.OverlayValues[95] = d95
					ps169.OverlayValues[96] = d96
					ps169.OverlayValues[97] = d97
					ps169.OverlayValues[98] = d98
					ps169.OverlayValues[99] = d99
					ps169.OverlayValues[100] = d100
					ps169.OverlayValues[102] = d102
					ps169.OverlayValues[103] = d103
					ps169.OverlayValues[104] = d104
					ps169.OverlayValues[105] = d105
					ps169.OverlayValues[106] = d106
					ps169.OverlayValues[163] = d163
					ps169.OverlayValues[164] = d164
					ps169.OverlayValues[165] = d165
					snap170 := d3
					snap171 := d4
					snap172 := d5
					snap173 := d6
					snap174 := d7
					snap175 := d8
					snap176 := d9
					snap177 := d10
					snap178 := d11
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
					snap200 := d47
					snap201 := d48
					snap202 := d86
					snap203 := d87
					snap204 := d88
					snap205 := d89
					snap206 := d90
					snap207 := d92
					snap208 := d93
					snap209 := d94
					snap210 := d95
					snap211 := d96
					snap212 := d97
					snap213 := d98
					snap214 := d99
					snap215 := d100
					snap216 := d102
					snap217 := d103
					snap218 := d104
					snap219 := d105
					snap220 := d106
					snap221 := d163
					snap222 := d164
					snap223 := d165
					alloc224 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps169)
					}
					ctx.RestoreAllocState(alloc224)
					d3 = snap170
					d4 = snap171
					d5 = snap172
					d6 = snap173
					d7 = snap174
					d8 = snap175
					d9 = snap176
					d10 = snap177
					d11 = snap178
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
					d47 = snap200
					d48 = snap201
					d86 = snap202
					d87 = snap203
					d88 = snap204
					d89 = snap205
					d90 = snap206
					d92 = snap207
					d93 = snap208
					d94 = snap209
					d95 = snap210
					d96 = snap211
					d97 = snap212
					d98 = snap213
					d99 = snap214
					d100 = snap215
					d102 = snap216
					d103 = snap217
					d104 = snap218
					d105 = snap219
					d106 = snap220
					d163 = snap221
					d164 = snap222
					d165 = snap223
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps168)
					}
					return result
					ctx.FreeDesc(&d164)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d225 := ps.PhiValues[0]
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d225)
							} else {
								ctx.EmitStoreToStack(d225, int32(bbs[7].PhiBase)+int32(0))
							}
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
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
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
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
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
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
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != LocNone {
						d225 = ps.OverlayValues[225]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					var d226 JITValueDesc
					if d3.Loc == LocImm {
						d226 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d226 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d226)
					}
					if d226.Loc == LocReg && d3.Loc == LocReg && d226.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d226)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d226)
					ctx.EnsureDesc(&d100)
					ctx.EnsureDescsTogether(&d226, &d100)
					var d227 JITValueDesc
					if d226.Loc == LocImm && d100.Loc == LocImm {
						d227 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d226.Imm.Int() < d100.Imm.Int())}
					} else if d100.Loc == LocImm {
						r25 := ctx.AllocRegExcept(d226.Reg)
						if d100.Imm.Int() >= -2147483648 && d100.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d226.Reg, int32(d100.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d100.Imm.Int()))
							ctx.EmitCmpInt64(d226.Reg, RegR11)
						}
						ctx.EmitSetcc(r25, CondSignedLess)
						d227 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r25}
						ctx.BindReg(r25, &d227)
					} else if d226.Loc == LocImm {
						r26 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d226.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d100.Reg)
						ctx.EmitSetcc(r26, CondSignedLess)
						d227 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r26}
						ctx.BindReg(r26, &d227)
					} else {
						r27 := ctx.AllocRegExcept(d226.Reg)
						ctx.EmitCmpInt64(d226.Reg, d100.Reg)
						ctx.EmitSetcc(r27, CondSignedLess)
						d227 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r27}
						ctx.BindReg(r27, &d227)
					}
					ctx.FreeDesc(&d100)
					d228 = d227
					ctx.EnsureDesc(&d228)
					if d228.Loc != LocImm && d228.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d228.Loc == LocImm {
						if d228.Imm.Bool() {
							if ps.General {
							}
							ps229 := PhiState{General: ps.General}
							ps229.OverlayValues = make([]JITValueDesc, 229)
							ps229.OverlayValues[3] = d3
							ps229.OverlayValues[4] = d4
							ps229.OverlayValues[5] = d5
							ps229.OverlayValues[6] = d6
							ps229.OverlayValues[7] = d7
							ps229.OverlayValues[8] = d8
							ps229.OverlayValues[9] = d9
							ps229.OverlayValues[10] = d10
							ps229.OverlayValues[11] = d11
							ps229.OverlayValues[26] = d26
							ps229.OverlayValues[27] = d27
							ps229.OverlayValues[28] = d28
							ps229.OverlayValues[29] = d29
							ps229.OverlayValues[30] = d30
							ps229.OverlayValues[31] = d31
							ps229.OverlayValues[32] = d32
							ps229.OverlayValues[33] = d33
							ps229.OverlayValues[34] = d34
							ps229.OverlayValues[35] = d35
							ps229.OverlayValues[36] = d36
							ps229.OverlayValues[37] = d37
							ps229.OverlayValues[38] = d38
							ps229.OverlayValues[39] = d39
							ps229.OverlayValues[40] = d40
							ps229.OverlayValues[41] = d41
							ps229.OverlayValues[42] = d42
							ps229.OverlayValues[43] = d43
							ps229.OverlayValues[44] = d44
							ps229.OverlayValues[45] = d45
							ps229.OverlayValues[46] = d46
							ps229.OverlayValues[47] = d47
							ps229.OverlayValues[48] = d48
							ps229.OverlayValues[86] = d86
							ps229.OverlayValues[87] = d87
							ps229.OverlayValues[88] = d88
							ps229.OverlayValues[89] = d89
							ps229.OverlayValues[90] = d90
							ps229.OverlayValues[92] = d92
							ps229.OverlayValues[93] = d93
							ps229.OverlayValues[94] = d94
							ps229.OverlayValues[95] = d95
							ps229.OverlayValues[96] = d96
							ps229.OverlayValues[97] = d97
							ps229.OverlayValues[98] = d98
							ps229.OverlayValues[99] = d99
							ps229.OverlayValues[100] = d100
							ps229.OverlayValues[102] = d102
							ps229.OverlayValues[103] = d103
							ps229.OverlayValues[104] = d104
							ps229.OverlayValues[105] = d105
							ps229.OverlayValues[106] = d106
							ps229.OverlayValues[163] = d163
							ps229.OverlayValues[164] = d164
							ps229.OverlayValues[165] = d165
							ps229.OverlayValues[225] = d225
							ps229.OverlayValues[226] = d226
							ps229.OverlayValues[227] = d227
							ps229.OverlayValues[228] = d228
							return bbs[8].RenderPS(ps229)
						}
						if ps.General {
						}
						ps230 := PhiState{General: ps.General}
						ps230.OverlayValues = make([]JITValueDesc, 229)
						ps230.OverlayValues[3] = d3
						ps230.OverlayValues[4] = d4
						ps230.OverlayValues[5] = d5
						ps230.OverlayValues[6] = d6
						ps230.OverlayValues[7] = d7
						ps230.OverlayValues[8] = d8
						ps230.OverlayValues[9] = d9
						ps230.OverlayValues[10] = d10
						ps230.OverlayValues[11] = d11
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
						ps230.OverlayValues[47] = d47
						ps230.OverlayValues[48] = d48
						ps230.OverlayValues[86] = d86
						ps230.OverlayValues[87] = d87
						ps230.OverlayValues[88] = d88
						ps230.OverlayValues[89] = d89
						ps230.OverlayValues[90] = d90
						ps230.OverlayValues[92] = d92
						ps230.OverlayValues[93] = d93
						ps230.OverlayValues[94] = d94
						ps230.OverlayValues[95] = d95
						ps230.OverlayValues[96] = d96
						ps230.OverlayValues[97] = d97
						ps230.OverlayValues[98] = d98
						ps230.OverlayValues[99] = d99
						ps230.OverlayValues[100] = d100
						ps230.OverlayValues[102] = d102
						ps230.OverlayValues[103] = d103
						ps230.OverlayValues[104] = d104
						ps230.OverlayValues[105] = d105
						ps230.OverlayValues[106] = d106
						ps230.OverlayValues[163] = d163
						ps230.OverlayValues[164] = d164
						ps230.OverlayValues[165] = d165
						ps230.OverlayValues[225] = d225
						ps230.OverlayValues[226] = d226
						ps230.OverlayValues[227] = d227
						ps230.OverlayValues[228] = d228
						return bbs[9].RenderPS(ps230)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d231 := ps.PhiValues[0]
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d231)
							} else {
								ctx.EmitStoreToStack(d231, int32(bbs[7].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					lbl23 := ctx.ReserveLabel()
					lbl24 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d228.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl23)
					ctx.EmitJmp(lbl24)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl10)
					ps232 := PhiState{General: true}
					ps232.OverlayValues = make([]JITValueDesc, 232)
					ps232.OverlayValues[3] = d3
					ps232.OverlayValues[4] = d4
					ps232.OverlayValues[5] = d5
					ps232.OverlayValues[6] = d6
					ps232.OverlayValues[7] = d7
					ps232.OverlayValues[8] = d8
					ps232.OverlayValues[9] = d9
					ps232.OverlayValues[10] = d10
					ps232.OverlayValues[11] = d11
					ps232.OverlayValues[26] = d26
					ps232.OverlayValues[27] = d27
					ps232.OverlayValues[28] = d28
					ps232.OverlayValues[29] = d29
					ps232.OverlayValues[30] = d30
					ps232.OverlayValues[31] = d31
					ps232.OverlayValues[32] = d32
					ps232.OverlayValues[33] = d33
					ps232.OverlayValues[34] = d34
					ps232.OverlayValues[35] = d35
					ps232.OverlayValues[36] = d36
					ps232.OverlayValues[37] = d37
					ps232.OverlayValues[38] = d38
					ps232.OverlayValues[39] = d39
					ps232.OverlayValues[40] = d40
					ps232.OverlayValues[41] = d41
					ps232.OverlayValues[42] = d42
					ps232.OverlayValues[43] = d43
					ps232.OverlayValues[44] = d44
					ps232.OverlayValues[45] = d45
					ps232.OverlayValues[46] = d46
					ps232.OverlayValues[47] = d47
					ps232.OverlayValues[48] = d48
					ps232.OverlayValues[86] = d86
					ps232.OverlayValues[87] = d87
					ps232.OverlayValues[88] = d88
					ps232.OverlayValues[89] = d89
					ps232.OverlayValues[90] = d90
					ps232.OverlayValues[92] = d92
					ps232.OverlayValues[93] = d93
					ps232.OverlayValues[94] = d94
					ps232.OverlayValues[95] = d95
					ps232.OverlayValues[96] = d96
					ps232.OverlayValues[97] = d97
					ps232.OverlayValues[98] = d98
					ps232.OverlayValues[99] = d99
					ps232.OverlayValues[100] = d100
					ps232.OverlayValues[102] = d102
					ps232.OverlayValues[103] = d103
					ps232.OverlayValues[104] = d104
					ps232.OverlayValues[105] = d105
					ps232.OverlayValues[106] = d106
					ps232.OverlayValues[163] = d163
					ps232.OverlayValues[164] = d164
					ps232.OverlayValues[165] = d165
					ps232.OverlayValues[225] = d225
					ps232.OverlayValues[226] = d226
					ps232.OverlayValues[227] = d227
					ps232.OverlayValues[228] = d228
					ps232.OverlayValues[231] = d231
					ps233 := PhiState{General: true}
					ps233.OverlayValues = make([]JITValueDesc, 232)
					ps233.OverlayValues[3] = d3
					ps233.OverlayValues[4] = d4
					ps233.OverlayValues[5] = d5
					ps233.OverlayValues[6] = d6
					ps233.OverlayValues[7] = d7
					ps233.OverlayValues[8] = d8
					ps233.OverlayValues[9] = d9
					ps233.OverlayValues[10] = d10
					ps233.OverlayValues[11] = d11
					ps233.OverlayValues[26] = d26
					ps233.OverlayValues[27] = d27
					ps233.OverlayValues[28] = d28
					ps233.OverlayValues[29] = d29
					ps233.OverlayValues[30] = d30
					ps233.OverlayValues[31] = d31
					ps233.OverlayValues[32] = d32
					ps233.OverlayValues[33] = d33
					ps233.OverlayValues[34] = d34
					ps233.OverlayValues[35] = d35
					ps233.OverlayValues[36] = d36
					ps233.OverlayValues[37] = d37
					ps233.OverlayValues[38] = d38
					ps233.OverlayValues[39] = d39
					ps233.OverlayValues[40] = d40
					ps233.OverlayValues[41] = d41
					ps233.OverlayValues[42] = d42
					ps233.OverlayValues[43] = d43
					ps233.OverlayValues[44] = d44
					ps233.OverlayValues[45] = d45
					ps233.OverlayValues[46] = d46
					ps233.OverlayValues[47] = d47
					ps233.OverlayValues[48] = d48
					ps233.OverlayValues[86] = d86
					ps233.OverlayValues[87] = d87
					ps233.OverlayValues[88] = d88
					ps233.OverlayValues[89] = d89
					ps233.OverlayValues[90] = d90
					ps233.OverlayValues[92] = d92
					ps233.OverlayValues[93] = d93
					ps233.OverlayValues[94] = d94
					ps233.OverlayValues[95] = d95
					ps233.OverlayValues[96] = d96
					ps233.OverlayValues[97] = d97
					ps233.OverlayValues[98] = d98
					ps233.OverlayValues[99] = d99
					ps233.OverlayValues[100] = d100
					ps233.OverlayValues[102] = d102
					ps233.OverlayValues[103] = d103
					ps233.OverlayValues[104] = d104
					ps233.OverlayValues[105] = d105
					ps233.OverlayValues[106] = d106
					ps233.OverlayValues[163] = d163
					ps233.OverlayValues[164] = d164
					ps233.OverlayValues[165] = d165
					ps233.OverlayValues[225] = d225
					ps233.OverlayValues[226] = d226
					ps233.OverlayValues[227] = d227
					ps233.OverlayValues[228] = d228
					ps233.OverlayValues[231] = d231
					snap234 := d3
					snap235 := d4
					snap236 := d5
					snap237 := d6
					snap238 := d7
					snap239 := d8
					snap240 := d9
					snap241 := d10
					snap242 := d11
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
					snap264 := d47
					snap265 := d48
					snap266 := d86
					snap267 := d87
					snap268 := d88
					snap269 := d89
					snap270 := d90
					snap271 := d92
					snap272 := d93
					snap273 := d94
					snap274 := d95
					snap275 := d96
					snap276 := d97
					snap277 := d98
					snap278 := d99
					snap279 := d100
					snap280 := d102
					snap281 := d103
					snap282 := d104
					snap283 := d105
					snap284 := d106
					snap285 := d163
					snap286 := d164
					snap287 := d165
					snap288 := d225
					snap289 := d226
					snap290 := d227
					snap291 := d228
					snap292 := d231
					alloc293 := ctx.SnapshotAllocState()
					if !bbs[9].Rendered {
						bbs[9].RenderPS(ps233)
					}
					ctx.RestoreAllocState(alloc293)
					d3 = snap234
					d4 = snap235
					d5 = snap236
					d6 = snap237
					d7 = snap238
					d8 = snap239
					d9 = snap240
					d10 = snap241
					d11 = snap242
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
					d47 = snap264
					d48 = snap265
					d86 = snap266
					d87 = snap267
					d88 = snap268
					d89 = snap269
					d90 = snap270
					d92 = snap271
					d93 = snap272
					d94 = snap273
					d95 = snap274
					d96 = snap275
					d97 = snap276
					d98 = snap277
					d99 = snap278
					d100 = snap279
					d102 = snap280
					d103 = snap281
					d104 = snap282
					d105 = snap283
					d106 = snap284
					d163 = snap285
					d164 = snap286
					d165 = snap287
					d225 = snap288
					d226 = snap289
					d227 = snap290
					d228 = snap291
					d231 = snap292
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps232)
					}
					return result
					ctx.FreeDesc(&d227)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
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
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
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
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
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
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != LocNone {
						d225 = ps.OverlayValues[225]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != LocNone {
						d227 = ps.OverlayValues[227]
					}
					if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != LocNone {
						d228 = ps.OverlayValues[228]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					ctx.ReclaimUntrackedRegs()
					var d294 JITValueDesc
					if d8.SliceSizeKnown {
						d294 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d8.KnownSliceLen))}
					} else if d8.Loc == LocImm {
						d294 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d8.StackOff))}
					} else if d8.Loc == LocStackTriple {
						d294 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d8.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d8)
						if d8.Loc == LocRegPair || d8.Loc == LocRegTriple {
							d294 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg2, ID: 0}
						} else if d8.Loc == LocReg {
							d294 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d226)
					ctx.EnsureDesc(&d294)
					ctx.EnsureDescsTogether(&d226, &d294)
					var d295 JITValueDesc
					if d226.Loc == LocImm && d294.Loc == LocImm {
						d295 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d226.Imm.Int() < d294.Imm.Int())}
					} else if d294.Loc == LocImm {
						r28 := ctx.AllocRegExcept(d226.Reg)
						if d294.Imm.Int() >= -2147483648 && d294.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d226.Reg, int32(d294.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d294.Imm.Int()))
							ctx.EmitCmpInt64(d226.Reg, RegR11)
						}
						ctx.EmitSetcc(r28, CondSignedLess)
						d295 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r28}
						ctx.BindReg(r28, &d295)
					} else if d226.Loc == LocImm {
						r29 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d226.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d294.Reg)
						ctx.EmitSetcc(r29, CondSignedLess)
						d295 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r29}
						ctx.BindReg(r29, &d295)
					} else {
						r30 := ctx.AllocRegExcept(d226.Reg)
						ctx.EmitCmpInt64(d226.Reg, d294.Reg)
						ctx.EmitSetcc(r30, CondSignedLess)
						d295 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r30}
						ctx.BindReg(r30, &d295)
					}
					ctx.FreeDesc(&d294)
					d296 = d295
					ctx.EnsureDesc(&d296)
					if d296.Loc != LocImm && d296.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d296.Loc == LocImm {
						if d296.Imm.Bool() {
							if ps.General {
							}
							ps297 := PhiState{General: ps.General}
							ps297.OverlayValues = make([]JITValueDesc, 297)
							ps297.OverlayValues[3] = d3
							ps297.OverlayValues[4] = d4
							ps297.OverlayValues[5] = d5
							ps297.OverlayValues[6] = d6
							ps297.OverlayValues[7] = d7
							ps297.OverlayValues[8] = d8
							ps297.OverlayValues[9] = d9
							ps297.OverlayValues[10] = d10
							ps297.OverlayValues[11] = d11
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
							ps297.OverlayValues[47] = d47
							ps297.OverlayValues[48] = d48
							ps297.OverlayValues[86] = d86
							ps297.OverlayValues[87] = d87
							ps297.OverlayValues[88] = d88
							ps297.OverlayValues[89] = d89
							ps297.OverlayValues[90] = d90
							ps297.OverlayValues[92] = d92
							ps297.OverlayValues[93] = d93
							ps297.OverlayValues[94] = d94
							ps297.OverlayValues[95] = d95
							ps297.OverlayValues[96] = d96
							ps297.OverlayValues[97] = d97
							ps297.OverlayValues[98] = d98
							ps297.OverlayValues[99] = d99
							ps297.OverlayValues[100] = d100
							ps297.OverlayValues[102] = d102
							ps297.OverlayValues[103] = d103
							ps297.OverlayValues[104] = d104
							ps297.OverlayValues[105] = d105
							ps297.OverlayValues[106] = d106
							ps297.OverlayValues[163] = d163
							ps297.OverlayValues[164] = d164
							ps297.OverlayValues[165] = d165
							ps297.OverlayValues[225] = d225
							ps297.OverlayValues[226] = d226
							ps297.OverlayValues[227] = d227
							ps297.OverlayValues[228] = d228
							ps297.OverlayValues[231] = d231
							ps297.OverlayValues[294] = d294
							ps297.OverlayValues[295] = d295
							ps297.OverlayValues[296] = d296
							return bbs[10].RenderPS(ps297)
						}
						if ps.General {
						}
						ps298 := PhiState{General: ps.General}
						ps298.OverlayValues = make([]JITValueDesc, 297)
						ps298.OverlayValues[3] = d3
						ps298.OverlayValues[4] = d4
						ps298.OverlayValues[5] = d5
						ps298.OverlayValues[6] = d6
						ps298.OverlayValues[7] = d7
						ps298.OverlayValues[8] = d8
						ps298.OverlayValues[9] = d9
						ps298.OverlayValues[10] = d10
						ps298.OverlayValues[11] = d11
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
						ps298.OverlayValues[47] = d47
						ps298.OverlayValues[48] = d48
						ps298.OverlayValues[86] = d86
						ps298.OverlayValues[87] = d87
						ps298.OverlayValues[88] = d88
						ps298.OverlayValues[89] = d89
						ps298.OverlayValues[90] = d90
						ps298.OverlayValues[92] = d92
						ps298.OverlayValues[93] = d93
						ps298.OverlayValues[94] = d94
						ps298.OverlayValues[95] = d95
						ps298.OverlayValues[96] = d96
						ps298.OverlayValues[97] = d97
						ps298.OverlayValues[98] = d98
						ps298.OverlayValues[99] = d99
						ps298.OverlayValues[100] = d100
						ps298.OverlayValues[102] = d102
						ps298.OverlayValues[103] = d103
						ps298.OverlayValues[104] = d104
						ps298.OverlayValues[105] = d105
						ps298.OverlayValues[106] = d106
						ps298.OverlayValues[163] = d163
						ps298.OverlayValues[164] = d164
						ps298.OverlayValues[165] = d165
						ps298.OverlayValues[225] = d225
						ps298.OverlayValues[226] = d226
						ps298.OverlayValues[227] = d227
						ps298.OverlayValues[228] = d228
						ps298.OverlayValues[231] = d231
						ps298.OverlayValues[294] = d294
						ps298.OverlayValues[295] = d295
						ps298.OverlayValues[296] = d296
						return bbs[11].RenderPS(ps298)
					}
					if !ps.General {
						ps.General = true
						return bbs[8].RenderPS(ps)
					}
					lbl25 := ctx.ReserveLabel()
					lbl26 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d296.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl25)
					ctx.EmitJmp(lbl26)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl12)
					ps299 := PhiState{General: true}
					ps299.OverlayValues = make([]JITValueDesc, 297)
					ps299.OverlayValues[3] = d3
					ps299.OverlayValues[4] = d4
					ps299.OverlayValues[5] = d5
					ps299.OverlayValues[6] = d6
					ps299.OverlayValues[7] = d7
					ps299.OverlayValues[8] = d8
					ps299.OverlayValues[9] = d9
					ps299.OverlayValues[10] = d10
					ps299.OverlayValues[11] = d11
					ps299.OverlayValues[26] = d26
					ps299.OverlayValues[27] = d27
					ps299.OverlayValues[28] = d28
					ps299.OverlayValues[29] = d29
					ps299.OverlayValues[30] = d30
					ps299.OverlayValues[31] = d31
					ps299.OverlayValues[32] = d32
					ps299.OverlayValues[33] = d33
					ps299.OverlayValues[34] = d34
					ps299.OverlayValues[35] = d35
					ps299.OverlayValues[36] = d36
					ps299.OverlayValues[37] = d37
					ps299.OverlayValues[38] = d38
					ps299.OverlayValues[39] = d39
					ps299.OverlayValues[40] = d40
					ps299.OverlayValues[41] = d41
					ps299.OverlayValues[42] = d42
					ps299.OverlayValues[43] = d43
					ps299.OverlayValues[44] = d44
					ps299.OverlayValues[45] = d45
					ps299.OverlayValues[46] = d46
					ps299.OverlayValues[47] = d47
					ps299.OverlayValues[48] = d48
					ps299.OverlayValues[86] = d86
					ps299.OverlayValues[87] = d87
					ps299.OverlayValues[88] = d88
					ps299.OverlayValues[89] = d89
					ps299.OverlayValues[90] = d90
					ps299.OverlayValues[92] = d92
					ps299.OverlayValues[93] = d93
					ps299.OverlayValues[94] = d94
					ps299.OverlayValues[95] = d95
					ps299.OverlayValues[96] = d96
					ps299.OverlayValues[97] = d97
					ps299.OverlayValues[98] = d98
					ps299.OverlayValues[99] = d99
					ps299.OverlayValues[100] = d100
					ps299.OverlayValues[102] = d102
					ps299.OverlayValues[103] = d103
					ps299.OverlayValues[104] = d104
					ps299.OverlayValues[105] = d105
					ps299.OverlayValues[106] = d106
					ps299.OverlayValues[163] = d163
					ps299.OverlayValues[164] = d164
					ps299.OverlayValues[165] = d165
					ps299.OverlayValues[225] = d225
					ps299.OverlayValues[226] = d226
					ps299.OverlayValues[227] = d227
					ps299.OverlayValues[228] = d228
					ps299.OverlayValues[231] = d231
					ps299.OverlayValues[294] = d294
					ps299.OverlayValues[295] = d295
					ps299.OverlayValues[296] = d296
					ps300 := PhiState{General: true}
					ps300.OverlayValues = make([]JITValueDesc, 297)
					ps300.OverlayValues[3] = d3
					ps300.OverlayValues[4] = d4
					ps300.OverlayValues[5] = d5
					ps300.OverlayValues[6] = d6
					ps300.OverlayValues[7] = d7
					ps300.OverlayValues[8] = d8
					ps300.OverlayValues[9] = d9
					ps300.OverlayValues[10] = d10
					ps300.OverlayValues[11] = d11
					ps300.OverlayValues[26] = d26
					ps300.OverlayValues[27] = d27
					ps300.OverlayValues[28] = d28
					ps300.OverlayValues[29] = d29
					ps300.OverlayValues[30] = d30
					ps300.OverlayValues[31] = d31
					ps300.OverlayValues[32] = d32
					ps300.OverlayValues[33] = d33
					ps300.OverlayValues[34] = d34
					ps300.OverlayValues[35] = d35
					ps300.OverlayValues[36] = d36
					ps300.OverlayValues[37] = d37
					ps300.OverlayValues[38] = d38
					ps300.OverlayValues[39] = d39
					ps300.OverlayValues[40] = d40
					ps300.OverlayValues[41] = d41
					ps300.OverlayValues[42] = d42
					ps300.OverlayValues[43] = d43
					ps300.OverlayValues[44] = d44
					ps300.OverlayValues[45] = d45
					ps300.OverlayValues[46] = d46
					ps300.OverlayValues[47] = d47
					ps300.OverlayValues[48] = d48
					ps300.OverlayValues[86] = d86
					ps300.OverlayValues[87] = d87
					ps300.OverlayValues[88] = d88
					ps300.OverlayValues[89] = d89
					ps300.OverlayValues[90] = d90
					ps300.OverlayValues[92] = d92
					ps300.OverlayValues[93] = d93
					ps300.OverlayValues[94] = d94
					ps300.OverlayValues[95] = d95
					ps300.OverlayValues[96] = d96
					ps300.OverlayValues[97] = d97
					ps300.OverlayValues[98] = d98
					ps300.OverlayValues[99] = d99
					ps300.OverlayValues[100] = d100
					ps300.OverlayValues[102] = d102
					ps300.OverlayValues[103] = d103
					ps300.OverlayValues[104] = d104
					ps300.OverlayValues[105] = d105
					ps300.OverlayValues[106] = d106
					ps300.OverlayValues[163] = d163
					ps300.OverlayValues[164] = d164
					ps300.OverlayValues[165] = d165
					ps300.OverlayValues[225] = d225
					ps300.OverlayValues[226] = d226
					ps300.OverlayValues[227] = d227
					ps300.OverlayValues[228] = d228
					ps300.OverlayValues[231] = d231
					ps300.OverlayValues[294] = d294
					ps300.OverlayValues[295] = d295
					ps300.OverlayValues[296] = d296
					snap301 := d3
					snap302 := d4
					snap303 := d5
					snap304 := d6
					snap305 := d7
					snap306 := d8
					snap307 := d9
					snap308 := d10
					snap309 := d11
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
					snap331 := d47
					snap332 := d48
					snap333 := d86
					snap334 := d87
					snap335 := d88
					snap336 := d89
					snap337 := d90
					snap338 := d92
					snap339 := d93
					snap340 := d94
					snap341 := d95
					snap342 := d96
					snap343 := d97
					snap344 := d98
					snap345 := d99
					snap346 := d100
					snap347 := d102
					snap348 := d103
					snap349 := d104
					snap350 := d105
					snap351 := d106
					snap352 := d163
					snap353 := d164
					snap354 := d165
					snap355 := d225
					snap356 := d226
					snap357 := d227
					snap358 := d228
					snap359 := d231
					snap360 := d294
					snap361 := d295
					snap362 := d296
					alloc363 := ctx.SnapshotAllocState()
					if !bbs[11].Rendered {
						bbs[11].RenderPS(ps300)
					}
					ctx.RestoreAllocState(alloc363)
					d3 = snap301
					d4 = snap302
					d5 = snap303
					d6 = snap304
					d7 = snap305
					d8 = snap306
					d9 = snap307
					d10 = snap308
					d11 = snap309
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
					d47 = snap331
					d48 = snap332
					d86 = snap333
					d87 = snap334
					d88 = snap335
					d89 = snap336
					d90 = snap337
					d92 = snap338
					d93 = snap339
					d94 = snap340
					d95 = snap341
					d96 = snap342
					d97 = snap343
					d98 = snap344
					d99 = snap345
					d100 = snap346
					d102 = snap347
					d103 = snap348
					d104 = snap349
					d105 = snap350
					d106 = snap351
					d163 = snap352
					d164 = snap353
					d165 = snap354
					d225 = snap355
					d226 = snap356
					d227 = snap357
					d228 = snap358
					d231 = snap359
					d294 = snap360
					d295 = snap361
					d296 = snap362
					if !bbs[10].Rendered {
						return bbs[10].RenderPS(ps299)
					}
					return result
					ctx.FreeDesc(&d295)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
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
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
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
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
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
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != LocNone {
						d225 = ps.OverlayValues[225]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != LocNone {
						d227 = ps.OverlayValues[227]
					}
					if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != LocNone {
						d228 = ps.OverlayValues[228]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d34)
					var d364 JITValueDesc
					if d34.Loc == LocImm {
						d364 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d34.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d34.Reg)
						ctx.EmitMovRegReg(scratch, d34.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d364 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d364)
					}
					if d364.Loc == LocReg && d34.Loc == LocReg && d364.Reg == d34.Reg {
						ctx.TransferReg(d34.Reg)
						d34.Loc = LocNone
					}
					ctx.FreeDesc(&d34)
					ctx.EnsureDesc(&d364)
					ctx.EnsureDesc(&d364)
					ctx.EnsureDesc(&d364)
					d366 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.SyncDesc(&d364)
					d367 = d5
					d367.ID = 0
					d368 = d366
					d368.ID = 0
					d369 = ctx.EmitSliceElementAddress(&d367, &d368, int32(16))
					ctx.FreeDesc(&d368)
					ctx.EmitStoreScmerAt(&d369, &d364)
					ctx.FreeDesc(&d369)
					ctx.EnsureDesc(&d29)
					var d370 JITValueDesc
					if d29.Loc == LocImm {
						d370 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d29.Imm.Int() > 0)}
					} else {
						r31 := ctx.AllocRegExcept(d29.Reg)
						ctx.EmitCmpRegImm32(d29.Reg, 0)
						ctx.EmitSetcc(r31, CondSignedGreater)
						d370 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r31}
						ctx.BindReg(r31, &d370)
					}
					d371 = d370
					ctx.EnsureDesc(&d371)
					if d371.Loc != LocImm && d371.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d371.Loc == LocImm {
						if d371.Imm.Bool() {
							if ps.General {
							}
							ps372 := PhiState{General: ps.General}
							ps372.OverlayValues = make([]JITValueDesc, 372)
							ps372.OverlayValues[3] = d3
							ps372.OverlayValues[4] = d4
							ps372.OverlayValues[5] = d5
							ps372.OverlayValues[6] = d6
							ps372.OverlayValues[7] = d7
							ps372.OverlayValues[8] = d8
							ps372.OverlayValues[9] = d9
							ps372.OverlayValues[10] = d10
							ps372.OverlayValues[11] = d11
							ps372.OverlayValues[26] = d26
							ps372.OverlayValues[27] = d27
							ps372.OverlayValues[28] = d28
							ps372.OverlayValues[29] = d29
							ps372.OverlayValues[30] = d30
							ps372.OverlayValues[31] = d31
							ps372.OverlayValues[32] = d32
							ps372.OverlayValues[33] = d33
							ps372.OverlayValues[34] = d34
							ps372.OverlayValues[35] = d35
							ps372.OverlayValues[36] = d36
							ps372.OverlayValues[37] = d37
							ps372.OverlayValues[38] = d38
							ps372.OverlayValues[39] = d39
							ps372.OverlayValues[40] = d40
							ps372.OverlayValues[41] = d41
							ps372.OverlayValues[42] = d42
							ps372.OverlayValues[43] = d43
							ps372.OverlayValues[44] = d44
							ps372.OverlayValues[45] = d45
							ps372.OverlayValues[46] = d46
							ps372.OverlayValues[47] = d47
							ps372.OverlayValues[48] = d48
							ps372.OverlayValues[86] = d86
							ps372.OverlayValues[87] = d87
							ps372.OverlayValues[88] = d88
							ps372.OverlayValues[89] = d89
							ps372.OverlayValues[90] = d90
							ps372.OverlayValues[92] = d92
							ps372.OverlayValues[93] = d93
							ps372.OverlayValues[94] = d94
							ps372.OverlayValues[95] = d95
							ps372.OverlayValues[96] = d96
							ps372.OverlayValues[97] = d97
							ps372.OverlayValues[98] = d98
							ps372.OverlayValues[99] = d99
							ps372.OverlayValues[100] = d100
							ps372.OverlayValues[102] = d102
							ps372.OverlayValues[103] = d103
							ps372.OverlayValues[104] = d104
							ps372.OverlayValues[105] = d105
							ps372.OverlayValues[106] = d106
							ps372.OverlayValues[163] = d163
							ps372.OverlayValues[164] = d164
							ps372.OverlayValues[165] = d165
							ps372.OverlayValues[225] = d225
							ps372.OverlayValues[226] = d226
							ps372.OverlayValues[227] = d227
							ps372.OverlayValues[228] = d228
							ps372.OverlayValues[231] = d231
							ps372.OverlayValues[294] = d294
							ps372.OverlayValues[295] = d295
							ps372.OverlayValues[296] = d296
							ps372.OverlayValues[364] = d364
							ps372.OverlayValues[365] = d365
							ps372.OverlayValues[366] = d366
							ps372.OverlayValues[367] = d367
							ps372.OverlayValues[368] = d368
							ps372.OverlayValues[369] = d369
							ps372.OverlayValues[370] = d370
							ps372.OverlayValues[371] = d371
							return bbs[12].RenderPS(ps372)
						}
						if ps.General {
						}
						ps373 := PhiState{General: ps.General}
						ps373.OverlayValues = make([]JITValueDesc, 372)
						ps373.OverlayValues[3] = d3
						ps373.OverlayValues[4] = d4
						ps373.OverlayValues[5] = d5
						ps373.OverlayValues[6] = d6
						ps373.OverlayValues[7] = d7
						ps373.OverlayValues[8] = d8
						ps373.OverlayValues[9] = d9
						ps373.OverlayValues[10] = d10
						ps373.OverlayValues[11] = d11
						ps373.OverlayValues[26] = d26
						ps373.OverlayValues[27] = d27
						ps373.OverlayValues[28] = d28
						ps373.OverlayValues[29] = d29
						ps373.OverlayValues[30] = d30
						ps373.OverlayValues[31] = d31
						ps373.OverlayValues[32] = d32
						ps373.OverlayValues[33] = d33
						ps373.OverlayValues[34] = d34
						ps373.OverlayValues[35] = d35
						ps373.OverlayValues[36] = d36
						ps373.OverlayValues[37] = d37
						ps373.OverlayValues[38] = d38
						ps373.OverlayValues[39] = d39
						ps373.OverlayValues[40] = d40
						ps373.OverlayValues[41] = d41
						ps373.OverlayValues[42] = d42
						ps373.OverlayValues[43] = d43
						ps373.OverlayValues[44] = d44
						ps373.OverlayValues[45] = d45
						ps373.OverlayValues[46] = d46
						ps373.OverlayValues[47] = d47
						ps373.OverlayValues[48] = d48
						ps373.OverlayValues[86] = d86
						ps373.OverlayValues[87] = d87
						ps373.OverlayValues[88] = d88
						ps373.OverlayValues[89] = d89
						ps373.OverlayValues[90] = d90
						ps373.OverlayValues[92] = d92
						ps373.OverlayValues[93] = d93
						ps373.OverlayValues[94] = d94
						ps373.OverlayValues[95] = d95
						ps373.OverlayValues[96] = d96
						ps373.OverlayValues[97] = d97
						ps373.OverlayValues[98] = d98
						ps373.OverlayValues[99] = d99
						ps373.OverlayValues[100] = d100
						ps373.OverlayValues[102] = d102
						ps373.OverlayValues[103] = d103
						ps373.OverlayValues[104] = d104
						ps373.OverlayValues[105] = d105
						ps373.OverlayValues[106] = d106
						ps373.OverlayValues[163] = d163
						ps373.OverlayValues[164] = d164
						ps373.OverlayValues[165] = d165
						ps373.OverlayValues[225] = d225
						ps373.OverlayValues[226] = d226
						ps373.OverlayValues[227] = d227
						ps373.OverlayValues[228] = d228
						ps373.OverlayValues[231] = d231
						ps373.OverlayValues[294] = d294
						ps373.OverlayValues[295] = d295
						ps373.OverlayValues[296] = d296
						ps373.OverlayValues[364] = d364
						ps373.OverlayValues[365] = d365
						ps373.OverlayValues[366] = d366
						ps373.OverlayValues[367] = d367
						ps373.OverlayValues[368] = d368
						ps373.OverlayValues[369] = d369
						ps373.OverlayValues[370] = d370
						ps373.OverlayValues[371] = d371
						return bbs[13].RenderPS(ps373)
					}
					if !ps.General {
						ps.General = true
						return bbs[9].RenderPS(ps)
					}
					lbl27 := ctx.ReserveLabel()
					lbl28 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d371.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl27)
					ctx.EmitJmp(lbl28)
					ctx.MarkLabel(lbl27)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl28)
					ctx.EmitJmp(lbl14)
					ps374 := PhiState{General: true}
					ps374.OverlayValues = make([]JITValueDesc, 372)
					ps374.OverlayValues[3] = d3
					ps374.OverlayValues[4] = d4
					ps374.OverlayValues[5] = d5
					ps374.OverlayValues[6] = d6
					ps374.OverlayValues[7] = d7
					ps374.OverlayValues[8] = d8
					ps374.OverlayValues[9] = d9
					ps374.OverlayValues[10] = d10
					ps374.OverlayValues[11] = d11
					ps374.OverlayValues[26] = d26
					ps374.OverlayValues[27] = d27
					ps374.OverlayValues[28] = d28
					ps374.OverlayValues[29] = d29
					ps374.OverlayValues[30] = d30
					ps374.OverlayValues[31] = d31
					ps374.OverlayValues[32] = d32
					ps374.OverlayValues[33] = d33
					ps374.OverlayValues[34] = d34
					ps374.OverlayValues[35] = d35
					ps374.OverlayValues[36] = d36
					ps374.OverlayValues[37] = d37
					ps374.OverlayValues[38] = d38
					ps374.OverlayValues[39] = d39
					ps374.OverlayValues[40] = d40
					ps374.OverlayValues[41] = d41
					ps374.OverlayValues[42] = d42
					ps374.OverlayValues[43] = d43
					ps374.OverlayValues[44] = d44
					ps374.OverlayValues[45] = d45
					ps374.OverlayValues[46] = d46
					ps374.OverlayValues[47] = d47
					ps374.OverlayValues[48] = d48
					ps374.OverlayValues[86] = d86
					ps374.OverlayValues[87] = d87
					ps374.OverlayValues[88] = d88
					ps374.OverlayValues[89] = d89
					ps374.OverlayValues[90] = d90
					ps374.OverlayValues[92] = d92
					ps374.OverlayValues[93] = d93
					ps374.OverlayValues[94] = d94
					ps374.OverlayValues[95] = d95
					ps374.OverlayValues[96] = d96
					ps374.OverlayValues[97] = d97
					ps374.OverlayValues[98] = d98
					ps374.OverlayValues[99] = d99
					ps374.OverlayValues[100] = d100
					ps374.OverlayValues[102] = d102
					ps374.OverlayValues[103] = d103
					ps374.OverlayValues[104] = d104
					ps374.OverlayValues[105] = d105
					ps374.OverlayValues[106] = d106
					ps374.OverlayValues[163] = d163
					ps374.OverlayValues[164] = d164
					ps374.OverlayValues[165] = d165
					ps374.OverlayValues[225] = d225
					ps374.OverlayValues[226] = d226
					ps374.OverlayValues[227] = d227
					ps374.OverlayValues[228] = d228
					ps374.OverlayValues[231] = d231
					ps374.OverlayValues[294] = d294
					ps374.OverlayValues[295] = d295
					ps374.OverlayValues[296] = d296
					ps374.OverlayValues[364] = d364
					ps374.OverlayValues[365] = d365
					ps374.OverlayValues[366] = d366
					ps374.OverlayValues[367] = d367
					ps374.OverlayValues[368] = d368
					ps374.OverlayValues[369] = d369
					ps374.OverlayValues[370] = d370
					ps374.OverlayValues[371] = d371
					ps375 := PhiState{General: true}
					ps375.OverlayValues = make([]JITValueDesc, 372)
					ps375.OverlayValues[3] = d3
					ps375.OverlayValues[4] = d4
					ps375.OverlayValues[5] = d5
					ps375.OverlayValues[6] = d6
					ps375.OverlayValues[7] = d7
					ps375.OverlayValues[8] = d8
					ps375.OverlayValues[9] = d9
					ps375.OverlayValues[10] = d10
					ps375.OverlayValues[11] = d11
					ps375.OverlayValues[26] = d26
					ps375.OverlayValues[27] = d27
					ps375.OverlayValues[28] = d28
					ps375.OverlayValues[29] = d29
					ps375.OverlayValues[30] = d30
					ps375.OverlayValues[31] = d31
					ps375.OverlayValues[32] = d32
					ps375.OverlayValues[33] = d33
					ps375.OverlayValues[34] = d34
					ps375.OverlayValues[35] = d35
					ps375.OverlayValues[36] = d36
					ps375.OverlayValues[37] = d37
					ps375.OverlayValues[38] = d38
					ps375.OverlayValues[39] = d39
					ps375.OverlayValues[40] = d40
					ps375.OverlayValues[41] = d41
					ps375.OverlayValues[42] = d42
					ps375.OverlayValues[43] = d43
					ps375.OverlayValues[44] = d44
					ps375.OverlayValues[45] = d45
					ps375.OverlayValues[46] = d46
					ps375.OverlayValues[47] = d47
					ps375.OverlayValues[48] = d48
					ps375.OverlayValues[86] = d86
					ps375.OverlayValues[87] = d87
					ps375.OverlayValues[88] = d88
					ps375.OverlayValues[89] = d89
					ps375.OverlayValues[90] = d90
					ps375.OverlayValues[92] = d92
					ps375.OverlayValues[93] = d93
					ps375.OverlayValues[94] = d94
					ps375.OverlayValues[95] = d95
					ps375.OverlayValues[96] = d96
					ps375.OverlayValues[97] = d97
					ps375.OverlayValues[98] = d98
					ps375.OverlayValues[99] = d99
					ps375.OverlayValues[100] = d100
					ps375.OverlayValues[102] = d102
					ps375.OverlayValues[103] = d103
					ps375.OverlayValues[104] = d104
					ps375.OverlayValues[105] = d105
					ps375.OverlayValues[106] = d106
					ps375.OverlayValues[163] = d163
					ps375.OverlayValues[164] = d164
					ps375.OverlayValues[165] = d165
					ps375.OverlayValues[225] = d225
					ps375.OverlayValues[226] = d226
					ps375.OverlayValues[227] = d227
					ps375.OverlayValues[228] = d228
					ps375.OverlayValues[231] = d231
					ps375.OverlayValues[294] = d294
					ps375.OverlayValues[295] = d295
					ps375.OverlayValues[296] = d296
					ps375.OverlayValues[364] = d364
					ps375.OverlayValues[365] = d365
					ps375.OverlayValues[366] = d366
					ps375.OverlayValues[367] = d367
					ps375.OverlayValues[368] = d368
					ps375.OverlayValues[369] = d369
					ps375.OverlayValues[370] = d370
					ps375.OverlayValues[371] = d371
					snap376 := d3
					snap377 := d4
					snap378 := d5
					snap379 := d6
					snap380 := d7
					snap381 := d8
					snap382 := d9
					snap383 := d10
					snap384 := d11
					snap385 := d26
					snap386 := d27
					snap387 := d28
					snap388 := d29
					snap389 := d30
					snap390 := d31
					snap391 := d32
					snap392 := d33
					snap393 := d34
					snap394 := d35
					snap395 := d36
					snap396 := d37
					snap397 := d38
					snap398 := d39
					snap399 := d40
					snap400 := d41
					snap401 := d42
					snap402 := d43
					snap403 := d44
					snap404 := d45
					snap405 := d46
					snap406 := d47
					snap407 := d48
					snap408 := d86
					snap409 := d87
					snap410 := d88
					snap411 := d89
					snap412 := d90
					snap413 := d92
					snap414 := d93
					snap415 := d94
					snap416 := d95
					snap417 := d96
					snap418 := d97
					snap419 := d98
					snap420 := d99
					snap421 := d100
					snap422 := d102
					snap423 := d103
					snap424 := d104
					snap425 := d105
					snap426 := d106
					snap427 := d163
					snap428 := d164
					snap429 := d165
					snap430 := d225
					snap431 := d226
					snap432 := d227
					snap433 := d228
					snap434 := d231
					snap435 := d294
					snap436 := d295
					snap437 := d296
					snap438 := d364
					snap439 := d365
					snap440 := d366
					snap441 := d367
					snap442 := d368
					snap443 := d369
					snap444 := d370
					snap445 := d371
					alloc446 := ctx.SnapshotAllocState()
					if !bbs[13].Rendered {
						bbs[13].RenderPS(ps375)
					}
					ctx.RestoreAllocState(alloc446)
					d3 = snap376
					d4 = snap377
					d5 = snap378
					d6 = snap379
					d7 = snap380
					d8 = snap381
					d9 = snap382
					d10 = snap383
					d11 = snap384
					d26 = snap385
					d27 = snap386
					d28 = snap387
					d29 = snap388
					d30 = snap389
					d31 = snap390
					d32 = snap391
					d33 = snap392
					d34 = snap393
					d35 = snap394
					d36 = snap395
					d37 = snap396
					d38 = snap397
					d39 = snap398
					d40 = snap399
					d41 = snap400
					d42 = snap401
					d43 = snap402
					d44 = snap403
					d45 = snap404
					d46 = snap405
					d47 = snap406
					d48 = snap407
					d86 = snap408
					d87 = snap409
					d88 = snap410
					d89 = snap411
					d90 = snap412
					d92 = snap413
					d93 = snap414
					d94 = snap415
					d95 = snap416
					d96 = snap417
					d97 = snap418
					d98 = snap419
					d99 = snap420
					d100 = snap421
					d102 = snap422
					d103 = snap423
					d104 = snap424
					d105 = snap425
					d106 = snap426
					d163 = snap427
					d164 = snap428
					d165 = snap429
					d225 = snap430
					d226 = snap431
					d227 = snap432
					d228 = snap433
					d231 = snap434
					d294 = snap435
					d295 = snap436
					d296 = snap437
					d364 = snap438
					d365 = snap439
					d366 = snap440
					d367 = snap441
					d368 = snap442
					d369 = snap443
					d370 = snap444
					d371 = snap445
					if !bbs[12].Rendered {
						return bbs[12].RenderPS(ps374)
					}
					return result
					ctx.FreeDesc(&d370)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
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
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
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
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
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
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != LocNone {
						d225 = ps.OverlayValues[225]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != LocNone {
						d227 = ps.OverlayValues[227]
					}
					if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != LocNone {
						d228 = ps.OverlayValues[228]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
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
					if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != LocNone {
						d368 = ps.OverlayValues[368]
					}
					if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != LocNone {
						d369 = ps.OverlayValues[369]
					}
					if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != LocNone {
						d370 = ps.OverlayValues[370]
					}
					if len(ps.OverlayValues) > 371 && ps.OverlayValues[371].Loc != LocNone {
						d371 = ps.OverlayValues[371]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d226)
					d448 = ctx.EmitSliceElementAddress(&d8, &d226, 16)
					ctx.EnsureDesc(&d448)
					r32 := ctx.AllocRegExcept(d448.Reg)
					ctx.EmitMovRegMem(r32, d448.Reg, 8)
					ctx.EmitMovRegMem(d448.Reg, d448.Reg, 0)
					d447 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d448.Reg, Reg2: r32}
					ctx.BindReg(d448.Reg, &d447)
					ctx.BindReg(r32, &d447)
					ctx.EnsureDesc(&d226)
					ctx.SyncDesc(&d447)
					ctx.StabilizeDescAcrossNestedCall(&d226)
					d449 = d99
					d449.ID = 0
					d450 = d226
					d450.ID = 0
					d451 = ctx.EmitSliceElementAddress(&d449, &d450, int32(16))
					ctx.FreeDesc(&d450)
					ctx.EmitStoreScmerAt(&d451, &d447)
					ctx.FreeDesc(&d451)
					ctx.FreeDesc(&d447)
					if ps.General {
						ctx.SyncDesc(&d226)
						if d226.Loc == LocReg {
							ctx.ProtectReg(d226.Reg)
						} else if d226.Loc == LocRegPair {
							ctx.ProtectReg(d226.Reg)
							ctx.ProtectReg(d226.Reg2)
						}
						d452 = d226
						if d452.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d452)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d452)
						} else {
							ctx.EmitStoreToStack(d452, int32(bbs[7].PhiBase)+int32(0))
						}
						if d226.Loc == LocReg {
							ctx.UnprotectReg(d226.Reg)
						} else if d226.Loc == LocRegPair {
							ctx.UnprotectReg(d226.Reg)
							ctx.UnprotectReg(d226.Reg2)
						}
					}
					ps453 := PhiState{General: ps.General}
					ps453.OverlayValues = make([]JITValueDesc, 453)
					ps453.OverlayValues[3] = d3
					ps453.OverlayValues[4] = d4
					ps453.OverlayValues[5] = d5
					ps453.OverlayValues[6] = d6
					ps453.OverlayValues[7] = d7
					ps453.OverlayValues[8] = d8
					ps453.OverlayValues[9] = d9
					ps453.OverlayValues[10] = d10
					ps453.OverlayValues[11] = d11
					ps453.OverlayValues[26] = d26
					ps453.OverlayValues[27] = d27
					ps453.OverlayValues[28] = d28
					ps453.OverlayValues[29] = d29
					ps453.OverlayValues[30] = d30
					ps453.OverlayValues[31] = d31
					ps453.OverlayValues[32] = d32
					ps453.OverlayValues[33] = d33
					ps453.OverlayValues[34] = d34
					ps453.OverlayValues[35] = d35
					ps453.OverlayValues[36] = d36
					ps453.OverlayValues[37] = d37
					ps453.OverlayValues[38] = d38
					ps453.OverlayValues[39] = d39
					ps453.OverlayValues[40] = d40
					ps453.OverlayValues[41] = d41
					ps453.OverlayValues[42] = d42
					ps453.OverlayValues[43] = d43
					ps453.OverlayValues[44] = d44
					ps453.OverlayValues[45] = d45
					ps453.OverlayValues[46] = d46
					ps453.OverlayValues[47] = d47
					ps453.OverlayValues[48] = d48
					ps453.OverlayValues[86] = d86
					ps453.OverlayValues[87] = d87
					ps453.OverlayValues[88] = d88
					ps453.OverlayValues[89] = d89
					ps453.OverlayValues[90] = d90
					ps453.OverlayValues[92] = d92
					ps453.OverlayValues[93] = d93
					ps453.OverlayValues[94] = d94
					ps453.OverlayValues[95] = d95
					ps453.OverlayValues[96] = d96
					ps453.OverlayValues[97] = d97
					ps453.OverlayValues[98] = d98
					ps453.OverlayValues[99] = d99
					ps453.OverlayValues[100] = d100
					ps453.OverlayValues[102] = d102
					ps453.OverlayValues[103] = d103
					ps453.OverlayValues[104] = d104
					ps453.OverlayValues[105] = d105
					ps453.OverlayValues[106] = d106
					ps453.OverlayValues[163] = d163
					ps453.OverlayValues[164] = d164
					ps453.OverlayValues[165] = d165
					ps453.OverlayValues[225] = d225
					ps453.OverlayValues[226] = d226
					ps453.OverlayValues[227] = d227
					ps453.OverlayValues[228] = d228
					ps453.OverlayValues[231] = d231
					ps453.OverlayValues[294] = d294
					ps453.OverlayValues[295] = d295
					ps453.OverlayValues[296] = d296
					ps453.OverlayValues[364] = d364
					ps453.OverlayValues[365] = d365
					ps453.OverlayValues[366] = d366
					ps453.OverlayValues[367] = d367
					ps453.OverlayValues[368] = d368
					ps453.OverlayValues[369] = d369
					ps453.OverlayValues[370] = d370
					ps453.OverlayValues[371] = d371
					ps453.OverlayValues[447] = d447
					ps453.OverlayValues[448] = d448
					ps453.OverlayValues[449] = d449
					ps453.OverlayValues[450] = d450
					ps453.OverlayValues[451] = d451
					ps453.OverlayValues[452] = d452
					ps453.PhiValues = make([]JITValueDesc, 1)
					d454 = d226
					ps453.PhiValues[0] = d454
					if ps453.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps453)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
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
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
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
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
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
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != LocNone {
						d225 = ps.OverlayValues[225]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != LocNone {
						d227 = ps.OverlayValues[227]
					}
					if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != LocNone {
						d228 = ps.OverlayValues[228]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
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
					if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != LocNone {
						d368 = ps.OverlayValues[368]
					}
					if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != LocNone {
						d369 = ps.OverlayValues[369]
					}
					if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != LocNone {
						d370 = ps.OverlayValues[370]
					}
					if len(ps.OverlayValues) > 371 && ps.OverlayValues[371].Loc != LocNone {
						d371 = ps.OverlayValues[371]
					}
					if len(ps.OverlayValues) > 447 && ps.OverlayValues[447].Loc != LocNone {
						d447 = ps.OverlayValues[447]
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
					if len(ps.OverlayValues) > 454 && ps.OverlayValues[454].Loc != LocNone {
						d454 = ps.OverlayValues[454]
					}
					ctx.ReclaimUntrackedRegs()
					d455 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d226)
					ctx.SyncDesc(&d455)
					ctx.StabilizeDescAcrossNestedCall(&d226)
					d456 = d99
					d456.ID = 0
					d457 = d226
					d457.ID = 0
					d458 = ctx.EmitSliceElementAddress(&d456, &d457, int32(16))
					ctx.FreeDesc(&d457)
					ctx.EmitStoreScmerAt(&d458, &d455)
					ctx.FreeDesc(&d458)
					ctx.FreeDesc(&d455)
					if ps.General {
						ctx.SyncDesc(&d226)
						if d226.Loc == LocReg {
							ctx.ProtectReg(d226.Reg)
						} else if d226.Loc == LocRegPair {
							ctx.ProtectReg(d226.Reg)
							ctx.ProtectReg(d226.Reg2)
						}
						d459 = d226
						if d459.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d459)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d459)
						} else {
							ctx.EmitStoreToStack(d459, int32(bbs[7].PhiBase)+int32(0))
						}
						if d226.Loc == LocReg {
							ctx.UnprotectReg(d226.Reg)
						} else if d226.Loc == LocRegPair {
							ctx.UnprotectReg(d226.Reg)
							ctx.UnprotectReg(d226.Reg2)
						}
					}
					ps460 := PhiState{General: ps.General}
					ps460.OverlayValues = make([]JITValueDesc, 460)
					ps460.OverlayValues[3] = d3
					ps460.OverlayValues[4] = d4
					ps460.OverlayValues[5] = d5
					ps460.OverlayValues[6] = d6
					ps460.OverlayValues[7] = d7
					ps460.OverlayValues[8] = d8
					ps460.OverlayValues[9] = d9
					ps460.OverlayValues[10] = d10
					ps460.OverlayValues[11] = d11
					ps460.OverlayValues[26] = d26
					ps460.OverlayValues[27] = d27
					ps460.OverlayValues[28] = d28
					ps460.OverlayValues[29] = d29
					ps460.OverlayValues[30] = d30
					ps460.OverlayValues[31] = d31
					ps460.OverlayValues[32] = d32
					ps460.OverlayValues[33] = d33
					ps460.OverlayValues[34] = d34
					ps460.OverlayValues[35] = d35
					ps460.OverlayValues[36] = d36
					ps460.OverlayValues[37] = d37
					ps460.OverlayValues[38] = d38
					ps460.OverlayValues[39] = d39
					ps460.OverlayValues[40] = d40
					ps460.OverlayValues[41] = d41
					ps460.OverlayValues[42] = d42
					ps460.OverlayValues[43] = d43
					ps460.OverlayValues[44] = d44
					ps460.OverlayValues[45] = d45
					ps460.OverlayValues[46] = d46
					ps460.OverlayValues[47] = d47
					ps460.OverlayValues[48] = d48
					ps460.OverlayValues[86] = d86
					ps460.OverlayValues[87] = d87
					ps460.OverlayValues[88] = d88
					ps460.OverlayValues[89] = d89
					ps460.OverlayValues[90] = d90
					ps460.OverlayValues[92] = d92
					ps460.OverlayValues[93] = d93
					ps460.OverlayValues[94] = d94
					ps460.OverlayValues[95] = d95
					ps460.OverlayValues[96] = d96
					ps460.OverlayValues[97] = d97
					ps460.OverlayValues[98] = d98
					ps460.OverlayValues[99] = d99
					ps460.OverlayValues[100] = d100
					ps460.OverlayValues[102] = d102
					ps460.OverlayValues[103] = d103
					ps460.OverlayValues[104] = d104
					ps460.OverlayValues[105] = d105
					ps460.OverlayValues[106] = d106
					ps460.OverlayValues[163] = d163
					ps460.OverlayValues[164] = d164
					ps460.OverlayValues[165] = d165
					ps460.OverlayValues[225] = d225
					ps460.OverlayValues[226] = d226
					ps460.OverlayValues[227] = d227
					ps460.OverlayValues[228] = d228
					ps460.OverlayValues[231] = d231
					ps460.OverlayValues[294] = d294
					ps460.OverlayValues[295] = d295
					ps460.OverlayValues[296] = d296
					ps460.OverlayValues[364] = d364
					ps460.OverlayValues[365] = d365
					ps460.OverlayValues[366] = d366
					ps460.OverlayValues[367] = d367
					ps460.OverlayValues[368] = d368
					ps460.OverlayValues[369] = d369
					ps460.OverlayValues[370] = d370
					ps460.OverlayValues[371] = d371
					ps460.OverlayValues[447] = d447
					ps460.OverlayValues[448] = d448
					ps460.OverlayValues[449] = d449
					ps460.OverlayValues[450] = d450
					ps460.OverlayValues[451] = d451
					ps460.OverlayValues[452] = d452
					ps460.OverlayValues[454] = d454
					ps460.OverlayValues[455] = d455
					ps460.OverlayValues[456] = d456
					ps460.OverlayValues[457] = d457
					ps460.OverlayValues[458] = d458
					ps460.OverlayValues[459] = d459
					ps460.PhiValues = make([]JITValueDesc, 1)
					d461 = d226
					ps460.PhiValues[0] = d461
					if ps460.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps460)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
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
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
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
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
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
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != LocNone {
						d225 = ps.OverlayValues[225]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != LocNone {
						d227 = ps.OverlayValues[227]
					}
					if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != LocNone {
						d228 = ps.OverlayValues[228]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
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
					if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != LocNone {
						d368 = ps.OverlayValues[368]
					}
					if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != LocNone {
						d369 = ps.OverlayValues[369]
					}
					if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != LocNone {
						d370 = ps.OverlayValues[370]
					}
					if len(ps.OverlayValues) > 371 && ps.OverlayValues[371].Loc != LocNone {
						d371 = ps.OverlayValues[371]
					}
					if len(ps.OverlayValues) > 447 && ps.OverlayValues[447].Loc != LocNone {
						d447 = ps.OverlayValues[447]
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
					if len(ps.OverlayValues) > 454 && ps.OverlayValues[454].Loc != LocNone {
						d454 = ps.OverlayValues[454]
					}
					if len(ps.OverlayValues) > 455 && ps.OverlayValues[455].Loc != LocNone {
						d455 = ps.OverlayValues[455]
					}
					if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != LocNone {
						d456 = ps.OverlayValues[456]
					}
					if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != LocNone {
						d457 = ps.OverlayValues[457]
					}
					if len(ps.OverlayValues) > 458 && ps.OverlayValues[458].Loc != LocNone {
						d458 = ps.OverlayValues[458]
					}
					if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != LocNone {
						d459 = ps.OverlayValues[459]
					}
					if len(ps.OverlayValues) > 461 && ps.OverlayValues[461].Loc != LocNone {
						d461 = ps.OverlayValues[461]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d29)
					ctx.EnsureDesc(&d29)
					var d462 JITValueDesc
					if d29.Loc == LocImm {
						d462 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d29.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d29.Reg)
						ctx.EmitMovRegReg(scratch, d29.Reg)
						ctx.EmitSubRegImm32(scratch, int32(1))
						d462 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d462)
					}
					if d462.Loc == LocReg && d29.Loc == LocReg && d462.Reg == d29.Reg {
						ctx.TransferReg(d29.Reg)
						d29.Loc = LocNone
					}
					ctx.FreeDesc(&d29)
					ctx.EnsureDesc(&d462)
					ctx.EnsureDesc(&d462)
					ctx.EnsureDesc(&d462)
					d464 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.SyncDesc(&d462)
					d465 = d5
					d465.ID = 0
					d466 = d464
					d466.ID = 0
					d467 = ctx.EmitSliceElementAddress(&d465, &d466, int32(16))
					ctx.FreeDesc(&d466)
					ctx.EmitStoreScmerAt(&d467, &d462)
					ctx.FreeDesc(&d467)
					d468 = args[0]
					d468.ID = 0
					ctx.SyncDesc(&d468)
					if d468.Loc == LocRegPair || d468.Loc == LocStackPair || d468.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d468, &result)
						result.Type = d468.Type
					} else {
						switch d468.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d468)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d468)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d468)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d468, &result)
							result.Type = d468.Type
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d3)
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
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
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
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
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
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
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != LocNone {
						d225 = ps.OverlayValues[225]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != LocNone {
						d227 = ps.OverlayValues[227]
					}
					if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != LocNone {
						d228 = ps.OverlayValues[228]
					}
					if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != LocNone {
						d231 = ps.OverlayValues[231]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
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
					if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != LocNone {
						d368 = ps.OverlayValues[368]
					}
					if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != LocNone {
						d369 = ps.OverlayValues[369]
					}
					if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != LocNone {
						d370 = ps.OverlayValues[370]
					}
					if len(ps.OverlayValues) > 371 && ps.OverlayValues[371].Loc != LocNone {
						d371 = ps.OverlayValues[371]
					}
					if len(ps.OverlayValues) > 447 && ps.OverlayValues[447].Loc != LocNone {
						d447 = ps.OverlayValues[447]
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
					if len(ps.OverlayValues) > 454 && ps.OverlayValues[454].Loc != LocNone {
						d454 = ps.OverlayValues[454]
					}
					if len(ps.OverlayValues) > 455 && ps.OverlayValues[455].Loc != LocNone {
						d455 = ps.OverlayValues[455]
					}
					if len(ps.OverlayValues) > 456 && ps.OverlayValues[456].Loc != LocNone {
						d456 = ps.OverlayValues[456]
					}
					if len(ps.OverlayValues) > 457 && ps.OverlayValues[457].Loc != LocNone {
						d457 = ps.OverlayValues[457]
					}
					if len(ps.OverlayValues) > 458 && ps.OverlayValues[458].Loc != LocNone {
						d458 = ps.OverlayValues[458]
					}
					if len(ps.OverlayValues) > 459 && ps.OverlayValues[459].Loc != LocNone {
						d459 = ps.OverlayValues[459]
					}
					if len(ps.OverlayValues) > 461 && ps.OverlayValues[461].Loc != LocNone {
						d461 = ps.OverlayValues[461]
					}
					if len(ps.OverlayValues) > 462 && ps.OverlayValues[462].Loc != LocNone {
						d462 = ps.OverlayValues[462]
					}
					if len(ps.OverlayValues) > 463 && ps.OverlayValues[463].Loc != LocNone {
						d463 = ps.OverlayValues[463]
					}
					if len(ps.OverlayValues) > 464 && ps.OverlayValues[464].Loc != LocNone {
						d464 = ps.OverlayValues[464]
					}
					if len(ps.OverlayValues) > 465 && ps.OverlayValues[465].Loc != LocNone {
						d465 = ps.OverlayValues[465]
					}
					if len(ps.OverlayValues) > 466 && ps.OverlayValues[466].Loc != LocNone {
						d466 = ps.OverlayValues[466]
					}
					if len(ps.OverlayValues) > 467 && ps.OverlayValues[467].Loc != LocNone {
						d467 = ps.OverlayValues[467]
					}
					if len(ps.OverlayValues) > 468 && ps.OverlayValues[468].Loc != LocNone {
						d468 = ps.OverlayValues[468]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d46)
					d469 = d6
					_ = d469
					ctx.StabilizeDescForControlFlow(&d469)
					d470 = d46
					_ = d470
					ctx.StabilizeDescForControlFlow(&d470)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl29 := ctx.ReserveLabel()
					_ = lbl29
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl29)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d469)
					ctx.EnsureDesc(&d469)
					d469 = JITPrepareScmerGoArg(ctx, d469)
					ctx.EnsureDesc(&d470)
					ctx.EnsureDesc(&d470)
					ctx.EnsureDesc(&d470)
					if d470.Loc != LocRegTriple && d470.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d471 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d471.Loc == LocRegPair || d471.Loc == LocStackPair || d471.Loc == LocRegTriple || d471.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d469)
					ctx.SyncDesc(&d470)
					ctx.SyncDesc(&d471)
					d472 = ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d469, d470, d471}, 2)
					d472.NoHeapPointer = false
					ctx.BindReg(d472.Reg, &d472)
					ctx.BindReg(d472.Reg2, &d472)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d472)
					ctx.FreeDesc(&d6)
					d473 = args[0]
					d473.ID = 0
					ctx.SyncDesc(&d473)
					if d473.Loc == LocRegPair || d473.Loc == LocStackPair || d473.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d473, &result)
						result.Type = d473.Type
					} else {
						switch d473.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d473)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d473)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d473)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d473, &result)
							result.Type = d473.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps474 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps474)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				declaration := declarations["window_flush"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
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
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
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
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d113 JITValueDesc
				_ = d113
				var d114 JITValueDesc
				_ = d114
				var d115 JITValueDesc
				_ = d115
				var d153 JITValueDesc
				_ = d153
				var d154 JITValueDesc
				_ = d154
				var d155 JITValueDesc
				_ = d155
				var d158 JITValueDesc
				_ = d158
				var d198 JITValueDesc
				_ = d198
				var d199 JITValueDesc
				_ = d199
				var d200 JITValueDesc
				_ = d200
				var d201 JITValueDesc
				_ = d201
				var d202 JITValueDesc
				_ = d202
				var d204 JITValueDesc
				_ = d204
				var d205 JITValueDesc
				_ = d205
				var d206 JITValueDesc
				_ = d206
				var d207 JITValueDesc
				_ = d207
				var d209 JITValueDesc
				_ = d209
				var d210 JITValueDesc
				_ = d210
				var d211 JITValueDesc
				_ = d211
				var d212 JITValueDesc
				_ = d212
				var d213 JITValueDesc
				_ = d213
				var d214 JITValueDesc
				_ = d214
				var d217 JITValueDesc
				_ = d217
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
				var d280 JITValueDesc
				_ = d280
				var d281 JITValueDesc
				_ = d281
				var d282 JITValueDesc
				_ = d282
				var d283 JITValueDesc
				_ = d283
				var d284 JITValueDesc
				_ = d284
				var d285 JITValueDesc
				_ = d285
				var d286 JITValueDesc
				_ = d286
				var d287 JITValueDesc
				_ = d287
				var d288 JITValueDesc
				_ = d288
				var d289 JITValueDesc
				_ = d289
				var d290 JITValueDesc
				_ = d290
				var d291 JITValueDesc
				_ = d291
				var d292 JITValueDesc
				_ = d292
				var d293 JITValueDesc
				_ = d293
				var d294 JITValueDesc
				_ = d294
				var d295 JITValueDesc
				_ = d295
				var d297 JITValueDesc
				_ = d297
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				var bbs [13]BBDescriptor
				bbs[7].PhiBase = int32(phiBase0) + int32(0)
				bbs[7].PhiCount = uint16(1)
				bbs[10].PhiBase = int32(phiBase0) + int32(16)
				bbs[10].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 5}, {Color: 1, Width: 1, Cost: 4}}, Count: 2})
				defer ctx.ReleaseRegisterHomes(registerHomes1)
				var r0 Reg
				phiHomeOK2 := registerHomes1.Available&(uint16(1)<<1) == uint16(1)<<1
				if phiHomeOK2 {
					r0 = registerHomes1.Registers[1]
				}
				var r1 Reg
				phiHomeOK3 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK3 {
					r1 = registerHomes1.Registers[0]
				}
				var d4 JITValueDesc
				if phiHomeOK2 {
					d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
					ctx.BindReg(r0, &d4)
				} else {
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				}
				_ = d4
				var d5 JITValueDesc
				if phiHomeOK3 {
					d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
					ctx.BindReg(r1, &d5)
				} else {
					d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				}
				_ = d5
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbpos_0_11 := int32(-1)
				_ = bbpos_0_11
				lbl12 := ctx.ReserveLabel()
				_ = lbl12
				bbpos_0_12 := int32(-1)
				_ = bbpos_0_12
				lbl13 := ctx.ReserveLabel()
				_ = lbl13
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d4)
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d5)
					} else {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					ctx.ReclaimUntrackedRegs()
					d6 = args[0]
					d6.ID = 0
					var d7 JITValueDesc
					if d6.Type == tagSlice {
						d7 = jitKnownSliceHeader(ctx, &d6)
					} else {
						d7 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d6}, 3)
					}
					ctx.BindReg(d7.Reg, &d7)
					ctx.BindReg(d7.Reg2, &d7)
					ctx.BindReg(d7.Reg3, &d7)
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.FreeDesc(&d6)
					d8 = args[1]
					d8.ID = 0
					ctx.StabilizeDescForControlFlow(&d8)
					d9 = args[2]
					d9.ID = 0
					var d10 JITValueDesc
					if d9.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d9.Imm.Int())}
					} else if d9.Type == tagInt && d9.Loc == LocRegPair {
						ctx.FreeReg(d9.Reg)
						d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d9.Reg2}
						ctx.BindReg(d9.Reg2, &d10)
						ctx.BindReg(d9.Reg2, &d10)
					} else if d9.Type == tagInt && d9.Loc == LocReg {
						d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d9.Reg}
						ctx.BindReg(d9.Reg, &d10)
						ctx.BindReg(d9.Reg, &d10)
					} else {
						d10 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d9}, 1)
						d10.Type = tagInt
						ctx.BindReg(d10.Reg, &d10)
					}
					ctx.FreeDesc(&d9)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d10)
					ctx.StabilizeDescForControlFlow(&d10)
					var d12 JITValueDesc
					if d7.SliceSizeKnown {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d7.KnownSliceLen))}
					} else if d7.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d7.StackOff))}
					} else if d7.Loc == LocStackTriple {
						d12 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d7.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d7)
						if d7.Loc == LocRegPair || d7.Loc == LocRegTriple {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d7.Reg2, ID: 0}
						} else if d7.Loc == LocReg {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d7.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d12)
					var d13 JITValueDesc
					if d12.Loc == LocImm {
						d13 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.Int() < 3)}
					} else {
						r2 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d12.Reg, 3)
						ctx.EmitSetcc(r2, CondSignedLess)
						d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d13)
					}
					ctx.FreeDesc(&d12)
					d14 = d13
					ctx.EnsureDesc(&d14)
					if d14.Loc != LocImm && d14.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d14.Loc == LocImm {
						if d14.Imm.Bool() {
							if ps.General {
							}
							ps15 := PhiState{General: ps.General}
							ps15.OverlayValues = make([]JITValueDesc, 15)
							ps15.OverlayValues[4] = d4
							ps15.OverlayValues[5] = d5
							ps15.OverlayValues[6] = d6
							ps15.OverlayValues[7] = d7
							ps15.OverlayValues[8] = d8
							ps15.OverlayValues[9] = d9
							ps15.OverlayValues[10] = d10
							ps15.OverlayValues[11] = d11
							ps15.OverlayValues[12] = d12
							ps15.OverlayValues[13] = d13
							ps15.OverlayValues[14] = d14
							return bbs[1].RenderPS(ps15)
						}
						if ps.General {
						}
						ps16 := PhiState{General: ps.General}
						ps16.OverlayValues = make([]JITValueDesc, 15)
						ps16.OverlayValues[4] = d4
						ps16.OverlayValues[5] = d5
						ps16.OverlayValues[6] = d6
						ps16.OverlayValues[7] = d7
						ps16.OverlayValues[8] = d8
						ps16.OverlayValues[9] = d9
						ps16.OverlayValues[10] = d10
						ps16.OverlayValues[11] = d11
						ps16.OverlayValues[12] = d12
						ps16.OverlayValues[13] = d13
						ps16.OverlayValues[14] = d14
						return bbs[2].RenderPS(ps16)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d14.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl3)
					ps17 := PhiState{General: true}
					ps17.OverlayValues = make([]JITValueDesc, 15)
					ps17.OverlayValues[4] = d4
					ps17.OverlayValues[5] = d5
					ps17.OverlayValues[6] = d6
					ps17.OverlayValues[7] = d7
					ps17.OverlayValues[8] = d8
					ps17.OverlayValues[9] = d9
					ps17.OverlayValues[10] = d10
					ps17.OverlayValues[11] = d11
					ps17.OverlayValues[12] = d12
					ps17.OverlayValues[13] = d13
					ps17.OverlayValues[14] = d14
					ps18 := PhiState{General: true}
					ps18.OverlayValues = make([]JITValueDesc, 15)
					ps18.OverlayValues[4] = d4
					ps18.OverlayValues[5] = d5
					ps18.OverlayValues[6] = d6
					ps18.OverlayValues[7] = d7
					ps18.OverlayValues[8] = d8
					ps18.OverlayValues[9] = d9
					ps18.OverlayValues[10] = d10
					ps18.OverlayValues[11] = d11
					ps18.OverlayValues[12] = d12
					ps18.OverlayValues[13] = d13
					ps18.OverlayValues[14] = d14
					snap19 := d4
					snap20 := d5
					snap21 := d6
					snap22 := d7
					snap23 := d8
					snap24 := d9
					snap25 := d10
					snap26 := d11
					snap27 := d12
					snap28 := d13
					snap29 := d14
					alloc30 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps18)
					}
					ctx.RestoreAllocState(alloc30)
					d4 = snap19
					d5 = snap20
					d6 = snap21
					d7 = snap22
					d8 = snap23
					d9 = snap24
					d10 = snap25
					d11 = snap26
					d12 = snap27
					d13 = snap28
					d14 = snap29
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps17)
					}
					return result
					ctx.FreeDesc(&d13)
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d4)
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d5)
					} else {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d4)
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d5)
					} else {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					ctx.ReclaimUntrackedRegs()
					d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					d33 = ctx.EmitSliceElementAddress(&d7, &d31, 16)
					ctx.EnsureDesc(&d33)
					r3 := ctx.AllocRegExcept(d33.Reg)
					ctx.EmitMovRegMem(r3, d33.Reg, 8)
					ctx.EmitMovRegMem(d33.Reg, d33.Reg, 0)
					d32 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d33.Reg, Reg2: r3}
					ctx.BindReg(d33.Reg, &d32)
					ctx.BindReg(r3, &d32)
					var d34 JITValueDesc
					if d32.Loc == LocImm {
						d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d32.Imm.Int())}
					} else if d32.Type == tagInt && d32.Loc == LocRegPair {
						ctx.FreeReg(d32.Reg)
						d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg2}
						ctx.BindReg(d32.Reg2, &d34)
						ctx.BindReg(d32.Reg2, &d34)
					} else if d32.Type == tagInt && d32.Loc == LocReg {
						d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg}
						ctx.BindReg(d32.Reg, &d34)
						ctx.BindReg(d32.Reg, &d34)
					} else {
						d34 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d32}, 1)
						d34.Type = tagInt
						ctx.BindReg(d34.Reg, &d34)
					}
					ctx.FreeDesc(&d32)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d34)
					ctx.StabilizeDescForControlFlow(&d34)
					d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(3)}
					var d37 JITValueDesc
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocRegPair || d7.Loc == LocRegTriple {
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d7.Reg2}
						ctx.BindReg(d7.Reg2, &d37)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d36)
					ctx.EnsureDesc(&d37)
					var d39 JITValueDesc
					if d37.Loc == LocImm && d36.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d37.Imm.Int() - d36.Imm.Int())}
					} else {
						r4 := ctx.AllocReg()
						if d37.Loc == LocImm {
							ctx.EmitMovRegImm64(r4, uint64(d37.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r4, d37.Reg)
						}
						if d36.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d36.Imm.Int()))
							ctx.EmitSubInt64(r4, RegR11)
						} else {
							ctx.EmitSubInt64(r4, d36.Reg)
						}
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d39)
					}
					var d40 JITValueDesc
					r5 := ctx.EmitSliceDataAfterLow(&d7, &d36, 16)
					d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
					ctx.BindReg(r5, &d40)
					ctx.BindReg(r5, &d40)
					var d41 JITValueDesc
					var r6 Reg
					var r7 Reg
					ctx.SyncDesc(&d40)
					ctx.EnsureDesc(&d40)
					if d40.Loc == LocImm {
						r6 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, uint64(d40.Imm.Int()))
					} else {
						r6 = d40.Reg
					}
					ctx.ProtectReg(r6)
					ctx.SyncDesc(&d39)
					ctx.EnsureDesc(&d39)
					if d39.Loc == LocImm {
						r7 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, uint64(d39.Imm.Int()))
					} else {
						r7 = d39.Reg
					}
					ctx.ProtectReg(r7)
					r8 := ctx.EmitSliceCapAfterLow(&d7, &d36, r6, r7)
					ctx.UnprotectReg(r7)
					ctx.UnprotectReg(r6)
					d41 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8}
					ctx.BindReg(r6, &d41)
					ctx.BindReg(r7, &d41)
					ctx.BindReg(r8, &d41)
					ctx.BindReg(r6, &d41)
					ctx.BindReg(r7, &d41)
					ctx.BindReg(r8, &d41)
					ctx.StabilizeDescForControlFlow(&d41)
					ctx.EnsureDesc(&d34)
					var d42 JITValueDesc
					if d34.Loc == LocImm {
						d42 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d34.Imm.Int() <= 0)}
					} else {
						r9 := ctx.AllocRegExcept(d34.Reg)
						ctx.EmitCmpRegImm32(d34.Reg, 0)
						ctx.EmitSetcc(r9, CondSignedLessOrEqual)
						d42 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
						ctx.BindReg(r9, &d42)
					}
					d43 = d42
					ctx.EnsureDesc(&d43)
					if d43.Loc != LocImm && d43.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d43.Loc == LocImm {
						if d43.Imm.Bool() {
							if ps.General {
							}
							ps44 := PhiState{General: ps.General}
							ps44.OverlayValues = make([]JITValueDesc, 44)
							ps44.OverlayValues[4] = d4
							ps44.OverlayValues[5] = d5
							ps44.OverlayValues[6] = d6
							ps44.OverlayValues[7] = d7
							ps44.OverlayValues[8] = d8
							ps44.OverlayValues[9] = d9
							ps44.OverlayValues[10] = d10
							ps44.OverlayValues[11] = d11
							ps44.OverlayValues[12] = d12
							ps44.OverlayValues[13] = d13
							ps44.OverlayValues[14] = d14
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
							ps44.OverlayValues[41] = d41
							ps44.OverlayValues[42] = d42
							ps44.OverlayValues[43] = d43
							return bbs[3].RenderPS(ps44)
						}
						if ps.General {
						}
						ps45 := PhiState{General: ps.General}
						ps45.OverlayValues = make([]JITValueDesc, 44)
						ps45.OverlayValues[4] = d4
						ps45.OverlayValues[5] = d5
						ps45.OverlayValues[6] = d6
						ps45.OverlayValues[7] = d7
						ps45.OverlayValues[8] = d8
						ps45.OverlayValues[9] = d9
						ps45.OverlayValues[10] = d10
						ps45.OverlayValues[11] = d11
						ps45.OverlayValues[12] = d12
						ps45.OverlayValues[13] = d13
						ps45.OverlayValues[14] = d14
						ps45.OverlayValues[31] = d31
						ps45.OverlayValues[32] = d32
						ps45.OverlayValues[33] = d33
						ps45.OverlayValues[34] = d34
						ps45.OverlayValues[35] = d35
						ps45.OverlayValues[36] = d36
						ps45.OverlayValues[37] = d37
						ps45.OverlayValues[38] = d38
						ps45.OverlayValues[39] = d39
						ps45.OverlayValues[40] = d40
						ps45.OverlayValues[41] = d41
						ps45.OverlayValues[42] = d42
						ps45.OverlayValues[43] = d43
						return bbs[6].RenderPS(ps45)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d43.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl7)
					ps46 := PhiState{General: true}
					ps46.OverlayValues = make([]JITValueDesc, 44)
					ps46.OverlayValues[4] = d4
					ps46.OverlayValues[5] = d5
					ps46.OverlayValues[6] = d6
					ps46.OverlayValues[7] = d7
					ps46.OverlayValues[8] = d8
					ps46.OverlayValues[9] = d9
					ps46.OverlayValues[10] = d10
					ps46.OverlayValues[11] = d11
					ps46.OverlayValues[12] = d12
					ps46.OverlayValues[13] = d13
					ps46.OverlayValues[14] = d14
					ps46.OverlayValues[31] = d31
					ps46.OverlayValues[32] = d32
					ps46.OverlayValues[33] = d33
					ps46.OverlayValues[34] = d34
					ps46.OverlayValues[35] = d35
					ps46.OverlayValues[36] = d36
					ps46.OverlayValues[37] = d37
					ps46.OverlayValues[38] = d38
					ps46.OverlayValues[39] = d39
					ps46.OverlayValues[40] = d40
					ps46.OverlayValues[41] = d41
					ps46.OverlayValues[42] = d42
					ps46.OverlayValues[43] = d43
					ps47 := PhiState{General: true}
					ps47.OverlayValues = make([]JITValueDesc, 44)
					ps47.OverlayValues[4] = d4
					ps47.OverlayValues[5] = d5
					ps47.OverlayValues[6] = d6
					ps47.OverlayValues[7] = d7
					ps47.OverlayValues[8] = d8
					ps47.OverlayValues[9] = d9
					ps47.OverlayValues[10] = d10
					ps47.OverlayValues[11] = d11
					ps47.OverlayValues[12] = d12
					ps47.OverlayValues[13] = d13
					ps47.OverlayValues[14] = d14
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
					snap48 := d4
					snap49 := d5
					snap50 := d6
					snap51 := d7
					snap52 := d8
					snap53 := d9
					snap54 := d10
					snap55 := d11
					snap56 := d12
					snap57 := d13
					snap58 := d14
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
					snap69 := d41
					snap70 := d42
					snap71 := d43
					alloc72 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps47)
					}
					ctx.RestoreAllocState(alloc72)
					d4 = snap48
					d5 = snap49
					d6 = snap50
					d7 = snap51
					d8 = snap52
					d9 = snap53
					d10 = snap54
					d11 = snap55
					d12 = snap56
					d13 = snap57
					d14 = snap58
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
					d41 = snap69
					d42 = snap70
					d43 = snap71
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps46)
					}
					return result
					ctx.FreeDesc(&d42)
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d4)
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d5)
					} else {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d4)
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d5)
					} else {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[7].PhiBase)+int32(0))
						}
					}
					ps73 := PhiState{General: ps.General}
					ps73.OverlayValues = make([]JITValueDesc, 44)
					ps73.OverlayValues[4] = d4
					ps73.OverlayValues[5] = d5
					ps73.OverlayValues[6] = d6
					ps73.OverlayValues[7] = d7
					ps73.OverlayValues[8] = d8
					ps73.OverlayValues[9] = d9
					ps73.OverlayValues[10] = d10
					ps73.OverlayValues[11] = d11
					ps73.OverlayValues[12] = d12
					ps73.OverlayValues[13] = d13
					ps73.OverlayValues[14] = d14
					ps73.OverlayValues[31] = d31
					ps73.OverlayValues[32] = d32
					ps73.OverlayValues[33] = d33
					ps73.OverlayValues[34] = d34
					ps73.OverlayValues[35] = d35
					ps73.OverlayValues[36] = d36
					ps73.OverlayValues[37] = d37
					ps73.OverlayValues[38] = d38
					ps73.OverlayValues[39] = d39
					ps73.OverlayValues[40] = d40
					ps73.OverlayValues[41] = d41
					ps73.OverlayValues[42] = d42
					ps73.OverlayValues[43] = d43
					ps73.PhiValues = make([]JITValueDesc, 1)
					d74 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps73.PhiValues[0] = d74
					if ps73.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps73)
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d4)
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d5)
					} else {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					ctx.ReclaimUntrackedRegs()
					var d75 JITValueDesc
					if d41.SliceSizeKnown {
						d75 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d41.KnownSliceLen))}
					} else if d41.Loc == LocImm {
						d75 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d41.StackOff))}
					} else if d41.Loc == LocStackTriple {
						d75 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d41.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d41)
						if d41.Loc == LocRegPair || d41.Loc == LocRegTriple {
							d75 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg2, ID: 0}
						} else if d41.Loc == LocReg {
							d75 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d75)
					ctx.EnsureDesc(&d34)
					var d76 JITValueDesc
					if d75.Loc == LocImm && d34.Loc == LocImm {
						d76 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d75.Imm.Int() % d34.Imm.Int())}
					} else {
						d76 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntRem), []JITValueDesc{d75, d34}, 1)
					}
					if d76.Loc == LocReg && d75.Loc == LocReg && d76.Reg == d75.Reg {
						ctx.TransferReg(d75.Reg)
						d75.Loc = LocNone
					}
					ctx.FreeDesc(&d75)
					ctx.EnsureDesc(&d76)
					var d77 JITValueDesc
					if d76.Loc == LocImm {
						d77 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d76.Imm.Int() != 0)}
					} else {
						r10 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d76.Reg, 0)
						ctx.EmitSetcc(r10, CondNotEqual)
						d77 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
						ctx.BindReg(r10, &d77)
					}
					ctx.FreeDesc(&d76)
					d78 = d77
					ctx.EnsureDesc(&d78)
					if d78.Loc != LocImm && d78.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d78.Loc == LocImm {
						if d78.Imm.Bool() {
							if ps.General {
							}
							ps79 := PhiState{General: ps.General}
							ps79.OverlayValues = make([]JITValueDesc, 79)
							ps79.OverlayValues[4] = d4
							ps79.OverlayValues[5] = d5
							ps79.OverlayValues[6] = d6
							ps79.OverlayValues[7] = d7
							ps79.OverlayValues[8] = d8
							ps79.OverlayValues[9] = d9
							ps79.OverlayValues[10] = d10
							ps79.OverlayValues[11] = d11
							ps79.OverlayValues[12] = d12
							ps79.OverlayValues[13] = d13
							ps79.OverlayValues[14] = d14
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
							ps79.OverlayValues[74] = d74
							ps79.OverlayValues[75] = d75
							ps79.OverlayValues[76] = d76
							ps79.OverlayValues[77] = d77
							ps79.OverlayValues[78] = d78
							return bbs[3].RenderPS(ps79)
						}
						if ps.General {
						}
						ps80 := PhiState{General: ps.General}
						ps80.OverlayValues = make([]JITValueDesc, 79)
						ps80.OverlayValues[4] = d4
						ps80.OverlayValues[5] = d5
						ps80.OverlayValues[6] = d6
						ps80.OverlayValues[7] = d7
						ps80.OverlayValues[8] = d8
						ps80.OverlayValues[9] = d9
						ps80.OverlayValues[10] = d10
						ps80.OverlayValues[11] = d11
						ps80.OverlayValues[12] = d12
						ps80.OverlayValues[13] = d13
						ps80.OverlayValues[14] = d14
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
						ps80.OverlayValues[74] = d74
						ps80.OverlayValues[75] = d75
						ps80.OverlayValues[76] = d76
						ps80.OverlayValues[77] = d77
						ps80.OverlayValues[78] = d78
						return bbs[4].RenderPS(ps80)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d78.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl18)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl5)
					ps81 := PhiState{General: true}
					ps81.OverlayValues = make([]JITValueDesc, 79)
					ps81.OverlayValues[4] = d4
					ps81.OverlayValues[5] = d5
					ps81.OverlayValues[6] = d6
					ps81.OverlayValues[7] = d7
					ps81.OverlayValues[8] = d8
					ps81.OverlayValues[9] = d9
					ps81.OverlayValues[10] = d10
					ps81.OverlayValues[11] = d11
					ps81.OverlayValues[12] = d12
					ps81.OverlayValues[13] = d13
					ps81.OverlayValues[14] = d14
					ps81.OverlayValues[31] = d31
					ps81.OverlayValues[32] = d32
					ps81.OverlayValues[33] = d33
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
					ps81.OverlayValues[74] = d74
					ps81.OverlayValues[75] = d75
					ps81.OverlayValues[76] = d76
					ps81.OverlayValues[77] = d77
					ps81.OverlayValues[78] = d78
					ps82 := PhiState{General: true}
					ps82.OverlayValues = make([]JITValueDesc, 79)
					ps82.OverlayValues[4] = d4
					ps82.OverlayValues[5] = d5
					ps82.OverlayValues[6] = d6
					ps82.OverlayValues[7] = d7
					ps82.OverlayValues[8] = d8
					ps82.OverlayValues[9] = d9
					ps82.OverlayValues[10] = d10
					ps82.OverlayValues[11] = d11
					ps82.OverlayValues[12] = d12
					ps82.OverlayValues[13] = d13
					ps82.OverlayValues[14] = d14
					ps82.OverlayValues[31] = d31
					ps82.OverlayValues[32] = d32
					ps82.OverlayValues[33] = d33
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
					ps82.OverlayValues[74] = d74
					ps82.OverlayValues[75] = d75
					ps82.OverlayValues[76] = d76
					ps82.OverlayValues[77] = d77
					ps82.OverlayValues[78] = d78
					snap83 := d4
					snap84 := d5
					snap85 := d6
					snap86 := d7
					snap87 := d8
					snap88 := d9
					snap89 := d10
					snap90 := d11
					snap91 := d12
					snap92 := d13
					snap93 := d14
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
					snap104 := d41
					snap105 := d42
					snap106 := d43
					snap107 := d74
					snap108 := d75
					snap109 := d76
					snap110 := d77
					snap111 := d78
					alloc112 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps82)
					}
					ctx.RestoreAllocState(alloc112)
					d4 = snap83
					d5 = snap84
					d6 = snap85
					d7 = snap86
					d8 = snap87
					d9 = snap88
					d10 = snap89
					d11 = snap90
					d12 = snap91
					d13 = snap92
					d14 = snap93
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
					d41 = snap104
					d42 = snap105
					d43 = snap106
					d74 = snap107
					d75 = snap108
					d76 = snap109
					d77 = snap110
					d78 = snap111
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps81)
					}
					return result
					ctx.FreeDesc(&d77)
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d4)
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d5)
					} else {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					ctx.ReclaimUntrackedRegs()
					var d113 JITValueDesc
					if d41.SliceSizeKnown {
						d113 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d41.KnownSliceLen))}
					} else if d41.Loc == LocImm {
						d113 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d41.StackOff))}
					} else if d41.Loc == LocStackTriple {
						d113 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d41.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d41)
						if d41.Loc == LocRegPair || d41.Loc == LocRegTriple {
							d113 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg2, ID: 0}
						} else if d41.Loc == LocReg {
							d113 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d113)
					var d114 JITValueDesc
					if d113.Loc == LocImm {
						d114 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d113.Imm.Int() == 0)}
					} else {
						r11 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d113.Reg, 0)
						ctx.EmitSetcc(r11, CondEqual)
						d114 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
						ctx.BindReg(r11, &d114)
					}
					ctx.FreeDesc(&d113)
					d115 = d114
					ctx.EnsureDesc(&d115)
					if d115.Loc != LocImm && d115.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d115.Loc == LocImm {
						if d115.Imm.Bool() {
							if ps.General {
							}
							ps116 := PhiState{General: ps.General}
							ps116.OverlayValues = make([]JITValueDesc, 116)
							ps116.OverlayValues[4] = d4
							ps116.OverlayValues[5] = d5
							ps116.OverlayValues[6] = d6
							ps116.OverlayValues[7] = d7
							ps116.OverlayValues[8] = d8
							ps116.OverlayValues[9] = d9
							ps116.OverlayValues[10] = d10
							ps116.OverlayValues[11] = d11
							ps116.OverlayValues[12] = d12
							ps116.OverlayValues[13] = d13
							ps116.OverlayValues[14] = d14
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
							ps116.OverlayValues[41] = d41
							ps116.OverlayValues[42] = d42
							ps116.OverlayValues[43] = d43
							ps116.OverlayValues[74] = d74
							ps116.OverlayValues[75] = d75
							ps116.OverlayValues[76] = d76
							ps116.OverlayValues[77] = d77
							ps116.OverlayValues[78] = d78
							ps116.OverlayValues[113] = d113
							ps116.OverlayValues[114] = d114
							ps116.OverlayValues[115] = d115
							return bbs[3].RenderPS(ps116)
						}
						if ps.General {
						}
						ps117 := PhiState{General: ps.General}
						ps117.OverlayValues = make([]JITValueDesc, 116)
						ps117.OverlayValues[4] = d4
						ps117.OverlayValues[5] = d5
						ps117.OverlayValues[6] = d6
						ps117.OverlayValues[7] = d7
						ps117.OverlayValues[8] = d8
						ps117.OverlayValues[9] = d9
						ps117.OverlayValues[10] = d10
						ps117.OverlayValues[11] = d11
						ps117.OverlayValues[12] = d12
						ps117.OverlayValues[13] = d13
						ps117.OverlayValues[14] = d14
						ps117.OverlayValues[31] = d31
						ps117.OverlayValues[32] = d32
						ps117.OverlayValues[33] = d33
						ps117.OverlayValues[34] = d34
						ps117.OverlayValues[35] = d35
						ps117.OverlayValues[36] = d36
						ps117.OverlayValues[37] = d37
						ps117.OverlayValues[38] = d38
						ps117.OverlayValues[39] = d39
						ps117.OverlayValues[40] = d40
						ps117.OverlayValues[41] = d41
						ps117.OverlayValues[42] = d42
						ps117.OverlayValues[43] = d43
						ps117.OverlayValues[74] = d74
						ps117.OverlayValues[75] = d75
						ps117.OverlayValues[76] = d76
						ps117.OverlayValues[77] = d77
						ps117.OverlayValues[78] = d78
						ps117.OverlayValues[113] = d113
						ps117.OverlayValues[114] = d114
						ps117.OverlayValues[115] = d115
						return bbs[5].RenderPS(ps117)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d115.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl6)
					ps118 := PhiState{General: true}
					ps118.OverlayValues = make([]JITValueDesc, 116)
					ps118.OverlayValues[4] = d4
					ps118.OverlayValues[5] = d5
					ps118.OverlayValues[6] = d6
					ps118.OverlayValues[7] = d7
					ps118.OverlayValues[8] = d8
					ps118.OverlayValues[9] = d9
					ps118.OverlayValues[10] = d10
					ps118.OverlayValues[11] = d11
					ps118.OverlayValues[12] = d12
					ps118.OverlayValues[13] = d13
					ps118.OverlayValues[14] = d14
					ps118.OverlayValues[31] = d31
					ps118.OverlayValues[32] = d32
					ps118.OverlayValues[33] = d33
					ps118.OverlayValues[34] = d34
					ps118.OverlayValues[35] = d35
					ps118.OverlayValues[36] = d36
					ps118.OverlayValues[37] = d37
					ps118.OverlayValues[38] = d38
					ps118.OverlayValues[39] = d39
					ps118.OverlayValues[40] = d40
					ps118.OverlayValues[41] = d41
					ps118.OverlayValues[42] = d42
					ps118.OverlayValues[43] = d43
					ps118.OverlayValues[74] = d74
					ps118.OverlayValues[75] = d75
					ps118.OverlayValues[76] = d76
					ps118.OverlayValues[77] = d77
					ps118.OverlayValues[78] = d78
					ps118.OverlayValues[113] = d113
					ps118.OverlayValues[114] = d114
					ps118.OverlayValues[115] = d115
					ps119 := PhiState{General: true}
					ps119.OverlayValues = make([]JITValueDesc, 116)
					ps119.OverlayValues[4] = d4
					ps119.OverlayValues[5] = d5
					ps119.OverlayValues[6] = d6
					ps119.OverlayValues[7] = d7
					ps119.OverlayValues[8] = d8
					ps119.OverlayValues[9] = d9
					ps119.OverlayValues[10] = d10
					ps119.OverlayValues[11] = d11
					ps119.OverlayValues[12] = d12
					ps119.OverlayValues[13] = d13
					ps119.OverlayValues[14] = d14
					ps119.OverlayValues[31] = d31
					ps119.OverlayValues[32] = d32
					ps119.OverlayValues[33] = d33
					ps119.OverlayValues[34] = d34
					ps119.OverlayValues[35] = d35
					ps119.OverlayValues[36] = d36
					ps119.OverlayValues[37] = d37
					ps119.OverlayValues[38] = d38
					ps119.OverlayValues[39] = d39
					ps119.OverlayValues[40] = d40
					ps119.OverlayValues[41] = d41
					ps119.OverlayValues[42] = d42
					ps119.OverlayValues[43] = d43
					ps119.OverlayValues[74] = d74
					ps119.OverlayValues[75] = d75
					ps119.OverlayValues[76] = d76
					ps119.OverlayValues[77] = d77
					ps119.OverlayValues[78] = d78
					ps119.OverlayValues[113] = d113
					ps119.OverlayValues[114] = d114
					ps119.OverlayValues[115] = d115
					snap120 := d4
					snap121 := d5
					snap122 := d6
					snap123 := d7
					snap124 := d8
					snap125 := d9
					snap126 := d10
					snap127 := d11
					snap128 := d12
					snap129 := d13
					snap130 := d14
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
					snap141 := d41
					snap142 := d42
					snap143 := d43
					snap144 := d74
					snap145 := d75
					snap146 := d76
					snap147 := d77
					snap148 := d78
					snap149 := d113
					snap150 := d114
					snap151 := d115
					alloc152 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps119)
					}
					ctx.RestoreAllocState(alloc152)
					d4 = snap120
					d5 = snap121
					d6 = snap122
					d7 = snap123
					d8 = snap124
					d9 = snap125
					d10 = snap126
					d11 = snap127
					d12 = snap128
					d13 = snap129
					d14 = snap130
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
					d41 = snap141
					d42 = snap142
					d43 = snap143
					d74 = snap144
					d75 = snap145
					d76 = snap146
					d77 = snap147
					d78 = snap148
					d113 = snap149
					d114 = snap150
					d115 = snap151
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps118)
					}
					return result
					ctx.FreeDesc(&d114)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d153 := ps.PhiValues[0]
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d153)
							} else {
								ctx.EmitStoreToStack(d153, int32(bbs[7].PhiBase)+int32(0))
							}
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d4)
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d5)
					} else {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d4 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDescsTogether(&d4, &d10)
					var d154 JITValueDesc
					if d4.Loc == LocImm && d10.Loc == LocImm {
						d154 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() < d10.Imm.Int())}
					} else if d10.Loc == LocImm {
						r12 := ctx.AllocRegExcept(d4.Reg)
						if d10.Imm.Int() >= -2147483648 && d10.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d4.Reg, int32(d10.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d10.Imm.Int()))
							ctx.EmitCmpInt64(d4.Reg, RegR11)
						}
						ctx.EmitSetcc(r12, CondSignedLess)
						d154 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
						ctx.BindReg(r12, &d154)
					} else if d4.Loc == LocImm {
						r13 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d10.Reg)
						ctx.EmitSetcc(r13, CondSignedLess)
						d154 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
						ctx.BindReg(r13, &d154)
					} else {
						r14 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitCmpInt64(d4.Reg, d10.Reg)
						ctx.EmitSetcc(r14, CondSignedLess)
						d154 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
						ctx.BindReg(r14, &d154)
					}
					ctx.FreeDesc(&d10)
					d155 = d154
					ctx.EnsureDesc(&d155)
					if d155.Loc != LocImm && d155.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d155.Loc == LocImm {
						if d155.Imm.Bool() {
							if ps.General {
							}
							ps156 := PhiState{General: ps.General}
							ps156.OverlayValues = make([]JITValueDesc, 156)
							ps156.OverlayValues[4] = d4
							ps156.OverlayValues[5] = d5
							ps156.OverlayValues[6] = d6
							ps156.OverlayValues[7] = d7
							ps156.OverlayValues[8] = d8
							ps156.OverlayValues[9] = d9
							ps156.OverlayValues[10] = d10
							ps156.OverlayValues[11] = d11
							ps156.OverlayValues[12] = d12
							ps156.OverlayValues[13] = d13
							ps156.OverlayValues[14] = d14
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
							ps156.OverlayValues[41] = d41
							ps156.OverlayValues[42] = d42
							ps156.OverlayValues[43] = d43
							ps156.OverlayValues[74] = d74
							ps156.OverlayValues[75] = d75
							ps156.OverlayValues[76] = d76
							ps156.OverlayValues[77] = d77
							ps156.OverlayValues[78] = d78
							ps156.OverlayValues[113] = d113
							ps156.OverlayValues[114] = d114
							ps156.OverlayValues[115] = d115
							ps156.OverlayValues[153] = d153
							ps156.OverlayValues[154] = d154
							ps156.OverlayValues[155] = d155
							return bbs[8].RenderPS(ps156)
						}
						if ps.General {
						}
						ps157 := PhiState{General: ps.General}
						ps157.OverlayValues = make([]JITValueDesc, 156)
						ps157.OverlayValues[4] = d4
						ps157.OverlayValues[5] = d5
						ps157.OverlayValues[6] = d6
						ps157.OverlayValues[7] = d7
						ps157.OverlayValues[8] = d8
						ps157.OverlayValues[9] = d9
						ps157.OverlayValues[10] = d10
						ps157.OverlayValues[11] = d11
						ps157.OverlayValues[12] = d12
						ps157.OverlayValues[13] = d13
						ps157.OverlayValues[14] = d14
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
						ps157.OverlayValues[41] = d41
						ps157.OverlayValues[42] = d42
						ps157.OverlayValues[43] = d43
						ps157.OverlayValues[74] = d74
						ps157.OverlayValues[75] = d75
						ps157.OverlayValues[76] = d76
						ps157.OverlayValues[77] = d77
						ps157.OverlayValues[78] = d78
						ps157.OverlayValues[113] = d113
						ps157.OverlayValues[114] = d114
						ps157.OverlayValues[115] = d115
						ps157.OverlayValues[153] = d153
						ps157.OverlayValues[154] = d154
						ps157.OverlayValues[155] = d155
						return bbs[9].RenderPS(ps157)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d158 := ps.PhiValues[0]
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d158)
							} else {
								ctx.EmitStoreToStack(d158, int32(bbs[7].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d155.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl22)
					ctx.EmitJmp(lbl23)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl10)
					ps159 := PhiState{General: true}
					ps159.OverlayValues = make([]JITValueDesc, 159)
					ps159.OverlayValues[4] = d4
					ps159.OverlayValues[5] = d5
					ps159.OverlayValues[6] = d6
					ps159.OverlayValues[7] = d7
					ps159.OverlayValues[8] = d8
					ps159.OverlayValues[9] = d9
					ps159.OverlayValues[10] = d10
					ps159.OverlayValues[11] = d11
					ps159.OverlayValues[12] = d12
					ps159.OverlayValues[13] = d13
					ps159.OverlayValues[14] = d14
					ps159.OverlayValues[31] = d31
					ps159.OverlayValues[32] = d32
					ps159.OverlayValues[33] = d33
					ps159.OverlayValues[34] = d34
					ps159.OverlayValues[35] = d35
					ps159.OverlayValues[36] = d36
					ps159.OverlayValues[37] = d37
					ps159.OverlayValues[38] = d38
					ps159.OverlayValues[39] = d39
					ps159.OverlayValues[40] = d40
					ps159.OverlayValues[41] = d41
					ps159.OverlayValues[42] = d42
					ps159.OverlayValues[43] = d43
					ps159.OverlayValues[74] = d74
					ps159.OverlayValues[75] = d75
					ps159.OverlayValues[76] = d76
					ps159.OverlayValues[77] = d77
					ps159.OverlayValues[78] = d78
					ps159.OverlayValues[113] = d113
					ps159.OverlayValues[114] = d114
					ps159.OverlayValues[115] = d115
					ps159.OverlayValues[153] = d153
					ps159.OverlayValues[154] = d154
					ps159.OverlayValues[155] = d155
					ps159.OverlayValues[158] = d158
					ps160 := PhiState{General: true}
					ps160.OverlayValues = make([]JITValueDesc, 159)
					ps160.OverlayValues[4] = d4
					ps160.OverlayValues[5] = d5
					ps160.OverlayValues[6] = d6
					ps160.OverlayValues[7] = d7
					ps160.OverlayValues[8] = d8
					ps160.OverlayValues[9] = d9
					ps160.OverlayValues[10] = d10
					ps160.OverlayValues[11] = d11
					ps160.OverlayValues[12] = d12
					ps160.OverlayValues[13] = d13
					ps160.OverlayValues[14] = d14
					ps160.OverlayValues[31] = d31
					ps160.OverlayValues[32] = d32
					ps160.OverlayValues[33] = d33
					ps160.OverlayValues[34] = d34
					ps160.OverlayValues[35] = d35
					ps160.OverlayValues[36] = d36
					ps160.OverlayValues[37] = d37
					ps160.OverlayValues[38] = d38
					ps160.OverlayValues[39] = d39
					ps160.OverlayValues[40] = d40
					ps160.OverlayValues[41] = d41
					ps160.OverlayValues[42] = d42
					ps160.OverlayValues[43] = d43
					ps160.OverlayValues[74] = d74
					ps160.OverlayValues[75] = d75
					ps160.OverlayValues[76] = d76
					ps160.OverlayValues[77] = d77
					ps160.OverlayValues[78] = d78
					ps160.OverlayValues[113] = d113
					ps160.OverlayValues[114] = d114
					ps160.OverlayValues[115] = d115
					ps160.OverlayValues[153] = d153
					ps160.OverlayValues[154] = d154
					ps160.OverlayValues[155] = d155
					ps160.OverlayValues[158] = d158
					snap161 := d4
					snap162 := d5
					snap163 := d6
					snap164 := d7
					snap165 := d8
					snap166 := d9
					snap167 := d10
					snap168 := d11
					snap169 := d12
					snap170 := d13
					snap171 := d14
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
					snap182 := d41
					snap183 := d42
					snap184 := d43
					snap185 := d74
					snap186 := d75
					snap187 := d76
					snap188 := d77
					snap189 := d78
					snap190 := d113
					snap191 := d114
					snap192 := d115
					snap193 := d153
					snap194 := d154
					snap195 := d155
					snap196 := d158
					alloc197 := ctx.SnapshotAllocState()
					if !bbs[9].Rendered {
						bbs[9].RenderPS(ps160)
					}
					ctx.RestoreAllocState(alloc197)
					d4 = snap161
					d5 = snap162
					d6 = snap163
					d7 = snap164
					d8 = snap165
					d9 = snap166
					d10 = snap167
					d11 = snap168
					d12 = snap169
					d13 = snap170
					d14 = snap171
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
					d41 = snap182
					d42 = snap183
					d43 = snap184
					d74 = snap185
					d75 = snap186
					d76 = snap187
					d77 = snap188
					d78 = snap189
					d113 = snap190
					d114 = snap191
					d115 = snap192
					d153 = snap193
					d154 = snap194
					d155 = snap195
					d158 = snap196
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps159)
					}
					return result
					ctx.FreeDesc(&d154)
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d4)
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d5)
					} else {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d34)
					var d198 JITValueDesc
					ctx.EnsureDesc(&d41)
					if d41.Loc == LocRegPair || d41.Loc == LocRegTriple {
						d198 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg2}
						ctx.BindReg(d41.Reg2, &d198)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d41)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d198)
					var d200 JITValueDesc
					if d198.Loc == LocImm && d34.Loc == LocImm {
						d200 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d198.Imm.Int() - d34.Imm.Int())}
					} else {
						r15 := ctx.AllocReg()
						if d198.Loc == LocImm {
							ctx.EmitMovRegImm64(r15, uint64(d198.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r15, d198.Reg)
						}
						if d34.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d34.Imm.Int()))
							ctx.EmitSubInt64(r15, RegR11)
						} else {
							ctx.EmitSubInt64(r15, d34.Reg)
						}
						d200 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
						ctx.BindReg(r15, &d200)
					}
					var d201 JITValueDesc
					r16 := ctx.EmitSliceDataAfterLow(&d41, &d34, 16)
					d201 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
					ctx.BindReg(r16, &d201)
					ctx.BindReg(r16, &d201)
					var d202 JITValueDesc
					var r17 Reg
					var r18 Reg
					ctx.SyncDesc(&d201)
					ctx.EnsureDesc(&d201)
					if d201.Loc == LocImm {
						r17 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r17, uint64(d201.Imm.Int()))
					} else {
						r17 = d201.Reg
					}
					ctx.ProtectReg(r17)
					ctx.SyncDesc(&d200)
					ctx.EnsureDesc(&d200)
					if d200.Loc == LocImm {
						r18 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r18, uint64(d200.Imm.Int()))
					} else {
						r18 = d200.Reg
					}
					ctx.ProtectReg(r18)
					r19 := ctx.EmitSliceCapAfterLow(&d41, &d34, r17, r18)
					ctx.UnprotectReg(r18)
					ctx.UnprotectReg(r17)
					d202 = JITValueDesc{Loc: LocRegTriple, Reg: r17, Reg2: r18, Reg3: r19}
					ctx.BindReg(r17, &d202)
					ctx.BindReg(r18, &d202)
					ctx.BindReg(r19, &d202)
					ctx.BindReg(r17, &d202)
					ctx.BindReg(r18, &d202)
					ctx.BindReg(r19, &d202)
					ctx.EnsureDesc(&d41)
					ctx.EnsureDesc(&d202)
					ctx.EnsureDesc(&d41)
					ctx.EnsureDesc(&d202)
					callResults203 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d41, d202}, []uint8{1}, []uint8{0})
					d204 = callResults203[0]
					d204.Type = tagInt
					var d205 JITValueDesc
					if d41.SliceSizeKnown {
						d205 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d41.KnownSliceLen))}
					} else if d41.Loc == LocImm {
						d205 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d41.StackOff))}
					} else if d41.Loc == LocStackTriple {
						d205 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d41.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d41)
						if d41.Loc == LocRegPair || d41.Loc == LocRegTriple {
							d205 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg2, ID: 0}
						} else if d41.Loc == LocReg {
							d205 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d205)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDescsTogether(&d205, &d34)
					var d206 JITValueDesc
					if d205.Loc == LocImm && d34.Loc == LocImm {
						d206 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d205.Imm.Int() - d34.Imm.Int())}
					} else if d34.Loc == LocImm && d34.Imm.Int() == 0 {
						var r20 Reg
						if phiHomeOK3 && r1 != d205.Reg {
							r20 = r1
						} else {
							r20 = ctx.AllocRegExcept(d205.Reg)
						}
						ctx.EmitMovRegReg(r20, d205.Reg)
						d206 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r20}
						ctx.BindReg(r20, &d206)
					} else if d205.Loc == LocImm {
						var scratch Reg
						if phiHomeOK3 && r1 != d34.Reg {
							scratch = r1
						} else {
							scratch = ctx.AllocRegExcept(d34.Reg)
						}
						ctx.EmitMovRegImm64(scratch, uint64(d205.Imm.Int()))
						ctx.EmitSubInt64(scratch, d34.Reg)
						d206 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d206)
					} else if d34.Loc == LocImm {
						var scratch Reg
						if phiHomeOK3 && r1 != d205.Reg {
							scratch = r1
						} else {
							scratch = ctx.AllocRegExcept(d205.Reg)
						}
						ctx.EmitMovRegReg(scratch, d205.Reg)
						if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d34.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d34.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d206 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d206)
					} else {
						var r21 Reg
						if phiHomeOK3 && r1 != d205.Reg && r1 != d34.Reg {
							r21 = r1
						} else {
							r21 = ctx.AllocRegExcept(d205.Reg, d34.Reg)
						}
						ctx.EmitMovRegReg(r21, d205.Reg)
						ctx.EmitSubInt64(r21, d34.Reg)
						d206 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r21}
						ctx.BindReg(r21, &d206)
					}
					if d206.Loc == LocReg && d205.Loc == LocReg && d206.Reg == d205.Reg {
						ctx.TransferReg(d205.Reg)
						d205.Loc = LocNone
					}
					ctx.FreeDesc(&d205)
					ctx.FreeDesc(&d34)
					if ps.General {
						ctx.SyncDesc(&d206)
						if d206.Loc == LocReg {
							ctx.ProtectReg(d206.Reg)
						} else if d206.Loc == LocRegPair {
							ctx.ProtectReg(d206.Reg)
							ctx.ProtectReg(d206.Reg2)
						}
						d207 = d206
						if d207.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d207)
						if phiHomeOK3 {
							ctx.EmitMovToReg(r1, d207)
						} else {
							ctx.EmitStoreToStack(d207, int32(bbs[10].PhiBase)+int32(0))
						}
						if d206.Loc == LocReg {
							ctx.UnprotectReg(d206.Reg)
						} else if d206.Loc == LocRegPair {
							ctx.UnprotectReg(d206.Reg)
							ctx.UnprotectReg(d206.Reg2)
						}
					}
					ps208 := PhiState{General: ps.General}
					ps208.OverlayValues = make([]JITValueDesc, 208)
					ps208.OverlayValues[4] = d4
					ps208.OverlayValues[5] = d5
					ps208.OverlayValues[6] = d6
					ps208.OverlayValues[7] = d7
					ps208.OverlayValues[8] = d8
					ps208.OverlayValues[9] = d9
					ps208.OverlayValues[10] = d10
					ps208.OverlayValues[11] = d11
					ps208.OverlayValues[12] = d12
					ps208.OverlayValues[13] = d13
					ps208.OverlayValues[14] = d14
					ps208.OverlayValues[31] = d31
					ps208.OverlayValues[32] = d32
					ps208.OverlayValues[33] = d33
					ps208.OverlayValues[34] = d34
					ps208.OverlayValues[35] = d35
					ps208.OverlayValues[36] = d36
					ps208.OverlayValues[37] = d37
					ps208.OverlayValues[38] = d38
					ps208.OverlayValues[39] = d39
					ps208.OverlayValues[40] = d40
					ps208.OverlayValues[41] = d41
					ps208.OverlayValues[42] = d42
					ps208.OverlayValues[43] = d43
					ps208.OverlayValues[74] = d74
					ps208.OverlayValues[75] = d75
					ps208.OverlayValues[76] = d76
					ps208.OverlayValues[77] = d77
					ps208.OverlayValues[78] = d78
					ps208.OverlayValues[113] = d113
					ps208.OverlayValues[114] = d114
					ps208.OverlayValues[115] = d115
					ps208.OverlayValues[153] = d153
					ps208.OverlayValues[154] = d154
					ps208.OverlayValues[155] = d155
					ps208.OverlayValues[158] = d158
					ps208.OverlayValues[198] = d198
					ps208.OverlayValues[199] = d199
					ps208.OverlayValues[200] = d200
					ps208.OverlayValues[201] = d201
					ps208.OverlayValues[202] = d202
					ps208.OverlayValues[204] = d204
					ps208.OverlayValues[205] = d205
					ps208.OverlayValues[206] = d206
					ps208.OverlayValues[207] = d207
					ps208.PhiValues = make([]JITValueDesc, 1)
					d209 = d206
					ps208.PhiValues[0] = d209
					if ps208.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps208)
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d4)
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d5)
					} else {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
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
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					ctx.ReclaimUntrackedRegs()
					d210 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d210)
					if d210.Loc == LocRegPair || d210.Loc == LocStackPair || d210.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d210, &result)
						result.Type = d210.Type
					} else {
						switch d210.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d210)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d210)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d210)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d210, &result)
							result.Type = d210.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d211 := ps.PhiValues[0]
							if phiHomeOK3 {
								ctx.EmitMovToReg(r1, d211)
							} else {
								ctx.EmitStoreToStack(d211, int32(bbs[10].PhiBase)+int32(0))
							}
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d4)
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d5)
					} else {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
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
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d5 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					var d212 JITValueDesc
					if d41.SliceSizeKnown {
						d212 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d41.KnownSliceLen))}
					} else if d41.Loc == LocImm {
						d212 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d41.StackOff))}
					} else if d41.Loc == LocStackTriple {
						d212 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d41.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d41)
						if d41.Loc == LocRegPair || d41.Loc == LocRegTriple {
							d212 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg2, ID: 0}
						} else if d41.Loc == LocReg {
							d212 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d212)
					ctx.EnsureDescsTogether(&d5, &d212)
					var d213 JITValueDesc
					if d5.Loc == LocImm && d212.Loc == LocImm {
						d213 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.Int() < d212.Imm.Int())}
					} else if d212.Loc == LocImm {
						r22 := ctx.AllocRegExcept(d5.Reg)
						if d212.Imm.Int() >= -2147483648 && d212.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d5.Reg, int32(d212.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d212.Imm.Int()))
							ctx.EmitCmpInt64(d5.Reg, RegR11)
						}
						ctx.EmitSetcc(r22, CondSignedLess)
						d213 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
						ctx.BindReg(r22, &d213)
					} else if d5.Loc == LocImm {
						r23 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d212.Reg)
						ctx.EmitSetcc(r23, CondSignedLess)
						d213 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
						ctx.BindReg(r23, &d213)
					} else {
						r24 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpInt64(d5.Reg, d212.Reg)
						ctx.EmitSetcc(r24, CondSignedLess)
						d213 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
						ctx.BindReg(r24, &d213)
					}
					ctx.FreeDesc(&d212)
					d214 = d213
					ctx.EnsureDesc(&d214)
					if d214.Loc != LocImm && d214.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d214.Loc == LocImm {
						if d214.Imm.Bool() {
							if ps.General {
							}
							ps215 := PhiState{General: ps.General}
							ps215.OverlayValues = make([]JITValueDesc, 215)
							ps215.OverlayValues[4] = d4
							ps215.OverlayValues[5] = d5
							ps215.OverlayValues[6] = d6
							ps215.OverlayValues[7] = d7
							ps215.OverlayValues[8] = d8
							ps215.OverlayValues[9] = d9
							ps215.OverlayValues[10] = d10
							ps215.OverlayValues[11] = d11
							ps215.OverlayValues[12] = d12
							ps215.OverlayValues[13] = d13
							ps215.OverlayValues[14] = d14
							ps215.OverlayValues[31] = d31
							ps215.OverlayValues[32] = d32
							ps215.OverlayValues[33] = d33
							ps215.OverlayValues[34] = d34
							ps215.OverlayValues[35] = d35
							ps215.OverlayValues[36] = d36
							ps215.OverlayValues[37] = d37
							ps215.OverlayValues[38] = d38
							ps215.OverlayValues[39] = d39
							ps215.OverlayValues[40] = d40
							ps215.OverlayValues[41] = d41
							ps215.OverlayValues[42] = d42
							ps215.OverlayValues[43] = d43
							ps215.OverlayValues[74] = d74
							ps215.OverlayValues[75] = d75
							ps215.OverlayValues[76] = d76
							ps215.OverlayValues[77] = d77
							ps215.OverlayValues[78] = d78
							ps215.OverlayValues[113] = d113
							ps215.OverlayValues[114] = d114
							ps215.OverlayValues[115] = d115
							ps215.OverlayValues[153] = d153
							ps215.OverlayValues[154] = d154
							ps215.OverlayValues[155] = d155
							ps215.OverlayValues[158] = d158
							ps215.OverlayValues[198] = d198
							ps215.OverlayValues[199] = d199
							ps215.OverlayValues[200] = d200
							ps215.OverlayValues[201] = d201
							ps215.OverlayValues[202] = d202
							ps215.OverlayValues[204] = d204
							ps215.OverlayValues[205] = d205
							ps215.OverlayValues[206] = d206
							ps215.OverlayValues[207] = d207
							ps215.OverlayValues[209] = d209
							ps215.OverlayValues[210] = d210
							ps215.OverlayValues[211] = d211
							ps215.OverlayValues[212] = d212
							ps215.OverlayValues[213] = d213
							ps215.OverlayValues[214] = d214
							return bbs[11].RenderPS(ps215)
						}
						if ps.General {
						}
						ps216 := PhiState{General: ps.General}
						ps216.OverlayValues = make([]JITValueDesc, 215)
						ps216.OverlayValues[4] = d4
						ps216.OverlayValues[5] = d5
						ps216.OverlayValues[6] = d6
						ps216.OverlayValues[7] = d7
						ps216.OverlayValues[8] = d8
						ps216.OverlayValues[9] = d9
						ps216.OverlayValues[10] = d10
						ps216.OverlayValues[11] = d11
						ps216.OverlayValues[12] = d12
						ps216.OverlayValues[13] = d13
						ps216.OverlayValues[14] = d14
						ps216.OverlayValues[31] = d31
						ps216.OverlayValues[32] = d32
						ps216.OverlayValues[33] = d33
						ps216.OverlayValues[34] = d34
						ps216.OverlayValues[35] = d35
						ps216.OverlayValues[36] = d36
						ps216.OverlayValues[37] = d37
						ps216.OverlayValues[38] = d38
						ps216.OverlayValues[39] = d39
						ps216.OverlayValues[40] = d40
						ps216.OverlayValues[41] = d41
						ps216.OverlayValues[42] = d42
						ps216.OverlayValues[43] = d43
						ps216.OverlayValues[74] = d74
						ps216.OverlayValues[75] = d75
						ps216.OverlayValues[76] = d76
						ps216.OverlayValues[77] = d77
						ps216.OverlayValues[78] = d78
						ps216.OverlayValues[113] = d113
						ps216.OverlayValues[114] = d114
						ps216.OverlayValues[115] = d115
						ps216.OverlayValues[153] = d153
						ps216.OverlayValues[154] = d154
						ps216.OverlayValues[155] = d155
						ps216.OverlayValues[158] = d158
						ps216.OverlayValues[198] = d198
						ps216.OverlayValues[199] = d199
						ps216.OverlayValues[200] = d200
						ps216.OverlayValues[201] = d201
						ps216.OverlayValues[202] = d202
						ps216.OverlayValues[204] = d204
						ps216.OverlayValues[205] = d205
						ps216.OverlayValues[206] = d206
						ps216.OverlayValues[207] = d207
						ps216.OverlayValues[209] = d209
						ps216.OverlayValues[210] = d210
						ps216.OverlayValues[211] = d211
						ps216.OverlayValues[212] = d212
						ps216.OverlayValues[213] = d213
						ps216.OverlayValues[214] = d214
						return bbs[12].RenderPS(ps216)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d217 := ps.PhiValues[0]
							if phiHomeOK3 {
								ctx.EmitMovToReg(r1, d217)
							} else {
								ctx.EmitStoreToStack(d217, int32(bbs[10].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[10].RenderPS(ps)
					}
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d214.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl24)
					ctx.EmitJmp(lbl25)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl13)
					ps218 := PhiState{General: true}
					ps218.OverlayValues = make([]JITValueDesc, 218)
					ps218.OverlayValues[4] = d4
					ps218.OverlayValues[5] = d5
					ps218.OverlayValues[6] = d6
					ps218.OverlayValues[7] = d7
					ps218.OverlayValues[8] = d8
					ps218.OverlayValues[9] = d9
					ps218.OverlayValues[10] = d10
					ps218.OverlayValues[11] = d11
					ps218.OverlayValues[12] = d12
					ps218.OverlayValues[13] = d13
					ps218.OverlayValues[14] = d14
					ps218.OverlayValues[31] = d31
					ps218.OverlayValues[32] = d32
					ps218.OverlayValues[33] = d33
					ps218.OverlayValues[34] = d34
					ps218.OverlayValues[35] = d35
					ps218.OverlayValues[36] = d36
					ps218.OverlayValues[37] = d37
					ps218.OverlayValues[38] = d38
					ps218.OverlayValues[39] = d39
					ps218.OverlayValues[40] = d40
					ps218.OverlayValues[41] = d41
					ps218.OverlayValues[42] = d42
					ps218.OverlayValues[43] = d43
					ps218.OverlayValues[74] = d74
					ps218.OverlayValues[75] = d75
					ps218.OverlayValues[76] = d76
					ps218.OverlayValues[77] = d77
					ps218.OverlayValues[78] = d78
					ps218.OverlayValues[113] = d113
					ps218.OverlayValues[114] = d114
					ps218.OverlayValues[115] = d115
					ps218.OverlayValues[153] = d153
					ps218.OverlayValues[154] = d154
					ps218.OverlayValues[155] = d155
					ps218.OverlayValues[158] = d158
					ps218.OverlayValues[198] = d198
					ps218.OverlayValues[199] = d199
					ps218.OverlayValues[200] = d200
					ps218.OverlayValues[201] = d201
					ps218.OverlayValues[202] = d202
					ps218.OverlayValues[204] = d204
					ps218.OverlayValues[205] = d205
					ps218.OverlayValues[206] = d206
					ps218.OverlayValues[207] = d207
					ps218.OverlayValues[209] = d209
					ps218.OverlayValues[210] = d210
					ps218.OverlayValues[211] = d211
					ps218.OverlayValues[212] = d212
					ps218.OverlayValues[213] = d213
					ps218.OverlayValues[214] = d214
					ps218.OverlayValues[217] = d217
					ps219 := PhiState{General: true}
					ps219.OverlayValues = make([]JITValueDesc, 218)
					ps219.OverlayValues[4] = d4
					ps219.OverlayValues[5] = d5
					ps219.OverlayValues[6] = d6
					ps219.OverlayValues[7] = d7
					ps219.OverlayValues[8] = d8
					ps219.OverlayValues[9] = d9
					ps219.OverlayValues[10] = d10
					ps219.OverlayValues[11] = d11
					ps219.OverlayValues[12] = d12
					ps219.OverlayValues[13] = d13
					ps219.OverlayValues[14] = d14
					ps219.OverlayValues[31] = d31
					ps219.OverlayValues[32] = d32
					ps219.OverlayValues[33] = d33
					ps219.OverlayValues[34] = d34
					ps219.OverlayValues[35] = d35
					ps219.OverlayValues[36] = d36
					ps219.OverlayValues[37] = d37
					ps219.OverlayValues[38] = d38
					ps219.OverlayValues[39] = d39
					ps219.OverlayValues[40] = d40
					ps219.OverlayValues[41] = d41
					ps219.OverlayValues[42] = d42
					ps219.OverlayValues[43] = d43
					ps219.OverlayValues[74] = d74
					ps219.OverlayValues[75] = d75
					ps219.OverlayValues[76] = d76
					ps219.OverlayValues[77] = d77
					ps219.OverlayValues[78] = d78
					ps219.OverlayValues[113] = d113
					ps219.OverlayValues[114] = d114
					ps219.OverlayValues[115] = d115
					ps219.OverlayValues[153] = d153
					ps219.OverlayValues[154] = d154
					ps219.OverlayValues[155] = d155
					ps219.OverlayValues[158] = d158
					ps219.OverlayValues[198] = d198
					ps219.OverlayValues[199] = d199
					ps219.OverlayValues[200] = d200
					ps219.OverlayValues[201] = d201
					ps219.OverlayValues[202] = d202
					ps219.OverlayValues[204] = d204
					ps219.OverlayValues[205] = d205
					ps219.OverlayValues[206] = d206
					ps219.OverlayValues[207] = d207
					ps219.OverlayValues[209] = d209
					ps219.OverlayValues[210] = d210
					ps219.OverlayValues[211] = d211
					ps219.OverlayValues[212] = d212
					ps219.OverlayValues[213] = d213
					ps219.OverlayValues[214] = d214
					ps219.OverlayValues[217] = d217
					snap220 := d4
					snap221 := d5
					snap222 := d6
					snap223 := d7
					snap224 := d8
					snap225 := d9
					snap226 := d10
					snap227 := d11
					snap228 := d12
					snap229 := d13
					snap230 := d14
					snap231 := d31
					snap232 := d32
					snap233 := d33
					snap234 := d34
					snap235 := d35
					snap236 := d36
					snap237 := d37
					snap238 := d38
					snap239 := d39
					snap240 := d40
					snap241 := d41
					snap242 := d42
					snap243 := d43
					snap244 := d74
					snap245 := d75
					snap246 := d76
					snap247 := d77
					snap248 := d78
					snap249 := d113
					snap250 := d114
					snap251 := d115
					snap252 := d153
					snap253 := d154
					snap254 := d155
					snap255 := d158
					snap256 := d198
					snap257 := d199
					snap258 := d200
					snap259 := d201
					snap260 := d202
					snap261 := d204
					snap262 := d205
					snap263 := d206
					snap264 := d207
					snap265 := d209
					snap266 := d210
					snap267 := d211
					snap268 := d212
					snap269 := d213
					snap270 := d214
					snap271 := d217
					alloc272 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps219)
					}
					ctx.RestoreAllocState(alloc272)
					d4 = snap220
					d5 = snap221
					d6 = snap222
					d7 = snap223
					d8 = snap224
					d9 = snap225
					d10 = snap226
					d11 = snap227
					d12 = snap228
					d13 = snap229
					d14 = snap230
					d31 = snap231
					d32 = snap232
					d33 = snap233
					d34 = snap234
					d35 = snap235
					d36 = snap236
					d37 = snap237
					d38 = snap238
					d39 = snap239
					d40 = snap240
					d41 = snap241
					d42 = snap242
					d43 = snap243
					d74 = snap244
					d75 = snap245
					d76 = snap246
					d77 = snap247
					d78 = snap248
					d113 = snap249
					d114 = snap250
					d115 = snap251
					d153 = snap252
					d154 = snap253
					d155 = snap254
					d158 = snap255
					d198 = snap256
					d199 = snap257
					d200 = snap258
					d201 = snap259
					d202 = snap260
					d204 = snap261
					d205 = snap262
					d206 = snap263
					d207 = snap264
					d209 = snap265
					d210 = snap266
					d211 = snap267
					d212 = snap268
					d213 = snap269
					d214 = snap270
					d217 = snap271
					if !bbs[11].Rendered {
						return bbs[11].RenderPS(ps218)
					}
					return result
					ctx.FreeDesc(&d213)
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d4)
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d5)
					} else {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
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
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					ctx.ReclaimUntrackedRegs()
					d273 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d5)
					ctx.SyncDesc(&d273)
					ctx.StabilizeDescAcrossNestedCall(&d5)
					d274 = d41
					d274.ID = 0
					d275 = d5
					d275.ID = 0
					d276 = ctx.EmitSliceElementAddress(&d274, &d275, int32(16))
					ctx.FreeDesc(&d275)
					ctx.EmitStoreScmerAt(&d276, &d273)
					ctx.FreeDesc(&d276)
					ctx.FreeDesc(&d273)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d5)
					var d277 JITValueDesc
					if d5.Loc == LocImm {
						d277 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d5.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK3 {
							scratch = r1
						} else {
							scratch = ctx.AllocRegExcept(d5.Reg)
						}
						ctx.EmitMovRegReg(scratch, d5.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d277 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d277)
					}
					if d277.Loc == LocReg && d5.Loc == LocReg && d277.Reg == d5.Reg {
						ctx.TransferReg(d5.Reg)
						d5.Loc = LocNone
					}
					ctx.FreeDesc(&d5)
					if ps.General {
						ctx.SyncDesc(&d277)
						if d277.Loc == LocReg {
							ctx.ProtectReg(d277.Reg)
						} else if d277.Loc == LocRegPair {
							ctx.ProtectReg(d277.Reg)
							ctx.ProtectReg(d277.Reg2)
						}
						d278 = d277
						if d278.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d278)
						if phiHomeOK3 {
							ctx.EmitMovToReg(r1, d278)
						} else {
							ctx.EmitStoreToStack(d278, int32(bbs[10].PhiBase)+int32(0))
						}
						if d277.Loc == LocReg {
							ctx.UnprotectReg(d277.Reg)
						} else if d277.Loc == LocRegPair {
							ctx.UnprotectReg(d277.Reg)
							ctx.UnprotectReg(d277.Reg2)
						}
					}
					ps279 := PhiState{General: ps.General}
					ps279.OverlayValues = make([]JITValueDesc, 279)
					ps279.OverlayValues[4] = d4
					ps279.OverlayValues[5] = d5
					ps279.OverlayValues[6] = d6
					ps279.OverlayValues[7] = d7
					ps279.OverlayValues[8] = d8
					ps279.OverlayValues[9] = d9
					ps279.OverlayValues[10] = d10
					ps279.OverlayValues[11] = d11
					ps279.OverlayValues[12] = d12
					ps279.OverlayValues[13] = d13
					ps279.OverlayValues[14] = d14
					ps279.OverlayValues[31] = d31
					ps279.OverlayValues[32] = d32
					ps279.OverlayValues[33] = d33
					ps279.OverlayValues[34] = d34
					ps279.OverlayValues[35] = d35
					ps279.OverlayValues[36] = d36
					ps279.OverlayValues[37] = d37
					ps279.OverlayValues[38] = d38
					ps279.OverlayValues[39] = d39
					ps279.OverlayValues[40] = d40
					ps279.OverlayValues[41] = d41
					ps279.OverlayValues[42] = d42
					ps279.OverlayValues[43] = d43
					ps279.OverlayValues[74] = d74
					ps279.OverlayValues[75] = d75
					ps279.OverlayValues[76] = d76
					ps279.OverlayValues[77] = d77
					ps279.OverlayValues[78] = d78
					ps279.OverlayValues[113] = d113
					ps279.OverlayValues[114] = d114
					ps279.OverlayValues[115] = d115
					ps279.OverlayValues[153] = d153
					ps279.OverlayValues[154] = d154
					ps279.OverlayValues[155] = d155
					ps279.OverlayValues[158] = d158
					ps279.OverlayValues[198] = d198
					ps279.OverlayValues[199] = d199
					ps279.OverlayValues[200] = d200
					ps279.OverlayValues[201] = d201
					ps279.OverlayValues[202] = d202
					ps279.OverlayValues[204] = d204
					ps279.OverlayValues[205] = d205
					ps279.OverlayValues[206] = d206
					ps279.OverlayValues[207] = d207
					ps279.OverlayValues[209] = d209
					ps279.OverlayValues[210] = d210
					ps279.OverlayValues[211] = d211
					ps279.OverlayValues[212] = d212
					ps279.OverlayValues[213] = d213
					ps279.OverlayValues[214] = d214
					ps279.OverlayValues[217] = d217
					ps279.OverlayValues[273] = d273
					ps279.OverlayValues[274] = d274
					ps279.OverlayValues[275] = d275
					ps279.OverlayValues[276] = d276
					ps279.OverlayValues[277] = d277
					ps279.OverlayValues[278] = d278
					ps279.PhiValues = make([]JITValueDesc, 1)
					d280 = d277
					ps279.PhiValues[0] = d280
					if ps279.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps279)
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d4)
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d5)
					} else {
						d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
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
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != LocNone {
						d217 = ps.OverlayValues[217]
					}
					if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
						d273 = ps.OverlayValues[273]
					}
					if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
						d274 = ps.OverlayValues[274]
					}
					if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
						d275 = ps.OverlayValues[275]
					}
					if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
						d276 = ps.OverlayValues[276]
					}
					if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != LocNone {
						d277 = ps.OverlayValues[277]
					}
					if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != LocNone {
						d278 = ps.OverlayValues[278]
					}
					if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != LocNone {
						d280 = ps.OverlayValues[280]
					}
					ctx.ReclaimUntrackedRegs()
					d281 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					d283 = ctx.EmitSliceElementAddress(&d7, &d281, 16)
					ctx.EnsureDesc(&d283)
					r25 := ctx.AllocRegExcept(d283.Reg)
					ctx.EmitMovRegMem(r25, d283.Reg, 8)
					ctx.EmitMovRegMem(d283.Reg, d283.Reg, 0)
					d282 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d283.Reg, Reg2: r25}
					ctx.BindReg(d283.Reg, &d282)
					ctx.BindReg(r25, &d282)
					var d284 JITValueDesc
					if d282.Loc == LocImm {
						d284 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d282.Imm.Int())}
					} else if d282.Type == tagInt && d282.Loc == LocRegPair {
						ctx.FreeReg(d282.Reg)
						d284 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d282.Reg2}
						ctx.BindReg(d282.Reg2, &d284)
						ctx.BindReg(d282.Reg2, &d284)
					} else if d282.Type == tagInt && d282.Loc == LocReg {
						d284 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d282.Reg}
						ctx.BindReg(d282.Reg, &d284)
						ctx.BindReg(d282.Reg, &d284)
					} else {
						d284 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d282}, 1)
						d284.Type = tagInt
						ctx.BindReg(d284.Reg, &d284)
					}
					ctx.FreeDesc(&d282)
					ctx.EnsureDesc(&d284)
					ctx.EnsureDesc(&d284)
					var d285 JITValueDesc
					if d284.Loc == LocImm {
						d285 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d284.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d284.Reg)
						ctx.EmitMovRegReg(scratch, d284.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d285 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d285)
					}
					if d285.Loc == LocReg && d284.Loc == LocReg && d285.Reg == d284.Reg {
						ctx.TransferReg(d284.Reg)
						d284.Loc = LocNone
					}
					ctx.FreeDesc(&d284)
					ctx.EnsureDesc(&d285)
					d286 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.SyncDesc(&d285)
					d287 = d7
					d287.ID = 0
					d288 = d286
					d288.ID = 0
					d289 = ctx.EmitSliceElementAddress(&d287, &d288, int32(16))
					ctx.FreeDesc(&d288)
					ctx.EmitStoreScmerAt(&d289, &d285)
					ctx.FreeDesc(&d289)
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d41)
					d290 = d8
					_ = d290
					ctx.StabilizeDescForControlFlow(&d290)
					d291 = d41
					_ = d291
					ctx.StabilizeDescForControlFlow(&d291)
					ctx.StabilizeDescForControlFlow(&d41)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl26 := ctx.ReserveLabel()
					_ = lbl26
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl26)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d290)
					ctx.EnsureDesc(&d290)
					d290 = JITPrepareScmerGoArg(ctx, d290)
					ctx.EnsureDesc(&d291)
					ctx.EnsureDesc(&d291)
					ctx.EnsureDesc(&d291)
					if d291.Loc != LocRegTriple && d291.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d292 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d292.Loc == LocRegPair || d292.Loc == LocStackPair || d292.Loc == LocRegTriple || d292.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d290)
					ctx.SyncDesc(&d291)
					ctx.SyncDesc(&d292)
					d293 = ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d290, d291, d292}, 2)
					d293.NoHeapPointer = false
					ctx.BindReg(d293.Reg, &d293)
					ctx.BindReg(d293.Reg2, &d293)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d293)
					ctx.FreeDesc(&d8)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					var d294 JITValueDesc
					if d4.Loc == LocImm {
						d294 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK2 {
							scratch = r0
						} else {
							scratch = ctx.AllocRegExcept(d4.Reg)
						}
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d294 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d294)
					}
					if d294.Loc == LocReg && d4.Loc == LocReg && d294.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.FreeDesc(&d4)
					if ps.General {
						ctx.SyncDesc(&d294)
						if d294.Loc == LocReg {
							ctx.ProtectReg(d294.Reg)
						} else if d294.Loc == LocRegPair {
							ctx.ProtectReg(d294.Reg)
							ctx.ProtectReg(d294.Reg2)
						}
						d295 = d294
						if d295.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d295)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d295)
						} else {
							ctx.EmitStoreToStack(d295, int32(bbs[7].PhiBase)+int32(0))
						}
						if d294.Loc == LocReg {
							ctx.UnprotectReg(d294.Reg)
						} else if d294.Loc == LocRegPair {
							ctx.UnprotectReg(d294.Reg)
							ctx.UnprotectReg(d294.Reg2)
						}
					}
					ps296 := PhiState{General: ps.General}
					ps296.OverlayValues = make([]JITValueDesc, 296)
					ps296.OverlayValues[4] = d4
					ps296.OverlayValues[5] = d5
					ps296.OverlayValues[6] = d6
					ps296.OverlayValues[7] = d7
					ps296.OverlayValues[8] = d8
					ps296.OverlayValues[9] = d9
					ps296.OverlayValues[10] = d10
					ps296.OverlayValues[11] = d11
					ps296.OverlayValues[12] = d12
					ps296.OverlayValues[13] = d13
					ps296.OverlayValues[14] = d14
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
					ps296.OverlayValues[74] = d74
					ps296.OverlayValues[75] = d75
					ps296.OverlayValues[76] = d76
					ps296.OverlayValues[77] = d77
					ps296.OverlayValues[78] = d78
					ps296.OverlayValues[113] = d113
					ps296.OverlayValues[114] = d114
					ps296.OverlayValues[115] = d115
					ps296.OverlayValues[153] = d153
					ps296.OverlayValues[154] = d154
					ps296.OverlayValues[155] = d155
					ps296.OverlayValues[158] = d158
					ps296.OverlayValues[198] = d198
					ps296.OverlayValues[199] = d199
					ps296.OverlayValues[200] = d200
					ps296.OverlayValues[201] = d201
					ps296.OverlayValues[202] = d202
					ps296.OverlayValues[204] = d204
					ps296.OverlayValues[205] = d205
					ps296.OverlayValues[206] = d206
					ps296.OverlayValues[207] = d207
					ps296.OverlayValues[209] = d209
					ps296.OverlayValues[210] = d210
					ps296.OverlayValues[211] = d211
					ps296.OverlayValues[212] = d212
					ps296.OverlayValues[213] = d213
					ps296.OverlayValues[214] = d214
					ps296.OverlayValues[217] = d217
					ps296.OverlayValues[273] = d273
					ps296.OverlayValues[274] = d274
					ps296.OverlayValues[275] = d275
					ps296.OverlayValues[276] = d276
					ps296.OverlayValues[277] = d277
					ps296.OverlayValues[278] = d278
					ps296.OverlayValues[280] = d280
					ps296.OverlayValues[281] = d281
					ps296.OverlayValues[282] = d282
					ps296.OverlayValues[283] = d283
					ps296.OverlayValues[284] = d284
					ps296.OverlayValues[285] = d285
					ps296.OverlayValues[286] = d286
					ps296.OverlayValues[287] = d287
					ps296.OverlayValues[288] = d288
					ps296.OverlayValues[289] = d289
					ps296.OverlayValues[290] = d290
					ps296.OverlayValues[291] = d291
					ps296.OverlayValues[292] = d292
					ps296.OverlayValues[293] = d293
					ps296.OverlayValues[294] = d294
					ps296.OverlayValues[295] = d295
					ps296.PhiValues = make([]JITValueDesc, 1)
					d297 = d294
					ps296.PhiValues[0] = d297
					if ps296.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps296)
					return result
				}
				ps298 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps298)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  62,
		},
	})
}
