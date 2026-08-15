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
	"strings"
	"testing"
)

var optimizedReducerBenchmarkResult Scmer

func ownedAppendReducerExpression() Scmer {
	return NewSlice([]Scmer{
		NewSymbol("lambda"),
		NewSlice([]Scmer{NewSymbol("acc"), NewSymbol("value")}),
		NewSlice([]Scmer{NewSymbol("append"), NewSymbol("acc"), NewSymbol("value")}),
	})
}

func optimizeReducerForTest(t *testing.T, source string, accumulator *TypeDescriptor, values ...*TypeDescriptor) (Scmer, *optimizerMetainfo) {
	t.Helper()
	ome := newOptimizerMetainfo()
	oc := &OptimizerContext{Env: &Globalenv, Ome: &ome}
	optimized, _ := oc.OptimizeReducerCallback(Read("reducer ownership test", source), accumulator, values...)
	return optimized, &ome
}

func TestOptimizeReducerCommitsMutationAfterStableOwnership(t *testing.T) {
	optimized, ome := optimizeReducerForTest(t,
		`(lambda (acc value) (append acc value))`,
		&TypeDescriptor{Kind: "list", Transfer: true, Length: UnknownLength},
		&TypeDescriptor{Kind: "any", Length: UnknownLength})
	serialized := serializedTestExpr(t, &Globalenv, optimized)
	if !strings.Contains(serialized, "append_mut") {
		t.Fatalf("stable owned reducer did not select append_mut: %s", serialized)
	}
	if ome.rewrite.callbackAnalyses != 0 || ome.rewrite.callbackClones != 0 {
		t.Fatalf("stable reducer used speculative work: analyses=%d clones=%d",
			ome.rewrite.callbackAnalyses, ome.rewrite.callbackClones)
	}
}

func TestOptimizeReducerRejectsMutationAfterOwnershipLoss(t *testing.T) {
	optimized, ome := optimizeReducerForTest(t,
		`(lambda (acc borrowed use_acc) (if use_acc (append acc 9) borrowed))`,
		&TypeDescriptor{Kind: "list", Transfer: true, Length: UnknownLength},
		&TypeDescriptor{Kind: "list", Length: UnknownLength},
		&TypeDescriptor{Kind: "bool", Length: UnknownLength})
	serialized := serializedTestExpr(t, &Globalenv, optimized)
	if strings.Contains(serialized, "append_mut") {
		t.Fatalf("reducer retained append_mut after loop-carried ownership was lost: %s", serialized)
	}
	if ome.rewrite.callbackAnalyses == 0 || ome.rewrite.callbackClones != 0 {
		t.Fatalf("ownership stabilization should iterate only over type facts: analyses=%d clones=%d",
			ome.rewrite.callbackAnalyses, ome.rewrite.callbackClones)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, &Globalenv))
	borrowedStorage := make([]Scmer, 1, 2)
	borrowedStorage[0] = NewInt(1)
	borrowed := NewSlice(borrowedStorage)
	first := fn(NewSlice(nil), borrowed, NewBool(false))
	second := fn(first, NewSlice(nil), NewBool(true))
	if got := borrowed.Slice(); len(got) != 1 || got[0].Int() != 1 {
		t.Fatalf("borrowed accumulator was mutated: %s", String(borrowed))
	}
	if got := second.Slice(); len(got) != 2 || got[0].Int() != 1 || got[1].Int() != 9 {
		t.Fatalf("unexpected reducer result: %s", String(second))
	}
}

func TestOptimizeReducerTreatsQuotedAccumulatorAsBorrowed(t *testing.T) {
	optimized, _ := optimizeReducerForTest(t,
		`(lambda (acc use_acc) (if use_acc (append acc 9) (quote (1))))`,
		&TypeDescriptor{Kind: "list", Transfer: true, Length: UnknownLength},
		&TypeDescriptor{Kind: "bool", Length: UnknownLength})
	serialized := serializedTestExpr(t, &Globalenv, optimized)
	if strings.Contains(serialized, "append_mut") {
		t.Fatalf("quoted reducer accumulator was treated as exclusively owned: %s", serialized)
	}
}

func BenchmarkOptimizeOwnedReducer(b *testing.B) {
	accumulator := &TypeDescriptor{Kind: "list", Transfer: true, Length: UnknownLength}
	value := &TypeDescriptor{Kind: "any", Length: UnknownLength}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ome := newOptimizerMetainfo()
		oc := &OptimizerContext{Env: &Globalenv, Ome: &ome}
		optimizedReducerBenchmarkResult, _ = oc.OptimizeReducerCallback(ownedAppendReducerExpression(), accumulator, value)
	}
}
