/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "sync"
import "context"
import "github.com/launix-de/memcp/scm"

// The table owns its declaration/write boundary. A writer is retained by the
// native transaction resource scope, not by an individual row or callback.
// Readers never register with this owner.
type tableWriteAdmission struct {
	mu          sync.Mutex
	writers     map[*transactionState]int
	declaration *transactionState
	statement   *statementTableWriteLease
	// Exact pending program names belong to the exclusive declaration lease.
	// Unrelated system cache programs remain free to publish during validation.
	triggerNames          map[string]struct{}
	writersHighWater      int
	triggerNamesHighWater int
	retainedMemory        *publicCatalogMemory
	changed               chan struct{}
}

type tableWriteLease struct {
	owner       *tableWriteAdmission
	transaction *transactionState
	declaration bool
	once        sync.Once
}

// A checked statement keeps its real table writer exclusion through producer
// drain AND savepoint undo. It is distinct from declaration-only admission and
// may upgrade this SAME transaction's earlier shared writer lease without
// waiting for another transaction while retaining those earlier rights.
type statementTableWriteLease struct {
	owner       *tableWriteAdmission
	operation   *nativeMutationOperation
	transaction *transactionState
	once        sync.Once
}

func (owner *tableWriteAdmission) admitStatement(operation *nativeMutationOperation, transaction *transactionState) (*statementTableWriteLease, error) {
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if operation == nil || transaction == nil {
		return nil, fmt.Errorf("statement writer requires native operation ownership")
	}
	if owner.statement != nil {
		if owner.statement.operation == operation && owner.statement.transaction == transaction {
			return owner.statement, nil
		}
		return nil, fmt.Errorf("native statement conflicts with an active table writer; retry transaction")
	}
	if owner.declaration != nil {
		return nil, fmt.Errorf("native statement conflicts with a pending declaration; retry transaction")
	}
	for writer := range owner.writers {
		if writer != transaction {
			return nil, fmt.Errorf("native statement conflicts with an active table writer; retry transaction")
		}
	}
	lease := &statementTableWriteLease{owner: owner, operation: operation, transaction: transaction}
	owner.statement = lease
	return lease, nil
}

func (lease *statementTableWriteLease) Close() error {
	lease.once.Do(func() {
		owner := lease.owner
		owner.mu.Lock()
		defer owner.mu.Unlock()
		if owner.statement != lease {
			panic("native statement writer lost its table admission")
		}
		owner.statement = nil
		lease.owner, lease.operation, lease.transaction = nil, nil, nil
		if owner.changed != nil {
			close(owner.changed)
			owner.changed = nil
		}
	})
	return nil
}

func (lease *tableWriteLease) Close() error {
	lease.once.Do(func() {
		owner := lease.owner
		owner.mu.Lock()
		defer owner.mu.Unlock()
		if lease.declaration {
			if owner.declaration != lease.transaction {
				panic("native declaration admission lost its owner")
			}
			owner.declaration = nil
			owner.triggerNames = nil
			owner.triggerNamesHighWater = 0
			owner.retainedMemory = nil
		} else {
			if owner.writers[lease.transaction] <= 0 {
				panic("native write admission released without ownership")
			}
			owner.writers[lease.transaction]--
			if owner.writers[lease.transaction] == 0 {
				delete(owner.writers, lease.transaction)
				if len(owner.writers) == 0 {
					owner.writers = nil
					owner.writersHighWater = 0
				}
			}
		}
		if owner.changed != nil {
			close(owner.changed)
			owner.changed = nil
		}
		lease.owner, lease.transaction = nil, nil
	})
	return nil
}

// Take table.mu before this owner's mutex. That same order protects
// trigger insertion, so checking a name and reserving it cannot race a publisher.
func (t *table) reserveDeclarationTrigger(transaction *transactionState, name string, replaceStatement bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	owner := &t.writeAdmission
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.declaration != transaction || transaction == nil {
		panic("native trigger reservation requires its declaration owner")
	}
	for _, trigger := range t.Triggers {
		if trigger.Name == name && (!replaceStatement || !trigger.Timing.isStatement() || trigger.IsSystem) {
			panic("foreign declaration trigger name already exists: " + name)
		}
	}
	if _, exists := owner.triggerNames[name]; exists {
		panic("foreign declaration trigger name already reserved: " + name)
	}
	if owner.triggerNames == nil {
		owner.triggerNames = make(map[string]struct{})
	}
	owner.triggerNames[name] = struct{}{}
	owner.triggerNamesHighWater = max(owner.triggerNamesHighWater, len(owner.triggerNames))
}

func (t *table) releaseDeclarationTrigger(transaction *transactionState, name string) {
	owner := &t.writeAdmission
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.declaration != transaction || transaction == nil {
		panic("native trigger reservation lost its declaration owner")
	}
	delete(owner.triggerNames, name)
	if len(owner.triggerNames) == 0 {
		owner.triggerNames = nil
		owner.triggerNamesHighWater = 0
	}
}

// Map keys borrow program strings and transaction handles. Only the FIRST
// retained declaration lease additionally exposes its resource's scalar bytes;
// there is no transaction/AST walk or changes.mu acquisition by a reader.
func (owner *tableWriteAdmission) metadataMemory() uint {
	owner.mu.Lock()
	defer owner.mu.Unlock()
	return publicCatalogMapFootprint(owner.writersHighWater) + publicCatalogMapFootprint(owner.triggerNamesHighWater) + owner.retainedMemory.bytes()
}

// Called once for the first admitted endpoint; savepoints keep this anchor
// even when all amendments are rewound. Close detaches it with that same lease.
func (lease *tableWriteLease) attachRetainedMemory(memory *publicCatalogMemory) {
	owner := lease.owner
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if !lease.declaration || owner.declaration != lease.transaction || owner.retainedMemory != nil {
		panic("native declaration memory lost its first endpoint lease")
	}
	owner.retainedMemory = memory
}

// Both the declaration reservation and actual live program must still match
// before the existing durable decision is prepared. Takes table.mu first.
func (t *table) validateDeclarationTrigger(transaction *transactionState, name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	owner := &t.writeAdmission
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.declaration != transaction || transaction == nil {
		panic("native trigger reservation lost its declaration owner")
	}
	if _, exists := owner.triggerNames[name]; !exists {
		panic("native declaration trigger reservation is missing: " + name)
	}
	for _, trigger := range t.Triggers {
		if trigger.Name == name {
			panic("foreign declaration trigger name already exists: " + name)
		}
	}
}

// Caller holds table.mu through the actual trigger-list mutation. Non-system
// definitions borrow the ordinary fail-fast semantic maintenance admission;
// system cache churn only conflicts with an explicitly reserved program name.
func (t *table) admitTriggerMutationLocked(name string, system bool) func() {
	if !system {
		t.requireTriggerDeclarationMutationLocked()
		return t.admitNativeMaintenance()
	}
	owner := &t.writeAdmission
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if _, reserved := owner.triggerNames[name]; reserved {
		panic("native declaration conflict: trigger name has a pending declaration: " + name)
	}
	for _, trigger := range t.Triggers {
		if trigger.Name == name && !trigger.IsSystem {
			panic("user and system trigger definitions cannot replace one another")
		}
	}
	return func() {}
}

func (owner *tableWriteAdmission) admit(ctx context.Context, transaction *transactionState, declaration, wait bool) (*tableWriteLease, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		owner.mu.Lock()
		if declaration {
			// No blocking upgrade: two existing writers must never wait for
			// each other while retaining their native transaction leases.
			if owner.declaration != nil || owner.statement != nil || len(owner.writers) != 0 {
				owner.mu.Unlock()
				return nil, fmt.Errorf("native declaration conflict: active table writer; retry transaction")
			}
			owner.declaration = transaction
			owner.mu.Unlock()
			return &tableWriteLease{owner: owner, transaction: transaction, declaration: true}, nil
		}
		if owner.declaration == nil && (owner.statement == nil || owner.statement.transaction == transaction) {
			if owner.writers == nil {
				owner.writers = make(map[*transactionState]int)
			}
			owner.writers[transaction]++
			owner.writersHighWater = max(owner.writersHighWater, len(owner.writers))
			owner.mu.Unlock()
			return &tableWriteLease{owner: owner, transaction: transaction}, nil
		}
		if transaction != nil && owner.declaration == transaction {
			owner.mu.Unlock()
			return nil, fmt.Errorf("native metadata-only transaction does not admit row mutation")
		}
		if !wait {
			statement := owner.statement != nil
			owner.mu.Unlock()
			if statement {
				return nil, fmt.Errorf("native statement conflict: table writer is still active; retry maintenance")
			}
			return nil, fmt.Errorf("native declaration conflict: table has a pending declaration; retry maintenance")
		}
		if owner.changed == nil {
			owner.changed = make(chan struct{})
		}
		changed := owner.changed
		owner.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// admitNativeWrite is called before defaults, triggers, identities, routing or
// native mutation callbacks. The nil-transaction API retains a statement lease;
// explicit/autocommit transactions retain one lease until their final decision.
func (t *table) admitNativeWrite(tx *TxContext) func() {
	if t.mutationReadOnly {
		panic("native result relation is read-only")
	}
	if tx != nil && tx.mutationOperation != nil {
		tx.requireMutationBinding()
		// Synchronous user triggers can reach another stored-data target. It
		// joins the SAME undo exclusion before any row/default/callback effect;
		// a conflicting or out-of-profile endpoint fails without waiting.
		tx.mutationOperation.admitStatementTable(t)
	}
	// Public transactional declarations remain disabled on distributed tables
	// until their single admission authority is carried by native schema transfer.
	if !t.requiresNativeAdmission() {
		return func() {}
	}
	ctx := scanQueryContext(tx, context.Background())
	if tx != nil {
		tx.admitCatalogMutation(t)
		tx.noteNativeRowAccess(t)
		if tx.publicDefaultMutationAllowed(t) {
			// The admitted default amendment already retains this exact table's
			// exclusive declaration lease through the same native decision.
			return func() {}
		}
		key := fmt.Sprintf("native-table-write/%p", t)
		tx.getOrComputeResource(key, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
			lease, err := t.writeAdmission.admit(ctx, tx.transactionState, false, true)
			if err != nil {
				panic(err)
			}
			return scm.NewAny(lease)
		}))
		return func() {}
	}
	lease, err := t.writeAdmission.admit(ctx, nil, false, true)
	if err != nil {
		panic(err)
	}
	return func() { _ = lease.Close() }
}

// Semantic maintenance joins the same writer owner before changing a public
// declaration or retiring its payload. It does not wait while an exclusive
// catalog amendment retains ownership, and it never registers row readers or
// physical index/cache preparation as writers.
func (t *table) admitNativeMaintenance() func() {
	if t != nil && t.mutationReadOnly {
		panic("native result relation is read-only")
	}
	if t == nil || !t.requiresNativeAdmission() {
		return func() {}
	}
	lease, err := t.writeAdmission.admit(context.Background(), nil, false, false)
	if err != nil {
		panic(err)
	}
	return func() { _ = lease.Close() }
}

// Runtime mode activation does not retire an existing local declaration lease.
// Its complete startup/schema-transfer admission is a separate owner boundary;
// a writer must never bypass the already accepted local owner in the meantime.
func (t *table) requiresNativeAdmission() bool {
	if activeReplication(t.schema) == nil {
		return true
	}
	owner := &t.writeAdmission
	owner.mu.Lock()
	defer owner.mu.Unlock()
	return owner.declaration != nil || owner.statement != nil
}

// Database retirement/backend publication freezes the table set through its
// existing persistence lifecycle owner before borrowing these operation leases.
// Admission is fail-fast; a failed batch releases every earlier table lease.
func (db *database) admitNativeMaintenance() (release func()) {
	operation := db.admitLocalCatalog(true)
	var releases []func()
	accepted := false
	defer func() {
		if !accepted {
			for i := len(releases) - 1; i >= 0; i-- {
				releases[i]()
			}
			_ = operation.Close()
		}
	}()
	func() {
		db.schemalock.Lock()
		defer db.schemalock.Unlock()
		for _, t := range db.tables.GetAll() {
			// A schema importer already has its actual coherence authority.
			// Consult the REAL local gate even if mode publication is starting
			// or active; optional adapter selection cannot bypass an old owner.
			lease, err := t.writeAdmission.admit(context.Background(), nil, false, false)
			if err != nil {
				panic(err)
			}
			releases = append(releases, func() { _ = lease.Close() })
		}
	}()
	accepted = true
	return func() {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
		_ = operation.Close()
	}
}
