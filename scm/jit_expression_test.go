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
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"unsafe"
)

var jitListBenchmarkSink Scmer

func gcPointerWordOffsets(typ reflect.Type, base uintptr) []int32 {
	var result []int32
	var walk func(reflect.Type, uintptr)
	walk = func(current reflect.Type, offset uintptr) {
		switch current.Kind() {
		case reflect.Pointer, reflect.UnsafePointer, reflect.Map, reflect.Chan, reflect.Func:
			result = append(result, int32(offset))
		case reflect.Interface:
			result = append(result, int32(offset+unsafe.Sizeof(uintptr(0))))
		case reflect.Slice, reflect.String:
			result = append(result, int32(offset))
		case reflect.Array:
			for index := 0; index < current.Len(); index++ {
				walk(current.Elem(), offset+uintptr(index)*current.Elem().Size())
			}
		case reflect.Struct:
			for index := 0; index < current.NumField(); index++ {
				field := current.Field(index)
				walk(field.Type, offset+field.Offset)
			}
		}
	}
	walk(typ, base)
	return result
}

func TestJITProcStackPointerOffsetsMatchLayout(t *testing.T) {
	want := gcPointerWordOffsets(reflect.TypeOf(Proc{}), 0)
	got := jitProcStackPointerOffsets[:]
	if !slices.Equal(got, want) {
		t.Fatalf("Proc pointer words = %v, want %v", got, want)
	}
}

func TestJITGeneratedProcFieldOffsetsFollowCurrentLayout(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (value) (close_procedure value))`)
	originalMeta := &ProcOptimizerMeta{Return: tiZero.WithKind(KindInt), HasReturn: true}
	original := NewProcStruct(Proc{
		Params:        NewSlice(nil),
		Body:          NewInt(7),
		En:            &Env{Vars: Vars{}, Outer: &Globalenv},
		OptimizerMeta: originalMeta,
	})
	closed := Apply(compiled, original)
	if !closed.IsProc() || closed.Proc().OptimizerMeta == nil {
		t.Fatal("generated close_procedure lost Proc optimizer metadata")
	}
	if closed.Proc().OptimizerMeta == originalMeta {
		t.Fatal("generated close_procedure reused mutable Proc optimizer metadata")
	}
	if !closed.Proc().OptimizerMeta.HasReturn || closed.Proc().OptimizerMeta.Return.Kind() != KindInt {
		t.Fatalf("generated close_procedure read stale Proc field offsets: %#v", closed.Proc().OptimizerMeta)
	}
}

//go:noinline
func growJITCallbackTestStack(depth int) int {
	var frame [1024]byte
	frame[0] = byte(depth)
	if depth == 0 {
		runtime.KeepAlive(&frame)
		return int(frame[0])
	}
	result := int(frame[0]) + growJITCallbackTestStack(depth-1)
	runtime.KeepAlive(&frame)
	return result
}

func jitCallbackTestSafepoint(args ...Scmer) Scmer {
	runtime.GC()
	_ = growJITCallbackTestStack(64)
	return args[0]
}

func TestJITFunctionValueCarriesOriginalProcAndInlineCaptures(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (captured)
		(lambda (value) (list captured value)))`)
	inner := Apply(compiled, NewString("outer"))
	if inner.Proc() == nil || inner.Proc().JITCode == 0 {
		t.Fatal("capturing lambda has no native function")
	}
	function := inner.Proc().jitFunction()
	if got := JITProcForFunction(function); got != inner.Proc() {
		t.Fatalf("funcval Proc = %p, want %p", got, inner.Proc())
	}
	funcval := *(*unsafe.Pointer)(unsafe.Pointer(&function))
	if got := (*Proc)(funcval); got != inner.Proc() {
		t.Fatalf("funcval Proc = %p, want %p", got, inner.Proc())
	}
	captures := jitProcCaptures(inner.Proc())
	if len(captures) != 1 {
		t.Fatalf("capture count = %d, want 1", len(captures))
	}
	capture := *(*Scmer)(unsafe.Add(funcval, unsafe.Offsetof(ProcJIT{}.Context)))
	if !Equal(capture, captures[0]) {
		t.Fatalf("inline capture = %s, want %s", String(capture), String(captures[0]))
	}
	if got := Apply(inner, NewString("inner")); !Equal(got, NewSlice([]Scmer{NewString("outer"), NewString("inner")})) {
		t.Fatalf("capturing funcval returned %s", String(got))
	}
}

func TestJITScanShapedMaintenanceCallbacksCaptureOuterRow(t *testing.T) {
	const name = "jit_test_scan_shaped_maintenance_callbacks"
	called := false
	params := []*TypeDescriptor{
		{Kind: "any"}, {Kind: "any"}, {Kind: "list"},
		{Kind: "func", NoEscape: true, Params: []*TypeDescriptor{{Kind: "any", Variadic: true}}, Return: &TypeDescriptor{Kind: "bool"}},
		{Kind: "list"},
		{Kind: "func", NoEscape: true, Params: []*TypeDescriptor{{Kind: "any", Variadic: true}}, Return: &TypeDescriptor{Kind: "any"}},
		{Kind: "any", Optional: true}, {Kind: "func", NoEscape: true, Optional: true}, {Kind: "bool", Optional: true},
	}
	Declare(&Globalenv, &Declaration{
		Name: name,
		Fn: func(args ...Scmer) Scmer {
			result := make(chan Scmer, 1)
			filter := PrepareSerialProc(args[3])
			mapReduce := PrepareSerialProc(args[5])
			go func() {
				if ToBool(filter.Function(NewInt(7))) {
					result <- mapReduce.Function(NewNil(), NewFunc(func(...Scmer) Scmer {
						called = true
						return NewNil()
					}))
					return
				}
				result <- NewNil()
			}()
			return <-result
		},
		Type: &TypeDescriptor{Kind: "func", HasSideEffects: true, Params: params, Return: &TypeDescriptor{Kind: "any"}},
	})
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	raw := Eval(Read(t.Name(), `(lambda (OLD NEW session tx)
		(jit_test_scan_shaped_maintenance_callbacks session nil (quote ("k0"))
			(lambda (value) (equal? value (get_assoc OLD "id")))
			(quote ("$invalidate:value"))
			(lambda (acc update) (begin (update) acc)) nil nil false))`), &Globalenv)
	compiled := CompileJIT(raw, true)
	if !compiled.IsProc() || compiled.Proc().Compiled == nil {
		t.Fatal("raw maintenance procedure did not compile")
	}
	old := NewFastDictValue(1)
	old.Set(NewString("id"), NewInt(7), nil)
	Apply(compiled, NewFastDict(old), NewNil(), NewNil(), NewNil())
	if !called {
		t.Fatal("scan-shaped maintenance callback lost its captured trigger row")
	}
}

func TestJITProcForFunctionRejectsOrdinaryGoFunction(t *testing.T) {
	ordinary := func(...Scmer) Scmer { return NewNil() }
	if got := JITProcForFunction(ordinary); got != nil {
		t.Fatalf("ordinary Go function resolved to Proc %p", got)
	}
}

func TestSerializeJITFunctionUsesOriginalProc(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (value) (+ value 1))`)
	function := NewFunc(compiled.Proc().jitFunction())
	serialized := SerializeToString(function, &Globalenv)
	if serialized == "[unserializable native func]" || serialized == "[native func]" {
		t.Fatalf("JIT function was not serialized through its Proc: %s", serialized)
	}
	if !strings.HasPrefix(serialized, "(lambda ") {
		t.Fatalf("JIT function serialization = %s, want lambda", serialized)
	}
}

func TestJITDynamicCallReceivesBoundLocalLambda(t *testing.T) {
	compiled := compileJITExpressionTestProc(t,
		`(lambda (consumer value) (begin (define producer (lambda () value)) (consumer producer)))`)
	consumer := NewFunc(func(args ...Scmer) Scmer {
		if len(args) != 1 || args[0].GetTag() != tagProc {
			t.Fatalf("consumer received %v, want one Proc", args)
		}
		return Apply(args[0])
	})
	if got := Apply(compiled, consumer, NewInt(42)); !Equal(got, NewInt(42)) {
		t.Fatalf("dynamic callback result = %s, want 42", String(got))
	}
}

func TestJITMetadataCallUsesBoundProcFuncval(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (captured)
		(lambda () captured))`)
	bound := Apply(compiled, NewString("bound capture"))
	if !bound.IsProc() || bound.Proc().Compiled == nil {
		t.Fatal("capturing lambda has no JIT entry")
	}
	entry := bound.Proc().Compiled
	direct := entry.JITDirect
	entry.JITDirect = 0
	t.Cleanup(func() { entry.JITDirect = direct })
	if got := Apply(bound); !Equal(got, NewString("bound capture")) {
		t.Fatalf("metadata call read %s, want concrete Proc capture", SerializeToString(got, &Globalenv))
	}
}

func TestJITMetadataCallPreservesAndShortCircuit(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (schema s policy planning_session tx)
		(begin
			(set s (if (and (>= (strlen s) 3) (equal? (substr s 0 3) "/*!"))
				(match s
					(regex "^/\\*![0-9]+[\\r\\n\\t ]+((?is:CREATE[\\r\\n\\t ]+TRIGGER.*))[\\r\\n\\t ]*\\*/$" _ body) body
					s)
				s))
			(or
				(and (>= (strlen s) 25)
					(equal? (substr s 0 25) "SELECT LOGFILE_GROUP_NAME"))
				(and (>= (strlen s) 31)
					(equal? (substr s 0 31) "SELECT DISTINCT TABLESPACE_NAME")))))`)
	for _, input := range []string{
		"NULL",
		"SHOW",
		"SELECT 1",
		"CREATE TABLE user(username text, password text, admin boolean DEFAULT FALSE) ENGINE=SAFE",
	} {
		if got := Apply(compiled, NewString("system"), NewString(input), NewBool(true)); !got.IsBool() || got.Bool() {
			t.Fatalf("input %q returned %s, want false", input, SerializeToString(got, &Globalenv))
		}
	}
}

func TestJITEscapingProducerPreservesNestedOuterCaptures(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (consumer parse_query policy session parse_fn schema tx)
		(begin
			(define exact_compile (lambda (compile_tx)
				(begin
					(define resolved_policy policy)
					(define compile_policy resolved_policy)
					(with_session session (lambda ()
						(list parse_fn schema parse_query compile_policy session compile_tx))))))
			(consumer exact_compile tx)))`)
	session := NewSession()
	parseFn := NewFunc(func(...Scmer) Scmer { return NewNil() })
	policy := NewSymbol("policy")
	tx := NewSymbol("tx")
	consumer := NewFunc(func(args ...Scmer) Scmer { return Apply(args[0], args[1]) })
	got := Apply(compiled, consumer, NewString("SELECT 1"), policy, session, parseFn, NewString("schema"), tx)
	want := NewSlice([]Scmer{parseFn, NewString("schema"), NewString("SELECT 1"), policy, session, tx})
	if !Equal(got, want) {
		t.Fatalf("nested producer captures = %s, want %s", SerializeToString(got, &Globalenv), SerializeToString(want, &Globalenv))
	}
}

func TestJITCloseProcedureMaterializesInlineCaptures(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (captured)
		(lambda () (lambda () captured)))`)
	closed := CloseProcedure(Apply(compiled, NewString("persisted capture")))
	if closed.Proc().JITCode != 0 || closed.Proc().Compiled != nil {
		t.Fatal("closed capturing procedure retained process-local JIT state")
	}
	inner := Apply(closed)
	if got := Apply(inner); !Equal(got, NewString("persisted capture")) {
		t.Fatalf("closed nested capture returned %s", SerializeToString(got, &Globalenv))
	}
}

func TestJITNoEscapeCallbackUsesStackFuncval(t *testing.T) {
	const name = "jit_test_noescape_callback"
	const safepointName = "jit_test_callback_safepoint"
	consumer := func(args ...Scmer) Scmer {
		if len(args) != 2 || args[0].GetTag() != tagProc || args[0].Proc().JITCode == 0 {
			panic("noescape callback was not passed as a native Proc")
		}
		if JITProcForFunction(args[0].Proc().jitFunction()) != args[0].Proc() {
			panic("noescape callback lost its original Proc")
		}
		prepared := PrepareSerialProc(args[0])
		return prepared.Function(args[1:2]...)
	}
	declaration := &Declaration{
		Name: name,
		Fn:   consumer,
		Type: &TypeDescriptor{Kind: "func", Forbidden: true, Params: []*TypeDescriptor{
			{Kind: "func", NoEscape: true, SameGoroutine: true, Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "any"}},
			{Kind: "any"},
		}, Return: &TypeDescriptor{Kind: "any"}},
	}
	Declare(&Globalenv, declaration)
	safepointDeclaration := &Declaration{
		Name: safepointName,
		Fn:   jitCallbackTestSafepoint,
		Type: &TypeDescriptor{Kind: "func", Forbidden: true, Params: []*TypeDescriptor{
			{Kind: "any"},
		}, Return: &TypeDescriptor{Kind: "any"}},
	}
	Declare(&Globalenv, safepointDeclaration)
	defer func() {
		delete(Globalenv.Vars, Symbol(name))
		delete(declarations, name)
		delete(declarationsByFunction, FunctionIdentity(consumer))
		delete(Globalenv.Vars, Symbol(safepointName))
		delete(declarations, safepointName)
		delete(declarationsByFunction, FunctionIdentity(jitCallbackTestSafepoint))
	}()
	compiled := compileJITExpressionTestProc(t, `(lambda (captured value)
		(begin
			(jit_test_callback_safepoint value)
			(jit_test_noescape_callback (lambda (item) (+ (jit_test_callback_safepoint item) captured)) value)))`)
	args := []Scmer{NewInt(5), NewInt(7)}
	if got := compiled.Proc().jitFunction()(args...); !Equal(got, NewInt(12)) {
		t.Fatalf("stack funcval returned %s, want 12", String(got))
	}
	if allocations := testing.AllocsPerRun(100, func() {
		if got := compiled.Proc().jitFunction()(args...); !Equal(got, NewInt(12)) {
			t.Fatalf("stack funcval returned %s, want 12", String(got))
		}
	}); allocations != 0 {
		t.Fatalf("noescape callback call allocated %.2f objects, want 0", allocations)
	}
}

func TestJITTransferringCallbackCrossesGoroutineWithMaterializedFrame(t *testing.T) {
	const name = "jit_test_cross_goroutine_callback"
	consumer := func(args ...Scmer) Scmer {
		result := make(chan Scmer, 1)
		go func(callback, value Scmer) {
			prepared := PrepareSerialProc(callback)
			result <- prepared.Call([]Scmer{value})
		}(args[0], args[1])
		return <-result
	}
	Declare(&Globalenv, &Declaration{
		Name: name,
		Fn:   consumer,
		Type: &TypeDescriptor{Kind: "func", Forbidden: true, Params: []*TypeDescriptor{
			{Kind: "func", NoEscape: true, Params: []*TypeDescriptor{{Kind: "any", Transfer: true}}, Return: &TypeDescriptor{Kind: "any", Transfer: true}},
			{Kind: "any"},
		}, Return: &TypeDescriptor{Kind: "any"}},
	})
	defer func() {
		delete(Globalenv.Vars, Symbol(name))
		delete(declarations, name)
		delete(declarationsByFunction, FunctionIdentity(consumer))
	}()
	compiled := compileJITExpressionTestProc(t, `(lambda (captured value)
		(jit_test_cross_goroutine_callback (lambda (item) (list captured item)) value))`)
	want := NewSlice([]Scmer{NewInt(5), NewInt(7)})
	if got := Apply(compiled, NewInt(5), NewInt(7)); !Equal(got, want) {
		t.Fatalf("cross-goroutine callback returned %s, want %s", String(got), String(want))
	}
}

func TestJITCaptureFreeTransferringCallbackStaysDirect(t *testing.T) {
	const name = "jit_test_capture_free_transfer_callback"
	consumer := func(args ...Scmer) Scmer {
		prepared := PrepareSerialProc(args[0])
		return prepared.Call(args[1:2])
	}
	Declare(&Globalenv, &Declaration{
		Name: name,
		Fn:   consumer,
		Type: &TypeDescriptor{Kind: "func", Forbidden: true, Params: []*TypeDescriptor{
			{Kind: "func", NoEscape: true, Params: []*TypeDescriptor{{Kind: "any", Transfer: true}}, Return: &TypeDescriptor{Kind: "any", Transfer: true}},
			{Kind: "any"},
		}, Return: &TypeDescriptor{Kind: "any"}},
	})
	defer func() {
		delete(Globalenv.Vars, Symbol(name))
		delete(declarations, name)
		delete(declarationsByFunction, FunctionIdentity(consumer))
	}()
	compiled := compileJITExpressionTestProc(t, `(lambda (value)
		(jit_test_capture_free_transfer_callback (lambda (item) (+ item 1)) value))`)
	args := []Scmer{NewInt(7)}
	if got := compiled.Proc().jitFunction()(args...); !Equal(got, NewInt(8)) {
		t.Fatalf("capture-free transfer callback returned %s, want 8", String(got))
	}
	if allocations := testing.AllocsPerRun(100, func() {
		if got := compiled.Proc().jitFunction()(args...); !Equal(got, NewInt(8)) {
			t.Fatalf("capture-free transfer callback returned %s, want 8", String(got))
		}
	}); allocations != 0 {
		t.Fatalf("capture-free transfer callback allocated %.2f objects, want 0", allocations)
	}
}

func TestJITEscapingCallbackUsesOneTypedAllocation(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (captured)
		(lambda (value) (+ captured value)))`)
	args := []Scmer{NewInt(5)}
	var closure Scmer
	if allocations := testing.AllocsPerRun(100, func() {
		closure = compiled.Proc().jitFunction()(args...)
	}); allocations != 1 {
		t.Fatalf("escaping callback binding allocated %.2f objects, want 1", allocations)
	}
	runtime.GC()
	_ = growJITCallbackTestStack(64)
	if closure.Proc() == nil || JITProcForFunction(closure.Proc().jitFunction()) != closure.Proc() {
		t.Fatal("escaping callback lost its Go-compatible Proc funcval")
	}
	if got := closure.Proc().jitFunction()(NewInt(7)); !Equal(got, NewInt(12)) {
		t.Fatalf("escaping callback returned %s, want 12", String(got))
	}
}

func TestJITCallPadsMissingArguments(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (query planning_session tx)
		(list query planning_session tx))`)
	want := NewSlice([]Scmer{NewString("t"), NewNil(), NewNil()})
	if got := Apply(compiled, NewString("t")); !Equal(got, want) {
		t.Fatalf("padded native call returned %s, want %s", String(got), String(want))
	}
}

func TestJITMissingArgumentRemainsNilAcrossCallbackCapture(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (query planning_session tx)
		(reduce (list query) (lambda (_ src) planning_session) nil))`)
	if got := Apply(compiled, NewString("t")); !got.IsNil() {
		t.Fatalf("captured missing argument returned %s, want nil", String(got))
	}
}

func TestJITStackListPreservesNestedListWords(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (membership)
		(begin
			(define stage (nth membership 0))
			(merge (list (list (list 'work 1)) (nth stage 11))))))`)
	facts := NewSlice([]Scmer{NewSlice([]Scmer{NewSymbol("fact"), NewInt(2)})})
	stageItems := make([]Scmer, 12)
	for index := range stageItems {
		stageItems[index] = NewNil()
	}
	stageItems[11] = facts
	membership := NewSlice([]Scmer{NewSlice(stageItems)})
	want := NewSlice([]Scmer{
		NewSlice([]Scmer{NewSymbol("work"), NewInt(1)}),
		NewSlice([]Scmer{NewSymbol("fact"), NewInt(2)}),
	})
	if got := Apply(compiled, membership); !Equal(got, want) {
		t.Fatalf("nested stack-list value = %s, want %s", String(got), String(want))
	}
}

func TestJITNestedListProducerSurvivesWideLocalFrame(t *testing.T) {
	source := `(lambda (stage) (begin `
	for index := 0; index < 48; index++ {
		source += fmt.Sprintf(`(define value%d (list %d)) `, index, index)
	}
	source += `(merge (list value0 value7 value19 value31 value47 (nth stage 11)))))`
	compiled := compileJITExpressionTestProc(t, source)
	stage := make([]Scmer, 12)
	for index := range stage {
		stage[index] = NewNil()
	}
	stage[11] = NewSlice([]Scmer{NewInt(99)})
	want := NewSlice([]Scmer{NewInt(0), NewInt(7), NewInt(19), NewInt(31), NewInt(47), NewInt(99)})
	if got := Apply(compiled, NewSlice(stage)); !Equal(got, want) {
		t.Fatalf("wide-frame nested list = %s, want %s", String(got), String(want))
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

func compileJITExpressionTestProc(t testing.TB, source string) Scmer {
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

func TestJITGlobalCallableLookupWritesBranchResultDirectly(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (add)
		(if add + -))`)

	tests := []struct {
		add  bool
		want int
	}{
		{add: true, want: 10},
		{add: false, want: 4},
	}
	for _, test := range tests {
		callable := Apply(compiled, NewBool(test.add))
		got := Apply(callable, NewInt(7), NewInt(3))
		if ToInt(got) != test.want {
			t.Fatalf("global callable for add=%v returned %s, want %d", test.add, String(got), test.want)
		}
	}
}

func TestJITDynamicNativeFuncPreservesClosureContextAcrossGC(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (callback value) (callback value))`)
	captured := NewString("captured native closure context")
	callback := NewFunc(func(args ...Scmer) Scmer {
		_ = jitCallbackTestSafepoint(args...)
		return NewSlice([]Scmer{captured, args[0]})
	})
	argument := NewString("dynamic argument")
	want := NewSlice([]Scmer{captured, argument})
	if got := Apply(compiled, callback, argument); !Equal(got, want) {
		t.Fatalf("dynamic native callback returned %s, want %s", String(got), String(want))
	}
}

func BenchmarkJITDynamicNativeFuncCall(b *testing.B) {
	compiled := compileJITExpressionTestProc(b, `(lambda (callback value) (callback value))`)
	callback := NewFunc(func(args ...Scmer) Scmer { return args[0] })
	argument := NewInt(42)
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		jitListBenchmarkSink = Apply(compiled, callback, argument)
	}
}

func TestJITOrdinaryBuiltinsDispatchThroughDeclarationEmitter(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	tests := []struct {
		name string
	}{
		{name: "jit-enabled?"},
		{name: "cdr"},
		{name: "error"},
	}
	for _, test := range tests {
		declaration := declarations[test.name]
		if declaration == nil || declaration.Type == nil || declaration.Type.JITEmit == nil {
			t.Fatalf("builtin %s has no declaration emitter", test.name)
		}
		original := declaration.Type.JITEmit
		called := func() (called bool) {
			declaration.Type.JITEmit = func(ctx *JITContext, args []Scmer, descs []JITValueDesc, result JITValueDesc) JITValueDesc {
				called = true
				return original(ctx, args, descs, result)
			}
			defer func() { declaration.Type.JITEmit = original }()
			params := NewSlice(nil)
			call := []Scmer{Globalenv.Vars[Symbol(test.name)]}
			if test.name != "jit-enabled?" {
				params = NewSlice([]Scmer{NewSymbol("value")})
				call = append(call, NewNthLocalVar(0))
			}
			compiled := jitCompile(NewProcStruct(Proc{
				Params:       params,
				Body:         NewSlice(call),
				En:           &Globalenv,
				NumVars:      len(call) - 1,
				NumberedOnly: true,
			}))
			if compiled.Proc() == nil || compiled.Proc().Compiled == nil {
				t.Fatalf("builtin %s test procedure did not compile", test.name)
			}
			return called
		}()
		if !called {
			t.Fatalf("builtin %s bypassed its declaration emitter", test.name)
		}
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
	callableType := &TypeDescriptor{Kind: "func", Return: &TypeDescriptor{Kind: "list"}}
	typedList := NewTypedFunc(List, RegisterCallableType(callableType))
	if got := typedList.CallableType(); got != callableType {
		t.Fatalf("retaining function lost callable type metadata: got %p, want %p", got, callableType)
	}
	first := Apply(compiled, typedList, NewInt(1))
	_ = Apply(compiled, typedList, NewInt(99))
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
	requireNoDynamicJITCalls(t, compiled)
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

func TestJITKnownLambdaMaterializesAtGeneratedBuiltinCallBoundary(t *testing.T) {
	declaration := declarations["map"]
	previous := declaration.Type.JITInlineCallbacks
	declaration.Type.JITInlineCallbacks = false
	defer func() { declaration.Type.JITInlineCallbacks = previous }()

	compiled := compileJITExpressionTestProc(t,
		`(lambda (values offset) (map values (lambda (value) (+ value offset))))`)
	if coverage := compiled.Proc().Compiled.Coverage; coverage.NativeCalls == 0 {
		t.Fatalf("expected generated map call boundary, got %+v", coverage)
	}
	got := Apply(compiled, NewSlice([]Scmer{NewInt(1), NewInt(2)}), NewInt(10))
	want := NewSlice([]Scmer{NewInt(11), NewInt(12)})
	if !Equal(got, want) {
		t.Fatalf("generated call-boundary lambda returned %s, want %s", String(got), String(want))
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

func TestJITExpressionStrLikeWithDynamicValue(t *testing.T) {
	compiled := compileJITExpressionTestProc(t,
		`(lambda (value) (strlike (concat value) "a%" "utf8mb4_general_ci"))`)
	requireNoDynamicJITCalls(t, compiled)
	if got := Apply(compiled, NewString("Alpha")); !Equal(got, NewBool(true)) {
		t.Fatalf("dynamic case-insensitive LIKE returned %s, want true", String(got))
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
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
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

func TestJITExpressionMergeInsideReducePreservesLoopHomes(t *testing.T) {
	// reduce keeps its accumulator and cursor live while its known callback is
	// emitted. merge has two independently planned loop cursors of its own. This
	// composition exercises nested register-home allocation rather than the much
	// easier case where merge owns the complete register bank.
	compiled := compileJITExpressionTestProc(t, `(lambda (groups)
		(reduce groups (lambda (acc group) (merge (list acc group))) '()))`)
	groupItems := make([]Scmer, 128)
	wantItems := make([]Scmer, 0, 256)
	for i := range groupItems {
		left, right := NewInt(int64(2*i)), NewInt(int64(2*i+1))
		groupItems[i] = NewSlice([]Scmer{left, right})
		wantItems = append(wantItems, left, right)
	}
	groups := NewSlice(groupItems)
	want := NewSlice(wantItems)
	if got := Apply(compiled, groups); !Equal(got, want) {
		t.Fatalf("nested merge returned %s, want %s", String(got), String(want))
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

func TestJITExpressionPreservesSelfEvaluatingEmptyList(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	root := NewProcStruct(Proc{
		Params:  NewSlice(nil),
		Body:    NewSlice(nil),
		En:      &Globalenv,
		NumVars: 0,
	})
	compiled := jitCompile(root)
	got := Apply(compiled)
	if !got.IsSlice() || len(got.Slice()) != 0 {
		t.Fatalf("self-evaluating empty list became %s, want empty list", SerializeToString(got, &Globalenv))
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

func TestJITSQLNotPreservesUnknownFromIn(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (value) (sql_not (sql_in (list ((lambda () nil)) 1) value)))`)
	if got := Apply(compiled, NewInt(0)); !got.IsNil() {
		t.Fatalf("JIT NOT IN returned %s, want nil", String(got))
	}
	if got := Apply(compiled, NewInt(1)); !got.IsBool() || got.Bool() {
		t.Fatalf("JIT NOT IN returned %s, want false", String(got))
	}
	if got := Apply(compiled, NewInt(2)); !got.IsNil() {
		t.Fatalf("JIT NOT IN returned %s, want nil", String(got))
	}
	rowProc := compileJITExpressionTestProc(t, `(lambda (resultrow)
		(resultrow (list "negated" (sql_not (sql_in (list ((lambda () nil)) 1) 0)))))`)
	got := Apply(rowProc, NewFunc(func(args ...Scmer) Scmer { return args[0] }))
	want := NewSlice([]Scmer{NewString("negated"), NewNil()})
	if !Equal(got, want) {
		t.Fatalf("JIT row NOT IN returned %s, want %s", String(got), String(want))
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
