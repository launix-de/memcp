/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "sync"
import "time"
import "testing"
import "runtime"
import "context"
import "sync/atomic"
import "runtime/debug"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

func TestClusterForwardKeepsNewerRevocation(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	r := cluster.manager
	from := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	to := from
	to.Shard = uuid.NewString()
	old := r.resourceCell(from)
	old.ownstate.Initialize(mosi.S)
	old.clusterstate.Initialize(mosi.S)
	r.updateRecord(from, func(record *replicaRecord) bool {
		record.State, record.Owner, record.Epoch = "S", "remote", 1
		return true
	})
	revoked := r.resourceCell(to)
	revoked.ownstate.Initialize(mosi.IO)
	revoked.clusterstate.Initialize(mosi.IO)
	r.updateRecord(to, func(record *replicaRecord) bool {
		record.State, record.Owner, record.Epoch = "IO", "remote", 2
		return true
	})
	r.forwardResource(from, to)
	current := r.resourceCell(from)
	if current != revoked || current.ownstate.Phase() != mosi.IO || current.clusterstate.Phase() != mosi.IO || r.directory.Records[r.resourceKey(to)].Epoch != 2 {
		t.Fatal("native UUID forwarding resurrected a revoked historical reader")
	}
}

func TestClusterUniqueProbeUsesAcquiredSuccessor(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tuniqueacquiredsuccessor")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	source := tbl.ActiveShards()[0]
	successor := NewShard(tbl)
	write := successor.GetWrite(nil)
	successor.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(42)}}, false, nil, nil, false, nil)
	write.Release()
	cluster := &nodeCluster{self: clusterMember{ID: "local"}, ctx: context.Background(), databases: map[string]bool{tbl.schema.Name: true}}
	cluster.initializeManager()
	r := cluster.manager
	cell := cluster.declareShard(source, false)
	resource := r.shardResource(source)
	r.updateRecord(resource, func(record *replicaRecord) bool { record.Owner = "remote"; return true })
	peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 4)}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: "remote"}, peer: peer}}})
	request, _ := cell.ownstate.Request(mosi.S)
	oldMounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(oldMounts)
	oldService := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(oldService)
	topology := tbl.activeTopology()
	type outcome struct {
		accepted, collisions int
		failure              any
		stack                string
	}
	done := make(chan outcome, 1)
	go func() {
		result := outcome{}
		defer func() {
			result.failure = recover()
			if result.failure != nil {
				result.stack = string(debug.Stack())
			}
			done <- result
		}()
		tbl.ProcessUniqueCollision([]string{"id"}, [][]scm.Scmer{{scm.NewInt(42)}}, false,
			func(rows [][]scm.Scmer, _ func() bool) { result.accepted += len(rows) }, nil,
			func(string, []scm.Scmer) { result.collisions++ }, 0, nil)
	}()
	deadline := time.Now().Add(time.Second)
	for topology.operations.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("unique probe did not retain source topology")
		}
		runtime.Gosched()
	}
	// Complete the cold acquisition with a replacement containing the key.
	// The probe still owns the original topology, but GetRead returns successor.
	// Attach the existing authority before publishing the successor, as native
	// rebuild does. Otherwise the waiter may declare an unrelated IO cell in
	// the test-only gap between publication and coherence attachment.
	successor.coherence.Store(cell)
	r.forwardResource(resource, r.shardResource(successor))
	tbl.mu.Lock()
	tbl.Shards = []*storageShard{successor}
	tbl.publishTopologyLocked()
	tbl.mu.Unlock()
	cell.current.Store(successor)
	if _, err := cell.ownstate.Grant(request.Flight().ID(), mosi.S, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-done:
		if result.failure != nil || result.accepted != 0 || result.collisions != 1 {
			t.Fatalf("successor key missed: %+v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("unique probe did not finish after grant")
	}
}

func TestClusterWALReplayRecoversIntegerCounterWithoutAllocator(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tclusterwalcounter", 0)
	c := tbl.Columns[0]
	c.AutoIncrement = true
	c.UpdateSanitizer()
	tbl.publishShowColumnsSnapshot(true)
	tbl.schema.save()
	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{
		{scm.NewInt(10000), scm.NewString("explicit")},
		{scm.NewInt(10001), scm.NewString("explicit")},
		{scm.NewNil(), scm.NewString("generated")},
	}, nil, scm.NewNil(), false, nil)
	atomic.StoreUint64(&tbl.Auto_increment, 0)
	// Materialization replays already allocated IDs under shard ownership.
	// No live cluster/allocator is available here, as in a cold handover.
	oldMounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(oldMounts)
	oldService := nodeScans.Swap(nil)
	defer nodeScans.Store(oldService)
	rows := 0
	for _, source := range tbl.ActiveShards() {
		// A fresh unpublished loader avoids the database loader's eager
		// AUTO_INCREMENT recovery replaying this same fixture a second time.
		encoded, err := json.Marshal(source)
		if err != nil {
			t.Fatal(err)
		}
		shard := new(storageShard)
		if err := json.Unmarshal(encoded, shard); err != nil {
			t.Fatal(err)
		}
		shard.t = tbl
		func() {
			shard.mu.Lock()
			defer shard.mu.Unlock()
			shard.load()
			rows += len(shard.inserts)
		}()
		if shard.logfile != nil {
			shard.logfile.Close()
		}
	}
	if rows != 3 || atomic.LoadUint64(&tbl.Auto_increment) != 10002 {
		t.Fatalf("local WAL recovery: rows=%d counter=%d", rows, atomic.LoadUint64(&tbl.Auto_increment))
	}
}

func TestClusterInsertKeepsLocalLaneBeforeForeignNewest(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tclusterinsertlane")
	local := tbl.ActiveShards()[0]
	for i := 0; i < 5; i++ {
		tbl.Shards = append(tbl.Shards, NewShard(tbl))
	}
	tbl.publishTopologyLocked()
	oldMounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(oldMounts)
	cluster := &nodeCluster{self: clusterMember{ID: "local"}, databases: map[string]bool{tbl.schema.Name: true}}
	cluster.initializeManager()
	for _, shard := range tbl.ActiveShards() {
		cluster.declareShard(shard, shard == local)
	}
	oldService := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(oldService)
	for i := 0; i < 10; i++ {
		shard, release, next := tbl.routeFreeInsert(tbl.activeTopology(), 1)
		if shard != local || next != tbl.activeTopology() {
			t.Fatal("foreign newest displaced local writer")
		}
		release()
	}
	foreign := tbl.ActiveShards()[5]
	if cluster.manager.shardCell(foreign).ownstate.Phase() == mosi.M {
		t.Fatal("routing claimed foreign permission")
	}
}

func TestClusterAppendImportRetainsLiveShardAndCounter(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tclusterappendimport")
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(5)}}, nil, scm.NewNil(), false, nil)
	original := tbl.ActiveShards()[0]
	encoded, err := json.Marshal(tbl)
	if err != nil {
		t.Fatal(err)
	}
	var incoming table
	if err = json.Unmarshal(encoded, &incoming); err != nil {
		t.Fatal(err)
	}
	incoming.schema = tbl.schema
	for _, shard := range incoming.Shards {
		shard.t = &incoming
	}
	incoming.Shards = append(incoming.Shards, NewShard(&incoming))
	incoming.publishTopologyLocked()
	atomic.StoreUint64(&incoming.Auto_increment, 100)
	r := &replicationManager{}
	if merged, err := r.mergeClusterInsertTopology(tbl, &incoming); !merged || err != nil {
		t.Fatalf("append rejected: %v", err)
	}
	if tbl.ActiveShards()[0] != original || len(original.inserts) != 1 || len(tbl.ActiveShards()) != 2 {
		t.Fatal("live generation replaced or lost")
	}
	if atomic.LoadUint64(&tbl.Auto_increment) != 100 {
		t.Fatal("counter update discarded with unchanged declaration")
	}
	incoming.Shards = append(incoming.Shards, NewShard(&incoming))
	incoming.publishTopologyLocked()
	incoming.Columns[0].Typ = "TEXT"
	if merged, _ := r.mergeClusterInsertTopology(tbl, &incoming); merged {
		t.Fatal("semantic DDL mistaken for writer append")
	}
}

func TestClusterForwardKeepsCurrentIncarnationReplayMark(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local", Boot: "current"}}
	cluster.initializeManager()
	r := cluster.manager
	from := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	to := from
	to.Shard = uuid.NewString()
	r.resourceCell(from).ownstate.Initialize(mosi.M)
	r.resourceCell(to)
	r.updateRecord(from, func(record *replicaRecord) bool {
		record.State, record.Owner = "M", "local"
		record.Requests = map[string]replicaRequestMark{"local": {Boot: "current", Sequence: 1, ID: "completed", Completed: true}}
		return true
	})
	r.updateRecord(to, func(record *replicaRecord) bool {
		record.Requests = map[string]replicaRequestMark{"local": {Boot: "predecessor", Sequence: 99}}
		return true
	})
	r.forwardResource(from, to)
	mark := r.directory.Records[r.resourceKey(to)].Requests["local"]
	if mark.Boot != "current" || mark.Sequence != 1 || !mark.Completed || mark.ID != "completed" {
		t.Fatalf("forwarding discarded completed current-process request: %+v", mark)
	}
}

func TestClusterReadUpgradeRetainsRegisteredEpoch(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	r := cluster.manager
	resource := mosiResource{Database: "tenant"}
	cell := r.resourceCell(resource)
	flight, _ := cell.ownstate.Request(mosi.S)
	cell.ownstate.Request(mosi.M)
	if err := r.acceptGrant(mosiPacket{Resource: resource, ID: flight.Flight().ID(), State: "S", Epoch: 12, Owner: "owner"}); err != nil {
		t.Fatal(err)
	}
	if cell.ownstate.Phase() != mosi.MR || r.directory.Records[resource.key()].Epoch != 12 {
		t.Fatal("weaker grant lost epoch or admitted a reader")
	}
	select {
	case <-flight.Flight().Done():
		t.Fatal("weaker grant completed the write")
	default:
	}
}

func TestClusterAppendImportFollowsNativeUUIDForwarding(t *testing.T) {
	for _, stale := range []bool{true, false} {
		name := "stale-reference"
		if !stale {
			name = "local-modifier"
		}
		t.Run(name, func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "tclusterappendforward")
			original := tbl.ActiveShards()[0]
			cluster := &nodeCluster{self: clusterMember{ID: "local"}}
			cluster.initializeManager()
			r := cluster.manager
			cell := cluster.declareShard(original, true)
			encoded, err := json.Marshal(tbl)
			if err != nil {
				t.Fatal(err)
			}
			var incoming table
			if err := json.Unmarshal(encoded, &incoming); err != nil {
				t.Fatal(err)
			}
			incoming.schema = tbl.schema
			for _, shard := range incoming.Shards {
				shard.t = &incoming
			}
			successor := NewShard(tbl)
			r.forwardResource(r.shardResource(original), r.shardResource(successor))
			if stale {
				tbl.Shards[0] = successor
				tbl.publishTopologyLocked()
				cell.current.Store(successor)
			} else {
				incoming.Shards[0] = successor
			}
			incoming.Shards = append(incoming.Shards, NewShard(&incoming))
			incoming.publishTopologyLocked()
			merged, err := r.mergeClusterInsertTopology(tbl, &incoming)
			if stale {
				if !merged || err != nil || tbl.ActiveShards()[0] != successor || len(tbl.ActiveShards()) != 2 {
					t.Fatalf("stale append replaced successor or lost lane: merged=%v error=%v", merged, err)
				}
			} else if merged || err == nil || tbl.ActiveShards()[0] != original || len(tbl.ActiveShards()) != 1 {
				t.Fatal("append replaced a local modifier or partially published its topology")
			}
		})
	}
}

func TestClusterRestartedInvalidPeerAcknowledgesRevocation(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	peer := &mosiPeer{outgoing: make(chan mosiMessage, 4), done: make(chan struct{})}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: "owner"}, peer: peer}}})
	resource := mosiResource{Database: "tenant"}
	r := cluster.manager
	predecessor := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	target := predecessor
	target.Shard = uuid.NewString()
	if err := r.invalidateResource(context.Background(), "owner", mosiPacket{Resource: target, ID: "unknown-revoke", Epoch: 42, Forwards: []shardForward{{predecessor, target}}}); err != nil {
		t.Fatal(err)
	}
	if r.lookupCell(predecessor) != nil || r.resourceCell(target).ownstate.Phase() != mosi.IO {
		t.Fatal("restart revocation fabricated predecessor authority")
	}
	if err := r.invalidateResource(context.Background(), "owner", mosiPacket{Resource: resource, ID: "revoke", Epoch: 42}); err != nil {
		t.Fatal(err)
	}
	if r.resourceCell(resource).ownstate.Phase() != mosi.IO || r.directory.Records[resource.key()].Epoch != 42 {
		t.Fatal("revocation fabricated access or lost recovery epoch")
	}
	r.resourceCell(resource).ownstate.Publish(mosi.S)
	if err := r.invalidateResource(context.Background(), "owner", mosiPacket{Resource: resource, ID: "future", Epoch: 43}); err == nil {
		t.Fatal("future revocation replaced an active read proof")
	}
}

func TestClusterConcurrentForwardsPreserveAuthorityEpoch(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	tbl := setupScanParallelTestTable(t, "tclusteraliaspublication")
	original := tbl.ActiveShards()[0]
	cell := cluster.declareShard(original, true)
	resource := *cell.resource.Load()
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool { record.Epoch = 71; return true })
	var pending sync.WaitGroup
	for range 64 {
		pending.Add(1)
		next := &storageShard{t: tbl, uuid: uuid.New()}
		go func() {
			defer pending.Done()
			cluster.manager.forwardResource(resource, cluster.manager.shardResource(next))
		}()
	}
	pending.Wait()
	current := *cell.resource.Load()
	record := cluster.manager.directory.Records[current.key()]
	if record.Epoch != 71 || record.State != "M" || record.Owner != "local" || len(cluster.manager.directory.Records) != 1 {
		t.Fatal("alias publication lost authoritative directory entry")
	}
}

func TestClusterRequestSequenceBelongsToProcessIncarnation(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local", Boot: "new-process"}}
	cluster.initializeManager()
	r := cluster.manager
	resource := mosiResource{Database: "tenant"}
	cell := r.resourceCell(resource)
	cell.ownstate.Initialize(mosi.O)
	cell.clusterstate.Initialize(mosi.O)
	pending, _ := cell.ownstate.Request(mosi.M)
	r.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.State, record.Epoch = "local", "O", 5
		record.Requests = map[string]replicaRequestMark{"local": {Sequence: 99, Write: true, ID: "old-flight", Completed: true, Boot: "old-process"}}
		return true
	})
	packet := mosiPacket{Kind: "write", Resource: resource, ID: pending.Flight().ID(), Sequence: 1, Boot: "new-process"}
	if err := r.coordinate(context.Background(), "local", packet); err != nil {
		t.Fatal(err)
	}
	if cell.ownstate.Phase() != mosi.M || cell.clusterstate.Phase() != mosi.M || r.directory.Records[resource.key()].Epoch != 6 {
		t.Fatal("new process's first request was suppressed or published inward before outward authority")
	}
	if err := r.coordinate(context.Background(), "local", packet); err != nil || r.directory.Records[resource.key()].Epoch != 6 {
		t.Fatal("same-incarnation replay executed twice")
	}
	packet.Boot = "old-process"
	if err := r.coordinate(context.Background(), "local", packet); err == nil {
		t.Fatal("queued old-incarnation request was accepted")
	}
}

func TestClusterInvalidationFollowsRebuiltOwnerUUID(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	peer := &mosiPeer{outgoing: make(chan mosiMessage, 4), done: make(chan struct{})}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: "owner"}, peer: peer}}})
	r := cluster.manager
	from := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	to := from
	to.Shard = uuid.NewString()
	cell := r.resourceCell(from)
	cell.ownstate.Initialize(mosi.S)
	cell.clusterstate.Initialize(mosi.S)
	r.updateRecord(from, func(record *replicaRecord) bool {
		record.Resource, record.Owner, record.Epoch = from, "owner", 7
		return true
	})
	packet := mosiPacket{Resource: to, ID: "revoke", Epoch: 7, Forwards: []shardForward{{from, to}}}
	if err := r.acceptRevocationRoutes("stranger", packet); err == nil || r.lookupCell(to) != nil {
		t.Fatal("foreign revocation changed native UUID routing")
	}
	if err := r.invalidateResource(context.Background(), "owner", packet); err != nil {
		t.Fatal(err)
	}
	if r.lookupCell(to) != cell || cell.ownstate.Phase() != mosi.IO || cell.clusterstate.Phase() != mosi.IO {
		t.Fatal("successor invalidation left predecessor readable")
	}
}

func TestClusterReferenceAcknowledgesInstalledGrant(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tclustergrantack", 4)
	RebuildTable(tbl, true, false)
	shard := tbl.ActiveShards()[0]
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	peer := &mosiPeer{outgoing: make(chan mosiMessage, 4), done: make(chan struct{})}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: "memcp://11111111-1111-4111-8111-111111111111"}, peer: peer}}})
	r := cluster.manager
	resource := r.shardResource(shard)
	cell := cluster.declareShard(shard, false)
	pending, _ := cell.ownstate.Request(mosi.S)
	r.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Epoch = "memcp://11111111-1111-4111-8111-111111111111", 7
		return true
	})
	packet := mosiPacket{Kind: "reference", Resource: resource, ID: pending.Flight().ID(), Epoch: 7, Content: shard.uuid.String(), State: "S", Owner: "memcp://11111111-1111-4111-8111-111111111111"}
	if err := cluster.installReference("memcp://11111111-1111-4111-8111-111111111111", packet); err != nil {
		t.Fatal(err)
	}
	ack := <-peer.outgoing
	if ack.Packet == nil || ack.Packet.Kind != "ack" || cell.ownstate.Phase() != mosi.S || cell.clusterstate.Phase() != mosi.S {
		t.Fatal("reference acknowledged before publishing its installed grant")
	}
	select {
	case <-pending.Flight().Done():
		if pending.Flight().Err() != nil {
			t.Fatal(pending.Flight().Err())
		}
	default:
		t.Fatal("reference ACK left the acquired flight pending")
	}
	if err := r.invalidateResource(context.Background(), "memcp://11111111-1111-4111-8111-111111111111", mosiPacket{Resource: resource, ID: "revoke", Epoch: 7}); err != nil {
		t.Fatal(err)
	}
	packet.Kind = "grant"
	if err := r.acceptGrant(packet); err != nil {
		t.Fatal(err)
	}
	if cell.ownstate.Phase() != mosi.IO {
		t.Fatal("completed grant replay resurrected the invalidated reader")
	}
}

func TestClusterCounterReservationKeepsSharedSchemaOwner(t *testing.T) {
	tbl, engine := createDurabilityTestTable(t, "tclustercounterowner", 0)
	cluster := &nodeCluster{self: clusterMember{ID: "local"}, ctx: context.Background(), databases: map[string]bool{tbl.schema.Name: true}}
	cluster.initializeManager()
	r := cluster.manager
	resource := mosiResource{Database: tbl.schema.Name}
	cell := r.resourceCell(resource)
	cell.ownstate.Initialize(mosi.O)
	cell.clusterstate.Initialize(mosi.O)
	r.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Epoch, record.State = "local", 11, "O"
		record.Readers = map[string]bool{"reader": true}
		return true
	})
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	reply, err := cluster.reserveIDs(context.Background(), clusterIDRequest{Database: tbl.schema.Name, Table: tbl.Name, Count: 8})
	if err != nil || reply.Base != 0 {
		t.Fatalf("reserve: %+v %v", reply, err)
	}
	if cell.ownstate.Phase() != mosi.O || r.directory.Records[resource.key()].Epoch != 11 || !r.directory.Records[resource.key()].Readers["reader"] {
		t.Fatal("reservation upgraded schema ownership or revoked its reader")
	}
	var saved struct {
		Tables map[string]struct {
			Counter uint64 `json:"Auto_increment"`
		} `json:"tables"`
	}
	if err := json.Unmarshal(engine.ReadSchema(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Tables[tbl.Name].Counter != 8 {
		t.Fatal("acknowledged range was not in the ordinary durable schema checkpoint")
	}
}

func TestClusterHandoverAckKeepsNewerInboundGrant(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tclusterhandovernewgrant", 4)
	RebuildTable(tbl, true, false)
	cluster := &nodeCluster{self: clusterMember{ID: "local"}, ctx: context.Background()}
	cluster.initializeManager()
	r := cluster.manager
	shard := tbl.ActiveShards()[0]
	cell := cluster.declareShard(shard, true)
	resource := r.shardResource(shard)
	peer := &mosiPeer{outgoing: make(chan mosiMessage, 8), done: make(chan struct{})}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: "remote"}, peer: peer}}})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- r.coordinate(ctx, "remote", mosiPacket{Kind: "write", ID: "outgoing", Resource: resource, Sequence: 1})
	}()
	var header mosiPacket
	select {
	case message := <-peer.outgoing:
		header = *message.Packet
		if header.Kind != "reference" || header.State != "M" {
			t.Fatalf("unexpected handover: %+v", header)
		}
	case err := <-done:
		t.Fatalf("handover ended before reference: %v", err)
	case <-ctx.Done():
		t.Fatal("handover did not send reference")
	}
	// The new owner has installed M and can answer a fresh read request before
	// this old source's outgoing ACK worker resumes. No wall-clock sleep orders it.
	pending, _ := cell.resolved().ownstate.Request(mosi.S)
	if err := r.acceptGrant(mosiPacket{Resource: header.Resource, ID: pending.Flight().ID(), State: "S", Owner: "remote", Epoch: header.Epoch, Content: header.Content}); err != nil {
		t.Fatal(err)
	}
	ack := header
	ack.Kind = "ack"
	if err := r.receivePacket("remote", &ack); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("handover did not finish after ACK")
	}
	record := r.directory.Records[r.resourceKey(resource)]
	if cell.resolved().ownstate.Phase() != mosi.S || record.State != "S" || record.Owner != "remote" || record.Epoch != header.Epoch {
		t.Fatalf("outgoing ACK overwrote newer grant: phase=%s record=%+v", cell.resolved().ownstate.Phase(), record)
	}
	if !record.Requests["remote"].Completed {
		t.Fatal("outgoing request completion was lost")
	}
}

func TestClusterReferenceGrantSurvivesInvalidSuccessor(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tclusterreferencependinguuid", 4)
	RebuildTable(tbl, true, false)
	shard := tbl.ActiveShards()[0]
	remote := "memcp://11111111-1111-4111-8111-111111111111"
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	peer := &mosiPeer{outgoing: make(chan mosiMessage, 4), done: make(chan struct{})}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: remote}, peer: peer}}})
	r := cluster.manager
	target := r.shardResource(shard)
	predecessor := target
	predecessor.Shard = uuid.NewString()
	old := r.resourceCell(predecessor)
	old.current.Store(shard)
	shard.coherence.Store(old)
	old.ownstate.Initialize(mosi.S)
	old.clusterstate.Initialize(mosi.S)
	r.updateRecord(predecessor, func(record *replicaRecord) bool {
		record.Owner, record.State, record.Epoch = remote, "S", 3
		return true
	})
	pending, _ := old.ownstate.Request(mosi.M)
	invalid := r.resourceCell(target)
	invalid.ownstate.Initialize(mosi.IO)
	invalid.clusterstate.Initialize(mosi.IO)
	r.updateRecord(target, func(record *replicaRecord) bool {
		record.Owner, record.State, record.Epoch = remote, "IO", 4
		return true
	})
	packet := mosiPacket{Kind: "reference", ID: pending.Flight().ID(), Resource: predecessor, Epoch: 5, State: "M", Owner: r.id, Content: target.Shard, Forwards: []shardForward{{From: predecessor, To: target}}}
	if err := cluster.installReference(remote, packet); err != nil {
		t.Fatal(err)
	}
	ack := <-peer.outgoing
	current := r.resourceCell(target)
	if ack.Packet == nil || ack.Packet.Kind != "ack" || current.ownstate.Phase() != mosi.M || current.clusterstate.Phase() != mosi.M {
		t.Fatalf("reference ACK lost the pending writer: phase=%s", current.ownstate.Phase())
	}
	select {
	case <-pending.Flight().Done():
		if pending.Flight().Err() != nil {
			t.Fatal(pending.Flight().Err())
		}
	default:
		t.Fatal("reference ACK left original flight pending")
	}
	record := r.directory.Records[r.resourceKey(target)]
	if record.Owner != r.id || record.Epoch != 5 {
		t.Fatalf("incoming owner grant was lost: %+v", record)
	}
}
