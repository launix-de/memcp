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
import "runtime"
import "context"
import "strings"
import "encoding/json"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

// Version 10 combines authenticated Scheme operations with adaptive ordered windows.
const scanWireVersion = 10

type remoteScanShard struct {
	Schema   uint64       `json:"schema"`
	Resource mosiResource `json:"resource"`
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
			result = append(result, referenceOffers(*part.Object)...)
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
	Order         *remoteScanOrder      `json:"order,omitempty"`
	Operator      shardScanOperator     `json:"operator"`
	Stride        int                   `json:"stride,omitempty"`
	Batch         []schemeValue         `json:"batch,omitempty"`
	Input         *scm.ClusterReference `json:"input,omitempty"`
	Transaction   *scm.ClusterReference `json:"transaction,omitempty"`
	Access        remoteScanAccess      `json:"access"`
	Version       int                   `json:"version"`
	Shard         remoteScanShard       `json:"shard"`
	FilterColumns []string              `json:"filter_columns"`
	Filter        json.RawMessage       `json:"filter"`
	MapColumns    []string              `json:"map_columns"`
	MapReducer    json.RawMessage       `json:"mapreducer"`
	Neutral       json.RawMessage       `json:"neutral"`
}
type remoteScanResult struct {
	Rejected       bool            `json:"rejected,omitempty"` // capacity refusal before executing scan callbacks
	Version        int             `json:"version"`
	Shard          remoteScanShard `json:"shard"`
	Value          json.RawMessage `json:"value"`
	OutCount       int64           `json:"out_count"`
	InputCount     int64           `json:"input_count"`
	CandidateCount int64           `json:"candidate_count"`
}

// The authenticated connection owns operation cancellation and duplicate suppression.
// All execution uses the identity in the outer control envelope.
type scanRPCSession struct {
	peer      *mosiPeer
	peerReady chan struct{}
	tasks     scanTaskRegistry
	service   *nodeScanService // immutable connection-owned role adapter
	remoteID  string           // authenticated identity, never taken from an RPC assertion
}

func newScanRPCSession() *scanRPCSession {
	return &scanRPCSession{peerReady: make(chan struct{})}
}
func (s *scanRPCSession) bindPeer(peer *mosiPeer) { s.peer = peer; close(s.peerReady) }
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
	if request.Version != scanWireVersion || !request.Shard.Resource.valid() || request.Shard.Resource.Shard == "" {
		return nil, fmt.Errorf("invalid remote scan contract")
	}
	if request.Operator > shardOrdered || request.Stride < 0 || (request.Operator != shardReduce && request.Stride != 0) || (request.Stride == 0 && len(request.Batch) != 0) || (request.Stride > 0 && len(request.Batch)%request.Stride != 0) {
		return nil, fmt.Errorf("invalid remote scan invocation")
	}
	if (request.Operator == shardOrdered) != (request.Order != nil) {
		return nil, fmt.Errorf("missing or unexpected ordered scan contract")
	}
	for _, cols := range [][]string{request.FilterColumns, request.MapColumns} {
		for _, col := range cols {
			if slot, batch := parseBatchPseudoColName(col); batch && slot >= request.Stride {
				return nil, fmt.Errorf("batch column outside its input row")
			}
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
		return nil, fmt.Errorf("remote execution requires an operation envelope")
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
	decode := scm.ProcedureDecode{Prepare: prepareScanProcedure(binding), Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) { return s.service.importObject(ctx, s.peer, ref) }}
	var order *shardOrderInvocation
	if request.Order != nil {
		order, err = request.Order.bind(decode)
		if err != nil {
			return nil, err
		}
	}
	batch := make([]scm.Scmer, len(request.Batch))
	for i, item := range request.Batch {
		var err error
		batch[i], err = item.reconstruct(scm.ProcedureDecode{Prepare: prepareScanProcedure(binding), Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) { return s.service.importObject(ctx, s.peer, ref) }})
		if err != nil {
			return nil, err
		}
	}
	neutral, err := schemeValue(request.Neutral).reconstruct(decode)
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
			lease := shard.getRead(tx, true, false)
			defer lease.Release()
			return execute(shard)
		}
	}
	err = withInput(func(shard *storageShard) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		access, err := request.Access.bind(scm.ProcedureDecode{Prepare: prepareScanProcedure(binding), Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) { return s.service.importObject(ctx, s.peer, ref) }})
		if err != nil {
			return err
		}
		for _, item := range request.Access.Boundaries {
			boundary := ScanBoundaryFromScmer(item)
			for _, slot := range []int{boundary.LowerSlot(), boundary.UpperSlot()} {
				if slot <= -2 && -slot-2 >= request.Stride {
					return fmt.Errorf("batch boundary outside its input row")
				}
			}
		}
		// Only the native shard handler executes the operator. The database owns
		// resolution/acquisition; the transport knows neither placement nor scans.
		value, err := shard.HandleClusterRequest(ctx, scm.ClusterRequest{Method: "scan", Payload: shardScanInvocation{order: order, operator: request.Operator, stride: request.Stride, batchdata: batch, tx: tx, access: access, conditionCols: request.FilterColumns, condition: filter, callbackCols: request.MapColumns, mapReduce: reducer, neutral: neutral}})
		if err != nil {
			return err
		}
		result := value.(scanResult)
		var offered []scm.ClusterReference
		bindings := (&scm.SerializationContext{}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
			ref, err := s.service.exportObject(s.peer, object, view)
			if err == nil {
				offered = append(offered, ref)
			}
			return ref, err
		})
		encoded, err := freezeScanValue(result.res, bindings)
		if err != nil {
			s.service.cancelObjectOffers(s.peer, offered)
			return err
		}
		output = remoteScanResult{Version: scanWireVersion, Shard: request.Shard, Value: json.RawMessage(encoded), OutCount: result.outCount, InputCount: result.inputCount, CandidateCount: result.candidateCount}
		return nil
	})
	if _, capacity := err.(clusterCapacityRejection); capacity {
		return remoteScanResult{Rejected: true}, nil
	}
	if err != nil {
		return nil, err
	}
	reply, err := boundedScanReply(output)
	if err != nil {
		s.service.cancelObjectOffers(s.peer, output.ClusterObjectOffers())
	}
	if err == nil && sharedClusterDatabase(database.Name) {
		s.service.cluster.received.Add(1) // completed remote task, outside all row loops
	}
	return reply, err
}

// The measured faster path retains the first encoding. The frame encoder
// still bounds the actual outer message, including headers, at send time.
type encodedReply struct {
	data   json.RawMessage
	offers []scm.ClusterReference
}

func (reply encodedReply) MarshalJSON() ([]byte, error)                { return reply.data, nil }
func (reply encodedReply) ClusterObjectOffers() []scm.ClusterReference { return reply.offers }
func boundedScanReply(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(data) > 192*1024 {
		return nil, fmt.Errorf("result exceeds node frame limit")
	}
	var offers []scm.ClusterReference
	if source, ok := value.(interface{ ClusterObjectOffers() []scm.ClusterReference }); ok {
		offers = source.ClusterObjectOffers()
	}
	return encodedReply{data, offers}, nil
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
				lease = selected.getRead(tx, true, false)
			}()
			if capacity {
				return clusterCapacityRejection{}
			}
			defer lease.Release()
			shard := lease.shard
			if shard.uuid.String() != expected.Resource.Shard {
				return fmt.Errorf("remote scan shard changed")
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

type peerScanExecutor struct {
	input      *scm.ClusterReference
	inputLease scm.Scmer // keeps the imported old-generation handle until its query is unreachable
	peer       *mosiPeer
	ctx        context.Context
	prepared   remoteScanShard
}

func (origin *peerScanExecutor) ExecuteShardScan(resource mosiResource, invocation shardScanInvocation) scanResult {
	executor := *origin
	executor.ctx = scanQueryContext(invocation.tx, origin.ctx)
	defer runtime.KeepAlive(executor.inputLease)
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
	mapReduce := invocation.mapReduce
	if invocation.operator != shardReduce {
		mapReduce = scm.Eval(scm.Read("remote exists", "(lambda args false)"), &scm.Globalenv)
	}
	reducer, err := scm.MarshalProcedure(mapReduce, bindings)
	if err != nil {
		panic(err)
	}
	neutral, err := freezeScanValue(invocation.neutral, bindings)
	if err != nil {
		panic(err)
	}
	task := remoteScanTask{Operator: invocation.operator, Stride: invocation.stride, Input: executor.input, Version: scanWireVersion, Shard: executor.prepared, FilterColumns: invocation.conditionCols, Filter: filter, MapColumns: invocation.callbackCols, MapReducer: reducer, Neutral: json.RawMessage(neutral)}
	if invocation.order != nil {
		task.Order, err = freezeScanOrder(invocation.order, bindings)
		if err != nil {
			panic(err)
		}
	}
	for _, item := range invocation.batchdata {
		value, err := freezeScanValue(item, bindings)
		if err != nil {
			panic(err)
		}
		task.Batch = append(task.Batch, value)
	}
	task.Access, err = freezeScanAccess(invocation.access, bindings)
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
	if response.Version != scanWireVersion || response.Shard != task.Shard {
		panic("mismatched remote scan response")
	}
	defer executor.peer.scanService.discardObjectOffers(executor.peer, response.ClusterObjectOffers())
	value, err := schemeValue(response.Value).reconstruct(scm.ProcedureDecode{Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) {
		return executor.peer.scanService.importObject(executor.ctx, executor.peer, ref)
	}})
	if err != nil {
		panic(err)
	}
	return scanResult{res: value, outCount: response.OutCount, inputCount: response.InputCount, candidateCount: response.CandidateCount}
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
		cell = r.lookupCell(resource)
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
		// Native declarations own execution-argument metadata. Rebinding does
		// not enumerate operators or inspect which functions a lambda may call.
		declaration := scm.DeclarationForValue(items[0])
		if declaration != nil && declaration.Type != nil && len(declaration.Type.Params) > 0 {
			label := declaration.Type.Params[0].Label
			if label == "tx" || label == "transaction" {
				arg := items[1].WithoutSourceInfo()
				rebound := arg.IsSlice() && len(arg.Slice()) > 0 && storageProcedureSymbol(arg.Slice()[0], "node_query_context")
				if arg.IsNil() || storageProcedureSymbol(arg, "nil") || rebound {
					result[1] = queryContext
				}
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
