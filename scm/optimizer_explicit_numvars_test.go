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
package scm

import "testing"
import "encoding/json"

func TestSerialProcUsesNumberedFixedParams(t *testing.T) {
	lambda := Eval(Optimize(Read("test", "(lambda (value) (+ value value))"), &Globalenv, nil), &Globalenv)
	if !lambda.Proc().NumberedOnly {
		t.Fatal("expected optimized lambda to use numbered bindings only")
	}
	got := serialTestCallable(lambda)(NewInt(6))
	if ToInt(got) != 12 {
		t.Fatalf("expected numbered callback result 12, got %v", got)
	}
}

func TestOptimizedLambdaPreservesNamesAndBindingProof(t *testing.T) {
	code := Optimize(Read("test", "(lambda (value) (list value (quote value)))"), &Globalenv, nil)
	proc := Eval(code, &Globalenv).Proc()
	if !proc.Params.Slice()[0].SymbolEquals("value") {
		t.Fatal("optimized lambda changed its parameter name")
	}
	if len(code.Slice()) != 5 || code.Slice()[4].GetTag() != tagAny {
		t.Fatal("optimized lambda must retain its precomputed binding proof")
	}
	Validate(code, "func")
	want := NewSlice([]Scmer{NewInt(6), NewSymbol("value")})
	if got := Apply(NewProc(proc), NewInt(6)); !Equal(got, want) {
		t.Fatalf("numbered argument and quoted symbol: got %s, want %s", String(got), String(want))
	}
}

func TestOptimizedLambdaRetainsAssignedNamedBinding(t *testing.T) {
	code := Optimize(Read("test", "(lambda (value) (begin (set value (+ value 1)) value))"), &Globalenv, nil)
	proc := Eval(code, &Globalenv)
	if !proc.Proc().Params.Slice()[0].SymbolEquals("value") {
		t.Fatal("assigned named parameter must retain its lexical binding")
	}
	if got := Apply(proc, NewInt(6)); ToInt(got) != 7 {
		t.Fatalf("assigned named parameter: got %s, want 7", String(got))
	}
}

func TestOptimizedBindingProofPreservesClosureSlots(t *testing.T) {
	code := Optimize(Read("test", "(lambda (value) (lambda (increment) (+ value increment)))"), &Globalenv, nil)
	outer := Eval(code, &Globalenv)
	first := Apply(outer, NewInt(6))
	second := Apply(outer, NewInt(9))
	if got := Apply(first, NewInt(2)); ToInt(got) != 8 {
		t.Fatalf("first closure: got %s, want 8", String(got))
	}
	if got := Apply(second, NewInt(3)); ToInt(got) != 12 {
		t.Fatalf("second closure: got %s, want 12", String(got))
	}
}

func TestAnonymousCallbackParamsKeepDistinctSlotTypes(t *testing.T) {
	parent := newOptimizerMetainfo()
	parent.pendingCallbackParams = []*TypeDescriptor{{Kind: "int"}, {Kind: "string"}}
	child := newOptimizerMetainfo()
	slots := 2
	child.nextSlot = &slots
	parent.applyPendingCallbackParams(NewSlice([]Scmer{NewSymbol("_"), NewSymbol("_")}), &child)
	for index, kind := range []string{"int", "string"} {
		got := child.numberedTypes[NthLocalVar(index)]
		if got == nil || got.Kind != kind {
			t.Fatalf("anonymous slot %d lost its %s callback type: %v", index, kind, got)
		}
	}
}

func TestOptimizedVariadicBindingKeepsArgumentOwnership(t *testing.T) {
	proc := Eval(Optimize(Read("test", "(lambda values (list values (quote values)))"), &Globalenv, nil), &Globalenv)
	args := []Scmer{NewInt(6), NewInt(9)}
	got := Apply(proc, args...)
	args[0] = NewInt(100)
	want := NewSlice([]Scmer{NewSlice([]Scmer{NewInt(6), NewInt(9)}), NewSymbol("values")})
	if !Equal(got, want) {
		t.Fatalf("optimized variadic binding: got %s, want %s", String(got), String(want))
	}
}

func TestOptimizedBindingProofRejectsExtraOperands(t *testing.T) {
	proc := Eval(Optimize(Read("test", "(lambda (value) (+ value 1))"), &Globalenv, nil), &Globalenv)
	defer func() {
		if recover() == nil {
			t.Fatal("binding metadata must not change fixed arity")
		}
	}()
	Eval(NewSlice([]Scmer{proc, NewInt(1), NewInt(2)}), &Globalenv)
}

func TestLambdaBindingProofRejectsRewrittenBody(t *testing.T) {
	code := Optimize(Read("test", "(lambda (value) (+ value 1))"), &Globalenv, nil)
	items := append([]Scmer(nil), code.Slice()...)
	items[2] = NewSymbol("value")
	proc := Eval(NewSlice(items), &Globalenv)
	if proc.Proc().NumberedOnly {
		t.Fatal("rewritten named body reused the old numbered-only proof")
	}
	if got := Apply(proc, NewInt(6)); ToInt(got) != 6 {
		t.Fatalf("rewritten named body: got %s, want 6", String(got))
	}
}

func TestLambdaBindingProofSerializationFallsBackSafely(t *testing.T) {
	code := Optimize(Read("test", "(lambda (value) (+ value 1))"), &Globalenv, nil)
	encoded, err := json.Marshal(code)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Scmer
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	// Text serialization of a procedure restores symbolic parameter references;
	// raw optimized AST text is a diagnostic display of numbered instructions.
	text := SerializeToString(Eval(code, &Globalenv), &Globalenv)
	for _, restored := range []Scmer{decoded, Read("test", text)} {
		Validate(restored, "func")
		if got := Apply(Eval(restored, &Globalenv), NewInt(6)); ToInt(got) != 7 {
			t.Fatalf("restored lambda: got %s, want 7", String(got))
		}
	}
}

func TestLambdaBindingProofCannotBeForgedBySource(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("ordinary source operands must still obey lambda arity")
		}
	}()
	Validate(Read("test", "(lambda (value) value 1 true)"), "func")
}

func TestSerialProcUsesNumberedVariadicParam(t *testing.T) {
	lambda := NewProcStruct(Proc{
		Params:       NewSymbol("values"),
		Body:         NewSlice([]Scmer{NewSymbol("list"), NewNthLocalVar(0)}),
		En:           &Globalenv,
		NumVars:      1,
		NumberedOnly: true,
	})
	got := serialTestCallable(lambda)(NewInt(3), NewInt(4))
	want := NewSlice([]Scmer{NewSlice([]Scmer{NewInt(3), NewInt(4)})})
	if !Equal(got, want) {
		t.Fatalf("expected numbered variadic callback result %v, got %v", want, got)
	}
}

func TestApplyVariadicProcOwnsRetainedArguments(t *testing.T) {
	for _, numbered := range []bool{false, true} {
		body := NewSymbol("values")
		numVars := 0
		if numbered {
			body = NewNthLocalVar(0)
			numVars = 1
		}
		lambda := NewProcStruct(Proc{
			Params: NewSymbol("values"), Body: body, En: &Globalenv,
			NumVars: numVars, NumberedOnly: numbered,
		})
		args := []Scmer{NewString("original"), NewInt(42)}
		got := Apply(lambda, args...)
		args[0] = NewString("reused")
		want := NewSlice([]Scmer{NewString("original"), NewInt(42)})
		if !Equal(got, want) {
			t.Fatalf("numbered=%t: returned arguments = %s, want %s", numbered, String(got), String(want))
		}
	}
}

func TestSerialProcExplicitNumVarsKeepsNamedParamBinding(t *testing.T) {
	lambda := Eval(Read("test", "(lambda ($update) ($update) 1)"), &Globalenv)
	called := false
	update := NewFunc(func(args ...Scmer) Scmer {
		called = true
		return NewInt(7)
	})
	got := serialTestCallable(lambda)(update)
	if !called {
		t.Fatal("expected explicit-numvars callback to invoke bound parameter")
	}
	if ToInt(got) != 7 {
		t.Fatalf("expected callback result 7, got %v", got)
	}
}

func TestSerialProcExplicitNumVarsKeepsNamedVariadicBinding(t *testing.T) {
	lambda := NewProcStruct(Proc{
		Params:  NewSymbol("values"),
		Body:    NewSymbol("values"),
		En:      &Globalenv,
		NumVars: 1,
	})
	got := serialTestCallable(lambda)(NewInt(3), NewInt(4))
	want := NewSlice([]Scmer{NewInt(3), NewInt(4)})
	if !Equal(got, want) {
		t.Fatalf("expected named variadic callback result %v, got %v", want, got)
	}
}

func TestSerialProcKeepsCompatibilitySlotsForNamedProc(t *testing.T) {
	lambda := NewProcStruct(Proc{
		Params:  NewSlice([]Scmer{NewSymbol("value")}),
		Body:    NewSlice([]Scmer{NewSymbol("list"), NewNthLocalVar(1)}),
		En:      &Globalenv,
		NumVars: 1,
	})
	got := serialTestCallable(lambda)(NewInt(3))
	if !Equal(got, NewSlice([]Scmer{NewNil()})) {
		t.Fatalf("expected an unbound compatibility slot to be nil, got %v", got)
	}
}

func BenchmarkSerialProcNumberedAdapter(b *testing.B) {
	body := NewNthLocalVar(0)
	for i := 0; i < 256; i++ {
		body = NewSlice([]Scmer{NewSymbol("+"), body, NewInt(1)})
	}
	proc := NewProcStruct(Proc{
		Params:       NewSlice([]Scmer{NewSymbol("value")}),
		Body:         body,
		En:           &Globalenv,
		NumVars:      1,
		NumberedOnly: true,
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		serialTestCallable(proc)
	}
}
