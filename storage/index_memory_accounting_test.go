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

func attachTestSkipList(idx *StorageIndex, state *storageIndexState, cache *sync.Map, key skipListKey, skipList *SkipList) int64 {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	return idx.attachSkipListLocked(state, cache, 0, key, skipList)
}

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

func TestStorageIndexPartialEvictionDropsOnlySingleUseChildren(t *testing.T) {
	idx := &StorageIndex{}
	cache := new(sync.Map)
	idx.baseState.active = true
	idx.baseState.skipLists = []*sync.Map{cache}
	coldKey := skipListKey{pattern: "%one-off%", collation: "utf8_bin"}
	hotKey := skipListKey{pattern: "%paged%", collation: "utf8_bin"}
	cold := &SkipList{}
	hot := &SkipList{}
	coldBytes := attachTestSkipList(idx, &idx.baseState, cache, coldKey, cold)
	hotBytes := attachTestSkipList(idx, &idx.baseState, cache, hotKey, hot)
	if _, ok := loadCachedSkipList(cache, hotKey); !ok {
		t.Fatal("second exact use did not find hot child")
	}
	candidateBytes := idx.baseState.skipListPartialCandidateBytes

	offer := idx.evictionOffer(coldBytes + hotBytes + 1024)
	if offer.partialBytes != coldBytes+hotBytes || offer.fullBytes != coldBytes+hotBytes+1024 {
		t.Fatalf("offer = %+v, want partial=%d full=%d", offer, coldBytes+hotBytes, coldBytes+hotBytes+1024)
	}
	result := idx.evict(evictPartial, offer.fullBytes, new([numEvictableTypes]int64))
	expectedFreed := skipListEntrySize(coldKey, cold) + candidateBytes
	if !result.success || result.fullyEvicted || result.freedBytes != expectedFreed {
		t.Fatalf("partial result = %+v", result)
	}
	if _, ok := cache.Load(coldKey); ok {
		t.Fatal("partial eviction retained one-use child")
	}
	if got, ok := cache.Load(hotKey); !ok || got != hot {
		t.Fatal("partial eviction removed repeatedly used child")
	}
	if !idx.baseState.active {
		t.Fatal("partial eviction removed the parent index")
	}
	expectedRemaining := coldBytes + hotBytes - expectedFreed
	if got := idx.skipListCacheBytes.Load(); got != expectedRemaining {
		t.Fatalf("child bytes = %d, want hot bytes %d", got, expectedRemaining)
	}
	if got := idx.skipListPartialBytes.Load(); got != 0 {
		t.Fatalf("partial candidate bytes = %d, want 0", got)
	}
}

func TestStorageIndexPartialEvictionRetainsSecondUse(t *testing.T) {
	idx := &StorageIndex{}
	cache := new(sync.Map)
	idx.baseState.skipLists = []*sync.Map{cache}
	key := skipListKey{pattern: "%reused%", collation: "utf8_bin"}
	skipList := &SkipList{}
	bytes := attachTestSkipList(idx, &idx.baseState, cache, key, skipList)
	if got := idx.evictionOffer(bytes).partialBytes; got != bytes {
		t.Fatalf("first-use offer = %d, want %d", got, bytes)
	}
	loadCachedSkipList(cache, key)
	candidateBytes := idx.baseState.skipListPartialCandidateBytes
	if got := idx.evictionOffer(bytes).partialBytes; got != bytes {
		t.Fatalf("pre-sampling offer = %d, want upper bound %d", got, bytes)
	}
	result := idx.evict(evictPartial, bytes, new([numEvictableTypes]int64))
	if !result.success || result.fullyEvicted || result.freedBytes != candidateBytes {
		t.Fatalf("partial eviction did not release hot admission metadata: %+v", result)
	}
	if _, ok := cache.Load(key); !ok {
		t.Fatal("hot child disappeared")
	}
	if got := idx.skipListCacheBytes.Load(); got != bytes-candidateBytes {
		t.Fatalf("child bytes = %d, want %d", got, bytes-candidateBytes)
	}
	if got := idx.skipListPartialBytes.Load(); got != 0 {
		t.Fatalf("partial candidate bytes = %d, want 0", got)
	}
}

func TestStorageIndexPromotionRacesPartialEvictionAccounting(t *testing.T) {
	for iteration := 0; iteration < 1000; iteration++ {
		idx := &StorageIndex{}
		cache := new(sync.Map)
		idx.baseState.skipLists = []*sync.Map{cache}
		key := skipListKey{pattern: "%race%", collation: "utf8_bin"}
		skipList := &SkipList{}
		bytes := attachTestSkipList(idx, &idx.baseState, cache, key, skipList)
		start := make(chan struct{})
		done := make(chan struct{})
		go func() {
			<-start
			skipList.recordUse()
			close(done)
		}()
		close(start)
		idx.evict(evictPartial, bytes, new([numEvictableTypes]int64))
		<-done
		if got := idx.skipListPartialBytes.Load(); got != 0 {
			t.Fatalf("iteration %d: partial candidate bytes = %d, want 0", iteration, got)
		}
		hotBytes := skipListEntrySize(key, skipList)
		if got := idx.skipListCacheBytes.Load(); got != 0 && got != hotBytes {
			t.Fatalf("iteration %d: total bytes = %d, want 0 or %d", iteration, got, hotBytes)
		}
		_, retained := cache.Load(key)
		if retained != (idx.skipListCacheBytes.Load() == hotBytes) {
			t.Fatalf("iteration %d: retained=%t disagrees with accounting", iteration, retained)
		}
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
	idx := &StorageIndex{}
	idx.baseState.active = true
	idx.baseState.skipLists = []*sync.Map{cache}
	attachTestSkipList(idx, &idx.baseState, cache, key, queryReference)

	result := idx.evict(evictPartial, 1024, new([numEvictableTypes]int64))
	if !result.success {
		t.Fatal("partial eviction failed")
	}
	if _, ok := cache.Load(key); ok {
		t.Fatal("one-use child remained cached")
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
	childBytes := attachTestSkipList(idx, &idx.baseState, cache, key, skipList)
	manager.UpdateSizeAsync(idx, childBytes)

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
	if _, ok := cache.Load(key); ok {
		t.Fatal("partial pressure eviction retained one-use child")
	}
}
