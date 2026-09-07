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
				d5 := d3
				_ = d5
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				lbl0 := ctx.ReserveLabel()
				_ = lbl0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d4 = JITPrepareScmerGoArg(ctx, d4)
				d5 = JITPrepareGoSliceArg(ctx, d5)
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
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d127 JITValueDesc
				_ = d127
				var d128 JITValueDesc
				_ = d128
				var d129 JITValueDesc
				_ = d129
				var d130 JITValueDesc
				_ = d130
				var d131 JITValueDesc
				_ = d131
				var d133 JITValueDesc
				_ = d133
				var d134 JITValueDesc
				_ = d134
				var d135 JITValueDesc
				_ = d135
				var d136 JITValueDesc
				_ = d136
				var d137 JITValueDesc
				_ = d137
				var d138 JITValueDesc
				_ = d138
				var d139 JITValueDesc
				_ = d139
				var d140 JITValueDesc
				_ = d140
				var d141 JITValueDesc
				_ = d141
				var d143 JITValueDesc
				_ = d143
				var d144 JITValueDesc
				_ = d144
				var d145 JITValueDesc
				_ = d145
				var d146 JITValueDesc
				_ = d146
				var d147 JITValueDesc
				_ = d147
				var d256 JITValueDesc
				_ = d256
				var d257 JITValueDesc
				_ = d257
				var d258 JITValueDesc
				_ = d258
				var d373 JITValueDesc
				_ = d373
				var d374 JITValueDesc
				_ = d374
				var d375 JITValueDesc
				_ = d375
				var d376 JITValueDesc
				_ = d376
				var d379 JITValueDesc
				_ = d379
				var d502 JITValueDesc
				_ = d502
				var d503 JITValueDesc
				_ = d503
				var d504 JITValueDesc
				_ = d504
				var d635 JITValueDesc
				_ = d635
				var d636 JITValueDesc
				_ = d636
				var d637 JITValueDesc
				_ = d637
				var d638 JITValueDesc
				_ = d638
				var d639 JITValueDesc
				_ = d639
				var d640 JITValueDesc
				_ = d640
				var d641 JITValueDesc
				_ = d641
				var d786 JITValueDesc
				_ = d786
				var d787 JITValueDesc
				_ = d787
				var d788 JITValueDesc
				_ = d788
				var d789 JITValueDesc
				_ = d789
				var d790 JITValueDesc
				_ = d790
				var d792 JITValueDesc
				_ = d792
				var d793 JITValueDesc
				_ = d793
				var d794 JITValueDesc
				_ = d794
				var d795 JITValueDesc
				_ = d795
				var d796 JITValueDesc
				_ = d796
				var d798 JITValueDesc
				_ = d798
				var d799 JITValueDesc
				_ = d799
				var d800 JITValueDesc
				_ = d800
				var d801 JITValueDesc
				_ = d801
				var d802 JITValueDesc
				_ = d802
				var d803 JITValueDesc
				_ = d803
				var d804 JITValueDesc
				_ = d804
				var d805 JITValueDesc
				_ = d805
				var d806 JITValueDesc
				_ = d806
				var d807 JITValueDesc
				_ = d807
				var d808 JITValueDesc
				_ = d808
				var d809 JITValueDesc
				_ = d809
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [14]BBDescriptor
				bbs[7].PhiBase = int32(phiBase0) + int32(0)
				bbs[7].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
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
						d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d8)
					}
					ctx.FreeDesc(&d7)
					d9 = d8
					ctx.EnsureDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
					ctx.EmitJump(d9.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d8)
					snap12 := d1
					snap13 := d2
					snap14 := d3
					snap15 := d4
					snap16 := d5
					snap17 := d6
					snap18 := d7
					snap19 := d8
					snap20 := d9
					alloc21 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc21)
					d1 = snap12
					d2 = snap13
					d3 = snap14
					d4 = snap15
					d5 = snap16
					d6 = snap17
					d7 = snap18
					d8 = snap19
					d9 = snap20
					ctx.RestoreAllocState(alloc21)
					d1 = snap12
					d2 = snap13
					d3 = snap14
					d4 = snap15
					d5 = snap16
					d6 = snap17
					d7 = snap18
					d8 = snap19
					d9 = snap20
					ps22 := PhiState{General: true}
					ps22.OverlayValues = make([]JITValueDesc, 10)
					ps22.OverlayValues[1] = d1
					ps22.OverlayValues[2] = d2
					ps22.OverlayValues[3] = d3
					ps22.OverlayValues[4] = d4
					ps22.OverlayValues[5] = d5
					ps22.OverlayValues[6] = d6
					ps22.OverlayValues[7] = d7
					ps22.OverlayValues[8] = d8
					ps22.OverlayValues[9] = d9
					ps23 := PhiState{General: true}
					ps23.OverlayValues = make([]JITValueDesc, 10)
					ps23.OverlayValues[1] = d1
					ps23.OverlayValues[2] = d2
					ps23.OverlayValues[3] = d3
					ps23.OverlayValues[4] = d4
					ps23.OverlayValues[5] = d5
					ps23.OverlayValues[6] = d6
					ps23.OverlayValues[7] = d7
					ps23.OverlayValues[8] = d8
					ps23.OverlayValues[9] = d9
					snap24 := d1
					snap25 := d2
					snap26 := d3
					snap27 := d4
					snap28 := d5
					snap29 := d6
					snap30 := d7
					snap31 := d8
					snap32 := d9
					alloc33 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps23)
					}
					ctx.RestoreAllocState(alloc33)
					d1 = snap24
					d2 = snap25
					d3 = snap26
					d4 = snap27
					d5 = snap28
					d6 = snap29
					d7 = snap30
					d8 = snap31
					d9 = snap32
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps22)
					}
					return result
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
					d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d36 = ctx.EmitSliceElementAddress(&d3, &d34, 16)
					ctx.EnsureDesc(&d36)
					r1 := ctx.AllocRegExcept(d36.Reg)
					ctx.EmitMovRegMem(r1, d36.Reg, 8)
					ctx.EmitMovRegMem(d36.Reg, d36.Reg, 0)
					d35 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d36.Reg, Reg2: r1}
					ctx.BindReg(d36.Reg, &d35)
					ctx.BindReg(r1, &d35)
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
					d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					d41 = ctx.EmitSliceElementAddress(&d3, &d39, 16)
					ctx.EnsureDesc(&d41)
					r2 := ctx.AllocRegExcept(d41.Reg)
					ctx.EmitMovRegMem(r2, d41.Reg, 8)
					ctx.EmitMovRegMem(d41.Reg, d41.Reg, 0)
					d40 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d41.Reg, Reg2: r2}
					ctx.BindReg(d41.Reg, &d40)
					ctx.BindReg(r2, &d40)
					var d42 JITValueDesc
					if d40.Loc == LocImm {
						d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d40.Imm.Int())}
					} else if d40.Type == tagInt && d40.Loc == LocRegPair {
						ctx.FreeReg(d40.Reg)
						d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d40.Reg2}
						ctx.BindReg(d40.Reg2, &d42)
						ctx.BindReg(d40.Reg2, &d42)
					} else if d40.Type == tagInt && d40.Loc == LocReg {
						d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d40.Reg}
						ctx.BindReg(d40.Reg, &d42)
						ctx.BindReg(d40.Reg, &d42)
					} else {
						d42 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d40}, 1)
						d42.Type = tagInt
						ctx.BindReg(d42.Reg, &d42)
					}
					ctx.FreeDesc(&d40)
					ctx.EnsureDesc(&d42)
					ctx.EnsureDesc(&d42)
					ctx.StabilizeDescForControlFlow(&d42)
					d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					d46 = ctx.EmitSliceElementAddress(&d3, &d44, 16)
					ctx.EnsureDesc(&d46)
					r3 := ctx.AllocRegExcept(d46.Reg)
					ctx.EmitMovRegMem(r3, d46.Reg, 8)
					ctx.EmitMovRegMem(d46.Reg, d46.Reg, 0)
					d45 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d46.Reg, Reg2: r3}
					ctx.BindReg(d46.Reg, &d45)
					ctx.BindReg(r3, &d45)
					var d47 JITValueDesc
					if d45.Loc == LocImm {
						d47 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d45.Imm.Int())}
					} else if d45.Type == tagInt && d45.Loc == LocRegPair {
						ctx.FreeReg(d45.Reg)
						d47 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d45.Reg2}
						ctx.BindReg(d45.Reg2, &d47)
						ctx.BindReg(d45.Reg2, &d47)
					} else if d45.Type == tagInt && d45.Loc == LocReg {
						d47 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d45.Reg}
						ctx.BindReg(d45.Reg, &d47)
						ctx.BindReg(d45.Reg, &d47)
					} else {
						d47 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d45}, 1)
						d47.Type = tagInt
						ctx.BindReg(d47.Reg, &d47)
					}
					ctx.FreeDesc(&d45)
					ctx.EnsureDesc(&d47)
					ctx.EnsureDesc(&d47)
					ctx.StabilizeDescForControlFlow(&d47)
					d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(3)}
					var d50 JITValueDesc
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2}
						ctx.BindReg(d3.Reg2, &d50)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d49)
					ctx.EnsureDesc(&d50)
					var d52 JITValueDesc
					if d50.Loc == LocImm && d49.Loc == LocImm {
						d52 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d50.Imm.Int() - d49.Imm.Int())}
					} else {
						r4 := ctx.AllocReg()
						if d50.Loc == LocImm {
							ctx.EmitMovRegImm64(r4, uint64(d50.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r4, d50.Reg)
						}
						if d49.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d49.Imm.Int()))
							ctx.EmitSubInt64(r4, RegR11)
						} else {
							ctx.EmitSubInt64(r4, d49.Reg)
						}
						d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d52)
					}
					var d53 JITValueDesc
					r5 := ctx.EmitSliceDataAfterLow(&d3, &d49, 16)
					d53 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
					ctx.BindReg(r5, &d53)
					ctx.BindReg(r5, &d53)
					var d54 JITValueDesc
					var r6 Reg
					var r7 Reg
					ctx.SyncDesc(&d53)
					ctx.EnsureDesc(&d53)
					if d53.Loc == LocImm {
						r6 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, uint64(d53.Imm.Int()))
					} else {
						r6 = d53.Reg
					}
					ctx.ProtectReg(r6)
					ctx.SyncDesc(&d52)
					ctx.EnsureDesc(&d52)
					if d52.Loc == LocImm {
						r7 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, uint64(d52.Imm.Int()))
					} else {
						r7 = d52.Reg
					}
					ctx.ProtectReg(r7)
					r8 := ctx.EmitSliceCapAfterLow(&d3, &d49, r6, r7)
					ctx.UnprotectReg(r7)
					ctx.UnprotectReg(r6)
					d54 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8}
					ctx.BindReg(r6, &d54)
					ctx.BindReg(r7, &d54)
					ctx.BindReg(r8, &d54)
					ctx.BindReg(r6, &d54)
					ctx.BindReg(r7, &d54)
					ctx.BindReg(r8, &d54)
					ctx.StabilizeDescForControlFlow(&d54)
					ctx.EnsureDesc(&d47)
					var d55 JITValueDesc
					if d47.Loc == LocImm {
						d55 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d47.Imm.Int() <= 0)}
					} else {
						r9 := ctx.AllocRegExcept(d47.Reg)
						ctx.EmitCmpRegImm32(d47.Reg, 0)
						d55 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondSignedLessOrEqual}
						ctx.BindReg(r9, &d55)
					}
					d56 = d55
					ctx.EnsureDesc(&d56)
					if d56.Loc != LocImm && d56.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d56.Loc == LocImm {
						if d56.Imm.Bool() {
							if ps.General {
							}
							ps57 := PhiState{General: ps.General}
							ps57.OverlayValues = make([]JITValueDesc, 57)
							ps57.OverlayValues[1] = d1
							ps57.OverlayValues[2] = d2
							ps57.OverlayValues[3] = d3
							ps57.OverlayValues[4] = d4
							ps57.OverlayValues[5] = d5
							ps57.OverlayValues[6] = d6
							ps57.OverlayValues[7] = d7
							ps57.OverlayValues[8] = d8
							ps57.OverlayValues[9] = d9
							ps57.OverlayValues[34] = d34
							ps57.OverlayValues[35] = d35
							ps57.OverlayValues[36] = d36
							ps57.OverlayValues[37] = d37
							ps57.OverlayValues[38] = d38
							ps57.OverlayValues[39] = d39
							ps57.OverlayValues[40] = d40
							ps57.OverlayValues[41] = d41
							ps57.OverlayValues[42] = d42
							ps57.OverlayValues[43] = d43
							ps57.OverlayValues[44] = d44
							ps57.OverlayValues[45] = d45
							ps57.OverlayValues[46] = d46
							ps57.OverlayValues[47] = d47
							ps57.OverlayValues[48] = d48
							ps57.OverlayValues[49] = d49
							ps57.OverlayValues[50] = d50
							ps57.OverlayValues[51] = d51
							ps57.OverlayValues[52] = d52
							ps57.OverlayValues[53] = d53
							ps57.OverlayValues[54] = d54
							ps57.OverlayValues[55] = d55
							ps57.OverlayValues[56] = d56
							return bbs[3].RenderPS(ps57)
						}
						if ps.General {
						}
						ps58 := PhiState{General: ps.General}
						ps58.OverlayValues = make([]JITValueDesc, 57)
						ps58.OverlayValues[1] = d1
						ps58.OverlayValues[2] = d2
						ps58.OverlayValues[3] = d3
						ps58.OverlayValues[4] = d4
						ps58.OverlayValues[5] = d5
						ps58.OverlayValues[6] = d6
						ps58.OverlayValues[7] = d7
						ps58.OverlayValues[8] = d8
						ps58.OverlayValues[9] = d9
						ps58.OverlayValues[34] = d34
						ps58.OverlayValues[35] = d35
						ps58.OverlayValues[36] = d36
						ps58.OverlayValues[37] = d37
						ps58.OverlayValues[38] = d38
						ps58.OverlayValues[39] = d39
						ps58.OverlayValues[40] = d40
						ps58.OverlayValues[41] = d41
						ps58.OverlayValues[42] = d42
						ps58.OverlayValues[43] = d43
						ps58.OverlayValues[44] = d44
						ps58.OverlayValues[45] = d45
						ps58.OverlayValues[46] = d46
						ps58.OverlayValues[47] = d47
						ps58.OverlayValues[48] = d48
						ps58.OverlayValues[49] = d49
						ps58.OverlayValues[50] = d50
						ps58.OverlayValues[51] = d51
						ps58.OverlayValues[52] = d52
						ps58.OverlayValues[53] = d53
						ps58.OverlayValues[54] = d54
						ps58.OverlayValues[55] = d55
						ps58.OverlayValues[56] = d56
						return bbs[6].RenderPS(ps58)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					ctx.EmitJump(d56.Condition, lbl4)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d55)
					snap59 := d1
					snap60 := d2
					snap61 := d3
					snap62 := d4
					snap63 := d5
					snap64 := d6
					snap65 := d7
					snap66 := d8
					snap67 := d9
					snap68 := d34
					snap69 := d35
					snap70 := d36
					snap71 := d37
					snap72 := d38
					snap73 := d39
					snap74 := d40
					snap75 := d41
					snap76 := d42
					snap77 := d43
					snap78 := d44
					snap79 := d45
					snap80 := d46
					snap81 := d47
					snap82 := d48
					snap83 := d49
					snap84 := d50
					snap85 := d51
					snap86 := d52
					snap87 := d53
					snap88 := d54
					snap89 := d55
					snap90 := d56
					alloc91 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc91)
					d1 = snap59
					d2 = snap60
					d3 = snap61
					d4 = snap62
					d5 = snap63
					d6 = snap64
					d7 = snap65
					d8 = snap66
					d9 = snap67
					d34 = snap68
					d35 = snap69
					d36 = snap70
					d37 = snap71
					d38 = snap72
					d39 = snap73
					d40 = snap74
					d41 = snap75
					d42 = snap76
					d43 = snap77
					d44 = snap78
					d45 = snap79
					d46 = snap80
					d47 = snap81
					d48 = snap82
					d49 = snap83
					d50 = snap84
					d51 = snap85
					d52 = snap86
					d53 = snap87
					d54 = snap88
					d55 = snap89
					d56 = snap90
					ctx.RestoreAllocState(alloc91)
					d1 = snap59
					d2 = snap60
					d3 = snap61
					d4 = snap62
					d5 = snap63
					d6 = snap64
					d7 = snap65
					d8 = snap66
					d9 = snap67
					d34 = snap68
					d35 = snap69
					d36 = snap70
					d37 = snap71
					d38 = snap72
					d39 = snap73
					d40 = snap74
					d41 = snap75
					d42 = snap76
					d43 = snap77
					d44 = snap78
					d45 = snap79
					d46 = snap80
					d47 = snap81
					d48 = snap82
					d49 = snap83
					d50 = snap84
					d51 = snap85
					d52 = snap86
					d53 = snap87
					d54 = snap88
					d55 = snap89
					d56 = snap90
					ps92 := PhiState{General: true}
					ps92.OverlayValues = make([]JITValueDesc, 57)
					ps92.OverlayValues[1] = d1
					ps92.OverlayValues[2] = d2
					ps92.OverlayValues[3] = d3
					ps92.OverlayValues[4] = d4
					ps92.OverlayValues[5] = d5
					ps92.OverlayValues[6] = d6
					ps92.OverlayValues[7] = d7
					ps92.OverlayValues[8] = d8
					ps92.OverlayValues[9] = d9
					ps92.OverlayValues[34] = d34
					ps92.OverlayValues[35] = d35
					ps92.OverlayValues[36] = d36
					ps92.OverlayValues[37] = d37
					ps92.OverlayValues[38] = d38
					ps92.OverlayValues[39] = d39
					ps92.OverlayValues[40] = d40
					ps92.OverlayValues[41] = d41
					ps92.OverlayValues[42] = d42
					ps92.OverlayValues[43] = d43
					ps92.OverlayValues[44] = d44
					ps92.OverlayValues[45] = d45
					ps92.OverlayValues[46] = d46
					ps92.OverlayValues[47] = d47
					ps92.OverlayValues[48] = d48
					ps92.OverlayValues[49] = d49
					ps92.OverlayValues[50] = d50
					ps92.OverlayValues[51] = d51
					ps92.OverlayValues[52] = d52
					ps92.OverlayValues[53] = d53
					ps92.OverlayValues[54] = d54
					ps92.OverlayValues[55] = d55
					ps92.OverlayValues[56] = d56
					ps93 := PhiState{General: true}
					ps93.OverlayValues = make([]JITValueDesc, 57)
					ps93.OverlayValues[1] = d1
					ps93.OverlayValues[2] = d2
					ps93.OverlayValues[3] = d3
					ps93.OverlayValues[4] = d4
					ps93.OverlayValues[5] = d5
					ps93.OverlayValues[6] = d6
					ps93.OverlayValues[7] = d7
					ps93.OverlayValues[8] = d8
					ps93.OverlayValues[9] = d9
					ps93.OverlayValues[34] = d34
					ps93.OverlayValues[35] = d35
					ps93.OverlayValues[36] = d36
					ps93.OverlayValues[37] = d37
					ps93.OverlayValues[38] = d38
					ps93.OverlayValues[39] = d39
					ps93.OverlayValues[40] = d40
					ps93.OverlayValues[41] = d41
					ps93.OverlayValues[42] = d42
					ps93.OverlayValues[43] = d43
					ps93.OverlayValues[44] = d44
					ps93.OverlayValues[45] = d45
					ps93.OverlayValues[46] = d46
					ps93.OverlayValues[47] = d47
					ps93.OverlayValues[48] = d48
					ps93.OverlayValues[49] = d49
					ps93.OverlayValues[50] = d50
					ps93.OverlayValues[51] = d51
					ps93.OverlayValues[52] = d52
					ps93.OverlayValues[53] = d53
					ps93.OverlayValues[54] = d54
					ps93.OverlayValues[55] = d55
					ps93.OverlayValues[56] = d56
					snap94 := d1
					snap95 := d2
					snap96 := d3
					snap97 := d4
					snap98 := d5
					snap99 := d6
					snap100 := d7
					snap101 := d8
					snap102 := d9
					snap103 := d34
					snap104 := d35
					snap105 := d36
					snap106 := d37
					snap107 := d38
					snap108 := d39
					snap109 := d40
					snap110 := d41
					snap111 := d42
					snap112 := d43
					snap113 := d44
					snap114 := d45
					snap115 := d46
					snap116 := d47
					snap117 := d48
					snap118 := d49
					snap119 := d50
					snap120 := d51
					snap121 := d52
					snap122 := d53
					snap123 := d54
					snap124 := d55
					snap125 := d56
					alloc126 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps93)
					}
					ctx.RestoreAllocState(alloc126)
					d1 = snap94
					d2 = snap95
					d3 = snap96
					d4 = snap97
					d5 = snap98
					d6 = snap99
					d7 = snap100
					d8 = snap101
					d9 = snap102
					d34 = snap103
					d35 = snap104
					d36 = snap105
					d37 = snap106
					d38 = snap107
					d39 = snap108
					d40 = snap109
					d41 = snap110
					d42 = snap111
					d43 = snap112
					d44 = snap113
					d45 = snap114
					d46 = snap115
					d47 = snap116
					d48 = snap117
					d49 = snap118
					d50 = snap119
					d51 = snap120
					d52 = snap121
					d53 = snap122
					d54 = snap123
					d55 = snap124
					d56 = snap125
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps92)
					}
					return result
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d47)
					var d127 JITValueDesc
					ctx.EnsureDesc(&d54)
					if d54.Loc == LocRegPair || d54.Loc == LocRegTriple {
						d127 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d54.Reg2}
						ctx.BindReg(d54.Reg2, &d127)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d54)
					ctx.EnsureDesc(&d47)
					ctx.EnsureDesc(&d127)
					var d129 JITValueDesc
					if d127.Loc == LocImm && d47.Loc == LocImm {
						d129 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d127.Imm.Int() - d47.Imm.Int())}
					} else {
						r10 := ctx.AllocReg()
						if d127.Loc == LocImm {
							ctx.EmitMovRegImm64(r10, uint64(d127.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r10, d127.Reg)
						}
						if d47.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d47.Imm.Int()))
							ctx.EmitSubInt64(r10, RegR11)
						} else {
							ctx.EmitSubInt64(r10, d47.Reg)
						}
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d129)
					}
					var d130 JITValueDesc
					r11 := ctx.EmitSliceDataAfterLow(&d54, &d47, 16)
					d130 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
					ctx.BindReg(r11, &d130)
					ctx.BindReg(r11, &d130)
					var d131 JITValueDesc
					var r12 Reg
					var r13 Reg
					ctx.SyncDesc(&d130)
					ctx.EnsureDesc(&d130)
					if d130.Loc == LocImm {
						r12 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r12, uint64(d130.Imm.Int()))
					} else {
						r12 = d130.Reg
					}
					ctx.ProtectReg(r12)
					ctx.SyncDesc(&d129)
					ctx.EnsureDesc(&d129)
					if d129.Loc == LocImm {
						r13 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r13, uint64(d129.Imm.Int()))
					} else {
						r13 = d129.Reg
					}
					ctx.ProtectReg(r13)
					r14 := ctx.EmitSliceCapAfterLow(&d54, &d47, r12, r13)
					ctx.UnprotectReg(r13)
					ctx.UnprotectReg(r12)
					d131 = JITValueDesc{Loc: LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14}
					ctx.BindReg(r12, &d131)
					ctx.BindReg(r13, &d131)
					ctx.BindReg(r14, &d131)
					ctx.BindReg(r12, &d131)
					ctx.BindReg(r13, &d131)
					ctx.BindReg(r14, &d131)
					ctx.EnsureDesc(&d54)
					ctx.EnsureDesc(&d131)
					callResults132 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d54, d131}, []uint8{1}, []uint8{0})
					d133 = callResults132[0]
					d133.Type = tagInt
					var d134 JITValueDesc
					if d54.SliceSizeKnown {
						d134 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d54.KnownSliceLen))}
					} else if d54.Loc == LocImm {
						d134 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d54.StackOff))}
					} else if d54.Loc == LocStackTriple {
						d134 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d54.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d54)
						if d54.Loc == LocRegPair || d54.Loc == LocRegTriple {
							d134 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d54.Reg2, ID: 0}
						} else if d54.Loc == LocReg {
							d134 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d54.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d134)
					ctx.EnsureDesc(&d47)
					ctx.EnsureDescsTogether(&d134, &d47)
					var d135 JITValueDesc
					if d134.Loc == LocImm && d47.Loc == LocImm {
						d135 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d134.Imm.Int() - d47.Imm.Int())}
					} else if d47.Loc == LocImm && d47.Imm.Int() == 0 {
						r15 := ctx.AllocRegExcept(d134.Reg)
						ctx.EmitMovRegReg(r15, d134.Reg)
						d135 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
						ctx.BindReg(r15, &d135)
					} else if d134.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d47.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d134.Imm.Int()))
						ctx.EmitSubInt64(scratch, d47.Reg)
						d135 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d135)
					} else if d47.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d134.Reg)
						ctx.EmitMovRegReg(scratch, d134.Reg)
						if d47.Imm.Int() >= -2147483648 && d47.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d47.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d47.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d135 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d135)
					} else {
						r16 := ctx.AllocRegExcept(d134.Reg, d47.Reg)
						ctx.EmitMovRegReg(r16, d134.Reg)
						ctx.EmitSubInt64(r16, d47.Reg)
						d135 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
						ctx.BindReg(r16, &d135)
					}
					if d135.Loc == LocReg && d134.Loc == LocReg && d135.Reg == d134.Reg {
						ctx.TransferReg(d134.Reg)
						d134.Loc = LocNone
					}
					ctx.FreeDesc(&d134)
					ctx.EnsureDesc(&d135)
					var d136 JITValueDesc
					ctx.EnsureDesc(&d54)
					if d54.Loc == LocRegPair || d54.Loc == LocRegTriple {
						d136 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d54.Reg2}
						ctx.BindReg(d54.Reg2, &d136)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d54)
					ctx.EnsureDesc(&d135)
					ctx.EnsureDesc(&d136)
					var d138 JITValueDesc
					if d136.Loc == LocImm && d135.Loc == LocImm {
						d138 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d136.Imm.Int() - d135.Imm.Int())}
					} else {
						r17 := ctx.AllocReg()
						if d136.Loc == LocImm {
							ctx.EmitMovRegImm64(r17, uint64(d136.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r17, d136.Reg)
						}
						if d135.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d135.Imm.Int()))
							ctx.EmitSubInt64(r17, RegR11)
						} else {
							ctx.EmitSubInt64(r17, d135.Reg)
						}
						d138 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
						ctx.BindReg(r17, &d138)
					}
					var d139 JITValueDesc
					r18 := ctx.EmitSliceDataAfterLow(&d54, &d135, 16)
					d139 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
					ctx.BindReg(r18, &d139)
					ctx.BindReg(r18, &d139)
					var d140 JITValueDesc
					var r19 Reg
					var r20 Reg
					ctx.SyncDesc(&d139)
					ctx.EnsureDesc(&d139)
					if d139.Loc == LocImm {
						r19 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r19, uint64(d139.Imm.Int()))
					} else {
						r19 = d139.Reg
					}
					ctx.ProtectReg(r19)
					ctx.SyncDesc(&d138)
					ctx.EnsureDesc(&d138)
					if d138.Loc == LocImm {
						r20 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r20, uint64(d138.Imm.Int()))
					} else {
						r20 = d138.Reg
					}
					ctx.ProtectReg(r20)
					r21 := ctx.EmitSliceCapAfterLow(&d54, &d135, r19, r20)
					ctx.UnprotectReg(r20)
					ctx.UnprotectReg(r19)
					d140 = JITValueDesc{Loc: LocRegTriple, Reg: r19, Reg2: r20, Reg3: r21}
					ctx.BindReg(r19, &d140)
					ctx.BindReg(r20, &d140)
					ctx.BindReg(r21, &d140)
					ctx.BindReg(r19, &d140)
					ctx.BindReg(r20, &d140)
					ctx.BindReg(r21, &d140)
					ctx.StabilizeDescForControlFlow(&d140)
					ctx.FreeDesc(&d135)
					var d141 JITValueDesc
					if d140.SliceSizeKnown {
						d141 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d140.KnownSliceLen))}
					} else if d140.Loc == LocImm {
						d141 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d140.StackOff))}
					} else if d140.Loc == LocStackTriple {
						d141 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d140.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d140)
						if d140.Loc == LocRegPair || d140.Loc == LocRegTriple {
							d141 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d140.Reg2, ID: 0}
						} else if d140.Loc == LocReg {
							d141 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d140.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d141)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[7].PhiBase)+int32(0))
					}
					ps142 := PhiState{General: ps.General}
					ps142.OverlayValues = make([]JITValueDesc, 142)
					ps142.OverlayValues[1] = d1
					ps142.OverlayValues[2] = d2
					ps142.OverlayValues[3] = d3
					ps142.OverlayValues[4] = d4
					ps142.OverlayValues[5] = d5
					ps142.OverlayValues[6] = d6
					ps142.OverlayValues[7] = d7
					ps142.OverlayValues[8] = d8
					ps142.OverlayValues[9] = d9
					ps142.OverlayValues[34] = d34
					ps142.OverlayValues[35] = d35
					ps142.OverlayValues[36] = d36
					ps142.OverlayValues[37] = d37
					ps142.OverlayValues[38] = d38
					ps142.OverlayValues[39] = d39
					ps142.OverlayValues[40] = d40
					ps142.OverlayValues[41] = d41
					ps142.OverlayValues[42] = d42
					ps142.OverlayValues[43] = d43
					ps142.OverlayValues[44] = d44
					ps142.OverlayValues[45] = d45
					ps142.OverlayValues[46] = d46
					ps142.OverlayValues[47] = d47
					ps142.OverlayValues[48] = d48
					ps142.OverlayValues[49] = d49
					ps142.OverlayValues[50] = d50
					ps142.OverlayValues[51] = d51
					ps142.OverlayValues[52] = d52
					ps142.OverlayValues[53] = d53
					ps142.OverlayValues[54] = d54
					ps142.OverlayValues[55] = d55
					ps142.OverlayValues[56] = d56
					ps142.OverlayValues[127] = d127
					ps142.OverlayValues[128] = d128
					ps142.OverlayValues[129] = d129
					ps142.OverlayValues[130] = d130
					ps142.OverlayValues[131] = d131
					ps142.OverlayValues[133] = d133
					ps142.OverlayValues[134] = d134
					ps142.OverlayValues[135] = d135
					ps142.OverlayValues[136] = d136
					ps142.OverlayValues[137] = d137
					ps142.OverlayValues[138] = d138
					ps142.OverlayValues[139] = d139
					ps142.OverlayValues[140] = d140
					ps142.OverlayValues[141] = d141
					ps142.PhiValues = make([]JITValueDesc, 1)
					d143 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps142.PhiValues[0] = d143
					if ps142.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps142)
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					ctx.ReclaimUntrackedRegs()
					var d144 JITValueDesc
					if d54.SliceSizeKnown {
						d144 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d54.KnownSliceLen))}
					} else if d54.Loc == LocImm {
						d144 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d54.StackOff))}
					} else if d54.Loc == LocStackTriple {
						d144 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d54.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d54)
						if d54.Loc == LocRegPair || d54.Loc == LocRegTriple {
							d144 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d54.Reg2, ID: 0}
						} else if d54.Loc == LocReg {
							d144 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d54.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d144)
					ctx.EnsureDesc(&d47)
					ctx.EnsureDescsTogether(&d144, &d47)
					var d145 JITValueDesc
					if d144.Loc == LocImm && d47.Loc == LocImm {
						d145 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d144.Imm.Int() % d47.Imm.Int())}
					} else {
						d145 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntRem), []JITValueDesc{d144, d47}, 1)
					}
					if d145.Loc == LocReg && d144.Loc == LocReg && d145.Reg == d144.Reg {
						ctx.TransferReg(d144.Reg)
						d144.Loc = LocNone
					}
					ctx.FreeDesc(&d144)
					ctx.FreeDesc(&d47)
					ctx.EnsureDesc(&d145)
					var d146 JITValueDesc
					if d145.Loc == LocImm {
						d146 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d145.Imm.Int() != 0)}
					} else {
						r22 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d145.Reg, 0)
						d146 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondNotEqual}
						ctx.BindReg(r22, &d146)
					}
					ctx.FreeDesc(&d145)
					d147 = d146
					ctx.EnsureDesc(&d147)
					if d147.Loc != LocImm && d147.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d147.Loc == LocImm {
						if d147.Imm.Bool() {
							if ps.General {
							}
							ps148 := PhiState{General: ps.General}
							ps148.OverlayValues = make([]JITValueDesc, 148)
							ps148.OverlayValues[1] = d1
							ps148.OverlayValues[2] = d2
							ps148.OverlayValues[3] = d3
							ps148.OverlayValues[4] = d4
							ps148.OverlayValues[5] = d5
							ps148.OverlayValues[6] = d6
							ps148.OverlayValues[7] = d7
							ps148.OverlayValues[8] = d8
							ps148.OverlayValues[9] = d9
							ps148.OverlayValues[34] = d34
							ps148.OverlayValues[35] = d35
							ps148.OverlayValues[36] = d36
							ps148.OverlayValues[37] = d37
							ps148.OverlayValues[38] = d38
							ps148.OverlayValues[39] = d39
							ps148.OverlayValues[40] = d40
							ps148.OverlayValues[41] = d41
							ps148.OverlayValues[42] = d42
							ps148.OverlayValues[43] = d43
							ps148.OverlayValues[44] = d44
							ps148.OverlayValues[45] = d45
							ps148.OverlayValues[46] = d46
							ps148.OverlayValues[47] = d47
							ps148.OverlayValues[48] = d48
							ps148.OverlayValues[49] = d49
							ps148.OverlayValues[50] = d50
							ps148.OverlayValues[51] = d51
							ps148.OverlayValues[52] = d52
							ps148.OverlayValues[53] = d53
							ps148.OverlayValues[54] = d54
							ps148.OverlayValues[55] = d55
							ps148.OverlayValues[56] = d56
							ps148.OverlayValues[127] = d127
							ps148.OverlayValues[128] = d128
							ps148.OverlayValues[129] = d129
							ps148.OverlayValues[130] = d130
							ps148.OverlayValues[131] = d131
							ps148.OverlayValues[133] = d133
							ps148.OverlayValues[134] = d134
							ps148.OverlayValues[135] = d135
							ps148.OverlayValues[136] = d136
							ps148.OverlayValues[137] = d137
							ps148.OverlayValues[138] = d138
							ps148.OverlayValues[139] = d139
							ps148.OverlayValues[140] = d140
							ps148.OverlayValues[141] = d141
							ps148.OverlayValues[143] = d143
							ps148.OverlayValues[144] = d144
							ps148.OverlayValues[145] = d145
							ps148.OverlayValues[146] = d146
							ps148.OverlayValues[147] = d147
							return bbs[3].RenderPS(ps148)
						}
						if ps.General {
						}
						ps149 := PhiState{General: ps.General}
						ps149.OverlayValues = make([]JITValueDesc, 148)
						ps149.OverlayValues[1] = d1
						ps149.OverlayValues[2] = d2
						ps149.OverlayValues[3] = d3
						ps149.OverlayValues[4] = d4
						ps149.OverlayValues[5] = d5
						ps149.OverlayValues[6] = d6
						ps149.OverlayValues[7] = d7
						ps149.OverlayValues[8] = d8
						ps149.OverlayValues[9] = d9
						ps149.OverlayValues[34] = d34
						ps149.OverlayValues[35] = d35
						ps149.OverlayValues[36] = d36
						ps149.OverlayValues[37] = d37
						ps149.OverlayValues[38] = d38
						ps149.OverlayValues[39] = d39
						ps149.OverlayValues[40] = d40
						ps149.OverlayValues[41] = d41
						ps149.OverlayValues[42] = d42
						ps149.OverlayValues[43] = d43
						ps149.OverlayValues[44] = d44
						ps149.OverlayValues[45] = d45
						ps149.OverlayValues[46] = d46
						ps149.OverlayValues[47] = d47
						ps149.OverlayValues[48] = d48
						ps149.OverlayValues[49] = d49
						ps149.OverlayValues[50] = d50
						ps149.OverlayValues[51] = d51
						ps149.OverlayValues[52] = d52
						ps149.OverlayValues[53] = d53
						ps149.OverlayValues[54] = d54
						ps149.OverlayValues[55] = d55
						ps149.OverlayValues[56] = d56
						ps149.OverlayValues[127] = d127
						ps149.OverlayValues[128] = d128
						ps149.OverlayValues[129] = d129
						ps149.OverlayValues[130] = d130
						ps149.OverlayValues[131] = d131
						ps149.OverlayValues[133] = d133
						ps149.OverlayValues[134] = d134
						ps149.OverlayValues[135] = d135
						ps149.OverlayValues[136] = d136
						ps149.OverlayValues[137] = d137
						ps149.OverlayValues[138] = d138
						ps149.OverlayValues[139] = d139
						ps149.OverlayValues[140] = d140
						ps149.OverlayValues[141] = d141
						ps149.OverlayValues[143] = d143
						ps149.OverlayValues[144] = d144
						ps149.OverlayValues[145] = d145
						ps149.OverlayValues[146] = d146
						ps149.OverlayValues[147] = d147
						return bbs[4].RenderPS(ps149)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					ctx.EmitJump(d147.Condition, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d146)
					snap150 := d1
					snap151 := d2
					snap152 := d3
					snap153 := d4
					snap154 := d5
					snap155 := d6
					snap156 := d7
					snap157 := d8
					snap158 := d9
					snap159 := d34
					snap160 := d35
					snap161 := d36
					snap162 := d37
					snap163 := d38
					snap164 := d39
					snap165 := d40
					snap166 := d41
					snap167 := d42
					snap168 := d43
					snap169 := d44
					snap170 := d45
					snap171 := d46
					snap172 := d47
					snap173 := d48
					snap174 := d49
					snap175 := d50
					snap176 := d51
					snap177 := d52
					snap178 := d53
					snap179 := d54
					snap180 := d55
					snap181 := d56
					snap182 := d127
					snap183 := d128
					snap184 := d129
					snap185 := d130
					snap186 := d131
					snap187 := d133
					snap188 := d134
					snap189 := d135
					snap190 := d136
					snap191 := d137
					snap192 := d138
					snap193 := d139
					snap194 := d140
					snap195 := d141
					snap196 := d143
					snap197 := d144
					snap198 := d145
					snap199 := d146
					snap200 := d147
					alloc201 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc201)
					d1 = snap150
					d2 = snap151
					d3 = snap152
					d4 = snap153
					d5 = snap154
					d6 = snap155
					d7 = snap156
					d8 = snap157
					d9 = snap158
					d34 = snap159
					d35 = snap160
					d36 = snap161
					d37 = snap162
					d38 = snap163
					d39 = snap164
					d40 = snap165
					d41 = snap166
					d42 = snap167
					d43 = snap168
					d44 = snap169
					d45 = snap170
					d46 = snap171
					d47 = snap172
					d48 = snap173
					d49 = snap174
					d50 = snap175
					d51 = snap176
					d52 = snap177
					d53 = snap178
					d54 = snap179
					d55 = snap180
					d56 = snap181
					d127 = snap182
					d128 = snap183
					d129 = snap184
					d130 = snap185
					d131 = snap186
					d133 = snap187
					d134 = snap188
					d135 = snap189
					d136 = snap190
					d137 = snap191
					d138 = snap192
					d139 = snap193
					d140 = snap194
					d141 = snap195
					d143 = snap196
					d144 = snap197
					d145 = snap198
					d146 = snap199
					d147 = snap200
					ctx.RestoreAllocState(alloc201)
					d1 = snap150
					d2 = snap151
					d3 = snap152
					d4 = snap153
					d5 = snap154
					d6 = snap155
					d7 = snap156
					d8 = snap157
					d9 = snap158
					d34 = snap159
					d35 = snap160
					d36 = snap161
					d37 = snap162
					d38 = snap163
					d39 = snap164
					d40 = snap165
					d41 = snap166
					d42 = snap167
					d43 = snap168
					d44 = snap169
					d45 = snap170
					d46 = snap171
					d47 = snap172
					d48 = snap173
					d49 = snap174
					d50 = snap175
					d51 = snap176
					d52 = snap177
					d53 = snap178
					d54 = snap179
					d55 = snap180
					d56 = snap181
					d127 = snap182
					d128 = snap183
					d129 = snap184
					d130 = snap185
					d131 = snap186
					d133 = snap187
					d134 = snap188
					d135 = snap189
					d136 = snap190
					d137 = snap191
					d138 = snap192
					d139 = snap193
					d140 = snap194
					d141 = snap195
					d143 = snap196
					d144 = snap197
					d145 = snap198
					d146 = snap199
					d147 = snap200
					ps202 := PhiState{General: true}
					ps202.OverlayValues = make([]JITValueDesc, 148)
					ps202.OverlayValues[1] = d1
					ps202.OverlayValues[2] = d2
					ps202.OverlayValues[3] = d3
					ps202.OverlayValues[4] = d4
					ps202.OverlayValues[5] = d5
					ps202.OverlayValues[6] = d6
					ps202.OverlayValues[7] = d7
					ps202.OverlayValues[8] = d8
					ps202.OverlayValues[9] = d9
					ps202.OverlayValues[34] = d34
					ps202.OverlayValues[35] = d35
					ps202.OverlayValues[36] = d36
					ps202.OverlayValues[37] = d37
					ps202.OverlayValues[38] = d38
					ps202.OverlayValues[39] = d39
					ps202.OverlayValues[40] = d40
					ps202.OverlayValues[41] = d41
					ps202.OverlayValues[42] = d42
					ps202.OverlayValues[43] = d43
					ps202.OverlayValues[44] = d44
					ps202.OverlayValues[45] = d45
					ps202.OverlayValues[46] = d46
					ps202.OverlayValues[47] = d47
					ps202.OverlayValues[48] = d48
					ps202.OverlayValues[49] = d49
					ps202.OverlayValues[50] = d50
					ps202.OverlayValues[51] = d51
					ps202.OverlayValues[52] = d52
					ps202.OverlayValues[53] = d53
					ps202.OverlayValues[54] = d54
					ps202.OverlayValues[55] = d55
					ps202.OverlayValues[56] = d56
					ps202.OverlayValues[127] = d127
					ps202.OverlayValues[128] = d128
					ps202.OverlayValues[129] = d129
					ps202.OverlayValues[130] = d130
					ps202.OverlayValues[131] = d131
					ps202.OverlayValues[133] = d133
					ps202.OverlayValues[134] = d134
					ps202.OverlayValues[135] = d135
					ps202.OverlayValues[136] = d136
					ps202.OverlayValues[137] = d137
					ps202.OverlayValues[138] = d138
					ps202.OverlayValues[139] = d139
					ps202.OverlayValues[140] = d140
					ps202.OverlayValues[141] = d141
					ps202.OverlayValues[143] = d143
					ps202.OverlayValues[144] = d144
					ps202.OverlayValues[145] = d145
					ps202.OverlayValues[146] = d146
					ps202.OverlayValues[147] = d147
					ps203 := PhiState{General: true}
					ps203.OverlayValues = make([]JITValueDesc, 148)
					ps203.OverlayValues[1] = d1
					ps203.OverlayValues[2] = d2
					ps203.OverlayValues[3] = d3
					ps203.OverlayValues[4] = d4
					ps203.OverlayValues[5] = d5
					ps203.OverlayValues[6] = d6
					ps203.OverlayValues[7] = d7
					ps203.OverlayValues[8] = d8
					ps203.OverlayValues[9] = d9
					ps203.OverlayValues[34] = d34
					ps203.OverlayValues[35] = d35
					ps203.OverlayValues[36] = d36
					ps203.OverlayValues[37] = d37
					ps203.OverlayValues[38] = d38
					ps203.OverlayValues[39] = d39
					ps203.OverlayValues[40] = d40
					ps203.OverlayValues[41] = d41
					ps203.OverlayValues[42] = d42
					ps203.OverlayValues[43] = d43
					ps203.OverlayValues[44] = d44
					ps203.OverlayValues[45] = d45
					ps203.OverlayValues[46] = d46
					ps203.OverlayValues[47] = d47
					ps203.OverlayValues[48] = d48
					ps203.OverlayValues[49] = d49
					ps203.OverlayValues[50] = d50
					ps203.OverlayValues[51] = d51
					ps203.OverlayValues[52] = d52
					ps203.OverlayValues[53] = d53
					ps203.OverlayValues[54] = d54
					ps203.OverlayValues[55] = d55
					ps203.OverlayValues[56] = d56
					ps203.OverlayValues[127] = d127
					ps203.OverlayValues[128] = d128
					ps203.OverlayValues[129] = d129
					ps203.OverlayValues[130] = d130
					ps203.OverlayValues[131] = d131
					ps203.OverlayValues[133] = d133
					ps203.OverlayValues[134] = d134
					ps203.OverlayValues[135] = d135
					ps203.OverlayValues[136] = d136
					ps203.OverlayValues[137] = d137
					ps203.OverlayValues[138] = d138
					ps203.OverlayValues[139] = d139
					ps203.OverlayValues[140] = d140
					ps203.OverlayValues[141] = d141
					ps203.OverlayValues[143] = d143
					ps203.OverlayValues[144] = d144
					ps203.OverlayValues[145] = d145
					ps203.OverlayValues[146] = d146
					ps203.OverlayValues[147] = d147
					snap204 := d1
					snap205 := d2
					snap206 := d3
					snap207 := d4
					snap208 := d5
					snap209 := d6
					snap210 := d7
					snap211 := d8
					snap212 := d9
					snap213 := d34
					snap214 := d35
					snap215 := d36
					snap216 := d37
					snap217 := d38
					snap218 := d39
					snap219 := d40
					snap220 := d41
					snap221 := d42
					snap222 := d43
					snap223 := d44
					snap224 := d45
					snap225 := d46
					snap226 := d47
					snap227 := d48
					snap228 := d49
					snap229 := d50
					snap230 := d51
					snap231 := d52
					snap232 := d53
					snap233 := d54
					snap234 := d55
					snap235 := d56
					snap236 := d127
					snap237 := d128
					snap238 := d129
					snap239 := d130
					snap240 := d131
					snap241 := d133
					snap242 := d134
					snap243 := d135
					snap244 := d136
					snap245 := d137
					snap246 := d138
					snap247 := d139
					snap248 := d140
					snap249 := d141
					snap250 := d143
					snap251 := d144
					snap252 := d145
					snap253 := d146
					snap254 := d147
					alloc255 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps203)
					}
					ctx.RestoreAllocState(alloc255)
					d1 = snap204
					d2 = snap205
					d3 = snap206
					d4 = snap207
					d5 = snap208
					d6 = snap209
					d7 = snap210
					d8 = snap211
					d9 = snap212
					d34 = snap213
					d35 = snap214
					d36 = snap215
					d37 = snap216
					d38 = snap217
					d39 = snap218
					d40 = snap219
					d41 = snap220
					d42 = snap221
					d43 = snap222
					d44 = snap223
					d45 = snap224
					d46 = snap225
					d47 = snap226
					d48 = snap227
					d49 = snap228
					d50 = snap229
					d51 = snap230
					d52 = snap231
					d53 = snap232
					d54 = snap233
					d55 = snap234
					d56 = snap235
					d127 = snap236
					d128 = snap237
					d129 = snap238
					d130 = snap239
					d131 = snap240
					d133 = snap241
					d134 = snap242
					d135 = snap243
					d136 = snap244
					d137 = snap245
					d138 = snap246
					d139 = snap247
					d140 = snap248
					d141 = snap249
					d143 = snap250
					d144 = snap251
					d145 = snap252
					d146 = snap253
					d147 = snap254
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps202)
					}
					return result
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					ctx.ReclaimUntrackedRegs()
					var d256 JITValueDesc
					if d54.SliceSizeKnown {
						d256 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d54.KnownSliceLen))}
					} else if d54.Loc == LocImm {
						d256 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d54.StackOff))}
					} else if d54.Loc == LocStackTriple {
						d256 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d54.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d54)
						if d54.Loc == LocRegPair || d54.Loc == LocRegTriple {
							d256 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d54.Reg2, ID: 0}
						} else if d54.Loc == LocReg {
							d256 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d54.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d256)
					var d257 JITValueDesc
					if d256.Loc == LocImm {
						d257 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d256.Imm.Int() == 0)}
					} else {
						r23 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d256.Reg, 0)
						d257 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r23, Condition: CondEqual}
						ctx.BindReg(r23, &d257)
					}
					ctx.FreeDesc(&d256)
					d258 = d257
					ctx.EnsureDesc(&d258)
					if d258.Loc != LocImm && d258.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d258.Loc == LocImm {
						if d258.Imm.Bool() {
							if ps.General {
							}
							ps259 := PhiState{General: ps.General}
							ps259.OverlayValues = make([]JITValueDesc, 259)
							ps259.OverlayValues[1] = d1
							ps259.OverlayValues[2] = d2
							ps259.OverlayValues[3] = d3
							ps259.OverlayValues[4] = d4
							ps259.OverlayValues[5] = d5
							ps259.OverlayValues[6] = d6
							ps259.OverlayValues[7] = d7
							ps259.OverlayValues[8] = d8
							ps259.OverlayValues[9] = d9
							ps259.OverlayValues[34] = d34
							ps259.OverlayValues[35] = d35
							ps259.OverlayValues[36] = d36
							ps259.OverlayValues[37] = d37
							ps259.OverlayValues[38] = d38
							ps259.OverlayValues[39] = d39
							ps259.OverlayValues[40] = d40
							ps259.OverlayValues[41] = d41
							ps259.OverlayValues[42] = d42
							ps259.OverlayValues[43] = d43
							ps259.OverlayValues[44] = d44
							ps259.OverlayValues[45] = d45
							ps259.OverlayValues[46] = d46
							ps259.OverlayValues[47] = d47
							ps259.OverlayValues[48] = d48
							ps259.OverlayValues[49] = d49
							ps259.OverlayValues[50] = d50
							ps259.OverlayValues[51] = d51
							ps259.OverlayValues[52] = d52
							ps259.OverlayValues[53] = d53
							ps259.OverlayValues[54] = d54
							ps259.OverlayValues[55] = d55
							ps259.OverlayValues[56] = d56
							ps259.OverlayValues[127] = d127
							ps259.OverlayValues[128] = d128
							ps259.OverlayValues[129] = d129
							ps259.OverlayValues[130] = d130
							ps259.OverlayValues[131] = d131
							ps259.OverlayValues[133] = d133
							ps259.OverlayValues[134] = d134
							ps259.OverlayValues[135] = d135
							ps259.OverlayValues[136] = d136
							ps259.OverlayValues[137] = d137
							ps259.OverlayValues[138] = d138
							ps259.OverlayValues[139] = d139
							ps259.OverlayValues[140] = d140
							ps259.OverlayValues[141] = d141
							ps259.OverlayValues[143] = d143
							ps259.OverlayValues[144] = d144
							ps259.OverlayValues[145] = d145
							ps259.OverlayValues[146] = d146
							ps259.OverlayValues[147] = d147
							ps259.OverlayValues[256] = d256
							ps259.OverlayValues[257] = d257
							ps259.OverlayValues[258] = d258
							return bbs[3].RenderPS(ps259)
						}
						if ps.General {
						}
						ps260 := PhiState{General: ps.General}
						ps260.OverlayValues = make([]JITValueDesc, 259)
						ps260.OverlayValues[1] = d1
						ps260.OverlayValues[2] = d2
						ps260.OverlayValues[3] = d3
						ps260.OverlayValues[4] = d4
						ps260.OverlayValues[5] = d5
						ps260.OverlayValues[6] = d6
						ps260.OverlayValues[7] = d7
						ps260.OverlayValues[8] = d8
						ps260.OverlayValues[9] = d9
						ps260.OverlayValues[34] = d34
						ps260.OverlayValues[35] = d35
						ps260.OverlayValues[36] = d36
						ps260.OverlayValues[37] = d37
						ps260.OverlayValues[38] = d38
						ps260.OverlayValues[39] = d39
						ps260.OverlayValues[40] = d40
						ps260.OverlayValues[41] = d41
						ps260.OverlayValues[42] = d42
						ps260.OverlayValues[43] = d43
						ps260.OverlayValues[44] = d44
						ps260.OverlayValues[45] = d45
						ps260.OverlayValues[46] = d46
						ps260.OverlayValues[47] = d47
						ps260.OverlayValues[48] = d48
						ps260.OverlayValues[49] = d49
						ps260.OverlayValues[50] = d50
						ps260.OverlayValues[51] = d51
						ps260.OverlayValues[52] = d52
						ps260.OverlayValues[53] = d53
						ps260.OverlayValues[54] = d54
						ps260.OverlayValues[55] = d55
						ps260.OverlayValues[56] = d56
						ps260.OverlayValues[127] = d127
						ps260.OverlayValues[128] = d128
						ps260.OverlayValues[129] = d129
						ps260.OverlayValues[130] = d130
						ps260.OverlayValues[131] = d131
						ps260.OverlayValues[133] = d133
						ps260.OverlayValues[134] = d134
						ps260.OverlayValues[135] = d135
						ps260.OverlayValues[136] = d136
						ps260.OverlayValues[137] = d137
						ps260.OverlayValues[138] = d138
						ps260.OverlayValues[139] = d139
						ps260.OverlayValues[140] = d140
						ps260.OverlayValues[141] = d141
						ps260.OverlayValues[143] = d143
						ps260.OverlayValues[144] = d144
						ps260.OverlayValues[145] = d145
						ps260.OverlayValues[146] = d146
						ps260.OverlayValues[147] = d147
						ps260.OverlayValues[256] = d256
						ps260.OverlayValues[257] = d257
						ps260.OverlayValues[258] = d258
						return bbs[5].RenderPS(ps260)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					ctx.EmitJump(d258.Condition, lbl4)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d257)
					snap261 := d1
					snap262 := d2
					snap263 := d3
					snap264 := d4
					snap265 := d5
					snap266 := d6
					snap267 := d7
					snap268 := d8
					snap269 := d9
					snap270 := d34
					snap271 := d35
					snap272 := d36
					snap273 := d37
					snap274 := d38
					snap275 := d39
					snap276 := d40
					snap277 := d41
					snap278 := d42
					snap279 := d43
					snap280 := d44
					snap281 := d45
					snap282 := d46
					snap283 := d47
					snap284 := d48
					snap285 := d49
					snap286 := d50
					snap287 := d51
					snap288 := d52
					snap289 := d53
					snap290 := d54
					snap291 := d55
					snap292 := d56
					snap293 := d127
					snap294 := d128
					snap295 := d129
					snap296 := d130
					snap297 := d131
					snap298 := d133
					snap299 := d134
					snap300 := d135
					snap301 := d136
					snap302 := d137
					snap303 := d138
					snap304 := d139
					snap305 := d140
					snap306 := d141
					snap307 := d143
					snap308 := d144
					snap309 := d145
					snap310 := d146
					snap311 := d147
					snap312 := d256
					snap313 := d257
					snap314 := d258
					alloc315 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc315)
					d1 = snap261
					d2 = snap262
					d3 = snap263
					d4 = snap264
					d5 = snap265
					d6 = snap266
					d7 = snap267
					d8 = snap268
					d9 = snap269
					d34 = snap270
					d35 = snap271
					d36 = snap272
					d37 = snap273
					d38 = snap274
					d39 = snap275
					d40 = snap276
					d41 = snap277
					d42 = snap278
					d43 = snap279
					d44 = snap280
					d45 = snap281
					d46 = snap282
					d47 = snap283
					d48 = snap284
					d49 = snap285
					d50 = snap286
					d51 = snap287
					d52 = snap288
					d53 = snap289
					d54 = snap290
					d55 = snap291
					d56 = snap292
					d127 = snap293
					d128 = snap294
					d129 = snap295
					d130 = snap296
					d131 = snap297
					d133 = snap298
					d134 = snap299
					d135 = snap300
					d136 = snap301
					d137 = snap302
					d138 = snap303
					d139 = snap304
					d140 = snap305
					d141 = snap306
					d143 = snap307
					d144 = snap308
					d145 = snap309
					d146 = snap310
					d147 = snap311
					d256 = snap312
					d257 = snap313
					d258 = snap314
					ctx.RestoreAllocState(alloc315)
					d1 = snap261
					d2 = snap262
					d3 = snap263
					d4 = snap264
					d5 = snap265
					d6 = snap266
					d7 = snap267
					d8 = snap268
					d9 = snap269
					d34 = snap270
					d35 = snap271
					d36 = snap272
					d37 = snap273
					d38 = snap274
					d39 = snap275
					d40 = snap276
					d41 = snap277
					d42 = snap278
					d43 = snap279
					d44 = snap280
					d45 = snap281
					d46 = snap282
					d47 = snap283
					d48 = snap284
					d49 = snap285
					d50 = snap286
					d51 = snap287
					d52 = snap288
					d53 = snap289
					d54 = snap290
					d55 = snap291
					d56 = snap292
					d127 = snap293
					d128 = snap294
					d129 = snap295
					d130 = snap296
					d131 = snap297
					d133 = snap298
					d134 = snap299
					d135 = snap300
					d136 = snap301
					d137 = snap302
					d138 = snap303
					d139 = snap304
					d140 = snap305
					d141 = snap306
					d143 = snap307
					d144 = snap308
					d145 = snap309
					d146 = snap310
					d147 = snap311
					d256 = snap312
					d257 = snap313
					d258 = snap314
					ps316 := PhiState{General: true}
					ps316.OverlayValues = make([]JITValueDesc, 259)
					ps316.OverlayValues[1] = d1
					ps316.OverlayValues[2] = d2
					ps316.OverlayValues[3] = d3
					ps316.OverlayValues[4] = d4
					ps316.OverlayValues[5] = d5
					ps316.OverlayValues[6] = d6
					ps316.OverlayValues[7] = d7
					ps316.OverlayValues[8] = d8
					ps316.OverlayValues[9] = d9
					ps316.OverlayValues[34] = d34
					ps316.OverlayValues[35] = d35
					ps316.OverlayValues[36] = d36
					ps316.OverlayValues[37] = d37
					ps316.OverlayValues[38] = d38
					ps316.OverlayValues[39] = d39
					ps316.OverlayValues[40] = d40
					ps316.OverlayValues[41] = d41
					ps316.OverlayValues[42] = d42
					ps316.OverlayValues[43] = d43
					ps316.OverlayValues[44] = d44
					ps316.OverlayValues[45] = d45
					ps316.OverlayValues[46] = d46
					ps316.OverlayValues[47] = d47
					ps316.OverlayValues[48] = d48
					ps316.OverlayValues[49] = d49
					ps316.OverlayValues[50] = d50
					ps316.OverlayValues[51] = d51
					ps316.OverlayValues[52] = d52
					ps316.OverlayValues[53] = d53
					ps316.OverlayValues[54] = d54
					ps316.OverlayValues[55] = d55
					ps316.OverlayValues[56] = d56
					ps316.OverlayValues[127] = d127
					ps316.OverlayValues[128] = d128
					ps316.OverlayValues[129] = d129
					ps316.OverlayValues[130] = d130
					ps316.OverlayValues[131] = d131
					ps316.OverlayValues[133] = d133
					ps316.OverlayValues[134] = d134
					ps316.OverlayValues[135] = d135
					ps316.OverlayValues[136] = d136
					ps316.OverlayValues[137] = d137
					ps316.OverlayValues[138] = d138
					ps316.OverlayValues[139] = d139
					ps316.OverlayValues[140] = d140
					ps316.OverlayValues[141] = d141
					ps316.OverlayValues[143] = d143
					ps316.OverlayValues[144] = d144
					ps316.OverlayValues[145] = d145
					ps316.OverlayValues[146] = d146
					ps316.OverlayValues[147] = d147
					ps316.OverlayValues[256] = d256
					ps316.OverlayValues[257] = d257
					ps316.OverlayValues[258] = d258
					ps317 := PhiState{General: true}
					ps317.OverlayValues = make([]JITValueDesc, 259)
					ps317.OverlayValues[1] = d1
					ps317.OverlayValues[2] = d2
					ps317.OverlayValues[3] = d3
					ps317.OverlayValues[4] = d4
					ps317.OverlayValues[5] = d5
					ps317.OverlayValues[6] = d6
					ps317.OverlayValues[7] = d7
					ps317.OverlayValues[8] = d8
					ps317.OverlayValues[9] = d9
					ps317.OverlayValues[34] = d34
					ps317.OverlayValues[35] = d35
					ps317.OverlayValues[36] = d36
					ps317.OverlayValues[37] = d37
					ps317.OverlayValues[38] = d38
					ps317.OverlayValues[39] = d39
					ps317.OverlayValues[40] = d40
					ps317.OverlayValues[41] = d41
					ps317.OverlayValues[42] = d42
					ps317.OverlayValues[43] = d43
					ps317.OverlayValues[44] = d44
					ps317.OverlayValues[45] = d45
					ps317.OverlayValues[46] = d46
					ps317.OverlayValues[47] = d47
					ps317.OverlayValues[48] = d48
					ps317.OverlayValues[49] = d49
					ps317.OverlayValues[50] = d50
					ps317.OverlayValues[51] = d51
					ps317.OverlayValues[52] = d52
					ps317.OverlayValues[53] = d53
					ps317.OverlayValues[54] = d54
					ps317.OverlayValues[55] = d55
					ps317.OverlayValues[56] = d56
					ps317.OverlayValues[127] = d127
					ps317.OverlayValues[128] = d128
					ps317.OverlayValues[129] = d129
					ps317.OverlayValues[130] = d130
					ps317.OverlayValues[131] = d131
					ps317.OverlayValues[133] = d133
					ps317.OverlayValues[134] = d134
					ps317.OverlayValues[135] = d135
					ps317.OverlayValues[136] = d136
					ps317.OverlayValues[137] = d137
					ps317.OverlayValues[138] = d138
					ps317.OverlayValues[139] = d139
					ps317.OverlayValues[140] = d140
					ps317.OverlayValues[141] = d141
					ps317.OverlayValues[143] = d143
					ps317.OverlayValues[144] = d144
					ps317.OverlayValues[145] = d145
					ps317.OverlayValues[146] = d146
					ps317.OverlayValues[147] = d147
					ps317.OverlayValues[256] = d256
					ps317.OverlayValues[257] = d257
					ps317.OverlayValues[258] = d258
					snap318 := d1
					snap319 := d2
					snap320 := d3
					snap321 := d4
					snap322 := d5
					snap323 := d6
					snap324 := d7
					snap325 := d8
					snap326 := d9
					snap327 := d34
					snap328 := d35
					snap329 := d36
					snap330 := d37
					snap331 := d38
					snap332 := d39
					snap333 := d40
					snap334 := d41
					snap335 := d42
					snap336 := d43
					snap337 := d44
					snap338 := d45
					snap339 := d46
					snap340 := d47
					snap341 := d48
					snap342 := d49
					snap343 := d50
					snap344 := d51
					snap345 := d52
					snap346 := d53
					snap347 := d54
					snap348 := d55
					snap349 := d56
					snap350 := d127
					snap351 := d128
					snap352 := d129
					snap353 := d130
					snap354 := d131
					snap355 := d133
					snap356 := d134
					snap357 := d135
					snap358 := d136
					snap359 := d137
					snap360 := d138
					snap361 := d139
					snap362 := d140
					snap363 := d141
					snap364 := d143
					snap365 := d144
					snap366 := d145
					snap367 := d146
					snap368 := d147
					snap369 := d256
					snap370 := d257
					snap371 := d258
					alloc372 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps317)
					}
					ctx.RestoreAllocState(alloc372)
					d1 = snap318
					d2 = snap319
					d3 = snap320
					d4 = snap321
					d5 = snap322
					d6 = snap323
					d7 = snap324
					d8 = snap325
					d9 = snap326
					d34 = snap327
					d35 = snap328
					d36 = snap329
					d37 = snap330
					d38 = snap331
					d39 = snap332
					d40 = snap333
					d41 = snap334
					d42 = snap335
					d43 = snap336
					d44 = snap337
					d45 = snap338
					d46 = snap339
					d47 = snap340
					d48 = snap341
					d49 = snap342
					d50 = snap343
					d51 = snap344
					d52 = snap345
					d53 = snap346
					d54 = snap347
					d55 = snap348
					d56 = snap349
					d127 = snap350
					d128 = snap351
					d129 = snap352
					d130 = snap353
					d131 = snap354
					d133 = snap355
					d134 = snap356
					d135 = snap357
					d136 = snap358
					d137 = snap359
					d138 = snap360
					d139 = snap361
					d140 = snap362
					d141 = snap363
					d143 = snap364
					d144 = snap365
					d145 = snap366
					d146 = snap367
					d147 = snap368
					d256 = snap369
					d257 = snap370
					d258 = snap371
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps316)
					}
					return result
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d373 := ps.PhiValues[0]
							ctx.EnsureDesc(&d373)
							ctx.EmitStoreToStack(d373, int32(bbs[7].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != LocNone {
						d256 = ps.OverlayValues[256]
					}
					if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
						d257 = ps.OverlayValues[257]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != LocNone {
						d373 = ps.OverlayValues[373]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d374 JITValueDesc
					if d1.Loc == LocImm {
						d374 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d374 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d374)
					}
					if d374.Loc == LocReg && d1.Loc == LocReg && d374.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d374)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d374)
					ctx.EnsureDesc(&d141)
					ctx.EnsureDescsTogether(&d374, &d141)
					var d375 JITValueDesc
					if d374.Loc == LocImm && d141.Loc == LocImm {
						d375 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d374.Imm.Int() < d141.Imm.Int())}
					} else if d141.Loc == LocImm {
						r24 := ctx.AllocRegExcept(d374.Reg)
						if d141.Imm.Int() >= -2147483648 && d141.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d374.Reg, int32(d141.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d141.Imm.Int()))
							ctx.EmitCmpInt64(d374.Reg, RegR11)
						}
						d375 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r24, Condition: CondSignedLess}
						ctx.BindReg(r24, &d375)
					} else if d374.Loc == LocImm {
						r25 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d374.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d141.Reg)
						d375 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r25, Condition: CondSignedLess}
						ctx.BindReg(r25, &d375)
					} else {
						r26 := ctx.AllocRegExcept(d374.Reg)
						ctx.EmitCmpInt64(d374.Reg, d141.Reg)
						d375 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondSignedLess}
						ctx.BindReg(r26, &d375)
					}
					d376 = d375
					ctx.EnsureDesc(&d376)
					if d376.Loc != LocImm && d376.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d376.Loc == LocImm {
						if d376.Imm.Bool() {
							if ps.General {
							}
							ps377 := PhiState{General: ps.General}
							ps377.OverlayValues = make([]JITValueDesc, 377)
							ps377.OverlayValues[1] = d1
							ps377.OverlayValues[2] = d2
							ps377.OverlayValues[3] = d3
							ps377.OverlayValues[4] = d4
							ps377.OverlayValues[5] = d5
							ps377.OverlayValues[6] = d6
							ps377.OverlayValues[7] = d7
							ps377.OverlayValues[8] = d8
							ps377.OverlayValues[9] = d9
							ps377.OverlayValues[34] = d34
							ps377.OverlayValues[35] = d35
							ps377.OverlayValues[36] = d36
							ps377.OverlayValues[37] = d37
							ps377.OverlayValues[38] = d38
							ps377.OverlayValues[39] = d39
							ps377.OverlayValues[40] = d40
							ps377.OverlayValues[41] = d41
							ps377.OverlayValues[42] = d42
							ps377.OverlayValues[43] = d43
							ps377.OverlayValues[44] = d44
							ps377.OverlayValues[45] = d45
							ps377.OverlayValues[46] = d46
							ps377.OverlayValues[47] = d47
							ps377.OverlayValues[48] = d48
							ps377.OverlayValues[49] = d49
							ps377.OverlayValues[50] = d50
							ps377.OverlayValues[51] = d51
							ps377.OverlayValues[52] = d52
							ps377.OverlayValues[53] = d53
							ps377.OverlayValues[54] = d54
							ps377.OverlayValues[55] = d55
							ps377.OverlayValues[56] = d56
							ps377.OverlayValues[127] = d127
							ps377.OverlayValues[128] = d128
							ps377.OverlayValues[129] = d129
							ps377.OverlayValues[130] = d130
							ps377.OverlayValues[131] = d131
							ps377.OverlayValues[133] = d133
							ps377.OverlayValues[134] = d134
							ps377.OverlayValues[135] = d135
							ps377.OverlayValues[136] = d136
							ps377.OverlayValues[137] = d137
							ps377.OverlayValues[138] = d138
							ps377.OverlayValues[139] = d139
							ps377.OverlayValues[140] = d140
							ps377.OverlayValues[141] = d141
							ps377.OverlayValues[143] = d143
							ps377.OverlayValues[144] = d144
							ps377.OverlayValues[145] = d145
							ps377.OverlayValues[146] = d146
							ps377.OverlayValues[147] = d147
							ps377.OverlayValues[256] = d256
							ps377.OverlayValues[257] = d257
							ps377.OverlayValues[258] = d258
							ps377.OverlayValues[373] = d373
							ps377.OverlayValues[374] = d374
							ps377.OverlayValues[375] = d375
							ps377.OverlayValues[376] = d376
							return bbs[8].RenderPS(ps377)
						}
						if ps.General {
						}
						ps378 := PhiState{General: ps.General}
						ps378.OverlayValues = make([]JITValueDesc, 377)
						ps378.OverlayValues[1] = d1
						ps378.OverlayValues[2] = d2
						ps378.OverlayValues[3] = d3
						ps378.OverlayValues[4] = d4
						ps378.OverlayValues[5] = d5
						ps378.OverlayValues[6] = d6
						ps378.OverlayValues[7] = d7
						ps378.OverlayValues[8] = d8
						ps378.OverlayValues[9] = d9
						ps378.OverlayValues[34] = d34
						ps378.OverlayValues[35] = d35
						ps378.OverlayValues[36] = d36
						ps378.OverlayValues[37] = d37
						ps378.OverlayValues[38] = d38
						ps378.OverlayValues[39] = d39
						ps378.OverlayValues[40] = d40
						ps378.OverlayValues[41] = d41
						ps378.OverlayValues[42] = d42
						ps378.OverlayValues[43] = d43
						ps378.OverlayValues[44] = d44
						ps378.OverlayValues[45] = d45
						ps378.OverlayValues[46] = d46
						ps378.OverlayValues[47] = d47
						ps378.OverlayValues[48] = d48
						ps378.OverlayValues[49] = d49
						ps378.OverlayValues[50] = d50
						ps378.OverlayValues[51] = d51
						ps378.OverlayValues[52] = d52
						ps378.OverlayValues[53] = d53
						ps378.OverlayValues[54] = d54
						ps378.OverlayValues[55] = d55
						ps378.OverlayValues[56] = d56
						ps378.OverlayValues[127] = d127
						ps378.OverlayValues[128] = d128
						ps378.OverlayValues[129] = d129
						ps378.OverlayValues[130] = d130
						ps378.OverlayValues[131] = d131
						ps378.OverlayValues[133] = d133
						ps378.OverlayValues[134] = d134
						ps378.OverlayValues[135] = d135
						ps378.OverlayValues[136] = d136
						ps378.OverlayValues[137] = d137
						ps378.OverlayValues[138] = d138
						ps378.OverlayValues[139] = d139
						ps378.OverlayValues[140] = d140
						ps378.OverlayValues[141] = d141
						ps378.OverlayValues[143] = d143
						ps378.OverlayValues[144] = d144
						ps378.OverlayValues[145] = d145
						ps378.OverlayValues[146] = d146
						ps378.OverlayValues[147] = d147
						ps378.OverlayValues[256] = d256
						ps378.OverlayValues[257] = d257
						ps378.OverlayValues[258] = d258
						ps378.OverlayValues[373] = d373
						ps378.OverlayValues[374] = d374
						ps378.OverlayValues[375] = d375
						ps378.OverlayValues[376] = d376
						return bbs[9].RenderPS(ps378)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d379 := ps.PhiValues[0]
							ctx.EnsureDesc(&d379)
							ctx.EmitStoreToStack(d379, int32(bbs[7].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					ctx.EmitJump(d376.Condition, lbl9)
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
					}
					ctx.FreeDesc(&d375)
					snap380 := d1
					snap381 := d2
					snap382 := d3
					snap383 := d4
					snap384 := d5
					snap385 := d6
					snap386 := d7
					snap387 := d8
					snap388 := d9
					snap389 := d34
					snap390 := d35
					snap391 := d36
					snap392 := d37
					snap393 := d38
					snap394 := d39
					snap395 := d40
					snap396 := d41
					snap397 := d42
					snap398 := d43
					snap399 := d44
					snap400 := d45
					snap401 := d46
					snap402 := d47
					snap403 := d48
					snap404 := d49
					snap405 := d50
					snap406 := d51
					snap407 := d52
					snap408 := d53
					snap409 := d54
					snap410 := d55
					snap411 := d56
					snap412 := d127
					snap413 := d128
					snap414 := d129
					snap415 := d130
					snap416 := d131
					snap417 := d133
					snap418 := d134
					snap419 := d135
					snap420 := d136
					snap421 := d137
					snap422 := d138
					snap423 := d139
					snap424 := d140
					snap425 := d141
					snap426 := d143
					snap427 := d144
					snap428 := d145
					snap429 := d146
					snap430 := d147
					snap431 := d256
					snap432 := d257
					snap433 := d258
					snap434 := d373
					snap435 := d374
					snap436 := d375
					snap437 := d376
					snap438 := d379
					alloc439 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc439)
					d1 = snap380
					d2 = snap381
					d3 = snap382
					d4 = snap383
					d5 = snap384
					d6 = snap385
					d7 = snap386
					d8 = snap387
					d9 = snap388
					d34 = snap389
					d35 = snap390
					d36 = snap391
					d37 = snap392
					d38 = snap393
					d39 = snap394
					d40 = snap395
					d41 = snap396
					d42 = snap397
					d43 = snap398
					d44 = snap399
					d45 = snap400
					d46 = snap401
					d47 = snap402
					d48 = snap403
					d49 = snap404
					d50 = snap405
					d51 = snap406
					d52 = snap407
					d53 = snap408
					d54 = snap409
					d55 = snap410
					d56 = snap411
					d127 = snap412
					d128 = snap413
					d129 = snap414
					d130 = snap415
					d131 = snap416
					d133 = snap417
					d134 = snap418
					d135 = snap419
					d136 = snap420
					d137 = snap421
					d138 = snap422
					d139 = snap423
					d140 = snap424
					d141 = snap425
					d143 = snap426
					d144 = snap427
					d145 = snap428
					d146 = snap429
					d147 = snap430
					d256 = snap431
					d257 = snap432
					d258 = snap433
					d373 = snap434
					d374 = snap435
					d375 = snap436
					d376 = snap437
					d379 = snap438
					ctx.RestoreAllocState(alloc439)
					d1 = snap380
					d2 = snap381
					d3 = snap382
					d4 = snap383
					d5 = snap384
					d6 = snap385
					d7 = snap386
					d8 = snap387
					d9 = snap388
					d34 = snap389
					d35 = snap390
					d36 = snap391
					d37 = snap392
					d38 = snap393
					d39 = snap394
					d40 = snap395
					d41 = snap396
					d42 = snap397
					d43 = snap398
					d44 = snap399
					d45 = snap400
					d46 = snap401
					d47 = snap402
					d48 = snap403
					d49 = snap404
					d50 = snap405
					d51 = snap406
					d52 = snap407
					d53 = snap408
					d54 = snap409
					d55 = snap410
					d56 = snap411
					d127 = snap412
					d128 = snap413
					d129 = snap414
					d130 = snap415
					d131 = snap416
					d133 = snap417
					d134 = snap418
					d135 = snap419
					d136 = snap420
					d137 = snap421
					d138 = snap422
					d139 = snap423
					d140 = snap424
					d141 = snap425
					d143 = snap426
					d144 = snap427
					d145 = snap428
					d146 = snap429
					d147 = snap430
					d256 = snap431
					d257 = snap432
					d258 = snap433
					d373 = snap434
					d374 = snap435
					d375 = snap436
					d376 = snap437
					d379 = snap438
					ps440 := PhiState{General: true}
					ps440.OverlayValues = make([]JITValueDesc, 380)
					ps440.OverlayValues[1] = d1
					ps440.OverlayValues[2] = d2
					ps440.OverlayValues[3] = d3
					ps440.OverlayValues[4] = d4
					ps440.OverlayValues[5] = d5
					ps440.OverlayValues[6] = d6
					ps440.OverlayValues[7] = d7
					ps440.OverlayValues[8] = d8
					ps440.OverlayValues[9] = d9
					ps440.OverlayValues[34] = d34
					ps440.OverlayValues[35] = d35
					ps440.OverlayValues[36] = d36
					ps440.OverlayValues[37] = d37
					ps440.OverlayValues[38] = d38
					ps440.OverlayValues[39] = d39
					ps440.OverlayValues[40] = d40
					ps440.OverlayValues[41] = d41
					ps440.OverlayValues[42] = d42
					ps440.OverlayValues[43] = d43
					ps440.OverlayValues[44] = d44
					ps440.OverlayValues[45] = d45
					ps440.OverlayValues[46] = d46
					ps440.OverlayValues[47] = d47
					ps440.OverlayValues[48] = d48
					ps440.OverlayValues[49] = d49
					ps440.OverlayValues[50] = d50
					ps440.OverlayValues[51] = d51
					ps440.OverlayValues[52] = d52
					ps440.OverlayValues[53] = d53
					ps440.OverlayValues[54] = d54
					ps440.OverlayValues[55] = d55
					ps440.OverlayValues[56] = d56
					ps440.OverlayValues[127] = d127
					ps440.OverlayValues[128] = d128
					ps440.OverlayValues[129] = d129
					ps440.OverlayValues[130] = d130
					ps440.OverlayValues[131] = d131
					ps440.OverlayValues[133] = d133
					ps440.OverlayValues[134] = d134
					ps440.OverlayValues[135] = d135
					ps440.OverlayValues[136] = d136
					ps440.OverlayValues[137] = d137
					ps440.OverlayValues[138] = d138
					ps440.OverlayValues[139] = d139
					ps440.OverlayValues[140] = d140
					ps440.OverlayValues[141] = d141
					ps440.OverlayValues[143] = d143
					ps440.OverlayValues[144] = d144
					ps440.OverlayValues[145] = d145
					ps440.OverlayValues[146] = d146
					ps440.OverlayValues[147] = d147
					ps440.OverlayValues[256] = d256
					ps440.OverlayValues[257] = d257
					ps440.OverlayValues[258] = d258
					ps440.OverlayValues[373] = d373
					ps440.OverlayValues[374] = d374
					ps440.OverlayValues[375] = d375
					ps440.OverlayValues[376] = d376
					ps440.OverlayValues[379] = d379
					ps441 := PhiState{General: true}
					ps441.OverlayValues = make([]JITValueDesc, 380)
					ps441.OverlayValues[1] = d1
					ps441.OverlayValues[2] = d2
					ps441.OverlayValues[3] = d3
					ps441.OverlayValues[4] = d4
					ps441.OverlayValues[5] = d5
					ps441.OverlayValues[6] = d6
					ps441.OverlayValues[7] = d7
					ps441.OverlayValues[8] = d8
					ps441.OverlayValues[9] = d9
					ps441.OverlayValues[34] = d34
					ps441.OverlayValues[35] = d35
					ps441.OverlayValues[36] = d36
					ps441.OverlayValues[37] = d37
					ps441.OverlayValues[38] = d38
					ps441.OverlayValues[39] = d39
					ps441.OverlayValues[40] = d40
					ps441.OverlayValues[41] = d41
					ps441.OverlayValues[42] = d42
					ps441.OverlayValues[43] = d43
					ps441.OverlayValues[44] = d44
					ps441.OverlayValues[45] = d45
					ps441.OverlayValues[46] = d46
					ps441.OverlayValues[47] = d47
					ps441.OverlayValues[48] = d48
					ps441.OverlayValues[49] = d49
					ps441.OverlayValues[50] = d50
					ps441.OverlayValues[51] = d51
					ps441.OverlayValues[52] = d52
					ps441.OverlayValues[53] = d53
					ps441.OverlayValues[54] = d54
					ps441.OverlayValues[55] = d55
					ps441.OverlayValues[56] = d56
					ps441.OverlayValues[127] = d127
					ps441.OverlayValues[128] = d128
					ps441.OverlayValues[129] = d129
					ps441.OverlayValues[130] = d130
					ps441.OverlayValues[131] = d131
					ps441.OverlayValues[133] = d133
					ps441.OverlayValues[134] = d134
					ps441.OverlayValues[135] = d135
					ps441.OverlayValues[136] = d136
					ps441.OverlayValues[137] = d137
					ps441.OverlayValues[138] = d138
					ps441.OverlayValues[139] = d139
					ps441.OverlayValues[140] = d140
					ps441.OverlayValues[141] = d141
					ps441.OverlayValues[143] = d143
					ps441.OverlayValues[144] = d144
					ps441.OverlayValues[145] = d145
					ps441.OverlayValues[146] = d146
					ps441.OverlayValues[147] = d147
					ps441.OverlayValues[256] = d256
					ps441.OverlayValues[257] = d257
					ps441.OverlayValues[258] = d258
					ps441.OverlayValues[373] = d373
					ps441.OverlayValues[374] = d374
					ps441.OverlayValues[375] = d375
					ps441.OverlayValues[376] = d376
					ps441.OverlayValues[379] = d379
					snap442 := d1
					snap443 := d2
					snap444 := d3
					snap445 := d4
					snap446 := d5
					snap447 := d6
					snap448 := d7
					snap449 := d8
					snap450 := d9
					snap451 := d34
					snap452 := d35
					snap453 := d36
					snap454 := d37
					snap455 := d38
					snap456 := d39
					snap457 := d40
					snap458 := d41
					snap459 := d42
					snap460 := d43
					snap461 := d44
					snap462 := d45
					snap463 := d46
					snap464 := d47
					snap465 := d48
					snap466 := d49
					snap467 := d50
					snap468 := d51
					snap469 := d52
					snap470 := d53
					snap471 := d54
					snap472 := d55
					snap473 := d56
					snap474 := d127
					snap475 := d128
					snap476 := d129
					snap477 := d130
					snap478 := d131
					snap479 := d133
					snap480 := d134
					snap481 := d135
					snap482 := d136
					snap483 := d137
					snap484 := d138
					snap485 := d139
					snap486 := d140
					snap487 := d141
					snap488 := d143
					snap489 := d144
					snap490 := d145
					snap491 := d146
					snap492 := d147
					snap493 := d256
					snap494 := d257
					snap495 := d258
					snap496 := d373
					snap497 := d374
					snap498 := d375
					snap499 := d376
					snap500 := d379
					alloc501 := ctx.SnapshotAllocState()
					if !bbs[9].Rendered {
						bbs[9].RenderPS(ps441)
					}
					ctx.RestoreAllocState(alloc501)
					d1 = snap442
					d2 = snap443
					d3 = snap444
					d4 = snap445
					d5 = snap446
					d6 = snap447
					d7 = snap448
					d8 = snap449
					d9 = snap450
					d34 = snap451
					d35 = snap452
					d36 = snap453
					d37 = snap454
					d38 = snap455
					d39 = snap456
					d40 = snap457
					d41 = snap458
					d42 = snap459
					d43 = snap460
					d44 = snap461
					d45 = snap462
					d46 = snap463
					d47 = snap464
					d48 = snap465
					d49 = snap466
					d50 = snap467
					d51 = snap468
					d52 = snap469
					d53 = snap470
					d54 = snap471
					d55 = snap472
					d56 = snap473
					d127 = snap474
					d128 = snap475
					d129 = snap476
					d130 = snap477
					d131 = snap478
					d133 = snap479
					d134 = snap480
					d135 = snap481
					d136 = snap482
					d137 = snap483
					d138 = snap484
					d139 = snap485
					d140 = snap486
					d141 = snap487
					d143 = snap488
					d144 = snap489
					d145 = snap490
					d146 = snap491
					d147 = snap492
					d256 = snap493
					d257 = snap494
					d258 = snap495
					d373 = snap496
					d374 = snap497
					d375 = snap498
					d376 = snap499
					d379 = snap500
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps440)
					}
					return result
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != LocNone {
						d256 = ps.OverlayValues[256]
					}
					if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
						d257 = ps.OverlayValues[257]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != LocNone {
						d373 = ps.OverlayValues[373]
					}
					if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != LocNone {
						d374 = ps.OverlayValues[374]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != LocNone {
						d376 = ps.OverlayValues[376]
					}
					if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != LocNone {
						d379 = ps.OverlayValues[379]
					}
					ctx.ReclaimUntrackedRegs()
					var d502 JITValueDesc
					if d6.SliceSizeKnown {
						d502 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d6.KnownSliceLen))}
					} else if d6.Loc == LocImm {
						d502 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d6.StackOff))}
					} else if d6.Loc == LocStackTriple {
						d502 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d6.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d6)
						if d6.Loc == LocRegPair || d6.Loc == LocRegTriple {
							d502 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg2, ID: 0}
						} else if d6.Loc == LocReg {
							d502 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d374)
					ctx.EnsureDesc(&d502)
					ctx.EnsureDescsTogether(&d374, &d502)
					var d503 JITValueDesc
					if d374.Loc == LocImm && d502.Loc == LocImm {
						d503 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d374.Imm.Int() < d502.Imm.Int())}
					} else if d502.Loc == LocImm {
						r27 := ctx.AllocRegExcept(d374.Reg)
						if d502.Imm.Int() >= -2147483648 && d502.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d374.Reg, int32(d502.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d502.Imm.Int()))
							ctx.EmitCmpInt64(d374.Reg, RegR11)
						}
						d503 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r27, Condition: CondSignedLess}
						ctx.BindReg(r27, &d503)
					} else if d374.Loc == LocImm {
						r28 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d374.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d502.Reg)
						d503 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r28, Condition: CondSignedLess}
						ctx.BindReg(r28, &d503)
					} else {
						r29 := ctx.AllocRegExcept(d374.Reg)
						ctx.EmitCmpInt64(d374.Reg, d502.Reg)
						d503 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r29, Condition: CondSignedLess}
						ctx.BindReg(r29, &d503)
					}
					ctx.FreeDesc(&d502)
					d504 = d503
					ctx.EnsureDesc(&d504)
					if d504.Loc != LocImm && d504.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d504.Loc == LocImm {
						if d504.Imm.Bool() {
							if ps.General {
							}
							ps505 := PhiState{General: ps.General}
							ps505.OverlayValues = make([]JITValueDesc, 505)
							ps505.OverlayValues[1] = d1
							ps505.OverlayValues[2] = d2
							ps505.OverlayValues[3] = d3
							ps505.OverlayValues[4] = d4
							ps505.OverlayValues[5] = d5
							ps505.OverlayValues[6] = d6
							ps505.OverlayValues[7] = d7
							ps505.OverlayValues[8] = d8
							ps505.OverlayValues[9] = d9
							ps505.OverlayValues[34] = d34
							ps505.OverlayValues[35] = d35
							ps505.OverlayValues[36] = d36
							ps505.OverlayValues[37] = d37
							ps505.OverlayValues[38] = d38
							ps505.OverlayValues[39] = d39
							ps505.OverlayValues[40] = d40
							ps505.OverlayValues[41] = d41
							ps505.OverlayValues[42] = d42
							ps505.OverlayValues[43] = d43
							ps505.OverlayValues[44] = d44
							ps505.OverlayValues[45] = d45
							ps505.OverlayValues[46] = d46
							ps505.OverlayValues[47] = d47
							ps505.OverlayValues[48] = d48
							ps505.OverlayValues[49] = d49
							ps505.OverlayValues[50] = d50
							ps505.OverlayValues[51] = d51
							ps505.OverlayValues[52] = d52
							ps505.OverlayValues[53] = d53
							ps505.OverlayValues[54] = d54
							ps505.OverlayValues[55] = d55
							ps505.OverlayValues[56] = d56
							ps505.OverlayValues[127] = d127
							ps505.OverlayValues[128] = d128
							ps505.OverlayValues[129] = d129
							ps505.OverlayValues[130] = d130
							ps505.OverlayValues[131] = d131
							ps505.OverlayValues[133] = d133
							ps505.OverlayValues[134] = d134
							ps505.OverlayValues[135] = d135
							ps505.OverlayValues[136] = d136
							ps505.OverlayValues[137] = d137
							ps505.OverlayValues[138] = d138
							ps505.OverlayValues[139] = d139
							ps505.OverlayValues[140] = d140
							ps505.OverlayValues[141] = d141
							ps505.OverlayValues[143] = d143
							ps505.OverlayValues[144] = d144
							ps505.OverlayValues[145] = d145
							ps505.OverlayValues[146] = d146
							ps505.OverlayValues[147] = d147
							ps505.OverlayValues[256] = d256
							ps505.OverlayValues[257] = d257
							ps505.OverlayValues[258] = d258
							ps505.OverlayValues[373] = d373
							ps505.OverlayValues[374] = d374
							ps505.OverlayValues[375] = d375
							ps505.OverlayValues[376] = d376
							ps505.OverlayValues[379] = d379
							ps505.OverlayValues[502] = d502
							ps505.OverlayValues[503] = d503
							ps505.OverlayValues[504] = d504
							return bbs[10].RenderPS(ps505)
						}
						if ps.General {
						}
						ps506 := PhiState{General: ps.General}
						ps506.OverlayValues = make([]JITValueDesc, 505)
						ps506.OverlayValues[1] = d1
						ps506.OverlayValues[2] = d2
						ps506.OverlayValues[3] = d3
						ps506.OverlayValues[4] = d4
						ps506.OverlayValues[5] = d5
						ps506.OverlayValues[6] = d6
						ps506.OverlayValues[7] = d7
						ps506.OverlayValues[8] = d8
						ps506.OverlayValues[9] = d9
						ps506.OverlayValues[34] = d34
						ps506.OverlayValues[35] = d35
						ps506.OverlayValues[36] = d36
						ps506.OverlayValues[37] = d37
						ps506.OverlayValues[38] = d38
						ps506.OverlayValues[39] = d39
						ps506.OverlayValues[40] = d40
						ps506.OverlayValues[41] = d41
						ps506.OverlayValues[42] = d42
						ps506.OverlayValues[43] = d43
						ps506.OverlayValues[44] = d44
						ps506.OverlayValues[45] = d45
						ps506.OverlayValues[46] = d46
						ps506.OverlayValues[47] = d47
						ps506.OverlayValues[48] = d48
						ps506.OverlayValues[49] = d49
						ps506.OverlayValues[50] = d50
						ps506.OverlayValues[51] = d51
						ps506.OverlayValues[52] = d52
						ps506.OverlayValues[53] = d53
						ps506.OverlayValues[54] = d54
						ps506.OverlayValues[55] = d55
						ps506.OverlayValues[56] = d56
						ps506.OverlayValues[127] = d127
						ps506.OverlayValues[128] = d128
						ps506.OverlayValues[129] = d129
						ps506.OverlayValues[130] = d130
						ps506.OverlayValues[131] = d131
						ps506.OverlayValues[133] = d133
						ps506.OverlayValues[134] = d134
						ps506.OverlayValues[135] = d135
						ps506.OverlayValues[136] = d136
						ps506.OverlayValues[137] = d137
						ps506.OverlayValues[138] = d138
						ps506.OverlayValues[139] = d139
						ps506.OverlayValues[140] = d140
						ps506.OverlayValues[141] = d141
						ps506.OverlayValues[143] = d143
						ps506.OverlayValues[144] = d144
						ps506.OverlayValues[145] = d145
						ps506.OverlayValues[146] = d146
						ps506.OverlayValues[147] = d147
						ps506.OverlayValues[256] = d256
						ps506.OverlayValues[257] = d257
						ps506.OverlayValues[258] = d258
						ps506.OverlayValues[373] = d373
						ps506.OverlayValues[374] = d374
						ps506.OverlayValues[375] = d375
						ps506.OverlayValues[376] = d376
						ps506.OverlayValues[379] = d379
						ps506.OverlayValues[502] = d502
						ps506.OverlayValues[503] = d503
						ps506.OverlayValues[504] = d504
						return bbs[11].RenderPS(ps506)
					}
					if !ps.General {
						ps.General = true
						return bbs[8].RenderPS(ps)
					}
					ctx.EmitJump(d504.Condition, lbl11)
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
					}
					ctx.FreeDesc(&d503)
					snap507 := d1
					snap508 := d2
					snap509 := d3
					snap510 := d4
					snap511 := d5
					snap512 := d6
					snap513 := d7
					snap514 := d8
					snap515 := d9
					snap516 := d34
					snap517 := d35
					snap518 := d36
					snap519 := d37
					snap520 := d38
					snap521 := d39
					snap522 := d40
					snap523 := d41
					snap524 := d42
					snap525 := d43
					snap526 := d44
					snap527 := d45
					snap528 := d46
					snap529 := d47
					snap530 := d48
					snap531 := d49
					snap532 := d50
					snap533 := d51
					snap534 := d52
					snap535 := d53
					snap536 := d54
					snap537 := d55
					snap538 := d56
					snap539 := d127
					snap540 := d128
					snap541 := d129
					snap542 := d130
					snap543 := d131
					snap544 := d133
					snap545 := d134
					snap546 := d135
					snap547 := d136
					snap548 := d137
					snap549 := d138
					snap550 := d139
					snap551 := d140
					snap552 := d141
					snap553 := d143
					snap554 := d144
					snap555 := d145
					snap556 := d146
					snap557 := d147
					snap558 := d256
					snap559 := d257
					snap560 := d258
					snap561 := d373
					snap562 := d374
					snap563 := d375
					snap564 := d376
					snap565 := d379
					snap566 := d502
					snap567 := d503
					snap568 := d504
					alloc569 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc569)
					d1 = snap507
					d2 = snap508
					d3 = snap509
					d4 = snap510
					d5 = snap511
					d6 = snap512
					d7 = snap513
					d8 = snap514
					d9 = snap515
					d34 = snap516
					d35 = snap517
					d36 = snap518
					d37 = snap519
					d38 = snap520
					d39 = snap521
					d40 = snap522
					d41 = snap523
					d42 = snap524
					d43 = snap525
					d44 = snap526
					d45 = snap527
					d46 = snap528
					d47 = snap529
					d48 = snap530
					d49 = snap531
					d50 = snap532
					d51 = snap533
					d52 = snap534
					d53 = snap535
					d54 = snap536
					d55 = snap537
					d56 = snap538
					d127 = snap539
					d128 = snap540
					d129 = snap541
					d130 = snap542
					d131 = snap543
					d133 = snap544
					d134 = snap545
					d135 = snap546
					d136 = snap547
					d137 = snap548
					d138 = snap549
					d139 = snap550
					d140 = snap551
					d141 = snap552
					d143 = snap553
					d144 = snap554
					d145 = snap555
					d146 = snap556
					d147 = snap557
					d256 = snap558
					d257 = snap559
					d258 = snap560
					d373 = snap561
					d374 = snap562
					d375 = snap563
					d376 = snap564
					d379 = snap565
					d502 = snap566
					d503 = snap567
					d504 = snap568
					ctx.RestoreAllocState(alloc569)
					d1 = snap507
					d2 = snap508
					d3 = snap509
					d4 = snap510
					d5 = snap511
					d6 = snap512
					d7 = snap513
					d8 = snap514
					d9 = snap515
					d34 = snap516
					d35 = snap517
					d36 = snap518
					d37 = snap519
					d38 = snap520
					d39 = snap521
					d40 = snap522
					d41 = snap523
					d42 = snap524
					d43 = snap525
					d44 = snap526
					d45 = snap527
					d46 = snap528
					d47 = snap529
					d48 = snap530
					d49 = snap531
					d50 = snap532
					d51 = snap533
					d52 = snap534
					d53 = snap535
					d54 = snap536
					d55 = snap537
					d56 = snap538
					d127 = snap539
					d128 = snap540
					d129 = snap541
					d130 = snap542
					d131 = snap543
					d133 = snap544
					d134 = snap545
					d135 = snap546
					d136 = snap547
					d137 = snap548
					d138 = snap549
					d139 = snap550
					d140 = snap551
					d141 = snap552
					d143 = snap553
					d144 = snap554
					d145 = snap555
					d146 = snap556
					d147 = snap557
					d256 = snap558
					d257 = snap559
					d258 = snap560
					d373 = snap561
					d374 = snap562
					d375 = snap563
					d376 = snap564
					d379 = snap565
					d502 = snap566
					d503 = snap567
					d504 = snap568
					ps570 := PhiState{General: true}
					ps570.OverlayValues = make([]JITValueDesc, 505)
					ps570.OverlayValues[1] = d1
					ps570.OverlayValues[2] = d2
					ps570.OverlayValues[3] = d3
					ps570.OverlayValues[4] = d4
					ps570.OverlayValues[5] = d5
					ps570.OverlayValues[6] = d6
					ps570.OverlayValues[7] = d7
					ps570.OverlayValues[8] = d8
					ps570.OverlayValues[9] = d9
					ps570.OverlayValues[34] = d34
					ps570.OverlayValues[35] = d35
					ps570.OverlayValues[36] = d36
					ps570.OverlayValues[37] = d37
					ps570.OverlayValues[38] = d38
					ps570.OverlayValues[39] = d39
					ps570.OverlayValues[40] = d40
					ps570.OverlayValues[41] = d41
					ps570.OverlayValues[42] = d42
					ps570.OverlayValues[43] = d43
					ps570.OverlayValues[44] = d44
					ps570.OverlayValues[45] = d45
					ps570.OverlayValues[46] = d46
					ps570.OverlayValues[47] = d47
					ps570.OverlayValues[48] = d48
					ps570.OverlayValues[49] = d49
					ps570.OverlayValues[50] = d50
					ps570.OverlayValues[51] = d51
					ps570.OverlayValues[52] = d52
					ps570.OverlayValues[53] = d53
					ps570.OverlayValues[54] = d54
					ps570.OverlayValues[55] = d55
					ps570.OverlayValues[56] = d56
					ps570.OverlayValues[127] = d127
					ps570.OverlayValues[128] = d128
					ps570.OverlayValues[129] = d129
					ps570.OverlayValues[130] = d130
					ps570.OverlayValues[131] = d131
					ps570.OverlayValues[133] = d133
					ps570.OverlayValues[134] = d134
					ps570.OverlayValues[135] = d135
					ps570.OverlayValues[136] = d136
					ps570.OverlayValues[137] = d137
					ps570.OverlayValues[138] = d138
					ps570.OverlayValues[139] = d139
					ps570.OverlayValues[140] = d140
					ps570.OverlayValues[141] = d141
					ps570.OverlayValues[143] = d143
					ps570.OverlayValues[144] = d144
					ps570.OverlayValues[145] = d145
					ps570.OverlayValues[146] = d146
					ps570.OverlayValues[147] = d147
					ps570.OverlayValues[256] = d256
					ps570.OverlayValues[257] = d257
					ps570.OverlayValues[258] = d258
					ps570.OverlayValues[373] = d373
					ps570.OverlayValues[374] = d374
					ps570.OverlayValues[375] = d375
					ps570.OverlayValues[376] = d376
					ps570.OverlayValues[379] = d379
					ps570.OverlayValues[502] = d502
					ps570.OverlayValues[503] = d503
					ps570.OverlayValues[504] = d504
					ps571 := PhiState{General: true}
					ps571.OverlayValues = make([]JITValueDesc, 505)
					ps571.OverlayValues[1] = d1
					ps571.OverlayValues[2] = d2
					ps571.OverlayValues[3] = d3
					ps571.OverlayValues[4] = d4
					ps571.OverlayValues[5] = d5
					ps571.OverlayValues[6] = d6
					ps571.OverlayValues[7] = d7
					ps571.OverlayValues[8] = d8
					ps571.OverlayValues[9] = d9
					ps571.OverlayValues[34] = d34
					ps571.OverlayValues[35] = d35
					ps571.OverlayValues[36] = d36
					ps571.OverlayValues[37] = d37
					ps571.OverlayValues[38] = d38
					ps571.OverlayValues[39] = d39
					ps571.OverlayValues[40] = d40
					ps571.OverlayValues[41] = d41
					ps571.OverlayValues[42] = d42
					ps571.OverlayValues[43] = d43
					ps571.OverlayValues[44] = d44
					ps571.OverlayValues[45] = d45
					ps571.OverlayValues[46] = d46
					ps571.OverlayValues[47] = d47
					ps571.OverlayValues[48] = d48
					ps571.OverlayValues[49] = d49
					ps571.OverlayValues[50] = d50
					ps571.OverlayValues[51] = d51
					ps571.OverlayValues[52] = d52
					ps571.OverlayValues[53] = d53
					ps571.OverlayValues[54] = d54
					ps571.OverlayValues[55] = d55
					ps571.OverlayValues[56] = d56
					ps571.OverlayValues[127] = d127
					ps571.OverlayValues[128] = d128
					ps571.OverlayValues[129] = d129
					ps571.OverlayValues[130] = d130
					ps571.OverlayValues[131] = d131
					ps571.OverlayValues[133] = d133
					ps571.OverlayValues[134] = d134
					ps571.OverlayValues[135] = d135
					ps571.OverlayValues[136] = d136
					ps571.OverlayValues[137] = d137
					ps571.OverlayValues[138] = d138
					ps571.OverlayValues[139] = d139
					ps571.OverlayValues[140] = d140
					ps571.OverlayValues[141] = d141
					ps571.OverlayValues[143] = d143
					ps571.OverlayValues[144] = d144
					ps571.OverlayValues[145] = d145
					ps571.OverlayValues[146] = d146
					ps571.OverlayValues[147] = d147
					ps571.OverlayValues[256] = d256
					ps571.OverlayValues[257] = d257
					ps571.OverlayValues[258] = d258
					ps571.OverlayValues[373] = d373
					ps571.OverlayValues[374] = d374
					ps571.OverlayValues[375] = d375
					ps571.OverlayValues[376] = d376
					ps571.OverlayValues[379] = d379
					ps571.OverlayValues[502] = d502
					ps571.OverlayValues[503] = d503
					ps571.OverlayValues[504] = d504
					snap572 := d1
					snap573 := d2
					snap574 := d3
					snap575 := d4
					snap576 := d5
					snap577 := d6
					snap578 := d7
					snap579 := d8
					snap580 := d9
					snap581 := d34
					snap582 := d35
					snap583 := d36
					snap584 := d37
					snap585 := d38
					snap586 := d39
					snap587 := d40
					snap588 := d41
					snap589 := d42
					snap590 := d43
					snap591 := d44
					snap592 := d45
					snap593 := d46
					snap594 := d47
					snap595 := d48
					snap596 := d49
					snap597 := d50
					snap598 := d51
					snap599 := d52
					snap600 := d53
					snap601 := d54
					snap602 := d55
					snap603 := d56
					snap604 := d127
					snap605 := d128
					snap606 := d129
					snap607 := d130
					snap608 := d131
					snap609 := d133
					snap610 := d134
					snap611 := d135
					snap612 := d136
					snap613 := d137
					snap614 := d138
					snap615 := d139
					snap616 := d140
					snap617 := d141
					snap618 := d143
					snap619 := d144
					snap620 := d145
					snap621 := d146
					snap622 := d147
					snap623 := d256
					snap624 := d257
					snap625 := d258
					snap626 := d373
					snap627 := d374
					snap628 := d375
					snap629 := d376
					snap630 := d379
					snap631 := d502
					snap632 := d503
					snap633 := d504
					alloc634 := ctx.SnapshotAllocState()
					if !bbs[11].Rendered {
						bbs[11].RenderPS(ps571)
					}
					ctx.RestoreAllocState(alloc634)
					d1 = snap572
					d2 = snap573
					d3 = snap574
					d4 = snap575
					d5 = snap576
					d6 = snap577
					d7 = snap578
					d8 = snap579
					d9 = snap580
					d34 = snap581
					d35 = snap582
					d36 = snap583
					d37 = snap584
					d38 = snap585
					d39 = snap586
					d40 = snap587
					d41 = snap588
					d42 = snap589
					d43 = snap590
					d44 = snap591
					d45 = snap592
					d46 = snap593
					d47 = snap594
					d48 = snap595
					d49 = snap596
					d50 = snap597
					d51 = snap598
					d52 = snap599
					d53 = snap600
					d54 = snap601
					d55 = snap602
					d56 = snap603
					d127 = snap604
					d128 = snap605
					d129 = snap606
					d130 = snap607
					d131 = snap608
					d133 = snap609
					d134 = snap610
					d135 = snap611
					d136 = snap612
					d137 = snap613
					d138 = snap614
					d139 = snap615
					d140 = snap616
					d141 = snap617
					d143 = snap618
					d144 = snap619
					d145 = snap620
					d146 = snap621
					d147 = snap622
					d256 = snap623
					d257 = snap624
					d258 = snap625
					d373 = snap626
					d374 = snap627
					d375 = snap628
					d376 = snap629
					d379 = snap630
					d502 = snap631
					d503 = snap632
					d504 = snap633
					if !bbs[10].Rendered {
						return bbs[10].RenderPS(ps570)
					}
					return result
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != LocNone {
						d256 = ps.OverlayValues[256]
					}
					if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
						d257 = ps.OverlayValues[257]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != LocNone {
						d373 = ps.OverlayValues[373]
					}
					if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != LocNone {
						d374 = ps.OverlayValues[374]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != LocNone {
						d376 = ps.OverlayValues[376]
					}
					if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != LocNone {
						d379 = ps.OverlayValues[379]
					}
					if len(ps.OverlayValues) > 502 && ps.OverlayValues[502].Loc != LocNone {
						d502 = ps.OverlayValues[502]
					}
					if len(ps.OverlayValues) > 503 && ps.OverlayValues[503].Loc != LocNone {
						d503 = ps.OverlayValues[503]
					}
					if len(ps.OverlayValues) > 504 && ps.OverlayValues[504].Loc != LocNone {
						d504 = ps.OverlayValues[504]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d42)
					ctx.EnsureDesc(&d42)
					var d635 JITValueDesc
					if d42.Loc == LocImm {
						d635 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d42.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d42.Reg)
						ctx.EmitMovRegReg(scratch, d42.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d635 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d635)
					}
					if d635.Loc == LocReg && d42.Loc == LocReg && d635.Reg == d42.Reg {
						ctx.TransferReg(d42.Reg)
						d42.Loc = LocNone
					}
					ctx.FreeDesc(&d42)
					ctx.EnsureDesc(&d635)
					ctx.EnsureDesc(&d635)
					ctx.EnsureDesc(&d635)
					d637 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.SyncDesc(&d635)
					d638 = d3
					d638.ID = 0
					d639 = d637
					d639.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d638, &d639, &d635, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d638, &d639, &d635, int32(16))
					}
					ctx.FreeDesc(&d639)
					ctx.EnsureDesc(&d37)
					var d640 JITValueDesc
					if d37.Loc == LocImm {
						d640 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d37.Imm.Int() > 0)}
					} else {
						r30 := ctx.AllocRegExcept(d37.Reg)
						ctx.EmitCmpRegImm32(d37.Reg, 0)
						d640 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r30, Condition: CondSignedGreater}
						ctx.BindReg(r30, &d640)
					}
					d641 = d640
					ctx.EnsureDesc(&d641)
					if d641.Loc != LocImm && d641.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d641.Loc == LocImm {
						if d641.Imm.Bool() {
							if ps.General {
							}
							ps642 := PhiState{General: ps.General}
							ps642.OverlayValues = make([]JITValueDesc, 642)
							ps642.OverlayValues[1] = d1
							ps642.OverlayValues[2] = d2
							ps642.OverlayValues[3] = d3
							ps642.OverlayValues[4] = d4
							ps642.OverlayValues[5] = d5
							ps642.OverlayValues[6] = d6
							ps642.OverlayValues[7] = d7
							ps642.OverlayValues[8] = d8
							ps642.OverlayValues[9] = d9
							ps642.OverlayValues[34] = d34
							ps642.OverlayValues[35] = d35
							ps642.OverlayValues[36] = d36
							ps642.OverlayValues[37] = d37
							ps642.OverlayValues[38] = d38
							ps642.OverlayValues[39] = d39
							ps642.OverlayValues[40] = d40
							ps642.OverlayValues[41] = d41
							ps642.OverlayValues[42] = d42
							ps642.OverlayValues[43] = d43
							ps642.OverlayValues[44] = d44
							ps642.OverlayValues[45] = d45
							ps642.OverlayValues[46] = d46
							ps642.OverlayValues[47] = d47
							ps642.OverlayValues[48] = d48
							ps642.OverlayValues[49] = d49
							ps642.OverlayValues[50] = d50
							ps642.OverlayValues[51] = d51
							ps642.OverlayValues[52] = d52
							ps642.OverlayValues[53] = d53
							ps642.OverlayValues[54] = d54
							ps642.OverlayValues[55] = d55
							ps642.OverlayValues[56] = d56
							ps642.OverlayValues[127] = d127
							ps642.OverlayValues[128] = d128
							ps642.OverlayValues[129] = d129
							ps642.OverlayValues[130] = d130
							ps642.OverlayValues[131] = d131
							ps642.OverlayValues[133] = d133
							ps642.OverlayValues[134] = d134
							ps642.OverlayValues[135] = d135
							ps642.OverlayValues[136] = d136
							ps642.OverlayValues[137] = d137
							ps642.OverlayValues[138] = d138
							ps642.OverlayValues[139] = d139
							ps642.OverlayValues[140] = d140
							ps642.OverlayValues[141] = d141
							ps642.OverlayValues[143] = d143
							ps642.OverlayValues[144] = d144
							ps642.OverlayValues[145] = d145
							ps642.OverlayValues[146] = d146
							ps642.OverlayValues[147] = d147
							ps642.OverlayValues[256] = d256
							ps642.OverlayValues[257] = d257
							ps642.OverlayValues[258] = d258
							ps642.OverlayValues[373] = d373
							ps642.OverlayValues[374] = d374
							ps642.OverlayValues[375] = d375
							ps642.OverlayValues[376] = d376
							ps642.OverlayValues[379] = d379
							ps642.OverlayValues[502] = d502
							ps642.OverlayValues[503] = d503
							ps642.OverlayValues[504] = d504
							ps642.OverlayValues[635] = d635
							ps642.OverlayValues[636] = d636
							ps642.OverlayValues[637] = d637
							ps642.OverlayValues[638] = d638
							ps642.OverlayValues[639] = d639
							ps642.OverlayValues[640] = d640
							ps642.OverlayValues[641] = d641
							return bbs[12].RenderPS(ps642)
						}
						if ps.General {
						}
						ps643 := PhiState{General: ps.General}
						ps643.OverlayValues = make([]JITValueDesc, 642)
						ps643.OverlayValues[1] = d1
						ps643.OverlayValues[2] = d2
						ps643.OverlayValues[3] = d3
						ps643.OverlayValues[4] = d4
						ps643.OverlayValues[5] = d5
						ps643.OverlayValues[6] = d6
						ps643.OverlayValues[7] = d7
						ps643.OverlayValues[8] = d8
						ps643.OverlayValues[9] = d9
						ps643.OverlayValues[34] = d34
						ps643.OverlayValues[35] = d35
						ps643.OverlayValues[36] = d36
						ps643.OverlayValues[37] = d37
						ps643.OverlayValues[38] = d38
						ps643.OverlayValues[39] = d39
						ps643.OverlayValues[40] = d40
						ps643.OverlayValues[41] = d41
						ps643.OverlayValues[42] = d42
						ps643.OverlayValues[43] = d43
						ps643.OverlayValues[44] = d44
						ps643.OverlayValues[45] = d45
						ps643.OverlayValues[46] = d46
						ps643.OverlayValues[47] = d47
						ps643.OverlayValues[48] = d48
						ps643.OverlayValues[49] = d49
						ps643.OverlayValues[50] = d50
						ps643.OverlayValues[51] = d51
						ps643.OverlayValues[52] = d52
						ps643.OverlayValues[53] = d53
						ps643.OverlayValues[54] = d54
						ps643.OverlayValues[55] = d55
						ps643.OverlayValues[56] = d56
						ps643.OverlayValues[127] = d127
						ps643.OverlayValues[128] = d128
						ps643.OverlayValues[129] = d129
						ps643.OverlayValues[130] = d130
						ps643.OverlayValues[131] = d131
						ps643.OverlayValues[133] = d133
						ps643.OverlayValues[134] = d134
						ps643.OverlayValues[135] = d135
						ps643.OverlayValues[136] = d136
						ps643.OverlayValues[137] = d137
						ps643.OverlayValues[138] = d138
						ps643.OverlayValues[139] = d139
						ps643.OverlayValues[140] = d140
						ps643.OverlayValues[141] = d141
						ps643.OverlayValues[143] = d143
						ps643.OverlayValues[144] = d144
						ps643.OverlayValues[145] = d145
						ps643.OverlayValues[146] = d146
						ps643.OverlayValues[147] = d147
						ps643.OverlayValues[256] = d256
						ps643.OverlayValues[257] = d257
						ps643.OverlayValues[258] = d258
						ps643.OverlayValues[373] = d373
						ps643.OverlayValues[374] = d374
						ps643.OverlayValues[375] = d375
						ps643.OverlayValues[376] = d376
						ps643.OverlayValues[379] = d379
						ps643.OverlayValues[502] = d502
						ps643.OverlayValues[503] = d503
						ps643.OverlayValues[504] = d504
						ps643.OverlayValues[635] = d635
						ps643.OverlayValues[636] = d636
						ps643.OverlayValues[637] = d637
						ps643.OverlayValues[638] = d638
						ps643.OverlayValues[639] = d639
						ps643.OverlayValues[640] = d640
						ps643.OverlayValues[641] = d641
						return bbs[13].RenderPS(ps643)
					}
					if !ps.General {
						ps.General = true
						return bbs[9].RenderPS(ps)
					}
					ctx.EmitJump(d641.Condition, lbl13)
					if bbs[13].Rendered {
						ctx.EmitJmp(lbl14)
					}
					ctx.FreeDesc(&d640)
					snap644 := d1
					snap645 := d2
					snap646 := d3
					snap647 := d4
					snap648 := d5
					snap649 := d6
					snap650 := d7
					snap651 := d8
					snap652 := d9
					snap653 := d34
					snap654 := d35
					snap655 := d36
					snap656 := d37
					snap657 := d38
					snap658 := d39
					snap659 := d40
					snap660 := d41
					snap661 := d42
					snap662 := d43
					snap663 := d44
					snap664 := d45
					snap665 := d46
					snap666 := d47
					snap667 := d48
					snap668 := d49
					snap669 := d50
					snap670 := d51
					snap671 := d52
					snap672 := d53
					snap673 := d54
					snap674 := d55
					snap675 := d56
					snap676 := d127
					snap677 := d128
					snap678 := d129
					snap679 := d130
					snap680 := d131
					snap681 := d133
					snap682 := d134
					snap683 := d135
					snap684 := d136
					snap685 := d137
					snap686 := d138
					snap687 := d139
					snap688 := d140
					snap689 := d141
					snap690 := d143
					snap691 := d144
					snap692 := d145
					snap693 := d146
					snap694 := d147
					snap695 := d256
					snap696 := d257
					snap697 := d258
					snap698 := d373
					snap699 := d374
					snap700 := d375
					snap701 := d376
					snap702 := d379
					snap703 := d502
					snap704 := d503
					snap705 := d504
					snap706 := d635
					snap707 := d636
					snap708 := d637
					snap709 := d638
					snap710 := d639
					snap711 := d640
					snap712 := d641
					alloc713 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc713)
					d1 = snap644
					d2 = snap645
					d3 = snap646
					d4 = snap647
					d5 = snap648
					d6 = snap649
					d7 = snap650
					d8 = snap651
					d9 = snap652
					d34 = snap653
					d35 = snap654
					d36 = snap655
					d37 = snap656
					d38 = snap657
					d39 = snap658
					d40 = snap659
					d41 = snap660
					d42 = snap661
					d43 = snap662
					d44 = snap663
					d45 = snap664
					d46 = snap665
					d47 = snap666
					d48 = snap667
					d49 = snap668
					d50 = snap669
					d51 = snap670
					d52 = snap671
					d53 = snap672
					d54 = snap673
					d55 = snap674
					d56 = snap675
					d127 = snap676
					d128 = snap677
					d129 = snap678
					d130 = snap679
					d131 = snap680
					d133 = snap681
					d134 = snap682
					d135 = snap683
					d136 = snap684
					d137 = snap685
					d138 = snap686
					d139 = snap687
					d140 = snap688
					d141 = snap689
					d143 = snap690
					d144 = snap691
					d145 = snap692
					d146 = snap693
					d147 = snap694
					d256 = snap695
					d257 = snap696
					d258 = snap697
					d373 = snap698
					d374 = snap699
					d375 = snap700
					d376 = snap701
					d379 = snap702
					d502 = snap703
					d503 = snap704
					d504 = snap705
					d635 = snap706
					d636 = snap707
					d637 = snap708
					d638 = snap709
					d639 = snap710
					d640 = snap711
					d641 = snap712
					ctx.RestoreAllocState(alloc713)
					d1 = snap644
					d2 = snap645
					d3 = snap646
					d4 = snap647
					d5 = snap648
					d6 = snap649
					d7 = snap650
					d8 = snap651
					d9 = snap652
					d34 = snap653
					d35 = snap654
					d36 = snap655
					d37 = snap656
					d38 = snap657
					d39 = snap658
					d40 = snap659
					d41 = snap660
					d42 = snap661
					d43 = snap662
					d44 = snap663
					d45 = snap664
					d46 = snap665
					d47 = snap666
					d48 = snap667
					d49 = snap668
					d50 = snap669
					d51 = snap670
					d52 = snap671
					d53 = snap672
					d54 = snap673
					d55 = snap674
					d56 = snap675
					d127 = snap676
					d128 = snap677
					d129 = snap678
					d130 = snap679
					d131 = snap680
					d133 = snap681
					d134 = snap682
					d135 = snap683
					d136 = snap684
					d137 = snap685
					d138 = snap686
					d139 = snap687
					d140 = snap688
					d141 = snap689
					d143 = snap690
					d144 = snap691
					d145 = snap692
					d146 = snap693
					d147 = snap694
					d256 = snap695
					d257 = snap696
					d258 = snap697
					d373 = snap698
					d374 = snap699
					d375 = snap700
					d376 = snap701
					d379 = snap702
					d502 = snap703
					d503 = snap704
					d504 = snap705
					d635 = snap706
					d636 = snap707
					d637 = snap708
					d638 = snap709
					d639 = snap710
					d640 = snap711
					d641 = snap712
					ps714 := PhiState{General: true}
					ps714.OverlayValues = make([]JITValueDesc, 642)
					ps714.OverlayValues[1] = d1
					ps714.OverlayValues[2] = d2
					ps714.OverlayValues[3] = d3
					ps714.OverlayValues[4] = d4
					ps714.OverlayValues[5] = d5
					ps714.OverlayValues[6] = d6
					ps714.OverlayValues[7] = d7
					ps714.OverlayValues[8] = d8
					ps714.OverlayValues[9] = d9
					ps714.OverlayValues[34] = d34
					ps714.OverlayValues[35] = d35
					ps714.OverlayValues[36] = d36
					ps714.OverlayValues[37] = d37
					ps714.OverlayValues[38] = d38
					ps714.OverlayValues[39] = d39
					ps714.OverlayValues[40] = d40
					ps714.OverlayValues[41] = d41
					ps714.OverlayValues[42] = d42
					ps714.OverlayValues[43] = d43
					ps714.OverlayValues[44] = d44
					ps714.OverlayValues[45] = d45
					ps714.OverlayValues[46] = d46
					ps714.OverlayValues[47] = d47
					ps714.OverlayValues[48] = d48
					ps714.OverlayValues[49] = d49
					ps714.OverlayValues[50] = d50
					ps714.OverlayValues[51] = d51
					ps714.OverlayValues[52] = d52
					ps714.OverlayValues[53] = d53
					ps714.OverlayValues[54] = d54
					ps714.OverlayValues[55] = d55
					ps714.OverlayValues[56] = d56
					ps714.OverlayValues[127] = d127
					ps714.OverlayValues[128] = d128
					ps714.OverlayValues[129] = d129
					ps714.OverlayValues[130] = d130
					ps714.OverlayValues[131] = d131
					ps714.OverlayValues[133] = d133
					ps714.OverlayValues[134] = d134
					ps714.OverlayValues[135] = d135
					ps714.OverlayValues[136] = d136
					ps714.OverlayValues[137] = d137
					ps714.OverlayValues[138] = d138
					ps714.OverlayValues[139] = d139
					ps714.OverlayValues[140] = d140
					ps714.OverlayValues[141] = d141
					ps714.OverlayValues[143] = d143
					ps714.OverlayValues[144] = d144
					ps714.OverlayValues[145] = d145
					ps714.OverlayValues[146] = d146
					ps714.OverlayValues[147] = d147
					ps714.OverlayValues[256] = d256
					ps714.OverlayValues[257] = d257
					ps714.OverlayValues[258] = d258
					ps714.OverlayValues[373] = d373
					ps714.OverlayValues[374] = d374
					ps714.OverlayValues[375] = d375
					ps714.OverlayValues[376] = d376
					ps714.OverlayValues[379] = d379
					ps714.OverlayValues[502] = d502
					ps714.OverlayValues[503] = d503
					ps714.OverlayValues[504] = d504
					ps714.OverlayValues[635] = d635
					ps714.OverlayValues[636] = d636
					ps714.OverlayValues[637] = d637
					ps714.OverlayValues[638] = d638
					ps714.OverlayValues[639] = d639
					ps714.OverlayValues[640] = d640
					ps714.OverlayValues[641] = d641
					ps715 := PhiState{General: true}
					ps715.OverlayValues = make([]JITValueDesc, 642)
					ps715.OverlayValues[1] = d1
					ps715.OverlayValues[2] = d2
					ps715.OverlayValues[3] = d3
					ps715.OverlayValues[4] = d4
					ps715.OverlayValues[5] = d5
					ps715.OverlayValues[6] = d6
					ps715.OverlayValues[7] = d7
					ps715.OverlayValues[8] = d8
					ps715.OverlayValues[9] = d9
					ps715.OverlayValues[34] = d34
					ps715.OverlayValues[35] = d35
					ps715.OverlayValues[36] = d36
					ps715.OverlayValues[37] = d37
					ps715.OverlayValues[38] = d38
					ps715.OverlayValues[39] = d39
					ps715.OverlayValues[40] = d40
					ps715.OverlayValues[41] = d41
					ps715.OverlayValues[42] = d42
					ps715.OverlayValues[43] = d43
					ps715.OverlayValues[44] = d44
					ps715.OverlayValues[45] = d45
					ps715.OverlayValues[46] = d46
					ps715.OverlayValues[47] = d47
					ps715.OverlayValues[48] = d48
					ps715.OverlayValues[49] = d49
					ps715.OverlayValues[50] = d50
					ps715.OverlayValues[51] = d51
					ps715.OverlayValues[52] = d52
					ps715.OverlayValues[53] = d53
					ps715.OverlayValues[54] = d54
					ps715.OverlayValues[55] = d55
					ps715.OverlayValues[56] = d56
					ps715.OverlayValues[127] = d127
					ps715.OverlayValues[128] = d128
					ps715.OverlayValues[129] = d129
					ps715.OverlayValues[130] = d130
					ps715.OverlayValues[131] = d131
					ps715.OverlayValues[133] = d133
					ps715.OverlayValues[134] = d134
					ps715.OverlayValues[135] = d135
					ps715.OverlayValues[136] = d136
					ps715.OverlayValues[137] = d137
					ps715.OverlayValues[138] = d138
					ps715.OverlayValues[139] = d139
					ps715.OverlayValues[140] = d140
					ps715.OverlayValues[141] = d141
					ps715.OverlayValues[143] = d143
					ps715.OverlayValues[144] = d144
					ps715.OverlayValues[145] = d145
					ps715.OverlayValues[146] = d146
					ps715.OverlayValues[147] = d147
					ps715.OverlayValues[256] = d256
					ps715.OverlayValues[257] = d257
					ps715.OverlayValues[258] = d258
					ps715.OverlayValues[373] = d373
					ps715.OverlayValues[374] = d374
					ps715.OverlayValues[375] = d375
					ps715.OverlayValues[376] = d376
					ps715.OverlayValues[379] = d379
					ps715.OverlayValues[502] = d502
					ps715.OverlayValues[503] = d503
					ps715.OverlayValues[504] = d504
					ps715.OverlayValues[635] = d635
					ps715.OverlayValues[636] = d636
					ps715.OverlayValues[637] = d637
					ps715.OverlayValues[638] = d638
					ps715.OverlayValues[639] = d639
					ps715.OverlayValues[640] = d640
					ps715.OverlayValues[641] = d641
					snap716 := d1
					snap717 := d2
					snap718 := d3
					snap719 := d4
					snap720 := d5
					snap721 := d6
					snap722 := d7
					snap723 := d8
					snap724 := d9
					snap725 := d34
					snap726 := d35
					snap727 := d36
					snap728 := d37
					snap729 := d38
					snap730 := d39
					snap731 := d40
					snap732 := d41
					snap733 := d42
					snap734 := d43
					snap735 := d44
					snap736 := d45
					snap737 := d46
					snap738 := d47
					snap739 := d48
					snap740 := d49
					snap741 := d50
					snap742 := d51
					snap743 := d52
					snap744 := d53
					snap745 := d54
					snap746 := d55
					snap747 := d56
					snap748 := d127
					snap749 := d128
					snap750 := d129
					snap751 := d130
					snap752 := d131
					snap753 := d133
					snap754 := d134
					snap755 := d135
					snap756 := d136
					snap757 := d137
					snap758 := d138
					snap759 := d139
					snap760 := d140
					snap761 := d141
					snap762 := d143
					snap763 := d144
					snap764 := d145
					snap765 := d146
					snap766 := d147
					snap767 := d256
					snap768 := d257
					snap769 := d258
					snap770 := d373
					snap771 := d374
					snap772 := d375
					snap773 := d376
					snap774 := d379
					snap775 := d502
					snap776 := d503
					snap777 := d504
					snap778 := d635
					snap779 := d636
					snap780 := d637
					snap781 := d638
					snap782 := d639
					snap783 := d640
					snap784 := d641
					alloc785 := ctx.SnapshotAllocState()
					if !bbs[13].Rendered {
						bbs[13].RenderPS(ps715)
					}
					ctx.RestoreAllocState(alloc785)
					d1 = snap716
					d2 = snap717
					d3 = snap718
					d4 = snap719
					d5 = snap720
					d6 = snap721
					d7 = snap722
					d8 = snap723
					d9 = snap724
					d34 = snap725
					d35 = snap726
					d36 = snap727
					d37 = snap728
					d38 = snap729
					d39 = snap730
					d40 = snap731
					d41 = snap732
					d42 = snap733
					d43 = snap734
					d44 = snap735
					d45 = snap736
					d46 = snap737
					d47 = snap738
					d48 = snap739
					d49 = snap740
					d50 = snap741
					d51 = snap742
					d52 = snap743
					d53 = snap744
					d54 = snap745
					d55 = snap746
					d56 = snap747
					d127 = snap748
					d128 = snap749
					d129 = snap750
					d130 = snap751
					d131 = snap752
					d133 = snap753
					d134 = snap754
					d135 = snap755
					d136 = snap756
					d137 = snap757
					d138 = snap758
					d139 = snap759
					d140 = snap760
					d141 = snap761
					d143 = snap762
					d144 = snap763
					d145 = snap764
					d146 = snap765
					d147 = snap766
					d256 = snap767
					d257 = snap768
					d258 = snap769
					d373 = snap770
					d374 = snap771
					d375 = snap772
					d376 = snap773
					d379 = snap774
					d502 = snap775
					d503 = snap776
					d504 = snap777
					d635 = snap778
					d636 = snap779
					d637 = snap780
					d638 = snap781
					d639 = snap782
					d640 = snap783
					d641 = snap784
					if !bbs[12].Rendered {
						return bbs[12].RenderPS(ps714)
					}
					return result
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != LocNone {
						d256 = ps.OverlayValues[256]
					}
					if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
						d257 = ps.OverlayValues[257]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != LocNone {
						d373 = ps.OverlayValues[373]
					}
					if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != LocNone {
						d374 = ps.OverlayValues[374]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != LocNone {
						d376 = ps.OverlayValues[376]
					}
					if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != LocNone {
						d379 = ps.OverlayValues[379]
					}
					if len(ps.OverlayValues) > 502 && ps.OverlayValues[502].Loc != LocNone {
						d502 = ps.OverlayValues[502]
					}
					if len(ps.OverlayValues) > 503 && ps.OverlayValues[503].Loc != LocNone {
						d503 = ps.OverlayValues[503]
					}
					if len(ps.OverlayValues) > 504 && ps.OverlayValues[504].Loc != LocNone {
						d504 = ps.OverlayValues[504]
					}
					if len(ps.OverlayValues) > 635 && ps.OverlayValues[635].Loc != LocNone {
						d635 = ps.OverlayValues[635]
					}
					if len(ps.OverlayValues) > 636 && ps.OverlayValues[636].Loc != LocNone {
						d636 = ps.OverlayValues[636]
					}
					if len(ps.OverlayValues) > 637 && ps.OverlayValues[637].Loc != LocNone {
						d637 = ps.OverlayValues[637]
					}
					if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != LocNone {
						d638 = ps.OverlayValues[638]
					}
					if len(ps.OverlayValues) > 639 && ps.OverlayValues[639].Loc != LocNone {
						d639 = ps.OverlayValues[639]
					}
					if len(ps.OverlayValues) > 640 && ps.OverlayValues[640].Loc != LocNone {
						d640 = ps.OverlayValues[640]
					}
					if len(ps.OverlayValues) > 641 && ps.OverlayValues[641].Loc != LocNone {
						d641 = ps.OverlayValues[641]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d374)
					d787 = ctx.EmitSliceElementAddress(&d6, &d374, 16)
					ctx.EnsureDesc(&d787)
					r31 := ctx.AllocRegExcept(d787.Reg)
					ctx.EmitMovRegMem(r31, d787.Reg, 8)
					ctx.EmitMovRegMem(d787.Reg, d787.Reg, 0)
					d786 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d787.Reg, Reg2: r31}
					ctx.BindReg(d787.Reg, &d786)
					ctx.BindReg(r31, &d786)
					ctx.EnsureDesc(&d374)
					ctx.SyncDesc(&d786)
					d788 = d140
					d788.ID = 0
					d789 = d374
					d789.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d788, &d789, &d786, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d374)
						d789 = d374
						d789.ID = 0
						ctx.EmitStoreScmerSliceElement(&d788, &d789, &d786, int32(16))
					}
					ctx.FreeDesc(&d789)
					ctx.FreeDesc(&d786)
					if ps.General {
						ctx.SyncDesc(&d374)
						if d374.Loc == LocReg {
							ctx.ProtectReg(d374.Reg)
						} else if d374.Loc == LocRegPair {
							ctx.ProtectReg(d374.Reg)
							ctx.ProtectReg(d374.Reg2)
						}
						d790 = d374
						if d790.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d790)
						ctx.EmitStoreToStack(d790, int32(bbs[7].PhiBase)+int32(0))
						if d374.Loc == LocReg {
							ctx.UnprotectReg(d374.Reg)
						} else if d374.Loc == LocRegPair {
							ctx.UnprotectReg(d374.Reg)
							ctx.UnprotectReg(d374.Reg2)
						}
					}
					ps791 := PhiState{General: ps.General}
					ps791.OverlayValues = make([]JITValueDesc, 791)
					ps791.OverlayValues[1] = d1
					ps791.OverlayValues[2] = d2
					ps791.OverlayValues[3] = d3
					ps791.OverlayValues[4] = d4
					ps791.OverlayValues[5] = d5
					ps791.OverlayValues[6] = d6
					ps791.OverlayValues[7] = d7
					ps791.OverlayValues[8] = d8
					ps791.OverlayValues[9] = d9
					ps791.OverlayValues[34] = d34
					ps791.OverlayValues[35] = d35
					ps791.OverlayValues[36] = d36
					ps791.OverlayValues[37] = d37
					ps791.OverlayValues[38] = d38
					ps791.OverlayValues[39] = d39
					ps791.OverlayValues[40] = d40
					ps791.OverlayValues[41] = d41
					ps791.OverlayValues[42] = d42
					ps791.OverlayValues[43] = d43
					ps791.OverlayValues[44] = d44
					ps791.OverlayValues[45] = d45
					ps791.OverlayValues[46] = d46
					ps791.OverlayValues[47] = d47
					ps791.OverlayValues[48] = d48
					ps791.OverlayValues[49] = d49
					ps791.OverlayValues[50] = d50
					ps791.OverlayValues[51] = d51
					ps791.OverlayValues[52] = d52
					ps791.OverlayValues[53] = d53
					ps791.OverlayValues[54] = d54
					ps791.OverlayValues[55] = d55
					ps791.OverlayValues[56] = d56
					ps791.OverlayValues[127] = d127
					ps791.OverlayValues[128] = d128
					ps791.OverlayValues[129] = d129
					ps791.OverlayValues[130] = d130
					ps791.OverlayValues[131] = d131
					ps791.OverlayValues[133] = d133
					ps791.OverlayValues[134] = d134
					ps791.OverlayValues[135] = d135
					ps791.OverlayValues[136] = d136
					ps791.OverlayValues[137] = d137
					ps791.OverlayValues[138] = d138
					ps791.OverlayValues[139] = d139
					ps791.OverlayValues[140] = d140
					ps791.OverlayValues[141] = d141
					ps791.OverlayValues[143] = d143
					ps791.OverlayValues[144] = d144
					ps791.OverlayValues[145] = d145
					ps791.OverlayValues[146] = d146
					ps791.OverlayValues[147] = d147
					ps791.OverlayValues[256] = d256
					ps791.OverlayValues[257] = d257
					ps791.OverlayValues[258] = d258
					ps791.OverlayValues[373] = d373
					ps791.OverlayValues[374] = d374
					ps791.OverlayValues[375] = d375
					ps791.OverlayValues[376] = d376
					ps791.OverlayValues[379] = d379
					ps791.OverlayValues[502] = d502
					ps791.OverlayValues[503] = d503
					ps791.OverlayValues[504] = d504
					ps791.OverlayValues[635] = d635
					ps791.OverlayValues[636] = d636
					ps791.OverlayValues[637] = d637
					ps791.OverlayValues[638] = d638
					ps791.OverlayValues[639] = d639
					ps791.OverlayValues[640] = d640
					ps791.OverlayValues[641] = d641
					ps791.OverlayValues[786] = d786
					ps791.OverlayValues[787] = d787
					ps791.OverlayValues[788] = d788
					ps791.OverlayValues[789] = d789
					ps791.OverlayValues[790] = d790
					ps791.PhiValues = make([]JITValueDesc, 1)
					d792 = d374
					ps791.PhiValues[0] = d792
					if ps791.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps791)
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != LocNone {
						d256 = ps.OverlayValues[256]
					}
					if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
						d257 = ps.OverlayValues[257]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != LocNone {
						d373 = ps.OverlayValues[373]
					}
					if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != LocNone {
						d374 = ps.OverlayValues[374]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != LocNone {
						d376 = ps.OverlayValues[376]
					}
					if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != LocNone {
						d379 = ps.OverlayValues[379]
					}
					if len(ps.OverlayValues) > 502 && ps.OverlayValues[502].Loc != LocNone {
						d502 = ps.OverlayValues[502]
					}
					if len(ps.OverlayValues) > 503 && ps.OverlayValues[503].Loc != LocNone {
						d503 = ps.OverlayValues[503]
					}
					if len(ps.OverlayValues) > 504 && ps.OverlayValues[504].Loc != LocNone {
						d504 = ps.OverlayValues[504]
					}
					if len(ps.OverlayValues) > 635 && ps.OverlayValues[635].Loc != LocNone {
						d635 = ps.OverlayValues[635]
					}
					if len(ps.OverlayValues) > 636 && ps.OverlayValues[636].Loc != LocNone {
						d636 = ps.OverlayValues[636]
					}
					if len(ps.OverlayValues) > 637 && ps.OverlayValues[637].Loc != LocNone {
						d637 = ps.OverlayValues[637]
					}
					if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != LocNone {
						d638 = ps.OverlayValues[638]
					}
					if len(ps.OverlayValues) > 639 && ps.OverlayValues[639].Loc != LocNone {
						d639 = ps.OverlayValues[639]
					}
					if len(ps.OverlayValues) > 640 && ps.OverlayValues[640].Loc != LocNone {
						d640 = ps.OverlayValues[640]
					}
					if len(ps.OverlayValues) > 641 && ps.OverlayValues[641].Loc != LocNone {
						d641 = ps.OverlayValues[641]
					}
					if len(ps.OverlayValues) > 786 && ps.OverlayValues[786].Loc != LocNone {
						d786 = ps.OverlayValues[786]
					}
					if len(ps.OverlayValues) > 787 && ps.OverlayValues[787].Loc != LocNone {
						d787 = ps.OverlayValues[787]
					}
					if len(ps.OverlayValues) > 788 && ps.OverlayValues[788].Loc != LocNone {
						d788 = ps.OverlayValues[788]
					}
					if len(ps.OverlayValues) > 789 && ps.OverlayValues[789].Loc != LocNone {
						d789 = ps.OverlayValues[789]
					}
					if len(ps.OverlayValues) > 790 && ps.OverlayValues[790].Loc != LocNone {
						d790 = ps.OverlayValues[790]
					}
					if len(ps.OverlayValues) > 792 && ps.OverlayValues[792].Loc != LocNone {
						d792 = ps.OverlayValues[792]
					}
					ctx.ReclaimUntrackedRegs()
					d793 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d374)
					ctx.SyncDesc(&d793)
					d794 = d140
					d794.ID = 0
					d795 = d374
					d795.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d794, &d795, &d793, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d374)
						d795 = d374
						d795.ID = 0
						ctx.EmitStoreScmerSliceElement(&d794, &d795, &d793, int32(16))
					}
					ctx.FreeDesc(&d795)
					ctx.FreeDesc(&d793)
					if ps.General {
						ctx.SyncDesc(&d374)
						if d374.Loc == LocReg {
							ctx.ProtectReg(d374.Reg)
						} else if d374.Loc == LocRegPair {
							ctx.ProtectReg(d374.Reg)
							ctx.ProtectReg(d374.Reg2)
						}
						d796 = d374
						if d796.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d796)
						ctx.EmitStoreToStack(d796, int32(bbs[7].PhiBase)+int32(0))
						if d374.Loc == LocReg {
							ctx.UnprotectReg(d374.Reg)
						} else if d374.Loc == LocRegPair {
							ctx.UnprotectReg(d374.Reg)
							ctx.UnprotectReg(d374.Reg2)
						}
					}
					ps797 := PhiState{General: ps.General}
					ps797.OverlayValues = make([]JITValueDesc, 797)
					ps797.OverlayValues[1] = d1
					ps797.OverlayValues[2] = d2
					ps797.OverlayValues[3] = d3
					ps797.OverlayValues[4] = d4
					ps797.OverlayValues[5] = d5
					ps797.OverlayValues[6] = d6
					ps797.OverlayValues[7] = d7
					ps797.OverlayValues[8] = d8
					ps797.OverlayValues[9] = d9
					ps797.OverlayValues[34] = d34
					ps797.OverlayValues[35] = d35
					ps797.OverlayValues[36] = d36
					ps797.OverlayValues[37] = d37
					ps797.OverlayValues[38] = d38
					ps797.OverlayValues[39] = d39
					ps797.OverlayValues[40] = d40
					ps797.OverlayValues[41] = d41
					ps797.OverlayValues[42] = d42
					ps797.OverlayValues[43] = d43
					ps797.OverlayValues[44] = d44
					ps797.OverlayValues[45] = d45
					ps797.OverlayValues[46] = d46
					ps797.OverlayValues[47] = d47
					ps797.OverlayValues[48] = d48
					ps797.OverlayValues[49] = d49
					ps797.OverlayValues[50] = d50
					ps797.OverlayValues[51] = d51
					ps797.OverlayValues[52] = d52
					ps797.OverlayValues[53] = d53
					ps797.OverlayValues[54] = d54
					ps797.OverlayValues[55] = d55
					ps797.OverlayValues[56] = d56
					ps797.OverlayValues[127] = d127
					ps797.OverlayValues[128] = d128
					ps797.OverlayValues[129] = d129
					ps797.OverlayValues[130] = d130
					ps797.OverlayValues[131] = d131
					ps797.OverlayValues[133] = d133
					ps797.OverlayValues[134] = d134
					ps797.OverlayValues[135] = d135
					ps797.OverlayValues[136] = d136
					ps797.OverlayValues[137] = d137
					ps797.OverlayValues[138] = d138
					ps797.OverlayValues[139] = d139
					ps797.OverlayValues[140] = d140
					ps797.OverlayValues[141] = d141
					ps797.OverlayValues[143] = d143
					ps797.OverlayValues[144] = d144
					ps797.OverlayValues[145] = d145
					ps797.OverlayValues[146] = d146
					ps797.OverlayValues[147] = d147
					ps797.OverlayValues[256] = d256
					ps797.OverlayValues[257] = d257
					ps797.OverlayValues[258] = d258
					ps797.OverlayValues[373] = d373
					ps797.OverlayValues[374] = d374
					ps797.OverlayValues[375] = d375
					ps797.OverlayValues[376] = d376
					ps797.OverlayValues[379] = d379
					ps797.OverlayValues[502] = d502
					ps797.OverlayValues[503] = d503
					ps797.OverlayValues[504] = d504
					ps797.OverlayValues[635] = d635
					ps797.OverlayValues[636] = d636
					ps797.OverlayValues[637] = d637
					ps797.OverlayValues[638] = d638
					ps797.OverlayValues[639] = d639
					ps797.OverlayValues[640] = d640
					ps797.OverlayValues[641] = d641
					ps797.OverlayValues[786] = d786
					ps797.OverlayValues[787] = d787
					ps797.OverlayValues[788] = d788
					ps797.OverlayValues[789] = d789
					ps797.OverlayValues[790] = d790
					ps797.OverlayValues[792] = d792
					ps797.OverlayValues[793] = d793
					ps797.OverlayValues[794] = d794
					ps797.OverlayValues[795] = d795
					ps797.OverlayValues[796] = d796
					ps797.PhiValues = make([]JITValueDesc, 1)
					d798 = d374
					ps797.PhiValues[0] = d798
					if ps797.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps797)
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != LocNone {
						d256 = ps.OverlayValues[256]
					}
					if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
						d257 = ps.OverlayValues[257]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != LocNone {
						d373 = ps.OverlayValues[373]
					}
					if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != LocNone {
						d374 = ps.OverlayValues[374]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != LocNone {
						d376 = ps.OverlayValues[376]
					}
					if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != LocNone {
						d379 = ps.OverlayValues[379]
					}
					if len(ps.OverlayValues) > 502 && ps.OverlayValues[502].Loc != LocNone {
						d502 = ps.OverlayValues[502]
					}
					if len(ps.OverlayValues) > 503 && ps.OverlayValues[503].Loc != LocNone {
						d503 = ps.OverlayValues[503]
					}
					if len(ps.OverlayValues) > 504 && ps.OverlayValues[504].Loc != LocNone {
						d504 = ps.OverlayValues[504]
					}
					if len(ps.OverlayValues) > 635 && ps.OverlayValues[635].Loc != LocNone {
						d635 = ps.OverlayValues[635]
					}
					if len(ps.OverlayValues) > 636 && ps.OverlayValues[636].Loc != LocNone {
						d636 = ps.OverlayValues[636]
					}
					if len(ps.OverlayValues) > 637 && ps.OverlayValues[637].Loc != LocNone {
						d637 = ps.OverlayValues[637]
					}
					if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != LocNone {
						d638 = ps.OverlayValues[638]
					}
					if len(ps.OverlayValues) > 639 && ps.OverlayValues[639].Loc != LocNone {
						d639 = ps.OverlayValues[639]
					}
					if len(ps.OverlayValues) > 640 && ps.OverlayValues[640].Loc != LocNone {
						d640 = ps.OverlayValues[640]
					}
					if len(ps.OverlayValues) > 641 && ps.OverlayValues[641].Loc != LocNone {
						d641 = ps.OverlayValues[641]
					}
					if len(ps.OverlayValues) > 786 && ps.OverlayValues[786].Loc != LocNone {
						d786 = ps.OverlayValues[786]
					}
					if len(ps.OverlayValues) > 787 && ps.OverlayValues[787].Loc != LocNone {
						d787 = ps.OverlayValues[787]
					}
					if len(ps.OverlayValues) > 788 && ps.OverlayValues[788].Loc != LocNone {
						d788 = ps.OverlayValues[788]
					}
					if len(ps.OverlayValues) > 789 && ps.OverlayValues[789].Loc != LocNone {
						d789 = ps.OverlayValues[789]
					}
					if len(ps.OverlayValues) > 790 && ps.OverlayValues[790].Loc != LocNone {
						d790 = ps.OverlayValues[790]
					}
					if len(ps.OverlayValues) > 792 && ps.OverlayValues[792].Loc != LocNone {
						d792 = ps.OverlayValues[792]
					}
					if len(ps.OverlayValues) > 793 && ps.OverlayValues[793].Loc != LocNone {
						d793 = ps.OverlayValues[793]
					}
					if len(ps.OverlayValues) > 794 && ps.OverlayValues[794].Loc != LocNone {
						d794 = ps.OverlayValues[794]
					}
					if len(ps.OverlayValues) > 795 && ps.OverlayValues[795].Loc != LocNone {
						d795 = ps.OverlayValues[795]
					}
					if len(ps.OverlayValues) > 796 && ps.OverlayValues[796].Loc != LocNone {
						d796 = ps.OverlayValues[796]
					}
					if len(ps.OverlayValues) > 798 && ps.OverlayValues[798].Loc != LocNone {
						d798 = ps.OverlayValues[798]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d37)
					var d799 JITValueDesc
					if d37.Loc == LocImm {
						d799 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d37.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d37.Reg)
						ctx.EmitMovRegReg(scratch, d37.Reg)
						ctx.EmitSubRegImm32(scratch, int32(1))
						d799 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d799)
					}
					if d799.Loc == LocReg && d37.Loc == LocReg && d799.Reg == d37.Reg {
						ctx.TransferReg(d37.Reg)
						d37.Loc = LocNone
					}
					ctx.FreeDesc(&d37)
					ctx.EnsureDesc(&d799)
					ctx.EnsureDesc(&d799)
					ctx.EnsureDesc(&d799)
					d801 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.SyncDesc(&d799)
					d802 = d3
					d802.ID = 0
					d803 = d801
					d803.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d802, &d803, &d799, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d802, &d803, &d799, int32(16))
					}
					ctx.FreeDesc(&d803)
					d804 = args[0]
					d804.ID = 0
					ctx.SyncDesc(&d804)
					if d804.Loc == LocRegPair || d804.Loc == LocStackPair || d804.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d804, &result)
						result.Type = d804.Type
					} else {
						switch d804.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d804)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d804)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d804)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d804, &result)
							result.Type = d804.Type
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != LocNone {
						d256 = ps.OverlayValues[256]
					}
					if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
						d257 = ps.OverlayValues[257]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != LocNone {
						d373 = ps.OverlayValues[373]
					}
					if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != LocNone {
						d374 = ps.OverlayValues[374]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != LocNone {
						d376 = ps.OverlayValues[376]
					}
					if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != LocNone {
						d379 = ps.OverlayValues[379]
					}
					if len(ps.OverlayValues) > 502 && ps.OverlayValues[502].Loc != LocNone {
						d502 = ps.OverlayValues[502]
					}
					if len(ps.OverlayValues) > 503 && ps.OverlayValues[503].Loc != LocNone {
						d503 = ps.OverlayValues[503]
					}
					if len(ps.OverlayValues) > 504 && ps.OverlayValues[504].Loc != LocNone {
						d504 = ps.OverlayValues[504]
					}
					if len(ps.OverlayValues) > 635 && ps.OverlayValues[635].Loc != LocNone {
						d635 = ps.OverlayValues[635]
					}
					if len(ps.OverlayValues) > 636 && ps.OverlayValues[636].Loc != LocNone {
						d636 = ps.OverlayValues[636]
					}
					if len(ps.OverlayValues) > 637 && ps.OverlayValues[637].Loc != LocNone {
						d637 = ps.OverlayValues[637]
					}
					if len(ps.OverlayValues) > 638 && ps.OverlayValues[638].Loc != LocNone {
						d638 = ps.OverlayValues[638]
					}
					if len(ps.OverlayValues) > 639 && ps.OverlayValues[639].Loc != LocNone {
						d639 = ps.OverlayValues[639]
					}
					if len(ps.OverlayValues) > 640 && ps.OverlayValues[640].Loc != LocNone {
						d640 = ps.OverlayValues[640]
					}
					if len(ps.OverlayValues) > 641 && ps.OverlayValues[641].Loc != LocNone {
						d641 = ps.OverlayValues[641]
					}
					if len(ps.OverlayValues) > 786 && ps.OverlayValues[786].Loc != LocNone {
						d786 = ps.OverlayValues[786]
					}
					if len(ps.OverlayValues) > 787 && ps.OverlayValues[787].Loc != LocNone {
						d787 = ps.OverlayValues[787]
					}
					if len(ps.OverlayValues) > 788 && ps.OverlayValues[788].Loc != LocNone {
						d788 = ps.OverlayValues[788]
					}
					if len(ps.OverlayValues) > 789 && ps.OverlayValues[789].Loc != LocNone {
						d789 = ps.OverlayValues[789]
					}
					if len(ps.OverlayValues) > 790 && ps.OverlayValues[790].Loc != LocNone {
						d790 = ps.OverlayValues[790]
					}
					if len(ps.OverlayValues) > 792 && ps.OverlayValues[792].Loc != LocNone {
						d792 = ps.OverlayValues[792]
					}
					if len(ps.OverlayValues) > 793 && ps.OverlayValues[793].Loc != LocNone {
						d793 = ps.OverlayValues[793]
					}
					if len(ps.OverlayValues) > 794 && ps.OverlayValues[794].Loc != LocNone {
						d794 = ps.OverlayValues[794]
					}
					if len(ps.OverlayValues) > 795 && ps.OverlayValues[795].Loc != LocNone {
						d795 = ps.OverlayValues[795]
					}
					if len(ps.OverlayValues) > 796 && ps.OverlayValues[796].Loc != LocNone {
						d796 = ps.OverlayValues[796]
					}
					if len(ps.OverlayValues) > 798 && ps.OverlayValues[798].Loc != LocNone {
						d798 = ps.OverlayValues[798]
					}
					if len(ps.OverlayValues) > 799 && ps.OverlayValues[799].Loc != LocNone {
						d799 = ps.OverlayValues[799]
					}
					if len(ps.OverlayValues) > 800 && ps.OverlayValues[800].Loc != LocNone {
						d800 = ps.OverlayValues[800]
					}
					if len(ps.OverlayValues) > 801 && ps.OverlayValues[801].Loc != LocNone {
						d801 = ps.OverlayValues[801]
					}
					if len(ps.OverlayValues) > 802 && ps.OverlayValues[802].Loc != LocNone {
						d802 = ps.OverlayValues[802]
					}
					if len(ps.OverlayValues) > 803 && ps.OverlayValues[803].Loc != LocNone {
						d803 = ps.OverlayValues[803]
					}
					if len(ps.OverlayValues) > 804 && ps.OverlayValues[804].Loc != LocNone {
						d804 = ps.OverlayValues[804]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d54)
					d805 = d4
					_ = d805
					d806 = d54
					_ = d806
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl15 := ctx.ReserveLabel()
					_ = lbl15
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d805 = JITPrepareScmerGoArg(ctx, d805)
					d806 = JITPrepareGoSliceArg(ctx, d806)
					if d806.Loc != LocRegTriple && d806.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d807 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d807.Loc == LocRegPair || d807.Loc == LocStackPair || d807.Loc == LocRegTriple || d807.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d805)
					ctx.SyncDesc(&d806)
					ctx.SyncDesc(&d807)
					d808 = ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d805, d806, d807}, 2)
					d808.NoHeapPointer = false
					ctx.BindReg(d808.Reg, &d808)
					ctx.BindReg(d808.Reg2, &d808)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d808)
					d809 = args[0]
					d809.ID = 0
					ctx.SyncDesc(&d809)
					if d809.Loc == LocRegPair || d809.Loc == LocStackPair || d809.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d809, &result)
						result.Type = d809.Type
					} else {
						switch d809.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d809)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d809)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d809)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d809, &result)
							result.Type = d809.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps810 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps810)
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
				var d52 JITValueDesc
				_ = d52
				var d108 JITValueDesc
				_ = d108
				var d109 JITValueDesc
				_ = d109
				var d110 JITValueDesc
				_ = d110
				var d111 JITValueDesc
				_ = d111
				var d112 JITValueDesc
				_ = d112
				var d177 JITValueDesc
				_ = d177
				var d178 JITValueDesc
				_ = d178
				var d179 JITValueDesc
				_ = d179
				var d250 JITValueDesc
				_ = d250
				var d251 JITValueDesc
				_ = d251
				var d252 JITValueDesc
				_ = d252
				var d255 JITValueDesc
				_ = d255
				var d332 JITValueDesc
				_ = d332
				var d333 JITValueDesc
				_ = d333
				var d334 JITValueDesc
				_ = d334
				var d335 JITValueDesc
				_ = d335
				var d336 JITValueDesc
				_ = d336
				var d338 JITValueDesc
				_ = d338
				var d339 JITValueDesc
				_ = d339
				var d340 JITValueDesc
				_ = d340
				var d342 JITValueDesc
				_ = d342
				var d343 JITValueDesc
				_ = d343
				var d344 JITValueDesc
				_ = d344
				var d345 JITValueDesc
				_ = d345
				var d346 JITValueDesc
				_ = d346
				var d349 JITValueDesc
				_ = d349
				var d454 JITValueDesc
				_ = d454
				var d455 JITValueDesc
				_ = d455
				var d456 JITValueDesc
				_ = d456
				var d457 JITValueDesc
				_ = d457
				var d459 JITValueDesc
				_ = d459
				var d460 JITValueDesc
				_ = d460
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
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
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
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d10)
					}
					ctx.FreeDesc(&d9)
					d11 = d10
					ctx.EnsureDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
					ctx.EmitJump(d11.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d10)
					snap14 := d1
					snap15 := d2
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
					ctx.RestoreAllocState(alloc25)
					d1 = snap14
					d2 = snap15
					d3 = snap16
					d4 = snap17
					d5 = snap18
					d6 = snap19
					d7 = snap20
					d8 = snap21
					d9 = snap22
					d10 = snap23
					d11 = snap24
					ctx.RestoreAllocState(alloc25)
					d1 = snap14
					d2 = snap15
					d3 = snap16
					d4 = snap17
					d5 = snap18
					d6 = snap19
					d7 = snap20
					d8 = snap21
					d9 = snap22
					d10 = snap23
					d11 = snap24
					ps26 := PhiState{General: true}
					ps26.OverlayValues = make([]JITValueDesc, 12)
					ps26.OverlayValues[1] = d1
					ps26.OverlayValues[2] = d2
					ps26.OverlayValues[3] = d3
					ps26.OverlayValues[4] = d4
					ps26.OverlayValues[5] = d5
					ps26.OverlayValues[6] = d6
					ps26.OverlayValues[7] = d7
					ps26.OverlayValues[8] = d8
					ps26.OverlayValues[9] = d9
					ps26.OverlayValues[10] = d10
					ps26.OverlayValues[11] = d11
					ps27 := PhiState{General: true}
					ps27.OverlayValues = make([]JITValueDesc, 12)
					ps27.OverlayValues[1] = d1
					ps27.OverlayValues[2] = d2
					ps27.OverlayValues[3] = d3
					ps27.OverlayValues[4] = d4
					ps27.OverlayValues[5] = d5
					ps27.OverlayValues[6] = d6
					ps27.OverlayValues[7] = d7
					ps27.OverlayValues[8] = d8
					ps27.OverlayValues[9] = d9
					ps27.OverlayValues[10] = d10
					ps27.OverlayValues[11] = d11
					snap28 := d1
					snap29 := d2
					snap30 := d3
					snap31 := d4
					snap32 := d5
					snap33 := d6
					snap34 := d7
					snap35 := d8
					snap36 := d9
					snap37 := d10
					snap38 := d11
					alloc39 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps27)
					}
					ctx.RestoreAllocState(alloc39)
					d1 = snap28
					d2 = snap29
					d3 = snap30
					d4 = snap31
					d5 = snap32
					d6 = snap33
					d7 = snap34
					d8 = snap35
					d9 = snap36
					d10 = snap37
					d11 = snap38
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps26)
					}
					return result
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
					d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					d42 = ctx.EmitSliceElementAddress(&d4, &d40, 16)
					ctx.EnsureDesc(&d42)
					r1 := ctx.AllocRegExcept(d42.Reg)
					ctx.EmitMovRegMem(r1, d42.Reg, 8)
					ctx.EmitMovRegMem(d42.Reg, d42.Reg, 0)
					d41 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d42.Reg, Reg2: r1}
					ctx.BindReg(d42.Reg, &d41)
					ctx.BindReg(r1, &d41)
					var d43 JITValueDesc
					if d41.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d41.Imm.Int())}
					} else if d41.Type == tagInt && d41.Loc == LocRegPair {
						ctx.FreeReg(d41.Reg)
						d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg2}
						ctx.BindReg(d41.Reg2, &d43)
						ctx.BindReg(d41.Reg2, &d43)
					} else if d41.Type == tagInt && d41.Loc == LocReg {
						d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg}
						ctx.BindReg(d41.Reg, &d43)
						ctx.BindReg(d41.Reg, &d43)
					} else {
						d43 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d41}, 1)
						d43.Type = tagInt
						ctx.BindReg(d43.Reg, &d43)
					}
					ctx.FreeDesc(&d41)
					ctx.EnsureDesc(&d43)
					ctx.EnsureDesc(&d43)
					ctx.StabilizeDescForControlFlow(&d43)
					d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(3)}
					var d46 JITValueDesc
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
						d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2}
						ctx.BindReg(d4.Reg2, &d46)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d45)
					ctx.EnsureDesc(&d46)
					var d48 JITValueDesc
					if d46.Loc == LocImm && d45.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d46.Imm.Int() - d45.Imm.Int())}
					} else {
						r2 := ctx.AllocReg()
						if d46.Loc == LocImm {
							ctx.EmitMovRegImm64(r2, uint64(d46.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r2, d46.Reg)
						}
						if d45.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d45.Imm.Int()))
							ctx.EmitSubInt64(r2, RegR11)
						} else {
							ctx.EmitSubInt64(r2, d45.Reg)
						}
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d48)
					}
					var d49 JITValueDesc
					r3 := ctx.EmitSliceDataAfterLow(&d4, &d45, 16)
					d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
					ctx.BindReg(r3, &d49)
					ctx.BindReg(r3, &d49)
					var d50 JITValueDesc
					var r4 Reg
					var r5 Reg
					ctx.SyncDesc(&d49)
					ctx.EnsureDesc(&d49)
					if d49.Loc == LocImm {
						r4 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r4, uint64(d49.Imm.Int()))
					} else {
						r4 = d49.Reg
					}
					ctx.ProtectReg(r4)
					ctx.SyncDesc(&d48)
					ctx.EnsureDesc(&d48)
					if d48.Loc == LocImm {
						r5 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r5, uint64(d48.Imm.Int()))
					} else {
						r5 = d48.Reg
					}
					ctx.ProtectReg(r5)
					r6 := ctx.EmitSliceCapAfterLow(&d4, &d45, r4, r5)
					ctx.UnprotectReg(r5)
					ctx.UnprotectReg(r4)
					d50 = JITValueDesc{Loc: LocRegTriple, Reg: r4, Reg2: r5, Reg3: r6}
					ctx.BindReg(r4, &d50)
					ctx.BindReg(r5, &d50)
					ctx.BindReg(r6, &d50)
					ctx.BindReg(r4, &d50)
					ctx.BindReg(r5, &d50)
					ctx.BindReg(r6, &d50)
					ctx.StabilizeDescForControlFlow(&d50)
					ctx.EnsureDesc(&d43)
					var d51 JITValueDesc
					if d43.Loc == LocImm {
						d51 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d43.Imm.Int() <= 0)}
					} else {
						r7 := ctx.AllocRegExcept(d43.Reg)
						ctx.EmitCmpRegImm32(d43.Reg, 0)
						d51 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondSignedLessOrEqual}
						ctx.BindReg(r7, &d51)
					}
					d52 = d51
					ctx.EnsureDesc(&d52)
					if d52.Loc != LocImm && d52.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d52.Loc == LocImm {
						if d52.Imm.Bool() {
							if ps.General {
							}
							ps53 := PhiState{General: ps.General}
							ps53.OverlayValues = make([]JITValueDesc, 53)
							ps53.OverlayValues[1] = d1
							ps53.OverlayValues[2] = d2
							ps53.OverlayValues[3] = d3
							ps53.OverlayValues[4] = d4
							ps53.OverlayValues[5] = d5
							ps53.OverlayValues[6] = d6
							ps53.OverlayValues[7] = d7
							ps53.OverlayValues[8] = d8
							ps53.OverlayValues[9] = d9
							ps53.OverlayValues[10] = d10
							ps53.OverlayValues[11] = d11
							ps53.OverlayValues[40] = d40
							ps53.OverlayValues[41] = d41
							ps53.OverlayValues[42] = d42
							ps53.OverlayValues[43] = d43
							ps53.OverlayValues[44] = d44
							ps53.OverlayValues[45] = d45
							ps53.OverlayValues[46] = d46
							ps53.OverlayValues[47] = d47
							ps53.OverlayValues[48] = d48
							ps53.OverlayValues[49] = d49
							ps53.OverlayValues[50] = d50
							ps53.OverlayValues[51] = d51
							ps53.OverlayValues[52] = d52
							return bbs[3].RenderPS(ps53)
						}
						if ps.General {
						}
						ps54 := PhiState{General: ps.General}
						ps54.OverlayValues = make([]JITValueDesc, 53)
						ps54.OverlayValues[1] = d1
						ps54.OverlayValues[2] = d2
						ps54.OverlayValues[3] = d3
						ps54.OverlayValues[4] = d4
						ps54.OverlayValues[5] = d5
						ps54.OverlayValues[6] = d6
						ps54.OverlayValues[7] = d7
						ps54.OverlayValues[8] = d8
						ps54.OverlayValues[9] = d9
						ps54.OverlayValues[10] = d10
						ps54.OverlayValues[11] = d11
						ps54.OverlayValues[40] = d40
						ps54.OverlayValues[41] = d41
						ps54.OverlayValues[42] = d42
						ps54.OverlayValues[43] = d43
						ps54.OverlayValues[44] = d44
						ps54.OverlayValues[45] = d45
						ps54.OverlayValues[46] = d46
						ps54.OverlayValues[47] = d47
						ps54.OverlayValues[48] = d48
						ps54.OverlayValues[49] = d49
						ps54.OverlayValues[50] = d50
						ps54.OverlayValues[51] = d51
						ps54.OverlayValues[52] = d52
						return bbs[6].RenderPS(ps54)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					ctx.EmitJump(d52.Condition, lbl4)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d51)
					snap55 := d1
					snap56 := d2
					snap57 := d3
					snap58 := d4
					snap59 := d5
					snap60 := d6
					snap61 := d7
					snap62 := d8
					snap63 := d9
					snap64 := d10
					snap65 := d11
					snap66 := d40
					snap67 := d41
					snap68 := d42
					snap69 := d43
					snap70 := d44
					snap71 := d45
					snap72 := d46
					snap73 := d47
					snap74 := d48
					snap75 := d49
					snap76 := d50
					snap77 := d51
					snap78 := d52
					alloc79 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc79)
					d1 = snap55
					d2 = snap56
					d3 = snap57
					d4 = snap58
					d5 = snap59
					d6 = snap60
					d7 = snap61
					d8 = snap62
					d9 = snap63
					d10 = snap64
					d11 = snap65
					d40 = snap66
					d41 = snap67
					d42 = snap68
					d43 = snap69
					d44 = snap70
					d45 = snap71
					d46 = snap72
					d47 = snap73
					d48 = snap74
					d49 = snap75
					d50 = snap76
					d51 = snap77
					d52 = snap78
					ctx.RestoreAllocState(alloc79)
					d1 = snap55
					d2 = snap56
					d3 = snap57
					d4 = snap58
					d5 = snap59
					d6 = snap60
					d7 = snap61
					d8 = snap62
					d9 = snap63
					d10 = snap64
					d11 = snap65
					d40 = snap66
					d41 = snap67
					d42 = snap68
					d43 = snap69
					d44 = snap70
					d45 = snap71
					d46 = snap72
					d47 = snap73
					d48 = snap74
					d49 = snap75
					d50 = snap76
					d51 = snap77
					d52 = snap78
					ps80 := PhiState{General: true}
					ps80.OverlayValues = make([]JITValueDesc, 53)
					ps80.OverlayValues[1] = d1
					ps80.OverlayValues[2] = d2
					ps80.OverlayValues[3] = d3
					ps80.OverlayValues[4] = d4
					ps80.OverlayValues[5] = d5
					ps80.OverlayValues[6] = d6
					ps80.OverlayValues[7] = d7
					ps80.OverlayValues[8] = d8
					ps80.OverlayValues[9] = d9
					ps80.OverlayValues[10] = d10
					ps80.OverlayValues[11] = d11
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
					ps81 := PhiState{General: true}
					ps81.OverlayValues = make([]JITValueDesc, 53)
					ps81.OverlayValues[1] = d1
					ps81.OverlayValues[2] = d2
					ps81.OverlayValues[3] = d3
					ps81.OverlayValues[4] = d4
					ps81.OverlayValues[5] = d5
					ps81.OverlayValues[6] = d6
					ps81.OverlayValues[7] = d7
					ps81.OverlayValues[8] = d8
					ps81.OverlayValues[9] = d9
					ps81.OverlayValues[10] = d10
					ps81.OverlayValues[11] = d11
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
					snap82 := d1
					snap83 := d2
					snap84 := d3
					snap85 := d4
					snap86 := d5
					snap87 := d6
					snap88 := d7
					snap89 := d8
					snap90 := d9
					snap91 := d10
					snap92 := d11
					snap93 := d40
					snap94 := d41
					snap95 := d42
					snap96 := d43
					snap97 := d44
					snap98 := d45
					snap99 := d46
					snap100 := d47
					snap101 := d48
					snap102 := d49
					snap103 := d50
					snap104 := d51
					snap105 := d52
					alloc106 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps81)
					}
					ctx.RestoreAllocState(alloc106)
					d1 = snap82
					d2 = snap83
					d3 = snap84
					d4 = snap85
					d5 = snap86
					d6 = snap87
					d7 = snap88
					d8 = snap89
					d9 = snap90
					d10 = snap91
					d11 = snap92
					d40 = snap93
					d41 = snap94
					d42 = snap95
					d43 = snap96
					d44 = snap97
					d45 = snap98
					d46 = snap99
					d47 = snap100
					d48 = snap101
					d49 = snap102
					d50 = snap103
					d51 = snap104
					d52 = snap105
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps80)
					}
					return result
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[7].PhiBase)+int32(0))
					}
					ps107 := PhiState{General: ps.General}
					ps107.OverlayValues = make([]JITValueDesc, 53)
					ps107.OverlayValues[1] = d1
					ps107.OverlayValues[2] = d2
					ps107.OverlayValues[3] = d3
					ps107.OverlayValues[4] = d4
					ps107.OverlayValues[5] = d5
					ps107.OverlayValues[6] = d6
					ps107.OverlayValues[7] = d7
					ps107.OverlayValues[8] = d8
					ps107.OverlayValues[9] = d9
					ps107.OverlayValues[10] = d10
					ps107.OverlayValues[11] = d11
					ps107.OverlayValues[40] = d40
					ps107.OverlayValues[41] = d41
					ps107.OverlayValues[42] = d42
					ps107.OverlayValues[43] = d43
					ps107.OverlayValues[44] = d44
					ps107.OverlayValues[45] = d45
					ps107.OverlayValues[46] = d46
					ps107.OverlayValues[47] = d47
					ps107.OverlayValues[48] = d48
					ps107.OverlayValues[49] = d49
					ps107.OverlayValues[50] = d50
					ps107.OverlayValues[51] = d51
					ps107.OverlayValues[52] = d52
					ps107.PhiValues = make([]JITValueDesc, 1)
					d108 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps107.PhiValues[0] = d108
					if ps107.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps107)
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					ctx.ReclaimUntrackedRegs()
					var d109 JITValueDesc
					if d50.SliceSizeKnown {
						d109 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d50.KnownSliceLen))}
					} else if d50.Loc == LocImm {
						d109 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d50.StackOff))}
					} else if d50.Loc == LocStackTriple {
						d109 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d50.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d50)
						if d50.Loc == LocRegPair || d50.Loc == LocRegTriple {
							d109 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d50.Reg2, ID: 0}
						} else if d50.Loc == LocReg {
							d109 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d50.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d109)
					ctx.EnsureDesc(&d43)
					ctx.EnsureDescsTogether(&d109, &d43)
					var d110 JITValueDesc
					if d109.Loc == LocImm && d43.Loc == LocImm {
						d110 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d109.Imm.Int() % d43.Imm.Int())}
					} else {
						d110 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntRem), []JITValueDesc{d109, d43}, 1)
					}
					if d110.Loc == LocReg && d109.Loc == LocReg && d110.Reg == d109.Reg {
						ctx.TransferReg(d109.Reg)
						d109.Loc = LocNone
					}
					ctx.FreeDesc(&d109)
					ctx.EnsureDesc(&d110)
					var d111 JITValueDesc
					if d110.Loc == LocImm {
						d111 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d110.Imm.Int() != 0)}
					} else {
						r8 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d110.Reg, 0)
						d111 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondNotEqual}
						ctx.BindReg(r8, &d111)
					}
					ctx.FreeDesc(&d110)
					d112 = d111
					ctx.EnsureDesc(&d112)
					if d112.Loc != LocImm && d112.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
							ps113.OverlayValues[40] = d40
							ps113.OverlayValues[41] = d41
							ps113.OverlayValues[42] = d42
							ps113.OverlayValues[43] = d43
							ps113.OverlayValues[44] = d44
							ps113.OverlayValues[45] = d45
							ps113.OverlayValues[46] = d46
							ps113.OverlayValues[47] = d47
							ps113.OverlayValues[48] = d48
							ps113.OverlayValues[49] = d49
							ps113.OverlayValues[50] = d50
							ps113.OverlayValues[51] = d51
							ps113.OverlayValues[52] = d52
							ps113.OverlayValues[108] = d108
							ps113.OverlayValues[109] = d109
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
						ps114.OverlayValues[40] = d40
						ps114.OverlayValues[41] = d41
						ps114.OverlayValues[42] = d42
						ps114.OverlayValues[43] = d43
						ps114.OverlayValues[44] = d44
						ps114.OverlayValues[45] = d45
						ps114.OverlayValues[46] = d46
						ps114.OverlayValues[47] = d47
						ps114.OverlayValues[48] = d48
						ps114.OverlayValues[49] = d49
						ps114.OverlayValues[50] = d50
						ps114.OverlayValues[51] = d51
						ps114.OverlayValues[52] = d52
						ps114.OverlayValues[108] = d108
						ps114.OverlayValues[109] = d109
						ps114.OverlayValues[110] = d110
						ps114.OverlayValues[111] = d111
						ps114.OverlayValues[112] = d112
						return bbs[4].RenderPS(ps114)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					ctx.EmitJump(d112.Condition, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d111)
					snap115 := d1
					snap116 := d2
					snap117 := d3
					snap118 := d4
					snap119 := d5
					snap120 := d6
					snap121 := d7
					snap122 := d8
					snap123 := d9
					snap124 := d10
					snap125 := d11
					snap126 := d40
					snap127 := d41
					snap128 := d42
					snap129 := d43
					snap130 := d44
					snap131 := d45
					snap132 := d46
					snap133 := d47
					snap134 := d48
					snap135 := d49
					snap136 := d50
					snap137 := d51
					snap138 := d52
					snap139 := d108
					snap140 := d109
					snap141 := d110
					snap142 := d111
					snap143 := d112
					alloc144 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc144)
					d1 = snap115
					d2 = snap116
					d3 = snap117
					d4 = snap118
					d5 = snap119
					d6 = snap120
					d7 = snap121
					d8 = snap122
					d9 = snap123
					d10 = snap124
					d11 = snap125
					d40 = snap126
					d41 = snap127
					d42 = snap128
					d43 = snap129
					d44 = snap130
					d45 = snap131
					d46 = snap132
					d47 = snap133
					d48 = snap134
					d49 = snap135
					d50 = snap136
					d51 = snap137
					d52 = snap138
					d108 = snap139
					d109 = snap140
					d110 = snap141
					d111 = snap142
					d112 = snap143
					ctx.RestoreAllocState(alloc144)
					d1 = snap115
					d2 = snap116
					d3 = snap117
					d4 = snap118
					d5 = snap119
					d6 = snap120
					d7 = snap121
					d8 = snap122
					d9 = snap123
					d10 = snap124
					d11 = snap125
					d40 = snap126
					d41 = snap127
					d42 = snap128
					d43 = snap129
					d44 = snap130
					d45 = snap131
					d46 = snap132
					d47 = snap133
					d48 = snap134
					d49 = snap135
					d50 = snap136
					d51 = snap137
					d52 = snap138
					d108 = snap139
					d109 = snap140
					d110 = snap141
					d111 = snap142
					d112 = snap143
					ps145 := PhiState{General: true}
					ps145.OverlayValues = make([]JITValueDesc, 113)
					ps145.OverlayValues[1] = d1
					ps145.OverlayValues[2] = d2
					ps145.OverlayValues[3] = d3
					ps145.OverlayValues[4] = d4
					ps145.OverlayValues[5] = d5
					ps145.OverlayValues[6] = d6
					ps145.OverlayValues[7] = d7
					ps145.OverlayValues[8] = d8
					ps145.OverlayValues[9] = d9
					ps145.OverlayValues[10] = d10
					ps145.OverlayValues[11] = d11
					ps145.OverlayValues[40] = d40
					ps145.OverlayValues[41] = d41
					ps145.OverlayValues[42] = d42
					ps145.OverlayValues[43] = d43
					ps145.OverlayValues[44] = d44
					ps145.OverlayValues[45] = d45
					ps145.OverlayValues[46] = d46
					ps145.OverlayValues[47] = d47
					ps145.OverlayValues[48] = d48
					ps145.OverlayValues[49] = d49
					ps145.OverlayValues[50] = d50
					ps145.OverlayValues[51] = d51
					ps145.OverlayValues[52] = d52
					ps145.OverlayValues[108] = d108
					ps145.OverlayValues[109] = d109
					ps145.OverlayValues[110] = d110
					ps145.OverlayValues[111] = d111
					ps145.OverlayValues[112] = d112
					ps146 := PhiState{General: true}
					ps146.OverlayValues = make([]JITValueDesc, 113)
					ps146.OverlayValues[1] = d1
					ps146.OverlayValues[2] = d2
					ps146.OverlayValues[3] = d3
					ps146.OverlayValues[4] = d4
					ps146.OverlayValues[5] = d5
					ps146.OverlayValues[6] = d6
					ps146.OverlayValues[7] = d7
					ps146.OverlayValues[8] = d8
					ps146.OverlayValues[9] = d9
					ps146.OverlayValues[10] = d10
					ps146.OverlayValues[11] = d11
					ps146.OverlayValues[40] = d40
					ps146.OverlayValues[41] = d41
					ps146.OverlayValues[42] = d42
					ps146.OverlayValues[43] = d43
					ps146.OverlayValues[44] = d44
					ps146.OverlayValues[45] = d45
					ps146.OverlayValues[46] = d46
					ps146.OverlayValues[47] = d47
					ps146.OverlayValues[48] = d48
					ps146.OverlayValues[49] = d49
					ps146.OverlayValues[50] = d50
					ps146.OverlayValues[51] = d51
					ps146.OverlayValues[52] = d52
					ps146.OverlayValues[108] = d108
					ps146.OverlayValues[109] = d109
					ps146.OverlayValues[110] = d110
					ps146.OverlayValues[111] = d111
					ps146.OverlayValues[112] = d112
					snap147 := d1
					snap148 := d2
					snap149 := d3
					snap150 := d4
					snap151 := d5
					snap152 := d6
					snap153 := d7
					snap154 := d8
					snap155 := d9
					snap156 := d10
					snap157 := d11
					snap158 := d40
					snap159 := d41
					snap160 := d42
					snap161 := d43
					snap162 := d44
					snap163 := d45
					snap164 := d46
					snap165 := d47
					snap166 := d48
					snap167 := d49
					snap168 := d50
					snap169 := d51
					snap170 := d52
					snap171 := d108
					snap172 := d109
					snap173 := d110
					snap174 := d111
					snap175 := d112
					alloc176 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps146)
					}
					ctx.RestoreAllocState(alloc176)
					d1 = snap147
					d2 = snap148
					d3 = snap149
					d4 = snap150
					d5 = snap151
					d6 = snap152
					d7 = snap153
					d8 = snap154
					d9 = snap155
					d10 = snap156
					d11 = snap157
					d40 = snap158
					d41 = snap159
					d42 = snap160
					d43 = snap161
					d44 = snap162
					d45 = snap163
					d46 = snap164
					d47 = snap165
					d48 = snap166
					d49 = snap167
					d50 = snap168
					d51 = snap169
					d52 = snap170
					d108 = snap171
					d109 = snap172
					d110 = snap173
					d111 = snap174
					d112 = snap175
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps145)
					}
					return result
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					ctx.ReclaimUntrackedRegs()
					var d177 JITValueDesc
					if d50.SliceSizeKnown {
						d177 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d50.KnownSliceLen))}
					} else if d50.Loc == LocImm {
						d177 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d50.StackOff))}
					} else if d50.Loc == LocStackTriple {
						d177 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d50.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d50)
						if d50.Loc == LocRegPair || d50.Loc == LocRegTriple {
							d177 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d50.Reg2, ID: 0}
						} else if d50.Loc == LocReg {
							d177 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d50.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d177)
					var d178 JITValueDesc
					if d177.Loc == LocImm {
						d178 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d177.Imm.Int() == 0)}
					} else {
						r9 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d177.Reg, 0)
						d178 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondEqual}
						ctx.BindReg(r9, &d178)
					}
					ctx.FreeDesc(&d177)
					d179 = d178
					ctx.EnsureDesc(&d179)
					if d179.Loc != LocImm && d179.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d179.Loc == LocImm {
						if d179.Imm.Bool() {
							if ps.General {
							}
							ps180 := PhiState{General: ps.General}
							ps180.OverlayValues = make([]JITValueDesc, 180)
							ps180.OverlayValues[1] = d1
							ps180.OverlayValues[2] = d2
							ps180.OverlayValues[3] = d3
							ps180.OverlayValues[4] = d4
							ps180.OverlayValues[5] = d5
							ps180.OverlayValues[6] = d6
							ps180.OverlayValues[7] = d7
							ps180.OverlayValues[8] = d8
							ps180.OverlayValues[9] = d9
							ps180.OverlayValues[10] = d10
							ps180.OverlayValues[11] = d11
							ps180.OverlayValues[40] = d40
							ps180.OverlayValues[41] = d41
							ps180.OverlayValues[42] = d42
							ps180.OverlayValues[43] = d43
							ps180.OverlayValues[44] = d44
							ps180.OverlayValues[45] = d45
							ps180.OverlayValues[46] = d46
							ps180.OverlayValues[47] = d47
							ps180.OverlayValues[48] = d48
							ps180.OverlayValues[49] = d49
							ps180.OverlayValues[50] = d50
							ps180.OverlayValues[51] = d51
							ps180.OverlayValues[52] = d52
							ps180.OverlayValues[108] = d108
							ps180.OverlayValues[109] = d109
							ps180.OverlayValues[110] = d110
							ps180.OverlayValues[111] = d111
							ps180.OverlayValues[112] = d112
							ps180.OverlayValues[177] = d177
							ps180.OverlayValues[178] = d178
							ps180.OverlayValues[179] = d179
							return bbs[3].RenderPS(ps180)
						}
						if ps.General {
						}
						ps181 := PhiState{General: ps.General}
						ps181.OverlayValues = make([]JITValueDesc, 180)
						ps181.OverlayValues[1] = d1
						ps181.OverlayValues[2] = d2
						ps181.OverlayValues[3] = d3
						ps181.OverlayValues[4] = d4
						ps181.OverlayValues[5] = d5
						ps181.OverlayValues[6] = d6
						ps181.OverlayValues[7] = d7
						ps181.OverlayValues[8] = d8
						ps181.OverlayValues[9] = d9
						ps181.OverlayValues[10] = d10
						ps181.OverlayValues[11] = d11
						ps181.OverlayValues[40] = d40
						ps181.OverlayValues[41] = d41
						ps181.OverlayValues[42] = d42
						ps181.OverlayValues[43] = d43
						ps181.OverlayValues[44] = d44
						ps181.OverlayValues[45] = d45
						ps181.OverlayValues[46] = d46
						ps181.OverlayValues[47] = d47
						ps181.OverlayValues[48] = d48
						ps181.OverlayValues[49] = d49
						ps181.OverlayValues[50] = d50
						ps181.OverlayValues[51] = d51
						ps181.OverlayValues[52] = d52
						ps181.OverlayValues[108] = d108
						ps181.OverlayValues[109] = d109
						ps181.OverlayValues[110] = d110
						ps181.OverlayValues[111] = d111
						ps181.OverlayValues[112] = d112
						ps181.OverlayValues[177] = d177
						ps181.OverlayValues[178] = d178
						ps181.OverlayValues[179] = d179
						return bbs[5].RenderPS(ps181)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					ctx.EmitJump(d179.Condition, lbl4)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d178)
					snap182 := d1
					snap183 := d2
					snap184 := d3
					snap185 := d4
					snap186 := d5
					snap187 := d6
					snap188 := d7
					snap189 := d8
					snap190 := d9
					snap191 := d10
					snap192 := d11
					snap193 := d40
					snap194 := d41
					snap195 := d42
					snap196 := d43
					snap197 := d44
					snap198 := d45
					snap199 := d46
					snap200 := d47
					snap201 := d48
					snap202 := d49
					snap203 := d50
					snap204 := d51
					snap205 := d52
					snap206 := d108
					snap207 := d109
					snap208 := d110
					snap209 := d111
					snap210 := d112
					snap211 := d177
					snap212 := d178
					snap213 := d179
					alloc214 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc214)
					d1 = snap182
					d2 = snap183
					d3 = snap184
					d4 = snap185
					d5 = snap186
					d6 = snap187
					d7 = snap188
					d8 = snap189
					d9 = snap190
					d10 = snap191
					d11 = snap192
					d40 = snap193
					d41 = snap194
					d42 = snap195
					d43 = snap196
					d44 = snap197
					d45 = snap198
					d46 = snap199
					d47 = snap200
					d48 = snap201
					d49 = snap202
					d50 = snap203
					d51 = snap204
					d52 = snap205
					d108 = snap206
					d109 = snap207
					d110 = snap208
					d111 = snap209
					d112 = snap210
					d177 = snap211
					d178 = snap212
					d179 = snap213
					ctx.RestoreAllocState(alloc214)
					d1 = snap182
					d2 = snap183
					d3 = snap184
					d4 = snap185
					d5 = snap186
					d6 = snap187
					d7 = snap188
					d8 = snap189
					d9 = snap190
					d10 = snap191
					d11 = snap192
					d40 = snap193
					d41 = snap194
					d42 = snap195
					d43 = snap196
					d44 = snap197
					d45 = snap198
					d46 = snap199
					d47 = snap200
					d48 = snap201
					d49 = snap202
					d50 = snap203
					d51 = snap204
					d52 = snap205
					d108 = snap206
					d109 = snap207
					d110 = snap208
					d111 = snap209
					d112 = snap210
					d177 = snap211
					d178 = snap212
					d179 = snap213
					ps215 := PhiState{General: true}
					ps215.OverlayValues = make([]JITValueDesc, 180)
					ps215.OverlayValues[1] = d1
					ps215.OverlayValues[2] = d2
					ps215.OverlayValues[3] = d3
					ps215.OverlayValues[4] = d4
					ps215.OverlayValues[5] = d5
					ps215.OverlayValues[6] = d6
					ps215.OverlayValues[7] = d7
					ps215.OverlayValues[8] = d8
					ps215.OverlayValues[9] = d9
					ps215.OverlayValues[10] = d10
					ps215.OverlayValues[11] = d11
					ps215.OverlayValues[40] = d40
					ps215.OverlayValues[41] = d41
					ps215.OverlayValues[42] = d42
					ps215.OverlayValues[43] = d43
					ps215.OverlayValues[44] = d44
					ps215.OverlayValues[45] = d45
					ps215.OverlayValues[46] = d46
					ps215.OverlayValues[47] = d47
					ps215.OverlayValues[48] = d48
					ps215.OverlayValues[49] = d49
					ps215.OverlayValues[50] = d50
					ps215.OverlayValues[51] = d51
					ps215.OverlayValues[52] = d52
					ps215.OverlayValues[108] = d108
					ps215.OverlayValues[109] = d109
					ps215.OverlayValues[110] = d110
					ps215.OverlayValues[111] = d111
					ps215.OverlayValues[112] = d112
					ps215.OverlayValues[177] = d177
					ps215.OverlayValues[178] = d178
					ps215.OverlayValues[179] = d179
					ps216 := PhiState{General: true}
					ps216.OverlayValues = make([]JITValueDesc, 180)
					ps216.OverlayValues[1] = d1
					ps216.OverlayValues[2] = d2
					ps216.OverlayValues[3] = d3
					ps216.OverlayValues[4] = d4
					ps216.OverlayValues[5] = d5
					ps216.OverlayValues[6] = d6
					ps216.OverlayValues[7] = d7
					ps216.OverlayValues[8] = d8
					ps216.OverlayValues[9] = d9
					ps216.OverlayValues[10] = d10
					ps216.OverlayValues[11] = d11
					ps216.OverlayValues[40] = d40
					ps216.OverlayValues[41] = d41
					ps216.OverlayValues[42] = d42
					ps216.OverlayValues[43] = d43
					ps216.OverlayValues[44] = d44
					ps216.OverlayValues[45] = d45
					ps216.OverlayValues[46] = d46
					ps216.OverlayValues[47] = d47
					ps216.OverlayValues[48] = d48
					ps216.OverlayValues[49] = d49
					ps216.OverlayValues[50] = d50
					ps216.OverlayValues[51] = d51
					ps216.OverlayValues[52] = d52
					ps216.OverlayValues[108] = d108
					ps216.OverlayValues[109] = d109
					ps216.OverlayValues[110] = d110
					ps216.OverlayValues[111] = d111
					ps216.OverlayValues[112] = d112
					ps216.OverlayValues[177] = d177
					ps216.OverlayValues[178] = d178
					ps216.OverlayValues[179] = d179
					snap217 := d1
					snap218 := d2
					snap219 := d3
					snap220 := d4
					snap221 := d5
					snap222 := d6
					snap223 := d7
					snap224 := d8
					snap225 := d9
					snap226 := d10
					snap227 := d11
					snap228 := d40
					snap229 := d41
					snap230 := d42
					snap231 := d43
					snap232 := d44
					snap233 := d45
					snap234 := d46
					snap235 := d47
					snap236 := d48
					snap237 := d49
					snap238 := d50
					snap239 := d51
					snap240 := d52
					snap241 := d108
					snap242 := d109
					snap243 := d110
					snap244 := d111
					snap245 := d112
					snap246 := d177
					snap247 := d178
					snap248 := d179
					alloc249 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps216)
					}
					ctx.RestoreAllocState(alloc249)
					d1 = snap217
					d2 = snap218
					d3 = snap219
					d4 = snap220
					d5 = snap221
					d6 = snap222
					d7 = snap223
					d8 = snap224
					d9 = snap225
					d10 = snap226
					d11 = snap227
					d40 = snap228
					d41 = snap229
					d42 = snap230
					d43 = snap231
					d44 = snap232
					d45 = snap233
					d46 = snap234
					d47 = snap235
					d48 = snap236
					d49 = snap237
					d50 = snap238
					d51 = snap239
					d52 = snap240
					d108 = snap241
					d109 = snap242
					d110 = snap243
					d111 = snap244
					d112 = snap245
					d177 = snap246
					d178 = snap247
					d179 = snap248
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps215)
					}
					return result
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d250 := ps.PhiValues[0]
							ctx.EnsureDesc(&d250)
							ctx.EmitStoreToStack(d250, int32(bbs[7].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
					}
					if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != LocNone {
						d250 = ps.OverlayValues[250]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDescsTogether(&d1, &d7)
					var d251 JITValueDesc
					if d1.Loc == LocImm && d7.Loc == LocImm {
						d251 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d7.Imm.Int())}
					} else if d7.Loc == LocImm {
						r10 := ctx.AllocRegExcept(d1.Reg)
						if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d7.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d7.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, RegR11)
						}
						d251 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondSignedLess}
						ctx.BindReg(r10, &d251)
					} else if d1.Loc == LocImm {
						r11 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d7.Reg)
						d251 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r11, Condition: CondSignedLess}
						ctx.BindReg(r11, &d251)
					} else {
						r12 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d7.Reg)
						d251 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedLess}
						ctx.BindReg(r12, &d251)
					}
					ctx.FreeDesc(&d7)
					d252 = d251
					ctx.EnsureDesc(&d252)
					if d252.Loc != LocImm && d252.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d252.Loc == LocImm {
						if d252.Imm.Bool() {
							if ps.General {
							}
							ps253 := PhiState{General: ps.General}
							ps253.OverlayValues = make([]JITValueDesc, 253)
							ps253.OverlayValues[1] = d1
							ps253.OverlayValues[2] = d2
							ps253.OverlayValues[3] = d3
							ps253.OverlayValues[4] = d4
							ps253.OverlayValues[5] = d5
							ps253.OverlayValues[6] = d6
							ps253.OverlayValues[7] = d7
							ps253.OverlayValues[8] = d8
							ps253.OverlayValues[9] = d9
							ps253.OverlayValues[10] = d10
							ps253.OverlayValues[11] = d11
							ps253.OverlayValues[40] = d40
							ps253.OverlayValues[41] = d41
							ps253.OverlayValues[42] = d42
							ps253.OverlayValues[43] = d43
							ps253.OverlayValues[44] = d44
							ps253.OverlayValues[45] = d45
							ps253.OverlayValues[46] = d46
							ps253.OverlayValues[47] = d47
							ps253.OverlayValues[48] = d48
							ps253.OverlayValues[49] = d49
							ps253.OverlayValues[50] = d50
							ps253.OverlayValues[51] = d51
							ps253.OverlayValues[52] = d52
							ps253.OverlayValues[108] = d108
							ps253.OverlayValues[109] = d109
							ps253.OverlayValues[110] = d110
							ps253.OverlayValues[111] = d111
							ps253.OverlayValues[112] = d112
							ps253.OverlayValues[177] = d177
							ps253.OverlayValues[178] = d178
							ps253.OverlayValues[179] = d179
							ps253.OverlayValues[250] = d250
							ps253.OverlayValues[251] = d251
							ps253.OverlayValues[252] = d252
							return bbs[8].RenderPS(ps253)
						}
						if ps.General {
						}
						ps254 := PhiState{General: ps.General}
						ps254.OverlayValues = make([]JITValueDesc, 253)
						ps254.OverlayValues[1] = d1
						ps254.OverlayValues[2] = d2
						ps254.OverlayValues[3] = d3
						ps254.OverlayValues[4] = d4
						ps254.OverlayValues[5] = d5
						ps254.OverlayValues[6] = d6
						ps254.OverlayValues[7] = d7
						ps254.OverlayValues[8] = d8
						ps254.OverlayValues[9] = d9
						ps254.OverlayValues[10] = d10
						ps254.OverlayValues[11] = d11
						ps254.OverlayValues[40] = d40
						ps254.OverlayValues[41] = d41
						ps254.OverlayValues[42] = d42
						ps254.OverlayValues[43] = d43
						ps254.OverlayValues[44] = d44
						ps254.OverlayValues[45] = d45
						ps254.OverlayValues[46] = d46
						ps254.OverlayValues[47] = d47
						ps254.OverlayValues[48] = d48
						ps254.OverlayValues[49] = d49
						ps254.OverlayValues[50] = d50
						ps254.OverlayValues[51] = d51
						ps254.OverlayValues[52] = d52
						ps254.OverlayValues[108] = d108
						ps254.OverlayValues[109] = d109
						ps254.OverlayValues[110] = d110
						ps254.OverlayValues[111] = d111
						ps254.OverlayValues[112] = d112
						ps254.OverlayValues[177] = d177
						ps254.OverlayValues[178] = d178
						ps254.OverlayValues[179] = d179
						ps254.OverlayValues[250] = d250
						ps254.OverlayValues[251] = d251
						ps254.OverlayValues[252] = d252
						return bbs[9].RenderPS(ps254)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d255 := ps.PhiValues[0]
							ctx.EnsureDesc(&d255)
							ctx.EmitStoreToStack(d255, int32(bbs[7].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					ctx.EmitJump(d252.Condition, lbl9)
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
					}
					ctx.FreeDesc(&d251)
					snap256 := d1
					snap257 := d2
					snap258 := d3
					snap259 := d4
					snap260 := d5
					snap261 := d6
					snap262 := d7
					snap263 := d8
					snap264 := d9
					snap265 := d10
					snap266 := d11
					snap267 := d40
					snap268 := d41
					snap269 := d42
					snap270 := d43
					snap271 := d44
					snap272 := d45
					snap273 := d46
					snap274 := d47
					snap275 := d48
					snap276 := d49
					snap277 := d50
					snap278 := d51
					snap279 := d52
					snap280 := d108
					snap281 := d109
					snap282 := d110
					snap283 := d111
					snap284 := d112
					snap285 := d177
					snap286 := d178
					snap287 := d179
					snap288 := d250
					snap289 := d251
					snap290 := d252
					snap291 := d255
					alloc292 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc292)
					d1 = snap256
					d2 = snap257
					d3 = snap258
					d4 = snap259
					d5 = snap260
					d6 = snap261
					d7 = snap262
					d8 = snap263
					d9 = snap264
					d10 = snap265
					d11 = snap266
					d40 = snap267
					d41 = snap268
					d42 = snap269
					d43 = snap270
					d44 = snap271
					d45 = snap272
					d46 = snap273
					d47 = snap274
					d48 = snap275
					d49 = snap276
					d50 = snap277
					d51 = snap278
					d52 = snap279
					d108 = snap280
					d109 = snap281
					d110 = snap282
					d111 = snap283
					d112 = snap284
					d177 = snap285
					d178 = snap286
					d179 = snap287
					d250 = snap288
					d251 = snap289
					d252 = snap290
					d255 = snap291
					ctx.RestoreAllocState(alloc292)
					d1 = snap256
					d2 = snap257
					d3 = snap258
					d4 = snap259
					d5 = snap260
					d6 = snap261
					d7 = snap262
					d8 = snap263
					d9 = snap264
					d10 = snap265
					d11 = snap266
					d40 = snap267
					d41 = snap268
					d42 = snap269
					d43 = snap270
					d44 = snap271
					d45 = snap272
					d46 = snap273
					d47 = snap274
					d48 = snap275
					d49 = snap276
					d50 = snap277
					d51 = snap278
					d52 = snap279
					d108 = snap280
					d109 = snap281
					d110 = snap282
					d111 = snap283
					d112 = snap284
					d177 = snap285
					d178 = snap286
					d179 = snap287
					d250 = snap288
					d251 = snap289
					d252 = snap290
					d255 = snap291
					ps293 := PhiState{General: true}
					ps293.OverlayValues = make([]JITValueDesc, 256)
					ps293.OverlayValues[1] = d1
					ps293.OverlayValues[2] = d2
					ps293.OverlayValues[3] = d3
					ps293.OverlayValues[4] = d4
					ps293.OverlayValues[5] = d5
					ps293.OverlayValues[6] = d6
					ps293.OverlayValues[7] = d7
					ps293.OverlayValues[8] = d8
					ps293.OverlayValues[9] = d9
					ps293.OverlayValues[10] = d10
					ps293.OverlayValues[11] = d11
					ps293.OverlayValues[40] = d40
					ps293.OverlayValues[41] = d41
					ps293.OverlayValues[42] = d42
					ps293.OverlayValues[43] = d43
					ps293.OverlayValues[44] = d44
					ps293.OverlayValues[45] = d45
					ps293.OverlayValues[46] = d46
					ps293.OverlayValues[47] = d47
					ps293.OverlayValues[48] = d48
					ps293.OverlayValues[49] = d49
					ps293.OverlayValues[50] = d50
					ps293.OverlayValues[51] = d51
					ps293.OverlayValues[52] = d52
					ps293.OverlayValues[108] = d108
					ps293.OverlayValues[109] = d109
					ps293.OverlayValues[110] = d110
					ps293.OverlayValues[111] = d111
					ps293.OverlayValues[112] = d112
					ps293.OverlayValues[177] = d177
					ps293.OverlayValues[178] = d178
					ps293.OverlayValues[179] = d179
					ps293.OverlayValues[250] = d250
					ps293.OverlayValues[251] = d251
					ps293.OverlayValues[252] = d252
					ps293.OverlayValues[255] = d255
					ps294 := PhiState{General: true}
					ps294.OverlayValues = make([]JITValueDesc, 256)
					ps294.OverlayValues[1] = d1
					ps294.OverlayValues[2] = d2
					ps294.OverlayValues[3] = d3
					ps294.OverlayValues[4] = d4
					ps294.OverlayValues[5] = d5
					ps294.OverlayValues[6] = d6
					ps294.OverlayValues[7] = d7
					ps294.OverlayValues[8] = d8
					ps294.OverlayValues[9] = d9
					ps294.OverlayValues[10] = d10
					ps294.OverlayValues[11] = d11
					ps294.OverlayValues[40] = d40
					ps294.OverlayValues[41] = d41
					ps294.OverlayValues[42] = d42
					ps294.OverlayValues[43] = d43
					ps294.OverlayValues[44] = d44
					ps294.OverlayValues[45] = d45
					ps294.OverlayValues[46] = d46
					ps294.OverlayValues[47] = d47
					ps294.OverlayValues[48] = d48
					ps294.OverlayValues[49] = d49
					ps294.OverlayValues[50] = d50
					ps294.OverlayValues[51] = d51
					ps294.OverlayValues[52] = d52
					ps294.OverlayValues[108] = d108
					ps294.OverlayValues[109] = d109
					ps294.OverlayValues[110] = d110
					ps294.OverlayValues[111] = d111
					ps294.OverlayValues[112] = d112
					ps294.OverlayValues[177] = d177
					ps294.OverlayValues[178] = d178
					ps294.OverlayValues[179] = d179
					ps294.OverlayValues[250] = d250
					ps294.OverlayValues[251] = d251
					ps294.OverlayValues[252] = d252
					ps294.OverlayValues[255] = d255
					snap295 := d1
					snap296 := d2
					snap297 := d3
					snap298 := d4
					snap299 := d5
					snap300 := d6
					snap301 := d7
					snap302 := d8
					snap303 := d9
					snap304 := d10
					snap305 := d11
					snap306 := d40
					snap307 := d41
					snap308 := d42
					snap309 := d43
					snap310 := d44
					snap311 := d45
					snap312 := d46
					snap313 := d47
					snap314 := d48
					snap315 := d49
					snap316 := d50
					snap317 := d51
					snap318 := d52
					snap319 := d108
					snap320 := d109
					snap321 := d110
					snap322 := d111
					snap323 := d112
					snap324 := d177
					snap325 := d178
					snap326 := d179
					snap327 := d250
					snap328 := d251
					snap329 := d252
					snap330 := d255
					alloc331 := ctx.SnapshotAllocState()
					if !bbs[9].Rendered {
						bbs[9].RenderPS(ps294)
					}
					ctx.RestoreAllocState(alloc331)
					d1 = snap295
					d2 = snap296
					d3 = snap297
					d4 = snap298
					d5 = snap299
					d6 = snap300
					d7 = snap301
					d8 = snap302
					d9 = snap303
					d10 = snap304
					d11 = snap305
					d40 = snap306
					d41 = snap307
					d42 = snap308
					d43 = snap309
					d44 = snap310
					d45 = snap311
					d46 = snap312
					d47 = snap313
					d48 = snap314
					d49 = snap315
					d50 = snap316
					d51 = snap317
					d52 = snap318
					d108 = snap319
					d109 = snap320
					d110 = snap321
					d111 = snap322
					d112 = snap323
					d177 = snap324
					d178 = snap325
					d179 = snap326
					d250 = snap327
					d251 = snap328
					d252 = snap329
					d255 = snap330
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps293)
					}
					return result
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
					}
					if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != LocNone {
						d250 = ps.OverlayValues[250]
					}
					if len(ps.OverlayValues) > 251 && ps.OverlayValues[251].Loc != LocNone {
						d251 = ps.OverlayValues[251]
					}
					if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != LocNone {
						d252 = ps.OverlayValues[252]
					}
					if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != LocNone {
						d255 = ps.OverlayValues[255]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d43)
					var d332 JITValueDesc
					ctx.EnsureDesc(&d50)
					if d50.Loc == LocRegPair || d50.Loc == LocRegTriple {
						d332 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d50.Reg2}
						ctx.BindReg(d50.Reg2, &d332)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d50)
					ctx.EnsureDesc(&d43)
					ctx.EnsureDesc(&d332)
					var d334 JITValueDesc
					if d332.Loc == LocImm && d43.Loc == LocImm {
						d334 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d332.Imm.Int() - d43.Imm.Int())}
					} else {
						r13 := ctx.AllocReg()
						if d332.Loc == LocImm {
							ctx.EmitMovRegImm64(r13, uint64(d332.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r13, d332.Reg)
						}
						if d43.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()))
							ctx.EmitSubInt64(r13, RegR11)
						} else {
							ctx.EmitSubInt64(r13, d43.Reg)
						}
						d334 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r13}
						ctx.BindReg(r13, &d334)
					}
					var d335 JITValueDesc
					r14 := ctx.EmitSliceDataAfterLow(&d50, &d43, 16)
					d335 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
					ctx.BindReg(r14, &d335)
					ctx.BindReg(r14, &d335)
					var d336 JITValueDesc
					var r15 Reg
					var r16 Reg
					ctx.SyncDesc(&d335)
					ctx.EnsureDesc(&d335)
					if d335.Loc == LocImm {
						r15 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r15, uint64(d335.Imm.Int()))
					} else {
						r15 = d335.Reg
					}
					ctx.ProtectReg(r15)
					ctx.SyncDesc(&d334)
					ctx.EnsureDesc(&d334)
					if d334.Loc == LocImm {
						r16 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r16, uint64(d334.Imm.Int()))
					} else {
						r16 = d334.Reg
					}
					ctx.ProtectReg(r16)
					r17 := ctx.EmitSliceCapAfterLow(&d50, &d43, r15, r16)
					ctx.UnprotectReg(r16)
					ctx.UnprotectReg(r15)
					d336 = JITValueDesc{Loc: LocRegTriple, Reg: r15, Reg2: r16, Reg3: r17}
					ctx.BindReg(r15, &d336)
					ctx.BindReg(r16, &d336)
					ctx.BindReg(r17, &d336)
					ctx.BindReg(r15, &d336)
					ctx.BindReg(r16, &d336)
					ctx.BindReg(r17, &d336)
					ctx.EnsureDesc(&d50)
					ctx.EnsureDesc(&d336)
					callResults337 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d50, d336}, []uint8{1}, []uint8{0})
					d338 = callResults337[0]
					d338.Type = tagInt
					var d339 JITValueDesc
					if d50.SliceSizeKnown {
						d339 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d50.KnownSliceLen))}
					} else if d50.Loc == LocImm {
						d339 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d50.StackOff))}
					} else if d50.Loc == LocStackTriple {
						d339 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d50.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d50)
						if d50.Loc == LocRegPair || d50.Loc == LocRegTriple {
							d339 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d50.Reg2, ID: 0}
						} else if d50.Loc == LocReg {
							d339 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d50.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d339)
					ctx.EnsureDesc(&d43)
					ctx.EnsureDescsTogether(&d339, &d43)
					var d340 JITValueDesc
					if d339.Loc == LocImm && d43.Loc == LocImm {
						d340 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d339.Imm.Int() - d43.Imm.Int())}
					} else if d43.Loc == LocImm && d43.Imm.Int() == 0 {
						r18 := ctx.AllocRegExcept(d339.Reg)
						ctx.EmitMovRegReg(r18, d339.Reg)
						d340 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d340)
					} else if d339.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d43.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d339.Imm.Int()))
						ctx.EmitSubInt64(scratch, d43.Reg)
						d340 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d340)
					} else if d43.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d339.Reg)
						ctx.EmitMovRegReg(scratch, d339.Reg)
						if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d43.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d340 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d340)
					} else {
						r19 := ctx.AllocRegExcept(d339.Reg, d43.Reg)
						ctx.EmitMovRegReg(r19, d339.Reg)
						ctx.EmitSubInt64(r19, d43.Reg)
						d340 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
						ctx.BindReg(r19, &d340)
					}
					if d340.Loc == LocReg && d339.Loc == LocReg && d340.Reg == d339.Reg {
						ctx.TransferReg(d339.Reg)
						d339.Loc = LocNone
					}
					ctx.EnsureDesc(&d340)
					ctx.EmitStoreToStack(d340, int32(bbs[10].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d340)
					ctx.FreeDesc(&d339)
					ctx.FreeDesc(&d43)
					if ps.General {
					}
					ps341 := PhiState{General: ps.General}
					ps341.OverlayValues = make([]JITValueDesc, 341)
					ps341.OverlayValues[1] = d1
					ps341.OverlayValues[2] = d2
					ps341.OverlayValues[3] = d3
					ps341.OverlayValues[4] = d4
					ps341.OverlayValues[5] = d5
					ps341.OverlayValues[6] = d6
					ps341.OverlayValues[7] = d7
					ps341.OverlayValues[8] = d8
					ps341.OverlayValues[9] = d9
					ps341.OverlayValues[10] = d10
					ps341.OverlayValues[11] = d11
					ps341.OverlayValues[40] = d40
					ps341.OverlayValues[41] = d41
					ps341.OverlayValues[42] = d42
					ps341.OverlayValues[43] = d43
					ps341.OverlayValues[44] = d44
					ps341.OverlayValues[45] = d45
					ps341.OverlayValues[46] = d46
					ps341.OverlayValues[47] = d47
					ps341.OverlayValues[48] = d48
					ps341.OverlayValues[49] = d49
					ps341.OverlayValues[50] = d50
					ps341.OverlayValues[51] = d51
					ps341.OverlayValues[52] = d52
					ps341.OverlayValues[108] = d108
					ps341.OverlayValues[109] = d109
					ps341.OverlayValues[110] = d110
					ps341.OverlayValues[111] = d111
					ps341.OverlayValues[112] = d112
					ps341.OverlayValues[177] = d177
					ps341.OverlayValues[178] = d178
					ps341.OverlayValues[179] = d179
					ps341.OverlayValues[250] = d250
					ps341.OverlayValues[251] = d251
					ps341.OverlayValues[252] = d252
					ps341.OverlayValues[255] = d255
					ps341.OverlayValues[332] = d332
					ps341.OverlayValues[333] = d333
					ps341.OverlayValues[334] = d334
					ps341.OverlayValues[335] = d335
					ps341.OverlayValues[336] = d336
					ps341.OverlayValues[338] = d338
					ps341.OverlayValues[339] = d339
					ps341.OverlayValues[340] = d340
					ps341.PhiValues = make([]JITValueDesc, 1)
					if ps341.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps341)
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
					}
					if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != LocNone {
						d250 = ps.OverlayValues[250]
					}
					if len(ps.OverlayValues) > 251 && ps.OverlayValues[251].Loc != LocNone {
						d251 = ps.OverlayValues[251]
					}
					if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != LocNone {
						d252 = ps.OverlayValues[252]
					}
					if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != LocNone {
						d255 = ps.OverlayValues[255]
					}
					if len(ps.OverlayValues) > 332 && ps.OverlayValues[332].Loc != LocNone {
						d332 = ps.OverlayValues[332]
					}
					if len(ps.OverlayValues) > 333 && ps.OverlayValues[333].Loc != LocNone {
						d333 = ps.OverlayValues[333]
					}
					if len(ps.OverlayValues) > 334 && ps.OverlayValues[334].Loc != LocNone {
						d334 = ps.OverlayValues[334]
					}
					if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
						d335 = ps.OverlayValues[335]
					}
					if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
						d336 = ps.OverlayValues[336]
					}
					if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != LocNone {
						d338 = ps.OverlayValues[338]
					}
					if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
						d339 = ps.OverlayValues[339]
					}
					if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
						d340 = ps.OverlayValues[340]
					}
					ctx.ReclaimUntrackedRegs()
					d342 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d342)
					if d342.Loc == LocRegPair || d342.Loc == LocStackPair || d342.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d342, &result)
						result.Type = d342.Type
					} else {
						switch d342.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d342)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d342)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d342)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d342, &result)
							result.Type = d342.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d343 := ps.PhiValues[0]
							ctx.EnsureDesc(&d343)
							ctx.EmitStoreToStack(d343, int32(bbs[10].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
					}
					if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != LocNone {
						d250 = ps.OverlayValues[250]
					}
					if len(ps.OverlayValues) > 251 && ps.OverlayValues[251].Loc != LocNone {
						d251 = ps.OverlayValues[251]
					}
					if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != LocNone {
						d252 = ps.OverlayValues[252]
					}
					if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != LocNone {
						d255 = ps.OverlayValues[255]
					}
					if len(ps.OverlayValues) > 332 && ps.OverlayValues[332].Loc != LocNone {
						d332 = ps.OverlayValues[332]
					}
					if len(ps.OverlayValues) > 333 && ps.OverlayValues[333].Loc != LocNone {
						d333 = ps.OverlayValues[333]
					}
					if len(ps.OverlayValues) > 334 && ps.OverlayValues[334].Loc != LocNone {
						d334 = ps.OverlayValues[334]
					}
					if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
						d335 = ps.OverlayValues[335]
					}
					if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
						d336 = ps.OverlayValues[336]
					}
					if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != LocNone {
						d338 = ps.OverlayValues[338]
					}
					if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
						d339 = ps.OverlayValues[339]
					}
					if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
						d340 = ps.OverlayValues[340]
					}
					if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
						d342 = ps.OverlayValues[342]
					}
					if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
						d343 = ps.OverlayValues[343]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					var d344 JITValueDesc
					if d50.SliceSizeKnown {
						d344 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d50.KnownSliceLen))}
					} else if d50.Loc == LocImm {
						d344 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d50.StackOff))}
					} else if d50.Loc == LocStackTriple {
						d344 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d50.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d50)
						if d50.Loc == LocRegPair || d50.Loc == LocRegTriple {
							d344 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d50.Reg2, ID: 0}
						} else if d50.Loc == LocReg {
							d344 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d50.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d344)
					ctx.EnsureDescsTogether(&d2, &d344)
					var d345 JITValueDesc
					if d2.Loc == LocImm && d344.Loc == LocImm {
						d345 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d344.Imm.Int())}
					} else if d344.Loc == LocImm {
						r20 := ctx.AllocRegExcept(d2.Reg)
						if d344.Imm.Int() >= -2147483648 && d344.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d2.Reg, int32(d344.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d344.Imm.Int()))
							ctx.EmitCmpInt64(d2.Reg, RegR11)
						}
						d345 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r20, Condition: CondSignedLess}
						ctx.BindReg(r20, &d345)
					} else if d2.Loc == LocImm {
						r21 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d344.Reg)
						d345 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r21, Condition: CondSignedLess}
						ctx.BindReg(r21, &d345)
					} else {
						r22 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpInt64(d2.Reg, d344.Reg)
						d345 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondSignedLess}
						ctx.BindReg(r22, &d345)
					}
					ctx.FreeDesc(&d344)
					d346 = d345
					ctx.EnsureDesc(&d346)
					if d346.Loc != LocImm && d346.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d346.Loc == LocImm {
						if d346.Imm.Bool() {
							if ps.General {
							}
							ps347 := PhiState{General: ps.General}
							ps347.OverlayValues = make([]JITValueDesc, 347)
							ps347.OverlayValues[1] = d1
							ps347.OverlayValues[2] = d2
							ps347.OverlayValues[3] = d3
							ps347.OverlayValues[4] = d4
							ps347.OverlayValues[5] = d5
							ps347.OverlayValues[6] = d6
							ps347.OverlayValues[7] = d7
							ps347.OverlayValues[8] = d8
							ps347.OverlayValues[9] = d9
							ps347.OverlayValues[10] = d10
							ps347.OverlayValues[11] = d11
							ps347.OverlayValues[40] = d40
							ps347.OverlayValues[41] = d41
							ps347.OverlayValues[42] = d42
							ps347.OverlayValues[43] = d43
							ps347.OverlayValues[44] = d44
							ps347.OverlayValues[45] = d45
							ps347.OverlayValues[46] = d46
							ps347.OverlayValues[47] = d47
							ps347.OverlayValues[48] = d48
							ps347.OverlayValues[49] = d49
							ps347.OverlayValues[50] = d50
							ps347.OverlayValues[51] = d51
							ps347.OverlayValues[52] = d52
							ps347.OverlayValues[108] = d108
							ps347.OverlayValues[109] = d109
							ps347.OverlayValues[110] = d110
							ps347.OverlayValues[111] = d111
							ps347.OverlayValues[112] = d112
							ps347.OverlayValues[177] = d177
							ps347.OverlayValues[178] = d178
							ps347.OverlayValues[179] = d179
							ps347.OverlayValues[250] = d250
							ps347.OverlayValues[251] = d251
							ps347.OverlayValues[252] = d252
							ps347.OverlayValues[255] = d255
							ps347.OverlayValues[332] = d332
							ps347.OverlayValues[333] = d333
							ps347.OverlayValues[334] = d334
							ps347.OverlayValues[335] = d335
							ps347.OverlayValues[336] = d336
							ps347.OverlayValues[338] = d338
							ps347.OverlayValues[339] = d339
							ps347.OverlayValues[340] = d340
							ps347.OverlayValues[342] = d342
							ps347.OverlayValues[343] = d343
							ps347.OverlayValues[344] = d344
							ps347.OverlayValues[345] = d345
							ps347.OverlayValues[346] = d346
							return bbs[11].RenderPS(ps347)
						}
						if ps.General {
						}
						ps348 := PhiState{General: ps.General}
						ps348.OverlayValues = make([]JITValueDesc, 347)
						ps348.OverlayValues[1] = d1
						ps348.OverlayValues[2] = d2
						ps348.OverlayValues[3] = d3
						ps348.OverlayValues[4] = d4
						ps348.OverlayValues[5] = d5
						ps348.OverlayValues[6] = d6
						ps348.OverlayValues[7] = d7
						ps348.OverlayValues[8] = d8
						ps348.OverlayValues[9] = d9
						ps348.OverlayValues[10] = d10
						ps348.OverlayValues[11] = d11
						ps348.OverlayValues[40] = d40
						ps348.OverlayValues[41] = d41
						ps348.OverlayValues[42] = d42
						ps348.OverlayValues[43] = d43
						ps348.OverlayValues[44] = d44
						ps348.OverlayValues[45] = d45
						ps348.OverlayValues[46] = d46
						ps348.OverlayValues[47] = d47
						ps348.OverlayValues[48] = d48
						ps348.OverlayValues[49] = d49
						ps348.OverlayValues[50] = d50
						ps348.OverlayValues[51] = d51
						ps348.OverlayValues[52] = d52
						ps348.OverlayValues[108] = d108
						ps348.OverlayValues[109] = d109
						ps348.OverlayValues[110] = d110
						ps348.OverlayValues[111] = d111
						ps348.OverlayValues[112] = d112
						ps348.OverlayValues[177] = d177
						ps348.OverlayValues[178] = d178
						ps348.OverlayValues[179] = d179
						ps348.OverlayValues[250] = d250
						ps348.OverlayValues[251] = d251
						ps348.OverlayValues[252] = d252
						ps348.OverlayValues[255] = d255
						ps348.OverlayValues[332] = d332
						ps348.OverlayValues[333] = d333
						ps348.OverlayValues[334] = d334
						ps348.OverlayValues[335] = d335
						ps348.OverlayValues[336] = d336
						ps348.OverlayValues[338] = d338
						ps348.OverlayValues[339] = d339
						ps348.OverlayValues[340] = d340
						ps348.OverlayValues[342] = d342
						ps348.OverlayValues[343] = d343
						ps348.OverlayValues[344] = d344
						ps348.OverlayValues[345] = d345
						ps348.OverlayValues[346] = d346
						return bbs[12].RenderPS(ps348)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d349 := ps.PhiValues[0]
							ctx.EnsureDesc(&d349)
							ctx.EmitStoreToStack(d349, int32(bbs[10].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[10].RenderPS(ps)
					}
					ctx.EmitJump(d346.Condition, lbl12)
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
					}
					ctx.FreeDesc(&d345)
					snap350 := d1
					snap351 := d2
					snap352 := d3
					snap353 := d4
					snap354 := d5
					snap355 := d6
					snap356 := d7
					snap357 := d8
					snap358 := d9
					snap359 := d10
					snap360 := d11
					snap361 := d40
					snap362 := d41
					snap363 := d42
					snap364 := d43
					snap365 := d44
					snap366 := d45
					snap367 := d46
					snap368 := d47
					snap369 := d48
					snap370 := d49
					snap371 := d50
					snap372 := d51
					snap373 := d52
					snap374 := d108
					snap375 := d109
					snap376 := d110
					snap377 := d111
					snap378 := d112
					snap379 := d177
					snap380 := d178
					snap381 := d179
					snap382 := d250
					snap383 := d251
					snap384 := d252
					snap385 := d255
					snap386 := d332
					snap387 := d333
					snap388 := d334
					snap389 := d335
					snap390 := d336
					snap391 := d338
					snap392 := d339
					snap393 := d340
					snap394 := d342
					snap395 := d343
					snap396 := d344
					snap397 := d345
					snap398 := d346
					snap399 := d349
					alloc400 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc400)
					d1 = snap350
					d2 = snap351
					d3 = snap352
					d4 = snap353
					d5 = snap354
					d6 = snap355
					d7 = snap356
					d8 = snap357
					d9 = snap358
					d10 = snap359
					d11 = snap360
					d40 = snap361
					d41 = snap362
					d42 = snap363
					d43 = snap364
					d44 = snap365
					d45 = snap366
					d46 = snap367
					d47 = snap368
					d48 = snap369
					d49 = snap370
					d50 = snap371
					d51 = snap372
					d52 = snap373
					d108 = snap374
					d109 = snap375
					d110 = snap376
					d111 = snap377
					d112 = snap378
					d177 = snap379
					d178 = snap380
					d179 = snap381
					d250 = snap382
					d251 = snap383
					d252 = snap384
					d255 = snap385
					d332 = snap386
					d333 = snap387
					d334 = snap388
					d335 = snap389
					d336 = snap390
					d338 = snap391
					d339 = snap392
					d340 = snap393
					d342 = snap394
					d343 = snap395
					d344 = snap396
					d345 = snap397
					d346 = snap398
					d349 = snap399
					ctx.RestoreAllocState(alloc400)
					d1 = snap350
					d2 = snap351
					d3 = snap352
					d4 = snap353
					d5 = snap354
					d6 = snap355
					d7 = snap356
					d8 = snap357
					d9 = snap358
					d10 = snap359
					d11 = snap360
					d40 = snap361
					d41 = snap362
					d42 = snap363
					d43 = snap364
					d44 = snap365
					d45 = snap366
					d46 = snap367
					d47 = snap368
					d48 = snap369
					d49 = snap370
					d50 = snap371
					d51 = snap372
					d52 = snap373
					d108 = snap374
					d109 = snap375
					d110 = snap376
					d111 = snap377
					d112 = snap378
					d177 = snap379
					d178 = snap380
					d179 = snap381
					d250 = snap382
					d251 = snap383
					d252 = snap384
					d255 = snap385
					d332 = snap386
					d333 = snap387
					d334 = snap388
					d335 = snap389
					d336 = snap390
					d338 = snap391
					d339 = snap392
					d340 = snap393
					d342 = snap394
					d343 = snap395
					d344 = snap396
					d345 = snap397
					d346 = snap398
					d349 = snap399
					ps401 := PhiState{General: true}
					ps401.OverlayValues = make([]JITValueDesc, 350)
					ps401.OverlayValues[1] = d1
					ps401.OverlayValues[2] = d2
					ps401.OverlayValues[3] = d3
					ps401.OverlayValues[4] = d4
					ps401.OverlayValues[5] = d5
					ps401.OverlayValues[6] = d6
					ps401.OverlayValues[7] = d7
					ps401.OverlayValues[8] = d8
					ps401.OverlayValues[9] = d9
					ps401.OverlayValues[10] = d10
					ps401.OverlayValues[11] = d11
					ps401.OverlayValues[40] = d40
					ps401.OverlayValues[41] = d41
					ps401.OverlayValues[42] = d42
					ps401.OverlayValues[43] = d43
					ps401.OverlayValues[44] = d44
					ps401.OverlayValues[45] = d45
					ps401.OverlayValues[46] = d46
					ps401.OverlayValues[47] = d47
					ps401.OverlayValues[48] = d48
					ps401.OverlayValues[49] = d49
					ps401.OverlayValues[50] = d50
					ps401.OverlayValues[51] = d51
					ps401.OverlayValues[52] = d52
					ps401.OverlayValues[108] = d108
					ps401.OverlayValues[109] = d109
					ps401.OverlayValues[110] = d110
					ps401.OverlayValues[111] = d111
					ps401.OverlayValues[112] = d112
					ps401.OverlayValues[177] = d177
					ps401.OverlayValues[178] = d178
					ps401.OverlayValues[179] = d179
					ps401.OverlayValues[250] = d250
					ps401.OverlayValues[251] = d251
					ps401.OverlayValues[252] = d252
					ps401.OverlayValues[255] = d255
					ps401.OverlayValues[332] = d332
					ps401.OverlayValues[333] = d333
					ps401.OverlayValues[334] = d334
					ps401.OverlayValues[335] = d335
					ps401.OverlayValues[336] = d336
					ps401.OverlayValues[338] = d338
					ps401.OverlayValues[339] = d339
					ps401.OverlayValues[340] = d340
					ps401.OverlayValues[342] = d342
					ps401.OverlayValues[343] = d343
					ps401.OverlayValues[344] = d344
					ps401.OverlayValues[345] = d345
					ps401.OverlayValues[346] = d346
					ps401.OverlayValues[349] = d349
					ps402 := PhiState{General: true}
					ps402.OverlayValues = make([]JITValueDesc, 350)
					ps402.OverlayValues[1] = d1
					ps402.OverlayValues[2] = d2
					ps402.OverlayValues[3] = d3
					ps402.OverlayValues[4] = d4
					ps402.OverlayValues[5] = d5
					ps402.OverlayValues[6] = d6
					ps402.OverlayValues[7] = d7
					ps402.OverlayValues[8] = d8
					ps402.OverlayValues[9] = d9
					ps402.OverlayValues[10] = d10
					ps402.OverlayValues[11] = d11
					ps402.OverlayValues[40] = d40
					ps402.OverlayValues[41] = d41
					ps402.OverlayValues[42] = d42
					ps402.OverlayValues[43] = d43
					ps402.OverlayValues[44] = d44
					ps402.OverlayValues[45] = d45
					ps402.OverlayValues[46] = d46
					ps402.OverlayValues[47] = d47
					ps402.OverlayValues[48] = d48
					ps402.OverlayValues[49] = d49
					ps402.OverlayValues[50] = d50
					ps402.OverlayValues[51] = d51
					ps402.OverlayValues[52] = d52
					ps402.OverlayValues[108] = d108
					ps402.OverlayValues[109] = d109
					ps402.OverlayValues[110] = d110
					ps402.OverlayValues[111] = d111
					ps402.OverlayValues[112] = d112
					ps402.OverlayValues[177] = d177
					ps402.OverlayValues[178] = d178
					ps402.OverlayValues[179] = d179
					ps402.OverlayValues[250] = d250
					ps402.OverlayValues[251] = d251
					ps402.OverlayValues[252] = d252
					ps402.OverlayValues[255] = d255
					ps402.OverlayValues[332] = d332
					ps402.OverlayValues[333] = d333
					ps402.OverlayValues[334] = d334
					ps402.OverlayValues[335] = d335
					ps402.OverlayValues[336] = d336
					ps402.OverlayValues[338] = d338
					ps402.OverlayValues[339] = d339
					ps402.OverlayValues[340] = d340
					ps402.OverlayValues[342] = d342
					ps402.OverlayValues[343] = d343
					ps402.OverlayValues[344] = d344
					ps402.OverlayValues[345] = d345
					ps402.OverlayValues[346] = d346
					ps402.OverlayValues[349] = d349
					snap403 := d1
					snap404 := d2
					snap405 := d3
					snap406 := d4
					snap407 := d5
					snap408 := d6
					snap409 := d7
					snap410 := d8
					snap411 := d9
					snap412 := d10
					snap413 := d11
					snap414 := d40
					snap415 := d41
					snap416 := d42
					snap417 := d43
					snap418 := d44
					snap419 := d45
					snap420 := d46
					snap421 := d47
					snap422 := d48
					snap423 := d49
					snap424 := d50
					snap425 := d51
					snap426 := d52
					snap427 := d108
					snap428 := d109
					snap429 := d110
					snap430 := d111
					snap431 := d112
					snap432 := d177
					snap433 := d178
					snap434 := d179
					snap435 := d250
					snap436 := d251
					snap437 := d252
					snap438 := d255
					snap439 := d332
					snap440 := d333
					snap441 := d334
					snap442 := d335
					snap443 := d336
					snap444 := d338
					snap445 := d339
					snap446 := d340
					snap447 := d342
					snap448 := d343
					snap449 := d344
					snap450 := d345
					snap451 := d346
					snap452 := d349
					alloc453 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps402)
					}
					ctx.RestoreAllocState(alloc453)
					d1 = snap403
					d2 = snap404
					d3 = snap405
					d4 = snap406
					d5 = snap407
					d6 = snap408
					d7 = snap409
					d8 = snap410
					d9 = snap411
					d10 = snap412
					d11 = snap413
					d40 = snap414
					d41 = snap415
					d42 = snap416
					d43 = snap417
					d44 = snap418
					d45 = snap419
					d46 = snap420
					d47 = snap421
					d48 = snap422
					d49 = snap423
					d50 = snap424
					d51 = snap425
					d52 = snap426
					d108 = snap427
					d109 = snap428
					d110 = snap429
					d111 = snap430
					d112 = snap431
					d177 = snap432
					d178 = snap433
					d179 = snap434
					d250 = snap435
					d251 = snap436
					d252 = snap437
					d255 = snap438
					d332 = snap439
					d333 = snap440
					d334 = snap441
					d335 = snap442
					d336 = snap443
					d338 = snap444
					d339 = snap445
					d340 = snap446
					d342 = snap447
					d343 = snap448
					d344 = snap449
					d345 = snap450
					d346 = snap451
					d349 = snap452
					if !bbs[11].Rendered {
						return bbs[11].RenderPS(ps401)
					}
					return result
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
					}
					if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != LocNone {
						d250 = ps.OverlayValues[250]
					}
					if len(ps.OverlayValues) > 251 && ps.OverlayValues[251].Loc != LocNone {
						d251 = ps.OverlayValues[251]
					}
					if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != LocNone {
						d252 = ps.OverlayValues[252]
					}
					if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != LocNone {
						d255 = ps.OverlayValues[255]
					}
					if len(ps.OverlayValues) > 332 && ps.OverlayValues[332].Loc != LocNone {
						d332 = ps.OverlayValues[332]
					}
					if len(ps.OverlayValues) > 333 && ps.OverlayValues[333].Loc != LocNone {
						d333 = ps.OverlayValues[333]
					}
					if len(ps.OverlayValues) > 334 && ps.OverlayValues[334].Loc != LocNone {
						d334 = ps.OverlayValues[334]
					}
					if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
						d335 = ps.OverlayValues[335]
					}
					if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
						d336 = ps.OverlayValues[336]
					}
					if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != LocNone {
						d338 = ps.OverlayValues[338]
					}
					if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
						d339 = ps.OverlayValues[339]
					}
					if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
						d340 = ps.OverlayValues[340]
					}
					if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
						d342 = ps.OverlayValues[342]
					}
					if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
						d343 = ps.OverlayValues[343]
					}
					if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
						d344 = ps.OverlayValues[344]
					}
					if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != LocNone {
						d345 = ps.OverlayValues[345]
					}
					if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
						d346 = ps.OverlayValues[346]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
					}
					ctx.ReclaimUntrackedRegs()
					d454 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d2)
					ctx.SyncDesc(&d454)
					d455 = d50
					d455.ID = 0
					d456 = d2
					d456.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d455, &d456, &d454, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d2)
						d456 = d2
						d456.ID = 0
						ctx.EmitStoreScmerSliceElement(&d455, &d456, &d454, int32(16))
					}
					ctx.FreeDesc(&d456)
					ctx.FreeDesc(&d454)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					var d457 JITValueDesc
					if d2.Loc == LocImm {
						d457 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d457 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d457)
					}
					if d457.Loc == LocReg && d2.Loc == LocReg && d457.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d457)
					ctx.EmitStoreToStack(d457, int32(bbs[10].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d457)
					if ps.General {
					}
					ps458 := PhiState{General: ps.General}
					ps458.OverlayValues = make([]JITValueDesc, 458)
					ps458.OverlayValues[1] = d1
					ps458.OverlayValues[2] = d2
					ps458.OverlayValues[3] = d3
					ps458.OverlayValues[4] = d4
					ps458.OverlayValues[5] = d5
					ps458.OverlayValues[6] = d6
					ps458.OverlayValues[7] = d7
					ps458.OverlayValues[8] = d8
					ps458.OverlayValues[9] = d9
					ps458.OverlayValues[10] = d10
					ps458.OverlayValues[11] = d11
					ps458.OverlayValues[40] = d40
					ps458.OverlayValues[41] = d41
					ps458.OverlayValues[42] = d42
					ps458.OverlayValues[43] = d43
					ps458.OverlayValues[44] = d44
					ps458.OverlayValues[45] = d45
					ps458.OverlayValues[46] = d46
					ps458.OverlayValues[47] = d47
					ps458.OverlayValues[48] = d48
					ps458.OverlayValues[49] = d49
					ps458.OverlayValues[50] = d50
					ps458.OverlayValues[51] = d51
					ps458.OverlayValues[52] = d52
					ps458.OverlayValues[108] = d108
					ps458.OverlayValues[109] = d109
					ps458.OverlayValues[110] = d110
					ps458.OverlayValues[111] = d111
					ps458.OverlayValues[112] = d112
					ps458.OverlayValues[177] = d177
					ps458.OverlayValues[178] = d178
					ps458.OverlayValues[179] = d179
					ps458.OverlayValues[250] = d250
					ps458.OverlayValues[251] = d251
					ps458.OverlayValues[252] = d252
					ps458.OverlayValues[255] = d255
					ps458.OverlayValues[332] = d332
					ps458.OverlayValues[333] = d333
					ps458.OverlayValues[334] = d334
					ps458.OverlayValues[335] = d335
					ps458.OverlayValues[336] = d336
					ps458.OverlayValues[338] = d338
					ps458.OverlayValues[339] = d339
					ps458.OverlayValues[340] = d340
					ps458.OverlayValues[342] = d342
					ps458.OverlayValues[343] = d343
					ps458.OverlayValues[344] = d344
					ps458.OverlayValues[345] = d345
					ps458.OverlayValues[346] = d346
					ps458.OverlayValues[349] = d349
					ps458.OverlayValues[454] = d454
					ps458.OverlayValues[455] = d455
					ps458.OverlayValues[456] = d456
					ps458.OverlayValues[457] = d457
					ps458.PhiValues = make([]JITValueDesc, 1)
					if ps458.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps458)
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
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
					}
					if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != LocNone {
						d250 = ps.OverlayValues[250]
					}
					if len(ps.OverlayValues) > 251 && ps.OverlayValues[251].Loc != LocNone {
						d251 = ps.OverlayValues[251]
					}
					if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != LocNone {
						d252 = ps.OverlayValues[252]
					}
					if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != LocNone {
						d255 = ps.OverlayValues[255]
					}
					if len(ps.OverlayValues) > 332 && ps.OverlayValues[332].Loc != LocNone {
						d332 = ps.OverlayValues[332]
					}
					if len(ps.OverlayValues) > 333 && ps.OverlayValues[333].Loc != LocNone {
						d333 = ps.OverlayValues[333]
					}
					if len(ps.OverlayValues) > 334 && ps.OverlayValues[334].Loc != LocNone {
						d334 = ps.OverlayValues[334]
					}
					if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
						d335 = ps.OverlayValues[335]
					}
					if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
						d336 = ps.OverlayValues[336]
					}
					if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != LocNone {
						d338 = ps.OverlayValues[338]
					}
					if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
						d339 = ps.OverlayValues[339]
					}
					if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
						d340 = ps.OverlayValues[340]
					}
					if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
						d342 = ps.OverlayValues[342]
					}
					if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
						d343 = ps.OverlayValues[343]
					}
					if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
						d344 = ps.OverlayValues[344]
					}
					if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != LocNone {
						d345 = ps.OverlayValues[345]
					}
					if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
						d346 = ps.OverlayValues[346]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
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
					ctx.ReclaimUntrackedRegs()
					d459 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					d461 = ctx.EmitSliceElementAddress(&d4, &d459, 16)
					ctx.EnsureDesc(&d461)
					r23 := ctx.AllocRegExcept(d461.Reg)
					ctx.EmitMovRegMem(r23, d461.Reg, 8)
					ctx.EmitMovRegMem(d461.Reg, d461.Reg, 0)
					d460 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d461.Reg, Reg2: r23}
					ctx.BindReg(d461.Reg, &d460)
					ctx.BindReg(r23, &d460)
					var d462 JITValueDesc
					if d460.Loc == LocImm {
						d462 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d460.Imm.Int())}
					} else if d460.Type == tagInt && d460.Loc == LocRegPair {
						ctx.FreeReg(d460.Reg)
						d462 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d460.Reg2}
						ctx.BindReg(d460.Reg2, &d462)
						ctx.BindReg(d460.Reg2, &d462)
					} else if d460.Type == tagInt && d460.Loc == LocReg {
						d462 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d460.Reg}
						ctx.BindReg(d460.Reg, &d462)
						ctx.BindReg(d460.Reg, &d462)
					} else {
						d462 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d460}, 1)
						d462.Type = tagInt
						ctx.BindReg(d462.Reg, &d462)
					}
					ctx.FreeDesc(&d460)
					ctx.EnsureDesc(&d462)
					ctx.EnsureDesc(&d462)
					var d463 JITValueDesc
					if d462.Loc == LocImm {
						d463 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d462.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d462.Reg)
						ctx.EmitMovRegReg(scratch, d462.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d463 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d463)
					}
					if d463.Loc == LocReg && d462.Loc == LocReg && d463.Reg == d462.Reg {
						ctx.TransferReg(d462.Reg)
						d462.Loc = LocNone
					}
					ctx.FreeDesc(&d462)
					ctx.EnsureDesc(&d463)
					d464 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.SyncDesc(&d463)
					d465 = d4
					d465.ID = 0
					d466 = d464
					d466.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d465, &d466, &d463, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d465, &d466, &d463, int32(16))
					}
					ctx.FreeDesc(&d466)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d50)
					d467 = d5
					_ = d467
					d468 = d50
					_ = d468
					ctx.StabilizeDescForControlFlow(&d50)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl14 := ctx.ReserveLabel()
					_ = lbl14
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d467 = JITPrepareScmerGoArg(ctx, d467)
					d468 = JITPrepareGoSliceArg(ctx, d468)
					if d468.Loc != LocRegTriple && d468.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d469 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d469.Loc == LocRegPair || d469.Loc == LocStackPair || d469.Loc == LocRegTriple || d469.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d467)
					ctx.SyncDesc(&d468)
					ctx.SyncDesc(&d469)
					d470 = ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d467, d468, d469}, 2)
					d470.NoHeapPointer = false
					ctx.BindReg(d470.Reg, &d470)
					ctx.BindReg(d470.Reg2, &d470)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d470)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d471 JITValueDesc
					if d1.Loc == LocImm {
						d471 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d471 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d471)
					}
					if d471.Loc == LocReg && d1.Loc == LocReg && d471.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d471)
					ctx.EmitStoreToStack(d471, int32(bbs[7].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d471)
					if ps.General {
					}
					ps472 := PhiState{General: ps.General}
					ps472.OverlayValues = make([]JITValueDesc, 472)
					ps472.OverlayValues[1] = d1
					ps472.OverlayValues[2] = d2
					ps472.OverlayValues[3] = d3
					ps472.OverlayValues[4] = d4
					ps472.OverlayValues[5] = d5
					ps472.OverlayValues[6] = d6
					ps472.OverlayValues[7] = d7
					ps472.OverlayValues[8] = d8
					ps472.OverlayValues[9] = d9
					ps472.OverlayValues[10] = d10
					ps472.OverlayValues[11] = d11
					ps472.OverlayValues[40] = d40
					ps472.OverlayValues[41] = d41
					ps472.OverlayValues[42] = d42
					ps472.OverlayValues[43] = d43
					ps472.OverlayValues[44] = d44
					ps472.OverlayValues[45] = d45
					ps472.OverlayValues[46] = d46
					ps472.OverlayValues[47] = d47
					ps472.OverlayValues[48] = d48
					ps472.OverlayValues[49] = d49
					ps472.OverlayValues[50] = d50
					ps472.OverlayValues[51] = d51
					ps472.OverlayValues[52] = d52
					ps472.OverlayValues[108] = d108
					ps472.OverlayValues[109] = d109
					ps472.OverlayValues[110] = d110
					ps472.OverlayValues[111] = d111
					ps472.OverlayValues[112] = d112
					ps472.OverlayValues[177] = d177
					ps472.OverlayValues[178] = d178
					ps472.OverlayValues[179] = d179
					ps472.OverlayValues[250] = d250
					ps472.OverlayValues[251] = d251
					ps472.OverlayValues[252] = d252
					ps472.OverlayValues[255] = d255
					ps472.OverlayValues[332] = d332
					ps472.OverlayValues[333] = d333
					ps472.OverlayValues[334] = d334
					ps472.OverlayValues[335] = d335
					ps472.OverlayValues[336] = d336
					ps472.OverlayValues[338] = d338
					ps472.OverlayValues[339] = d339
					ps472.OverlayValues[340] = d340
					ps472.OverlayValues[342] = d342
					ps472.OverlayValues[343] = d343
					ps472.OverlayValues[344] = d344
					ps472.OverlayValues[345] = d345
					ps472.OverlayValues[346] = d346
					ps472.OverlayValues[349] = d349
					ps472.OverlayValues[454] = d454
					ps472.OverlayValues[455] = d455
					ps472.OverlayValues[456] = d456
					ps472.OverlayValues[457] = d457
					ps472.OverlayValues[459] = d459
					ps472.OverlayValues[460] = d460
					ps472.OverlayValues[461] = d461
					ps472.OverlayValues[462] = d462
					ps472.OverlayValues[463] = d463
					ps472.OverlayValues[464] = d464
					ps472.OverlayValues[465] = d465
					ps472.OverlayValues[466] = d466
					ps472.OverlayValues[467] = d467
					ps472.OverlayValues[468] = d468
					ps472.OverlayValues[469] = d469
					ps472.OverlayValues[470] = d470
					ps472.OverlayValues[471] = d471
					ps472.PhiValues = make([]JITValueDesc, 1)
					if ps472.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps472)
					return result
				}
				ps473 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps473)
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
