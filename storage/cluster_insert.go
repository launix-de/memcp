/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "time"
import "bytes"
import "context"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

// The current schema owner reserves the existing native table counter. There
// is no seed/master allocator, second counter, or durable permissions catalog.
// Requests are once per INSERT batch/block, before any shard lock or ownership.
type clusterIDRequest struct {
	Database string `json:"database"`
	Table    string `json:"table"`
	Explicit uint64 `json:"explicit"`
	Count    uint64 `json:"count"`
}

type clusterIDReply struct {
	Base uint64 `json:"base"`
}

func (cluster *nodeCluster) reserveInsertIDs(t *table, explicit, count uint64) uint64 {
	if cluster == nil {
		panic(mosi.NetworkError)
	}
	request := clusterIDRequest{t.schema.Name, t.Name, explicit, count}
	reply, err := cluster.reserveIDs(cluster.ctx, request)
	if err != nil {
		panic(err)
	}
	// This is only an advisory local high-water snapshot. Future reservations
	// always consult the current schema owner, including after a handover.
	t.reserveLocalAutoIncrement(reply.Base+count, 0)
	return reply.Base
}

func (cluster *nodeCluster) reserveIDs(ctx context.Context, request clusterIDRequest) (reply clusterIDReply, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("%v", failure)
		}
	}()
	if !cluster.databases[request.Database] || request.Table == "" {
		return reply, fmt.Errorf("invalid cluster ID reservation")
	}
	owner, err := cluster.locate(ctx, mosiResource{Database: request.Database})
	if err != nil {
		return reply, err
	}
	if owner != cluster.self.ID {
		peer := cluster.peer(owner)
		if peer == nil {
			return reply, fmt.Errorf(mosi.NetworkError)
		}
		err = cluster.service.control(ctx, peer, "cluster.reserve_ids", request, &reply, true)
		return reply, err
	}
	db := GetDatabase(request.Database)
	if db == nil {
		return reply, fmt.Errorf("ID reservation database disappeared")
	}
	// Reservation changes the atomic runtime counter, not the logical schema.
	// Reuse the owner's existing directory transition boundary so a handover
	// cannot copy the old counter between reservation and its durable checkpoint.
	// This never upgrades an O owner to schema M or invalidates its readers.
	cell := cluster.manager.resourceCell(mosiResource{Database: request.Database})
	releaseCoordination := cell.coordinate(ctx)
	phase := cell.ownstate.Phase()
	if phase != mosi.M && phase != mosi.O {
		releaseCoordination()
		return cluster.reserveIDs(ctx, request) // routing changed; nothing was reserved
	}
	defer releaseCoordination()
	t := db.tables.Get(request.Table)
	if t == nil {
		return reply, fmt.Errorf("ID reservation table disappeared")
	}
	reply.Base = t.reserveLocalAutoIncrement(request.Explicit, request.Count)
	// Persist reservations before acknowledging: other nodes can commit their
	// allocated IDs without returning the allocator's shard WAL to this node.
	// An interrupted request may leave a gap and must never be blindly replayed.
	db.save(cell)
	return reply, nil
}

func (cluster *nodeCluster) handleIDReservation(ctx context.Context, data []byte) (any, error) {
	var request clusterIDRequest
	if err := unmarshalNode(data, &request); err != nil {
		return nil, err
	}
	return cluster.reserveIDs(ctx, request)
}

// Schema imports that only append writer lanes must preserve the existing live
// table and shard objects: they may own concurrent local deltas/transactions.
// The incoming schema owner serialized the complete durable topology already.
func (r *replicationManager) mergeClusterInsertTopology(previous, incoming *table) (bool, error) {
	if previous.ShardMode != ShardModeFree || incoming.ShardMode != ShardModeFree {
		return false, nil
	}
	before, after := previous.activeTopology(), incoming.activeTopology()
	if len(after.shards) <= len(before.shards) {
		return false, nil
	}
	for i, shard := range before.shards {
		if r.resourceKey(r.shardResource(shard)) != r.resourceKey(r.shardResource(after.shards[i])) {
			return false, nil
		}
	}
	if !bytes.Equal(r.replicaDeclaration(previous, false), r.replicaDeclaration(incoming, false)) {
		return false, nil
	}
	// Validate the complete append before replacing any generation. A stale
	// predecessor reference follows native UUID forwarding and keeps local data.
	for i, old := range before.shards {
		next := after.shards[i]
		if old.uuid == next.uuid || r.canonicalResource(r.shardResource(next)).Shard != next.uuid.String() {
			continue
		}
		cell := r.shardCell(old)
		if phase := cell.ownstate.Phase(); phase == mosi.M || phase == mosi.O {
			return false, fmt.Errorf("shared schema conflicts with a local modifier")
		}
	}
	for i, old := range before.shards {
		next := after.shards[i]
		if old.uuid == next.uuid || r.canonicalResource(r.shardResource(next)).Shard != next.uuid.String() {
			continue
		}
		cell := r.shardCell(old)
		next.t = previous
		next.coherence.Store(cell)
		if err := r.replaceShard(r.shardResource(old), old, next); err != nil {
			return false, err
		}
		cell.current.Store(next)
	}
	previous.reserveLocalAutoIncrement(atomic.LoadUint64(&incoming.Auto_increment), 0)
	previous.mu.Lock()
	defer previous.mu.Unlock()
	for _, shard := range after.shards[len(before.shards):] {
		shard.t = previous
		previous.Shards = append(previous.Shards, shard)
	}
	previous.publishSchemaTopology(ShardModeFree, previous.Shards, previous.PDimensions)
	previous.publishTopologyLocked()
	return true, nil
}

// An INSERT is one operation, not a dispatch acknowledgement followed by a
// replay. Defaults, ID reservation, UNIQUE checking and callbacks run together.
type remoteInsertTask struct {
	bindings         *scm.SerializationContext
	Version          int                   `json:"version"`
	Database         string                `json:"database"`
	Table            string                `json:"table"`
	Schema           uint64                `json:"schema"`
	Columns          []string              `json:"columns"`
	Rows             schemeValue           `json:"rows"`
	CollisionColumns []string              `json:"collision_columns"`
	Collision        []byte                `json:"collision"`
	MergeNull        bool                  `json:"merge_null"`
	Transaction      *scm.ClusterReference `json:"transaction,omitempty"`
}
type remoteInsertResult struct {
	Rejected bool   `json:"rejected,omitempty"`
	Count    int    `json:"count"`
	FirstID  *int64 `json:"first_id,omitempty"`
}

// A global UNIQUE check needs every partition, so use the current owner of
// its first native input. This is neither the contact node nor a schema leader:
// placement follows the existing shard ownership. Partition-local constraints
// can instead choose the first destination of this batch.
func (t *table) insertPeer(columns []string, rows [][]scm.Scmer) (*storageShard, *mosiPeer) {
	if !sharedClusterSource(t) || len(t.Unique) == 0 || len(rows) == 0 {
		return nil, nil
	}
	cluster := activeNodeCluster()
	if cluster == nil {
		return nil, nil
	}
	topology := t.activeTopology()
	if len(topology.shards) == 0 {
		return nil, nil
	}
	index := 0
	localKeys := true
	for _, key := range t.Unique {
		for _, dimension := range topology.dimensions {
			found := false
			for _, name := range key.Cols {
				found = found || name == dimension.Column
			}
			localKeys = localKeys && found
		}
	}
	if localKeys && topology.mode != ShardModeFree {
		key := make([]scm.Scmer, len(topology.dimensions))
		for i, dimension := range topology.dimensions {
			for j, column := range columns {
				if column == dimension.Column && j < len(rows[0]) {
					key[i] = rows[0][j]
				}
			}
		}
		index = computeShardIndex(topology.dimensions, key)
	}
	shard := topology.shards[index]
	cell := cluster.manager.shardCell(shard).resolved()
	if cell.ownstate.Phase() == mosi.M {
		return nil, nil
	}
	// Owner hints are already known MOSI state; never query a peer for them.
	r := cluster.manager
	r.mu.Lock()
	owner := r.directory.Records[r.resourceKey(*cell.resource.Load())].Owner
	r.mu.Unlock()
	if owner == "" || owner == cluster.self.ID {
		return nil, nil
	}
	return shard, cluster.peer(owner)
}

func (t *table) placeInsert(columns []string, rows [][]scm.Scmer, collisionCols []string, collision scm.Scmer, mergeNull bool, firstID func(int64), tx *TxContext) (int, bool) {
	shard, peer := t.insertPeer(columns, rows)
	if peer == nil {
		// Cold admission is decided by the native access boundary. A warm
		// local writer needs no cold payload and must not move just because
		// the node has no room for another shard.
		return 0, false
	}
	cluster := activeNodeCluster()
	feedback := cluster.insertPlacementSlot(shard)
	prior := feedback.Load()
	started := time.Now()
	if t.shouldClaimInsert(shard, peer.objectPeer.ID, prior, started) && claimInsertShard(shard, tx) {
		// Native ownership is acquired before any row effects. Once execution
		// starts, errors propagate; they never replay a partially applied batch.
		count := t.insertLocal(columns, rows, collisionCols, collision, mergeNull, firstID, tx)
		recordInsertClaim(feedback, shard, peer.objectPeer.ID, prior, time.Since(started))
		return count, true
	}
	// Only a capacity refusal before invocation and effects permits choosing
	// another peer. Execution errors never retry.
	candidates := (&clusterScanExecutor{cluster: cluster, expected: remoteScanShard{Resource: mosiResource{Database: t.schema.Name, Table: t.Name}}}).candidates()
	if peer != nil {
		started := time.Now()
		response := t.sendInsert(peer, columns, rows, collisionCols, collision, mergeNull, tx)
		if !response.Rejected {
			recordInsertDelegation(feedback, shard, peer.objectPeer.ID, time.Now(), time.Since(started))
			if firstID != nil && response.FirstID != nil {
				firstID(*response.FirstID)
			}
			return response.Count, true
		}
		if GlobalCache.canLoadClusterShard(nil) {
			return 0, false
		}
	}
	for _, candidate := range candidates {
		if candidate.peer == peer {
			continue
		}
		response := t.sendInsert(candidate.peer, columns, rows, collisionCols, collision, mergeNull, tx)
		if response.Rejected {
			continue
		}
		if firstID != nil && response.FirstID != nil {
			firstID(*response.FirstID)
		}
		return response.Count, true
	}
	panic("cluster has insufficient shard RAM capacity for INSERT")
}

func (t *table) sendInsert(peer *mosiPeer, columns []string, rows [][]scm.Scmer, collisionCols []string, collision scm.Scmer, mergeNull bool, tx *TxContext) remoteInsertResult {
	ctx := scanQueryContext(tx, peer.ctx)
	owner := tx
	if owner == nil || !owner.queryActive.Load() {
		owner = NewTxContext(TxCursorStability)
		owner.beginQuery(nil, 0, "remote insert")
		defer owner.endQuery(0)
	}
	service := peer.scanService
	var offers []scm.ClusterReference
	submitted := false
	defer func() {
		if !submitted {
			service.cancelObjectOffers(peer, offers)
		}
	}()
	var bindings *scm.SerializationContext
	bindings = (&scm.SerializationContext{Invocation: scm.NewAny(owner)}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
		ref, err := service.exportObject(peer, object, view, bindings)
		if err == nil {
			offers = append(offers, ref)
		}
		return ref, err
	})
	bindings = service.handleBindings(bindings, peer)
	input := make([]scm.Scmer, len(rows))
	for i, row := range rows {
		input[i] = scm.NewSlice(row)
	}
	encoded, err := freezeScanValue(scm.NewSlice(input), bindings)
	if err != nil {
		panic(err)
	}
	callback, err := freezeScanValue(collision, bindings)
	if err != nil {
		panic(err)
	}
	request := remoteInsertTask{bindings: bindings, Version: scanWireVersion, Database: t.schema.Name, Table: t.Name, Schema: t.filterSchemaFingerprint(), Columns: columns, Rows: schemeValue(encoded), CollisionColumns: collisionCols, Collision: []byte(callback), MergeNull: mergeNull}
	if tx != nil {
		view, err := scm.ClusterObjectValue(tx.transactionState, ctx, scm.ClusterRequest{Method: "view"})
		if err != nil {
			panic(err)
		}
		ref, err := service.exportObject(peer, tx.transactionState, view, bindings)
		if err != nil {
			panic(err)
		}
		offers = append(offers, ref)
		request.Transaction = &ref
	}
	activeNodeCluster().delegated.Add(1)
	var response remoteInsertResult
	submitted = true
	if err := service.control(ctx, peer, "insert.execute", request, &response, true, offers...); err != nil {
		panicRemoteOperation(err)
	}
	return response
}

func (database *database) executeInsertTask(ctx context.Context, session *scanRPCSession, request remoteInsertTask) (any, error) {
	if request.Version != scanWireVersion || request.Database != database.Name || request.Table == "" {
		return nil, fmt.Errorf("invalid remote INSERT")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !GlobalCache.canLoadClusterShard(nil) {
		return remoteInsertResult{Rejected: true}, nil
	}
	invocation, err := session.beginInvocation(ctx, request.Transaction, "remote insert")
	if err != nil {
		return nil, err
	}
	defer invocation.close()
	lease := database.GetRead(invocation.tx)
	defer lease.Release()
	table := database.GetTableForTx(request.Table, false, invocation.tx)
	if table == nil || table.filterSchemaFingerprint() != request.Schema {
		return nil, fmt.Errorf("remote INSERT schema changed")
	}
	input, err := request.Rows.reconstruct(invocation.decode)
	if err != nil {
		return nil, err
	}
	if !input.IsSlice() {
		return nil, fmt.Errorf("invalid INSERT rows")
	}
	rows := make([][]scm.Scmer, len(input.Slice()))
	for i, row := range input.Slice() {
		if !row.IsSlice() || len(row.Slice()) != len(request.Columns) {
			return nil, fmt.Errorf("invalid INSERT row")
		}
		rows[i] = row.Slice()
	}
	callback, err := schemeValue(request.Collision).reconstruct(invocation.decode)
	if err != nil {
		return nil, err
	}
	var result remoteInsertResult
	mutationTx := invocation.tx
	if request.Transaction == nil {
		// The invocation owns cancellation and decoding. Native inserts without
		// a transaction retain immediate publication on the execution peer.
		mutationTx = nil
	}
	// Ownership can change after dispatch. Native GetWrite/UNIQUE validation
	// resolve that race; this already-started command must never be redispatched.
	result.Count = table.insertLocal(request.Columns, rows, request.CollisionColumns, callback, request.MergeNull, func(id int64) {
		if result.FirstID == nil {
			result.FirstID = &id
		}
	}, mutationTx)
	if err := invocation.commit(ctx); err != nil {
		return nil, err
	}
	session.service.cluster.received.Add(1)
	return result, nil
}

func (task remoteInsertTask) serializationBindings() *scm.SerializationContext { return task.bindings }
