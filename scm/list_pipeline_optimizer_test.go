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
