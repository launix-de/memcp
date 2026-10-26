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

// Query admission uses the existing cache-manager actor and total node budget.
// It is once per shard operator, never inside a value loop or a MOSI read check.
// Resident generation bytes are credited once: loading converts a reservation
// into cache ownership rather than charging the same payload twice.
type queryGenerationBudget struct {
	payload int64
	scratch int64
	users   int64
}
type queryBudgetOp struct {
	shard            *storageShard
	payload, scratch int64
	release          bool
	reply            chan queryRAMDecision
}
type queryRAMDecision struct {
	Allowed  bool
	Version  uint64
	Capacity int64
}

type queryRAMLease struct {
	manager *CacheManager
	shard   *storageShard
	scratch int64
}

func (cm *CacheManager) queryReservedBytes() int64 {
	var result int64
	for shard, item := range cm.queryGenerations {
		resident := int64(0)
		if entry := cm.itemMap[shard]; entry != nil {
			resident = entry.size
		}
		result = saturatingAdd(result, max(0, item.payload-resident)+item.scratch)
	}
	return result
}
func (cm *CacheManager) queryBudgetOperation(op *queryBudgetOp) {
	if cm.queryGenerations == nil {
		cm.queryGenerations = make(map[*storageShard]queryGenerationBudget)
	}
	prior := cm.queryGenerations[op.shard]
	if op.release {
		prior.users--
		prior.scratch -= op.scratch
		if prior.users == 0 {
			delete(cm.queryGenerations, op.shard)
		} else {
			cm.queryGenerations[op.shard] = prior
		}
		cm.queryVersion++
		if cm.queryChanged != nil {
			close(cm.queryChanged)
		}
		cm.queryChanged = make(chan struct{})
		return
	}
	next := prior
	next.users++
	next.payload = max(next.payload, op.payload)
	next.scratch += op.scratch
	cm.queryGenerations[op.shard] = next
	// Reclaim idle cache through the ordinary floor/lifetime protocol before
	// rejecting capacity. Active generation readers still prevent detachment.
	cm.runEvictionChecks(0)
	reserved := cm.queryReservedBytes()
	// Admission cannot evict through an active reader or the cache floor. The
	// normal eviction pass still runs after this operation; rejection lets the
	// caller choose another node before any payload/callback has been touched.
	allowed := cm.memoryBudget > 0 && cm.currentMemory+reserved <= cm.memoryBudget
	if !allowed {
		if prior.users == 0 {
			delete(cm.queryGenerations, op.shard)
		} else {
			cm.queryGenerations[op.shard] = prior
		}
	} else {
		cm.queryPeak = max(cm.queryPeak, cm.currentMemory+reserved)
	}
	op.reply <- queryRAMDecision{allowed, cm.queryVersion, cm.memoryBudget}
}
func (cm *CacheManager) tryQueryRAM(shard *storageShard, payload, scratch int64) (*queryRAMLease, queryRAMDecision) {
	if cm.opChan == nil || cm.stopped.Load() {
		return nil, queryRAMDecision{}
	}
	reply := make(chan queryRAMDecision, 1)
	cm.opChan <- cacheOp{query: &queryBudgetOp{shard: shard, payload: payload, scratch: scratch, reply: reply}}
	decision := <-reply
	if !decision.Allowed {
		return nil, decision
	}
	return &queryRAMLease{cm, shard, scratch}, decision
}

// The acquiring operator owns this lease exclusively and releases it exactly
// once, including acquisition panic and RPC failure. No retained query objects.
func (lease *queryRAMLease) Release() {
	if lease == nil {
		return
	}
	done := make(chan struct{})
	lease.manager.opChan <- cacheOp{query: &queryBudgetOp{shard: lease.shard, scratch: lease.scratch, release: true}, done: done}
	<-done
}

// Capacity notification is an actor-owned version/channel, not a poll loop or
// a MOSI flag. Recording the failed admission's version prevents a release
// between rejection and wait from becoming a lost wakeup.
func (cm *CacheManager) waitQueryCapacity(ctx context.Context, version uint64) error {
	if cm.opChan == nil || cm.stopped.Load() {
		return fmt.Errorf("cache budget stopped")
	}
	reply := make(chan (<-chan struct{}), 1)
	cm.opChan <- cacheOp{queryWatch: reply, queryVersion: version}
	changed := <-reply
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-changed:
		return nil
	case <-timer.C:
		return nil
	}
}

// Cold loading checks its existing reservation through the same actor. This
// replaces the generic ShardSize heuristic only for admitted shared mounts;
// it neither registers another reader nor charges the same future bytes twice.
func (cm *CacheManager) queryPayloadCovered(shard *storageShard) bool {
	if cm.opChan == nil || cm.stopped.Load() {
		return false
	}
	reply := make(chan bool, 1)
	cm.opChan <- cacheOp{queryCovered: shard, queryCoveredReply: reply}
	return <-reply
}
