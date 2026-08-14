/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

func TestOptimizeProcToSerialFunctionUsesNumberedFixedParams(t *testing.T) {
	lambda := Eval(Optimize(Read("test", "(lambda (value) (+ value value))"), &Globalenv, nil), &Globalenv)
	if !lambda.Proc().NumberedOnly {
		t.Fatal("expected optimized lambda to use numbered bindings only")
	}
	got := OptimizeProcToSerialFunction(lambda)(NewInt(6))
	if ToInt(got) != 12 {
		t.Fatalf("expected numbered callback result 12, got %v", got)
	}
}

func TestOptimizeProcToSerialFunctionUsesNumberedVariadicParam(t *testing.T) {
	lambda := NewProcStruct(Proc{
		Params:       NewSymbol("values"),
		Body:         NewSlice([]Scmer{NewSymbol("list"), NewNthLocalVar(0)}),
		En:           &Globalenv,
		NumVars:      1,
		NumberedOnly: true,
	})
	got := OptimizeProcToSerialFunction(lambda)(NewInt(3), NewInt(4))
	want := NewSlice([]Scmer{NewSlice([]Scmer{NewInt(3), NewInt(4)})})
	if !Equal(got, want) {
		t.Fatalf("expected numbered variadic callback result %v, got %v", want, got)
	}
}

func TestOptimizeProcToSerialFunctionExplicitNumVarsKeepsNamedParamBinding(t *testing.T) {
	lambda := Eval(Read("test", "(lambda ($update) ($update) 1)"), &Globalenv)
	called := false
	update := NewFunc(func(args ...Scmer) Scmer {
		called = true
		return NewInt(7)
	})
	got := OptimizeProcToSerialFunction(lambda)(update)
	if !called {
		t.Fatal("expected explicit-numvars callback to invoke bound parameter")
	}
	if ToInt(got) != 7 {
		t.Fatalf("expected callback result 7, got %v", got)
	}
}

func TestOptimizeProcToSerialFunctionExplicitNumVarsKeepsNamedVariadicBinding(t *testing.T) {
	lambda := NewProcStruct(Proc{
		Params:  NewSymbol("values"),
		Body:    NewSymbol("values"),
		En:      &Globalenv,
		NumVars: 1,
	})
	got := OptimizeProcToSerialFunction(lambda)(NewInt(3), NewInt(4))
	want := NewSlice([]Scmer{NewInt(3), NewInt(4)})
	if !Equal(got, want) {
		t.Fatalf("expected named variadic callback result %v, got %v", want, got)
	}
}

func BenchmarkOptimizeProcToSerialFunctionNumberedAdapter(b *testing.B) {
	body := NewNthLocalVar(0)
	for i := 0; i < 256; i++ {
		body = NewSlice([]Scmer{NewSymbol("+"), body, NewInt(1)})
	}
	proc := NewProcStruct(Proc{
		Params:       NewSlice([]Scmer{NewSymbol("value")}),
		Body:         body,
		En:           &Globalenv,
		NumVars:      1,
		NumberedOnly: true,
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		OptimizeProcToSerialFunction(proc)
	}
}
