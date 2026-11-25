/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "context"
import "strings"
import "unsafe"
import "github.com/launix-de/memcp/scm"

// This is a continuation of the existing outer completion owner, not another
// transaction registry. owner.mu/changed serialize its coarse statement entry;
// owner.closeMu is the sole cleanup/continuation executor. There is no row-level
// registration and no new synchronization family.
type nativeAbortContinuation struct {
	owner             *nativeMutationOperation
	original          *TxContext
	originalState     *transactionState
	callback, payload scm.Scmer
	scope             *relationScope
	effects           []*nativeMutationEffect
	old, new          *nativeTransitionRelation
	mode              *databaseLocalDeclarationLease
	database          *database
	context           context.Context
	sessionState      *scm.SessionState
	querySequence     uint64
	phase             nativeAbortPhase
	active            bool
	leaf              *TxContext
	failure           any
	bytes             uint
}

type nativeAbortPhase uint8

const (
	nativeAbortPending nativeAbortPhase = iota
	nativeAbortRunning
	nativeAbortClosing
	nativeAbortClosed
)

// Only source-sized inert language data crosses the transfer. Copy container
// headers and string backing before its old lexical scopes unwind. A function,
// native handle, session or transaction cannot be smuggled into this payload.
func sealAbortPayload(value scm.Scmer, path map[scm.Scmer]bool) scm.Scmer {
	value = value.WithoutSourceInfo()
	if value.IsNil() || value.IsBool() || value.IsInt() || value.IsFloat() || value.IsDate() || value.IsSymbol() {
		return value
	}
	if value.IsUUID() {
		id, err := scm.UUIDValue(value)
		if err != nil {
			panic(err)
		}
		return scm.NewUUID(id)
	}
	if value.IsString() {
		return scm.NewString(strings.Clone(value.String()))
	}
	if value.IsSlice() || value.IsFastDict() {
		if path[value] {
			panic("cyclic native continuation payload")
		}
		path[value] = true
		defer delete(path, value)
		var source []scm.Scmer
		if value.IsFastDict() {
			source = value.FastDict().ToList()
		} else {
			source = value.Slice()
		}
		items := make([]scm.Scmer, len(source))
		for i, item := range source {
			items[i] = sealAbortPayload(item, path)
		}
		if value.IsFastDict() {
			dict := scm.NewFastDictValue(len(items) / 2)
			for i := 0; i < len(items); i += 2 {
				dict.Set(items[i], items[i+1], nil)
			}
			return scm.NewFastDict(dict)
		}
		return scm.NewSlice(items)
	}
	panic("native continuation payload requires immutable scalar or list data")
}

func (operation *nativeMutationOperation) prepareAbortContinuation(callback, payload scm.Scmer) *nativeAbortContinuation {
	if !callback.IsProc() && !callback.IsNativeFunc() && !callback.IsJIT() {
		panic("native abort continuation requires an invocation callback")
	}
	operation.mu.Lock()
	effect, binding, root := operation.completionEffect, operation.invocation, operation.completionRoot
	valid := operation.abortAllowed && !operation.closed && binding != nil && root != nil && effect != nil
	operation.mu.Unlock()
	if !valid || effect.old == nil || effect.new == nil {
		panic("native abort continuation requires its actual materialized completion")
	}
	access := effect.old.access.Load()
	if access == nil || effect.new.access.Load() == nil || access.root != root {
		panic("native abort continuation lost its transition lifetime")
	}
	payload = sealAbortPayload(payload, make(map[scm.Scmer]bool))
	// The same native database authority is pinned BEFORE original resources
	// retire. No new read-lock acquisition is hidden in a row callback.
	mode := effect.target.schema.admitLocalCatalog(false)
	accepted := false
	defer func() {
		if !accepted {
			_ = mode.Close()
		}
	}()
	root.mu.Lock()
	defer root.mu.Unlock()
	if root.state != operation.state || root.statement == nil || root.statement.finished || root.abortContinuation != nil {
		panic("native abort continuation lost its original statement owner")
	}
	continuation := &nativeAbortContinuation{owner: root, original: root.invocation, originalState: root.state,
		callback: callback, payload: payload, scope: access.scope, old: effect.old, new: effect.new,
		mode: mode, database: effect.target.schema, context: scanQueryContext(binding, context.Background()),
		sessionState: binding.SessionState, querySequence: binding.querySeq.Load()}
	// The scope may belong to a nested completion. Its actual effect directory
	// is captured by that owner's scope handoff, never inferred from SQL names.
	continuation.bytes = uint(unsafe.Sizeof(*continuation)) + scm.ComputeSize(payload)
	continuation.database.adjustMetadataMemory(0, continuation.bytes)
	root.abortContinuation = continuation
	accepted = true
	return continuation
}

func (operation *nativeMutationOperation) retainedAbortScope(scope *relationScope) *nativeAbortContinuation {
	root := operation
	if operation.undoOwner != nil {
		root = operation.undoOwner
	}
	root.mu.Lock()
	defer root.mu.Unlock()
	continuation := root.abortContinuation
	if continuation != nil && continuation.scope == scope && continuation.phase != nativeAbortClosed {
		continuation.effects = operation.effectOrder
		return continuation
	}
	return nil
}

// The old completion descriptor is revoked before Undo. Fresh descriptors are
// installed only after the original operation and resources have retired.
func (continuation *nativeAbortContinuation) publishReads() {
	for _, effect := range continuation.effects {
		if effect.old != nil {
			effect.old.access.Store(&nativeTransitionReadAccess{scope: continuation.scope, continuation: continuation})
			effect.new.access.Store(&nativeTransitionReadAccess{scope: continuation.scope, continuation: continuation})
		}
	}
}

// One callback invocation is recorded before entering language code. Retry owns
// only pending decisions/resource retirement; it can NEVER replay that callback.
// The existing outer closeMu is held, but no old decision/schema/shard lock is.
func (operation *nativeMutationOperation) finishAbortContinuation() {
	operation.mu.Lock()
	continuation := operation.abortContinuation
	operation.mu.Unlock()
	if continuation == nil {
		return
	}
	if continuation.phase == nativeAbortClosed {
		return
	}
	if continuation.phase == nativeAbortPending {
		original := continuation.original
		original.finalizationMu.Lock()
		func() {
			defer original.finalizationMu.Unlock()
			original.finishResourceDecision()
			// Resource cleanup is not inferred from the row rollback or a
			// cleared invocation pointer. Attest the actual terminal owner.
			scope := original.resourceScope()
			scope.mu.Lock()
			original.mu.Lock()
			ready := original.State == TxAborted && original.checkedMutation == operation &&
				original.mutationOperations == 0 && original.mutationSavepoints == 0 &&
				scope.closed && scope.constructing == 0 && len(scope.entries) == 0
			original.mu.Unlock()
			scope.mu.Unlock()
			if !ready {
				panic("native abort continuation requires retired original resources")
			}
			if !original.Session.IsNil() {
				scm.Apply(original.Session, scm.NewString("transaction"), scm.NewNil())
			}
		}()
		operation.mu.Lock()
		continuation.phase = nativeAbortRunning
		operation.mu.Unlock()
		func() {
			defer func() { continuation.failure = recover() }()
			if err := continuation.context.Err(); err != nil {
				panic(err)
			}
			continuation.publishReads()
			execute := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return continuation.statement(args...) })
			scm.Apply(continuation.callback, execute, NewTableScmer(continuation.old.table), NewTableScmer(continuation.new.table), continuation.payload)
		}()
		operation.mu.Lock()
		continuation.phase = nativeAbortClosing
		operation.mu.Unlock()
	}
	operation.mu.Lock()
	for continuation.active {
		operation.changed.Wait()
	}
	operation.mu.Unlock()
	// A failed accepted leaf decision has its normal native finisher. No leaf
	// producer is re-entered, and no original-T journal is adopted or reset.
	continuation.finishLeaf()
	for _, effect := range continuation.effects {
		if effect.old != nil {
			effect.old.access.Store(nil)
			effect.new.access.Store(nil)
		}
	}
	continuation.scope.close()
	for _, effect := range continuation.effects {
		if effect.old != nil {
			effect.old.clearPayload()
			effect.new.clearPayload()
		}
	}
	continuation.original.finalizationMu.Lock()
	func() {
		defer continuation.original.finalizationMu.Unlock()
		if !continuation.original.Session.IsNil() {
			scm.Apply(continuation.original.Session, scm.NewString("transaction"), scm.NewNil())
		}
		continuation.original.finishRequestedSessionClose()
	}()
	operation.mu.Lock()
	failure := continuation.failure
	continuation.failure = nil
	continuation.callback, continuation.payload = scm.NewNil(), scm.NewNil()
	continuation.scope, continuation.effects, continuation.old, continuation.new = nil, nil, nil, nil
	continuation.context, continuation.sessionState = nil, nil
	continuation.original = nil
	continuation.phase = nativeAbortClosed
	state := continuation.originalState
	continuation.originalState = nil
	state.mu.Lock()
	if state.checkedMutation != operation {
		state.mu.Unlock()
		operation.mu.Unlock()
		panic("abort continuation lost its recovery marker")
	}
	state.checkedMutation = nil
	state.mu.Unlock()
	if failure != nil {
		operation.completionFailure = &nativeTransactionAbortTransfer{transactionID: state.ID, continuationFailure: failure}
	}
	operation.mu.Unlock()
	continuation.database.adjustMetadataMemory(continuation.bytes, 0)
	continuation.bytes = 0
	_ = continuation.mode.Close()
	continuation.mode, continuation.database = nil, nil
}

// Each admitted leaf owns an ordinary fresh ACID transaction and its existing
// work/undo/decision resources. The original reusable executor is never reset.
func (continuation *nativeAbortContinuation) statement(args ...scm.Scmer) (result scm.Scmer) {
	if len(args) != 2 || (!args[0].IsProc() && !args[0].IsNativeFunc() && !args[0].IsJIT()) ||
		(!args[1].IsProc() && !args[1].IsNativeFunc() && !args[1].IsJIT()) {
		panic("native continuation statement requires a fresh session and producer")
	}
	owner := continuation.owner
	owner.mu.Lock()
	if continuation.phase != nativeAbortRunning || continuation.active || continuation.leaf != nil {
		owner.mu.Unlock()
		panic("native continuation statement is not admitting work")
	}
	continuation.active = true
	owner.mu.Unlock()
	defer func() { owner.mu.Lock(); continuation.active = false; owner.changed.Broadcast(); owner.mu.Unlock() }()
	if err := continuation.context.Err(); err != nil {
		panic(err)
	}
	session, producer := args[0], args[1]
	if !scm.Apply(session, scm.NewString("__memcp_tx")).IsNil() {
		panic("native continuation requires a fresh execution facade")
	}
	tx := NewTxContext(TxACID)
	tx.autoCommit, tx.Session, tx.SessionState, tx.invocationContext, tx.postAbort = true, session, continuation.sessionState, continuation.context, continuation
	tx.querySeq.Store(continuation.querySequence)
	tx.statementGeneration.Store(statementGenerationCounter.Add(1))
	tx.queryActive.Store(true)
	// The scheduling limit is initialized by NewTxContext; no row owns another
	// budget or a pointer to the retired original executor.
	owner.mu.Lock()
	continuation.leaf = tx
	owner.mu.Unlock()
	scm.Apply(session, scm.NewString("__memcp_tx"), txContextScmer(tx))
	defer scm.Apply(session, scm.NewString("__memcp_tx"), scm.NewNil())
	var failure any
	func() { defer func() { failure = recover() }(); result = scm.Apply(producer, txContextScmer(tx)) }()
	func() {
		defer func() {
			if cause := recover(); cause != nil && failure == nil {
				failure = cause
			}
		}()
		tx.endStatement()
	}()
	if failure == nil {
		if err := tx.Commit(); err != nil {
			panic(err)
		}
	} else {
		tx.Rollback()
	}
	owner.mu.Lock()
	continuation.leaf = nil
	owner.mu.Unlock()
	if failure != nil {
		panic(failure)
	}
	return result
}

func (continuation *nativeAbortContinuation) finishLeaf() {
	continuation.owner.mu.Lock()
	tx := continuation.leaf
	continuation.owner.mu.Unlock()
	if tx == nil {
		return
	}
	if tx.decisionPending() || tx.State == TxCommitted {
		if err := tx.Commit(); err != nil {
			panic(err)
		}
	} else {
		tx.Rollback()
	}
	continuation.owner.mu.Lock()
	continuation.leaf = nil
	continuation.owner.mu.Unlock()
}

// Cold producer entry attests this exact fresh state. Escaped facades never
// gain another decision or a persistent trigger policy after the callback.
func (continuation *nativeAbortContinuation) requireProducer(tx *TxContext) {
	continuation.owner.mu.Lock()
	defer continuation.owner.mu.Unlock()
	if !continuation.active || continuation.leaf == nil || continuation.leaf.transactionState != tx.transactionState ||
		continuation.phase == nativeAbortPending || continuation.phase == nativeAbortClosed {
		panic("post-decision execution has finished")
	}
}

// Only execution bindings derived by native owners carry this immutable
// selection policy. It changes user-program selection, not system invariants.
type nativeTriggerProfile uint8

const (
	nativeAllTriggers nativeTriggerProfile = iota
	nativeInternalTriggers
)

func nativeTriggerProfileFor(tx *TxContext) nativeTriggerProfile {
	if tx != nil && tx.postAbort != nil {
		return nativeInternalTriggers
	}
	return nativeAllTriggers
}

// Both native system programs and hidden Scheme maintenance programs have
// existing non-user declaration owners (the visible argument of createtrigger).
// Names, language strings and generated SQL text do not classify a program.
func isUserTrigger(header TriggerDescription) bool {
	return !header.IsSystem && !header.Hidden
}
