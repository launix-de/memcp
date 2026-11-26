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
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

func TestComputeProxyRestoresPersistedProcedures(t *testing.T) {
	// Literal historical JSON, independent of the current procedure encoder.
	// Both symbolic and optimized local-variable bodies occur in persisted files.
	for _, encoded := range []string{
		`[{"symbol":"lambda"},[{"symbol":"value"}],[{"symbol":"*"},{"symbol":"value"},2]]`,
		`[{"symbol":"lambda"},[{"symbol":"value"}],[{"symbol":"*"},{"var":0},2],1]`,
		`[{"symbol":"lambda"},[{"symbol":"value"}],[{"symbol":"if"},false,[{"symbol":"table"},"memcp-tests","items"],[{"symbol":"*"},{"symbol":"value"},2]]]`,
		`[{"symbol":"lambda"}`, // corrupt JSON must fail, not become a nil computor
	} {
		for version := uint8(0); version <= 2; version++ {
			t.Run(fmt.Sprintf("version%d/%s", version, encoded), func(t *testing.T) {
				var wire bytes.Buffer
				// Magic 50 is already consumed by the shard loader. The shared
				// v0 prefix carries count, input names, and computor JSON.
				for _, field := range []any{version, uint32(1), uint16(1), uint16(5)} {
					if err := binary.Write(&wire, binary.LittleEndian, field); err != nil {
						t.Fatal(err)
					}
				}
				wire.WriteString("value")
				if err := binary.Write(&wire, binary.LittleEndian, uint32(len(encoded))); err != nil {
					t.Fatal(err)
				}
				wire.WriteString(encoded)
				// Uncompressed, no main storage, empty delta and valid-mask.
				wire.Write(make([]byte, 10))
				if version > 0 {
					wire.WriteByte(0) // ordinary, unordered computed column
				}
				if !json.Valid([]byte(encoded)) {
					defer func() {
						if got := recover(); got == nil || !strings.Contains(fmt.Sprint(got), "decode computor") {
							t.Fatalf("invalid computor JSON panic = %v", got)
						}
					}()
					new(StorageComputeProxy).Deserialize(&wire)
					return
				}
				var proxy StorageComputeProxy
				if got := proxy.Deserialize(&wire); got != 1 {
					t.Fatalf("restored count = %d, want 1", got)
				}
				if strings.Contains(encoded, `"symbol":"table"`) {
					call := proxy.computor.Proc().Body.Slice()[2].Slice()
					if len(call) != 5 || call[2].Bool() || call[4].Bool() {
						t.Fatal("historical computor did not migrate its native table call")
					}
				}
				for _, value := range []int64{5, 100, -3} {
					if got := scm.Apply(proxy.computor, scm.NewInt(value)); !scm.Equal(got, scm.NewInt(value*2)) {
						t.Fatalf("restored computor(%d) = %s, want %d", value, scm.String(got), value*2)
					}
				}
			})
		}
	}
}

func serializeScmerForTest(v scm.Scmer) string {
	var b bytes.Buffer
	scm.Serialize(&b, v, &scm.Globalenv)
	return b.String()
}

func triggerPlanStringForTest(tr TriggerDescription) string {
	if tr.Func.IsProc() {
		return serializeScmerForTest(tr.Func.Proc().Body)
	}
	return serializeScmerForTest(tr.Func)
}

func findTriggerByPrefixAndTiming(triggers []TriggerDescription, prefix string, timing TriggerTiming) (TriggerDescription, bool) {
	for _, tr := range triggers {
		if strings.HasPrefix(tr.Name, prefix) && tr.Timing == timing {
			return tr, true
		}
	}
	return TriggerDescription{}, false
}

func listAst(items ...scm.Scmer) scm.Scmer {
	result := make([]scm.Scmer, 1+len(items))
	result[0] = scm.NewSymbol("list")
	copy(result[1:], items)
	return scm.NewSlice(result)
}

func lambdaAst(params []string, body scm.Scmer) scm.Scmer {
	paramItems := make([]scm.Scmer, len(params))
	for i, p := range params {
		paramItems[i] = scm.NewSymbol(p)
	}
	return scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice(paramItems),
		body,
	})
}

func TestStorageAnalyzersRecognizeLoweredSpecialFormHeads(t *testing.T) {
	constantOne := scm.NewSlice([]scm.Scmer{
		scm.Globalenv.Vars[scm.Symbol("lambda")],
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("acc")}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("+"), scm.NewSymbol("acc"), scm.NewInt(1)}),
	})
	if constantOne.Slice()[0].IsSymbol() {
		t.Fatal("optimizer did not lower lambda head for the regression fixture")
	}
	if !isConstantOneAggregate(constantOne) {
		t.Fatal("constant COUNT mapreducer was hidden by its lowered lambda head")
	}

	outer := scm.Globalenv.Vars[scm.Symbol("outer")]
	if !scanSymbolIs(outer, "outer") {
		t.Fatal("scan analyzer did not recognize the lowered outer form")
	}
}

func nestedScanAst(schema, table, outerParam string) scm.Scmer {
	return scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("scan"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("session"), scm.NewString("__memcp_tx")}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("table"), scm.NewString(schema), scm.NewBool(false), scm.NewString(table), scm.NewBool(false)}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), newScanAccessSchema(scanAccessConsumerScan, nil, -1)}),
		listAst(),
		listAst(scm.NewString("ref_id")),
		lambdaAst([]string{"src.ref_id"}, scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("equal?"),
			scm.NewSymbol("src.ref_id"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("outer"), scm.NewInt(1), scm.NewSymbol(outerParam)}),
		})),
		listAst(scm.NewString("val")),
		lambdaAst([]string{"acc", "val"}, scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("+"), scm.NewSymbol("acc"), scm.NewSymbol("val"),
		})),
		scm.NewInt(0),
		scm.NewSymbol("+"),
	})
}

func TestExtractScanJoinInfoIncludesDynamicTablePlan(t *testing.T) {
	dynamicTable := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("begin"),
		nestedScanAst("dynamic_dependency", "acl", "driver_id"),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("table"),
			scm.NewString("dynamic_dependency"), scm.NewBool(false),
			scm.NewString("driver"), scm.NewBool(false),
		}),
	})
	computor := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("scan"),
		scm.NewNil(),
		dynamicTable,
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), newScanAccessSchema(scanAccessConsumerScan, nil, -1)}),
		listAst(),
		listAst(),
		lambdaAst(nil, scm.NewBool(true)),
		listAst(),
		lambdaAst([]string{"acc"}, scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("+"), scm.NewSymbol("acc"), scm.NewInt(1),
		})),
		scm.NewInt(0),
		scm.NewSymbol("+"),
	})

	refs := extractScanJoinInfo(computor)
	found := false
	for _, ref := range refs {
		if ref.schema == "dynamic_dependency" && ref.table == "acl" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("dynamic table plan dependency missing from %#v", refs)
	}
}

func TestComputeTriggersGuardRelevantSourceColumns(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-compute-trigger-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tcomputetrigger")

	CreateDatabase("tcomputetrigger", false)
	base, _ := CreateTable("tcomputetrigger", "base", Safe, false)
	src, _ := CreateTable("tcomputetrigger", "src", Safe, false)

	base.CreateColumn("id", "INT", nil, nil)
	base.CreateColumn("ref_id", "INT", nil, nil)
	base.CreateColumn("cached", "INT", nil, nil)
	src.CreateColumn("ref_id", "INT", nil, nil)
	src.CreateColumn("val", "INT", nil, nil)
	src.CreateColumn("note", "TEXT", nil, nil)

	computor := lambdaAst([]string{"ref_id"}, nestedScanAst("tcomputetrigger", "src", "ref_id"))
	computor.Slice()[2].Slice()[3] = scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice([]scm.Scmer{
		scm.NewSlice([]scm.Scmer{newScanAccessHeader(0, scanAccessConsumerScan, 0, -1), scm.NewNil(),
			scm.NewSlice([]scm.Scmer{scm.NewString("ref_id")})}),
	})})
	refs := extractScanJoinInfo(computor)
	base.registerComputeTriggers("cached", computor)

	prefix := ".cache:base:cached|scan0|src|"
	var triggerCount int
	for _, tr := range src.Triggers {
		if strings.HasPrefix(tr.Name, prefix) {
			triggerCount++
		}
	}
	if triggerCount != 3 {
		t.Fatalf("compute dependency trigger count = %d, want 3 (refs=%#v body=%s)", triggerCount, refs, serializeScmerForTest(computor))
	}

	tr, ok := findTriggerByPrefixAndTiming(src.Triggers, prefix, AfterUpdate)
	if !ok {
		t.Fatal("missing AfterUpdate compute dependency trigger")
	}
	plan := triggerPlanStringForTest(tr)
	for _, want := range []string{`(get_assoc OLD "ref_id")`, `(get_assoc NEW "ref_id")`, `(get_assoc OLD "val")`, `(get_assoc NEW "val")`} {
		if !strings.Contains(plan, want) {
			t.Fatalf("compute trigger plan missing %s:\n%s", want, plan)
		}
	}
	if strings.Contains(plan, `"note"`) {
		t.Fatalf("compute trigger plan should ignore unrelated note column:\n%s", plan)
	}
}

func TestLookupComputeTriggersInvalidateMatchingRows(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-lookup-trigger-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tlookuptrigger")

	CreateDatabase("tlookuptrigger", false)
	base, _ := CreateTable("tlookuptrigger", "base", Safe, false)
	src, _ := CreateTable("tlookuptrigger", "src", Safe, false)
	base.CreateColumn("id", "INT", nil, nil)
	base.CreateColumn("ref_id", "INT", nil, nil)
	base.CreateColumn("cached", "INT", nil, nil)
	src.CreateColumn("ref_id", "INT", nil, nil)
	src.CreateColumn("val", "INT", nil, nil)

	computorSource := `(lambda (ref_id)
		(scan nil (table "tlookuptrigger" false "src" false)
			'(369435906932736) '()
			'("ref_id") (lambda (source_ref_id) (equal? source_ref_id (outer 1 ref_id)))
			'("val") (lambda (acc val) val)
			0 (lambda (old value) value) false))`
	rawComputor := scm.Eval(scm.Read("raw lookup trigger test", computorSource, true), &scm.Globalenv)
	compiledComputor := scm.Eval(scm.Optimize(scm.Read("compiled lookup trigger test", computorSource, true), &scm.Globalenv, nil), &scm.Globalenv)
	for variant, computor := range map[string]scm.Scmer{"raw": rawComputor, "compiled": compiledComputor} {
		refs := extractScanJoinInfo(computor)
		if len(refs) != 1 || len(refs[0].srcCols) != 1 || refs[0].srcCols[0] != "ref_id" || refs[0].inputCols[0] != "ref_id" {
			t.Fatalf("%s lookup relation was not extracted: %#v", variant, refs)
		}
	}
	base.registerComputeTriggers("cached", compiledComputor)

	tr, ok := findTriggerByPrefixAndTiming(src.Triggers, ".cache:base:cached|scan0|src|", AfterUpdate)
	if !ok {
		t.Fatal("missing AfterUpdate lookup dependency trigger")
	}
	plan := triggerPlanStringForTest(tr)
	if !strings.Contains(plan, `"$invalidate:cached"`) {
		t.Fatalf("lookup-shaped compute trigger must invalidate matching cache rows:\n%s", plan)
	}
	if strings.Contains(plan, `(invalidatecolumn`) {
		t.Fatalf("lookup-shaped compute trigger must not invalidate the complete cache:\n%s", plan)
	}
}

func TestQualifiedLookupComputeTriggerUsesPhysicalTargetColumn(t *testing.T) {
	oldBasepath := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	const schemaName = "tqualifiedlookuptrigger"
	defer databases.Remove(schemaName)
	CreateDatabase(schemaName, false)
	base, _ := CreateTable(schemaName, "base", Safe, false)
	src, _ := CreateTable(schemaName, "src", Safe, false)
	base.CreateColumn("ref_id", "INT", nil, nil)
	base.CreateColumn("cached", "INT", nil, nil)
	src.CreateColumn("ref_id", "INT", nil, nil)
	src.CreateColumn("val", "INT", nil, nil)
	base.Insert([]string{"ref_id", "cached"}, [][]scm.Scmer{{scm.NewInt(7), scm.NewInt(0)}}, nil, scm.NewNil(), false, nil)

	computorSource := `(lambda (base.ref_id)
		(scan nil (table "tqualifiedlookuptrigger" false "src" false)
			'(369435906932736) '()
			'("ref_id") (lambda (source_ref_id) (equal? source_ref_id (outer 1 base.ref_id)))
			'("val") (lambda (acc val) val)
			0 (lambda (old value) value) false))`
	computor := scm.Read(t.Name(), computorSource, true)
	refs := extractScanJoinInfo(computor)
	if len(refs) != 1 || len(refs[0].srcCols) != 1 || refs[0].srcCols[0] != "ref_id" {
		t.Fatalf("lookup relation not extracted: %#v", refs)
	}
	// Feed the qualified planner input explicitly; other tests may normalize the lambda.
	refs[0].inputCols = []string{"base.ref_id"}
	base.registerComputeTriggersWithRefs("cached", computor, refs)
	tr, ok := findTriggerByPrefixAndTiming(src.Triggers, ".cache:base:cached|scan0|src|", AfterInsert)
	if !ok {
		t.Fatal("missing qualified lookup dependency trigger")
	}
	// A new source row must not abort its INSERT while the cache is invalidated.
	src.Insert([]string{"ref_id", "val"}, [][]scm.Scmer{{scm.NewInt(7), scm.NewInt(11)}}, nil, scm.NewNil(), false, nil)
	plan := triggerPlanStringForTest(tr)
	if !strings.Contains(plan, `"ref_id"`) || strings.Contains(plan, `"base.ref_id"`) {
		t.Fatalf("dependency trigger scans a nonphysical target column:\n%s", plan)
	}
}

func TestExtractScanJoinInfoUsesCompiledAccessWhenResidualIsEmpty(t *testing.T) {
	source := `(lambda (ref_id)
		(scan nil (table "tcompileddependency" false "src" false)
			'()
			(list ref_id)
			'() (lambda () true)
			'("value") (lambda (acc value) value)
			nil nil false))`
	computor := scm.Read(t.Name(), source, true)
	root := stripSourceInfo(computor).Slice()
	scan := stripSourceInfo(root[2]).Slice()
	schema := scm.NewSlice([]scm.Scmer{
		newScanAccessHeader(1, scanAccessConsumerScan, 0, -1),
		newScanBoundarySpec("source_ref", EqualMatcher, 0, 0, true, true, "", false, -1, nil, nil, "", false),
	})
	scan[3] = scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), schema})
	refs := extractScanJoinInfo(computor)
	if len(refs) != 1 || refs[0].schema != "tcompileddependency" || refs[0].table != "src" ||
		len(refs[0].srcCols) != 1 || refs[0].srcCols[0] != "source_ref" ||
		len(refs[0].inputCols) != 1 || refs[0].inputCols[0] != "ref_id" {
		t.Fatalf("compiled access dependency was not extracted: %#v; plan=%s", refs, serializeScmerForTest(computor))
	}
}

func TestExplicitReadDependenciesSurvivePrunedFilters(t *testing.T) {
	for _, residual := range []bool{false, true} {
		for _, dynamicValues := range []bool{false, true} {
			t.Run(fmt.Sprintf("residual=%t/dynamic_values=%t", residual, dynamicValues), func(t *testing.T) {
				computor := scm.Read(t.Name(), `(lambda (ref_id)
					(scan nil (table "tcompileddependency" false "src" false)
						'() (list ref_id 1)
						'() (lambda () true)
						'("value") (lambda (acc value) value)
						nil nil false))`, true)
				root := stripSourceInfo(computor).Slice()
				scan := stripSourceInfo(root[2]).Slice()
				readColumns := scm.NewSlice([]scm.Scmer{scm.NewString("source_ref"), scm.NewString("allowed"), scm.NewString("priority"), scm.NewString("weight")})
				schema := scm.NewSlice([]scm.Scmer{
					scm.NewSlice([]scm.Scmer{newScanAccessHeader(1, scanAccessConsumerScan, 0, -1), scm.NewNil(), readColumns}),
					newScanBoundarySpec("source_ref", EqualMatcher, 0, 0, true, true, "", false, -1, nil, nil, "", false),
				})
				scan[3] = scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), schema})
				if residual {
					scan[5] = scm.Read(t.Name(), `'("source_ref")`, true)
					scan[6] = scm.Read(t.Name(), `(lambda (source_ref) (equal? source_ref (outer 1 ref_id)))`, true)
				}
				if dynamicValues {
					scan[4] = scm.NewSymbol("runtime_values")
				}
				refs := extractScanJoinInfo(computor)
				if len(refs) != 1 {
					t.Fatalf("expected one source dependency, got %#v", refs)
				}
				if refs[0].unknownReads {
					t.Fatal("explicit complete readset was classified as unknown")
				}
				for _, column := range []string{"source_ref", "allowed", "priority", "weight"} {
					if !slices.Contains(refs[0].condCols, column) {
						t.Errorf("explicit predicate dependency %q missing from %#v", column, refs[0])
					}
				}
				if residual || !dynamicValues {
					if strings.Join(refs[0].srcCols, ",") != "source_ref" || strings.Join(refs[0].inputCols, ",") != "ref_id" {
						t.Fatalf("constant/expression filters changed selective join mapping: %#v", refs[0])
					}
				} else if len(refs[0].srcCols) != 0 || len(refs[0].inputCols) != 0 {
					t.Fatalf("dynamic access values manufactured a reverse mapping: %#v", refs[0])
				}
			})
		}
	}
}

func TestLegacyScanAccessDoesNotClaimCompleteReadDependencies(t *testing.T) {
	for _, access := range []scm.Scmer{
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice([]scm.Scmer{
			newScanAccessHeader(0, scanAccessConsumerScan, 0, -1),
		})}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice([]scm.Scmer{
			scm.NewSlice([]scm.Scmer{newScanAccessHeader(0, scanAccessConsumerScan, 0, -1), scm.NewNil()}),
		})}),
		scm.NewSymbol("dynamic_access"),
	} {
		scan := nestedScanAst("legacy", "src", "ref_id")
		scan.Slice()[3] = access
		refs := extractScanJoinInfo(lambdaAst([]string{"ref_id"}, scan))
		if len(refs) != 1 || !refs[0].unknownReads {
			t.Fatalf("legacy/dynamic access claimed a complete filter readset: %#v", refs)
		}
		if got := scanRelevantSourceCols(refs[0], nil); len(got) != 0 {
			t.Fatalf("legacy access restricted update invalidation to partial readset: %v", got)
		}
		if strings.Join(refs[0].srcCols, ",") != "ref_id" || strings.Join(refs[0].inputCols, ",") != "ref_id" {
			t.Fatalf("legacy read uncertainty discarded selective reverse join: %#v", refs[0])
		}
	}
	scan := nestedScanAst("legacy", "src", "ref_id")
	scan.Slice()[3] = scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice(nil)})
	refs := extractScanJoinInfo(lambdaAst([]string{"ref_id"}, scan))
	if len(refs) != 1 || refs[0].unknownReads || len(scanRelevantSourceCols(refs[0], nil)) != 2 {
		t.Fatalf("empty access lost its complete residual/mapper readset: %#v", refs)
	}
}

func TestNestedComputorLambdaDoesNotReplaceTargetInputs(t *testing.T) {
	nested := lambdaAst([]string{"group_key"}, nestedScanAst("nested_dependency", "src", "group_key"))
	computor := lambdaAst([]string{"ref_id"}, nested)
	refs := extractScanJoinInfo(computor)
	if len(refs) != 1 {
		t.Fatalf("expected one nested source dependency, got %#v", refs)
	}
	if len(refs[0].srcCols) != 0 || len(refs[0].inputCols) != 0 {
		t.Fatalf("nested cache parameters were mistaken for target-table inputs: %#v", refs[0])
	}
}

// A schema written by an older binary can contain generated guards that omit
// compiled-only predicates. Rebinding their target must also replace that code.
func TestRestoredComputeDependencyTriggersRefreshGuards(t *testing.T) {
	oldBasepath := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	const schemaName = "trestoredcomputedependency"
	defer databases.Remove(schemaName)
	CreateDatabase(schemaName, false)
	base, _ := CreateTable(schemaName, "base", Safe, false)
	src, _ := CreateTable(schemaName, "src", Safe, false)
	base.CreateColumn("ref_id", "INT", nil, nil)
	base.CreateColumn("cached", "INT", nil, nil)
	src.CreateColumn("ref_id", "INT", nil, nil)
	src.CreateColumn("val", "INT", nil, nil)
	src.CreateColumn("allowed", "INT", nil, nil)

	for _, orc := range []bool{false, true} {
		t.Run(fmt.Sprintf("orc=%t", orc), func(t *testing.T) {
			computor := lambdaAst([]string{"ref_id"}, nestedScanAst(schemaName, "src", "ref_id"))
			computor.Slice()[2].Slice()[3] = scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice([]scm.Scmer{
				scm.NewSlice([]scm.Scmer{newScanAccessHeader(0, scanAccessConsumerScan, 0, -1), scm.NewNil(),
					scm.NewSlice([]scm.Scmer{scm.NewString("ref_id")})}),
			})})
			register := func() {
				if orc {
					base.registerORCDependencyTriggers("cached", base.Columns[1], extractScanJoinInfo(computor))
				} else {
					base.registerComputeTriggers("cached", computor)
				}
			}
			register()
			prefix := ".cache:base:cached|scan0|src|"
			if orc {
				prefix = ".orcdep:base:cached|scan0|src|"
			}
			old, found := findTriggerByPrefixAndTiming(src.Triggers, prefix, AfterUpdate)
			if !found || strings.Contains(triggerPlanStringForTest(old), `"allowed"`) {
				t.Fatal("fixture must begin with the old incomplete update guard")
			}
			encoded, err := json.Marshal(src.Triggers)
			if err != nil {
				t.Fatal(err)
			}
			var restored []TriggerDescription
			if err := json.Unmarshal(encoded, &restored); err != nil {
				t.Fatal(err)
			}
			src.mu.Lock()
			src.Triggers = restored
			src.mu.Unlock()
			// Exercise lazy compilation too: cached native/vector code from the
			// restored body must not survive replacement of its scalar procedure.
			src.GetTriggers(AfterUpdate)
			before := make([]string, len(src.Triggers))
			for i, tr := range src.Triggers {
				before[i] = tr.Name
			}
			scan := computor.Slice()[2].Slice()
			scan[3] = scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice([]scm.Scmer{
				scm.NewSlice([]scm.Scmer{newScanAccessHeader(0, scanAccessConsumerScan, 0, -1), scm.NewNil(),
					scm.NewSlice([]scm.Scmer{scm.NewString("ref_id"), scm.NewString("allowed")})}),
			})})
			register()
			register()
			after := make([]string, len(src.Triggers))
			for i, tr := range src.Triggers {
				after[i] = tr.Name
			}
			if !slices.Equal(before, after) {
				t.Fatalf("refresh changed trigger count/order: %v -> %v", before, after)
			}
			updated, found := findTriggerByPrefixAndTiming(src.Triggers, prefix, AfterUpdate)
			if !found {
				t.Fatal("updated trigger missing")
			}
			plan := triggerPlanStringForTest(updated)
			for _, want := range []string{`(get_assoc OLD "allowed")`, `(get_assoc NEW "allowed")`} {
				if !strings.Contains(plan, want) {
					t.Fatalf("restored trigger kept obsolete dependency guard, missing %s:\n%s", want, plan)
				}
			}
			parsedBody := stripSourceInfo(scm.Read(t.Name(), plan, true)).Slice()
			guard := scm.Eval(lambdaAst([]string{"OLD", "NEW"}, parsedBody[1]), &scm.Globalenv)
			oldRow := src.rowToDict(dataset{scm.NewInt(7), scm.NewInt(10), scm.NewInt(1)})
			newRow := src.rowToDict(dataset{scm.NewInt(7), scm.NewInt(10), scm.NewInt(0)})
			if !scm.ToBool(scm.Apply(guard, oldRow, newRow)) {
				t.Fatal("updated guard suppressed an allowed-only mutation")
			}
			if scm.ToBool(scm.Apply(guard, oldRow, oldRow)) {
				t.Fatal("updated guard invalidated an unchanged source row")
			}
			if !updated.acquireTarget(nil) {
				t.Fatal("refreshed trigger target remained inert")
			}
			updated.releaseTarget()
		})
	}
}

func TestLookupInvalidationIncludesComputedDeltaRows(t *testing.T) {
	db := newDatabase()
	db.Name = "tlookupdelta"
	tbl := &table{schema: db, Name: "base", Columns: []*column{{Name: "cached"}}}
	shard := &storageShard{
		t:            tbl,
		main_count:   1,
		columns:      make(map[string]ColumnStorage),
		deltaColumns: make(map[string]int),
	}
	proxy := &StorageComputeProxy{
		delta:   map[uint32]scm.Scmer{1: scm.NewInt(20)},
		shard:   shard,
		colName: "cached",
		count:   1,
	}
	proxy.validMask.Set(0, true)
	proxy.validMask.Set(1, true)
	shard.columns["cached"] = proxy

	mapReduceFn := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		scm.Apply(args[1])
		return args[0]
	})
	mr := shard.OpenMapReducer([]string{"$invalidate:cached"}, mapReduceFn, false, 0, nil, nil)
	mr.processDeltaBlock(scm.NewNil(), []uint32{1})
	mr.FlushSideEffects()

	if !proxy.validMask.Get(0) {
		t.Fatal("delta-row invalidation cleared an unrelated main cache row")
	}
	if proxy.validMask.Get(1) {
		t.Fatal("delta-row invalidation did not clear the matching cache row")
	}
	if _, ok := proxy.delta[1]; ok {
		t.Fatal("delta-row invalidation retained the stale cached value")
	}
}

func buildIntStorageForLookupTest(values ...int64) *StorageInt {
	result := new(StorageInt)
	result.prepare()
	for i, value := range values {
		result.scan(uint32(i), scm.NewInt(value))
	}
	result.init(uint32(len(values)))
	for i, value := range values {
		result.build(uint32(i), scm.NewInt(value))
	}
	result.finish()
	return result
}

func buildScmerStorageForLookupTest(values ...int64) *StorageSCMER {
	result := new(StorageSCMER)
	result.init(uint32(len(values)))
	for i, value := range values {
		result.build(uint32(i), scm.NewInt(value))
	}
	result.finish()
	return result
}

func TestLookupInvalidationKeepsCompressedBaseRows(t *testing.T) {
	db := newDatabase()
	db.Name = "tlookupcompressed"
	tbl := &table{schema: db, Name: "base", Columns: []*column{{Name: "input"}, {Name: "cached"}}}
	shard := &storageShard{
		t:            tbl,
		main_count:   3,
		columns:      make(map[string]ColumnStorage),
		deltaColumns: make(map[string]int),
	}
	input := buildScmerStorageForLookupTest(1, 2, 3)
	proxy := &StorageComputeProxy{
		main:       buildIntStorageForLookupTest(11, 12, 13),
		delta:      make(map[uint32]scm.Scmer),
		compressed: true,
		computor: scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
			return scm.NewInt(args[0].Int() + 10)
		}),
		inputCols: []string{"input"},
		shard:     shard,
		colName:   "cached",
		count:     3,
	}
	shard.columns["input"] = input
	shard.columns["cached"] = proxy

	input.SetValue(1, scm.NewInt(20))
	proxy.Invalidate(1)

	if !proxy.compressed {
		t.Fatal("one lookup invalidation discarded the compressed cache base")
	}
	for idx, want := range []int64{11, 30, 13} {
		if got := proxy.GetValue(uint32(idx)).Int(); got != want {
			t.Fatalf("cached row %d = %d, want %d", idx, got, want)
		}
	}
	if len(proxy.delta) != 1 {
		t.Fatalf("sparse override count = %d, want 1", len(proxy.delta))
	}
}

func newAdaptiveLookupProxyForTest(rowCount int) (*StorageComputeProxy, *int) {
	db := newDatabase()
	db.Name = "tlookupadaptive"
	tbl := &table{schema: db, Name: "base", Columns: []*column{{Name: "input"}, {Name: "cached"}}}
	shard := &storageShard{
		t:            tbl,
		main_count:   uint32(rowCount),
		columns:      make(map[string]ColumnStorage),
		deltaColumns: make(map[string]int),
	}
	values := make([]int64, rowCount)
	for i := range values {
		values[i] = int64(i)
	}
	input := buildScmerStorageForLookupTest(values...)
	calls := 0
	proxy := &StorageComputeProxy{
		main:       buildScmerStorageForLookupTest(values...),
		delta:      make(map[uint32]scm.Scmer),
		compressed: true,
		computor: scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
			calls++
			return args[0]
		}),
		inputCols: []string{"input"},
		shard:     shard,
		colName:   "cached",
		count:     uint32(rowCount),
	}
	shard.columns["input"] = input
	shard.columns["cached"] = proxy
	return proxy, &calls
}

func TestLookupInvalidationFallsBackForExpensiveBroadFanout(t *testing.T) {
	proxy, calls := newAdaptiveLookupProxyForTest(64)
	proxy.lastRecomputeNs.Store(1)
	recids := make(map[uint32]struct{}, 64)
	for i := uint32(0); i < 64; i++ {
		recids[i] = struct{}{}
	}

	proxy.InvalidateRows(recids)

	if proxy.compressed {
		t.Fatal("broad point invalidation did not fall back to full lazy invalidation")
	}
	if *calls >= len(recids) {
		t.Fatalf("adaptive probe recomputed %d rows, want fewer than %d", *calls, len(recids))
	}
}

func TestLookupInvalidationKeepsCheaperExactFanout(t *testing.T) {
	proxy, calls := newAdaptiveLookupProxyForTest(64)
	proxy.lastRecomputeNs.Store(1 << 62)
	recids := make(map[uint32]struct{}, 64)
	for i := uint32(0); i < 64; i++ {
		recids[i] = struct{}{}
	}

	proxy.InvalidateRows(recids)

	if !proxy.compressed {
		t.Fatal("cheap point invalidation unexpectedly discarded the compressed cache")
	}
	if *calls != len(recids) {
		t.Fatalf("exact invalidation recomputed %d rows, want %d", *calls, len(recids))
	}
}

func TestORCDependencyTriggersUseRelevantColumnsAndInvalidateSuffix(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-orc-trigger-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("torctrigger")

	CreateDatabase("torctrigger", false)
	base, _ := CreateTable("torctrigger", "base", Safe, false)
	src, _ := CreateTable("torctrigger", "src", Safe, false)

	base.CreateColumn("id", "INT", nil, nil)
	base.CreateColumn("sortk", "INT", nil, nil)
	base.CreateColumn("ref_id", "INT", nil, nil)
	base.CreateColumn("running", "INT", nil, nil)
	src.CreateColumn("ref_id", "INT", nil, nil)
	src.CreateColumn("val", "INT", nil, nil)
	src.CreateColumn("note", "TEXT", nil, nil)

	mapReduceFn := lambdaAst([]string{"acc", "$set", "ref_id"}, scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("begin"),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("define"),
			scm.NewSymbol("new_acc"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("+"), scm.NewSymbol("acc"), nestedScanAst("torctrigger", "src", "ref_id")}),
		}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("$set"), scm.NewSymbol("new_acc")}),
		scm.NewSymbol("new_acc"),
	}))
	for i, col := range base.Columns {
		if col.Name == "running" {
			base.Columns[i].OrcSortCols = []string{"sortk"}
			base.Columns[i].OrcSortDirs = []bool{false}
			base.Columns[i].OrcMapCols = []string{"ref_id"}
			base.Columns[i].OrcMapReduceFn = mapReduceFn
			base.Columns[i].OrcReduceInit = scm.NewInt(0)
			break
		}
	}
	refs := extractScanJoinInfo(mapReduceFn)
	base.registerORCTriggers("running")

	prefix := ".orcdep:base:running|scan0|src|"
	var triggerCount int
	for _, tr := range src.Triggers {
		if strings.HasPrefix(tr.Name, prefix) {
			triggerCount++
		}
	}
	if triggerCount != 3 {
		t.Fatalf("ORC dependency trigger count = %d, want 3 (refs=%#v mapreduce=%s)", triggerCount, refs, serializeScmerForTest(mapReduceFn))
	}

	tr, ok := findTriggerByPrefixAndTiming(src.Triggers, prefix, AfterUpdate)
	if !ok {
		t.Fatal("missing AfterUpdate ORC dependency trigger")
	}
	plan := triggerPlanStringForTest(tr)
	for _, want := range []string{`(get_assoc OLD "ref_id")`, `(get_assoc NEW "ref_id")`, `(get_assoc OLD "val")`, `(get_assoc NEW "val")`, `invalidateorc`, `"sortk"`} {
		if !strings.Contains(plan, want) {
			t.Fatalf("ORC trigger plan missing %s:\n%s", want, plan)
		}
	}
	if strings.Contains(plan, `"note"`) {
		t.Fatalf("ORC trigger plan should ignore unrelated note column:\n%s", plan)
	}
}

// Compression must consume one materialization: fresh list results have stable
// contents but distinct identities, so rerunning the callback breaks dictionaries.
func TestComputeProxyCompressionMaterializesOnce(t *testing.T) {
	proxy, calls := newAdaptiveLookupProxyForTest(3)
	proxy.main = nil
	proxy.compressed = false
	proxy.computor = scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		*calls++
		return scm.NewSlice([]scm.Scmer{args[0], scm.NewInt(42)})
	})
	proxy.Compress(nil)
	if *calls != 3 {
		t.Fatalf("computed %d times for three rows", *calls)
	}
	for i := uint32(0); i < 3; i++ {
		got := proxy.main.GetValue(i).Slice()
		if len(got) != 2 || got[0].Int() != int64(i) || got[1].Int() != 42 {
			t.Fatalf("row %d: %v", i, got)
		}
	}
	proxy.Compress(nil)
	if *calls != 3 {
		t.Fatalf("valid compressed values were recomputed: %d", *calls)
	}
}

func TestComputeProxyCompressionAllowsDependencyInvalidation(t *testing.T) {
	proxy, _ := newAdaptiveLookupProxyForTest(3)
	proxy.main = nil
	proxy.compressed = false
	invalidate := true
	proxy.computor = scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		if invalidate {
			invalidate = false
			proxy.InvalidateAll()
		}
		return args[0]
	})

	done := make(chan struct{})
	go func() {
		proxy.Compress(nil)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("computed-column dependency invalidation deadlocked compression")
	}
	if proxy.compressed {
		t.Fatal("compression published values from an invalidated revision")
	}

	proxy.Compress(nil)
	if !proxy.compressed {
		t.Fatal("stable retry did not publish compressed values")
	}
	for i := uint32(0); i < 3; i++ {
		if got := proxy.GetValue(i).Int(); got != int64(i) {
			t.Fatalf("row %d = %d, want %d", i, got, i)
		}
	}
}

func TestOrderedKeysetScanRetainsTargetAndProducerDependencies(t *testing.T) {
	source := nestedScanAst("keys", "grant", "ref_id")
	target := nestedScanAst("keys", "items", "ref_id")
	values := target.Slice()
	values[0] = scm.NewSymbol("scan_order_keys")
	values[2] = scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), values[2], source,
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString("ref_id")}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString("owner")})})
	refs := extractScanJoinInfo(target)
	if len(refs) != 2 || refs[0].table != "items" || refs[1].table != "grant" {
		t.Fatalf("ordered keyset lost target or producer dependency: %#v", refs)
	}
	found := false
	for _, col := range refs[0].condCols {
		found = found || col == "owner"
	}
	if !found {
		t.Fatalf("ordered keyset lost prefix dependency: %#v", refs[0])
	}
	if findScanNode(target, "keys", "items") == nil || !containsScan(target) {
		t.Fatal("ordered keyset was not recognized as a scan")
	}
}

// Literal historical frames exercise the reader independently of its writer.
func TestComputeProxyHistoricalV3RoundTrip(t *testing.T) {
	const recipe = `{"scmer_custom":10,"value":[1,[{"symbol":"lambda"},[{"symbol":"value"}],[{"scmer_custom":8,"value":[1,"*"]},{"var":0},2],1]]}`
	const payload = `{"scmer_bson_v1":[16,"KgAAAA=="]}`
	var frame bytes.Buffer
	for _, field := range []any{uint8(3), uint32(1), uint16(1), uint16(5)} {
		if err := binary.Write(&frame, binary.LittleEndian, field); err != nil {
			t.Fatal(err)
		}
	}
	frame.WriteString("value")
	binary.Write(&frame, binary.LittleEndian, uint64(len(recipe)))
	frame.WriteString(recipe)
	frame.Write([]byte{0, 0}) // uncompressed sparse delta
	binary.Write(&frame, binary.LittleEndian, uint32(1))
	binary.Write(&frame, binary.LittleEndian, uint32(0))
	binary.Write(&frame, binary.LittleEndian, uint64(len(payload)))
	frame.WriteString(payload)
	binary.Write(&frame, binary.LittleEndian, uint32(1))
	binary.Write(&frame, binary.LittleEndian, uint32(0))
	frame.WriteByte(0)
	original := append([]byte(nil), frame.Bytes()...)
	var proxy StorageComputeProxy
	if got := proxy.Deserialize(&frame); got != 1 || frame.Len() != 0 {
		t.Fatalf("extent=%d trailing=%d", got, frame.Len())
	}
	assert := func(p *StorageComputeProxy) {
		if !scm.Equal(scm.Apply(p.computor, scm.NewInt(9)), scm.NewInt(18)) {
			t.Fatal("native procedure did not restore")
		}
		kind, raw := p.delta[0].BSONRaw()
		if kind != 16 || !bytes.Equal(raw, []byte{42, 0, 0, 0}) {
			t.Fatal("native BSON identity was lost")
		}
	}
	assert(&proxy)
	cloned := cloneComputeProxyRows(&proxy, &storageShard{}, []uint32{0})
	if cloned.persistedVersion != 3 {
		t.Fatal("rebuild discarded historical native value encoding")
	}
	assert(cloned)
	var encoded bytes.Buffer
	proxy.Serialize(&encoded)
	if bytes := encoded.Bytes(); bytes[0] != 50 || bytes[1] != 3 {
		t.Fatal("historical v3 was downgraded on rewrite")
	}
	var restored StorageComputeProxy
	restored.Deserialize(bytes.NewReader(encoded.Bytes()[1:]))
	assert(&restored)
	// A shortened payload must fail before creating a partially initialized proxy.
	for _, end := range []int{1, 7, 17, len(original) - 1} {
		t.Run(fmt.Sprintf("truncated-%d", end), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("truncated v3 frame accepted")
				}
			}()
			new(StorageComputeProxy).Deserialize(bytes.NewReader(original[:end]))
		})
	}
}

func TestHistoricalExactScalarFrameAndNestedProxy(t *testing.T) {
	payloads := []string{`{"$memcp.scmer":"bson-v1","type":16,"payload":"KgAAAA"}`, `{"scmer_date_v1":[1700000000,123456789,17]}`}
	var frame bytes.Buffer
	frame.WriteByte(1) // magic 3 already consumed, historical version 1
	binary.Write(&frame, binary.LittleEndian, uint64(len(payloads)))
	for _, payload := range payloads {
		binary.Write(&frame, binary.LittleEndian, uint64(len(payload)))
		frame.WriteString(payload)
	}
	frame.WriteString("tail")
	var exact storageSCMERExact
	if count := exact.Deserialize(&frame); count != 2 || frame.String() != "tail" {
		t.Fatal("exact scalar reader consumed enclosing trailer")
	}
	if !exact.values[0].IsBSON() || !exact.values[1].IsDate() {
		t.Fatal("native scalar identities were not restored")
	}
	if exact.values[1].DateNanosecond() != 123456789 || exact.values[1].DateZone() != 17 {
		t.Fatal("historical date parts were changed")
	}
	var saved bytes.Buffer
	exact.Serialize(&saved)
	if saved.Bytes()[0] != 3 || saved.Bytes()[1] != 2 || saved.Bytes()[2] != 1 {
		t.Fatal("loaded exact scalar frame did not use current generic encoding")
	}
	var reread storageSCMERExact
	reread.Deserialize(bytes.NewReader(saved.Bytes()[1:]))
	if !reread.values[0].IsBSON() || !reread.values[1].IsDate() {
		t.Fatal("rewrite changed native scalar identities")
	}
	proxy := &StorageComputeProxy{main: &exact, compressed: true, count: 2, persistedVersion: 3}
	proxy.validMask.AtomicSet(0, true)
	proxy.validMask.AtomicSet(1, true)
	saved.Reset()
	proxy.Serialize(&saved)
	var restored StorageComputeProxy
	restored.Deserialize(bytes.NewReader(saved.Bytes()[1:]))
	if !restored.validMask.Get(0) || !restored.validMask.Get(1) || !restored.main.GetValue(0).IsBSON() || !restored.main.GetValue(1).IsDate() {
		t.Fatal("nested frame lost payload or outer validity")
	}
}

func TestNativeDateStorageRetainsIdentityOnRebuild(t *testing.T) {
	for _, value := range []scm.Scmer{scm.NewDate(1700000000), scm.NewDateWithZone(1700000000, 17), scm.NewDateNanoWithZone(1700000000, 123456789, 17)} {
		var scalar StorageSCMER
		scalar.prepare()
		scalar.scan(0, value)
		if scalar.proposeCompression(1) != nil {
			t.Fatal("fractional native date proposed lossy compression")
		}
		scalar.init(1)
		scalar.build(0, value)
		scalar.finish()
		var data bytes.Buffer
		scalar.Serialize(&data)
		if data.Bytes()[0] != 3 {
			t.Fatal("fractional native date used legacy JSON storage")
		}
		var restored storageSCMERExact
		restored.Deserialize(bytes.NewReader(data.Bytes()[1:]))
		if got := restored.GetValue(0); !got.IsDate() || got.Int() != value.Int() || got.DateNanosecond() != value.DateNanosecond() || got.DateZone() != value.DateZone() {
			t.Fatal("fractional storage changed date identity", got)
		}
		proxy := &StorageComputeProxy{count: 1, delta: map[uint32]scm.Scmer{0: value}, computor: scm.Globalenv.Vars[scm.Symbol("list")]}
		proxy.validMask.AtomicSet(0, true)
		data.Reset()
		proxy.Serialize(&data)
		if data.Bytes()[1] != 3 {
			t.Fatal("fractional computed result used lossy v2 encoding")
		}
		var reloaded StorageComputeProxy
		reloaded.Deserialize(bytes.NewReader(data.Bytes()[1:]))
		if got := reloaded.GetValue(0); !got.IsDate() || got.Int() != value.Int() || got.DateNanosecond() != value.DateNanosecond() || got.DateZone() != value.DateZone() {
			t.Fatal("computed date lost fractional precision", got)
		}
	}
}

func TestCompactNativeDateFrame(t *testing.T) {
	// Literal magic-10/version-0 components, independent of the new writer.
	literal := func(nanos int64, nanoNull bool) []byte {
		var data bytes.Buffer
		data.Write([]byte{3, 2, 2})
		binary.Write(&data, binary.LittleEndian, uint64(1))
		for part, offset := range []int64{1700000000, nanos, 17} {
			nullFlag := byte(0)
			if part == 1 && nanoNull {
				nullFlag = 1
			}
			data.Write([]byte{10, 1, nullFlag, 0})
			binary.Write(&data, binary.LittleEndian, uint32(0))
			for _, value := range []uint64{2, 1, uint64(offset), 0, 0, 0} {
				binary.Write(&data, binary.LittleEndian, value)
			}
		}
		return data.Bytes()
	}
	frame := literal(123456789, false)
	reader := bytes.NewBuffer(append(append([]byte(nil), frame[1:]...), []byte("tail")...))
	var restored storageSCMERExact
	if count := restored.Deserialize(reader); count != 1 || reader.String() != "tail" {
		t.Fatal("native date frame crossed its boundary")
	}
	if got := restored.GetValue(0); !got.IsDate() || got.Int() != 1700000000 || got.DateNanosecond() != 123456789 || got.DateZone() != 17 {
		t.Fatal("literal compact date changed", got)
	}
	values := []scm.Scmer{scm.NewDateNanoWithZone(-1, 999999999, 2047), scm.NewNil(), scm.NewDateNanoWithZone(-(1 << 44), 0, 0)}
	var saved bytes.Buffer
	(&storageSCMERExact{StorageSCMER: StorageSCMER{values: values}}).Serialize(&saved)
	if saved.Bytes()[0] != 3 || saved.Bytes()[1] != 2 || saved.Bytes()[2] != 2 {
		t.Fatal("homogeneous dates did not select compact version 2")
	}
	var loaded storageSCMERExact
	loaded.Deserialize(bytes.NewReader(saved.Bytes()[1:]))
	for i, want := range values {
		got, err := scm.MarshalPersistentJSON(loaded.GetValue(uint32(i)))
		if err != nil {
			t.Fatal(err)
		}
		expected, err := scm.MarshalPersistentJSON(want)
		if err != nil || string(got) != string(expected) {
			t.Fatalf("compact date parts changed: %s -> %s", expected, got)
		}
	}
	for name, malformed := range map[string][]byte{"truncated": frame[:len(frame)-1], "out-of-range": literal(1000000000, false), "null-mask": literal(0, true)} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid compact date was accepted")
				}
			}()
			var invalid storageSCMERExact
			invalid.Deserialize(bytes.NewReader(malformed[1:]))
		})
	}
	mixed := &storageSCMERExact{StorageSCMER: StorageSCMER{values: []scm.Scmer{values[0], scm.NewInt(7)}}}
	saved.Reset()
	mixed.Serialize(&saved)
	if saved.Bytes()[1] != 2 || saved.Bytes()[2] != 1 {
		t.Fatal("mixed values lost their permanent JSON payload")
	}
}

func TestCompactNativeDateReaders(t *testing.T) {
	values := []scm.Scmer{scm.NewDateNanoWithZone(1700000000, 1, 17), scm.NewNil(), scm.NewDateNanoWithZone(-1, 999999999, 2047), scm.NewDate(1700000001)}
	var encoded bytes.Buffer
	(&storageSCMERExact{StorageSCMER: StorageSCMER{values: values}}).Serialize(&encoded)
	var column storageSCMERExact
	column.Deserialize(bytes.NewReader(encoded.Bytes()[1:]))
	column.finish()
	reader := column.GetCachedReader(ReadCompressed)
	defer ReleaseColumnReader(reader)
	if scm.JITEnabled() && (compiledColumnGetValue(reader) == nil || compiledColumnGetValueRange(reader) == nil || compiledColumnGetValueMulti(reader) == nil) {
		t.Fatal("packed date did not compile all three native read modes")
	}
	check := func(got, want scm.Scmer) {
		t.Helper()
		a, err := scm.MarshalPersistentJSON(got)
		if err != nil {
			t.Fatal(err)
		}
		b, err := scm.MarshalPersistentJSON(want)
		if err != nil || !bytes.Equal(a, b) {
			t.Fatalf("date reader changed parts: %s -> %s", b, a)
		}
	}
	for i, want := range values {
		value := reader.GetValue(uint32(i))
		runtime.GC() // Correctness-only: returned fractional pointers cross the native ABI.
		check(value, want)
	}
	strided := make([]scm.Scmer, 12)
	reader.GetValueRange(1, 3, strided, 3)
	for i, want := range values[1:] {
		check(strided[i*3], want)
	}
	reader.GetValueMulti([]uint32{3, 0, 2, 1}, strided, 2)
	for i, index := range []int{3, 0, 2, 1} {
		check(strided[i*2], values[index])
	}
	if column.JITValueType() != scm.JITTypeUnknown {
		t.Fatal("nullable date advertised a uniform physical type")
	}
	var rewritten bytes.Buffer
	column.Serialize(&rewritten)
	var reopened storageSCMERExact
	reopened.Deserialize(bytes.NewReader(rewritten.Bytes()[1:]))
	for i, want := range values {
		check(reopened.GetValue(uint32(i)), want)
	}
}

func TestCompactNativeDateRetainsCompressedFootprint(t *testing.T) {
	const count = 60000
	values := make([]scm.Scmer, count)
	for i := range values {
		values[i] = scm.NewDate(1700000000 + int64(i)*3600)
	}
	var encoded bytes.Buffer
	(&storageSCMERExact{StorageSCMER: StorageSCMER{values: values}}).Serialize(&encoded)
	var column storageSCMERExact
	if rows := column.Deserialize(bytes.NewReader(encoded.Bytes()[1:])); rows != count {
		t.Fatalf("date rows = %d", rows)
	}
	if size := column.ComputeSize(); size >= count*16 {
		t.Fatalf("compressed date column materialized its row bag: %d bytes", size)
	}
	if column.JITValueType() != scm.TagDate {
		t.Fatal("non-null packed date lost its physical type")
	}
	for _, index := range []uint32{0, 12345, count - 1} {
		if got := column.GetValue(index); !got.IsDate() || got.Int() != values[index].Int() || got.DateNanosecond() != 0 {
			t.Fatalf("compressed date row %d changed", index)
		}
	}
}
