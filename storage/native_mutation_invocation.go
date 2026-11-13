/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "github.com/launix-de/memcp/scm"

// This is a private setup primitive, not yet a registered API or a grant to
// mutate a target. The target/admission and all producer/control ports must be
// completed before callers can expose it to Scheme.
func newNativeMutationBinding(parent *TxContext, executionSession scm.Scmer) (*nativeMutationOperation, error) {
	if parent == nil {
		return nil, fmt.Errorf("native mutation producer requires an actual transaction")
	}
	if parent.mutationOperation != nil {
		return newNestedNativeMutationBinding(parent, executionSession)
	}
	parent.bindingMu.Lock()
	defer parent.bindingMu.Unlock()
	if parent.bindingAttempt != nil {
		return nil, newNativeMutationBusyError(parent.bindingAttempt.state, "mutation entry")
	}
	state := parent.transactionState
	if state == nil {
		return nil, fmt.Errorf("native mutation producer has no transaction binding")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.mutationAdmissionClosed || state.checkedMutation != nil {
		return nil, newNativeMutationBusyError(state, "mutation entry")
	}
	if state.State != TxActive {
		return nil, fmt.Errorf("native mutation producer requires an active transaction")
	}
	if executionSession.IsNil() {
		executionSession = parent.Session
	}
	// A fresh binding owns its work/readers. Session is initialized exactly once,
	// never temporarily installed on the reusable parent executor. Caller-provided
	// context data does NOT turn an imported/synthetic invocation into a local one.
	binding := &TxContext{transactionState: state, Session: executionSession, SessionState: parent.SessionState,
		remoteObject: parent.remoteObject, invocationContext: parent.invocationContext, relationData: parent.relationData}
	binding.fanoutOwner = parent
	if parent.fanoutOwner != nil {
		binding.fanoutOwner = parent.fanoutOwner
	}
	if parent.queryActive.Load() {
		binding.querySeq.Store(parent.querySeq.Load())
		binding.statementGeneration.Store(parent.statementGeneration.Load())
		binding.queryInfo.Store(parent.queryInfo.Load())
	} else {
		// Native controls can have an actual producer without a frontend request.
		// This is a cache/work scope, not a forged connection/query identity.
		binding.statementGeneration.Store(statementGenerationCounter.Add(1))
	}
	binding.queryActive.Store(true)
	operation := newNativeMutationOperationLocked(state, binding)
	operation.localOrigin = !parent.Session.IsNil() && parent.remoteObject == nil && !parent.remoteInvocation && state.coordinator == nil
	if parent.queryActive.Load() {
		// The query's endStatement detaches this exact list under the same state.mu.
		// Calling RetainQueryWork here would recursively lock that mutex.
		if !parent.retainQueryWorkLocked(operation.close) {
			// No target/effect has been exposed. Unwind this constructor directly;
			// normal operation.close would recursively acquire state.mu.
			state.mutationOperations--
			binding.mutationOperation = nil
			operation.state = nil
			operation.invocation = nil
			return nil, fmt.Errorf("native mutation parent statement is retiring")
		}
	}
	return operation, nil
}

// A nested synchronous SQL producer gets its own immutable execution-session
// binding and work lists. Only its actual live parent operation grants entry;
// another TxContext with an equal state pointer cannot construct this child.
// The parent joins this child's cleanup before giving back its own row rights.
func newNestedNativeMutationBinding(parent *TxContext, executionSession scm.Scmer) (*nativeMutationOperation, error) {
	parent.bindingMu.Lock()
	defer parent.bindingMu.Unlock()
	outer := parent.mutationOperation
	outer.mu.Lock()
	defer outer.mu.Unlock()
	state := outer.state
	if outer.closed || outer.producerDone || state == nil || parent != outer.invocation {
		return nil, fmt.Errorf("nested native mutation parent has finished")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	undo := outer
	if outer.undoOwner != nil {
		undo = outer.undoOwner
	}
	if state.State != TxActive || state.decisionStarted || state.mutationAdmissionClosed || state.checkedMutation != undo || !parent.queryActive.Load() {
		return nil, newNativeMutationBusyError(state, "nested mutation entry")
	}
	if executionSession.IsNil() {
		executionSession = parent.Session
	}
	binding := &TxContext{transactionState: state, Session: executionSession, SessionState: parent.SessionState,
		remoteObject: parent.remoteObject, invocationContext: parent.invocationContext, fanoutOwner: parent.fanoutOwner, relationData: parent.relationData}
	binding.querySeq.Store(parent.querySeq.Load())
	binding.statementGeneration.Store(parent.statementGeneration.Load())
	binding.queryInfo.Store(parent.queryInfo.Load())
	binding.queryActive.Store(true)
	child := newNativeMutationOperationLocked(state, binding)
	child.localOrigin, child.undoOwner, child.completionRoot = outer.localOrigin, undo, outer.completionRoot
	outer.linkChildLocked(child)
	if !parent.retainQueryWorkLocked(child.close) {
		outer.unlinkChildLocked(child)
		state.mutationOperations--
		binding.mutationOperation = nil
		child.state, child.invocation, child.undoOwner = nil, nil, nil
		return nil, fmt.Errorf("nested native mutation parent is retiring")
	}
	return child, nil
}
