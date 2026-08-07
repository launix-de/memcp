/*
Copyright (C) 2026  Carl-Philip Haensch

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
	"fmt"
	"testing"
)

func nestedLexicalScopeExpr(depth int) Scmer {
	body := NewSlice([]Scmer{NewSymbol("+"), NewSymbol("captured"), NewInt(1)})
	for i := depth - 1; i >= 0; i-- {
		param := NewSymbol(fmt.Sprintf("local_%d", i))
		body = NewSlice([]Scmer{
			NewSymbol("lambda"),
			NewSlice([]Scmer{param}),
			body,
		})
	}
	return NewSlice([]Scmer{
		NewSymbol("begin"),
		NewSlice([]Scmer{NewSymbol("define"), NewSymbol("captured"), NewInt(41)}),
		body,
	})
}

func TestOptimizeNestedLambdaParametersDoNotHideOuterBinding(t *testing.T) {
	optimized := Optimize(nestedLexicalScopeExpr(4), &Globalenv)
	value := Eval(optimized, &Globalenv)
	for i := 0; i < 4; i++ {
		value = Apply(value, NewInt(int64(i)))
	}
	if got := ToInt(value); got != 42 {
		t.Fatalf("optimized nested closure returned %d, want 42", got)
	}
}

func TestOptimizeNestedLambdaRestoresRepeatedShadow(t *testing.T) {
	expr := Read("repeated nested shadow", `(begin
		(define value 40)
		(lambda (value)
			(lambda (value)
				(+ value 2))))`)
	optimized := Optimize(expr, &Globalenv)
	outer := Apply(Eval(optimized, &Globalenv), NewInt(10))
	if got := ToInt(Apply(outer, NewInt(20))); got != 22 {
		t.Fatalf("optimized nested shadow returned %d, want 22", got)
	}
}

func TestOptimizeNestedLexicalScopesAllocationBudget(t *testing.T) {
	allocs := testing.AllocsPerRun(5, func() {
		Optimize(nestedLexicalScopeExpr(128), &Globalenv)
	})
	if allocs >= 5000 {
		t.Fatalf("nested lexical optimization allocated %.0f objects, want fewer than 5000", allocs)
	}
}

func benchmarkOptimizeNestedLexicalScopes(b *testing.B, depth int) {
	b.ReportAllocs()
	for b.Loop() {
		Optimize(nestedLexicalScopeExpr(depth), &Globalenv)
	}
}

func BenchmarkOptimizeNestedLexicalScopes32(b *testing.B) {
	benchmarkOptimizeNestedLexicalScopes(b, 32)
}

func BenchmarkOptimizeNestedLexicalScopes64(b *testing.B) {
	benchmarkOptimizeNestedLexicalScopes(b, 64)
}

func BenchmarkOptimizeNestedLexicalScopes128(b *testing.B) {
	benchmarkOptimizeNestedLexicalScopes(b, 128)
}
