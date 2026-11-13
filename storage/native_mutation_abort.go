/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "errors"
import "context"
import "github.com/launix-de/memcp/scm"

// A terminal control transfer owns no state/Session/operation reference. The
// pending fact belongs to the actual operation, even if Scheme catches this
// value. Frontends classify the type, never an error-message substring.
type nativeTransactionAbortTransfer struct{ transactionID uint64 }

func (transfer *nativeTransactionAbortTransfer) Error() string {
	return "native transaction abort requested by statement completion"
}

func isNativeTransactionAbortTransfer(value any) bool {
	if wrapped, ok := value.(scm.Scmer); ok {
		value = wrapped.Any()
	}
	failure, ok := value.(error)
	if !ok {
		return false
	}
	var transfer *nativeTransactionAbortTransfer
	return errors.As(failure, &transfer)
}

func (tx *TxContext) requestMutationAbort() {
	if tx == nil || tx.mutationOperation == nil {
		panic("transaction abort transfer requires a native completion invocation")
	}
	operation := tx.mutationOperation
	leave := operation.borrowMutation(tx)
	defer leave(nil)
	operation.mu.Lock()
	allowed := operation.abortAllowed && operation.completionRoot != nil
	root := operation.completionRoot
	state := operation.state
	operation.mu.Unlock()
	if !allowed {
		panic("transaction abort transfer requires a native completion invocation")
	}
	root.mu.Lock()
	if root.state != state || root.statement == nil || root.statement.finished {
		root.mu.Unlock()
		panic("transaction abort transfer lost its statement owner")
	}
	root.statement.abort = true
	root.mu.Unlock()
	panic(&nativeTransactionAbortTransfer{transactionID: state.ID})
}

// Caller owns the actual state.finalizationMu and the drained outer operation.
// Reuse the SAME resource/decision/CS-undo owners; do not call public Rollback
// from a live callback or close resources while the operation still owns them.
// A failed row undo keeps the operation, exclusions and exact retry suffixes.
func (operation *nativeMutationOperation) rollbackDrainedStatement(invocation *TxContext) {
	state := operation.state
	operation.mu.Lock()
	valid := operation.closed && operation.producerDone && operation.activeCalls == 0 && operation.openFrames == 0 &&
		operation.undoOwner == nil && operation.statement != nil && operation.statement.abort && operation.invocation == invocation
	operation.mu.Unlock()
	if !valid {
		panic("native transaction abort requires its drained outer statement")
	}
	state.mu.Lock()
	valid = state.checkedMutation == operation && state.mutationOperations == 1 && state.Mode == TxCursorStability && !state.decisionStarted
	aborted := state.State == TxAborted
	active := state.State == TxActive
	state.mu.Unlock()
	if !valid || !active && !aborted {
		panic("native transaction abort lost its original active transaction")
	}
	if aborted {
		return // successful row decision; only later lease/resource retirement remains
	}
	invocation.beginMutationResourceDecision(operation)
	completed := false
	defer func() {
		if !completed {
			invocation.finishResourceDecision()
		}
	}()
	release, err := invocation.lockDecision(context.Background(), true, invocation.beforeDecision("rollback"))
	if err != nil {
		panic(err)
	}
	func() {
		defer func() {
			release()
			if invocation.State == TxAborted {
				invocation.retireCatalogSavepoint(0, 0)
			}
		}()
		invocation.rollbackLocked()
	}()
	// Resources close ONLY after finishStatement and release retire the native
	// operation. Existing closeResources intentionally rejects active producers.
	completed = true
}

func initMutationAbortBuiltins(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "mutation_columns", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) != 1 {
			panic("mutation_columns requires its native completion invocation")
		}
		tx := scmerToTxContext(a[0])
		if tx == nil || tx.mutationOperation == nil {
			panic("mutation_columns requires its native completion invocation")
		}
		operation := tx.mutationOperation
		operation.mu.Lock()
		defer operation.mu.Unlock()
		if !operation.abortAllowed || operation.closed || operation.producerDone || operation.invocation != tx || operation.completionRoot == nil {
			panic("mutation_columns requires a live native completion invocation")
		}
		return operation.completionColumns
	}, Type: &scm.TypeDescriptor{Kind: "func", Description: "borrows the native producer's immutable declared assignment names from its actual completion invocation",
		Params: []*scm.TypeDescriptor{{Kind: "any", Label: "tx"}}, Return: &scm.TypeDescriptor{Kind: "list"}}})
	scm.Declare(en, &scm.Declaration{Name: "request_transaction_abort", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) != 1 {
			panic("request_transaction_abort requires its native completion invocation")
		}
		scmerToTxContext(a[0]).requestMutationAbort()
		return scm.NewNil()
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
		Description: "requests terminal whole-transaction rollback after the native statement producer and callbacks have drained",
		Params:      []*scm.TypeDescriptor{{Kind: "any", Label: "tx"}}, Return: &scm.TypeDescriptor{Kind: "any"}}})
	scm.Declare(en, &scm.Declaration{Name: "transaction_abort_transfer?", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) != 1 {
			panic(fmt.Errorf("transaction_abort_transfer? requires one failure value"))
		}
		return scm.NewBool(isNativeTransactionAbortTransfer(a[0]))
	}, Type: &scm.TypeDescriptor{Kind: "func", Const: true,
		Description: "classifies a native terminal transaction-abort transfer without granting transaction authority",
		Params:      []*scm.TypeDescriptor{{Kind: "any", Label: "failure"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
}
