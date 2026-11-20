/*
Copyright (C) 2026 MemCP Contributors

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"sync"
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

func waitCacheMapLifetimeEvent(t *testing.T, event <-chan struct{}) {
	t.Helper()
	select {
	case <-event:
	case <-time.After(5 * time.Second):
		t.Fatal("cache factory lifetime event did not complete")
	}
}

func TestCacheMapShutdownJoinsFactoriesBeyondTheirWaiters(t *testing.T) {
	for _, test := range []struct {
		name     string
		detached bool
		panic    bool
	}{
		{name: "detached request", detached: true},
		{name: "factory panic", panic: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			owner := newCacheMapFactoryLifetime()
			cache := &cacheMap{
				entries:   make(map[string]*cacheMapEntry),
				flights:   make(map[string]*cacheMapFlight),
				factories: owner,
			}
			started, release, readerDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			var failure any
			t.Cleanup(func() {
				releaseOnce.Do(func() { close(release) })
				waitCacheMapLifetimeEvent(t, readerDone)
				if err := cache.Close(); err != nil {
					t.Error(err)
				}
			})
			cache.store("cached", scm.NewInt(7))
			go func() {
				defer close(readerDone)
				defer func() { failure = recover() }()
				cache.getOrCompute("working", scm.NewNil(), scm.NewFunc(func(...scm.Scmer) scm.Scmer {
					close(started)
					<-release
					if test.panic {
						panic("factory unwind")
					}
					return scm.NewInt(42)
				}))
			}()
			waitCacheMapLifetimeEvent(t, started)
			if test.detached {
				cache.mu.RLock()
				flight := cache.flights["working"]
				cache.mu.RUnlock()
				cache.cancelWaiter("working", flight)
				cache.mu.RLock()
				retained := cache.flights["working"] != nil
				cache.mu.RUnlock()
				if retained {
					t.Fatal("cancelled waiter did not detach its flight")
				}
			}
			stopped, stoppedAgain := make(chan struct{}), make(chan struct{})
			go func() { owner.shutdown(); close(stopped) }()
			waitCacheMapLifetimeEvent(t, owner.context.Done())
			go func() { owner.shutdown(); close(stoppedAgain) }()
			select {
			case <-stopped:
				t.Fatal("shutdown returned while its factory was still executing")
			default:
			}
			// Cache hits retain their existing semantics; shutdown only closes
			// admission of new background factory work.
			if value := cache.getOrCompute("cached", scm.NewNil(), scm.NewNil()); value.Int() != 7 {
				t.Fatalf("shutdown changed an existing cache hit: %v", value)
			}
			var rejected any
			func() {
				defer func() { rejected = recover() }()
				cache.getOrCompute("new", scm.NewNil(), scm.NewFunc(func(...scm.Scmer) scm.Scmer {
					t.Error("shutdown admitted a new factory")
					return scm.NewNil()
				}))
			}()
			if rejected != "cachemap factories are shut down" {
				t.Fatalf("new factory admission was not rejected: %v", rejected)
			}
			releaseOnce.Do(func() { close(release) })
			waitCacheMapLifetimeEvent(t, stopped)
			waitCacheMapLifetimeEvent(t, stoppedAgain)
			waitCacheMapLifetimeEvent(t, readerDone)
			if test.panic {
				origin, ok := failure.(interface {
					OriginalFailure() any
					OriginalStack() string
				})
				if !ok || origin.OriginalFailure() != "factory unwind" {
					t.Fatalf("factory panic did not unwind through its waiter: %v", failure)
				}
				if origin.OriginalStack() == "" {
					t.Fatal("factory panic lost its origin stack")
				}
			}
		})
	}
}
