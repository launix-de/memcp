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

 window_flush shifts in count positions of nils, honoring any remaining leading skip.
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

// The leading skip belongs to the window, including shifts performed while
// flushing its tail. Padding advances the window but is not an emitted row.
func skipOwnedWindowEmission(win []Scmer) bool {
	skip := win[0].Int()
	if skip <= 0 {
		return false
	}
	win[0] = NewInt(skip - 1)
	return true
}

func windowMut(a ...Scmer) Scmer {
	win := asSlice(a[0], "window_mut")
	vals := asSlice(a[2], "window_mut vals")
	slots, stride := ownedWindowSlots(win, "window_mut", ": window must have at least 3 elements (skip, counter, stride)")
	shiftOwnedWindow(win, slots, stride, vals)
	if skipOwnedWindowEmission(win) {
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
		if skipOwnedWindowEmission(win) {
			continue
		}
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
				var d11 JITValueDesc
				_ = d11
				var d13 JITValueDesc
				_ = d13
				var d15 JITValueDesc
				_ = d15
				var phiBase17 int32
				_ = phiBase17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
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
				var d34 JITValueDesc
				_ = d34
				var d36 JITValueDesc
				_ = d36
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
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
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
				var d76 JITValueDesc
				_ = d76
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
				var d148 JITValueDesc
				_ = d148
				var d149 JITValueDesc
				_ = d149
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
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d3)
					d9 = d1
					_ = d9
					ctx.StabilizeDescForControlFlow(&d9)
					inlineHome10 := d9
					inlineHome10.ID = 0
					d11 = d7
					_ = d11
					ctx.StabilizeDescForControlFlow(&d11)
					inlineHome12 := d11
					inlineHome12.ID = 0
					d13 = d8
					_ = d13
					ctx.StabilizeDescForControlFlow(&d13)
					inlineHome14 := d13
					inlineHome14.ID = 0
					d15 = d3
					_ = d15
					ctx.StabilizeDescForControlFlow(&d15)
					inlineHome16 := d15
					inlineHome16.ID = 0
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.StabilizeDescForControlFlow(&d7)
					phiBase17 = ctx.AllocStack(int32(16))
					d18 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
					_ = d18
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
					d9 = inlineHome10
					d11 = inlineHome12
					d13 = inlineHome14
					d15 = inlineHome16
					d18 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocRegPair || d11.Loc == LocRegTriple {
						d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d11.Reg2}
						ctx.BindReg(d11.Reg2, &d19)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDescsTogether(&d19, &d13)
					if d19.Loc == LocImm && d13.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d19.Imm.Int() - d13.Imm.Int())}
					} else {
						ctx.ProtectDescRegisters(&d19)
						ctx.ProtectDescRegisters(&d13)
						r0 := ctx.AllocReg()
						if d19.Loc == LocImm {
							ctx.EmitMovRegImm64(r0, uint64(d19.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r0, d19.Reg)
						}
						if d13.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d13.Imm.Int()))
							ctx.EmitSubInt64(r0, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r0, d13.Reg)
						}
						ctx.UnprotectDescRegisters(&d13)
						ctx.UnprotectDescRegisters(&d19)
						d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
						ctx.BindReg(r0, &d21)
					}
					r1 := ctx.EmitSliceDataAfterLow(&d11, &d13, 16)
					d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
					ctx.BindReg(r1, &d22)
					ctx.BindReg(r1, &d22)
					var r2 Reg
					var r3 Reg
					ctx.SyncDesc(&d22)
					ctx.EnsureDesc(&d22)
					if d22.Loc == LocImm {
						r2 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r2, uint64(d22.Imm.Int()))
					} else {
						r2 = d22.Reg
					}
					ctx.ProtectReg(r2)
					ctx.SyncDesc(&d21)
					ctx.EnsureDesc(&d21)
					if d21.Loc == LocImm {
						r3 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r3, uint64(d21.Imm.Int()))
					} else {
						r3 = d21.Reg
					}
					ctx.ProtectReg(r3)
					r4 := ctx.EmitSliceCapAfterLow(&d11, &d13, r2, r3)
					ctx.UnprotectReg(r3)
					ctx.UnprotectReg(r2)
					d23 = JITValueDesc{Loc: LocRegTriple, Reg: r2, Reg2: r3, Reg3: r4}
					ctx.BindReg(r2, &d23)
					ctx.BindReg(r3, &d23)
					ctx.BindReg(r4, &d23)
					ctx.BindReg(r2, &d23)
					ctx.BindReg(r3, &d23)
					ctx.BindReg(r4, &d23)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d23)
					callResults24 := JITEmitGoCallResults(ctx, jitGoABI_79ac37f1e11427b6, []JITValueDesc{d11, d23})
					d25 = callResults24[0]
					d25.Type = tagInt
					ctx.ReclaimUntrackedRegs()
					if d11.SliceSizeKnown {
						d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d11.KnownSliceLen))}
					} else if d11.Loc == LocImm {
						d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d11.StackOff))}
					} else if d11.Loc == LocStackTriple {
						d26 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d11.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d11)
						if d11.Loc == LocRegPair || d11.Loc == LocRegTriple {
							d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d11.Reg2, ID: 0}
						} else if d11.Loc == LocReg {
							d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d11.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d26
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d26)
					ctx.EnsureDesc(&d13)
					ctx.SyncDesc(&d26)
					ctx.SyncDesc(&d13)
					if d26.Loc == LocImm && d13.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d26.Imm.Int() - d13.Imm.Int())}
					} else if d13.Loc == LocImm && d13.Imm.Int() == 0 {
						ctx.EnsureDesc(&d26)
						r5 := ctx.AllocRegExcept(d26.Reg)
						ctx.EmitMovRegReg(r5, d26.Reg)
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d27)
					} else if d26.Loc == LocImm {
						ctx.EnsureDesc(&d13)
						scratch := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d26.Imm.Int()))
						ctx.EmitIntBinary(JITIntSub, 64, scratch, &d13)
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d27)
					} else if d13.Loc == LocImm {
						ctx.EnsureDesc(&d26)
						scratch := ctx.AllocRegExcept(d26.Reg)
						ctx.EmitMovRegReg(scratch, d26.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, d13.Imm.Int())
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d27)
					} else {
						ctx.EnsureDesc(&d26)
						ctx.SyncDesc(&d13)
						r6 := ctx.AllocRegExceptOperand(&d13, d26.Reg)
						ctx.EmitMovRegReg(r6, d26.Reg)
						ctx.EmitIntBinary(JITIntSub, 64, r6, &d13)
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
						ctx.BindReg(r6, &d27)
					}
					if d27.Loc == LocReg && d26.Loc == LocReg && d27.Reg == d26.Reg {
						ctx.TransferReg(d26.Reg)
						d26.Loc = LocNone
					}
					ctx.FreeDesc(&d26)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocRegPair || d11.Loc == LocRegTriple {
						d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d11.Reg2}
						ctx.BindReg(d11.Reg2, &d28)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d28)
					ctx.EnsureDescsTogether(&d28, &d27)
					if d28.Loc == LocImm && d27.Loc == LocImm {
						d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d28.Imm.Int() - d27.Imm.Int())}
					} else {
						ctx.ProtectDescRegisters(&d28)
						ctx.ProtectDescRegisters(&d27)
						r7 := ctx.AllocReg()
						if d28.Loc == LocImm {
							ctx.EmitMovRegImm64(r7, uint64(d28.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r7, d28.Reg)
						}
						if d27.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d27.Imm.Int()))
							ctx.EmitSubInt64(r7, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r7, d27.Reg)
						}
						ctx.UnprotectDescRegisters(&d27)
						ctx.UnprotectDescRegisters(&d28)
						d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
						ctx.BindReg(r7, &d30)
					}
					r8 := ctx.EmitSliceDataAfterLow(&d11, &d27, 16)
					d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
					ctx.BindReg(r8, &d31)
					ctx.BindReg(r8, &d31)
					var r9 Reg
					var r10 Reg
					ctx.SyncDesc(&d31)
					ctx.EnsureDesc(&d31)
					if d31.Loc == LocImm {
						r9 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r9, uint64(d31.Imm.Int()))
					} else {
						r9 = d31.Reg
					}
					ctx.ProtectReg(r9)
					ctx.SyncDesc(&d30)
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocImm {
						r10 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r10, uint64(d30.Imm.Int()))
					} else {
						r10 = d30.Reg
					}
					ctx.ProtectReg(r10)
					r11 := ctx.EmitSliceCapAfterLow(&d11, &d27, r9, r10)
					ctx.UnprotectReg(r10)
					ctx.UnprotectReg(r9)
					d32 = JITValueDesc{Loc: LocRegTriple, Reg: r9, Reg2: r10, Reg3: r11}
					ctx.BindReg(r9, &d32)
					ctx.BindReg(r10, &d32)
					ctx.BindReg(r11, &d32)
					ctx.BindReg(r9, &d32)
					ctx.BindReg(r10, &d32)
					ctx.BindReg(r11, &d32)
					ctx.StabilizeDescForControlFlow(&d32)
					inlineHome33 := d32
					inlineHome33.ID = 0
					ctx.FreeDesc(&d27)
					ctx.ReclaimUntrackedRegs()
					if d32.SliceSizeKnown {
						d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d32.KnownSliceLen))}
					} else if d32.Loc == LocImm {
						d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d32.StackOff))}
					} else if d32.Loc == LocStackTriple {
						d34 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d32.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d32)
						if d32.Loc == LocRegPair || d32.Loc == LocRegTriple {
							d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg2, ID: 0}
						} else if d32.Loc == LocReg {
							d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d34
					ctx.StabilizeDescForControlFlow(&d34)
					inlineHome35 := d34
					inlineHome35.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(phiBase17)+int32(0))
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d9 = inlineHome10
					d11 = inlineHome12
					d13 = inlineHome14
					d15 = inlineHome16
					d32 = inlineHome33
					d34 = inlineHome35
					d18 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d18.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitMovRegReg(scratch, d18.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d36)
					}
					if d36.Loc == LocReg && d18.Loc == LocReg && d36.Reg == d18.Reg {
						ctx.TransferReg(d18.Reg)
						d18.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d36)
					inlineHome37 := d36
					inlineHome37.ID = 0
					ctx.FreeDesc(&d18)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d36)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDescsTogether(&d36, &d34)
					if d36.Loc == LocImm && d34.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d36.Imm.Int() < d34.Imm.Int())}
					} else if d34.Loc == LocImm {
						r12 := ctx.AllocRegExcept(d36.Reg)
						if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d36.Reg, int32(d34.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d34.Imm.Int()))
							ctx.EmitCmpInt64(d36.Reg, ctx.ScratchReg)
						}
						d38 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedLess}
						ctx.BindReg(r12, &d38)
					} else if d36.Loc == LocImm {
						r13 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d36.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d34.Reg)
						d38 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r13, Condition: CondSignedLess}
						ctx.BindReg(r13, &d38)
					} else {
						r14 := ctx.AllocRegExcept(d36.Reg)
						ctx.EmitCmpInt64(d36.Reg, d34.Reg)
						d38 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r14, Condition: CondSignedLess}
						ctx.BindReg(r14, &d38)
					}
					ctx.ReclaimUntrackedRegs()
					d39 = d38
					ctx.SyncDesc(&d39)
					if d39.Loc != LocImm && d39.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d39.Loc == LocImm {
						if d39.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d39.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d38)
						ctx.MarkLabel(lbl11)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d9 = inlineHome10
					d11 = inlineHome12
					d13 = inlineHome14
					d15 = inlineHome16
					d32 = inlineHome33
					d34 = inlineHome35
					d36 = inlineHome37
					d18 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.ReclaimUntrackedRegs()
					d42 = ctx.EmitSliceElementAddress(&d9, &d40, 16)
					ctx.EnsureDesc(&d42)
					r15 := ctx.AllocRegExcept(d42.Reg)
					ctx.EmitMovRegMem(r15, d42.Reg, 8)
					ctx.EmitMovRegMem(d42.Reg, d42.Reg, 0)
					d41 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d42.Reg, Reg2: r15}
					ctx.BindReg(d42.Reg, &d41)
					ctx.BindReg(r15, &d41)
					ctx.ReclaimUntrackedRegs()
					if d41.Loc != LocImm {
						ctx.EnsureDesc(&d41)
					}
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
						boxed := JITPrepareScmerGoArg(ctx, d41)
						d43 = ctx.EmitGoCallScalar(jitGoABI_317743fd938661dd, []JITValueDesc{boxed}, 1)
						d43.Type = tagInt
						ctx.BindReg(d43.Reg, &d43)
					}
					ctx.FreeDesc(&d41)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d43)
					ctx.EnsureDesc(&d43)
					if d43.Loc == LocImm {
						d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d43.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d43.Reg)
						ctx.EmitMovRegReg(scratch, d43.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d44)
					}
					if d44.Loc == LocReg && d43.Loc == LocReg && d44.Reg == d43.Reg {
						ctx.TransferReg(d43.Reg)
						d43.Loc = LocNone
					}
					ctx.FreeDesc(&d43)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d44)
					ctx.ReclaimUntrackedRegs()
					d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d44)
					ctx.SyncDesc(&d9)
					ctx.SyncDesc(&d45)
					d46 = d9
					d46.ID = 0
					d47 = d45
					d47.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d46, &d47, &d44, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d46, &d47, &d44, int32(16))
					}
					ctx.FreeDesc(&d47)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl4)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d9 = inlineHome10
					d11 = inlineHome12
					d13 = inlineHome14
					d15 = inlineHome16
					d32 = inlineHome33
					d34 = inlineHome35
					d36 = inlineHome37
					d18 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d15.SliceSizeKnown {
						d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d15.KnownSliceLen))}
					} else if d15.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d15.StackOff))}
					} else if d15.Loc == LocStackTriple {
						d48 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d15.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d15)
						if d15.Loc == LocRegPair || d15.Loc == LocRegTriple {
							d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d15.Reg2, ID: 0}
						} else if d15.Loc == LocReg {
							d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d15.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d48
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d36)
					ctx.EnsureDesc(&d48)
					ctx.EnsureDescsTogether(&d36, &d48)
					if d36.Loc == LocImm && d48.Loc == LocImm {
						d49 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d36.Imm.Int() < d48.Imm.Int())}
					} else if d48.Loc == LocImm {
						r16 := ctx.AllocRegExcept(d36.Reg)
						if d48.Imm.Int() >= -2147483648 && d48.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d36.Reg, int32(d48.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d48.Imm.Int()))
							ctx.EmitCmpInt64(d36.Reg, ctx.ScratchReg)
						}
						d49 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondSignedLess}
						ctx.BindReg(r16, &d49)
					} else if d36.Loc == LocImm {
						r17 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d36.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d48.Reg)
						d49 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondSignedLess}
						ctx.BindReg(r17, &d49)
					} else {
						r18 := ctx.AllocRegExcept(d36.Reg)
						ctx.EmitCmpInt64(d36.Reg, d48.Reg)
						d49 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r18, Condition: CondSignedLess}
						ctx.BindReg(r18, &d49)
					}
					ctx.FreeDesc(&d48)
					ctx.ReclaimUntrackedRegs()
					d50 = d49
					ctx.SyncDesc(&d50)
					if d50.Loc != LocImm && d50.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					if d50.Loc == LocImm {
						if d50.Imm.Bool() {
							ctx.MarkLabel(lbl13)
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl10)
						}
					} else {
						ctx.EmitJump(d50.Condition, lbl13)
						ctx.EmitJmp(lbl14)
						ctx.FreeDesc(&d49)
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl10)
					}
					bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d9 = inlineHome10
					d11 = inlineHome12
					d13 = inlineHome14
					d15 = inlineHome16
					d32 = inlineHome33
					d34 = inlineHome35
					d36 = inlineHome37
					d18 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d51 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d36)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d51)
					ctx.SyncDesc(&d32)
					ctx.SyncDesc(&d36)
					d52 = d32
					d52.ID = 0
					d53 = d36
					d53.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d52, &d53, &d51, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d36)
						d53 = d36
						d53.ID = 0
						ctx.EmitStoreScmerSliceElement(&d52, &d53, &d51, int32(16))
					}
					ctx.FreeDesc(&d53)
					ctx.FreeDesc(&d51)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d36)
					if d36.Loc == LocReg || d36.Loc == LocFPReg {
						ctx.ProtectReg(d36.Reg)
					} else if d36.Loc == LocRegPair {
						ctx.ProtectReg(d36.Reg)
						ctx.ProtectReg(d36.Reg2)
					}
					d54 = d36
					if d54.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d54)
					ctx.EmitStoreToStack(d54, int32(phiBase17)+int32(0))
					if d36.Loc == LocReg || d36.Loc == LocFPReg {
						ctx.UnprotectReg(d36.Reg)
					} else if d36.Loc == LocRegPair {
						ctx.UnprotectReg(d36.Reg)
						ctx.UnprotectReg(d36.Reg2)
					}
					ctx.EmitJmp(lbl6)
					bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d9 = inlineHome10
					d11 = inlineHome12
					d13 = inlineHome14
					d15 = inlineHome16
					d32 = inlineHome33
					d34 = inlineHome35
					d36 = inlineHome37
					d18 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase17) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d36)
					ctx.ReclaimUntrackedRegs()
					d56 = ctx.EmitSliceElementAddress(&d15, &d36, 16)
					ctx.EnsureDesc(&d56)
					r19 := ctx.AllocRegExcept(d56.Reg)
					ctx.EmitMovRegMem(r19, d56.Reg, 8)
					ctx.EmitMovRegMem(d56.Reg, d56.Reg, 0)
					d55 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d56.Reg, Reg2: r19}
					ctx.BindReg(d56.Reg, &d55)
					ctx.BindReg(r19, &d55)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d36)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d55)
					ctx.SyncDesc(&d32)
					ctx.SyncDesc(&d36)
					d57 = d32
					d57.ID = 0
					d58 = d36
					d58.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d57, &d58, &d55, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d36)
						d58 = d36
						d58.ID = 0
						ctx.EmitStoreScmerSliceElement(&d57, &d58, &d55, int32(16))
					}
					ctx.FreeDesc(&d58)
					ctx.FreeDesc(&d55)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d36)
					if d36.Loc == LocReg || d36.Loc == LocFPReg {
						ctx.ProtectReg(d36.Reg)
					} else if d36.Loc == LocRegPair {
						ctx.ProtectReg(d36.Reg)
						ctx.ProtectReg(d36.Reg2)
					}
					d59 = d36
					if d59.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d59)
					ctx.EmitStoreToStack(d59, int32(phiBase17)+int32(0))
					if d36.Loc == LocReg || d36.Loc == LocFPReg {
						ctx.UnprotectReg(d36.Reg)
					} else if d36.Loc == LocRegPair {
						ctx.UnprotectReg(d36.Reg)
						ctx.UnprotectReg(d36.Reg2)
					}
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl4)
					ctx.FreeDesc(&d8)
					d7 = ctx.StabilizeForNested(d7)
					ctx.EnsureDesc(&d1)
					d60 = d1
					_ = d60
					ctx.StabilizeDescForControlFlow(&d60)
					inlineHome61 := d60
					inlineHome61.ID = 0
					lbl15 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl16 := ctx.ReserveLabel()
					_ = lbl16
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					lbl17 := ctx.ReserveLabel()
					_ = lbl17
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					lbl18 := ctx.ReserveLabel()
					_ = lbl18
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					d60 = inlineHome61
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d62 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					d64 = ctx.EmitSliceElementAddress(&d60, &d62, 16)
					ctx.EnsureDesc(&d64)
					r20 := ctx.AllocRegExcept(d64.Reg)
					ctx.EmitMovRegMem(r20, d64.Reg, 8)
					ctx.EmitMovRegMem(d64.Reg, d64.Reg, 0)
					d63 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d64.Reg, Reg2: r20}
					ctx.BindReg(d64.Reg, &d63)
					ctx.BindReg(r20, &d63)
					ctx.ReclaimUntrackedRegs()
					if d63.Loc != LocImm {
						ctx.EnsureDesc(&d63)
					}
					if d63.Loc == LocImm {
						d65 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d63.Imm.Int())}
					} else if d63.Type == tagInt && d63.Loc == LocRegPair {
						ctx.FreeReg(d63.Reg)
						d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d63.Reg2}
						ctx.BindReg(d63.Reg2, &d65)
						ctx.BindReg(d63.Reg2, &d65)
					} else if d63.Type == tagInt && d63.Loc == LocReg {
						d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d63.Reg}
						ctx.BindReg(d63.Reg, &d65)
						ctx.BindReg(d63.Reg, &d65)
					} else {
						boxed := JITPrepareScmerGoArg(ctx, d63)
						d65 = ctx.EmitGoCallScalar(jitGoABI_317743fd938661dd, []JITValueDesc{boxed}, 1)
						d65.Type = tagInt
						ctx.BindReg(d65.Reg, &d65)
					}
					ctx.StabilizeDescForControlFlow(&d65)
					inlineHome66 := d65
					inlineHome66.ID = 0
					ctx.FreeDesc(&d63)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocImm {
						d67 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d65.Imm.Int() <= 0)}
					} else {
						r21 := ctx.AllocRegExcept(d65.Reg)
						ctx.EmitCmpRegImm32(d65.Reg, 0)
						d67 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r21, Condition: CondSignedLessOrEqual}
						ctx.BindReg(r21, &d67)
					}
					ctx.ReclaimUntrackedRegs()
					d68 = d67
					ctx.SyncDesc(&d68)
					if d68.Loc != LocImm && d68.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					if d68.Loc == LocImm {
						if d68.Imm.Bool() {
							ctx.MarkLabel(lbl19)
							ctx.EmitJmp(lbl17)
						} else {
							ctx.MarkLabel(lbl20)
							ctx.EmitJmp(lbl18)
						}
					} else {
						ctx.EmitJump(d68.Condition, lbl19)
						ctx.EmitJmp(lbl20)
						ctx.FreeDesc(&d67)
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl17)
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl18)
					}
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					d60 = inlineHome61
					d65 = inlineHome66
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocImm {
						d69 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d65.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d65.Reg)
						ctx.EmitMovRegReg(scratch, d65.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
						d69 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d69)
					}
					if d69.Loc == LocReg && d65.Loc == LocReg && d69.Reg == d65.Reg {
						ctx.TransferReg(d65.Reg)
						d65.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d69)
					ctx.ReclaimUntrackedRegs()
					d70 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d69)
					ctx.SyncDesc(&d60)
					ctx.SyncDesc(&d70)
					d71 = d60
					d71.ID = 0
					d72 = d70
					d72.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d71, &d72, &d69, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d71, &d72, &d69, int32(16))
					}
					ctx.FreeDesc(&d72)
					ctx.ReclaimUntrackedRegs()
					r22 := ctx.AllocReg()
					d73 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r22, d73)
					}
					ctx.EmitJmp(lbl15)
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					d60 = inlineHome61
					d65 = inlineHome66
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d74 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ctx.EnsureDesc(&d74)
					if d74.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r22, d74)
					}
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl15)
					d75 = JITValueDesc{Loc: LocReg, Reg: r22}
					ctx.BindReg(r22, &d75)
					ctx.BindReg(r22, &d75)
					d76 = d75
					ctx.EnsureDesc(&d76)
					if d76.Loc != LocImm && d76.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d76.Loc == LocImm {
						if d76.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d76.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap77 := d0
						snap78 := d1
						snap79 := d2
						snap80 := d3
						snap81 := d4
						snap82 := d5
						snap83 := d7
						snap84 := d8
						snap85 := d9
						snap86 := d11
						snap87 := d13
						snap88 := d15
						snap89 := d18
						snap90 := d19
						snap91 := d20
						snap92 := d21
						snap93 := d22
						snap94 := d23
						snap95 := d25
						snap96 := d26
						snap97 := d27
						snap98 := d28
						snap99 := d29
						snap100 := d30
						snap101 := d31
						snap102 := d32
						snap103 := d34
						snap104 := d36
						snap105 := d38
						snap106 := d39
						snap107 := d40
						snap108 := d41
						snap109 := d42
						snap110 := d43
						snap111 := d44
						snap112 := d45
						snap113 := d46
						snap114 := d47
						snap115 := d48
						snap116 := d49
						snap117 := d50
						snap118 := d51
						snap119 := d52
						snap120 := d53
						snap121 := d54
						snap122 := d55
						snap123 := d56
						snap124 := d57
						snap125 := d58
						snap126 := d59
						snap127 := d60
						snap128 := d62
						snap129 := d63
						snap130 := d64
						snap131 := d65
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
						alloc142 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc142)
						d0 = snap77
						d1 = snap78
						d2 = snap79
						d3 = snap80
						d4 = snap81
						d5 = snap82
						d7 = snap83
						d8 = snap84
						d9 = snap85
						d11 = snap86
						d13 = snap87
						d15 = snap88
						d18 = snap89
						d19 = snap90
						d20 = snap91
						d21 = snap92
						d22 = snap93
						d23 = snap94
						d25 = snap95
						d26 = snap96
						d27 = snap97
						d28 = snap98
						d29 = snap99
						d30 = snap100
						d31 = snap101
						d32 = snap102
						d34 = snap103
						d36 = snap104
						d38 = snap105
						d39 = snap106
						d40 = snap107
						d41 = snap108
						d42 = snap109
						d43 = snap110
						d44 = snap111
						d45 = snap112
						d46 = snap113
						d47 = snap114
						d48 = snap115
						d49 = snap116
						d50 = snap117
						d51 = snap118
						d52 = snap119
						d53 = snap120
						d54 = snap121
						d55 = snap122
						d56 = snap123
						d57 = snap124
						d58 = snap125
						d59 = snap126
						d60 = snap127
						d62 = snap128
						d63 = snap129
						d64 = snap130
						d65 = snap131
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
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d75)
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
					d143 = args[0]
					d143.ID = 0
					ctx.SyncDesc(&d143)
					if d143.Loc == LocRegPair || d143.Loc == LocStackPair || d143.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d143, &result)
						result.Type = d143.Type
					} else {
						switch d143.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d143)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d143)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d143)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d143, &result)
							result.Type = d143.Type
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
					d144 = args[1]
					d144.ID = 0
					ctx.EnsureDesc(&d144)
					ctx.EnsureDesc(&d7)
					d145 = d144
					_ = d145
					d146 = d7
					_ = d146
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl21 := ctx.ReserveLabel()
					_ = lbl21
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl21)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d145 = JITPrepareScmerGoArg(ctx, d145)
					d146 = JITPrepareGoSliceArg(ctx, d146)
					if d146.Loc != LocRegTriple && d146.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d147 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d147.Loc == LocRegPair || d147.Loc == LocStackPair || d147.Loc == LocRegTriple || d147.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d145)
					ctx.SyncDesc(&d146)
					ctx.SyncDesc(&d147)
					d148 = ctx.EmitGoCallScalar(jitGoABI_4138fb0145801900, []JITValueDesc{d145, d146, d147}, 2)
					d148.NoHeapPointer = false
					ctx.BindReg(d148.Reg, &d148)
					ctx.BindReg(d148.Reg2, &d148)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d148)
					ctx.FreeDesc(&d144)
					d149 = args[0]
					d149.ID = 0
					ctx.SyncDesc(&d149)
					if d149.Loc == LocRegPair || d149.Loc == LocStackPair || d149.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d149, &result)
						result.Type = d149.Type
					} else {
						switch d149.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d149)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d149)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d149)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d149, &result)
							result.Type = d149.Type
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
			JITInlineCost:  65,
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
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
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
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
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
				var bbs [6]BBDescriptor
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
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
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
						return bbs[4].Render()
					}
					ctx.EmitJump(d15.Condition, lbl3)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d14)
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
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
						bbs[4].Render()
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
					d7 = ctx.StabilizeForNested(d7)
					d12 = ctx.StabilizeForNested(d12)
					d13 = ctx.StabilizeForNested(d13)
					d14 = ctx.StabilizeForNested(d14)
					ctx.EnsureDesc(&d5)
					d30 = d5
					_ = d30
					ctx.StabilizeDescForControlFlow(&d30)
					inlineHome31 := d30
					inlineHome31.ID = 0
					ctx.StabilizeDescForControlFlow(&d5)
					lbl7 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d30 = inlineHome31
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					d34 = ctx.EmitSliceElementAddress(&d30, &d32, 16)
					ctx.EnsureDesc(&d34)
					r4 := ctx.AllocRegExcept(d34.Reg)
					ctx.EmitMovRegMem(r4, d34.Reg, 8)
					ctx.EmitMovRegMem(d34.Reg, d34.Reg, 0)
					d33 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d34.Reg, Reg2: r4}
					ctx.BindReg(d34.Reg, &d33)
					ctx.BindReg(r4, &d33)
					ctx.ReclaimUntrackedRegs()
					if d33.Loc != LocImm {
						ctx.EnsureDesc(&d33)
					}
					if d33.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d33.Imm.Int())}
					} else if d33.Type == tagInt && d33.Loc == LocRegPair {
						ctx.FreeReg(d33.Reg)
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d33.Reg2}
						ctx.BindReg(d33.Reg2, &d35)
						ctx.BindReg(d33.Reg2, &d35)
					} else if d33.Type == tagInt && d33.Loc == LocReg {
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d33.Reg}
						ctx.BindReg(d33.Reg, &d35)
						ctx.BindReg(d33.Reg, &d35)
					} else {
						boxed := JITPrepareScmerGoArg(ctx, d33)
						d35 = ctx.EmitGoCallScalar(jitGoABI_317743fd938661dd, []JITValueDesc{boxed}, 1)
						d35.Type = tagInt
						ctx.BindReg(d35.Reg, &d35)
					}
					ctx.StabilizeDescForControlFlow(&d35)
					inlineHome36 := d35
					inlineHome36.ID = 0
					ctx.FreeDesc(&d33)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d35)
					if d35.Loc == LocImm {
						d37 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d35.Imm.Int() <= 0)}
					} else {
						r5 := ctx.AllocRegExcept(d35.Reg)
						ctx.EmitCmpRegImm32(d35.Reg, 0)
						d37 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLessOrEqual}
						ctx.BindReg(r5, &d37)
					}
					ctx.ReclaimUntrackedRegs()
					d38 = d37
					ctx.SyncDesc(&d38)
					if d38.Loc != LocImm && d38.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					if d38.Loc == LocImm {
						if d38.Imm.Bool() {
							ctx.MarkLabel(lbl11)
							ctx.EmitJmp(lbl9)
						} else {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl10)
						}
					} else {
						ctx.EmitJump(d38.Condition, lbl11)
						ctx.EmitJmp(lbl12)
						ctx.FreeDesc(&d37)
						ctx.MarkLabel(lbl11)
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl10)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d30 = inlineHome31
					d35 = inlineHome36
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d35)
					ctx.EnsureDesc(&d35)
					if d35.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d35.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d35.Reg)
						ctx.EmitMovRegReg(scratch, d35.Reg)
						ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d39)
					}
					if d39.Loc == LocReg && d35.Loc == LocReg && d39.Reg == d35.Reg {
						ctx.TransferReg(d35.Reg)
						d35.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d39)
					ctx.ReclaimUntrackedRegs()
					d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d39)
					ctx.SyncDesc(&d30)
					ctx.SyncDesc(&d40)
					d41 = d30
					d41.ID = 0
					d42 = d40
					d42.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d41, &d42, &d39, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d41, &d42, &d39, int32(16))
					}
					ctx.FreeDesc(&d42)
					ctx.ReclaimUntrackedRegs()
					r6 := ctx.AllocReg()
					d43 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					ctx.EnsureDesc(&d43)
					if d43.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r6, d43)
					}
					ctx.EmitJmp(lbl7)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d30 = inlineHome31
					d35 = inlineHome36
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d44 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ctx.EnsureDesc(&d44)
					if d44.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r6, d44)
					}
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl7)
					d45 = JITValueDesc{Loc: LocReg, Reg: r6}
					ctx.BindReg(r6, &d45)
					ctx.BindReg(r6, &d45)
					d46 = d45
					ctx.EnsureDesc(&d46)
					if d46.Loc != LocImm && d46.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d46.Loc == LocImm {
						if d46.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d46.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap47 := d3
						snap48 := d4
						snap49 := d5
						snap50 := d6
						snap51 := d7
						snap52 := d8
						snap53 := d9
						snap54 := d10
						snap55 := d12
						snap56 := d13
						snap57 := d14
						snap58 := d15
						snap59 := d29
						snap60 := d30
						snap61 := d32
						snap62 := d33
						snap63 := d34
						snap64 := d35
						snap65 := d37
						snap66 := d38
						snap67 := d39
						snap68 := d40
						snap69 := d41
						snap70 := d42
						snap71 := d43
						snap72 := d44
						snap73 := d45
						snap74 := d46
						alloc75 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc75)
						d3 = snap47
						d4 = snap48
						d5 = snap49
						d6 = snap50
						d7 = snap51
						d8 = snap52
						d9 = snap53
						d10 = snap54
						d12 = snap55
						d13 = snap56
						d14 = snap57
						d15 = snap58
						d29 = snap59
						d30 = snap60
						d32 = snap61
						d33 = snap62
						d34 = snap63
						d35 = snap64
						d37 = snap65
						d38 = snap66
						d39 = snap67
						d40 = snap68
						d41 = snap69
						d42 = snap70
						d43 = snap71
						d44 = snap72
						d45 = snap73
						d46 = snap74
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d45)
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
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						d76 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK2 {
							scratch = r0
						} else {
							scratch = ctx.AllocRegExcept(d3.Reg)
						}
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d76 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d76)
					}
					if d76.Loc == LocReg && d3.Loc == LocReg && d76.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.SyncDesc(&d76)
					if d76.Loc == LocReg || d76.Loc == LocFPReg {
						ctx.ProtectReg(d76.Reg)
					} else if d76.Loc == LocRegPair {
						ctx.ProtectReg(d76.Reg)
						ctx.ProtectReg(d76.Reg2)
					}
					d77 = d76
					if d77.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d77)
					if phiHomeOK2 {
						ctx.EmitMovToReg(r0, d77)
					} else {
						ctx.EmitStoreToStack(d77, int32(bbs[1].PhiBase)+int32(0))
					}
					if d76.Loc == LocReg || d76.Loc == LocFPReg {
						ctx.UnprotectReg(d76.Reg)
					} else if d76.Loc == LocRegPair {
						ctx.UnprotectReg(d76.Reg)
						ctx.UnprotectReg(d76.Reg2)
					}
					return bbs[1].Render()
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					ctx.ReclaimUntrackedRegs()
					d78 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d78)
					if d78.Loc == LocRegPair || d78.Loc == LocStackPair || d78.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d78, &result)
						result.Type = d78.Type
					} else {
						switch d78.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d78)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d78)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d78)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d78, &result)
							result.Type = d78.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					ctx.ReclaimUntrackedRegs()
					d79 = args[1]
					d79.ID = 0
					ctx.EnsureDesc(&d79)
					ctx.EnsureDesc(&d12)
					d80 = d79
					_ = d80
					d81 = d12
					_ = d81
					ctx.StabilizeDescForControlFlow(&d12)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl13 := ctx.ReserveLabel()
					_ = lbl13
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d80 = JITPrepareScmerGoArg(ctx, d80)
					d81 = JITPrepareGoSliceArg(ctx, d81)
					if d81.Loc != LocRegTriple && d81.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d82 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d82.Loc == LocRegPair || d82.Loc == LocStackPair || d82.Loc == LocRegTriple || d82.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d80)
					ctx.SyncDesc(&d81)
					ctx.SyncDesc(&d82)
					d83 = ctx.EmitGoCallScalar(jitGoABI_4138fb0145801900, []JITValueDesc{d80, d81, d82}, 2)
					d83.NoHeapPointer = false
					ctx.BindReg(d83.Reg, &d83)
					ctx.BindReg(d83.Reg2, &d83)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d83)
					ctx.FreeDesc(&d79)
					return bbs[3].Render()
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
			JITInlineCost:  38,
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
