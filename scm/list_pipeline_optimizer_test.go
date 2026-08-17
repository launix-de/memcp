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

func TestOptimizeFusesMergeUniqueOverMap(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(merge_unique (map values (lambda (value) (+ value 1)))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "merge_unique_map") {
		t.Fatalf("merge_unique/map pipeline was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(2), NewInt(3)}))
	want := NewSlice([]Scmer{NewInt(2), NewInt(3), NewInt(4)})
	if !Equal(got, want) {
		t.Fatalf("fused merge_unique/map returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeFusesMergeUniqueOverFilterMap(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (values)
		(merge_unique (map (filter values (lambda (value) (> value 1))) (lambda (value) (* value 2)))))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "merge_unique_map_filter") {
		t.Fatalf("merge_unique/map/filter pipeline was not fused: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	got := fn(NewSlice([]Scmer{NewInt(0), NewInt(1), NewInt(2), NewInt(2), NewInt(3)}))
	want := NewSlice([]Scmer{NewInt(4), NewInt(6)})
	if !Equal(got, want) {
		t.Fatalf("fused merge_unique/map/filter returned %s, want %s", String(got), String(want))
	}
}
