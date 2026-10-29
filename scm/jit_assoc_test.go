// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
package scm

import "fmt"
import "runtime"
import "strings"
import "testing"

// Exercise generated bodies independently of the production profitability
// policy. RetainsCallArgs already forces that policy to choose the body; only
// this serial test changes the declaration, and it restores the real contract.
func TestGeneratedAssocEmitter(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires native JIT")
	}
	for _, name := range []string{"set_assoc", "set_assoc_mut"} {
		declaration := declarations[name]
		if declaration.Type.JITInlineCost == 65535 {
			t.Fatalf("%s has no generated body", name)
		}
		old := declaration.RetainsCallArgs
		declaration.RetainsCallArgs = true
		func() {
			defer func() { declaration.RetainsCallArgs = old }()
			for _, count := range []int{1, 4, 6} {
				var source strings.Builder
				fmt.Fprintf(&source, "(lambda (input value) (%s input", name)
				for i := 0; i < count; i++ {
					fmt.Fprintf(&source, " \"key-%d\" value", i)
				}
				source.WriteString("))")
				proc := jitCompile(preparedTestProc(t, source.String()))
				compiled := proc.Proc().compiled()
				if compiled == nil || compiled.Coverage.InlinedCalls == 0 || compiled.Coverage.NativeCalls != 0 {
					t.Fatalf("%s/%d did not use generated setter body", name, count)
				}
				for _, hashed := range []bool{false, true} {
					input := NewSlice([]Scmer{NewString("key-0"), NewInt(-1)})
					if hashed {
						input = benchmarkFastDict(8)
					}
					for iteration := 0; iteration < 3; iteration++ {
						result := Apply(proc, input, NewInt(int64(iteration+7)))
						for i := 0; i < count; i++ {
							if got := Apply(result, NewString(fmt.Sprintf("key-%d", i))); got.Int() != int64(iteration+7) {
								t.Fatalf("%s/%d hashed=%v: %s", name, count, hashed, String(result))
							}
						}
						if name == "set_assoc" {
							want := int64(-1)
							if hashed {
								want = 0
							}
							if Apply(input, NewString("key-0")).Int() != want {
								t.Fatal("immutable setter modified input")
							}
						}
					}
					pointerValue := NewString(strings.Repeat("pointer-bearing value", 20))
					result := Apply(proc, input, pointerValue)
					runtime.GC()
					for i := 0; i < count; i++ {
						if got := Apply(result, NewString(fmt.Sprintf("key-%d", i))); !Equal(got, pointerValue) {
							t.Fatalf("%s/%d lost pointer-bearing value across GC", name, count)
						}
					}
				}
			}
			// The optional callback is a runtime value, not a compiler-only
			// lambda template. Its structural arm must materialize the virtual
			// argument slice and preserve ordinary prepared-merger semantics.
			proc := jitCompile(preparedTestProc(t, "(lambda (input value merge) ("+name+" input \"key-0\" value merge))"))
			merger := NewFunc(func(args ...Scmer) Scmer { return NewInt(args[0].Int() + args[1].Int()) })
			input := NewSlice([]Scmer{NewString("key-0"), NewInt(3)})
			if got := Apply(Apply(proc, input, NewInt(7), merger), NewString("key-0")); got.Int() != 10 {
				t.Fatalf("%s runtime merger: %s", name, String(got))
			}
		}()
	}
}

func BenchmarkAssocGeneratedVsNativeBoundary(b *testing.B) {
	for _, force := range []bool{false, true} {
		b.Run(fmt.Sprint("emitter=", force), func(b *testing.B) {
			declaration := declarations["set_assoc_mut"]
			old := declaration.RetainsCallArgs
			declaration.RetainsCallArgs = force
			defer func() { declaration.RetainsCallArgs = old }()
			proc := Eval(Optimize(Read("assoc codegen probe", `(lambda (input value) (set_assoc_mut input "key-0" value))`), &Globalenv, nil), &Globalenv)
			if jitEnabled {
				proc = jitCompile(proc)
			}
			input := benchmarkFastDict(8)
			args := []Scmer{input, NewInt(7)}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				assocBenchmarkSink = Apply(proc, args...)
			}
		})
	}
}

// Measure compiler allocations separately from invocation cost. Parsing and
// optimization are outside the timed region; each iteration gets a fresh Proc.
func BenchmarkAssocCompilation(b *testing.B) {
	if !jitEnabled {
		b.Skip("requires native JIT")
	}
	for _, force := range []bool{false, true} {
		b.Run(fmt.Sprint("emitter=", force), func(b *testing.B) {
			declaration := declarations["set_assoc_mut"]
			old := declaration.RetainsCallArgs
			declaration.RetainsCallArgs = force
			defer func() { declaration.RetainsCallArgs = old }()
			expression := Optimize(Read("assoc compile", `(lambda (input value) (set_assoc_mut input "key-0" value))`), &Globalenv, nil)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compiled := jitCompile(Eval(expression, &Globalenv))
				if compiled.Proc().compiled() == nil {
					b.Fatal("native compilation failed")
				}
				assocBenchmarkSink = compiled
			}
		})
	}
}
