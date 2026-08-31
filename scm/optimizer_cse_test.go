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

import "testing"

func countOptimizerCalls(expression Scmer, name string) int {
	if stripped, ok := scmerStripSourceInfo(expression); ok {
		expression = stripped
	}
	items, ok := scmerSlice(expression)
	if !ok || len(items) == 0 {
		return 0
	}
	count := 0
	if declaration := DeclarationForValue(items[0]); declaration != nil && declaration.Name == name {
		count++
	}
	if scmerIsSymbol(items[0], "quote") {
		return count
	}
	for _, item := range items {
		count += countOptimizerCalls(item, name)
	}
	return count
}

func optimizeCSETestProc(t *testing.T, source string) Scmer {
	t.Helper()
	environment := newOptimizerTestEnv()
	return Eval(Optimize(Read(t.Name(), source), environment, nil), environment)
}

func TestOptimizeSharesFoldableCallAcrossBeginExpressions(t *testing.T) {
	proc := optimizeCSETestProc(t, `(lambda (value enabled)
		(begin
			(define direct (equal? (toUpper value) "SELECT"))
			(define conditional (and enabled (equal? (toUpper value) "SELECT")))
			(define diagnostic (equal? (toUpper value) "EXPLAIN COMPILE"))
			(list direct conditional diagnostic)))`)
	if calls := countOptimizerCalls(proc.Proc().Body, "toUpper"); calls != 1 {
		t.Fatalf("optimized body contains %d toUpper calls, want 1: %s", calls, SerializeToString(proc.Proc().Body, nil))
	}
	fn := OptimizeProcToSerialFunction(proc)
	got := fn(NewString("select"), NewBool(true))
	want := NewSlice([]Scmer{NewBool(true), NewBool(true), NewBool(false)})
	if !Equal(got, want) {
		t.Fatalf("shared foldable result = %s, want %s; body: %s", String(got), String(want), SerializeToString(proc.Proc().Body, nil))
	}
}

func TestOptimizeDoesNotHoistFoldableCallFromLazyPath(t *testing.T) {
	proc := optimizeCSETestProc(t, `(lambda (value enabled)
		(begin
			(define conditional (and enabled (equal? (toUpper value) "SELECT")))
			(define direct (equal? (toUpper value) "SELECT"))
			(list conditional direct)))`)
	if calls := countOptimizerCalls(proc.Proc().Body, "toUpper"); calls != 2 {
		t.Fatalf("optimized body contains %d toUpper calls, want 2: %s", calls, SerializeToString(proc.Proc().Body, nil))
	}
	fn := OptimizeProcToSerialFunction(proc)
	got := fn(NewString("select"), NewBool(false))
	want := NewSlice([]Scmer{NewBool(false), NewBool(true)})
	if !Equal(got, want) {
		t.Fatalf("lazy foldable result = %s, want %s", String(got), String(want))
	}
}

func TestOptimizeDoesNotHoistFoldableCallAcrossEarlierArgument(t *testing.T) {
	proc := optimizeCSETestProc(t, `(lambda (prefix value)
		(begin
			(define combined (concat prefix (toUpper value)))
			(define direct (toUpper value))
			(list combined direct)))`)
	if calls := countOptimizerCalls(proc.Proc().Body, "toUpper"); calls != 2 {
		t.Fatalf("optimized body contains %d toUpper calls, want 2: %s", calls, SerializeToString(proc.Proc().Body, nil))
	}
}

func TestOptimizeDoesNotTreatShadowedCallableAsDeclaredFoldable(t *testing.T) {
	proc := optimizeCSETestProc(t, `(lambda (toUpper value)
		(begin
			(define first (toUpper value))
			(define second (toUpper value))
			(list first second)))`)
	fn := OptimizeProcToSerialFunction(proc)
	calls := 0
	callable := NewFunc(func(arguments ...Scmer) Scmer {
		calls++
		return arguments[0]
	})
	fn(callable, NewString("value"))
	if calls != 2 {
		t.Fatalf("shadowed callable ran %d times, want 2", calls)
	}
}

func TestOptimizeDoesNotShareFreshFoldableResult(t *testing.T) {
	proc := optimizeCSETestProc(t, `(lambda (value)
		(begin
			(define first (split value ","))
			(define second (split value ","))
			(list first second)))`)
	if calls := countOptimizerCalls(proc.Proc().Body, "split"); calls != 2 {
		t.Fatalf("optimized body contains %d split calls, want 2: %s", calls, SerializeToString(proc.Proc().Body, nil))
	}
}

func TestOptimizeLowersBooleanRegexMatch(t *testing.T) {
	proc := optimizeCSETestProc(t, `(lambda (value)
		(match value (regex "^\\s*SELECT\\b" _) true _ false))`)
	if calls := countOptimizerCalls(proc.Proc().Body, "match"); calls != 0 {
		t.Fatalf("optimized body retains match: %s", SerializeToString(proc.Proc().Body, nil))
	}
}

func TestOptimizeKeepsCapturingRegexMatch(t *testing.T) {
	proc := optimizeCSETestProc(t, `(lambda (value)
		(match value (regex "^(.*)$" _ captured) captured _ false))`)
	if calls := countOptimizerCalls(proc.Proc().Body, "match"); calls != 1 {
		t.Fatalf("capturing regex match was lowered: %s", SerializeToString(proc.Proc().Body, nil))
	}
}

func TestOptimizeBooleanRegexMatchPreservesTypeFailure(t *testing.T) {
	proc := optimizeCSETestProc(t, `(lambda (value)
		(match value (regex "^SELECT" _) true _ false))`)
	fn := OptimizeProcToSerialFunction(proc)
	defer func() {
		if recover() == nil {
			t.Fatal("lowered regex match accepted a non-string value")
		}
	}()
	fn(NewNil())
}

func BenchmarkFoldableCSEUpperClassifiers(b *testing.B) {
	environment := newOptimizerTestEnv()
	proc := Eval(Optimize(Read(b.Name(), `(lambda (value enabled)
		(begin
			(define direct (equal? (toUpper value) "SELECT * FROM ITEMS"))
			(define guarded (and enabled (equal? (toUpper value) "SELECT * FROM ITEMS")))
			(define diagnostic (equal? (toUpper value) "EXPLAIN COMPILE SELECT * FROM ITEMS"))
			(list direct guarded diagnostic)))`), environment, nil), environment)
	fn := OptimizeProcToSerialFunction(proc)
	value := NewString("select * from items")
	enabled := NewBool(true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fn(value, enabled)
	}
}

func BenchmarkOptimizedBooleanRegexMatch(b *testing.B) {
	environment := newOptimizerTestEnv()
	proc := Eval(Optimize(Read(b.Name(), `(lambda (value)
		(match value (regex "^\\s*SELECT\\b" _) true _ false))`), environment, nil), environment)
	fn := OptimizeProcToSerialFunction(proc)
	value := NewString("SELECT * FROM items WHERE id = 42")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fn(value)
	}
}
