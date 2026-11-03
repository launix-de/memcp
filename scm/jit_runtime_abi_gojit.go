//go:build goexperiment.jit

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
	"runtime/jit"
	"sync"
	"unsafe"
)

type jitFunctionHeader = jit.FunctionHeader

type jitBufferFull = jit.CodeBufferFull
type jitRuntimeFrameFixup = jit.FrameFixup
type jitRuntimeStackFixup = jit.StackCheckFixup

var jitGoConventions sync.Map // map[uint64]jit.CallingConvention

func jitRememberGoFunction(address uint64, fn any) {
	if !jit.SupportedEmitters() {
		return
	}
	if _, exists := jitGoConventions.Load(address); !exists {
		jitGoConventions.LoadOrStore(address, jit.GetCallingConventionForFunc(fn))
	}
}

func jitABIIntRegs() []Reg {
	if !jit.SupportedEmitters() {
		return jitGoABIIntRegs()
	}
	r := jit.RegisterConfig()
	regs := make([]Reg, len(r.IntegerArguments))
	for i, reg := range r.IntegerArguments {
		regs[i] = Reg(reg.Number)
	}
	return regs
}

func jitRuntimeFunction[F any](context unsafe.Pointer) F {
	return jit.Function[F]((*jit.FunctionHeader)(context))
}

func jitRuntimeFunctionContext[F any](fn F) unsafe.Pointer { return jit.FunctionContext(fn) }

func jitRuntimeLayoutGoCallArgs(address uint64, words []goCallArgWord) ([]goCallArgLocation, int) {
	value, exists := jitGoConventions.Load(address)
	if !exists {
		panic("jit: Go helper has no registered function signature")
	}
	c := value.(jit.CallingConvention)
	locations := make([]goCallArgLocation, 0, len(words))
	for _, input := range c.Inputs {
		for _, part := range input.Parts {
			if part.Location.Kind == jit.InRegister {
				if part.Location.Register.Class != jit.IntegerRegister {
					panic("jit: Go helper requires FP argument lowering")
				}
				locations = append(locations, goCallArgLocation{size: part.Size, inReg: true, reg: Reg(part.Location.Register.Number)})
			} else {
				locations = append(locations, goCallArgLocation{size: part.Size, stackOff: int32(part.Location.StackOffset - c.Registers.StackArgumentOffset)})
			}
		}
	}
	if len(locations) != len(words) {
		panic("jit: Go helper arguments do not match its function signature")
	}
	if c.CallFrameSize > jitGoSpillBytes {
		panic("jit: Go helper exceeds reserved call area")
	}
	return locations, int(c.StackArgumentsSize / 8)
}

func jitEmitRuntimeFrame(ctx *JITContext) jitRuntimeFrameFixup {
	ctx.FlushRegisterMoves()
	return jit.EmitFrame(ctx, jit.FramePlan{Frame: jit.PhysicalRegister{Class: jit.IntegerRegister, Number: uint8(ctx.FrameReg)}})
}

func jitPatchRuntimeFrame(ctx *JITContext, fixup jitRuntimeFrameFixup, size int32) {
	jit.PatchFrame(ctx, fixup, size)
}

func jitEmitRuntimeReturn(ctx *JITContext) {
	ctx.FlushRegisterMoves()
	jit.EmitReturn(ctx, jit.PhysicalRegister{Class: jit.IntegerRegister, Number: uint8(ctx.FrameReg)})
}

func jitEmitRuntimeCall(ctx *JITContext, address uint64, target Reg) {
	ctx.FlushRegisterMoves()
	p := jit.CallPlan{Entry: uintptr(address)}
	if address == 0 {
		p.Indirect = true
		p.Target = jit.PhysicalRegister{Class: jit.IntegerRegister, Number: uint8(target)}
	}
	jit.EmitCall(ctx, p)
}

func jitEmitRuntimeStackCheck(ctx *JITContext) jitRuntimeStackFixup {
	ctx.FlushRegisterMoves()
	return jit.EmitStackCheck(ctx, 0, 0)
}

func jitPatchRuntimeStackCheck(ctx *JITContext, f jitRuntimeStackFixup, size int32, grow JITLabel) {
	jit.PatchStackCheck(ctx, f, uintptr(size), int(ctx.Labels[grow]))
}

// Growth metadata uses the same regular frame recipe as every other safepoint;
// it is already final and must not be reinterpreted as the old caller-spill map.
func jitEmitRuntimeStackGrow(ctx *JITContext, signature reflect.Type, retry JITLabel) {
	ctx.FlushRegisterMoves()
	m := jit.EmitStackGrow(ctx, jit.GetCallingConventionForType(signature), jit.PhysicalRegister{Class: jit.IntegerRegister, Number: uint8(ctx.FrameReg)}, int(ctx.Labels[retry]))
	ctx.Safepoints = append(ctx.Safepoints, jitSafepoint{pcOffset: int32(m.PCOffset), entry: true, entryFrameWords: m.FrameWords, entryPointerMap: m.PointerMask, regularEntry: true})
}

func jitRuntimeSignature[F any]() reflect.Type { return reflect.TypeFor[F]() }

var _ jit.CodeBuffer = (*JITContext)(nil)

func jitHeaderForEntry(pc uintptr) jitFunctionHeader {
	var h jitFunctionHeader
	h.SetEntry(pc)
	return h
}

func jitEmitRuntimeClosureCall(ctx *JITContext, context Reg) {
	ctx.FlushRegisterMoves()
	reg := jit.PhysicalRegister{Class: jit.IntegerRegister, Number: uint8(context)}
	jit.EmitCall(ctx, jit.CallPlan{Closure: &reg})
}

func jitEmitRuntimeLeave(ctx *JITContext) {
	ctx.FlushRegisterMoves()
	jit.EmitLeave(ctx, jit.PhysicalRegister{Class: jit.IntegerRegister, Number: uint8(ctx.FrameReg)})
}

func jitRuntimeFreeGPRegs() uint64 {
	if !jit.SupportedEmitters() {
		return jitArchFreeGPRegs()
	}
	return jitArchFreeGPRegs() & jit.RegisterConfig().Allocatable[jit.IntegerRegister]
}

// Proc's SSA-visible code word is also its FunctionHeader prefix. Fail the
// build if the runtime expands that prefix instead of silently reinterpreting
// adjacent Scheme fields. Storage funcvals embed FunctionHeader directly.
var _ [unsafe.Sizeof(jit.FunctionHeader{}) - unsafe.Sizeof(Proc{}.JITCode)]byte
var _ [unsafe.Sizeof(Proc{}.JITCode) - unsafe.Sizeof(jit.FunctionHeader{})]byte
var _ [0 - unsafe.Offsetof(Proc{}.JITCode)]byte
