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

func BenchmarkEvalOptimizedNativeCall(b *testing.B) {
	environment := &Env{Vars: Vars{Symbol("x"): NewInt(1)}, Outer: &Globalenv}
	expression := Optimize(Read(b.Name(), "(+ x 2)"), environment, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Eval(expression, environment)
	}
}

func BenchmarkEvalOptimizedSpecialForm(b *testing.B) {
	environment := &Env{Vars: Vars{Symbol("x"): NewBool(true)}, Outer: &Globalenv}
	expression := Optimize(Read(b.Name(), "(if x (+ 1 2) 0)"), environment, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Eval(expression, environment)
	}
}

func benchmarkOptimizedForm(b *testing.B, source string, vars Vars) {
	b.Helper()
	environment := &Env{Vars: vars, Outer: &Globalenv}
	expression := Optimize(Read(b.Name(), source), environment, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Eval(expression, environment)
	}
}

func BenchmarkEvalOptimizedEval(b *testing.B) {
	benchmarkOptimizedForm(b, "(eval code)", Vars{Symbol("code"): NewInt(7)})
}

func BenchmarkEvalOptimizedIf(b *testing.B) {
	benchmarkOptimizedForm(b, "(if x 1 0)", Vars{Symbol("x"): NewBool(true)})
}

func BenchmarkEvalOptimizedCoalesceNil(b *testing.B) {
	benchmarkOptimizedForm(b, "(coalesceNil nil x)", Vars{Symbol("x"): NewInt(1)})
}

func BenchmarkEvalOptimizedMatch(b *testing.B) {
	benchmarkOptimizedForm(b, "(match x 1 1 _ 0)", Vars{Symbol("x"): NewInt(1)})
}

func BenchmarkEvalOptimizedBegin(b *testing.B) {
	benchmarkOptimizedForm(b, "(begin (define y x) (eval (quote y)))", Vars{Symbol("x"): NewInt(1)})
}

func BenchmarkEvalOptimizedBeginMut(b *testing.B) {
	benchmarkOptimizedForm(b, "(begin_mut 0 (define y x) (eval (quote y)))", Vars{Symbol("x"): NewInt(1)})
}

func BenchmarkEvalOptimizedBangBegin(b *testing.B) {
	benchmarkOptimizedForm(b, "(!begin x x)", Vars{Symbol("x"): NewInt(1)})
}
