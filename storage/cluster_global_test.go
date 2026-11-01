/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "sync"
import "testing"
import "context"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/storage/mosi"

func globalTestCluster(t *testing.T, database string) *nodeCluster {
	t.Helper()
	self := clusterMember{"memcp://" + uuid.NewString(), uuid.NewString(), "ws://127.0.0.1:4321/cluster"}
	cluster := &nodeCluster{self: self, ctx: context.Background(), databases: map[string]bool{database: true}}
	cluster.initializeManager()
	cluster.view.Store(&clusterView{members: []clusterMember{self}})
	cell := cluster.manager.resourceCell(globalResource)
	cell.ownstate.Initialize(mosi.M)
	cell.clusterstate.Initialize(mosi.M)
	return cluster
}

func TestClusterGlobalIDBlocksSurviveRestartWithoutReusingTail(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_global_ids", 1)
	first := globalTestCluster(t, tbl.schema.Name)
	if id := first.nextTransactionID(); id != 1 {
		t.Fatal(id)
	}
	var wg sync.WaitGroup
	ids := make(chan uint64, 128)
	for range 128 {
		wg.Go(func() { ids <- first.nextTransactionID() })
	}
	wg.Wait()
	close(ids)
	seen := map[uint64]bool{1: true}
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate ID %d", id)
		}
		seen[id] = true
	}
	// Authority here is an explicit live-peer recovery fixture, never inferred
	// from the saved high-water mark. A real reboot starts with zero/IO rights.
	restarted := globalTestCluster(t, tbl.schema.Name)
	if id := restarted.nextTransactionID(); id != clusterIDBlockSize+1 {
		t.Fatalf("reused a reserved tail after restart: %d", id)
	}
}

func TestClusterGlobalTransferRetainsDataButDoesNotGrantRights(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_global_transfer", 1)
	source := globalTestCluster(t, tbl.schema.Name)
	source.nextTransactionID()
	data := source.manager.resourceCell(globalResource).global.Load()
	encoded, _ := json.Marshal(data)
	target := globalTestCluster(t, tbl.schema.Name)
	cell := target.manager.resourceCell(globalResource)
	cell.ownstate.Initialize(mosi.IO)
	packet := mosiPacket{Resource: globalResource, Content: globalContent(encoded), Object: string(encoded)}
	if err := target.installGlobal(cell, packet); err != nil {
		t.Fatal(err)
	}
	if cell.ownstate.Phase() != mosi.IO || cell.global.Load().NextID != clusterIDBlockSize+1 {
		t.Fatal("payload installation fabricated authority or lost high-water mark")
	}
	packet.Object += " "
	if err := target.installGlobal(cell, packet); err == nil {
		t.Fatal("accepted changed payload")
	}
}
