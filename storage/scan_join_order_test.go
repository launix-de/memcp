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

import "testing"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"

func setupScanJoinOrderTable(t *testing.T, database string, name string, columns []string, rows [][]scm.Scmer) *table {
	t.Helper()
	table, _ := CreateTable(database, name, Memory, true)
	for _, column := range columns {
		table.CreateColumn(column, "INT", nil, nil)
	}
	table.Insert(columns, rows, nil, scm.NewNil(), false, nil)
	return table
}

func TestScanJoinOrderFiltersJoinsAndBrakesInDriverOrder(t *testing.T) {
	database := "tscanjoinorder"
	databases.Remove(database)
	t.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)

	orders := setupScanJoinOrderTable(t, database, "orders", []string{"id", "customer_id"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(10)},
		{scm.NewInt(2), scm.NewInt(20)},
		{scm.NewInt(3), scm.NewInt(10)},
		{scm.NewInt(4), scm.NewInt(30)},
		{scm.NewInt(5), scm.NewInt(10)},
	})
	customers := setupScanJoinOrderTable(t, database, "customers", []string{"id", "region"}, [][]scm.Scmer{
		{scm.NewInt(10), scm.NewInt(1)},
		{scm.NewInt(20), scm.NewInt(2)},
		{scm.NewInt(30), scm.NewInt(1)},
	})
	tags := setupScanJoinOrderTable(t, database, "tags", []string{"customer_id", "tag", "enabled"}, [][]scm.Scmer{
		{scm.NewInt(10), scm.NewInt(100), scm.NewInt(1)},
		{scm.NewInt(10), scm.NewInt(101), scm.NewInt(1)},
		{scm.NewInt(20), scm.NewInt(200), scm.NewInt(1)},
		{scm.NewInt(30), scm.NewInt(300), scm.NewInt(0)},
	})

	_, descending := integerOrder(true)
	equalOne := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewBool(values[0].Int() == 1)
	})
	got := make([][2]int64, 0)
	mapReduceFn := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		got = append(got, [2]int64{values[1].Int(), values[2].Int()})
		return values[0]
	})

	scanJoinOrder(nil, scanJoinOrderSpec{
		inputs: []scanJoinOrderInput{
			{table: orders},
			{table: customers, filterCols: []string{"region"}, filter: equalOne,
				sourceKeyCols: []scanJoinOrderColumn{{table: 0, column: "customer_id"}}, targetKeyCols: []string{"id"}},
			{table: tags, filterCols: []string{"enabled"}, filter: equalOne,
				sourceKeyCols: []scanJoinOrderColumn{{table: 1, column: "id"}}, targetKeyCols: []string{"customer_id"}},
		},
		orderCols:     []scanJoinOrderColumn{{table: 0, column: "id"}},
		orderDirs:     []func(...scm.Scmer) scm.Scmer{descending},
		offset:        1,
		limit:         3,
		mapCols:       []scanJoinOrderColumn{{table: 0, column: "id"}, {table: 2, column: "tag"}},
		mapReduceFn:   mapReduceFn,
		neutral:       scm.NewNil(),
		notFoundValue: scm.NewNil(),
		batchedProbe:  true,
	})

	want := [][2]int64{{5, 101}, {3, 100}, {3, 101}}
	if len(got) != len(want) {
		t.Fatalf("joined rows = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("joined rows = %v, want %v", got, want)
		}
	}
}

func TestScanJoinOrderDoesNotJoinNullKeys(t *testing.T) {
	database := "tscanjoinordernull"
	databases.Remove(database)
	t.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	left := setupScanJoinOrderTable(t, database, "left_rows", []string{"id", "key_col"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewNil()},
	})
	right := setupScanJoinOrderTable(t, database, "right_rows", []string{"key_col"}, [][]scm.Scmer{
		{scm.NewNil()},
	})
	_, ascending := integerOrder(false)
	result := scanJoinOrder(nil, scanJoinOrderSpec{
		inputs: []scanJoinOrderInput{
			{table: left},
			{table: right, sourceKeyCols: []scanJoinOrderColumn{{table: 0, column: "key_col"}}, targetKeyCols: []string{"key_col"}},
		},
		orderCols:     []scanJoinOrderColumn{{table: 0, column: "id"}},
		orderDirs:     []func(...scm.Scmer) scm.Scmer{ascending},
		limit:         1,
		mapCols:       []scanJoinOrderColumn{{table: 0, column: "id"}},
		mapReduceFn:   scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[1] }),
		neutral:       scm.NewNil(),
		notFoundValue: scm.NewString("missing"),
	})
	if result.String() != "missing" {
		t.Fatalf("NULL equi-join result = %v, want missing", result)
	}
}

func TestScanJoinOrderDeclarationSupportsOrderFromJoinedTable(t *testing.T) {
	Init(scm.Globalenv)
	database := "tscanjoinorderdeclare"
	databases.Remove(database)
	t.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	left := setupScanJoinOrderTable(t, database, "left_rows", []string{"id", "key_col"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(10)},
		{scm.NewInt(2), scm.NewInt(20)},
	})
	right := setupScanJoinOrderTable(t, database, "right_rows", []string{"key_col", "rank_col"}, [][]scm.Scmer{
		{scm.NewInt(10), scm.NewInt(200)},
		{scm.NewInt(20), scm.NewInt(100)},
	})
	ascendingValue, _ := integerOrder(false)
	got := make([]int64, 0, 1)
	mapReduceFn := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		got = append(got, values[1].Int())
		return values[0]
	})
	trueFn := scanJoinTrue()
	ref := func(tableIndex int64, column string) scm.Scmer {
		return scm.NewSlice([]scm.Scmer{scm.NewInt(tableIndex), scm.NewString(column)})
	}

	scm.Apply(scm.Globalenv.Vars[scm.Symbol("scan_join_order")],
		scm.NewNil(),
		scm.NewSlice([]scm.Scmer{NewTableScmer(left), NewTableScmer(right)}),
		scm.NewSlice([]scm.Scmer{scm.NewSlice(nil), scm.NewSlice(nil)}),
		scm.NewSlice([]scm.Scmer{trueFn, trueFn}),
		scm.NewSlice([]scm.Scmer{scm.NewSlice([]scm.Scmer{
			scm.NewSlice([]scm.Scmer{scm.NewInt(0), scm.NewString("key_col"), scm.NewString("key_col")}),
		})}),
		scm.NewSlice(nil), scm.NewNil(),
		scm.NewSlice([]scm.Scmer{ref(1, "rank_col")}),
		scm.NewSlice([]scm.Scmer{ascendingValue}),
		scm.NewInt(0), scm.NewInt(0), scm.NewInt(1),
		scm.NewSlice([]scm.Scmer{ref(0, "id")}), mapReduceFn, scm.NewNil())

	if len(got) != 1 || got[0] != 2 {
		t.Fatalf("declared joined-table order returned %v, want [2]", got)
	}
}

func repartitionScanJoinOrderTable(t *testing.T, table *table, dimensions []shardDimension) {
	t.Helper()
	RebuildTable(table, true, false)
	if !table.beginManualRepartition() {
		t.Fatal("manual repartition was not claimed")
	}
	table.repartition(dimensions)
}

func TestScanJoinOrderPrunesCompatibleJoinKeyShardPairs(t *testing.T) {
	oldAnalyzeMinItems := Settings.AnalyzeMinItems
	Settings.AnalyzeMinItems = 1 << 30
	t.Cleanup(func() { Settings.AnalyzeMinItems = oldAnalyzeMinItems })
	database := "tscanjoinorderpartition"
	databases.Remove(database)
	t.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	leftRows := make([][]scm.Scmer, 120)
	rightRows := make([][]scm.Scmer, 120)
	for i := range leftRows {
		leftRows[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewInt(int64(i))}
		rightRows[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewInt(int64(1000 - i))}
	}
	left := setupScanJoinOrderTable(t, database, "left_rows", []string{"id", "key_col"}, leftRows)
	right := setupScanJoinOrderTable(t, database, "right_rows", []string{"key_col", "rank_col"}, rightRows)
	dimension := shardDimension{Column: "key_col", NumPartitions: 4, Pivots: []scm.Scmer{
		scm.NewInt(29), scm.NewInt(59), scm.NewInt(89),
	}}
	repartitionScanJoinOrderTable(t, left, []shardDimension{dimension})
	repartitionScanJoinOrderTable(t, right, []shardDimension{{
		Column: "key_col", NumPartitions: dimension.NumPartitions, Pivots: append([]scm.Scmer(nil), dimension.Pivots...),
	}})

	_, descending := integerOrder(true)
	spec := scanJoinOrderSpec{
		inputs: []scanJoinOrderInput{
			{table: left},
			{table: right, sourceKeyCols: []scanJoinOrderColumn{{table: 0, column: "key_col"}}, targetKeyCols: []string{"key_col"}},
		},
		orderCols:   []scanJoinOrderColumn{{table: 0, column: "id"}},
		orderDirs:   []func(...scm.Scmer) scm.Scmer{descending},
		limit:       5,
		mapCols:     []scanJoinOrderColumn{{table: 0, column: "id"}},
		mapReduceFn: scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[1] }),
	}
	prepareScanJoinOrderSpec(&spec)
	streams := make([][]*scanJoinOrderShardStream, 2)
	streams[0] = collectScanJoinOrderShardStreams(nil, &spec.inputs[0], []scm.Scmer{scm.NewString("id")}, []func(...scm.Scmer) scm.Scmer{descending})
	_, ascending := integerOrder(false)
	streams[1] = collectScanJoinOrderShardStreams(nil, &spec.inputs[1], []scm.Scmer{scm.NewString("key_col")}, []func(...scm.Scmer) scm.Scmer{ascending})
	allPairs := len(streams[0]) * len(streams[1])
	combinations := scanJoinOrderShardCombinations(&spec, streams)
	if len(combinations) >= allPairs {
		t.Fatalf("compatible join partitioning retained %d of %d shard pairs", len(combinations), allPairs)
	}
	if len(combinations) != 4 {
		t.Fatalf("compatible join partitioning produced %d runners, want 4", len(combinations))
	}

	got := make([]int64, 0, 5)
	spec.mapReduceFn = scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		got = append(got, values[1].Int())
		return values[0]
	})
	scanJoinOrder(nil, spec)
	want := []int64{119, 118, 117, 116, 115}
	if !equalInt64s(got, want) {
		t.Fatalf("partitioned global Top-K = %v, want %v", got, want)
	}
}

func TestScanJoinOrderRejectsCombineWithLimit(t *testing.T) {
	database := "tscanjoinorderreduce2limit"
	databases.Remove(database)
	t.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	table := setupScanJoinOrderTable(t, database, "rows", []string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}})
	_, ascending := integerOrder(false)
	defer func() {
		if recover() == nil {
			t.Fatal("reduce2 with LIMIT did not panic")
		}
	}()
	scanJoinOrder(nil, scanJoinOrderSpec{
		inputs:      []scanJoinOrderInput{{table: table}},
		orderCols:   []scanJoinOrderColumn{{table: 0, column: "id"}},
		orderDirs:   []func(...scm.Scmer) scm.Scmer{ascending},
		limit:       1,
		mapCols:     []scanJoinOrderColumn{{table: 0, column: "id"}},
		mapReduceFn: scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[1] }),
		combineFn:   scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[1] }),
	})
}

func TestScanJoinOrderCombineCombinesUnlimitedRunnerPartials(t *testing.T) {
	database := "tscanjoinorderreduce2"
	databases.Remove(database)
	t.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	left := setupScanJoinOrderTable(t, database, "left_rows", []string{"key_col", "amount"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(10)},
		{scm.NewInt(2), scm.NewInt(20)},
		{scm.NewInt(3), scm.NewInt(30)},
		{scm.NewInt(4), scm.NewInt(40)},
	})
	right := setupScanJoinOrderTable(t, database, "right_rows", []string{"key_col"}, [][]scm.Scmer{
		{scm.NewInt(1)}, {scm.NewInt(2)}, {scm.NewInt(3)}, {scm.NewInt(4)},
	})
	dimension := shardDimension{Column: "key_col", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(2)}}
	repartitionScanJoinOrderTable(t, left, []shardDimension{dimension})
	repartitionScanJoinOrderTable(t, right, []shardDimension{{
		Column: "key_col", NumPartitions: dimension.NumPartitions, Pivots: append([]scm.Scmer(nil), dimension.Pivots...),
	}})
	var localReduceCalls atomic.Int64
	var globalReduceCalls atomic.Int64
	result := scanJoinOrder(nil, scanJoinOrderSpec{
		inputs: []scanJoinOrderInput{
			{table: left},
			{table: right, sourceKeyCols: []scanJoinOrderColumn{{table: 0, column: "key_col"}}, targetKeyCols: []string{"key_col"}},
		},
		limit:   -1,
		mapCols: []scanJoinOrderColumn{{table: 0, column: "amount"}},
		mapReduceFn: scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			localReduceCalls.Add(1)
			return scm.NewInt(values[0].Int() + values[1].Int())
		}),
		combineFn: scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			globalReduceCalls.Add(1)
			return scm.NewInt(values[0].Int() + values[1].Int())
		}),
		neutral: scm.NewInt(0),
	})
	if result.Int() != 100 {
		t.Fatalf("reduce2 joined sum = %v, want 100", result)
	}
	if localReduceCalls.Load() != 4 {
		t.Fatalf("runner-local reduce calls = %d, want 4", localReduceCalls.Load())
	}
	if globalReduceCalls.Load() != 2 {
		t.Fatalf("global reduce2 calls = %d, want 2", globalReduceCalls.Load())
	}
}

func TestScanJoinOrderUsesSerialCountPipeline(t *testing.T) {
	database := "tscanjoinorderserialcount"
	databases.Remove(database)
	t.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	left := setupScanJoinOrderTable(t, database, "left_rows", []string{"key_col"}, [][]scm.Scmer{
		{scm.NewInt(1)}, {scm.NewInt(2)}, {scm.NewInt(3)},
	})
	right := setupScanJoinOrderTable(t, database, "right_rows", []string{"key_col"}, [][]scm.Scmer{
		{scm.NewInt(1)}, {scm.NewInt(2)}, {scm.NewInt(3)},
	})
	mapReduceFn := scm.Eval(scm.Optimize(scm.Read("scan_join_order count mapreducer", "(lambda (acc value) (+ acc 1))"), &scm.Globalenv, nil), &scm.Globalenv)
	result := scanJoinOrder(nil, scanJoinOrderSpec{
		inputs: []scanJoinOrderInput{
			{table: left},
			{table: right, sourceKeyCols: []scanJoinOrderColumn{{table: 0, column: "key_col"}}, targetKeyCols: []string{"key_col"}},
		},
		limit:       -1,
		mapCols:     []scanJoinOrderColumn{{table: 0, column: "key_col"}},
		mapReduceFn: mapReduceFn,
		neutral:     scm.NewInt(0),
	})
	if result.Int() != 3 {
		t.Fatalf("serial COUNT pipeline result = %v, want 3", result)
	}
}
