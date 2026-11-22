/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"fmt"
	"maps"
	"unicode/utf8"
	"unsafe"

	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
)

// A name action changes visible metadata, never its physical row/file key.
// It belongs to the sole ordered native declaration interpreter.
type catalogColumnRename struct {
	Table      string           `json:"table"`
	Identity   string           `json:"identity"`
	Storage    string           `json:"storage"`
	Before     string           `json:"before"`
	After      string           `json:"after"`
	Expected   storedColumnType `json:"expected"`
	IgnoreCase bool             `json:"ignorecase"`
}

func (record *catalogDecisionRecord) hasNameDeclarationActions() bool {
	for _, t := range record.Tables {
		if t != nil && (t.rawVisibleName() != t.Name || t.hasBoundColumnNames()) {
			return true
		}
	}
	for _, d := range record.ColumnAppends {
		if d.NameBinding != nil {
			return true
		}
	}
	for _, action := range record.NativeDeclarations {
		if action.Operation == "column_rename" || action.Operation == "table_rename" {
			return true
		}
	}
	return false
}

func (record *catalogDecisionRecord) hasOwnedKeyDeclarationActions() bool {
	identities := make(map[string]bool, len(record.Tables))
	for _, t := range record.Tables {
		if t != nil {
			identities[t.RuntimeIdentity] = true
		}
	}
	for _, action := range record.NativeDeclarations {
		if action.Key != nil && identities[action.Key.Identity] {
			return true
		}
	}
	return false
}

func (d *catalogColumnRename) memory() uint {
	if d == nil {
		return 0
	}
	return uint(unsafe.Sizeof(*d)) + uint(len(d.Table)+len(d.Identity)+len(d.Storage)+len(d.Before)+len(d.After)+len(d.Expected.Type)+len(d.Expected.Collation)) + uint(cap(d.Expected.Dimensions))*uint(unsafe.Sizeof(int(0)))
}

func validateColumnRenameShape(d *catalogColumnRename) error {
	if d == nil || d.Table == "" || d.Storage == "" || d.Before == "" || d.After == "" || !utf8.ValidString(d.Table) || !utf8.ValidString(d.Storage) || !utf8.ValidString(d.Before) || !utf8.ValidString(d.After) {
		return fmt.Errorf("invalid native column rename")
	}
	id, err := uuid.Parse(d.Identity)
	if err != nil || id == uuid.Nil {
		return fmt.Errorf("invalid native column rename generation")
	}
	if d.Expected.Type == "" || !utf8.ValidString(d.Expected.Type) || !utf8.ValidString(d.Expected.Collation) {
		return fmt.Errorf("invalid native column rename declaration")
	}
	for _, n := range d.Expected.Dimensions {
		if n < -1 {
			return fmt.Errorf("invalid native column rename dimension")
		}
	}
	return nil
}

func (db *database) projectColumnRename(d *catalogColumnRename, columns []*column) (*table, []*column) {
	if err := validateColumnRenameShape(d); err != nil {
		panic(err)
	}
	t := db.tables.Get(d.Table)
	if t == nil || t.RuntimeIdentity != d.Identity || !t.committedAt(db.catalogView().version) {
		panic("column rename endpoint generation was replaced")
	}
	if columns == nil {
		columns = t.columnDeclarations()
	}
	index := -1
	for i, c := range columns {
		if c.Name == d.Storage {
			if c.displayName() != d.Before || !storedColumnTypeOf(c).equal(d.Expected) {
				panic("column rename lost its expected declaration")
			}
			if c.IsTemp || isRuntimeComputedColumn(c) || c.ComputorDefinition != nil {
				panic("column rename requires an ordinary stored column")
			}
			index = i
		} else if c.displayName() == d.After || d.IgnoreCase && scm.CaseFoldKey(c.displayName()) == scm.CaseFoldKey(d.After) {
			panic("column rename target already exists")
		}
	}
	if index < 0 {
		panic("column rename physical declaration was removed")
	}
	replacement := cloneStoredColumnDeclaration(columns[index])
	replacement.visibleName = d.After
	updated := append([]*column(nil), columns...)
	updated[index] = replacement
	return t, updated
}

func (tx *TxContext) renamePublicColumn(t *table, before, after, storage string, expected storedColumnType, ignoreCase bool) bool {
	if tx == nil || tx.transactionState == nil || tx.State != TxActive || tx.coordinator != nil || t == nil || activeReplication(t.schema) != nil || sharedClusterDatabase(t.schema.Name) || t.Ephemeral || t.ScopedSchema {
		panic("column rename requires an original local native transaction")
	}
	finish := tx.beginCatalogConstruction(catalogValidationConstruction, t)
	defer func() {
		failure := recover()
		finish()
		tx.retireUnacceptedPublicCatalogPhase()
		if failure != nil {
			panic(failure)
		}
	}()
	requireTableMaintenance(t.schema.Name, t.Name, maintenanceAlter)
	d := &catalogColumnRename{Table: t.Name, Identity: t.RuntimeIdentity, Storage: storage, Before: before, After: after, Expected: expected, IgnoreCase: ignoreCase}
	if err := validateColumnRenameShape(d); err != nil {
		panic(err)
	}
	return tx.stagePublicCatalogDeclaration(t.schema, catalogColumnKind, func(changes *publicCatalogChanges) bool { return changes.stageColumnRename(tx, t, d) })
}

func (changes *publicCatalogChanges) stageColumnRename(tx *TxContext, t *table, d *catalogColumnRename) bool {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	if tx.hasTableDrops() {
		panic("column rename cannot join table retirement in this profile")
	}
	accepted := false
	endpoints, finishLease := changes.acquireNameLeases(tx, t)
	defer func() { finishLease(accepted) }()
	restore := t.guardCatalogDeclarationInput(tx.transactionState)
	defer func() {
		if !accepted {
			restore()
			changes.publishViews(tx.publicForeignAmendments(), changes.views.Load().tables)
		}
	}()
	lease := changes.leases[t]
	changes.mu.Unlock()
	func() { defer changes.mu.Lock(); lease.retainDDL(t, tx.transactionState) }()
	if changes.leases[t] != lease {
		panic("column rename lost its native declaration lease")
	}
	access := changes.db.GetWrite(tx)
	defer access.Release()
	changes.db.schemalock.Lock()
	defer changes.db.schemalock.Unlock()
	changes.validateConstraintStateEndpointsLocked(tx)
	observed := t.showColumnsSnapshot.Load()
	base := changes.views.Load().tables[t]
	if base == nil {
		base = observed
	}
	_, columns := changes.db.projectColumnRename(d, base.metadata.columnNames.declarations)
	if d.Before == d.After {
		return true
	}
	view, bytes := replaceStoredColumnMetadata(t, base, columns)
	a := &catalogStoredColumnAmendment{rename: d, table: t, columns: columns, previous: changes.views.Load().tables[t], columnBytes: storedColumnBytes(columns[base.metadata.columnNames.exact[d.Storage].ordinal])}
	a.retainedBytes = uint(unsafe.Sizeof(*a)) + uint(cap(columns))*uint(unsafe.Sizeof((*column)(nil))) + a.columnBytes + bytes + d.memory()
	history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{column: a})
	views := maps.Clone(changes.views.Load().tables)
	views[t] = view
	history, views = changes.nameNeighborHistory(tx, t, endpoints, history, views)
	if _, exists := changes.baselines[t]; !exists {
		changes.baselines[t] = foreignBaseline(observed)
	}
	changes.baselinesHighWater = max(changes.baselinesHighWater, len(changes.baselines))
	changes.viewsHighWater = max(changes.viewsHighWater, len(views))
	changes.amendmentBytes += a.retainedBytes
	changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	tx.noteStatementMutation(t)
	tx.catalogAmendments.Store(&history)
	changes.publishViews(history, views)
	accepted = true
	return true
}

func (t *table) hasBoundColumnNames() bool {
	for _, c := range t.Columns {
		if c.visibleName != "" {
			return true
		}
	}
	return false
}
