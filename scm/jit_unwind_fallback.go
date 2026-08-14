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

// jitWrapCallTarget is retained so common code compiles in stock Go builds.
func jitWrapCallTarget(fn func(...Scmer) Scmer) func(...Scmer) Scmer {
	return fn
}

// callJIT must be unreachable: stock Go cannot unwind panics through JIT frames.
func callJIT(_ func(...Scmer) Scmer, _ ...Scmer) Scmer {
	panic("JIT support is disabled in this build")
}
