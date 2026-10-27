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

import "time"
import "context"

type clusterCapacityRejection struct{}

func (clusterCapacityRejection) Error() string { return "cluster shard RAM capacity unavailable" }

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
	return tx.Mode == TxCursorStability || tx.Mode == TxACID
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
	expected := remoteScanShard{Schema: shard.t.filterSchemaFingerprint(), Resource: mosiResource{shard.t.schema.Name, shard.t.Name, shard.uuid.String()}}
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
				panic(shardReadDelegation{&shardScanTarget{resource: expected.Resource, executor: &clusterScanExecutor{cluster: cluster, expected: expected, preferred: link}, origin: origin}})
			}
		}
	}

}

type clusterScanExecutor struct {
	cluster   *nodeCluster
	expected  remoteScanShard
	preferred *clusterLink
}

func (executor *clusterScanExecutor) ExecuteShardScan(resource mosiResource, invocation shardScanInvocation) scanResult {
	if !clusterScanSupported(invocation.tx) {
		panic("automatic cluster placement requires native CS or ACID read visibility")
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
			remote := peerScanExecutor{peer: link.peer, ctx: ctx, prepared: executor.expected, kind: "value", next: &link.sequence}
			result = remote.ExecuteShardScan(resource, invocation)
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
	expected := remoteScanShard{Schema: shard.t.filterSchemaFingerprint(), Resource: mosiResource{shard.t.schema.Name, shard.t.Name, shard.uuid.String()}}
	return shardReadDelegation{&shardScanTarget{resource: expected.Resource, origin: origin, executor: &clusterScanExecutor{cluster: cluster, expected: expected}}}
}
