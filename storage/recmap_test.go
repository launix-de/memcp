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
	"sync/atomic"
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

func recMapTestTable(tb testing.TB, database, name string, columns []string, rows [][]scm.Scmer) *table {
	tb.Helper()
	tbl, _ := CreateTable(database, name, Memory, true)
	for _, column := range columns {
		tbl.CreateColumn(column, "INT", nil, nil)
	}
	tbl.Insert(columns, rows, nil, scm.NewNil(), false, nil)
	return tbl
}

func recMapTargetValue(target recMapTarget, column string) scm.Scmer {
	if target.shard == nil {
		return scm.NewNil()
	}
	release := target.shard.GetRead()
	defer release()
	storage := target.shard.getColumnStorageOrPanic(column, false, nil)
	target.shard.mu.RLock()
	defer target.shard.mu.RUnlock()
	if target.recid < target.shard.main_count {
		return storage.GetValue(target.recid)
	}
	if _, proxy := storage.(*StorageComputeProxy); proxy {
		return storage.GetValue(target.recid)
	}
	return target.shard.getDelta(int(target.recid-target.shard.main_count), column)
}

func TestRecMapPrunedDomainImageAndComposition(t *testing.T) {
	database := "trecmap"
	databases.Remove(database)
	t.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)

	source := recMapTestTable(t, database, "source", []string{"id", "mid_id"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(10)},
		{scm.NewInt(2), scm.NewInt(20)},
		{scm.NewInt(3), scm.NewInt(10)},
		{scm.NewInt(4), scm.NewInt(99)},
		{scm.NewInt(5), scm.NewInt(20)},
	})
	middle := recMapTestTable(t, database, "middle", []string{"id", "fyear"}, [][]scm.Scmer{
		{scm.NewInt(10), scm.NewInt(2100)},
		{scm.NewInt(20), scm.NewInt(2200)},
	})
	target := recMapTestTable(t, database, "target", []string{"id", "value"}, [][]scm.Scmer{
		{scm.NewInt(100), scm.NewInt(7)},
		{scm.NewInt(200), scm.NewInt(9)},
	})
	result := GetDatabase(database).rebuild(true, false, true)
	if len(result.errors) > 0 {
		t.Fatalf("rebuild errors: %v", result.errors)
	}

	domain := source.scanRecSet(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
		[]string{"id"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			return scm.NewBool(values[0].Int() <= 4)
		}))
	allRows := newScanAccessSchema(scanAccessConsumerScan, nil, -1)
	trueFilter := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	first := scanRecMap(nil, NewTableScmer(source), allRows, nil,
		[]string{"id"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			return scm.NewBool(values[0].Int() <= 4)
		}), []string{"mid_id"}, newRecMapEquiFirstMapper(nil, middle, []string{"id"}, scm.NewNil()), middle)
	if first.count != 4 {
		t.Fatalf("first RecMap domain rows = %d, want 4", first.count)
	}
	middleDomain := first.image()
	if middleDomain.count != 2 {
		t.Fatalf("first RecMap image rows = %d, want 2", middleDomain.count)
	}
	Init(scm.Globalenv)
	operatorMap := scm.Apply(scm.Globalenv.Vars[scm.Symbol("scan_recmap")],
		scm.NewNil(), NewRecSetScmer(domain), allRows, scm.NewSlice(nil),
		scm.NewSlice(nil), trueFilter,
		scm.NewSlice([]scm.Scmer{scm.NewString("mid_id")}),
		newRecMapEquiFirstMapper(nil, middle, []string{"id"}, scm.NewNil()), NewTableScmer(middle))
	if !operatorMap.IsCustom(TagRecMap) {
		t.Fatalf("scan_recmap returned %s, want RecMap", scm.String(operatorMap))
	}
	operatorImage := scm.Apply(scm.Globalenv.Vars[scm.Symbol("recmap_image")], operatorMap)
	if !operatorImage.IsCustom(TagRecSet) || RecSetFromScmer(operatorImage).count != 2 {
		t.Fatalf("recmap_image returned %s, want two-row RecSet", scm.String(operatorImage))
	}
	second := scanRecMap(nil, NewRecSetScmer(middleDomain), allRows, nil,
		nil, trueFilter, []string{"fyear"}, newRecMapEquiFirstMapper(nil, target, []string{"id"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			return scm.NewSlice([]scm.Scmer{scm.NewInt(values[0].Int() - 2000)})
		})), target)
	composed := composeRecMaps(first, second)
	if len(composed.shards) == 0 || len(composed.shards[0].targets) == 0 || composed.shards[0].targets[0].shard == nil {
		t.Fatal("composed RecMap lost its target row identity")
	}
	activeSourceShards := source.ActiveShards()
	if len(activeSourceShards) != 1 || activeSourceShards[0] != composed.shards[0].sourceShard {
		t.Fatal("composed RecMap does not reference the active source shard")
	}
	descending := scm.Apply(scm.Globalenv.Vars[scm.Symbol("collate")],
		scm.NewString("bin"), scm.NewBool(true)).Func()
	window := composed.orderRecSet(nil, []string{"target"}, []string{"value"},
		[]func(...scm.Scmer) scm.Scmer{descending}, 0, 2)
	if window.count != 2 || !window.contains(composed.shards[0].sourceShard, 0) ||
		!window.contains(composed.shards[0].sourceShard, 1) {
		t.Fatalf("target-ordered RecMap window did not retain the expected first two source rows")
	}
	orderedIDs := make([]int64, 0, 2)
	window.scan_order(nil, allRows, nil, nil, trueFilter,
		[]scm.Scmer{scm.NewString("id")}, []func(...scm.Scmer) scm.Scmer{descending},
		0, 0, -1, []string{"id"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			orderedIDs = append(orderedIDs, values[1].Int())
			return values[0]
		}), scm.NewNil(), false, scm.NewNil(), nil, scm.NewNil())
	if len(orderedIDs) != 2 || orderedIDs[0] != 2 || orderedIDs[1] != 1 {
		t.Fatalf("ordered RecMap window scan = %v, want [2 1]", orderedIDs)
	}
	mixedWindow := composed.orderRecSet(nil,
		[]string{"target", "source"}, []string{"value", "id"},
		[]func(...scm.Scmer) scm.Scmer{descending, descending}, 1, 2)
	mixedIDs := make([]int64, 0, 2)
	mixedWindow.scan_order(nil, allRows, nil, nil, trueFilter,
		[]scm.Scmer{scm.NewString("id")}, []func(...scm.Scmer) scm.Scmer{descending},
		0, 0, -1, []string{"id"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			mixedIDs = append(mixedIDs, values[1].Int())
			return values[0]
		}), scm.NewNil(), false, scm.NewNil(), nil, scm.NewNil())
	if len(mixedIDs) != 2 || mixedIDs[0] != 3 || mixedIDs[1] != 1 {
		t.Fatalf("mixed target/source RecMap window scan = %v, want [3 1]", mixedIDs)
	}
	mappedValue := NewRecMapScmer(composed)
	directCall := recMapCallClosure(composed.shards[0].sourceShard, nil)
	columns := scm.NewSlice([]scm.Scmer{scm.NewString("value")})
	valueMapper := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })
	ifNull := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewNil() })
	if got := (*directCall)(composed.shards[0].sourceRecIDs[0], mappedValue, columns, valueMapper, ifNull); got.IsNil() {
		t.Fatal("row-bound RecMap call did not resolve its source identity")
	}
	if got := scm.Apply(scm.NewClosure(directCall, composed.shards[0].sourceRecIDs[0]), mappedValue, columns, valueMapper, ifNull); got.IsNil() {
		t.Fatal("Scmer RecMap closure did not resolve its source identity")
	}
	sum := source.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
		nil, scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
		[]string{"$recmap_call"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			value := scm.Apply(values[1], mappedValue, columns, valueMapper, ifNull)
			if value.IsNil() {
				return values[0]
			}
			return scm.NewInt(values[0].Int() + value.Int())
		}), scm.NewInt(0), scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			return scm.NewInt(values[0].Int() + values[1].Int())
		}), false)
	if sum.Int() != 23 {
		t.Fatalf("$recmap_call projected sum = %d, want 23", sum.Int())
	}

	want := map[int64]scm.Scmer{
		1: scm.NewInt(7),
		2: scm.NewInt(9),
		3: scm.NewInt(7),
		4: scm.NewNil(),
	}
	for _, part := range composed.shards {
		for i, recid := range part.sourceRecIDs {
			id := recMapTargetValue(recMapTarget{shard: part.sourceShard, recid: recid}, "id").Int()
			got := recMapTargetValue(part.targets[i], "value")
			if !scm.Equal(got, want[id]) {
				t.Fatalf("source id %d mapped value = %s, want %s", id, scm.String(got), scm.String(want[id]))
			}
		}
	}
	for _, part := range domain.shards {
		if _, found := composed.lookup(part.shard, 4); found {
			t.Fatal("RecMap contains source row outside the pruned input domain")
		}
	}
}

func TestScanRecMapFiltersBatchMappingAndMissingTarget(t *testing.T) {
	database := "trecmap_batch"
	databases.Remove(database)
	t.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	source := recMapTestTable(t, database, "source", []string{"id", "target_id"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(10)},
		{scm.NewInt(2), scm.NewInt(20)},
		{scm.NewInt(3), scm.NewInt(99)},
		{scm.NewInt(4), scm.NewInt(10)},
	})
	target := recMapTestTable(t, database, "target", []string{"id", "value"}, [][]scm.Scmer{
		{scm.NewInt(10), scm.NewNil()},
		{scm.NewInt(20), scm.NewInt(8)},
	})
	if result := GetDatabase(database).rebuild(true, false, true); len(result.errors) > 0 {
		t.Fatalf("rebuild errors: %v", result.errors)
	}
	access := newScanAccessSchema(scanAccessConsumerScan, nil, -1)
	mapper := newRecMapEquiFirstMapper(nil, target, []string{"id"}, scm.NewNil())
	batchCalls := 0
	batchMapper := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		batchCalls++
		return scm.Apply(mapper, args...)
	})
	filter := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewBool(values[0].Int() <= 3)
	})
	mapped := scanRecMap(nil, NewTableScmer(source), access, nil, []string{"id"}, filter,
		[]string{"target_id"}, batchMapper, target)
	if mapped.count != 3 || batchCalls != 1 {
		t.Fatalf("table+filter mapped %d rows in %d batches, want 3 rows in one batch", mapped.count, batchCalls)
	}
	part := &mapped.shards[0]
	call := recMapCallClosure(part.sourceShard, nil)
	columns := scm.NewSlice([]scm.Scmer{scm.NewString("value")})
	mapperCalls, missingCalls := 0, 0
	valueMapper := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		mapperCalls++
		if values[0].IsNil() {
			return scm.NewString("matched-null")
		}
		return values[0]
	})
	ifNull := scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		missingCalls++
		return scm.NewString("missing")
	})
	ref := NewRecMapScmer(mapped)
	for id, want := range map[uint32]string{0: "matched-null", 1: "8", 2: "missing", 3: "missing"} {
		got := (*call)(id, ref, columns, valueMapper, ifNull)
		if scm.String(got) != want {
			t.Fatalf("source recid %d = %s, want %s", id, scm.String(got), want)
		}
	}
	if mapperCalls != 2 || missingCalls != 2 {
		t.Fatalf("matched callback %d, no-match callback %d; want 2 each", mapperCalls, missingCalls)
	}
	input := source.scanRecSet(nil, access, nil, []string{"id"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewBool(values[0].Int() >= 2)
	}))
	narrowed := scanRecMap(nil, NewRecSetScmer(input), access, nil, []string{"id"}, filter,
		[]string{"target_id"}, mapper, target)
	if narrowed.count != 2 {
		t.Fatalf("RecSet+filter mapped %d rows, want only the intersection", narrowed.count)
	}
	if _, found := narrowed.lookup(part.sourceShard, 0); found {
		t.Fatal("RecMap includes a source row outside its RecSet input")
	}
	// The constructor accepts arbitrary 0..1 batch mappings, not only its
	// equality-join convenience mapper. The callback can select a target row
	// using visibility, ranking, or an application-specific lookup.
	targetShard := target.ActiveShards()[0]
	custom := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		batch := args[0].Slice()
		refs := make([]scm.Scmer, len(batch))
		for i, tuple := range batch {
			if tuple.Slice()[0].Int() == 2 {
				refs[i] = newRecordRef(targetShard, 1)
			}
		}
		return scm.NewSlice(refs)
	})
	customMap := scanRecMap(nil, NewRecSetScmer(input), access, nil, nil,
		scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
		[]string{"id"}, custom, target)
	if got := (*recMapCallClosure(customMap.shards[0].sourceShard, nil))(1,
		NewRecMapScmer(customMap), columns, valueMapper, ifNull); got.Int() != 8 {
		t.Fatalf("custom row mapping = %s, want 8", scm.String(got))
	}
	moreRows := make([][]scm.Scmer, recMapMapperBatchSize+1)
	for i := range moreRows {
		moreRows[i] = []scm.Scmer{scm.NewInt(int64(i + 100)), scm.NewInt(10)}
	}
	source.Insert([]string{"id", "target_id"}, moreRows, nil, scm.NewNil(), false, nil)
	boundedCalls := 0
	bounded := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		boundedCalls++
		if n := len(args[0].Slice()); n > recMapMapperBatchSize {
			t.Fatalf("mapper received %d tuples, limit %d", n, recMapMapperBatchSize)
		}
		return scm.NewSlice(make([]scm.Scmer, len(args[0].Slice())))
	})
	all := scanRecMap(nil, NewTableScmer(source), access, nil, nil,
		scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
		[]string{"id"}, bounded, target)
	if all.count != int64(4+len(moreRows)) || boundedCalls != 2 {
		t.Fatalf("bounded mapper: %d rows in %d calls, want %d rows in 2 calls",
			all.count, boundedCalls, 4+len(moreRows))
	}
}

func TestScanRecMapShardParallelMapping(t *testing.T) {
	database := "trecmap_parallel"
	databases.Remove(database)
	t.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)
	source, _ := CreateTable(database, "source", Memory, true)
	source.CreateColumn("id", "INT", nil, nil)
	source.ShardMode = ShardModePartition
	source.PDimensions = []shardDimension{{Column: "id", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(10)}}}
	source.PShards = []*storageShard{NewShard(source), NewShard(source)}
	source.publishTopologyLocked()
	source.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}, {scm.NewInt(11)}}, nil, scm.NewNil(), false, nil)
	target := recMapTestTable(t, database, "target", []string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}})
	oldProcs := runtime.GOMAXPROCS(4)
	t.Cleanup(func() { runtime.GOMAXPROCS(oldProcs) })
	var inFlight, peak atomic.Int32
	mapper := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		active := inFlight.Add(1)
		for {
			old := peak.Load()
			if active <= old || peak.CompareAndSwap(old, active) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		inFlight.Add(-1)
		return scm.NewSlice(make([]scm.Scmer, len(args[0].Slice())))
	})
	mapped := scanRecMap(nil, NewTableScmer(source), newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
		nil, scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
		[]string{"id"}, mapper, target)
	if mapped.count != 2 || len(mapped.shards) != 2 {
		t.Fatalf("mapped %d records across %d shards, want 2 across 2", mapped.count, len(mapped.shards))
	}
	if peak.Load() < 2 {
		t.Fatalf("peak concurrent shard mappers = %d, want at least 2", peak.Load())
	}
}

func BenchmarkWindowRecMapAgainstRepeatedProbe(b *testing.B) {
	// Model one adaptive ORDER BY window: only 72 source rows are relevant and
	// they repeat 16 correlation keys. image_only isolates the cost of retaining
	// source-to-target correspondence from the already-required pruned join.
	const sourceRows = 8_192
	const targetRows = 2_048
	const windowRows = 72
	const windowKeys = 16
	database := fmt.Sprintf("bench_recmap_%p", b)
	databases.Remove(database)
	b.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)

	sourceData := make([][]scm.Scmer, sourceRows)
	for i := range sourceData {
		sourceData[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewInt(int64(i % windowKeys))}
	}
	targetData := make([][]scm.Scmer, targetRows)
	for i := range targetData {
		targetData[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewInt(int64(i * 3))}
	}
	source := recMapTestTable(b, database, "source", []string{"id", "target_id"}, sourceData)
	target := recMapTestTable(b, database, "target", []string{"id", "value"}, targetData)
	target.Unique = append(target.Unique, uniqueKey{Id: "PRIMARY", Cols: []string{"id"}})
	result := GetDatabase(database).rebuild(true, false, true)
	if len(result.errors) > 0 {
		b.Fatalf("rebuild errors: %v", result.errors)
	}
	domain := source.scanRecSet(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
		[]string{"id"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			return scm.NewBool(values[0].Int() < windowRows)
		}))

	b.Run("repeated_scalar_probe", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			rows := domain.collectRecMapSourceRows(nil, []string{"target_id"})
			for _, row := range rows {
				keys := recSetProjectKeys{width: 1, values: row.key}
				if firstRecSetTargetForBenchmark(target.projectJoinKeysToRecSet(nil, []string{"id"}, keys, nil)).shard == nil {
					b.Fatal("missing repeated probe target")
				}
			}
		}
	})
	b.Run("memoized_scalar_probe", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			rows := domain.collectRecMapSourceRows(nil, []string{"target_id"})
			cache := make(map[int64]recMapTarget, windowKeys)
			for _, row := range rows {
				key := row.key[0].Int()
				targetRef, found := cache[key]
				if !found {
					keys := recSetProjectKeys{width: 1, values: row.key}
					targetRef = firstRecSetTargetForBenchmark(target.projectJoinKeysToRecSet(nil, []string{"id"}, keys, nil))
					cache[key] = targetRef
				}
				if targetRef.shard == nil {
					b.Fatal("missing memoized probe target")
				}
			}
		}
	})
	b.Run("window_recmap", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			mapped := scanRecMap(nil, NewRecSetScmer(domain), newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
				nil, scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
				[]string{"target_id"}, newRecMapEquiFirstMapper(nil, target, []string{"id"}, scm.NewNil()), target)
			if mapped.count != windowRows {
				b.Fatalf("mapped rows = %d, want %d", mapped.count, windowRows)
			}
		}
	})
	b.Run("window_recset_image_only", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			image := domain.projectJoin(nil, []string{"target_id"}, target, []string{"id"})
			if image.count != windowKeys {
				b.Fatalf("projected rows = %d, want %d", image.count, windowKeys)
			}
		}
	})
}

func firstRecSetTargetForBenchmark(rows *recSet) recMapTarget {
	if rows == nil {
		return recMapTarget{}
	}
	for partIndex := range rows.shards {
		part := &rows.shards[partIndex]
		var target recMapTarget
		part.forEachID(func(recid uint32) bool {
			target = recMapTarget{shard: part.shard, recid: recid}
			return false
		})
		if target.shard != nil {
			return target
		}
	}
	return recMapTarget{}
}
