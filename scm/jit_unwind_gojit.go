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
	"runtime"
	"runtime/jit"
	"unsafe"
)

// registerJITArena registers a JIT arena with the Go runtime so the
// unwinder, GC, and panic/recover can walk through JIT frames.
// The Describe callback resolves PCs to Scheme source locations
// via the arena's source map (populated during JIT compilation).
func registerJITArena(a *jitArena) interface{} {
	start := uintptr(a.base)
	return jit.Register(jit.Region{
		Start:  start,
		End:    start + uintptr(a.size),
		Unwind: jit.UnwindDeclare,
		Describe: func(pc uintptr) (name, file string, line int, ok bool) {
			offset := int32(pc - uintptr(a.base))
			entries := a.loadSourceEntries()
			// Binary search: find last entry with offset <= target
			lo, hi := 0, len(entries)
			for lo < hi {
				mid := int(uint(lo+hi) >> 1)
				if entries[mid].offset <= offset {
					lo = mid + 1
				} else {
					hi = mid
				}
			}
			if lo > 0 {
				e := &entries[lo-1]
				return "MemCP JIT", e.file, int(e.line), true
			}
			return "MemCP JIT", "", 0, true
		},
	})
}

func publishJITStackMaps(a *jitArena, maps []jitStackMap) {
	if len(maps) == 0 {
		return
	}
	handle, ok := a.handle.(jit.Handle)
	if !ok {
		panic("jit: invalid runtime registration handle")
	}
	runtimeMaps := make([]jit.StackMap, len(maps))
	for i := range maps {
		if maps[i].frameWords == 0 {
			panic("jit: invalid empty unwind frame")
		}
		frameBaseOffset := (maps[i].frameWords - 1) * unsafe.Sizeof(uintptr(0))
		runtimeMaps[i] = jit.StackMap{
			PCOffset:    maps[i].pcOffset,
			FrameWords:  maps[i].frameWords,
			PointerMask: maps[i].pointerMap,
			HasUnwind:   true,
			// Each safepoint already carries its final static plus dynamic frame
			// size. Address the saved frame pointer directly instead of depending
			// on transient delta words in the outgoing Go-call spill area.
			UnwindBaseOffset: frameBaseOffset,
			CallerPCOffset:   8,
			CallerSPOffset:   16,
			CallerBPOffset:   0,
		}
	}
	handle.AddStackMaps(runtimeMaps...)
}

// jitWrapCallTarget returns fn unchanged — runtime/jit handles unwinding
// natively, so no trampoline is needed. Zero overhead.
func jitWrapCallTarget(fn func(...Scmer) Scmer) func(...Scmer) Scmer {
	return fn
}

// callJIT invokes a JIT-compiled function directly. With runtime/jit,
// panics propagate naturally through the unwinder.
//
//go:noinline
func callJIT(native func(...Scmer) Scmer, args ...Scmer) Scmer {
	result := native(args...)
	// Keep an ordinary Go frame between independently compiled JIT entries.
	// The runtime currently resolves one registered foreign frame at a time;
	// retaining this bridge also gives nested calls an unambiguous Go caller PC.
	runtime.KeepAlive(native)
	return result
}
