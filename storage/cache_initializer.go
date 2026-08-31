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

import "strings"
import "github.com/launix-de/memcp/scm"

type cacheInitializerRun struct {
	done       chan struct{}
	panicValue any
}

// initializeCache runs initialize exactly once for the lifetime of a canonical
// planner cache. Concurrent callers share both completion and failure. A later
// call retries after a failed run. The caller must pass the owning query session
// explicitly because initialization may begin inside a shard worker.
func (t *table) initializeCache(ss *scm.SessionState, initialize func()) (initialized bool) {
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
		t.cacheInitMu.Unlock()
		panic("cache initialization requires a query session")
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

	initialize()
	return true
}
