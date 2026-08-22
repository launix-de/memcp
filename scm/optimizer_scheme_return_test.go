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
	"bytes"
	"strings"
	"testing"
)

func newOptimizerTestEnv() *Env {
	return &Env{Vars: make(Vars), Outer: &Globalenv}
}

func optimizeTestSource(t testing.TB, env *Env, source string) Scmer {
	t.Helper()
	return Optimize(Read("optimizer return test", source), env, nil)
}

func serializedTestExpr(t testing.TB, env *Env, expr Scmer) string {
	t.Helper()
	var out bytes.Buffer
	Serialize(&out, expr, env)
	return out.String()
}

func TestSchemeHelperFreshReturnEnablesMutRewrite(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define fresh_pair (lambda (a b) (list a b)))`, env)

	optimized := optimizeTestSource(t, env, `(append (fresh_pair 1 2) 3)`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "append_mut") {
		t.Fatalf("fresh Scheme helper did not enable append_mut: %s", serialized)
	}
	if got := Eval(optimized, env); !Equal(got, NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3)})) {
		t.Fatalf("optimized helper call returned %s", String(got))
	}
}

func TestSchemeHelperReturnLengthPropagates(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define fresh_pair_len (lambda (a b) (list a b)))`, env)

	optimized := optimizeTestSource(t, env, `(count (fresh_pair_len 1 2))`)
	if !optimized.IsInt() || optimized.Int() != 2 {
		t.Fatalf("expected exact helper return length to fold count, got %s", serializedTestExpr(t, env, optimized))
	}
}

func TestSchemeHelperReturnHintFollowsEnvironmentChain(t *testing.T) {
	parent := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define inherited_pair (lambda (a b) (list a b)))`, parent)
	child := &Env{Vars: make(Vars), Outer: parent}

	optimized := optimizeTestSource(t, child, `(append (inherited_pair 1 2) 3)`)
	if serialized := serializedTestExpr(t, child, optimized); !strings.Contains(serialized, "append_mut") {
		t.Fatalf("inherited helper hint did not enable append_mut: %s", serialized)
	}
}

func TestSchemeHelperLocalBeginDoesNotReplaceOuterHint(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define outer_pair (lambda (a b) (list a b)))`, env)
	optimizeTestSource(t, env, `(begin (define outer_pair (lambda (value) value)) (outer_pair 1))`)

	optimized := optimizeTestSource(t, env, `(append (outer_pair 1 2) 3)`)
	if serialized := serializedTestExpr(t, env, optimized); !strings.Contains(serialized, "append_mut") {
		t.Fatalf("local begin replaced outer helper hint: %s", serialized)
	}
}

func TestSchemeHelperSharedReturnStaysImmutable(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define shared_pair (list 1 2))`, env)
	EvalAll("optimizer return test", `(define return_shared_pair (lambda () shared_pair))`, env)

	optimized := optimizeTestSource(t, env, `(append (return_shared_pair) 3)`)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "append_mut") {
		t.Fatalf("shared Scheme helper return enabled append_mut: %s", serialized)
	}
	Eval(optimized, env)
	if got := env.FindRead("shared_pair").Vars["shared_pair"]; len(got.Slice()) != 2 {
		t.Fatalf("shared helper result was mutated: %s", String(got))
	}
}

func TestSchemeHelperRecursiveReturnStaysConservative(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define recursive_pair (lambda (n) (if (= n 0) (list n) (recursive_pair (- n 1)))))`, env)

	optimized := optimizeTestSource(t, env, `(append (recursive_pair 2) 3)`)
	if serialized := serializedTestExpr(t, env, optimized); strings.Contains(serialized, "append_mut") {
		t.Fatalf("recursive Scheme helper enabled append_mut: %s", serialized)
	}
}

func TestSchemeHelperEvalReturnStaysConservative(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define evaluated_pair (lambda () (eval '(list 1 2))))`, env)

	optimized := optimizeTestSource(t, env, `(append (evaluated_pair) 3)`)
	if serialized := serializedTestExpr(t, env, optimized); strings.Contains(serialized, "append_mut") {
		t.Fatalf("eval-using Scheme helper enabled append_mut: %s", serialized)
	}
}

func TestSchemeHelperDynamicRebindInvalidatesReturnInfo(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return test", `(define rebound_pair (lambda () (list 1 2)))`, env)
	EvalAll("optimizer return test", `(define rebound_pair (lambda (value) value))`, env)

	optimized := optimizeTestSource(t, env, `(append (rebound_pair (list 1 2)) 3)`)
	if serialized := serializedTestExpr(t, env, optimized); strings.Contains(serialized, "append_mut") {
		t.Fatalf("dynamically rebound Scheme helper kept stale ownership: %s", serialized)
	}
}

func TestSchemeReturnHintsPreserveNestedMutRewrite(t *testing.T) {
	env := newOptimizerTestEnv()
	optimized := optimizeTestSource(t, env, `(set_assoc (filter (list) (lambda (x) true)) "k" "v")`)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "set_assoc_mut") {
		t.Fatalf("nested fresh result did not enable set_assoc_mut: %s", serialized)
	}
}

func TestOptimizeInlinesHelperWithCaptureFreeNestedLambda(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("nested lambda inline test", `(define filter_owned (lambda (values)
		(filter values (lambda (x) (> x 1)))))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (a b c)
		(filter_owned (list a b c)))`)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "filter_owned") {
		t.Fatalf("helper with capture-free nested lambda was not inlined: %s", serialized)
	}
	if !strings.Contains(serialized, "!list") && !strings.Contains(serialized, "filter_mut") {
		t.Fatalf("inlined helper did not reuse or stack-allocate its input: %s", serialized)
	}
	if got := Apply(Eval(optimized, env), NewInt(0), NewInt(2), NewInt(3)); !Equal(got, NewSlice([]Scmer{NewInt(2), NewInt(3)})) {
		t.Fatalf("inlined helper returned %s", String(got))
	}
}

func TestInlinedNestedLambdaHelperDoesNotMutateBorrowedParameter(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("nested lambda inline test", `(define filter_borrowed (lambda (values)
		(filter values (lambda (x) (> x 1)))))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (values) (filter_borrowed values))`)
	if serialized := serializedTestExpr(t, env, optimized); strings.Contains(serialized, "filter_mut") {
		t.Fatalf("borrowed helper parameter selected filter_mut: %s", serialized)
	}
	shared := NewSlice([]Scmer{NewInt(0), NewInt(2), NewInt(3)})
	if got := Apply(Eval(optimized, env), shared); !Equal(got, NewSlice([]Scmer{NewInt(2), NewInt(3)})) {
		t.Fatalf("borrowed helper returned %s", String(got))
	}
	if !Equal(shared, NewSlice([]Scmer{NewInt(0), NewInt(2), NewInt(3)})) {
		t.Fatalf("borrowed helper mutated its input: %s", String(shared))
	}
}

func TestOptimizeKeepsNestedLambdaHelperWithOuterCapture(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("nested lambda inline test", `(define add_offset (lambda (offset values)
		(map values (lambda (value) (+ value offset)))))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (values) (add_offset 2 values))`)
	if serialized := serializedTestExpr(t, env, optimized); !strings.Contains(serialized, "add_offset") {
		t.Fatalf("helper with nested outer capture was inlined: %s", serialized)
	}
	got := Apply(Eval(optimized, env), NewSlice([]Scmer{NewInt(1), NewInt(3)}))
	if !Equal(got, NewSlice([]Scmer{NewInt(3), NewInt(5)})) {
		t.Fatalf("capturing helper returned %s", String(got))
	}
}

func TestOptimizeKeepsNestedLambdaHelperFromDifferentEnvironment(t *testing.T) {
	source := newOptimizerTestEnv()
	EvalAll("nested lambda inline test", `(define nested_captured_value 7)`, source)
	EvalAll("nested lambda inline test", `(define nested_captured_helper (lambda (values)
		(map values (lambda (value) (+ value nested_captured_value)))))`, source)

	target := newOptimizerTestEnv()
	target.Vars[Symbol("nested_captured_value")] = NewInt(9)
	target.Vars[Symbol("nested_captured_helper")] = source.Vars[Symbol("nested_captured_helper")]
	optimized := optimizeTestSource(t, target, `(lambda (values) (nested_captured_helper values))`)
	if serialized := serializedTestExpr(t, target, optimized); !strings.Contains(serialized, "nested_captured_helper") {
		t.Fatalf("nested helper from a different environment was inlined: %s", serialized)
	}
	got := Apply(Eval(optimized, target), NewSlice([]Scmer{NewInt(1)}))
	if !Equal(got, NewSlice([]Scmer{NewInt(8)})) {
		t.Fatalf("nested helper used target capture: %s", String(got))
	}
}

func TestNestedLambdaCaptureAnalysisHandlesCycles(t *testing.T) {
	items := make([]Scmer, 2)
	cycle := NewSlice(items)
	items[0] = NewSymbol("lambda")
	items[1] = cycle
	if expressionContainsOuterReference(cycle) {
		t.Fatal("capture analysis treated a small capture-free cycle as an outer reference")
	}
}

func TestOptimizeInlinesSingleUseLeafProc(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("leaf inline test", `(define leaf_second (lambda (values) (nth values 1)))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (values) (leaf_second values))`)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "leaf_second") {
		t.Fatalf("single-use leaf proc was not inlined: %s", serialized)
	}
	got := Apply(Eval(optimized, env), NewSlice([]Scmer{NewInt(4), NewInt(9)}))
	if ToInt(got) != 9 {
		t.Fatalf("inlined leaf proc returned %s, want 9", String(got))
	}
}

func TestOptimizeKeepsMultiUseLeafProcCall(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("leaf inline test", `(define leaf_twice (lambda (value) (+ value value)))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (value) (leaf_twice value))`)
	if serialized := serializedTestExpr(t, env, optimized); !strings.Contains(serialized, "leaf_twice") {
		t.Fatalf("multi-use leaf proc was inlined: %s", serialized)
	}
}

func TestOptimizeKeepsRecursiveLeafProcCall(t *testing.T) {
	env := newOptimizerTestEnv()
	EvalAll("leaf inline test", `(define leaf_recurse (lambda (value)
		(if (equal? value 0) 0 (leaf_recurse (- value 1)))))`, env)

	optimized := optimizeTestSource(t, env, `(lambda (value) (leaf_recurse value))`)
	if serialized := serializedTestExpr(t, env, optimized); !strings.Contains(serialized, "leaf_recurse") {
		t.Fatalf("recursive leaf proc was inlined: %s", serialized)
	}
}

func TestOptimizeKeepsLeafProcWithDifferentCapturedBinding(t *testing.T) {
	source := newOptimizerTestEnv()
	EvalAll("leaf inline test", `(define captured_value 7)`, source)
	proc := Eval(Optimize(Read("leaf inline test", `(lambda (value) (+ value captured_value))`), source, nil), source)

	target := newOptimizerTestEnv()
	target.Vars[Symbol("captured_value")] = NewInt(9)
	target.Vars[Symbol("captured_leaf")] = proc
	optimized := optimizeTestSource(t, target, `(lambda (value) (captured_leaf value))`)
	if serialized := serializedTestExpr(t, target, optimized); !strings.Contains(serialized, "captured_leaf") {
		t.Fatalf("leaf proc with a different captured binding was inlined: %s", serialized)
	}
}

func BenchmarkOptimizeSchemeHelperFreshReturn(b *testing.B) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return benchmark", `(define benchmark_fresh_pair (lambda (a b) (list a b)))`, env)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		expr := NewSlice([]Scmer{
			NewSymbol("append"),
			NewSlice([]Scmer{NewSymbol("benchmark_fresh_pair"), NewInt(1), NewInt(2)}),
			NewInt(3),
		})
		Optimize(expr, env, nil)
	}
}

func BenchmarkSchemeHelperOwnedReturnAppend(b *testing.B) {
	env := newOptimizerTestEnv()
	EvalAll("optimizer return benchmark", `(define benchmark_filtered (lambda (a b c d) (filter (list a b c d) (lambda (x) (> x 1)))))`, env)
	optimized := optimizeTestSource(b, env, `(lambda (a b c d e) (append (benchmark_filtered a b c d) e))`)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	args := []Scmer{NewInt(0), NewInt(2), NewInt(3), NewInt(4), NewInt(5)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := fn(args...)
		if len(result.Slice()) != 4 {
			b.Fatal("unexpected result length")
		}
	}
}

func BenchmarkOptimizeInlinedNestedLambdaHelper(b *testing.B) {
	env := newOptimizerTestEnv()
	EvalAll("nested lambda inline benchmark", `(define benchmark_filter_owned (lambda (values) (filter values (lambda (x) (> x 1)))))`, env)
	optimized := optimizeTestSource(b, env, `(lambda (a b c d e)
		(append (benchmark_filter_owned (list a b c d)) e))`)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	args := []Scmer{NewInt(0), NewInt(2), NewInt(3), NewInt(4), NewInt(5)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := fn(args...)
		if len(result.Slice()) != 4 {
			b.Fatal("unexpected result length")
		}
	}
}
