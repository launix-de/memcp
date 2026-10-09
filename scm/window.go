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

import "sort"
import "sync"
import "unsafe"
import "container/heap"

// streamTopK holds only the best OFFSET+LIMIT complete values. The producer
// may reuse a flat tuple after emit returns; copy retained tuples at admission.
// All access, including the prepared comparator, belongs to the emit mutex.
type streamTopKEntry struct {
	value    Scmer
	sequence uint64
}

type streamTopK struct {
	entries []streamTopKEntry
	compare SerialProc
	args    [2]Scmer
}

func (h *streamTopK) before(a, b streamTopKEntry) bool {
	h.args[0], h.args[1] = a.value, b.value
	if ToBool(h.compare.Call(h.args[:])) {
		return true
	}
	h.args[0], h.args[1] = b.value, a.value
	if ToBool(h.compare.Call(h.args[:])) {
		return false
	}
	return a.sequence < b.sequence
}
func (h *streamTopK) Len() int           { return len(h.entries) }
func (h *streamTopK) Less(i, j int) bool { return h.before(h.entries[j], h.entries[i]) }
func (h *streamTopK) Swap(i, j int)      { h.entries[i], h.entries[j] = h.entries[j], h.entries[i] }
func (h *streamTopK) Push(v any)         { h.entries = append(h.entries, v.(streamTopKEntry)) }
func (h *streamTopK) Pop() any {
	i := len(h.entries) - 1
	v := h.entries[i]
	h.entries[i] = streamTopKEntry{}
	h.entries = h.entries[:i]
	return v
}

func streamWindowTopK(a []Scmer, offset, limit int) Scmer {
	if limit < 0 {
		panic("stream_window_reduce: ordered producer requires a finite limit")
	}
	if offset > int(^uint(0)>>1)-limit {
		panic("stream_window_reduce: ordered window overflows")
	}
	result := a[3]
	if limit == 0 {
		return result
	}
	keep := offset + limit
	h := streamTopK{compare: PrepareSerialProc(a[5])}
	var mu sync.Mutex
	var sequence uint64
	emit := NewFunc(func(values ...Scmer) Scmer {
		mu.Lock()
		defer mu.Unlock()
		if len(values) != 1 {
			panic("stream_window_reduce: emit expects exactly one complete value")
		}
		entry := streamTopKEntry{value: values[0], sequence: sequence}
		sequence++
		if len(h.entries) >= keep && !h.before(entry, h.entries[0]) {
			return NewNil()
		}
		if entry.value.IsSlice() {
			entry.value = NewSlice(append([]Scmer(nil), entry.value.Slice()...))
		}
		if len(h.entries) < keep {
			heap.Push(&h, entry)
		} else {
			h.entries[0] = entry
			heap.Fix(&h, 0)
		}
		return NewNil()
	})
	producer := PrepareSerialProc(a[4])
	producer.Call([]Scmer{emit})
	sort.Slice(h.entries, func(i, j int) bool { return h.before(h.entries[i], h.entries[j]) })
	reducer := PrepareSerialProc(a[2])
	var args [2]Scmer
	for i := offset; i < len(h.entries); i++ {
		args[0], args[1] = result, h.entries[i].value
		result = reducer.Call(args[:])
	}
	return result
}

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
			if len(a) == 6 && !a[5].IsNil() {
				return streamWindowTopK(a, offset, limit)
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
				{Kind: "func", Label: "less", Description: "optional strict ordering for an unordered producer; retains only offset+limit values and requires a finite limit", Optional: true, Params: []*TypeDescriptor{{Kind: "any", Label: "left"}, {Kind: "any", Label: "right"}}, Return: &TypeDescriptor{Kind: "bool"}},
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
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
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
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
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
				var d89 JITValueDesc
				_ = d89
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
				var d146 JITValueDesc
				_ = d146
				var d147 JITValueDesc
				_ = d147
				var d148 JITValueDesc
				_ = d148
				var d203 JITValueDesc
				_ = d203
				var d204 JITValueDesc
				_ = d204
				var d205 JITValueDesc
				_ = d205
				var d263 JITValueDesc
				_ = d263
				var d264 JITValueDesc
				_ = d264
				var d265 JITValueDesc
				_ = d265
				var d326 JITValueDesc
				_ = d326
				var d327 JITValueDesc
				_ = d327
				var d328 JITValueDesc
				_ = d328
				var d329 JITValueDesc
				_ = d329
				var d330 JITValueDesc
				_ = d330
				var d331 JITValueDesc
				_ = d331
				var d332 JITValueDesc
				_ = d332
				var d400 JITValueDesc
				_ = d400
				var d401 JITValueDesc
				_ = d401
				var d402 JITValueDesc
				_ = d402
				var d403 JITValueDesc
				_ = d403
				var d404 JITValueDesc
				_ = d404
				var d405 JITValueDesc
				_ = d405
				var d406 JITValueDesc
				_ = d406
				var d407 JITValueDesc
				_ = d407
				var d408 JITValueDesc
				_ = d408
				var d409 JITValueDesc
				_ = d409
				var d410 JITValueDesc
				_ = d410
				var d411 JITValueDesc
				_ = d411
				var d412 JITValueDesc
				_ = d412
				var d413 JITValueDesc
				_ = d413
				var d414 JITValueDesc
				_ = d414
				var d415 JITValueDesc
				_ = d415
				var d416 JITValueDesc
				_ = d416
				var d417 JITValueDesc
				_ = d417
				var d418 JITValueDesc
				_ = d418
				var d419 JITValueDesc
				_ = d419
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [14]BBDescriptor
				bbs[7].PhiBase = int32(phiBase0) + int32(0)
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					if d3.Loc == LocRegTriple {
						ctx.BindReg(d3.Reg, &d3)
						ctx.BindReg(d3.Reg2, &d3)
						ctx.BindReg(d3.Reg3, &d3)
					}
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
					ctx.StabilizeDescForControlFlow(&d4)
					d5 = args[2]
					d5.ID = 0
					if d5.Type == tagSlice {
						d6 = jitKnownSliceHeader(ctx, &d5)
					} else {
						d6 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d5}, 3)
					}
					if d6.Loc == LocRegTriple {
						ctx.BindReg(d6.Reg, &d6)
						ctx.BindReg(d6.Reg2, &d6)
						ctx.BindReg(d6.Reg3, &d6)
					}
					ctx.StabilizeDescForControlFlow(&d6)
					ctx.FreeDesc(&d5)
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d9.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d8)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap10 := d1
						snap11 := d2
						snap12 := d3
						snap13 := d4
						snap14 := d5
						snap15 := d6
						snap16 := d7
						snap17 := d8
						snap18 := d9
						alloc19 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc19)
						d1 = snap10
						d2 = snap11
						d3 = snap12
						d4 = snap13
						d5 = snap14
						d6 = snap15
						d7 = snap16
						d8 = snap17
						d9 = snap18
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["window_mut"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d22 = ctx.EmitSliceElementAddress(&d3, &d20, 16)
					ctx.EnsureDesc(&d22)
					r1 := ctx.AllocRegExcept(d22.Reg)
					ctx.EmitMovRegMem(r1, d22.Reg, 8)
					ctx.EmitMovRegMem(d22.Reg, d22.Reg, 0)
					d21 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d22.Reg, Reg2: r1}
					ctx.BindReg(d22.Reg, &d21)
					ctx.BindReg(r1, &d21)
					if d21.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d21.Imm.Int())}
					} else if d21.Type == tagInt && d21.Loc == LocRegPair {
						ctx.FreeReg(d21.Reg)
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d21.Reg2}
						ctx.BindReg(d21.Reg2, &d23)
						ctx.BindReg(d21.Reg2, &d23)
					} else if d21.Type == tagInt && d21.Loc == LocReg {
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d21.Reg}
						ctx.BindReg(d21.Reg, &d23)
						ctx.BindReg(d21.Reg, &d23)
					} else {
						d23 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d21}, 1)
						d23.Type = tagInt
						ctx.BindReg(d23.Reg, &d23)
					}
					ctx.FreeDesc(&d21)
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d23)
					ctx.StabilizeDescForControlFlow(&d23)
					d25 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					d27 = ctx.EmitSliceElementAddress(&d3, &d25, 16)
					ctx.EnsureDesc(&d27)
					r2 := ctx.AllocRegExcept(d27.Reg)
					ctx.EmitMovRegMem(r2, d27.Reg, 8)
					ctx.EmitMovRegMem(d27.Reg, d27.Reg, 0)
					d26 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d27.Reg, Reg2: r2}
					ctx.BindReg(d27.Reg, &d26)
					ctx.BindReg(r2, &d26)
					if d26.Loc == LocImm {
						d28 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d26.Imm.Int())}
					} else if d26.Type == tagInt && d26.Loc == LocRegPair {
						ctx.FreeReg(d26.Reg)
						d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d26.Reg2}
						ctx.BindReg(d26.Reg2, &d28)
						ctx.BindReg(d26.Reg2, &d28)
					} else if d26.Type == tagInt && d26.Loc == LocReg {
						d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d26.Reg}
						ctx.BindReg(d26.Reg, &d28)
						ctx.BindReg(d26.Reg, &d28)
					} else {
						d28 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d26}, 1)
						d28.Type = tagInt
						ctx.BindReg(d28.Reg, &d28)
					}
					ctx.FreeDesc(&d26)
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d28)
					ctx.StabilizeDescForControlFlow(&d28)
					d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					d32 = ctx.EmitSliceElementAddress(&d3, &d30, 16)
					ctx.EnsureDesc(&d32)
					r3 := ctx.AllocRegExcept(d32.Reg)
					ctx.EmitMovRegMem(r3, d32.Reg, 8)
					ctx.EmitMovRegMem(d32.Reg, d32.Reg, 0)
					d31 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d32.Reg, Reg2: r3}
					ctx.BindReg(d32.Reg, &d31)
					ctx.BindReg(r3, &d31)
					if d31.Loc == LocImm {
						d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d31.Imm.Int())}
					} else if d31.Type == tagInt && d31.Loc == LocRegPair {
						ctx.FreeReg(d31.Reg)
						d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg2}
						ctx.BindReg(d31.Reg2, &d33)
						ctx.BindReg(d31.Reg2, &d33)
					} else if d31.Type == tagInt && d31.Loc == LocReg {
						d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg}
						ctx.BindReg(d31.Reg, &d33)
						ctx.BindReg(d31.Reg, &d33)
					} else {
						d33 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d31}, 1)
						d33.Type = tagInt
						ctx.BindReg(d33.Reg, &d33)
					}
					ctx.FreeDesc(&d31)
					ctx.EnsureDesc(&d33)
					ctx.EnsureDesc(&d33)
					ctx.StabilizeDescForControlFlow(&d33)
					d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(3)}
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2}
						ctx.BindReg(d3.Reg2, &d36)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d35)
					ctx.EnsureDesc(&d36)
					if d36.Loc == LocImm && d35.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d36.Imm.Int() - d35.Imm.Int())}
					} else {
						r4 := ctx.AllocReg()
						if d36.Loc == LocImm {
							ctx.EmitMovRegImm64(r4, uint64(d36.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r4, d36.Reg)
						}
						if d35.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d35.Imm.Int()))
							ctx.EmitSubInt64(r4, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r4, d35.Reg)
						}
						d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d38)
					}
					r5 := ctx.EmitSliceDataAfterLow(&d3, &d35, 16)
					d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
					ctx.BindReg(r5, &d39)
					ctx.BindReg(r5, &d39)
					var r6 Reg
					var r7 Reg
					ctx.SyncDesc(&d39)
					ctx.EnsureDesc(&d39)
					if d39.Loc == LocImm {
						r6 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, uint64(d39.Imm.Int()))
					} else {
						r6 = d39.Reg
					}
					ctx.ProtectReg(r6)
					ctx.SyncDesc(&d38)
					ctx.EnsureDesc(&d38)
					if d38.Loc == LocImm {
						r7 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, uint64(d38.Imm.Int()))
					} else {
						r7 = d38.Reg
					}
					ctx.ProtectReg(r7)
					r8 := ctx.EmitSliceCapAfterLow(&d3, &d35, r6, r7)
					ctx.UnprotectReg(r7)
					ctx.UnprotectReg(r6)
					d40 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8}
					ctx.BindReg(r6, &d40)
					ctx.BindReg(r7, &d40)
					ctx.BindReg(r8, &d40)
					ctx.BindReg(r6, &d40)
					ctx.BindReg(r7, &d40)
					ctx.BindReg(r8, &d40)
					ctx.StabilizeDescForControlFlow(&d40)
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						d41 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d33.Imm.Int() <= 0)}
					} else {
						r9 := ctx.AllocRegExcept(d33.Reg)
						ctx.EmitCmpRegImm32(d33.Reg, 0)
						d41 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondSignedLessOrEqual}
						ctx.BindReg(r9, &d41)
					}
					d42 = d41
					ctx.EnsureDesc(&d42)
					if d42.Loc != LocImm && d42.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d42.Loc == LocImm {
						if d42.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitJump(d42.Condition, lbl4)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d41)
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap43 := d1
						snap44 := d2
						snap45 := d3
						snap46 := d4
						snap47 := d5
						snap48 := d6
						snap49 := d7
						snap50 := d8
						snap51 := d9
						snap52 := d20
						snap53 := d21
						snap54 := d22
						snap55 := d23
						snap56 := d24
						snap57 := d25
						snap58 := d26
						snap59 := d27
						snap60 := d28
						snap61 := d29
						snap62 := d30
						snap63 := d31
						snap64 := d32
						snap65 := d33
						snap66 := d34
						snap67 := d35
						snap68 := d36
						snap69 := d37
						snap70 := d38
						snap71 := d39
						snap72 := d40
						snap73 := d41
						snap74 := d42
						alloc75 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc75)
						d1 = snap43
						d2 = snap44
						d3 = snap45
						d4 = snap46
						d5 = snap47
						d6 = snap48
						d7 = snap49
						d8 = snap50
						d9 = snap51
						d20 = snap52
						d21 = snap53
						d22 = snap54
						d23 = snap55
						d24 = snap56
						d25 = snap57
						d26 = snap58
						d27 = snap59
						d28 = snap60
						d29 = snap61
						d30 = snap62
						d31 = snap63
						d32 = snap64
						d33 = snap65
						d34 = snap66
						d35 = snap67
						d36 = snap68
						d37 = snap69
						d38 = snap70
						d39 = snap71
						d40 = snap72
						d41 = snap73
						d42 = snap74
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["window_mut"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d33)
					ctx.EnsureDesc(&d40)
					if d40.Loc == LocRegPair || d40.Loc == LocRegTriple {
						d76 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d40.Reg2}
						ctx.BindReg(d40.Reg2, &d76)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d33)
					ctx.EnsureDesc(&d76)
					if d76.Loc == LocImm && d33.Loc == LocImm {
						d78 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d76.Imm.Int() - d33.Imm.Int())}
					} else {
						r10 := ctx.AllocReg()
						if d76.Loc == LocImm {
							ctx.EmitMovRegImm64(r10, uint64(d76.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r10, d76.Reg)
						}
						if d33.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d33.Imm.Int()))
							ctx.EmitSubInt64(r10, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r10, d33.Reg)
						}
						d78 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d78)
					}
					r11 := ctx.EmitSliceDataAfterLow(&d40, &d33, 16)
					d79 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
					ctx.BindReg(r11, &d79)
					ctx.BindReg(r11, &d79)
					var r12 Reg
					var r13 Reg
					ctx.SyncDesc(&d79)
					ctx.EnsureDesc(&d79)
					if d79.Loc == LocImm {
						r12 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r12, uint64(d79.Imm.Int()))
					} else {
						r12 = d79.Reg
					}
					ctx.ProtectReg(r12)
					ctx.SyncDesc(&d78)
					ctx.EnsureDesc(&d78)
					if d78.Loc == LocImm {
						r13 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r13, uint64(d78.Imm.Int()))
					} else {
						r13 = d78.Reg
					}
					ctx.ProtectReg(r13)
					r14 := ctx.EmitSliceCapAfterLow(&d40, &d33, r12, r13)
					ctx.UnprotectReg(r13)
					ctx.UnprotectReg(r12)
					d80 = JITValueDesc{Loc: LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14}
					ctx.BindReg(r12, &d80)
					ctx.BindReg(r13, &d80)
					ctx.BindReg(r14, &d80)
					ctx.BindReg(r12, &d80)
					ctx.BindReg(r13, &d80)
					ctx.BindReg(r14, &d80)
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d80)
					callResults81 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d40, d80}, []uint8{1}, []uint8{0})
					d82 = callResults81[0]
					d82.Type = tagInt
					if d40.SliceSizeKnown {
						d83 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d40.KnownSliceLen))}
					} else if d40.Loc == LocImm {
						d83 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d40.StackOff))}
					} else if d40.Loc == LocStackTriple {
						d83 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d40.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d40)
						if d40.Loc == LocRegPair || d40.Loc == LocRegTriple {
							d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d40.Reg2, ID: 0}
						} else if d40.Loc == LocReg {
							d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d40.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d83)
					ctx.EnsureDesc(&d33)
					ctx.SyncDesc(&d83)
					ctx.SyncDesc(&d33)
					if d83.Loc == LocImm && d33.Loc == LocImm {
						d84 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d83.Imm.Int() - d33.Imm.Int())}
					} else if d33.Loc == LocImm && d33.Imm.Int() == 0 {
						ctx.EnsureDesc(&d83)
						r15 := ctx.AllocRegExcept(d83.Reg)
						ctx.EmitMovRegReg(r15, d83.Reg)
						d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
						ctx.BindReg(r15, &d84)
					} else if d83.Loc == LocImm {
						ctx.EnsureDesc(&d33)
						scratch := ctx.AllocRegExcept(d33.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d83.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d33)
						d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d84)
					} else if d33.Loc == LocImm {
						ctx.EnsureDesc(&d83)
						scratch := ctx.AllocRegExcept(d83.Reg)
						ctx.EmitMovRegReg(scratch, d83.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d33.Imm.Int())
						d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d84)
					} else {
						ctx.EnsureDesc(&d83)
						ctx.SyncDesc(&d33)
						r16 := ctx.AllocRegExceptOperand(&d33, d83.Reg)
						ctx.EmitMovRegReg(r16, d83.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r16, &d33)
						d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
						ctx.BindReg(r16, &d84)
					}
					if d84.Loc == LocReg && d83.Loc == LocReg && d84.Reg == d83.Reg {
						ctx.TransferReg(d83.Reg)
						d83.Loc = LocNone
					}
					ctx.FreeDesc(&d83)
					ctx.EnsureDesc(&d84)
					ctx.EnsureDesc(&d40)
					if d40.Loc == LocRegPair || d40.Loc == LocRegTriple {
						d85 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d40.Reg2}
						ctx.BindReg(d40.Reg2, &d85)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d84)
					ctx.EnsureDesc(&d85)
					if d85.Loc == LocImm && d84.Loc == LocImm {
						d87 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d85.Imm.Int() - d84.Imm.Int())}
					} else {
						r17 := ctx.AllocReg()
						if d85.Loc == LocImm {
							ctx.EmitMovRegImm64(r17, uint64(d85.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r17, d85.Reg)
						}
						if d84.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d84.Imm.Int()))
							ctx.EmitSubInt64(r17, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r17, d84.Reg)
						}
						d87 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
						ctx.BindReg(r17, &d87)
					}
					r18 := ctx.EmitSliceDataAfterLow(&d40, &d84, 16)
					d88 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
					ctx.BindReg(r18, &d88)
					ctx.BindReg(r18, &d88)
					var r19 Reg
					var r20 Reg
					ctx.SyncDesc(&d88)
					ctx.EnsureDesc(&d88)
					if d88.Loc == LocImm {
						r19 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r19, uint64(d88.Imm.Int()))
					} else {
						r19 = d88.Reg
					}
					ctx.ProtectReg(r19)
					ctx.SyncDesc(&d87)
					ctx.EnsureDesc(&d87)
					if d87.Loc == LocImm {
						r20 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r20, uint64(d87.Imm.Int()))
					} else {
						r20 = d87.Reg
					}
					ctx.ProtectReg(r20)
					r21 := ctx.EmitSliceCapAfterLow(&d40, &d84, r19, r20)
					ctx.UnprotectReg(r20)
					ctx.UnprotectReg(r19)
					d89 = JITValueDesc{Loc: LocRegTriple, Reg: r19, Reg2: r20, Reg3: r21}
					ctx.BindReg(r19, &d89)
					ctx.BindReg(r20, &d89)
					ctx.BindReg(r21, &d89)
					ctx.BindReg(r19, &d89)
					ctx.BindReg(r20, &d89)
					ctx.BindReg(r21, &d89)
					ctx.StabilizeDescForControlFlow(&d89)
					ctx.FreeDesc(&d84)
					if d89.SliceSizeKnown {
						d90 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d89.KnownSliceLen))}
					} else if d89.Loc == LocImm {
						d90 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d89.StackOff))}
					} else if d89.Loc == LocStackTriple {
						d90 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d89.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d89)
						if d89.Loc == LocRegPair || d89.Loc == LocRegTriple {
							d90 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d89.Reg2, ID: 0}
						} else if d89.Loc == LocReg {
							d90 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d89.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d90)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[7].PhiBase)+int32(0))
					return bbs[7].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					if d40.SliceSizeKnown {
						d91 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d40.KnownSliceLen))}
					} else if d40.Loc == LocImm {
						d91 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d40.StackOff))}
					} else if d40.Loc == LocStackTriple {
						d91 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d40.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d40)
						if d40.Loc == LocRegPair || d40.Loc == LocRegTriple {
							d91 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d40.Reg2, ID: 0}
						} else if d40.Loc == LocReg {
							d91 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d40.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d91)
					ctx.EnsureDesc(&d33)
					ctx.EnsureDescsTogether(&d91, &d33)
					if d91.Loc == LocImm && d33.Loc == LocImm {
						d92 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d91.Imm.Int() % d33.Imm.Int())}
					} else {
						d92 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntRem), []JITValueDesc{d91, d33}, 1)
					}
					if d92.Loc == LocReg && d91.Loc == LocReg && d92.Reg == d91.Reg {
						ctx.TransferReg(d91.Reg)
						d91.Loc = LocNone
					}
					ctx.FreeDesc(&d91)
					ctx.FreeDesc(&d33)
					ctx.EnsureDesc(&d92)
					if d92.Loc == LocImm {
						d93 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d92.Imm.Int() != 0)}
					} else {
						r22 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d92.Reg, 0)
						d93 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondNotEqual}
						ctx.BindReg(r22, &d93)
					}
					ctx.FreeDesc(&d92)
					d94 = d93
					ctx.EnsureDesc(&d94)
					if d94.Loc != LocImm && d94.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d94.Loc == LocImm {
						if d94.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitJump(d94.Condition, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d93)
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap95 := d1
						snap96 := d2
						snap97 := d3
						snap98 := d4
						snap99 := d5
						snap100 := d6
						snap101 := d7
						snap102 := d8
						snap103 := d9
						snap104 := d20
						snap105 := d21
						snap106 := d22
						snap107 := d23
						snap108 := d24
						snap109 := d25
						snap110 := d26
						snap111 := d27
						snap112 := d28
						snap113 := d29
						snap114 := d30
						snap115 := d31
						snap116 := d32
						snap117 := d33
						snap118 := d34
						snap119 := d35
						snap120 := d36
						snap121 := d37
						snap122 := d38
						snap123 := d39
						snap124 := d40
						snap125 := d41
						snap126 := d42
						snap127 := d76
						snap128 := d77
						snap129 := d78
						snap130 := d79
						snap131 := d80
						snap132 := d82
						snap133 := d83
						snap134 := d84
						snap135 := d85
						snap136 := d86
						snap137 := d87
						snap138 := d88
						snap139 := d89
						snap140 := d90
						snap141 := d91
						snap142 := d92
						snap143 := d93
						snap144 := d94
						alloc145 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc145)
						d1 = snap95
						d2 = snap96
						d3 = snap97
						d4 = snap98
						d5 = snap99
						d6 = snap100
						d7 = snap101
						d8 = snap102
						d9 = snap103
						d20 = snap104
						d21 = snap105
						d22 = snap106
						d23 = snap107
						d24 = snap108
						d25 = snap109
						d26 = snap110
						d27 = snap111
						d28 = snap112
						d29 = snap113
						d30 = snap114
						d31 = snap115
						d32 = snap116
						d33 = snap117
						d34 = snap118
						d35 = snap119
						d36 = snap120
						d37 = snap121
						d38 = snap122
						d39 = snap123
						d40 = snap124
						d41 = snap125
						d42 = snap126
						d76 = snap127
						d77 = snap128
						d78 = snap129
						d79 = snap130
						d80 = snap131
						d82 = snap132
						d83 = snap133
						d84 = snap134
						d85 = snap135
						d86 = snap136
						d87 = snap137
						d88 = snap138
						d89 = snap139
						d90 = snap140
						d91 = snap141
						d92 = snap142
						d93 = snap143
						d94 = snap144
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					if d40.SliceSizeKnown {
						d146 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d40.KnownSliceLen))}
					} else if d40.Loc == LocImm {
						d146 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d40.StackOff))}
					} else if d40.Loc == LocStackTriple {
						d146 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d40.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d40)
						if d40.Loc == LocRegPair || d40.Loc == LocRegTriple {
							d146 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d40.Reg2, ID: 0}
						} else if d40.Loc == LocReg {
							d146 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d40.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d146)
					if d146.Loc == LocImm {
						d147 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d146.Imm.Int() == 0)}
					} else {
						r23 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d146.Reg, 0)
						d147 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r23, Condition: CondEqual}
						ctx.BindReg(r23, &d147)
					}
					ctx.FreeDesc(&d146)
					d148 = d147
					ctx.EnsureDesc(&d148)
					if d148.Loc != LocImm && d148.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d148.Loc == LocImm {
						if d148.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d148.Condition, lbl4)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d147)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap149 := d1
						snap150 := d2
						snap151 := d3
						snap152 := d4
						snap153 := d5
						snap154 := d6
						snap155 := d7
						snap156 := d8
						snap157 := d9
						snap158 := d20
						snap159 := d21
						snap160 := d22
						snap161 := d23
						snap162 := d24
						snap163 := d25
						snap164 := d26
						snap165 := d27
						snap166 := d28
						snap167 := d29
						snap168 := d30
						snap169 := d31
						snap170 := d32
						snap171 := d33
						snap172 := d34
						snap173 := d35
						snap174 := d36
						snap175 := d37
						snap176 := d38
						snap177 := d39
						snap178 := d40
						snap179 := d41
						snap180 := d42
						snap181 := d76
						snap182 := d77
						snap183 := d78
						snap184 := d79
						snap185 := d80
						snap186 := d82
						snap187 := d83
						snap188 := d84
						snap189 := d85
						snap190 := d86
						snap191 := d87
						snap192 := d88
						snap193 := d89
						snap194 := d90
						snap195 := d91
						snap196 := d92
						snap197 := d93
						snap198 := d94
						snap199 := d146
						snap200 := d147
						snap201 := d148
						alloc202 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc202)
						d1 = snap149
						d2 = snap150
						d3 = snap151
						d4 = snap152
						d5 = snap153
						d6 = snap154
						d7 = snap155
						d8 = snap156
						d9 = snap157
						d20 = snap158
						d21 = snap159
						d22 = snap160
						d23 = snap161
						d24 = snap162
						d25 = snap163
						d26 = snap164
						d27 = snap165
						d28 = snap166
						d29 = snap167
						d30 = snap168
						d31 = snap169
						d32 = snap170
						d33 = snap171
						d34 = snap172
						d35 = snap173
						d36 = snap174
						d37 = snap175
						d38 = snap176
						d39 = snap177
						d40 = snap178
						d41 = snap179
						d42 = snap180
						d76 = snap181
						d77 = snap182
						d78 = snap183
						d79 = snap184
						d80 = snap185
						d82 = snap186
						d83 = snap187
						d84 = snap188
						d85 = snap189
						d86 = snap190
						d87 = snap191
						d88 = snap192
						d89 = snap193
						d90 = snap194
						d91 = snap195
						d92 = snap196
						d93 = snap197
						d94 = snap198
						d146 = snap199
						d147 = snap200
						d148 = snap201
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d203 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d203 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d203)
					}
					if d203.Loc == LocReg && d1.Loc == LocReg && d203.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d203)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d90)
					ctx.EnsureDescsTogether(&d203, &d90)
					if d203.Loc == LocImm && d90.Loc == LocImm {
						d204 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d203.Imm.Int() < d90.Imm.Int())}
					} else if d90.Loc == LocImm {
						r24 := ctx.AllocRegExcept(d203.Reg)
						if d90.Imm.Int() >= -2147483648 && d90.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d203.Reg, int32(d90.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d90.Imm.Int()))
							ctx.EmitCmpInt64(d203.Reg, ctx.ScratchReg)
						}
						d204 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r24, Condition: CondSignedLess}
						ctx.BindReg(r24, &d204)
					} else if d203.Loc == LocImm {
						r25 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d203.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d90.Reg)
						d204 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r25, Condition: CondSignedLess}
						ctx.BindReg(r25, &d204)
					} else {
						r26 := ctx.AllocRegExcept(d203.Reg)
						ctx.EmitCmpInt64(d203.Reg, d90.Reg)
						d204 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondSignedLess}
						ctx.BindReg(r26, &d204)
					}
					d205 = d204
					ctx.EnsureDesc(&d205)
					if d205.Loc != LocImm && d205.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d205.Loc == LocImm {
						if d205.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[9].Render()
					}
					ctx.EmitJump(d205.Condition, lbl9)
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
					}
					ctx.FreeDesc(&d204)
					ctx.FlushRegisterMoves()
					if !bbs[9].Rendered {
						snap206 := d1
						snap207 := d2
						snap208 := d3
						snap209 := d4
						snap210 := d5
						snap211 := d6
						snap212 := d7
						snap213 := d8
						snap214 := d9
						snap215 := d20
						snap216 := d21
						snap217 := d22
						snap218 := d23
						snap219 := d24
						snap220 := d25
						snap221 := d26
						snap222 := d27
						snap223 := d28
						snap224 := d29
						snap225 := d30
						snap226 := d31
						snap227 := d32
						snap228 := d33
						snap229 := d34
						snap230 := d35
						snap231 := d36
						snap232 := d37
						snap233 := d38
						snap234 := d39
						snap235 := d40
						snap236 := d41
						snap237 := d42
						snap238 := d76
						snap239 := d77
						snap240 := d78
						snap241 := d79
						snap242 := d80
						snap243 := d82
						snap244 := d83
						snap245 := d84
						snap246 := d85
						snap247 := d86
						snap248 := d87
						snap249 := d88
						snap250 := d89
						snap251 := d90
						snap252 := d91
						snap253 := d92
						snap254 := d93
						snap255 := d94
						snap256 := d146
						snap257 := d147
						snap258 := d148
						snap259 := d203
						snap260 := d204
						snap261 := d205
						alloc262 := ctx.SnapshotAllocState()
						bbs[9].Render()
						ctx.RestoreAllocState(alloc262)
						d1 = snap206
						d2 = snap207
						d3 = snap208
						d4 = snap209
						d5 = snap210
						d6 = snap211
						d7 = snap212
						d8 = snap213
						d9 = snap214
						d20 = snap215
						d21 = snap216
						d22 = snap217
						d23 = snap218
						d24 = snap219
						d25 = snap220
						d26 = snap221
						d27 = snap222
						d28 = snap223
						d29 = snap224
						d30 = snap225
						d31 = snap226
						d32 = snap227
						d33 = snap228
						d34 = snap229
						d35 = snap230
						d36 = snap231
						d37 = snap232
						d38 = snap233
						d39 = snap234
						d40 = snap235
						d41 = snap236
						d42 = snap237
						d76 = snap238
						d77 = snap239
						d78 = snap240
						d79 = snap241
						d80 = snap242
						d82 = snap243
						d83 = snap244
						d84 = snap245
						d85 = snap246
						d86 = snap247
						d87 = snap248
						d88 = snap249
						d89 = snap250
						d90 = snap251
						d91 = snap252
						d92 = snap253
						d93 = snap254
						d94 = snap255
						d146 = snap256
						d147 = snap257
						d148 = snap258
						d203 = snap259
						d204 = snap260
						d205 = snap261
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					if d6.SliceSizeKnown {
						d263 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d6.KnownSliceLen))}
					} else if d6.Loc == LocImm {
						d263 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d6.StackOff))}
					} else if d6.Loc == LocStackTriple {
						d263 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d6.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d6)
						if d6.Loc == LocRegPair || d6.Loc == LocRegTriple {
							d263 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg2, ID: 0}
						} else if d6.Loc == LocReg {
							d263 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d203)
					ctx.EnsureDesc(&d263)
					ctx.EnsureDescsTogether(&d203, &d263)
					if d203.Loc == LocImm && d263.Loc == LocImm {
						d264 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d203.Imm.Int() < d263.Imm.Int())}
					} else if d263.Loc == LocImm {
						r27 := ctx.AllocRegExcept(d203.Reg)
						if d263.Imm.Int() >= -2147483648 && d263.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d203.Reg, int32(d263.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d263.Imm.Int()))
							ctx.EmitCmpInt64(d203.Reg, ctx.ScratchReg)
						}
						d264 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r27, Condition: CondSignedLess}
						ctx.BindReg(r27, &d264)
					} else if d203.Loc == LocImm {
						r28 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d203.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d263.Reg)
						d264 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r28, Condition: CondSignedLess}
						ctx.BindReg(r28, &d264)
					} else {
						r29 := ctx.AllocRegExcept(d203.Reg)
						ctx.EmitCmpInt64(d203.Reg, d263.Reg)
						d264 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r29, Condition: CondSignedLess}
						ctx.BindReg(r29, &d264)
					}
					ctx.FreeDesc(&d263)
					d265 = d264
					ctx.EnsureDesc(&d265)
					if d265.Loc != LocImm && d265.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d265.Loc == LocImm {
						if d265.Imm.Bool() {
							return bbs[10].Render()
						}
						return bbs[11].Render()
					}
					ctx.EmitJump(d265.Condition, lbl11)
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
					}
					ctx.FreeDesc(&d264)
					ctx.FlushRegisterMoves()
					if !bbs[11].Rendered {
						snap266 := d1
						snap267 := d2
						snap268 := d3
						snap269 := d4
						snap270 := d5
						snap271 := d6
						snap272 := d7
						snap273 := d8
						snap274 := d9
						snap275 := d20
						snap276 := d21
						snap277 := d22
						snap278 := d23
						snap279 := d24
						snap280 := d25
						snap281 := d26
						snap282 := d27
						snap283 := d28
						snap284 := d29
						snap285 := d30
						snap286 := d31
						snap287 := d32
						snap288 := d33
						snap289 := d34
						snap290 := d35
						snap291 := d36
						snap292 := d37
						snap293 := d38
						snap294 := d39
						snap295 := d40
						snap296 := d41
						snap297 := d42
						snap298 := d76
						snap299 := d77
						snap300 := d78
						snap301 := d79
						snap302 := d80
						snap303 := d82
						snap304 := d83
						snap305 := d84
						snap306 := d85
						snap307 := d86
						snap308 := d87
						snap309 := d88
						snap310 := d89
						snap311 := d90
						snap312 := d91
						snap313 := d92
						snap314 := d93
						snap315 := d94
						snap316 := d146
						snap317 := d147
						snap318 := d148
						snap319 := d203
						snap320 := d204
						snap321 := d205
						snap322 := d263
						snap323 := d264
						snap324 := d265
						alloc325 := ctx.SnapshotAllocState()
						bbs[11].Render()
						ctx.RestoreAllocState(alloc325)
						d1 = snap266
						d2 = snap267
						d3 = snap268
						d4 = snap269
						d5 = snap270
						d6 = snap271
						d7 = snap272
						d8 = snap273
						d9 = snap274
						d20 = snap275
						d21 = snap276
						d22 = snap277
						d23 = snap278
						d24 = snap279
						d25 = snap280
						d26 = snap281
						d27 = snap282
						d28 = snap283
						d29 = snap284
						d30 = snap285
						d31 = snap286
						d32 = snap287
						d33 = snap288
						d34 = snap289
						d35 = snap290
						d36 = snap291
						d37 = snap292
						d38 = snap293
						d39 = snap294
						d40 = snap295
						d41 = snap296
						d42 = snap297
						d76 = snap298
						d77 = snap299
						d78 = snap300
						d79 = snap301
						d80 = snap302
						d82 = snap303
						d83 = snap304
						d84 = snap305
						d85 = snap306
						d86 = snap307
						d87 = snap308
						d88 = snap309
						d89 = snap310
						d90 = snap311
						d91 = snap312
						d92 = snap313
						d93 = snap314
						d94 = snap315
						d146 = snap316
						d147 = snap317
						d148 = snap318
						d203 = snap319
						d204 = snap320
						d205 = snap321
						d263 = snap322
						d264 = snap323
						d265 = snap324
					}
					if !bbs[10].Rendered {
						return bbs[10].Render()
					}
					return result
					return result
				}
				bbs[9].Render = func() JITValueDesc {
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					bbs[9].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d28)
					if d28.Loc == LocImm {
						d326 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d28.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d28.Reg)
						ctx.EmitMovRegReg(scratch, d28.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d326 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d326)
					}
					if d326.Loc == LocReg && d28.Loc == LocReg && d326.Reg == d28.Reg {
						ctx.TransferReg(d28.Reg)
						d28.Loc = LocNone
					}
					ctx.FreeDesc(&d28)
					ctx.EnsureDesc(&d326)
					ctx.EnsureDesc(&d326)
					ctx.EnsureDesc(&d326)
					d328 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.SyncDesc(&d326)
					d329 = d3
					d329.ID = 0
					d330 = d328
					d330.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d329, &d330, &d326, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d329, &d330, &d326, int32(16))
					}
					ctx.FreeDesc(&d330)
					ctx.EnsureDesc(&d23)
					if d23.Loc == LocImm {
						d331 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() > 0)}
					} else {
						r30 := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitCmpRegImm32(d23.Reg, 0)
						d331 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r30, Condition: CondSignedGreater}
						ctx.BindReg(r30, &d331)
					}
					d332 = d331
					ctx.EnsureDesc(&d332)
					if d332.Loc != LocImm && d332.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d332.Loc == LocImm {
						if d332.Imm.Bool() {
							return bbs[12].Render()
						}
						return bbs[13].Render()
					}
					ctx.EmitJump(d332.Condition, lbl13)
					if bbs[13].Rendered {
						ctx.EmitJmp(lbl14)
					}
					ctx.FreeDesc(&d331)
					ctx.FlushRegisterMoves()
					if !bbs[13].Rendered {
						snap333 := d1
						snap334 := d2
						snap335 := d3
						snap336 := d4
						snap337 := d5
						snap338 := d6
						snap339 := d7
						snap340 := d8
						snap341 := d9
						snap342 := d20
						snap343 := d21
						snap344 := d22
						snap345 := d23
						snap346 := d24
						snap347 := d25
						snap348 := d26
						snap349 := d27
						snap350 := d28
						snap351 := d29
						snap352 := d30
						snap353 := d31
						snap354 := d32
						snap355 := d33
						snap356 := d34
						snap357 := d35
						snap358 := d36
						snap359 := d37
						snap360 := d38
						snap361 := d39
						snap362 := d40
						snap363 := d41
						snap364 := d42
						snap365 := d76
						snap366 := d77
						snap367 := d78
						snap368 := d79
						snap369 := d80
						snap370 := d82
						snap371 := d83
						snap372 := d84
						snap373 := d85
						snap374 := d86
						snap375 := d87
						snap376 := d88
						snap377 := d89
						snap378 := d90
						snap379 := d91
						snap380 := d92
						snap381 := d93
						snap382 := d94
						snap383 := d146
						snap384 := d147
						snap385 := d148
						snap386 := d203
						snap387 := d204
						snap388 := d205
						snap389 := d263
						snap390 := d264
						snap391 := d265
						snap392 := d326
						snap393 := d327
						snap394 := d328
						snap395 := d329
						snap396 := d330
						snap397 := d331
						snap398 := d332
						alloc399 := ctx.SnapshotAllocState()
						bbs[13].Render()
						ctx.RestoreAllocState(alloc399)
						d1 = snap333
						d2 = snap334
						d3 = snap335
						d4 = snap336
						d5 = snap337
						d6 = snap338
						d7 = snap339
						d8 = snap340
						d9 = snap341
						d20 = snap342
						d21 = snap343
						d22 = snap344
						d23 = snap345
						d24 = snap346
						d25 = snap347
						d26 = snap348
						d27 = snap349
						d28 = snap350
						d29 = snap351
						d30 = snap352
						d31 = snap353
						d32 = snap354
						d33 = snap355
						d34 = snap356
						d35 = snap357
						d36 = snap358
						d37 = snap359
						d38 = snap360
						d39 = snap361
						d40 = snap362
						d41 = snap363
						d42 = snap364
						d76 = snap365
						d77 = snap366
						d78 = snap367
						d79 = snap368
						d80 = snap369
						d82 = snap370
						d83 = snap371
						d84 = snap372
						d85 = snap373
						d86 = snap374
						d87 = snap375
						d88 = snap376
						d89 = snap377
						d90 = snap378
						d91 = snap379
						d92 = snap380
						d93 = snap381
						d94 = snap382
						d146 = snap383
						d147 = snap384
						d148 = snap385
						d203 = snap386
						d204 = snap387
						d205 = snap388
						d263 = snap389
						d264 = snap390
						d265 = snap391
						d326 = snap392
						d327 = snap393
						d328 = snap394
						d329 = snap395
						d330 = snap396
						d331 = snap397
						d332 = snap398
					}
					if !bbs[12].Rendered {
						return bbs[12].Render()
					}
					return result
					return result
				}
				bbs[10].Render = func() JITValueDesc {
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					bbs[10].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d203)
					d401 = ctx.EmitSliceElementAddress(&d6, &d203, 16)
					ctx.EnsureDesc(&d401)
					r31 := ctx.AllocRegExcept(d401.Reg)
					ctx.EmitMovRegMem(r31, d401.Reg, 8)
					ctx.EmitMovRegMem(d401.Reg, d401.Reg, 0)
					d400 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d401.Reg, Reg2: r31}
					ctx.BindReg(d401.Reg, &d400)
					ctx.BindReg(r31, &d400)
					ctx.EnsureDesc(&d203)
					ctx.SyncDesc(&d400)
					d402 = d89
					d402.ID = 0
					d403 = d203
					d403.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d402, &d403, &d400, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d203)
						d403 = d203
						d403.ID = 0
						ctx.EmitStoreScmerSliceElement(&d402, &d403, &d400, int32(16))
					}
					ctx.FreeDesc(&d403)
					ctx.FreeDesc(&d400)
					ctx.SyncDesc(&d203)
					if d203.Loc == LocReg || d203.Loc == LocFPReg {
						ctx.ProtectReg(d203.Reg)
					} else if d203.Loc == LocRegPair {
						ctx.ProtectReg(d203.Reg)
						ctx.ProtectReg(d203.Reg2)
					}
					d404 = d203
					if d404.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d404)
					ctx.EmitStoreToStack(d404, int32(bbs[7].PhiBase)+int32(0))
					if d203.Loc == LocReg || d203.Loc == LocFPReg {
						ctx.UnprotectReg(d203.Reg)
					} else if d203.Loc == LocRegPair {
						ctx.UnprotectReg(d203.Reg)
						ctx.UnprotectReg(d203.Reg2)
					}
					return bbs[7].Render()
					return result
				}
				bbs[11].Render = func() JITValueDesc {
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
						return result
					}
					bbs[11].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d405 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d203)
					ctx.SyncDesc(&d405)
					d406 = d89
					d406.ID = 0
					d407 = d203
					d407.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d406, &d407, &d405, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d203)
						d407 = d203
						d407.ID = 0
						ctx.EmitStoreScmerSliceElement(&d406, &d407, &d405, int32(16))
					}
					ctx.FreeDesc(&d407)
					ctx.FreeDesc(&d405)
					ctx.SyncDesc(&d203)
					if d203.Loc == LocReg || d203.Loc == LocFPReg {
						ctx.ProtectReg(d203.Reg)
					} else if d203.Loc == LocRegPair {
						ctx.ProtectReg(d203.Reg)
						ctx.ProtectReg(d203.Reg2)
					}
					d408 = d203
					if d408.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d408)
					ctx.EmitStoreToStack(d408, int32(bbs[7].PhiBase)+int32(0))
					if d203.Loc == LocReg || d203.Loc == LocFPReg {
						ctx.UnprotectReg(d203.Reg)
					} else if d203.Loc == LocRegPair {
						ctx.UnprotectReg(d203.Reg)
						ctx.UnprotectReg(d203.Reg2)
					}
					return bbs[7].Render()
					return result
				}
				bbs[12].Render = func() JITValueDesc {
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
						return result
					}
					bbs[12].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d23)
					if d23.Loc == LocImm {
						d409 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d23.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitMovRegReg(scratch, d23.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
						d409 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d409)
					}
					if d409.Loc == LocReg && d23.Loc == LocReg && d409.Reg == d23.Reg {
						ctx.TransferReg(d23.Reg)
						d23.Loc = LocNone
					}
					ctx.FreeDesc(&d23)
					ctx.EnsureDesc(&d409)
					ctx.EnsureDesc(&d409)
					ctx.EnsureDesc(&d409)
					d411 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.SyncDesc(&d409)
					d412 = d3
					d412.ID = 0
					d413 = d411
					d413.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d412, &d413, &d409, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d412, &d413, &d409, int32(16))
					}
					ctx.FreeDesc(&d413)
					d414 = args[0]
					d414.ID = 0
					ctx.SyncDesc(&d414)
					if d414.Loc == LocRegPair || d414.Loc == LocStackPair || d414.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d414, &result)
						result.Type = d414.Type
					} else {
						switch d414.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d414)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d414)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d414)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d414, &result)
							result.Type = d414.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[13].Render = func() JITValueDesc {
					if bbs[13].Rendered {
						ctx.EmitJmp(lbl14)
						return result
					}
					bbs[13].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d40)
					d415 = d4
					_ = d415
					d416 = d40
					_ = d416
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl15 := ctx.ReserveLabel()
					_ = lbl15
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d415 = JITPrepareScmerGoArg(ctx, d415)
					d416 = JITPrepareGoSliceArg(ctx, d416)
					if d416.Loc != LocRegTriple && d416.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d417 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d417.Loc == LocRegPair || d417.Loc == LocStackPair || d417.Loc == LocRegTriple || d417.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d415)
					ctx.SyncDesc(&d416)
					ctx.SyncDesc(&d417)
					d418 = ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d415, d416, d417}, 2)
					d418.NoHeapPointer = false
					ctx.BindReg(d418.Reg, &d418)
					ctx.BindReg(d418.Reg2, &d418)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d418)
					d419 = args[0]
					d419.ID = 0
					ctx.SyncDesc(&d419)
					if d419.Loc == LocRegPair || d419.Loc == LocStackPair || d419.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d419, &result)
						result.Type = d419.Type
					} else {
						switch d419.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d419)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d419)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d419)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d419, &result)
							result.Type = d419.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
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
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d95 JITValueDesc
				_ = d95
				var d96 JITValueDesc
				_ = d96
				var d97 JITValueDesc
				_ = d97
				var d130 JITValueDesc
				_ = d130
				var d131 JITValueDesc
				_ = d131
				var d166 JITValueDesc
				_ = d166
				var d167 JITValueDesc
				_ = d167
				var d168 JITValueDesc
				_ = d168
				var d169 JITValueDesc
				_ = d169
				var d170 JITValueDesc
				_ = d170
				var d172 JITValueDesc
				_ = d172
				var d173 JITValueDesc
				_ = d173
				var d174 JITValueDesc
				_ = d174
				var d175 JITValueDesc
				_ = d175
				var d176 JITValueDesc
				_ = d176
				var d177 JITValueDesc
				_ = d177
				var d178 JITValueDesc
				_ = d178
				var d225 JITValueDesc
				_ = d225
				var d226 JITValueDesc
				_ = d226
				var d227 JITValueDesc
				_ = d227
				var d228 JITValueDesc
				_ = d228
				var d229 JITValueDesc
				_ = d229
				var d230 JITValueDesc
				_ = d230
				var d231 JITValueDesc
				_ = d231
				var d232 JITValueDesc
				_ = d232
				var d233 JITValueDesc
				_ = d233
				var d234 JITValueDesc
				_ = d234
				var d235 JITValueDesc
				_ = d235
				var d236 JITValueDesc
				_ = d236
				var d237 JITValueDesc
				_ = d237
				var d238 JITValueDesc
				_ = d238
				var d239 JITValueDesc
				_ = d239
				var d240 JITValueDesc
				_ = d240
				var d241 JITValueDesc
				_ = d241
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [13]BBDescriptor
				bbs[7].PhiBase = int32(phiBase0) + int32(0)
				bbs[10].PhiBase = int32(phiBase0) + int32(16)
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d3 = args[0]
					d3.ID = 0
					if d3.Type == tagSlice {
						d4 = jitKnownSliceHeader(ctx, &d3)
					} else {
						d4 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d3}, 3)
					}
					if d4.Loc == LocRegTriple {
						ctx.BindReg(d4.Reg, &d4)
						ctx.BindReg(d4.Reg2, &d4)
						ctx.BindReg(d4.Reg3, &d4)
					}
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.FreeDesc(&d3)
					d5 = args[1]
					d5.ID = 0
					ctx.StabilizeDescForControlFlow(&d5)
					d6 = args[2]
					d6.ID = 0
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
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d11.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d10)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap12 := d1
						snap13 := d2
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
						bbs[2].Render()
						ctx.RestoreAllocState(alloc23)
						d1 = snap12
						d2 = snap13
						d3 = snap14
						d4 = snap15
						d5 = snap16
						d6 = snap17
						d7 = snap18
						d8 = snap19
						d9 = snap20
						d10 = snap21
						d11 = snap22
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["window_flush"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d24 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					d26 = ctx.EmitSliceElementAddress(&d4, &d24, 16)
					ctx.EnsureDesc(&d26)
					r1 := ctx.AllocRegExcept(d26.Reg)
					ctx.EmitMovRegMem(r1, d26.Reg, 8)
					ctx.EmitMovRegMem(d26.Reg, d26.Reg, 0)
					d25 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d26.Reg, Reg2: r1}
					ctx.BindReg(d26.Reg, &d25)
					ctx.BindReg(r1, &d25)
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
					d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(3)}
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
						d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2}
						ctx.BindReg(d4.Reg2, &d30)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d29)
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocImm && d29.Loc == LocImm {
						d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d30.Imm.Int() - d29.Imm.Int())}
					} else {
						r2 := ctx.AllocReg()
						if d30.Loc == LocImm {
							ctx.EmitMovRegImm64(r2, uint64(d30.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r2, d30.Reg)
						}
						if d29.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d29.Imm.Int()))
							ctx.EmitSubInt64(r2, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r2, d29.Reg)
						}
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d32)
					}
					r3 := ctx.EmitSliceDataAfterLow(&d4, &d29, 16)
					d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
					ctx.BindReg(r3, &d33)
					ctx.BindReg(r3, &d33)
					var r4 Reg
					var r5 Reg
					ctx.SyncDesc(&d33)
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocImm {
						r4 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r4, uint64(d33.Imm.Int()))
					} else {
						r4 = d33.Reg
					}
					ctx.ProtectReg(r4)
					ctx.SyncDesc(&d32)
					ctx.EnsureDesc(&d32)
					if d32.Loc == LocImm {
						r5 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r5, uint64(d32.Imm.Int()))
					} else {
						r5 = d32.Reg
					}
					ctx.ProtectReg(r5)
					r6 := ctx.EmitSliceCapAfterLow(&d4, &d29, r4, r5)
					ctx.UnprotectReg(r5)
					ctx.UnprotectReg(r4)
					d34 = JITValueDesc{Loc: LocRegTriple, Reg: r4, Reg2: r5, Reg3: r6}
					ctx.BindReg(r4, &d34)
					ctx.BindReg(r5, &d34)
					ctx.BindReg(r6, &d34)
					ctx.BindReg(r4, &d34)
					ctx.BindReg(r5, &d34)
					ctx.BindReg(r6, &d34)
					ctx.StabilizeDescForControlFlow(&d34)
					ctx.EnsureDesc(&d27)
					if d27.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d27.Imm.Int() <= 0)}
					} else {
						r7 := ctx.AllocRegExcept(d27.Reg)
						ctx.EmitCmpRegImm32(d27.Reg, 0)
						d35 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondSignedLessOrEqual}
						ctx.BindReg(r7, &d35)
					}
					d36 = d35
					ctx.EnsureDesc(&d36)
					if d36.Loc != LocImm && d36.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d36.Loc == LocImm {
						if d36.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitJump(d36.Condition, lbl4)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d35)
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap37 := d1
						snap38 := d2
						snap39 := d3
						snap40 := d4
						snap41 := d5
						snap42 := d6
						snap43 := d7
						snap44 := d8
						snap45 := d9
						snap46 := d10
						snap47 := d11
						snap48 := d24
						snap49 := d25
						snap50 := d26
						snap51 := d27
						snap52 := d28
						snap53 := d29
						snap54 := d30
						snap55 := d31
						snap56 := d32
						snap57 := d33
						snap58 := d34
						snap59 := d35
						snap60 := d36
						alloc61 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc61)
						d1 = snap37
						d2 = snap38
						d3 = snap39
						d4 = snap40
						d5 = snap41
						d6 = snap42
						d7 = snap43
						d8 = snap44
						d9 = snap45
						d10 = snap46
						d11 = snap47
						d24 = snap48
						d25 = snap49
						d26 = snap50
						d27 = snap51
						d28 = snap52
						d29 = snap53
						d30 = snap54
						d31 = snap55
						d32 = snap56
						d33 = snap57
						d34 = snap58
						d35 = snap59
						d36 = snap60
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["window_flush"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[7].PhiBase)+int32(0))
					return bbs[7].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					if d34.SliceSizeKnown {
						d62 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d34.KnownSliceLen))}
					} else if d34.Loc == LocImm {
						d62 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d34.StackOff))}
					} else if d34.Loc == LocStackTriple {
						d62 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d34.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d34)
						if d34.Loc == LocRegPair || d34.Loc == LocRegTriple {
							d62 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg2, ID: 0}
						} else if d34.Loc == LocReg {
							d62 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d62)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDescsTogether(&d62, &d27)
					if d62.Loc == LocImm && d27.Loc == LocImm {
						d63 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d62.Imm.Int() % d27.Imm.Int())}
					} else {
						d63 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntRem), []JITValueDesc{d62, d27}, 1)
					}
					if d63.Loc == LocReg && d62.Loc == LocReg && d63.Reg == d62.Reg {
						ctx.TransferReg(d62.Reg)
						d62.Loc = LocNone
					}
					ctx.FreeDesc(&d62)
					ctx.EnsureDesc(&d63)
					if d63.Loc == LocImm {
						d64 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d63.Imm.Int() != 0)}
					} else {
						r8 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d63.Reg, 0)
						d64 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondNotEqual}
						ctx.BindReg(r8, &d64)
					}
					ctx.FreeDesc(&d63)
					d65 = d64
					ctx.EnsureDesc(&d65)
					if d65.Loc != LocImm && d65.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d65.Loc == LocImm {
						if d65.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitJump(d65.Condition, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d64)
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap66 := d1
						snap67 := d2
						snap68 := d3
						snap69 := d4
						snap70 := d5
						snap71 := d6
						snap72 := d7
						snap73 := d8
						snap74 := d9
						snap75 := d10
						snap76 := d11
						snap77 := d24
						snap78 := d25
						snap79 := d26
						snap80 := d27
						snap81 := d28
						snap82 := d29
						snap83 := d30
						snap84 := d31
						snap85 := d32
						snap86 := d33
						snap87 := d34
						snap88 := d35
						snap89 := d36
						snap90 := d62
						snap91 := d63
						snap92 := d64
						snap93 := d65
						alloc94 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc94)
						d1 = snap66
						d2 = snap67
						d3 = snap68
						d4 = snap69
						d5 = snap70
						d6 = snap71
						d7 = snap72
						d8 = snap73
						d9 = snap74
						d10 = snap75
						d11 = snap76
						d24 = snap77
						d25 = snap78
						d26 = snap79
						d27 = snap80
						d28 = snap81
						d29 = snap82
						d30 = snap83
						d31 = snap84
						d32 = snap85
						d33 = snap86
						d34 = snap87
						d35 = snap88
						d36 = snap89
						d62 = snap90
						d63 = snap91
						d64 = snap92
						d65 = snap93
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					if d34.SliceSizeKnown {
						d95 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d34.KnownSliceLen))}
					} else if d34.Loc == LocImm {
						d95 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d34.StackOff))}
					} else if d34.Loc == LocStackTriple {
						d95 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d34.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d34)
						if d34.Loc == LocRegPair || d34.Loc == LocRegTriple {
							d95 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg2, ID: 0}
						} else if d34.Loc == LocReg {
							d95 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d95)
					if d95.Loc == LocImm {
						d96 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d95.Imm.Int() == 0)}
					} else {
						r9 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d95.Reg, 0)
						d96 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondEqual}
						ctx.BindReg(r9, &d96)
					}
					ctx.FreeDesc(&d95)
					d97 = d96
					ctx.EnsureDesc(&d97)
					if d97.Loc != LocImm && d97.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d97.Loc == LocImm {
						if d97.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d97.Condition, lbl4)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d96)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap98 := d1
						snap99 := d2
						snap100 := d3
						snap101 := d4
						snap102 := d5
						snap103 := d6
						snap104 := d7
						snap105 := d8
						snap106 := d9
						snap107 := d10
						snap108 := d11
						snap109 := d24
						snap110 := d25
						snap111 := d26
						snap112 := d27
						snap113 := d28
						snap114 := d29
						snap115 := d30
						snap116 := d31
						snap117 := d32
						snap118 := d33
						snap119 := d34
						snap120 := d35
						snap121 := d36
						snap122 := d62
						snap123 := d63
						snap124 := d64
						snap125 := d65
						snap126 := d95
						snap127 := d96
						snap128 := d97
						alloc129 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc129)
						d1 = snap98
						d2 = snap99
						d3 = snap100
						d4 = snap101
						d5 = snap102
						d6 = snap103
						d7 = snap104
						d8 = snap105
						d9 = snap106
						d10 = snap107
						d11 = snap108
						d24 = snap109
						d25 = snap110
						d26 = snap111
						d27 = snap112
						d28 = snap113
						d29 = snap114
						d30 = snap115
						d31 = snap116
						d32 = snap117
						d33 = snap118
						d34 = snap119
						d35 = snap120
						d36 = snap121
						d62 = snap122
						d63 = snap123
						d64 = snap124
						d65 = snap125
						d95 = snap126
						d96 = snap127
						d97 = snap128
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDescsTogether(&d1, &d7)
					if d1.Loc == LocImm && d7.Loc == LocImm {
						d130 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d7.Imm.Int())}
					} else if d7.Loc == LocImm {
						r10 := ctx.AllocRegExcept(d1.Reg)
						if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d7.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d7.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, ctx.ScratchReg)
						}
						d130 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondSignedLess}
						ctx.BindReg(r10, &d130)
					} else if d1.Loc == LocImm {
						r11 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d7.Reg)
						d130 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r11, Condition: CondSignedLess}
						ctx.BindReg(r11, &d130)
					} else {
						r12 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d7.Reg)
						d130 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedLess}
						ctx.BindReg(r12, &d130)
					}
					ctx.FreeDesc(&d7)
					d131 = d130
					ctx.EnsureDesc(&d131)
					if d131.Loc != LocImm && d131.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d131.Loc == LocImm {
						if d131.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[9].Render()
					}
					ctx.EmitJump(d131.Condition, lbl9)
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
					}
					ctx.FreeDesc(&d130)
					ctx.FlushRegisterMoves()
					if !bbs[9].Rendered {
						snap132 := d1
						snap133 := d2
						snap134 := d3
						snap135 := d4
						snap136 := d5
						snap137 := d6
						snap138 := d7
						snap139 := d8
						snap140 := d9
						snap141 := d10
						snap142 := d11
						snap143 := d24
						snap144 := d25
						snap145 := d26
						snap146 := d27
						snap147 := d28
						snap148 := d29
						snap149 := d30
						snap150 := d31
						snap151 := d32
						snap152 := d33
						snap153 := d34
						snap154 := d35
						snap155 := d36
						snap156 := d62
						snap157 := d63
						snap158 := d64
						snap159 := d65
						snap160 := d95
						snap161 := d96
						snap162 := d97
						snap163 := d130
						snap164 := d131
						alloc165 := ctx.SnapshotAllocState()
						bbs[9].Render()
						ctx.RestoreAllocState(alloc165)
						d1 = snap132
						d2 = snap133
						d3 = snap134
						d4 = snap135
						d5 = snap136
						d6 = snap137
						d7 = snap138
						d8 = snap139
						d9 = snap140
						d10 = snap141
						d11 = snap142
						d24 = snap143
						d25 = snap144
						d26 = snap145
						d27 = snap146
						d28 = snap147
						d29 = snap148
						d30 = snap149
						d31 = snap150
						d32 = snap151
						d33 = snap152
						d34 = snap153
						d35 = snap154
						d36 = snap155
						d62 = snap156
						d63 = snap157
						d64 = snap158
						d65 = snap159
						d95 = snap160
						d96 = snap161
						d97 = snap162
						d130 = snap163
						d131 = snap164
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d34)
					if d34.Loc == LocRegPair || d34.Loc == LocRegTriple {
						d166 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg2}
						ctx.BindReg(d34.Reg2, &d166)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d166)
					if d166.Loc == LocImm && d27.Loc == LocImm {
						d168 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d166.Imm.Int() - d27.Imm.Int())}
					} else {
						r13 := ctx.AllocReg()
						if d166.Loc == LocImm {
							ctx.EmitMovRegImm64(r13, uint64(d166.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r13, d166.Reg)
						}
						if d27.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d27.Imm.Int()))
							ctx.EmitSubInt64(r13, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r13, d27.Reg)
						}
						d168 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r13}
						ctx.BindReg(r13, &d168)
					}
					r14 := ctx.EmitSliceDataAfterLow(&d34, &d27, 16)
					d169 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
					ctx.BindReg(r14, &d169)
					ctx.BindReg(r14, &d169)
					var r15 Reg
					var r16 Reg
					ctx.SyncDesc(&d169)
					ctx.EnsureDesc(&d169)
					if d169.Loc == LocImm {
						r15 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r15, uint64(d169.Imm.Int()))
					} else {
						r15 = d169.Reg
					}
					ctx.ProtectReg(r15)
					ctx.SyncDesc(&d168)
					ctx.EnsureDesc(&d168)
					if d168.Loc == LocImm {
						r16 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r16, uint64(d168.Imm.Int()))
					} else {
						r16 = d168.Reg
					}
					ctx.ProtectReg(r16)
					r17 := ctx.EmitSliceCapAfterLow(&d34, &d27, r15, r16)
					ctx.UnprotectReg(r16)
					ctx.UnprotectReg(r15)
					d170 = JITValueDesc{Loc: LocRegTriple, Reg: r15, Reg2: r16, Reg3: r17}
					ctx.BindReg(r15, &d170)
					ctx.BindReg(r16, &d170)
					ctx.BindReg(r17, &d170)
					ctx.BindReg(r15, &d170)
					ctx.BindReg(r16, &d170)
					ctx.BindReg(r17, &d170)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d170)
					callResults171 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d34, d170}, []uint8{1}, []uint8{0})
					d172 = callResults171[0]
					d172.Type = tagInt
					if d34.SliceSizeKnown {
						d173 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d34.KnownSliceLen))}
					} else if d34.Loc == LocImm {
						d173 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d34.StackOff))}
					} else if d34.Loc == LocStackTriple {
						d173 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d34.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d34)
						if d34.Loc == LocRegPair || d34.Loc == LocRegTriple {
							d173 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg2, ID: 0}
						} else if d34.Loc == LocReg {
							d173 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d173)
					ctx.EnsureDesc(&d27)
					ctx.SyncDesc(&d173)
					ctx.SyncDesc(&d27)
					if d173.Loc == LocImm && d27.Loc == LocImm {
						d174 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d173.Imm.Int() - d27.Imm.Int())}
					} else if d27.Loc == LocImm && d27.Imm.Int() == 0 {
						ctx.EnsureDesc(&d173)
						r18 := ctx.AllocRegExcept(d173.Reg)
						ctx.EmitMovRegReg(r18, d173.Reg)
						d174 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d174)
					} else if d173.Loc == LocImm {
						ctx.EnsureDesc(&d27)
						scratch := ctx.AllocRegExcept(d27.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d173.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d27)
						d174 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d174)
					} else if d27.Loc == LocImm {
						ctx.EnsureDesc(&d173)
						scratch := ctx.AllocRegExcept(d173.Reg)
						ctx.EmitMovRegReg(scratch, d173.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d27.Imm.Int())
						d174 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d174)
					} else {
						ctx.EnsureDesc(&d173)
						ctx.SyncDesc(&d27)
						r19 := ctx.AllocRegExceptOperand(&d27, d173.Reg)
						ctx.EmitMovRegReg(r19, d173.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r19, &d27)
						d174 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
						ctx.BindReg(r19, &d174)
					}
					if d174.Loc == LocReg && d173.Loc == LocReg && d174.Reg == d173.Reg {
						ctx.TransferReg(d173.Reg)
						d173.Loc = LocNone
					}
					ctx.EnsureDesc(&d174)
					ctx.EmitStoreToStack(d174, int32(bbs[10].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d174)
					ctx.FreeDesc(&d173)
					ctx.FreeDesc(&d27)
					return bbs[10].Render()
					return result
				}
				bbs[9].Render = func() JITValueDesc {
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					bbs[9].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d175 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d175)
					if d175.Loc == LocRegPair || d175.Loc == LocStackPair || d175.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d175, &result)
						result.Type = d175.Type
					} else {
						switch d175.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d175)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d175)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d175)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d175, &result)
							result.Type = d175.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[10].Render = func() JITValueDesc {
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					bbs[10].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					if d34.SliceSizeKnown {
						d176 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d34.KnownSliceLen))}
					} else if d34.Loc == LocImm {
						d176 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d34.StackOff))}
					} else if d34.Loc == LocStackTriple {
						d176 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d34.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d34)
						if d34.Loc == LocRegPair || d34.Loc == LocRegTriple {
							d176 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg2, ID: 0}
						} else if d34.Loc == LocReg {
							d176 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d176)
					ctx.EnsureDescsTogether(&d2, &d176)
					if d2.Loc == LocImm && d176.Loc == LocImm {
						d177 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d176.Imm.Int())}
					} else if d176.Loc == LocImm {
						r20 := ctx.AllocRegExcept(d2.Reg)
						if d176.Imm.Int() >= -2147483648 && d176.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d2.Reg, int32(d176.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d176.Imm.Int()))
							ctx.EmitCmpInt64(d2.Reg, ctx.ScratchReg)
						}
						d177 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r20, Condition: CondSignedLess}
						ctx.BindReg(r20, &d177)
					} else if d2.Loc == LocImm {
						r21 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d176.Reg)
						d177 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r21, Condition: CondSignedLess}
						ctx.BindReg(r21, &d177)
					} else {
						r22 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpInt64(d2.Reg, d176.Reg)
						d177 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondSignedLess}
						ctx.BindReg(r22, &d177)
					}
					ctx.FreeDesc(&d176)
					d178 = d177
					ctx.EnsureDesc(&d178)
					if d178.Loc != LocImm && d178.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d178.Loc == LocImm {
						if d178.Imm.Bool() {
							return bbs[11].Render()
						}
						return bbs[12].Render()
					}
					ctx.EmitJump(d178.Condition, lbl12)
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
					}
					ctx.FreeDesc(&d177)
					ctx.FlushRegisterMoves()
					if !bbs[12].Rendered {
						snap179 := d1
						snap180 := d2
						snap181 := d3
						snap182 := d4
						snap183 := d5
						snap184 := d6
						snap185 := d7
						snap186 := d8
						snap187 := d9
						snap188 := d10
						snap189 := d11
						snap190 := d24
						snap191 := d25
						snap192 := d26
						snap193 := d27
						snap194 := d28
						snap195 := d29
						snap196 := d30
						snap197 := d31
						snap198 := d32
						snap199 := d33
						snap200 := d34
						snap201 := d35
						snap202 := d36
						snap203 := d62
						snap204 := d63
						snap205 := d64
						snap206 := d65
						snap207 := d95
						snap208 := d96
						snap209 := d97
						snap210 := d130
						snap211 := d131
						snap212 := d166
						snap213 := d167
						snap214 := d168
						snap215 := d169
						snap216 := d170
						snap217 := d172
						snap218 := d173
						snap219 := d174
						snap220 := d175
						snap221 := d176
						snap222 := d177
						snap223 := d178
						alloc224 := ctx.SnapshotAllocState()
						bbs[12].Render()
						ctx.RestoreAllocState(alloc224)
						d1 = snap179
						d2 = snap180
						d3 = snap181
						d4 = snap182
						d5 = snap183
						d6 = snap184
						d7 = snap185
						d8 = snap186
						d9 = snap187
						d10 = snap188
						d11 = snap189
						d24 = snap190
						d25 = snap191
						d26 = snap192
						d27 = snap193
						d28 = snap194
						d29 = snap195
						d30 = snap196
						d31 = snap197
						d32 = snap198
						d33 = snap199
						d34 = snap200
						d35 = snap201
						d36 = snap202
						d62 = snap203
						d63 = snap204
						d64 = snap205
						d65 = snap206
						d95 = snap207
						d96 = snap208
						d97 = snap209
						d130 = snap210
						d131 = snap211
						d166 = snap212
						d167 = snap213
						d168 = snap214
						d169 = snap215
						d170 = snap216
						d172 = snap217
						d173 = snap218
						d174 = snap219
						d175 = snap220
						d176 = snap221
						d177 = snap222
						d178 = snap223
					}
					if !bbs[11].Rendered {
						return bbs[11].Render()
					}
					return result
					return result
				}
				bbs[11].Render = func() JITValueDesc {
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
						return result
					}
					bbs[11].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d225 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d2)
					ctx.SyncDesc(&d225)
					d226 = d34
					d226.ID = 0
					d227 = d2
					d227.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d226, &d227, &d225, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d2)
						d227 = d2
						d227.ID = 0
						ctx.EmitStoreScmerSliceElement(&d226, &d227, &d225, int32(16))
					}
					ctx.FreeDesc(&d227)
					ctx.FreeDesc(&d225)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d228 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d228 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d228)
					}
					if d228.Loc == LocReg && d2.Loc == LocReg && d228.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d228)
					ctx.EmitStoreToStack(d228, int32(bbs[10].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d228)
					return bbs[10].Render()
					return result
				}
				bbs[12].Render = func() JITValueDesc {
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
						return result
					}
					bbs[12].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d229 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					d231 = ctx.EmitSliceElementAddress(&d4, &d229, 16)
					ctx.EnsureDesc(&d231)
					r23 := ctx.AllocRegExcept(d231.Reg)
					ctx.EmitMovRegMem(r23, d231.Reg, 8)
					ctx.EmitMovRegMem(d231.Reg, d231.Reg, 0)
					d230 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d231.Reg, Reg2: r23}
					ctx.BindReg(d231.Reg, &d230)
					ctx.BindReg(r23, &d230)
					if d230.Loc == LocImm {
						d232 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d230.Imm.Int())}
					} else if d230.Type == tagInt && d230.Loc == LocRegPair {
						ctx.FreeReg(d230.Reg)
						d232 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d230.Reg2}
						ctx.BindReg(d230.Reg2, &d232)
						ctx.BindReg(d230.Reg2, &d232)
					} else if d230.Type == tagInt && d230.Loc == LocReg {
						d232 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d230.Reg}
						ctx.BindReg(d230.Reg, &d232)
						ctx.BindReg(d230.Reg, &d232)
					} else {
						d232 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d230}, 1)
						d232.Type = tagInt
						ctx.BindReg(d232.Reg, &d232)
					}
					ctx.FreeDesc(&d230)
					ctx.EnsureDesc(&d232)
					ctx.EnsureDesc(&d232)
					if d232.Loc == LocImm {
						d233 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d232.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d232.Reg)
						ctx.EmitMovRegReg(scratch, d232.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d233 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d233)
					}
					if d233.Loc == LocReg && d232.Loc == LocReg && d233.Reg == d232.Reg {
						ctx.TransferReg(d232.Reg)
						d232.Loc = LocNone
					}
					ctx.FreeDesc(&d232)
					ctx.EnsureDesc(&d233)
					d234 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.SyncDesc(&d233)
					d235 = d4
					d235.ID = 0
					d236 = d234
					d236.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d235, &d236, &d233, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d235, &d236, &d233, int32(16))
					}
					ctx.FreeDesc(&d236)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d34)
					d237 = d5
					_ = d237
					d238 = d34
					_ = d238
					ctx.StabilizeDescForControlFlow(&d34)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl14 := ctx.ReserveLabel()
					_ = lbl14
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d237 = JITPrepareScmerGoArg(ctx, d237)
					d238 = JITPrepareGoSliceArg(ctx, d238)
					if d238.Loc != LocRegTriple && d238.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d239 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d239.Loc == LocRegPair || d239.Loc == LocStackPair || d239.Loc == LocRegTriple || d239.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d237)
					ctx.SyncDesc(&d238)
					ctx.SyncDesc(&d239)
					d240 = ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d237, d238, d239}, 2)
					d240.NoHeapPointer = false
					ctx.BindReg(d240.Reg, &d240)
					ctx.BindReg(d240.Reg2, &d240)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d240)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d241 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d241 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d241)
					}
					if d241.Loc == LocReg && d1.Loc == LocReg && d241.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d241)
					ctx.EmitStoreToStack(d241, int32(bbs[7].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d241)
					return bbs[7].Render()
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  62,
		},
	})
}
