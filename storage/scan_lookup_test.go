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

	if got := tbl.scanLookup(tx, "key", scm.NewInt(1), "value"); !scm.Equal(got, scm.NewString("one")) {
		t.Fatalf("scanLookup existing value = %s, want one", scm.String(got))
	}
	if got := tbl.scanLookup(tx, "key", scm.NewInt(2), "value"); !got.IsNil() {
		t.Fatalf("scanLookup NULL value = %s, want nil", scm.String(got))
	}
	if got := tbl.scanLookup(tx, "key", scm.NewInt(99), "value"); !got.IsNil() {
		t.Fatalf("scanLookup missing value = %s, want nil", scm.String(got))
	}

	defer func() {
		if got := recover(); fmt.Sprint(got) != scalarSubselectOverflow {
			t.Fatalf("scanLookup duplicate panic = %v, want %q", got, scalarSubselectOverflow)
		}
	}()
	tbl.scanLookup(tx, "key", scm.NewInt(3), "value")
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
		scm.NewString("key"),
		scm.NewInt(7),
		scm.NewString("value"),
	)
	if !scm.Equal(got, scm.NewString("seven")) {
		t.Fatalf("scan_lookup operator = %s, want seven", scm.String(got))
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
	// Warm the adaptive index before measuring the steady-state operator.
	tbl.scanLookup(tx, "key", key, "value")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tbl.scanLookup(tx, "key", key, "value")
	}
}
