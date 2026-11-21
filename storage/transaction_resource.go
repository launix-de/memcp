/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "io"
import "fmt"
import "sync"
import "github.com/launix-de/memcp/scm"

type transactionResource struct {
	done    chan struct{}
	value   scm.Scmer
	failure any
}
type transactionResources struct {
	mu           sync.Mutex
	closeMu      sync.Mutex
	entries      map[string]*transactionResource
	deciding     bool
	closed       bool
	constructing int
	// Monotone declaration/row phase, protected by mu and reset with this owner.
	publicRowsEntered bool
}

func (tx *transactionState) resourceScope() *transactionResources { return &tx.resources }

// Accepted decisions retain their native owner through catalog publication,
// independently of whether the row phase already reached TxCommitted.
func (tx *transactionState) decisionPending() bool {
	return tx != nil && ((tx.State == TxActive && tx.decisionStarted) || (tx.State == TxCommitted && tx.catalogCommit != nil))
}

// A cleanup failure never changes the durable native decision or makes the
// completed transaction rollbackable again.
type transactionResourceCleanupError struct {
	state TxState
	cause any
}

func (err *transactionResourceCleanupError) Error() string {
	outcome := "finished"
	if err.state == TxCommitted {
		outcome = "committed"
	} else if err.state == TxAborted {
		outcome = "rolled back"
	}
	return fmt.Sprintf("transaction %s; resource cleanup failed: %v", outcome, err.cause)
}
func (tx *TxContext) finishResourceCommit() (err error) {
	// Successful commit ends the session transaction even if cleanup reports an
	// error. Automatic rollback instead leaves an explicit failed owner parked
	// until the caller acknowledges recovery through tx_rollback.
	if tx.State == TxCommitted && !tx.decisionPending() && !tx.Session.IsNil() {
		scm.Apply(tx.Session, scm.NewString("transaction"), scm.NewNil())
	}
	// Independent cleanup must still run after one closer fails. A pending
	// decision retains both its resources and any requested session close.
	attempt := func(close func()) {
		defer func() {
			if failure := recover(); failure != nil && err == nil {
				err = &transactionResourceCleanupError{tx.State, failure}
			}
		}()
		close()
	}
	attempt(tx.finishResourceDecision)
	attempt(tx.finishRequestedSessionClose)
	return err
}
func closeTransactionResource(value scm.Scmer) error {
	var resource any = value.Any()
	if object, ok := scm.ClusterObjectOf(value); ok {
		resource = object
	}
	if closer, ok := resource.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
func (tx *TxContext) getOrComputeResource(key string, producer scm.Scmer) scm.Scmer {
	if tx == nil || tx.transactionState == nil {
		panic("transaction resource requires an active native transaction")
	}
	state := tx.transactionState
	scope := state.resourceScope()
	scope.mu.Lock()
	if scope.closed {
		scope.mu.Unlock()
		panic("transaction resource scope is closed")
	}
	if entry := scope.entries[key]; entry != nil {
		scope.mu.Unlock()
		ctx := cacheMapContext(scm.NewAny(tx))
		select {
		case <-entry.done:
			if entry.failure != nil {
				panic(entry.failure)
			}
			return entry.value
		case <-ctx.Done():
			panic(ctx.Err())
		}
	}
	if scope.deciding {
		scope.mu.Unlock()
		panic("transaction decision does not accept new resources")
	}
	if scope.entries == nil {
		scope.entries = make(map[string]*transactionResource)
	}
	entry := &transactionResource{done: make(chan struct{})}
	scope.entries[key] = entry
	scope.constructing++
	scope.mu.Unlock()
	func() { defer func() { entry.failure = recover() }(); entry.value = scm.Apply(producer) }()
	scope.mu.Lock()
	scope.constructing--
	if entry.failure != nil {
		delete(scope.entries, key)
	}
	close(entry.done)
	scope.mu.Unlock()
	if entry.failure != nil {
		panic(entry.failure)
	}
	return entry.value
}

// A decision must not finish from inside its own resource constructor, or race
// an unfinished constructor. Factories run outside all native decision locks.
func (tx *transactionState) beginResourceDecision() {
	tx.beginMutationResourceDecision(nil)
}

// The only nonterminal exception is the SAME checked statement's private
// savepoint setup/restore. Public controls never borrow this permission.
func (tx *transactionState) beginMutationResourceDecision(operation *nativeMutationOperation) {
	scope := tx.resourceScope()
	scope.mu.Lock()
	defer scope.mu.Unlock()
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.relationDataOrigins != 0 && (operation == nil || tx.checkedMutation != operation || tx.mutationOperations != 1) {
		panic(newNativeMutationBusyError(tx, "source data decision"))
	}
	if tx.mutationOperations != 0 && (operation == nil || tx.checkedMutation != operation || tx.mutationOperations != 1) {
		panic(newNativeMutationBusyError(tx, "transaction decision"))
	}
	if scope.constructing != 0 {
		panic("transaction resource initialization is still active")
	}
	scope.deciding = true
}
func (tx *transactionState) finishResourceDecision() {
	// The durable row decision does not retire a still-owned catalog publication.
	// Keep declaration admission/resources and the session available for COMMIT retry.
	if tx.decisionPending() {
		return
	}
	scope := tx.resourceScope()
	if tx.State == TxActive {
		scope.mu.Lock()
		tx.mu.Lock()
		if !tx.decisionStarted && tx.mutationDecisionAttempts == 0 {
			scope.deciding = false
		}
		tx.mu.Unlock()
		scope.mu.Unlock()
		return
	}
	tx.closeResources()
}
func (tx *transactionState) closeResources() {
	if tx.decisionPending() {
		panic("native transaction decision is pending; retry COMMIT")
	}
	scope := tx.resourceScope()
	scope.closeMu.Lock()
	defer scope.closeMu.Unlock()
	scope.mu.Lock()
	tx.mu.Lock()
	mutating := tx.mutationOperations != 0
	schemaChanges := len(tx.relationSchemaChanges) != 0 || len(tx.relationSchemaRetirements) != 0
	tx.mu.Unlock()
	if mutating {
		scope.mu.Unlock()
		panic(newNativeMutationBusyError(tx, "resource retirement"))
	}
	if scope.constructing != 0 {
		scope.mu.Unlock()
		panic("transaction resource initialization is still active")
	}
	if schemaChanges {
		scope.deciding = true
		scope.mu.Unlock()
		// Every cleanup/reset entrypoint uses this SAME schema-decision owner before
		// releasing its resources. Publication never runs while mutation leases live.
		if tx.State == TxCommitted {
			tx.relationSchemaDecision.Store(relationSchemaCommitted)
		} else {
			tx.relationSchemaDecision.Store(relationSchemaAborted)
		}
		tx.finishRelationSchema(0, tx.State == TxCommitted)
		scope.mu.Lock()
	}
	if len(scope.entries) == 0 {
		scope.closed = true
		scope.deciding = true
		scope.mu.Unlock()
		return
	}
	scope.closed = true
	scope.deciding = true
	entries := scope.entries
	scope.entries = nil
	scope.mu.Unlock()
	var first error
	failed := make(map[string]*transactionResource)
	for key, entry := range entries {
		func() {
			defer func() {
				if failure := recover(); failure != nil {
					failed[key] = entry
					if first == nil {
						first = fmt.Errorf("resource cleanup: %v", failure)
					}
				}
			}()
			if err := closeTransactionResource(entry.value); err != nil {
				failed[key] = entry
				if first == nil {
					first = err
				}
			}
		}()
	}
	scope.mu.Lock()
	scope.entries = failed
	scope.mu.Unlock()

	if first != nil {
		panic(&transactionResourceCleanupError{tx.State, first})
	}
}
func initTransactionResources(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "tx_get_or_compute_resource", Fn: func(a ...scm.Scmer) scm.Scmer {
		return scmerToTxContext(a[0]).getOrComputeResource(a[1].String(), a[2])
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Description: "constructs and owns one native transaction-lifetime resource per key; completed transactions close io.Closer resources and release all values", Params: []*scm.TypeDescriptor{
		{Kind: "any", Label: "transaction"}, {Kind: "string", Label: "key"}, {Kind: "func", Label: "producer", CallsOnce: true, Params: []*scm.TypeDescriptor{}, Return: &scm.TypeDescriptor{Kind: "any"}}}, Return: &scm.TypeDescriptor{Kind: "any"}}})
}
