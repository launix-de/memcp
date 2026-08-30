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

func TestOptimizeLowersAndSerializesSpecialForms(t *testing.T) {
	environment := &Env{Vars: Vars{Symbol("condition"): NewBool(true)}, Outer: &Globalenv}
	expression := Optimize(Read(t.Name(), "(if condition (quote yes) nil)"), environment, nil)
	items := expression.Slice()
	if items[0].GetTag() != tagSpecialForm || items[0].SpecialFormName() != "if" {
		t.Fatalf("if head was not lowered to a special form: %s", SerializeToString(expression, environment))
	}
	quoted := items[2].Slice()
	if quoted[0].GetTag() != tagSpecialForm || quoted[0].SpecialFormName() != "quote" {
		t.Fatalf("quote head was not lowered to a special form: %s", SerializeToString(expression, environment))
	}
	if serialized := SerializeToString(expression, environment); serialized != "(if condition (quote yes) nil)" {
		t.Fatalf("special forms did not serialize back to syntax: %s", serialized)
	}
	deoptimized := DeoptimizeExpr(expression).Slice()
	if !deoptimized[0].IsSymbol() || !deoptimized[0].SymbolEquals("if") {
		t.Fatalf("deoptimizer did not restore special form head: %s", SerializeToString(NewSlice(deoptimized), environment))
	}
	deoptimizedQuote := deoptimized[2].Slice()
	if !deoptimizedQuote[0].IsSymbol() || !deoptimizedQuote[0].SymbolEquals("quote") {
		t.Fatalf("deoptimizer did not restore nested special form: %s", SerializeToString(NewSlice(deoptimizedQuote), environment))
	}
	if _, exists := Globalenv.Vars[Symbol("if")]; exists {
		t.Fatal("special form leaked into ordinary global bindings")
	}
}

func TestEvalUnoptimizedSpecialForm(t *testing.T) {
	if result := Eval(Read(t.Name(), "(if true 1 0)"), &Globalenv); result.Int() != 1 {
		t.Fatalf("unoptimized special form returned %s", String(result))
	}
}

func TestSpecialFormNameOnlyWinsInCallHead(t *testing.T) {
	environment := &Env{Vars: Vars{Symbol("if"): NewInt(7)}, Outer: &Globalenv}
	if result := Eval(NewSymbol("if"), environment); result.Int() != 7 {
		t.Fatalf("special form shadowed lexical value: %s", String(result))
	}
	if result := Eval(Read(t.Name(), "(if true 1 0)"), environment); result.Int() != 1 {
		t.Fatalf("lexical value shadowed special form call: %s", String(result))
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
