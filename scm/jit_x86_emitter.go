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

// Keep the unwind marker above the register-argument spill area used by Go
// callees. MemCP's JIT call bridge supports at most nine ABI words (72 bytes).
const jitGoSpillBytes = uintptr(128)

// jitNextCallback is the runtime/jit Next callback for unwinding through
// JIT frames with standard RBP frame setup (push rbp; mov rbp, rsp).
func jitNextCallback(pc, sp uintptr) (callerPC, callerSP, callerBP uintptr, ok bool) {
	// Every Go callback emitted below places two copies of the distance from
	// its current RSP to the JIT RBP at the top of the stack first. Using a
	// relative distance keeps the marker valid when Go relocates its stack.
	// Two words preserve the Go ABI stack alignment.
	// frame.fp of the Go callee therefore points at this marker regardless of
	// temporary JIT spills or local frame size.
	//   [JIT_RBP + 0] = saved outer RBP
	//   [JIT_RBP + 8] = Go caller's return address
	bpDelta := *(*uintptr)(unsafe.Pointer(sp + jitGoSpillBytes))
	jitRBP := sp + jitGoSpillBytes + 16 + bpDelta
	goRetAddr := *(*uintptr)(unsafe.Pointer(jitRBP + 8))
	return goRetAddr, jitRBP + 16, 0, true
}

// jitCompileProc compiles a Proc body to amd64 machine code or returns nil.
func jitCompileProc(proc *Proc) []byte {
	code, _ := jitCompileProcWithRoots(proc)
	return code
}

// jitCompileProcWithRoots compiles a Proc body to amd64 machine code and
// returns GC roots for pointer constants embedded into immediates.
func jitCompileProcWithRoots(proc *Proc) ([]byte, []unsafe.Pointer) {
	const defaultCodeBufSize = 16 * 1024
	ptr, _ := globalJITPool.Alloc(defaultCodeBufSize)
	buf := &execBuf{ptr: ptr, n: defaultCodeBufSize}
	codeLen, roots, _, _, _ := jitCompileProcToExec(proc, buf)
	if codeLen == 0 {
		return nil, nil
	}
	code := make([]byte, codeLen)
	copy(code, (*[1 << 30]byte)(buf.ptr)[:codeLen:codeLen])
	return code, roots
}

// jitCompileProcToExec compiles a Proc body directly into writable executable memory.
// Returns code length, GC roots, an overflow flag, and whether the call boundary
// must provide a fresh variadic array that becomes the owned list result.
func jitCompileProcToExec(proc *Proc, buf *execBuf) (int, []unsafe.Pointer, bool, bool, []JITHiddenArg) {
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
	return jitCompileExprBodyToExec(proc, body, proc.NumVars, buf)
}

// jitCompileExprBodyToExec compiles a Scheme expression body into a writable
// executable buffer using Declaration.JITEmit callbacks.
func jitCompileExprBodyToExec(proc *Proc, body Scmer, numVars int, buf *execBuf) (codeLen int, roots []unsafe.Pointer, overflow bool, transferInputArgs bool, hiddenArgs []JITHiddenArg) {
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
			transferInputArgs = false
			hiddenArgs = nil
		}
	}()

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
	ctx := &JITContext{
		Ptr:            buf.ptr,
		Start:          buf.ptr,
		End:            unsafe.Add(buf.ptr, buf.n),
		FreeRegs:       freeRegs,
		AllRegs:        freeRegs,
		SliceBase:      RegR12,
		InputArgCount:  inputArgCount,
		LocalSlotCount: numVars,
		Arena:          buf.arena,
	}
	ctx.W = ctx // self-reference for backward-compat ctx.W.Emit calls

	// Unified frame: push rbp; mov rbp, rsp; sub rsp, <fixup>
	// All frame access via [RSP + offset]. MaxBPOffset patched at the end.
	// Epilog: leave; ret.
	ctx.emitByte(0x55)                    // push rbp
	ctx.emitBytes(0x48, 0x89, 0xE5)       // mov rbp, rsp
	frameFixup := ctx.EmitSubRSP32Fixup() // sub rsp, <patched>

	ctx.emitMovRegReg(RegR12, RegRAX) // save incoming args slice
	useInputFrame := proc != nil && proc.NumberedOnly && numVars == inputArgCount
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
		}
		if inputSlots < numVars {
			nilPtr, nilAux := NewNil().RawWords()
			for i := inputSlots; i < numVars; i++ {
				dstOff := int32(i * 16)
				ctx.EmitMovRegImm64(RegR11, uint64(nilPtr))
				ctx.EmitStoreRegMem(RegR11, RegRSP, dstOff)
				ctx.EmitMovRegImm64(RegR11, nilAux)
				ctx.EmitStoreRegMem(RegR11, RegRSP, dstOff+8)
			}
		}
		ctx.emitMovRegReg(RegR12, RegRSP)
		ctx.SliceBaseTracksRSP = true
	}

	// Map lambda parameters to local stack slots so symbol lookup remains correct
	// even when the optimizer did not rewrite body symbols to NthLocalVar.
	if proc != nil {
		var vars map[Symbol]JITValueDesc
		putVar := func(sym Symbol, index int) {
			if vars == nil {
				vars = make(map[Symbol]JITValueDesc, inputArgCount)
			}
			desc := JITValueDesc{
				Loc: LocInputPair, Type: JITTypeUnknown, StackOff: int32(index),
			}
			if !useInputFrame && index < numVars {
				desc.Loc = LocStackPair
				desc.StackOff = int32(index * 16)
			}
			vars[sym] = desc
		}
		switch proc.Params.GetTag() {
		case tagSlice:
			params := proc.Params.Slice()
			for i := 0; i < len(params) && (inputArgCount < 0 || i < inputArgCount); i++ {
				if params[i].GetTag() != tagSymbol {
					continue
				}
				putVar(params[i].Symbol(), i)
			}
		case tagSymbol:
			if inputArgCount > 0 {
				putVar(proc.Params.Symbol(), 0)
			}
		}
		if len(vars) > 0 {
			ctx.Env = &JITEnv{Vars: vars}
		}
	}

	// Compile body, place result into RAX+RBX (Scmer return registers)
	result := JITValueDesc{Loc: LocRegPair, Reg: RegRAX, Reg2: RegRBX}
	desc := jitCompileExpr(ctx, body, RegR12, result)

	// If result came back as LocImm, materialize into RAX+RBX
	if desc.Loc == LocImm {
		switch desc.Imm.GetTag() {
		case tagBool:
			ctx.EmitMakeBool(result, desc)
		case tagInt:
			ctx.EmitMakeInt(result, desc)
		case tagFloat:
			ctx.EmitMakeFloat(result, desc)
		case tagNil:
			ctx.EmitMakeNil(result)
		default:
			return 0, nil, false, false, nil
		}
		// fall through to epilog
	} else {
		ctx.EnsureDesc(&desc)
		switch desc.Loc {
		case LocRegPair:
			if desc.Reg != RegRAX {
				ctx.emitMovRegReg(RegRAX, desc.Reg)
			}
			if desc.Reg2 != RegRBX {
				ctx.emitMovRegReg(RegRBX, desc.Reg2)
			}
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
				return 0, nil, false, false, nil
			}
		default:
			return 0, nil, false, false, nil
		}
	}
	// Unified epilog: patch SUB RSP with max frame size, then leave; ret.
	frameSize := ctx.MaxBPOffset + ctx.MaxSpillOffset
	frameSize = (frameSize + 15) &^ 15
	ctx.PatchInt32(frameFixup, frameSize)
	ctx.emitByte(0xC9) // leave
	ctx.emitByte(0xC3) // ret

	ctx.ResolveFixupsFinal()
	codeLen = int(uintptr(ctx.Ptr) - uintptr(ctx.Start))
	return codeLen, ctx.ConstRoots, false, ctx.TransferInputArgs, ctx.HiddenArgs
}

func jitEnsureResultPair(ctx *JITContext, result JITValueDesc) JITValueDesc {
	if result.Loc == LocRegPair {
		return JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: result.Reg, Reg2: result.Reg2}
	}
	return JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
}

func jitPlaceIntoPair(ctx *JITContext, src *JITValueDesc, target JITValueDesc) JITValueDesc {
	if target.Loc != LocRegPair {
		panic("jit: jitPlaceIntoPair requires LocRegPair target")
	}
	// Placement changes representation, not the proven Scheme type. Keeping the
	// fact lets downstream generated SSA eliminate checks and write barriers.
	target.Type = src.Type
	target.NoHeapPointer = src.NoHeapPointer
	// Keep descriptor location in sync with spill metadata before we read Reg/Reg2.
	if src.Loc != LocImm {
		ctx.EnsureDesc(src)
	}
	switch src.Loc {
	case LocImm:
		switch src.Imm.GetTag() {
		case tagBool:
			ctx.EmitMakeBool(target, *src)
		case tagInt:
			ctx.EmitMakeInt(target, *src)
		case tagFloat:
			ctx.EmitMakeFloat(target, *src)
		case tagNil:
			ctx.EmitMakeNil(target)
		default:
			ptr, aux := src.Imm.RawWords()
			ctx.EmitMovRegImm64(target.Reg, uint64(ptr))
			ctx.EmitMovRegImm64(target.Reg2, aux)
		}
		return target
	case LocStack, LocStackPair:
		ctx.EnsureDesc(src)
		return jitPlaceIntoPair(ctx, src, target)
	case LocRegPair:
		if src.Reg != target.Reg {
			ctx.emitMovRegReg(target.Reg, src.Reg)
		}
		if src.Reg2 != target.Reg2 {
			ctx.emitMovRegReg(target.Reg2, src.Reg2)
		}
		if src.Reg != target.Reg && src.Reg2 != target.Reg2 {
			ctx.FreeDesc(src)
		}
		return target
	case LocReg:
		switch src.Type {
		case tagBool:
			ctx.EmitMakeBool(target, *src)
		case tagInt:
			ctx.EmitMakeInt(target, *src)
		case tagFloat:
			ctx.EmitMakeFloat(target, *src)
		default:
			panic("jit: cannot materialize LocReg with unknown type into Scmer pair")
		}
		ctx.FreeDesc(src)
		return target
	default:
		panic("jit: unsupported source location for pair materialization")
	}
}

func jitList0() Scmer                             { return List() }
func jitList1(a Scmer) Scmer                      { return List(a) }
func jitList2(a, b Scmer) Scmer                   { return List(a, b) }
func jitList3(a, b, c Scmer) Scmer                { return List(a, b, c) }
func jitList4(a, b, c, d Scmer) Scmer             { return List(a, b, c, d) }
func jitList5(a, b, c, d, e Scmer) Scmer          { return List(a, b, c, d, e) }
func jitList6(a, b, c, d, e, f Scmer) Scmer       { return List(a, b, c, d, e, f) }
func jitList7(a, b, c, d, e, f, g Scmer) Scmer    { return List(a, b, c, d, e, f, g) }
func jitList8(a, b, c, d, e, f, g, h Scmer) Scmer { return List(a, b, c, d, e, f, g, h) }

func jitMakeScmerSlice(length, capacity int) []Scmer {
	return make([]Scmer, length, capacity)
}

func jitStoreScmerAt(address *Scmer, value Scmer) {
	*address = value
}

func jitInvokeCallback1(callback, arg0 Scmer) Scmer {
	return callback.Func()(arg0)
}

func jitInvokeCallback2(callback, arg0, arg1 Scmer) Scmer {
	return callback.Func()(arg0, arg1)
}

func jitInvokeCallback3(callback, arg0, arg1, arg2 Scmer) Scmer {
	return callback.Func()(arg0, arg1, arg2)
}

// EmitSliceElementAddress lowers the address part shared by SSA slice loads and
// stores.
func (ctx *JITContext) EmitSliceElementAddress(slice, index *JITValueDesc, elementSize int32) JITValueDesc {
	ctx.EnsureDesc(slice)
	if slice.Loc != LocRegTriple {
		panic("jit: slice element address requires a Go slice header")
	}
	ctx.ProtectReg(slice.Reg)
	ctx.ProtectReg(slice.Reg2)
	ctx.ProtectReg(slice.Reg3)
	ctx.EnsureDesc(index)
	excluded := []Reg{slice.Reg, slice.Reg2, slice.Reg3}
	if index.Loc == LocReg {
		excluded = append(excluded, index.Reg)
	}
	address := ctx.AllocRegExcept(excluded...)
	if index.Loc == LocImm {
		ctx.EmitMovRegImm64(address, uint64(index.Imm.Int())*uint64(elementSize))
	} else if index.Loc == LocReg {
		ctx.EmitMovRegReg(address, index.Reg)
		switch elementSize {
		case 8:
			ctx.EmitShlRegImm8(address, 3)
		case 16:
			ctx.EmitShlRegImm8(address, 4)
		default:
			factor := ctx.AllocRegExcept(append(excluded, address)...)
			ctx.EmitMovRegImm64(factor, uint64(elementSize))
			ctx.EmitImulInt64(address, factor)
			ctx.FreeReg(factor)
		}
	} else {
		panic("jit: slice element index requires an integer descriptor")
	}
	ctx.EmitAddInt64(address, slice.Reg)
	ctx.UnprotectReg(slice.Reg)
	ctx.UnprotectReg(slice.Reg2)
	ctx.UnprotectReg(slice.Reg3)
	result := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: address}
	ctx.BindReg(address, &result)
	return result
}

// EmitStoreScmerAt stores a Scmer through an address produced by
// EmitSliceElementAddress. Values whose descriptor proves that their ptr word
// is nil or a static runtime sentinel need no GC write barrier and are emitted
// as two plain stores. All other values retain Go's authoritative write barrier
// through the generic trampoline.
func (ctx *JITContext) EmitStoreScmerAt(address, value *JITValueDesc) {
	ctx.EnsureDesc(address)
	if address.Loc != LocReg {
		panic("jit: Scmer store requires an address register")
	}

	switch value.Type {
	case tagNil, tagFloat, tagInt, tagBool, tagDate, tagNthLocalVar:
		value.NoHeapPointer = true
	}
	if value.NoHeapPointer {
		ctx.ProtectReg(address.Reg)
		ctx.EnsureDesc(value)
		ctx.UnprotectReg(address.Reg)
		if value.Loc == LocImm {
			ctx.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(value.Imm.ptr))))
			ctx.EmitStoreRegMem(RegR11, address.Reg, 0)
			ctx.EmitMovRegImm64(RegR11, value.Imm.aux)
			ctx.EmitStoreRegMem(RegR11, address.Reg, 8)
			return
		}
		if value.Loc == LocRegPair {
			ctx.EmitStoreRegMem(value.Reg, address.Reg, 0)
			ctx.EmitStoreRegMem(value.Reg2, address.Reg, 8)
			return
		}
	}

	ctx.EnsureDesc(value)
	ctx.EmitGoCallVoid(GoFuncAddr(jitStoreScmerAt), []JITValueDesc{*address, *value})
}

func jitReturnHasNoHeapPointer(td *TypeDescriptor) bool {
	if td == nil {
		return false
	}
	switch td.Kind {
	case "bool", "date", "int", "int|nil", "nil", "number", "number|nil":
		return true
	default:
		return false
	}
}

var jitMatchEmptyListStorage [1]Scmer

func jitMatchAsGoSlice(value Scmer) []Scmer {
	list, ok := scmerAsSlice(value)
	if !ok {
		return nil
	}
	// A non-nil pointer distinguishes a successfully normalized empty list from
	// a non-list without adding a fourth ABI result word to the emitted call.
	if len(list) == 0 && unsafe.SliceData(list) == nil {
		return jitMatchEmptyListStorage[:0]
	}
	return list
}

func jitMatchSymbolLiteralWords(valuePtr *byte, valueAux uint64, expectedPtr *byte, expectedAux uint64) bool {
	value := Scmer{ptr: valuePtr, aux: valueAux}
	expected := Scmer{ptr: expectedPtr, aux: expectedAux}
	actualName, actualOK := scmerSymbolName(value)
	expectedName, expectedOK := scmerSymbolName(expected)
	return actualOK && expectedOK && actualName == expectedName
}

func jitMatchEqualWords(valuePtr *byte, valueAux uint64, expectedPtr *byte, expectedAux uint64) bool {
	return Equal(Scmer{ptr: valuePtr, aux: valueAux}, Scmer{ptr: expectedPtr, aux: expectedAux})
}

func jitMatchIsString(value Scmer) bool {
	_, ok := scmerAsString(value)
	return ok
}

type jitMatchOutcome struct {
	possible bool
	always   bool
}

func jitMatchBranchEnv(outer *JITEnv) *JITEnv {
	env := &JITEnv{Outer: outer}
	if outer != nil && len(outer.Numbered) > 0 {
		env.Numbered = append([]JITValueDesc(nil), outer.Numbered...)
	}
	return env
}

func jitMatchStableValue(ctx *JITContext, value JITValueDesc) JITValueDesc {
	if value.Loc == LocImm || value.Loc == LocStackPair || value.Loc == LocInputPair || value.Loc == LocVirtualSlice {
		return value
	}
	src := value
	ctx.EnsureDesc(&src)
	if src.Loc == LocReg {
		ptrReg := ctx.AllocRegExcept(src.Reg)
		target := JITValueDesc{Loc: LocRegPair, Type: src.Type, Reg: ptrReg, Reg2: ctx.AllocRegExcept(src.Reg, ptrReg)}
		src = jitPlaceIntoPair(ctx, &src, target)
	}
	if src.Loc != LocRegPair {
		panic("jit: match value is not a Scmer pair")
	}
	off := ctx.AllocStack(16)
	ctx.EmitStoreScmerToStack(src, off)
	ctx.FreeDesc(&src)
	return JITValueDesc{
		Loc: LocStackPair, Type: value.Type, StackOff: off,
		KnownSliceLen: value.KnownSliceLen, KnownSliceCap: value.KnownSliceCap,
		SliceSizeKnown: value.SliceSizeKnown, NoHeapPointer: value.NoHeapPointer,
	}
}

func jitMatchBindValue(ctx *JITContext, env *JITEnv, pattern Scmer, value JITValueDesc) {
	if pattern.IsSymbol() && pattern.SymbolEquals("_") {
		return
	}
	bound := value
	if value.Loc != LocImm {
		bound = jitMatchStableValue(ctx, value)
	}
	if pattern.IsNthLocalVar() {
		idx := int(pattern.NthLocalVar())
		if idx >= len(env.Numbered) {
			numbered := make([]JITValueDesc, idx+1)
			copy(numbered, env.Numbered)
			env.Numbered = numbered
		}
		env.Numbered[idx] = bound
		return
	}
	if !pattern.IsSymbol() {
		panic("jit: invalid match binding")
	}
	if env.Vars == nil {
		env.Vars = make(map[Symbol]JITValueDesc)
	}
	env.Vars[pattern.Symbol()] = bound
}

func jitMatchMaterializeImm(ctx *JITContext, value Scmer) JITValueDesc {
	ptrReg := ctx.AllocReg()
	auxReg := ctx.AllocRegExcept(ptrReg)
	target := JITValueDesc{Loc: LocRegPair, Type: value.GetTag(), Reg: ptrReg, Reg2: auxReg}
	source := JITValueDesc{Loc: LocImm, Type: value.GetTag(), Imm: value}
	return jitPlaceIntoPair(ctx, &source, target)
}

func jitEmitMatchBool(ctx *JITContext, condition JITValueDesc, failLabel uint8) jitMatchOutcome {
	if condition.Loc == LocImm {
		return jitMatchOutcome{possible: condition.Imm.Bool(), always: condition.Imm.Bool()}
	}
	ctx.EnsureDesc(&condition)
	if condition.Loc != LocReg {
		panic("jit: match condition is not scalar")
	}
	// Go's internal ABI only defines the low byte of a bool result. Clear stale
	// upper bits before using it as a branch condition.
	ctx.EmitAndRegImm32(condition.Reg, 1)
	ctx.EmitCmpRegImm32(condition.Reg, 0)
	ctx.EmitJcc(CcE, failLabel)
	ctx.FreeDesc(&condition)
	return jitMatchOutcome{possible: true, always: false}
}

func jitEmitMatchTag(ctx *JITContext, value *JITValueDesc, tag uint8, failLabel uint8) jitMatchOutcome {
	condition := ctx.EmitTagEqualsBorrowed(value, tag, JITValueDesc{Loc: LocAny})
	return jitEmitMatchBool(ctx, condition, failLabel)
}

func jitMatchLoadElement(ctx *JITContext, slice *JITValueDesc, index int) JITValueDesc {
	ctx.EnsureDesc(slice)
	if slice.Loc != LocRegTriple {
		panic("jit: match list requires a Go slice header")
	}
	ctx.ProtectReg(slice.Reg)
	ctx.ProtectReg(slice.Reg2)
	ctx.ProtectReg(slice.Reg3)
	ptrReg := ctx.AllocRegExcept(slice.Reg, slice.Reg2, slice.Reg3)
	auxReg := ctx.AllocRegExcept(slice.Reg, slice.Reg2, slice.Reg3, ptrReg)
	ctx.UnprotectReg(slice.Reg3)
	ctx.UnprotectReg(slice.Reg2)
	ctx.UnprotectReg(slice.Reg)
	ctx.EmitMovRegMem(ptrReg, slice.Reg, int32(index*16))
	ctx.EmitMovRegMem(auxReg, slice.Reg, int32(index*16+8))
	result := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ptrReg, Reg2: auxReg}
	ctx.BindReg(ptrReg, &result)
	ctx.BindReg(auxReg, &result)
	return result
}

func jitMatchSliceTail(ctx *JITContext, slice *JITValueDesc) JITValueDesc {
	ctx.EnsureDesc(slice)
	if slice.Loc != LocRegTriple {
		panic("jit: match cons requires a Go slice header")
	}
	ctx.ProtectReg(slice.Reg)
	ctx.ProtectReg(slice.Reg2)
	ctx.ProtectReg(slice.Reg3)
	ptrReg := ctx.AllocRegExcept(slice.Reg, slice.Reg2, slice.Reg3)
	auxReg := ctx.AllocRegExcept(slice.Reg, slice.Reg2, slice.Reg3, ptrReg)
	tmpReg := ctx.AllocRegExcept(slice.Reg, slice.Reg2, slice.Reg3, ptrReg, auxReg)
	ctx.UnprotectReg(slice.Reg3)
	ctx.UnprotectReg(slice.Reg2)
	ctx.UnprotectReg(slice.Reg)
	ctx.EmitMovRegReg(ptrReg, slice.Reg)
	ctx.EmitAddRegImm32(ptrReg, 16)
	ctx.EmitMovRegReg(auxReg, slice.Reg2)
	ctx.EmitSubRegImm32(auxReg, 1)
	ctx.EmitShlRegImm8(auxReg, sliceCapBits)
	ctx.EmitMovRegReg(tmpReg, slice.Reg3)
	ctx.EmitSubRegImm32(tmpReg, 1)
	ctx.EmitOrInt64(auxReg, tmpReg)
	ctx.EmitShlRegImm8(auxReg, 8)
	ctx.EmitOrRegImm32(auxReg, int32(tagSlice))
	ctx.FreeReg(tmpReg)
	result := JITValueDesc{Loc: LocRegPair, Type: tagSlice, Reg: ptrReg, Reg2: auxReg}
	if slice.SliceSizeKnown {
		result.KnownSliceLen = slice.KnownSliceLen - 1
		result.KnownSliceCap = slice.KnownSliceCap - 1
		result.SliceSizeKnown = true
	}
	ctx.BindReg(ptrReg, &result)
	ctx.BindReg(auxReg, &result)
	return result
}

func jitMatchListValue(ctx *JITContext, value JITValueDesc, failLabel uint8) (JITValueDesc, JITValueDesc, jitMatchOutcome) {
	if value.Loc == LocImm {
		list, ok := scmerAsSlice(value.Imm)
		if !ok {
			return JITValueDesc{}, JITValueDesc{}, jitMatchOutcome{}
		}
		normalized := JITValueDesc{
			Loc: LocImm, Type: tagSlice, Imm: NewSlice(list),
			KnownSliceLen: int32(len(list)), KnownSliceCap: int32(cap(list)), SliceSizeKnown: true,
		}
		return normalized, JITValueDesc{}, jitMatchOutcome{possible: true, always: true}
	}
	if value.Type != JITTypeUnknown && value.Type != tagSlice && value.Type != tagFastDict && value.Type != tagSourceInfo {
		return JITValueDesc{}, JITValueDesc{}, jitMatchOutcome{}
	}

	normalized := value
	typeKnown := value.Type == tagSlice
	if !typeKnown {
		header := ctx.EmitGoCallScalar(GoFuncAddr(jitMatchAsGoSlice), []JITValueDesc{value}, 3)
		ctx.BindReg(header.Reg, &header)
		ctx.BindReg(header.Reg2, &header)
		ctx.BindReg(header.Reg3, &header)
		ctx.EmitCmpRegImm32(header.Reg, 0)
		ctx.EmitJcc(CcE, failLabel)
		normalized = ctx.EmitNewSliceFromGoSlice(&header)
	}
	normalized.KnownSliceLen = value.KnownSliceLen
	normalized.KnownSliceCap = value.KnownSliceCap
	normalized.SliceSizeKnown = value.SliceSizeKnown
	normalized = jitMatchStableValue(ctx, normalized)
	sliceValue := normalized
	header := jitKnownSliceHeader(ctx, &sliceValue)
	ctx.FreeDesc(&sliceValue)
	header.KnownSliceLen = value.KnownSliceLen
	header.KnownSliceCap = value.KnownSliceCap
	header.SliceSizeKnown = value.SliceSizeKnown
	return normalized, header, jitMatchOutcome{possible: true, always: typeKnown}
}

func jitMatchLiteral(ctx *JITContext, value JITValueDesc, pattern Scmer, failLabel uint8) jitMatchOutcome {
	if value.Loc == LocImm {
		matched := Equal(value.Imm, pattern)
		return jitMatchOutcome{possible: matched, always: matched}
	}
	expected := jitMatchMaterializeImm(ctx, pattern)
	condition := ctx.EmitGoCallScalar(GoFuncAddr(jitMatchEqualWords), []JITValueDesc{value, expected}, 1)
	ctx.FreeDesc(&expected)
	return jitEmitMatchBool(ctx, condition, failLabel)
}

func jitMatchDynamicValue(ctx *JITContext, value, expected JITValueDesc, failLabel uint8) jitMatchOutcome {
	if value.Loc == LocImm && expected.Loc == LocImm {
		matched := Equal(value.Imm, expected.Imm)
		return jitMatchOutcome{possible: matched, always: matched}
	}
	materialize := func(desc JITValueDesc) JITValueDesc {
		if desc.Loc == LocImm {
			return jitMatchMaterializeImm(ctx, desc.Imm)
		}
		if desc.Loc == LocReg {
			ptrReg := ctx.AllocRegExcept(desc.Reg)
			target := JITValueDesc{Loc: LocRegPair, Type: desc.Type, Reg: ptrReg, Reg2: ctx.AllocRegExcept(desc.Reg, ptrReg)}
			return jitPlaceIntoPair(ctx, &desc, target)
		}
		return desc
	}
	actualArg := materialize(value)
	expectedArg := materialize(expected)
	condition := ctx.EmitGoCallScalar(GoFuncAddr(jitMatchEqualWords), []JITValueDesc{actualArg, expectedArg}, 1)
	if actualArg.Loc == LocRegPair && value.Loc != LocRegPair {
		ctx.FreeDesc(&actualArg)
	}
	if expectedArg.Loc == LocRegPair && expected.Loc != LocRegPair {
		ctx.FreeDesc(&expectedArg)
	}
	return jitEmitMatchBool(ctx, condition, failLabel)
}

func jitMatchBoolLiteral(ctx *JITContext, value JITValueDesc, expected bool, failLabel uint8) jitMatchOutcome {
	if value.Loc == LocImm {
		matched := value.Imm.IsBool() && value.Imm.Bool() == expected
		return jitMatchOutcome{possible: matched, always: matched}
	}
	if value.Type != JITTypeUnknown && value.Type != tagBool {
		return jitMatchOutcome{}
	}
	typeAlways := value.Type == tagBool
	if !typeAlways {
		jitEmitMatchTag(ctx, &value, tagBool, failLabel)
	}
	pair := value
	ctx.EnsureDesc(&pair)
	if pair.Loc != LocRegPair {
		panic("jit: bool match requires a Scmer pair")
	}
	expectedAux := makeAux(tagBool, 0)
	if expected {
		expectedAux = makeAux(tagBool, 1)
	}
	ctx.EmitCmpRegImm32(pair.Reg2, int32(expectedAux))
	ctx.EmitJcc(CcNE, failLabel)
	ctx.FreeDesc(&pair)
	return jitMatchOutcome{possible: true, always: false}
}

func jitMatchSymbol(ctx *JITContext, value JITValueDesc, expected Scmer, failLabel uint8) jitMatchOutcome {
	if value.Loc == LocImm {
		matched := func() bool {
			actualName, actualOK := scmerSymbolName(value.Imm)
			expectedName, expectedOK := scmerSymbolName(expected)
			return actualOK && expectedOK && actualName == expectedName
		}()
		return jitMatchOutcome{possible: matched, always: matched}
	}
	if value.Type != JITTypeUnknown && value.Type != tagSymbol && value.Type != tagSourceInfo {
		return jitMatchOutcome{}
	}
	expectedValue := jitMatchMaterializeImm(ctx, expected)
	condition := ctx.EmitGoCallScalar(GoFuncAddr(jitMatchSymbolLiteralWords), []JITValueDesc{value, expectedValue}, 1)
	ctx.FreeDesc(&expectedValue)
	return jitEmitMatchBool(ctx, condition, failLabel)
}

func jitMatchNumber(ctx *JITContext, value JITValueDesc, failLabel uint8) jitMatchOutcome {
	if value.Loc == LocImm {
		matched := value.Imm.IsInt() || value.Imm.IsFloat()
		return jitMatchOutcome{possible: matched, always: matched}
	}
	if value.Type != JITTypeUnknown {
		matched := value.Type == tagInt || value.Type == tagFloat
		return jitMatchOutcome{possible: matched, always: matched}
	}
	tmp := value
	ctx.EnsureDesc(&tmp)
	if tmp.Loc != LocRegPair {
		panic("jit: number match requires a Scmer pair")
	}
	tagReg := ctx.AllocRegExcept(tmp.Reg, tmp.Reg2)
	ctx.emitGetTagRegs(tagReg, tmp.Reg, tmp.Reg2)
	accepted := ctx.ReserveLabel()
	ctx.EmitCmpRegImm8(tagReg, tagInt)
	ctx.EmitJcc(CcE, accepted)
	ctx.EmitCmpRegImm8(tagReg, tagFloat)
	ctx.EmitJcc(CcNE, failLabel)
	ctx.MarkLabel(accepted)
	ctx.FreeReg(tagReg)
	ctx.FreeDesc(&tmp)
	return jitMatchOutcome{possible: true, always: false}
}

func jitMatchString(ctx *JITContext, value JITValueDesc, failLabel uint8) jitMatchOutcome {
	if value.Loc == LocImm {
		_, matched := scmerAsString(value.Imm)
		return jitMatchOutcome{possible: matched, always: matched}
	}
	if value.Type == tagString {
		return jitMatchOutcome{possible: true, always: true}
	}
	if value.Type != JITTypeUnknown && value.Type != tagAny && value.Type != tagSourceInfo {
		return jitMatchOutcome{}
	}
	condition := ctx.EmitGoCallScalar(GoFuncAddr(jitMatchIsString), []JITValueDesc{value}, 1)
	return jitEmitMatchBool(ctx, condition, failLabel)
}

func jitMatchFixedList(ctx *JITContext, value JITValueDesc, patterns []Scmer, env *JITEnv, failLabel uint8) jitMatchOutcome {
	if value.Loc == LocVirtualSlice {
		if len(value.Virtual) != len(patterns) {
			return jitMatchOutcome{}
		}
		allAlways := true
		for i, pattern := range patterns {
			outcome := jitCompileMatchPattern(ctx, value.Virtual[i], pattern, env, failLabel)
			if !outcome.possible {
				return jitMatchOutcome{}
			}
			allAlways = allAlways && outcome.always
		}
		return jitMatchOutcome{possible: true, always: allAlways}
	}
	normalized, header, listOutcome := jitMatchListValue(ctx, value, failLabel)
	if !listOutcome.possible {
		return jitMatchOutcome{}
	}
	if normalized.Loc == LocImm {
		list := normalized.Imm.Slice()
		if len(list) != len(patterns) {
			return jitMatchOutcome{}
		}
		for i, pattern := range patterns {
			outcome := jitCompileMatchPattern(ctx, JITValueDesc{Loc: LocImm, Type: list[i].GetTag(), Imm: list[i]}, pattern, env, failLabel)
			if !outcome.possible {
				return jitMatchOutcome{}
			}
		}
		return jitMatchOutcome{possible: true, always: true}
	}
	if header.SliceSizeKnown {
		if int(header.KnownSliceLen) != len(patterns) {
			ctx.FreeDesc(&header)
			ctx.FreeDesc(&normalized)
			return jitMatchOutcome{}
		}
	} else {
		ctx.EmitCmpRegImm32(header.Reg2, int32(len(patterns)))
		ctx.EmitJcc(CcNE, failLabel)
		listOutcome.always = false
	}
	ctx.FreeDesc(&header)
	allAlways := listOutcome.always
	for i, pattern := range patterns {
		sliceValue := normalized
		header = jitKnownSliceHeader(ctx, &sliceValue)
		ctx.FreeDesc(&sliceValue)
		element := jitMatchLoadElement(ctx, &header, i)
		ctx.FreeDesc(&header)
		outcome := jitCompileMatchPattern(ctx, element, pattern, env, failLabel)
		ctx.FreeDesc(&element)
		if !outcome.possible {
			ctx.FreeDesc(&normalized)
			return jitMatchOutcome{}
		}
		allAlways = allAlways && outcome.always
	}
	ctx.FreeDesc(&normalized)
	return jitMatchOutcome{possible: true, always: allAlways}
}

func jitMatchCons(ctx *JITContext, value JITValueDesc, patterns []Scmer, env *JITEnv, failLabel uint8) jitMatchOutcome {
	if len(patterns) != 2 {
		panic("jit: cons match expects head and tail patterns")
	}
	if value.Loc == LocVirtualSlice {
		if len(value.Virtual) == 0 {
			return jitMatchOutcome{}
		}
		tail := JITValueDesc{
			Loc: LocVirtualSlice, Type: tagSlice, Virtual: value.Virtual[1:],
			KnownSliceLen: int32(len(value.Virtual) - 1), KnownSliceCap: int32(len(value.Virtual) - 1), SliceSizeKnown: true,
		}
		first := jitCompileMatchPattern(ctx, value.Virtual[0], patterns[0], env, failLabel)
		second := jitCompileMatchPattern(ctx, tail, patterns[1], env, failLabel)
		return jitMatchOutcome{possible: first.possible && second.possible, always: first.always && second.always}
	}
	normalized, header, listOutcome := jitMatchListValue(ctx, value, failLabel)
	if !listOutcome.possible {
		return jitMatchOutcome{}
	}
	if normalized.Loc == LocImm {
		list := normalized.Imm.Slice()
		if len(list) == 0 {
			return jitMatchOutcome{}
		}
		head := JITValueDesc{Loc: LocImm, Type: list[0].GetTag(), Imm: list[0]}
		tailValue := NewSlice(list[1:])
		tail := JITValueDesc{Loc: LocImm, Type: tagSlice, Imm: tailValue, KnownSliceLen: int32(len(list) - 1), KnownSliceCap: int32(cap(list) - 1), SliceSizeKnown: true}
		first := jitCompileMatchPattern(ctx, head, patterns[0], env, failLabel)
		second := jitCompileMatchPattern(ctx, tail, patterns[1], env, failLabel)
		return jitMatchOutcome{possible: first.possible && second.possible, always: first.always && second.always}
	}
	if header.SliceSizeKnown {
		if header.KnownSliceLen == 0 {
			ctx.FreeDesc(&header)
			ctx.FreeDesc(&normalized)
			return jitMatchOutcome{}
		}
	} else {
		ctx.EmitCmpRegImm32(header.Reg2, 0)
		ctx.EmitJcc(CcE, failLabel)
		listOutcome.always = false
	}
	head := jitMatchLoadElement(ctx, &header, 0)
	tail := jitMatchSliceTail(ctx, &header)
	first := jitCompileMatchPattern(ctx, head, patterns[0], env, failLabel)
	second := jitCompileMatchPattern(ctx, tail, patterns[1], env, failLabel)
	ctx.FreeDesc(&head)
	ctx.FreeDesc(&tail)
	ctx.FreeDesc(&header)
	ctx.FreeDesc(&normalized)
	return jitMatchOutcome{
		possible: first.possible && second.possible,
		always:   listOutcome.always && first.always && second.always,
	}
}

func jitCompileMatchPattern(ctx *JITContext, value JITValueDesc, pattern Scmer, env *JITEnv, failLabel uint8) jitMatchOutcome {
	for pattern.IsSourceInfo() {
		pattern = pattern.SourceInfo().value
	}
	switch pattern.GetTag() {
	case tagInt, tagFloat, tagString:
		return jitMatchLiteral(ctx, value, pattern, failLabel)
	case tagNthLocalVar:
		if int(pattern.NthLocalVar()) >= len(env.Numbered) {
			return jitMatchLiteral(ctx, value, pattern, failLabel)
		}
		jitMatchBindValue(ctx, env, pattern, value)
		return jitMatchOutcome{possible: true, always: true}
	case tagSymbol:
		switch pattern.String() {
		case "nil":
			if value.Loc == LocImm {
				matched := value.Imm.IsNil()
				return jitMatchOutcome{possible: matched, always: matched}
			}
			if value.Type != JITTypeUnknown {
				matched := value.Type == tagNil
				return jitMatchOutcome{possible: matched, always: matched}
			}
			return jitEmitMatchTag(ctx, &value, tagNil, failLabel)
		case "true":
			return jitMatchBoolLiteral(ctx, value, true, failLabel)
		case "false":
			return jitMatchBoolLiteral(ctx, value, false, failLabel)
		default:
			jitMatchBindValue(ctx, env, pattern, value)
			return jitMatchOutcome{possible: true, always: true}
		}
	case tagSlice:
		p := pattern.Slice()
		if len(p) == 0 {
			panic("jit: empty match pattern")
		}
		if _, direct := scmerSymbolName(p[0]); !direct {
			if nested, ok := scmerAsSlice(p[0]); ok && len(nested) > 0 {
				if head, ok := scmerSymbolName(nested[0]); ok && (head == "symbol" || head == "quote") {
					return jitMatchFixedList(ctx, value, p, env, failLabel)
				}
			}
			panic("jit: unsupported nested match pattern")
		}
		name, _ := scmerSymbolName(p[0])
		switch name {
		case "eval":
			if len(p) != 2 {
				panic("jit: malformed eval match pattern")
			}
			outerEnv := ctx.Env
			ctx.Env = env
			expected := jitCompileExpr(ctx, p[1], ctx.SliceBase, JITValueDesc{Loc: LocAny})
			ctx.Env = outerEnv
			return jitMatchDynamicValue(ctx, value, expected, failLabel)
		case "var":
			if len(p) != 2 || !p[1].IsInt() {
				panic("jit: malformed var match pattern")
			}
			jitMatchBindValue(ctx, env, NewNthLocalVar(NthLocalVar(p[1].Int())), value)
			return jitMatchOutcome{possible: true, always: true}
		case "list":
			return jitMatchFixedList(ctx, value, p[1:], env, failLabel)
		case "quote", "symbol":
			if len(p) != 2 {
				panic("jit: malformed symbol match pattern")
			}
			return jitMatchSymbol(ctx, value, p[1], failLabel)
		case "string?":
			if len(p) != 2 {
				panic("jit: malformed string match pattern")
			}
			outcome := jitMatchString(ctx, value, failLabel)
			if !outcome.possible {
				return outcome
			}
			inner := jitCompileMatchPattern(ctx, value, p[1], env, failLabel)
			return jitMatchOutcome{possible: inner.possible, always: outcome.always && inner.always}
		case "number?":
			if len(p) != 2 {
				panic("jit: malformed number match pattern")
			}
			outcome := jitMatchNumber(ctx, value, failLabel)
			if !outcome.possible {
				return outcome
			}
			inner := jitCompileMatchPattern(ctx, value, p[1], env, failLabel)
			return jitMatchOutcome{possible: inner.possible, always: outcome.always && inner.always}
		case "list?":
			if len(p) != 2 {
				panic("jit: malformed list match pattern")
			}
			normalized, header, outcome := jitMatchListValue(ctx, value, failLabel)
			if !outcome.possible {
				return outcome
			}
			ctx.FreeDesc(&header)
			inner := jitCompileMatchPattern(ctx, normalized, p[1], env, failLabel)
			ctx.FreeDesc(&normalized)
			return jitMatchOutcome{possible: inner.possible, always: outcome.always && inner.always}
		case "cons":
			return jitMatchCons(ctx, value, p[1:], env, failLabel)
		default:
			panic("jit: unsupported match pattern " + name)
		}
	default:
		panic("jit: unsupported match pattern type")
	}
}

func jitCompileMatch(ctx *JITContext, list []Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	if len(list) < 2 {
		panic("jit: match expects a value")
	}
	valueExpr := list[1]
	for valueExpr.IsSourceInfo() {
		valueExpr = valueExpr.SourceInfo().value
	}
	var value JITValueDesc
	if items, ok := scmerAsSlice(valueExpr); ok && len(items) > 0 && items[0].SymbolEquals("list") {
		values := make([]JITValueDesc, len(items)-1)
		for i := 1; i < len(items); i++ {
			item := jitCompileExpr(ctx, items[i], sliceBase, JITValueDesc{Loc: LocAny})
			values[i-1] = jitMatchStableValue(ctx, item)
		}
		value = JITValueDesc{
			Loc: LocVirtualSlice, Type: tagSlice, Virtual: values,
			KnownSliceLen: int32(len(values)), KnownSliceCap: int32(len(values)), SliceSizeKnown: true,
		}
	} else {
		value = jitCompileExpr(ctx, valueExpr, sliceBase, JITValueDesc{Loc: LocAny})
	}
	value = jitMatchStableValue(ctx, value)
	target := jitEnsureResultPair(ctx, result)
	endLabel := ctx.ReserveLabel()
	hasBranch := false
	terminated := false
	baseEnv := ctx.Env
	i := 2
	for i+1 < len(list) {
		nextLabel := ctx.ReserveLabel()
		branchEnv := jitMatchBranchEnv(baseEnv)
		outcome := jitCompileMatchPattern(ctx, value, list[i], branchEnv, nextLabel)
		if outcome.possible {
			ctx.Env = branchEnv
			branchValue := jitCompileExpr(ctx, list[i+1], sliceBase, target)
			ctx.Env = baseEnv
			_ = jitPlaceIntoPair(ctx, &branchValue, target)
			ctx.EmitJmp(endLabel)
			hasBranch = true
		}
		ctx.MarkLabel(nextLabel)
		i += 2
		if outcome.always {
			terminated = true
			break
		}
	}
	ctx.Env = baseEnv
	if !terminated {
		var fallback JITValueDesc
		if i < len(list) {
			fallback = jitCompileExpr(ctx, list[i], sliceBase, target)
		} else {
			fallback = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
		}
		_ = jitPlaceIntoPair(ctx, &fallback, target)
	}
	if hasBranch {
		ctx.MarkLabel(endLabel)
	}
	ctx.FreeDesc(&value)
	ctx.BindReg(target.Reg, &target)
	ctx.BindReg(target.Reg2, &target)
	return target
}

// EmitNewSliceFromGoSlice retags a Go []Scmer header as a Scheme list without
// allocating or copying its backing storage. Ownership of the data pointer is
// transferred from slice to the returned descriptor.
func (ctx *JITContext) EmitNewSliceFromGoSlice(slice *JITValueDesc) JITValueDesc {
	ctx.EnsureDesc(slice)
	if slice.Loc != LocRegTriple {
		panic("jit: NewSlice requires a Go slice header")
	}
	ptrReg, lenReg, capReg := slice.Reg, slice.Reg2, slice.Reg3
	auxReg := ctx.AllocRegExcept(ptrReg, lenReg, capReg)
	ctx.EmitMovRegReg(auxReg, lenReg)
	ctx.EmitShlRegImm8(auxReg, sliceCapBits)
	ctx.EmitOrInt64(auxReg, capReg)
	ctx.EmitShlRegImm8(auxReg, 8)
	ctx.EmitOrRegImm32(auxReg, int32(tagSlice))
	oldID := slice.ID
	for _, r := range [...]Reg{ptrReg, lenReg, capReg} {
		if owner := ctx.RegOwners[r]; owner != nil && (oldID == 0 || owner.ID == oldID) {
			ctx.RegOwners[r] = nil
		}
	}
	ctx.FreeRegs |= 1 << uint(lenReg)
	ctx.FreeRegs |= 1 << uint(capReg)
	ctx.FreeRegs &^= 1 << uint(ptrReg)
	slice.Loc = LocNone
	result := JITValueDesc{Loc: LocRegPair, Type: tagSlice, Reg: ptrReg, Reg2: auxReg}
	ctx.BindReg(ptrReg, &result)
	ctx.BindReg(auxReg, &result)
	return result
}

// jitMaterializeVirtualSlice lowers the virtual variadic array produced from
// List's Go SSA. A normal list always gets fresh backing storage, preserving
// the optimizer's Transfer contract even when the JIT is invoked through
// apply with a caller-owned argument slice. Only the internal !list form may
// borrow invocation-frame storage under its optimizer-proven NoEscape scope.
func jitMaterializeVirtualSlice(ctx *JITContext, virtual JITValueDesc, result JITValueDesc) JITValueDesc {
	if virtual.Loc != LocVirtualSlice {
		panic("jit: expected virtual variadic slice")
	}
	if len(virtual.Virtual) > 8 {
		panic("jit: variadic slice materialization supports at most 8 elements")
	}
	forwardsInput := result.Loc == LocRegPair && result.Reg == RegRAX && result.Reg2 == RegRBX && len(virtual.Virtual) == ctx.InputArgCount
	for i := range virtual.Virtual {
		if virtual.Virtual[i].Loc != LocInputPair || virtual.Virtual[i].StackOff != int32(i) {
			forwardsInput = false
			break
		}
	}
	if forwardsInput {
		// The call boundary gives this invocation a fresh variadic array. List's
		// body is therefore declarative: keep RAX as the result pointer and emit
		// only the Scmer slice tag/length word required in RBX.
		ctx.TransferInputArgs = true
		ctx.EmitMovRegImm64(RegRBX, makeAux(tagSlice, makeSliceAux(ctx.InputArgCount, ctx.InputArgCount)))
		return JITValueDesc{Loc: LocRegPair, Type: tagSlice, Reg: RegRAX, Reg2: RegRBX}
	}
	pairs := make([]JITValueDesc, len(virtual.Virtual))
	for i := range virtual.Virtual {
		src := virtual.Virtual[i]
		if src.Loc == LocRegPair || src.Loc == LocStackPair || src.Loc == LocInputPair {
			pairs[i] = src
			continue
		}
		ctx.EnsureDesc(&src)
		target := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
		pairs[i] = jitPlaceIntoPair(ctx, &src, target)
		ctx.BindReg(pairs[i].Reg, &pairs[i])
		ctx.BindReg(pairs[i].Reg2, &pairs[i])
	}
	var addr uint64
	switch len(pairs) {
	case 0:
		addr = GoFuncAddr(jitList0)
	case 1:
		addr = GoFuncAddr(jitList1)
	case 2:
		addr = GoFuncAddr(jitList2)
	case 3:
		addr = GoFuncAddr(jitList3)
	case 4:
		addr = GoFuncAddr(jitList4)
	case 5:
		addr = GoFuncAddr(jitList5)
	case 6:
		addr = GoFuncAddr(jitList6)
	case 7:
		addr = GoFuncAddr(jitList7)
	case 8:
		addr = GoFuncAddr(jitList8)
	}
	if result.Loc == LocRegPair {
		materialized := ctx.EmitGoCallScalarInto(addr, pairs, result)
		materialized.Type = tagSlice
		return materialized
	}
	materialized := ctx.EmitGoCallScalar(addr, pairs, 2)
	materialized.Type = tagSlice
	ctx.BindReg(materialized.Reg, &materialized)
	ctx.BindReg(materialized.Reg2, &materialized)
	return materialized
}

func jitCondToBool(ctx *JITContext, cond *JITValueDesc) JITValueDesc {
	return ctx.EmitBoolDesc(cond, JITValueDesc{Loc: LocAny})
}

// jitCompileStackList lowers the optimizer-internal
// (!list NthLocalVar(start) count expr...) form. The optimizer emits !list only
// for a proven NoEscape argument, so its backing slots may live in the current
// invocation frame and must never be returned from the JIT entry point.
func jitCompileStackList(ctx *JITContext, list []Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	if result.Loc != LocAny {
		panic("jit: !list cannot escape its NoEscape consumer")
	}
	if len(list) < 3 || !list[1].IsNthLocalVar() || !list[2].IsInt() {
		panic("jit: malformed optimized !list")
	}
	start := int(list[1].NthLocalVar())
	count := int(list[2].Int())
	if count < 0 || len(list) != count+3 || start < 0 || start+count > ctx.LocalSlotCount {
		panic("jit: !list slots outside invocation frame")
	}
	if !ctx.SliceBaseTracksRSP {
		panic("jit: !list requires an invocation-local stack frame")
	}
	for i := 0; i < count; i++ {
		value := jitCompileExpr(ctx, list[i+3], sliceBase, JITValueDesc{Loc: LocAny})
		ctx.EnsureDesc(&value)
		if value.Loc != LocRegPair && value.Loc != LocImm {
			target := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			value = jitPlaceIntoPair(ctx, &value, target)
		}
		ctx.EmitStoreScmerToStack(value, int32((start+i)*16))
		ctx.FreeDesc(&value)
	}
	target := jitEnsureResultPair(ctx, result)
	ctx.EmitLeaRegMem(target.Reg, RegRSP, int32(start*16))
	ctx.EmitMovRegImm64(target.Reg2, makeAux(tagSlice, makeSliceAux(count, count)))
	target.Type = tagSlice
	target.KnownSliceLen = int32(count)
	target.KnownSliceCap = int32(count)
	target.SliceSizeKnown = true
	ctx.BindReg(target.Reg, &target)
	ctx.BindReg(target.Reg2, &target)
	return target
}

// jitKnownSliceHeader unwraps a Scmer whose tag is already proven to be a
// slice. This is the constant-folded counterpart of jitAsSlice: it emits only
// the ptr/len/cap extraction needed by the following list primitive and no Go
// call or runtime type check.
func jitKnownSliceHeader(ctx *JITContext, value *JITValueDesc) JITValueDesc {
	ptrReg := ctx.AllocReg()
	lenReg := ctx.AllocRegExcept(ptrReg)
	capReg := ctx.AllocRegExcept(ptrReg, lenReg)
	if value.Loc == LocImm {
		ctx.TrackImm(value.Imm)
		ptrWord, auxWord := value.Imm.RawWords()
		length, capacity := decodeSliceAux(auxWord)
		ctx.EmitMovRegImm64(ptrReg, uint64(ptrWord))
		ctx.EmitMovRegImm64(lenReg, uint64(length))
		ctx.EmitMovRegImm64(capReg, uint64(capacity))
	} else {
		ctx.EnsureDesc(value)
		if value.Loc != LocRegPair {
			panic("jit: known slice is not a Scmer pair")
		}
		ctx.EmitMovRegReg(ptrReg, value.Reg)
		ctx.EmitMovRegReg(lenReg, value.Reg2)
		ctx.EmitShrRegImm8(lenReg, 8+sliceCapBits)
		ctx.EmitMovRegReg(capReg, value.Reg2)
		ctx.EmitShrRegImm8(capReg, 8)
		ctx.EmitAndRegImm32(capReg, int32(sliceCapMask))
	}
	result := JITValueDesc{
		Loc: LocRegTriple, Type: tagSlice, Reg: ptrReg, Reg2: lenReg, Reg3: capReg,
		KnownSliceLen: value.KnownSliceLen, KnownSliceCap: value.KnownSliceCap, SliceSizeKnown: value.SliceSizeKnown,
	}
	ctx.BindReg(ptrReg, &result)
	ctx.BindReg(lenReg, &result)
	ctx.BindReg(capReg, &result)
	return result
}

// jitCondToBoolBorrowed evaluates truthiness without consuming cond.
func jitCondToBoolBorrowed(ctx *JITContext, cond *JITValueDesc) JITValueDesc {
	if cond.Loc == LocImm {
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(cond.Imm.Bool())}
	}
	if cond.Type == tagNil {
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
	}
	if cond.Type == tagDate {
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
	}

	// Known primitive truthiness without consuming the original value.
	if cond.Type == tagBool || cond.Type == tagInt || cond.Type == tagFloat {
		tmp := *cond
		ctx.EnsureDesc(&tmp)
		tmpLoc := tmp.Loc
		tmpReg := tmp.Reg
		tmpReg2 := tmp.Reg2
		var valReg Reg
		switch tmp.Loc {
		case LocReg:
			valReg = tmp.Reg
		case LocRegPair:
			valReg = tmp.Reg2
		default:
			panic("jit: borrowed bool test needs register value")
		}
		dst := ctx.AllocReg()
		if dst != valReg {
			ctx.emitMovRegReg(dst, valReg)
		}
		if cond.Type == tagFloat {
			mask := ctx.AllocReg()
			ctx.EmitMovRegImm64(mask, 0x7fffffffffffffff)
			ctx.EmitAndInt64(dst, mask)
			ctx.FreeReg(mask)
		} else if cond.Type == tagBool {
			// Bool payload is auxVal in bits [63:8]; low 8 bits hold the tag.
			ctx.EmitShrRegImm8(dst, 8)
		}
		ctx.EmitCmpRegImm32(dst, 0)
		ctx.EmitSetcc(dst, CcNE)
		switch tmpLoc {
		case LocReg:
			if dst != tmpReg {
				ctx.FreeReg(tmpReg)
			}
		case LocRegPair:
			if dst == tmpReg {
				ctx.FreeReg(tmpReg2)
			} else if dst == tmpReg2 {
				ctx.FreeReg(tmpReg)
			} else {
				ctx.FreeReg(tmpReg)
				ctx.FreeReg(tmpReg2)
			}
		default:
			ctx.FreeDesc(&tmp)
		}
		return JITValueDesc{Loc: LocReg, Type: tagBool, Reg: dst}
	}

	out := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Bool), []JITValueDesc{*cond}, 1)
	ctx.EmitAndRegImm32(out.Reg, 1)
	out.Type = tagBool
	return out
}

// jitIsNilBorrowed checks nil-ness without consuming v.
func jitIsNilBorrowed(ctx *JITContext, v *JITValueDesc) JITValueDesc {
	if v.Loc == LocImm {
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(v.Imm.IsNil())}
	}
	if v.Type != JITTypeUnknown {
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(v.Type == tagNil)}
	}
	tmp := *v
	ctx.EnsureDesc(&tmp)
	if tmp.Loc != LocRegPair {
		ctx.FreeDesc(&tmp)
		out := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.IsNil), []JITValueDesc{*v}, 1)
		ctx.EmitAndRegImm32(out.Reg, 1)
		out.Type = tagBool
		return out
	}
	tagReg := ctx.AllocReg()
	ctx.emitGetTagRegs(tagReg, tmp.Reg, tmp.Reg2)
	ctx.EmitCmpRegImm8(tagReg, tagNil)
	ctx.EmitSetcc(tagReg, CcE)
	ctx.FreeDesc(&tmp)
	return JITValueDesc{Loc: LocReg, Type: tagBool, Reg: tagReg}
}

func jitBuildIfTail(tail []Scmer) Scmer {
	if len(tail) == 0 {
		return NewNil()
	}
	if len(tail) == 1 {
		return tail[0]
	}
	parts := make([]Scmer, 0, len(tail)+1)
	parts = append(parts, NewSymbol("if"))
	parts = append(parts, tail...)
	return NewSlice(parts)
}

func jitEmitGoVariadicCallFromExprs(ctx *JITContext, fn func(...Scmer) Scmer, argExprs []Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	argc := len(argExprs)
	var argsSlice JITValueDesc
	stackBytes := int32(argc * 16)
	if argc > 0 {
		// Reserve one contiguous frame for all variadic Scmer arguments.
		ctx.EmitSubRSP32(stackBytes)
		if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
			ctx.EmitMovRegReg(ctx.SliceBase, RegRSP)
		}
		// Compile each argument directly towards its final stack slot.
		for i := 0; i < len(argExprs); i++ {
			slotOff := int32(i * 16)
			slot := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: slotOff}
			v := jitCompileExpr(ctx, argExprs[i], sliceBase, slot)
			if !(v.Loc == LocStackPair && v.MemPtr == 0 && v.StackOff == slotOff) {
				tmp := JITValueDesc{
					Loc:  LocRegPair,
					Type: JITTypeUnknown,
					Reg:  ctx.AllocReg(),
					Reg2: ctx.AllocReg(),
				}
				_ = jitPlaceIntoPair(ctx, &v, tmp)
				ctx.EmitStoreRegMem(tmp.Reg, RegRSP, slotOff)
				ctx.EmitStoreRegMem(tmp.Reg2, RegRSP, slotOff+8)
				ctx.FreeDesc(&tmp)
			}
			ctx.FreeDesc(&v)
		}
		// argslice: ptr + len (cap = len inside EmitGoCallVariadic).
		argsSlice = JITValueDesc{
			Loc:  LocRegPair,
			Type: JITTypeUnknown,
			Reg:  ctx.AllocReg(),
			Reg2: ctx.AllocReg(),
		}
		ctx.EmitMovRegReg(argsSlice.Reg, RegRSP)
		ctx.EmitMovRegImm64(argsSlice.Reg2, uint64(argc))
		ctx.BindReg(argsSlice.Reg, &argsSlice)
		ctx.BindReg(argsSlice.Reg2, &argsSlice)
	} else {
		argsSlice = JITValueDesc{
			Loc:  LocRegPair,
			Type: JITTypeUnknown,
			Reg:  ctx.AllocReg(),
			Reg2: ctx.AllocReg(),
		}
		ctx.EmitMovRegImm64(argsSlice.Reg, 0)
		ctx.EmitMovRegImm64(argsSlice.Reg2, 0)
		ctx.BindReg(argsSlice.Reg, &argsSlice)
		ctx.BindReg(argsSlice.Reg2, &argsSlice)
	}

	out := ctx.EmitGoCallVariadic(fn, argsSlice, result)
	ctx.FreeDesc(&argsSlice)
	if stackBytes != 0 {
		ctx.EmitAddRSP32(stackBytes)
		if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
			ctx.EmitMovRegReg(ctx.SliceBase, RegRSP)
		}
	}
	return out
}

// jitEmitCondJump emits branch code equivalent to Eval(...).Bool():
// jumps to trueLbl when expr is truthy, otherwise to falseLbl.
// It short-circuits nested (and ...)/(or ...)/(if ...) directly without
// forcing intermediate boolean materialization.
func jitEmitCondJump(ctx *JITContext, expr Scmer, sliceBase Reg, trueLbl, falseLbl uint8) {
	if expr.GetTag() == tagSourceInfo {
		expr = expr.SourceInfo().value
	}
	if expr.GetTag() == tagSlice {
		list := expr.Slice()
		if len(list) > 0 && list[0].IsSymbol() {
			switch string(list[0].Symbol()) {
			case "and":
				// Eval semantics: (and) => true
				if len(list) <= 1 {
					ctx.EmitJmp(trueLbl)
					return
				}
				for i := 1; i < len(list)-1; i++ {
					nextLbl := ctx.ReserveLabel()
					jitEmitCondJump(ctx, list[i], sliceBase, nextLbl, falseLbl)
					ctx.MarkLabel(nextLbl)
				}
				jitEmitCondJump(ctx, list[len(list)-1], sliceBase, trueLbl, falseLbl)
				return
			case "or":
				// Eval semantics: (or) => false
				if len(list) <= 1 {
					ctx.EmitJmp(falseLbl)
					return
				}
				for i := 1; i < len(list)-1; i++ {
					nextLbl := ctx.ReserveLabel()
					jitEmitCondJump(ctx, list[i], sliceBase, trueLbl, nextLbl)
					ctx.MarkLabel(nextLbl)
				}
				jitEmitCondJump(ctx, list[len(list)-1], sliceBase, trueLbl, falseLbl)
				return
			case "if":
				// Eval semantics: chain of condition/value pairs plus optional else.
				i := 1
				for i+1 < len(list) {
					thenCondLbl := ctx.ReserveLabel()
					nextCondLbl := ctx.ReserveLabel()
					jitEmitCondJump(ctx, list[i], sliceBase, thenCondLbl, nextCondLbl)
					ctx.MarkLabel(thenCondLbl)
					jitEmitCondJump(ctx, list[i+1], sliceBase, trueLbl, falseLbl)
					ctx.MarkLabel(nextCondLbl)
					i += 2
				}
				if i < len(list) {
					jitEmitCondJump(ctx, list[i], sliceBase, trueLbl, falseLbl)
				} else {
					// No else branch => nil => false
					ctx.EmitJmp(falseLbl)
				}
				return
			}
		}
	}

	cond := jitCompileExpr(ctx, expr, sliceBase, JITValueDesc{Loc: LocAny})
	b := jitCondToBool(ctx, &cond)
	if b.Loc == LocImm {
		if b.Imm.Bool() {
			ctx.EmitJmp(trueLbl)
		} else {
			ctx.EmitJmp(falseLbl)
		}
		return
	}
	ctx.EmitCmpRegImm32(b.Reg, 0)
	ctx.EmitJcc(CcNE, trueLbl)
	ctx.EmitJmp(falseLbl)
	ctx.FreeDesc(&b)
}

// jitCompileExpr recursively compiles a Scheme expression to machine code.
// sliceBase is the GPR holding the variadic args slice pointer.
// result tells the emitter where to place the output.
// Panics on unsupported expressions (caught by jitCompileExprBodyToExec).
func jitCompileExpr(ctx *JITContext, expr Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	if expr.GetTag() == tagSourceInfo {
		si := expr.SourceInfo()
		if ctx.Arena != nil && si.source != "" {
			codeOffset := int32(uintptr(ctx.Ptr) - uintptr(ctx.Arena.base))
			ctx.Arena.addSourceEntry(jitSourceEntry{
				offset: codeOffset,
				file:   si.source,
				line:   int32(si.line),
			})
		}
		expr = si.value
	}
	switch expr.GetTag() {
	case tagNil:
		ctx.TrackImm(expr)
		return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: expr}
	case tagBool:
		ctx.TrackImm(expr)
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: expr}
	case tagInt:
		ctx.TrackImm(expr)
		return JITValueDesc{Loc: LocImm, Type: tagInt, Imm: expr}
	case tagFloat:
		ctx.TrackImm(expr)
		return JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: expr}
	case tagString:
		ctx.TrackImm(expr)
		return JITValueDesc{Loc: LocImm, Type: tagString, Imm: expr}
	case tagSymbol:
		sym := expr.Symbol()
		if string(sym) == "nil" {
			imm := NewNil()
			ctx.TrackImm(imm)
			return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
		}
		if ctx.Env != nil {
			if desc, ok := ctx.Env.Lookup(sym); ok {
				if desc.Loc == LocImm {
					ctx.TrackImm(desc.Imm)
				}
				return desc
			}
		}
		if v, ok := Globalenv.Vars[sym]; ok {
			ctx.TrackImm(v)
			return JITValueDesc{Loc: LocImm, Type: v.GetTag(), Imm: v}
		}
		panic("jit: unresolved symbol " + string(sym))
	case tagNthLocalVar:
		// Load parameter: check inline env first (JITEmitProcInline places args here).
		idx := int(expr.NthLocalVar())
		if ctx.Env != nil && idx < len(ctx.Env.Numbered) {
			src := ctx.Env.Numbered[idx]
			if result.Loc == LocRegPair {
				switch src.Loc {
				case LocImm:
					ctx.TrackImm(src.Imm)
					ptr, aux := src.Imm.RawWords()
					ctx.EmitMovRegImm64(result.Reg, uint64(ptr))
					ctx.EmitMovRegImm64(result.Reg2, aux)
					d := JITValueDesc{Loc: LocRegPair, Type: src.Type, Reg: result.Reg, Reg2: result.Reg2}
					ctx.BindReg(result.Reg, &d)
					ctx.BindReg(result.Reg2, &d)
					return d
				case LocRegPair:
					ctx.EnsureDesc(&src)
					if src.Reg != result.Reg {
						ctx.emitMovRegReg(result.Reg, src.Reg)
					}
					if src.Reg2 != result.Reg2 {
						ctx.emitMovRegReg(result.Reg2, src.Reg2)
					}
					d := JITValueDesc{Loc: LocRegPair, Type: src.Type, Reg: result.Reg, Reg2: result.Reg2}
					ctx.BindReg(result.Reg, &d)
					ctx.BindReg(result.Reg2, &d)
					return d
				}
			}
			switch src.Loc {
			case LocImm:
				ctx.TrackImm(src.Imm)
				return src // constants are always safe to alias
			case LocInputPair, LocStack, LocStackPair, LocStackTriple:
				// Preserve lazy locations across an inlined Proc boundary. The
				// consumer decides whether it needs registers; eagerly loading here
				// both loses the callback argument's real stack location and creates
				// avoidable register pressure in nested expressions.
				return src
			case LocReg:
				// Allocate a fresh register so each use is independently freeable.
				r := ctx.AllocRegExcept(src.Reg)
				ctx.emitMovRegReg(r, src.Reg)
				d := JITValueDesc{Loc: LocReg, Type: src.Type, Reg: r}
				ctx.BindReg(r, &d)
				return d
			case LocRegPair:
				r1 := ctx.AllocRegExcept(src.Reg, src.Reg2)
				r2 := ctx.AllocRegExcept(src.Reg, src.Reg2, r1)
				ctx.emitMovRegReg(r1, src.Reg)
				ctx.emitMovRegReg(r2, src.Reg2)
				d := JITValueDesc{Loc: LocRegPair, Type: src.Type, Reg: r1, Reg2: r2}
				ctx.BindReg(r1, &d)
				ctx.BindReg(r2, &d)
				return d
			}
		}
		if result.Loc == LocRegPair {
			ctx.EmitLoadArgPair(result.Reg, result.Reg2, sliceBase, idx)
			d := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: result.Reg, Reg2: result.Reg2}
			ctx.BindReg(result.Reg, &d)
			ctx.BindReg(result.Reg2, &d)
			return d
		}
		// Fallback: load from args slice: ptr at [base+i*16], aux at [base+i*16+8]
		ptrReg := ctx.AllocReg()
		auxReg := ctx.AllocReg()
		ctx.emitMovRegMem(ptrReg, sliceBase, int32(idx*16))
		ctx.emitMovRegMem(auxReg, sliceBase, int32(idx*16+8))
		d := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ptrReg, Reg2: auxReg}
		ctx.BindReg(ptrReg, &d)
		ctx.BindReg(auxReg, &d)
		return d
	case tagSlice:
		list := expr.Slice()
		if len(list) == 0 {
			imm := NewNil()
			ctx.TrackImm(imm)
			return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
		}
		// Resolve operator
		if !list[0].IsSymbol() {
			panic("jit: non-symbol in call position")
		}
		name := string(list[0].Symbol())
		switch name {
		case "jit-enabled?":
			if len(list) != 1 {
				panic("jit: jit-enabled? does not accept arguments")
			}
			imm := NewBool(jitEnabled)
			ctx.TrackImm(imm)
			return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: imm}
		case "outer":
			if len(list) != 2 || ctx.Env == nil || ctx.Env.Outer == nil {
				panic("jit: invalid outer reference")
			}
			current := ctx.Env
			ctx.Env = current.Outer
			defer func() { ctx.Env = current }()
			return jitCompileExpr(ctx, list[1], sliceBase, result)
		case "quote":
			if len(list) < 2 {
				imm := NewNil()
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
			}
			q := list[1]
			if q.GetTag() == tagSourceInfo {
				q = q.SourceInfo().value
			}
			ctx.TrackImm(q)
			return JITValueDesc{Loc: LocImm, Type: q.GetTag(), Imm: q}
		case "!list":
			return jitCompileStackList(ctx, list, sliceBase, result)
		case "match", "match_mut":
			return jitCompileMatch(ctx, list, sliceBase, result)
		case "if":
			if len(list) < 3 {
				imm := NewNil()
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
			}
			target := jitEnsureResultPair(ctx, result)
			var endLbl uint8
			hasDynamic := false
			i := 1
			for i+1 < len(list) {
				cond := jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
				b := jitCondToBool(ctx, &cond)
				if b.Loc == LocImm {
					if b.Imm.Bool() {
						thenVal := jitCompileExpr(ctx, list[i+1], sliceBase, target)
						_ = jitPlaceIntoPair(ctx, &thenVal, target)
						if hasDynamic {
							ctx.MarkLabel(endLbl)
						}
						ctx.BindReg(target.Reg, &target)
						ctx.BindReg(target.Reg2, &target)
						return target
					}
					i += 2
					continue
				}
				if !hasDynamic {
					endLbl = ctx.ReserveLabel()
					hasDynamic = true
				}
				nextCondLbl := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(b.Reg, 0)
				ctx.EmitJcc(CcE, nextCondLbl)
				ctx.FreeDesc(&b)
				thenVal := jitCompileExpr(ctx, list[i+1], sliceBase, target)
				_ = jitPlaceIntoPair(ctx, &thenVal, target)
				ctx.EmitJmp(endLbl)
				ctx.MarkLabel(nextCondLbl)
				i += 2
			}
			if i < len(list) {
				elseVal := jitCompileExpr(ctx, list[i], sliceBase, target)
				_ = jitPlaceIntoPair(ctx, &elseVal, target)
			} else {
				nilDesc := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				_ = jitPlaceIntoPair(ctx, &nilDesc, target)
			}
			if hasDynamic {
				ctx.MarkLabel(endLbl)
			}
			ctx.BindReg(target.Reg, &target)
			ctx.BindReg(target.Reg2, &target)
			return target
		case "and":
			if len(list) <= 1 {
				imm := NewBool(true)
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: imm}
			}
			target := jitEnsureResultPair(ctx, result)
			var falseLbl uint8
			var endLbl uint8
			hasDynamic := false
			compileTimeFalse := false
			for i := 1; i < len(list); i++ {
				c := jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
				b := jitCondToBool(ctx, &c)
				if b.Loc == LocImm {
					if !b.Imm.Bool() {
						compileTimeFalse = true
						break
					}
					continue
				}
				if !hasDynamic {
					falseLbl = ctx.ReserveLabel()
					endLbl = ctx.ReserveLabel()
					hasDynamic = true
				}
				ctx.EmitCmpRegImm32(b.Reg, 0)
				ctx.EmitJcc(CcE, falseLbl)
				ctx.FreeDesc(&b)
			}
			if compileTimeFalse {
				if hasDynamic {
					ctx.MarkLabel(falseLbl)
				}
				falseDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				_ = jitPlaceIntoPair(ctx, &falseDesc, target)
				ctx.BindReg(target.Reg, &target)
				ctx.BindReg(target.Reg2, &target)
				return target
			}
			if !hasDynamic {
				trueDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
				_ = jitPlaceIntoPair(ctx, &trueDesc, target)
				ctx.BindReg(target.Reg, &target)
				ctx.BindReg(target.Reg2, &target)
				return target
			}
			trueDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
			_ = jitPlaceIntoPair(ctx, &trueDesc, target)
			ctx.EmitJmp(endLbl)
			ctx.MarkLabel(falseLbl)
			falseDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
			_ = jitPlaceIntoPair(ctx, &falseDesc, target)
			ctx.MarkLabel(endLbl)
			ctx.BindReg(target.Reg, &target)
			ctx.BindReg(target.Reg2, &target)
			return target
		case "or":
			if len(list) <= 1 {
				imm := NewBool(false)
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: imm}
			}
			target := jitEnsureResultPair(ctx, result)
			var trueLbl uint8
			var endLbl uint8
			hasDynamic := false
			compileTimeTrue := false
			for i := 1; i < len(list); i++ {
				c := jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
				b := jitCondToBool(ctx, &c)
				if b.Loc == LocImm {
					if b.Imm.Bool() {
						compileTimeTrue = true
						break
					}
					continue
				}
				if !hasDynamic {
					trueLbl = ctx.ReserveLabel()
					endLbl = ctx.ReserveLabel()
					hasDynamic = true
				}
				ctx.EmitCmpRegImm32(b.Reg, 0)
				ctx.EmitJcc(CcNE, trueLbl)
				ctx.FreeDesc(&b)
			}
			if compileTimeTrue {
				if hasDynamic {
					ctx.MarkLabel(trueLbl)
				}
				trueDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
				_ = jitPlaceIntoPair(ctx, &trueDesc, target)
				ctx.BindReg(target.Reg, &target)
				ctx.BindReg(target.Reg2, &target)
				return target
			}
			if !hasDynamic {
				falseDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				_ = jitPlaceIntoPair(ctx, &falseDesc, target)
				ctx.BindReg(target.Reg, &target)
				ctx.BindReg(target.Reg2, &target)
				return target
			}
			falseDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
			_ = jitPlaceIntoPair(ctx, &falseDesc, target)
			ctx.EmitJmp(endLbl)
			ctx.MarkLabel(trueLbl)
			trueDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
			_ = jitPlaceIntoPair(ctx, &trueDesc, target)
			ctx.MarkLabel(endLbl)
			ctx.BindReg(target.Reg, &target)
			ctx.BindReg(target.Reg2, &target)
			return target
		case "coalesce":
			// Eval semantics:
			// return first truthy value; if none truthy, return last value; empty => nil.
			if len(list) <= 1 {
				imm := NewNil()
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
			}
			target := jitEnsureResultPair(ctx, result)
			endLbl := ctx.ReserveLabel()
			for i := 1; i < len(list); i++ {
				v := jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
				if i == len(list)-1 {
					_ = jitPlaceIntoPair(ctx, &v, target)
					break
				}
				if v.Loc == LocImm {
					if v.Imm.Bool() {
						_ = jitPlaceIntoPair(ctx, &v, target)
						ctx.EmitJmp(endLbl)
						break
					}
					continue
				}
				b := jitCondToBoolBorrowed(ctx, &v)
				if b.Loc == LocImm {
					if b.Imm.Bool() {
						_ = jitPlaceIntoPair(ctx, &v, target)
						ctx.EmitJmp(endLbl)
					}
					ctx.FreeDesc(&v)
					continue
				}
				takeLbl := ctx.ReserveLabel()
				nextLbl := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(b.Reg, 0)
				ctx.EmitJcc(CcNE, takeLbl)
				ctx.EmitJmp(nextLbl)
				ctx.MarkLabel(takeLbl)
				_ = jitPlaceIntoPair(ctx, &v, target)
				ctx.EmitJmp(endLbl)
				ctx.MarkLabel(nextLbl)
				ctx.FreeDesc(&b)
				ctx.FreeDesc(&v)
			}
			ctx.MarkLabel(endLbl)
			ctx.BindReg(target.Reg, &target)
			ctx.BindReg(target.Reg2, &target)
			return target
		case "coalesceNil":
			// Eval semantics:
			// return first non-nil value among args; empty => nil.
			if len(list) <= 1 {
				imm := NewNil()
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
			}
			target := jitEnsureResultPair(ctx, result)
			endLbl := ctx.ReserveLabel()
			for i := 1; i < len(list); i++ {
				v := jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
				if v.Loc == LocImm {
					if !v.Imm.IsNil() {
						_ = jitPlaceIntoPair(ctx, &v, target)
						ctx.EmitJmp(endLbl)
						break
					}
					continue
				}
				isNil := jitIsNilBorrowed(ctx, &v)
				if isNil.Loc == LocImm {
					if !isNil.Imm.Bool() {
						_ = jitPlaceIntoPair(ctx, &v, target)
						ctx.EmitJmp(endLbl)
					}
					ctx.FreeDesc(&v)
					continue
				}
				takeLbl := ctx.ReserveLabel()
				nextLbl := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(isNil.Reg, 0)
				ctx.EmitJcc(CcE, takeLbl) // isNil == 0 => take value
				ctx.EmitJmp(nextLbl)
				ctx.MarkLabel(takeLbl)
				_ = jitPlaceIntoPair(ctx, &v, target)
				ctx.EmitJmp(endLbl)
				ctx.MarkLabel(nextLbl)
				ctx.FreeDesc(&isNil)
				ctx.FreeDesc(&v)
			}
			ctx.MarkLabel(endLbl)
			ctx.BindReg(target.Reg, &target)
			ctx.BindReg(target.Reg2, &target)
			return target
		case "lambda":
			if len(list) < 3 {
				panic("jit: lambda expects params and body")
			}
			params := list[1]
			if params.IsSourceInfo() {
				params = params.SourceInfo().value
			}
			body := list[2]
			numVars := 0
			if len(list) > 3 {
				numVars = int(ToInt(list[3]))
			}

			// Build variadic builder args:
			// [params, body, numVars, key1, val1, ...]
			// keys are quoted Symbol or quoted NthLocalVar.
			argExprs := make([]Scmer, 0, 16)
			argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), params}))
			argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), body}))
			argExprs = append(argExprs, NewInt(int64(numVars)))

			// Capture non-global free symbol variables from current lexical scope.
			for _, sym := range jitLambdaFreeSymbols(params, body) {
				if ctx.Env != nil {
					if d, ok := ctx.Env.Lookup(sym); ok {
						_ = d
						argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), NewSymbol(string(sym))}))
						argExprs = append(argExprs, NewSymbol(string(sym)))
						continue
					}
				}
				// Globals are resolved through closure Outer env.
				if _, ok := Globalenv.Vars[sym]; ok {
					continue
				}
				// Leave unresolved symbols late-bound via closure Outer env.
			}

			// Capture optimized outer(var i) references as numbered captures.
			for _, idx := range jitLambdaOuterVarIndices(body) {
				key := NewNthLocalVar(idx)
				argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), key}))
				argExprs = append(argExprs, key)
			}

			return jitEmitGoVariadicCallFromExprs(ctx, jitBuildLambdaClosure, argExprs, sliceBase, result)
		case "error":
			// Keep one real Go callback in the experimental JIT so panic/recover is
			// exercised across the registered JIT frame. More complex variadic
			// error formatting remains on the interpreter fallback path.
			if len(list) != 2 {
				panic("jit: variadic error is not supported")
			}
			arg := jitCompileExpr(ctx, list[1], sliceBase, JITValueDesc{Loc: LocAny})
			pair := JITValueDesc{
				Loc:  LocRegPair,
				Type: JITTypeUnknown,
				Reg:  ctx.AllocReg(),
				Reg2: ctx.AllocReg(),
			}
			pair = jitPlaceIntoPair(ctx, &arg, pair)
			ctx.EmitGoCallVoid(GoFuncAddr(jitPanic), []JITValueDesc{pair})
			ctx.FreeDesc(&pair)
			target := jitEnsureResultPair(ctx, result)
			ctx.EmitMakeNil(target)
			return target
		}
		decl, ok := declarations[name]
		if !ok {
			panic("jit: unknown callable " + name)
		}
		if name == "strlike" {
			panic("jit: strlike emitter is not supported")
		}
		// Pointer-bearing return values need a complete stack map/liveness
		// contract for Go callbacks. Keep them interpreted until that contract is
		// implemented instead of exposing half-supported generated emitters.
		if decl.Type != nil && decl.Type.Return != nil && decl.Type.Return.Kind == "string" {
			panic("jit: pointer-bearing return is not supported: " + name)
		}
		if decl.Type != nil && decl.Type.JITEmit != nil {
			if decl.Type.JITVirtualArgs {
				args := make([]JITValueDesc, len(list)-1)
				for i := 1; i < len(list); i++ {
					argExpr := list[i]
					for argExpr.GetTag() == tagSourceInfo {
						argExpr = argExpr.SourceInfo().value
					}
					if argExpr.IsNthLocalVar() {
						idx := int(argExpr.NthLocalVar())
						if idx < ctx.InputArgCount {
							args[i-1] = JITValueDesc{Loc: LocInputPair, Type: JITTypeUnknown, StackOff: int32(idx)}
						} else if ctx.Env != nil && idx < len(ctx.Env.Numbered) {
							args[i-1] = ctx.Env.Numbered[idx]
						} else {
							args[i-1] = jitCompileExpr(ctx, argExpr, sliceBase, JITValueDesc{Loc: LocAny})
						}
					} else {
						args[i-1] = jitCompileExpr(ctx, argExpr, sliceBase, JITValueDesc{Loc: LocAny})
					}
				}
				out := decl.Type.JITEmit(ctx, list[1:], args, result)
				out.NoHeapPointer = jitReturnHasNoHeapPointer(decl.Type.Return)
				return out
			}
			// Generated declaration emitters still require a general nested-result
			// preservation contract. Special forms such as (outer ...) have their
			// own lowering and do not create another generated emitter.
			for argIndex, argExpr := range list[1:] {
				for argExpr.GetTag() == tagSourceInfo {
					argExpr = argExpr.SourceInfo().value
				}
				if argExpr.GetTag() == tagSlice {
					nested := argExpr.Slice()
					isQuote := len(nested) > 0 && nested[0].IsSymbol() && nested[0].Symbol() == Symbol("quote")
					param := jitDeclarationParam(decl, argIndex)
					isLambdaTemplate := param != nil && param.Kind == "func" && len(nested) > 0 && nested[0].IsSymbol() && nested[0].SymbolEquals("lambda")
					// !list's backing storage is the current JIT frame. Until the
					// generated-emitter contract can express result aliasing, only
					// nth may consume it: nth returns one Scmer value, never a view
					// into the list backing array.
					isStackListForNth := name == "nth" && len(nested) > 0 && nested[0].IsSymbol() && nested[0].Symbol() == Symbol("!list")
					var nestedType *TypeDescriptor
					if len(nested) > 0 && nested[0].IsSymbol() {
						if nestedDecl, exists := declarations[string(nested[0].Symbol())]; exists {
							nestedType = nestedDecl.Type
						}
					}
					isSpecialForm := nestedType == nil
					if !isQuote && !isStackListForNth && !isLambdaTemplate && !isSpecialForm {
						panic("jit: nested generated emitter has pointer-bearing or unknown result: " + SerializeToString(argExpr, &Globalenv))
					}
				}
			}
			// Compile arguments (intermediate results use LocAny).
			// Use a stack-allocated buffer for the common case of <=8 args;
			// fall back to heap allocation for larger expressions.
			var argsBuf [8]JITValueDesc
			n := len(list) - 1
			var args []JITValueDesc
			if n <= len(argsBuf) {
				args = argsBuf[:n]
			} else {
				args = make([]JITValueDesc, n)
			}
			protectedRegs := make([]Reg, 0, len(list)*2)
			for i := 1; i < len(list); i++ {
				args[i-1] = jitCompileCallArgument(ctx, decl, i-1, list[i], sliceBase)
				// Keep argument descriptors tracked while compiling later args and
				// inside the callee JITEmit body. Without rebinding to args[] slots,
				// register spills/reuse can leave stale copies in args and break
				// non-commutative operators (e.g. subtraction).
				switch args[i-1].Loc {
				case LocReg:
					ctx.BindReg(args[i-1].Reg, &args[i-1])
					ctx.ProtectReg(args[i-1].Reg)
					protectedRegs = append(protectedRegs, args[i-1].Reg)
				case LocRegPair:
					ctx.BindReg(args[i-1].Reg, &args[i-1])
					ctx.BindReg(args[i-1].Reg2, &args[i-1])
					ctx.ProtectReg(args[i-1].Reg)
					ctx.ProtectReg(args[i-1].Reg2)
					protectedRegs = append(protectedRegs, args[i-1].Reg, args[i-1].Reg2)
				}
			}
			// Keep call arguments resident while compiling the callee emitter so
			// later arguments cannot evict values that the emitter still consumes.
			defer func() {
				for _, r := range protectedRegs {
					ctx.UnprotectReg(r)
				}
			}()
			// Argument compilation may have rendered mutually exclusive type paths.
			// Their path-local temporaries are intentionally unowned after merging;
			// reclaim them before entering another generated emitter while retaining
			// the explicitly bound and protected argument descriptors above.
			ctx.ReclaimUntrackedRegs()
			labelsBefore := ctx.LabelNext
			out := decl.Type.JITEmit(ctx, list[1:], args, result)
			// A generated emitter may render several runtime control-flow paths.
			// Its mutable result descriptor then contains the type of whichever
			// path happened to be rendered last, not a valid merged type. Keep the
			// placement but discard that path-local type information.
			if ctx.LabelNext != labelsBefore && out.Loc == LocRegPair {
				out.Type = JITTypeUnknown
			}
			out.NoHeapPointer = jitReturnHasNoHeapPointer(decl.Type.Return)
			if out.Loc == LocImm {
				ctx.TrackImm(out.Imm)
			}
			return out
		}
		// Declarations without a dedicated emitter stay interpreted. The former
		// generic variadic bridge built argument arrays below the fixed frame and
		// also made every ordinary runtime function part of the experimental ABI.
		panic("jit: callable has no JIT emitter: " + name)
	default:
		panic(fmt.Sprintf("jit: unsupported expression tag=%d expr=%s", expr.GetTag(), SerializeToString(expr, &Globalenv)))
	}
}

// JITEmitProcInline emits a Proc's body inline into the current JIT stream.
// args[i] provides the pre-placed descriptor for the i-th parameter (NthLocalVar(i)).
// Each NthLocalVar reference emits a fresh register copy so the descriptor is
// independently freeable per use site (safe for expressions that reference a
// parameter more than once).
// sliceBase is passed through to jitCompileExpr for any fallback slice-based
// NthLocalVar loads (in practice not reached when all params are in args).
// Panics on any un-emittable sub-expression — callers should recover and fall back.
func JITEmitProcInline(ctx *JITContext, proc *Proc, args []JITValueDesc, sliceBase Reg, result JITValueDesc) JITValueDesc {
	return JITEmitProcInlineWithOuter(ctx, proc, ctx.Env, args, sliceBase, result)
}

func JITEmitProcInlineWithOuter(ctx *JITContext, proc *Proc, outer *JITEnv, args []JITValueDesc, sliceBase Reg, result JITValueDesc) JITValueDesc {
	innerEnv := &JITEnv{
		Numbered: args,
		Outer:    outer,
	}
	oldEnv := ctx.Env
	ctx.Env = innerEnv
	defer func() { ctx.Env = oldEnv }()

	body := proc.Body
	if body.GetTag() == tagSourceInfo {
		body = body.SourceInfo().value
	}
	out := jitCompileExpr(ctx, body, sliceBase, result)
	if result.Loc == LocRegPair && (out.Loc != LocRegPair || out.Reg != result.Reg || out.Reg2 != result.Reg2) {
		return jitPlaceIntoPair(ctx, &out, result)
	}
	return out
}

func jitLambdaTemplate(expr Scmer, outer *JITEnv) (*JITLambdaTemplate, bool) {
	for expr.GetTag() == tagSourceInfo {
		expr = expr.SourceInfo().value
	}
	if expr.GetTag() != tagSlice {
		return nil, false
	}
	parts := expr.Slice()
	if len(parts) < 3 || !parts[0].IsSymbol() || !parts[0].SymbolEquals("lambda") {
		return nil, false
	}
	params := parts[1]
	for params.GetTag() == tagSourceInfo {
		params = params.SourceInfo().value
	}
	numVars := 0
	if len(parts) > 3 {
		numVars = int(ToInt(parts[3]))
	}
	return &JITLambdaTemplate{Proc: Proc{Params: params, Body: parts[2], NumVars: numVars, NumberedOnly: procCanUseNumberedOnly(params, parts[2], numVars)}, Outer: outer}, true
}

func jitDeclarationParam(decl *Declaration, index int) *TypeDescriptor {
	if decl == nil || decl.Type == nil || len(decl.Type.Params) == 0 {
		return nil
	}
	if index < len(decl.Type.Params) {
		return decl.Type.Params[index]
	}
	last := decl.Type.Params[len(decl.Type.Params)-1]
	if last.Variadic {
		return last
	}
	return nil
}

func jitCompileCallArgument(ctx *JITContext, decl *Declaration, index int, expr Scmer, sliceBase Reg) JITValueDesc {
	param := jitDeclarationParam(decl, index)
	if param != nil && param.Kind == "func" {
		if lambda, ok := jitLambdaTemplate(expr, ctx.Env); ok {
			return JITValueDesc{Loc: LocLambdaTemplate, Type: tagProc, Lambda: lambda}
		}
	}
	hasCallback := false
	for _, candidate := range decl.Type.Params {
		if candidate.Kind == "func" {
			hasCallback = true
			break
		}
	}
	if hasCallback {
		for expr.GetTag() == tagSourceInfo {
			expr = expr.SourceInfo().value
		}
		if expr.IsNthLocalVar() {
			idx := int(expr.NthLocalVar())
			if ctx.Env != nil && idx < len(ctx.Env.Numbered) {
				return ctx.Env.Numbered[idx]
			}
			if idx < ctx.InputArgCount {
				return JITValueDesc{Loc: LocInputPair, Type: JITTypeUnknown, StackOff: int32(idx)}
			}
		}
	}
	return jitCompileExpr(ctx, expr, sliceBase, JITValueDesc{Loc: LocAny})
}

// There is deliberately no later peephole optimizer. This backend is the final
// stage of an intelligent one-pass emitter: known types and constants eliminate
// checks and branches before bytes are written, impossible specializations
// abort to the interpreter fallback, and dynamic cases emit only their required
// checks. Immediate width, moves, inlining, and control flow are selected here
// instead of producing generic code for a second optimization pass.
// AMD64 register constants for the Go register ABI.
//
// Go register ABI (amd64): args in RAX, RBX, RCX, RDX, RSI, RDI, R8-R15
// Scmer return: RAX=ptr, RBX=aux
// Variadic args: RAX=slice_ptr, RBX=slice_len, RCX=slice_cap
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
func (ctx *JITContext) EmitCmpFloat64Setcc(dst, left, right Reg, cc byte) {
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

// EmitJcc emits a conditional jump with a rel32 fixup.
func (ctx *JITContext) EmitJcc(cc byte, labelID uint8) {
	ctx.emitBytes(0x0F, 0x80|cc) // Jcc rel32
	ctx.AddFixup(labelID, 4, true)
	ctx.emitU32(0) // placeholder
}

// EmitJmp emits an unconditional JMP rel32.
func (ctx *JITContext) EmitJmp(labelID uint8) {
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

// Condition code constants for EmitJcc
const (
	CcE  byte = 0x04 // JE  / JZ  (ZF=1)
	CcNE byte = 0x05 // JNE / JNZ (ZF=0)
	CcBE byte = 0x06 // JBE (unsigned <=)
	CcA  byte = 0x07 // JA  (unsigned >)
	CcL  byte = 0x0C // JL        (SF!=OF)
	CcGE byte = 0x0D // JGE       (SF=OF)
	CcLE byte = 0x0E // JLE       (ZF=1 || SF!=OF)
	CcG  byte = 0x0F // JG        (ZF=0 && SF=OF)
	CcB  byte = 0x02 // JB  (unsigned <)
	CcAE byte = 0x03 // JAE (unsigned >=)
)

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
func (ctx *JITContext) EmitSetcc(dst Reg, cc byte) {
	dstEnc := byte(dst & 7)
	// SETcc r/m8: 0F 9x /0
	if dst >= 8 {
		ctx.emitBytes(0x41, 0x0F, 0x90|cc, 0xC0|dstEnc)
	} else if dst >= 4 {
		ctx.emitBytes(0x40, 0x0F, 0x90|cc, 0xC0|dstEnc) // REX for SIL/DIL/BPL/SPL
	} else {
		ctx.emitBytes(0x0F, 0x90|cc, 0xC0|dstEnc)
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
	ctx.emitGetTagRegs(dst, src.Reg, src.Reg2)
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
	ctx.emitGetTagRegs(tagReg, src.Reg, src.Reg2)
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
		ctx.emitGetTagRegs(tagReg, src.Reg, src.Reg2)
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
		} else if src.Type == tagBool {
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
	pair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
	pair = jitPlaceIntoPair(ctx, src, pair)
	out := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Bool), []JITValueDesc{pair}, 1)
	// Go bool returns may leave upper bits undefined; normalize to 0|1.
	ctx.EmitAndRegImm32(out.Reg, 1)
	out.Type = tagBool
	ctx.FreeDesc(&pair)
	ctx.FreeDesc(src)
	return emitResult(out)
}

// EmitMovToReg moves a JITValueDesc value into a specific GPR register.
// Handles LocImm (materializes constant) and LocReg (register-to-register move).
func (ctx *JITContext) EmitMovToReg(dst Reg, src JITValueDesc) {
	switch src.Loc {
	case LocImm:
		ctx.EmitMovRegImm64(dst, uint64(src.Imm.Int()))
	case LocReg:
		if src.Reg != dst {
			ctx.emitMovRegReg(dst, src.Reg)
		}
	}
}

// emitGetTagRegs emits inline code for (Scmer).GetTag().
// Input: ptrReg holds s.ptr, auxReg holds s.aux.
// Output: result in dstReg as uint16.
// Logic: if ptr == &scmerIntSentinel → tagInt (4)
//
//	if ptr == &scmerFloatSentinel → tagFloat (3)
//	else → aux & 0xFF
func (ctx *JITContext) emitGetTagRegs(dst, ptrReg, auxReg Reg) {
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
}

// EmitPopReg emits POP r64
func (ctx *JITContext) EmitPopReg(r Reg) {
	if r >= 8 {
		ctx.emitBytes(0x41, 0x58|byte(r&7))
	} else {
		ctx.emitByte(0x58 | byte(r))
	}
}

// EmitCallIndirect emits an unwind marker followed by MOV R11, imm64; CALL R11.
func (ctx *JITContext) EmitCallIndirect(addr uint64) {
	ctx.emitCallIndirectWithSetup(addr, nil)
}

func (ctx *JITContext) emitCallIndirectWithSetup(addr uint64, setup func(callFrameBytes int32)) {
	ctx.EmitMovRegReg(RegR11, RegRBP)
	ctx.EmitSubInt64(RegR11, RegRSP)
	ctx.EmitPushReg(RegR11)
	ctx.EmitPushReg(RegR11)
	ctx.EmitSubRSP32(int32(jitGoSpillBytes))
	if setup != nil {
		setup(int32(jitGoSpillBytes + 16))
	}
	ctx.EmitMovRegImm64(RegR11, addr)
	ctx.emitBytes(0x41, 0xFF, 0xD3) // CALL R11
	ctx.EmitAddRSP32(int32(jitGoSpillBytes + 16))
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

	target := result
	targetHasRegs := target.Loc == LocRegPair
	if targetHasRegs {
		target.Type = JITTypeUnknown
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
			if r == target.Reg || r == target.Reg2 {
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
	ctx.EmitAddRSP32(int32(jitGoSpillBytes + 16))

	if !targetHasRegs {
		target = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
		targetHasRegs = true
	}
	if target.Reg != RegRAX {
		ctx.EmitMovRegReg(target.Reg, RegRAX)
	}
	if target.Reg2 != RegRBX {
		ctx.EmitMovRegReg(target.Reg2, RegRBX)
	}
	for i, r := range liveRegs {
		ctx.EmitMovRegMem(r, RegRSP, int32(i*8))
	}
	if frameBytes != 0 {
		ctx.EmitAddRSP32(frameBytes)
	}
	ctx.BindReg(target.Reg, &target)
	ctx.BindReg(target.Reg2, &target)

	if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
		ctx.emitMovRegReg(ctx.SliceBase, RegRSP)
	}
	return target
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
}

// EmitAddRSP emits ADD RSP, imm8 to release stack space.
func (ctx *JITContext) EmitAddRSP(n uint8) {
	ctx.emitBytes(0x48, 0x83, 0xC4, n)
}

// EmitSubRSP32Fixup emits SUB RSP, imm32 with a zero placeholder and returns
// a pointer to the 4-byte immediate so it can be patched later via PatchInt32.
func (ctx *JITContext) EmitSubRSP32Fixup() unsafe.Pointer {
	ctx.emitBytes(0x48, 0x81, 0xEC)
	ctx.emitU32(0)
	return unsafe.Add(ctx.Ptr, -4)
}

// PatchInt32 writes a 32-bit little-endian value at the given position.
func (ctx *JITContext) PatchInt32(pos unsafe.Pointer, val int32) {
	*(*int32)(pos) = val
}

// EmitAddRSP32 emits ADD RSP, imm32.
func (ctx *JITContext) EmitAddRSP32(val int32) {
	ctx.emitBytes(0x48, 0x81, 0xC4)
	ctx.emitU32(uint32(val))
}

// EmitSubRSP32 emits SUB RSP, imm32.
func (ctx *JITContext) EmitSubRSP32(val int32) {
	ctx.emitBytes(0x48, 0x81, 0xEC)
	ctx.emitU32(uint32(val))
}

// EmitStoreToStack stores a JITValueDesc value to a stack slot at [RSP+disp].
// Uses R11 as scratch for LocImm values.
// Frame slots and this helper both use offsets from the function's stable RSP.
func (ctx *JITContext) EmitStoreToStack(src JITValueDesc, disp int32) {
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
	switch desc.Loc {
	case LocRegPair:
		ctx.EmitStoreRegMem(desc.Reg, RegRSP, disp)
		ctx.EmitStoreRegMem(desc.Reg2, RegRSP, disp+8)
	case LocImm:
		// Store ptr word
		ctx.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(desc.Imm.ptr))))
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp)
		// Store aux word
		ctx.EmitMovRegImm64(RegR11, desc.Imm.aux)
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp+8)
	default:
		panic("jit: EmitStoreScmerToStack: unsupported location")
	}
}
