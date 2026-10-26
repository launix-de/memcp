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

import "io"
import "fmt"
import "time"
import "context"
import "encoding/json"
import "encoding/binary"
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

// Numeric immutable generations are the initial automatic-placement contract.
// Read bounded format headers, not the payload, to estimate loading before it
// happens. Unknown encodings fail explicitly instead of guessing from compressed
// object size or silently risking OOM. No on-disk/backend format is changed.
func estimateClusterShard(shard *storageShard) (int64, int64, error) {
	var payload, rows int64
	columns := shard.t.columnDeclarations()
	for _, column := range columns {
		if !column.Computor.IsNil() || column.IsTemp {
			return 0, 0, fmt.Errorf("automatic cluster scans require stored numeric columns")
		}
		reader := shard.t.schema.persistence.ReadColumn(shard.uuid.String(), column.Name)
		var header [24]byte
		_, err := io.ReadFull(reader, header[:1])
		if err == nil {
			size := 24
			if header[0] == 12 {
				size = 16
			}
			_, err = io.ReadFull(reader, header[1:size])
		}
		reader.Close()
		if err != nil {
			return 0, 0, fmt.Errorf("reading cluster column header: %w", err)
		}
		count, bytes := uint64(0), uint64(0)
		switch header[0] {
		case 10:
			if header[3] != 0 {
				return 0, 0, fmt.Errorf("unsupported cluster integer version")
			}
			chunks := binary.LittleEndian.Uint64(header[8:16])
			if chunks > 1<<35 {
				return 0, 0, fmt.Errorf("cluster integer payload exceeds supported extent")
			}
			bytes = chunks * 8
			count = binary.LittleEndian.Uint64(header[16:24])
		case 12:
			if header[1] != 0 && header[1] != '1' {
				return 0, 0, fmt.Errorf("unsupported cluster float version")
			}
			count = binary.LittleEndian.Uint64(header[8:16])
			bytes = count * 8
		case 11:
			if header[1] != 0 && header[1] != '1' {
				return 0, 0, fmt.Errorf("unsupported cluster sequence version")
			}
			count = binary.LittleEndian.Uint64(header[8:16])
			sequences := binary.LittleEndian.Uint64(header[16:24])
			if sequences > count {
				return 0, 0, fmt.Errorf("invalid cluster sequence extent")
			}
			bytes = sequences*24 + 512
		default:
			return 0, 0, fmt.Errorf("automatic cluster scans currently support numeric generations only (encoding %d)", header[0])
		}
		if count > uint64(^uint32(0)) || bytes > uint64(1<<40) {
			return 0, 0, fmt.Errorf("cluster column estimate exceeds supported extent")
		}
		rows = max(rows, int64(count))
		payload = saturatingAdd(payload, int64(bytes)+256)
	}
	payload = saturatingAdd(payload, rows*2+4096) // generation maps/visibility buffers
	scratch := int64(512<<10) + int64(len(columns))*defaultScanBufferSize*32
	return payload, scratch, nil
}
func clusterScanSupported(tx *TxContext) bool {
	if tx == nil {
		return true
	}
	tx.mu.Lock()
	defer tx.mu.Unlock()
	return tx.Mode == TxCursorStability && tx.autoCommit && len(tx.shards) == 0
}
func (cluster *nodeCluster) prepareLocalScan(shard *storageShard, tx *TxContext) *queryRAMLease {
	if !clusterScanSupported(tx) {
		panic("cluster scans require cursor-stability autocommit")
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
				panic(shardReadDelegation{&shardScanTarget{resource: expected.Resource, generation: shard.uuid, executor: &clusterScanExecutor{cluster: cluster, expected: expected, preferred: link}, origin: shard}})
			}
		}
	}
	payload, scratch, err := estimateClusterShard(shard)
	if err != nil {
		panic(err)
	}
	if lease, _ := GlobalCache.tryQueryRAM(shard, payload, scratch); lease != nil {
		return lease
	}
	cluster.capacity.Add(1)
	panic(shardReadDelegation{&shardScanTarget{resource: expected.Resource, generation: shard.uuid, executor: &clusterScanExecutor{cluster: cluster, expected: expected}, origin: shard}})
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
	payload, scratch, err := estimateClusterShard(shard)
	if err != nil {
		return nil, err
	}
	lease, decision := GlobalCache.tryQueryRAM(shard, payload, scratch)
	if lease == nil {
		return clusterReduceReply{Rejected: true, Version: decision.Version, Permanent: decision.Capacity <= 0 || payload+scratch > decision.Capacity}, nil
	}
	defer lease.Release()
	access := shard.GetRead(nil)
	defer access.Release()
	shard = access.shard
	// Immutable shared mounts have no concurrent modifier. Capture the actual
	// extent on the executor under its normal storage ownership, before running
	// the existing fenced RPC. No payload or physical row IDs leave this node.
	// The S-only immutable mount excludes writers/rebuild, and access pins the
	// loaded generation against eviction. These scalar/slice-header reads need
	// no additional RWMutex reader write on the cache line.
	clean := len(shard.inserts) == 0 && shard.deletions.Count() == 0
	task.Shard.Horizon = shardScanHorizon{shard.main_count, 0}
	if !clean {
		return nil, fmt.Errorf("shared cluster generations must be rebuilt before mounting")
	}
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
	for {
		var waitLink *clusterLink
		var waitVersion uint64
		for _, link := range candidates {
			// A connection already known closed cannot have accepted this task. Skip
			// that stale placement hint; never retry a call after enqueue/ambiguity.
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
						if decision, ok := failure.(clusterCapacityRejection); ok {
							rejected = true
							if !decision.Permanent && waitLink == nil {
								waitLink = link
								waitVersion = decision.Version
							}
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
		// A temporary admission rejection has executed no callback. Wait on the
		// existing budget actor's release version before trying placement again.
		// Computation failures and ambiguous network completion never reach here.
		if waitLink == nil {
			panic("cluster has insufficient RAM capacity for this shard reduction")
		}
		if err := waitLink.peer.call(ctx, "cluster.wait_capacity", waitVersion, nil); err != nil {
			panic(err)
		}
	}

}
