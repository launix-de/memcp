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

const scanWireVersion = 1

type shardScanHorizon struct {
	Main  uint32 `json:"main"`
	Delta uint32 `json:"delta"`
}
type remoteScanShard struct {
	Schema     uint64           `json:"schema"`
	Resource   mosiResource     `json:"resource"`
	Generation uuid.UUID        `json:"generation"`
	Horizon    shardScanHorizon `json:"horizon"`
}
type remoteScanDescribe struct {
	Version  int    `json:"version"`
	Database string `json:"database"`
	Table    string `json:"table"`
}
type remoteScanTask struct {
	Version       int             `json:"version"`
	Task          uint64          `json:"task"`
	Attempt       string          `json:"attempt"`
	Visibility    string          `json:"visibility"`
	Shard         remoteScanShard `json:"shard"`
	FilterColumns []string        `json:"filter_columns"`
	Filter        json.RawMessage `json:"filter"`
	MapColumns    []string        `json:"map_columns"`
	MapReducer    json.RawMessage `json:"mapreducer"`
	Neutral       json.RawMessage `json:"neutral"`
	ResultKind    string          `json:"result_kind"`
}
type remoteScanResult struct {
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
	service  *nodeScanService // immutable connection-owned role adapter
	remoteID string           // authenticated identity, never taken from an RPC assertion
	mu       sync.Mutex
	high     uint64
	seen     [1024]uint64
	workers  chan struct{}
}

func newScanRPCSession() *scanRPCSession { return &scanRPCSession{workers: make(chan struct{}, 8)} }
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
	// Membership control must not be rejected just because all compute slots
	// are busy; overload is not evidence that an authenticated node disappeared.
	if method == "cluster.hello" || method == "cluster.wait_capacity" {
		if s.service == nil || s.service.cluster == nil {
			return nil, fmt.Errorf("cluster membership is not enabled")
		}
		return s.service.cluster.handle(s, ctx, s.remoteID, method, data)
	}
	select {
	case s.workers <- struct{}{}:
		defer func() { <-s.workers }()
	default:
		if method == "cluster.reduce" {
			return clusterReduceReply{Rejected: true}, nil
		}
		return nil, fmt.Errorf("remote scan worker limit")
	}
	if strings.HasPrefix(method, "cluster.") {
		if s.service == nil || s.service.cluster == nil {
			return nil, fmt.Errorf("cluster membership is not enabled")
		}
		return s.service.cluster.handle(s, ctx, s.remoteID, method, data)
	}

	switch method {
	case "scan.describe":
		var request remoteScanDescribe
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		if request.Version != scanWireVersion || !validReplicaDatabase(request.Database) || request.Table == "" {
			return nil, fmt.Errorf("invalid remote scan source")
		}
		if sharedClusterDatabase(request.Database) {
			return nil, fmt.Errorf("shared cluster sources require automatic bounded placement")
		}
		db := GetDatabase(request.Database)
		if db == nil {
			return nil, fmt.Errorf("unknown remote database")
		}
		lease := db.GetRead(nil)
		defer lease.Release()
		tbl := db.GetTable(request.Table)
		if tbl == nil || tbl.isEphemeralQueryTable() {
			return nil, fmt.Errorf("unknown or ephemeral remote table")
		}
		result := make([]remoteScanShard, 0)
		var scanErr any
		var mu sync.Mutex
		done := tbl.iterateShardsParallel(nil, scanAccess{}, nil, func(shard *storageShard, _ bool, failure any) {
			if failure != nil {
				mu.Lock()
				scanErr = failure
				mu.Unlock()
				return
			}
			shard.mu.RLock()
			part := remoteScanShard{Schema: tbl.filterSchemaFingerprint(), Resource: mosiResource{Database: db.Name, Table: tbl.Name, Shard: shard.logicalID().String()}, Generation: shard.uuid, Horizon: shardScanHorizon{shard.main_count, uint32(len(shard.inserts))}}
			shard.mu.RUnlock()
			mu.Lock()
			result = append(result, part)
			mu.Unlock()
		})
		if done != nil {
			<-done
		}
		if scanErr != nil {
			return nil, fmt.Errorf("remote scan acquisition: %v", scanErr)
		}
		return boundedScanReply(result)
	case "scan.execute":
		var request remoteScanTask
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		if sharedClusterDatabase(request.Shard.Resource.Database) {
			return nil, fmt.Errorf("shared cluster tasks require RAM admission through cluster.reduce")
		}
		return s.executeScanTask(ctx, request)
	}
	return nil, fmt.Errorf("unknown remote scan method")
}

// Both explicitly addressed and automatically placed reductions enter this
// same checked executor. Admission is owned by the caller; replay, procedure,
// generation, visibility and result validation are not duplicated by adapters.
func (s *scanRPCSession) executeScanTask(ctx context.Context, request remoteScanTask) (any, error) {
	if request.Version != scanWireVersion || request.Visibility != "cursor-stability/captured-horizon" || !request.Shard.Resource.valid() || request.Shard.Resource.Shard == "" || request.Shard.Generation == uuid.Nil {
		return nil, fmt.Errorf("invalid remote scan contract")
	}
	if _, err := uuid.Parse(request.Attempt); err != nil {
		return nil, fmt.Errorf("invalid scan attempt")
	}
	if request.ResultKind != "scalar" && request.ResultKind != "group_sums" {
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
	filter, err := scm.UnmarshalProcedure(request.Filter)
	if err != nil {
		return nil, err
	}
	reducer, err := scm.UnmarshalProcedure(request.MapReducer)
	if err != nil {
		return nil, err
	}
	if err := validateRemoteScanReadOnly(filter); err != nil {
		return nil, err
	}
	if err := validateRemoteScanReadOnly(reducer); err != nil {
		return nil, err
	}
	neutral, err := decodeScanData(request.Neutral, false)
	if err != nil {
		return nil, err
	}
	var output remoteScanResult
	err = withRemoteScanShard(request.Shard, func(shard *storageShard) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		access := runtimeScanAccess(nil)
		access.ensureRuntime().horizon = &request.Shard.Horizon
		value, count, candidates := shard.scan(access, request.FilterColumns, filter, request.MapColumns, reducer, neutral, 0, nil, nil, nil)
		encoded, err := encodeScanResult(value, request.ResultKind)
		if err != nil {
			return err
		}
		output = remoteScanResult{Version: scanWireVersion, Task: request.Task, Attempt: request.Attempt, Shard: request.Shard, Kind: request.ResultKind, Value: encoded, OutCount: count, InputCount: int64(request.Shard.Horizon.Main) + int64(request.Shard.Horizon.Delta), CandidateCount: candidates}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return boundedScanReply(output)
}

// Mutating storage/native callbacks need a remote transaction participant and
// are rejected before execution. Local functional reducers and explicit errors
// are allowed; this does not claim to sandbox mutually trusted cluster code.
func validateRemoteScanReadOnly(value scm.Scmer) error {
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
		if declaration.Type.HasSideEffects && declaration.Name != "error" {
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
func withRemoteScanShard(expected remoteScanShard, execute func(*storageShard) error) error {
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
		var selected *storageShard
		for _, shard := range topology.shards {
			if shard != nil && shard.logicalID().String() == expected.Resource.Shard {
				selected = shard
				break
			}
		}
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
			lease := selected.GetRead(nil)
			defer lease.Release()
			shard := lease.shard
			if shard.uuid != expected.Generation || shard.logicalID().String() != expected.Resource.Shard {
				return fmt.Errorf("remote scan generation changed")
			}
			shard.mu.RLock()
			valid := shard.main_count == expected.Horizon.Main && uint64(len(shard.inserts)) >= uint64(expected.Horizon.Delta)
			shard.mu.RUnlock()
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
	if depth > 64 {
		return fmt.Errorf("remote aggregate nesting limit")
	}
	if value.IsNil() || value.IsBool() || value.IsInt() || value.IsFloat() || value.IsString() {
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
	return fmt.Errorf("remote scan cannot transport runtime values")
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
func encodeScanResult(value scm.Scmer, kind string) (json.RawMessage, error) {
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
	cluster  bool // immutable shared-generation placement captures extent at executor
	peer     *mosiPeer
	ctx      context.Context
	prepared remoteScanShard
	kind     string
	next     *atomic.Uint64
}

func (executor *peerScanExecutor) ExecuteShardScan(resource mosiResource, _ uuid.UUID, invocation shardScanInvocation) scanResult {
	if invocation.access.len() != 0 {
		panic("remote scan requires explicit serialized filter without physical access constraints")
	}
	filter, err := scm.MarshalProcedure(invocation.condition)
	if err != nil {
		panic(err)
	}
	reducer, err := scm.MarshalProcedure(invocation.mapReduce)
	if err != nil {
		panic(err)
	}
	if err := checkScanData(invocation.neutral, 0); err != nil {
		panic(err)
	}
	neutral, err := json.Marshal(invocation.neutral)
	if err != nil {
		panic(err)
	}
	task := remoteScanTask{Version: scanWireVersion, Task: executor.next.Add(1), Attempt: uuid.NewString(), Visibility: "cursor-stability/captured-horizon", Shard: executor.prepared, FilterColumns: invocation.conditionCols, Filter: filter, MapColumns: invocation.callbackCols, MapReducer: reducer, Neutral: neutral, ResultKind: executor.kind}
	if resource != task.Shard.Resource {
		panic("remote scan resource mismatch")
	}
	var response remoteScanResult
	if executor.cluster {
		var reply clusterReduceReply
		if err := executor.peer.call(executor.ctx, "cluster.reduce", task, &reply); err != nil {
			panic(err)
		}
		if reply.Rejected {
			panic(clusterCapacityRejection{Version: reply.Version, Permanent: reply.Permanent})
		}
		if reply.Result == nil {
			panic("missing cluster reduction")
		}
		response = *reply.Result
		// This mode accepts only immutable, delta-free mounted generations. The
		// worker captures their extent after admission; identity/schema remain exact.
		task.Shard.Horizon = response.Shard.Horizon
	} else if err := executor.peer.call(executor.ctx, "scan.execute", task, &response); err != nil {
		panic(err)
	}
	if response.Version != scanWireVersion || response.Task != task.Task || response.Attempt != task.Attempt || response.Shard != task.Shard || response.Kind != task.ResultKind {
		panic("mismatched remote scan response")
	}
	value, err := decodeScanData(response.Value, response.Kind == "group_sums")
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
	var shards []remoteScanShard
	if err := peer.call(ctx, "scan.describe", remoteScanDescribe{scanWireVersion, a[2].String(), a[3].String()}, &shards); err != nil {
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
		stub := &storageShard{t: stubTable, uuid: part.Generation, lineage: identity}
		target := &shardScanTarget{resource: part.Resource, generation: part.Generation, executor: &peerScanExecutor{peer: peer, ctx: ctx, prepared: part, kind: kind, next: &sequence}}
		stub.scanTarget.Store(target)
		var result scanResult
		runAcquiredShardScan(tx, stub, true, func(shard *storageShard, _ bool, failure any) {
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
