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
	estimate := shards[0].EstimateFilteredRows(cols[:1], condition, 3, nil)
	if estimate.rows != 3 || !estimate.capped || estimate.examined != 5 ||
		estimate.population != "table_rows" || estimate.coverage != "sampled" {
		t.Fatalf("estimate = %+v; want 3 sampled matches after 5 table rows", estimate)
	}
}

func TestEstimateFilteredRowsMarksCappedIndexPopulationAsLowerBound(t *testing.T) {
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
		shard.iterateIndex(nil, bounds, lower, upperLast, len(shard.inserts), buf[:], 1, nil,
			func([]uint32) bool { return true })
		shard.mu.RUnlock()
	}

	estimate := shard.EstimateFilteredRows(cols[:1], condition, 1, nil)
	if estimate.rows != 1 || !estimate.capped || estimate.examined != 1 ||
		estimate.population != "index_candidates" || estimate.coverage != "lower_bound" {
		t.Fatalf("estimate = %+v; want capped index lower bound", estimate)
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
	callback := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewNil() })
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
				tbl.scan(nil, filterCols, condition, mapCols, callback,
					scm.NewNil(), scm.NewNil(), scm.NewNil(), false)
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
	callback := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewNil() })
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
					tbl.scan(nil, allCols[:1], condition, mapCols, callback,
						scm.NewNil(), scm.NewNil(), scm.NewNil(), false)
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
						tbl.scan(nil, filterCols, condition, mapCols, callback,
							scm.NewNil(), scm.NewNil(), scm.NewNil(), false)
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
	callback := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewNil() })
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
					tbl.scan(nil, filterCols, condition, nil, callback,
						scm.NewNil(), scm.NewNil(), scm.NewNil(), false)
				}
			})
		}
	}
}
