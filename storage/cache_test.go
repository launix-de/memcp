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

import "sync"
import "time"
import "runtime"
import "testing"
import "sync/atomic"
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
	withoutMinimum.evict(1, 0, 0, nil, nil)
	if _, ok := withoutMinimum.itemMap[unpinned]; ok {
		t.Fatal("zero minimum lifetime prevented immediate pressure eviction")
	}

	withMinimum, pinned := newManager(time.Minute)
	withMinimum.evict(1, 0, 0, nil, nil)
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

	cm.evict(100, 60, 0, nil, nil)
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

	cm.evict(100, 10, 0, nil, nil)
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

	cm.evict(300, 200, 0, nil, nil)
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
	cache.prepareCache("range", sources, nil, prepare)
	source.publishTopologyLocked()
	cache.prepareCache("range", sources, nil, prepare)
	if calls != 1 {
		t.Fatalf("physical publication repeated logical preparation: %d", calls)
	}
	source.beginContributionMutation()
	source.endContributionMutation()
	cache.prepareCache("range", sources, nil, prepare)
	if calls != 2 {
		t.Fatalf("source mutation did not refresh preparation: %d", calls)
	}
	cache.prepareCache("range", []scm.Scmer{NewTableScmer(&table{})}, nil, prepare)
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
		cache.prepareCache("range", nil, nil, prepare)
	}()
	cache.prepareCache("range", nil, nil, prepare)
	cache.prepareCache("range", nil, nil, prepare)
	if calls != 2 {
		t.Fatalf("failed preparation was retained or successful retry was lost: %d", calls)
	}
}

// Factories run outside the cache lock and concurrent misses share one result.
// The second key is written while the factory is paused, so lock ownership is
// tested without relying on a sleep or on scheduler speed.
func TestCacheMapConstructorFactorySharesConcurrentMiss(t *testing.T) {
	var calls atomic.Int64
	started, release := make(chan struct{}), make(chan struct{})
	cache := NewCacheMap(scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return args[0]
	}))
	var readers sync.WaitGroup
	results := make(chan scm.Scmer, 16)
	for range 16 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			results <- scm.Apply(cache, scm.NewString("shared"))
		}()
	}
	<-started
	scm.Apply(cache, scm.NewString("other"), scm.NewInt(9))
	close(release)
	readers.Wait()
	close(results)
	for value := range results {
		if scm.String(value) != "shared" {
			t.Fatalf("factory received the wrong key: %v", value)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("concurrent misses ran the constructor factory %d times", calls.Load())
	}
}

func TestCacheMapFactoriesRetryFailuresAndRetainNil(t *testing.T) {
	var calls int
	cache := NewCacheMap(scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		calls++
		if calls == 1 {
			panic("factory failed")
		}
		return scm.NewNil()
	}))
	func() {
		defer func() {
			if recovered := recover(); recovered != "factory failed" {
				t.Fatalf("factory panic was not forwarded: %v", recovered)
			}
		}()
		scm.Apply(cache, scm.NewString("missing"))
	}()
	for range 2 {
		if !scm.Apply(cache, scm.NewString("missing")).IsNil() {
			t.Fatal("factory nil was not preserved")
		}
	}
	if calls != 2 {
		t.Fatalf("failed value was cached or nil was recomputed: %d factory calls", calls)
	}
}

func TestCacheMapAccessFactoryOverridesConstructorAndStoresCallables(t *testing.T) {
	constructor := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewInt(1) })
	local := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewInt(2) })
	cache := NewCacheMap(constructor)
	key := scm.NewString("key")
	if scm.Apply(cache, key, scm.NewNil(), local).Int() != 2 {
		t.Fatal("access-local factory did not override the constructor")
	}
	if scm.Apply(cache, key).Int() != 2 {
		t.Fatal("constructor replaced the cached value")
	}
	scm.Apply(cache, key, constructor)
	if !scm.Apply(cache, key).IsNativeFunc() {
		t.Fatal("two-argument setter treated the stored callable as a factory")
	}
	other := NewCacheMap(constructor)
	if scm.Apply(other, key).Int() != 1 {
		t.Fatal("independent cache instances shared values")
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
				cm.runEvictionChecks(0, nil)
			case "persisted":
				cm.persistedBudget = 1
				cm.runEvictionChecks(0, nil)
			case "system":
				// Zero available system RAM can reduce the effective budget to zero.
				cm.evict(cm.currentMemory, 0, 0, nil, nil)
			case "allocation":
				cm.memoryBudget = 150
				cm.runEvictionChecks(1000, nil)
			case "expiry":
				cm.evictExpired()
			}
			if cm.currentMemory != 100 {
				t.Fatalf("retained %d, want floor 100", cm.currentMemory)
			}
			cm.evict(cm.currentMemory, 0, 1000, nil, nil)
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
	cm.evict(cm.currentMemory, 0, 0, nil, nil)
	if cm.currentMemory != 110 || len(object.modes) != 1 || object.modes[0] != evictPartial {
		t.Fatalf("expected partial eviction only: memory=%d modes=%v", cm.currentMemory, object.modes)
	}
	cm.minimumMemory = 0
	cm.evict(cm.currentMemory, 0, 0, nil, nil)
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

// Retirement is opportunistic. DDL/eviction contention must leave both data and
// lifetime pins intact and return promptly, never wait while owning schema state.
func TestDiscardCacheValueDefersToDDLAndEviction(t *testing.T) {
	tbl := &table{schema: &database{}, Name: ".grp:retention", PersistencyMode: Cache}
	args := func() scm.Scmer {
		return tbl.discardCacheValue(nil, []string{"axis"}, []scm.Scmer{scm.NewInt(1)}, "value", scm.NewInt(7))
	}
	tbl.ddlMu.Lock()
	if result := args(); scm.ToInt(result) != 0 {
		t.Fatal("retired a value while DDL owned the definition")
	}
	tbl.ddlMu.Unlock()
	if tbl.cacheUsers != 0 {
		t.Fatal("DDL contention leaked the table cache pin")
	}
	if !tbl.beginCacheEviction() {
		t.Fatal("could not start eviction after retirement deferred")
	}
	if result := args(); scm.ToInt(result) != 0 {
		t.Fatal("retired a value while cache eviction owned the table")
	}
	if tbl.cacheUsers != -1 {
		t.Fatal("retirement changed an evicted table's lifetime state")
	}
}

// Only semantic preparation results cross into the planner. Domain changes and
// logical cell mutations discard that result; physical rebuilds never do.
func TestRangePreparationReusesOnlyUnchangedDomainProof(t *testing.T) {
	cache, driver, source := &table{}, &table{}, &table{}
	var previous []scm.Scmer
	prepare := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		previous = append(previous, args[0])
		return scm.NewBool(true)
	})
	run := func() {
		cache.prepareCache("range", []scm.Scmer{NewTableScmer(driver)}, []scm.Scmer{NewTableScmer(source)}, prepare)
	}
	mutate := func(tbl *table) { tbl.beginContributionMutation(); tbl.endContributionMutation() }
	run()
	mutate(source)
	run()
	if len(previous) != 2 || !previous[0].IsNil() || !scm.ToBool(previous[1]) {
		t.Fatalf("source DML lost domain proof: %v", previous)
	}
	driver.publishTopologyLocked()
	source.publishTopologyLocked()
	cache.publishTopologyLocked()
	run()
	if len(previous) != 2 {
		t.Fatal("topology change repeated preparation")
	}
	mutate(driver)
	run()
	if !previous[2].IsNil() {
		t.Fatal("driver mutation reused stale proof")
	}
	mutate(cache)
	run()
	if !previous[3].IsNil() {
		t.Fatal("another domain's merge reused stale proof")
	}
	cache.cacheGeneration.Add(1)
	run()
	if !previous[4].IsNil() {
		t.Fatal("eviction reused stale proof")
	}
}

func TestCacheAdmissionReclaimsPublishedOwnersWithoutIdleDelay(t *testing.T) {
	var manager CacheManager
	manager.Init(16, 16, 0)
	defer manager.Stop()
	pointer := new(int)
	published := time.Now()
	manager.AddItem(pointer, 32, TypeShard,
		func(any, *[numEvictableTypes]int64) bool { return true },
		func(any) time.Time { return published }, nil)
	if usage := manager.Stat(); usage.CurrentMemory != 0 || !manager.canLoadClusterShard(nil) {
		// This object has no live lease. Recent use affects eviction ranking,
		// but must not strand an over-budget payload until an idle timer passes.
		t.Fatalf("published evictable payload prevented subsequent admission: %+v", usage)
	}
}

func TestCachePressureDoesNotDisplaceShardPayload(t *testing.T) {
	var manager CacheManager
	manager.Init(1000, 700, 0)
	defer manager.Stop()
	var payloadEvicted, cacheEvicted atomic.Bool
	published := time.Now()
	manager.AddItem(new(int), 600, TypeShard,
		func(any, *[numEvictableTypes]int64) bool { payloadEvicted.Store(true); return true },
		func(any) time.Time { return time.Now().Add(-time.Hour) }, nil)
	manager.AddItem(new(int), 800, TypeCacheEntry,
		func(any, *[numEvictableTypes]int64) bool { cacheEvicted.Store(true); return true },
		func(any) time.Time { return published }, nil)
	manager.Stat()
	if payloadEvicted.Load() || !cacheEvicted.Load() {
		t.Fatalf("cache growth: payload evicted=%v cache evicted=%v stat=%+v", payloadEvicted.Load(), cacheEvicted.Load(), manager.Stat())
	}
	manager.CheckPressure(1000)
	if payloadEvicted.Load() {
		t.Fatal("a pending cache allocation displaced native shard storage")
	}
	manager.AddItem(new(int), 200, TypeShard,
		func(any, *[numEvictableTypes]int64) bool { return true }, func(any) time.Time { return published }, nil)
	manager.Stat()
	if !payloadEvicted.Load() {
		t.Fatal("new shard storage failed to displace older shard storage")
	}
	manager.UpdateBudget(100, 700, 0)
	if manager.Stat().ShardMemory > 100 {
		t.Fatal("an explicit total-budget reduction did not shrink the payload budget")
	}
}

func TestClusterTableQuotaReclaimsOnlyThatTablesPayload(t *testing.T) {
	previous := sharedClusterMounts.Swap(&map[string]bool{})
	defer sharedClusterMounts.Store(previous)
	var manager CacheManager
	manager.Init(6000, 6000, 0)
	defer manager.Stop()
	outer, inner := &table{}, &table{}
	var outerEvicted, innerEvicted atomic.Bool
	add := func(tbl *table, size int64, evicted *atomic.Bool, used time.Time) {
		pointer := &storageShard{t: tbl}
		cleanup := func(any, *[numEvictableTypes]int64) bool { evicted.Store(true); return true }
		item := newSoftItem(pointer, size, TypeShard, cleanup, func(any) time.Time { return used }, nil, 0, 0)
		// The fixture models reclaimable payload with the real native owner.
		item.object = atomicCacheObject{pointer: pointer, cleanup: cleanup}
		manager.opChan <- cacheOp{add: item}
		manager.Stat()
	}
	add(inner, 600, &innerEvicted, time.Now().Add(-2*time.Hour))
	add(outer, 600, &outerEvicted, time.Now().Add(-time.Hour))
	add(outer, 600, new(atomic.Bool), time.Now())
	if !outerEvicted.Load() || innerEvicted.Load() {
		t.Fatal("outer table displaced the inner table's reserved payload")
	}
	if !manager.canLoadClusterShard(outer) || !manager.canLoadClusterShard(inner) {
		t.Fatal("evicted payload left admission closed")
	}
}

// Queued allocation accounting must not repeatedly traverse the entire cache
// while the same native payload remains pinned. A subsequent barrier still
// observes every byte and closed admission; releasing the pin permits eviction.
type testPinnedAccountingObject struct {
	offers   atomic.Int64
	released atomic.Bool
}

func (o *testPinnedAccountingObject) evictionOffer(size int64) evictionOffer {
	o.offers.Add(1)
	if o.released.Load() {
		return evictionOffer{fullBytes: size}
	}
	return evictionOffer{}
}

func (o *testPinnedAccountingObject) evict(_ evictionMode, size int64, _ *[numEvictableTypes]int64) evictionResult {
	if !o.released.Load() {
		return evictionResult{}
	}
	return evictionResult{freedBytes: size, fullyEvicted: true, success: true}
}

func TestCacheQueuedAccountingPreservesAdmissionAndPins(t *testing.T) {
	var manager CacheManager
	manager.Init(128, 128, 0)
	defer manager.Stop()
	object := new(testPinnedAccountingObject)
	published := time.Now()
	manager.AddItem(object, 16, TypeShard, nil, func(any) time.Time { return published }, nil)
	if !manager.canLoadClusterShard(nil) {
		t.Fatal("fixture unexpectedly blocks admission")
	}
	gate := make(chan struct{})
	manager.opChan <- cacheOp{ready: gate}
	for i := 0; i < 100; i++ {
		manager.UpdateSizeAsync(object, 2)
	}
	parked := make(chan struct{})
	manager.opChan <- cacheOp{ready: parked}
	close(gate)
	deadline := time.Now().Add(time.Second)
	for manager.canLoadClusterShard(nil) {
		if time.Now().After(deadline) {
			close(parked)
			t.Fatal("queued growth left admission open while the next owner was still busy")
		}
		runtime.Gosched()
	}
	close(parked)
	usage := manager.Stat()
	if usage.ShardMemory != 216 || manager.canLoadClusterShard(nil) {
		t.Fatalf("queued accounting lost bytes or admitted over budget: %+v", usage)
	}
	if offers := object.offers.Load(); offers > 2 {
		t.Fatalf("one queued batch retried pinned eviction %d times", offers)
	}
	object.released.Store(true)
	manager.SetSize(object, 216)
	if usage := manager.Stat(); usage.ShardMemory != 0 || !manager.canLoadClusterShard(nil) {
		t.Fatalf("released payload was not reclaimed before admission: %+v", usage)
	}
}

func TestCachePublishesBudgetWithoutAdmissionTransition(t *testing.T) {
	var manager CacheManager
	manager.Init(1024, 1024, 0)
	defer manager.Stop()
	before := manager.shardCapacity.Load()
	manager.SetShardRAMPercent(25)
	after := manager.shardCapacity.Load()
	if before == nil || after == nil || !before.Allowed || !after.Allowed || before.Budget != 512 || after.Budget != 256 {
		t.Fatalf("cold placement did not observe the changed budget: before=%+v after=%+v", before, after)
	}
}

func TestCanonicalPrejoinConcurrentInitializer(t *testing.T) {
	source, _ := createDurabilityTestTable(t, "prejoin_init_owner", 0)
	cache, _ := CreateTable(nil, source.schema.Name, ".prejoin:concurrent", Cache, false)
	var builds atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	results := make(chan bool, 12)
	build := func(*scm.SessionState) {
		builds.Add(1)
		close(started)
		<-release
	}
	for range 12 {
		go func() { results <- cache.initializeCache(nil, build) }()
	}
	<-started
	close(release)
	owners := 0
	for range 12 {
		if <-results {
			owners++
		}
	}
	if owners != 1 || builds.Load() != 1 {
		t.Fatalf("canonical initializer ran %d times with %d creators", builds.Load(), owners)
	}
	if cache.initializeCache(nil, func(*scm.SessionState) { t.Fatal("warm cache rebuilt") }) {
		t.Fatal("warm guard reported a new build")
	}
}

func TestCanonicalPrejoinDependencySurvivesRollbackAndEviction(t *testing.T) {
	source, _ := createDurabilityTestTable(t, "prejoin_dependency_owner", 0)
	db := source.schema
	cache, _ := CreateTable(nil, db.Name, ".prejoin:lifetime", Cache, false)
	cache.RuntimeOnly = true
	owner, observer := NewTxContext(TxACID), NewTxContext(TxACID)
	defer owner.Rollback()
	defer observer.Rollback()
	owner.readSchema(db)
	observer.readSchema(db)
	calls := 0
	register := func() {
		scm.Apply(scm.Globalenv.Vars[scm.Symbol("createtrigger")],
			NewTableScmer(source), scm.NewString(".prejoin:lifetime|items|after_insert"), scm.NewString("after_insert"),
			scm.NewString(""), scm.NewString(""), scm.NewFunc(func(...scm.Scmer) scm.Scmer {
				calls++
				return scm.NewBool(true)
			}), scm.NewBool(false), scm.NewAny(owner), NewTableScmer(cache))
	}
	cache.initializeCache(nil, func(*scm.SessionState) { register() })
	if owner.hasSchemaWriter(db) {
		t.Fatal("runtime cache dependency became user declaration DDL")
	}
	owner.Rollback()
	if db.GetTable(cache.Name) != cache || !cache.cacheInitialized {
		t.Fatal("using transaction rollback discarded the canonical cache")
	}
	source.ExecuteTriggers(AfterInsert, nil, dataset{scm.NewInt(1), scm.NewString("committed")}, nil)
	if calls != 1 || cache.cacheTriggerSources.Load() == nil {
		t.Fatal("rollback removed maintenance or its reverse dependency")
	}
	GlobalCache.Stop()
	defer resumeOwnershipTestCache()
	if !keytableCleanup(cache, nil) || len(source.GetTriggers(AfterInsert)) != 0 {
		t.Fatal("eviction left stale maintenance dependencies")
	}
	if cache.acquireCacheUse() {
		t.Fatal("evicted cache remained available")
	}
}

func TestCanonicalPrejoinPrivateMaterialization(t *testing.T) {
	source, _ := createDurabilityTestTable(t, "prejoin_private_owner", 0)
	columns := []scm.Scmer{scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewString("int"), scm.NewString("bin")})}
	producer := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		var workers sync.WaitGroup
		for i := range 64 {
			workers.Add(1)
			go func() {
				defer workers.Done()
				scm.Apply(args[0], scm.NewSlice([]scm.Scmer{scm.NewInt(int64(i))}))
			}()
		}
		workers.Wait()
		return scm.NewBool(true)
	})
	var private *table
	result := materializeQueryTable(nil, source.schema, columns, producer, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		private = TableFromScmer(args[0])
		if source.schema.GetTable(private.Name) != nil {
			t.Fatal("private values entered the shared namespace")
		}
		seen, sum := make(map[int64]bool), int64(0)
		for _, shard := range private.ActiveShards() {
			reader := shard.ColumnReaderTx(nil, "value", false, ReadCompressed)
			for row := uint32(0); row < shard.main_count+uint32(len(shard.deltaRows())); row++ {
				value := reader.GetValue(row).Int()
				if seen[value] || value < 0 || value >= 64 {
					t.Fatalf("materialized row is duplicated or invalid: %d", value)
				}
				seen[value], sum = true, sum+value
			}
			ReleaseColumnReader(reader)
		}
		if len(seen) != 64 {
			t.Fatalf("materialized %d distinct values, want 64", len(seen))
		}
		return scm.NewInt(sum)
	}))
	if result.Int() != 2016 || atomic.LoadInt64(&private.cacheUsers) != 0 {
		t.Fatalf("private materialization returned %s or retained its producer pin", scm.String(result))
	}
	GlobalCache.Stop()
	defer resumeOwnershipTestCache()
	if !keytableCleanup(private, nil) {
		t.Fatal("private table cannot be reclaimed with its index/column children")
	}
}

func TestCanonicalPrejoinDefersUnfinishedSourceDecision(t *testing.T) {
	source, _ := createDurabilityTestTable(t, "prejoin_pending_owner", 1)
	cache, _ := CreateTable(nil, source.schema.Name, ".prejoin:pending", Cache, false)
	cache.RuntimeOnly = true
	owner := NewTxContext(TxACID)
	defer owner.Rollback()
	owner.AddToDeleteMask(source.ActiveShards()[0], 0)
	builder := NewTxContext(TxCursorStability)
	builder.autoCommit = true
	defer builder.Rollback()
	fills := 0
	initializer := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		return scm.Apply(scm.Globalenv.Vars[scm.Symbol("initialize_cache_table")], args[0], NewTableScmer(cache),
			scm.NewSlice([]scm.Scmer{NewTableScmer(source)}),
			scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
			scm.NewFunc(func(...scm.Scmer) scm.Scmer { fills++; return scm.NewBool(true) }),
			scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), scm.NewBool(true))
	})
	cache.OnInit.Store(&initializer)
	prepare := scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		cache.awaitCreationInitialization(scm.NewAny(builder))
		return scm.NewBool(true)
	})
	if trySharedCachePreparation(builder, prepare) || fills != 0 || cache.creationPanic != nil || cache.onInitComplete.Load() {
		t.Fatal("unfinished source decision published or permanently poisoned the cache")
	}
	owner.Rollback()
	if !trySharedCachePreparation(builder, prepare) || fills != 1 || !cache.cacheInitialized {
		t.Fatal("cache did not retry after the source decision completed")
	}
	later := NewTxContext(TxACID)
	defer later.Rollback()
	later.AddToDeleteMask(source.ActiveShards()[0], 0)
	if !trySharedCachePreparation(builder, prepare) || fills != 1 {
		t.Fatal("a later source writer slowed or rebuilt a complete warm cache")
	}
}

func TestCanonicalPrejoinPendingMutationIncludesRetainedTopology(t *testing.T) {
	source, _ := createDurabilityTestTable(t, "prejoin_retained_owner", 0)
	old := source.ActiveShards()[0]
	old.beginTransactionUse()
	defer old.endTransactionUse()
	replacement := NewShard(source)
	source.mu.Lock()
	source.Shards = []*storageShard{replacement}
	source.publishTopologyLocked()
	source.mu.Unlock()
	if source.ActiveShards()[0].activeTransactions.Load() != 0 || source.pendingCacheMutations.Load() != 1 {
		t.Fatal("current-only shard accounting lost a retained generation's pending decision")
	}
}

func TestCanonicalPrejoinRetiresAfterUntrackedOwnershipChange(t *testing.T) {
	source, _ := createDurabilityTestTable(t, "prejoin_remote_retirement", 0)
	makeCache := func(name string) *table {
		scm.Apply(scm.Globalenv.Vars[scm.Symbol("createtable")], scm.NewString(source.schema.Name), scm.NewString(name),
			scm.NewSlice(nil), scm.NewSlice([]scm.Scmer{scm.NewString("engine"), scm.NewString("cache"), scm.NewString("runtime_only"), scm.NewBool(true)}), scm.NewBool(false), scm.NewNil())
		return source.schema.GetTable(name)
	}
	root := makeCache(".prejoin:retirement")
	root.cacheTracksSourceRows.Store(true)
	child := makeCache(".grp:retirement")
	link := func(parent, target *table, name string) {
		parent.AddTrigger(TriggerDescription{Name: name, Timing: AfterInsert, Hidden: true,
			cacheTarget: target, Acquire: target.acquireCacheUseForTrigger, Release: target.releaseCacheUse})
	}
	source.AddTrigger(TriggerDescription{Name: "user_trigger", Timing: AfterInsert})
	link(source, root, ".prejoin:retirement")
	link(root, child, ".cache:retirement")
	if !root.acquireCacheUse() {
		t.Fatal("cannot pin original reader")
	}
	defer func() {
		if atomic.LoadInt64(&root.cacheUsers) > 0 {
			root.releaseCacheUse()
		}
	}()
	invalidateDependentCaches(source, false)
	if root.cacheRowsInvalidated.Load() {
		t.Fatal("ordinary committed maintenance retired shared membership")
	}
	invalidateDependentCaches(source, true)
	GlobalCache.Stat()
	if !root.cacheRowsInvalidated.Load() || source.schema.tables.Get(root.Name) != root {
		t.Fatal("invalid membership was reusable or its active reader was removed")
	}
	if source.schema.tables.Get(child.Name) != nil {
		t.Fatal("native owner did not retire dependent cache first")
	}
	if trySharedCachePreparation(nil, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		root.awaitCreationInitialization(scm.NewNil())
		return scm.NewBool(true)
	})) {
		t.Fatal("invalidated membership was admitted for another query")
	}
	late := makeCache(".grp:retirement-late")
	link(root, late, ".cache:late-consumer")
	GlobalCache.Stat()
	if source.schema.tables.Get(late.Name) != nil {
		t.Fatal("late registration escaped native retirement")
	}
	root.releaseCacheUse()
	GlobalCache.Stat()
	if source.schema.tables.Get(root.Name) != nil {
		t.Fatal("native owner retained an unpinned invalid cache")
	}
	remaining := source.GetTriggers(AfterInsert)
	if len(remaining) != 1 || remaining[0].Name != "user_trigger" {
		t.Fatal("retirement removed a user trigger or retained a cache edge")
	}
	fresh := makeCache(root.Name)
	if fresh == root || fresh.cacheRowsInvalidated.Load() {
		t.Fatal("canonical name retained its retired lifetime")
	}
	builds := 0
	for i := 0; i < 2; i++ {
		fresh.initializeCache(nil, func(*scm.SessionState) { builds++ })
	}
	if builds != 1 {
		t.Fatal("recreated cache was not reusable across invocations")
	}
}

func TestCanonicalPrejoinRejectsSourceChangeDuringColdFill(t *testing.T) {
	source, _ := createDurabilityTestTable(t, "prejoin_fill_revalidation", 0)
	cache, _ := CreateTable(nil, source.schema.Name, ".prejoin:fill-revalidation", Cache, false)
	cache.RuntimeOnly = true
	prepare := scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		return scm.Apply(scm.Globalenv.Vars[scm.Symbol("initialize_cache_table")], scm.NewNil(), NewTableScmer(cache),
			scm.NewSlice([]scm.Scmer{NewTableScmer(source)}),
			scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
			scm.NewFunc(func(...scm.Scmer) scm.Scmer {
				source.beginContributionMutation()
				source.endContributionMutation()
				return scm.NewBool(true)
			}),
			scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), scm.NewBool(true))
	})
	if trySharedCachePreparation(nil, prepare) || !cache.cacheRowsInvalidated.Load() || cache.cacheInitialized {
		t.Fatal("source changes during fill certified a reusable shared membership")
	}
	GlobalCache.Stat()
}
