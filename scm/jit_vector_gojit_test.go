//go:build goexperiment.jit && amd64

/*
Copyright (C) 2026  MemCP Contributors

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
	"math"
	"testing"
)

var jitDotBenchmarkSink Scmer

func TestJITDotKeepsNativeFloatAccumulators(t *testing.T) {
	left := NewSlice([]Scmer{NewFloat(3), NewFloat(4)})
	right := NewSlice([]Scmer{NewFloat(3), NewFloat(4)})
	for _, test := range []struct {
		name   string
		source string
		want   float64
	}{
		{name: "dot", source: `(lambda (left right) (dot left right))`, want: 25},
		{name: "cosine", source: `(lambda (left right) (dot left right "COSINE"))`, want: 1},
		{name: "euclidean", source: `(lambda (left right) (dot left right "EUCLIDEAN"))`, want: 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			compiled := jitCompile(Eval(Read(t.Name(), test.source), &Globalenv))
			if compiled.GetTag() != tagProc || compiled.Proc().Compiled == nil {
				t.Fatal("dot expression did not compile")
			}
			got := Apply(compiled, left, right).Float()
			if math.Abs(got-test.want) > 1e-12 {
				t.Fatalf("dot result = %g, want %g", got, test.want)
			}
		})
	}
}

func BenchmarkJITDotNativeFP(b *testing.B) {
	const count = 4096
	left := make([]Scmer, count)
	right := make([]Scmer, count)
	for index := range left {
		left[index] = NewFloat(float64(index%31) + 0.25)
		right[index] = NewFloat(float64(index%17) + 0.75)
	}
	args := []Scmer{NewSlice(left), NewSlice(right)}
	for _, test := range []struct {
		name   string
		source string
	}{
		{"dot", `(lambda (left right) (dot left right))`},
		{"cosine", `(lambda (left right) (dot left right "COSINE"))`},
	} {
		b.Run(test.name, func(b *testing.B) {
			compiled := jitCompile(Eval(Read(b.Name(), test.source), &Globalenv))
			if compiled.GetTag() != tagProc || compiled.Proc().Compiled == nil {
				b.Fatal("dot benchmark did not compile")
			}
			b.ReportAllocs()
			b.ResetTimer()
			for iteration := 0; iteration < b.N; iteration++ {
				jitDotBenchmarkSink = Apply(compiled, args...)
			}
		})
	}
}
