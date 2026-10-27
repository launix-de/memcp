/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "testing"
import "sync"
import "context"
import "github.com/google/uuid"
import "encoding/json"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

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
	incoming.Shards = append(incoming.Shards, NewShard(&incoming))
	incoming.publishTopologyLocked()
	atomic.StoreUint64(&incoming.Auto_increment, 100)
	if !mergeClusterInsertTopology(tbl, &incoming) {
		t.Fatal("append rejected")
	}
	if tbl.ActiveShards()[0] != original || len(original.inserts) != 1 || len(tbl.ActiveShards()) != 2 {
		t.Fatal("live generation replaced or lost")
	}
	if atomic.LoadUint64(&tbl.Auto_increment) != 100 {
		t.Fatal("counter update discarded with unchanged declaration")
	}
	incoming.Columns[0].Typ = "TEXT"
	if mergeClusterInsertTopology(tbl, &incoming) {
		t.Fatal("semantic DDL mistaken for writer append")
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

func TestClusterRestartedInvalidPeerAcknowledgesRevocation(t *testing.T) {
	cluster := &nodeCluster{self: clusterMember{ID: "local"}}
	cluster.initializeManager()
	peer := &mosiPeer{outgoing: make(chan mosiMessage, 4), done: make(chan struct{})}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: "owner"}, peer: peer}}})
	resource := mosiResource{Database: "tenant"}
	r := cluster.manager
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

func TestClusterConcurrentAliasesPreserveAuthorityEpoch(t *testing.T) {
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
		next := &storageShard{t: tbl, uuid: uuid.New(), lineage: original.logicalID()}
		go func() { defer pending.Done(); cluster.aliasShard(resource, next) }()
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
