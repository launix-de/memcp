/*
Copyright (C) 2026  MemCP Contributors

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

import "time"
import "testing"
import "container/heap"
import "github.com/launix-de/memcp/scm"

type testPartialCacheObject struct {
	partialBytes int64
	modes        []evictionMode
}

type testRejectingCacheObject struct{}

func (*testRejectingCacheObject) evictionOffer(currentSize int64) evictionOffer {
	return evictionOffer{partialBytes: currentSize, fullBytes: currentSize}
}

func (*testRejectingCacheObject) evict(evictionMode, int64, *[numEvictableTypes]int64) evictionResult {
	return evictionResult{}
}

func (o *testPartialCacheObject) evictionOffer(currentSize int64) evictionOffer {
	return evictionOffer{partialBytes: o.partialBytes, fullBytes: currentSize}
}

func (o *testPartialCacheObject) evict(mode evictionMode, currentSize int64, _ *[numEvictableTypes]int64) evictionResult {
	o.modes = append(o.modes, mode)
	if mode == evictPartial {
		freed := o.partialBytes
		o.partialBytes = 0
		return evictionResult{freedBytes: freed, success: freed > 0}
	}
	return evictionResult{freedBytes: currentSize, fullyEvicted: true, success: true}
}

func TestEvictExpiredRetriesFailedCleanup(t *testing.T) {
	pointer := new(int)
	attempts := 0
	registeredAt := time.Now().Add(-time.Hour).UnixNano()
	item := &softItem{
		pointer:         pointer,
		evictType:       TypeIndex,
		heapIndex:       -1,
		expiryIndex:     -1,
		registeredAt:    registeredAt,
		maxIdleTime:     int64(time.Minute),
		estimatedExpiry: registeredAt + int64(time.Minute),
		getLastUsed:     func(any) time.Time { return time.Unix(0, registeredAt) },
		getScore:        func(any) float64 { return 0 },
	}
	item.object = atomicCacheObject{pointer: pointer, cleanup: func(any, *[numEvictableTypes]int64) bool {
		attempts++
		return attempts > 1
	}}
	cm := &CacheManager{
		countByType: [numEvictableTypes]int64{TypeIndex: 1},
		itemMap:     map[any]*softItem{pointer: item},
	}
	heap.Push(&cm.expH, item)

	cm.evictExpired()
	if _, ok := cm.itemMap[pointer]; !ok {
		t.Fatal("item was removed after cleanup reported lock contention")
	}
	if cm.expH.Len() != 1 {
		t.Fatalf("expiry retry entries = %d, want 1", cm.expH.Len())
	}

	cm.expH[0].estimatedExpiry = time.Now().Add(-time.Second).UnixNano()
	cm.evictExpired()
	if _, ok := cm.itemMap[pointer]; ok {
		t.Fatal("item remained registered after successful cleanup retry")
	}
	if attempts != 2 {
		t.Fatalf("cleanup attempts = %d, want 2", attempts)
	}
}

func TestEvictionHonorsOnlyExplicitMinimumLifetime(t *testing.T) {
	newManager := func(minLifetime time.Duration) (*CacheManager, *int) {
		pointer := new(int)
		item := &softItem{
			pointer:       pointer,
			size:          1,
			evictType:     TypeIndex,
			evictionScore: evictableWeights[TypeIndex],
			heapIndex:     -1,
			expiryIndex:   -1,
			registeredAt:  time.Now().UnixNano(),
			minLifetime:   int64(minLifetime),
			getLastUsed:   func(any) time.Time { return time.Time{} },
			object: atomicCacheObject{
				pointer: pointer,
				cleanup: func(any, *[numEvictableTypes]int64) bool { return true },
			},
		}
		cm := &CacheManager{
			currentMemory: 1,
			sizeByType:    [numEvictableTypes]int64{TypeIndex: 1},
			countByType:   [numEvictableTypes]int64{TypeIndex: 1},
			itemMap:       map[any]*softItem{pointer: item},
		}
		heap.Push(&cm.h, item)
		return cm, pointer
	}

	withoutMinimum, unpinned := newManager(0)
	withoutMinimum.evict(1, 0, 0, nil)
	if _, ok := withoutMinimum.itemMap[unpinned]; ok {
		t.Fatal("zero minimum lifetime prevented immediate pressure eviction")
	}

	withMinimum, pinned := newManager(time.Minute)
	withMinimum.evict(1, 0, 0, nil)
	if _, ok := withMinimum.itemMap[pinned]; !ok {
		t.Fatal("explicit minimum lifetime did not protect the item")
	}
}

func TestEvictionUsesPartialOfferBeforeFullRemoval(t *testing.T) {
	pointer := new(testPartialCacheObject)
	pointer.partialBytes = 40
	item := &softItem{
		pointer:       pointer,
		size:          100,
		evictType:     TypeIndex,
		evictionScore: 100 * evictableWeights[TypeIndex],
		object:        pointer,
		getLastUsed:   func(any) time.Time { return time.Time{} },
		heapIndex:     -1,
		expiryIndex:   -1,
	}
	cm := &CacheManager{
		currentMemory: 100,
		sizeByType:    [numEvictableTypes]int64{TypeIndex: 100},
		countByType:   [numEvictableTypes]int64{TypeIndex: 1},
		itemMap:       map[any]*softItem{pointer: item},
	}
	heap.Push(&cm.h, item)

	cm.evict(100, 60, 0, nil)
	if len(pointer.modes) != 1 || pointer.modes[0] != evictPartial {
		t.Fatalf("eviction modes = %v, want one partial action", pointer.modes)
	}
	if _, ok := cm.itemMap[pointer]; !ok {
		t.Fatal("partial eviction removed top-level object")
	}
	if item.size != 60 || cm.currentMemory != 60 {
		t.Fatalf("sizes after partial eviction: item=%d manager=%d", item.size, cm.currentMemory)
	}
}

func TestEvictionFallsBackToFullAfterPartialOffer(t *testing.T) {
	pointer := new(testPartialCacheObject)
	pointer.partialBytes = 20
	item := &softItem{
		pointer:       pointer,
		size:          100,
		evictType:     TypeIndex,
		evictionScore: 100 * evictableWeights[TypeIndex],
		object:        pointer,
		getLastUsed:   func(any) time.Time { return time.Time{} },
		heapIndex:     -1,
		expiryIndex:   -1,
	}
	cm := &CacheManager{
		currentMemory: 100,
		sizeByType:    [numEvictableTypes]int64{TypeIndex: 100},
		countByType:   [numEvictableTypes]int64{TypeIndex: 1},
		itemMap:       map[any]*softItem{pointer: item},
	}
	heap.Push(&cm.h, item)

	cm.evict(100, 10, 0, nil)
	if len(pointer.modes) != 2 || pointer.modes[0] != evictPartial || pointer.modes[1] != evictFull {
		t.Fatalf("eviction modes = %v, want partial then full", pointer.modes)
	}
	if _, ok := cm.itemMap[pointer]; ok {
		t.Fatal("full fallback retained top-level object")
	}
}

func TestRemoveEagerlyReleasesExpiryReference(t *testing.T) {
	pointer := new(int)
	item := newSoftItem(
		pointer,
		64,
		TypeCacheEntry,
		func(any, *[numEvictableTypes]int64) bool { return true },
		func(any) time.Time { return time.Now() },
		nil,
		0,
		time.Hour,
	)
	cm := &CacheManager{itemMap: make(map[any]*softItem)}
	cm.addInternal(item)
	if cm.expH.Len() != 1 {
		t.Fatalf("expiry entries after add = %d, want 1", cm.expH.Len())
	}
	backing := cm.expH[:cap(cm.expH)]
	cm.removeInternal(pointer, nil)
	if cm.expH.Len() != 0 || item.expiryIndex != -1 {
		t.Fatalf("expiry state after remove: len=%d index=%d", cm.expH.Len(), item.expiryIndex)
	}
	if backing[0] != nil {
		t.Fatal("expiry heap backing array retained removed object")
	}
}

func TestEvictionExpandsTopKAfterRejectedOffer(t *testing.T) {
	rejected := new(testRejectingCacheObject)
	accepted := new(int)
	old := func(any) time.Time { return time.Time{} }
	first := &softItem{
		pointer: rejected, size: 200, evictType: TypeIndex,
		evictionScore: 200 * evictableWeights[TypeIndex], object: rejected,
		getLastUsed: old, heapIndex: -1, expiryIndex: -1,
	}
	second := &softItem{
		pointer: accepted, size: 100, evictType: TypeIndex,
		evictionScore: 100 * evictableWeights[TypeIndex],
		object: atomicCacheObject{pointer: accepted, cleanup: func(any, *[numEvictableTypes]int64) bool {
			return true
		}},
		getLastUsed: old, heapIndex: -1, expiryIndex: -1,
	}
	cm := &CacheManager{
		currentMemory: 300,
		sizeByType:    [numEvictableTypes]int64{TypeIndex: 300},
		countByType:   [numEvictableTypes]int64{TypeIndex: 2},
		itemMap: map[any]*softItem{
			rejected: first,
			accepted: second,
		},
	}
	heap.Push(&cm.h, first)
	heap.Push(&cm.h, second)

	cm.evict(300, 200, 0, nil)
	if _, ok := cm.itemMap[rejected]; !ok {
		t.Fatal("rejected top-k object was removed")
	}
	if _, ok := cm.itemMap[accepted]; ok {
		t.Fatal("manager did not expand top-k after rejected offer")
	}
}

func TestRangePreparationLifetimeFollowsLogicalData(t *testing.T) {
	cache, source := &table{}, &table{}
	calls := 0
	prepare := scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		calls++
		return scm.NewBool(true)
	})
	sources := []scm.Scmer{NewTableScmer(source)}
	cache.prepareCache("range", sources, prepare)
	source.publishTopologyLocked()
	cache.prepareCache("range", sources, prepare)
	if calls != 1 {
		t.Fatalf("physical publication repeated logical preparation: %d", calls)
	}
	source.beginContributionMutation()
	source.endContributionMutation()
	cache.prepareCache("range", sources, prepare)
	if calls != 2 {
		t.Fatalf("source mutation did not refresh preparation: %d", calls)
	}
	cache.prepareCache("range", []scm.Scmer{NewTableScmer(&table{})}, prepare)
	if calls != 3 {
		t.Fatal("a replacement source reused the old preparation")
	}
}

func TestRangePreparationFailureIsRetried(t *testing.T) {
	cache := &table{}
	calls := 0
	prepare := scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		calls++
		if calls == 1 {
			panic("incomplete range preparation")
		}
		return scm.NewBool(true)
	})
	func() {
		defer func() {
			if recover() != "incomplete range preparation" {
				t.Error("preparation failure was not propagated")
			}
		}()
		cache.prepareCache("range", nil, prepare)
	}()
	cache.prepareCache("range", nil, prepare)
	cache.prepareCache("range", nil, prepare)
	if calls != 2 {
		t.Fatalf("failed preparation was retained or successful retry was lost: %d", calls)
	}
}

func TestCacheMapResidentBytesFollowReplacementAndEviction(t *testing.T) {
	cache := &cacheMap{entries: make(map[string]*cacheMapEntry)}
	cache.store("domain", scm.NewString("small"))
	old := cache.entries["domain"]
	if cache.residentBytes.Load() != old.size {
		t.Fatal("memo resident bytes omit the initial entry")
	}
	cache.store("domain", scm.NewString("larger scalar result"))
	next := cache.entries["domain"]
	defer GlobalCache.Remove(next)
	cacheMapCleanup(old, nil)
	if cache.residentBytes.Load() != next.size {
		t.Fatal("old entry cleanup subtracted its replacement")
	}
	cacheMapCleanup(next, nil)
	if cache.residentBytes.Load() != 0 || len(cache.entries) != 0 {
		t.Fatal("evicted memo entry remains resident")
	}
}

func TestMinimumMemoryEvictionFloor(t *testing.T) {
	for _, cause := range []string{"total", "persisted", "system", "allocation", "expiry"} {
		t.Run(cause, func(t *testing.T) {
			cm := &CacheManager{minimumMemory: 100, itemMap: make(map[any]*softItem)}
			for i := 0; i < 3; i++ {
				pointer := new(int)
				item := newSoftItem(pointer, 50, TypeIndex,
					func(any, *[numEvictableTypes]int64) bool { return true },
					func(any) time.Time { return time.Time{} }, nil, 0, time.Nanosecond)
				cm.addInternal(item)
			}
			time.Sleep(time.Millisecond)
			switch cause {
			case "total":
				cm.memoryBudget = 1 // minimum wins even over a smaller maximum
				cm.runEvictionChecks(0)
			case "persisted":
				cm.persistedBudget = 1
				cm.runEvictionChecks(0)
			case "system":
				// Zero available system RAM can reduce the effective budget to zero.
				cm.evict(cm.currentMemory, 0, 0, nil)
			case "allocation":
				cm.memoryBudget = 150
				cm.runEvictionChecks(1000)
			case "expiry":
				cm.evictExpired()
			}
			if cm.currentMemory != 100 {
				t.Fatalf("retained %d, want floor 100", cm.currentMemory)
			}
			cm.evict(cm.currentMemory, 0, 1000, nil)
			cm.evictExpired()
			if cm.currentMemory != 100 {
				t.Fatalf("subsequent eviction crossed floor: %d", cm.currentMemory)
			}
		})
	}
}

func TestMinimumMemorySkipsOversizedOffers(t *testing.T) {
	cm := &CacheManager{minimumMemory: 100, itemMap: make(map[any]*softItem)}
	object := &testPartialCacheObject{partialBytes: 30}
	item := newSoftItem(object, 140, TypeShard, nil, func(any) time.Time { return time.Time{} }, nil, 0, 0)
	cm.addInternal(item)
	cm.evict(cm.currentMemory, 0, 0, nil)
	if cm.currentMemory != 110 || len(object.modes) != 1 || object.modes[0] != evictPartial {
		t.Fatalf("expected partial eviction only: memory=%d modes=%v", cm.currentMemory, object.modes)
	}
	cm.minimumMemory = 0
	cm.evict(cm.currentMemory, 0, 0, nil)
	if cm.currentMemory != 0 {
		t.Fatalf("disabled floor retained %d bytes", cm.currentMemory)
	}
}

func TestMinimumMemoryBudgetUpdate(t *testing.T) {
	cm := &CacheManager{}
	cm.Init(1000, 500, 100)
	defer cm.Stop()
	if got := cm.Stat().MinimumMemory; got != 100 {
		t.Fatalf("initial minimum = %d", got)
	}
	cm.UpdateBudget(2000, 800, 400)
	stat := cm.Stat()
	if stat.MinimumMemory != 400 || stat.MemoryBudget != 2000 || stat.PersistedBudget != 800 {
		t.Fatalf("budget update lost: %+v", stat)
	}
}
