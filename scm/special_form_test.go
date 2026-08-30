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
	"encoding/json"
	"testing"
)

func TestOptimizeLowersAndSerializesSpecialForms(t *testing.T) {
	environment := &Env{Vars: Vars{Symbol("condition"): NewBool(true)}, Outer: &Globalenv}
	expression := Optimize(Read(t.Name(), "(and condition (quote yes))"), environment, nil)
	items := expression.Slice()
	if items[0].GetTag() != tagSpecialForm || items[0].SpecialFormName() != "and" {
		t.Fatalf("and head was not lowered to a special form: %s", SerializeToString(expression, environment))
	}
	quoted := items[2].Slice()
	if quoted[0].GetTag() != tagSpecialForm || quoted[0].SpecialFormName() != "quote" {
		t.Fatalf("quote head was not lowered to a special form: %s", SerializeToString(expression, environment))
	}
	if serialized := SerializeToString(expression, environment); serialized != "(and condition (quote yes))" {
		t.Fatalf("special forms did not serialize back to syntax: %s", serialized)
	}
	deoptimized := DeoptimizeExpr(expression).Slice()
	if !deoptimized[0].IsSymbol() || !deoptimized[0].SymbolEquals("and") {
		t.Fatalf("deoptimizer did not restore special form head: %s", SerializeToString(NewSlice(deoptimized), environment))
	}
	deoptimizedQuote := deoptimized[2].Slice()
	if !deoptimizedQuote[0].IsSymbol() || !deoptimizedQuote[0].SymbolEquals("quote") {
		t.Fatalf("deoptimizer did not restore nested special form: %s", SerializeToString(NewSlice(deoptimizedQuote), environment))
	}
	if value := Globalenv.Vars[Symbol("if")]; value.GetTag() != tagSpecialForm {
		t.Fatal("special form was not registered in the global environment")
	}
}

func TestSpecialFormSharesDeclarationOptimizerAndJITMetadata(t *testing.T) {
	oldTitles, oldDeclarations, oldFunctions := declaration_titles, declarations, declarationsByFunction
	declaration_titles = nil
	declarations = make(map[string]*Declaration)
	declarationsByFunction = make(map[uintptr]*Declaration)
	defer func() {
		declaration_titles, declarations, declarationsByFunction = oldTitles, oldDeclarations, oldFunctions
	}()

	environment := &Env{Vars: make(Vars), Outer: &Globalenv}
	hookCalled := false
	definition := &Declaration{
		Name: "special-declaration-hook-test",
		Type: &TypeDescriptor{
			Kind:   "func",
			Return: &TypeDescriptor{Kind: "int"},
			JITEmit: func(_ *JITContext, _ []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
				return result
			},
		},
		Optimize: func(_ []Scmer, _ *OptimizerContext, _ bool) (Scmer, *TypeDescriptor) {
			hookCalled = true
			return NewInt(42), &TypeDescriptor{Kind: "int", Const: true, Transfer: true}
		},
	}
	DeclareSpecialForm(environment, definition, func(_ []Scmer, _ *Env) Scmer { return NewInt(1) }, definition.Type.JITEmit)
	value := environment.Vars[Symbol(definition.Name)]
	defer delete(specialFormNames, value.ptr)

	if resolved := DeclarationForValue(value); resolved != definition {
		t.Fatalf("special form resolved declaration %p, want %p", resolved, definition)
	}
	if DeclarationForValue(value).Type.JITEmit == nil {
		t.Fatal("special form declaration lost its JIT emitter")
	}
	optimized := Optimize(NewSlice([]Scmer{NewSymbol(definition.Name)}), environment, nil)
	if !hookCalled {
		t.Fatal("special form declaration optimizer hook was not called")
	}
	if !optimized.IsInt() || optimized.Int() != 42 {
		t.Fatalf("special form declaration optimizer returned %s, want 42", String(optimized))
	}
}

func TestEveryGlobalSpecialFormHasDeclarationJITHooks(t *testing.T) {
	for symbol, value := range Globalenv.Vars {
		if value.GetTag() != tagSpecialForm {
			continue
		}
		definition := DeclarationForValue(value)
		if definition == nil || !definition.IsSpecialForm {
			t.Fatalf("global special form %s has no special-form declaration", symbol)
		}
		if definition.Type == nil || definition.Type.JITEmit == nil {
			t.Fatalf("global special form %s has no declaration JIT emitter", symbol)
		}
	}
	for _, name := range []string{"if", "and", "or"} {
		definition := declarations[name]
		if definition == nil || definition.Type == nil || definition.Type.JITEmitCond == nil {
			t.Fatalf("lazy boolean special form %s has no declaration branch emitter", name)
		}
	}
}

func TestEvalUnoptimizedSpecialForm(t *testing.T) {
	if result := Eval(Read(t.Name(), "(if true 1 0)"), &Globalenv); result.Int() != 1 {
		t.Fatalf("unoptimized special form returned %s", String(result))
	}
}

func TestSpecialFormJSONRestoresSymbol(t *testing.T) {
	original := Globalenv.Vars[Symbol("lambda")]
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored Scmer
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if !restored.IsSymbol() || !restored.SymbolEquals("lambda") {
		t.Fatalf("special form JSON did not restore syntax symbol: %s", String(restored))
	}
}

func TestSpecialFormDispatchFollowsSymbolResolution(t *testing.T) {
	localIf := NewFunc(func(...Scmer) Scmer { return NewInt(7) })
	environment := &Env{Vars: Vars{Symbol("if"): localIf}, Outer: &Globalenv}
	if result := Eval(NewSymbol("if"), environment); result != localIf {
		t.Fatalf("special form shadowed lexical value: %s", String(result))
	}
	if result := Eval(Read(t.Name(), "(if true 1 0)"), environment); result.Int() != 7 {
		t.Fatalf("call did not use resolved lexical value: %s", String(result))
	}
}

func TestOptimizerKeepsLambdaSyntaxAndLowersItsBody(t *testing.T) {
	expression := Optimize(Read(t.Name(), "(lambda (value) (outer value))"), &Globalenv, nil)
	items := expression.Slice()
	if !items[0].IsSymbol() || !items[0].SymbolEquals("lambda") {
		t.Fatalf("lambda scope boundary was lowered before callback analysis: %s", SerializeToString(expression, &Globalenv))
	}
	body := items[2].Slice()
	if body[0].GetTag() != tagSpecialForm || body[0].SpecialFormName() != "outer" {
		t.Fatalf("compiler-internal outer form remained shadowable: %s", SerializeToString(expression, &Globalenv))
	}
}

func TestOptimizerDoesNotLowerConstructedSyntaxData(t *testing.T) {
	environment := &Env{Vars: make(Vars), Outer: &Globalenv}
	expression := Optimize(Read(t.Name(), "(list (quote if) true 1 0)"), environment, nil)
	value := Eval(expression, environment)
	items := value.Slice()
	if !items[0].IsSymbol() || !items[0].SymbolEquals("if") {
		t.Fatalf("constructed syntax head became executable special form: %s", SerializeToString(value, environment))
	}
}

func TestSpecialFormTailRestart(t *testing.T) {
	environment := &Env{Vars: make(Vars), Outer: &Globalenv}
	EvalAll(t.Name(), `(define special_form_tail_restart
		(lambda (n)
			(if (> n 0)
				(special_form_tail_restart (- n 1))
				n)))`, environment)
	result := Apply(environment.Vars[Symbol("special_form_tail_restart")], NewInt(10000))
	if result.Int() != 0 {
		t.Fatalf("tail-recursive special form returned %s", String(result))
	}
}
