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

import "github.com/google/btree"
import "github.com/launix-de/memcp/scm"

func TestPlannerIndexProbeDoesNotIncreaseIndexSavings(t *testing.T) {
	Init(scm.Globalenv)
	dbname := "test_estimate_no_savings"
	CreateDatabase(dbname, true)
	defer databases.Remove(dbname)

	tbl, _ := CreateTable(dbname, "items", Memory, true)
	tbl.CreateColumn("id", "INT", nil, nil)
	rows := make([][]scm.Scmer, 0, 16)
	for i := 0; i < 16; i++ {
		rows = append(rows, []scm.Scmer{scm.NewInt(int64(i))})
	}
	tbl.Insert([]string{"id"}, rows, nil, scm.NewNil(), false, nil)

	shard := tbl.Shards[0]
	bounds := boundaries{{col: "id", matcher: EqualMatcher, lower: scm.NewInt(5), lowerInclusive: true, upper: scm.NewInt(5), upperInclusive: true}}
	lower, upperLast := indexFromBoundaries(bounds)
	var buf [8]uint32
	shard.mu.RLock()
	shard.iterateIndex(nil, bounds, lower, upperLast, len(shard.inserts), buf[:], 0, nil, func(batch []uint32) bool {
		return true
	})
	shard.mu.RUnlock()
	if len(shard.Indexes) != 1 {
		t.Fatalf("metadata indexes = %d, want 1", len(shard.Indexes))
	}
	if shard.Indexes[0].Savings != 0 {
		t.Fatalf("estimate must not count as index usage, savings = %v", shard.Indexes[0].Savings)
	}
	if shard.Indexes[0].baseState.active {
		t.Fatal("estimate must not materialize a cold auto-index before real usage")
	}
}

func TestStorageIndexComputeSizeCountsDeltaBtree(t *testing.T) {
	idx := &StorageIndex{}
	state := &idx.baseState
	state.deltaBtree = btree.NewG[indexPair](8, func(a, b indexPair) bool {
		return a.itemid < b.itemid
	})
	state.deltaBtree.ReplaceOrInsert(indexPair{itemid: 1, data: []scm.Scmer{scm.NewInt(1)}})
	state.deltaBtree.ReplaceOrInsert(indexPair{itemid: 2, data: []scm.Scmer{scm.NewInt(2)}})

	baseOnly := uint(24 * 8)
	if got := idx.ComputeSize(); got <= baseOnly {
		t.Fatalf("ComputeSize() = %d, want larger than base-only size %d", got, baseOnly)
	}
}

func TestRemoveIndexChildrenInternalDropsSkipListRecords(t *testing.T) {
	idx := &StorageIndex{}
	other := &StorageIndex{}
	entry := &skipListCacheEntry{index: idx, colIdx: 0, pattern: "%needle%"}
	otherEntry := &skipListCacheEntry{index: other, colIdx: 0, pattern: "%needle%"}
	cm := &CacheManager{
		itemMap: map[any]*softItem{
			entry:      {pointer: entry, size: 128, evictType: TypeIndex, heapIndex: -1},
			otherEntry: {pointer: otherEntry, size: 64, evictType: TypeIndex, heapIndex: -1},
		},
		currentMemory: 192,
		sizeByType:    [numEvictableTypes]int64{TypeIndex: 192},
		countByType:   [numEvictableTypes]int64{TypeIndex: 2},
	}

	var freed [numEvictableTypes]int64
	cm.removeIndexChildrenInternal(idx, &freed)
	if _, ok := cm.itemMap[entry]; ok {
		t.Fatal("skip list cache entry for removed index is still registered")
	}
	if _, ok := cm.itemMap[otherEntry]; !ok {
		t.Fatal("skip list cache entry for other index was removed")
	}
	if freed[TypeIndex] != 128 {
		t.Fatalf("freed index bytes = %d, want 128", freed[TypeIndex])
	}
	if cm.currentMemory != 64 {
		t.Fatalf("currentMemory = %d, want 64", cm.currentMemory)
	}
}
