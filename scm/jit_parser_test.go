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

func TestJITParserGrammarMatchesPackrat(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	environment := &Env{Vars: make(Vars), Outer: &Globalenv}
	parserValue := Eval(Read("jit parser grammar", `(parser '(
		(atom "SELECT" true)
		(define values (+ (regex "[a-z]+" false true) ","))
		$
	) values)`), environment)
	environment.Vars[Symbol("test_parser")] = parserValue
	parser := parserValue.Parser()
	want := parser.Execute(" select alpha,beta,gamma ", environment)

	jitCompileEnvironmentParsers(environment)
	if parser.Compiled == nil {
		t.Fatal("parser grammar was not compiled")
	}
	if parser.JITProgram == nil || !parser.JITProgram.inlineActions {
		t.Fatal("parser grammar did not inline generator actions")
	}
	if action := parser.JITProgram.rules[parser.JITRule].action; !action.IsNil() {
		t.Fatalf("compiled parser retained an Apply action: %s", String(action))
	}
	got := parser.Execute(" select alpha,beta,gamma ", environment)
	if !Equal(got, want) {
		t.Fatalf("compiled parser returned %s, want %s", String(got), String(want))
	}
}

func TestJITParserGrammarSpillsWideGeneratorCaptures(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	environment := &Env{Vars: make(Vars), Outer: &Globalenv}
	parserValue := Eval(Read("wide jit parser grammar", `(parser '(
		(define a (regex "[a-z]+" false false)) ","
		(define b (regex "[a-z]+" false false)) ","
		(define c (regex "[a-z]+" false false)) ","
		(define d (regex "[a-z]+" false false)) ","
		(define e (regex "[a-z]+" false false)) ","
		(define f (regex "[a-z]+" false false)) ","
		(define g (regex "[a-z]+" false false)) ","
		(define h (regex "[a-z]+" false false)) $)
		(list a b c d e f g h) "")`), environment)
	environment.Vars[Symbol("wide_parser")] = parserValue
	parser := parserValue.Parser()
	want := parser.Execute("a,b,c,d,e,f,g,h", environment)

	jitCompileEnvironmentParsers(environment)
	if parser.Compiled == nil {
		t.Fatal("wide parser grammar was not compiled")
	}
	got := parser.Execute("a,b,c,d,e,f,g,h", environment)
	if !Equal(got, want) {
		t.Fatalf("compiled wide parser returned %s, want %s", String(got), String(want))
	}
}

func TestJITParserTemplatesFuseNamedCaptures(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (input) (begin
		(define word (parser (regex "[a-z]+" false false)))
		((parser '((define first word) "," (define second word) $)
			(list first second) "") input)))`)
	requireNoDynamicJITCalls(t, compiled)
	got := compiled.Proc().Compiled.Call(NewString("alpha,beta"))
	want := NewSlice([]Scmer{NewString("alpha"), NewString("beta")})
	if got.GetTag() != tagSlice || !Equal(got, want) {
		t.Fatalf("fused parser returned tag=%d aux=%x, want %s; body=%s", got.GetTag(), got.aux, String(want), SerializeToString(compiled.Proc().Body, &Globalenv))
	}
}

func TestJITParserGeneratorKeepsNestedCaptureListsDistinct(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (input) (begin
		(define word (parser (regex "[a-z]+" false false)))
		((parser '(
			(define column word) ":" (define kind word)
			(define dimensions (parser empty '((quote list))))
			(define typeparams (parser empty (list)))
			$)
			'((quote list) "column" column kind dimensions (cons (quote list) typeparams)) "") input)))`)
	requireNoDynamicJITCalls(t, compiled)
	got := compiled.Proc().Compiled.Call(NewString("username:text"))
	want := NewSlice([]Scmer{
		NewSymbol("list"), NewString("column"), NewString("username"), NewString("text"),
		NewSlice([]Scmer{NewSymbol("list")}), NewSlice([]Scmer{NewSymbol("list")}),
	})
	if !Equal(got, want) {
		t.Fatalf("fused parser returned %s, want %s", String(got), String(want))
	}
}

func TestJITParserGeneratorReadsAncestorAndOuterScopes(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (input prefix) (begin
		(define word (parser (regex "[a-z]+" false false)))
		((parser '(
			(define left word) ":"
			(define nested (parser '((define right word))
				(list prefix left right) ""))
			$)
			nested "") input)))`)
	requireNoDynamicJITCalls(t, compiled)
	got := compiled.Proc().Compiled.Call(NewString("alpha:beta"), NewString("scope"))
	want := NewSlice([]Scmer{NewString("scope"), NewString("alpha"), NewString("beta")})
	if !Equal(got, want) {
		t.Fatalf("nested generator returned %s, want %s", String(got), String(want))
	}
}

func TestJITParserRecursiveRightBranchKeepsOptionalBindings(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (input) (begin
		(define word (parser (regex "[A-Z]" false true)))
		(define combine (lambda (left right) (list left right)))
		(define core (parser '(
			(define value word)
			(? (atom "WHERE" true) (define condition word))
			(? (atom "ORDER" true) (define order word))
			(? (atom "LIMIT" true) (define limit word)))
			(list value condition order limit)))
		(define select (parser (or
			(parser '((define left core) (atom "UNION" true) (define right select))
				(combine left right))
			core)))
		((parser (define command select) command) input)))`)
	requireNoDynamicJITCalls(t, compiled)
	got := compiled.Proc().Compiled.Call(NewString("A UNION B WHERE C ORDER D LIMIT E"))
	want := NewSlice([]Scmer{
		NewSlice([]Scmer{NewString("A"), NewNil(), NewNil(), NewNil()}),
		NewSlice([]Scmer{NewString("B"), NewString("C"), NewString("D"), NewString("E")}),
	})
	if !Equal(got, want) {
		t.Fatalf("recursive parser returned %s, want %s", String(got), String(want))
	}
}
