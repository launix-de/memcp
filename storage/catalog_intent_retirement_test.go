/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"testing"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

func cancelledCatalogInputFixture(t *testing.T) (*TxContext, *publicCatalogChanges, *table, *table, *table) {
	t.Helper()
	db := newDatabase()
	db.Name = "cancelled_catalog_input_" + t.Name()
	incarnation, transaction := "test-incarnation", "test-original-transaction"
	db.incarnation.Store(&incarnation)
	databases.Set(db.Name, db)
	t.Cleanup(func() { databases.Remove(db.Name) })
	tx := &TxContext{transactionState: &transactionState{State: TxActive}}
	tx.walTxID.Store(&transaction)
	tx.catalogIdentity.Store(&catalogIdentity{database: db, Database: db.Name, Incarnation: incarnation, Decision: "decision", Namespace: "namespace"})
	newTable := func(name string) *table {
		table := &table{schema: db, Name: name, RuntimeIdentity: name + "-identity"}
		table.showColumnsSnapshot.Store(&tableShowColumnsSnapshot{metadata: &tableShowColumnsSnapshotMetadata{
			columnNames: &tableColumnNamesSnapshot{}, constraints: emptyTableConstraints,
		}})
		db.tables.Set(name, table)
		return table
	}
	cancelled, surviving, dropped := newTable("cancelled_constructor"), newTable("public_target"), newTable("public_drop")
	cancelled.CreatorTxID, cancelled.CreatorIncarnation = transaction, incarnation
	cancelled.CatalogDecisionID, cancelled.CreatorNamespace = "decision", "namespace"
	cancelled.creator.Store(tx.transactionState)
	intents := []*table{cancelled}
	tx.catalogIntents.Store(&intents)
	changes := &publicCatalogChanges{owner: tx.transactionState, db: db, memory: &publicCatalogMemory{},
		leases: make(map[*table]*tableWriteLease), baselines: make(map[*table]publicForeignBaseline)}
	for _, table := range []*table{cancelled, surviving, dropped} {
		table.writeAdmission.declaration = tx.transactionState
		lease := &tableWriteLease{owner: &table.writeAdmission, transaction: tx.transactionState, declaration: true}
		if table != dropped {
			table.ddlMu.Lock()
			lease.ddlTable.Store(table)
		}
		changes.leases[table] = lease
		changes.baselines[table] = publicForeignBaseline{declarations: scm.NewString(table.Name)}
	}
	changes.anchor = changes.leases[cancelled]
	changes.anchor.attachRetainedMemory(changes.memory)
	history := []catalogDeclarationAmendment{{tableDrop: &catalogTableDropAmendment{table: dropped}}}
	tx.catalogAmendments.Store(&history)
	tx.publicCatalog.Store(changes)
	changes.views.Store(&publicCatalogViews{tables: map[*table]*tableShowColumnsSnapshot{
		cancelled: cancelled.showColumnsSnapshot.Load(), surviving: surviving.showColumnsSnapshot.Load(),
	}, allocators: map[*table]*catalogAllocatorReset{cancelled: nil, surviving: nil}})
	changes.refreshMemoryLocked()
	t.Cleanup(func() {
		for _, lease := range changes.leases {
			_ = lease.Close()
		}
	})
	return tx, changes, cancelled, surviving, dropped
}

func TestCancelledCatalogConstructorRetiresOnlyItsRetainedInputs(t *testing.T) {
	tx, changes, cancelled, surviving, dropped := cancelledCatalogInputFixture(t)
	lease, survivorLease, dropLease := changes.leases[cancelled], changes.leases[surviving], changes.leases[dropped]
	previous := changes.views.Load()
	validateCancelledCatalogIntents(tx, changes.db, []*table{cancelled}, tx.publicForeignAmendments())
	changes.retireCancelledCatalogInputs(tx, []*table{cancelled})
	views := changes.views.Load()
	_, baselineRetained := changes.baselines[cancelled]
	if changes.leases[cancelled] != nil || baselineRetained ||
		views.tables[cancelled] != nil || lease.owner != nil || lease.ddlTable.Load() != nil || cancelled.writeAdmission.declaration != nil {
		t.Fatal("cancelled constructor retained an endpoint lease, baseline or input")
	}
	if _, exists := views.allocators[cancelled]; exists {
		t.Fatal("cancelled constructor retained its headless allocator slot")
	}
	if previous.tables[cancelled] == nil || len(previous.allocators) != 2 {
		t.Fatal("retirement mutated a previously published immutable input")
	}
	if changes.leases[surviving] != survivorLease || changes.leases[dropped] != dropLease ||
		views.tables[surviving] != surviving.showColumnsSnapshot.Load() || views.checkedWriters[surviving] != nativeColumnBindingRowInput {
		t.Fatal("cancelling a constructor retired surviving public input leases")
	}
	if _, exists := views.allocators[surviving]; !exists {
		t.Fatal("surviving original owner lost its headless allocator baseline")
	}
	if _, exists := views.dropped[dropped]; !exists {
		t.Fatal("constructor cancellation lost the earlier public DROP prefix")
	}
	if changes.anchor == nil || changes.anchor == lease || changes.anchor.owner.retainedMemory != changes.memory ||
		changes.memory.snapshot.Load().bytes != 2*uint(unsafe.Sizeof(tableWriteLease{})) {
		t.Fatal("constructor cancellation lost its retained-memory owner")
	}
	if changes.db.tables.Get(cancelled.Name) != cancelled {
		t.Fatal("input retirement removed storage before the discard owner")
	}
	// The native discard entry can now remove this exact generation; later
	// publication must validate every surviving input without a missing-table skip.
	changes.db.tables.Remove(cancelled.Name)
	changes.publishViews(tx.publicForeignAmendments(), views.tables)
	_ = lease.Close() // the existing lease's retirement remains idempotent
	if changes.leases[surviving] != survivorLease || changes.leases[dropped] != dropLease {
		t.Fatal("repeated cleanup changed a surviving lease")
	}
}

func TestCancelledCatalogConstructorRefusesWrongGenerationOrSurvivingReferences(t *testing.T) {
	tx, changes, cancelled, surviving, _ := cancelledCatalogInputFixture(t)
	before, anchor := changes.views.Load(), changes.anchor
	assertUnchanged := func() {
		t.Helper()
		if changes.views.Load() != before || changes.anchor != anchor || changes.leases[cancelled] != anchor || anchor.owner == nil ||
			changes.leases[surviving] == nil || changes.db.tables.Get(cancelled.Name) != cancelled {
			t.Fatal("failed cancellation changed retained input or generation ownership")
		}
	}
	replacement := &table{Name: cancelled.Name, RuntimeIdentity: cancelled.RuntimeIdentity, schema: changes.db}
	changes.db.tables.Set(cancelled.Name, replacement)
	requirePanic(t, func() { validateCancelledCatalogIntents(tx, changes.db, []*table{cancelled}, nil) })
	changes.db.tables.Set(cancelled.Name, cancelled)
	assertUnchanged()
	cancelled.CatalogVersion.Store(1)
	requirePanic(t, func() { validateCancelledCatalogIntents(tx, changes.db, []*table{cancelled}, nil) })
	cancelled.CatalogVersion.Store(0)
	assertUnchanged()
	for _, entry := range []catalogDeclarationAmendment{
		{table: cancelled},
		{allocator: &catalogAllocatorReset{table: cancelled}},
		{tableRename: &catalogTableRenameAmendment{table: cancelled}},
		{column: &catalogStoredColumnAmendment{table: cancelled}},
		{publicKey: &catalogPublicKeyAmendment{table: cancelled}},
		{tableDrop: &catalogTableDropAmendment{table: cancelled}},
		{check: &catalogCheckAmendment{table: cancelled}},
		{checkDrop: &catalogCheckDropAmendment{table: cancelled}},
		{foreign: &catalogForeignAmendment{endpoints: [2]*table{surviving, cancelled}}},
		{foreignDrop: &catalogForeignDropAmendment{endpoints: [2]*table{cancelled, surviving}}},
		{constraintState: &catalogConstraintStateAmendment{views: []catalogConstraintStateView{{table: cancelled}}}},
		{statementTrigger: &catalogStatementTriggerAmendment{slot: &catalogStatementTriggerSlot{table: cancelled}}},
	} {
		requirePanic(t, func() {
			validateCancelledCatalogIntents(tx, changes.db, []*table{cancelled}, []catalogDeclarationAmendment{entry})
		})
		assertUnchanged()
	}
	original := anchor.transaction
	anchor.transaction = &transactionState{State: TxActive}
	requirePanic(t, func() { changes.retireCancelledCatalogInputs(tx, []*table{cancelled}) })
	anchor.transaction = original
	assertUnchanged()
}

func TestCancelledCatalogConstructorWithoutSurvivingEndpointReleasesAnchor(t *testing.T) {
	tx, changes, cancelled, surviving, dropped := cancelledCatalogInputFixture(t)
	for _, table := range []*table{surviving, dropped} {
		_ = changes.leases[table].Close()
		delete(changes.leases, table)
		delete(changes.baselines, table)
	}
	tx.catalogAmendments.Store(nil)
	changes.views.Store(&publicCatalogViews{tables: map[*table]*tableShowColumnsSnapshot{
		cancelled: cancelled.showColumnsSnapshot.Load(),
	}, allocators: map[*table]*catalogAllocatorReset{cancelled: nil}})
	validateCancelledCatalogIntents(tx, changes.db, []*table{cancelled}, nil)
	changes.retireCancelledCatalogInputs(tx, []*table{cancelled})
	if changes.anchor != nil || len(changes.leases) != 0 || len(changes.baselines) != 0 ||
		len(changes.views.Load().tables) != 0 || len(changes.views.Load().allocators) != 0 ||
		changes.memory.snapshot.Load().bytes != 0 || changes.dbMemoryBytes == 0 {
		t.Fatal("last constructor retirement retained endpoint state or lost the database resource charge")
	}
}
