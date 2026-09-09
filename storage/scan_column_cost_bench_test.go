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
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/launix-de/memcp/scm"
)

const scanColumnCostRows = 60000

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
	var buf [16]uint32
	for range 2 {
		shard.mu.RLock()
		shard.iterateIndex(nil, runtimeScanAccess(bounds), len(shard.inserts), buf[:], 1, nil,
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
		var buf [128]uint32
		shard.mu.RLock()
		shard.iterateIndex(nil, runtimeScanAccess(bounds), len(shard.inserts), buf[:], 100, nil,
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

// Same workload on the development baseline and candidate: full residual
// filtering, precompiled metadata, no per-iteration AST compilation or I/O.
func BenchmarkCompleteFilterFeedback(b *testing.B) {
	Init(scm.Globalenv)
	tbl, cols := scanColumnCostTable(b, "filter_feedback", 65536)
	filter := scm.Read("feedback-benchmark", "(lambda (x) (not (equal? (mod x 10) 0)))")
	schema, values, _ := compileScanAccess(scm.NewSlice([]scm.Scmer{scm.NewString(cols[0])}), filter)
	for i := range values {
		values[i] = scm.Eval(values[i], &scm.Globalenv)
	}
	condition := scm.Eval(scm.Optimize(filter, &scm.Globalenv, nil), &scm.Globalenv)
	mapper := scm.Globalenv.Vars[scm.Symbol("scan_count")]
	combine := scm.Globalenv.Vars[scm.Symbol("+")]
	run := func() { tbl.scan(nil, schema, values, cols[:1], condition, nil, mapper, scm.NewInt(0), combine, false) }
	run()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		run()
	}
}

func BenchmarkScanFilterSpectrum(b *testing.B) {
	for _, rows := range []int{1, 8, 64, 1024, 8192, 60000} {
		tbl, cols := scanColumnCostTable(b, fmt.Sprintf("filter_spectrum_%d", rows), rows)
		for _, shape := range []struct {
			name      string
			source    string
			columnCnt int
		}{
			{name: "simple", source: "(lambda (a) (> a -1))", columnCnt: 1},
			{name: "arithmetic", source: "(lambda (a b c) (> (+ a (* b c)) -1))", columnCnt: 3},
		} {
			condition := scm.Eval(scm.Optimize(scm.Read("scan filter spectrum", shape.source), &scm.Globalenv, nil), &scm.Globalenv)
			callback := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })
			b.Run(fmt.Sprintf("rows=%05d/shape=%s", rows, shape.name), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					tbl.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
						cols[:shape.columnCnt], condition, nil, callback, scm.NewNil(), scm.NewNil(), false)
				}
			})
		}
	}
}

func TestJITTypedFilterBufferCompactsRecordIDs(t *testing.T) {
	if !scm.JITEnabled() {
		t.Skip("requires the JIT experiment")
	}
	costs := scm.CurrentJITCosts()
	t.Logf("filter calibration: compile %.0f ns, scalar %.2f ns/row, fused %.2f ns/row, minimum %d rows",
		costs.FilterBufferCompileNS, costs.FilterBufferCurrentNS, costs.FilterBufferFusedNS, costs.FilterBufferMinimumRows)
	predicate := optimizedScanProc(t, "(lambda (a b) (and (> a 1) (< b 8)))")
	proc := predicate.Proc()
	if proc == nil {
		t.Fatal("optimized predicate has no JIT procedure")
	}
	kernel := scm.CompileJITFilterBuffer(proc, []uint8{scm.TagInt, scm.TagInt})
	if kernel == nil {
		t.Fatal("typed filter buffer did not compile")
	}
	if count := kernel(nil, nil); count != 0 {
		t.Fatalf("empty filter returned %d records", count)
	}
	ids := []uint32{10, 11, 12, 13}
	values := []scm.Scmer{
		scm.NewInt(1), scm.NewInt(7),
		scm.NewInt(2), scm.NewInt(8),
		scm.NewInt(3), scm.NewInt(6),
		scm.NewInt(4), scm.NewInt(5),
	}
	if count := kernel(ids, values); count != 2 || ids[0] != 12 || ids[1] != 13 {
		t.Fatalf("filtered ids = %v count=%d, want [12 13] count=2", ids, count)
	}
}

func BenchmarkFilterBufferLocalCompile(b *testing.B) {
	if !scm.JITEnabled() {
		b.Skip("requires JIT")
	}
	for _, shape := range []struct {
		name, source string
		width        int
	}{
		{"simple", "(lambda (a) (> a 127))", 1},
		{"arithmetic", "(lambda (a b c) (> (+ a (* b c)) 127))", 3},
	} {
		proc := benchmarkMapReduceFusionProc(b, shape.source).Proc()
		for _, typed := range []bool{false, true} {
			tags := make([]uint8, shape.width)
			for i := range tags {
				tags[i] = scm.JITTypeUnknown
				if typed {
					tags[i] = scm.TagInt
				}
			}
			b.Run(fmt.Sprintf("%s/typed=%v", shape.name, typed), func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					kernel := scm.CompileJITFilterBuffer(proc, tags)
					if kernel == nil {
						b.Fatal("compile failed")
					}
					runtime.KeepAlive(kernel)
				}
			})
		}
	}
}

func BenchmarkFilterBufferLocalScan(b *testing.B) {
	if !scm.JITEnabled() {
		b.Skip("requires JIT")
	}
	readers := make([]ColumnReader, 16)
	emitters := make([]scm.JITStorageGetValueEmitter, len(readers))
	valueTypes := make([]uint8, len(readers))
	for i := range readers {
		values := make([]scm.Scmer, 60000)
		for row := range values {
			values[row] = scm.NewInt(int64(row + i))
		}
		storage := buildStorageInt(values)
		readers[i] = newCachedColumnReaderTx(storage, nil)
		emitters[i] = storage.JITEmit
		valueTypes[i] = storage.JITValueType()
	}
	for _, shape := range []struct {
		name, source string
		width        int
	}{
		{"simple", "(lambda (a) (> a 127))", 1},
		{"arithmetic", "(lambda (a b c) (> (+ a (* b c)) 127))", 3},
		{"distinct16", "(lambda (c0 c1 c2 c3 c4 c5 c6 c7 c8 c9 c10 c11 c12 c13 c14 c15) (and (> c0 -1) (< c1 60001) (> c2 -3) (< c3 60003) (> c4 -5) (< c5 60005) (> c6 -7) (< c7 60007) (> c8 -9) (< c9 60009) (> c10 -11) (< c11 60011) (> c12 -13) (< c13 60013) (> c14 -15) (< c15 60015)))", 16},
	} {
		proc := benchmarkMapReduceFusionProc(b, shape.source)
		for _, rows := range []int{0, 1, 64, 1024, 8192, 60000} {
			for _, mode := range []string{"lambda", "buffer-dynamic", "buffer-mixed", "buffer-typed", "direct-typed"} {
				b.Run(fmt.Sprintf("%s/rows=%d/%s", shape.name, rows, mode), func(b *testing.B) {
					ids := make([]uint32, defaultScanBufferSize)
					args := make([]scm.Scmer, shape.width)
					want := 0
					for row := 0; row < rows; row++ {
						for col := range args {
							args[col] = readers[col].GetValue(uint32(row))
						}
						if scm.ToBool(scm.Apply(proc, args...)) {
							want++
						}
					}
					tags := make([]uint8, shape.width)
					for i := range tags {
						tags[i] = scm.JITTypeUnknown
						if mode == "buffer-typed" || mode == "direct-typed" || (mode == "buffer-mixed" && i%2 == 0) {
							tags[i] = valueTypes[i]
						}
					}
					b.ReportAllocs()
					b.ResetTimer()
					for range b.N {
						filter := typedScanFilter{width: shape.width}
						if mode != "lambda" {
							if mode == "direct-typed" {
								filter.kernel = scm.CompileJITFilterStorage(proc.Proc(), tags, emitters[:shape.width])
							} else {
								filter.kernel = scm.CompileJITFilterBuffer(proc.Proc(), tags)
							}
							if filter.kernel == nil {
								b.Fatal("compile failed")
							}
							filter.multiFuncs = make([]scm.JITStorageGetValueMultiFunc, shape.width)
							for i := range filter.multiFuncs {
								filter.multiFuncs[i] = compiledColumnGetValueMulti(readers[i])
							}
						}
						count := 0
						for offset := 0; offset < rows; offset += len(ids) {
							batch := ids[:min(len(ids), rows-offset)]
							for i := range batch {
								batch[i] = uint32(offset + i)
							}
							if filter.kernel != nil {
								if mode == "direct-typed" {
									count += filter.kernel(batch, nil)
									continue
								}
								count += filter.filterMain(batch, readers[:shape.width])
								continue
							}
							for _, id := range batch {
								for i := range args {
									args[i] = readers[i].GetValue(id)
								}
								if scm.ToBool(scm.Apply(proc, args...)) {
									count++
								}
							}
						}
						filter.close()
						if count != want {
							b.Fatalf("got %d matches, want %d", count, want)
						}
						runtime.KeepAlive(count)
					}
				})
			}
		}
	}
}

func TestFilterBufferComparisonWidths(t *testing.T) {
	if !scm.JITEnabled() {
		t.Skip("requires JIT")
	}
	for width := 1; width <= 16; width++ {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			params, terms := make([]string, width), make([]string, width)
			tags := make([]uint8, width)
			for i := range params {
				params[i] = fmt.Sprintf("c%d", i)
				terms[i] = fmt.Sprintf("(> c%d %d)", i, -i-1)
				if i%2 == 1 {
					terms[i] = fmt.Sprintf("(< c%d %d)", i, 60000+i)
				}
				tags[i] = scm.TagInt
			}
			proc := benchmarkMapReduceFusionProc(t, fmt.Sprintf("(lambda (%s) (and %s))", strings.Join(params, " "), strings.Join(terms, " ")))
			kernel := scm.CompileJITFilterBuffer(proc.Proc(), tags)
			if kernel == nil {
				t.Fatal("compile failed")
			}
			ids := make([]uint32, 1024)
			batch := make([]scm.Scmer, len(ids)*width)
			var want []uint32
			for row := range ids {
				ids[row] = uint32(row)
				for col := range tags {
					batch[row*width+col] = scm.NewInt(int64(row + col))
				}
				if row%3 == 0 {
					col := (row / 3) % width
					boundary := -col - 1
					if col%2 == 1 {
						boundary = 60000 + col
					}
					batch[row*width+col] = scm.NewInt(int64(boundary))
				} else {
					want = append(want, uint32(row))
				}
			}
			if got := kernel(ids, batch); got != len(want) {
				t.Fatalf("got %d, want %d", got, len(want))
			}
			for i, id := range want {
				if ids[i] != id {
					t.Fatalf("row %d: got ID %d, want %d", i, ids[i], id)
				}
			}
		})
	}
}

func TestFilterBufferWideArithmetic(t *testing.T) {
	if !scm.JITEnabled() {
		t.Skip("requires JIT")
	}
	for _, width := range []int{1, 3, 8, 16} {
		params := make([]string, width)
		tags := make([]uint8, width)
		for i := range params {
			params[i] = fmt.Sprintf("c%d", i)
			tags[i] = scm.TagInt
		}
		proc := benchmarkMapReduceFusionProc(t, fmt.Sprintf("(lambda (%s) (> (+ %s) 100))", strings.Join(params, " "), strings.Join(params, " ")))
		kernel := scm.CompileJITFilterBuffer(proc.Proc(), tags)
		if kernel == nil {
			t.Fatalf("width %d failed compilation", width)
		}
		ids := make([]uint32, 128)
		values := make([]scm.Scmer, len(ids)*width)
		var want []uint32
		for row := range ids {
			ids[row] = uint32(row + 200)
			for col := 0; col < width; col++ {
				values[row*width+col] = scm.NewInt(int64(row + col))
			}
			if scm.ToBool(scm.Apply(proc, values[row*width:(row+1)*width]...)) {
				want = append(want, ids[row])
			}
		}
		count := kernel(ids, values)
		if count != len(want) {
			t.Fatalf("width %d count %d want %d", width, count, len(want))
		}
		for i := range want {
			if ids[i] != want[i] {
				t.Fatalf("width %d row %d got %d want %d", width, i, ids[i], want[i])
			}
		}
	}
}

func TestTypedFilterMixedMainDelta(t *testing.T) {
	if !scm.JITEnabled() {
		t.Skip("requires JIT")
	}
	shard := benchmarkMapReduceFusionShard(4)
	shard.mu.Lock()
	shard.deltaColumns["amount"] = 0
	shard.inserts = [][]scm.Scmer{{scm.NewFloat(3.5)}, {scm.NewNil()}}
	shard.mu.Unlock()
	column := shard.getColumnStorageOrPanic("amount", false, nil)
	readers := []ColumnReader{newCachedColumnReaderTx(column, nil)}
	predicate := benchmarkMapReduceFusionProc(t, "(lambda (a) (> a 2))")
	kernel := scm.CompileJITFilterBuffer(predicate.Proc(), []uint8{column.JITValueType()})
	if kernel == nil {
		t.Fatal("filter did not compile")
	}
	filter := typedScanFilter{shard: shard, mainCount: 4, width: 1, kernel: kernel,
		multiFuncs: []scm.JITStorageGetValueMultiFunc{compiledColumnGetValueMulti(readers[0])}}
	defer filter.close()
	condition := scm.PrepareSerialProc(predicate)
	ids := []uint32{0, 4, 3, 5, 1, 2}
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	count := filter.filterBatch(ids, []string{"amount"}, []ColumnStorage{column}, readers,
		make([]mapArgGetter, 1), make([]scm.Scmer, 1), &condition)
	want := []uint32{4, 3, 2}
	if count != len(want) {
		t.Fatalf("got %d matches, want %d", count, len(want))
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("got %v, want %v", ids[:count], want)
		}
	}
}

// Exercise concurrent ownership transfers through the actual shared pool.
// Both scan consumers must finish every read/write before returning a buffer.
func TestScanBulkBufferConcurrentRelease(t *testing.T) {
	for _, reducer := range []bool{false, true} {
		t.Run(fmt.Sprintf("reducer=%v", reducer), func(t *testing.T) {
			var workers sync.WaitGroup
			for worker := 0; worker < 8; worker++ {
				workers.Add(1)
				go func(worker int) {
					defer workers.Done()
					for iteration := 0; iteration < 2000; iteration++ {
						pooled, _ := mapReducerBulkBufferPools[0].Get().(*mapReducerBulkBuffer)
						if pooled == nil {
							pooled = &mapReducerBulkBuffer{values: make([]scm.Scmer, defaultScanBufferSize)}
						}
						want := scm.NewInt(int64(worker*2000 + iteration + 1))
						for index := range pooled.values {
							pooled.values[index] = want
						}
						runtime.Gosched()
						for _, value := range pooled.values {
							if !scm.Equal(value, want) {
								t.Errorf("buffer changed while owned by worker %d", worker)
								return
							}
						}
						if reducer {
							mapper := ShardMapReducer{mainBulkBuffer: pooled, mainBulkValues: pooled.values}
							mapper.Close()
							if mapper.mainBulkBuffer != nil || mapper.mainBulkValues != nil {
								t.Error("reducer retained released buffer")
								return
							}
						} else {
							filter := typedScanFilter{width: 1, pooled: pooled, values: pooled.values}
							filter.close()
							if filter.pooled != nil || filter.values != nil {
								t.Error("filter retained released buffer")
								return
							}
						}
					}
				}(worker)
			}
			workers.Wait()
		})
	}
}

// Measure whole-table RecSet feedback independently of SQL plan preparation.
func BenchmarkCompleteRecSetFeedback(b *testing.B) {
	Init(scm.Globalenv)
	tbl, cols := scanColumnCostTable(b, "recset_feedback", 65536)
	filter := scm.Read("feedback-benchmark", "(lambda (x) (not (equal? (mod x 10) 0)))")
	schema, values, _ := compileScanAccess(scm.NewSlice([]scm.Scmer{scm.NewString(cols[0])}), filter)
	for i := range values {
		values[i] = scm.Eval(values[i], &scm.Globalenv)
	}
	condition := scm.Eval(scm.Optimize(filter, &scm.Globalenv, nil), &scm.Globalenv)
	run := func() { tbl.scanRecSet(nil, schema, values, cols[:1], condition) }
	run()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		run()
	}
}
