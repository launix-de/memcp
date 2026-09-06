/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

var benchmarkBoundaries boundaries

func testEqualScanAccess(column string, value scm.Scmer) (scm.Scmer, []scm.Scmer) {
	return scm.NewSlice([]scm.Scmer{
		newScanAccessHeader(1, scanAccessConsumerScan, 0, -1),
		scm.NewString("equal"), scm.NewString(column), newScanAccessBoundaryMeta(0, 0, 3), scm.NewString(""),
	}), []scm.Scmer{value}
}

func testUpperScanAccess(column string, value scm.Scmer, inclusive bool) (scm.Scmer, []scm.Scmer) {
	flags := int64(0)
	if inclusive {
		flags = 2
	}
	return scm.NewSlice([]scm.Scmer{
		newScanAccessHeader(1, scanAccessConsumerScan, 0, -1),
		scm.NewString("range"), scm.NewString(column), newScanAccessBoundaryMeta(-1, 0, flags), scm.NewString(""),
	}), []scm.Scmer{value}
}

func TestCompileScanAccessReadsRuntimeValuesWithoutAllocation(t *testing.T) {
	columns := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("list"), scm.NewString("tenant"), scm.NewString("created_at"),
	})
	filter := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("tenant"), scm.NewSymbol("created")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("and"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewSymbol("tenant"), scm.NewSymbol("wanted_tenant")}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol(">="), scm.NewSymbol("created"), scm.NewSymbol("minimum_created")}),
		}),
	})
	schema, bindingExprs, ok := compileScanAccess(columns, filter)
	if !ok || len(bindingExprs) != 2 {
		t.Fatalf("compileScanAccess returned ok=%v bindings=%d", ok, len(bindingExprs))
	}
	values := []scm.Scmer{scm.NewInt(7), scm.NewInt(100)}
	access, valid := scanAccessFromScheme(schema, values, nil)
	if !valid || access.len() != 2 {
		t.Fatalf("scanAccessFromScheme returned valid=%v boundaries=%d", valid, access.len())
	}
	bound := boundaries{access.boundary(0), access.boundary(1)}
	if bound[0].col != "tenant" || bound[0].matcher != EqualMatcher || bound[0].lower.Int() != 7 {
		t.Fatalf("unexpected compiled equality: %#v", bound[0])
	}
	if bound[1].col != "created_at" || bound[1].matcher != RangeMatcher || bound[1].lower.Int() != 100 || !bound[1].lowerInclusive {
		t.Fatalf("unexpected compiled range: %#v", bound[1])
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		_, _ = scanAccessFromScheme(schema, values, nil)
	}); allocs != 0 {
		t.Fatalf("compiled scan access view allocated %.2f times per run, want 0", allocs)
	}
}

func TestScanAccessValuesExprCachesConstantsAndBindsDynamicValues(t *testing.T) {
	static := scanAccessValuesExpr([]scm.Scmer{scm.NewString("ready"), scm.NewInt(7)})
	staticItems, ok := scmerSlice(static)
	if !ok || len(staticItems) != 2 || !scanSymbolIs(staticItems[0], "quote") {
		t.Fatalf("constant access values = %s, want quoted cached vector", scm.String(static))
	}
	dynamic := scanAccessValuesExpr([]scm.Scmer{scm.NewSymbol("outer_id")})
	dynamicItems, ok := scmerSlice(dynamic)
	if !ok || len(dynamicItems) != 2 || !scanSymbolIs(dynamicItems[0], "list") {
		t.Fatalf("dynamic access values = %s, want runtime list", scm.String(dynamic))
	}
}

func TestCompileScanAccessCarriesComputedFormulaRuntimeConstants(t *testing.T) {
	columns := scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString("doc")})
	path := scm.NewSlice([]scm.Scmer{scm.NewSymbol("session"), scm.NewString("v1")})
	wanted := scm.NewSlice([]scm.Scmer{scm.NewSymbol("session"), scm.NewString("v2")})
	filter := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("doc")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("equal??"),
			scm.NewSlice([]scm.Scmer{
				scm.Globalenv.Vars[scm.Symbol("json_value")], scm.NewSymbol("doc"), path, scm.NewString("UNSIGNED"),
			}),
			wanted,
		}),
	})
	schema, bindings, ok := compileScanAccess(columns, filter)
	if !ok || len(bindings) != 2 {
		t.Fatalf("computed access compilation returned ok=%v bindings=%d", ok, len(bindings))
	}
	items := schema.Slice()
	meta, _ := decodeScanAccessHeader(items[0])
	if meta.projections != 1 || items[len(items)-1].String() != "doc" {
		t.Fatalf("computed access schema omitted mapper columns: %s", scm.String(schema))
	}
	boundaryMeta := decodeScanAccessBoundaryMeta(items[scanAccessSchemaHeaderSize+2])
	if mapperSlot := int(boundaryMeta.flags>>3) - 1; mapperSlot != 1 {
		t.Fatalf("computed mapper slot = %d, want 1", mapperSlot)
	}

	mapper := scm.Eval(scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("doc")}),
		scm.NewSlice([]scm.Scmer{
			scm.Globalenv.Vars[scm.Symbol("json_value")], scm.NewSymbol("doc"), scm.NewString("$.tenant"), scm.NewString("UNSIGNED"),
		}),
	}), &scm.Globalenv)
	descriptor := compileComputedScanIndex(mapper, []string{"doc"})
	access, valid := scanAccessFromScheme(schema, []scm.Scmer{scm.NewInt(17), descriptor}, nil)
	bound := access.boundary(0)
	if !valid || bound.col != ".(json_value doc \"$.tenant\" \"UNSIGNED\")" ||
		len(bound.mapCols) != 1 || bound.mapCols[0] != "doc" || !bound.mapFn.IsProc() {
		t.Fatalf("unexpected computed access boundary: valid=%v boundary=%#v", valid, bound)
	}
}

func TestCompileScanAccessRejectsDisjunction(t *testing.T) {
	columns := scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString("id")})
	filter := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("id")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("or"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewSymbol("id"), scm.NewInt(1)}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewSymbol("id"), scm.NewInt(2)}),
		}),
	})
	if _, _, ok := compileScanAccess(columns, filter); ok {
		t.Fatal("compileScanAccess accepted a disjunction")
	}
}

func TestCompileScanAccessEncodesBatchSlots(t *testing.T) {
	columns := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("list"), scm.NewString("id"), scm.NewString("#0"),
	})
	filter := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("id"), scm.NewSymbol("batch_id")}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewSymbol("id"), scm.NewSymbol("batch_id")}),
	})
	schema, bindings, ok := compileScanAccessMode(columns, filter, true)
	if !ok || len(bindings) != 0 {
		t.Fatalf("batch access compilation returned ok=%v bindings=%d", ok, len(bindings))
	}
	access, valid := scanAccessFromScheme(schema, nil, nil)
	bound := access.boundary(0)
	if !valid || access.len() != 1 || !bound.lowerBatch || !bound.upperBatch ||
		bound.lowerBatchSubidx != 0 || bound.upperBatchSubidx != 0 {
		t.Fatalf("unexpected compiled batch boundary: valid=%v boundary=%#v", valid, bound)
	}
	scratch := acquireScanAnalyzeScratch()
	defer releaseScanAnalyzeScratch(scratch)
	access = access.useScratch(scratch)
	batchData := []scm.Scmer{scm.NewInt(7), scm.NewInt(11)}
	boundAccess := access.withBatch(1, batchData, 1)
	indexBounds := newScanIndexBounds(boundAccess)
	if indexBounds.lower(0).Int() != 11 || indexBounds.upperLast().Int() != 11 {
		t.Fatalf("batch index view = [%v,%v], want [11,11]", indexBounds.lower(0), indexBounds.upperLast())
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		boundAccess = access.withBatch(1, batchData, 1)
		indexBounds = newScanIndexBounds(boundAccess)
	}); allocs != 0 {
		t.Fatalf("batch index binding allocated %.2f objects, want zero", allocs)
	}
}

func TestCompileScanAccessKeepsCandidateHooksAfterSortedPrefix(t *testing.T) {
	columns := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("list"), scm.NewString("text"), scm.NewString("tenant"), scm.NewString("$recset_contains"),
	})
	filter := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("text"), scm.NewSymbol("tenant"), scm.NewSymbol("in_set")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("and"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("strlike"), scm.NewSymbol("text"), scm.NewSymbol("pattern"), scm.NewString("utf8mb4_unicode_ci")}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewSymbol("tenant"), scm.NewSymbol("wanted_tenant")}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("in_set"), scm.NewSymbol("allowed_rows")}),
		}),
	})
	schema, bindings, ok := compileScanAccess(columns, filter)
	if !ok || len(bindings) != 3 {
		t.Fatalf("hook access compilation returned ok=%v bindings=%d", ok, len(bindings))
	}
	values := []scm.Scmer{scm.NewInt(7), scm.NewNil(), scm.NewString("prefix%")}
	access, valid := scanAccessFromScheme(schema, values, nil)
	if !valid || access.len() != 3 {
		t.Fatalf("compiled hooks returned valid=%v boundaries=%d", valid, access.len())
	}
	bound := boundaries{access.boundary(0), access.boundary(1), access.boundary(2)}
	if bound[0].matcher != EqualMatcher || bound[0].col != "tenant" {
		t.Fatalf("equality is not the leading physical boundary: %#v", bound)
	}
	if bound[1].matcher != RecSetMatcher {
		t.Fatalf("compiled access omitted the RecSet hook: %#v", bound)
	}
	like := bound[2]
	if like.matcher != LikeMatcher || like.collation != "utf8mb4_unicode_ci" || like.lower.String() != "prefix%" {
		t.Fatalf("unexpected LIKE hook: %#v", like)
	}
}

func TestPruneScanResidualDropsExactProbesAndKeepsLike(t *testing.T) {
	columns := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("list"), scm.NewString("id"), scm.NewString("age"), scm.NewString("name"),
	})
	filter := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("id"), scm.NewSymbol("age"), scm.NewSymbol("name")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("and"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal?"), scm.NewSymbol("id"), scm.NewSymbol("wanted_id")}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol(">="), scm.NewSymbol("age"), scm.NewInt(18)}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("strlike"), scm.NewSymbol("name"), scm.NewString("A%")}),
		}),
	})

	residualColumns, residualFilter := pruneScanResidual(columns, filter, false)
	columnItems, ok := scanStaticColumns(residualColumns)
	if !ok || len(columnItems) != 1 || columnItems[0].String() != "name" {
		t.Fatalf("residual columns = %s, want only name", scm.SerializeToString(residualColumns, &scm.Globalenv))
	}
	params, body, ok := scanLambdaParts(residualFilter)
	if !ok || len(params) != 1 || !params[0].SymbolEquals("name") {
		t.Fatalf("residual filter params = %s, want only name", scm.SerializeToString(residualFilter, &scm.Globalenv))
	}
	bodyItems, ok := scmerSlice(body)
	if !ok || len(bodyItems) != 3 || !scanSymbolIs(bodyItems[0], "strlike") {
		t.Fatalf("residual filter body = %s, want LIKE", scm.SerializeToString(body, &scm.Globalenv))
	}
}

func TestPruneScanResidualKeepsSecondRangeColumn(t *testing.T) {
	columns := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("list"), scm.NewString("a"), scm.NewString("b"),
	})
	filter := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("a"), scm.NewSymbol("b")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("and"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("<="), scm.NewSymbol("a"), scm.NewInt(2)}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("<="), scm.NewSymbol("b"), scm.NewInt(1)}),
		}),
	})

	residualColumns, residualFilter := pruneScanResidual(columns, filter, false)
	columnItems, ok := scanStaticColumns(residualColumns)
	if !ok || len(columnItems) != 1 || columnItems[0].String() != "b" {
		t.Fatalf("residual columns = %s, want only b", scm.SerializeToString(residualColumns, &scm.Globalenv))
	}
	_, body, ok := scanLambdaParts(residualFilter)
	bodyItems, sliced := scmerSlice(body)
	if !ok || !sliced || len(bodyItems) != 3 || !scanSymbolIs(bodyItems[0], "<=") {
		t.Fatalf("second range residual = %s, want b <= 1", scm.SerializeToString(residualFilter, &scm.Globalenv))
	}
}

func TestCompileScanAccessKeepsPointProbeWithDuplicatePredicate(t *testing.T) {
	columns := scm.Read(t.Name(), `(list "id")`)
	filter := scm.Read(t.Name(), `(lambda (id) (and (equal?? id wanted) (equal?? id wanted)))`)
	schema, bindings, compiled := compileScanAccess(columns, filter)
	if !compiled || len(bindings) != 1 {
		t.Fatalf("duplicate equality compiled=%v bindings=%d", compiled, len(bindings))
	}
	access, valid := scanAccessFromScheme(schema, []scm.Scmer{scm.NewInt(7)}, nil)
	if !valid || access.len() != 1 || access.boundary(0).col != "id" {
		t.Fatalf("duplicate equality access = %#v", access)
	}
}

func TestPruneScanResidualKeepsConflictingDuplicatePredicate(t *testing.T) {
	columns := scm.Read(t.Name(), `(list "id")`)
	filter := scm.Read(t.Name(), `(lambda (id) (and (equal?? id 1) (equal?? id 2)))`)
	residualColumns, residualFilter := pruneScanResidual(columns, filter, false)
	columnItems, ok := scanStaticColumns(residualColumns)
	if !ok || len(columnItems) != 1 || columnItems[0].String() != "id" {
		t.Fatalf("conflicting residual columns = %s", scm.SerializeToString(residualColumns, &scm.Globalenv))
	}
	_, body, ok := scanLambdaParts(residualFilter)
	items, sliced := scmerSlice(body)
	if !ok || !sliced || len(items) != 3 || !scanSymbolIs(items[0], "equal??") || scm.ToInt(items[2]) != 2 {
		t.Fatalf("conflicting residual = %s, want id = 2", scm.SerializeToString(residualFilter, &scm.Globalenv))
	}
}

func TestScanAccessNullProbeIsImpossibleWithoutTreatingNilCheckAsImpossible(t *testing.T) {
	exactSchema, _ := testEqualScanAccess("id", scm.NewNil())
	exact, valid := scanAccessFromScheme(exactSchema, []scm.Scmer{scm.NewNil()}, nil)
	if !valid || !exact.impossible() {
		t.Fatal("runtime NULL equality probe must be impossible")
	}
	nilCheckSchema := scm.NewSlice([]scm.Scmer{
		newScanAccessHeader(1, scanAccessConsumerScan, 0, -1),
		scm.NewString("equal"), scm.NewString("id"), newScanAccessBoundaryMeta(-1, -1, 3), scm.NewString(""),
	})
	nilCheck, valid := scanAccessFromScheme(nilCheckSchema, nil, nil)
	if !valid || nilCheck.impossible() {
		t.Fatal("IS NULL access must remain executable")
	}
	if boundary := nilCheck.boundary(0); !boundary.lower.IsNil() || !boundary.upper.IsNil() {
		t.Fatal("IS NULL access must bind explicit NULL endpoints")
	}
	nullSafeCall := scm.Read(t.Name(), `(scan nil table_value
		'(369435906932736) '()
		(list "id") (lambda (id) (equal?? id wanted_id))
		(list "id") (lambda (acc id) id) nil nil false)`)
	nullSafeItems, ok := scmerSlice(nullSafeCall)
	if !ok {
		t.Fatal("failed to parse null-safe scan")
	}
	nullSafeSchema, _, compiled := compileScanAccess(nullSafeItems[5], nullSafeItems[6])
	nullSafe, valid := scanAccessFromScheme(nullSafeSchema, []scm.Scmer{scm.NewNil()}, nil)
	if !compiled || !valid || nullSafe.impossible() {
		t.Fatal("null-safe equality must retain its residual NULL match")
	}
	_, residual := pruneScanResidual(nullSafeItems[5], nullSafeItems[6], false)
	if _, body, lambda := scanLambdaParts(residual); !lambda || scanExprIsTrue(body) {
		t.Fatal("null-safe equality must not be pruned from the residual filter")
	}
	constantColumns := scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice([]scm.Scmer{scm.NewString("id")})})
	constantFilter := scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("id")}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewSymbol("id"), scm.NewInt(42)})})
	prunedColumns, prunedFilter := pruneScanResidual(constantColumns, constantFilter, false)
	if columns, ok := scanStaticColumns(prunedColumns); !ok || len(columns) != 0 {
		t.Fatalf("constant non-NULL equality retained columns: %#v", prunedColumns)
	}
	if _, body, lambda := scanLambdaParts(prunedFilter); !lambda || !scanExprIsTrue(body) {
		t.Fatal("constant non-NULL equality must be pruned from the residual filter")
	}
}

func TestCompileScanAccessAcceptsParsedSourceInfo(t *testing.T) {
	call := scm.Read(t.Name(), `(scan nil table_value
		'(369435906932736) '()
		(list "id") (lambda (id) (equal?? id wanted_id))
		(list "id") (lambda (acc id) id) nil nil false)`)
	items, ok := scmerSlice(call)
	if !ok || len(items) < 5 {
		t.Fatalf("unexpected parsed scan expression: %s", scm.SerializeToString(call, &scm.Globalenv))
	}
	if _, bindings, compiled := compileScanAccess(items[5], items[6]); !compiled || len(bindings) != 1 {
		t.Fatalf("parsed scan access compiled=%v bindings=%d", compiled, len(bindings))
	}
}

func TestCompileScanAccessPreservesSQLEqualityCollation(t *testing.T) {
	columns := scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString("name")})
	filter := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("name")}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewSymbol("name"), scm.NewString("alpha")}),
	})
	schema, values, compiled := compileScanAccess(columns, filter)
	access, valid := scanAccessFromScheme(schema, values, nil)
	if !compiled || !valid || access.len() != 1 {
		t.Fatalf("SQL equality access compiled=%v valid=%v boundaries=%d", compiled, valid, access.len())
	}
	if got := access.boundary(0).collation; got != "utf8mb4_general_ci" {
		t.Fatalf("SQL equality access collation = %q, want utf8mb4_general_ci", got)
	}
}

func TestSortedBoundariesCoverCondition(t *testing.T) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal??"),
		scm.NewSymbol("value"),
		scm.NewString("needle"),
	})
	condition := buildProc([]string{"value"}, body)
	bounds := extractBoundaries([]string{"meta_key"}, condition)
	if !sortedBoundariesCoverCondition([]string{"meta_key"}, condition, runtimeScanAccess(bounds)) {
		t.Fatal("simple equality should be covered by its extracted boundary")
	}
	bounds[0].upper = scm.NewString("other")
	if sortedBoundariesCoverCondition([]string{"meta_key"}, condition, runtimeScanAccess(bounds)) {
		t.Fatal("different boundary must not cover the condition")
	}
}

func TestSortedBoundariesCoverRuntimeNullSafeEquality(t *testing.T) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal??"),
		scm.NewSymbol("value"),
		scm.NewString("needle"),
	})
	condition := buildProc([]string{"value"}, body)
	bounds := extractBoundaries([]string{"meta_key"}, condition)
	if len(bounds) != 1 || !bounds[0].nullSafe || bounds[0].collation != "utf8mb4_general_ci" {
		t.Fatalf("NULL-aware equality boundary = %#v", bounds)
	}
	if !sortedBoundariesCoverCondition([]string{"meta_key"}, condition, runtimeScanAccess(bounds)) {
		t.Fatal("non-NULL equal?? should be covered by its identical runtime boundary")
	}

	nullBody := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal??"),
		scm.NewSymbol("value"),
		scm.NewNil(),
	})
	nullCondition := buildProc([]string{"value"}, nullBody)
	nullBounds := extractBoundaries([]string{"meta_key"}, nullCondition)
	if sortedBoundariesCoverCondition([]string{"meta_key"}, nullCondition, runtimeScanAccess(nullBounds)) {
		t.Fatal("runtime NULL equal?? must retain its residual predicate")
	}
}

func TestSortedBoundariesDoNotCoverAfterRangeSuffix(t *testing.T) {
	condition := buildProc([]string{"row_number"}, scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("<="), scm.NewSymbol("row_number"), scm.NewInt(1),
	}))
	access := runtimeScanAccess(boundaries{
		{col: "ID", matcher: RangeMatcher, upper: scm.NewInt(2), upperInclusive: true},
		{col: "row_number", matcher: RangeMatcher, upper: scm.NewInt(1), upperInclusive: true},
	})
	if sortedBoundariesCoverCondition([]string{"row_number"}, condition, access) {
		t.Fatal("a boundary after the first range dimension is not enforced by the index prefix")
	}
}

func TestCoveredScanAccessSuppressesReadResiduals(t *testing.T) {
	plannerAccess := scanAccess{plannerFilterCovered: true}
	if !scanAccessCoversResidual(plannerAccess) {
		t.Fatal("planner coverage should suppress read residuals, including autocommit")
	}
	if !scanAccessCoversResidual(coveredRuntimeScanAccess(nil)) {
		t.Fatal("mandatory internal boundaries should remain unconditionally covered")
	}
}

func TestCompileScanAccessListKeepsEmptySchemas(t *testing.T) {
	columns := scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("list")})})
	filters := scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), buildProc(nil, scm.NewBool(true))})
	schemas, values, compiled, compiledAny := compileScanAccessList(columns, filters, false)
	if compiledAny || len(schemas) != 1 || len(schemas[0].Slice()) != 0 || len(values) != 0 || len(compiled) != 1 || compiled[0] {
		t.Fatalf("empty access list compiled as schemas=%v values=%v compiled=%v compiledAny=%v", schemas, values, compiled, compiledAny)
	}
}

func TestBoundaryConstantThroughCapturedCallable(t *testing.T) {
	dictionary := scm.NewFastDictValue(1)
	dictionary.Set(scm.NewString("id"), scm.NewInt(424), nil)
	lookup := scm.NewFastDict(dictionary)
	outerEnv := &scm.Env{VarsNumbered: []scm.Scmer{lookup}, Outer: &scm.Globalenv}
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal??"),
		scm.NewNthLocalVar(0),
		scm.NewSlice([]scm.Scmer{
			scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("outer"),
				scm.NewInt(1),
				scm.NewNthLocalVar(0),
			}),
			scm.NewString("id"),
		}),
	})
	condition := scm.NewProcStruct(scm.Proc{
		Params:       scm.NewSlice([]scm.Scmer{scm.NewSymbol("id")}),
		Body:         body,
		En:           outerEnv,
		NumVars:      1,
		NumberedOnly: true,
	})

	bounds := extractBoundaries([]string{"ID"}, condition)
	if len(bounds) != 1 || bounds[0].col != "ID" || bounds[0].matcher != EqualMatcher || bounds[0].lower.Int() != 424 {
		t.Fatalf("captured callable boundary = %#v, want ID = 424", bounds)
	}
}

func TestBoundaryConstantThroughJITInlineCapture(t *testing.T) {
	dictionary := scm.NewFastDictValue(1)
	dictionary.Set(scm.NewString("id"), scm.NewInt(424), nil)
	factory := scm.EvalAllJIT(t.Name(), `(lambda (lookup)
		(lambda (id) (equal?? id (lookup "id"))))`, &scm.Globalenv)
	condition := scm.Apply(factory, scm.NewFastDict(dictionary))
	if condition.Proc().Compiled == nil {
		t.Skip("requires GOEXPERIMENT=jit")
	}

	base, captures := condition.Proc().JITCapturedLocals()
	if base != 1 || len(captures) != 1 {
		t.Fatalf("JIT captures = base %d, count %d; want base 1, count 1", base, len(captures))
	}
	bounds := extractBoundaries([]string{"ID"}, condition)
	if len(bounds) != 1 || bounds[0].col != "ID" || bounds[0].matcher != EqualMatcher || bounds[0].lower.Int() != 424 {
		t.Fatalf("JIT captured callable boundary = %#v, want ID = 424", bounds)
	}
}

func TestComputedBoundaryThroughCapturedCallable(t *testing.T) {
	dictionary := scm.NewFastDictValue(2)
	dictionary.Set(scm.NewString("offset"), scm.NewInt(1), nil)
	dictionary.Set(scm.NewString("value"), scm.NewInt(18), nil)
	lookup := scm.NewFastDict(dictionary)
	outerEnv := &scm.Env{VarsNumbered: []scm.Scmer{lookup}, Outer: &scm.Globalenv}
	captured := func(key string) scm.Scmer {
		return scm.NewSlice([]scm.Scmer{
			scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("outer"),
				scm.NewInt(1),
				scm.NewNthLocalVar(0),
			}),
			scm.NewString(key),
		})
	}
	computed := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("+"),
		scm.NewNthLocalVar(0),
		captured("offset"),
	})
	condition := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("number")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("equal??"),
			computed,
			captured("value"),
		}),
		En:           outerEnv,
		NumVars:      1,
		NumberedOnly: true,
	})

	bounds := extractBoundaries([]string{"number"}, condition)
	if len(bounds) != 1 || bounds[0].matcher != EqualMatcher || bounds[0].lower.Int() != 18 {
		t.Fatalf("captured computed boundary = %#v, want computed value = 18", bounds)
	}
	if bounds[0].col != ".(+ number 1)" {
		t.Fatalf("computed column = %q, want stable materialized expression", bounds[0].col)
	}
	if bounds[0].mapFn.IsNil() || scm.Apply(bounds[0].mapFn, scm.NewInt(17)).Int() != 18 {
		t.Fatal("computed index mapper did not retain the materialized expression")
	}
}

func TestCapturedConstantDoesNotBecomeComputedColumn(t *testing.T) {
	outerEnv := &scm.Env{VarsNumbered: []scm.Scmer{scm.NewInt(1)}, Outer: &scm.Globalenv}
	captured := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("outer"),
		scm.NewInt(1),
		scm.NewNthLocalVar(0),
	})
	condition := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("number")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("equal??"),
			scm.NewInt(1),
			captured,
		}),
		En:           outerEnv,
		NumVars:      1,
		NumberedOnly: true,
	})

	if bounds := extractBoundaries([]string{"number"}, condition); len(bounds) != 0 {
		t.Fatalf("captured constant boundary = %#v, want no computed column", bounds)
	}
}

func BenchmarkExtractBoundariesEqual(b *testing.B) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal?"),
		scm.NewSymbol("x"),
		scm.NewInt(42),
	})
	condition := buildProc([]string{"x"}, body)
	columns := []string{"id"}
	var storage [4]columnboundaries
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkBoundaries = extractBoundariesInto(storage[:0], columns, condition)
	}
}

func BenchmarkExtractBoundariesAnd(b *testing.B) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("and"),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("equal?"),
			scm.NewSymbol("tenant"),
			scm.NewInt(42),
		}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol(">="),
			scm.NewSymbol("created"),
			scm.NewInt(100),
		}),
	})
	condition := buildProc([]string{"tenant", "created"}, body)
	columns := []string{"tenant", "created"}
	var storage [4]columnboundaries
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkBoundaries = extractBoundariesInto(storage[:0], columns, condition)
	}
}

func TestSimpleAndBoundariesReuseCallerStorage(t *testing.T) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("and"),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("equal?"),
			scm.NewSymbol("tenant"),
			scm.NewInt(42),
		}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol(">="),
			scm.NewSymbol("created"),
			scm.NewInt(100),
		}),
	})
	condition := buildProc([]string{"tenant", "created"}, body)
	columns := []string{"tenant_id", "created_at"}
	var storage [4]columnboundaries

	got := extractBoundariesInto(storage[:0], columns, condition)
	if len(got) != 2 {
		t.Fatalf("simple AND boundaries = %d, want 2", len(got))
	}
	if &got[0] != &storage[0] {
		t.Fatal("simple AND did not reuse caller-owned boundary storage")
	}
	if got[0].col != "tenant_id" || got[0].matcher != EqualMatcher || got[0].lower.Int() != 42 {
		t.Fatalf("unexpected equality boundary: %#v", got[0])
	}
	if got[1].col != "created_at" || got[1].matcher != RangeMatcher || got[1].lower.Int() != 100 || !got[1].lowerInclusive {
		t.Fatalf("unexpected range boundary: %#v", got[1])
	}

	allocs := testing.AllocsPerRun(1000, func() {
		benchmarkBoundaries = extractBoundariesInto(storage[:0], columns, condition)
	})
	if allocs != 0 {
		t.Fatalf("simple AND extraction allocated %.2f times per run, want 0", allocs)
	}
}

// buildProc constructs a Proc with the given param names and body AST.
func buildProc(params []string, body scm.Scmer) scm.Scmer {
	paramSlice := make([]scm.Scmer, len(params))
	for i, p := range params {
		paramSlice[i] = scm.NewSymbol(p)
	}
	return scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice(paramSlice),
		Body:   body,
		En:     &scm.Env{Vars: make(scm.Vars)},
	})
}

// TestBoundaryEqual verifies that equal? produces EqualMatcher.
func TestBoundaryEqual(t *testing.T) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal?"),
		scm.NewSymbol("x"),
		scm.NewString("hello"),
	})
	cond := buildProc([]string{"x"}, body)
	bounds := extractBoundaries([]string{"name"}, cond)
	if len(bounds) != 1 {
		t.Fatalf("expected 1 boundary, got %d", len(bounds))
	}
	if bounds[0].matcher.Kind() != "equal" {
		t.Errorf("expected equal matcher, got %q", bounds[0].matcher.Kind())
	}
	if bounds[0].col != "name" {
		t.Errorf("expected col 'name', got %q", bounds[0].col)
	}
	if bounds[0].lower.String() != "hello" {
		t.Errorf("expected lower 'hello', got %v", bounds[0].lower)
	}
}

func TestBoundaryEqualResolvesNestedOuterNumberedVar(t *testing.T) {
	root := &scm.Env{Vars: make(scm.Vars), VarsNumbered: []scm.Scmer{scm.NewInt(33)}}
	middle := &scm.Env{Vars: make(scm.Vars), VarsNumbered: []scm.Scmer{scm.NewInt(22)}, Outer: root}
	immediate := &scm.Env{Vars: make(scm.Vars), VarsNumbered: []scm.Scmer{scm.NewInt(11)}, Outer: middle}
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal??"),
		scm.NewNthLocalVar(0),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("outer"),
			scm.NewInt(3),
			scm.NewNthLocalVar(0),
		}),
	})
	cond := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("key")}),
		Body:   body,
		En:     immediate,
	})

	bounds := extractBoundaries([]string{"k0"}, cond)
	if len(bounds) != 1 {
		t.Fatalf("expected nested outer key to produce one boundary, got %d", len(bounds))
	}
	if bounds[0].col != "k0" || bounds[0].matcher.Kind() != "equal" {
		t.Fatalf("unexpected nested outer boundary: %#v", bounds[0])
	}
	if got := scm.ToInt(bounds[0].lower); got != 33 {
		t.Fatalf("nested outer boundary = %d, want 33", got)
	}
}

func TestBoundaryEqualMissingOuterFrameIsNotExtracted(t *testing.T) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal??"),
		scm.NewNthLocalVar(0),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("outer"),
			scm.NewInt(2),
			scm.NewNthLocalVar(0),
		}),
	})
	cond := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("key")}),
		Body:   body,
		En:     &scm.Env{Vars: make(scm.Vars)},
	})

	if bounds := extractBoundaries([]string{"k0"}, cond); len(bounds) != 0 {
		t.Fatalf("missing outer frame produced boundaries: %#v", bounds)
	}
}

// TestBoundaryRange verifies that < produces RangeMatcher.
func TestBoundaryRange(t *testing.T) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("<"),
		scm.NewSymbol("x"),
		scm.NewInt(100),
	})
	cond := buildProc([]string{"x"}, body)
	bounds := extractBoundaries([]string{"age"}, cond)
	if len(bounds) != 1 {
		t.Fatalf("expected 1 boundary, got %d", len(bounds))
	}
	if bounds[0].matcher.Kind() != "range" {
		t.Errorf("expected range matcher, got %q", bounds[0].matcher.Kind())
	}
}

func TestComputedBoundaryRejectsQueryBinding(t *testing.T) {
	jsonValue := scm.Globalenv.Vars[scm.Symbol("json_value")]
	if declaration := scm.DeclarationForValue(jsonValue); declaration == nil || !declaration.IsFoldable() {
		t.Fatal("json_value must expose Const metadata through DeclarationForValue")
	}
	path := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("session"),
		scm.NewString("v1"),
	})
	extraction := scm.NewSlice([]scm.Scmer{
		jsonValue,
		scm.NewSymbol("payload"),
		path,
		scm.NewString("UNSIGNED"),
	})
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal??"),
		extraction,
		scm.NewInt(17),
	})
	condition := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("payload")}),
		Body:   body,
		En:     &scm.Globalenv,
	})

	bounds := extractBoundaries([]string{"payload"}, condition)
	if len(bounds) != 0 {
		t.Fatalf("query-bound expression produced a shared computed index: %#v", bounds)
	}
}

func TestComputedBoundaryRejectsNonConstDeclaredCall(t *testing.T) {
	params := []scm.Scmer{scm.NewSymbol("payload")}
	expression := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("concat"),
		scm.NewSymbol("payload"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("uuid")}),
	})
	if isRawDataset(params, expression) {
		t.Fatal("row expression containing non-Const uuid call must not be indexable")
	}
}

func TestParameterizedBooleanFallbackDoesNotCreateVariantIndex(t *testing.T) {
	params := scm.NewSlice([]scm.Scmer{scm.NewSymbol("id")})
	values := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("list"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("session"), scm.NewString("v1")}),
	})
	body := scm.NewSlice([]scm.Scmer{
		scm.Globalenv.Vars[scm.Symbol("sql_in")],
		values,
		scm.NewSymbol("id"),
	})
	condition := scm.NewProcStruct(scm.Proc{Params: params, Body: body, En: &scm.Globalenv})

	if bounds := extractBoundaries([]string{"id"}, condition); len(bounds) != 0 {
		t.Fatalf("parameterized boolean fallback produced computed bounds: %#v", bounds)
	}
}

func TestComputedProcedureRejectsImplicitSessionRead(t *testing.T) {
	read := scm.NewSlice([]scm.Scmer{scm.NewSymbol("session"), scm.NewString("v1")})
	proc := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("value")}),
		Body:   scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewSymbol("value"), read}),
		En:     &scm.Globalenv,
	})

	if !hasImplicitComputeContext(proc) {
		t.Fatal("implicit session read was accepted by a shared computed procedure")
	}
}

func TestScanBufferSizeUsesSmallBufferOnlyForBoundUniqueKey(t *testing.T) {
	tbl := &table{Unique: []uniqueKey{{Id: "PRIMARY", Cols: []string{"tenant_id", "id"}}}}
	point := func(col string, value scm.Scmer) columnboundaries {
		return columnboundaries{
			col: col, matcher: EqualMatcher,
			lower: value, lowerInclusive: true,
			upper: value, upperInclusive: true,
		}
	}

	if got := tbl.scanBufferSize(runtimeScanAccess(boundaries{point("tenant_id", scm.NewInt(9)), point("id", scm.NewInt(42))})); got != uniquePointScanBufferSize {
		t.Fatalf("fully bound unique key buffer = %d, want %d", got, uniquePointScanBufferSize)
	}
	if got := tbl.scanBufferSize(runtimeScanAccess(boundaries{point("id", scm.NewInt(42))})); got != defaultScanBufferSize {
		t.Fatalf("partially bound unique key buffer = %d, want %d", got, defaultScanBufferSize)
	}
	if got := tbl.scanBufferSize(runtimeScanAccess(boundaries{point("tenant_id", scm.NewInt(9)), point("id", scm.NewNil())})); got != defaultScanBufferSize {
		t.Fatalf("NULL unique key buffer = %d, want %d", got, defaultScanBufferSize)
	}
	rangeBoundary := point("id", scm.NewInt(42))
	rangeBoundary.matcher = RangeMatcher
	if got := tbl.scanBufferSize(runtimeScanAccess(boundaries{point("tenant_id", scm.NewInt(9)), rangeBoundary})); got != defaultScanBufferSize {
		t.Fatalf("range-bound unique key buffer = %d, want %d", got, defaultScanBufferSize)
	}
}

func TestWidenDistinctEqualityPointsProducesRange(t *testing.T) {
	left := boundaries{{
		col: "actor_id", matcher: EqualMatcher,
		lower: scm.NewInt(7), lowerInclusive: true,
		upper: scm.NewInt(7), upperInclusive: true,
	}}
	right := boundaries{{
		col: "actor_id", matcher: EqualMatcher,
		lower: scm.NewInt(8), lowerInclusive: true,
		upper: scm.NewInt(8), upperInclusive: true,
	}}

	got := widenBounds(left, right)
	if len(got) != 1 {
		t.Fatalf("expected one widened boundary, got %d", len(got))
	}
	if got[0].matcher.Kind() != "range" {
		t.Fatalf("widened matcher = %q, want range", got[0].matcher.Kind())
	}
	if got[0].lower.String() != "7" || got[0].upper.String() != "8" {
		t.Fatalf("widened bounds = [%v..%v], want [7..8]", got[0].lower, got[0].upper)
	}
}

func TestWidenNullAndValueEqualityPointsProducesRange(t *testing.T) {
	left := boundaries{{
		col: "actor_id", matcher: EqualMatcher,
		lower: scm.NewNil(), lowerInclusive: true,
		upper: scm.NewNil(), upperInclusive: true,
	}}
	right := boundaries{{
		col: "actor_id", matcher: EqualMatcher,
		lower: scm.NewInt(8), lowerInclusive: true,
		upper: scm.NewInt(8), upperInclusive: true,
	}}

	got := widenBounds(left, right)
	if len(got) != 1 {
		t.Fatalf("expected one widened boundary, got %d", len(got))
	}
	if got[0].matcher.Kind() != "range" {
		t.Fatalf("NULL/value widened matcher = %q, want range", got[0].matcher.Kind())
	}
	if !got[0].lower.IsNil() || got[0].upper.Int() != 8 {
		t.Fatalf("NULL/value widened bounds = [%v,%v], want [NULL,8]", got[0].lower, got[0].upper)
	}
	if !got[0].lowerInclusive || !got[0].upperInclusive {
		t.Fatal("NULL/value widened bounds must include both original points")
	}
}

func TestEffectiveBoundaryInclusivenessUsesIndexedRange(t *testing.T) {
	bounds := boundaries{
		{col: "discount", matcher: RangeMatcher, lowerInclusive: true, upperInclusive: true},
		{col: "quantity", matcher: RangeMatcher, lowerInclusive: false, upperInclusive: false},
	}
	access := runtimeScanAccess(bounds)
	lowerInclusive, upperInclusive := effectiveBoundaryInclusiveness(access, newScanIndexBounds(access))
	if !lowerInclusive || !upperInclusive {
		t.Fatalf("effective boundary inclusiveness = (%t, %t), want (true, true)", lowerInclusive, upperInclusive)
	}
}

// TestBoundaryLikePrefixIsRange verifies that prefix LIKE "foo%" becomes RangeMatcher.
func TestBoundaryLikePrefixIsRange(t *testing.T) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("strlike"),
		scm.NewSymbol("x"),
		scm.NewString("foo%"),
		scm.NewString("utf8mb4_bin"),
	})
	cond := buildProc([]string{"x"}, body)
	bounds := extractBoundaries([]string{"name"}, cond)
	if len(bounds) != 1 {
		t.Fatalf("expected 1 boundary, got %d", len(bounds))
	}
	if bounds[0].matcher.Kind() != "range" {
		t.Errorf("expected range matcher for prefix LIKE, got %q", bounds[0].matcher.Kind())
	}
}

// TestBoundaryLikeNonPrefix verifies that "%Klaus%" produces LikeMatcher.
func TestBoundaryLikeNonPrefix(t *testing.T) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("strlike"),
		scm.NewSymbol("x"),
		scm.NewString("%Klaus%"),
	})
	cond := buildProc([]string{"x"}, body)
	bounds := extractBoundaries([]string{"name"}, cond)
	if len(bounds) != 1 {
		t.Fatalf("expected 1 boundary, got %d", len(bounds))
	}
	if bounds[0].matcher.Kind() != "like" {
		t.Errorf("expected like matcher, got %q", bounds[0].matcher.Kind())
	}
	if !bounds[0].matcher.IsPointLike() {
		t.Error("LIKE matcher should be point-like")
	}
	if bounds[0].matcher.IsSorted() {
		t.Error("LIKE matcher should not be sorted")
	}
}

func TestRecSetFilterBoundariesKeepLikeAndMembershipHooks(t *testing.T) {
	owner := &recSet{}
	schema := scm.NewSlice([]scm.Scmer{
		newScanAccessHeader(1, scanAccessConsumerScan, 0, -1),
		scm.NewString("like"), scm.NewString("search"), newScanAccessBoundaryMeta(0, 0, 3), scm.NewString(""),
	})
	access := recSetScanAccess(owner, schema, []scm.Scmer{scm.NewString("%needle%")})
	if access.len() != 2 {
		t.Fatalf("combined RecSet/LIKE boundaries = %d, want 2", access.len())
	}
	if access.boundary(0).matcher != LikeMatcher || access.boundary(1).matcher != RecSetMatcher {
		t.Fatalf("combined matcher order = (%s, %s), want (like, recset)",
			access.boundary(0).matcher.Kind(), access.boundary(1).matcher.Kind())
	}
	if !access.boundary(1).lower.IsCustom(TagRecSet) || RecSetFromScmer(access.boundary(1).lower) != owner {
		t.Fatal("RecSet boundary did not retain the exact input membership")
	}
}

// TestMatcherIsPointLike verifies IsPointLike for all matcher types.
func TestMatcherIsPointLike(t *testing.T) {
	if !EqualMatcher.IsPointLike() {
		t.Error("EqualMatcher should be point-like")
	}
	if !LikeMatcher.IsPointLike() {
		t.Error("LikeMatcher should be point-like")
	}
	if RangeMatcher.IsPointLike() {
		t.Error("RangeMatcher should not be point-like")
	}
}

// TestMatcherIsSorted verifies IsSorted for all matcher types.
func TestMatcherIsSorted(t *testing.T) {
	if !EqualMatcher.IsSorted() {
		t.Error("EqualMatcher should be sorted")
	}
	if !RangeMatcher.IsSorted() {
		t.Error("RangeMatcher should be sorted")
	}
	if LikeMatcher.IsSorted() {
		t.Error("LikeMatcher should not be sorted")
	}
}

// TestRowWithinBoundsEqual verifies sorted (equal) column matching via lower/upper.
func TestRowWithinBoundsEqual(t *testing.T) {
	idx := &StorageIndex{Cols: []string{"id"}, ColMatchers: []IndexAnalyzer{EqualMatcher}}
	access := runtimeScanAccess(boundaries{{col: "id", matcher: EqualMatcher, lower: scm.NewInt(5), upper: scm.NewInt(5)}})
	indexBounds := newScanIndexBounds(access)

	inRange, _ := idx.rowWithinBounds(access, indexBounds, 1, 0, 1, 0, true, true, func(i int) scm.Scmer { return scm.NewInt(5) })
	if !inRange {
		t.Error("expected match for equal value")
	}
	inRange, beyond := idx.rowWithinBounds(access, indexBounds, 1, 0, 1, 0, true, true, func(i int) scm.Scmer { return scm.NewInt(10) })
	if inRange {
		t.Error("expected no match for different value")
	}
	if !beyond {
		t.Error("expected beyond=true for value > equal point")
	}
}

// TestRowWithinBoundsLike verifies that LIKE columns are skipped in rowWithinBounds.
func TestRowWithinBoundsLike(t *testing.T) {
	idx := &StorageIndex{Cols: []string{"name"}, ColMatchers: []IndexAnalyzer{LikeMatcher}}
	access := runtimeScanAccess(boundaries{{col: "name", matcher: LikeMatcher, lower: scm.NewString("%Klaus%"), upper: scm.NewString("%Klaus%")}})

	// rowWithinBounds skips non-sorted columns entirely
	inRange, _ := idx.rowWithinBounds(access, newScanIndexBounds(access), 1, -1, 0, 0, true, true, func(i int) scm.Scmer { return scm.NewString("anything") })
	if !inRange {
		t.Error("expected inRange=true (LIKE skipped in rowWithinBounds)")
	}
}

// TestMatcherKindEqual verifies index deduplication by kind.
func TestMatcherKindEqual(t *testing.T) {
	if !matcherKindEqual(EqualMatcher, EqualMatcher) {
		t.Error("same matcher should be kind-equal")
	}
	if matcherKindEqual(EqualMatcher, LikeMatcher) {
		t.Error("different matchers should not be kind-equal")
	}
	if matcherKindEqual(RangeMatcher, LikeMatcher) {
		t.Error("range and like should not be kind-equal")
	}
}

func TestSortedIndexMatchersAreInterchangeable(t *testing.T) {
	for _, query := range []IndexAnalyzer{EqualMatcher, RangeMatcher} {
		for _, indexed := range []IndexAnalyzer{nil, EqualMatcher, RangeMatcher} {
			if !indexMatcherCompatible(query, indexed) {
				t.Fatalf("query matcher %v did not accept sorted index matcher %v", query, indexed)
			}
		}
		if indexMatcherCompatible(query, LikeMatcher) {
			t.Fatalf("query matcher %v accepted a custom LIKE index", query)
		}
	}
}

func TestEqualityPrefixRequiresActiveSortedIndex(t *testing.T) {
	idx := &StorageIndex{Cols: []string{"a", "b"}}
	shard := &storageShard{Indexes: []*StorageIndex{idx}}

	if shard.hasEqualityIndexPrefix(nil, []string{"a"}) {
		t.Fatal("inactive sorted index must not select the sparse RecSet lookup path")
	}
	idx.baseState.active = true
	if !shard.hasEqualityIndexPrefix(nil, []string{"a"}) {
		t.Fatal("active sorted compound index must cover its equality prefix")
	}
	idx.ColMatchers = []IndexAnalyzer{LikeMatcher, nil}
	if shard.hasEqualityIndexPrefix(nil, []string{"a"}) {
		t.Fatal("custom analyzer index must not cover an equality prefix")
	}
}

func TestSnapshotDropsLegacySortedMatcherMetadata(t *testing.T) {
	legacy := &StorageIndex{
		Cols:        []string{"status", "id"},
		ColMatchers: []IndexAnalyzer{EqualMatcher, RangeMatcher},
	}
	snapshot := snapshotIndexesForRebuild([]*StorageIndex{legacy})
	if len(snapshot) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(snapshot))
	}
	if len(snapshot[0].ColMatchers) != 0 {
		t.Fatalf("snapshot retained sorted matcher metadata: %#v", snapshot[0].ColMatchers)
	}
}
