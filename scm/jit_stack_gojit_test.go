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
	"bytes"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

var jitFrameClearInstruction = []byte{0xf3, 0x48, 0xab}

func jitEntryCode(entry *JITEntryPoint) []byte {
	return unsafe.Slice((*byte)(entry.CodePtr), entry.CodeLen)
}

func TestJITFrameClearingLimitedToParserControlFlow(t *testing.T) {
	t.Run("query projection", func(t *testing.T) {
		compiled := compileJITExpressionTestProc(t, `(lambda (a b) (list "a" a "b" b))`)
		if bytes.Contains(jitEntryCode(compiled.Proc().Compiled), jitFrameClearInstruction) {
			t.Fatal("ordinary expression clears its entire JIT frame")
		}
	})

	t.Run("parser", func(t *testing.T) {
		compiled := compileJITExpressionTestProc(t, `(lambda (input) (begin
			(define word (parser (regex "[a-z]+" false false)))
			((parser '((define value word) $) value "") input)))`)
		if !bytes.Contains(jitEntryCode(compiled.Proc().Compiled), jitFrameClearInstruction) {
			t.Fatal("parser expression does not clear shared JIT frame targets")
		}
	})
}

func TestJITScalarPointerSpillRemainsInStackMap(t *testing.T) {
	code := make([]byte, 128)
	start := unsafe.Pointer(&code[0])
	ctx := &JITContext{
		Start:      start,
		Ptr:        start,
		End:        unsafe.Add(start, len(code)),
		AllRegs:    1 << uint(RegRAX),
		FrameReg:   RegRBP,
		StackReg:   RegRSP,
		ScratchReg: RegR11,
	}
	value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: RegRAX, RelocatablePointer: true}
	ctx.BindReg(RegRAX, &value)

	if got := ctx.AllocReg(); got != RegRAX {
		t.Fatalf("spilled register %d, want %d", got, RegRAX)
	}
	root := jitStackRoot{base: jitStackRootFrameBP, offset: -8}
	if _, ok := ctx.StackRoots[root]; !ok {
		t.Fatal("relocatable scalar spill is missing from the stack map")
	}
}

func TestJITEntryGrowsStackInPrologue(t *testing.T) {
	value := NewSymbol("value")
	gcEcho := NewFunc(func(args ...Scmer) Scmer {
		runtime.GC()
		return args[0]
	})
	proc := NewProcStruct(Proc{
		Params:  NewSlice([]Scmer{value}),
		Body:    NewSlice([]Scmer{gcEcho, NewNthLocalVar(0)}),
		En:      &Globalenv,
		NumVars: 4096,
	})
	compiled := jitCompile(proc)
	if !compiled.IsProc() || compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("large-frame procedure was not JIT compiled")
	}

	done := make(chan Scmer, 1)
	want := "pointer-bearing input survives stack relocation"
	go func() {
		done <- Apply(compiled, NewString(want))
	}()
	select {
	case result := <-done:
		if !result.IsString() || result.String() != want {
			t.Fatalf("unexpected result after JIT stack growth: %s", result.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("JIT stack growth did not resume the generated procedure")
	}
}
