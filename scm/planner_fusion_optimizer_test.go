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
	fn := serialTestCallable(Eval(optimized, env))
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
	fn := serialTestCallable(Eval(optimized, env))
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
	return serialTestCallable(Eval(optimized, env))
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
		((quote and) a b) (merge (list (split_and_terms a) (split_and_terms b)))
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
	fn := serialTestCallable(Eval(optimized, env))
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

const plannerTaggedAssocFoldSource = `(define planner_test_resolve_alias (lambda (value fallback)
	(coalesceNil value fallback)))
(define planner_test_tagged_assoc_fold (lambda (default_alias expr aliases)
	(match expr
		((symbol get_column) tblvar _ _ _) (set_assoc aliases (planner_test_resolve_alias tblvar default_alias) true)
		((quote get_column) tblvar _ _ _) (set_assoc aliases (planner_test_resolve_alias tblvar default_alias) true)
		(cons head tail) (reduce tail (lambda (found item)
			(planner_test_tagged_assoc_fold default_alias item found))
			(planner_test_tagged_assoc_fold default_alias head aliases))
		_ aliases)))`

func TestOptimizeRecognizesRecursiveTaggedAssocFold(t *testing.T) {
	env := newOptimizerTestEnv()
	definitionOffset := strings.Index(plannerTaggedAssocFoldSource, "(define planner_test_tagged_assoc_fold")
	definition, ok := scmerSlice(Read(t.Name(), plannerTaggedAssocFoldSource[definitionOffset:]))
	if !ok || len(definition) != 3 {
		t.Fatal("could not parse recursive tagged assoc fold definition")
	}
	if _, recognized := optimizeRecursiveTaggedAssocFold(Symbol("planner_test_tagged_assoc_fold"), definition[2]); !recognized {
		t.Fatalf("raw recursive tagged assoc fold was not recognized: %s", String(definition[2]))
	}
	EvalAll(t.Name(), plannerTaggedAssocFoldSource, env)
	proc := env.Vars[Symbol("planner_test_tagged_assoc_fold")]
	if !proc.IsProc() {
		t.Fatal("tagged assoc fold was not defined")
	}
	serialized := serializedTestExpr(t, env, proc.Proc().Body)
	if !strings.Contains(serialized, "optimizer_tree_collect_tagged_nth_unique") || strings.Contains(serialized, "(match ") {
		t.Fatalf("recursive tagged assoc fold was not lowered after full-shape recognition: %s", serialized)
	}
	got := Apply(proc,
		NewSymbol("default"),
		NewSlice([]Scmer{
			NewSymbol("root"),
			NewSlice([]Scmer{NewSymbol("get_column"), NewSymbol("a"), NewNil(), NewNil(), NewNil()}),
			NewSlice([]Scmer{NewSymbol("nested"), NewSlice([]Scmer{NewSymbol("get_column"), NewNil(), NewNil(), NewNil(), NewNil()})}),
		}),
		NewSlice(nil),
	)
	want := NewSlice([]Scmer{NewSymbol("a"), NewBool(true), NewSymbol("default"), NewBool(true)})
	if !Equal(got, want) {
		t.Fatalf("lowered tagged assoc fold returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeRejectsPartialTaggedAssocFoldShape(t *testing.T) {
	env := newOptimizerTestEnv()
	source := strings.Replace(plannerTaggedAssocFoldSource, "\t\t_ aliases)))", "\t\t_ (list))))", 1)
	EvalAll(t.Name(), source, env)
	proc := env.Vars[Symbol("planner_test_tagged_assoc_fold")]
	if !proc.IsProc() {
		t.Fatal("near-miss tagged assoc fold was not defined")
	}
	if serialized := serializedTestExpr(t, env, proc.Proc().Body); strings.Contains(serialized, "optimizer_tree_collect_tagged_nth_unique") {
		t.Fatalf("partial tagged assoc fold shape was lowered: %s", serialized)
	}
}

const plannerTaggedPredicateSource = `(define planner_test_predicate_resolve (lambda (value fallback)
	(coalesceNil value fallback)))
(define planner_test_tagged_predicate (lambda (default_alias alias expr)
	(match expr
		((symbol get_column) tblvar _ _ _) (equal?? (planner_test_predicate_resolve tblvar default_alias) alias)
		((quote get_column) tblvar _ _ _) (equal?? (planner_test_predicate_resolve tblvar default_alias) alias)
		(cons _head tail) (reduce tail (lambda (found item)
			(or found (planner_test_tagged_predicate default_alias alias item))) false)
		_ false)))
(define planner_test_tagged_predicate_set (lambda (default_alias aliases expr)
	(reduce (coalesceNil aliases '()) (lambda (found alias)
		(or found (planner_test_tagged_predicate default_alias alias expr))) false)))`

func TestOptimizeRecognizesTaggedPredicateAndCandidateSet(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll(t.Name(), plannerTaggedPredicateSource, env)
	predicate := env.Vars[Symbol("planner_test_tagged_predicate")]
	setPredicate := env.Vars[Symbol("planner_test_tagged_predicate_set")]
	if !predicate.IsProc() || !setPredicate.IsProc() {
		t.Fatal("tagged predicates were not defined")
	}
	for name, proc := range map[string]Scmer{"single": predicate, "set": setPredicate} {
		serialized := serializedTestExpr(t, env, proc.Proc().Body)
		if !strings.Contains(serialized, "optimizer_expr_tagged_nth_matches_any") || strings.Contains(serialized, "(reduce ") || strings.Contains(serialized, "(match ") {
			t.Fatalf("%s tagged predicate was not fused after full-shape recognition: %s", name, serialized)
		}
	}
	tree := NewSlice([]Scmer{
		NewSlice([]Scmer{
			NewSlice([]Scmer{NewSymbol("lambda"), NewSlice(nil), NewSlice([]Scmer{NewSymbol("get_column"), NewSymbol("operator"), NewNil(), NewNil(), NewNil()})}),
		}),
		NewSlice([]Scmer{NewSymbol("nested"), NewSlice([]Scmer{NewSymbol("get_column"), NewSymbol("operand"), NewNil(), NewNil(), NewNil()})}),
	})
	if got := Apply(predicate, NewSymbol("default"), NewSymbol("operator"), tree); !got.IsBool() || got.Bool() {
		t.Fatalf("tagged predicate entered an operator head: %s", String(got))
	}
	if got := Apply(setPredicate, NewSymbol("default"), NewSlice([]Scmer{NewSymbol("missing"), NewSymbol("operand")}), tree); !got.IsBool() || !got.Bool() {
		t.Fatalf("tagged candidate-set predicate missed an operand: %s", String(got))
	}
}

func TestOptimizeRejectsPartialTaggedPredicateShape(t *testing.T) {
	env := newOptimizerTestEnv()
	source := strings.Replace(plannerTaggedPredicateSource, "\t\t_ false)))", "\t\t_ true)))", 1)
	EvalAll(t.Name(), source, env)
	predicate := env.Vars[Symbol("planner_test_tagged_predicate")]
	if !predicate.IsProc() {
		t.Fatal("near-miss tagged predicate was not defined")
	}
	if serialized := serializedTestExpr(t, env, predicate.Proc().Body); strings.Contains(serialized, "optimizer_expr_tagged_nth_matches_any") {
		t.Fatalf("partial tagged predicate shape was fused: %s", serialized)
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
	fn := serialTestCallable(Eval(optimized, env))
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
