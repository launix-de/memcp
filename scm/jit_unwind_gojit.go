//go:build goexperiment.jit

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
	"runtime/jit"
	"unsafe"
)

// jitDescribeCallback returns traceback info for a PC inside a JIT arena.
// Must not allocate Go memory (called from the runtime on the system stack).
func jitDescribeCallback(pc uintptr) (name, file string, line int, ok bool) {
	return "MemCP JIT", "", 0, true
}

// registerJITArena registers a JIT arena with the Go runtime so the
// unwinder, GC, and panic/recover can walk through JIT frames.
func registerJITArena(base unsafe.Pointer, size int) interface{} {
	start := uintptr(base)
	return jit.Register(jit.Region{
		Start:    start,
		End:      start + uintptr(size),
		Unwind:   jit.UnwindDeclare,
		Describe: jitDescribeCallback,
		Next:     jitNextCallback,
	})
}

// jitWrapCallTarget returns fn unchanged — runtime/jit handles unwinding
// natively, so no trampoline is needed. Zero overhead.
func jitWrapCallTarget(fn func(...Scmer) Scmer) func(...Scmer) Scmer {
	return fn
}

// callJIT invokes a JIT-compiled function directly. With runtime/jit,
// panics propagate naturally through the unwinder.
func callJIT(native func(...Scmer) Scmer, args ...Scmer) Scmer {
	return native(args...)
}
