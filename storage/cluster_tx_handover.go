/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "context"
import "fmt"
import "sort"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

// A transaction/shard overlay inherits its exact native generation's MOSI
// authority. There is no independent flag which could diverge from the shard.
func (st *storageShardTransaction) MOSIState() mosi.Phase {
	if st.clusterShard != nil {
		if cell := st.clusterShard.coherence.Load(); cell != nil {
			return cell.resolved().ownstate.Phase()
		}
	}
	if st.readOnly {
		return mosi.S
	}
	return mosi.M
}

// Migration closes decision admission without holding a Go mutex over network
// I/O. Readers continue using their already acquired native generations.
func (tx *TxContext) lockDecision(ctx context.Context) func() {
	for {
		tx.decisionMu.Lock()
		if tx.migrations == 0 {
			return tx.decisionMu.Unlock
		}
		done := tx.migrationDone
		tx.decisionMu.Unlock()
		select {
		case <-done:
		case <-ctx.Done():
			panic(ctx.Err())
		}
	}
}

func (tx *transactionState) beginMigration() (func(), error) {
	if !tx.decisionMu.TryLock() {
		return nil, errClusterReadBusy
	}
	if tx.migrations == 0 {
		tx.migrationDone = make(chan struct{})
	}
	tx.migrations++
	tx.decisionMu.Unlock()
	return func() {
		tx.decisionMu.Lock()
		tx.migrations--
		if tx.migrations == 0 {
			close(tx.migrationDone)
		}
		tx.decisionMu.Unlock()
	}, nil
}

func (tx *TxContext) bindShardEndpointLocked(st *storageShardTransaction) *storageShardTransaction {
	if tx.endpoints == nil {
		tx.endpoints = make(map[uuid.UUID]*storageShardTransaction)
	}
	endpoint := tx.endpoints[st.shardUUID]
	if endpoint == nil {
		endpoint = st
		tx.endpoints[st.shardUUID] = st
	}
	if st.endpoint == nil {
		st.endpoint = endpoint
	}
	if endpoint != st && st.clusterShard != nil && !st.readOnly {
		endpoint.forwarded.Store(st)
	}
	return endpoint
}
func (tx *TxContext) ownedShardLocked(id uuid.UUID) *storageShardTransaction {
	var result *storageShardTransaction
	if directory := tx.shards.Load(); directory != nil {
		directory.Range(func(key, value any) bool {
			if key.(*storageShard).uuid == id {
				result = value.(*storageShardTransaction)
				return false
			}
			return true
		})
	}
	return result
}
func (st *storageShardTransaction) ClusterForwardedHandle() *scm.ClusterHandle {
	if next := st.forwarded.Load(); next != nil {
		return next.ClusterForwardedHandle()
	}
	return st.remoteObject
}

func (st *storageShardTransaction) ClusterEndpoint() scm.ClusterObject {
	if st.endpoint != nil {
		return st.endpoint
	}
	return st
}
func (st *storageShardTransaction) ownerURI() string {
	if next := st.forwarded.Load(); next != nil {
		return next.ownerURI()
	}
	if st.participantOwner != "" {
		return st.participantOwner
	}
	if st.remoteObject != nil {
		return st.remoteObject.Reference.Owner
	}
	if service := nodeScans.Load(); service != nil {
		return service.authNodeID()
	}
	return ""
}
func (st *storageShardTransaction) waitAuthority(ctx context.Context) error {
	if st.clusterShard == nil || !sharedClusterSource(st.clusterShard.t) {
		return nil
	}
	cell := st.clusterShard.coherence.Load()
	if cell == nil {
		return fmt.Errorf("transaction shard has no coherence cell")
	}
	cell = cell.resolved()
	if pending := cell.ownstate.Snapshot().Flight(); pending != nil {
		select {
		case <-pending.Done():
			return pending.Err()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// Decision publication and undo acquire native write rights once per shard.
func (tx *TxContext) claimDecisionRights() func() {
	shards := make([]*storageShard, 0)
	tx.mu.Lock()
	for shard := range tx.shardTransactions() {
		shards = append(shards, shard)
	}
	tx.mu.Unlock()
	sort.Slice(shards, func(i, j int) bool { return shards[i].uuid.String() < shards[j].uuid.String() })
	var releases []func()
	complete := false
	release := func() {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
	}
	defer func() {
		if !complete {
			release()
		}
	}()
	for _, shard := range shards {
		if r := activeReplication(shard.t.schema); r != nil && validReplicaDatabase(shard.t.schema.Name) && !shard.t.isEphemeralQueryTable() {
			cell := r.shardCell(shard)
			_, done := r.writeResource(*cell.resource.Load(), tx)
			releases = append(releases, done)
		}
		shard.ensureMutationLog()
	}
	complete = true
	return release
}
