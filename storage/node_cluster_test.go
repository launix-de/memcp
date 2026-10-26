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
import "strings"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

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
	if err := cluster.remember("unproved-identity", clusterHello{scanWireVersion, remote, nil}); err == nil {
		t.Fatal("RPC assertion replaced authenticated identity")
	}
	if cluster.view.Load() != before {
		t.Fatal("rejected roster changed published view")
	}
	if err := cluster.remember(remote.ID, clusterHello{scanWireVersion, remote, []clusterMember{self}}); err != nil {
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

func TestClusterControlDoesNotConsumeComputeSlots(t *testing.T) {
	session := newScanRPCSession()
	session.service = &nodeScanService{cluster: &nodeCluster{}}
	for i := 0; i < cap(session.workers); i++ {
		session.workers <- struct{}{}
	}
	// Invalid input must reach the control handler even while every compute
	// slot is occupied. A busy reducer cannot prevent owner discovery.
	_, err := session.handle(context.Background(), "cluster.report", []byte(`{"database":"unmounted"}`))
	if err == nil || err.Error() != "invalid cluster report" {
		t.Fatalf("control message was blocked by compute admission: %v", err)
	}
	if len(session.workers) != cap(session.workers) {
		t.Fatal("control path borrowed a compute slot")
	}
}

func TestClusterRebuildUUIDAliasesShareAuthority(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}}
	cluster.initializeManager()
	r := cluster.manager
	old := mosiResource{Database: "alias_db", Table: "items", Shard: uuid.NewString()}
	cell := r.resourceCell(old)
	r.updateRecord(old, func(record *replicaRecord) bool {
		record.Owner, record.State, record.Epoch = r.id, "M", 7
		return true
	})
	next := &storageShard{uuid: uuid.New()}
	cluster.aliasShard(old, next)
	current := mosiResource{Database: old.Database, Table: old.Table, Shard: next.uuid.String()}
	if r.resourceCell(current) != cell || r.resourceCell(old) != cell {
		t.Fatal("rebuild created independently writable permission cells")
	}
	// Repeated publication can happen while translating an old native handle.
	// It must not erase the canonical owner record or create another directory.
	cluster.aliasShard(old, next)
	r.updateRecord(old, func(record *replicaRecord) bool { record.Generation = next.uuid.String(); return true })
	if len(r.directory.Records) != 1 || r.directory.Records[current.key()].Epoch != 7 || r.recordKey(old) != current.key() {
		t.Fatal("UUID forwarding lost canonical authority")
	}
}

func TestClusterQueuedRequestForwardsAfterOwnerHandover(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}}
	cluster.initializeManager()
	owner := "memcp://" + uuid.NewString()
	requester := "memcp://" + uuid.NewString()
	peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 1)}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: owner}, peer: peer}}})
	resource := mosiResource{Database: "forward_db"}
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Epoch, record.State = owner, 4, "IO"
		return true
	})
	request := mosiPacket{Kind: "write", Resource: resource, ID: uuid.NewString(), Sequence: 8}
	if err := cluster.manager.coordinate(context.Background(), requester, request); err != nil {
		t.Fatal(err)
	}
	message := <-peer.outgoing
	if message.Packet == nil || message.Packet.Kind != "write" || message.Packet.From != requester || message.Packet.To != owner || message.Packet.ID != request.ID {
		t.Fatal("old owner recalled its successor instead of forwarding the original request")
	}
	if cluster.manager.directory.Records[resource.key()].Epoch != 4 {
		t.Fatal("forwarding manufactured a new ownership epoch")
	}
}

func TestClusterCancelledReaderRetainsSharedRequestFlight(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}, ctx: context.Background()}
	cluster.initializeManager()
	owner := "memcp://" + uuid.NewString()
	peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 1)}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: owner}, peer: peer}}})
	resource := mosiResource{Database: "cancel_db"}
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Epoch = owner, 1
		return true
	})
	cell := cluster.manager.resourceCell(resource)
	request, _ := cell.ownstate.Request(mosi.S)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	func() {
		defer func() {
			if failure := recover(); failure != context.Canceled {
				t.Fatalf("wrong cancelled waiter failure: %v", failure)
			}
		}()
		cluster.obtain(ctx, resource, mosi.S)
	}()
	if cell.ownstate.Phase() != mosi.SR || request.Flight().Err() != nil || len(peer.outgoing) != 0 {
		t.Fatal("one reader cancellation aborted or duplicated the common flight")
	}
	if _, err := cell.ownstate.Grant(request.Flight().ID(), mosi.S, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if cluster.obtain(context.Background(), resource, mosi.S) != cell {
		t.Fatal("remaining readers could not use the received grant")
	}
}

func TestClusterLiveRevocationCannotRecoverOldOwnerRights(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}}
	cluster.initializeManager()
	cluster.view.Store(&clusterView{members: []clusterMember{cluster.self}})
	resource := mosiResource{Database: "live_revocation_db"}
	cell := cluster.manager.resourceCell(resource)
	cell.ownstate.Establish(mosi.IO)
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Epoch, record.State = cluster.self.ID, 9, "IO"
		return true
	})
	if _, err := cluster.locate(context.Background(), resource); err == nil || cell.ownstate.Phase() != mosi.IO {
		t.Fatal("live handover was mistaken for restart recovery and promoted")
	}
	successor := "memcp://" + uuid.NewString()
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool { record.Handoff = successor; return true })
	owner, err := cluster.locate(context.Background(), resource)
	if err != nil || owner != successor || cell.ownstate.Phase() != mosi.IO {
		t.Fatal("pending handover did not route to its successor without acquiring permission")
	}
}

func TestClusterColdHandoverRetainsEarlierOwnersCommitAuthority(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_cluster_commit_lineage", 1)
	db := tbl.schema
	first := "memcp://" + uuid.NewString()
	sender := "memcp://" + uuid.NewString()
	cluster := &nodeCluster{}
	cluster.view.Store(&clusterView{members: []clusterMember{{ID: first}, {ID: sender}}})
	old := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(old)
	for _, owner := range []string{first, sender} {
		_, entries, log := db.persistence.ReplayLog(transactionLogName + "." + strings.TrimPrefix(owner, "memcp://"))
		for range entries {
		}
		log.Write(LogEntryCommit{txID: owner})
		log.Flush(true)
		log.Close()
	}
	if err := db.importClusterCommits(sender); err != nil {
		t.Fatal(err)
	}
	if !db.transactionCommitted(first) || !db.transactionCommitted(sender) {
		t.Fatal("cold forwarding lost the original modifier's durable commit proof")
	}
}

func TestClusterGrantedSchemaReadDoesNotConsultSession(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_cluster_granted_schema", 1)
	cluster := &nodeCluster{databases: map[string]bool{tbl.schema.Name: true}}
	cluster.initializeManager()
	cell := cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name})
	cell.ownstate.Establish(mosi.S)
	mounts := map[string]bool{tbl.schema.Name: true}
	oldMounts := sharedClusterMounts.Swap(&mounts)
	defer sharedClusterMounts.Store(oldMounts)
	oldService := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(oldService)
	tx := &TxContext{Session: scm.NewFunc(func(...scm.Scmer) scm.Scmer { panic("granted schema read consulted user session") })}
	tbl.schema.GetRead(tx).Release()
}
