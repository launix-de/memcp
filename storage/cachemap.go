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
}

// NewCacheMap creates a new cachemap and returns a Scheme function.
// (cachemap key value) — set entry
// (cachemap key) — get entry (or nil)
// (cachemap) — list all keys
func NewCacheMap(a ...scm.Scmer) scm.Scmer {
	cm := &cacheMap{
		entries: make(map[string]*cacheMapEntry),
	}
	return scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
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
			valueSize := int64(scm.ComputeSize(value))
			entrySize := valueSize + cacheMapEntryOverhead + int64(len(key))

			entry := &cacheMapEntry{
				cm:    cm,
				key:   key,
				value: value,
				size:  entrySize,
			}
			entry.lastUsed.Store(time.Now().UnixNano())

			// Remove old entry from CacheManager if it exists
			cm.mu.Lock()
			if old, ok := cm.entries[key]; ok {
				GlobalCache.Remove(old)
			}
			cm.entries[key] = entry
			cm.mu.Unlock()

			// Register with CacheManager for LRU eviction
			GlobalCache.AddItem(
				entry,
				entrySize,
				TypeCacheEntry,
				cacheMapCleanup,
				cacheMapGetLastUsed,
				nil,
			)

			return value
		default:
			panic("cachemap: expected 0, 1, or 2 arguments")
		}
	})
}

// cacheMapCleanup is called by CacheManager on eviction.
func cacheMapCleanup(pointer any, freedByType *[numEvictableTypes]int64) bool {
	entry := pointer.(*cacheMapEntry)
	entry.cm.mu.Lock()
	delete(entry.cm.entries, entry.key)
	entry.cm.mu.Unlock()
	return true
}

// cacheMapGetLastUsed returns the last access time for LRU scoring.
func cacheMapGetLastUsed(pointer any) time.Time {
	entry := pointer.(*cacheMapEntry)
	return time.Unix(0, entry.lastUsed.Load())
}
