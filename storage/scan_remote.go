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
import "sync/atomic"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

// Version 6 confirms both installation and cluster permission in a reference ACK.
const scanWireVersion = 6

type shardScanHorizon struct {
	Main  uint32 `json:"main"`
	Delta uint32 `json:"delta"`
}
type remoteScanShard struct {
	Schema   uint64           `json:"schema"`
	Resource mosiResource     `json:"resource"`
	Horizon  shardScanHorizon `json:"horizon"`
}
type remoteScanDescriptor struct {
	remoteScanShard
	Object *scm.ClusterReference `json:"object,omitempty"`
}

type remoteScanDescription []remoteScanDescriptor

func (description remoteScanDescription) ClusterObjectOffers() []scm.ClusterReference {
	var result []scm.ClusterReference
	for _, part := range description {
		if part.Object != nil {
			result = append(result, scanWireReferences(scanWireValue{Kind: "object", Reference: part.Object})...)
		}
	}
	return result
}

type remoteScanDescribe struct {
	Version  int    `json:"version"`
	Database string `json:"database"`
	Table    string `json:"table"`
}

type remoteScanTask struct {
	Input         *scm.ClusterReference `json:"input,omitempty"`
	Transaction   *scm.ClusterReference `json:"transaction,omitempty"`
	Access        remoteScanAccess      `json:"access"`
	Version       int                   `json:"version"`
	Task          uint64                `json:"task"`
	Attempt       string                `json:"attempt"`
	Visibility    string                `json:"visibility"`
	Shard         remoteScanShard       `json:"shard"`
	FilterColumns []string              `json:"filter_columns"`
	Filter        json.RawMessage       `json:"filter"`
	MapColumns    []string              `json:"map_columns"`
	MapReducer    json.RawMessage       `json:"mapreducer"`
	Neutral       json.RawMessage       `json:"neutral"`
	ResultKind    string                `json:"result_kind"`
}
type remoteScanResult struct {
	Rejected       bool            `json:"rejected,omitempty"` // capacity refusal before executing scan callbacks
	Version        int             `json:"version"`
	Task           uint64          `json:"task"`
	Attempt        string          `json:"attempt"`
	Shard          remoteScanShard `json:"shard"`
	Kind           string          `json:"kind"`
	Value          json.RawMessage `json:"value"`
	OutCount       int64           `json:"out_count"`
	InputCount     int64           `json:"input_count"`
	CandidateCount int64           `json:"candidate_count"`
}

// A connection owns a bounded replay window. Old task numbers are rejected,
// never forgotten into an executable state. No rights or task state is persisted.
type scanRPCSession struct {
	peer      *mosiPeer
	peerReady chan struct{}
	tasks     scanTaskRegistry
	service   *nodeScanService // immutable connection-owned role adapter
	remoteID  string           // authenticated identity, never taken from an RPC assertion
	mu        sync.Mutex
	high      uint64
	seen      [1024]uint64
}

func newScanRPCSession() *scanRPCSession {
	return &scanRPCSession{peerReady: make(chan struct{})}
}
func (s *scanRPCSession) bindPeer(peer *mosiPeer) { s.peer = peer; close(s.peerReady) }
func (s *scanRPCSession) claim(task uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task == 0 || task <= s.high && s.high-task >= uint64(len(s.seen)) || s.seen[task%uint64(len(s.seen))] == task {
		return fmt.Errorf("replayed or expired scan task")
	}
	if task > s.high {
		s.high = task
	}
	s.seen[task%uint64(len(s.seen))] = task
	return nil
}
func (s *scanRPCSession) handle(ctx context.Context, method string, data json.RawMessage) (any, error) {

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Control traffic (membership, owner discovery and residency hints) must
	// not consume compute slots. Saturated shard execution is neither evidence
	// of a dead owner nor a reason to reject the protocol needed to finish it.
	if method == "node.control" || method == "scan.result" || method == "scan.cancel" {
		<-s.peerReady
		return s.channelMessage(ctx, method, data)
	}
	if strings.HasPrefix(method, "object.") {
		return s.handleObject(ctx, method, data)
	}
	if strings.HasPrefix(method, "cluster.") {
		if s.service == nil || s.service.cluster == nil {
			return nil, fmt.Errorf("cluster membership is not enabled")
		}
		return s.service.cluster.HandleClusterRequest(context.WithValue(ctx, clusterObjectSessionKey{}, s), scm.ClusterRequest{Method: method, Payload: data})
	}

	switch method {
	case "scan.describe":
		var request remoteScanDescribe
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		object := GetDatabase(request.Database)
		if object == nil {
			return nil, fmt.Errorf("unknown remote database")
		}
		return object.HandleClusterRequest(context.WithValue(ctx, clusterObjectSessionKey{}, s), scm.ClusterRequest{Method: method, Payload: request})
	case "scan.execute":
		var request remoteScanTask
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		object := GetDatabase(request.Shard.Resource.Database)
		if object == nil {
			return nil, fmt.Errorf("unknown remote database")
		}
		return object.HandleClusterRequest(context.WithValue(ctx, clusterObjectSessionKey{}, s), scm.ClusterRequest{Method: method, Payload: request})
	}
	return nil, fmt.Errorf("unknown remote scan method")
}

// The database handler owns the sole scan preparation path, including shared-
// backend tasks and optionally retained historical inputs. It reconstructs TX
// and captures once, acquires one native read generation, then delegates the
// prepared invocation to that shard's HandleClusterRequest. No transport adapter
// prepares, reacquires or directly executes a scan.
func (database *database) executeScanTask(ctx context.Context, s *scanRPCSession, request remoteScanTask, input *storageShard) (any, error) {
	if request.Version != scanWireVersion || request.Visibility != "cursor-stability/captured-horizon" || !request.Shard.Resource.valid() || request.Shard.Resource.Shard == "" {
		return nil, fmt.Errorf("invalid remote scan contract")
	}
	if _, err := uuid.Parse(request.Attempt); err != nil {
		return nil, fmt.Errorf("invalid scan attempt")
	}
	if request.ResultKind != "scalar" && request.ResultKind != "group_sums" && request.ResultKind != "value" {
		return nil, fmt.Errorf("unsupported remote result kind")
	}
	if err := s.claim(request.Task); err != nil {
		return nil, err
	}
	for _, cols := range [][]string{request.FilterColumns, request.MapColumns} {
		for _, col := range cols {
			if strings.HasPrefix(col, "$") {
				return nil, fmt.Errorf("remote scan rejects query-local or mutation columns")
			}
		}
	}
	// Request ownership is created once outside shard item loops. Existing
	// scheduling-boundary checks observe cancellation; nested scans inherit this
	// context and forward cancellation to their own remote return channels.
	id, _ := ctx.Value(remoteScanContextKey{}).(string)
	if id == "" {
		id = uuid.NewString()
	}
	ss := &scm.SessionState{}
	seq := ss.BeginQuery("Cluster", "remote shard reduction")
	ss.SetQueryContext(seq, ctx)
	tx := NewTxContext(TxCursorStability)
	tx.autoCommit = true
	if request.Transaction != nil {
		value, err := s.service.importObject(ctx, s.peer, *request.Transaction)
		if err != nil {
			return nil, err
		}
		origin, ok := value.Any().(*TxContext)
		if !ok || (origin.Mode != TxCursorStability && origin.Mode != TxACID) {
			return nil, fmt.Errorf("invalid transaction object")
		}
		// The peer endpoint is shared, never its mutable executor. Even if two
		// subscans import the same serialized capture, beginQuery/endQuery must
		// belong to distinct native participants with independent cancellation.
		if _, err := origin.BindClusterInvocation(scm.NewAny(tx)); err != nil {
			return nil, err
		}
	}
	tx.beginQuery(ss, seq, "remote shard reduction")
	defer tx.endQuery(seq)
	stop := context.AfterFunc(ctx, func() { ss.KillQuery(seq) })
	defer stop()
	s.service.executions.Store(id, tx)
	defer s.service.executions.Delete(id)
	binding := scm.NewSlice([]scm.Scmer{scm.NewSymbol("node_query_context"), scm.NewString(id)})
	filter, err := scm.UnmarshalProcedure(request.Filter, scm.ProcedureDecode{Prepare: prepareScanProcedure(binding), Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) { return s.service.importObject(ctx, s.peer, ref) }})
	if err != nil {
		return nil, err
	}
	reducer, err := scm.UnmarshalProcedure(request.MapReducer, scm.ProcedureDecode{Prepare: prepareScanProcedure(binding), Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) { return s.service.importObject(ctx, s.peer, ref) }})
	if err != nil {
		return nil, err
	}
	if err := validateRemoteScanReadOnly(filter); err != nil {
		return nil, err
	}
	if err := validateRemoteScanReadOnly(reducer); err != nil {
		return nil, err
	}
	neutral, err := decodeScanValue(request.Neutral, request.ResultKind)
	if err != nil {
		return nil, err
	}
	var output remoteScanResult
	withInput := func(execute func(*storageShard) error) error { return withRemoteScanShard(&request.Shard, tx, execute) }
	if input != nil {
		withInput = func(execute func(*storageShard) error) error {
			shard := input
			if shard.uuid.String() != request.Shard.Resource.Shard || shard.t.filterSchemaFingerprint() != request.Shard.Schema || shard.t.schema.Name != request.Shard.Resource.Database || shard.t.Name != request.Shard.Resource.Table {
				return fmt.Errorf("retained input generation/schema mismatch")
			}
			lease := shard.getRead(tx, true)
			defer lease.Release()
			horizon := shard.captureScanHorizon()
			if horizon.Main != request.Shard.Horizon.Main || horizon.Delta < request.Shard.Horizon.Delta {
				return fmt.Errorf("retained input horizon mismatch")
			}
			return execute(shard)
		}
	}
	err = withInput(func(shard *storageShard) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		access, err := request.Access.bind()
		if err != nil {
			return err
		}
		access.ensureRuntime().horizon = &request.Shard.Horizon
		// Only the native shard handler executes the operator. The database owns
		// resolution/acquisition; the transport knows neither placement nor scans.
		value, err := shard.HandleClusterRequest(ctx, scm.ClusterRequest{Method: "scan", Payload: shardScanInvocation{tx: tx, access: access, conditionCols: request.FilterColumns, condition: filter, callbackCols: request.MapColumns, mapReduce: reducer, neutral: neutral}})
		if err != nil {
			return err
		}
		result := value.(scanResult)
		encoded, err := encodeScanResult(result.res, request.ResultKind)
		if err != nil {
			return err
		}
		output = remoteScanResult{Version: scanWireVersion, Task: request.Task, Attempt: request.Attempt, Shard: request.Shard, Kind: request.ResultKind, Value: encoded, OutCount: result.outCount, InputCount: int64(request.Shard.Horizon.Main) + int64(request.Shard.Horizon.Delta), CandidateCount: result.candidateCount}
		return nil
	})
	if _, capacity := err.(clusterCapacityRejection); capacity {
		return remoteScanResult{Rejected: true}, nil
	}
	if err != nil {
		return nil, err
	}
	reply, err := boundedScanReply(output)
	if err == nil && sharedClusterDatabase(database.Name) {
		s.service.cluster.received.Add(1) // completed remote task, outside all row loops
	}
	return reply, err
}

// Mutating storage/native callbacks need a remote transaction participant and
// are rejected before execution. Local functional reducers and explicit errors
// are allowed; this does not claim to sandbox mutually trusted cluster code.
func validateRemoteScanReadOnly(value scm.Scmer) error {
	if remoteMutationColumn(value) {
		return fmt.Errorf("remote callback contains a mutation column")
	}

	value = value.WithoutSourceInfo()
	if value.IsProc() {
		return validateRemoteScanReadOnly(value.Proc().Body)
	}
	if !value.IsSlice() {
		return nil
	}
	items := value.Slice()
	if len(items) == 0 {
		return nil
	}
	if head := items[0].WithoutSourceInfo(); head.IsSymbol() && head.String() == "quote" {
		return nil
	}
	if declaration := scm.DeclarationForValue(items[0]); declaration != nil && declaration.Type != nil {
		if declaration.Name == "quote" {
			return nil
		}
		if declaration.Type.HasSideEffects && declaration.Name != "error" && declaration.Name != "scan" && declaration.Name != "scan_exists" && declaration.Name != "scan_order" && declaration.Name != "scan_remote" {
			return fmt.Errorf("remote scan callback has side effects: %s", declaration.Name)
		}
		if declaration.Name == "eval" || declaration.Name == "scheme" {
			return fmt.Errorf("remote scan callback cannot evaluate dynamic code")
		}
	}
	for _, item := range items {
		if err := validateRemoteScanReadOnly(item); err != nil {
			return err
		}
	}
	return nil
}

func boundedScanReply(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(data) > 192*1024 {
		return nil, fmt.Errorf("remote aggregate exceeds node frame limit")
	}
	return value, nil
}

// Acquire through the same schema/shard MOSI and generation lifetime boundaries
// as a local scan. A changed physical generation aborts rather than rebinding IDs.
func withRemoteScanShard(expected *remoteScanShard, tx *TxContext, execute func(*storageShard) error) (err error) {
	// A shared-backend worker either admits this generation once or refuses
	// before scan callbacks. Capacity refusal alone permits choosing another peer.
	shared := sharedClusterDatabase(expected.Resource.Database)
	ctx := scanQueryContext(tx, context.Background())
	if shared {
		acquireClusterRead(ctx, mosiResource{Database: expected.Resource.Database})
	}
	db := GetDatabase(expected.Resource.Database)
	if db == nil {
		return fmt.Errorf("remote schema disappeared")
	}
	schemaLease := db.GetRead(nil)
	defer schemaLease.Release()
	tbl := db.GetTable(expected.Resource.Table)
	if tbl == nil {
		return fmt.Errorf("remote table disappeared")
	}
	if expected.Schema == 0 || tbl.filterSchemaFingerprint() != expected.Schema {
		return fmt.Errorf("remote scan schema changed")
	}
	declarations := tbl.columnNamesSnapshot.Load()
	for {
		topology := tbl.activeTopology()
		selected := findScanShard(tbl, expected.Resource)
		if selected == nil {
			return fmt.Errorf("remote shard disappeared")
		}
		if !topology.acquireOperation() {
			continue
		}
		selected.activeScanners.Add(1)
		if tbl.topology.Load() != topology {
			selected.activeScanners.Add(-1)
			topology.releaseOperation()
			continue
		}
		return func() error {
			defer topology.releaseOperation()
			defer selected.activeScanners.Add(-1)
			// Acquire current coherence rights, but retain the requested shard's
			// physical IDs and captured extent through native snapshot access.
			if r := activeReplication(tbl.schema); r != nil {
				cell := r.shardCell(selected)
				if shared {
					acquireClusterRead(ctx, *cell.resource.Load())
				} else {
					r.obtain(*cell.resource.Load(), mosi.S, "", tx)
				}
			}
			var lease ReadAccess
			var capacity bool
			func() {
				defer func() {
					if failure := recover(); failure != nil {
						if _, delegated := failure.(shardReadDelegation); shared && delegated {
							capacity = true
						} else {
							panic(failure)
						}
					}
				}()
				lease = selected.getRead(tx, true)
			}()
			if capacity {
				return clusterCapacityRejection{}
			}
			defer lease.Release()
			shard := lease.shard
			if shard.uuid.String() != expected.Resource.Shard {
				return fmt.Errorf("remote scan shard changed")
			}
			horizon := shard.captureScanHorizon()
			if shared {
				expected.Horizon = horizon
			}
			valid := horizon.Main == expected.Horizon.Main && horizon.Delta >= expected.Horizon.Delta
			if !valid {
				return fmt.Errorf("remote scan horizon changed")
			}
			if err := execute(shard); err != nil {
				return err
			}
			if tbl.columnNamesSnapshot.Load() != declarations || tbl.filterSchemaFingerprint() != expected.Schema || db.tables.Get(tbl.Name) != tbl {
				return fmt.Errorf("remote scan schema changed")
			}
			return nil
		}()
	}
}
func checkScanData(value scm.Scmer, depth int) error {
	value = value.WithoutSourceInfo()
	if depth > 64 {
		return fmt.Errorf("remote aggregate nesting limit")
	}
	if value.IsNil() || value.IsBool() || value.IsInt() || value.IsFloat() || value.IsString() {
		return nil
	}
	if value.IsFastDict() {
		for _, item := range value.FastDict().ToList() {
			if err := checkScanData(item, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if value.IsSlice() {
		for _, item := range value.Slice() {
			if err := checkScanData(item, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("remote scan cannot transport runtime values (tag %d, %T)", value.GetTag(), value.Any())
}
func decodeScanData(data json.RawMessage, groups bool) (scm.Scmer, error) {
	var value scm.Scmer
	if err := json.Unmarshal(data, &value); err != nil {
		return scm.NewNil(), err
	}
	if err := checkScanData(value, 0); err != nil {
		return scm.NewNil(), err
	}
	if groups {
		if !value.IsSlice() || len(value.Slice())%2 != 0 {
			return scm.NewNil(), fmt.Errorf("invalid remote grouped reduction")
		}
		pairs := value.Slice()
		dict := scm.NewFastDictValue(len(pairs) / 2)
		for i := 0; i < len(pairs); i += 2 {
			dict.Set(pairs[i], pairs[i+1], nil)
		}
		value = scm.NewFastDict(dict)
	}
	return value, nil
}
func decodeScanValue(data json.RawMessage, kind string) (scm.Scmer, error) {
	if kind != "value" {
		return decodeScanData(data, kind == "group_sums")
	}
	var value scanWireValue
	if err := json.Unmarshal(data, &value); err != nil {
		return scm.NewNil(), err
	}
	return value.thaw(0)
}
func encodeScanResult(value scm.Scmer, kind string) (json.RawMessage, error) {
	if kind == "value" {
		frozen, err := freezeScanValue(value, 0)
		if err != nil {
			return nil, err
		}
		return json.Marshal(frozen)
	}
	if kind == "group_sums" {
		if value.IsFastDict() {
			value = scm.NewSlice(value.FastDict().ToList())
		}
		if !value.IsSlice() || len(value.Slice())%2 != 0 {
			return nil, fmt.Errorf("grouped remote scan must return an association")
		}
	} else if value.IsSlice() || value.IsFastDict() {
		return nil, fmt.Errorf("scalar remote scan cannot return rows or associations")
	}
	if err := checkScanData(value, 0); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

type peerScanExecutor struct {
	input      *scm.ClusterReference
	inputLease scm.Scmer // keeps the imported old-generation handle until its query is unreachable
	peer       *mosiPeer
	ctx        context.Context
	prepared   remoteScanShard
	kind       string
	next       *atomic.Uint64
}

func (executor *peerScanExecutor) ExecuteShardScan(resource mosiResource, invocation shardScanInvocation) scanResult {
	condition := invocation.condition
	if condition.IsNil() {
		condition = scm.Eval(scm.Read("remote unconditional filter", "(lambda args true)"), &scm.Globalenv)
	}
	bindings := new(scm.SerializationContext)
	if tx := invocation.tx; tx != nil {
		if existing := tx.remoteBindings.Load(); existing != nil {
			bindings = existing
		} else if !tx.remoteBindings.CompareAndSwap(nil, bindings) {
			bindings = tx.remoteBindings.Load()
		}
	}
	var offers []scm.ClusterReference
	defer func() { executor.peer.scanService.cancelObjectOffers(executor.peer, offers) }()
	bindings = bindings.WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
		ref, err := executor.peer.scanService.exportObject(executor.peer, object, view)
		if err == nil {
			offers = append(offers, ref)
		}
		return ref, err
	})
	filter, err := scm.MarshalProcedure(condition, bindings)
	if err != nil {
		panic(err)
	}
	reducer, err := scm.MarshalProcedure(invocation.mapReduce, bindings)
	if err != nil {
		panic(err)
	}
	neutral, err := encodeScanResult(invocation.neutral, executor.kind)
	if err != nil {
		panic(err)
	}
	task := remoteScanTask{Input: executor.input, Version: scanWireVersion, Task: executor.next.Add(1), Attempt: uuid.NewString(), Visibility: "cursor-stability/captured-horizon", Shard: executor.prepared, FilterColumns: invocation.conditionCols, Filter: filter, MapColumns: invocation.callbackCols, MapReducer: reducer, Neutral: neutral, ResultKind: executor.kind}
	task.Access, err = freezeScanAccess(invocation.access)
	if err != nil {
		panic(err)
	}
	if invocation.tx != nil {
		// Export only unmodified remote inputs. A modified generation is kept
		// resident by native TX pins and must execute at its participant; a cold
		// path must not silently drop its private delete/undelete overlay.
		if db := GetDatabase(resource.Database); db != nil {
			if r := activeReplication(db); r != nil {
				if shard := r.findShard(resource); shard != nil && invocation.tx.getShardTx(shard) != nil {
					panic("remote modified input requires its transaction participant")
				}
			}
		}
		view, err := scm.ClusterObjectValue(invocation.tx, executor.ctx, scm.ClusterRequest{Method: "view"})
		if err != nil {
			panic(err)
		}
		ref, err := executor.peer.scanService.exportObject(executor.peer, invocation.tx, view)
		if err != nil {
			panic(err)
		}
		offers = append(offers, ref)
		task.Transaction = &ref
	}
	if resource != task.Shard.Resource {
		panic("remote scan resource mismatch")
	}
	response := executor.dispatch(task)
	if sharedClusterDatabase(resource.Database) {
		task.Shard.Horizon = response.Shard.Horizon
	}
	if response.Version != scanWireVersion || response.Task != task.Task || response.Attempt != task.Attempt || response.Shard != task.Shard || response.Kind != task.ResultKind {
		panic("mismatched remote scan response")
	}
	value, err := decodeScanValue(response.Value, response.Kind)
	if err != nil {
		panic(err)
	}
	return scanResult{res: value, outCount: response.OutCount, inputCount: response.InputCount, candidateCount: response.CandidateCount}
}

func remoteScan(a ...scm.Scmer) scm.Scmer {
	service := nodeScans.Load()
	if service == nil {
		panic("scan_remote requires --cluster-secret")
	}
	tx := scmerToTxContext(a[0])
	if tx != nil {
		tx.mu.Lock()
		supported := tx.Mode == TxCursorStability && tx.autoCommit && len(tx.shards) == 0
		tx.mu.Unlock()
		if !supported {
			panic("remote scans require a cursor-stability autocommit statement")
		}
	}
	ctx := context.Background()
	if ss := SessionStateFromTx(tx); ss != nil {
		if queryContext := ss.QueryContext(querySeqFromTx(tx)); queryContext != nil {
			ctx = queryContext
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	peer, _, err := service.connect(ctx, a[1].String())
	if err != nil {
		panic(err)
	}
	defer service.forget(peer)
	var shards []remoteScanDescriptor
	if err := service.control(ctx, peer, "scan.describe", remoteScanDescribe{scanWireVersion, a[2].String(), a[3].String()}, &shards); err != nil {
		panic(err)
	}
	kind := "scalar"
	if len(a) > 10 {
		kind = a[10].String()
	}
	columns := func(value scm.Scmer) []string {
		result := make([]string, len(value.Slice()))
		for i, col := range value.Slice() {
			result[i] = col.String()
		}
		return result
	}
	invocation := shardScanInvocation{tx: tx, conditionCols: columns(a[4]), condition: a[5], callbackCols: columns(a[6]), mapReduce: a[7], neutral: a[8]}
	acc := a[8]
	var sequence atomic.Uint64
	for _, part := range shards {
		if part.Resource.Database != a[2].String() || part.Resource.Table != a[3].String() || !part.Resource.valid() {
			panic("invalid remote scan directory response")
		}
		stubTable := &table{Name: part.Resource.Table, schema: &database{Name: part.Resource.Database}}
		identity, err := uuid.Parse(part.Resource.Shard)
		if err != nil {
			panic(err)
		}
		stub := &storageShard{t: stubTable, uuid: identity}
		remote := &peerScanExecutor{peer: peer, ctx: ctx, prepared: part.remoteScanShard, input: part.Object, kind: kind, next: &sequence}
		if part.Object != nil {
			value, err := service.importObject(ctx, peer, *part.Object)
			if err != nil {
				panic(err)
			}
			remote.inputLease = value
			object, ok := value.Any().(*remoteShardObject)
			if !ok {
				panic("remote source is not a shard handle")
			}
			// Descriptions carry a transfer offer. Execution MUST address the
			// imported peer lease, after that offer has been consumed atomically.
			remote.input = &object.handle.Reference
		}
		target := &shardScanTarget{resource: part.Resource, executor: remote}
		stub.scanTarget.Store(target)
		var result scanResult
		runAcquiredShardScan(tx, stub, true, nil, func(shard *storageShard, _ bool, failure any) {
			delegation, ok := failure.(shardReadDelegation)
			if !ok {
				if failure != nil {
					panic(failure)
				}
				panic("remote stub acquired local payload")
			}
			result = invocation.delegate(shard, delegation)
		})
		if result.err.r != nil {
			panic(result.err)
		}
		if result.outCount > 0 {
			acc = scm.Apply(a[9], acc, result.res)
		}
	}
	return acc
}

func remoteMutationColumn(value scm.Scmer) bool {
	value = value.WithoutSourceInfo()
	if value.IsString() {
		return value.String() == "$update" || strings.HasPrefix(value.String(), "$increment:")
	}
	if value.IsSlice() {
		for _, item := range value.Slice() {
			if remoteMutationColumn(item) {
				return true
			}
		}
	}
	return false
}

// Old UUIDs remain usable while a query/transfer retains that exact shard.
// The weak directory is populated at native rebuild publication, never by rows.
func findScanShard(table *table, resource mosiResource) *storageShard {
	for _, shard := range table.activeTopology().shards {
		if shard != nil && shard.uuid.String() == resource.Shard {
			return shard
		}
	}
	var cell *replicaCell
	if r := replication.Load(); r != nil {
		if snapshot := r.cellSnapshot.Load(); snapshot != nil {
			cell = (*snapshot)[resource.key()]
		}
	}
	if cell == nil {
		if cluster := activeNodeCluster(); cluster != nil {
			cell = cluster.manager.lookupCell(resource)
		}
	}
	if cell != nil {
		if current := cell.current.Load(); current != nil && current.t == table && current.uuid.String() == resource.Shard {
			return current
		}
		if retired := cell.retiredShards.Load(); retired != nil {
			for _, entry := range *retired {
				if shard := entry.Value(); shard != nil && shard.t == table && shard.uuid.String() == resource.Shard {
					return shard
				}
			}
		}
	}
	return nil
}

// bindProcedureScanContext is storage policy, not a Scheme language rule. It
// rebinds only storage scan invocations to the current worker's cancellation/TX
// scope before compilation. A standalone SCM function named scan is untouched.
func bindProcedureScanContext(value, queryContext scm.Scmer) scm.Scmer {
	plain := value.WithoutSourceInfo()
	if !plain.IsSlice() {
		return value
	}
	items := plain.Slice()
	if len(items) == 0 {
		return value
	}
	if storageProcedureSymbol(items[0], "quote") {
		return value
	}
	result := make([]scm.Scmer, len(items))
	for i, item := range items {
		result[i] = bindProcedureScanContext(item, queryContext)
	}
	if len(items) > 1 {
		switch {
		case storageProcedureSymbol(items[0], "scan"), storageProcedureSymbol(items[0], "scan_exists"), storageProcedureSymbol(items[0], "scan_order"), storageProcedureSymbol(items[0], "scan_remote"):
			arg := items[1].WithoutSourceInfo()
			rebound := arg.IsSlice() && len(arg.Slice()) > 0 && storageProcedureSymbol(arg.Slice()[0], "node_query_context")
			if arg.IsNil() || storageProcedureSymbol(arg, "nil") || rebound {
				result[1] = queryContext
			}
		}
	}
	return scm.NewSlice(result)
}

func storageProcedureSymbol(value scm.Scmer, name string) bool {
	value = value.WithoutSourceInfo()
	return value.IsSymbol() && value.String() == name
}
func prepareScanProcedure(binding scm.Scmer) func(scm.Scmer) (scm.Scmer, error) {
	return func(expression scm.Scmer) (scm.Scmer, error) {
		return bindProcedureScanContext(expression, binding), nil
	}
}
