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
	"strings"
	"testing"
)

var assocBenchmarkSink Scmer

func benchmarkAssocPairs(size int) []Scmer {
	pairs := make([]Scmer, 0, size*2)
	for i := 0; i < size; i++ {
		pairs = append(pairs, NewString(fmt.Sprintf("key-%d", i)), NewInt(int64(i)))
	}
	return pairs
}

func benchmarkFastDict(size int) Scmer {
	dict := NewFastDictValue(size)
	pairs := benchmarkAssocPairs(size)
	for i := 0; i < len(pairs); i += 2 {
		dict.Set(pairs[i], pairs[i+1], nil)
	}
	return NewFastDict(dict)
}

func TestFastDictHashCollisionFallback(t *testing.T) {
	dict := NewFastDictValue(3)
	forcedHash := uint64(42)
	dict.setHashed(NewString("first"), NewInt(1), nil, forcedHash)
	dict.setHashed(NewString("second"), NewInt(2), nil, forcedHash)
	dict.setHashed(NewString("third"), NewInt(3), nil, forcedHash)
	dict.setHashed(NewString("second"), NewInt(20), nil, forcedHash)

	for key, want := range map[string]int64{"first": 1, "second": 20, "third": 3} {
		got, ok := dict.findPos(NewString(key), forcedHash)
		if !ok || dict.Pairs[got+1].Int() != want {
			t.Fatalf("collision lookup for %q returned position %d, found=%v", key, got, ok)
		}
	}
	if len(dict.collisions[forcedHash]) != 2 {
		t.Fatalf("expected two secondary collision positions, got %v", dict.collisions[forcedHash])
	}
}

func BenchmarkAssocLookup(b *testing.B) {
	for _, size := range []int{1, 2, 4, 5, 8, 12, 64} {
		key := NewString(fmt.Sprintf("key-%d", size-1))
		b.Run(fmt.Sprintf("list/%d", size), func(b *testing.B) {
			dict := NewSlice(benchmarkAssocPairs(size))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				assocBenchmarkSink = Apply(dict, key)
			}
		})
		b.Run(fmt.Sprintf("fastdict/%d", size), func(b *testing.B) {
			dict := benchmarkFastDict(size)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				assocBenchmarkSink = Apply(dict, key)
			}
		})
	}
}

func BenchmarkFastDictBuild(b *testing.B) {
	for _, size := range []int{1, 2, 4, 5, 8, 12, 64} {
		pairs := benchmarkAssocPairs(size)
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				dict := NewFastDictValue(size)
				for pos := 0; pos < len(pairs); pos += 2 {
					dict.Set(pairs[pos], pairs[pos+1], nil)
				}
				assocBenchmarkSink = NewFastDict(dict)
			}
		})
	}
}

func TestFunctionalAssocBuilderUsesOwnedAccumulator(t *testing.T) {
	env := newOptimizerTestEnv()
	optimized := optimizeTestSource(t, env, `(lambda (items)
		(reduce items (lambda (index item)
			(set_assoc index item item)) '()))`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "set_assoc_mut") {
		t.Fatalf("functional assoc builder did not use owned accumulator: %s", serialized)
	}
}

func BenchmarkFunctionalAssocBuild(b *testing.B) {
	env := newOptimizerTestEnv()
	optimized := optimizeTestSource(b, env, `(lambda (items)
		(reduce items (lambda (index item)
			(set_assoc index item item)) '()))`)
	build := OptimizeProcToSerialFunction(Eval(optimized, env))
	for _, size := range []int{1, 2, 4, 5, 8, 12, 64} {
		items := make([]Scmer, size)
		for i := range items {
			items[i] = NewString(fmt.Sprintf("key-%d", i))
		}
		input := NewSlice(items)
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				assocBenchmarkSink = build(input)
			}
		})
	}
}
