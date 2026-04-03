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

func TestOptimizeProcToSerialFunctionExplicitNumVarsKeepsNamedParamBinding(t *testing.T) {
	lambda := Eval(Read("test", "(lambda ($update) ($update) 1)"), &Globalenv)
	called := false
	update := NewFunc(func(args ...Scmer) Scmer {
		called = true
		return NewInt(7)
	})
	got := OptimizeProcToSerialFunction(lambda)(update)
	if !called {
		t.Fatal("expected explicit-numvars callback to invoke bound parameter")
	}
	if ToInt(got) != 7 {
		t.Fatalf("expected callback result 7, got %v", got)
	}
}
