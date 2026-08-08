/*
Copyright (C) 2026  Carl-Philip Hänsch

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
package scm

import "sync"
import "sync/atomic"
import "testing"

func TestMemoizedFunctionUsesStructuralArgumentKeys(t *testing.T) {
	var calls atomic.Int64
	fn := NewFunc(func(args ...Scmer) Scmer {
		calls.Add(1)
		return NewInt(42)
	})
	memo := &memoizedFunction{}

	first := []Scmer{NewSlice([]Scmer{NewInt(1), NewString("x")})}
	second := []Scmer{NewSlice([]Scmer{NewInt(1), NewString("x")})}
	if got := memo.apply(fn, first); got.Int() != 42 {
		t.Fatalf("first result = %v, want 42", got)
	}
	if got := memo.apply(fn, second); got.Int() != 42 {
		t.Fatalf("cached result = %v, want 42", got)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls = %d, want 1", got)
	}
}

func TestMemoizedFunctionDoesNotCachePanics(t *testing.T) {
	var calls atomic.Int64
	fn := NewFunc(func(args ...Scmer) Scmer {
		if calls.Add(1) == 1 {
			panic("first call fails")
		}
		return NewInt(7)
	})
	memo := &memoizedFunction{}

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("first call did not panic")
			}
		}()
		memo.apply(fn, []Scmer{NewInt(1)})
	}()
	if got := memo.apply(fn, []Scmer{NewInt(1)}); got.Int() != 7 {
		t.Fatalf("retry result = %v, want 7", got)
	}
}

func TestMemoizedFunctionAllowsConcurrentMisses(t *testing.T) {
	fn := NewFunc(func(args ...Scmer) Scmer {
		return NewInt(args[0].Int() + 1)
	})
	memo := &memoizedFunction{}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := memo.apply(fn, []Scmer{NewInt(9)}); got.Int() != 10 {
				t.Errorf("result = %v, want 10", got)
			}
		}()
	}
	wg.Wait()
}

func TestMemoizedFunctionPromotesToFastDict(t *testing.T) {
	fn := NewFunc(func(args ...Scmer) Scmer {
		return NewInt(args[0].Int() * 2)
	})
	memo := &memoizedFunction{}

	for i := int64(0); i < 8; i++ {
		if got := memo.apply(fn, []Scmer{NewInt(i)}); got.Int() != i*2 {
			t.Fatalf("result for %d = %v, want %d", i, got, i*2)
		}
	}
	if memo.dict == nil {
		t.Fatal("cache did not promote to FastDict")
	}
	if memo.entries != nil {
		t.Fatalf("linear entries retained after promotion: %v", memo.entries)
	}
	if got := memo.apply(fn, []Scmer{NewInt(3)}); got.Int() != 6 {
		t.Fatalf("promoted cache result = %v, want 6", got)
	}
}
