/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"

// A completed lexical MEMORY scope cannot escape or be resurrected by the
// caller's savepoints. Its ordinary row journals nevertheless belonged to that
// SAME caller while the program was running. Retire only those dead identities;
// never commit, roll back or replace the caller's remaining transaction.
func (scope *relationScope) validateCallerRetirement(caller *TxContext) {
	if scope.dataPolicy != relationCallerData || scope.transactionalSchema || caller.transactionState == nil ||
		caller.remoteObject != nil || caller.remoteInvocation || caller.coordinator != nil {
		panic("caller scope retirement requires its original local MEMORY data owner")
	}
	caller.requireMutationBinding()
	// Reject a callback closing its own input BEFORE waiting for closeMu:
	// another closer may already hold it while joining this same callback.
	if scope.hasCallerInput(caller.transactionState) {
		panic("caller scope retirement requires completed native inputs")
	}
}

// The same lexical scope owns retryable cleanup after it detaches its dead
// identities from the original caller. Successful invocations retain no header.
type relationCallerRetirement struct {
	state      *transactionState
	readers    []queryReaderLease
	columns    map[*column]*table
	statements []*statementTableWriteLease
	writers    []*tableWriteLease
}

func (retirement *relationCallerRetirement) finish() {
	var first any
	attempt := func(release func()) (ok bool) {
		defer func() {
			if failure := recover(); failure != nil {
				if first == nil {
					first = failure
				}
				ok = false
			}
		}()
		release()
		return true
	}
	pendingReader := false
	for index, reader := range retirement.readers {
		if reader.release != nil {
			if attempt(reader.release) {
				retirement.readers[index] = queryReaderLease{}
			} else {
				pendingReader = true
			}
		}
	}
	// Preserve the existing reader-before-column release order on retry too.
	if !pendingReader {
		for column, target := range retirement.columns {
			if attempt(func() { target.releaseColumnCacheUse(column) }) {
				delete(retirement.columns, column)
			}
		}
	}
	for index, lease := range retirement.statements {
		if lease != nil && attempt(func() {
			if err := lease.Close(); err != nil {
				panic(err)
			}
		}) {
			retirement.statements[index] = nil
		}
	}
	for index, lease := range retirement.writers {
		if lease != nil && attempt(func() {
			if err := lease.Close(); err != nil {
				panic(err)
			}
		}) {
			retirement.writers[index] = nil
		}
	}
	if first != nil {
		panic(first)
	}
}

func (scope *relationScope) finishCallerRetirement() {
	scope.callerRetirement.finish()
}

func (scope *relationScope) retireCallerData(caller *TxContext, tables []*table) {
	scope.detachCallerData(caller, tables)
	scope.finishCallerRetirement()
}

func (scope *relationScope) detachCallerData(caller *TxContext, tables []*table) {
	state := caller.transactionState
	owned := make(map[*table]bool, len(tables))
	for _, target := range tables {
		if !target.Ephemeral || target.PersistencyMode != Memory || target.ScopedSchema ||
			target.dataOwner != nil || target.relationLifetime != scope || !scope.owns(target) {
			panic("caller scope retirement lost its exact MEMORY identity")
		}
		owned[target] = true
	}
	// Decisions snapshot their shard directory outside state.mu. Their existing
	// lifecycle owner must therefore exclude retirement through the whole detach.
	// No scope/schema/shard lock is held while acquiring this owner.
	state.decisionMu.Lock()
	defer state.decisionMu.Unlock()
	resources := state.resourceScope()
	var retiredResources []*tableWriteLease
	var readers []queryReaderLease
	columns := make(map[*column]*table)
	var operation *nativeMutationOperation
	func() {
		resources.mu.Lock()
		defer resources.mu.Unlock()
		state.mu.Lock()
		defer state.mu.Unlock()
		if state.decisionStarted {
			panic("caller scope retirement cannot change an accepted decision")
		}
		// Validate every constructor and retained writer before detaching any
		// original journal. The formatted key alone is not an owner proof.
		for target := range owned {
			entry := resources.entries[fmt.Sprintf("native-table-write/%p", target)]
			if entry != nil {
				select {
				case <-entry.done:
				default:
					panic("caller scope retirement has an active writer constructor")
				}
				lease, ok := entry.value.Any().(*tableWriteLease)
				if !ok || lease.owner != &target.writeAdmission || lease.transaction != state {
					panic("caller scope retirement lost its original native writer")
				}
			}
			target.writeAdmission.mu.Lock()
			escaped := target.writeAdmission.writers[state].unframedCallbacks
			foreign := false
			for writer, capability := range target.writeAdmission.writers {
				foreign = foreign || writer != state && capability.leases != 0
			}
			foreign = foreign || target.writeAdmission.statement != nil && target.writeAdmission.statement.transaction != state
			target.writeAdmission.mu.Unlock()
			if foreign {
				panic("caller scope retirement has another native writer owner")
			}
			if escaped {
				panic("caller scope retirement has retained unframed mutation callbacks")
			}
		}
		for shard, participant := range state.shardTransactions() {
			if owned[shard.t] {
				state.releaseShardTransactionLocked(shard, participant)
			}
		}
		for column, table := range caller.queryColumns {
			if owned[table] {
				delete(caller.queryColumns, column)
				columns[column] = table
			}
		}
		keptReaders := caller.queryReaderReleases[:0]
		for _, reader := range caller.queryReaderReleases {
			if owned[reader.source] {
				readers = append(readers, reader)
			} else {
				keptReaders = append(keptReaders, reader)
			}
		}
		clear(caller.queryReaderReleases[len(keptReaders):])
		caller.queryReaderReleases = keptReaders
		for target := range owned {
			key := fmt.Sprintf("native-table-write/%p", target)
			if resource := resources.entries[key]; resource != nil {
				retiredResources = append(retiredResources, resource.value.Any().(*tableWriteLease))
				delete(resources.entries, key)
			}
		}
		keptDeletes := state.repartitionDeletes[:0]
		for _, action := range state.repartitionDeletes {
			if !owned[action.table] {
				keptDeletes = append(keptDeletes, action)
			}
		}
		clear(state.repartitionDeletes[len(keptDeletes):])
		state.repartitionDeletes = keptDeletes
		operation = state.checkedMutation
	}()
	// The enclosing public producer is still running. Its completed local
	// child has no live frame; its dead lexical targets no longer need that
	// producer's statement exclusions or savepoint journal.
	var leases []*statementTableWriteLease
	if operation != nil {
		operation.mu.Lock()
		if statement := operation.statement; statement != nil {
			for target := range owned {
				if lease := statement.tables[target]; lease != nil {
					leases = append(leases, lease)
					delete(statement.tables, target)
				}
			}
		}
		operation.mu.Unlock()
	}
	scope.callerRetirement = &relationCallerRetirement{state: state, readers: readers, columns: columns, statements: leases, writers: retiredResources}
}

// The original journal must be released BEFORE joining maintenance: a
// repartition's existing retirement owner can itself await that journal pin.
// Caller now owns each table's maintenanceMu, so no private rebuild can
// republish one of these masks after this final prune.
func (retirement *relationCallerRetirement) pruneProjections(tables []*table) {
	owned := make(map[*table]bool, len(tables))
	for _, target := range tables {
		owned[target] = true
	}
	state := retirement.state
	// Projections are non-owning masks, but still own their bitmap payload.
	// Concurrent projections for unrelated sources keep their CAS publication.
	for {
		previous := state.visibilityProjections.Load()
		if previous == nil {
			break
		}
		next := maps.Clone(*previous)
		for identity, projection := range next {
			if shard := projection.target.Value(); shard != nil && owned[shard.t] {
				delete(next, identity)
			}
		}
		if len(next) == len(*previous) {
			break
		}
		if state.visibilityProjections.CompareAndSwap(previous, &next) {
			for identity, projection := range *previous {
				if next[identity] == nil {
					if shard := projection.target.Value(); shard != nil {
						shard.visibilityProjections.CompareAndDelete(state, projection)
						shard.t.signalTransactionDrain()
					}
				}
			}
			break
		}
	}
}

// Acquire at the existing physical input boundary, not in a row loop. Closing
// the native capability seals new inputs and joins already admitted readers or
// writers before their original journals can be discarded.
func (t *table) callerRelationScope() *relationScope {
	if t == nil || !t.Ephemeral || t.PersistencyMode != Memory || t.dataOwner != nil {
		return nil
	}
	scope := t.relationLifetime
	if scope == nil || scope.dataPolicy != relationCallerData || scope.transactionalSchema != t.ScopedSchema {
		return nil
	}
	return scope
}

// The exact native producer keeps each attested lexical input scope. Sibling
// tables may reuse that lifetime; their actual row-write rights remain separate.
// A same-state invocation or another scope is never such a proof.
func (t *table) callerInputBorrowed(tx *TxContext) bool {
	if tx == nil || tx.mutationOperation == nil {
		return false
	}
	scope := t.callerRelationScope()
	if scope == nil {
		return false
	}
	inputs := tx.mutationOperation.callerInputs.Load()
	return inputs != nil && (*inputs)[scope]
}

// A coarse source/producer boundary must also join the actual native
// invocation. Map membership alone cannot prevent that invocation retiring
// between admission and the first physical input. Row sinks instead use their
// existing borrowMutation/frame owner before any effect.
func (t *table) beginCallerInput(tx *TxContext) func() {
	scope := t.callerRelationScope()
	if scope == nil {
		return nil
	}
	if t.ScopedSchema {
		// SQL temporary generations use the SAME coarse input/lifetime owner.
		// This also excludes their active readers from a pending header change.
		tx.requireScopedRelationOwner(t)
	}
	if t.callerInputBorrowed(tx) {
		leave := tx.mutationOperation.borrowMutation(tx)
		return func() { leave(nil) }
	}
	return scope.beginDataUse(tx)
}
