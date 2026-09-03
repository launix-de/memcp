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
	"fmt"
	"runtime"
	"testing"
)

var jitListBenchmarkSink Scmer

func TestJITPreviewCallArgumentResolvesSymbols(t *testing.T) {
	local := JITValueDesc{Loc: LocStackPair, Type: tagFastDict, StackOff: 32}
	ctx := &JITContext{Env: &JITEnv{Vars: map[Symbol]JITValueDesc{
		Symbol("dynamic_argument"): local,
	}}}
	if got := jitPreviewCallArgument(ctx, nil, 0, NewSymbol("dynamic_argument")); got.Loc != local.Loc || got.Type != local.Type || got.StackOff != local.StackOff {
		t.Fatalf("local symbol preview = %+v, want %+v", got, local)
	}

	const globalName = Symbol("jit_preview_global_argument")
	previous, existed := Globalenv.Vars[globalName]
	Globalenv.Vars[globalName] = NewSlice([]Scmer{NewInt(1), NewInt(2)})
	defer func() {
		if existed {
			Globalenv.Vars[globalName] = previous
		} else {
			delete(Globalenv.Vars, globalName)
		}
	}()
	got := jitPreviewCallArgument(ctx, nil, 0, NewSymbol(string(globalName)))
	if got.Loc != LocImm || got.Type != tagSlice || !got.SliceSizeKnown || got.KnownSliceLen != 2 {
		t.Fatalf("global symbol preview = %+v, want known two-item slice", got)
	}
}

func TestJITExpressionKeepsLocalAcrossConsecutiveDirectProcCalls(t *testing.T) {
	const candidateName = Symbol("jit_test_candidate_facts")
	const factsName = Symbol("jit_test_stage_facts")
	for name, source := range map[Symbol]string{
		candidateName: `(lambda (stage session) (list (list (quote candidate) (nth stage 0))))`,
		factsName:     `(lambda (stage) (nth stage 11))`,
	} {
		previous, existed := Globalenv.Vars[name]
		compiled := compileJITExpressionTestProc(t, source)
		Globalenv.Vars[name] = compiled
		defer func() {
			if existed {
				Globalenv.Vars[name] = previous
			} else {
				delete(Globalenv.Vars, name)
			}
		}()
	}
	compiled := compileJITExpressionTestProc(t, `(lambda (membership session)
		(begin
			(define stage (nth membership 0))
			(merge (list
				(jit_test_candidate_facts stage session)
				(jit_test_stage_facts stage)))))`)
	stage := []Scmer{NewInt(7), NewNil(), NewNil(), NewNil(), NewNil(), NewNil(), NewNil(), NewNil(), NewNil(), NewNil(), NewNil(), NewSlice([]Scmer{NewSlice([]Scmer{NewSymbol("fact"), NewInt(42)})})}
	want := NewSlice([]Scmer{NewSlice([]Scmer{NewSymbol("candidate"), NewInt(7)}), NewSlice([]Scmer{NewSymbol("fact"), NewInt(42)})})
	if got := compiled.Proc().Compiled.Call(NewSlice([]Scmer{NewSlice(stage)}), NewNil()); !Equal(got, want) {
		t.Fatalf("consecutive direct calls = %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionKeepsComputedListAcrossWideDirectProcCall(t *testing.T) {
	const calleeName = Symbol("jit_test_wide_list_callee")
	previous, existed := Globalenv.Vars[calleeName]
	Globalenv.Vars[calleeName] = compileJITExpressionTestProc(t, `(lambda (all lookup stage nested sink)
		(merge (list (list stage) all)))`)
	defer func() {
		if existed {
			Globalenv.Vars[calleeName] = previous
		} else {
			delete(Globalenv.Vars, calleeName)
		}
	}()
	compiled := compileJITExpressionTestProc(t, `(lambda (stage tail)
		(begin
			(define catalog (merge (list (list stage) tail)))
			(jit_test_wide_list_callee catalog catalog stage true nil)))`)
	want := NewSlice([]Scmer{NewSymbol("stage"), NewSymbol("stage"), NewSymbol("tail")})
	if got := compiled.Proc().Compiled.Call(NewSymbol("stage"), NewSlice([]Scmer{NewSymbol("tail")})); !Equal(got, want) {
		t.Fatalf("wide direct call = %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionDirectProcCallRelocatesArgsOnStackGrowth(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	const calleeName = Symbol("jit_test_stack_growing_callee")
	previous, existed := Globalenv.Vars[calleeName]
	callee := jitCompile(NewProcStruct(Proc{
		Params:       NewSlice([]Scmer{NewSymbol("value")}),
		Body:         NewNthLocalVar(0),
		En:           &Globalenv,
		NumVars:      2048,
		NumberedOnly: true,
	}))
	Globalenv.Vars[calleeName] = callee
	defer func() {
		if existed {
			Globalenv.Vars[calleeName] = previous
		} else {
			delete(Globalenv.Vars, calleeName)
		}
	}()
	caller := compileJITExpressionTestProc(t, `(lambda (value) (jit_test_stack_growing_callee value))`)
	want := NewSlice([]Scmer{NewString("stack-relocated")})
	if got := callJITExpressionAtDepth(caller, want, 128); !Equal(got, want) {
		t.Fatalf("direct call after stack growth = %s, want %s", String(got), String(want))
	}
}

func callJITExpressionAtDepth(callable, arg Scmer, depth int) Scmer {
	var frame [512]byte
	frame[0] = byte(depth)
	if depth == 0 {
		result := Apply(callable, arg)
		runtime.KeepAlive(frame)
		return result
	}
	result := callJITExpressionAtDepth(callable, arg, depth-1)
	runtime.KeepAlive(frame)
	return result
}

func compileJITExpressionTestProc(t *testing.T, source string) Scmer {
	t.Helper()
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	expression := Optimize(Read(t.Name(), source), &Globalenv, nil)
	proc := Eval(expression, &Globalenv)
	compiled := jitCompile(proc)
	if compiled.GetTag() != tagProc || compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatalf("expression did not compile: %s", SerializeToString(expression, &Globalenv))
	}
	return compiled
}

func requireNoDynamicJITCalls(t *testing.T, compiled Scmer) {
	t.Helper()
	coverage := compiled.Proc().Compiled.Coverage
	if coverage.DynamicCalls != 0 {
		t.Fatalf("expected complete expression lowering, got %+v", coverage)
	}
}

func TestJITExpressionListResultOwnsBackingStorage(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (value) (list value))`)
	requireNoDynamicJITCalls(t, compiled)
	first := compiled.Proc().Compiled.Call(NewSymbol("list"))
	_ = compiled.Proc().Compiled.Call(NewString("replacement"))
	want := NewSlice([]Scmer{NewSymbol("list")})
	if !Equal(first, want) {
		t.Fatalf("retained list changed after a later invocation: %s", String(first))
	}
	empty := NewSlice(nil)
	if got := compiled.Proc().Compiled.Call(empty); !Equal(got, NewSlice([]Scmer{empty})) {
		t.Fatalf("retained nested empty list changed: %s", String(got))
	}
}

func TestJITDynamicListCallOwnsBackingStorage(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (callback value) (callback value 2 3))`)
	first := Apply(compiled, NewFunc(List), NewInt(1))
	_ = Apply(compiled, NewFunc(List), NewInt(99))
	want := NewSlice([]Scmer{NewInt(1), NewInt(2), NewInt(3)})
	if !Equal(first, want) {
		t.Fatalf("dynamic list result changed after frame reuse: got %s, want %s", String(first), String(want))
	}
}

func TestJITExpressionBeginDefine(t *testing.T) {
	proc := &Proc{
		Params: NewSlice([]Scmer{NewSymbol("x")}),
		Body: NewSlice([]Scmer{
			NewSymbol("begin"),
			NewSlice([]Scmer{
				NewSymbol("define"), NewSymbol("y"),
				NewSlice([]Scmer{NewSymbol("+"), NewNthLocalVar(0), NewInt(1)}),
			}),
			NewSymbol("y"),
		}),
		NumVars: 1,
	}
	compiled := jitCompile(NewProcStruct(*proc))
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("begin/define expression did not compile")
	}
	requireNoDynamicJITCalls(t, compiled)
	if got := Apply(compiled, NewInt(41)); !Equal(got, NewInt(42)) {
		t.Fatalf("unexpected begin/define result: %s", String(got))
	}
}

func TestJITExpressionListForms(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		compiled := compileJITExpressionTestProc(t, `(lambda (a b) (list a b))`)
		requireNoDynamicJITCalls(t, compiled)
		got := Apply(compiled, NewInt(1), NewInt(2))
		want := NewSlice([]Scmer{NewInt(1), NewInt(2)})
		if !Equal(got, want) {
			t.Fatalf("unexpected list result: %s", String(got))
		}
	})

	t.Run("!list", func(t *testing.T) {
		proc := Proc{
			Params: NewSlice([]Scmer{NewSymbol("a"), NewSymbol("b")}),
			Body: NewSlice([]Scmer{
				NewSymbol("nth"),
				NewSlice([]Scmer{NewSymbol("!list"), NewNthLocalVar(2), NewInt(2), NewNthLocalVar(0), NewNthLocalVar(1)}),
				NewInt(1),
			}),
			NumVars: 4,
		}
		if !jitEnabled {
			t.Skip("requires GOEXPERIMENT=jit")
		}
		compiled := jitCompile(NewProcStruct(proc))
		if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
			t.Fatal("!list expression did not compile")
		}
		requireNoDynamicJITCalls(t, compiled)
		if got := Apply(compiled, NewInt(1), NewInt(2)); !Equal(got, NewInt(2)) {
			t.Fatalf("unexpected !list result: %s", String(got))
		}
	})

	t.Run("!!list", func(t *testing.T) {
		compiled := compileJITExpressionTestProc(t, `(lambda () (!!list 4))`)
		requireNoDynamicJITCalls(t, compiled)
		got := Apply(compiled).Slice()
		if len(got) != 0 || cap(got) != 4 {
			t.Fatalf("unexpected !!list shape: len=%d cap=%d", len(got), cap(got))
		}
	})
}

func BenchmarkJITListMaterialization(b *testing.B) {
	if !jitEnabled {
		b.Skip("requires GOEXPERIMENT=jit")
	}
	source := preparedTestProc(b, `(lambda (id value) (list "id" id "value" value))`)
	compiled := jitCompile(source)
	if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		b.Fatal("list projection did not compile")
	}
	entry := compiled.Proc().Compiled
	args := []Scmer{NewInt(1), NewInt(71)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jitListBenchmarkSink = entry.Native(args...)
	}
	runtime.KeepAlive(entry)
}

func TestJITExpressionConditionalBorrowedListResult(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (node)
		(if (> (count node) 4) (nth node 4) '()))`)
	want := NewSlice([]Scmer{NewString("required")})
	got := Apply(compiled, NewSlice([]Scmer{
		NewInt(0), NewInt(1), NewInt(2), NewInt(3), want,
	}))
	if !Equal(got, want) {
		t.Fatalf("unexpected conditional list result: got %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionReduceLambdaArgumentOrder(t *testing.T) {
	compiled := compileJITExpressionTestProc(t,
		`(lambda (items initial) (reduce items (lambda (acc item) (list acc item)) initial))`)
	if coverage := compiled.Proc().Compiled.Coverage; coverage.DynamicCalls != 1 {
		t.Fatalf("expected reduce to use one generic callback fallback, got %+v", coverage)
	}
	got := Apply(compiled,
		NewSlice([]Scmer{NewString("a"), NewString("b")}),
		NewString("initial"))
	want := NewSlice([]Scmer{
		NewSlice([]Scmer{NewString("initial"), NewString("a")}),
		NewString("b"),
	})
	if !Equal(got, want) {
		t.Fatalf("unexpected reduce result: got %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionFusedMapReducerArithmetic(t *testing.T) {
	compiled := compileJITExpressionTestProc(t,
		`(lambda (acc value id) (+ acc (+ (* value 3) id)))`)
	requireNoDynamicJITCalls(t, compiled)
	if got := Apply(compiled, NewInt(10), NewInt(4), NewInt(2)); !Equal(got, NewInt(24)) {
		t.Fatalf("unexpected fused arithmetic result: %s", String(got))
	}
}

func TestJITExpressionKeepsAdjacentStringLengths(t *testing.T) {
	compiled := compileJITExpressionTestProc(t,
		`(lambda () (list "ID" "post_status" "post_type"))`)
	requireNoDynamicJITCalls(t, compiled)
	got := Apply(compiled)
	want := NewSlice([]Scmer{
		NewString("ID"),
		NewString("post_status"),
		NewString("post_type"),
	})
	if !Equal(got, want) {
		t.Fatalf("adjacent string metadata was corrupted: got %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionKeepsQueryPlanColumnNames(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	testEnv := &Env{Vars: Vars{
		Symbol("test_scan_order"): NewFunc(func(args ...Scmer) Scmer { return args[9] }),
		Symbol("test_table"):      NewFunc(func(...Scmer) Scmer { return NewString("table") }),
	}, Outer: &Globalenv}
	expression := Optimize(Read(t.Name(), `(lambda (session tx resultrow resultfields)
		(!begin
			(resultfields (quote ("ID" "post_status" "post_type")))
			(test_scan_order tx (test_table "wpbench" "wp_posts") (quote ()) (lambda () true)
				(quote ("post_title")) (quote ((collate "utf8mb4" false)))
				0 0 (session "v1") (list "ID" "post_status" "post_type")
				(lambda (__scan_acc wp_posts.ID wp_posts.post_status wp_posts.post_type)
					(resultrow (list "ID" wp_posts.ID "post_status" wp_posts.post_status "post_type" wp_posts.post_type)))
				4 nil false)))`), testEnv, nil)
	compiled := jitCompile(Eval(expression, testEnv))
	if compiled.GetTag() != tagProc || compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("query-plan expression did not compile")
	}
	session := NewFunc(func(...Scmer) Scmer { return NewInt(5) })
	resultrow := NewFunc(func(args ...Scmer) Scmer { return args[0] })
	resultfields := NewFunc(func(...Scmer) Scmer { return NewNil() })
	got := Apply(compiled, session, NewNil(), resultrow, resultfields)
	want := NewSlice([]Scmer{NewString("ID"), NewString("post_status"), NewString("post_type")})
	if !Equal(got, want) {
		t.Fatalf("query-plan column names were corrupted: got %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionHigherOrderClosureCapture(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (values blocked) (begin
		(define blocked_set (reduce blocked (lambda (acc item) (set_assoc acc item true)) '()))
		(filter values (lambda (item) (not (has_assoc? blocked_set item))))))`)
	got := Apply(compiled,
		NewSlice([]Scmer{NewString("a"), NewString("b"), NewString("c")}),
		NewSlice([]Scmer{NewString("b")}))
	want := NewSlice([]Scmer{NewString("a"), NewString("c")})
	if !Equal(got, want) {
		t.Fatalf("unexpected captured higher-order result: got %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionReusesCompiledClosureTemplate(t *testing.T) {
	compiled := compileJITExpressionTestProc(t,
		`(lambda (captured) (lambda (value) (+ captured value)))`)
	first := Apply(compiled, NewInt(40))
	second := Apply(compiled, NewInt(41))
	if first.Proc() == nil || first.Proc().Compiled == nil || second.Proc() == nil || second.Proc().Compiled == nil {
		t.Fatal("captured closures did not receive native entry points")
	}
	if first.Proc().Compiled.CodePtr != second.Proc().Compiled.CodePtr {
		t.Fatal("captured closures recompiled an identical lambda body")
	}
	if got := Apply(first, NewInt(2)); !Equal(got, NewInt(42)) {
		t.Fatalf("first captured closure returned %s, want 42", String(got))
	}
	if got := Apply(second, NewInt(2)); !Equal(got, NewInt(43)) {
		t.Fatalf("second captured closure returned %s, want 43", String(got))
	}
}

func TestJITExpressionNestedClosureKeepsDeepCallableCapture(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (callback)
		(lambda (level_one)
			(lambda (level_two)
				(lambda (value) (callback value)))))`)
	callback := NewFunc(func(args ...Scmer) Scmer { return args[0] })
	levelOne := Apply(compiled, callback)
	levelTwo := Apply(levelOne, NewNil())
	inner := Apply(levelTwo, NewNil())
	want := NewString("deep capture")
	if got := Apply(inner, want); !Equal(got, want) {
		t.Fatalf("nested JIT closure returned %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionCapturesNumberedValueAtExactOuterDepth(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	callback := NewFunc(func(args ...Scmer) Scmer { return args[0] })
	params := make([]Scmer, 14)
	args := make([]Scmer, 14)
	for index := range params {
		params[index] = NewSymbol(fmt.Sprintf("arg-%d", index))
		args[index] = NewNil()
	}
	args[13] = callback
	outerCall := NewSlice([]Scmer{NewSymbol("outer"), NewInt(1), NewNthLocalVar(13)})
	innerCall := NewSlice([]Scmer{outerCall, NewNthLocalVar(0)})
	lambda := NewSlice([]Scmer{
		NewSymbol("lambda"),
		NewSlice([]Scmer{NewSymbol("value")}),
		innerCall,
		NewInt(1),
	})
	root := NewProcStruct(Proc{
		Params:  NewSlice(params),
		Body:    lambda,
		En:      &Globalenv,
		NumVars: 14,
	})
	compiled := jitCompile(root)
	inner := Apply(compiled, args...)
	want := NewString("exact outer frame")
	if got := Apply(inner, want); !Equal(got, want) {
		t.Fatalf("deep numbered capture returned %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionCapturesNumberedValueAcrossBeginScope(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	outer := NewSlice([]Scmer{NewSymbol("outer"), NewInt(2), NewNthLocalVar(0)})
	inner := NewSlice([]Scmer{
		NewSymbol("lambda"),
		NewSlice(nil),
		NewSlice([]Scmer{NewSymbol("begin"), outer}),
		NewInt(0),
	})
	root := NewProcStruct(Proc{
		Params:  NewSlice([]Scmer{NewSymbol("captured")}),
		Body:    inner,
		En:      &Globalenv,
		NumVars: 1,
	})
	compiled := jitCompile(root)
	want := NewSlice([]Scmer{NewString("key"), NewInt(7)})
	closure := Apply(compiled, want)
	if got := Apply(closure); !Equal(got, want) {
		t.Fatalf("begin-scoped JIT capture returned %s, want %s", String(got), String(want))
	}
	if got := WithSession(NewSession(), closure); !Equal(got, want) {
		t.Fatalf("session-bound JIT capture returned %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionWithSessionRebindsNativeCapture(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (session) (lambda () session))`)
	first := NewSession()
	second := NewSession()
	closure := Apply(compiled, first)
	if got := Apply(closure); !Equal(got, first) {
		t.Fatalf("native closure returned %s, want original session", String(got))
	}
	if got := WithSession(second, closure); !Equal(got, second) {
		t.Fatalf("session-bound native closure returned %s, want replacement session", String(got))
	}
}

func TestJITExpressionWithSessionRebindsRuntimeEnvironment(t *testing.T) {
	first := NewFunc(func(...Scmer) Scmer { return NewInt(1) })
	second := NewFunc(func(...Scmer) Scmer { return NewInt(2) })
	env := &Env{Vars: Vars{Symbol("session"): first}, Outer: &Globalenv}
	expression := Optimize(Read(t.Name(), `(lambda () (eval '(session)))`), env, nil)
	compiled := jitCompile(Eval(expression, env))
	if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("runtime-environment closure did not compile")
	}
	if got := Apply(compiled); !Equal(got, NewInt(1)) {
		t.Fatalf("native closure returned %s, want original environment", String(got))
	}
	if got := WithSession(second, compiled); !Equal(got, NewInt(2)) {
		t.Fatalf("session-bound native closure returned %s, want rebound environment", String(got))
	}
}

func TestJITExpressionNestedRuntimeLambdaCapturesNamedEnvironment(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	want := NewString("named environment callable")
	env := &Env{
		Vars: Vars{
			Symbol("local_print"): NewFunc(func(...Scmer) Scmer { return want }),
		},
		Outer: &Globalenv,
	}
	expression := Optimize(Read(t.Name(), `(lambda () (begin
		(define inner (lambda () (begin (eval 'nil) (local_print))))
		(inner)))`), env, nil)
	compiled := jitCompile(Eval(expression, env))
	if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("outer runtime-environment closure did not compile")
	}
	if got := Apply(compiled); !Equal(got, want) {
		t.Fatalf("nested runtime lambda returned %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionNestedLambdaKeepsParameterSeparateFromCapture(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (req)
		(lambda (username) (equal? username (req "username"))))`)
	req := NewSlice([]Scmer{NewString("username"), NewString("root")})
	inner := Apply(compiled, req)
	if inner.Proc() == nil {
		t.Fatalf("nested lambda returned %s, want procedure", String(inner))
	}
	if got := Apply(inner, NewString("root")); !Equal(got, NewBool(true)) {
		t.Fatalf("nested lambda body %s returned %s, want true", String(inner), String(got))
	}
}

func TestJITExpressionNestedLambdaCapturesShadowedOuterParameter(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (value inner)
		((lambda (value) (list value (outer 1 value))) inner))`)
	want := NewSlice([]Scmer{NewInt(7), NewInt(5)})
	if got := Apply(compiled, NewInt(5), NewInt(7)); !Equal(got, want) {
		t.Fatalf("shadowed outer parameter capture returned %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionCallbackKeepsParameterSeparateFromCapture(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (req)
		(map '("root") (lambda (username) (equal? username (req "username")))))`)
	req := NewSlice([]Scmer{NewString("username"), NewString("root")})
	got := Apply(compiled, req)
	want := NewSlice([]Scmer{NewBool(true)})
	if !Equal(got, want) {
		t.Fatalf("callback returned %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionNestedAnonymousLambdasPreserveOuterDepth(t *testing.T) {
	param := func(name string) Scmer { return NewSlice([]Scmer{NewSymbol(name)}) }
	call := func(lambda Scmer, value int64) Scmer {
		return NewSlice([]Scmer{lambda, NewInt(value)})
	}
	lambda := func(name string, body Scmer) Scmer {
		return NewSlice([]Scmer{NewSymbol("lambda"), param(name), body, NewInt(1)})
	}
	outer := NewSlice([]Scmer{NewSymbol("outer"), NewInt(3), NewNthLocalVar(0)})
	body := call(lambda("b", call(lambda("c", call(lambda("d", outer), 3)), 2)), 1)
	root := NewProcStruct(Proc{
		Params:  param("root"),
		Body:    body,
		En:      &Globalenv,
		NumVars: 1,
	})
	compiled := jitCompile(root)
	want := NewString("root value")
	if got := Apply(compiled, want); !Equal(got, want) {
		t.Fatalf("nested outer reference returned %s", SerializeToString(got, &Globalenv))
	}
}

func TestJITExpressionTransferredMergeStackList(t *testing.T) {
	source := `(lambda (head tail)
		(merge (list (nth head 2) (list (nth head 0)) (nth tail 0))))`
	compiled := compileJITExpressionTestProc(t, source)
	head := NewSlice([]Scmer{
		NewString("source"),
		NewSlice(nil),
		NewSlice([]Scmer{NewString("generated")}),
	})
	tail := NewSlice([]Scmer{
		NewSlice([]Scmer{NewString("tail")}),
		NewSlice(nil),
		NewSlice(nil),
	})
	got := Apply(compiled, head, tail)
	want := NewSlice([]Scmer{NewString("generated"), NewString("source"), NewString("tail")})
	if !Equal(got, want) {
		t.Fatalf("unexpected transferred merge result: got %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionConsTailOfSingleElementList(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	source := `(lambda (sources)
		(match sources
			(cons src rest) rest
			_ 'not-a-list))`
	expression := Optimize(Read(t.Name(), source), &Globalenv, nil)
	interpreted := Eval(expression, &Globalenv)
	proc := *interpreted.Proc()
	proc.NumVars = 5
	proc.NumberedOnly = false
	compiled := jitCompile(NewProcStruct(proc))
	if compiled.GetTag() != tagProc || compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatalf("expression did not compile: %s", SerializeToString(expression, &Globalenv))
	}
	got := Apply(compiled, NewSlice([]Scmer{NewString("source")}))
	want := NewSlice(nil)
	if !Equal(got, want) {
		t.Fatalf("unexpected cons tail: got %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionTailSelfCallAdvancesArguments(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	const name = Symbol("jit_test_tail_self_call")
	previous, existed := Globalenv.Vars[name]
	defer func() {
		if existed {
			Globalenv.Vars[name] = previous
		} else {
			delete(Globalenv.Vars, name)
		}
	}()
	EvalAllJIT(t.Name(), `(define jit_test_tail_self_call (lambda (values)
		(match values
			(cons _ rest) (jit_test_tail_self_call rest)
			_ true)))`, &Globalenv)
	compiled := Globalenv.Vars[name]
	if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("tail-recursive procedure was not JIT compiled")
	}
	if got := Apply(compiled, NewSlice([]Scmer{NewInt(1), NewInt(2)})); !got.IsBool() || !got.Bool() {
		t.Fatalf("tail-recursive JIT call returned %s, want true", String(got))
	}
}

func TestJITExpressionRecursiveMatchKeepsEarlierFixedListBranch(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	const name = Symbol("jit_test_recursive_match_branches")
	previous, existed := Globalenv.Vars[name]
	defer func() {
		if existed {
			Globalenv.Vars[name] = previous
		} else {
			delete(Globalenv.Vars, name)
		}
	}()
	EvalAllJIT(t.Name(), `(define jit_test_recursive_match_branches (lambda (expr)
		(match expr
			((symbol get_column) tblvar _tbl_ignorecase _col _col_ignorecase)
				(and (string? tblvar) (strlike tblvar "__exists_%"))
			((quote get_column) tblvar _tbl_ignorecase _col _col_ignorecase)
				(and (string? tblvar) (strlike tblvar "__exists_%"))
			(cons _head tail) (reduce tail (lambda (found item)
				(or found (jit_test_recursive_match_branches item))) false)
			_ false)))`, &Globalenv)
	compiled := Globalenv.Vars[name]
	if compiled.GetTag() != tagProc || compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("recursive match procedure was not JIT compiled")
	}
	Globalenv.Vars[name] = compiled
	input := NewSlice([]Scmer{
		NewSymbol("get_column"), NewString("__exists_source"),
		NewBool(false), NewString("value"), NewBool(false),
	})
	if got := Apply(compiled, input); !got.IsBool() || !got.Bool() {
		t.Fatalf("earlier fixed-list branch returned %s, want true", String(got))
	}
	if got := callJITExpressionAtDepth(compiled, input, 128); !got.IsBool() || !got.Bool() {
		t.Fatalf("deep-stack fixed-list branch returned %s, want true", String(got))
	}
}

func TestJITExpressionWideMatchPreservesEveryCapture(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (query) (match query
		((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts)
			(list schema tables fields condition group having order limit offset hidden stages facts)
		_ nil))`)
	want := make([]Scmer, 12)
	for index := range want {
		want[index] = NewInt(int64(index + 1))
	}
	input := append([]Scmer{NewSymbol("query-block")}, want...)
	got := Apply(compiled, NewSlice(input))
	if !Equal(got, NewSlice(want)) {
		t.Fatalf("wide match returned %s, want %s", String(got), String(NewSlice(want)))
	}
}

func TestJITExpressionQueryBlockMatchHelpersKeepConditionAndStage(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (query order limit offset) (begin
		(define select_order (lambda (query) (match query
			((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts) order
			_ nil)))
		(define clear_stage (lambda (query) (match query
			((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts)
				(list (quote query-block) schema tables fields condition group having nil nil nil '() '() '())
			_ query)))
		(define combine (lambda (left right)
			(list left (clear_stage right) (select_order right))))
		(combine (quote left) query)))`)
	query := NewSlice([]Scmer{
		NewSymbol("query-block"), NewString("schema"), NewString("tables"), NewString("fields"),
		NewString("condition"), NewString("group"), NewString("having"), NewString("order"),
		NewInt(2), NewInt(1), NewString("hidden"), NewString("stages"), NewString("facts"),
	})
	got := Apply(compiled, query, NewNil(), NewNil(), NewNil())
	items := got.Slice()
	if len(items) != 3 || !Equal(items[2], NewString("order")) {
		t.Fatalf("query-block helpers returned %s, want order capture", String(got))
	}
	cleared := items[1].Slice()
	if len(cleared) != 13 || !Equal(cleared[4], NewString("condition")) {
		t.Fatalf("clear-stage helper returned %s, want condition capture", String(items[1]))
	}
}

func TestJITExpressionRecursiveTransferredMergeStackList(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	EvalAll(t.Name(), `(define jit_test_recursive_merge (lambda (sources)
		(match sources
			(cons src rest) (begin
				(define head (list src '() '()))
				(define tail (jit_test_recursive_merge rest))
				(define generated_sources (nth head 2))
				(list
					(merge (list generated_sources (list (nth head 0)) (nth tail 0)))
					(merge_unique (list (nth head 1) (nth tail 1)))
					(merge_unique (list generated_sources (nth tail 2)))
					(nth tail 3)))
			_ (list '() '() '() '()))))`, &Globalenv)
	compiled := jitCompile(Globalenv.Vars[Symbol("jit_test_recursive_merge")])
	if compiled.GetTag() != tagProc || compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("recursive merge procedure was not JIT compiled")
	}
	Globalenv.Vars[Symbol("jit_test_recursive_merge")] = compiled
	got := Apply(compiled, NewSlice([]Scmer{NewString("source")}))
	want := NewSlice([]Scmer{
		NewSlice([]Scmer{NewString("source")}),
		NewSlice(nil),
		NewSlice(nil),
		NewSlice(nil),
	})
	if !Equal(got, want) {
		t.Fatalf("unexpected recursive transferred merge result: got %s, want %s", String(got), String(want))
	}
}

func TestJITExpressionRecursiveNthResult(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	EvalAll(t.Name(), `(define jit_test_recursive_nth (lambda (sources)
		(match sources
			(cons src rest) (begin
				(define tail (jit_test_recursive_nth rest))
				(nth tail 0))
			_ (list '()))))`, &Globalenv)
	compiled := jitCompile(Globalenv.Vars[Symbol("jit_test_recursive_nth")])
	if compiled.GetTag() != tagProc || compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("recursive nth procedure was not JIT compiled")
	}
	Globalenv.Vars[Symbol("jit_test_recursive_nth")] = compiled
	got := Apply(compiled, NewSlice([]Scmer{NewString("source")}))
	want := NewSlice(nil)
	if !Equal(got, want) {
		gotPtr, gotAux := got.RawWords()
		t.Fatalf("unexpected recursive nth result: got ptr=%x aux=%x tag=%d, want %s", gotPtr, gotAux, got.GetTag(), String(want))
	}
}

func TestJITExpressionRecursivePanicAcrossDirectFrames(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	const name = Symbol("jit_test_direct_recursive_panic")
	previous, existed := Globalenv.Vars[name]
	defer func() {
		if existed {
			Globalenv.Vars[name] = previous
		} else {
			delete(Globalenv.Vars, name)
		}
	}()
	EvalAll(t.Name(), `(define jit_test_direct_recursive_panic (lambda (depth)
		(if (> depth 0)
			(+ 1 (jit_test_direct_recursive_panic (- depth 1)))
			(error "nested-jit-panic"))))`, &Globalenv)
	compiled := jitCompile(Globalenv.Vars[name])
	if compiled.GetTag() != tagProc || compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("recursive panic procedure was not JIT compiled")
	}
	Globalenv.Vars[name] = compiled

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		Apply(compiled, NewInt(4))
	}()
	if recovered == nil {
		t.Fatal("panic did not unwind through consecutive JIT frames")
	}
}
