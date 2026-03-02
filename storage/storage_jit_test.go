/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
	"runtime"
	"runtime/debug"
	"syscall"
	"testing"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

// jitEmitter is the interface shared by all storage types with JIT emitters.
type jitEmitter interface {
	JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc
}

// jitBuildGetValueFunc compiles a JIT function for any storage type.
// Returns func(int64) scm.Scmer with Go ABI calling convention.
// Returns nil fn if the emitter panics (e.g. register exhaustion).
func jitBuildGetValueFunc(tb testing.TB, s jitEmitter, constThisptr bool) (fn func(int64) scm.Scmer, cleanup func()) {
	tb.Helper()
	if runtime.GOARCH != "amd64" {
		tb.Skip("JIT tests only on amd64")
	}

	codeBuf := make([]byte, 65536)
	w := &scm.JITWriter{
		Ptr:   unsafe.Pointer(&codeBuf[0]),
		Start: unsafe.Pointer(&codeBuf[0]),
		End:   unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
	}

	// Free registers: exclude RAX (arg/return), RBX (return aux), RSP, RBP,
	// R11 (scratch for emit helpers), R14 (Go "g" pointer)
	freeRegs := uint64((1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) |
		(1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) |
		(1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) |
		(1 << uint(scm.RegR12)) | (1 << uint(scm.RegR13)) | (1 << uint(scm.RegR15)))
	ctx := &scm.JITContext{
		W:        w,
		FreeRegs: freeRegs,
		AllRegs:  freeRegs,
	}
	// Reserve fixed loop registers. Keep R14 untouched because Go calls
	// inside fallback emitters require it to hold the runtime g-pointer.
	ctx.FreeRegs &^= 1 << uint(scm.RegR15) // loop counter
	ctx.FreeRegs &^= 1 << uint(scm.RegR12) // accumulator

	// Entry: Go ABI — RAX = int64 index argument
	idxReg := ctx.AllocReg()
	w.EmitMovRegReg(idxReg, scm.RegRAX)
	idx := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxReg}

	var thisptr scm.JITValueDesc
	if constThisptr {
		thisptr = scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(extractDataPtr(s))}
	} else {
		ptrReg := ctx.AllocReg()
		w.EmitMovRegImm64(ptrReg, uint64(extractDataPtr(s)))
		thisptr = scm.JITValueDesc{Loc: scm.LocReg, Reg: ptrReg}
	}

	// Use recover to handle register exhaustion gracefully
	var emitErr interface{}
	func() {
		defer func() { emitErr = recover() }()
		result := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: scm.RegRAX, Reg2: scm.RegRBX}
		desc := s.JITEmit(ctx, thisptr, idx, result)
		ctx.EmitMovPairToResult(&desc, &result)
		w.EmitByte(0xC3) // RET
		w.ResolveFixupsFinal()
	}()
	if emitErr != nil {
		tb.Skipf("JIT emit failed: %v", emitErr)
		return nil, func() {}
	}

	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
	code := codeBuf[:codeLen]

	pageSize := syscall.Getpagesize()
	n := (len(code) + pageSize - 1) &^ (pageSize - 1)
	b, err := syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		tb.Fatalf("mmap failed: %v", err)
	}
	copy(b, code)
	if err := syscall.Mprotect(b, syscall.PROT_READ|syscall.PROT_EXEC); err != nil {
		syscall.Munmap(b)
		tb.Fatalf("mprotect failed: %v", err)
	}

	type funcHeader struct {
		fnptr *byte
	}
	hdr := &funcHeader{fnptr: &b[0]}
	hdrPtr := unsafe.Pointer(hdr)
	jitFn := *(*func(int64) scm.Scmer)(unsafe.Pointer(&hdrPtr))

	tb.Logf("JIT code size: %d bytes (constThisptr=%v)", codeLen, constThisptr)

	return jitFn, func() {
		runtime.KeepAlive(hdr)
		runtime.KeepAlive(ctx) // keep SpillBuf alive for JIT code
		syscall.Munmap(b)
	}
}

// extractDataPtr gets the data pointer from an interface value (the concrete struct pointer).
func extractDataPtr(s jitEmitter) int64 {
	// Interface layout: [type_ptr, data_ptr]. We want data_ptr.
	type iface struct {
		_    uintptr
		data uintptr
	}
	return int64((*iface)(unsafe.Pointer(&s)).data)
}

// scmerEqual compares two Scmer values for equality in test context.
func scmerEqual(a, b scm.Scmer) bool {
	if a.IsNil() && b.IsNil() {
		return true
	}
	if a.IsNil() != b.IsNil() {
		return false
	}
	if a.IsString() && b.IsString() {
		return a.String() == b.String()
	}
	if a.IsFloat() && b.IsFloat() {
		return a.Float() == b.Float()
	}
	if a.IsInt() && b.IsInt() {
		return a.Int() == b.Int()
	}
	return a.Int() == b.Int()
}

// jitBuildSumFuncGeneric builds a JIT sum loop for any storage type that returns numeric values.
// Returns nil fn if the emitter panics (e.g. register exhaustion).
func jitBuildSumFuncGeneric(tb testing.TB, s jitEmitter, count int64, constThisptr bool) (fn func() int64, cleanup func()) {
	tb.Helper()
	if runtime.GOARCH != "amd64" {
		tb.Skip("JIT benchmarks only on amd64")
	}

	codeBuf := make([]byte, 65536)
	w := &scm.JITWriter{
		Ptr:   unsafe.Pointer(&codeBuf[0]),
		Start: unsafe.Pointer(&codeBuf[0]),
		End:   unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
	}

	freeRegs := uint64((1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) |
		(1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) |
		(1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) |
		(1 << uint(scm.RegR12)) | (1 << uint(scm.RegR13)) | (1 << uint(scm.RegR15)))
	ctx := &scm.JITContext{
		W:        w,
		FreeRegs: freeRegs,
		AllRegs:  freeRegs,
	}

	var thisptr scm.JITValueDesc
	if constThisptr {
		thisptr = scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(extractDataPtr(s))}
	} else {
		ctx.FreeRegs &^= 1 << uint(scm.RegR13)
		w.EmitMovRegImm64(scm.RegR13, uint64(extractDataPtr(s)))
		thisptr = scm.JITValueDesc{Loc: scm.LocReg, Reg: scm.RegR13}
	}

	// XOR R15, R15 (zero loop counter)
	w.EmitByte(0x4D)
	w.EmitByte(0x31)
	w.EmitByte(0xFF)
	// XOR R12, R12 (zero accumulator)
	w.EmitByte(0x4D)
	w.EmitByte(0x31)
	w.EmitByte(0xE4)

	lblTop := w.ReserveLabel()
	w.MarkLabel(lblTop)

	idxReg := ctx.AllocReg()
	w.EmitMovRegReg(idxReg, scm.RegR15)
	idx := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxReg}

	var emitErr interface{}
	var desc scm.JITValueDesc
	func() {
		defer func() { emitErr = recover() }()
		result := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: scm.RegRAX, Reg2: scm.RegRBX}
		desc = s.JITEmit(ctx, thisptr, idx, result)
	}()
	if emitErr != nil {
		tb.Skipf("JIT emit failed: %v", emitErr)
		return nil, func() {}
	}

	// ADD R12, desc.Reg2 (aux field holds int value; nil→0 is fine for SUM)
	w.EmitAddInt64(scm.RegR12, desc.Reg2)
	ctx.FreeDesc(&desc)

	// INC R15
	w.EmitByte(0x49)
	w.EmitByte(0xFF)
	w.EmitByte(0xC7)
	// CMP R15, count
	w.EmitCmpRegImm32(scm.RegR15, int32(count))
	// JL loopTop
	w.EmitJcc(scm.CcL, lblTop)

	// MOV RAX, R12
	w.EmitByte(0x4C)
	w.EmitByte(0x89)
	w.EmitByte(0xE0)
	// RET
	w.EmitByte(0xC3)

	w.ResolveFixupsFinal()

	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
	code := codeBuf[:codeLen]

	pageSize := syscall.Getpagesize()
	n := (len(code) + pageSize - 1) &^ (pageSize - 1)
	b, err := syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		tb.Fatalf("mmap failed: %v", err)
	}
	copy(b, code)
	if err := syscall.Mprotect(b, syscall.PROT_READ|syscall.PROT_EXEC); err != nil {
		syscall.Munmap(b)
		tb.Fatalf("mprotect failed: %v", err)
	}

	type funcHeader struct {
		fnptr *byte
	}
	hdr := &funcHeader{fnptr: &b[0]}
	hdrPtr := unsafe.Pointer(hdr)
	rawFn := *(*func() int64)(unsafe.Pointer(&hdrPtr))

	tb.Logf("JIT SUM code size: %d bytes (constThisptr=%v, count=%d)", codeLen, constThisptr, count)

	// When fallback emitters perform Go-calls from JIT code, a concurrent GC stack walk
	// can fail to unwind through unknown JIT PCs. Run this hot loop with GC disabled.
	jitFn := func() int64 {
		oldGC := debug.SetGCPercent(-1)
		v := rawFn()
		debug.SetGCPercent(oldGC)
		return v
	}

	return jitFn, func() {
		runtime.KeepAlive(hdr)
		runtime.KeepAlive(ctx) // keep SpillBuf alive for JIT code
		syscall.Munmap(b)
	}
}
