// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
package scm

import "testing"

func TestTextPositionOperators(t *testing.T) {
	for _, tc := range []struct {
		expression string
		args       []Scmer
		want       Scmer
	}{
		{`(lambda (s) (string_find s "x" 0 false "utf16"))`, []Scmer{NewString("雪😀x")}, NewInt(3)},
		{`(lambda (s) (string_find s "x" 0 false "rune"))`, []Scmer{NewString("雪😀x")}, NewInt(2)},
		{`(lambda (s) (string_find s "x" 0 true "rune"))`, []Scmer{NewString("KX")}, NewInt(1)},
		{`(lambda (s) (string_find s "x" 0 false "rune"))`, []Scmer{NewString("no")}, NewInt(-1)},
		{`(lambda (s) (string_find s "x" 1 false "utf16"))`, []Scmer{NewString("😀x")}, NewInt(2)},
		{`(lambda (s) (regexp_find s ".x$" false "utf16"))`, []Scmer{NewString("😀x")}, NewInt(1)},
		{`(lambda (s) (regexp_find s ".x$" false "rune"))`, []Scmer{NewString("😀x")}, NewInt(0)},
		{`(lambda (s) (regexp_find s "[0-9].*$" false "rune"))`, []Scmer{NewString("abc42")}, NewInt(3)},
		{`(lambda (s) (regexp_find s "A\\D" true "rune"))`, []Scmer{NewString("aa")}, NewInt(0)},
	} {
		plain := Eval(Read(t.Name(), tc.expression, true), &Globalenv)
		optimized := Eval(Optimize(Read(t.Name(), tc.expression, true), &Globalenv, nil), &Globalenv)
		compiled := CompileJIT(NewProcStruct(*calibrationProcedure(tc.expression)), true)
		for _, fn := range []Scmer{plain, optimized, compiled} {
			if got := Apply(fn, tc.args...); !Equal(got, tc.want) {
				t.Fatalf("%s: got %v want %v", tc.expression, got, tc.want)
			}
		}
	}
}
