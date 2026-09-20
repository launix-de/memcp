//go:build !amd64 && !arm64 && !riscv64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/

package scm

import "unsafe"

func jitCompileProcToExec(*Proc, *execBuf, bool) (int, []unsafe.Pointer, []*JITEntryPoint, bool, []JITHiddenArg, bool, JITCoverage) {
	return 0, nil, nil, false, nil, false, JITCoverage{}
}
