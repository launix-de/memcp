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
