/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "testing"
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
