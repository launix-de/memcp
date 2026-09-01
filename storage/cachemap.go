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
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/launix-de/memcp/scm"
)

// Per-entry overhead: cacheMapEntry struct (~80 bytes) + map bucket slot (~128 bytes) + softItem (~120 bytes)
const cacheMapEntryOverhead = 328

type cacheMapEntry struct {
	cm       *cacheMap
	key      string
	value    scm.Scmer
	size     int64
	lastUsed atomic.Int64 // UnixNano timestamp, lock-free for concurrent reads
}

type cacheMap struct {
	mu      sync.RWMutex
	entries map[string]*cacheMapEntry
	flights map[string]*cacheMapFlight
}

type cacheMapFlight struct {
	done       chan struct{}
	cancel     context.CancelFunc
	value      scm.Scmer
	panicValue any
	failed     bool
	waiters    int
}

var cacheMapCallableType = &scm.TypeDescriptor{Kind: "func", Label: "cachemap", Description: "thread-safe cache accessor",
	Params: []*scm.TypeDescriptor{
		{Kind: "any", Label: "key_or_operation", Description: "cache key, or get_or_compute", Optional: true},
		{Kind: "any", Label: "value_or_key", Description: "value to store, or key for get_or_compute", Optional: true},
		{Kind: "func", CallsOnce: true, Label: "producer", Description: "zero-argument value producer used by get_or_compute", Optional: true, Params: []*scm.TypeDescriptor{}, Return: &scm.TypeDescriptor{Kind: "any", Label: "value"}},
	},
	Return: &scm.TypeDescriptor{Kind: "any", Label: "result", Description: "cached value, previous value, or key list depending on the operation"},
}

var cacheMapCallableTypeID = scm.RegisterCallableType(cacheMapCallableType)

// NewCacheMap creates a new cachemap and returns a Scheme function.
// (cachemap key value) — set entry
// (cachemap key) — get entry (or nil)
// (cachemap) — list all keys
// (cachemap "get_or_compute" key tx producer) — get or run producer once per key
func NewCacheMap(a ...scm.Scmer) scm.Scmer {
	cm := &cacheMap{
		entries: make(map[string]*cacheMapEntry),
		flights: make(map[string]*cacheMapFlight),
	}
	return scm.NewTypedFunc(func(a ...scm.Scmer) scm.Scmer {
		switch len(a) {
		case 0:
			// list all keys
			cm.mu.RLock()
			keys := make([]scm.Scmer, 0, len(cm.entries))
			for k := range cm.entries {
				keys = append(keys, scm.NewString(k))
			}
			cm.mu.RUnlock()
			return scm.NewSlice(keys)
		case 1:
			// get
			key := scm.String(a[0])
			cm.mu.RLock()
			entry, ok := cm.entries[key]
			cm.mu.RUnlock()
			if !ok {
				return scm.NewNil()
			}
			entry.lastUsed.Store(time.Now().UnixNano())
			return entry.value
		case 2:
			// set
			key := scm.String(a[0])
			value := a[1]
			cm.store(key, value)
			return value
		case 3:
			if scm.String(a[0]) != "get_or_compute" {
				panic("cachemap: unknown 3-argument operation")
			}
			return cm.getOrCompute(scm.String(a[1]), scm.NewNil(), a[2])
		case 4:
			if scm.String(a[0]) != "get_or_compute" {
				panic("cachemap: unknown 4-argument operation")
			}
			return cm.getOrCompute(scm.String(a[1]), a[2], a[3])
		default:
			panic("cachemap: expected 0, 1, 2, 3, or 4 arguments")
		}
	}, cacheMapCallableTypeID)
}

func newCacheMapEntry(cm *cacheMap, key string, value scm.Scmer) *cacheMapEntry {
	entry := &cacheMapEntry{
		cm:    cm,
		key:   key,
		value: value,
		size:  int64(scm.ComputeSize(value)) + cacheMapEntryOverhead + int64(len(key)),
	}
	entry.lastUsed.Store(time.Now().UnixNano())
	return entry
}

func registerCacheMapEntry(entry *cacheMapEntry) {
	GlobalCache.AddItem(
		entry,
		entry.size,
		TypeCacheEntry,
		cacheMapCleanup,
		cacheMapGetLastUsed,
		nil,
	)
}

func (cm *cacheMap) store(key string, value scm.Scmer) {
	entry := newCacheMapEntry(cm, key, value)
	cm.mu.Lock()
	old := cm.entries[key]
	cm.entries[key] = entry
	cm.mu.Unlock()
	if old != nil {
		GlobalCache.Remove(old)
	}
	registerCacheMapEntry(entry)
}

func cacheMapContext(queryState scm.Scmer) context.Context {
	if queryState.IsNil() {
		return context.Background()
	}
	tx := scmerToTxContext(queryState)
	if tx == nil {
		return context.Background()
	}
	ss, seq := tx.QuerySessionState()
	if ss == nil || seq == 0 {
		return context.Background()
	}
	if ctx := ss.QueryContext(seq); ctx != nil {
		return ctx
	}
	return context.Background()
}

func (cm *cacheMap) getOrCompute(key string, queryState scm.Scmer, producer scm.Scmer) scm.Scmer {
	cm.mu.RLock()
	if entry := cm.entries[key]; entry != nil {
		entry.lastUsed.Store(time.Now().UnixNano())
		cm.mu.RUnlock()
		return entry.value
	}
	cm.mu.RUnlock()

	cm.mu.Lock()
	if entry := cm.entries[key]; entry != nil {
		entry.lastUsed.Store(time.Now().UnixNano())
		cm.mu.Unlock()
		return entry.value
	}
	flight := cm.flights[key]
	if flight == nil {
		compileCtx, cancel := context.WithCancel(context.Background())
		flight = &cacheMapFlight{
			done:    make(chan struct{}),
			cancel:  cancel,
			waiters: 1,
		}
		cm.flights[key] = flight
		go cm.runProducer(compileCtx, key, producer, flight)
	} else {
		flight.waiters++
	}
	cm.mu.Unlock()

	ctx := cacheMapContext(queryState)
	select {
	case <-flight.done:
		if flight.failed {
			panic(flight.panicValue)
		}
		return flight.value
	case <-ctx.Done():
		cm.cancelWaiter(key, flight)
		panic(ctx.Err())
	}
}

func (cm *cacheMap) cancelWaiter(key string, flight *cacheMapFlight) {
	cm.mu.Lock()
	if cm.flights[key] == flight {
		flight.waiters--
		if flight.waiters == 0 {
			delete(cm.flights, key)
			flight.cancel()
		}
	}
	cm.mu.Unlock()
}

func (cm *cacheMap) runProducer(ctx context.Context, key string, producer scm.Scmer, flight *cacheMapFlight) {
	value, panicValue, failed := runCacheMapProducer(ctx, producer)

	var entry *cacheMapEntry
	if !failed && ctx.Err() == nil {
		entry = newCacheMapEntry(cm, key, value)
	}

	cm.mu.Lock()
	if cm.flights[key] != flight {
		cm.mu.Unlock()
		close(flight.done)
		return
	}
	delete(cm.flights, key)
	flight.value = value
	flight.panicValue = panicValue
	flight.failed = failed || ctx.Err() != nil
	var old *cacheMapEntry
	if entry != nil {
		old = cm.entries[key]
		cm.entries[key] = entry
	}
	close(flight.done)
	cm.mu.Unlock()

	if old != nil {
		GlobalCache.Remove(old)
	}
	if entry != nil {
		registerCacheMapEntry(entry)
	}
}

func runCacheMapProducer(ctx context.Context, producer scm.Scmer) (value scm.Scmer, panicValue any, failed bool) {
	failed = true
	defer func() {
		if recovered := recover(); recovered != nil {
			panicValue = recovered
		}
	}()
	value = scm.Apply(producer)
	failed = false
	return
}

// cacheMapCleanup is called by CacheManager on eviction.
func cacheMapCleanup(pointer any, freedByType *[numEvictableTypes]int64) bool {
	entry := pointer.(*cacheMapEntry)
	entry.cm.mu.Lock()
	if entry.cm.entries[entry.key] == entry {
		delete(entry.cm.entries, entry.key)
	}
	entry.cm.mu.Unlock()
	return true
}

// cacheMapGetLastUsed returns the last access time for LRU scoring.
func cacheMapGetLastUsed(pointer any) time.Time {
	entry := pointer.(*cacheMapEntry)
	return time.Unix(0, entry.lastUsed.Load())
}
