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
	"sort"
	"strings"
	"time"

	units "github.com/docker/go-units"
	"github.com/launix-de/memcp/scm"
)

// EvictableType identifies the kind of cached object for factor lookup and stat reporting.
type EvictableType uint8

const (
	TypeTempColumn    EvictableType = iota // factor 1
	TypeShard                              // factor 5
	TypeIndex                              // factor 25
	TypeTempKeytable                       // factor 100
	numEvictableTypes                      // sentinel for array sizing
)

// evictableFactors maps EvictableType → rebuild cost factor.
// Higher factor = more protected = lower evictionScore.
var evictableFactors = [numEvictableTypes]int64{1, 5, 25, 100}

var evictableNames = [numEvictableTypes]string{"TempColumn", "Shard", "Index", "TempKeytable"}

type softItem struct {
	pointer       any
	size          int64
	evictType     EvictableType
	evictionScore int64 // = size / factor (static, max-heap key)
	cleanup       func(pointer any, freedByType *[numEvictableTypes]int64) bool
	getLastUsed   func(pointer any) time.Time
	getScore      func(pointer any) float64 // optional type-specific telemetry
	heapIndex     int                       // position in heap (-1 if not in heap)
	dynamicScore  float64                   // scratch field for Phase 2
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

// CacheManager manages memory-limited soft references with two-phase eviction.
// Two budgets: persistedBudget (shards+indexes) and memoryBudget (total).
type CacheManager struct {
	memoryBudget    int64 // total budget (default 90% of RAM)
	persistedBudget int64 // budget for persisted shards+indexes (default 30% of RAM)
	currentMemory   int64

	sizeByType  [numEvictableTypes]int64
	countByType [numEvictableTypes]int64

	h       softItemHeap
	itemMap map[any]*softItem

	opChan chan cacheOp
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
	factor := evictableFactors[evictType]
	item := &softItem{
		pointer:       pointer,
		size:          size,
		evictType:     evictType,
		evictionScore: size / factor,
		cleanup:       cleanup,
		getLastUsed:   getLastUsed,
		getScore:      getScore,
		heapIndex:     -1,
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
	done := make(chan struct{})
	cm.opChan <- cacheOp{del: pointer, done: done}
	<-done
}

// UpdateSize adjusts the tracked size by delta. Recomputes evictionScore and fixes heap.
func (cm *CacheManager) UpdateSize(pointer any, delta int64) {
	if cm.opChan == nil {
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
	done := make(chan struct{})
	cm.opChan <- cacheOp{pressureSize: additionalSize, done: done}
	<-done
}

// Stat returns per-type evictable sizes and counts.
func (cm *CacheManager) Stat() CacheStat {
	if cm.opChan == nil {
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
	// Tier 2: total budget (all types)
	if cm.memoryBudget > 0 {
		cm.evict(cm.currentMemory, cm.memoryBudget, additionalSize, nil)
	}
}

// run is the single-threaded goroutine handling all operations.
func (cm *CacheManager) run() {
	for op := range cm.opChan {
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
	cm.itemMap[item.pointer] = item
	cm.currentMemory += item.size
	scm.AdjustMemStats(item.size)
	cm.sizeByType[item.evictType] += item.size
	cm.countByType[item.evictType]++
	heap.Push(&cm.h, item)
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
	item.evictionScore = item.size / evictableFactors[item.evictType]
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

	// score dynamically
	now := time.Now()
	for _, c := range candidates {
		age := now.Sub(c.getLastUsed(c.pointer)).Seconds()
		telemetry := 0.0
		if c.getScore != nil {
			telemetry = c.getScore(c.pointer)
		}
		c.dynamicScore = age - telemetry*telemetryWeight
		// high dynamicScore = old + unimportant → evict
	}

	// sort by dynamicScore (worst first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].dynamicScore > candidates[j].dynamicScore
	})

	// evict worst 50%
	evictCount := len(candidates) / 2
	if evictCount < 1 && len(candidates) > 0 {
		evictCount = 1
	}
	var freedByType [numEvictableTypes]int64
	var totalFreed int64
	for i := 0; i < evictCount; i++ {
		c := candidates[i]
		// check again — previous cleanup in this loop may have freed this item recursively
		if _, ok := cm.itemMap[c.pointer]; !ok {
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

	// push survivors back into heap
	for i := evictCount; i < len(candidates); i++ {
		c := candidates[i]
		if _, ok := cm.itemMap[c.pointer]; ok {
			heap.Push(&cm.h, c)
		}
	}

	// log summary
	if totalFreed > 0 {
		shardColsOnly := freedByType[TypeShard] - freedByType[TypeIndex]
		if shardColsOnly < 0 {
			shardColsOnly = 0
		}
		log.Printf("memory pressure: freed %s total (%s temp columns, %s shard columns, %s indexes, %s keytables)",
			units.BytesSize(float64(totalFreed)),
			units.BytesSize(float64(freedByType[TypeTempColumn])),
			units.BytesSize(float64(shardColsOnly)),
			units.BytesSize(float64(freedByType[TypeIndex])),
			units.BytesSize(float64(freedByType[TypeTempKeytable])),
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
	return b.String()
}
