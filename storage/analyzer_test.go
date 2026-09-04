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

func TestCompileScanAccessBindsRuntimeValuesWithoutAllocation(t *testing.T) {
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
	var storage [scanAnalyzeScratchCapacity]columnboundaries
	bound, valid := bindCompiledScanAccess(schema, values, storage[:0])
	if !valid || len(bound) != 2 {
		t.Fatalf("bindCompiledScanAccess returned valid=%v boundaries=%d", valid, len(bound))
	}
	if bound[0].col != "tenant" || bound[0].matcher != EqualMatcher || bound[0].lower.Int() != 7 {
		t.Fatalf("unexpected compiled equality: %#v", bound[0])
	}
	if bound[1].col != "created_at" || bound[1].matcher != RangeMatcher || bound[1].lower.Int() != 100 || !bound[1].lowerInclusive {
		t.Fatalf("unexpected compiled range: %#v", bound[1])
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		_, _ = bindCompiledScanAccess(schema, values, storage[:0])
	}); allocs != 0 {
		t.Fatalf("compiled scan access binding allocated %.2f times per run, want 0", allocs)
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
	var storage [scanAnalyzeScratchCapacity]columnboundaries
	bound, valid := bindCompiledScanAccess(schema, nil, storage[:0])
	if !valid || len(bound) != 1 || !bound[0].lowerBatch || !bound[0].upperBatch ||
		bound[0].lowerBatchSubidx != 0 || bound[0].upperBatchSubidx != 0 {
		t.Fatalf("unexpected compiled batch boundary: valid=%v boundary=%#v", valid, bound)
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
	var storage [scanAnalyzeScratchCapacity]columnboundaries
	bound, valid := bindCompiledScanAccess(schema, values, storage[:0])
	if !valid || len(bound) != 3 {
		t.Fatalf("compiled hooks returned valid=%v boundaries=%d", valid, len(bound))
	}
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

func TestCompileScanAccessAcceptsParsedSourceInfo(t *testing.T) {
	call := scm.Read(t.Name(), `(scan nil table_value
		(list "id") (lambda (id) (equal?? id wanted_id))
		(list "id") (lambda (acc id) id) nil nil false)`)
	items, ok := scmerSlice(call)
	if !ok || len(items) < 5 {
		t.Fatalf("unexpected parsed scan expression: %s", scm.SerializeToString(call, &scm.Globalenv))
	}
	if _, bindings, compiled := compileScanAccess(items[3], items[4]); !compiled || len(bindings) != 1 {
		t.Fatalf("parsed scan access compiled=%v bindings=%d", compiled, len(bindings))
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
	if !sortedBoundariesCoverCondition([]string{"meta_key"}, condition, bounds) {
		t.Fatal("simple equality should be covered by its extracted boundary")
	}
	bounds[0].upper = scm.NewString("other")
	if sortedBoundariesCoverCondition([]string{"meta_key"}, condition, bounds) {
		t.Fatal("different boundary must not cover the condition")
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

	if got := tbl.scanBufferSize(boundaries{point("tenant_id", scm.NewInt(9)), point("id", scm.NewInt(42))}); got != uniquePointScanBufferSize {
		t.Fatalf("fully bound unique key buffer = %d, want %d", got, uniquePointScanBufferSize)
	}
	if got := tbl.scanBufferSize(boundaries{point("id", scm.NewInt(42))}); got != defaultScanBufferSize {
		t.Fatalf("partially bound unique key buffer = %d, want %d", got, defaultScanBufferSize)
	}
	if got := tbl.scanBufferSize(boundaries{point("tenant_id", scm.NewInt(9)), point("id", scm.NewNil())}); got != defaultScanBufferSize {
		t.Fatalf("NULL unique key buffer = %d, want %d", got, defaultScanBufferSize)
	}
	rangeBoundary := point("id", scm.NewInt(42))
	rangeBoundary.matcher = RangeMatcher
	if got := tbl.scanBufferSize(boundaries{point("tenant_id", scm.NewInt(9)), rangeBoundary}); got != defaultScanBufferSize {
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
	lower, _ := indexFromBoundaries(bounds)
	lowerInclusive, upperInclusive := effectiveBoundaryInclusiveness(bounds, lower)
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
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("strlike"),
		scm.NewSymbol("text"),
		scm.NewString("%needle%"),
	})
	condition := buildProc([]string{"text"}, body)
	owner := &recSet{}
	bounds := recSetFilterBoundaries(owner, []string{"search"}, condition)
	if len(bounds) != 2 {
		t.Fatalf("combined RecSet/LIKE boundaries = %d, want 2", len(bounds))
	}
	if bounds[0].matcher != LikeMatcher || bounds[1].matcher != RecSetMatcher {
		t.Fatalf("combined matcher order = (%s, %s), want (like, recset)",
			bounds[0].matcher.Kind(), bounds[1].matcher.Kind())
	}
	if !bounds[1].lower.IsCustom(TagRecSet) || RecSetFromScmer(bounds[1].lower) != owner {
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
	lower := []scm.Scmer{scm.NewInt(5)}

	inRange, _ := idx.rowWithinBounds(boundaries{}, 1, lower, scm.NewInt(5), true, func(i int) scm.Scmer { return scm.NewInt(5) })
	if !inRange {
		t.Error("expected match for equal value")
	}
	inRange, beyond := idx.rowWithinBounds(boundaries{}, 1, lower, scm.NewInt(5), true, func(i int) scm.Scmer { return scm.NewInt(10) })
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
	lower := []scm.Scmer{scm.NewString("%Klaus%")}

	// rowWithinBounds skips non-sorted columns entirely
	inRange, _ := idx.rowWithinBounds(boundaries{}, 1, lower, scm.NewString("%Klaus%"), true, func(i int) scm.Scmer { return scm.NewString("anything") })
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
