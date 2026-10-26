/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/
package storage

import "fmt"
import "time"
import "context"
import "encoding/json"
import "github.com/google/uuid"

type clusterCapacityRejection struct {
	Version   uint64
	Permanent bool
}
type clusterReduceReply struct {
	Version   uint64            `json:"version"`
	Permanent bool              `json:"permanent"`
	Rejected  bool              `json:"rejected"`
	Result    *remoteScanResult `json:"result,omitempty"`
}
type clusterShardHint struct {
	Warm bool `json:"warm"`
}

func clusterScanSupported(tx *TxContext) bool {
	if tx == nil {
		return true
	}
	// Mode and autocommit are statement-owned immutable admission metadata.
	// A locally modified shard retains M and resident payload, so its own TX
	// overlays cannot be delegated by the cold-shard placement path.
	return tx.Mode == TxCursorStability && tx.autoCommit
}
func (cluster *nodeCluster) prepareLocalScan(shard *storageShard, tx *TxContext, origins ...*storageShard) {
	if !clusterScanSupported(tx) {
		panic("cluster scans require cursor-stability autocommit")
	}
	// Already resident means compute here. Do not query peers, read storage
	// headers, or write any shared accounting for this decision.
	if shard.state() != COLD {
		return
	}
	origin := shard
	if len(origins) > 0 {
		origin = origins[0]
	}
	expected := remoteScanShard{Schema: shard.t.filterSchemaFingerprint(), Resource: mosiResource{shard.t.schema.Name, shard.t.Name, shard.logicalID().String()}, Generation: shard.uuid}
	// Permission precedes even bounded backend-header I/O. The shared mount
	// supplies S only; residency and membership never fabricate a stronger grant.
	acquireClusterSnapshot(expected.Resource)
	links := cluster.view.Load().links
	// A warm local generation stays local. For a cold generation, ask existing
	// peers for residency metadata only; never fetch the shard just to decide
	// where to compute. All routing/IPC is outside row and filter-batch loops.
	if shard.state() == COLD {
		for _, link := range links {
			ctx, cancel := context.WithTimeout(cluster.ctx, 2*time.Second)
			var hint clusterShardHint
			err := link.peer.call(ctx, "cluster.hint", expected, &hint)
			cancel()
			if err == nil && hint.Warm {
				cluster.locality.Add(1)
				panic(shardReadDelegation{&shardScanTarget{resource: expected.Resource, generation: shard.uuid, executor: &clusterScanExecutor{cluster: cluster, expected: expected, preferred: link}, origin: origin}})
			}
		}
	}
	if GlobalCache.canLoadClusterShard() {
		return
	}
	cluster.capacity.Add(1)
	panic(shardReadDelegation{&shardScanTarget{resource: expected.Resource, generation: shard.uuid, executor: &clusterScanExecutor{cluster: cluster, expected: expected}, origin: origin}})
}
func clusterSnapshotShard(expected remoteScanShard) (*storageShard, error) {
	if !expected.Resource.valid() || expected.Resource.Shard == "" || expected.Generation == uuid.Nil || expected.Schema == 0 {
		return nil, fmt.Errorf("invalid cluster shard identity")
	}
	if !sharedClusterDatabase(expected.Resource.Database) {
		return nil, fmt.Errorf("database is not mounted for cluster scans")
	}
	db := GetDatabase(expected.Resource.Database)
	if db == nil {
		return nil, fmt.Errorf("unknown shared database")
	}
	access := db.GetRead(nil)
	defer access.Release()
	table := db.GetTable(expected.Resource.Table)
	if table == nil || table.filterSchemaFingerprint() != expected.Schema {
		return nil, fmt.Errorf("cluster schema changed")
	}
	for _, shard := range table.activeTopology().shards {
		if shard != nil && shard.logicalID().String() == expected.Resource.Shard && shard.uuid == expected.Generation {
			return shard, nil
		}
	}
	return nil, fmt.Errorf("cluster generation changed")
}
func (cluster *nodeCluster) compute(session *scanRPCSession, ctx context.Context, method string, data json.RawMessage) (any, error) {
	if method == "cluster.hint" {
		var expected remoteScanShard
		if err := json.Unmarshal(data, &expected); err != nil {
			return nil, err
		}
		shard, err := clusterSnapshotShard(expected)
		if err != nil {
			return nil, err
		}
		return clusterShardHint{shard.state() != COLD}, nil
	}
	var task remoteScanTask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	if task.ResultKind != "scalar" {
		return nil, fmt.Errorf("automatic cluster placement currently supports scalar reductions")
	}
	shard, err := clusterSnapshotShard(task.Shard)
	if err != nil {
		return nil, err
	}
	acquireClusterSnapshot(task.Shard.Resource)
	if shard.state() == COLD && !GlobalCache.canLoadClusterShard() {
		return clusterReduceReply{Rejected: true}, nil
	}
	access := shard.GetRead(nil)
	defer access.Release()
	shard = access.shard
	// Pin the native generation, then borrow its existing published extent.
	// Cancellation is checked at task/shard boundaries, never in item loops.
	task.Shard.Horizon = shard.captureScanHorizon()
	result, err := session.executeScanTask(ctx, task)
	if err != nil {
		return nil, err
	}
	value, ok := result.(remoteScanResult)
	if !ok {
		return nil, fmt.Errorf("invalid cluster reduction result")
	}
	cluster.received.Add(1)
	return clusterReduceReply{Result: &value}, nil
}

type clusterScanExecutor struct {
	cluster   *nodeCluster
	expected  remoteScanShard
	preferred *clusterLink
}

func (executor *clusterScanExecutor) ExecuteShardScan(resource mosiResource, generation uuid.UUID, invocation shardScanInvocation) scanResult {
	if !clusterScanSupported(invocation.tx) || invocation.neutral.IsSlice() || invocation.neutral.IsFastDict() {
		panic("automatic cluster placement requires a read-only scalar reduction")
	}
	ctx := executor.cluster.ctx
	if ss := SessionStateFromTx(invocation.tx); ss != nil {
		if query := ss.QueryContext(querySeqFromTx(invocation.tx)); query != nil {
			ctx = query
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	links := executor.cluster.view.Load().links
	candidates := make([]*clusterLink, 0, len(links))
	if executor.preferred != nil {
		candidates = append(candidates, executor.preferred)
	}
	for _, link := range links {
		if link != executor.preferred {
			candidates = append(candidates, link)
		}
	}
	for _, link := range candidates {
		select {
		case <-link.peer.done:
			continue
		default:
		}
		rejected := false
		var result scanResult
		func() {
			defer func() {
				if failure := recover(); failure != nil {
					if _, ok := failure.(clusterCapacityRejection); ok {
						rejected = true
					} else {
						panic(failure)
					}
				}
			}()
			remote := peerScanExecutor{peer: link.peer, ctx: ctx, prepared: executor.expected, kind: "scalar", next: &link.sequence, cluster: true}
			result = remote.ExecuteShardScan(resource, generation, invocation)
		}()
		if !rejected {
			executor.cluster.delegated.Add(1)
			return result
		}
	}
	panic("cluster has insufficient shard RAM capacity")
}
