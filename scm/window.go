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
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d129 JITValueDesc
				_ = d129
				var d130 JITValueDesc
				_ = d130
				var d131 JITValueDesc
				_ = d131
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
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
				var d142 JITValueDesc
				_ = d142
				var d143 JITValueDesc
				_ = d143
				var d145 JITValueDesc
				_ = d145
				var d146 JITValueDesc
				_ = d146
				var d147 JITValueDesc
				_ = d147
				var d148 JITValueDesc
				_ = d148
				var d149 JITValueDesc
				_ = d149
				var d258 JITValueDesc
				_ = d258
				var d259 JITValueDesc
				_ = d259
				var d260 JITValueDesc
				_ = d260
				var d375 JITValueDesc
				_ = d375
				var d376 JITValueDesc
				_ = d376
				var d377 JITValueDesc
				_ = d377
				var d378 JITValueDesc
				_ = d378
				var d381 JITValueDesc
				_ = d381
				var d504 JITValueDesc
				_ = d504
				var d505 JITValueDesc
				_ = d505
				var d506 JITValueDesc
				_ = d506
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
				var d642 JITValueDesc
				_ = d642
				var d643 JITValueDesc
				_ = d643
				var d788 JITValueDesc
				_ = d788
				var d789 JITValueDesc
				_ = d789
				var d790 JITValueDesc
				_ = d790
				var d791 JITValueDesc
				_ = d791
				var d792 JITValueDesc
				_ = d792
				var d794 JITValueDesc
				_ = d794
				var d795 JITValueDesc
				_ = d795
				var d796 JITValueDesc
				_ = d796
				var d797 JITValueDesc
				_ = d797
				var d798 JITValueDesc
				_ = d798
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
				var d810 JITValueDesc
				_ = d810
				var d811 JITValueDesc
				_ = d811
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [14]BBDescriptor
				bbs[7].PhiBase = int32(phiBase0) + int32(0)
				bbs[7].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 18}}, Count: 1})
				defer ctx.ReleaseRegisterHomes(registerHomes1)
				var r0 Reg
				phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK2 {
					r0 = registerHomes1.Registers[0]
				}
				var d3 JITValueDesc
				if phiHomeOK2 {
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d10)
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
					ctx.EmitJump(d11.Condition, lbl2)
					ctx.FreeDesc(&d10)
					snap14 := d3
					snap15 := d4
					snap16 := d5
					snap17 := d6
					snap18 := d7
					snap19 := d8
					snap20 := d9
					snap21 := d10
					snap22 := d11
					alloc23 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc23)
					d3 = snap14
					d4 = snap15
					d5 = snap16
					d6 = snap17
					d7 = snap18
					d8 = snap19
					d9 = snap20
					d10 = snap21
					d11 = snap22
					ctx.RestoreAllocState(alloc23)
					d3 = snap14
					d4 = snap15
					d5 = snap16
					d6 = snap17
					d7 = snap18
					d8 = snap19
					d9 = snap20
					d10 = snap21
					d11 = snap22
					ps24 := PhiState{General: true}
					ps24.OverlayValues = make([]JITValueDesc, 12)
					ps24.OverlayValues[3] = d3
					ps24.OverlayValues[4] = d4
					ps24.OverlayValues[5] = d5
					ps24.OverlayValues[6] = d6
					ps24.OverlayValues[7] = d7
					ps24.OverlayValues[8] = d8
					ps24.OverlayValues[9] = d9
					ps24.OverlayValues[10] = d10
					ps24.OverlayValues[11] = d11
					ps25 := PhiState{General: true}
					ps25.OverlayValues = make([]JITValueDesc, 12)
					ps25.OverlayValues[3] = d3
					ps25.OverlayValues[4] = d4
					ps25.OverlayValues[5] = d5
					ps25.OverlayValues[6] = d6
					ps25.OverlayValues[7] = d7
					ps25.OverlayValues[8] = d8
					ps25.OverlayValues[9] = d9
					ps25.OverlayValues[10] = d10
					ps25.OverlayValues[11] = d11
					snap26 := d3
					snap27 := d4
					snap28 := d5
					snap29 := d6
					snap30 := d7
					snap31 := d8
					snap32 := d9
					snap33 := d10
					snap34 := d11
					alloc35 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps25)
					}
					ctx.RestoreAllocState(alloc35)
					d3 = snap26
					d4 = snap27
					d5 = snap28
					d6 = snap29
					d7 = snap30
					d8 = snap31
					d9 = snap32
					d10 = snap33
					d11 = snap34
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps24)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
					d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d38 = ctx.EmitSliceElementAddress(&d5, &d36, 16)
					ctx.EnsureDesc(&d38)
					r2 := ctx.AllocRegExcept(d38.Reg)
					ctx.EmitMovRegMem(r2, d38.Reg, 8)
					ctx.EmitMovRegMem(d38.Reg, d38.Reg, 0)
					d37 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d38.Reg, Reg2: r2}
					ctx.BindReg(d38.Reg, &d37)
					ctx.BindReg(r2, &d37)
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
					d41 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					d43 = ctx.EmitSliceElementAddress(&d5, &d41, 16)
					ctx.EnsureDesc(&d43)
					r3 := ctx.AllocRegExcept(d43.Reg)
					ctx.EmitMovRegMem(r3, d43.Reg, 8)
					ctx.EmitMovRegMem(d43.Reg, d43.Reg, 0)
					d42 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d43.Reg, Reg2: r3}
					ctx.BindReg(d43.Reg, &d42)
					ctx.BindReg(r3, &d42)
					var d44 JITValueDesc
					if d42.Loc == LocImm {
						d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d42.Imm.Int())}
					} else if d42.Type == tagInt && d42.Loc == LocRegPair {
						ctx.FreeReg(d42.Reg)
						d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d42.Reg2}
						ctx.BindReg(d42.Reg2, &d44)
						ctx.BindReg(d42.Reg2, &d44)
					} else if d42.Type == tagInt && d42.Loc == LocReg {
						d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d42.Reg}
						ctx.BindReg(d42.Reg, &d44)
						ctx.BindReg(d42.Reg, &d44)
					} else {
						d44 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d42}, 1)
						d44.Type = tagInt
						ctx.BindReg(d44.Reg, &d44)
					}
					ctx.FreeDesc(&d42)
					ctx.EnsureDesc(&d44)
					ctx.EnsureDesc(&d44)
					ctx.StabilizeDescForControlFlow(&d44)
					d46 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					d48 = ctx.EmitSliceElementAddress(&d5, &d46, 16)
					ctx.EnsureDesc(&d48)
					r4 := ctx.AllocRegExcept(d48.Reg)
					ctx.EmitMovRegMem(r4, d48.Reg, 8)
					ctx.EmitMovRegMem(d48.Reg, d48.Reg, 0)
					d47 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d48.Reg, Reg2: r4}
					ctx.BindReg(d48.Reg, &d47)
					ctx.BindReg(r4, &d47)
					var d49 JITValueDesc
					if d47.Loc == LocImm {
						d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d47.Imm.Int())}
					} else if d47.Type == tagInt && d47.Loc == LocRegPair {
						ctx.FreeReg(d47.Reg)
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg2}
						ctx.BindReg(d47.Reg2, &d49)
						ctx.BindReg(d47.Reg2, &d49)
					} else if d47.Type == tagInt && d47.Loc == LocReg {
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg}
						ctx.BindReg(d47.Reg, &d49)
						ctx.BindReg(d47.Reg, &d49)
					} else {
						d49 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d47}, 1)
						d49.Type = tagInt
						ctx.BindReg(d49.Reg, &d49)
					}
					ctx.FreeDesc(&d47)
					ctx.EnsureDesc(&d49)
					ctx.EnsureDesc(&d49)
					ctx.StabilizeDescForControlFlow(&d49)
					d51 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(3)}
					var d52 JITValueDesc
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
						d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2}
						ctx.BindReg(d5.Reg2, &d52)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d52)
					var d54 JITValueDesc
					if d52.Loc == LocImm && d51.Loc == LocImm {
						d54 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d52.Imm.Int() - d51.Imm.Int())}
					} else {
						r5 := ctx.AllocReg()
						if d52.Loc == LocImm {
							ctx.EmitMovRegImm64(r5, uint64(d52.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r5, d52.Reg)
						}
						if d51.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d51.Imm.Int()))
							ctx.EmitSubInt64(r5, RegR11)
						} else {
							ctx.EmitSubInt64(r5, d51.Reg)
						}
						d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d54)
					}
					var d55 JITValueDesc
					r6 := ctx.EmitSliceDataAfterLow(&d5, &d51, 16)
					d55 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
					ctx.BindReg(r6, &d55)
					ctx.BindReg(r6, &d55)
					var d56 JITValueDesc
					var r7 Reg
					var r8 Reg
					ctx.SyncDesc(&d55)
					ctx.EnsureDesc(&d55)
					if d55.Loc == LocImm {
						r7 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, uint64(d55.Imm.Int()))
					} else {
						r7 = d55.Reg
					}
					ctx.ProtectReg(r7)
					ctx.SyncDesc(&d54)
					ctx.EnsureDesc(&d54)
					if d54.Loc == LocImm {
						r8 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r8, uint64(d54.Imm.Int()))
					} else {
						r8 = d54.Reg
					}
					ctx.ProtectReg(r8)
					r9 := ctx.EmitSliceCapAfterLow(&d5, &d51, r7, r8)
					ctx.UnprotectReg(r8)
					ctx.UnprotectReg(r7)
					d56 = JITValueDesc{Loc: LocRegTriple, Reg: r7, Reg2: r8, Reg3: r9}
					ctx.BindReg(r7, &d56)
					ctx.BindReg(r8, &d56)
					ctx.BindReg(r9, &d56)
					ctx.BindReg(r7, &d56)
					ctx.BindReg(r8, &d56)
					ctx.BindReg(r9, &d56)
					ctx.StabilizeDescForControlFlow(&d56)
					ctx.EnsureDesc(&d49)
					var d57 JITValueDesc
					if d49.Loc == LocImm {
						d57 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d49.Imm.Int() <= 0)}
					} else {
						r10 := ctx.AllocRegExcept(d49.Reg)
						ctx.EmitCmpRegImm32(d49.Reg, 0)
						d57 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondSignedLessOrEqual}
						ctx.BindReg(r10, &d57)
					}
					d58 = d57
					ctx.EnsureDesc(&d58)
					if d58.Loc != LocImm && d58.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d58.Loc == LocImm {
						if d58.Imm.Bool() {
							if ps.General {
							}
							ps59 := PhiState{General: ps.General}
							ps59.OverlayValues = make([]JITValueDesc, 59)
							ps59.OverlayValues[3] = d3
							ps59.OverlayValues[4] = d4
							ps59.OverlayValues[5] = d5
							ps59.OverlayValues[6] = d6
							ps59.OverlayValues[7] = d7
							ps59.OverlayValues[8] = d8
							ps59.OverlayValues[9] = d9
							ps59.OverlayValues[10] = d10
							ps59.OverlayValues[11] = d11
							ps59.OverlayValues[36] = d36
							ps59.OverlayValues[37] = d37
							ps59.OverlayValues[38] = d38
							ps59.OverlayValues[39] = d39
							ps59.OverlayValues[40] = d40
							ps59.OverlayValues[41] = d41
							ps59.OverlayValues[42] = d42
							ps59.OverlayValues[43] = d43
							ps59.OverlayValues[44] = d44
							ps59.OverlayValues[45] = d45
							ps59.OverlayValues[46] = d46
							ps59.OverlayValues[47] = d47
							ps59.OverlayValues[48] = d48
							ps59.OverlayValues[49] = d49
							ps59.OverlayValues[50] = d50
							ps59.OverlayValues[51] = d51
							ps59.OverlayValues[52] = d52
							ps59.OverlayValues[53] = d53
							ps59.OverlayValues[54] = d54
							ps59.OverlayValues[55] = d55
							ps59.OverlayValues[56] = d56
							ps59.OverlayValues[57] = d57
							ps59.OverlayValues[58] = d58
							return bbs[3].RenderPS(ps59)
						}
						if ps.General {
						}
						ps60 := PhiState{General: ps.General}
						ps60.OverlayValues = make([]JITValueDesc, 59)
						ps60.OverlayValues[3] = d3
						ps60.OverlayValues[4] = d4
						ps60.OverlayValues[5] = d5
						ps60.OverlayValues[6] = d6
						ps60.OverlayValues[7] = d7
						ps60.OverlayValues[8] = d8
						ps60.OverlayValues[9] = d9
						ps60.OverlayValues[10] = d10
						ps60.OverlayValues[11] = d11
						ps60.OverlayValues[36] = d36
						ps60.OverlayValues[37] = d37
						ps60.OverlayValues[38] = d38
						ps60.OverlayValues[39] = d39
						ps60.OverlayValues[40] = d40
						ps60.OverlayValues[41] = d41
						ps60.OverlayValues[42] = d42
						ps60.OverlayValues[43] = d43
						ps60.OverlayValues[44] = d44
						ps60.OverlayValues[45] = d45
						ps60.OverlayValues[46] = d46
						ps60.OverlayValues[47] = d47
						ps60.OverlayValues[48] = d48
						ps60.OverlayValues[49] = d49
						ps60.OverlayValues[50] = d50
						ps60.OverlayValues[51] = d51
						ps60.OverlayValues[52] = d52
						ps60.OverlayValues[53] = d53
						ps60.OverlayValues[54] = d54
						ps60.OverlayValues[55] = d55
						ps60.OverlayValues[56] = d56
						ps60.OverlayValues[57] = d57
						ps60.OverlayValues[58] = d58
						return bbs[6].RenderPS(ps60)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					ctx.EmitJump(d58.Condition, lbl4)
					ctx.FreeDesc(&d57)
					snap61 := d3
					snap62 := d4
					snap63 := d5
					snap64 := d6
					snap65 := d7
					snap66 := d8
					snap67 := d9
					snap68 := d10
					snap69 := d11
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
					snap91 := d57
					snap92 := d58
					alloc93 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc93)
					d3 = snap61
					d4 = snap62
					d5 = snap63
					d6 = snap64
					d7 = snap65
					d8 = snap66
					d9 = snap67
					d10 = snap68
					d11 = snap69
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
					d57 = snap91
					d58 = snap92
					ctx.RestoreAllocState(alloc93)
					d3 = snap61
					d4 = snap62
					d5 = snap63
					d6 = snap64
					d7 = snap65
					d8 = snap66
					d9 = snap67
					d10 = snap68
					d11 = snap69
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
					d57 = snap91
					d58 = snap92
					ps94 := PhiState{General: true}
					ps94.OverlayValues = make([]JITValueDesc, 59)
					ps94.OverlayValues[3] = d3
					ps94.OverlayValues[4] = d4
					ps94.OverlayValues[5] = d5
					ps94.OverlayValues[6] = d6
					ps94.OverlayValues[7] = d7
					ps94.OverlayValues[8] = d8
					ps94.OverlayValues[9] = d9
					ps94.OverlayValues[10] = d10
					ps94.OverlayValues[11] = d11
					ps94.OverlayValues[36] = d36
					ps94.OverlayValues[37] = d37
					ps94.OverlayValues[38] = d38
					ps94.OverlayValues[39] = d39
					ps94.OverlayValues[40] = d40
					ps94.OverlayValues[41] = d41
					ps94.OverlayValues[42] = d42
					ps94.OverlayValues[43] = d43
					ps94.OverlayValues[44] = d44
					ps94.OverlayValues[45] = d45
					ps94.OverlayValues[46] = d46
					ps94.OverlayValues[47] = d47
					ps94.OverlayValues[48] = d48
					ps94.OverlayValues[49] = d49
					ps94.OverlayValues[50] = d50
					ps94.OverlayValues[51] = d51
					ps94.OverlayValues[52] = d52
					ps94.OverlayValues[53] = d53
					ps94.OverlayValues[54] = d54
					ps94.OverlayValues[55] = d55
					ps94.OverlayValues[56] = d56
					ps94.OverlayValues[57] = d57
					ps94.OverlayValues[58] = d58
					ps95 := PhiState{General: true}
					ps95.OverlayValues = make([]JITValueDesc, 59)
					ps95.OverlayValues[3] = d3
					ps95.OverlayValues[4] = d4
					ps95.OverlayValues[5] = d5
					ps95.OverlayValues[6] = d6
					ps95.OverlayValues[7] = d7
					ps95.OverlayValues[8] = d8
					ps95.OverlayValues[9] = d9
					ps95.OverlayValues[10] = d10
					ps95.OverlayValues[11] = d11
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
					snap96 := d3
					snap97 := d4
					snap98 := d5
					snap99 := d6
					snap100 := d7
					snap101 := d8
					snap102 := d9
					snap103 := d10
					snap104 := d11
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
					snap126 := d57
					snap127 := d58
					alloc128 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps95)
					}
					ctx.RestoreAllocState(alloc128)
					d3 = snap96
					d4 = snap97
					d5 = snap98
					d6 = snap99
					d7 = snap100
					d8 = snap101
					d9 = snap102
					d10 = snap103
					d11 = snap104
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
					d57 = snap126
					d58 = snap127
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps94)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
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
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d49)
					var d129 JITValueDesc
					ctx.EnsureDesc(&d56)
					if d56.Loc == LocRegPair || d56.Loc == LocRegTriple {
						d129 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d56.Reg2}
						ctx.BindReg(d56.Reg2, &d129)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d49)
					ctx.EnsureDesc(&d129)
					var d131 JITValueDesc
					if d129.Loc == LocImm && d49.Loc == LocImm {
						d131 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d129.Imm.Int() - d49.Imm.Int())}
					} else {
						r11 := ctx.AllocReg()
						if d129.Loc == LocImm {
							ctx.EmitMovRegImm64(r11, uint64(d129.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r11, d129.Reg)
						}
						if d49.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d49.Imm.Int()))
							ctx.EmitSubInt64(r11, RegR11)
						} else {
							ctx.EmitSubInt64(r11, d49.Reg)
						}
						d131 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
						ctx.BindReg(r11, &d131)
					}
					var d132 JITValueDesc
					r12 := ctx.EmitSliceDataAfterLow(&d56, &d49, 16)
					d132 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
					ctx.BindReg(r12, &d132)
					ctx.BindReg(r12, &d132)
					var d133 JITValueDesc
					var r13 Reg
					var r14 Reg
					ctx.SyncDesc(&d132)
					ctx.EnsureDesc(&d132)
					if d132.Loc == LocImm {
						r13 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r13, uint64(d132.Imm.Int()))
					} else {
						r13 = d132.Reg
					}
					ctx.ProtectReg(r13)
					ctx.SyncDesc(&d131)
					ctx.EnsureDesc(&d131)
					if d131.Loc == LocImm {
						r14 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r14, uint64(d131.Imm.Int()))
					} else {
						r14 = d131.Reg
					}
					ctx.ProtectReg(r14)
					r15 := ctx.EmitSliceCapAfterLow(&d56, &d49, r13, r14)
					ctx.UnprotectReg(r14)
					ctx.UnprotectReg(r13)
					d133 = JITValueDesc{Loc: LocRegTriple, Reg: r13, Reg2: r14, Reg3: r15}
					ctx.BindReg(r13, &d133)
					ctx.BindReg(r14, &d133)
					ctx.BindReg(r15, &d133)
					ctx.BindReg(r13, &d133)
					ctx.BindReg(r14, &d133)
					ctx.BindReg(r15, &d133)
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d133)
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d133)
					callResults134 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d56, d133}, []uint8{1}, []uint8{0})
					d135 = callResults134[0]
					d135.Type = tagInt
					var d136 JITValueDesc
					if d56.SliceSizeKnown {
						d136 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d56.KnownSliceLen))}
					} else if d56.Loc == LocImm {
						d136 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d56.StackOff))}
					} else if d56.Loc == LocStackTriple {
						d136 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d56.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d56)
						if d56.Loc == LocRegPair || d56.Loc == LocRegTriple {
							d136 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d56.Reg2, ID: 0}
						} else if d56.Loc == LocReg {
							d136 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d56.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d136)
					ctx.EnsureDesc(&d49)
					ctx.EnsureDescsTogether(&d136, &d49)
					var d137 JITValueDesc
					if d136.Loc == LocImm && d49.Loc == LocImm {
						d137 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d136.Imm.Int() - d49.Imm.Int())}
					} else if d49.Loc == LocImm && d49.Imm.Int() == 0 {
						r16 := ctx.AllocRegExcept(d136.Reg)
						ctx.EmitMovRegReg(r16, d136.Reg)
						d137 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
						ctx.BindReg(r16, &d137)
					} else if d136.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d49.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d136.Imm.Int()))
						ctx.EmitSubInt64(scratch, d49.Reg)
						d137 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d137)
					} else if d49.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d136.Reg)
						ctx.EmitMovRegReg(scratch, d136.Reg)
						if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d49.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d49.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d137 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d137)
					} else {
						r17 := ctx.AllocRegExcept(d136.Reg, d49.Reg)
						ctx.EmitMovRegReg(r17, d136.Reg)
						ctx.EmitSubInt64(r17, d49.Reg)
						d137 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
						ctx.BindReg(r17, &d137)
					}
					if d137.Loc == LocReg && d136.Loc == LocReg && d137.Reg == d136.Reg {
						ctx.TransferReg(d136.Reg)
						d136.Loc = LocNone
					}
					ctx.FreeDesc(&d136)
					ctx.EnsureDesc(&d137)
					var d138 JITValueDesc
					ctx.EnsureDesc(&d56)
					if d56.Loc == LocRegPair || d56.Loc == LocRegTriple {
						d138 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d56.Reg2}
						ctx.BindReg(d56.Reg2, &d138)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d137)
					ctx.EnsureDesc(&d138)
					var d140 JITValueDesc
					if d138.Loc == LocImm && d137.Loc == LocImm {
						d140 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d138.Imm.Int() - d137.Imm.Int())}
					} else {
						r18 := ctx.AllocReg()
						if d138.Loc == LocImm {
							ctx.EmitMovRegImm64(r18, uint64(d138.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r18, d138.Reg)
						}
						if d137.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d137.Imm.Int()))
							ctx.EmitSubInt64(r18, RegR11)
						} else {
							ctx.EmitSubInt64(r18, d137.Reg)
						}
						d140 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d140)
					}
					var d141 JITValueDesc
					r19 := ctx.EmitSliceDataAfterLow(&d56, &d137, 16)
					d141 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
					ctx.BindReg(r19, &d141)
					ctx.BindReg(r19, &d141)
					var d142 JITValueDesc
					var r20 Reg
					var r21 Reg
					ctx.SyncDesc(&d141)
					ctx.EnsureDesc(&d141)
					if d141.Loc == LocImm {
						r20 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r20, uint64(d141.Imm.Int()))
					} else {
						r20 = d141.Reg
					}
					ctx.ProtectReg(r20)
					ctx.SyncDesc(&d140)
					ctx.EnsureDesc(&d140)
					if d140.Loc == LocImm {
						r21 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r21, uint64(d140.Imm.Int()))
					} else {
						r21 = d140.Reg
					}
					ctx.ProtectReg(r21)
					r22 := ctx.EmitSliceCapAfterLow(&d56, &d137, r20, r21)
					ctx.UnprotectReg(r21)
					ctx.UnprotectReg(r20)
					d142 = JITValueDesc{Loc: LocRegTriple, Reg: r20, Reg2: r21, Reg3: r22}
					ctx.BindReg(r20, &d142)
					ctx.BindReg(r21, &d142)
					ctx.BindReg(r22, &d142)
					ctx.BindReg(r20, &d142)
					ctx.BindReg(r21, &d142)
					ctx.BindReg(r22, &d142)
					ctx.StabilizeDescForControlFlow(&d142)
					ctx.FreeDesc(&d137)
					var d143 JITValueDesc
					if d142.SliceSizeKnown {
						d143 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d142.KnownSliceLen))}
					} else if d142.Loc == LocImm {
						d143 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d142.StackOff))}
					} else if d142.Loc == LocStackTriple {
						d143 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d142.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d142)
						if d142.Loc == LocRegPair || d142.Loc == LocRegTriple {
							d143 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d142.Reg2, ID: 0}
						} else if d142.Loc == LocReg {
							d143 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d142.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d143)
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[7].PhiBase)+int32(0))
						}
					}
					ps144 := PhiState{General: ps.General}
					ps144.OverlayValues = make([]JITValueDesc, 144)
					ps144.OverlayValues[3] = d3
					ps144.OverlayValues[4] = d4
					ps144.OverlayValues[5] = d5
					ps144.OverlayValues[6] = d6
					ps144.OverlayValues[7] = d7
					ps144.OverlayValues[8] = d8
					ps144.OverlayValues[9] = d9
					ps144.OverlayValues[10] = d10
					ps144.OverlayValues[11] = d11
					ps144.OverlayValues[36] = d36
					ps144.OverlayValues[37] = d37
					ps144.OverlayValues[38] = d38
					ps144.OverlayValues[39] = d39
					ps144.OverlayValues[40] = d40
					ps144.OverlayValues[41] = d41
					ps144.OverlayValues[42] = d42
					ps144.OverlayValues[43] = d43
					ps144.OverlayValues[44] = d44
					ps144.OverlayValues[45] = d45
					ps144.OverlayValues[46] = d46
					ps144.OverlayValues[47] = d47
					ps144.OverlayValues[48] = d48
					ps144.OverlayValues[49] = d49
					ps144.OverlayValues[50] = d50
					ps144.OverlayValues[51] = d51
					ps144.OverlayValues[52] = d52
					ps144.OverlayValues[53] = d53
					ps144.OverlayValues[54] = d54
					ps144.OverlayValues[55] = d55
					ps144.OverlayValues[56] = d56
					ps144.OverlayValues[57] = d57
					ps144.OverlayValues[58] = d58
					ps144.OverlayValues[129] = d129
					ps144.OverlayValues[130] = d130
					ps144.OverlayValues[131] = d131
					ps144.OverlayValues[132] = d132
					ps144.OverlayValues[133] = d133
					ps144.OverlayValues[135] = d135
					ps144.OverlayValues[136] = d136
					ps144.OverlayValues[137] = d137
					ps144.OverlayValues[138] = d138
					ps144.OverlayValues[139] = d139
					ps144.OverlayValues[140] = d140
					ps144.OverlayValues[141] = d141
					ps144.OverlayValues[142] = d142
					ps144.OverlayValues[143] = d143
					ps144.PhiValues = make([]JITValueDesc, 1)
					d145 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps144.PhiValues[0] = d145
					if ps144.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps144)
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
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
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
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
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
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					ctx.ReclaimUntrackedRegs()
					var d146 JITValueDesc
					if d56.SliceSizeKnown {
						d146 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d56.KnownSliceLen))}
					} else if d56.Loc == LocImm {
						d146 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d56.StackOff))}
					} else if d56.Loc == LocStackTriple {
						d146 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d56.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d56)
						if d56.Loc == LocRegPair || d56.Loc == LocRegTriple {
							d146 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d56.Reg2, ID: 0}
						} else if d56.Loc == LocReg {
							d146 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d56.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d146)
					ctx.EnsureDesc(&d49)
					ctx.EnsureDescsTogether(&d146, &d49)
					var d147 JITValueDesc
					if d146.Loc == LocImm && d49.Loc == LocImm {
						d147 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d146.Imm.Int() % d49.Imm.Int())}
					} else {
						d147 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntRem), []JITValueDesc{d146, d49}, 1)
					}
					if d147.Loc == LocReg && d146.Loc == LocReg && d147.Reg == d146.Reg {
						ctx.TransferReg(d146.Reg)
						d146.Loc = LocNone
					}
					ctx.FreeDesc(&d146)
					ctx.FreeDesc(&d49)
					ctx.EnsureDesc(&d147)
					var d148 JITValueDesc
					if d147.Loc == LocImm {
						d148 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d147.Imm.Int() != 0)}
					} else {
						r23 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d147.Reg, 0)
						d148 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r23, Condition: CondNotEqual}
						ctx.BindReg(r23, &d148)
					}
					ctx.FreeDesc(&d147)
					d149 = d148
					ctx.EnsureDesc(&d149)
					if d149.Loc != LocImm && d149.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d149.Loc == LocImm {
						if d149.Imm.Bool() {
							if ps.General {
							}
							ps150 := PhiState{General: ps.General}
							ps150.OverlayValues = make([]JITValueDesc, 150)
							ps150.OverlayValues[3] = d3
							ps150.OverlayValues[4] = d4
							ps150.OverlayValues[5] = d5
							ps150.OverlayValues[6] = d6
							ps150.OverlayValues[7] = d7
							ps150.OverlayValues[8] = d8
							ps150.OverlayValues[9] = d9
							ps150.OverlayValues[10] = d10
							ps150.OverlayValues[11] = d11
							ps150.OverlayValues[36] = d36
							ps150.OverlayValues[37] = d37
							ps150.OverlayValues[38] = d38
							ps150.OverlayValues[39] = d39
							ps150.OverlayValues[40] = d40
							ps150.OverlayValues[41] = d41
							ps150.OverlayValues[42] = d42
							ps150.OverlayValues[43] = d43
							ps150.OverlayValues[44] = d44
							ps150.OverlayValues[45] = d45
							ps150.OverlayValues[46] = d46
							ps150.OverlayValues[47] = d47
							ps150.OverlayValues[48] = d48
							ps150.OverlayValues[49] = d49
							ps150.OverlayValues[50] = d50
							ps150.OverlayValues[51] = d51
							ps150.OverlayValues[52] = d52
							ps150.OverlayValues[53] = d53
							ps150.OverlayValues[54] = d54
							ps150.OverlayValues[55] = d55
							ps150.OverlayValues[56] = d56
							ps150.OverlayValues[57] = d57
							ps150.OverlayValues[58] = d58
							ps150.OverlayValues[129] = d129
							ps150.OverlayValues[130] = d130
							ps150.OverlayValues[131] = d131
							ps150.OverlayValues[132] = d132
							ps150.OverlayValues[133] = d133
							ps150.OverlayValues[135] = d135
							ps150.OverlayValues[136] = d136
							ps150.OverlayValues[137] = d137
							ps150.OverlayValues[138] = d138
							ps150.OverlayValues[139] = d139
							ps150.OverlayValues[140] = d140
							ps150.OverlayValues[141] = d141
							ps150.OverlayValues[142] = d142
							ps150.OverlayValues[143] = d143
							ps150.OverlayValues[145] = d145
							ps150.OverlayValues[146] = d146
							ps150.OverlayValues[147] = d147
							ps150.OverlayValues[148] = d148
							ps150.OverlayValues[149] = d149
							return bbs[3].RenderPS(ps150)
						}
						if ps.General {
						}
						ps151 := PhiState{General: ps.General}
						ps151.OverlayValues = make([]JITValueDesc, 150)
						ps151.OverlayValues[3] = d3
						ps151.OverlayValues[4] = d4
						ps151.OverlayValues[5] = d5
						ps151.OverlayValues[6] = d6
						ps151.OverlayValues[7] = d7
						ps151.OverlayValues[8] = d8
						ps151.OverlayValues[9] = d9
						ps151.OverlayValues[10] = d10
						ps151.OverlayValues[11] = d11
						ps151.OverlayValues[36] = d36
						ps151.OverlayValues[37] = d37
						ps151.OverlayValues[38] = d38
						ps151.OverlayValues[39] = d39
						ps151.OverlayValues[40] = d40
						ps151.OverlayValues[41] = d41
						ps151.OverlayValues[42] = d42
						ps151.OverlayValues[43] = d43
						ps151.OverlayValues[44] = d44
						ps151.OverlayValues[45] = d45
						ps151.OverlayValues[46] = d46
						ps151.OverlayValues[47] = d47
						ps151.OverlayValues[48] = d48
						ps151.OverlayValues[49] = d49
						ps151.OverlayValues[50] = d50
						ps151.OverlayValues[51] = d51
						ps151.OverlayValues[52] = d52
						ps151.OverlayValues[53] = d53
						ps151.OverlayValues[54] = d54
						ps151.OverlayValues[55] = d55
						ps151.OverlayValues[56] = d56
						ps151.OverlayValues[57] = d57
						ps151.OverlayValues[58] = d58
						ps151.OverlayValues[129] = d129
						ps151.OverlayValues[130] = d130
						ps151.OverlayValues[131] = d131
						ps151.OverlayValues[132] = d132
						ps151.OverlayValues[133] = d133
						ps151.OverlayValues[135] = d135
						ps151.OverlayValues[136] = d136
						ps151.OverlayValues[137] = d137
						ps151.OverlayValues[138] = d138
						ps151.OverlayValues[139] = d139
						ps151.OverlayValues[140] = d140
						ps151.OverlayValues[141] = d141
						ps151.OverlayValues[142] = d142
						ps151.OverlayValues[143] = d143
						ps151.OverlayValues[145] = d145
						ps151.OverlayValues[146] = d146
						ps151.OverlayValues[147] = d147
						ps151.OverlayValues[148] = d148
						ps151.OverlayValues[149] = d149
						return bbs[4].RenderPS(ps151)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					ctx.EmitJump(d149.Condition, lbl4)
					ctx.FreeDesc(&d148)
					snap152 := d3
					snap153 := d4
					snap154 := d5
					snap155 := d6
					snap156 := d7
					snap157 := d8
					snap158 := d9
					snap159 := d10
					snap160 := d11
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
					snap182 := d57
					snap183 := d58
					snap184 := d129
					snap185 := d130
					snap186 := d131
					snap187 := d132
					snap188 := d133
					snap189 := d135
					snap190 := d136
					snap191 := d137
					snap192 := d138
					snap193 := d139
					snap194 := d140
					snap195 := d141
					snap196 := d142
					snap197 := d143
					snap198 := d145
					snap199 := d146
					snap200 := d147
					snap201 := d148
					snap202 := d149
					alloc203 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc203)
					d3 = snap152
					d4 = snap153
					d5 = snap154
					d6 = snap155
					d7 = snap156
					d8 = snap157
					d9 = snap158
					d10 = snap159
					d11 = snap160
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
					d57 = snap182
					d58 = snap183
					d129 = snap184
					d130 = snap185
					d131 = snap186
					d132 = snap187
					d133 = snap188
					d135 = snap189
					d136 = snap190
					d137 = snap191
					d138 = snap192
					d139 = snap193
					d140 = snap194
					d141 = snap195
					d142 = snap196
					d143 = snap197
					d145 = snap198
					d146 = snap199
					d147 = snap200
					d148 = snap201
					d149 = snap202
					ctx.RestoreAllocState(alloc203)
					d3 = snap152
					d4 = snap153
					d5 = snap154
					d6 = snap155
					d7 = snap156
					d8 = snap157
					d9 = snap158
					d10 = snap159
					d11 = snap160
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
					d57 = snap182
					d58 = snap183
					d129 = snap184
					d130 = snap185
					d131 = snap186
					d132 = snap187
					d133 = snap188
					d135 = snap189
					d136 = snap190
					d137 = snap191
					d138 = snap192
					d139 = snap193
					d140 = snap194
					d141 = snap195
					d142 = snap196
					d143 = snap197
					d145 = snap198
					d146 = snap199
					d147 = snap200
					d148 = snap201
					d149 = snap202
					ps204 := PhiState{General: true}
					ps204.OverlayValues = make([]JITValueDesc, 150)
					ps204.OverlayValues[3] = d3
					ps204.OverlayValues[4] = d4
					ps204.OverlayValues[5] = d5
					ps204.OverlayValues[6] = d6
					ps204.OverlayValues[7] = d7
					ps204.OverlayValues[8] = d8
					ps204.OverlayValues[9] = d9
					ps204.OverlayValues[10] = d10
					ps204.OverlayValues[11] = d11
					ps204.OverlayValues[36] = d36
					ps204.OverlayValues[37] = d37
					ps204.OverlayValues[38] = d38
					ps204.OverlayValues[39] = d39
					ps204.OverlayValues[40] = d40
					ps204.OverlayValues[41] = d41
					ps204.OverlayValues[42] = d42
					ps204.OverlayValues[43] = d43
					ps204.OverlayValues[44] = d44
					ps204.OverlayValues[45] = d45
					ps204.OverlayValues[46] = d46
					ps204.OverlayValues[47] = d47
					ps204.OverlayValues[48] = d48
					ps204.OverlayValues[49] = d49
					ps204.OverlayValues[50] = d50
					ps204.OverlayValues[51] = d51
					ps204.OverlayValues[52] = d52
					ps204.OverlayValues[53] = d53
					ps204.OverlayValues[54] = d54
					ps204.OverlayValues[55] = d55
					ps204.OverlayValues[56] = d56
					ps204.OverlayValues[57] = d57
					ps204.OverlayValues[58] = d58
					ps204.OverlayValues[129] = d129
					ps204.OverlayValues[130] = d130
					ps204.OverlayValues[131] = d131
					ps204.OverlayValues[132] = d132
					ps204.OverlayValues[133] = d133
					ps204.OverlayValues[135] = d135
					ps204.OverlayValues[136] = d136
					ps204.OverlayValues[137] = d137
					ps204.OverlayValues[138] = d138
					ps204.OverlayValues[139] = d139
					ps204.OverlayValues[140] = d140
					ps204.OverlayValues[141] = d141
					ps204.OverlayValues[142] = d142
					ps204.OverlayValues[143] = d143
					ps204.OverlayValues[145] = d145
					ps204.OverlayValues[146] = d146
					ps204.OverlayValues[147] = d147
					ps204.OverlayValues[148] = d148
					ps204.OverlayValues[149] = d149
					ps205 := PhiState{General: true}
					ps205.OverlayValues = make([]JITValueDesc, 150)
					ps205.OverlayValues[3] = d3
					ps205.OverlayValues[4] = d4
					ps205.OverlayValues[5] = d5
					ps205.OverlayValues[6] = d6
					ps205.OverlayValues[7] = d7
					ps205.OverlayValues[8] = d8
					ps205.OverlayValues[9] = d9
					ps205.OverlayValues[10] = d10
					ps205.OverlayValues[11] = d11
					ps205.OverlayValues[36] = d36
					ps205.OverlayValues[37] = d37
					ps205.OverlayValues[38] = d38
					ps205.OverlayValues[39] = d39
					ps205.OverlayValues[40] = d40
					ps205.OverlayValues[41] = d41
					ps205.OverlayValues[42] = d42
					ps205.OverlayValues[43] = d43
					ps205.OverlayValues[44] = d44
					ps205.OverlayValues[45] = d45
					ps205.OverlayValues[46] = d46
					ps205.OverlayValues[47] = d47
					ps205.OverlayValues[48] = d48
					ps205.OverlayValues[49] = d49
					ps205.OverlayValues[50] = d50
					ps205.OverlayValues[51] = d51
					ps205.OverlayValues[52] = d52
					ps205.OverlayValues[53] = d53
					ps205.OverlayValues[54] = d54
					ps205.OverlayValues[55] = d55
					ps205.OverlayValues[56] = d56
					ps205.OverlayValues[57] = d57
					ps205.OverlayValues[58] = d58
					ps205.OverlayValues[129] = d129
					ps205.OverlayValues[130] = d130
					ps205.OverlayValues[131] = d131
					ps205.OverlayValues[132] = d132
					ps205.OverlayValues[133] = d133
					ps205.OverlayValues[135] = d135
					ps205.OverlayValues[136] = d136
					ps205.OverlayValues[137] = d137
					ps205.OverlayValues[138] = d138
					ps205.OverlayValues[139] = d139
					ps205.OverlayValues[140] = d140
					ps205.OverlayValues[141] = d141
					ps205.OverlayValues[142] = d142
					ps205.OverlayValues[143] = d143
					ps205.OverlayValues[145] = d145
					ps205.OverlayValues[146] = d146
					ps205.OverlayValues[147] = d147
					ps205.OverlayValues[148] = d148
					ps205.OverlayValues[149] = d149
					snap206 := d3
					snap207 := d4
					snap208 := d5
					snap209 := d6
					snap210 := d7
					snap211 := d8
					snap212 := d9
					snap213 := d10
					snap214 := d11
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
					snap236 := d57
					snap237 := d58
					snap238 := d129
					snap239 := d130
					snap240 := d131
					snap241 := d132
					snap242 := d133
					snap243 := d135
					snap244 := d136
					snap245 := d137
					snap246 := d138
					snap247 := d139
					snap248 := d140
					snap249 := d141
					snap250 := d142
					snap251 := d143
					snap252 := d145
					snap253 := d146
					snap254 := d147
					snap255 := d148
					snap256 := d149
					alloc257 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps205)
					}
					ctx.RestoreAllocState(alloc257)
					d3 = snap206
					d4 = snap207
					d5 = snap208
					d6 = snap209
					d7 = snap210
					d8 = snap211
					d9 = snap212
					d10 = snap213
					d11 = snap214
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
					d57 = snap236
					d58 = snap237
					d129 = snap238
					d130 = snap239
					d131 = snap240
					d132 = snap241
					d133 = snap242
					d135 = snap243
					d136 = snap244
					d137 = snap245
					d138 = snap246
					d139 = snap247
					d140 = snap248
					d141 = snap249
					d142 = snap250
					d143 = snap251
					d145 = snap252
					d146 = snap253
					d147 = snap254
					d148 = snap255
					d149 = snap256
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps204)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
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
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
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
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					ctx.ReclaimUntrackedRegs()
					var d258 JITValueDesc
					if d56.SliceSizeKnown {
						d258 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d56.KnownSliceLen))}
					} else if d56.Loc == LocImm {
						d258 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d56.StackOff))}
					} else if d56.Loc == LocStackTriple {
						d258 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d56.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d56)
						if d56.Loc == LocRegPair || d56.Loc == LocRegTriple {
							d258 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d56.Reg2, ID: 0}
						} else if d56.Loc == LocReg {
							d258 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d56.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d258)
					var d259 JITValueDesc
					if d258.Loc == LocImm {
						d259 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d258.Imm.Int() == 0)}
					} else {
						r24 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d258.Reg, 0)
						d259 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r24, Condition: CondEqual}
						ctx.BindReg(r24, &d259)
					}
					ctx.FreeDesc(&d258)
					d260 = d259
					ctx.EnsureDesc(&d260)
					if d260.Loc != LocImm && d260.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d260.Loc == LocImm {
						if d260.Imm.Bool() {
							if ps.General {
							}
							ps261 := PhiState{General: ps.General}
							ps261.OverlayValues = make([]JITValueDesc, 261)
							ps261.OverlayValues[3] = d3
							ps261.OverlayValues[4] = d4
							ps261.OverlayValues[5] = d5
							ps261.OverlayValues[6] = d6
							ps261.OverlayValues[7] = d7
							ps261.OverlayValues[8] = d8
							ps261.OverlayValues[9] = d9
							ps261.OverlayValues[10] = d10
							ps261.OverlayValues[11] = d11
							ps261.OverlayValues[36] = d36
							ps261.OverlayValues[37] = d37
							ps261.OverlayValues[38] = d38
							ps261.OverlayValues[39] = d39
							ps261.OverlayValues[40] = d40
							ps261.OverlayValues[41] = d41
							ps261.OverlayValues[42] = d42
							ps261.OverlayValues[43] = d43
							ps261.OverlayValues[44] = d44
							ps261.OverlayValues[45] = d45
							ps261.OverlayValues[46] = d46
							ps261.OverlayValues[47] = d47
							ps261.OverlayValues[48] = d48
							ps261.OverlayValues[49] = d49
							ps261.OverlayValues[50] = d50
							ps261.OverlayValues[51] = d51
							ps261.OverlayValues[52] = d52
							ps261.OverlayValues[53] = d53
							ps261.OverlayValues[54] = d54
							ps261.OverlayValues[55] = d55
							ps261.OverlayValues[56] = d56
							ps261.OverlayValues[57] = d57
							ps261.OverlayValues[58] = d58
							ps261.OverlayValues[129] = d129
							ps261.OverlayValues[130] = d130
							ps261.OverlayValues[131] = d131
							ps261.OverlayValues[132] = d132
							ps261.OverlayValues[133] = d133
							ps261.OverlayValues[135] = d135
							ps261.OverlayValues[136] = d136
							ps261.OverlayValues[137] = d137
							ps261.OverlayValues[138] = d138
							ps261.OverlayValues[139] = d139
							ps261.OverlayValues[140] = d140
							ps261.OverlayValues[141] = d141
							ps261.OverlayValues[142] = d142
							ps261.OverlayValues[143] = d143
							ps261.OverlayValues[145] = d145
							ps261.OverlayValues[146] = d146
							ps261.OverlayValues[147] = d147
							ps261.OverlayValues[148] = d148
							ps261.OverlayValues[149] = d149
							ps261.OverlayValues[258] = d258
							ps261.OverlayValues[259] = d259
							ps261.OverlayValues[260] = d260
							return bbs[3].RenderPS(ps261)
						}
						if ps.General {
						}
						ps262 := PhiState{General: ps.General}
						ps262.OverlayValues = make([]JITValueDesc, 261)
						ps262.OverlayValues[3] = d3
						ps262.OverlayValues[4] = d4
						ps262.OverlayValues[5] = d5
						ps262.OverlayValues[6] = d6
						ps262.OverlayValues[7] = d7
						ps262.OverlayValues[8] = d8
						ps262.OverlayValues[9] = d9
						ps262.OverlayValues[10] = d10
						ps262.OverlayValues[11] = d11
						ps262.OverlayValues[36] = d36
						ps262.OverlayValues[37] = d37
						ps262.OverlayValues[38] = d38
						ps262.OverlayValues[39] = d39
						ps262.OverlayValues[40] = d40
						ps262.OverlayValues[41] = d41
						ps262.OverlayValues[42] = d42
						ps262.OverlayValues[43] = d43
						ps262.OverlayValues[44] = d44
						ps262.OverlayValues[45] = d45
						ps262.OverlayValues[46] = d46
						ps262.OverlayValues[47] = d47
						ps262.OverlayValues[48] = d48
						ps262.OverlayValues[49] = d49
						ps262.OverlayValues[50] = d50
						ps262.OverlayValues[51] = d51
						ps262.OverlayValues[52] = d52
						ps262.OverlayValues[53] = d53
						ps262.OverlayValues[54] = d54
						ps262.OverlayValues[55] = d55
						ps262.OverlayValues[56] = d56
						ps262.OverlayValues[57] = d57
						ps262.OverlayValues[58] = d58
						ps262.OverlayValues[129] = d129
						ps262.OverlayValues[130] = d130
						ps262.OverlayValues[131] = d131
						ps262.OverlayValues[132] = d132
						ps262.OverlayValues[133] = d133
						ps262.OverlayValues[135] = d135
						ps262.OverlayValues[136] = d136
						ps262.OverlayValues[137] = d137
						ps262.OverlayValues[138] = d138
						ps262.OverlayValues[139] = d139
						ps262.OverlayValues[140] = d140
						ps262.OverlayValues[141] = d141
						ps262.OverlayValues[142] = d142
						ps262.OverlayValues[143] = d143
						ps262.OverlayValues[145] = d145
						ps262.OverlayValues[146] = d146
						ps262.OverlayValues[147] = d147
						ps262.OverlayValues[148] = d148
						ps262.OverlayValues[149] = d149
						ps262.OverlayValues[258] = d258
						ps262.OverlayValues[259] = d259
						ps262.OverlayValues[260] = d260
						return bbs[5].RenderPS(ps262)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					ctx.EmitJump(d260.Condition, lbl4)
					ctx.FreeDesc(&d259)
					snap263 := d3
					snap264 := d4
					snap265 := d5
					snap266 := d6
					snap267 := d7
					snap268 := d8
					snap269 := d9
					snap270 := d10
					snap271 := d11
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
					snap293 := d57
					snap294 := d58
					snap295 := d129
					snap296 := d130
					snap297 := d131
					snap298 := d132
					snap299 := d133
					snap300 := d135
					snap301 := d136
					snap302 := d137
					snap303 := d138
					snap304 := d139
					snap305 := d140
					snap306 := d141
					snap307 := d142
					snap308 := d143
					snap309 := d145
					snap310 := d146
					snap311 := d147
					snap312 := d148
					snap313 := d149
					snap314 := d258
					snap315 := d259
					snap316 := d260
					alloc317 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc317)
					d3 = snap263
					d4 = snap264
					d5 = snap265
					d6 = snap266
					d7 = snap267
					d8 = snap268
					d9 = snap269
					d10 = snap270
					d11 = snap271
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
					d57 = snap293
					d58 = snap294
					d129 = snap295
					d130 = snap296
					d131 = snap297
					d132 = snap298
					d133 = snap299
					d135 = snap300
					d136 = snap301
					d137 = snap302
					d138 = snap303
					d139 = snap304
					d140 = snap305
					d141 = snap306
					d142 = snap307
					d143 = snap308
					d145 = snap309
					d146 = snap310
					d147 = snap311
					d148 = snap312
					d149 = snap313
					d258 = snap314
					d259 = snap315
					d260 = snap316
					ctx.RestoreAllocState(alloc317)
					d3 = snap263
					d4 = snap264
					d5 = snap265
					d6 = snap266
					d7 = snap267
					d8 = snap268
					d9 = snap269
					d10 = snap270
					d11 = snap271
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
					d57 = snap293
					d58 = snap294
					d129 = snap295
					d130 = snap296
					d131 = snap297
					d132 = snap298
					d133 = snap299
					d135 = snap300
					d136 = snap301
					d137 = snap302
					d138 = snap303
					d139 = snap304
					d140 = snap305
					d141 = snap306
					d142 = snap307
					d143 = snap308
					d145 = snap309
					d146 = snap310
					d147 = snap311
					d148 = snap312
					d149 = snap313
					d258 = snap314
					d259 = snap315
					d260 = snap316
					ps318 := PhiState{General: true}
					ps318.OverlayValues = make([]JITValueDesc, 261)
					ps318.OverlayValues[3] = d3
					ps318.OverlayValues[4] = d4
					ps318.OverlayValues[5] = d5
					ps318.OverlayValues[6] = d6
					ps318.OverlayValues[7] = d7
					ps318.OverlayValues[8] = d8
					ps318.OverlayValues[9] = d9
					ps318.OverlayValues[10] = d10
					ps318.OverlayValues[11] = d11
					ps318.OverlayValues[36] = d36
					ps318.OverlayValues[37] = d37
					ps318.OverlayValues[38] = d38
					ps318.OverlayValues[39] = d39
					ps318.OverlayValues[40] = d40
					ps318.OverlayValues[41] = d41
					ps318.OverlayValues[42] = d42
					ps318.OverlayValues[43] = d43
					ps318.OverlayValues[44] = d44
					ps318.OverlayValues[45] = d45
					ps318.OverlayValues[46] = d46
					ps318.OverlayValues[47] = d47
					ps318.OverlayValues[48] = d48
					ps318.OverlayValues[49] = d49
					ps318.OverlayValues[50] = d50
					ps318.OverlayValues[51] = d51
					ps318.OverlayValues[52] = d52
					ps318.OverlayValues[53] = d53
					ps318.OverlayValues[54] = d54
					ps318.OverlayValues[55] = d55
					ps318.OverlayValues[56] = d56
					ps318.OverlayValues[57] = d57
					ps318.OverlayValues[58] = d58
					ps318.OverlayValues[129] = d129
					ps318.OverlayValues[130] = d130
					ps318.OverlayValues[131] = d131
					ps318.OverlayValues[132] = d132
					ps318.OverlayValues[133] = d133
					ps318.OverlayValues[135] = d135
					ps318.OverlayValues[136] = d136
					ps318.OverlayValues[137] = d137
					ps318.OverlayValues[138] = d138
					ps318.OverlayValues[139] = d139
					ps318.OverlayValues[140] = d140
					ps318.OverlayValues[141] = d141
					ps318.OverlayValues[142] = d142
					ps318.OverlayValues[143] = d143
					ps318.OverlayValues[145] = d145
					ps318.OverlayValues[146] = d146
					ps318.OverlayValues[147] = d147
					ps318.OverlayValues[148] = d148
					ps318.OverlayValues[149] = d149
					ps318.OverlayValues[258] = d258
					ps318.OverlayValues[259] = d259
					ps318.OverlayValues[260] = d260
					ps319 := PhiState{General: true}
					ps319.OverlayValues = make([]JITValueDesc, 261)
					ps319.OverlayValues[3] = d3
					ps319.OverlayValues[4] = d4
					ps319.OverlayValues[5] = d5
					ps319.OverlayValues[6] = d6
					ps319.OverlayValues[7] = d7
					ps319.OverlayValues[8] = d8
					ps319.OverlayValues[9] = d9
					ps319.OverlayValues[10] = d10
					ps319.OverlayValues[11] = d11
					ps319.OverlayValues[36] = d36
					ps319.OverlayValues[37] = d37
					ps319.OverlayValues[38] = d38
					ps319.OverlayValues[39] = d39
					ps319.OverlayValues[40] = d40
					ps319.OverlayValues[41] = d41
					ps319.OverlayValues[42] = d42
					ps319.OverlayValues[43] = d43
					ps319.OverlayValues[44] = d44
					ps319.OverlayValues[45] = d45
					ps319.OverlayValues[46] = d46
					ps319.OverlayValues[47] = d47
					ps319.OverlayValues[48] = d48
					ps319.OverlayValues[49] = d49
					ps319.OverlayValues[50] = d50
					ps319.OverlayValues[51] = d51
					ps319.OverlayValues[52] = d52
					ps319.OverlayValues[53] = d53
					ps319.OverlayValues[54] = d54
					ps319.OverlayValues[55] = d55
					ps319.OverlayValues[56] = d56
					ps319.OverlayValues[57] = d57
					ps319.OverlayValues[58] = d58
					ps319.OverlayValues[129] = d129
					ps319.OverlayValues[130] = d130
					ps319.OverlayValues[131] = d131
					ps319.OverlayValues[132] = d132
					ps319.OverlayValues[133] = d133
					ps319.OverlayValues[135] = d135
					ps319.OverlayValues[136] = d136
					ps319.OverlayValues[137] = d137
					ps319.OverlayValues[138] = d138
					ps319.OverlayValues[139] = d139
					ps319.OverlayValues[140] = d140
					ps319.OverlayValues[141] = d141
					ps319.OverlayValues[142] = d142
					ps319.OverlayValues[143] = d143
					ps319.OverlayValues[145] = d145
					ps319.OverlayValues[146] = d146
					ps319.OverlayValues[147] = d147
					ps319.OverlayValues[148] = d148
					ps319.OverlayValues[149] = d149
					ps319.OverlayValues[258] = d258
					ps319.OverlayValues[259] = d259
					ps319.OverlayValues[260] = d260
					snap320 := d3
					snap321 := d4
					snap322 := d5
					snap323 := d6
					snap324 := d7
					snap325 := d8
					snap326 := d9
					snap327 := d10
					snap328 := d11
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
					snap350 := d57
					snap351 := d58
					snap352 := d129
					snap353 := d130
					snap354 := d131
					snap355 := d132
					snap356 := d133
					snap357 := d135
					snap358 := d136
					snap359 := d137
					snap360 := d138
					snap361 := d139
					snap362 := d140
					snap363 := d141
					snap364 := d142
					snap365 := d143
					snap366 := d145
					snap367 := d146
					snap368 := d147
					snap369 := d148
					snap370 := d149
					snap371 := d258
					snap372 := d259
					snap373 := d260
					alloc374 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps319)
					}
					ctx.RestoreAllocState(alloc374)
					d3 = snap320
					d4 = snap321
					d5 = snap322
					d6 = snap323
					d7 = snap324
					d8 = snap325
					d9 = snap326
					d10 = snap327
					d11 = snap328
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
					d57 = snap350
					d58 = snap351
					d129 = snap352
					d130 = snap353
					d131 = snap354
					d132 = snap355
					d133 = snap356
					d135 = snap357
					d136 = snap358
					d137 = snap359
					d138 = snap360
					d139 = snap361
					d140 = snap362
					d141 = snap363
					d142 = snap364
					d143 = snap365
					d145 = snap366
					d146 = snap367
					d147 = snap368
					d148 = snap369
					d149 = snap370
					d258 = snap371
					d259 = snap372
					d260 = snap373
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps318)
					}
					return result
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d375 := ps.PhiValues[0]
							ctx.EnsureDesc(&d375)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d375)
							} else {
								ctx.EmitStoreToStack(d375, int32(bbs[7].PhiBase)+int32(0))
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
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
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
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
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
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 259 && ps.OverlayValues[259].Loc != LocNone {
						d259 = ps.OverlayValues[259]
					}
					if len(ps.OverlayValues) > 260 && ps.OverlayValues[260].Loc != LocNone {
						d260 = ps.OverlayValues[260]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					if phiHomeOK2 && d3.Loc == LocReg {
						ctx.BindReg(r0, &d3)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					var d376 JITValueDesc
					if d3.Loc == LocImm {
						d376 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d376 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d376)
					}
					if d376.Loc == LocReg && d3.Loc == LocReg && d376.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d376)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d376)
					ctx.EnsureDesc(&d143)
					ctx.EnsureDescsTogether(&d376, &d143)
					var d377 JITValueDesc
					if d376.Loc == LocImm && d143.Loc == LocImm {
						d377 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d376.Imm.Int() < d143.Imm.Int())}
					} else if d143.Loc == LocImm {
						r25 := ctx.AllocRegExcept(d376.Reg)
						if d143.Imm.Int() >= -2147483648 && d143.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d376.Reg, int32(d143.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d143.Imm.Int()))
							ctx.EmitCmpInt64(d376.Reg, RegR11)
						}
						d377 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r25, Condition: CondSignedLess}
						ctx.BindReg(r25, &d377)
					} else if d376.Loc == LocImm {
						r26 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d376.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d143.Reg)
						d377 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondSignedLess}
						ctx.BindReg(r26, &d377)
					} else {
						r27 := ctx.AllocRegExcept(d376.Reg)
						ctx.EmitCmpInt64(d376.Reg, d143.Reg)
						d377 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r27, Condition: CondSignedLess}
						ctx.BindReg(r27, &d377)
					}
					d378 = d377
					ctx.EnsureDesc(&d378)
					if d378.Loc != LocImm && d378.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d378.Loc == LocImm {
						if d378.Imm.Bool() {
							if ps.General {
							}
							ps379 := PhiState{General: ps.General}
							ps379.OverlayValues = make([]JITValueDesc, 379)
							ps379.OverlayValues[3] = d3
							ps379.OverlayValues[4] = d4
							ps379.OverlayValues[5] = d5
							ps379.OverlayValues[6] = d6
							ps379.OverlayValues[7] = d7
							ps379.OverlayValues[8] = d8
							ps379.OverlayValues[9] = d9
							ps379.OverlayValues[10] = d10
							ps379.OverlayValues[11] = d11
							ps379.OverlayValues[36] = d36
							ps379.OverlayValues[37] = d37
							ps379.OverlayValues[38] = d38
							ps379.OverlayValues[39] = d39
							ps379.OverlayValues[40] = d40
							ps379.OverlayValues[41] = d41
							ps379.OverlayValues[42] = d42
							ps379.OverlayValues[43] = d43
							ps379.OverlayValues[44] = d44
							ps379.OverlayValues[45] = d45
							ps379.OverlayValues[46] = d46
							ps379.OverlayValues[47] = d47
							ps379.OverlayValues[48] = d48
							ps379.OverlayValues[49] = d49
							ps379.OverlayValues[50] = d50
							ps379.OverlayValues[51] = d51
							ps379.OverlayValues[52] = d52
							ps379.OverlayValues[53] = d53
							ps379.OverlayValues[54] = d54
							ps379.OverlayValues[55] = d55
							ps379.OverlayValues[56] = d56
							ps379.OverlayValues[57] = d57
							ps379.OverlayValues[58] = d58
							ps379.OverlayValues[129] = d129
							ps379.OverlayValues[130] = d130
							ps379.OverlayValues[131] = d131
							ps379.OverlayValues[132] = d132
							ps379.OverlayValues[133] = d133
							ps379.OverlayValues[135] = d135
							ps379.OverlayValues[136] = d136
							ps379.OverlayValues[137] = d137
							ps379.OverlayValues[138] = d138
							ps379.OverlayValues[139] = d139
							ps379.OverlayValues[140] = d140
							ps379.OverlayValues[141] = d141
							ps379.OverlayValues[142] = d142
							ps379.OverlayValues[143] = d143
							ps379.OverlayValues[145] = d145
							ps379.OverlayValues[146] = d146
							ps379.OverlayValues[147] = d147
							ps379.OverlayValues[148] = d148
							ps379.OverlayValues[149] = d149
							ps379.OverlayValues[258] = d258
							ps379.OverlayValues[259] = d259
							ps379.OverlayValues[260] = d260
							ps379.OverlayValues[375] = d375
							ps379.OverlayValues[376] = d376
							ps379.OverlayValues[377] = d377
							ps379.OverlayValues[378] = d378
							return bbs[8].RenderPS(ps379)
						}
						if ps.General {
						}
						ps380 := PhiState{General: ps.General}
						ps380.OverlayValues = make([]JITValueDesc, 379)
						ps380.OverlayValues[3] = d3
						ps380.OverlayValues[4] = d4
						ps380.OverlayValues[5] = d5
						ps380.OverlayValues[6] = d6
						ps380.OverlayValues[7] = d7
						ps380.OverlayValues[8] = d8
						ps380.OverlayValues[9] = d9
						ps380.OverlayValues[10] = d10
						ps380.OverlayValues[11] = d11
						ps380.OverlayValues[36] = d36
						ps380.OverlayValues[37] = d37
						ps380.OverlayValues[38] = d38
						ps380.OverlayValues[39] = d39
						ps380.OverlayValues[40] = d40
						ps380.OverlayValues[41] = d41
						ps380.OverlayValues[42] = d42
						ps380.OverlayValues[43] = d43
						ps380.OverlayValues[44] = d44
						ps380.OverlayValues[45] = d45
						ps380.OverlayValues[46] = d46
						ps380.OverlayValues[47] = d47
						ps380.OverlayValues[48] = d48
						ps380.OverlayValues[49] = d49
						ps380.OverlayValues[50] = d50
						ps380.OverlayValues[51] = d51
						ps380.OverlayValues[52] = d52
						ps380.OverlayValues[53] = d53
						ps380.OverlayValues[54] = d54
						ps380.OverlayValues[55] = d55
						ps380.OverlayValues[56] = d56
						ps380.OverlayValues[57] = d57
						ps380.OverlayValues[58] = d58
						ps380.OverlayValues[129] = d129
						ps380.OverlayValues[130] = d130
						ps380.OverlayValues[131] = d131
						ps380.OverlayValues[132] = d132
						ps380.OverlayValues[133] = d133
						ps380.OverlayValues[135] = d135
						ps380.OverlayValues[136] = d136
						ps380.OverlayValues[137] = d137
						ps380.OverlayValues[138] = d138
						ps380.OverlayValues[139] = d139
						ps380.OverlayValues[140] = d140
						ps380.OverlayValues[141] = d141
						ps380.OverlayValues[142] = d142
						ps380.OverlayValues[143] = d143
						ps380.OverlayValues[145] = d145
						ps380.OverlayValues[146] = d146
						ps380.OverlayValues[147] = d147
						ps380.OverlayValues[148] = d148
						ps380.OverlayValues[149] = d149
						ps380.OverlayValues[258] = d258
						ps380.OverlayValues[259] = d259
						ps380.OverlayValues[260] = d260
						ps380.OverlayValues[375] = d375
						ps380.OverlayValues[376] = d376
						ps380.OverlayValues[377] = d377
						ps380.OverlayValues[378] = d378
						return bbs[9].RenderPS(ps380)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d381 := ps.PhiValues[0]
							ctx.EnsureDesc(&d381)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d381)
							} else {
								ctx.EmitStoreToStack(d381, int32(bbs[7].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					ctx.EmitJump(d378.Condition, lbl9)
					ctx.FreeDesc(&d377)
					snap382 := d3
					snap383 := d4
					snap384 := d5
					snap385 := d6
					snap386 := d7
					snap387 := d8
					snap388 := d9
					snap389 := d10
					snap390 := d11
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
					snap412 := d57
					snap413 := d58
					snap414 := d129
					snap415 := d130
					snap416 := d131
					snap417 := d132
					snap418 := d133
					snap419 := d135
					snap420 := d136
					snap421 := d137
					snap422 := d138
					snap423 := d139
					snap424 := d140
					snap425 := d141
					snap426 := d142
					snap427 := d143
					snap428 := d145
					snap429 := d146
					snap430 := d147
					snap431 := d148
					snap432 := d149
					snap433 := d258
					snap434 := d259
					snap435 := d260
					snap436 := d375
					snap437 := d376
					snap438 := d377
					snap439 := d378
					snap440 := d381
					alloc441 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc441)
					d3 = snap382
					d4 = snap383
					d5 = snap384
					d6 = snap385
					d7 = snap386
					d8 = snap387
					d9 = snap388
					d10 = snap389
					d11 = snap390
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
					d57 = snap412
					d58 = snap413
					d129 = snap414
					d130 = snap415
					d131 = snap416
					d132 = snap417
					d133 = snap418
					d135 = snap419
					d136 = snap420
					d137 = snap421
					d138 = snap422
					d139 = snap423
					d140 = snap424
					d141 = snap425
					d142 = snap426
					d143 = snap427
					d145 = snap428
					d146 = snap429
					d147 = snap430
					d148 = snap431
					d149 = snap432
					d258 = snap433
					d259 = snap434
					d260 = snap435
					d375 = snap436
					d376 = snap437
					d377 = snap438
					d378 = snap439
					d381 = snap440
					ctx.RestoreAllocState(alloc441)
					d3 = snap382
					d4 = snap383
					d5 = snap384
					d6 = snap385
					d7 = snap386
					d8 = snap387
					d9 = snap388
					d10 = snap389
					d11 = snap390
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
					d57 = snap412
					d58 = snap413
					d129 = snap414
					d130 = snap415
					d131 = snap416
					d132 = snap417
					d133 = snap418
					d135 = snap419
					d136 = snap420
					d137 = snap421
					d138 = snap422
					d139 = snap423
					d140 = snap424
					d141 = snap425
					d142 = snap426
					d143 = snap427
					d145 = snap428
					d146 = snap429
					d147 = snap430
					d148 = snap431
					d149 = snap432
					d258 = snap433
					d259 = snap434
					d260 = snap435
					d375 = snap436
					d376 = snap437
					d377 = snap438
					d378 = snap439
					d381 = snap440
					ps442 := PhiState{General: true}
					ps442.OverlayValues = make([]JITValueDesc, 382)
					ps442.OverlayValues[3] = d3
					ps442.OverlayValues[4] = d4
					ps442.OverlayValues[5] = d5
					ps442.OverlayValues[6] = d6
					ps442.OverlayValues[7] = d7
					ps442.OverlayValues[8] = d8
					ps442.OverlayValues[9] = d9
					ps442.OverlayValues[10] = d10
					ps442.OverlayValues[11] = d11
					ps442.OverlayValues[36] = d36
					ps442.OverlayValues[37] = d37
					ps442.OverlayValues[38] = d38
					ps442.OverlayValues[39] = d39
					ps442.OverlayValues[40] = d40
					ps442.OverlayValues[41] = d41
					ps442.OverlayValues[42] = d42
					ps442.OverlayValues[43] = d43
					ps442.OverlayValues[44] = d44
					ps442.OverlayValues[45] = d45
					ps442.OverlayValues[46] = d46
					ps442.OverlayValues[47] = d47
					ps442.OverlayValues[48] = d48
					ps442.OverlayValues[49] = d49
					ps442.OverlayValues[50] = d50
					ps442.OverlayValues[51] = d51
					ps442.OverlayValues[52] = d52
					ps442.OverlayValues[53] = d53
					ps442.OverlayValues[54] = d54
					ps442.OverlayValues[55] = d55
					ps442.OverlayValues[56] = d56
					ps442.OverlayValues[57] = d57
					ps442.OverlayValues[58] = d58
					ps442.OverlayValues[129] = d129
					ps442.OverlayValues[130] = d130
					ps442.OverlayValues[131] = d131
					ps442.OverlayValues[132] = d132
					ps442.OverlayValues[133] = d133
					ps442.OverlayValues[135] = d135
					ps442.OverlayValues[136] = d136
					ps442.OverlayValues[137] = d137
					ps442.OverlayValues[138] = d138
					ps442.OverlayValues[139] = d139
					ps442.OverlayValues[140] = d140
					ps442.OverlayValues[141] = d141
					ps442.OverlayValues[142] = d142
					ps442.OverlayValues[143] = d143
					ps442.OverlayValues[145] = d145
					ps442.OverlayValues[146] = d146
					ps442.OverlayValues[147] = d147
					ps442.OverlayValues[148] = d148
					ps442.OverlayValues[149] = d149
					ps442.OverlayValues[258] = d258
					ps442.OverlayValues[259] = d259
					ps442.OverlayValues[260] = d260
					ps442.OverlayValues[375] = d375
					ps442.OverlayValues[376] = d376
					ps442.OverlayValues[377] = d377
					ps442.OverlayValues[378] = d378
					ps442.OverlayValues[381] = d381
					ps443 := PhiState{General: true}
					ps443.OverlayValues = make([]JITValueDesc, 382)
					ps443.OverlayValues[3] = d3
					ps443.OverlayValues[4] = d4
					ps443.OverlayValues[5] = d5
					ps443.OverlayValues[6] = d6
					ps443.OverlayValues[7] = d7
					ps443.OverlayValues[8] = d8
					ps443.OverlayValues[9] = d9
					ps443.OverlayValues[10] = d10
					ps443.OverlayValues[11] = d11
					ps443.OverlayValues[36] = d36
					ps443.OverlayValues[37] = d37
					ps443.OverlayValues[38] = d38
					ps443.OverlayValues[39] = d39
					ps443.OverlayValues[40] = d40
					ps443.OverlayValues[41] = d41
					ps443.OverlayValues[42] = d42
					ps443.OverlayValues[43] = d43
					ps443.OverlayValues[44] = d44
					ps443.OverlayValues[45] = d45
					ps443.OverlayValues[46] = d46
					ps443.OverlayValues[47] = d47
					ps443.OverlayValues[48] = d48
					ps443.OverlayValues[49] = d49
					ps443.OverlayValues[50] = d50
					ps443.OverlayValues[51] = d51
					ps443.OverlayValues[52] = d52
					ps443.OverlayValues[53] = d53
					ps443.OverlayValues[54] = d54
					ps443.OverlayValues[55] = d55
					ps443.OverlayValues[56] = d56
					ps443.OverlayValues[57] = d57
					ps443.OverlayValues[58] = d58
					ps443.OverlayValues[129] = d129
					ps443.OverlayValues[130] = d130
					ps443.OverlayValues[131] = d131
					ps443.OverlayValues[132] = d132
					ps443.OverlayValues[133] = d133
					ps443.OverlayValues[135] = d135
					ps443.OverlayValues[136] = d136
					ps443.OverlayValues[137] = d137
					ps443.OverlayValues[138] = d138
					ps443.OverlayValues[139] = d139
					ps443.OverlayValues[140] = d140
					ps443.OverlayValues[141] = d141
					ps443.OverlayValues[142] = d142
					ps443.OverlayValues[143] = d143
					ps443.OverlayValues[145] = d145
					ps443.OverlayValues[146] = d146
					ps443.OverlayValues[147] = d147
					ps443.OverlayValues[148] = d148
					ps443.OverlayValues[149] = d149
					ps443.OverlayValues[258] = d258
					ps443.OverlayValues[259] = d259
					ps443.OverlayValues[260] = d260
					ps443.OverlayValues[375] = d375
					ps443.OverlayValues[376] = d376
					ps443.OverlayValues[377] = d377
					ps443.OverlayValues[378] = d378
					ps443.OverlayValues[381] = d381
					snap444 := d3
					snap445 := d4
					snap446 := d5
					snap447 := d6
					snap448 := d7
					snap449 := d8
					snap450 := d9
					snap451 := d10
					snap452 := d11
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
					snap474 := d57
					snap475 := d58
					snap476 := d129
					snap477 := d130
					snap478 := d131
					snap479 := d132
					snap480 := d133
					snap481 := d135
					snap482 := d136
					snap483 := d137
					snap484 := d138
					snap485 := d139
					snap486 := d140
					snap487 := d141
					snap488 := d142
					snap489 := d143
					snap490 := d145
					snap491 := d146
					snap492 := d147
					snap493 := d148
					snap494 := d149
					snap495 := d258
					snap496 := d259
					snap497 := d260
					snap498 := d375
					snap499 := d376
					snap500 := d377
					snap501 := d378
					snap502 := d381
					alloc503 := ctx.SnapshotAllocState()
					if !bbs[9].Rendered {
						bbs[9].RenderPS(ps443)
					}
					ctx.RestoreAllocState(alloc503)
					d3 = snap444
					d4 = snap445
					d5 = snap446
					d6 = snap447
					d7 = snap448
					d8 = snap449
					d9 = snap450
					d10 = snap451
					d11 = snap452
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
					d57 = snap474
					d58 = snap475
					d129 = snap476
					d130 = snap477
					d131 = snap478
					d132 = snap479
					d133 = snap480
					d135 = snap481
					d136 = snap482
					d137 = snap483
					d138 = snap484
					d139 = snap485
					d140 = snap486
					d141 = snap487
					d142 = snap488
					d143 = snap489
					d145 = snap490
					d146 = snap491
					d147 = snap492
					d148 = snap493
					d149 = snap494
					d258 = snap495
					d259 = snap496
					d260 = snap497
					d375 = snap498
					d376 = snap499
					d377 = snap500
					d378 = snap501
					d381 = snap502
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps442)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
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
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
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
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 259 && ps.OverlayValues[259].Loc != LocNone {
						d259 = ps.OverlayValues[259]
					}
					if len(ps.OverlayValues) > 260 && ps.OverlayValues[260].Loc != LocNone {
						d260 = ps.OverlayValues[260]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != LocNone {
						d376 = ps.OverlayValues[376]
					}
					if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != LocNone {
						d377 = ps.OverlayValues[377]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != LocNone {
						d381 = ps.OverlayValues[381]
					}
					ctx.ReclaimUntrackedRegs()
					var d504 JITValueDesc
					if d8.SliceSizeKnown {
						d504 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d8.KnownSliceLen))}
					} else if d8.Loc == LocImm {
						d504 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d8.StackOff))}
					} else if d8.Loc == LocStackTriple {
						d504 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d8.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d8)
						if d8.Loc == LocRegPair || d8.Loc == LocRegTriple {
							d504 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg2, ID: 0}
						} else if d8.Loc == LocReg {
							d504 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d376)
					ctx.EnsureDesc(&d504)
					ctx.EnsureDescsTogether(&d376, &d504)
					var d505 JITValueDesc
					if d376.Loc == LocImm && d504.Loc == LocImm {
						d505 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d376.Imm.Int() < d504.Imm.Int())}
					} else if d504.Loc == LocImm {
						r28 := ctx.AllocRegExcept(d376.Reg)
						if d504.Imm.Int() >= -2147483648 && d504.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d376.Reg, int32(d504.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d504.Imm.Int()))
							ctx.EmitCmpInt64(d376.Reg, RegR11)
						}
						d505 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r28, Condition: CondSignedLess}
						ctx.BindReg(r28, &d505)
					} else if d376.Loc == LocImm {
						r29 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d376.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d504.Reg)
						d505 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r29, Condition: CondSignedLess}
						ctx.BindReg(r29, &d505)
					} else {
						r30 := ctx.AllocRegExcept(d376.Reg)
						ctx.EmitCmpInt64(d376.Reg, d504.Reg)
						d505 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r30, Condition: CondSignedLess}
						ctx.BindReg(r30, &d505)
					}
					ctx.FreeDesc(&d504)
					d506 = d505
					ctx.EnsureDesc(&d506)
					if d506.Loc != LocImm && d506.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d506.Loc == LocImm {
						if d506.Imm.Bool() {
							if ps.General {
							}
							ps507 := PhiState{General: ps.General}
							ps507.OverlayValues = make([]JITValueDesc, 507)
							ps507.OverlayValues[3] = d3
							ps507.OverlayValues[4] = d4
							ps507.OverlayValues[5] = d5
							ps507.OverlayValues[6] = d6
							ps507.OverlayValues[7] = d7
							ps507.OverlayValues[8] = d8
							ps507.OverlayValues[9] = d9
							ps507.OverlayValues[10] = d10
							ps507.OverlayValues[11] = d11
							ps507.OverlayValues[36] = d36
							ps507.OverlayValues[37] = d37
							ps507.OverlayValues[38] = d38
							ps507.OverlayValues[39] = d39
							ps507.OverlayValues[40] = d40
							ps507.OverlayValues[41] = d41
							ps507.OverlayValues[42] = d42
							ps507.OverlayValues[43] = d43
							ps507.OverlayValues[44] = d44
							ps507.OverlayValues[45] = d45
							ps507.OverlayValues[46] = d46
							ps507.OverlayValues[47] = d47
							ps507.OverlayValues[48] = d48
							ps507.OverlayValues[49] = d49
							ps507.OverlayValues[50] = d50
							ps507.OverlayValues[51] = d51
							ps507.OverlayValues[52] = d52
							ps507.OverlayValues[53] = d53
							ps507.OverlayValues[54] = d54
							ps507.OverlayValues[55] = d55
							ps507.OverlayValues[56] = d56
							ps507.OverlayValues[57] = d57
							ps507.OverlayValues[58] = d58
							ps507.OverlayValues[129] = d129
							ps507.OverlayValues[130] = d130
							ps507.OverlayValues[131] = d131
							ps507.OverlayValues[132] = d132
							ps507.OverlayValues[133] = d133
							ps507.OverlayValues[135] = d135
							ps507.OverlayValues[136] = d136
							ps507.OverlayValues[137] = d137
							ps507.OverlayValues[138] = d138
							ps507.OverlayValues[139] = d139
							ps507.OverlayValues[140] = d140
							ps507.OverlayValues[141] = d141
							ps507.OverlayValues[142] = d142
							ps507.OverlayValues[143] = d143
							ps507.OverlayValues[145] = d145
							ps507.OverlayValues[146] = d146
							ps507.OverlayValues[147] = d147
							ps507.OverlayValues[148] = d148
							ps507.OverlayValues[149] = d149
							ps507.OverlayValues[258] = d258
							ps507.OverlayValues[259] = d259
							ps507.OverlayValues[260] = d260
							ps507.OverlayValues[375] = d375
							ps507.OverlayValues[376] = d376
							ps507.OverlayValues[377] = d377
							ps507.OverlayValues[378] = d378
							ps507.OverlayValues[381] = d381
							ps507.OverlayValues[504] = d504
							ps507.OverlayValues[505] = d505
							ps507.OverlayValues[506] = d506
							return bbs[10].RenderPS(ps507)
						}
						if ps.General {
						}
						ps508 := PhiState{General: ps.General}
						ps508.OverlayValues = make([]JITValueDesc, 507)
						ps508.OverlayValues[3] = d3
						ps508.OverlayValues[4] = d4
						ps508.OverlayValues[5] = d5
						ps508.OverlayValues[6] = d6
						ps508.OverlayValues[7] = d7
						ps508.OverlayValues[8] = d8
						ps508.OverlayValues[9] = d9
						ps508.OverlayValues[10] = d10
						ps508.OverlayValues[11] = d11
						ps508.OverlayValues[36] = d36
						ps508.OverlayValues[37] = d37
						ps508.OverlayValues[38] = d38
						ps508.OverlayValues[39] = d39
						ps508.OverlayValues[40] = d40
						ps508.OverlayValues[41] = d41
						ps508.OverlayValues[42] = d42
						ps508.OverlayValues[43] = d43
						ps508.OverlayValues[44] = d44
						ps508.OverlayValues[45] = d45
						ps508.OverlayValues[46] = d46
						ps508.OverlayValues[47] = d47
						ps508.OverlayValues[48] = d48
						ps508.OverlayValues[49] = d49
						ps508.OverlayValues[50] = d50
						ps508.OverlayValues[51] = d51
						ps508.OverlayValues[52] = d52
						ps508.OverlayValues[53] = d53
						ps508.OverlayValues[54] = d54
						ps508.OverlayValues[55] = d55
						ps508.OverlayValues[56] = d56
						ps508.OverlayValues[57] = d57
						ps508.OverlayValues[58] = d58
						ps508.OverlayValues[129] = d129
						ps508.OverlayValues[130] = d130
						ps508.OverlayValues[131] = d131
						ps508.OverlayValues[132] = d132
						ps508.OverlayValues[133] = d133
						ps508.OverlayValues[135] = d135
						ps508.OverlayValues[136] = d136
						ps508.OverlayValues[137] = d137
						ps508.OverlayValues[138] = d138
						ps508.OverlayValues[139] = d139
						ps508.OverlayValues[140] = d140
						ps508.OverlayValues[141] = d141
						ps508.OverlayValues[142] = d142
						ps508.OverlayValues[143] = d143
						ps508.OverlayValues[145] = d145
						ps508.OverlayValues[146] = d146
						ps508.OverlayValues[147] = d147
						ps508.OverlayValues[148] = d148
						ps508.OverlayValues[149] = d149
						ps508.OverlayValues[258] = d258
						ps508.OverlayValues[259] = d259
						ps508.OverlayValues[260] = d260
						ps508.OverlayValues[375] = d375
						ps508.OverlayValues[376] = d376
						ps508.OverlayValues[377] = d377
						ps508.OverlayValues[378] = d378
						ps508.OverlayValues[381] = d381
						ps508.OverlayValues[504] = d504
						ps508.OverlayValues[505] = d505
						ps508.OverlayValues[506] = d506
						return bbs[11].RenderPS(ps508)
					}
					if !ps.General {
						ps.General = true
						return bbs[8].RenderPS(ps)
					}
					ctx.EmitJump(d506.Condition, lbl11)
					ctx.FreeDesc(&d505)
					snap509 := d3
					snap510 := d4
					snap511 := d5
					snap512 := d6
					snap513 := d7
					snap514 := d8
					snap515 := d9
					snap516 := d10
					snap517 := d11
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
					snap539 := d57
					snap540 := d58
					snap541 := d129
					snap542 := d130
					snap543 := d131
					snap544 := d132
					snap545 := d133
					snap546 := d135
					snap547 := d136
					snap548 := d137
					snap549 := d138
					snap550 := d139
					snap551 := d140
					snap552 := d141
					snap553 := d142
					snap554 := d143
					snap555 := d145
					snap556 := d146
					snap557 := d147
					snap558 := d148
					snap559 := d149
					snap560 := d258
					snap561 := d259
					snap562 := d260
					snap563 := d375
					snap564 := d376
					snap565 := d377
					snap566 := d378
					snap567 := d381
					snap568 := d504
					snap569 := d505
					snap570 := d506
					alloc571 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc571)
					d3 = snap509
					d4 = snap510
					d5 = snap511
					d6 = snap512
					d7 = snap513
					d8 = snap514
					d9 = snap515
					d10 = snap516
					d11 = snap517
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
					d57 = snap539
					d58 = snap540
					d129 = snap541
					d130 = snap542
					d131 = snap543
					d132 = snap544
					d133 = snap545
					d135 = snap546
					d136 = snap547
					d137 = snap548
					d138 = snap549
					d139 = snap550
					d140 = snap551
					d141 = snap552
					d142 = snap553
					d143 = snap554
					d145 = snap555
					d146 = snap556
					d147 = snap557
					d148 = snap558
					d149 = snap559
					d258 = snap560
					d259 = snap561
					d260 = snap562
					d375 = snap563
					d376 = snap564
					d377 = snap565
					d378 = snap566
					d381 = snap567
					d504 = snap568
					d505 = snap569
					d506 = snap570
					ctx.RestoreAllocState(alloc571)
					d3 = snap509
					d4 = snap510
					d5 = snap511
					d6 = snap512
					d7 = snap513
					d8 = snap514
					d9 = snap515
					d10 = snap516
					d11 = snap517
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
					d57 = snap539
					d58 = snap540
					d129 = snap541
					d130 = snap542
					d131 = snap543
					d132 = snap544
					d133 = snap545
					d135 = snap546
					d136 = snap547
					d137 = snap548
					d138 = snap549
					d139 = snap550
					d140 = snap551
					d141 = snap552
					d142 = snap553
					d143 = snap554
					d145 = snap555
					d146 = snap556
					d147 = snap557
					d148 = snap558
					d149 = snap559
					d258 = snap560
					d259 = snap561
					d260 = snap562
					d375 = snap563
					d376 = snap564
					d377 = snap565
					d378 = snap566
					d381 = snap567
					d504 = snap568
					d505 = snap569
					d506 = snap570
					ps572 := PhiState{General: true}
					ps572.OverlayValues = make([]JITValueDesc, 507)
					ps572.OverlayValues[3] = d3
					ps572.OverlayValues[4] = d4
					ps572.OverlayValues[5] = d5
					ps572.OverlayValues[6] = d6
					ps572.OverlayValues[7] = d7
					ps572.OverlayValues[8] = d8
					ps572.OverlayValues[9] = d9
					ps572.OverlayValues[10] = d10
					ps572.OverlayValues[11] = d11
					ps572.OverlayValues[36] = d36
					ps572.OverlayValues[37] = d37
					ps572.OverlayValues[38] = d38
					ps572.OverlayValues[39] = d39
					ps572.OverlayValues[40] = d40
					ps572.OverlayValues[41] = d41
					ps572.OverlayValues[42] = d42
					ps572.OverlayValues[43] = d43
					ps572.OverlayValues[44] = d44
					ps572.OverlayValues[45] = d45
					ps572.OverlayValues[46] = d46
					ps572.OverlayValues[47] = d47
					ps572.OverlayValues[48] = d48
					ps572.OverlayValues[49] = d49
					ps572.OverlayValues[50] = d50
					ps572.OverlayValues[51] = d51
					ps572.OverlayValues[52] = d52
					ps572.OverlayValues[53] = d53
					ps572.OverlayValues[54] = d54
					ps572.OverlayValues[55] = d55
					ps572.OverlayValues[56] = d56
					ps572.OverlayValues[57] = d57
					ps572.OverlayValues[58] = d58
					ps572.OverlayValues[129] = d129
					ps572.OverlayValues[130] = d130
					ps572.OverlayValues[131] = d131
					ps572.OverlayValues[132] = d132
					ps572.OverlayValues[133] = d133
					ps572.OverlayValues[135] = d135
					ps572.OverlayValues[136] = d136
					ps572.OverlayValues[137] = d137
					ps572.OverlayValues[138] = d138
					ps572.OverlayValues[139] = d139
					ps572.OverlayValues[140] = d140
					ps572.OverlayValues[141] = d141
					ps572.OverlayValues[142] = d142
					ps572.OverlayValues[143] = d143
					ps572.OverlayValues[145] = d145
					ps572.OverlayValues[146] = d146
					ps572.OverlayValues[147] = d147
					ps572.OverlayValues[148] = d148
					ps572.OverlayValues[149] = d149
					ps572.OverlayValues[258] = d258
					ps572.OverlayValues[259] = d259
					ps572.OverlayValues[260] = d260
					ps572.OverlayValues[375] = d375
					ps572.OverlayValues[376] = d376
					ps572.OverlayValues[377] = d377
					ps572.OverlayValues[378] = d378
					ps572.OverlayValues[381] = d381
					ps572.OverlayValues[504] = d504
					ps572.OverlayValues[505] = d505
					ps572.OverlayValues[506] = d506
					ps573 := PhiState{General: true}
					ps573.OverlayValues = make([]JITValueDesc, 507)
					ps573.OverlayValues[3] = d3
					ps573.OverlayValues[4] = d4
					ps573.OverlayValues[5] = d5
					ps573.OverlayValues[6] = d6
					ps573.OverlayValues[7] = d7
					ps573.OverlayValues[8] = d8
					ps573.OverlayValues[9] = d9
					ps573.OverlayValues[10] = d10
					ps573.OverlayValues[11] = d11
					ps573.OverlayValues[36] = d36
					ps573.OverlayValues[37] = d37
					ps573.OverlayValues[38] = d38
					ps573.OverlayValues[39] = d39
					ps573.OverlayValues[40] = d40
					ps573.OverlayValues[41] = d41
					ps573.OverlayValues[42] = d42
					ps573.OverlayValues[43] = d43
					ps573.OverlayValues[44] = d44
					ps573.OverlayValues[45] = d45
					ps573.OverlayValues[46] = d46
					ps573.OverlayValues[47] = d47
					ps573.OverlayValues[48] = d48
					ps573.OverlayValues[49] = d49
					ps573.OverlayValues[50] = d50
					ps573.OverlayValues[51] = d51
					ps573.OverlayValues[52] = d52
					ps573.OverlayValues[53] = d53
					ps573.OverlayValues[54] = d54
					ps573.OverlayValues[55] = d55
					ps573.OverlayValues[56] = d56
					ps573.OverlayValues[57] = d57
					ps573.OverlayValues[58] = d58
					ps573.OverlayValues[129] = d129
					ps573.OverlayValues[130] = d130
					ps573.OverlayValues[131] = d131
					ps573.OverlayValues[132] = d132
					ps573.OverlayValues[133] = d133
					ps573.OverlayValues[135] = d135
					ps573.OverlayValues[136] = d136
					ps573.OverlayValues[137] = d137
					ps573.OverlayValues[138] = d138
					ps573.OverlayValues[139] = d139
					ps573.OverlayValues[140] = d140
					ps573.OverlayValues[141] = d141
					ps573.OverlayValues[142] = d142
					ps573.OverlayValues[143] = d143
					ps573.OverlayValues[145] = d145
					ps573.OverlayValues[146] = d146
					ps573.OverlayValues[147] = d147
					ps573.OverlayValues[148] = d148
					ps573.OverlayValues[149] = d149
					ps573.OverlayValues[258] = d258
					ps573.OverlayValues[259] = d259
					ps573.OverlayValues[260] = d260
					ps573.OverlayValues[375] = d375
					ps573.OverlayValues[376] = d376
					ps573.OverlayValues[377] = d377
					ps573.OverlayValues[378] = d378
					ps573.OverlayValues[381] = d381
					ps573.OverlayValues[504] = d504
					ps573.OverlayValues[505] = d505
					ps573.OverlayValues[506] = d506
					snap574 := d3
					snap575 := d4
					snap576 := d5
					snap577 := d6
					snap578 := d7
					snap579 := d8
					snap580 := d9
					snap581 := d10
					snap582 := d11
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
					snap604 := d57
					snap605 := d58
					snap606 := d129
					snap607 := d130
					snap608 := d131
					snap609 := d132
					snap610 := d133
					snap611 := d135
					snap612 := d136
					snap613 := d137
					snap614 := d138
					snap615 := d139
					snap616 := d140
					snap617 := d141
					snap618 := d142
					snap619 := d143
					snap620 := d145
					snap621 := d146
					snap622 := d147
					snap623 := d148
					snap624 := d149
					snap625 := d258
					snap626 := d259
					snap627 := d260
					snap628 := d375
					snap629 := d376
					snap630 := d377
					snap631 := d378
					snap632 := d381
					snap633 := d504
					snap634 := d505
					snap635 := d506
					alloc636 := ctx.SnapshotAllocState()
					if !bbs[11].Rendered {
						bbs[11].RenderPS(ps573)
					}
					ctx.RestoreAllocState(alloc636)
					d3 = snap574
					d4 = snap575
					d5 = snap576
					d6 = snap577
					d7 = snap578
					d8 = snap579
					d9 = snap580
					d10 = snap581
					d11 = snap582
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
					d57 = snap604
					d58 = snap605
					d129 = snap606
					d130 = snap607
					d131 = snap608
					d132 = snap609
					d133 = snap610
					d135 = snap611
					d136 = snap612
					d137 = snap613
					d138 = snap614
					d139 = snap615
					d140 = snap616
					d141 = snap617
					d142 = snap618
					d143 = snap619
					d145 = snap620
					d146 = snap621
					d147 = snap622
					d148 = snap623
					d149 = snap624
					d258 = snap625
					d259 = snap626
					d260 = snap627
					d375 = snap628
					d376 = snap629
					d377 = snap630
					d378 = snap631
					d381 = snap632
					d504 = snap633
					d505 = snap634
					d506 = snap635
					if !bbs[10].Rendered {
						return bbs[10].RenderPS(ps572)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
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
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
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
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 259 && ps.OverlayValues[259].Loc != LocNone {
						d259 = ps.OverlayValues[259]
					}
					if len(ps.OverlayValues) > 260 && ps.OverlayValues[260].Loc != LocNone {
						d260 = ps.OverlayValues[260]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != LocNone {
						d376 = ps.OverlayValues[376]
					}
					if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != LocNone {
						d377 = ps.OverlayValues[377]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != LocNone {
						d381 = ps.OverlayValues[381]
					}
					if len(ps.OverlayValues) > 504 && ps.OverlayValues[504].Loc != LocNone {
						d504 = ps.OverlayValues[504]
					}
					if len(ps.OverlayValues) > 505 && ps.OverlayValues[505].Loc != LocNone {
						d505 = ps.OverlayValues[505]
					}
					if len(ps.OverlayValues) > 506 && ps.OverlayValues[506].Loc != LocNone {
						d506 = ps.OverlayValues[506]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d44)
					ctx.EnsureDesc(&d44)
					var d637 JITValueDesc
					if d44.Loc == LocImm {
						d637 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d44.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d44.Reg)
						ctx.EmitMovRegReg(scratch, d44.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d637 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d637)
					}
					if d637.Loc == LocReg && d44.Loc == LocReg && d637.Reg == d44.Reg {
						ctx.TransferReg(d44.Reg)
						d44.Loc = LocNone
					}
					ctx.FreeDesc(&d44)
					ctx.EnsureDesc(&d637)
					ctx.EnsureDesc(&d637)
					ctx.EnsureDesc(&d637)
					d639 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.SyncDesc(&d637)
					d640 = d5
					d640.ID = 0
					d641 = d639
					d641.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d640, &d641, &d637, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d640, &d641, &d637, int32(16))
					}
					ctx.FreeDesc(&d641)
					ctx.EnsureDesc(&d39)
					var d642 JITValueDesc
					if d39.Loc == LocImm {
						d642 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d39.Imm.Int() > 0)}
					} else {
						r31 := ctx.AllocRegExcept(d39.Reg)
						ctx.EmitCmpRegImm32(d39.Reg, 0)
						d642 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r31, Condition: CondSignedGreater}
						ctx.BindReg(r31, &d642)
					}
					d643 = d642
					ctx.EnsureDesc(&d643)
					if d643.Loc != LocImm && d643.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d643.Loc == LocImm {
						if d643.Imm.Bool() {
							if ps.General {
							}
							ps644 := PhiState{General: ps.General}
							ps644.OverlayValues = make([]JITValueDesc, 644)
							ps644.OverlayValues[3] = d3
							ps644.OverlayValues[4] = d4
							ps644.OverlayValues[5] = d5
							ps644.OverlayValues[6] = d6
							ps644.OverlayValues[7] = d7
							ps644.OverlayValues[8] = d8
							ps644.OverlayValues[9] = d9
							ps644.OverlayValues[10] = d10
							ps644.OverlayValues[11] = d11
							ps644.OverlayValues[36] = d36
							ps644.OverlayValues[37] = d37
							ps644.OverlayValues[38] = d38
							ps644.OverlayValues[39] = d39
							ps644.OverlayValues[40] = d40
							ps644.OverlayValues[41] = d41
							ps644.OverlayValues[42] = d42
							ps644.OverlayValues[43] = d43
							ps644.OverlayValues[44] = d44
							ps644.OverlayValues[45] = d45
							ps644.OverlayValues[46] = d46
							ps644.OverlayValues[47] = d47
							ps644.OverlayValues[48] = d48
							ps644.OverlayValues[49] = d49
							ps644.OverlayValues[50] = d50
							ps644.OverlayValues[51] = d51
							ps644.OverlayValues[52] = d52
							ps644.OverlayValues[53] = d53
							ps644.OverlayValues[54] = d54
							ps644.OverlayValues[55] = d55
							ps644.OverlayValues[56] = d56
							ps644.OverlayValues[57] = d57
							ps644.OverlayValues[58] = d58
							ps644.OverlayValues[129] = d129
							ps644.OverlayValues[130] = d130
							ps644.OverlayValues[131] = d131
							ps644.OverlayValues[132] = d132
							ps644.OverlayValues[133] = d133
							ps644.OverlayValues[135] = d135
							ps644.OverlayValues[136] = d136
							ps644.OverlayValues[137] = d137
							ps644.OverlayValues[138] = d138
							ps644.OverlayValues[139] = d139
							ps644.OverlayValues[140] = d140
							ps644.OverlayValues[141] = d141
							ps644.OverlayValues[142] = d142
							ps644.OverlayValues[143] = d143
							ps644.OverlayValues[145] = d145
							ps644.OverlayValues[146] = d146
							ps644.OverlayValues[147] = d147
							ps644.OverlayValues[148] = d148
							ps644.OverlayValues[149] = d149
							ps644.OverlayValues[258] = d258
							ps644.OverlayValues[259] = d259
							ps644.OverlayValues[260] = d260
							ps644.OverlayValues[375] = d375
							ps644.OverlayValues[376] = d376
							ps644.OverlayValues[377] = d377
							ps644.OverlayValues[378] = d378
							ps644.OverlayValues[381] = d381
							ps644.OverlayValues[504] = d504
							ps644.OverlayValues[505] = d505
							ps644.OverlayValues[506] = d506
							ps644.OverlayValues[637] = d637
							ps644.OverlayValues[638] = d638
							ps644.OverlayValues[639] = d639
							ps644.OverlayValues[640] = d640
							ps644.OverlayValues[641] = d641
							ps644.OverlayValues[642] = d642
							ps644.OverlayValues[643] = d643
							return bbs[12].RenderPS(ps644)
						}
						if ps.General {
						}
						ps645 := PhiState{General: ps.General}
						ps645.OverlayValues = make([]JITValueDesc, 644)
						ps645.OverlayValues[3] = d3
						ps645.OverlayValues[4] = d4
						ps645.OverlayValues[5] = d5
						ps645.OverlayValues[6] = d6
						ps645.OverlayValues[7] = d7
						ps645.OverlayValues[8] = d8
						ps645.OverlayValues[9] = d9
						ps645.OverlayValues[10] = d10
						ps645.OverlayValues[11] = d11
						ps645.OverlayValues[36] = d36
						ps645.OverlayValues[37] = d37
						ps645.OverlayValues[38] = d38
						ps645.OverlayValues[39] = d39
						ps645.OverlayValues[40] = d40
						ps645.OverlayValues[41] = d41
						ps645.OverlayValues[42] = d42
						ps645.OverlayValues[43] = d43
						ps645.OverlayValues[44] = d44
						ps645.OverlayValues[45] = d45
						ps645.OverlayValues[46] = d46
						ps645.OverlayValues[47] = d47
						ps645.OverlayValues[48] = d48
						ps645.OverlayValues[49] = d49
						ps645.OverlayValues[50] = d50
						ps645.OverlayValues[51] = d51
						ps645.OverlayValues[52] = d52
						ps645.OverlayValues[53] = d53
						ps645.OverlayValues[54] = d54
						ps645.OverlayValues[55] = d55
						ps645.OverlayValues[56] = d56
						ps645.OverlayValues[57] = d57
						ps645.OverlayValues[58] = d58
						ps645.OverlayValues[129] = d129
						ps645.OverlayValues[130] = d130
						ps645.OverlayValues[131] = d131
						ps645.OverlayValues[132] = d132
						ps645.OverlayValues[133] = d133
						ps645.OverlayValues[135] = d135
						ps645.OverlayValues[136] = d136
						ps645.OverlayValues[137] = d137
						ps645.OverlayValues[138] = d138
						ps645.OverlayValues[139] = d139
						ps645.OverlayValues[140] = d140
						ps645.OverlayValues[141] = d141
						ps645.OverlayValues[142] = d142
						ps645.OverlayValues[143] = d143
						ps645.OverlayValues[145] = d145
						ps645.OverlayValues[146] = d146
						ps645.OverlayValues[147] = d147
						ps645.OverlayValues[148] = d148
						ps645.OverlayValues[149] = d149
						ps645.OverlayValues[258] = d258
						ps645.OverlayValues[259] = d259
						ps645.OverlayValues[260] = d260
						ps645.OverlayValues[375] = d375
						ps645.OverlayValues[376] = d376
						ps645.OverlayValues[377] = d377
						ps645.OverlayValues[378] = d378
						ps645.OverlayValues[381] = d381
						ps645.OverlayValues[504] = d504
						ps645.OverlayValues[505] = d505
						ps645.OverlayValues[506] = d506
						ps645.OverlayValues[637] = d637
						ps645.OverlayValues[638] = d638
						ps645.OverlayValues[639] = d639
						ps645.OverlayValues[640] = d640
						ps645.OverlayValues[641] = d641
						ps645.OverlayValues[642] = d642
						ps645.OverlayValues[643] = d643
						return bbs[13].RenderPS(ps645)
					}
					if !ps.General {
						ps.General = true
						return bbs[9].RenderPS(ps)
					}
					ctx.EmitJump(d643.Condition, lbl13)
					ctx.FreeDesc(&d642)
					snap646 := d3
					snap647 := d4
					snap648 := d5
					snap649 := d6
					snap650 := d7
					snap651 := d8
					snap652 := d9
					snap653 := d10
					snap654 := d11
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
					snap676 := d57
					snap677 := d58
					snap678 := d129
					snap679 := d130
					snap680 := d131
					snap681 := d132
					snap682 := d133
					snap683 := d135
					snap684 := d136
					snap685 := d137
					snap686 := d138
					snap687 := d139
					snap688 := d140
					snap689 := d141
					snap690 := d142
					snap691 := d143
					snap692 := d145
					snap693 := d146
					snap694 := d147
					snap695 := d148
					snap696 := d149
					snap697 := d258
					snap698 := d259
					snap699 := d260
					snap700 := d375
					snap701 := d376
					snap702 := d377
					snap703 := d378
					snap704 := d381
					snap705 := d504
					snap706 := d505
					snap707 := d506
					snap708 := d637
					snap709 := d638
					snap710 := d639
					snap711 := d640
					snap712 := d641
					snap713 := d642
					snap714 := d643
					alloc715 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc715)
					d3 = snap646
					d4 = snap647
					d5 = snap648
					d6 = snap649
					d7 = snap650
					d8 = snap651
					d9 = snap652
					d10 = snap653
					d11 = snap654
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
					d57 = snap676
					d58 = snap677
					d129 = snap678
					d130 = snap679
					d131 = snap680
					d132 = snap681
					d133 = snap682
					d135 = snap683
					d136 = snap684
					d137 = snap685
					d138 = snap686
					d139 = snap687
					d140 = snap688
					d141 = snap689
					d142 = snap690
					d143 = snap691
					d145 = snap692
					d146 = snap693
					d147 = snap694
					d148 = snap695
					d149 = snap696
					d258 = snap697
					d259 = snap698
					d260 = snap699
					d375 = snap700
					d376 = snap701
					d377 = snap702
					d378 = snap703
					d381 = snap704
					d504 = snap705
					d505 = snap706
					d506 = snap707
					d637 = snap708
					d638 = snap709
					d639 = snap710
					d640 = snap711
					d641 = snap712
					d642 = snap713
					d643 = snap714
					ctx.RestoreAllocState(alloc715)
					d3 = snap646
					d4 = snap647
					d5 = snap648
					d6 = snap649
					d7 = snap650
					d8 = snap651
					d9 = snap652
					d10 = snap653
					d11 = snap654
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
					d57 = snap676
					d58 = snap677
					d129 = snap678
					d130 = snap679
					d131 = snap680
					d132 = snap681
					d133 = snap682
					d135 = snap683
					d136 = snap684
					d137 = snap685
					d138 = snap686
					d139 = snap687
					d140 = snap688
					d141 = snap689
					d142 = snap690
					d143 = snap691
					d145 = snap692
					d146 = snap693
					d147 = snap694
					d148 = snap695
					d149 = snap696
					d258 = snap697
					d259 = snap698
					d260 = snap699
					d375 = snap700
					d376 = snap701
					d377 = snap702
					d378 = snap703
					d381 = snap704
					d504 = snap705
					d505 = snap706
					d506 = snap707
					d637 = snap708
					d638 = snap709
					d639 = snap710
					d640 = snap711
					d641 = snap712
					d642 = snap713
					d643 = snap714
					ps716 := PhiState{General: true}
					ps716.OverlayValues = make([]JITValueDesc, 644)
					ps716.OverlayValues[3] = d3
					ps716.OverlayValues[4] = d4
					ps716.OverlayValues[5] = d5
					ps716.OverlayValues[6] = d6
					ps716.OverlayValues[7] = d7
					ps716.OverlayValues[8] = d8
					ps716.OverlayValues[9] = d9
					ps716.OverlayValues[10] = d10
					ps716.OverlayValues[11] = d11
					ps716.OverlayValues[36] = d36
					ps716.OverlayValues[37] = d37
					ps716.OverlayValues[38] = d38
					ps716.OverlayValues[39] = d39
					ps716.OverlayValues[40] = d40
					ps716.OverlayValues[41] = d41
					ps716.OverlayValues[42] = d42
					ps716.OverlayValues[43] = d43
					ps716.OverlayValues[44] = d44
					ps716.OverlayValues[45] = d45
					ps716.OverlayValues[46] = d46
					ps716.OverlayValues[47] = d47
					ps716.OverlayValues[48] = d48
					ps716.OverlayValues[49] = d49
					ps716.OverlayValues[50] = d50
					ps716.OverlayValues[51] = d51
					ps716.OverlayValues[52] = d52
					ps716.OverlayValues[53] = d53
					ps716.OverlayValues[54] = d54
					ps716.OverlayValues[55] = d55
					ps716.OverlayValues[56] = d56
					ps716.OverlayValues[57] = d57
					ps716.OverlayValues[58] = d58
					ps716.OverlayValues[129] = d129
					ps716.OverlayValues[130] = d130
					ps716.OverlayValues[131] = d131
					ps716.OverlayValues[132] = d132
					ps716.OverlayValues[133] = d133
					ps716.OverlayValues[135] = d135
					ps716.OverlayValues[136] = d136
					ps716.OverlayValues[137] = d137
					ps716.OverlayValues[138] = d138
					ps716.OverlayValues[139] = d139
					ps716.OverlayValues[140] = d140
					ps716.OverlayValues[141] = d141
					ps716.OverlayValues[142] = d142
					ps716.OverlayValues[143] = d143
					ps716.OverlayValues[145] = d145
					ps716.OverlayValues[146] = d146
					ps716.OverlayValues[147] = d147
					ps716.OverlayValues[148] = d148
					ps716.OverlayValues[149] = d149
					ps716.OverlayValues[258] = d258
					ps716.OverlayValues[259] = d259
					ps716.OverlayValues[260] = d260
					ps716.OverlayValues[375] = d375
					ps716.OverlayValues[376] = d376
					ps716.OverlayValues[377] = d377
					ps716.OverlayValues[378] = d378
					ps716.OverlayValues[381] = d381
					ps716.OverlayValues[504] = d504
					ps716.OverlayValues[505] = d505
					ps716.OverlayValues[506] = d506
					ps716.OverlayValues[637] = d637
					ps716.OverlayValues[638] = d638
					ps716.OverlayValues[639] = d639
					ps716.OverlayValues[640] = d640
					ps716.OverlayValues[641] = d641
					ps716.OverlayValues[642] = d642
					ps716.OverlayValues[643] = d643
					ps717 := PhiState{General: true}
					ps717.OverlayValues = make([]JITValueDesc, 644)
					ps717.OverlayValues[3] = d3
					ps717.OverlayValues[4] = d4
					ps717.OverlayValues[5] = d5
					ps717.OverlayValues[6] = d6
					ps717.OverlayValues[7] = d7
					ps717.OverlayValues[8] = d8
					ps717.OverlayValues[9] = d9
					ps717.OverlayValues[10] = d10
					ps717.OverlayValues[11] = d11
					ps717.OverlayValues[36] = d36
					ps717.OverlayValues[37] = d37
					ps717.OverlayValues[38] = d38
					ps717.OverlayValues[39] = d39
					ps717.OverlayValues[40] = d40
					ps717.OverlayValues[41] = d41
					ps717.OverlayValues[42] = d42
					ps717.OverlayValues[43] = d43
					ps717.OverlayValues[44] = d44
					ps717.OverlayValues[45] = d45
					ps717.OverlayValues[46] = d46
					ps717.OverlayValues[47] = d47
					ps717.OverlayValues[48] = d48
					ps717.OverlayValues[49] = d49
					ps717.OverlayValues[50] = d50
					ps717.OverlayValues[51] = d51
					ps717.OverlayValues[52] = d52
					ps717.OverlayValues[53] = d53
					ps717.OverlayValues[54] = d54
					ps717.OverlayValues[55] = d55
					ps717.OverlayValues[56] = d56
					ps717.OverlayValues[57] = d57
					ps717.OverlayValues[58] = d58
					ps717.OverlayValues[129] = d129
					ps717.OverlayValues[130] = d130
					ps717.OverlayValues[131] = d131
					ps717.OverlayValues[132] = d132
					ps717.OverlayValues[133] = d133
					ps717.OverlayValues[135] = d135
					ps717.OverlayValues[136] = d136
					ps717.OverlayValues[137] = d137
					ps717.OverlayValues[138] = d138
					ps717.OverlayValues[139] = d139
					ps717.OverlayValues[140] = d140
					ps717.OverlayValues[141] = d141
					ps717.OverlayValues[142] = d142
					ps717.OverlayValues[143] = d143
					ps717.OverlayValues[145] = d145
					ps717.OverlayValues[146] = d146
					ps717.OverlayValues[147] = d147
					ps717.OverlayValues[148] = d148
					ps717.OverlayValues[149] = d149
					ps717.OverlayValues[258] = d258
					ps717.OverlayValues[259] = d259
					ps717.OverlayValues[260] = d260
					ps717.OverlayValues[375] = d375
					ps717.OverlayValues[376] = d376
					ps717.OverlayValues[377] = d377
					ps717.OverlayValues[378] = d378
					ps717.OverlayValues[381] = d381
					ps717.OverlayValues[504] = d504
					ps717.OverlayValues[505] = d505
					ps717.OverlayValues[506] = d506
					ps717.OverlayValues[637] = d637
					ps717.OverlayValues[638] = d638
					ps717.OverlayValues[639] = d639
					ps717.OverlayValues[640] = d640
					ps717.OverlayValues[641] = d641
					ps717.OverlayValues[642] = d642
					ps717.OverlayValues[643] = d643
					snap718 := d3
					snap719 := d4
					snap720 := d5
					snap721 := d6
					snap722 := d7
					snap723 := d8
					snap724 := d9
					snap725 := d10
					snap726 := d11
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
					snap748 := d57
					snap749 := d58
					snap750 := d129
					snap751 := d130
					snap752 := d131
					snap753 := d132
					snap754 := d133
					snap755 := d135
					snap756 := d136
					snap757 := d137
					snap758 := d138
					snap759 := d139
					snap760 := d140
					snap761 := d141
					snap762 := d142
					snap763 := d143
					snap764 := d145
					snap765 := d146
					snap766 := d147
					snap767 := d148
					snap768 := d149
					snap769 := d258
					snap770 := d259
					snap771 := d260
					snap772 := d375
					snap773 := d376
					snap774 := d377
					snap775 := d378
					snap776 := d381
					snap777 := d504
					snap778 := d505
					snap779 := d506
					snap780 := d637
					snap781 := d638
					snap782 := d639
					snap783 := d640
					snap784 := d641
					snap785 := d642
					snap786 := d643
					alloc787 := ctx.SnapshotAllocState()
					if !bbs[13].Rendered {
						bbs[13].RenderPS(ps717)
					}
					ctx.RestoreAllocState(alloc787)
					d3 = snap718
					d4 = snap719
					d5 = snap720
					d6 = snap721
					d7 = snap722
					d8 = snap723
					d9 = snap724
					d10 = snap725
					d11 = snap726
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
					d57 = snap748
					d58 = snap749
					d129 = snap750
					d130 = snap751
					d131 = snap752
					d132 = snap753
					d133 = snap754
					d135 = snap755
					d136 = snap756
					d137 = snap757
					d138 = snap758
					d139 = snap759
					d140 = snap760
					d141 = snap761
					d142 = snap762
					d143 = snap763
					d145 = snap764
					d146 = snap765
					d147 = snap766
					d148 = snap767
					d149 = snap768
					d258 = snap769
					d259 = snap770
					d260 = snap771
					d375 = snap772
					d376 = snap773
					d377 = snap774
					d378 = snap775
					d381 = snap776
					d504 = snap777
					d505 = snap778
					d506 = snap779
					d637 = snap780
					d638 = snap781
					d639 = snap782
					d640 = snap783
					d641 = snap784
					d642 = snap785
					d643 = snap786
					if !bbs[12].Rendered {
						return bbs[12].RenderPS(ps716)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
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
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
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
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 259 && ps.OverlayValues[259].Loc != LocNone {
						d259 = ps.OverlayValues[259]
					}
					if len(ps.OverlayValues) > 260 && ps.OverlayValues[260].Loc != LocNone {
						d260 = ps.OverlayValues[260]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != LocNone {
						d376 = ps.OverlayValues[376]
					}
					if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != LocNone {
						d377 = ps.OverlayValues[377]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != LocNone {
						d381 = ps.OverlayValues[381]
					}
					if len(ps.OverlayValues) > 504 && ps.OverlayValues[504].Loc != LocNone {
						d504 = ps.OverlayValues[504]
					}
					if len(ps.OverlayValues) > 505 && ps.OverlayValues[505].Loc != LocNone {
						d505 = ps.OverlayValues[505]
					}
					if len(ps.OverlayValues) > 506 && ps.OverlayValues[506].Loc != LocNone {
						d506 = ps.OverlayValues[506]
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
					if len(ps.OverlayValues) > 642 && ps.OverlayValues[642].Loc != LocNone {
						d642 = ps.OverlayValues[642]
					}
					if len(ps.OverlayValues) > 643 && ps.OverlayValues[643].Loc != LocNone {
						d643 = ps.OverlayValues[643]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d376)
					d789 = ctx.EmitSliceElementAddress(&d8, &d376, 16)
					ctx.EnsureDesc(&d789)
					r32 := ctx.AllocRegExcept(d789.Reg)
					ctx.EmitMovRegMem(r32, d789.Reg, 8)
					ctx.EmitMovRegMem(d789.Reg, d789.Reg, 0)
					d788 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d789.Reg, Reg2: r32}
					ctx.BindReg(d789.Reg, &d788)
					ctx.BindReg(r32, &d788)
					ctx.EnsureDesc(&d376)
					ctx.SyncDesc(&d788)
					d790 = d142
					d790.ID = 0
					d791 = d376
					d791.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d790, &d791, &d788, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d376)
						d791 = d376
						d791.ID = 0
						ctx.EmitStoreScmerSliceElement(&d790, &d791, &d788, int32(16))
					}
					ctx.FreeDesc(&d791)
					ctx.FreeDesc(&d788)
					if ps.General {
						ctx.SyncDesc(&d376)
						if d376.Loc == LocReg {
							ctx.ProtectReg(d376.Reg)
						} else if d376.Loc == LocRegPair {
							ctx.ProtectReg(d376.Reg)
							ctx.ProtectReg(d376.Reg2)
						}
						d792 = d376
						if d792.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d792)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d792)
						} else {
							ctx.EmitStoreToStack(d792, int32(bbs[7].PhiBase)+int32(0))
						}
						if d376.Loc == LocReg {
							ctx.UnprotectReg(d376.Reg)
						} else if d376.Loc == LocRegPair {
							ctx.UnprotectReg(d376.Reg)
							ctx.UnprotectReg(d376.Reg2)
						}
					}
					ps793 := PhiState{General: ps.General}
					ps793.OverlayValues = make([]JITValueDesc, 793)
					ps793.OverlayValues[3] = d3
					ps793.OverlayValues[4] = d4
					ps793.OverlayValues[5] = d5
					ps793.OverlayValues[6] = d6
					ps793.OverlayValues[7] = d7
					ps793.OverlayValues[8] = d8
					ps793.OverlayValues[9] = d9
					ps793.OverlayValues[10] = d10
					ps793.OverlayValues[11] = d11
					ps793.OverlayValues[36] = d36
					ps793.OverlayValues[37] = d37
					ps793.OverlayValues[38] = d38
					ps793.OverlayValues[39] = d39
					ps793.OverlayValues[40] = d40
					ps793.OverlayValues[41] = d41
					ps793.OverlayValues[42] = d42
					ps793.OverlayValues[43] = d43
					ps793.OverlayValues[44] = d44
					ps793.OverlayValues[45] = d45
					ps793.OverlayValues[46] = d46
					ps793.OverlayValues[47] = d47
					ps793.OverlayValues[48] = d48
					ps793.OverlayValues[49] = d49
					ps793.OverlayValues[50] = d50
					ps793.OverlayValues[51] = d51
					ps793.OverlayValues[52] = d52
					ps793.OverlayValues[53] = d53
					ps793.OverlayValues[54] = d54
					ps793.OverlayValues[55] = d55
					ps793.OverlayValues[56] = d56
					ps793.OverlayValues[57] = d57
					ps793.OverlayValues[58] = d58
					ps793.OverlayValues[129] = d129
					ps793.OverlayValues[130] = d130
					ps793.OverlayValues[131] = d131
					ps793.OverlayValues[132] = d132
					ps793.OverlayValues[133] = d133
					ps793.OverlayValues[135] = d135
					ps793.OverlayValues[136] = d136
					ps793.OverlayValues[137] = d137
					ps793.OverlayValues[138] = d138
					ps793.OverlayValues[139] = d139
					ps793.OverlayValues[140] = d140
					ps793.OverlayValues[141] = d141
					ps793.OverlayValues[142] = d142
					ps793.OverlayValues[143] = d143
					ps793.OverlayValues[145] = d145
					ps793.OverlayValues[146] = d146
					ps793.OverlayValues[147] = d147
					ps793.OverlayValues[148] = d148
					ps793.OverlayValues[149] = d149
					ps793.OverlayValues[258] = d258
					ps793.OverlayValues[259] = d259
					ps793.OverlayValues[260] = d260
					ps793.OverlayValues[375] = d375
					ps793.OverlayValues[376] = d376
					ps793.OverlayValues[377] = d377
					ps793.OverlayValues[378] = d378
					ps793.OverlayValues[381] = d381
					ps793.OverlayValues[504] = d504
					ps793.OverlayValues[505] = d505
					ps793.OverlayValues[506] = d506
					ps793.OverlayValues[637] = d637
					ps793.OverlayValues[638] = d638
					ps793.OverlayValues[639] = d639
					ps793.OverlayValues[640] = d640
					ps793.OverlayValues[641] = d641
					ps793.OverlayValues[642] = d642
					ps793.OverlayValues[643] = d643
					ps793.OverlayValues[788] = d788
					ps793.OverlayValues[789] = d789
					ps793.OverlayValues[790] = d790
					ps793.OverlayValues[791] = d791
					ps793.OverlayValues[792] = d792
					ps793.PhiValues = make([]JITValueDesc, 1)
					d794 = d376
					ps793.PhiValues[0] = d794
					if ps793.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps793)
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
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
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
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
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
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 259 && ps.OverlayValues[259].Loc != LocNone {
						d259 = ps.OverlayValues[259]
					}
					if len(ps.OverlayValues) > 260 && ps.OverlayValues[260].Loc != LocNone {
						d260 = ps.OverlayValues[260]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != LocNone {
						d376 = ps.OverlayValues[376]
					}
					if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != LocNone {
						d377 = ps.OverlayValues[377]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != LocNone {
						d381 = ps.OverlayValues[381]
					}
					if len(ps.OverlayValues) > 504 && ps.OverlayValues[504].Loc != LocNone {
						d504 = ps.OverlayValues[504]
					}
					if len(ps.OverlayValues) > 505 && ps.OverlayValues[505].Loc != LocNone {
						d505 = ps.OverlayValues[505]
					}
					if len(ps.OverlayValues) > 506 && ps.OverlayValues[506].Loc != LocNone {
						d506 = ps.OverlayValues[506]
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
					if len(ps.OverlayValues) > 642 && ps.OverlayValues[642].Loc != LocNone {
						d642 = ps.OverlayValues[642]
					}
					if len(ps.OverlayValues) > 643 && ps.OverlayValues[643].Loc != LocNone {
						d643 = ps.OverlayValues[643]
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
					if len(ps.OverlayValues) > 791 && ps.OverlayValues[791].Loc != LocNone {
						d791 = ps.OverlayValues[791]
					}
					if len(ps.OverlayValues) > 792 && ps.OverlayValues[792].Loc != LocNone {
						d792 = ps.OverlayValues[792]
					}
					if len(ps.OverlayValues) > 794 && ps.OverlayValues[794].Loc != LocNone {
						d794 = ps.OverlayValues[794]
					}
					ctx.ReclaimUntrackedRegs()
					d795 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d376)
					ctx.SyncDesc(&d795)
					d796 = d142
					d796.ID = 0
					d797 = d376
					d797.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d796, &d797, &d795, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d376)
						d797 = d376
						d797.ID = 0
						ctx.EmitStoreScmerSliceElement(&d796, &d797, &d795, int32(16))
					}
					ctx.FreeDesc(&d797)
					ctx.FreeDesc(&d795)
					if ps.General {
						ctx.SyncDesc(&d376)
						if d376.Loc == LocReg {
							ctx.ProtectReg(d376.Reg)
						} else if d376.Loc == LocRegPair {
							ctx.ProtectReg(d376.Reg)
							ctx.ProtectReg(d376.Reg2)
						}
						d798 = d376
						if d798.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d798)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d798)
						} else {
							ctx.EmitStoreToStack(d798, int32(bbs[7].PhiBase)+int32(0))
						}
						if d376.Loc == LocReg {
							ctx.UnprotectReg(d376.Reg)
						} else if d376.Loc == LocRegPair {
							ctx.UnprotectReg(d376.Reg)
							ctx.UnprotectReg(d376.Reg2)
						}
					}
					ps799 := PhiState{General: ps.General}
					ps799.OverlayValues = make([]JITValueDesc, 799)
					ps799.OverlayValues[3] = d3
					ps799.OverlayValues[4] = d4
					ps799.OverlayValues[5] = d5
					ps799.OverlayValues[6] = d6
					ps799.OverlayValues[7] = d7
					ps799.OverlayValues[8] = d8
					ps799.OverlayValues[9] = d9
					ps799.OverlayValues[10] = d10
					ps799.OverlayValues[11] = d11
					ps799.OverlayValues[36] = d36
					ps799.OverlayValues[37] = d37
					ps799.OverlayValues[38] = d38
					ps799.OverlayValues[39] = d39
					ps799.OverlayValues[40] = d40
					ps799.OverlayValues[41] = d41
					ps799.OverlayValues[42] = d42
					ps799.OverlayValues[43] = d43
					ps799.OverlayValues[44] = d44
					ps799.OverlayValues[45] = d45
					ps799.OverlayValues[46] = d46
					ps799.OverlayValues[47] = d47
					ps799.OverlayValues[48] = d48
					ps799.OverlayValues[49] = d49
					ps799.OverlayValues[50] = d50
					ps799.OverlayValues[51] = d51
					ps799.OverlayValues[52] = d52
					ps799.OverlayValues[53] = d53
					ps799.OverlayValues[54] = d54
					ps799.OverlayValues[55] = d55
					ps799.OverlayValues[56] = d56
					ps799.OverlayValues[57] = d57
					ps799.OverlayValues[58] = d58
					ps799.OverlayValues[129] = d129
					ps799.OverlayValues[130] = d130
					ps799.OverlayValues[131] = d131
					ps799.OverlayValues[132] = d132
					ps799.OverlayValues[133] = d133
					ps799.OverlayValues[135] = d135
					ps799.OverlayValues[136] = d136
					ps799.OverlayValues[137] = d137
					ps799.OverlayValues[138] = d138
					ps799.OverlayValues[139] = d139
					ps799.OverlayValues[140] = d140
					ps799.OverlayValues[141] = d141
					ps799.OverlayValues[142] = d142
					ps799.OverlayValues[143] = d143
					ps799.OverlayValues[145] = d145
					ps799.OverlayValues[146] = d146
					ps799.OverlayValues[147] = d147
					ps799.OverlayValues[148] = d148
					ps799.OverlayValues[149] = d149
					ps799.OverlayValues[258] = d258
					ps799.OverlayValues[259] = d259
					ps799.OverlayValues[260] = d260
					ps799.OverlayValues[375] = d375
					ps799.OverlayValues[376] = d376
					ps799.OverlayValues[377] = d377
					ps799.OverlayValues[378] = d378
					ps799.OverlayValues[381] = d381
					ps799.OverlayValues[504] = d504
					ps799.OverlayValues[505] = d505
					ps799.OverlayValues[506] = d506
					ps799.OverlayValues[637] = d637
					ps799.OverlayValues[638] = d638
					ps799.OverlayValues[639] = d639
					ps799.OverlayValues[640] = d640
					ps799.OverlayValues[641] = d641
					ps799.OverlayValues[642] = d642
					ps799.OverlayValues[643] = d643
					ps799.OverlayValues[788] = d788
					ps799.OverlayValues[789] = d789
					ps799.OverlayValues[790] = d790
					ps799.OverlayValues[791] = d791
					ps799.OverlayValues[792] = d792
					ps799.OverlayValues[794] = d794
					ps799.OverlayValues[795] = d795
					ps799.OverlayValues[796] = d796
					ps799.OverlayValues[797] = d797
					ps799.OverlayValues[798] = d798
					ps799.PhiValues = make([]JITValueDesc, 1)
					d800 = d376
					ps799.PhiValues[0] = d800
					if ps799.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps799)
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
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
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
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
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
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 259 && ps.OverlayValues[259].Loc != LocNone {
						d259 = ps.OverlayValues[259]
					}
					if len(ps.OverlayValues) > 260 && ps.OverlayValues[260].Loc != LocNone {
						d260 = ps.OverlayValues[260]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != LocNone {
						d376 = ps.OverlayValues[376]
					}
					if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != LocNone {
						d377 = ps.OverlayValues[377]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != LocNone {
						d381 = ps.OverlayValues[381]
					}
					if len(ps.OverlayValues) > 504 && ps.OverlayValues[504].Loc != LocNone {
						d504 = ps.OverlayValues[504]
					}
					if len(ps.OverlayValues) > 505 && ps.OverlayValues[505].Loc != LocNone {
						d505 = ps.OverlayValues[505]
					}
					if len(ps.OverlayValues) > 506 && ps.OverlayValues[506].Loc != LocNone {
						d506 = ps.OverlayValues[506]
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
					if len(ps.OverlayValues) > 642 && ps.OverlayValues[642].Loc != LocNone {
						d642 = ps.OverlayValues[642]
					}
					if len(ps.OverlayValues) > 643 && ps.OverlayValues[643].Loc != LocNone {
						d643 = ps.OverlayValues[643]
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
					if len(ps.OverlayValues) > 791 && ps.OverlayValues[791].Loc != LocNone {
						d791 = ps.OverlayValues[791]
					}
					if len(ps.OverlayValues) > 792 && ps.OverlayValues[792].Loc != LocNone {
						d792 = ps.OverlayValues[792]
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
					if len(ps.OverlayValues) > 797 && ps.OverlayValues[797].Loc != LocNone {
						d797 = ps.OverlayValues[797]
					}
					if len(ps.OverlayValues) > 798 && ps.OverlayValues[798].Loc != LocNone {
						d798 = ps.OverlayValues[798]
					}
					if len(ps.OverlayValues) > 800 && ps.OverlayValues[800].Loc != LocNone {
						d800 = ps.OverlayValues[800]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d39)
					ctx.EnsureDesc(&d39)
					var d801 JITValueDesc
					if d39.Loc == LocImm {
						d801 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d39.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d39.Reg)
						ctx.EmitMovRegReg(scratch, d39.Reg)
						ctx.EmitSubRegImm32(scratch, int32(1))
						d801 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d801)
					}
					if d801.Loc == LocReg && d39.Loc == LocReg && d801.Reg == d39.Reg {
						ctx.TransferReg(d39.Reg)
						d39.Loc = LocNone
					}
					ctx.FreeDesc(&d39)
					ctx.EnsureDesc(&d801)
					ctx.EnsureDesc(&d801)
					ctx.EnsureDesc(&d801)
					d803 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.SyncDesc(&d801)
					d804 = d5
					d804.ID = 0
					d805 = d803
					d805.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d804, &d805, &d801, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d804, &d805, &d801, int32(16))
					}
					ctx.FreeDesc(&d805)
					d806 = args[0]
					d806.ID = 0
					ctx.SyncDesc(&d806)
					if d806.Loc == LocRegPair || d806.Loc == LocStackPair || d806.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d806, &result)
						result.Type = d806.Type
					} else {
						switch d806.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d806)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d806)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d806)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d806, &result)
							result.Type = d806.Type
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
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
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
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
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
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
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
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 259 && ps.OverlayValues[259].Loc != LocNone {
						d259 = ps.OverlayValues[259]
					}
					if len(ps.OverlayValues) > 260 && ps.OverlayValues[260].Loc != LocNone {
						d260 = ps.OverlayValues[260]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != LocNone {
						d376 = ps.OverlayValues[376]
					}
					if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != LocNone {
						d377 = ps.OverlayValues[377]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != LocNone {
						d381 = ps.OverlayValues[381]
					}
					if len(ps.OverlayValues) > 504 && ps.OverlayValues[504].Loc != LocNone {
						d504 = ps.OverlayValues[504]
					}
					if len(ps.OverlayValues) > 505 && ps.OverlayValues[505].Loc != LocNone {
						d505 = ps.OverlayValues[505]
					}
					if len(ps.OverlayValues) > 506 && ps.OverlayValues[506].Loc != LocNone {
						d506 = ps.OverlayValues[506]
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
					if len(ps.OverlayValues) > 642 && ps.OverlayValues[642].Loc != LocNone {
						d642 = ps.OverlayValues[642]
					}
					if len(ps.OverlayValues) > 643 && ps.OverlayValues[643].Loc != LocNone {
						d643 = ps.OverlayValues[643]
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
					if len(ps.OverlayValues) > 791 && ps.OverlayValues[791].Loc != LocNone {
						d791 = ps.OverlayValues[791]
					}
					if len(ps.OverlayValues) > 792 && ps.OverlayValues[792].Loc != LocNone {
						d792 = ps.OverlayValues[792]
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
					if len(ps.OverlayValues) > 797 && ps.OverlayValues[797].Loc != LocNone {
						d797 = ps.OverlayValues[797]
					}
					if len(ps.OverlayValues) > 798 && ps.OverlayValues[798].Loc != LocNone {
						d798 = ps.OverlayValues[798]
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
					if len(ps.OverlayValues) > 805 && ps.OverlayValues[805].Loc != LocNone {
						d805 = ps.OverlayValues[805]
					}
					if len(ps.OverlayValues) > 806 && ps.OverlayValues[806].Loc != LocNone {
						d806 = ps.OverlayValues[806]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d56)
					d807 = d6
					_ = d807
					d808 = d56
					_ = d808
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl15 := ctx.ReserveLabel()
					_ = lbl15
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d807)
					ctx.EnsureDesc(&d807)
					d807 = JITPrepareScmerGoArg(ctx, d807)
					ctx.EnsureDesc(&d808)
					ctx.EnsureDesc(&d808)
					ctx.EnsureDesc(&d808)
					if d808.Loc != LocRegTriple && d808.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d809 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d809.Loc == LocRegPair || d809.Loc == LocStackPair || d809.Loc == LocRegTriple || d809.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d807)
					ctx.SyncDesc(&d808)
					ctx.SyncDesc(&d809)
					d810 = ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d807, d808, d809}, 2)
					d810.NoHeapPointer = false
					ctx.BindReg(d810.Reg, &d810)
					ctx.BindReg(d810.Reg2, &d810)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d810)
					d811 = args[0]
					d811.ID = 0
					ctx.SyncDesc(&d811)
					if d811.Loc == LocRegPair || d811.Loc == LocStackPair || d811.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d811, &result)
						result.Type = d811.Type
					} else {
						switch d811.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d811)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d811)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d811)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d811, &result)
							result.Type = d811.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps812 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps812)
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
				var d111 JITValueDesc
				_ = d111
				var d112 JITValueDesc
				_ = d112
				var d113 JITValueDesc
				_ = d113
				var d114 JITValueDesc
				_ = d114
				var d115 JITValueDesc
				_ = d115
				var d180 JITValueDesc
				_ = d180
				var d181 JITValueDesc
				_ = d181
				var d182 JITValueDesc
				_ = d182
				var d253 JITValueDesc
				_ = d253
				var d254 JITValueDesc
				_ = d254
				var d255 JITValueDesc
				_ = d255
				var d258 JITValueDesc
				_ = d258
				var d335 JITValueDesc
				_ = d335
				var d336 JITValueDesc
				_ = d336
				var d337 JITValueDesc
				_ = d337
				var d338 JITValueDesc
				_ = d338
				var d339 JITValueDesc
				_ = d339
				var d341 JITValueDesc
				_ = d341
				var d342 JITValueDesc
				_ = d342
				var d343 JITValueDesc
				_ = d343
				var d344 JITValueDesc
				_ = d344
				var d346 JITValueDesc
				_ = d346
				var d347 JITValueDesc
				_ = d347
				var d348 JITValueDesc
				_ = d348
				var d349 JITValueDesc
				_ = d349
				var d350 JITValueDesc
				_ = d350
				var d351 JITValueDesc
				_ = d351
				var d354 JITValueDesc
				_ = d354
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
				var d474 JITValueDesc
				_ = d474
				var d475 JITValueDesc
				_ = d475
				var d476 JITValueDesc
				_ = d476
				var d477 JITValueDesc
				_ = d477
				var d478 JITValueDesc
				_ = d478
				var d479 JITValueDesc
				_ = d479
				var d480 JITValueDesc
				_ = d480
				var d481 JITValueDesc
				_ = d481
				var d482 JITValueDesc
				_ = d482
				var d483 JITValueDesc
				_ = d483
				var d485 JITValueDesc
				_ = d485
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
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 38}, {Color: 1, Width: 1, Cost: 25}}, Count: 2})
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
					d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
				} else {
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				}
				_ = d4
				var d5 JITValueDesc
				if phiHomeOK3 {
					d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
						d13 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d13)
					}
					ctx.FreeDesc(&d12)
					d14 = d13
					ctx.EnsureDesc(&d14)
					if d14.Loc != LocImm && d14.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
					ctx.EmitJump(d14.Condition, lbl2)
					ctx.FreeDesc(&d13)
					snap17 := d4
					snap18 := d5
					snap19 := d6
					snap20 := d7
					snap21 := d8
					snap22 := d9
					snap23 := d10
					snap24 := d11
					snap25 := d12
					snap26 := d13
					snap27 := d14
					alloc28 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc28)
					d4 = snap17
					d5 = snap18
					d6 = snap19
					d7 = snap20
					d8 = snap21
					d9 = snap22
					d10 = snap23
					d11 = snap24
					d12 = snap25
					d13 = snap26
					d14 = snap27
					ctx.RestoreAllocState(alloc28)
					d4 = snap17
					d5 = snap18
					d6 = snap19
					d7 = snap20
					d8 = snap21
					d9 = snap22
					d10 = snap23
					d11 = snap24
					d12 = snap25
					d13 = snap26
					d14 = snap27
					ps29 := PhiState{General: true}
					ps29.OverlayValues = make([]JITValueDesc, 15)
					ps29.OverlayValues[4] = d4
					ps29.OverlayValues[5] = d5
					ps29.OverlayValues[6] = d6
					ps29.OverlayValues[7] = d7
					ps29.OverlayValues[8] = d8
					ps29.OverlayValues[9] = d9
					ps29.OverlayValues[10] = d10
					ps29.OverlayValues[11] = d11
					ps29.OverlayValues[12] = d12
					ps29.OverlayValues[13] = d13
					ps29.OverlayValues[14] = d14
					ps30 := PhiState{General: true}
					ps30.OverlayValues = make([]JITValueDesc, 15)
					ps30.OverlayValues[4] = d4
					ps30.OverlayValues[5] = d5
					ps30.OverlayValues[6] = d6
					ps30.OverlayValues[7] = d7
					ps30.OverlayValues[8] = d8
					ps30.OverlayValues[9] = d9
					ps30.OverlayValues[10] = d10
					ps30.OverlayValues[11] = d11
					ps30.OverlayValues[12] = d12
					ps30.OverlayValues[13] = d13
					ps30.OverlayValues[14] = d14
					snap31 := d4
					snap32 := d5
					snap33 := d6
					snap34 := d7
					snap35 := d8
					snap36 := d9
					snap37 := d10
					snap38 := d11
					snap39 := d12
					snap40 := d13
					snap41 := d14
					alloc42 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps30)
					}
					ctx.RestoreAllocState(alloc42)
					d4 = snap31
					d5 = snap32
					d6 = snap33
					d7 = snap34
					d8 = snap35
					d9 = snap36
					d10 = snap37
					d11 = snap38
					d12 = snap39
					d13 = snap40
					d14 = snap41
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps29)
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
					d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					d45 = ctx.EmitSliceElementAddress(&d7, &d43, 16)
					ctx.EnsureDesc(&d45)
					r3 := ctx.AllocRegExcept(d45.Reg)
					ctx.EmitMovRegMem(r3, d45.Reg, 8)
					ctx.EmitMovRegMem(d45.Reg, d45.Reg, 0)
					d44 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d45.Reg, Reg2: r3}
					ctx.BindReg(d45.Reg, &d44)
					ctx.BindReg(r3, &d44)
					var d46 JITValueDesc
					if d44.Loc == LocImm {
						d46 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d44.Imm.Int())}
					} else if d44.Type == tagInt && d44.Loc == LocRegPair {
						ctx.FreeReg(d44.Reg)
						d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg2}
						ctx.BindReg(d44.Reg2, &d46)
						ctx.BindReg(d44.Reg2, &d46)
					} else if d44.Type == tagInt && d44.Loc == LocReg {
						d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg}
						ctx.BindReg(d44.Reg, &d46)
						ctx.BindReg(d44.Reg, &d46)
					} else {
						d46 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d44}, 1)
						d46.Type = tagInt
						ctx.BindReg(d46.Reg, &d46)
					}
					ctx.FreeDesc(&d44)
					ctx.EnsureDesc(&d46)
					ctx.EnsureDesc(&d46)
					ctx.StabilizeDescForControlFlow(&d46)
					d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(3)}
					var d49 JITValueDesc
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocRegPair || d7.Loc == LocRegTriple {
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d7.Reg2}
						ctx.BindReg(d7.Reg2, &d49)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d48)
					ctx.EnsureDesc(&d49)
					var d51 JITValueDesc
					if d49.Loc == LocImm && d48.Loc == LocImm {
						d51 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d49.Imm.Int() - d48.Imm.Int())}
					} else {
						r4 := ctx.AllocReg()
						if d49.Loc == LocImm {
							ctx.EmitMovRegImm64(r4, uint64(d49.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r4, d49.Reg)
						}
						if d48.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d48.Imm.Int()))
							ctx.EmitSubInt64(r4, RegR11)
						} else {
							ctx.EmitSubInt64(r4, d48.Reg)
						}
						d51 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d51)
					}
					var d52 JITValueDesc
					r5 := ctx.EmitSliceDataAfterLow(&d7, &d48, 16)
					d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
					ctx.BindReg(r5, &d52)
					ctx.BindReg(r5, &d52)
					var d53 JITValueDesc
					var r6 Reg
					var r7 Reg
					ctx.SyncDesc(&d52)
					ctx.EnsureDesc(&d52)
					if d52.Loc == LocImm {
						r6 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, uint64(d52.Imm.Int()))
					} else {
						r6 = d52.Reg
					}
					ctx.ProtectReg(r6)
					ctx.SyncDesc(&d51)
					ctx.EnsureDesc(&d51)
					if d51.Loc == LocImm {
						r7 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, uint64(d51.Imm.Int()))
					} else {
						r7 = d51.Reg
					}
					ctx.ProtectReg(r7)
					r8 := ctx.EmitSliceCapAfterLow(&d7, &d48, r6, r7)
					ctx.UnprotectReg(r7)
					ctx.UnprotectReg(r6)
					d53 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8}
					ctx.BindReg(r6, &d53)
					ctx.BindReg(r7, &d53)
					ctx.BindReg(r8, &d53)
					ctx.BindReg(r6, &d53)
					ctx.BindReg(r7, &d53)
					ctx.BindReg(r8, &d53)
					ctx.StabilizeDescForControlFlow(&d53)
					ctx.EnsureDesc(&d46)
					var d54 JITValueDesc
					if d46.Loc == LocImm {
						d54 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d46.Imm.Int() <= 0)}
					} else {
						r9 := ctx.AllocRegExcept(d46.Reg)
						ctx.EmitCmpRegImm32(d46.Reg, 0)
						d54 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondSignedLessOrEqual}
						ctx.BindReg(r9, &d54)
					}
					d55 = d54
					ctx.EnsureDesc(&d55)
					if d55.Loc != LocImm && d55.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d55.Loc == LocImm {
						if d55.Imm.Bool() {
							if ps.General {
							}
							ps56 := PhiState{General: ps.General}
							ps56.OverlayValues = make([]JITValueDesc, 56)
							ps56.OverlayValues[4] = d4
							ps56.OverlayValues[5] = d5
							ps56.OverlayValues[6] = d6
							ps56.OverlayValues[7] = d7
							ps56.OverlayValues[8] = d8
							ps56.OverlayValues[9] = d9
							ps56.OverlayValues[10] = d10
							ps56.OverlayValues[11] = d11
							ps56.OverlayValues[12] = d12
							ps56.OverlayValues[13] = d13
							ps56.OverlayValues[14] = d14
							ps56.OverlayValues[43] = d43
							ps56.OverlayValues[44] = d44
							ps56.OverlayValues[45] = d45
							ps56.OverlayValues[46] = d46
							ps56.OverlayValues[47] = d47
							ps56.OverlayValues[48] = d48
							ps56.OverlayValues[49] = d49
							ps56.OverlayValues[50] = d50
							ps56.OverlayValues[51] = d51
							ps56.OverlayValues[52] = d52
							ps56.OverlayValues[53] = d53
							ps56.OverlayValues[54] = d54
							ps56.OverlayValues[55] = d55
							return bbs[3].RenderPS(ps56)
						}
						if ps.General {
						}
						ps57 := PhiState{General: ps.General}
						ps57.OverlayValues = make([]JITValueDesc, 56)
						ps57.OverlayValues[4] = d4
						ps57.OverlayValues[5] = d5
						ps57.OverlayValues[6] = d6
						ps57.OverlayValues[7] = d7
						ps57.OverlayValues[8] = d8
						ps57.OverlayValues[9] = d9
						ps57.OverlayValues[10] = d10
						ps57.OverlayValues[11] = d11
						ps57.OverlayValues[12] = d12
						ps57.OverlayValues[13] = d13
						ps57.OverlayValues[14] = d14
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
						return bbs[6].RenderPS(ps57)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					ctx.EmitJump(d55.Condition, lbl4)
					ctx.FreeDesc(&d54)
					snap58 := d4
					snap59 := d5
					snap60 := d6
					snap61 := d7
					snap62 := d8
					snap63 := d9
					snap64 := d10
					snap65 := d11
					snap66 := d12
					snap67 := d13
					snap68 := d14
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
					snap79 := d53
					snap80 := d54
					snap81 := d55
					alloc82 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc82)
					d4 = snap58
					d5 = snap59
					d6 = snap60
					d7 = snap61
					d8 = snap62
					d9 = snap63
					d10 = snap64
					d11 = snap65
					d12 = snap66
					d13 = snap67
					d14 = snap68
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
					d53 = snap79
					d54 = snap80
					d55 = snap81
					ctx.RestoreAllocState(alloc82)
					d4 = snap58
					d5 = snap59
					d6 = snap60
					d7 = snap61
					d8 = snap62
					d9 = snap63
					d10 = snap64
					d11 = snap65
					d12 = snap66
					d13 = snap67
					d14 = snap68
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
					d53 = snap79
					d54 = snap80
					d55 = snap81
					ps83 := PhiState{General: true}
					ps83.OverlayValues = make([]JITValueDesc, 56)
					ps83.OverlayValues[4] = d4
					ps83.OverlayValues[5] = d5
					ps83.OverlayValues[6] = d6
					ps83.OverlayValues[7] = d7
					ps83.OverlayValues[8] = d8
					ps83.OverlayValues[9] = d9
					ps83.OverlayValues[10] = d10
					ps83.OverlayValues[11] = d11
					ps83.OverlayValues[12] = d12
					ps83.OverlayValues[13] = d13
					ps83.OverlayValues[14] = d14
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
					ps84 := PhiState{General: true}
					ps84.OverlayValues = make([]JITValueDesc, 56)
					ps84.OverlayValues[4] = d4
					ps84.OverlayValues[5] = d5
					ps84.OverlayValues[6] = d6
					ps84.OverlayValues[7] = d7
					ps84.OverlayValues[8] = d8
					ps84.OverlayValues[9] = d9
					ps84.OverlayValues[10] = d10
					ps84.OverlayValues[11] = d11
					ps84.OverlayValues[12] = d12
					ps84.OverlayValues[13] = d13
					ps84.OverlayValues[14] = d14
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
					snap85 := d4
					snap86 := d5
					snap87 := d6
					snap88 := d7
					snap89 := d8
					snap90 := d9
					snap91 := d10
					snap92 := d11
					snap93 := d12
					snap94 := d13
					snap95 := d14
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
					snap106 := d53
					snap107 := d54
					snap108 := d55
					alloc109 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps84)
					}
					ctx.RestoreAllocState(alloc109)
					d4 = snap85
					d5 = snap86
					d6 = snap87
					d7 = snap88
					d8 = snap89
					d9 = snap90
					d10 = snap91
					d11 = snap92
					d12 = snap93
					d13 = snap94
					d14 = snap95
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
					d53 = snap106
					d54 = snap107
					d55 = snap108
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps83)
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[7].PhiBase)+int32(0))
						}
					}
					ps110 := PhiState{General: ps.General}
					ps110.OverlayValues = make([]JITValueDesc, 56)
					ps110.OverlayValues[4] = d4
					ps110.OverlayValues[5] = d5
					ps110.OverlayValues[6] = d6
					ps110.OverlayValues[7] = d7
					ps110.OverlayValues[8] = d8
					ps110.OverlayValues[9] = d9
					ps110.OverlayValues[10] = d10
					ps110.OverlayValues[11] = d11
					ps110.OverlayValues[12] = d12
					ps110.OverlayValues[13] = d13
					ps110.OverlayValues[14] = d14
					ps110.OverlayValues[43] = d43
					ps110.OverlayValues[44] = d44
					ps110.OverlayValues[45] = d45
					ps110.OverlayValues[46] = d46
					ps110.OverlayValues[47] = d47
					ps110.OverlayValues[48] = d48
					ps110.OverlayValues[49] = d49
					ps110.OverlayValues[50] = d50
					ps110.OverlayValues[51] = d51
					ps110.OverlayValues[52] = d52
					ps110.OverlayValues[53] = d53
					ps110.OverlayValues[54] = d54
					ps110.OverlayValues[55] = d55
					ps110.PhiValues = make([]JITValueDesc, 1)
					d111 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps110.PhiValues[0] = d111
					if ps110.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps110)
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
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					ctx.ReclaimUntrackedRegs()
					var d112 JITValueDesc
					if d53.SliceSizeKnown {
						d112 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d53.KnownSliceLen))}
					} else if d53.Loc == LocImm {
						d112 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d53.StackOff))}
					} else if d53.Loc == LocStackTriple {
						d112 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d53.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d53)
						if d53.Loc == LocRegPair || d53.Loc == LocRegTriple {
							d112 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg2, ID: 0}
						} else if d53.Loc == LocReg {
							d112 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d112)
					ctx.EnsureDesc(&d46)
					ctx.EnsureDescsTogether(&d112, &d46)
					var d113 JITValueDesc
					if d112.Loc == LocImm && d46.Loc == LocImm {
						d113 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d112.Imm.Int() % d46.Imm.Int())}
					} else {
						d113 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntRem), []JITValueDesc{d112, d46}, 1)
					}
					if d113.Loc == LocReg && d112.Loc == LocReg && d113.Reg == d112.Reg {
						ctx.TransferReg(d112.Reg)
						d112.Loc = LocNone
					}
					ctx.FreeDesc(&d112)
					ctx.EnsureDesc(&d113)
					var d114 JITValueDesc
					if d113.Loc == LocImm {
						d114 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d113.Imm.Int() != 0)}
					} else {
						r10 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d113.Reg, 0)
						d114 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondNotEqual}
						ctx.BindReg(r10, &d114)
					}
					ctx.FreeDesc(&d113)
					d115 = d114
					ctx.EnsureDesc(&d115)
					if d115.Loc != LocImm && d115.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
							ps116.OverlayValues[43] = d43
							ps116.OverlayValues[44] = d44
							ps116.OverlayValues[45] = d45
							ps116.OverlayValues[46] = d46
							ps116.OverlayValues[47] = d47
							ps116.OverlayValues[48] = d48
							ps116.OverlayValues[49] = d49
							ps116.OverlayValues[50] = d50
							ps116.OverlayValues[51] = d51
							ps116.OverlayValues[52] = d52
							ps116.OverlayValues[53] = d53
							ps116.OverlayValues[54] = d54
							ps116.OverlayValues[55] = d55
							ps116.OverlayValues[111] = d111
							ps116.OverlayValues[112] = d112
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
						ps117.OverlayValues[43] = d43
						ps117.OverlayValues[44] = d44
						ps117.OverlayValues[45] = d45
						ps117.OverlayValues[46] = d46
						ps117.OverlayValues[47] = d47
						ps117.OverlayValues[48] = d48
						ps117.OverlayValues[49] = d49
						ps117.OverlayValues[50] = d50
						ps117.OverlayValues[51] = d51
						ps117.OverlayValues[52] = d52
						ps117.OverlayValues[53] = d53
						ps117.OverlayValues[54] = d54
						ps117.OverlayValues[55] = d55
						ps117.OverlayValues[111] = d111
						ps117.OverlayValues[112] = d112
						ps117.OverlayValues[113] = d113
						ps117.OverlayValues[114] = d114
						ps117.OverlayValues[115] = d115
						return bbs[4].RenderPS(ps117)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					ctx.EmitJump(d115.Condition, lbl4)
					ctx.FreeDesc(&d114)
					snap118 := d4
					snap119 := d5
					snap120 := d6
					snap121 := d7
					snap122 := d8
					snap123 := d9
					snap124 := d10
					snap125 := d11
					snap126 := d12
					snap127 := d13
					snap128 := d14
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
					snap139 := d53
					snap140 := d54
					snap141 := d55
					snap142 := d111
					snap143 := d112
					snap144 := d113
					snap145 := d114
					snap146 := d115
					alloc147 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc147)
					d4 = snap118
					d5 = snap119
					d6 = snap120
					d7 = snap121
					d8 = snap122
					d9 = snap123
					d10 = snap124
					d11 = snap125
					d12 = snap126
					d13 = snap127
					d14 = snap128
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
					d53 = snap139
					d54 = snap140
					d55 = snap141
					d111 = snap142
					d112 = snap143
					d113 = snap144
					d114 = snap145
					d115 = snap146
					ctx.RestoreAllocState(alloc147)
					d4 = snap118
					d5 = snap119
					d6 = snap120
					d7 = snap121
					d8 = snap122
					d9 = snap123
					d10 = snap124
					d11 = snap125
					d12 = snap126
					d13 = snap127
					d14 = snap128
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
					d53 = snap139
					d54 = snap140
					d55 = snap141
					d111 = snap142
					d112 = snap143
					d113 = snap144
					d114 = snap145
					d115 = snap146
					ps148 := PhiState{General: true}
					ps148.OverlayValues = make([]JITValueDesc, 116)
					ps148.OverlayValues[4] = d4
					ps148.OverlayValues[5] = d5
					ps148.OverlayValues[6] = d6
					ps148.OverlayValues[7] = d7
					ps148.OverlayValues[8] = d8
					ps148.OverlayValues[9] = d9
					ps148.OverlayValues[10] = d10
					ps148.OverlayValues[11] = d11
					ps148.OverlayValues[12] = d12
					ps148.OverlayValues[13] = d13
					ps148.OverlayValues[14] = d14
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
					ps148.OverlayValues[111] = d111
					ps148.OverlayValues[112] = d112
					ps148.OverlayValues[113] = d113
					ps148.OverlayValues[114] = d114
					ps148.OverlayValues[115] = d115
					ps149 := PhiState{General: true}
					ps149.OverlayValues = make([]JITValueDesc, 116)
					ps149.OverlayValues[4] = d4
					ps149.OverlayValues[5] = d5
					ps149.OverlayValues[6] = d6
					ps149.OverlayValues[7] = d7
					ps149.OverlayValues[8] = d8
					ps149.OverlayValues[9] = d9
					ps149.OverlayValues[10] = d10
					ps149.OverlayValues[11] = d11
					ps149.OverlayValues[12] = d12
					ps149.OverlayValues[13] = d13
					ps149.OverlayValues[14] = d14
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
					ps149.OverlayValues[111] = d111
					ps149.OverlayValues[112] = d112
					ps149.OverlayValues[113] = d113
					ps149.OverlayValues[114] = d114
					ps149.OverlayValues[115] = d115
					snap150 := d4
					snap151 := d5
					snap152 := d6
					snap153 := d7
					snap154 := d8
					snap155 := d9
					snap156 := d10
					snap157 := d11
					snap158 := d12
					snap159 := d13
					snap160 := d14
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
					snap171 := d53
					snap172 := d54
					snap173 := d55
					snap174 := d111
					snap175 := d112
					snap176 := d113
					snap177 := d114
					snap178 := d115
					alloc179 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps149)
					}
					ctx.RestoreAllocState(alloc179)
					d4 = snap150
					d5 = snap151
					d6 = snap152
					d7 = snap153
					d8 = snap154
					d9 = snap155
					d10 = snap156
					d11 = snap157
					d12 = snap158
					d13 = snap159
					d14 = snap160
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
					d53 = snap171
					d54 = snap172
					d55 = snap173
					d111 = snap174
					d112 = snap175
					d113 = snap176
					d114 = snap177
					d115 = snap178
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps148)
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
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
					ctx.ReclaimUntrackedRegs()
					var d180 JITValueDesc
					if d53.SliceSizeKnown {
						d180 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d53.KnownSliceLen))}
					} else if d53.Loc == LocImm {
						d180 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d53.StackOff))}
					} else if d53.Loc == LocStackTriple {
						d180 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d53.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d53)
						if d53.Loc == LocRegPair || d53.Loc == LocRegTriple {
							d180 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg2, ID: 0}
						} else if d53.Loc == LocReg {
							d180 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d180)
					var d181 JITValueDesc
					if d180.Loc == LocImm {
						d181 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d180.Imm.Int() == 0)}
					} else {
						r11 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d180.Reg, 0)
						d181 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r11, Condition: CondEqual}
						ctx.BindReg(r11, &d181)
					}
					ctx.FreeDesc(&d180)
					d182 = d181
					ctx.EnsureDesc(&d182)
					if d182.Loc != LocImm && d182.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d182.Loc == LocImm {
						if d182.Imm.Bool() {
							if ps.General {
							}
							ps183 := PhiState{General: ps.General}
							ps183.OverlayValues = make([]JITValueDesc, 183)
							ps183.OverlayValues[4] = d4
							ps183.OverlayValues[5] = d5
							ps183.OverlayValues[6] = d6
							ps183.OverlayValues[7] = d7
							ps183.OverlayValues[8] = d8
							ps183.OverlayValues[9] = d9
							ps183.OverlayValues[10] = d10
							ps183.OverlayValues[11] = d11
							ps183.OverlayValues[12] = d12
							ps183.OverlayValues[13] = d13
							ps183.OverlayValues[14] = d14
							ps183.OverlayValues[43] = d43
							ps183.OverlayValues[44] = d44
							ps183.OverlayValues[45] = d45
							ps183.OverlayValues[46] = d46
							ps183.OverlayValues[47] = d47
							ps183.OverlayValues[48] = d48
							ps183.OverlayValues[49] = d49
							ps183.OverlayValues[50] = d50
							ps183.OverlayValues[51] = d51
							ps183.OverlayValues[52] = d52
							ps183.OverlayValues[53] = d53
							ps183.OverlayValues[54] = d54
							ps183.OverlayValues[55] = d55
							ps183.OverlayValues[111] = d111
							ps183.OverlayValues[112] = d112
							ps183.OverlayValues[113] = d113
							ps183.OverlayValues[114] = d114
							ps183.OverlayValues[115] = d115
							ps183.OverlayValues[180] = d180
							ps183.OverlayValues[181] = d181
							ps183.OverlayValues[182] = d182
							return bbs[3].RenderPS(ps183)
						}
						if ps.General {
						}
						ps184 := PhiState{General: ps.General}
						ps184.OverlayValues = make([]JITValueDesc, 183)
						ps184.OverlayValues[4] = d4
						ps184.OverlayValues[5] = d5
						ps184.OverlayValues[6] = d6
						ps184.OverlayValues[7] = d7
						ps184.OverlayValues[8] = d8
						ps184.OverlayValues[9] = d9
						ps184.OverlayValues[10] = d10
						ps184.OverlayValues[11] = d11
						ps184.OverlayValues[12] = d12
						ps184.OverlayValues[13] = d13
						ps184.OverlayValues[14] = d14
						ps184.OverlayValues[43] = d43
						ps184.OverlayValues[44] = d44
						ps184.OverlayValues[45] = d45
						ps184.OverlayValues[46] = d46
						ps184.OverlayValues[47] = d47
						ps184.OverlayValues[48] = d48
						ps184.OverlayValues[49] = d49
						ps184.OverlayValues[50] = d50
						ps184.OverlayValues[51] = d51
						ps184.OverlayValues[52] = d52
						ps184.OverlayValues[53] = d53
						ps184.OverlayValues[54] = d54
						ps184.OverlayValues[55] = d55
						ps184.OverlayValues[111] = d111
						ps184.OverlayValues[112] = d112
						ps184.OverlayValues[113] = d113
						ps184.OverlayValues[114] = d114
						ps184.OverlayValues[115] = d115
						ps184.OverlayValues[180] = d180
						ps184.OverlayValues[181] = d181
						ps184.OverlayValues[182] = d182
						return bbs[5].RenderPS(ps184)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					ctx.EmitJump(d182.Condition, lbl4)
					ctx.FreeDesc(&d181)
					snap185 := d4
					snap186 := d5
					snap187 := d6
					snap188 := d7
					snap189 := d8
					snap190 := d9
					snap191 := d10
					snap192 := d11
					snap193 := d12
					snap194 := d13
					snap195 := d14
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
					snap206 := d53
					snap207 := d54
					snap208 := d55
					snap209 := d111
					snap210 := d112
					snap211 := d113
					snap212 := d114
					snap213 := d115
					snap214 := d180
					snap215 := d181
					snap216 := d182
					alloc217 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc217)
					d4 = snap185
					d5 = snap186
					d6 = snap187
					d7 = snap188
					d8 = snap189
					d9 = snap190
					d10 = snap191
					d11 = snap192
					d12 = snap193
					d13 = snap194
					d14 = snap195
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
					d53 = snap206
					d54 = snap207
					d55 = snap208
					d111 = snap209
					d112 = snap210
					d113 = snap211
					d114 = snap212
					d115 = snap213
					d180 = snap214
					d181 = snap215
					d182 = snap216
					ctx.RestoreAllocState(alloc217)
					d4 = snap185
					d5 = snap186
					d6 = snap187
					d7 = snap188
					d8 = snap189
					d9 = snap190
					d10 = snap191
					d11 = snap192
					d12 = snap193
					d13 = snap194
					d14 = snap195
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
					d53 = snap206
					d54 = snap207
					d55 = snap208
					d111 = snap209
					d112 = snap210
					d113 = snap211
					d114 = snap212
					d115 = snap213
					d180 = snap214
					d181 = snap215
					d182 = snap216
					ps218 := PhiState{General: true}
					ps218.OverlayValues = make([]JITValueDesc, 183)
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
					ps218.OverlayValues[43] = d43
					ps218.OverlayValues[44] = d44
					ps218.OverlayValues[45] = d45
					ps218.OverlayValues[46] = d46
					ps218.OverlayValues[47] = d47
					ps218.OverlayValues[48] = d48
					ps218.OverlayValues[49] = d49
					ps218.OverlayValues[50] = d50
					ps218.OverlayValues[51] = d51
					ps218.OverlayValues[52] = d52
					ps218.OverlayValues[53] = d53
					ps218.OverlayValues[54] = d54
					ps218.OverlayValues[55] = d55
					ps218.OverlayValues[111] = d111
					ps218.OverlayValues[112] = d112
					ps218.OverlayValues[113] = d113
					ps218.OverlayValues[114] = d114
					ps218.OverlayValues[115] = d115
					ps218.OverlayValues[180] = d180
					ps218.OverlayValues[181] = d181
					ps218.OverlayValues[182] = d182
					ps219 := PhiState{General: true}
					ps219.OverlayValues = make([]JITValueDesc, 183)
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
					ps219.OverlayValues[43] = d43
					ps219.OverlayValues[44] = d44
					ps219.OverlayValues[45] = d45
					ps219.OverlayValues[46] = d46
					ps219.OverlayValues[47] = d47
					ps219.OverlayValues[48] = d48
					ps219.OverlayValues[49] = d49
					ps219.OverlayValues[50] = d50
					ps219.OverlayValues[51] = d51
					ps219.OverlayValues[52] = d52
					ps219.OverlayValues[53] = d53
					ps219.OverlayValues[54] = d54
					ps219.OverlayValues[55] = d55
					ps219.OverlayValues[111] = d111
					ps219.OverlayValues[112] = d112
					ps219.OverlayValues[113] = d113
					ps219.OverlayValues[114] = d114
					ps219.OverlayValues[115] = d115
					ps219.OverlayValues[180] = d180
					ps219.OverlayValues[181] = d181
					ps219.OverlayValues[182] = d182
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
					snap241 := d53
					snap242 := d54
					snap243 := d55
					snap244 := d111
					snap245 := d112
					snap246 := d113
					snap247 := d114
					snap248 := d115
					snap249 := d180
					snap250 := d181
					snap251 := d182
					alloc252 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps219)
					}
					ctx.RestoreAllocState(alloc252)
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
					d53 = snap241
					d54 = snap242
					d55 = snap243
					d111 = snap244
					d112 = snap245
					d113 = snap246
					d114 = snap247
					d115 = snap248
					d180 = snap249
					d181 = snap250
					d182 = snap251
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps218)
					}
					return result
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d253 := ps.PhiValues[0]
							ctx.EnsureDesc(&d253)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d253)
							} else {
								ctx.EmitStoreToStack(d253, int32(bbs[7].PhiBase)+int32(0))
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
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
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
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != LocNone {
						d253 = ps.OverlayValues[253]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d4 = ps.PhiValues[0]
					}
					if phiHomeOK2 && d4.Loc == LocReg {
						ctx.BindReg(r0, &d4)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDescsTogether(&d4, &d10)
					var d254 JITValueDesc
					if d4.Loc == LocImm && d10.Loc == LocImm {
						d254 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() < d10.Imm.Int())}
					} else if d10.Loc == LocImm {
						r12 := ctx.AllocRegExcept(d4.Reg)
						if d10.Imm.Int() >= -2147483648 && d10.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d4.Reg, int32(d10.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d10.Imm.Int()))
							ctx.EmitCmpInt64(d4.Reg, RegR11)
						}
						d254 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedLess}
						ctx.BindReg(r12, &d254)
					} else if d4.Loc == LocImm {
						r13 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d10.Reg)
						d254 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r13, Condition: CondSignedLess}
						ctx.BindReg(r13, &d254)
					} else {
						r14 := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitCmpInt64(d4.Reg, d10.Reg)
						d254 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r14, Condition: CondSignedLess}
						ctx.BindReg(r14, &d254)
					}
					ctx.FreeDesc(&d10)
					d255 = d254
					ctx.EnsureDesc(&d255)
					if d255.Loc != LocImm && d255.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d255.Loc == LocImm {
						if d255.Imm.Bool() {
							if ps.General {
							}
							ps256 := PhiState{General: ps.General}
							ps256.OverlayValues = make([]JITValueDesc, 256)
							ps256.OverlayValues[4] = d4
							ps256.OverlayValues[5] = d5
							ps256.OverlayValues[6] = d6
							ps256.OverlayValues[7] = d7
							ps256.OverlayValues[8] = d8
							ps256.OverlayValues[9] = d9
							ps256.OverlayValues[10] = d10
							ps256.OverlayValues[11] = d11
							ps256.OverlayValues[12] = d12
							ps256.OverlayValues[13] = d13
							ps256.OverlayValues[14] = d14
							ps256.OverlayValues[43] = d43
							ps256.OverlayValues[44] = d44
							ps256.OverlayValues[45] = d45
							ps256.OverlayValues[46] = d46
							ps256.OverlayValues[47] = d47
							ps256.OverlayValues[48] = d48
							ps256.OverlayValues[49] = d49
							ps256.OverlayValues[50] = d50
							ps256.OverlayValues[51] = d51
							ps256.OverlayValues[52] = d52
							ps256.OverlayValues[53] = d53
							ps256.OverlayValues[54] = d54
							ps256.OverlayValues[55] = d55
							ps256.OverlayValues[111] = d111
							ps256.OverlayValues[112] = d112
							ps256.OverlayValues[113] = d113
							ps256.OverlayValues[114] = d114
							ps256.OverlayValues[115] = d115
							ps256.OverlayValues[180] = d180
							ps256.OverlayValues[181] = d181
							ps256.OverlayValues[182] = d182
							ps256.OverlayValues[253] = d253
							ps256.OverlayValues[254] = d254
							ps256.OverlayValues[255] = d255
							return bbs[8].RenderPS(ps256)
						}
						if ps.General {
						}
						ps257 := PhiState{General: ps.General}
						ps257.OverlayValues = make([]JITValueDesc, 256)
						ps257.OverlayValues[4] = d4
						ps257.OverlayValues[5] = d5
						ps257.OverlayValues[6] = d6
						ps257.OverlayValues[7] = d7
						ps257.OverlayValues[8] = d8
						ps257.OverlayValues[9] = d9
						ps257.OverlayValues[10] = d10
						ps257.OverlayValues[11] = d11
						ps257.OverlayValues[12] = d12
						ps257.OverlayValues[13] = d13
						ps257.OverlayValues[14] = d14
						ps257.OverlayValues[43] = d43
						ps257.OverlayValues[44] = d44
						ps257.OverlayValues[45] = d45
						ps257.OverlayValues[46] = d46
						ps257.OverlayValues[47] = d47
						ps257.OverlayValues[48] = d48
						ps257.OverlayValues[49] = d49
						ps257.OverlayValues[50] = d50
						ps257.OverlayValues[51] = d51
						ps257.OverlayValues[52] = d52
						ps257.OverlayValues[53] = d53
						ps257.OverlayValues[54] = d54
						ps257.OverlayValues[55] = d55
						ps257.OverlayValues[111] = d111
						ps257.OverlayValues[112] = d112
						ps257.OverlayValues[113] = d113
						ps257.OverlayValues[114] = d114
						ps257.OverlayValues[115] = d115
						ps257.OverlayValues[180] = d180
						ps257.OverlayValues[181] = d181
						ps257.OverlayValues[182] = d182
						ps257.OverlayValues[253] = d253
						ps257.OverlayValues[254] = d254
						ps257.OverlayValues[255] = d255
						return bbs[9].RenderPS(ps257)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d258 := ps.PhiValues[0]
							ctx.EnsureDesc(&d258)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d258)
							} else {
								ctx.EmitStoreToStack(d258, int32(bbs[7].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					ctx.EmitJump(d255.Condition, lbl9)
					ctx.FreeDesc(&d254)
					snap259 := d4
					snap260 := d5
					snap261 := d6
					snap262 := d7
					snap263 := d8
					snap264 := d9
					snap265 := d10
					snap266 := d11
					snap267 := d12
					snap268 := d13
					snap269 := d14
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
					snap280 := d53
					snap281 := d54
					snap282 := d55
					snap283 := d111
					snap284 := d112
					snap285 := d113
					snap286 := d114
					snap287 := d115
					snap288 := d180
					snap289 := d181
					snap290 := d182
					snap291 := d253
					snap292 := d254
					snap293 := d255
					snap294 := d258
					alloc295 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc295)
					d4 = snap259
					d5 = snap260
					d6 = snap261
					d7 = snap262
					d8 = snap263
					d9 = snap264
					d10 = snap265
					d11 = snap266
					d12 = snap267
					d13 = snap268
					d14 = snap269
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
					d53 = snap280
					d54 = snap281
					d55 = snap282
					d111 = snap283
					d112 = snap284
					d113 = snap285
					d114 = snap286
					d115 = snap287
					d180 = snap288
					d181 = snap289
					d182 = snap290
					d253 = snap291
					d254 = snap292
					d255 = snap293
					d258 = snap294
					ctx.RestoreAllocState(alloc295)
					d4 = snap259
					d5 = snap260
					d6 = snap261
					d7 = snap262
					d8 = snap263
					d9 = snap264
					d10 = snap265
					d11 = snap266
					d12 = snap267
					d13 = snap268
					d14 = snap269
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
					d53 = snap280
					d54 = snap281
					d55 = snap282
					d111 = snap283
					d112 = snap284
					d113 = snap285
					d114 = snap286
					d115 = snap287
					d180 = snap288
					d181 = snap289
					d182 = snap290
					d253 = snap291
					d254 = snap292
					d255 = snap293
					d258 = snap294
					ps296 := PhiState{General: true}
					ps296.OverlayValues = make([]JITValueDesc, 259)
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
					ps296.OverlayValues[43] = d43
					ps296.OverlayValues[44] = d44
					ps296.OverlayValues[45] = d45
					ps296.OverlayValues[46] = d46
					ps296.OverlayValues[47] = d47
					ps296.OverlayValues[48] = d48
					ps296.OverlayValues[49] = d49
					ps296.OverlayValues[50] = d50
					ps296.OverlayValues[51] = d51
					ps296.OverlayValues[52] = d52
					ps296.OverlayValues[53] = d53
					ps296.OverlayValues[54] = d54
					ps296.OverlayValues[55] = d55
					ps296.OverlayValues[111] = d111
					ps296.OverlayValues[112] = d112
					ps296.OverlayValues[113] = d113
					ps296.OverlayValues[114] = d114
					ps296.OverlayValues[115] = d115
					ps296.OverlayValues[180] = d180
					ps296.OverlayValues[181] = d181
					ps296.OverlayValues[182] = d182
					ps296.OverlayValues[253] = d253
					ps296.OverlayValues[254] = d254
					ps296.OverlayValues[255] = d255
					ps296.OverlayValues[258] = d258
					ps297 := PhiState{General: true}
					ps297.OverlayValues = make([]JITValueDesc, 259)
					ps297.OverlayValues[4] = d4
					ps297.OverlayValues[5] = d5
					ps297.OverlayValues[6] = d6
					ps297.OverlayValues[7] = d7
					ps297.OverlayValues[8] = d8
					ps297.OverlayValues[9] = d9
					ps297.OverlayValues[10] = d10
					ps297.OverlayValues[11] = d11
					ps297.OverlayValues[12] = d12
					ps297.OverlayValues[13] = d13
					ps297.OverlayValues[14] = d14
					ps297.OverlayValues[43] = d43
					ps297.OverlayValues[44] = d44
					ps297.OverlayValues[45] = d45
					ps297.OverlayValues[46] = d46
					ps297.OverlayValues[47] = d47
					ps297.OverlayValues[48] = d48
					ps297.OverlayValues[49] = d49
					ps297.OverlayValues[50] = d50
					ps297.OverlayValues[51] = d51
					ps297.OverlayValues[52] = d52
					ps297.OverlayValues[53] = d53
					ps297.OverlayValues[54] = d54
					ps297.OverlayValues[55] = d55
					ps297.OverlayValues[111] = d111
					ps297.OverlayValues[112] = d112
					ps297.OverlayValues[113] = d113
					ps297.OverlayValues[114] = d114
					ps297.OverlayValues[115] = d115
					ps297.OverlayValues[180] = d180
					ps297.OverlayValues[181] = d181
					ps297.OverlayValues[182] = d182
					ps297.OverlayValues[253] = d253
					ps297.OverlayValues[254] = d254
					ps297.OverlayValues[255] = d255
					ps297.OverlayValues[258] = d258
					snap298 := d4
					snap299 := d5
					snap300 := d6
					snap301 := d7
					snap302 := d8
					snap303 := d9
					snap304 := d10
					snap305 := d11
					snap306 := d12
					snap307 := d13
					snap308 := d14
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
					snap319 := d53
					snap320 := d54
					snap321 := d55
					snap322 := d111
					snap323 := d112
					snap324 := d113
					snap325 := d114
					snap326 := d115
					snap327 := d180
					snap328 := d181
					snap329 := d182
					snap330 := d253
					snap331 := d254
					snap332 := d255
					snap333 := d258
					alloc334 := ctx.SnapshotAllocState()
					if !bbs[9].Rendered {
						bbs[9].RenderPS(ps297)
					}
					ctx.RestoreAllocState(alloc334)
					d4 = snap298
					d5 = snap299
					d6 = snap300
					d7 = snap301
					d8 = snap302
					d9 = snap303
					d10 = snap304
					d11 = snap305
					d12 = snap306
					d13 = snap307
					d14 = snap308
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
					d53 = snap319
					d54 = snap320
					d55 = snap321
					d111 = snap322
					d112 = snap323
					d113 = snap324
					d114 = snap325
					d115 = snap326
					d180 = snap327
					d181 = snap328
					d182 = snap329
					d253 = snap330
					d254 = snap331
					d255 = snap332
					d258 = snap333
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps296)
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
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
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != LocNone {
						d253 = ps.OverlayValues[253]
					}
					if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != LocNone {
						d254 = ps.OverlayValues[254]
					}
					if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != LocNone {
						d255 = ps.OverlayValues[255]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d46)
					var d335 JITValueDesc
					ctx.EnsureDesc(&d53)
					if d53.Loc == LocRegPair || d53.Loc == LocRegTriple {
						d335 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg2}
						ctx.BindReg(d53.Reg2, &d335)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d53)
					ctx.EnsureDesc(&d46)
					ctx.EnsureDesc(&d335)
					var d337 JITValueDesc
					if d335.Loc == LocImm && d46.Loc == LocImm {
						d337 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d335.Imm.Int() - d46.Imm.Int())}
					} else {
						r15 := ctx.AllocReg()
						if d335.Loc == LocImm {
							ctx.EmitMovRegImm64(r15, uint64(d335.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r15, d335.Reg)
						}
						if d46.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d46.Imm.Int()))
							ctx.EmitSubInt64(r15, RegR11)
						} else {
							ctx.EmitSubInt64(r15, d46.Reg)
						}
						d337 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
						ctx.BindReg(r15, &d337)
					}
					var d338 JITValueDesc
					r16 := ctx.EmitSliceDataAfterLow(&d53, &d46, 16)
					d338 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
					ctx.BindReg(r16, &d338)
					ctx.BindReg(r16, &d338)
					var d339 JITValueDesc
					var r17 Reg
					var r18 Reg
					ctx.SyncDesc(&d338)
					ctx.EnsureDesc(&d338)
					if d338.Loc == LocImm {
						r17 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r17, uint64(d338.Imm.Int()))
					} else {
						r17 = d338.Reg
					}
					ctx.ProtectReg(r17)
					ctx.SyncDesc(&d337)
					ctx.EnsureDesc(&d337)
					if d337.Loc == LocImm {
						r18 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r18, uint64(d337.Imm.Int()))
					} else {
						r18 = d337.Reg
					}
					ctx.ProtectReg(r18)
					r19 := ctx.EmitSliceCapAfterLow(&d53, &d46, r17, r18)
					ctx.UnprotectReg(r18)
					ctx.UnprotectReg(r17)
					d339 = JITValueDesc{Loc: LocRegTriple, Reg: r17, Reg2: r18, Reg3: r19}
					ctx.BindReg(r17, &d339)
					ctx.BindReg(r18, &d339)
					ctx.BindReg(r19, &d339)
					ctx.BindReg(r17, &d339)
					ctx.BindReg(r18, &d339)
					ctx.BindReg(r19, &d339)
					ctx.EnsureDesc(&d53)
					ctx.EnsureDesc(&d339)
					ctx.EnsureDesc(&d53)
					ctx.EnsureDesc(&d339)
					callResults340 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d53, d339}, []uint8{1}, []uint8{0})
					d341 = callResults340[0]
					d341.Type = tagInt
					var d342 JITValueDesc
					if d53.SliceSizeKnown {
						d342 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d53.KnownSliceLen))}
					} else if d53.Loc == LocImm {
						d342 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d53.StackOff))}
					} else if d53.Loc == LocStackTriple {
						d342 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d53.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d53)
						if d53.Loc == LocRegPair || d53.Loc == LocRegTriple {
							d342 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg2, ID: 0}
						} else if d53.Loc == LocReg {
							d342 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d342)
					ctx.EnsureDesc(&d46)
					ctx.EnsureDescsTogether(&d342, &d46)
					var d343 JITValueDesc
					if d342.Loc == LocImm && d46.Loc == LocImm {
						d343 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d342.Imm.Int() - d46.Imm.Int())}
					} else if d46.Loc == LocImm && d46.Imm.Int() == 0 {
						var r20 Reg
						if phiHomeOK3 && r1 != d342.Reg {
							r20 = r1
						} else {
							r20 = ctx.AllocRegExcept(d342.Reg)
						}
						ctx.EmitMovRegReg(r20, d342.Reg)
						d343 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r20}
						ctx.BindReg(r20, &d343)
					} else if d342.Loc == LocImm {
						var scratch Reg
						if phiHomeOK3 && r1 != d46.Reg {
							scratch = r1
						} else {
							scratch = ctx.AllocRegExcept(d46.Reg)
						}
						ctx.EmitMovRegImm64(scratch, uint64(d342.Imm.Int()))
						ctx.EmitSubInt64(scratch, d46.Reg)
						d343 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d343)
					} else if d46.Loc == LocImm {
						var scratch Reg
						if phiHomeOK3 && r1 != d342.Reg {
							scratch = r1
						} else {
							scratch = ctx.AllocRegExcept(d342.Reg)
						}
						ctx.EmitMovRegReg(scratch, d342.Reg)
						if d46.Imm.Int() >= -2147483648 && d46.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d46.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d46.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d343 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d343)
					} else {
						var r21 Reg
						if phiHomeOK3 && r1 != d342.Reg && r1 != d46.Reg {
							r21 = r1
						} else {
							r21 = ctx.AllocRegExcept(d342.Reg, d46.Reg)
						}
						ctx.EmitMovRegReg(r21, d342.Reg)
						ctx.EmitSubInt64(r21, d46.Reg)
						d343 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r21}
						ctx.BindReg(r21, &d343)
					}
					if d343.Loc == LocReg && d342.Loc == LocReg && d343.Reg == d342.Reg {
						ctx.TransferReg(d342.Reg)
						d342.Loc = LocNone
					}
					ctx.FreeDesc(&d342)
					ctx.FreeDesc(&d46)
					if ps.General {
						ctx.SyncDesc(&d343)
						if d343.Loc == LocReg {
							ctx.ProtectReg(d343.Reg)
						} else if d343.Loc == LocRegPair {
							ctx.ProtectReg(d343.Reg)
							ctx.ProtectReg(d343.Reg2)
						}
						d344 = d343
						if d344.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d344)
						if phiHomeOK3 {
							ctx.EmitMovToReg(r1, d344)
						} else {
							ctx.EmitStoreToStack(d344, int32(bbs[10].PhiBase)+int32(0))
						}
						if d343.Loc == LocReg {
							ctx.UnprotectReg(d343.Reg)
						} else if d343.Loc == LocRegPair {
							ctx.UnprotectReg(d343.Reg)
							ctx.UnprotectReg(d343.Reg2)
						}
					}
					ps345 := PhiState{General: ps.General}
					ps345.OverlayValues = make([]JITValueDesc, 345)
					ps345.OverlayValues[4] = d4
					ps345.OverlayValues[5] = d5
					ps345.OverlayValues[6] = d6
					ps345.OverlayValues[7] = d7
					ps345.OverlayValues[8] = d8
					ps345.OverlayValues[9] = d9
					ps345.OverlayValues[10] = d10
					ps345.OverlayValues[11] = d11
					ps345.OverlayValues[12] = d12
					ps345.OverlayValues[13] = d13
					ps345.OverlayValues[14] = d14
					ps345.OverlayValues[43] = d43
					ps345.OverlayValues[44] = d44
					ps345.OverlayValues[45] = d45
					ps345.OverlayValues[46] = d46
					ps345.OverlayValues[47] = d47
					ps345.OverlayValues[48] = d48
					ps345.OverlayValues[49] = d49
					ps345.OverlayValues[50] = d50
					ps345.OverlayValues[51] = d51
					ps345.OverlayValues[52] = d52
					ps345.OverlayValues[53] = d53
					ps345.OverlayValues[54] = d54
					ps345.OverlayValues[55] = d55
					ps345.OverlayValues[111] = d111
					ps345.OverlayValues[112] = d112
					ps345.OverlayValues[113] = d113
					ps345.OverlayValues[114] = d114
					ps345.OverlayValues[115] = d115
					ps345.OverlayValues[180] = d180
					ps345.OverlayValues[181] = d181
					ps345.OverlayValues[182] = d182
					ps345.OverlayValues[253] = d253
					ps345.OverlayValues[254] = d254
					ps345.OverlayValues[255] = d255
					ps345.OverlayValues[258] = d258
					ps345.OverlayValues[335] = d335
					ps345.OverlayValues[336] = d336
					ps345.OverlayValues[337] = d337
					ps345.OverlayValues[338] = d338
					ps345.OverlayValues[339] = d339
					ps345.OverlayValues[341] = d341
					ps345.OverlayValues[342] = d342
					ps345.OverlayValues[343] = d343
					ps345.OverlayValues[344] = d344
					ps345.PhiValues = make([]JITValueDesc, 1)
					d346 = d343
					ps345.PhiValues[0] = d346
					if ps345.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps345)
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
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
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
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != LocNone {
						d253 = ps.OverlayValues[253]
					}
					if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != LocNone {
						d254 = ps.OverlayValues[254]
					}
					if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != LocNone {
						d255 = ps.OverlayValues[255]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
						d335 = ps.OverlayValues[335]
					}
					if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
						d336 = ps.OverlayValues[336]
					}
					if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != LocNone {
						d337 = ps.OverlayValues[337]
					}
					if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != LocNone {
						d338 = ps.OverlayValues[338]
					}
					if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
						d339 = ps.OverlayValues[339]
					}
					if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != LocNone {
						d341 = ps.OverlayValues[341]
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
					if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
						d346 = ps.OverlayValues[346]
					}
					ctx.ReclaimUntrackedRegs()
					d347 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d347)
					if d347.Loc == LocRegPair || d347.Loc == LocStackPair || d347.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d347, &result)
						result.Type = d347.Type
					} else {
						switch d347.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d347)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d347)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d347)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d347, &result)
							result.Type = d347.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d348 := ps.PhiValues[0]
							ctx.EnsureDesc(&d348)
							if phiHomeOK3 {
								ctx.EmitMovToReg(r1, d348)
							} else {
								ctx.EmitStoreToStack(d348, int32(bbs[10].PhiBase)+int32(0))
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
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
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
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != LocNone {
						d253 = ps.OverlayValues[253]
					}
					if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != LocNone {
						d254 = ps.OverlayValues[254]
					}
					if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != LocNone {
						d255 = ps.OverlayValues[255]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
						d335 = ps.OverlayValues[335]
					}
					if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
						d336 = ps.OverlayValues[336]
					}
					if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != LocNone {
						d337 = ps.OverlayValues[337]
					}
					if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != LocNone {
						d338 = ps.OverlayValues[338]
					}
					if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
						d339 = ps.OverlayValues[339]
					}
					if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != LocNone {
						d341 = ps.OverlayValues[341]
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
					if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
						d346 = ps.OverlayValues[346]
					}
					if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != LocNone {
						d347 = ps.OverlayValues[347]
					}
					if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
						d348 = ps.OverlayValues[348]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d5 = ps.PhiValues[0]
					}
					if phiHomeOK3 && d5.Loc == LocReg {
						ctx.BindReg(r1, &d5)
					}
					ctx.ReclaimUntrackedRegs()
					var d349 JITValueDesc
					if d53.SliceSizeKnown {
						d349 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d53.KnownSliceLen))}
					} else if d53.Loc == LocImm {
						d349 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d53.StackOff))}
					} else if d53.Loc == LocStackTriple {
						d349 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d53.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d53)
						if d53.Loc == LocRegPair || d53.Loc == LocRegTriple {
							d349 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg2, ID: 0}
						} else if d53.Loc == LocReg {
							d349 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d349)
					ctx.EnsureDescsTogether(&d5, &d349)
					var d350 JITValueDesc
					if d5.Loc == LocImm && d349.Loc == LocImm {
						d350 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.Int() < d349.Imm.Int())}
					} else if d349.Loc == LocImm {
						r22 := ctx.AllocRegExcept(d5.Reg)
						if d349.Imm.Int() >= -2147483648 && d349.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d5.Reg, int32(d349.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d349.Imm.Int()))
							ctx.EmitCmpInt64(d5.Reg, RegR11)
						}
						d350 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondSignedLess}
						ctx.BindReg(r22, &d350)
					} else if d5.Loc == LocImm {
						r23 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d349.Reg)
						d350 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r23, Condition: CondSignedLess}
						ctx.BindReg(r23, &d350)
					} else {
						r24 := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitCmpInt64(d5.Reg, d349.Reg)
						d350 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r24, Condition: CondSignedLess}
						ctx.BindReg(r24, &d350)
					}
					ctx.FreeDesc(&d349)
					d351 = d350
					ctx.EnsureDesc(&d351)
					if d351.Loc != LocImm && d351.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d351.Loc == LocImm {
						if d351.Imm.Bool() {
							if ps.General {
							}
							ps352 := PhiState{General: ps.General}
							ps352.OverlayValues = make([]JITValueDesc, 352)
							ps352.OverlayValues[4] = d4
							ps352.OverlayValues[5] = d5
							ps352.OverlayValues[6] = d6
							ps352.OverlayValues[7] = d7
							ps352.OverlayValues[8] = d8
							ps352.OverlayValues[9] = d9
							ps352.OverlayValues[10] = d10
							ps352.OverlayValues[11] = d11
							ps352.OverlayValues[12] = d12
							ps352.OverlayValues[13] = d13
							ps352.OverlayValues[14] = d14
							ps352.OverlayValues[43] = d43
							ps352.OverlayValues[44] = d44
							ps352.OverlayValues[45] = d45
							ps352.OverlayValues[46] = d46
							ps352.OverlayValues[47] = d47
							ps352.OverlayValues[48] = d48
							ps352.OverlayValues[49] = d49
							ps352.OverlayValues[50] = d50
							ps352.OverlayValues[51] = d51
							ps352.OverlayValues[52] = d52
							ps352.OverlayValues[53] = d53
							ps352.OverlayValues[54] = d54
							ps352.OverlayValues[55] = d55
							ps352.OverlayValues[111] = d111
							ps352.OverlayValues[112] = d112
							ps352.OverlayValues[113] = d113
							ps352.OverlayValues[114] = d114
							ps352.OverlayValues[115] = d115
							ps352.OverlayValues[180] = d180
							ps352.OverlayValues[181] = d181
							ps352.OverlayValues[182] = d182
							ps352.OverlayValues[253] = d253
							ps352.OverlayValues[254] = d254
							ps352.OverlayValues[255] = d255
							ps352.OverlayValues[258] = d258
							ps352.OverlayValues[335] = d335
							ps352.OverlayValues[336] = d336
							ps352.OverlayValues[337] = d337
							ps352.OverlayValues[338] = d338
							ps352.OverlayValues[339] = d339
							ps352.OverlayValues[341] = d341
							ps352.OverlayValues[342] = d342
							ps352.OverlayValues[343] = d343
							ps352.OverlayValues[344] = d344
							ps352.OverlayValues[346] = d346
							ps352.OverlayValues[347] = d347
							ps352.OverlayValues[348] = d348
							ps352.OverlayValues[349] = d349
							ps352.OverlayValues[350] = d350
							ps352.OverlayValues[351] = d351
							return bbs[11].RenderPS(ps352)
						}
						if ps.General {
						}
						ps353 := PhiState{General: ps.General}
						ps353.OverlayValues = make([]JITValueDesc, 352)
						ps353.OverlayValues[4] = d4
						ps353.OverlayValues[5] = d5
						ps353.OverlayValues[6] = d6
						ps353.OverlayValues[7] = d7
						ps353.OverlayValues[8] = d8
						ps353.OverlayValues[9] = d9
						ps353.OverlayValues[10] = d10
						ps353.OverlayValues[11] = d11
						ps353.OverlayValues[12] = d12
						ps353.OverlayValues[13] = d13
						ps353.OverlayValues[14] = d14
						ps353.OverlayValues[43] = d43
						ps353.OverlayValues[44] = d44
						ps353.OverlayValues[45] = d45
						ps353.OverlayValues[46] = d46
						ps353.OverlayValues[47] = d47
						ps353.OverlayValues[48] = d48
						ps353.OverlayValues[49] = d49
						ps353.OverlayValues[50] = d50
						ps353.OverlayValues[51] = d51
						ps353.OverlayValues[52] = d52
						ps353.OverlayValues[53] = d53
						ps353.OverlayValues[54] = d54
						ps353.OverlayValues[55] = d55
						ps353.OverlayValues[111] = d111
						ps353.OverlayValues[112] = d112
						ps353.OverlayValues[113] = d113
						ps353.OverlayValues[114] = d114
						ps353.OverlayValues[115] = d115
						ps353.OverlayValues[180] = d180
						ps353.OverlayValues[181] = d181
						ps353.OverlayValues[182] = d182
						ps353.OverlayValues[253] = d253
						ps353.OverlayValues[254] = d254
						ps353.OverlayValues[255] = d255
						ps353.OverlayValues[258] = d258
						ps353.OverlayValues[335] = d335
						ps353.OverlayValues[336] = d336
						ps353.OverlayValues[337] = d337
						ps353.OverlayValues[338] = d338
						ps353.OverlayValues[339] = d339
						ps353.OverlayValues[341] = d341
						ps353.OverlayValues[342] = d342
						ps353.OverlayValues[343] = d343
						ps353.OverlayValues[344] = d344
						ps353.OverlayValues[346] = d346
						ps353.OverlayValues[347] = d347
						ps353.OverlayValues[348] = d348
						ps353.OverlayValues[349] = d349
						ps353.OverlayValues[350] = d350
						ps353.OverlayValues[351] = d351
						return bbs[12].RenderPS(ps353)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d354 := ps.PhiValues[0]
							ctx.EnsureDesc(&d354)
							if phiHomeOK3 {
								ctx.EmitMovToReg(r1, d354)
							} else {
								ctx.EmitStoreToStack(d354, int32(bbs[10].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[10].RenderPS(ps)
					}
					ctx.EmitJump(d351.Condition, lbl12)
					ctx.FreeDesc(&d350)
					snap355 := d4
					snap356 := d5
					snap357 := d6
					snap358 := d7
					snap359 := d8
					snap360 := d9
					snap361 := d10
					snap362 := d11
					snap363 := d12
					snap364 := d13
					snap365 := d14
					snap366 := d43
					snap367 := d44
					snap368 := d45
					snap369 := d46
					snap370 := d47
					snap371 := d48
					snap372 := d49
					snap373 := d50
					snap374 := d51
					snap375 := d52
					snap376 := d53
					snap377 := d54
					snap378 := d55
					snap379 := d111
					snap380 := d112
					snap381 := d113
					snap382 := d114
					snap383 := d115
					snap384 := d180
					snap385 := d181
					snap386 := d182
					snap387 := d253
					snap388 := d254
					snap389 := d255
					snap390 := d258
					snap391 := d335
					snap392 := d336
					snap393 := d337
					snap394 := d338
					snap395 := d339
					snap396 := d341
					snap397 := d342
					snap398 := d343
					snap399 := d344
					snap400 := d346
					snap401 := d347
					snap402 := d348
					snap403 := d349
					snap404 := d350
					snap405 := d351
					snap406 := d354
					alloc407 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc407)
					d4 = snap355
					d5 = snap356
					d6 = snap357
					d7 = snap358
					d8 = snap359
					d9 = snap360
					d10 = snap361
					d11 = snap362
					d12 = snap363
					d13 = snap364
					d14 = snap365
					d43 = snap366
					d44 = snap367
					d45 = snap368
					d46 = snap369
					d47 = snap370
					d48 = snap371
					d49 = snap372
					d50 = snap373
					d51 = snap374
					d52 = snap375
					d53 = snap376
					d54 = snap377
					d55 = snap378
					d111 = snap379
					d112 = snap380
					d113 = snap381
					d114 = snap382
					d115 = snap383
					d180 = snap384
					d181 = snap385
					d182 = snap386
					d253 = snap387
					d254 = snap388
					d255 = snap389
					d258 = snap390
					d335 = snap391
					d336 = snap392
					d337 = snap393
					d338 = snap394
					d339 = snap395
					d341 = snap396
					d342 = snap397
					d343 = snap398
					d344 = snap399
					d346 = snap400
					d347 = snap401
					d348 = snap402
					d349 = snap403
					d350 = snap404
					d351 = snap405
					d354 = snap406
					ctx.RestoreAllocState(alloc407)
					d4 = snap355
					d5 = snap356
					d6 = snap357
					d7 = snap358
					d8 = snap359
					d9 = snap360
					d10 = snap361
					d11 = snap362
					d12 = snap363
					d13 = snap364
					d14 = snap365
					d43 = snap366
					d44 = snap367
					d45 = snap368
					d46 = snap369
					d47 = snap370
					d48 = snap371
					d49 = snap372
					d50 = snap373
					d51 = snap374
					d52 = snap375
					d53 = snap376
					d54 = snap377
					d55 = snap378
					d111 = snap379
					d112 = snap380
					d113 = snap381
					d114 = snap382
					d115 = snap383
					d180 = snap384
					d181 = snap385
					d182 = snap386
					d253 = snap387
					d254 = snap388
					d255 = snap389
					d258 = snap390
					d335 = snap391
					d336 = snap392
					d337 = snap393
					d338 = snap394
					d339 = snap395
					d341 = snap396
					d342 = snap397
					d343 = snap398
					d344 = snap399
					d346 = snap400
					d347 = snap401
					d348 = snap402
					d349 = snap403
					d350 = snap404
					d351 = snap405
					d354 = snap406
					ps408 := PhiState{General: true}
					ps408.OverlayValues = make([]JITValueDesc, 355)
					ps408.OverlayValues[4] = d4
					ps408.OverlayValues[5] = d5
					ps408.OverlayValues[6] = d6
					ps408.OverlayValues[7] = d7
					ps408.OverlayValues[8] = d8
					ps408.OverlayValues[9] = d9
					ps408.OverlayValues[10] = d10
					ps408.OverlayValues[11] = d11
					ps408.OverlayValues[12] = d12
					ps408.OverlayValues[13] = d13
					ps408.OverlayValues[14] = d14
					ps408.OverlayValues[43] = d43
					ps408.OverlayValues[44] = d44
					ps408.OverlayValues[45] = d45
					ps408.OverlayValues[46] = d46
					ps408.OverlayValues[47] = d47
					ps408.OverlayValues[48] = d48
					ps408.OverlayValues[49] = d49
					ps408.OverlayValues[50] = d50
					ps408.OverlayValues[51] = d51
					ps408.OverlayValues[52] = d52
					ps408.OverlayValues[53] = d53
					ps408.OverlayValues[54] = d54
					ps408.OverlayValues[55] = d55
					ps408.OverlayValues[111] = d111
					ps408.OverlayValues[112] = d112
					ps408.OverlayValues[113] = d113
					ps408.OverlayValues[114] = d114
					ps408.OverlayValues[115] = d115
					ps408.OverlayValues[180] = d180
					ps408.OverlayValues[181] = d181
					ps408.OverlayValues[182] = d182
					ps408.OverlayValues[253] = d253
					ps408.OverlayValues[254] = d254
					ps408.OverlayValues[255] = d255
					ps408.OverlayValues[258] = d258
					ps408.OverlayValues[335] = d335
					ps408.OverlayValues[336] = d336
					ps408.OverlayValues[337] = d337
					ps408.OverlayValues[338] = d338
					ps408.OverlayValues[339] = d339
					ps408.OverlayValues[341] = d341
					ps408.OverlayValues[342] = d342
					ps408.OverlayValues[343] = d343
					ps408.OverlayValues[344] = d344
					ps408.OverlayValues[346] = d346
					ps408.OverlayValues[347] = d347
					ps408.OverlayValues[348] = d348
					ps408.OverlayValues[349] = d349
					ps408.OverlayValues[350] = d350
					ps408.OverlayValues[351] = d351
					ps408.OverlayValues[354] = d354
					ps409 := PhiState{General: true}
					ps409.OverlayValues = make([]JITValueDesc, 355)
					ps409.OverlayValues[4] = d4
					ps409.OverlayValues[5] = d5
					ps409.OverlayValues[6] = d6
					ps409.OverlayValues[7] = d7
					ps409.OverlayValues[8] = d8
					ps409.OverlayValues[9] = d9
					ps409.OverlayValues[10] = d10
					ps409.OverlayValues[11] = d11
					ps409.OverlayValues[12] = d12
					ps409.OverlayValues[13] = d13
					ps409.OverlayValues[14] = d14
					ps409.OverlayValues[43] = d43
					ps409.OverlayValues[44] = d44
					ps409.OverlayValues[45] = d45
					ps409.OverlayValues[46] = d46
					ps409.OverlayValues[47] = d47
					ps409.OverlayValues[48] = d48
					ps409.OverlayValues[49] = d49
					ps409.OverlayValues[50] = d50
					ps409.OverlayValues[51] = d51
					ps409.OverlayValues[52] = d52
					ps409.OverlayValues[53] = d53
					ps409.OverlayValues[54] = d54
					ps409.OverlayValues[55] = d55
					ps409.OverlayValues[111] = d111
					ps409.OverlayValues[112] = d112
					ps409.OverlayValues[113] = d113
					ps409.OverlayValues[114] = d114
					ps409.OverlayValues[115] = d115
					ps409.OverlayValues[180] = d180
					ps409.OverlayValues[181] = d181
					ps409.OverlayValues[182] = d182
					ps409.OverlayValues[253] = d253
					ps409.OverlayValues[254] = d254
					ps409.OverlayValues[255] = d255
					ps409.OverlayValues[258] = d258
					ps409.OverlayValues[335] = d335
					ps409.OverlayValues[336] = d336
					ps409.OverlayValues[337] = d337
					ps409.OverlayValues[338] = d338
					ps409.OverlayValues[339] = d339
					ps409.OverlayValues[341] = d341
					ps409.OverlayValues[342] = d342
					ps409.OverlayValues[343] = d343
					ps409.OverlayValues[344] = d344
					ps409.OverlayValues[346] = d346
					ps409.OverlayValues[347] = d347
					ps409.OverlayValues[348] = d348
					ps409.OverlayValues[349] = d349
					ps409.OverlayValues[350] = d350
					ps409.OverlayValues[351] = d351
					ps409.OverlayValues[354] = d354
					snap410 := d4
					snap411 := d5
					snap412 := d6
					snap413 := d7
					snap414 := d8
					snap415 := d9
					snap416 := d10
					snap417 := d11
					snap418 := d12
					snap419 := d13
					snap420 := d14
					snap421 := d43
					snap422 := d44
					snap423 := d45
					snap424 := d46
					snap425 := d47
					snap426 := d48
					snap427 := d49
					snap428 := d50
					snap429 := d51
					snap430 := d52
					snap431 := d53
					snap432 := d54
					snap433 := d55
					snap434 := d111
					snap435 := d112
					snap436 := d113
					snap437 := d114
					snap438 := d115
					snap439 := d180
					snap440 := d181
					snap441 := d182
					snap442 := d253
					snap443 := d254
					snap444 := d255
					snap445 := d258
					snap446 := d335
					snap447 := d336
					snap448 := d337
					snap449 := d338
					snap450 := d339
					snap451 := d341
					snap452 := d342
					snap453 := d343
					snap454 := d344
					snap455 := d346
					snap456 := d347
					snap457 := d348
					snap458 := d349
					snap459 := d350
					snap460 := d351
					snap461 := d354
					alloc462 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps409)
					}
					ctx.RestoreAllocState(alloc462)
					d4 = snap410
					d5 = snap411
					d6 = snap412
					d7 = snap413
					d8 = snap414
					d9 = snap415
					d10 = snap416
					d11 = snap417
					d12 = snap418
					d13 = snap419
					d14 = snap420
					d43 = snap421
					d44 = snap422
					d45 = snap423
					d46 = snap424
					d47 = snap425
					d48 = snap426
					d49 = snap427
					d50 = snap428
					d51 = snap429
					d52 = snap430
					d53 = snap431
					d54 = snap432
					d55 = snap433
					d111 = snap434
					d112 = snap435
					d113 = snap436
					d114 = snap437
					d115 = snap438
					d180 = snap439
					d181 = snap440
					d182 = snap441
					d253 = snap442
					d254 = snap443
					d255 = snap444
					d258 = snap445
					d335 = snap446
					d336 = snap447
					d337 = snap448
					d338 = snap449
					d339 = snap450
					d341 = snap451
					d342 = snap452
					d343 = snap453
					d344 = snap454
					d346 = snap455
					d347 = snap456
					d348 = snap457
					d349 = snap458
					d350 = snap459
					d351 = snap460
					d354 = snap461
					if !bbs[11].Rendered {
						return bbs[11].RenderPS(ps408)
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
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
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
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != LocNone {
						d253 = ps.OverlayValues[253]
					}
					if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != LocNone {
						d254 = ps.OverlayValues[254]
					}
					if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != LocNone {
						d255 = ps.OverlayValues[255]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
						d335 = ps.OverlayValues[335]
					}
					if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
						d336 = ps.OverlayValues[336]
					}
					if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != LocNone {
						d337 = ps.OverlayValues[337]
					}
					if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != LocNone {
						d338 = ps.OverlayValues[338]
					}
					if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
						d339 = ps.OverlayValues[339]
					}
					if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != LocNone {
						d341 = ps.OverlayValues[341]
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
					if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
						d346 = ps.OverlayValues[346]
					}
					if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != LocNone {
						d347 = ps.OverlayValues[347]
					}
					if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
						d348 = ps.OverlayValues[348]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
					}
					if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
						d350 = ps.OverlayValues[350]
					}
					if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
						d351 = ps.OverlayValues[351]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					ctx.ReclaimUntrackedRegs()
					d463 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d5)
					ctx.SyncDesc(&d463)
					d464 = d53
					d464.ID = 0
					d465 = d5
					d465.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d464, &d465, &d463, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d5)
						d465 = d5
						d465.ID = 0
						ctx.EmitStoreScmerSliceElement(&d464, &d465, &d463, int32(16))
					}
					ctx.FreeDesc(&d465)
					ctx.FreeDesc(&d463)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d5)
					var d466 JITValueDesc
					if d5.Loc == LocImm {
						d466 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d5.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK3 {
							scratch = r1
						} else {
							scratch = ctx.AllocRegExcept(d5.Reg)
						}
						ctx.EmitMovRegReg(scratch, d5.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d466 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d466)
					}
					if d466.Loc == LocReg && d5.Loc == LocReg && d466.Reg == d5.Reg {
						ctx.TransferReg(d5.Reg)
						d5.Loc = LocNone
					}
					if ps.General {
						ctx.SyncDesc(&d466)
						if d466.Loc == LocReg {
							ctx.ProtectReg(d466.Reg)
						} else if d466.Loc == LocRegPair {
							ctx.ProtectReg(d466.Reg)
							ctx.ProtectReg(d466.Reg2)
						}
						d467 = d466
						if d467.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d467)
						if phiHomeOK3 {
							ctx.EmitMovToReg(r1, d467)
						} else {
							ctx.EmitStoreToStack(d467, int32(bbs[10].PhiBase)+int32(0))
						}
						if d466.Loc == LocReg {
							ctx.UnprotectReg(d466.Reg)
						} else if d466.Loc == LocRegPair {
							ctx.UnprotectReg(d466.Reg)
							ctx.UnprotectReg(d466.Reg2)
						}
					}
					ps468 := PhiState{General: ps.General}
					ps468.OverlayValues = make([]JITValueDesc, 468)
					ps468.OverlayValues[4] = d4
					ps468.OverlayValues[5] = d5
					ps468.OverlayValues[6] = d6
					ps468.OverlayValues[7] = d7
					ps468.OverlayValues[8] = d8
					ps468.OverlayValues[9] = d9
					ps468.OverlayValues[10] = d10
					ps468.OverlayValues[11] = d11
					ps468.OverlayValues[12] = d12
					ps468.OverlayValues[13] = d13
					ps468.OverlayValues[14] = d14
					ps468.OverlayValues[43] = d43
					ps468.OverlayValues[44] = d44
					ps468.OverlayValues[45] = d45
					ps468.OverlayValues[46] = d46
					ps468.OverlayValues[47] = d47
					ps468.OverlayValues[48] = d48
					ps468.OverlayValues[49] = d49
					ps468.OverlayValues[50] = d50
					ps468.OverlayValues[51] = d51
					ps468.OverlayValues[52] = d52
					ps468.OverlayValues[53] = d53
					ps468.OverlayValues[54] = d54
					ps468.OverlayValues[55] = d55
					ps468.OverlayValues[111] = d111
					ps468.OverlayValues[112] = d112
					ps468.OverlayValues[113] = d113
					ps468.OverlayValues[114] = d114
					ps468.OverlayValues[115] = d115
					ps468.OverlayValues[180] = d180
					ps468.OverlayValues[181] = d181
					ps468.OverlayValues[182] = d182
					ps468.OverlayValues[253] = d253
					ps468.OverlayValues[254] = d254
					ps468.OverlayValues[255] = d255
					ps468.OverlayValues[258] = d258
					ps468.OverlayValues[335] = d335
					ps468.OverlayValues[336] = d336
					ps468.OverlayValues[337] = d337
					ps468.OverlayValues[338] = d338
					ps468.OverlayValues[339] = d339
					ps468.OverlayValues[341] = d341
					ps468.OverlayValues[342] = d342
					ps468.OverlayValues[343] = d343
					ps468.OverlayValues[344] = d344
					ps468.OverlayValues[346] = d346
					ps468.OverlayValues[347] = d347
					ps468.OverlayValues[348] = d348
					ps468.OverlayValues[349] = d349
					ps468.OverlayValues[350] = d350
					ps468.OverlayValues[351] = d351
					ps468.OverlayValues[354] = d354
					ps468.OverlayValues[463] = d463
					ps468.OverlayValues[464] = d464
					ps468.OverlayValues[465] = d465
					ps468.OverlayValues[466] = d466
					ps468.OverlayValues[467] = d467
					ps468.PhiValues = make([]JITValueDesc, 1)
					d469 = d466
					ps468.PhiValues[0] = d469
					if ps468.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps468)
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
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK3 {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1, ID: 0}
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
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
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
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != LocNone {
						d253 = ps.OverlayValues[253]
					}
					if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != LocNone {
						d254 = ps.OverlayValues[254]
					}
					if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != LocNone {
						d255 = ps.OverlayValues[255]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
						d335 = ps.OverlayValues[335]
					}
					if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
						d336 = ps.OverlayValues[336]
					}
					if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != LocNone {
						d337 = ps.OverlayValues[337]
					}
					if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != LocNone {
						d338 = ps.OverlayValues[338]
					}
					if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
						d339 = ps.OverlayValues[339]
					}
					if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != LocNone {
						d341 = ps.OverlayValues[341]
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
					if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
						d346 = ps.OverlayValues[346]
					}
					if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != LocNone {
						d347 = ps.OverlayValues[347]
					}
					if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
						d348 = ps.OverlayValues[348]
					}
					if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
						d349 = ps.OverlayValues[349]
					}
					if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
						d350 = ps.OverlayValues[350]
					}
					if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
						d351 = ps.OverlayValues[351]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
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
					if len(ps.OverlayValues) > 469 && ps.OverlayValues[469].Loc != LocNone {
						d469 = ps.OverlayValues[469]
					}
					ctx.ReclaimUntrackedRegs()
					d470 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					d472 = ctx.EmitSliceElementAddress(&d7, &d470, 16)
					ctx.EnsureDesc(&d472)
					r25 := ctx.AllocRegExcept(d472.Reg)
					ctx.EmitMovRegMem(r25, d472.Reg, 8)
					ctx.EmitMovRegMem(d472.Reg, d472.Reg, 0)
					d471 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d472.Reg, Reg2: r25}
					ctx.BindReg(d472.Reg, &d471)
					ctx.BindReg(r25, &d471)
					var d473 JITValueDesc
					if d471.Loc == LocImm {
						d473 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d471.Imm.Int())}
					} else if d471.Type == tagInt && d471.Loc == LocRegPair {
						ctx.FreeReg(d471.Reg)
						d473 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d471.Reg2}
						ctx.BindReg(d471.Reg2, &d473)
						ctx.BindReg(d471.Reg2, &d473)
					} else if d471.Type == tagInt && d471.Loc == LocReg {
						d473 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d471.Reg}
						ctx.BindReg(d471.Reg, &d473)
						ctx.BindReg(d471.Reg, &d473)
					} else {
						d473 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d471}, 1)
						d473.Type = tagInt
						ctx.BindReg(d473.Reg, &d473)
					}
					ctx.FreeDesc(&d471)
					ctx.EnsureDesc(&d473)
					ctx.EnsureDesc(&d473)
					var d474 JITValueDesc
					if d473.Loc == LocImm {
						d474 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d473.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d473.Reg)
						ctx.EmitMovRegReg(scratch, d473.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d474 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d474)
					}
					if d474.Loc == LocReg && d473.Loc == LocReg && d474.Reg == d473.Reg {
						ctx.TransferReg(d473.Reg)
						d473.Loc = LocNone
					}
					ctx.FreeDesc(&d473)
					ctx.EnsureDesc(&d474)
					d475 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.SyncDesc(&d474)
					d476 = d7
					d476.ID = 0
					d477 = d475
					d477.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d476, &d477, &d474, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d476, &d477, &d474, int32(16))
					}
					ctx.FreeDesc(&d477)
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d53)
					d478 = d8
					_ = d478
					d479 = d53
					_ = d479
					ctx.StabilizeDescForControlFlow(&d53)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl14 := ctx.ReserveLabel()
					_ = lbl14
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d478)
					ctx.EnsureDesc(&d478)
					d478 = JITPrepareScmerGoArg(ctx, d478)
					ctx.EnsureDesc(&d479)
					ctx.EnsureDesc(&d479)
					ctx.EnsureDesc(&d479)
					if d479.Loc != LocRegTriple && d479.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d480 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d480.Loc == LocRegPair || d480.Loc == LocStackPair || d480.Loc == LocRegTriple || d480.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d478)
					ctx.SyncDesc(&d479)
					ctx.SyncDesc(&d480)
					d481 = ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d478, d479, d480}, 2)
					d481.NoHeapPointer = false
					ctx.BindReg(d481.Reg, &d481)
					ctx.BindReg(d481.Reg2, &d481)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d481)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					var d482 JITValueDesc
					if d4.Loc == LocImm {
						d482 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK2 {
							scratch = r0
						} else {
							scratch = ctx.AllocRegExcept(d4.Reg)
						}
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d482 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d482)
					}
					if d482.Loc == LocReg && d4.Loc == LocReg && d482.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					if ps.General {
						ctx.SyncDesc(&d482)
						if d482.Loc == LocReg {
							ctx.ProtectReg(d482.Reg)
						} else if d482.Loc == LocRegPair {
							ctx.ProtectReg(d482.Reg)
							ctx.ProtectReg(d482.Reg2)
						}
						d483 = d482
						if d483.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d483)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d483)
						} else {
							ctx.EmitStoreToStack(d483, int32(bbs[7].PhiBase)+int32(0))
						}
						if d482.Loc == LocReg {
							ctx.UnprotectReg(d482.Reg)
						} else if d482.Loc == LocRegPair {
							ctx.UnprotectReg(d482.Reg)
							ctx.UnprotectReg(d482.Reg2)
						}
					}
					ps484 := PhiState{General: ps.General}
					ps484.OverlayValues = make([]JITValueDesc, 484)
					ps484.OverlayValues[4] = d4
					ps484.OverlayValues[5] = d5
					ps484.OverlayValues[6] = d6
					ps484.OverlayValues[7] = d7
					ps484.OverlayValues[8] = d8
					ps484.OverlayValues[9] = d9
					ps484.OverlayValues[10] = d10
					ps484.OverlayValues[11] = d11
					ps484.OverlayValues[12] = d12
					ps484.OverlayValues[13] = d13
					ps484.OverlayValues[14] = d14
					ps484.OverlayValues[43] = d43
					ps484.OverlayValues[44] = d44
					ps484.OverlayValues[45] = d45
					ps484.OverlayValues[46] = d46
					ps484.OverlayValues[47] = d47
					ps484.OverlayValues[48] = d48
					ps484.OverlayValues[49] = d49
					ps484.OverlayValues[50] = d50
					ps484.OverlayValues[51] = d51
					ps484.OverlayValues[52] = d52
					ps484.OverlayValues[53] = d53
					ps484.OverlayValues[54] = d54
					ps484.OverlayValues[55] = d55
					ps484.OverlayValues[111] = d111
					ps484.OverlayValues[112] = d112
					ps484.OverlayValues[113] = d113
					ps484.OverlayValues[114] = d114
					ps484.OverlayValues[115] = d115
					ps484.OverlayValues[180] = d180
					ps484.OverlayValues[181] = d181
					ps484.OverlayValues[182] = d182
					ps484.OverlayValues[253] = d253
					ps484.OverlayValues[254] = d254
					ps484.OverlayValues[255] = d255
					ps484.OverlayValues[258] = d258
					ps484.OverlayValues[335] = d335
					ps484.OverlayValues[336] = d336
					ps484.OverlayValues[337] = d337
					ps484.OverlayValues[338] = d338
					ps484.OverlayValues[339] = d339
					ps484.OverlayValues[341] = d341
					ps484.OverlayValues[342] = d342
					ps484.OverlayValues[343] = d343
					ps484.OverlayValues[344] = d344
					ps484.OverlayValues[346] = d346
					ps484.OverlayValues[347] = d347
					ps484.OverlayValues[348] = d348
					ps484.OverlayValues[349] = d349
					ps484.OverlayValues[350] = d350
					ps484.OverlayValues[351] = d351
					ps484.OverlayValues[354] = d354
					ps484.OverlayValues[463] = d463
					ps484.OverlayValues[464] = d464
					ps484.OverlayValues[465] = d465
					ps484.OverlayValues[466] = d466
					ps484.OverlayValues[467] = d467
					ps484.OverlayValues[469] = d469
					ps484.OverlayValues[470] = d470
					ps484.OverlayValues[471] = d471
					ps484.OverlayValues[472] = d472
					ps484.OverlayValues[473] = d473
					ps484.OverlayValues[474] = d474
					ps484.OverlayValues[475] = d475
					ps484.OverlayValues[476] = d476
					ps484.OverlayValues[477] = d477
					ps484.OverlayValues[478] = d478
					ps484.OverlayValues[479] = d479
					ps484.OverlayValues[480] = d480
					ps484.OverlayValues[481] = d481
					ps484.OverlayValues[482] = d482
					ps484.OverlayValues[483] = d483
					ps484.PhiValues = make([]JITValueDesc, 1)
					d485 = d482
					ps484.PhiValues[0] = d485
					if ps484.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps484)
					return result
				}
				ps486 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps486)
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
