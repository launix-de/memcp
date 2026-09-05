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
	"io"
	"math"
	"math/bits"
	"sort"
	"sync/atomic"
	"unsafe"
)

func jitParkCallArgument(ctx *JITContext, value *JITValueDesc) bool {
	switch value.Loc {
	case LocReg:
		reg := value.Reg
		ctx.UnprotectReg(reg)
		off := ctx.AllocStack(8)
		ctx.EmitStoreToStack(*value, off)
		ctx.RegOwners[reg] = nil
		ctx.FreeReg(reg)
		value.Loc, value.StackOff, value.Reg = LocStack, off, 0
		return true
	case LocRegPair:
		reg, reg2 := value.Reg, value.Reg2
		ctx.UnprotectReg(reg)
		ctx.UnprotectReg(reg2)
		off := ctx.AllocStack(16)
		ctx.EmitStoreScmerToStack(*value, off)
		ctx.RegOwners[reg], ctx.RegOwners[reg2] = nil, nil
		ctx.FreeReg(reg)
		ctx.FreeReg(reg2)
		value.Loc, value.StackOff, value.Reg, value.Reg2 = LocStackPair, off, 0, 0
		value.Rooted = true
		return true
	case LocRegTriple:
		reg, reg2, reg3 := value.Reg, value.Reg2, value.Reg3
		ctx.UnprotectReg(reg)
		ctx.UnprotectReg(reg2)
		ctx.UnprotectReg(reg3)
		off := ctx.AllocStack(24)
		ctx.EmitStoreRegMem(reg, ctx.StackReg, off)
		ctx.EmitStoreRegMem(reg2, ctx.StackReg, off+8)
		ctx.EmitStoreRegMem(reg3, ctx.StackReg, off+16)
		ctx.setStackPointer(jitStackRootFrameSP, off, true)
		ctx.RegOwners[reg], ctx.RegOwners[reg2], ctx.RegOwners[reg3] = nil, nil, nil
		ctx.FreeReg(reg)
		ctx.FreeReg(reg2)
		ctx.FreeReg(reg3)
		value.Loc, value.StackOff, value.Reg, value.Reg2, value.Reg3 = LocStackTriple, off, 0, 0, 0
		value.Rooted = true
		return true
	}
	return false
}

// This file contains architecture-independent Scheme expression lowering. It
// operates on JITValueDesc values and the common register-bank/emitter API;
// instruction encoding and ABI frame construction remain architecture-owned.
func jitRequiredLocalSlots(expr Scmer, minimum int) int {
	for expr.IsSourceInfo() {
		expr = expr.SourceInfo().value
	}
	if expr.IsNthLocalVar() {
		required := int(expr.NthLocalVar()) + 1
		if required > minimum {
			return required
		}
		return minimum
	}
	if !expr.IsSlice() {
		return minimum
	}
	items := expr.Slice()
	if len(items) > 0 {
		switch jitSyntaxKind(items[0]) {
		case SyntaxQuote, SyntaxParser, SyntaxLambda:
			return minimum
		}
	}
	for _, item := range items {
		minimum = jitRequiredLocalSlots(item, minimum)
	}
	return minimum
}

func jitSelfSymbols(proc *Proc) map[Symbol]struct{} {
	if proc == nil {
		return nil
	}
	candidates := jitLambdaFreeSymbols(proc.Params, proc.Body)
	if len(candidates) == 0 {
		return nil
	}
	result := make(map[Symbol]struct{})
	for _, symbol := range candidates {
		found := false
		for env := proc.En; env != nil; env = env.Outer {
			if value, exists := env.Vars[symbol]; exists {
				if value.GetTag() == tagProc && value.Proc() == proc {
					result[symbol] = struct{}{}
				}
				found = true
				break
			}
		}
		if found || proc.En == &Globalenv {
			continue
		}
		if value, exists := Globalenv.Vars[symbol]; exists && value.GetTag() == tagProc && value.Proc() == proc {
			result[symbol] = struct{}{}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func jitIsSelfCall(ctx *JITContext, symbol Symbol) bool {
	if _, self := ctx.SelfSymbols[symbol]; !self {
		return false
	}
	if ctx.Env != nil {
		if _, shadowed := ctx.Env.Lookup(symbol); shadowed {
			return false
		}
	}
	return true
}

func jitEnsureResultPair(ctx *JITContext, result JITValueDesc) JITValueDesc {
	if result.Loc == LocRegPair {
		result.Type = JITTypeUnknown
		return result
	}
	ctx.ReclaimUntrackedRegs()
	return jitAllocTrackedPair(ctx, JITTypeUnknown)
}

func jitAllocTrackedPair(ctx *JITContext, valueType uint8) JITValueDesc {
	reg := ctx.AllocReg()
	desc := JITValueDesc{Loc: LocRegPair, Type: valueType, Reg: reg, Reg2: ctx.AllocRegExcept(reg)}
	ctx.BindReg(desc.Reg, &desc)
	ctx.BindReg(desc.Reg2, &desc)
	return desc
}

func jitDescRegs(desc JITValueDesc) []Reg {
	switch desc.Loc {
	case LocReg:
		return []Reg{desc.Reg}
	case LocRegPair:
		return []Reg{desc.Reg, desc.Reg2}
	case LocRegTriple:
		return []Reg{desc.Reg, desc.Reg2, desc.Reg3}
	default:
		return nil
	}
}

func jitPlaceIntoPair(ctx *JITContext, src *JITValueDesc, target JITValueDesc) JITValueDesc {
	if target.Loc != LocRegPair {
		panic("jit: jitPlaceIntoPair requires LocRegPair target")
	}
	ctx.SyncDesc(src)
	// Placement changes representation, not the proven Scheme type. Keeping the
	// fact lets downstream generated SSA eliminate checks and write barriers.
	target.Type = src.Type
	target.NoHeapPointer = src.NoHeapPointer
	target.RelocatablePointer = src.RelocatablePointer
	target.Rooted = src.Rooted
	// A producer that spills directly into a phi/callback slot should also be
	// consumed from that slot directly. Loading through a temporary pair defeats
	// the target-register API and can require four live registers for a two-word
	// move under nested parser/lambda pressure.
	if src.Loc == LocStackPair {
		base := ctx.StackReg
		if src.StackOff < 0 {
			base = ctx.FrameReg
		}
		ctx.EmitMovRegMem(target.Reg, base, src.StackOff)
		ctx.EmitMovRegMem(target.Reg2, base, src.StackOff+8)
		return target
	}
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
	case LocInputPair, LocClosurePair, LocStack, LocStackPair:
		ctx.EnsureDesc(src)
		return jitPlaceIntoPair(ctx, src, target)
	case LocRegPair:
		if src.Reg != target.Reg {
			ctx.EmitMovRegReg(target.Reg, src.Reg)
		}
		if src.Reg2 != target.Reg2 {
			ctx.EmitMovRegReg(target.Reg2, src.Reg2)
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

// jitCopyScmerToPair gives a nested Go call its own two-register Scmer value.
// Function values and other compile-time constants are LocImm descriptors, but
// Go's ABI still expects both Scmer words; flattenArgs deliberately treats an
// immediate as one scalar word unless the caller makes that representation
// explicit.
func jitCopyScmerToPair(ctx *JITContext, src JITValueDesc) JITValueDesc {
	ctx.EnsureDesc(&src)
	if src.Loc != LocRegPair {
		if src.Loc == LocReg {
			ctx.ProtectReg(src.Reg)
		}
		target := jitAllocTrackedPair(ctx, src.Type)
		if src.Loc == LocReg {
			ctx.UnprotectReg(src.Reg)
		}
		return jitPlaceIntoPair(ctx, &src, target)
	}
	ctx.ProtectReg(src.Reg)
	ctx.ProtectReg(src.Reg2)
	target := jitAllocTrackedPair(ctx, src.Type)
	ctx.EmitMovRegReg(target.Reg, src.Reg)
	ctx.EmitMovRegReg(target.Reg2, src.Reg2)
	ctx.UnprotectReg(src.Reg2)
	ctx.UnprotectReg(src.Reg)
	return target
}

// JITPrepareScmerGoArg restores the two-word Scmer representation required by
// Go's ABI after the JIT type system has folded a value to an unboxed scalar.
// Already boxed register, input, and spill values stay in place so native call
// boundaries do not introduce copies when both words are already available.
func JITPrepareScmerGoArg(ctx *JITContext, src JITValueDesc) JITValueDesc {
	ctx.SyncDesc(&src)
	switch src.Loc {
	case LocRegPair, LocStackPair, LocInputPair:
		return src
	case LocImm, LocReg, LocStack:
		return jitCopyScmerToPair(ctx, src)
	default:
		panic("jit: Go call expects a Scmer argument")
	}
}

func jitPlaceScmerIntoTarget(ctx *JITContext, src JITValueDesc, target JITValueDesc) JITValueDesc {
	if target.Loc == LocAny {
		return src
	}
	if target.Loc != LocRegPair && target.Loc != LocStackPair {
		panic("jit: Scmer target must be a register pair or stack pair")
	}
	if src.Loc != LocImm && src.Loc != LocRegPair && src.Loc != LocStackPair && src.Loc != LocInputPair {
		pair := jitAllocTrackedPair(ctx, src.Type)
		src = jitPlaceIntoPair(ctx, &src, pair)
	}
	target.Type = src.Type
	target.NoHeapPointer = src.NoHeapPointer
	target.RelocatablePointer = src.RelocatablePointer
	target.Rooted = src.Rooted
	if target.Loc == LocStackPair {
		ctx.EmitCopyScmerToDesc(&target, &src)
		if target.StackOff < 0 {
			ctx.setStackPointer(jitStackRootFrameBP, target.StackOff, !target.NoHeapPointer)
		} else {
			ctx.setStackPointer(jitStackRootFrameSP, target.StackOff-ctx.DynamicSP, !target.NoHeapPointer)
		}
		ctx.FreeDesc(&src)
		target.Rooted = true
		return target
	}
	return jitPlaceIntoPair(ctx, &src, target)
}

// jitEmitKnownDeclaration recursively invokes a generated builtin emitter when
// a Go function value resolves to a registered Declaration. The active set is
// a generic recursion guard: self-recursive builtins keep the normal Go-call
// path until their machine-code recursion lowering is selected explicitly.
func jitEmitKnownDeclaration(ctx *JITContext, callable JITValueDesc, args []JITValueDesc, result JITValueDesc) (JITValueDesc, bool) {
	if callable.Loc != LocImm || callable.Imm.GetTag() != tagFunc {
		return JITValueDesc{}, false
	}
	declaration := declarationsByFunction[FunctionIdentity(callable.Imm.Func())]
	if declaration == nil || declaration.Type == nil || declaration.Type.JITEmit == nil {
		return JITValueDesc{}, false
	}
	if ctx.ActiveBuiltinEmitters == nil {
		ctx.ActiveBuiltinEmitters = make(map[*Declaration]uint16)
	}
	if ctx.ActiveBuiltinEmitters[declaration] != 0 {
		return JITValueDesc{}, false
	}
	ctx.ActiveBuiltinEmitters[declaration]++
	defer func() { ctx.ActiveBuiltinEmitters[declaration]-- }()
	emitted := declaration.Type.JITEmit(ctx, nil, args, JITValueDesc{Loc: LocAny})
	return jitPlaceScmerIntoTarget(ctx, emitted, result), true
}

func jitMakeScmerSlice(length, capacity int) []Scmer {
	return make([]Scmer, length, capacity)
}

func jitMakeByteSlice(length, capacity int) []byte {
	return make([]byte, length, capacity)
}

// JITMakeScmerSlice exposes the allocation boundary to generated emitters in
// packages outside scm. It is called by emitted code, not while generating it.
func JITMakeScmerSlice(length, capacity int) []Scmer {
	return jitMakeScmerSlice(length, capacity)
}

// JITMakeByteSlice exposes the byte-slice allocation boundary to generated
// emitters in packages outside scm.
func JITMakeByteSlice(length, capacity int) []byte {
	return jitMakeByteSlice(length, capacity)
}

// JITMakeUint32Slice allocates scratch record identifiers for generated bulk
// readers while preserving Go's precise slice-element pointer metadata.
func JITMakeUint32Slice(length, capacity int) []uint32 {
	return make([]uint32, length, capacity)
}

// JITMakeUint64Slice allocates decoded integer scratch space for generated
// bulk readers.
func JITMakeUint64Slice(length, capacity int) []uint64 {
	return make([]uint64, length, capacity)
}

// JITMakeInt64Slice allocates scratch decoded integers for generated bulk
// readers while preserving Go's precise slice-element pointer metadata.
func JITMakeInt64Slice(length, capacity int) []int64 {
	return make([]int64, length, capacity)
}

// JITMakeIntSlice allocates machine-word scratch positions for generated bulk
// readers.
func JITMakeIntSlice(length, capacity int) []int {
	return make([]int, length, capacity)
}

// JITMakeBoolSlice allocates null masks for generated bulk readers.
func JITMakeBoolSlice(length, capacity int) []bool {
	return make([]bool, length, capacity)
}

// JITAtomicAddUint64 preserves instrumentation-counter semantics at a native
// boundary generated outside the scm package.
func JITAtomicAddUint64(address *uint64, delta uint64) uint64 {
	return atomic.AddUint64(address, delta)
}

func jitStringToBytes(value string) []byte {
	return []byte(value)
}

func jitBytesToString(value []byte) string {
	return string(value)
}

func jitCopyScmerSlice(dst, src []Scmer) int {
	return copy(dst, src)
}

// JITCopyScmerSlice exposes the typed slice-copy boundary to generated
// emitters outside the scm package.
func JITCopyScmerSlice(dst, src []Scmer) int {
	return jitCopyScmerSlice(dst, src)
}

func jitCopyByteSlice(dst, src []byte) int {
	return copy(dst, src)
}

func jitAssertString(value any) (string, bool) {
	result, ok := value.(string)
	return result, ok
}

func jitAssertReader(value any) (io.Reader, bool) {
	result, ok := value.(io.Reader)
	return result, ok
}

func jitAssertScmerFunction(value any) (func(...Scmer) Scmer, bool) {
	result, ok := value.(func(...Scmer) Scmer)
	return result, ok
}

func jitAssertScmer(value any) Scmer {
	return value.(Scmer)
}

func jitReaderToAny(value io.Reader) any {
	return value
}

func jitInvokeMergeCallback(callback func(Scmer, Scmer) Scmer, oldValue, newValue Scmer) Scmer {
	return callback(oldValue, newValue)
}

func jitMakeReservedList(capacityValue Scmer) Scmer {
	capacity := int(capacityValue.Int())
	if capacity < 0 {
		capacity = 0
	}
	return NewSlice(make([]Scmer, 0, capacity))
}

func jitStoreScmerAt(address *Scmer, value Scmer) {
	*address = value
}

func jitResolveRuntimeSymbol(env *Env, symbol Scmer) Scmer {
	if env == nil {
		panic(fmt.Sprintf("jit: invalid runtime environment while resolving %s", symbol.String()))
	}
	sym := mustSymbol(symbol)
	binding := env.FindRead(sym)
	if binding == nil {
		panic("jit: unresolved symbol " + string(sym))
	}
	return binding.Vars[sym]
}

func jitCurrentRuntimeEnv(ctx *JITContext) JITValueDesc {
	if !ctx.UsesRuntimeEnv {
		env, _ := ctx.RuntimeEnv.Any().(*Env)
		ctx.TrackPointer(unsafe.Pointer(env))
		reg := ctx.AllocReg()
		value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: reg, RelocatablePointer: true, Rooted: true}
		ctx.BindReg(reg, &value)
		ctx.EmitMovRegImm64(reg, uint64(uintptr(unsafe.Pointer(env))))
		return value
	}
	return JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: ctx.RuntimeEnvOff, RelocatablePointer: true, Rooted: true}
}

func jitResolveGlobalSymbol(symbol Scmer) Scmer {
	sym := mustSymbol(symbol)
	if value, ok := Globalenv.Vars[sym]; ok {
		return value
	}
	panic("jit: unresolved global symbol " + string(sym))
}

func jitApplyCallable0(callable Scmer, env *Env) Scmer {
	if callable.GetTag() == tagProc {
		if proc := callable.Proc(); proc != nil && proc.JITCode != 0 {
			return proc.callJIT(nil)
		}
	}
	return ApplyEx(callable, nil, env)
}

func jitApplyCallable1(callable Scmer, env *Env, arg0 Scmer) Scmer {
	if callable.GetTag() == tagProc {
		if proc := callable.Proc(); proc != nil && proc.JITCode != 0 {
			return proc.callJIT([]Scmer{arg0})
		}
	}
	return ApplyEx(callable, []Scmer{arg0}, env)
}

func jitApplyCallable2(callable Scmer, env *Env, arg0, arg1 Scmer) Scmer {
	if callable.GetTag() == tagProc {
		if proc := callable.Proc(); proc != nil && proc.JITCode != 0 {
			return proc.callJIT([]Scmer{arg0, arg1})
		}
	}
	return ApplyEx(callable, []Scmer{arg0, arg1}, env)
}

func jitApplyCallableSlice(callable Scmer, env *Env, args []Scmer) Scmer {
	if callable.GetTag() == tagProc {
		if proc := callable.Proc(); proc != nil && proc.JITCode != 0 {
			return proc.callJIT(args)
		}
	}
	return ApplyEx(callable, args, env)
}

func jitInvokeCallbackSlice(callback Scmer, args []Scmer) Scmer {
	return Apply(callback, args...)
}

func jitInvokeGoFunctionSlice(callback func(...Scmer) Scmer, args []Scmer) Scmer {
	return callback(args...)
}

// EmitSliceElementAddress lowers the address part shared by SSA slice loads and
// stores.
func (ctx *JITContext) EmitSliceElementAddress(slice, index *JITValueDesc, elementSize int32) JITValueDesc {
	ctx.ReclaimUntrackedRegs()
	slicePtr := Reg(0)
	loadedPtr := false
	if slice.GoArray && slice.Loc == LocStack {
		slicePtr = ctx.AllocReg()
		base := ctx.StackReg
		if slice.StackOff < 0 {
			base = ctx.FrameReg
		}
		ctx.EmitMovRegMem(slicePtr, base, slice.StackOff)
		loadedPtr = true
	} else if slice.Loc == LocStackTriple || slice.Loc == LocStackPair {
		slicePtr = ctx.AllocReg()
		base := ctx.StackReg
		if slice.StackOff < 0 {
			base = ctx.FrameReg
		}
		ctx.EmitMovRegMem(slicePtr, base, slice.StackOff)
		loadedPtr = true
	} else {
		ctx.EnsureDesc(slice)
		if slice.GoArray && slice.Loc == LocReg {
			slicePtr = slice.Reg
			ctx.ProtectReg(slicePtr)
		} else if slice.Loc != LocRegTriple && slice.Loc != LocRegPair {
			panic(fmt.Sprintf("jit: slice element address requires a Go slice, string header, or array (loc=%d array=%t id=%d)", slice.Loc, slice.GoArray, slice.ID))
		} else {
			slicePtr = slice.Reg
			ctx.ProtectReg(slicePtr)
		}
	}
	ctx.EnsureDesc(index)
	excluded := []Reg{slicePtr}
	if index.Loc == LocReg {
		excluded = append(excluded, index.Reg)
	}
	available := ctx.FreeRegs & ctx.AllRegs &^ ctx.ProtectedRegs
	for _, reg := range excluded {
		available &^= 1 << uint(reg)
	}
	if available == 0 {
		stableSlice := ctx.stabilizeForNested(*slice)
		stableIndex := ctx.stabilizeForNested(*index)
		if loadedPtr {
			ctx.FreeReg(slicePtr)
		} else {
			ctx.UnprotectReg(slicePtr)
		}
		// Reserve the surviving result before snapshotting allocator state. A
		// slot created inside PreserveOuterRegs would be made reusable again by
		// RestoreOuterRegs even though the returned descriptor still names it.
		off := ctx.AllocStack(8)
		ctx.EmitMovRegImm64(ctx.ScratchReg, 0)
		ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, off)
		ctx.setStackPointer(jitStackRootFrameSP, off-ctx.DynamicSP, true)
		outer := ctx.PreserveOuterRegs()
		address := ctx.EmitSliceElementAddress(&stableSlice, &stableIndex, elementSize)
		ctx.EmitStoreRegMem(address.Reg, ctx.StackReg, off)
		ctx.FreeDesc(&address)
		ctx.RestoreOuterRegs(outer)
		return JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: off, Rooted: true, RelocatablePointer: true}
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
	ctx.EmitAddInt64(address, slicePtr)
	if loadedPtr {
		ctx.FreeReg(slicePtr)
	} else {
		ctx.UnprotectReg(slicePtr)
	}
	result := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: address, RelocatablePointer: true}
	ctx.BindReg(address, &result)
	return result
}

// EnsureGoStringHeader materializes a Go string's data pointer and length as
// the same two-word descriptor used by ABI calls and indexed loads.
func (ctx *JITContext) EnsureGoStringHeader(value *JITValueDesc) {
	if value.Loc == LocImm {
		ctx.TrackImm(value.Imm)
		data := ctx.AllocReg()
		length := ctx.AllocRegExcept(data)
		ptrWord, _ := value.Imm.RawWords()
		ctx.EmitMovRegImm64(data, uint64(ptrWord))
		ctx.EmitMovRegImm64(length, uint64(len(value.Imm.String())))
		*value = JITValueDesc{Loc: LocRegPair, Reg: data, Reg2: length, Rooted: true}
		ctx.BindReg(data, value)
		ctx.BindReg(length, value)
		return
	}
	ctx.EnsureDesc(value)
	if value.Loc != LocRegPair && value.Loc != LocStackPair {
		panic("jit: Go string header requires a two-word descriptor")
	}
}

// EmitSliceCapAfterLow computes cap(slice)-low without forcing a stack-backed
// slice header into three registers. Callers pass registers that already hold
// the new pointer/length so allocation cannot destroy them.
func (ctx *JITContext) EmitSliceCapAfterLow(slice, low *JITValueDesc, excluded ...Reg) Reg {
	ctx.ReclaimUntrackedRegs()
	ctx.SyncDesc(slice)
	ctx.SyncDesc(low)
	ctx.EnsureDesc(low)
	if low.Loc == LocReg {
		excluded = append(excluded, low.Reg)
	}
	result := ctx.AllocRegExcept(excluded...)
	switch slice.Loc {
	case LocStackTriple:
		base := ctx.StackReg
		if slice.StackOff < 0 {
			base = ctx.FrameReg
		}
		ctx.EmitMovRegMem(result, base, slice.StackOff+16)
	case LocRegTriple:
		ctx.EmitMovRegReg(result, slice.Reg3)
	default:
		ctx.EnsureDesc(slice)
		if slice.Loc != LocRegTriple {
			panic("jit: slice capacity requires a Go slice header")
		}
		ctx.EmitMovRegReg(result, slice.Reg3)
	}
	if low.Loc == LocImm {
		if low.Imm.Int() >= -2147483648 && low.Imm.Int() <= 2147483647 {
			ctx.EmitSubRegImm32(result, int32(low.Imm.Int()))
		} else {
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(low.Imm.Int()))
			ctx.EmitSubInt64(result, ctx.ScratchReg)
		}
	} else {
		ctx.EmitSubInt64(result, low.Reg)
	}
	return result
}

// EmitSliceDataAfterLow computes slice.data + low*elementSize without assuming
// that a control-flow-stabilized slice header still occupies registers.
func (ctx *JITContext) EmitSliceDataAfterLow(slice, low *JITValueDesc, elementSize int32, excluded ...Reg) Reg {
	ctx.ReclaimUntrackedRegs()
	ctx.SyncDesc(slice)
	ctx.SyncDesc(low)
	ctx.EnsureDesc(low)
	if low.Loc == LocReg {
		excluded = append(excluded, low.Reg)
	}
	result := ctx.AllocRegExcept(excluded...)
	// Allocating the result may spill the slice header. Resolve its current
	// location only after that allocation, then load just the data word.
	ctx.SyncDesc(slice)
	switch slice.Loc {
	case LocImm:
		ctx.EmitMovRegImm64(result, uint64(slice.Imm.Int()))
	case LocStackPair, LocStackTriple:
		base := ctx.StackReg
		if slice.StackOff < 0 {
			base = ctx.FrameReg
		}
		ctx.EmitMovRegMem(result, base, slice.StackOff)
	case LocRegPair, LocRegTriple:
		ctx.EmitMovRegReg(result, slice.Reg)
	default:
		ctx.EnsureDesc(slice)
		if slice.Loc != LocRegPair && slice.Loc != LocRegTriple {
			panic(fmt.Sprintf("jit: slice data requires a Go slice or string header (loc=%d type=%d id=%d)", slice.Loc, slice.Type, slice.ID))
		}
		ctx.EmitMovRegReg(result, slice.Reg)
	}

	if low.Loc == LocImm {
		offset := low.Imm.Int() * int64(elementSize)
		if offset >= -2147483648 && offset <= 2147483647 {
			ctx.EmitAddRegImm32(result, int32(offset))
		} else {
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(offset))
			ctx.EmitAddInt64(result, ctx.ScratchReg)
		}
	} else {
		switch elementSize {
		case 1:
			ctx.EmitAddInt64(result, low.Reg)
		case 2, 4, 8, 16:
			ctx.EmitMovRegReg(ctx.ScratchReg, low.Reg)
			ctx.EmitShlRegImm8(ctx.ScratchReg, uint8(bits.TrailingZeros32(uint32(elementSize))))
			ctx.EmitAddInt64(result, ctx.ScratchReg)
		default:
			ctx.EmitMovRegReg(ctx.ScratchReg, low.Reg)
			factor := ctx.AllocRegExcept(append(excluded, result, low.Reg)...)
			ctx.EmitMovRegImm64(factor, uint64(elementSize))
			ctx.EmitImulInt64(ctx.ScratchReg, factor)
			ctx.FreeReg(factor)
			ctx.EmitAddInt64(result, ctx.ScratchReg)
		}
	}
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
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(uintptr(unsafe.Pointer(value.Imm.ptr))))
			ctx.EmitStoreRegMem(ctx.ScratchReg, address.Reg, 0)
			ctx.EmitMovRegImm64(ctx.ScratchReg, value.Imm.aux)
			ctx.EmitStoreRegMem(ctx.ScratchReg, address.Reg, 8)
			return
		}
		if value.Loc == LocRegPair {
			ctx.EmitStoreRegMem(value.Reg, address.Reg, 0)
			ctx.EmitStoreRegMem(value.Reg2, address.Reg, 8)
			return
		}
		if value.Loc == LocReg {
			ctx.ProtectReg(address.Reg)
			ctx.ProtectReg(value.Reg)
			pair := jitAllocTrackedPair(ctx, value.Type)
			switch value.Type {
			case tagInt:
				ctx.EmitMakeInt(pair, *value)
			case tagFloat:
				ctx.EmitMakeFloat(pair, *value)
			case tagBool:
				ctx.EmitMakeBool(pair, *value)
			case tagNil:
				ctx.EmitMakeNil(pair)
			default:
				ctx.UnprotectReg(value.Reg)
				ctx.UnprotectReg(address.Reg)
				ctx.FreeDesc(&pair)
				panic("jit: unsupported unboxed Scmer slice store")
			}
			ctx.UnprotectReg(value.Reg)
			ctx.UnprotectReg(address.Reg)
			ctx.EmitStoreRegMem(pair.Reg, address.Reg, 0)
			ctx.EmitStoreRegMem(pair.Reg2, address.Reg, 8)
			ctx.FreeDesc(&pair)
			return
		}
	}

	ctx.SyncDesc(value)
	stored := *value
	if stored.Loc != LocRegPair && stored.Loc != LocStackPair && stored.Loc != LocInputPair {
		stored = JITPrepareScmerGoArg(ctx, stored)
	}
	ctx.EmitGoCallVoid(GoFuncAddr(jitStoreScmerAt), []JITValueDesc{*address, stored})
	if stored.ID != value.ID || stored.Loc != value.Loc {
		ctx.FreeDesc(&stored)
	}
}

// EmitLoadScmerToStack lets a slice-element producer write directly into a
// stack-backed consumer (most importantly a phi slot) without first occupying
// two allocator registers for the Scmer pair.
func (ctx *JITContext) EmitLoadScmerToStack(address *JITValueDesc, targetOff int32) {
	ctx.SyncDesc(address)
	if address.Loc == LocStack {
		base := ctx.StackReg
		if address.StackOff < 0 {
			base = ctx.FrameReg
		}
		ctx.EmitMovRegMem(ctx.ScratchReg, base, address.StackOff)
		ctx.EmitMovRegMem(ctx.ScratchReg, ctx.ScratchReg, 0)
		ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, targetOff)
		ctx.EmitMovRegMem(ctx.ScratchReg, base, address.StackOff)
		ctx.EmitMovRegMem(ctx.ScratchReg, ctx.ScratchReg, 8)
		ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, targetOff+8)
	} else {
		ctx.EnsureDesc(address)
		if address.Loc != LocReg {
			panic("jit: Scmer load requires an address")
		}
		ctx.EmitMovRegMem(ctx.ScratchReg, address.Reg, 0)
		ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, targetOff)
		ctx.EmitMovRegMem(ctx.ScratchReg, address.Reg, 8)
		ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, targetOff+8)
	}
	ctx.setStackPointer(jitStackRootFrameSP, targetOff-ctx.DynamicSP, true)
}

func (ctx *JITContext) stabilizeForNested(value JITValueDesc) JITValueDesc {
	ctx.SyncDesc(&value)
	originalID := value.ID
	var words int32
	switch value.Loc {
	case LocReg:
		words = 1
	case LocRegPair:
		words = 2
	case LocRegTriple:
		words = 3
	default:
		return value
	}
	off := ctx.AllocSpill(words * 8)
	regs := [...]Reg{value.Reg, value.Reg2, value.Reg3}
	for i := int32(0); i < words; i++ {
		ctx.EmitStoreRegMem(regs[i], ctx.FrameReg, off+i*8)
		ctx.setStackPointer(jitStackRootFrameBP, off+i*8, jitValueWordIsPointer(value, i))
		if owner := ctx.RegOwners[regs[i]]; owner == nil || owner.ID == originalID {
			ctx.RegOwners[regs[i]] = nil
			ctx.FreeRegs |= 1 << uint(regs[i])
		}
	}
	value.StackOff = off
	value.Reg, value.Reg2, value.Reg3 = 0, 0, 0
	switch words {
	case 1:
		value.Loc = LocStack
	case 2:
		value.Loc = LocStackPair
	case 3:
		value.Loc = LocStackTriple
	}
	if originalID != 0 {
		if owner := ctx.descOwners[originalID]; owner != nil {
			*owner = value
		}
		if ctx.descSpills == nil {
			ctx.descSpills = make(map[uint32]descSpillMeta)
		}
		ctx.descSpills[originalID] = descSpillMeta{loc: value.Loc, stackOff: off}
	}
	return value
}

// EmitCopyStackWords copies a stack-backed descriptor into an stack-pointer-relative
// generated frame slot with one scratch register. It avoids materializing
// multiword phi sources merely to store them back to the stack.
func (ctx *JITContext) EmitCopyStackWords(src JITValueDesc, dst int32, words int) {
	base := ctx.StackReg
	if src.StackOff < 0 {
		base = ctx.FrameReg
	}
	scratch := ctx.AllocReg()
	start, end, step := 0, words, 1
	if base == ctx.StackReg && dst > src.StackOff && dst < src.StackOff+int32(words*8) {
		start, end, step = words-1, -1, -1
	}
	for i := start; i != end; i += step {
		ctx.EmitMovRegMem(scratch, base, src.StackOff+int32(i*8))
		ctx.EmitStoreRegMem(scratch, ctx.StackReg, dst+int32(i*8))
	}
	ctx.FreeReg(scratch)
}

// EmitCopyDescWords copies a scalar/pair/triple into an existing stack-backed
// descriptor. Both source and destination may independently live in the stable
// RSP frame or the RBP spill zone.
func (ctx *JITContext) EmitCopyDescWords(dst, src *JITValueDesc, words int) {
	ctx.SyncDesc(dst)
	ctx.SyncDesc(src)
	dstBase := ctx.StackReg
	if dst.StackOff < 0 {
		dstBase = ctx.FrameReg
	}
	if src.Loc == LocImm {
		if words != 1 {
			panic("jit: multiword immediate descriptor copy")
		}
		var word uint64
		switch src.Type {
		case tagFloat:
			word = math.Float64bits(src.Imm.Float())
		case tagBool:
			if src.Imm.Bool() {
				word = 1
			}
		default:
			word = uint64(src.Imm.Int())
		}
		ctx.EmitMovRegImm64(ctx.ScratchReg, word)
		ctx.EmitStoreRegMem(ctx.ScratchReg, dstBase, dst.StackOff)
		return
	}
	if src.Loc == LocStack || src.Loc == LocStackPair || src.Loc == LocStackTriple {
		srcBase := ctx.StackReg
		if src.StackOff < 0 {
			srcBase = ctx.FrameReg
		}
		scratch := ctx.AllocReg()
		start, end, step := 0, words, 1
		if srcBase == dstBase && dst.StackOff > src.StackOff && dst.StackOff < src.StackOff+int32(words*8) {
			start, end, step = words-1, -1, -1
		}
		for i := start; i != end; i += step {
			off := int32(i * 8)
			ctx.EmitMovRegMem(scratch, srcBase, src.StackOff+off)
			ctx.EmitStoreRegMem(scratch, dstBase, dst.StackOff+off)
		}
		ctx.FreeReg(scratch)
		return
	}
	ctx.EnsureDesc(src)
	regs := [...]Reg{src.Reg, src.Reg2, src.Reg3}
	for i := 0; i < words; i++ {
		ctx.EmitStoreRegMem(regs[i], dstBase, dst.StackOff+int32(i*8))
	}
}

func (ctx *JITContext) EmitCopyScmerToDesc(dst, src *JITValueDesc) {
	ctx.SyncDesc(src)
	switch src.Loc {
	case LocRegPair, LocStackPair, LocInputPair:
		ctx.EmitCopyDescWords(dst, src, 2)
	default:
		pair := jitCopyScmerToPair(ctx, *src)
		ctx.EmitCopyDescWords(dst, &pair, 2)
		ctx.FreeDesc(&pair)
	}
}

// EmitStoreScalarAt stores a raw Go scalar through an address descriptor.
// Unlike EmitStoreScmerAt, it respects the pointee width and does not write a
// two-word Scheme value into compact Go arrays such as []uint32 or []bool.
func (ctx *JITContext) EmitStoreScalarAt(address, value *JITValueDesc, width int) {
	ctx.SyncDesc(address)
	ctx.EnsureDesc(address)
	if address.Loc != LocReg {
		panic("jit: scalar store address is not a register")
	}
	ctx.ProtectReg(address.Reg)
	ctx.SyncDesc(value)
	valueReg := value.Reg
	if value.Loc == LocImm {
		var word uint64
		switch value.Type {
		case tagFloat:
			word = math.Float64bits(value.Imm.Float())
		case tagBool:
			if value.Imm.Bool() {
				word = 1
			}
		default:
			word = uint64(value.Imm.Int())
		}
		ctx.EmitMovRegImm64(ctx.ScratchReg, word)
		valueReg = ctx.ScratchReg
	} else {
		ctx.EnsureDesc(value)
		if value.Loc != LocReg {
			panic("jit: scalar store value is not a register")
		}
		valueReg = value.Reg
	}
	switch width {
	case 1:
		ctx.EmitStoreRegMemB(valueReg, address.Reg, 0)
	case 2:
		ctx.EmitStoreRegMemW(valueReg, address.Reg, 0)
	case 4:
		ctx.EmitStoreRegMemL(valueReg, address.Reg, 0)
	case 8:
		ctx.EmitStoreRegMem(valueReg, address.Reg, 0)
	default:
		panic("jit: unsupported scalar store width")
	}
	ctx.UnprotectReg(address.Reg)
}

// EmitZeroDescWords writes a typed Go zero value into an existing stack home.
// It is primarily used for nil pointers, slices, interfaces, and functions in
// multi-result inlined helpers, whose width is known from the Go signature.
func (ctx *JITContext) EmitZeroDescWords(dst *JITValueDesc, words int) {
	ctx.SyncDesc(dst)
	dstBase := ctx.StackReg
	if dst.StackOff < 0 {
		dstBase = ctx.FrameReg
	}
	ctx.EmitMovRegImm64(ctx.ScratchReg, 0)
	for i := 0; i < words; i++ {
		ctx.EmitStoreRegMem(ctx.ScratchReg, dstBase, dst.StackOff+int32(i*8))
	}
}

// PrepareScmerStackTarget reserves a pointer-bearing result slot in every
// subsequent safepoint map before nested emission snapshots allocator state.
// The nil initialization makes the slot safe to scan on paths where its
// producer has not run yet.
func (ctx *JITContext) PrepareScmerStackTarget(off int32) {
	ctx.PreparePointerStackTarget(off, 2)
}

func (ctx *JITContext) PreparePointerStackTarget(off int32, words int) {
	target := JITValueDesc{Loc: LocStack, Type: tagNil, StackOff: off, NoHeapPointer: true}
	ctx.EmitZeroDescWords(&target, words)
	ctx.setStackPointer(jitStackRootFrameSP, off-ctx.DynamicSP, true)
}

// StabilizeDescForControlFlow gives a register-backed SSA value a fixed stack
// home at its producer. Machine code in successor blocks can then be entered
// repeatedly without depending on the allocator state used while those blocks
// were emitted once.
func (ctx *JITContext) StabilizeDescForControlFlow(desc *JITValueDesc) {
	ctx.SyncDesc(desc)
	words := int32(0)
	loc := desc.Loc
	switch loc {
	case LocReg:
		words = 1
	case LocRegPair:
		words = 2
	case LocRegTriple:
		words = 3
	default:
		return
	}
	off := ctx.AllocStack(words * 8)
	regs := [...]Reg{desc.Reg, desc.Reg2, desc.Reg3}
	for i := int32(0); i < words; i++ {
		ctx.EmitStoreRegMem(regs[i], ctx.StackReg, off+i*8)
		ctx.setStackPointer(jitStackRootFrameSP, off+i*8-ctx.DynamicSP, jitValueWordIsPointer(*desc, i))
		owner := ctx.RegOwners[regs[i]]
		ownsReg := owner == desc || (owner != nil && desc.ID != 0 && owner.ID == desc.ID)
		if ownsReg {
			ctx.RegOwners[regs[i]] = nil
			ctx.FreeRegs |= 1 << uint(regs[i])
		}
	}
	desc.Reg, desc.Reg2, desc.Reg3 = 0, 0, 0
	desc.StackOff = off
	if desc.ID != 0 && ctx.descSpills != nil {
		delete(ctx.descSpills, desc.ID)
	}
	desc.ID = 0
	switch loc {
	case LocReg:
		desc.Loc = LocStack
	case LocRegPair:
		desc.Loc = LocStackPair
	case LocRegTriple:
		desc.Loc = LocStackTriple
	}
}

// EmitIsStringBorrowed implements Scmer.IsString without consuming src.
// Strings have three representations, so an exact tagString comparison is
// insufficient for compressed and binary string values produced by storages.
func (ctx *JITContext) EmitIsStringBorrowed(src *JITValueDesc, result JITValueDesc) JITValueDesc {
	isStringTag := func(tag uint8) bool {
		return tag == tagString || tag == tagCString || tag == tagBString
	}
	if src.Loc == LocImm {
		value := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(isStringTag(src.Imm.GetTag()))}
		if result.Loc == LocAny {
			return value
		}
		ctx.EmitMakeBool(result, value)
		return result
	}
	if src.Type != JITTypeUnknown {
		value := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(isStringTag(src.Type))}
		if result.Loc == LocAny {
			return value
		}
		ctx.EmitMakeBool(result, value)
		return result
	}

	tmp := *src
	tmp.ID = 0
	ctx.EnsureDesc(&tmp)
	ctx.ProtectReg(tmp.Reg)
	ctx.ProtectReg(tmp.Reg2)
	tagReg := ctx.AllocRegExcept(tmp.Reg, tmp.Reg2)
	rangeReg := ctx.AllocRegExcept(tmp.Reg, tmp.Reg2, tagReg)
	ctx.UnprotectReg(tmp.Reg2)
	ctx.UnprotectReg(tmp.Reg)
	ctx.EmitGetTagRegs(tagReg, tmp.Reg, tmp.Reg2)
	ctx.EmitMovRegReg(rangeReg, tagReg)
	ctx.EmitSubRegImm32(rangeReg, int32(tagCString))
	ctx.EmitCmpRegImm32(rangeReg, int32(tagBString-tagCString))
	ctx.EmitSetcc(rangeReg, CondUnsignedBelowOrEqual)
	ctx.EmitCmpRegImm32(tagReg, int32(tagString))
	ctx.EmitSetcc(tagReg, CondEqual)
	ctx.EmitOrInt64(tagReg, rangeReg)
	ctx.FreeReg(rangeReg)
	value := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: tagReg}
	ctx.BindReg(tagReg, &value)
	if result.Loc == LocAny {
		return value
	}
	ctx.EmitMakeBool(result, value)
	ctx.FreeDesc(&value)
	return result
}

// StabilizeDescAcrossNestedCall moves a value into the non-reusable spill zone.
// Ordinary control-flow homes may be released by a recursively emitted helper;
// callback-live values must therefore not share their storage with its locals.
func (ctx *JITContext) StabilizeDescAcrossNestedCall(desc *JITValueDesc) {
	ctx.SyncDesc(desc)
	if (desc.Loc == LocStack || desc.Loc == LocStackPair || desc.Loc == LocStackTriple) && desc.StackOff < 0 {
		return
	}
	words := int32(0)
	sourceLoc := desc.Loc
	switch sourceLoc {
	case LocReg, LocStack:
		words = 1
	case LocRegPair, LocStackPair:
		words = 2
	case LocRegTriple, LocStackTriple:
		words = 3
	default:
		return
	}
	off := ctx.AllocSpill(words * 8)
	if sourceLoc == LocStack || sourceLoc == LocStackPair || sourceLoc == LocStackTriple {
		for i := int32(0); i < words; i++ {
			ctx.EmitMovRegMem(ctx.ScratchReg, ctx.StackReg, desc.StackOff+i*8)
			ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.FrameReg, off+i*8)
			ctx.setStackPointer(jitStackRootFrameBP, off+i*8, jitValueWordIsPointer(*desc, i))
		}
	} else {
		regs := [...]Reg{desc.Reg, desc.Reg2, desc.Reg3}
		for i := int32(0); i < words; i++ {
			ctx.EmitStoreRegMem(regs[i], ctx.FrameReg, off+i*8)
			ctx.setStackPointer(jitStackRootFrameBP, off+i*8, jitValueWordIsPointer(*desc, i))
			owner := ctx.RegOwners[regs[i]]
			if owner == desc || (owner != nil && desc.ID != 0 && owner.ID == desc.ID) {
				ctx.RegOwners[regs[i]] = nil
				ctx.FreeRegs |= 1 << uint(regs[i])
			}
		}
	}
	desc.Reg, desc.Reg2, desc.Reg3 = 0, 0, 0
	desc.StackOff = off
	if desc.ID != 0 && ctx.descSpills != nil {
		delete(ctx.descSpills, desc.ID)
	}
	desc.ID = 0
	switch words {
	case 1:
		desc.Loc = LocStack
	case 2:
		desc.Loc = LocStackPair
	case 3:
		desc.Loc = LocStackTriple
	}
}

// StabilizeCallbackArgs copies stack-pointer-relative temporary argument arrays into the
// invocation's frame-pointer-relative spill zone before recursively emitted code adds its
// own control flow and Go calls.
func (ctx *JITContext) StabilizeCallbackArgs(args []JITValueDesc) []JITValueDesc {
	stable := make([]JITValueDesc, len(args))
	for i, arg := range args {
		ctx.syncDescSpill(&arg)
		if arg.Loc != LocStackPair || arg.StackOff < 0 {
			stable[i] = arg
			continue
		}
		off := ctx.AllocSpill(16)
		scratch := ctx.AllocReg()
		ctx.EmitMovRegMem(scratch, ctx.StackReg, arg.StackOff)
		ctx.EmitStoreRegMem(scratch, ctx.FrameReg, off)
		ctx.EmitMovRegMem(scratch, ctx.StackReg, arg.StackOff+8)
		ctx.EmitStoreRegMem(scratch, ctx.FrameReg, off+8)
		ctx.FreeReg(scratch)
		ctx.setStackPointer(jitStackRootFrameBP, off, jitValueWordIsPointer(arg, 0))
		stable[i] = JITValueDesc{Loc: LocStackPair, Type: arg.Type, StackOff: off, NoHeapPointer: arg.NoHeapPointer, Rooted: true}
	}
	return stable
}

// StabilizeJITEnv clones a lexical environment and gives every runtime value a
// frame-pointer-relative home. Recursive emitters may then use the complete
// register bank without invalidating captures described by their outer scope.
func (ctx *JITContext) StabilizeJITEnv(env *JITEnv) *JITEnv {
	if env == nil {
		return nil
	}
	stable := &JITEnv{
		Vars:      make(map[Symbol]JITValueDesc, len(env.Vars)),
		Numbered:  make([]JITValueDesc, len(env.Numbered)),
		Outer:     ctx.StabilizeJITEnv(env.Outer),
		StackBase: env.StackBase,
	}
	symbols := make([]Symbol, 0, len(env.Vars))
	for symbol := range env.Vars {
		symbols = append(symbols, symbol)
	}
	sort.Slice(symbols, func(left, right int) bool { return symbols[left] < symbols[right] })
	for _, symbol := range symbols {
		stable.Vars[symbol] = ctx.stabilizeJITEnvValue(env.Vars[symbol])
	}
	for index, value := range env.Numbered {
		stable.Numbered[index] = ctx.stabilizeJITEnvValue(value)
	}
	return stable
}

func (ctx *JITContext) stabilizeJITEnvValue(value JITValueDesc) JITValueDesc {
	ctx.SyncDesc(&value)
	words := 0
	stableLoc := LocNone
	switch value.Loc {
	case LocReg, LocStack:
		words, stableLoc = 1, LocStack
	case LocRegPair, LocStackPair, LocInputPair:
		words, stableLoc = 2, LocStackPair
	case LocRegTriple, LocStackTriple:
		words, stableLoc = 3, LocStackTriple
	default:
		return value
	}
	if (value.Loc == LocStack || value.Loc == LocStackPair || value.Loc == LocStackTriple) && value.StackOff < 0 {
		return value
	}
	off := ctx.AllocSpill(int32(words * 8))
	stable := JITValueDesc{
		Loc:                stableLoc,
		Type:               value.Type,
		StackOff:           off,
		NoHeapPointer:      value.NoHeapPointer,
		RelocatablePointer: value.RelocatablePointer,
		Rooted:             value.Rooted,
	}
	ctx.EmitCopyDescWords(&stable, &value, words)
	if jitValueWordIsPointer(value, 0) {
		ctx.setStackPointer(jitStackRootFrameBP, off, true)
		stable.Rooted = true
	}
	return stable
}

// EmitGoPanic emits the cold generated bounds-check failure path without
// constructing a Scheme string. Go strings use two ABI words (data, length),
// which a scalar LocImm descriptor intentionally cannot represent.
func (ctx *JITContext) EmitGoPanic(message string) {
	var resultsBuf [16]Reg
	words := []goCallArgWord{
		{loc: LocImm, imm: uint64(uintptr(unsafe.Pointer(unsafe.StringData(message))))},
		{loc: LocImm, imm: uint64(len(message))},
	}
	ctx.EmitGoCall(GoFuncAddr(jitPanicString), words, 0, &resultsBuf, nil)
}

// jitRootScmer gives a pointer-bearing intermediate an invocation-local stack
// home. runtime/jit's precise safepoint maps keep that slot visible to the GC
// and relocate it when the owning goroutine stack grows.
func jitRootScmer(ctx *JITContext, value JITValueDesc) JITValueDesc {
	if value.Rooted || value.NoHeapPointer || value.Loc == LocImm {
		if value.Loc == LocImm {
			ctx.TrackImm(value.Imm)
		}
		value.Rooted = true
		return value
	}
	valueOff := ctx.AllocStack(16)
	valueType := value.Type
	noHeapPointer := value.NoHeapPointer
	ctx.EmitStoreScmerToStack(value, valueOff)
	ctx.FreeDesc(&value)
	rootedValue := JITValueDesc{Loc: LocStackPair, Type: valueType, StackOff: valueOff, NoHeapPointer: noHeapPointer}
	rootedValue.Rooted = true
	return rootedValue
}

// JITEmitGoCallResults emits a static Go call with multiple return values and
// immediately gives every result an invocation-local stack home. pointerMasks
// describes the pointer-bearing ABI words of each result and therefore keeps
// fresh strings, slices, interfaces, and pointers live across later safepoints.
func JITEmitGoCallResults(ctx *JITContext, funcAddr uint64, args []JITValueDesc, wordCounts, pointerMasks []uint8) []JITValueDesc {
	if len(wordCounts) != len(pointerMasks) {
		panic("jit: result word counts and pointer masks differ")
	}
	totalWords := 0
	for _, count := range wordCounts {
		if count < 1 || count > 3 {
			panic("jit: unsupported Go result width")
		}
		totalWords += int(count)
	}
	var wordsBuf [16]goCallArgWord
	words := ctx.flattenArgs(args, &wordsBuf)
	results := make([]JITValueDesc, len(wordCounts))
	resultOffs := make([]int32, totalWords)
	word := 0
	for i, count := range wordCounts {
		off := ctx.AllocSpill(int32(count) * 8)
		for part := 0; part < int(count); part++ {
			resultOffs[word] = off + int32(part*8)
			ctx.setStackPointer(jitStackRootFrameBP, resultOffs[word], pointerMasks[i]&(1<<part) != 0)
			word++
		}
		switch count {
		case 1:
			results[i] = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: off}
		case 2:
			results[i] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: off}
		case 3:
			results[i] = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: off}
		}
	}
	ctx.EmitGoCallToFrame(funcAddr, words, resultOffs)
	return results
}

func JITCloneScmerSlice(values []Scmer) []Scmer {
	return append([]Scmer(nil), values...)
}

func JITAppendScmerSlice(values, added []Scmer) []Scmer {
	return append(values, added...)
}

// JITAppendScmerSliceCopy preserves Go's escaping slice-literal semantics for
// arrays that jitgen represents in the invocation frame. The returned slice
// must never retain that transient backing storage, even when added is empty.
func JITAppendScmerSliceCopy(values, added []Scmer) []Scmer {
	result := make([]Scmer, len(values), len(values)+len(added))
	copy(result, values)
	return append(result, added...)
}

func JITNewSliceCopy(values []Scmer) Scmer {
	return NewSlice(append([]Scmer(nil), values...))
}

func jitNewSliceResult(values []Scmer, length uint64) Scmer {
	return Scmer{
		(*byte)(unsafe.Pointer(unsafe.SliceData(values))),
		makeAux(tagSlice, length<<sliceCapBits|length),
	}
}

func jitNewSlice2(a, b Scmer) Scmer {
	values := []Scmer{a, b}
	return jitNewSliceResult(values, 2)
}

func jitNewSlice4(a, b, c, d Scmer) Scmer {
	values := []Scmer{a, b, c, d}
	return jitNewSliceResult(values, 4)
}

// Go's amd64 ABI has nine integer argument registers. Wider projection rows
// keep their first four Scmers in those registers and pass the remaining
// stack-backed values through the ninth register as a single pointer.
func jitNewSlice6(a, b, c, d Scmer, tail *[2]Scmer) Scmer {
	values := []Scmer{a, b, c, d, tail[0], tail[1]}
	return jitNewSliceResult(values, 6)
}

func jitNewSlice8(a, b, c, d Scmer, tail *[4]Scmer) Scmer {
	values := []Scmer{a, b, c, d, tail[0], tail[1], tail[2], tail[3]}
	return jitNewSliceResult(values, 8)
}

func jitDirectSliceBuilder(length int) uint64 {
	switch length {
	case 2:
		return GoFuncAddr(jitNewSlice2)
	case 4:
		return GoFuncAddr(jitNewSlice4)
	case 6:
		return GoFuncAddr(jitNewSlice6)
	case 8:
		return GoFuncAddr(jitNewSlice8)
	default:
		return 0
	}
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

func (ctx *JITContext) EmitNewSliceFromGoSlice(slice *JITValueDesc) JITValueDesc {
	ctx.SyncDesc(slice)
	// A Go slice assembled by generated code may still use an invocation-frame
	// array (for example after a transferred produce/filter pipeline). NewSlice
	// is an escaping Scheme value, so materialize independent backing here.
	// Under nested parser/callback pressure, return directly into a spill slot.
	// The consumer can copy that Scmer into its final target without reserving a
	// transient register pair merely to cross this helper-call boundary.
	if bits.OnesCount64(ctx.FreeRegs&ctx.AllRegs&^ctx.ProtectedRegs) < 2 {
		off := ctx.AllocSpill(16)
		var wordsBuf [16]goCallArgWord
		words := ctx.flattenArgs([]JITValueDesc{*slice}, &wordsBuf)
		ctx.EmitGoCallToFrame(GoFuncAddr(JITNewSliceCopy), words, []int32{off, off + 8})
		ctx.setStackPointer(jitStackRootFrameBP, off, true)
		ctx.FreeDesc(slice)
		return JITValueDesc{Loc: LocStackPair, Type: tagSlice, StackOff: off, Rooted: true}
	}
	result := ctx.EmitGoCallScalar(GoFuncAddr(JITNewSliceCopy), []JITValueDesc{*slice}, 2)
	ctx.FreeDesc(slice)
	result.Type = tagSlice
	result.Rooted = true
	return result
}

// jitMaterializeVirtualSlice lowers the virtual variadic array produced from
// List's Go SSA. A normal list always gets fresh backing storage. Ownership is
// a property of the list builtin, never of the general Proc call ABI. Only the
// internal !list form may borrow invocation-frame storage under its
// optimizer-proven NoEscape scope.
func jitMaterializeVirtualSlice(ctx *JITContext, virtual JITValueDesc, result JITValueDesc) JITValueDesc {
	if virtual.Loc != LocVirtualSlice {
		panic("jit: expected virtual variadic slice")
	}
	stable := make([]JITValueDesc, len(virtual.Virtual))
	for i := range virtual.Virtual {
		src := virtual.Virtual[i]
		ctx.syncDescSpill(&src)
		if src.Loc != LocRegPair && src.Loc != LocStackPair && src.Loc != LocInputPair {
			ctx.EnsureDesc(&src)
			target := jitAllocTrackedPair(ctx, JITTypeUnknown)
			src = jitPlaceIntoPair(ctx, &src, target)
		}
		// Spill each producer as soon as it is complete. Descriptor spill metadata
		// preserves aliases used by later arguments while keeping register pressure
		// independent of the virtual list length.
		stable[i] = ctx.stabilizeForNested(src)
	}
	if builder := jitDirectSliceBuilder(len(stable)); builder != 0 {
		callArgs := stable
		tailBytes := int32(0)
		var tail JITValueDesc
		var wideCallArgs [5]JITValueDesc
		if len(stable) > 4 {
			tailBytes = int32((len(stable) - 4) * 16)
			tailOff := ctx.AllocStack(tailBytes)
			for i := 4; i < len(stable); i++ {
				ctx.EmitStoreScmerToStack(stable[i], tailOff+int32((i-4)*16))
				ctx.FreeDesc(&stable[i])
			}
			tailReg := ctx.AllocReg()
			ctx.EmitLeaRegMem(tailReg, ctx.StackReg, tailOff)
			tail = JITValueDesc{Loc: LocReg, Reg: tailReg, RelocatablePointer: true, Rooted: true}
			ctx.BindReg(tailReg, &tail)
			copy(wideCallArgs[:4], stable[:4])
			wideCallArgs[4] = tail
			callArgs = wideCallArgs[:]
		}
		materialized := ctx.EmitGoCallScalar(builder, callArgs, 2)
		ctx.FreeDesc(&tail)
		for i := range stable {
			ctx.FreeDesc(&stable[i])
		}
		if tailBytes != 0 {
			ctx.FreeStack(tailBytes)
		}
		materialized.Type = tagSlice
		materialized.Rooted = true
		return jitPlaceScmerIntoTarget(ctx, materialized, result)
	}
	backingOff := ctx.AllocStack(int32(len(stable) * 16))
	for i := range stable {
		ctx.EmitStoreScmerToStack(stable[i], backingOff+int32(i*16))
		ctx.FreeDesc(&stable[i])
	}
	ptrReg := ctx.AllocReg()
	lenReg := ctx.AllocRegExcept(ptrReg)
	capReg := ctx.AllocRegExcept(ptrReg, lenReg)
	ctx.EmitLeaRegMem(ptrReg, ctx.StackReg, backingOff)
	ctx.EmitMovRegImm64(lenReg, uint64(len(stable)))
	ctx.EmitMovRegImm64(capReg, uint64(len(stable)))
	header := JITValueDesc{Loc: LocRegTriple, Type: tagSlice, Reg: ptrReg, Reg2: lenReg, Reg3: capReg, Rooted: true}
	ctx.BindReg(ptrReg, &header)
	ctx.BindReg(lenReg, &header)
	ctx.BindReg(capReg, &header)
	if bits.OnesCount64(ctx.FreeRegs&ctx.AllRegs&^ctx.ProtectedRegs) < 2 {
		off := ctx.AllocSpill(16)
		var wordsBuf [16]goCallArgWord
		words := ctx.flattenArgs([]JITValueDesc{header}, &wordsBuf)
		ctx.EmitGoCallToFrame(GoFuncAddr(JITNewSliceCopy), words, []int32{off, off + 8})
		ctx.setStackPointer(jitStackRootFrameBP, off, true)
		ctx.FreeDesc(&header)
		materialized := JITValueDesc{Loc: LocStackPair, Type: tagSlice, StackOff: off, Rooted: true}
		return jitPlaceScmerIntoTarget(ctx, materialized, result)
	}
	materialized := ctx.EmitGoCallScalar(GoFuncAddr(JITNewSliceCopy), []JITValueDesc{header}, 2)
	ctx.FreeDesc(&header)
	materialized.Type = tagSlice
	materialized.Rooted = true
	return jitPlaceScmerIntoTarget(ctx, materialized, result)
}

// jitMaterializeVirtualGoSlice turns compile-time argument descriptions into a
// runtime []Scmer header. JITGen uses it whenever a Go builtin needs an actual
// slice (for example append) instead of merely indexing the virtual arguments.
func jitMaterializeVirtualGoSlice(ctx *JITContext, elements []JITValueDesc) JITValueDesc {
	length := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(elements))), NoHeapPointer: true}
	results := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{length, length}, []uint8{3}, []uint8{1})
	header := results[0]
	header.Type = tagSlice
	for i := range elements {
		index := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(i)), NoHeapPointer: true}
		address := ctx.EmitSliceElementAddress(&header, &index, 16)
		value := elements[i]
		value.ID = 0 // caller-owned placement; storing must not transfer ownership
		ctx.EmitStoreScmerAt(&address, &value)
		ctx.FreeDesc(&address)
	}
	return header
}

func jitCondToBool(ctx *JITContext, cond *JITValueDesc) JITValueDesc {
	return ctx.EmitBoolDesc(cond, JITValueDesc{Loc: LocAny})
}

// jitCompileCondition gives an arbitrarily large child CFG a stable producer
// target before lowering Scheme truthiness. This is shared by every special
// form so nested map/reduce emitters see the same register-bank contract.
func jitCompileCondition(ctx *JITContext, expr Scmer, sliceBase Reg) JITValueDesc {
	if ctx.StackPhiTargets {
		target := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: ctx.AllocStack(16), Rooted: true}
		condition := jitCompileExpr(ctx, expr, sliceBase, target)
		return jitCondToBool(ctx, &condition)
	}
	target := jitAllocTrackedPair(ctx, JITTypeUnknown)
	ctx.ProtectReg(target.Reg)
	ctx.ProtectReg(target.Reg2)
	condition := jitCompileExpr(ctx, expr, sliceBase, target)
	ctx.UnprotectReg(target.Reg2)
	ctx.UnprotectReg(target.Reg)
	return jitCondToBool(ctx, &condition)
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
	if count < 0 || len(list) != count+3 || start < 0 || ctx.Env == nil || start+count > len(ctx.Env.Numbered) {
		panic("jit: !list slots outside invocation frame")
	}
	backingOff := int32(0)
	for i := 0; i < count; i++ {
		slot := ctx.Env.Numbered[start+i]
		ctx.SyncDesc(&slot)
		if slot.Loc != LocStackPair || (i > 0 && slot.StackOff != backingOff+int32(i*16)) {
			panic("jit: !list requires contiguous invocation-frame slots")
		}
		if i == 0 {
			backingOff = slot.StackOff
		}
	}
	for i := 0; i < count; i++ {
		value := jitCompileExpr(ctx, list[i+3], sliceBase, JITValueDesc{Loc: LocAny})
		ctx.EnsureDesc(&value)
		if value.Loc != LocRegPair && value.Loc != LocImm {
			ptrReg := ctx.AllocReg()
			target := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ptrReg, Reg2: ctx.AllocRegExcept(ptrReg)}
			value = jitPlaceIntoPair(ctx, &value, target)
		}
		ctx.EmitStoreScmerToStack(value, backingOff+int32(i*16))
		ctx.FreeDesc(&value)
	}
	target := jitEnsureResultPair(ctx, result)
	ctx.EmitLeaRegMem(target.Reg, ctx.StackReg, backingOff)
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
	var ptrReg, lenReg, capReg Reg
	if value.Loc == LocRegPair {
		// Header extraction needs three scalar registers. Canonicalize the source
		// pair once so those registers can be reused and subsequent consumers can
		// address the same stable stack value directly.
		jitParkCallArgument(ctx, value)
	} else if value.Loc == LocInputPair {
		off := ctx.AllocStack(16)
		ctx.EmitStoreScmerToStack(*value, off)
		value.Loc, value.StackOff = LocStackPair, off
		value.Rooted = true
	}
	if value.Loc == LocImm {
		ptrReg = ctx.AllocReg()
		lenReg = ctx.AllocRegExcept(ptrReg)
		capReg = ctx.AllocRegExcept(ptrReg, lenReg)
		ctx.TrackImm(value.Imm)
		ptrWord, auxWord := value.Imm.RawWords()
		length, capacity := decodeSliceAux(auxWord)
		ctx.EmitMovRegImm64(ptrReg, uint64(ptrWord))
		ctx.EmitMovRegImm64(lenReg, uint64(length))
		ctx.EmitMovRegImm64(capReg, uint64(capacity))
	} else if value.Loc == LocStackPair {
		ctx.ReclaimUntrackedRegs()
		ptrReg = ctx.AllocReg()
		lenReg = ctx.AllocRegExcept(ptrReg)
		capReg = ctx.AllocRegExcept(ptrReg, lenReg)
		base := ctx.StackReg
		if value.StackOff < 0 {
			base = ctx.FrameReg
		}
		ctx.EmitMovRegMem(ptrReg, base, value.StackOff)
		ctx.EmitMovRegMem(lenReg, base, value.StackOff+8)
		ctx.EmitShrRegImm8(lenReg, 8+sliceCapBits)
		ctx.EmitMovRegMem(capReg, base, value.StackOff+8)
		ctx.EmitShrRegImm8(capReg, 8)
		ctx.EmitAndRegImm32(capReg, int32(sliceCapMask))
	} else {
		ctx.EnsureDesc(value)
		if value.Loc != LocRegPair {
			panic("jit: known slice is not a Scmer pair")
		}
		// The output allocation may spill unrelated values, but it must not spill
		// the two input words before their register identities have been embedded
		// in the emitted moves below.
		ctx.ProtectReg(value.Reg)
		ctx.ProtectReg(value.Reg2)
		ptrReg = ctx.AllocRegExcept(value.Reg, value.Reg2)
		lenReg = ctx.AllocRegExcept(value.Reg, value.Reg2, ptrReg)
		capReg = ctx.AllocRegExcept(value.Reg, value.Reg2, ptrReg, lenReg)
		ctx.EmitMovRegReg(ptrReg, value.Reg)
		ctx.EmitMovRegReg(lenReg, value.Reg2)
		ctx.EmitShrRegImm8(lenReg, 8+sliceCapBits)
		ctx.EmitMovRegReg(capReg, value.Reg2)
		ctx.EmitShrRegImm8(capReg, 8)
		ctx.EmitAndRegImm32(capReg, int32(sliceCapMask))
		ctx.UnprotectReg(value.Reg)
		ctx.UnprotectReg(value.Reg2)
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
		ctx.EnsureDesc(cond)
		var valReg Reg
		var sourceRegs []Reg
		switch cond.Loc {
		case LocReg:
			valReg = cond.Reg
			sourceRegs = append(sourceRegs, cond.Reg)
		case LocRegPair:
			valReg = cond.Reg2
			sourceRegs = append(sourceRegs, cond.Reg, cond.Reg2)
		default:
			panic("jit: borrowed bool test needs register value")
		}
		for _, reg := range sourceRegs {
			ctx.ProtectReg(reg)
		}
		dst := ctx.AllocReg()
		if dst != valReg {
			ctx.EmitMovRegReg(dst, valReg)
		}
		if cond.Type == tagFloat {
			mask := ctx.AllocRegExcept(dst)
			ctx.EmitMovRegImm64(mask, 0x7fffffffffffffff)
			ctx.EmitAndInt64(dst, mask)
			ctx.FreeReg(mask)
		} else if cond.Type == tagBool && cond.Loc == LocRegPair {
			// Bool payload is auxVal in bits [63:8]; low 8 bits hold the tag.
			ctx.EmitShrRegImm8(dst, 8)
		}
		ctx.EmitCmpRegImm32(dst, 0)
		ctx.EmitSetcc(dst, CondNotEqual)
		for _, reg := range sourceRegs {
			ctx.UnprotectReg(reg)
		}
		result := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: dst}
		ctx.BindReg(dst, &result)
		return result
	}

	ctx.EnsureDesc(cond)
	sourceRegs := jitDescRegs(*cond)
	for _, reg := range sourceRegs {
		ctx.ProtectReg(reg)
	}
	out := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Bool), []JITValueDesc{*cond}, 1)
	for _, reg := range sourceRegs {
		ctx.UnprotectReg(reg)
	}
	ctx.EmitAndRegImm32(out.Reg, 1)
	out.Type = tagBool
	ctx.BindReg(out.Reg, &out)
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
	ctx.ReclaimUntrackedRegs()
	ctx.EnsureDesc(v)
	sourceRegs := jitDescRegs(*v)
	for _, reg := range sourceRegs {
		ctx.ProtectReg(reg)
	}
	if v.Loc != LocRegPair {
		out := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.IsNil), []JITValueDesc{*v}, 1)
		for _, reg := range sourceRegs {
			ctx.UnprotectReg(reg)
		}
		ctx.EmitAndRegImm32(out.Reg, 1)
		out.Type = tagBool
		ctx.BindReg(out.Reg, &out)
		return out
	}
	tagReg := ctx.AllocReg()
	ctx.EmitGetTagRegs(tagReg, v.Reg, v.Reg2)
	ctx.EmitCmpRegImm8(tagReg, tagNil)
	ctx.EmitSetcc(tagReg, CondEqual)
	for _, reg := range sourceRegs {
		ctx.UnprotectReg(reg)
	}
	result := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: tagReg}
	ctx.BindReg(tagReg, &result)
	return result
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

func jitEmitGoVariadicCallFromExprs(ctx *JITContext, fn func(...Scmer) Scmer, argExprs []Scmer, sliceBase Reg, result JITValueDesc, retainsArgs bool) JITValueDesc {
	return jitEmitDeclaredGoVariadicCallFromExprs(ctx, fn, nil, argExprs, sliceBase, result, retainsArgs)
}

func jitEmitDeclaredGoVariadicCallFromExprs(ctx *JITContext, fn func(...Scmer) Scmer, decl *Declaration, argExprs []Scmer, sliceBase Reg, result JITValueDesc, retainsArgs bool) JITValueDesc {
	argc := len(argExprs)
	stackStart := ctx.BPOffset
	argsOff := int32(0)
	if argc > 0 {
		argsOff = ctx.AllocStack(int32(argc * 16))
		for i := range argExprs {
			expected := JITValueDesc{Loc: LocAny}
			if param := jitDeclarationParam(decl, i); param != nil && param.Kind == "func" && param.NoEscape && param.SameGoroutine {
				expected.StackFunc = true
			}
			jitCompileRootedCallValueAtResult(ctx, argExprs[i], sliceBase, argsOff+int32(i*16), expected)
		}
		// Every argument is stable in the frame now. Argument emitters may have
		// consumed temporary registers without descriptor ownership; none remain
		// live across the variadic call boundary.
		ctx.ReclaimUntrackedRegs()
	}
	var argsSlice, heapArgs JITValueDesc
	stackBytes := int32(argc * 16)
	if retainsArgs {
		elements := make([]JITValueDesc, argc)
		for i := range elements {
			elements[i] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: argsOff + int32(i*16), Rooted: true}
		}
		heapArgs = jitMaterializeVirtualGoSlice(ctx, elements)
		ctx.EnsureDesc(&heapArgs)
		argsSlice = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: heapArgs.Reg, Reg2: heapArgs.Reg2, Rooted: true}
	} else if argc > 0 {
		// Captures and other lexical values live in the static JIT frame. Compile
		// them there before moving the stack pointer, then copy them into the contiguous
		// variadic call area. Reading LocStackPair directly after reserving call space would
		// otherwise address the temporary call area instead of the lexical slot.
		ctx.EmitReserveStackBytes(stackBytes)
		for i := range argExprs {
			slotOff := int32(i * 16)
			sourceOff := stackBytes + argsOff + int32(i*16)
			ctx.EmitMovRegMem(ctx.ScratchReg, ctx.StackReg, sourceOff)
			ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, slotOff)
			ctx.EmitMovRegMem(ctx.ScratchReg, ctx.StackReg, sourceOff+8)
			ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, slotOff+8)
			ctx.setStackPointer(jitStackRootFrameSP, slotOff-ctx.DynamicSP, true)
		}
		// argslice: ptr + len (cap = len inside EmitGoCallVariadic).
		argsSlice = jitAllocTrackedPair(ctx, JITTypeUnknown)
		ctx.EmitMovRegReg(argsSlice.Reg, ctx.StackReg)
		ctx.EmitMovRegImm64(argsSlice.Reg2, uint64(argc))
	} else {
		argsSlice = jitAllocTrackedPair(ctx, JITTypeUnknown)
		ctx.EmitMovRegImm64(argsSlice.Reg, 0)
		ctx.EmitMovRegImm64(argsSlice.Reg2, 0)
	}

	out := ctx.EmitGoCallVariadic(fn, argsSlice, result)
	if retainsArgs {
		ctx.FreeDesc(&heapArgs)
	} else {
		ctx.FreeDesc(&argsSlice)
	}
	if !retainsArgs && stackBytes != 0 {
		ctx.EmitReleaseStackBytes(stackBytes)
		if ctx.SliceBaseTracksRSP && ctx.SliceBase != ctx.StackReg {
			ctx.EmitMovRegReg(ctx.SliceBase, ctx.StackReg)
		}
	}
	ctx.FreeStack(ctx.BPOffset - stackStart)
	return out
}

func jitEmitGoVariadicCallFromDescs(ctx *JITContext, fn func(...Scmer) Scmer, values []JITValueDesc, result JITValueDesc) JITValueDesc {
	argc := len(values)
	stackStart := ctx.BPOffset
	argsOff := int32(0)
	if argc > 0 {
		argsOff = ctx.AllocStack(int32(argc * 16))
		for i := range values {
			value := values[i]
			ctx.EnsureDesc(&value)
			if value.Loc != LocImm && value.Loc != LocRegPair && value.Loc != LocStackPair && value.Loc != LocInputPair {
				pair := jitAllocTrackedPair(ctx, value.Type)
				value = jitPlaceIntoPair(ctx, &value, pair)
			}
			ctx.EmitStoreScmerToStack(value, argsOff+int32(i*16))
			ctx.FreeDesc(&value)
		}
	}
	argsSlice := jitAllocTrackedPair(ctx, JITTypeUnknown)
	if argc > 0 {
		ctx.EmitLeaRegMem(argsSlice.Reg, ctx.StackReg, argsOff)
		ctx.EmitMovRegImm64(argsSlice.Reg2, uint64(argc))
	} else {
		ctx.EmitMovRegImm64(argsSlice.Reg, 0)
		ctx.EmitMovRegImm64(argsSlice.Reg2, 0)
	}
	out := ctx.EmitGoCallVariadic(fn, argsSlice, result)
	ctx.FreeDesc(&argsSlice)
	ctx.FreeStack(ctx.BPOffset - stackStart)
	return out
}

func jitCompileRootedCallValueAt(ctx *JITContext, expr Scmer, sliceBase Reg, off int32) JITValueDesc {
	return jitCompileRootedCallValueAtResult(ctx, expr, sliceBase, off, JITValueDesc{Loc: LocAny})
}

func jitCompileRootedCallValueAtResult(ctx *JITContext, expr Scmer, sliceBase Reg, off int32, result JITValueDesc) JITValueDesc {
	value := jitCompileExpr(ctx, expr, sliceBase, result)
	// Input values remain reachable through the caller's argument slice for the
	// complete native invocation. The safepoint map relocates a saved input
	// pointer if a callback grows the goroutine stack.
	if value.Loc == LocInputPair {
		value.Rooted = true
	}
	pair := value
	if pair.Loc != LocImm && pair.Loc != LocRegPair && pair.Loc != LocStackPair && pair.Loc != LocInputPair {
		pair = jitAllocTrackedPair(ctx, JITTypeUnknown)
		pair = jitPlaceIntoPair(ctx, &value, pair)
	}
	if pair.Loc == LocRegPair && pair.ID == 0 {
		ctx.BindReg(pair.Reg, &pair)
		ctx.BindReg(pair.Reg2, &pair)
	}
	pair = jitRootScmer(ctx, pair)
	ctx.EmitStoreScmerToStack(pair, off)
	valueType := pair.Type
	noHeapPointer := pair.NoHeapPointer
	rooted := pair.Rooted
	ctx.FreeDesc(&pair)
	return JITValueDesc{Loc: LocStackPair, Type: valueType, StackOff: off, NoHeapPointer: noHeapPointer, Rooted: rooted}
}

func jitCompileRootedCallValue(ctx *JITContext, expr Scmer, sliceBase Reg) JITValueDesc {
	off := ctx.AllocStack(16)
	return jitCompileRootedCallValueAt(ctx, expr, sliceBase, off)
}

func jitVisibleSymbols(ctx *JITContext) []Symbol {
	seen := make(map[Symbol]struct{})
	symbols := make([]Symbol, 0)
	for env := ctx.Env; env != nil; env = env.Outer {
		for symbol := range env.Vars {
			if _, exists := seen[symbol]; exists {
				continue
			}
			seen[symbol] = struct{}{}
			symbols = append(symbols, symbol)
		}
	}
	sort.Slice(symbols, func(i, j int) bool { return symbols[i] < symbols[j] })
	return symbols
}

func jitRuntimeCaptureArgExprs(ctx *JITContext) []Scmer {
	args := []Scmer{ctx.RuntimeEnv}
	depth := 0
	for env := ctx.Env; env != nil; env = env.Outer {
		args = append(args, NewAny(jitRuntimeEnvFrameMarker), NewNil())
		symbols := make([]Symbol, 0, len(env.Vars))
		for symbol := range env.Vars {
			symbols = append(symbols, symbol)
		}
		sort.Slice(symbols, func(i, j int) bool { return symbols[i] < symbols[j] })
		for _, symbol := range symbols {
			key := NewSymbol(string(symbol))
			value := Scmer(key)
			if depth > 0 {
				value = NewSlice([]Scmer{NewSymbol("outer"), NewInt(int64(depth)), value})
			}
			args = append(args, NewSlice([]Scmer{NewSymbol("quote"), key}), value)
		}
		for index := range env.Numbered {
			key := NewNthLocalVar(NthLocalVar(index))
			value := Scmer(key)
			if depth > 0 {
				value = NewSlice([]Scmer{NewSymbol("outer"), NewInt(int64(depth)), value})
			}
			args = append(args, NewSlice([]Scmer{NewSymbol("quote"), key}), value)
		}
		depth++
	}
	if ctx.Env == nil {
		args = append(args, NewAny(jitRuntimeEnvFrameMarker), NewNil())
	}
	return args
}

func jitCompileSpecialThunk(ctx *JITContext, body Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	symbols := jitVisibleSymbols(ctx)
	params := make([]Scmer, 0, ctx.LocalSlotCount+len(symbols))
	argExprs := make([]Scmer, 0, 1+ctx.LocalSlotCount+len(symbols))
	for index := 0; index < ctx.LocalSlotCount; index++ {
		params = append(params, NewSymbol(fmt.Sprintf("\x00jit-slot-%d", index)))
		argExprs = append(argExprs, NewNthLocalVar(NthLocalVar(index)))
	}
	for _, symbol := range symbols {
		params = append(params, NewSymbol(string(symbol)))
		argExprs = append(argExprs, NewSymbol(string(symbol)))
	}
	outer, ok := ctx.RuntimeEnv.Any().(*Env)
	if !ok || outer == nil {
		panic("jit: invalid special-form thunk environment")
	}
	callable := jitCompileModeDeferred(true, NewProcStruct(Proc{
		Params:  NewSlice(params),
		Body:    body,
		En:      outer,
		NumVars: len(params),
	}))
	if callable.GetTag() != tagProc || callable.Proc() == nil || callable.Proc().Compiled == nil {
		panic("jit: special-form child did not compile")
	}
	argExprs = append([]Scmer{callable}, argExprs...)
	return jitEmitGoVariadicCallFromExprs(ctx, jitMakeSpecialThunk, argExprs, sliceBase, result, false)
}

func jitStaticProcForExpr(ctx *JITContext, expr Scmer) (Scmer, *Proc, bool) {
	for expr.GetTag() == tagSourceInfo {
		expr = expr.SourceInfo().value
	}
	if expr.GetTag() == tagProc {
		proc := expr.Proc()
		if proc != nil && proc.JITCode == 0 && atomic.LoadUint32(&proc.jitCompiling) == 0 {
			expr = jitCompileModeDeferred(true, expr)
			proc = expr.Proc()
		}
		return expr, proc, proc != nil && proc.JITCode != 0 && proc.Compiled != nil && proc.Compiled.JITDirect != 0
	}
	if expr.GetTag() != tagSymbol {
		return Scmer{}, nil, false
	}
	symbol := expr.Symbol()
	if ctx.Env != nil {
		if value, exists := ctx.Env.Lookup(symbol); exists {
			if value.Loc == LocImm && value.Imm.GetTag() == tagProc {
				proc := value.Imm.Proc()
				if proc != nil && proc.JITCode == 0 && atomic.LoadUint32(&proc.jitCompiling) == 0 {
					value.Imm = jitCompileModeDeferred(true, value.Imm)
					proc = value.Imm.Proc()
				}
				return value.Imm, proc, proc != nil && proc.JITCode != 0 && proc.Compiled != nil && proc.Compiled.JITDirect != 0
			}
			return Scmer{}, nil, false
		}
	}
	if runtimeEnv, ok := ctx.RuntimeEnv.Any().(*Env); ok && runtimeEnv != nil {
		if binding := runtimeEnv.FindRead(symbol); binding != nil && binding != &Globalenv {
			return Scmer{}, nil, false
		}
	}
	value, exists := Globalenv.Vars[symbol]
	if !exists || value.GetTag() != tagProc {
		return Scmer{}, nil, false
	}
	proc := value.Proc()
	if proc != nil && proc.JITCode == 0 && atomic.LoadUint32(&proc.jitCompiling) == 0 {
		value = jitCompileModeDeferred(true, value)
		proc = value.Proc()
	}
	return value, proc, proc != nil && proc.JITCode != 0 && proc.Compiled != nil && proc.Compiled.JITDirect != 0
}

func jitCompileStaticProcCall(ctx *JITContext, callable Scmer, proc *Proc, operands []Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	stackStart := ctx.BPOffset
	operandOff := ctx.AllocStack(int32(len(operands) * 16))
	for index, operand := range operands {
		jitCompileRootedCallValueAt(ctx, operand, sliceBase, operandOff+int32(index*16))
	}
	ctx.TrackImm(callable)
	ctx.TrackEntry(proc.Compiled)
	fnReg := ctx.AllocReg()
	ctx.EmitMovRegImm64(fnReg, uint64(uintptr(unsafe.Pointer(proc))))
	fnValue := JITValueDesc{Loc: LocReg, Type: tagFunc, Reg: fnReg, RelocatablePointer: true}
	ctx.BindReg(fnReg, &fnValue)
	argsPtr := ctx.AllocRegExcept(fnReg)
	argsLen := ctx.AllocRegExcept(fnReg, argsPtr)
	argsSlice := JITValueDesc{Loc: LocRegPair, Type: tagSlice, Reg: argsPtr, Reg2: argsLen}
	ctx.BindReg(argsPtr, &argsSlice)
	ctx.BindReg(argsLen, &argsSlice)
	if len(operands) == 0 {
		ctx.EmitMovRegImm64(argsPtr, 0)
	} else {
		ctx.EmitLeaRegMem(argsPtr, ctx.StackReg, operandOff)
	}
	ctx.EmitMovRegImm64(argsLen, uint64(len(operands)))
	resultOff := ctx.AllocSpill(16)
	callResult := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: resultOff, Rooted: true}
	ctx.EmitZeroDescWords(&callResult, 2)
	ctx.setStackPointer(jitStackRootFrameBP, resultOff, true)
	ctx.EmitProcCall(fnValue, argsSlice, callResult)
	ctx.FreeDesc(&fnValue)
	ctx.FreeDesc(&argsSlice)
	ctx.Coverage.DirectProcs++
	out := jitPlaceScmerIntoTarget(ctx, callResult, result)
	ctx.FreeStack(ctx.BPOffset - stackStart)
	return out
}

func jitCompileDynamicCall(ctx *JITContext, callableExpr Scmer, operands []Scmer, decl *Declaration, sliceBase Reg, result JITValueDesc) JITValueDesc {
	if parser, ok := jitStaticParserForExpr(ctx, callableExpr); ok {
		if len(operands) != 1 {
			panic("jit: parser call expects one input")
		}
		input := jitCompileRootedCallValue(ctx, operands[0], sliceBase)
		return jitEmitStaticParser(ctx, parser, input, result)
	}
	if parser, ok := jitParserTemplateForExpr(ctx, callableExpr); ok {
		if len(operands) != 1 {
			panic("jit: parser call expects one input")
		}
		input := jitCompileRootedCallValue(ctx, operands[0], sliceBase)
		return jitEmitParserTemplate(ctx, parser, input, result)
	}
	if callable, proc, ok := jitStaticProcForExpr(ctx, callableExpr); ok && (proc.Compiled.JITArity < 0 || proc.Compiled.JITArity == len(operands)) {
		return jitCompileStaticProcCall(ctx, callable, proc, operands, sliceBase, result)
	}
	ctx.Coverage.DynamicCalls++
	stackStart := ctx.BPOffset
	callable := jitCompileRootedCallValue(ctx, callableExpr, sliceBase)
	operandOff := ctx.AllocStack(int32(len(operands) * 16))
	operandValues := make([]JITValueDesc, len(operands))
	for index, operand := range operands {
		expected := JITValueDesc{Loc: LocAny}
		if param := jitDeclarationParam(decl, index); param != nil && param.Kind == "func" && param.NoEscape && param.SameGoroutine {
			expected.StackFunc = true
		}
		operandValues[index] = jitCompileRootedCallValueAtResult(ctx, operand, sliceBase, operandOff+int32(index*16), expected)
	}
	out := jitEmitDynamicCallableAt(ctx, callable, operandValues, operandOff, result)
	ctx.FreeStack(ctx.BPOffset - stackStart)
	return out
}

// jitEmitDynamicCallableAt lowers a runtime callable whose arguments already
// occupy a contiguous invocation-frame array. Both Scheme procedures and Go
// funcvals are called directly; only unsupported callable kinds retain the
// interpreter boundary. Generated higher-order builtins use this entry point
// so their dynamic callback path gets the same stack-aware dispatch as an
// ordinary Scheme call.
func jitEmitDynamicCallableAt(ctx *JITContext, callable JITValueDesc, operandValues []JITValueDesc, operandOff int32, result JITValueDesc) JITValueDesc {
	resultOff := ctx.AllocSpill(16)
	callResult := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: resultOff, Rooted: true}
	ctx.EmitZeroDescWords(&callResult, 2)
	ctx.setStackPointer(jitStackRootFrameBP, resultOff, true)
	// Every dispatch arm must start from the same register state. In particular,
	// generated higher-order builtins can enter here while most registers hold
	// loop state. Keep the callable in a branch-stable home, then restore this
	// snapshot before emitting each mutually exclusive arm.
	ctx.StabilizeDescAcrossNestedCall(&callable)
	fallbackLabel := ctx.ReserveLabel()
	endLabel := ctx.ReserveLabel()

	callableValue := callable
	ctx.EnsureDesc(&callableValue)
	if callableValue.Loc != LocRegPair {
		panic("jit: dynamic callable must be a Scmer pair")
	}
	tag := ctx.AllocRegExcept(callableValue.Reg, callableValue.Reg2)
	ctx.EmitGetTagRegs(tag, callableValue.Reg, callableValue.Reg2)
	nativeFuncLabel := ctx.ReserveLabel()
	ctx.EmitCmpRegImm8(tag, tagFunc)
	ctx.EmitJcc(CcE, nativeFuncLabel)
	ctx.EmitCmpRegImm8(tag, tagProc)
	ctx.FreeReg(tag)
	ctx.EmitJcc(CcNE, fallbackLabel)
	ctx.FreeDesc(&callableValue)
	dispatchState := ctx.SnapshotAllocState()

	callableValue = callable
	ctx.EnsureDesc(&callableValue)
	// Scheme procedures need a direct, arity-compatible JIT entry. Their *Proc
	// pointer is already the funcval context consumed by EmitProcCall.
	metadata := ctx.AllocRegExcept(callableValue.Reg, callableValue.Reg2)
	ctx.EmitMovRegMem(metadata, callableValue.Reg, int32(unsafe.Offsetof(Proc{}.Compiled)))
	ctx.EmitCmpRegImm32(metadata, 0)
	ctx.EmitJcc(CcE, fallbackLabel)
	direct := ctx.AllocRegExcept(callableValue.Reg, callableValue.Reg2, metadata)
	ctx.EmitMovRegMem(direct, metadata, int32(unsafe.Offsetof(JITEntryPoint{}.JITDirect)))
	ctx.EmitCmpRegImm32(direct, 0)
	ctx.FreeReg(direct)
	ctx.EmitJcc(CcE, fallbackLabel)
	arityOK := ctx.ReserveLabel()
	arity := ctx.AllocRegExcept(callableValue.Reg, callableValue.Reg2, metadata)
	ctx.EmitMovRegMem(arity, metadata, int32(unsafe.Offsetof(JITEntryPoint{}.JITArity)))
	ctx.EmitCmpRegImm32(arity, int32(len(operandValues)))
	ctx.EmitJcc(CcE, arityOK)
	ctx.EmitCmpRegImm32(arity, -1)
	ctx.EmitJcc(CcNE, fallbackLabel)
	ctx.MarkLabel(arityOK)
	ctx.FreeReg(arity)
	ctx.FreeReg(metadata)
	codeReg := ctx.AllocRegExcept(callableValue.Reg, callableValue.Reg2)
	ctx.EmitMovRegMem(codeReg, callableValue.Reg, int32(unsafe.Offsetof(Proc{}.JITCode)))
	ctx.EmitCmpRegImm32(codeReg, 0)
	ctx.FreeReg(codeReg)
	ctx.EmitJcc(CcE, fallbackLabel)
	fnReg := ctx.AllocRegExcept(callableValue.Reg, callableValue.Reg2)
	ctx.EmitMovRegReg(fnReg, callableValue.Reg)
	fnValue := JITValueDesc{Loc: LocReg, Type: tagFunc, Reg: fnReg, RelocatablePointer: true}
	ctx.BindReg(fnReg, &fnValue)
	ctx.FreeDesc(&callableValue)
	argsSlice := jitEmitDynamicCallArgs(ctx, fnReg, operandOff, len(operandValues))
	ctx.EmitProcCall(fnValue, argsSlice, callResult)
	ctx.FreeDesc(&fnValue)
	ctx.FreeDesc(&argsSlice)
	ctx.Coverage.DirectProcs++
	ctx.EmitJmp(endLabel)

	ctx.RestoreAllocState(dispatchState)
	ctx.MarkLabel(nativeFuncLabel)
	nativeCallable := callable
	ctx.EnsureDesc(&nativeCallable)
	if nativeCallable.Loc != LocRegPair {
		panic("jit: dynamic native callable must be a Scmer pair")
	}
	// A tagFunc Scmer points at the Go variable holding the funcval pointer,
	// whereas a compiled Proc is itself laid out as the funcval object.
	fnReg = ctx.AllocRegExcept(nativeCallable.Reg, nativeCallable.Reg2)
	ctx.EmitMovRegMem(fnReg, nativeCallable.Reg, 0)
	// Retaining native functions need the existing owned-argument boundary.
	// Declarations populate this small identity set; the hot path remains a
	// sequence of comparisons without a Go helper or a builtin-name policy.
	for _, retainingIdentityValue := range retainingCallArgFunctionIdentities {
		retainingIdentity := ctx.AllocRegExcept(nativeCallable.Reg, nativeCallable.Reg2, fnReg)
		ctx.EmitMovRegImm64(retainingIdentity, uint64(retainingIdentityValue))
		ctx.EmitCmpInt64(fnReg, retainingIdentity)
		ctx.FreeReg(retainingIdentity)
		ctx.EmitJcc(CcE, fallbackLabel)
	}
	fnValue = JITValueDesc{Loc: LocReg, Type: tagFunc, Reg: fnReg, RelocatablePointer: true}
	ctx.BindReg(fnReg, &fnValue)
	ctx.FreeDesc(&nativeCallable)
	argsSlice = jitEmitDynamicCallArgs(ctx, fnReg, operandOff, len(operandValues))
	ctx.EmitGoFuncCall(fnValue, argsSlice, callResult)
	ctx.FreeDesc(&fnValue)
	ctx.FreeDesc(&argsSlice)
	ctx.Coverage.NativeCalls++
	ctx.EmitJmp(endLabel)

	ctx.RestoreAllocState(dispatchState)
	ctx.MarkLabel(fallbackLabel)
	args := make([]JITValueDesc, 0, len(operandValues)+2)
	args = append(args, callable)

	env := jitCurrentRuntimeEnv(ctx)
	args = append(args, env)
	if len(operandValues) <= 2 {
		args = append(args, operandValues...)
	} else {
		// Keep the variadic backing array in the invocation-local JIT frame. Its
		// Scmer pointer words are part of the precise map at the Apply callback,
		// and each pointer-bearing value is mirrored into the hidden Go root
		// slice.
		sliceOff := ctx.AllocStack(24)
		ptr := ctx.AllocReg()
		ctx.EmitLeaRegMem(ptr, ctx.StackReg, operandOff)
		ctx.EmitStoreRegMem(ptr, ctx.StackReg, sliceOff)
		ctx.FreeReg(ptr)
		ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(len(operandValues)))
		ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, sliceOff+8)
		ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, sliceOff+16)
		ctx.setStackPointer(jitStackRootFrameSP, sliceOff, true)
		args = append(args, JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: sliceOff})
	}

	var fn any
	switch len(operandValues) {
	case 0:
		fn = jitApplyCallable0
	case 1:
		fn = jitApplyCallable1
	case 2:
		fn = jitApplyCallable2
	default:
		fn = jitApplyCallableSlice
	}
	fallbackResult := jitAllocTrackedPair(ctx, JITTypeUnknown)
	fallbackResult = ctx.EmitGoCallScalarInto(GoFuncAddr(fn), args, fallbackResult)
	ctx.FreeDesc(&env)
	ctx.EmitCopyScmerToDesc(&callResult, &fallbackResult)
	ctx.FreeDesc(&fallbackResult)
	ctx.RestoreAllocState(dispatchState)
	ctx.MarkLabel(endLabel)
	var out JITValueDesc
	if result.Loc == LocStackPair {
		out = result
		ctx.EmitCopyScmerToDesc(&out, &callResult)
		out.Type = JITTypeUnknown
		out.Rooted = true
	} else {
		out = jitEnsureResultPair(ctx, result)
		out = jitPlaceIntoPair(ctx, &callResult, out)
		out.Type = JITTypeUnknown
	}
	return out
}

func jitEmitDynamicCallArgs(ctx *JITContext, fnReg Reg, operandOff int32, operandCount int) JITValueDesc {
	argsPtr := ctx.AllocRegExcept(fnReg)
	argsLen := ctx.AllocRegExcept(fnReg, argsPtr)
	argsSlice := JITValueDesc{Loc: LocRegPair, Type: tagSlice, Reg: argsPtr, Reg2: argsLen, NoHeapPointer: false}
	ctx.BindReg(argsPtr, &argsSlice)
	ctx.BindReg(argsLen, &argsSlice)
	if operandCount == 0 {
		ctx.EmitMovRegImm64(argsPtr, 0)
	} else {
		ctx.EmitLeaRegMem(argsPtr, ctx.StackReg, operandOff)
	}
	ctx.EmitMovRegImm64(argsLen, uint64(operandCount))
	return argsSlice
}

func jitCompileDynamicHigherOrderCall(ctx *JITContext, callableExpr Scmer, operands []Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	recursiveLambdas := ctx.RecursiveLambdas
	ctx.RecursiveLambdas = false
	defer func() { ctx.RecursiveLambdas = recursiveLambdas }()
	return jitCompileDynamicCall(ctx, callableExpr, operands, nil, sliceBase, result)
}

const jitBuiltinInlineBudget = 2048
const jitTrivialVirtualInlineCost = 2

// jitEmitGeneratedCallBoundary materializes compiler-only lambda templates
// only when a generated builtin emitter chooses its native call boundary.
// Other arguments were already evaluated into descriptors and must not be
// evaluated again. The declaration decides whether the resulting Proc funcval
// may live in the current JIT frame or needs one typed heap allocation.
func jitEmitGeneratedCallBoundary(ctx *JITContext, declaration *Declaration, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
	for index := range args {
		if args[index].Loc != LocLambdaTemplate {
			continue
		}
		if index >= len(sourceArgs) {
			panic("jit: generated callback has no source expression")
		}
		expected := JITValueDesc{Loc: LocAny}
		if param := jitDeclarationParam(declaration, index); param != nil && param.Kind == "func" && param.NoEscape && param.SameGoroutine {
			expected.StackFunc = true
		}
		args[index] = jitCompileExpr(ctx, sourceArgs[index], ctx.SliceBase, expected)
	}
	return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
}

// jitGeneratedEmitterInline implements the shared size/type policy used by
// generated declaration hooks. Call dispatch itself remains declaration-owned:
// the expression compiler invokes Type.JITEmit, and that hook asks this helper
// whether to render its SSA body or emit its native call boundary.
func jitGeneratedEmitterInline(ctx *JITContext, declaration *Declaration, args []JITValueDesc) bool {
	if !jitEnabled || declaration == nil || declaration.Type == nil {
		return false
	}
	inline := declaration.RetainsCallArgs
	knownTypes, knownShapes, knownArgs := 0, 0, 0
	hasVirtualArgs := false
	knownCallback, hasCallback := false, false
	for index, arg := range args {
		if arg.Type != JITTypeUnknown {
			knownTypes++
		}
		hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
		hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
		if hasKnownShape {
			knownShapes++
		}
		if arg.Type != JITTypeUnknown || hasKnownShape {
			knownArgs++
		}
		parameter := jitDeclarationParam(declaration, index)
		if parameter != nil && parameter.Kind == "func" {
			hasCallback = true
			if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
				(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
				knownCallback = true
			}
		}
	}
	cost := int(declaration.Type.JITInlineCost)
	if !inline && hasCallback {
		inline = declaration.Type.JITInlineCallbacks && knownCallback
	} else if !inline {
		switch {
		case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
			inline = true
		case declaration.Type.JITVirtualArgs && hasVirtualArgs && cost <= 32:
			inline = true
		case len(args) > 0 && knownTypes == len(args) && cost <= 256:
			inline = true
		case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
			inline = true
		}
		if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
			inline = false
		}
		if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
			inline = false
		}
	}
	if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
		return false
	}
	if !inline {
		return false
	}
	ctx.BuiltinInlineCost += cost
	ctx.Coverage.InlinedCalls++
	return true
}

func jitCompileSelfCall(ctx *JITContext, operands []Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	if ctx.SelfParamCount < 0 || len(operands) > ctx.SelfParamCount {
		panic("jit: invalid self call arity")
	}
	ctx.Coverage.InlinedCalls++
	stackStart := ctx.BPOffset
	argsOff := ctx.AllocStack(int32(ctx.SelfParamCount * 16))
	for i := 0; i < ctx.SelfParamCount; i++ {
		if i < len(operands) {
			jitCompileRootedCallValueAt(ctx, operands[i], sliceBase, argsOff+int32(i*16))
			continue
		}
		ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, argsOff+int32(i*16))
	}

	if !ctx.HasSelfLoop {
		panic("jit: recursive call has no current function value")
	}

	argsPtr := ctx.AllocReg()
	argsLen := ctx.AllocRegExcept(argsPtr)
	argsSlice := JITValueDesc{Loc: LocRegPair, Type: tagSlice, Reg: argsPtr, Reg2: argsLen}
	ctx.BindReg(argsPtr, &argsSlice)
	ctx.BindReg(argsLen, &argsSlice)
	ctx.EmitLeaRegMem(argsPtr, ctx.StackReg, argsOff)
	ctx.EmitMovRegImm64(argsLen, uint64(ctx.SelfParamCount))
	fnValue := JITValueDesc{
		Loc: LocStack, Type: tagFunc, StackOff: ctx.CurrentFuncOff,
		Rooted: true, RelocatablePointer: true,
	}
	resultOff := ctx.AllocSpill(16)
	callResult := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: resultOff, Rooted: true}
	ctx.EmitZeroDescWords(&callResult, 2)
	ctx.setStackPointer(jitStackRootFrameBP, resultOff, true)
	ctx.EmitProcCall(fnValue, argsSlice, callResult)
	ctx.FreeDesc(&argsSlice)
	target := jitPlaceScmerIntoTarget(ctx, callResult, result)
	ctx.FreeStack(ctx.BPOffset - stackStart)
	return target
}

func jitCompileRuntimeSymbol(ctx *JITContext, symbol Scmer, result JITValueDesc) JITValueDesc {
	ctx.ReclaimUntrackedRegs()
	symbolImm := JITValueDesc{Loc: LocImm, Type: tagSymbol, Imm: symbol}
	ctx.TrackImm(symbol)
	env := jitCurrentRuntimeEnv(ctx)
	symbolPair := jitAllocTrackedPair(ctx, tagSymbol)
	symbolPair = jitPlaceIntoPair(ctx, &symbolImm, symbolPair)
	target := jitEnsureResultPair(ctx, result)
	out := ctx.EmitGoCallScalarInto(GoFuncAddr(jitResolveRuntimeSymbol), []JITValueDesc{env, symbolPair}, target)
	out.Type = JITTypeUnknown
	out = jitRootScmer(ctx, out)
	ctx.FreeDesc(&env)
	ctx.FreeDesc(&symbolPair)
	return out
}

func jitCompileRuntimeGlobalSymbol(ctx *JITContext, symbol Scmer, result JITValueDesc) JITValueDesc {
	ctx.ReclaimUntrackedRegs()
	symbolImm := JITValueDesc{Loc: LocImm, Type: tagSymbol, Imm: symbol}
	ctx.TrackImm(symbol)
	symbolPair := jitAllocTrackedPair(ctx, tagSymbol)
	symbolPair = jitPlaceIntoPair(ctx, &symbolImm, symbolPair)
	target := jitEnsureResultPair(ctx, result)
	out := ctx.EmitGoCallScalarInto(GoFuncAddr(jitResolveGlobalSymbol), []JITValueDesc{symbolPair}, target)
	out.Type = JITTypeUnknown
	out = jitRootScmer(ctx, out)
	ctx.FreeDesc(&symbolPair)
	return out
}

// jitEmitCondJump emits branch code equivalent to Eval(...).Bool():
// jumps to trueLbl when expr is truthy, otherwise to falseLbl.
// It short-circuits nested (and ...)/(or ...)/(if ...) directly without
// forcing intermediate boolean materialization.
func jitEmitCondJump(ctx *JITContext, expr Scmer, sliceBase Reg, trueLbl, falseLbl JITLabel) {
	if expr.GetTag() == tagSourceInfo {
		expr = expr.SourceInfo().value
	}
	if expr.GetTag() == tagSlice {
		list := expr.Slice()
		if len(list) > 0 {
			if declaration := DeclarationForValue(list[0]); declaration != nil &&
				declaration.IsSpecialForm && declaration.Type != nil && declaration.Type.JITEmitCond != nil {
				declaration.Type.JITEmitCond(ctx, list[1:], trueLbl, falseLbl)
				return
			}
		}
	}

	// Conditions are consumed immediately, but generated multi-block emitters
	// still need a stable pair destination while rendering their internal CFG.
	// Reserving it here prevents a returned spill descriptor from becoming
	// relative to a later nested emitter's stack frame.
	b := jitCompileCondition(ctx, expr, sliceBase)
	if b.Loc == LocImm {
		if b.Imm.Bool() {
			ctx.EmitJmp(trueLbl)
		} else {
			ctx.EmitJmp(falseLbl)
		}
		return
	}
	ctx.EmitCmpRegImm32(b.Reg, 0)
	ctx.EmitJump(CondNotEqual, trueLbl)
	ctx.EmitJmp(falseLbl)
	ctx.FreeDesc(&b)
	ctx.ReclaimUntrackedRegs()
}

// jitCompileExpr recursively compiles a Scheme expression to machine code.
// sliceBase is the GPR holding the variadic args slice pointer.
// result tells the emitter where to place the output.
// Panics on unsupported expressions (caught by jitCompileExprBodyToExec).
func jitCompileExpr(ctx *JITContext, expr Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	ctx.Coverage.Expressions++
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
	case tagNil, tagBool, tagInt, tagFloat, tagDate, tagString, tagRegex, tagVector,
		tagFunc, tagFuncEnv, tagJIT, tagParser, tagFastDict, tagAny,
		tagClosure, tagPromise:
		// Keep Eval's self-evaluating literal contract. Pointer-bearing constants
		// are retained through the entry point's ConstRoots for the lifetime of
		// the generated code.
		ctx.TrackImm(expr)
		return JITValueDesc{Loc: LocImm, Type: expr.GetTag(), Imm: expr}
	case tagProc:
		if ctx.RecursiveLambdas && expr.Proc() != nil && expr.Proc().Compiled == nil {
			compiled := jitCompileModeDeferred(true, expr)
			if compiled.GetTag() != tagProc || compiled.Proc() == nil || compiled.Proc().Compiled == nil {
				panic("jit: embedded procedure could not be compiled")
			}
			expr = compiled
		}
		ctx.TrackImm(expr)
		return JITValueDesc{Loc: LocImm, Type: tagProc, Imm: expr}
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
		// A compiled closure keeps its interpreter environment in RuntimeEnv.
		// Preserve lexical shadowing before considering a same-named global for
		// static specialization; the global may only be used when it is the
		// binding the interpreter would resolve as well.
		if runtimeEnv, ok := ctx.RuntimeEnv.Any().(*Env); ok && runtimeEnv != nil {
			if binding := runtimeEnv.FindRead(sym); binding != nil && binding != &Globalenv {
				if _, exists := binding.Vars[sym]; exists {
					return jitCompileRuntimeSymbol(ctx, expr, result)
				}
			}
		}
		if v, ok := Globalenv.Vars[sym]; ok {
			switch v.GetTag() {
			case tagFunc, tagFuncEnv, tagProc, tagJIT, tagClosure, tagPromise:
				// Callable bindings stay late-bound so redefinition and future
				// specialization invalidation retain Scheme semantics.
				return jitCompileRuntimeGlobalSymbol(ctx, expr, result)
			default:
				ctx.TrackImm(v)
				return JITValueDesc{Loc: LocImm, Type: v.GetTag(), Imm: v}
			}
		}
		return jitCompileRuntimeSymbol(ctx, expr, result)
	case tagNthLocalVar:
		// Load parameter: check inline env first (JITEmitProcInline places args here).
		idx := int(expr.NthLocalVar())
		if ctx.Env != nil && idx < len(ctx.Env.Numbered) {
			src := ctx.Env.Numbered[idx]
			if src.Loc == LocParserTemplate {
				return src
			}
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
						ctx.EmitMovRegReg(result.Reg, src.Reg)
					}
					if src.Reg2 != result.Reg2 {
						ctx.EmitMovRegReg(result.Reg2, src.Reg2)
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
			case LocInputPair, LocClosurePair, LocStack, LocStackPair, LocStackTriple:
				// Preserve lazy locations across an inlined Proc boundary. The
				// consumer decides whether it needs registers; eagerly loading here
				// both loses the callback argument's real stack location and creates
				// avoidable register pressure in nested expressions.
				return src
			case LocReg:
				// Allocate a fresh register so each use is independently freeable.
				r := ctx.AllocRegExcept(src.Reg)
				ctx.EmitMovRegReg(r, src.Reg)
				d := JITValueDesc{Loc: LocReg, Type: src.Type, Reg: r}
				ctx.BindReg(r, &d)
				return d
			case LocRegPair:
				r1 := ctx.AllocRegExcept(src.Reg, src.Reg2)
				r2 := ctx.AllocRegExcept(src.Reg, src.Reg2, r1)
				ctx.EmitMovRegReg(r1, src.Reg)
				ctx.EmitMovRegReg(r2, src.Reg2)
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
		ctx.EmitMovRegMem(ptrReg, sliceBase, int32(idx*16))
		ctx.EmitMovRegMem(auxReg, sliceBase, int32(idx*16+8))
		d := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ptrReg, Reg2: auxReg}
		ctx.BindReg(ptrReg, &d)
		ctx.BindReg(auxReg, &d)
		return d
	case tagSlice:
		list := expr.Slice()
		if len(list) == 0 {
			// Match Eval: an empty slice is a self-evaluating empty list, not
			// Scheme nil. This distinction matters for typed native parameters
			// such as the scan access-values vector.
			imm := expr
			ctx.TrackImm(imm)
			return JITValueDesc{Loc: LocImm, Type: tagSlice, Imm: imm,
				KnownSliceLen: 0, KnownSliceCap: 0, SliceSizeKnown: true}
		}
		// Resolve operator
		head, headOK := scmerSymbol(list[0])
		decl := DeclarationForValue(list[0])
		if !headOK {
			// The optimizer resolves ordinary call heads to their native function
			// values. Keep using the declaration emitter in that representation as
			// well: routing an already-known variadic function through ApplyEx would
			// let it retain the JIT's temporary argument frame.
			if decl == nil || decl.IsSpecialForm {
				return jitCompileDynamicCall(ctx, list[0], list[1:], nil, sliceBase, result)
			}
			head = Symbol(decl.Name)
		}
		if decl != nil && decl.IsSpecialForm {
			if decl.Type == nil || decl.Type.JITEmit == nil {
				panic("jit: special form has no declaration emitter: " + decl.Name)
			}
			return decl.Type.JITEmit(ctx, list[1:], nil, result)
		}
		if jitIsSelfCall(ctx, head) {
			return jitCompileSelfCall(ctx, list[1:], sliceBase, result)
		}
		if decl == nil {
			var ok bool
			decl, ok = declarations[string(head)]
			if !ok {
				return jitCompileDynamicCall(ctx, list[0], list[1:], nil, sliceBase, result)
			}
		}
		if decl.Type != nil && decl.Type.JITEmit != nil {
			if decl.Type.JITVirtualArgs {
				args := make([]JITValueDesc, len(list)-1)
				for i := 1; i < len(list); i++ {
					param := jitDeclarationParam(decl, i-1)
					if param != nil && param.Kind == "func" {
						args[i-1] = jitCompileCallArgument(ctx, decl, i-1, list[i], sliceBase)
						continue
					}
					argExpr := list[i]
					for argExpr.GetTag() == tagSourceInfo {
						argExpr = argExpr.SourceInfo().value
					}
					if argExpr.IsNthLocalVar() {
						idx := int(argExpr.NthLocalVar())
						if ctx.Env != nil && idx < len(ctx.Env.Numbered) {
							args[i-1] = ctx.Env.Numbered[idx]
						} else if idx < ctx.InputArgCount {
							args[i-1] = JITValueDesc{Loc: LocInputPair, Type: JITTypeUnknown, StackOff: int32(idx)}
						} else {
							args[i-1] = jitCompileExpr(ctx, argExpr, sliceBase, JITValueDesc{Loc: LocAny})
						}
					} else {
						args[i-1] = jitCompileExpr(ctx, argExpr, sliceBase, JITValueDesc{Loc: LocAny})
					}
					args[i-1] = ctx.stabilizeForNested(args[i-1])
				}
				ctx.ReclaimUntrackedRegs()
				if decl.Type.JITInlineCost == 0 {
					ctx.Coverage.InlinedCalls++
				}
				labelsBefore := len(ctx.Labels)
				out := decl.Type.JITEmit(ctx, list[1:], args, result)
				// Virtual-argument emitters use the same generated control-flow
				// machinery as eagerly compiled declarations. A descriptor's Type
				// after rendering several paths belongs only to the path rendered
				// last; consumers must inspect the emitted Scmer tag at runtime.
				if len(ctx.Labels) != labelsBefore && out.Loc == LocRegPair {
					out.Type = JITTypeUnknown
				}
				out.NoHeapPointer = jitReturnHasNoHeapPointer(decl.Type.Return)
				return out
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
				// Nested emitters may need several scratch registers while earlier
				// arguments are still live. Park the oldest resident arguments in
				// their final frame slots instead of treating register exhaustion as
				// a reason to abandon the complete native procedure.
				for bits.OnesCount64(ctx.FreeRegs&ctx.AllRegs&^ctx.ProtectedRegs) < 6 {
					parked := false
					for previous := 0; previous < i-1; previous++ {
						if jitParkCallArgument(ctx, &args[previous]) {
							parked = true
							break
						}
					}
					if !parked {
						break
					}
				}
				args[i-1] = jitCompileCallArgument(ctx, decl, i-1, list[i], sliceBase)
				if args[i-1].Loc == LocRegPair && !args[i-1].NoHeapPointer {
					args[i-1] = jitRootScmer(ctx, args[i-1])
				}
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
			// Keep call arguments resident only while compiling this callee. A
			// function-scoped defer would retain every nested emitter's inputs until
			// the complete outer expression returned, exhausting the allocator and
			// letting stale path-local values leak into later sibling emitters.
			// Argument compilation may have rendered mutually exclusive type paths.
			// Their path-local temporaries are intentionally unowned after merging;
			// reclaim them before entering another generated emitter while retaining
			// the explicitly bound and protected argument descriptors above.
			ctx.ReclaimUntrackedRegs()
			allocatedBeforeEmitter := ctx.AllRegs &^ ctx.FreeRegs
			labelsBefore := len(ctx.Labels)
			if decl.Type.JITInlineCost == 0 {
				ctx.Coverage.InlinedCalls++
			}
			out := decl.Type.JITEmit(ctx, list[1:], args, result)
			ctx.SyncDesc(&out)
			// Generated control-flow emitters may spill their result placement while
			// rendering sibling blocks and then write the final value back into its
			// registers. Rebind that placement at the declaration boundary so stale
			// spill metadata cannot replace the freshly produced result.
			switch out.Loc {
			case LocReg:
				ctx.BindReg(out.Reg, &out)
			case LocRegPair:
				ctx.BindReg(out.Reg, &out)
				ctx.BindReg(out.Reg2, &out)
			case LocRegTriple:
				ctx.BindReg(out.Reg, &out)
				ctx.BindReg(out.Reg2, &out)
				ctx.BindReg(out.Reg3, &out)
			}
			outputRegs := uint64(0)
			switch out.Loc {
			case LocReg:
				outputRegs = 1 << uint(out.Reg)
			case LocRegPair:
				outputRegs = 1<<uint(out.Reg) | 1<<uint(out.Reg2)
			case LocRegTriple:
				outputRegs = 1<<uint(out.Reg) | 1<<uint(out.Reg2) | 1<<uint(out.Reg3)
			}
			// SSA path rendering can leave path-local descriptors resident even
			// though they are unreachable after the generated emitter returns. Keep
			// only registers that predated this call or carry its result.
			internalRegs := (ctx.AllRegs &^ ctx.FreeRegs) &^ allocatedBeforeEmitter &^ outputRegs
			for r := Reg(0); r <= ctx.LastIntReg; r++ {
				if internalRegs&(1<<uint(r)) != 0 {
					ctx.FreeReg(r)
				}
			}
			for _, r := range protectedRegs {
				ctx.UnprotectReg(r)
			}
			// The args slice owns the descriptors that were bound above, so the call
			// boundary must release them. FreeDesc preserves registers that the
			// returned output descriptor has already rebound to itself.
			for i := range args {
				ctx.FreeDesc(&args[i])
			}
			// A generated emitter may render several runtime control-flow paths.
			// Its mutable result descriptor then contains the type of whichever
			// path happened to be rendered last, not a valid merged type. Keep the
			// placement but discard that path-local type information.
			if len(ctx.Labels) != labelsBefore && out.Loc == LocRegPair {
				out.Type = JITTypeUnknown
			}
			out.NoHeapPointer = jitReturnHasNoHeapPointer(decl.Type.Return)
			if out.Loc == LocImm {
				ctx.TrackImm(out.Imm)
			}
			return out
		}
		// A known native declaration needs neither Apply nor dynamic callable
		// dispatch. Its recursive parameter descriptors also tell lambda emission
		// which callback funcvals may live in this invocation's stack frame.
		ctx.Coverage.NativeCalls++
		return jitEmitDeclaredGoVariadicCallFromExprs(ctx, decl.Fn, decl, list[1:], sliceBase, result, decl.RetainsCallArgs)
	default:
		if expr.GetTag() >= 100 {
			// Storage and extensions use opaque custom literal tags. Eval returns
			// those values unchanged, so the emitter must do the same.
			ctx.TrackImm(expr)
			return JITValueDesc{Loc: LocImm, Type: expr.GetTag(), Imm: expr}
		}
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
	localCount := jitRequiredLocalSlots(proc.Body, proc.NumVars)
	if localCount < len(args) {
		localCount = len(args)
	}
	numbered := make([]JITValueDesc, localCount)
	copy(numbered, args)
	for index := len(args); index < len(numbered); index++ {
		off := ctx.AllocStack(16)
		nilValue := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil(), NoHeapPointer: true}
		ctx.EmitStoreScmerToStack(nilValue, off)
		numbered[index] = JITValueDesc{Loc: LocStackPair, Type: tagNil, StackOff: off, NoHeapPointer: true, Rooted: true}
	}
	var vars map[Symbol]JITValueDesc
	bindParam := func(param Scmer, index int) {
		for param.IsSourceInfo() {
			param = param.SourceInfo().value
		}
		if !param.IsSymbol() || param.SymbolEquals("_") || index >= len(numbered) {
			return
		}
		if vars == nil {
			vars = make(map[Symbol]JITValueDesc)
		}
		vars[param.Symbol()] = numbered[index]
	}
	params := proc.Params
	for params.IsSourceInfo() {
		params = params.SourceInfo().value
	}
	switch params.GetTag() {
	case tagSlice:
		for index, param := range params.Slice() {
			bindParam(param, index)
		}
	case tagSymbol:
		bindParam(params, 0)
	}
	innerEnv := &JITEnv{
		Vars:     vars,
		Numbered: numbered,
		Outer:    outer,
	}
	return JITEmitProcInlineWithEnv(ctx, proc, innerEnv, sliceBase, result)
}

// JITEmitProcInlineWithEnv emits a body in an already constructed lexical
// frame. Callers that track named and numbered bindings together must use this
// form so explicit outer-depth references see exactly one environment per
// Scheme scope.
func JITEmitProcInlineWithEnv(ctx *JITContext, proc *Proc, env *JITEnv, sliceBase Reg, result JITValueDesc) JITValueDesc {
	oldEnv := ctx.Env
	ctx.Env = env
	defer func() { ctx.Env = oldEnv }()

	body := proc.Body
	if body.GetTag() == tagSourceInfo {
		body = body.SourceInfo().value
	}
	compileTarget := result
	if result.Loc == LocStackPair {
		compileTarget = JITValueDesc{Loc: LocAny}
	}
	out := jitCompileExpr(ctx, body, sliceBase, compileTarget)
	if result.Loc == LocStackPair {
		ctx.EnsureDesc(&out)
		if out.Loc != LocRegPair && out.Loc != LocImm && out.Loc != LocStackPair && out.Loc != LocInputPair {
			pair := jitAllocTrackedPair(ctx, out.Type)
			out = jitPlaceIntoPair(ctx, &out, pair)
		}
		ctx.EmitCopyScmerToDesc(&result, &out)
		result.Type = out.Type
		result.NoHeapPointer = out.NoHeapPointer
		result.Rooted = true
		ctx.FreeDesc(&out)
		return result
	}
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
	if len(parts) < 3 || !scmerIsSymbol(parts[0], "lambda") {
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

func jitDeclarationHasCallback(decl *Declaration) bool {
	if decl == nil || decl.Type == nil {
		return false
	}
	for _, param := range decl.Type.Params {
		if param != nil && param.Kind == "func" {
			return true
		}
	}
	return false
}

func jitLambdaTemplateCapturesFrame(ctx *JITContext, lambda *JITLambdaTemplate) bool {
	if lambda == nil || jitExpressionConsumesRuntimeEnv(lambda.Proc.Body) {
		return true
	}
	for _, symbol := range jitLambdaFreeSymbols(lambda.Proc.Params, lambda.Proc.Body) {
		if ctx.Env != nil {
			if _, ok := ctx.Env.Lookup(symbol); ok {
				return true
			}
		}
		if _, ok := Globalenv.Vars[symbol]; ok {
			continue
		}
		if runtimeEnv, ok := ctx.RuntimeEnv.Any().(*Env); ok && runtimeEnv != nil {
			if binding := runtimeEnv.FindRead(symbol); binding != nil && binding != &Globalenv {
				if _, exists := binding.Vars[symbol]; exists {
					return true
				}
			}
		}
	}
	outerCaptures, namedOuterCaptures := jitLambdaOuterCaptures(lambda.Proc.Body, true)
	return len(outerCaptures) != 0 || len(namedOuterCaptures) != 0
}

func jitCompileCallArgument(ctx *JITContext, decl *Declaration, index int, expr Scmer, sliceBase Reg) JITValueDesc {
	param := jitDeclarationParam(decl, index)
	if param != nil && param.Kind == "func" {
		transfersInput := false
		for _, callbackParam := range param.Params {
			transfersInput = transfersInput || callbackParam != nil && callbackParam.Transfer
		}
		if lambda, ok := jitLambdaTemplate(expr, ctx.Env); ok {
			// A transferring callback can retain or return storage rooted in its
			// invocation frame. Cross-goroutine consumers therefore need an
			// independently materialized procedure only when the lambda actually
			// captures that frame. Capture-free scan callbacks remain safe direct
			// calls and avoid an allocation on every query execution.
			if !transfersInput || param.SameGoroutine || !jitLambdaTemplateCapturesFrame(ctx, lambda) {
				return JITValueDesc{Loc: LocLambdaTemplate, Type: tagProc, Lambda: lambda}
			}
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

// There is deliberately no later peephole optimizer. This lowering stage and
// each architecture backend form the final intelligent one-pass emitter: known
// types and constants eliminate checks and branches before instructions are
// written, impossible specializations abort to the interpreter fallback, and
// dynamic cases emit only their required checks. Immediate width, moves,
// inlining, and control flow are selected during emission instead of producing
// generic code for a second optimization pass.
