/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "sync"
import "testing"
import "context"
import "github.com/launix-de/memcp/scm"
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

func TestClusterTransactionExportKeepsNumericIdentityAndLocalRegistration(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_export_identity", 2)
	tx := NewTxContext(TxACID)
	tx.AddToDeleteMask(tbl.ActiveShards()[0], 0)
	defer tx.Rollback()
	localID := tx.ID
	cluster := globalTestCluster(t, tbl.schema.Name)
	previous := nodeScans.Swap(&nodeScanService{cluster: cluster})
	defer nodeScans.Store(previous)
	value, err := tx.HandleClusterRequest(context.Background(), scm.ClusterRequest{Method: "view"})
	if err != nil {
		t.Fatal(err)
	}
	view := value.(scm.Scmer).Slice()
	identity := view[6].String()
	if tx.ID != localID || view[2].Int() != int64(localID) || tx.clusterNumber.Load() == 0 {
		t.Fatal("export replaced peer-local registration or omitted cluster identity")
	}
	tx.ensureClusterIdentity()
	if tx.ClusterObjectID() != identity || tx.getShardTx(tbl.ActiveShards()[0]).clusterTxID != identity {
		t.Fatal("later shared access changed an already exported identity")
	}
}

func TestClusterGlobalContactsRestoreHintsWithoutAuthority(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_global_contacts", 1)
	source := globalTestCluster(t, tbl.schema.Name)
	source.nextTransactionID()
	target := globalTestCluster(t, tbl.schema.Name)
	target.members = make(map[string]clusterMember)
	cell := target.manager.resourceCell(globalResource)
	cell.ownstate.Initialize(mosi.IO)
	cell.clusterstate.Initialize(mosi.IO)
	if err := target.restoreGlobalContacts(); err != nil {
		t.Fatal(err)
	}
	if target.members[source.self.ID].Address != source.self.Address || cell.ownstate.Phase() != mosi.IO || cell.global.Load() != nil {
		t.Fatal("contact recovery lost discovery or fabricated authority")
	}
	data := *source.manager.resourceCell(globalResource).global.Load()
	data.Namespace = "foreign-namespace"
	encoded, _ := json.Marshal(data)
	if err := target.installGlobal(cell, mosiPacket{Object: string(encoded), Content: globalContent(encoded)}); err == nil {
		t.Fatal("accepted another shared namespace")
	}
}

func TestClusterGlobalFirstHandoverBeforeAnyIDAllocation(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_global_first_handover", 1)
	source := globalTestCluster(t, tbl.schema.Name)
	content, err := source.prepareGlobal()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(source.manager.resourceCell(globalResource).global.Load())
	target := globalTestCluster(t, tbl.schema.Name)
	cell := target.manager.resourceCell(globalResource)
	cell.ownstate.Initialize(mosi.IO)
	if err := target.installGlobal(cell, mosiPacket{Object: string(data), Content: content}); err != nil {
		t.Fatal(err)
	}
	if cell.global.Load().NextID != 1 || cell.ownstate.Phase() != mosi.IO {
		t.Fatal("initial handover allocated an ID or granted rights")
	}
}

func TestClusterImportedTransactionRegistersItsOwnPeerLocalNumber(t *testing.T) {
	local := NewTxContext(TxACID)
	defer local.Rollback()
	identity := uuid.NewString()
	handle := scm.ClusterHandle{View: scm.NewSlice([]scm.Scmer{
		scm.NewInt(int64(TxACID)), scm.NewBool(false), scm.NewInt(int64(local.ID)),
		scm.NewString("cluster/" + identity), scm.NewInt(0), scm.NewInt(0), scm.NewString(identity),
	})}
	first := importTransactionState(handle).Any().(*TxContext)
	second := importTransactionState(handle).Any().(*TxContext)
	if first.ID == local.ID || second.ID != first.ID || first.ClusterObjectID() != identity {
		t.Fatal("import reused another local registration or changed its cluster identity")
	}
}
