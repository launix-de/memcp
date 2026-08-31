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
	"fmt"
	"strings"
	"testing"
)

func TestOptimizeFlattensMappedRangeWithoutIntermediate(t *testing.T) {
	env := newOptimizerTestEnv()
	optimized := optimizeTestSource(t, env, `(lambda (count)
		(merge (map (produceN count) (lambda (index) (list index (+ index 1))))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "flat_map_range") || strings.Contains(serialized, "produceN") {
		t.Fatalf("mapped range was not fused with flatten: %s", serialized)
	}
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewInt(3))
	want := NewSlice([]Scmer{NewInt(0), NewInt(1), NewInt(1), NewInt(2), NewInt(2), NewInt(3)})
	if !Equal(got, want) {
		t.Fatalf("fused mapped range returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeFlattensAssocMapWithoutIntermediate(t *testing.T) {
	env := newOptimizerTestEnv()
	optimized := optimizeTestSource(t, env, `(lambda (dict)
		(merge (extract_assoc dict (lambda (key value) (list key value)))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "flat_map_assoc") || strings.Contains(serialized, "extract_assoc") {
		t.Fatalf("assoc map was not fused with flatten: %s", serialized)
	}
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewString("a"), NewInt(1), NewString("b"), NewInt(2)}))
	want := NewSlice([]Scmer{NewString("a"), NewInt(1), NewString("b"), NewInt(2)})
	if !Equal(got, want) {
		t.Fatalf("fused assoc map returned %s, want %s", String(got), String(want))
	}
}

func benchmarkPlannerFusionProc(b *testing.B, source string) func(...Scmer) Scmer {
	b.Helper()
	env := newOptimizerTestEnv()
	optimized := optimizeTestSource(b, env, source)
	return OptimizeProcToSerialFunction(Eval(optimized, env))
}

func BenchmarkPlannerFlattenMappedRange(b *testing.B) {
	fn := benchmarkPlannerFusionProc(b, `(lambda (count)
		(merge (map (produceN count) (lambda (index) (list index (+ index 1))))))`)
	count := NewInt(128)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := fn(count); got.Slice()[255].Int() != 128 {
			b.Fatal("unexpected mapped range result")
		}
	}
}

func plannerFusionBenchmarkDict(entries int) Scmer {
	fd := NewFastDictValue(entries)
	for i := 0; i < entries; i++ {
		fd.Set(NewString(fmt.Sprintf("k%03d", i)), NewInt(int64(i+1000)), nil)
	}
	return NewFastDict(fd)
}

func BenchmarkPlannerFlattenAssoc(b *testing.B) {
	fn := benchmarkPlannerFusionProc(b, `(lambda (dict)
		(merge (extract_assoc dict (lambda (key value) (list key value)))))`)
	dict := plannerFusionBenchmarkDict(128)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := fn(dict); len(got.Slice()) != 256 {
			b.Fatal("unexpected flattened assoc result")
		}
	}
}

const plannerFusionSplitAndTermsSource = `(define split_and_terms (lambda (expr)
	(match (coalesceNil expr true)
		((symbol and) a b) (merge (list (split_and_terms a) (split_and_terms b)))
		(cons head tail) (if (or (equal? head (quote and)) (equal? head (symbol "and")))
			(merge (map tail split_and_terms))
			(list expr))
		_ (list expr))))`

func TestOptimizeFiltersAndTermsWithoutIntermediate(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("planner fusion test", plannerFusionSplitAndTermsSource, env)
	optimized := optimizeTestSource(t, env, `(lambda (tree)
		(filter (split_and_terms tree) (lambda (term) (> term 2))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "filter_and_terms") || strings.Contains(serialized, "(split_and_terms ") {
		t.Fatalf("AND-tree split and filter were not fused: %s", serialized)
	}
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	tree := NewSlice([]Scmer{
		NewSymbol("and"),
		NewInt(1),
		NewSlice([]Scmer{NewSymbol("and"), NewInt(2), NewInt(3)}),
		NewInt(4),
	})
	got := fn(tree)
	want := NewSlice([]Scmer{NewInt(3), NewInt(4)})
	if !Equal(got, want) {
		t.Fatalf("fused AND-tree filter returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeDoesNotFuseUnrecognizedSplitAndTerms(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("planner fusion test", `(define split_and_terms (lambda (expr) (list expr)))`, env)
	optimized := optimizeTestSource(t, env, `(lambda (tree)
		(filter (split_and_terms tree) (lambda (term) true)))`)
	if serialized := serializedTestExpr(t, env, optimized); strings.Contains(serialized, "filter_and_terms") {
		t.Fatalf("unrecognized sequence producer was fused: %s", serialized)
	}
}

func plannerFusionAndTree(depth int, next *int64) Scmer {
	if depth == 0 {
		value := NewInt(*next)
		*next++
		return value
	}
	return NewSlice([]Scmer{
		NewSymbol("and"),
		plannerFusionAndTree(depth-1, next),
		plannerFusionAndTree(depth-1, next),
	})
}

func BenchmarkPlannerFilterAndTerms(b *testing.B) {
	env := newOptimizerTestEnv()
	EvalAll("planner fusion benchmark", plannerFusionSplitAndTermsSource, env)
	optimized := optimizeTestSource(b, env, `(lambda (tree)
		(filter (split_and_terms tree) (lambda (term) (> term 127))))`)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	var next int64
	tree := plannerFusionAndTree(8, &next)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := fn(tree); len(got.Slice()) != 128 {
			b.Fatal("unexpected filtered AND terms result")
		}
	}
}
