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

import "fmt"
import "sync"
import "time"
import "strings"
import "sync/atomic"
import "unsafe"
import "github.com/launix-de/memcp/scm"

// One optional RAM owner per immutable OverlayBlob generation. It deliberately
// has no pointer back to its column/shard/database: a soft registration cannot
// keep a retired persisted generation alive. Hashes deduplicate within an owner;
// different owners charge their own copies. No reference count authorizes I/O.
//
// Existing cachemap entries admit on the first access, expire by idle time when
// configured, and have per-key synchronization/registrations. A single partial
// cacheObject instead batches synchronization and owns probation metadata too.
// mu protects everything except lastUsed (published once per batch). Eviction
// only TryLocks: readers may publish to CacheManager while holding storage locks.
type blobRAMCache struct {
	requestedBytes int64 // largest denied admission in this batch, under mu
	mu             sync.Mutex
	entries        map[[32]byte]blobRAMEntry
	payloadBytes   int64
	savedWork      float64
	registered     bool
	lastUsed       atomic.Int64
}

// Queue publication under mu to preserve mutation order, but let the manager
// process it only after Unlock. Otherwise a newly registered probation owner
// is ineligible (busy) and pressure steals a warm owner's contents instead.
func (c *blobRAMCache) endBatch(before int64) {
	requested := c.requestedBytes
	c.requestedBytes = 0
	size := c.size()
	var ready, done chan struct{}
	if size > 0 && (!c.registered || size != before) {
		ready, done = make(chan struct{}), make(chan struct{})
		op := cacheOp{ready: ready, done: done}
		if !c.registered {
			c.registered = true
			op.add = newSoftItem(c, size, TypeCacheEntry, nil, blobRAMLastUsed, blobRAMScore, 0, 0)
		} else {
			op.updatePtr, op.updateDelta = c, size-before
		}
		GlobalCache.opChan <- op
	}
	c.mu.Unlock()
	if ready != nil {
		close(ready)
		<-done
	}
	if requested > 0 {
		GlobalCache.CheckPressure(requested)
	}
}

type blobRAMEntry struct {
	value    string // empty on probation; external blobs cannot have empty contents
	decodeNS float64
	reuses   uint8 // saturates; historical popularity cannot grow without bound
}

// Conservative allocation estimates include map load factor, growth/overflow,
// string headers, allocator rounding, and the global softItem/map/heap slot.
// Map entries are never deleted individually, so this bound does not shrink
// while Go retains buckets. Payload strings are cloned to exact-length storage;
// temporary gzip/Builder buffers belong to the executing reader, not the cache.
const blobRAMEntryBytes = 256
const blobRAMRegistrationBytes = 512

// Account for allocator size classes, not just logical string length. Small
// objects reserve a conservative 25% allowance; larger objects round to Go's
// 8 KiB heap pages. String headers are already in blobRAMEntryBytes.
func blobRAMStringBytes(length int) int64 {
	bytes := int64(length)
	if bytes <= 32768 {
		return (bytes + bytes/4 + 63) &^ 63
	}
	return (bytes + 8191) &^ 8191
}

func (c *blobRAMCache) size() int64 {
	if c.entries == nil {
		return 0
	}
	return blobRAMRegistrationBytes + int64(len(c.entries))*blobRAMEntryBytes + c.payloadBytes
}

// ComputeSize is inclusive. The fixed object survives payload eviction; only
// cachedBytes is registered as the independently reclaimable child portion.
func (c *blobRAMCache) ComputeSize() uint {
	return uint(unsafe.Sizeof(*c)) + c.cachedBytes()
}

func (c *blobRAMCache) cachedBytes() uint {
	c.mu.Lock()
	defer c.mu.Unlock()
	return uint(c.size())
}

func blobRAMLastUsed(pointer any) time.Time {
	return time.Unix(0, pointer.(*blobRAMCache).lastUsed.Load())
}

// Saved decode nanoseconds per resident byte supplies reuse/work-per-space
// telemetry to the existing heuristic (without changing type weights).
// Only successful measured reloads train decode cost. At most eight reuses are
// credited, and every pressure-driven partial eviction halves the history.
func blobRAMScore(pointer any) float64 {
	c := pointer.(*blobRAMCache)
	if !c.mu.TryLock() {
		return 0
	}
	defer c.mu.Unlock()
	if size := c.size(); size > 0 {
		return c.savedWork / float64(size)
	}
	return 0
}

func (c *blobRAMCache) read(s *OverlayBlob, hash [32]byte, limit *int64) scm.Scmer {
	e, seen := c.entries[hash]
	if e.value != "" {
		if e.reuses < 8 {
			e.reuses++
			c.savedWork += e.decodeNS
			c.entries[hash] = e
		}
		return scm.NewString(e.value)
	}
	if *limit < 0 {
		*limit = GlobalCache.Stat().MemoryBudget
	}
	start := time.Now()
	value, found := s.readBlob(hash)
	if !found {
		panic("OverlayBlob: missing blob " + fmt.Sprintf("%x", hash))
	}
	elapsed := float64(time.Since(start).Nanoseconds())
	if c.entries == nil {
		c.entries = make(map[[32]byte]blobRAMEntry)
	}
	if seen && e.decodeNS > 0 {
		e.decodeNS = (e.decodeNS + elapsed) / 2
		bytes := blobRAMStringBytes(len(value.String()))
		if *limit > 0 && c.size()+bytes > *limit {
			if bytes+blobRAMRegistrationBytes+blobRAMEntryBytes <= *limit {
				c.requestedBytes = max(c.requestedBytes, bytes)
			}
			c.entries[hash] = e
			return value
		}
		e.value = strings.Clone(value.String())
		e.reuses = 1
		c.payloadBytes += blobRAMStringBytes(len(e.value))
		c.savedWork += e.decodeNS
		value = scm.NewString(e.value)
	} else {
		e.decodeNS = elapsed // first read records evidence, never retains payload
	}
	c.entries[hash] = e
	return value
}

func (c *blobRAMCache) evictionOffer(size int64) evictionOffer {
	if !c.mu.TryLock() {
		return evictionOffer{}
	}
	defer c.mu.Unlock()
	if c.payloadBytes == 0 {
		// Probation contains no useful decoded representation to preserve.
		return evictionOffer{partialBytes: size, fullBytes: size}
	}
	return evictionOffer{partialBytes: c.payloadBytes, fullBytes: size}
}

func (c *blobRAMCache) evict(mode evictionMode, size int64, _ *[numEvictableTypes]int64) evictionResult {
	if !c.mu.TryLock() {
		return evictionResult{}
	}
	defer c.mu.Unlock()
	if mode == evictFull || c.payloadBytes == 0 {
		c.entries = nil
		c.payloadBytes = 0
		c.savedWork = 0
		c.registered = false
		return evictionResult{freedBytes: size, fullyEvicted: true, success: true}
	}
	// Shed the lower-benefit payloads first. Keep metadata until full eviction;
	// successful readers hold ordinary immutable Go strings through reclamation.
	threshold := c.savedWork / float64(c.payloadBytes)
	before := c.payloadBytes
	c.savedWork = 0
	for hash, e := range c.entries {
		if e.value != "" {
			bytes := blobRAMStringBytes(len(e.value))
			if e.decodeNS*float64(e.reuses)/float64(bytes) <= threshold {
				c.payloadBytes -= bytes
				e.value = ""
				e.reuses = 0
				e.decodeNS = 0 // require fresh evidence before readmission
			} else {
				e.reuses = (e.reuses + 1) / 2
				c.savedWork += e.decodeNS * float64(e.reuses)
			}
			c.entries[hash] = e
		}
	}
	return evictionResult{freedBytes: before - c.payloadBytes, success: before > c.payloadBytes}
}

// Called only by exclusive generation construction/source replacement.
func (s *OverlayBlob) resetBlobRAM() {
	if old := s.ram; old != nil {
		old.mu.Lock()
		registered := old.registered
		old.mu.Unlock()
		if registered {
			GlobalCache.Remove(old)
		}
	}
	s.ram = &blobRAMCache{}
}
