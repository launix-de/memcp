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
package storage

import (
	"strings"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func scanHoistTestCall(condition scm.Scmer) []scm.Scmer {
	row := scm.NewNthLocalVar(0)
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("optimize"),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("if"),
			condition,
			scm.NewSlice([]scm.Scmer{scm.NewSymbol(">"), row, scm.NewInt(10)}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("<"), row, scm.NewInt(0)}),
		}),
	})
	filter := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("row")}),
		body,
		scm.NewInt(1),
	})
	return []scm.Scmer{
		scm.NewSymbol("scan"), scm.NewNil(), scm.NewSymbol("table_value"),
		scm.NewSlice([]scm.Scmer{scm.NewString("value")}), filter,
		scm.NewSlice([]scm.Scmer{scm.NewString("value")}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("row")}), row, scm.NewInt(1)}),
		scm.NewNil(), scm.NewNil(), scm.NewNil(), scm.NewBool(false),
	}
}

func TestScanHoistsInvariantFilterCondition(t *testing.T) {
	condition := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal??"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("outer"), scm.NewInt(1), scm.NewNthLocalVar(2)}),
		scm.NewInt(1),
	})
	rewritten := tryScanInvariantFilterRewrite(scanHoistTestCall(condition))
	if rewritten.IsNil() {
		t.Fatal("invariant filter condition was not hoisted")
	}
	serialized := scm.SerializeToString(rewritten, &scm.Globalenv)
	if !strings.Contains(serialized, "(if (equal?? (var 2) 1) (lambda") {
		t.Fatalf("condition was not lifted out of the lambda frame: %s", serialized)
	}
	if strings.Contains(serialized, "(outer 1 (var 2))") {
		t.Fatalf("lifted condition retained the inner lambda's outer hop: %s", serialized)
	}
}

func TestScanKeepsRowDependentFilterConditionInsideLambda(t *testing.T) {
	condition := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal??"), scm.NewNthLocalVar(0), scm.NewInt(1),
	})
	if rewritten := tryScanInvariantFilterRewrite(scanHoistTestCall(condition)); !rewritten.IsNil() {
		t.Fatalf("row-dependent condition was hoisted: %s", scm.SerializeToString(rewritten, &scm.Globalenv))
	}
}

func TestScanKeepsEffectfulFilterConditionInsideLambda(t *testing.T) {
	condition := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("scan_exists"), scm.NewNil(), scm.NewSymbol("other_table"),
		scm.NewSlice(nil),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice(nil), scm.NewBool(true)}),
	})
	if rewritten := tryScanInvariantFilterRewrite(scanHoistTestCall(condition)); !rewritten.IsNil() {
		t.Fatalf("table-reading condition was hoisted: %s", scm.SerializeToString(rewritten, &scm.Globalenv))
	}
}

func TestScanKeepsPotentiallyFailingFilterConditionInsideLambda(t *testing.T) {
	condition := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("/"), scm.NewInt(1), scm.NewInt(0),
	})
	if rewritten := tryScanInvariantFilterRewrite(scanHoistTestCall(condition)); !rewritten.IsNil() {
		t.Fatalf("potentially failing condition was hoisted: %s", scm.SerializeToString(rewritten, &scm.Globalenv))
	}
}

func TestScanKeepsUnknownBindingInsideLambda(t *testing.T) {
	condition := scm.NewSymbol("unknown_binding")
	if rewritten := tryScanInvariantFilterRewrite(scanHoistTestCall(condition)); !rewritten.IsNil() {
		t.Fatalf("unknown binding was hoisted: %s", scm.SerializeToString(rewritten, &scm.Globalenv))
	}
}

func BenchmarkScanInvariantFilterDispatch(b *testing.B) {
	condition := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	trueBranch := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		return scm.NewBool(args[0].Int() >= 500)
	})
	falseBranch := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		return scm.NewBool(args[0].Int() < 500)
	})
	perRow := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		if scm.ToBool(scm.Apply(condition)) {
			return scm.Apply(trueBranch, args...)
		}
		return scm.Apply(falseBranch, args...)
	})
	rows := make([]scm.Scmer, 1000)
	for i := range rows {
		rows[i] = scm.NewInt(int64(i))
	}

	b.Run("per_row_if", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, row := range rows {
				scm.Apply(perRow, row)
			}
		}
	})
	b.Run("hoisted_if", func(b *testing.B) {
		selected := falseBranch
		if scm.ToBool(scm.Apply(condition)) {
			selected = trueBranch
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, row := range rows {
				scm.Apply(selected, row)
			}
		}
	})
}
