/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"
import "unsafe"
import "sync/atomic"
import "github.com/google/uuid"

// A reset is a declaration, not a reservation. Its original native transaction
// selects the epoch. Actual reservations are nontransactional within that epoch:
// savepoint rewind selects an earlier branch, never rewinds a surviving cursor.
type catalogAllocatorReset struct {
	table  *table
	column string
	state  atomic.Pointer[tableAllocatorState]
	bytes  uint
}

type catalogAllocatorResetDeclaration struct {
	Table  string              `json:"table"`
	Column string              `json:"column"`
	State  tableAllocatorState `json:"state"`
}

func (t *table) transactionAllocator(tx *TxContext) *catalogAllocatorReset {
	if tx == nil {
		return nil
	}
	changes := tx.publicCatalogChanges()
	if changes == nil {
		return nil
	}
	if view := changes.views.Load(); view != nil {
		return view.allocators[t]
	}
	return nil
}

func (branch *catalogAllocatorReset) reserve(input tableIDExplicit, count uint64) tableIDReservation {
	for {
		previous := branch.state.Load()
		reservation, next, changed := epochTableIDReservation(*previous, input, count)
		if !changed || branch.state.CompareAndSwap(previous, &next) {
			return reservation
		}
	}
}

func (t *table) resetTableIDs(column, token string, current, next int64, hasNext bool, tx *TxContext) bool {
	if tx == nil || t.RuntimeIdentity == "" || !localCatalogRowDatabase(t.schema) {
		panic("allocator reset requires its original local native transaction and table generation")
	}
	if !hasNext && next != 0 {
		panic("exhausted allocator cannot carry a next value")
	}
	requireTableMaintenance(t.schema.Name, t.Name, maintenanceAlter)
	// Recover legacy observations before declaration admission, never while
	// holding a table's exclusive maintenance or physical shard ownership.
	t.prepareAllocatorRecovery()
	finish := tx.beginCatalogConstruction(catalogAllocatorResetConstruction, nil, t)
	defer func() {
		failure := recover()
		finish()
		tx.retireUnacceptedPublicCatalogPhase()
		if failure != nil {
			panic(failure)
		}
	}()
	return tx.stagePublicCatalogDeclaration(t.schema, catalogAllocatorResetKind, nil, func(changes *publicCatalogChanges) bool {
		changes.mu.Lock()
		defer changes.mu.Unlock()
		defer changes.refreshMemoryLocked()
		accepted := false
		finishLease := changes.acquireStoredColumnLease(tx, t, true)
		defer func() { finishLease(accepted) }()
		access := t.schema.GetWrite(tx)
		defer access.Release()
		t.schema.schemalock.Lock()
		defer t.schema.schemalock.Unlock()
		if t.schema.tables.Get(t.Name) != t || !t.visibleIn(tx, t.schema.catalogView().version) {
			panic("allocator reset lost its visible table generation")
		}
		observed := t.showColumnsSnapshot.Load()
		if private := changes.views.Load().tables[t]; private != nil {
			observed = private
		}
		found := false
		for _, c := range observed.metadata.columnNames.declarations {
			if c.Name == column && c.AutoIncrement {
				found = true
				break
			}
		}
		if !found {
			panic("allocator reset requires the native identity column")
		}
		state := t.localTableAllocator()
		if branch := t.transactionAllocator(tx); branch != nil {
			state = *branch.state.Load()
		}
		if tableAllocatorToken(state) != token {
			panic("allocator reset observation changed; retry transaction")
		}
		state.Version, state.Epoch = 2, uuid.NewString()
		state.Current, state.Next, state.HasNext = current, next, hasNext
		if err := state.validate(); err != nil {
			panic(err)
		}
		branch := &catalogAllocatorReset{table: t, column: column}
		branch.state.Store(&state)
		branch.bytes = uint(unsafe.Sizeof(*branch)) + uint(unsafe.Sizeof(state)) + uint(len(column)+len(state.Table)+len(state.Epoch))
		history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{allocator: branch})
		views := maps.Clone(changes.views.Load().tables)
		if views[t] == nil {
			views[t] = observed
		}
		changes.amendmentBytes += branch.bytes
		changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
		tx.catalogAmendments.Store(&history)
		changes.publishViews(history, views)
		accepted = true
		return true
	})
}

func (tx *TxContext) allocatorResetDeclarations() []catalogAllocatorResetDeclaration {
	changes := tx.publicCatalogChanges()
	if changes == nil {
		return nil
	}
	view := changes.views.Load()
	if view == nil {
		return nil
	}
	var result []catalogAllocatorResetDeclaration
	// History order is deterministic; only the selected final branch survives.
	for _, entry := range view.history {
		a := entry.allocator
		if a == nil || view.allocators[a.table] != a {
			continue
		}
		if _, dropped := view.dropped[a.table]; dropped {
			continue
		}
		result = append(result, catalogAllocatorResetDeclaration{a.table.Name, a.column, *a.state.Load()})
	}
	return result
}

// Caller owns the native catalog publication boundary or unpublished recovery.
// Runtime publication never consults WAL maps or takes their log mutex.
// The decision, not arithmetic max, selects the new epoch. WAL can only advance
// a cursor inside that selected epoch, including after schema checkpoint lag.
func (db *database) publishAllocatorResets(declarations []catalogAllocatorResetDeclaration, recovering bool) {
	for _, d := range declarations {
		t := db.tables.Get(d.Table)
		if t == nil || t.RuntimeIdentity != d.State.Table {
			panic("allocator reset lost its declared table UUID")
		}
		state := d.State
		if recovering {
			if logged, ok := db.tableAllocators[state.Table+"/"+state.Epoch]; ok {
				var err error
				state, err = mergeTableAllocator(state, logged)
				if err != nil {
					panic(err)
				}
			}
		}
		t.allocator.Store(&state)
		if recovering {
			t.Allocator = &state
		}
	}
}

func validateAllocatorResetDeclarations(declarations []catalogAllocatorResetDeclaration) error {
	seen := make(map[string]bool, len(declarations))
	for _, d := range declarations {
		if d.Table == "" || d.Column == "" || d.State.Version != 2 || seen[d.State.Table] {
			return fmt.Errorf("invalid native allocator reset declaration")
		}
		if err := d.State.validate(); err != nil {
			return err
		}
		seen[d.State.Table] = true
	}
	return nil
}
