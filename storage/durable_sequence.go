/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"encoding/binary"
	"math"

	"github.com/launix-de/memcp/scm"
)

// Durable sequences are database-wide write values, unrelated
// to shard generations or planner reuse stamps. A durable reservation precedes
// query work. DML consumes it without I/O or catalog locks; rollback consumes
// numbers too. Recovery skips the unused tail so a crash cannot reuse a value.
// The counter is necessarily shared by writers; no read/scan hot path uses it.
const sequenceReservation = uint64(1) << 32

func (db *database) prepareSequence() {
	db.sequenceEnabled.Store(true)
	if limit, next := db.sequenceLimit.Load(), db.sequenceNext.Load(); limit >= next && limit-next >= sequenceReservation/2 {
		return
	}
	db.sequenceReserveMu.Lock()
	defer db.sequenceReserveMu.Unlock()
	next := db.sequenceNext.Load()
	limit := db.sequenceLimit.Load()
	if limit >= next && limit-next >= sequenceReservation/2 {
		return
	}
	db.schemalock.Lock()
	if db.SequenceHighWater > math.MaxUint64-sequenceReservation {
		db.schemalock.Unlock()
		panic("durable sequence exhausted")
	}
	highWater := db.SequenceHighWater + sequenceReservation
	db.SequenceHighWater = highWater
	// Publish the consumable interval only after its complete schema snapshot
	// has reached stable storage. Failures leave the previous interval in force.
	db.metadataRevision++
	snapshot := db.captureSchemaSnapshotLocked()
	db.schemalock.Unlock()
	db.commitSchemaSnapshot(snapshot, true)
	db.sequenceLimit.Store(highWater)
}

func (db *database) allocateSequence(count uint64) uint64 {
	if count == 0 {
		return db.sequenceNext.Load()
	}
	for {
		next := db.sequenceNext.Load()
		limit := db.sequenceLimit.Load()
		if next > limit || count > limit-next {
			panic("sequence reservation exhausted; prepare the database before DML")
		}
		if db.sequenceNext.CompareAndSwap(next, next+count) {
			return next + 1
		}
	}
}

func sequenceToken(value uint64) scm.Scmer {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	return scm.NewString(string(encoded[:]))
}

func initSequenceBuiltins(en scm.Env) {
	scm.Declare(&en, &scm.Declaration{
		Name: "next_sequence_token",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			db := GetDatabase(a[0].String())
			if db == nil {
				panic("database does not exist")
			}
			// Callers prepare a durable interval at initialization or statement
			// boundaries. Trigger callbacks consume only a CAS; no I/O or locks.
			return sequenceToken(db.allocateSequence(1))
		},
		Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
			Description: "consumes one previously reserved durable big-endian binary sequence token",
			Params:      []*scm.TypeDescriptor{{Kind: "string", Label: "database"}},
			Return:      &scm.TypeDescriptor{Kind: "string"}},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "prepare_sequence",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			db := GetDatabase(a[0].String())
			if db != nil {
				db.ensureLoaded()
				db.prepareSequence()
			}
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
			Description: "reserves durable sequence values before acquiring DML locks",
			Params:      []*scm.TypeDescriptor{{Kind: "string", Label: "database"}},
			Return:      &scm.TypeDescriptor{Kind: "bool"}},
	})
}
