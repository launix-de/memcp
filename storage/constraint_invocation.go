/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "sync"
import "github.com/launix-de/memcp/scm"

// A definition's historical scan uses its actual invocation session, without
// installing it on the reusable transaction executor or retaining it in code.
// Catalog construction, not a mutation counter, fences this metadata producer.
type constraintDefinitionInvocation struct {
	closeMu sync.Mutex
	binding *TxContext
}

func bindConstraintDefinition(parent *TxContext, session scm.Scmer) *constraintDefinitionInvocation {
	if parent == nil || session.IsNil() || (!session.IsProc() && !session.IsNativeFunc() && !session.IsJIT()) {
		panic("constraint definition requires a native transaction and invocation session")
	}
	parent.bindingMu.Lock()
	defer parent.bindingMu.Unlock()
	state := parent.transactionState
	if state == nil || parent.bindingAttempt != nil || parent.mutationOperation != nil {
		panic("constraint definition invocation is not admitting metadata")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.State != TxActive || state.decisionStarted || state.mutationAdmissionClosed || state.mutationOperations != 0 {
		panic(newNativeMutationBusyError(state, "constraint definition"))
	}
	if parent.Session.IsNil() || parent.remoteObject != nil || parent.remoteInvocation || state.coordinator != nil {
		panic("constraint definition requires a local native invocation")
	}
	binding := &TxContext{transactionState: state, Session: session, SessionState: parent.SessionState,
		invocationContext: parent.invocationContext, fanoutOwner: parent}
	if parent.fanoutOwner != nil {
		binding.fanoutOwner = parent.fanoutOwner
	}
	binding.querySeq.Store(parent.querySeq.Load())
	binding.statementGeneration.Store(parent.statementGeneration.Load())
	binding.queryInfo.Store(parent.queryInfo.Load())
	binding.queryActive.Store(true)
	invocation := &constraintDefinitionInvocation{binding: binding}
	// The synchronous producer owns this lock until its finish defer. Parent
	// query cleanup cannot clear the binding or join its lists before the body
	// has even entered catalog construction (which fences decision/reset).
	invocation.closeMu.Lock()
	if parent.queryActive.Load() && !parent.retainQueryWorkLocked(invocation.close) {
		panic("constraint definition parent invocation is retiring")
	}
	return invocation
}

func (invocation *constraintDefinitionInvocation) finish() {
	invocation.closeMu.Unlock()
	invocation.close()
}

// Concurrent parent/producer cleanup joins the same actual work. A second
// closer cannot treat queryActive=false as proof that the first join finished.
func (invocation *constraintDefinitionInvocation) close() {
	invocation.closeMu.Lock()
	defer invocation.closeMu.Unlock()
	if invocation.binding != nil {
		invocation.binding.endStatement()
		invocation.binding = nil
	}
}
