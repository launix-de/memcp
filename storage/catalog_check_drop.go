/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"
import "slices"
import "unsafe"
import "context"
import "strings"
import "github.com/launix-de/memcp/scm"

// Removal retains the exact inert predecessor recipe and native incarnation.
// It never serializes a compiled program or reinterprets an ADD-only envelope.
type catalogCheckDropAmendment struct {
	declaration   catalogCheckDeclaration
	table         *table
	previous      *tableShowColumnsSnapshot
	retainedBytes uint
	retiredBytes  uint
}

func checkMetadataContains(snapshot *tableShowColumnsSnapshot, name string) bool {
	constraints := snapshot.metadata.constraints.Slice()
	if len(constraints) == 6 {
		for _, row := range constraints[5].Slice() {
			if strings.EqualFold(foreignMetadataValue(row, "Id").String(), name) {
				return true
			}
		}
	}
	return false
}

func (tx *TxContext) dropCatalogCheck(t *table, name string) bool {
	if tx == nil || tx.transactionState == nil || tx.State != TxActive || tx.decisionStarted {
		panic("check removal requires an active native transaction")
	}
	// A non-CHECK target is a read-only miss, before entering the declaration
	// phase. The frontend may then dispatch another actual constraint kind.
	t.schema.GetRead(tx)
	if !checkMetadataContains(t.declarationMetadataForTx(tx), name) {
		return false
	}
	finish := tx.beginCatalogConstruction(catalogEnforcementConstruction, t)
	defer finish()
	if tx.coordinator != nil || tx.catalogIntents.Load() != nil || t.Ephemeral {
		panic("check removal requires a local public declaration transaction")
	}
	return tx.stagePublicCatalogDeclaration(t.schema, catalogEnforcementKind, func(changes *publicCatalogChanges) bool {
		return changes.stageCheckDrop(tx, t, name)
	})
}

func (changes *publicCatalogChanges) stageCheckDrop(tx *TxContext, t *table, name string) bool {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	accepted := false
	acquired := changes.leases[t] == nil
	if acquired {
		lease, err := t.writeAdmission.admit(scanQueryContext(tx, context.Background()), tx.transactionState, tableCatalogDeclarationAdmission, false)
		if err != nil {
			panic(err)
		}
		changes.leases[t] = lease
		changes.leasesHighWater = max(changes.leasesHighWater, len(changes.leases))
		if changes.anchor == nil {
			changes.refreshMemoryLocked()
			lease.attachRetainedMemory(changes.memory)
			changes.anchor = lease
		}
	}
	defer func() {
		if !accepted && acquired {
			if changes.anchor == changes.leases[t] {
				changes.anchor = nil
			}
			_ = changes.leases[t].Close()
			delete(changes.leases, t)
			delete(changes.baselines, t)
		}
	}()
	db := changes.db
	access := db.GetWrite(tx)
	defer access.Release()
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	changes.validateConstraintStateEndpointsLocked(tx)
	requireTableMaintenance(db.Name, t.Name, maintenanceAlter)
	if t.schema != db || db.tables.Get(t.Name) != t || !t.committedAt(db.catalogView().version) {
		panic("check removal endpoint generation was replaced")
	}
	observed := t.showColumnsSnapshot.Load()
	views := maps.Clone(changes.views.Load().tables)
	previous := views[t]
	base := previous
	if base == nil {
		base = observed
	}
	if !checkMetadataContains(base, name) {
		panic("check declaration changed during removal admission")
	}
	var definition *checkDeclaration
	for _, check := range t.Checks {
		if strings.EqualFold(check.Id, name) {
			definition = check
			break
		}
	}
	if definition == nil {
		panic("check removal requires an existing published declaration")
	}
	// Reserve precisely the existing two native CHECK headers. This is not
	// permission to replace an arbitrary system or user trigger by its name.
	t.reserveCheckDrop(tx.transactionState, definition)
	defer func() {
		if !accepted {
			t.releaseDeclarationTrigger(tx.transactionState, definition.triggerName(BeforeInsert))
			t.releaseDeclarationTrigger(tx.transactionState, definition.triggerName(BeforeUpdate))
		}
	}()
	view, bytes := removeCheckMetadata(base, definition.Id, false)
	amendment := &catalogCheckDropAmendment{
		declaration: catalogCheckDeclaration{Table: t.Name, Identity: t.RuntimeIdentity, Check: definition},
		table:       t, previous: previous,
	}
	amendment.retainedBytes = uint(unsafe.Sizeof(*amendment)) + bytes
	amendment.retiredBytes = definition.payloadFootprint
	for _, row := range observed.metadata.constraints.Slice()[5].Slice() {
		if foreignMetadataValue(row, "Id").String() == definition.Id {
			amendment.retiredBytes += uint(cap(row.Slice())+cap(foreignMetadataValue(row, "Columns").Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
		}
	}
	history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{checkDrop: amendment})
	if _, exists := changes.baselines[t]; !exists {
		changes.baselines[t] = foreignBaseline(observed)
	}
	changes.baselinesHighWater = max(changes.baselinesHighWater, len(changes.baselines))
	views[t] = view
	changes.viewsHighWater = max(changes.viewsHighWater, len(views))
	changes.amendmentBytes += amendment.retainedBytes
	changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	tx.catalogAmendments.Store(&history)
	changes.publishViews(history, views)
	accepted = true
	return true
}

func removeCheckMetadata(base *tableShowColumnsSnapshot, name string, published bool) (*tableShowColumnsSnapshot, uint) {
	var result tableShowColumnsSnapshot
	var metadata tableShowColumnsSnapshotMetadata
	if published {
		result, metadata = *base, *base.metadata
	} else {
		metadata.columnNames, metadata.columnRows = base.metadata.columnNames, base.metadata.columnRows
	}
	result.metadata = &metadata
	old := base.metadata.constraints.Slice()
	constraints := append([]scm.Scmer(nil), old...)
	rows := make([]scm.Scmer, 0, len(old[5].Slice()))
	for _, row := range old[5].Slice() {
		if foreignMetadataValue(row, "Id").String() != name {
			rows = append(rows, row)
		}
	}
	constraints[5] = scm.NewSlice(rows)
	metadata.constraints = scm.NewSlice(constraints)
	metadata.foreignByID = buildForeignMetadataDirectory(metadata.constraints)
	declarations := append([]scm.Scmer(nil), base.metadata.declarations.Slice()...)
	declarations[3] = metadata.constraints
	metadata.declarations = scm.NewSlice(declarations)
	bytes := foreignMetadataContainers(&result, nil, nil, declarations)
	bytes += uint(cap(rows)) * uint(unsafe.Sizeof(scm.Scmer{}))
	return &result, bytes
}

func (t *table) reserveCheckDrop(transaction *transactionState, definition *checkDeclaration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	owner := &t.writeAdmission
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.declaration != transaction || transaction == nil {
		panic("check removal lost its native declaration owner")
	}
	for _, timing := range []TriggerTiming{BeforeInsert, BeforeUpdate} {
		name := definition.triggerName(timing)
		if _, exists := owner.triggerNames[name]; exists {
			panic("check removal trigger name already reserved: " + name)
		}
		found := false
		for _, trigger := range t.Triggers {
			if trigger.Name == name {
				if !trigger.IsSystem || trigger.Timing != timing || trigger.checkDefinition != definition {
					panic("check removal lost its actual native trigger: " + name)
				}
				found = true
			}
		}
		if !found {
			panic("check removal trigger is missing: " + name)
		}
	}
	if owner.triggerNames == nil {
		owner.triggerNames = make(map[string]struct{})
	}
	owner.triggerNames[definition.triggerName(BeforeInsert)] = struct{}{}
	owner.triggerNames[definition.triggerName(BeforeUpdate)] = struct{}{}
	owner.triggerNamesHighWater = max(owner.triggerNamesHighWater, len(owner.triggerNames))
}

func (tx *TxContext) publicCheckDrops() []catalogCheckDeclaration {
	var result []catalogCheckDeclaration
	for _, entry := range tx.publicForeignAmendments() {
		if entry.checkDrop != nil {
			result = append(result, entry.checkDrop.declaration)
		}
	}
	return result
}

func validateCatalogCheckDrop(db *database, declaration catalogCheckDeclaration) (*table, *checkDeclaration) {
	validateCatalogCheck(db, declaration)
	t := db.tables.Get(declaration.Table)
	for _, definition := range t.Checks {
		if definition.Id == declaration.Check.Id && definition.Language == declaration.Check.Language &&
			definition.Source == declaration.Check.Source && slices.Equal(definition.Columns, declaration.Check.Columns) {
			return t, definition
		}
	}
	panic("check removal predecessor declaration changed")
}

func (tx *TxContext) validatePublicCheckDrops(db *database) error {
	for _, declaration := range tx.publicCheckDrops() {
		t, definition := validateCatalogCheckDrop(db, declaration)
		t.mu.Lock()
		for _, timing := range []TriggerTiming{BeforeInsert, BeforeUpdate} {
			name := definition.triggerName(timing)
			owner := &t.writeAdmission
			owner.mu.Lock()
			_, reserved := owner.triggerNames[name]
			owned := owner.declaration == tx.transactionState
			owner.mu.Unlock()
			if !reserved || !owned {
				t.mu.Unlock()
				return fmt.Errorf("check removal lost its native trigger reservation")
			}
			found := false
			for _, trigger := range t.Triggers {
				found = found || trigger.Name == name && trigger.IsSystem && trigger.Timing == timing && trigger.checkDefinition == definition
			}
			if !found {
				t.mu.Unlock()
				return fmt.Errorf("check removal predecessor trigger changed")
			}
		}
		t.mu.Unlock()
	}
	return nil
}

// Accepted native publication/recovery already owns schema and endpoint
// admission. Retire only headers belonging to this exact native recipe.
func (t *table) removeCheckHeaders(definition *checkDeclaration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	kept := t.Triggers[:0]
	for _, trigger := range t.Triggers {
		if trigger.checkDefinition != definition && !(trigger.checkDefinition == nil && inertCheckTrigger(definition, trigger)) {
			kept = append(kept, trigger)
		}
	}
	clear(t.Triggers[len(kept):])
	t.Triggers = kept
	t.refreshTriggerMemoryLocked()
}

func (db *database) recoverPublicCheckDrops(declarations []catalogCheckDeclaration) {
	for _, declaration := range declarations {
		t, definition := validateCatalogCheckDrop(db, declaration)
		t.Checks = slices.DeleteFunc(t.Checks, func(check *checkDeclaration) bool { return check == definition })
		t.removeCheckHeaders(definition)
		t.refreshCheckMemoryLocked()
	}
}
