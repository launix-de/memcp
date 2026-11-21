/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "runtime"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"

// nativeMutationBusyError is a pre-decision conflict, never a durable unknown
// outcome. Wrappers must preserve its exact native owner and the original error.
type nativeMutationBusyError struct {
	transactionID uint64
	operation     string
	reason        string
}

func newNativeMutationBusyError(state *transactionState, operation string) *nativeMutationBusyError {
	// Catching an error must not retain a transaction's masks/resources/Session.
	// This scalar identifies the failure only; it is NEVER a mutation grant.
	failure := &nativeMutationBusyError{operation: operation}
	if state != nil {
		failure.transactionID = state.ID
	}
	return failure
}

func (failure *nativeMutationBusyError) Error() string {
	if failure.reason != "" {
		return fmt.Sprintf("%s; retry %s", failure.reason, failure.operation)
	}
	return fmt.Sprintf("native mutation is still active; retry %s", failure.operation)
}

// One attempt owns the reusable executor's slot while its independent decision
// binding uses the OLD state. No I/O or finalizer wait occurs under bindingMu.
// These primitives remain unwired until the query/control wrapper port exists.
type nativeMutationBindingAttempt struct {
	executor *TxContext
	state    *transactionState
	decision *TxContext
	busy     bool
}

// This is an existing session's recovery marker, not an outcome classification.
// Only the attempt that owns the SAME executor slot may preserve it. The busy
// error stays a pre-decision conflict and is returned unchanged on success.
func (tx *TxContext) withMutationDecision(operation string, body func(*TxContext) error) error {
	attempt, err := tx.acquireMutationDecisionAttempt(operation)
	if err != nil {
		return err
	}
	return attempt.runDecision(body)
}

func (tx *TxContext) acquireMutationDecisionAttempt(operation string) (*nativeMutationBindingAttempt, error) {
	attempt, err := tx.beginMutationBindingAttempt(operation)
	if err != nil {
		if attempt != nil {
			promoted := false
			defer func() {
				if !promoted {
					attempt.finish()
				}
			}()
			var resumeFailure any
			func() {
				defer func() { resumeFailure = recover() }()
				promoted = attempt.resumeCheckedUndo()
			}()
			if promoted {
				return attempt, nil
			}
			if resumeFailure != nil {
				if failure, ok := resumeFailure.(error); ok {
					err = fmt.Errorf("native statement undo retry failed: %w", failure)
				} else {
					err = fmt.Errorf("native statement undo retry failed: %v", resumeFailure)
				}
			}
			if !tx.Session.IsNil() {
				var markerFailure any
				func() {
					defer func() { markerFailure = recover() }()
					scm.Apply(tx.Session, scm.NewString("transaction"), scm.NewBool(true))
				}()
				if markerFailure != nil {
					return nil, fmt.Errorf("%w; preserving native transaction ownership: %v", err, markerFailure)
				}
			}
		}
		return nil, err
	}
	return attempt, nil
}

func (attempt *nativeMutationBindingAttempt) resumeCheckedUndo() bool {
	state := attempt.state
	state.mu.Lock()
	operation := state.checkedMutation
	state.mu.Unlock()
	if operation == nil || !operation.resumeDrained() {
		return false
	}
	executor := attempt.executor
	executor.bindingMu.Lock()
	defer executor.bindingMu.Unlock()
	if executor.bindingAttempt != attempt || executor.transactionState != state || !attempt.busy {
		panic("native undo continuation lost its original binding attempt")
	}
	scope := state.resourceScope()
	scope.mu.Lock()
	defer scope.mu.Unlock()
	state.mu.Lock()
	defer state.mu.Unlock()
	terminal := state.State == TxAborted && scope.closed && len(scope.entries) == 0
	if state.mutationOperations != 0 || state.mutationSavepoints != 0 || scope.constructing != 0 ||
		(scope.closed || state.State != TxActive) && !terminal || state.decisionStarted {
		return false
	}
	state.mutationDecisionAttempts++
	state.mutationAdmissionClosed = true
	scope.deciding = true
	attempt.busy = false
	attempt.decision = executor.captureMutationDecisionLocked(state)
	return true
}

// Caller owns this executor's bindingMu; the actual query setup/end owner uses
// that same lock for the captured fields. No work/reader lists are copied.
func (tx *TxContext) captureMutationDecisionLocked(state *transactionState) *TxContext {
	decision := &TxContext{transactionState: state, Session: tx.Session, SessionState: tx.SessionState,
		invocationContext: tx.invocationContext, remoteObject: tx.remoteObject, decisionExecutor: tx}
	decision.querySeq.Store(tx.querySeq.Load())
	decision.statementGeneration.Store(tx.statementGeneration.Load())
	decision.queryInfo.Store(tx.queryInfo.Load())
	decision.queryActive.Store(tx.queryActive.Load())
	return decision
}

// State replacement is private and occurs while the attempt still owns the
// original finalizer AND executor slot. Old resources retire before publication;
// a failed prefix leaves the old slot intact and retryable.
func (attempt *nativeMutationBindingAttempt) resetBinding(previous *TxContext, mode TxMode, session scm.Scmer) error {
	if previous != nil {
		if previous.decisionPending() {
			return fmt.Errorf("transaction commit outcome is unknown; retry COMMIT")
		}
		if previous.sessionClose != nil {
			return fmt.Errorf("native connection cleanup is still pending")
		}
		if previous.State == TxActive {
			return fmt.Errorf("active native transaction must finish before binding reset")
		}
		previous.closeResources()
	}
	replacement := &transactionState{ID: atomic.AddUint64(&txIDCounter, 1), Mode: mode, State: TxActive}
	if mode == TxACID {
		replacement.SnapshotEpoch = atomic.LoadUint64(&GlobalCommitEpoch)
	}
	attempt.replaceState(replacement)
	executor := attempt.executor
	executor.bindingMu.Lock()
	defer executor.bindingMu.Unlock()
	if executor.bindingAttempt != attempt {
		panic("native reset lost its original binding attempt")
	}
	executor.Session = session
	executor.fanoutLimit.Store(int32(runtime.GOMAXPROCS(0)))
	executor.fanoutInUse.Store(0)
	executor.remoteBindings.Store(nil)
	executor.remoteObject = nil
	return nil
}

func (tx *TxContext) beginMutationBindingAttempt(operation string) (*nativeMutationBindingAttempt, error) {
	if tx.relationData != nil {
		return nil, newNativeMutationBusyError(tx.transactionState, "private execution view decision")
	}
	if tx.mutationOperation != nil {
		return nil, newNativeMutationBusyError(tx.transactionState, operation)
	}
	tx.bindingMu.Lock()
	defer tx.bindingMu.Unlock()
	if tx.bindingAttempt != nil {
		return nil, newNativeMutationBusyError(tx.bindingAttempt.state, operation)
	}
	state := tx.transactionState
	attempt := &nativeMutationBindingAttempt{executor: tx, state: state}
	if state != nil {
		scope := state.resourceScope()
		scope.mu.Lock()
		state.mu.Lock()
		if state.mutationOperations != 0 || state.mutationSavepoints != 0 || state.relationDataOrigins != 0 || scope.constructing != 0 || (scope.deciding && state.mutationDecisionAttempts == 0 && state.mutationSavepoints == 0 && state.State == TxActive && !state.decisionStarted) {
			failure := newNativeMutationBusyError(state, operation)
			if scope.constructing != 0 {
				failure.reason = "transaction resource initialization is still active"
			}
			state.mu.Unlock()
			scope.mu.Unlock()
			attempt.busy = true
			tx.bindingAttempt = attempt
			return attempt, failure
		}
		state.mutationDecisionAttempts++
		state.mutationAdmissionClosed = true
		scope.deciding = true
		state.mu.Unlock()
		scope.mu.Unlock()
		attempt.decision = tx.captureMutationDecisionLocked(state)
	}
	tx.bindingAttempt = attempt
	return attempt, nil
}

// runDecision owns failure-prefix unwind, even if the finalizer body panics.
// The attempt never resolves executor.transactionState as its decision owner.
// The initial/unpublished nil-state reset has no previous native finalizer.
func (attempt *nativeMutationBindingAttempt) runDecision(body func(*TxContext) error) error {
	if attempt.busy {
		panic("busy native binding cannot enter a decision")
	}
	if attempt.state != nil {
		attempt.state.finalizationMu.Lock()
		defer attempt.state.finalizationMu.Unlock()
	}
	defer attempt.finish()
	return body(attempt.decision)
}

// finish runs while the OLD finalizer is still owned. A failed pre-decision
// attempt may reopen its active state; accepted decisions stay sealed for their
// existing finisher. This neither changes nor acknowledges the actual outcome.
func (attempt *nativeMutationBindingAttempt) finish() {
	executor := attempt.executor
	executor.bindingMu.Lock()
	defer executor.bindingMu.Unlock()
	if executor.bindingAttempt != attempt {
		panic("native transaction binding attempt lost its executor")
	}
	if state := attempt.state; state != nil && !attempt.busy {
		scope := state.resourceScope()
		scope.mu.Lock()
		state.mu.Lock()
		if state.mutationDecisionAttempts <= 0 {
			state.mu.Unlock()
			scope.mu.Unlock()
			panic("native decision binding lost its state admission")
		}
		state.mutationDecisionAttempts--
		// Independent contexts can wait on the SAME finalizer. One finishing
		// attempt must not reopen writers in front of another admitted decision.
		if !scope.closed && state.mutationDecisionAttempts == 0 && state.mutationSavepoints == 0 && state.State == TxActive && !state.decisionStarted {
			state.mutationAdmissionClosed = false
			scope.deciding = false
		}
		state.mu.Unlock()
		scope.mu.Unlock()
	}
	executor.bindingAttempt = nil
}

// replaceState is only for the attempt's finalizer body, after the OLD decision
// and all its resource cleanup have completed. The replacement was constructed
// privately. Mutating/query callbacks may never call this as a reset shortcut.
func (attempt *nativeMutationBindingAttempt) replaceState(replacement *transactionState) {
	if replacement == nil || replacement == attempt.state {
		panic("native transaction replacement requires a new private state")
	}
	executor := attempt.executor
	executor.bindingMu.Lock()
	defer executor.bindingMu.Unlock()
	if executor.bindingAttempt != attempt || executor.transactionState != attempt.state {
		panic("native transaction binding attempt lost its original state")
	}
	if state := attempt.state; state != nil {
		state.mu.Lock()
		if state.mutationOperations != 0 || state.State == TxActive || state.decisionPending() || state.sessionClose != nil {
			state.mu.Unlock()
			panic("native transaction binding still owns unfinished work")
		}
		state.mu.Unlock()
		// Do not invert the existing scope.mu -> state.mu savepoint order.
		// A terminal native decision alone does not attest resource retirement:
		// failed closers remain in this SAME scope for their existing retry owner.
		scope := state.resourceScope()
		scope.mu.Lock()
		resourcesRetired := scope.closed && scope.constructing == 0 && len(scope.entries) == 0
		scope.mu.Unlock()
		if !resourcesRetired {
			panic("native transaction binding still owns resource cleanup")
		}
	}
	executor.transactionState = replacement
}
