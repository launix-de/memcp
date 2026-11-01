/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/

package storage

import "io"
import "fmt"
import "sync"
import "time"
import "bytes"
import "context"
import "strings"
import "testing"
import "runtime"
import "net/http"
import "compress/gzip"
import "encoding/json"
import "encoding/binary"
import "encoding/base64"
import "net/http/httptest"
import "github.com/google/uuid"
import "github.com/gorilla/websocket"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

func TestReplicationOrderedScanRetainsColdPayloadThroughCallback(t *testing.T) {
	for _, name := range []string{"direct", "first", "multi", "recset", "batch", "direct-panic", "first-panic", "multi-panic", "recset-panic", "batch-panic"} {
		t.Run(name, func(t *testing.T) {
			path := strings.TrimSuffix(name, "-panic")
			failCallback := path != name
			tbl, _ := createDurabilityTestTable(t, "torderedcold", 4)
			RebuildTable(tbl, true, false)
			shard := tbl.ActiveShards()[0]
			part := newRecSetShardFromSortedIDs(shard, 4, []uint32{0, 1, 2, 3})
			source := &recSet{table: tbl, shards: []recSetShard{part}, count: 4}
			var freed [numEvictableTypes]int64
			if !shardCleanup(shard, &freed) || shard.state() != COLD {
				t.Fatal("fixture did not become cold")
			}
			_, order := integerOrder(false)
			condition := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
			callback := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
				if shardCleanup(shard, &freed) {
					t.Error("evicted the shard while an ordered callback still owns its rows")
				}
				if failCallback {
					panic("ordered callback failed")
				}
				return scm.NewInt(args[0].Int() + args[1].Int())
			})
			schema := newScanAccessSchema(scanAccessConsumerScan, nil, -1)
			spec := scanOrderTableSpec{table: tbl, accessSchema: schema, condition: condition,
				sortcols: []scm.Scmer{scm.NewString("id")}, callbackCols: []string{"id"}, callback: callback,
				perTableOffset: -1, perTableLimit: -1}
			var got scm.Scmer
			var failure any
			func() {
				defer func() { failure = recover() }()
				switch path {
				case "multi":
					got = scanOrderMulti(nil, []scanOrderTableSpec{spec}, []func(...scm.Scmer) scm.Scmer{order}, 0, 0, 4, scm.NewInt(0), false, scm.NewNil())
				case "recset":
					got = source.scan_order(nil, schema, nil, nil, condition, spec.sortcols, []func(...scm.Scmer) scm.Scmer{order}, 0, 0, 4, spec.callbackCols, callback, scm.NewInt(0), false, scm.NewNil(), nil, scm.NewNil())
				case "batch":
					got = scanOrderBatchAccept(nil, spec, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return args[0] }), spec.sortcols, []func(...scm.Scmer) scm.Scmer{order}, 0, 0, 4, spec.callbackCols, callback, scm.NewInt(0), false, scm.NewNil())
				default:
					limit := 4
					if path == "first" {
						spec.sortcols = nil
						limit = 1
					}
					var dirs []func(...scm.Scmer) scm.Scmer
					if len(spec.sortcols) != 0 {
						dirs = []func(...scm.Scmer) scm.Scmer{order}
					}
					got = tbl.scan_order(nil, schema, nil, nil, condition, spec.sortcols, dirs, 0, 0, limit, spec.callbackCols, callback, scm.NewInt(0), false, scm.NewNil(), nil, scm.NewNil())
				}
			}()
			if failCallback {
				if fmt.Sprint(failure) != "ordered callback failed" {
					t.Fatalf("callback panic = %v", failure)
				}
			} else if failure != nil {
				t.Fatalf("ordered scan failed: %v", failure)
			}
			want := int64(10)
			if path == "first" {
				want = 1
			}
			if !failCallback && got.Int() != want {
				t.Fatalf("sum = %v, want %d", got, want)
			}
			if shard.cacheReaders.Load() != 0 {
				t.Fatal("ordered scan leaked its payload lease")
			}
			if !shardCleanup(shard, &freed) {
				t.Fatal("completed ordered scan still prevents eviction")
			}
		})
	}
}

func TestReplicationOrderedMutationRequiresWritePermission(t *testing.T) {
	for _, path := range []string{"first", "direct", "multi", "batch"} {
		t.Run(path, func(t *testing.T) {
			tbl, _ := createDurabilityTestTable(t, "torderedrights", 3)
			r := &replicationManager{cells: make(map[string]*replicaCell), directory: replicaDirectory{Records: make(map[string]replicaRecord)}}
			r.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
			shard := tbl.ActiveShards()[0]
			r.shardCell(shard).ownstate.Initialize(mosi.S)
			previous := replication.Swap(r)
			defer replication.Store(previous)
			called := false
			var failure any
			func() {
				defer func() { failure = recover() }()
				var sorts []scm.Scmer
				var dirs []func(...scm.Scmer) scm.Scmer
				if path != "first" {
					_, order := integerOrder(false)
					sorts, dirs = []scm.Scmer{scm.NewString("id")}, []func(...scm.Scmer) scm.Scmer{order}
				}
				callback := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
					called = true
					scm.Apply(args[1])
					return args[0]
				})
				spec := scanOrderTableSpec{table: tbl, accessSchema: newScanAccessSchema(scanAccessConsumerScan, nil, -1), condition: trueCondition(), sortcols: sorts, callbackCols: []string{"$update"}, callback: callback, perTableOffset: -1, perTableLimit: -1}
				switch path {
				case "multi":
					scanOrderMulti(nil, []scanOrderTableSpec{spec}, dirs, 0, 0, 1, scm.NewInt(0), false, scm.NewNil())
				case "batch":
					scanOrderBatchAccept(nil, spec, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return args[0] }), sorts, dirs, 0, 0, 1, spec.callbackCols, callback, scm.NewInt(0), false, scm.NewNil())
				default:
					tbl.scan_order(nil, spec.accessSchema, nil, nil, spec.condition, sorts, dirs, 0, 0, 1, spec.callbackCols, callback, scm.NewInt(0), false, scm.NewNil(), nil, scm.NewNil())
				}
			}()
			if failure == nil || !strings.Contains(fmt.Sprint(failure), mosi.NetworkError) {
				t.Errorf("write without a reachable owner failed with %v, want %s", failure, mosi.NetworkError)
			}
			if called || shard.deletions.Count() != 0 {
				t.Error("ordered mutation ran without exclusive MOSI permission")
			}
		})
	}
}

func TestReplicationOrderedMutationDrainsNativeTransactionBeforeHandover(t *testing.T) {
	for _, mode := range []TxMode{TxCursorStability, TxACID} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			tbl, _ := createDurabilityTestTable(t, "torderedtxrights", 3)
			r := &replicationManager{cells: make(map[string]*replicaCell), directory: replicaDirectory{Records: make(map[string]replicaRecord)}}
			r.resourceCell(mosiResource{Database: tbl.schema.Name}).ownstate.Initialize(mosi.M)
			shard := tbl.ActiveShards()[0]
			cell := r.shardCell(shard)
			cell.ownstate.Initialize(mosi.M)
			previous := replication.Swap(r)
			defer replication.Store(previous)
			tx := NewTxContext(mode)
			defer replicationLeave(tx)
			defer tx.Rollback()
			_, order := integerOrder(false)
			tbl.scan_order(tx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, trueCondition(),
				[]scm.Scmer{scm.NewString("id")}, []func(...scm.Scmer) scm.Scmer{order}, 0, 0, 1,
				[]string{"$update"}, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
					if handover := cell.ownstate.TryHandover(cell.writesDrained); handover != nil {
						handover.Release()
						t.Error("ownership handover was admitted during the mutation callback")
					}
					scm.Apply(args[1])
					return args[0]
				}), scm.NewInt(0), false, scm.NewNil(), nil, scm.NewNil())
			if tx.IsVisible(shard, 0) {
				t.Fatal("transaction did not delete the first row")
			}
			if handover := cell.ownstate.TryHandover(cell.writesDrained); handover != nil {
				handover.Release()
				t.Fatal("native transaction did not defer ownership handover")
			}
			tx.Rollback()
			replicationLeave(tx)
			if shard.deletions.Count() != 0 {
				t.Fatal("rollback did not restore the ordered deletion")
			}
			handover := cell.ownstate.TryHandover(cell.writesDrained)
			if handover == nil {
				t.Fatal("completed transaction retained write admission")
			}
			handover.Release()
		})
	}
}

func TestReplicationOrderedRecSetKeepsHistoricalGeneration(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "torderedhistory", 3)
	RebuildTable(tbl, true, false)
	original := tbl.ActiveShards()[0]
	part := newRecSetShardFromSortedIDs(original, 3, []uint32{0, 1, 2})
	source := &recSet{table: tbl, shards: []recSetShard{part}, count: 3}
	r := &replicationManager{cells: make(map[string]*replicaCell), directory: replicaDirectory{Records: make(map[string]replicaRecord)}}
	cell := r.shardCell(original)
	replacement := original.rebuild(true)
	if replacement == original {
		t.Fatal("fixture did not produce a successor")
	}
	// A payload hint alone is not publication. Replace the native topology
	// before checking mutations against the retained historical physical IDs.
	if err := r.replaceShard(r.shardResource(original), original, replacement); err != nil {
		t.Fatal(err)
	}
	cell.ownstate.Initialize(mosi.IO)
	previous := replication.Swap(r)
	defer replication.Store(previous)
	_, order := integerOrder(false)
	got := source.scan_order(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, trueCondition(),
		[]scm.Scmer{scm.NewString("id")}, []func(...scm.Scmer) scm.Scmer{order}, 0, 0, 3,
		[]string{"id"}, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return scm.NewInt(args[0].Int() + args[1].Int()) }),
		scm.NewInt(0), false, scm.NewNil(), nil, scm.NewNil())
	if got.Int() != 6 || cell.ownstate.Phase() != mosi.IO {
		t.Fatalf("historical sum = %v, current rights = %v", got, cell.ownstate.Phase())
	}
	if original.cacheReaders.Load() != 0 {
		t.Fatal("historical ordered scan leaked its read capability")
	}
	// Acquiring current M must never reinterpret these historical physical IDs.
	cell.ownstate.Initialize(mosi.M)
	called := false
	var failure any
	func() {
		defer func() { failure = recover() }()
		source.scan_order(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, trueCondition(),
			[]scm.Scmer{scm.NewString("id")}, []func(...scm.Scmer) scm.Scmer{order}, 0, 0, 1,
			[]string{"$update"}, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
				called = true
				return args[0]
			}),
			scm.NewInt(0), false, scm.NewNil(), nil, scm.NewNil())
	}()
	if called || !strings.Contains(fmt.Sprint(failure), "generation changed; retry statement") {
		t.Fatalf("historical mutation called = %v, failure = %v", called, failure)
	}
	handover := cell.ownstate.TryHandover(cell.writesDrained)
	if handover == nil {
		t.Fatal("content mismatch leaked its write capability")
	}
	handover.Release()
}

func TestMosiBinaryEnvelopeRoundTripAndLimits(t *testing.T) {
	for _, message := range []mosiMessage{
		{ID: 7, Method: "authenticate", Data: json.RawMessage(`{"user":"alice"}`)},
		{ID: 7, Error: "denied"},
		{Packet: &mosiPacket{Kind: "grant", ID: "request", Epoch: 42, State: "M", Content: "content"}},
		{Packet: &mosiPacket{Kind: "chunk", ID: "request", Payload: bytes.Repeat([]byte{0xff, 0, 1}, 20000)}},
	} {
		frame, err := encodeMosiMessage(message)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := decodeMosiMessage(frame)
		if err != nil {
			t.Fatal(err)
		}
		want, _ := json.Marshal(message)
		got, _ := json.Marshal(decoded)
		if !bytes.Equal(got, want) {
			t.Fatalf("binary round trip changed the message: %s", got)
		}
		if message.Packet != nil && message.Packet.Kind == "chunk" && len(frame) > len(message.Packet.Payload)+1024 {
			t.Fatal("binary chunk was expanded rather than carried as raw bytes")
		}
	}
	valid, _ := encodeMosiMessage(mosiMessage{ID: 1})
	badVersion := bytes.Clone(valid)
	badVersion[4] = 1
	badLength := bytes.Clone(valid)
	binary.BigEndian.PutUint32(badLength[5:9], ^uint32(0))
	inlineMetadata := []byte(`{"packet":{"kind":"chunk","payload":"eA=="}}`)
	inlinePayload := make([]byte, 9+len(inlineMetadata))
	copy(inlinePayload, "MOSI\x02")
	binary.BigEndian.PutUint32(inlinePayload[5:9], uint32(len(inlineMetadata)))
	copy(inlinePayload[9:], inlineMetadata)
	for _, frame := range [][]byte{nil, valid[:8], badVersion, badLength, append(bytes.Clone(valid), 1), inlinePayload, make([]byte, mosiMaxMessageBytes+1)} {
		if _, err := decodeMosiMessage(frame); err == nil {
			t.Fatal("malformed or oversized binary envelope was accepted")
		}
	}
	for _, message := range []mosiMessage{
		{Data: json.RawMessage(`"` + strings.Repeat("x", mosiMaxMessageBytes) + `"`)},
		{Packet: &mosiPacket{Kind: "grant", Payload: []byte{1}}},
		{Packet: &mosiPacket{Kind: "chunk", Payload: make([]byte, mosiPacketBytes+1025)}},
	} {
		if _, err := encodeMosiMessage(message); err == nil {
			t.Fatal("invalid outgoing binary envelope was accepted")
		}
	}
}

func TestMosiEndpointURLs(t *testing.T) {
	for input, want := range map[string]string{
		"127.0.0.1:7331": "ws://127.0.0.1:7331/cluster", "[::1]:4321": "ws://[::1]:4321/cluster",
		"http://node:4321": "ws://node:4321/cluster", "https://node/": "wss://node/cluster",
		"wss://node/prefix/cluster": "wss://node/prefix/cluster",
	} {
		if got, err := mosiEndpointURL(input); err != nil || got != want {
			t.Errorf("%q resolved to %q, %v; want %q", input, got, err, want)
		}
	}
	for _, input := range []string{"node", "http://", "ftp://node", "http://user:secret@node", "http://node?secret=x", "https://node/#fragment"} {
		if _, err := mosiEndpointURL(input); err == nil {
			t.Errorf("invalid node URL accepted: %q", input)
		}
	}
}

func TestMosiHTTPRolesKeepIndependentAuthenticationBoundaries(t *testing.T) {
	register := func(role string, status int) func() {
		t.Helper()
		release, err := registerMosiRole(role, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { w.WriteHeader(status) }))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(release)
		return release
	}
	releaseReplication := register("replication", http.StatusUnauthorized)
	register("test-peer", http.StatusAccepted)
	request := func(role string) int {
		req := httptest.NewRequest(http.MethodGet, mosiHTTPPath, nil)
		req.Header.Set("X-MemCP-Role", role)
		response := httptest.NewRecorder()
		serveMosiHTTP(response, req)
		return response.Code
	}
	if request("replication") != http.StatusUnauthorized || request("test-peer") != http.StatusAccepted || request("cluster") != http.StatusForbidden {
		t.Fatal("a role was routed to another adapter's authentication boundary")
	}
	releaseReplication()
	if request("replication") != http.StatusForbidden || request("test-peer") != http.StatusAccepted {
		t.Fatal("removing one role disrupted the other role")
	}
	register("replication", http.StatusNoContent)
	releaseReplication()
	if request("replication") != http.StatusNoContent {
		t.Fatal("old role cleanup removed its replacement")
	}
}

func TestReplicationWebSocketUsesNormalHTTPRoute(t *testing.T) {
	r := &replicationManager{id: uuid.NewString(), peers: make(map[string]*mosiPeer)}
	finished := make(chan struct{})
	unregister, err := registerMosiRole("replication", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer close(finished)
		r.accept(w, req)
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer unregister()
	fallback := scm.NewAny(http.NotFoundHandler())
	router := scm.HTTPRoute(scm.NewString("/custom/node"), scm.NewAny(http.HandlerFunc(serveMosiHTTP)), fallback)
	server := httptest.NewServer(router.Any().(http.Handler))
	defer server.Close()
	target, _ := mosiEndpointURL(server.URL + "/custom/node")
	dialer := *websocket.DefaultDialer
	dialer.Subprotocols = []string{mosiSubprotocol}
	header := http.Header{"X-Memcp-Replica": []string{uuid.NewString()}, "X-Memcp-Role": []string{"replication"}}
	conn, _, err := dialer.Dial(target, header)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { conn.Close(); <-finished }()
	if conn.Subprotocol() != mosiSubprotocol {
		t.Fatal("normal HTTP endpoint did not negotiate the node protocol")
	}
	frame, _ := encodeMosiMessage(mosiMessage{ID: 9, Method: "unsupported"})
	if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		t.Fatal(err)
	}
	conn.SetReadDeadline(time.Now().Add(time.Second))
	kind, reply, err := conn.ReadMessage()
	if err != nil || kind != websocket.BinaryMessage {
		t.Fatalf("node reply was not binary: kind=%d err=%v", kind, err)
	}
	message, err := decodeMosiMessage(reply)
	if err != nil || message.ID != 9 || message.Error == "" {
		t.Fatalf("unexpected node reply: %+v, %v", message, err)
	}
	// Old JSON/text frames are rejected rather than silently downgraded.
	conn.WriteMessage(websocket.TextMessage, []byte(`{"id":10,"method":"unsupported"}`))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("text message was accepted by the binary node protocol")
	}
}

func TestReplicationWebSocketRejectsUnsupportedProtocol(t *testing.T) {
	r := &replicationManager{}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		response := httptest.NewRecorder()
		r.accept(response, httptest.NewRequest(method, mosiHTTPPath, nil))
		if response.Code != http.StatusUpgradeRequired && response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("invalid node handshake returned %d", response.Code)
		}
	}
}

func TestReplicationReclaimIncludesEmptyAndColdShards(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "treplicareclaim", 3)
	empty, _ := CreateTable(table.schema.Name, "empty", Safe, false)
	empty.CreateColumn("id", "INT", nil, nil)
	r := &replicationManager{id: "slave", cells: make(map[string]*replicaCell), directory: replicaDirectory{Records: make(map[string]replicaRecord)}}
	schema := mosiResource{Database: table.schema.Name}
	r.resourceCell(schema).ownstate.Initialize(mosi.M)
	r.schemaShards(table.schema, r.id, true)
	for _, item := range table.schema.tables.GetAll() {
		for _, shard := range item.ActiveShards() {
			r.shardCell(shard).ownstate.RestoreBackend(mosi.M)
		}
	}
	if !r.reclaimDatabase(table.schema.Name) {
		t.Fatal("current backend-only exclusive generations were not reclaimed")
	}
	for _, item := range table.schema.tables.GetAll() {
		for _, shard := range item.ActiveShards() {
			if r.shardCell(shard).ownstate.Phase() != mosi.M {
				t.Fatalf("%s: active shard was not acquired", item.Name)
			}
		}
	}
}

func TestReplicationPeriodicReclaimContinuesAfterOfflineTenant(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "z_periodic_reclaim", 3)
	r := &replicationManager{id: "slave", cells: make(map[string]*replicaCell), directory: replicaDirectory{
		Records:     make(map[string]replicaRecord),
		Initialized: map[string]bool{"a_offline_tenant": true, table.schema.Name: true, "pending_tenant": false, "system": true},
	}}
	r.resourceCell(mosiResource{Database: "a_offline_tenant"}).ownstate.Initialize(mosi.S)
	r.resourceCell(mosiResource{Database: table.schema.Name}).ownstate.Initialize(mosi.M)
	r.schemaShards(table.schema, r.id, true)
	for _, shard := range table.ActiveShards() {
		r.shardCell(shard).ownstate.RestoreBackend(mosi.M)
	}
	previous := replication.Swap(r)
	defer replication.Store(previous)
	failures := ReclaimReplicationDatabases()
	if len(failures) != 1 || !strings.Contains(failures[0], "a_offline_tenant") || !strings.Contains(failures[0], mosi.NetworkError) {
		t.Fatalf("expected the offline tenant failure only, got %v", failures)
	}
	for _, shard := range table.ActiveShards() {
		if r.shardCell(shard).ownstate.Phase() != mosi.M {
			t.Fatal("later tenant was not reclaimed after the offline failure")
		}
	}
	if r.resourceCell(mosiResource{Database: "a_offline_tenant"}).ownstate.Phase() == mosi.M {
		t.Fatal("offline tenant was promoted without upstream permission")
	}
	for _, name := range []string{"system", "pending_tenant"} {
		if r.cells[name] != nil {
			t.Fatalf("periodic maintenance acquired excluded database %s", name)
		}
	}
	for _, manager := range []*replicationManager{nil, {master: true}, {}} {
		if manager != nil && !manager.master {
			manager.retired.Store(true)
		}
		replication.Store(manager)
		if failures := ReclaimReplicationDatabases(); len(failures) != 0 {
			t.Fatalf("inactive periodic reclaim failed: %v", failures)
		}
	}
}

func TestReplicationPeriodicReclaimUsesOrdinaryMosiGrant(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_periodic_remote_reclaim", 3)
	r := &replicationManager{id: "slave", cells: make(map[string]*replicaCell), directory: replicaDirectory{
		Records:     make(map[string]replicaRecord),
		Users:       map[string]replicaAuth{"tenant": {Hash: "cached-proof", Databases: map[string]bool{table.schema.Name: true}}},
		Initialized: map[string]bool{table.schema.Name: true},
	}}
	schema := mosiResource{Database: table.schema.Name}
	r.resourceCell(schema).ownstate.Initialize(mosi.S)
	r.schemaShards(table.schema, r.id, true)
	peer := &mosiPeer{outgoing: make(chan mosiMessage, 8), done: make(chan struct{})}
	r.upstream = peer
	previous := replication.Swap(r)
	defer replication.Store(previous)
	var failures []string
	done := make(chan struct{})
	go func() {
		failures = ReclaimReplicationDatabases()
		close(done)
	}()
	defer func() {
		close(peer.done)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("periodic reclaim did not finish after disconnect")
		}
	}()
	select {
	case message := <-peer.outgoing:
		packet := message.Packet
		if packet == nil || packet.Kind != "write" || packet.Resource != schema || packet.User != "tenant" || packet.Hash != "cached-proof" || packet.Sequence == 0 {
			t.Fatalf("unexpected periodic ownership request: %+v", packet)
		}
		if r.resourceCell(schema).ownstate.Phase() != mosi.MR {
			t.Fatal("request granted rights before the upstream grant")
		}
		if err := r.acceptGrant(mosiPacket{ID: packet.ID, Resource: schema, State: "M", Epoch: 2, Owner: r.id}); err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("periodic reclaim did not request schema M")
	}
	select {
	case <-done:
		if len(failures) != 0 {
			t.Fatalf("periodic reclaim failed after grant: %v", failures)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("periodic reclaim did not finish after grant")
	}
	if r.resourceCell(schema).ownstate.Phase() != mosi.M || r.directory.Records[r.resourceKey(schema)].Epoch != 2 {
		t.Fatal("periodic reclaim did not retain the durable upstream grant")
	}
}

func replicaAdmissionFixture(name string) (*replicationManager, *replicaCell, scm.Scmer) {
	r := &replicationManager{cells: make(map[string]*replicaCell)}
	c := r.resourceCell(mosiResource{Database: name})
	database := scm.NewString(name)
	session := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		if a[0].String() == "schema" {
			return database
		}
		return scm.NewNil()
	})
	return r, c, session
}

func TestReplicationParallelReadsIgnoreControlLocks(t *testing.T) {
	old := replication.Load()
	defer replication.Store(old)
	for _, state := range []string{"M", "O", "S"} {
		r, c, session := replicaAdmissionFixture("parallel_analytics")
		c.ownstate.Initialize(mosi.ParsePhase(state))
		c.initialized.Store(true)
		replication.Store(r)
		// Hold every control-plane lock, including closed writer admission.
		// Already-authorized reads must still complete on their own snapshots.
		r.mu.Lock()
		handover := c.ownstate.TryHandover(c.writesDrained)
		if handover == nil {
			t.Fatal("unexpected active write")
		}
		c.acquire.Lock()
		c.directory.Lock()
		c.checkpoint.Lock()
		c.sendMu.Lock()
		c.importPublication.Lock()
		done := make(chan struct{})
		go func() {
			var readers sync.WaitGroup
			for i := 0; i < 32; i++ {
				readers.Add(1)
				go func() {
					defer readers.Done()
					tx := NewTxContext(TxCursorStability)
					for j := 0; j < 1000; j++ {
						replicationEnter(tx, session)
						replicationLeave(tx)
					}
				}()
			}
			readers.Wait()
			close(done)
		}()
		timedOut := false
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			timedOut = true
		}
		c.importPublication.Unlock()
		c.sendMu.Unlock()
		c.checkpoint.Unlock()
		c.directory.Unlock()
		c.acquire.Unlock()
		handover.Release()
		r.mu.Unlock()
		<-done
		if timedOut {
			t.Fatalf("%s reads waited for shared control locks", state)
		}
		if c.ownstate.Phase().String() != state {
			t.Fatalf("%s read wrote the shared flags", state)
		}
	}
}

func BenchmarkReplicationReadAdmission(b *testing.B) {
	old := replication.Load()
	defer replication.Store(old)
	r, c, session := replicaAdmissionFixture("parallel_analytics")
	state := "S"
	c.ownstate.Initialize(mosi.ParsePhase(state))
	c.initialized.Store(true)
	replication.Store(r)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		tx := NewTxContext(TxCursorStability)
		for pb.Next() {
			replicationEnter(tx, session)
			replicationLeave(tx)
		}
	})
	runtime.KeepAlive(r)
}

func TestReplicationBidirectionalRPCAndDisconnect(t *testing.T) {
	accepted := make(chan *mosiPeer, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, req, nil)
		if err != nil {
			return
		}
		accepted <- newMosiPeer(conn, func(ctx context.Context, method string, data json.RawMessage) (any, error) {
			if method == "block" {
				<-ctx.Done()
				return nil, ctx.Err()
			}
			return string(data), nil
		}, nil)
	}))
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	client := newMosiPeer(conn, func(ctx context.Context, method string, data json.RawMessage) (any, error) { return string(data), nil }, nil)
	master := <-accepted
	defer client.close()
	defer master.close()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p := client
			if i%2 == 0 {
				p = master
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			var output string
			if err := p.call(ctx, "echo", i, &output); err != nil || output != fmt.Sprint(i) {
				t.Errorf("RPC %d: %q %v", i, output, err)
			}
		}(i)
	}
	wg.Wait()
	finished := make(chan error, 1)
	go func() { finished <- client.call(context.Background(), "block", nil, nil) }()
	client.close()
	select {
	case err := <-finished:
		if err == nil {
			t.Fatal("disconnected RPC succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("disconnect did not release pending RPC")
	}
}

func TestReplicationPasswordProofs(t *testing.T) {
	// Fixed mysql_native_password vector: SHA1(password), salt bytes 0..19.
	raw, err := base64.StdEncoding.DecodeString("LTascojglBT9PkRYCeOarSGjZrM=")
	if err != nil {
		t.Fatal(err)
	}
	scramble, _ := base64.StdEncoding.DecodeString("OfHT2dhT7wGkMn9zhgTPsGY26qc=")
	salt := make([]byte, 20)
	for i := range salt {
		salt[i] = byte(i)
	}
	if !replicationPasswordProof(raw, "", salt, scramble) {
		t.Fatal("valid MySQL challenge rejected")
	}
	if !replicationPasswordProof(raw, "LTascojglBT9PkRYCeOarSGjZrM=", nil, nil) {
		t.Fatal("valid HTTP hash rejected")
	}
	scramble[2] ^= 1
	if replicationPasswordProof(raw, "", salt, scramble) {
		t.Fatal("tampered challenge accepted")
	}
	if replicationPasswordProof(raw, "", salt, scramble[:2]) {
		t.Fatal("short challenge accepted")
	}
	if replicationPasswordProof(raw, "wrong", nil, nil) {
		t.Fatal("wrong password accepted")
	}
}

func TestReplicationSnapshotWaitsForMaintenance(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "treplicationmaintenance", 3)
	r := &replicationManager{cells: make(map[string]*replicaCell), directory: replicaDirectory{Records: make(map[string]replicaRecord)}}
	resource := mosiResource{Database: tbl.schema.Name, Table: tbl.Name, Shard: tbl.ActiveShards()[0].uuid.String()}
	tbl.maintenanceMu.Lock()
	locked := true
	defer func() {
		if locked {
			tbl.maintenanceMu.Unlock()
		}
	}()
	done := make(chan error, 1)
	go func() { _, err := r.prepareResource(resource); done <- err }()
	select {
	case err := <-done:
		t.Fatalf("snapshot skipped busy table: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	tbl.maintenanceMu.Unlock()
	locked = false
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("snapshot did not resume")
	}
}

func TestReplicationBoundedChunkStream(t *testing.T) {
	peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 8)}
	finished := make(chan error, 1)
	payload := bytes.Repeat([]byte("streamed shard content\n"), 100000)
	go func() {
		finished <- peer.sendObject(mosiPacket{ID: "stream"}, "column", "payload", io.NopCloser(bytes.NewReader(payload)))
	}()
	total, packets := 0, 0
	for {
		message := <-peer.outgoing
		packet := message.Packet
		if packet.Kind == "object_end" {
			break
		}
		if packet.Kind != "chunk" {
			continue
		}
		decoded, err := decodeMosiChunk(*packet)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(decoded, payload[total:total+len(decoded)]) {
			t.Fatal("stream corrupted")
		}
		total += len(decoded)
		packets++
	}
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if total != len(payload) || packets < 2 {
		t.Fatalf("not an incremental shard stream: %d/%d in %d packets", total, len(payload), packets)
	}
	var expanded bytes.Buffer
	writer := gzip.NewWriter(&expanded)
	writer.Write(make([]byte, mosiPacketBytes+1))
	writer.Close()
	if _, err := decodeMosiChunk(mosiPacket{Payload: expanded.Bytes()}); err == nil {
		t.Fatal("unbounded expanded chunk accepted")
	}
}

func TestReplicationSnapshotKeepsGenerationAndFiles(t *testing.T) {
	table, engine := createDurabilityTestTable(t, "treplicasnapshot", 3)
	r := &replicationManager{cells: make(map[string]*replicaCell), directory: replicaDirectory{Records: make(map[string]replicaRecord)}}
	resource := mosiResource{Database: table.schema.Name, Table: table.Name, Shard: table.ActiveShards()[0].uuid.String()}
	if _, err := r.prepareResource(resource); err != nil {
		t.Fatal(err)
	}
	cell := r.resourceCell(resource)
	original := cell.current.Load()
	cell.ownstate.Initialize(mosi.S)
	previous := replication.Swap(r)
	defer replication.Store(previous)
	replacement := original.rebuild(true)
	if replacement == original {
		t.Fatal("forced rebuild did not produce a successor")
	}
	if err := r.replaceShard(resource, original, replacement); err != nil {
		t.Fatal(err)
	}
	cell.current.Store(replacement)
	access := original.GetRead(nil, false)
	if access.shard != replacement {
		access.Release()
		t.Fatal("ordinary read did not resolve the current content")
	}
	access.Release()
	// These query-local IDs already own the historical content. Losing
	// permission to the current shard must not cause projection to fetch or
	// reinterpret newer rows, even when that owner is now offline.
	cell.ownstate.Publish(mosi.IO)
	access = original.acquireSnapshotRead(nil)
	if access.shard != original {
		access.Release()
		t.Fatal("physical snapshot IDs were rebound to another content")
	}
	access.Release()
	original.RemoveFromDisk()
	reader := engine.ReadColumn(original.uuid.String(), "id")
	defer reader.Close()
	if missing, ok := reader.(interface{ Missing() bool }); ok && missing.Missing() {
		t.Fatal("retirement deleted a reachable snapshot's column file")
	}
	if _, _, complete := CleanDatabase(table.schema); complete {
		t.Fatal("cleanup ignored historical content ownership")
	}
	runtime.KeepAlive(original)
}

func TestReplicationCleanupRetainsCompletedBackup(t *testing.T) {
	table, engine := createDurabilityTestTable(t, "treplicabackupgc", 3)
	r := &replicationManager{master: true, id: "master", cells: make(map[string]*replicaCell), directory: replicaDirectory{Records: make(map[string]replicaRecord)}}
	resource := mosiResource{Database: table.schema.Name, Table: table.Name, Shard: table.ActiveShards()[0].uuid.String()}
	if _, err := r.prepareResource(resource); err != nil {
		t.Fatal(err)
	}
	backup := r.resourceCell(resource).current.Load()
	// A newer remote schema can advertise a UUID before its data arrives. It
	// cannot displace the last completed backup's backend ownership proof.
	pending := new(storageShard)
	data, _ := json.Marshal(map[string]string{"uuid": uuid.NewString(), "mosi": resource.Shard})
	if err := json.Unmarshal(data, pending); err != nil {
		t.Fatal(err)
	}
	pending.t = table
	r.forwardResource(resource, mosiResource{Database: resource.Database, Table: resource.Table, Shard: pending.uuid.String()})
	table.mu.Lock()
	table.Shards = []*storageShard{pending}
	table.publishTopologyLocked()
	table.mu.Unlock()
	cell := r.shardCell(pending)
	cell.current.Store(pending)
	cell.ownstate.Initialize(mosi.IO)
	r.directory.Records[r.resourceKey(resource)] = replicaRecord{Resource: resource, Owner: "slave", State: "IO", Epoch: 1, Content: backup.uuid.String()}
	r.backups = map[string]replicaBackup{r.resourceKey(resource): {Resource: resource, Content: backup.uuid.String()}}
	orphan := uuid.NewString()
	writer := engine.WriteColumn(orphan, "unused")
	writer.Write([]byte("orphan"))
	finishColumnWrite(writer, true)
	previous := replication.Swap(r)
	defer replication.Store(previous)
	if removed := cleanShards(table.schema); removed == 0 {
		t.Fatal("cleanup retained unrelated orphan files")
	}
	reader := engine.ReadColumn(backup.uuid.String(), "payload")
	if missing, ok := reader.(interface{ Missing() bool }); ok && missing.Missing() {
		reader.Close()
		t.Fatal("cleanup deleted the last complete backup")
	}
	reader.Close()
	if err := r.restoreBackup(r.directory.Records[r.resourceKey(resource)]); err != nil {
		t.Fatal(err)
	}
	cell.ownstate.Publish(mosi.M)
	access := cell.current.Load().GetRead(nil, false)
	defer access.Release()
	value := access.shard.getColumnStorageOrPanic("payload", false, nil).GetValue(0)
	if value.String() != "row-00000001" {
		t.Fatalf("completed backup was not restored: %v", value)
	}
}

// Both transfer admission and column validation must wait for the complete
// lazy catalog, including when startup cleanup already owns its initialization.
func TestReplicationImportWaitsForSchemaLoad(t *testing.T) {
	for _, entry := range []string{"begin", "column"} {
		for _, loading := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/loading=%t", entry, loading), func(t *testing.T) {
				table, engine := createDurabilityTestTable(t, "treplicaimportload", 1)
				blocking := &blockingSchemaReadPersistence{
					PersistenceEngine: engine,
					entered:           make(chan struct{}, 2),
					release:           make(chan struct{}),
				}
				db := newDatabase()
				db.Name, db.persistence, db.srState = table.schema.Name, blocking, COLD
				databases.Set(db.Name, db)
				var release sync.Once
				unblock := func() { release.Do(func() { close(blocking.release) }) }
				var workers sync.WaitGroup
				t.Cleanup(func() {
					unblock()
					workers.Wait()
					db.closeTransactionLog()
					databases.Set(db.Name, table.schema)
				})
				if loading {
					workers.Add(1)
					go func() {
						defer workers.Done()
						db.ensureLoaded()
					}()
					<-blocking.entered
				}
				packet := mosiPacket{
					Resource: mosiResource{Database: db.Name, Table: table.Name, Shard: uuid.NewString()},
					Content:  uuid.NewString(), Kind: "object", ObjectKind: "column", Object: "payload",
				}
				r := &replicationManager{directory: replicaDirectory{Records: make(map[string]replicaRecord)}}
				done := make(chan error, 1)
				workers.Add(1)
				go func() {
					defer workers.Done()
					if entry == "begin" {
						transfer, err := r.startImport(packet)
						if transfer != nil {
							transfer.abort()
						}
						done <- err
					} else {
						transfer := &replicaImport{header: packet, engine: blocking}
						defer transfer.abort()
						done <- transfer.packet(packet)
					}
				}()
				if !loading {
					select {
					case <-blocking.entered:
					case err := <-done:
						t.Fatalf("%s returned without loading its cold catalog: %v", entry, err)
					case <-time.After(2 * time.Second):
						t.Fatal("import did not start schema initialization")
					}
				}
				select {
				case err := <-done:
					t.Fatalf("%s returned before schema publication: %v", entry, err)
				case <-time.After(50 * time.Millisecond):
				}
				unblock()
				select {
				case err := <-done:
					if err != nil {
						t.Fatalf("import after schema publication: %v", err)
					}
				case <-time.After(2 * time.Second):
					t.Fatal("import did not resume after schema publication")
				}
				workers.Wait()
				if blocking.calls.Load() != 1 {
					t.Fatal("import loaded the schema more than once")
				}
				loaded := db.tables.Get(table.Name)
				if loaded == nil || loaded.schema != db {
					t.Fatal("import did not observe the initialized table")
				}
			})
		}
	}
}

func TestReplicationRepeatedTransferKeepsCompletedColumns(t *testing.T) {
	table, engine := createDurabilityTestTable(t, "treplicaresend", 3)
	r := &replicationManager{master: true, id: "master", cells: make(map[string]*replicaCell), directory: replicaDirectory{Records: make(map[string]replicaRecord)}}
	resource := mosiResource{Database: table.schema.Name, Table: table.Name, Shard: table.ActiveShards()[0].uuid.String()}
	content, err := r.prepareResource(resource)
	if err != nil {
		t.Fatal(err)
	}
	r.directory.Records[r.resourceKey(resource)] = replicaRecord{Resource: resource, Owner: "slave", State: "IO", Content: content}
	read := func() []byte {
		t.Helper()
		reader := engine.ReadColumn(content, "payload")
		defer reader.Close()
		data, err := io.ReadAll(reader)
		if err != nil || len(data) == 0 {
			t.Fatalf("completed column is missing: %v", err)
		}
		return data
	}
	before := read()
	for _, complete := range []bool{false, true} {
		packet := mosiPacket{ID: uuid.NewString(), Resource: resource, Content: content, Kind: "object", ObjectKind: "column", Object: "payload"}
		transfer, err := r.startImport(packet)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(transfer.abort)
		if err := transfer.packet(packet); err != nil {
			t.Fatal(err)
		}
		// Even before any chunk arrives, opening a duplicate transfer must
		// leave the completed file readable by snapshots and retirement.
		if !bytes.Equal(read(), before) {
			t.Fatal("resend truncated a completed content on object open")
		}
		if complete {
			var packed bytes.Buffer
			writer := gzip.NewWriter(&packed)
			writer.Write(before)
			writer.Close()
			packet.Kind, packet.Payload = "chunk", packed.Bytes()
			if err := transfer.packet(packet); err != nil {
				t.Fatal(err)
			}
			packet.Kind, packet.Payload = "object_end", nil
			if err := transfer.packet(packet); err != nil {
				t.Fatal(err)
			}
		}
		transfer.abort()
		if !bytes.Equal(read(), before) {
			t.Fatal("completed or interrupted resend replaced immutable bytes")
		}
	}
}

func TestReplicationImportDefersCleanup(t *testing.T) {
	table, engine := createDurabilityTestTable(t, "treplicainflight", 3)
	r := &replicationManager{master: true, id: "master", cells: make(map[string]*replicaCell), directory: replicaDirectory{Records: make(map[string]replicaRecord)}}
	resource := mosiResource{Database: table.schema.Name, Table: table.Name, Shard: table.ActiveShards()[0].uuid.String()}
	if _, err := r.prepareResource(resource); err != nil {
		t.Fatal(err)
	}
	// This unpublished fixture has no historical query owners.
	r.resourceCell(resource).retiredShards.Store(nil)
	packet := mosiPacket{ID: "pending", Resource: resource, Content: uuid.NewString(), Kind: "object", ObjectKind: "column", Object: "payload"}
	transfer, err := r.startImport(packet)
	if err != nil {
		t.Fatal(err)
	}
	r.imports.Store("slave/pending", transfer)
	previous := replication.Swap(r)
	defer func() {
		transfer.abort()
		r.imports.Delete("slave/pending")
		replication.Store(previous)
	}()
	if err := transfer.packet(packet); err != nil {
		t.Fatal(err)
	}
	if blobs, shards, complete := CleanDatabase(table.schema); blobs != 0 || shards != 0 || complete {
		t.Fatalf("cleanup ran through a private transfer: %d blobs, %d shards, complete=%v", blobs, shards, complete)
	}
	reader := engine.ReadColumn(packet.Content, "payload")
	if missing, ok := reader.(interface{ Missing() bool }); ok && missing.Missing() {
		reader.Close()
		t.Fatal("cleanup deleted the incoming content")
	}
	reader.Close()
	transfer.abort()
	r.imports.Delete("slave/pending")
	table.schema.persistenceLifecycle.Lock()
	cleanShards(table.schema)
	table.schema.persistenceLifecycle.Unlock()
	reader = engine.ReadColumn(packet.Content, "payload")
	defer reader.Close()
	if missing, ok := reader.(interface{ Missing() bool }); !ok || !missing.Missing() {
		t.Fatal("aborted private content was retained after import cleanup")
	}
}

func TestReplicationBackupBackpressureDoesNotHoldActiveWriter(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "tbackupwriteradmission", 3)
	r := replicaRecoveryFixture()
	r.master = false
	r.directory.Users["root"] = replicaAuth{Admin: true}
	cell := r.declareShard(table.ActiveShards()[0], r.id, true)
	resource := *cell.resource.Load()
	// The queue is deliberately unbuffered: receiving begin proves preparation
	// finished; stopping consumption then holds the transport at its next packet.
	peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage)}
	r.upstream = peer
	done := make(chan struct{})
	go func() { defer close(done); r.backupDatabase(resource.Database, "root") }()
	defer func() { close(peer.done); <-done }()
	select {
	case message := <-peer.outgoing:
		if message.Packet == nil || message.Packet.Kind != "begin" {
			t.Fatal("backup did not begin")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("backup did not reach the transport")
	}
	writer, wait := cell.ownstate.TryWrite(nil, false)
	if writer == nil || wait != nil {
		t.Fatal("network backpressure retained local writer admission")
	}
	if !cell.ownstate.Phase().Readable() {
		t.Fatal("active backup writer excluded readers")
	}
	writer.Release()
}

func TestReplicationTransferRemainsBoundToPreparedGeneration(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "tboundtransfergeneration", 3)
	r := replicaRecoveryFixture()
	r.schemaShards(table.schema, r.id, true)
	previous := replication.Swap(r)
	defer replication.Store(previous)
	resource := *r.shardCell(table.ActiveShards()[0]).resource.Load()
	content, err := r.prepareResource(resource)
	if err != nil {
		t.Fatal(err)
	}
	transfer, err := r.captureResource(resource, content)
	if err != nil {
		t.Fatal(err)
	}
	defer transfer.release()
	if !r.hasTransfers(resource.Database) {
		t.Fatal("snapshot has no GC lifetime")
	}
	if _, _, complete := CleanDatabase(table.schema); complete {
		t.Fatal("GC ignored an outgoing snapshot")
	}
	failure := func() (failure any) {
		defer func() { failure = recover() }()
		DropDatabase(resource.Database, false)
		return nil
	}()
	if failure == nil || databases.Get(resource.Database) != table.schema {
		t.Fatal("DROP DATABASE retired an active transfer namespace")
	}
	// Publish another physical content before the transfer starts. Every
	// packet, including begin, must still refer to the captured original UUID.
	table.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(99), scm.NewString("later-write")}}, nil, scm.NewNil(), false, nil)
	next, err := r.prepareResource(resource)
	if err != nil {
		t.Fatal(err)
	}
	if next == content {
		t.Fatal("fixture did not replace its content")
	}
	peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 64)}
	if err := r.sendResource(peer, mosiPacket{ID: uuid.NewString(), Resource: resource, Content: content}, transfer); err != nil {
		t.Fatal(err)
	}
	if r.hasTransfers(resource.Database) {
		t.Fatal("completed transfer retained its GC lifetime")
	}
	if _, err := r.captureResource(resource, uuid.NewString()); err == nil || r.hasTransfers(resource.Database) {
		t.Fatal("failed capture leaked a GC lifetime")
	}
	close(peer.outgoing)
	packets := 0
	for message := range peer.outgoing {
		if message.Packet.Content != content {
			t.Fatalf("transfer changed UUID %s -> %s", content, message.Packet.Content)
		}
		packets++
	}
	if packets < 3 {
		t.Fatal("transfer omitted its immutable data")
	}
}

func TestReplicationNativeWriterContinuesWhileForeignDDLHasSchema(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "tnativetxschemacontinue", 3)
	r := replicaRecoveryFixture()
	r.master = false
	r.schemaShards(table.schema, r.id, true)
	r.resourceCell(mosiResource{Database: table.schema.Name}).ownstate.Initialize(mosi.M)
	previous := replication.Swap(r)
	defer replication.Store(previous)
	tx := NewTxContext(TxACID)
	defer tx.Rollback()
	table.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(99), scm.NewString("pending-native-write")}}, nil, scm.NewNil(), false, nil, tx)
	cell := r.shardCell(table.ActiveShards()[0])
	if cell.writesDrained() {
		t.Fatal("native transaction was not registered")
	}
	r.resourceCell(mosiResource{Database: table.schema.Name}).ownstate.Publish(mosi.IO)
	_, release := r.writeResource(*cell.resource.Load(), tx)
	release()
	if !cell.ownstate.Phase().Readable() {
		t.Fatal("DDL wait revoked the existing data transaction")
	}
}

func TestReplicationActiveWriterPreventsPayloadEviction(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "tactivewritereviction", 3)
	r := replicaRecoveryFixture()
	r.schemaShards(table.schema, r.id, true)
	resource := *r.shardCell(table.ActiveShards()[0]).resource.Load()
	if _, err := r.prepareResource(resource); err != nil {
		t.Fatal(err)
	}
	previous := replication.Swap(r)
	defer replication.Store(previous)
	shard := table.ActiveShards()[0]
	access := shard.GetWrite(nil)
	released := false
	defer func() {
		if !released {
			access.Release()
		}
	}()
	var freed [numEvictableTypes]int64
	if shardCleanup(shard, &freed) {
		t.Fatal("eviction detached an admitted writer's payload")
	}
	read := shard.GetRead(nil, false)
	if read.shard.Count() != 3 {
		t.Fatal("active writer blocked its ordinary snapshot reader")
	}
	read.Release()
	access.Release()
	released = true
	if !shardCleanup(shard, &freed) {
		t.Fatal("completed writer retained payload admission")
	}
}

func TestReplicationRebuildForwardsUUIDAndRights(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_uuid_forwarding", 4)
	RebuildTable(table, true, false)
	original := table.ActiveShards()[0]
	r := replicaRecoveryFixture()
	previous := replication.Swap(r)
	defer replication.Store(previous)
	cell := r.declareShard(original, r.id, true)
	access := original.GetRead(nil, false)
	defer access.Release()
	resource := *cell.resource.Load()
	for i := 0; i < 3; i++ {
		before := table.ActiveShards()[0]
		RebuildTable(table, true, false)
		after := table.ActiveShards()[0]
		if after.uuid == before.uuid || before.loadNext() != after {
			t.Fatal("rebuild did not publish a distinct UUID through native forwarding")
		}
		if cell.current.Load() != after || cell.resource.Load().Shard != after.uuid.String() || r.resourceCell(resource) != cell {
			t.Fatal("old UUID did not forward to the successor's authority")
		}
		if _, obsolete := r.directory.Records[resource.key()]; obsolete {
			t.Fatal("predecessor retained an independent permission directory")
		}
		encoded, err := json.Marshal(after)
		if err != nil || string(encoded) != `"`+after.uuid.String()+`"` {
			t.Fatal("shard persistence retained another identity", string(encoded), err)
		}
	}
	if access.shard != original || findScanShard(table, resource) != original {
		t.Fatal("rebuild rebound an existing reader's shard")
	}
	original.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(5), scm.NewString("forwarded")}}, false, nil, nil, false, nil)
	if got := table.ActiveShards()[0].Count(); got != 5 {
		t.Fatalf("write through old UUID reached %d rows, want 5", got)
	}
	r.publishLocal(resource, mosi.IO)
	if cell.ownstate.Phase() != mosi.IO || table.ActiveShards()[0].coherence.Load() != cell {
		t.Fatal("old UUID invalidation did not revoke successor permission")
	}
}

func TestReplicationForwardingPreservesPendingFlight(t *testing.T) {
	r := replicaRecoveryFixture()
	from := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	to := from
	to.Shard = uuid.NewString()
	cell := r.resourceCell(from)
	cell.ownstate.Initialize(mosi.IO)
	pending, _ := cell.ownstate.Request(mosi.S)
	r.forwardResource(from, to)
	if err := r.acceptGrant(mosiPacket{ID: pending.Flight().ID(), Resource: from, State: "S", Epoch: 1}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-pending.Flight().Done():
	default:
		t.Fatal("forwarding lost a pending request")
	}
	if r.resourceCell(to) != cell || cell.ownstate.Phase() != mosi.S || cell.resource.Load().Shard != to.Shard {
		t.Fatal("grant through the old UUID did not authorize the successor")
	}
}

func TestReplicationForwardingJoinsWaitingAliases(t *testing.T) {
	r := replicaRecoveryFixture()
	from := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	to := from
	to.Shard = uuid.NewString()
	source, target := r.resourceCell(from), r.resourceCell(to)
	source.ownstate.Initialize(mosi.IO)
	target.ownstate.Initialize(mosi.IO)
	source.ownstate.Request(mosi.S)
	pending, _ := target.ownstate.Request(mosi.S)
	completed := make(chan *replicaCell, 1)
	go func() { completed <- r.obtain(from, mosi.S, "", nil) }()
	r.forwardResource(from, to)
	if err := r.acceptGrant(mosiPacket{ID: pending.Flight().ID(), Resource: to, State: "S", Epoch: 1}); err != nil {
		t.Fatal(err)
	}
	select {
	case cell := <-completed:
		if cell != target || r.resourceCell(from) != target {
			t.Fatal("waiting alias did not follow the shared grant")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("forwarding stranded a waiting reader")
	}
}

func TestReplicationLateTransferCannotReplaceSuccessor(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_late_uuid_transfer", 3)
	r := replicaRecoveryFixture()
	previous := replication.Swap(r)
	defer replication.Store(previous)
	cell := r.declareShard(table.ActiveShards()[0], r.id, true)
	resource := *cell.resource.Load()
	first, err := r.prepareResource(resource)
	if err != nil {
		t.Fatal(err)
	}
	held := cell.current.Load()
	RebuildTable(table, true, false)
	latest := cell.current.Load()
	transfer, err := r.startImport(mosiPacket{Resource: resource, Content: first})
	if err != nil {
		t.Fatal(err)
	}
	defer transfer.abort()
	if !transfer.knownContent {
		t.Fatal("resend could truncate a retained shard")
	}
	if err := r.finishImport("remote-peer", transfer); err != nil {
		t.Fatal(err)
	}
	if !transfer.superseded || cell.current.Load() != latest || latest == held {
		t.Fatal("late transfer replaced a newer shard")
	}
	runtime.KeepAlive(held)
}

func TestReplicationSchemaDeclarationDoesNotUndoForwarding(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_schema_uuid_forward", 3)
	r := replicaRecoveryFixture()
	original := table.ActiveShards()[0]
	cell := r.declareShard(original, r.id, true)
	from := *cell.resource.Load()
	to := from
	to.Shard = uuid.NewString()
	r.forwardResource(from, to)
	r.declareShard(original, r.id, true)
	if *cell.resource.Load() != to || r.resourceCell(from) != r.resourceCell(to) {
		t.Fatal("schema loading reverted an existing UUID route")
	}
	captured := r.cellSnapshot.Load()
	previousName := table.Name
	defer func() { table.Name = previousName }()
	table.Name = "renamed_items"
	r.declareShard(original, r.id, true)
	renamed := to
	renamed.Table = table.Name
	if r.canonicalResource(from) != renamed || r.canonicalResource(to) != renamed ||
		*cell.resource.Load() != renamed || r.directory.Records[to.key()].Resource != renamed {
		t.Fatal("schema rename did not publish one canonical successor identity")
	}
	if *(*captured)[from.key()].resource != to {
		t.Fatal("schema rename changed an already captured routing snapshot")
	}
}

func TestReplicationCheckpointRoutesBeforeActivePublication(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "replica_checkpoint_routes", 3)
	r := replicaRecoveryFixture()
	previous := replication.Swap(r)
	defer replication.Store(previous)
	source := tbl.ActiveShards()[0]
	cell := r.declareShard(source, r.id, true)
	resource := r.shardResource(source)
	successor := source.rebuild(true)
	// Match overflow rebuild: checkpoint the completed successor first, while
	// ordinary readers still retain the old active topology.
	tbl.mu.Lock()
	tbl.Shards = []*storageShard{successor}
	tbl.publishSchemaTopology(ShardModeFree, tbl.Shards, tbl.PDimensions)
	tbl.mu.Unlock()
	tbl.schema.save()
	if r.lookupCell(r.shardResource(successor)) != nil {
		t.Fatal("fixture already published runtime forwarding")
	}
	transfer, err := r.captureResource(mosiResource{Database: tbl.schema.Name}, "")
	if err != nil {
		t.Fatal(err)
	}
	defer transfer.release()
	if r.lookupCell(r.shardResource(successor)) != cell || r.resourceKey(resource) != r.shardResource(successor).key() {
		t.Fatal("advertised successor UUID lost its existing authority")
	}
	if len(transfer.routes) == 0 || cell.ownstate.Phase() != mosi.M {
		t.Fatal("schema capture omitted UUID routing or changed permission")
	}
}

func TestReplicationRequestLearnsOnlyOwnerForwarding(t *testing.T) {
	for _, owner := range []string{"source", "other"} {
		t.Run(owner, func(t *testing.T) {
			r := replicaRecoveryFixture()
			from := mosiResource{Database: "example", Table: "items", Shard: uuid.NewString()}
			to := from
			to.Shard = uuid.NewString()
			cell := r.resourceCell(from)
			cell.ownstate.Initialize(mosi.IO)
			r.updateRecord(from, func(record *replicaRecord) bool {
				record.Owner, record.State, record.Epoch = owner, "IO", 4
				return true
			})
			err := r.acceptRequestRoutes("source", mosiPacket{Resource: to, Forwards: []shardForward{{from, to}}})
			if owner == "other" {
				if err == nil || r.lookupCell(to) != nil {
					t.Fatal("non-owner redirected another shard", err)
				}
				return
			}
			if err != nil || r.lookupCell(to) != cell || cell.ownstate.Phase() != mosi.IO {
				t.Fatal("owner forwarding lost its cell or fabricated permission", err)
			}
			record := r.directory.Records[to.key()]
			if record.Owner != owner || record.Epoch != 4 {
				t.Fatal("forwarding changed ownership", record)
			}
		})
	}
}

func TestReplicationSharedSchemaKeepsNewerShard(t *testing.T) {
	tbl, engine := createDurabilityTestTable(t, "replica_shared_schema_successor", 3)
	RebuildTable(tbl, true, false)
	older := append([]byte(nil), engine.ReadSchema()...)
	r := replicaRecoveryFixture()
	r.cluster = &nodeCluster{manager: r, self: clusterMember{ID: r.id}}
	previous := replication.Swap(r)
	defer replication.Store(previous)
	source := tbl.ActiveShards()[0]
	cell := r.declareShard(source, r.id, true)
	resource := r.shardResource(source)
	successor := source.rebuild(true)
	if err := r.replaceShard(resource, source, successor); err != nil {
		t.Fatal(err)
	}
	cell.current.Store(successor)
	cell.ownstate.Publish(mosi.O)
	if err := r.installSchema("peer", tbl.schema.Name, engine, older); err != nil {
		t.Fatal(err)
	}
	if current := tbl.schema.tables.Get(tbl.Name).ActiveShards()[0]; current != successor || cell.current.Load() != successor {
		t.Fatal("old schema reference replaced the installed successor")
	}
	if r.canonicalResource(resource).Shard != successor.uuid.String() || cell.ownstate.Phase() != mosi.O {
		t.Fatal("old schema reference changed routing or permission")
	}
}

func TestReplicationStatusResolvesCatalogUUIDToOneAuthority(t *testing.T) {
	r := replicaRecoveryFixture()
	old := mosiResource{Database: "status_routes", Table: "items", Shard: uuid.NewString()}
	current := old
	current.Shard = uuid.NewString()
	cell := r.resourceCell(old)
	cell.ownstate.Establish(mosi.M)
	r.updateRecord(old, func(record *replicaRecord) bool {
		record.Owner, record.State = r.id, "M"
		return true
	})
	r.forwardResource(old, current)
	data, err := r.statusSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		Authority map[string]struct{ Resource mosiResource }
		Records   map[string]replicaRecord
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Authority[old.key()].Resource != current || snapshot.Authority[current.key()].Resource != current {
		t.Fatal("diagnostics lost the canonical UUID")
	}
	if len(snapshot.Records) != 1 || snapshot.Records[current.key()].Owner != r.id {
		t.Fatal("diagnostics duplicated or changed ownership")
	}
}

func TestReplicationAccessRetainsPublishedShardUntilNativeSwap(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "replica_native_swap", 3)
	r := replicaRecoveryFixture()
	previous := replication.Swap(r)
	defer replication.Store(previous)
	source := tbl.ActiveShards()[0]
	cell := r.declareShard(source, r.id, true)
	oldReader := source.GetRead(nil, false)
	defer oldReader.Release()
	successor := source.rebuild(true)
	r.forwardResource(r.shardResource(source), r.shardResource(successor))
	// Model the handoff window: the successor has a prospective native
	// topology, while the table still atomically publishes the source.
	successor.generation.Store(&tableShardTopology{})
	cell.current.Store(successor)
	if got := r.readShard(source, nil); got != source {
		t.Fatal("read selected successor before the native topology swap")
	}
	write := source.GetWrite(nil)
	if write.shard != source {
		write.Release()
		t.Fatal("write skipped the active source before the native swap")
	}
	write.Release()
	// The source remains the write target; native mutation forwarding keeps
	// both physical shards current while readers retain their original input.
	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(99), scm.NewString("after rebuild")}}, nil, scm.NewNil(), false, nil, nil)
	if source.Count() != 4 || successor.Count() != 4 {
		t.Fatal("source write was not forwarded to the successor")
	}
	tbl.mu.Lock()
	tbl.Shards = []*storageShard{successor}
	tbl.publishTopologyLocked()
	tbl.mu.Unlock()
	if r.readShard(source, nil) != successor || oldReader.shard != source {
		t.Fatal("native swap lost current selection or the held historical reader")
	}
	// Model the other side of the swap: the active topology is new while the
	// payload hint is still old. A handover must see transactions on the new
	// input and must not transfer its rights before those transactions finish.
	cell.current.Store(source)
	if r.readShard(source, nil) != successor {
		t.Fatal("fresh read reused an old input after the native swap")
	}
	freshWrite := source.GetWrite(nil)
	if freshWrite.shard != successor {
		freshWrite.Release()
		t.Fatal("fresh write reused an old input after the native swap")
	}
	freshWrite.Release()
	successor.beginTransactionUse()
	if cell.publishedShard() != successor || cell.writesDrained() {
		successor.endTransactionUse()
		t.Fatal("handover ignored the native published successor")
	}
	successor.endTransactionUse()
	if !cell.writesDrained() || oldReader.shard != source {
		t.Fatal("drained transaction or retained old reader was lost")
	}
}
