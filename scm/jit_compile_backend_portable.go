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

// The expression compiler is shared; all machine instructions are selected by
// the target backend, including Go call boundaries and precise stack maps.
func jitCompileProcToExec(proc *Proc, buf *execBuf, recursiveLambdas bool) (int, []unsafe.Pointer, []*JITEntryPoint, bool, []JITHiddenArg, bool, JITCoverage) {
	codeLen, roots, dependencies, overflow, hidden, stable, coverage := jitCompileProcNative(proc, buf, recursiveLambdas)
	if codeLen != 0 {
		jitFlushInstructionCache(uintptr(buf.ptr), uintptr(buf.ptr)+uintptr(codeLen))
	}
	return codeLen, roots, dependencies, overflow, hidden, stable, coverage
}
