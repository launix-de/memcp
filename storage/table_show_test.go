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
	"sync"
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

func TestShowColumnBorrowsPublishedDescription(t *testing.T) {
	tbl := showColumnsTestTable(2)
	tbl.Columns[0].Name = "MixedCase"
	tbl.Columns[1].Name = "Σ"
	tbl.publishShowColumnsSnapshot(false)
	rows := tbl.ShowColumns().Slice()
	for _, name := range []string{"MixedCase", "mixedcase", "MIXEDCASE"} {
		got := tbl.ShowColumn(name, true)
		if got.IsNil() || &got.Slice()[0] != &rows[0].Slice()[0] {
			t.Fatalf("lookup %q did not borrow the published column description", name)
		}
	}
	if !tbl.ShowColumn("mixedcase", false).IsNil() || !tbl.ShowColumn("missing", true).IsNil() {
		t.Fatal("exact or missing lookup returned a different column")
	}
	if got := tbl.ShowColumn("ς", true); got.IsNil() || showColumnProperty(got, "Field").String() != "Σ" {
		t.Fatal("column description lookup disagrees with native Unicode name resolution")
	}
	if !((*table)(nil)).ShowColumn("missing", true).IsNil() {
		t.Fatal("nil table returned a column description")
	}
}

func TestShowColumnReadsWithoutTableOrShardOwnership(t *testing.T) {
	tbl := showColumnsTestTable(1)
	tbl.publishShowColumnsSnapshot(false)
	shard := &storageShard{t: tbl}
	tbl.Shards = []*storageShard{shard}
	tbl.ddlMu.Lock()
	defer tbl.ddlMu.Unlock()
	tbl.mu.Lock()
	defer tbl.mu.Unlock()
	shard.mu.Lock()
	defer shard.mu.Unlock()
	done := make(chan scm.Scmer, 1)
	go func() { done <- tbl.ShowColumn("column_0", true) }()
	select {
	case got := <-done:
		if got.IsNil() {
			t.Fatal("published column description is missing")
		}
	case <-time.After(time.Second):
		t.Fatal("column metadata lookup waited for table or shard ownership")
	}
}

func TestShowColumnInvalidationPublishesCurrentDeclaration(t *testing.T) {
	tbl := showColumnsTestTable(1)
	tbl.publishShowColumnsSnapshot(false)
	before := tbl.ShowColumn("column_0", false)
	tbl.Columns[0].Name = "replacement"
	tbl.Columns[0].Typdimensions = []int{512}
	tbl.invalidateShowColumnsSnapshot()
	if !tbl.ShowColumn("column_0", false).IsNil() {
		t.Fatal("old name survived metadata invalidation")
	}
	after := tbl.ShowColumn("replacement", false)
	if after.IsNil() || showColumnProperty(after, "Type").String() != "VARCHAR(512)" {
		t.Fatal("lookup needs lazy rendering or retained the old declaration")
	}
	if showColumnProperty(before, "Field").String() != "column_0" {
		t.Fatal("publication changed the retained previous description")
	}
}

func TestShowColumnExactNameWinsFoldedCollision(t *testing.T) {
	tbl := showColumnsTestTable(2)
	tbl.Columns[0].Name = "Name"
	tbl.Columns[1].Name = "name"
	tbl.Columns[1].Typdimensions = []int{31}
	tbl.publishShowColumnsSnapshot(false)
	if got := showColumnProperty(tbl.ShowColumn("name", true), "Type").String(); got != "VARCHAR(31)" {
		t.Fatalf("exact name lost to a folded collision: %s", got)
	}
	if got := showColumnProperty(tbl.ShowColumn("NAME", true), "Field").String(); got != "Name" {
		t.Fatalf("folded lookup changed native first-declaration resolution: %s", got)
	}
}

func TestShowColumnKeepsOnePublicationDuringReplacement(t *testing.T) {
	tbl := showColumnsTestTable(1)
	before := tbl.buildShowColumnsSnapshot(0, nil)
	tbl.Columns[0].Typdimensions = []int{31}
	tbl.Columns[0].AllowNull = false
	after := tbl.buildShowColumnsSnapshot(0, nil)
	tbl.showColumnsSnapshot.Store(before)
	var readers sync.WaitGroup
	for range 4 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for range 1000 {
				row := tbl.ShowColumn("column_0", false)
				typ := showColumnProperty(row, "Type").String()
				nullable := showColumnProperty(row, "Null").Bool()
				if !((typ == "VARCHAR(255)" && nullable) || (typ == "VARCHAR(31)" && !nullable)) {
					t.Errorf("mixed metadata publication: %s, nullable=%t", typ, nullable)
					return
				}
			}
		}()
	}
	for range 1000 {
		tbl.showColumnsSnapshot.Store(after)
		tbl.showColumnsSnapshot.Store(before)
	}
	readers.Wait()
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
	shard := &storageShard{t: tbl, main_count: 9}
	shard.setState(WRITE)
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
	shard := &storageShard{t: tbl, main_count: 9}
	shard.setState(WRITE)
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
	tbl.publishShowColumnsSnapshot(false)
	publishedToken := tbl.PlannerStatsToken(true)
	if publishedToken == 0 {
		t.Fatal("published planner statistics have no dependency token")
	}
	publishedFingerprint := tbl.PlannerStatisticsFingerprint(true)
	tbl.publishShowColumnsSnapshot(false)
	if tbl.PlannerStatsToken(true) == publishedToken {
		t.Fatal("republished planner statistics retained their generation token")
	}
	if got := tbl.PlannerStatisticsFingerprint(true); got != publishedFingerprint {
		t.Fatalf("unchanged planner statistics changed fingerprint from %d to %d", publishedFingerprint, got)
	}
	publishedToken = tbl.PlannerStatsToken(true)

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
	if got := tbl.PlannerStatsToken(true); got != publishedToken {
		t.Fatalf("immutable planner statistics changed token from %d to %d", publishedToken, got)
	}

	tbl.adjustPlannerRows(9)
	adjustedToken := tbl.PlannerStatsToken(true)
	if adjustedToken == publishedToken {
		t.Fatal("planner row-count update retained a stale dependency token")
	}
	if got := tbl.PlannerStatisticsFingerprint(true); got != publishedFingerprint {
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
	tbl.publishShowColumnsSnapshot(false)
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
	column, ok := tbl.createColumnLocked("value", "TEXT", nil, nil, false)
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
	tbl.publishShowColumnsSnapshot(false)
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
	tbl.publishShowColumnsSnapshot(false)
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
	tbl.publishShowColumnsSnapshot(false)

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

func TestShowColumnsBoundsIntegerDistinctByPublishedDomain(t *testing.T) {
	cases := []struct {
		name           string
		typ            string
		min, max       int64
		nulls          uint64
		estimate, want uint64
	}{
		{"repeated keys", "INT", 1, 32, 0, 8192, 32},
		{"nullable negative keys", "BIGINT", -3, 3, 1, 8192, 8},
		{"sparse domain", "INT", 1, 1000, 0, 17, 17},
		{"full signed range", "BIGINT", -1 << 63, 1<<63 - 1, 0, 17, 17},
		{"nullable almost full range", "BIGINT", -1 << 63, 1<<63 - 2, 1, 17, 17},
		{"decimal domain is not integral", "DECIMAL", 1, 32, 0, 8192, 8192},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tbl := showColumnsTestTable(1)
			col := tbl.Columns[0]
			col.Typ = tc.typ
			tbl.PlannerRowEstimate.value.Store(tc.estimate)
			atomic.StoreUint64(&col.DistinctEstimate, tc.estimate)
			col.PlannerStats.Store(&columnPlannerStatistics{
				Confidence: 1, Source: "rebuild", NullCount: tc.nulls,
				MinEstimate: scm.NewInt(tc.min), MaxEstimate: scm.NewInt(tc.max),
			})
			tbl.publishShowColumnsSnapshot(false)
			if got := showColumnProperty(tbl.ShowColumns().Slice()[0], "DistinctEstimate").Int(); got != int64(tc.want) {
				t.Fatalf("SHOW distinct = %d, want %d", got, tc.want)
			}
			columns, _ := tbl.PlannerStatistics().FastDict().Get(scm.NewString("columns"))
			stats, _ := columns.FastDict().Get(scm.NewString(col.Name))
			var got scm.Scmer
			for _, pair := range stats.Slice() {
				if pair.Slice()[0].String() == "distinct" {
					got = pair.Slice()[1]
				}
			}
			if got.Int() != int64(tc.want) {
				t.Fatalf("planner distinct = %d, want %d", got.Int(), tc.want)
			}
			if got := col.distinctEstimateFor(tbl); uint64(got) != tc.want {
				t.Fatalf("direct distinct = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestUniqueNullPolicySchemaRoundTrip(t *testing.T) {
	tbl := showColumnsTestTable(1)
	tbl.Unique = []uniqueKey{
		{Id: "equal_nulls", Cols: []string{"column_0"}, NullsNotDistinct: true},
		{Id: "distinct_nulls", Cols: []string{"column_0"}},
	}
	encoded, err := json.Marshal(tbl)
	if err != nil {
		t.Fatal(err)
	}
	var restored table
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if !restored.Unique[0].NullsNotDistinct || restored.Unique[1].NullsNotDistinct {
		t.Fatal("schema round trip changed the per-key NULL policy")
	}
	// Decode old metadata into a fresh object, as schema loading does.
	var legacy uniqueKey
	if err := json.Unmarshal([]byte(`{"Id":"legacy","Cols":["column_0"]}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.NullsNotDistinct {
		t.Fatal("legacy metadata must retain distinct NULL values")
	}
}

func TestUniqueConstraintNameSchemaRoundTrip(t *testing.T) {
	tbl := showColumnsTestTable(1)
	tbl.Unique[0].ConstraintName = "declared_pk"
	encoded, err := json.Marshal(tbl)
	if err != nil {
		t.Fatal(err)
	}
	var restored table
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	key := restored.Unique[0]
	if key.Id != "PRIMARY" || key.ConstraintName != "declared_pk" || !key.matchesName("DECLARED_PK") || !key.matchesName("primary") || key.matchesName("other") {
		t.Fatalf("restored key lost its native ID or declaration name: %+v", key)
	}
	var legacy uniqueKey
	if err := json.Unmarshal([]byte(`{"Id":"PRIMARY","Cols":["column_0"]}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.ConstraintName != "" || !legacy.matchesName("PRIMARY") || legacy.matchesName("") {
		t.Fatal("legacy keys must retain their original name resolution")
	}
}

func TestConstraintMetadataOwnsArraysAndRetainsInvalidatedColumns(t *testing.T) {
	tbl := showColumnsTestTable(2)
	tbl.Name = "child"
	tbl.Unique[0].ConstraintName = "pk_public"
	tbl.Unique[0].NullsNotDistinct = true
	tbl.Foreign = []foreignKey{{Id: "fk", Tbl1: "child", Cols1: []string{"column_1"}, Tbl2: "parent", Cols2: []string{"id"}}}
	tbl.publishShowColumnsSnapshot(true)
	first := tbl.Constraints()
	key := showColumnProperty(first, "Unique").Slice()[0]
	fk := showColumnProperty(first, "ForeignKeys").Slice()[0]
	tbl.Unique[0].Cols[0] = "renamed_key_column"
	tbl.Foreign[0].Cols1[0] = "renamed_child_column"
	if showColumnProperty(key, "Cols").Slice()[0].String() != "column_0" ||
		showColumnProperty(fk, "LocalColumns").Slice()[0].String() != "column_1" ||
		showColumnProperty(key, "constraint_name").String() != "pk_public" || !showColumnProperty(key, "nulls_not_distinct").Bool() {
		t.Fatal("borrowed constraint value changed with its native declaration")
	}
	tbl.publishShowColumnsSnapshot(true)
	second := tbl.Constraints()
	tbl.invalidateShowColumnsSnapshot()
	if got := tbl.Constraints(); &got.Slice()[0] != &second.Slice()[0] {
		t.Fatal("temporary column invalidation erased immutable constraints")
	}
	tbl.ShowColumns()
	tbl.PlannerStatistics()
	if got := tbl.Constraints(); &got.Slice()[0] != &second.Slice()[0] {
		t.Fatal("lazy column rendering regenerated the constraint directory")
	}
}

func TestConstraintMetadataMaintenanceCannotReplaceNewDeclaration(t *testing.T) {
	tbl := showColumnsTestTable(1)
	tbl.Name = "child"
	tbl.publishShowColumnsSnapshot(true)
	old := tbl.showColumnsSnapshot.Load()
	maintenance := tbl.buildShowColumnsSnapshot(17, nil)
	tbl.Foreign = []foreignKey{{Id: "fk", Tbl1: "child", Cols1: []string{"column_0"}, Tbl2: "parent", Cols2: []string{"id"}}}
	tbl.publishShowColumnsSnapshot(true)
	if tbl.showColumnsSnapshot.CompareAndSwap(old, maintenance) {
		t.Fatal("maintenance replaced a newer constraint publication")
	}
	tbl.publishShowColumnsSnapshot(false)
	if len(showColumnProperty(tbl.Constraints(), "ForeignKeys").Slice()) != 1 {
		t.Fatal("maintenance rendering lost a current constraint")
	}
	encoded, err := json.Marshal(tbl)
	if err != nil {
		t.Fatal(err)
	}
	var restored table
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	restored.publishShowColumnsSnapshot(true)
	if len(showColumnProperty(restored.Constraints(), "ForeignKeys").Slice()) != 1 {
		t.Fatal("restoring native declarations did not reconstruct metadata")
	}
}

func TestConstraintMetadataConcurrentForeignPublicationAndMaintenance(t *testing.T) {
	tbl := showColumnsTestTable(1)
	tbl.Name = "child"
	tbl.schema = &database{Name: "constraint_publication"}
	tbl.publishShowColumnsSnapshot(true)
	var workers sync.WaitGroup
	var published atomic.Int64
	published.Store(-1)
	workers.Add(7)
	for reader := 0; reader < 4; reader++ {
		go func() {
			defer workers.Done()
			for i := 0; i < 1000; i++ {
				minimum := published.Load()
				value := tbl.Constraints()
				foreign := showColumnProperty(value, "ForeignKeys").Slice()
				if minimum >= 0 && (len(foreign) != 1 || int64(scm.ToInt(showColumnProperty(foreign[0], "Id"))) < minimum) {
					t.Error("invalidation restored an older FK declaration")
					return
				}
				for _, key := range showColumnProperty(value, "Unique").Slice() {
					if showColumnProperty(key, "Cols").Slice()[0].String() != "column_0" {
						t.Error("reader observed an incomplete key declaration")
						return
					}
				}
			}
		}()
	}
	go func() {
		defer workers.Done()
		for i := 0; i < 200; i++ {
			// Native FK endpoint mutation owns schemalock, not a parent DDL lock.
			tbl.schema.schemalock.Lock()
			tbl.Foreign = []foreignKey{{Id: fmt.Sprint(i), Tbl1: "child", Cols1: []string{"column_0"}, Tbl2: "parent", Cols2: []string{"id"}}}
			tbl.publishShowColumnsSnapshot(true)
			published.Store(int64(i))
			tbl.schema.schemalock.Unlock()
		}
	}()
	go func() {
		defer workers.Done()
		for i := 0; i < 200; i++ {
			tbl.ddlMu.RLock()
			tbl.publishShowColumnsSnapshot(false)
			tbl.ddlMu.RUnlock()
		}
	}()
	go func() {
		defer workers.Done()
		for i := 0; i < 1000; i++ {
			tbl.ddlMu.Lock()
			tbl.invalidateShowColumnsSnapshot()
			tbl.ddlMu.Unlock()
		}
	}()
	workers.Wait()
	if got := showColumnProperty(tbl.Constraints(), "ForeignKeys").Slice(); len(got) != 1 || showColumnProperty(got[0], "Id").String() != "199" {
		t.Fatal("maintenance lost the final FK declaration")
	}
}

func TestConstraintMetadataNativeDDLPublication(t *testing.T) {
	child := setupScanParallelTestTable(t, "constraint_native_ddl")
	parent, _ := CreateTable(child.schema.Name, "parent", Memory, false)
	parent.CreateColumn("id", "INT", nil, nil)
	create := scm.Globalenv.Vars["createforeignkey"]
	drop := scm.Globalenv.Vars["dropforeignkey"]
	for iteration := 0; iteration < 4; iteration++ {
		name := fmt.Sprintf("extra_%d", iteration)
		var writers sync.WaitGroup
		writers.Add(2)
		go func() {
			defer writers.Done()
			scm.Apply(create, NewTableScmer(child), scm.NewString("fk"), showStringSlice([]string{"id"}),
				NewTableScmer(parent), showStringSlice([]string{"id"}), scm.NewString("restrict"), scm.NewString("restrict"))
		}()
		go func() {
			defer writers.Done()
			child.CreateColumn(name, "INT", nil, nil)
		}()
		writers.Wait()
		for _, endpoint := range []*table{child, parent} {
			if len(showColumnProperty(endpoint.Constraints(), "ForeignKeys").Slice()) != 1 {
				t.Fatal("native FK or column publication lost endpoint metadata")
			}
		}
		scm.Apply(drop, NewTableScmer(child), scm.NewString("fk"))
		if len(showColumnProperty(child.Constraints(), "ForeignKeys").Slice()) != 0 ||
			len(showColumnProperty(parent.Constraints(), "ForeignKeys").Slice()) != 0 {
			t.Fatal("native DROP FOREIGN KEY did not publish both endpoints")
		}
	}
}

func columnDeclarationTestTuple(row scm.Scmer) scm.Scmer {
	return scm.NewSlice([]scm.Scmer{
		showColumnProperty(row, "RawType"), showColumnProperty(row, "Dimensions"),
		showColumnProperty(row, "Null"), showColumnProperty(row, "Collation"),
	})
}

func TestAlterColumnDeclarationRetainsOldContractAndPersistence(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "tcolumndeclarationpersist", 3)
	tbl.CreateColumn("label", "VARCHAR", []int{3}, []scm.Scmer{scm.NewString("default"), scm.NewString("old"), scm.NewString("null"), scm.NewBool(false)})
	old := tbl.columnDeclarations()[2]
	before := tbl.ShowColumn("label", false)
	declaration := scm.NewSlice([]scm.Scmer{scm.NewString("VARCHAR"), scm.NewSlice([]scm.Scmer{scm.NewInt(25)}), scm.NewBool(true), scm.NewString(old.Collation)})
	if !tbl.AlterColumnDeclaration("label", declaration, columnDeclarationTestTuple(before)) {
		t.Fatal("declaration replacement failed")
	}
	if old.Typdimensions[0] != 3 || old.AllowNull || old.sanitizer(scm.NewString("abcdef")).String() != "abc" {
		t.Fatal("ALTER changed the contract retained by an earlier native batch")
	}
	after := tbl.ShowColumn("label", false)
	if showColumnProperty(after, "Dimensions").Slice()[0].Int() != 25 || !showColumnProperty(after, "Null").Bool() || showColumnProperty(after, "Default").String() != "old" {
		t.Fatal("ALTER did not publish the complete declaration with its original default")
	}
	tbl.Insert([]string{"id", "payload", "label"}, [][]scm.Scmer{{scm.NewInt(4), scm.NewString("inserted"), scm.NewString("longer text")}}, nil, scm.NewNil(), false, nil)
	restored := reloadTableFromPersistence(t, "tcolumndeclarationpersist", persistence)
	row := restored.ShowColumn("label", false)
	if showColumnProperty(row, "Dimensions").Slice()[0].Int() != 25 || !showColumnProperty(row, "Null").Bool() || showColumnProperty(row, "Default").String() != "old" {
		t.Fatal("complete declaration did not survive persistence reload")
	}
}

func TestAlterColumnDeclarationChecksExpectedContractBeforeMutation(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tcolumndeclarationprecondition", 0)
	before := tbl.ShowColumn("payload", false)
	expected := columnDeclarationTestTuple(before)
	declaration := scm.NewSlice([]scm.Scmer{scm.NewString("TEXT"), scm.NewSlice(nil), scm.NewBool(false), showColumnProperty(before, "Collation")})
	tbl.AlterColumnDeclaration("payload", declaration, expected)
	panicked := false
	func() {
		defer func() { panicked = recover() != nil }()
		tbl.AlterColumnDeclaration("payload", expected, expected)
	}()
	if !panicked || showColumnProperty(tbl.ShowColumn("payload", false), "Null").Bool() {
		t.Fatal("stale compatibility proof overwrote a concurrent declaration")
	}
	// Failed publication must release DDL and schema ownership.
	tbl.AlterColumnDeclaration("payload", expected, declaration)
}

func TestAlterColumnDeclarationOverlapsNativeInsertAndPointReaders(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tcolumndeclarationconcurrent", 0)
	tbl.CreateColumn("label", "VARCHAR", []int{1}, nil)
	started := make(chan struct{})
	done := make(chan any, 2)
	go func() {
		defer func() { done <- recover() }()
		close(started)
		for id := int64(1); id <= 40; id++ {
			tbl.Insert([]string{"id", "payload", "label"}, [][]scm.Scmer{{scm.NewInt(id), scm.NewString("payload"), scm.NewString("x")}}, nil, scm.NewNil(), false, nil)
			if row := tbl.ShowColumn("label", false); showColumnProperty(row, "RawType").String() != "VARCHAR" || len(showColumnProperty(row, "Dimensions").Slice()) != 1 {
				panic("mixed column declaration")
			}
		}
	}()
	<-started
	go func() {
		defer func() { done <- recover() }()
		for iteration := 0; iteration < 40; iteration++ {
			tbl.AlterColumn("label", "comment", scm.NewString(fmt.Sprintf("description-%d", iteration)))
		}
	}()
	for width := int64(2); width <= 20; width++ {
		before := tbl.ShowColumn("label", false)
		declaration := scm.NewSlice([]scm.Scmer{scm.NewString("VARCHAR"), scm.NewSlice([]scm.Scmer{scm.NewInt(width)}), scm.NewBool(true), showColumnProperty(before, "Collation")})
		tbl.AlterColumnDeclaration("label", declaration, columnDeclarationTestTuple(before))
	}
	for range 2 {
		if failure := <-done; failure != nil {
			t.Fatalf("overlapping native insert/legacy ALTER failed: %v", failure)
		}
	}
	assertInitializedColumn(t, tbl, "label", scm.NewString("x"), 40)
}

func TestShowColumnMetadataCountsBorrowedDefaultOnce(t *testing.T) {
	tbl := showColumnsTestTable(1)
	tbl.publishShowColumnsSnapshot(false)
	before := tbl.metadataMemory()
	value := scm.NewString(fmt.Sprintf("%040000d", 1))
	tbl.Columns[0].Default = value
	tbl.publishShowColumnsSnapshot(false)
	after := tbl.metadataMemory()
	if delta := after - before; delta < uint(len(value.String())) || delta > uint(len(value.String())+64) {
		t.Fatalf("one borrowed default charged %d bytes, want one retained payload", delta)
	}
	tbl.Columns[0].InitialValue = value
	tbl.Columns[0].InitialValues = map[string]columnInitialExtent{"original": {}}
	shared := tbl.metadataMemory()
	if shared <= after || shared-after >= uint(len(value.String())) {
		t.Fatal("the same frozen default was charged again for its ADD extent")
	}
	if row := tbl.ShowColumn("column_0", false); showColumnProperty(row, "Default") != value {
		t.Fatal("point description no longer borrows the column default")
	}
}
