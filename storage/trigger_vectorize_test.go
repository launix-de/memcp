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
package storage

import (
	"testing"

	"github.com/launix-de/memcp/scm"
)

// TestVectorizeTriggerDeletePattern tests that the DELETE trigger pattern is recognized.
func TestVectorizeTriggerDeletePattern(t *testing.T) {
	// Build a trigger body that matches the prejoin DELETE pattern:
	// (lambda (OLD NEW) (scan schema tbl (list "grp") (lambda (_pj.grp) (equal? _pj.grp (get_assoc OLD "grp"))) (list "$update") (lambda ($update) ($update)) + 0 nil false))
	filterBody := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal?"),
		scm.NewSymbol("_pj.grp"),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("get_assoc"),
			scm.NewSymbol("OLD"),
			scm.NewString("grp"),
		}),
	})
	filterFn := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("_pj.grp")}),
		Body:   filterBody,
		En:     &scm.Env{Vars: make(scm.Vars)},
	})

	scanBody := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("scan"),
		scm.NewString("mydb"),
		scm.NewString(".prejoin:mytable"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString("grp")}),
		filterFn,
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString("$update")}),
		scm.NewProcStruct(scm.Proc{
			Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("$update")}),
			Body:   scm.NewSlice([]scm.Scmer{scm.NewSymbol("$update")}),
			En:     &scm.Env{Vars: make(scm.Vars)},
		}),
		scm.NewSymbol("+"),
		scm.NewInt(0),
		scm.NewNil(),
		scm.NewBool(false),
	})

	triggerProc := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("OLD"), scm.NewSymbol("NEW")}),
		Body:   scanBody,
		En:     &scm.Env{Vars: make(scm.Vars)},
	})

	// Test key extraction
	key := extractGetAssocOldKey(filterFn)
	if key != "grp" {
		t.Fatalf("expected key 'grp', got '%s'", key)
	}

	// Test param index
	idx := findEqualParamIdx(filterFn)
	if idx != 0 {
		t.Fatalf("expected param idx 0, got %d", idx)
	}

	// Test vectorization
	vf := VectorizeTrigger(triggerProc)
	if vf.IsNil() {
		t.Fatal("expected vectorized function, got nil")
	}
	t.Log("Vectorization succeeded")
}

// TestVectorizeTriggerNonMatchingPattern tests that non-DELETE patterns return nil.
func TestVectorizeTriggerNonMatchingPattern(t *testing.T) {
	// A simple (droptable ...) body — should not vectorize
	body := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("droptable"),
		scm.NewString("mydb"),
		scm.NewString("mytable"),
		scm.NewBool(true),
	})
	proc := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("OLD"), scm.NewSymbol("NEW")}),
		Body:   body,
		En:     &scm.Env{Vars: make(scm.Vars)},
	})

	vf := VectorizeTrigger(proc)
	if !vf.IsNil() {
		t.Fatal("expected nil for non-matching pattern")
	}
}
