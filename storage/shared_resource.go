/*
Copyright (C) 2025-2026  Carl-Philip Hänsch

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
package storage

// Shared resource state used for lazy loaded objects.
// COLD: not loaded yet; SHARED: loaded for read; WRITE: loaded and exclusively writable.
type SharedState uint8

const (
	COLD   SharedState = 0
	SHARED SharedState = 1
	WRITE  SharedState = 2
)

// SharedResource is the access boundary for a lazily loaded local or clustered
// resource. GetRead/GetExclusive must not be bypassed by callers that happen to
// know today's in-process representation: a cluster implementation may satisfy a
// read by loading a local SHARED copy or by routing work to a current shard owner.
// The returned release function delimits that acquired capability and can evolve
// from the current local lifetime tracking into a cluster reference release.
type SharedResource interface {
	GetState() SharedState
	GetRead() func()      // acquire read access; returns release()
	GetExclusive() func() // acquire exclusive access; returns release()
}
