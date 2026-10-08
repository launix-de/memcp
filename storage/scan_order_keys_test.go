/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "testing"
import "github.com/launix-de/memcp/scm"

// Both fixture layouts interleave every stream in the requested output order.
// The compressed permutation exercises actual packed column/row-ID reads.
func orderedPrefixMergeFixture(t testing.TB, streams int, native bool) func() {
	const perKey = 1024
	keys := make([]scm.Scmer, streams)
	for i := range keys {
		keys[i] = scm.NewInt(int64(i))
	}
	less := scm.Apply(scm.Globalenv.Vars[scm.Symbol("collate")], scm.NewString("bin"), scm.NewBool(false)).Func()
	prefix := &orderedPrefixAccess{column: "key", keys: recSetProjectKeys{width: 1, values: keys}, order: []string{"stamp"}, dirs: []func(...scm.Scmer) scm.Scmer{less}}
	cols := []colGetter{
		{raw: ColumnReaderFunc(func(id uint32) scm.Scmer { return scm.NewInt(int64(id) / perKey) })},
		{raw: ColumnReaderFunc(func(id uint32) scm.Scmer { return scm.NewInt(int64(id%perKey)*int64(streams) + int64(id)/perKey) })},
	}
	index := &StorageIndex{t: &storageShard{main_count: uint32(streams * perKey)}, Cols: []string{"key", "stamp"}}
	main := StorageInt{}
	if !native {
		rows := uint32(streams * perKey)
		keyCol, stampCol := &StorageInt{}, &StorageInt{}
		main.initValuesUInt32(rows, 0, rows-1)
		keyCol.initValuesUInt32(rows, 0, uint32(streams-1))
		stampCol.initValuesUInt32(rows, 0, rows-1)
		for id := uint32(0); id < rows; id++ {
			keyCol.buildValueUInt32(id, id%uint32(streams))
			stampCol.buildValueUInt32(id, id)
			main.buildValueUInt32(id, (id%perKey)*uint32(streams)+id/perKey)
		}
		cols = []colGetter{{raw: keyCol}, {raw: stampCol}}
	}
	buf := make([]uint32, 32)
	return func() {
		count := 0
		index.iteratePrefixMerge(nil, prefix, cols, main, nil, native, 0, buf, func(ids []uint32) bool {
			for _, id := range ids {
				want := uint32(count%streams*perKey + count/streams)
				if !native {
					want = uint32(count)
				}
				if id != want {
					t.Fatalf("row %d: got %d want %d", count, id, want)
				}
				count++
			}
			return count < 1024
		})
		if count != 1024 {
			t.Fatalf("count=%d", count)
		}
	}
}

func TestOrderedPrefixMergeHeadReplacement(t *testing.T) {
	for _, streams := range []int{3, 40, 256} {
		for _, native := range []bool{true, false} {
			t.Run(fmt.Sprintf("streams%d/native%v", streams, native), func(t *testing.T) {
				run := orderedPrefixMergeFixture(t, streams, native)
				for i := 0; i < 3; i++ {
					run()
				}
			})
		}
	}
}

// Isolate prefix seek/merge from SQL compilation, ACL preparation and mapping.
func BenchmarkOrderedPrefixMerge(b *testing.B) {
	for _, streams := range []int{3, 40, 256} {
		for _, native := range []bool{true, false} {
			b.Run(fmt.Sprintf("streams%d/native%v", streams, native), func(b *testing.B) {
				run := orderedPrefixMergeFixture(b, streams, native)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					run()
				}
			})
		}
	}
}

func TestOrderedPrefixBoundsCanonicalComparator(t *testing.T) {
	canonical := scm.Apply(scm.Globalenv.Vars[scm.Symbol("collate")], scm.NewString("bin"), scm.NewBool(false)).Func()
	prefix := &orderedPrefixAccess{column: "key"}
	bounds := orderedPrefixBounds(scanAccess{}, prefix)
	relation, meta := bounds.boundaryOrder(0)
	if meta != "bin:asc" {
		t.Fatalf("order metadata = %q", meta)
	}
	less := scm.OrderRelationLess(relation)
	values := []scm.Scmer{scm.NewNil(), scm.NewBool(false), scm.NewBool(true), scm.NewInt(-1), scm.NewInt(2), scm.NewFloat(2.5), scm.NewString("true"), scm.NewSymbol("view")}
	for _, a := range values {
		for _, b := range values {
			if got, want := less(a, b), scm.ToBool(canonical(a, b)); got != want {
				t.Fatalf("prefix order differs for %v, %v", a, b)
			}
		}
	}
	left, right := scm.NewInt(1), scm.NewInt(2)
	if allocations := testing.AllocsPerRun(1000, func() {
		if !less(left, right) {
			panic("wrong order")
		}
	}); allocations != 0 {
		t.Fatalf("prefix comparison allocates %g times; want zero", allocations)
	}
	if scm.FunctionIdentity(relation) != scm.FunctionIdentity(canonical) {
		t.Fatal("prefix comparator lost canonical identity")
	}
}
