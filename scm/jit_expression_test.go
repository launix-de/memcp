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

import "testing"

func compileJITExpressionTestProc(t *testing.T, source string) Scmer {
	t.Helper()
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	expression := Optimize(Read(t.Name(), source), &Globalenv, nil)
	proc := Eval(expression, &Globalenv)
	compiled := jitCompile(proc)
	if compiled.GetTag() != tagProc || compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatalf("expression did not compile: %s", SerializeToString(expression, &Globalenv))
	}
	return compiled
}

func requireNoDynamicJITCalls(t *testing.T, compiled Scmer) {
	t.Helper()
	coverage := compiled.Proc().Compiled.Coverage
	if coverage.DynamicCalls != 0 {
		t.Fatalf("expected complete expression lowering, got %+v", coverage)
	}
}

func TestJITExpressionBeginDefine(t *testing.T) {
	proc := &Proc{
		Params: NewSlice([]Scmer{NewSymbol("x")}),
		Body: NewSlice([]Scmer{
			NewSymbol("begin"),
			NewSlice([]Scmer{
				NewSymbol("define"), NewSymbol("y"),
				NewSlice([]Scmer{NewSymbol("+"), NewNthLocalVar(0), NewInt(1)}),
			}),
			NewSymbol("y"),
		}),
		NumVars: 1,
	}
	compiled := jitCompile(NewProcStruct(*proc))
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("begin/define expression did not compile")
	}
	requireNoDynamicJITCalls(t, compiled)
	if got := Apply(compiled, NewInt(41)); !Equal(got, NewInt(42)) {
		t.Fatalf("unexpected begin/define result: %s", String(got))
	}
}

func TestJITExpressionListForms(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		compiled := compileJITExpressionTestProc(t, `(lambda (a b) (list a b))`)
		requireNoDynamicJITCalls(t, compiled)
		got := Apply(compiled, NewInt(1), NewInt(2))
		want := NewSlice([]Scmer{NewInt(1), NewInt(2)})
		if !Equal(got, want) {
			t.Fatalf("unexpected list result: %s", String(got))
		}
	})

	t.Run("!list", func(t *testing.T) {
		proc := Proc{
			Params: NewSlice([]Scmer{NewSymbol("a"), NewSymbol("b")}),
			Body: NewSlice([]Scmer{
				NewSymbol("nth"),
				NewSlice([]Scmer{NewSymbol("!list"), NewNthLocalVar(2), NewInt(2), NewNthLocalVar(0), NewNthLocalVar(1)}),
				NewInt(1),
			}),
			NumVars: 4,
		}
		if !jitEnabled {
			t.Skip("requires GOEXPERIMENT=jit")
		}
		compiled := jitCompile(NewProcStruct(proc))
		if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
			t.Fatal("!list expression did not compile")
		}
		requireNoDynamicJITCalls(t, compiled)
		if got := Apply(compiled, NewInt(1), NewInt(2)); !Equal(got, NewInt(2)) {
			t.Fatalf("unexpected !list result: %s", String(got))
		}
	})

	t.Run("!!list", func(t *testing.T) {
		compiled := compileJITExpressionTestProc(t, `(lambda () (!!list 4))`)
		requireNoDynamicJITCalls(t, compiled)
		got := Apply(compiled).Slice()
		if len(got) != 0 || cap(got) != 4 {
			t.Fatalf("unexpected !!list shape: len=%d cap=%d", len(got), cap(got))
		}
	})
}

func TestJITExpressionReduceLambdaArgumentOrder(t *testing.T) {
	compiled := compileJITExpressionTestProc(t,
		`(lambda (items initial) (reduce items (lambda (acc item) (list acc item)) initial))`)
	if coverage := compiled.Proc().Compiled.Coverage; coverage.DynamicCalls != 1 {
		t.Fatalf("expected reduce to use one generic callback fallback, got %+v", coverage)
	}
	got := Apply(compiled,
		NewSlice([]Scmer{NewString("a"), NewString("b")}),
		NewString("initial"))
	want := NewSlice([]Scmer{
		NewSlice([]Scmer{NewString("initial"), NewString("a")}),
		NewString("b"),
	})
	if !Equal(got, want) {
		t.Fatalf("unexpected reduce result: got %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionHigherOrderClosureCapture(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (values blocked) (begin
		(define blocked_set (reduce blocked (lambda (acc item) (set_assoc acc item true)) '()))
		(filter values (lambda (item) (not (has_assoc? blocked_set item))))))`)
	got := Apply(compiled,
		NewSlice([]Scmer{NewString("a"), NewString("b"), NewString("c")}),
		NewSlice([]Scmer{NewString("b")}))
	want := NewSlice([]Scmer{NewString("a"), NewString("c")})
	if !Equal(got, want) {
		t.Fatalf("unexpected captured higher-order result: got %s, want %s", String(got), String(want))
	}
}
