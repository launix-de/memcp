//go:build goexperiment.jit && (arm64 || riscv64)

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/

package scm

import (
	"runtime"
	"testing"
)

func TestJITPortableLeafParametersAndLiteralsExecuteNatively(t *testing.T) {
	identity := NewProc(&Proc{
		Params: NewSlice([]Scmer{NewSymbol("value")}),
		Body:   NewNthLocalVar(0),
		En:     &Globalenv,
	})
	compiledIdentity := CompileJIT(identity, true)
	if proc := compiledIdentity.Proc(); proc == nil || proc.JITCode == 0 {
		t.Fatal("identity procedure did not compile natively")
	}
	if got := Apply(compiledIdentity, NewInt(37)); !Equal(got, NewInt(37)) {
		t.Fatalf("native identity returned %s", String(got))
	}

	constant := NewProc(&Proc{Params: NewSlice(nil), Body: NewInt(42), En: &Globalenv})
	compiledConstant := CompileJIT(constant, true)
	if proc := compiledConstant.Proc(); proc == nil || proc.JITCode == 0 {
		t.Fatal("constant procedure did not compile natively")
	}
	if got := Apply(compiledConstant); !Equal(got, NewInt(42)) {
		t.Fatalf("native constant returned %s", String(got))
	}

	stringConstant := NewProc(&Proc{Params: NewSlice(nil), Body: NewString("portable-root"), En: &Globalenv})
	compiledString := CompileJIT(stringConstant, true)
	if proc := compiledString.Proc(); proc == nil || proc.JITCode == 0 {
		t.Fatal("string constant procedure did not compile natively")
	}
	runtime.GC()
	if got := Apply(compiledString); !Equal(got, NewString("portable-root")) {
		t.Fatalf("native rooted string constant returned %s", String(got))
	}

	addition := NewProc(&Proc{
		Params: NewSlice([]Scmer{NewSymbol("left"), NewSymbol("right")}),
		Body: NewSlice([]Scmer{
			NewSymbol("+"), NewSymbol("left"), NewSymbol("right"),
		}),
		En: &Globalenv,
	})
	compiledAddition := CompileJIT(addition, true)
	if proc := compiledAddition.Proc(); proc == nil || proc.JITCode != 0 {
		t.Fatal("unsupported addition must remain an interpreted procedure")
	}
	if got := Apply(compiledAddition, NewInt(2), NewInt(3)); !Equal(got, NewInt(5)) {
		t.Fatalf("fallback addition returned %s", String(got))
	}
}
