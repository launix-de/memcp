/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

// A physical WAL can keep needing a database decision after its live deletion
// bitmap is empty. This is deliberately conservative: a shard-local commit may
// make an individual entry self-contained, but only a new checkpoint generation
// clears this fact. Closing/reopening or moving an append handle never clears it.
func (s *storageShard) noteRecoveryEntry(entry interface{}) {
	var transaction string
	switch value := entry.(type) {
	case LogEntryInsert:
		transaction = value.txID
	case LogEntryInsertHidden:
		transaction = value.txID
	case LogEntryDelete:
		transaction = value.txID
	case LogEntryUndelete:
		transaction = value.txID
	}
	if transaction != "" {
		s.walNeedsDecision.Store(true)
	}
}

// Native mutation/private-generation owners already serialize WAL publication.
// Latch before Write: even a failed barrier can have accepted the frame.
func (s *storageShard) writeRecoveryEntry(entry interface{}) {
	s.noteRecoveryEntry(entry)
	s.logfile.Write(entry)
}

// Prepared attachment can precede its projected WAL writes. Reserve dependence
// before that attachment becomes visible, including retries of an old decision.
func (s *storageShard) reserveRecoveryDecision() {
	if s.logfile != nil {
		s.walNeedsDecision.Store(true)
	}
}

// The caller owns persistenceLifecycle through durable schema publication and
// compaction. Capture actual native roots, never the earlier rebuild work list.
// Cold generations are unknown; asking this question does not load or replay.
func (db *database) independentRecoveryTopology() (map[*table]*tableShardTopology, bool) {
	roots := make(map[*table]*tableShardTopology)
	for _, table := range db.tables.GetAll() {
		topology := table.topology.Load()
		if topology == nil {
			return nil, false
		}
		roots[table] = topology
	}
	return roots, db.recoveryTopologyStillIndependent(roots)
}

// Re-attest after joining transactionMu without acquiring shard/schema locks:
// commits take shard locks before transactionMu. Dependency is monotonic within
// a generation and is reserved before old prepared intents can reach it. New
// decisions after the caller's cutoff remain in the existing compaction suffix.
func (db *database) recoveryTopologyStillIndependent(roots map[*table]*tableShardTopology) bool {
	tables := db.tables.GetAll()
	if len(tables) != len(roots) {
		return false
	}
	for _, table := range tables {
		topology := roots[table]
		if topology == nil || table.topology.Load() != topology {
			return false
		}
		for _, shard := range topology.shards {
			if shard != nil && (shard.state() == COLD || shard.walNeedsDecision.Load()) {
				return false
			}
		}
	}
	return true
}
