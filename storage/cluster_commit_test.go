/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "testing"
import "context"
import "sync/atomic"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

func distributedCommitFixture(t *testing.T) (*TxContext, *TxContext, *storageShard, *storageShard) {
	t.Helper()
	local, _ := createDurabilityTestTable(t, "t_commit_local", 4)
	remote, _ := CreateTable(local.schema.Name, "remote_items", Safe, false)
	remote.CreateColumn("id", "INT", nil, nil)
	remote.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}, {scm.NewInt(2)}}, nil, scm.NewNil(), false, nil)
	mounts := map[string]bool{local.schema.Name: true}
	old := sharedClusterMounts.Swap(&mounts)
	t.Cleanup(func() { sharedClusterMounts.Store(old) })
	coordinator, worker := NewTxContext(TxACID), NewTxContext(TxACID)
	id := "cluster/" + coordinator.ClusterObjectID()
	coordinator.walTxID.Store(&id)
	worker.walTxID.Store(&id)
	worker.clusterID = coordinator.ClusterObjectID()
	left, right := local.ActiveShards()[0], remote.ActiveShards()[0]
	cluster := globalTestCluster(t, local.schema.Name)
	for _, shard := range []*storageShard{left, right} {
		cell := cluster.manager.shardCell(shard)
		cell.current.Store(shard)
		cell.ownstate.Initialize(mosi.M)
		cell.clusterstate.Initialize(mosi.M)
	}
	oldService := nodeScans.Swap(&nodeScanService{cluster: cluster})
	t.Cleanup(func() { nodeScans.Store(oldService) })
	coordinator.AddToDeleteMask(left, 0)
	worker.AddToDeleteMask(right, 0)
	participant := worker.getShardTx(right)
	participant.decisionToken = "test-capability"
	imported := &storageShardTransaction{clusterTxID: worker.clusterID, shardUUID: right.uuid, decisionToken: participant.decisionToken,
		remoteObject: &scm.ClusterHandle{Reference: scm.ClusterReference{Owner: "worker"}, Call: func(ctx context.Context, request scm.ClusterRequest) (scm.Scmer, error) {
			value, err := participant.HandleClusterRequest(ctx, request)
			if err != nil {
				return scm.NewNil(), err
			}
			return value.(scm.Scmer), nil
		}}}
	coordinator.participants = map[uuid.UUID]*storageShardTransaction{right.uuid: imported}
	return coordinator, worker, left, right
}

func TestClusterACIDCommitPreparesAllOwnersBeforePublication(t *testing.T) {
	tx, worker, left, right := distributedCommitFixture(t)
	remote := tx.participants[right.uuid].remoteObject
	call := remote.Call
	remote.Call = func(ctx context.Context, request scm.ClusterRequest) (scm.Scmer, error) {
		value, err := call(ctx, request)
		if request.Method == "prepare" && err == nil {
			if left.deletions.Get(0) || right.deletions.Get(0) {
				t.Error("prepare published private changes")
			}
			if readDistributedDecision(left.t.schema.persistence, tx.durableID()) {
				t.Error("commit decision preceded final prepare vote")
			}
		}
		return value, err
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if !left.deletions.Get(0) || !right.deletions.Get(0) || tx.State != TxCommitted || worker.State != TxCommitted {
		t.Fatal("commit missed a native owner")
	}
	if left.prepared.Load() != nil || right.prepared.Load() != nil {
		t.Fatal("commit retained prepared write admission")
	}
}

func TestClusterACIDPrepareFailureRollsBackEveryOwner(t *testing.T) {
	tx, worker, left, right := distributedCommitFixture(t)
	remote := tx.participants[right.uuid].remoteObject
	call := remote.Call
	remote.Call = func(ctx context.Context, request scm.ClusterRequest) (scm.Scmer, error) {
		value, err := call(ctx, request)
		if request.Method == "prepare" && err == nil {
			return scm.NewNil(), fmt.Errorf("lost prepare reply")
		}
		return value, err
	}
	if err := tx.Commit(); err == nil {
		t.Fatal("accepted failed preparation")
	}
	if tx.State != TxAborted || worker.State != TxAborted || left.deletions.Get(0) || right.deletions.Get(0) {
		t.Fatal("failed preparation changed committed data")
	}
	if readDistributedDecision(left.t.schema.persistence, tx.durableID()) {
		t.Fatal("failed preparation acquired commit authority")
	}
	if left.prepared.Load() != nil || right.prepared.Load() != nil {
		t.Fatal("rollback retained write admission")
	}
}

func TestClusterACIDLostCommitReplyCanBeRetried(t *testing.T) {
	tx, worker, left, right := distributedCommitFixture(t)
	remote := tx.participants[right.uuid].remoteObject
	call := remote.Call
	var failed atomic.Bool
	remote.Call = func(ctx context.Context, request scm.ClusterRequest) (scm.Scmer, error) {
		value, err := call(ctx, request)
		if request.Method == "commit" && err == nil && !failed.Swap(true) {
			return scm.NewNil(), fmt.Errorf("lost commit reply")
		}
		return value, err
	}
	if err := tx.Commit(); !commitOutcomeUnknown(err) {
		t.Fatalf("lost commit reply is not ambiguous: %v", err)
	}
	if worker.State != TxCommitted || tx.State != TxActive {
		t.Fatal("ambiguous decision lost its retry owner")
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if tx.State != TxCommitted || !left.deletions.Get(0) || !right.deletions.Get(0) {
		t.Fatal("retry lost committed effects")
	}
}

func TestClusterACIDRecoveryUsesCommonDecisionWithoutOwnerPublication(t *testing.T) {
	tx, _, left, _ := distributedCommitFixture(t)
	if err := tx.prepareACID(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() { tx.releasePreparation(); tx.rollbackACID() }()
	if left.deletions.Get(0) {
		t.Fatal("prepared deletion is visible")
	}
	engine := left.t.schema.persistence
	log := engine.SwapLog(distributedDecisionLog(tx.durableID()), []any{LogEntryCommit{txID: tx.durableID()}}, true)
	log.Close()
	// Replay the native WAL without a participant COMMIT marker. The shared
	// decision is sufficient authority to recover every prepared visibility intent.
	recovered := &storageShard{uuid: left.uuid, t: left.t, columns: make(map[string]ColumnStorage), deltaColumns: make(map[string]int)}
	recovered.mu.Lock()
	recovered.load()
	recovered.mu.Unlock()
	defer recovered.logfile.Close()
	if !recovered.deletions.Get(0) {
		t.Fatal("recovery discarded the common committed decision")
	}
}
