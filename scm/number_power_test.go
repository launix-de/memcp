/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import (
	"math"
	"testing"
)

func TestNumberPowerContracts(t *testing.T) {
	cases := []struct{ base, exponent, want Scmer }{
		{NewInt(3), NewInt(39), NewInt(4052555153018976267)},
		{NewInt(-2), NewInt(63), NewInt(math.MinInt64)},
		{NewInt(2), NewInt(63), NewFloat(9223372036854775808.0)},
		{NewInt(0), NewInt(0), NewInt(1)},
		{NewFloat(9), NewFloat(.5), NewFloat(3)},
		{NewInt(2), NewInt(-3), NewFloat(.125)},
		{NewNil(), NewInt(2), NewNil()},
	}
	declaration := declarations["number_power"]
	optimized := Eval(Optimize(Read(t.Name(), `(lambda (base exponent) (number_power base exponent))`, true), &Globalenv, nil), &Globalenv)
	for _, c := range cases {
		for _, fn := range []Scmer{Globalenv.Vars["number_power"], optimized} {
			got := Apply(fn, c.base, c.exponent)
			if !Equal(got, c.want) {
				t.Fatalf("%v^%v got %v want %v", c.base, c.exponent, got, c.want)
			}
			if c.want.IsInt() && !got.IsInt() {
				t.Fatal("exact integer result changed carrier")
			}
		}
		if got := declaration.Fn(c.base, c.exponent); !Equal(got, c.want) {
			t.Fatal("direct native result", got)
		}
	}
	if got := declaration.Fn(NewInt(-2), NewFloat(.5)); !math.IsNaN(got.Float()) {
		t.Fatal("real domain result", got)
	}
}

func TestNumberPowerJITBoundary(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires native JIT")
	}
	compiled := compileJITExpressionTestProc(t, `(lambda (base exponent) (number_power base exponent))`)
	requireNoDynamicJITCalls(t, compiled)
	if got := Apply(compiled, NewInt(3), NewInt(39)); !got.IsInt() || got.Int() != 4052555153018976267 {
		t.Fatal("compiled exact result", got)
	}
	if got := Apply(compiled, NewInt(2), NewInt(-3)); got.Float() != .125 {
		t.Fatal("compiled fractional result", got)
	}
}
