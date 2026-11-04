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

// Prepared once per Go entry point. The emitter only reads these immutable
// slices; it never rebuilds ABI layouts or allocates result-register arrays.
type jitPreparedGoABI struct {
	convention                jit.CallingConvention
	arguments                 []goCallArgLocation
	results                   []Reg
	resultCounts, resultMasks []uint8
	unsupportedResults        bool
}

var jitGoConventions sync.Map     // map[uint64]*jitPreparedGoABI
var jitGoTypeConventions sync.Map // map[reflect.Type]*jit.CallingConvention
var jitScmerClosureConvention = func() jit.CallingConvention {
	if !jit.SupportedEmitters() {
		return jit.CallingConvention{}
	}
	return *jitConventionForType(reflect.TypeFor[func(...Scmer) Scmer]())
}()

func jitRememberGoFunction(address uint64, fn any) {
	if !jit.SupportedEmitters() {
		return
	}
	if _, exists := jitGoConventions.Load(address); exists {
		return
	}
	c := *jitConventionForType(reflect.TypeOf(fn))
	p := &jitPreparedGoABI{convention: c}
	for _, input := range c.Inputs {
		for _, part := range input.Parts {
			loc := goCallArgLocation{size: part.Size}
			if part.Location.Kind == jit.InRegister {
				loc.inReg, loc.reg = true, Reg(part.Location.Register.Number)
				loc.fp = part.Location.Register.Class == jit.FloatRegister
			} else {
				loc.stackOff = int32(part.Location.StackOffset - c.Registers.StackArgumentOffset)
			}
			p.arguments = append(p.arguments, loc)
		}
	}
	p.resultCounts, p.resultMasks = make([]uint8, len(c.Outputs)), make([]uint8, len(c.Outputs))
	for i, output := range c.Outputs {
		words := (output.Type.Size() + 7) / 8
		if words < 1 || words > 3 {
			p.unsupportedResults = true
		}
		p.resultCounts[i] = uint8(words)
		for _, part := range output.Parts {
			if part.Pointer {
				p.resultMasks[i] |= 1 << uint(part.ValueOffset/8)
			}
			if part.Location.Kind != jit.InRegister || part.Location.Register.Class != jit.IntegerRegister {
				p.unsupportedResults = true
			}
			p.results = append(p.results, Reg(part.Location.Register.Number))
		}
	}
	jitGoConventions.LoadOrStore(address, p)
}

func jitConventionForType(signature reflect.Type) *jit.CallingConvention {
	if c, ok := jitGoTypeConventions.Load(signature); ok {
		return c.(*jit.CallingConvention)
	}
	c := jit.GetCallingConventionForType(signature)
	actual, _ := jitGoTypeConventions.LoadOrStore(signature, &c)
	return actual.(*jit.CallingConvention)
}

func jitPreparedABI(address uint64) *jitPreparedGoABI {
	p, ok := jitGoConventions.Load(address)
	if !ok {
		panic("jit: Go helper has no prepared function signature")
	}
	return p.(*jitPreparedGoABI)
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
	p := jitPreparedABI(address)
	if len(p.arguments) != len(words) {
		panic("jit: Go arguments do not match signature")
	}
	for _, loc := range p.arguments {
		if loc.fp {
			panic("jit: Go helper requires FP argument lowering")
		}
	}
	return p.arguments, int(p.convention.StackArgumentsSize / 8)
}

// The final sixteen bytes belong to the JIT bridge, outside all ABI homes.
func jitRuntimeCallBytes(address uint64) int32 {
	c := &jitScmerClosureConvention
	if address != 0 {
		c = &jitPreparedABI(address).convention
	}
	if c.CallFrameSize > 1<<30 {
		panic("jit: Go call frame too large")
	}
	return int32(c.CallFrameSize) + 16
}

func jitRuntimeResultRegs(address uint64, count int) []Reg {
	if count == 0 {
		return nil
	}
	p := jitPreparedABI(address)
	if p.unsupportedResults {
		panic("jit: Go result requires stack or FP lowering")
	}
	if count > len(p.results) {
		panic("jit: Go result count exceeds signature")
	}
	return p.results[:count]
}

func jitRuntimeGoResultShape(address uint64) ([]uint8, []uint8) {
	p := jitPreparedABI(address)
	if p.unsupportedResults {
		panic("jit: unsupported Go result representation")
	}
	return p.resultCounts, p.resultMasks
}

func jitRuntimeScmerClosureABI() jitClosureRegisters {
	if !jit.SupportedEmitters() {
		return jitLegacyClosureRegisters()
	}
	c := &jitScmerClosureConvention
	if len(c.Inputs) != 1 || len(c.Inputs[0].Parts) != 3 || len(c.Outputs) != 1 || len(c.Outputs[0].Parts) != 2 {
		panic("jit: unsupported Scmer closure representation")
	}
	partReg := func(p jit.ValuePart) Reg {
		if p.Location.Kind != jit.InRegister || p.Location.Register.Class != jit.IntegerRegister {
			panic("jit: unsupported Scmer closure ABI location")
		}
		return Reg(p.Location.Register.Number)
	}
	return jitClosureRegisters{partReg(c.Inputs[0].Parts[0]), partReg(c.Inputs[0].Parts[1]), partReg(c.Inputs[0].Parts[2]), Reg(c.Registers.Closure.Number), partReg(c.Outputs[0].Parts[0]), partReg(c.Outputs[0].Parts[1])}
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
	m := jit.EmitStackGrow(ctx, *jitConventionForType(signature), jit.PhysicalRegister{Class: jit.IntegerRegister, Number: uint8(ctx.FrameReg)}, int(ctx.Labels[retry]))
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

var jitRuntimeAllocatableGP = func() uint64 {
	if !jit.SupportedEmitters() {
		return jitArchFreeGPRegs()
	}
	return jitArchFreeGPRegs() & jit.RegisterConfig().Allocatable[jit.IntegerRegister]
}()

func jitRuntimeFreeGPRegs() uint64 { return jitRuntimeAllocatableGP }

// Proc's SSA-visible code word is also its FunctionHeader prefix. Fail the
// build if the runtime expands that prefix instead of silently reinterpreting
// adjacent Scheme fields. Storage funcvals embed FunctionHeader directly.
var _ [unsafe.Sizeof(jit.FunctionHeader{}) - unsafe.Sizeof(Proc{}.JITCode)]byte
var _ [unsafe.Sizeof(Proc{}.JITCode) - unsafe.Sizeof(jit.FunctionHeader{})]byte
var _ [0 - unsafe.Offsetof(Proc{}.JITCode)]byte

func jitRuntimeStoreScmer(address *Scmer, value Scmer) { jit.Store(address, value) }
func jitRuntimeClearScmer(address *Scmer)              { jit.Clear(address) }
func jitEmitRuntimeLoadFunctionContext(ctx *JITContext, target, slot Reg) {
	ctx.FlushRegisterMoves()
	jit.EmitLoadFunctionContext(ctx, jit.PhysicalRegister{Class: jit.IntegerRegister, Number: uint8(target)}, jit.PhysicalRegister{Class: jit.IntegerRegister, Number: uint8(slot)})
}

// The slow path clears old pointer contents through Go's deletion barrier.
// Its caller writes a proven non-heap-pointer value immediately afterwards.
func jitEmitPointerFreeStoreBarrier(ctx *JITContext, address Reg, restore func()) {
	ctx.FlushRegisterMoves()
	// Scratch is not an allocator home, but MemCP may keep its borrowed input
	// base there (R12 on amd64). Preserve that role on both branch paths, before
	// the slow path's ordinary Go-call preservation observes the registers.
	scratch := Reg(jitScmerClosureConvention.Registers.Scratch.Number)
	savedBase := int32(0)
	if ctx.SliceBase == scratch && !ctx.SliceBaseTracksRSP {
		savedBase = ctx.AllocSpill(8)
		ctx.EmitStoreRegMem(ctx.SliceBase, ctx.FrameReg, savedBase)
		ctx.setStackPointer(jitStackRootFrameBP, savedBase, true)
	}
	restoreBase := func() {
		if ctx.SliceBase != scratch {
			return
		}
		if ctx.SliceBaseTracksRSP {
			ctx.EmitLeaRegMem(ctx.SliceBase, ctx.StackReg, ctx.DynamicSP)
		} else {
			ctx.EmitMovRegMem(ctx.SliceBase, ctx.FrameReg, savedBase)
		}
	}
	check := jit.EmitWriteBarrierCheck(ctx, 0)
	restoreBase()
	done := ctx.ReserveLabel()
	ctx.EmitJmp(done)
	jit.PatchWriteBarrierCheck(ctx, check, ctx.Offset())
	restoreBase()
	// The call bridge may use its own scratch register before argument setup.
	// Give the transient interior pointer a rooted, frame-relative home first.
	off := ctx.AllocSpill(8)
	ctx.EmitStoreRegMem(address, ctx.FrameReg, off)
	ctx.setStackPointer(jitStackRootFrameBP, off, true)
	var results [16]Reg
	ctx.EmitGoCall(GoFuncAddr(jitRuntimeClearScmer), []goCallArgWord{{loc: LocStack, stackOff: off}}, 0, &results, nil)
	ctx.setStackPointer(jitStackRootFrameBP, off, false)
	if restore != nil {
		restore()
	}
	ctx.MarkLabel(done)
	if savedBase != 0 {
		ctx.setStackPointer(jitStackRootFrameBP, savedBase, false)
	}
}
