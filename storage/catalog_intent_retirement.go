/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "maps"
import "slices"

// Savepoint restoration has selected the declaration prefix before private
// constructors are discarded. This is an endpoint-reference check on that
// same history, not another declaration projection or recovery interpreter.
func (entry catalogDeclarationAmendment) referencesTable(t *table) bool {
	if entry.table == t || entry.allocator != nil && entry.allocator.table == t ||
		entry.column != nil && entry.column.table == t ||
		entry.publicKey != nil && entry.publicKey.table == t ||
		entry.tableDrop != nil && entry.tableDrop.table == t ||
		entry.tableRename != nil && entry.tableRename.table == t ||
		entry.check != nil && entry.check.table == t ||
		entry.checkDrop != nil && entry.checkDrop.table == t ||
		entry.statementTrigger != nil && entry.statementTrigger.slot.table == t {
		return true
	}
	if entry.foreign != nil && slices.Contains(entry.foreign.endpoints[:], t) ||
		entry.foreignDrop != nil && slices.Contains(entry.foreignDrop.endpoints[:], t) {
		return true
	}
	if state := entry.constraintState; state != nil {
		for _, view := range state.views {
			if view.table == t {
				return true
			}
		}
		for _, states := range [][]*catalogConstraintState{state.previous, state.current} {
			for _, s := range states {
				if s != nil && slices.Contains(s.endpoints(), t) {
					return true
				}
			}
		}
	}
	return false
}

// Caller owns schema publication and the existing drained savepoint/rollback
// boundary. A same-name replacement, even with copied IDs, is never authority
// to release a lease or remove storage. A future prepared version is not a
// published cut: failed preparation must still retire its original constructor.
// Validate the whole suffix before effects.
func validateCancelledCatalogIntents(tx *TxContext, db *database, cancelled []*table, history []catalogDeclarationAmendment) {
	for _, t := range cancelled {
		if t == nil || t.schema != db || db.tables.Get(t.Name) != t || t.RuntimeIdentity == "" ||
			t.creator.Load() != tx.transactionState || t.committedAt(db.catalogView().version) || !tx.ownsPendingTable(t) {
			panic("private catalog retirement lost its original native generation")
		}
		for _, entry := range history {
			if entry.referencesTable(t) {
				panic("private catalog retirement retains a declaration endpoint")
			}
		}
	}
}

// Caller owns changes.mu before database access/schema publication. Only the
// exact constructors already attested by discardCatalogIntents can reach here.
// A surviving table keeps all its headless allocator/column/checked inputs.
func (changes *publicCatalogChanges) retireCancelledCatalogInputs(tx *TxContext, cancelled []*table) {
	if changes.owner != tx.transactionState || changes.db == nil {
		panic("private catalog input retirement lost its declaration owner")
	}
	for _, t := range cancelled {
		if lease := changes.leases[t]; lease != nil {
			owner := &t.writeAdmission
			owner.mu.Lock()
			valid := lease.owner == owner && lease.transaction == changes.owner && lease.declaration && owner.declaration == changes.owner &&
				(lease.ddlTable.Load() == nil || lease.ddlTable.Load() == t)
			owner.mu.Unlock()
			if !valid {
				panic("private catalog input retirement lost its original declaration lease")
			}
		}
		// These public program slots have independent reservation lifetimes.
		// Their existing profiles cannot target private constructors; never
		// discard one silently if an inconsistent caller supplies that program.
		for _, slot := range changes.statementTriggers {
			if slot.table == t {
				panic("private catalog retirement retains a statement trigger endpoint")
			}
		}
		for _, state := range changes.constraintStates {
			if slices.Contains(state.endpoints(), t) {
				panic("private catalog retirement retains a constraint endpoint")
			}
		}
	}
	views := changes.views.Load()
	if views == nil {
		panic("private catalog input retirement lost its immutable declaration view")
	}
	tables := maps.Clone(views.tables)
	for _, t := range cancelled {
		if lease := changes.leases[t]; lease != nil {
			if changes.anchor == lease {
				changes.anchor = nil
			}
			_ = lease.Close()
			delete(changes.leases, t)
		}
		delete(changes.baselines, t)
		delete(tables, t)
	}
	if changes.anchor == nil {
		for _, lease := range changes.leases {
			if lease == nil {
				continue
			}
			lease.attachRetainedMemory(changes.memory)
			changes.anchor = lease
			break
		}
	}
	// Reuse the existing immutable summary owner for every column, key,
	// namespace and checked-writer field. It drops allocator headless slots
	// only when their actual endpoint lease has just been retired above.
	changes.publishViews(tx.publicForeignAmendments(), tables)
	changes.refreshMemoryLocked()
}
