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

const recursiveSpecializationBenchmarkSource = `(define benchmark_recursive_collect (lambda (expr acc)
	(match expr
		((symbol node) child) (benchmark_recursive_collect child acc)
		((quote node) child) (benchmark_recursive_collect (list (symbol "node") child) acc)
		((symbol value) value) (append acc value)
		((quote value) value) (benchmark_recursive_collect (list (symbol "value") value) acc)
		(cons _head tail) (reduce tail (lambda (state child)
			(benchmark_recursive_collect child state)) acc)
		_ acc)))
(define benchmark_recursive_collect_entry (lambda (expr)
	(benchmark_recursive_collect (list (quote node) expr) (list))))`

func BenchmarkOptimizeRecursiveOwnershipVariants(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		env := newOptimizerTestEnv()
		EvalAll(b.Name(), recursiveSpecializationBenchmarkSource, env)
		if !env.Vars[Symbol("benchmark_recursive_collect_entry")].IsProc() {
			b.Fatal("recursive benchmark Proc was not defined")
		}
	}
}

const nestedOwnershipBenchmarkSource = `(define benchmark_nested_filter (lambda (wrapper)
	(match wrapper
		(cons child _tail) (filter child (lambda (value) (> value 63)))
		_ '())))
(define benchmark_nested_filter_entry (lambda (seed)
	(benchmark_nested_filter (list (map (produceN 128) (lambda (value) (+ value seed)))))))`

func BenchmarkNestedOwnershipMatchFilter(b *testing.B) {
	env := newOptimizerTestEnv()
	EvalAll(b.Name(), nestedOwnershipBenchmarkSource, env)
	fn := OptimizeProcToSerialFunction(env.Vars[Symbol("benchmark_nested_filter_entry")])
	if got := fn(NewInt(0)); len(got.Slice()) != 64 {
		b.Fatalf("nested filter returned %s", String(got))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := fn(NewInt(0)); len(got.Slice()) != 64 {
			b.Fatal("invalid result")
		}
	}
}
