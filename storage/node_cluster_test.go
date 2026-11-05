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

import "fmt"
import "time"
import "context"
import "reflect"
import "strings"
import "testing"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

func TestClusterShardRAMUsesMeasuredLedgerAndFraction(t *testing.T) {
	var cache CacheManager
	cache.Init(10<<20, 10<<20, 0)
	defer cache.Stop()
	cache.SetShardRAMPercent(50)
	if !cache.canLoadClusterShard(nil) {
		t.Fatal("empty ledger rejected")
	}
	// A pinned native object cannot be evicted to manufacture admission.
	cache.AddItem(new(int), 5<<20, TypeShard, func(any, *[numEvictableTypes]int64) bool { return false }, func(any) time.Time { return time.Now() }, nil)
	if cache.canLoadClusterShard(nil) {
		t.Fatal("admitted a cold shard at the 50% boundary")
	}
	cache.SetShardRAMPercent(60)
	if !cache.canLoadClusterShard(nil) {
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
	(&nodeCluster{}).prepareLocalScan(shard, nil, false)
}

func TestClusterColdSubscanUsesLocalCapacityBeforeOwnerRouting(t *testing.T) {
	previous := GlobalCache.shardCapacity.Swap(&shardRAMSnapshot{Allowed: true})
	defer GlobalCache.shardCapacity.Store(previous)
	shard := &storageShard{t: &table{schema: &database{Name: "local_subscan"}, Name: "items"}, uuid: uuid.New()}
	shard.setState(COLD)
	cluster := &nodeCluster{}
	cluster.initializeManager()
	resource := mosiResource{Database: shard.t.schema.Name, Table: shard.t.Name, Shard: shard.uuid.String()}
	cluster.manager.resourceCell(resource).ownstate.Initialize(mosi.S)
	// Granted S plus available RAM needs no owner lookup or connection, even
	// when the physical columns have been evicted and must be loaded again.
	cluster.prepareLocalScan(shard, nil, false)
}

func TestClusterMutationUsesNodeBudgetBeyondReadWorkingSetShare(t *testing.T) {
	previous := GlobalCache.shardCapacity.Swap(&shardRAMSnapshot{Allowed: true})
	defer GlobalCache.shardCapacity.Store(previous)
	tbl := &table{}
	tbl.shardRAMFull.Store(true)
	if GlobalCache.canLoadClusterShard(tbl) {
		t.Fatal("read admission exceeded its per-table share")
	}
	tbl.schema, tbl.Name = &database{Name: "local_write"}, "items"
	shard := &storageShard{t: tbl, uuid: uuid.New()}
	shard.setState(COLD)
	// A multi-shard transaction cannot evict its modified generations. It may
	// use the remaining node shard budget without routing a fitting write away.
	(&nodeCluster{}).prepareLocalScan(shard, nil, true)
}

func TestClusterTableShardRAMLeavesRoomForInnerScans(t *testing.T) {
	var cache CacheManager
	cache.Init(30<<20, 30<<20, 0)
	defer cache.Stop()
	tables := []*table{{}, {}, {}}
	for _, tbl := range tables {
		pointer := &storageShard{t: tbl}
		cleanup := func(any, *[numEvictableTypes]int64) bool { return false }
		item := newSoftItem(pointer, 5<<20, TypeShard, cleanup, func(any) time.Time { return time.Now() }, nil, 0, 0)
		item.object = atomicCacheObject{pointer: pointer, cleanup: cleanup}
		cache.opChan <- cacheOp{add: item}
		cache.Stat()
		if cache.canLoadClusterShard(tbl) {
			t.Fatal("table exceeded one third of the shard budget")
		}
		for _, other := range tables {
			if other != tbl && !other.shardRAMFull.Load() && !cache.canLoadClusterShard(other) && cache.Stat().ShardMemory < 15<<20 {
				t.Fatal("outer table consumed the inner table's admission share")
			}
		}
	}
	if cache.canLoadClusterShard(nil) {
		t.Fatal("three pinned tables exceeded the total shard budget")
	}
}

func TestClusterAdmissionPublicationIgnoresCacheAllocations(t *testing.T) {
	var cache CacheManager
	cache.Init(30<<20, 30<<20, 0)
	defer cache.Stop()
	before := cache.shardCapacity.Load()
	pointer := new(int)
	cache.AddItem(pointer, 1<<20, TypeIndex,
		func(any, *[numEvictableTypes]int64) bool { return false }, func(any) time.Time { return time.Now() }, nil)
	cache.UpdateSize(pointer, 1<<20)
	if cache.shardCapacity.Load() != before {
		t.Fatal("index allocation invalidated the published shard admission cache line")
	}
	cache.AddItem(new(int), 1<<20, TypeShard,
		func(any, *[numEvictableTypes]int64) bool { return false }, func(any) time.Time { return time.Now() }, nil)
	if cache.shardCapacity.Load() != before {
		t.Fatal("payload allocation below the threshold republished unchanged admission")
	}
}

func TestClusterColdPlacementUsesAllPeersAndKeepsWarmPreference(t *testing.T) {
	links := []*clusterLink{{}, {}, {}}
	cluster := &nodeCluster{}
	cluster.view.Store(&clusterView{links: links})
	executor := &clusterScanExecutor{cluster: cluster}
	used := make(map[*clusterLink]bool)
	for i := 0; i < 64; i++ {
		executor.expected.Resource.Shard = fmt.Sprintf("shard-%d", i)
		candidates := executor.candidates()
		used[candidates[0]] = true
		if executor.candidates()[0] != candidates[0] {
			t.Fatal("repeat access lost shard affinity")
		}
		seen := make(map[*clusterLink]bool)
		for _, candidate := range candidates {
			seen[candidate] = true
		}
		if len(candidates) != len(links) || len(seen) != len(links) {
			t.Fatal("capacity fallback omitted or duplicated a peer")
		}
	}
	if len(used) != len(links) {
		t.Fatalf("equal-cost placement used %d of %d peers", len(used), len(links))
	}
	executor.preferred = links[1]
	if candidates := executor.candidates(); candidates[0] != links[1] || len(candidates) != len(links) {
		t.Fatal("shard affinity overrode warm residency")
	}
	cluster.view.Store(&clusterView{})
	executor.preferred = nil
	if len(executor.candidates()) != 0 {
		t.Fatal("empty roster produced candidates")
	}
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
	for _, version := range []int{18, scanWireVersion + 1} {
		if err := cluster.remember(remote.ID, clusterHello{version, remote, []clusterMember{self}}); err == nil {
			t.Fatalf("accepted incompatible wire version %d", version)
		}
		if cluster.view.Load() != before {
			t.Fatal("incompatible peer changed published roster")
		}
	}
	if err := cluster.remember(remote.ID, clusterHello{scanWireVersion, remote, []clusterMember{self}}); err != nil {
		t.Fatal(err)
	}
	if len(cluster.view.Load().members) != 2 || len(before.members) != 1 {
		t.Fatal("roster publication mutated an existing snapshot")
	}
}

func TestClusterRetiredResidencyProbeIsRejected(t *testing.T) {
	old := sharedClusterMounts.Swap(nil)
	defer sharedClusterMounts.Store(old)
	cluster := &nodeCluster{}
	if _, err := cluster.handle(newScanRPCSession(), context.Background(), "peer", "cluster.hint", []byte(`{"resource":{"database":"system","table":"user"}}`)); err == nil {
		t.Fatal("retired residency probe is still supported")
	}
}

func TestClusterMountCannotBypassAdmissionThroughExplicitScanRPC(t *testing.T) {
	old := sharedClusterMounts.Swap(nil)
	defer sharedClusterMounts.Store(old)
	peer, service, _ := setupRemoteScanConnection(t)
	if err := prepareSharedClusterMounts("mounted_snapshot"); err != nil {
		t.Fatal(err)
	}
	var reply any
	err := service.control(context.Background(), peer, "scan.describe", remoteScanDescribe{scanWireVersion, "mounted_snapshot", "items"}, &reply, true)
	if err == nil {
		t.Fatal("unbudgeted describe opened a mounted snapshot")
	}
	task := remoteScanTask{Shard: remoteScanShard{Resource: mosiResource{Database: "mounted_snapshot", Table: "items", Shard: uuid.NewString()}}}
	err = service.control(context.Background(), peer, "scan.execute", task, &reply, true)
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
	tx := NewTxContext(TxCursorStability)
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
	// Control discovery has no compute semaphore. Its domain handler must be
	// reachable independently of the UUID scopes of concurrently running scans.
	_, err := session.handle(context.Background(), "cluster.report", mustNodeMetadata(mosiResource{Database: "unmounted"}))
	if err == nil || err.Error() != "invalid cluster report" {
		t.Fatalf("control message was blocked by compute admission: %v", err)
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
	r.forwardResource(old, mosiResource{Database: old.Database, Table: old.Table, Shard: next.uuid.String()})
	current := mosiResource{Database: old.Database, Table: old.Table, Shard: next.uuid.String()}
	if r.resourceCell(current) != cell || r.resourceCell(old) != cell {
		t.Fatal("rebuild created independently writable permission cells")
	}
	// Repeated publication can happen while translating an old native handle.
	// It must not erase the canonical owner record or create another directory.
	r.forwardResource(old, mosiResource{Database: old.Database, Table: old.Table, Shard: next.uuid.String()})
	r.updateRecord(old, func(record *replicaRecord) bool { record.Content = next.uuid.String(); return true })
	if len(r.directory.Records) != 1 || r.directory.Records[current.key()].Epoch != 7 || r.resourceKey(old) != current.key() {
		t.Fatal("UUID forwarding lost canonical authority")
	}
}

func TestClusterQueuedRequestForwardsAfterOwnerHandover(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}}
	cluster.initializeManager()
	owner := "memcp://" + uuid.NewString()
	requester := "memcp://" + uuid.NewString()
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
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
	message := takeMosi(peer.outgoing)
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
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
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
	if cell.ownstate.Phase() != mosi.SR || request.Flight().Err() != nil || peer.outgoing.len() != 0 {
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
		_, entries, log := db.persistence.ReplayLog(transactionLogName+"."+strings.TrimPrefix(owner, "memcp://"), false)
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
	tx := NewTxContext(TxCursorStability)
	cluster := &nodeCluster{databases: map[string]bool{tbl.schema.Name: true}}
	cluster.initializeManager()
	cell := cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name})
	cell.ownstate.Establish(mosi.S)
	mounts := map[string]bool{tbl.schema.Name: true}
	oldMounts := sharedClusterMounts.Swap(&mounts)
	defer sharedClusterMounts.Store(oldMounts)
	oldService := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(oldService)
	tx.Session = scm.NewFunc(func(...scm.Scmer) scm.Scmer { panic("granted schema read consulted user session") })
	tbl.schema.GetRead(tx).Release()
}

func TestClusterDiscoveryIgnoresUnavailableNonOwner(t *testing.T) {
	for _, scenario := range []string{"live-owner", "dead-owner", "restart-with-missing-peer"} {
		t.Run(scenario, func(t *testing.T) {
			cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}, service: &nodeScanService{}}
			cluster.initializeManager()
			dead := clusterMember{ID: "memcp://" + uuid.NewString()}
			live := clusterMember{ID: "memcp://" + uuid.NewString()}
			unavailable := &mosiPeer{done: make(chan struct{})}
			close(unavailable.done)
			available := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
			cluster.view.Store(&clusterView{members: []clusterMember{cluster.self, dead, live}, links: []*clusterLink{{member: dead, peer: unavailable}, {member: live, peer: available}}})
			resource := mosiResource{Database: "discovery_db"}
			cell := cluster.manager.resourceCell(resource)
			if scenario != "restart-with-missing-peer" {
				cell.ownstate.Establish(mosi.IO)
			}
			cluster.manager.updateRecord(resource, func(record *replicaRecord) bool { record.Owner, record.Epoch = dead.ID, 6; return true })
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			go func() {
				select {
				case <-available.outgoing.ready:
					message := takeMosi(available.outgoing)
					var request struct {
						Return string `json:"return"`
					}
					if decodeTestInvocation(message.Channel, message.Payload, &request) != nil {
						return
					}
					answer := replicaRecord{Resource: resource, State: "S", Owner: dead.ID, Epoch: 7}
					if scenario == "live-owner" {
						answer.State, answer.Owner = "M", live.ID
					}
					if scenario == "restart-with-missing-peer" {
						answer.Owner = cluster.self.ID
					}
					if value, ok := cluster.service.handles.Load(controlReturnSlot(request.Return)); ok {
						deliverTestCompletion(t, value, scanCompletion{Data: mustNodeMetadata(answer)})
					}
				case <-ctx.Done():
				}
			}()
			owner, err := cluster.locate(ctx, resource)
			want := dead.ID
			if scenario == "live-owner" {
				want = live.ID
			}
			if scenario == "restart-with-missing-peer" {
				if err == nil || cell.ownstate.Initialized() {
					t.Fatal("incomplete discovery recovered owner permission")
				}
				return
			}
			if err != nil || owner != want {
				t.Fatalf("owner lookup = %s, %v; want %s", owner, err, want)
			}
			if cell.ownstate.Phase() != mosi.IO || cluster.manager.directory.Retired[dead.ID] || len(cluster.view.Load().members) != 3 {
				t.Fatal("discovery reclaimed rights or retired the unavailable participant")
			}
		})
	}
}

func TestClusterRejectedReferenceDoesNotPublishForwarding(t *testing.T) {
	for _, reason := range []string{"stale", "sender", "flight", "cross-table"} {
		t.Run(reason, func(t *testing.T) {
			cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}}
			cluster.initializeManager()
			r := cluster.manager
			resource := mosiResource{Database: "reference_db", Table: "items", Shard: uuid.NewString()}
			successor := resource
			successor.Shard = uuid.NewString()
			owner := "memcp://" + uuid.NewString()
			cell := r.resourceCell(resource)
			cell.ownstate.Initialize(mosi.S)
			r.updateRecord(resource, func(record *replicaRecord) bool {
				record.Owner, record.State, record.Epoch = owner, "S", 5
				return true
			})
			packet := mosiPacket{Resource: resource, ID: uuid.NewString(), Epoch: 5, Content: successor.Shard, Forwards: []shardForward{{resource, successor}}}
			source := owner
			switch reason {
			case "stale":
				packet.Epoch = 4
			case "sender":
				source = "memcp://" + uuid.NewString()
			case "cross-table":
				packet.Forwards[0].From.Table = "other"
			case "flight":
				cell.ownstate.Publish(mosi.IO)
				cell.ownstate.Request(mosi.S)
			}
			failure := map[string]string{"stale": "stale shared reference", "sender": "unsolicited shared reference", "flight": "unexpected shared reference request", "cross-table": "invalid transfer forwarding"}[reason]
			if err := cluster.installReference(source, packet); err == nil || !strings.Contains(err.Error(), failure) {
				t.Fatal("reference was not rejected for its invalid header", err)
			}
			if r.canonicalResource(resource) != resource || r.lookupCell(successor) != nil || *cell.resource.Load() != resource {
				t.Fatal("rejected reference changed UUID routing")
			}
			if record := r.directory.Records[resource.key()]; record.Owner != owner || record.Epoch != 5 {
				t.Fatal("rejected reference changed ownership", record)
			}
		})
	}
}

func TestClusterAccessCancellationKeepsCommonFlight(t *testing.T) {
	for _, write := range []bool{false, true} {
		t.Run(fmt.Sprint("write=", write), func(t *testing.T) {
			nodeCtx, stopNode := context.WithCancel(context.Background())
			defer stopNode()
			cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}, ctx: nodeCtx}
			cluster.initializeManager()
			owner := "memcp://" + uuid.NewString()
			peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
			cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: owner}, peer: peer}}})
			tbl := &table{schema: &database{Name: "cancel_access_db"}, Name: "items"}
			shard := &storageShard{t: tbl, uuid: uuid.New()}
			r := cluster.manager
			resource := r.shardResource(shard)
			cell := r.shardCell(shard)
			cell.ownstate.Initialize(mosi.IO)
			r.resourceCell(mosiResource{Database: resource.Database}).ownstate.Initialize(mosi.S)
			r.updateRecord(resource, func(record *replicaRecord) bool {
				record.Owner, record.Epoch = owner, 1
				return true
			})
			wanted := mosi.S
			if write {
				wanted = mosi.M
			}
			request, _ := cell.ownstate.Request(wanted)
			ctx, cancel := context.WithCancel(context.Background())
			ss := &scm.SessionState{}
			seq := ss.BeginQuery("Query", "cancel admission")
			ss.SetQueryContext(seq, ctx)
			tx := NewTxContext(TxCursorStability)
			tx.beginQuery(ss, seq, "cancel admission")
			defer tx.endQuery(seq)
			cancel()
			done := make(chan any, 1)
			go func() {
				defer func() { done <- recover() }()
				if write {
					_, release, _ := r.writeResource(resource, tx, true)
					release()
				} else {
					r.readShard(shard, tx)
				}
			}()
			select {
			case failure := <-done:
				if failure != context.Canceled {
					t.Fatal("wrong cancelled access result", failure)
				}
			case <-time.After(time.Second):
				stopNode()
				<-done
				t.Fatal("access ignored query cancellation")
			}
			if request.Flight().Err() != nil || peer.outgoing.len() != 0 {
				t.Fatal("cancelled access changed the common request")
			}
			if _, err := cell.ownstate.Grant(request.Flight().ID(), wanted, func() error { return nil }); err != nil {
				t.Fatal(err)
			}
			if r.obtain(resource, wanted, "", nil) != cell {
				t.Fatal("remaining waiter lost the completed grant")
			}
		})
	}
}

func TestClusterReferenceKeepsAcquisitionID(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "cluster_reference_request", 0)
	cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}}
	cluster.initializeManager()
	r := cluster.manager
	requester := "memcp://" + uuid.NewString()
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: requester}, peer: peer}}})
	resource := mosiResource{Database: tbl.schema.Name}
	cell := r.resourceCell(resource)
	cell.ownstate.Establish(mosi.M)
	cell.clusterstate.Establish(mosi.M)
	r.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.State, record.Epoch = r.id, "M", 1
		return true
	})
	request := mosiPacket{Kind: "read", ID: uuid.NewString(), Resource: resource, Sequence: 1}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- r.coordinate(ctx, requester, request) }()
	select {
	case <-peer.outgoing.ready:
		message := takeMosi(peer.outgoing)
		if message.Packet == nil || message.Packet.Kind != "reference" || message.Packet.ID != request.ID {
			t.Fatal("reference lost the pending acquisition identity", message.Packet)
		}
		step, ok := r.steps.Load(request.ID)
		if !ok {
			t.Fatal("reference has no matching installation acknowledgement")
		}
		step.(*replicaStep).finish(nil)
	case <-ctx.Done():
		t.Fatal("missing reference", ctx.Err())
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestClusterSchemaLocationsAreRoutingNotPermissions(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}, databases: map[string]bool{"tenant": true}}
	cluster.initializeManager()
	schema := cluster.manager.resourceCell(mosiResource{Database: "tenant"})
	cluster.manager.updateRecord(mosiResource{Database: "tenant"}, func(record *replicaRecord) bool { record.Owner = "schema-owner"; return true })
	pending, _ := schema.ownstate.Request(mosi.S)
	resource := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	entry := replicaRecord{Resource: resource, Owner: "shard-owner", Epoch: 9, Content: resource.Shard}
	request := clusterSchemaLocations{Header: mosiPacket{Resource: mosiResource{Database: "tenant"}, ID: pending.Flight().ID()}, Entries: []replicaRecord{entry}}
	if err := cluster.installSchemaLocations("foreign", request); err == nil {
		t.Fatal("foreign schema source accepted")
	}
	if err := cluster.installSchemaLocations("schema-owner", request); err != nil {
		t.Fatal(err)
	}
	cell := cluster.manager.resourceCell(resource)
	if cell.ownstate.Phase() != mosi.IO || cell.clusterstate.Phase() != mosi.IO || cell.ownstate.Initialized() {
		t.Fatal("location established authority")
	}
	record := cluster.manager.directory.Records[resource.key()]
	if record.Owner != entry.Owner || record.Epoch != entry.Epoch {
		t.Fatal("location omitted", record)
	}
	cell.ownstate.Initialize(mosi.S)
	entry.Owner, entry.Epoch = "late-owner", 10
	cluster.installLocationHint(entry)
	if cluster.manager.directory.Records[resource.key()].Owner != "shard-owner" {
		t.Fatal("routing overwrote an admitted native reader")
	}
	request.Header.ID = "old-flight"
	if err := cluster.installSchemaLocations("schema-owner", request); err == nil {
		t.Fatal("late schema routing accepted")
	}
}

// A request can be forwarded after its recorded schema owner handed over M.
// Location batches and the native reference must accept the same new epoch,
// without granting permission from the location metadata itself.
func TestClusterSchemaLocationsFollowForwardedAcquisition(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}, databases: map[string]bool{"tenant": true}}
	cluster.initializeManager()
	resource := mosiResource{Database: "tenant"}
	schema := cluster.manager.resourceCell(resource)
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Epoch = "previous-owner", 5
		return true
	})
	pending, _ := schema.ownstate.Request(mosi.M)
	shard := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	request := clusterSchemaLocations{
		Header:  mosiPacket{Resource: resource, ID: pending.Flight().ID(), Epoch: 5},
		Entries: []replicaRecord{{Resource: shard, Owner: "shard-owner", Epoch: 3}},
	}
	if err := cluster.installSchemaLocations("successor-owner", request); err == nil {
		t.Fatal("foreign same-epoch location transfer accepted")
	}
	request.Header.Epoch = 6
	if err := cluster.installSchemaLocations("successor-owner", request); err != nil {
		t.Fatal("forwarded native acquisition rejected", err)
	}
	if schema.ownstate.Phase() != mosi.MR || cluster.manager.resourceCell(shard).ownstate.Initialized() {
		t.Fatal("location batch published native permission")
	}
	request.Header.ID = "another-flight"
	if err := cluster.installSchemaLocations("successor-owner", request); err == nil {
		t.Fatal("foreign acquisition accepted")
	}
}

func TestClusterShardLocationQueriesOnlySchemaOwner(t *testing.T) {
	service := &nodeScanService{}
	schemaPeer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	unrelated := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	cluster := &nodeCluster{self: clusterMember{ID: "local"}, service: service}
	cluster.initializeManager()
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: "schema-owner"}, peer: schemaPeer}, {member: clusterMember{ID: "unrelated"}, peer: unrelated}}})
	schemaResource := mosiResource{Database: "tenant"}
	schema := cluster.manager.resourceCell(schemaResource)
	schema.ownstate.Initialize(mosi.S)
	cluster.manager.updateRecord(schemaResource, func(record *replicaRecord) bool { record.Owner = "schema-owner"; return true })
	resource := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	finished := make(chan string, 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() {
		owner, err := cluster.locate(ctx, resource)
		if err != nil {
			finished <- err.Error()
		} else {
			finished <- owner
		}
	}()
	message := takeMosi(schemaPeer.outgoing)
	var request struct {
		Return, Method string
		Data           []byte
	}
	if err := decodeTestInvocation(message.Channel, message.Payload, &request); err != nil || request.Method != "cluster.location" {
		t.Fatal("per-shard discovery did not use schema owner", request, err)
	}
	route, _ := service.handles.Load(controlReturnSlot(request.Return))
	deliverTestCompletion(t, route, scanCompletion{Data: mustNodeMetadata(replicaRecord{Resource: resource, Owner: "schema-owner", Epoch: 3})})
	if owner := <-finished; owner != "schema-owner" {
		t.Fatal(owner)
	}
	if unrelated.outgoing.len() != 0 {
		t.Fatal("queried unrelated peer for shard placement")
	}
	owner, err := cluster.locate(ctx, resource)
	if err != nil || owner != "schema-owner" || schemaPeer.outgoing.len() != 0 {
		t.Fatal("known route performed another probe", owner, err)
	}
}

func TestClusterSchemaLocationTransferUsesBoundedBatches(t *testing.T) {
	service := &nodeScanService{}
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	cluster := &nodeCluster{self: clusterMember{ID: "local"}, service: service}
	cluster.initializeManager()
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: "target"}, peer: peer}}})
	for i := 0; i < 131; i++ {
		resource := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
		cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
			record.Owner, record.Epoch = "owner", 3
			record.Readers = map[string]bool{"reader": true}
			return true
		})
	}
	counts := make(chan []int, 1)
	go func() {
		var batches []int
		for i := 0; i < 3; i++ {
			message := takeMosi(peer.outgoing)
			var request struct {
				Return string
				Data   clusterSchemaLocations
			}
			if err := decodeTestInvocation(message.Channel, message.Payload, &request); err != nil {
				t.Error(err)
				return
			}
			batches = append(batches, len(request.Data.Entries))
			for _, entry := range request.Data.Entries {
				if len(entry.Readers) != 0 || len(entry.Requests) != 0 {
					t.Error("schema directory copied shard reader/operation ownership")
				}
			}
			route, _ := service.handles.Load(controlReturnSlot(request.Return))
			deliverTestCompletion(t, route, scanCompletion{})
		}
		counts <- batches
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := cluster.transferSchemaLocations(ctx, "target", mosiPacket{Resource: mosiResource{Database: "tenant"}, ID: "schema-flight"}); err != nil {
		t.Fatal(err)
	}
	batches := <-counts
	if len(batches) != 3 || batches[0] != 64 || batches[1] != 64 || batches[2] != 3 {
		t.Fatal("unbounded or incomplete location transfer", batches)
	}
}

func TestClusterSchemaLocationsRetainNativeUUIDAliases(t *testing.T) {
	cluster := &nodeCluster{}
	cluster.initializeManager()
	old := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	current := old
	current.Shard = uuid.NewString()
	cluster.manager.updateRecord(old, func(record *replicaRecord) bool { record.Owner, record.Epoch = "owner", 9; return true })
	cluster.manager.forwardResource(old, current)
	entries := cluster.schemaLocations("tenant", "")
	if len(entries) != 2 {
		t.Fatal("native UUID route disappeared during schema transfer", entries)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Resource.Shard] = true
		if entry.Owner != "owner" || entry.Epoch != 9 {
			t.Fatal("UUID route lost current ownership", entry)
		}
	}
	if !seen[old.Shard] || !seen[current.Shard] {
		t.Fatal("missing predecessor or successor", entries)
	}
	if cluster.manager.resourceCell(old).ownstate.Initialized() {
		t.Fatal("inventory established rights")
	}
}

func TestClusterSchemaRecoveryRestoresBoundedRoutesWithoutPermissions(t *testing.T) {
	service := &nodeScanService{}
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	local := &nodeCluster{self: clusterMember{ID: "local"}, service: service}
	local.initializeManager()
	owner := &nodeCluster{self: clusterMember{ID: "live"}, databases: map[string]bool{"tenant": true}}
	owner.initializeManager()
	owner.view.Store(&clusterView{members: []clusterMember{owner.self, local.self}})
	resources := make([]mosiResource, 131)
	for i := range resources {
		resources[i] = mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
		owner.manager.updateRecord(resources[i], func(record *replicaRecord) bool {
			record.Owner, record.Epoch = "modifier", 7
			record.Readers = map[string]bool{"reader": true}
			return true
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- local.recoverSchemaLocations(ctx, "tenant", []*clusterLink{{member: owner.self, peer: peer}})
	}()
	for _, count := range []int{64, 64, 3} {
		select {
		case <-peer.outgoing.ready:
			message := takeMosi(peer.outgoing)
			var request struct {
				Return string
				Data   []byte
			}
			if err := decodeTestInvocation(message.Channel, message.Payload, &request); err != nil {
				t.Fatal(err)
			}
			value, err := owner.handleLocations(ctx, local.self.ID, "cluster.schema_locations", request.Data)
			if err != nil {
				t.Fatal(err)
			}
			page := value.(clusterSchemaLocations)
			if len(page.Entries) != count {
				t.Fatal("unbounded recovery batch", len(page.Entries))
			}
			route, _ := service.handles.Load(controlReturnSlot(request.Return))
			deliverTestCompletion(t, route, scanCompletion{Data: mustNodeMetadata(page)})
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	for _, resource := range resources {
		cell := local.manager.resourceCell(resource)
		if cell.ownstate.Initialized() || cell.clusterstate.Initialized() {
			t.Fatal("recovery hint granted authority")
		}
		record := local.manager.directory.Records[resource.key()]
		if record.Owner != "modifier" || record.Epoch != 7 || len(record.Readers) != 0 {
			t.Fatal("recovery lost or copied ownership", record)
		}
	}
}

func TestClusterGetReadDelegationIsAnExplicitCallerCapability(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "treadcapability", 4)
	RebuildTable(tbl, true, false)
	shard := tbl.ActiveShards()[0]
	var freed [numEvictableTypes]int64
	if !shardCleanup(shard, &freed) {
		t.Fatal("fixture did not become cold")
	}
	cluster := &nodeCluster{self: clusterMember{ID: "local"}, ctx: context.Background(), databases: map[string]bool{tbl.schema.Name: true}}
	cluster.initializeManager()
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: "owner"}, peer: &mosiPeer{done: make(chan struct{})}}}})
	cell := cluster.declareShard(shard, false)
	cell.ownstate.Initialize(mosi.S)
	cell.clusterstate.Initialize(mosi.S)
	cluster.manager.updateRecord(*cell.resource.Load(), func(record *replicaRecord) bool { record.Owner, record.Epoch = "owner", 3; return true })
	oldMounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(oldMounts)
	oldService := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(oldService)
	// Ownership alone no longer delegates: this explicit operator capability
	// is exercised only when local shard admission refuses the job.
	previousAdmission := GlobalCache.shardCapacity.Swap(&shardRAMSnapshot{Allowed: false})
	defer GlobalCache.shardCapacity.Store(previousAdmission)
	delegated := false
	func() {
		defer func() {
			if failure := recover(); failure != nil {
				if _, ok := failure.(shardReadDelegation); ok {
					delegated = true
				} else {
					panic(failure)
				}
			}
		}()
		access := shard.GetRead(nil, true)
		access.Release()
	}()
	if !delegated || shard.state() != COLD || shard.cacheReaders.Load() != 0 {
		t.Fatal("operator read did not delegate before local payload acquisition")
	}
	GlobalCache.shardCapacity.Store(previousAdmission)
	access := shard.GetRead(nil, false)
	defer access.Release()
	if access.shard != shard || shard.state() == COLD || shard.main_count != 4 {
		t.Fatal("native payload reader was incorrectly delegated")
	}
}

func TestClusterSchemaLocationConsumesNativeRebuildForwarding(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tlocationforwarding", 4)
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	from := mosiResource{Database: tbl.schema.Name, Table: tbl.Name, Shard: uuid.NewString()}
	to := from
	to.Shard = uuid.NewString()
	cell := cluster.manager.resourceCell(from)
	cluster.manager.updateRecord(from, func(record *replicaRecord) bool { record.Owner, record.Epoch = "remote", 7; return true })
	// The shared schema checkpoint has advanced its existing UUID edge while
	// this node's table payload/catalog remains unloaded and unchanged.
	data, _ := json.Marshal(map[string]any{"Name": tbl.schema.Name, "Tables": map[string]any{tbl.Name: map[string]any{"shard_forwards": map[string]string{from.Shard: to.Shard}}}})
	backend := tbl.schema.persistence
	tbl.schema.persistence = clusterSchemaView{backend, data}
	defer func() { tbl.schema.persistence = backend }()
	entry, err := cluster.shardLocation(to)
	if err != nil || entry.Resource != to || entry.Owner != "remote" || entry.Epoch != 7 {
		t.Fatal("schema owner lost rebuild routing", entry, err)
	}
	if cluster.manager.resourceCell(to) != cell.resolved() || cell.resolved().ownstate.Phase() != mosi.IO || cell.resolved().ownstate.Initialized() {
		t.Fatal("UUID routing created backend permission")
	}
}

func TestClusterSchemaLocationCannotPromoteRevokedSelf(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	cluster.view.Store(&clusterView{members: []clusterMember{cluster.self}})
	schemaResource := mosiResource{Database: "tenant"}
	schema := cluster.manager.resourceCell(schemaResource)
	schema.ownstate.Initialize(mosi.M)
	cluster.manager.updateRecord(schemaResource, func(record *replicaRecord) bool { record.Owner = "local"; record.Epoch = 3; return true })
	resource := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	cell := cluster.manager.resourceCell(resource)
	cell.ownstate.Initialize(mosi.IO)
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool { record.Owner = "local"; record.Epoch = 3; return true })
	if owner, err := cluster.locate(context.Background(), resource); err == nil {
		t.Fatal("revoked local rights became backend authority", owner)
	}
	if cell.ownstate.Phase() != mosi.IO {
		t.Fatal("location lookup changed permission")
	}
}

func TestClusterSchemaLocationKeepsAcceptedOwnerAfterStaleSelfHint(t *testing.T) {
	for _, fixture := range []struct {
		epoch   uint64
		handoff bool
	}{{7, false}, {8, false}, {7, true}, {8, true}} {
		hintEpoch := fixture.epoch
		t.Run(fmt.Sprint(hintEpoch, "/handoff=", fixture.handoff), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			service := &nodeScanService{}
			cluster := &nodeCluster{self: clusterMember{ID: "local"}, service: service, ctx: ctx}
			cluster.initializeManager()
			service.cluster = cluster
			peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
			cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: "schema-owner"}, peer: peer}}})
			schemaResource := mosiResource{Database: "tenant"}
			cluster.manager.resourceCell(schemaResource).ownstate.Initialize(mosi.S)
			cluster.manager.updateRecord(schemaResource, func(record *replicaRecord) bool {
				record.Owner, record.Epoch = "schema-owner", 1
				return true
			})
			resource := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
			cell := cluster.manager.resourceCell(resource)
			cell.ownstate.Initialize(mosi.IO)
			cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
				record.Owner, record.Epoch = "actual-owner", 8
				if fixture.handoff {
					record.Owner, record.Handoff = "local", "actual-owner"
				}
				return true
			})
			finished := make(chan error, 1)
			go func() {
				owner, err := cluster.locateViaSchema(ctx, resource)
				if err == nil && owner != "actual-owner" {
					err = fmt.Errorf("returned rejected owner %s", owner)
				}
				finished <- err
			}()
			select {
			case <-peer.outgoing.ready:
				message := takeMosi(peer.outgoing)
				var request struct{ Return, Method string }
				if err := unmarshalNode(message.Data, &request); err != nil || request.Method != "cluster.location" {
					t.Fatal("missing location request", request, err)
				}
				route, ok := service.handles.Load(controlReturnSlot(request.Return))
				if !ok {
					t.Fatal("location request omitted return handle")
				}
				deliverTestCompletion(t, route, scanCompletion{Data: mustNodeMetadata(replicaRecord{Resource: resource, Owner: "local", Epoch: hintEpoch})})
			case <-ctx.Done():
				t.Fatal("location request did not arrive")
			}
			if err := <-finished; err != nil {
				t.Fatal(err)
			}
			expectedOwner := "actual-owner"
			if fixture.handoff {
				expectedOwner = "local"
			}
			if cell.ownstate.Phase() != mosi.IO || cluster.report(resource).Owner != expectedOwner {
				t.Fatal("stale hint changed native permission or accepted route")
			}
		})
	}
}

func TestClusterReportJoinsUUIDDirectoryPublication(t *testing.T) {
	cluster := &nodeCluster{}
	cluster.initializeManager()
	r := cluster.manager
	old := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	next := old
	next.Shard = uuid.NewString()
	cell := r.resourceCell(old)
	cell.ownstate.Initialize(mosi.S)
	r.updateRecord(old, func(record *replicaRecord) bool { record.Owner, record.Epoch = "owner", 7; return true })
	r.mu.Lock()
	// Match the interval before the complete routing publication, while its
	// existing directory lock is held. Native identity still names the old UUID.
	record := r.directory.Records[old.key()]
	record.Resource = next
	r.directory.Records[next.key()] = record
	delete(r.directory.Records, old.key())
	r.cells[next.key()] = cell
	done := make(chan replicaRecord, 1)
	go func() { done <- cluster.report(next) }()
	var early *replicaRecord
	select {
	case report := <-done:
		early = &report
	case <-time.After(20 * time.Millisecond):
	}
	r.publishCellsLocked(cell, &next)
	r.mu.Unlock()
	if early != nil {
		t.Fatal("report bypassed unfinished UUID publication", *early)
	}
	select {
	case report := <-done:
		if report.Owner != "owner" || report.State != "S" || report.Resource != next {
			t.Fatal("published owner disappeared", report)
		}
	case <-time.After(time.Second):
		t.Fatal("report did not join UUID publication")
	}
}

func TestClusterShardRestartRecoveryUsesLiveReports(t *testing.T) {
	for _, learnedFromSchema := range []bool{false, true} {
		t.Run(fmt.Sprint("schema-hint=", learnedFromSchema), func(t *testing.T) {
			service := &nodeScanService{}
			remote := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
			cluster := &nodeCluster{self: clusterMember{ID: "local"}, service: service}
			cluster.initializeManager()
			cluster.view.Store(&clusterView{members: []clusterMember{cluster.self, {ID: "reader"}}, links: []*clusterLink{{member: clusterMember{ID: "reader"}, peer: remote}}})
			resource := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
			cell := cluster.manager.resourceCell(resource)
			if learnedFromSchema {
				schema := mosiResource{Database: "tenant"}
				cluster.manager.resourceCell(schema).ownstate.Initialize(mosi.S)
				cluster.manager.updateRecord(schema, func(record *replicaRecord) bool { record.Owner, record.Epoch = "reader", 1; return true })
			} else {
				cluster.manager.updateRecord(resource, func(record *replicaRecord) bool { record.Owner = "local"; record.Epoch = 3; return true })
			}
			finished := make(chan error, 1)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			go func() {
				owner, err := cluster.locate(ctx, resource)
				if err == nil && owner != "local" {
					err = fmt.Errorf("wrong recovered owner %s", owner)
				}
				finished <- err
			}()
			methods := []string{"cluster.report"}
			if learnedFromSchema {
				methods = append([]string{"cluster.location"}, methods...)
			}
			for _, method := range methods {
				message := takeMosi(remote.outgoing)
				var request struct{ Return, Method string }
				if err := decodeTestInvocation(message.Channel, message.Payload, &request); err != nil || request.Method != method {
					t.Fatal("restart omitted native recovery", request, err)
				}
				if cell.ownstate.Initialized() {
					t.Fatal("schema hint granted authority before live recovery")
				}
				route, _ := service.handles.Load(controlReturnSlot(request.Return))
				deliverTestCompletion(t, route, scanCompletion{Data: mustNodeMetadata(replicaRecord{Resource: resource, Owner: "local", State: "S", Epoch: 3})})
			}
			if err := <-finished; err != nil {
				t.Fatal(err)
			}
			if cell.ownstate.Phase() != mosi.O || !cluster.manager.directory.Records[resource.key()].Readers["reader"] {
				t.Fatal("live read proof did not restore its owner directory")
			}
		})
	}
}

func TestClusterCacheTryReadDeclinesActiveNativeTransaction(t *testing.T) {
	for _, mode := range []TxMode{TxCursorStability, TxACID} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			tbl, _ := createDurabilityTestTable(t, "t_cache_try_read", 3)
			self := clusterMember{ID: "memcp://" + uuid.NewString(), Boot: uuid.NewString()}
			reader := clusterMember{ID: "memcp://" + uuid.NewString(), Boot: uuid.NewString()}
			cluster := &nodeCluster{self: self}
			cluster.view.Store(&clusterView{members: []clusterMember{self, reader}})
			cluster.initializeManager()
			r := cluster.manager
			shard := tbl.ActiveShards()[0]
			cell := r.declareShard(shard, self.ID, true)
			tx := NewTxContext(mode)
			tx.mu.Lock()
			tx.getOrCreateShardTxLocked(shard)
			tx.mu.Unlock()
			defer tx.Rollback()
			packet := mosiPacket{Kind: "read", Resource: *cell.resource.Load(),
				ID: uuid.NewString(), Sequence: 1, Boot: reader.Boot, TryRead: true}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			if err := r.coordinate(ctx, reader.ID, packet); err != errClusterReadBusy {
				t.Fatalf("opportunistic read waited for the transaction or changed authority: %v", err)
			}
			if cell.ownstate.Phase() != mosi.M || cell.ownstate.ClosingWrites() || cell.writesDrained() {
				t.Fatal("declined cache admission changed native transaction ownership")
			}
			tx.Rollback()
			packet.Sequence++
			if err := r.coordinate(ctx, reader.ID, packet); err != errClusterReadBusy {
				t.Fatalf("cache admission compacted a modified payload instead of delegating its task: %v", err)
			}
			if cell.ownstate.Phase() != mosi.M || cell.ownstate.ClosingWrites() || !cell.writesDrained() {
				t.Fatal("declined preparation changed the completed transaction's authority")
			}
			releaseCoordination := cell.coordinate(context.Background())
			err := r.coordinate(ctx, reader.ID, packet)
			releaseCoordination()
			if err != errClusterReadBusy {
				t.Fatalf("opportunistic read waited for directory ownership: %v", err)
			}
		})
	}
}

func TestClusterCacheTryReadLeavesExistingAcquisitionAlone(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}, ctx: context.Background()}
	cluster.initializeManager()
	owner := "memcp://" + uuid.NewString()
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: owner}, peer: peer}}})
	resource := mosiResource{Database: "cache_try_existing"}
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Epoch = owner, 1
		return true
	})
	cell := cluster.manager.resourceCell(resource)
	request, _ := cell.ownstate.Request(mosi.S)
	if cluster.tryRead(context.Background(), resource) {
		t.Fatal("a pending acquisition certified cache reuse")
	}
	if cell.ownstate.Phase() != mosi.SR || request.Flight().Err() != nil || peer.outgoing.len() != 0 {
		t.Fatal("cache admission cancelled or duplicated the existing acquisition")
	}
}

func TestClusterCacheTryReadUsesGrantedBackendWithoutOwnerLookup(t *testing.T) {
	cluster := &nodeCluster{}
	cluster.initializeManager()
	resource := mosiResource{Database: "cache_backend_read", Table: "items", Shard: uuid.NewString()}
	cell := cluster.manager.resourceCell(resource)
	cell.ownstate.Initialize(mosi.S)
	cell.ownstate.BackendOnly()
	// There is no network, roster or contact peer. Eviction retained a valid
	// S grant; authorizing another read must consume that existing proof.
	if !cluster.tryRead(context.Background(), resource) || cell.ownstate.Phase() != mosi.S {
		t.Fatal("an existing backend read permission required owner discovery")
	}
}

func TestClusterRemoteMutationRedirectsBeforeRecallingParticipant(t *testing.T) {
	for _, handoff := range []bool{false, true} {
		t.Run(fmt.Sprint(handoff), func(t *testing.T) {
			tbl, _ := createDurabilityTestTable(t, "t_remote_mutation_redirect", 4)
			shard := tbl.ActiveShards()[0]
			cluster := &nodeCluster{self: clusterMember{ID: "former-owner"}, ctx: context.Background(), databases: map[string]bool{tbl.schema.Name: true}}
			cluster.initializeManager()
			cluster.view.Store(&clusterView{})
			schema := cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name})
			schema.ownstate.Initialize(mosi.M)
			schema.clusterstate.Initialize(mosi.M)
			cell := cluster.declareShard(shard, false)
			cell.ownstate.Initialize(mosi.IO)
			cell.clusterstate.Initialize(mosi.IO)
			resource := *cell.resource.Load()
			cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
				record.Owner, record.Epoch = "transaction-participant", 4
				if handoff {
					record.Owner, record.Handoff = cluster.self.ID, "transaction-participant"
				}
				return true
			})
			oldMounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
			defer sharedClusterMounts.Store(oldMounts)
			oldService := nodeScans.Swap(&nodeScanService{cluster: cluster})
			defer nodeScans.Store(oldService)
			defer func() {
				if failure := recover(); failure != nil {
					t.Fatalf("stale task tried to reacquire participant rights: %v", failure)
				}
			}()
			executed := false
			err := withRemoteScanShard(&remoteScanShard{Schema: tbl.filterSchemaFingerprint(), Resource: resource}, nil, true, false, "coordinator", "",
				func(*storageShard) error { executed = true; return nil })
			redirect, ok := err.(clusterScanRedirect)
			if !ok || redirect.Owner != "transaction-participant" || redirect.Epoch != 4 || redirect.Generation != resource.Shard || executed || cell.ownstate.Phase() != mosi.IO {
				t.Fatalf("stale task recalled participant rights or executed callbacks: %v, executed=%v", err, executed)
			}
		})
	}
}

// Discovery is a sequence of live samples, not one distributed snapshot. A
// handover between replies may legitimately expose both owners at successive
// epochs; only two distinct owners at the same epoch conflict.
func TestClusterDiscoveryFollowsOwnerEpochAcrossReplies(t *testing.T) {
	for _, epochs := range [][2]uint64{{7, 8}, {8, 7}, {8, 8}} {
		t.Run(fmt.Sprint(epochs), func(t *testing.T) {
			cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}, service: &nodeScanService{}}
			cluster.initializeManager()
			resource := mosiResource{Database: "discovery_epoch"}
			cell := cluster.manager.resourceCell(resource)
			cell.ownstate.Initialize(mosi.IO)
			members := []clusterMember{cluster.self}
			var links []*clusterLink
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			for _, epoch := range epochs {
				member := clusterMember{ID: "memcp://" + uuid.NewString()}
				peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
				members = append(members, member)
				links = append(links, &clusterLink{member: member, peer: peer})
				go func(epoch uint64) {
					select {
					case <-peer.outgoing.ready:
						message := takeMosi(peer.outgoing)
						var request struct {
							Return string `json:"return"`
						}
						if err := decodeTestInvocation(message.Channel, message.Payload, &request); err != nil {
							return
						}
						answer := replicaRecord{Resource: resource, State: "M", Owner: member.ID, Epoch: epoch}
						if value, ok := cluster.service.handles.Load(controlReturnSlot(request.Return)); ok {
							deliverTestCompletion(t, value, scanCompletion{Data: mustNodeMetadata(answer)})
						}
					case <-ctx.Done():
					}
				}(epoch)
			}
			cluster.view.Store(&clusterView{members: members, links: links})
			owner, err := cluster.locate(ctx, resource)
			if epochs[0] == epochs[1] {
				if err == nil {
					t.Fatal("accepted conflicting owners at one epoch")
				}
			} else {
				want := members[1].ID
				if epochs[1] > epochs[0] {
					want = members[2].ID
				}
				if err != nil || owner != want {
					t.Fatalf("owner=%s error=%v, want %s", owner, err, want)
				}
			}
			if cell.ownstate.Phase() != mosi.IO {
				t.Fatal("discovery granted local rights")
			}
		})
	}
}

func TestClusterAcquisitionRefreshesOwnerAfterUUIDForward(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}, ctx: context.Background()}
	cluster.initializeManager()
	oldOwner, newOwner := "memcp://"+uuid.NewString(), "memcp://"+uuid.NewString()
	oldPeer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	newPeer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: oldOwner}, peer: oldPeer}, {member: clusterMember{ID: newOwner}, peer: newPeer}}})
	r := cluster.manager
	from := mosiResource{Database: "owner_refresh", Table: "items", Shard: uuid.NewString()}
	to := from
	to.Shard = uuid.NewString()
	old := r.resourceCell(from)
	old.ownstate.Initialize(mosi.IO)
	r.updateRecord(from, func(record *replicaRecord) bool { record.Owner, record.Epoch = oldOwner, 1; return true })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan any, 1)
	go func() {
		defer func() {
			if failure := recover(); failure != nil {
				done <- failure
			}
		}()
		done <- cluster.obtain(ctx, from, mosi.M)
	}()
	select {
	case <-oldPeer.outgoing.ready:
		takeMosi(oldPeer.outgoing)
	case <-ctx.Done():
		t.Fatal("initial acquisition missing")
	}
	target := r.resourceCell(to)
	target.ownstate.Initialize(mosi.IO)
	r.updateRecord(to, func(record *replicaRecord) bool { record.Owner, record.Epoch = newOwner, 2; return true })
	// A newer inbound identity supersedes the old placeholder and wakes its waiters.
	r.forwardResource(from, to)
	select {
	case <-oldPeer.outgoing.ready:
		message := takeMosi(oldPeer.outgoing)
		t.Fatalf("successor request reused stale owner: %+v", message.Packet)
	case <-newPeer.outgoing.ready:
		message := takeMosi(newPeer.outgoing)
		if message.Packet == nil || message.Packet.Resource != to {
			t.Fatal("lost successor identity")
		}
		if _, err := target.ownstate.Grant(message.Packet.ID, mosi.M, func() error { return nil }); err != nil {
			t.Fatal(err)
		}
	case result := <-done:
		t.Fatalf("acquisition ended without requesting successor: %v", result)
	case <-ctx.Done():
		t.Fatal("successor acquisition missing")
	}
	select {
	case result := <-done:
		if result != target {
			t.Fatalf("acquisition lost forwarded grant: %v", result)
		}
	case <-ctx.Done():
		t.Fatal("forwarded acquisition did not finish")
	}
}

func TestClusterAcquisitionQueuesBehindOutgoingHandover(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}, ctx: context.Background()}
	cluster.initializeManager()
	owner := "memcp://" + uuid.NewString()
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: owner}, peer: peer}}})
	r := cluster.manager
	resource := mosiResource{Database: "handover_queue", Table: "items", Shard: uuid.NewString()}
	schema := mosiResource{Database: resource.Database}
	r.resourceCell(schema).ownstate.Initialize(mosi.M)
	r.updateRecord(schema, func(record *replicaRecord) bool { record.Owner, record.Epoch = r.id, 1; return true })
	cell := r.resourceCell(resource)
	cell.ownstate.Initialize(mosi.IO)
	r.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Handoff, record.Epoch = r.id, owner, 1
		return true
	})
	releaseCoordination := cell.coordinate(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan any, 1)
	go func() {
		defer func() {
			if failure := recover(); failure != nil {
				done <- failure
			}
		}()
		done <- cluster.obtain(ctx, resource, mosi.M)
	}()
	// Wait for native acquisition admission without allowing the old handover
	// to complete. A fresh destination cannot route this UUID yet.
	for !cell.ownstate.Phase().Requested() && ctx.Err() == nil {
		time.Sleep(time.Millisecond)
	}
	select {
	case <-peer.outgoing.ready:
		takeMosi(peer.outgoing)
		releaseCoordination()
		t.Fatal("request preceded the destination's reference installation")
	case <-time.After(20 * time.Millisecond):
	}
	r.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Handoff, record.Epoch = owner, "", 2
		return true
	})
	releaseCoordination()
	select {
	case <-peer.outgoing.ready:
		message := takeMosi(peer.outgoing)
		if message.Packet == nil || message.Packet.Resource != resource {
			t.Fatal("handover lost request identity")
		}
		if _, err := cell.ownstate.Grant(message.Packet.ID, mosi.M, func() error { return nil }); err != nil {
			t.Fatal(err)
		}
	case failure := <-done:
		t.Fatalf("acquisition ended before the destination grant: %v", failure)
	case <-ctx.Done():
		t.Fatal("request did not follow the completed handover")
	}
	select {
	case result := <-done:
		if result != cell {
			t.Fatalf("handover lost native grant: %v", result)
		}
	case <-ctx.Done():
		t.Fatal("queued acquisition did not finish")
	}
}

func TestClusterGrantedReadDoesNotRediscoverMissingOwner(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}, ctx: context.Background()}
	cluster.initializeManager()
	owner := "memcp://" + uuid.NewString()
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	cluster.view.Store(&clusterView{members: []clusterMember{cluster.self, {ID: owner}}, links: []*clusterLink{{member: clusterMember{ID: owner}, peer: peer}}})
	resource := mosiResource{Database: "granted_read"}
	r := cluster.manager
	cell := r.resourceCell(resource)
	cell.ownstate.Initialize(mosi.IO)
	r.updateRecord(resource, func(record *replicaRecord) bool { record.Owner, record.Epoch = owner, 1; return true })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan any, 1)
	go func() {
		defer func() {
			if failure := recover(); failure != nil {
				done <- failure
			}
		}()
		done <- cluster.obtain(ctx, resource, mosi.S)
	}()
	select {
	case <-peer.outgoing.ready:
		message := takeMosi(peer.outgoing)
		// Losing a contact hint does not revoke an already issued S grant.
		cluster.view.Store(&clusterView{members: []clusterMember{cluster.self}})
		if _, err := cell.ownstate.Grant(message.Packet.ID, mosi.S, func() error { return nil }); err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("read acquisition missing")
	}
	select {
	case result := <-done:
		if result != cell {
			t.Fatalf("received S grant required fresh owner discovery: %v", result)
		}
	case <-ctx.Done():
		t.Fatal("granted read did not finish")
	}
}

func TestClusterColdPlacementExcludesRefusedOwner(t *testing.T) {
	links := []*clusterLink{
		{member: clusterMember{ID: "refused"}},
		{member: clusterMember{ID: "available-a"}},
		{member: clusterMember{ID: "available-b"}},
	}
	cluster := &nodeCluster{}
	cluster.view.Store(&clusterView{links: links})
	executor := clusterScanExecutor{cluster: cluster, preferred: links[0], exclude: "refused"}
	candidates := executor.candidates()
	if len(candidates) != 2 {
		t.Fatalf("capacity retry retained %d peers, want 2", len(candidates))
	}
	for _, candidate := range candidates {
		if candidate == links[0] {
			t.Fatal("capacity retry kept the owner which refused admission")
		}
	}
}

func TestClusterShardRecoveryRestoresCurrentContentFromOlderSchema(t *testing.T) {
	for _, variant := range []string{"older-resource-uuid", "already-canonical-resource-uuid", "unbound-canonical-resource-uuid"} {
		t.Run(variant, func(t *testing.T) {
			testClusterShardRecoveryContent(t, variant)
		})
	}
}

func testClusterShardRecoveryContent(t *testing.T, variant string) {
	tbl, _ := createDurabilityTestTable(t, "t_cluster_recover_current_content", 4)
	RebuildTable(tbl, true, false)
	checkpoint, err := json.Marshal(tbl)
	if err != nil {
		t.Fatal(err)
	}
	oldID := tbl.ActiveShards()[0].uuid
	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(99), scm.NewString("new content")}}, nil, scm.NewNil(), false, nil)
	RebuildTable(tbl, true, false)
	content := tbl.ActiveShards()[0].uuid.String()
	if content == oldID.String() {
		t.Fatal("fixture did not publish a successor")
	}
	// Restart from an older schema checkpoint. Data-resource coherence already
	// knows the newer durable content; the schema holder need not own that data.
	var restored table
	if err := json.Unmarshal(checkpoint, &restored); err != nil {
		t.Fatal(err)
	}
	restored.schema = tbl.schema
	for _, shard := range restored.Shards {
		shard.t = &restored
	}
	restored.publishTopologyLocked()
	tbl.schema.tables.Set(tbl.Name, &restored)
	self, reader := "memcp://"+uuid.NewString(), "memcp://"+uuid.NewString()
	service := &nodeScanService{}
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	cluster := &nodeCluster{self: clusterMember{ID: self}, service: service, ctx: context.Background()}
	cluster.initializeManager()
	service.cluster = cluster
	cluster.view.Store(&clusterView{members: []clusterMember{cluster.self, {ID: reader}}, links: []*clusterLink{{member: clusterMember{ID: reader}, peer: peer}}})
	resource := mosiResource{Database: tbl.schema.Name, Table: tbl.Name, Shard: oldID.String()}
	cell := cluster.manager.shardCell(restored.ActiveShards()[0])
	if variant != "older-resource-uuid" {
		successor := resource
		successor.Shard = content
		cluster.manager.forwardResource(resource, successor)
		resource = successor
		if variant == "unbound-canonical-resource-uuid" {
			cell.current.Store(nil)
		}
	}
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Epoch, record.Content = self, 3, oldID.String()
		return true
	})
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	oldService := nodeScans.Swap(service)
	defer nodeScans.Store(oldService)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		owner, err := cluster.locate(ctx, resource)
		if err == nil && owner != self {
			err = fmt.Errorf("wrong recovery owner %s", owner)
		}
		finished <- err
	}()
	select {
	case <-peer.outgoing.ready:
		message := takeMosi(peer.outgoing)
		var request struct{ Return, Method string }
		if err := unmarshalNode(message.Data, &request); err != nil || request.Method != "cluster.report" || cell.ownstate.Initialized() {
			t.Fatal("recovery omitted live authority proof", request, err)
		}
		route, ok := service.handles.Load(controlReturnSlot(request.Return))
		if !ok {
			t.Fatal("recovery omitted its return handle")
		}
		deliverTestCompletion(t, route, scanCompletion{Data: mustNodeMetadata(replicaRecord{Resource: resource, Owner: self, State: "S", Epoch: 3, Content: content})})
	case <-ctx.Done():
		t.Fatal("recovery did not request live proof")
	}
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	current := restored.ActiveShards()[0]
	if current.uuid.String() != content || cluster.manager.resourceCell(resource).publishedShard() != current {
		t.Fatal("recovered authority still addresses the older checkpoint")
	}
	if current.state() != COLD {
		t.Fatal("authority recovery eagerly loaded the shard payload")
	}
	if restored.CountExact() != 5 {
		t.Fatal("recovery lost the row absent from the older checkpoint")
	}
}

func TestClusterTopologyPublishesEveryRebuildSuccessor(t *testing.T) {
	for _, imported := range []bool{false, true} {
		name := "rebuild-chain"
		if imported {
			name = "import-with-lagging-payload-hint"
		}
		t.Run(name, func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "t_cluster_rebuild_chain_publication")
			source := tbl.ActiveShards()[0]
			// Successors are private generations, not independently declared M owners.
			middle, last := NewShard(tbl), NewShard(tbl)
			cluster := globalTestCluster(t, tbl.schema.Name)
			mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
			defer sharedClusterMounts.Store(mounts)
			service := nodeScans.Swap(&nodeScanService{cluster: cluster})
			defer nodeScans.Store(service)
			cell := cluster.declareShard(source, true)
			middle.coherence.Store(cell)
			last.coherence.Store(cell)
			if imported {
				if err := cluster.manager.replaceShard(cluster.manager.shardResource(source), source, middle); err != nil {
					t.Fatal(err)
				}
				// Import publishes topology before its caller advances this advisory
				// hint. A following rebuild must start at the published generation.
				cell.current.Store(source)
			} else {
				source.next.Store(middle)
				source.nextReady.Store(true)
			}
			middle.next.Store(last)
			middle.nextReady.Store(true)
			tbl.mu.Lock()
			tbl.Shards = []*storageShard{last}
			tbl.publishTopologyLocked()
			tbl.mu.Unlock()
			for _, shard := range []*storageShard{source, middle, last} {
				resource := cluster.manager.shardResource(shard)
				if found := cluster.manager.lookupCell(resource); found != cell {
					t.Errorf("published native UUID %s has no shared authority: %p, want %p", shard.uuid, found, cell)
				}
			}
			if cell.current.Load() != last || cell.resource.Load().Shard != last.uuid.String() {
				t.Fatal("topology publication did not advance its native authority")
			}
		})
	}
}

func TestClusterDelegationDistinguishesHandoverFromRoutingCycle(t *testing.T) {
	for _, progress := range []string{"unchanged", "epoch", "generation"} {
		t.Run(progress, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			service := &nodeScanService{}
			cluster := &nodeCluster{service: service, ctx: ctx}
			var links []*clusterLink
			for _, id := range []string{"owner-a", "owner-b"} {
				peer := &mosiPeer{ctx: ctx, done: make(chan struct{}), outgoing: newMosiMailbox(), scanService: service, objectPeer: clusterObjectPeer{ID: id, Boot: "boot"}}
				links = append(links, &clusterLink{member: clusterMember{ID: id}, peer: peer})
			}
			cluster.view.Store(&clusterView{links: links})
			expected := remoteScanShard{Schema: 1, Resource: mosiResource{Database: "delegation", Table: "items", Shard: uuid.NewString()}}
			executor := &clusterScanExecutor{cluster: cluster, expected: expected, preferred: links[0]}
			condition := scm.Eval(scm.Read(t.Name(), "(lambda () true)", true), &scm.Globalenv)
			finished := make(chan scanResult, 1)
			go func() {
				var result scanResult
				defer func() {
					if failure := recover(); failure != nil {
						result.err = scanError{r: failure}
					}
					finished <- result
				}()
				result = executor.ExecuteShardScan(expected.Resource, shardScanInvocation{operator: shardExists, condition: condition, neutral: scm.NewBool(false)})
			}()
			var result scanResult
			completed := false
			for step := 0; step < 3 && !completed; step++ {
				peer := links[step%2].peer
				select {
				case <-peer.outgoing.ready:
					message := takeMosi(peer.outgoing)
					var request struct{ Return, Method string }
					if err := unmarshalNode(message.Data, &request); err != nil || request.Method != "scan.execute" {
						t.Fatal("delegation did not use its native scan contract", request, err)
					}
					response := remoteScanResult{Redirect: &clusterScanRedirect{Owner: links[(step+1)%2].member.ID, Epoch: 4, Generation: expected.Resource.Shard}}
					if progress == "epoch" && step == 1 {
						response.Redirect.Epoch = 5
					}
					if progress == "generation" && step == 1 {
						response.Redirect.Generation = uuid.NewString()
					}
					if progress != "unchanged" && step == 2 {
						response = remoteScanResult{Version: scanWireVersion, Shard: expected, Value: []byte("true"), OutCount: 1}
					}
					entry, _ := service.handles.Load(controlReturnSlot(request.Return))
					deliverTestCompletion(t, entry, scanCompletion{Data: mustNodeMetadata(response)})
				case result = <-finished:
					completed = true
				case <-ctx.Done():
					t.Fatal("delegation did not finish", ctx.Err())
				}
			}
			if !completed {
				select {
				case result = <-finished:
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			if progress != "unchanged" {
				if result.err.r != nil || !result.res.Bool() {
					t.Fatal("live ownership progress was rejected as a cycle", result.err.r)
				}
			} else if result.err.r == nil || !strings.Contains(fmt.Sprint(result.err.r), "ownership route cycled") {
				t.Fatal("unchanged cyclic hints were allowed to loop", result.err.r)
			}
		})
	}
}

func TestClusterSchemaPublicationKeepsRebuildCoordination(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_cluster_schema_rebuild_authority")
	source := tbl.ActiveShards()[0]
	successor := NewShard(tbl)
	cluster := globalTestCluster(t, tbl.schema.Name)
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	cell := cluster.declareShard(source, true)
	coordination := cell.clusterstate.TryCoordinate()
	if coordination == nil {
		t.Fatal("could not start the source handover")
	}
	defer coordination.Release()
	// A routing hint may already have created an invalid placeholder. It must
	// share the source's active coordination before a schema can expose its UUID.
	next := cluster.manager.shardResource(successor)
	cluster.manager.resourceCell(next).ownstate.Initialize(mosi.IO)
	cluster.manager.updateRecord(next, func(record *replicaRecord) bool {
		record.Owner, record.Epoch = cluster.self.ID, 1
		return true
	})
	successor.coherence.Store(cell)
	source.next.Store(successor)
	source.nextReady.Store(true)
	tbl.mu.Lock()
	tbl.Shards = []*storageShard{successor}
	tbl.publishSchemaTopology(ShardModeFree, tbl.Shards, tbl.PDimensions)
	tbl.mu.Unlock()
	encoded, err := json.Marshal(tbl)
	if err != nil {
		t.Fatal(err)
	}
	var persisted struct{ Shards []string }
	if err := json.Unmarshal(encoded, &persisted); err != nil || len(persisted.Shards) != 1 || persisted.Shards[0] != successor.uuid.String() {
		t.Fatal("schema fixture did not expose its private successor", err)
	}
	published := cluster.manager.resourceCell(next).resolved()
	if published != cell {
		t.Error("schema exposed a rebuild UUID with an independent coherence cell")
	}
	if concurrent := published.clusterstate.TryCoordinate(); concurrent != nil {
		concurrent.Release()
		t.Error("schema allowed a second handover while the source handover was active")
	}
	if cell.current.Load() != source || tbl.ActiveShards()[0] != source {
		t.Fatal("schema publication replaced the still-active native input")
	}
	tbl.mu.Lock()
	tbl.publishTopologyLocked()
	tbl.mu.Unlock()
	if cell.current.Load() != successor {
		t.Fatal("native publication did not advance its payload hint")
	}
}

func TestClusterRemoteMutationJoinsPendingOwnerGrant(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_remote_pending_owner", 4)
	shard := tbl.ActiveShards()[0]
	cluster := globalTestCluster(t, tbl.schema.Name)
	owner := clusterMember{ID: "old-owner", Boot: uuid.NewString()}
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	cluster.view.Store(&clusterView{members: []clusterMember{cluster.self, owner}, links: []*clusterLink{{member: owner, peer: peer}}})
	schema := cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name})
	schema.ownstate.Initialize(mosi.M)
	cell := cluster.declareShard(shard, false)
	cell.ownstate.Initialize(mosi.IO)
	cell.clusterstate.Initialize(mosi.M)
	resource := *cell.resource.Load()
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Epoch = owner.ID, 4
		return true
	})
	pending, _ := cell.ownstate.Request(mosi.M)
	oldMounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(oldMounts)
	oldService := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(oldService)
	// A group reservation cannot join this flight while its coordinator may
	// retain another group needed by that same incoming handover.
	err := withRemoteScanShard(&remoteScanShard{Schema: tbl.filterSchemaFingerprint(), Resource: resource}, nil, false, true, "coordinator", "", func(*storageShard) error {
		t.Fatal("ungranted owner-group reservation executed a callback")
		return nil
	})
	if busy, ok := err.(uniqueReservationBusy); !ok || busy.shard.Resource != resource || cell.ownstate.Snapshot().Flight() != pending.Flight() {
		t.Fatal("reservation waited for or replaced the native pending grant", err)
	}
	done := make(chan error, 1)
	executed := make(chan struct{}, 1)
	go func() {
		defer func() {
			if failure := recover(); failure != nil {
				done <- fmt.Errorf("pending acquisition failed: %v", failure)
			}
		}()
		done <- withRemoteScanShard(&remoteScanShard{Schema: tbl.filterSchemaFingerprint(), Resource: resource}, nil, true, false, "coordinator", "", func(*storageShard) error { executed <- struct{}{}; return nil })
	}()
	select {
	case err := <-done:
		t.Fatalf("pending owner bounced a task back to its grant source: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	if _, err := cell.ownstate.Grant(pending.Flight().ID(), mosi.M, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("task did not join its existing native grant")
	}
	select {
	case <-executed:
	default:
		t.Fatal("granted task did not execute")
	}
}

func TestClusterQueuedReadFollowsNativeHandoverBeforeGranting(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_queued_read_handover", 3)
	cluster := globalTestCluster(t, tbl.schema.Name)
	r := cluster.manager
	cell := cluster.declareShard(tbl.ActiveShards()[0], false)
	cell.ownstate.Initialize(mosi.IO)
	cell.clusterstate.Initialize(mosi.IO)
	resource := *cell.resource.Load()
	target := clusterMember{ID: "memcp://" + uuid.NewString(), Boot: uuid.NewString()}
	peer := &mosiPeer{ctx: context.Background(), done: make(chan struct{}), outgoing: newMosiMailbox()}
	cluster.view.Store(&clusterView{members: []clusterMember{cluster.self, target}, links: []*clusterLink{{member: target, peer: peer}}})
	// The source already revoked M; its recipient can write before the source
	// ACK worker resumes. A queued local read must follow that recipient.
	r.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Handoff, record.Epoch, record.State = r.id, target.ID, 3, "M"
		return true
	})
	pending, _ := cell.ownstate.Request(mosi.S)
	packet := mosiPacket{Kind: "read", Resource: resource, ID: pending.Flight().ID(), Sequence: r.requestSequence(pending.Flight()), Boot: cluster.self.Boot}
	if err := r.coordinate(context.Background(), r.id, packet); err != nil {
		t.Fatal(err)
	}
	if cell.ownstate.ReadReady() || cell.clusterstate.Phase().Readable() {
		t.Fatal("former owner granted itself O while its handover recipient holds M")
	}
	select {
	case <-peer.outgoing.ready:
		message := takeMosi(peer.outgoing)
		if message.Packet == nil || message.Packet.ID != packet.ID || message.Packet.To != target.ID || message.Packet.From != r.id {
			t.Fatal("queued acquisition lost its native request identity")
		}
	default:
		t.Fatal("queued acquisition was not forwarded to the handover recipient")
	}
	r.mu.Lock()
	record := r.directory.Records[r.resourceKey(resource)]
	r.mu.Unlock()
	grant := mosiPacket{Kind: "reference", Resource: resource, ID: packet.ID, Epoch: 3, State: "S", Owner: target.ID}
	if _, err := cluster.referenceSource(target.ID, grant, cell, record); err != nil {
		t.Fatalf("named handover recipient could not answer the queued flight: %v", err)
	}
	if _, err := cluster.referenceSource("foreign-peer", grant, cell, record); err == nil {
		t.Fatal("unrelated peer answered a same-epoch queued flight")
	}
}

// Two native UUIDs can both have sent a request before a rebuild route arrives.
// Folding their cells must not turn the loser's legitimate reply into a forged
// request, nor may a weaker reply satisfy an outstanding write request.
func TestClusterIssuedAcquisitionSurvivesUUIDFolding(t *testing.T) {
	for _, pending := range []mosi.Phase{mosi.IO, mosi.S, mosi.M} {
		t.Run(pending.String(), func(t *testing.T) {
			cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}}
			cluster.initializeManager()
			r := cluster.manager
			from := mosiResource{Database: "folded_flight", Table: "items", Shard: uuid.NewString()}
			to := from
			to.Shard = uuid.NewString()
			owner := "memcp://" + uuid.NewString()
			original, successor := r.resourceCell(from), r.resourceCell(to)
			original.ownstate.Initialize(mosi.IO)
			successor.ownstate.Initialize(mosi.IO)
			r.updateRecord(from, func(record *replicaRecord) bool { record.Owner, record.Epoch = owner, 3; return true })
			r.updateRecord(to, func(record *replicaRecord) bool { record.Owner, record.Epoch = owner, 4; return true })
			read, _ := original.ownstate.Request(mosi.S)
			request := mosiPacket{Resource: from, Kind: "read", ID: read.Flight().ID()}
			r.issueAcquisition(original, request)
			original.ownstate.Request(mosi.M)
			request.Kind = "write"
			r.issueAcquisition(original, request)
			var waiting *mosi.Publication
			if pending != mosi.IO {
				waiting, _ = successor.ownstate.Request(pending)
				kind := "read"
				if pending == mosi.M {
					kind = "write"
				}
				r.issueAcquisition(successor, mosiPacket{Kind: kind, ID: waiting.Flight().ID(), Resource: to})
			}
			r.forwardResource(from, to)
			if original.resolved() != successor {
				t.Fatal("fixture did not replace the issuing cell")
			}
			record := cluster.report(to)
			answer := mosiPacket{Resource: from, ID: request.ID, State: "S", Owner: owner, Epoch: 5}
			if _, err := cluster.referenceSource(owner, answer, successor, record); err != nil {
				t.Fatalf("known read reply lost across UUID folding: %v", err)
			}
			for _, mutation := range []string{"id", "resource", "sender"} {
				bad, source := answer, owner
				switch mutation {
				case "id":
					bad.ID = uuid.NewString()
				case "resource":
					bad.Resource.Shard = uuid.NewString()
				case "sender":
					source, bad.Epoch = "foreign", record.Epoch
				}
				cell := successor
				if mutation == "resource" {
					cell = r.resourceCell(bad.Resource)
				}
				if _, err := cluster.referenceSource(source, bad, cell, record); err == nil {
					t.Fatalf("invalid %s accepted an issued request ID", mutation)
				}
			}
			if err := r.grantAcquired(successor, answer); err != nil {
				t.Fatal(err)
			}
			r.finishAcquisition(answer)
			if pending == mosi.M && successor.ownstate.Phase() != mosi.MR {
				t.Fatal("weaker predecessor reply completed canonical write acquisition")
			}
			answer.State, answer.Owner, answer.Epoch = "M", r.id, 6
			if !r.issuedAcquisition(successor, answer) {
				t.Fatal("read completion discarded the pending write reply")
			}
			if err := r.grantAcquired(successor, answer); err != nil {
				t.Fatal(err)
			}
			r.finishAcquisition(answer)
			if successor.ownstate.Phase() != mosi.M || successor.clusterstate.Phase() != mosi.M {
				t.Fatal("valid predecessor write reply did not grant canonical authority")
			}
			if waiting != nil {
				select {
				case <-waiting.Flight().Done():
				default:
					t.Fatal("canonical waiters did not wake")
				}
			}
			if waiting != nil {
				confirmation := mosiPacket{Resource: to, ID: waiting.Flight().ID(), State: "O", Owner: r.id, Epoch: 7}
				if pending == mosi.M {
					confirmation.State = "M"
				}
				if err := r.acceptGrant(confirmation); err != nil {
					t.Fatal(err)
				}
				if successor.ownstate.Phase() != mosi.M {
					t.Fatal("late read confirmation downgraded already granted M")
				}
				if pending == mosi.M && cluster.report(to).Epoch != 7 {
					t.Fatal("already granted M lost the later request's authority epoch")
				}
			}
			if len(r.acquisitions) != 0 {
				t.Fatal("terminal replies retained request capabilities")
			}
		})
	}
}

func TestClusterSupersededAcquisitionCompletesProtocolCapability(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_superseded_flight", 3)
	owner := globalTestCluster(t, tbl.schema.Name)
	r := owner.manager
	cell := owner.declareShard(tbl.ActiveShards()[0], true)
	resource := *cell.resource.Load()
	requester := clusterMember{ID: "memcp://" + uuid.NewString(), Boot: uuid.NewString()}
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	owner.view.Store(&clusterView{members: []clusterMember{owner.self, requester}, links: []*clusterLink{{member: requester, peer: peer}}})
	origin := &nodeCluster{self: requester}
	origin.initializeManager()
	origin.view.Store(&clusterView{links: []*clusterLink{{member: owner.self, peer: peer}}})
	original := origin.manager.resourceCell(resource)
	original.ownstate.Initialize(mosi.IO)
	old, _ := original.ownstate.Request(mosi.M)
	packet := mosiPacket{Kind: "write", Resource: resource, ID: old.Flight().ID(), Sequence: 8, Boot: requester.Boot}
	origin.manager.issueAcquisition(original, packet)
	original.ownstate.Fail(packet.ID, errClusterAcquisitionSuperseded, mosi.M)
	current, _ := original.ownstate.Request(mosi.M)
	newRequest := packet
	newRequest.ID, newRequest.Sequence = current.Flight().ID(), 9
	origin.manager.issueAcquisition(original, newRequest)
	r.updateRecord(resource, func(record *replicaRecord) bool {
		record.Requests = map[string]replicaRequestMark{requester.ID: {Boot: requester.Boot, ID: newRequest.ID, Sequence: 9, Write: true}}
		return true
	})
	if err := r.coordinate(context.Background(), requester.ID, packet); err != nil {
		t.Fatal(err)
	}
	select {
	case <-peer.outgoing.ready:
		message := takeMosi(peer.outgoing)
		if message.Packet == nil || message.Packet.ID != packet.ID || message.Packet.Kind != "failure" || message.Packet.Error != errClusterAcquisitionSuperseded.Error() {
			t.Fatalf("superseded request did not receive its own terminal reply: %#v", message.Packet)
		}
		if err := origin.manager.applyResourcePacket(owner.self.ID, message.Packet); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatal("superseded request was silently dropped")
	}
	if _, exists := origin.manager.acquisitions[packet.ID]; exists {
		t.Fatal("terminal failure retained the old protocol capability")
	}
	if !origin.manager.issuedAcquisition(original, mosiPacket{ID: newRequest.ID, State: "M"}) {
		t.Fatal("old failure discarded the newer request")
	}
	select {
	case <-current.Flight().Done():
		t.Fatal("old failure canceled the newer native acquisition")
	default:
	}
}

func TestClusterLocationReplyRefreshesRequestedAlias(t *testing.T) {
	for _, scenario := range []struct{ newer, granted bool }{{false, false}, {true, false}, {true, true}} {
		t.Run(fmt.Sprint("newer=", scenario.newer, "/native-grant=", scenario.granted), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			service := &nodeScanService{}
			cluster := &nodeCluster{self: clusterMember{ID: "requester"}, service: service, ctx: ctx}
			cluster.initializeManager()
			var links []*clusterLink
			for _, id := range []string{"schema-owner", "old-owner", "current-owner", "newest-owner"} {
				peer := &mosiPeer{ctx: ctx, done: make(chan struct{}), outgoing: newMosiMailbox(), scanService: service}
				links = append(links, &clusterLink{member: clusterMember{ID: id}, peer: peer})
			}
			cluster.view.Store(&clusterView{links: links})
			r := cluster.manager
			from := mosiResource{Database: "location_reply", Table: "items", Shard: uuid.NewString()}
			to := from
			to.Shard = uuid.NewString()
			r.resourceCell(mosiResource{Database: from.Database}).ownstate.Initialize(mosi.IO)
			r.updateRecord(mosiResource{Database: from.Database}, func(record *replicaRecord) bool { record.Owner = links[0].member.ID; return true })
			original := r.resourceCell(from)
			phase := mosi.IO
			if scenario.granted {
				phase = mosi.S
			}
			original.ownstate.Initialize(phase)
			r.updateRecord(from, func(record *replicaRecord) bool {
				record.Owner, record.Epoch, record.State = links[1].member.ID, 3, phase.String()
				return true
			})
			want, epoch := links[2].member.ID, uint64(4)
			if scenario.newer {
				cluster.installLocationHint(replicaRecord{Resource: to, Owner: links[3].member.ID, Epoch: 7, Content: to.Shard})
				want, epoch = links[3].member.ID, 7
			}
			done := make(chan error, 1)
			go func() {
				owner, err := cluster.locateViaSchema(ctx, from)
				if err == nil && owner != want {
					err = fmt.Errorf("resolved owner %s, want %s", owner, want)
				}
				done <- err
			}()
			select {
			case <-links[0].peer.outgoing.ready:
				message := takeMosi(links[0].peer.outgoing)
				var request struct {
					Return, Method string
					Data           []byte
				}
				if err := unmarshalNode(message.Data, &request); err != nil || request.Method != "cluster.location" {
					t.Fatal("invalid location request", err)
				}
				entry, _ := service.handles.Load(controlReturnSlot(request.Return))
				deliverTestCompletion(t, entry, scanCompletion{Data: mustNodeMetadata(replicaRecord{Resource: to, Owner: links[2].member.ID, Epoch: 4, Content: to.Shard})})
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			lookupWant, content := want, to.Shard
			if scenario.granted {
				lookupWant, epoch, content = links[1].member.ID, 3, ""
			}
			owner, err := cluster.locate(ctx, from)
			if err != nil || owner != lookupWant {
				t.Fatalf("next owner grouping reused stale alias owner: %s, %v; want %s", owner, err, lookupWant)
			}
			if record := cluster.report(from); record.Epoch != epoch || record.Content != content {
				t.Fatal("alias lost the accepted canonical publication", record)
			}
			if original.resolved() != original || r.lookupCell(to) == original || *original.resource.Load() != from || original.ownstate.Phase() != phase || r.lookupCell(to).ownstate.Phase() != mosi.IO {
				t.Fatal("a routing hint merged cells or granted native rights")
			}
		})
	}
}

// Test adapters inspect the parameter layout selected by the actual channel.
// This record is never serialized or used by production code.
func decodeTestInvocation(channel string, payload []byte, destination any) error {
	address, _, body, err := decodeInvocation(channel, payload)
	if err != nil {
		return err
	}
	v := reflect.ValueOf(destination).Elem()
	if field := v.FieldByName("Return"); field.IsValid() {
		field.SetString(address)
	}
	if field := v.FieldByName("Method"); field.IsValid() {
		field.SetString(channel)
	}
	if field := v.FieldByName("Data"); field.IsValid() {
		if field.Type() == reflect.TypeOf([]byte(nil)) {
			field.SetBytes(body)
		} else {
			return unmarshalNode(body, field.Addr().Interface())
		}
	}
	return nil
}
