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

type catalogTableRename struct {
	Storage    string `json:"storage"`
	Identity   string `json:"identity"`
	Before     string `json:"before"`
	After      string `json:"after"`
	IgnoreCase bool   `json:"ignorecase"`
}

type catalogTableRenameAmendment struct {
	declaration *catalogTableRename
	table       *table
	previous    *tableShowColumnsSnapshot
	bytes       uint
}

// Part of the existing immutable original declaration publication, never a
// second catalog or a row capability. Physical registered identities stay put.
type catalogNamespaceView struct {
	names  map[*table]string
	exact  map[string]*table
	folded map[string]*table
}

func nativeNamespaceProjection(history []catalogDeclarationAmendment) *catalogNamespaceView {
	var result *catalogNamespaceView
	for _, entry := range history {
		if a := entry.tableRename; a != nil {
			if result == nil {
				result = &catalogNamespaceView{names: make(map[*table]string), exact: make(map[string]*table), folded: make(map[string]*table)}
			}
			result.names[a.table] = a.declaration.After
		}
	}
	if result == nil {
		return nil
	}
	for t, name := range result.names {
		result.exact[name] = t
		key := scm.CaseFoldKey(name)
		if _, exists := result.folded[key]; exists {
			result.folded[key] = nil
		} else {
			result.folded[key] = t
		}
	}
	return result
}

func (tx *TxContext) namespaceView(db *database) *catalogNamespaceView {
	if changes := tx.publicCatalogChanges(); changes != nil && changes.db == db {
		if views := changes.views.Load(); views != nil {
			return views.names
		}
	}
	return nil
}

func (t *table) rawVisibleName() string {
	if value := t.publishedName.Load(); value != nil {
		return *value
	}
	return t.Name
}

func (t *table) visibleNameAt(tx *TxContext, view *catalogSnapshot) string {
	if names := tx.namespaceView(t.schema); names != nil {
		if name, found := names.names[t]; found {
			return name
		}
	}
	if view != nil {
		if name, found := view.names[t]; found {
			return name
		}
	}
	return t.rawVisibleName() // unpublished constructor/load only
}

func validateTableRenameShape(d *catalogTableRename) error {
	if d == nil || d.Storage == "" || d.Before == "" || d.After == "" || !utf8.ValidString(d.Storage) || !utf8.ValidString(d.Before) || !utf8.ValidString(d.After) {
		return fmt.Errorf("invalid native table rename")
	}
	if id, err := uuid.Parse(d.Identity); err != nil || id == uuid.Nil {
		return fmt.Errorf("invalid native table rename identity")
	}
	return nil
}

func (tx *TxContext) renameCatalogTable(t *table, newName string, ignoreCase bool, identity string) bool {
	if tx == nil || tx.State != TxActive || tx.coordinator != nil || tx.remoteObject != nil || activeReplication(t.schema) != nil || sharedClusterDatabase(t.schema.Name) || t.Ephemeral || t.ScopedSchema {
		panic("table rename requires an original local native transaction")
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
	// The original owner may rename its exact unpublished generation. Public
	// name lookup intentionally rejects that generation for every other caller.
	identityValue := t.NativeIdentity(tx)
	if identityValue.IsNil() || identityValue.String() != identity {
		panic("table rename lost its expected native generation")
	}
	if !declaredTableMaintenanceCapabilities(t.schema.Name, t.Name).allows(maintenanceRename) {
		panic("Table " + t.schema.Name + "." + t.visibleNameAt(tx, t.schema.catalogView()) + " is required by MemCP and does not allow rename")
	}
	d := &catalogTableRename{t.Name, identity, t.visibleNameAt(tx, t.schema.catalogView()), newName, ignoreCase}
	if err := validateTableRenameShape(d); err != nil {
		panic(err)
	}
	return tx.stagePublicCatalogDeclaration(t.schema, catalogColumnKind, func(changes *publicCatalogChanges) bool { return changes.stageTableRename(tx, t, d) })
}

func (changes *publicCatalogChanges) stageTableRename(tx *TxContext, t *table, d *catalogTableRename) bool {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
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
		panic("table rename lost its original native declaration lease")
	}
	access := changes.db.GetWrite(tx)
	defer access.Release()
	changes.db.schemalock.Lock()
	defer changes.db.schemalock.Unlock()
	db := changes.db
	if db.tables.Get(d.Storage) != t || t.RuntimeIdentity != d.Identity || !t.visibleIn(tx, db.catalogView().version) || t.visibleNameAt(tx, db.catalogView()) != d.Before {
		panic("table rename lost its expected generation")
	}
	if db.namespaceOccupant(tx, d.After, d.IgnoreCase, t) != nil {
		panic("table rename target already exists")
	}
	if d.Before == d.After {
		return true
	}
	observed := t.showColumnsSnapshot.Load()
	base := changes.views.Load().tables[t]
	if base == nil {
		base = observed
	}
	a := &catalogTableRenameAmendment{declaration: d, table: t, previous: changes.views.Load().tables[t]}
	a.bytes = uint(unsafe.Sizeof(*a)) + uint(unsafe.Sizeof(*d)) + uint(len(d.Storage)+len(d.Identity)+len(d.Before)+len(d.After))
	history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{tableRename: a})
	views := maps.Clone(changes.views.Load().tables)
	views[t] = base
	history, views = changes.nameNeighborHistory(tx, t, endpoints, history, views)
	if _, exists := changes.baselines[t]; !exists {
		changes.baselines[t] = foreignBaseline(observed)
	}
	changes.baselinesHighWater = max(changes.baselinesHighWater, len(changes.baselines))
	changes.viewsHighWater = max(changes.viewsHighWater, len(views))
	changes.amendmentBytes += a.bytes
	changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	tx.noteStatementMutation(t)
	tx.catalogAmendments.Store(&history)
	changes.publishViews(history, views)
	changes.publishNamespaceReservationsLocked()
	accepted = true
	return true
}

// Names are reserved in the existing immutable pending namespace lane. The
// original resource owns both publication and removal of its finite aliases.
func (changes *publicCatalogChanges) publishNamespaceReservationsLocked() {
	key := (&TxContext{transactionState: changes.owner}).pendingCatalogKey()
	aliases := maps.Clone(changes.db.catalogView().pendingNames)
	if aliases == nil {
		aliases = make(map[string]map[*table]string)
	}
	names := changes.views.Load().names
	if names == nil || len(names.names) == 0 {
		delete(aliases, key)
	} else {
		aliases[key] = names.names
	}
	changes.db.publishTableNamesLocked(aliases)
}

func (changes *publicCatalogChanges) closeNamespaceReservationsLocked() {
	key := (&TxContext{transactionState: changes.owner}).pendingCatalogKey()
	if changes.db.catalogView().pendingNames[key] == nil {
		return
	}
	access := changes.db.GetWrite(nil)
	defer access.Release()
	changes.db.schemalock.Lock()
	defer changes.db.schemalock.Unlock()
	aliases := maps.Clone(changes.db.catalogView().pendingNames)
	delete(aliases, key)
	changes.db.publishTableNamesLocked(aliases)
}

// Caller owns the native schema namespace. Foreign owners' reservations are
// conflicts even though their pending logical address is not publicly visible.
func (db *database) namespaceOccupant(tx *TxContext, name string, ignoreCase bool, excluded *table) *table {
	matches := func(candidate string) bool {
		return candidate == name || ignoreCase && scm.CaseFoldKey(candidate) == scm.CaseFoldKey(name)
	}
	for _, t := range db.tables.GetAll() {
		if t == excluded || t.droppedAt(db.catalogView().version) || tx.ownsTableDrop(t) {
			continue
		}
		if matches(t.visibleNameAt(tx, db.catalogView())) {
			return t
		}
	}
	for _, byTable := range db.catalogView().pendingNames {
		for t, alias := range byTable {
			if t != excluded && !tx.ownsTableDrop(t) && matches(alias) {
				return t
			}
		}
	}
	return nil
}

func (tx *TxContext) hasNameAmendments() bool {
	for _, entry := range tx.publicForeignAmendments() {
		if entry.tableRename != nil || entry.column != nil && entry.column.rename != nil {
			return true
		}
	}
	return false
}

// Original-T FK retirement facts are merged only with this already-leased
// endpoint. No unleased catalog graph is copied or optimistically repaired.
func (tx *TxContext) tableHasSurvivingForeign(t *table) bool {
	history := tx.publicForeignAmendments()
	retired := make(map[catalogForeignIdentity][]catalogForeignDeclaration)
	for _, entry := range history {
		if d := entry.foreignDrop; d != nil {
			identity := foreignIdentity(d.declaration)
			retired[identity] = append(retired[identity], d.declaration)
		}
	}
	for _, fk := range t.Foreign {
		if !retiredForeignRecipe(t.schema, fk, retired) {
			return true
		}
	}
	for _, addition := range currentForeignAdditions(history) {
		if addition != nil && (addition.declaration.Foreign.Tbl1 == t.Name || addition.declaration.Foreign.Tbl2 == t.Name) {
			return true
		}
	}
	return false
}

func (db *database) projectTableRename(d *catalogTableRename, p *catalogNativeProjection, scope catalogForeignEndpointScope) {
	if err := validateTableRenameShape(d); err != nil {
		panic(err)
	}
	t := db.tables.Get(d.Storage)
	if t == nil || t.RuntimeIdentity != d.Identity || (!t.committedAt(db.catalogView().version) && !scope.owns(t)) {
		panic("table rename endpoint generation was replaced")
	}
	before, found := p.tableNames[t]
	if !found {
		before = t.rawVisibleName()
	}
	if before != d.Before {
		panic("table rename lost its namespace predecessor")
	}
	for _, other := range db.tables.GetAll() {
		if other == t || other.droppedAt(db.catalogView().version) {
			continue
		}
		if scope.retires(other) {
			for _, facts := range p.foreign {
				for _, fk := range facts {
					if fk.Foreign.Tbl1 == other.Name || fk.Foreign.Tbl2 == other.Name {
						panic("retired namespace predecessor still has foreign endpoints")
					}
				}
			}
			continue
		}
		name, found := p.tableNames[other]
		if !found {
			name = other.rawVisibleName()
		}
		if name == d.After || d.IgnoreCase && scm.CaseFoldKey(name) == scm.CaseFoldKey(d.After) {
			panic("table rename target namespace is occupied")
		}
	}
	p.tableNames[t] = d.After
	if p.propertyColumns[t] == nil {
		p.propertyColumns[t] = t.columnDeclarations()
	}
}

// The existing schema owner checks the final namespace as one set. This also
// validates newly created tagged headers when no explicit rename action exists.
func (db *database) validateNameNamespace(names map[*table]string, scope catalogForeignEndpointScope) {
	used := make(map[string]*table)
	for _, t := range db.tables.GetAll() {
		if t.droppedAt(db.catalogView().version) || scope.retires(t) {
			continue
		}
		if !t.committedAt(db.catalogView().version) && !scope.owns(t) {
			continue
		}
		name, found := names[t]
		if !found {
			name = t.rawVisibleName()
		}
		if before := used[name]; before != nil && before != t {
			panic("native final table namespace has duplicate names")
		}
		used[name] = t
	}
}
