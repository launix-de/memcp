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
import "github.com/launix-de/memcp/scm"

func setupBatchAcceptTable(t *testing.T, database string, rows int) *table {
	t.Helper()
	table := setupScanParallelTestTable(t, database)
	table.CreateColumn("grp", "INT", nil, nil)
	values := make([][]scm.Scmer, rows)
	for i := range values {
		values[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewInt(int64(i % 3))}
	}
	table.Insert([]string{"id", "grp"}, values, nil, scm.NewNil(), false, nil)
	return table
}

func integerOrder(descending bool) (scm.Scmer, func(...scm.Scmer) scm.Scmer) {
	relation := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		if descending {
			return scm.NewBool(scm.ToInt(values[0]) > scm.ToInt(values[1]))
		}
		return scm.NewBool(scm.ToInt(values[0]) < scm.ToInt(values[1]))
	})
	return relation, scm.OptimizeProcToSerialFunction(relation)
}

func recSetModuloFilter(batchSizes *[]int64, divisor int, remainder int) scm.Scmer {
	condition := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewBool(scm.ToInt(values[0])%divisor == remainder)
	})
	return scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		batch := RecSetFromScmer(values[0])
		*batchSizes = append(*batchSizes, batch.count)
		return NewRecSetScmer(batch.filterToRecSet(batch.tx, []string{"id"}, condition))
	})
}

func runBatchAcceptIDs(table *table, input *recSet, batchFilter scm.Scmer, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, offset int, limit int) []int64 {
	got := make([]int64, 0, limit)
	mapFn := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		got = append(got, int64(scm.ToInt(values[0])))
		return values[0]
	})
	reduceFn := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[1] })
	source := scanOrderTableSpec{table: table}
	if input != nil {
		source.table = nil
		source.recset = input
	}
	scanOrderBatchAccept(nil, source, batchFilter, sortcols, sortdirs, 0, offset, limit,
		[]string{"id"}, mapFn, reduceFn, scm.NewNil(), false, scm.NewNil())
	return got
}

func TestScanOrderBatchAcceptDoublesDisjointOrderedBatches(t *testing.T) {
	table := setupBatchAcceptTable(t, "tbatchacceptordered", 20)
	_, ascending := integerOrder(false)
	batchSizes := make([]int64, 0)
	got := runBatchAcceptIDs(table, nil, recSetModuloFilter(&batchSizes, 4, 0),
		[]scm.Scmer{scm.NewString("id")}, []func(...scm.Scmer) scm.Scmer{ascending}, 2, 3)

	want := []int64{8, 12, 16}
	if !equalInt64s(got, want) {
		t.Fatalf("accepted ordered rows = %v, want %v", got, want)
	}
	wantBatchSizes := []int64{5, 10, 5}
	if !equalInt64s(batchSizes, wantBatchSizes) {
		t.Fatalf("batch sizes = %v, want disjoint growing windows %v", batchSizes, wantBatchSizes)
	}
}

func TestScanOrderBatchAcceptSupportsRecSetInput(t *testing.T) {
	table := setupBatchAcceptTable(t, "tbatchacceptrecset", 20)
	even := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewBool(scm.ToInt(values[0])%2 == 0)
	})
	input := table.scanRecSet(nil, []string{"id"}, even)
	_, descending := integerOrder(true)
	batchSizes := make([]int64, 0)
	got := runBatchAcceptIDs(table, input, recSetModuloFilter(&batchSizes, 4, 0),
		[]scm.Scmer{scm.NewString("id")}, []func(...scm.Scmer) scm.Scmer{descending}, 0, 2)

	want := []int64{16, 12}
	if !equalInt64s(got, want) {
		t.Fatalf("accepted RecSet rows = %v, want %v", got, want)
	}
	if wantSizes := []int64{2, 4}; !equalInt64s(batchSizes, wantSizes) {
		t.Fatalf("RecSet batch sizes = %v, want %v", batchSizes, wantSizes)
	}
}

func TestScanOrderBatchAcceptGreedyWithoutOrder(t *testing.T) {
	table := setupBatchAcceptTable(t, "tbatchacceptgreedy", 12)
	batchSizes := make([]int64, 0)
	got := runBatchAcceptIDs(table, nil, recSetModuloFilter(&batchSizes, 2, 1), nil, nil, 0, 3)

	want := []int64{1, 3, 5}
	if !equalInt64s(got, want) {
		t.Fatalf("greedy accepted rows = %v, want %v", got, want)
	}
	if wantSizes := []int64{3, 6}; !equalInt64s(batchSizes, wantSizes) {
		t.Fatalf("greedy batch sizes = %v, want %v", batchSizes, wantSizes)
	}
}

func TestScanOrderBatchAcceptMixedOrderAcrossShards(t *testing.T) {
	table := setupBatchAcceptTable(t, "tbatchacceptshards", 30)
	RebuildTable(table, true, false)
	if !table.beginManualRepartition() {
		t.Fatal("manual repartition was not claimed")
	}
	table.repartition([]shardDimension{table.NewShardDimension("id", 3)})
	if len(table.ActiveShards()) < 2 {
		t.Fatalf("repartition produced %d shard(s), want at least 2", len(table.ActiveShards()))
	}
	_, ascending := integerOrder(false)
	_, descending := integerOrder(true)
	batchSizes := make([]int64, 0)
	got := runBatchAcceptIDs(table, nil, recSetModuloFilter(&batchSizes, 2, 0),
		[]scm.Scmer{scm.NewString("grp"), scm.NewString("id")},
		[]func(...scm.Scmer) scm.Scmer{ascending, descending}, 1, 7)

	want := []int64{18, 12, 6, 0, 28, 22, 16}
	if !equalInt64s(got, want) {
		t.Fatalf("mixed-order multishard rows = %v, want %v", got, want)
	}
}

func TestScanOrderBatchAcceptRejectsNonSubset(t *testing.T) {
	table := setupBatchAcceptTable(t, "tbatchacceptsubset", 10)
	allRows := table.scanRecSet(nil, nil, scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }))
	badFilter := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return NewRecSetScmer(allRows) })
	_, ascending := integerOrder(false)

	defer func() {
		if recover() == nil {
			t.Fatal("non-subset batch filter result did not panic")
		}
	}()
	runBatchAcceptIDs(table, nil, badFilter, []scm.Scmer{scm.NewString("id")},
		[]func(...scm.Scmer) scm.Scmer{ascending}, 0, 2)
}

func TestScanOrderBatchAcceptRejectsInputFromDifferentTransaction(t *testing.T) {
	table := setupBatchAcceptTable(t, "tbatchacceptinputtx", 4)
	inputTx := &TxContext{}
	otherTx := &TxContext{}
	input := table.scanRecSet(nil, nil,
		scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }))
	input.tx = inputTx

	defer func() {
		if recover() == nil {
			t.Fatal("input RecSet from another transaction did not panic")
		}
	}()
	scanOrderBatchAccept(otherTx, scanOrderTableSpec{recset: input},
		scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] }),
		nil, nil, 0, 0, 1, []string{"id"},
		scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] }),
		scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[1] }),
		scm.NewNil(), false, scm.NewNil())
}

func TestScanOrderBatchAcceptDeclarationSignature(t *testing.T) {
	table := setupBatchAcceptTable(t, "tbatchacceptdeclare", 6)
	identityFilter := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })
	relation, _ := integerOrder(true)
	got := make([]int64, 0, 2)
	mapFn := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		got = append(got, int64(scm.ToInt(values[0])))
		return values[0]
	})
	reduceFn := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[1] })

	scm.Apply(scm.Globalenv.Vars[scm.Symbol("scan_order_batch_accept")],
		scm.NewNil(),
		NewTableScmer(table),
		identityFilter,
		scm.NewSlice([]scm.Scmer{scm.NewString("id")}),
		scm.NewSlice([]scm.Scmer{relation}),
		scm.NewInt(0),
		scm.NewInt(1),
		scm.NewInt(2),
		scm.NewSlice([]scm.Scmer{scm.NewString("id")}),
		mapFn,
		reduceFn,
		scm.NewNil(),
	)

	if want := []int64{4, 3}; !equalInt64s(got, want) {
		t.Fatalf("declared operator rows = %v, want %v", got, want)
	}
}

func equalInt64s(a []int64, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type countingColumnReader struct {
	values     []scm.Scmer
	singleRead int
	multiRead  int
}

func (r *countingColumnReader) GetValue(recid uint32) scm.Scmer {
	r.singleRead++
	return r.values[recid]
}

func (r *countingColumnReader) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	r.multiRead++
	for i, recid := range recids {
		target[i*stride] = r.values[recid]
	}
}

func (r *countingColumnReader) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	for i := uint32(0); i < count; i++ {
		target[int(i)*stride] = r.values[recid+i]
	}
}

func TestShardMapReducerBulkReadsFinalMapColumns(t *testing.T) {
	first := &countingColumnReader{values: []scm.Scmer{
		scm.NewInt(10), scm.NewInt(20), scm.NewInt(30), scm.NewInt(40),
	}}
	second := &countingColumnReader{values: []scm.Scmer{
		scm.NewInt(1), scm.NewInt(2), scm.NewInt(3), scm.NewInt(4),
	}}
	got := make([]int64, 0, 3)
	mapper := &ShardMapReducer{
		mainGetters: []mapArgGetter{
			func(id uint32, _ uint32) scm.Scmer { return first.GetValue(id) },
			func(id uint32, _ uint32) scm.Scmer { return second.GetValue(id) },
			func(id uint32, _ uint32) scm.Scmer { return scm.NewInt(int64(id * 100)) },
		},
		mainBulkReaders: []ColumnReader{first, second, nil},
		args:            make([]scm.Scmer, 3),
		reduceArgs:      make([]scm.Scmer, 2),
		mapProgram: scm.PrepareSerialProc(scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			got = append(got, int64(scm.ToInt(values[0])+scm.ToInt(values[1])+scm.ToInt(values[2])))
			return values[0]
		})),
		reduceProgram: scm.PrepareSerialProc(scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[1] })),
		mainCount:     4,
	}
	mapper.Stream(scm.NewNil(), []uint32{3, 1, 2}, nil)

	if first.multiRead != 1 || first.singleRead != 0 || second.multiRead != 1 || second.singleRead != 0 {
		t.Fatalf("map reads: first=(multi=%d single=%d) second=(multi=%d single=%d), want one multi and no singles each",
			first.multiRead, first.singleRead, second.multiRead, second.singleRead)
	}
	if want := []int64{344, 122, 233}; !equalInt64s(got, want) {
		t.Fatalf("mapped values = %v, want %v", got, want)
	}
}
