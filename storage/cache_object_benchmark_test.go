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
		b.Run(fmt.Sprintf("children-%d", childCount), func(b *testing.B) {
			idx := new(StorageIndex)
			cache := new(sync.Map)
			var size int64
			for child := 0; child < childCount; child++ {
				key := skipListKey{pattern: fmt.Sprintf("%%term-%d%%", child), collation: "utf8_bin"}
				skipList := new(SkipList)
				cache.Store(key, skipList)
				size += skipListEntrySize(key, skipList)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for iteration := 0; iteration < b.N; iteration++ {
				idx.baseState.skipLists = []*sync.Map{cache}
				idx.baseState.skipListBytes.Store(size)
				idx.skipListCacheBytes.Store(size)
				result := idx.evict(evictPartial, size+1024, new([numEvictableTypes]int64))
				if !result.success || result.freedBytes != size {
					b.Fatalf("partial result = %+v, want %d bytes", result, size)
				}
			}
			b.ReportMetric(float64(childCount), "children/op")
		})
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
