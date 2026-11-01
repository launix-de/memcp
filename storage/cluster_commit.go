/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "sort"
import "sync"
import "context"
import "strings"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// Preparation retains native write admission, not shard mutexes or readers.
// WAL intents remain invisible until the common durable decision is made.
type acidPreparation struct {
	tx       *transactionState
	shards   []*storageShard
	releases []func()
	done     chan struct{}
	once     sync.Once
}

func (preparation *acidPreparation) release() {
	preparation.once.Do(func() {
		for _, shard := range preparation.shards {
			shard.prepared.CompareAndSwap(preparation, nil)
		}
		for i := len(preparation.releases) - 1; i >= 0; i-- {
			preparation.releases[i]()
		}
		close(preparation.done)
	})
}

func (tx *TxContext) releasePreparation() {
	if tx.prepared != nil {
		tx.prepared.release()
	}
}

func validateACID(shards []*storageShard, directory map[*storageShard]*storageShardTransaction, preparation *acidPreparation) error {
	for _, shard := range shards {
		if other := shard.prepared.Load(); other != nil && other != preparation {
			return fmt.Errorf("ACID commit conflict: shard has a prepared transaction")
		}
		st := directory[shard]
		for _, recid := range st.DeleteRecids {
			if st.DeleteMask.Get(uint(recid)) && shard.deletions.Get(uint(recid)) {
				return fmt.Errorf("ACID commit conflict: row %d already deleted", recid)
			}
		}
	}
	return nil
}

// Both local and distributed commits use these same native visibility intents.
// No new logfile format or second mutation implementation is introduced.
func (tx *TxContext) writeACIDIntents(shards []*storageShard, directory map[*storageShard]*storageShardTransaction) {
	for _, shard := range shards {
		tx.RegisterTouchedShard(shard)
		if shard.logfile == nil {
			continue
		}
		st := directory[shard]
		for _, recid := range st.DeleteRecids {
			if st.DeleteMask.Get(uint(recid)) {
				shard.logfile.Write(LogEntryDelete{idx: recid, txID: tx.durableID()})
			}
		}
		for _, recid := range st.UndeleteRecids {
			if st.UndeleteMask.Get(uint(recid)) {
				shard.logfile.Write(LogEntryUndelete{idx: recid, txID: tx.durableID()})
			}
		}
	}
}

func (tx *TxContext) prepareACID(ctx context.Context) error {
	if tx.prepared != nil {
		return nil
	}
	tx.mu.Lock()
	directory := tx.shardTransactions()
	tx.mu.Unlock()
	preparation := &acidPreparation{tx: tx.transactionState, done: make(chan struct{})}
	for shard := range directory {
		preparation.shards = append(preparation.shards, shard)
	}
	sort.Slice(preparation.shards, func(i, j int) bool { return preparation.shards[i].uuid.String() < preparation.shards[j].uuid.String() })
	complete := false
	defer func() {
		if !complete {
			preparation.release()
		}
	}()
	for _, shard := range preparation.shards {
		if err := ctx.Err(); err != nil {
			return err
		}
		if r := activeReplication(shard.t.schema); r != nil {
			cell := r.shardCell(shard)
			// An OCC preparation declines a busy writer rather than waiting
			// while other peers retain their own prepared native write scopes.
			writer, _ := cell.ownstate.TryWrite(tx.transactionState, cell.transactionContinues(tx))
			if writer == nil {
				return fmt.Errorf("ACID prepare conflict: shard write admission is busy")
			}
			preparation.releases = append(preparation.releases, writer.Release)
		}
		shard.mu.Lock()
		if err := validateACID([]*storageShard{shard}, directory, preparation); err != nil {
			shard.mu.Unlock()
			return err
		}
		shard.prepared.Store(preparation)
		shard.mu.Unlock()
	}
	if err := persistenceCall(func() {
		tx.writeACIDIntents(preparation.shards, directory)
		for _, shard := range preparation.shards {
			if shard.logfile != nil {
				shard.logfile.Flush(shard.t.PersistencyMode == Safe)
			}
		}
	}); err != nil {
		return err
	}
	tx.prepared = preparation
	complete = true
	return nil
}

func distributedDecisionLog(txID string) string {
	return ".cluster-decision-v1-" + uuid.NewSHA1(uuid.NameSpaceOID, []byte(txID)).String()
}

// Replay recognizes the existing native commit frame in a transaction-private
// decision stream. Absence is not permission to abort a live prepared owner.
func readDistributedDecision(engine PersistenceEngine, txID string) bool {
	committed, entries, log := engine.ReplayLog(distributedDecisionLog(txID))
	for range entries {
	}
	log.Close()
	_, found := committed[txID]
	return found
}

func (tx *TxContext) commitDistributedACID() error {
	tx.decisionMu.Lock()
	defer tx.decisionMu.Unlock()
	if tx.State == TxCommitted {
		return nil
	}
	if tx.State != TxActive {
		return fmt.Errorf("transaction is not active")
	}
	engine := sharedCommitBackend()
	if engine == nil {
		return fmt.Errorf("distributed commit requires the shared cluster backend")
	}
	ctx := scanQueryContext(tx, context.Background())
	owners := tx.remoteParticipantOwners()
	keys := make([]string, 0, len(owners))
	for owner := range owners {
		keys = append(keys, owner)
	}
	sort.Strings(keys)
	call := func(owner, method string) error {
		st := owners[owner]
		_, err := st.HandleClusterRequest(ctx, scm.ClusterRequest{Method: method, Arguments: []scm.Scmer{scm.NewString(st.decisionToken)}})
		return err
	}
	if !tx.decisionStarted {
		failure := tx.prepareACID(ctx)
		if failure == nil {
			for _, owner := range keys {
				if err := call(owner, "prepare"); err != nil {
					failure = err
					break
				}
			}
		}
		if failure != nil {
			// Roll back all enlisted owners, including an uncertain prepare
			// reply. No commit authority has been attempted at this point.
			for _, owner := range keys {
				if err := call(owner, "rollback"); err != nil {
					return fmt.Errorf("prepare failed: %v; rollback: %w", failure, err)
				}
			}
			tx.rollbackACID()
			tx.releasePreparation()
			return failure
		}
		tx.decisionStarted = true
	}
	if !tx.commitDecided {
		if err := persistenceCall(func() {
			log := engine.SwapLog(distributedDecisionLog(tx.durableID()), []any{LogEntryCommit{txID: tx.durableID()}}, true)
			log.Close()
		}); err != nil {
			return distributedOutcomeUnknown(err)
		}
		tx.commitDecided = true
	}
	for _, owner := range keys {
		if err := call(owner, "commit"); err != nil {
			return distributedOutcomeUnknown(err)
		}
	}
	if err := tx.commitACID(); err != nil {
		return distributedOutcomeUnknown(err)
	}
	tx.mu.Lock()
	tx.participants = nil
	tx.mu.Unlock()
	return nil
}

func isDistributedTxID(id string) bool { return strings.HasPrefix(id, "cluster/") }

func distributedOutcomeUnknown(err error) error {
	if _, ok := err.(*PersistenceFailure); ok {
		return markCommitOutcomeUnknown(err)
	}
	return &PersistenceFailure{Backend: "cluster", Operation: "transaction.commit", Err: err, OutcomeUnknown: true}
}
