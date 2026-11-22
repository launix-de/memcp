/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
	"sync/atomic"
)

// The normal catalog reserves names for both committed and unpublished tables.
// One immutable publication determines visibility and case-folded lookup, so a
// multi-table decision never becomes a sequence of individually visible names.
type catalogSnapshot struct {
	version      uint64
	exact        map[string]*table
	names        map[*table]string
	pendingExact map[string]map[string]*table
	pendingNames map[string]map[*table]string
	folded       map[string]string
	pending      map[string]map[string]string
}
type catalogVersion struct{ atomic.Uint64 }

func (v *catalogVersion) MarshalJSON() ([]byte, error) { return json.Marshal(v.Load()) }
func (v *catalogVersion) UnmarshalJSON(data []byte) error {
	var value uint64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	v.Store(value)
	return nil
}

func (db *database) catalogView() *catalogSnapshot {
	if view := db.catalog.Load(); view != nil {
		return view
	}
	return &catalogSnapshot{version: db.catalogVersion}
}

// Pair the immutable logical cut with its runtime directory. DROP detaches
// physical entries after the cut; an old cut cannot consume a partial detach.
func (db *database) catalogTablesSnapshot() (*catalogSnapshot, []*table) {
	for {
		view := db.catalog.Load()
		tables := db.tables.GetAll()
		if db.catalog.Load() != view {
			continue
		}
		if view == nil {
			view = &catalogSnapshot{version: db.catalogVersion}
		}
		return view, tables
	}
}
func (t *table) committedAt(version uint64) bool {
	if t.droppedAt(version) {
		return false
	}
	if t.ScopedSchema {
		if t.CatalogVersion.Load() != 0 {
			return true
		}
		owner := t.creator.Load()
		return owner != nil && owner.relationSchemaDecision.Load() == relationSchemaCommitted
	}
	if t.CreatorTxID == "" {
		return true
	}
	return t.CatalogVersion.Load() != 0 && t.CatalogVersion.Load() <= version
}
func (t *table) visibleIn(tx *TxContext, version uint64) bool {
	if tx.ownsTableDrop(t) || t.droppedAt(version) {
		return false
	}
	if t.committedAt(version) {
		return true
	}
	return tx != nil && tx.State == TxActive && (tx.ownsPendingTable(t) || tx.ownsScopedTable(t))
}

// NativeIdentity projects the existing persisted generation identity. A stale
// handle or another transaction's unpublished generation cannot become an
// authority merely because its former name still exists in the catalog.
// Local legacy identities are checkpointed by the load producer. Other schemas
// may still lack one; a getter must never allocate one on their publisher's behalf.
func (t *table) NativeIdentity(tx *TxContext) scm.Scmer {
	if t == nil || t.schema == nil {
		return scm.NewNil()
	}
	db := databases.Get(t.schema.Name)
	if db == nil || db != t.schema {
		panic("native relation identity belongs to a replaced database")
	}
	view := db.catalogView()
	current := db.tables.Get(t.Name)
	if current == nil || !current.visibleIn(tx, view.version) || current.RuntimeIdentity != t.RuntimeIdentity {
		panic("native relation identity is not visible in its current generation")
	}
	if current.RuntimeIdentity == "" {
		return scm.NewNil()
	}
	if _, err := uuid.Parse(current.RuntimeIdentity); err != nil {
		panic("native relation identity is invalid")
	}
	return scm.NewString(current.RuntimeIdentity)
}
func (db *database) GetTableForTx(name string, ignoreCase bool, tx *TxContext) *table {
	db.ensureLoaded()
	view := db.catalogView()
	if len(name) > 0 && name[0] == '.' {
		if t := db.tables.Get(name); t != nil && t.isHiddenFromShowTables() && t.visibleIn(tx, view.version) {
			return t
		}
	}
	own := tx.namespaceView(db)
	selected := (*table)(nil)
	accept := func(candidate *table) {
		if candidate == nil || !candidate.visibleIn(tx, view.version) {
			return
		}
		visible := candidate.visibleNameAt(tx, view)
		if visible != name && !(ignoreCase && scm.CaseFoldKey(visible) == scm.CaseFoldKey(name)) {
			return
		}
		if selected != nil && selected != candidate {
			panic("ambiguous table name: " + name)
		}
		selected = candidate
	}
	if !ignoreCase {
		if own != nil {
			accept(own.exact[name])
		}
		if tx != nil && tx.State == TxActive {
			accept(view.pendingExact[tx.pendingCatalogKey()][name])
		}
		accept(view.exact[name])
	} else {
		key := scm.CaseFoldKey(name)
		if own != nil {
			if candidate, found := own.folded[key]; found {
				if candidate == nil {
					panic("ambiguous table name: " + name)
				}
				accept(candidate)
			}
		}
		if tx != nil && tx.State == TxActive {
			if keyName, found := view.pending[tx.pendingCatalogKey()][key]; found {
				if keyName == "" {
					panic("ambiguous table name: " + name)
				}
				accept(db.tables.Get(keyName))
			}
		}
		if keyName, found := view.folded[key]; found {
			if keyName == "" {
				panic("ambiguous table name: " + name)
			}
			accept(db.tables.Get(keyName))
		}
	}
	if selected != nil {
		return selected
	}
	return db.getLegacyTempTable(name, tx, view.version)
}

func (tx *TxContext) hasCatalogIntents() bool {
	return tx != nil && (tx.catalogIntents.Load() != nil || tx.publicDeclarationPosition() != 0)
}

func (db *database) showTableMetadata(tx *TxContext) scm.Scmer {
	db.ensureLoaded()
	view, tables := db.catalogTablesSnapshot()
	rows := make([]scm.Scmer, 0, len(tables))
	for _, t := range tables {
		if !t.visibleIn(tx, view.version) || t.isHiddenFromShowTables() {
			continue
		}
		engine := showEngineStr(t)
		stats := t.statistics()
		memory := showBuildMemory(t.residentMemory()).Slice()
		rows = append(rows, scm.NewSlice(append(memory, []scm.Scmer{
			scm.NewString("name"), scm.NewString(t.visibleNameAt(tx, view)),
			scm.NewString("engine"), scm.NewString(engine),
			scm.NewString("row_count"), scm.NewInt(stats.rowCount),
			scm.NewString("collation"), scm.NewString(t.Collation),
			scm.NewString("comment"), scm.NewString(t.Comment),
		}...)))
	}
	return scm.NewSlice(rows)
}

func (db *database) incarnationID() string {
	if id := db.incarnation.Load(); id != nil {
		return *id
	}
	return ""
}

func (tx *transactionState) pendingCatalogKey() string {
	if identity := tx.catalogIdentity.Load(); identity != nil {
		return identity.Decision
	}
	return ""
}
func (tx *transactionState) ownsPendingTable(t *table) bool {
	identity := tx.catalogIdentity.Load()
	return identity != nil && tx.durableID() == t.CreatorTxID && identity.Decision == t.CatalogDecisionID && identity.Incarnation == t.CreatorIncarnation && identity.Namespace == t.CreatorNamespace && identity.Database == t.schema.Name
}

func (tx *TxContext) catalogIntentTables() []*table {
	if intents := tx.catalogIntents.Load(); intents != nil {
		return *intents
	}
	return nil
}

func (tx *transactionState) ownsScopedTable(t *table) bool {
	return t.ScopedSchema && tx.durableID() == t.CreatorTxID
}
