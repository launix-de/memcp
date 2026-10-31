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

func TestTreeNodeCountPreservesOccurrenceMetrics(t *testing.T) {
	shared := NewSlice([]Scmer{NewSymbol("leaf"), NewInt(7)})
	root := NewSlice([]Scmer{shared, shared, NewSlice(nil), NewNil()})
	count := Apply(Globalenv.Vars["tree_node_count"], root)
	if count.Int() != 9 {
		t.Fatalf("shared subtrees or empty/scalar leaves were discounted: %s", String(count))
	}
	legacy := Eval(Read("legacy tree metric", `(lambda (node)
		(match node
			(cons head tail) (+ 1 (tree_node_count head)
				(reduce tail (lambda (total item) (+ total (tree_node_count item))) 0))
			_ 1))`), &Globalenv)
	if got := Apply(legacy, root); got.Int() != count.Int() {
		t.Fatalf("native tree metric changed Scheme traversal semantics: %s != %s", String(count), String(got))
	}
}

func TestOptimizeBuiltinHookFoldsQuotedCode(t *testing.T) {
	expr := Read("optimize builtin hook", "(optimize '(+ 1 2))")
	optimized := Optimize(expr, &Globalenv, nil)
	result := Eval(optimized, &Globalenv)
	if ToInt(result) != 3 {
		t.Fatalf("optimized builtin call returned %s, want 3", String(result))
	}
}

func TestOptimizeTelemetryCallbackIsNotRunDuringOuterOptimization(t *testing.T) {
	expr := Read("optimize telemetry hook", `(optimize '(+ 1 2) (lambda (stats) (error "premature telemetry")))`)
	optimized := Optimize(expr, &Globalenv, nil)
	if serialized := serializedTestExpr(t, &Globalenv, optimized); !strings.Contains(serialized, "(optimize ") {
		t.Fatalf("telemetry call was folded before runtime: %s", serialized)
	}
}

func TestTryFoldRawExpressions(t *testing.T) {
	for _, tc := range []struct {
		code string
		want string
	}{
		{"(+ 1 (* 2 3))", "7"},
		{"(if 0 1)", "nil"},
		{"(and 0 (error \"unreachable\"))", "false"},
		{"(or 1 (error \"unreachable\"))", "true"},
		{"(if 1 (+ 2 3) (error \"unreachable\"))", "5"},
		{"(list (+ 1 2) (list 4 5))", "(3 (4 5))"},
		{"(quote (unknown_function unknown_variable))", "(unknown_function unknown_variable)"},
		{"(quote unknown_variable)", "unknown_variable"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			for _, optimize := range []bool{false, true} {
				expr := Read("tryfold", "(tryfold (quote "+tc.code+"))")
				if optimize {
					expr = Optimize(expr, &Globalenv, nil)
				}
				if got := String(Eval(expr, &Globalenv)); got != tc.want {
					t.Fatalf("optimize=%v: got %s, want %s", optimize, got, tc.want)
				}
			}
		})
	}
}

func TestTryFoldFailurePreservesAST(t *testing.T) {
	for _, code := range []string{
		"(+ unknown_variable 1)",
		"(now)",
		"(symbol \"unknown_variable\")",
		"(error \"cannot fold\")",
		"(and (error \"must evaluate first\") 0)",
		"(or (error \"must evaluate first\") 1)",
		"(begin (error \"must evaluate first\") 7)",
	} {
		t.Run(code, func(t *testing.T) {
			expr := Read("tryfold failure", code)
			before := serializedTestExpr(t, &Globalenv, expr)
			func() {
				defer func() {
					if recover() == nil {
						t.Fatal("nonconstant/throwing expression was folded")
					}
				}()
				TryFold(expr, &Globalenv)
			}()
			if after := serializedTestExpr(t, &Globalenv, expr); after != before {
				t.Fatalf("failed folding changed AST: before %s, after %s", before, after)
			}
			fallback := Read("tryfold fallback", "(try (lambda () (tryfold (quote "+code+"))) (lambda (_error) \"original\"))")
			if got := String(Eval(fallback, &Globalenv)); got != "original" {
				t.Fatalf("try did not recover folding failure: %s", got)
			}
		})
	}
}

func TestTryFoldUsesDeclarationFoldability(t *testing.T) {
	env := Env{Vars: make(Vars), Outer: &Globalenv}
	calls := 0
	fn := func(a ...Scmer) Scmer {
		calls++
		return NewInt(42)
	}
	name := "test_tryfold_declared_function"
	declaration := &Declaration{Name: name, Fn: fn,
		Type: &TypeDescriptor{Kind: "func", Return: &TypeDescriptor{Kind: "int"}}}
	Declare(&env, declaration)
	t.Cleanup(func() {
		delete(declarations, name)
		delete(declarationsByFunction, FunctionIdentity(fn))
		declaration_titles = declaration_titles[:len(declaration_titles)-1]
	})
	expr := NewSlice([]Scmer{NewSymbol(name)})
	for _, code := range []Scmer{
		expr,
		NewSlice([]Scmer{NewSymbol("begin"), expr, NewInt(7)}),
		NewSlice([]Scmer{NewSymbol("and"), expr, NewBool(false)}),
		NewSlice([]Scmer{NewSymbol("or"), expr, NewBool(true)}),
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("nonfoldable declaration was evaluated")
				}
			}()
			TryFold(code, &env)
		}()
	}
	if calls != 0 {
		t.Fatal("nonfoldable function was called")
	}
	declaration.Type.Const = true
	if got := ToInt(TryFold(expr, &env)); got != 42 || calls != 1 {
		t.Fatalf("foldable declaration: got %d, calls %d", got, calls)
	}
}
