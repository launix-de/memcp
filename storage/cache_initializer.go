/*
Copyright (C) 2026  Carl-Philip Haensch

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
import "strings"
import "github.com/launix-de/memcp/scm"

type cacheInitializerRun struct {
	done       chan struct{}
	panicValue any
}

// initializeCache runs initialize exactly once for the lifetime of a canonical
// planner cache. Concurrent callers share both completion and failure. A later
// call retries after a failed run. A SQL query passes its owning query session.
// Internal cache builders deliberately run outside a client query; for those,
// use one unregistered SessionState solely as the table-lock owner for the
// lifetime of this initialization. It is not process-list or cancellation state.
func (t *table) initializeCache(ss *scm.SessionState, initialize func(*scm.SessionState)) (initialized bool) {
	if !strings.HasPrefix(t.Name, ".") {
		panic("cache initialization requires a dot-prefixed cache table")
	}

	t.cacheInitMu.Lock()
	if t.cacheInitialized {
		t.cacheInitMu.Unlock()
		return false
	}
	if running := t.cacheInitializerRun; running != nil {
		t.cacheInitMu.Unlock()
		<-running.done
		if running.panicValue != nil {
			panic(running.panicValue)
		}
		return false
	}
	run := &cacheInitializerRun{
		done: make(chan struct{}),
	}
	if ss == nil {
		ss = &scm.SessionState{}
	}
	t.cacheInitializerRun = run
	t.cacheInitMu.Unlock()

	defer func() {
		panicValue := recover()
		t.cacheInitMu.Lock()
		if panicValue == nil {
			t.cacheInitialized = true
		} else {
			run.panicValue = panicValue
		}
		t.cacheInitializerRun = nil
		close(run.done)
		t.cacheInitMu.Unlock()
		if panicValue != nil {
			panic(panicValue)
		}
	}()

	initialize(ss)
	return true
}

// prepareCache owns memoization of logical range preparation requests. Neither
// cache eviction nor source mutation stamps are exposed to the Scheme caller.
// The caller supplies the requested domain and a synchronous, idempotent recipe.
// Its returned scalar (including nil) may also be a normalization witness.
// A recipe receives its previous result only while domain sources are unchanged;
// changes to data sources alone allow incremental preparation from that result.
// Aggregate payloads are maintained independently by their computed columns.
func (t *table) prepareCache(key string, domainSources, dataSources []scm.Scmer, prepare scm.Scmer) scm.Scmer {
	sources := make([]scm.Scmer, 0, len(domainSources)+len(dataSources))
	sources = append(sources, domainSources...)
	sources = append(sources, dataSources...)
	t.cachePreparationMu.Lock()
	defer t.cachePreparationMu.Unlock()
	cache := t.cachePreparations.Load()
	if cache == nil {
		cache = &cacheMap{entries: make(map[string]*cacheMapEntry), flights: make(map[string]*cacheMapFlight)}
		t.cachePreparations.Store(cache)
	}
	stamp := make([]scm.Scmer, 2, 2+2*len(sources))
	stamp[0] = scm.NewInt(int64(t.cacheGeneration.Load()))
	stamp[1] = scm.NewInt(int64(len(domainSources)))
	stable, domainStable := true, true
	for i, source := range sources {
		src := TableFromScmer(source)
		if src.contributionWriters.Load() != 0 {
			stable = false
			if i < len(domainSources) {
				domainStable = false
			}
		}
		identity := src.contributionIdentity.Load()
		if identity == 0 {
			src.contributionIdentity.CompareAndSwap(0, contributionTableID.Add(1))
			identity = src.contributionIdentity.Load()
		}
		stamp = append(stamp, scm.NewInt(int64(identity)), scm.NewInt(int64(src.cacheDataRevision.Load())))
	}
	version := scm.NewSlice(stamp)
	cache.mu.RLock()
	entry := cache.entries[key]
	targetUnchanged := entry != nil && t.contributionWriters.Load() == 0 &&
		uint64(entry.value.Slice()[2].Int()) == t.cacheDataRevision.Load()
	if stable && targetUnchanged && scm.Equal(entry.value.Slice()[0], version) {
		entry.lastUsed.Store(time.Now().UnixNano())
		value := entry.value.Slice()[1]
		cache.mu.RUnlock()
		return value
	}
	// Only the recipe's semantic result crosses the boundary. A data change
	// may reuse it, but a changed driver domain or target lifetime must not.
	previous := scm.NewNil()
	if domainStable && targetUnchanged {
		old := entry.value.Slice()[0].Slice()
		prefix := 2 + 2*len(domainSources)
		sameDomain := len(old) >= prefix
		for i := 0; sameDomain && i < prefix; i++ {
			sameDomain = scm.Equal(old[i], stamp[i])
		}
		if sameDomain {
			previous = entry.value.Slice()[1]
		}
	}
	cache.mu.RUnlock()
	value := scm.Apply(prepare, previous)
	// A changing source cannot certify the prepared domain. Keep the work but
	// retry its idempotent recipe on the next request rather than retaining it.
	for i, source := range sources {
		src := TableFromScmer(source)
		stable = stable && src.contributionWriters.Load() == 0 &&
			src.cacheDataRevision.Load() == uint64(stamp[3+2*i].Int())
	}
	if stable && t.cacheGeneration.Load() == uint64(stamp[0].Int()) {
		cache.store(key, scm.NewSlice([]scm.Scmer{version, value, scm.NewInt(int64(t.cacheDataRevision.Load()))}))
	}
	return value
}

// discardCacheValue forgets one logical payload, preserving other aggregates at
// the same coordinate. It runs at a normal query/maintenance boundary, NEVER on
// the CacheManager owner goroutine: scans and mutation callbacks may block and
// update the memory ledger. Callers own the semantic domain mutex and must prove
// that no retained snapshot of this payload depends on the discarded coordinate.
// DDL stays read-locked through the scan so a newly added payload cannot be lost.
func (t *table) discardCacheValue(currentTx *TxContext, keys []string, values []scm.Scmer, payload string, expected scm.Scmer) scm.Scmer {
	if len(keys) == 0 || len(keys) != len(values) || expected.IsNil() {
		panic("discard_cache_value requires a complete coordinate and a non-null expected payload")
	}
	if txRequiresQueryLocalCache(currentTx) {
		panic("discard_cache_value cannot mutate a shared cache from a transaction-local view")
	}
	if !t.acquireCacheUse() {
		return scm.NewInt(0)
	}
	defer t.releaseCacheUse()
	// Pin every payload while holding schema metadata rights. Column eviction
	// uses these pins, not ddlMu. Never carry schemalock into scan execution.
	t.schema.schemalock.RLock()
	if !t.ddlMu.TryRLock() {
		t.schema.schemalock.RUnlock()
		return scm.NewInt(0)
	}
	defer t.ddlMu.RUnlock()
	pinned := make([]*column, 0, len(t.Columns))
	defer func() {
		for _, col := range pinned {
			col.releaseCacheUse()
		}
	}()
	otherPayloads := make([]string, 0, len(t.Columns))
	metadataReady := func() bool {
		defer t.schema.schemalock.RUnlock()
		if t.PersistencyMode != Cache || !strings.HasPrefix(t.Name, ".grp:") {
			panic("discard_cache_value requires an ephemeral group cache")
		}
		keySet := make(map[string]bool, len(keys))
		for _, key := range keys {
			if keySet[key] || key == payload {
				panic("discard_cache_value has duplicate or overlapping coordinate columns")
			}
			keySet[key] = true
		}
		foundPayload, foundKeys := false, 0
		for _, col := range t.Columns {
			if !col.acquireCacheUse() {
				return false
			}
			pinned = append(pinned, col)
			if keySet[col.Name] {
				foundKeys++
				continue
			}
			if !col.IsTemp || col.Computor.IsNil() {
				panic("discard_cache_value coordinate omits a stored dimension")
			}
			if col.Name == payload {
				foundPayload = true
			} else {
				otherPayloads = append(otherPayloads, col.Name)
			}
		}
		if !foundPayload || foundKeys != len(keys) {
			panic("discard_cache_value references an unknown coordinate or payload")
		}
		return true
	}()
	if !metadataReady {
		return scm.NewInt(0)
	}
	condition := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		for i := range keys {
			if !scm.Equal(a[i], values[i]) {
				return scm.NewBool(false)
			}
		}
		return scm.NewBool(true)
	})
	mapCols := append(otherPayloads, payload, "$set:"+payload, "$update")
	forget := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		n := len(otherPayloads)
		if !scm.Equal(a[n+1], expected) {
			return a[0]
		}
		for _, other := range a[1 : n+1] {
			if !other.IsNil() {
				scm.Apply(a[n+2], scm.NewNil())
				return scm.NewInt(int64(scm.ToInt(a[0]) + 1))
			}
		}
		// Release the payload reference even while a deleted row remains a tombstone.
		scm.Apply(a[n+2], scm.NewNil())
		// This cell has no other materialized aggregate. Remove its logical row;
		// normal cache rebuild compacts tombstones. Do not claim its row bytes as
		// immediately reclaimed or delete any persistent files here.
		scm.Apply(a[n+3])
		return scm.NewInt(int64(scm.ToInt(a[0]) + 1))
	})
	combine := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		return scm.NewInt(int64(scm.ToInt(a[0]) + scm.ToInt(a[1])))
	})
	return t.scan(currentTx, scm.NewSlice(newExactScanAccessSchema(keys)), values,
		keys, condition, mapCols, forget, scm.NewInt(0), combine, false)
}
