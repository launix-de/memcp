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
	"fmt"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func setupScanLookupTable(tb testing.TB, database string, rows [][]scm.Scmer) *table {
	tb.Helper()
	databases.Remove(database)
	tb.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	tbl, _ := CreateTable(database, "items", Memory, true)
	tbl.CreateColumn("key", "INT", nil, nil)
	tbl.CreateColumn("value", "VARCHAR", nil, nil)
	tbl.Insert([]string{"key", "value"}, rows, nil, scm.NewNil(), false, nil)
	return tbl
}

func TestScanLookupReturnsValueNullAndCardinalityError(t *testing.T) {
	tbl := setupScanLookupTable(t, "test_scan_lookup", [][]scm.Scmer{
		{scm.NewInt(1), scm.NewString("one")},
		{scm.NewInt(2), scm.NewNil()},
		{scm.NewInt(3), scm.NewString("first")},
		{scm.NewInt(3), scm.NewString("second")},
	})
	tx := NewTxContext(TxCursorStability)

	if got := tbl.scanLookup(tx, []string{"key"}, []scm.Scmer{scm.NewInt(1)}, "value", true); !scm.Equal(got, scm.NewString("one")) {
		t.Fatalf("scanLookup existing value = %s, want one", scm.String(got))
	}
	if got := tbl.scanLookup(tx, []string{"key"}, []scm.Scmer{scm.NewInt(2)}, "value", true); !got.IsNil() {
		t.Fatalf("scanLookup NULL value = %s, want nil", scm.String(got))
	}
	if got := tbl.scanLookup(tx, []string{"key"}, []scm.Scmer{scm.NewInt(99)}, "value", true); !got.IsNil() {
		t.Fatalf("scanLookup missing value = %s, want nil", scm.String(got))
	}

	defer func() {
		if got := recover(); fmt.Sprint(got) != scalarSubselectOverflow {
			t.Fatalf("scanLookup duplicate panic = %v, want %q", got, scalarSubselectOverflow)
		}
	}()
	tbl.scanLookup(tx, []string{"key"}, []scm.Scmer{scm.NewInt(3)}, "value", true)
}

func TestScanLookupSchemeOperator(t *testing.T) {
	Init(scm.Globalenv)
	tbl := setupScanLookupTable(t, "test_scan_lookup_operator", [][]scm.Scmer{
		{scm.NewInt(7), scm.NewString("seven")},
	})
	got := scm.Apply(
		scm.Globalenv.Vars[scm.Symbol("scan_lookup")],
		scm.NewAny(NewTxContext(TxCursorStability)),
		NewTableScmer(tbl),
		scm.NewSlice([]scm.Scmer{scm.NewString("key")}),
		scm.NewSlice([]scm.Scmer{scm.NewInt(7)}),
		scm.NewString("value"),
	)
	if !scm.Equal(got, scm.NewString("seven")) {
		t.Fatalf("scan_lookup operator = %s, want seven", scm.String(got))
	}
	exists := scm.Apply(
		scm.Globalenv.Vars[scm.Symbol("scan_lookup")],
		scm.NewAny(NewTxContext(TxCursorStability)),
		NewTableScmer(tbl),
		scm.NewSlice([]scm.Scmer{scm.NewString("key")}),
		scm.NewSlice([]scm.Scmer{scm.NewInt(7)}),
	)
	if !scm.ToBool(exists) {
		t.Fatal("scan_lookup existence operator returned false")
	}
}

func TestScanLookupCompositeValueAndExists(t *testing.T) {
	database := "test_scan_lookup_composite"
	databases.Remove(database)
	t.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	tbl, _ := CreateTable(database, "items", Memory, true)
	tbl.CreateColumn("tenant", "INT", nil, nil)
	tbl.CreateColumn("key", "INT", nil, nil)
	tbl.CreateColumn("value", "VARCHAR", nil, nil)
	tbl.Insert([]string{"tenant", "key", "value"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(7), scm.NewString("one-seven")},
		{scm.NewInt(2), scm.NewInt(7), scm.NewString("two-seven")},
		{scm.NewInt(2), scm.NewInt(8), scm.NewString("two-eight")},
	}, nil, scm.NewNil(), false, nil)
	tx := NewTxContext(TxCursorStability)
	cols := []string{"tenant", "key"}

	if got := tbl.scanLookup(tx, cols, []scm.Scmer{scm.NewInt(2), scm.NewInt(7)}, "value", true); !scm.Equal(got, scm.NewString("two-seven")) {
		t.Fatalf("composite scanLookup = %s, want two-seven", scm.String(got))
	}
	if got := tbl.scanLookup(tx, cols, []scm.Scmer{scm.NewInt(9), scm.NewInt(7)}, "", false); scm.ToBool(got) {
		t.Fatal("missing composite existence lookup returned true")
	}
	if got := tbl.scanLookup(tx, cols, []scm.Scmer{scm.NewInt(2), scm.NewInt(7)}, "", false); !scm.ToBool(got) {
		t.Fatal("matching composite existence lookup returned false")
	}
	if got := tbl.scanLookup(tx, []string{"key"}, []scm.Scmer{scm.NewInt(7)}, "", false); !scm.ToBool(got) {
		t.Fatal("existence lookup with multiple matches returned false")
	}
}

func BenchmarkScanLookupWithTx(b *testing.B) {
	rows := make([][]scm.Scmer, 1024)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewString("value")}
	}
	tbl := setupScanLookupTable(b, "bench_scan_lookup", rows)
	tx := NewTxContext(TxCursorStability)
	key := scm.NewInt(511)
	cols := []string{"key"}
	values := []scm.Scmer{key}
	// Warm the adaptive index before measuring the steady-state operator.
	tbl.scanLookup(tx, cols, values, "value", true)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tbl.scanLookup(tx, cols, values, "value", true)
	}
}

func BenchmarkScanLookupDimensions(b *testing.B) {
	database := "bench_scan_lookup_dimensions"
	databases.Remove(database)
	b.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	tbl, _ := CreateTable(database, "items", Memory, true)
	tbl.CreateColumn("tenant", "INT", nil, nil)
	tbl.CreateColumn("key", "INT", nil, nil)
	tbl.CreateColumn("value", "VARCHAR", nil, nil)
	rows := make([][]scm.Scmer, 1024)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i % 16)), scm.NewInt(int64(i)), scm.NewString("value")}
	}
	tbl.Insert([]string{"tenant", "key", "value"}, rows, nil, scm.NewNil(), false, nil)
	tx := NewTxContext(TxCursorStability)
	oneCol := []string{"key"}
	oneValue := []scm.Scmer{scm.NewInt(511)}
	twoCols := []string{"tenant", "key"}
	twoValues := []scm.Scmer{scm.NewInt(15), scm.NewInt(511)}

	cases := []struct {
		name        string
		cols        []string
		values      []scm.Scmer
		resultCol   string
		returnValue bool
	}{
		{name: "value_one", cols: oneCol, values: oneValue, resultCol: "value", returnValue: true},
		{name: "value_two", cols: twoCols, values: twoValues, resultCol: "value", returnValue: true},
		{name: "exists_one", cols: oneCol, values: oneValue},
		{name: "exists_two", cols: twoCols, values: twoValues},
	}
	for _, bench := range cases {
		b.Run(bench.name, func(b *testing.B) {
			tbl.scanLookup(tx, bench.cols, bench.values, bench.resultCol, bench.returnValue)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				tbl.scanLookup(tx, bench.cols, bench.values, bench.resultCol, bench.returnValue)
			}
		})
	}
}
