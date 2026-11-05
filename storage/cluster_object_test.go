/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "weak"
import "time"
import "sync"
import "bytes"
import "context"
import "strings"
import "runtime"
import "testing"
import "reflect"
import "sync/atomic"
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
	first, err := owner.exportObject(peer, object, view, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := owner.exportObject(peer, object, view, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.URI != second.URI || first.Token == second.Token {
		t.Fatal("exports did not share owner identity with independent handles")
	}
	// Two successive Go lifetimes of one peer may overlap while old GC delivery
	// is pending. They need generation IDs, not a boolean peer membership.
	sessionRPC := &scanRPCSession{service: owner, peer: peer}
	for _, ref := range []scm.ClusterReference{first, second} {
		if _, err := sessionRPC.resolveObject(ref); err != nil {
			t.Fatal("sender did not pregrant its reference", err)
		}
	}

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
	ref, err := owner.exportObject(peer, old, view, nil)
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
	if err := client.control(context.Background(), peer, "scan.execute", task, &reduced, true); err != nil {
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

func TestClusterCallbackRejectsCallsAfterExportingQueryEnds(t *testing.T) {
	createDurabilityTestTable(t, "t_cluster_callback_query", 1)
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	tx := NewTxContext(TxCursorStability)
	tx.beginQuery(nil, 7, "callback owner")
	defer tx.endQuery(7)
	var calls atomic.Int32
	callback := scm.NewClusterCallback(context.Background(), func(...scm.Scmer) scm.Scmer {
		calls.Add(1)
		return scm.NewBool(true)
	}, nil)
	env := &scm.Env{Vars: scm.Vars{scm.Symbol("output"): callback}, Outer: &scm.Globalenv}
	proc := scm.Eval(scm.Read(t.Name(), `(lambda () (output))`, true), env)
	bindings := (&scm.SerializationContext{Invocation: scm.NewAny(tx)}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
		return owner.exportObject(source, object, view, nil)
	})
	bindings = owner.handleBindings(bindings, source)
	data, err := scm.MarshalProcedure(proc, bindings)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	remote, err := scm.UnmarshalProcedure(data, scm.ProcedureDecode{ResolveHandle: client.handleResolver(ctx, peer, scm.NewNil()), Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) {
		return client.importObject(context.Background(), peer, ref)
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !scm.Apply(remote).Bool() || calls.Load() != 1 {
		t.Fatal("live callback did not execute")
	}
	tx.endQuery(7)
	remaining := 0
	owner.handles.Range(func(_, _ any) bool { remaining++; return true })
	if remaining != 0 {
		t.Fatalf("finished query retained %d handle slots", remaining)
	}
	cancel()
	var failure any
	func() {
		defer func() { failure = recover() }()
		scm.Apply(remote)
	}()
	if failure == nil || calls.Load() != 1 {
		t.Fatal("retained remote handle accessed completed query state")
	}
	runtime.KeepAlive(remote)
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
	proc := scm.Eval(scm.Read(t.Name(), `(lambda () (session "x"))`, true), environment)
	bindings := (&scm.SerializationContext{}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
		return owner.exportObject(source, object, view, nil)
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
	if !p.dispatchPacket(&mosiPacket{ID: "stream", Kind: "begin"}, 1, time.Time{}) {
		t.Fatal("dispatch rejected")
	}
	<-entered
	if !p.dispatchPacket(&mosiPacket{ID: "stream", Kind: "end"}, 1, time.Time{}) || !p.dispatchPacket(&mosiPacket{ID: "other", Kind: "grant"}, 1, time.Time{}) {
		t.Fatal("dispatch rejected")
	}
	select {
	case <-independent:
	case <-time.After(time.Second):
		t.Fatal("blocked transfer prevented unrelated message execution")
	}
	close(resume)
	p.packets.Wait()
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
	ref, err := owner.exportObject(source, object, view, nil)
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
	factory := scm.Eval(scm.Read(t.Name(), `(lambda () (+ (session "x") 3))`, true), env)
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
	ref, err := owner.exportObject(source, overlay, view, nil)
	if err != nil {
		t.Fatal(err)
	}
	value, err := client.importObject(context.Background(), peer, ref)
	if err != nil {
		t.Fatal(err)
	}
	imported := value.Any().(*storageShardTransaction)
	if imported.shardUUID != shard.uuid || imported.clusterTxID != overlay.clusterTxID || imported.MOSIState() != mosi.S {
		t.Fatal("reference lost physical identity or acquired write authority")
	}
	if imported.DeleteMask.Count() != 0 || !overlay.DeleteMask.Get(1) {
		t.Fatal("routing view duplicated private state instead of retaining it on its native shard")
	}
	if _, err := scm.ClusterObjectValue(imported, context.Background(), scm.ClusterRequest{Method: "commit"}); err == nil {
		t.Fatal("read overlay acquired coordinator mutation rights")
	}
	runtime.KeepAlive(value)
}

func TestClusterParticipantDecisionJoinsNativeInvocations(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_cluster_tx_join", 4)
	worker := NewTxContext(TxACID)
	worker.AddToDeleteMask(tbl.ActiveShards()[0], 1)
	worker.mu.Lock()
	participant := worker.shardTransactions()[tbl.ActiveShards()[0]]
	participant.decisionToken = "test-decision"
	worker.mu.Unlock()
	invocation := worker.newInvocation()
	invocation.remoteObject = &scm.ClusterHandle{}
	invocation.beginQuery(nil, 8, "in-flight mutation")
	defer invocation.endQuery(8)
	finished := make(chan error, 1)
	go func() {
		_, err := participant.executeClusterRequest(context.Background(), scm.ClusterRequest{Method: "rollback", Arguments: []scm.Scmer{scm.NewString(participant.decisionToken)}})
		finished <- err
	}()
	select {
	case err := <-finished:
		t.Fatalf("decision returned before invocation finished: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	invocation.endQuery(8)
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("native transaction decision did not finish after joining")
	}
	if worker.State != TxAborted || tbl.ActiveShards()[0].activeTransactions.Load() != 0 {
		t.Fatal("joined rollback left a native transaction active")
	}
}

func TestClusterFailedParticipantEnlistmentReleasesNativeGeneration(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_cluster_enlist_failure", 4)
	shard := tbl.ActiveShards()[0]
	tx := NewTxContext(TxACID)
	defer tx.Rollback()
	tx.coordinator = &scm.ClusterHandle{Call: func(context.Context, scm.ClusterRequest) (scm.Scmer, error) {
		return scm.NewNil(), fmt.Errorf("enlistment cancelled before mutation")
	}}
	var failure any
	func() {
		defer func() { failure = recover() }()
		shard.GetWrite(tx).Release()
	}()
	if fmt.Sprint(failure) != "enlistment cancelled before mutation" {
		t.Fatalf("unexpected enlistment failure: %v", failure)
	}
	if len(tx.shardTransactions()) != 0 || shard.activeTransactions.Load() != 0 {
		t.Fatal("failed participant retained native transaction authority")
	}
}

// Savepoint recovery must address the peer's native state once, including shards
// first touched after the savepoint, while preserving earlier statement effects.
func TestClusterNativeParticipantSavepointPreservesEarlierMutations(t *testing.T) {
	first, _ := createDurabilityTestTable(t, "t_cluster_sp_first", 4)
	second, _ := createDurabilityTestTable(t, "t_cluster_sp_second", 4)
	worker := NewTxContext(TxACID)
	defer worker.Rollback()
	coordinator := NewTxContext(TxACID)
	defer coordinator.Rollback()
	worker.AddToDeleteMask(first.ActiveShards()[0], 1)
	worker.mu.Lock()
	participant := worker.shardTransactions()[first.ActiveShards()[0]]
	participant.decisionToken = "test-native-participant"
	worker.mu.Unlock()
	imported := &storageShardTransaction{shardUUID: participant.shardUUID, decisionToken: participant.decisionToken}
	imported.remoteObject = &scm.ClusterHandle{
		Reference: scm.ClusterReference{Owner: "worker"},
		Call: func(ctx context.Context, request scm.ClusterRequest) (scm.Scmer, error) {
			value, err := participant.executeClusterRequest(ctx, request)
			if err != nil {
				return scm.NewNil(), err
			}
			return value.(scm.Scmer), nil
		},
	}
	coordinator.participants = map[uuid.UUID]*storageShardTransaction{participant.shardUUID: imported}
	sp := coordinator.CreateSavepoint()
	worker.AddToDeleteMask(first.ActiveShards()[0], 2)
	worker.AddToDeleteMask(second.ActiveShards()[0], 0)
	coordinator.RollbackToSavepoint(sp)
	if worker.IsVisible(first.ActiveShards()[0], 1) || !worker.IsVisible(first.ActiveShards()[0], 2) || !worker.IsVisible(second.ActiveShards()[0], 0) {
		t.Fatal("remote savepoint lost earlier effects or retained later shard effects")
	}
	if worker.State != TxActive {
		t.Fatal("savepoint rollback ended the native transaction")
	}
	// A newly enlisted owner rolls back to the empty native savepoint.
	coordinator.RollbackToSavepoint(Savepoint{})
	if !worker.IsVisible(first.ActiveShards()[0], 1) {
		t.Fatal("new participant retained a mutation after savepoint recovery")
	}
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
	ref, err := owner.exportObject(source, table, view, nil)
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
	pending, err := owner.exportObject(source, object, view, nil)
	if err != nil {
		t.Fatal(err)
	}
	live, err := owner.exportObject(source, object, view, nil)
	if err != nil {
		t.Fatal(err)
	}
	owner.commitObjectOffers(source, []scm.ClusterReference{live})
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
	ref, err := owner.exportObject(source, object, view, nil)
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
	if !p.dispatchPacket(&mosiPacket{ID: "schema", Resource: mosiResource{Database: "dependent"}}, 1, time.Time{}) {
		t.Fatal("schema dispatch failed")
	}
	<-entered
	if !p.dispatchPacket(&mosiPacket{ID: "data", Resource: mosiResource{Database: "dependent", Table: "t", Shard: uuid.NewString()}}, 1, time.Time{}) ||
		!p.dispatchPacket(&mosiPacket{ID: "independent", Resource: mosiResource{Database: "independent"}}, 1, time.Time{}) {
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
	p.packets.Wait()
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
	ref, err := owner.exportObject(source, object, view, nil)
	if err != nil {
		t.Fatal(err)
	}
	id := client.nextHandle.Add(1)
	receiver := scm.NewResultReceiver()
	client.handles.Store(id, &handleSlot{peer: peer.objectPeer, receiver: &controlResultHandle{service: client, peer: peer, receiver: receiver}})
	defer client.handles.Delete(id)
	completion, _ := marshalNode(scanCompletion{Offers: []scm.ClusterReference{ref}})
	s := &scanRPCSession{service: client, peer: peer}
	if err := s.handleValue(id, schemeValue(completion)); err != nil {
		t.Fatal(err)
	}
	// Cancellation after delivery but before collection must release the offer.
	receiver.Close()
	abandoned := receiver.Wait().Any().(scanCompletion)
	client.discardObjectOffers(peer, abandoned.Offers)
	waitObjectGC(t, func() bool { return objectEntryCount(owner) == 0 })
	// A late duplicate is harmless and cannot replace the accepted result.
	completion, _ = marshalNode(scanCompletion{Data: []byte("late"), Offers: []scm.ClusterReference{ref}})
	if err := s.handleValue(id, schemeValue(completion)); err != nil {
		t.Fatal(err)
	}
	if len(receiver.Wait().Any().(scanCompletion).Data) != 0 {
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
	ref, err := owner.exportObject(peer, old, view, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Drop one current row so the successor no longer owns its blob. The old
	// physical generation's column and manifest still need that blob on disk.
	old.UpdateFunction(0, false, false, nil, nil)()
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
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
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
	message := takeMosi(peer.outgoing)
	var request struct {
		Return string `json:"return"`
	}
	if err := decodeTestInvocation(message.Channel, message.Payload, &request); err != nil {
		t.Fatal(err)
	}
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
		record.Handoff = remote.ID
		return true
	})
	data, _ := marshalNode(replicaRecord{Resource: resource, Owner: self.ID, Epoch: 9, State: "IO"})
	entry, ok := service.handles.Load(controlReturnSlot(request.Return))
	if !ok {
		t.Fatal("missing discovery return collector")
	}
	deliverTestCompletion(t, entry, scanCompletion{Data: data})
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
	ref, err := owner.exportObject(source, object, view, nil)
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
	producer := scm.Eval(scm.Read(t.Name(), "(lambda (tx) ("+callback+" tx))", true), &scm.Globalenv)
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
		return owner.exportObject(source, object, view, nil)
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
	check := func(expected int) {
		t.Helper()
		r := owner.objectRegistry()
		r.mu.Lock()
		defer r.mu.Unlock()
		if len(r.entries) != 1 {
			t.Fatal("expected one owner registration", len(r.entries))
		}
		for _, entry := range r.entries {
			if len(entry.leases) != expected {
				t.Fatal("independent transfer grants did not match live subscans", len(entry.leases), len(entry.tokens))
			}
		}
	}
	// No callback has run: import MUST already own a real Go peer handle.
	check(2)
	runtime.KeepAlive(first)
	first = scm.NewNil()
	for range 3 {
		runtime.GC()
	}
	waitObjectGC(t, func() bool {
		r := owner.objectRegistry()
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, entry := range r.entries {
			return len(entry.leases) == 1
		}
		return false
	})
	check(1)
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
	first, err := owner.exportObject(peer, object, view, nil)
	if err != nil {
		t.Fatal(err)
	}
	owner.commitObjectOffers(peer, []scm.ClusterReference{first})
	pending, err := owner.exportObject(peer, object, view, nil)
	if err != nil {
		t.Fatal(err)
	}
	old := first
	old.Token = first.Token
	if err := owner.releaseObject(peer, old); err != nil {
		t.Fatal(err)
	}
	if objectEntryCount(owner) != 1 {
		t.Fatal("old GC release dropped the in-flight serialization reservation")
	}
	owner.commitObjectOffers(peer, []scm.ClusterReference{pending})
	if err := owner.releaseObject(peer, old); err != nil {
		t.Fatal(err)
	}
	if objectEntryCount(owner) != 1 {
		t.Fatal("late old release revoked the new peer lifetime")
	}
	if err := owner.releaseObject(peer, pending); err != nil {
		t.Fatal(err)
	}
	if objectEntryCount(owner) != 0 {
		t.Fatal("last lease remained registered")
	}
}

// Once dispatch is accepted, cancellation must be enqueued before the
// collector returns, including when cancellation wins its first select.
func TestClusterCancellationQueuesMessageBeforeCollectorReturns(t *testing.T) {
	for _, control := range []bool{true, false} {
		t.Run(fmt.Sprint("control=", control), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			service := &nodeScanService{}
			peer := &mosiPeer{scanService: service, done: make(chan struct{}), outgoing: newMosiMailbox()}
			finished := make(chan any, 1)
			go func() {
				if control {
					finished <- service.control(ctx, peer, "fixture", nil, nil, true)
					return
				}
				defer func() { finished <- recover() }()
				executor := &peerScanExecutor{peer: peer, ctx: ctx}
				executor.dispatch(remoteScanTask{})
			}()
			request := takeMosi(peer.outgoing)
			address, _, _, err := decodeInvocation(request.Channel, request.Payload)
			if err != nil {
				t.Fatal(err)
			}
			cancel()
			select {
			case <-peer.outgoing.ready:
				message := takeMosi(peer.outgoing)
				var id string
				if message.Channel != "scan.cancel" || unmarshalNode(message.Payload, &id) != nil || id != address {
					t.Fatal("wrong remote cancellation", message)
				}
			case <-time.After(time.Second):
				t.Fatal("collector did not send cancellation")
			}
			select {
			case result := <-finished:
				t.Fatal("collector returned before remote completion", result)
			default:
			}
			// The existing return channel is also the join acknowledgement. No
			// cancelled invocation may release its captures before this delivery.
			entry, ok := service.handles.Load(controlReturnSlot(address))
			if !ok {
				t.Fatal("cancelled invocation dropped its return channel")
			}
			deliverTestCompletion(t, entry, scanCompletion{})
			select {
			case result := <-finished:
				if result != context.Canceled {
					t.Fatal("wrong cancellation result", result)
				}
			case <-time.After(time.Second):
				t.Fatal("operation did not unwind after remote completion")
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
	value := scm.Eval(scm.Read(t.Name(), source, true), &scm.Globalenv)
	if TableFromScmer(value) != tbl {
		t.Fatal("portable catalog constructor changed table identity", source)
	}
}

func TestClusterObjectReleaseRetriesWithoutAnotherGCOrReconnect(t *testing.T) {
	identity := clusterObjectPeer{ID: "memcp://" + uuid.NewString(), Boot: uuid.NewString()}
	peer := &mosiPeer{objectPeer: identity, done: make(chan struct{}), outgoing: newMosiMailbox()}
	service := &nodeScanService{peers: map[*mosiPeer]bool{peer: true}}
	registry := service.objectRegistry()
	defer service.stopObjects()
	wait := func(condition func() bool) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if condition() {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("release worker did not progress")
	}
	// A full publication window rejects immediately, without a remote ack.
	// Its release worker retains the work for a later independent retry.
	peer.outgoing.bytes.Store(32 << 20)
	registry.mu.Lock()
	registry.releases = append(registry.releases, clusterObjectRelease{peer: identity, reference: scm.ClusterReference{URI: "http://owner/cluster/" + uuid.NewString()}})
	registry.mu.Unlock()
	registry.wake <- struct{}{}
	// Observe a completed refusal, not the fleeting empty list between taking
	// work and restoring it. Publication no longer has a mutex to pause there.
	wait(func() bool {
		registry.mu.Lock()
		defer registry.mu.Unlock()
		return peer.outgoing.rejected.Load() > 0 && len(registry.releases) == 1
	})
	peer.outgoing.bytes.Store(0)
	select {
	case <-peer.outgoing.ready:
		message := takeMosi(peer.outgoing)
		if !strings.HasPrefix(message.Channel, "object/") {
			t.Fatal("release is not fire-and-forget", message.Channel)
		}
		if _, exists := service.handles.Load(""); exists {
			t.Fatal("release allocated a reply collector")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("release was not retried without another GC or reconnect")
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
	ref, err := owner.exportObject(peer, run, view, nil)
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
		ref, err := owner.exportObject(source, object, view, nil)
		if err == nil {
			offers = append(offers, ref)
		}
		return ref, err
	})
	defer func() { owner.cancelObjectOffers(source, offers) }()
	// An earlier callback demanded only a; the data dictionary also contains an
	// alias of the session and a callback demanding b, before any receiver row.
	first := scm.Eval(scm.Read(t.Name(), `(lambda () (session "a"))`, true), env)
	if _, err := scm.MarshalProcedure(first, bindings); err != nil {
		t.Fatal(err)
	}
	dict := scm.NewFastDictValue(64)
	dict.Set(scm.NewString("session"), session, nil)
	dict.Set(scm.NewString("callback"), scm.Eval(scm.Read(t.Name(), `(lambda () (session "b"))`, true), env), nil)
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

// A transferred overlay is already enlisted before MOSI publishes its grant.
// Its first local writer must retain the capability held by older endpoints.
func TestClusterTransferredParticipantKeepsEnlistedCapability(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_cluster_enlisted_capability", 4)
	worker := NewTxContext(TxACID)
	defer worker.Rollback()
	shard := tbl.ActiveShards()[0]
	worker.AddToDeleteMask(shard, 1)
	participant := worker.getShardTx(shard)
	const capability = "already-enlisted-by-live-handover"
	participant.decisionToken = capability
	worker.coordinator = &scm.ClusterHandle{Call: func(_ context.Context, request scm.ClusterRequest) (scm.Scmer, error) {
		if request.Method != "enlist" || request.Arguments[1].String() != capability {
			t.Errorf("first writer replaced the transferred capability: %v", request.Method)
		}
		return scm.NewBool(true), nil
	}}
	if err := participant.registerParticipant(context.Background()); err != nil {
		t.Fatal(err)
	}
	// A stale endpoint still carries the capability acknowledged at handover.
	endpoint := &storageShardTransaction{}
	endpoint.forwarded.Store(participant)
	for _, invalid := range []string{"", "foreign-capability"} {
		if _, err := endpoint.executeClusterRequest(context.Background(), scm.ClusterRequest{Method: "savepoint", Arguments: []scm.Scmer{scm.NewString(invalid)}}); err == nil {
			t.Fatal("forwarding admitted an unenlisted capability")
		}
	}
	if _, err := endpoint.executeClusterRequest(context.Background(), scm.ClusterRequest{Method: "savepoint", Arguments: []scm.Scmer{scm.NewString(capability)}}); err != nil {
		t.Fatalf("first write invalidated the enlisted endpoint: %v", err)
	}
}

//go:noinline
func prepareSharedLiteral(t *testing.T, owner, client *nodeScanService, peer *mosiPeer) weak.Pointer[scm.FastDict] {
	t.Helper()
	source := objectOwnerPeer(t, owner, client)
	bindings := (&scm.SerializationContext{}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
		return owner.exportObject(source, object, view, nil)
	})
	dict := scm.NewFastDictValue(512)
	for i := 0; i < 512; i++ {
		dict.Set(scm.NewInt(int64(i)), scm.NewInt(int64(i+7)), nil)
	}
	first, err := freezeScanValue(scm.NewFastDict(dict), bindings)
	if err != nil {
		t.Fatal(err)
	}
	second, err := freezeScanValue(scm.NewFastDict(dict), bindings)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) > 1024 || len(second) > 1024 {
		t.Fatal("dictionary was inlined per shard", len(first), len(second))
	}
	// This test reconstructs wire values directly, outside the socket reader.
	// Wait for the preceding declaration to arrive; real tasks follow it on
	// the same stream and therefore cannot overtake its inline publication.
	for _, reference := range serializedReferences(first) {
		if len(reference.View) != 0 {
			continue
		}
		deadline := time.Now().Add(time.Second)
		for {
			if _, arrived := peer.objectViews.Load(reference.URI); arrived {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("immutable declaration did not arrive before direct reconstruction")
			}
			time.Sleep(time.Millisecond)
		}
	}
	// Mutation after export must not change the frozen statement snapshot.
	dict.Set(scm.NewInt(0), scm.NewInt(900), nil)
	decode := scm.ProcedureDecode{Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) {
		return client.importObject(context.Background(), peer, ref)
	}}
	type restored struct {
		value scm.Scmer
		err   error
	}
	results := make(chan restored, 16)
	wires := []schemeValue{first, second}
	for len(wires) < 16 {
		wire, err := freezeScanValue(scm.NewFastDict(dict), bindings)
		if err != nil {
			t.Fatal(err)
		}
		wires = append(wires, wire)
	}
	for _, wire := range wires {
		go func(wire schemeValue) { value, err := wire.reconstruct(decode); results <- restored{value, err} }(wire)
	}
	values := make([]scm.Scmer, 0, len(wires))
	for range wires {
		got := <-results
		if got.err != nil {
			t.Fatal(got.err)
		}
		if len(values) > 0 && values[0].FastDict() != got.value.FastDict() {
			t.Fatal("concurrent imports reconstructed independent dictionaries")
		}
		values = append(values, got.value)
	}
	a, b := values[0], values[1]
	if a.FastDict() != b.FastDict() {
		t.Fatal("same lease reconstructed the dictionary twice")
	}
	if !scm.Equal(a, b) {
		t.Fatal("dictionary aliases changed")
	}
	env := &scm.Env{Vars: scm.Vars{scm.Symbol("captured"): a}, Outer: &scm.Globalenv}
	proc := scm.Eval(scm.Optimize(scm.Read(t.Name(), `(lambda (v) (set_assoc captured 0 v))`, false), env, nil), env)
	proc = scm.CompileJIT(proc, true)
	left, right := scm.Apply(proc, scm.NewInt(11)), scm.Apply(proc, scm.NewInt(12))
	original, _ := a.FastDict().Get(scm.NewInt(0))
	one, _ := left.FastDict().Get(scm.NewInt(0))
	two, _ := right.FastDict().Get(scm.NewInt(0))
	if original.Int() != 7 || one.Int() != 11 || two.Int() != 12 {
		t.Fatal("shared capture lost immutable update semantics", original, one, two)
	}
	// All keys and values must survive, including the pre-mutation value.
	for i, item := range a.FastDict().Pairs {
		if i%2 == 1 && item.Int() != a.FastDict().Pairs[i-1].Int()+7 {
			t.Fatal("snapshot changed", i, item)
		}
	}
	return weak.Make(a.FastDict())
}

func TestClusterImmutableCaptureSharesSnapshotAndReleasesWithGC(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	value := prepareSharedLiteral(t, owner, client, peer)
	waitObjectGC(t, func() bool { return value.Value() == nil && objectEntryCount(owner) == 0 })
}

func TestClusterPregrantedSnapshotImportsAfterDisconnect(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	session := scm.NewSession()
	scm.Apply(session, scm.NewString("key"), scm.NewInt(17))
	object, _ := scm.ClusterObjectOf(session)
	view, _ := scm.ClusterObjectValue(object, context.Background(), scm.ClusterRequest{Method: "view"})
	ref, err := owner.exportObject(source, object, view, nil)
	if err != nil {
		t.Fatal(err)
	}
	owner.commitObjectOffers(source, []scm.ClusterReference{ref})
	peer.close()
	peer.wait()
	value, err := client.importObject(context.Background(), peer, ref)
	if err != nil {
		t.Fatal("materialization waited for acquisition", err)
	}
	if got := scm.Apply(value, scm.NewString("key")); got.Int() != 17 {
		t.Fatal("pushed snapshot changed", got)
	}
	if objectEntryCount(owner) != 1 {
		t.Fatal("disconnect revoked the pregranted lifetime")
	}
	runtime.KeepAlive(value)
}

func TestClusterViewDeclarationsPublishOnceAndRespectBudget(t *testing.T) {
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox()}
	view := []byte(`"pushed constructor"`)
	if !peer.declareObjectView("owner/1", view) || !peer.declareObjectView("owner/1", view) {
		t.Fatal("declaration failed")
	}
	if peer.outgoing.len() != 1 {
		t.Fatal("known view was sent twice")
	}
	message := takeMosi(peer.outgoing)
	if message.Channel != "object.declare" {
		t.Fatal("declaration allocated a result channel")
	}
	receiver := &mosiPeer{}
	if err := receiver.publishObjectView(message.Channel, message.Payload); err != nil {
		t.Fatal(err)
	}
	if got, ok := receiver.objectViews.Load("owner/1"); !ok || string(got.([]byte)) != string(view) {
		t.Fatal("declaration was not published before task admission")
	}
	if peer.declareObjectView("too-large", make([]byte, objectViewBudget)) {
		t.Fatal("declaration bypassed byte budget")
	}
	if peer.outgoing.len() != 0 {
		t.Fatal("non-admitted view was partially queued")
	}
	peer.forgetObjectView("owner/1")
	message = takeMosi(peer.outgoing)
	if err := receiver.publishObjectView(message.Channel, message.Payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := receiver.objectViews.Load("owner/1"); ok {
		t.Fatal("retired source remained cached")
	}
}

func TestClusterCancelledReturnHandleDiscardsLateGrants(t *testing.T) {
	service := &nodeScanService{}
	defer service.stopObjects()
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox(), objectPeer: clusterObjectPeer{"owner", "boot"}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() { finished <- service.control(ctx, peer, "fixture", nil, nil, false) }()
	request := takeMosi(peer.outgoing)
	address, _, _, err := decodeInvocation(request.Channel, request.Payload)
	if err != nil {
		t.Fatal(err)
	}
	slot := controlReturnSlot(address)
	previous, ok := service.channels.Load(handleChannel(slot))
	if !ok {
		t.Fatal("missing original receiver")
	}
	cancel()
	if err := <-finished; err != context.Canceled {
		t.Fatal("wrong wait cancellation", err)
	}
	entry, ok := service.handles.Load(slot)
	if !ok {
		t.Fatal("cancelled waiter discarded its necessary result cleanup")
	}
	if _, abandoned := entry.(*handleSlot).receiver.(*abandonedControlHandle); !abandoned {
		t.Fatal("cancelled result retained query state")
	}
	if cancelMessage := takeMosi(peer.outgoing); cancelMessage.Channel != "scan.cancel" {
		t.Fatal("missing remote cancellation")
	}
	ref := scm.ClusterReference{URI: "http://owner/cluster/1", Owner: "owner", Incarnation: "boot", Token: "1", Type: "session"}
	payload, err := marshalNode(scanCompletion{Offers: []scm.ClusterReference{ref}})
	if err != nil {
		t.Fatal(err)
	}
	// The reader may have fetched the old receiver before cancellation
	// replaced it with the state-free disposer. Exercise that exact handoff.
	previous.(scm.ClusterRequestHandler).HandleClusterRequest(&channelDelivery{Context: context.Background(), peer: peer}, payload)
	peer.workers.Wait()
	if _, ok := service.handles.Load(slot); ok {
		t.Fatal("late result retained its cleanup slot")
	}
	message := takeMosi(peer.outgoing)
	var discarded []scm.ClusterReference
	if message.Channel != "object.discard" || unmarshalNode(message.Payload, &discarded) != nil || len(discarded) != 1 || discarded[0].Token != ref.Token {
		t.Fatal("abandoned result leaked its grant")
	}
}

func TestClusterDuplicateResultPreservesAcceptedGrant(t *testing.T) {
	service := &nodeScanService{id: "client"}
	defer service.stopObjects()
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox(), objectPeer: clusterObjectPeer{"owner", "boot"}}
	receiver := scm.NewResultReceiver()
	handler := &controlResultHandle{service: service, peer: peer, receiver: receiver}
	original := scm.ClusterReference{URI: "owner/1", Owner: "owner", Incarnation: "boot", Token: "1", Type: "session"}
	deliver := func(refs ...scm.ClusterReference) {
		if err := handler.receive(mustNodeMetadata(scanCompletion{Offers: refs})); err != nil {
			t.Fatal(err)
		}
	}
	deliver(original)
	deliver(original)
	if peer.outgoing.len() != 0 {
		t.Fatal("duplicate released the first result's still-unconsumed grant")
	}
	unexpected := original
	unexpected.Token = "2"
	deliver(original, unexpected)
	message := takeMosi(peer.outgoing)
	var discarded []scm.ClusterReference
	if message.Channel != "object.discard" || unmarshalNode(message.Payload, &discarded) != nil || len(discarded) != 1 || discarded[0].Token != "2" {
		t.Fatal("duplicate leaked a separate unused grant", discarded)
	}
	accepted := receiver.Wait().Any().(scanCompletion)
	if len(accepted.Offers) != 1 || accepted.Offers[0].Token != original.Token {
		t.Fatal("duplicate replaced the accepted result")
	}
}

func TestClusterImportedTransactionExportRetainsRoot(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	tx := NewTxContext(TxCursorStability)
	view, err := scm.ClusterObjectValue(tx.transactionState, context.Background(), scm.ClusterRequest{Method: "view"})
	if err != nil {
		t.Fatal(err)
	}
	root, err := owner.exportObject(source, tx.transactionState, view)
	if err != nil {
		t.Fatal(err)
	}
	// Separate services share this test process's native identity directory.
	// Construct the proxy held by a different process explicitly, so exporting
	// it exercises forwarding rather than resolving back to the native root.
	proxy := &transactionState{clusterID: tx.ClusterObjectID(), coordinator: &scm.ClusterHandle{Reference: root, View: view}}
	forwarded, err := client.exportObject(peer, proxy, view)
	if err != nil {
		t.Fatal(err)
	}
	if forwarded.URI != root.URI || forwarded.Owner != root.Owner || forwarded.Incarnation != root.Incarnation {
		t.Fatalf("imported transaction acquired a foreign root: %+v, want %+v", forwarded, root)
	}
	value, err := owner.importObject(context.Background(), source, forwarded)
	if err != nil || value.Any().(*TxContext).transactionState != tx.transactionState {
		t.Fatalf("returned root no longer resolves to its native transaction: %v", err)
	}
}

func TestClusterObjectReleaseReconnectKeepsServiceConnection(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	object, _ := scm.ClusterObjectOf(scm.NewSession())
	ref, err := owner.exportObject(source, object, scm.NewNil())
	if err != nil {
		t.Fatal(err)
	}
	identity := peer.objectPeer
	peer.close()
	peer.wait()
	if !client.deliverObjectRelease(clusterObjectRelease{peer: identity, reference: ref}) {
		t.Fatal("release could not reconnect to its owner")
	}
	if client.objectConnection(identity) == nil {
		t.Fatal("release closed a published connection before its queued message and other borrowers could finish")
	}
	deadline := time.Now().Add(time.Second)
	for objectEntryCount(owner) != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if objectEntryCount(owner) != 0 {
		t.Fatal("reconnected release did not reach its owner")
	}
}

func TestClusterDuplicateLinkDoesNotCancelBorrowedHandle(t *testing.T) {
	prior, client, endpoint := setupRemoteScanConnection(t)
	candidate, _, err := client.connect(context.Background(), strings.Replace(endpoint, "127.0.0.1", "localhost", 1))
	if err != nil {
		t.Fatal(err)
	}
	defer candidate.close()
	owner := nodeScans.Load()
	entered, release := make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	callback := newClusterReceiver(-1, func(ctx context.Context, _ scm.Scmer) (scm.Scmer, error) {
		close(entered)
		select {
		case <-release:
			return scm.NewInt(42), nil
		case <-ctx.Done():
			return scm.NewNil(), ctx.Err()
		}
	})
	defer callback.Close()
	defer unblock()
	ref, err := owner.exportObject(objectOwnerPeer(t, owner, client), callback, scm.NewNil())
	if err != nil {
		t.Fatal(err)
	}
	view, _ := freezeScanValue(scm.NewNil())
	ref.View = []byte(view)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		var reply clusterObjectReply
		err := client.control(ctx, candidate, "object.call", clusterObjectCall{Reference: ref, Method: "receive", Arguments: []schemeValue{view}}, &reply, true)
		if err == nil {
			result, decodeErr := reply.Value.reconstruct()
			if decodeErr != nil {
				err = decodeErr
			} else if result.Int() != 42 {
				err = fmt.Errorf("borrowed callback returned %v", result)
			}
		}
		finished <- err
	}()
	select {
	case <-entered:
	case err := <-finished:
		t.Fatal("callback was not admitted", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	member := clusterMember{ID: prior.objectPeer.ID, Boot: prior.objectPeer.Boot, Address: endpoint}
	cluster := &nodeCluster{service: client, members: map[string]clusterMember{member.ID: member}, links: map[string]*clusterLink{member.ID: {member: member, peer: prior}}}
	cluster.publishLocked()
	if got := cluster.installLink(&clusterLink{member: member, peer: candidate}); got != prior {
		t.Fatal("duplicate changed the primary routing connection")
	}
	unblock()
	if err := <-finished; err != nil {
		t.Fatal("routing publication cancelled another user's admitted callback", err)
	}
}

func TestClusterConnectReusesManagedEndpoint(t *testing.T) {
	peer, client, endpoint := setupRemoteScanConnection(t)
	next, id, err := client.connect(context.Background(), endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer next.close()
	if next != peer || id != peer.objectPeer.ID {
		t.Fatal("repeated setup allocated another connection for the same endpoint")
	}
}

func TestClusterForwardedOverlayExportNeedsNoOwnerRoundTrip(t *testing.T) {
	service := &nodeScanService{id: "relay"}
	defer service.stopObjects()
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox(), objectPeer: clusterObjectPeer{"target", "boot"}}
	object := &storageShardTransaction{remoteObject: &scm.ClusterHandle{
		Reference: scm.ClusterReference{Owner: "original", Incarnation: "old", Token: "lease", Type: "tx-shard"},
		Call: func(context.Context, scm.ClusterRequest) (scm.Scmer, error) {
			t.Fatal("export contacted original owner")
			return scm.NewNil(), nil
		},
	}}
	ref, err := service.exportObject(peer, object, scm.NewNil(), nil)
	if err != nil || ref.Owner != service.authNodeID() || ref.Token == "" {
		t.Fatal("failed to pregrant forwarding endpoint", ref, err)
	}
	registry := service.objectRegistry()
	registry.mu.Lock()
	retained := registry.entries[ref.URI].object
	registry.mu.Unlock()
	if retained != object || object.remoteObject == nil {
		t.Fatal("forwarder lost the original import's Go lifetime")
	}
	if peer.outgoing.len() != 0 {
		t.Fatal("export unexpectedly required remote coordination")
	}
}

func TestClusterResultReceiverGCReleasesHandleAndDisconnectHook(t *testing.T) {
	service := &nodeScanService{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	peer := &mosiPeer{ctx: ctx, done: make(chan struct{}), objectPeer: clusterObjectPeer{"peer", "boot"}}
	abandon := func() uint64 {
		receiver := scm.NewResultReceiver()
		bindings := service.handleBindings(&scm.SerializationContext{}, peer)
		expression, err := bindings.HandleExpression(receiver)
		if err != nil {
			t.Fatal(err)
		}
		slot := uint64(expression.Slice()[1].Int())
		runtime.KeepAlive(receiver)
		return slot
	}
	slot := abandon()
	waitObjectGC(t, func() bool {
		_, handle := service.handles.Load(slot)
		_, channel := service.channels.Load(handleChannel(slot))
		return !handle && !channel
	})
	cancel() // the stopped disconnect hook cannot resurrect the receiver
}

func TestClusterDeregisteredChannelsDiscardLateNativeGrants(t *testing.T) {
	for _, binaryCompletion := range []bool{false, true} {
		t.Run(fmt.Sprint(binaryCompletion), func(t *testing.T) {
			_, client, _ := setupRemoteScanConnection(t)
			owner := nodeScans.Load()
			source := objectOwnerPeer(t, owner, client)
			object, _ := scm.ClusterObjectOf(scm.NewSession())
			view, _ := scm.ClusterObjectValue(object, context.Background(), scm.ClusterRequest{Method: "view"})
			ref, err := owner.exportObject(source, object, view, nil)
			if err != nil {
				t.Fatal(err)
			}
			owner.commitObjectOffers(source, []scm.ClusterReference{ref})
			var payload []byte
			if binaryCompletion {
				payload = mustNodeMetadata(scanCompletion{Offers: []scm.ClusterReference{ref}})
			} else {
				frozen, err := freezeScanValue(scm.NewAny(object), (&scm.SerializationContext{}).WithExport(func(scm.ClusterObject, scm.Scmer) (scm.ClusterReference, error) { return ref, nil }))
				if err != nil {
					t.Fatal(err)
				}
				payload = frozen
			}
			if err := source.send(context.Background(), mosiMessage{Channel: handleChannel(client.allocateHandleSlot()), Payload: payload}); err != nil {
				t.Fatal(err)
			}
			waitObjectGC(t, func() bool { return objectEntryCount(owner) == 0 })
			client.handles.Range(func(key, _ any) bool { t.Error("late delivery retained a handler", key); return true })
		})
	}
}

func TestClusterForwardedTransactionRootKeepsOriginalCoordinator(t *testing.T) {
	service := &nodeScanService{id: "relay"}
	defer service.stopObjects()
	peer := &mosiPeer{objectPeer: clusterObjectPeer{"target", "boot"}}
	root := scm.ClusterReference{URI: "http://original/cluster/transactions/17", Owner: "original", Incarnation: "original-boot", Type: "transaction", View: []byte("root view")}
	state := &transactionState{coordinator: &scm.ClusterHandle{Reference: root}}
	ref, err := service.exportObject(peer, state, scm.NewNil(), nil)
	if err != nil || !reflect.DeepEqual(ref, root) {
		t.Fatal("MOSI export introduced a transaction-root relay", ref, err)
	}
	if objectEntryCount(service) != 0 {
		t.Fatal("routing root allocated a lifetime registration")
	}
}

func TestClusterObjectCallUsesRegisteredAddressWithoutRepeatingReference(t *testing.T) {
	service := &nodeScanService{}
	defer service.stopObjects()
	peer := &mosiPeer{done: make(chan struct{}), outgoing: newMosiMailbox(), objectPeer: clusterObjectPeer{"owner", "boot"}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	payload, err := scm.EncodeClusterRequest(scm.ClusterRequest{Method: "ack", Arguments: []scm.Scmer{scm.NewInt(64), scm.NewInt(128)}})
	if err != nil {
		t.Fatal(err)
	}
	reference := scm.ClusterReference{URI: "http://owner/cluster/7", Owner: "owner", Incarnation: "boot", Token: "grant", Type: "ordered-run", View: []byte(strings.Repeat("immutable-view", 1024))}
	finished := make(chan error, 1)
	go func() {
		finished <- service.control(ctx, peer, "object.call", clusterObjectCall{Reference: reference, Payload: payload}, nil, false)
	}()
	message := takeMosi(peer.outgoing)
	if message.Channel != "object/7" || len(message.Payload) == 0 || message.Payload[0] != handleValue {
		t.Fatal("object did not use its registered receiver", message.Channel)
	}
	var request struct {
		Return string
		Data   []byte
	}
	if err := decodeTestInvocation("object.call", message.Payload[1:], &request); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(request.Data, payload) {
		t.Fatal("invocation repeated its URI/type/immutable view instead of raw operation bytes")
	}
	cancel()
	if err := <-finished; err != context.Canceled {
		t.Fatal(err)
	}
	service.deleteHandle(controlReturnSlot(request.Return))
}

//go:noinline
func exportCollectibleTransactionChannel(t *testing.T, service *nodeScanService) (string, weak.Pointer[transactionState]) {
	t.Helper()
	tx := NewTxContext(TxCursorStability)
	view, err := tx.ClusterView(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ref, err := service.exportObject(&mosiPeer{objectPeer: clusterObjectPeer{"reader", "boot"}}, tx.transactionState, view, nil)
	if err != nil {
		t.Fatal(err)
	}
	channel := objectChannel(ref.URI)
	if _, exists := service.channels.Load(channel); !exists {
		t.Fatal("transaction root did not register its raw receiver")
	}
	return channel, weak.Make(tx.transactionState)
}

func TestClusterTransactionRootChannelFollowsNativeGC(t *testing.T) {
	service := &nodeScanService{id: "root-owner"}
	defer service.stopObjects()
	channel, state := exportCollectibleTransactionChannel(t, service)
	waitObjectGC(t, func() bool {
		_, registered := service.channels.Load(channel)
		return state.Value() == nil && !registered
	})
}

func TestClusterObjectOldIncarnationCannotCallOrReleaseNewOwner(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	value := scm.NewSession()
	scm.Apply(value, scm.NewString("key"), scm.NewInt(7))
	object, _ := scm.ClusterObjectOf(value)
	view, err := scm.ClusterObjectValue(object, context.Background(), scm.ClusterRequest{Method: "view"})
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := owner.exportObject(source, object, view, nil)
	if err != nil {
		t.Fatal(err)
	}
	// A restarted owner can reuse the same address and local grant number.
	// Its authenticated boot must fence both old calls and delayed GC closes.
	stale := fresh
	stale.Incarnation = uuid.NewString()
	previous := &mosiPeer{objectPeer: clusterObjectPeer{ID: peer.objectPeer.ID, Boot: stale.Incarnation}}
	call := client.objectCaller(previous, stale, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := call(ctx, scm.ClusterRequest{Method: "call", Arguments: []scm.Scmer{scm.NewString("key"), scm.NewInt(99)}}); err == nil {
		t.Fatal("old handle executed on a different owner incarnation")
	}
	if got := scm.Apply(value, scm.NewString("key")); got.Int() != 7 {
		t.Fatal("old call changed the new owner", got)
	}
	if client.deliverObjectRelease(clusterObjectRelease{peer: previous.objectPeer, reference: stale}) {
		t.Fatal("old GC release was published to a different incarnation")
	}
	if objectEntryCount(owner) != 1 {
		t.Fatal("old cleanup revoked the new owner's grant")
	}
}

func TestClusterObjectGCReconnectKeepsQueuedCloseConnectionAlive(t *testing.T) {
	peer, client, _ := setupRemoteScanConnection(t)
	owner := nodeScans.Load()
	source := objectOwnerPeer(t, owner, client)
	object, _ := scm.ClusterObjectOf(scm.NewSession())
	view, err := scm.ClusterObjectValue(object, context.Background(), scm.ClusterRequest{Method: "view"})
	if err != nil {
		t.Fatal(err)
	}
	reference, err := owner.exportObject(source, object, view, nil)
	if err != nil {
		t.Fatal(err)
	}
	identity := peer.objectPeer
	client.forget(peer)
	if !client.deliverObjectRelease(clusterObjectRelease{peer: identity, reference: reference}) {
		t.Fatal("same-incarnation reconnect did not queue its close")
	}
	if client.objectConnection(identity) == nil {
		t.Fatal("GC closed its socket before the asynchronous writer could send")
	}
	waitObjectGC(t, func() bool { return objectEntryCount(owner) == 0 })
}
