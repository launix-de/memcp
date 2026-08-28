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
	"github.com/carli2/hybridsort"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	units "github.com/docker/go-units"
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
// evictionScore = size * weight (multiplication, not division).
// Higher weight = higher evictionScore = evicted sooner.
// Low weight = more protected (expensive to rebuild).
//
// Multiplication is used instead of division because:
//   - Integer arithmetic (no floats)
//   - Allows fine-grained upward scaling (e.g. LIKE skip lists can use weight 100+)
//   - Division cannot distinguish between "slightly less important" items
//
// Phase 1 (heap) pulls candidates by evictionScore (max-heap).
// Phase 2 sorts candidates by dynamicScore = age * evictionScore - telemetry*1000.
// Callers that need a minimum lifetime set it explicitly. Query-local cache
// consumers should instead retain a Go reference for the duration of their use.
//
//	TempCol Shard Index TempKT CacheEntry StringDict
var evictableWeights = [numEvictableTypes]int64{20, 1, 20, 2, 20, 20}

var evictableNames = [numEvictableTypes]string{"TempColumn", "Shard", "Index", "TempKeytable", "CacheEntry", "StringDict"}

type softItem struct {
	pointer         any
	size            int64
	evictType       EvictableType
	evictionScore   int64 // = size * weight (static, max-heap key); higher = evicted sooner
	object          cacheObject
	getLastUsed     func(pointer any) time.Time
	getScore        func(pointer any) float64 // optional type-specific telemetry
	heapIndex       int                       // position in heap (-1 if not in heap)
	expiryIndex     int                       // position in expiryHeap (-1 if no expiry is registered)
	estimatedExpiry int64                     // UnixNano; refreshed lazily when the deadline is reached
	dynamicScore    float64                   // scratch field for Phase 2
	registeredAt    int64                     // UnixNano; set once in addInternal as fallback for items whose lastAccessed starts at zero
	minLifetime     int64                     // minimum idle nanos before eviction (0 = none)
	maxIdleTime     int64                     // force-evict if idle for this many nanos (0 = no limit)
}

type evictionMode uint8

const (
	evictPartial evictionMode = iota
	evictFull
)

// evictionOffer is a revalidated recommendation, not a promise. Concurrent
// users may change an object between offer collection and execution.
type evictionOffer struct {
	partialBytes int64
	fullBytes    int64
}

type evictionResult struct {
	freedBytes   int64
	fullyEvicted bool
	success      bool
}

// cacheObject lets one top-level registration own and selectively shed its
// internal caches. Implementations must never call public CacheManager methods:
// eviction runs on the manager's single-owner goroutine.
type cacheObject interface {
	evictionOffer(currentSize int64) evictionOffer
	evict(mode evictionMode, currentSize int64, freedByType *[numEvictableTypes]int64) evictionResult
}

// atomicCacheObject adapts existing all-or-nothing cleanup callbacks to the
// same protocol. For atomic objects the partial and full alternatives coincide.
type atomicCacheObject struct {
	pointer any
	cleanup func(pointer any, freedByType *[numEvictableTypes]int64) bool
}

func (o atomicCacheObject) evictionOffer(currentSize int64) evictionOffer {
	return evictionOffer{partialBytes: currentSize, fullBytes: currentSize}
}

func (o atomicCacheObject) evict(_ evictionMode, currentSize int64, freedByType *[numEvictableTypes]int64) evictionResult {
	if !o.cleanup(o.pointer, freedByType) {
		return evictionResult{}
	}
	return evictionResult{freedBytes: currentSize, fullyEvicted: true, success: true}
}

// expiryHeap is a min-heap on estimatedExpiry (soonest expiry at top).
// It stores the same softItem as the main heap so removal can eagerly unlink
// expiry metadata and release the complete object graph immediately.
type expiryHeap []*softItem

func (h expiryHeap) Len() int           { return len(h) }
func (h expiryHeap) Less(i, j int) bool { return h[i].estimatedExpiry < h[j].estimatedExpiry }
func (h expiryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].expiryIndex = i
	h[j].expiryIndex = j
}
func (h *expiryHeap) Push(x any) {
	item := x.(*softItem)
	item.expiryIndex = len(*h)
	*h = append(*h, item)
}
func (h *expiryHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.expiryIndex = -1
	*h = old[:n-1]
	return item
}

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
	// All metadata below is owned by run(). Query workers only publish lifecycle
	// messages; cache-object children never become global registrations.
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
	runDone chan struct{}
	stopped atomic.Bool
}

type cacheOp struct {
	add                *softItem
	del                any
	updatePtr          any
	updateDelta        int64
	setSize            bool
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
	cm.runDone = make(chan struct{})
	heap.Init(&cm.h)
	heap.Init(&cm.expH)
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
	<-cm.runDone
}

func newSoftItem(pointer any, size int64, evictType EvictableType, cleanup func(any, *[numEvictableTypes]int64) bool, getLastUsed func(any) time.Time, getScore func(any) float64, minLifetime, maxIdleTime time.Duration) *softItem {
	object := cacheObject(atomicCacheObject{pointer: pointer, cleanup: cleanup})
	if topLevel, ok := pointer.(cacheObject); ok {
		object = topLevel
	}
	return &softItem{
		pointer:       pointer,
		size:          size,
		evictType:     evictType,
		evictionScore: size * evictableWeights[evictType],
		object:        object,
		getLastUsed:   getLastUsed,
		getScore:      getScore,
		heapIndex:     -1,
		expiryIndex:   -1,
		minLifetime:   int64(minLifetime),
		maxIdleTime:   int64(maxIdleTime),
	}
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
	item := newSoftItem(pointer, size, evictType, cleanup, getLastUsed, getScore, time.Second, 0)
	done := make(chan struct{})
	cm.opChan <- cacheOp{add: item, done: done}
	<-done
}

// AddItemEx is like AddItem but allows specifying per-item lifetimes.
// minLifetime: minimum idle duration before this item is eligible for eviction (0 = none).
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
	item := newSoftItem(pointer, size, evictType, cleanup, getLastUsed, getScore, minLifetime, maxIdleTime)
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

// SetSize replaces the accounted size of pointer. Use it after recomputing an
// owner's complete footprint; unlike UpdateSize, repeated calls with the same
// observation are idempotent.
func (cm *CacheManager) SetSize(pointer any, size int64) {
	if cm.opChan == nil || cm.stopped.Load() {
		return
	}
	done := make(chan struct{})
	cm.opChan <- cacheOp{updatePtr: pointer, updateDelta: size, setSize: true, done: done}
	<-done
}

// UpdateSizeAsync queues an ownership-local size change without waiting for
// eviction. It is intended for callers that already hold storage locks. Such
// objects serialize enqueue order with their own lock; the single manager then
// applies the deltas in channel order and runs the normal pressure check.
func (cm *CacheManager) UpdateSizeAsync(pointer any, delta int64) {
	if delta == 0 || cm.opChan == nil || cm.stopped.Load() {
		return
	}
	cm.opChan <- cacheOp{updatePtr: pointer, updateDelta: delta}
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
	defer close(cm.runDone)
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
				if op.setSize {
					cm.setSizeInternal(op.updatePtr, op.updateDelta)
				} else {
					cm.updateSizeInternal(op.updatePtr, op.updateDelta)
				}
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

// evictExpired removes items whose maxIdleTime has been exceeded. Cache hits
// only update object-local timestamps; when a deadline is reached we either
// evict or reinsert the item at its refreshed deadline.
func (cm *CacheManager) evictExpired() {
	nowNano := time.Now().UnixNano()
	var freedByType [numEvictableTypes]int64
	for cm.expH.Len() > 0 && cm.expH[0].estimatedExpiry <= nowNano {
		item := heap.Pop(&cm.expH).(*softItem)
		// Recheck actual idle time — item may have been used since we estimated.
		lastActive := item.registeredAt
		if lu := item.getLastUsed(item.pointer).UnixNano(); lu > lastActive {
			lastActive = lu
		}
		actualExpiry := lastActive + item.maxIdleTime
		if actualExpiry > nowNano {
			item.estimatedExpiry = actualExpiry
			heap.Push(&cm.expH, item)
			continue
		}
		result := item.object.evict(evictFull, item.size, &freedByType)
		if result.success && result.fullyEvicted {
			cm.removeInternal(item.pointer, &freedByType)
		} else {
			// Lock contention means the item is currently in use. Keep its expiry
			// tracked so the next ticker can retry instead of leaking it forever.
			item.estimatedExpiry = nowNano + int64(time.Minute)
			heap.Push(&cm.expH, item)
		}
	}
}

// addInternal inserts a new softItem.
func (cm *CacheManager) addInternal(item *softItem) {
	item.registeredAt = time.Now().UnixNano()
	if old, ok := cm.itemMap[item.pointer]; ok {
		// re-registration: update in place
		delta := item.size - old.size
		cm.currentMemory += delta
		cm.sizeByType[old.evictType] -= old.size
		cm.countByType[old.evictType]--
		cm.sizeByType[item.evictType] += item.size
		cm.countByType[item.evictType]++
		// Replace both heap nodes so neither heap retains the old registration.
		item.heapIndex = old.heapIndex
		if item.heapIndex >= 0 {
			cm.h[item.heapIndex] = item
			heap.Fix(&cm.h, item.heapIndex)
		}
		if old.expiryIndex >= 0 {
			heap.Remove(&cm.expH, old.expiryIndex)
		}
		if item.maxIdleTime > 0 {
			item.estimatedExpiry = time.Now().UnixNano() + item.maxIdleTime
			heap.Push(&cm.expH, item)
		}
		cm.itemMap[item.pointer] = item
		return
	}
	cm.itemMap[item.pointer] = item
	cm.currentMemory += item.size
	cm.sizeByType[item.evictType] += item.size
	cm.countByType[item.evictType]++
	heap.Push(&cm.h, item)
	if item.maxIdleTime > 0 {
		item.estimatedExpiry = item.registeredAt + item.maxIdleTime
		heap.Push(&cm.expH, item)
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
	cm.sizeByType[item.evictType] -= item.size
	cm.countByType[item.evictType]--
	if freedByType != nil {
		freedByType[item.evictType] += item.size
	}
	if item.heapIndex >= 0 {
		heap.Remove(&cm.h, item.heapIndex)
	}
	if item.expiryIndex >= 0 {
		heap.Remove(&cm.expH, item.expiryIndex)
	}
	delete(cm.itemMap, pointer)
}

// updateSizeInternal adjusts size and recomputes heap position.
func (cm *CacheManager) updateSizeInternal(pointer any, delta int64) {
	item, ok := cm.itemMap[pointer]
	if !ok {
		return
	}
	if item.size+delta < 0 {
		delta = -item.size
	}
	cm.currentMemory += delta
	cm.sizeByType[item.evictType] += delta
	item.size += delta
	item.evictionScore = item.size * evictableWeights[item.evictType]
	if item.heapIndex >= 0 {
		heap.Fix(&cm.h, item.heapIndex)
	}
}

func (cm *CacheManager) setSizeInternal(pointer any, size int64) {
	item, ok := cm.itemMap[pointer]
	if !ok {
		return
	}
	if size < 0 {
		size = 0
	}
	cm.updateSizeInternal(pointer, size-item.size)
}

// telemetryWeight calibrates telemetry (rebuild-decayed usage score) against
// age*weight (seconds * type priority). With K=50 and weight=20 (index):
//
//	Savings=4  → protected ~10s after last access
//	Savings=50 → protected ~125s after last access
//
// Telemetry is decayed by 0.9x at each rebuild, so it reflects recent usage rate.
const telemetryWeight = 50.0

type evictionCandidate struct {
	item  *softItem
	offer evictionOffer
}

func (cm *CacheManager) usage(typeFilter func(EvictableType) bool) int64 {
	if typeFilter == nil {
		return cm.currentMemory
	}
	var result int64
	for evictType, size := range cm.sizeByType {
		if typeFilter(EvictableType(evictType)) {
			result += size
		}
	}
	return result
}

func (cm *CacheManager) applyEviction(candidate evictionCandidate, mode evictionMode, freedByType *[numEvictableTypes]int64) int64 {
	item := candidate.item
	if _, ok := cm.itemMap[item.pointer]; !ok {
		return 0
	}
	before := cm.currentMemory
	result := item.object.evict(mode, item.size, freedByType)
	if !result.success {
		return 0
	}
	if result.fullyEvicted {
		cm.removeInternal(item.pointer, freedByType)
		return before - cm.currentMemory
	}
	freed := result.freedBytes
	if freed < 0 {
		freed = 0
	}
	if freed > item.size {
		freed = item.size
	}
	if freed == 0 {
		return 0
	}
	cm.currentMemory -= freed
	cm.sizeByType[item.evictType] -= freed
	freedByType[item.evictType] += freed
	item.size -= freed
	item.evictionScore = item.size * evictableWeights[item.evictType]
	return before - cm.currentMemory
}

// evict uses bounded candidate batches. Each batch first collects immutable
// offers, then applies partial alternatives greedily before full alternatives.
// If actual reclamation is smaller than proposed, another top-k batch is read.
func (cm *CacheManager) evict(currentUsage, budget, additionalSize int64, typeFilter func(EvictableType) bool) {
	if currentUsage+additionalSize <= budget {
		return
	}
	targetBudget := budget - additionalSize
	if targetBudget < 0 {
		targetBudget = 0
	}
	var held []*softItem
	var skipped []*softItem
	var freedByType [numEvictableTypes]int64
	var totalFreed int64

	for cm.usage(typeFilter) > targetBudget && cm.h.Len() > 0 {
		needToFree := cm.usage(typeFilter) - targetBudget
		candidateTarget := needToFree * 2
		if candidateTarget < 1 {
			candidateTarget = 1
		}
		var candidates []evictionCandidate
		var candidateSum int64
		for candidateSum < candidateTarget && cm.h.Len() > 0 {
			item := heap.Pop(&cm.h).(*softItem)
			if typeFilter != nil && !typeFilter(item.evictType) {
				skipped = append(skipped, item)
				continue
			}
			candidates = append(candidates, evictionCandidate{item: item, offer: item.object.evictionOffer(item.size)})
			candidateSum += item.size
		}
		if len(candidates) == 0 {
			break
		}

		now := time.Now()
		for i := range candidates {
			item := candidates[i].item
			age := now.Sub(item.getLastUsed(item.pointer)).Seconds()
			telemetry := 0.0
			if item.getScore != nil {
				telemetry = item.getScore(item.pointer)
			}
			item.dynamicScore = age*float64(evictableWeights[item.evictType]) - telemetry*telemetryWeight
		}
		hybridsort.Slice(candidates, func(i, j int) bool {
			return candidates[i].item.dynamicScore > candidates[j].item.dynamicScore
		})

		eligible := func(item *softItem) bool {
			lastActive := item.registeredAt
			if lu := item.getLastUsed(item.pointer).UnixNano(); lu > lastActive {
				lastActive = lu
			}
			return now.UnixNano()-lastActive >= item.minLifetime
		}
		// Pass 1: prefer shedding object-owned caches while preserving parents.
		for _, candidate := range candidates {
			if cm.usage(typeFilter) <= targetBudget {
				break
			}
			if candidate.offer.partialBytes <= 0 || !eligible(candidate.item) {
				continue
			}
			totalFreed += cm.applyEviction(candidate, evictPartial, &freedByType)
		}
		// Pass 2: fully evict surviving parents if partial reclamation was insufficient.
		for _, candidate := range candidates {
			if cm.usage(typeFilter) <= targetBudget {
				break
			}
			if candidate.offer.fullBytes <= 0 || !eligible(candidate.item) {
				continue
			}
			totalFreed += cm.applyEviction(candidate, evictFull, &freedByType)
		}
		for _, candidate := range candidates {
			if _, ok := cm.itemMap[candidate.item.pointer]; ok {
				held = append(held, candidate.item)
			}
		}
	}
	for _, item := range held {
		heap.Push(&cm.h, item)
	}
	for _, item := range skipped {
		heap.Push(&cm.h, item)
	}

	// log summary
	if totalFreed > 0 {
		log.Printf("memory pressure: freed %s total (%s temp columns, %s shard columns, %s indexes, %s keytables, %s cache entries, %s string dicts)",
			units.BytesSize(float64(totalFreed)),
			units.BytesSize(float64(freedByType[TypeTempColumn])),
			units.BytesSize(float64(freedByType[TypeShard])),
			units.BytesSize(float64(freedByType[TypeIndex])),
			units.BytesSize(float64(freedByType[TypeTempKeytable])),
			units.BytesSize(float64(freedByType[TypeCacheEntry])),
			units.BytesSize(float64(freedByType[TypeStringDict])),
		)
	}
}

// FormatStat returns a human-readable string of the cache state.
func (cs CacheStat) FormatStat() string {
	// Every byte has one owner. Indexes, materialized string dictionaries and
	// temp columns are therefore disjoint from their parent shard totals.

	var b strings.Builder
	b.WriteString(fmt.Sprintf("TotalBudget = %s\tPersistedBudget = %s\tTracked = %s\tPersisted = %s\n",
		units.BytesSize(float64(cs.MemoryBudget)),
		units.BytesSize(float64(cs.PersistedBudget)),
		units.BytesSize(float64(cs.CurrentMemory)),
		units.BytesSize(float64(cs.PersistedMemory))))
	b.WriteString("Type                     \tCount\tSize\n")
	b.WriteString(fmt.Sprintf("%-25s\t%d\t%s\n", "Temp columns", cs.CountByType[TypeTempColumn], units.BytesSize(float64(cs.SizeByType[TypeTempColumn]))))
	b.WriteString(fmt.Sprintf("%-25s\t%d\t%s\n", "Shard columns", cs.CountByType[TypeShard], units.BytesSize(float64(cs.SizeByType[TypeShard]))))
	b.WriteString(fmt.Sprintf("%-25s\t%d\t%s\n", "Indexes", cs.CountByType[TypeIndex], units.BytesSize(float64(cs.SizeByType[TypeIndex]))))
	b.WriteString(fmt.Sprintf("%-25s\t%d\t%s\n", "Temp keytables", cs.CountByType[TypeTempKeytable], units.BytesSize(float64(cs.SizeByType[TypeTempKeytable]))))
	b.WriteString(fmt.Sprintf("%-25s\t%d\t%s\n", "Cache entries", cs.CountByType[TypeCacheEntry], units.BytesSize(float64(cs.SizeByType[TypeCacheEntry]))))
	b.WriteString(fmt.Sprintf("%-25s\t%d\t%s\n", "String dicts (lz4)", cs.CountByType[TypeStringDict], units.BytesSize(float64(cs.SizeByType[TypeStringDict]))))
	return b.String()
}
