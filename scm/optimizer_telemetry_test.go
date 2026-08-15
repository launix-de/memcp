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
