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
			r := &replicationManager{cells: make(map[string]*replicaCell), journal: replicaJournal{Records: make(map[string]replicaRecord)}}
			r.resourceCell(mosiResource{Database: tbl.schema.Name}).rights.Initialize(mosi.M)
			shard := tbl.ActiveShards()[0]
			r.shardCell(shard).rights.Initialize(mosi.S)
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

func TestReplicationOrderedMutationRetainsTransactionWriteLease(t *testing.T) {
	for _, mode := range []TxMode{TxCursorStability, TxACID} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			tbl, _ := createDurabilityTestTable(t, "torderedtxrights", 3)
			r := &replicationManager{cells: make(map[string]*replicaCell), journal: replicaJournal{Records: make(map[string]replicaRecord)}}
			r.resourceCell(mosiResource{Database: tbl.schema.Name}).rights.Initialize(mosi.M)
			shard := tbl.ActiveShards()[0]
			cell := r.shardCell(shard)
			cell.rights.Initialize(mosi.M)
			previous := replication.Swap(r)
			defer replication.Store(previous)
			tx := NewTxContext(mode)
			defer replicationLeave(tx)
			defer tx.Rollback()
			_, order := integerOrder(false)
			tbl.scan_order(tx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, trueCondition(),
				[]scm.Scmer{scm.NewString("id")}, []func(...scm.Scmer) scm.Scmer{order}, 0, 0, 1,
				[]string{"$update"}, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
					if cell.gate.TryLock() {
						cell.gate.Unlock()
						t.Error("ownership handover was admitted during the mutation callback")
					}
					scm.Apply(args[1])
					return args[0]
				}), scm.NewInt(0), false, scm.NewNil(), nil, scm.NewNil())
			if tx.IsVisible(shard, 0) {
				t.Fatal("transaction did not delete the first row")
			}
			if cell.gate.TryLock() {
				cell.gate.Unlock()
				t.Fatal("ordered scan released the write lease before transaction completion")
			}
			tx.Rollback()
			replicationLeave(tx)
			if shard.deletions.Count() != 0 {
				t.Fatal("rollback did not restore the ordered deletion")
			}
			if !cell.gate.TryLock() {
				t.Fatal("completed transaction retained its write lease")
			}
			cell.gate.Unlock()
		})
	}
}

func TestReplicationOrderedRecSetKeepsHistoricalGeneration(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "torderedhistory", 3)
	RebuildTable(tbl, true, false)
	original := tbl.ActiveShards()[0]
	part := newRecSetShardFromSortedIDs(original, 3, []uint32{0, 1, 2})
	source := &recSet{table: tbl, shards: []recSetShard{part}, count: 3}
	r := &replicationManager{cells: make(map[string]*replicaCell), journal: replicaJournal{Records: make(map[string]replicaRecord)}}
	cell := r.shardCell(original)
	replacement := original.rebuild(true)
	if replacement == original {
		t.Fatal("fixture did not produce a successor")
	}
	cell.current.Store(replacement)
	cell.rights.Initialize(mosi.IO)
	previous := replication.Swap(r)
	defer replication.Store(previous)
	_, order := integerOrder(false)
	got := source.scan_order(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, trueCondition(),
		[]scm.Scmer{scm.NewString("id")}, []func(...scm.Scmer) scm.Scmer{order}, 0, 0, 3,
		[]string{"id"}, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return scm.NewInt(args[0].Int() + args[1].Int()) }),
		scm.NewInt(0), false, scm.NewNil(), nil, scm.NewNil())
	if got.Int() != 6 || cell.rights.Phase() != mosi.IO {
		t.Fatalf("historical sum = %v, current rights = %v", got, cell.rights.Phase())
	}
	if original.cacheReaders.Load() != 0 {
		t.Fatal("historical ordered scan leaked its read capability")
	}
	// Acquiring current M must never reinterpret these historical physical IDs.
	cell.rights.Initialize(mosi.M)
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
	if !cell.gate.TryLock() {
		t.Fatal("generation mismatch leaked its write capability")
	}
	cell.gate.Unlock()
}

func TestReplicationReclaimIncludesEmptyAndColdShards(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "treplicareclaim", 3)
	empty, _ := CreateTable(table.schema.Name, "empty", Safe, false)
	empty.CreateColumn("id", "INT", nil, nil)
	r := &replicationManager{id: "slave", cells: make(map[string]*replicaCell), directory: replicaDirectory{Records: make(map[string]replicaRecord)}}
	schema := mosiResource{Database: table.schema.Name}
	r.resourceCell(schema).rights.Initialize(mosi.M)
	r.schemaShards(table.schema, r.id, true)
	for _, item := range table.schema.tables.GetAll() {
		for _, shard := range item.ActiveShards() {
			r.shardCell(shard).rights.RestoreBackend(mosi.M)
		}
	}
	if !r.reclaimDatabase(table.schema.Name) {
		t.Fatal("current backend-only exclusive generations were not reclaimed")
	}
	for _, item := range table.schema.tables.GetAll() {
		for _, shard := range item.ActiveShards() {
			if r.shardCell(shard).rights.Phase() != mosi.M {
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
	r.resourceCell(mosiResource{Database: "a_offline_tenant"}).rights.Initialize(mosi.S)
	r.resourceCell(mosiResource{Database: table.schema.Name}).rights.Initialize(mosi.M)
	r.schemaShards(table.schema, r.id, true)
	for _, shard := range table.ActiveShards() {
		r.shardCell(shard).rights.RestoreBackend(mosi.M)
	}
	previous := replication.Swap(r)
	defer replication.Store(previous)
	failures := ReclaimReplicationDatabases()
	if len(failures) != 1 || !strings.Contains(failures[0], "a_offline_tenant") || !strings.Contains(failures[0], mosi.NetworkError) {
		t.Fatalf("expected the offline tenant failure only, got %v", failures)
	}
	for _, shard := range table.ActiveShards() {
		if r.shardCell(shard).rights.Phase() != mosi.M {
			t.Fatal("later tenant was not reclaimed after the offline failure")
		}
	}
	if r.resourceCell(mosiResource{Database: "a_offline_tenant"}).rights.Phase() == mosi.M {
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
	r.resourceCell(schema).rights.Initialize(mosi.S)
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
		if r.resourceCell(schema).rights.Phase() != mosi.MR {
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
	if r.resourceCell(schema).rights.Phase() != mosi.M || r.directory.Records[schema.key()].Epoch != 2 {
		t.Fatal("periodic reclaim did not retain the durable upstream grant")
	}
}

func replicaAdmissionFixture(name string) (*replicationManager, *replicaCell, scm.Scmer) {
	r := &replicationManager{cells: make(map[string]*replicaCell)}
	c := r.cell(name)
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
		c.rights.Initialize(mosi.ParsePhase(state))
		c.initialized.Store(true)
		replication.Store(r)
		// Hold every control-plane lock, including the write/handover gate.
		// Already-authorized reads must still complete on their own snapshots.
		r.mu.Lock()
		c.gate.Lock()
		c.acquire.Lock()
		c.directory.Lock()
		c.checkpoint.Lock()
		c.importPublication.Lock()
		done := make(chan struct{})
		go func() {
			var readers sync.WaitGroup
			for i := 0; i < 32; i++ {
				readers.Add(1)
				go func() {
					defer readers.Done()
					tx := &TxContext{}
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
		c.checkpoint.Unlock()
		c.directory.Unlock()
		c.acquire.Unlock()
		c.gate.Unlock()
		r.mu.Unlock()
		<-done
		if timedOut {
			t.Fatalf("%s reads waited for shared control locks", state)
		}
		if c.rights.Phase().String() != state {
			t.Fatalf("%s read wrote the shared flags", state)
		}
	}
}

func BenchmarkReplicationReadAdmission(b *testing.B) {
	old := replication.Load()
	defer replication.Store(old)
	r, c, session := replicaAdmissionFixture("parallel_analytics")
	state := "S"
	c.rights.Initialize(mosi.ParsePhase(state))
	c.initialized.Store(true)
	replication.Store(r)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		tx := &TxContext{}
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
	resource := mosiResource{Database: tbl.schema.Name, Table: tbl.Name, Shard: tbl.ActiveShards()[0].logicalID().String()}
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
	payload := bytes.Repeat([]byte("streamed shard generation\n"), 100000)
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
	resource := mosiResource{Database: table.schema.Name, Table: table.Name, Shard: table.ActiveShards()[0].logicalID().String()}
	if _, err := r.prepareResource(resource); err != nil {
		t.Fatal(err)
	}
	cell := r.resourceCell(resource)
	original := cell.current.Load()
	cell.rights.Initialize(mosi.S)
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
	access := original.GetRead(nil)
	if access.shard != replacement {
		access.Release()
		t.Fatal("ordinary read did not resolve the current generation")
	}
	access.Release()
	// These query-local IDs already own the historical generation. Losing
	// permission to the current shard must not cause projection to fetch or
	// reinterpret newer rows, even when that owner is now offline.
	cell.rights.Publish(mosi.IO)
	access = original.acquireSnapshotRead(nil)
	if access.shard != original {
		access.Release()
		t.Fatal("physical snapshot IDs were rebound to another generation")
	}
	access.Release()
	original.RemoveFromDisk()
	reader := engine.ReadColumn(original.uuid.String(), "id")
	defer reader.Close()
	if missing, ok := reader.(interface{ Missing() bool }); ok && missing.Missing() {
		t.Fatal("retirement deleted a reachable snapshot's column file")
	}
	if _, _, complete := CleanDatabase(table.schema); complete {
		t.Fatal("cleanup ignored historical generation ownership")
	}
	runtime.KeepAlive(original)
}

func TestReplicationCleanupRetainsCompletedBackup(t *testing.T) {
	table, engine := createDurabilityTestTable(t, "treplicabackupgc", 3)
	r := &replicationManager{master: true, id: "master", cells: make(map[string]*replicaCell), directory: replicaDirectory{Records: make(map[string]replicaRecord)}}
	resource := mosiResource{Database: table.schema.Name, Table: table.Name, Shard: table.ActiveShards()[0].logicalID().String()}
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
	table.mu.Lock()
	table.Shards = []*storageShard{pending}
	table.publishTopologyLocked()
	table.mu.Unlock()
	cell := r.shardCell(pending)
	cell.current.Store(pending)
	cell.rights.Initialize(mosi.IO)
	r.directory.Records[resource.key()] = replicaRecord{Resource: resource, Owner: "slave", State: "IO", Epoch: 1, Generation: backup.uuid.String()}
	r.backups = map[string]replicaBackup{resource.key(): {Resource: resource, Generation: backup.uuid.String()}}
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
	if err := r.restoreBackup(r.directory.Records[resource.key()]); err != nil {
		t.Fatal(err)
	}
	cell.rights.Publish(mosi.M)
	access := cell.current.Load().GetRead(nil)
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
					Resource:   mosiResource{Database: db.Name, Table: table.Name, Shard: uuid.NewString()},
					Generation: uuid.NewString(), Kind: "object", ObjectKind: "column", Object: "payload",
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
	resource := mosiResource{Database: table.schema.Name, Table: table.Name, Shard: table.ActiveShards()[0].logicalID().String()}
	generation, err := r.prepareResource(resource)
	if err != nil {
		t.Fatal(err)
	}
	r.directory.Records[resource.key()] = replicaRecord{Resource: resource, Owner: "slave", State: "IO", Generation: generation}
	read := func() []byte {
		t.Helper()
		reader := engine.ReadColumn(generation, "payload")
		defer reader.Close()
		data, err := io.ReadAll(reader)
		if err != nil || len(data) == 0 {
			t.Fatalf("completed column is missing: %v", err)
		}
		return data
	}
	before := read()
	for _, complete := range []bool{false, true} {
		packet := mosiPacket{ID: uuid.NewString(), Resource: resource, Generation: generation, Kind: "object", ObjectKind: "column", Object: "payload"}
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
			t.Fatal("resend truncated a completed generation on object open")
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
	resource := mosiResource{Database: table.schema.Name, Table: table.Name, Shard: table.ActiveShards()[0].logicalID().String()}
	if _, err := r.prepareResource(resource); err != nil {
		t.Fatal(err)
	}
	// This unpublished fixture has no historical query owners.
	r.resourceCell(resource).retiredShards.Store(nil)
	packet := mosiPacket{ID: "pending", Resource: resource, Generation: uuid.NewString(), Kind: "object", ObjectKind: "column", Object: "payload"}
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
	reader := engine.ReadColumn(packet.Generation, "payload")
	if missing, ok := reader.(interface{ Missing() bool }); ok && missing.Missing() {
		reader.Close()
		t.Fatal("cleanup deleted the incoming generation")
	}
	reader.Close()
	transfer.abort()
	r.imports.Delete("slave/pending")
	table.schema.persistenceLifecycle.Lock()
	cleanShards(table.schema)
	table.schema.persistenceLifecycle.Unlock()
	reader = engine.ReadColumn(packet.Generation, "payload")
	defer reader.Close()
	if missing, ok := reader.(interface{ Missing() bool }); !ok || !missing.Missing() {
		t.Fatal("aborted private generation was retained after import cleanup")
	}
}
