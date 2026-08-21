/*
Copyright (C) 2026  MemCP Contributors

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
import "testing"
import "container/heap"

func TestEvictExpiredRetriesFailedCleanup(t *testing.T) {
	pointer := new(int)
	attempts := 0
	registeredAt := time.Now().Add(-time.Hour).UnixNano()
	item := &softItem{
		pointer:      pointer,
		evictType:    TypeIndex,
		heapIndex:    -1,
		registeredAt: registeredAt,
		maxIdleTime:  int64(time.Minute),
		getLastUsed:  func(any) time.Time { return time.Unix(0, registeredAt) },
		getScore:     func(any) float64 { return 0 },
		cleanup: func(any, *[numEvictableTypes]int64) bool {
			attempts++
			return attempts > 1
		},
	}
	cm := &CacheManager{
		countByType: [numEvictableTypes]int64{TypeIndex: 1},
		itemMap:     map[any]*softItem{pointer: item},
	}
	heap.Push(&cm.expH, expiryEntry{estimatedExpiry: registeredAt + item.maxIdleTime, pointer: pointer})

	cm.evictExpired()
	if _, ok := cm.itemMap[pointer]; !ok {
		t.Fatal("item was removed after cleanup reported lock contention")
	}
	if cm.expH.Len() != 1 {
		t.Fatalf("expiry retry entries = %d, want 1", cm.expH.Len())
	}

	cm.expH[0].estimatedExpiry = time.Now().Add(-time.Second).UnixNano()
	cm.evictExpired()
	if _, ok := cm.itemMap[pointer]; ok {
		t.Fatal("item remained registered after successful cleanup retry")
	}
	if attempts != 2 {
		t.Fatalf("cleanup attempts = %d, want 2", attempts)
	}
}
