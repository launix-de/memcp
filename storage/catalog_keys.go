/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "context"
import "github.com/launix-de/memcp/scm"

// Declaration amendments belong to the same private generation as CREATE.
// Immutable previous slices preserve interleaved column/key savepoint changes.
type catalogDeclarationAmendment struct {
	table            *table
	keys             []uniqueKey
	columns          []*column
	check            *catalogCheckAmendment
	foreign          *catalogForeignAmendment
	constraintState  *catalogConstraintStateAmendment
	source           *catalogSourceAmendment
	statementTrigger *catalogStatementTriggerAmendment
	columnDefault    *catalogColumnDefaultAmendment
}

func (tx *TxContext) catalogDeclarationPosition() int {
	if amendments := tx.catalogAmendments.Load(); amendments != nil {
		return len(*amendments)
	}
	return 0
}
func catalogKeyArguments(t *table, key uniqueKey) []scm.Scmer {
	columns := make([]scm.Scmer, len(key.Cols))
	for i, column := range key.Cols {
		columns[i] = scm.NewString(column)
	}
	return []scm.Scmer{scm.NewString(t.schema.Name), scm.NewString(t.Name), scm.NewString(t.RuntimeIdentity), scm.NewString(key.Id), scm.NewSlice(columns), scm.NewBool(key.NullsNotDistinct), scm.NewString(key.ConstraintName)}
}

func (tx *TxContext) createCatalogKey(t *table, key uniqueKey) bool {
	if tx != nil && tx.ownsScopedTable(t) {
		return tx.createScopedRelationKey(t, key)
	}
	if tx == nil || tx.State != TxActive || tx.decisionStarted || !tx.ownsPendingTable(t) {
		panic("unique declaration requires its active native catalog owner")
	}
	if tx.coordinator != nil {
		value, err := tx.coordinator.Call(scanQueryContext(tx, context.Background()), scm.ClusterRequest{Method: "catalog_key", Arguments: catalogKeyArguments(t, key)})
		if err != nil {
			panic(err)
		}
		return scm.ToBool(value)
	}
	return tx.amendCatalogDeclaration(t, func(current *table) bool { return declareUnpublishedKey(current, key) })
}

// Both constructor-owned durable and scoped MEMORY generations use the same
// native key declaration. Caller owns schema/DDL publication and has established
// that the unpublished generation has no rows or concurrent native writer.
func declareUnpublishedKey(current *table, key uniqueKey) bool {
	current.requireStoredUniqueKey(key)
	if key.Id == "" || len(key.Cols) == 0 {
		panic("unique declaration requires a name and columns")
	}
	seen := make(map[string]bool, len(key.Cols))
	for _, name := range key.Cols {
		if seen[name] {
			panic("duplicate unique declaration column: " + name)
		}
		seen[name] = true
		if _, exists := current.ResolveColumnName(name, false); !exists {
			panic("unknown unique declaration column: " + name)
		}
	}
	for _, existing := range current.Unique {
		if existing.matchesName(key.Id) || (key.ConstraintName != "" && existing.matchesName(key.ConstraintName)) {
			return false
		}
	}
	if (key.ConstraintName != "" && current.schema.hasDeclarationNameLocked(key.ConstraintName, nil)) || current.hasDeclarationName(key.Id, nil, true) {
		panic("declaration name already exists: " + key.Id)
	}
	key.Cols = append([]string(nil), key.Cols...)
	current.Unique = append(append([]uniqueKey(nil), current.Unique...), key)
	return true
}

// The private catalog owner admits immutable declaration changes before row
// access. The same history position covers keys and columns, including remote
// savepoints, without publishing an intermediate schema or checkpoint.
func (tx *TxContext) amendCatalogDeclaration(t *table, amend func(*table) bool) bool {
	if tx == nil || tx.State != TxActive || tx.decisionStarted || !tx.ownsPendingTable(t) {
		panic("declaration amendment requires its active native catalog owner")
	}
	finish := tx.beginCatalogConstruction(false)
	defer finish()
	if tx.rowAccessed.Load() {
		panic("transactional declaration amendments must precede native row access")
	}
	identity := tx.catalogIdentity.Load()
	db := GetDatabase(t.schema.Name)
	if db == nil || db != identity.database || db.incarnationID() != identity.Incarnation {
		panic("transaction catalog database was replaced")
	}
	access := db.GetWrite(tx)
	defer access.Release()
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	current := db.tables.Get(t.Name)
	if current == nil || current.RuntimeIdentity != t.RuntimeIdentity || !tx.ownsPendingTable(current) || GetDatabase(db.Name) != db || db.incarnationID() != identity.Incarnation {
		panic("declaration amendment lost its native catalog generation")
	}
	current.ddlMu.Lock()
	defer current.ddlMu.Unlock()
	if tx.rowAccessed.Load() {
		panic("transactional declaration amendments must precede native row access")
	}
	previous := catalogDeclarationAmendment{table: current, keys: append([]uniqueKey(nil), current.Unique...), columns: append([]*column(nil), current.Columns...)}
	if !amend(current) {
		return false
	}
	tx.mu.Lock()
	var history []catalogDeclarationAmendment
	if existing := tx.catalogAmendments.Load(); existing != nil {
		history = *existing
	}
	history = appendCatalogDeclarationHistory(history, previous)
	tx.catalogAmendments.Store(&history)
	tx.mu.Unlock()
	current.publishShowColumnsSnapshot(true)
	return true
}

// One immutable amendment history owns both private column/key changes and
// public two-endpoint foreign-key views. Savepoints retain only its position.
func appendCatalogDeclarationHistory(previous []catalogDeclarationAmendment, amendment catalogDeclarationAmendment) []catalogDeclarationAmendment {
	history := make([]catalogDeclarationAmendment, len(previous)+1)
	copy(history, previous)
	history[len(previous)] = amendment
	return history
}

// Caller has drained row workers and released decisionMu before schema M.
func (tx *TxContext) restoreCatalogDeclarations(keep int) {
	if tx.publicCatalogChanges() != nil {
		tx.restorePublicDeclarations(keep)
		return
	}
	if tx.coordinator != nil && tx.catalogIdentity.Load() != nil {
		return
	} // coordinator retires declarations and tables in one request
	amendments := tx.catalogAmendments.Load()
	if amendments == nil || len(*amendments) <= keep {
		return
	}
	if keep < 0 {
		panic("invalid native catalog declaration boundary")
	}
	identity := tx.catalogIdentity.Load()
	db := GetDatabase(identity.Database)
	if db == nil || db != identity.database || db.incarnationID() != identity.Incarnation {
		return
	}
	access := db.GetWrite(nil)
	defer access.Release()
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	for i := len(*amendments) - 1; i >= keep; i-- {
		amendment := (*amendments)[i]
		current := db.tables.Get(amendment.table.Name)
		if current == nil || current.RuntimeIdentity != amendment.table.RuntimeIdentity || !tx.ownsPendingTable(current) {
			continue
		}
		current.ddlMu.Lock()
		current.Unique = append([]uniqueKey(nil), amendment.keys...)
		current.Columns = append([]*column(nil), amendment.columns...)
		current.publishShowColumnsSnapshot(true)
		current.ddlMu.Unlock()
	}
	remaining := append([]catalogDeclarationAmendment(nil), (*amendments)[:keep]...)
	if len(remaining) == 0 {
		tx.catalogAmendments.Store(nil)
	} else {
		tx.catalogAmendments.Store(&remaining)
	}
}

func (tx *TxContext) retireCatalogSavepoint(tables, declarations int) {
	if tx.coordinator != nil && tx.catalogIdentity.Load() != nil {
		_, err := tx.coordinator.Call(scanQueryContext(tx, context.Background()), scm.ClusterRequest{Method: "catalog_retire", Arguments: []scm.Scmer{scm.NewInt(int64(tables)), scm.NewInt(int64(declarations))}})
		if err != nil {
			panic(err)
		}
		return
	}
	tx.restoreCatalogDeclarations(declarations)
	tx.discardCatalogIntents(tables)
}

func (tx *transactionState) applyCatalogKey(ctx context.Context, arguments []scm.Scmer) (scm.Scmer, error) {
	if len(arguments) != 7 {
		return scm.NewNil(), fmt.Errorf("invalid native catalog key declaration")
	}
	invocation := &TxContext{transactionState: tx, invocationContext: ctx}
	db := GetDatabase(arguments[0].String())
	if db == nil {
		return scm.NewNil(), fmt.Errorf("catalog database was removed")
	}
	target := db.GetTableForTx(arguments[1].String(), false, invocation)
	if target == nil || target.RuntimeIdentity != arguments[2].String() {
		return scm.NewNil(), fmt.Errorf("catalog key generation was replaced")
	}
	key := uniqueKey{Id: arguments[3].String(), Cols: scmerSliceToStrings(arguments[4].Slice()), NullsNotDistinct: scm.ToBool(arguments[5]), ConstraintName: arguments[6].String()}
	return scm.NewBool(invocation.createCatalogKey(target, key)), nil
}
