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
		{scm.NewNil(), scm.NewString("null-key")},
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
	if got := tbl.scanLookup(tx, []string{"key"}, []scm.Scmer{scm.NewNil()}, "", false); scm.ToBool(got) {
		t.Fatal("scanLookup matched a SQL NULL key")
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
		scm.NewSlice([]scm.Scmer{scm.NewString("scan_lookup_v1"), scm.NewInt(1), scm.NewString("key"), scm.NewString("value"), scm.NewInt(1), scm.NewString("value")}),
		scm.NewSlice([]scm.Scmer{scm.NewInt(7)}),
	)
	if !scm.Equal(got, scm.NewString("seven")) {
		t.Fatalf("scan_lookup operator = %s, want seven", scm.String(got))
	}
	exists := scm.Apply(
		scm.Globalenv.Vars[scm.Symbol("scan_lookup")],
		scm.NewAny(NewTxContext(TxCursorStability)),
		NewTableScmer(tbl),
		scm.NewSlice([]scm.Scmer{scm.NewString("scan_lookup_v1"), scm.NewInt(1), scm.NewString("key"), scm.NewString("exists"), scm.NewInt(0)}),
		scm.NewSlice([]scm.Scmer{scm.NewInt(7)}),
	)
	if !scm.ToBool(exists) {
		t.Fatal("scan_lookup existence operator returned false")
	}
}

func TestScanLookupPlanBindingDoesNotAllocate(t *testing.T) {
	schema := scm.NewSlice([]scm.Scmer{
		scm.NewString("scan_lookup_v1"), scm.NewInt(2),
		scm.NewString("tenant"), scm.NewString("key"),
		scm.NewString("exists"), scm.NewInt(0),
	})
	values := scm.NewSlice([]scm.Scmer{scm.NewInt(1), scm.NewInt(7)})
	if allocs := testing.AllocsPerRun(1000, func() {
		_ = parseScanLookupPlan(schema, values)
	}); allocs != 0 {
		t.Fatalf("scan_lookup plan binding allocated %.2f times per run, want 0", allocs)
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

func TestScanLookupMapReturnsComputedValueAfterCardinalityCheck(t *testing.T) {
	database := "test_scan_lookup_map"
	databases.Remove(database)
	t.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	tbl, _ := CreateTable(database, "items", Memory, true)
	tbl.CreateColumn("tenant", "INT", nil, nil)
	tbl.CreateColumn("key", "INT", nil, nil)
	tbl.CreateColumn("left_value", "INT", nil, nil)
	tbl.CreateColumn("right_value", "INT", nil, nil)
	tbl.Insert([]string{"tenant", "key", "left_value", "right_value"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(7), scm.NewInt(10), scm.NewInt(20)},
		{scm.NewInt(1), scm.NewInt(8), scm.NewInt(30), scm.NewInt(40)},
		{scm.NewInt(2), scm.NewInt(8), scm.NewInt(50), scm.NewInt(60)},
		{scm.NewInt(3), scm.NewInt(9), scm.NewInt(70), scm.NewInt(80)},
		{scm.NewInt(3), scm.NewInt(9), scm.NewInt(90), scm.NewInt(100)},
	}, nil, scm.NewNil(), false, nil)
	tx := NewTxContext(TxCursorStability)
	calls := 0
	mapper := scm.PrepareSerialProc(scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		calls++
		return scm.NewInt(values[0].Int() + values[1].Int())
	}))

	got := tbl.scanLookupMap(tx,
		[]string{"tenant", "key"},
		[]scm.Scmer{scm.NewInt(1), scm.NewInt(7)},
		[]string{"left_value", "right_value"},
		&mapper,
	)
	if !scm.Equal(got, scm.NewInt(30)) || calls != 1 {
		t.Fatalf("scanLookupMap = %s with %d mapper calls, want 30 with one call", scm.String(got), calls)
	}
	RebuildTable(tbl, true, false)
	got = tbl.scanLookupMap(tx,
		[]string{"tenant", "key"},
		[]scm.Scmer{scm.NewInt(1), scm.NewInt(7)},
		[]string{"left_value", "right_value"},
		&mapper,
	)
	if !scm.Equal(got, scm.NewInt(30)) || calls != 2 {
		t.Fatalf("rebuilt scanLookupMap = %s with %d mapper calls, want 30 with two", scm.String(got), calls)
	}
	if got := tbl.scanLookupMap(tx,
		[]string{"key"}, []scm.Scmer{scm.NewInt(99)},
		[]string{"left_value", "right_value"}, &mapper,
	); !got.IsNil() || calls != 2 {
		t.Fatalf("missing scanLookupMap = %s with %d total mapper calls, want nil and two", scm.String(got), calls)
	}
	if got := tbl.scanLookupMap(tx,
		[]string{"key"}, []scm.Scmer{scm.NewNil()},
		[]string{"left_value", "right_value"}, &mapper,
	); !got.IsNil() || calls != 2 {
		t.Fatalf("NULL-key scanLookupMap = %s with %d total mapper calls, want nil and two", scm.String(got), calls)
	}

	func() {
		defer func() {
			if got := recover(); fmt.Sprint(got) != scalarSubselectOverflow {
				t.Fatalf("duplicate scanLookupMap panic = %v, want %q", got, scalarSubselectOverflow)
			}
		}()
		tbl.scanLookupMap(tx,
			[]string{"tenant", "key"},
			[]scm.Scmer{scm.NewInt(3), scm.NewInt(9)},
			[]string{"left_value", "right_value"},
			&mapper,
		)
	}()
	if calls != 2 {
		t.Fatalf("duplicate scanLookupMap invoked mapper; total calls = %d, want 2", calls)
	}
}

func TestCompiledScanLookupMapCanPerformNestedLookup(t *testing.T) {
	Init(scm.Globalenv)
	tbl := setupScanLookupTable(t, "test_scan_lookup_map_operator", [][]scm.Scmer{
		{scm.NewInt(7), scm.NewString("seven")},
	})
	operator := scm.Globalenv.Vars[scm.Symbol("scan_lookup")]
	tx := NewTxContext(TxCursorStability)
	txValue := scm.NewAny(tx)
	tableValue := NewTableScmer(tbl)
	mapper := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.Apply(
			scm.Globalenv.Vars[scm.Symbol("scan_lookup")],
			txValue,
			tableValue,
			scm.NewSlice([]scm.Scmer{scm.NewString("scan_lookup_v1"), scm.NewInt(1), scm.NewString("key"), scm.NewString("value"), scm.NewInt(1), scm.NewString("value")}),
			scm.NewSlice([]scm.Scmer{scm.NewInt(7)}),
		)
	})
	got := scm.Apply(operator,
		txValue,
		tableValue,
		scm.NewSlice([]scm.Scmer{scm.NewString("scan_lookup_v1"), scm.NewInt(1), scm.NewString("key"), scm.NewString("map"), scm.NewInt(1), scm.NewString("value")}),
		scm.NewSlice([]scm.Scmer{scm.NewInt(7), mapper}),
	)
	if !scm.Equal(got, scm.NewString("seven")) {
		t.Fatalf("nested compiled scan_lookup operator = %s, want seven", scm.String(got))
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

func BenchmarkCompiledScanLookupWithTx(b *testing.B) {
	rows := make([][]scm.Scmer, 1024)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewString("value")}
	}
	tbl := setupScanLookupTable(b, "bench_compiled_scan_lookup", rows)
	tx := NewTxContext(TxCursorStability)
	schema := scm.NewSlice([]scm.Scmer{
		scm.NewString("scan_lookup_v1"), scm.NewInt(1), scm.NewString("key"),
		scm.NewString("value"), scm.NewInt(1), scm.NewString("value"),
	})
	values := scm.NewSlice([]scm.Scmer{scm.NewInt(511)})
	executeCompiledScanLookup(tbl, tx, schema, values)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		executeCompiledScanLookup(tbl, tx, schema, values)
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

func BenchmarkScanLookupMap(b *testing.B) {
	database := "bench_scan_lookup_map"
	databases.Remove(database)
	b.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	tbl, _ := CreateTable(database, "items", Memory, true)
	tbl.CreateColumn("key", "INT", nil, nil)
	tbl.CreateColumn("left_value", "INT", nil, nil)
	tbl.CreateColumn("right_value", "INT", nil, nil)
	rows := make([][]scm.Scmer, 1024)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewInt(int64(i + 1)), scm.NewInt(int64(i + 2))}
	}
	tbl.Insert([]string{"key", "left_value", "right_value"}, rows, nil, scm.NewNil(), false, nil)
	tx := NewTxContext(TxCursorStability)
	lookupCols := []string{"key"}
	lookupValues := []scm.Scmer{scm.NewInt(511)}
	mapCols := []string{"left_value", "right_value"}
	mapProgram := scm.PrepareSerialProc(scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewInt(values[0].Int() + values[1].Int())
	}))
	tbl.scanLookupMap(tx, lookupCols, lookupValues, mapCols, &mapProgram)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tbl.scanLookupMap(tx, lookupCols, lookupValues, mapCols, &mapProgram)
	}
}

func BenchmarkCompiledScanLookupMap(b *testing.B) {
	database := "bench_compiled_scan_lookup_map"
	databases.Remove(database)
	b.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	tbl, _ := CreateTable(database, "items", Memory, true)
	tbl.CreateColumn("key", "INT", nil, nil)
	tbl.CreateColumn("left_value", "INT", nil, nil)
	tbl.CreateColumn("right_value", "INT", nil, nil)
	rows := make([][]scm.Scmer, 1024)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewInt(int64(i + 1)), scm.NewInt(int64(i + 2))}
	}
	tbl.Insert([]string{"key", "left_value", "right_value"}, rows, nil, scm.NewNil(), false, nil)
	tx := NewTxContext(TxCursorStability)
	mapper := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewInt(values[0].Int() + values[1].Int())
	})
	schema := scm.NewSlice([]scm.Scmer{
		scm.NewString("scan_lookup_v1"), scm.NewInt(1), scm.NewString("key"),
		scm.NewString("map"), scm.NewInt(2), scm.NewString("left_value"), scm.NewString("right_value"),
	})
	values := scm.NewSlice([]scm.Scmer{scm.NewInt(511), mapper})
	executeCompiledScanLookup(tbl, tx, schema, values)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		executeCompiledScanLookup(tbl, tx, schema, values)
	}
}
