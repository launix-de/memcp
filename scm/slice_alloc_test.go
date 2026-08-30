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
	optimized := Optimize(lambdaExpr, &Globalenv, nil)
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
	optimized := Optimize(expr, &Globalenv, nil)
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
	optimized := Optimize(expr, &Globalenv, nil)
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
	got := Eval(Optimize(expr, &Globalenv, nil), &Globalenv)
	if ToInt(got) != 1 {
		t.Fatalf("optimized numbered lambda returned %s, want 1", got.String())
	}
}

func TestOptimizeImproperConsStaysCons(t *testing.T) {
	expr := Read("improper cons", `(lambda (tail) (cons 1 tail))`)
	optimized := Optimize(expr, &Globalenv, nil)
	if serialized := serializeSliceAllocTestExpr(t, optimized); !strings.Contains(serialized, "cons") {
		t.Fatalf("improper cons was rewritten as a proper list: %s", serialized)
	}
}

func TestOptimizePreservesSetAndReadInNestedLambda(t *testing.T) {
	expr := Read("nested set", `((lambda (counter)
		((lambda () (!begin
			(set counter (+ counter 1))
			counter)))) 0)`)
	got := Eval(Optimize(expr, &Globalenv, nil), &Globalenv)
	if ToInt(got) != 1 {
		t.Fatalf("optimized nested set returned %s, want 1", got.String())
	}
}

func TestOptimizeNumbersRepeatedLocalBinding(t *testing.T) {
	expr := Read("numbered local", `(lambda (value)
		(begin
			(define doubled (+ value value))
			(+ doubled doubled)))`)
	optimized := Optimize(expr, &Globalenv, nil)
	serialized := serializeSliceAllocTestExpr(t, optimized)
	if strings.Contains(serialized, "define doubled") {
		t.Fatalf("repeated local binding stayed symbolic: %s", serialized)
	}
	if !strings.Contains(serialized, "setN") {
		t.Fatalf("repeated local binding did not get a numbered slot: %s", serialized)
	}
	if got := Apply(Eval(optimized, &Globalenv), NewInt(3)); ToInt(got) != 12 {
		t.Fatalf("optimized local binding returned %s, want 12", String(got))
	}
}

func TestOptimizeSingleUseLocalTransfersFreshValue(t *testing.T) {
	expr := Read("single-use local ownership", `(lambda (value)
		(begin
			(define values (list value))
			(append values 2)))`)
	optimized := Optimize(expr, &Globalenv, nil)
	serialized := serializeSliceAllocTestExpr(t, optimized)
	if !strings.Contains(serialized, "append_mut") {
		t.Fatalf("single-use fresh local did not transfer ownership: %s", serialized)
	}
	if got := Apply(Eval(optimized, &Globalenv), NewInt(1)); !Equal(got, NewSlice([]Scmer{NewInt(1), NewInt(2)})) {
		t.Fatalf("optimized single-use binding returned %s", String(got))
	}
}

func TestOptimizeMultiUseLocalKeepsFreshValueBorrowed(t *testing.T) {
	expr := Read("multi-use local ownership", `(lambda (value)
		(begin
			(define values (list value))
			(list (append values 2) values)))`)
	optimized := Optimize(expr, &Globalenv, nil)
	serialized := serializeSliceAllocTestExpr(t, optimized)
	if !strings.Contains(serialized, "setN") {
		t.Fatalf("multi-use local binding was not retained: %s", serialized)
	}
	if strings.Contains(serialized, "append_mut") {
		t.Fatalf("multi-use local incorrectly transferred ownership: %s", serialized)
	}
	want := NewSlice([]Scmer{
		NewSlice([]Scmer{NewInt(1), NewInt(2)}),
		NewSlice([]Scmer{NewInt(1)}),
	})
	if got := Apply(Eval(optimized, &Globalenv), NewInt(1)); !Equal(got, want) {
		t.Fatalf("optimized multi-use binding returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeCapturedSingleUseLocalKeepsFreshValueBorrowed(t *testing.T) {
	expr := Read("captured single-use local ownership", `(lambda (value)
		(begin
			(define values (list value))
			(define extend (lambda () (append values 2)))
			(extend)))`)
	optimized := Optimize(expr, &Globalenv, nil)
	serialized := serializeSliceAllocTestExpr(t, optimized)
	if strings.Contains(serialized, "append_mut") {
		t.Fatalf("captured local incorrectly transferred ownership: %s", serialized)
	}
	if got := Apply(Eval(optimized, &Globalenv), NewInt(1)); !Equal(got, NewSlice([]Scmer{NewInt(1), NewInt(2)})) {
		t.Fatalf("optimized captured binding returned %s", String(got))
	}
}

func BenchmarkRunSingleUseLocalOwnership(b *testing.B) {
	expr := Read("single-use local ownership benchmark", `(lambda (value)
		(begin
			(define values (list value value value value value value value value
				value value value value value value value value))
			(filter values (lambda (value) (> value 0)))))`)
	optimized := Optimize(expr, &Globalenv, nil)
	if serialized := serializeSliceAllocTestExpr(b, optimized); !strings.Contains(serialized, "filter_mut") {
		b.Fatalf("single-use fresh local did not transfer ownership: %s", serialized)
	}
	fn := OptimizeProcToSerialFunction(Eval(optimized, &Globalenv))
	value := NewInt(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := fn(value); len(got.Slice()) != 16 {
			b.Fatalf("optimized filter returned %s", String(got))
		}
	}
}

func TestOptimizeNumbersLocalBindingsDeterministically(t *testing.T) {
	source := `(lambda (value)
		(begin
			(define first (+ value 1))
			(define second (+ value 2))
			(+ first first second second)))`
	want := serializeSliceAllocTestExpr(t, Optimize(Read("numbered local reference", source), &Globalenv, nil))
	for i := 0; i < 100; i++ {
		got := serializeSliceAllocTestExpr(t, Optimize(Read("numbered local repeat", source), &Globalenv, nil))
		if got != want {
			t.Fatalf("numbered local slots changed between runs:\nwant %s\n got %s", want, got)
		}
	}
}

func TestOptimizeExtendsExplicitNumVarsForLocalBinding(t *testing.T) {
	expr := Read("numbered local explicit frame", `(lambda (value)
		(begin
			(define doubled (+ value value))
			(+ doubled doubled))
		1)`)
	optimized := Optimize(expr, &Globalenv, nil)
	items := optimized.Slice()
	if len(items) != 4 || ToInt(items[3]) != 2 {
		t.Fatalf("explicit frame was not extended for local binding: %s", serializeSliceAllocTestExpr(t, optimized))
	}
	if got := Apply(Eval(optimized, &Globalenv), NewInt(4)); ToInt(got) != 16 {
		t.Fatalf("optimized explicit-frame binding returned %s, want 16", String(got))
	}
}

func TestOptimizeKeepsEvalVisibleBindingNamed(t *testing.T) {
	expr := Read("dynamic local", `(lambda (value)
		(begin
			(define dynamic_value (+ value value))
			(+ dynamic_value (eval 'dynamic_value))))`)
	optimized := Optimize(expr, &Globalenv, nil)
	serialized := serializeSliceAllocTestExpr(t, optimized)
	if !strings.Contains(serialized, "define dynamic_value") {
		t.Fatalf("eval-visible local binding was numbered: %s", serialized)
	}
	if got := Apply(Eval(optimized, &Globalenv), NewInt(5)); ToInt(got) != 20 {
		t.Fatalf("optimized eval-visible binding returned %s, want 20", String(got))
	}
}

func TestOptimizeKeepsParserVisibleBindingNamed(t *testing.T) {
	expr := Read("dynamic parser local", `(lambda ()
		(begin
			(define parser_rule (parser (atom "FOO" true) "inner"))
			(define parser_value (parser parser_rule "ok"))
			(list parser_rule parser_value)))`)
	optimized := Optimize(expr, &Globalenv, nil)
	serialized := serializeSliceAllocTestExpr(t, optimized)
	if !strings.Contains(serialized, "define parser_rule") {
		t.Fatalf("parser-visible local binding was numbered: %s", serialized)
	}
	result := Apply(Eval(optimized, &Globalenv))
	if len(result.Slice()) != 2 || !result.Slice()[1].IsParser() {
		t.Fatalf("optimized parser-visible binding returned %s", String(result))
	}
}

func TestOptimizeKeepsForwardReferencedBindingNamed(t *testing.T) {
	expr := Read("forward local", `(lambda (value)
		(begin
			(define read_later (lambda () later))
			(define later (+ value value))
			(+ (read_later) later)))`)
	optimized := Optimize(expr, &Globalenv, nil)
	serialized := serializeSliceAllocTestExpr(t, optimized)
	if !strings.Contains(serialized, "define later") {
		t.Fatalf("forward-referenced local binding was numbered: %s", serialized)
	}
	if got := Apply(Eval(optimized, &Globalenv), NewInt(6)); ToInt(got) != 24 {
		t.Fatalf("optimized forward binding returned %s, want 24", String(got))
	}
}

func TestOptimizeKeepsInitializerReferenceNamed(t *testing.T) {
	expr := Read("initializer local", `(lambda (value)
		(begin
			(define current (+ current value))
			current))`)
	optimized := Optimize(expr, &Globalenv, nil)
	serialized := serializeSliceAllocTestExpr(t, optimized)
	if !strings.Contains(serialized, "define current") {
		t.Fatalf("initializer-referenced local binding was numbered: %s", serialized)
	}
}

func TestOptimizeKeepsMultiplyDefinedBindingNamed(t *testing.T) {
	expr := Read("redefined local", `(lambda (value)
		(begin
			(define current (+ value 1))
			(define before current)
			(set current (+ value 2))
			(+ before current)))`)
	optimized := Optimize(expr, &Globalenv, nil)
	serialized := serializeSliceAllocTestExpr(t, optimized)
	if !strings.Contains(serialized, "define current") || !strings.Contains(serialized, "set current") {
		t.Fatalf("multiply defined local binding was numbered: %s", serialized)
	}
	if got := Apply(Eval(optimized, &Globalenv), NewInt(3)); ToInt(got) != 9 {
		t.Fatalf("optimized redefined binding returned %s, want 9", String(got))
	}
}

func TestOptimizeNumberedLocalSurvivesClosureCapture(t *testing.T) {
	expr := Read("captured numbered local", `(lambda (value)
		(begin
			(define doubled (+ value value))
			(define read_doubled (lambda () doubled))
			(+ doubled (read_doubled))))`)
	optimized := Optimize(expr, &Globalenv, nil)
	serialized := serializeSliceAllocTestExpr(t, optimized)
	if !strings.Contains(serialized, "setN") || !strings.Contains(serialized, "outer") {
		t.Fatalf("captured local did not use a numbered outer slot: %s", serialized)
	}
	if got := Apply(Eval(optimized, &Globalenv), NewInt(7)); ToInt(got) != 28 {
		t.Fatalf("optimized captured binding returned %s, want 28", String(got))
	}
}

func TestOptimizeNumbersLocalInsideBeginMutReserve(t *testing.T) {
	expr := Read("begin mut local", `(lambda (value)
		(begin_mut 1
			(define doubled (+ value value))
			(+ doubled doubled)))`)
	optimized := Optimize(expr, &Globalenv, nil)
	serialized := serializeSliceAllocTestExpr(t, optimized)
	if !strings.Contains(serialized, "setN") {
		t.Fatalf("begin_mut local binding was not numbered: %s", serialized)
	}
	if got := Apply(Eval(optimized, &Globalenv), NewInt(8)); ToInt(got) != 32 {
		t.Fatalf("optimized begin_mut binding returned %s, want 32", String(got))
	}
}

func benchmarkRepeatedLocalBindings(b *testing.B, serial bool) {
	expr := Read("numbered local benchmark", `(lambda (value)
		(begin
			(define v0 (+ value 1))
			(define v1 (+ value 2))
			(define v2 (+ value 3))
			(define v3 (+ value 4))
			(define v4 (+ value 5))
			(define v5 (+ value 6))
			(define v6 (+ value 7))
			(define v7 (+ value 8))
			(+ v0 v0 v1 v1 v2 v2 v3 v3 v4 v4 v5 v5 v6 v6 v7 v7)))`)
	proc := Eval(Optimize(expr, &Globalenv, nil), &Globalenv)
	value := NewInt(3)
	b.ReportAllocs()
	b.ResetTimer()
	if serial {
		fn := OptimizeProcToSerialFunction(proc)
		for i := 0; i < b.N; i++ {
			fn(value)
		}
		return
	}
	for i := 0; i < b.N; i++ {
		Apply(proc, value)
	}
}

func BenchmarkRepeatedLocalBindingsApply(b *testing.B) {
	benchmarkRepeatedLocalBindings(b, false)
}

func BenchmarkRepeatedLocalBindingsSerial(b *testing.B) {
	benchmarkRepeatedLocalBindings(b, true)
}

func benchmarkGeneratedConsChain(b *testing.B, width int) {
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		tail := NewSlice([]Scmer{NewSymbol("quote"), NewSlice(nil)})
		for i := width - 1; i >= 0; i-- {
			tail = NewSlice([]Scmer{NewSymbol("cons"), NewInt(int64(i)), tail})
		}
		Optimize(tail, &Globalenv, nil)
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
		Optimize(expr, &Globalenv, nil)
	}
}
