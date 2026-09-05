//go:build !goexperiment.jit || !amd64

/*
Copyright (C) 2026  Carl-Philip Hänsch

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

// JITEnabled reports whether this binary contains the native JIT backend.
func JITEnabled() bool { return false }

func CompileJITStorageGetValue(JITStorageGetValueEmitter) JITStorageGetValueFunc {
	return nil
}

func CompileJITStorageGetValueRange(JITStorageGetValueRangeEmitter) JITStorageGetValueRangeFunc {
	return nil
}

func CompileJITStorageGetValueMulti(JITStorageGetValueMultiEmitter) JITStorageGetValueMultiFunc {
	return nil
}
