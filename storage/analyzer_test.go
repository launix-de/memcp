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

// TestBoundaryOpEqual verifies that equal? produces boundaryOpEqual.
func TestBoundaryOpEqual(t *testing.T) {
	// (equal? col "hello")
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
	if bounds[0].op != boundaryOpEqual {
		t.Errorf("expected boundaryOpEqual, got %d", bounds[0].op)
	}
	if bounds[0].col != "name" {
		t.Errorf("expected col 'name', got %q", bounds[0].col)
	}
	if bounds[0].lower.String() != "hello" {
		t.Errorf("expected lower 'hello', got %v", bounds[0].lower)
	}
}

// TestBoundaryOpRange verifies that < produces boundaryOpRange.
func TestBoundaryOpRange(t *testing.T) {
	// (< col 100)
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
	if bounds[0].op != boundaryOpRange {
		t.Errorf("expected boundaryOpRange, got %d", bounds[0].op)
	}
}

// TestBoundaryOpLikePrefixIsRange verifies that prefix LIKE "foo%" becomes a range.
func TestBoundaryOpLikePrefixIsRange(t *testing.T) {
	// (strlike col "foo%")
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
	if bounds[0].op != boundaryOpRange {
		t.Errorf("expected boundaryOpRange for prefix LIKE, got %d", bounds[0].op)
	}
	if bounds[0].lower.String() != "foo" {
		t.Errorf("expected lower 'foo', got %v", bounds[0].lower)
	}
}

// TestBoundaryOpLikeNonPrefix verifies that non-prefix LIKE "%Klaus%" produces boundaryOpLike.
func TestBoundaryOpLikeNonPrefix(t *testing.T) {
	// (strlike col "%Klaus%")
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
	if bounds[0].op != boundaryOpLike {
		t.Errorf("expected boundaryOpLike, got %d", bounds[0].op)
	}
	if bounds[0].col != "name" {
		t.Errorf("expected col 'name', got %q", bounds[0].col)
	}
	if bounds[0].lower.String() != "%Klaus%" {
		t.Errorf("expected pattern '%%Klaus%%', got %v", bounds[0].lower)
	}
}

// TestBoundaryOpLikeIsPointLike verifies that LIKE boundaries are treated as point-like
// for canonical index ordering (sorted before range columns).
func TestBoundaryOpLikeIsPointLike(t *testing.T) {
	if !boundaryIsPoint(columnboundaries{op: boundaryOpLike}) {
		t.Error("boundaryOpLike should be point-like")
	}
	if !boundaryIsPoint(columnboundaries{op: boundaryOpEqual}) {
		t.Error("boundaryOpEqual should be point-like")
	}
	if boundaryIsPoint(columnboundaries{op: boundaryOpRange}) {
		t.Error("boundaryOpRange should not be point-like")
	}
}

// TestRowWithinBoundsLike verifies that rowWithinBounds applies LIKE matching.
func TestRowWithinBoundsLike(t *testing.T) {
	idx := &StorageIndex{Cols: []string{"name"}}
	ops := []boundaryOp{boundaryOpLike}
	pattern := scm.NewString("%Klaus%")
	lower := []scm.Scmer{pattern}

	// matching row
	inRange, beyond := idx.rowWithinBounds(1, lower, pattern, true, ops, func(i int) scm.Scmer {
		return scm.NewString("Hans Klaus Müller")
	})
	if !inRange {
		t.Error("expected inRange=true for matching LIKE")
	}
	if beyond {
		t.Error("expected beyond=false for LIKE")
	}

	// non-matching row
	inRange, beyond = idx.rowWithinBounds(1, lower, pattern, true, ops, func(i int) scm.Scmer {
		return scm.NewString("Hans Peter Müller")
	})
	if inRange {
		t.Error("expected inRange=false for non-matching LIKE")
	}
	if beyond {
		t.Error("expected beyond=false for LIKE (no sort-order beyond)")
	}

	// nil value
	inRange, _ = idx.rowWithinBounds(1, lower, pattern, true, ops, func(i int) scm.Scmer {
		return scm.NewNil()
	})
	if inRange {
		t.Error("expected inRange=false for nil value with LIKE")
	}
}

// TestAddConstraintOpPromotion verifies that AND-merging promotes to the stronger op.
func TestAddConstraintOpPromotion(t *testing.T) {
	b1 := boundaries{columnboundaries{col: "name", op: boundaryOpRange, lower: scm.NewString("a"), upper: scm.NewString("z")}}
	b2 := columnboundaries{col: "name", op: boundaryOpEqual, lower: scm.NewString("hello"), upper: scm.NewString("hello")}
	result := addConstraint(b1, b2)
	if result[0].op != boundaryOpEqual {
		t.Errorf("expected boundaryOpEqual after promotion, got %d", result[0].op)
	}
}

// TestWidenBoundsOpDemotion verifies that OR-merging demotes to the weaker op.
func TestWidenBoundsOpDemotion(t *testing.T) {
	a := boundaries{columnboundaries{col: "name", op: boundaryOpEqual, lower: scm.NewString("hello"), upper: scm.NewString("hello")}}
	b := boundaries{columnboundaries{col: "name", op: boundaryOpLike, lower: scm.NewString("%test%"), upper: scm.NewString("%test%")}}
	result := widenBounds(a, b)
	if len(result) != 1 {
		t.Fatalf("expected 1 boundary, got %d", len(result))
	}
	if result[0].op != boundaryOpEqual {
		t.Errorf("expected boundaryOpEqual (weaker numerically) after demotion, got %d", result[0].op)
	}
}
