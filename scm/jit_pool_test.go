/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

package scm

import "testing"

func TestJITPoolUnmapsSealedArenaAfterLastLease(t *testing.T) {
	var pool jitPool
	_, first, _ := pool.Alloc(16)
	pool.Free(first)
	if first.unmapped {
		t.Fatal("current arena was unmapped while it could still be reused")
	}

	_, second, _ := pool.Alloc(jitArenaSize)
	if !first.unmapped || first.mapping != nil {
		t.Fatal("sealed arena was not unmapped after its last lease")
	}

	pool.Free(second)
	pool.shutdown()
	if !second.unmapped || second.mapping != nil {
		t.Fatal("shutdown did not unmap the current arena")
	}
}

func TestJITPoolKeepsSealedArenaUntilLastLease(t *testing.T) {
	var pool jitPool
	_, first, _ := pool.Alloc(16)
	_, _, _ = pool.Alloc(16)
	pool.Free(first)

	_, second, _ := pool.Alloc(jitArenaSize)
	if first.unmapped {
		t.Fatal("sealed arena was unmapped with a live lease")
	}
	pool.Free(first)
	if !first.unmapped {
		t.Fatal("sealed arena remained mapped after its last lease")
	}

	pool.Free(second)
	pool.shutdown()
}
