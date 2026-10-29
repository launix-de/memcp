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

func groupedAssocTestProc(t testing.TB) Scmer {
	t.Helper()
	proc := preparedTestProc(t, `(lambda (rows)
		(reduce rows (lambda (dict row)
			(set_assoc dict (list (nth row 0)) (list (nth row 1) 1)
				(lambda (old new) (list (+ (nth old 0) (nth new 0)) (+ (nth old 1) (nth new 1))))))
			(lambda () (list))))`)
	if jitEnabled {
		proc = jitCompile(proc)
		if proc.Proc().compiled() == nil {
			t.Fatal("grouped reducer did not compile")
		}
	}
	return proc
}

func groupedAssocTestRows() Scmer {
	rows := make([]Scmer, 256)
	for i := range rows {
		rows[i] = NewSlice([]Scmer{NewInt(int64(i % 64)), NewInt(int64(i))})
	}
	return NewSlice(rows)
}

func checkGroupedAssocResult(t testing.TB, result Scmer) {
	t.Helper()
	for key := int64(0); key < 64; key++ {
		got := Apply(result, NewSlice([]Scmer{NewInt(key)}))
		want := NewSlice([]Scmer{NewInt(4*key + 384), NewInt(4)})
		if !Equal(got, want) {
			t.Fatalf("group %d: got %s, want %s", key, String(got), String(want))
		}
	}
}

func TestGroupedAssocReducerKnownMerger(t *testing.T) {
	proc := groupedAssocTestProc(t)
	rows := groupedAssocTestRows()
	first := Apply(proc, rows)
	checkGroupedAssocResult(t, first)
	checkGroupedAssocResult(t, Apply(proc, rows))
	runtime.GC()
	checkGroupedAssocResult(t, first)
}

func BenchmarkGroupedAssocReducer(b *testing.B) {
	proc := groupedAssocTestProc(b)
	call := PrepareSerialProc(proc)
	args := []Scmer{groupedAssocTestRows()}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		assocBenchmarkSink = call.Call(args)
	}
	b.StopTimer()
	checkGroupedAssocResult(b, assocBenchmarkSink)
}

// FastDict iteration invokes a nested Go closure. Its appended result header
// must stay the same cell across the iterator and callback inline scopes.
func TestJITExtractAssocCapturedResultHeader(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires JIT")
	}
	proc := preparedTestProc(t, `(lambda (dict) (extract_assoc dict (lambda (key value) (list key value))))`)
	proc = jitCompile(proc)
	if proc.Proc().compiled() == nil {
		t.Fatal("extractor did not compile")
	}
	for _, count := range []int{0, 1, 8, 32} {
		fd := NewFastDictValue(count)
		var pairs, want []Scmer
		for i := 0; i < count; i++ {
			key, value := NewString(fmt.Sprint(i)), NewString(strings.Repeat(fmt.Sprint(i), 20))
			fd.Set(key, value, nil)
			pairs = append(pairs, key, value)
			want = append(want, NewSlice([]Scmer{key, value}))
		}
		for _, input := range []Scmer{NewSlice(pairs), NewFastDict(fd)} {
			got := Apply(proc, input)
			if !Equal(got, NewSlice(want)) {
				t.Fatalf("count=%d input=%d got=%s want=%s", count, input.GetTag(), String(got), String(NewSlice(want)))
			}
			runtime.GC()
			if !Equal(got, NewSlice(want)) {
				t.Fatal("extracted values lost across GC")
			}
		}
	}
}

func TestJITExtractAssocIdentityValues(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires JIT")
	}
	for _, callback := range []string{`(lambda (key value) value)`, `(lambda (key value) key)`, `(lambda (key value) nil)`} {
		proc := jitCompile(preparedTestProc(t, `(lambda (dict) (extract_assoc dict `+callback+`))`))
		fd := NewFastDictValue(16)
		for i := 0; i < 16; i++ {
			fd.Set(NewString(fmt.Sprint(i)), NewSlice([]Scmer{NewInt(int64(i)), NewString("value")}), nil)
		}
		for _, input := range []Scmer{NewFastDict(fd), NewSlice(fd.Pairs)} {
			native := Apply(Globalenv.Vars["extract_assoc"], input, Eval(Optimize(Read("callback", callback), &Globalenv, nil), &Globalenv))
			got := Apply(proc, input)
			if !Equal(got, native) {
				t.Fatalf("%s input=%d got %s want %s", callback, input.GetTag(), String(got), String(native))
			}
		}
	}
}
