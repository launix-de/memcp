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
	"strings"
	"testing"
)

func TestOptimizeLowersListMembershipFolds(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		lowered string
	}{
		{
			name:    "subset",
			source:  `(lambda (required available) (reduce (coalesceNil required '()) (lambda (ok value) (and ok (contains? available value))) true))`,
			lowered: "list_contains_all",
		},
		{
			name:    "intersection",
			source:  `(lambda (left right) (reduce left (lambda (found value) (or found (contains? right value))) false))`,
			lowered: "list_contains_any",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			optimized, env := optimizeListPipeline(t, tc.source)
			serialized := serializedTestExpr(t, env, optimized)
			if !strings.Contains(serialized, tc.lowered) {
				t.Fatalf("membership fold was not lowered to %s: %s", tc.lowered, serialized)
			}
			fn := OptimizeProcToSerialFunction(Eval(optimized, env))
			got := fn(
				NewSlice([]Scmer{NewString("a"), NewString("c")}),
				NewSlice([]Scmer{NewString("a"), NewString("b"), NewString("c")}))
			if !got.Bool() {
				t.Fatalf("lowered membership fold returned %s", String(got))
			}
		})
	}
}

func TestListMembershipFoldPreservesCoerciveEquality(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (required available)
		(reduce required (lambda (ok value) (and ok (contains? available value))) true))`)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	required := make([]Scmer, 12)
	available := make([]Scmer, 16)
	for index := range available {
		available[index] = NewString(fmt.Sprintf("%d", index))
	}
	for index := range required {
		required[index] = NewInt(int64(index + 4))
	}
	if !fn(NewSlice(required), NewSlice(available)).Bool() {
		t.Fatal("hashed membership fold lost Equal's numeric/string coercion")
	}
}

func TestListMembershipFoldSemantics(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		input     []Scmer
		available []Scmer
		want      bool
	}{
		{
			name:      "all missing",
			source:    `(lambda (required available) (reduce required (lambda (ok value) (and ok (contains? available value))) true))`,
			input:     []Scmer{NewString("a"), NewString("c")},
			available: []Scmer{NewString("a"), NewString("b")},
		},
		{
			name:      "any missing",
			source:    `(lambda (left right) (reduce left (lambda (found value) (or found (contains? right value))) false))`,
			input:     []Scmer{NewString("c"), NewString("d")},
			available: []Scmer{NewString("a"), NewString("b")},
		},
		{
			name:      "all empty",
			source:    `(lambda (required available) (reduce required (lambda (ok value) (and ok (contains? available value))) true))`,
			available: []Scmer{NewString("a")},
			want:      true,
		},
		{
			name:      "any empty",
			source:    `(lambda (left right) (reduce left (lambda (found value) (or found (contains? right value))) false))`,
			available: []Scmer{NewString("a")},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			optimized, env := optimizeListPipeline(t, tc.source)
			fn := OptimizeProcToSerialFunction(Eval(optimized, env))
			got := fn(NewSlice(tc.input), NewSlice(tc.available)).Bool()
			if got != tc.want {
				t.Fatalf("got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestListMembershipFoldKeepsEmptyInputValidationOrder(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (required available)
		(reduce required (lambda (ok value) (and ok (contains? available value))) true))`)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	if !fn(NewSlice(nil), NewInt(1)).Bool() {
		t.Fatal("empty all-fold did not preserve its true neutral")
	}
}

func TestOptimizeKeepsComputedMembershipCaptureInsideReducer(t *testing.T) {
	optimized, env := optimizeListPipeline(t, `(lambda (required available)
		(reduce required (lambda (ok value) (and ok (contains? (car available) value))) true))`)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "list_contains_all") {
		t.Fatalf("computed callback capture was evaluated eagerly: %s", serialized)
	}
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	if !fn(NewSlice(nil), NewSlice(nil)).Bool() {
		t.Fatal("empty fold lost its neutral value")
	}
}

func BenchmarkPlannerJoinSetSubset(b *testing.B) {
	optimized, env := optimizeListPipeline(b, `(lambda (required available)
		(reduce (coalesceNil required '()) (lambda (ok alias) (and ok (contains? available alias))) true))`)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	required := make([]Scmer, 12)
	available := make([]Scmer, 16)
	for index := range available {
		available[index] = NewString(fmt.Sprintf("alias_%02d", index))
	}
	for index := range required {
		required[index] = available[index+4]
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !fn(NewSlice(required), NewSlice(available)).Bool() {
			b.Fatal("subset unexpectedly failed")
		}
	}
}
