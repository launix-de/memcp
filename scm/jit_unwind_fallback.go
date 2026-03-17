//go:build !goexperiment.jit

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

// registerJITArena is a no-op on stock Go builds without runtime/jit.
func registerJITArena(a *jitArena) interface{} {
	return nil
}

// jitWrapCallTarget is a no-op on the fallback path — the panic
// catching happens in callJIT instead (at the Go→JIT boundary).
func jitWrapCallTarget(fn func(...Scmer) Scmer) func(...Scmer) Scmer {
	return fn
}

// callJIT invokes a JIT-compiled function with defer/recover.
// Panics from Go functions called by JIT code are caught here
// (on the pure Go stack) and re-thrown so outer recover() works.
// All state lives on the stack — no globals, no race conditions.
func callJIT(native func(...Scmer) Scmer, args ...Scmer) Scmer {
	var result Scmer
	var pv interface{}
	func() {
		defer func() { pv = recover() }()
		result = native(args...)
	}()
	if pv != nil {
		panic(pv)
	}
	return result
}
