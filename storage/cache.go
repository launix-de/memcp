/*
Copyright (C) 2025  MemCP Contributors

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
	"sort"
	"time"
)

type softItem struct {
	pointer        any
	size           int64
	priorityFactor int
	cleanup        func(pointer any)
	getLastUsed    func(pointer any) time.Time
	effectiveTime  time.Time
}

// CacheManager manages memory-limited soft references.
type CacheManager struct {
	memoryBudget  int64
	currentMemory int64

	items    []softItem
	indexMap map[any]int // pointer -> index in items slice

	opChan chan cacheOp
}

type cacheOp struct {
	add  *softItem
	del  any
	done chan struct{}
}

// NewCacheManager creates a new CacheManager with given memory budget.
func NewCacheManager(memoryBudget int64) *CacheManager {
	cm := &CacheManager{
		memoryBudget: memoryBudget,
		items:        make([]softItem, 0),
		indexMap:     make(map[any]int),
		opChan:       make(chan cacheOp, 1024),
	}
	go cm.run()
	return cm
}

// AddItem inserts a new item into the cache. Cleanup is called if over budget.
func (cm *CacheManager) AddItem(
	pointer any,
	size int64,
	priorityFactor int,
	cleanup func(pointer any),
	getLastUsed func(pointer any) time.Time,
) {
	item := &softItem{
		pointer:        pointer,
		size:           size,
		priorityFactor: priorityFactor,
		cleanup:        cleanup,
		getLastUsed:    getLastUsed,
		effectiveTime:  time.Now(), // always now for new items
	}
	done := make(chan struct{})
	cm.opChan <- cacheOp{add: item, done: done}
	<-done
}

// Delete removes an item from the cache immediately.
func (cm *CacheManager) Delete(pointer any) {
	done := make(chan struct{})
	cm.opChan <- cacheOp{del: pointer, done: done}
	<-done
}

// run is the single-threaded goroutine handling all operations and cleanup.
func (cm *CacheManager) run() {
	for op := range cm.opChan {
		if op.add != nil {
			cm.add(op.add)
		} else if op.del != nil {
			cm.delete(op.del)
		}
		if op.done != nil {
			close(op.done)
		}
	}
}

// add inserts a new softItem and triggers cleanup if over budget.
func (cm *CacheManager) add(item *softItem) {
	idx := len(cm.items)
	cm.items = append(cm.items, *item)
	cm.indexMap[item.pointer] = idx
	cm.currentMemory += item.size

	if cm.currentMemory > cm.memoryBudget {
		cm.cleanup()
	}
}

// delete removes a softItem immediately.
func (cm *CacheManager) delete(pointer any) {
	idx, ok := cm.indexMap[pointer]
	if !ok {
		return
	}
	item := cm.items[idx]
	item.cleanup(item.pointer)
	cm.currentMemory -= item.size

	lastIdx := len(cm.items) - 1
	if idx != lastIdx {
		cm.items[idx] = cm.items[lastIdx]
		cm.indexMap[cm.items[idx].pointer] = idx
	}
	cm.items = cm.items[:lastIdx]
	delete(cm.indexMap, pointer)
}

// cleanup frees memory to respect the memory budget (simple-stupid approach).
func (cm *CacheManager) cleanup() {
	if cm.currentMemory <= cm.memoryBudget {
		return
	}

	targetMemory := cm.memoryBudget * 75 / 100 // free until 75% of budget

	// Step 1: recompute effectiveTime for all items
	for i := range cm.items {
		cm.items[i].effectiveTime = cm.items[i].getLastUsed(cm.items[i].pointer)
	}

	// Step 2: sort items by effectiveTime (oldest first)
	sort.Slice(cm.items, func(i, j int) bool {
		return cm.items[i].effectiveTime.Before(cm.items[j].effectiveTime)
	})

	// Step 3: evict oldest items until memory is under target
	i := 0
	for cm.currentMemory > targetMemory && i < len(cm.items) {
		item := cm.items[i]
		item.cleanup(item.pointer)
		cm.currentMemory -= item.size
		delete(cm.indexMap, item.pointer)
		i++
	}

	// Step 4: compact the slice
	cm.items = cm.items[i:]
	for idx, item := range cm.items {
		cm.indexMap[item.pointer] = idx
	}
}

