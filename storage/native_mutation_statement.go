/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"

// One outer native producer owns one undo boundary, including synchronous user
// trigger effects. This is not an isolation-mode conversion or a second journal.
// It remains private until all producer/control/callback callers are ported.
type nativeMutationStatement struct {
	database  *database
	savepoint Savepoint
	ready     bool
	running   bool
	finished  bool
	failure   any
	abort     bool
	tables    map[*table]*statementTableWriteLease
}

// Establish the state exclusion before any target/default/trigger effect. The
// same state.mu also owns native producer admission and public savepoint entry.
func (operation *nativeMutationOperation) beginStatement(db *database) {
	operation.mu.Lock()
	defer operation.mu.Unlock()
	if operation.closed || operation.statement != nil || !operation.localOrigin {
		panic("statement undo requires an unexposed local native producer")
	}
	state := operation.state
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.Mode != TxCursorStability && state.Mode != TxACID {
		panic("CHECK row mutation requires a supported native transaction mode")
	}
	if state.State != TxActive || state.decisionStarted || state.mutationOperations != 1 || state.checkedMutation != nil || state.mutationAdmissionClosed {
		panic(newNativeMutationBusyError(state, "statement entry"))
	}
	operation.statement = &nativeMutationStatement{database: db}
	state.checkedMutation = operation
}

// Called at actual target setup, outside shard/schema/UNIQUE locks. A later
// trigger target joins this same owner before its first native row effect.
// Taking no blocking upgrade avoids waiting while an earlier target is held.
func (operation *nativeMutationOperation) admitStatementTable(t *table) {
	operation.mu.Lock()
	outer, completion := operation.undoOwner, operation.completionRoot
	operation.mu.Unlock()
	if outer != nil {
		outer.admitCompletionStatementTable(t, completion)
		return
	}
	operation.admitCompletionStatementTable(t, nil)
}

// Only a private completion-derived child may join targets while its outer
// producer is sealed. Old producer/frame capabilities receive no such grant.
func (operation *nativeMutationOperation) admitCompletionStatementTable(t *table, completion *nativeMutationOperation) {
	operation.mu.Lock()
	defer operation.mu.Unlock()
	statement := operation.statement
	if operation.closed && !(operation.completing && completion == operation) || statement == nil || statement.finished || t == nil || t.schema != statement.database {
		panic("native statement target is outside its admitted database")
	}
	if statement.tables[t] != nil {
		return
	}
	lease, err := t.writeAdmission.admitStatement(operation, operation.state)
	if err != nil {
		panic(err)
	}
	// The exclusive statement lease now prevents native semantic retirement.
	// Re-attest the actual public endpoint without acquiring schema under a
	// callback/operation lock. The database mode resource rechecks its own
	// identity before the first effect. Scope-owned temporary relations require
	// their separate lifetime contract and are not silently treated as public.
	name := t.publishedName.Load()
	private := t.dataOwner != nil && t.Ephemeral && t.PersistencyMode == Memory &&
		operation.invocation.relationData != nil
	if private {
		coordinates, finish := operation.invocation.relationData.borrow()
		private = coordinates.owner == t.dataOwner && coordinates.writable && t.dataOwner.owns(t)
		finish()
	}
	if name == nil || GetDatabase(t.schema.Name) != t.schema || t.schema.tables.Get(*name) != t ||
		(!private && !t.committedAt(t.schema.catalogView().version)) {
		_ = lease.Close()
		panic("native statement requires the current committed public target")
	}
	if statement.tables == nil {
		statement.tables = make(map[*table]*statementTableWriteLease)
	}
	statement.tables[t] = lease
}

// Native public INSERT is an actual writer entry, not an inner row-reader
// usage counter. Its borrow covers defaults, triggers, routing and all sinks,
// including calls made by a foreign holder of this immutable invocation.
func (operation *nativeMutationOperation) borrowMutation(binding *TxContext) func(any) {
	operation.mu.Lock()
	if operation.closed || operation.producerDone || operation.state == nil || operation.invocation != binding || operation.state != binding.transactionState {
		operation.mu.Unlock()
		panic("native mutation invocation has finished")
	}
	operation.activeCalls++
	operation.mu.Unlock()
	return func(failure any) {
		if failure != nil {
			operation.failStatement(failure)
		}
		operation.mu.Lock()
		operation.activeCalls--
		operation.changed.Broadcast()
		operation.mu.Unlock()
	}
}

// Called only at stored-data mutation admission. It grants no right to bypass
// a frame's borrowed UNIQUE/batch lifetime and performs no reader hot-loop work.
func (tx *TxContext) requireMutationBinding() {
	if operation := tx.mutationOperation; operation != nil {
		operation.mu.Lock()
		defer operation.mu.Unlock()
		if operation.closed || operation.state == nil || operation.state != tx.transactionState || operation.invocation != tx {
			panic("native mutation invocation has finished")
		}
		return
	}
	tx.mu.Lock()
	busy := tx.checkedMutation != nil
	tx.mu.Unlock()
	if busy {
		panic(newNativeMutationBusyError(tx.transactionState, "row mutation"))
	}
}

// All initial targets are admitted before this call. The standalone resource
// factory runs without operation/state/shard locks, pins the actual database
// mode until native decision, and rejects replacement/distributed authority.
func (operation *nativeMutationOperation) prepareStatement() {
	invocation := operation.invocation
	statement := operation.statement
	if statement == nil || statement.ready {
		panic("native statement undo was not prepared exactly once")
	}
	invocation.admitStandaloneMutation(statement.database)
	statement.savepoint = invocation.createMutationSavepoint()
	operation.mu.Lock()
	defer operation.mu.Unlock()
	statement.ready, statement.running = true, true
}

// The native producer/mutation boundary records errors even if caller Scheme
// catches them. A caught failed row must not publish a partial checked command.
func (operation *nativeMutationOperation) failStatement(failure any) {
	operation.mu.Lock()
	if outer := operation.undoOwner; outer != nil {
		operation.mu.Unlock()
		outer.failStatement(failure)
		return
	}
	defer operation.mu.Unlock()
	if statement := operation.statement; statement != nil && statement.failure == nil {
		statement.failure = failure
	}
}

// Called by the one cleanup owner AFTER sealAndDrain and endStatement, but
// BEFORE release drops native state ownership. It may be retried after failed
// undo; the same leases and savepoint stay retained until restoration succeeds.
func (operation *nativeMutationOperation) finishStatement() any {
	operation.mu.Lock()
	statement := operation.statement
	if statement == nil || statement.finished {
		operation.mu.Unlock()
		return nil
	}
	if !operation.closed || !operation.producerDone || operation.activeCalls != 0 || operation.openFrames != 0 {
		operation.mu.Unlock()
		panic("native statement undo requires a drained producer")
	}
	statement.running = false
	invocation := operation.invocation
	operation.mu.Unlock()
	if statement.abort {
		operation.rollbackDrainedStatement(invocation)
	} else if statement.ready && statement.failure != nil {
		invocation.restoreMutationSavepoint(statement.savepoint)
	}
	// Do not release exclusions on a failed restore, or label it restored.
	for _, lease := range statement.tables {
		_ = lease.Close()
	}
	operation.mu.Lock()
	defer operation.mu.Unlock()
	state := operation.state
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.checkedMutation != operation {
		panic("native statement undo lost its transaction owner")
	}
	state.checkedMutation = nil
	statement.finished = true
	statement.tables = nil
	statement.savepoint = Savepoint{}
	statement.database = nil
	if statement.abort {
		return &nativeTransactionAbortTransfer{transactionID: state.ID}
	}
	if statement.ready && statement.failure != nil {
		return &nativeStatementRestoredError{cause: statement.failure}
	}
	return statement.failure
}

// This typed fact means the original statement failed AND its whole undo
// succeeded. Only an explicit-at-entry wrapper may preserve earlier Tx work.
// It never turns a failed rollback, busy decision or arbitrary error into success.
type nativeStatementRestoredError struct{ cause any }

func (failure *nativeStatementRestoredError) Error() string { return fmt.Sprint(failure.cause) }

// OriginalFailure is consumed only at the language handler edge. Native
// transaction owners retain this restored marker while deciding rollback.
func (failure *nativeStatementRestoredError) OriginalFailure() any { return failure.cause }
func (failure *nativeStatementRestoredError) Unwrap() error {
	cause, _ := failure.cause.(error)
	return cause
}

// Public/native savepoints seal producer admission for their entire operation.
// Strict triggers inside a checked statement use its ONE outer undo boundary;
// arbitrary public savepoint rewinds cannot borrow that internal permission.
func (tx *TxContext) admitMutationSavepoint() func() {
	state := tx.transactionState
	state.mu.Lock()
	// Legacy parallel trigger savepoints still serialize at lockDecision. All
	// of them seal NEW producer entry until the last exits; a second savepoint
	// is not itself a new producer. Terminal decisions remain excluded.
	if state.mutationOperations != 0 || state.mutationDecisionAttempts != 0 || state.mutationAdmissionClosed && state.mutationSavepoints == 0 {
		state.mu.Unlock()
		panic(newNativeMutationBusyError(state, "savepoint"))
	}
	state.mutationAdmissionClosed = true
	state.mutationSavepoints++
	state.mu.Unlock()
	return func() {
		state.mu.Lock()
		defer state.mu.Unlock()
		state.mutationSavepoints--
		if state.mutationSavepoints == 0 && state.mutationDecisionAttempts == 0 && state.State == TxActive && !state.decisionStarted {
			state.mutationAdmissionClosed = false
		}
	}
}

func (tx *TxContext) hasStatementUndo() bool {
	if tx == nil || tx.mutationOperation == nil {
		return false
	}
	operation := tx.mutationOperation
	operation.mu.Lock()
	if outer := operation.undoOwner; outer != nil {
		operation.mu.Unlock()
		operation = outer
		operation.mu.Lock()
	}
	defer operation.mu.Unlock()
	return operation.statement != nil && operation.statement.running && !operation.statement.finished
}
