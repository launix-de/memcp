/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "weak"
import "time"
import "sync"
import "context"
import "strings"
import "runtime"
import "testing"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

func objectOwnerPeer(t *testing.T, owner, client *nodeScanService) *mosiPeer {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		owner.mu.Lock()
		for peer := range owner.peers {
			if peer.objectPeer.ID == client.authNodeID() {
				owner.mu.Unlock()
				return peer
			}
		}
		owner.mu.Unlock()
		time.Sleep(time.Millisecond)
	}
	t.Fatal("missing authenticated object peer")
	return nil
}
func objectEntryCount(s *nodeScanService) int {
	r := s.objectRegistry()
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.entries)
}
func waitObjectGC(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		runtime.GC()
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("GC/reference publication did not finish")
}

func TestClusterObjectIndependentHandlesAndIncarnation(t *testing.T) {
	owner := &nodeScanService{id: uuid.NewString()}
	defer owner.stopObjects()
	peer := &mosiPeer{objectPeer: clusterObjectPeer{uuid.NewString(), uuid.NewString()}}
	value := scm.NewSession()
	object, _ := scm.ClusterObjectOf(value)
	view, _ := scm.ClusterObjectValue(object, context.Background(), scm.ClusterRequest{Method: "view"})
	first, err := owner.exportObject(peer, object, view)
	if err != nil {
		t.Fatal(err)
	}
	second, err := owner.exportObject(peer, object, view)
	if err != nil {
		t.Fatal(err)
	}
	if first.URI != second.URI || first.Token == second.Token {
		t.Fatal("exports did not share owner identity with independent handles")
	}
	// Two successive Go lifetimes of one peer may overlap while old GC delivery
	// is pending. They need generation IDs, not a boolean peer membership.
	sessionRPC := &scanRPCSession{service: owner, peer: peer}
	acquire := func(ref scm.ClusterReference) scm.ClusterReference {
		lease := uuid.NewString()
		data, _ := json.Marshal(clusterObjectAcquisition{ref, lease})
		if _, err := sessionRPC.handleObject(context.Background(), "object.acquire", data); err != nil {
			t.Fatal(err)
		}
		ref.Token = lease
		return ref
	}
	first, second = acquire(first), acquire(second)
	if err := owner.releaseObject(peer, first); err != nil {
		t.Fatal(err)
	}
	if err := owner.releaseObject(peer, first); err != nil {
		t.Fatal(err)
	}
	if objectEntryCount(owner) != 1 {
		t.Fatal("late old release revoked the newer reference")
	}
	foreign := &mosiPeer{objectPeer: clusterObjectPeer{peer.objectPeer.ID, uuid.NewString()}}
	if err := owner.releaseObject(foreign, second); err == nil {
		t.Fatal("another process incarnation released a live handle")
	}
	if err := owner.releaseObject(peer, second); err != nil {
		t.Fatal(err)
	}
	if objectEntryCount(owner) != 0 {
		t.Fatal("last reference did not deregister owner object")
	}
}

//go:noinline
func prepareHistoricalObject(t *testing.T, owner *nodeScanService, peer *mosiPeer, table *table) (scm.ClusterReference, weak.Pointer[storageShard], remoteScanTask) {
	old := table.ActiveShards()[0]
	view, err := scm.ClusterObjectValue(old, context.Background(), scm.ClusterRequest{Method: "view"})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := owner.exportObject(peer, old, view)
	if err != nil {
		t.Fatal(err)
	}
	task := makeRemoteScanTask(t, remoteScanShard{Schema: table.filterSchemaFingerprint(), Resource: mosiResource{Database: table.schema.Name, Table: table.Name, Shard: old.uuid.String()}})
	return ref, weak.Make(old), task
}

//go:noinline
func historicalObjectAlive(ptr weak.Pointer[storageShard]) bool { return ptr.Value() != nil }

func TestClusterObjectHistoricalScanRebuildAndPeerGC(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_cluster_object_gc", 4)
	RebuildTable(table, true, false)
	peer, client, _ := setupRemoteScanConnection(t)
	defer client.stopObjects()
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	ref, old, task := prepareHistoricalObject(t, owner, source, table)
	// The peer's query retains only its imported object. The owner's registry
	// must keep the exact old Go shard alive after topology points elsewhere.
	remote, err := client.importObject(context.Background(), peer, ref)
	if err != nil {
		t.Fatal(err)
	}
	RebuildTable(table, true, false)
	if table.ActiveShards()[0].uuid.String() == task.Shard.Resource.Shard {
		t.Fatal("fixture did not replace the shard")
	}
	runtime.GC()
	if !historicalObjectAlive(old) || objectEntryCount(owner) != 1 {
		t.Fatal("owner lost generation still retained by remote query")
	}
	task.Input = &remote.Any().(*remoteShardObject).handle.Reference
	var reduced remoteScanResult
	if err := client.control(context.Background(), peer, "scan.execute", task, &reduced); err != nil {
		t.Fatal(err)
	}
	value, err := schemeValue(reduced.Value).reconstruct()
	if err != nil || value.Int() != 9 {
		t.Fatal("old-generation statistics changed", value, err)
	}
	task.Input = nil // the task borrowed a pointer into the imported handle
	runtime.KeepAlive(remote)
	remote = scm.NewNil()
	waitObjectGC(t, func() bool { return objectEntryCount(owner) == 0 })
	waitObjectGC(t, func() bool { return !historicalObjectAlive(old) })
	// Retiring the old object never changes the newly published partition.
	if table.ActiveShards()[0].uuid.String() == task.Shard.Resource.Shard {
		t.Fatal("release restored old topology")
	}
}

func TestClusterObjectSerializedSessionUsesOwnerEndpoint(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_cluster_session_wire", 1)
	_ = table
	peer, client, _ := setupRemoteScanConnection(t)
	defer client.stopObjects()
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	session := scm.NewSession()
	session.Func()(scm.NewString("x"), scm.NewInt(17))
	environment := &scm.Env{Vars: scm.Vars{scm.Symbol("session"): session}, Outer: &scm.Globalenv}
	proc := scm.Eval(scm.Read(t.Name(), `(lambda () (session "x"))`), environment)
	bindings := (&scm.SerializationContext{}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
		return owner.exportObject(source, object, view)
	})
	data, err := scm.MarshalProcedure(proc, bindings)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := scm.UnmarshalProcedure(data, scm.ProcedureDecode{Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) {
		return client.importObject(context.Background(), peer, ref)
	}})
	if err != nil {
		t.Fatal(err)
	}
	session.Func()(scm.NewString("x"), scm.NewInt(91))
	if result := scm.Apply(restored); result.Int() != 17 {
		t.Fatal("prepared session view changed during statement", result)
	}
	if objectEntryCount(owner) != 1 {
		t.Fatal("serializer bypassed object registration")
	}
	runtime.KeepAlive(restored)
}

func TestClusterPacketDispatchDoesNotBlockReaderAndKeepsTransferOrder(t *testing.T) {
	var mu sync.Mutex
	var order []string
	entered := make(chan struct{})
	resume := make(chan struct{})
	independent := make(chan struct{})
	p := &mosiPeer{done: make(chan struct{}), packet: func(packet *mosiPacket) error {
		if packet.Kind == "begin" {
			close(entered)
			<-resume
		}
		mu.Lock()
		order = append(order, packet.Kind)
		mu.Unlock()
		if packet.ID == "other" {
			close(independent)
		}
		return nil
	}}
	if !p.dispatchPacket(&mosiPacket{ID: "stream", Kind: "begin"}, 1) {
		t.Fatal("dispatch rejected")
	}
	<-entered
	if !p.dispatchPacket(&mosiPacket{ID: "stream", Kind: "end"}, 1) || !p.dispatchPacket(&mosiPacket{ID: "other", Kind: "grant"}, 1) {
		t.Fatal("dispatch rejected")
	}
	select {
	case <-independent:
	case <-time.After(time.Second):
		t.Fatal("blocked transfer prevented unrelated message execution")
	}
	close(resume)
	p.workers.Wait()
	mu.Lock()
	defer mu.Unlock()
	if fmt.Sprint(order) != "[grant begin end]" {
		t.Fatal("transfer packets reordered", order)
	}
	if p.packetBytes.Load() != 0 || len(p.packetOrder) != 0 {
		t.Fatal("completed packet workers leaked queue state")
	}
}

func TestClusterObjectOwnerCacheMapFactoryAndNestedRuntimeCapture(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	defer client.stopObjects()
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	cache := NewCacheMap()
	object, _ := scm.ClusterObjectOf(cache)
	view, err := scm.ClusterObjectValue(object, context.Background(), scm.ClusterRequest{Method: "view"})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := owner.exportObject(source, object, view)
	if err != nil {
		t.Fatal(err)
	}
	remote, err := client.importObject(context.Background(), peer, ref)
	if err != nil {
		t.Fatal(err)
	}
	session := scm.NewSession()
	session.Func()(scm.NewString("x"), scm.NewInt(7))
	env := &scm.Env{Vars: scm.Vars{scm.Symbol("session"): session}, Outer: &scm.Globalenv}
	factory := scm.Eval(scm.Read(t.Name(), `(lambda () (+ (session "x") 3))`), env)
	value := scm.Apply(remote, scm.NewString("key"), scm.NewNil(), factory)
	if value.Int() != 10 || scm.Apply(cache, scm.NewString("key")).Int() != 10 {
		t.Fatal("factory did not execute at cache owner", value)
	}
	if scm.Apply(remote, scm.NewString("key")).Int() != 10 {
		t.Fatal("factory result was not published in the imported view")
	}
	scm.Apply(cache, scm.NewString("owner-only"), scm.NewInt(17))
	if scm.Apply(remote, scm.NewString("owner-only")).Int() != 17 {
		t.Fatal("unprepared cache miss silently became nil instead of using owner")
	}
	// The owner mutation must round-trip. Subsequent reads of this prepared
	// accessor stay local, including a cached SQL NULL.
	scm.Apply(remote, scm.NewString("key"), scm.NewNil())
	if !scm.Apply(remote, scm.NewString("key")).IsNil() || !scm.Apply(cache, scm.NewString("key")).IsNil() {
		t.Fatal("owner cache mutation lost NULL")
	}
	runtime.KeepAlive(remote)
}

func TestClusterObjectNativeOverlayReferenceKeepsPhysicalIdentity(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_cluster_overlay_ref", 4)
	peer, client, _ := setupRemoteScanConnection(t)
	defer client.stopObjects()
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	shard := table.ActiveShards()[0]
	tx := NewTxContext(TxACID)
	tx.mu.Lock()
	overlay := tx.getOrCreateShardTxLocked(shard)
	tx.mu.Unlock()
	defer tx.Rollback()
	overlay.mu.Lock()
	overlay.DeleteMask.Set(1, true)
	overlay.DeleteRecids = append(overlay.DeleteRecids, 1)
	overlay.mu.Unlock()
	view, err := scm.ClusterObjectValue(overlay, context.Background(), scm.ClusterRequest{Method: "view"})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := owner.exportObject(source, overlay, view)
	if err != nil {
		t.Fatal(err)
	}
	value, err := client.importObject(context.Background(), peer, ref)
	if err != nil {
		t.Fatal(err)
	}
	imported := value.Any().(*storageShardTransaction)
	if imported.shardUUID != shard.uuid || !imported.DeleteMask.Get(1) || imported.DeleteMask.Get(2) {
		t.Fatal("overlay did not preserve native UUID/visibility")
	}
	if _, err := scm.ClusterObjectValue(imported, context.Background(), scm.ClusterRequest{Method: "commit"}); err == nil {
		t.Fatal("read overlay acquired coordinator mutation rights")
	}
	runtime.KeepAlive(value)
}

func TestClusterObjectCatalogNamesNeedNoRegistration(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_cluster_catalog_ref", 1)
	peer, client, _ := setupRemoteScanConnection(t)
	defer client.stopObjects()
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	view, err := scm.ClusterObjectValue(table, context.Background(), scm.ClusterRequest{Method: "view"})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := owner.exportObject(source, table, view)
	if err != nil {
		t.Fatal(err)
	}
	value, err := client.importObject(context.Background(), peer, ref)
	if err != nil {
		t.Fatal(err)
	}
	if TableFromScmer(value) != table || ref.Token != "" || objectEntryCount(owner) != 0 {
		t.Fatal("catalog resolution allocated a temporary registration")
	}
}

func TestClusterObjectOfferCancellationCannotDropAcquiredReader(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	defer client.stopObjects()
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	session := scm.NewSession()
	object, _ := scm.ClusterObjectOf(session)
	view, _ := scm.ClusterObjectValue(object, context.Background(), scm.ClusterRequest{Method: "view"})
	pending, err := owner.exportObject(source, object, view)
	if err != nil {
		t.Fatal(err)
	}
	live, err := owner.exportObject(source, object, view)
	if err != nil {
		t.Fatal(err)
	}
	value, err := client.importObject(context.Background(), peer, live)
	if err != nil {
		t.Fatal(err)
	}
	owner.cancelObjectOffers(source, []scm.ClusterReference{pending, live})
	if objectEntryCount(owner) != 1 {
		t.Fatal("cancelled offer erased an acquired reader")
	}
	r := owner.objectRegistry()
	r.mu.Lock()
	tokens := len(r.entries[live.URI].leases)
	r.mu.Unlock()
	if tokens != 1 {
		t.Fatal("unacquired offer was retained", tokens)
	}
	runtime.KeepAlive(value)
}

func TestClusterObjectRepeatedImportSharesOneGCLifetime(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	session := scm.NewSession()
	object, _ := scm.ClusterObjectOf(session)
	view, _ := scm.ClusterObjectValue(object, context.Background(), scm.ClusterRequest{Method: "view"})
	ref, err := owner.exportObject(source, object, view)
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.importObject(context.Background(), peer, ref)
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.importObject(context.Background(), peer, ref)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := scm.ClusterObjectOf(first)
	b, _ := scm.ClusterObjectOf(second)
	if a != b {
		t.Fatal("one token reconstructed two independent GC leases")
	}
	first = scm.NewNil()
	a = nil
	for range 3 {
		runtime.GC()
	}
	if objectEntryCount(owner) != 1 {
		t.Fatal("first alias released a still live imported handle")
	}
	runtime.KeepAlive(second)
	runtime.KeepAlive(b)
	second = scm.NewNil()
	b = nil
	waitObjectGC(t, func() bool { return objectEntryCount(owner) == 0 })
}

func TestClusterPacketSchemaPublicationPrecedesDependentShard(t *testing.T) {
	entered := make(chan struct{})
	resume := make(chan struct{})
	data := make(chan struct{})
	other := make(chan struct{})
	p := &mosiPeer{done: make(chan struct{}), packet: func(packet *mosiPacket) error {
		switch packet.ID {
		case "schema":
			close(entered)
			<-resume
		case "data":
			select {
			case <-resume:
			default:
				t.Error("shard import overtook the schema introducing its identity")
			}
			close(data)
		case "independent":
			close(other)
		}
		return nil
	}}
	if !p.dispatchPacket(&mosiPacket{ID: "schema", Resource: mosiResource{Database: "dependent"}}, 1) {
		t.Fatal("schema dispatch failed")
	}
	<-entered
	if !p.dispatchPacket(&mosiPacket{ID: "data", Resource: mosiResource{Database: "dependent", Table: "t", Shard: uuid.NewString()}}, 1) ||
		!p.dispatchPacket(&mosiPacket{ID: "independent", Resource: mosiResource{Database: "independent"}}, 1) {
		t.Fatal("data dispatch failed")
	}
	select {
	case <-other:
	case <-time.After(time.Second):
		t.Fatal("schema work serialized an unrelated database")
	}
	select {
	case <-data:
		t.Fatal("dependent shard executed before schema publication")
	default:
	}
	close(resume)
	p.workers.Wait()
	select {
	case <-data:
	default:
		t.Fatal("dependent shard did not resume")
	}
}

func TestClusterObjectAbandonedCompletionDiscardsOffers(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	value := scm.NewSession()
	object, _ := scm.ClusterObjectOf(value)
	view, _ := scm.ClusterObjectValue(object, context.Background(), scm.ClusterRequest{Method: "view"})
	ref, err := owner.exportObject(source, object, view)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	results := make(chan scanCompletion, 1)
	receiver := newClusterReceiver(1, func(_ context.Context, v scm.Scmer) error { results <- v.Any().(scanCompletion); return nil })
	receiver.Seal()
	client.returns.Store(id, &receiverRoute{peer, receiver})
	defer client.returns.Delete(id)
	completion, _ := json.Marshal(scanCompletion{Return: id, Offers: []scm.ClusterReference{ref}})
	s := &scanRPCSession{service: client, peer: peer}
	if _, err := s.channelMessage(context.Background(), "scan.result", completion); err != nil {
		t.Fatal(err)
	}
	// Cancellation after delivery but before collection must release the offer.
	receiver.Close()
	abandoned := <-results
	client.discardObjectOffers(peer, abandoned.Offers)
	waitObjectGC(t, func() bool { return objectEntryCount(owner) == 0 })
	// A late duplicate is harmless and cannot repopulate a closed collector.
	if _, err := s.channelMessage(context.Background(), "scan.result", completion); err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatal("late completion was delivered to a closed collector")
	}
}

//go:noinline
func prepareRegisteredDiskGeneration(t *testing.T, owner *nodeScanService, peer *mosiPeer, tbl *table) (scm.ClusterReference, weak.Pointer[storageShard]) {
	old := tbl.ActiveShards()[0]
	view, err := scm.ClusterObjectValue(old, context.Background(), scm.ClusterRequest{Method: "view"})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := owner.exportObject(peer, old, view)
	if err != nil {
		t.Fatal(err)
	}
	// Drop one current row so the successor no longer owns its blob. The old
	// physical generation's column and manifest still need that blob on disk.
	old.UpdateFunction(0, false, false, nil)()
	RebuildTable(tbl, true, false)
	if tbl.ActiveShards()[0].uuid == old.uuid {
		t.Fatal("rebuild kept old UUID")
	}
	old.RemoveFromDisk()
	return ref, weak.Make(old)
}

func TestClusterObjectRegisteredGenerationProtectsFilesAndBlobs(t *testing.T) {
	defer setupGCTest(t)()
	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)
	insertLongRows(t, tbl, []string{
		strings.Repeat("A", maxInlineBlobBytes+800),
		strings.Repeat("B", maxInlineBlobBytes+800),
		strings.Repeat("C", maxInlineBlobBytes+800),
	})
	_, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	peer := objectOwnerPeer(t, owner, client)
	ref, old := prepareRegisteredDiskGeneration(t, owner, peer, tbl)
	blobs, _, complete := CleanDatabase(tbl.schema)
	if !complete || blobs != 0 {
		t.Fatal("registered historical blob was removed", blobs, complete)
	}
	found := false
	for _, name := range shardFilesOnDisk(t, "gcdb") {
		if strings.HasPrefix(name, ref.URI[strings.LastIndex(ref.URI, "/")+1:]+"-") {
			found = true
		}
	}
	if !found {
		t.Fatal("registered generation lost its column files")
	}
	owner.cancelObjectOffers(peer, []scm.ClusterReference{ref})
	waitObjectGC(t, func() bool { return !historicalObjectAlive(old) })
	blobs, files, complete := CleanDatabase(tbl.schema)
	if !complete || blobs == 0 || files == 0 {
		t.Fatal("released generation remained on disk", blobs, files, complete)
	}
}

func TestClusterDiscoveryRefreshesRoutingAfterConcurrentHandover(t *testing.T) {
	self := clusterMember{ID: "memcp://" + uuid.NewString()}
	remote := clusterMember{ID: "memcp://" + uuid.NewString()}
	peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 1)}
	service := &nodeScanService{}
	cluster := &nodeCluster{self: self, service: service}
	cluster.initializeManager()
	cluster.view.Store(&clusterView{members: []clusterMember{self, remote}, links: []*clusterLink{{member: remote, peer: peer}}})
	resource := mosiResource{Database: "concurrent_handover"}
	cell := cluster.manager.resourceCell(resource)
	cell.ownstate.Establish(mosi.IO)
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Epoch, record.State = self.ID, 9, "IO"
		return true
	})
	type result struct {
		owner string
		err   error
	}
	finished := make(chan result, 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() {
		owner, err := cluster.locate(ctx, resource)
		finished <- result{owner, err}
	}()
	// Discovery has sampled local IO and is awaiting another live peer. Publish
	// the successor now; the peer's already captured answer still names us.
	message := <-peer.outgoing
	var request struct {
		Return string `json:"return"`
	}
	if err := json.Unmarshal(message.Data, &request); err != nil {
		t.Fatal(err)
	}
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
		record.Handoff = remote.ID
		return true
	})
	data, _ := json.Marshal(replicaRecord{Resource: resource, Owner: self.ID, Epoch: 9, State: "IO"})
	entry, ok := service.returns.Load(request.Return)
	if !ok {
		t.Fatal("missing discovery return collector")
	}
	deliverTestCompletion(t, entry, scanCompletion{Return: request.Return, Data: data})
	reply := <-finished
	if reply.err != nil || reply.owner != remote.ID || cell.ownstate.Phase() != mosi.IO {
		t.Fatal("live handover became a recovery failure or regained rights", reply, cell.ownstate.Phase())
	}
}

func TestClusterObjectScopedSessionForwardsInvocationCancellation(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	session := scm.NewSession()
	object, _ := scm.ClusterObjectOf(session)
	view, _ := scm.ClusterObjectValue(object, context.Background(), scm.ClusterRequest{Method: "view"})
	ref, err := owner.exportObject(source, object, view)
	if err != nil {
		t.Fatal(err)
	}
	remote, err := client.importObject(context.Background(), peer, ref)
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	ownerFinished := make(chan struct{})
	const callback = "test_cluster_scoped_cancel"
	// This native fixture stands at the nested scan boundary: its supplied
	// participant must carry the originating invocation's cancellation signal.
	scm.Declare(&scm.Globalenv, &scm.Declaration{Name: callback, Fn: func(args ...scm.Scmer) scm.Scmer {
		ctx := scanQueryContext(scmerToTxContext(args[0]), context.Background())
		close(entered)
		defer close(ownerFinished)
		<-ctx.Done()
		panic(ctx.Err())
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Params: []*scm.TypeDescriptor{{Kind: "any"}}, Return: &scm.TypeDescriptor{Kind: "any"}}})
	producer := scm.Eval(scm.Read(t.Name(), "(lambda (tx) ("+callback+" tx))"), &scm.Globalenv)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ss := &scm.SessionState{}
	seq := ss.BeginQuery("Cluster", "scoped cancellation source")
	ss.SetQueryContext(seq, ctx)
	tx := NewTxContext(TxCursorStability)
	tx.beginQuery(ss, seq, "scoped cancellation source")
	defer tx.endQuery(seq)
	finished := make(chan any, 1)
	go func() {
		defer func() { finished <- recover() }()
		scm.Apply(remote, scm.NewString("get_or_compute_scoped"), scm.NewAny(tx), scm.NewString("nested"), scm.NewAny(tx), producer)
	}()
	select {
	case <-entered:
	case failure := <-finished:
		t.Fatalf("callback did not start: %v", failure)
	case <-time.After(time.Second):
		peer.close()
		t.Fatal("callback did not start")
	}
	cancel()
	select {
	case failure := <-finished:
		if failure == nil || !strings.Contains(fmt.Sprint(failure), "context canceled") {
			t.Fatal("scoped callback lost caller cancellation", failure)
		}
	case <-time.After(time.Second):
		peer.close()
		t.Fatal("caller did not unwind cancellation")
	}
	select {
	case <-ownerFinished:
	case <-time.After(time.Second):
		peer.close()
		t.Fatal("nested owner callback outlived originating cancellation")
	}
}

// The wire codec must honor object serialization too. This portable value has
// no URI type registration and deliberately cannot use a reference exporter.
type portableClusterTestValue struct{}

func (*portableClusterTestValue) ClusterObjectType() string { return "portable-wire-test" }
func (*portableClusterTestValue) Serialize(*scm.SerializationContext) (scm.Scmer, error) {
	return scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice([]scm.Scmer{scm.NewString("portable"), scm.NewInt(42)})}), nil
}
func (*portableClusterTestValue) HandleClusterRequest(context.Context, scm.ClusterRequest) (any, error) {
	return nil, fmt.Errorf("portable value has no remote owner")
}
func TestClusterObjectWireUsesMandatorySerializer(t *testing.T) {
	encoded, err := freezeScanValue(scm.NewAny(&portableClusterTestValue{}), &scm.SerializationContext{})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := encoded.reconstruct(scm.ProcedureDecode{})
	if err != nil {
		t.Fatal(err)
	}
	if !scm.Equal(decoded, scm.NewSlice([]scm.Scmer{scm.NewString("portable"), scm.NewInt(42)})) {
		t.Fatal("wire replaced object-owned serialization with a reference", decoded)
	}
}

//go:noinline
func importNeverCalledSubscan(t *testing.T, owner, client *nodeScanService, source, peer *mosiPeer, session scm.Scmer) scm.Scmer {
	bindings := (&scm.SerializationContext{}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
		return owner.exportObject(source, object, view)
	})
	body := scm.NewSlice([]scm.Scmer{scm.NewSymbol("if"), scm.NewSymbol("take"), session, scm.NewNil()})
	proc := scm.Eval(scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("take")}), body}), &scm.Globalenv)
	data, err := scm.MarshalProcedure(proc, bindings)
	if err != nil {
		t.Fatal(err)
	}
	received, err := scm.UnmarshalProcedure(data, scm.ProcedureDecode{Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) {
		return client.importObject(context.Background(), peer, ref)
	}})
	if err != nil {
		t.Fatal(err)
	}
	return received
}
func TestClusterObjectDeduplicatesOverlappingEmptySubscansAndGC(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	session := scm.NewSession()
	first := importNeverCalledSubscan(t, owner, client, source, peer, session)
	second := importNeverCalledSubscan(t, owner, client, source, peer, session)
	check := func() {
		t.Helper()
		r := owner.objectRegistry()
		r.mu.Lock()
		defer r.mu.Unlock()
		if len(r.entries) != 1 {
			t.Fatal("expected one owner registration", len(r.entries))
		}
		for _, entry := range r.entries {
			if len(entry.leases) != 1 || len(entry.tokens) != 0 {
				t.Fatal("subscans did not join one materialized peer lease", len(entry.leases), len(entry.tokens))
			}
		}
	}
	// No callback has run: import MUST already own a real Go peer handle.
	check()
	runtime.KeepAlive(first)
	first = scm.NewNil()
	for range 3 {
		runtime.GC()
	}
	check()
	runtime.KeepAlive(second)
	second = scm.NewNil()
	waitObjectGC(t, func() bool { return objectEntryCount(owner) == 0 })
}

func TestClusterObjectPendingTransferSurvivesPriorLeaseRelease(t *testing.T) {
	owner := &nodeScanService{id: uuid.NewString()}
	defer owner.stopObjects()
	peer := &mosiPeer{objectPeer: clusterObjectPeer{uuid.NewString(), uuid.NewString()}}
	object, _ := scm.ClusterObjectOf(scm.NewSession())
	view, _ := scm.ClusterObjectValue(object, context.Background(), scm.ClusterRequest{Method: "view"})
	first, err := owner.exportObject(peer, object, view)
	if err != nil {
		t.Fatal(err)
	}
	s := &scanRPCSession{service: owner, peer: peer}
	firstLease := uuid.NewString()
	data, _ := json.Marshal(clusterObjectAcquisition{first, firstLease})
	if _, err := s.handleObject(context.Background(), "object.acquire", data); err != nil {
		t.Fatal(err)
	}
	pending, err := owner.exportObject(peer, object, view)
	if err != nil {
		t.Fatal(err)
	}
	old := first
	old.Token = firstLease
	if err := owner.releaseObject(peer, old); err != nil {
		t.Fatal(err)
	}
	if objectEntryCount(owner) != 1 {
		t.Fatal("old GC release dropped the in-flight serialization reservation")
	}
	secondLease := uuid.NewString()
	data, _ = json.Marshal(clusterObjectAcquisition{pending, secondLease})
	if _, err := s.handleObject(context.Background(), "object.acquire", data); err != nil {
		t.Fatal(err)
	}
	if err := owner.releaseObject(peer, old); err != nil {
		t.Fatal(err)
	}
	if objectEntryCount(owner) != 1 {
		t.Fatal("late old release revoked the new peer lifetime")
	}
	pending.Token = secondLease
	if err := owner.releaseObject(peer, pending); err != nil {
		t.Fatal(err)
	}
	if objectEntryCount(owner) != 0 {
		t.Fatal("last lease remained registered")
	}
}

// Model the valid cancellation window in which Done is closed but an AfterFunc
// callback has not started yet. Stopping the hook must not lose remote cancel.
type delayedClusterCancelContext struct {
	context.Context
	done       chan struct{}
	registered chan struct{}
}

func (c *delayedClusterCancelContext) Done() <-chan struct{} { return c.done }
func (c *delayedClusterCancelContext) Err() error {
	select {
	case <-c.done:
		return context.Canceled
	default:
		return nil
	}
}
func (c *delayedClusterCancelContext) AfterFunc(func()) func() bool {
	close(c.registered)
	return func() bool { return true } // scheduler has not started the hook
}
func TestClusterCancellationQueuesMessageBeforeStoppingDeferredHook(t *testing.T) {
	for _, control := range []bool{true, false} {
		t.Run(fmt.Sprint("control=", control), func(t *testing.T) {
			ctx := &delayedClusterCancelContext{context.Background(), make(chan struct{}), make(chan struct{})}
			service := &nodeScanService{}
			peer := &mosiPeer{scanService: service, done: make(chan struct{}), outgoing: make(chan mosiMessage, 4)}
			finished := make(chan any, 1)
			go func() {
				if control {
					finished <- service.control(ctx, peer, "fixture", nil, nil)
					return
				}
				defer func() { finished <- recover() }()
				executor := &peerScanExecutor{peer: peer, ctx: ctx}
				executor.dispatch(remoteScanTask{})
			}()
			request := <-peer.outgoing
			var envelope struct {
				Return string `json:"return"`
			}
			if err := json.Unmarshal(request.Data, &envelope); err != nil {
				t.Fatal(err)
			}
			<-ctx.registered
			close(ctx.done)
			select {
			case result := <-finished:
				if result != context.Canceled {
					t.Fatal("wrong cancellation result", result)
				}
			case <-time.After(time.Second):
				t.Fatal("operation did not unwind")
			}
			select {
			case message := <-peer.outgoing:
				var id string
				if message.Method != "scan.cancel" || json.Unmarshal(message.Data, &id) != nil || id != envelope.Return {
					t.Fatal("wrong remote cancellation", message)
				}
			default:
				t.Fatal("Done won select and deferred stop suppressed the remote cancellation")
			}
		})
	}
}

func TestClusterObjectCatalogSerializationNeedsNoExportContext(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_catalog_serialization", 1)
	source := scm.SerializeToString(NewTableScmer(tbl), &scm.Globalenv)
	if source != tbl.String() {
		t.Fatal("network reconstruction changed local catalog text/plan identity", source, tbl.String())
	}
	value := scm.Eval(scm.Read(t.Name(), source), &scm.Globalenv)
	if TableFromScmer(value) != tbl {
		t.Fatal("portable catalog constructor changed table identity", source)
	}
}

func TestClusterObjectReleaseRetriesWithoutAnotherGCOrReconnect(t *testing.T) {
	identity := clusterObjectPeer{ID: "memcp://" + uuid.NewString(), Boot: uuid.NewString()}
	peer := &mosiPeer{objectPeer: identity, done: make(chan struct{}), outgoing: make(chan mosiMessage, 1)}
	service := &nodeScanService{peers: map[*mosiPeer]bool{peer: true}}
	registry := service.objectRegistry()
	defer service.stopObjects()
	registry.mu.Lock()
	registry.releases = append(registry.releases, clusterObjectRelease{peer: identity, reference: scm.ClusterReference{URI: "http://owner/cluster/" + uuid.NewString()}})
	registry.mu.Unlock()
	registry.wake <- struct{}{}
	for attempt := range 2 {
		select {
		case message := <-peer.outgoing:
			var request struct {
				Return string `json:"return"`
				Method string `json:"method"`
			}
			if err := json.Unmarshal(message.Data, &request); err != nil || request.Method != "object.release" {
				t.Fatalf("unexpected release request: %+v, %v", request, err)
			}
			entry, ok := service.returns.Load(request.Return)
			if !ok {
				t.Fatal("release has no completion collector")
			}
			completion := scanCompletion{}
			if attempt == 0 {
				completion.Panic = &scanPanic{Message: "temporary connection failure"}
			}
			deliverTestCompletion(t, entry, completion)
		case <-time.After(3 * time.Second):
			t.Fatal("release was not retried without another GC or reconnect", attempt)
		}
	}
}

func TestClusterObjectRegistryConcurrentInitializationAndWake(t *testing.T) {
	service := &nodeScanService{}
	defer service.stopObjects()
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(9)
	go func() { defer workers.Done(); <-start; service.objectRegistry() }()
	for i := 0; i < 8; i++ {
		go func() { defer workers.Done(); <-start; service.wakeObjectReleases() }()
	}
	close(start)
	workers.Wait()
	registry := service.objectRegistry()
	if registry.wake == nil || registry.stop == nil || registry.entries == nil || registry.identities == nil {
		t.Fatal("registry publication is incomplete")
	}
}

// A paged result is a consumer of the physical input too. Rebuild forwarding
// must not make maintenance delete that old generation while the peer still
// owns the registered run; after release, ordinary Go GC retires it.
func prepareRegisteredOrderedRun(t *testing.T, owner *nodeScanService, peer *mosiPeer, tbl *table) (scm.ClusterReference, weak.Pointer[storageShard]) {
	t.Helper()
	old := tbl.ActiveShards()[0]
	_, direction := persistableTestOrder(false)
	run := old.prepareOrderedRun(shardScanInvocation{condition: scanJoinTrue(), order: &shardOrderInvocation{columns: []scm.Scmer{scm.NewString("id")}, directions: []func(...scm.Scmer) scm.Scmer{direction}, projected: []string{"id"}, limit: -1}})
	view, err := scm.ClusterObjectValue(run, context.Background(), scm.ClusterRequest{Method: "view"})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := owner.exportObject(peer, run, view)
	if err != nil {
		t.Fatal(err)
	}
	RebuildTable(tbl, true, false)
	if tbl.ActiveShards()[0].uuid == old.uuid {
		t.Fatal("rebuild retained UUID")
	}
	old.RemoveFromDisk()
	return ref, weak.Make(old)
}
func TestClusterObjectOrderedRunRetainsHistoricalGeneration(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tordered_run_gc", 4)
	RebuildTable(tbl, true, false)
	_, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	peer := objectOwnerPeer(t, owner, client)
	ref, old := prepareRegisteredOrderedRun(t, owner, peer, tbl)
	runtime.GC()
	if !historicalObjectAlive(old) {
		t.Fatal("registered ordered run lost its physical input")
	}
	if len(registeredClusterShards(tbl.schema.Name)) == 0 {
		t.Fatal("ordered input absent from maintenance snapshot")
	}
	owner.cancelObjectOffers(peer, []scm.ClusterReference{ref})
	waitObjectGC(t, func() bool { return !historicalObjectAlive(old) })
}

func TestClusterObjectDictionaryCallbacksPrepareCompleteSessionView(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	session := scm.NewSession()
	scm.Apply(session, scm.NewString("a"), scm.NewInt(7))
	scm.Apply(session, scm.NewString("b"), scm.NewInt(8))
	env := &scm.Env{Vars: scm.Vars{scm.Symbol("session"): session}, Outer: &scm.Globalenv}
	var offers []scm.ClusterReference
	bindings := (&scm.SerializationContext{}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
		ref, err := owner.exportObject(source, object, view)
		if err == nil {
			offers = append(offers, ref)
		}
		return ref, err
	})
	defer func() { owner.cancelObjectOffers(source, offers) }()
	// An earlier callback demanded only a; the data dictionary also contains an
	// alias of the session and a callback demanding b, before any receiver row.
	first := scm.Eval(scm.Read(t.Name(), `(lambda () (session "a"))`), env)
	if _, err := scm.MarshalProcedure(first, bindings); err != nil {
		t.Fatal(err)
	}
	dict := scm.NewFastDictValue(64)
	dict.Set(scm.NewString("session"), session, nil)
	dict.Set(scm.NewString("callback"), scm.Eval(scm.Read(t.Name(), `(lambda () (session "b"))`), env), nil)
	for i := 0; i < 64; i++ {
		dict.Set(scm.NewInt(int64(i)), scm.NewNil(), nil)
	}
	data, err := freezeScanValue(scm.NewFastDict(dict), bindings)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := data.reconstruct(scm.ProcedureDecode{Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) {
		return client.importObject(context.Background(), peer, ref)
	}})
	if err != nil {
		t.Fatal(err)
	}
	callback, ok := restored.FastDict().Get(scm.NewString("callback"))
	if !ok || scm.Apply(callback).Int() != 8 {
		t.Fatal("nested callback lost its session binding", callback)
	}
}
