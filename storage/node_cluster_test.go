/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/
package storage

import "time"
import "context"
import "testing"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

func TestClusterQueryBudgetCreditsResidentGenerationOnce(t *testing.T) {
	const mib = int64(1 << 20)
	var cache CacheManager
	cache.Init(10*mib, 10*mib, 0)
	defer cache.Stop()
	shard := &storageShard{}
	shard.cacheReaders.Store(1)
	cache.AddItem(shard, 2*mib, TypeShard, func(any, *[numEvictableTypes]int64) bool { return false }, func(any) time.Time { return time.Now() }, nil)
	first, _ := cache.tryQueryRAM(shard, 4*mib, mib)
	if first == nil {
		t.Fatal("first operator rejected")
	}
	second, _ := cache.tryQueryRAM(shard, 4*mib, mib)
	if second == nil {
		first.Release()
		t.Fatal("shared resident bytes counted twice")
	}
	if cache.Stat().QueryReserved != 4*mib {
		t.Fatal("missing-payload/scratch reservation mismatch")
	}
	cache.SetSize(shard, 4*mib)
	if cache.Stat().QueryReserved != 2*mib {
		t.Fatal("loading did not convert reservation into resident credit")
	}
	other := &storageShard{}
	if unexpected, _ := cache.tryQueryRAM(other, 4*mib, mib); unexpected != nil {
		unexpected.Release()
		t.Fatal("over-budget work admitted")
	}
	first.Release()
	second.Release()
	if cache.Stat().QueryReserved != 0 {
		t.Fatal("operator reservation leaked")
	}
	final, _ := cache.tryQueryRAM(other, 4*mib, mib)
	if final == nil {
		t.Fatal("released capacity was not reusable")
	}
	final.Release()
	if stat := cache.Stat(); stat.QueryPeak > stat.MemoryBudget {
		t.Fatal("admission peak exceeded node budget")
	}
}

func TestClusterRosterRequiresBoundIdentityAndRetainsSnapshot(t *testing.T) {
	self := clusterMember{uuid.NewString(), uuid.NewString(), "ws://127.0.0.1:4001/cluster"}
	remote := clusterMember{uuid.NewString(), uuid.NewString(), "ws://127.0.0.1:4002/cluster"}
	cluster := &nodeCluster{self: self, members: map[string]clusterMember{self.ID: self}, links: make(map[string]*clusterLink)}
	cluster.publishLocked()
	before := cluster.view.Load()
	if err := cluster.remember("unproved-identity", clusterHello{1, remote, nil}); err == nil {
		t.Fatal("RPC assertion replaced authenticated identity")
	}
	if cluster.view.Load() != before {
		t.Fatal("rejected roster changed published view")
	}
	if err := cluster.remember(remote.ID, clusterHello{1, remote, []clusterMember{self}}); err != nil {
		t.Fatal(err)
	}
	if len(cluster.view.Load().members) != 2 || len(before.members) != 1 {
		t.Fatal("roster publication mutated an existing snapshot")
	}
}

func TestClusterHintCannotAccessUnMountedDatabase(t *testing.T) {
	old := sharedClusterMounts.Swap(nil)
	defer sharedClusterMounts.Store(old)
	cluster := &nodeCluster{}
	if _, err := cluster.handle(newScanRPCSession(), context.Background(), "peer", "cluster.hint", []byte(`{"resource":{"database":"system","table":"user"}}`)); err == nil {
		t.Fatal("cluster hint bypassed mount boundary")
	}
}

func TestClusterAdmissionRejectsUnknownColumnBeforeLoading(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_cluster_header_fence", 2)
	shard := table.ActiveShards()[0]
	// Fresh delta-only generations have no immutable column header. Placement
	// must refuse them before a native loader can use an unbounded row estimate.
	if _, _, err := estimateClusterShard(shard); err == nil {
		t.Fatal("delta-only generation admitted as immutable numeric shard")
	}
}

func TestClusterCapacityReleaseBeforeWatchCannotLoseWakeup(t *testing.T) {
	var cache CacheManager
	cache.Init(4<<20, 4<<20, 0)
	defer cache.Stop()
	lease, decision := cache.tryQueryRAM(&storageShard{}, 1<<20, 1<<20)
	if lease == nil {
		t.Fatal("missing test admission")
	}
	lease.Release()
	reply := make(chan (<-chan struct{}), 1)
	cache.opChan <- cacheOp{queryWatch: reply, queryVersion: decision.Version}
	signal := <-reply
	select {
	case <-signal:
	default:
		t.Fatal("release before watcher registration lost the capacity wakeup")
	}
}

func TestClusterMountCannotBypassAdmissionThroughExplicitScanRPC(t *testing.T) {
	old := sharedClusterMounts.Swap(nil)
	defer sharedClusterMounts.Store(old)
	peer, _, _ := setupRemoteScanConnection(t)
	if err := prepareSharedClusterMounts("mounted_snapshot"); err != nil {
		t.Fatal(err)
	}
	var reply any
	err := peer.call(context.Background(), "scan.describe", remoteScanDescribe{scanWireVersion, "mounted_snapshot", "items"}, &reply)
	if err == nil {
		t.Fatal("unbudgeted describe opened a mounted snapshot")
	}
	task := remoteScanTask{Shard: remoteScanShard{Resource: mosiResource{Database: "mounted_snapshot", Table: "items", Shard: uuid.NewString()}, Generation: uuid.New()}}
	err = peer.call(context.Background(), "scan.execute", task, &reply)
	if err == nil {
		t.Fatal("explicit scan RPC bypassed mandatory cluster admission")
	}
}

func TestClusterMountDoesNotSubscribeThroughReplication(t *testing.T) {
	oldMounts := sharedClusterMounts.Swap(nil)
	defer sharedClusterMounts.Store(oldMounts)
	table, _ := createDurabilityTestTable(t, "t_cluster_domain_boundary", 1)
	if err := prepareSharedClusterMounts(table.schema.Name); err != nil {
		t.Fatal(err)
	}
	manager := replicaRecoveryFixture()
	oldManager := replication.Swap(manager)
	defer replication.Store(oldManager)
	if activeReplication(table.schema) != nil {
		t.Fatal("shared snapshot entered the replication adapter")
	}
	if activeReplication(&database{Name: "separate_replica_tenant"}) != manager {
		t.Fatal("unrelated replication namespace was disabled")
	}
	tx := &TxContext{}
	session := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewString(table.schema.Name) })
	replicationEnter(tx, session)
	if tx.replicationActive || len(manager.directory.Records) != 0 {
		t.Fatal("opening a shared snapshot subscribed it to a replication parent")
	}
	request := replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{{Resource: mosiResource{Database: table.schema.Name}, State: "M", Owner: "slave", Epoch: 1}}}
	if _, err := manager.recoverPeer(context.Background(), "slave", request, replicaAuth{Admin: true}); err == nil {
		t.Fatal("replication recovery claimed a shared snapshot namespace")
	}
}
