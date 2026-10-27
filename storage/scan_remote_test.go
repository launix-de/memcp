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
import "context"
import "strings"
import "testing"
import "net/http"
import "encoding/json"
import "net/http/httptest"
import "github.com/google/uuid"
import "github.com/gorilla/websocket"
import "github.com/launix-de/memcp/scm"

const testScanSecret = "scan-test-secret-32-bytes-constant"

func setupRemoteScanConnection(t *testing.T) (*mosiPeer, *nodeScanService, string) {
	t.Helper()
	old := nodeScans.Swap(nil)
	if err := startNodeScans(testScanSecret); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/cluster", serveMosiHTTP)
	server := httptest.NewServer(mux)
	client := &nodeScanService{id: uuid.NewString(), secret: []byte(testScanSecret), peers: make(map[*mosiPeer]bool)}
	peer, _, err := client.connect(context.Background(), server.URL+"/cluster")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		client.forget(peer)
		client.stopObjects()
		StopNodeScans()
		nodeScans.Store(old)
		server.Close()
	})
	return peer, client, server.URL + "/cluster"
}
func makeRemoteScanTask(t *testing.T, shard remoteScanShard) remoteScanTask {
	t.Helper()
	encode := func(source string) json.RawMessage {
		data, err := scm.MarshalProcedure(scm.Eval(scm.Read(t.Name(), source), &scm.Globalenv))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	return remoteScanTask{Version: scanWireVersion, Task: 1, Attempt: uuid.NewString(), Visibility: "cursor-stability/captured-horizon", Shard: shard, FilterColumns: []string{"id"}, Filter: encode(`(lambda (id) (> id 1))`), MapColumns: []string{"id"}, MapReducer: encode(`(lambda (acc id) (+ acc id))`), Neutral: json.RawMessage(`0`), ResultKind: "scalar"}
}
func TestRemoteScanRPCExecutesReductionWithCapturedHorizon(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tremotescanrpc", 4)
	RebuildTable(tbl, true, false)
	peer, _, _ := setupRemoteScanConnection(t)
	var parts []remoteScanShard
	if err := peer.call(context.Background(), "scan.describe", remoteScanDescribe{scanWireVersion, tbl.schema.Name, tbl.Name}, &parts); err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 {
		t.Fatal("expected one shard")
	}
	// A future insert after the described scan extent must not enter this task.
	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(1000), scm.NewString("future")}}, nil, scm.NewNil(), false, nil)
	task := makeRemoteScanTask(t, parts[0])
	var response remoteScanResult
	if err := peer.call(context.Background(), "scan.execute", task, &response); err != nil {
		t.Fatal(err)
	}
	value, err := decodeScanData(response.Value, false)
	if err != nil {
		t.Fatal(err)
	}
	if value.Int() != 9 || response.OutCount != 3 || response.Shard != parts[0] || response.Attempt != task.Attempt {
		t.Fatalf("wrong reduction or task identity: %+v", response)
	}
	if err := peer.call(context.Background(), "scan.execute", task, &response); err == nil || !strings.Contains(err.Error(), "replayed") {
		t.Fatalf("duplicate task: %v", err)
	}
	task.Task++
	task.Attempt = uuid.NewString()
	task.Shard.Resource.Shard = uuid.NewString()
	if err := peer.call(context.Background(), "scan.execute", task, &response); err == nil || !strings.Contains(err.Error(), "shard disappeared") {
		t.Fatalf("stale generation: %v", err)
	}
	for _, shard := range tbl.ActiveShards() {
		if shard.activeScanners.Load() != 0 || shard.cacheReaders.Load() != 0 {
			t.Fatal("RPC leaked a scan or payload lease")
		}
	}
	if tbl.activeTopology().operations.Load() != 0 {
		t.Fatal("RPC leaked topology registration")
	}
}
func TestRemoteScanRejectsRawRowsAndWrongSecret(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tremotescanreject", 2)
	peer, _, endpoint := setupRemoteScanConnection(t)
	var parts []remoteScanShard
	if err := peer.call(context.Background(), "scan.describe", remoteScanDescribe{scanWireVersion, tbl.schema.Name, tbl.Name}, &parts); err != nil {
		t.Fatal(err)
	}
	task := makeRemoteScanTask(t, parts[0])
	task.Neutral = json.RawMessage(`[]`)
	data, err := scm.MarshalProcedure(scm.Eval(scm.Read(t.Name(), `(lambda (acc id) (append acc (list id)))`), &scm.Globalenv))
	if err != nil {
		t.Fatal(err)
	}
	task.MapReducer = data
	var response remoteScanResult
	if err := peer.call(context.Background(), "scan.execute", task, &response); err == nil || !strings.Contains(err.Error(), "cannot return rows") {
		t.Fatalf("raw rows accepted: %v", err)
	}
	wrong := &nodeScanService{id: uuid.NewString(), secret: []byte("different-node-secret"), peers: make(map[*mosiPeer]bool)}
	if rejected, _, err := wrong.connect(context.Background(), endpoint); err == nil {
		wrong.forget(rejected)
		t.Fatal("wrong node secret accepted")
	}
	// An unauthenticated node may receive a public challenge, never the secret
	// or a scan result. Sending a task instead of proof closes that connection.
	dialer := *websocket.DefaultDialer
	dialer.Subprotocols = []string{mosiSubprotocol}
	address, _ := mosiEndpointURL(endpoint)
	conn, _, err := dialer.Dial(address, http.Header{"X-Memcp-Role": []string{"scan"}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, challenge, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(challenge), testScanSecret) {
		t.Fatal("challenge leaked the node secret")
	}
	if err := scanAuthMessage(conn, "scan.execute", task); err != nil {
		t.Fatal(err)
	}
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("unauthenticated scan received a response")
	}
}

func TestClusterAuthenticationRejectsReplayedProof(t *testing.T) {
	_, _, endpoint := setupRemoteScanConnection(t)
	address, _ := mosiEndpointURL(endpoint)
	dialer := *websocket.DefaultDialer
	dialer.Subprotocols = []string{mosiSubprotocol}
	dial := func() *websocket.Conn {
		conn, _, err := dialer.Dial(address, http.Header{"X-Memcp-Role": []string{"scan"}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Close() })
		return conn
	}
	first := dial()
	var original scanChallenge
	if err := readScanAuth(first, "scan.challenge", &original); err != nil {
		t.Fatal(err)
	}
	client := scanChallenge{Version: scanWireVersion, Node: uuid.NewString(), Nonce: uuid.NewString()}
	client.Proof = scanAuthProof([]byte(testScanSecret), "client", original, client)
	first.Close()
	second := dial()
	var fresh scanChallenge
	if err := readScanAuth(second, "scan.challenge", &fresh); err != nil {
		t.Fatal(err)
	}
	if fresh.Nonce == original.Nonce {
		t.Fatal("authentication reused its server nonce")
	}
	if err := scanAuthMessage(second, "scan.proof", client); err != nil {
		t.Fatal(err)
	}
	if _, _, err := second.ReadMessage(); err == nil {
		t.Fatal("a proof for an older challenge authenticated a fresh connection")
	}
}

func TestRemoteScanMessagesHaveNoRPCHandle(t *testing.T) {
	peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 1)}
	id := uuid.NewString()
	if err := postScanMessage(peer, "scan.cancel", id); err != nil {
		t.Fatal(err)
	}
	message := <-peer.outgoing
	if message.ID != 0 || message.Method != "scan.cancel" || len(peer.pending) != 0 {
		t.Fatal("fire-and-forget message allocated an RPC handle")
	}
	var decoded string
	if err := json.Unmarshal(message.Data, &decoded); err != nil || decoded != id {
		t.Fatal("return channel identity changed")
	}
}

func TestRemoteScanReturnChannelDeliversResultsAndPanics(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_scan_return_channel", 4)
	peer, service, _ := setupRemoteScanConnection(t)
	var parts []remoteScanShard
	if err := service.control(context.Background(), peer, "scan.describe", remoteScanDescribe{scanWireVersion, tbl.schema.Name, tbl.Name}, &parts); err != nil {
		t.Fatal(err)
	}
	executor := &peerScanExecutor{peer: peer, ctx: context.Background()}
	result := executor.dispatch(makeRemoteScanTask(t, parts[0]))
	value, err := decodeScanData(result.Value, false)
	if err != nil || value.Int() != 9 {
		t.Fatalf("wrong return channel result: %v, %v", value, err)
	}
	task := makeRemoteScanTask(t, parts[0])
	task.Task++
	task.MapReducer, err = scm.MarshalProcedure(scm.Eval(scm.Read(t.Name(), `(lambda (acc id) (error "worker panic"))`), &scm.Globalenv))
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			failure := recover()
			if failure == nil || !strings.Contains(fmt.Sprint(failure), "worker panic") {
				t.Fatalf("worker panic not forwarded: %v", failure)
			}
		}()
		executor.dispatch(task)
	}()
	service.returns.Range(func(key, value any) bool {
		t.Errorf("completed return channel retained: %v", key)
		return true
	})
	peer.mu.Lock()
	defer peer.mu.Unlock()
	if len(peer.pending) != 0 {
		t.Fatal("remote computation retained an RPC handle")
	}
}

func TestRemoteScanCancellationBeforeDispatchAndAfterCompletion(t *testing.T) {
	var registry scanTaskRegistry
	id := uuid.NewString()
	registry.cancelTask(id)
	if _, err := registry.begin(context.Background(), id); err == nil {
		t.Fatal("cancel-before-dispatch admitted work")
	}
	id = uuid.NewString()
	scope, err := registry.begin(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	registry.cancelTask(id)
	if scope.ctx.Err() != context.Canceled {
		t.Fatal("active scope did not forward cancellation")
	}
	registry.finish(id)
	registry.cancelTask(id)
	if _, err := registry.begin(context.Background(), id); err == nil {
		t.Fatal("late cancellation permitted replay")
	}
	for i := 0; i < 2048; i++ {
		registry.cancelTask(uuid.NewString())
	}
	if len(registry.active) != 0 || len(registry.terminal) != 1024 {
		t.Fatalf("unbounded cancellation state: %d/%d", len(registry.active), len(registry.terminal))
	}
}

func TestRemoteScanNativePartialValuesKeepTupleKeysAndNULL(t *testing.T) {
	key := scm.NewSlice([]scm.Scmer{scm.NewNil(), scm.NewInt(7)})
	dict := scm.NewFastDictValue(1)
	dict.Set(key, scm.NewSlice([]scm.Scmer{scm.NewInt(42), scm.NewNil()}), nil)
	data, err := encodeScanResult(scm.NewFastDict(dict), "value")
	if err != nil {
		t.Fatal(err)
	}
	result, err := decodeScanValue(data, "value")
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsFastDict() || !scm.Equal(scm.NewSlice(result.FastDict().ToList()), scm.NewSlice(dict.ToList())) {
		t.Fatal("native grouped partial lost its domain/payload", result)
	}
	if _, err := encodeScanResult(scm.NewAny(new(int)), "value"); err == nil {
		t.Fatal("runtime pointer entered the partial-result codec")
	}
}

func TestRemoteScanAccessKeepsBoundedNULLAndMandatoryConstraints(t *testing.T) {
	// An unbounded endpoint and a bound SQL NULL must remain distinguishable.
	// Losing mandatory/null-safe flags could otherwise widen the remote scan.
	schema := []scm.Scmer{
		newScanAccessHeader(1, "value", 0, -1),
		newScanBoundarySpec("id", EqualMatcher, 0, -1, true, false, "", true, -1, nil, nil, "", true),
	}
	access := exactScanAccess(schema, []scm.Scmer{scm.NewNil()})
	access.plannerFilterCovered = true
	frozen, err := freezeScanAccess(access)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(frozen)
	if err != nil {
		t.Fatal(err)
	}
	var received remoteScanAccess
	if err := json.Unmarshal(data, &received); err != nil {
		t.Fatal(err)
	}
	bound, err := received.bind()
	if err != nil {
		t.Fatal(err)
	}
	boundary, _ := bound.boundaryParts(0)
	if bound.len() != 1 || boundary.ColumnName() != "id" || !boundary.Mandatory() || !boundary.NullSafe() || boundary.UpperSlot() != -1 || !bound.boundValue(0, false).IsNil() || !bound.runtime.filterCovered {
		t.Fatal("remote access lost its native constraints", string(data))
	}
	received.Boundaries[0] = newScanBoundarySpec("id", EqualMatcher, 4, -1, true, false, "", true, -1, nil, nil, "", true)
	if _, err := received.bind(); err == nil {
		t.Fatal("accepted boundary outside its operand frame")
	}
}

func TestRemoteScanPartialStripsParserLocationsBeforeChoosingCodec(t *testing.T) {
	parsed := scm.Read(t.Name(), `(42 (7))`)
	data, err := encodeScanResult(parsed, "value")
	if err != nil {
		t.Fatal(err)
	}
	value, err := decodeScanValue(data, "value")
	if err != nil {
		t.Fatal(err)
	}
	want := scm.NewSlice([]scm.Scmer{scm.NewInt(42), scm.NewSlice([]scm.Scmer{scm.NewInt(7)})})
	if !scm.Equal(value, want) {
		t.Fatal("parser metadata changed the reduction value", value)
	}
}

func TestRemoteScanRetainsUUIDAcrossRebuild(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_remote_uuid_rebuild", 4)
	RebuildTable(table, true, false)
	r := replicaRecoveryFixture()
	previous := replication.Swap(r)
	defer replication.Store(previous)
	held := table.ActiveShards()[0]
	r.declareShard(held, r.id, true)
	access := held.GetRead(nil)
	defer access.Release()
	peer, _, _ := setupRemoteScanConnection(t)
	var parts []remoteScanShard
	if err := peer.call(context.Background(), "scan.describe", remoteScanDescribe{scanWireVersion, table.schema.Name, table.Name}, &parts); err != nil {
		t.Fatal(err)
	}
	table.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(1000), scm.NewString("after-horizon")}}, nil, scm.NewNil(), false, nil)
	RebuildTable(table, true, false)
	current := table.ActiveShards()[0]
	if held.uuid == current.uuid {
		t.Fatal("fixture did not rebuild")
	}
	var freed [numEvictableTypes]int64
	if !shardCleanup(current, &freed) || current.state() != COLD {
		t.Fatal("successor did not become cold")
	}
	var result remoteScanResult
	if err := peer.call(context.Background(), "scan.execute", makeRemoteScanTask(t, parts[0]), &result); err != nil {
		t.Fatal(err)
	}
	value, err := decodeScanData(result.Value, false)
	if err != nil || value.Int() != 9 || result.OutCount != 3 || result.Shard.Resource.Shard != held.uuid.String() {
		t.Fatalf("remote scan lost its old shard/horizon: %+v, %v", result, err)
	}
	if current.state() != COLD {
		t.Fatal("historical scan loaded an unrelated successor payload")
	}
}
