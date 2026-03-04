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
	// Free registers: exclude RAX (arg/return), RBX (return aux), RSP, RBP,
	// R11 (scratch for emit helpers), R14 (Go "g" pointer)
	freeRegs := uint64((1 << uint(scm.RegRAX)) | (1 << uint(scm.RegRBX)) |
		(1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) |
		(1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) |
		(1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) |
		(1 << uint(scm.RegR12)) | (1 << uint(scm.RegR13)) | (1 << uint(scm.RegR15)))
	ctx := &scm.JITContext{
		Ptr:      unsafe.Pointer(&codeBuf[0]),
		Start:    unsafe.Pointer(&codeBuf[0]),
		End:      unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
		FreeRegs: freeRegs,
		AllRegs:  freeRegs,
	}
	// Keep R14 untouched because Go calls inside fallback emitters require
	// it to hold the runtime g-pointer. No loop-reserved registers needed here.

	// Entry: Go ABI — RAX = int64 index argument
	idxReg := ctx.AllocReg()
	ctx.EmitMovRegReg(idxReg, scm.RegRAX)
	idx := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxReg}

	var thisptr scm.JITValueDesc
	if constThisptr {
		thisptr = scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(extractDataPtr(s))}
	} else {
		ptrReg := ctx.AllocReg()
		ctx.EmitMovRegImm64(ptrReg, uint64(extractDataPtr(s)))
		thisptr = scm.JITValueDesc{Loc: scm.LocReg, Reg: ptrReg}
	}

	// Use recover to handle register exhaustion gracefully
	var emitErr interface{}
	func() {
		defer func() { emitErr = recover() }()
		result := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: scm.RegRAX, Reg2: scm.RegRBX}
		desc := s.JITEmit(ctx, thisptr, idx, result)
		ctx.EmitMovPairToResult(&desc, &result)
		ctx.EmitByte(0xC3) // RET
		ctx.ResolveFixupsFinal()
	}()
	if emitErr != nil {
		tb.Skipf("JIT emit failed: %v", emitErr)
		return nil, func() {}
	}

	codeLen := int(uintptr(ctx.Ptr) - uintptr(ctx.Start))
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
	freeRegs := uint64((1 << uint(scm.RegRAX)) | (1 << uint(scm.RegRBX)) |
		(1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) |
		(1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) |
		(1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) |
		(1 << uint(scm.RegR12)) | (1 << uint(scm.RegR13)) | (1 << uint(scm.RegR15)))
	ctx := &scm.JITContext{
		Ptr:      unsafe.Pointer(&codeBuf[0]),
		Start:    unsafe.Pointer(&codeBuf[0]),
		End:      unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
		FreeRegs: freeRegs,
		AllRegs:  freeRegs,
	}

	var thisptr scm.JITValueDesc
	if constThisptr {
		thisptr = scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(extractDataPtr(s))}
	} else {
		ctx.FreeRegs &^= 1 << uint(scm.RegR13)
		ctx.AllRegs &^= 1 << uint(scm.RegR13)
		ctx.EmitMovRegImm64(scm.RegR13, uint64(extractDataPtr(s)))
		thisptr = scm.JITValueDesc{Loc: scm.LocReg, Reg: scm.RegR13}
	}

	// Loop phis on stack:
	//   [RSP+0] = idx
	//   [RSP+8] = sum
	spFixup := ctx.EmitSubRSP32Fixup()
	ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 0)
	ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 8)

	lblTop := ctx.ReserveLabel()
	lblEnd := ctx.ReserveLabel()
	ctx.MarkLabel(lblTop)

	// if idx >= count: break
	idxCmpReg := ctx.AllocReg()
	ctx.EmitLoadFromStack(idxCmpReg, 0)
	ctx.EmitCmpRegImm32(idxCmpReg, int32(count))
	ctx.FreeReg(idxCmpReg)
	ctx.EmitJcc(scm.CcGE, lblEnd)

	// Load loop index again for the body, so emitter-side register reuse does
	// not affect the loop-carried value in the phi slot.
	idxArgReg := ctx.AllocReg()
	ctx.EmitLoadFromStack(idxArgReg, 0)
	idx := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxArgReg}

	var emitErr interface{}
	var desc scm.JITValueDesc
	func() {
		defer func() { emitErr = recover() }()
		// Allow typed emitters to keep unboxed results in a single register.
		result := scm.JITValueDesc{Loc: scm.LocAny}
		desc = s.JITEmit(ctx, thisptr, idx, result)
	}()
	if emitErr != nil {
		tb.Skipf("JIT emit failed: %v", emitErr)
		return nil, func() {}
	}

	idxAlias := false
	switch desc.Loc {
	case scm.LocReg:
		idxAlias = desc.Reg == idxArgReg
	case scm.LocRegPair:
		idxAlias = desc.Reg == idxArgReg || desc.Reg2 == idxArgReg
	}
	if !idxAlias {
		ctx.FreeReg(idxArgReg)
	}

	// sum += Int(desc) with Scmer semantics (nil->0, float->int(float), int->int)
	sumReg := ctx.AllocReg()
	ctx.EmitLoadFromStack(sumReg, 8)
	switch desc.Loc {
	case scm.LocImm:
		ctx.EmitMovRegImm64(scm.RegR11, uint64(desc.Imm.Int()))
		ctx.EmitAddInt64(sumReg, scm.RegR11)
	case scm.LocReg:
		if desc.Type != scm.TagInt {
			panic("jit sum: unsupported LocReg non-int result")
		}
		ctx.EmitAddInt64(sumReg, desc.Reg)
	case scm.LocRegPair:
		switch desc.Type {
		case scm.TagInt:
			ctx.EmitAddInt64(sumReg, desc.Reg2)
		case scm.TagFloat:
			floatIntReg := ctx.AllocReg()
			ctx.EmitCvtFloatBitsToInt64(floatIntReg, desc.Reg2)
			ctx.EmitAddInt64(sumReg, floatIntReg)
			ctx.FreeReg(floatIntReg)
		case scm.TagNil:
			// nil contributes 0
		default:
			tagPtrReg := ctx.AllocReg()
			tagAuxReg := ctx.AllocReg()
			ctx.EmitMovRegReg(tagPtrReg, desc.Reg)
			ctx.EmitMovRegReg(tagAuxReg, desc.Reg2)
			tagSrc := scm.JITValueDesc{Loc: scm.LocRegPair, Type: desc.Type, Reg: tagPtrReg, Reg2: tagAuxReg}
			tagDesc := ctx.EmitGetTagDesc(&tagSrc, scm.JITValueDesc{Loc: scm.LocAny})
			switch tagDesc.Loc {
			case scm.LocImm:
				switch tagDesc.Imm.Int() {
				case int64(scm.TagInt):
					ctx.EmitAddInt64(sumReg, desc.Reg2)
				case int64(scm.TagFloat):
					floatIntReg := ctx.AllocReg()
					ctx.EmitCvtFloatBitsToInt64(floatIntReg, desc.Reg2)
					ctx.EmitAddInt64(sumReg, floatIntReg)
					ctx.FreeReg(floatIntReg)
				}
			case scm.LocReg:
				lblAddInt := ctx.ReserveLabel()
				lblAddFloat := ctx.ReserveLabel()
				lblDone := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(tagDesc.Reg, int32(scm.TagInt))
				ctx.EmitJcc(scm.CcE, lblAddInt)
				ctx.EmitCmpRegImm32(tagDesc.Reg, int32(scm.TagFloat))
				ctx.EmitJcc(scm.CcE, lblAddFloat)
				ctx.EmitJmp(lblDone)
				ctx.MarkLabel(lblAddInt)
				ctx.EmitAddInt64(sumReg, desc.Reg2)
				ctx.EmitJmp(lblDone)
				ctx.MarkLabel(lblAddFloat)
				floatIntReg := ctx.AllocReg()
				ctx.EmitCvtFloatBitsToInt64(floatIntReg, desc.Reg2)
				ctx.EmitAddInt64(sumReg, floatIntReg)
				ctx.FreeReg(floatIntReg)
				ctx.MarkLabel(lblDone)
				ctx.FreeReg(tagDesc.Reg)
			default:
				panic("jit sum: unsupported tag descriptor location")
			}
		}
	default:
		panic("jit sum: unsupported result descriptor location")
	}
	ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocReg, Reg: sumReg}, 8)
	ctx.FreeReg(sumReg)
	ctx.FreeDesc(&desc)

	// idx++
	idxReg := ctx.AllocReg()
	ctx.EmitLoadFromStack(idxReg, 0)
	ctx.EmitMovRegImm64(scm.RegR11, 1)
	ctx.EmitAddInt64(idxReg, scm.RegR11)
	ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocReg, Reg: idxReg}, 0)
	ctx.FreeReg(idxReg)

	ctx.EmitJmp(lblTop)
	ctx.MarkLabel(lblEnd)

	// return sum
	retReg := ctx.AllocReg()
	ctx.EmitLoadFromStack(retReg, 8)
	if retReg != scm.RegRAX {
		ctx.EmitMovRegReg(scm.RegRAX, retReg)
	}
	ctx.FreeReg(retReg)

	ctx.PatchInt32(spFixup, 16)
	ctx.EmitAddRSP32(16)
	// RET
	ctx.EmitByte(0xC3)

	ctx.ResolveFixupsFinal()

	codeLen := int(uintptr(ctx.Ptr) - uintptr(ctx.Start))
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
