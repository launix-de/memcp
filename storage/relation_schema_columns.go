/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"context"
	"fmt"
	"maps"

	"github.com/launix-de/memcp/scm"
)

// An original decision owns the pending immutable header and its existing
// table declaration lease. Physical columns remain in the committed layout
// until the row decision; old input/reader descriptors are never rewritten.
type relationSchemaLayout struct {
	table   *table
	scope   *relationScope
	lease   *tableWriteLease
	view    *tableShowColumnsSnapshot
	retired map[string]bool
}

type relationSchemaLayoutChange struct {
	before *relationSchemaLayout
	after  *relationSchemaLayout
}

func (tx *transactionState) scopedRelationLayout(t *table) *relationSchemaLayout {
	if tx != nil {
		if layouts := tx.relationSchemaLayouts.Load(); layouts != nil {
			return (*layouts)[t]
		}
	}
	return nil
}

func (tx *TxContext) scopedColumnMutationAllowed(t *table) bool {
	return tx != nil && tx.State == TxActive && tx.mutationOperation != nil && tx.scopedRelationLayout(t) != nil
}

// Caller owns tx.mu. Readers borrow one immutable native header at operator
// setup, just as public column declarations do; there is no row-loop lock.
func (tx *transactionState) storeScopedRelationLayout(layout *relationSchemaLayout) {
	current := make(map[*table]*relationSchemaLayout)
	if previous := tx.relationSchemaLayouts.Load(); previous != nil {
		current = maps.Clone(*previous)
	}
	current[layout.table] = layout
	tx.relationSchemaLayouts.Store(&current)
}

// Match the existing native scratch/input owner: a supplied handle or an
// Ephemeral flag alone conveys neither a connection nor a lexical namespace.
// Both declaration construction and physical inputs attest this same slot.
func (tx *TxContext) requireScopedRelationOwner(t *table) *relationScope {
	if tx == nil || tx.transactionState == nil || tx.State != TxActive || tx.decisionStarted ||
		tx.coordinator != nil || tx.remoteObject != nil || tx.remoteInvocation || tx.relationData != nil ||
		t == nil || !t.Ephemeral || !t.ScopedSchema || t.PersistencyMode != Memory || t.dataOwner != nil ||
		activeReplication(t.schema) != nil || sharedClusterDatabase(t.schema.Name) {
		panic("column declaration requires its original local caller-owned relation")
	}
	scope := t.relationLifetime
	if scope == nil || !scope.transactionalSchema || scope.dataPolicy != relationCallerData ||
		tx.Session.IsNil() || scope.owner != relationScopeOwnerFor(tx.Session) || !scope.owns(t) {
		panic("column declaration belongs to another native relation scope owner")
	}
	if GetDatabase(t.schema.Name) != t.schema || t.schema.tables.Get(t.Name) != t ||
		!t.visibleIn(tx, t.schema.catalogView().version) {
		panic("column declaration lost its native relation generation")
	}
	// A retained handle to an earlier DROP/recreate generation cannot mutate
	// the replacement or a table outside this caller's current lexical slots.
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if scope.closing || scope.schemaClosed.Load() {
		panic("relation scope is closed")
	}
	found := false
	if slots := scope.schemaSlots.Load(); slots != nil {
		for _, slot := range *slots {
			view := slot.view.Load()
			if view != nil && view.visible(tx) == t {
				if view.constructing || view.owner != nil && view.owner != tx.transactionState {
					panic("relation schema slot has an unfinished native owner")
				}
				found = true
			}
		}
	}
	if !found {
		panic("column declaration is outside its native lexical generation")
	}
	return scope
}

func (tx *TxContext) requireScopedColumnOwner(t *table) *relationScope {
	if t != nil && (t.mutationReadOnly || t.transitionRelation != nil) {
		panic("native result relation is read-only")
	}
	return tx.requireScopedRelationOwner(t)
}

func (tx *TxContext) dropScopedColumn(t *table, name string, ifExists bool, expectedInput scm.Scmer) bool {
	scope := tx.requireScopedColumnOwner(t)
	finish := tx.beginCatalogConstruction(catalogValidationConstruction, nil, t)
	defer finish()
	releaseScope := scope.beginUse()
	defer releaseScope()
	tx.requireDeclarationEndpoint(t, maintenanceAlter)
	tx.getOrComputeResource(fmt.Sprintf("relation-schema-retirement/%p", scope), scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		return scm.NewAny(&relationSchemaRetirement{scope})
	}))
	// The actual endpoint owner upgrades only this already-attested scoped
	// transaction's completed writer. A raw retained callback or active producer
	// refuses the change before any header publication, without waiting on T.
	value := tx.getOrComputeResource(fmt.Sprintf("relation-schema-layout/%p", t), scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		lease, err := t.writeAdmission.admit(scanQueryContext(tx, context.Background()), tx.transactionState, tableScopedDeclarationAdmission, false)
		if err != nil {
			panic(err)
		}
		ready := false
		defer func() {
			if !ready {
				_ = lease.Close()
			}
		}()
		lease.retainDDL(t, tx.transactionState)
		tx.requireScopedColumnOwner(t)
		layout := &relationSchemaLayout{table: t, scope: scope, lease: lease, view: t.catalogVisibleMetadata()}
		// Even a later validation refusal retains this exact endpoint resource
		// through T. Its baseline journal entry keeps scope close from waiting
		// for that resource before the original decision can release it.
		scope.mu.Lock()
		scope.schemaDecisions++
		scope.mu.Unlock()
		tx.mu.Lock()
		tx.relationSchemaChanges = append(tx.relationSchemaChanges, &relationSchemaChange{scope: scope,
			layout: &relationSchemaLayoutChange{layout, layout}})
		tx.storeScopedRelationLayout(layout)
		tx.mu.Unlock()
		ready = true
		return scm.NewAny(lease)
	}))
	lease, ok := value.Any().(*tableWriteLease)
	if !ok || lease == nil || lease.owner != &t.writeAdmission || lease.transaction != tx.transactionState {
		panic("column declaration lost its native endpoint lease")
	}
	access := t.schema.GetWrite(tx)
	defer access.Release()
	tx.requireScopedColumnOwner(t)
	t.schema.schemalock.Lock()
	defer t.schema.schemalock.Unlock()
	t.mu.Lock()
	lifecycle := false
	for _, tr := range t.Triggers {
		lifecycle = lifecycle || tr.Timing == AfterDropColumn
	}
	t.mu.Unlock()
	if lifecycle {
		panic("transactional column retirement requires lifecycle consumers to be removed")
	}
	before := tx.scopedRelationLayout(t)
	columns := before.view.metadata.columnNames.declarations
	ordinal := -1
	for i, c := range columns {
		if c.Name == name {
			ordinal = i
			break
		}
	}
	if ordinal < 0 {
		if ifExists {
			return false
		}
		panic("drop column does not exist: " + t.Name + "." + name)
	}
	c := columns[ordinal]
	if !expectedInput.IsNil() {
		requireExpectedStoredColumnInput(c, expectedInput)
	}
	// DROP owns removal of its default too, using the same native declaration
	// formula as public chronological clear-default -> drop. No callback runs.
	if !c.Default.IsNil() || c.DefaultName != "" || c.DefaultExpression != "" || c.DefaultDeferred {
		columns = append([]*column(nil), columns...)
		c = cloneStoredColumnDeclaration(c)
		c.Alter("default", scm.NewNil())
		columns[ordinal] = c
	}
	d := &catalogColumnDrop{t.Name, t.RuntimeIdentity, c.Name, c.displayName(), ordinal, storedColumnTypeOf(c)}
	if err := validateColumnDropShape(d); err != nil {
		panic(err)
	}
	columns = t.projectStoredColumnDrop(d, columns, newCatalogNativeProjection(t.schema))
	view, _ := replaceStoredColumnMetadata(t, before.view, columns, t.columnNamesSnapshot.Load())
	retired := maps.Clone(before.retired)
	if retired == nil {
		retired = make(map[string]bool)
	}
	retired[c.Name] = true
	after := &relationSchemaLayout{table: t, scope: scope, lease: lease, view: view, retired: retired}
	change := &relationSchemaChange{scope: scope, layout: &relationSchemaLayoutChange{before, after}}
	// Reattest the completion boundary after validation: another operator may
	// have entered while this builder was preparing its immutable header. All
	// gates below are existing cold input/statement owners; no wait, callback or
	// schema acquisition occurs while they are held.
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, tr := range t.Triggers {
		if tr.Timing == AfterDropColumn {
			panic("column retirement acquired a lifecycle consumer")
		}
	}
	t.writeAdmission.mu.Lock()
	defer t.writeAdmission.mu.Unlock()
	if t.writeAdmission.declaration != tx.transactionState || t.writeAdmission.statement != nil ||
		t.writeAdmission.writers[tx.transactionState].unframedCallbacks {
		panic("column declaration input is in use by a native producer")
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if scope.dataInputs[tx.transactionState] != 0 {
		panic("column declaration input is in use by the original caller")
	}
	tx.noteStatementMutation(t)
	scope.schemaDecisions++
	tx.mu.Lock()
	tx.relationSchemaChanges = append(tx.relationSchemaChanges, change)
	tx.storeScopedRelationLayout(after)
	tx.mu.Unlock()
	return true
}

func (tx *transactionState) restoreScopedRelationLayout(change *relationSchemaLayoutChange) {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	tx.storeScopedRelationLayout(change.before)
}

// Ordinary query preparation may install native temporary computed columns
// while borrowing this same endpoint's DDL lease. They retain their existing
// query/cache lifetime, not the SQL stored-column decision or its row Undo.
func (tx *TxContext) refreshScopedRuntimeColumns(t *table) {
	t.schema.schemalock.Lock()
	defer t.schema.schemalock.Unlock()
	before := tx.scopedRelationLayout(t)
	if before == nil {
		return
	}
	columns := append([]*column(nil), before.view.metadata.columnNames.declarations...)
	for _, c := range t.columnDeclarations() {
		if !c.IsTemp {
			continue
		}
		found := false
		for _, previous := range columns {
			found = found || previous.Name == c.Name
		}
		if !found {
			columns = append(columns, c)
		}
	}
	if len(columns) == len(before.view.metadata.columnNames.declarations) {
		return
	}
	view, _ := replaceStoredColumnMetadata(t, before.view, columns, t.columnNamesSnapshot.Load())
	after := *before
	after.view = view
	tx.mu.Lock()
	tx.storeScopedRelationLayout(&after)
	tx.mu.Unlock()
}

// The resource decision has already joined checked producers and row Undo.
// Its retained endpoint DDL lease prevents a rebuild from discarding original
// columns before this boundary. No shard payload is deleted or reinterpreted.
func (tx *transactionState) publishScopedRelationLayouts() {
	layouts := tx.relationSchemaLayouts.Load()
	if layouts == nil {
		return
	}
	for t, layout := range *layouts {
		if len(layout.retired) == 0 {
			continue
		}
		access := t.schema.GetWrite(nil)
		func() {
			defer access.Release()
			t.schema.schemalock.Lock()
			defer t.schema.schemalock.Unlock()
			if GetDatabase(t.schema.Name) != t.schema || t.schema.tables.Get(t.Name) != t || !layout.scope.owns(t) {
				panic("relation column decision lost its native generation")
			}
			t.Columns = append([]*column(nil), layout.view.metadata.columnNames.declarations...)
			if t.DroppedColumns == nil {
				t.DroppedColumns = make(map[string]bool)
			}
			for name := range layout.retired {
				t.DroppedColumns[name] = true
			}
			t.publishShowColumnsSnapshot(true)
		}()
	}
}
