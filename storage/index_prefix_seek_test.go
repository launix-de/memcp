// Copyright (C) 2026 MemCP Contributors
// SPDX-License-Identifier: GPL-3.0-or-later
package storage

import (
	"github.com/launix-de/memcp/scm"
	"testing"
)

func setupDeltaPrefixSeek(tb testing.TB) *storageShard {
	tb.Helper()
	name := "test_delta_prefix_seek"
	databases.Remove(name)
	tb.Cleanup(func() { databases.Remove(name) })
	CreateDatabase(name, true)
	tbl, _ := CreateTable(name, "history", Memory, true)
	tbl.CreateColumn("parent", "INT", nil, nil)
	tbl.CreateColumn("version", "INT", nil, nil)
	rows := make([][]scm.Scmer, 0, 16384)
	for parent := 0; parent < 4096; parent++ {
		for version := 0; version < 4; version++ {
			rows = append(rows, []scm.Scmer{scm.NewInt(int64(parent)), scm.NewInt(int64(version))})
		}
	}
	tbl.Insert([]string{"parent", "version"}, rows, nil, scm.NewNil(), false, nil)
	return tbl.Shards[0]
}

func deltaPrefixSeek(shard *storageShard, parent int64, inclusive bool) []uint32 {
	release := shard.GetRead(nil, false)
	defer release.Release()
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	bounds := runtimeScanAccess(analyzedBoundaries{
		{col: "parent", matcher: EqualMatcher, lower: scm.NewInt(parent), upper: scm.NewInt(parent), lowerInclusive: true, upperInclusive: true},
		{col: "version", matcher: RangeMatcher, upper: scm.NewInt(2), upperInclusive: inclusive},
	})
	var buf [8]uint32
	var rows []uint32
	shard.iterateIndexForce(nil, bounds, len(shard.deltaRows()), buf[:], false, func(batch []uint32) bool { rows = append(rows, batch...); return true })
	return rows
}

func TestDeltaIndexPrefixWithUnboundedSuffix(t *testing.T) {
	shard := setupDeltaPrefixSeek(t)
	for _, parent := range []int64{0, 2048, 4095, 5000} {
		for _, inclusive := range []bool{false, true} {
			rows := deltaPrefixSeek(shard, parent, inclusive)
			want := 2
			if inclusive {
				want = 3
			}
			if parent == 5000 {
				want = 0
			}
			if len(rows) != want {
				t.Fatalf("parent %d inclusive %v got %v", parent, inclusive, rows)
			}
			for i, row := range rows {
				if row != uint32(parent*4+int64(i)) {
					t.Fatalf("unexpected row %d at %d", row, i)
				}
			}
		}
	}
}

func BenchmarkDeltaIndexPrefixWithUnboundedSuffix(b *testing.B) {
	shard := setupDeltaPrefixSeek(b)
	deltaPrefixSeek(shard, 4095, true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := deltaPrefixSeek(shard, 4095, true); len(got) != 3 {
			b.Fatalf("wrong result %v", got)
		}
	}
}

// Cold source counts, rather than index length or directory order, decide
// between complete prefix matches. Logical input dependencies are deduplicated.
func TestIndexPrefixMinimizesColumnLoads(t *testing.T) {
	tbl := &table{Columns: []*column{
		{Name: "id"}, {Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"},
		{Name: "computed", ComputorInputCols: []string{"b", "c"}},
	}}
	shard := &storageShard{t: tbl, columns: map[string]ColumnStorage{
		"id": new(StorageSparse), "a": nil, "b": nil,
		"c": new(StorageSparse), "d": new(StorageSparse), "computed": new(StorageSparse),
	}}
	makeIndex := func(names ...string) *StorageIndex {
		index := &StorageIndex{t: shard, Cols: names, ColOrderMeta: make([]string, len(names))}
		for i := range names {
			index.ColOrderMeta[i] = "bin:asc"
		}
		return index
	}
	access := runtimeScanAccess(analyzedBoundaries{
		{col: "id", matcher: EqualMatcher, lower: scm.NewInt(1), upper: scm.NewInt(1), lowerInclusive: true, upperInclusive: true},
	})
	coldShort := makeIndex("id", "a")
	coldWide := makeIndex("id", "a", "b")
	warmWide := makeIndex("id", "c", "d")
	oneColdWide := makeIndex("id", "b", "c", "d")
	mapped := makeIndex("id", ".mapped")
	mapped.ColMapFn = []scm.Scmer{scm.NewNil(), scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return args[0] })}
	mapped.ColMapCols = [][]string{nil, {"b", "b", "c"}}
	computed := makeIndex("id", "computed")
	wrongOrder := makeIndex("id", "c")
	wrongOrder.ColOrderMeta[0] = "bin:desc"
	wrongMatcher := makeIndex("id", "c")
	wrongMatcher.ColMatchers = []IndexAnalyzer{RecSetMatcher}
	for _, tc := range []struct {
		name    string
		indexes []*StorageIndex
		want    *StorageIndex
		missing int
	}{
		{"warm wide beats cold short", []*StorageIndex{coldShort, warmWide}, warmWide, 0},
		{"fewer loads beats shorter key", []*StorageIndex{coldWide, oneColdWide}, oneColdWide, 1},
		{"mapped inputs count once", []*StorageIndex{coldWide, mapped}, mapped, 1},
		{"computed inputs count", []*StorageIndex{coldWide, computed}, computed, 1},
		{"incompatible order rejected", []*StorageIndex{wrongOrder, coldShort}, coldShort, 1},
		{"incompatible matcher rejected", []*StorageIndex{wrongMatcher, coldShort}, coldShort, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shard.mu.RLock()
			defer shard.mu.RUnlock()
			got, missing := shard.leastColdIndex(access, 1, tc.indexes)
			if got != tc.want || missing != tc.missing {
				t.Fatalf("index=%p missing=%d; want index=%p missing=%d", got, missing, tc.want, tc.missing)
			}
		})
	}
	full := runtimeScanAccess(analyzedBoundaries{
		{col: "id", matcher: EqualMatcher, lower: scm.NewInt(1), upper: scm.NewInt(1), lowerInclusive: true, upperInclusive: true},
		{col: "a", matcher: EqualMatcher, lower: scm.NewInt(2), upper: scm.NewInt(2), lowerInclusive: true, upperInclusive: true},
	})
	if got, _ := shard.leastColdIndex(full, 2, []*StorageIndex{makeIndex("id"), warmWide, coldWide}); got != coldWide {
		t.Fatal("selected an index that did not cover the complete requested prefix")
	}

	// No zero-load candidate exists: prepare the cheaper wide key only.
	tbl.PersistencyMode = Memory
	tbl.schema = &database{Name: "memcp-tests"}
	shard.setState(SHARED)
	shard.main_count = 1
	shard.plannerMainRows.Store(1)
	shard.Indexes = []*StorageIndex{coldWide, oneColdWide}
	lease := shard.GetRead(nil, false)
	defer lease.Release()
	access.preparedIndex = shard.prepareIndexColumns(access, false, nil)
	if access.preparedIndex != oneColdWide {
		t.Fatal("preparation did not retain its selected covering key")
	}
	if shard.columns["b"] == nil || shard.columns["a"] != nil || len(shard.Indexes) != 2 {
		t.Fatal("preparation loaded an avoidable column or created a duplicate index")
	}
	if got, missing := shard.leastColdIndex(access, 1, shard.Indexes); got != oneColdWide || missing != 0 {
		t.Fatal("iteration did not retain a fully prepared covering index")
	}
}

// A covering prefix is required for reuse; a short existing key must not stop
// definition creation or the ordinary savings-triggered physical index build.
func TestIndexPrefixStillCreatesAndBuildsMissingKeys(t *testing.T) {
	rows := make([][]scm.Scmer, 1024)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewString("value")}
	}
	tbl := setupScanLookupTable(t, "memcp-tests", rows)
	shard := tbl.ActiveShards()[0]
	lease := shard.GetRead(nil, false)
	defer lease.Release()
	for width := 1; width <= 2; width++ {
		bounds := analyzedBoundaries{
			{col: "key", matcher: EqualMatcher, lower: scm.NewInt(511), upper: scm.NewInt(511), lowerInclusive: true, upperInclusive: true},
		}
		if width == 2 {
			bounds = append(bounds, analyzedBoundary{col: "value", matcher: EqualMatcher,
				lower: scm.NewString("value"), upper: scm.NewString("value"), lowerInclusive: true, upperInclusive: true})
		}
		access := runtimeScanAccess(bounds)
		for probe := 1; probe <= 2; probe++ {
			if _, found := shard.scanFirstRecord(access, nil, scm.NewBool(true), nil, nil, nil); !found {
				t.Fatal("missing fixture row")
			}
			if len(shard.Indexes) != width {
				t.Fatalf("width=%d probe=%d: definitions=%d, want %d", width, probe, len(shard.Indexes), width)
			}
			index := shard.Indexes[width-1]
			if len(index.Cols) != width {
				t.Fatal("reused an incomplete prefix")
			}
			index.mu.Lock()
			active := index.baseState != nil && index.baseState.active
			index.mu.Unlock()
			if active != (probe == 2) {
				t.Fatalf("width=%d probe=%d: active=%t; adaptive build did not run at its normal threshold", width, probe, active)
			}
		}
	}
}
