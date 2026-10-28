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
import "time"
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
	return remoteScanTask{Version: scanWireVersion, Shard: shard, FilterColumns: []string{"id"}, Filter: encode(`(lambda (id) (> id 1))`), MapColumns: []string{"id"}, MapReducer: encode(`(lambda (acc id) (+ acc id))`), Neutral: json.RawMessage(`"0"`)}
}
func TestRemoteScanRPCExecutesReductionWithNativeExtent(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tremotescanrpc", 4)
	RebuildTable(tbl, true, false)
	peer, _, _ := setupRemoteScanConnection(t)
	var parts []remoteScanShard
	if err := peer.scanService.control(context.Background(), peer, "scan.describe", remoteScanDescribe{scanWireVersion, tbl.schema.Name, tbl.Name}, &parts); err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 {
		t.Fatal("expected one shard")
	}
	// Describing a generation does not freeze its append-only delta. The native
	// operator captures its input extent when execution starts.
	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(1000), scm.NewString("future")}}, nil, scm.NewNil(), false, nil)
	task := makeRemoteScanTask(t, parts[0])
	var response remoteScanResult
	if err := peer.scanService.control(context.Background(), peer, "scan.execute", task, &response); err != nil {
		t.Fatal(err)
	}
	value, err := schemeValue(response.Value).reconstruct()
	if err != nil {
		t.Fatal(err)
	}
	if value.Int() != 1009 || response.OutCount != 4 || response.Shard != parts[0] {
		t.Fatalf("wrong reduction or task identity: %+v", response)
	}

	task.Shard.Resource.Shard = uuid.NewString()
	if err := peer.scanService.control(context.Background(), peer, "scan.execute", task, &response); err == nil || !strings.Contains(err.Error(), "shard disappeared") {
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

// A factory is a normal serialized closure in the existing initializer slot.
// Each remote invocation constructs its own state; captures are materialized
// through the same decoder as the filter and reducer before execution begins.
func TestRemoteScanInitializerFactory(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tremotescaninitializer", 4)
	peer, _, _ := setupRemoteScanConnection(t)
	var parts []remoteScanShard
	if err := peer.scanService.control(context.Background(), peer, "scan.describe", remoteScanDescribe{scanWireVersion, tbl.schema.Name, tbl.Name}, &parts); err != nil {
		t.Fatal(err)
	}
	factory := scm.Eval(scm.Read(t.Name(), `(begin (define initial 10) (lambda () initial))`), &scm.Globalenv)
	for i := 0; i < 2; i++ {
		task := makeRemoteScanTask(t, parts[0])
		var err error
		encoded, failure := freezeScanValue(factory, new(scm.SerializationContext))
		task.Neutral, err = json.RawMessage(encoded), failure
		if err != nil {
			t.Fatal(err)
		}
		var response remoteScanResult
		if err := peer.scanService.control(context.Background(), peer, "scan.execute", task, &response); err != nil {
			t.Fatal(err)
		}
		value, err := schemeValue(response.Value).reconstruct()
		if err != nil || value.Int() != 19 {
			t.Fatalf("result=%s err=%v", scm.String(value), err)
		}
	}
}

func TestRemoteScanTransportsListsAndRejectsWrongSecret(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tremotescanreject", 2)
	peer, _, endpoint := setupRemoteScanConnection(t)
	var parts []remoteScanShard
	if err := peer.scanService.control(context.Background(), peer, "scan.describe", remoteScanDescribe{scanWireVersion, tbl.schema.Name, tbl.Name}, &parts); err != nil {
		t.Fatal(err)
	}
	task := makeRemoteScanTask(t, parts[0])
	task.Neutral = json.RawMessage(`"'()"`)
	data, err := scm.MarshalProcedure(scm.Eval(scm.Read(t.Name(), `(lambda (acc id) (append acc (list id)))`), &scm.Globalenv))
	if err != nil {
		t.Fatal(err)
	}
	task.MapReducer = data
	var response remoteScanResult
	if err := peer.scanService.control(context.Background(), peer, "scan.execute", task, &response); err != nil {
		t.Fatal(err)
	}
	list, err := schemeValue(response.Value).reconstruct()
	if err != nil || len(list.Slice()) != 1 || list.Slice()[0].Slice()[0].Int() != 2 {
		t.Fatal("list result changed", list, err)
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
	value, err := schemeValue(result.Value).reconstruct()
	if err != nil || value.Int() != 9 {
		t.Fatalf("wrong return channel result: %v, %v", value, err)
	}
	task := makeRemoteScanTask(t, parts[0])

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
	data, err := freezeScanValue(scm.NewFastDict(dict))
	if err != nil {
		t.Fatal(err)
	}
	result, err := schemeValue(data).reconstruct()
	if err != nil {
		t.Fatal(err)
	}
	if !scm.Equal(scm.Apply(scm.Globalenv.Vars[scm.Symbol("get_assoc")], result, key), scm.NewSlice([]scm.Scmer{scm.NewInt(42), scm.NewNil()})) {
		t.Fatal("native grouped partial lost its domain/payload", result)
	}
	if _, err := freezeScanValue(scm.NewAny(new(int))); err == nil {
		t.Fatal("runtime pointer entered the partial-result codec")
	}
}

func TestRemoteScanAccessKeepsBoundedNULLAndMandatoryConstraints(t *testing.T) {
	registerScanBoundaryFormats()
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
	data, err := freezeScanValue(parsed)
	if err != nil {
		t.Fatal(err)
	}
	value, err := schemeValue(data).reconstruct()
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
	access := held.GetRead(nil, false)
	defer access.Release()
	peer, _, _ := setupRemoteScanConnection(t)
	var parts []remoteScanShard
	if err := peer.scanService.control(context.Background(), peer, "scan.describe", remoteScanDescribe{scanWireVersion, table.schema.Name, table.Name}, &parts); err != nil {
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
	if err := peer.scanService.control(context.Background(), peer, "scan.execute", makeRemoteScanTask(t, parts[0]), &result); err != nil {
		t.Fatal(err)
	}
	value, err := schemeValue(result.Value).reconstruct()
	if err != nil || value.Int() != 1009 || result.OutCount != 4 || result.Shard.Resource.Shard != held.uuid.String() {
		t.Fatalf("remote scan lost its retained generation or forwarded inserts: %+v, %v", result, err)
	}
	if current.state() != COLD {
		t.Fatal("historical scan loaded an unrelated successor payload")

	}
}

func TestProcedureWireNestedScanContextRebindSkipsQuotedData(t *testing.T) {
	input := scm.Read(t.Name(), `(lambda () (list (scan nil source) (scan (node_query_context "previous-hop") source) (quote (scan nil literal))))`)
	binding := scm.Read(t.Name(), `(node_query_context "current-hop")`)
	bound := scm.SerializeToString(bindProcedureScanContext(input, binding), nil)
	if strings.Count(bound, "current-hop") != 2 || strings.Contains(bound, "previous-hop") || !strings.Contains(bound, "(scan nil literal)") {
		t.Fatalf("nested scope rebinding changed quoted data or retained its parent: %s", bound)
	}
}

func TestRemoteScanNativeExistsAndBatch(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_remote_operator_contract", 4)
	RebuildTable(table, true, false)
	peer, _, _ := setupRemoteScanConnection(t)
	var parts []remoteScanShard
	if err := peer.scanService.control(context.Background(), peer, "scan.describe", remoteScanDescribe{scanWireVersion, table.schema.Name, table.Name}, &parts); err != nil {
		t.Fatal(err)
	}
	// Specialized kernels capture their extent at execution, just like native
	// reduction; inserts before dispatch belong to that same generation.
	table.ActiveShards()[0].Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(500), scm.NewString("before-dispatch")}}, false, nil, nil, false, nil)
	procedure := func(source string) json.RawMessage {
		result, err := scm.MarshalProcedure(scm.Eval(scm.Read(t.Name(), source), &scm.Globalenv))
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	for _, test := range []struct {
		filter string
		match  bool
	}{
		{`(lambda (id) (> id 1))`, true},
		{`(lambda (id) (> id 100))`, true}, // newly appended delta row
		{`(lambda (id) (> id 500))`, false},
	} {
		task := makeRemoteScanTask(t, parts[0])

		task.Operator = shardExists
		task.Filter = procedure(test.filter)
		var result remoteScanResult
		if err := peer.scanService.control(context.Background(), peer, "scan.execute", task, &result); err != nil {
			t.Fatal(err)
		}
		value, err := schemeValue(result.Value).reconstruct()
		if err != nil || scm.ToBool(value) != test.match {
			t.Fatal("remote EXISTS lost native match/miss semantics", result, err)
		}
	}
	task := makeRemoteScanTask(t, parts[0])

	task.Stride = 2
	for _, value := range []int64{2, 10, 4, 20} {
		frozen, err := freezeScanValue(scm.NewInt(value))
		if err != nil {
			t.Fatal(err)
		}
		task.Batch = append(task.Batch, frozen)
	}
	task.FilterColumns = []string{"id", "#0"}
	task.Filter = procedure(`(lambda (id key) (equal? id key))`)
	task.MapColumns = []string{"id", "#1"}
	task.MapReducer = procedure(`(lambda (acc id delta) (+ acc id delta))`)
	task.Access.Boundaries = []scm.Scmer{newScanBoundarySpec("id", EqualMatcher, -2, -2, true, true, "", false, -1, nil, nil, "", true)}
	var result remoteScanResult
	if err := peer.scanService.control(context.Background(), peer, "scan.execute", task, &result); err != nil {
		t.Fatal(err)
	}
	value, err := schemeValue(result.Value).reconstruct()
	if err != nil || value.Int() != 36 {
		t.Fatal("remote batch lost row bindings", value, result, err)
	}
}

func TestRemoteScanComputedAndBatchBoundaryRoundTrip(t *testing.T) {
	mapper := scm.Eval(scm.Read(t.Name(), `(lambda (id) (mod id 7))`), &scm.Globalenv)
	descriptor := scm.NewSlice([]scm.Scmer{scm.NewString("id-mod-7"), mapper})
	access := scanAccess{runtime: &scanAccessRuntime{suffix: scanAccessSegment{
		items:  []scm.Scmer{newScanBoundarySpec("", EqualMatcher, -2, -2, true, true, "", false, 0, []string{"id"}, nil, "", true)},
		values: []scm.Scmer{descriptor},
	}}}
	frozen, err := freezeScanAccess(access, new(scm.SerializationContext))
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
	bound, err := received.bind(scm.ProcedureDecode{})
	if err != nil {
		t.Fatal(err)
	}
	columns, restored := bound.boundaryMap(0)
	boundary, _ := bound.boundaryParts(0)
	if boundary.LowerSlot() != -2 || boundary.UpperSlot() != -2 || bound.boundaryColumn(0) != "id-mod-7" || len(columns) != 1 || columns[0] != "id" || scm.Apply(restored, scm.NewInt(16)).Int() != 2 {
		t.Fatal("computed boundary/batch key lost its native contract", string(data))
	}
}

func TestRemoteScanNativeLookupKinds(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_remote_lookup_contract", 4)
	RebuildTable(table, true, false)
	peer, _, _ := setupRemoteScanConnection(t)
	var parts []remoteScanShard
	if err := peer.scanService.control(context.Background(), peer, "scan.describe", remoteScanDescribe{scanWireVersion, table.schema.Name, table.Name}, &parts); err != nil {
		t.Fatal(err)
	}
	table.ActiveShards()[0].Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(99), scm.NewString("before-dispatch")}}, false, nil, nil, false, nil)
	for _, operator := range []shardScanOperator{shardLookupOne, shardLookupMany, shardLookupMapOne, shardLookupMapMany} {
		for _, key := range []int64{2, 99, 100} {
			task := makeRemoteScanTask(t, parts[0])
			task.Operator = operator
			task.MapColumns = []string{"id"}
			var err error
			task.Access, err = freezeScanAccess(exactScanAccess(newExactScanAccessSchema([]string{"id"}), []scm.Scmer{scm.NewInt(key)}))
			if err != nil {
				t.Fatal(err)
			}
			if operator >= shardLookupMapOne {

				neutral, failure := freezeScanValue(scm.NewNil())
				task.Neutral, err = json.RawMessage(neutral), failure
				if err != nil {
					t.Fatal(err)
				}
			}
			var result remoteScanResult
			if err := peer.scanService.control(context.Background(), peer, "scan.execute", task, &result); err != nil {
				t.Fatal(err)
			}
			value, err := schemeValue(result.Value).reconstruct()
			if err != nil {
				t.Fatal(err)
			}
			if key == 100 {
				if result.OutCount != 0 {
					t.Fatal("lookup miss lost native cardinality", result)
				}
			} else {
				if operator >= shardLookupMapOne {
					value = value.Slice()[0]
				}
				if result.OutCount != 1 || value.Int() != key {
					t.Fatal("lookup changed projected value/cardinality", value, result)
				}
			}
		}
	}
}

// A remote ordered run uses the native sort, retains its captured physical IDs,
// and feeds bounded pages into the same coordinator merge. Projection callbacks
// execute only on selected rows; the worker releases its payload before paging.
func TestRemoteScanOrderedRunPagesAndNativeExtent(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tremoteorderedrun", 700)
	RebuildTable(tbl, true, false)
	source := tbl.ActiveShards()[0]
	peer, _, _ := setupRemoteScanConnection(t)
	var parts []remoteScanShard
	if err := peer.scanService.control(context.Background(), peer, "scan.describe", remoteScanDescribe{scanWireVersion, tbl.schema.Name, tbl.Name}, &parts); err != nil {
		t.Fatal(err)
	}
	source.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(9999), scm.NewString("future")}}, false, nil, nil, false, nil)
	executor := &peerScanExecutor{peer: peer, ctx: context.Background(), prepared: parts[0]}
	stub := &storageShard{uuid: source.uuid, t: tbl}
	target := &shardScanTarget{resource: parts[0].Resource, executor: executor}
	stub.scanTarget.Store(target)
	_, direction := persistableTestOrder(true)
	calls := 0
	maxBuffered := 0
	var queue *shardqueue
	callback := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		calls++
		maxBuffered = max(maxBuffered, len(queue.remote.rows))
		return scm.NewInt(args[0].Int() + args[1].Int())
	})
	spec := scanOrderTableSpec{remoteBatchSize: 256, table: tbl, condition: scm.Eval(scm.Read(t.Name(), "(lambda () true)"), &scm.Globalenv), sortcols: []scm.Scmer{scm.NewString("id")}, callbackCols: []string{"id"}, callback: callback, perTableOffset: -1, perTableLimit: -1}
	queue = delegateOrderedQueue(nil, stub, scanAccess{}, &spec, []func(...scm.Scmer) scm.Scmer{direction}, 0, 5, 605, shardReadDelegation{target})
	if source.cacheReaders.Load() != 0 {
		t.Fatal("remote run retained native payload after preparation")
	}
	// A current routing stub is not the identity of these retained physical IDs.
	captured := queue.orderedInputUUID()
	queue.shard = &storageShard{uuid: uuid.New(), t: tbl}
	if queue.orderedInputUUID() != captured {
		t.Fatal("routing metadata rebound the captured generation")
	}
	pageCalls := 0 // the first page arrived with the run reference
	original := queue.remote.run.handle.Call
	queue.remote.run.handle.Call = func(ctx context.Context, request scm.ClusterRequest) (scm.Scmer, error) {
		if request.Method == "ack" {
			pageCalls++
		}
		return original(ctx, request)
	}
	value, _, _ := consumeDirectScanOrder(queue, &spec, nil, 0, 5, 600, scm.NewInt(0), false, scm.NewNil())
	if pageCalls != 2 {
		t.Fatalf("paging made %d roundtrips, want two requests after the piggybacked page", pageCalls)
	}
	// Descending 9999,700..1: skip five, consume 696..97. The insert before
	// execution participates in the native ordered input.
	if calls != 600 || value.Int() != int64((696+97)*600/2) {
		t.Fatalf("calls=%d sum=%v", calls, value)
	}
	if maxBuffered > 2*spec.remoteBatchSize {
		t.Fatalf("coordinator accumulated %d projected rows beyond its two batch windows", maxBuffered)
	}
	if queue.remote.next <= 256 {
		t.Fatal("ordered scan did not consume multiple bounded pages")
	}
	if queue.remote.pending != nil || queue.remote.cancel != nil || queue.remote.rows != nil {
		t.Fatal("ordered scan did not finish and join its shard job")
	}
}

func TestRemoteOrderedBatchFanoutAndCredit(t *testing.T) {
	if orderedScanBatchSize(1, -1) != maxOrderedBatch || orderedScanBatchSize(1000, -1) != minOrderedBatch || orderedScanBatchSize(1, 3) != minOrderedBatch {
		t.Fatal("ordered window did not adapt to fanout and LIMIT")
	}
	run := &orderedScanRun{shard: &storageShard{uuid: uuid.New()}, batchSize: 16, columns: []string{"id"}, universe: 64}
	for id := 0; id < 64; id++ {
		run.rows = append(run.rows, scm.NewSlice([]scm.Scmer{scm.NewInt(int64(id)), scm.NewInt(int64(id))}))
	}
	view, err := scm.ClusterObjectValue(run, context.Background(), scm.ClusterRequest{Method: "view"})
	if err != nil {
		t.Fatal(err)
	}
	ack := func(consumed, start int) scm.ClusterRequest {
		return scm.ClusterRequest{Method: "ack", Arguments: []scm.Scmer{scm.NewInt(int64(consumed)), scm.NewInt(int64(start))}}
	}
	if _, err := run.HandleClusterRequest(context.Background(), ack(7, 16)); err == nil {
		t.Fatal("job accepted credit before half the batch was consumed")
	}
	started := make(chan scm.ClusterRequest, 1)
	allow := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	remote := &remoteOrderQueue{ctx: ctx, cancel: cancel, batchSize: 16, total: 64, columns: map[string]int{"id": 0}, rows: make(map[uint32][]scm.Scmer)}
	remote.run = &remoteOrderedRun{scm.ClusterHandle{View: view, Call: func(ctx context.Context, request scm.ClusterRequest) (scm.Scmer, error) {
		if request.Method == "ack" {
			started <- request
			select {
			case <-allow:
			case <-ctx.Done():
				return scm.NewNil(), ctx.Err()
			}
		}
		return scm.ClusterObjectValue(run, ctx, request)
	}}}
	defer remote.close()
	queue := &shardqueue{remote: remote, universe: 64}
	remote.fetch(queue)
	for i := 0; i < 7; i++ {
		queue.consumeOrdered(1, true)
	}
	if remote.pending != nil {
		t.Fatal("coordinator requested a successor before the half-window")
	}
	queue.consumeOrdered(1, true)
	select {
	case request := <-started:
		if request.Arguments[0].Int() != 8 || request.Arguments[1].Int() != 16 {
			t.Fatal("ACK did not identify consumed credit and the next batch")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("half-window consumption did not prefetch")
	}
	// A delayed peer must not prevent consumption of the remaining current batch.
	for i := 0; i < 7; i++ {
		queue.consumeOrdered(1, true)
	}
	if len(queue.items) != 1 || len(remote.rows) != 1 {
		t.Fatal("coordinator retained already discarded rows")
	}
	close(allow)
	queue.consumeOrdered(1, true)
	if len(queue.items) != 16 || queue.items[0] != 16 || remote.pending != nil {
		t.Fatal("successor batch was not installed in order")
	}
	if _, err := run.HandleClusterRequest(ctx, ack(8, 16)); err == nil {
		t.Fatal("job accepted a repeated ACK")
	}
}

func TestRemoteOrderedLimitJoinsPendingBatch(t *testing.T) {
	run := &orderedScanRun{shard: &storageShard{uuid: uuid.New()}, batchSize: 16, universe: 64}
	for id := 0; id < 64; id++ {
		run.rows = append(run.rows, scm.NewSlice([]scm.Scmer{scm.NewInt(int64(id))}))
	}
	view, err := scm.ClusterObjectValue(run, context.Background(), scm.ClusterRequest{Method: "view"})
	if err != nil {
		t.Fatal(err)
	}
	started, joined := make(chan struct{}), make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	remote := &remoteOrderQueue{ctx: ctx, cancel: cancel, batchSize: 16, total: 64, columns: map[string]int{}, rows: make(map[uint32][]scm.Scmer)}
	remote.run = &remoteOrderedRun{scm.ClusterHandle{View: view, Call: func(ctx context.Context, request scm.ClusterRequest) (scm.Scmer, error) {
		if request.Method == "ack" {
			close(started)
			<-ctx.Done()
			close(joined)
			return scm.NewNil(), ctx.Err()
		}
		return scm.ClusterObjectValue(run, ctx, request)
	}}}
	queue := &shardqueue{remote: remote, universe: 64}
	remote.fetch(queue)
	queue.consumeOrdered(8, true)
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("successor receive did not start")
	}
	queue.closeReaders() // the same cleanup used for LIMIT, errors and cancellation
	select {
	case <-joined:
	default:
		t.Fatal("LIMIT released the queue before joining its pending receive")
	}
	if !run.closed || run.rows != nil || remote.pending != nil || remote.rows != nil {
		t.Fatal("finished job retained buffered answers")
	}
	if _, err := run.HandleClusterRequest(context.Background(), scm.ClusterRequest{Method: "view"}); err == nil {
		t.Fatal("finished shard job was reopened")
	}
	queue.closeReaders() // idempotent cleanup after nested error unwinding
}

func TestRemoteScanPortableKeyPredicate(t *testing.T) {
	Init(scm.Globalenv)
	filter := scm.Eval(scm.Read(t.Name(), "(lambda (value) (> value 10))"), &scm.Globalenv)
	predicate := newScanKeyPredicate([]scanJoinOrderDriverConstraint{{positions: []int{0}, keys: [][]scm.Scmer{{scm.NewInt(2)}, {scm.NewInt(4)}}}}, filter, 1)
	encoded, err := scm.MarshalProcedure(predicate)
	if err != nil {
		t.Fatalf("%v; wire=%s", err, encoded)
	}
	if !strings.Contains(string(encoded), "scan_key_predicate") {
		t.Fatal("key predicate lost portable constructor")
	}
	restored, err := scm.UnmarshalProcedure(encoded, scm.ProcedureDecode{})
	if err != nil {
		t.Fatalf("%v; wire=%s", err, encoded)
	}
	for _, sample := range []struct {
		key, extra int64
		want       bool
	}{{2, 11, true}, {2, 9, false}, {3, 11, false}, {4, 12, true}} {
		if got := scm.ToBool(scm.Apply(restored, scm.NewInt(sample.key), scm.NewInt(sample.extra))); got != sample.want {
			t.Errorf("%+v = %v", sample, got)
		}
	}
	if scm.ToBool(scm.Apply(restored, scm.NewNil(), scm.NewInt(11))) {
		t.Fatal("NULL matched a join key")
	}
}

func TestRemoteScanLargeDictionaryUsesBulkSerialize(t *testing.T) {
	dict := scm.NewFastDictValue(256)
	for i := 0; i < 256; i++ {
		dict.Set(scm.NewSlice([]scm.Scmer{scm.NewNil(), scm.NewInt(int64(i))}), scm.NewSlice([]scm.Scmer{scm.NewInt(int64(i)), scm.NewNil()}), nil)
	}
	encoded, err := freezeScanValue(scm.NewFastDict(dict))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(encoded), "set_assoc") != 1 {
		t.Fatal("dictionary was not serialized as one bulk construction")
	}
	value, err := encoded.reconstruct()
	if err != nil {
		t.Fatal(err)
	}
	if !value.IsFastDict() {
		t.Fatal("large dictionary reconstructed as a linear association")
	}
	for i := 0; i < 256; i++ {
		key := scm.NewSlice([]scm.Scmer{scm.NewNil(), scm.NewInt(int64(i))})
		got, ok := value.FastDict().Get(key)
		if !ok || got.Slice()[0].Int() != int64(i) || !got.Slice()[1].IsNil() {
			t.Fatal("tuple/NULL semantics changed", i, got)
		}
	}
}

func TestRemoteOperationCannotReplayAfterHistoryEviction(t *testing.T) {
	var registry scanTaskRegistry
	first, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.begin(context.Background(), first.String()); err != nil {
		t.Fatal(err)
	}
	registry.finish(first.String())
	for i := 0; i < 2048; i++ {
		id, err := uuid.NewV7()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := registry.begin(context.Background(), id.String()); err != nil {
			t.Fatal(err)
		}
		registry.finish(id.String())
	}
	if _, err := registry.begin(context.Background(), first.String()); err == nil {
		t.Fatal("evicted operation became executable again")
	}
}

func TestRemoteScanReconnectRejectsOldOperationIdentity(t *testing.T) {
	first, client, endpoint := setupRemoteScanConnection(t)
	if first.operationPrefix == "" {
		t.Fatal("query connection lacks its authenticated binding")
	}
	operation, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	oldID := first.operationPrefix + "/" + operation.String()
	first.close()
	first.wait()
	next, _, err := client.connect(context.Background(), endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer client.forget(next)
	if next.objectPeer != first.objectPeer || next.operationPrefix == first.operationPrefix {
		t.Fatal("reconnect lost incarnation identity or reused an old operation domain")
	}
	var output any
	request := struct {
		Return, Method string
		Data           json.RawMessage
	}{oldID, "object.release", json.RawMessage(`{}`)}
	if err := next.call(context.Background(), "node.control", request, &output); err == nil || !strings.Contains(err.Error(), "invalid operation identity") {
		t.Fatal("old operation entered the new connection", err)
	}
}
