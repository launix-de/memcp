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

import "testing"

func TestOptimizeQuoteIsBorrowedPassthrough(t *testing.T) {
	ome := newOptimizerMetainfo()
	optimized, resultType := OptimizeEx(Read("test", "(quote ((1 2) (3 4)))"), &Globalenv, &ome, true)
	if resultType.Const() {
		t.Fatal("source-level quote must remain an optimizer barrier")
	}
	if resultType.Transfer() {
		t.Fatal("quoted list must not transfer ownership")
	}
	want := Eval(Read("test", "(quote ((1 2) (3 4)))"), &Globalenv)
	if got := Eval(optimized, &Globalenv); !Equal(got, want) {
		t.Fatalf("optimized quote evaluated to %s", String(got))
	}
}

func TestOptimizeConstantListUsesSingleQuote(t *testing.T) {
	nested := make([]Scmer, 1, constListQuoteThreshold)
	nested[0] = NewSymbol("list")
	for i := 0; i < constListQuoteThreshold; i++ {
		nested = append(nested, NewSlice([]Scmer{NewSymbol("list"), NewInt(int64(i))}))
	}
	ome := newOptimizerMetainfo()
	optimized, resultType := OptimizeEx(NewSlice(nested), &Globalenv, &ome, true)
	if !resultType.Const() || resultType.Transfer() {
		t.Fatalf("constant list type = const %t, transfer %t; want borrowed constant", resultType.Const(), resultType.Transfer())
	}
	items, ok := scmerSlice(optimized)
	if !ok || len(items) != 2 || !scmerIsSymbol(items[0], "quote") {
		t.Fatalf("optimized constant list = %s; want one quote wrapper", String(optimized))
	}
	if countQuoteForms(optimized) != 1 {
		t.Fatalf("optimized nested constant contains embedded quote forms: %s", String(optimized))
	}
	if got := Eval(optimized, &Globalenv); len(got.Slice()) != constListQuoteThreshold {
		t.Fatalf("optimized constant list has %d elements", len(got.Slice()))
	}
}

func TestWrapConstantListRetainsBackingStorage(t *testing.T) {
	literal := NewSlice(make([]Scmer, constListQuoteThreshold))
	td := &TypeDescriptor{Const: true, Transfer: true}
	wrapped := wrapConstListForCode(literal, td, false).Slice()
	if len(wrapped) != 2 || !scmerIsSymbol(wrapped[0], "quote") {
		t.Fatalf("wrapped constant = %s; want quote", String(NewSlice(wrapped)))
	}
	if td.Transfer {
		t.Fatal("quoted constant must not transfer ownership")
	}
	if wrapped[1].ptr != literal.ptr || wrapped[1].aux != literal.aux {
		t.Fatal("quote wrapper copied constant list storage")
	}
}

func TestWrapEmbeddedConstantListDoesNotAddQuotes(t *testing.T) {
	literal := NewSlice(make([]Scmer, constListQuoteThreshold))
	wrapped := wrapConstListForCode(literal, &TypeDescriptor{Const: true}, true)
	wrappedItems, ok := scmerSlice(wrapped)
	if !ok || len(wrappedItems) == 0 || !scmerIsSymbol(wrappedItems[0], "list") {
		t.Fatalf("embedded constant list = %s; want list constructor", String(wrapped))
	}
	if countQuoteForms(wrapped) != 0 {
		t.Fatalf("embedded constant list contains quote forms: %s", String(wrapped))
	}
}

func TestOptimizeSmallConstantListKeepsFreshOwnership(t *testing.T) {
	ome := newOptimizerMetainfo()
	optimized, resultType := OptimizeEx(Read("test", "(list 1 2)"), &Globalenv, &ome, true)
	items, ok := scmerSlice(optimized)
	if !ok || len(items) != 3 || !scmerIsSymbol(items[0], "list") {
		t.Fatalf("small constant list = %s; want list constructor", String(optimized))
	}
	if !resultType.Const() || !resultType.Transfer() {
		t.Fatalf("small constant list type = const %t, transfer %t; want fresh constant", resultType.Const(), resultType.Transfer())
	}
}

func TestOptimizeDoesNotFoldMutatingCallOnQuotedList(t *testing.T) {
	expr := Read("test", "(nth_mut (quote (1 2)) 0 9)")
	exprItems, _ := scmerSlice(expr)
	literal := Eval(exprItems[1], &Globalenv)
	literal, _ = scmerStripSourceInfo(literal)
	optimized := Optimize(expr, &Globalenv, nil)
	items, ok := scmerSlice(optimized)
	if !ok || len(items) == 0 || !scmerIsSymbol(items[0], "nth_mut") {
		t.Fatalf("mutating call was folded over shared literal: %s", String(optimized))
	}
	if got := literal.Slice()[0].Int(); got != 1 {
		t.Fatalf("optimizer mutated quoted literal to %d", got)
	}
}

func countQuoteForms(val Scmer) int {
	items, ok := scmerSlice(val)
	if !ok {
		return 0
	}
	count := 0
	if len(items) > 0 && scmerIsSymbol(items[0], "quote") {
		count++
	}
	for _, item := range items {
		count += countQuoteForms(item)
	}
	return count
}
