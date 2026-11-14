/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "runtime/debug"
import "github.com/launix-de/memcp/scm"

// Mutation consumes the RecSet's original native generations, never the table's
// current topology. The normal shard scan retains its writer admission, mapper,
// UpdateFunctionBatch, UNIQUE/row effects, statement Undo and transition owner.
func (r *recSet) scanMutation(tx *TxContext, schema scm.Scmer, values []scm.Scmer,
	conditionCols []string, condition scm.Scmer, callbackCols []string,
	mapper, neutral, combine scm.Scmer, outer bool) scm.Scmer {
	tx, finishData := r.table.writeDataContext(tx)
	defer finishData()

	if sharedClusterSource(r.table) {
		panic("native RecSet mutation currently requires local target generations")
	}
	r.table.admitCheckMutation(tx, "RecSet UPDATE/DELETE")
	release := r.table.admitNativeWrite(tx)
	defer release()
	r.table.beginContributionMutation()
	defer r.table.endContributionMutation()
	r.table.mutationMu.Lock()
	defer r.table.mutationMu.Unlock()
	ss := SessionStateFromTx(tx)
	querySeq := querySeqFromTx(tx)
	if r.table.hasTableLock() {
		r.table.waitTableLock(ss, querySeq, true)
	}
	touchTempColumns(r.table, conditionCols, callbackCols)
	access := recSetScanAccess(r, schema, values)
	accumulator := scm.ApplyOnce(neutral, nil)
	emptyResult := func() scm.Scmer {
		if outer {
			args := make([]scm.Scmer, len(callbackCols)+1)
			args[0] = accumulator
			return scm.Apply(mapper, args...)
		}
		return accumulator
	}
	if access.impossible() {
		return emptyResult()
	}
	shards := make([]*storageShard, 0, len(r.shards))
	for _, part := range r.shards {
		if part.count != 0 {
			shards = append(shards, part.shard)
		}
	}
	// Acquire the whole native writer set before the first effect. In
	// particular, an identity-changing owner cannot leave a partial mutation.
	writes := getShardWrites(shards, tx)
	defer func() {
		for index := len(writes) - 1; index >= 0; index-- {
			writes[index].Release()
		}
	}()
	for index, write := range writes {
		if write.shard != shards[index] {
			panic("native RecSet mutation cannot reinterpret a record generation")
		}
	}
	var results scanResultCollector
	results.prepare(len(shards))
	done := runFanoutTasks(tx, len(shards), func(index int, _ bool) {
		withTxSession(tx, func() scm.Scmer {
			defer func() {
				if failure := recover(); failure != nil {
					results.send(len(shards) == 1, scanResult{err: scanError{failure, string(debug.Stack())}})
				}
			}()
			if ss != nil && ss.IsKilledSeq(querySeq) {
				panic("query killed")
			}
			value, count, candidates := shards[index].scan(access, conditionCols, condition,
				callbackCols, mapper, neutral, 0, nil, tx, ss, true)
			results.send(len(shards) == 1, scanResult{res: value, outCount: count, candidateCount: candidates})
			return scm.NewNil()
		})
	})
	results.finish(done)
	hadValue := false
	var failure scanError
	for result, ok := results.next(); ok; result, ok = results.next() {
		if result.err.r != nil {
			if failure.r == nil {
				failure = result.err
			}
			continue
		}
		if failure.r == nil && result.outCount > 0 {
			hadValue = true
			if !combine.IsNil() {
				accumulator = scm.Apply(combine, accumulator, result.res)
			}
		}
	}
	if failure.r != nil {
		panic(failure)
	}
	if !hadValue {
		return emptyResult()
	}
	return accumulator
}
