//go:build goexperiment.jit && amd64

// Copyright (C) 2026 MemCP Contributors
// SPDX-License-Identifier: GPL-3.0-or-later
package scm

import "testing"

func TestJITResolvedSpecialFormLiteral(t *testing.T) {
	// Query-cache preparations build executable syntax as data. Optimization
	// resolves !begin to a special-form value before the enclosing thunk is JITed.
	syntax, ok := Globalenv.Vars[Symbol("!begin")]
	if !ok {
		t.Fatal("!begin not registered")
	}
	var resolve func(Scmer) Scmer
	resolve = func(value Scmer) Scmer {
		value = value.WithoutSourceInfo()
		if value.IsSymbol() && value.Symbol() == Symbol("!begin") {
			return syntax
		}
		if value.IsSlice() {
			items := append([]Scmer(nil), value.Slice()...)
			for i := range items {
				items[i] = resolve(items[i])
			}
			return NewSlice(items)
		}
		return value
	}
	for index, source := range []string{
		`(lambda () !begin)`,
		`(lambda (x) (list !begin x))`,
		`(lambda (x) (lambda () (list !begin x)))`,
	} {
		expression := resolve(Optimize(Read(t.Name(), source), &Globalenv, nil))
		proc := jitCompile(Eval(expression, &Globalenv))
		if proc.Proc() == nil || proc.Proc().JITCode == 0 {
			t.Fatalf("resolved literal did not compile: %s", source)
		}
		var args []Scmer
		if index > 0 {
			args = []Scmer{NewInt(7)}
		}
		got := Apply(proc, args...)
		if got.IsProc() {
			got = Apply(got)
		}
		if got.IsSlice() {
			parts := got.Slice()
			if len(parts) != 2 || parts[1].Int() != 7 {
				t.Fatalf("invalid syntax payload: %v", got)
			}
			got = parts[0]
		}
		if !Equal(got, syntax) {
			t.Fatalf("resolved syntax changed: %v", got)
		}
	}
}
