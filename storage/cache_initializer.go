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
// Aggregate payloads are maintained independently by their computed columns.
func (t *table) prepareCache(key string, sources []scm.Scmer, prepare scm.Scmer) scm.Scmer {
	t.cachePreparationMu.Lock()
	defer t.cachePreparationMu.Unlock()
	cache := t.cachePreparations.Load()
	if cache == nil {
		cache = &cacheMap{entries: make(map[string]*cacheMapEntry), flights: make(map[string]*cacheMapFlight)}
		t.cachePreparations.Store(cache)
	}
	stamp := make([]scm.Scmer, 1, 1+2*len(sources))
	stamp[0] = scm.NewInt(int64(t.cacheGeneration.Load()))
	stable := true
	for _, source := range sources {
		src := TableFromScmer(source)
		if src.contributionWriters.Load() != 0 {
			stable = false
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
	if stable && entry != nil && scm.Equal(entry.value.Slice()[0], version) {
		entry.lastUsed.Store(time.Now().UnixNano())
		value := entry.value.Slice()[1]
		cache.mu.RUnlock()
		return value
	}
	cache.mu.RUnlock()
	value := scm.Apply(prepare)
	// A changing source cannot certify the prepared domain. Keep the work but
	// retry its idempotent recipe on the next request rather than retaining it.
	for i, source := range sources {
		src := TableFromScmer(source)
		stable = stable && src.contributionWriters.Load() == 0 &&
			src.cacheDataRevision.Load() == uint64(stamp[2+2*i].Int())
	}
	if stable && t.cacheGeneration.Load() == uint64(stamp[0].Int()) {
		cache.store(key, scm.NewSlice([]scm.Scmer{version, value}))
	}
	return value
}
