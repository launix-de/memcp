/*
Copyright (C) 2026  Carl-Philip Haensch

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

func requireAssocValue(t *testing.T, dict Scmer, key, want Scmer) {
	t.Helper()
	if !dict.IsFastDict() {
		t.Fatalf("result is not a FastDict: %s", String(dict))
	}
	got, ok := dict.FastDict().Get(key)
	if !ok || !Equal(got, want) {
		t.Fatalf("group %s = %s, want %s", String(key), String(got), String(want))
	}
}

func TestGroupAssocReducesEachKeyFromNeutral(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(group_assoc values
			(lambda (value) (> value 2))
			(lambda (sum value) (+ sum value))
			0))`)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3), NewInt(4)}))
	requireAssocValue(t, got, NewBool(false), NewInt(3))
	requireAssocValue(t, got, NewBool(true), NewInt(7))
}

func TestOptimizeGroupAssocLowersAppendAndCountReducers(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "append",
			source: `(lambda (pairs) (group_assoc pairs car
				(lambda (values pair) (append values (cadr pair))) '()))`,
			want: "group_assoc_append",
		},
		{
			name:   "count",
			source: `(lambda (values) (group_assoc values (lambda (value) value) (lambda (count value) (+ count 1)) 0))`,
			want:   "group_assoc_count",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			optimized, env := optimizeListPipeline(t, tc.source)
			serialized := serializedTestExpr(t, env, optimized)
			if !strings.Contains(serialized, tc.want) || strings.Contains(serialized, "(group_assoc ") {
				t.Fatalf("group_assoc reducer was not lowered to %s: %s", tc.want, serialized)
			}
		})
	}
}

func TestGroupAssocPhysicalLoweringsPreserveResults(t *testing.T) {
	appendOptimized, appendEnv := optimizeListPipeline(t, `(lambda (pairs) (group_assoc pairs car
		(lambda (values pair) (append values (cadr pair))) '()))`)
	appendFn := OptimizeProcToSerialFunction(Eval(appendOptimized, appendEnv))
	input := NewSlice([]Scmer{
		NewSlice([]Scmer{NewString("a"), NewInt(1)}),
		NewSlice([]Scmer{NewString("b"), NewInt(2)}),
		NewSlice([]Scmer{NewString("a"), NewInt(3)}),
	})
	grouped := appendFn(input)
	requireAssocValue(t, grouped, NewString("a"), NewSlice([]Scmer{NewInt(1), NewInt(3)}))
	requireAssocValue(t, grouped, NewString("b"), NewSlice([]Scmer{NewInt(2)}))

	countOptimized, countEnv := optimizeListPipeline(t, `(lambda (values)
		(group_assoc values (lambda (value) value) (lambda (count value) (+ count 1)) 0))`)
	countFn := OptimizeProcToSerialFunction(Eval(countOptimized, countEnv))
	counted := countFn(NewSlice([]Scmer{NewString("a"), NewString("b"), NewString("a")}))
	requireAssocValue(t, counted, NewString("a"), NewInt(2))
	requireAssocValue(t, counted, NewString("b"), NewInt(1))
}

func TestGroupAssocBareItemCallbackKeepsNestedFrame(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (nodes aliases connected)
		(group_assoc connected car (lambda (subsets subset) (append subsets subset)) '()))`)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	subsetA := NewSlice([]Scmer{NewString("a"), NewString("b")})
	subsetB := NewSlice([]Scmer{NewString("a"), NewString("c")})
	got := fn(NewNil(), NewString("outer-alias"), NewSlice([]Scmer{subsetA, subsetB}))
	requireAssocValue(t, got, NewString("a"), NewSlice([]Scmer{subsetA, subsetB}))
}

func TestOptimizeRecognizesImperativeAssocReducers(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "append",
			source: `(lambda (pairs) (reduce pairs (lambda (dict pair)
				(set_assoc dict (car pair)
					(append (get_assoc dict (car pair) '()) (cadr pair)))) '()))`,
			want: "group_assoc_append_reduce",
		},
		{
			name: "count",
			source: `(lambda (values) (reduce values (lambda (dict value)
				(set_assoc dict value (+ (get_assoc dict value 0) 1))) '()))`,
			want: "group_assoc_count_reduce",
		},
		{
			name: "append with stable local key",
			source: `(lambda (pairs) (reduce pairs (lambda (dict pair)
				(begin
					(define key (car pair))
					(set_assoc dict key
						(append (get_assoc dict key '()) (cadr pair))))) '()))`,
			want: "group_assoc_append_reduce",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			optimized, env := optimizeListPipeline(t, tc.source)
			serialized := serializedTestExpr(t, env, optimized)
			if !strings.Contains(serialized, tc.want) || strings.Contains(serialized, "(reduce ") {
				t.Fatalf("imperative assoc reducer was not normalized to %s: %s", tc.want, serialized)
			}
		})
	}
}

func TestOptimizeKeepsNonEquivalentAssocReducer(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (pairs) (reduce pairs (lambda (dict pair)
		(set_assoc dict (car pair)
			(append (get_assoc dict (cadr pair) '()) pair))) '()))`)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "group_assoc_append_reduce") || !strings.Contains(serialized, "reduce") {
		t.Fatalf("mismatched get/set keys were incorrectly normalized: %s", serialized)
	}
}

var groupAssocBenchmarkSink Scmer

func benchmarkGroupedInput(size, groups int) Scmer {
	values := make([]Scmer, size)
	for i := range values {
		values[i] = NewSlice([]Scmer{NewInt(int64(i % groups)), NewInt(int64(i))})
	}
	return NewSlice(values)
}

func BenchmarkOptimizerGroupedAppend(b *testing.B) {
	fn := benchmarkPlannerFusionProc(b, `(lambda (pairs) (reduce pairs (lambda (dict pair)
		(set_assoc dict (car pair)
			(append (get_assoc dict (car pair) '()) (cadr pair)))) '()))`)
	input := benchmarkGroupedInput(2048, 32)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		groupAssocBenchmarkSink = fn(input)
	}
}

func BenchmarkOptimizerGroupedCount(b *testing.B) {
	fn := benchmarkPlannerFusionProc(b, `(lambda (pairs) (reduce pairs (lambda (dict pair)
		(set_assoc dict (car pair) (+ (get_assoc dict (car pair) 0) 1))) '()))`)
	input := benchmarkGroupedInput(2048, 32)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		groupAssocBenchmarkSink = fn(input)
	}
}
