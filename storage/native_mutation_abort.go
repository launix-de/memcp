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
type nativeTransactionAbortTransfer struct {
	transactionID       uint64
	continuationFailure any
}

func (transfer *nativeTransactionAbortTransfer) Error() string {
	if transfer.continuationFailure != nil {
		return fmt.Sprintf("native transaction abort completed; continuation failed: %v", transfer.continuationFailure)
	}
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

func (tx *TxContext) requestMutationAbort(prepare scm.Scmer) {
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
	alreadyRequested := root.statement.abort
	previousFailure := root.statement.abortPreparationFailure
	root.statement.abort = true
	root.mu.Unlock()
	if alreadyRequested {
		panic(&nativeTransactionAbortTransfer{transactionID: state.ID, continuationFailure: previousFailure})
	}
	// The admitted rollback is terminal BEFORE evaluating frontend preparation,
	// sealing its payload or acquiring another lifetime pin. Failure cannot turn
	// this request into a restored statement that keeps earlier Tx writes alive.
	if !prepare.IsNil() {
		func() {
			defer func() {
				if failure := recover(); failure != nil {
					root.mu.Lock()
					if root.statement.abortPreparationFailure == nil {
						root.statement.abortPreparationFailure = failure
					}
					failure = root.statement.abortPreparationFailure
					root.mu.Unlock()
					panic(&nativeTransactionAbortTransfer{transactionID: state.ID, continuationFailure: failure})
				}
			}()
			if !prepare.IsProc() && !prepare.IsNativeFunc() && !prepare.IsJIT() {
				panic("native abort preparation requires an invocation callback")
			}
			prepared := scm.Apply(prepare)
			if !prepared.IsSlice() || len(prepared.Slice()) != 2 {
				panic("native abort preparation requires callback and immutable payload")
			}
			operation.prepareAbortContinuation(prepared.Slice()[0], prepared.Slice()[1])
		}()
	}
	panic(&nativeTransactionAbortTransfer{transactionID: state.ID})
}

// Caller owns the actual state.finalizationMu and the drained outer operation.
// Reuse the SAME resource/decision/undo owners; do not call public Rollback
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
	valid = state.checkedMutation == operation && state.mutationOperations == 1 && (state.Mode == TxCursorStability || state.Mode == TxACID) && !state.decisionStarted
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
		if len(a) < 1 || len(a) > 2 {
			panic("request_transaction_abort requires completion and optional preparation callback")
		}
		prepare := scm.NewNil()
		if len(a) > 1 {
			prepare = a[1]
		}
		scmerToTxContext(a[0]).requestMutationAbort(prepare)
		return scm.NewNil()
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
		Description: "requests terminal whole-transaction rollback before optional once-only preparation returns a callback and inert payload; native drain and Undo precede that callback's independent statement decisions",
		Params:      []*scm.TypeDescriptor{{Kind: "any", Label: "tx"}, {Kind: "func", Label: "prepare", Optional: true, CallsOnce: true, Params: []*scm.TypeDescriptor{}, Return: &scm.TypeDescriptor{Kind: "list"}}}, Return: &scm.TypeDescriptor{Kind: "any"}}})
	scm.Declare(en, &scm.Declaration{Name: "transaction_abort_failure", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) != 1 {
			panic("transaction_abort_failure requires one failure")
		}
		value := a[0].Any()
		failure, ok := value.(error)
		var transfer *nativeTransactionAbortTransfer
		if ok && errors.As(failure, &transfer) && transfer.continuationFailure != nil {
			if data, ok := transfer.continuationFailure.(scm.Scmer); ok {
				return data
			}
			return scm.NewAny(transfer.continuationFailure)
		}
		return scm.NewNil()
	}, Type: &scm.TypeDescriptor{Kind: "func", Const: true, Description: "projects an actual post-abort continuation failure without erasing terminal disposition or granting execution authority", Params: []*scm.TypeDescriptor{{Kind: "any", Label: "failure"}}, Return: &scm.TypeDescriptor{Kind: "any"}}})
	scm.Declare(en, &scm.Declaration{Name: "transaction_abort_transfer?", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) != 1 {
			panic(fmt.Errorf("transaction_abort_transfer? requires one failure value"))
		}
		return scm.NewBool(isNativeTransactionAbortTransfer(a[0]))
	}, Type: &scm.TypeDescriptor{Kind: "func", Const: true,
		Description: "classifies a native terminal transaction-abort transfer without granting transaction authority",
		Params:      []*scm.TypeDescriptor{{Kind: "any", Label: "failure"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
}
