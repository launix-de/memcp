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

import "sync"
import "testing"
import "time"

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

func TestStorageIndexPartialEvictionOwnsSkipListChildren(t *testing.T) {
	idx := &StorageIndex{}
	cache := new(sync.Map)
	idx.baseState.active = true
	idx.baseState.skipLists = []*sync.Map{cache}
	key := skipListKey{pattern: "%needle%", collation: "utf8_bin"}
	skipList := &SkipList{}
	cache.Store(key, skipList)
	childBytes := skipListEntrySize(key, skipList)
	idx.baseState.skipListBytes.Store(childBytes)
	idx.skipListCacheBytes.Store(childBytes)

	offer := idx.evictionOffer(childBytes + 1024)
	if offer.partialBytes != childBytes || offer.fullBytes != childBytes+1024 {
		t.Fatalf("offer = %+v, want partial=%d full=%d", offer, childBytes, childBytes+1024)
	}
	result := idx.evict(evictPartial, offer.fullBytes, new([numEvictableTypes]int64))
	if !result.success || result.fullyEvicted || result.freedBytes != childBytes {
		t.Fatalf("partial result = %+v", result)
	}
	if idx.baseState.skipLists[0] == cache {
		t.Fatal("partial eviction retained the published child map")
	}
	if _, ok := idx.baseState.skipLists[0].Load(key); ok {
		t.Fatal("replacement child map is not empty")
	}
	if !idx.baseState.active {
		t.Fatal("partial eviction removed the parent index")
	}
	if got := idx.skipListCacheBytes.Load(); got != 0 {
		t.Fatalf("child bytes = %d, want 0", got)
	}
}

func TestLoadCachedSkipListUsesExactTermAndTracksReuse(t *testing.T) {
	cache := &sync.Map{}
	shortKey := skipListKey{pattern: "%car%", collation: "utf8_bin"}
	longKey := skipListKey{pattern: "%carglass%", collation: "utf8_bin"}
	short := &SkipList{}
	cache.Store(shortKey, short)

	if _, ok := loadCachedSkipList(cache, longKey); ok {
		t.Fatal("a cached substring must not be reused for a different LIKE term")
	}
	if got := short.hitCount.Load(); got != 0 {
		t.Fatalf("substring lookup changed hit count to %d, want 0", got)
	}

	first, ok := loadCachedSkipList(cache, shortKey)
	if !ok || first != short {
		t.Fatal("exact LIKE term was not loaded from cache")
	}
	firstUsed := short.lastUsed()
	if firstUsed.IsZero() || short.hitCount.Load() != 1 {
		t.Fatal("exact cache hit did not update per-term lifecycle")
	}
	loadCachedSkipList(cache, shortKey)
	if short.lastUsed().Before(firstUsed) {
		t.Fatal("repeated cache hit moved last-used timestamp backwards")
	}
	if got := short.cacheScore(); got != 2 {
		t.Fatalf("cache score = %v, want 2 exact uses", got)
	}
}

func TestStorageIndexPartialEvictionKeepsActiveQueryReference(t *testing.T) {
	cache := &sync.Map{}
	key := skipListKey{pattern: "%needle%", collation: "utf8_bin"}
	queryReference := &SkipList{}
	cache.Store(key, queryReference)
	idx := &StorageIndex{}
	idx.baseState.active = true
	idx.baseState.skipLists = []*sync.Map{cache}
	idx.baseState.skipListBytes.Store(skipListEntrySize(key, queryReference))
	idx.skipListCacheBytes.Store(skipListEntrySize(key, queryReference))

	result := idx.evict(evictPartial, 1024, new([numEvictableTypes]int64))
	if !result.success {
		t.Fatal("partial eviction failed")
	}
	if idx.baseState.skipLists[0] == cache {
		t.Fatal("old cache remained published on the parent")
	}
	if got, ok := cache.Load(key); !ok || got != queryReference {
		t.Fatal("active query's snapshotted map was invalidated")
	}
	if queryReference.cursor().skip != queryReference {
		t.Fatal("active query reference was invalidated")
	}
}

func TestCacheManagerTracksOnlyTopLevelIndex(t *testing.T) {
	manager := new(CacheManager)
	manager.Init(0, 0)
	defer manager.Stop()

	cache := new(sync.Map)
	idx := &StorageIndex{}
	idx.baseState.active = true
	idx.baseState.skipLists = []*sync.Map{cache}
	key := skipListKey{pattern: "%needle%", collation: "utf8_bin"}
	skipList := &SkipList{}
	childBytes := skipListEntrySize(key, skipList)
	const baseBytes = int64(1024)
	manager.AddItemEx(
		idx,
		baseBytes,
		TypeIndex,
		func(any, *[numEvictableTypes]int64) bool { return true },
		func(any) time.Time { return time.Time{} },
		nil,
		0,
		0,
	)
	idx.mu.Lock()
	cache.Store(key, skipList)
	idx.baseState.skipListBytes.Store(childBytes)
	idx.skipListCacheBytes.Store(childBytes)
	manager.UpdateSizeAsync(idx, childBytes)
	idx.mu.Unlock()

	before := manager.Stat()
	if before.CountByType[TypeIndex] != 1 {
		t.Fatalf("global index records = %d, want one top-level object", before.CountByType[TypeIndex])
	}
	if before.SizeByType[TypeIndex] != baseBytes+childBytes {
		t.Fatalf("top-level size = %d, want base+children %d", before.SizeByType[TypeIndex], baseBytes+childBytes)
	}
	manager.UpdateBudget(baseBytes, 0)
	after := manager.Stat()
	if after.CountByType[TypeIndex] != 1 || after.SizeByType[TypeIndex] != baseBytes {
		t.Fatalf("after partial eviction: count=%d size=%d", after.CountByType[TypeIndex], after.SizeByType[TypeIndex])
	}
	if idx.baseState.skipLists[0] == cache {
		t.Fatal("partial pressure eviction retained published child map")
	}
}
