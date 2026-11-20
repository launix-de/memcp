/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "sync"
import "errors"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"

// A logical table witness survives physical shard rebuilds. Admission revisions
// include uncommitted foreign writers: this intentionally conservative first
// implementation may abort instead of waiting, but cannot accept a changed read.
type serializableReadWitness struct {
	table    *table
	database *database
	name     string
	revision atomic.Uint64
}

type serializableValidation struct {
	transaction *transactionState
	mu          sync.Mutex // directory publication/decision only, never a row loop
	databases   sync.Map   // per-database local-mode pins are existing transaction resources
	reads       sync.Map   // native Go hash-trie: immutable read nodes, first-table publication only
	failed      atomic.Bool
	frozen      bool
	closed      bool
	closeOnce   sync.Once
	reserved    []*table
}

func (validation *serializableValidation) witness(owner *tableWriteAdmission) *serializableReadWitness {
	if value, found := validation.reads.Load(owner); found {
		return value.(*serializableReadWitness)
	}
	return nil
}

type serializationValidationFailure struct{}

func (*serializationValidationFailure) Error() string {
	return "serializable read changed; retry transaction"
}

func (*serializationValidationFailure) RequiresTransactionRollback() bool { return true }

func serializationConflict() error { return &serializationValidationFailure{} }

func isSerializationFailure(cause any) bool {
	failure, ok := nativeFailureOriginalValue(cause).(error)
	var serialization *serializationValidationFailure
	return ok && errors.As(failure, &serialization)
}

func (validation *serializableValidation) conflict() error {
	validation.failed.Store(true)
	return serializationConflict()
}

func (witness *serializableReadWitness) current() bool {
	return GetDatabase(witness.database.Name) == witness.database &&
		witness.database.tables.Get(witness.name) == witness.table &&
		witness.table.writeAdmission.revision.Load() == witness.revision.Load()
}

func (tx *TxContext) validateSerializableReads() error {
	if tx == nil || tx.transactionState == nil {
		return nil
	}
	validation := tx.serializable.Load()
	if validation == nil {
		return nil
	}
	if validation.failed.Load() {
		return serializationConflict()
	}
	return nil
}

// Local validation cannot survive an unannounced switch to a distributed
// writer path. Reuse the existing database mode-publication lease, once per
// observed/written database. Subsequent operators/streaming writes only Load.
func (tx *TxContext) requireSerializableLocal(t *table) {
	if tx == nil || tx.transactionState == nil {
		return
	}
	validation := tx.serializable.Load()
	if validation == nil || t.callerExecutionRelation() || t.dataOwner != nil {
		return
	}
	if _, pinned := validation.databases.Load(t.schema); pinned {
		return
	}
	tx.getOrComputeResource(fmt.Sprintf("native-serializable-database/%p", t.schema), scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		lease := t.schema.admitLocalCatalog(false)
		validation.databases.Store(t.schema, true)
		return scm.NewAny(lease)
	}))
}

// This is a logical operator boundary, before empty/range/lookup fast paths.
// Native invocation-only private relations retain their independent owners.
func (t *table) recordSerializableRead(tx *TxContext) {
	// Native transition results are immutable input of the enclosing checked
	// statement, not stored relations observed until transaction completion.
	// Their owner retires them after trigger readers drain. The scan still
	// enforces its existing transition read capability and lifetime.
	if tx == nil || tx.transactionState == nil || t.transitionRelation != nil || t.callerExecutionRelation() || t.dataOwner != nil {
		return
	}
	validation := tx.serializable.Load()
	if validation == nil {
		return
	}
	// Unpublished declarations keep their attested creator/catalog decision,
	// including reads and keys before commit; no foreign stored-data witness.
	if tx.ownsPendingTable(t) || tx.ownsScopedTable(t) {
		return
	}
	tx.requireSerializableLocal(t)
	if err := tx.validateSerializableReads(); err != nil {
		panic(err)
	}
	if witness := validation.witness(&t.writeAdmission); witness != nil {
		if !witness.current() {
			panic(validation.conflict())
		}
		return
	}
	// Namespace before table admission is the same order as semantic DDL.
	t.schema.schemalock.RLock()
	defer t.schema.schemalock.RUnlock()
	owner := &t.writeAdmission
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.foreignWriterLocked(tx.transactionState) {
		panic(validation.conflict())
	}
	validation.mu.Lock()
	defer validation.mu.Unlock()
	if validation.closed || validation.frozen {
		panic(validation.conflict())
	}
	if t.schema.tables.Get(t.Name) != t {
		// Unpublished native declarations have a different catalog decision;
		// they must not be mistaken for a committed namespace incarnation.
		panic("serializable reads require a published native table")
	}
	if validation.witness(owner) != nil {
		return
	}
	witness := &serializableReadWitness{table: t, database: t.schema, name: t.Name}
	witness.revision.Store(owner.revision.Load())
	if owner.serializableObservers == nil {
		owner.serializableObservers = make(map[*transactionState]*serializableValidation)
	}
	owner.serializableObservers[tx.transactionState] = validation
	owner.observersHighWater = max(owner.observersHighWater, len(owner.serializableObservers))
	validation.reads.Store(owner, witness)
}

func (owner *tableWriteAdmission) foreignWriterLocked(transaction *transactionState) bool {
	if owner.declaration != nil && owner.declaration != transaction ||
		owner.statement != nil && owner.statement.transaction != transaction {
		return true
	}
	for writer := range owner.writers {
		if writer != transaction {
			return true
		}
	}
	return false
}

// Called under the EXISTING table admission mutex. Every retained prepared
// reader already validated this same revision under this same owner, so a new
// writer cannot pass between validation and durable row/schema publication.
func (owner *tableWriteAdmission) advanceSerializableWriteLocked(transaction *transactionState) error {
	for reader := range owner.serializableReaders {
		if reader != transaction {
			if transaction != nil {
				if validation := transaction.serializable.Load(); validation != nil {
					return validation.conflict()
				}
			}
			return serializationConflict()
		}
	}
	var validation *serializableValidation
	var witness *serializableReadWitness
	if transaction != nil {
		validation = transaction.serializable.Load()
		if validation != nil {
			if validation.failed.Load() {
				return serializationConflict()
			}
			witness = validation.witness(owner)
			if witness != nil && witness.revision.Load() != owner.revision.Load() {
				return validation.conflict()
			}
		}
	}
	// Existing observations are invalidated before this writer is admitted.
	// Later reads consult one atomic poison bit, never traverse the read set.
	for observer, observed := range owner.serializableObservers {
		if observer != transaction {
			observed.failed.Store(true)
		}
	}
	revision := owner.revision.Add(1)
	if witness != nil {
		witness.revision.Store(revision)
	}
	return nil
}

// The same admitted writer updates its own observation only if no foreign
// admission intervened. Releasing its statement lease is still logical writer
// completion, not a new independent transaction or a witness reset.
func (owner *tableWriteAdmission) finishSerializableWriteLocked(transaction *transactionState) {
	before := owner.revision.Load()
	after := owner.revision.Add(1)
	if transaction == nil {
		return
	}
	validation := transaction.serializable.Load()
	if validation == nil {
		return
	}
	if witness := validation.witness(owner); witness != nil {
		if witness.revision.Load() == before {
			witness.revision.Store(after)
		} else {
			validation.failed.Store(true)
		}
	}
}

// Reservations, unlike read observations, are retained only from prepare to
// native decision completion. Fail-fast acquisition cannot deadlock on two
// read/write sets and needs no global lock or per-row synchronization.
func (tx *TxContext) prepareSerializable() error {
	validation := tx.serializable.Load()
	if validation == nil {
		return nil
	}
	validation.mu.Lock()
	if validation.frozen {
		validation.mu.Unlock()
		return tx.validateSerializableReads()
	}
	validation.frozen = true
	validation.mu.Unlock()
	if err := tx.validateSerializableReads(); err != nil {
		return err
	}
	var failure error
	validation.reads.Range(func(key, value any) bool {
		owner := key.(*tableWriteAdmission)
		witness := value.(*serializableReadWitness)
		owner.mu.Lock()
		if owner.foreignWriterLocked(tx.transactionState) || !witness.current() {
			owner.mu.Unlock()
			failure = validation.conflict()
			return false
		}
		if owner.serializableReaders == nil {
			owner.serializableReaders = make(map[*transactionState]struct{})
		}
		owner.serializableReaders[tx.transactionState] = struct{}{}
		owner.readersHighWater = max(owner.readersHighWater, len(owner.serializableReaders))
		owner.mu.Unlock()
		validation.reserved = append(validation.reserved, witness.table)
		return true
	})
	if failure != nil {
		validation.releaseReservations()
		return failure
	}
	return nil
}

func (validation *serializableValidation) releaseReservations() {
	for _, table := range validation.reserved {
		owner := &table.writeAdmission
		owner.mu.Lock()
		delete(owner.serializableReaders, validation.transaction)
		if len(owner.serializableReaders) == 0 {
			owner.serializableReaders = nil
			owner.readersHighWater = 0
		}
		owner.mu.Unlock()
	}
	validation.reserved = nil
}

func (validation *serializableValidation) Close() error {
	validation.closeOnce.Do(func() {
		// First-reader registration takes owner.mu before validation.mu. Mark
		// closed here, then release validation.mu before visiting table owners.
		// Native decision completion has already drained its producers.
		validation.mu.Lock()
		validation.closed = true
		validation.mu.Unlock()
		validation.releaseReservations()
		validation.reads.Range(func(key, _ any) bool {
			owner := key.(*tableWriteAdmission)
			owner.mu.Lock()
			delete(owner.serializableObservers, validation.transaction)
			if len(owner.serializableObservers) == 0 {
				owner.serializableObservers = nil
				owner.observersHighWater = 0
			}
			owner.mu.Unlock()
			return true
		})
		validation.reads.Clear()
		validation.databases.Clear()
		validation.transaction = nil
	})
	return nil
}

func initSerializableValidation(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "transaction_serialization_failure?", Fn: func(args ...scm.Scmer) scm.Scmer {
		return scm.NewBool(isSerializationFailure(args[0]))
	}, Type: &scm.TypeDescriptor{Kind: "func", Const: true,
		Description: "Recognizes an actual native serialization validation failure through original-failure projection; displayed text grants no authority.",
		Params:      []*scm.TypeDescriptor{{Kind: "any", Label: "failure"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "tx_serializable", Fn: func(args ...scm.Scmer) scm.Scmer {
		tx := scmerToTxContext(args[0])
		if tx == nil || tx.transactionState == nil {
			panic("serializable validation requires an active ACID transaction")
		}
		if err := tx.requireCommittable(); err != nil {
			panic(err)
		}
		tx.getOrComputeResource("native-serializable-validation", scm.NewFunc(func(...scm.Scmer) scm.Scmer {
			tx.mu.Lock()
			defer tx.mu.Unlock()
			if tx.State != TxActive || tx.Mode != TxACID || tx.decisionStarted || tx.prepared != nil {
				panic("serializable validation requires an active ACID transaction")
			}
			validation := &serializableValidation{transaction: tx.transactionState}
			tx.serializable.Store(validation)
			return scm.NewAny(validation)
		}))
		if err := tx.validateSerializableReads(); err != nil {
			panic(err)
		}
		return scm.NewBool(true)
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
		Description: "Enables optimistic serializable validation for subsequent local native reads, retaining original ACID journals and decision. Logical table changes abort with serialization failure; prepared reads exclude foreign publication until the native decision completes.",
		Params:      []*scm.TypeDescriptor{{Kind: "any", Label: "transaction"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
}
