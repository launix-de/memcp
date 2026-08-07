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
	"bytes"
	"strings"
	"testing"
	"unsafe"
)

func serializeSliceAllocTestExpr(t testing.TB, expr Scmer) string {
	t.Helper()
	var out bytes.Buffer
	Serialize(&out, expr, &Globalenv)
	return out.String()
}

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

func TestOptimizeGeneratedConsChainToList(t *testing.T) {
	expr := Read("generated cons chain", `(lambda (a b c) (cons a (cons b (cons c '()))))`)
	optimized := Optimize(expr, &Globalenv)
	serialized := serializeSliceAllocTestExpr(t, optimized)
	if strings.Contains(serialized, "cons") {
		t.Fatalf("generated cons chain was not flattened: %s", serialized)
	}

	result := Apply(Eval(optimized, &Globalenv), NewInt(1), NewInt(2), NewInt(3))
	expected := NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3)})
	if !Equal(result, expected) {
		t.Fatalf("unexpected result: got %s, want %s", String(result), String(expected))
	}
}

func TestOptimizeGeneratedConsChainWithListTail(t *testing.T) {
	expr := Read("generated cons chain with list tail", `(lambda (a b c) (cons a (cons b (list c 4))))`)
	optimized := Optimize(expr, &Globalenv)
	serialized := serializeSliceAllocTestExpr(t, optimized)
	if strings.Contains(serialized, "cons") {
		t.Fatalf("generated cons chain with list tail was not flattened: %s", serialized)
	}

	result := Apply(Eval(optimized, &Globalenv), NewInt(1), NewInt(2), NewInt(3))
	expected := NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3), NewInt(4)})
	if !Equal(result, expected) {
		t.Fatalf("unexpected result: got %s, want %s", String(result), String(expected))
	}
}

func TestOptimizePreservesSetInNumberedLambda(t *testing.T) {
	expr := Read("numbered set", `((lambda (counter) (!begin (set counter (+ counter 1)) counter) 1) 0)`)
	got := Eval(Optimize(expr, &Globalenv), &Globalenv)
	if ToInt(got) != 1 {
		t.Fatalf("optimized numbered lambda returned %s, want 1", got.String())
	}
}

func TestOptimizeImproperConsStaysCons(t *testing.T) {
	expr := Read("improper cons", `(lambda (tail) (cons 1 tail))`)
	optimized := Optimize(expr, &Globalenv)
	if serialized := serializeSliceAllocTestExpr(t, optimized); !strings.Contains(serialized, "cons") {
		t.Fatalf("improper cons was rewritten as a proper list: %s", serialized)
	}
}

func TestOptimizePreservesSetAndReadInNestedLambda(t *testing.T) {
	expr := Read("nested set", `((lambda (counter)
		((lambda () (!begin
			(set counter (+ counter 1))
			counter)))) 0)`)
	got := Eval(Optimize(expr, &Globalenv), &Globalenv)
	if ToInt(got) != 1 {
		t.Fatalf("optimized nested set returned %s, want 1", got.String())
	}
}

func benchmarkGeneratedConsChain(b *testing.B, width int) {
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		tail := NewSlice([]Scmer{NewSymbol("quote"), NewSlice(nil)})
		for i := width - 1; i >= 0; i-- {
			tail = NewSlice([]Scmer{NewSymbol("cons"), NewInt(int64(i)), tail})
		}
		Optimize(tail, &Globalenv)
	}
}

func BenchmarkOptimizeGeneratedConsChain8(b *testing.B) {
	benchmarkGeneratedConsChain(b, 8)
}

func BenchmarkOptimizeGeneratedConsChain32(b *testing.B) {
	benchmarkGeneratedConsChain(b, 32)
}

func BenchmarkOptimizeImproperCons(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		expr := NewSlice([]Scmer{NewSymbol("cons"), NewInt(1), NewSymbol("tail")})
		Optimize(expr, &Globalenv)
	}
}
