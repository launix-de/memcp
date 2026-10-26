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

func TestClusterShardRAMUsesMeasuredLedgerAndFraction(t *testing.T) {
	var cache CacheManager
	cache.Init(10<<20, 10<<20, 0)
	defer cache.Stop()
	cache.SetShardRAMPercent(50)
	if !cache.canLoadClusterShard() {
		t.Fatal("empty ledger rejected")
	}
	// A pinned native object cannot be evicted to manufacture admission.
	cache.AddItem(new(int), 5<<20, TypeShard, func(any, *[numEvictableTypes]int64) bool { return false }, func(any) time.Time { return time.Now() }, nil)
	if cache.canLoadClusterShard() {
		t.Fatal("admitted a cold shard at the 50% boundary")
	}
	cache.SetShardRAMPercent(60)
	if !cache.canLoadClusterShard() {
		t.Fatal("runtime fraction change was ignored")
	}
	stat := cache.Stat()
	if stat.ShardMemory != 5<<20 || stat.ShardBudget != 6<<20 {
		t.Fatal("ledger mismatch")
	}
}
func TestClusterWarmScanDoesNotReadBackendOrReserveRAM(t *testing.T) {
	shard := &storageShard{}
	shard.setState(SHARED)
	// No table/backend, peers or budget actor exist. A warm decision needs none.
	(&nodeCluster{}).prepareLocalScan(shard, nil)
}

func TestClusterRosterRequiresBoundIdentityAndRetainsSnapshot(t *testing.T) {
	self := clusterMember{"memcp://" + uuid.NewString(), uuid.NewString(), "ws://127.0.0.1:4001/cluster"}
	remote := clusterMember{"memcp://" + uuid.NewString(), uuid.NewString(), "ws://127.0.0.1:4002/cluster"}
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
	oldService := nodeScans.Load()
	outer := &replicationManager{}
	nodeScans.Store(&nodeScanService{cluster: &nodeCluster{manager: outer}})
	defer nodeScans.Store(oldService)
	if activeReplication(table.schema) != outer {
		t.Fatal("cluster source did not select its own domain")
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
