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
	"sort"
	"unsafe"
)

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
	if len(items) > 0 && items[0].IsSymbol() {
		switch string(items[0].Symbol()) {
		case "quote", "parser", "lambda":
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
	result := make(map[Symbol]struct{})
	seen := make(map[*Env]struct{})
	visit := func(env *Env) {
		for env != nil {
			if _, ok := seen[env]; ok {
				return
			}
			seen[env] = struct{}{}
			for symbol, value := range env.Vars {
				if value.GetTag() == tagProc && value.Proc() == proc {
					result[symbol] = struct{}{}
				}
			}
			env = env.Outer
		}
	}
	visit(proc.En)
	visit(&Globalenv)
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
	return jitAllocTrackedPair(ctx, JITTypeUnknown)
}

func jitAllocTrackedPair(ctx *JITContext, valueType uint8) JITValueDesc {
	reg := ctx.AllocReg()
	desc := JITValueDesc{Loc: LocRegPair, Type: valueType, Reg: reg, Reg2: ctx.AllocRegExcept(reg)}
	ctx.BindReg(desc.Reg, &desc)
	ctx.BindReg(desc.Reg2, &desc)
	return desc
}

func jitPlaceIntoPair(ctx *JITContext, src *JITValueDesc, target JITValueDesc) JITValueDesc {
	if target.Loc != LocRegPair {
		panic("jit: jitPlaceIntoPair requires LocRegPair target")
	}
	// Placement changes representation, not the proven Scheme type. Keeping the
	// fact lets downstream generated SSA eliminate checks and write barriers.
	target.Type = src.Type
	target.NoHeapPointer = src.NoHeapPointer
	target.Rooted = src.Rooted
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

func jitMakeReservedList(capacity int) Scmer {
	if capacity < 0 {
		capacity = 0
	}
	return NewSlice(make([]Scmer, 0, capacity))
}

func jitCdrScmer(value Scmer) Scmer {
	list := asSlice(value, "cdr")
	if len(list) == 0 {
		return NewSlice(nil)
	}
	return NewSlice(list[1:])
}

func jitStoreScmerAt(address *Scmer, value Scmer) {
	*address = value
}

func jitResolveRuntimeSymbol(envValue, symbol Scmer) Scmer {
	env, ok := envValue.Any().(*Env)
	if !ok || env == nil {
		panic("jit: invalid runtime environment")
	}
	sym := mustSymbol(symbol)
	binding := env.FindRead(sym)
	if binding == nil {
		panic("jit: unresolved symbol " + string(sym))
	}
	return binding.Vars[sym]
}

func jitResolveGlobalSymbol(symbol Scmer) Scmer {
	sym := mustSymbol(symbol)
	if value, ok := Globalenv.Vars[sym]; ok {
		return value
	}
	panic("jit: unresolved global symbol " + string(sym))
}

func jitApplyCallable0(callable, envValue Scmer) Scmer {
	if callable.GetTag() == tagProc {
		if proc := callable.Proc(); proc != nil && proc.Compiled != nil {
			return proc.Compiled.Call()
		}
	}
	return ApplyEx(callable, nil, envValue.Any().(*Env))
}

func jitApplyCallable1(callable, envValue, arg0 Scmer) Scmer {
	if callable.GetTag() == tagProc {
		if proc := callable.Proc(); proc != nil && proc.Compiled != nil {
			return proc.Compiled.Call(arg0)
		}
	}
	return ApplyEx(callable, []Scmer{arg0}, envValue.Any().(*Env))
}

func jitApplyCallable2(callable, envValue, arg0, arg1 Scmer) Scmer {
	if callable.GetTag() == tagProc {
		if proc := callable.Proc(); proc != nil && proc.Compiled != nil {
			return proc.Compiled.Call(arg0, arg1)
		}
	}
	return ApplyEx(callable, []Scmer{arg0, arg1}, envValue.Any().(*Env))
}

func jitApplyCallableSlice(callable, envValue Scmer, args []Scmer) Scmer {
	if callable.GetTag() == tagProc {
		if proc := callable.Proc(); proc != nil && proc.Compiled != nil {
			return proc.Compiled.Call(args...)
		}
	}
	return ApplyEx(callable, args, envValue.Any().(*Env))
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
	}

	ctx.ProtectReg(address.Reg)
	ctx.EnsureDesc(value)
	ctx.UnprotectReg(address.Reg)
	ctx.EmitGoCallVoid(GoFuncAddr(jitStoreScmerAt), []JITValueDesc{*address, *value})
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
		ctx.setStackPointer(jitStackRootFrameBP, off, true)
		stable[i] = JITValueDesc{Loc: LocStackPair, Type: arg.Type, StackOff: off, NoHeapPointer: arg.NoHeapPointer, Rooted: true}
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

func jitAutoImportReturnSafe(name string, td *TypeDescriptor) bool {
	if td == nil {
		return false
	}
	if jitReturnHasNoHeapPointer(td) {
		return true
	}
	// These accessors only borrow storage already rooted by an input value; they
	// do not create a new pointer-bearing object behind the JIT boundary.
	switch name {
	case "nth", "car", "cadr", "cdr":
		return true
	default:
		return false
	}
}

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
	forwardsInput := result.Loc == LocRegPair && result.Reg == ctx.ResultPtrReg && result.Reg2 == ctx.ResultAuxReg && len(virtual.Virtual) == ctx.InputArgCount
	for i := range virtual.Virtual {
		if virtual.Virtual[i].Loc != LocInputPair || virtual.Virtual[i].StackOff != int32(i) {
			forwardsInput = false
			break
		}
	}
	if forwardsInput {
		// The call boundary gives this invocation a fresh variadic array. List's
		// body is therefore declarative: keep the incoming data pointer in the
		// result pointer register and emit only the Scmer slice tag/length word.
		ctx.TransferInputArgs = true
		ctx.EmitMovRegImm64(ctx.ResultAuxReg, makeAux(tagSlice, makeSliceAux(ctx.InputArgCount, ctx.InputArgCount)))
		return JITValueDesc{Loc: LocRegPair, Type: tagSlice, Reg: ctx.ResultPtrReg, Reg2: ctx.ResultAuxReg}
	}
	// Fresh list construction is explicitly JIT-testable, but automatic module
	// activation remains conservative until all allocation/callback arities have
	// complete GC and ABI coverage.
	ctx.AutoImportSafe = false
	pairs := make([]JITValueDesc, len(virtual.Virtual))
	for i := range virtual.Virtual {
		src := virtual.Virtual[i]
		ctx.syncDescSpill(&src)
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
	if len(pairs) > 4 {
		length := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(pairs))), NoHeapPointer: true}
		header := ctx.EmitGoCallScalar(GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{length, length}, 3)
		header.Type = tagSlice
		ctx.BindReg(header.Reg, &header)
		ctx.BindReg(header.Reg2, &header)
		ctx.BindReg(header.Reg3, &header)
		for i := range pairs {
			index := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(i)), NoHeapPointer: true}
			address := ctx.EmitSliceElementAddress(&header, &index, 16)
			ctx.EmitStoreScmerAt(&address, &pairs[i])
			ctx.FreeDesc(&address)
			ctx.FreeDesc(&pairs[i])
		}
		materialized := ctx.EmitNewSliceFromGoSlice(&header)
		if result.Loc == LocRegPair {
			return jitPlaceIntoPair(ctx, &materialized, result)
		}
		return materialized
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
	ctx.EmitLeaRegMem(target.Reg, ctx.StackReg, int32(start*16))
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
			ctx.EmitMovRegReg(dst, valReg)
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
		ctx.EmitSetcc(dst, CondNotEqual)
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
	ctx.EmitGetTagRegs(tagReg, tmp.Reg, tmp.Reg2)
	ctx.EmitCmpRegImm8(tagReg, tagNil)
	ctx.EmitSetcc(tagReg, CondEqual)
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
	stackStart := ctx.BPOffset
	argsOff := int32(0)
	if argc > 0 {
		argsOff = ctx.AllocStack(int32(argc * 16))
		for i := range argExprs {
			jitCompileRootedCallValueAt(ctx, argExprs[i], sliceBase, argsOff+int32(i*16))
		}
	}
	var argsSlice JITValueDesc
	stackBytes := int32(argc * 16)
	if argc > 0 {
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
	ctx.FreeDesc(&argsSlice)
	if stackBytes != 0 {
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
	stackBytes := int32(argc * 16)
	if argc > 0 {
		ctx.EmitReserveStackBytes(stackBytes)
		for i := range values {
			slotOff := int32(i * 16)
			sourceOff := stackBytes + argsOff + slotOff
			ctx.EmitMovRegMem(ctx.ScratchReg, ctx.StackReg, sourceOff)
			ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, slotOff)
			ctx.EmitMovRegMem(ctx.ScratchReg, ctx.StackReg, sourceOff+8)
			ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, slotOff+8)
			ctx.setStackPointer(jitStackRootFrameSP, slotOff-ctx.DynamicSP, true)
		}
		ctx.EmitMovRegReg(argsSlice.Reg, ctx.StackReg)
		ctx.EmitMovRegImm64(argsSlice.Reg2, uint64(argc))
	} else {
		ctx.EmitMovRegImm64(argsSlice.Reg, 0)
		ctx.EmitMovRegImm64(argsSlice.Reg2, 0)
	}
	out := ctx.EmitGoCallVariadic(fn, argsSlice, result)
	ctx.FreeDesc(&argsSlice)
	if stackBytes != 0 {
		ctx.EmitReleaseStackBytes(stackBytes)
		if ctx.SliceBaseTracksRSP && ctx.SliceBase != ctx.StackReg {
			ctx.EmitMovRegReg(ctx.SliceBase, ctx.StackReg)
		}
	}
	ctx.FreeStack(ctx.BPOffset - stackStart)
	return out
}

func jitCompileRootedCallValueAt(ctx *JITContext, expr Scmer, sliceBase Reg, off int32) JITValueDesc {
	value := jitCompileExpr(ctx, expr, sliceBase, JITValueDesc{Loc: LocAny})
	// Input values remain reachable through JITEntryPoint.Call's args slice for
	// the complete native invocation. Calls into Go set NeedsStableArgs, which
	// additionally copies and pins that slice before entering generated code.
	// The safepoint map relocates a saved input pointer if a callback grows
	// the goroutine stack.
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
			for level := 0; level < depth; level++ {
				value = NewSlice([]Scmer{NewSymbol("outer"), value})
			}
			args = append(args, NewSlice([]Scmer{NewSymbol("quote"), key}), value)
		}
		if depth == 0 {
			for index := 0; index < ctx.LocalSlotCount; index++ {
				key := NewNthLocalVar(NthLocalVar(index))
				args = append(args, NewSlice([]Scmer{NewSymbol("quote"), key}), key)
			}
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
	return jitEmitGoVariadicCallFromExprs(ctx, jitMakeSpecialThunk, argExprs, sliceBase, result)
}

func jitCompileDynamicCall(ctx *JITContext, callableExpr Scmer, operands []Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	ctx.Coverage.DynamicCalls++
	stackStart := ctx.BPOffset
	callable := jitCompileRootedCallValue(ctx, callableExpr, sliceBase)
	args := make([]JITValueDesc, 0, len(operands)+2)
	args = append(args, callable)

	envImm := JITValueDesc{Loc: LocImm, Type: tagAny, Imm: ctx.RuntimeEnv}
	ctx.TrackImm(ctx.RuntimeEnv)
	envOff := ctx.AllocStack(16)
	envPair := jitAllocTrackedPair(ctx, tagAny)
	envPair = jitPlaceIntoPair(ctx, &envImm, envPair)
	ctx.EmitStoreScmerToStack(envPair, envOff)
	ctx.FreeDesc(&envPair)
	args = append(args, JITValueDesc{Loc: LocStackPair, Type: tagAny, StackOff: envOff})
	if len(operands) <= 2 {
		for _, operand := range operands {
			args = append(args, jitCompileRootedCallValue(ctx, operand, sliceBase))
		}
	} else {
		// Keep the variadic backing array in the invocation-local JIT frame. Its
		// Scmer pointer words are part of the precise map at the Apply callback,
		// and each pointer-bearing value is mirrored into the hidden Go root
		// slice.
		operandOff := ctx.AllocStack(int32(len(operands) * 16))
		for i, operand := range operands {
			jitCompileRootedCallValueAt(ctx, operand, sliceBase, operandOff+int32(i*16))
		}

		sliceOff := ctx.AllocStack(24)
		ptr := ctx.AllocReg()
		ctx.EmitLeaRegMem(ptr, ctx.StackReg, operandOff)
		ctx.EmitStoreRegMem(ptr, ctx.StackReg, sliceOff)
		ctx.FreeReg(ptr)
		ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(len(operands)))
		ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, sliceOff+8)
		ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, sliceOff+16)
		ctx.setStackPointer(jitStackRootFrameSP, sliceOff, true)
		args = append(args, JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: sliceOff})
	}

	target := jitEnsureResultPair(ctx, result)
	var fn any
	switch len(operands) {
	case 0:
		fn = jitApplyCallable0
	case 1:
		fn = jitApplyCallable1
	case 2:
		fn = jitApplyCallable2
	default:
		fn = jitApplyCallableSlice
	}
	out := ctx.EmitGoCallScalarInto(GoFuncAddr(fn), args, target)
	out.Type = JITTypeUnknown
	ctx.FreeStack(ctx.BPOffset - stackStart)
	out = jitRootScmer(ctx, out)
	return out
}

func jitCompileDynamicHigherOrderCall(ctx *JITContext, callableExpr Scmer, operands []Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	recursiveLambdas := ctx.RecursiveLambdas
	ctx.RecursiveLambdas = false
	defer func() { ctx.RecursiveLambdas = recursiveLambdas }()
	return jitCompileDynamicCall(ctx, callableExpr, operands, sliceBase, result)
}

func jitIsNativeReturnTarget(ctx *JITContext, result JITValueDesc) bool {
	return result.Loc == LocRegPair && result.Reg == ctx.ResultPtrReg && result.Reg2 == ctx.ResultAuxReg
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

	if ctx.HasSelfLoop && jitIsNativeReturnTarget(ctx, result) {
		for i := 0; i < ctx.SelfParamCount; i++ {
			ctx.EmitCopyStackWords(JITValueDesc{Loc: LocStackPair, StackOff: argsOff + int32(i*16)}, int32(i*16), 2)
		}
		for i := ctx.SelfParamCount; i < ctx.LocalSlotCount; i++ {
			ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(i*16))
		}
		ctx.FreeStack(ctx.BPOffset - stackStart)
		ctx.EmitJmp(ctx.SelfLoopLabel)
		return result
	}

	argsPtr := ctx.AllocReg()
	ctx.EmitLeaRegMem(argsPtr, ctx.StackReg, argsOff)
	words := []goCallArgWord{
		{loc: LocReg, reg: argsPtr},
		{loc: LocImm, imm: uint64(ctx.SelfParamCount)},
		{loc: LocImm, imm: uint64(ctx.SelfParamCount)},
	}
	target := jitEnsureResultPair(ctx, result)
	var resultsBuf [16]Reg
	targets := [...]Reg{target.Reg, target.Reg2}
	results := ctx.EmitGoCall(uint64(uintptr(ctx.Start)), words, 2, &resultsBuf, targets[:])
	ctx.FreeReg(argsPtr)
	target.Reg = results[0]
	target.Reg2 = results[1]
	target.Type = JITTypeUnknown
	ctx.BindReg(target.Reg, &target)
	ctx.BindReg(target.Reg2, &target)
	ctx.FreeStack(ctx.BPOffset - stackStart)
	return target
}

func jitCompileRuntimeSymbol(ctx *JITContext, symbol Scmer, result JITValueDesc) JITValueDesc {
	envImm := JITValueDesc{Loc: LocImm, Type: tagAny, Imm: ctx.RuntimeEnv}
	symbolImm := JITValueDesc{Loc: LocImm, Type: tagSymbol, Imm: symbol}
	ctx.TrackImm(ctx.RuntimeEnv)
	ctx.TrackImm(symbol)
	envPair := jitAllocTrackedPair(ctx, tagAny)
	envPair = jitPlaceIntoPair(ctx, &envImm, envPair)
	symbolPair := jitAllocTrackedPair(ctx, tagSymbol)
	symbolPair = jitPlaceIntoPair(ctx, &symbolImm, symbolPair)
	target := jitEnsureResultPair(ctx, result)
	out := ctx.EmitGoCallScalarInto(GoFuncAddr(jitResolveRuntimeSymbol), []JITValueDesc{envPair, symbolPair}, target)
	out.Type = JITTypeUnknown
	out = jitRootScmer(ctx, out)
	ctx.FreeDesc(&envPair)
	ctx.FreeDesc(&symbolPair)
	return out
}

func jitCompileRuntimeGlobalSymbol(ctx *JITContext, symbol Scmer, result JITValueDesc) JITValueDesc {
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
	ctx.EmitJump(CondNotEqual, trueLbl)
	ctx.EmitJmp(falseLbl)
	ctx.FreeDesc(&b)
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
	case tagNil, tagBool, tagInt, tagFloat, tagDate, tagString, tagVector,
		tagFunc, tagFuncEnv, tagProc, tagJIT, tagParser, tagFastDict, tagAny,
		tagClosure, tagPromise:
		// Keep Eval's self-evaluating literal contract. Pointer-bearing constants
		// are retained through the entry point's ConstRoots for the lifetime of
		// the generated code.
		ctx.TrackImm(expr)
		return JITValueDesc{Loc: LocImm, Type: expr.GetTag(), Imm: expr}
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
			case LocInputPair, LocStack, LocStackPair, LocStackTriple:
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
			imm := NewNil()
			ctx.TrackImm(imm)
			return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
		}
		// Resolve operator
		if !list[0].IsSymbol() {
			return jitCompileDynamicCall(ctx, list[0], list[1:], sliceBase, result)
		}
		name := string(list[0].Symbol())
		if jitIsSelfCall(ctx, list[0].Symbol()) {
			return jitCompileSelfCall(ctx, list[1:], sliceBase, result)
		}
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
		case "eval":
			if len(list) < 2 {
				panic("jit: eval expects an expression")
			}
			args := make([]Scmer, 0, 1+2*ctx.LocalSlotCount)
			args = append(args, list[1])
			args = append(args, jitRuntimeCaptureArgExprs(ctx)...)
			return jitEmitGoVariadicCallFromExprs(ctx, jitEvalSpecial, args, sliceBase, result)
		case "time":
			if len(list) < 2 {
				panic("jit: time expects an expression")
			}
			body := jitCompileSpecialThunk(ctx, list[1], sliceBase, JITValueDesc{Loc: LocAny})
			args := []JITValueDesc{body}
			if len(list) > 2 {
				args = append(args, jitCompileSpecialThunk(ctx, list[2], sliceBase, JITValueDesc{Loc: LocAny}))
			} else {
				args = append(args, JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()})
			}
			return jitEmitGoVariadicCallFromDescs(ctx, jitTimeSpecial, args, result)
		case "define", "set":
			if len(list) != 3 {
				panic("jit: malformed " + name)
			}
			binding := list[1]
			for binding.IsSourceInfo() {
				binding = binding.SourceInfo().value
			}
			if !binding.IsSymbol() {
				panic("jit: " + name + " target is not a symbol")
			}
			value := jitCompileExpr(ctx, list[2], sliceBase, result)
			ctx.EnsureDesc(&value)
			stored := value
			if stored.Loc != LocRegPair && stored.Loc != LocImm && stored.Loc != LocStackPair && stored.Loc != LocInputPair {
				pair := jitAllocTrackedPair(ctx, stored.Type)
				stored = jitPlaceIntoPair(ctx, &stored, pair)
			}
			off := ctx.AllocStack(16)
			ctx.EmitStoreScmerToStack(stored, off)
			if ctx.Env == nil {
				ctx.Env = &JITEnv{Vars: make(map[Symbol]JITValueDesc)}
			} else if ctx.Env.Vars == nil {
				ctx.Env.Vars = make(map[Symbol]JITValueDesc)
			}
			ctx.Env.Vars[binding.Symbol()] = JITValueDesc{
				Loc:           LocStackPair,
				Type:          stored.Type,
				StackOff:      off,
				NoHeapPointer: stored.NoHeapPointer,
				Rooted:        true,
			}
			return stored
		case "setN":
			if len(list) != 3 {
				panic("jit: malformed setN")
			}
			targetVar := list[1]
			for targetVar.IsSourceInfo() {
				targetVar = targetVar.SourceInfo().value
			}
			if !targetVar.IsNthLocalVar() {
				panic("jit: setN target is not a numbered local")
			}
			idx := int(targetVar.NthLocalVar())
			if idx < 0 || idx >= ctx.LocalSlotCount || !ctx.SliceBaseTracksRSP {
				panic("jit: setN target outside invocation frame")
			}
			value := jitCompileExpr(ctx, list[2], sliceBase, result)
			ctx.EnsureDesc(&value)
			if value.Loc != LocRegPair && value.Loc != LocImm && value.Loc != LocStackPair && value.Loc != LocInputPair {
				pair := jitAllocTrackedPair(ctx, value.Type)
				value = jitPlaceIntoPair(ctx, &value, pair)
			}
			ctx.EmitStoreScmerToStack(value, int32(idx*16))
			return value
		case "parser":
			if len(list) < 2 {
				panic("jit: parser expects syntax")
			}
			generator := NewNil()
			whitespace := NewNil()
			ignoreResult := false
			if len(list) > 2 {
				generator = list[2]
				ignoreResult = true
			}
			if len(list) > 3 {
				whitespace = list[3]
			}
			args := []Scmer{
				NewSlice([]Scmer{NewSymbol("quote"), list[1]}),
				NewSlice([]Scmer{NewSymbol("quote"), generator}),
				NewSlice([]Scmer{NewSymbol("quote"), whitespace}),
				NewBool(ignoreResult),
			}
			args = append(args, jitRuntimeCaptureArgExprs(ctx)...)
			return jitEmitGoVariadicCallFromExprs(ctx, jitParserSpecial, args, sliceBase, result)
		case "optimizer_proc_return":
			if len(list) != 3 {
				panic("jit: optimizer_proc_return expects procedure and return metadata")
			}
			value := jitCompileExpr(ctx, list[1], sliceBase, JITValueDesc{Loc: LocAny})
			metadata := jitCompileSpecialThunk(ctx, list[2], sliceBase, JITValueDesc{Loc: LocAny})
			return jitEmitGoVariadicCallFromDescs(ctx, jitOptimizerProcReturnSpecial, []JITValueDesc{value, metadata}, result)
		case "begin":
			if len(list) == 1 {
				imm := NewNil()
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
			}
			outerEnv := ctx.Env
			ctx.Env = &JITEnv{Vars: make(map[Symbol]JITValueDesc), Outer: outerEnv}
			defer func() { ctx.Env = outerEnv }()
			for _, form := range list[1 : len(list)-1] {
				value := jitCompileExpr(ctx, form, sliceBase, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&value)
			}
			return jitCompileExpr(ctx, list[len(list)-1], sliceBase, result)
		case "begin_mut":
			if len(list) < 3 {
				panic("jit: begin_mut expects a reserve and body")
			}
			reserveExpr := list[1]
			for reserveExpr.IsSourceInfo() {
				reserveExpr = reserveExpr.SourceInfo().value
			}
			if reserveExpr.GetTag() != tagInt && reserveExpr.GetTag() != tagFloat {
				panic("jit: begin_mut reserve must be optimized to a number")
			}
			reserve := int(ToInt(reserveExpr))
			if reserve < 0 {
				reserve = 0
			}
			if reserve > ctx.LocalSlotCount {
				panic("jit: begin_mut reserve exceeds invocation frame")
			}
			outerEnv := ctx.Env
			ctx.Env = &JITEnv{Vars: make(map[Symbol]JITValueDesc), Outer: outerEnv}
			defer func() { ctx.Env = outerEnv }()
			for _, form := range list[2 : len(list)-1] {
				value := jitCompileExpr(ctx, form, sliceBase, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&value)
			}
			return jitCompileExpr(ctx, list[len(list)-1], sliceBase, result)
		case "!begin":
			if len(list) == 1 {
				imm := NewNil()
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
			}
			for _, form := range list[1 : len(list)-1] {
				value := jitCompileExpr(ctx, form, sliceBase, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&value)
			}
			return jitCompileExpr(ctx, list[len(list)-1], sliceBase, result)
		case "!list":
			return jitCompileStackList(ctx, list, sliceBase, result)
		case "!!list":
			var capacityExpr Scmer
			switch {
			case len(list) == 3 && list[1].IsNthLocalVar():
				start := int(list[1].NthLocalVar())
				capacity := int(ToInt(list[2]))
				if capacity < 0 {
					capacity = 0
				}
				if start < 0 || start+capacity > ctx.LocalSlotCount {
					panic("jit: !!list slots outside invocation frame")
				}
				capacityExpr = NewInt(int64(capacity))
			case len(list) == 2:
				capacityExpr = list[1]
			default:
				panic("jit: malformed !!list")
			}
			capacity := jitCompileExpr(ctx, capacityExpr, sliceBase, JITValueDesc{Loc: LocAny})
			ctx.AutoImportSafe = false
			target := jitEnsureResultPair(ctx, result)
			out := ctx.EmitGoCallScalarInto(GoFuncAddr(jitMakeReservedList), []JITValueDesc{capacity}, target)
			out.Type = tagSlice
			out.KnownSliceLen = 0
			if capacityExpr.IsInt() {
				out.KnownSliceCap = int32(ToInt(capacityExpr))
				if out.KnownSliceCap < 0 {
					out.KnownSliceCap = 0
				}
				out.SliceSizeKnown = true
			}
			return out
		case "cdr":
			if len(list) != 2 {
				panic("jit: cdr expects exactly one argument")
			}
			stackStart := ctx.BPOffset
			value := jitCompileRootedCallValue(ctx, list[1], sliceBase)
			target := jitEnsureResultPair(ctx, result)
			out := ctx.EmitGoCallScalarInto(GoFuncAddr(jitCdrScmer), []JITValueDesc{value}, target)
			out.Type = tagSlice
			ctx.FreeStack(ctx.BPOffset - stackStart)
			return jitRootScmer(ctx, out)
		case "parallel":
			args := make([]JITValueDesc, 0, len(list)-1)
			for _, child := range list[1:] {
				args = append(args, jitCompileSpecialThunk(ctx, child, sliceBase, JITValueDesc{Loc: LocAny}))
			}
			return jitEmitGoVariadicCallFromDescs(ctx, jitParallelSpecial, args, result)
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
				ctx.EmitJump(CondEqual, nextCondLbl)
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
				ctx.EmitJump(CondEqual, falseLbl)
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
				ctx.EmitJump(CondNotEqual, trueLbl)
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
				ctx.EmitJump(CondNotEqual, takeLbl)
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
				ctx.EmitJump(CondEqual, takeLbl) // isNil == 0 => take value
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

			builder := jitBuildLambdaClosure
			if ctx.RecursiveLambdas {
				builder = jitBuildCompiledLambdaClosure
			}
			return jitEmitGoVariadicCallFromExprs(ctx, builder, argExprs, sliceBase, result)
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
			return jitCompileDynamicCall(ctx, list[0], list[1:], sliceBase, result)
		}
		if name == "strlike" {
			panic("jit: strlike emitter is not supported")
		}
		// Pointer-bearing return values need a complete stack map/liveness
		// contract for Go callbacks. Keep them interpreted until that contract is
		// implemented instead of exposing half-supported generated emitters.
		if decl.Type != nil && decl.Type.Return != nil && decl.Type.Return.Kind == "string" {
			return jitCompileDynamicCall(ctx, list[0], list[1:], sliceBase, result)
		}
		if decl.Type != nil && decl.Type.JITEmit != nil {
			ctx.Coverage.InlinedCalls++
			if !jitAutoImportReturnSafe(name, decl.Type.Return) {
				ctx.AutoImportSafe = false
			}
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
				}
				out := decl.Type.JITEmit(ctx, list[1:], args, result)
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
			labelsBefore := ctx.LabelNext
			out := decl.Type.JITEmit(ctx, list[1:], args, result)
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
		return jitCompileDynamicCall(ctx, list[0], list[1:], sliceBase, result)
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
		ctx.EmitStoreScmerToStack(out, result.StackOff)
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

// There is deliberately no later peephole optimizer. This lowering stage and
// each architecture backend form the final intelligent one-pass emitter: known
// types and constants eliminate checks and branches before instructions are
// written, impossible specializations abort to the interpreter fallback, and
// dynamic cases emit only their required checks. Immediate width, moves,
// inlining, and control flow are selected during emission instead of producing
// generic code for a second optimization pass.
