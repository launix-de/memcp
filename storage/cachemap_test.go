/*
Copyright (C) 2026  MemCP Contributors

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

func TestCompileCacheBoundsDistinctColdProducers(t *testing.T) {
	cache := NewCacheMap(scm.NewString("compile"))
	limit := (runtime.GOMAXPROCS(0) + 1) / 2
	if limit < 1 {
		limit = 1
	}
	if limit > 4 {
		limit = 4
	}
	producerCount := limit + 4
	started := make(chan struct{}, producerCount)
	release := make(chan struct{})
	var callers sync.WaitGroup

	for i := 0; i < producerCount; i++ {
		callers.Add(1)
		go func(key string) {
			defer callers.Done()
			producer := scm.NewFunc(func(_ ...scm.Scmer) scm.Scmer {
				started <- struct{}{}
				<-release
				return scm.NewString(key)
			})
			scm.Apply(cache,
				scm.NewString("get_or_compute"),
				scm.NewString(key),
				producer,
			)
		}(fmt.Sprintf("shape-%d", i))
	}

	for i := 0; i < limit; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(release)
			callers.Wait()
			t.Fatal("compile producer did not reach the admission limit")
		}
	}
	select {
	case <-started:
		close(release)
		callers.Wait()
		t.Fatalf("more than %d distinct cold producers compiled concurrently", limit)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	callers.Wait()
}

func TestCompileCacheAdmissionAccountsForProducerWeight(t *testing.T) {
	cache := NewCacheMap(scm.NewString("compile"))
	limit := (runtime.GOMAXPROCS(0) + 1) / 2
	if limit < 1 {
		limit = 1
	}
	if limit > 4 {
		limit = 4
	}

	heavyStarted := make(chan struct{}, 1)
	heavyRelease := make(chan struct{})
	lightStarted := make(chan struct{}, 1)
	var callers sync.WaitGroup
	callers.Add(2)
	go func() {
		defer callers.Done()
		producer := scm.NewFunc(func(_ ...scm.Scmer) scm.Scmer {
			heavyStarted <- struct{}{}
			<-heavyRelease
			return scm.NewString("heavy")
		})
		scm.Apply(cache,
			scm.NewString("get_or_compute"),
			scm.NewString("heavy"),
			producer,
			scm.NewInt(int64(limit)),
		)
	}()
	select {
	case <-heavyStarted:
	case <-time.After(time.Second):
		close(heavyRelease)
		callers.Wait()
		t.Fatal("weighted producer did not start")
	}

	go func() {
		defer callers.Done()
		producer := scm.NewFunc(func(_ ...scm.Scmer) scm.Scmer {
			lightStarted <- struct{}{}
			return scm.NewString("light")
		})
		scm.Apply(cache,
			scm.NewString("get_or_compute"),
			scm.NewString("light"),
			producer,
			scm.NewInt(1),
		)
	}()
	select {
	case <-lightStarted:
		close(heavyRelease)
		callers.Wait()
		t.Fatal("light producer bypassed the weighted admission budget")
	case <-time.After(50 * time.Millisecond):
	}

	close(heavyRelease)
	select {
	case <-lightStarted:
	case <-time.After(time.Second):
		callers.Wait()
		t.Fatal("light producer did not start after weighted release")
	}
	callers.Wait()
}

func TestCompileCacheAdmissionLetsLightProducerBypassQueuedHeavyProducer(t *testing.T) {
	limit := (runtime.GOMAXPROCS(0) + 1) / 2
	if limit < 1 {
		limit = 1
	}
	if limit > 4 {
		limit = 4
	}
	if limit == 1 {
		t.Skip("one admission slot cannot retain spare capacity")
	}

	cache := NewCacheMap(scm.NewString("compile"))
	firstStarted := make(chan struct{}, 1)
	firstRelease := make(chan struct{})
	queuedStarted := make(chan struct{}, 1)
	queuedRelease := make(chan struct{})
	lightStarted := make(chan struct{}, 1)
	var callers sync.WaitGroup
	startProducer := func(key string, weight int, started chan<- struct{}, release <-chan struct{}) {
		callers.Add(1)
		go func() {
			defer callers.Done()
			producer := scm.NewFunc(func(_ ...scm.Scmer) scm.Scmer {
				started <- struct{}{}
				if release != nil {
					<-release
				}
				return scm.NewString(key)
			})
			scm.Apply(cache,
				scm.NewString("get_or_compute"),
				scm.NewString(key),
				producer,
				scm.NewInt(int64(weight)),
			)
		}()
	}

	startProducer("first-heavy", limit-1, firstStarted, firstRelease)
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		close(firstRelease)
		callers.Wait()
		t.Fatal("first weighted producer did not start")
	}

	startProducer("queued-heavy", limit, queuedStarted, queuedRelease)
	select {
	case <-queuedStarted:
		close(firstRelease)
		close(queuedRelease)
		callers.Wait()
		t.Fatal("full-weight producer started without enough capacity")
	case <-time.After(50 * time.Millisecond):
	}

	startProducer("light", 1, lightStarted, nil)
	select {
	case <-lightStarted:
	case <-time.After(time.Second):
		close(firstRelease)
		<-queuedStarted
		close(queuedRelease)
		callers.Wait()
		t.Fatal("light producer was blocked behind a queued full-weight producer")
	}

	close(firstRelease)
	select {
	case <-queuedStarted:
	case <-time.After(time.Second):
		close(queuedRelease)
		callers.Wait()
		t.Fatal("queued full-weight producer did not start after capacity was released")
	}
	close(queuedRelease)
	callers.Wait()
}
