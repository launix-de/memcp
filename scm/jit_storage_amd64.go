//go:build goexperiment.jit && amd64

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
	"fmt"
	"runtime"
	"unsafe"
)

type jitStorageABI uint8

const (
	jitStorageGetValueABI jitStorageABI = iota
	jitStorageGetValueRangeABI
	jitStorageGetValueMultiABI
)

type jitStorageFuncValue struct {
	code  uintptr
	owner *jitStorageCodeBatch
}

// jitStorageCodeBatch owns the scalar and bulk readers compiled for one
// finished storage. Keeping their entry metadata behind one funcval context
// gives all three functions one lifetime and, more importantly, lets the
// compiler place their machine code consecutively in a single reservation.
type jitStorageCodeBatch struct {
	entries []*JITEntryPoint
}

type jitStorageEmitBody func(*JITContext)

type jitStorageCompileRequest struct {
	abi  jitStorageABI
	emit jitStorageEmitBody
}

// JITEnabled reports whether this binary contains the native JIT backend.
func JITEnabled() bool { return true }

// CompileJITStorageGetValue compiles a generated scalar storage emitter into
// the exact func(uint32) Scmer Go ABI used by column consumers.
func CompileJITStorageGetValue(emit JITStorageGetValueEmitter) JITStorageGetValueFunc {
	getValue, _, _ := CompileJITStorageReaders(emit, nil, nil)
	return getValue
}

// CompileJITStorageReaders compiles the three readers of one immutable storage
// into one densely packed code reservation. A storage owns these functions as
// a unit, so separate 16-KiB reservations and cleanup leases only waste arena
// and instruction-TLB capacity without providing useful lifetime granularity.
func CompileJITStorageReaders(getValue JITStorageGetValueEmitter, getValueRange JITStorageGetValueRangeEmitter, getValueMulti JITStorageGetValueMultiEmitter) (JITStorageGetValueFunc, JITStorageGetValueRangeFunc, JITStorageGetValueMultiFunc) {
	requests := make([]jitStorageCompileRequest, 0, 3)
	if getValue != nil {
		requests = append(requests, jitStorageCompileRequest{jitStorageGetValueABI, func(ctx *JITContext) {
			idx := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: RegRAX, NoHeapPointer: true}
			result := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX}
			// RAX initially belongs to the input while RBX already belongs to the
			// eventual result. Explicit ownership keeps either ABI word from becoming
			// anonymous scratch, but still lets the input release RAX at its last use.
			ctx.BindReg(RegRAX, &idx)
			ctx.BindReg(RegRBX, &result)
			out := getValue(ctx, idx, result)
			ctx.EmitMovPairToResult(&out, &result)
		}})
	}
	if getValueRange != nil {
		requests = append(requests, jitStorageCompileRequest{jitStorageGetValueRangeABI, func(ctx *JITContext) {
			recid := jitStorageScalarArg(ctx, RegRAX)
			count := jitStorageScalarArg(ctx, RegRBX)
			target := jitStorageSliceArg(ctx, RegRCX, RegRDI, RegRSI)
			stride := jitStorageScalarArg(ctx, RegR8)
			result := JITValueDesc{Loc: LocStackPair, Type: tagNil, StackOff: ctx.AllocStack(16), Rooted: true}
			_ = getValueRange(ctx, recid, count, target, stride, result)
		}})
	}
	if getValueMulti != nil {
		requests = append(requests, jitStorageCompileRequest{jitStorageGetValueMultiABI, func(ctx *JITContext) {
			recids := jitStorageSliceArg(ctx, RegRAX, RegRBX, RegRCX)
			target := jitStorageSliceArg(ctx, RegRDI, RegRSI, RegR8)
			stride := jitStorageScalarArg(ctx, RegR9)
			result := JITValueDesc{Loc: LocStackPair, Type: tagNil, StackOff: ctx.AllocStack(16), Rooted: true}
			_ = getValueMulti(ctx, recids, target, stride, result)
		}})
	}

	entries, holders := compileJITStorageBatch(requests)
	if len(entries) != len(requests) {
		return nil, nil, nil
	}
	var scalar JITStorageGetValueFunc
	var ranged JITStorageGetValueRangeFunc
	var multi JITStorageGetValueMultiFunc
	for index, request := range requests {
		valuePointer := unsafe.Pointer(holders[index])
		switch request.abi {
		case jitStorageGetValueABI:
			scalar = *(*JITStorageGetValueFunc)(unsafe.Pointer(&valuePointer))
		case jitStorageGetValueRangeABI:
			ranged = *(*JITStorageGetValueRangeFunc)(unsafe.Pointer(&valuePointer))
		case jitStorageGetValueMultiABI:
			multi = *(*JITStorageGetValueMultiFunc)(unsafe.Pointer(&valuePointer))
		}
	}
	return scalar, ranged, multi
}

// CompileJITStorageGetValueRange compiles a generated consecutive-range
// emitter. Stride remains a runtime argument in R8.
func CompileJITStorageGetValueRange(emit JITStorageGetValueRangeEmitter) JITStorageGetValueRangeFunc {
	_, getValueRange, _ := CompileJITStorageReaders(nil, emit, nil)
	return getValueRange
}

// CompileJITStorageGetValueMulti compiles a generated arbitrary-record
// emitter. Stride remains a runtime argument in R9.
func CompileJITStorageGetValueMulti(emit JITStorageGetValueMultiEmitter) JITStorageGetValueMultiFunc {
	_, _, getValueMulti := CompileJITStorageReaders(nil, nil, emit)
	return getValueMulti
}

func jitStorageScalarArg(ctx *JITContext, reg Reg) JITValueDesc {
	desc := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: reg, NoHeapPointer: true}
	ctx.BindReg(reg, &desc)
	return desc
}

func jitStorageSliceArg(ctx *JITContext, data, length, capacity Reg) JITValueDesc {
	desc := JITValueDesc{Loc: LocRegTriple, Type: tagSlice, Reg: data, Reg2: length, Reg3: capacity}
	ctx.BindReg(data, &desc)
	ctx.BindReg(length, &desc)
	ctx.BindReg(capacity, &desc)
	return desc
}

func compileJITStorageBatch(requests []jitStorageCompileRequest) ([]*JITEntryPoint, []*jitStorageFuncValue) {
	entries := make([]*JITEntryPoint, len(requests))
	holders := make([]*jitStorageFuncValue, len(requests))
	if len(requests) == 0 {
		return entries, holders
	}
	for _, registerInputs := range [...]bool{true, false} {
		for _, codeCap := range [...]int{16 * 1024, 64 * 1024, 256 * 1024, 1024 * 1024} {
			ptr, arena, reservation := globalJITPool.Alloc(codeCap * len(requests))
			type compiled struct {
				index     int
				ptr       unsafe.Pointer
				codeLen   int
				frameSize int32
				roots     []unsafe.Pointer
				maps      []jitStackMap
			}
			compiledReaders := make([]compiled, 0, len(requests))
			cursor := 0
			retryLarger := false
			retryStable := false
			for index, request := range requests {
				functionPtr := unsafe.Add(ptr, cursor)
				// emitJITStorageFunction expresses stack-map PCs relative to the
				// reservation offset. Each packed function therefore gets a tiny
				// metadata view with its real arena offset; only the outer reservation
				// participates in publication and lifetime accounting.
				functionReservation := &jitCodeReservation{offset: reservation.offset + cursor}
				buf := &execBuf{ptr: functionPtr, n: codeCap, arena: arena, reservation: functionReservation}
				codeLen, roots, overflow, needsStableInputs := emitJITStorageFunction(buf, request.abi, request.emit, registerInputs)
				if needsStableInputs && registerInputs {
					retryStable = true
					break
				}
				if overflow {
					retryLarger = true
					break
				}
				if codeLen == 0 {
					continue
				}
				compiledReaders = append(compiledReaders, compiled{
					index: index, ptr: functionPtr, codeLen: codeLen,
					frameSize: buf.stackFrameSize, roots: roots, maps: buf.stackMaps,
				})
				cursor = (cursor + codeLen + 15) &^ 15
			}
			if retryLarger || retryStable {
				arena.complete(reservation, nil)
				globalJITPool.Free(arena)
				if retryStable {
					break
				}
				continue
			}
			if len(compiledReaders) == 0 {
				arena.complete(reservation, nil)
				globalJITPool.Free(arena)
				return entries, holders
			}

			allMaps := make([]jitStackMap, 0)
			batch := &jitStorageCodeBatch{}
			for _, function := range compiledReaders {
				entry := &JITEntryPoint{
					DebugName:      jitStorageABIName(requests[function.index].abi),
					StackFrameSize: function.frameSize,
					CodePtr:        function.ptr,
					CodeLen:        function.codeLen,
					Arena:          arena,
					ConstRoots:     function.roots,
				}
				entries[function.index] = entry
				batch.entries = append(batch.entries, entry)
				allMaps = append(allMaps, function.maps...)
			}
			for index, entry := range entries {
				if entry != nil {
					holders[index] = &jitStorageFuncValue{code: uintptr(entry.CodePtr), owner: batch}
				}
			}
			globalJITPool.Trim(arena, reservation, cursor)
			runtime.AddCleanup(batch, releaseJITEntryPoint, jitCodeLease{
				pool:  &globalJITPool,
				arena: arena,
				code:  uintptr(ptr),
			})
			arena.complete(reservation, allMaps)
			for _, entry := range entries {
				if entry == nil {
					continue
				}
				maybeDumpJITCode(entry.CodePtr, (*[1 << 30]byte)(entry.CodePtr)[:entry.CodeLen:entry.CodeLen])
				maybeLogJITCodeName(entry)
			}
			return entries, holders
		}
	}
	return entries, holders
}

func jitStorageABIName(abi jitStorageABI) string {
	switch abi {
	case jitStorageGetValueABI:
		return "storage.GetValue"
	case jitStorageGetValueRangeABI:
		return "storage.GetValueRange"
	case jitStorageGetValueMultiABI:
		return "storage.GetValueMulti"
	default:
		return "storage.unknown"
	}
}

func emitJITStorageFunction(buf *execBuf, abi jitStorageABI, emit jitStorageEmitBody, registerInputs bool) (codeLen int, roots []unsafe.Pointer, overflow bool, needsStableInputs bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			overflow = recovered == jitCodeOverflowPanic
			needsStableInputs = recovered == jitStorageNeedsStableInputs
			if JITLog {
				fmt.Println("storage JIT panic", recovered)
			}
			codeLen = 0
			roots = nil
		}
	}()

	allRegs := uint64((1 << uint(RegRAX)) | (1 << uint(RegRBX)) | (1 << uint(RegRCX)) |
		(1 << uint(RegRDX)) | (1 << uint(RegRSI)) | (1 << uint(RegRDI)) |
		(1 << uint(RegR8)) | (1 << uint(RegR9)) | (1 << uint(RegR10)) |
		(1 << uint(RegR13)) | (1 << uint(RegR15)))
	occupied := jitStorageABIInputRegisters(abi)
	if abi == jitStorageGetValueABI {
		occupied |= 1<<uint(RegRAX) | 1<<uint(RegRBX)
	}
	registerBank := jitX86RegisterBank
	if registerInputs {
		// Four temporaries cover the deepest current getter expression while
		// leaving the remaining registers available to preplanned loop homes.
		registerBank.TemporaryReserve = 4
	}
	ctx := &JITContext{
		Ptr:                      buf.ptr,
		Start:                    buf.ptr,
		End:                      unsafe.Add(buf.ptr, buf.n),
		FreeRegs:                 allRegs &^ occupied,
		AllRegs:                  allRegs,
		RegisterBank:             registerBank,
		StorageInputsInRegisters: registerInputs,
		SliceBase:                RegR12,
		StackReg:                 RegRSP,
		FrameReg:                 RegRBP,
		ScratchReg:               RegR11,
		ResultPtrReg:             RegRAX,
		ResultAuxReg:             RegRBX,
		LastIntReg:               RegR15,
		HasFrame:                 true,
		FrameRoots:               make(map[jitStackRoot]struct{}),
		Arena:                    buf.arena,
	}
	ctx.W = ctx
	guardOffset, stackSmall, moreStackPC := jitRuntimeStackCheck()
	stackRetry := ctx.ReserveLabel()
	stackGrow := ctx.ReserveLabel()
	ctx.MarkLabel(stackRetry)
	ctx.emitMovRegReg(RegR11, RegRSP)
	ctx.emitBytes(0x49, 0x81, 0xEB)
	stackCheckFixup := ctx.Ptr
	ctx.emitU32(0)
	ctx.EmitJcc(CondUnsignedBelow, stackGrow)
	ctx.emitBytes(0x4D, 0x3B, 0x9E)
	ctx.emitU32(uint32(guardOffset))
	ctx.EmitJcc(CondUnsignedBelowOrEqual, stackGrow)

	ctx.emitByte(0x55)
	ctx.emitBytes(0x48, 0x89, 0xE5)
	frameFixup := ctx.EmitSubRSP32Fixup()
	frameInit := ctx.ReserveLabel()
	frameBody := ctx.ReserveLabel()
	ctx.EmitJmp(frameInit)
	ctx.MarkLabel(frameBody)

	closureOff := ctx.AllocStack(8)
	ctx.EmitStoreRegMem(RegRDX, RegRSP, closureOff)
	ctx.setStackPointer(jitStackRootFrameSP, closureOff, true)
	leafBody := ctx.Ptr
	leafFrameLimit := ctx.MaxBPOffset
	emit(ctx)
	// A storage emitter without calls or storage beyond the precautionary
	// closure root is a true leaf. No GC safepoint can observe its frame, and
	// the calling Go funcval remains the owner of the code lease. Compact the
	// already-emitted body to the entry instead of charging every scalar read
	// for an unused stack probe, frame and root initialization.
	leaf := ctx.MaxBPOffset == leafFrameLimit && ctx.MaxSpillOffset == 0 &&
		ctx.MaxDynamicSP == 0 && len(ctx.Safepoints) == 0
	if leaf {
		ctx.emitByte(0xC3)
		// Discard the two unresolved prologue-to-morestack fixups. A proven leaf
		// enters after the prologue and cannot reach either edge; all body-local
		// labels still resolve normally before relocation.
		bodyOffset := int32(uintptr(leafBody) - uintptr(ctx.Start))
		kept := ctx.Fixups[:0]
		for _, fixup := range ctx.Fixups {
			if fixup.CodePos >= bodyOffset {
				kept = append(kept, fixup)
			}
		}
		ctx.Fixups = kept
		ctx.ResolveFixupsFinal()
		bodyLength := int(uintptr(ctx.Ptr) - uintptr(leafBody))
		destination := unsafe.Slice((*byte)(buf.ptr), bodyLength)
		source := unsafe.Slice((*byte)(leafBody), bodyLength)
		copy(destination, source)
		buf.stackFrameSize = 0
		buf.stackMaps = nil
		return bodyLength, ctx.ConstRoots, false, false
	}

	frameSize := (ctx.MaxBPOffset + ctx.MaxSpillOffset + 15) &^ 15
	buf.stackFrameSize = frameSize
	ctx.PatchInt32(frameFixup, frameSize)
	checkedFrame := frameSize + ctx.MaxDynamicSP + 8 - int32(stackSmall)
	if checkedFrame < 0 {
		checkedFrame = 0
	}
	ctx.PatchInt32(stackCheckFixup, checkedFrame)
	ctx.emitByte(0xC9)
	ctx.emitByte(0xC3)

	ctx.MarkLabel(frameInit)
	if frameRoots := jitSortedFrameRoots(ctx.FrameRoots); len(frameRoots) != 0 {
		ctx.emitBytes(0x45, 0x31, 0xDB)
		for _, root := range frameRoots {
			base := RegRSP
			if root.base == jitStackRootFrameBP {
				base = RegRBP
			}
			ctx.EmitStoreRegMem(RegR11, base, root.offset)
		}
	}
	ctx.EmitJmp(frameBody)

	ctx.MarkLabel(stackGrow)
	entryWords, entryPointers := jitStorageSpillEntryArgs(ctx, abi)
	ctx.EmitMovRegImm64(RegR11, uint64(moreStackPC))
	ctx.emitBytes(0x41, 0xFF, 0xD3)
	ctx.Safepoints = append(ctx.Safepoints, jitSafepoint{
		pcOffset:        int32(uintptr(ctx.Ptr) - uintptr(ctx.Start)),
		entry:           true,
		entryFrameWords: entryWords,
		entryPointerMap: entryPointers,
	})
	jitStorageReloadEntryArgs(ctx, abi)
	ctx.EmitJmp(stackRetry)

	arenaOffset := 0
	if buf.reservation != nil {
		arenaOffset = buf.reservation.offset
	}
	buf.stackMaps = ctx.finalizeStackMaps(frameSize, arenaOffset)
	ctx.ResolveFixupsFinal()
	return int(uintptr(ctx.Ptr) - uintptr(ctx.Start)), ctx.ConstRoots, false, false
}

func jitStorageABIInputRegisters(abi jitStorageABI) uint64 {
	var registers []Reg
	switch abi {
	case jitStorageGetValueABI:
		registers = []Reg{RegRAX}
	case jitStorageGetValueRangeABI:
		registers = []Reg{RegRAX, RegRBX, RegRCX, RegRDI, RegRSI, RegR8}
	case jitStorageGetValueMultiABI:
		registers = []Reg{RegRAX, RegRBX, RegRCX, RegRDI, RegRSI, RegR8, RegR9}
	}
	var mask uint64
	for _, reg := range registers {
		mask |= 1 << uint(reg)
	}
	return mask
}

func jitStorageSpillEntryArgs(ctx *JITContext, abi jitStorageABI) (uintptr, []byte) {
	switch abi {
	case jitStorageGetValueABI:
		ctx.emitStoreRegMem32(RegRAX, RegRSP, 8)
		return 2, []byte{0}
	case jitStorageGetValueRangeABI:
		ctx.emitStoreRegMem32(RegRAX, RegRSP, 8)
		ctx.emitStoreRegMem32(RegRBX, RegRSP, 12)
		ctx.EmitStoreRegMem(RegRCX, RegRSP, 16)
		ctx.EmitStoreRegMem(RegRDI, RegRSP, 24)
		ctx.EmitStoreRegMem(RegRSI, RegRSP, 32)
		ctx.EmitStoreRegMem(RegR8, RegRSP, 40)
		return 6, []byte{0b00000100}
	case jitStorageGetValueMultiABI:
		ctx.EmitStoreRegMem(RegRAX, RegRSP, 8)
		ctx.EmitStoreRegMem(RegRBX, RegRSP, 16)
		ctx.EmitStoreRegMem(RegRCX, RegRSP, 24)
		ctx.EmitStoreRegMem(RegRDI, RegRSP, 32)
		ctx.EmitStoreRegMem(RegRSI, RegRSP, 40)
		ctx.EmitStoreRegMem(RegR8, RegRSP, 48)
		ctx.EmitStoreRegMem(RegR9, RegRSP, 56)
		return 8, []byte{0b00010010}
	default:
		panic("jit: unknown storage ABI")
	}
}

func jitStorageReloadEntryArgs(ctx *JITContext, abi jitStorageABI) {
	switch abi {
	case jitStorageGetValueABI:
		ctx.emitMovRegMem32(RegRAX, RegRSP, 8)
	case jitStorageGetValueRangeABI:
		ctx.emitMovRegMem32(RegRAX, RegRSP, 8)
		ctx.emitMovRegMem32(RegRBX, RegRSP, 12)
		ctx.EmitMovRegMem(RegRCX, RegRSP, 16)
		ctx.EmitMovRegMem(RegRDI, RegRSP, 24)
		ctx.EmitMovRegMem(RegRSI, RegRSP, 32)
		ctx.EmitMovRegMem(RegR8, RegRSP, 40)
	case jitStorageGetValueMultiABI:
		ctx.EmitMovRegMem(RegRAX, RegRSP, 8)
		ctx.EmitMovRegMem(RegRBX, RegRSP, 16)
		ctx.EmitMovRegMem(RegRCX, RegRSP, 24)
		ctx.EmitMovRegMem(RegRDI, RegRSP, 32)
		ctx.EmitMovRegMem(RegRSI, RegRSP, 40)
		ctx.EmitMovRegMem(RegR8, RegRSP, 48)
		ctx.EmitMovRegMem(RegR9, RegRSP, 56)
	}
}

func (ctx *JITContext) emitStoreRegMem32(src, base Reg, disp int32) {
	rex := byte(0x40)
	needRex := false
	if src >= 8 {
		rex |= 0x04
		needRex = true
	}
	if base >= 8 {
		rex |= 0x01
		needRex = true
	}
	baseEnc := byte(base & 7)
	srcEnc := byte(src & 7)
	emitPrefix := func() {
		if needRex {
			ctx.emitByte(rex)
		}
	}
	if disp == 0 && baseEnc != 5 {
		emitPrefix()
		ctx.emitBytes(0x89, srcEnc<<3|baseEnc)
		if baseEnc == 4 {
			ctx.emitByte(0x24)
		}
	} else if disp >= -128 && disp <= 127 {
		emitPrefix()
		ctx.emitBytes(0x89, 0x40|srcEnc<<3|baseEnc)
		if baseEnc == 4 {
			ctx.emitByte(0x24)
		}
		ctx.emitByte(byte(int8(disp)))
	} else {
		emitPrefix()
		ctx.emitBytes(0x89, 0x80|srcEnc<<3|baseEnc)
		if baseEnc == 4 {
			ctx.emitByte(0x24)
		}
		ctx.emitU32(uint32(disp))
	}
}

func (ctx *JITContext) emitMovRegMem32(dst, base Reg, disp int32) {
	ctx.emitRegMemOp32(0x8B, dst, base, disp)
}
