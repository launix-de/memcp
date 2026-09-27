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
