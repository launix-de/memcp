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
		c.state.Store(&state)
		replication.Store(r)
		// Hold every control-plane lock, including the write/handover gate.
		// Already-authorized reads must still complete on their own snapshots.
		r.mu.Lock()
		c.gate.Lock()
		c.acquire.Lock()
		c.directory.Lock()
		c.checkpoint.Lock()
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
		c.checkpoint.Unlock()
		c.directory.Unlock()
		c.acquire.Unlock()
		c.gate.Unlock()
		r.mu.Unlock()
		<-done
		if timedOut {
			t.Fatalf("%s reads waited for shared control locks", state)
		}
		if c.state.Load() != &state {
			t.Fatalf("%s read wrote the shared flags", state)
		}
	}
}

func BenchmarkReplicationReadAdmission(b *testing.B) {
	old := replication.Load()
	defer replication.Store(old)
	r, c, session := replicaAdmissionFixture("parallel_analytics")
	state := "S"
	c.state.Store(&state)
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

func TestReplicationOldSnapshotSurvivesImportAndRebuild(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "treplicationsnapshot", 3)
	dump, err := exportReplicaDatabase(tbl.schema.Name)
	if err != nil {
		t.Fatal(err)
	}
	if err := importReplicaDatabase(tbl.schema.Name, dump); err != nil {
		t.Fatal(err)
	}
	previous := GetDatabase(tbl.schema.Name).GetTable("items").ActiveShards()[0]
	lease := previous.GetRead(nil)
	defer lease.Release()
	r, _, _ := replicaAdmissionFixture(tbl.schema.Name)
	oldManager := replication.Load()
	replication.Store(r)
	defer replication.Store(oldManager)
	// A running reader owns the old snapshot; importing and rebuilding the
	// successor may proceed without draining that query or adding a read lock.
	if err := importReplicaDatabase(tbl.schema.Name, dump); err != nil {
		t.Fatal(err)
	}
	current := GetDatabase(tbl.schema.Name)
	current.GetTable("items").Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(4), scm.NewString("new-row")}}, nil, scm.NewNil(), false, nil)
	if errors, complete := rebuildDatabaseAndCompact(current, true, false, false); len(errors) != 0 || !complete {
		t.Fatalf("rebuild: %v, complete=%v", errors, complete)
	}
	if _, _, complete := CleanDatabase(current); complete {
		t.Fatal("cleanup ignored a reachable old catalog")
	}
	reader := previous.ColumnReaderTx(nil, "payload", false, ReadCompressed|readOperatorOwned)
	defer ReleaseColumnReader(reader)
	if got := reader.GetValue(0).String(); got != "row-00000001" {
		t.Fatalf("old snapshot changed: %s", got)
	}
	runtime.KeepAlive(previous)
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
		})
	}))
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	client := newReplicationPeer(conn, func(ctx context.Context, method string, data json.RawMessage) (any, error) { return string(data), nil })
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
	tbl.maintenanceMu.Lock()
	locked := true
	defer func() {
		if locked {
			tbl.maintenanceMu.Unlock()
		}
	}()
	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() { data, err := exportReplicaDatabase(tbl.schema.Name); done <- result{data, err} }()
	select {
	case outcome := <-done:
		t.Fatalf("snapshot skipped a busy table: %v", outcome.err)
	case <-time.After(30 * time.Millisecond):
	}
	tbl.maintenanceMu.Unlock()
	locked = false
	var outcome result
	select {
	case outcome = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("snapshot did not resume after maintenance")
	}
	if outcome.err != nil {
		t.Fatal(outcome.err)
	}
	if err := importReplicaDatabase(tbl.schema.Name, outcome.data); err != nil {
		t.Fatal(err)
	}
	loaded := GetDatabase(tbl.schema.Name).GetTable("items")
	add := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewInt(a[0].Int() + a[1].Int()) })
	schema := scm.NewSlice([]scm.Scmer{newScanAccessHeader(0, scanAccessConsumerScan, 0, -1)})
	sum := loaded.scan(nil, schema, nil, nil, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), []string{"id"}, add, scm.NewInt(0), add, false)
	if sum.Int() != 6 {
		t.Fatalf("snapshot lost rows from skipped maintenance: %v", sum)
	}
}
