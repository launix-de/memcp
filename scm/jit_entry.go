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

// ShardJITPool manages mmap'd page allocation per shard. Defined here as
// a placeholder; the full implementation will be added when the page
// allocator is built.
type ShardJITPool struct {
}

// JITEntryPoint holds a JIT-compiled function alongside its original
// Scheme representation for serialization and fallback.
type JITEntryPoint struct {
	Native   func(...Scmer) Scmer // compiled native function pointer
	Pages    []*JITPage           // mmap'd pages holding machine code
	Pool     *ShardJITPool        // pool for returning pages
	Proc     Proc                 // original Proc for serialization
	Arch     string               // runtime.GOARCH at compile time
	BodyHash uint64               // hash of Proc.Body for cache invalidation
}

// tagJIT is the first custom tag slot for JIT-compiled functions.
const tagJIT = 100

// NewJIT wraps a JITEntryPoint as a Scmer value.
func NewJIT(jep *JITEntryPoint) Scmer {
	return NewCustom(tagJIT, unsafe.Pointer(jep))
}

// IsJIT reports whether the value is a JIT-compiled function.
func (s Scmer) IsJIT() bool {
	return s.IsCustom(tagJIT)
}

// JIT returns the JITEntryPoint for a JIT-compiled value.
func (s Scmer) JIT() *JITEntryPoint {
	return (*JITEntryPoint)(s.Custom(tagJIT))
}
