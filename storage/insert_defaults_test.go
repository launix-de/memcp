/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "testing"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

func TestInsertDefaultsShareContextWithoutChangingCaller(t *testing.T) {
	calls := 0
	context := scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewInt(73)})
	calculator := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		calls++
		if len(args) != 1 || !scm.Equal(args[0], context) {
			t.Fatal("calculator lost invocation context")
		}
		return scm.NewInt(int64(calls))
	})
	declared := &column{Name: "created", Default: scm.NewNil(), DefaultPresent: true, DefaultCalculator: &calculator, AllowNull: true}
	tbl := &table{Columns: []*column{declared}}
	columns := []string{"id"}
	values := [][]scm.Scmer{{scm.NewInt(1)}, {scm.NewInt(2)}}
	newColumns, rows := tbl.assignInsertDefaults(columns, values, context)
	if len(newColumns) != 2 || newColumns[1] != "created" || len(columns) != 1 || len(values[0]) != 1 {
		t.Fatal("default changed caller-owned shape")
	}
	if calls != 2 || rows[0][1].Int() != 1 || rows[1][1].Int() != 2 {
		t.Fatal("calculator not evaluated independently per omitted row")
	}
	rows[0][0] = scm.NewInt(99)
	if values[0][0].Int() != 1 {
		t.Fatal("materialized row aliases caller input")
	}
	_, explicit := tbl.assignInsertDefaults([]string{"id", "created"}, [][]scm.Scmer{{scm.NewInt(3), scm.NewNil()}}, context)
	if calls != 2 || !explicit[0][1].IsNil() {
		t.Fatal("explicit NULL replaced by default")
	}
}

func TestDefaultRecipeSurvivesSerialization(t *testing.T) {
	recipe := scm.EvalAll("default recipe", `(lambda (context) (if (nil? context) 11 context))`, &scm.Globalenv)
	recipe = scm.CloseProcedure(recipe)
	original := &column{Name: "value", Default: scm.NewNil(), DefaultPresent: true, DefaultCalculator: &recipe}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored column
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !restored.hasDefault() || restored.defaultValue(scm.NewInt(17)).Int() != 17 {
		t.Fatal("closed recipe did not restore")
	}
	literal := &column{Default: scm.NewNil(), DefaultPresent: true}
	if !literal.hasDefault() || !literal.defaultValue(scm.NewNil()).IsNil() {
		t.Fatal("declared NULL default lost presence")
	}
}

func TestGroupedColumnOptionsRejectPartialChanges(t *testing.T) {
	c := &column{Default: scm.NewInt(3), DefaultPresent: true, Metadata: scm.NewString("old")}
	requirePanic(t, func() {
		c.Alter("options", scm.NewSlice([]scm.Scmer{scm.NewString("metadata"), scm.NewString("new"), scm.NewString("default"), scm.NewInt(7), scm.NewString("unknown"), scm.NewNil()}))
	})
	if c.Default.Int() != 3 || c.Metadata.String() != "old" {
		t.Fatal("failed options changed declaration")
	}
	c.Alter("options", scm.NewSlice([]scm.Scmer{scm.NewString("metadata"), scm.NewString("new"), scm.NewString("drop_default"), scm.NewBool(true)}))
	if c.hasDefault() || c.Metadata.String() != "new" {
		t.Fatal("grouped default and metadata not published")
	}
}

func TestDefaultRecipeBackfillKeepsFutureInvocationValues(t *testing.T) {
	old := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = old }()
	Init(scm.Globalenv)
	CreateDatabase("default_backfill_context", false)
	defer databases.Remove("default_backfill_context")
	tbl, _ := CreateTable("default_backfill_context", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}, {scm.NewInt(2)}}, nil, scm.NewNil(), false, nil)
	recipe := scm.EvalAll("default recipe", `(lambda (context) context)`, &scm.Globalenv)
	attrs := []scm.Scmer{scm.NewString("default_calculator"), recipe, scm.NewString("initial_value"), scm.NewInt(99)}
	if !tbl.CreateColumn("value", "ANY", nil, attrs) {
		t.Fatal("ADD with explicit backfill failed")
	}
	context := scm.NewInt(101)
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(3)}}, nil, scm.NewNil(), false, nil, InsertOptions{CalculatorContext: &context})
	tbl.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(4), scm.NewNil()}}, nil, scm.NewNil(), false, nil, InsertOptions{CalculatorContext: &context})
	check := func(current *table, withFifth bool) {
		t.Helper()
		seen := make(map[int64]scm.Scmer)
		for _, shard := range current.ActiveShards() {
			func() {
				release := shard.GetRead(nil)
				defer release()
				id := shard.ColumnReaderTx(nil, "id", false)
				value := shard.ColumnReaderTx(nil, "value", false)
				shard.mu.RLock()
				defer shard.mu.RUnlock()
				for row := uint32(0); row < shard.main_count+uint32(len(shard.inserts)); row++ {
					if !shard.deletions.Get(uint(row)) {
						seen[id(row).Int()] = value(row)
					}
				}
			}()
		}
		count := 4
		if withFifth {
			count++
			if seen[5].Int() != 202 {
				t.Fatal("restored INSERT lost invocation binding", seen[5])
			}
		}
		if len(seen) != count || seen[1].Int() != 99 || seen[2].Int() != 99 || seen[3].Int() != 101 || !seen[4].IsNil() {
			t.Fatal("backfill froze future defaults or changed explicit NULL", seen)
		}
	}
	check(tbl, false)
	RebuildTable(tbl, true, false)
	check(tbl, false)
	restored := reloadTableFromPersistence(t, tbl.schema.Name, tbl.schema.persistence)
	check(restored, false)
	context = scm.NewInt(202)
	restored.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(5)}}, nil, scm.NewNil(), false, nil, InsertOptions{CalculatorContext: &context})
	check(restored, true)
}

func TestDefaultRecipeBackfillNeedsExplicitInitialValue(t *testing.T) {
	old := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = old }()
	Init(scm.Globalenv)
	CreateDatabase("default_backfill_validation", false)
	defer databases.Remove("default_backfill_validation")
	tbl, _ := CreateTable("default_backfill_validation", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
	recipe := scm.EvalAll("default recipe", `(lambda (context) context)`, &scm.Globalenv)
	requirePanic(t, func() {
		tbl.CreateColumn("value", "ANY", nil, []scm.Scmer{scm.NewString("default_calculator"), recipe})
	})
	if len(tbl.Columns) != 1 || tbl.Count() != 1 {
		t.Fatal("rejected calculator backfill published a column or changed rows")
	}
}

func TestAddedColumnBackfillSurvivesNameReuseAndDefaultChanges(t *testing.T) {
	old := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = old }()
	Init(scm.Globalenv)
	CreateDatabase("default_backfill_name_reuse", false)
	defer databases.Remove("default_backfill_name_reuse")
	tbl, _ := CreateTable("default_backfill_name_reuse", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("value", "INT", nil, nil)
	tbl.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewInt(41)}}, nil, scm.NewNil(), false, nil)
	if !tbl.DropColumn("value") || !tbl.CreateColumn("value", "INT", nil, []scm.Scmer{scm.NewString("default"), scm.NewInt(9)}) {
		t.Fatal("DROP/ADD of the same column name failed")
	}
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(2)}}, nil, scm.NewNil(), false, nil)
	scm.Apply(scm.Globalenv.Vars[scm.Symbol("altercolumn")], NewTableScmer(tbl), scm.NewString("value"), scm.NewString("default"), scm.NewInt(3))
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(3)}}, nil, scm.NewNil(), false, nil)
	tbl.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(4), scm.NewNil()}}, nil, scm.NewNil(), false, nil)
	check := func(current *table) {
		t.Helper()
		seen := make(map[int64]scm.Scmer)
		for _, shard := range current.ActiveShards() {
			func() {
				release := shard.GetRead(nil)
				defer release()
				id := shard.ColumnReaderTx(nil, "id", false)
				value := shard.ColumnReaderTx(nil, "value", false)
				shard.mu.RLock()
				defer shard.mu.RUnlock()
				for row := uint32(0); row < shard.main_count+uint32(len(shard.inserts)); row++ {
					if !shard.deletions.Get(uint(row)) {
						seen[id(row).Int()] = value(row)
					}
				}
			}()
		}
		if len(seen) != 4 || seen[1].Int() != 9 || seen[2].Int() != 9 || seen[3].Int() != 3 || !seen[4].IsNil() {
			t.Fatal("backfill, later default, or explicit NULL changed", seen)
		}
	}
	check(tbl)
	// Keep the old INSERT records in the WAL: a rebuilt fixture would hide
	// whether recovery wrongly reuses the removed column's value of 41.
	check(reloadTableFromPersistence(t, tbl.schema.Name, tbl.schema.persistence))
}
