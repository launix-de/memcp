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
	"testing"

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
	first := projectRecMap(nil, domain, []string{"mid_id"}, scm.NewNil(), middle, []string{"id"}, nil, scm.NewNil())
	if first.count != 4 {
		t.Fatalf("first RecMap domain rows = %d, want 4", first.count)
	}
	middleDomain := first.image()
	if middleDomain.count != 2 {
		t.Fatalf("first RecMap image rows = %d, want 2", middleDomain.count)
	}
	Init(scm.Globalenv)
	operatorMap := scm.Apply(scm.Globalenv.Vars[scm.Symbol("recmap_project_join")],
		scm.NewNil(), NewRecSetScmer(domain),
		scm.NewSlice([]scm.Scmer{scm.NewString("mid_id")}), scm.NewNil(), NewTableScmer(middle),
		scm.NewSlice([]scm.Scmer{scm.NewString("id")}), scm.NewSlice(nil), scm.NewNil())
	if !operatorMap.IsCustom(TagRecMap) {
		t.Fatalf("recmap_project_join returned %s, want RecMap", scm.String(operatorMap))
	}
	operatorImage := scm.Apply(scm.Globalenv.Vars[scm.Symbol("recmap_image")], operatorMap)
	if !operatorImage.IsCustom(TagRecSet) || RecSetFromScmer(operatorImage).count != 2 {
		t.Fatalf("recmap_image returned %s, want two-row RecSet", scm.String(operatorImage))
	}
	second := projectRecMap(nil, middleDomain, []string{"fyear"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewSlice([]scm.Scmer{scm.NewInt(values[0].Int() - 2000)})
	}), target, []string{"id"}, []string{"value"}, scm.NewNil())
	composed := composeRecMaps(first, second)
	if len(composed.shards) == 0 || len(composed.shards[0].targets) == 0 || composed.shards[0].targets[0].value.IsNil() {
		t.Fatal("composed RecMap lost its projected target value")
	}
	activeSourceShards := source.ActiveShards()
	if len(activeSourceShards) != 1 || activeSourceShards[0] != composed.shards[0].sourceShard {
		t.Fatal("composed RecMap does not reference the active source shard")
	}
	mappedValue := NewRecMapScmer(composed)
	directCall := recMapCallClosure(composed.shards[0].sourceShard)
	if got := (*directCall)(composed.shards[0].sourceRecIDs[0], mappedValue); got.IsNil() {
		t.Fatal("row-bound RecMap call did not resolve its source identity")
	}
	if got := scm.Apply(scm.NewClosure(directCall, composed.shards[0].sourceRecIDs[0]), mappedValue); got.IsNil() {
		t.Fatal("Scmer RecMap closure did not resolve its source identity")
	}
	sum := source.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
		nil, scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
		[]string{"$recmap_call"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			value := scm.Apply(values[1], mappedValue)
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
			mapped := projectRecMap(nil, domain, []string{"target_id"}, scm.NewNil(), target, []string{"id"}, nil, scm.NewNil())
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
