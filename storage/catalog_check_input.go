/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "strings"

// This is native input admission, not SQL validation policy. A public CHECK
// may retain the same completed original-T writer only under local ACID
// ownership. Other modes/private/distributed generations keep their old gate.
func (tx *TxContext) publicCheckRowInput(t *table) bool {
	tx.mu.Lock()
	mode := tx.Mode
	tx.mu.Unlock()
	return mode == TxACID && !tx.hasTableDrops() && localCatalogRowDatabase(t.schema) &&
		t.committedAt(t.schema.catalogView().version) && t.schema.tables.Get(t.Name) == t &&
		!t.Ephemeral && !t.ScopedSchema && t.dataOwner == nil && t.PersistencyMode != Cache
}

// Both CHECK producers use the existing validated declaration lease. The
// retained typed profile describes only the checked input boundary; raw/native
// callbacks cannot borrow it, including after a savepoint removes all deltas.
func (changes *publicCatalogChanges) acquireCheckInput(tx *TxContext, t *table, rowInput bool) func(bool) {
	finishLease := changes.acquireStoredColumnLease(tx, t, rowInput)
	ready := false
	defer func() {
		if !ready {
			finishLease(false)
		}
	}()
	lease := changes.leases[t]
	previous := lease.inputProfile
	restore := func() {}
	if rowInput {
		restore = t.guardCatalogDeclarationInput(tx.transactionState)
		lease.inputProfile = nativeCheckRetirementRowInput
	}
	ready = true
	return func(accepted bool) {
		if !accepted {
			lease.inputProfile = previous
			restore()
		}
		finishLease(accepted)
	}
}

// Only this actual native lease carries the admitted CHECK binding profile.
// Metadata/generation and original writer ownership are reattested by the
// immutable view producer; active enforcement/history vetoes still dominate.
func (changes *publicCatalogChanges) retainedCheckInputTables() map[*table]struct{} {
	var result map[*table]struct{}
	for t, lease := range changes.leases {
		if lease == nil || lease.inputProfile != nativeCheckRetirementRowInput {
			continue
		}
		baseline, tracked := changes.baselines[t]
		if changes.owner == nil || !tracked || t.schema != changes.db || GetDatabase(changes.db.Name) != changes.db ||
			changes.db.tables.Get(t.Name) != t || t.RuntimeIdentity == "" ||
			t.showColumnsSnapshot.Load().metadata.declarations != baseline.declarations {
			panic("retained CHECK input lost its native generation or declaration")
		}
		owner := &t.writeAdmission
		owner.mu.Lock()
		valid := lease.owner == owner && lease.transaction == changes.owner && lease.declaration &&
			owner.declaration == changes.owner && owner.declarationGuardedInput
		owner.mu.Unlock()
		if !valid {
			panic("retained CHECK input lost its original declaration lease")
		}
		if result == nil {
			result = make(map[*table]struct{})
		}
		result[t] = struct{}{}
	}
	return result
}

func (tx *TxContext) retiredCheck(t *table, name string) *catalogCheckDropAmendment {
	for _, entry := range tx.publicForeignAmendments() {
		if drop := entry.checkDrop; drop != nil && drop.table == t && strings.EqualFold(drop.declaration.Check.Id, name) {
			return drop
		}
	}
	return nil
}
