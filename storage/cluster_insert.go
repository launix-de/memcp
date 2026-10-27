/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "context"
import "encoding/json"
import "sync/atomic"
import "bytes"
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
		err = cluster.service.control(ctx, peer, "cluster.reserve_ids", request, &reply)
		return reply, err
	}
	db := GetDatabase(request.Database)
	if db == nil {
		return reply, fmt.Errorf("ID reservation database disappeared")
	}
	access := db.GetWrite(nil)
	defer access.Release()
	t := db.tables.Get(request.Table)
	if t == nil {
		return reply, fmt.Errorf("ID reservation table disappeared")
	}
	reply.Base = t.reserveLocalAutoIncrement(request.Explicit, request.Count)
	// Persist reservations before acknowledging: other nodes can commit their
	// allocated IDs without returning the allocator's shard WAL to this node.
	// An interrupted request may leave a gap and must never be blindly replayed.
	db.save()
	return reply, nil
}

func (cluster *nodeCluster) handleIDReservation(ctx context.Context, data json.RawMessage) (any, error) {
	var request clusterIDRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, err
	}
	return cluster.reserveIDs(ctx, request)
}

// Schema imports that only append writer lanes must preserve the existing live
// table and shard objects: they may own concurrent local deltas/transactions.
// The incoming schema owner serialized the complete durable topology already.
func mergeClusterInsertTopology(previous, incoming *table) bool {
	if previous.ShardMode != ShardModeFree || incoming.ShardMode != ShardModeFree {
		return false
	}
	before, after := previous.activeTopology(), incoming.activeTopology()
	if len(after.shards) < len(before.shards) {
		return false
	}
	for i, shard := range before.shards {
		if shard.logicalID() != after.shards[i].logicalID() {
			return false
		}
	}
	if !bytes.Equal(replicaDeclaration(previous, false), replicaDeclaration(incoming, false)) {
		return false
	}

	for i, old := range before.shards {
		next := after.shards[i]
		if old.uuid == next.uuid {
			continue
		}
		if len(after.shards) == len(before.shards) {
			return false
		} // ordinary generation import owns this case
		cluster := activeNodeCluster()
		if cluster == nil {
			return false
		}
		r := cluster.manager
		cell := r.shardCell(old)
		if phase := cell.ownstate.Phase(); phase == mosi.M || phase == mosi.O {
			return false
		}
		next.t = previous
		next.coherence.Store(cell)
		if err := r.replaceShard(r.shardResource(old), old, next); err != nil {
			panic(err)
		}
		cell.current.Store(next)
	}
	previous.reserveLocalAutoIncrement(atomic.LoadUint64(&incoming.Auto_increment), 0)
	if len(after.shards) == len(before.shards) {
		return true
	}
	previous.mu.Lock()
	defer previous.mu.Unlock()
	for _, shard := range after.shards[len(before.shards):] {
		shard.t = previous
		previous.Shards = append(previous.Shards, shard)
	}
	previous.publishSchemaTopology(ShardModeFree, previous.Shards, previous.PDimensions)
	previous.publishTopologyLocked()
	return true
}

// Global INSERT constraints currently serialize through existing schema write
// admission. Acquire all read generations first: a foreign cold source may need
// schema M to checkpoint its handover. Never hold a data writer while probing a
// second writer's UNIQUE keys. No read-side registration or new lock is added.
func (t *table) clusterInsertConstraints(tx *TxContext) WriteAccess {
	if !sharedClusterSource(t) || len(t.Unique) == 0 {
		return WriteAccess{}
	}
	cluster := activeNodeCluster()
	for {
		for _, shard := range t.ActiveShards() {
			access := shard.GetRead(tx)
			access.Release()
		}
		access := t.schema.GetWrite(nil)
		ready := true
		for _, shard := range t.ActiveShards() {
			if !cluster.manager.shardCell(shard).ownstate.Phase().Readable() {
				ready = false
				break
			}
		}
		if ready {
			return access
		}
		access.Release()
	}
}
