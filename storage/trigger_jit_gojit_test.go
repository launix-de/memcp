//go:build goexperiment.jit && amd64

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

package storage

import (
	"testing"

	"github.com/launix-de/memcp/scm"
)

func requireCompiledTrigger(t *testing.T, trigger TriggerDescription) {
	t.Helper()
	if !trigger.Func.IsProc() || trigger.Func.Proc() == nil {
		t.Fatalf("trigger function is not a procedure: %s", scm.String(trigger.Func))
	}
	if trigger.Func.Proc().Compiled == nil || trigger.Func.Proc().JITCode == 0 {
		t.Fatal("trigger procedure has no native entry point")
	}
	if got := scm.Apply(trigger.Func, scm.NewNil(), scm.NewNil(), scm.NewNil(), scm.NewNil()); !scm.Equal(got, scm.NewInt(42)) {
		t.Fatalf("compiled trigger result = %s, want 42", scm.String(got))
	}
}

func triggerJITTestPlan() scm.Scmer {
	return scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("OLD"), scm.NewSymbol("NEW"),
			scm.NewSymbol("session"), scm.NewSymbol("tx"),
		}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("+"), scm.NewInt(40), scm.NewInt(2),
		}),
	})
}

func TestFinalizeTriggerCompilationUsesJIT(t *testing.T) {
	trigger := TriggerDescription{
		Name:     "direct",
		Func:     scm.Eval(triggerJITTestPlan(), &scm.Globalenv),
		Language: "sql",
	}
	finalizeTriggerCompilation(&trigger)
	requireCompiledTrigger(t, trigger)
}

func TestLazyTriggerPlanUsesJIT(t *testing.T) {
	trigger := TriggerDescription{
		Name:     "lazy",
		FuncPlan: triggerJITTestPlan(),
		Language: "sql",
	}
	compileTriggerForUse("test", "items", &trigger)
	requireCompiledTrigger(t, trigger)
}

func TestLazyTriggerPlanIsOptimizedBeforeJIT(t *testing.T) {
	trigger := TriggerDescription{
		Name: "lazy-mutation",
		FuncPlan: scm.Read(t.Name(), `(lambda (OLD NEW session tx) (begin
			(define changed_rows NEW)
			(!begin
				(set changed_rows (set_assoc changed_rows "name" "always"))
				(if (> (get_assoc NEW "value") 100)
					(set changed_rows (set_assoc changed_rows "value" 100)) nil))
			changed_rows))`),
		Language: "sql",
	}
	compileTriggerForUse("test", "items", &trigger)
	if trigger.Func.Proc() == nil || trigger.Func.Proc().Compiled == nil {
		t.Fatal("lazy trigger procedure has no native entry point")
	}
	newRow := scm.NewFastDictValue(2)
	newRow.Set(scm.NewString("name"), scm.NewString("original"), nil)
	newRow.Set(scm.NewString("value"), scm.NewInt(42), nil)
	got := scm.Apply(trigger.Func, scm.NewNil(), scm.NewFastDict(newRow), scm.NewNil(), scm.NewNil())
	if name := scm.Apply(got, scm.NewString("name")); !scm.Equal(name, scm.NewString("always")) {
		t.Fatalf("compiled false branch lost preceding mutation: got %s", scm.String(got))
	}
}

func TestInternalTriggerUsesJIT(t *testing.T) {
	trigger := TriggerDescription{
		Name:     "internal",
		IsSystem: true,
		Func: buildFKProc(scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("+"), scm.NewInt(40), scm.NewInt(2),
		})),
		Hidden: true,
	}
	finalizeTriggerCompilation(&trigger)
	requireCompiledTrigger(t, trigger)
}

func TestHiddenPhysicalTriggerKeepsCallbackCompilationBoundary(t *testing.T) {
	trigger := TriggerDescription{
		Name:   "generated-physical-plan",
		Func:   scm.Eval(triggerJITTestPlan(), &scm.Globalenv),
		Hidden: true,
	}
	finalizeTriggerCompilation(&trigger)
	if trigger.Func.Proc() == nil {
		t.Fatalf("hidden trigger function is not a procedure: %s", scm.String(trigger.Func))
	}
	if trigger.Func.Proc().Compiled != nil || trigger.Func.Proc().JITCode != 0 {
		t.Fatal("hidden physical trigger unexpectedly compiled its outer orchestration")
	}
	if got := scm.Apply(trigger.Func, scm.NewNil(), scm.NewNil(), scm.NewNil(), scm.NewNil()); !scm.Equal(got, scm.NewInt(42)) {
		t.Fatalf("hidden trigger fallback returned %s, want 42", scm.String(got))
	}
}
