/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "testing"

func TestLambdaRebindScalarContextsShareCode(t *testing.T) {
	env := &Env{Vars: Vars{Symbol("captured"): NewInt(99)}, Outer: &Globalenv}
	old := Eval(Read(t.Name(), `(lambda (x) (+ x captured))`, false), env)
	var native uintptr
	for i := 0; i < 64; i++ {
		rebound := LambdaRebind(old, NewInt(int64(i)))
		if got := Apply(rebound, NewInt(7)); got.Int() != int64(i+7) {
			t.Fatal("wrong rebound value", i, got)
		}
		if jitEnabled {
			if i == 0 {
				native = rebound.Proc().nativeCode()
			}
			if native == 0 || rebound.Proc().nativeCode() != native {
				t.Fatal("recompiled code per binding")
			}
		}
	}
	if got := Apply(old, NewInt(7)); got.Int() != 106 {
		t.Fatal("rebinding changed original", got)
	}
}
func TestLambdaRebindNestedAndNumberedContexts(t *testing.T) {
	for _, source := range []string{`(lambda (x) ((lambda (y) (+ y captured)) x))`, `(lambda (x) (begin (+ x captured)))`, `(lambda (x) (match x 0 captured _ (+ x captured)))`, `(lambda args (+ (nth args 0) captured))`, `(lambda (x) (eval (list (quote +) x captured)))`} {
		env := &Env{Vars: Vars{Symbol("captured"): NewInt(9)}, Outer: &Globalenv}
		old := Eval(Read(t.Name(), source, false), env)
		rebound := LambdaRebind(old, NewInt(31))
		if got := Apply(rebound, NewInt(7)); got.Int() != 38 {
			t.Fatal(source, got)
		}
	}
}

func TestLambdaRebindVariadicCodeIsShared(t *testing.T) {
	old := Eval(Read(t.Name(), `(lambda args (+ (nth args 0) captured))`, false), &Env{Vars: Vars{Symbol("captured"): NewInt(99)}, Outer: &Globalenv})
	var code uintptr
	for i := 0; i < 64; i++ {
		value := LambdaRebind(old, NewInt(int64(i)))
		if Apply(value, NewInt(7), NewInt(11)).Int() != int64(i+7) {
			t.Fatal("variadic binding changed")
		}
		if jitEnabled {
			if i == 0 {
				code = value.Proc().nativeCode()
			}
			if code == 0 || code != value.Proc().nativeCode() {
				t.Fatal("variadic lambda was recompiled per binding")
			}
		}
	}
}
