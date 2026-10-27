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

type clusterCapacityRejection struct{}
type clusterReduceReply struct {
	Rejected bool              `json:"rejected"`
	Result   *remoteScanResult `json:"result,omitempty"`
}
type clusterShardHint struct {
	Warm bool `json:"warm"`
}

// scanQueryContext is used only when scheduling/acquiring a cold shard. The
// already-authorized read path must not consult SessionState's context mutex.
func scanQueryContext(tx *TxContext, fallback context.Context) context.Context {
	if ss := SessionStateFromTx(tx); ss != nil {
		if ctx := ss.QueryContext(querySeqFromTx(tx)); ctx != nil {
			return ctx
		}
	}
	return fallback
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
	// Existing local snapshots retain all native transaction modes. Only a
	// genuinely remote task needs a supported remote visibility contract.
	if shard.state() != COLD || !clusterScanSupported(tx) {
		return
	}
	origin := shard
	if len(origins) > 0 {
		origin = origins[0]
	}
	expected := remoteScanShard{Schema: shard.t.filterSchemaFingerprint(), Resource: mosiResource{shard.t.schema.Name, shard.t.Name, shard.uuid.String()}, Generation: shard.uuid}
	// Residency is only a hint. Permission acquisition remains at GetRead;
	// no backend bytes are read to decide whether execution should move.
	acquireClusterRead(scanQueryContext(tx, cluster.ctx), expected.Resource)
	links := cluster.view.Load().links
	// A warm local generation stays local. For a cold generation, ask existing
	// peers for residency metadata only; never fetch the shard just to decide
	// where to compute. All routing/IPC is outside row and filter-batch loops.
	if shard.state() == COLD {
		for _, link := range links {
			ctx, cancel := context.WithTimeout(cluster.ctx, 2*time.Second)
			var hint clusterShardHint
			err := cluster.service.control(ctx, link.peer, "cluster.hint", expected, &hint)
			cancel()
			if err == nil && hint.Warm {
				cluster.locality.Add(1)
				panic(shardReadDelegation{&shardScanTarget{resource: expected.Resource, generation: shard.uuid, executor: &clusterScanExecutor{cluster: cluster, expected: expected, preferred: link}, origin: origin}})
			}
		}
	}

}
func clusterSnapshotShard(ctx context.Context, expected remoteScanShard) (*storageShard, error) {
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
	acquireClusterRead(ctx, mosiResource{Database: db.Name})
	access := db.GetRead(nil)
	defer access.Release()
	table := db.GetTable(expected.Resource.Table)
	if table == nil || table.filterSchemaFingerprint() != expected.Schema {
		return nil, fmt.Errorf("cluster schema changed")
	}
	for _, shard := range table.activeTopology().shards {
		if shard != nil && shard.uuid.String() == expected.Resource.Shard && shard.uuid == expected.Generation {
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
		shard, err := clusterSnapshotShard(ctx, expected)
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
	shard, err := clusterSnapshotShard(ctx, task.Shard)
	if err != nil {
		return nil, err
	}
	acquireClusterRead(ctx, task.Shard.Resource)
	var access ReadAccess
	rejected := false
	func() {
		defer func() {
			if failure := recover(); failure != nil {
				if _, ok := failure.(shardReadDelegation); ok {
					rejected = true
				} else {
					panic(failure)
				}
			}
		}()
		access = shard.GetRead(nil)
	}()
	if rejected {
		return clusterReduceReply{Rejected: true}, nil
	}
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

// The native acquisition boundary raises the same typed panic before loading.
// Only an outer operator/reducer may translate it into a delegated task.
func (cluster *nodeCluster) shardRAMUnavailable(shard, origin *storageShard) shardReadDelegation {
	cluster.capacity.Add(1)
	expected := remoteScanShard{Schema: shard.t.filterSchemaFingerprint(), Resource: mosiResource{shard.t.schema.Name, shard.t.Name, shard.uuid.String()}, Generation: shard.uuid}
	return shardReadDelegation{&shardScanTarget{resource: expected.Resource, generation: shard.uuid, origin: origin, executor: &clusterScanExecutor{cluster: cluster, expected: expected}}}
}
