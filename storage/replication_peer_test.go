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

import "fmt"
import "io"
import "bytes"
import "compress/gzip"
import "sync"
import "time"
import "context"
import "strings"
import "testing"
import "runtime"
import "net/http"
import "encoding/json"
import "encoding/base64"
import "net/http/httptest"
import "github.com/google/uuid"
import "github.com/gorilla/websocket"
import "github.com/launix-de/memcp/scm"

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
		c.rights.initialize(parseMosiPhase(state))
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
		if c.rights.phase().String() != state {
			t.Fatalf("%s read wrote the shared flags", state)
		}
	}
}

func BenchmarkReplicationReadAdmission(b *testing.B) {
	old := replication.Load()
	defer replication.Store(old)
	r, c, session := replicaAdmissionFixture("parallel_analytics")
	state := "S"
	c.rights.initialize(parseMosiPhase(state))
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
	accepted := make(chan *replicationPeer, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, req, nil)
		if err != nil {
			return
		}
		accepted <- newReplicationPeer(conn, func(ctx context.Context, method string, data json.RawMessage) (any, error) {
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
	client := newReplicationPeer(conn, func(ctx context.Context, method string, data json.RawMessage) (any, error) { return string(data), nil }, nil)
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
	r := &replicationManager{cells: make(map[string]*replicaCell), journal: replicaJournal{Records: make(map[string]replicaRecord)}}
	resource := replicaResource{Database: tbl.schema.Name, Table: tbl.Name, Shard: tbl.ActiveShards()[0].logicalID().String()}
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
	peer := &replicationPeer{done: make(chan struct{}), outgoing: make(chan replicationMessage, 8)}
	finished := make(chan error, 1)
	payload := bytes.Repeat([]byte("streamed shard generation\n"), 100000)
	go func() {
		finished <- peer.sendObject(replicaPacket{ID: "stream"}, "column", "payload", io.NopCloser(bytes.NewReader(payload)))
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
		decoded, err := decodeReplicaChunk(*packet)
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
	writer.Write(make([]byte, replicaPacketBytes+1))
	writer.Close()
	if _, err := decodeReplicaChunk(replicaPacket{Payload: expanded.Bytes()}); err == nil {
		t.Fatal("unbounded expanded chunk accepted")
	}
}

func TestReplicationSnapshotKeepsGenerationAndFiles(t *testing.T) {
	table, engine := createDurabilityTestTable(t, "treplicasnapshot", 3)
	r := &replicationManager{cells: make(map[string]*replicaCell), journal: replicaJournal{Records: make(map[string]replicaRecord)}}
	resource := replicaResource{Database: table.schema.Name, Table: table.Name, Shard: table.ActiveShards()[0].logicalID().String()}
	if _, err := r.prepareResource(resource); err != nil {
		t.Fatal(err)
	}
	cell := r.resourceCell(resource)
	original := cell.current.Load()
	cell.rights.initialize(mosiS)
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
	cell.rights.publish(mosiIO)
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
	r := &replicationManager{master: true, id: "master", cells: make(map[string]*replicaCell), journal: replicaJournal{Records: make(map[string]replicaRecord)}}
	resource := replicaResource{Database: table.schema.Name, Table: table.Name, Shard: table.ActiveShards()[0].logicalID().String()}
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
	cell.rights.initialize(mosiIO)
	r.journal.Records[resource.key()] = replicaRecord{Resource: resource, Owner: "slave", State: "IO", Epoch: 1, Generation: backup.uuid.String()}
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
	if err := r.restoreBackup(r.journal.Records[resource.key()]); err != nil {
		t.Fatal(err)
	}
	cell.rights.publish(mosiM)
	access := cell.current.Load().GetRead(nil)
	defer access.Release()
	value := access.shard.getColumnStorageOrPanic("payload", false, nil).GetValue(0)
	if value.String() != "row-00000001" {
		t.Fatalf("completed backup was not restored: %v", value)
	}
}

func TestReplicationRepeatedTransferKeepsCompletedColumns(t *testing.T) {
	table, engine := createDurabilityTestTable(t, "treplicaresend", 3)
	r := &replicationManager{master: true, id: "master", cells: make(map[string]*replicaCell), journal: replicaJournal{Records: make(map[string]replicaRecord)}}
	resource := replicaResource{Database: table.schema.Name, Table: table.Name, Shard: table.ActiveShards()[0].logicalID().String()}
	generation, err := r.prepareResource(resource)
	if err != nil {
		t.Fatal(err)
	}
	r.journal.Records[resource.key()] = replicaRecord{Resource: resource, Owner: "slave", State: "IO", Generation: generation}
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
		packet := replicaPacket{ID: uuid.NewString(), Resource: resource, Generation: generation, Kind: "object", ObjectKind: "column", Object: "payload"}
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
	r := &replicationManager{master: true, id: "master", cells: make(map[string]*replicaCell), journal: replicaJournal{Records: make(map[string]replicaRecord)}}
	resource := replicaResource{Database: table.schema.Name, Table: table.Name, Shard: table.ActiveShards()[0].logicalID().String()}
	if _, err := r.prepareResource(resource); err != nil {
		t.Fatal(err)
	}
	// This unpublished fixture has no historical query owners.
	r.resourceCell(resource).retiredShards.Store(nil)
	packet := replicaPacket{ID: "pending", Resource: resource, Generation: uuid.NewString(), Kind: "object", ObjectKind: "column", Object: "payload"}
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
