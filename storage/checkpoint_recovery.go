/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "context"
import "github.com/launix-de/memcp/scm"

// A checkpoint changes physical row IDs, not transaction ownership. These are
// immutable WAL recipes derived from the existing native undo journals. They
// retain only IDs, never sessions, transaction executors or source payloads.
type checkpointRecoveryCell struct {
	inserts []string
	deletes []string
}
type checkpointRecoveryProjection map[uint32]checkpointRecoveryCell

// Caller holds source.mu. A decision owns finalizationMu before shard locks;
// never wait in the reverse order. Failed TryLock paths release every borrowed
// ancestor/decision before returning a wait to run outside source.mu.
func (source *storageShard) captureCheckpointRecoveryLocked() (checkpointRecoveryProjection, func()) {
	if source.t.PersistencyMode != Safe && source.t.PersistencyMode != Logged {
		return nil, nil
	}
	origins := []*storageShard{source}
	seen := map[*storageShard]bool{source: true}
	for origin := source.rebuildSource.Value(); origin != nil && !seen[origin]; origin = origin.rebuildSource.Value() {
		seen[origin] = true
		if !origin.mu.TryLock() {
			for _, held := range origins[1:] {
				held.mu.Unlock()
			}
			return nil, func() { origin.mu.Lock(); origin.mu.Unlock() }
		}
		origins = append(origins, origin)
	}
	states := make(map[*transactionState]bool)
	for _, origin := range origins {
		origin.overlays.Range(func(key, value any) bool {
			state := key.(*transactionState)
			st := value.(*storageShardTransaction)
			if state.Mode == TxCursorStability && (st.InsertMask.Count() != 0 || st.DeletedMask.Count() != 0) {
				states[state] = true
			}
			return true
		})
	}
	var decisions []*transactionState
	release := func() {
		for _, state := range decisions {
			state.finalizationMu.Unlock()
		}
		for _, origin := range origins[1:] {
			origin.mu.Unlock()
		}
	}
	for state := range states {
		db := source.t.schema
		durable := source.t.PersistencyMode == Safe
		reserved, exists := state.recoveryDecisionDatabases.Load(db)
		if state.coordinator != nil && (!exists || durable && !reserved.(bool)) {
			release()
			return nil, func() {
				_, err := state.executeClusterRequest(context.Background(), scm.ClusterRequest{Method: "checkpoint_decision", Arguments: []scm.Scmer{scm.NewString(db.Name), scm.NewString(db.incarnationID()), scm.NewBool(durable)}})
				if err != nil {
					panic(err)
				}
				// Admission is per actual database and required durability, not
				// the transaction-wide decision bit. A later database or SAFE
				// checkpoint must reserve its own coordinator authority too.
				state.finalizationMu.Lock()
				defer state.finalizationMu.Unlock()
				state.reserveCheckpointDatabase(db, durable)
			}
		}
		if !state.finalizationMu.TryLock() {
			release()
			return nil, func() { state.finalizationMu.Lock(); state.finalizationMu.Unlock() }
		}
		decisions = append(decisions, state)
	}
	defer release()
	projection := make(checkpointRecoveryProjection)
	for _, origin := range origins {
		origin.overlays.Range(func(key, value any) bool {
			state := key.(*transactionState)
			if !states[state] || state.coordinator == nil && state.State != TxActive {
				return true
			}
			st := value.(*storageShardTransaction)
			txID := state.durableID()
			mapped := false
			add := func(recid uint, inserted bool) {
				row := uint32(recid)
				current := origin
				hops := 0
				for current != source {
					if hops >= len(origins) {
						panic("checkpoint recovery source lineage is cyclic")
					}
					next := current.loadNext()
					if next == nil || !current.nextReady.Load() {
						return
					}
					translated, exists := current.translateNextRecid(row)
					if !exists {
						return
					}
					row, current = translated, next
					hops++
				}
				cell := projection[row]
				if inserted {
					cell.inserts = appendCheckpointOwner(cell.inserts, txID)
				} else {
					cell.deletes = appendCheckpointOwner(cell.deletes, txID)
				}
				projection[row] = cell
				mapped = true
			}
			st.InsertMask.Iterate(func(recid uint) { add(recid, true) })
			st.DeletedMask.Iterate(func(recid uint) { add(recid, false) })
			if mapped {
				state.reserveCheckpointDatabase(source.t.schema, source.t.PersistencyMode == Safe)
			}
			return true
		})
	}
	return projection, nil
}

func appendCheckpointOwner(owners []string, owner string) []string {
	for _, previous := range owners {
		if previous == owner {
			return owners
		}
	}
	return append(owners, owner)
}

// Live visibility stays in the existing deletion/undo masks. Recovery starts
// with committed base visibility and applies the existing conditional records.
// An inserted-and-deleted version must see its undelete BEFORE its delete.
func (target *storageShard) writeCheckpointRecoveryLocked(recid uint32, cell checkpointRecoveryCell) {
	if target.logfile == nil {
		return
	}
	target.logVisibilityChangeLocked(recid, len(cell.inserts) != 0, "")
	for _, txID := range cell.inserts {
		target.logVisibilityChangeLocked(recid, false, txID)
	}
	for _, txID := range cell.deletes {
		target.logVisibilityChangeLocked(recid, true, txID)
	}
}

// The existing native coordinator owns the decision even when a shard journal
// was imported elsewhere. Reserve DB authority before allowing that decision;
// a previously completed local-WAL decision can be made independently durable.
func (state *transactionState) checkpointDecision(database, incarnation string, durable bool) error {
	state.finalizationMu.Lock()
	defer state.finalizationMu.Unlock()
	db := GetDatabase(database)
	if db == nil {
		return fmt.Errorf("checkpoint decision database no longer exists")
	}
	db.persistenceLifecycle.RLock()
	defer db.persistenceLifecycle.RUnlock()
	db.ensureLoaded()
	if GetDatabase(database) != db || db.incarnationID() != incarnation {
		return fmt.Errorf("checkpoint decision database incarnation changed")
	}
	// Empty is the existing anonymous legacy incarnation, not a wildcard.
	// CREATE always publishes a fresh nonempty UUID; imports preserve that
	// established identity and hold persistenceLifecycle exclusively. Accept
	// two legacy documents under the same contract as native schema import.
	// Registry revalidation under the lifetime lease rejects local replacement.
	state.reserveCheckpointDatabase(db, durable)
	if state.State != TxCommitted {
		return nil
	}
	required, _ := state.recoveryDecisionDatabases.Load(db)
	return db.commitTransaction(state.durableID(), required.(bool))
}

// Caller owns the actual decision finalizer. The ordinary one-WAL fast path is
// unchanged; only transactions whose rows enter a conditional checkpoint need
// database authority independent of that WAL's eventual retirement.
// The source's walNeedsDecision remains latched through private construction.
// Before the new topology is published, writeRecoveryEntry latches the same
// dependency on its successor. Existing database compaction therefore cannot
// discard this authority in the unpublished-successor window.
func (state *transactionState) reserveCheckpointDatabase(db *database, durable bool) {
	if previous, exists := state.recoveryDecisionDatabases.Load(db); exists {
		durable = durable || previous.(bool)
	}
	state.recoveryDecisionDatabases.Store(db, durable)
	state.recoveryNeedsDatabaseDecision.Store(true)
}

// Publication callers invoke this only after the private builder returned.
// Source/Next forwarding therefore remains intact on a failed barrier; the
// builder's destructive failure cleanup is no longer on the stack.
func (source *storageShard) flushRebuildSuccessor(next *storageShard) {
	if next != nil && next != source && next.logfile != nil {
		next.logfile.Flush(next.t.PersistencyMode == Safe)
	}
}
