/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "context"
import "github.com/launix-de/memcp/scm"

// This private constructor is called ONLY by the drained operation's close
// owner. It does not reopen the producer, its query work or any escaped frame.
// Completion gets an independent invocation/work scope over the SAME state.
func (operation *nativeMutationOperation) newCompletionBinding(effect *nativeMutationEffect) *nativeMutationOperation {
	operation.mu.Lock()
	defer operation.mu.Unlock()
	state, parent := operation.state, operation.invocation
	if state == nil || parent == nil || !operation.closed || !operation.producerDone || operation.activeCalls != 0 || operation.openFrames != 0 {
		panic("native completion requires its drained producer")
	}
	undo := operation
	if operation.undoOwner != nil {
		undo = operation.undoOwner
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.checkedMutation != undo || state.State != TxActive || state.decisionStarted {
		panic("native completion lost its original statement owner")
	}
	binding := &TxContext{transactionState: state, Session: parent.Session, SessionState: parent.SessionState,
		remoteObject: parent.remoteObject, invocationContext: parent.invocationContext, fanoutOwner: parent.fanoutOwner, relationData: parent.relationData}
	binding.querySeq.Store(parent.querySeq.Load())
	binding.queryInfo.Store(parent.queryInfo.Load())
	binding.statementGeneration.Store(statementGenerationCounter.Add(1))
	binding.queryActive.Store(true)
	child := newNativeMutationOperationLocked(state, binding)
	child.localOrigin, child.undoOwner, child.completionRoot, child.abortAllowed = operation.localOrigin, undo, undo, true
	child.completionColumns = effect.columns
	if effect.replacement {
		child.replacementTarget, child.replacementEvent = effect.target, effect.event
	}
	operation.linkChildLocked(child)
	return child
}

// One completion pass occurs AFTER all producer/frame/query work has joined,
// while the SAME statement undo/target exclusions remain held. Publication and
// all callbacks run outside source shard/UNIQUE/schema locks. A retry after a
// failed undo can retire the scope, but never replays an executed callback.
func (operation *nativeMutationOperation) completeEffects() {
	operation.mu.Lock()
	if operation.effectsAttempted {
		operation.mu.Unlock()
		operation.joinChildren()
		operation.closeEffectScope()
		return
	}
	operation.effectsAttempted = true
	invocation := operation.invocation
	effects := operation.effectOrder
	undo := operation
	if operation.undoOwner != nil {
		undo = operation.undoOwner
	}
	operation.completing = true
	operation.mu.Unlock()
	defer func() {
		operation.mu.Lock()
		operation.completing = false
		operation.mu.Unlock()
	}()
	undo.mu.Lock()
	failed := undo.statement != nil && undo.statement.failure != nil
	undo.mu.Unlock()
	if failed {
		operation.closeEffectScope()
		return
	}
	var failure any
	func() {
		defer func() { failure = recover() }()
		for _, effect := range effects {
			if len(effect.programs) == 0 || operation.abortRequested() {
				continue
			}
			operation.mu.Lock()
			scope := operation.effectScope
			operation.mu.Unlock()
			if scope == nil {
				scope = newRelationScope(relationScopeOwnerFor(invocation.Session), nil, relationCallerData, false)
				operation.mu.Lock()
				operation.effectScope = scope
				operation.mu.Unlock()
			}
			effect.old.prepare()
			effect.new.prepare()
			effect.old.publish(scope, undo)
			effect.new.publish(scope, undo)
			completion := operation.newCompletionBinding(effect)
			completion.runProducer(func(tx *TxContext) scm.Scmer {
				for _, program := range effect.programs {
					if operation.abortRequested() {
						break
					}
					if failure := scanQueryContext(tx, context.Background()).Err(); failure != nil {
						panic(failure)
					}
					if program.program.IsNil() {
						panic("native statement completion has no finalized program")
					}
					func() {
						defer func() {
							if cause := recover(); cause != nil {
								panic(fmt.Errorf("native statement completion %s failed: %w", program.name, nativeMutationFailure(cause)))
							}
						}()
						scm.Apply(program.program, NewTableScmer(effect.old.table), NewTableScmer(effect.new.table), tx.Session, txContextScmer(tx))
						if failure := scanQueryContext(tx, context.Background()).Err(); failure != nil {
							panic(failure)
						}
					}()
				}
				return scm.NewNil()
			})
		}
	}()
	if failure != nil {
		operation.failStatement(failure)
	}
	// A failed completion child remains linked until its actual close succeeds.
	// Never retire its borrowed parent relations or enter outer undo first.
	operation.joinChildren()
	// Scope.close attempts all children/relations and retains failed entries.
	// Do not erase the op's retry continuation if retirement itself fails.
	operation.closeEffectScope()
}

func nativeMutationFailure(value any) error {
	if wrapped, ok := value.(scm.Scmer); ok {
		value = wrapped.Any()
	}
	if failure, ok := value.(error); ok {
		return failure
	}
	return fmt.Errorf("%v", value)
}

func (operation *nativeMutationOperation) closeEffectScope() {
	for _, effect := range operation.effectOrder {
		if effect.old != nil {
			effect.old.access.Store(nil)
			effect.new.access.Store(nil)
		}
	}
	operation.mu.Lock()
	scope := operation.effectScope
	operation.mu.Unlock()
	if scope != nil {
		scope.close()
		operation.mu.Lock()
		operation.effectScope = nil
		operation.mu.Unlock()
	}
	for _, effect := range operation.effectOrder {
		if effect.old != nil {
			effect.old.clearPayload()
			effect.new.clearPayload()
		}
	}
}

func (operation *nativeMutationOperation) abortRequested() bool {
	if operation.undoOwner != nil {
		operation = operation.undoOwner
	}
	operation.mu.Lock()
	defer operation.mu.Unlock()
	return operation.statement != nil && operation.statement.abort
}
