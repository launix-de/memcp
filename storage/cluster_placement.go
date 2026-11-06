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
import "fmt"
import "context"
import "runtime"
import "hash/fnv"
import "sync/atomic"
import "github.com/google/uuid"

type clusterCapacityRejection struct{}

func (clusterCapacityRejection) Error() string { return "cluster shard RAM capacity unavailable" }

type clusterScanRedirect struct {
	Owner      string `json:"owner"`
	Epoch      uint64 `json:"epoch"`
	Generation string `json:"generation"`
}

func (redirect clusterScanRedirect) Error() string {
	return "cluster scan owner changed to " + redirect.Owner
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
	// Native transaction identity binds delegated statements to the existing
	// participant overlays on their shard owner.
	return tx.Mode == TxCursorStability || tx.Mode == TxACID
}
func (cluster *nodeCluster) prepareLocalScan(shard *storageShard, tx *TxContext, mutation bool, origins ...*storageShard) {
	// Existing local snapshots retain all native transaction modes. Only a
	// genuinely remote task needs a supported remote visibility contract.
	if shard.state() != COLD || !clusterScanSupported(tx) {
		return
	}
	// A job executes its subscans here while this node has room. Native GetRead
	// acquires S/O as necessary; another owner is not a reason to ship the job.
	tbl := shard.t
	if mutation {
		// Modified generations retain native TX pins. Their allocation is bounded
		// by the node's shard budget, rather than an evictable read-working-set share.
		tbl = nil
	}
	origin := shard
	if len(origins) > 0 {
		origin = origins[0]
	}
	expected := remoteScanShard{Schema: shard.t.filterSchemaFingerprint(), Resource: mosiResource{Database: shard.t.schema.Name, Table: shard.t.Name, Shard: shard.uuid.String()}}
	if GlobalCache.canLoadClusterShard(tbl) && (mutation || cluster.tryRead(scanQueryContext(tx, cluster.ctx), expected.Resource)) {
		return
	}
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
	exclude   string // an owner explicitly refused capacity for this invocation
}

// Equal-cost cold candidates follow shard affinity rather than roster order.
// Always trying the first peer funnels every capacity-delegated task through
// one worker even when other nodes have unused RAM. A stable rotation spreads
// different shards while keeping repeat accesses local to their chosen peer.
// Warm residency remains the cheaper candidate and takes precedence.
func (executor *clusterScanExecutor) candidates() []*clusterLink {
	links := executor.cluster.view.Load().links
	candidates := make([]*clusterLink, 0, len(links))
	if executor.preferred != nil && (executor.exclude == "" || executor.preferred.member.ID != executor.exclude) {
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
		if link != executor.preferred && (executor.exclude == "" || link.member.ID != executor.exclude) {
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
	candidates := executor.candidates()
	attempted := make(map[string]bool)
	var redirected map[string]clusterScanRedirect
	acquireFrom := executor.exclude
	for len(candidates) > 0 {
		link := candidates[0]
		candidates = candidates[1:]
		if attempted[link.member.ID] {
			continue
		}
		attempted[link.member.ID] = true
		select {
		case <-link.peer.done:
			continue
		default:
		}
		rejected := false
		var redirect *clusterScanRedirect
		var result scanResult
		func() {
			defer func() {
				if failure := recover(); failure != nil {
					if route, ok := failure.(clusterScanRedirect); ok {
						redirect = &route
					} else if _, ok := failure.(clusterCapacityRejection); ok {
						rejected = true
					} else {
						panic(failure)
					}
				}
			}()
			remote := peerScanExecutor{peer: link.peer, ctx: ctx, prepared: executor.expected, acquireFrom: acquireFrom}
			result = remote.ExecuteShardScan(resource, invocation)
		}()
		if redirect != nil {
			// Discovery spans live ownership publications. Returning to a peer
			// after a new epoch/generation is progress, not a cycle. Refusals
			// precede callback execution, so following them never replays effects.
			previous, seen := redirected[link.member.ID]
			if seen && previous.Generation == redirect.Generation && redirect.Epoch <= previous.Epoch {
				panic(fmt.Errorf("cluster scan ownership route cycled: peer=%s generation=%s epoch=%d owner=%s", link.member.ID, redirect.Generation, redirect.Epoch, redirect.Owner))
			}
			if redirected == nil {
				redirected = make(map[string]clusterScanRedirect)
			}
			// Native UUID forwarding is monotonic. Retain one latest refusal
			// per peer, not an unbounded history of ownership changes.
			redirected[link.member.ID] = *redirect
			var target *clusterLink
			for _, candidate := range executor.cluster.view.Load().links {
				if candidate.member.ID == redirect.Owner {
					target = candidate
					break
				}
			}
			if target == nil {
				panic(fmt.Errorf("cluster scan owner %s is unavailable", redirect.Owner))
			}
			attempted[target.member.ID] = false
			candidates = append([]*clusterLink{target}, candidates...)
			continue
		}
		if rejected {
			acquireFrom = link.member.ID
		}
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
	expected := remoteScanShard{Schema: shard.t.filterSchemaFingerprint(), Resource: mosiResource{Database: shard.t.schema.Name, Table: shard.t.Name, Shard: shard.uuid.String()}}
	return shardReadDelegation{&shardScanTarget{resource: expected.Resource, origin: origin, executor: &clusterScanExecutor{cluster: cluster, expected: expected}}}
}

// Feedback is bounded per node, allocated only for genuinely remote INSERTs.
// Colliding slots lose a prediction, never authority: native MOSI admission and
// UNIQUE reservations remain the sole synchronization mechanism. Local M never
// touches this cache. GOMAXPROCS observes the process's cgroup CPU allowance.
type insertPlacementFeedback struct {
	slots []atomic.Pointer[insertPlacementSample]
}

type insertPlacementSample struct {
	shard  uuid.UUID
	owner  string
	last   time.Time
	rent   time.Duration
	remote time.Duration
	claim  time.Duration
}

func (cluster *nodeCluster) insertPlacementSlot(shard *storageShard) *atomic.Pointer[insertPlacementSample] {
	feedback := cluster.insertPlacement.Load()
	if feedback == nil {
		candidate := &insertPlacementFeedback{slots: make([]atomic.Pointer[insertPlacementSample], 4*runtime.GOMAXPROCS(0))}
		cluster.insertPlacement.CompareAndSwap(nil, candidate)
		feedback = cluster.insertPlacement.Load()
	}
	var hash uint64 = 14695981039346656037
	for _, b := range shard.uuid {
		hash = (hash ^ uint64(b)) * 1099511628211
	}
	return &feedback.slots[hash%uint64(len(feedback.slots))]
}

// Rent-or-buy: the first access delegates. Repeated delegation pays for a
// transfer once accumulated remote work exceeds its estimated startup cost.
// The cold prior covers two control exchanges and decoding/index construction
// (500ns per estimated row); a completed claim replaces it with measured time.
// This is advisory costing, not a deadline or a permission to steal ownership.
func (t *table) shouldClaimInsert(shard *storageShard, owner string, sample *insertPlacementSample, now time.Time) bool {
	if sample == nil || sample.shard != shard.uuid || sample.owner != owner {
		return false
	}
	capacity := GlobalCache.shardCapacity.Load()
	if capacity == nil || !capacity.Allowed {
		return false
	}
	shards := max(1, len(t.activeTopology().shards))
	rows := uint64(t.CountEstimate()) / uint64(shards)
	statistics := t.statistics()
	bytes := uint64(max(int64(0), statistics.sizeBytes)) / uint64(shards)
	if bytes == 0 && rows == 0 {
		return false // unknown size does not justify speculative payload admission
	}
	if bytes == 0 {
		bytes = rows * uint64(max(1, len(t.columnDeclarations()))) * 32
	}
	// Avoid speculative acquisition of a generation comparable to the entire
	// node budget. The actual load still passes native admission and may refuse.
	if bytes > uint64(max(int64(0), capacity.Budget/4)) {
		return false
	}
	buy := sample.claim
	if buy == 0 {
		buy = 2*sample.remote + time.Duration(rows)*500*time.Nanosecond
	}
	return now.Sub(sample.last) <= max(time.Second, 32*buy) && sample.rent >= buy
}

func recordInsertDelegation(slot *atomic.Pointer[insertPlacementSample], shard *storageShard, owner string, now time.Time, elapsed time.Duration) {
	for {
		previous := slot.Load()
		next := &insertPlacementSample{shard: shard.uuid, owner: owner, last: now, rent: elapsed, remote: elapsed}
		if previous != nil && previous.shard == shard.uuid && previous.owner == owner {
			next.claim = previous.claim
			next.remote = (previous.remote + elapsed) / 2
			if now.Sub(previous.last) <= max(time.Second, 32*max(previous.claim, previous.remote)) {
				next.rent += previous.rent
			}
		}
		if slot.CompareAndSwap(previous, next) {
			return
		}
	}
}

func recordInsertClaim(slot *atomic.Pointer[insertPlacementSample], shard *storageShard, owner string, previous *insertPlacementSample, elapsed time.Duration) {
	next := &insertPlacementSample{shard: shard.uuid, owner: owner, last: time.Now(), claim: elapsed}
	// A competing statement may have published fresher feedback; it is only
	// a prediction and must never make the storage operation wait.
	slot.CompareAndSwap(previous, next)
}

// Only native pre-effect capacity rejection can choose another placement.
// Acquisition, cancellation and execution failures keep their normal semantics.
func claimInsertShard(shard *storageShard, tx *TxContext) (acquired bool) {
	defer func() {
		if failure := recover(); failure != nil {
			if _, capacity := failure.(shardReadDelegation); !capacity {
				panic(failure)
			}
		}
	}()
	lease := shard.GetWrite(tx)
	lease.Release()
	return true
}
