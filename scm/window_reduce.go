/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

// The functional entry points preserve a borrowed window. Its fixed-size
// slots, rather than emitted rows, are copied before callbacks can run. The
// optimizer may transfer an exclusively owned window to the matching private
// implementation; both return (next-window, consumer-accumulator).
func copyWindowReductionArguments(a []Scmer, name string) []Scmer {
	if len(a) != 5 {
		panic(name + " expects window, emitter, operand, reducer and accumulator")
	}
	window := asSlice(a[0], name)
	ownedWindowSlots(window, name, ": window must have at least 3 elements")
	owned := append([]Scmer(nil), a...)
	owned[0] = NewSlice(append([]Scmer(nil), window...))
	return owned
}

func windowReduce(a ...Scmer) Scmer {
	return windowMutReduce(copyWindowReductionArguments(a, "window_reduce")...)
}

func windowFlushReduce(a ...Scmer) Scmer {
	return windowFlushMutReduce(copyWindowReductionArguments(a, "window_flush_reduce")...)
}

func windowReductionResult(window, accumulator Scmer) Scmer {
	return NewSlice([]Scmer{window, accumulator})
}

func windowEmissionResult(emitter, reducer, accumulator Scmer, slots []Scmer) Scmer {
	// ApplyOnce gives escaping closures their own frame and copies retaining
	// native argument arrays. The next window never aliases a returned frame.
	value := ApplyOnce(emitter, slots)
	if reducer.IsNil() {
		return accumulator
	}
	var args [2]Scmer
	args[0], args[1] = accumulator, value
	return ApplyOnce(reducer, args[:])
}

func windowMutReduce(a ...Scmer) Scmer {
	if len(a) != 5 {
		panic("window_mut_reduce expects window, emitter, values, reducer and accumulator")
	}
	win := asSlice(a[0], "window_mut_reduce")
	vals := asSlice(a[2], "window_mut_reduce vals")
	slots, stride := ownedWindowSlots(win, "window_mut_reduce", ": window must have at least 3 elements")
	shiftOwnedWindow(win, slots, stride, vals)
	if skipOwnedWindowEmission(win) {
		return windowReductionResult(a[0], a[4])
	}
	return windowReductionResult(a[0], windowEmissionResult(a[1], a[3], a[4], slots))
}

func windowFlushMutReduce(a ...Scmer) Scmer {
	if len(a) != 5 {
		panic("window_flush_mut_reduce expects window, emitter, count, reducer and accumulator")
	}
	win := asSlice(a[0], "window_flush_mut_reduce")
	count := int(a[2].Int())
	slots, stride := ownedWindowSlots(win, "window_flush_mut_reduce", ": window must have at least 3 elements")
	if count <= 0 {
		return windowReductionResult(a[0], a[4])
	}
	if count == 1 {
		shiftOwnedWindow(win, slots, stride, nil)
		if skipOwnedWindowEmission(win) {
			return windowReductionResult(a[0], a[4])
		}
		return windowReductionResult(a[0], windowEmissionResult(a[1], a[3], a[4], slots))
	}
	emitter := PrepareSerialProc(a[1])
	result := a[4]
	if a[3].IsNil() {
		for n := 0; n < count; n++ {
			shiftOwnedWindow(win, slots, stride, nil)
			if skipOwnedWindowEmission(win) {
				continue
			}
			emitter.Call(slots)
		}
		return windowReductionResult(a[0], result)
	}
	reducer := PrepareSerialProc(a[3])
	var args [2]Scmer
	for n := 0; n < count; n++ {
		shiftOwnedWindow(win, slots, stride, nil)
		if skipOwnedWindowEmission(win) {
			continue
		}
		args[0], args[1] = result, emitter.Call(slots)
		result = reducer.Call(args[:])
	}
	return windowReductionResult(a[0], result)
}

func windowReductionType(name, operand, operandKind string, mutating bool) *TypeDescriptor {
	return &TypeDescriptor{Kind: "func", Description: "Return next sliding window and reduced emitter result; nil reducer retains effect-only emission", HasSideEffects: true, Forbidden: mutating,
		Params: []*TypeDescriptor{
			{Kind: "list", Label: "window"},
			{Kind: "func", Label: "emitter", SameGoroutine: true, NoEscape: true, Params: []*TypeDescriptor{{Kind: "any", Label: "values", Variadic: true}}, Return: &TypeDescriptor{Kind: "any"}},
			{Kind: operandKind, Label: operand},
			{Kind: "func|nil", Label: "reducer", SameGoroutine: true, NoEscape: true, Params: []*TypeDescriptor{{Kind: "any", Label: "accumulator"}, {Kind: "any", Label: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
			{Kind: "any", Label: "accumulator"},
		},
		// Only the new outer window is owned. Its scalar payloads and the
		// arbitrary callback result receive no invented ownership guarantee.
		Return: &TypeDescriptor{Kind: "list", Length: 2, Transfer: true, Keys: map[string]*TypeDescriptor{
			"0": {Kind: "list", Transfer: true}, "1": {Kind: "any"},
		}},
		JITInlineCost: 65535,
		JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			ctx.Coverage.NativeCalls++
			return jitEmitGeneratedCallBoundary(ctx, declarations[name], sourceArgs, args, result)
		},
		JITVirtualArgs: true,
	}
}

func initWindowReducers() {
	for _, spec := range []struct {
		name, mutName, operand, operandKind string
		functional, mutating                func(...Scmer) Scmer
	}{
		{"window_reduce", "window_mut_reduce", "values", "list", windowReduce, windowMutReduce},
		{"window_flush_reduce", "window_flush_mut_reduce", "count", "number", windowFlushReduce, windowFlushMutReduce},
	} {
		name := spec.name
		Declare(&Globalenv, &Declaration{
			Name: spec.name, Fn: spec.functional,
			Type:                     windowReductionType(spec.name, spec.operand, spec.operandKind, false),
			Optimize:                 FirstParameterMutable(spec.mutName),
			OptimizeFirstArgTransfer: true,
		})
		Declare(&Globalenv, &Declaration{
			Name: spec.mutName, Fn: spec.mutating, OptimizerOnly: true,
			Type: windowReductionType(spec.mutName, spec.operand, spec.operandKind, true),
			// A transported/persisted optimized call returns to functional source;
			// the receiver proves its own ownership before selecting mutation.
			SerializeCall: func(arguments []Scmer) (Scmer, error) {
				return NewSlice(append([]Scmer{NewSymbol(name)}, arguments...)), nil
			},
		})
	}
}
