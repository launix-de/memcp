/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/
package scm

import "bytes"
import "strings"
import "testing"

func TestProcedureWireSnapshotsCapturesAndRunsWithoutSourceEnvironment(t *testing.T) {
	for _, optimize := range []bool{false, true} {
		for _, source := range []string{
			`(lambda (row) (+ row captured))`,
			`(lambda (row) (+ row (outer 1 captured)))`,
			`(lambda (row) (lambda (other) (+ row other)))`,
		} {
			env := &Env{Vars: Vars{Symbol("captured"): NewInt(7)}, Outer: &Globalenv}
			value := Read(t.Name(), source)
			if optimize {
				value = Optimize(value, env, nil)
			}
			original := Eval(value, env)
			data, err := MarshalProcedure(original)
			if err != nil {
				t.Fatalf("optimize=%v source=%s: %v", optimize, source, err)
			}
			// Mutation of the sender's frame after encoding must have no effect.
			env.Vars[Symbol("captured")] = NewInt(900)
			restored, err := UnmarshalProcedure(data)
			if err != nil {
				t.Fatal(err)
			}
			if restored.Proc().En != &Globalenv || restored.Proc().nativeCode() != 0 {
				t.Fatal("receiver retained source environment or executable code")
			}
			got := Apply(restored, NewInt(3))
			if got.IsProc() {
				got = Apply(got, NewInt(7))
			}
			if got.Int() != 10 {
				t.Fatalf("optimize=%v source=%s: got %v", optimize, source, got)
			}
		}
	}
}

func TestProcedureWireRejectsRuntimeCapturesAndUnclosedNestedBindings(t *testing.T) {
	for name, captured := range map[string]Scmer{
		"native closure": NewFunc(func(args ...Scmer) Scmer { return args[0] }),
		"session":        NewSession(),
		"Go pointer":     NewAny(new(int)),
	} {
		t.Run(name, func(t *testing.T) {
			env := &Env{Vars: Vars{Symbol("captured"): captured}, Outer: &Globalenv}
			value := Eval(Read(t.Name(), `(lambda () captured)`), env)
			if _, err := MarshalProcedure(value); err == nil {
				t.Fatal("process-local capture was accepted")
			}
		})
	}
	// CloseProcedure intentionally leaves nested lambda scopes intact. A
	// source-only wire encoding must reject a remaining external binding.
	env := &Env{Vars: Vars{Symbol("wire_nested_capture"): NewInt(7)}, Outer: &Globalenv}
	value := Eval(Read(t.Name(), `(lambda (row) (lambda () (+ row wire_nested_capture)))`), env)
	if _, err := MarshalProcedure(value); err == nil || !strings.Contains(err.Error(), "unclosed lexical capture") {
		t.Fatalf("unclosed nested capture: %v", err)
	}
}

func TestProcedureWireRejectsInvalidEnvelopesAndBounds(t *testing.T) {
	for _, data := range [][]byte{
		[]byte(`{"version":2,"code":null}`), []byte(`{"version":1,"code":17}`),
		[]byte(`{"version":1`), bytes.Repeat([]byte(" "), maxProcedureWireBytes+1),
	} {
		if _, err := UnmarshalProcedure(data); err == nil {
			t.Fatal("invalid envelope accepted")
		}
	}
	env := &Env{Vars: Vars{Symbol("large"): NewString(strings.Repeat("x", maxProcedureWireBytes))}, Outer: &Globalenv}
	if _, err := MarshalProcedure(Eval(Read(t.Name(), `(lambda () large)`), env)); err == nil {
		t.Fatal("oversized captured payload accepted")
	}
}

func TestProcedureWireJITCaptures(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (captured) (lambda (row) (+ row captured)))`)
	bound := Apply(compiled, NewInt(7))
	data, err := MarshalProcedure(bound)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := UnmarshalProcedure(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := Apply(restored, NewInt(3)); got.Int() != 10 {
		t.Fatalf("JIT capture roundtrip returned %v", got)
	}
	if restored.Proc().nativeCode() != 0 || restored.Proc().Compiled != nil {
		t.Fatal("wire retained process-local JIT metadata")
	}
}
