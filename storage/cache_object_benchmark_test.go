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
