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

import "fmt"
import "strconv"
import "sync"
import "testing"
import "time"

func BenchmarkCacheManagerAsyncSizeUpdate(b *testing.B) {
	manager := new(CacheManager)
	manager.Init(0, 0)
	pointer := new(int)
	manager.AddItem(
		pointer,
		1<<30,
		TypeIndex,
		func(any, *[numEvictableTypes]int64) bool { return true },
		func(any) time.Time { return time.Time{} },
		nil,
	)
	b.Cleanup(manager.Stop)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		delta := int64(1)
		for pb.Next() {
			manager.UpdateSizeAsync(pointer, delta)
			delta = -delta
		}
	})
	manager.Stat()
}

func BenchmarkStorageIndexEvictionOffer(b *testing.B) {
	idx := new(StorageIndex)
	idx.skipListCacheBytes.Store(64 << 20)
	idx.skipListPartialBytes.Store(32 << 20)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			offer := idx.evictionOffer(128 << 20)
			if offer.partialBytes == 0 {
				b.Fatal("missing partial offer")
			}
		}
	})
}

func BenchmarkStorageIndexPartialEviction(b *testing.B) {
	for _, childCount := range []int{1_000, 10_000, 100_000} {
		for _, hotEvery := range []int{0, 2} {
			name := fmt.Sprintf("cold-%d", childCount)
			if hotEvery > 0 {
				name = fmt.Sprintf("half-hot-%d", childCount)
			}
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				for iteration := 0; iteration < b.N; iteration++ {
					b.StopTimer()
					idx := new(StorageIndex)
					cache := new(sync.Map)
					idx.baseState.skipLists = []*sync.Map{cache}
					var size int64
					var coldSize int64
					for child := 0; child < childCount; child++ {
						key := skipListKey{pattern: fmt.Sprintf("%%term-%d%%", child), collation: "utf8_bin"}
						skipList := new(SkipList)
						childSize := attachTestSkipList(idx, &idx.baseState, cache, key, skipList)
						size += childSize
						if hotEvery > 0 && child%hotEvery == 0 {
							skipList.recordUse()
						} else {
							coldSize += skipListEntrySize(key, skipList)
						}
					}
					candidateBytes := idx.baseState.skipListPartialCandidateBytes
					b.StartTimer()
					result := idx.evict(evictPartial, size+1024, new([numEvictableTypes]int64))
					expectedFreed := coldSize + candidateBytes
					if !result.success || result.freedBytes != expectedFreed {
						b.Fatalf("partial result = %+v, want %d bytes", result, expectedFreed)
					}
				}
				b.ReportMetric(float64(childCount), "candidates/op")
			})
		}
	}
}

func BenchmarkLikeSkipListExactHitParallel(b *testing.B) {
	cache := new(sync.Map)
	key := skipListKey{pattern: "%carglass%", collation: "utf8_bin"}
	cache.Store(key, new(SkipList))
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, ok := loadCachedSkipList(cache, key); !ok {
				b.Fatal("exact cache entry disappeared")
			}
		}
	})
}

func BenchmarkLikeSkipListCacheAdmission(b *testing.B) {
	keys := make([]skipListKey, b.N)
	skipLists := make([]*SkipList, b.N)
	for i := range keys {
		keys[i] = skipListKey{pattern: "%term-" + strconv.Itoa(i) + "%", collation: "utf8_bin"}
		skipLists[i] = new(SkipList)
	}
	cache := new(sync.Map)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range keys {
		cache.Store(keys[i], skipLists[i])
	}
}
