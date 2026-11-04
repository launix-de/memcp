/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "time"
import "context"
import "sync"
import "testing"
import "github.com/launix-de/memcp/storage/mosi"
import "github.com/launix-de/memcp/scm"
import "github.com/google/uuid"

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
	access := shard.getRead(nil, shardReadOptions{snapshot: true})
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

func TestClusterCaptureRetainsOverlayWhenHelperBecomesPersistent(t *testing.T) {
	fixture, _ := createDurabilityTestTable(t, "t_handover_helper_engine", 1)
	tbl, _ := CreateTable(fixture.schema.Name, ".helper", Cache, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
	shard := tbl.ActiveShards()[0]
	tx := NewTxContext(TxACID)
	tx.AddToDeleteMask(shard, 0)
	defer tx.Rollback()
	// Engine changes can make a previously query-local descriptor eligible for
	// MOSI. Its already-open native overlay must remain discoverable for capture.
	tbl.PersistencyMode = Safe
	cluster := globalTestCluster(t, tbl.schema.Name)
	cell := cluster.manager.shardCell(shard)
	cell.current.Store(shard)
	cell.ownstate.Initialize(mosi.M)
	cell.clusterstate.Initialize(mosi.M)
	capture, err := cluster.manager.captureLiveShard(cell, shard)
	if err != nil {
		t.Fatal(err)
	}
	defer capture.release()
	if len(capture.overlays) != 1 || capture.overlays[0].transaction != tx.transactionState || !capture.overlays[0].DeleteMask.Get(0) {
		t.Fatal("engine change hid an active private overlay from MOSI handover")
	}
}

// The outbound M request proves the decision is waiting for physical ownership.
// Its transaction must remain migratable: the owner may need to transfer that
// same transaction's overlay to satisfy this request.
func TestClusterDecisionAcquiresWriteBeforeClosingMigration(t *testing.T) {
	for _, commit := range []bool{true, false} {
		name := "rollback"
		if commit {
			name = "commit"
		}
		t.Run(name, func(t *testing.T) {
			tbl, _ := createDurabilityTestTable(t, "t_decision_acquisition", 2)
			shard := tbl.ActiveShards()[0]
			earlier := NewShard(tbl)
			access := earlier.GetWrite(nil)
			earlier.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(42)}}, false, nil, nil, false, nil)
			access.Release()
			if earlier.uuid.String() > shard.uuid.String() {
				earlier, shard = shard, earlier
			}
			tx := NewTxContext(TxACID)
			tx.AddToDeleteMask(earlier, 0)
			tx.AddToDeleteMask(shard, 0)
			cluster := globalTestCluster(t, tbl.schema.Name)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cluster.ctx = ctx
			owner := clusterMember{ID: "memcp://" + uuid.NewString(), Boot: uuid.NewString()}
			peer := &mosiPeer{ctx: ctx, done: make(chan struct{}), outgoing: newMosiMailbox()}
			cluster.view.Store(&clusterView{members: []clusterMember{cluster.self, owner}, links: []*clusterLink{{member: owner, peer: peer}}})
			mounts := map[string]bool{tbl.schema.Name: true}
			previousMounts := sharedClusterMounts.Swap(&mounts)
			defer sharedClusterMounts.Store(previousMounts)
			previousService := nodeScans.Swap(&nodeScanService{cluster: cluster})
			defer nodeScans.Store(previousService)
			r := cluster.manager
			r.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
			first := r.shardCell(earlier)
			first.current.Store(earlier)
			first.ownstate.Initialize(mosi.M)
			cell := r.shardCell(shard)
			cell.current.Store(shard)
			cell.ownstate.Initialize(mosi.S)
			resource := *cell.resource.Load()
			r.updateRecord(resource, func(record *replicaRecord) bool {
				record.Resource, record.Owner, record.State, record.Epoch = resource, owner.ID, "S", 1
				return true
			})
			done := make(chan error, 1)
			go func() {
				if commit {
					done <- tx.Commit()
				} else {
					tx.Rollback()
					done <- nil
				}
			}()
			select {
			case <-peer.outgoing.ready:
				takeMosi(peer.outgoing)
			case <-time.After(5 * time.Second):
				t.Fatal("decision did not request native M")
			}
			if first.ownstate.Writing() {
				t.Error("decision retained an earlier writer while waiting for a remote M")
			}
			releaseMigration, migrationErr := tx.beginMigration()
			pending := cell.ownstate.Snapshot().Flight()
			if pending == nil {
				t.Fatal("decision no longer has an acquisition flight")
			}
			if _, err := cell.ownstate.Grant(pending.ID(), mosi.M, func() error { return nil }); err != nil {
				t.Fatal(err)
			}
			if migrationErr != nil {
				<-done
				t.Fatalf("decision waiting for M blocked its own handover: %v", migrationErr)
			}
			select {
			case err := <-done:
				releaseMigration()
				t.Fatalf("decision passed an unfinished migration: %v", err)
			case <-time.After(20 * time.Millisecond):
			}
			releaseMigration()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("decision did not finish after M grant and migration")
			}
			if shard.deletions.Get(0) != commit || earlier.deletions.Get(0) != commit {
				t.Fatal("decision lost the native transaction effect")
			}
		})
	}
}
