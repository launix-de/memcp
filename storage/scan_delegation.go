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
import "strings"
import "runtime/debug"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// shardScanTarget is immutable routing metadata for a remote-only stub. The
// query initializes it before using the stub. It grants no MOSI permission
// and must never be attached to a retained historical generation. The executor
// borrows that query and must not enter the persistent catalog.
type shardScanTarget struct {
	resource   mosiResource
	generation uuid.UUID
	executor   shardScanExecutor
}

// shardReadDelegation is raised only before GetRead touches local payload.
// Only an acquisition boundary may consume it; a callback panic is never a
// request to retry/delegate an operator which might already have run.
type shardReadDelegation struct{ target *shardScanTarget }

func (d shardReadDelegation) Error() string {
	return "remote shard requires delegated scan execution"
}

// shardScanInvocation is borrowed query-local input, NOT a wire representation.
// A transport adapter must freeze the callbacks and access bindings and bind
// visibility/transactions explicitly before sending. TxContext, scanAccess,
// runtime closures, RecSets and RecMaps must never be serialized as Go objects.
// This first boundary supports ordinary unbatched scans only; physical source
// sets and mutations fail explicitly until their remote contracts exist.
type shardScanInvocation struct {
	tx            *TxContext
	access        scanAccess
	conditionCols []string
	condition     scm.Scmer
	callbackCols  []string
	mapReduce     scm.Scmer
	neutral       scm.Scmer
	stride        int
	batchdata     []scm.Scmer
	source        *recSet
}

// ExecuteShardScan runs at most once per selected shard in this invocation.
// Errors and ambiguous network completion propagate; this boundary never retries.
// The result uses the existing shard-local reduction and coordinator combine
// contract. A future wire adapter must validate/encode its scalar/group result.
type shardScanExecutor interface {
	ExecuteShardScan(mosiResource, uuid.UUID, shardScanInvocation) scanResult
}

func (invocation shardScanInvocation) delegate(shard *storageShard, failure shardReadDelegation) (result scanResult) {
	// A transport/executor failure must enter the ordinary result collector,
	// including on the direct single-shard path's recovery stack.
	defer func() {
		if recovered := recover(); recovered != nil {
			result = scanResult{err: scanError{recovered, string(debug.Stack())}}
		}
	}()
	target := failure.target
	if target == nil || shard.scanTarget.Load() != target || target.executor == nil || shard.t == nil ||
		target.resource.Database != shard.t.schema.Name || target.resource.Table != shard.t.Name ||
		target.resource.Shard != shard.logicalID().String() || target.generation != shard.uuid || !target.resource.valid() {
		panic("invalid remote shard scan target")
	}
	if invocation.source != nil || invocation.stride != 0 || len(invocation.batchdata) != 0 {
		panic("remote shard scan does not support physical source sets or batches")
	}
	for _, name := range invocation.callbackCols {
		if name == "$update" || strings.HasPrefix(name, "$increment:") {
			panic("remote shard scan mutations require a transaction participant protocol")
		}
	}
	if ss := SessionStateFromTx(invocation.tx); ss != nil && ss.IsKilledSeq(querySeqFromTx(invocation.tx)) {
		panic("query killed")
	}
	result = target.executor.ExecuteShardScan(target.resource, target.generation, invocation)
	if result.err.r == nil && (result.outCount < 0 || result.inputCount < 0 || result.candidateCount < 0) {
		panic(fmt.Errorf("invalid remote shard scan counts"))
	}
	return result
}
