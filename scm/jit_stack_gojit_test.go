//go:build goexperiment.jit && amd64

/*
Copyright (C) 2026  MemCP Contributors

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
	"testing"
	"time"
)

func TestJITEntryGrowsStackInPrologue(t *testing.T) {
	value := NewSymbol("value")
	proc := NewProcStruct(Proc{
		Params:  NewSlice([]Scmer{value}),
		Body:    NewNthLocalVar(0),
		En:      &Globalenv,
		NumVars: 4096,
	})
	compiled := jitCompile(proc)
	if !compiled.IsProc() || compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("large-frame procedure was not JIT compiled")
	}

	done := make(chan Scmer, 1)
	go func() {
		done <- Apply(compiled, NewInt(42))
	}()
	select {
	case result := <-done:
		if !result.IsInt() || result.Int() != 42 {
			t.Fatalf("unexpected result after JIT stack growth: %s", result.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("JIT stack growth did not resume the generated procedure")
	}
}
