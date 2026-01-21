/*
Copyright (C) 2024  Carl-Philip Hänsch

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

package NonLockingReadMap

import "sort"
import "sync/atomic"
import "unsafe"

import "golang.org/x/exp/constraints"

/*
this is a read optimized map.

properties of this map:
 - read in O(log(N))
 - read is always nonblocking
 - write in O(N*log(N))
 - write is optimistic, worst case is a eternal loop (with a probability of 0%)
 - use this map if you read often but write very seldom
 - internally, a ordered list is rebuilt each time there is a write

*/

type Sizable interface {
	ComputeSize() uint
}

type KeyGetter[TK constraints.Ordered] interface {
	Sizable
	GetKey() TK
}

type NonLockingReadMap[T KeyGetter[TK], TK constraints.Ordered] struct {
	p atomic.Pointer[[]*T]
}

func New[T KeyGetter[TK], TK constraints.Ordered]() NonLockingReadMap[T, TK] {
	var result NonLockingReadMap[T, TK]
	result.p.Store(new([]*T))
	return result
}

func (b NonLockingReadMap[T, TK]) ComputeSize() uint {
	dataptr := b.p.Load()
	var sz uint = 16 /* allocation of struct */ + 8 /* atomic pointer */ + 16 /* allocation of slice */ + 24 /* slice */ + 8*uint(len(*dataptr)) /* slice storage */
	for _, v := range *dataptr {
		sz += (*v).ComputeSize()
	}
	return sz
}

func (m NonLockingReadMap[T, TK]) GetAll() []*T {
	return *m.p.Load()
}

func (m NonLockingReadMap[T, TK]) Get(key TK) *T {
	v, _, _ := m.FindItem(key)
	return v
}

func (m NonLockingReadMap[T, TK]) FindItem(key TK) (*T, int, *[]*T) {
	items := m.p.Load()
	lower := 0
	upper := len(*items)
	for {
		if lower == upper {
			return nil, -1, items
		}
		pivot := (lower + upper) / 2
		item := (*items)[pivot]
		itemkey := (*item).GetKey()
		if key == itemkey {
			return item, pivot, items
		} else if key < itemkey {
			upper = pivot
		} else {
			lower = pivot + 1
		}
	}
}

func (m *NonLockingReadMap[T, TK]) Set(v *T) *T {
restart:
	item, pivot, handle := m.FindItem((*v).GetKey())

	if pivot != -1 {
		if !atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&(*handle)[pivot])), unsafe.Pointer(item), unsafe.Pointer(v)) {
			goto restart
		}
		if !m.p.CompareAndSwap(handle, handle) {
			goto restart
		}
		return item
	}

	newhandle := new([]*T)
	*newhandle = make([]*T, 0, len(*handle)+1)
	*newhandle = append(*newhandle, (*handle)...)
	*newhandle = append(*newhandle, v)
	sort.Slice(*newhandle, func(i, j int) bool {
		return (*(*newhandle)[i]).GetKey() < (*(*newhandle)[j]).GetKey()
	})
	if !m.p.CompareAndSwap(handle, newhandle) {
		goto restart
	}
	return nil
}

func (m *NonLockingReadMap[T, TK]) Remove(key TK) *T {
restart:
	item, pivot, handle := m.FindItem(key)
	if pivot == -1 {
		return nil
	}
	newhandle := new([]*T)
	*newhandle = make([]*T, 0, len(*handle)-1)
	*newhandle = append(*newhandle, (*handle)[:pivot]...)
	*newhandle = append(*newhandle, (*handle)[pivot+1:]...)
	if !m.p.CompareAndSwap(handle, newhandle) {
		goto restart
	}
	return item
}
