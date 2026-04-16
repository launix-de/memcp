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

func nestedScanAst(schema, table, outerParam string) scm.Scmer {
	return scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("scan"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("session"), scm.NewString("__memcp_tx")}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("table"), scm.NewString(schema), scm.NewString(table)}),
		listAst(scm.NewString("ref_id")),
		lambdaAst([]string{"src.ref_id"}, scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("equal?"),
			scm.NewSymbol("src.ref_id"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("outer"), scm.NewSymbol(outerParam)}),
		})),
		listAst(scm.NewString("val")),
		lambdaAst([]string{"val"}, scm.NewSymbol("val")),
		scm.NewSymbol("+"),
		scm.NewInt(0),
	})
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
