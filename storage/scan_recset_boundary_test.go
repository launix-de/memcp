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
package storage

import (
	"testing"

	"github.com/launix-de/memcp/scm"
)

func setupAdaptiveRecSetOrderTable(t *testing.T, database string, rows int) *table {
	t.Helper()
	tbl := setupScanParallelTestTable(t, database)
	tbl.CreateColumn("rank", "INT", nil, nil)
	values := make([][]scm.Scmer, rows)
	for i := range values {
		// RecID zero occurs in the middle of the rank index. A sequential
		// membership scan therefore has to pass half the table before its first
		// hit, while a RecSet-dominant traversal touches only the chosen IDs.
		values[i] = []scm.Scmer{
			scm.NewInt(int64(i)),
			scm.NewInt(int64((i + rows/2) % rows)),
		}
	}
	tbl.Insert([]string{"id", "rank"}, values, nil, scm.NewNil(), false, nil)
	RebuildTable(tbl, true, false)
	return tbl
}

func recSetForIDs(tbl *table, ids map[int64]bool) *recSet {
	return tbl.scanRecSet(nil, []string{"id"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewBool(ids[values[0].Int()])
	}), newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil)
}

func scanOrderedRecSetIDs(tbl *table, source *recSet, limit int) []int64 {
	_, order := integerOrder(false)
	return scanOrderedRecSetIDsWithOrder(tbl, source, limit, order)
}

func scanOrderedRecSetIDsWithOrder(tbl *table, source *recSet, limit int, order func(...scm.Scmer) scm.Scmer) []int64 {
	return scanOrderedRecSetIDsWithCondition(tbl, source, limit, order,
		scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }))
}

func scanOrderedRecSetIDsWithCondition(tbl *table, source *recSet, limit int, order func(...scm.Scmer) scm.Scmer, condition scm.Scmer) []int64 {
	result := make([]int64, 0, limit)
	source.scan_order(nil,
		[]string{"id"}, condition,
		[]scm.Scmer{scm.NewString("rank")}, []func(...scm.Scmer) scm.Scmer{order},
		0, 0, limit, []string{"id"},
		scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			result = append(result, values[1].Int())
			return values[0]
		}),
		scm.NewNil(), false, scm.NewNil(), nil, scm.NewNil(), newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil)
	return result
}

func buildRankOrderIndex(t *testing.T, tbl *table) func(...scm.Scmer) scm.Scmer {
	t.Helper()
	_, order := integerOrder(false)
	bounds, ok := extendBoundariesWithSortCols(nil,
		[]scm.Scmer{scm.NewString("rank")}, []func(...scm.Scmer) scm.Scmer{order})
	if !ok {
		t.Fatal("rank ORDER boundary was not constructed")
	}
	lower, upper := indexFromBoundaries(bounds)
	shard := tbl.ActiveShards()[0]
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	var buf [8]uint32
	shard.iterateIndexForce(nil, scanAccess{suffix: bounds}, lower, upper, len(shard.inserts), buf[:], false,
		func([]uint32) bool { return false })
	return order
}

func TestRecSetScanSourceAddsExactBoundary(t *testing.T) {
	tbl := setupAdaptiveRecSetOrderTable(t, "trecsetboundary", 200)
	source := recSetForIDs(tbl, map[int64]bool{3: true, 103: true, 150: true})
	condition := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewBool(values[0].Int() >= 100)
	})
	got := make([]int64, 0, 2)
	source.scan(nil, []string{"id"}, condition, []string{"id"},
		scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			got = append(got, values[1].Int())
			return values[0]
		}), scm.NewNil(), scm.NewNil(), false, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil)

	if want := []int64{103, 150}; !equalInt64s(got, want) {
		t.Fatalf("RecSet boundary scan rows = %v, want %v", got, want)
	}
	if !source.scanExists(nil, []string{"id"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewBool(values[0].Int() == 150)
	}), newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil) {
		t.Fatal("scan_exists did not consume the RecSet boundary")
	}
}

func TestRepeatedUnorderedRecSetScanDoesNotBuildMembershipIndex(t *testing.T) {
	tbl := setupAdaptiveRecSetOrderTable(t, "trecsetdirectrepeat", 2_000)
	source := recSetForIDs(tbl, map[int64]bool{3: true, 103: true, 1_503: true})
	condition := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewBool(values[0].Int() >= 100)
	})

	for range 4 {
		got := make([]int64, 0, 2)
		source.scan(nil, []string{"id"}, condition, []string{"id"},
			scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
				got = append(got, values[1].Int())
				return values[0]
			}), scm.NewNil(), scm.NewNil(), false, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil)
		if want := []int64{103, 1_503}; !equalInt64s(got, want) {
			t.Fatalf("repeated RecSet scan rows = %v, want %v", got, want)
		}
	}

	for _, index := range tbl.ActiveShards()[0].Indexes {
		if len(index.Cols) != 1 || index.Cols[0] != "$recset_contains" {
			continue
		}
		index.mu.Lock()
		active := index.baseState.active
		storedRows := index.baseState.mainIndexes.count
		index.mu.Unlock()
		if active || storedRows != 0 {
			t.Fatalf("repeated exact RecSet scan built a membership index: active=%t rows=%d", active, storedRows)
		}
	}
}

func TestSparseOrderedRecSetBuildsCompressedInversePositions(t *testing.T) {
	const rows = 20_000
	tbl := setupAdaptiveRecSetOrderTable(t, "trecsetsparseorder", rows)
	order := buildRankOrderIndex(t, tbl)
	source := recSetForIDs(tbl, map[int64]bool{0: true, 1: true, rows - 1: true})
	for _, index := range tbl.ActiveShards()[0].Indexes {
		index.mu.Lock()
		inverseCount := index.baseState.mainIndexPositions.count
		index.mu.Unlock()
		if inverseCount != 0 {
			t.Fatalf("building the forward index eagerly built %d inverse positions", inverseCount)
		}
	}

	if got, want := scanOrderedRecSetIDsWithOrder(tbl, source, 3, order), []int64{rows - 1, 0, 1}; !equalInt64s(got, want) {
		t.Fatalf("sparse ordered RecSet rows = %v, want %v", got, want)
	}

	shard := tbl.ActiveShards()[0]
	foundCompressedInverse := false
	for _, index := range shard.Indexes {
		index.mu.Lock()
		inverseCount := index.baseState.mainIndexPositions.count
		inverseBytes := index.baseState.mainIndexPositions.ComputeSize()
		index.mu.Unlock()
		if inverseCount > 0 {
			foundCompressedInverse = true
			if inverseBytes >= uint(rows*4) {
				t.Fatalf("inverse StorageInt uses %d bytes, expected compression below raw %d-byte uint32 array", inverseBytes, rows*4)
			}
		}
	}
	if !foundCompressedInverse {
		t.Fatal("sparse ordered RecSet did not build a compressed inverse index")
	}
}

func TestRepeatedLargeOrderedRecSetUsesInversePositions(t *testing.T) {
	const rows = 100_000
	tbl := setupAdaptiveRecSetOrderTable(t, "trecsetlargeinverse", rows)
	order := buildRankOrderIndex(t, tbl)
	ids := make(map[int64]bool, 2_912)
	for id := int64(0); id < 2_912; id++ {
		ids[id] = true
	}
	source := recSetForIDs(tbl, ids)
	want := []int64{0, 1, 2, 3, 4}

	// The first scan builds the compressed RecID-to-index-position map. The
	// second exercises its heap-backed path because the RecSet exceeds the
	// 1024-entry stack scratch buffer.
	for attempt := range 2 {
		if got := scanOrderedRecSetIDsWithOrder(tbl, source, len(want), order); !equalInt64s(got, want) {
			t.Fatalf("large ordered RecSet attempt %d rows = %v, want %v", attempt+1, got, want)
		}
	}
}

func TestDenseOrderedRecSetKeepsIndexDrivenTraversal(t *testing.T) {
	const rows = 4_000
	tbl := setupAdaptiveRecSetOrderTable(t, "trecsetdenseorder", rows)
	order := buildRankOrderIndex(t, tbl)
	source := tbl.scanRecSet(nil, []string{"id"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewBool(values[0].Int()%10 != 0)
	}), newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil)

	got := scanOrderedRecSetIDsWithOrder(tbl, source, 3, order)
	if want := []int64{rows/2 + 1, rows/2 + 2, rows/2 + 3}; !equalInt64s(got, want) {
		t.Fatalf("dense ordered RecSet rows = %v, want %v", got, want)
	}

	for _, index := range tbl.ActiveShards()[0].Indexes {
		index.mu.Lock()
		inverseCount := index.baseState.mainIndexPositions.count
		inverseBytes := index.baseState.mainIndexPositions.ComputeSize()
		index.mu.Unlock()
		if inverseCount > 0 {
			t.Fatalf("dense ordered RecSet unexpectedly built %d-byte inverse index", inverseBytes)
		}
	}
}

func TestRecSetTraversalCrossoversUseEffectiveIndexSpan(t *testing.T) {
	const universe = int64(800_000)
	if !orderedRecSetDominates(1_024, universe, universe, 72) {
		t.Fatal("sparse LIMIT 72 RecSet should dominate a full ordered index span")
	}
	if orderedRecSetDominates(4_096, universe, universe, 72) {
		t.Fatal("dense LIMIT 72 RecSet should keep the ordered base-index iterator")
	}
	if orderedRecSetDominates(1_024, universe, 1_000, 72) {
		t.Fatal("a narrow access interval should dominate the RecSet sort kernel")
	}
	if !unorderedRecSetDominates(16_384, universe) {
		t.Fatal("sparse unordered scan should iterate its RecSet directly")
	}
	if unorderedRecSetDominates(100_000, 100_000) {
		t.Fatal("unordered scan should retain an equally narrow base-index interval")
	}
}

func TestOrderedRecSetSwitchesAfterResidualFilterRejectsPrefix(t *testing.T) {
	const rows = 20_000
	tbl := setupAdaptiveRecSetOrderTable(t, "trecsetadaptivefilter", rows)
	order := buildRankOrderIndex(t, tbl)
	ids := make(map[int64]bool, 1_024)
	for id := int64(0); id < 1_024; id++ {
		ids[id] = true
	}
	source := recSetForIDs(tbl, ids)
	condition := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewBool(values[0].Int() >= 1_000)
	})

	got := scanOrderedRecSetIDsWithCondition(tbl, source, 3, order, condition)
	if want := []int64{1_000, 1_001, 1_002}; !equalInt64s(got, want) {
		t.Fatalf("adaptive residual-filter rows = %v, want %v", got, want)
	}

	foundInverse := false
	for _, index := range tbl.ActiveShards()[0].Indexes {
		index.mu.Lock()
		foundInverse = foundInverse || index.baseState.mainIndexPositions.count > 0
		index.mu.Unlock()
	}
	if !foundInverse {
		t.Fatal("rejecting residual filter did not switch to the inverse RecSet kernel")
	}
}

func TestOrderedInverseRecSetInterlacesDeltaRows(t *testing.T) {
	tbl := setupAdaptiveRecSetOrderTable(t, "trecsetorderdelta", 100)
	order := buildRankOrderIndex(t, tbl)
	tbl.Insert([]string{"id", "rank"}, [][]scm.Scmer{
		{scm.NewInt(100), scm.NewInt(75)},
		{scm.NewInt(101), scm.NewInt(25)},
	}, nil, scm.NewNil(), false, nil)
	source := recSetForIDs(tbl, map[int64]bool{
		0: true, 50: true, 100: true, 101: true,
	})

	got := scanOrderedRecSetIDsWithOrder(tbl, source, 4, order)
	if want := []int64{50, 101, 0, 100}; !equalInt64s(got, want) {
		t.Fatalf("inverse RecSet main/delta merge rows = %v, want %v", got, want)
	}
}
