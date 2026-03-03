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

func jitExecPtrUnary(tb testing.TB, code []byte) func(uintptr) {
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
	fn := *(*func(uintptr))(unsafe.Pointer(&hp))
	runtime.KeepAlive(h)
	return fn
}

// emitFibInlineFromN emits a straight-line inlined Fibonacci implementation.
// Input: n in nReg, Output: fib(n) in nReg.
func emitFibInlineFromN(w *JITWriter, nReg Reg) {
	doneInput := w.ReserveLabel()
	doneFib := w.ReserveLabel()
	loop := w.ReserveLabel()

	w.EmitCmpRegImm32(nReg, 1)
	w.EmitJcc(CcLE, doneInput)

	// a=0, b=1, j=2
	w.EmitMovRegImm64(RegRDX, 0)
	w.EmitMovRegImm64(RegR8, 1)
	w.EmitMovRegImm64(RegR9, 2)

	w.MarkLabel(loop)
	w.EmitCmpInt64(RegR9, nReg)
	w.EmitJcc(CcG, doneFib)
	w.EmitMovRegReg(RegR10, RegRDX)
	w.EmitAddInt64(RegR10, RegR8)
	w.EmitMovRegReg(RegRDX, RegR8)
	w.EmitMovRegReg(RegR8, RegR10)
	w.EmitAddRegImm32(RegR9, 1)
	w.EmitJmp(loop)

	w.MarkLabel(doneFib)
	w.EmitMovRegReg(nReg, RegR8)
	w.MarkLabel(doneInput)
}

func emitFibArrayFillJIT(tb testing.TB) []byte {
	tb.Helper()

	codeBuf := make([]byte, 1024)
	w := &JITWriter{
		Ptr:   unsafe.Pointer(&codeBuf[0]),
		Start: unsafe.Pointer(&codeBuf[0]),
		End:   unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-16),
	}

	// Signature: func(base uintptr), base in RAX.
	// Outer phi-like loop over i in RBX: for i=0..9 { out[i] = float64(fib(i)) }
	w.EmitMovRegImm64(RegRBX, 0) // i = 0
	outerLoop := w.ReserveLabel()
	done := w.ReserveLabel()
	w.MarkLabel(outerLoop)
	w.EmitCmpRegImm32(RegRBX, 10)
	w.EmitJcc(CcGE, done)

	// "Inline call" Fibonacci: n=i in RCX, result also in RCX.
	w.EmitMovRegReg(RegRCX, RegRBX)
	emitFibInlineFromN(w, RegRCX)

	// Convert int64 fib result to float64 bits in RCX.
	w.EmitCvtInt64ToFloat64(RegX1, RegRCX)

	// Address calc: addr = base + i*8 in RDX; store float bits.
	w.EmitMovRegReg(RegRDX, RegRBX)
	w.EmitShlRegImm8(RegRDX, 3)
	w.EmitAddInt64(RegRDX, RegRAX)
	w.EmitStoreRegMem(RegRCX, RegRDX, 0)

	w.EmitAddRegImm32(RegRBX, 1)
	w.EmitJmp(outerLoop)

	w.MarkLabel(done)
	w.EmitByte(0xC3)
	w.ResolveFixupsFinal()
	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
	return codeBuf[:codeLen]
}

func TestJITHandwrittenFibOuterLoopInlineStoresFloatArray(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	code := emitFibArrayFillJIT(t)
	fill := jitExecPtrUnary(t, code)

	var out [10]float64
	fill(uintptr(unsafe.Pointer(&out[0])))

	for i := 0; i < len(out); i++ {
		want := float64(fibGo(int64(i)))
		if out[i] != want {
			t.Fatalf("out[%d]=%v want %v", i, out[i], want)
		}
	}
}
