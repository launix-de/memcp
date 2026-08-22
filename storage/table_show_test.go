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
	"fmt"
	"sync/atomic"
	"testing"

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

func TestShowColumnsRefreshesChangedStatistics(t *testing.T) {
	tbl := showColumnsTestTable(1)
	before := tbl.ShowColumns()
	atomic.StoreUint64(&tbl.Columns[0].DistinctEstimate, 17)
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
