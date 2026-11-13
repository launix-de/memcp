/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "context"

// This immutable declaration snapshot belongs to the SAME schema journal as
// scoped CREATE/DROP. It contains native metadata only, never frontend syntax,
// rows, callbacks, sessions or another transaction decision.
type relationSchemaKeyDeclaration struct {
	table *table
	keys  []uniqueKey
}

func (tx *TxContext) createScopedRelationKey(t *table, key uniqueKey) bool {
	if tx == nil || tx.State != TxActive || tx.decisionStarted || !tx.ownsScopedTable(t) || t.committedAt(t.schema.catalogView().version) {
		panic("scoped key declaration requires its active constructor transaction")
	}
	scope := t.relationLifetime
	if scope == nil || !scope.transactionalSchema || tx.coordinator != nil {
		panic("scoped key declaration requires its native relation lifetime owner")
	}
	finish := tx.beginCatalogConstruction(false)
	defer finish()
	releaseScope := scope.beginUse()
	defer releaseScope()
	db := GetDatabase(t.schema.Name)
	if db == nil || db != t.schema {
		panic("scoped key declaration database was replaced")
	}
	access := db.GetWrite(tx)
	defer access.Release()
	current := db.tables.Get(t.Name)
	if current == nil || current.RuntimeIdentity != t.RuntimeIdentity || !tx.ownsScopedTable(current) {
		panic("scoped key declaration lost its native generation")
	}
	// A short exclusive declaration lease closes the race with native writers.
	// Existing row work is not upgraded or replayed: this finite constructor
	// profile ends before the first INSERT, and the lease is released before it.
	lease, err := current.writeAdmission.admit(scanQueryContext(tx, context.Background()), tx.transactionState, true, false)
	if err != nil {
		panic(err)
	}
	defer lease.Close()
	if current.CountExact() != 0 {
		panic("scoped key declaration requires an empty constructor-owned generation")
	}
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	current.ddlMu.Lock()
	defer current.ddlMu.Unlock()
	previous := &relationSchemaKeyDeclaration{table: current, keys: append([]uniqueKey(nil), current.Unique...)}
	if !declareUnpublishedKey(current, key) {
		return false
	}
	change := &relationSchemaChange{scope: scope, declaration: previous}
	scope.mu.Lock()
	scope.schemaDecisions++
	scope.mu.Unlock()
	tx.mu.Lock()
	tx.relationSchemaChanges = append(tx.relationSchemaChanges, change)
	tx.mu.Unlock()
	current.publishShowColumnsSnapshot(true)
	return true
}

// Savepoint row Undo has finished before this immutable metadata restoration.
// Its writer exclusions may still be held, so no new maintenance lease or row
// scan is acquired here. Physical generation retirement remains at Tx cleanup.
func (tx *transactionState) restoreScopedRelationKeys(previous *relationSchemaKeyDeclaration) {
	db := GetDatabase(previous.table.schema.Name)
	if db == nil || db != previous.table.schema {
		return
	}
	access := db.GetWrite(nil)
	defer access.Release()
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	current := db.tables.Get(previous.table.Name)
	if current == nil || current.RuntimeIdentity != previous.table.RuntimeIdentity || !tx.ownsScopedTable(current) {
		return
	}
	current.ddlMu.Lock()
	defer current.ddlMu.Unlock()
	current.Unique = append([]uniqueKey(nil), previous.keys...)
	current.publishShowColumnsSnapshot(true)
}
