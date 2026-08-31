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

import (
	"strings"
	"testing"
)

func TestOptimizeMatchEliminatesEquivalentSymbolBranches(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("match optimizer test", `(define match_equivalent_symbols (lambda (expr)
		(match expr
			((symbol add) left right) (+ left right)
			((quote add) left right) 991
			((symbol sub) left right) (- left right)
			((quote sub) left right) 992
			_ 0)))`, env)
	proc := env.FindRead(Symbol("match_equivalent_symbols")).Vars[Symbol("match_equivalent_symbols")]
	serialized := serializedTestExpr(t, env, proc.Proc().Body)
	if strings.Contains(serialized, "991") || strings.Contains(serialized, "992") {
		t.Fatalf("unreachable equivalent match branches survived optimization: %s", serialized)
	}
	if strings.Contains(serialized, "(quote add)") || strings.Contains(serialized, "(quote sub)") {
		t.Fatalf("symbol-literal patterns were not canonicalized: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(proc)
	add := NewSlice([]Scmer{NewSymbol("add"), NewInt(7), NewInt(5)})
	sub := NewSlice([]Scmer{NewSymbol("sub"), NewInt(7), NewInt(5)})
	if got := fn(add); !Equal(got, NewInt(12)) {
		t.Fatalf("optimized add match returned %s, want 12", String(got))
	}
	if got := fn(sub); !Equal(got, NewInt(2)) {
		t.Fatalf("optimized sub match returned %s, want 2", String(got))
	}
}

func TestOptimizeMatchEliminatesNestedEquivalentSymbolBranches(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("match optimizer test", `(define match_nested_equivalent_symbols (lambda (expr)
		(match expr
			((symbol not) ((symbol exists) value)) value
			((quote not) ((quote exists) value)) 991
			_ 0)))`, env)
	proc := env.FindRead(Symbol("match_nested_equivalent_symbols")).Vars[Symbol("match_nested_equivalent_symbols")]
	serialized := serializedTestExpr(t, env, proc.Proc().Body)
	if strings.Contains(serialized, "991") {
		t.Fatalf("unreachable nested match branch survived optimization: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(proc)
	expr := NewSlice([]Scmer{
		NewSymbol("not"),
		NewSlice([]Scmer{NewSymbol("exists"), NewInt(42)}),
	})
	if got := fn(expr); !Equal(got, NewInt(42)) {
		t.Fatalf("optimized nested match returned %s, want 42", String(got))
	}
}

func TestOptimizeMatchKeepsDynamicEvalBranches(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("match optimizer test", `(define match_dynamic_eval (lambda (expr expected)
		(match expr
			(eval expected) 1
			(eval expected) 2
			0)))`, env)
	proc := env.FindRead(Symbol("match_dynamic_eval")).Vars[Symbol("match_dynamic_eval")]
	serialized := serializedTestExpr(t, env, proc.Proc().Body)
	if strings.Count(serialized, "(eval ") != 2 {
		t.Fatalf("dynamic match patterns were incorrectly combined: %s", serialized)
	}
}

const recursiveMatchBenchmarkSource = `(define recursive_match_benchmark
	(lambda (expr)
		(match expr
			((symbol leaf) value) value
			((quote leaf) value) (+ value 0)
			((symbol add) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			((quote add) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			((symbol sub) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			((quote sub) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			((symbol and) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			((quote and) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			((symbol or) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			((quote or) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			((symbol equal) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			((quote equal) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			((symbol concat) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			((quote concat) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			((symbol multiply) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			((quote multiply) left right) (+ (recursive_match_benchmark left) (recursive_match_benchmark right))
			_ 0)))`

func recursiveMatchBenchmarkTree(depth int) Scmer {
	if depth == 0 {
		return NewSlice([]Scmer{NewSymbol("leaf"), NewInt(1)})
	}
	child := recursiveMatchBenchmarkTree(depth - 1)
	return NewSlice([]Scmer{NewSymbol("multiply"), child, child})
}

func BenchmarkOptimizeRecursiveMatchWalker(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := newOptimizerTestEnv()
		EvalAll("recursive match benchmark", recursiveMatchBenchmarkSource, env)
	}
}

func BenchmarkRecursiveMatchWalker(b *testing.B) {
	env := newOptimizerTestEnv()
	EvalAll("recursive match benchmark", recursiveMatchBenchmarkSource, env)
	fn := OptimizeProcToSerialFunction(env.FindRead(Symbol("recursive_match_benchmark")).Vars[Symbol("recursive_match_benchmark")])
	tree := recursiveMatchBenchmarkTree(9)
	want := NewInt(512)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := fn(tree); !Equal(got, want) {
			b.Fatalf("recursive match walker returned %s, want 512", String(got))
		}
	}
}
