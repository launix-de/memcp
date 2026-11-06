/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "sync"
import "time"
import "context"
import "runtime"
import "strings"
import "testing"
import "sync/atomic"
import "encoding/json"
import "runtime/debug"
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
	successor.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(42)}}, false, nil, nil, false, nil, nil)
	write.Release()
	cluster := &nodeCluster{self: clusterMember{ID: "local"}, ctx: context.Background(), databases: map[string]bool{tbl.schema.Name: true}}
	cluster.initializeManager()
	r := cluster.manager
	cell := cluster.declareShard(source, false)
	resource := r.shardResource(source)
	r.updateRecord(resource, func(record *replicaRecord) bool { record.Owner = "remote"; return true })
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
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
			func(rows [][]scm.Scmer, _ *uniqueWriteGuard) { result.accepted += len(rows) }, nil,
			func(string, []scm.Scmer) { result.collisions++ }, 0, nil, nil, nil)
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
			rows += len(shard.deltaRows())
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
	if tbl.ActiveShards()[0] != original || len(original.deltaRows()) != 1 || len(tbl.ActiveShards()) != 2 {
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
	peer := &mosiPeer{outgoing: newMosiMailbox(), done: make(chan struct{})}
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
	peer := &mosiPeer{outgoing: newMosiMailbox(), done: make(chan struct{})}
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
	peer := &mosiPeer{outgoing: newMosiMailbox(), done: make(chan struct{})}
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
	ack := takeMosi(peer.outgoing)
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

// A local DDL request can publish MR before it acquires the allocator's
// directory boundary. The counter checkpoint still belongs to the O owner;
// acknowledging this range must make it durable without completing that DDL.
func TestClusterCounterCheckpointDuringLocalSchemaUpgrade(t *testing.T) {
	tbl, engine := createDurabilityTestTable(t, "tclustercounterupgrade", 0)
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
	releaseCoordination := cell.coordinate(context.Background())
	defer releaseCoordination()
	tbl.reserveLocalAutoIncrement(0, 8)
	pending, action := cell.ownstate.Request(mosi.M)
	if action != mosi.Send || pending.Retained() != mosi.O {
		t.Fatal("fixture did not start an O-to-M upgrade")
	}
	tbl.schema.save(cell)
	if cell.ownstate.Snapshot() != pending || !r.directory.Records[resource.key()].Readers["reader"] {
		t.Fatal("counter checkpoint completed the upgrade or revoked a reader")
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
		t.Fatal("reserved range was not durable while the upgrade was pending")
	}
	// A directory which has actually handed ownership away must still refuse
	// checkpoints, even if its local phase has not yet caught up.
	r.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner = "successor"
		return true
	})
	defer func() {
		if failure := recover(); failure != "counter checkpoint lost its owner" {
			t.Errorf("non-owner checkpoint: expected ownership rejection, got %v", failure)
		}
	}()
	tbl.schema.save(cell)
}

func TestClusterHandoverAckKeepsNewerInboundGrant(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tclusterhandovernewgrant", 4)
	RebuildTable(tbl, true, false)
	cluster := &nodeCluster{self: clusterMember{ID: "local"}, ctx: context.Background(), service: &nodeScanService{}}
	cluster.initializeManager()
	r := cluster.manager
	shard := tbl.ActiveShards()[0]
	cell := cluster.declareShard(shard, true)
	resource := r.shardResource(shard)
	previous := resource
	previous.Shard = uuid.NewString()
	oldest := resource
	oldest.Shard = uuid.NewString()
	r.forwardResource(previous, resource)
	r.forwardResource(oldest, previous)
	peer := &mosiPeer{outgoing: newMosiMailbox(), done: make(chan struct{})}
	schemaPeer := &mosiPeer{outgoing: newMosiMailbox(), done: make(chan struct{})}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: "remote"}, peer: peer}, {member: clusterMember{ID: "schema-owner"}, peer: schemaPeer}}})
	schema := mosiResource{Database: tbl.schema.Name}
	r.resourceCell(schema).ownstate.Establish(mosi.S)
	r.updateRecord(schema, func(record *replicaRecord) bool { record.Owner, record.Epoch = "schema-owner", 3; return true })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- r.coordinate(ctx, "remote", mosiPacket{Kind: "write", ID: "outgoing", Resource: resource, Sequence: 1})
	}()
	// A generation route must be acknowledged before its reference can wake
	// the next writer. It still names the source until the handover completes.
	select {
	case <-schemaPeer.outgoing.ready:
		message := takeMosi(schemaPeer.outgoing)
		var update struct {
			Return string
			Method string
			Data   []replicaRecord
		}
		if err := unmarshalNode(message.Data, &update); err != nil {
			t.Fatal(err)
		}
		if update.Method != "cluster.location_update" {
			t.Fatal("missing pre-reference generation route", update)
		}
		locations := make(map[string]replicaRecord)
		for _, entry := range update.Data {
			locations[entry.Resource.Shard] = entry
		}
		for _, wanted := range []mosiResource{r.canonicalResource(resource), previous, oldest} {
			if entry := locations[wanted.Shard]; entry.Resource != wanted || entry.Owner != "local" {
				t.Fatalf("missing historical UUID route before reference: %+v", wanted)
			}
		}
		// A schema owner which never saw these rebuilds must route old
		// schema/query UUIDs too, without gaining any shard permission.
		catalog := &nodeCluster{self: clusterMember{ID: "schema-owner"}, databases: map[string]bool{tbl.schema.Name: true}}
		catalog.initializeManager()
		catalog.manager.resourceCell(schema).ownstate.Initialize(mosi.M)
		catalog.manager.updateRecord(schema, func(record *replicaRecord) bool { record.Owner = "schema-owner"; return true })
		if err := catalog.registerLocations(ctx, update.Data); err != nil {
			t.Fatal(err)
		}
		for _, wanted := range []mosiResource{previous, oldest} {
			entry, err := catalog.shardLocation(wanted)
			if err != nil || entry.Owner != "local" || catalog.manager.resourceCell(wanted).ownstate.Phase() != mosi.IO {
				t.Fatalf("historical lookup lost route or gained authority: %+v %v", entry, err)
			}
		}
		if peer.outgoing.len() != 0 {
			t.Fatal("reference preceded schema location acknowledgement")
		}
		route, _ := cluster.service.handles.Load(controlReturnSlot(update.Return))
		deliverTestCompletion(t, route, scanCompletion{})
	case <-ctx.Done():
		t.Fatal("generation route missing", ctx.Err())
	}
	var header mosiPacket
	select {
	case <-peer.outgoing.ready:
		message := takeMosi(peer.outgoing)
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
	peer := &mosiPeer{outgoing: newMosiMailbox(), done: make(chan struct{})}
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
	ack := takeMosi(peer.outgoing)
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

func TestClusterForwardAlwaysPublishesResolvableTarget(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	r := cluster.manager
	from := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	r.resourceCell(from).ownstate.Establish(mosi.M)
	r.updateRecord(from, func(record *replicaRecord) bool {
		record.Resource, record.Owner, record.State = from, "local", "M"
		return true
	})
	for range 256 {
		r.resourceCell(mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()})
	}
	stop := make(chan struct{})
	violations := make(chan string, 1)
	var readers sync.WaitGroup
	for range 8 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				target := r.canonicalResource(from)
				if r.lookupCell(target) == nil {
					select {
					case violations <- "unregistered target " + target.key():
					default:
					}
					return
				}
				routes := r.shardRoutes(from)
				for _, route := range routes {
					if route.To != routes[0].To {
						select {
						case violations <- "mixed transfer targets in one captured snapshot":
						default:
						}
						return
					}
				}
			}
		}()
	}
	for range 64 {
		r.forwardResource(from, mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()})
	}
	close(stop)
	readers.Wait()
	select {
	case reason := <-violations:
		t.Fatal("incoherent routing publication:", reason)
	default:
	}
}

func TestClusterForwardRetainsCapturedRoutingIdentity(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	r := cluster.manager
	from := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	middle, last := from, from
	middle.Shard, last.Shard = uuid.NewString(), uuid.NewString()
	cell := r.resourceCell(from)
	initial := r.cellSnapshot.Load()
	if target := (*initial)[from.key()].resource; target == nil || *target != from {
		t.Fatal("cell constructor published an incomplete routing identity")
	}
	r.forwardResource(from, middle)
	captured := r.cellSnapshot.Load()
	r.forwardResource(from, last)
	if target := (*captured)[from.key()].resource; target == nil || *target != middle {
		t.Fatal("later forwarding rewrote a captured routing identity")
	}
	if (*captured)[middle.key()].cell != cell || r.lookupCell(last) != cell || r.canonicalResource(from) != last {
		t.Fatal("forwarding lost an alias or cloned native authority")
	}
	for _, route := range r.shardRoutes(from) {
		if route.To != last || route.From == last {
			t.Fatal("transfer exposed a reversed or obsolete successor route", route)
		}
	}
}

// Reporting consumes the native publication; both aliases must identify the
// same successor authority without changing an already captured routing view.
func TestClusterReportFollowsPublishedSuccessor(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	r := cluster.manager
	from := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	to := from
	to.Shard = uuid.NewString()
	cell := r.resourceCell(from)
	r.updateRecord(from, func(record *replicaRecord) bool {
		record.Owner, record.State, record.Epoch = "remote", "S", 7
		return true
	})
	cell.ownstate.Initialize(mosi.S)
	captured := r.cellSnapshot.Load()
	r.forwardResource(from, to)
	if target := (*captured)[from.key()].resource; target == nil || *target != from {
		t.Fatal("forwarding rewrote the captured predecessor identity")
	}
	for _, resource := range []mosiResource{from, to} {
		report := cluster.report(resource)
		if report.Resource != to || report.Owner != "remote" || report.State != "S" || report.Epoch != 7 {
			t.Fatalf("forwarded authority lost: %+v", report)
		}
	}
}

func TestClusterReportBeforeAuthorityInitialization(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	resource := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	check := func() {
		t.Helper()
		report := cluster.report(resource)
		if report.Resource != resource || report.State != "IO" || report.Owner != "" {
			t.Fatalf("uninitialized authority acquired rights: %+v", report)
		}
	}
	check()
	if cluster.manager.lookupCell(resource) != nil {
		t.Fatal("report registered an unknown resource")
	}
	// Construction publishes a complete identity, while MOSI remains unknown.
	cell := cluster.manager.resourceCell(resource)
	if target := cell.resource.Load(); target == nil || *target != resource {
		t.Fatal("constructor published an incomplete identity")
	}
	check()
	if cell.ownstate.Initialized() || cell.clusterstate.Initialized() {
		t.Fatal("report established unknown authority")
	}
}

// INSERT can wait for a shard whose current writer is finishing a remote
// transaction. Retaining an earlier bucket's writer across that wait would
// prevent the same transaction from completing on its coordinator.
func TestClusterPartitionInsertReleasesWritersBeforeRemoteAcquire(t *testing.T) {
	for _, unique := range []bool{false, true} {
		t.Run(fmt.Sprint(unique), func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "tinsertpartialwriters")
			tbl.ShardMode = ShardModePartition
			tbl.PDimensions = []shardDimension{{Column: "id", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(10)}}}
			first, second := NewShard(tbl), NewShard(tbl)
			tbl.PShards = []*storageShard{first, second}
			tbl.publishTopologyLocked()
			if unique {
				tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
			}
			cluster := globalTestCluster(t, tbl.schema.Name)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cluster.ctx = ctx
			owner := clusterMember{ID: "memcp://" + uuid.NewString(), Boot: uuid.NewString()}
			peer := &mosiPeer{ctx: ctx, done: make(chan struct{}), outgoing: newMosiMailbox()}
			cluster.view.Store(&clusterView{members: []clusterMember{cluster.self, owner}, links: []*clusterLink{{member: owner, peer: peer}}})
			mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
			defer sharedClusterMounts.Store(mounts)
			service := nodeScans.Swap(&nodeScanService{cluster: cluster})
			defer nodeScans.Store(service)
			r := cluster.manager
			r.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
			left, right := r.shardCell(first), r.shardCell(second)
			left.current.Store(first)
			left.ownstate.Initialize(mosi.M)
			right.current.Store(second)
			right.ownstate.Initialize(mosi.S)
			resource := *right.resource.Load()
			r.updateRecord(resource, func(record *replicaRecord) bool {
				record.Resource, record.Owner, record.State, record.Epoch = resource, owner.ID, "S", 1
				return true
			})
			if unique {
				// Peer execution now owns UNIQUE reservation preparation. Exercise that
				// same native admission boundary without pretending this mailbox-only
				// MOSI fixture implements the receiver/RPC transport. Real peer-channel
				// INSERT retries are covered by the distributed insertion scenarios.
				ready := newClusterReceiver(1, func(context.Context, scm.Scmer) (scm.Scmer, error) {
					return scm.NewNil(), fmt.Errorf("reservation became ready before all writers were admitted")
				})
				ready.Seal()
				defer ready.Close()
				_, err := first.serveUniqueScan(ctx, shardScanInvocation{neutral: scm.NewSlice([]scm.Scmer{
					scm.NewAny(ready), uniqueStringArray([]string{"id"}), uniqueRowArray([][]scm.Scmer{{scm.NewInt(1)}, {scm.NewInt(11)}}),
					scm.NewBool(false), scm.NewBool(true), uniqueStringArray([]string{first.uuid.String(), second.uuid.String()}),
				})})
				busy, ok := err.(uniqueReservationBusy)
				if !ok || busy.shard.Resource.Shard != second.uuid.String() {
					t.Fatalf("partial UNIQUE admission did not report the remote shard: %v", err)
				}
				if left.ownstate.Writing() {
					t.Fatal("UNIQUE admission retained its partial writer")
				}
			}
			done := make(chan any, 1)
			go func() {
				defer func() { done <- recover() }()
				if unique {
					// No partial reservation remains while native admission waits.
					access := second.GetWrite(nil)
					access.Release()
					return
				}
				tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}, {scm.NewInt(11)}}, nil, scm.NewNil(), false, nil)
			}()
			select {
			case <-peer.outgoing.ready:
				takeMosi(peer.outgoing)
			case <-time.After(5 * time.Second):
				t.Fatal("INSERT did not request its remote M")
			}
			if left.ownstate.Writing() {
				t.Error("INSERT retained an earlier writer across the remote acquisition")
			}
			if first.plannerDeltaRows.Load() != 0 || second.plannerDeltaRows.Load() != 0 {
				t.Error("INSERT mutated rows before admitting every target")
			}
			pending := right.ownstate.Snapshot().Flight()
			if pending == nil {
				t.Fatal("missing M flight")
			}
			if _, err := right.ownstate.Grant(pending.ID(), mosi.M, func() error { return nil }); err != nil {
				t.Fatal(err)
			}
			select {
			case failure := <-done:
				if failure != nil {
					t.Fatalf("INSERT: %v", failure)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("INSERT did not finish after granting M")
			}
			if unique {
				tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}, {scm.NewInt(11)}}, nil, scm.NewNil(), false, nil)
			}
			if first.plannerDeltaRows.Load() != 1 || second.plannerDeltaRows.Load() != 1 {
				t.Fatal("admission retry lost or duplicated an inserted row")
			}
		})
	}
}

func TestClusterLocationBatchRejectsBeforePublishing(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}, databases: map[string]bool{"locations": true, "other": true}}
	cluster.initializeManager()
	cluster.view.Store(&clusterView{members: []clusterMember{cluster.self, {ID: "peer"}}})
	valid := replicaRecord{Resource: mosiResource{Database: "locations", Table: "items", Shard: uuid.NewString()}, Owner: "owner", Epoch: 7}
	foreign := valid
	foreign.Resource.Database = "other"
	for _, entries := range [][]replicaRecord{nil, {}, {valid, foreign}} {
		payload, err := marshalNode(entries)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := cluster.handleLocations(context.Background(), "peer", "cluster.location_update", payload); err == nil {
			t.Fatal("accepted invalid location batch")
		}
		if cluster.manager.lookupCell(valid.Resource) != nil {
			t.Fatal("invalid batch published its valid prefix")
		}
	}
	payload, _ := marshalNode([]replicaRecord{valid})
	if _, err := cluster.handleLocations(context.Background(), "unintroduced", "cluster.location_update", payload); err == nil {
		t.Fatal("unauthenticated member published a location")
	}
	if cluster.manager.lookupCell(valid.Resource) != nil {
		t.Fatal("unauthenticated location publication")
	}
}

func TestClusterInsertSelectsUniqueDomainOwnerWithoutClaiming(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tinsertforwardplacement")
	tbl.ShardMode = ShardModePartition
	tbl.PDimensions = []shardDimension{{Column: "id", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(10)}}}
	left, right := NewShard(tbl), NewShard(tbl)
	tbl.PShards = []*storageShard{left, right}
	tbl.publishTopologyLocked()
	cluster := globalTestCluster(t, tbl.schema.Name)
	peer := &mosiPeer{ctx: context.Background(), done: make(chan struct{}), outgoing: newMosiMailbox()}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: "owner"}, peer: peer}}})
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	for _, shard := range []*storageShard{left, right} {
		cell := cluster.manager.shardCell(shard)
		cell.ownstate.Initialize(mosi.S)
		cluster.manager.updateRecord(*cell.resource.Load(), func(record *replicaRecord) bool { record.Owner = "owner"; return true })
	}
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	columns, rows := []string{"id"}, [][]scm.Scmer{{scm.NewInt(11)}}
	if _, selected := tbl.insertPeer(columns, rows); selected != peer {
		t.Fatal("partition-local INSERT did not choose destination owner")
	}
	cluster.manager.shardCell(right).ownstate.Publish(mosi.M)
	if _, selected := tbl.insertPeer(columns, rows); selected != nil {
		t.Fatal("already-local M was unnecessarily delegated")
	}
	if _, handled := tbl.placeInsert(columns, rows, nil, scm.NewNil(), false, nil, nil); handled || cluster.insertPlacement.Load() != nil {
		t.Fatal("local M touched remote placement feedback")
	}
	tbl.Unique = append(tbl.Unique, uniqueKey{Id: "global", Cols: []string{"writer", "value"}})
	if _, selected := tbl.insertPeer(columns, rows); selected != peer {
		t.Fatal("global UNIQUE ignored the common input owner")
	}
	// Unpartitioned UNIQUE checks have the same shared input domain. Forward
	// to its owner instead of manufacturing another lane and claiming schema M.
	tbl.ShardMode = ShardModeFree
	tbl.Shards = []*storageShard{left, right}
	tbl.publishTopologyLocked()
	if _, selected := tbl.insertPeer(columns, rows); selected != peer {
		t.Fatal("free-shard UNIQUE ignored the common input owner")
	}
	select {
	case <-peer.outgoing.ready:
		takeMosi(peer.outgoing)
		t.Fatal("placement queried or claimed known MOSI state")
	default:
	}
	if cluster.manager.shardCell(left).ownstate.Phase() != mosi.S {
		t.Fatal("placement changed data authority")
	}
}

func TestClusterMutationScanBorrowsAcquiredWriter(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tmutationacquiredwriter")
	tbl.CreateColumn("value", "INT", nil, nil)
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	tbl.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(42), scm.NewInt(0)}}, nil, scm.NewNil(), false, nil)
	shard := tbl.ActiveShards()[0]
	cluster := globalTestCluster(t, tbl.schema.Name)
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
	cell := cluster.declareShard(shard, true)
	owned := shard.GetWrite(nil)
	release := sync.OnceFunc(owned.Release)
	defer release()
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		value, count, _ := shard.scan(scanAccess{}, nil, scm.NewBool(true), []string{"value", "$update"}, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			if !cell.ownstate.Writing() {
				panic("mutation callback lost M")
			}
			scm.Apply(a[2], scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewInt(a[1].Int() + 1)}))
			return scm.NewInt(a[0].Int() + 1)
		}), scm.NewInt(0), 0, nil, nil, nil, true)
		if count != 1 || value.Int() != 1 {
			panic("mutation scan did not update exactly one row")
		}
	}()
	select {
	case failure := <-done:
		if failure != nil {
			t.Fatal(failure)
		}
	case <-time.After(3 * time.Second):
		release()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("mutation scan did not release its writer")
		}
		t.Fatal("nil-TX mutation scan re-entered its already-acquired M")
	}
	if !cell.ownstate.Writing() {
		t.Fatal("scan released its caller's writer")
	}
	release()
	if cell.ownstate.Writing() {
		t.Fatal("scan leaked a writer scope")
	}
}

func TestClusterUniqueChannelSerializesNativeCollisionUpdates(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_unique_channel_collision")
	tbl.CreateColumn("value", "INT", nil, nil)
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	tbl.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(42), scm.NewInt(0)}}, nil, scm.NewNil(), false, nil)
	cluster := globalTestCluster(t, tbl.schema.Name)
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
	for _, shard := range tbl.ActiveShards() {
		cluster.declareShard(shard, true)
	}
	const workers, repetitions = 8, 8
	start := make(chan struct{})
	failures := make(chan any, workers)
	var running sync.WaitGroup
	for range workers {
		running.Go(func() {
			defer func() { failures <- recover() }()
			<-start
			for range repetitions {
				tbl.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(42), scm.NewInt(0)}}, []string{"value", "$update"},
					scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
						runtime.Gosched()
						return scm.Apply(values[1], scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewInt(values[0].Int() + 1)}))
					}), false, nil)
			}
		})
	}
	close(start)
	running.Wait()
	close(failures)
	for failure := range failures {
		if failure != nil {
			t.Fatalf("native collision: %v", failure)
		}
	}
	shard := tbl.ActiveShards()[0]
	lease := shard.GetRead(nil, false)
	defer lease.Release()
	id, found := shard.GetRecordidForUnique(exactScanAccess(newExactScanAccessSchema([]string{"id"}), []scm.Scmer{scm.NewInt(42)}), nil, true)
	reader := shard.ColumnReaderTx(nil, "value", false, ReadAuto)
	defer ReleaseColumnReader(reader)
	if !found || reader.GetValue(id).Int() != workers*repetitions {
		t.Fatalf("native collision lost increments: found=%v value=%v", found, reader.GetValue(id))
	}
	if n := tbl.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(42), scm.NewInt(0)}}, []string{"value", "$set:value"},
		scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			return scm.Apply(values[1], scm.NewInt(values[0].Int()+1))
		}), false, nil); n != 2 {
		t.Fatalf("reserved column setter affected %d rows", n)
	}
	id, found = shard.GetRecordidForUnique(exactScanAccess(newExactScanAccessSchema([]string{"id"}), []scm.Scmer{scm.NewInt(42)}), nil, true)
	updated := shard.ColumnReaderTx(nil, "value", false, ReadAuto)
	defer ReleaseColumnReader(updated)
	if !found || updated.GetValue(id).Int() != workers*repetitions+1 {
		t.Fatal("reserved column setter lost its native effect")
	}

}

func TestClusterNativeUniqueUpdateReleasesBorrowedReservation(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_unique_update_borrowed")
	tbl.CreateColumn("value", "INT", nil, nil)
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}, {Id: "value", Cols: []string{"value"}}}
	tbl.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(42), scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
	cluster := globalTestCluster(t, tbl.schema.Name)
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
	shard := tbl.ActiveShards()[0]
	cell := cluster.declareShard(shard, true)
	for range 8 {
		tbl.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{"id"},
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(a[0].Int() == 42) }),
			[]string{"value", "$update"}, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
				return scm.Apply(a[2], scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewInt(a[1].Int() + 1)}))
			}), scm.NewInt(0), scm.Globalenv.Vars[scm.Symbol("+")], false)
		if cell.ownstate.Writing() {
			t.Fatal("native UPDATE retained its MOSI writer")
		}
		guard := tbl.guardUniqueWrites([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(42), scm.NewInt(0)}}, false, 0, nil)
		if !guard.tryLock() {
			t.Fatal("native UPDATE retained a UNIQUE stripe after publishing")
		}
		guard.unlock()
	}
}

func TestClusterUniqueCallbackReleasesMOnPanic(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tuniquepanicrelease")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(42)}}, nil, scm.NewNil(), false, nil)
	shard := tbl.ActiveShards()[0]
	cluster := globalTestCluster(t, tbl.schema.Name)
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
	cell := cluster.declareShard(shard, true)
	var failure any
	func() {
		defer func() { failure = recover() }()
		tbl.ProcessUniqueCollision([]string{"id"}, [][]scm.Scmer{{scm.NewInt(42)}}, false,
			func([][]scm.Scmer, *uniqueWriteGuard) { t.Fatal("duplicate row accepted") }, []string{"$update"},
			func(string, []scm.Scmer) {
				if !cell.ownstate.Writing() {
					t.Fatal("callback executed without active M")
				}
				panic("callback failure")
			}, 0, nil, nil, nil)
	}()
	if failure != "callback failure" || cell.ownstate.Writing() {
		t.Fatalf("callback failure=%v retained writer=%v", failure, cell.ownstate.Writing())
	}
	access, ready := shard.getWrite(nil, false)
	if !ready {
		t.Fatal("failed callback stranded later writer")
	}
	access.Release()
}

func TestClusterUniquePreflightUsesMutationAdmissionAndKeepsNodeLimit(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tuniquemutationbudget")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(42)}}, nil, scm.NewNil(), false, nil)
	shard := tbl.ActiveShards()[0]
	cluster := globalTestCluster(t, tbl.schema.Name)
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
	cluster.declareShard(shard, true)
	budget := GlobalCache.Stat()
	defer GlobalCache.UpdateBudget(budget.MemoryBudget, budget.PersistedBudget, budget.MinimumMemory)
	tbl.shardRAMFull.Store(true)
	shard.setState(COLD)
	var readFailure any
	func() {
		defer func() { readFailure = recover() }()
		access := shard.GetRead(nil, false)
		defer access.Release()
	}()
	if _, rejected := readFailure.(shardReadDelegation); !rejected {
		t.Fatalf("ordinary read exceeded table reserve: %v", readFailure)
	}
	accepted := 0
	tbl.ProcessUniqueCollision([]string{"id"}, [][]scm.Scmer{{scm.NewInt(43)}}, false,
		func(rows [][]scm.Scmer, validate *uniqueWriteGuard) {
			if !validate.validReads() {
				t.Fatal("mutation preflight lost native read proof")
			}
			accepted += len(rows)
		}, nil,
		func(string, []scm.Scmer) { t.Fatal("unrelated key collided") }, 0, nil, nil, nil)
	if accepted != 1 {
		t.Fatal("fitting mutation preflight was refused at the read reserve")
	}
	shard.setState(COLD)
	// Change admission through its owner. A direct snapshot override can be
	// replaced by any cache-actor publication, including a pending size update.
	GlobalCache.UpdateBudget(0, budget.PersistedBudget, budget.MinimumMemory)
	GlobalCache.Stat()
	var writeFailure any
	func() {
		defer func() { writeFailure = recover() }()
		access := shard.getRead(nil, shardReadOptions{mutation: true})
		defer access.Release()
	}()
	if _, rejected := writeFailure.(shardReadDelegation); !rejected {
		t.Fatalf("mutation preflight exceeded node budget: %v", writeFailure)
	}
}

func TestClusterInsertCapacityRefusalPrecedesInvocation(t *testing.T) {
	previous := GlobalCache.shardCapacity.Swap(&shardRAMSnapshot{Allowed: false})
	defer GlobalCache.shardCapacity.Store(previous)
	db := &database{Name: "tenant"}
	request := remoteInsertTask{Version: scanWireVersion, Database: db.Name, Table: "items", Transaction: &scm.ClusterReference{}}
	// An unusable transaction reference and no invocation session prove that
	// refusal happens before importing captures, evaluating defaults or IDs.
	value, err := db.executeInsertTask(context.Background(), nil, request)
	if err != nil {
		t.Fatal(err)
	}
	if result, ok := value.(remoteInsertResult); !ok || !result.Rejected {
		t.Fatalf("capacity refusal = %#v", value)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := db.executeInsertTask(ctx, nil, request); err != context.Canceled {
		t.Fatalf("cancelled command admitted: %v", err)
	}
	request.Version = -1
	if _, err := db.executeInsertTask(context.Background(), nil, request); err == nil {
		t.Fatal("malformed command treated as a capacity refusal")
	}
}

func TestClusterWarmInsertStaysLocalAtColdAdmissionLimit(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "twarminsertadmission")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	shard := tbl.ActiveShards()[0]
	cluster := globalTestCluster(t, tbl.schema.Name)
	cluster.view.Store(&clusterView{members: []clusterMember{cluster.self}})
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
	cell := cluster.declareShard(shard, true)
	previous := GlobalCache.shardCapacity.Swap(&shardRAMSnapshot{Allowed: false})
	defer GlobalCache.shardCapacity.Store(previous)
	tbl.shardRAMFull.Store(true)
	if shard.state() == COLD {
		t.Fatal("fixture must already own warm payload")
	}
	count := tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(42)}}, nil, scm.NewNil(), false, nil)
	if count != 1 || shard.plannerDeltaRows.Load() != 1 {
		t.Fatal("warm INSERT did not use its local generation")
	}
	if cell.ownstate.Writing() || cluster.delegated.Load() != 0 {
		t.Fatal("warm INSERT retained M or delegated despite existing local access")
	}
}

func TestClusterUniqueReservationDeduplicatesSuccessorAliases(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_unique_reservation_alias")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	source := tbl.ActiveShards()[0]
	successor := NewShard(tbl)
	successor.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(42)}}, false, nil, nil, false, nil, nil)
	cluster := globalTestCluster(t, tbl.schema.Name)
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
	cell := cluster.declareShard(source, true)
	successor.coherence.Store(cell)
	cluster.manager.forwardResource(cluster.manager.shardResource(source), cluster.manager.shardResource(successor))
	cell.current.Store(successor)
	tbl.mu.Lock()
	tbl.Shards = []*storageShard{successor}
	tbl.publishTopologyLocked()
	tbl.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan *clusterReceiver, 1)
	receiver := newClusterReceiver(1, func(_ context.Context, value scm.Scmer) (scm.Scmer, error) {
		ready <- value.Any().(*clusterReceiver)
		return scm.NewNil(), nil
	})
	receiver.Seal()
	defer receiver.Close()
	input := shardScanInvocation{neutral: scm.NewSlice([]scm.Scmer{
		scm.NewAny(receiver), uniqueStringArray([]string{"id"}), uniqueRowArray([][]scm.Scmer{{scm.NewInt(42)}}),
		scm.NewBool(false), scm.NewBool(true), uniqueStringArray([]string{source.uuid.String(), successor.uuid.String()}),
	})}
	done := make(chan error, 1)
	go func() {
		defer func() {
			if value := recover(); value != nil {
				done <- fmt.Errorf("reservation panic: %v", value)
			}
		}()
		_, err := successor.serveUniqueScan(ctx, input)
		done <- err
	}()
	joined := false
	defer func() {
		cancel()
		if !joined {
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Error("reservation did not join after cancellation")
			}
		}
	}()
	select {
	case commands := <-ready:
		for _, id := range []string{source.uuid.String(), successor.uuid.String()} {
			value, err := callUniqueChannel(ctx, commands, scm.NewString("probe"), scm.NewString(id), uniqueStringArray([]string{"id"}), scm.NewSlice([]scm.Scmer{scm.NewInt(42)}), uniqueStringArray(nil))
			if err != nil || !value.Slice()[0].Bool() {
				t.Fatalf("alias probe: %v %v", value, err)
			}
		}
		if _, err := callUniqueChannel(ctx, commands, scm.NewString("finish")); err != nil {
			t.Fatal(err)
		}
	case err := <-done:
		joined = true
		t.Fatalf("successor alias reacquired its own nil-TX writer: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("reservation blocked on its own successor")
	}
	err := <-done
	joined = true
	if err != nil {
		t.Fatal(err)
	}
	if cell.ownstate.Writing() {
		t.Fatal("alias reservation retained its writer")
	}
}

func TestClusterInsertPlacementLearnsRepeatAccess(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_insert_placement_feedback")
	shard := tbl.ActiveShards()[0]
	cluster := globalTestCluster(t, tbl.schema.Name)
	slot := cluster.insertPlacementSlot(shard)
	now := time.Now()
	if tbl.shouldClaimInsert(shard, "owner", slot.Load(), now) {
		t.Fatal("one-off access speculatively acquired remote payload")
	}
	for range 16 {
		recordInsertDelegation(slot, shard, "owner", now, time.Millisecond)
	}
	if tbl.shouldClaimInsert(shard, "owner", slot.Load(), now) {
		t.Fatal("unknown generation size justified a speculative payload load")
	}
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(7)}}, nil, scm.NewNil(), false, nil)
	claimed := false
	for i := range 16 {
		recordInsertDelegation(slot, shard, "owner", now.Add(time.Duration(i)*time.Millisecond), time.Millisecond)
		if tbl.shouldClaimInsert(shard, "owner", slot.Load(), now.Add(time.Duration(i)*time.Millisecond)) {
			claimed = true
			break
		}
	}
	if !claimed {
		t.Fatal("repeated remote work never paid for local ownership")
	}
	if tbl.shouldClaimInsert(shard, "replacement-owner", slot.Load(), now) {
		t.Fatal("old owner feedback reused after ownership changed")
	}
	if tbl.shouldClaimInsert(shard, "owner", slot.Load(), now.Add(time.Hour)) {
		t.Fatal("sporadic access reused an expired burst")
	}
	// A hot input can exceed this node's entire RAM. Frequency cannot override
	// capacity; the native owner must retain its generation and execute remotely.
	tbl.PlannerRowEstimate.value.Store(1 << 40)
	if tbl.shouldClaimInsert(shard, "owner", slot.Load(), now) {
		t.Fatal("frequent inserts chose a generation larger than node RAM")
	}
}

func TestClusterLocalUniqueUpdaterCannotEscapeReservation(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_unique_local_escaped_updater")
	tbl.CreateColumn("value", "INT", nil, nil)
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	columns := []string{"id", "value"}
	rows := [][]scm.Scmer{{scm.NewInt(42), scm.NewInt(0)}}
	tbl.Insert(columns, rows, nil, scm.NewNil(), false, nil)
	cluster := globalTestCluster(t, tbl.schema.Name)
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
	for _, shard := range tbl.ActiveShards() {
		cluster.declareShard(shard, true)
	}
	var update scm.Scmer
	tbl.Insert(columns, rows, []string{"$update"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		update = values[0]
		return scm.NewBool(false)
	}), false, nil)
	func() {
		defer func() {
			if recover() == nil {
				t.Error("escaped updater mutated after its native reservation ended")
			}
		}()
		scm.Apply(update, scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewInt(99)}))
	}()
	shard := tbl.ActiveShards()[0]
	lease := shard.GetRead(nil, false)
	defer lease.Release()
	id, found := shard.GetRecordidForUnique(exactScanAccess(newExactScanAccessSchema([]string{"id"}), []scm.Scmer{scm.NewInt(42)}), nil, true)
	reader := shard.ColumnReaderTx(nil, "value", false, ReadAuto)
	defer ReleaseColumnReader(reader)
	if !found || reader.GetValue(id).Int() != 0 {
		t.Fatal("expired updater changed visible data")
	}
}

func TestClusterUniqueCoordinatorRetriesAfterLaneAppend(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_unique_coordinator_lane_append")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	tbl.Shards = []*storageShard{NewShard(tbl), NewShard(tbl)}
	tbl.publishTopologyLocked()
	cluster := globalTestCluster(t, tbl.schema.Name)
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
	for _, shard := range tbl.ActiveShards() {
		cluster.declareShard(shard, true)
	}
	columns, rows := []string{"id"}, [][]scm.Scmer{{scm.NewInt(42)}}
	guard := tbl.guardUniqueWrites(columns, rows, false, 0, nil)
	defer guard.release()
	set := tbl.reserveUniqueDomain(columns, rows, false, nil, nil, &guard)
	defer set.close(false)
	// Another INSERT may publish an empty writer lane between reservation and
	// the coordinator's first probe. No callback/effect has run yet: this must
	// restart preflight, not report a worker-domain extension to the application.
	next := NewShard(tbl)
	cluster.declareShard(next, true)
	tbl.Shards = append(tbl.Shards, next)
	tbl.publishTopologyLocked()
	defer func() {
		if _, retry := recover().(uniqueReadRetry); !retry {
			t.Error("coordinator did not retry a changed preflight domain")
		}
	}()
	set.probe(next, columns, rows[0], []string{"$update"})
}

func TestClusterUniqueKeyChangeChecksAppendedLane(t *testing.T) {
	for _, collision := range []bool{false, true} {
		t.Run(fmt.Sprint(collision), func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "t_unique_key_change_new_lane")
			tbl.CreateColumn("value", "INT", nil, nil)
			tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}, {Id: "value", Cols: []string{"value"}}}
			tbl.Shards = []*storageShard{NewShard(tbl), NewShard(tbl)}
			tbl.publishTopologyLocked()
			columns := []string{"id", "value"}
			rows := [][]scm.Scmer{{scm.NewInt(42), scm.NewInt(0)}}
			tbl.Insert(columns, rows, nil, scm.NewNil(), false, nil)
			next := NewShard(tbl)
			value := int64(100)
			if collision {
				value = 1
			}
			next.Insert(columns, [][]scm.Scmer{{scm.NewInt(99), scm.NewInt(value)}}, false, nil, nil, false, nil, nil)
			cluster := globalTestCluster(t, tbl.schema.Name)
			mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
			defer sharedClusterMounts.Store(mounts)
			service := nodeScans.Swap(&nodeScanService{cluster: cluster})
			defer nodeScans.Store(service)
			cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
			for _, shard := range tbl.ActiveShards() {
				cluster.declareShard(shard, true)
			}
			var failure any
			func() {
				defer func() { failure = recover() }()
				tbl.Insert(columns, rows, []string{"value", "$update"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
					cluster.declareShard(next, true)
					tbl.Shards = append(tbl.Shards, next)
					tbl.publishTopologyLocked()
					return scm.Apply(values[1], scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewInt(values[0].Int() + 1)}))
				}), false, nil)
			}()
			if collision {
				if failure == nil || !strings.Contains(fmt.Sprint(failure), "Duplicate entry") {
					t.Fatalf("new lane's conflicting key was not checked: %v", failure)
				}
			} else if failure != nil {
				t.Fatalf("unrelated lane append broke an atomic update: %v", failure)
			}
			var actual int64 = -1
			for _, shard := range tbl.ActiveShards() {
				lease := shard.GetRead(nil, false)
				id, found := lease.shard.GetRecordidForUnique(exactScanAccess(newExactScanAccessSchema([]string{"id"}), []scm.Scmer{scm.NewInt(42)}), nil, true)
				if found {
					reader := lease.shard.ColumnReaderTx(nil, "value", false, ReadAuto)
					actual = reader.GetValue(id).Int()
					ReleaseColumnReader(reader)
				}
				lease.Release()
			}
			want := int64(1)
			if collision {
				want = 0
			}
			if actual != want || tbl.CountExact() != 2 {
				t.Fatalf("key change published incorrectly: value=%d want=%d rows=%d", actual, want, tbl.CountExact())
			}
		})
	}
}

func TestClusterUniqueUnreservedReadNeedsHeldParticipant(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "t_unique_unreserved_partition")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	left, middle, right := NewShard(tbl), NewShard(tbl), NewShard(tbl)
	tbl.Shards = []*storageShard{left, middle}
	tbl.publishTopologyLocked()
	cluster := globalTestCluster(t, tbl.schema.Name)
	mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
	defer sharedClusterMounts.Store(mounts)
	service := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(service)
	cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
	cluster.declareShard(left, true)
	cluster.declareShard(middle, true)
	columns := []string{"id"}
	guard := tbl.guardUniqueWrites(columns, [][]scm.Scmer{{scm.NewInt(7)}}, false, 0, nil)
	defer guard.release()
	set := tbl.reserveUniqueDomain(columns, [][]scm.Scmer{{scm.NewInt(7)}}, false, nil, nil, &guard)
	defer set.close(false)
	cluster.declareShard(right, true)
	tbl.ShardMode, tbl.PShards = ShardModePartition, []*storageShard{left, middle, right}
	tbl.PDimensions = []shardDimension{{Column: "id", NumPartitions: 3, Pivots: []scm.Scmer{scm.NewInt(10), scm.NewInt(20)}}}
	tbl.publishTopologyLocked()
	additional := tbl.guardUniqueWrites(columns, [][]scm.Scmer{{scm.NewInt(99)}}, false, 0, nil)
	guard.extend(additional.mask)
	defer func() {
		if _, retry := recover().(uniqueReadRetry); !retry {
			t.Error("unreserved read accepted a domain without a retained participant")
		}
	}()
	set.probe(right, columns, []scm.Scmer{scm.NewInt(99)}, nil)
}

func TestClusterUniqueContentionDoesNotMoveOwnership(t *testing.T) {
	for _, kind := range []string{"stripe", "writer"} {
		t.Run(kind, func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "tunique_local_contention")
			tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
			cluster := globalTestCluster(t, tbl.schema.Name)
			mounts := sharedClusterMounts.Swap(&map[string]bool{tbl.schema.Name: true})
			defer sharedClusterMounts.Store(mounts)
			service := nodeScans.Swap(&nodeScanService{cluster: cluster})
			defer nodeScans.Store(service)
			cluster.manager.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
			shard := tbl.ActiveShards()[0]
			cell := cluster.manager.shardCell(shard)
			cell.current.Store(shard)
			cell.ownstate.Initialize(mosi.M)
			rows := [][]scm.Scmer{{scm.NewInt(7)}}
			guard := tbl.guardUniqueWrites([]string{"id"}, rows, false, 0, nil)
			if kind == "stripe" {
				guard.lock()
				defer guard.unlock()
			} else {
				writer, _ := cell.ownstate.TryWrite(nil, false)
				if writer == nil {
					t.Fatal("fixture writer not admitted")
				}
				defer writer.Release()
			}
			ready := newClusterReceiver(1, func(context.Context, scm.Scmer) (scm.Scmer, error) {
				return scm.NewNil(), fmt.Errorf("contended reservation became ready")
			})
			ready.Seal()
			defer ready.Close()
			_, err := shard.serveUniqueScan(context.Background(), shardScanInvocation{neutral: scm.NewSlice([]scm.Scmer{
				scm.NewAny(ready), uniqueStringArray([]string{"id"}), uniqueRowArray(rows), scm.NewBool(false), scm.NewBool(true), uniqueStringArray([]string{shard.uuid.String()}),
			})})
			busy, ok := err.(uniqueReservationBusy)
			if !ok || busy.retry != (kind == "stripe") || busy.capacity {
				t.Fatalf("contention was classified as placement work: %v", err)
			}
			if kind == "stripe" && cell.ownstate.Writing() {
				t.Fatal("contention retained its partial writer")
			}
			// This cluster fixture has no owner-discovery transport. Contention
			// must return locally without asking it to claim or move this shard.
			if kind == "stripe" {
				tbl.admitUnique(busy, nil)
			}
		})
	}
}

type liveReferencePhaseProbe struct {
	id         string
	firstChunk func()
}

func (*liveReferencePhaseProbe) ClusterObjectType() string     { return "shard-transfer" }
func (probe *liveReferencePhaseProbe) ClusterObjectID() string { return probe.id }
func (probe *liveReferencePhaseProbe) Serialize(bindings *scm.SerializationContext) (scm.Scmer, error) {
	return bindings.ReferenceExpression(probe)
}
func (probe *liveReferencePhaseProbe) HandleClusterRequest(_ context.Context, request scm.ClusterRequest) (any, error) {
	if request.Method != "next" {
		return nil, fmt.Errorf("unexpected transfer probe request")
	}
	probe.firstChunk()
	return scm.NewNil(), nil // deliberately incomplete: no payload may grant rights
}

func TestClusterLiveReferencePublishesEpochBeforePayloadCanMergeCells(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_live_reference_epoch", 4)
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	cluster := &nodeCluster{self: clusterMember{ID: client.authNodeID(), Boot: client.id}, ctx: context.Background(), service: client}
	cluster.initializeManager()
	remote := peer.objectPeer.ID
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: remote}, peer: peer}}})
	r := cluster.manager
	target := r.shardResource(tbl.ActiveShards()[0])
	predecessor := target
	predecessor.Shard = uuid.NewString()
	old := r.resourceCell(predecessor)
	old.ownstate.Initialize(mosi.S)
	old.clusterstate.Initialize(mosi.S)
	r.updateRecord(predecessor, func(record *replicaRecord) bool { record.Owner, record.Epoch = remote, 3; return true })
	pending, _ := old.ownstate.Request(mosi.M)
	next := r.resourceCell(target)
	next.ownstate.Initialize(mosi.IO)
	r.updateRecord(target, func(record *replicaRecord) bool { record.Owner, record.Epoch = remote, 4; return true })
	next.ownstate.Request(mosi.M)
	type observation struct {
		cell  *replicaCell
		epoch uint64
	}
	observed := make(chan observation, 1)
	probe := &liveReferencePhaseProbe{id: uuid.NewString(), firstChunk: func() {
		// Live payload installation can publish its physical successor before
		// returning to installReference. Its validated owner version must win.
		r.forwardResource(predecessor, target)
		r.mu.Lock()
		epoch := r.directory.Records[r.resourceKey(target)].Epoch
		r.mu.Unlock()
		observed <- observation{r.resourceCell(target), epoch}
	}}
	ref, err := owner.exportObject(objectOwnerPeer(t, owner, client), probe, scm.NewNil())
	if err != nil {
		t.Fatal(err)
	}
	packet := mosiPacket{Kind: "reference", ID: pending.Flight().ID(), Resource: predecessor, Epoch: 5, State: "M", Owner: r.id, Content: target.Shard, Capture: &ref}
	if err := cluster.installReference(remote, packet); err == nil {
		t.Fatal("incomplete live payload granted permission")
	}
	select {
	case got := <-observed:
		if got.cell != old || got.epoch != 5 {
			t.Fatalf("live installation retired the validated request before publishing its epoch: cell=%p want=%p epoch=%d", got.cell, old, got.epoch)
		}
	case <-time.After(time.Second):
		t.Fatal("live transfer did not reach its payload boundary")
	}
	if old.ownstate.Phase() != mosi.MR || old.ownstate.Snapshot().Flight() != pending.Flight() || old.ownstate.ReadReady() {
		t.Fatal("incomplete transfer lost its pending request or fabricated a grant")
	}
}
