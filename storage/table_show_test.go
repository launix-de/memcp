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
package storage

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

func showColumnsTestTable(columnCount int) *table {
	columns := make([]*column, columnCount)
	for i := range columns {
		columns[i] = &column{
			Name:          fmt.Sprintf("column_%d", i),
			Typ:           "VARCHAR",
			AllowNull:     true,
			Collation:     "utf8mb4_general_ci",
			Comment:       "original",
			Typdimensions: []int{255},
		}
	}
	return &table{
		Columns: columns,
		Unique:  []uniqueKey{{Id: "PRIMARY", Cols: []string{"column_0"}}},
	}
}

func showColumnProperty(row scm.Scmer, name string) scm.Scmer {
	values := row.Slice()
	for i := 0; i+1 < len(values); i += 2 {
		if scm.String(values[i]) == name {
			return values[i+1]
		}
	}
	return scm.NewNil()
}

func TestShowColumnsReusesImmutableSnapshot(t *testing.T) {
	tbl := showColumnsTestTable(3)
	first := tbl.ShowColumns()
	second := tbl.ShowColumns()

	firstRows := first.Slice()
	secondRows := second.Slice()
	if &firstRows[0] != &secondRows[0] {
		t.Fatal("ShowColumns rebuilt the outer metadata list")
	}
	if &firstRows[0].Slice()[0] != &secondRows[0].Slice()[0] {
		t.Fatal("ShowColumns rebuilt a column metadata row")
	}
}

func TestCountEstimateIsLockFreeAndPersisted(t *testing.T) {
	tbl := showColumnsTestTable(1)
	tbl.PlannerRowEstimate.value.Store(73)

	encoded, err := json.Marshal(tbl)
	if err != nil {
		t.Fatal(err)
	}
	var restored table
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if got := restored.CountEstimate(); got != 73 {
		t.Fatalf("restored CountEstimate() = %d, want 73", got)
	}

	// An estimate read must not consult shard state, even while it is exclusively
	// locked by a writer. This guards the compiler hot path against regressions
	// to GetRead/lazy loading.
	shard := &storageShard{t: tbl}
	tbl.Shards = []*storageShard{shard}
	tbl.mu.Lock()
	tbl.publishTopologyLocked()
	tbl.mu.Unlock()
	shard.mu.Lock()
	done := make(chan uint, 1)
	go func() { done <- tbl.CountEstimate() }()
	select {
	case got := <-done:
		if got != 73 {
			t.Fatalf("CountEstimate() = %d, want 73", got)
		}
	case <-time.After(time.Second):
		t.Fatal("CountEstimate blocked on the shard lock")
	}
	shard.mu.Unlock()
}

func TestCountEstimateFallsBackToSingleShardDeltaWithoutLocking(t *testing.T) {
	dbName := "test_count_estimate_delta_only"
	databases.Remove(dbName)
	t.Cleanup(func() { databases.Remove(dbName) })
	CreateDatabase(dbName, true)
	tbl, _ := CreateTable(dbName, "items", Memory, true)
	tbl.CreateColumn("id", "INT", nil, nil)
	values := [][]scm.Scmer{
		{scm.NewInt(1)},
		{scm.NewInt(2)},
		{scm.NewInt(3)},
	}
	tbl.Insert([]string{"id"}, values, nil, scm.NewNil(), false, nil)

	// Simulate a missing pre-REBUILD table statistic. The fallback must use the
	// atomically published delta length without looking through shard internals.
	tbl.PlannerRowEstimate.value.Store(0)
	shard := tbl.ActiveShards()[0]
	shard.mu.Lock()
	defer shard.mu.Unlock()
	done := make(chan uint, 1)
	go func() { done <- tbl.CountEstimate() }()
	select {
	case got := <-done:
		if got != 3 {
			t.Fatalf("CountEstimate() = %d, want 3 delta rows", got)
		}
	case <-time.After(time.Second):
		t.Fatal("delta-only CountEstimate blocked on the shard lock")
	}
}

func TestCountEstimateDoesNotTreatPersistedRowsAsDeltaOnly(t *testing.T) {
	tbl := showColumnsTestTable(1)
	shard := &storageShard{t: tbl, main_count: 9, srState: WRITE}
	shard.plannerMainRows.Store(9)
	shard.plannerDeltaRows.Store(3)
	tbl.Shards = []*storageShard{shard}
	tbl.mu.Lock()
	tbl.publishTopologyLocked()
	tbl.mu.Unlock()
	tbl.PlannerRowEstimate.value.Store(0)

	if got := tbl.CountEstimate(); got != 0 {
		t.Fatalf("CountEstimate() = %d, want unavailable estimate with main storage", got)
	}
}

func TestLegacyPlannerRowEstimateIsInitializedFromShards(t *testing.T) {
	tbl := showColumnsTestTable(1)
	// The first release containing planner_row_estimate could persist a zero
	// for an existing table before its lazy shards had been counted.
	tbl.PlannerRowEstimate.present.Store(true)
	shard := &storageShard{t: tbl, main_count: 9, srState: WRITE}
	tbl.Shards = []*storageShard{shard}
	tbl.publishTopologyLocked()

	tbl.initializeLegacyPlannerRowEstimate()

	if !tbl.PlannerRowEstimate.present.Load() {
		t.Fatal("legacy planner estimate was not marked initialized")
	}
	if got := tbl.CountEstimate(); got != 9 {
		t.Fatalf("legacy CountEstimate() = %d, want 9", got)
	}
}

func TestPlannerStatisticsUsesHashedImmutableSnapshot(t *testing.T) {
	tbl := showColumnsTestTable(2)
	tbl.Columns[1].Name = "MixedCase"
	tbl.PlannerRowEstimate.value.Store(91)
	tbl.Columns[1].PlannerStats.Store(&columnPlannerStatistics{
		Confidence:        1,
		Source:            "rebuild",
		AverageValueBytes: 12.5,
		MinEstimate:       scm.NewString("a"),
		MaxEstimate:       scm.NewString("z"),
	})
	atomic.StoreUint64(&tbl.Columns[1].DistinctEstimate, 17)
	tbl.publishShowColumnsSnapshot()
	publishedToken := tbl.PlannerStatsToken()
	if publishedToken == 0 {
		t.Fatal("published planner statistics have no dependency token")
	}
	publishedFingerprint := tbl.PlannerStatisticsFingerprint()
	tbl.publishShowColumnsSnapshot()
	if tbl.PlannerStatsToken() == publishedToken {
		t.Fatal("republished planner statistics retained their generation token")
	}
	if got := tbl.PlannerStatisticsFingerprint(); got != publishedFingerprint {
		t.Fatalf("unchanged planner statistics changed fingerprint from %d to %d", publishedFingerprint, got)
	}
	publishedToken = tbl.PlannerStatsToken()

	root := tbl.PlannerStatistics().FastDict()
	rowCount, ok := root.Get(scm.NewString("row_count"))
	if !ok || rowCount.Int() != 91 {
		t.Fatalf("planner row_count = %v, %t; want 91, true", rowCount, ok)
	}
	columnsValue, ok := root.Get(scm.NewString("columns"))
	if !ok {
		t.Fatal("planner snapshot has no columns index")
	}
	columnStats, ok := columnsValue.FastDict().Get(scm.NewString("mixedcase"))
	if !ok {
		t.Fatal("case-folded planner column lookup missed MixedCase")
	}
	if got := scm.ToInt(columnStats.Slice()[3].Slice()[1]); got != 17 {
		t.Fatalf("planner distinct estimate = %d, want 17", got)
	}
	rawType := ""
	for _, property := range columnStats.Slice() {
		pair := property.Slice()
		if len(pair) == 2 && scm.String(pair[0]) == "raw_type" {
			rawType = scm.String(pair[1])
			break
		}
	}
	if rawType != "VARCHAR" {
		t.Fatalf("planner raw type = %q, want VARCHAR", rawType)
	}
	if tbl.PlannerStatistics().FastDict() != root {
		t.Fatal("planner statistics rebuilt an already-published snapshot")
	}
	if got := tbl.PlannerStatsToken(); got != publishedToken {
		t.Fatalf("immutable planner statistics changed token from %d to %d", publishedToken, got)
	}

	tbl.adjustPlannerRows(9)
	adjustedToken := tbl.PlannerStatsToken()
	if adjustedToken == publishedToken {
		t.Fatal("planner row-count update retained a stale dependency token")
	}
	if got := tbl.PlannerStatisticsFingerprint(); got != publishedFingerprint {
		t.Fatalf("same-magnitude row-count update changed fingerprint from %d to %d", publishedFingerprint, got)
	}
	updatedRoot := tbl.PlannerStatistics().FastDict()
	updatedRows, _ := updatedRoot.Get(scm.NewString("row_count"))
	if updatedRows.Int() != 100 {
		t.Fatalf("planner row_count after insert batch = %d, want 100", updatedRows.Int())
	}
	updatedColumns, _ := updatedRoot.Get(scm.NewString("columns"))
	if updatedColumns.FastDict() != columnsValue.FastDict() {
		t.Fatal("row-count update rebuilt the immutable column catalog")
	}

	mapper := &ShardMapReducer{shard: &storageShard{t: tbl}, deletedRows: 4}
	mapper.FlushSideEffects()
	if got := tbl.CountEstimate(); got != 96 {
		t.Fatalf("CountEstimate() after delete batch = %d, want 96", got)
	}
	tbl.adjustPlannerRows(-200)
	if got := tbl.CountEstimate(); got != 0 {
		t.Fatalf("saturated CountEstimate() = %d, want 0", got)
	}
}

func TestPlannerStatisticsFingerprintUsesMagnitudeBuckets(t *testing.T) {
	const columnsFingerprint = uint64(12345)
	if plannerStatisticsFingerprint(91, columnsFingerprint) != plannerStatisticsFingerprint(100, columnsFingerprint) {
		t.Fatal("ordinary row-count growth crossed a planner fingerprint bucket")
	}
	if plannerStatisticsFingerprint(91, columnsFingerprint) == plannerStatisticsFingerprint(182, columnsFingerprint) {
		t.Fatal("doubling the row count retained the planner fingerprint")
	}
	if plannerMagnitudeBucket(17) != plannerMagnitudeBucket(31) {
		t.Fatal("same-magnitude distinct estimates use different buckets")
	}
	if plannerMagnitudeBucket(17) == plannerMagnitudeBucket(34) {
		t.Fatal("doubled distinct estimate retained its magnitude bucket")
	}
	col := &column{Name: "payload", Typ: "VARCHAR"}
	first := &columnPlannerStatistics{
		Confidence: 0.91, Source: "rebuild", NullFraction: 0.11, AverageValueBytes: 12.5,
	}
	nearby := &columnPlannerStatistics{
		Confidence: 0.95, Source: "rebuild", NullFraction: 0.12, AverageValueBytes: 15.9,
	}
	base := plannerColumnFingerprint(42, col, 17, first)
	if got := plannerColumnFingerprint(42, col, 31, nearby); got != base {
		t.Fatalf("nearby column statistics changed cost class from %d to %d", base, got)
	}
	wide := *nearby
	wide.AverageValueBytes = 32
	if got := plannerColumnFingerprint(42, col, 31, &wide); got == base {
		t.Fatal("large average-width change retained its column cost class")
	}
}

func BenchmarkPlannerStatisticsLookup(b *testing.B) {
	tbl := showColumnsTestTable(256)
	tbl.PlannerRowEstimate.value.Store(1_000_000)
	tbl.publishShowColumnsSnapshot()
	target := scm.NewString("column_255")
	plannerColumns, _ := tbl.PlannerStatistics().FastDict().Get(scm.NewString("columns"))
	showColumns := tbl.ShowColumns().Slice()

	b.Run("published_hash", func(b *testing.B) {
		for range b.N {
			if _, ok := plannerColumns.FastDict().Get(target); !ok {
				b.Fatal("planner column disappeared")
			}
		}
	})
	b.Run("legacy_show_linear", func(b *testing.B) {
		for range b.N {
			found := false
			for _, row := range showColumns {
				if scm.String(showColumnProperty(row, "Field")) == "column_255" {
					found = true
					break
				}
			}
			if !found {
				b.Fatal("SHOW column disappeared")
			}
		}
	})
}

func TestFreshBaseColumnIsNotMarkedComputed(t *testing.T) {
	db := &database{Name: "fresh-column-statistics"}
	tbl := &table{schema: db}
	column, ok := tbl.createColumnLocked("value", "TEXT", nil, nil)
	if !ok {
		t.Fatal("createColumnLocked rejected a fresh column")
	}
	if !column.Computor.IsNil() || !column.ComputorFilter.IsNil() {
		t.Fatal("fresh base column contains a non-nil computed expression")
	}
}

func TestShowColumnsPublishesReplacementSnapshot(t *testing.T) {
	tbl := showColumnsTestTable(1)
	before := tbl.ShowColumns()
	tbl.Columns[0].Comment = "updated"
	tbl.publishShowColumnsSnapshot()
	after := tbl.ShowColumns()

	if &before.Slice()[0] == &after.Slice()[0] {
		t.Fatal("metadata update retained the old snapshot")
	}
	if got := scm.String(showColumnProperty(after.Slice()[0], "Comment")); got != "updated" {
		t.Fatalf("updated comment = %q, want updated", got)
	}
}

func TestShowColumnsUsesExplicitStatisticsPublication(t *testing.T) {
	tbl := showColumnsTestTable(1)
	before := tbl.ShowColumns()
	atomic.StoreUint64(&tbl.Columns[0].DistinctEstimate, 17)
	tbl.publishShowColumnsSnapshot()
	after := tbl.ShowColumns()

	if &before.Slice()[0] == &after.Slice()[0] {
		t.Fatal("statistics update retained the old snapshot")
	}
	if got := showColumnProperty(after.Slice()[0], "DistinctEstimate").Int(); got != 17 {
		t.Fatalf("distinct estimate = %d, want 17", got)
	}
}

func TestResolveColumnNameUsesSnapshotIndex(t *testing.T) {
	tbl := showColumnsTestTable(3)
	tbl.Columns[1].Name = "MixedCase"
	tbl.Columns[2].Name = "Σ"
	tbl.publishShowColumnsSnapshot()

	if got, ok := tbl.ResolveColumnName("MixedCase", false); !ok || got != "MixedCase" {
		t.Fatalf("exact lookup = %q, %t; want MixedCase, true", got, ok)
	}
	if _, ok := tbl.ResolveColumnName("mixedcase", false); ok {
		t.Fatal("case-sensitive lookup accepted different case")
	}
	if got, ok := tbl.ResolveColumnName("mixedcase", true); !ok || got != "MixedCase" {
		t.Fatalf("case-insensitive lookup = %q, %t; want MixedCase, true", got, ok)
	}
	if _, ok := tbl.ResolveColumnName("missing", true); ok {
		t.Fatal("missing column resolved")
	}
	if got, ok := tbl.ResolveColumnName("ς", true); !ok || got != "Σ" {
		t.Fatalf("Unicode fold lookup = %q, %t; want Σ, true", got, ok)
	}

	tbl.Columns[1].Name = "Renamed"
	tbl.invalidateShowColumnsSnapshot()
	if _, ok := tbl.ResolveColumnName("MixedCase", false); ok {
		t.Fatal("invalidated column name remained visible")
	}
	if got, ok := tbl.ResolveColumnName("Renamed", false); !ok || got != "Renamed" {
		t.Fatalf("replacement lookup = %q, %t; want Renamed, true", got, ok)
	}
}

func BenchmarkShowColumnsCached(b *testing.B) {
	tbl := showColumnsTestTable(64)
	_ = tbl.ShowColumns()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = tbl.ShowColumns()
	}
}

func BenchmarkResolveColumnNameCached(b *testing.B) {
	tbl := showColumnsTestTable(256)
	tbl.publishColumnNamesSnapshot()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = tbl.ResolveColumnName("column_255", false)
	}
}
