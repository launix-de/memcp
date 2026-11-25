/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "errors"
import "context"
import "runtime"
import "github.com/launix-de/memcp/scm"

// Rollback-only is an admission property of the original active transaction,
// not a completed rollback. Its private rows and resource owners remain live.
type rollbackOnlyViolation struct{}

func (*rollbackOnlyViolation) Error() string {
	return "transaction is rollback-only; full ROLLBACK required"
}

func (tx *TxContext) requireCommittable() error {
	if tx != nil && tx.transactionState != nil && tx.State == TxActive && tx.rollbackOnly.Load() {
		return &rollbackOnlyViolation{}
	}
	return nil
}

// Existing query-owned scratch remains writable by read operators. Stored-data
// targets, including transactional scoped declarations, retain native admission.
func (tx *TxContext) noteStatementMutation(target *table) {
	if tx == nil || tx.transactionState == nil {
		return
	}
	if tx.failurePolicyExecuting.Load() {
		panic(newNativeMutationBusyError(tx.transactionState, "failure policy mutation"))
	}
	if target != nil && !target.ScopedSchema && target.callerExecutionRelation() {
		return
	}
	if err := tx.requireCommittable(); err != nil {
		panic(err)
	}
	if tx.failureTracking.Load() && !tx.statementMutationEntered.Load() {
		tx.statementMutationEntered.CompareAndSwap(false, true)
	}
}

// These failures retain their existing native recovery/decision authority.
// A language callback cannot turn a failed cleanup or unknown commit into a
// committable transaction by interpreting its displayed diagnostic.
func nativeFailurePolicyExcluded(cause any) bool {
	if value, ok := cause.(scm.Scmer); ok {
		return nativeFailurePolicyExcluded(value.Any())
	}
	if origin, ok := cause.(interface{ OriginalFailure() any }); ok && nativeFailurePolicyExcluded(origin.OriginalFailure()) {
		return true
	}
	failure, ok := cause.(error)
	if !ok {
		return false
	}
	var persistence *PersistenceFailure
	var cleanup *transactionResourceCleanupError
	var busy *nativeMutationBusyError
	var terminal *nativeTransactionAbortTransfer
	var fatal runtime.Error
	var mandatory interface{ RequiresTransactionRollback() bool }
	return (errors.As(failure, &mandatory) && mandatory.RequiresTransactionRollback()) || errors.Is(failure, context.Canceled) || errors.Is(failure, context.DeadlineExceeded) ||
		errors.As(failure, &persistence) || errors.As(failure, &cleanup) ||
		errors.As(failure, &busy) || errors.As(failure, &terminal) || errors.As(failure, &fatal) || commitOutcomeUnknown(failure)
}

// Request execution is sequential. Native producer admission uses the SAME
// state mutex, so status is published only after every writer has retired.
func (tx *TxContext) failureCompletion(transactionID, generation uint64, cause any) string {
	tx.mu.Lock()
	idle := tx.ID == transactionID && tx.State == TxActive && !tx.decisionPending() &&
		tx.mutationOperations == 0 && tx.mutationSavepoints == 0 && tx.relationDataOrigins == 0 && tx.checkedMutation == nil
	tx.mu.Unlock()
	if !idle {
		return "unrestored"
	}
	if failure, ok := cause.(error); ok {
		var restored *nativeStatementRestoredError
		if errors.As(failure, &restored) && restored.restoredBy(transactionID, generation) {
			return "restored"
		}
	}
	if !tx.statementMutationEntered.Load() {
		return "untouched"
	}
	return "unrestored"
}

func (tx *TxContext) chooseFailureDisposition(policy scm.Scmer, transactionID, generation uint64, cause any) (preserve, clearMarker bool) {
	if policy.IsNil() || tx.decisionPending() || nativeFailurePolicyExcluded(cause) {
		return false, false
	}
	// Reuse the actual native decision binding: callbacks cannot reset/commit
	// the executor or construct resources while interpreting this failure.
	attempt, err := tx.beginMutationBindingAttempt("statement failure policy")
	if err != nil {
		if attempt != nil {
			attempt.finish()
		}
		return false, false
	}
	err = attempt.runDecision(func(owner *TxContext) error {
		if owner == nil || owner.ID != transactionID || owner.decisionPending() {
			return nil
		}
		completion := owner.failureCompletion(transactionID, generation, cause)
		owner.failurePolicyExecuting.Store(true)
		var disposition scm.Scmer
		func() {
			defer owner.failurePolicyExecuting.Store(false)
			disposition = scm.Apply(policy, scm.FromAny(cause), scm.NewString(completion))
		}()
		if !disposition.IsString() {
			return fmt.Errorf("statement failure policy requires a disposition string")
		}
		switch disposition.String() {
		case "rollback":
			clearMarker = true
		case "preserve", "rollback-only":
			// The same pinned owner must still attest the completed statement.
			if completion == "unrestored" || owner.failureCompletion(transactionID, generation, cause) != completion {
				return nil
			}
			if disposition.String() == "rollback-only" {
				owner.rollbackOnly.Store(true)
			}
			preserve = true
		default:
			return fmt.Errorf("invalid statement failure disposition %q", disposition.String())
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return
}
