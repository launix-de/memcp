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
import "github.com/launix-de/memcp/scm"

// shardScanTarget is immutable routing metadata for a remote-only stub. The
// query initializes it before using the stub. It grants no MOSI permission
// and must never be attached to a retained historical generation. The executor
// borrows that query and must not enter the persistent catalog.
type shardScanTarget struct {
	origin   *storageShard // optional query-local route for a real shared generation; never published on it
	resource mosiResource
	executor shardScanExecutor
}

// shardReadDelegation is raised only before GetRead touches local payload.
// Only an acquisition boundary may consume it; a callback panic is never a
// request to retry/delegate an operator which might already have run.
type shardReadDelegation struct{ target *shardScanTarget }

func (d shardReadDelegation) Error() string {
	return "shard requires delegated execution before payload acquisition"
}

// shardScanInvocation is borrowed query-local input, NOT a wire representation.
// A transport adapter must freeze the callbacks and access bindings and bind
// visibility/transactions explicitly before sending. TxContext, scanAccess,
// runtime closures, RecSets and RecMaps must never be serialized as Go objects.
// Operators retain their native execution kind. Batch inputs are invocation data;
// physical source sets require generation-bound row visibility, never new IDs.
type shardScanOperator uint8

const (
	shardReduce shardScanOperator = iota
	shardExists
	shardLookupOne
	shardLookupMany
	shardLookupMapOne
	shardLookupMapMany
	shardOrdered
)

type shardScanInvocation struct {
	project       bool // no combiner: retain the serial/side-effect callback at its coordinator
	order         *shardOrderInvocation
	operator      shardScanOperator
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
	ExecuteShardScan(mosiResource, shardScanInvocation) scanResult
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
	validResource := target != nil && target.resource.Shard == shard.uuid.String()
	if target != nil && target.origin == shard {
		if cell := shard.coherence.Load(); cell != nil {
			if current := cell.current.Load(); current != nil {
				validResource = current.uuid.String() == target.resource.Shard
			}
		}
	}
	if target == nil || (shard.scanTarget.Load() != target && target.origin != shard) || target.executor == nil || shard.t == nil ||
		target.resource.Database != shard.t.schema.Name || target.resource.Table != shard.t.Name ||
		!validResource || !target.resource.valid() {
		panic("invalid remote shard scan target")
	}
	if invocation.source != nil {
		panic("remote shard scan requires a portable physical source set")
	}
	for _, name := range invocation.callbackCols {
		if name == "$update" || strings.HasPrefix(name, "$increment:") {
			panic("remote shard scan mutations require a transaction participant protocol")
		}
	}
	if ss := SessionStateFromTx(invocation.tx); ss != nil && ss.IsKilledSeq(querySeqFromTx(invocation.tx)) {
		panic("query killed")
	}
	if invocation.project && invocation.operator == shardReduce && invocation.stride == 0 {
		// No combiner means the callback's effects belong to this invocation, not
		// a worker partial. Ship only filter/projection work; receive bounded values
		// and invoke the existing callback here without serializing a result sink.
		spec := scanOrderTableSpec{conditionCols: invocation.conditionCols, condition: invocation.condition, callbackCols: invocation.callbackCols, callback: invocation.mapReduce}
		queue := delegateOrderedQueue(invocation.tx, shard, invocation.access, &spec, nil, 0, 0, -1, failure)
		defer queue.closeReaders()
		value := invocation.neutral
		for len(queue.items) > 0 {
			var broke bool
			value, broke = queue.remote.stream(&spec, queue, value, queue.items)
			if broke {
				break
			}
			queue.consumeOrdered(len(queue.items), false)
		}
		return scanResult{res: value, outCount: int64(queue.remote.total), inputCount: int64(queue.universe), candidateCount: queue.candidateCount}
	}
	result = target.executor.ExecuteShardScan(target.resource, invocation)
	if result.err.r == nil && (result.outCount < 0 || result.inputCount < 0 || result.candidateCount < 0) {
		panic(fmt.Errorf("invalid remote shard scan counts"))
	}
	return result
}

// Only acquisition failures enter this adapter. Local lookup kernels and their
// scalar-cardinality/visibility rules remain unchanged; a remote worker invokes
// those same kernels and returns projected values plus the native match count.
func delegateShardLookup(tx *TxContext, shard *storageShard, access scanAccess, columns []string, operator shardScanOperator, failure any) (scm.Scmer, int) {
	delegation, ok := failure.(shardReadDelegation)
	if !ok {
		panic(failure)
	}
	invocation := shardScanInvocation{operator: operator, tx: tx, access: access, callbackCols: columns,
		condition: scm.Eval(scm.Read("remote lookup", "(lambda () true)"), &scm.Globalenv)}
	result := invocation.delegate(shard, delegation)
	if result.err.r != nil {
		panic(result.err)
	}
	return result.res, int(result.outCount)
}
