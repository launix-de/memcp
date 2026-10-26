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

// ReadAccess delimits an acquired read capability without allocating a closure
// for every local lookup. It can evolve into a cluster reference release.
// It retains prepared payload against full shard eviction, not a query-wide
// snapshot or a row lock. Operators keep their existing synchronization and
// generation/visibility rules. Release must be called exactly once, including
// panic/cancellation unwinding; a zero access requires no cleanup.
type ReadAccess struct {
	scanRAM *queryRAMLease // optional cluster operator budget; not a reader/MOSI lock
	shard   *storageShard
}

type WriteAccess struct {
	shard   *storageShard
	release func()
}

func (access WriteAccess) Release() {
	if access.release != nil {
		access.release()
	}
}

func (access ReadAccess) Release() {
	if access.scanRAM != nil {
		defer access.scanRAM.Release()
	}
	if access.shard != nil {
		access.shard.releaseCacheRead()
	}
}

// SharedResource is the access boundary for a lazily loaded local or clustered
// resource. GetRead/GetWrite must not be bypassed by callers that happen to
// know today's in-process representation: a cluster implementation may satisfy a
// read by loading a local SHARED copy or by routing work to a current shard owner.
type SharedResource interface {
	GetState() SharedState
	GetRead(currentTx *TxContext) ReadAccess // cold-load waits observe this query
	GetWrite(currentTx *TxContext) WriteAccess
}
