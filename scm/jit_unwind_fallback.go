//go:build !gojit

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

import "unsafe"

// jitPanicSlot holds a panic value caught by a JIT→Go trampoline,
// pending re-throw at the Go→JIT call site. Safe as a package-level
// variable because JIT execution is sequential within each goroutine.
var jitPanicSlot interface{}

// registerJITArena is a no-op on stock Go builds without runtime/jit.
func registerJITArena(base unsafe.Pointer, size int) interface{} {
	return nil
}

// wrapNoPanic calls fn(args...) and recovers any panic. If fn panics,
// result is unspecified and panicvalue holds the recovered value.
func wrapNoPanic(fn func(...Scmer) Scmer, args ...Scmer) (result Scmer, panicvalue interface{}) {
	defer func() {
		panicvalue = recover()
	}()
	result = fn(args...)
	return
}

// jitWrapCallTarget wraps fn with a panic-catching trampoline so that
// panics in Go code called from JIT are caught before they reach the
// JIT frame. The caught panic is stored in jitPanicSlot for re-throw.
func jitWrapCallTarget(fn func(...Scmer) Scmer) func(...Scmer) Scmer {
	return func(args ...Scmer) Scmer {
		result, pv := wrapNoPanic(fn, args...)
		if pv != nil {
			jitPanicSlot = pv
		}
		return result
	}
}

// callJIT invokes a JIT-compiled function and re-throws any panic that
// was caught by the inner trampoline. This runs on the pure Go stack,
// so recover() in outer frames works normally.
func callJIT(native func(...Scmer) Scmer, args ...Scmer) Scmer {
	result := native(args...)
	if pv := jitPanicSlot; pv != nil {
		jitPanicSlot = nil
		panic(pv)
	}
	return result
}
