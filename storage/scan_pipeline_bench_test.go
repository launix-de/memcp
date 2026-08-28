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

func optimizedScanProc(tb testing.TB, source string) scm.Scmer {
	tb.Helper()
	return scm.Eval(scm.Optimize(scm.Read("scan pipeline benchmark", source), &scm.Globalenv, nil), &scm.Globalenv)
}

func TestScanPipelineSpecializationsPreserveMainAndDeltaResults(t *testing.T) {
	const dbName = "test_scan_pipeline_specializations"
	databases.Remove(dbName)
	CreateDatabase(dbName, true)
	tbl, _ := CreateTable(dbName, "items", Memory, true)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.Insert([]string{"id"}, [][]scm.Scmer{
		{scm.NewInt(1)},
		{scm.NewInt(2)},
		{scm.NewInt(3)},
	}, nil, scm.NewNil(), false, nil)

	constantOne := optimizedScanProc(t, "(lambda () 1)")
	identity := optimizedScanProc(t, "(lambda (value) value)")
	greaterEqualTwo := optimizedScanProc(t, "(lambda (value) (>= value 2))")
	reversedLessThan := optimizedScanProc(t, "(lambda (value) (< 1 value))")
	plus := scm.Globalenv.Vars[scm.Symbol("+")]
	sqlSum := scm.Globalenv.Vars[scm.Symbol("sql_sum_reduce")]

	assertResults := func(stage string) {
		t.Helper()
		for _, condition := range []scm.Scmer{greaterEqualTwo, reversedLessThan} {
			count := tbl.scan(nil, []string{"id"}, condition, nil, constantOne,
				plus, scm.NewInt(0), scm.NewNil(), false)
			if got := scm.ToInt(count); got != 2 {
				t.Fatalf("%s count = %d, want 2", stage, got)
			}
			sum := tbl.scan(nil, []string{"id"}, condition, []string{"id"}, identity,
				sqlSum, scm.NewNil(), scm.NewNil(), false)
			if got := scm.ToInt(sum); got != 5 {
				t.Fatalf("%s sum = %d, want 5", stage, got)
			}
		}
	}

	assertResults("delta")
	result := GetDatabase(dbName).rebuild(true, false, true)
	if len(result.errors) > 0 {
		t.Fatalf("rebuild errors: %v", result.errors)
	}
	assertResults("main")
}

// BenchmarkScanPipelineOLTP records the common physical callback shapes emitted
// for OLTP projections and aggregates. Keep the callbacks as optimized Scheme
// procedures: native Go benchmark callbacks would bypass the adapter and hide
// the setup and interpreter work this benchmark is intended to measure.
func BenchmarkScanPipelineOLTP(b *testing.B) {
	const rowsN = 60_000
	dbName := "bench_scan_pipeline_oltp"
	databases.Remove(dbName)
	CreateDatabase(dbName, true)
	tbl, _ := CreateTable(dbName, "items", Memory, true)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("amount", "INT", nil, nil)
	rows := make([][]scm.Scmer, rowsN)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewInt(int64(i%100 + 1))}
	}
	tbl.Insert([]string{"id", "amount"}, rows, nil, scm.NewNil(), false, nil)
	result := GetDatabase(dbName).rebuild(true, false, true)
	if len(result.errors) > 0 {
		b.Fatalf("rebuild errors: %v", result.errors)
	}

	trueFilter := optimizedScanProc(b, "(lambda () true)")
	lastIDFilter := optimizedScanProc(b, "(lambda (id) (equal?? id 59999))")
	constantOne := optimizedScanProc(b, "(lambda () 1)")
	identity := optimizedScanProc(b, "(lambda (value) value)")
	takeRight := optimizedScanProc(b, "(lambda (acc value) value)")
	plus := scm.Globalenv.Vars[scm.Symbol("+")]
	sqlSum := scm.Globalenv.Vars[scm.Symbol("sql_sum_reduce")]

	benchmarks := []struct {
		name          string
		conditionCols []string
		condition     scm.Scmer
		mapCols       []string
		mapper        scm.Scmer
		reducer       scm.Scmer
		neutral       scm.Scmer
	}{
		{name: "count", condition: trueFilter, mapper: constantOne, reducer: plus, neutral: scm.NewInt(0)},
		{name: "filtered_count", conditionCols: []string{"id"}, condition: lastIDFilter, mapper: constantOne, reducer: plus, neutral: scm.NewInt(0)},
		{name: "sum", condition: trueFilter, mapCols: []string{"amount"}, mapper: identity, reducer: sqlSum, neutral: scm.NewNil()},
		{name: "take_right", condition: trueFilter, mapCols: []string{"amount"}, mapper: identity, reducer: takeRight, neutral: scm.NewNil()},
	}

	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tbl.scan(nil, benchmark.conditionCols, benchmark.condition, benchmark.mapCols, benchmark.mapper,
					benchmark.reducer, benchmark.neutral, scm.NewNil(), false)
			}
		})
	}
}
