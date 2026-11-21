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
import "context"
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
	result := ApplyOnce(a[3], nil)
	if limit == 0 {
		return result
	}
	keep := offset + limit
	h := streamTopK{compare: PrepareSerialProc(a[5])}
	var mu sync.Mutex
	var sequence uint64
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	emit := NewClusterCallback(ctx, func(values ...Scmer) Scmer {
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
	}, nil)
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

// ownedWindowSlots validates only the caller-owned serial accumulator. Its
// storage is never retained by a producer or copied into an output row bag.
func ownedWindowSlots(win []Scmer, label, missing string) (slots []Scmer, stride int) {
	if len(win) < 3 {
		panic(label + missing)
	}
	stride = int(win[2].Int())
	slots = win[3:]
	if stride <= 0 || len(slots) == 0 || len(slots)%stride != 0 {
		panic(label + ": invalid window dimensions")
	}
	return
}

func shiftOwnedWindow(win, slots []Scmer, stride int, vals []Scmer) {
	// Keep borrowed emitter arguments physically oldest-to-newest, exactly as
	// the serial window owner requires. No callback frame is retained.
	copy(slots, slots[stride:])
	tail := slots[len(slots)-stride:]
	for i := range tail {
		if i < len(vals) {
			tail[i] = vals[i]
		} else {
			tail[i] = NewNil()
		}
	}
	win[1] = NewInt(win[1].Int() + 1)
}

func windowMut(a ...Scmer) Scmer {
	win := asSlice(a[0], "window_mut")
	vals := asSlice(a[2], "window_mut vals")
	slots, stride := ownedWindowSlots(win, "window_mut", ": window must have at least 3 elements (skip, counter, stride)")
	skip := int(win[0].Int())
	shiftOwnedWindow(win, slots, stride, vals)
	if skip > 0 {
		win[0] = NewInt(int64(skip - 1))
		return a[0]
	}
	Apply(a[1], slots...)
	return a[0]
}

func windowFlush(a ...Scmer) Scmer {
	win := asSlice(a[0], "window_flush")
	count := int(a[2].Int())
	slots, stride := ownedWindowSlots(win, "window_flush", ": window must have at least 3 elements")
	for n := 0; n < count; n++ {
		shiftOwnedWindow(win, slots, stride, nil)
		Apply(a[1], slots...)
	}
	return NewNil()
}

func init_window() {
	DeclareTitle("Window Functions")
	initWindowReducers()

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
				d7 := ctx.EmitGoCallScalar(jitGoABI_4138fb0145801900, []JITValueDesc{d4, d5, d6}, 2)
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
			result = ApplyOnce(a[3], nil)
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			emit := NewClusterCallback(ctx, func(values ...Scmer) Scmer {
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
			}, nil)
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
				{Kind: "any", Label: "neutral", Description: "initial accumulator or zero-argument factory"},
				{Kind: "func", Label: "producer", Description: "nested streaming plan called with a one-value emit callback", Params: []*TypeDescriptor{{Kind: "func", Label: "emit", Description: "emits one complete value", Params: []*TypeDescriptor{{Kind: "any", Label: "value"}}, Return: &TypeDescriptor{Kind: "any", Label: "result"}}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", Label: "less", Description: "optional strict ordering for an unordered producer; retains only offset+limit values and requires a finite limit", Optional: true, Params: []*TypeDescriptor{{Kind: "any", Label: "left"}, {Kind: "any", Label: "right"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return: &TypeDescriptor{Kind: "any"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: deferred call.
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
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
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
				var d16 JITValueDesc
				_ = d16
				var d18 JITValueDesc
				_ = d18
				var d20 JITValueDesc
				_ = d20
				var phiBase22 int32
				_ = phiBase22
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
				var d39 JITValueDesc
				_ = d39
				var d41 JITValueDesc
				_ = d41
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
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d66 JITValueDesc
				_ = d66
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
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
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
				var d134 JITValueDesc
				_ = d134
				var d135 JITValueDesc
				_ = d135
				var d136 JITValueDesc
				_ = d136
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
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
				var bbs [3]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
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
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					if d0.Type == tagSlice {
						d1 = jitKnownSliceHeader(ctx, &d0)
					} else {
						d1 = ctx.EmitGoCallScalar(jitGoABI_ab569823a8d42a56, []JITValueDesc{d0}, 3)
					}
					if d1.Loc == LocRegTriple {
						ctx.BindReg(d1.Reg, &d1)
						ctx.BindReg(d1.Reg2, &d1)
						ctx.BindReg(d1.Reg3, &d1)
					}
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.FreeDesc(&d0)
					d2 = args[2]
					d2.ID = 0
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(jitGoABI_ab569823a8d42a56, []JITValueDesc{d2}, 3)
					}
					if d3.Loc == LocRegTriple {
						ctx.BindReg(d3.Reg, &d3)
						ctx.BindReg(d3.Reg2, &d3)
						ctx.BindReg(d3.Reg3, &d3)
					}
					ctx.FreeDesc(&d2)
					d1 = JITPrepareGoSliceArg(ctx, d1)
					if d1.Loc != LocRegTriple && d1.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ownedWindowSlots arg0)")
					}
					d4 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("window_mut")}
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d4.Imm)
						ptrWord, _ := d4.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d4.Imm.String())))
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
					if d4.Loc != LocRegPair && d4.Loc != LocStackPair && d4.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ownedWindowSlots arg1)")
					}
					d5 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(": window must have at least 3 elements (skip, counter, stride)")}
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d5.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d5.Imm)
						ptrWord, _ := d5.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d5.Imm.String())))
						d5 = tmpPair
					} else if d5.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d5.Type, Reg: ctx.AllocRegExcept(d5.Reg), Reg2: ctx.AllocRegExcept(d5.Reg)}
						switch d5.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d5)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d5)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d5)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d5)
						d5 = tmpPair
					}
					if d5.Loc != LocRegPair && d5.Loc != LocStackPair && d5.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ownedWindowSlots arg2)")
					}
					ctx.SyncDesc(&d1)
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d5)
					callResults6 := JITEmitGoCallResults(ctx, GoFuncAddr(ownedWindowSlots), []JITValueDesc{d1, d4, d5})
					ctx.FreeDesc(&d4)
					ctx.FreeDesc(&d5)
					d7 = callResults6[0]
					_ = d7
					d8 = callResults6[1]
					_ = d8
					ctx.StabilizeDescForControlFlow(&d7)
					d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d11 = ctx.EmitSliceElementAddress(&d1, &d9, 16)
					ctx.EnsureDesc(&d11)
					r0 := ctx.AllocRegExcept(d11.Reg)
					ctx.EmitMovRegMem(r0, d11.Reg, 8)
					ctx.EmitMovRegMem(d11.Reg, d11.Reg, 0)
					d10 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d11.Reg, Reg2: r0}
					ctx.BindReg(d11.Reg, &d10)
					ctx.BindReg(r0, &d10)
					if d10.Loc != LocImm {
						ctx.EnsureDesc(&d10)
					}
					if d10.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d10.Imm.Int())}
					} else if d10.Type == tagInt && d10.Loc == LocRegPair {
						ctx.FreeReg(d10.Reg)
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg2}
						ctx.BindReg(d10.Reg2, &d12)
						ctx.BindReg(d10.Reg2, &d12)
					} else if d10.Type == tagInt && d10.Loc == LocReg {
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg}
						ctx.BindReg(d10.Reg, &d12)
						ctx.BindReg(d10.Reg, &d12)
					} else {
						boxed := JITPrepareScmerGoArg(ctx, d10)
						d12 = ctx.EmitGoCallScalar(jitGoABI_317743fd938661dd, []JITValueDesc{boxed}, 1)
						d12.Type = tagInt
						ctx.BindReg(d12.Reg, &d12)
					}
					ctx.FreeDesc(&d10)
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d12)
					ctx.StabilizeDescForControlFlow(&d12)
					d12 = ctx.StabilizeForNested(d12)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d3)
					d14 = d1
					_ = d14
					ctx.StabilizeDescForControlFlow(&d14)
					inlineHome15 := d14
					inlineHome15.ID = 0
					d16 = d7
					_ = d16
					ctx.StabilizeDescForControlFlow(&d16)
					inlineHome17 := d16
					inlineHome17.ID = 0
					d18 = d8
					_ = d18
					ctx.StabilizeDescForControlFlow(&d18)
					inlineHome19 := d18
					inlineHome19.ID = 0
					d20 = d3
					_ = d20
					ctx.StabilizeDescForControlFlow(&d20)
					inlineHome21 := d20
					inlineHome21.ID = 0
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.StabilizeDescForControlFlow(&d7)
					phiBase22 = ctx.AllocStack(int32(16))
					d23 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase22) + int32(0)}
					_ = d23
					lbl4 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl5 := ctx.ReserveLabel()
					_ = lbl5
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_4 := int32(-1)
					_ = bbpos_1_4
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_1_5 := int32(-1)
					_ = bbpos_1_5
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d14 = inlineHome15
					d16 = inlineHome17
					d18 = inlineHome19
					d20 = inlineHome21
					d23 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase22) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair || d16.Loc == LocRegTriple {
						d24 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg2}
						ctx.BindReg(d16.Reg2, &d24)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d24)
					ctx.EnsureDescsTogether(&d24, &d18)
					if d24.Loc == LocImm && d18.Loc == LocImm {
						d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d24.Imm.Int() - d18.Imm.Int())}
					} else {
						ctx.ProtectDescRegisters(&d24)
						ctx.ProtectDescRegisters(&d18)
						r1 := ctx.AllocReg()
						if d24.Loc == LocImm {
							ctx.EmitMovRegImm64(r1, uint64(d24.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r1, d24.Reg)
						}
						if d18.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d18.Imm.Int()))
							ctx.EmitSubInt64(r1, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r1, d18.Reg)
						}
						ctx.UnprotectDescRegisters(&d18)
						ctx.UnprotectDescRegisters(&d24)
						d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d26)
					}
					r2 := ctx.EmitSliceDataAfterLow(&d16, &d18, 16)
					d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
					ctx.BindReg(r2, &d27)
					ctx.BindReg(r2, &d27)
					var r3 Reg
					var r4 Reg
					ctx.SyncDesc(&d27)
					ctx.EnsureDesc(&d27)
					if d27.Loc == LocImm {
						r3 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r3, uint64(d27.Imm.Int()))
					} else {
						r3 = d27.Reg
					}
					ctx.ProtectReg(r3)
					ctx.SyncDesc(&d26)
					ctx.EnsureDesc(&d26)
					if d26.Loc == LocImm {
						r4 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r4, uint64(d26.Imm.Int()))
					} else {
						r4 = d26.Reg
					}
					ctx.ProtectReg(r4)
					r5 := ctx.EmitSliceCapAfterLow(&d16, &d18, r3, r4)
					ctx.UnprotectReg(r4)
					ctx.UnprotectReg(r3)
					d28 = JITValueDesc{Loc: LocRegTriple, Reg: r3, Reg2: r4, Reg3: r5}
					ctx.BindReg(r3, &d28)
					ctx.BindReg(r4, &d28)
					ctx.BindReg(r5, &d28)
					ctx.BindReg(r3, &d28)
					ctx.BindReg(r4, &d28)
					ctx.BindReg(r5, &d28)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d28)
					callResults29 := JITEmitGoCallResults(ctx, jitGoABI_79ac37f1e11427b6, []JITValueDesc{d16, d28})
					d30 = callResults29[0]
					d30.Type = tagInt
					ctx.ReclaimUntrackedRegs()
					if d16.SliceSizeKnown {
						d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d16.KnownSliceLen))}
					} else if d16.Loc == LocImm {
						d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d16.StackOff))}
					} else if d16.Loc == LocStackTriple {
						d31 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d16.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d16)
						if d16.Loc == LocRegPair || d16.Loc == LocRegTriple {
							d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg2, ID: 0}
						} else if d16.Loc == LocReg {
							d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d31
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d31)
					ctx.EnsureDesc(&d18)
					ctx.SyncDesc(&d31)
					ctx.SyncDesc(&d18)
					if d31.Loc == LocImm && d18.Loc == LocImm {
						d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d31.Imm.Int() - d18.Imm.Int())}
					} else if d18.Loc == LocImm && d18.Imm.Int() == 0 {
						ctx.EnsureDesc(&d31)
						r6 := ctx.AllocRegExcept(d31.Reg)
						ctx.EmitMovRegReg(r6, d31.Reg)
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d32)
					} else if d31.Loc == LocImm {
						ctx.EnsureDesc(&d18)
						scratch := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d31.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d18)
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d32)
					} else if d18.Loc == LocImm {
						ctx.EnsureDesc(&d31)
						scratch := ctx.AllocRegExcept(d31.Reg)
						ctx.EmitMovRegReg(scratch, d31.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d18.Imm.Int())
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d32)
					} else {
						ctx.EnsureDesc(&d31)
						ctx.SyncDesc(&d18)
						r7 := ctx.AllocRegExceptOperand(&d18, d31.Reg)
						ctx.EmitMovRegReg(r7, d31.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r7, &d18)
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d32)
					}
					if d32.Loc == LocReg && d31.Loc == LocReg && d32.Reg == d31.Reg {
						ctx.TransferReg(d31.Reg)
						d31.Loc = LocNone
					}
					ctx.FreeDesc(&d31)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair || d16.Loc == LocRegTriple {
						d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg2}
						ctx.BindReg(d16.Reg2, &d33)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d33)
					ctx.EnsureDescsTogether(&d33, &d32)
					if d33.Loc == LocImm && d32.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d33.Imm.Int() - d32.Imm.Int())}
					} else {
						ctx.ProtectDescRegisters(&d33)
						ctx.ProtectDescRegisters(&d32)
						r8 := ctx.AllocReg()
						if d33.Loc == LocImm {
							ctx.EmitMovRegImm64(r8, uint64(d33.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r8, d33.Reg)
						}
						if d32.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d32.Imm.Int()))
							ctx.EmitSubInt64(r8, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r8, d32.Reg)
						}
						ctx.UnprotectDescRegisters(&d32)
						ctx.UnprotectDescRegisters(&d33)
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d35)
					}
					r9 := ctx.EmitSliceDataAfterLow(&d16, &d32, 16)
					d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
					ctx.BindReg(r9, &d36)
					ctx.BindReg(r9, &d36)
					var r10 Reg
					var r11 Reg
					ctx.SyncDesc(&d36)
					ctx.EnsureDesc(&d36)
					if d36.Loc == LocImm {
						r10 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r10, uint64(d36.Imm.Int()))
					} else {
						r10 = d36.Reg
					}
					ctx.ProtectReg(r10)
					ctx.SyncDesc(&d35)
					ctx.EnsureDesc(&d35)
					if d35.Loc == LocImm {
						r11 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r11, uint64(d35.Imm.Int()))
					} else {
						r11 = d35.Reg
					}
					ctx.ProtectReg(r11)
					r12 := ctx.EmitSliceCapAfterLow(&d16, &d32, r10, r11)
					ctx.UnprotectReg(r11)
					ctx.UnprotectReg(r10)
					d37 = JITValueDesc{Loc: LocRegTriple, Reg: r10, Reg2: r11, Reg3: r12}
					ctx.BindReg(r10, &d37)
					ctx.BindReg(r11, &d37)
					ctx.BindReg(r12, &d37)
					ctx.BindReg(r10, &d37)
					ctx.BindReg(r11, &d37)
					ctx.BindReg(r12, &d37)
					ctx.StabilizeDescForControlFlow(&d37)
					inlineHome38 := d37
					inlineHome38.ID = 0
					ctx.FreeDesc(&d32)
					ctx.ReclaimUntrackedRegs()
					if d37.SliceSizeKnown {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d37.KnownSliceLen))}
					} else if d37.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d37.StackOff))}
					} else if d37.Loc == LocStackTriple {
						d39 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d37.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d37)
						if d37.Loc == LocRegPair || d37.Loc == LocRegTriple {
							d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d37.Reg2, ID: 0}
						} else if d37.Loc == LocReg {
							d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d37.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d39
					ctx.StabilizeDescForControlFlow(&d39)
					inlineHome40 := d39
					inlineHome40.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(phiBase22)+int32(0))
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d14 = inlineHome15
					d16 = inlineHome17
					d18 = inlineHome19
					d20 = inlineHome21
					d37 = inlineHome38
					d39 = inlineHome40
					d23 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase22) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d23)
					if d23.Loc == LocImm {
						d41 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d23.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitMovRegReg(scratch, d23.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d41 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d41)
					}
					if d41.Loc == LocReg && d23.Loc == LocReg && d41.Reg == d23.Reg {
						ctx.TransferReg(d23.Reg)
						d23.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d41)
					inlineHome42 := d41
					inlineHome42.ID = 0
					ctx.FreeDesc(&d23)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d41)
					ctx.EnsureDesc(&d39)
					ctx.EnsureDescsTogether(&d41, &d39)
					if d41.Loc == LocImm && d39.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d41.Imm.Int() < d39.Imm.Int())}
					} else if d39.Loc == LocImm {
						r13 := ctx.AllocRegExcept(d41.Reg)
						if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d41.Reg, int32(d39.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d39.Imm.Int()))
							ctx.EmitCmpInt64(d41.Reg, ctx.ScratchReg)
						}
						d43 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r13, Condition: CondSignedLess}
						ctx.BindReg(r13, &d43)
					} else if d41.Loc == LocImm {
						r14 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d41.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d39.Reg)
						d43 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r14, Condition: CondSignedLess}
						ctx.BindReg(r14, &d43)
					} else {
						r15 := ctx.AllocRegExcept(d41.Reg)
						ctx.EmitCmpInt64(d41.Reg, d39.Reg)
						d43 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r15, Condition: CondSignedLess}
						ctx.BindReg(r15, &d43)
					}
					ctx.ReclaimUntrackedRegs()
					d44 = d43
					ctx.SyncDesc(&d44)
					if d44.Loc != LocImm && d44.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d44.Loc == LocImm {
						if d44.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d44.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d43)
						ctx.MarkLabel(lbl11)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d14 = inlineHome15
					d16 = inlineHome17
					d18 = inlineHome19
					d20 = inlineHome21
					d37 = inlineHome38
					d39 = inlineHome40
					d41 = inlineHome42
					d23 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase22) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.ReclaimUntrackedRegs()
					d47 = ctx.EmitSliceElementAddress(&d14, &d45, 16)
					ctx.EnsureDesc(&d47)
					r16 := ctx.AllocRegExcept(d47.Reg)
					ctx.EmitMovRegMem(r16, d47.Reg, 8)
					ctx.EmitMovRegMem(d47.Reg, d47.Reg, 0)
					d46 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d47.Reg, Reg2: r16}
					ctx.BindReg(d47.Reg, &d46)
					ctx.BindReg(r16, &d46)
					ctx.ReclaimUntrackedRegs()
					if d46.Loc != LocImm {
						ctx.EnsureDesc(&d46)
					}
					if d46.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d46.Imm.Int())}
					} else if d46.Type == tagInt && d46.Loc == LocRegPair {
						ctx.FreeReg(d46.Reg)
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg2}
						ctx.BindReg(d46.Reg2, &d48)
						ctx.BindReg(d46.Reg2, &d48)
					} else if d46.Type == tagInt && d46.Loc == LocReg {
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg}
						ctx.BindReg(d46.Reg, &d48)
						ctx.BindReg(d46.Reg, &d48)
					} else {
						boxed := JITPrepareScmerGoArg(ctx, d46)
						d48 = ctx.EmitGoCallScalar(jitGoABI_317743fd938661dd, []JITValueDesc{boxed}, 1)
						d48.Type = tagInt
						ctx.BindReg(d48.Reg, &d48)
					}
					ctx.FreeDesc(&d46)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d48)
					ctx.EnsureDesc(&d48)
					if d48.Loc == LocImm {
						d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d48.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d48.Reg)
						ctx.EmitMovRegReg(scratch, d48.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d49)
					}
					if d49.Loc == LocReg && d48.Loc == LocReg && d49.Reg == d48.Reg {
						ctx.TransferReg(d48.Reg)
						d48.Loc = LocNone
					}
					ctx.FreeDesc(&d48)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d49)
					ctx.ReclaimUntrackedRegs()
					d50 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d49)
					ctx.SyncDesc(&d14)
					ctx.SyncDesc(&d50)
					d51 = d14
					d51.ID = 0
					d52 = d50
					d52.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d51, &d52, &d49, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d51, &d52, &d49, int32(16))
					}
					ctx.FreeDesc(&d52)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl4)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d14 = inlineHome15
					d16 = inlineHome17
					d18 = inlineHome19
					d20 = inlineHome21
					d37 = inlineHome38
					d39 = inlineHome40
					d41 = inlineHome42
					d23 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase22) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d20.SliceSizeKnown {
						d53 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d20.KnownSliceLen))}
					} else if d20.Loc == LocImm {
						d53 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d20.StackOff))}
					} else if d20.Loc == LocStackTriple {
						d53 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d20.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d20)
						if d20.Loc == LocRegPair || d20.Loc == LocRegTriple {
							d53 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg2, ID: 0}
						} else if d20.Loc == LocReg {
							d53 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d53
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d41)
					ctx.EnsureDesc(&d53)
					ctx.EnsureDescsTogether(&d41, &d53)
					if d41.Loc == LocImm && d53.Loc == LocImm {
						d54 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d41.Imm.Int() < d53.Imm.Int())}
					} else if d53.Loc == LocImm {
						r17 := ctx.AllocRegExcept(d41.Reg)
						if d53.Imm.Int() >= -2147483648 && d53.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d41.Reg, int32(d53.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d53.Imm.Int()))
							ctx.EmitCmpInt64(d41.Reg, ctx.ScratchReg)
						}
						d54 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondSignedLess}
						ctx.BindReg(r17, &d54)
					} else if d41.Loc == LocImm {
						r18 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d41.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d53.Reg)
						d54 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r18, Condition: CondSignedLess}
						ctx.BindReg(r18, &d54)
					} else {
						r19 := ctx.AllocRegExcept(d41.Reg)
						ctx.EmitCmpInt64(d41.Reg, d53.Reg)
						d54 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r19, Condition: CondSignedLess}
						ctx.BindReg(r19, &d54)
					}
					ctx.FreeDesc(&d53)
					ctx.ReclaimUntrackedRegs()
					d55 = d54
					ctx.SyncDesc(&d55)
					if d55.Loc != LocImm && d55.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					if d55.Loc == LocImm {
						if d55.Imm.Bool() {
							ctx.MarkLabel(lbl13)
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl10)
						}
					} else {
						ctx.EmitJump(d55.Condition, lbl13)
						ctx.EmitJmp(lbl14)
						ctx.FreeDesc(&d54)
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl10)
					}
					bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d14 = inlineHome15
					d16 = inlineHome17
					d18 = inlineHome19
					d20 = inlineHome21
					d37 = inlineHome38
					d39 = inlineHome40
					d41 = inlineHome42
					d23 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase22) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d56 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d41)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d56)
					ctx.SyncDesc(&d37)
					ctx.SyncDesc(&d41)
					d57 = d37
					d57.ID = 0
					d58 = d41
					d58.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d57, &d58, &d56, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d41)
						d58 = d41
						d58.ID = 0
						ctx.EmitStoreScmerSliceElement(&d57, &d58, &d56, int32(16))
					}
					ctx.FreeDesc(&d58)
					ctx.FreeDesc(&d56)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d41)
					if d41.Loc == LocReg || d41.Loc == LocFPReg {
						ctx.ProtectReg(d41.Reg)
					} else if d41.Loc == LocRegPair {
						ctx.ProtectReg(d41.Reg)
						ctx.ProtectReg(d41.Reg2)
					}
					d59 = d41
					if d59.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d59)
					ctx.EmitStoreToStack(d59, int32(phiBase22)+int32(0))
					if d41.Loc == LocReg || d41.Loc == LocFPReg {
						ctx.UnprotectReg(d41.Reg)
					} else if d41.Loc == LocRegPair {
						ctx.UnprotectReg(d41.Reg)
						ctx.UnprotectReg(d41.Reg2)
					}
					ctx.EmitJmp(lbl6)
					bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d14 = inlineHome15
					d16 = inlineHome17
					d18 = inlineHome19
					d20 = inlineHome21
					d37 = inlineHome38
					d39 = inlineHome40
					d41 = inlineHome42
					d23 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase22) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d41)
					ctx.ReclaimUntrackedRegs()
					d61 = ctx.EmitSliceElementAddress(&d20, &d41, 16)
					ctx.EnsureDesc(&d61)
					r20 := ctx.AllocRegExcept(d61.Reg)
					ctx.EmitMovRegMem(r20, d61.Reg, 8)
					ctx.EmitMovRegMem(d61.Reg, d61.Reg, 0)
					d60 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d61.Reg, Reg2: r20}
					ctx.BindReg(d61.Reg, &d60)
					ctx.BindReg(r20, &d60)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d41)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d60)
					ctx.SyncDesc(&d37)
					ctx.SyncDesc(&d41)
					d62 = d37
					d62.ID = 0
					d63 = d41
					d63.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d62, &d63, &d60, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d41)
						d63 = d41
						d63.ID = 0
						ctx.EmitStoreScmerSliceElement(&d62, &d63, &d60, int32(16))
					}
					ctx.FreeDesc(&d63)
					ctx.FreeDesc(&d60)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d41)
					if d41.Loc == LocReg || d41.Loc == LocFPReg {
						ctx.ProtectReg(d41.Reg)
					} else if d41.Loc == LocRegPair {
						ctx.ProtectReg(d41.Reg)
						ctx.ProtectReg(d41.Reg2)
					}
					d64 = d41
					if d64.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d64)
					ctx.EmitStoreToStack(d64, int32(phiBase22)+int32(0))
					if d41.Loc == LocReg || d41.Loc == LocFPReg {
						ctx.UnprotectReg(d41.Reg)
					} else if d41.Loc == LocRegPair {
						ctx.UnprotectReg(d41.Reg)
						ctx.UnprotectReg(d41.Reg2)
					}
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl4)
					ctx.FreeDesc(&d8)
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						d65 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.Int() > 0)}
					} else {
						r21 := ctx.AllocRegExcept(d12.Reg)
						ctx.EmitCmpRegImm32(d12.Reg, 0)
						d65 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r21, Condition: CondSignedGreater}
						ctx.BindReg(r21, &d65)
					}
					d66 = d65
					ctx.SyncDesc(&d66)
					if d66.Loc != LocImm && d66.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d66.Loc == LocImm {
						if d66.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d66.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d65)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap67 := d0
						snap68 := d1
						snap69 := d2
						snap70 := d3
						snap71 := d4
						snap72 := d5
						snap73 := d7
						snap74 := d8
						snap75 := d9
						snap76 := d10
						snap77 := d11
						snap78 := d12
						snap79 := d13
						snap80 := d14
						snap81 := d16
						snap82 := d18
						snap83 := d20
						snap84 := d23
						snap85 := d24
						snap86 := d25
						snap87 := d26
						snap88 := d27
						snap89 := d28
						snap90 := d30
						snap91 := d31
						snap92 := d32
						snap93 := d33
						snap94 := d34
						snap95 := d35
						snap96 := d36
						snap97 := d37
						snap98 := d39
						snap99 := d41
						snap100 := d43
						snap101 := d44
						snap102 := d45
						snap103 := d46
						snap104 := d47
						snap105 := d48
						snap106 := d49
						snap107 := d50
						snap108 := d51
						snap109 := d52
						snap110 := d53
						snap111 := d54
						snap112 := d55
						snap113 := d56
						snap114 := d57
						snap115 := d58
						snap116 := d59
						snap117 := d60
						snap118 := d61
						snap119 := d62
						snap120 := d63
						snap121 := d64
						snap122 := d65
						snap123 := d66
						alloc124 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc124)
						d0 = snap67
						d1 = snap68
						d2 = snap69
						d3 = snap70
						d4 = snap71
						d5 = snap72
						d7 = snap73
						d8 = snap74
						d9 = snap75
						d10 = snap76
						d11 = snap77
						d12 = snap78
						d13 = snap79
						d14 = snap80
						d16 = snap81
						d18 = snap82
						d20 = snap83
						d23 = snap84
						d24 = snap85
						d25 = snap86
						d26 = snap87
						d27 = snap88
						d28 = snap89
						d30 = snap90
						d31 = snap91
						d32 = snap92
						d33 = snap93
						d34 = snap94
						d35 = snap95
						d36 = snap96
						d37 = snap97
						d39 = snap98
						d41 = snap99
						d43 = snap100
						d44 = snap101
						d45 = snap102
						d46 = snap103
						d47 = snap104
						d48 = snap105
						d49 = snap106
						d50 = snap107
						d51 = snap108
						d52 = snap109
						d53 = snap110
						d54 = snap111
						d55 = snap112
						d56 = snap113
						d57 = snap114
						d58 = snap115
						d59 = snap116
						d60 = snap117
						d61 = snap118
						d62 = snap119
						d63 = snap120
						d64 = snap121
						d65 = snap122
						d66 = snap123
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						d125 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d12.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d12.Reg)
						ctx.EmitMovRegReg(scratch, d12.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
						d125 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d125)
					}
					if d125.Loc == LocReg && d12.Loc == LocReg && d125.Reg == d12.Reg {
						ctx.TransferReg(d12.Reg)
						d12.Loc = LocNone
					}
					ctx.FreeDesc(&d12)
					ctx.EnsureDesc(&d125)
					ctx.EnsureDesc(&d125)
					ctx.EnsureDesc(&d125)
					d127 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.SyncDesc(&d125)
					ctx.SyncDesc(&d1)
					ctx.SyncDesc(&d127)
					d128 = d1
					d128.ID = 0
					d129 = d127
					d129.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d128, &d129, &d125, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d128, &d129, &d125, int32(16))
					}
					ctx.FreeDesc(&d129)
					d130 = args[0]
					d130.ID = 0
					ctx.SyncDesc(&d130)
					if d130.Loc == LocRegPair || d130.Loc == LocStackPair || d130.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d130, &result)
						result.Type = d130.Type
					} else {
						switch d130.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d130)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d130)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d130)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d130, &result)
							result.Type = d130.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					ctx.ReclaimUntrackedRegs()
					d131 = args[1]
					d131.ID = 0
					ctx.EnsureDesc(&d131)
					ctx.EnsureDesc(&d7)
					d132 = d131
					_ = d132
					d133 = d7
					_ = d133
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl15 := ctx.ReserveLabel()
					_ = lbl15
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d132 = JITPrepareScmerGoArg(ctx, d132)
					d133 = JITPrepareGoSliceArg(ctx, d133)
					if d133.Loc != LocRegTriple && d133.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d134 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d134.Loc == LocRegPair || d134.Loc == LocStackPair || d134.Loc == LocRegTriple || d134.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d132)
					ctx.SyncDesc(&d133)
					ctx.SyncDesc(&d134)
					d135 = ctx.EmitGoCallScalar(jitGoABI_4138fb0145801900, []JITValueDesc{d132, d133, d134}, 2)
					d135.NoHeapPointer = false
					ctx.BindReg(d135.Reg, &d135)
					ctx.BindReg(d135.Reg2, &d135)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d135)
					ctx.FreeDesc(&d131)
					d136 = args[0]
					d136.ID = 0
					ctx.SyncDesc(&d136)
					if d136.Loc == LocRegPair || d136.Loc == LocStackPair || d136.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d136, &result)
						result.Type = d136.Type
					} else {
						switch d136.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d136)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d136)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d136)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d136, &result)
							result.Type = d136.Type
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
			JITInlineCost:  63,
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
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
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
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 17}}, Count: 1})
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					ctx.ReclaimUntrackedRegs()
					d4 = args[0]
					d4.ID = 0
					if d4.Type == tagSlice {
						d5 = jitKnownSliceHeader(ctx, &d4)
					} else {
						d5 = ctx.EmitGoCallScalar(jitGoABI_ab569823a8d42a56, []JITValueDesc{d4}, 3)
					}
					if d5.Loc == LocRegTriple {
						ctx.BindReg(d5.Reg, &d5)
						ctx.BindReg(d5.Reg2, &d5)
						ctx.BindReg(d5.Reg3, &d5)
					}
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					d6 = args[2]
					d6.ID = 0
					if d6.Loc != LocImm {
						ctx.EnsureDesc(&d6)
					}
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
						boxed := JITPrepareScmerGoArg(ctx, d6)
						d7 = ctx.EmitGoCallScalar(jitGoABI_317743fd938661dd, []JITValueDesc{boxed}, 1)
						d7.Type = tagInt
						ctx.BindReg(d7.Reg, &d7)
					}
					ctx.FreeDesc(&d6)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					ctx.StabilizeDescForControlFlow(&d7)
					d5 = JITPrepareGoSliceArg(ctx, d5)
					if d5.Loc != LocRegTriple && d5.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ownedWindowSlots arg0)")
					}
					d9 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("window_flush")}
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d9.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d9.Imm)
						ptrWord, _ := d9.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d9.Imm.String())))
						d9 = tmpPair
					} else if d9.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d9.Type, Reg: ctx.AllocRegExcept(d9.Reg), Reg2: ctx.AllocRegExcept(d9.Reg)}
						switch d9.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d9)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d9)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d9)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d9)
						d9 = tmpPair
					}
					if d9.Loc != LocRegPair && d9.Loc != LocStackPair && d9.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ownedWindowSlots arg1)")
					}
					d10 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(": window must have at least 3 elements")}
					ctx.EnsureDesc(&d10)
					if d10.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d10.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d10.Imm)
						ptrWord, _ := d10.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d10.Imm.String())))
						d10 = tmpPair
					} else if d10.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d10.Type, Reg: ctx.AllocRegExcept(d10.Reg), Reg2: ctx.AllocRegExcept(d10.Reg)}
						switch d10.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d10)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d10)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d10)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d10)
						d10 = tmpPair
					}
					if d10.Loc != LocRegPair && d10.Loc != LocStackPair && d10.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (ownedWindowSlots arg2)")
					}
					ctx.SyncDesc(&d5)
					ctx.SyncDesc(&d9)
					ctx.SyncDesc(&d10)
					callResults11 := JITEmitGoCallResults(ctx, GoFuncAddr(ownedWindowSlots), []JITValueDesc{d5, d9, d10})
					ctx.FreeDesc(&d9)
					ctx.FreeDesc(&d10)
					d12 = callResults11[0]
					_ = d12
					d13 = callResults11[1]
					_ = d13
					ctx.StabilizeDescForControlFlow(&d12)
					ctx.StabilizeDescForControlFlow(&d13)
					if phiHomeOK2 {
						ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)})
					} else {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
					}
					return bbs[1].Render()
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if phiHomeOK2 && d3.Loc == LocReg {
						ctx.BindReg(r0, &d3)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDescsTogether(&d3, &d7)
					if d3.Loc == LocImm && d7.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() < d7.Imm.Int())}
					} else if d7.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d3.Reg)
						if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d3.Reg, int32(d7.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d7.Imm.Int()))
							ctx.EmitCmpInt64(d3.Reg, ctx.ScratchReg)
						}
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d14)
					} else if d3.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d3.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d7.Reg)
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d14)
					} else {
						r3 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitCmpInt64(d3.Reg, d7.Reg)
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d14)
					}
					ctx.FreeDesc(&d7)
					d15 = d14
					ctx.SyncDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d15.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d14)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap16 := d3
						snap17 := d4
						snap18 := d5
						snap19 := d6
						snap20 := d7
						snap21 := d8
						snap22 := d9
						snap23 := d10
						snap24 := d12
						snap25 := d13
						snap26 := d14
						snap27 := d15
						alloc28 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc28)
						d3 = snap16
						d4 = snap17
						d5 = snap18
						d6 = snap19
						d7 = snap20
						d8 = snap21
						d9 = snap22
						d10 = snap23
						d12 = snap24
						d13 = snap25
						d14 = snap26
						d15 = snap27
					}
					if !bbs[2].Rendered {
						return bbs[2].Render()
					}
					return result
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					ctx.ReclaimUntrackedRegs()
					d5 = JITPrepareGoSliceArg(ctx, d5)
					if d5.Loc != LocRegTriple && d5.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (shiftOwnedWindow arg0)")
					}
					d12 = JITPrepareGoSliceArg(ctx, d12)
					if d12.Loc != LocRegTriple && d12.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (shiftOwnedWindow arg1)")
					}
					if d13.Loc == LocRegPair || d13.Loc == LocStackPair || d13.Loc == LocRegTriple || d13.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d29 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					d29 = JITPrepareGoSliceArg(ctx, d29)
					if d29.Loc != LocRegTriple && d29.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (shiftOwnedWindow arg3)")
					}
					ctx.SyncDesc(&d5)
					ctx.SyncDesc(&d12)
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d29)
					ctx.EmitGoCallVoid(GoFuncAddr(shiftOwnedWindow), []JITValueDesc{d5, d12, d13, d29})
					ctx.FreeDesc(&d29)
					d30 = args[1]
					d30.ID = 0
					ctx.EnsureDesc(&d30)
					ctx.EnsureDesc(&d12)
					d31 = d30
					_ = d31
					d32 = d12
					_ = d32
					ctx.StabilizeDescForControlFlow(&d12)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl5 := ctx.ReserveLabel()
					_ = lbl5
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d31 = JITPrepareScmerGoArg(ctx, d31)
					d32 = JITPrepareGoSliceArg(ctx, d32)
					if d32.Loc != LocRegTriple && d32.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d33.Loc == LocRegPair || d33.Loc == LocStackPair || d33.Loc == LocRegTriple || d33.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d31)
					ctx.SyncDesc(&d32)
					ctx.SyncDesc(&d33)
					d34 = ctx.EmitGoCallScalar(jitGoABI_4138fb0145801900, []JITValueDesc{d31, d32, d33}, 2)
					d34.NoHeapPointer = false
					ctx.BindReg(d34.Reg, &d34)
					ctx.BindReg(d34.Reg2, &d34)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d34)
					ctx.FreeDesc(&d30)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK2 {
							scratch = r0
						} else {
							scratch = ctx.AllocRegExcept(d3.Reg)
						}
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d35)
					}
					if d35.Loc == LocReg && d3.Loc == LocReg && d35.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.SyncDesc(&d35)
					if d35.Loc == LocReg || d35.Loc == LocFPReg {
						ctx.ProtectReg(d35.Reg)
					} else if d35.Loc == LocRegPair {
						ctx.ProtectReg(d35.Reg)
						ctx.ProtectReg(d35.Reg2)
					}
					d36 = d35
					if d36.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d36)
					if phiHomeOK2 {
						ctx.EmitMovToReg(r0, d36)
					} else {
						ctx.EmitStoreToStack(d36, int32(bbs[1].PhiBase)+int32(0))
					}
					if d35.Loc == LocReg || d35.Loc == LocFPReg {
						ctx.UnprotectReg(d35.Reg)
					} else if d35.Loc == LocRegPair {
						ctx.UnprotectReg(d35.Reg)
						ctx.UnprotectReg(d35.Reg2)
					}
					return bbs[1].Render()
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					ctx.ReclaimUntrackedRegs()
					d37 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d37)
					if d37.Loc == LocRegPair || d37.Loc == LocStackPair || d37.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d37, &result)
						result.Type = d37.Type
					} else {
						switch d37.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d37)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d37)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d37)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d37, &result)
							result.Type = d37.Type
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
			JITInlineCost:  24,
		},
	})
}

// BEGIN jitgen prepared Go ABI entries
var jitGoABI_317743fd938661dd uint64
var jitGoABI_4138fb0145801900 uint64
var jitGoABI_79ac37f1e11427b6 uint64
var jitGoABI_ab569823a8d42a56 uint64

func init() {
	jitGoABI_317743fd938661dd = GoFuncAddr(Scmer.Int)
	jitGoABI_4138fb0145801900 = GoFuncAddr(ApplyEx)
	jitGoABI_79ac37f1e11427b6 = GoFuncAddr(jitCopyScmerSlice)
	jitGoABI_ab569823a8d42a56 = GoFuncAddr(jitAsSlice)
}
