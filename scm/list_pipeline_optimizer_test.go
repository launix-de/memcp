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
	"fmt"
	"math"
	"strings"
	"testing"
)

func optimizeListPipeline(t testing.TB, source string) (Scmer, *Env) {
	t.Helper()
	env := newOptimizerTestEnv()
	optimized := Optimize(Read("list pipeline optimizer test", source), env, nil)
	return optimized, env
}

func TestJITFusedListPipelinesInlineKnownCallbacks(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	tests := []struct {
		name   string
		fused  string
		source string
	}{
		{"filter map", "filter_map", `(lambda (values) (filter (map values (lambda (value) (+ value 1))) (lambda (value) (> value 2))))`},
		{"map filter not null", "map_filter_notnull", `(lambda (values) (filter (map values (lambda (value) (if (> value 1) (* value 10) nil))) (lambda (value) (not (nil? value)))))`},
		{"map filter", "map_filter", `(lambda (values) (map (filter values (lambda (value) (> value 1))) (lambda (value) (+ value 1))))`},
		{"map map", "map_map", `(lambda (values) (map (map values (lambda (value) (+ value 1))) (lambda (value) (* value 2))))`},
		{"filter filter", "filter_filter", `(lambda (values) (filter (filter values (lambda (value) (> value 0))) (lambda (value) (< value 4))))`},
		{"mapped sum", "sum_map", `(lambda (values) (reduce values (lambda (total value) (+ total (* value 2))) 0))`},
		{"boolean any", "reduce_any", `(lambda (values) (reduce values (lambda (found value) (or found (> value 2))) false))`},
		{"boolean all", "reduce_all", `(lambda (values) (reduce values (lambda (valid value) (and valid (> value 2))) true))`},
		{"find mapped non-null", "find_map_notnull", `(lambda (values) (reduce values (lambda (found value) (if (not (nil? found)) found (if (> value 2) (* value 10) nil))) nil))`},
		{"reduce map", "reduce_map", `(lambda (values) (reduce (map values (lambda (value) (+ value 1))) (lambda (acc value) (+ acc value))))`},
		{"reduce filter", "reduce_filter", `(lambda (values) (reduce (filter values (lambda (value) (> value 1))) (lambda (acc value) (+ acc value))))`},
		{"reduce map filter", "reduce_map_filter", `(lambda (values) (reduce (filter (map values (lambda (value) (* value 2))) (lambda (value) (> value 2))) (lambda (acc value) (+ acc value))))`},
		{"reduce filter map", "reduce_filter_map", `(lambda (values) (reduce (map (filter values (lambda (value) (> value 1))) (lambda (value) (* value 2))) (lambda (acc value) (+ acc value))))`},
		{"cons map", "cons_map", `(lambda (head values) (cons head (map values (lambda (value) (+ value 1)))))`},
		{"flat map", "flat_map", `(lambda (values) (merge (map values (lambda (value) (list value (+ value 10))))))`},
		{"group append", "group_assoc_append", `(lambda (pairs) (group_assoc pairs (lambda (pair) (car pair)) (lambda (values pair) (append values (cadr pair))) '()))`},
		{"group count", "group_assoc_count", `(lambda (values) (group_assoc values (lambda (value) value) (lambda (count value) (+ count 1)) 0))`},
		{"group append reduce", "group_assoc_append_reduce", `(lambda (pairs) (reduce pairs (lambda (dict pair) (set_assoc dict (car pair) (append (get_assoc dict (car pair) '()) (cadr pair)))) '()))`},
		{"group count reduce", "group_assoc_count_reduce", `(lambda (values) (reduce values (lambda (dict value) (set_assoc dict value (+ (get_assoc dict value 0) 1))) '()))`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			optimized, env := optimizeListPipeline(t, test.source)
			serialized := serializedTestExpr(t, env, optimized)
			if !strings.Contains(serialized, test.fused) {
				t.Fatalf("pipeline was not fused to %s: %s", test.fused, serialized)
			}
			compiled := jitCompile(Eval(optimized, env))
			if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
				t.Fatal("fused pipeline did not compile")
			}
			coverage := compiled.Proc().Compiled.Coverage
			if coverage.DynamicCalls != 0 || coverage.InlinedCalls == 0 {
				t.Fatalf("known callbacks were not fully inlined: %+v", coverage)
			}
		})
	}
}

func TestJITFusedMappedSumExecutesWithStackPhiHome(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(reduce values (lambda (total value) (+ total (* value 2))) 0))`)
	compiled := jitCompile(Eval(optimized, env))
	if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("fused mapped sum did not compile")
	}
	values := NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3)})
	if got := Apply(compiled, values); !Equal(got, NewInt(12)) {
		t.Fatalf("fused mapped sum = %s, want 12", String(got))
	}
}

func BenchmarkJITFusedReduceFilterMap(b *testing.B) {
	if !jitEnabled {
		b.Skip("requires GOEXPERIMENT=jit")
	}
	optimized, env := optimizeListPipeline(b, `(lambda (values)
		(reduce (map (filter values (lambda (value) (> value 31))) (lambda (value) (* value 2)))
			(lambda (acc value) (+ acc value)) 0))`)
	compiled := jitCompile(Eval(optimized, env))
	if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		b.Fatal("fused reduce/filter/map benchmark did not compile")
	}
	valuesSlice := make([]Scmer, 128)
	for index := range valuesSlice {
		valuesSlice[index] = NewInt(int64(index))
	}
	values := NewSlice(valuesSlice)
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		jitListBenchmarkSink = Apply(compiled, values)
	}
}

func TestOptimizeFusesFilterOverMap(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(filter (map values (lambda (value) (+ value 1)))
			(lambda (value) (> value 2))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "filter_map") {
		t.Fatalf("filter/map pipeline was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(0), NewInt(2), NewInt(3)}))
	want := NewSlice([]Scmer{NewInt(3), NewInt(4)})
	if !Equal(got, want) {
		t.Fatalf("fused filter/map returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeFusesMapAndNonNilFilter(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(filter (map values (lambda (value) (if (> value 1) (* value 10) nil)))
			(lambda (value) (not (nil? value)))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "map_filter_notnull") {
		t.Fatalf("map/non-nil filter pipeline was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(0), NewInt(2), NewInt(3)}))
	want := NewSlice([]Scmer{NewInt(20), NewInt(30)})
	if !Equal(got, want) {
		t.Fatalf("fused map/non-nil filter returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeLowersBooleanReducers(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		fused    string
		input    []Scmer
		expected Scmer
	}{
		{
			name:     "or preserves unknown",
			source:   `(lambda (values) (reduce values (lambda (found value) (or found (if (equal? value 0) nil (> value 2)))) false))`,
			fused:    "reduce_any",
			input:    []Scmer{NewInt(0), NewInt(1)},
			expected: NewNil(),
		},
		{
			name:     "or short circuits true",
			source:   `(lambda (values) (reduce values (lambda (found value) (or found (if (equal? value 0) nil (> value 2)))) false))`,
			fused:    "reduce_any",
			input:    []Scmer{NewInt(0), NewInt(3)},
			expected: NewBool(true),
		},
		{
			name:     "and preserves unknown",
			source:   `(lambda (values) (reduce values (lambda (valid value) (and valid (if (equal? value 0) nil (> value 1)))) true))`,
			fused:    "reduce_all",
			input:    []Scmer{NewInt(0), NewInt(2)},
			expected: NewNil(),
		},
		{
			name:     "and short circuits false",
			source:   `(lambda (values) (reduce values (lambda (valid value) (and valid (if (equal? value 0) nil (> value 1)))) true))`,
			fused:    "reduce_all",
			input:    []Scmer{NewInt(0), NewInt(1)},
			expected: NewBool(false),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			optimized, env := optimizeListPipeline(t, tc.source)
			serialized := serializedTestExpr(t, env, optimized)
			if !strings.Contains(serialized, tc.fused) {
				t.Fatalf("boolean reducer was not lowered to %s: %s", tc.fused, serialized)
			}
			fn := OptimizeProcToSerialFunction(Eval(optimized, env))
			got := fn(NewSlice(tc.input))
			if !Equal(got, tc.expected) {
				t.Fatalf("lowered reducer returned %s, want %s", String(got), String(tc.expected))
			}
		})
	}
}

func TestOptimizeLowersFirstNonNilReducer(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(reduce values
			(lambda (found value)
				(if (not (nil? found)) found (if (> value 2) (* value 10) nil)))
			nil))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "find_map_notnull") {
		t.Fatalf("first non-nil reducer was not lowered: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(1), NewInt(3), NewInt(4)}))
	want := NewInt(30)
	if !Equal(got, want) {
		t.Fatalf("lowered first non-nil reducer returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeFusesFirstNonNilReducerOverRange(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (count target)
		(reduce (produceN count)
			(lambda (found index)
				(if (not (nil? found)) found (if (equal? index target) index nil)))
			nil))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "find_range_notnull") || strings.Contains(serialized, "produceN") {
		t.Fatalf("range search was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	if got := fn(NewInt(8), NewInt(5)); !Equal(got, NewInt(5)) {
		t.Fatalf("fused range search returned %s, want 5", String(got))
	}
	if got := fn(NewInt(8), NewInt(9)); !got.IsNil() {
		t.Fatalf("fused range search returned %s, want nil", String(got))
	}
}

func TestOptimizeFusesDirectFirstNonNilSearchOverRange(t *testing.T) {
	if declaration := DeclarationForValue(NewSymbol("find_map_notnull")); declaration == nil || declaration.Optimize == nil {
		t.Fatal("find_map_notnull declaration has no optimizer hook")
	}
	optimized, env := optimizeListPipeline(t, `(lambda (count target)
		(find_map_notnull (produceN count)
			(lambda (_ index) (if (equal? index target) index nil))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "find_range_notnull") || strings.Contains(serialized, "produceN") {
		t.Fatalf("direct range search was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	if got := fn(NewInt(8), NewInt(5)); !Equal(got, NewInt(5)) {
		t.Fatalf("direct fused range search returned %s, want 5", String(got))
	}
}

func TestOptimizeFusesGeneralReducerOverRange(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (count)
		(reduce (produceN count) (lambda (values index) (cons index values)) '()))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "reduce_range") || strings.Contains(serialized, "produceN") {
		t.Fatalf("range reducer was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewInt(4))
	want := NewSlice([]Scmer{NewInt(3), NewInt(2), NewInt(1), NewInt(0)})
	if !Equal(got, want) {
		t.Fatalf("fused range reducer returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeFusesRangeReducerWithoutNeutral(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (count)
		(reduce (produceN count) (lambda (total index) (+ total index))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "reduce_range") || strings.Contains(serialized, "produceN") {
		t.Fatalf("range reducer without neutral was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	if got := fn(NewInt(4)); !Equal(got, NewInt(6)) {
		t.Fatalf("fused range reducer returned %s, want 6", String(got))
	}
	if got := fn(NewInt(0)); !got.IsNil() {
		t.Fatalf("empty fused range reducer returned %s, want nil", String(got))
	}
}

func TestOptimizeRangeReducerPreservesRetainedCallbackArguments(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (count)
		(reduce (produceN count) list))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "reduce_range") || strings.Contains(serialized, "produceN") {
		t.Fatalf("range reducer with retaining callback was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewInt(3))
	want := NewSlice([]Scmer{
		NewSlice([]Scmer{NewInt(0), NewInt(1)}),
		NewInt(2),
	})
	if !Equal(got, want) {
		t.Fatalf("fused retaining range reducer returned %s, want %s", String(got), String(want))
	}
}

func BenchmarkOptimizerRangeFindPlanner(b *testing.B) {
	optimized, env := optimizeListPipeline(b, `(lambda (count target)
		(reduce (produceN count)
			(lambda (found index)
				(if (not (nil? found)) found (if (equal? index target) index nil)))
			nil))`)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	count := NewInt(64)
	target := NewInt(63)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := fn(count, target); !Equal(got, target) {
			b.Fatalf("range search returned %s, want %s", String(got), String(target))
		}
	}
}

func TestOptimizeLowersMappedSumReducer(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(reduce values (lambda (total value) (+ total (* value 2))) 0))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "sum_map") {
		t.Fatalf("mapped sum reducer was not lowered: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3)}))
	want := NewInt(12)
	if !Equal(got, want) {
		t.Fatalf("lowered mapped sum returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeKeepsReducerWithWrongNeutral(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(reduce values (lambda (found value) (or found (> value 1))) true))`)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "reduce_any") {
		t.Fatalf("OR reducer with non-identity neutral was unsafely lowered: %s", serialized)
	}
}

func TestOptimizeKeepsProducerFusionAheadOfBooleanReducer(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(reduce (map values (lambda (value) (> value 1)))
			(lambda (found value) (or found value)) false))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "reduce_map") || strings.Contains(serialized, "reduce_any") {
		t.Fatalf("boolean lowering displaced allocation-free producer fusion: %s", serialized)
	}
}

func TestOptimizeFusesFixedWidthMergeOverMap(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(merge (map values (lambda (value) (list value (+ value 10))))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "flat_map") {
		t.Fatalf("fixed-width merge/map pipeline was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(1), NewInt(2)}))
	want := NewSlice([]Scmer{NewInt(1), NewInt(11), NewInt(2), NewInt(12)})
	if !Equal(got, want) {
		t.Fatalf("fused merge/map returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeKeepsDynamicWidthMergeOverMap(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(merge (map values (lambda (value) value))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "flat_map") {
		t.Fatalf("dynamic-width merge/map pipeline was unsafely fused: %s", serialized)
	}
}

func TestOptimizeFusesFixedWidthMergeOverProduceN(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda ()
		(merge (map (produceN 3) (lambda (value) (list value (+ value 10))))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "flat_map") {
		t.Fatalf("fixed-width merge/produceN pipeline was not fused: %s", serialized)
	}

	got := OptimizeProcToSerialFunction(Eval(optimized, env))()
	want := NewSlice([]Scmer{NewInt(0), NewInt(10), NewInt(1), NewInt(11), NewInt(2), NewInt(12)})
	if !Equal(got, want) {
		t.Fatalf("fused merge/produceN returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeFusesMapOverFilter(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(map (filter values (lambda (value) (> value 1))) (lambda (value) (+ value 1))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "map_filter") {
		t.Fatalf("map/filter pipeline was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(0), NewInt(1), NewInt(2), NewInt(3)}))
	want := NewSlice([]Scmer{NewInt(3), NewInt(4)})
	if !Equal(got, want) {
		t.Fatalf("fused map/filter returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeFusesMapOverMap(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(map (map values (lambda (value) (+ value 1))) (lambda (value) (* value 2))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "map_map") {
		t.Fatalf("map/map pipeline was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3)}))
	want := NewSlice([]Scmer{NewInt(4), NewInt(6), NewInt(8)})
	if !Equal(got, want) {
		t.Fatalf("fused map/map returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeFusesFilterOverFilter(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(filter (filter values (lambda (value) (> value 0))) (lambda (value) (< value 4))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "filter_filter") {
		t.Fatalf("filter/filter pipeline was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(-1), NewInt(1), NewInt(3), NewInt(5)}))
	want := NewSlice([]Scmer{NewInt(1), NewInt(3)})
	if !Equal(got, want) {
		t.Fatalf("fused filter/filter returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeFusesReduceOverMap(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(reduce (map values (lambda (value) (+ value 1))) (lambda (acc value) (+ acc value))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "reduce_map") {
		t.Fatalf("reduce/map pipeline was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3)}))
	want := NewInt(9)
	if !Equal(got, want) {
		t.Fatalf("fused reduce/map returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeLeavesGeneralIndexedReduceUnfused(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(reduce (mapIndex values (lambda (index value) (+ index value)))
			(lambda (total value) (+ total value)) 0))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "mapIndex") || !strings.Contains(serialized, "reduce") || strings.Contains(serialized, "index_assoc") {
		t.Fatalf("general indexed reduce must stay unspecialized: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	if got := fn(NewSlice([]Scmer{NewInt(10), NewInt(20), NewInt(30)})); !Equal(got, NewInt(63)) {
		t.Fatalf("general indexed reduce returned %s, want 63", String(got))
	}
}

func TestOptimizeBuildsIndexedAssocWithoutMappedPairs(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (aliases)
		(reduce (mapIndex aliases (lambda (position alias) (list alias position)))
			(lambda (index entry) (set_assoc index (car entry) (cadr entry))) '()))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "index_assoc") || strings.Contains(serialized, "mapIndex") || strings.Contains(serialized, "reduce") {
		t.Fatalf("indexed assoc construction was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewString("left"), NewString("right"), NewString("left")}))
	want := NewSlice([]Scmer{NewString("left"), NewInt(2), NewString("right"), NewInt(1)})
	if !Equal(got, want) {
		t.Fatalf("fused indexed assoc returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeBuildsIndexedAssocFromOptionalAliases(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(begin
		(define alias-index (lambda (aliases)
			(reduce (mapIndex (coalesceNil aliases '()) (lambda (position alias) (list alias position)))
				(lambda (index entry) (set_assoc index (toLower (car entry)) entry)) '())))
		alias-index)`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "index_assoc") || strings.Contains(serialized, "mapIndex") || strings.Contains(serialized, "reduce") {
		t.Fatalf("optional alias index construction was not specialized: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewString("LEFT"), NewString("Right")}))
	want := NewSlice([]Scmer{
		NewString("left"), NewSlice([]Scmer{NewString("LEFT"), NewInt(0)}),
		NewString("right"), NewSlice([]Scmer{NewString("Right"), NewInt(1)}),
	})
	if !Equal(got, want) {
		t.Fatalf("fused optional alias index returned %s, want %s", String(got), String(want))
	}
	if got := fn(NewNil()); !Equal(got, NewSlice(nil)) {
		t.Fatalf("fused optional empty alias index returned %s, want empty assoc", String(got))
	}
}

func TestOptimizeFusesReduceOverFilter(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(reduce (filter values (lambda (value) (> value 1))) (lambda (acc value) (+ acc value))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "reduce_filter") {
		t.Fatalf("reduce/filter pipeline was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(4)}))
	want := NewInt(6)
	if !Equal(got, want) {
		t.Fatalf("fused reduce/filter returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeFusesReduceOverMapThenFilter(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(reduce (map (filter values (lambda (value) (> value 1))) (lambda (value) (* value 2)))
			(lambda (acc value) (+ acc value))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "reduce_filter_map") {
		t.Fatalf("reduce/map/filter pipeline was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3)}))
	want := NewInt(10)
	if !Equal(got, want) {
		t.Fatalf("fused reduce/map/filter returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeFusesMergeUniqueOverMapOfLists(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(merge_unique (map values (lambda (value) (list (+ value 1))))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "flat_map_unique") {
		t.Fatalf("merge_unique/map-of-lists pipeline was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(2), NewInt(3)}))
	want := NewSlice([]Scmer{NewInt(2), NewInt(3), NewInt(4)})
	if !Equal(got, want) {
		t.Fatalf("fused merge_unique/map returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeKeepsMergeUniqueOverMapWithUnknownItems(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(merge_unique (map values (lambda (value) value))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "flat_map_unique") {
		t.Fatalf("merge_unique/map with unproven mapper result was fused: %s", serialized)
	}
}

func TestOptimizeKeepsMergeUniqueOverFilterMapOfLists(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(merge_unique (map (filter values (lambda (value) (> value 1))) (lambda (value) (list (* value 2))))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "flat_map_unique") {
		t.Fatalf("merge_unique/map/filter-of-lists pipeline lost its filter: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(0), NewInt(1), NewInt(2), NewInt(2), NewInt(3)}))
	want := NewSlice([]Scmer{NewInt(4), NewInt(6)})
	if !Equal(got, want) {
		t.Fatalf("merge_unique/map/filter returned %s, want %s", String(got), String(want))
	}
}

func TestOrderedUniqueBuilderPreservesEqualAcrossHashDomains(t *testing.T) {
	builder := orderedUniqueBuilder{}
	for i := int64(0); i < orderedUniqueHashThreshold; i++ {
		builder.add(NewInt(i))
	}
	builder.add(NewFloat(3))
	builder.add(NewString("4"))
	got := NewSlice(builder.result())
	want := NewSlice([]Scmer{
		NewInt(0), NewInt(1), NewInt(2), NewInt(3),
		NewInt(4), NewInt(5), NewInt(6), NewInt(7),
	})
	if !Equal(got, want) {
		t.Fatalf("ordered unique fallback returned %s, want %s", String(got), String(want))
	}
}

func TestOrderedUniqueBuilderPreservesFirstValueInHashDomain(t *testing.T) {
	builder := orderedUniqueBuilder{}
	for i := 0; i < orderedUniqueHashThreshold; i++ {
		builder.add(NewString(fmt.Sprintf("column-%d", i)))
	}
	builder.add(NewSymbol("column-3"))
	got := builder.result()
	if len(got) != orderedUniqueHashThreshold || got[3].GetTag() != tagString {
		t.Fatalf("ordered unique hash path did not preserve the first value: %v", got)
	}

	floatBuilder := orderedUniqueBuilder{}
	for i := 0; i < orderedUniqueHashThreshold; i++ {
		floatBuilder.add(NewFloat(float64(i)))
	}
	floatBuilder.add(NewFloat(math.Copysign(0, -1)))
	if result := floatBuilder.result(); len(result) != orderedUniqueHashThreshold {
		t.Fatalf("ordered unique hash path treated signed zero as distinct: %v", result)
	}
}

func TestOptimizeMergeUniqueMutatesFreshList(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (a b c)
		(merge_unique (list (list a b) (list b c))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "merge_unique_mut") {
		t.Fatalf("fresh list did not select merge_unique_mut: %s", serialized)
	}
	if strings.Contains(serialized, "!list") {
		t.Fatalf("transferred merge_unique input was lowered to a frame-local list: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewInt(1), NewInt(2), NewInt(3))
	want := NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3)})
	if !Equal(got, want) {
		t.Fatalf("merge_unique_mut returned %s, want %s", String(got), String(want))
	}
	second := fn(NewInt(4), NewInt(5), NewInt(6))
	if !Equal(got, want) {
		t.Fatalf("later call mutated escaped result to %s, want %s", String(got), String(want))
	}
	secondWant := NewSlice([]Scmer{NewInt(4), NewInt(5), NewInt(6)})
	if !Equal(second, secondWant) {
		t.Fatalf("second merge_unique_mut returned %s, want %s", String(second), String(secondWant))
	}
}

func TestOptimizeKeepsDynamicReducerAfterMergeValidation(t *testing.T) {
	tests := []struct {
		name   string
		source string
		fused  string
	}{
		{
			name:   "segment catalog",
			source: `(lambda (parts reducer) (reduce (merge parts) reducer 0))`,
			fused:  "reduce_segments",
		},
		{
			name:   "two lists",
			source: `(lambda (left right reducer) (reduce (merge left right) reducer 0))`,
			fused:  "reduce_merge2",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			optimized, env := optimizeListPipeline(t, tc.source)
			serialized := serializedTestExpr(t, env, optimized)
			if strings.Contains(serialized, tc.fused) {
				t.Fatalf("dynamic reducer moved ahead of merge validation: %s", serialized)
			}
		})
	}
}

func BenchmarkPlannerReducerLowerings(b *testing.B) {
	values := make([]Scmer, 128)
	for i := range values {
		values[i] = NewInt(int64(i + 1))
	}
	input := NewSlice(values)
	tests := []struct {
		name   string
		source string
		check  func(Scmer) bool
	}{
		{
			name:   "SumMap",
			source: `(lambda (values) (reduce values (lambda (total value) (+ total (* value 2))) 0))`,
			check:  func(result Scmer) bool { return result.Int() == 16512 },
		},
		{
			name:   "AnyEarly",
			source: `(lambda (values) (reduce values (lambda (found value) (or found (> value 0))) false))`,
			check:  func(result Scmer) bool { return result.Bool() },
		},
		{
			name:   "AllEarly",
			source: `(lambda (values) (reduce values (lambda (valid value) (and valid (< value 1))) true))`,
			check:  func(result Scmer) bool { return !result.Bool() },
		},
		{
			name: "FindMapEarly",
			source: `(lambda (values)
				(reduce values (lambda (found value)
					(if (not (nil? found)) found (if (> value 0) value nil))) nil))`,
			check: func(result Scmer) bool { return result.Int() == 1 },
		},
		{
			name: "MapFilterNotNull",
			source: `(lambda (values)
				(filter (map values (lambda (value) (if (> value 64) value nil)))
					(lambda (value) (not (nil? value)))))`,
			check: func(result Scmer) bool { return len(result.Slice()) == 64 },
		},
	}
	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			optimized, env := optimizeListPipeline(b, tc.source)
			fn := OptimizeProcToSerialFunction(Eval(optimized, env))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if result := fn(input); !tc.check(result) {
					b.Fatal("unexpected benchmark result")
				}
			}
		})
	}
}

func BenchmarkOptimizerIndexedAliasMap(b *testing.B) {
	optimized, env := optimizeListPipeline(b, `(lambda (aliases)
		(reduce (mapIndex aliases (lambda (position alias) (list alias position)))
			(lambda (index entry) (set_assoc index (car entry) (cadr entry))) '()))`)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	aliases := make([]Scmer, 128)
	for i := range aliases {
		aliases[i] = NewString(fmt.Sprintf("alias-%d", i))
	}
	input := NewSlice(aliases)
	last := NewString("alias-127")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := fn(input)
		if got := declarations["get_assoc"].Fn(result, last); got.Int() != 127 {
			b.Fatalf("indexed alias map returned %s, want 127", String(got))
		}
	}
}

func BenchmarkPlannerFlatMapUnique(b *testing.B) {
	values := make([]Scmer, 128)
	for i := range values {
		values[i] = NewString(fmt.Sprintf("column-%d", i))
	}
	input := NewSlice(values)
	optimized, env := optimizeListPipeline(b, `(lambda (values)
		(merge_unique (map values (lambda (value) (list value value)))))`)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if result := fn(input); len(result.Slice()) != len(values) {
			b.Fatal("unexpected benchmark result")
		}
	}
}
