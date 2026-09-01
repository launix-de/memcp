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
	"os"
	"strings"
	"testing"

	"github.com/launix-de/memcp/scm"
)

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
		scm.NewSlice(nil),
		scm.NewInt(1),
	})
	if constantOne.Slice()[0].IsSymbol() {
		t.Fatal("optimizer did not lower lambda head for the regression fixture")
	}
	if !isConstantOneAggregate(constantOne) {
		t.Fatal("constant COUNT mapper was hidden by its lowered lambda head")
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
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("table"), scm.NewString(schema), scm.NewString(table)}),
		listAst(scm.NewString("ref_id")),
		lambdaAst([]string{"src.ref_id"}, scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("equal?"),
			scm.NewSymbol("src.ref_id"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("outer"), scm.NewInt(1), scm.NewSymbol(outerParam)}),
		})),
		listAst(scm.NewString("val")),
		lambdaAst([]string{"val"}, scm.NewSymbol("val")),
		scm.NewSymbol("+"),
		scm.NewInt(0),
	})
}

func TestExtractScanJoinInfoIncludesDynamicTablePlan(t *testing.T) {
	dynamicTable := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("begin"),
		nestedScanAst("dynamic_dependency", "acl", "driver_id"),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("table"),
			scm.NewString("dynamic_dependency"),
			scm.NewString("driver"),
		}),
	})
	computor := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("scan"),
		scm.NewNil(),
		dynamicTable,
		listAst(),
		lambdaAst(nil, scm.NewBool(true)),
		listAst(),
		lambdaAst(nil, scm.NewInt(1)),
		scm.NewSymbol("+"),
		scm.NewInt(0),
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
		(scan nil (table "tlookuptrigger" "src")
			'("ref_id") (lambda (source_ref_id) (equal? source_ref_id (outer 1 ref_id)))
			'("val") (lambda (val) val)
			(lambda (old value) value) 0 nil false))`
	rawComputor := scm.Eval(scm.Read("raw lookup trigger test", computorSource), &scm.Globalenv)
	compiledComputor := scm.Eval(scm.Optimize(scm.Read("compiled lookup trigger test", computorSource), &scm.Globalenv, nil), &scm.Globalenv)
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

	mapFn := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		return scm.Apply(args[0])
	})
	reduceFn := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		return args[1]
	})
	mr := shard.OpenMapReducer([]string{"$invalidate:cached"}, mapFn, reduceFn, false, 0, nil, nil)
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

	mapFn := lambdaAst([]string{"$set", "ref_id"}, scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("list"),
		scm.NewSymbol("$set"),
		nestedScanAst("torctrigger", "src", "ref_id"),
	}))
	reduceFn := lambdaAst([]string{"acc", "mapped"}, scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("begin"),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("define"),
			scm.NewSymbol("new_acc"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("+"), scm.NewSymbol("acc"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("cadr"), scm.NewSymbol("mapped")})}),
		}),
		scm.NewSlice([]scm.Scmer{scm.NewSlice([]scm.Scmer{scm.NewSymbol("car"), scm.NewSymbol("mapped")}), scm.NewSymbol("new_acc")}),
		scm.NewSymbol("new_acc"),
	}))
	for i, col := range base.Columns {
		if col.Name == "running" {
			base.Columns[i].OrcSortCols = []string{"sortk"}
			base.Columns[i].OrcSortDirs = []bool{false}
			base.Columns[i].OrcMapCols = []string{"ref_id"}
			base.Columns[i].OrcMapFn = mapFn
			base.Columns[i].OrcReduceFn = reduceFn
			base.Columns[i].OrcReduceInit = scm.NewInt(0)
			break
		}
	}
	refs := append(extractScanJoinInfo(mapFn), extractScanJoinInfo(reduceFn)...)
	base.registerORCTriggers("running")

	prefix := ".orcdep:base:running|scan0|src|"
	var triggerCount int
	for _, tr := range src.Triggers {
		if strings.HasPrefix(tr.Name, prefix) {
			triggerCount++
		}
	}
	if triggerCount != 3 {
		t.Fatalf("ORC dependency trigger count = %d, want 3 (refs=%#v map=%s)", triggerCount, refs, serializeScmerForTest(mapFn))
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
