//go:build !goexperiment.jit

/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package scm

import (
	"reflect"
	"sync/atomic"
	"unsafe"
)

// Stock Go never executes JIT code. These definitions keep compilation and
// interpreter builds independent of the experimental standard-library package.
type jitBufferFull struct{ Offset, Required, Available int }
type jitRuntimeFrameFixup struct{ position unsafe.Pointer }
type jitRuntimeStackFixup struct{}

func jitRememberGoFunction(uint64, any)                  {}
func jitABIIntRegs() []Reg                               { return jitGoABIIntRegs() }
func jitRuntimeFunction[F any](context unsafe.Pointer) F { return *(*F)(unsafe.Pointer(&context)) }
func jitRuntimeFunctionContext[F any](fn F) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&fn))
}
func jitRuntimeLayoutGoCallArgs(_ uint64, words []goCallArgWord) ([]goCallArgLocation, int) {
	return layoutGoCallWords(words)
}
func jitRuntimeCallBytes(uint64) int32               { return int32(jitGoSpillBytes + 16) }
func jitRuntimeResultRegs(_ uint64, count int) []Reg { return GoABIIntRegs[:count] }

func jitRuntimeGoResultShape(uint64) ([]uint8, []uint8) {
	panic("jit: result layout requires the JIT toolchain")
}

func jitEmitRuntimeFrame(ctx *JITContext) jitRuntimeFrameFixup {
	if jitPortableBackend {
		jitArchEmitNative(ctx, jitNativeProlog, 0, 0, 0, 0)
	} else {
		ctx.emitByte(0x55)
		ctx.emitBytes(0x48, 0x89, 0xe5)
	}
	return jitRuntimeFrameFixup{ctx.EmitSubRSP32Fixup()}
}
func jitPatchRuntimeFrame(ctx *JITContext, f jitRuntimeFrameFixup, size int32) {
	ctx.PatchInt32(f.position, size)
}
func jitEmitRuntimeReturn(ctx *JITContext) {
	if jitPortableBackend {
		jitArchEmitNative(ctx, jitNativeEpilog, 0, 0, 0, 0)
	} else {
		ctx.emitBytes(0xc9, 0xc3)
	}
}
func jitEmitRuntimeCall(ctx *JITContext, addr uint64, target Reg) {
	if addr != 0 {
		target = RegR12
		ctx.EmitMovRegImm64(target, addr)
	}
	if jitPortableBackend {
		jitArchEmitNative(ctx, jitNativeCall, target, 0, 0, 0)
	} else {
		if target >= 8 {
			ctx.emitByte(0x41)
		}
		ctx.emitBytes(0xff, 0xd0|byte(target&7))
	}
}
func jitEmitRuntimeStackCheck(*JITContext) jitRuntimeStackFixup                    { return jitRuntimeStackFixup{} }
func jitPatchRuntimeStackCheck(*JITContext, jitRuntimeStackFixup, int32, JITLabel) {}
func jitEmitRuntimeStackGrow(*JITContext, reflect.Type, JITLabel)                  {}
func jitRuntimeSignature[F any]() reflect.Type                                     { return reflect.TypeFor[F]() }

type jitFunctionHeader struct{ code uintptr }

func (h *jitFunctionHeader) SetEntry(pc uintptr)     { atomic.StoreUintptr(&h.code, pc) }
func (h *jitFunctionHeader) Entry() uintptr          { return atomic.LoadUintptr(&h.code) }
func jitHeaderForEntry(pc uintptr) jitFunctionHeader { return jitFunctionHeader{code: pc} }

func jitEmitRuntimeClosureCall(ctx *JITContext, context Reg) {
	ctx.EmitMovRegMem(RegR11, context, 0)
	jitEmitRuntimeCall(ctx, 0, RegR11)
}

func jitEmitRuntimeLeave(ctx *JITContext) {
	if jitPortableBackend {
		jitArchEmitNative(ctx, jitNativeLeave, 0, 0, 0, 0)
	} else {
		ctx.emitByte(0xc9)
	}
}

func jitRuntimeFreeGPRegs() uint64 { return jitArchFreeGPRegs() }

func jitRuntimeStoreScmer(address *Scmer, value Scmer) { *address = value }
func jitRuntimeClearScmer(address *Scmer)              { *address = Scmer{} }
func jitEmitRuntimeLoadFunctionContext(ctx *JITContext, target, slot Reg) {
	ctx.EmitMovRegMem(target, slot, 0)
}
func jitEmitPointerFreeStoreBarrier(*JITContext, Reg, func()) {}

func jitRuntimeScmerClosureABI() jitClosureRegisters { return jitLegacyClosureRegisters() }
