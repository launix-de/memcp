/*
Copyright (C) 2026  Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
*/
package scm

import (
	"runtime"
	"testing"
)

func growJITCallbackStack(depth int) byte {
	var frame [512]byte
	frame[0] = byte(depth)
	if depth == 0 {
		return frame[0]
	}
	result := frame[0] ^ growJITCallbackStack(depth-1)
	runtime.KeepAlive(frame)
	return result
}

func compileJITCallbackBenchmark(b *testing.B, source string, inline bool) Scmer {
	b.Helper()
	old := declarations["map"].Type.JITInlineCallbacks
	declarations["map"].Type.JITInlineCallbacks = inline
	defer func() { declarations["map"].Type.JITInlineCallbacks = old }()
	proc := Eval(Optimize(Read(b.Name(), source), &Globalenv, nil), &Globalenv)
	compiled := jitCompile(proc)
	if compiled.GetTag() != tagProc || compiled.Proc().Compiled == nil {
		b.Fatal("callback benchmark did not compile")
	}
	return compiled
}

func BenchmarkJITKnownMapCallback(b *testing.B) {
	values := make([]Scmer, 128)
	for i := range values {
		values[i] = NewInt(int64(i))
	}
	input := NewSlice(values)
	const source = `(lambda (values) (map values (lambda (value) (+ value 1))))`
	for _, tc := range []struct {
		name   string
		inline bool
	}{
		{"dynamic_callback", false},
		{"inlined_callback", true},
	} {
		b.Run(tc.name, func(b *testing.B) {
			compiled := compileJITCallbackBenchmark(b, source, tc.inline)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = Apply(compiled, input)
			}
		})
	}
}

func compileJITRuntimeReduceBenchmark(tb testing.TB) Scmer {
	tb.Helper()
	if !jitEnabled {
		tb.Skip("requires GOEXPERIMENT=jit")
	}
	declaration := declarations["reduce"]
	previous := declaration.RetainsCallArgs
	declaration.RetainsCallArgs = true
	defer func() { declaration.RetainsCallArgs = previous }()
	proc := Eval(Optimize(Read(tb.Name(), `(lambda (callback values) (reduce values callback 0))`), &Globalenv, nil), &Globalenv)
	compiled := jitCompile(proc)
	if compiled.GetTag() != tagProc || compiled.Proc().Compiled == nil {
		tb.Fatal("runtime callback benchmark did not compile")
	}
	return compiled
}

func TestJITRuntimeReduceCallbacksUseDirectBoundary(t *testing.T) {
	compiled := compileJITRuntimeReduceBenchmark(t)
	if coverage := compiled.Proc().Compiled.Coverage; coverage.InlinedCalls == 0 || coverage.DynamicCalls == 0 {
		t.Fatalf("runtime callback did not reach the generated direct boundary: %+v", coverage)
	}
	values := NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3), NewInt(4)})
	callbacks := []Scmer{
		jitCompile(Eval(Read(t.Name(), `(lambda (acc value) (+ acc value))`), &Globalenv)),
		NewFunc(func(args ...Scmer) Scmer {
			runtime.GC()
			_ = growJITCallbackStack(64)
			return NewInt(args[0].Int() + args[1].Int())
		}),
	}
	for _, callback := range callbacks {
		if got := Apply(compiled, callback, values); !Equal(got, NewInt(10)) {
			t.Fatalf("runtime reduce callback returned %s, want 10", String(got))
		}
	}
}

func TestJITRuntimeReduceCallbacksPreserveLiveValues(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	proc := Eval(Optimize(Read(t.Name(), `(lambda (callback values a b c d e f) (+ a b c d e f (reduce values callback 0)))`), &Globalenv, nil), &Globalenv)
	compiled := jitCompile(proc)
	if compiled.GetTag() != tagProc || compiled.Proc().Compiled == nil {
		t.Fatal("runtime callback live-value test did not compile")
	}
	values := NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3), NewInt(4)})
	callbacks := []Scmer{
		jitCompile(Eval(Read(t.Name(), `(lambda (acc value) (+ acc value))`), &Globalenv)),
		NewFunc(func(args ...Scmer) Scmer {
			runtime.GC()
			_ = growJITCallbackStack(64)
			return NewInt(args[0].Int() + args[1].Int())
		}),
	}
	for _, callback := range callbacks {
		got := Apply(compiled, callback, values, NewInt(10), NewInt(20), NewInt(30), NewInt(40), NewInt(50), NewInt(60))
		if !Equal(got, NewInt(220)) {
			t.Fatalf("runtime callback with live values returned %s, want 220", String(got))
		}
	}
}

func BenchmarkJITRuntimeReduceCallback(b *testing.B) {
	compiled := compileJITRuntimeReduceBenchmark(b)
	valuesSlice := make([]Scmer, 128)
	for index := range valuesSlice {
		valuesSlice[index] = NewInt(int64(index))
	}
	values := NewSlice(valuesSlice)
	callbacks := []struct {
		name  string
		value Scmer
	}{
		{"jit_proc", jitCompile(Eval(Read(b.Name(), `(lambda (acc value) (+ acc value))`), &Globalenv))},
		{"go_func", NewFunc(func(args ...Scmer) Scmer { return NewInt(args[0].Int() + args[1].Int()) })},
	}
	for _, callback := range callbacks {
		b.Run(callback.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for index := 0; index < b.N; index++ {
				jitListBenchmarkSink = Apply(compiled, callback.value, values)
			}
		})
	}
}
