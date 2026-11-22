/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "maps"
import "context"
import "strings"
import "github.com/launix-de/memcp/scm"

// Negative lookup may finish at this immutable, transaction-visible catalog
// sample. A positive lookup grants no mutation authority: the existing native
// writer still reacquires coherence and validates its current declaration.
// In particular, probing an absent kind must not acquire a maintenance writer
// and conflict with another declaration already owned by this transaction.
func (t *table) declarationLookupSnapshot(tx *TxContext) *tableShowColumnsSnapshot {
	name := t.publishedName.Load()
	if name == nil || GetDatabase(t.schema.Name) != t.schema || t.schema.tables.Get(t.Name) != t {
		// A coherence acquisition may replace a local handle. Only the existing
		// mutation owner may rebind and validate that successor; an old snapshot
		// must not establish absence in the current native relation.
		return nil
	}
	return t.declarationMetadataForTx(tx)
}

func (t *table) mayHaveUniqueDeclaration(name string, tx *TxContext) bool {
	snapshot := t.declarationLookupSnapshot(tx)
	if snapshot == nil {
		return true
	}
	for _, row := range snapshot.nativeConstraints().Slice()[1].Slice() {
		id := foreignMetadataValue(row, "Id").String()
		alias := foreignMetadataValue(row, "constraint_name").String()
		if strings.EqualFold(id, name) || alias != "" && strings.EqualFold(alias, name) {
			return true
		}
	}
	return false
}

func (t *table) mayHaveForeignDeclaration(name string, tx *TxContext) bool {
	snapshot := t.declarationLookupSnapshot(tx)
	if snapshot == nil {
		return true
	}
	for _, row := range snapshot.nativeConstraints().Slice()[3].Slice() {
		if foreignMetadataValue(row, "Role").String() == "referencing" &&
			strings.EqualFold(foreignMetadataValue(row, "Id").String(), name) {
			return true
		}
	}
	return false
}

func (t *table) mayHaveDefaultDeclaration(name string, tx *TxContext) bool {
	snapshot := t.declarationLookupSnapshot(tx)
	if snapshot == nil {
		return true
	}
	for _, column := range snapshot.metadata.columnNames.declarations {
		if column.DefaultName != "" && strings.EqualFold(column.DefaultName, name) {
			return true
		}
	}
	return false
}

// Declaration amendments belong to the same private generation as CREATE.
// Immutable previous slices preserve interleaved column/key savepoint changes.
type catalogDeclarationAmendment struct {
	table            *table
	keys             []uniqueKey
	columns          []*column
	check            *catalogCheckAmendment
	checkDrop        *catalogCheckDropAmendment
	publicKey        *catalogPublicKeyAmendment
	foreignDrop      *catalogForeignDropAmendment
	foreign          *catalogForeignAmendment
	constraintState  *catalogConstraintStateAmendment
	source           *catalogSourceAmendment
	statementTrigger *catalogStatementTriggerAmendment
	column           *catalogStoredColumnAmendment
	tableDrop        *catalogTableDropAmendment
	tableRename      *catalogTableRenameAmendment
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
	if tx.catalogDeclarationDataEntered(t) || tx.hasNameAmendments() || tx.hasTableDrops() {
		return tx.stagePublicKey(t, "add", key)
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
		if column := catalogColumnInView(current, name, nil); column == nil {
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
	finish := tx.beginCatalogConstruction(catalogDeclarationConstruction, nil)
	defer finish()
	if tx.catalogDeclarationDataEntered(t) {
		panic("transactional declaration amendments must precede native row access")
	}
	changes := tx.publicCatalogChanges()
	if changes != nil {
		changes.mu.Lock()
		defer changes.mu.Unlock()
		if changes.owner != tx.transactionState || changes.db != t.schema {
			panic("private amendment lost its joined catalog owner")
		}
		for _, entry := range tx.publicForeignAmendments() {
			if entry.foreign != nil && (entry.foreign.endpoints[0] == t || entry.foreign.endpoints[1] == t) || entry.column != nil && entry.column.table == t || entry.publicKey != nil && entry.publicKey.table == t || entry.statementTrigger != nil && entry.statementTrigger.slot.table == t {
				panic("private declaration is already bound to its endpoint input")
			}
		}
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
	if tx.publicCatalogChanges() != changes {
		panic("private declaration phase changed during native admission; retry statement")
	}
	if tx.catalogDeclarationDataEntered(current) {
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
	if changes != nil {
		views := maps.Clone(changes.views.Load().tables)
		if _, tracked := changes.baselines[current]; tracked {
			changes.baselines[current] = foreignBaseline(current.showColumnsSnapshot.Load())
			views[current] = current.showColumnsSnapshot.Load()
		}
		changes.publishViews(history, views)
		changes.refreshMemoryLocked()
	}
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
	tx.retireEmptyPublicCatalogPhase()
	tx.retireEmptyLateCreatePhase()
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
