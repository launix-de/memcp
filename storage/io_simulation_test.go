/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
This program is free software under the GNU General Public License, version 3
or later, WITHOUT ANY WARRANTY. See <https://www.gnu.org/licenses/>.
*/
package storage

import "io"
import "fmt"
import "math"
import "sync"
import "time"
import "bytes"
import "context"
import "testing"
import "net/http"
import "encoding/json"
import "net/http/httptest"
import "github.com/gorilla/websocket"
import "github.com/launix-de/memcp/scm"

func TestIOSimulationValidatesAndCreditsElapsedWork(t *testing.T) {
	for _, limits := range [][2]float64{{-1, 0}, {0, -1}, {math.NaN(), 0}, {0, math.Inf(1)}} {
		if _, err := newIOSimulation(limits[0], limits[1], new(ioCounters)); err == nil {
			t.Fatal("accepted invalid limits", limits)
		}
	}
	stats := new(ioCounters)
	profile, _ := newIOSimulation(8, 0, stats) // one MB/s
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !profile.transfer(context.Background(), 10000, time.Now().Add(-time.Second)) || stats.waitNS.Load() != 0 {
		t.Fatal("charged already elapsed work")
	}
	if profile.transfer(ctx, 100000, time.Now()) {
		t.Fatal("bandwidth wait ignored cancellation")
	}
	zero, _ := newIOSimulation(0, 0, new(ioCounters))
	if !zero.transfer(context.Background(), 100000000, time.Now()) || zero.stats.waitNS.Load() != 0 {
		t.Fatal("fullspeed injected a delay")
	}
}

func TestIOSimulationSharesBackendBandwidthAcrossStreams(t *testing.T) {
	profile, _ := newIOSimulation(8, 0, new(ioCounters))
	start := time.Now()
	var workers sync.WaitGroup
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			profile.transfer(context.Background(), 60000, start)
		}()
	}
	workers.Wait()
	if time.Since(start) < 115*time.Millisecond {
		t.Fatal("parallel streams bypassed the shared one-MB/s backend budget")
	}
}

func TestClusterReceiveConnectionsShareNodeBandwidth(t *testing.T) {
	profile, _ := newIOSimulation(8, 0, new(ioCounters))
	previous := clusterReceiveProfile.Swap(profile)
	defer clusterReceiveProfile.Store(previous)
	first, second := activeClusterReceiveIO(), activeClusterReceiveIO()
	start := time.Now()
	var workers sync.WaitGroup
	for _, connection := range []*ioSimulation{first, second} {
		workers.Add(1)
		go func(connection *ioSimulation) {
			defer workers.Done()
			connection.transfer(context.Background(), 60000, start)
		}(connection)
	}
	workers.Wait()
	if time.Since(start) < 115*time.Millisecond {
		t.Fatal("adding a peer multiplied the simulated node's ingress bandwidth")
	}
}

func TestClusterReceiveThrottlesMessagesAndJoinsCancellation(t *testing.T) {
	profile, _ := newIOSimulation(1, 40, new(ioCounters))
	previous := clusterReceiveProfile.Swap(profile)
	defer clusterReceiveProfile.Store(previous)
	serverPeer := make(chan *mosiPeer, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cluster" {
			w.Write([]byte("ordinary HTTP"))
			return
		}
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverPeer <- newMosiPeer(conn, func(context.Context, string, json.RawMessage) (any, error) { return true, nil }, nil)
	}))
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+server.URL[4:]+"/cluster", nil)
	if err != nil {
		t.Fatal(err)
	}
	client := newMosiPeer(conn, nil, nil)
	peer := <-serverPeer
	defer func() { client.close(); peer.close(); client.wait(); peer.wait() }()
	// Only native /cluster WebSocket messages enter this instrumentation.
	response, err := http.Get(server.URL + "/sql/probe")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if profile.stats.operations.Load() != 0 {
		t.Fatal("ordinary HTTP was throttled")
	}
	started := time.Now()
	if err := client.call(context.Background(), "echo", true, nil); err != nil {
		t.Fatal(err)
	}
	if time.Since(started) < 80*time.Millisecond {
		t.Fatal("round trip missed receiver delay on one direction")
	}
	if profile.stats.operations.Load() != 2 || profile.stats.waitNS.Load() == 0 {
		t.Fatal("receiver diagnostics missing")
	}
	// Idle readers must not build transmission credit. Two large messages
	// must still pay the first frame before consuming the next header.
	time.Sleep(250 * time.Millisecond)
	started = time.Now()
	payload := string(bytes.Repeat([]byte("x"), 24000))
	if err := client.call(context.Background(), "echo", payload, nil); err != nil {
		t.Fatal(err)
	}
	if err := client.call(context.Background(), "echo", payload, nil); err != nil {
		t.Fatal(err)
	}
	if time.Since(started) < 192*time.Millisecond {
		t.Fatal("bandwidth did not pace message headers")
	}
	ready := make(chan struct{})
	peer.workers.Add(1)
	go func() { defer peer.workers.Done(); close(ready); peer.waitDispatch(time.Now().Add(time.Hour)) }()
	<-ready
	peer.close()
	peer.wait() // must join, rather than leak a timer until its deadline
}

func TestFakeRemoteDurabilityStreamingAndAbort(t *testing.T) {
	profile, _ := newIOSimulation(8, 2, new(ioCounters))
	engine := &FakeRemote{FileStorage: &FileStorage{path: t.TempDir() + "/"}, simulation: profile}
	engine.WriteSchema([]byte(`{"tables":{}}`))
	if string(engine.ReadSchema()) != `{"tables":{}}` {
		t.Fatal("schema changed")
	}
	writer := engine.WriteColumn("shard", "value")
	data := bytes.Repeat([]byte("data"), 24000)
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	finishColumnWrite(writer, true)
	reader := engine.ReadColumn("shard", "value")
	actual, err := io.ReadAll(reader)
	reader.Close()
	if err != nil || !bytes.Equal(actual, data) {
		t.Fatal("column stream changed", err)
	}
	blob := engine.WriteBlob("abcd")
	blob.Write([]byte("must not be published"))
	abortPersistenceWrite(blob)
	missing := engine.ReadBlob("abcd")
	defer missing.Close()
	if !missing.(interface{ Missing() bool }).Missing() {
		t.Fatal("abort published buffered blob")
	}
	log := engine.OpenLog("rows")
	log.Write(LogEntryInsert{cols: []string{"id"}, values: [][]scm.Scmer{{scm.NewInt(7)}}})
	log.Flush(true)
	log.Close()
	_, entries, reopened := engine.ReplayLog("rows")
	count := 0
	for range entries {
		count++
	}
	reopened.Close()
	if count != 1 {
		t.Fatal("WAL replay changed", count)
	}
	if profile.stats.bytes.Load() < uint64(2*len(data)) || profile.stats.waitNS.Load() == 0 {
		t.Fatal("backend bypassed transfer accounting")
	}
	// SQL backend options arrive as strings, persisted JSON may use numbers.
	old := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = old }()
	configured := BackendRegistry["FakeRemote"]("memcp-tests", json.RawMessage(`{"backend":"FakeRemote","latency_ms":"2","bandwidth_mbps":8}`))
	if configured.BackendName() != "FakeRemote" {
		t.Fatal("backend declaration lost")
	}
	configured.WriteSchema([]byte(`{"tables":{}}`))
	if !bytes.Equal(configured.ReadSchema(), []byte(`{"tables":{}}`)) {
		t.Fatal("configured backend persistence failed")
	}
}

// Default-sized columns exceed the backend reader's buffer. Deserializers
// must consume each payload completely even when a Read returns a short chunk.
func TestFakeRemoteColumnRoundTrip(t *testing.T) {
	const count = 60000
	for _, fixture := range []struct {
		name   string
		column ColumnStorage
		fresh  func() ColumnStorage
		value  func(uint32) scm.Scmer
	}{
		{"int", new(StorageInt), func() ColumnStorage { return new(StorageInt) }, func(i uint32) scm.Scmer { return scm.NewInt(int64(i * 2654435761)) }},
		{"float", new(StorageFloat), func() ColumnStorage { return new(StorageFloat) }, func(i uint32) scm.Scmer { return scm.NewFloat(float64(i)*1.25 + 0.125) }},
		{"string", new(StorageString), func() ColumnStorage { return new(StorageString) }, func(i uint32) scm.Scmer {
			return scm.NewString(fmt.Sprintf("row/%08x/value/%08x", i*2654435761, i*2246822519))
		}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			profile, _ := newIOSimulation(0, 0, new(ioCounters))
			engine := &FakeRemote{FileStorage: &FileStorage{path: t.TempDir() + "/"}, simulation: profile}
			column := fixture.column
			column.prepare()
			for i := uint32(0); i < count; i++ {
				column.scan(i, fixture.value(i))
			}
			column.init(count)
			for i := uint32(0); i < count; i++ {
				column.build(i, fixture.value(i))
			}
			column.finish()
			writer := engine.WriteColumn("shard", fixture.name)
			column.Serialize(writer)
			finishColumnWrite(writer, true)
			reader := engine.ReadColumn("shard", fixture.name)
			defer reader.Close()
			var magic [1]byte
			if _, err := io.ReadFull(reader, magic[:]); err != nil {
				t.Fatal(err)
			}
			loaded := fixture.fresh()
			if n := loaded.Deserialize(reader); n != count {
				t.Fatalf("count=%d", n)
			}
			for i := uint32(0); i < count; i++ {
				if !scm.Equal(loaded.GetValue(i), column.GetValue(i)) {
					t.Fatalf("row %d: got %v, want %v", i, loaded.GetValue(i), column.GetValue(i))
				}
			}
			t.Run("truncated", func(t *testing.T) {
				var encoded bytes.Buffer
				column.Serialize(&encoded)
				defer func() {
					if failure := recover(); failure != io.ErrUnexpectedEOF && failure != io.EOF {
						t.Fatalf("truncated column must fail with EOF, got %v", failure)
					}
				}()
				fixture.fresh().Deserialize(bytes.NewReader(encoded.Bytes()[1 : encoded.Len()-1]))
			})
		})
	}
}
