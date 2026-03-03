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
package scm

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

func jitExecInt64Unary(tb testing.TB, code []byte) func(int64) int64 {
	tb.Helper()

	pageSize := syscall.Getpagesize()
	n := (len(code) + pageSize - 1) &^ (pageSize - 1)
	mem, err := syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		tb.Fatalf("mmap failed: %v", err)
	}
	copy(mem, code)
	if err := syscall.Mprotect(mem, syscall.PROT_READ|syscall.PROT_EXEC); err != nil {
		_ = syscall.Munmap(mem)
		tb.Fatalf("mprotect failed: %v", err)
	}
	tb.Cleanup(func() { _ = syscall.Munmap(mem) })

	type fnHeader struct{ fnptr *byte }
	h := &fnHeader{fnptr: &mem[0]}
	hp := unsafe.Pointer(h)
	fn := *(*func(int64) int64)(unsafe.Pointer(&hp))
	runtime.KeepAlive(h)
	return fn
}

func emitFibIterativeJIT(tb testing.TB) []byte {
	tb.Helper()

	codeBuf := make([]byte, 512)
	w := &JITWriter{
		Ptr:   unsafe.Pointer(&codeBuf[0]),
		Start: unsafe.Pointer(&codeBuf[0]),
		End:   unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-16),
	}

	// Input n in RAX, output in RAX.
	// if n <= 1 return n
	doneInput := w.ReserveLabel()
	doneFib := w.ReserveLabel()
	loop := w.ReserveLabel()

	w.EmitCmpRegImm32(RegRAX, 1)
	w.EmitJcc(CcLE, doneInput)

	// a=0, b=1, i=2
	w.EmitMovRegImm64(RegRBX, 0)
	w.EmitMovRegImm64(RegRCX, 1)
	w.EmitMovRegImm64(RegRDX, 2)

	w.MarkLabel(loop)
	// if i > n: return b
	w.EmitCmpInt64(RegRDX, RegRAX)
	w.EmitJcc(CcG, doneFib)

	// t = a + b; a = b; b = t; i++
	w.EmitMovRegReg(RegR8, RegRBX)
	w.EmitAddInt64(RegR8, RegRCX)
	w.EmitMovRegReg(RegRBX, RegRCX)
	w.EmitMovRegReg(RegRCX, RegR8)
	w.EmitAddRegImm32(RegRDX, 1)
	w.EmitJmp(loop)

	w.MarkLabel(doneFib)
	w.EmitMovRegReg(RegRAX, RegRCX)
	w.EmitByte(0xC3)

	w.MarkLabel(doneInput)
	w.EmitByte(0xC3)

	w.ResolveFixupsFinal()
	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
	return codeBuf[:codeLen]
}

func fibGo(n int64) int64 {
	if n <= 1 {
		return n
	}
	a, b := int64(0), int64(1)
	for i := int64(2); i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

func TestJITHandwrittenFibonacciIterative(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	code := emitFibIterativeJIT(t)
	if len(code) == 0 {
		t.Fatal("empty code")
	}
	jitFib := jitExecInt64Unary(t, code)

	for n := int64(0); n <= 40; n++ {
		got := jitFib(n)
		want := fibGo(n)
		if got != want {
			t.Fatalf("fib(%d): got %d, want %d", n, got, want)
		}
	}
}

