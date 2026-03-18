/*
Copyright (C) 2025-2026  MemCP Contributors

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
	"container/heap"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	units "github.com/docker/go-units"
	"github.com/launix-de/memcp/scm"
)

// EvictableType identifies the kind of cached object for factor lookup and stat reporting.
type EvictableType uint8

const (
	TypeTempColumn    EvictableType = iota // weight 20 — cheap to recompute
	TypeShard                              // weight 1  — expensive (disk I/O)
	TypeIndex                              // weight 20 — cheap (rebuild from shard data)
	TypeTempKeytable                       // weight 2  — medium (join intermediate)
	TypeCacheEntry                         // weight 20 — cheap (reload from disk)
	TypeStringDict                         // weight 20 — cheap (decompression)
	numEvictableTypes                      // sentinel for array sizing
)

// evictableWeights maps EvictableType → eviction weight.
// Higher weight = higher evictionScore = evicted sooner.
// Low weight = more protected (expensive to rebuild).
// Weights are exact inverses of the old factors (20/factor) to preserve behavior.
//                                         TempCol Shard Index TempKT CacheEntry StringDict
var evictableWeights = [numEvictableTypes]int64{20, 1, 20, 2, 20, 20}

var evictableNames = [numEvictableTypes]string{"TempColumn", "Shard", "Index", "TempKeytable", "CacheEntry", "StringDict"}

type softItem struct {
	pointer       any
	size          int64
	evictType     EvictableType
	evictionScore int64 // = size * weight (static, max-heap key); higher = evicted sooner
	cleanup       func(pointer any, freedByType *[numEvictableTypes]int64) bool
	getLastUsed   func(pointer any) time.Time
	getScore      func(pointer any) float64 // optional type-specific telemetry
	heapIndex     int                       // position in heap (-1 if not in heap)
	dynamicScore  float64                   // scratch field for Phase 2
	registeredAt  int64                     // UnixNano; set once in addInternal as fallback for items whose lastAccessed starts at zero
	minLifetime   int64                     // minimum idle nanos before eviction (0 = default 1s)
	maxIdleTime   int64                     // force-evict if idle for this many nanos (0 = no limit)
}

// expiryEntry is a lazy min-heap entry tracking when an item may expire.
// Lazy: stale entries (where the item has been re-used or removed) are
// discarded on pop rather than eagerly updated — O(log n) per expiry event.
type expiryEntry struct {
	estimatedExpiry int64 // unix nanos: when we expect this item to expire
	pointer         any   // key into itemMap
}

// expiryHeap is a min-heap on estimatedExpiry (soonest expiry at top).
type expiryHeap []expiryEntry

func (h expiryHeap) Len() int           { return len(h) }
func (h expiryHeap) Less(i, j int) bool { return h[i].estimatedExpiry < h[j].estimatedExpiry }
func (h expiryHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *expiryHeap) Push(x any)        { *h = append(*h, x.(expiryEntry)) }
func (h *expiryHeap) Pop() any          { old := *h; n := len(old); x := old[n-1]; *h = old[:n-1]; return x }

// softItemHeap implements container/heap.Interface as a max-heap on evictionScore.
type softItemHeap []*softItem

func (h softItemHeap) Len() int { return len(h) }
func (h softItemHeap) Less(i, j int) bool {
	return h[i].evictionScore > h[j].evictionScore // max-heap: highest score on top
}
func (h softItemHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].heapIndex = i
	h[j].heapIndex = j
}
func (h *softItemHeap) Push(x any) {
	item := x.(*softItem)
	item.heapIndex = len(*h)
	*h = append(*h, item)
}
func (h *softItemHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	item.heapIndex = -1
	*h = old[:n-1]
	return item
}

// isPersistedType returns true for types representing persisted (disk-reloadable) data.
func isPersistedType(t EvictableType) bool {
	return t == TypeShard || t == TypeIndex
}

// systemFreeThreshold is the minimum fraction of total RAM that must remain free
// system-wide before the cache triggers eviction (regardless of our own budget).
const systemFreeThreshold = 10 // percent

// systemPressureCheckInterval is the minimum time between /proc/meminfo reads.
const systemPressureCheckInterval = time.Second

// systemMemInfo returns (freeBytes, totalBytes) of physical RAM.
// freeBytes is MemAvailable from /proc/meminfo — includes page cache and reclaimable
// memory, which is the correct metric for "how much RAM can we use before the OS
// starts swapping". Falls back to syscall.Sysinfo.Freeram if /proc/meminfo is unavailable.
// Returns (0, 0) on error.
func systemMemInfo() (free, total int64) {
	if f, err := os.Open("/proc/meminfo"); err == nil {
		defer f.Close()
		var available, totalkb int64
		buf := make([]byte, 4096)
		n, _ := f.Read(buf)
		for _, line := range strings.SplitN(string(buf[:n]), "\n", 64) {
			var kb int64
			if strings.HasPrefix(line, "MemTotal:") {
				fmt.Sscanf(strings.TrimPrefix(line, "MemTotal:"), "%d", &kb)
				totalkb = kb
			} else if strings.HasPrefix(line, "MemAvailable:") {
				fmt.Sscanf(strings.TrimPrefix(line, "MemAvailable:"), "%d", &kb)
				available = kb
			}
			if totalkb > 0 && available > 0 {
				return available * 1024, totalkb * 1024
			}
		}
	}
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return 0, 0
	}
	unit := int64(info.Unit)
	return int64(info.Freeram) * unit, int64(info.Totalram) * unit
}

// CacheManager manages memory-limited soft references with two-phase eviction.
// Two budgets: persistedBudget (shards+indexes) and memoryBudget (total).
type CacheManager struct {
	memoryBudget    int64 // total budget (default 50% of RAM)
	persistedBudget int64 // budget for persisted shards+indexes (default 30% of RAM)
	currentMemory   int64

	lastSysCheck time.Time // last time /proc/meminfo was read for pressure check

	sizeByType  [numEvictableTypes]int64
	countByType [numEvictableTypes]int64

	h       softItemHeap
	expH    expiryHeap // min-heap for maxIdleTime expiry; lazy-deletion (stale entries discarded on pop)
	itemMap map[any]*softItem

	opChan  chan cacheOp
	stopped atomic.Bool
}

type cacheOp struct {
	add                *softItem
	del                any
	updatePtr          any
	updateDelta        int64
	budgetUpdate       bool
	budgetVal          int64
	persistedBudgetVal int64
	pressureSize       int64
	statResult         chan CacheStat
	done               chan struct{}
}

// CacheStat holds stat results returned via channel.
type CacheStat struct {
	SizeByType      [numEvictableTypes]int64
	CountByType     [numEvictableTypes]int64
	CurrentMemory   int64
	MemoryBudget    int64
	PersistedBudget int64
	PersistedMemory int64
}

// Init initializes the CacheManager with the given budgets and starts the background goroutine.
// Calling Init on an already-initialized CacheManager is a no-op.
func (cm *CacheManager) Init(memoryBudget, persistedBudget int64) {
	if cm.opChan != nil {
		return // already initialized
	}
	cm.memoryBudget = memoryBudget
	cm.persistedBudget = persistedBudget
	cm.itemMap = make(map[any]*softItem)
	cm.opChan = make(chan cacheOp, 1024)
	heap.Init(&cm.h)
	go cm.run()
}

// Stop signals the CacheManager goroutine to drain remaining ops and exit.
// After Stop returns, the run() goroutine has finished.
func (cm *CacheManager) Stop() {
	if cm.opChan == nil {
		return
	}
	if cm.stopped.Swap(true) {
		return // already stopped
	}
	close(cm.opChan)
}

// AddItem registers an evictable item. Triggers cleanup if over budget.
// No-op if the CacheManager is not initialized.
func (cm *CacheManager) AddItem(
	pointer any,
	size int64,
	evictType EvictableType,
	cleanup func(pointer any, freedByType *[numEvictableTypes]int64) bool,
	getLastUsed func(pointer any) time.Time,
	getScore func(pointer any) float64,
) {
	if cm.opChan == nil {
		return
	}
	if cm.stopped.Load() {
		return
	}
	weight := evictableWeights[evictType]
	item := &softItem{
		pointer:       pointer,
		size:          size,
		evictType:     evictType,
		evictionScore: size * weight,
		cleanup:       cleanup,
		getLastUsed:   getLastUsed,
		getScore:      getScore,
		heapIndex:     -1,
	}
	done := make(chan struct{})
	cm.opChan <- cacheOp{add: item, done: done}
	<-done
}

// AddItemEx is like AddItem but allows specifying per-item lifetimes.
// minLifetime: minimum idle duration before this item is eligible for eviction (0 = default 1s).
// maxIdleTime: force-evict the item if it has been idle for longer than this (0 = no limit).
func (cm *CacheManager) AddItemEx(
	pointer any,
	size int64,
	evictType EvictableType,
	cleanup func(pointer any, freedByType *[numEvictableTypes]int64) bool,
	getLastUsed func(pointer any) time.Time,
	getScore func(pointer any) float64,
	minLifetime, maxIdleTime time.Duration,
) {
	if cm.opChan == nil || cm.stopped.Load() {
		return
	}
	weight := evictableWeights[evictType]
	item := &softItem{
		pointer:       pointer,
		size:          size,
		evictType:     evictType,
		evictionScore: size * weight,
		cleanup:       cleanup,
		getLastUsed:   getLastUsed,
		getScore:      getScore,
		heapIndex:     -1,
		minLifetime:   int64(minLifetime),
		maxIdleTime:   int64(maxIdleTime),
	}
	done := make(chan struct{})
	cm.opChan <- cacheOp{add: item, done: done}
	<-done
}

// Remove deregisters an item WITHOUT calling cleanup.
// For normal lifecycle destruction (DropTable, DropColumn, Rebuild).
// Safe to call for pointers not in the map (no-op).
func (cm *CacheManager) Remove(pointer any) {
	if cm.opChan == nil {
		return
	}
	if cm.stopped.Load() {
		return
	}
	done := make(chan struct{})
	cm.opChan <- cacheOp{del: pointer, done: done}
	<-done
}

// UpdateSize adjusts the tracked size by delta. Recomputes evictionScore and fixes heap.
func (cm *CacheManager) UpdateSize(pointer any, delta int64) {
	if cm.opChan == nil {
		return
	}
	if cm.stopped.Load() {
		return
	}
	done := make(chan struct{})
	cm.opChan <- cacheOp{updatePtr: pointer, updateDelta: delta, done: done}
	<-done
}

// UpdateBudget changes both memory budgets (e.g. when MaxRamPercent or MaxPersistPercent changes).
func (cm *CacheManager) UpdateBudget(totalBudget, persistedBudget int64) {
	if cm.opChan == nil {
		return
	}
	if cm.stopped.Load() {
		return
	}
	done := make(chan struct{})
	cm.opChan <- cacheOp{budgetUpdate: true, budgetVal: totalBudget, persistedBudgetVal: persistedBudget, done: done}
	<-done
}

// CheckPressure proactively triggers eviction if currentMemory + additionalSize exceeds the budget.
// Use this before large allocations to free space ahead of time.
func (cm *CacheManager) CheckPressure(additionalSize int64) {
	if cm.opChan == nil {
		return
	}
	if cm.stopped.Load() {
		return
	}
	done := make(chan struct{})
	cm.opChan <- cacheOp{pressureSize: additionalSize, done: done}
	<-done
}

// Stat returns per-type evictable sizes and counts.
func (cm *CacheManager) Stat() CacheStat {
	if cm.opChan == nil {
		return CacheStat{}
	}
	if cm.stopped.Load() {
		return CacheStat{}
	}
	ch := make(chan CacheStat, 1)
	cm.opChan <- cacheOp{statResult: ch}
	return <-ch
}

// persistedMemory returns the sum of persisted (disk-reloadable) tracked memory.
func (cm *CacheManager) persistedMemory() int64 {
	return cm.sizeByType[TypeShard] + cm.sizeByType[TypeIndex]
}

// runEvictionChecks checks both persisted and total budgets and evicts as needed.
func (cm *CacheManager) runEvictionChecks(additionalSize int64) {
	// Tier 1: persisted budget (shards + indexes only)
	if cm.persistedBudget > 0 {
		cm.evict(cm.persistedMemory(), cm.persistedBudget, additionalSize, isPersistedType)
	}

	// Tier 2+3: total budget merged with system-pressure budget — single evict pass.
	// Use -1 as sentinel for "no constraint"; >=0 means evict to that byte limit.
	effectiveBudget := int64(-1)
	if cm.memoryBudget > 0 {
		effectiveBudget = cm.memoryBudget
	}

	// Check system-wide free RAM (throttled to once per second).
	now := time.Now()
	if now.Sub(cm.lastSysCheck) >= systemPressureCheckInterval {
		cm.lastSysCheck = now
		free, total := systemMemInfo()
		if total > 0 && free*100 < int64(systemFreeThreshold)*total {
			// How much we'd need to release for the system to reach the threshold.
			needed := total*int64(systemFreeThreshold)/100 - free
			sysBudget := cm.currentMemory - needed
			if sysBudget < 0 {
				sysBudget = 0
			}
			// Use the more restrictive of both budgets.
			if effectiveBudget < 0 || sysBudget < effectiveBudget {
				effectiveBudget = sysBudget
			}
		}
	}

	if effectiveBudget >= 0 {
		cm.evict(cm.currentMemory, effectiveBudget, additionalSize, nil)
	}
}

// run is the single-threaded goroutine handling all operations.
func (cm *CacheManager) run() {
	expireTicker := time.NewTicker(time.Minute)
	defer expireTicker.Stop()
	for {
		select {
		case op, ok := <-cm.opChan:
			if !ok {
				return
			}
			if op.add != nil {
				cm.addInternal(op.add)
			} else if op.del != nil {
				cm.removeByPointer(op.del)
			} else if op.updatePtr != nil {
				cm.updateSizeInternal(op.updatePtr, op.updateDelta)
			} else if op.budgetUpdate {
				cm.memoryBudget = op.budgetVal
				cm.persistedBudget = op.persistedBudgetVal
			} else if op.pressureSize > 0 {
				cm.runEvictionChecks(op.pressureSize)
			} else if op.statResult != nil {
				op.statResult <- CacheStat{
					SizeByType:      cm.sizeByType,
					CountByType:     cm.countByType,
					CurrentMemory:   cm.currentMemory,
					MemoryBudget:    cm.memoryBudget,
					PersistedBudget: cm.persistedBudget,
					PersistedMemory: cm.persistedMemory(),
				}
				close(op.statResult)
			}
			if op.done != nil {
				close(op.done)
			}
			// check if we need cleanup after add or updateSize
			cm.runEvictionChecks(0)
		case <-expireTicker.C:
			cm.evictExpired()
		}
	}
}

// evictExpired removes items whose maxIdleTime has been exceeded.
// Uses a lazy min-heap (expH): stale entries where the item has been re-used
// or already removed are discarded on pop — O(k log n) total, no O(n) scan.
func (cm *CacheManager) evictExpired() {
	nowNano := time.Now().UnixNano()
	var freedByType [numEvictableTypes]int64
	for cm.expH.Len() > 0 && cm.expH[0].estimatedExpiry <= nowNano {
		entry := heap.Pop(&cm.expH).(expiryEntry)
		item, ok := cm.itemMap[entry.pointer]
		if !ok {
			continue // lazily discard: item already removed
		}
		// Recheck actual idle time — item may have been used since we estimated.
		lastActive := item.registeredAt
		if lu := item.getLastUsed(item.pointer).UnixNano(); lu > lastActive {
			lastActive = lu
		}
		actualExpiry := lastActive + item.maxIdleTime
		if actualExpiry > nowNano {
			// Item was used after our estimate; push back with the updated deadline.
			heap.Push(&cm.expH, expiryEntry{actualExpiry, entry.pointer})
			continue
		}
		if item.cleanup(item.pointer, &freedByType) {
			cm.removeInternal(item.pointer, &freedByType)
		}
		// If cleanup failed (lock contention), we don't retry — item will linger
		// until the next ticker tick, which is acceptable.
	}
}

// addInternal inserts a new softItem.
func (cm *CacheManager) addInternal(item *softItem) {
	if old, ok := cm.itemMap[item.pointer]; ok {
		// re-registration: update in place
		delta := item.size - old.size
		cm.currentMemory += delta
		scm.AdjustMemStats(delta)
		cm.sizeByType[old.evictType] -= old.size
		cm.countByType[old.evictType]--
		cm.sizeByType[item.evictType] += item.size
		cm.countByType[item.evictType]++
		// copy heap position
		item.heapIndex = old.heapIndex
		if item.heapIndex >= 0 {
			cm.h[item.heapIndex] = item
			heap.Fix(&cm.h, item.heapIndex)
		}
		cm.itemMap[item.pointer] = item
		return
	}
	item.registeredAt = time.Now().UnixNano()
	cm.itemMap[item.pointer] = item
	cm.currentMemory += item.size
	scm.AdjustMemStats(item.size)
	cm.sizeByType[item.evictType] += item.size
	cm.countByType[item.evictType]++
	heap.Push(&cm.h, item)
	if item.maxIdleTime > 0 {
		heap.Push(&cm.expH, expiryEntry{item.registeredAt + item.maxIdleTime, item.pointer})
	}
}

// removeByPointer removes an item without calling cleanup.
func (cm *CacheManager) removeByPointer(pointer any) {
	cm.removeInternal(pointer, nil)
}

// removeInternal removes an item from bookkeeping. No cleanup call. Accepts freedByType for recursive accounting.
func (cm *CacheManager) removeInternal(pointer any, freedByType *[numEvictableTypes]int64) {
	item, ok := cm.itemMap[pointer]
	if !ok {
		return
	}
	cm.currentMemory -= item.size
	scm.AdjustMemStats(-item.size)
	cm.sizeByType[item.evictType] -= item.size
	cm.countByType[item.evictType]--
	if freedByType != nil {
		freedByType[item.evictType] += item.size
	}
	if item.heapIndex >= 0 {
		heap.Remove(&cm.h, item.heapIndex)
	}
	delete(cm.itemMap, pointer)
}

// updateSizeInternal adjusts size and recomputes heap position.
func (cm *CacheManager) updateSizeInternal(pointer any, delta int64) {
	item, ok := cm.itemMap[pointer]
	if !ok {
		return
	}
	cm.currentMemory += delta
	scm.AdjustMemStats(delta)
	cm.sizeByType[item.evictType] += delta
	item.size += delta
	item.evictionScore = item.size * evictableWeights[item.evictType]
	if item.heapIndex >= 0 {
		heap.Fix(&cm.h, item.heapIndex)
	}
}

const telemetryWeight = 1000.0 // weight for telemetry score vs LRU age in seconds

// evict runs two-phase eviction to bring currentUsage below budget.
// additionalSize accounts for an upcoming allocation that hasn't been tracked yet.
// typeFilter restricts which types are eviction candidates (nil = all types).
func (cm *CacheManager) evict(currentUsage, budget, additionalSize int64, typeFilter func(EvictableType) bool) {
	if currentUsage+additionalSize <= budget {
		return
	}
	needToFree := currentUsage + additionalSize - budget
	freeTarget := needToFree + budget*25/100
	candidateTarget := freeTarget * 2

	// Phase 1: pull candidates from max-heap (largest evictionScore first)
	var candidates []*softItem
	var skipped []*softItem
	var candidateSum int64
	for candidateSum < candidateTarget && cm.h.Len() > 0 {
		item := heap.Pop(&cm.h).(*softItem)
		if typeFilter != nil && !typeFilter(item.evictType) {
			skipped = append(skipped, item)
			continue
		}
		candidates = append(candidates, item)
		candidateSum += item.size
	}
	// push back items that didn't match the type filter
	for _, s := range skipped {
		heap.Push(&cm.h, s)
	}

	// Phase 2: remove candidates already freed by recursive side effects
	alive := candidates[:0]
	for _, c := range candidates {
		if _, ok := cm.itemMap[c.pointer]; ok {
			alive = append(alive, c)
		}
	}
	candidates = alive

	// score dynamically: age * evictionScore, reduced by telemetry
	// high dynamicScore = old + large + unimportant → evict first
	now := time.Now()
	for _, c := range candidates {
		age := now.Sub(c.getLastUsed(c.pointer)).Seconds()
		telemetry := 0.0
		if c.getScore != nil {
			telemetry = c.getScore(c.pointer)
		}
		c.dynamicScore = age*float64(c.evictionScore) - telemetry*telemetryWeight
	}

	// sort by dynamicScore (worst first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].dynamicScore > candidates[j].dynamicScore
	})

	// evict all candidates, oldest first; stop once we are within budget
	var freedByType [numEvictableTypes]int64
	var totalFreed int64
	for i := 0; i < len(candidates); i++ {
		if cm.currentMemory <= budget {
			// already within budget – push remaining survivors back
			for ; i < len(candidates); i++ {
				if _, ok := cm.itemMap[candidates[i].pointer]; ok {
					heap.Push(&cm.h, candidates[i])
				}
			}
			break
		}
		c := candidates[i]
		// check again — previous cleanup in this loop may have freed this item recursively
		if _, ok := cm.itemMap[c.pointer]; !ok {
			continue
		}
		// guarantee minimum lifetime before eviction: use the later of registeredAt and getLastUsed
		lastActive := c.registeredAt
		if lu := c.getLastUsed(c.pointer).UnixNano(); lu > lastActive {
			lastActive = lu
		}
		minLT := c.minLifetime
		if minLT == 0 {
			minLT = int64(time.Second)
		}
		idleNanos := now.UnixNano() - lastActive
		if idleNanos < minLT {
			heap.Push(&cm.h, c)
			continue
		}
		if !c.cleanup(c.pointer, &freedByType) {
			// cleanup couldn't acquire lock; push back for later retry
			heap.Push(&cm.h, c)
			continue
		}
		// removeInternal handles bookkeeping + recursive accounting
		cm.removeInternal(c.pointer, &freedByType)
		totalFreed += c.size
	}

	// survivors already pushed back inside the loop above (early-exit branch)

	// log summary
	if totalFreed > 0 {
		shardColsOnly := freedByType[TypeShard] - freedByType[TypeIndex]
		if shardColsOnly < 0 {
			shardColsOnly = 0
		}
		log.Printf("memory pressure: freed %s total (%s temp columns, %s shard columns, %s indexes, %s keytables, %s cache entries, %s string dicts)",
			units.BytesSize(float64(totalFreed)),
			units.BytesSize(float64(freedByType[TypeTempColumn])),
			units.BytesSize(float64(shardColsOnly)),
			units.BytesSize(float64(freedByType[TypeIndex])),
			units.BytesSize(float64(freedByType[TypeTempKeytable])),
			units.BytesSize(float64(freedByType[TypeCacheEntry])),
			units.BytesSize(float64(freedByType[TypeStringDict])),
		)
	}
}

// FormatStat returns a human-readable string of the cache state.
func (cs CacheStat) FormatStat() string {
	shardColsOnly := cs.SizeByType[TypeShard] - cs.SizeByType[TypeIndex]
	if shardColsOnly < 0 {
		shardColsOnly = 0
	}
	var totalEvictable int64
	for i := range cs.SizeByType {
		totalEvictable += cs.SizeByType[i]
	}
	// subtract index double-count for display
	totalEvictable -= cs.SizeByType[TypeIndex]

	var b strings.Builder
	b.WriteString(fmt.Sprintf("TotalBudget = %s\tPersistedBudget = %s\tTracked = %s\tPersisted = %s\n",
		units.BytesSize(float64(cs.MemoryBudget)),
		units.BytesSize(float64(cs.PersistedBudget)),
		units.BytesSize(float64(cs.CurrentMemory)),
		units.BytesSize(float64(cs.PersistedMemory))))
	b.WriteString("Type                     \tCount\tSize\n")
	b.WriteString(fmt.Sprintf("%-25s\t%d\t%s\n", "Temp columns", cs.CountByType[TypeTempColumn], units.BytesSize(float64(cs.SizeByType[TypeTempColumn]))))
	b.WriteString(fmt.Sprintf("%-25s\t%d\t%s\n", "Shard columns", cs.CountByType[TypeShard], units.BytesSize(float64(shardColsOnly))))
	b.WriteString(fmt.Sprintf("%-25s\t%d\t%s\n", "Indexes", cs.CountByType[TypeIndex], units.BytesSize(float64(cs.SizeByType[TypeIndex]))))
	b.WriteString(fmt.Sprintf("%-25s\t%d\t%s\n", "Temp keytables", cs.CountByType[TypeTempKeytable], units.BytesSize(float64(cs.SizeByType[TypeTempKeytable]))))
	b.WriteString(fmt.Sprintf("%-25s\t%d\t%s\n", "Cache entries", cs.CountByType[TypeCacheEntry], units.BytesSize(float64(cs.SizeByType[TypeCacheEntry]))))
	b.WriteString(fmt.Sprintf("%-25s\t%d\t%s\n", "String dicts (lz4)", cs.CountByType[TypeStringDict], units.BytesSize(float64(cs.SizeByType[TypeStringDict]))))
	return b.String()
}
