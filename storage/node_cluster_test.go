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
import "testing"
import "strings"
import "encoding/json"
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
	peer, _, _ := setupRemoteScanConnection(t)
	if err := prepareSharedClusterMounts("mounted_snapshot"); err != nil {
		t.Fatal(err)
	}
	var reply any
	err := peer.call(context.Background(), "scan.describe", remoteScanDescribe{scanWireVersion, "mounted_snapshot", "items"}, &reply)
	if err == nil {
		t.Fatal("unbudgeted describe opened a mounted snapshot")
	}
	task := remoteScanTask{Shard: remoteScanShard{Resource: mosiResource{Database: "mounted_snapshot", Table: "items", Shard: uuid.NewString()}}}
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
	_, err := session.handle(context.Background(), "cluster.report", []byte(`{"database":"unmounted"}`))
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
			available := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 1)}
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
				case message := <-available.outgoing:
					var request struct {
						Return string `json:"return"`
					}
					if json.Unmarshal(message.Data, &request) != nil {
						return
					}
					answer := replicaRecord{Resource: resource, State: "S", Owner: dead.ID, Epoch: 7}
					if scenario == "live-owner" {
						answer.State, answer.Owner = "M", live.ID
					}
					if scenario == "restart-with-missing-peer" {
						answer.Owner = cluster.self.ID
					}
					if value, ok := cluster.service.returns.Load(request.Return); ok {
						deliverTestCompletion(t, value, scanCompletion{Data: mustScanJSON(answer)})
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
			peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 2)}
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
					_, release := r.writeResource(resource, tx)
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
			if request.Flight().Err() != nil || len(peer.outgoing) != 0 {
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
	peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 4)}
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
	case message := <-peer.outgoing:
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
	schemaPeer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 2)}
	unrelated := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 2)}
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
	message := <-schemaPeer.outgoing
	var request struct {
		Return, Method string
		Data           json.RawMessage
	}
	if err := json.Unmarshal(message.Data, &request); err != nil || request.Method != "cluster.location" {
		t.Fatal("per-shard discovery did not use schema owner", request, err)
	}
	route, _ := service.returns.Load(request.Return)
	deliverTestCompletion(t, route, scanCompletion{Data: mustScanJSON(replicaRecord{Resource: resource, Owner: "schema-owner", Epoch: 3})})
	if owner := <-finished; owner != "schema-owner" {
		t.Fatal(owner)
	}
	if len(unrelated.outgoing) != 0 {
		t.Fatal("queried unrelated peer for shard placement")
	}
	owner, err := cluster.locate(ctx, resource)
	if err != nil || owner != "schema-owner" || len(schemaPeer.outgoing) != 0 {
		t.Fatal("known route performed another probe", owner, err)
	}
}

func TestClusterSchemaLocationTransferUsesBoundedBatches(t *testing.T) {
	service := &nodeScanService{}
	peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 1)}
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
			message := <-peer.outgoing
			var request struct {
				Return string
				Data   clusterSchemaLocations
			}
			if err := json.Unmarshal(message.Data, &request); err != nil {
				t.Error(err)
				return
			}
			batches = append(batches, len(request.Data.Entries))
			for _, entry := range request.Data.Entries {
				if len(entry.Readers) != 0 || len(entry.Requests) != 0 {
					t.Error("schema directory copied shard reader/operation ownership")
				}
			}
			route, _ := service.returns.Load(request.Return)
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
	peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 1)}
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
		case message := <-peer.outgoing:
			var request struct {
				Return string
				Data   json.RawMessage
			}
			if err := json.Unmarshal(message.Data, &request); err != nil {
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
			route, _ := service.returns.Load(request.Return)
			deliverTestCompletion(t, route, scanCompletion{Data: mustScanJSON(page)})
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
			remote := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 1)}
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
				message := <-remote.outgoing
				var request struct{ Return, Method string }
				if err := json.Unmarshal(message.Data, &request); err != nil || request.Method != method {
					t.Fatal("restart omitted native recovery", request, err)
				}
				if cell.ownstate.Initialized() {
					t.Fatal("schema hint granted authority before live recovery")
				}
				route, _ := service.returns.Load(request.Return)
				deliverTestCompletion(t, route, scanCompletion{Data: mustScanJSON(replicaRecord{Resource: resource, Owner: "local", State: "S", Epoch: 3})})
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
			cell.directory.Lock()
			err := r.coordinate(ctx, reader.ID, packet)
			cell.directory.Unlock()
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
	peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 1)}
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
	if cell.ownstate.Phase() != mosi.SR || request.Flight().Err() != nil || len(peer.outgoing) != 0 {
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
		return true
	})
	oldMounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(oldMounts)
	oldService := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(oldService)
	executed := false
	err := withRemoteScanShard(&remoteScanShard{Schema: tbl.filterSchemaFingerprint(), Resource: resource}, nil, true, "coordinator", "",
		func(*storageShard) error { executed = true; return nil })
	redirect, ok := err.(clusterScanRedirect)
	if !ok || redirect.owner != "transaction-participant" || executed || cell.ownstate.Phase() != mosi.IO {
		t.Fatalf("stale task recalled participant rights or executed callbacks: %v, executed=%v", err, executed)
	}
}
