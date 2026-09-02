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

import "runtime"
import "testing"

func preparedTestProc(t testing.TB, source string) Scmer {
	t.Helper()
	return Eval(Optimize(Read("serial proc test", source), &Globalenv, nil), &Globalenv)
}

func TestPrepareSerialProcShapes(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		kind     SerialProcKind
		argument int
		want     int64
	}{
		{name: "constant", source: "(lambda (value) 7)", kind: SerialProcConstant, argument: -1, want: 7},
		{name: "constant true", source: "(lambda () true)", kind: SerialProcConstant, argument: -1, want: 1},
		{name: "identity", source: "(lambda (value) value)", kind: SerialProcArgument, argument: 0, want: 11},
		{name: "second argument", source: "(lambda (acc value) value)", kind: SerialProcArgument, argument: 1, want: 13},
		{name: "native forwarding", source: "(lambda (acc value) (+ acc value))", kind: SerialProcNative, argument: -1, want: 24},
		{name: "native forwarding preserves argument order", source: "(lambda (left right) (- left right))", kind: SerialProcNative, argument: -1, want: -2},
		{name: "native argument constant", source: "(lambda (value) (equal?? value 11))", kind: SerialProcNativeArgConstant, argument: 0, want: 1},
		{name: "native constant argument", source: "(lambda (value) (< 10 value))", kind: SerialProcNativeArgConstant, argument: 0, want: 1},
		{name: "native arithmetic argument constant", source: "(lambda (value) (+ value 1))", kind: SerialProcNativeArgConstant, argument: 0, want: 12},
		{name: "general", source: "(lambda (value) (+ (* value 2) 1))", kind: SerialProcGeneral, argument: -1, want: 23},
	}
	args := []Scmer{NewInt(11), NewInt(13)}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := preparedTestProc(t, test.source)
			prepared := PrepareSerialProc(source)
			if prepared.Kind != test.kind || int(prepared.Argument) != test.argument {
				t.Fatalf("shape = (%d, %d), want (%d, %d); source=%s body=%s", prepared.Kind, prepared.Argument, test.kind, test.argument, source.String(), source.Proc().Body.String())
			}
			if got := int64(ToInt(prepared.Call(args))); got != test.want {
				t.Fatalf("result = %d, want %d", got, test.want)
			}
		})
	}
}

func TestPrepareSerialProcRecognizesDirectNative(t *testing.T) {
	prepared := PrepareSerialProc(Globalenv.Vars[Symbol("+")])
	if !prepared.IsNative(Symbol("+")) {
		t.Fatal("direct + callback was not recognized")
	}
	if got := prepared.Call([]Scmer{NewInt(3), NewInt(4)}).Int(); got != 7 {
		t.Fatalf("direct native result = %d, want 7", got)
	}
}

func TestPrepareSerialProcKeepsCompiledEntryPointAuthoritative(t *testing.T) {
	source := preparedTestProc(t, "(lambda () 7)")
	source.Proc().Compiled = &JITEntryPoint{Native: func(...Scmer) Scmer { return NewInt(99) }}
	prepared := PrepareSerialProc(source)
	if !jitEnabled {
		if prepared.Kind != SerialProcGeneral {
			t.Fatalf("compiled procedure shape = %d, want general without JIT runtime", prepared.Kind)
		}
		return
	}
	if prepared.Kind != SerialProcJIT {
		t.Fatalf("compiled procedure shape = %d, want direct JIT dispatch", prepared.Kind)
	}
	if got := prepared.Call(nil); !Equal(got, NewInt(99)) {
		t.Fatalf("direct JIT result = %s, want 99", String(got))
	}
}

func TestPrepareSerialProcUsesCompiledJITDirectly(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	source := preparedTestProc(t, "(lambda (acc value) (+ acc value))")
	compiled := jitCompile(source)
	if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("map-reducer did not compile")
	}
	prepared := PrepareSerialProc(compiled)
	if prepared.Kind != SerialProcJIT {
		t.Fatalf("compiled map-reducer shape = %d, want direct JIT dispatch", prepared.Kind)
	}
	if got := prepared.Call([]Scmer{NewInt(40), NewInt(2)}); !Equal(got, NewInt(42)) {
		t.Fatalf("direct JIT map-reducer result = %s, want 42", String(got))
	}
}

func BenchmarkSerialProcJITMapReducerDispatch(b *testing.B) {
	if !jitEnabled {
		b.Skip("requires GOEXPERIMENT=jit")
	}
	source := preparedTestProc(b, "(lambda (acc value) (+ acc value))")
	compiled := jitCompile(source)
	if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		b.Fatal("map-reducer did not compile")
	}
	direct := PrepareSerialProc(compiled)
	adapter := SerialProc{
		Kind:     SerialProcGeneral,
		borrowed: optimizeProcToSerialBorrowed(compiled),
	}
	b.Run("general_adapter", func(b *testing.B) {
		args := []Scmer{NewInt(1), NewInt(1)}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			args[0] = adapter.Call(args)
		}
	})
	b.Run("old_go_frame_bridge", func(b *testing.B) {
		args := []Scmer{NewInt(1), NewInt(1)}
		jitFn := direct.Function
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			args[0] = benchmarkJITGoFrameBridge(jitFn, args)
		}
		runtime.KeepAlive(&direct)
	})
	b.Run("direct_jit_entry", func(b *testing.B) {
		args := []Scmer{NewInt(1), NewInt(1)}
		jitFn := direct.Function
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			args[0] = jitFn(args...)
		}
		runtime.KeepAlive(&direct)
	})
}

//go:noinline
func benchmarkJITGoFrameBridge(native func(...Scmer) Scmer, args []Scmer) Scmer {
	result := native(args...)
	runtime.KeepAlive(native)
	return result
}

func TestPrepareSerialProcNativeForwardMatchesInterpreterAdapter(t *testing.T) {
	source := preparedTestProc(t, "(lambda (acc row) (append acc row))")
	empty := Eval(Read("serial proc test", "'()"), &Globalenv)
	row := NewSlice([]Scmer{NewInt(1)})
	baseline := OptimizeProcToSerialFunction(source)(empty, row)
	prepared := PrepareSerialProc(source)
	got := prepared.Call([]Scmer{empty, row})
	if !Equal(got, baseline) {
		t.Fatalf("native forwarding result = %s, adapter = %s; body=%s", got.String(), baseline.String(), source.Proc().Body.String())
	}
}

func TestPrepareSerialProcDoesNotReuseRetainedCallArguments(t *testing.T) {
	source := preparedTestProc(t, "(lambda (value) (list value))")
	prepared := PrepareSerialProc(source)
	if prepared.Kind != SerialProcGeneral {
		t.Fatalf("retaining native shape = %d, want general adapter", prepared.Kind)
	}
	args := []Scmer{NewInt(1)}
	first := prepared.Call(args)
	args[0] = NewInt(2)
	second := prepared.Call(args)
	if got := first.Slice()[0].Int(); got != 1 {
		t.Fatalf("first retained result changed to %d", got)
	}
	if got := second.Slice()[0].Int(); got != 2 {
		t.Fatalf("second retained result = %d, want 2", got)
	}
}

func TestPrepareSerialProcReusesNonCapturingBeginScope(t *testing.T) {
	source := preparedTestProc(t, "(lambda (value) (begin (define doubled (+ value value)) (+ doubled 1)))")
	prepared := PrepareSerialProc(source)
	args := []Scmer{NewInt(11)}
	if got := prepared.Call(args).Int(); got != 23 {
		t.Fatalf("first begin result = %d, want 23", got)
	}
	args[0] = NewInt(4)
	if got := prepared.Call(args).Int(); got != 9 {
		t.Fatalf("reused begin result = %d, want 9", got)
	}
	if allocations := testing.AllocsPerRun(100, func() {
		prepared.Call(args)
	}); allocations > 6 {
		t.Fatalf("non-capturing begin allocated %.2f objects per call; body=%s", allocations, source.Proc().Body.String())
	}
}

func TestSerialExprDetectsCapturedBeginScope(t *testing.T) {
	source := preparedTestProc(t, "(lambda (value) (begin (define captured value) (lambda () captured)))")
	if !serialExprMayCaptureEnv(source.Proc().Body) {
		t.Fatalf("capturing begin was classified as reusable: %s", source.Proc().Body.String())
	}
	dynamic := preparedTestProc(t, "(lambda (value) (begin (define captured value) (eval '(lambda () captured))))")
	if !serialExprMayCaptureEnv(dynamic.Proc().Body) {
		t.Fatalf("eval-created closure was classified as reusable: %s", dynamic.Proc().Body.String())
	}
}

func TestPrepareSerialProcTraversesExplicitOuterScope(t *testing.T) {
	factory := preparedTestProc(t, "(lambda (captured) (lambda (value) (+ value captured)))")
	callback := Apply(factory, NewInt(7))
	prepared := PrepareSerialProc(callback)
	if got := prepared.Call([]Scmer{NewInt(5)}).Int(); got != 12 {
		t.Fatalf("prepared outer-scope result = %d, want 12; body=%s", got, callback.Proc().Body.String())
	}
}

func TestEqualProceduresUsesIdentity(t *testing.T) {
	first := Eval(Read("procedure equality test", "(lambda (x) x)"), &Globalenv)
	second := Eval(Read("procedure equality test", "(lambda (x) (+ x 1))"), &Globalenv)
	if !Equal(first, first) {
		t.Fatal("a procedure must equal itself")
	}
	if Equal(first, second) {
		t.Fatal("distinct procedures must not compare equal through their printed representation")
	}
}
