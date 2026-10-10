/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "github.com/launix-de/memcp/scm"

// prepareIdentityRecovery runs at an INSERT boundary, before any shard lock.
// A selected partition cannot reconstruct IDs reserved in another cold WAL.
// Fresh tables also pass through this once, with already-initialized shards.
func (t *table) prepareIdentityRecovery() {
	for _, col := range t.Columns {
		if col.AutoIncrement {
			t.recoverIdentityCounter()
			return
		}
	}
}

func (t *table) recoverIdentityCounter() {
	if t.identityRecoveryState.Load() == 1 {
		return
	}
	t.identityRecoveryOnce.Do(func() {
		complete := false
		defer func() {
			if !complete {
				t.identityRecoveryState.Store(2)
			}
		}()
		topology := t.pinActiveTopology()
		defer topology.releaseOperation()
		for _, shard := range topology.shards {
			func() {
				release := shard.GetRead(nil)
				defer release()
			}()
		}
		complete = true
		t.identityRecoveryState.Store(1)
	})
	if t.identityRecoveryState.Load() != 1 {
		panic("identity recovery previously failed; restart before allocating another identity")
	}
}

// allocatorLimit is explicit neutral metadata, bound by the declaring frontend.
func allocatorLimit(c *column) (uint64, bool) { return c.AllocatorMax, c.AllocatorMax != 0 }

func allocatorExplicitValue(c *column, value scm.Scmer) uint64 {
	if c.AllocatorValue != nil {
		value = scm.Apply(*c.AllocatorValue, value)
	}
	if id := scm.ToInt(value); id > 0 {
		return uint64(id)
	}
	return 0
}
