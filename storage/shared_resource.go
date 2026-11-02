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

import "time"
import "context"
import "math/rand/v2"

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
	mutation *WriteAccess // existing M admission, only for a mutating cluster scan
	shard    *storageShard
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
	if access.mutation != nil {
		access.mutation.Release()
		return
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

// Retain a complete set of native writers without waiting while holding only
// part of the set. This also works when handover/rebuild changes physical UUIDs.
// Admission retries precede mutation; no row or statement is replayed.
func getShardWrites(shards []*storageShard, tx *TxContext) []WriteAccess {
	accesses := make([]WriteAccess, len(shards))
	complete := false
	release := func() {
		for i := len(accesses) - 1; i >= 0; i-- {
			accesses[i].Release()
			accesses[i] = WriteAccess{}
		}
	}
	defer func() {
		if !complete {
			release()
		}
	}()
	var retries uint
	for {
		retry := false
		for i, shard := range shards {
			access, ready := shard.getWrite(tx, false)
			if !ready {
				release()
				// Independent peers may otherwise recall the same M set in
				// lockstep forever. Stagger only failed, side-effect-free
				// admissions before the next request; normal batches never wait.
				limit := min(10*time.Millisecond, (10*time.Microsecond)<<min(retries, 10))
				retries++
				timer := time.NewTimer(time.Duration(rand.Int64N(int64(limit))) + time.Microsecond)
				select {
				case <-timer.C:
				case <-scanQueryContext(tx, context.Background()).Done():
					timer.Stop()
					panic(scanQueryContext(tx, context.Background()).Err())
				}
				probe := shard.GetWrite(tx)
				probe.Release()
				retry = true
				break
			}
			accesses[i] = access
		}
		if !retry {
			complete = true
			return accesses
		}
	}
}
