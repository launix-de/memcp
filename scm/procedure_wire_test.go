/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/
package scm

import "fmt"
import "context"
import "bytes"
import "strings"
import "testing"
import "encoding/json"

func TestProcedureWireSnapshotsCapturesAndRunsWithoutSourceEnvironment(t *testing.T) {
	for _, optimize := range []bool{false, true} {
		for _, source := range []string{
			`(lambda (row) (+ row captured))`,
			`(lambda (row) (+ row (outer 1 captured)))`,
			`(lambda (row) (lambda (other) (+ row other)))`,
		} {
			env := &Env{Vars: Vars{Symbol("captured"): NewInt(7)}, Outer: &Globalenv}
			value := Read(t.Name(), source)
			if optimize {
				value = Optimize(value, env, nil)
			}
			original := Eval(value, env)
			data, err := MarshalProcedure(original)
			if err != nil {
				t.Fatalf("optimize=%v source=%s: %v", optimize, source, err)
			}
			// Mutation of the sender's frame after encoding must have no effect.
			env.Vars[Symbol("captured")] = NewInt(900)
			restored, err := UnmarshalProcedure(data)
			if err != nil {
				t.Fatal(err)
			}
			if restored.Proc().En != &Globalenv || restored.Proc().nativeCode() != 0 {
				t.Fatal("receiver retained source environment or executable code")
			}
			got := Apply(restored, NewInt(3))
			if got.IsProc() {
				got = Apply(got, NewInt(7))
			}
			if got.Int() != 10 {
				t.Fatalf("optimize=%v source=%s: got %v, wire=%s", optimize, source, got, data)
			}
		}
	}
}

func TestProcedureWireRejectsRuntimeCapturesAndUnclosedNestedBindings(t *testing.T) {
	for name, captured := range map[string]Scmer{
		"native closure": NewFunc(func(args ...Scmer) Scmer { return args[0] }),
		"session":        NewSession(),
		"Go pointer":     NewAny(new(int)),
	} {
		t.Run(name, func(t *testing.T) {
			env := &Env{Vars: Vars{Symbol("captured"): captured}, Outer: &Globalenv}
			value := Eval(Read(t.Name(), `(lambda () captured)`), env)
			if _, err := MarshalProcedure(value); err == nil {
				t.Fatal("process-local capture was accepted")
			}
		})
	}
	// CloseProcedure intentionally leaves nested lambda scopes intact. A
	// source-only wire encoding must reject a remaining external binding.
	env := &Env{Vars: Vars{Symbol("wire_nested_capture"): NewInt(7)}, Outer: &Globalenv}
	value := Eval(Read(t.Name(), `(lambda (row) (lambda () (+ row wire_nested_capture)))`), env)
	if _, err := MarshalProcedure(value); err == nil || !strings.Contains(err.Error(), "unclosed lexical capture") {
		t.Fatalf("unclosed nested capture: %v", err)
	}
}

func TestProcedureWireRejectsInvalidEnvelopesAndBounds(t *testing.T) {
	for _, data := range [][]byte{
		[]byte(`{"version":2,"code":null}`), []byte(`{"version":1,"code":17}`),
		[]byte(`{"version":1`), bytes.Repeat([]byte(" "), maxProcedureWireBytes+1),
	} {
		if _, err := UnmarshalProcedure(data); err == nil {
			t.Fatal("invalid envelope accepted")
		}
	}
	env := &Env{Vars: Vars{Symbol("large"): NewString(strings.Repeat("x", maxProcedureWireBytes))}, Outer: &Globalenv}
	if _, err := MarshalProcedure(Eval(Read(t.Name(), `(lambda () large)`), env)); err == nil {
		t.Fatal("oversized captured payload accepted")
	}
}

func TestProcedureWireJITCaptures(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (captured) (lambda (row) (+ row captured)))`)
	bound := Apply(compiled, NewInt(7))
	data, err := MarshalProcedure(bound)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := UnmarshalProcedure(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := Apply(restored, NewInt(3)); got.Int() != 10 {
		t.Fatalf("JIT capture roundtrip returned %v", got)
	}
	if restored.Proc().nativeCode() != 0 || restored.Proc().Compiled != nil {
		t.Fatal("wire retained process-local JIT metadata")
	}
}

func TestProcedureWireCapturedData(t *testing.T) {
	for _, captured := range []Scmer{
		NewNil(), NewSymbol("literal_symbol"), NewString("a\x00b"),
		NewSlice([]Scmer{NewSymbol("lambda"), NewSlice(nil), NewInt(7)}),
		NewSlice([]Scmer{NewString("key"), NewSlice([]Scmer{NewInt(17)})}),
	} {
		env := &Env{Vars: Vars{Symbol("captured"): captured}, Outer: &Globalenv}
		data, err := MarshalProcedure(Eval(Read(t.Name(), `(lambda () captured)`), env))
		if err != nil {
			t.Fatal(err)
		}
		value, err := UnmarshalProcedure(data)
		if err != nil {
			t.Fatal(err)
		}
		if got := Apply(value); !Equal(got, captured) {
			t.Fatalf("captured data changed: %v != %v", got, captured)
		}
	}
	Globalenv.Vars[Symbol("wire_test_global")] = NewInt(7)
	defer delete(Globalenv.Vars, Symbol("wire_test_global"))
	if _, err := MarshalProcedure(Eval(Read(t.Name(), `(lambda () wire_test_global)`), &Globalenv)); err == nil {
		t.Fatal("mutable global was implicitly rebound at the receiver")
	}
}

func TestProcedureWireUsesSchemeSerializeAndDoesNotEvaluateBody(t *testing.T) {
	original := Eval(Read(t.Name(), `(lambda (row) (+ row 7))`), &Globalenv)
	data, err := MarshalProcedure(original)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"source":"(lambda `)) || bytes.Contains(data, []byte(`"code"`)) {
		t.Fatalf("wire is not Scheme Serialize source: %s", data)
	}
	if _, err := UnmarshalProcedure([]byte(`{"version":2,"source":"(lambda () (error \"not during decoding\"))"}`)); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{
		`(begin (error "not a lambda") (lambda () 1))`,
		`(lambda () 1) (error "trailing program")`,
		`(lambda () 1 (error "frame is code"))`,
	} {
		data := []byte(fmt.Sprintf(`{"version":2,"source":%q}`, source))
		if _, err := UnmarshalProcedure(data); err == nil {
			t.Fatalf("accepted executable envelope: %s", source)
		}
	}
}

func TestProcedureWireStatementSessionBindingsAreFrozen(t *testing.T) {
	session := NewSession()
	session.Func()(NewString("a"), NewInt(7))
	session.Func()(NewString("b"), NewInt(8))
	env := &Env{Vars: Vars{Symbol("session"): session}, Outer: &Globalenv}
	export := func(object ClusterObject, view Scmer) (ClusterReference, error) {
		data, _ := json.Marshal(view)
		return ClusterReference{URI: "http://owner/cluster/test", Owner: "owner", Incarnation: "boot", Token: "handle", Type: object.ClusterObjectType(), View: data}, nil
	}
	bindings := &SerializationContext{Export: export}
	run := func(key string, bindings *SerializationContext) Scmer {
		proc := Eval(Read(t.Name(), `(lambda (x) (+ x (session "`+key+`")))`), env)
		data, err := MarshalProcedure(proc, bindings)
		if err != nil {
			t.Fatal(err)
		}
		restored, err := UnmarshalProcedure(data, ProcedureDecode{Resolve: func(ref ClusterReference) (Scmer, error) {
			var view Scmer
			_ = json.Unmarshal(ref.View, &view)
			return ReconstructClusterObject(ClusterHandle{Reference: ref, View: view})
		}})
		if err != nil {
			t.Fatal(err)
		}
		return Apply(restored, NewInt(1))
	}
	if got := run("a", bindings); got.Int() != 8 {
		t.Fatal(got)
	}
	session.Func()(NewString("b"), NewInt(99))
	if got := run("b", bindings); got.Int() != 9 {
		t.Fatal("one statement observed two session revisions", got)
	}
	if got := run("b", &SerializationContext{Export: export}); got.Int() != 100 {
		t.Fatal("new statement retained the old session revision", got)
	}
}

func TestProcedureWireRuntimeReferencesInsideQuotedCapturesKeepIdentity(t *testing.T) {
	session := NewSession()
	Apply(session, NewString("x"), NewInt(3))
	object, _ := ClusterObjectOf(session)
	bindings := &SerializationContext{Export: func(object ClusterObject, view Scmer) (ClusterReference, error) {
		data, _ := json.Marshal(view)
		return ClusterReference{URI: "http://owner/cluster/session", Token: "one", Type: object.ClusterObjectType(), View: data}, nil
	}}
	env := &Env{Vars: Vars{Symbol("captured"): NewSlice([]Scmer{NewSymbol("label"), session, session})}, Outer: &Globalenv}
	data, err := MarshalProcedure(Eval(Read(t.Name(), `(lambda () captured)`), env), bindings)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := UnmarshalProcedure(data, ProcedureDecode{Resolve: func(ref ClusterReference) (Scmer, error) {
		var view Scmer
		if err := json.Unmarshal(ref.View, &view); err != nil {
			return NewNil(), err
		}
		return ReconstructClusterObject(ClusterHandle{Reference: ref, View: view, Call: func(ctx context.Context, r ClusterRequest) (Scmer, error) { return ClusterObjectValue(object, ctx, r) }})
	}})
	if err != nil {
		t.Fatal(err)
	}
	values := Apply(restored).Slice()
	if !values[0].IsSymbol() || values[0].String() != "label" || values[1] != values[2] {
		t.Fatal("quoted runtime captures lost data or object identity")
	}
	Apply(values[1], NewString("x"), NewInt(41))
	if Apply(values[2], NewString("x")).Int() != 41 || Apply(session, NewString("x")).Int() != 41 {
		t.Fatal("shared accessor did not route mutation to its owner")
	}
}

// A standalone embedding can serialize its own value without a storage import,
// registered URI resolver, export callback or cluster listener.
type standaloneSerializableValue struct {
	calls int
	fail  bool
}

func (*standaloneSerializableValue) ClusterObjectType() string { return "standalone-test-value" }
func (v *standaloneSerializableValue) Serialize(*SerializationContext) (Scmer, error) {
	v.calls++
	if v.fail {
		return NewNil(), fmt.Errorf("standalone serialization rejected")
	}
	return NewSlice([]Scmer{NewSymbol("quote"), NewSlice([]Scmer{NewString("portable"), NewInt(42)})}), nil
}
func (*standaloneSerializableValue) HandleClusterRequest(context.Context, ClusterRequest) (any, error) {
	return nil, fmt.Errorf("standalone value needs no cluster requests")
}

func TestProcedureWireStandaloneObjectOwnsSerialization(t *testing.T) {
	object := &standaloneSerializableValue{}
	var source bytes.Buffer
	Serialize(&source, NewAny(object), &Globalenv)
	value := Eval(Read(t.Name(), source.String()), &Globalenv)
	if !Equal(value, NewSlice([]Scmer{NewString("portable"), NewInt(42)})) || object.calls != 1 {
		t.Fatal("standalone serializer did not dispatch to its object's representation", source.String(), object.calls)
	}
	proc := Eval(NewSlice([]Scmer{NewSymbol("lambda"), NewSlice(nil), NewAny(object)}), &Globalenv)
	data, err := MarshalProcedure(proc)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := UnmarshalProcedure(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := Apply(restored); !Equal(got, NewSlice([]Scmer{NewString("portable"), NewInt(42)})) {
		t.Fatal("procedure did not use object-owned reconstruction", got)
	}
	object.fail = true
	if _, err := MarshalProcedure(proc); err == nil || !strings.Contains(err.Error(), "standalone serialization rejected") {
		t.Fatal("object's serialization error was bypassed", err)
	}
}

func TestProcedureWireEmbeddingPreparesBeforeCompilation(t *testing.T) {
	proc := Eval(Read(t.Name(), `(lambda (x) (+ x 1))`), &Globalenv)
	data, err := MarshalProcedure(proc)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	restored, err := UnmarshalProcedure(data, ProcedureDecode{Prepare: func(expression Scmer) (Scmer, error) {
		calls++
		items := append([]Scmer(nil), expression.WithoutSourceInfo().Slice()...)
		items[2] = NewSlice([]Scmer{NewSymbol("list"), items[2], NewInt(9)})
		return NewSlice(items), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	got := Apply(restored, NewInt(6))
	if calls != 1 || !Equal(got, NewSlice([]Scmer{NewInt(7), NewInt(9)})) {
		t.Fatal("embedding preparation was not applied before compilation", calls, got)
	}
	if _, err := UnmarshalProcedure(data, ProcedureDecode{Prepare: func(Scmer) (Scmer, error) { return NewNil(), fmt.Errorf("embedding refused") }}); err == nil {
		t.Fatal("embedding preparation error was ignored")
	}
}

type eagerWireTestValue struct{}

func (*eagerWireTestValue) ClusterObjectType() string { return "eager-test-value" }
func (*eagerWireTestValue) Serialize(*SerializationContext) (Scmer, error) {
	return NewSlice([]Scmer{NewSymbol("wire_test_construct_capture_once")}), nil
}
func (*eagerWireTestValue) HandleClusterRequest(context.Context, ClusterRequest) (any, error) {
	return nil, fmt.Errorf("constructor fixture has no remote requests")
}
func TestProcedureWireMaterializesCapturesBeforeAnyInvocation(t *testing.T) {
	constructions := 0
	Declare(&Globalenv, &Declaration{Name: "wire_test_construct_capture_once", Fn: func(...Scmer) Scmer {
		constructions++
		return NewSession()
	}, Type: &TypeDescriptor{Kind: "func", HasSideEffects: true, Return: &TypeDescriptor{Kind: "any"}}})
	object := NewAny(&eagerWireTestValue{})
	body := NewSlice([]Scmer{NewSymbol("if"), NewSymbol("take"), object, NewNil()})
	proc := Eval(NewSlice([]Scmer{NewSymbol("lambda"), NewSlice([]Scmer{NewSymbol("take")}), body}), &Globalenv)
	data, err := MarshalProcedure(proc)
	if err != nil {
		t.Fatal(err)
	}
	if constructions != 0 {
		t.Fatal("constructor executed on sending node")
	}
	received, err := UnmarshalProcedure(data)
	if err != nil {
		t.Fatal(err)
	}
	// This is the empty-shard case: the callback has not run even once.
	if constructions != 1 {
		t.Fatal("capture was not materialized on import", constructions)
	}
	for range 10 {
		Apply(received, NewBool(false))
	}
	first := Apply(received, NewBool(true))
	if first != Apply(received, NewBool(true)) || constructions != 1 {
		t.Fatal("row execution constructed another peer object", constructions)
	}
}
