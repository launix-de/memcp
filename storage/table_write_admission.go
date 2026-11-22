/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "sync"
import "context"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"

// The table owns its declaration/write boundary. A writer is retained by the
// native transaction resource scope, not by an individual row or callback.
// Ordinary readers never register. Serializable observations and prepared
// read-publication reservations join this existing table-local owner.
type tableWriteAdmission struct {
	mu                      sync.Mutex
	revision                atomic.Uint64
	serializableReaders     map[*transactionState]struct{}
	serializableObservers   map[*transactionState]*serializableValidation
	readersHighWater        int
	observersHighWater      int
	writers                 map[*transactionState]tableWriterCapability
	declaration             *transactionState
	declarationGuardedInput bool
	statement               *statementTableWriteLease
	// Exact pending program names belong to the exclusive declaration lease.
	// Unrelated system cache programs remain free to publish during validation.
	triggerNames          map[string]struct{}
	writersHighWater      int
	triggerNamesHighWater int
	retainedMemory        *publicCatalogMemory
	changed               chan struct{}
}

// This is the existing retained writer directory entry, not a row counter.
// Raw callback authority can outlive its producer; only framed producers attest
// callback retirement through the existing native mutation drain owner.
type tableWriterCapability struct {
	leases            int
	unframedCallbacks bool
}
type tableAdmissionKind uint8

const (
	tableRowWriteAdmission tableAdmissionKind = iota
	tableUnframedCallbackAdmission
	tableCatalogDeclarationAdmission
	tableValidatedDeclarationAdmission
)

type tableWriteLease struct {
	owner       *tableWriteAdmission
	transaction *transactionState
	declaration bool
	once        sync.Once
	// The existing declaration lease may retain schema stability for private
	// initialized slots until this same original transaction releases it.
	ddlOnce  sync.Once
	ddlTable atomic.Pointer[table]
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

// Caller owns operation.mu. A default-only declaration may lend its already
// exclusive endpoint to this same transaction's statement. Its immutable native
// writer summary excludes pending FK/CHECK/trigger enforcement declarations.
func (t *table) admitStatement(operation *nativeMutationOperation) (*statementTableWriteLease, error) {
	if operation == nil || operation.state == nil || operation.invocation == nil ||
		operation.invocation.transactionState != operation.state {
		return nil, fmt.Errorf("statement writer requires native operation ownership")
	}
	transaction := operation.state
	owner := &t.writeAdmission
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.statement != nil {
		if owner.statement.operation == operation && owner.statement.transaction == transaction {
			return owner.statement, nil
		}
		return nil, fmt.Errorf("native statement conflicts with an active table writer; retry transaction")
	}
	if owner.declaration != nil && (owner.declaration != transaction || !operation.invocation.publicColumnMutationAllowed(t) && operation.invocation.privateMutationSource(t) == nil) {
		return nil, fmt.Errorf("native statement conflicts with a pending declaration; retry transaction")
	}
	for writer := range owner.writers {
		if writer != transaction {
			return nil, fmt.Errorf("native statement conflicts with an active table writer; retry transaction")
		}
	}
	if err := owner.advanceSerializableWriteLocked(transaction); err != nil {
		return nil, err
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
		owner.finishSerializableWriteLocked(lease.transaction)
		owner.statement = nil
		lease.owner, lease.operation, lease.transaction = nil, nil, nil
		if owner.changed != nil {
			close(owner.changed)
			owner.changed = nil
		}
	})
	return nil
}

// All callers release amendment/schema/shard mutexes before entering Do.
// Concurrent constructors of one original T reuse this exact retained owner.
func (lease *tableWriteLease) retainDDL(t *table, transaction *transactionState) {
	if lease == nil || !lease.declaration {
		panic("private initialized column requires its declaration lease")
	}
	owner := &t.writeAdmission
	lease.ddlOnce.Do(func() {
		t.ddlMu.Lock()
		owner.mu.Lock()
		valid := lease.owner == owner && lease.transaction == transaction && owner.declaration == transaction
		if valid {
			lease.ddlTable.Store(t)
		}
		owner.mu.Unlock()
		if !valid {
			t.ddlMu.Unlock()
			panic("private initialized column lost its declaration lease while waiting")
		}
	})
	if lease.ddlTable.Load() != t {
		panic("private column DDL lease changed endpoint")
	}
}

func (lease *tableWriteLease) Close() error {
	lease.once.Do(func() {
		owner := lease.owner
		owner.mu.Lock()
		defer owner.mu.Unlock()
		owner.finishSerializableWriteLocked(lease.transaction)
		if t := lease.ddlTable.Swap(nil); t != nil {
			t.ddlMu.Unlock()
		}
		if lease.declaration {
			if owner.declaration != lease.transaction {
				panic("native declaration admission lost its owner")
			}
			owner.declaration = nil
			owner.declarationGuardedInput = false
			owner.triggerNames = nil
			owner.triggerNamesHighWater = 0
			owner.retainedMemory = nil
		} else {
			capability := owner.writers[lease.transaction]
			if capability.leases <= 0 {
				panic("native write admission released without ownership")
			}
			capability.leases--
			owner.writers[lease.transaction] = capability
			if capability.leases == 0 {
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
	return publicCatalogMapFootprint(owner.readersHighWater) + publicCatalogMapFootprint(owner.observersHighWater) + publicCatalogMapFootprint(owner.writersHighWater) + publicCatalogMapFootprint(owner.triggerNamesHighWater) + owner.retainedMemory.bytes()
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
	t.requireCatalogDataVisibility(nil)
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

func (owner *tableWriteAdmission) admit(ctx context.Context, transaction *transactionState, kind tableAdmissionKind, wait bool) (*tableWriteLease, error) {
	if kind > tableValidatedDeclarationAdmission {
		panic("invalid native table admission profile")
	}
	declaration := kind == tableCatalogDeclarationAdmission || kind == tableValidatedDeclarationAdmission
	if kind == tableUnframedCallbackAdmission && transaction == nil {
		return nil, fmt.Errorf("retained mutation callbacks require their original transaction")
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		owner.mu.Lock()
		if declaration {
			// No blocking upgrade: two existing writers must never wait for
			// each other while retaining their native transaction leases.
			ownDefaultWriter := false
			if transaction != nil {
				if identity := transaction.catalogIdentity.Load(); identity != nil && (identity.latePublic || kind == tableValidatedDeclarationAdmission) {
					ownDefaultWriter = len(owner.writers) == 1 && owner.writers[transaction].leases != 0
				}
			}
			if kind == tableValidatedDeclarationAdmission && owner.writers[transaction].unframedCallbacks {
				owner.mu.Unlock()
				return nil, fmt.Errorf("native unique declaration conflicts with retained unframed mutation callbacks; finish the original transaction")
			}
			if owner.declaration != nil || owner.statement != nil || len(owner.writers) != 0 && !ownDefaultWriter {
				owner.mu.Unlock()
				return nil, fmt.Errorf("native declaration conflict: active table writer; retry transaction")
			}
			if err := owner.advanceSerializableWriteLocked(transaction); err != nil {
				owner.mu.Unlock()
				return nil, err
			}
			owner.declaration = transaction
			owner.declarationGuardedInput = kind == tableValidatedDeclarationAdmission
			owner.mu.Unlock()
			return &tableWriteLease{owner: owner, transaction: transaction, declaration: true}, nil
		}
		ownCallbackBorrow := kind == tableUnframedCallbackAdmission && owner.declaration == transaction && !owner.declarationGuardedInput
		if (owner.declaration == nil || ownCallbackBorrow) && (owner.statement == nil || owner.statement.transaction == transaction) {
			if err := owner.advanceSerializableWriteLocked(transaction); err != nil {
				owner.mu.Unlock()
				return nil, err
			}
			if owner.writers == nil {
				owner.writers = make(map[*transactionState]tableWriterCapability)
			}
			capability := owner.writers[transaction]
			capability.leases++
			if kind == tableUnframedCallbackAdmission {
				capability.unframedCallbacks = true
			}
			owner.writers[transaction] = capability
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
	tx.requireSerializableLocal(t)
	tx.noteStatementMutation(t)
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
		input := tx.mutationInputFor(t)
		if tx.publicColumnMutationAllowed(t) || input != nil && input.source != nil {
			// The admitted default amendment already retains this exact table's
			// exclusive declaration lease through the same native decision.
			return func() {}
		}
		key := fmt.Sprintf("native-table-write/%p", t)
		tx.getOrComputeResource(key, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
			lease, err := t.writeAdmission.admit(ctx, tx.transactionState, tableRowWriteAdmission, true)
			if err != nil {
				panic(err)
			}
			return scm.NewAny(lease)
		}))
		return func() {}
	}
	lease, err := t.writeAdmission.admit(ctx, nil, tableRowWriteAdmission, true)
	if err != nil {
		panic(err)
	}
	return func() { _ = lease.Close() }
}

// Admit mutation callback construction once per actual native operator/input.
// The same writer mutex orders this capability with UNIQUE declaration entry:
// either callbacks are retained first and upgrade refuses them, or declaration
// admission wins and no raw callback can be constructed. No row callback gains
// a new lock/counter; framed operations keep their existing drain proof.
func (t *table) admitNativeMutationCallbacks(tx *TxContext) (release func()) {
	release = t.admitNativeWrite(tx)
	ready := false
	defer func() {
		if !ready {
			release()
		}
	}()
	if tx != nil && tx.transactionState != nil && tx.mutationOperation == nil && t.requiresNativeAdmission() {
		// An own DEFAULT declaration normally lends its lease without another
		// resource. Raw callback authority instead retains THIS ordinary writer
		// resource through the original Tx, including an empty-phase savepoint.
		key := fmt.Sprintf("native-table-write/%p", t)
		tx.getOrComputeResource(key, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
			lease, err := t.writeAdmission.admit(scanQueryContext(tx, context.Background()), tx.transactionState, tableUnframedCallbackAdmission, false)
			if err != nil {
				panic(err)
			}
			return scm.NewAny(lease)
		}))
		owner := &t.writeAdmission
		func() {
			owner.mu.Lock()
			defer owner.mu.Unlock()
			if owner.declaration != nil && (owner.declaration != tx.transactionState || owner.declarationGuardedInput || !tx.publicColumnMutationAllowed(t)) {
				panic("native mutation callback construction conflicts with a declaration")
			}
			capability := owner.writers[tx.transactionState]
			if capability.leases <= 0 {
				panic("mutation callbacks lost their original native writer lease")
			}
			capability.unframedCallbacks = true
			owner.writers[tx.transactionState] = capability
		}()
	}
	ready = true
	return release
}

// Borrow an already-held native declaration lease for a validated immutable
// input change. An earlier raw callback capability cannot be retired by DDL;
// checked producers already supplied their existing completion/frame drain.
func (t *table) guardCatalogDeclarationInput(transaction *transactionState) (restore func()) {
	owner := &t.writeAdmission
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if transaction == nil || owner.declaration != transaction {
		panic("validated declaration requires its actual endpoint lease")
	}
	if owner.writers[transaction].unframedCallbacks {
		panic("native declaration conflicts with retained unframed mutation callbacks; finish the original transaction")
	}
	previous := owner.declarationGuardedInput
	owner.declarationGuardedInput = true
	return func() { t.setGuardedCatalogDeclarationInput(transaction, previous) }
}
func (t *table) setGuardedCatalogDeclarationInput(transaction *transactionState, guarded bool) {
	owner := &t.writeAdmission
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if transaction == nil || owner.declaration != transaction {
		panic("validated declaration lost its actual endpoint lease")
	}
	owner.declarationGuardedInput = guarded
}

// Semantic maintenance joins the same writer owner before changing a public
// declaration or retiring its payload. It does not wait while an exclusive
// catalog amendment retains ownership, and it never registers row readers or
// physical index/cache preparation as writers.
func (t *table) admitNativeMaintenance() func() {
	if t != nil {
		t.requireCatalogDataVisibility(nil)
	}
	if t != nil && t.mutationReadOnly {
		panic("native result relation is read-only")
	}
	if t == nil || !t.requiresNativeAdmission() {
		return func() {}
	}
	lease, err := t.writeAdmission.admit(context.Background(), nil, tableRowWriteAdmission, false)
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
			lease, err := t.writeAdmission.admit(context.Background(), nil, tableRowWriteAdmission, false)
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
