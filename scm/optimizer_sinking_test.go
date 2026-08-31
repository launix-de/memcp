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
	"strings"
	"testing"
)

func TestOptimizeSinksSingleUseClosureIntoOnceCallback(t *testing.T) {
	env := newOptimizerTestEnv()
	optimized := Optimize(Read(t.Name(), `(lambda (session captured)
		(begin
			(define helper (lambda () (+ captured 1)))
			(with_session session (lambda () (helper)))))`), env, nil)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "setN") || strings.Contains(serialized, "define helper") {
		t.Fatalf("single-use closure stayed outside once callback: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	session := Eval(Read(t.Name(), `(newsession)`), env)
	if got := fn(session, NewInt(8)); !Equal(got, NewInt(9)) {
		t.Fatalf("sunk closure returned %s, want 9", String(got))
	}
}

func TestOptimizeKeepsClosureOutsideRepeatedCallback(t *testing.T) {
	env := newOptimizerTestEnv()
	optimized := Optimize(Read(t.Name(), `(lambda (session values captured)
		(begin
			(define helper (lambda (value) (+ captured value)))
			(map values (lambda (value)
				(with_session session (lambda () (helper value)))))))`), env, nil)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "setN") {
		t.Fatalf("inner once callback hid its repeated parent: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	session := Eval(Read(t.Name(), `(newsession)`), env)
	got := fn(session, NewSlice([]Scmer{NewInt(1), NewInt(2)}), NewInt(8))
	want := NewSlice([]Scmer{NewInt(9), NewInt(10)})
	if !Equal(got, want) {
		t.Fatalf("retained closure returned %s, want %s", String(got), String(want))
	}
}

func TestOptimizeKeepsBindingClosureAheadOfOnceCallback(t *testing.T) {
	env := newOptimizerTestEnv()
	optimized := Optimize(Read(t.Name(), `(lambda (session captured)
		(begin
			(define helper (lambda () (begin
				(define incremented (+ captured 1))
				incremented)))
			(with_session session (lambda () (helper)))))`), env, nil)
	serialized := serializedTestExpr(t, env, optimized)
	if helperAt, operatorAt := strings.Index(serialized, "setN"), strings.Index(serialized, "with_session"); helperAt < 0 || operatorAt < 0 || helperAt > operatorAt {
		t.Fatalf("binding closure crossed callback scope: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	session := Eval(Read(t.Name(), `(newsession)`), env)
	if got := fn(session, NewInt(8)); !Equal(got, NewInt(9)) {
		t.Fatalf("retained binding closure returned %s, want 9", String(got))
	}
}

func TestOptimizeSinksThroughTypedNativeCallable(t *testing.T) {
	env := newOptimizerTestEnv()
	cacheType := &TypeDescriptor{Kind: "func", Params: []*TypeDescriptor{
		{Kind: "any"},
		{Kind: "func", CallsOnce: true, Params: []*TypeDescriptor{}, Return: &TypeDescriptor{Kind: "any"}},
	}, Return: &TypeDescriptor{Kind: "any"}}
	env.Vars[Symbol("typed_cache")] = NewTypedFunc(func(...Scmer) Scmer {
		return NewInt(11)
	}, RegisterCallableType(cacheType))
	optimized := Optimize(Read(t.Name(), `(lambda (captured)
		(begin
			(define helper (lambda () (+ captured 1)))
			(typed_cache "key" (lambda () (helper)))))`), env, nil)
	serialized := serializedTestExpr(t, env, optimized)
	if strings.Contains(serialized, "setN") || strings.Contains(serialized, "define helper") {
		t.Fatalf("typed dynamic callable did not expose once callback: %s", serialized)
	}

	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	if got := fn(NewInt(8)); !Equal(got, NewInt(11)) {
		t.Fatalf("typed callable returned %s, want 11", String(got))
	}
}

func TestOptimizeKeepsClosureAheadOfTwoExecutedUses(t *testing.T) {
	env := newOptimizerTestEnv()
	optimized := Optimize(Read(t.Name(), `(lambda (captured)
		(begin
			(define helper (lambda () (+ captured 1)))
			(list (helper) (helper))))`), env, nil)
	serialized := serializedTestExpr(t, env, optimized)
	if !strings.Contains(serialized, "setN") {
		t.Fatalf("closure used twice on one path was duplicated: %s", serialized)
	}
}

func BenchmarkOptimizedOnceCallbackClosureSinking(b *testing.B) {
	env := newOptimizerTestEnv()
	optimized := Optimize(Read(b.Name(), `(lambda (session captured)
		(begin
			(define helper (lambda () (+ captured 1)))
			(with_session session (lambda () (helper)))))`), env, nil)
	fn := OptimizeProcToSerialFunction(Eval(optimized, env))
	session := Eval(Read(b.Name(), `(newsession)`), env)
	args := []Scmer{session, NewInt(8)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := fn(args...); !Equal(got, NewInt(9)) {
			b.Fatal(got)
		}
	}
}
