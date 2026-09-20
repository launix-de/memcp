//go:build arm64 || riscv64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/

package scm

import "unsafe"

// jitCompileProcToExec starts the non-x86 ports with the leaf subset needed by
// the architecture smoke tests. Unsupported expressions deliberately fall
// back to the interpreter until their universal emitters have native backend
// implementations; they must never enter the legacy x86 lowering path.
func jitCompileProcToExec(proc *Proc, buf *execBuf, _ bool) (int, []unsafe.Pointer, []*JITEntryPoint, bool, []JITHiddenArg, bool, JITCoverage) {
	return jitCompilePortableLeaf(proc, buf)
}

func jitCompilePortableLeaf(proc *Proc, buf *execBuf) (int, []unsafe.Pointer, []*JITEntryPoint, bool, []JITHiddenArg, bool, JITCoverage) {
	defer func() {
		if recover() != nil {
			// The full compiler uses panic as its bounded-buffer and unsupported
			// instruction escape. Keep the leaf port equally fail-closed.
		}
	}()
	if proc == nil || buf == nil || buf.ptr == nil {
		return 0, nil, nil, false, nil, false, JITCoverage{}
	}
	body := proc.Body
	for body.GetTag() == tagSourceInfo {
		body = body.SourceInfo().value
	}
	ctx := &JITContext{Ptr: buf.ptr, Start: buf.ptr, End: unsafe.Add(buf.ptr, buf.n)}
	ctx.W = ctx
	jitArchEmitLeafProlog(ctx)
	mapStart := int(uintptr(ctx.Ptr) - uintptr(ctx.Start))
	coverage := JITCoverage{Expressions: 1}
	if index, ok := jitPortableParameterIndex(proc, body); ok {
		// Load aux first because the incoming slice data register is also the
		// first Scmer result register on both Go/arm64 and Go/riscv64.
		ctx.EmitMovRegMem(RegRBX, RegRAX, int32(index*16+8))
		ctx.EmitMovRegMem(RegRAX, RegRAX, int32(index*16))
	} else if value, ok := jitPortableLeafLiteral(body); ok {
		ctx.TrackImm(value)
		ptr, aux := value.RawWords()
		ctx.EmitMovRegImm64(RegRAX, uint64(ptr))
		ctx.EmitMovRegImm64(RegRBX, aux)
	} else {
		return 0, nil, nil, false, nil, false, JITCoverage{}
	}
	mapEnd := int(uintptr(ctx.Ptr) - uintptr(ctx.Start))
	jitArchEmitLeafEpilog(ctx)
	codeLen := int(uintptr(ctx.Ptr) - uintptr(ctx.Start))
	buf.stackMaps = jitPortableLeafStackMaps(buf, mapStart, mapEnd)
	jitFlushInstructionCache(uintptr(ctx.Start), uintptr(ctx.Ptr))
	return codeLen, ctx.ConstRoots, nil, false, nil, false, coverage
}

func jitPortableLeafStackMaps(buf *execBuf, start, end int) []jitStackMap {
	arenaOffset := 0
	if buf.reservation != nil {
		arenaOffset = buf.reservation.offset
	}
	if end < start {
		return nil
	}
	maps := make([]jitStackMap, 0, (end-start)/4+1)
	for offset := start; offset <= end; offset += 4 {
		maps = append(maps, jitStackMap{
			pcOffset:   uintptr(arenaOffset + offset),
			frameWords: 1,
			pointerMap: []byte{0},
		})
	}
	return maps
}

func jitPortableParameterIndex(proc *Proc, body Scmer) (int, bool) {
	if body.IsNthLocalVar() {
		index := int(body.NthLocalVar())
		return index, index >= 0 && index < jitPortableParameterCount(proc)
	}
	if body.GetTag() != tagSymbol || proc.Params.GetTag() != tagSlice {
		return 0, false
	}
	for index, parameter := range proc.Params.Slice() {
		for parameter.GetTag() == tagSourceInfo {
			parameter = parameter.SourceInfo().value
		}
		if parameter.GetTag() == tagSymbol && parameter.Symbol() == body.Symbol() {
			return index, true
		}
	}
	return 0, false
}

func jitPortableParameterCount(proc *Proc) int {
	if proc.Params.GetTag() == tagSlice {
		return len(proc.Params.Slice())
	}
	if proc.Params.GetTag() == tagSymbol {
		return 1
	}
	return 0
}

func jitPortableLeafLiteral(body Scmer) (Scmer, bool) {
	switch body.GetTag() {
	case tagNil, tagBool, tagInt, tagFloat, tagDate, tagString:
		return body, true
	}
	return Scmer{}, false
}
