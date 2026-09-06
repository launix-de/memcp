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
	"strings"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func TestScanOrderOptimizerCompilesOrderIntoAccessSchema(t *testing.T) {
	Init(scm.Globalenv)
	expr := scm.Read(t.Name(), `(lambda (table_value)
		(scan_order nil table_value '() '() '() (lambda () true)
			'("rank") (list <) 0 0 10
			'("rank") (lambda (acc rank) rank) nil false nil '() nil))`)
	optimized := scm.Optimize(expr, &scm.Globalenv, nil)
	plan := scm.SerializeToString(optimized, &scm.Globalenv)
	if !strings.Contains(plan, `(scan_boundary "range" "rank" -1 -1 true true "" false)`) {
		t.Fatalf("optimized scan_order has no static order boundary: %s", plan)
	}
}

func TestScanOrderBatchAcceptOptimizerCompilesOrderIntoAccessSchema(t *testing.T) {
	Init(scm.Globalenv)
	expr := scm.Read(t.Name(), `(lambda (table_value)
		(scan_order_batch_accept nil table_value '() '()
			(lambda (input) input) '("rank") (list <) 0 0 10
			'("rank") (lambda (acc rank) rank) nil false nil))`)
	optimized := scm.Optimize(expr, &scm.Globalenv, nil)
	plan := scm.SerializeToString(optimized, &scm.Globalenv)
	if !strings.Contains(plan, `(scan_boundary "range" "rank" -1 -1 true true "" false)`) {
		t.Fatalf("optimized scan_order_batch_accept has no static order boundary: %s", plan)
	}
}

func TestExtendBoundariesRejectsPartiallyCoveredOrder(t *testing.T) {
	lookupSort := buildProc([]string{"foreign_id", "$tx"}, scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("nested_lookup"),
		scm.NewSymbol("foreign_id"),
		scm.NewSymbol("$tx"),
	}))
	original := analyzedBoundaries{{
		col: "tenant_id", matcher: EqualMatcher,
		lower: scm.NewInt(7), lowerInclusive: true,
		upper: scm.NewInt(7), upperInclusive: true,
	}}

	got, covered := extendBoundariesWithSortCols(original,
		[]scm.Scmer{lookupSort, scm.NewString("id")}, nil)

	if covered {
		t.Fatal("ORDER BY must not enable early LIMIT when its first key is not index-covered")
	}
	if len(got) != len(original) {
		t.Fatalf("partial ORDER BY coverage changed boundaries: got %d, want %d", len(got), len(original))
	}
}

func TestCompileScanOrderAccessStoresOrderInStaticSchema(t *testing.T) {
	directionValue, direction := integerOrder(false)
	schemaExpr, valuesExpr, compiled := compileScanOrderAccess(
		scm.NewSlice(nil), scm.NewSlice(nil),
		scm.NewSlice([]scm.Scmer{scm.NewString("rank")}),
		scm.NewSlice([]scm.Scmer{directionValue}),
	)
	if !compiled {
		t.Fatal("static ORDER BY did not compile into scan access")
	}
	schema, ok := scanStaticListElements(schemaExpr)
	if !ok || len(schema) != scanAccessSchemaHeaderSize+1 {
		t.Fatalf("compiled schema has invalid shape: %s", scm.String(schemaExpr))
	}
	meta, valid := decodeScanAccessHeader(schema[0])
	if !valid || meta.count != 1 {
		t.Fatalf("compiled schema boundary count = %d, valid = %t", meta.count, valid)
	}
	boundary := ScanBoundaryFromScmer(schema[scanAccessSchemaHeaderSize])
	if boundary.ColumnName() != "rank" || boundary.Analyzer() != RangeMatcher || boundary.Order() == nil {
		t.Fatalf("compiled ORDER BY boundary = %#v", boundary)
	}
	if boundary.OrderMetadata() != orderRelationMeta(direction) {
		t.Fatalf("compiled order metadata = %q, want %q", boundary.OrderMetadata(), orderRelationMeta(direction))
	}
	values, ok := scanStaticListElements(valuesExpr)
	if !ok || len(values) != 0 {
		t.Fatalf("direct ORDER BY added runtime values: %s", scm.String(valuesExpr))
	}

	access, ok := scanAccessFromScheme(scm.NewSlice(schema), values, nil)
	if !ok {
		t.Fatal("compiled ORDER BY schema was rejected by runtime")
	}
	access, _ = extendScanAccessWithSortCols(access,
		[]scm.Scmer{scm.NewString("rank")}, []func(...scm.Scmer) scm.Scmer{direction})
	if access.runtime != nil {
		t.Fatal("runtime rebuilt an ORDER BY segment already present in the static schema")
	}
}

func TestCompileScanOrderAccessPreservesPointPrefixAndRuntimeValues(t *testing.T) {
	directionValue, _ := integerOrder(true)
	point := newScanBoundarySpec("tenant", EqualMatcher, 0, 0, true, true,
		"", false, -1, nil, nil, "", false)
	schema := scm.NewSlice([]scm.Scmer{
		newScanAccessHeader(1, scanAccessConsumerCoveredScan, 0, -1), point,
	})
	wantedTenant := scm.NewSymbol("wanted_tenant")
	values := scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), wantedTenant})

	schemaExpr, valuesExpr, compiled := compileScanOrderAccess(
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), schema}), values,
		scm.NewSlice([]scm.Scmer{scm.NewString("rank")}),
		scm.NewSlice([]scm.Scmer{directionValue}),
	)
	if !compiled {
		t.Fatal("point-prefix ORDER BY did not compile into scan access")
	}
	compiledSchema, _ := scanStaticListElements(schemaExpr)
	meta, valid := decodeScanAccessHeader(compiledSchema[0])
	if !valid || meta.count != 2 || meta.consumer != scanAccessConsumerCoveredScan {
		t.Fatalf("compiled point-prefix metadata = %#v, valid = %t", meta, valid)
	}
	if got := ScanBoundaryFromScmer(compiledSchema[1]).ColumnName(); got != "tenant" {
		t.Fatalf("first boundary = %q, want tenant", got)
	}
	if got := ScanBoundaryFromScmer(compiledSchema[2]).ColumnName(); got != "rank" {
		t.Fatalf("second boundary = %q, want rank", got)
	}
	compiledValues, ok := scanStaticListElements(valuesExpr)
	if !ok || len(compiledValues) != 1 || !compiledValues[0].SymbolEquals("wanted_tenant") {
		t.Fatalf("runtime values changed: %s", scm.String(valuesExpr))
	}
}

func TestCompileScanOrderAccessPreservesOptimizedLocalValuesExpression(t *testing.T) {
	directionValue, _ := integerOrder(true)
	point := newScanBoundarySpec("tenant", EqualMatcher, 0, 0, true, true,
		"", false, -1, nil, nil, "", false)
	schema := scm.NewSlice([]scm.Scmer{
		newScanAccessHeader(1, scanAccessConsumerCoveredScan, 0, -1), point,
	})
	values := scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewNthLocalVar(1)})

	_, valuesExpr, compiled := compileScanOrderAccess(
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), schema}), values,
		scm.NewSlice([]scm.Scmer{scm.NewString("rank")}),
		scm.NewSlice([]scm.Scmer{directionValue}),
	)
	if !compiled {
		t.Fatal("point-prefix ORDER BY did not compile into scan access")
	}
	if valuesExpr != values {
		t.Fatalf("compiled ORDER BY rewrote optimized local values: got %s, want %s",
			scm.String(valuesExpr), scm.String(values))
	}
}

func TestCompileScanOrderAccessStoresComputedOrderMapper(t *testing.T) {
	Init(scm.Globalenv)
	directionValue, _ := integerOrder(false)
	computed := scm.Eval(scm.Optimize(scm.Read(t.Name(), `(lambda (rank) (+ rank 1))`), &scm.Globalenv, nil), &scm.Globalenv)
	proc := computed.Proc()
	if !isRawDataset(proc.Params.Slice(), proc.Body) {
		t.Fatal("computed ORDER BY fixture is not indexable")
	}
	schemaExpr, valuesExpr, compiled := compileScanOrderAccess(
		scm.NewSlice(nil), scm.NewSlice(nil),
		scm.NewSlice([]scm.Scmer{computed}),
		scm.NewSlice([]scm.Scmer{directionValue}),
	)
	if !compiled {
		t.Fatal("computed ORDER BY did not compile into scan access")
	}
	schema, _ := scanStaticListElements(schemaExpr)
	boundary := ScanBoundaryFromScmer(schema[scanAccessSchemaHeaderSize])
	if boundary.MapperSlot() != 0 || len(boundary.MapColumns()) != 1 || boundary.MapColumns()[0] != "rank" {
		t.Fatalf("computed order boundary has invalid mapper metadata: slot=%d columns=%v",
			boundary.MapperSlot(), boundary.MapColumns())
	}
	values, ok := scanStaticListElements(valuesExpr)
	if !ok || len(values) != 1 {
		t.Fatalf("computed ORDER BY values = %s", scm.String(valuesExpr))
	}
	descriptor, ok := scanStaticListElements(values[0])
	if !ok || len(descriptor) != 3 || !descriptor[0].SymbolEquals("compile_scan_computed_index") {
		t.Fatalf("computed ORDER BY mapper descriptor = %s", scm.String(values[0]))
	}
}

func TestCompileScanOrderAccessListCompilesEveryOrderedInput(t *testing.T) {
	directionValue, _ := integerOrder(false)
	emptySchema := scm.NewSlice(nil)
	schemasExpr, valuesExpr, compiled := compileScanOrderAccessList(
		scm.NewSlice([]scm.Scmer{emptySchema, emptySchema}), scm.NewSlice(nil),
		scm.NewSlice([]scm.Scmer{
			scm.NewSlice([]scm.Scmer{scm.NewString("rank")}),
			scm.NewSlice([]scm.Scmer{scm.NewString("rank")}),
		}),
		scm.NewSlice([]scm.Scmer{directionValue}),
	)
	if !compiled {
		t.Fatal("multi-source ORDER BY did not compile into scan access")
	}
	schemas, ok := scanStaticListElements(schemasExpr)
	if !ok || len(schemas) != 2 {
		t.Fatalf("compiled multi-source schemas have invalid shape: %s", scm.String(schemasExpr))
	}
	for i, schemaValue := range schemas {
		schema, static := scanStaticListElements(schemaValue)
		if !static || len(schema) != scanAccessSchemaHeaderSize+1 {
			t.Fatalf("compiled schema %d has invalid shape: %s", i, scm.String(schemaValue))
		}
		if got := ScanBoundaryFromScmer(schema[scanAccessSchemaHeaderSize]).ColumnName(); got != "rank" {
			t.Fatalf("compiled schema %d order column = %q, want rank", i, got)
		}
	}
	values, ok := scanStaticListElements(valuesExpr)
	if !ok || len(values) != 0 {
		t.Fatalf("direct multi-source ORDER BY added runtime values: %s", scm.String(valuesExpr))
	}
}

func TestCompileScanJoinDriverOrderAccessOnlyCompilesDriver(t *testing.T) {
	directionValue, _ := integerOrder(true)
	emptySchema := scm.NewSlice(nil)
	schemasExpr, _, compiled := compileScanJoinDriverOrderAccess(
		scm.NewSlice([]scm.Scmer{emptySchema, emptySchema}), scm.NewSlice(nil),
		scm.NewSlice([]scm.Scmer{
			scm.NewSlice([]scm.Scmer{scm.NewInt(0), scm.NewString("rank")}),
		}),
		scm.NewSlice([]scm.Scmer{directionValue}),
	)
	if !compiled {
		t.Fatal("join driver ORDER BY did not compile into scan access")
	}
	schemas, _ := scanStaticListElements(schemasExpr)
	driver, _ := scanStaticListElements(schemas[0])
	if len(driver) != scanAccessSchemaHeaderSize+1 || ScanBoundaryFromScmer(driver[1]).ColumnName() != "rank" {
		t.Fatalf("compiled join driver schema = %s", scm.String(schemas[0]))
	}
	inner, _ := scanStaticListElements(schemas[1])
	if len(inner) != 0 {
		t.Fatalf("join inner schema unexpectedly received final order: %s", scm.String(schemas[1]))
	}
}

func TestCoveredOrderedLimitBrakesInsideIndexBatch(t *testing.T) {
	const rows = 4096
	const limit = 20
	tbl := setupScanParallelTestTable(t, "tcoveredorderedlimit")
	tbl.CreateColumn("bucket", "VARCHAR(16)", nil, nil)
	tbl.CreateColumn("rank", "INT", nil, nil)
	values := make([][]scm.Scmer, rows)
	for i := range values {
		values[i] = []scm.Scmer{
			scm.NewInt(int64(i)),
			scm.NewString("target"),
			scm.NewInt(int64(i)),
		}
	}
	tbl.Insert([]string{"id", "bucket", "rank"}, values, nil, scm.NewNil(), false, nil)
	RebuildTable(tbl, true, false)

	equalSymbol := scm.Symbol("equal?")
	equalFn := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewBool(scm.Equal(values[0], values[1]))
	})
	condition := buildProc([]string{"bucket"}, scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal?"),
		scm.NewSymbol("bucket"),
		scm.NewString("target"),
	}))
	condition.Proc().En.Vars[equalSymbol] = equalFn
	_, order := integerOrder(false)
	bounds := extractBoundaries([]string{"bucket"}, condition)
	var covered bool
	bounds, covered = extendBoundariesWithSortCols(bounds,
		[]scm.Scmer{scm.NewString("rank")}, []func(...scm.Scmer) scm.Scmer{order})
	if !covered {
		t.Fatal("bucket/rank index does not cover ORDER BY")
	}
	shard := tbl.ActiveShards()[0]
	shard.mu.RLock()
	var buildBuf [8]uint32
	shard.iterateIndexForce(nil, runtimeScanAccess(bounds), len(shard.inserts), buildBuf[:], false,
		func([]uint32) bool { return false })
	shard.mu.RUnlock()
	run := func(acceptCols []string, accept scm.Scmer) *shardqueue {
		return shard.scan_order(coveredRuntimeScanAccess(bounds),
			[]string{"bucket"}, condition, acceptCols, accept,
			[]scm.Scmer{scm.NewString("rank")}, []func(...scm.Scmer) scm.Scmer{order},
			0, 0, limit, nil, nil, nil)
	}
	queue := run(nil, scm.NewNil())
	if queue.candidateCount != limit {
		t.Fatalf("covered ordered LIMIT examined %d candidates, want %d",
			queue.candidateCount, limit)
	}

	// An additional acceptance predicate can still reject index matches, so it
	// must retain the ordinary scan batch instead of claiming full coverage.
	acceptAll := buildProc([]string{"bucket"}, scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal?"),
		scm.NewSymbol("bucket"),
		scm.NewString("target"),
	}))
	acceptAll.Proc().En.Vars[equalSymbol] = equalFn
	queue = run([]string{"bucket"}, acceptAll)
	if queue.candidateCount <= limit {
		t.Fatalf("acceptance predicate incorrectly used covered LIMIT batch: %d candidates",
			queue.candidateCount)
	}

	// Visibility is checked after index iteration. A short covered batch may be
	// refilled, but must still return LIMIT live rows after deleted entries.
	shard.mu.Lock()
	for recid := uint(0); recid < 5; recid++ {
		shard.deletions.Set(recid, true)
	}
	shard.mu.Unlock()
	queue = run(nil, scm.NewNil())
	if len(queue.items) != limit {
		t.Fatalf("covered ordered LIMIT returned %d live rows after deletions, want %d",
			len(queue.items), limit)
	}
	if queue.candidateCount <= limit {
		t.Fatalf("covered ordered LIMIT did not refill deleted candidates: examined %d",
			queue.candidateCount)
	}
}
