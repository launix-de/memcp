/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "time"
import "sync"
import "testing"
import "github.com/launix-de/memcp/storage/mosi"
import "github.com/launix-de/memcp/scm"

func TestClusterSavepointFollowsPhysicalShardAcrossHandover(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_handover_savepoint", 3)
	source := tbl.ActiveShards()[0]
	tx := NewTxContext(TxACID)
	tx.AddToDeleteMask(source, 0)
	sp := tx.CreateSavepoint()
	tx.AddToDeleteMask(source, 1)
	original := tx.getShardTx(source)
	// A MOSI transfer reconstructs a distinct native Go object with the SAME
	// physical UUID and full journals; savepoints must survive that identity move.
	target := &storageShard{t: tbl, uuid: source.uuid}
	incoming := &storageShardTransaction{transaction: tx.transactionState, clusterShard: target,
		clusterTxID: original.clusterTxID, shardUUID: source.uuid,
		DeleteRecids: append([]uint32(nil), original.DeleteRecids...), DeleteMask: original.DeleteMask.Copy()}
	tx.mu.Lock()
	tx.shards.Store(&sync.Map{})
	tx.shards.Load().Store(target, incoming)
	source.overlays.Delete(tx.transactionState)
	source.endTransactionUse()
	if original.generation != nil {
		original.generation.releaseTransaction()
	}
	target.overlays.Store(tx.transactionState, incoming)
	target.beginTransactionUse()
	tx.bindShardEndpointLocked(incoming)
	tx.mu.Unlock()
	defer tx.Rollback()
	tx.RollbackToSavepoint(sp)
	if !incoming.DeleteMask.Get(0) || incoming.DeleteMask.Get(1) || len(incoming.DeleteRecids) != 1 {
		t.Fatal("rollback across a physical identity move undid pre-savepoint work or retained later work")
	}
	if source.deletions.Get(0) || source.deletions.Get(1) {
		t.Fatal("private savepoint rollback changed committed rows")
	}
}

func TestClusterDecisionWaitsForMigrationWhileReadersContinue(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_handover_decision", 2)
	shard := tbl.ActiveShards()[0]
	tx := NewTxContext(TxACID)
	tx.AddToDeleteMask(shard, 0)
	release, err := tx.beginMigration()
	if err != nil {
		t.Fatal(err)
	}
	started, done := make(chan struct{}), make(chan error, 1)
	go func() { close(started); done <- tx.Commit() }()
	<-started
	select {
	case err := <-done:
		release()
		t.Fatalf("commit passed an unfinished migration: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	access := shard.getRead(nil, true, false)
	if access.shard.deletions.Get(0) {
		t.Fatal("migration published a private deletion")
	}
	access.Release()
	release()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("commit did not resume after migration joined")
	}
	if !shard.deletions.Get(0) {
		t.Fatal("commit lost the transferred decision's native effect")
	}
}

func TestClusterNestedInvocationCanReturnToNativeCoordinator(t *testing.T) {
	root, invocation := NewTxContext(TxACID), NewTxContext(TxCursorStability)
	defer root.Rollback()
	value, err := root.BindClusterInvocation(scm.NewAny(invocation))
	if err != nil {
		t.Fatal(err)
	}
	bound := value.Any().(*TxContext)
	if bound.transactionState != root.transactionState || bound.remoteObject != nil {
		t.Fatal("nested return created a second coordinator or lost the native transaction")
	}
}

type captureFlushFailure struct{ PersistenceLogfile }

func (captureFlushFailure) Flush(bool) { panic("forced live capture flush failure") }

func TestClusterFailedLiveCaptureReleasesMaintenanceAndTransactionAdmission(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_handover_failed_capture", 2)
	shard := tbl.ActiveShards()[0]
	tx := NewTxContext(TxACID)
	tx.AddToDeleteMask(shard, 0)
	defer tx.Rollback()
	cluster := globalTestCluster(t, tbl.schema.Name)
	cell := cluster.manager.shardCell(shard)
	cell.current.Store(shard)
	cell.ownstate.Initialize(mosi.M)
	cell.clusterstate.Initialize(mosi.M)
	log := shard.logfile
	shard.logfile = captureFlushFailure{log}
	defer func() { shard.logfile = log }()
	if _, err := cluster.manager.prepareResource(*cell.resource.Load()); err == nil {
		t.Fatal("live capture accepted a failed WAL barrier")
	}
	if !tbl.maintenanceMu.TryLock() {
		t.Fatal("failed capture retained the table maintenance lock")
	}
	tbl.maintenanceMu.Unlock()
	tx.decisionMu.Lock()
	migrations := tx.migrations
	tx.decisionMu.Unlock()
	if migrations != 0 || cell.liveTransfer.Load() != nil || cell.ownstate.Phase() != mosi.M {
		t.Fatal("failed capture leaked migration admission, a transfer, or ownership")
	}
}
