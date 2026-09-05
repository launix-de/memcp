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
	"strings"
	"testing"

	"github.com/launix-de/memcp/scm"
)

const scanColumnCostRows = 65536

func scanColumnCostTable(b testing.TB, name string, rows int) (*table, []string) {
	b.Helper()
	dbName := "bench_scan_column_cost_" + name
	databases.Remove(dbName)
	b.Cleanup(func() { databases.Remove(dbName) })
	CreateDatabase(dbName, true)
	tbl, _ := CreateTable(dbName, "items", Memory, true)
	cols := make([]string, 16)
	for i := range cols {
		cols[i] = fmt.Sprintf("c%d", i)
		tbl.CreateColumn(cols[i], "INT", nil, nil)
	}
	if rows == 0 {
		return tbl, cols
	}
	values := make([][]scm.Scmer, rows)
	for row := range values {
		values[row] = make([]scm.Scmer, len(cols))
		for col := range cols {
			values[row][col] = scm.NewInt(int64(row + col))
		}
	}
	tbl.Insert(cols, values, nil, scm.NewNil(), false, nil)
	RebuildTable(tbl, true, false)
	return tbl, cols
}

func TestEstimateFilteredRowsReportsExaminedSample(t *testing.T) {
	tbl, cols := scanColumnCostTable(t, "estimate_sample", 10)
	condition := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		return scm.NewBool(args[0].Int()%2 == 0)
	})
	shards := tbl.ActiveShards()
	if len(shards) != 1 {
		t.Fatalf("active shards = %d, want 1", len(shards))
	}
	accessSchema, accessValues := newScanAccessSchema(scanAccessConsumerScan, nil, -1), []scm.Scmer(nil)
	estimate := shards[0].EstimateFilteredRows(cols[:1], condition, 3, nil, accessSchema, accessValues)
	if estimate.rows != 3 || !estimate.capped || estimate.examined != 5 ||
		estimate.population != "table_rows" || estimate.coverage != "sampled" {
		t.Fatalf("estimate = %+v; want 3 sampled matches after 5 table rows", estimate)
	}
}

func TestEstimateFilteredRowsRecognizesCompleteCappedIndexRange(t *testing.T) {
	tbl, cols := scanColumnCostTable(t, "estimate_index_population", 10)
	condition := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("c0")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("equal?"), scm.NewSymbol("c0"), scm.NewInt(4),
		}),
		En: &scm.Globalenv,
	})
	shard := tbl.ActiveShards()[0]
	bounds := extractBoundaries(cols[:1], condition)
	lower, upperLast := indexFromBoundaries(bounds)
	var buf [16]uint32
	for range 2 {
		shard.mu.RLock()
		shard.iterateIndex(nil, scanAccess{suffix: bounds}, lower, upperLast, len(shard.inserts), buf[:], 1, nil,
			func([]uint32) bool { return true })
		shard.mu.RUnlock()
	}

	accessSchema, accessValues := testEqualScanAccess("c0", scm.NewInt(4))
	estimate := shard.EstimateFilteredRows(cols[:1], condition, 1, nil, accessSchema, accessValues)
	if estimate.rows != 1 || estimate.capped || estimate.examined != 1 ||
		estimate.population != "index_candidates" || estimate.coverage != "exact" {
		t.Fatalf("estimate = %+v; want complete one-row index range", estimate)
	}
}

func TestScanSelectivityEstimateScalesIndexCandidatesByShardPopulation(t *testing.T) {
	Init(scm.Globalenv)
	dbName := "test_selectivity_index_population"
	databases.Remove(dbName)
	t.Cleanup(func() { databases.Remove(dbName) })
	CreateDatabase(dbName, true)
	tbl, _ := CreateTable(dbName, "items", Memory, true)
	tbl.CreateColumn("tenant", "INT", nil, nil)

	values := make([][]scm.Scmer, 1000)
	for row := range values {
		values[row] = []scm.Scmer{scm.NewInt(int64(row % 10))}
	}
	tbl.Insert([]string{"tenant"}, values, nil, scm.NewNil(), false, nil)
	RebuildTable(tbl, true, false)

	condition := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("tenant")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("equal?"), scm.NewSymbol("tenant"), scm.NewInt(4),
		}),
		En: &scm.Globalenv,
	})
	shard := tbl.ActiveShards()[0]
	for range 2 {
		bounds := extractBoundaries([]string{"tenant"}, condition)
		lower, upperLast := indexFromBoundaries(bounds)
		var buf [128]uint32
		shard.mu.RLock()
		shard.iterateIndex(nil, scanAccess{suffix: bounds}, lower, upperLast, len(shard.inserts), buf[:], 100, nil,
			func([]uint32) bool { return true })
		shard.mu.RUnlock()
	}
	accessSchema, accessValues := testEqualScanAccess("tenant", scm.NewInt(4))
	shardEstimate := shard.EstimateFilteredRows([]string{"tenant"}, condition, 512, nil, accessSchema, accessValues)
	if shardEstimate.population != "index_candidates" || shardEstimate.examined != 100 {
		t.Fatalf("shard estimate = %+v, want 100 index candidates", shardEstimate)
	}
	boundedEstimate := shard.EstimateFilteredRows([]string{"tenant"}, condition, 50, nil, accessSchema, accessValues)
	if boundedEstimate.rows != 100 || boundedEstimate.capped ||
		boundedEstimate.population != "index_candidates" || boundedEstimate.coverage != "upper_bound" {
		t.Fatalf("bounded shard estimate = %+v, want 100-row index upper bound", boundedEstimate)
	}

	estimate := scm.Apply(scm.Globalenv.Vars[scm.Symbol("scan_selectivity_estimate")],
		scm.NewNil(), NewTableScmer(tbl),
		accessSchema, scm.NewSlice(accessValues),
		scm.NewSlice([]scm.Scmer{scm.NewString("tenant")}), condition, scm.NewInt(512))
	fields := mustScmerSlice(estimate, "selectivity estimate")
	fieldInt := func(name string) int64 {
		for _, field := range fields {
			pair := mustScmerSlice(field, "selectivity estimate field")
			if scm.String(pair[0]) == name {
				return int64(scm.ToInt(pair[1]))
			}
		}
		t.Fatalf("missing selectivity estimate field %q", name)
		return 0
	}
	if got := fieldInt("rows"); got != 100 {
		t.Fatalf("estimated rows = %d, want 100", got)
	}
	if got := fieldInt("sampled"); got != 1000 {
		t.Fatalf("sampled population = %d, want 1000", got)
	}
}

func TestScanSelectivityEstimateLoadsColdPersistentShard(t *testing.T) {
	Init(scm.Globalenv)
	tbl, persistence := createDurabilityTestTable(t, "test_selectivity_cold_shard", 100)
	RebuildTable(tbl, true, false)

	db := newDatabase()
	db.Name = "test_selectivity_cold_shard"
	db.persistence = persistence
	db.srState = COLD
	db.ensureLoaded()
	coldTable := db.GetTable("items")
	if coldTable == nil {
		t.Fatal("reloaded database has no items table")
	}
	coldShard := coldTable.ActiveShards()[0]
	if coldShard.srState != COLD {
		t.Fatalf("reloaded shard state = %v, want COLD", coldShard.srState)
	}

	condition := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("id")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("<="), scm.NewSymbol("id"), scm.NewInt(50),
		}),
		En: &scm.Globalenv,
	})
	accessSchema, accessValues := testUpperScanAccess("id", scm.NewInt(50), true)
	estimate := scm.Apply(scm.Globalenv.Vars[scm.Symbol("scan_selectivity_estimate")],
		scm.NewNil(), NewTableScmer(coldTable),
		accessSchema, scm.NewSlice(accessValues),
		scm.NewSlice([]scm.Scmer{scm.NewString("id")}), condition, scm.NewInt(512))
	fields := mustScmerSlice(estimate, "cold selectivity estimate")
	values := make(map[string]int64, len(fields))
	for _, field := range fields {
		pair := mustScmerSlice(field, "cold selectivity estimate field")
		if pair[1].IsInt() {
			values[scm.String(pair[0])] = int64(scm.ToInt(pair[1]))
		}
	}
	if values["rows"] != 50 || values["sampled"] != 100 || values["input"] != 100 {
		t.Fatalf("cold selectivity estimate = %v, want rows=50 sampled=100 input=100", values)
	}
}

func TestCountEstimateSumsUnevenShards(t *testing.T) {
	oldShardSize := Settings.ShardSize
	Settings.ShardSize = 3
	t.Cleanup(func() { Settings.ShardSize = oldShardSize })

	tbl, _ := scanColumnCostTable(t, "count_estimate_uneven", 7)
	if shards := len(tbl.ActiveShards()); shards < 2 {
		t.Fatalf("active shards = %d, want a multi-shard table", shards)
	}
	if got := tbl.CountEstimate(); got != 7 {
		t.Fatalf("CountEstimate() = %d, want exact uneven-shard total 7", got)
	}
}

func benchmarkScanColumnMatrix(b *testing.B, name string, rows int, filterPasses bool, varyFilter bool) {
	tbl, allCols := scanColumnCostTable(b, name, rows)
	condition := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(filterPasses) })
	callback := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })
	for _, count := range []int{0, 1, 2, 4, 8, 16} {
		filterCols := []string(nil)
		mapCols := []string(nil)
		if varyFilter {
			filterCols = allCols[:count]
		} else {
			mapCols = allCols[:count]
		}
		b.Run(fmt.Sprintf("cols=%02d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tbl.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, filterCols, condition, mapCols, callback,
					scm.NewNil(), scm.NewNil(), false)
			}
		})
	}
}

func BenchmarkScanColumnReaderSetupFilter(b *testing.B) {
	benchmarkScanColumnMatrix(b, "setup_filter", 0, false, true)
}

func BenchmarkScanColumnReaderSetupMap(b *testing.B) {
	benchmarkScanColumnMatrix(b, "setup_map", 0, true, false)
}

func BenchmarkScanFilterColumnCost(b *testing.B) {
	benchmarkScanColumnMatrix(b, "filter", scanColumnCostRows, false, true)
}

func BenchmarkScanMapColumnCost(b *testing.B) {
	benchmarkScanColumnMatrix(b, "map", scanColumnCostRows, true, false)
}

func BenchmarkScanMapColumnCostBySelectivity(b *testing.B) {
	tbl, allCols := scanColumnCostTable(b, "map_selectivity", scanColumnCostRows)
	callback := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })
	for _, percent := range []int{1, 10, 50, 100} {
		threshold := int64(scanColumnCostRows * percent / 100)
		condition := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
			return scm.NewBool(args[0].Int() < threshold)
		})
		for _, count := range []int{0, 4, 16} {
			mapCols := allCols[:count]
			b.Run(fmt.Sprintf("selectivity=%03d/cols=%02d", percent, count), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					tbl.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, allCols[:1], condition, mapCols, callback,
						scm.NewNil(), scm.NewNil(), false)
				}
			})
		}
	}
}

func scanTypedColumnCostTable(b *testing.B, name string, typ string, rows int, value func(int, int) scm.Scmer) (*table, []string) {
	b.Helper()
	dbName := "bench_scan_column_type_" + name
	databases.Remove(dbName)
	b.Cleanup(func() { databases.Remove(dbName) })
	CreateDatabase(dbName, true)
	tbl, _ := CreateTable(dbName, "items", Memory, true)
	cols := make([]string, 8)
	for i := range cols {
		cols[i] = fmt.Sprintf("c%d", i)
		tbl.CreateColumn(cols[i], typ, nil, nil)
	}
	values := make([][]scm.Scmer, rows)
	for row := range values {
		values[row] = make([]scm.Scmer, len(cols))
		for col := range cols {
			values[row][col] = value(row, col)
		}
	}
	tbl.Insert(cols, values, nil, scm.NewNil(), false, nil)
	RebuildTable(tbl, true, false)
	return tbl, cols
}

func BenchmarkScanColumnCostByType(b *testing.B) {
	const rows = 32768
	types := []struct {
		name  string
		typ   string
		value func(int, int) scm.Scmer
	}{
		{name: "int", typ: "INT", value: func(row, col int) scm.Scmer {
			return scm.NewInt(int64(row + col))
		}},
		{name: "short_string", typ: "VARCHAR", value: func(row, col int) scm.Scmer {
			return scm.NewString(fmt.Sprintf("%08x", row*17+col))
		}},
		{name: "long_string", typ: "VARCHAR", value: func(row, col int) scm.Scmer {
			return scm.NewString(fmt.Sprintf("%08x-%s", row*17+col, strings.Repeat(string(rune('a'+col)), 112)))
		}},
	}
	falseFn := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(false) })
	trueFn := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	callback := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewNil() })
	for _, kind := range types {
		tbl, cols := scanTypedColumnCostTable(b, kind.name, kind.typ, rows, kind.value)
		for _, phase := range []string{"filter", "map"} {
			for _, count := range []int{0, 1, 4, 8} {
				filterCols := []string(nil)
				mapCols := []string(nil)
				condition := trueFn
				if phase == "filter" {
					filterCols = cols[:count]
					condition = falseFn
				} else {
					mapCols = cols[:count]
				}
				b.Run(fmt.Sprintf("type=%s/phase=%s/cols=%02d", kind.name, phase, count), func(b *testing.B) {
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						tbl.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, filterCols, condition, mapCols, callback,
							scm.NewNil(), scm.NewNil(), false)
					}
				})
			}
		}
	}
}

func scanColumnCostPredicate(b *testing.B, terms int, distinctCols bool) scm.Scmer {
	b.Helper()
	parts := make([]string, terms)
	params := []string{"c0"}
	if distinctCols {
		params = make([]string, terms)
	}
	for i := range parts {
		col := "c0"
		if distinctCols {
			col = fmt.Sprintf("c%d", i)
			params[i] = col
		}
		if i%2 == 0 {
			parts[i] = fmt.Sprintf("(> %s %d)", col, -i-1)
		} else {
			parts[i] = fmt.Sprintf("(< %s %d)", col, scanColumnCostRows+i)
		}
	}
	source := fmt.Sprintf("(lambda (%s) (and %s))", strings.Join(params, " "), strings.Join(parts, " "))
	return scm.Eval(scm.Optimize(scm.Read("scan column cost benchmark", source), &scm.Globalenv, nil), &scm.Globalenv)
}

func BenchmarkScanFilterExpressionCost(b *testing.B) {
	tbl, cols := scanColumnCostTable(b, "filter_expression", scanColumnCostRows)
	callback := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })
	for _, shape := range []string{"same_column", "distinct_columns"} {
		for _, terms := range []int{1, 2, 4, 8, 16} {
			distinctCols := shape == "distinct_columns"
			condition := scanColumnCostPredicate(b, terms, distinctCols)
			filterCols := cols[:1]
			if distinctCols {
				filterCols = cols[:terms]
			}
			b.Run(fmt.Sprintf("shape=%s/terms=%02d", shape, terms), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					tbl.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, filterCols, condition, nil, callback,
						scm.NewNil(), scm.NewNil(), false)
				}
			})
		}
	}
}
