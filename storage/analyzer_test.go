/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

import (
	"testing"

	"github.com/launix-de/memcp/scm"
)

// buildProc constructs a Proc with the given param names and body AST.
func buildProc(params []string, body scm.Scmer) scm.Scmer {
	paramSlice := make([]scm.Scmer, len(params))
	for i, p := range params {
		paramSlice[i] = scm.NewSymbol(p)
	}
	return scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice(paramSlice),
		Body:   body,
		En:     &scm.Env{Vars: make(scm.Vars)},
	})
}

// TestBoundaryEqual verifies that equal? produces EqualMatcher.
func TestBoundaryEqual(t *testing.T) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal?"),
		scm.NewSymbol("x"),
		scm.NewString("hello"),
	})
	cond := buildProc([]string{"x"}, body)
	bounds := extractBoundaries([]string{"name"}, cond)
	if len(bounds) != 1 {
		t.Fatalf("expected 1 boundary, got %d", len(bounds))
	}
	if bounds[0].matcher.Kind() != "equal" {
		t.Errorf("expected equal matcher, got %q", bounds[0].matcher.Kind())
	}
	if bounds[0].col != "name" {
		t.Errorf("expected col 'name', got %q", bounds[0].col)
	}
	if bounds[0].lower.String() != "hello" {
		t.Errorf("expected lower 'hello', got %v", bounds[0].lower)
	}
}

// TestBoundaryRange verifies that < produces RangeMatcher.
func TestBoundaryRange(t *testing.T) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("<"),
		scm.NewSymbol("x"),
		scm.NewInt(100),
	})
	cond := buildProc([]string{"x"}, body)
	bounds := extractBoundaries([]string{"age"}, cond)
	if len(bounds) != 1 {
		t.Fatalf("expected 1 boundary, got %d", len(bounds))
	}
	if bounds[0].matcher.Kind() != "range" {
		t.Errorf("expected range matcher, got %q", bounds[0].matcher.Kind())
	}
}

// TestBoundaryLikePrefixIsRange verifies that prefix LIKE "foo%" becomes RangeMatcher.
func TestBoundaryLikePrefixIsRange(t *testing.T) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("strlike"),
		scm.NewSymbol("x"),
		scm.NewString("foo%"),
	})
	cond := buildProc([]string{"x"}, body)
	bounds := extractBoundaries([]string{"name"}, cond)
	if len(bounds) != 1 {
		t.Fatalf("expected 1 boundary, got %d", len(bounds))
	}
	if bounds[0].matcher.Kind() != "range" {
		t.Errorf("expected range matcher for prefix LIKE, got %q", bounds[0].matcher.Kind())
	}
}

// TestBoundaryLikeNonPrefix verifies that "%Klaus%" produces LikeMatcher.
func TestBoundaryLikeNonPrefix(t *testing.T) {
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("strlike"),
		scm.NewSymbol("x"),
		scm.NewString("%Klaus%"),
	})
	cond := buildProc([]string{"x"}, body)
	bounds := extractBoundaries([]string{"name"}, cond)
	if len(bounds) != 1 {
		t.Fatalf("expected 1 boundary, got %d", len(bounds))
	}
	if bounds[0].matcher.Kind() != "like" {
		t.Errorf("expected like matcher, got %q", bounds[0].matcher.Kind())
	}
	if !bounds[0].matcher.IsPointLike() {
		t.Error("LIKE matcher should be point-like")
	}
	if bounds[0].matcher.IsSorted() {
		t.Error("LIKE matcher should not be sorted")
	}
}

// TestMatcherIsPointLike verifies IsPointLike for all matcher types.
func TestMatcherIsPointLike(t *testing.T) {
	if !EqualMatcher.IsPointLike() {
		t.Error("EqualMatcher should be point-like")
	}
	if !LikeMatcher.IsPointLike() {
		t.Error("LikeMatcher should be point-like")
	}
	if RangeMatcher.IsPointLike() {
		t.Error("RangeMatcher should not be point-like")
	}
}

// TestMatcherIsSorted verifies IsSorted for all matcher types.
func TestMatcherIsSorted(t *testing.T) {
	if !EqualMatcher.IsSorted() {
		t.Error("EqualMatcher should be sorted")
	}
	if !RangeMatcher.IsSorted() {
		t.Error("RangeMatcher should be sorted")
	}
	if LikeMatcher.IsSorted() {
		t.Error("LikeMatcher should not be sorted")
	}
}

// TestRowWithinBoundsEqual verifies equalMatcherData.Match + Beyond.
func TestRowWithinBoundsEqual(t *testing.T) {
	idx := &StorageIndex{Cols: []string{"id"}, ColMatchers: []BoundaryMatcher{EqualMatcher}}
	data := []BoundaryMatcherData{EqualMatcher.ProduceData(idx, 0, scm.NewInt(5), scm.NewInt(5))}

	inRange, _ := idx.rowWithinBounds(1, data, func(i int) scm.Scmer { return scm.NewInt(5) })
	if !inRange {
		t.Error("expected match for equal value")
	}
	inRange, beyond := idx.rowWithinBounds(1, data, func(i int) scm.Scmer { return scm.NewInt(10) })
	if inRange {
		t.Error("expected no match for different value")
	}
	if !beyond {
		t.Error("expected beyond=true for value > equal point")
	}
}

// TestRowWithinBoundsLike verifies that LIKE columns are skipped in rowWithinBounds
// (non-sorted columns are handled by block-level skipping, scan() filters exact).
func TestRowWithinBoundsLike(t *testing.T) {
	idx := &StorageIndex{Cols: []string{"name"}, ColMatchers: []BoundaryMatcher{LikeMatcher}}
	data := []BoundaryMatcherData{LikeMatcher.ProduceData(idx, 0, scm.NewString("%Klaus%"), scm.NewString("%Klaus%"))}

	// rowWithinBounds always returns true for non-sorted columns (LIKE)
	inRange, _ := idx.rowWithinBounds(1, data, func(i int) scm.Scmer { return scm.NewString("Hans Klaus Müller") })
	if !inRange {
		t.Error("expected inRange=true (LIKE skipped in rowWithinBounds)")
	}
	inRange, beyond := idx.rowWithinBounds(1, data, func(i int) scm.Scmer { return scm.NewString("Hans Peter Müller") })
	if !inRange {
		t.Error("expected inRange=true (LIKE skipped, scan() filters)")
	}
	if beyond {
		t.Error("expected beyond=false for LIKE")
	}
}

// TestMatcherKindEqual verifies index deduplication by kind.
func TestMatcherKindEqual(t *testing.T) {
	if !matcherKindEqual(EqualMatcher, EqualMatcher) {
		t.Error("same matcher should be kind-equal")
	}
	if matcherKindEqual(EqualMatcher, LikeMatcher) {
		t.Error("different matchers should not be kind-equal")
	}
	if matcherKindEqual(RangeMatcher, LikeMatcher) {
		t.Error("range and like should not be kind-equal")
	}
}
