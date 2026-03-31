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

import (
	"testing"
	"unsafe"
)

func TestScmerSlicePreservesCapacity(t *testing.T) {
	buf := make([]Scmer, 0, 1024)
	v := NewSlice(buf)
	got := v.Slice()
	if len(got) != 0 {
		t.Fatalf("expected len=0, got %d", len(got))
	}
	if cap(got) != 1024 {
		t.Fatalf("expected cap=1024, got %d", cap(got))
	}
}

func TestAppendMutReusesReservedCapacity(t *testing.T) {
	base := NewSlice(make([]Scmer, 0, 4))
	before := unsafe.Pointer(unsafe.SliceData(base.Slice()))
	result := Apply(Globalenv.Vars[Symbol("append_mut")], base, NewInt(1))
	got := result.Slice()
	if len(got) != 1 {
		t.Fatalf("expected len=1, got %d", len(got))
	}
	if cap(got) != 4 {
		t.Fatalf("expected cap=4, got %d", cap(got))
	}
	after := unsafe.Pointer(unsafe.SliceData(got))
	if before != after {
		t.Fatalf("append_mut reallocated backing storage")
	}
}

func TestDoubleBangListHeapFallback(t *testing.T) {
	expr := NewSlice([]Scmer{NewSymbol("!!list"), NewInt(3)})
	result := Eval(expr, &Globalenv)
	got := result.Slice()
	if len(got) != 0 {
		t.Fatalf("expected len=0, got %d", len(got))
	}
	if cap(got) != 3 {
		t.Fatalf("expected cap=3, got %d", cap(got))
	}
}

func TestOptimizeDoubleBangListAllocatesSlots(t *testing.T) {
	lambdaExpr := NewSlice([]Scmer{
		NewSymbol("lambda"),
		NewSlice([]Scmer{}),
		NewSlice([]Scmer{NewSymbol("!!list"), NewInt(4)}),
	})
	optimized := Optimize(lambdaExpr, &Globalenv)
	items := optimized.Slice()
	if len(items) != 4 {
		t.Fatalf("expected optimized lambda with NumVars, got %v", optimized)
	}
	if !items[3].IsInt() || items[3].Int() != 4 {
		t.Fatalf("expected NumVars=4, got %v", items[3])
	}
	body := items[2].Slice()
	if len(body) != 3 || !body[0].IsSymbol() || body[0].String() != "!!list" || !body[1].IsNthLocalVar() || body[1].NthLocalVar() != 0 || !body[2].IsInt() || body[2].Int() != 4 {
		t.Fatalf("unexpected optimized !!list body: %v", items[2])
	}

	proc := Eval(optimized, &Globalenv)
	result := Apply(proc)
	got := result.Slice()
	if len(got) != 0 {
		t.Fatalf("expected len=0, got %d", len(got))
	}
	if cap(got) != 4 {
		t.Fatalf("expected cap=4, got %d", cap(got))
	}
}
