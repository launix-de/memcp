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
	first := NewString("first")
	second := NewString("second")
	third := NewString("third")
	forcedHash := uint64(42)
	dict := &FastDict{
		Pairs:      []Scmer{first, NewInt(1), second, NewInt(20), third, NewInt(3)},
		index:      map[uint64]int{forcedHash: 0},
		collisions: map[uint64][]int{forcedHash: {2, 4}},
	}

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

func TestStructuralCatalogDoesNotUseSQLTruthinessForASTs(t *testing.T) {
	catalog := &structuralCatalog{}
	catalog.set(NewSlice([]Scmer{NewSymbol("if"), NewBool(true)}), NewString("literal"))
	catalog.set(NewSlice([]Scmer{NewSymbol("if"), NewSlice([]Scmer{NewSymbol("probe")})}), NewString("probe"))

	if len(catalog.entries) != 2 {
		t.Fatalf("structural catalog collapsed truthy AST nodes: %#v", catalog.entries)
	}
	got := catalog.get(NewSlice([]Scmer{NewSymbol("if"), NewSlice([]Scmer{NewSymbol("probe")})}))
	if got.String() != "probe" {
		t.Fatalf("probe-shaped AST lookup returned %s", SerializeToString(got, &Globalenv))
	}
}

func BenchmarkFunctionalAssocBuild(b *testing.B) {
	env := newOptimizerTestEnv()
	optimized := optimizeTestSource(b, env, `(lambda (items)
		(reduce items (lambda (index item)
			(set_assoc index item item)) '()))`)
	build := serialTestCallable(Eval(optimized, env))
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

// StorageSeq and StorageInt can represent the same SQL key with float and int
// tags in different shards. Their partial groups must merge into one bucket.
func TestFastDictNumericRepresentation(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, tuple := range []bool{false, true} {
			left, right := NewInt(1), NewFloat(1)
			if reverse {
				left, right = right, left
			}
			if tuple {
				left, right = NewSlice([]Scmer{left}), NewSlice([]Scmer{right})
			}
			dict := NewFastDictValue(4)
			dict.Set(left, NewInt(6), nil)
			dict.Set(right, NewInt(9), func(a, b Scmer) Scmer { return NewInt(a.Int() + b.Int()) })
			if len(dict.Pairs) != 2 {
				t.Fatalf("reverse=%v tuple=%v: equal numeric keys occupy %d buckets", reverse, tuple, len(dict.Pairs)/2)
			}
			for _, key := range []Scmer{left, right} {
				got, ok := dict.Get(key)
				if !ok || got.Int() != 15 {
					t.Fatalf("lookup = %v, %v; want 15", got, ok)
				}
			}
		}
	}
}

func TestFastDictNumericHashBoundaries(t *testing.T) {
	dict := NewFastDictValue(8)
	keys := []Scmer{NewInt(1 << 53), NewInt(1<<53 + 1), NewNil(), NewBool(false), NewInt(0)}
	for i, key := range keys {
		dict.Set(key, NewInt(int64(i)), nil)
	}
	for i, key := range keys {
		value, ok := dict.Get(key)
		if !ok || value.Int() != int64(i) {
			t.Fatalf("key %d collided with a distinct value: %v, %v", i, value, ok)
		}
	}
	value, ok := dict.Get(NewFloat(math.Copysign(0, -1)))
	if !ok || value.Int() != 4 {
		t.Fatalf("negative floating zero did not find integer zero: %v, %v", value, ok)
	}
}

func TestMergeAssocPreservesInputs(t *testing.T) {
	merge := PrepareSerialProc(Globalenv.Vars["merge_assoc"])
	for _, sliceSource := range []bool{false, true} {
		left := benchmarkFastDict(32)
		right := benchmarkFastDict(48)
		if sliceSource {
			right = NewSlice(append([]Scmer(nil), right.FastDict().Pairs...))
		}
		beforeLeft, beforeRight := SerializeToString(left, &Globalenv), SerializeToString(right, &Globalenv)
		combine := NewFunc(func(args ...Scmer) Scmer { return NewInt(args[0].Int() + args[1].Int()) })
		got := merge.Call([]Scmer{left, right, combine})
		for i := 0; i < 48; i++ {
			value, ok := got.FastDict().Get(NewString(fmt.Sprintf("key-%d", i)))
			want := int64(i)
			if i < 32 {
				want *= 2
			}
			if !ok || value.Int() != want {
				t.Fatalf("key %d: got %v, expected %d", i, value, want)
			}
		}
		if SerializeToString(left, &Globalenv) != beforeLeft || SerializeToString(right, &Globalenv) != beforeRight {
			t.Fatal("merge changed an input dictionary")
		}
		if merge.Call([]Scmer{left, NewSlice(nil)}) != left {
			t.Fatal("empty merge did not preserve the original value")
		}
		func() {
			defer func() {
				if recover() == nil {
					t.Error("merge callback error was swallowed")
				}
			}()
			calls := 0
			merge.Call([]Scmer{left, right, NewFunc(func(args ...Scmer) Scmer {
				calls++
				if calls == 5 {
					panic("merge failed")
				}
				return NewInt(args[0].Int() + args[1].Int() + 100)
			})})
		}()
		if SerializeToString(left, &Globalenv) != beforeLeft {
			t.Fatal("failed merge changed its input")
		}
	}
}

func BenchmarkMergeAssocImmutable(b *testing.B) {
	merge := PrepareSerialProc(Globalenv.Vars["merge_assoc"])
	for _, size := range []int{64, 512, 4096} {
		left, right := benchmarkFastDict(size), benchmarkFastDict(size)
		args := []Scmer{left, right}
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				assocBenchmarkSink = merge.Call(args)
			}
		})
	}
}

func TestAssocMergeOneShotCallbacks(t *testing.T) {
	for _, name := range []string{"set_assoc", "set_assoc_mut"} {
		for _, fast := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/fast=%v", name, fast), func(t *testing.T) {
				setter := Globalenv.Vars[Symbol(name)]
				makeDictionary := func() Scmer {
					if fast {
						dict := NewFastDictValue(8)
						dict.Set(NewInt(1), NewInt(11), nil)
						return NewFastDict(dict)
					}
					return NewSlice([]Scmer{NewInt(1), NewInt(11)})
				}
				calls := 0
				callback := NewFunc(func(args ...Scmer) Scmer {
					calls++
					if len(args) != 2 || args[0].Int() != 11 || args[1].Int() != 7 {
						t.Fatalf("merger arguments = %v; want old=11, new=7", args)
					}
					return NewInt(4)
				})
				result := Apply(setter, makeDictionary(), NewFloat(1), NewInt(7), callback)
				if calls != 1 || Apply(result, NewInt(1)).Int() != 4 {
					t.Fatalf("duplicate numeric key: calls=%d, result=%v", calls, result)
				}
				result = Apply(setter, makeDictionary(), NewInt(2), NewInt(7), callback)
				if calls != 1 || Apply(result, NewInt(2)).Int() != 7 {
					t.Fatal("missing key invoked the merger or changed the new value")
				}
				factory := preparedTestProc(t, "(lambda (captured) (lambda (old new) (+ captured (* old 10) new)))")
				captured := Apply(factory, NewInt(5))
				result = Apply(setter, makeDictionary(), NewInt(1), NewInt(7), captured)
				if got := Apply(result, NewInt(1)).Int(); got != 122 {
					t.Fatalf("captured merger result=%d; want 122", got)
				}
				result = Apply(setter, makeDictionary(), NewInt(1), NewInt(7), Globalenv.Vars["list"])
				pair := Apply(result, NewInt(1))
				Apply(setter, makeDictionary(), NewInt(1), NewInt(9), Globalenv.Vars["list"])
				if !Equal(pair, NewSlice([]Scmer{NewInt(11), NewInt(7)})) {
					t.Fatal("native merger retained a borrowed argument frame")
				}
				result = Apply(setter, makeDictionary(), NewInt(1), NewInt(7), NewBool(false))
				if got := Apply(result, NewInt(1)); !got.IsBool() || got.Bool() {
					t.Fatal("constant-result merger compatibility changed")
				}
			})
		}
	}
}

func TestAssocMissingKeyDoesNotPrepareUnusedMerger(t *testing.T) {
	merger := preparedTestProc(t, "(lambda (old new) (+ (* old 2) (* new 3) (* old new) (if (< old new) old new)))")
	for _, name := range []string{"set_assoc", "set_assoc_mut"} {
		t.Run(name, func(t *testing.T) {
			setter := Globalenv.Vars[Symbol(name)]
			args := []Scmer{NewSlice(nil), NewString("missing"), NewInt(7), merger}
			allocations := testing.AllocsPerRun(100, func() {
				assocBenchmarkSink = Apply(setter, args...)
			})
			if allocations > 8 {
				t.Fatalf("missing-key update allocated %.1f objects without invoking its merger", allocations)
			}
		})
	}
}

func BenchmarkAssocMergeOneShot(b *testing.B) {
	merger := preparedTestProc(b, "(lambda (old new) (+ (* old 2) (* new 3) (* old new) (if (< old new) old new)))")
	for _, name := range []string{"set_assoc", "set_assoc_mut"} {
		for _, existing := range []bool{false, true} {
			b.Run(fmt.Sprintf("%s/existing=%v", name, existing), func(b *testing.B) {
				setter := Globalenv.Vars[Symbol(name)]
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					var dictionary Scmer
					if existing {
						dictionary = NewSlice([]Scmer{NewString("key"), NewInt(11)})
					} else {
						dictionary = NewSlice(nil)
					}
					assocBenchmarkSink = Apply(setter, dictionary, NewString("key"), NewInt(7), merger)
				}
			})
		}
	}
}

func TestAssocMergeKeepsCompiledEntryAuthoritative(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	merger := preparedTestProc(t, "(lambda (old new) 7)")
	merger.Proc().Compiled = &JITEntryPoint{Native: func(...Scmer) Scmer { return NewInt(99) }}
	for _, name := range []string{"set_assoc", "set_assoc_mut"} {
		result := Apply(Globalenv.Vars[Symbol(name)], NewSlice([]Scmer{NewString("key"), NewInt(11)}), NewString("key"), NewInt(7), merger)
		if got := Apply(result, NewString("key")); !Equal(got, NewInt(99)) {
			t.Fatalf("%s compiled merger result = %s, want 99", name, String(got))
		}
	}
}
