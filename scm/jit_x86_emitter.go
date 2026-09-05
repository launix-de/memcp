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
package scm

import (
	"fmt"
	"math"
	"unsafe"
)

// These emitter definitions are compiled on every architecture because the
// generated Declaration.JITEmit callbacks reference their API. jitEnabled is
// true only for amd64 builds with GOEXPERIMENT=jit, so non-amd64 builds retain
// their interpreter fallback and never execute the x86 machine code.

var jitCodeOverflowPanic = &struct{}{}

func jitCapturedEnv(en *Env) *JITEnv {
	if en == nil || en == &Globalenv {
		return nil
	}
	out := &JITEnv{Outer: jitCapturedEnv(en.Outer)}
	if len(en.VarsNumbered) != 0 {
		out.Numbered = make([]JITValueDesc, len(en.VarsNumbered))
		for i, value := range en.VarsNumbered {
			out.Numbered[i] = JITValueDesc{Loc: LocImm, Type: value.GetTag(), Imm: value}
		}
	}
	if len(out.Numbered) == 0 && out.Outer == nil {
		return nil
	}
	return out
}

// Keep the unwind marker above the register-argument spill area used by Go
// callees. MemCP's JIT call bridge supports at most nine ABI words (72 bytes).
const jitGoSpillBytes = uintptr(128)

// jitCompileProc compiles a Proc body to amd64 machine code or returns nil.
func jitCompileProc(proc *Proc) []byte {
	code, _ := jitCompileProcWithRoots(proc)
	return code
}

// jitCompileProcWithRoots compiles a Proc body to amd64 machine code and
// returns GC roots for pointer constants embedded into immediates.
func jitCompileProcWithRoots(proc *Proc) ([]byte, []unsafe.Pointer) {
	const defaultCodeBufSize = 16 * 1024
	ptr, arena, reservation := globalJITPool.Alloc(defaultCodeBufSize)
	buf := &execBuf{ptr: ptr, n: defaultCodeBufSize, arena: arena, reservation: reservation}
	codeLen, roots, _, _, _, _, _ := jitCompileProcToExec(proc, buf, true)
	arena.complete(reservation, buf.stackMaps)
	defer globalJITPool.Free(arena)
	if codeLen == 0 {
		return nil, nil
	}
	code := make([]byte, codeLen)
	copy(code, (*[1 << 30]byte)(buf.ptr)[:codeLen:codeLen])
	return code, roots
}

// jitCompileProcToExec compiles a Proc body directly into writable executable memory.
// Returns code length, GC roots, direct-entry dependencies, overflow status,
// hidden arguments, Go-callback metadata, and lowering coverage.
func jitCompileProcToExec(proc *Proc, buf *execBuf, recursiveLambdas bool) (int, []unsafe.Pointer, []*JITEntryPoint, bool, []JITHiddenArg, bool, JITCoverage) {
	body := proc.Body
	if body.GetTag() == tagSourceInfo {
		si := body.SourceInfo()
		if buf.arena != nil && si.source != "" {
			codeOffset := int32(uintptr(buf.ptr) - uintptr(buf.arena.base))
			buf.arena.addSourceEntry(jitSourceEntry{
				offset: codeOffset,
				file:   si.source,
				line:   int32(si.line),
			})
		}
		body = si.value
	}
	return jitCompileExprBodyToExec(proc, body, proc.NumVars, buf, recursiveLambdas)
}

// jitCompileExprBodyToExec compiles a Scheme expression body into a writable
// executable buffer using Declaration.JITEmit callbacks.
func jitCompileExprBodyToExec(proc *Proc, body Scmer, numVars int, buf *execBuf, recursiveLambdas bool) (codeLen int, roots []unsafe.Pointer, dependencies []*JITEntryPoint, overflow bool, hiddenArgs []JITHiddenArg, needsStableArgs bool, coverage JITCoverage) {
	defer func() {
		if r := recover(); r != nil {
			if r == jitCodeOverflowPanic {
				overflow = true
			}
			if JITLog {
				fmt.Println("JIT panic", r)
			}
			codeLen = 0
			roots = nil
			dependencies = nil
			hiddenArgs = nil
			needsStableArgs = false
			coverage = JITCoverage{}
		}
	}()
	numVars = jitRequiredLocalSlots(body, numVars)

	// Free registers: all GPRs except RAX (result ptr), RBX (result aux),
	// RSP, RBP, R11 (scratch), R12 (slice base), R14 (Go goroutine ptr "g")
	freeRegs := uint64((1 << uint(RegRCX)) | (1 << uint(RegRDX)) |
		(1 << uint(RegRSI)) | (1 << uint(RegRDI)) |
		(1 << uint(RegR8)) | (1 << uint(RegR9)) | (1 << uint(RegR10)) |
		(1 << uint(RegR13)) | (1 << uint(RegR15)))
	inputArgCount := -1
	if proc != nil && proc.Params.GetTag() == tagSlice {
		inputArgCount = len(proc.Params.Slice())
	}
	selfSymbols := jitSelfSymbols(proc)
	ctx := &JITContext{
		Ptr:              buf.ptr,
		Start:            buf.ptr,
		End:              unsafe.Add(buf.ptr, buf.n),
		FreeRegs:         freeRegs,
		AllRegs:          freeRegs,
		SliceBase:        RegR12,
		StackReg:         RegRSP,
		FrameReg:         RegRBP,
		ScratchReg:       RegR11,
		ResultPtrReg:     RegRAX,
		ResultAuxReg:     RegRBX,
		LastIntReg:       RegR15,
		HasFrame:         true,
		InputArgCount:    inputArgCount,
		LocalSlotCount:   numVars,
		RecursiveLambdas: recursiveLambdas,
		StackPhiTargets:  jitExpressionContainsParser(body),
		UsesRuntimeEnv:   jitExpressionConsumesRuntimeEnv(body),
		SelfSymbols:      selfSymbols,
		SelfParamCount:   inputArgCount,
		FrameRoots:       make(map[jitStackRoot]struct{}),
		Arena:            buf.arena,
	}
	if len(selfSymbols) != 0 && inputArgCount >= 0 {
		ctx.SelfLoopLabel = ctx.ReserveLabel()
		ctx.HasSelfLoop = true
	}
	runtimeEnv := &Globalenv
	if proc != nil && proc.En != nil {
		runtimeEnv = proc.En
	}
	ctx.RuntimeEnv = NewAny(runtimeEnv)
	ctx.TrackImm(ctx.RuntimeEnv)
	ctx.W = ctx // self-reference for backward-compat ctx.W.Emit calls

	guardOffset, stackSmall, moreStackPC := jitRuntimeStackCheck()
	var stackCheckFrameFixup unsafe.Pointer
	var stackRetryLabel, stackGrowLabel JITLabel
	hasStackCheck := moreStackPC != 0 && inputArgCount >= 0
	if hasStackCheck {
		stackRetryLabel = ctx.ReserveLabel()
		stackGrowLabel = ctx.ReserveLabel()
		ctx.MarkLabel(stackRetryLabel)
		ctx.emitMovRegReg(RegR11, RegRSP)
		ctx.emitBytes(0x49, 0x81, 0xEB) // sub r11, frameSize-StackSmall
		ctx.emitU32(0)
		stackCheckFrameFixup = unsafe.Add(ctx.Ptr, -4)
		ctx.EmitJcc(CondUnsignedBelow, stackGrowLabel)
		// cmp r11, [r14+stackguard0]
		ctx.emitBytes(0x4D, 0x3B, 0x9E)
		ctx.emitU32(uint32(guardOffset))
		ctx.EmitJcc(CondUnsignedBelowOrEqual, stackGrowLabel)
	}

	// Unified frame: push rbp; mov rbp, rsp; sub rsp, <fixup>
	// All frame access via [RSP + offset]. MaxBPOffset patched at the end.
	// Epilog: leave; ret.
	ctx.emitByte(0x55)                    // push rbp
	ctx.emitBytes(0x48, 0x89, 0xE5)       // mov rbp, rsp
	frameFixup := ctx.EmitSubRSP32Fixup() // sub rsp, <patched>
	var frameInitLabel, frameBodyLabel JITLabel
	if !ctx.StackPhiTargets {
		// Ordinary expressions use precise root-slot initialization instead of
		// clearing every word in the frame. The root set is known only after the
		// one-pass body emitter has recorded all safepoints, so enter through a
		// small initializer emitted after the body and jump back here.
		frameInitLabel = ctx.ReserveLabel()
		frameBodyLabel = ctx.ReserveLabel()
		ctx.EmitJmp(frameInitLabel)
		ctx.MarkLabel(frameBodyLabel)
	}

	ctx.emitMovRegReg(RegR12, RegRAX) // save incoming args slice
	// Parser control flow can enter a shared block whose stack target was
	// produced only on a sibling path. Clear parser frames so every root named
	// by such a block starts as nil. Ordinary expressions initialize their
	// pointer-bearing targets before publishing them in a safepoint map, so
	// clearing their entire frame would only add work to every invocation.
	var frameWordsFixup unsafe.Pointer
	if ctx.StackPhiTargets {
		ctx.emitMovRegReg(RegRDI, RegRSP)
		ctx.emitBytes(0x31, 0xC0) // xor eax, eax
		ctx.emitByte(0xB9)        // mov ecx, <frame words>
		frameWordsFixup = ctx.Ptr
		ctx.emitU32(0)
		ctx.emitBytes(0xF3, 0x48, 0xAB) // rep stosq
	}
	useInputFrame := proc != nil && proc.NumberedOnly && numVars == inputArgCount && !ctx.HasSelfLoop
	// Allocate local vars via AllocStack.
	if numVars > 0 && !useInputFrame {
		ctx.AllocStack(int32(numVars * 16))
		inputSlots := numVars
		if inputArgCount >= 0 && inputArgCount < inputSlots {
			inputSlots = inputArgCount
		}
		for i := 0; i < inputSlots; i++ {
			srcOff := int32(i * 16)
			dstOff := int32(i * 16)
			ctx.EmitMovRegMem(RegR11, RegR12, srcOff)
			ctx.EmitStoreRegMem(RegR11, RegRSP, dstOff)
			ctx.EmitMovRegMem(RegR11, RegR12, srcOff+8)
			ctx.EmitStoreRegMem(RegR11, RegRSP, dstOff+8)
			ctx.setStackPointer(jitStackRootFrameSP, dstOff, true)
		}
		if inputSlots < numVars {
			nilPtr, nilAux := NewNil().RawWords()
			for i := inputSlots; i < numVars; i++ {
				dstOff := int32(i * 16)
				ctx.EmitMovRegImm64(RegR11, uint64(nilPtr))
				ctx.EmitStoreRegMem(RegR11, RegRSP, dstOff)
				ctx.EmitMovRegImm64(RegR11, nilAux)
				ctx.EmitStoreRegMem(RegR11, RegRSP, dstOff+8)
				ctx.setStackPointer(jitStackRootFrameSP, dstOff, true)
			}
		}
		ctx.OriginalArgsOff = ctx.AllocStack(8)
		ctx.EmitStoreRegMem(RegR12, RegRSP, ctx.OriginalArgsOff)
		ctx.setStackPointer(jitStackRootFrameSP, ctx.OriginalArgsOff, true)
		ctx.emitMovRegReg(RegR12, RegRSP)
		ctx.SliceBaseTracksRSP = true
	}
	captureCount := 0
	if proc != nil && proc.Compiled != nil {
		captureCount = proc.Compiled.CaptureCount
	}
	if proc != nil && (captureCount != 0 || ctx.UsesRuntimeEnv) {
		ctx.ClosureFuncOff = ctx.AllocStack(8)
		ctx.EmitStoreRegMem(RegRDX, RegRSP, ctx.ClosureFuncOff)
		ctx.setStackPointer(jitStackRootFrameSP, ctx.ClosureFuncOff, true)
		if ctx.UsesRuntimeEnv {
			ctx.RuntimeEnvOff = ctx.AllocStack(8)
			ctx.EmitMovRegMem(RegR11, RegRDX, int32(unsafe.Offsetof(Proc{}.En)))
			ctx.EmitStoreRegMem(RegR11, RegRSP, ctx.RuntimeEnvOff)
			ctx.setStackPointer(jitStackRootFrameSP, ctx.RuntimeEnvOff, true)
		}
	}
	if ctx.HasSelfLoop {
		ctx.CurrentFuncOff = ctx.AllocStack(8)
		ctx.EmitStoreRegMem(RegRDX, RegRSP, ctx.CurrentFuncOff)
		ctx.setStackPointer(jitStackRootFrameSP, ctx.CurrentFuncOff, true)
	}

	// Map lambda parameters to local stack slots so symbol lookup remains correct
	// even when the optimizer did not rewrite body symbols to NthLocalVar.
	if proc != nil {
		captured := jitCapturedEnv(proc.En)
		var vars map[Symbol]JITValueDesc
		var numbered []JITValueDesc
		if numVars > 0 {
			numbered = make([]JITValueDesc, numVars)
			for index := range numbered {
				desc := JITValueDesc{Loc: LocInputPair, Type: JITTypeUnknown, StackOff: int32(index)}
				if !useInputFrame {
					desc.Loc = LocStackPair
					desc.StackOff = int32(index * 16)
				}
				numbered[index] = desc
			}
			if proc.Compiled != nil {
				for index := 0; index < proc.Compiled.CaptureCount; index++ {
					slot := proc.Compiled.CaptureBase + index
					if slot < 0 || slot >= len(numbered) {
						panic("jit: closure capture slot outside local frame")
					}
					numbered[slot] = JITValueDesc{Loc: LocClosurePair, Type: JITTypeUnknown, StackOff: int32(index)}
				}
			}
		}
		putVar := func(sym Symbol, index int) {
			if vars == nil {
				vars = make(map[Symbol]JITValueDesc, inputArgCount)
			}
			desc := JITValueDesc{Loc: LocInputPair, Type: JITTypeUnknown, StackOff: int32(index)}
			if index < len(numbered) {
				desc = numbered[index]
			}
			vars[sym] = desc
		}
		switch proc.Params.GetTag() {
		case tagSlice:
			params := proc.Params.Slice()
			for i := 0; i < len(params) && (inputArgCount < 0 || i < inputArgCount); i++ {
				param := params[i]
				for param.GetTag() == tagSourceInfo {
					param = param.SourceInfo().value
				}
				if param.GetTag() != tagSymbol {
					continue
				}
				putVar(param.Symbol(), i)
			}
		case tagSymbol:
			if inputArgCount > 0 {
				putVar(proc.Params.Symbol(), 0)
			}
		}
		if len(vars) > 0 || len(numbered) > 0 || captured != nil {
			ctx.Env = &JITEnv{Vars: vars, Numbered: numbered, Outer: captured}
		}
	}
	if ctx.HasSelfLoop {
		ctx.MarkLabel(ctx.SelfLoopLabel)
	}

	// Compile body, place result into RAX+RBX (Scmer return registers)
	result := JITValueDesc{Loc: LocRegPair, Reg: RegRAX, Reg2: RegRBX}
	desc := jitCompileExpr(ctx, body, RegR12, result)

	// If result came back as LocImm, materialize into RAX+RBX
	if desc.Loc == LocImm {
		// Eval treats literals as complete Scmer values. Preserve both words for
		// pointer-bearing and opaque literals instead of restricting native Proc
		// returns to the four unboxed primitive tags.
		desc = jitPlaceIntoPair(ctx, &desc, result)
		// fall through to epilog
	} else {
		ctx.EnsureDesc(&desc)
		switch desc.Loc {
		case LocRegPair:
			ctx.EmitMovPairToResult(&desc, &result)
		case LocReg:
			ret := JITValueDesc{Loc: LocRegPair, Reg: RegRAX, Reg2: RegRBX}
			switch desc.Type {
			case tagBool:
				ctx.EmitMakeBool(ret, desc)
			case tagInt:
				ctx.EmitMakeInt(ret, desc)
			case tagFloat:
				ctx.EmitMakeFloat(ret, desc)
			default:
				return 0, nil, nil, false, nil, false, JITCoverage{}
			}
		default:
			return 0, nil, nil, false, nil, false, JITCoverage{}
		}
	}
	// Unified epilog: patch SUB RSP with max frame size, then leave; ret.
	frameSize := ctx.MaxBPOffset + ctx.MaxSpillOffset
	frameSize = (frameSize + 15) &^ 15
	buf.stackFrameSize = frameSize
	ctx.PatchInt32(frameFixup, frameSize)
	if frameWordsFixup != nil {
		ctx.PatchInt32(frameWordsFixup, frameSize/8)
	}
	if hasStackCheck {
		// Account for the frame record and the deepest temporary call area below
		// the fixed frame. Like Go's outgoing-argument area, DynamicSP is part of
		// the maximum stack demand even though it is reserved only at call sites.
		checkedFrame := frameSize + ctx.MaxDynamicSP + 8 - int32(stackSmall)
		if checkedFrame < 0 {
			checkedFrame = 0
		}
		ctx.PatchInt32(stackCheckFrameFixup, checkedFrame)
	}
	arenaOffset := 0
	if buf.reservation != nil {
		arenaOffset = buf.reservation.offset
	}
	ctx.emitByte(0xC9) // leave
	ctx.emitByte(0xC3) // ret
	if !ctx.StackPhiTargets {
		ctx.MarkLabel(frameInitLabel)
		roots := jitSortedFrameRoots(ctx.FrameRoots)
		if len(roots) != 0 {
			ctx.emitBytes(0x45, 0x31, 0xDB) // xor r11d, r11d
			for _, root := range roots {
				base := RegRSP
				if root.base == jitStackRootFrameBP {
					base = RegRBP
				}
				ctx.EmitStoreRegMem(RegR11, base, root.offset)
			}
		}
		ctx.EmitJmp(frameBodyLabel)
	}
	if hasStackCheck {
		ctx.MarkLabel(stackGrowLabel)
		// Match Go's regabi prolog: public slice arguments use their caller-owned
		// spill homes, while runtime.morestack preserves DX as closure context.
		ctx.EmitStoreRegMem(RegRAX, RegRSP, 8)
		ctx.EmitStoreRegMem(RegRBX, RegRSP, 16)
		ctx.EmitStoreRegMem(RegRCX, RegRSP, 24)
		ctx.EmitMovRegImm64(RegR11, uint64(moreStackPC))
		ctx.emitBytes(0x41, 0xFF, 0xD3) // call r11
		ctx.Safepoints = append(ctx.Safepoints, jitSafepoint{
			pcOffset: int32(uintptr(ctx.Ptr) - uintptr(ctx.Start)),
			entry:    true,
		})
		ctx.EmitMovRegMem(RegRAX, RegRSP, 8)
		ctx.EmitMovRegMem(RegRBX, RegRSP, 16)
		ctx.EmitMovRegMem(RegRCX, RegRSP, 24)
		ctx.EmitJmp(stackRetryLabel)
	}
	buf.stackMaps = ctx.finalizeStackMaps(frameSize, arenaOffset)

	ctx.ResolveFixupsFinal()
	codeLen = int(uintptr(ctx.Ptr) - uintptr(ctx.Start))
	return codeLen, ctx.ConstRoots, ctx.EntryRoots, false, ctx.HiddenArgs, ctx.NeedsStableArgs, ctx.Coverage
}

const (
	RegRAX Reg = 0
	RegRCX Reg = 1
	RegRDX Reg = 2
	RegRBX Reg = 3
	RegRSP Reg = 4
	RegRBP Reg = 5
	RegRSI Reg = 6
	RegRDI Reg = 7
	RegR8  Reg = 8
	RegR9  Reg = 9
	RegR10 Reg = 10
	RegR11 Reg = 11
	RegR12 Reg = 12
	RegR13 Reg = 13
	RegR14 Reg = 14
	RegR15 Reg = 15
	// XMM registers start at 16
	RegX0 Reg = 16
	RegX1 Reg = 17
	RegX2 Reg = 18
	RegX3 Reg = 19
	RegX4 Reg = 20
	RegX5 Reg = 21
)

// emitByte appends a single byte to the writer.
func (ctx *JITContext) ensureSpace(n uintptr) {
	if uintptr(ctx.Ptr)+n > uintptr(ctx.End) {
		panic(jitCodeOverflowPanic)
	}
}

func (ctx *JITContext) emitByte(b byte) {
	ctx.ensureSpace(1)
	*(*byte)(ctx.Ptr) = b
	ctx.Ptr = unsafe.Add(ctx.Ptr, 1)
}

// emitBytes appends raw bytes to the writer.
func (ctx *JITContext) emitBytes(bs ...byte) {
	ctx.ensureSpace(uintptr(len(bs)))
	for _, b := range bs {
		*(*byte)(ctx.Ptr) = b
		ctx.Ptr = unsafe.Add(ctx.Ptr, 1)
	}
}

// emitU32 appends a little-endian uint32.
func (ctx *JITContext) emitU32(v uint32) {
	ctx.ensureSpace(4)
	*(*uint32)(ctx.Ptr) = v
	ctx.Ptr = unsafe.Add(ctx.Ptr, 4)
}

// emitU64 appends a little-endian uint64.
func (ctx *JITContext) emitU64(v uint64) {
	ctx.ensureSpace(8)
	*(*uint64)(ctx.Ptr) = v
	ctx.Ptr = unsafe.Add(ctx.Ptr, 8)
}

// --- Return emitters ---

// EmitReturnInt emits: MOV RAX, &scmerIntSentinel; MOV RBX, value; RET
// Constructs NewInt(value) in the return registers.
func (ctx *JITContext) EmitReturnInt(src JITValueDesc) {
	// MOV RAX, imm64 (address of scmerIntSentinel)
	ctx.emitBytes(0x48, 0xB8)
	ctx.emitU64(uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
	switch src.Loc {
	case LocReg:
		if src.Reg != RegRBX {
			// MOV RBX, src.Reg
			ctx.emitMovRegReg(RegRBX, src.Reg)
		}
	case LocImm:
		// MOV RBX, imm64
		ctx.emitBytes(0x48, 0xBB)
		ctx.emitU64(uint64(src.Imm.Int()))
	}
	ctx.emitByte(0xC3) // RET
}

// EmitReturnFloat emits: MOV RAX, &scmerFloatSentinel; MOVQ XMM→RBX; RET
// Constructs NewFloat(value) in the return registers.
func (ctx *JITContext) EmitReturnFloat(src JITValueDesc) {
	// MOV RAX, imm64 (address of scmerFloatSentinel)
	ctx.emitBytes(0x48, 0xB8)
	ctx.emitU64(uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
	switch src.Loc {
	case LocReg:
		// MOVQ XMM -> RBX: 66 48 0F 7E C3 (for X0→RBX)
		ctx.emitMovqXmmToGpr(RegRBX, src.Reg)
	case LocImm:
		// MOV RBX, imm64 (raw float bits)
		ctx.emitBytes(0x48, 0xBB)
		ctx.emitU64(math.Float64bits(src.Imm.Float()))
	}
	ctx.emitByte(0xC3) // RET
}

// EmitReturnNil emits: XOR EAX,EAX; XOR EBX,EBX; RET
func (ctx *JITContext) EmitReturnNil() {
	ctx.emitBytes(
		0x31, 0xC0, // XOR EAX, EAX
		0x31, 0xDB, // XOR EBX, EBX
		0xC3, // RET
	)
}

// EmitReturnBool emits: XOR EAX,EAX; MOV RBX, makeAux(tagBool, 0/1); RET
func (ctx *JITContext) EmitReturnBool(src JITValueDesc) {
	ctx.emitBytes(0x31, 0xC0) // XOR EAX, EAX (ptr = nil for bool)
	switch src.Loc {
	case LocImm:
		var val uint64
		if src.Imm.Bool() {
			val = 1
		}
		aux := makeAux(tagBool, val)
		ctx.emitBytes(0x48, 0xBB) // MOV RBX, imm64
		ctx.emitU64(aux)
	case LocReg:
		// Build aux = (bool&1)<<8 | tagBool.
		// Keep it branchless so callers can feed arbitrary integer predicates.
		// First zero-extend the bool into RBX.
		ctx.emitMovRegReg(RegRBX, src.Reg)
		ctx.emitBytes(0x48, 0x81, 0xE3) // AND RBX, 0x01
		ctx.emitU32(1)
		ctx.EmitShlRegImm8(RegRBX, 8)
		// MOV RCX, tagBool
		ctx.emitBytes(0x48, 0xB9) // MOV RCX, imm64
		ctx.emitU64(uint64(tagBool))
		// OR RBX, RCX
		ctx.emitBytes(0x48, 0x09, 0xCB)
	}
	ctx.emitByte(0xC3) // RET
}

// --- Scmer construction emitters (no RET) ---

// EmitMakeBool constructs a Scmer bool into dst.Reg (ptr) and dst.Reg2 (aux).
// src.Reg holds the 0/1 boolean value.
func (ctx *JITContext) EmitMakeBool(dst JITValueDesc, src JITValueDesc) {
	// dst.Reg = nil (XOR reg, reg)
	ctx.emitXorReg(dst.Reg)
	switch src.Loc {
	case LocImm:
		var bval uint64
		if src.Imm.Bool() {
			bval = 1
		}
		aux := makeAux(tagBool, bval)
		ctx.EmitMovRegImm64(dst.Reg2, aux)
	case LocReg:
		// dst.Reg2 = ((src.Reg & 1) << 8) | tagBool
		if dst.Reg2 != src.Reg {
			ctx.emitMovRegReg(dst.Reg2, src.Reg)
		}
		ctx.emitAndRegImm32(dst.Reg2, 1)
		ctx.EmitShlRegImm8(dst.Reg2, 8)
		ctx.EmitMovRegImm64(RegR11, uint64(tagBool))
		ctx.emitOrRegReg(dst.Reg2, RegR11)
	}
}

// EmitMakeInt constructs a Scmer int into dst.Reg (ptr) and dst.Reg2 (aux).
// src.Reg holds the int64 value.
func (ctx *JITContext) EmitMakeInt(dst JITValueDesc, src JITValueDesc) {
	ctx.EmitMovRegImm64(dst.Reg, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
	switch src.Loc {
	case LocReg:
		if dst.Reg2 != src.Reg {
			ctx.emitMovRegReg(dst.Reg2, src.Reg)
		}
	case LocImm:
		ctx.EmitMovRegImm64(dst.Reg2, uint64(src.Imm.Int()))
	}
}

// EmitMakeFloat constructs a Scmer float into dst.Reg (ptr) and dst.Reg2 (aux).
// src.Reg holds the float64 bits as uint64.
func (ctx *JITContext) EmitMakeFloat(dst JITValueDesc, src JITValueDesc) {
	ctx.EmitMovRegImm64(dst.Reg, uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
	switch src.Loc {
	case LocReg:
		if dst.Reg2 != src.Reg {
			ctx.emitMovRegReg(dst.Reg2, src.Reg)
		}
	case LocImm:
		ctx.EmitMovRegImm64(dst.Reg2, math.Float64bits(src.Imm.Float())) // float bits stored in aux
	}
}

// EmitMakeNil constructs a Scmer nil into dst.Reg (ptr) and dst.Reg2 (aux).
func (ctx *JITContext) EmitMakeNil(dst JITValueDesc) {
	ctx.emitXorReg(dst.Reg)
	ctx.emitXorReg(dst.Reg2)
}

// emitXorReg emits XOR r32, r32 (zeros 64-bit register via 32-bit op)
func (ctx *JITContext) emitXorReg(r Reg) {
	if r >= 8 {
		ctx.emitBytes(0x45, 0x31, byte(0xC0|(byte(r&7)<<3)|byte(r&7)))
	} else {
		ctx.emitBytes(0x31, byte(0xC0|(byte(r)<<3)|byte(r)))
	}
}

// emitAndRegImm32 emits AND r64, sign-extended imm32
func (ctx *JITContext) emitAndRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01
	}
	modrm := byte(0xE0) | byte(dst&7) // /4 = AND
	ctx.emitBytes(rex, 0x81, modrm)
	ctx.emitU32(uint32(imm))
}

// emitOrRegReg emits OR dst, src (64-bit)
func (ctx *JITContext) emitOrRegReg(dst, src Reg) {
	ctx.emitAluRegReg(0x09, dst, src) // OR r/m64, r64
}

// --- ALU emitters (type-specialized) ---

// EmitAddInt64 emits: ADD dst, src (GPR += GPR)
func (ctx *JITContext) EmitAddInt64(dst, src Reg) {
	ctx.emitAluRegReg(0x01, dst, src) // ADD r/m64, r64
}

// EmitSubInt64 emits: SUB dst, src (GPR -= GPR)
func (ctx *JITContext) EmitSubInt64(dst, src Reg) {
	ctx.emitAluRegReg(0x29, dst, src) // SUB r/m64, r64
}

// EmitImulInt64 emits: IMUL dst, src (GPR *= GPR, signed)
func (ctx *JITContext) EmitImulInt64(dst, src Reg) {
	// IMUL dst, src: REX.W + 0F AF /r (dst = dst * src)
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x04 // REX.R
	}
	if src >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(dst&7) << 3) | byte(src&7)
	ctx.emitBytes(rex, 0x0F, 0xAF, modrm)
}

// EmitAddFloat64 emits: ADDSD dst, src (XMM += XMM)
func (ctx *JITContext) EmitAddFloat64(dst, src Reg) {
	ctx.emitMovqGprToXmm(RegX0, dst)
	ctx.emitMovqGprToXmm(RegX1, src)
	ctx.emitSseOp(0x58, RegX0, RegX1) // ADDSD
	ctx.emitMovqXmmToGpr(dst, RegX0)
}

// EmitSubFloat64 emits: SUBSD dst, src (XMM -= XMM)
func (ctx *JITContext) EmitSubFloat64(dst, src Reg) {
	ctx.emitMovqGprToXmm(RegX0, dst)
	ctx.emitMovqGprToXmm(RegX1, src)
	ctx.emitSseOp(0x5C, RegX0, RegX1) // SUBSD
	ctx.emitMovqXmmToGpr(dst, RegX0)
}

// EmitMulFloat64 emits: MULSD dst, src (XMM *= XMM)
func (ctx *JITContext) EmitMulFloat64(dst, src Reg) {
	ctx.emitMovqGprToXmm(RegX0, dst)
	ctx.emitMovqGprToXmm(RegX1, src)
	ctx.emitSseOp(0x59, RegX0, RegX1) // MULSD
	ctx.emitMovqXmmToGpr(dst, RegX0)
}

// EmitDivFloat64 emits: DIVSD dst, src (XMM /= XMM)
func (ctx *JITContext) EmitDivFloat64(dst, src Reg) {
	ctx.emitMovqGprToXmm(RegX0, dst)
	ctx.emitMovqGprToXmm(RegX1, src)
	ctx.emitSseOp(0x5E, RegX0, RegX1) // DIVSD
	ctx.emitMovqXmmToGpr(dst, RegX0)
}

// EmitCmpFloat64Setcc compares two float64 bit-patterns from GPRs and writes
// 0/1 into dst using SETcc on the floating-point flags.
func (ctx *JITContext) EmitCmpFloat64Setcc(dst, left, right Reg, cc JITCondition) {
	// UCOMISD sets CF/ZF/PF semantics; map signed integer CCs used by generic
	// lowering to their unordered/unsigned floating-point equivalents.
	switch cc {
	case CcL:
		cc = CcB
	case CcLE:
		cc = CcBE
	case CcG:
		cc = CcA
	case CcGE:
		cc = CcAE
	}
	ctx.emitMovqGprToXmm(RegX0, left)
	ctx.emitMovqGprToXmm(RegX1, right)
	// UCOMISD XMM0, XMM1
	ctx.emitBytes(0x66, 0x0F, 0x2E, 0xC1)
	ctx.EmitSetcc(dst, cc)
}

// --- Conversion emitters ---

// EmitCvtInt64ToFloat64 converts an int64 in gprSrc to float64 bits in gprSrc.
// Uses the XMM register corresponding to gprSrc as scratch:
//
//	CVTSI2SDQ xmm(gprSrc), gprSrc   — int64 → float64 in XMM
//	MOVQ      gprSrc, xmm(gprSrc)   — extract float64 bits back to GPR
func (ctx *JITContext) EmitCvtInt64ToFloat64(xmmDst, gprSrc Reg) {
	xmm := xmmDst - 16 // convert to XMM index (unsigned underflow is fine)
	rex := byte(0x48)
	if xmm >= 8 {
		rex |= 0x04 // REX.R
	}
	if gprSrc >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(xmm&7) << 3) | byte(gprSrc&7)
	// CVTSI2SDQ xmm, gpr (int64 → float64 in XMM)
	ctx.emitBytes(0xF2, rex, 0x0F, 0x2A, modrm)
	// MOVQ xmm → gpr (66 REX.W 0F 7E /r) — extract float64 bits to GPR
	ctx.emitBytes(0x66, rex, 0x0F, 0x7E, modrm)
}

// EmitCvtFloatBitsToInt64 converts raw float64 bits in gprSrc to int64 in dst.
// Uses XMM0 as scratch:
//
//	MOVQ XMM0, gprSrc
//	CVTTSD2SI dst, XMM0
func (ctx *JITContext) EmitCvtFloatBitsToInt64(dst, gprSrc Reg) {
	ctx.emitMovqGprToXmm(RegX0, gprSrc)
	xmm := RegX0 - 16
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x04 // REX.R
	}
	if xmm >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(dst&7) << 3) | byte(xmm&7)
	ctx.emitBytes(0xF2, rex, 0x0F, 0x2C, modrm)
}

// EmitXorpdReg emits: XORPD xmm, xmm (zero a float register)
func (ctx *JITContext) EmitXorpdReg(xmm Reg) {
	r := xmm - 16
	modrm := byte(0xC0) | (byte(r&7) << 3) | byte(r&7)
	if r >= 8 {
		ctx.emitBytes(0x66, 0x45, 0x0F, 0x57, modrm)
	} else {
		ctx.emitBytes(0x66, 0x0F, 0x57, modrm)
	}
}

// --- Load emitters ---

// EmitLoadArgInt64 emits code to load the int64 value of the idx-th variadic
// arg directly from the Scmer slice. Only valid when JIT type = int64.
// Loads a[idx].aux (which IS the raw int64) into dstReg.
// sliceBase is the GPR holding the slice pointer.
func (ctx *JITContext) EmitLoadArgInt64(dst, sliceBase Reg, idx int) {
	// MOV dst, [sliceBase + idx*16 + 8]  (aux field)
	ctx.emitMovRegMem(dst, sliceBase, int32(idx*16+8))
}

// EmitLoadArgFloat64 emits code to load the float64 value of the idx-th arg.
// Only valid when JIT type = float64.
// Loads a[idx].aux bits into xmmDst via MOVQ.
func (ctx *JITContext) EmitLoadArgFloat64(xmmDst, sliceBase Reg, idx int) {
	// MOVQ xmm, [sliceBase + idx*16 + 8]
	ctx.emitMovqMemToXmm(xmmDst, sliceBase, int32(idx*16+8))
}

// EmitLoadArgPair loads the idx-th Scmer (ptr+aux pair) from the args slice.
func (ctx *JITContext) EmitLoadArgPair(dstPtr, dstAux, sliceBase Reg, idx int) {
	ctx.emitMovRegMem(dstPtr, sliceBase, int32(idx*16))   // ptr field
	ctx.emitMovRegMem(dstAux, sliceBase, int32(idx*16+8)) // aux field
}

// EmitByte emits a single byte (exported for test harnesses).
func (ctx *JITContext) EmitByte(b byte) {
	ctx.emitByte(b)
}

// --- Compare emitters ---

// EmitCmpInt64 emits: CMP reg1, reg2
func (ctx *JITContext) EmitCmpInt64(a, b Reg) {
	ctx.emitAluRegReg(0x39, a, b) // CMP r/m64, r64
}

// EmitJump emits a conditional branch through the x86 rel32 encoding.
func (ctx *JITContext) EmitJump(cc JITCondition, labelID JITLabel) {
	ctx.emitBytes(0x0F, 0x80|x86ConditionCode(cc)) // Jcc rel32
	ctx.AddFixup(labelID, 4, true)
	ctx.emitU32(0) // placeholder
}

// EmitJcc keeps already-generated emitters source-compatible.
func (ctx *JITContext) EmitJcc(cc JITCondition, labelID JITLabel) {
	ctx.EmitJump(cc, labelID)
}

// EmitJmp emits an unconditional JMP rel32.
func (ctx *JITContext) EmitJmp(labelID JITLabel) {
	ctx.emitByte(0xE9) // JMP rel32
	ctx.AddFixup(labelID, 4, true)
	ctx.emitU32(0) // placeholder
}

// EmitJmpToPos emits an unconditional JMP rel32 to an already-known code position.
func (ctx *JITContext) EmitJmpToPos(targetPos int32) {
	curPos := int32(uintptr(ctx.Ptr)-uintptr(ctx.Start)) + 5
	off := targetPos - curPos
	ctx.emitByte(0xE9) // JMP rel32
	ctx.emitU32(uint32(off))
}

// EmitJumpTable dispatches an unsigned integer to one of labels. The table
// stores offsets from the current JIT entry point, keeping it relocatable
// inside an arena. Parser continuations use this instead of a native call
// stack, so recursive grammars retain one runtime-visible JIT frame.
func (ctx *JITContext) EmitJumpTable(index Reg, labels []JITLabel, invalid JITLabel) {
	if len(labels) == 0 {
		ctx.EmitJmp(invalid)
		return
	}
	ctx.EmitCmpRegImm32(index, int32(len(labels)))
	ctx.EmitJump(CondUnsignedAboveOrEqual, invalid)

	// MOV R11, entry start
	ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(uintptr(ctx.Start)))
	// MOVSXD index, dword ptr [R11 + index*4 + tableOffset]. The table starts
	// directly after ADD+JMP below (six bytes).
	tableOffset := int32(uintptr(ctx.Ptr)-uintptr(ctx.Start)) + 8 + 6
	rex := byte(0x49) // REX.W + REX.B for the R11 base
	if index >= 8 {
		rex |= 0x04 // REX.R: destination register
		rex |= 0x02 // REX.X: SIB index register
	}
	ctx.emitBytes(rex, 0x63, 0x84|byte((index&7)<<3), 0x83|byte((index&7)<<3))
	ctx.emitU32(uint32(tableOffset))
	ctx.EmitAddInt64(ctx.ScratchReg, index)
	ctx.emitBytes(0x41, 0xFF, 0xE3) // JMP R11
	for _, label := range labels {
		ctx.AddFixup(label, 4, false)
		ctx.emitU32(0)
	}
}

func x86ConditionCode(cc JITCondition) byte {
	switch cc {
	case CcE:
		return 0x04
	case CcNE:
		return 0x05
	case CcBE:
		return 0x06
	case CcA:
		return 0x07
	case CcL:
		return 0x0C
	case CcGE:
		return 0x0D
	case CcLE:
		return 0x0E
	case CcG:
		return 0x0F
	case CcB:
		return 0x02
	case CcAE:
		return 0x03
	default:
		panic("jit: unsupported x86 condition")
	}
}

// --- MOV helpers ---

// emitMovRegReg emits MOV dst, src (64-bit GPR to GPR)
// EmitMovRegReg emits MOV r64, r64 (no-op if dst == src)
func (ctx *JITContext) EmitMovRegReg(dst, src Reg) {
	if dst == src {
		return
	}
	ctx.emitMovRegReg(dst, src)
}

func (ctx *JITContext) emitMovRegReg(dst, src Reg) {
	rex := byte(0x48)
	if src >= 8 {
		rex |= 0x04 // REX.R
	}
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(src&7) << 3) | byte(dst&7)
	ctx.emitBytes(rex, 0x89, modrm) // MOV r/m64, r64
}

// EmitMovRegImm64 loads an immediate into a 64-bit register using the
// shortest encoding: XOR reg,reg (2-3 B) for 0, MOV r32,imm32 (5-6 B)
// for values ≤ 0xFFFFFFFF, or full MOV r64,imm64 (10 B) otherwise.
func (ctx *JITContext) EmitMovRegImm64(dst Reg, imm uint64) {
	dstEnc := byte(dst & 7)
	if imm == 0 {
		// XOR r32, r32 — zero-extends to 64 bits (2 or 3 bytes)
		if dst >= 8 {
			ctx.EmitByte(0x45) // REX.R + REX.B
		}
		ctx.emitBytes(0x31, 0xC0|(dstEnc<<3)|dstEnc)
		return
	}
	if imm <= 0xFFFFFFFF {
		// MOV r32, imm32 — zero-extends to 64 bits (5 or 6 bytes)
		if dst >= 8 {
			ctx.EmitByte(0x41) // REX.B
		}
		ctx.EmitByte(0xB8 | dstEnc)
		ctx.emitU32(uint32(imm))
		return
	}
	// Full MOV r64, imm64 (10 bytes)
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	ctx.emitBytes(rex, 0xB8|dstEnc)
	ctx.emitU64(imm)
}

// emitRegMemOp emits <opcode> dst, [base + disp] (REX.W r64, r/m64 with ModRM)
// opcode: 0x8B = MOV (load), 0x8D = LEA (address computation)
func (ctx *JITContext) emitRegMemOp(opcode byte, dst, base Reg, disp int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x04 // REX.R
	}
	if base >= 8 {
		rex |= 0x01 // REX.B
	}
	baseEnc := byte(base & 7)
	dstEnc := byte(dst & 7)

	if disp == 0 && baseEnc != 5 { // RBP/R13 always needs disp
		modrm := (dstEnc << 3) | baseEnc
		if baseEnc == 4 { // RSP/R12 needs SIB
			ctx.emitBytes(rex, opcode, modrm, 0x24)
		} else {
			ctx.emitBytes(rex, opcode, modrm)
		}
	} else if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, opcode, modrm, 0x24, byte(int8(disp)))
		} else {
			ctx.emitBytes(rex, opcode, modrm, byte(int8(disp)))
		}
	} else {
		modrm := 0x80 | (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, opcode, modrm, 0x24)
		} else {
			ctx.emitBytes(rex, opcode, modrm)
		}
		ctx.emitU32(uint32(disp))
	}
}

// emitMovRegMem emits MOV dst, [base + disp32] (load 64-bit from memory)
func (ctx *JITContext) emitMovRegMem(dst, base Reg, disp int32) {
	ctx.emitRegMemOp(0x8B, dst, base, disp)
}

// EmitMovRegMem emits MOV dst, [base + disp32] (load 64-bit from memory) — exported wrapper.
func (ctx *JITContext) EmitMovRegMem(dst, base Reg, disp int32) {
	ctx.emitMovRegMem(dst, base, disp)
}

// EmitOrRegMem emits OR dst, [base+disp]. Common lowering uses this to combine
// words directly from spill slots without reserving another general register.
func (ctx *JITContext) EmitOrRegMem(dst, base Reg, disp int32) {
	ctx.emitRegMemOp(0x0B, dst, base, disp)
}

// EmitMovRegMemB emits MOVZX dst, byte [base + disp32] (8-bit zero-extended load).
func (ctx *JITContext) EmitMovRegMemB(dst, base Reg, disp int32) {
	ctx.emitRegMemOp2(0x0F, 0xB6, dst, base, disp)
}

// EmitMovRegMemW emits MOVZX dst, word [base + disp32] (16-bit zero-extended load).
func (ctx *JITContext) EmitMovRegMemW(dst, base Reg, disp int32) {
	ctx.emitRegMemOp2(0x0F, 0xB7, dst, base, disp)
}

// EmitMovRegMemL emits MOV r32, [base + disp32] (32-bit zero-extended load).
func (ctx *JITContext) EmitMovRegMemL(dst, base Reg, disp int32) {
	ctx.emitRegMemOp32(0x8B, dst, base, disp)
}

// EmitLeaRegMem emits LEA dst, [base + disp32] (compute address, no memory access)
// For IndexAddr: LEA dst, [sliceBase + idx*16] computes &a[idx]
func (ctx *JITContext) EmitLeaRegMem(dst, base Reg, disp int32) {
	ctx.emitRegMemOp(0x8D, dst, base, disp)
}

// EmitMovRegMem64 loads a 64-bit value from an absolute memory address into dst.
// Uses dst itself as scratch for the address (avoids clobbering R11).
func (ctx *JITContext) EmitMovRegMem64(dst Reg, addr uintptr) {
	ctx.EmitMovRegImm64(dst, uint64(addr))
	ctx.emitMovRegMem(dst, dst, 0)
}

// EmitMovRegMem32 loads a 32-bit value (zero-extended to 64 bits) from an absolute address.
// Uses dst itself as scratch for the address (avoids clobbering R11).
func (ctx *JITContext) EmitMovRegMem32(dst Reg, addr uintptr) {
	ctx.EmitMovRegImm64(dst, uint64(addr))
	// MOV r32, [dst+0] — 32-bit load zero-extends to 64 bits (no REX.W)
	ctx.emitRegMemOp32(0x8B, dst, dst, 0)
}

// EmitMovRegMem8 loads a byte (zero-extended to 64 bits) from an absolute address.
// Uses dst itself as scratch for the address (avoids clobbering R11).
func (ctx *JITContext) EmitMovRegMem8(dst Reg, addr uintptr) {
	ctx.EmitMovRegImm64(dst, uint64(addr))
	// MOVZX r64, byte [dst+0]
	ctx.emitRegMemOp2(0x0F, 0xB6, dst, dst, 0)
}

// EmitMovRegMem16 loads a 16-bit value (zero-extended to 64 bits) from an absolute address.
// Uses dst itself as scratch for the address (avoids clobbering R11).
func (ctx *JITContext) EmitMovRegMem16(dst Reg, addr uintptr) {
	ctx.EmitMovRegImm64(dst, uint64(addr))
	// MOVZX r64, word [dst+0]
	ctx.emitRegMemOp2(0x0F, 0xB7, dst, dst, 0)
}

// emitRegMemOp32 emits a 32-bit register-memory operation (no REX.W, for zero-extending loads).
func (ctx *JITContext) emitRegMemOp32(opcode byte, dst, base Reg, disp int32) {
	rex := byte(0x40)
	needRex := false
	if dst >= 8 {
		rex |= 0x04 // REX.R
		needRex = true
	}
	if base >= 8 {
		rex |= 0x01 // REX.B
		needRex = true
	}
	baseEnc := byte(base & 7)
	dstEnc := byte(dst & 7)

	if disp == 0 && baseEnc != 5 {
		modrm := (dstEnc << 3) | baseEnc
		if needRex {
			if baseEnc == 4 {
				ctx.emitBytes(rex, opcode, modrm, 0x24)
			} else {
				ctx.emitBytes(rex, opcode, modrm)
			}
		} else {
			if baseEnc == 4 {
				ctx.emitBytes(opcode, modrm, 0x24)
			} else {
				ctx.emitBytes(opcode, modrm)
			}
		}
	} else if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (dstEnc << 3) | baseEnc
		if needRex {
			if baseEnc == 4 {
				ctx.emitBytes(rex, opcode, modrm, 0x24, byte(int8(disp)))
			} else {
				ctx.emitBytes(rex, opcode, modrm, byte(int8(disp)))
			}
		} else {
			if baseEnc == 4 {
				ctx.emitBytes(opcode, modrm, 0x24, byte(int8(disp)))
			} else {
				ctx.emitBytes(opcode, modrm, byte(int8(disp)))
			}
		}
	} else {
		modrm := 0x80 | (dstEnc << 3) | baseEnc
		if needRex {
			if baseEnc == 4 {
				ctx.emitBytes(rex, opcode, modrm, 0x24)
			} else {
				ctx.emitBytes(rex, opcode, modrm)
			}
		} else {
			if baseEnc == 4 {
				ctx.emitBytes(opcode, modrm, 0x24)
			} else {
				ctx.emitBytes(opcode, modrm)
			}
		}
		ctx.emitU32(uint32(disp))
	}
}

// emitRegMemOp2 emits a 2-byte opcode register-memory operation with REX.W (for MOVZX etc.).
func (ctx *JITContext) emitRegMemOp2(op1, op2 byte, dst, base Reg, disp int32) {
	rex := byte(0x48) // REX.W
	if dst >= 8 {
		rex |= 0x04 // REX.R
	}
	if base >= 8 {
		rex |= 0x01 // REX.B
	}
	baseEnc := byte(base & 7)
	dstEnc := byte(dst & 7)

	if disp == 0 && baseEnc != 5 {
		modrm := (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, op1, op2, modrm, 0x24)
		} else {
			ctx.emitBytes(rex, op1, op2, modrm)
		}
	} else if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, op1, op2, modrm, 0x24, byte(int8(disp)))
		} else {
			ctx.emitBytes(rex, op1, op2, modrm, byte(int8(disp)))
		}
	} else {
		modrm := 0x80 | (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, op1, op2, modrm, 0x24)
		} else {
			ctx.emitBytes(rex, op1, op2, modrm)
		}
		ctx.emitU32(uint32(disp))
	}
}

// --- SSE helpers ---

// emitSseOp emits F2 0F <op> xmmDst, xmmSrc (scalar double operation)
func (ctx *JITContext) emitSseOp(op byte, dst, src Reg) {
	d := dst - 16 // XMM index
	s := src - 16
	rex := byte(0)
	if d >= 8 || s >= 8 {
		rex = 0x40
		if d >= 8 {
			rex |= 0x04
		}
		if s >= 8 {
			rex |= 0x01
		}
	}
	modrm := byte(0xC0) | (byte(d&7) << 3) | byte(s&7)
	if rex != 0 {
		ctx.emitBytes(0xF2, rex, 0x0F, op, modrm)
	} else {
		ctx.emitBytes(0xF2, 0x0F, op, modrm)
	}
}

// emitMovqXmmToGpr emits MOVQ gprDst, xmmSrc (66 REX.W 0F 7E /r)
func (ctx *JITContext) emitMovqXmmToGpr(gpr, xmm Reg) {
	x := xmm - 16
	rex := byte(0x48) // REX.W
	if x >= 8 {
		rex |= 0x04 // REX.R
	}
	if gpr >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(x&7) << 3) | byte(gpr&7)
	ctx.emitBytes(0x66, rex, 0x0F, 0x7E, modrm)
}

// emitMovqGprToXmm emits MOVQ xmmDst, gprSrc (66 REX.W 0F 6E /r)
func (ctx *JITContext) emitMovqGprToXmm(xmm, gpr Reg) {
	x := xmm - 16
	rex := byte(0x48)
	if x >= 8 {
		rex |= 0x04 // REX.R
	}
	if gpr >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(x&7) << 3) | byte(gpr&7)
	ctx.emitBytes(0x66, rex, 0x0F, 0x6E, modrm)
}

// emitMovqMemToXmm emits MOVQ xmmDst, [base + disp32] (F3 0F 7E /r m64)
func (ctx *JITContext) emitMovqMemToXmm(xmm, base Reg, disp int32) {
	x := xmm - 16
	rex := byte(0)
	if x >= 8 || base >= 8 {
		rex = 0x40
		if x >= 8 {
			rex |= 0x04
		}
		if base >= 8 {
			rex |= 0x01
		}
	}
	baseEnc := byte(base & 7)
	xEnc := byte(x & 7)

	if rex != 0 {
		ctx.emitBytes(0xF3, rex, 0x0F, 0x7E)
	} else {
		ctx.emitBytes(0xF3, 0x0F, 0x7E)
	}

	if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (xEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(modrm, 0x24, byte(int8(disp)))
		} else {
			ctx.emitBytes(modrm, byte(int8(disp)))
		}
	} else {
		modrm := 0x80 | (xEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(modrm, 0x24)
		} else {
			ctx.emitBytes(modrm)
		}
		ctx.emitU32(uint32(disp))
	}
}

// --- Compare helpers ---

// EmitCmpRegImm32 emits CMP r64, sign-extended imm32
func (ctx *JITContext) EmitCmpRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xF8) | byte(dst&7) // /7 = CMP
	ctx.emitBytes(rex, 0x81, modrm)
	ctx.emitU32(uint32(imm))
}

// EmitCmpRegImm8 emits CMP r8, imm8 on the low byte of the register.
// This is used for compact Scmer tag checks where tags live in aux[7:0].
func (ctx *JITContext) EmitCmpRegImm8(dst Reg, imm uint8) {
	rex := byte(0x40) // force low-byte register encoding (incl. SIL/DIL/BPL/SPL)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xF8) | byte(dst&7) // /7 = CMP, mod=11, r/m=dst
	ctx.emitBytes(rex, 0x80, modrm, imm)
}

// EmitAddRegImm32 emits ADD r64, sign-extended imm32.
func (ctx *JITContext) EmitAddRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | byte(dst&7) // /0 = ADD
	ctx.emitBytes(rex, 0x81, modrm)
	ctx.emitU32(uint32(imm))
}

// EmitSubRegImm32 emits SUB r64, sign-extended imm32.
func (ctx *JITContext) EmitSubRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE8) | byte(dst&7) // /5 = SUB
	ctx.emitBytes(rex, 0x81, modrm)
	ctx.emitU32(uint32(imm))
}

// EmitOrRegImm32 emits OR r64, sign-extended imm32.
func (ctx *JITContext) EmitOrRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC8) | byte(dst&7) // /1 = OR
	ctx.emitBytes(rex, 0x81, modrm)
	ctx.emitU32(uint32(imm))
}

// EmitImulRegImm32 emits IMUL r64, r64, imm32.
func (ctx *JITContext) EmitImulRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x05 // REX.R | REX.B (reg and r/m are both dst)
	}
	modrm := byte(0xC0) | (byte(dst&7) << 3) | byte(dst&7)
	ctx.emitBytes(rex, 0x69, modrm)
	ctx.emitU32(uint32(imm))
}

// EmitIdivRegImm emits signed integer division of dst by imm and stores the quotient in dst.
func (ctx *JITContext) EmitIdivRegImm(dst Reg, imm int64) {
	if imm == 0 {
		panic("jit: divide by zero in EmitIdivRegImm")
	}
	restoreRAX := dst != RegRAX
	restoreRDX := dst != RegRDX
	if restoreRAX {
		ctx.EmitPushReg(RegRAX)
	}
	if restoreRDX {
		ctx.EmitPushReg(RegRDX)
	}
	if dst != RegRAX {
		ctx.emitMovRegReg(RegRAX, dst)
	}
	// CQO sign-extends RAX into RDX:RAX for IDIV.
	ctx.emitBytes(0x48, 0x99)
	ctx.EmitMovRegImm64(RegR11, uint64(imm))
	// IDIV r/m64
	ctx.emitBytes(0x49, 0xF7, 0xFB) // idiv r11
	if dst != RegRAX {
		ctx.emitMovRegReg(dst, RegRAX)
	}
	if restoreRDX {
		ctx.EmitPopReg(RegRDX)
	}
	if restoreRAX {
		ctx.EmitPopReg(RegRAX)
	}
}

// EmitIremRegImm emits signed integer remainder of dst by imm and stores the remainder in dst.
func (ctx *JITContext) EmitIremRegImm(dst Reg, imm int64) {
	if imm == 0 {
		panic("jit: modulo by zero in EmitIremRegImm")
	}
	restoreRAX := dst != RegRAX
	restoreRDX := dst != RegRDX
	if restoreRAX {
		ctx.EmitPushReg(RegRAX)
	}
	if restoreRDX {
		ctx.EmitPushReg(RegRDX)
	}
	if dst != RegRAX {
		ctx.emitMovRegReg(RegRAX, dst)
	}
	// CQO sign-extends RAX into RDX:RAX for IDIV.
	ctx.emitBytes(0x48, 0x99)
	ctx.EmitMovRegImm64(RegR11, uint64(imm))
	// IDIV r/m64
	ctx.emitBytes(0x49, 0xF7, 0xFB) // idiv r11
	if dst != RegRDX {
		ctx.emitMovRegReg(dst, RegRDX)
	}
	if restoreRDX {
		ctx.EmitPopReg(RegRDX)
	}
	if restoreRAX {
		ctx.EmitPopReg(RegRAX)
	}
}

// EmitSetcc emits SETcc r/m8 + MOVZX r32, r8 → zero-extended 0 or 1 in full 64-bit register
func (ctx *JITContext) EmitSetcc(dst Reg, cc JITCondition) {
	opcode := x86ConditionCode(cc)
	dstEnc := byte(dst & 7)
	// SETcc r/m8: 0F 9x /0
	if dst >= 8 {
		ctx.emitBytes(0x41, 0x0F, 0x90|opcode, 0xC0|dstEnc)
	} else if dst >= 4 {
		ctx.emitBytes(0x40, 0x0F, 0x90|opcode, 0xC0|dstEnc) // REX for SIL/DIL/BPL/SPL
	} else {
		ctx.emitBytes(0x0F, 0x90|opcode, 0xC0|dstEnc)
	}
	// MOVZX r32, r8: 0F B6 /r (32-bit write zeros upper 32)
	modrm := byte(0xC0) | (dstEnc << 3) | dstEnc
	if dst >= 8 {
		ctx.emitBytes(0x45, 0x0F, 0xB6, modrm)
	} else if dst >= 4 {
		ctx.emitBytes(0x40, 0x0F, 0xB6, modrm)
	} else {
		ctx.emitBytes(0x0F, 0xB6, modrm)
	}
}

// --- Shift emitters ---

// EmitShlRegImm8 emits SHL r64, imm8 (logical shift left by immediate)
func (ctx *JITContext) EmitShlRegImm8(dst Reg, imm uint8) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE0) | byte(dst&7) // /4 = SHL
	ctx.emitBytes(rex, 0xC1, modrm, imm)
}

// EmitShrRegImm8 emits SHR r64, imm8 (logical shift right by immediate)
func (ctx *JITContext) EmitShrRegImm8(dst Reg, imm uint8) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE8) | byte(dst&7) // /5 = SHR
	ctx.emitBytes(rex, 0xC1, modrm, imm)
}

// EmitSarRegImm8 emits SAR r64, imm8 (arithmetic shift right by immediate)
func (ctx *JITContext) EmitSarRegImm8(dst Reg, imm uint8) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xF8) | byte(dst&7) // /7 = SAR
	ctx.emitBytes(rex, 0xC1, modrm, imm)
}

// EmitShlRegCl emits SHL r64, CL (shift left by variable amount in CL register)
func (ctx *JITContext) EmitShlRegCl(dst Reg) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE0) | byte(dst&7) // /4 = SHL
	ctx.emitBytes(rex, 0xD3, modrm)
}

// EmitShrRegCl emits SHR r64, CL (shift right by variable amount in CL register)
func (ctx *JITContext) EmitShrRegCl(dst Reg) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE8) | byte(dst&7) // /5 = SHR
	ctx.emitBytes(rex, 0xD3, modrm)
}

// EmitShlRegClGo64 implements Go's uint64 variable-shift semantics. x86 masks
// CL to six bits, whereas Go requires a zero result for counts of 64 or more.
func (ctx *JITContext) EmitShlRegClGo64(dst Reg) {
	ctx.EmitShlRegCl(dst)
	done := ctx.ReserveLabel()
	ctx.EmitCmpRegImm32(RegRCX, 64)
	ctx.EmitJcc(CondUnsignedBelow, done)
	ctx.EmitXorInt64(dst, dst)
	ctx.MarkLabel(done)
}

// EmitShrRegClGo64 implements Go's uint64 variable-shift semantics. See
// EmitShlRegClGo64 for why the range check is required on x86.
func (ctx *JITContext) EmitShrRegClGo64(dst Reg) {
	ctx.EmitShrRegCl(dst)
	done := ctx.ReserveLabel()
	ctx.EmitCmpRegImm32(RegRCX, 64)
	ctx.EmitJcc(CondUnsignedBelow, done)
	ctx.EmitXorInt64(dst, dst)
	ctx.MarkLabel(done)
}

// EmitAndRegImm32 emits AND r64, imm32 (sign-extended)
func (ctx *JITContext) EmitAndRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE0) | byte(dst&7) // /4 = AND
	ctx.emitBytes(rex, 0x81, modrm)
	ctx.emitU32(uint32(imm))
}

// EmitOrInt64 emits OR dst, src (64-bit OR)
func (ctx *JITContext) EmitOrInt64(dst, src Reg) {
	ctx.emitAluRegReg(0x09, dst, src) // OR r/m64, r64
}

// EmitAndInt64 emits AND dst, src (64-bit AND)
func (ctx *JITContext) EmitAndInt64(dst, src Reg) {
	ctx.emitAluRegReg(0x21, dst, src) // AND r/m64, r64
}

// EmitXorInt64 emits XOR dst, src (64-bit XOR).
//
// XOR is part of the architecture-neutral integer emitter contract. Regex
// literal lowering uses it to compare an unaligned input word against a
// compile-time literal before applying the byte significance mask.
func (ctx *JITContext) EmitXorInt64(dst, src Reg) {
	ctx.emitAluRegReg(0x31, dst, src) // XOR r/m64, r64
}

// EmitMaskedLiteralCheck compares one to eight bytes at [base+disp] with a
// compile-time literal. mask selects significant bits; callers clear bit 5 in
// ASCII letters for case-insensitive matching. The unaligned load and native
// byte order intentionally remain in the architecture-specific emitter.
func (ctx *JITContext) EmitMaskedLiteralCheck(base Reg, disp int32, literal, mask []byte, failLabel JITLabel) {
	if len(literal) == 0 || len(literal) > 8 || len(mask) != len(literal) {
		panic("jit: x86 masked literal check requires one to eight bytes")
	}
	loaded := ctx.AllocRegExcept(base)
	switch len(literal) {
	case 1:
		ctx.EmitMovRegMemB(loaded, base, disp)
	case 2:
		ctx.EmitMovRegMemW(loaded, base, disp)
	case 4:
		ctx.EmitMovRegMemL(loaded, base, disp)
	case 8:
		ctx.EmitMovRegMem(loaded, base, disp)
	default:
		panic("jit: x86 masked literal check width must be 1, 2, 4, or 8")
	}
	want := ctx.AllocRegExcept(base, loaded)
	var literalWord uint64
	var maskWord uint64
	for i := range literal {
		literalWord |= uint64(literal[i]) << (8 * i)
		maskWord |= uint64(mask[i]) << (8 * i)
	}
	ctx.EmitMovRegImm64(want, literalWord)
	ctx.EmitXorInt64(loaded, want)
	if maskWord != ^uint64(0)>>(64-8*len(mask)) {
		ctx.EmitMovRegImm64(want, maskWord)
		ctx.EmitAndInt64(loaded, want)
	}
	ctx.EmitCmpRegImm32(loaded, 0)
	ctx.EmitJump(CondNotEqual, failLabel)
	ctx.FreeReg(want)
	ctx.FreeReg(loaded)
}

// --- GetTag ---

// EmitGetTagDesc extracts the type tag from a Scmer value descriptor.
// Follows the standard emitter contract: consumes src (frees registers),
// places the tag int into result according to result.Loc.
func (ctx *JITContext) EmitGetTagDesc(src *JITValueDesc, result JITValueDesc) JITValueDesc {
	if src.Loc == LocImm {
		r := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(src.Imm.GetTag()))}
		if result.Loc == LocAny {
			return r
		}
		ctx.EmitMakeInt(result, r)
		return result
	}
	if src.Type != JITTypeUnknown {
		// Type is known at compile time — constant-fold
		ctx.FreeDesc(src)
		r := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(src.Type))}
		if result.Loc == LocAny {
			return r
		}
		ctx.EmitMakeInt(result, r)
		return result
	}
	// Dynamic type: materialize spilled descriptors before reading Reg/Reg2.
	ctx.EnsureDesc(src)
	dst := ctx.AllocReg()
	ctx.EmitGetTagRegs(dst, src.Reg, src.Reg2)
	ctx.FreeDesc(src)
	r := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: dst}
	if result.Loc == LocAny {
		return r
	}
	ctx.EmitMakeInt(result, r)
	ctx.FreeReg(dst)
	return result
}

// EmitTagEquals checks if a Scmer's type tag equals a constant.
// Equivalent to GetTag(src) == tag. Consumes src.
func (ctx *JITContext) EmitTagEquals(src *JITValueDesc, tag uint8, result JITValueDesc) JITValueDesc {
	if src.Loc == LocImm {
		r := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(src.Imm.GetTag() == tag)}
		if result.Loc == LocAny {
			return r
		}
		ctx.EmitMakeBool(result, r)
		return result
	}
	if src.Type != JITTypeUnknown {
		// Type is known at compile time — constant-fold
		ctx.FreeDesc(src)
		r := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(src.Type == tag)}
		if result.Loc == LocAny {
			return r
		}
		ctx.EmitMakeBool(result, r)
		return result
	}
	// Dynamic type: materialize spilled descriptors before reading Reg/Reg2.
	ctx.EnsureDesc(src)
	tagReg := ctx.AllocReg()
	ctx.EmitGetTagRegs(tagReg, src.Reg, src.Reg2)
	ctx.FreeDesc(src)
	ctx.EmitCmpRegImm8(tagReg, tag)
	ctx.EmitSetcc(tagReg, CcE)
	r := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: tagReg}
	if result.Loc == LocAny {
		return r
	}
	ctx.EmitMakeBool(result, r)
	ctx.FreeReg(tagReg)
	return result
}

// EmitTagEqualsBorrowed checks if a Scmer's tag equals a constant without
// consuming/clobbering the source descriptor. This is required when the same
// SSA value is used both for a type predicate and later value extraction.
func (ctx *JITContext) EmitTagEqualsBorrowed(src *JITValueDesc, tag uint8, result JITValueDesc) JITValueDesc {
	emitOut := func(v JITValueDesc) JITValueDesc {
		if result.Loc == LocAny {
			return v
		}
		ctx.EmitMakeBool(result, v)
		ctx.FreeDesc(&v)
		return result
	}

	// Immediate and known-typed values can be folded without touching source regs.
	if src.Loc == LocImm {
		return emitOut(JITValueDesc{
			Loc:  LocImm,
			Type: tagBool,
			Imm:  NewBool(src.Imm.GetTag() == tag),
		})
	}
	if src.Type != JITTypeUnknown {
		return emitOut(JITValueDesc{
			Loc:  LocImm,
			Type: tagBool,
			Imm:  NewBool(src.Type == tag),
		})
	}

	// Borrowed fast path: read tag directly from pair registers without cloning.
	if src.Loc == LocRegPair {
		ctx.ProtectReg(src.Reg)
		ctx.ProtectReg(src.Reg2)
		tagReg := ctx.AllocRegExcept(src.Reg, src.Reg2)
		ctx.UnprotectReg(src.Reg2)
		ctx.UnprotectReg(src.Reg)
		ctx.EmitGetTagRegs(tagReg, src.Reg, src.Reg2)
		ctx.EmitCmpRegImm8(tagReg, tag)
		ctx.EmitSetcc(tagReg, CcE)
		return emitOut(JITValueDesc{Loc: LocReg, Type: tagBool, Reg: tagReg})
	}

	// Other borrowed forms: detached copy so EmitTagEquals may consume it safely.
	tmp := *src
	tmp.ID = 0
	return ctx.EmitTagEquals(&tmp, tag, result)
}

// EmitBoolDesc evaluates Scmer truthiness equivalent to (Scmer).Bool().
// It consumes src and returns a bool descriptor (LocImm or LocReg).
// Fast paths are emitted for compile-time constants and known primitive types;
// dynamic/complex cases fall back to calling Scmer.Bool.
func (ctx *JITContext) EmitBoolDesc(src *JITValueDesc, result JITValueDesc) JITValueDesc {
	emitResult := func(v JITValueDesc) JITValueDesc {
		if result.Loc == LocAny {
			return v
		}
		ctx.EmitMakeBool(result, v)
		ctx.FreeDesc(&v)
		return result
	}

	if src.Loc == LocImm {
		return emitResult(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(src.Imm.Bool())})
	}
	if src.Type == tagNil {
		ctx.FreeDesc(src)
		return emitResult(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)})
	}
	if src.Type == tagDate {
		ctx.FreeDesc(src)
		return emitResult(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)})
	}

	// Known primitive types can be lowered directly without helper calls.
	if src.Type == tagBool || src.Type == tagInt || src.Type == tagFloat {
		ctx.EnsureDesc(src)
		srcLoc := src.Loc
		srcReg := src.Reg
		srcReg2 := src.Reg2
		var valReg Reg
		switch src.Loc {
		case LocReg:
			valReg = src.Reg
		case LocRegPair:
			valReg = src.Reg2 // aux payload contains bool/int/float bits
		default:
			// EnsureDesc should have materialized stack/mem forms.
			panic("jit: EmitBoolDesc primitive type not in register location")
		}

		dst := ctx.AllocReg()
		if valReg != dst {
			ctx.emitMovRegReg(dst, valReg)
		}

		if src.Type == tagFloat {
			// Float truthiness is float64(bits) != 0.0. Mask sign bit so -0.0
			// becomes zero, then compare against zero.
			mask := ctx.AllocReg()
			ctx.EmitMovRegImm64(mask, 0x7fffffffffffffff)
			ctx.EmitAndInt64(dst, mask)
			ctx.FreeReg(mask)
		} else if src.Type == tagBool && srcLoc == LocRegPair {
			// Bool payload is auxVal in bits [63:8]; low 8 bits hold the tag.
			ctx.EmitShrRegImm8(dst, 8)
		}
		ctx.EmitCmpRegImm32(dst, 0)
		ctx.EmitSetcc(dst, CcNE)

		// Keep the register that now carries the boolean result alive.
		// FreeDesc on an aliased source would otherwise free dst.
		switch srcLoc {
		case LocReg:
			if dst != srcReg {
				ctx.FreeReg(srcReg)
			}
		case LocRegPair:
			if dst == srcReg {
				ctx.FreeReg(srcReg2)
			} else if dst == srcReg2 {
				ctx.FreeReg(srcReg)
			} else {
				ctx.FreeReg(srcReg)
				ctx.FreeReg(srcReg2)
			}
		default:
			ctx.FreeDesc(src)
		}
		return emitResult(JITValueDesc{Loc: LocReg, Type: tagBool, Reg: dst})
	}

	// Unknown or complex known types (string/symbol/slice/vector/fastdict/default):
	// materialize a Scmer pair and reuse the canonical runtime helper.
	pair := *src
	if pair.Loc != LocRegPair {
		pair = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
		pair = jitPlaceIntoPair(ctx, src, pair)
	}
	out := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Bool), []JITValueDesc{pair}, 1)
	// Go bool returns may leave upper bits undefined; normalize to 0|1.
	ctx.EmitAndRegImm32(out.Reg, 1)
	out.Type = tagBool
	ctx.FreeDesc(&pair)
	src.Loc = LocNone
	return emitResult(out)
}

// EmitMovToReg moves a scalar JITValueDesc into a specific GPR register.
// Register self-copies intentionally emit no instruction.
func (ctx *JITContext) EmitMovToReg(dst Reg, src JITValueDesc) {
	switch src.Loc {
	case LocImm:
		ctx.EmitMovRegImm64(dst, uint64(src.Imm.Int()))
	case LocReg:
		ctx.EmitMovRegReg(dst, src.Reg)
	case LocStack:
		ctx.EmitMovRegMem(dst, RegRSP, src.StackOff)
	default:
		panic("jit: scalar move requires immediate, register, or stack source")
	}
}

// emitGetTagRegs emits inline code for (Scmer).GetTag().
// Input: ptrReg holds s.ptr, auxReg holds s.aux.
// Output: result in dstReg as uint16.
// Logic: if ptr == &scmerIntSentinel → tagInt (4)
//
//	if ptr == &scmerFloatSentinel → tagFloat (3)
//	else → aux & 0xFF
func (ctx *JITContext) EmitGetTagRegs(dst, ptrReg, auxReg Reg) {
	// CMP ptrReg, &scmerIntSentinel (via R11 as scratch)
	ctx.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
	ctx.EmitCmpInt64(ptrReg, RegR11)
	// JE .is_int (patch later)
	ctx.emitBytes(0x0F, 0x84) // JE rel32
	isIntFixup := ctx.Ptr
	ctx.emitU32(0)

	// CMP ptrReg, &scmerFloatSentinel
	ctx.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
	ctx.EmitCmpInt64(ptrReg, RegR11)
	// JE .is_float (patch later)
	ctx.emitBytes(0x0F, 0x84) // JE rel32
	isFloatFixup := ctx.Ptr
	ctx.emitU32(0)

	// Default: dst = aux & 0xFF
	if dst != auxReg {
		ctx.emitMovRegReg(dst, auxReg)
	}
	ctx.EmitAndRegImm32(dst, 0xFF)
	// JMP .done
	ctx.emitByte(0xE9) // JMP rel32
	doneFixup := ctx.Ptr
	ctx.emitU32(0)

	// .is_int: dst = tagInt (4)
	isIntTarget := ctx.Ptr
	ctx.EmitMovRegImm64(dst, uint64(tagInt))
	// JMP .done
	ctx.emitByte(0xE9) // JMP rel32
	doneFixup2 := ctx.Ptr
	ctx.emitU32(0)

	// .is_float: dst = tagFloat (3)
	isFloatTarget := ctx.Ptr
	ctx.EmitMovRegImm64(dst, uint64(tagFloat))
	// fall through to .done

	// .done:
	doneTarget := ctx.Ptr

	// Patch fixups
	*(*int32)(isIntFixup) = int32(uintptr(isIntTarget) - uintptr(isIntFixup) - 4)
	*(*int32)(isFloatFixup) = int32(uintptr(isFloatTarget) - uintptr(isFloatFixup) - 4)
	*(*int32)(doneFixup) = int32(uintptr(doneTarget) - uintptr(doneFixup) - 4)
	*(*int32)(doneFixup2) = int32(uintptr(doneTarget) - uintptr(doneFixup2) - 4)
}

// --- PUSH/POP/CALL ---

// EmitPushReg emits PUSH r64
func (ctx *JITContext) EmitPushReg(r Reg) {
	if r >= 8 {
		ctx.emitBytes(0x41, 0x50|byte(r&7))
	} else {
		ctx.emitByte(0x50 | byte(r))
	}
	ctx.addDynamicStack(8)
}

// EmitPopReg emits POP r64
func (ctx *JITContext) EmitPopReg(r Reg) {
	if r >= 8 {
		ctx.emitBytes(0x41, 0x58|byte(r&7))
	} else {
		ctx.emitByte(0x58 | byte(r))
	}
	ctx.DynamicSP -= 8
	if ctx.DynamicSP < 0 {
		panic("jit: unbalanced stack pop")
	}
}

// EmitCallIndirect emits an unwind marker followed by MOV R12, imm64; CALL R12.
func (ctx *JITContext) EmitCallIndirect(addr uint64) {
	ctx.emitCallIndirectWithSetup(addr, nil, nil)
}

func (ctx *JITContext) emitCallIndirectWithSetup(addr uint64, setup func(callFrameBytes int32), roots []int32) {
	ctx.EmitMovRegReg(RegR11, RegRBP)
	ctx.EmitSubInt64(RegR11, RegRSP)
	ctx.EmitPushReg(RegR11)
	ctx.EmitPushReg(RegR11)
	ctx.EmitSubRSP32(int32(jitGoSpillBytes))
	if setup != nil {
		setup(int32(jitGoSpillBytes + 16))
	}
	ctx.EmitMovRegImm64(RegR12, addr)
	ctx.emitBytes(0x41, 0xFF, 0xD4) // CALL R12
	ctx.recordSafepoint(roots, int32(jitGoSpillBytes+16))
	ctx.EmitAddRSP32(int32(jitGoSpillBytes + 16))
}

func (ctx *JITContext) regHoldsPointer(r Reg) bool {
	owner := ctx.RegOwners[r]
	if owner == nil {
		return false
	}
	switch owner.Loc {
	case LocReg:
		return owner.Reg == r && owner.RelocatablePointer
	case LocRegPair:
		return owner.Reg == r && !owner.NoHeapPointer
	case LocRegTriple:
		return owner.Reg == r
	default:
		return false
	}
}

// recordSafepoint snapshots pointer liveness at the return PC of a Go call.
// transientRoots are offsets from the caller-save area. callAreaBytes is the
// additional space below that area while the call is active: Go helpers need
// their unwind marker and register spill space, while the compact Proc ABI has
// no additional caller-owned frame.
func (ctx *JITContext) recordSafepoint(transientRoots []int32, callAreaBytes int32) {
	roots := make([]jitStackRoot, 0, len(ctx.StackRoots)+len(transientRoots))
	for root := range ctx.StackRoots {
		if root.base == jitStackRootFrameSP && root.offset < -ctx.DynamicSP {
			panic(fmt.Sprintf("jit: stale dynamic stack root raw=%d dynamic=%d", root.offset, ctx.DynamicSP))
		}
		roots = append(roots, root)
	}
	for _, offset := range transientRoots {
		roots = append(roots, jitStackRoot{
			base:   jitStackRootCallSP,
			offset: callAreaBytes + offset,
		})
	}
	ctx.Safepoints = append(ctx.Safepoints, jitSafepoint{
		pcOffset:  int32(uintptr(ctx.Ptr) - uintptr(ctx.Start)),
		dynamicSP: ctx.DynamicSP,
		roots:     roots,
	})
}

func (ctx *JITContext) finalizeStackMaps(frameSize int32, arenaOffset int) []jitStackMap {
	if ctx.DynamicSP != 0 {
		panic("jit: unbalanced dynamic stack at function exit")
	}
	maps := make([]jitStackMap, len(ctx.Safepoints))
	for i, safepoint := range ctx.Safepoints {
		frameBytes := frameSize + safepoint.dynamicSP
		if frameBytes < 0 || frameBytes%8 != 0 {
			panic("jit: invalid safepoint frame size")
		}
		frameWords := uintptr(frameBytes/8 + 1) // include saved RBP
		pointerMap := make([]byte, (frameWords+7)/8)
		mark := func(offset int32, root jitStackRoot) {
			if offset < 0 || offset%8 != 0 || offset > frameBytes {
				panic(fmt.Sprintf("jit: pointer root %d outside safepoint frame %d (dynamic=%d, base=%d, raw=%d)", offset, frameBytes, safepoint.dynamicSP, root.base, root.offset))
			}
			word := uintptr(offset / 8)
			pointerMap[word/8] |= 1 << (word % 8)
		}
		for _, root := range safepoint.roots {
			switch root.base {
			case jitStackRootFrameSP:
				mark(safepoint.dynamicSP+root.offset, root)
			case jitStackRootFrameBP:
				mark(safepoint.dynamicSP+frameSize+root.offset, root)
			case jitStackRootCallSP:
				mark(root.offset, root)
			default:
				panic("jit: invalid stack root base")
			}
		}
		maps[i] = jitStackMap{
			pcOffset:   uintptr(arenaOffset) + uintptr(safepoint.pcOffset),
			frameWords: frameWords,
			pointerMap: pointerMap,
			entry:      safepoint.entry,
		}
	}
	return maps
}

// EmitGoCallVariadic emits a direct call to a func(...Scmer) Scmer function value.
//
// amd64 regabi function-value call:
//   - RDX = funcval pointer (payload/closure pointer)
//   - RAX = slice.data, RBX = slice.len, RCX = slice.cap
//   - CALL [RDX] (fnptr)
//
// argslice must describe a pair (ptr,len). The backing array is expected to be
// materialized and kept alive by caller-managed stack memory.
func (ctx *JITContext) EmitGoCallVariadic(f func(...Scmer) Scmer, argslice JITValueDesc, result JITValueDesc) JITValueDesc {
	fnData := *(*uintptr)(unsafe.Pointer(&f))
	if fnData == 0 {
		panic("jit: nil variadic function value")
	}

	arg := argslice
	if arg.Loc == LocStackPair {
		ctx.EnsureDesc(&arg)
	}
	if arg.Loc == LocImm {
		if !arg.Imm.IsNil() {
			panic("jit: variadic argslice LocImm must be nil")
		}
		arg = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
		ctx.EmitMovRegImm64(arg.Reg, 0)
		ctx.EmitMovRegImm64(arg.Reg2, 0)
		ctx.BindReg(arg.Reg, &arg)
		ctx.BindReg(arg.Reg2, &arg)
	}
	if arg.Loc != LocRegPair {
		panic(fmt.Sprintf("jit: variadic argslice must be LocRegPair/LocStackPair (got %d)", arg.Loc))
	}

	requestedTarget := result
	targetHasRegs := requestedTarget.Loc == LocRegPair
	target := requestedTarget
	if targetHasRegs {
		targetOff := ctx.AllocSpill(16)
		target = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: targetOff, Rooted: true}
		ctx.setStackPointer(jitStackRootFrameBP, targetOff, true)
	} else if target.Loc == LocStackPair {
		target.Type = JITTypeUnknown
	} else {
		targetOff := ctx.AllocSpill(16)
		target = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: targetOff, Rooted: true}
		ctx.setStackPointer(jitStackRootFrameBP, targetOff, true)
	}

	ctx.ReclaimUntrackedRegs()
	var liveRegsArr [16]Reg
	liveCount := 0
	for r := Reg(0); r <= RegR15; r++ {
		if r == RegRSP || r == RegRBP || r == RegR11 || r == RegR14 {
			continue
		}
		bit := uint64(1 << uint(r))
		if (ctx.AllRegs&bit) == 0 || (ctx.FreeRegs&bit) != 0 {
			continue
		}
		owner := ctx.RegOwners[r]
		if owner == nil {
			continue
		}
		valid := false
		switch owner.Loc {
		case LocReg:
			valid = owner.Reg == r
		case LocRegPair:
			valid = owner.Reg == r || owner.Reg2 == r
		}
		if !valid {
			continue
		}
		liveRegsArr[liveCount] = r
		liveCount++
	}
	liveRegs := liveRegsArr[:0]
	for i := 0; i < liveCount; i++ {
		r := liveRegsArr[i]
		if r == arg.Reg || r == arg.Reg2 {
			continue
		}
		if targetHasRegs {
			if r == requestedTarget.Reg || r == requestedTarget.Reg2 {
				continue
			}
		}
		liveRegs = append(liveRegs, r)
	}
	switch ctx.SliceBase {
	case RegRSP, RegRBP, RegR11, RegR14:
	default:
		if !ctx.SliceBaseTracksRSP {
			found := false
			for _, r := range liveRegs {
				if r == ctx.SliceBase {
					found = true
					break
				}
			}
			if !found {
				liveRegs = append(liveRegs, ctx.SliceBase)
			}
		}
	}

	// Single per-call frame (thread-safe):
	// [rsp+0..] = saved live regs
	frameBytes := int32(len(liveRegs)) * 8
	if frameBytes%16 != 0 {
		frameBytes += 8 // keep alignment equivalent to pre-call RSP parity
	}
	if frameBytes != 0 {
		ctx.EmitSubRSP32(frameBytes)
	}

	for i, r := range liveRegs {
		ctx.EmitStoreRegMem(r, RegRSP, int32(i*8))
	}
	transientRoots := make([]int32, 0, len(liveRegs))
	for i, r := range liveRegs {
		if ctx.regHoldsPointer(r) || r == ctx.SliceBase {
			transientRoots = append(transientRoots, int32(i*8))
		}
	}

	// Stage argslice into scratch regs, then set call registers.
	if arg.Reg != RegRAX {
		ctx.EmitMovRegReg(RegRAX, arg.Reg)
	}
	if arg.Reg2 != RegRBX {
		ctx.EmitMovRegReg(RegRBX, arg.Reg2)
	}
	ctx.EmitMovRegReg(RegRCX, RegRBX) // cap = len
	ctx.EmitMovRegImm64(RegRDX, uint64(fnData))
	ctx.EmitMovRegMem(RegR11, RegRDX, 0) // fnptr := [funcval]
	ctx.EmitMovRegReg(RegR13, RegRBP)
	ctx.EmitSubInt64(RegR13, RegRSP)
	ctx.EmitPushReg(RegR13)
	ctx.EmitPushReg(RegR13)
	ctx.EmitSubRSP32(int32(jitGoSpillBytes))
	ctx.emitBytes(0x41, 0xFF, 0xD3) // CALL R11
	ctx.recordSafepoint(transientRoots, int32(jitGoSpillBytes+16))
	ctx.EmitAddRSP32(int32(jitGoSpillBytes + 16))

	callResult := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX}
	base := ctx.StackReg
	targetOff := target.StackOff
	if target.StackOff < 0 {
		base = ctx.FrameReg
	} else {
		// Live-register preservation temporarily moves RSP below the fixed JIT
		// frame. Positive descriptors remain relative to the fixed frame base,
		// so compensate until the saved registers have been restored.
		targetOff += ctx.DynamicSP
	}
	ctx.EmitStoreRegMem(callResult.Reg, base, targetOff)
	ctx.EmitStoreRegMem(callResult.Reg2, base, targetOff+8)
	for i, r := range liveRegs {
		ctx.EmitMovRegMem(r, RegRSP, int32(i*8))
	}
	if frameBytes != 0 {
		ctx.EmitAddRSP32(frameBytes)
	}
	if targetHasRegs {
		ctx.EmitMovRegMem(requestedTarget.Reg, base, targetOff)
		ctx.EmitMovRegMem(requestedTarget.Reg2, base, targetOff+8)
		requestedTarget.Type = JITTypeUnknown
		ctx.BindReg(requestedTarget.Reg, &requestedTarget)
		ctx.BindReg(requestedTarget.Reg2, &requestedTarget)
		target = requestedTarget
	}

	if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
		ctx.emitMovRegReg(ctx.SliceBase, RegRSP)
	}
	return target
}

type jitFuncValueCallKind uint8

const (
	jitProcCall jitFuncValueCallKind = iota
	jitGoFuncCall
)

// EmitProcCall bitcasts a JIT-capable *Proc to func(...Scmer) Scmer and calls
// it through the compact JIT-to-JIT ABI. The Proc starts with the machine-code
// pointer and its inline capture context follows the Proc header, exactly like
// a Go funcval and its closure words.
func (ctx *JITContext) EmitProcCall(proc, argslice, result JITValueDesc) JITValueDesc {
	return ctx.emitFuncValueCall(proc, argslice, result, jitProcCall)
}

// EmitGoFuncCall calls a regular Go func(...Scmer) Scmer value. Unlike a JIT
// Proc, a Go callee requires the patched runtime's foreign-frame transition.
func (ctx *JITContext) EmitGoFuncCall(fn, argslice, result JITValueDesc) JITValueDesc {
	return ctx.emitFuncValueCall(fn, argslice, result, jitGoFuncCall)
}

// emitFuncValueCall contains only the register-save and shared regabi
// machinery. The call kind deliberately remains private: common JIT lowering
// must choose explicitly between EmitProcCall and EmitGoFuncCall.
func (ctx *JITContext) emitFuncValueCall(fn, argslice, result JITValueDesc, kind jitFuncValueCallKind) JITValueDesc {
	ctx.EnsureDescsTogether(&fn, &argslice)
	if fn.Loc != LocReg || argslice.Loc != LocRegPair || result.Loc != LocStackPair || result.StackOff >= 0 {
		panic("jit: invalid Proc.JIT call placement")
	}

	var liveRegsBuf [16]Reg
	liveRegs := ctx.collectLiveRegsForCall(&liveRegsBuf)
	kept := liveRegs[:0]
	for _, reg := range liveRegs {
		if reg != fn.Reg && reg != argslice.Reg && reg != argslice.Reg2 {
			kept = append(kept, reg)
		}
	}
	liveRegs = kept
	if !ctx.SliceBaseTracksRSP {
		found := false
		for _, reg := range liveRegs {
			found = found || reg == ctx.SliceBase
		}
		if !found {
			liveRegs = append(liveRegs, ctx.SliceBase)
		}
	}
	frameBytes := int32(len(liveRegs) * 8)
	if frameBytes%16 != 0 {
		frameBytes += 8
	}
	if frameBytes != 0 {
		ctx.EmitSubRSP32(frameBytes)
	}
	for index, reg := range liveRegs {
		ctx.EmitStoreRegMem(reg, ctx.StackReg, int32(index*8))
	}
	transientRoots := make([]int32, 0, len(liveRegs))
	for index, reg := range liveRegs {
		if ctx.regHoldsPointer(reg) || reg == ctx.SliceBase {
			transientRoots = append(transientRoots, int32(index*8))
		}
	}

	// Stage the slice and function value as one parallel move. The allocator may
	// place the function value in RAX/RBX or either slice word in RDX; sequential
	// moves would then destroy a source before its final consumer reads it.
	ctx.emitParallelRegMoves([]jitRegMove{
		{dst: RegRAX, src: argslice.Reg},
		{dst: RegRBX, src: argslice.Reg2},
		{dst: RegRDX, src: fn.Reg},
	})
	ctx.EmitMovRegReg(RegRCX, RegRBX)
	ctx.EmitMovRegMem(RegR11, RegRDX, 0)

	callAreaBytes := int32(jitGoSpillBytes)
	if kind == jitGoFuncCall {
		ctx.EmitMovRegReg(RegR13, RegRBP)
		ctx.EmitSubInt64(RegR13, RegRSP)
		ctx.EmitPushReg(RegR13)
		ctx.EmitPushReg(RegR13)
		callAreaBytes += 16
	}
	// Go callees own spill homes for register arguments in the caller's frame.
	// A JIT callee uses these homes before runtime.morestack just like compiled
	// Go code, so reserve the standard call area even on the compact path.
	ctx.EmitSubRSP32(int32(jitGoSpillBytes))
	ctx.emitBytes(0x41, 0xFF, 0xD3) // CALL R11
	ctx.recordSafepoint(transientRoots, callAreaBytes)
	ctx.EmitAddRSP32(callAreaBytes)

	ctx.EmitStoreRegMem(RegRAX, ctx.FrameReg, result.StackOff)
	ctx.EmitStoreRegMem(RegRBX, ctx.FrameReg, result.StackOff+8)
	for index, reg := range liveRegs {
		ctx.EmitMovRegMem(reg, ctx.StackReg, int32(index*8))
	}
	if frameBytes != 0 {
		ctx.EmitAddRSP32(frameBytes)
	}
	if ctx.SliceBaseTracksRSP {
		if ctx.SliceBase != ctx.StackReg {
			ctx.EmitMovRegReg(ctx.SliceBase, ctx.StackReg)
		}
	}
	return result
}

// emitStoreRegMem emits MOV [base + disp], src (store 64-bit register to memory)
func (ctx *JITContext) emitStoreRegMem(src, base Reg, disp int32) {
	rex := byte(0x48)
	if src >= 8 {
		rex |= 0x04 // REX.R
	}
	if base >= 8 {
		rex |= 0x01 // REX.B
	}
	baseEnc := byte(base & 7)
	srcEnc := byte(src & 7)

	if disp == 0 && baseEnc != 5 {
		modrm := (srcEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, 0x89, modrm, 0x24)
		} else {
			ctx.emitBytes(rex, 0x89, modrm)
		}
	} else if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (srcEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, 0x89, modrm, 0x24, byte(int8(disp)))
		} else {
			ctx.emitBytes(rex, 0x89, modrm, byte(int8(disp)))
		}
	} else {
		modrm := 0x80 | (srcEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, 0x89, modrm, 0x24)
		} else {
			ctx.emitBytes(rex, 0x89, modrm)
		}
		ctx.emitU32(uint32(disp))
	}
}

// --- GPR ALU encoding helper ---

// emitAluRegReg emits a REX.W ALU op: <opcode> r/m64, r64
// opcode: 0x01=ADD, 0x29=SUB, 0x39=CMP, 0x09=OR, 0x21=AND, 0x31=XOR
func (ctx *JITContext) emitAluRegReg(opcode byte, dst, src Reg) {
	rex := byte(0x48)
	if src >= 8 {
		rex |= 0x04
	}
	if dst >= 8 {
		rex |= 0x01
	}
	modrm := byte(0xC0) | (byte(src&7) << 3) | byte(dst&7)
	ctx.emitBytes(rex, opcode, modrm)
}

// EmitStoreRegMem is the exported version of emitStoreRegMem:
// MOV [base+disp], src (64-bit store).
func (ctx *JITContext) EmitStoreRegMem(src, base Reg, disp int32) {
	ctx.emitStoreRegMem(src, base, disp)
}

// EmitSubRSP emits SUB RSP, imm8 to reserve stack space.
func (ctx *JITContext) EmitSubRSP(n uint8) {
	ctx.emitBytes(0x48, 0x83, 0xEC, n)
	ctx.addDynamicStack(int32(n))
}

// EmitAddRSP emits ADD RSP, imm8 to release stack space.
func (ctx *JITContext) EmitAddRSP(n uint8) {
	ctx.emitBytes(0x48, 0x83, 0xC4, n)
	oldDynamicSP := ctx.DynamicSP
	ctx.DynamicSP -= int32(n)
	if ctx.DynamicSP < 0 {
		panic("jit: unbalanced stack release")
	}
	for root := range ctx.StackRoots {
		if root.base == jitStackRootFrameSP && root.offset >= -oldDynamicSP && root.offset < -ctx.DynamicSP {
			delete(ctx.StackRoots, root)
		}
	}
}

// EmitSubRSP32Fixup emits SUB RSP, imm32 with a zero placeholder and returns
// a pointer to the 4-byte immediate so it can be patched later via PatchInt32.
func (ctx *JITContext) EmitSubRSP32Fixup() unsafe.Pointer {
	ctx.emitBytes(0x48, 0x81, 0xEC)
	ctx.emitU32(0)
	return unsafe.Add(ctx.Ptr, -4)
}

// JITStandaloneFrame records compiler state while an emitter which is not
// nested in a normal JIT procedure uses its own machine stack frame.
type JITStandaloneFrame struct {
	active                bool
	fixup                 unsafe.Pointer
	bpOffset, maxBPOffset int32
	spillOffset, maxSpill int32
	stackReg, frameReg    Reg
	scratchReg            Reg
}

// BeginStandaloneFrame gives independently callable generated emitters a
// spill frame. Emitters nested in a compiled procedure already share its frame
// and take the no-op path, so production scan loops pay no extra prologue.
func (ctx *JITContext) BeginStandaloneFrame() JITStandaloneFrame {
	if ctx.HasFrame {
		return JITStandaloneFrame{}
	}
	state := JITStandaloneFrame{
		active:      true,
		bpOffset:    ctx.BPOffset,
		maxBPOffset: ctx.MaxBPOffset,
		spillOffset: ctx.SpillOffset,
		maxSpill:    ctx.MaxSpillOffset,
		stackReg:    ctx.StackReg,
		frameReg:    ctx.FrameReg,
		scratchReg:  ctx.ScratchReg,
	}
	ctx.emitByte(0x55)              // push rbp
	ctx.emitBytes(0x48, 0x89, 0xE5) // mov rbp, rsp
	state.fixup = ctx.EmitSubRSP32Fixup()
	ctx.StackReg = RegRSP
	ctx.FrameReg = RegRBP
	ctx.ScratchReg = RegR11
	ctx.HasFrame = true
	ctx.BPOffset = 0
	ctx.MaxBPOffset = 0
	ctx.SpillOffset = 0
	ctx.MaxSpillOffset = 0
	return state
}

// EndStandaloneFrame closes the frame opened by BeginStandaloneFrame and
// restores the surrounding compiler's allocation coordinates.
func (ctx *JITContext) EndStandaloneFrame(state JITStandaloneFrame) {
	if !state.active {
		return
	}
	frameSize := (ctx.MaxBPOffset + ctx.MaxSpillOffset + 15) &^ 15
	ctx.PatchInt32(state.fixup, frameSize)
	ctx.emitByte(0xC9) // leave
	ctx.BPOffset = state.bpOffset
	ctx.MaxBPOffset = state.maxBPOffset
	ctx.SpillOffset = state.spillOffset
	ctx.MaxSpillOffset = state.maxSpill
	ctx.StackReg = state.stackReg
	ctx.FrameReg = state.frameReg
	ctx.ScratchReg = state.scratchReg
	ctx.HasFrame = false
}

// PatchInt32 writes a 32-bit little-endian value at the given position.
func (ctx *JITContext) PatchInt32(pos unsafe.Pointer, val int32) {
	*(*int32)(pos) = val
}

// EmitAddRSP32 emits ADD RSP, imm32.
func (ctx *JITContext) EmitAddRSP32(val int32) {
	ctx.emitBytes(0x48, 0x81, 0xC4)
	ctx.emitU32(uint32(val))
	oldDynamicSP := ctx.DynamicSP
	ctx.DynamicSP -= val
	if ctx.DynamicSP < 0 {
		panic("jit: unbalanced stack release")
	}
	for root := range ctx.StackRoots {
		if root.base == jitStackRootFrameSP && root.offset >= -oldDynamicSP && root.offset < -ctx.DynamicSP {
			delete(ctx.StackRoots, root)
		}
	}
}

// EmitSubRSP32 emits SUB RSP, imm32.
func (ctx *JITContext) EmitSubRSP32(val int32) {
	ctx.emitBytes(0x48, 0x81, 0xEC)
	ctx.emitU32(uint32(val))
	ctx.addDynamicStack(val)
}

// EmitReserveStackBytes implements the common dynamic-stack reservation API.
func (ctx *JITContext) EmitReserveStackBytes(val int32) {
	ctx.EmitSubRSP32(val)
}

// EmitReleaseStackBytes implements the common dynamic-stack release API.
func (ctx *JITContext) EmitReleaseStackBytes(val int32) {
	ctx.EmitAddRSP32(val)
}

// EmitStoreToStack stores a JITValueDesc value to a stack slot at [RSP+disp].
// Uses R11 as scratch for LocImm values.
// Frame slots and this helper both use offsets from the function's stable RSP.
func (ctx *JITContext) EmitStoreToStack(src JITValueDesc, disp int32) {
	ctx.setStackPointer(jitStackRootFrameSP, disp-ctx.DynamicSP, false)
	switch src.Loc {
	case LocImm:
		var word uint64
		switch src.Imm.GetTag() {
		case tagFloat:
			word = math.Float64bits(src.Imm.Float())
		case tagBool:
			if src.Imm.Bool() {
				word = 1
			} else {
				word = 0
			}
		case tagNil:
			word = 0
		default:
			word = uint64(src.Imm.Int())
		}
		ctx.EmitMovRegImm64(RegR11, word)
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp)
	case LocReg:
		ctx.EmitStoreRegMem(src.Reg, RegRSP, disp)
	}
}

// EmitLoadFromStack loads a value from stack slot [RSP+disp] into a register.
// Frame slots and this helper both use offsets from the function's stable RSP.
func (ctx *JITContext) EmitLoadFromStack(dst Reg, disp int32) {
	ctx.EmitMovRegMem(dst, RegRSP, disp)
}

// EmitStoreScmerToStack stores a full Scmer (16 bytes: ptr at disp, aux at disp+8)
// from a LocRegPair or LocImm descriptor to consecutive stack slots [RSP+disp..RSP+disp+15].
// Uses R11 as scratch for LocImm values.
func (ctx *JITContext) EmitStoreScmerToStack(desc JITValueDesc, disp int32) {
	ctx.setStackPointer(jitStackRootFrameSP, disp-ctx.DynamicSP, true)
	switch desc.Loc {
	case LocRegPair:
		ctx.EmitStoreRegMem(desc.Reg, RegRSP, disp)
		ctx.EmitStoreRegMem(desc.Reg2, RegRSP, disp+8)
	case LocStackPair:
		base := RegRSP
		if desc.StackOff < 0 {
			base = RegRBP
		}
		if base == RegRSP && desc.StackOff == disp {
			return
		}
		// Stack allocation and phi placement may give the source and destination
		// overlapping 16-byte ranges. Copy backwards when the destination starts
		// inside the source; all other layouts are safe in forward order.
		if base == RegRSP && disp > desc.StackOff && disp < desc.StackOff+16 {
			ctx.EmitMovRegMem(RegR11, base, desc.StackOff+8)
			ctx.EmitStoreRegMem(RegR11, RegRSP, disp+8)
			ctx.EmitMovRegMem(RegR11, base, desc.StackOff)
			ctx.EmitStoreRegMem(RegR11, RegRSP, disp)
			return
		}
		ctx.EmitMovRegMem(RegR11, base, desc.StackOff)
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp)
		ctx.EmitMovRegMem(RegR11, base, desc.StackOff+8)
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp+8)
	case LocInputPair:
		base := ctx.SliceBase
		if ctx.SliceBaseTracksRSP && int(desc.StackOff) >= ctx.InputArgCount {
			base = ctx.AllocReg()
			ctx.EmitMovRegMem(base, RegRSP, ctx.OriginalArgsOff)
		}
		ctx.EmitMovRegMem(RegR11, base, desc.StackOff*16)
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp)
		ctx.EmitMovRegMem(RegR11, base, desc.StackOff*16+8)
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp+8)
		if base != ctx.SliceBase {
			ctx.FreeReg(base)
		}
	case LocClosurePair:
		value := desc
		ctx.EnsureDesc(&value)
		ctx.EmitStoreRegMem(value.Reg, RegRSP, disp)
		ctx.EmitStoreRegMem(value.Reg2, RegRSP, disp+8)
		ctx.FreeDesc(&value)
	case LocImm:
		// Store ptr word
		ctx.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(desc.Imm.ptr))))
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp)
		// Store aux word
		ctx.EmitMovRegImm64(RegR11, desc.Imm.aux)
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp+8)
	default:
		panic(fmt.Sprintf("jit: EmitStoreScmerToStack: unsupported location %d (type %d)", desc.Loc, desc.Type))
	}
}

// EmitStoreTypedScmerToStack boxes a scalar directly into its final stack
// destination. Phi producers use this instead of allocating a transient
// register pair only to copy both words into the phi slot immediately after.
func (ctx *JITContext) EmitStoreTypedScmerToStack(desc JITValueDesc, typ uint8, disp int32) {
	ctx.setStackPointer(jitStackRootFrameSP, disp-ctx.DynamicSP, true)
	if desc.Loc != LocReg && desc.Loc != LocImm {
		panic("jit: typed Scmer stack store requires a scalar descriptor")
	}

	switch typ {
	case tagInt:
		ctx.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp)
		if desc.Loc == LocReg {
			ctx.EmitStoreRegMem(desc.Reg, RegRSP, disp+8)
		} else {
			ctx.EmitMovRegImm64(RegR11, uint64(desc.Imm.Int()))
			ctx.EmitStoreRegMem(RegR11, RegRSP, disp+8)
		}
	case tagFloat:
		ctx.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp)
		if desc.Loc == LocReg {
			ctx.EmitStoreRegMem(desc.Reg, RegRSP, disp+8)
		} else {
			ctx.EmitMovRegImm64(RegR11, math.Float64bits(desc.Imm.Float()))
			ctx.EmitStoreRegMem(RegR11, RegRSP, disp+8)
		}
	case tagBool:
		ctx.EmitMovRegImm64(RegR11, 0)
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp)
		if desc.Loc == LocReg {
			ctx.emitMovRegReg(RegR11, desc.Reg)
			ctx.emitAndRegImm32(RegR11, 1)
			ctx.EmitShlRegImm8(RegR11, 8)
			ctx.EmitOrRegImm32(RegR11, int32(tagBool))
			ctx.EmitStoreRegMem(RegR11, RegRSP, disp+8)
		} else {
			ctx.EmitMovRegImm64(RegR11, desc.Imm.aux)
			ctx.EmitStoreRegMem(RegR11, RegRSP, disp+8)
		}
	case tagNil:
		ctx.EmitMovRegImm64(RegR11, 0)
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp)
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp+8)
	default:
		panic("jit: unsupported typed Scmer stack store")
	}
}
