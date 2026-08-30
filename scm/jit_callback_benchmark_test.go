/*
Copyright (C) 2026  Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
*/
package scm

import "testing"

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
