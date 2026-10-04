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

import "math"
import "testing"

func TestLessInt64Exact(t *testing.T) {
	values := []int64{-1 << 63, -1<<63 + 1, -9007199254740993, -9007199254740992, -1, 0, 1, 9007199254740992, 9007199254740993, 1<<63 - 2, 1<<63 - 1}
	for _, a := range values {
		for _, b := range values {
			if got := Less(NewInt(a), NewInt(b)); got != (a < b) {
				t.Errorf("Less(%d, %d) = %v, want %v", a, b, got, a < b)
			}
		}
	}
}

func TestJITIntegerOrderOperatorsExact(t *testing.T) {
	for _, op := range []string{"<", "<=", ">", ">="} {
		t.Run(op, func(t *testing.T) {
			fn := compileJITExpressionTestProc(t, "(lambda (a b) ("+op+" a b))")
			values := []int64{-1 << 63, -1<<63 + 1, -9007199254740993, -9007199254740992, 0, 9007199254740992, 9007199254740993, 1<<63 - 2, 1<<63 - 1}
			for _, a := range values {
				for _, b := range values {
					want := map[string]bool{"<": a < b, "<=": a <= b, ">": a > b, ">=": a >= b}[op]
					if got := Apply(fn, NewInt(a), NewInt(b)).Bool(); got != want {
						t.Fatalf("(%s %d %d) = %v, want %v", op, a, b, got, want)
					}
				}
			}
		})
	}
}

var lessBenchmarkSink bool

func BenchmarkLessInt64(b *testing.B) {
	values := []Scmer{NewInt(17), NewInt(18), NewInt(1 << 53), NewInt(1<<53 + 1)}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		lessBenchmarkSink = Less(values[i&3], values[(i+1)&3])
	}
}

// Compare optimized predicates with the unmodified builtin on mixed types.
// Float-to-integer normalization must preserve rounding and fallback semantics.
func TestOptimizeOrderedNumericConstants(t *testing.T) {
	constants := []float64{-1, 0, math.Copysign(0, -1), 1, 511, 1.5, 1e6, 1<<53 - 1, 1 << 53, -1 << 53, math.Inf(1), math.Inf(-1), math.NaN()}
	values := []Scmer{NewNil(), NewBool(false), NewBool(true), NewInt(math.MinInt64), NewInt(math.MaxInt64), NewInt(-1), NewInt(0), NewInt(1), NewInt(510), NewInt(511), NewInt(512), NewDate(511), NewSymbol("511"), NewAny("x"), NewInt(1<<53 - 1), NewInt(1 << 53), NewInt(1<<53 + 1), NewInt(-1<<53 - 1), NewInt(-1<<53 + 1), NewFloat(511), NewFloat(1.5), NewFloat(math.NaN()), NewFloat(math.Inf(1)), NewString("511"), NewString("1.5"), NewString("x"), NewSlice([]Scmer{NewInt(1)})}
	for _, op := range []string{"<", "<=", ">", ">="} {
		for _, c := range constants {
			for _, left := range []bool{false, true} {
				args := []Scmer{NewSymbol("x"), NewFloat(c)}
				if left {
					args[0], args[1] = args[1], args[0]
				}
				body := NewSlice([]Scmer{NewSymbol(op), args[0], args[1]})
				expr := NewSlice([]Scmer{NewSymbol("lambda"), NewSlice([]Scmer{NewSymbol("x")}), body})
				proc := Eval(Optimize(expr, &Globalenv, nil), &Globalenv).Proc()
				if c == 511 {
					optimized, ok := scmerSlice(proc.Body)
					index := 2
					if left {
						index = 1
					}
					if !ok || len(optimized) != 3 || !optimized[index].IsInt() || optimized[index].Int() != 511 {
						t.Fatalf("%s left=%v: integral bound was not normalized: %v", op, left, proc.Body)
					}
				}
				for _, x := range values {
					actualArgs := []Scmer{x, NewFloat(c)}
					if left {
						actualArgs[0], actualArgs[1] = actualArgs[1], actualArgs[0]
					}
					want := declarations[op].Fn(actualArgs...)
					got := Eval(proc.Body, &Env{Outer: &Globalenv, Vars: Vars{Symbol("x"): x}, VarsNumbered: []Scmer{x}})
					if !Equal(got, want) {
						t.Fatalf("%s c=%v left=%v x=%v: got %v want %v", op, c, left, x, got, want)
					}
				}
			}
		}
	}
}

func BenchmarkOptimizedOrderedComparison(b *testing.B) {
	proc := Eval(Optimize(Read("comparison-benchmark", "(lambda (x) (and (> x -1) (< x 60001) (>= x 0) (<= x 60000)))"), &Globalenv, nil), &Globalenv).Proc()
	env := &Env{Outer: &Globalenv, VarsNumbered: []Scmer{NewInt(511)}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lessBenchmarkSink = Eval(proc.Body, env).Bool()
	}
	if !lessBenchmarkSink {
		b.Fatal("qualifying row rejected")
	}
}
