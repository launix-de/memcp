/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "sort"
import "sync"
import "time"
import "context"
import "github.com/launix-de/memcp/storage/mosi"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// Preparation retains native write admission, not shard mutexes or readers.
// WAL intents remain invisible until the common durable decision is made.
type acidPreparation struct {
	tx         *transactionState
	shards     []*storageShard
	releases   []func()
	done       chan struct{}
	once       sync.Once
	planMu     sync.Mutex
	visibility map[*storageShard]map[uint32]bool
	closed     bool
	backend    PersistenceEngine
	owner      *TxContext
	watchOnce  sync.Once
}

func (preparation *acidPreparation) release() {
	preparation.once.Do(func() {
		preparation.planMu.Lock()
		preparation.closed = true
		for shard := range preparation.visibility {
			shard.prepared.CompareAndSwap(preparation, nil)
		}
		preparation.planMu.Unlock()
		for _, shard := range preparation.shards {
			shard.prepared.CompareAndSwap(preparation, nil)
		}
		for i := len(preparation.releases) - 1; i >= 0; i-- {
			preparation.releases[i]()
		}
		close(preparation.done)
	})
}

// Rebuild publication must preserve the prepared decision's WAL intents in
// every successor. These are native visibility records, still private until
// the common commit decision; no second mutation kernel is involved.
func (preparation *acidPreparation) projectLocked(source, target *storageShard) map[uint32]bool {
	preparation.planMu.Lock()
	defer preparation.planMu.Unlock()
	if preparation.closed {
		return nil
	}
	if other := target.prepared.Load(); other != nil && other != preparation {
		panic("prepared rebuild conflict")
	}
	projected := preparation.visibility[target]
	if projected == nil {
		projected = make(map[uint32]bool)
	}
	for oldRecid, deleted := range preparation.visibility[source] {
		if newRecid, ok := source.translateNextRecid(oldRecid); ok {
			projected[newRecid] = deleted
		}
	}
	preparation.visibility[target] = projected
	target.prepared.Store(preparation)
	// The producer owns this copy while WAL I/O runs outside publication locks.
	copy := make(map[uint32]bool, len(projected))
	for recid, deleted := range projected {
		copy[recid] = deleted
	}
	return copy
}

func (preparation *acidPreparation) persistProjected(target *storageShard, intents map[uint32]bool) {
	if intents == nil || target.logfile == nil {
		return
	}
	for recid, deleted := range intents {
		target.logVisibilityChangeLocked(recid, deleted, preparation.tx.durableID())
	}
	target.logfile.Flush(target.t.PersistencyMode == Safe)
}

// Native M preparation excludes writers and handover; the cache lease excludes
// full eviction. Readers never take the decision/publication locks during I/O.
func (preparation *acidPreparation) propagate(source *storageShard) {
	source.mu.Lock()
	target := source.loadNext()
	if target == nil || !source.nextReady.Load() {
		source.mu.Unlock()
		return
	}
	target.mu.Lock()
	intents := preparation.projectLocked(source, target)
	target.cacheReaders.Add(1)
	target.mu.Unlock()
	source.mu.Unlock()
	func() {
		defer target.releaseCacheRead()
		preparation.persistProjected(target, intents)
	}()
	preparation.propagate(target)
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
	tx.mu.Lock()
	directory := tx.shardTransactions()
	tx.mu.Unlock()
	preparation := tx.prepared
	if preparation == nil {
		preparation = &acidPreparation{tx: tx.transactionState, done: make(chan struct{}), backend: sharedCommitBackend(), owner: tx}
	}
	known := make(map[*storageShard]bool, len(preparation.shards))
	for _, shard := range preparation.shards {
		known[shard] = true
	}
	var added []*storageShard
	for shard := range directory {
		if !known[shard] {
			added = append(added, shard)
		}
	}
	sort.Slice(added, func(i, j int) bool { return added[i].uuid.String() < added[j].uuid.String() })
	complete := false
	defer func() {
		if !complete {
			preparation.release()
		}
	}()
	for _, shard := range added {
		if err := ctx.Err(); err != nil {
			return err
		}
		if r := activeReplication(shard.t.schema); r != nil && validReplicaDatabase(shard.t.schema.Name) && !shard.t.isEphemeralQueryTable() {
			cell := r.shardCell(shard)
			if cell.ownstate.Phase() != mosi.M {
				cell = r.obtain(*cell.resource.Load(), mosi.M, replicationUser(tx), tx)
			}
			writer, _ := cell.ownstate.TryWrite(tx.transactionState, cell.transactionContinues(tx))
			if writer == nil {
				return fmt.Errorf("ACID prepare conflict: shard write admission is busy")
			}
			preparation.releases = append(preparation.releases, writer.Release)
		}
		shard.ensureMutationLog()
		shard.mu.Lock()
		if err := validateACID([]*storageShard{shard}, directory, preparation); err != nil {
			shard.mu.Unlock()
			return err
		}
		shard.prepared.Store(preparation)
		intents := make(map[uint32]bool)
		st := directory[shard]
		for _, recid := range st.DeleteRecids {
			if st.DeleteMask.Get(uint(recid)) {
				intents[recid] = true
			}
		}
		for _, recid := range st.UndeleteRecids {
			if st.UndeleteMask.Get(uint(recid)) {
				intents[recid] = false
			}
		}
		preparation.planMu.Lock()
		if preparation.visibility == nil {
			preparation.visibility = make(map[*storageShard]map[uint32]bool)
		}
		preparation.visibility[shard] = intents
		preparation.planMu.Unlock()
		preparation.shards = append(preparation.shards, shard)
		shard.mu.Unlock()
		if err := persistenceCall(func() { preparation.propagate(shard) }); err != nil {
			return err
		}
	}
	if err := persistenceCall(func() {
		tx.writeACIDIntents(added, directory)
		for _, shard := range added {
			if shard.logfile != nil {
				shard.logfile.Flush(shard.t.PersistencyMode == Safe)
			}
		}
	}); err != nil {
		return err
	}
	tx.prepared = preparation
	preparation.followDecision()
	complete = true
	return nil
}

// A recorded decision belongs to the transaction, not to one frontend reply.
// Prepared native owners can finish it even if that connection disappears.
// Absence never authorizes abort or grants ownership; only the durable proof
// permits publication through the same native commit path.
func (preparation *acidPreparation) completeRecordedDecision(ctx context.Context) (complete bool, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("prepared transaction recovery: %v", failure)
		}
	}()
	if preparation.backend == nil || !readDistributedDecision(preparation.backend, preparation.tx.durableID()) {
		return false, nil
	}
	tx := preparation.owner
	if tx.coordinator == nil {
		// The originating frontend owns non-atomic session/executor fields.
		// Recovery joins that same executor instead of mutating them concurrently.
		if !tx.queryMu.TryLock() {
			return false, nil
		}
		defer tx.queryMu.Unlock()
	}
	if tx.transactionState != preparation.tx {
		return true, nil
	}
	defer func() {
		if tx.coordinator == nil && tx.State == TxCommitted && !tx.Session.IsNil() {
			tx.Session.Func()(scm.NewString("transaction"), scm.NewNil())
		}
	}()
	release := tx.lockDecision(ctx)
	defer release()
	if tx.State != TxActive {
		return true, nil
	}
	if tx.prepared != preparation {
		return true, nil
	}
	tx.decisionStarted, tx.commitDecided = true, true
	err = tx.commitLocked()
	return err == nil, err
}

func (preparation *acidPreparation) followDecision() {
	if preparation.backend == nil {
		return
	}
	cluster := activeNodeCluster()
	if cluster == nil || cluster.service == nil {
		return
	}
	preparation.watchOnce.Do(func() {
		cluster.service.mu.Lock()
		if cluster.service.stopping || cluster.ctx.Err() != nil {
			cluster.service.mu.Unlock()
			return
		}
		cluster.workers.Add(1)
		cluster.service.mu.Unlock()
		go func() {
			defer cluster.workers.Done()
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-preparation.done:
					return
				case <-cluster.ctx.Done():
					return
				case <-ticker.C:
					complete, err := preparation.completeRecordedDecision(cluster.ctx)
					if complete {
						return
					}
					if err != nil {
						fmt.Println("prepared transaction recovery:", err)
					}
				}
			}
		}()
	})
}

func distributedDecisionLog(txID string) string {
	return ".cluster-decision-v1-" + uuid.NewSHA1(uuid.NameSpaceOID, []byte(txID)).String()
}

// Replay recognizes the existing native commit frame in a transaction-private
// decision stream. Absence is not permission to abort a live prepared owner.
func readDistributedDecision(engine PersistenceEngine, txID string) bool {
	committed, entries, _ := engine.ReplayLog(distributedDecisionLog(txID), true)
	for range entries {
	}
	_, found := committed[txID]
	return found
}

func (tx *TxContext) commitDistributedACID() error {
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
	fallback := context.Background()
	if cluster := activeNodeCluster(); cluster != nil {
		fallback = cluster.ctx
	}
	ctx := scanQueryContext(tx, fallback)
	owners, epoch := tx.remoteParticipantSnapshot()
	keys := make([]string, 0)
	setKeys := func() {
		keys = keys[:0]
		for owner := range owners {
			keys = append(keys, owner)
		}
		sort.Strings(keys)
	}
	setKeys()
	call := func(owner, method string) error {
		st := owners[owner]
		_, err := st.HandleClusterRequest(ctx, scm.ClusterRequest{Method: method, Arguments: []scm.Scmer{scm.NewString(st.decisionToken)}})
		return err
	}
	if !tx.decisionStarted {
		var failure error
		for {
			failure = tx.prepareACID(ctx)
			if failure == nil {
				for _, owner := range keys {
					if err := call(owner, "prepare"); err != nil {
						failure = err
						break
					}
				}
			}
			latest, version := tx.remoteParticipantSnapshot()
			owners = latest
			setKeys()
			if failure != nil || version == epoch {
				break
			}
			epoch = version
		}
		if failure != nil {
			tx.releasePreparation()
			for _, owner := range keys {
				if err := call(owner, "rollback"); err != nil {
					return fmt.Errorf("prepare failed: %v; rollback: %w", failure, err)
				}
			}
			release := tx.claimDecisionRights()
			tx.rollbackACID()
			release()
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

func distributedOutcomeUnknown(err error) error {
	if _, ok := err.(*PersistenceFailure); ok {
		return markCommitOutcomeUnknown(err)
	}
	return &PersistenceFailure{Backend: "cluster", Operation: "transaction.commit", Err: err, OutcomeUnknown: true}
}
