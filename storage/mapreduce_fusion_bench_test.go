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
	"testing"

	"github.com/launix-de/memcp/scm"
)

const mapReduceFusionBenchmarkRows = 1_000_000

func TestJITMapReduceBufferBestOf(t *testing.T) {
	if !scm.JITEnabled() {
		t.Skip("requires the JIT experiment")
	}
	callback := benchmarkMapReduceFusionProc(t, "(lambda (acc amount) (+ acc amount))")
	minimumRows := scm.CurrentJITCosts().MapReduceBufferBreakEven(scm.JITExpressionCost(callback.Proc().Body), 1)
	if minimumRows <= 1 || minimumRows > mapReduceFusionBenchmarkRows {
		t.Fatalf("implausible calibrated buffer break-even: %d", minimumRows)
	}
	shard := benchmarkMapReduceFusionShard(minimumRows + 1)
	mapper := shard.OpenMapReducer([]string{"amount"}, callback, false, 0, nil, nil)
	defer mapper.Close()
	if mapper.bufferReduceProc == nil || mapper.bufferMinRows <= 1 {
		t.Fatalf("simple typed reducer was not admitted: proc=%p min_rows=%d", mapper.bufferReduceProc, mapper.bufferMinRows)
	}
	if got := mapper.Stream(scm.NewInt(0), []uint32{0}, nil); !scm.Equal(got, scm.NewInt(1)) {
		t.Fatalf("point reduction = %s, want 1", scm.String(got))
	}
	if mapper.bufferReduceFn != nil {
		t.Fatal("point reduction paid for a buffer-loop compilation")
	}

	mapper.bufferMinRows = 8
	ids := []uint32{1, 2, 3, 4, 5, 6, 7, 8}
	got := mapper.Stream(scm.NewInt(0), ids, nil)
	want := int64(0)
	for _, id := range ids {
		want += int64(id%251 + 1)
	}
	if !scm.Equal(got, scm.NewInt(want)) {
		t.Fatalf("fused reduction = %s, want %d", scm.String(got), want)
	}
	if mapper.bufferReduceFn == nil {
		t.Fatalf("profitable reducer did not compile after its break-even: seen=%d min=%d columns=%d kind=%d", mapper.bufferSeenRows, mapper.bufferMinRows, mapper.bufferColumnCount, mapper.mapReduceProgram.Kind)
	}
	count := benchmarkMapReduceFusionProc(t, "(lambda (acc) (+ acc 1))")
	countMapper := shard.OpenMapReducer(nil, count, false, 0, nil, nil)
	defer countMapper.Close()
	if countMapper.bufferReduceProc == nil {
		t.Fatal("zero-column accumulator reducer was not admitted")
	}
	countMapper.bufferMinRows = len(ids)
	if got := countMapper.Stream(scm.NewInt(0), ids, nil); !scm.Equal(got, scm.NewInt(int64(len(ids)))) {
		t.Fatalf("fused count = %s, want %d", scm.String(got), len(ids))
	}

	floatSum := benchmarkMapReduceFusionProc(t, "(lambda (acc value) (+ acc value))")
	floatKernel := scm.CompileJITMapReduceBuffer(floatSum.Proc(), []uint8{scm.NewFloat(0).GetTag()})
	floatValues := []scm.Scmer{scm.NewFloat(1.25), scm.NewFloat(2.5)}
	if floatKernel == nil || !scm.Equal(floatKernel(scm.NewInt(0), floatValues, len(floatValues)), scm.NewFloat(3.75)) {
		t.Fatal("buffer reducer did not preserve a dynamic accumulator type transition")
	}

	sumBinding := scm.Globalenv.FindRead(scm.Symbol("sql_sum_reduce"))
	if sumBinding == nil {
		t.Fatal("sql_sum_reduce is not registered")
	}
	nativeMapper := shard.OpenMapReducer([]string{"amount"}, sumBinding.Vars[scm.Symbol("sql_sum_reduce")], false, 0, nil, nil)
	defer nativeMapper.Close()
	if nativeMapper.bufferReduceProc == nil {
		wrapped := scm.PrepareJITBufferProc(sumBinding.Vars[scm.Symbol("sql_sum_reduce")], 2)
		wrappedCost := -1
		if wrapped != nil {
			wrappedCost = scm.JITExpressionCost(wrapped.Body)
		}
		t.Fatalf("native reducer with a JIT emitter was not admitted: kind=%d wrapped=%p cost=%d min=%d main=%d type=%d", nativeMapper.mapReduceProgram.Kind, wrapped, wrappedCost, scm.CurrentJITCosts().MapReduceBufferBreakEven(wrappedCost, 1), nativeMapper.mainCount, nativeMapper.mainCols[0].JITValueType())
	}
	nativeMapper.bufferMinRows = len(ids)
	if got := nativeMapper.Stream(scm.NewNil(), ids, nil); !scm.Equal(got, scm.NewInt(want)) {
		t.Fatalf("fused native SQL sum = %s, want %d", scm.String(got), want)
	}

	complex := benchmarkMapReduceFusionProc(t, "(lambda (acc amount) (+ acc (* amount 3)))")
	complexMapper := shard.OpenMapReducer([]string{"amount"}, complex, false, 0, nil, nil)
	defer complexMapper.Close()
	if complexMapper.bufferReduceProc != nil {
		t.Fatal("complex reducer entered the slower buffer-loop class")
	}

	wide := benchmarkMapReduceFusionProc(t, "(lambda (acc amount quantity) (+ acc (* amount quantity)))")
	wideMapper := shard.OpenMapReducer([]string{"amount", "quantity"}, wide, false, 0, nil, nil)
	defer wideMapper.Close()
	if wideMapper.bufferReduceProc != nil {
		t.Fatal("short wide reduction should not pay for adaptive compilation")
	}

	largeShard := benchmarkMapReduceFusionShard(mapReduceFusionBenchmarkRows)
	expression := benchmarkMapReduceFusionProc(t, "(lambda (acc amount quantity factor) (+ acc (+ amount (* quantity factor))))")
	expressionMapper := largeShard.OpenMapReducer([]string{"amount", "quantity", "factor"}, expression, false, 0, nil, nil)
	defer expressionMapper.Close()
	wantExpression := expressionMapper.Stream(scm.NewInt(0), ids, nil)
	expressionMapper.prefetchMainColumns(ids)
	valueTypes := make([]uint8, len(expressionMapper.mainCols))
	for index, column := range expressionMapper.mainCols {
		valueTypes[index] = column.JITValueType()
	}
	kernel := scm.CompileJITMapReduceBuffer(expression.Proc(), valueTypes)
	if kernel == nil {
		t.Fatal("arbitrary-arity reducer did not compile")
	}
	if got := kernel(scm.NewInt(0), expressionMapper.mainBulkValues, len(ids)); !scm.Equal(got, wantExpression) {
		t.Fatalf("typed expression reducer = %s, want %s", scm.String(got), scm.String(wantExpression))
	}
	lifecycleMapper := largeShard.OpenMapReducer([]string{"amount", "quantity", "factor"}, expression, false, 0, nil, nil)
	lifecycleMapper.prefetchMainColumns(ids)
	if len(lifecycleMapper.mainBulkValues) == 0 {
		t.Fatal("lifecycle test did not allocate the query-local value buffer")
	}
	lifecycleMapper.Close()
	if lifecycleMapper.mainBulkValues != nil || lifecycleMapper.bufferReduceProc != nil || lifecycleMapper.bufferValueTypes != nil {
		t.Fatal("Close retained query-local map-reduce buffer state")
	}
}

func benchmarkMapReduceFusionShard(rows int) *storageShard {
	amountValues := make([]scm.Scmer, rows)
	quantityValues := make([]scm.Scmer, rows)
	factorValues := make([]scm.Scmer, rows)
	for index := range rows {
		amountValues[index] = scm.NewInt(int64(index%251 + 1))
		quantityValues[index] = scm.NewInt(int64(index%7 + 1))
		factorValues[index] = scm.NewInt(int64(index%5 + 1))
	}
	amount := buildStorageInt(amountValues)
	quantity := buildStorageInt(quantityValues)
	factor := buildStorageInt(factorValues)
	tbl := &table{Columns: []*column{
		{Name: "amount", Typ: "int"},
		{Name: "quantity", Typ: "int"},
		{Name: "factor", Typ: "int"},
	}}
	shard := &storageShard{
		t: tbl,
		columns: map[string]ColumnStorage{
			"amount":   amount,
			"quantity": quantity,
			"factor":   factor,
		},
		deltaColumns: make(map[string]int),
		main_count:   uint32(rows),
	}
	shard.deletions.Reset()
	return shard
}

func benchmarkMapReduceFusionProc(tb testing.TB, source string) scm.Scmer {
	tb.Helper()
	proc := scm.CompileJIT(optimizedScanProc(tb, source), true)
	if scm.JITEnabled() && scm.PrepareSerialProc(proc).Kind != scm.SerialProcJIT {
		tb.Fatalf("benchmark callback did not JIT-compile: %s", source)
	}
	return proc
}

// BenchmarkMapReduceBufferedSpectrum measures the real main-storage consumer:
// typed bulk readers populate the row-major Scmer buffer before the prepared
// map-reducer folds it. Batch sizes cover point probes through full analytical
// scans so a fused-loop threshold can be selected from one stable fixture.
func BenchmarkMapReduceBufferedSpectrum(b *testing.B) {
	if !scm.JITEnabled() {
		b.Skip("requires the JIT experiment")
	}
	shard := benchmarkMapReduceFusionShard(mapReduceFusionBenchmarkRows)
	recids := make([]uint32, mapReduceFusionBenchmarkRows)
	for index := range recids {
		recids[index] = uint32(index)
	}

	shapes := []struct {
		name    string
		cols    []string
		source  string
		neutral scm.Scmer
	}{
		{name: "Count", source: "(lambda (acc) (+ acc 1))", neutral: scm.NewInt(0)},
		{name: "Sum", cols: []string{"amount"}, source: "(lambda (acc amount) (+ acc amount))", neutral: scm.NewInt(0)},
		{name: "AffineSum", cols: []string{"amount"}, source: "(lambda (acc amount) (+ acc (* amount 3)))", neutral: scm.NewInt(0)},
		{name: "TwoColumnSum", cols: []string{"amount", "quantity"}, source: "(lambda (acc amount quantity) (+ acc (* amount quantity)))", neutral: scm.NewInt(0)},
		{name: "ExpressionSum", cols: []string{"amount", "quantity", "factor"}, source: "(lambda (acc amount quantity factor) (+ acc (+ amount (* quantity factor))))", neutral: scm.NewInt(0)},
	}
	batchSizes := []int{1, 8, 64, 1_024, 60_000, mapReduceFusionBenchmarkRows}

	for _, shape := range shapes {
		callback := benchmarkMapReduceFusionProc(b, shape.source)
		mapper := shard.OpenMapReducer(shape.cols, callback, false, 0, nil, nil)
		defer mapper.Close()
		valueTypes := make([]uint8, len(mapper.mainCols))
		for index, column := range mapper.mainCols {
			valueTypes[index] = column.JITValueType()
		}
		proc := callback.Proc()
		fused := scm.CompileJITMapReduceBuffer(proc, valueTypes)
		if fused == nil {
			b.Fatalf("failed to compile fused %s reducer", shape.name)
		}
		for _, batchSize := range batchSizes {
			ids := recids[:batchSize]
			want := mapper.Stream(shape.neutral, ids, nil)
			mapper.prefetchMainColumns(ids)
			if got := fused(shape.neutral, mapper.mainBulkValues, batchSize); !scm.Equal(got, want) {
				b.Fatalf("fused %s/%d result changed: got %s, want %s", shape.name, batchSize, scm.String(got), scm.String(want))
			}
			name := fmt.Sprintf("%s/Rows%d/Current", shape.name, batchSize)
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				b.ReportMetric(float64(batchSize), "rows/op")
				b.ResetTimer()
				var result scm.Scmer
				for sample := 0; sample < b.N; sample++ {
					result = mapper.Stream(shape.neutral, ids, nil)
				}
				if !scm.Equal(result, want) {
					b.Fatalf("result changed: got %s, want %s", scm.String(result), scm.String(want))
				}
				runtime.KeepAlive(result)
			})
			b.Run(fmt.Sprintf("%s/Rows%d/Fused", shape.name, batchSize), func(b *testing.B) {
				b.ReportAllocs()
				b.ReportMetric(float64(batchSize), "rows/op")
				b.ResetTimer()
				var result scm.Scmer
				for sample := 0; sample < b.N; sample++ {
					mapper.prefetchMainColumns(ids)
					result = fused(shape.neutral, mapper.mainBulkValues, batchSize)
				}
				if !scm.Equal(result, want) {
					b.Fatalf("result changed: got %s, want %s", scm.String(result), scm.String(want))
				}
				runtime.KeepAlive(result)
			})
		}
	}
}

func BenchmarkMapReduceBufferCompile(b *testing.B) {
	if !scm.JITEnabled() {
		b.Skip("requires the JIT experiment")
	}
	shapes := []struct {
		name   string
		source string
		cols   int
	}{
		{name: "Sum", source: "(lambda (acc amount) (+ acc amount))", cols: 1},
		{name: "ExpressionSum", source: "(lambda (acc amount quantity factor) (+ acc (+ amount (* quantity factor))))", cols: 3},
	}
	for _, shape := range shapes {
		callback := benchmarkMapReduceFusionProc(b, shape.source)
		valueTypes := make([]uint8, shape.cols)
		for index := range valueTypes {
			valueTypes[index] = scm.NewInt(0).GetTag()
		}
		b.Run(shape.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				compiled := scm.CompileJITMapReduceBuffer(callback.Proc(), valueTypes)
				if compiled == nil {
					b.Fatal("failed to compile buffer reducer")
				}
				runtime.KeepAlive(compiled)
			}
		})
	}
}

// BenchmarkMapReduceBufferedBestOf includes mapper setup, the calibrated lazy
// compilation decision and the 1024-row scan batches used by production.
func BenchmarkMapReduceBufferedBestOf(b *testing.B) {
	if !scm.JITEnabled() {
		b.Skip("requires the JIT experiment")
	}
	shard := benchmarkMapReduceFusionShard(mapReduceFusionBenchmarkRows)
	recids := make([]uint32, mapReduceFusionBenchmarkRows)
	for index := range recids {
		recids[index] = uint32(index)
	}
	shapes := []struct {
		name    string
		cols    []string
		source  string
		neutral scm.Scmer
	}{
		{name: "Count", source: "(lambda (acc) (+ acc 1))", neutral: scm.NewInt(0)},
		{name: "Sum", cols: []string{"amount"}, source: "(lambda (acc amount) (+ acc amount))", neutral: scm.NewInt(0)},
		{name: "AffineSum", cols: []string{"amount"}, source: "(lambda (acc amount) (+ acc (* amount 3)))", neutral: scm.NewInt(0)},
		{name: "TwoColumnSum", cols: []string{"amount", "quantity"}, source: "(lambda (acc amount quantity) (+ acc (* amount quantity)))", neutral: scm.NewInt(0)},
		{name: "ExpressionSum", cols: []string{"amount", "quantity", "factor"}, source: "(lambda (acc amount quantity factor) (+ acc (+ amount (* quantity factor))))", neutral: scm.NewInt(0)},
	}
	for _, shape := range shapes {
		callback := benchmarkMapReduceFusionProc(b, shape.source)
		for _, rowCount := range []int{1, 8, 1_024, 60_000, mapReduceFusionBenchmarkRows} {
			for _, bestOf := range []bool{false, true} {
				mode := "Baseline"
				if bestOf {
					mode = "BestOf"
				}
				b.Run(fmt.Sprintf("%s/Rows%d/%s", shape.name, rowCount, mode), func(b *testing.B) {
					b.ReportAllocs()
					b.ReportMetric(float64(rowCount), "rows/op")
					var result scm.Scmer
					for sample := 0; sample < b.N; sample++ {
						mapper := shard.OpenMapReducer(shape.cols, callback, false, 0, nil, nil)
						if !bestOf {
							mapper.bufferReduceProc = nil
						}
						result = shape.neutral
						for offset := 0; offset < rowCount; offset += defaultScanBufferSize {
							end := min(offset+defaultScanBufferSize, rowCount)
							result = mapper.Stream(result, recids[offset:end], nil)
						}
						mapper.Close()
					}
					runtime.KeepAlive(result)
				})
			}
		}
	}
}
