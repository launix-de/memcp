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
import "hash/fnv"

type clusterCapacityRejection struct{}

func (clusterCapacityRejection) Error() string { return "cluster shard RAM capacity unavailable" }

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
	// Native transaction identity binds delegated statements to the existing
	// participant overlays on their shard owner.
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
	// Placement can delegate to an authorized owner without acquiring a local
	// shared copy first. In particular a reader must not recall an active writer
	// merely to send its ordinary scan back to that same native participant.
	owner, err := cluster.locate(scanQueryContext(tx, cluster.ctx), expected.Resource)
	if err != nil {
		panic(err)
	}

	if owner != "" && owner != cluster.self.ID {
		for _, link := range cluster.view.Load().links {
			if link.member.ID == owner {
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

// Equal-cost cold candidates follow shard affinity rather than roster order.
// Always trying the first peer funnels every capacity-delegated task through
// one worker even when other nodes have unused RAM. A stable rotation spreads
// different shards while keeping repeat accesses local to their chosen peer.
// Warm residency remains the cheaper candidate and takes precedence.
func (executor *clusterScanExecutor) candidates() []*clusterLink {
	links := executor.cluster.view.Load().links
	candidates := make([]*clusterLink, 0, len(links))
	if executor.preferred != nil {
		candidates = append(candidates, executor.preferred)
	}
	if len(links) == 0 {
		return candidates
	}
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(executor.expected.Resource.Shard))
	first := int(hash.Sum64() % uint64(len(links)))
	for i := range links {
		link := links[(first+i)%len(links)]
		if link != executor.preferred {
			candidates = append(candidates, link)
		}
	}
	return candidates
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
	for _, link := range executor.candidates() {
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
			remote := peerScanExecutor{peer: link.peer, ctx: ctx, prepared: executor.expected}
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
