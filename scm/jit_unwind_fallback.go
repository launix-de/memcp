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

// publishJITStackMaps is a no-op on stock Go. JIT execution is disabled, but
// keeping the common emitter path buildable makes (jit) retain its stub API.
func publishJITStackMaps(_ *jitArena, _ []jitStackMap) {}
