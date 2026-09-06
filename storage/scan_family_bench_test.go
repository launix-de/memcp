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

// BenchmarkScanFamilyFixedCosts keeps the empty-table setup paths of every
// physical scan operator visible to the allocation profiler. Row throughput is
// covered by the operator-specific benchmarks; these cases isolate dispatch,
// index-iterator scratch, queues and map/reduce setup.
func BenchmarkScanFamilyFixedCosts(b *testing.B) {
	tbl := benchScanTable(b, "family")
	trueFn := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	mapReduceFn := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })
	nilValue := scm.NewNil()
	sortCols := []scm.Scmer{scm.NewString("id")}
	sortDirs := []func(...scm.Scmer) scm.Scmer{scm.OptimizeProcToSerialFunction(scm.Globalenv.Vars[scm.Symbol("<")])}
	orderedSpec := scanOrderTableSpec{
		table:          tbl,
		accessSchema:   newScanAccessSchema(scanAccessConsumerScan, nil, -1),
		condition:      trueFn,
		sortcols:       sortCols,
		callbackCols:   []string{"id"},
		callback:       mapReduceFn,
		perTableOffset: -1,
		perTableLimit:  -1,
	}
	recSetInputTable := benchScanTable(b, "family_recset_input")
	recSetInputTable.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
	RebuildTable(recSetInputTable, true, false)
	recSetInput := recSetInputTable.scanRecSet(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, trueFn)
	identityRecSet := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })

	benchmarks := []struct {
		name string
		run  func()
	}{
		{
			name: "recset",
			run: func() {
				tbl.scanRecSet(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, trueFn)
			},
		},
		{
			name: "recset_input",
			run: func() {
				recSetInput.filterToRecSet(nil, nil, trueFn,
					newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil)
			},
		},
		{
			name: "exists",
			run: func() {
				tbl.scanExists(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, trueFn)
			},
		},
		{
			name: "batch",
			run: func() {
				tbl.scanWithBatch(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{"#0"}, trueFn, []string{"id"}, mapReduceFn,
					nilValue, nilValue, false, 1, []scm.Scmer{scm.NewInt(1)})
			},
		},
		{
			name: "order",
			run: func() {
				tbl.scan_order(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, trueFn, sortCols, sortDirs, 0, 0, 72,
					[]string{"id"}, mapReduceFn, nilValue, false, nilValue, nil, nilValue,
				)
			},
		},
		{
			name: "order_recset",
			run: func() {
				recSetInput.scan_order(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, trueFn, sortCols, sortDirs, 0, 0, 72,
					[]string{"id"}, mapReduceFn, nilValue, false, nilValue, nil, nilValue,
				)
			},
		},
		{
			name: "order_multi",
			run: func() {
				scanOrderMulti(nil, []scanOrderTableSpec{orderedSpec, orderedSpec}, sortDirs,
					0, 0, 72, nilValue, false, nilValue)
			},
		},
		{
			name: "order_batch_accept",
			run: func() {
				scanOrderBatchAccept(nil, orderedSpec, identityRecSet, sortCols, sortDirs,
					0, 0, 72, []string{"id"}, mapReduceFn, nilValue, false, nilValue)
			},
		},
	}

	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmark.run()
			}
		})
	}
}

// BenchmarkScanOrderPlannerCompiledFixedCosts includes optimizer lowering and
// SCM dispatch while retaining the resulting cached procedure across calls.
// It catches regressions where static ORDER BY access requirements fall back to
// invocation-time boundary construction.
func BenchmarkScanOrderPlannerCompiledFixedCosts(b *testing.B) {
	Init(scm.Globalenv)
	tbl := benchScanTable(b, "order_planner_compiled")
	expr := scm.Read(b.Name(), `(lambda (table_value)
		(scan_order nil table_value '() '() '() (lambda () true)
			'("id") (list <) 0 0 72
			'("id") (lambda (acc id) acc) nil false nil '() nil))`)
	proc := scm.Eval(scm.Optimize(expr, &scm.Globalenv, nil), &scm.Globalenv)
	tableValue := NewTableScmer(tbl)
	for range 3 {
		scm.Apply(proc, tableValue)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scm.Apply(proc, tableValue)
	}
}

// BenchmarkScanBatchCoveredFixedCosts isolates the steady-state setup of a
// planner-covered batch scan. Its access schema, batch values and column lists
// are retained exactly as they are in a cached physical plan.
func BenchmarkScanBatchCoveredFixedCosts(b *testing.B) {
	tbl := benchScanTable(b, "batch_covered")
	accessSchema := newScanAccessSchema(scanAccessConsumerCoveredScan, nil, -1)
	trueFn := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	mapReduceFn := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })
	callbackCols := []string{"id"}
	batchdata := []scm.Scmer{scm.NewInt(1)}
	neutral := scm.NewNil()
	tbl.scanWithBatch(nil, accessSchema, nil, nil, trueFn, callbackCols, mapReduceFn,
		neutral, neutral, false, 1, batchdata)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tbl.scanWithBatch(nil, accessSchema, nil, nil, trueFn, callbackCols, mapReduceFn,
			neutral, neutral, false, 1, batchdata)
	}
}
