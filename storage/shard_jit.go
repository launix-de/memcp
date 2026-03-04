//go:build amd64

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
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

// fusedMainFn is a JIT-compiled fused map+reduce loop for main storage.
// It iterates over the given record IDs, reads N columns inline, applies
// map and reduce functions inline (no interpreter dispatch), and returns
// the final accumulator.
type fusedMainFn func(acc scm.Scmer, ids []uint32) scm.Scmer

const enableFusedMainJIT = false

// extractColDataPtr extracts the concrete struct data pointer from a
// ColumnStorage interface value. Same technique as extractDataPtr in tests.
func extractColDataPtr(col ColumnStorage) int64 {
	type iface struct {
		_    uintptr
		data uintptr
	}
	return int64((*iface)(unsafe.Pointer(&col)).data)
}

// extractProc returns the *Proc from a Scmer that is tagProc or tagJIT.
// Returns nil if the Scmer holds neither.
func extractProc(v scm.Scmer) *scm.Proc {
	if v.IsProc() {
		return v.Proc()
	}
	if v.IsJIT() {
		p := &v.JIT().Proc
		return p
	}
	return nil
}

// compileFusedMainLoop JIT-compiles a fused phi-loop that fuses column reads
// with map+reduce into a single native code block. Returns (fn, cleanup) where
// cleanup must be called when the MapReducer is closed; returns (nil, nil) if
// compilation fails (caller uses interpreted fallback).
func compileFusedMainLoop(mainCols []ColumnStorage, isUpdate []bool, mapScmer scm.Scmer, reduceScmer scm.Scmer) (fusedMainFn, func()) {
	if !enableFusedMainJIT {
		return nil, nil
	}
	if runtime.GOARCH != "amd64" {
		return nil, nil
	}

	// No update columns: JIT cannot produce the UpdateFunction closure.
	for _, u := range isUpdate {
		if u {
			return nil, nil
		}
	}

	mapProc := extractProc(mapScmer)
	reduceProc := extractProc(reduceScmer)
	if mapProc == nil || reduceProc == nil {
		return nil, nil
	}

	codeBuf := make([]byte, 65536)

	// Free registers: exclude RAX/RBX (arg/result pair), RSP, RBP,
	// R11 (scratch for emit helpers), R12 (inline proc SliceBase), R14 (Go "g").
	freeRegs := uint64(
		(1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) |
			(1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) |
			(1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) |
			(1 << uint(scm.RegR13)) | (1 << uint(scm.RegR15)))
	ctx := &scm.JITContext{
		Ptr:       unsafe.Pointer(&codeBuf[0]),
		Start:     unsafe.Pointer(&codeBuf[0]),
		End:       unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
		FreeRegs:  freeRegs,
		AllRegs:   freeRegs,
		SliceBase: scm.RegR12,
		Env:       &scm.JITEnv{},
	}

	var compileErr interface{}
	func() {
		defer func() { compileErr = recover() }()
		emitFusedLoop(ctx, mainCols, mapProc, reduceProc)
	}()
	if compileErr != nil {
		return nil, nil
	}

	ctx.ResolveFixupsFinal()
	codeLen := int(uintptr(ctx.Ptr) - uintptr(ctx.Start))
	code := codeBuf[:codeLen]

	pageSize := syscall.Getpagesize()
	n := (len(code) + pageSize - 1) &^ (pageSize - 1)
	b, err := syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		return nil, nil
	}
	copy(b, code)
	if err := syscall.Mprotect(b, syscall.PROT_READ|syscall.PROT_EXEC); err != nil {
		syscall.Munmap(b)
		return nil, nil
	}

	type funcHeader struct{ fnptr *byte }
	hdr := &funcHeader{fnptr: &b[0]}
	hdrPtr := unsafe.Pointer(hdr)
	rawFn := *(*fusedMainFn)(unsafe.Pointer(&hdrPtr))

	// Wrap with GC disable to prevent stack-scan failures on unknown JIT PCs.
	fn := func(acc scm.Scmer, ids []uint32) scm.Scmer {
		oldGC := debug.SetGCPercent(-1)
		v := rawFn(acc, ids)
		debug.SetGCPercent(oldGC)
		return v
	}

	cleanup := func() {
		runtime.KeepAlive(hdr)
		runtime.KeepAlive(ctx) // keep SpillBuf alive for JIT fallback calls
		syscall.Munmap(b)
	}
	return fn, cleanup
}

// emitFusedLoop emits the phi-loop machine code.
//
// Function signature emitted: func(acc Scmer, ids []uint32) Scmer
//
// Go ABI register mapping (amd64):
//
//	acc.ptr → RAX, acc.aux → RBX
//	ids.ptr → RCX, ids.len → RDI, ids.cap → RSI (ignored)
//	Returns: RAX=result.ptr, RBX=result.aux
//
// Stack layout (48 bytes, 16-byte aligned):
//
//	[RSP+0]  = idx       (int64, 0..len(ids)-1)
//	[RSP+8]  = acc.ptr   (uintptr)
//	[RSP+16] = acc.aux   (uint64)
//	[RSP+24] = ids.ptr   (*uint32)
//	[RSP+32] = ids.len   (int64)
//	[RSP+40] = padding
//
// Panics on any un-emittable sub-expression; caught in compileFusedMainLoop.
func emitFusedLoop(ctx *scm.JITContext, mainCols []ColumnStorage, mapProc, reduceProc *scm.Proc) {
	const stackSize = int32(48)
	const offIdx = int32(0)
	const offAccPtr = int32(8)
	const offAccAux = int32(16)
	const offIdsPtr = int32(24)
	const offIdsLen = int32(32)

	// Prologue: allocate phi stack and save entry registers.
	spFixup := ctx.EmitSubRSP32Fixup()
	ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, offIdx)
	ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocReg, Reg: scm.RegRAX}, offAccPtr)
	ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocReg, Reg: scm.RegRBX}, offAccAux)
	ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocReg, Reg: scm.RegRCX}, offIdsPtr)
	ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocReg, Reg: scm.RegRDI}, offIdsLen)

	lblTop := ctx.ReserveLabel()
	lblEnd := ctx.ReserveLabel()
	ctx.MarkLabel(lblTop)

	// Loop guard: if idx >= ids.len, exit.
	idxChk := ctx.AllocReg()
	lenChk := ctx.AllocReg()
	ctx.EmitLoadFromStack(idxChk, offIdx)
	ctx.EmitLoadFromStack(lenChk, offIdsLen)
	ctx.EmitCmpInt64(idxChk, lenChk)
	ctx.FreeReg(lenChk)
	ctx.FreeReg(idxChk)
	ctx.EmitJcc(scm.CcGE, lblEnd)

	// Compute address of ids[idx]: R11 = ids.ptr + idx*4 (no SIB helper available).
	idxReg := ctx.AllocReg()
	idsPtrReg := ctx.AllocReg()
	ctx.EmitLoadFromStack(idxReg, offIdx)
	ctx.EmitLoadFromStack(idsPtrReg, offIdsPtr)
	ctx.EmitMovRegReg(scm.RegR11, idxReg)
	ctx.EmitAddInt64(scm.RegR11, scm.RegR11) // *2
	ctx.EmitAddInt64(scm.RegR11, scm.RegR11) // *4
	ctx.EmitAddInt64(scm.RegR11, idsPtrReg)
	ctx.FreeReg(idxReg)
	ctx.FreeReg(idsPtrReg)

	// Load recid = ids[idx] as uint32 (zero-extended to 64 bits).
	recidReg := ctx.AllocReg()
	ctx.EmitMovRegMemL(recidReg, scm.RegR11, 0)

	// Emit column reads: for each column, make a fresh copy of recid and call JITEmit.
	// ProtectReg prevents the allocator from evicting recidReg across N JITEmit calls.
	colDescs := make([]scm.JITValueDesc, len(mainCols))
	ctx.ProtectReg(recidReg)
	for i, col := range mainCols {
		thisptr := scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(extractColDataPtr(col))}
		idxCopy := ctx.AllocReg()
		ctx.EmitMovRegReg(idxCopy, recidReg)
		idxDesc := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxCopy}
		colDescs[i] = col.JITEmit(ctx, thisptr, idxDesc, scm.JITValueDesc{Loc: scm.LocAny})
	}
	ctx.UnprotectReg(recidReg)
	ctx.FreeReg(recidReg)

	// Emit map inline: mapProc(colDescs[0], ..., colDescs[N-1])
	mapResult := emitProcInlineWithStackArgs(ctx, mapProc, colDescs, scm.JITValueDesc{Loc: scm.LocAny})

	// Release column registers (originals; copies were freed by the map body).
	for i := range colDescs {
		ctx.FreeDesc(&colDescs[i])
	}

	// Load acc from phi slots (after freeing colDescs to reclaim registers).
	accPtrReg := ctx.AllocReg()
	accAuxReg := ctx.AllocReg()
	ctx.EmitLoadFromStack(accPtrReg, offAccPtr)
	ctx.EmitLoadFromStack(accAuxReg, offAccAux)
	accDesc := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: accPtrReg, Reg2: accAuxReg}

	// Emit reduce inline: reduceProc(acc, mapResult)
	reduceArgs := []scm.JITValueDesc{accDesc, mapResult}
	newAcc := emitProcInlineWithStackArgs(ctx, reduceProc, reduceArgs, scm.JITValueDesc{Loc: scm.LocAny})

	// Release acc and mapResult originals.
	ctx.FreeDesc(&accDesc)
	ctx.FreeDesc(&mapResult)

	// Store new accumulator back to phi slots.
	ctx.EmitStoreScmerToStack(newAcc, offAccPtr)
	ctx.FreeDesc(&newAcc)

	// idx++
	idxInc := ctx.AllocReg()
	ctx.EmitLoadFromStack(idxInc, offIdx)
	ctx.EmitMovRegImm64(scm.RegR11, 1)
	ctx.EmitAddInt64(idxInc, scm.RegR11)
	ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocReg, Reg: idxInc}, offIdx)
	ctx.FreeReg(idxInc)

	ctx.EmitJmp(lblTop)
	ctx.MarkLabel(lblEnd)

	// Epilogue: return acc from phi slots into RAX/RBX.
	ctx.EmitLoadFromStack(scm.RegRAX, offAccPtr)
	ctx.EmitLoadFromStack(scm.RegRBX, offAccAux)
	ctx.PatchInt32(spFixup, stackSize)
	ctx.EmitAddRSP32(stackSize)
	ctx.EmitByte(0xC3) // RET
}

// emitProcInlineWithStackArgs materializes proc arguments as a contiguous
// Scmer array on the stack and sets R12/ctx.SliceBase so legacy emitters that
// iterate via ctx.SliceBase (e.g. variadic arithmetic) remain correct.
func emitProcInlineWithStackArgs(ctx *scm.JITContext, proc *scm.Proc, args []scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
	if len(args) == 0 {
		oldBase := ctx.SliceBase
		ctx.SliceBase = scm.RegR12
		out := scm.JITEmitProcInline(ctx, proc, args, scm.RegR12, result)
		ctx.SliceBase = oldBase
		return out
	}

	argBytes := int32(len(args) * 16)
	spFixup := ctx.EmitSubRSP32Fixup()
	ctx.PatchInt32(spFixup, argBytes)
	for i := range args {
		switch args[i].Loc {
		case scm.LocRegPair, scm.LocImm:
			ctx.EmitStoreScmerToStack(args[i], int32(i*16))
		default:
			panic("jit: inline proc arg materialization requires LocRegPair/LocImm")
		}
	}
	ctx.EmitMovRegReg(scm.RegR12, scm.RegRSP)
	oldBase := ctx.SliceBase
	ctx.SliceBase = scm.RegR12
	out := scm.JITEmitProcInline(ctx, proc, args, scm.RegR12, result)
	ctx.SliceBase = oldBase
	ctx.EmitAddRSP32(argBytes)
	return out
}
