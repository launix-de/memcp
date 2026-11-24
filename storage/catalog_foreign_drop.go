/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"
import "sort"
import "slices"
import "unsafe"
import "context"
import "strings"
import "github.com/launix-de/memcp/scm"

// Retirement owns the exact committed predecessor at both native endpoints.
// The existing transaction history retains private metadata and reservations;
// no live declaration or enforcement program changes before acceptance.
type catalogForeignDropAmendment struct {
	declaration   catalogForeignDeclaration
	endpoints     [2]*table
	previous      [2]*tableShowColumnsSnapshot
	programs      []foreignKeyTrigger
	reservations  []foreignKeyTrigger
	addition      *catalogForeignAmendment
	previousState *catalogConstraintState
	retainedBytes uint
	retiredBytes  uint
}

func (tx *TxContext) dropCatalogForeign(child *table, name string) bool {
	if tx == nil || tx.transactionState == nil || tx.State != TxActive || tx.decisionStarted {
		panic("foreign removal requires an active native transaction")
	}
	child.schema.GetRead(tx).Release()
	if !child.mayHaveForeignDeclaration(name, tx) {
		return false
	}
	finish := tx.beginCatalogConstruction(catalogDeclarationConstruction, nil, child)
	defer func() { finish(); tx.retireUnacceptedPublicCatalogPhase() }()
	if tx.coordinator != nil || tx.clusterNumber.Load() != 0 || tx.remoteObject != nil || tx.remoteInvocation || tx.catalogIntents.Load() != nil || tx.hasTableDrops() || child.Ephemeral || child.ScopedSchema {
		panic("foreign removal requires a local public declaration transaction")
	}
	return tx.stagePublicCatalogDeclaration(child.schema, catalogForeignDropKind, nil, func(changes *publicCatalogChanges) bool {
		return changes.stageForeignDrop(tx, child, name)
	})
}

func (changes *publicCatalogChanges) stageForeignDrop(tx *TxContext, child *table, name string) bool {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	db := changes.db
	var declaration catalogForeignDeclaration
	var parent *table
	var addition *catalogForeignAmendment
	func() {
		access := db.GetRead(tx)
		defer access.Release()
		db.schemalock.RLock()
		defer db.schemalock.RUnlock()
		history := tx.publicForeignAmendments()
		for i := len(history) - 1; i >= 0; i-- {
			if pending := history[i].foreign; pending != nil && strings.EqualFold(pending.declaration.Foreign.Id, name) && pending.declaration.Foreign.Tbl1 == child.Name {
				addition = pending
				declaration = pending.declaration
				parent = db.tables.Get(declaration.Foreign.Tbl2)
				break
			}
		}
		if addition == nil {
			// Old schemas may contain equal FK names on different native pairs.
			// A child/name SQL address must select one complete predecessor;
			// never choose an arbitrary recipe from an ambiguous legacy header.
			var selected *foreignKey
			for _, fk := range child.Foreign {
				if fk.Tbl1 == child.Name && strings.EqualFold(fk.Id, name) {
					if selected != nil && !sameForeignDeclaration(*selected, fk) {
						panic("foreign removal has ambiguous native predecessors on its child")
					}
					copy := fk
					selected = &copy
				}
			}
			for _, fk := range child.Foreign {
				if fk.Tbl1 == child.Name && strings.EqualFold(fk.Id, name) {
					parent = db.tables.Get(fk.Tbl2)
					if parent == nil {
						panic("foreign removal predecessor endpoint is missing")
					}
					declaration = catalogForeignDeclaration{child.RuntimeIdentity, parent.RuntimeIdentity, fk}
					break
				}
			}
		}
		if parent == nil {
			panic("foreign removal requires an existing native declaration")
		}
		if addition == nil {
			validateCatalogForeignDrop(db, declaration, catalogForeignEndpointScope{owner: tx.transactionState})
		} else {
			validateCatalogForeign(db, declaration, tx.publicUniqueKeys(), tx.publicDeclarationViews(), catalogForeignEndpointScope{owner: tx.transactionState}, nil)
		}
	}()
	endpoints := foreignEndpointTables(child, parent)
	sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].Name < endpoints[j].Name })
	var acquired []*table
	accepted := false
	defer func() {
		if !accepted {
			for _, t := range acquired {
				if changes.anchor == changes.leases[t] {
					changes.anchor = nil
				}
				_ = changes.leases[t].Close()
				delete(changes.leases, t)
				delete(changes.baselines, t)
			}
		}
	}()
	for _, t := range endpoints {
		if changes.leases[t] != nil {
			continue
		}
		lease, err := t.writeAdmission.admit(scanQueryContext(tx, context.Background()), tx.transactionState, tableCatalogDeclarationAdmission, false)
		if err != nil {
			panic(err)
		}
		changes.leases[t] = lease
		changes.leasesHighWater = max(changes.leasesHighWater, len(changes.leases))
		acquired = append(acquired, t)
		if changes.anchor == nil {
			changes.refreshMemoryLocked()
			lease.attachRetainedMemory(changes.memory)
			changes.anchor = lease
		}
	}
	access := db.GetWrite(tx)
	defer access.Release()
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	changes.validateConstraintStateEndpointsLocked(tx)
	if addition == nil {
		validateCatalogForeignDrop(db, declaration, catalogForeignEndpointScope{owner: tx.transactionState})
	} else {
		validateCatalogForeign(db, declaration, tx.publicUniqueKeys(), tx.publicDeclarationViews(), catalogForeignEndpointScope{owner: tx.transactionState}, nil)
		if currentForeignAdditions(tx.publicForeignAmendments())[foreignIdentity(addition.declaration)] != addition {
			panic("foreign removal pending predecessor changed")
		}
	}
	amendment := &catalogForeignDropAmendment{declaration: declaration, addition: addition, previousState: changes.constraintStates[declaration.Foreign.Id]}
	amendment.retainedBytes = uint(unsafe.Sizeof(*amendment))
	views := maps.Clone(changes.views.Load().tables)
	for i, t := range endpoints {
		requireTableMaintenance(db.Name, t.Name, maintenanceAlter)
		amendment.endpoints[i], amendment.previous[i] = t, views[t]
		base := views[t]
		if base == nil {
			base = t.showColumnsSnapshot.Load()
		}
		if !foreignDropMetadataContains(t, base, declaration.Foreign) {
			panic("foreign removal predecessor changed during admission")
		}
		var bytes uint
		views[t], bytes = removeForeignDropMetadata(t, base, declaration.Foreign, false)
		amendment.retainedBytes += bytes
	}
	// Only four existing native headers are reserved. Compiling or executing
	// their callbacks is neither necessary nor permitted under publication.
	amendment.programs = foreignDropPrograms(child, parent, declaration.Foreign)
	var reserved []foreignKeyTrigger
	defer func() {
		if !accepted {
			releaseForeignTriggerPrograms(tx.transactionState, reserved)
		}
	}()
	if addition == nil {
		for _, program := range amendment.programs {
			program.table.reserveForeignDropProgram(tx.transactionState, program.trigger)
			reserved = append(reserved, program)
		}
	} else {
		for _, program := range amendment.programs {
			changes.validateForeignReservation(tx, program)
		}
	}
	amendment.reservations = reserved
	amendment.retainedBytes += uint(cap(amendment.programs)+cap(amendment.reservations)) * uint(unsafe.Sizeof(foreignKeyTrigger{}))
	for _, program := range amendment.programs {
		amendment.retainedBytes += uint(len(program.trigger.Name))
	}
	if addition == nil {
		for _, t := range endpoints {
			for _, current := range t.Foreign {
				if current.Id == declaration.Foreign.Id && current.Tbl1 == declaration.Foreign.Tbl1 && current.Tbl2 == declaration.Foreign.Tbl2 {
					amendment.retiredBytes += current.payloadFootprint
				}
			}
			for _, row := range t.showColumnsSnapshot.Load().metadata.constraints.Slice()[3].Slice() {
				if foreignMetadataValue(row, "Id").String() == declaration.Foreign.Id && foreignMetadataValue(row, "OtherTable").String() == foreignDropOther(t, declaration.Foreign) {
					amendment.retiredBytes += uint(cap(row.Slice())+cap(foreignMetadataValue(row, "LocalColumns").Slice())+cap(foreignMetadataValue(row, "OtherColumns").Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
				}
			}
		}
	}
	for _, t := range endpoints {
		if _, exists := changes.baselines[t]; !exists {
			changes.baselines[t] = foreignBaseline(t.showColumnsSnapshot.Load())
		}
	}
	changes.baselinesHighWater = max(changes.baselinesHighWater, len(changes.baselines))
	changes.viewsHighWater = max(changes.viewsHighWater, len(views))
	history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{foreignDrop: amendment})
	changes.amendmentBytes += amendment.retainedBytes
	changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	delete(changes.constraintStates, declaration.Foreign.Id)
	tx.catalogAmendments.Store(&history)
	changes.publishViews(history, views)
	accepted = true
	return true
}

func foreignDropMetadataContains(t *table, snapshot *tableShowColumnsSnapshot, fk foreignKey) bool {
	for _, row := range snapshot.nativeConstraints().Slice()[3].Slice() {
		if foreignMetadataValue(row, "Id").String() == fk.Id && foreignMetadataValue(row, "OtherTable").String() == foreignDropOther(t, fk) {
			return true
		}
	}
	return false
}

func foreignDropOther(t *table, fk foreignKey) string {
	if t.Name == fk.Tbl1 {
		return fk.Tbl2
	}
	return fk.Tbl1
}

func removeForeignDropMetadata(t *table, base *tableShowColumnsSnapshot, fk foreignKey, published bool) (*tableShowColumnsSnapshot, uint) {
	var result tableShowColumnsSnapshot
	var metadata tableShowColumnsSnapshotMetadata
	if published {
		result, metadata = *base, *base.metadata
	} else {
		metadata.columnNames, metadata.columnRows = base.metadata.columnNames, base.metadata.columnRows
	}
	result.metadata = &metadata
	constraints := append([]scm.Scmer(nil), base.nativeConstraints().Slice()...)
	rows := make([]scm.Scmer, 0, len(constraints[3].Slice()))
	for _, row := range constraints[3].Slice() {
		if foreignMetadataValue(row, "Id").String() != fk.Id || foreignMetadataValue(row, "OtherTable").String() != foreignDropOther(t, fk) {
			rows = append(rows, row)
		}
	}
	constraints[3] = scm.NewSlice(rows)
	metadata.constraints = scm.NewSlice(constraints)
	metadata.storageConstraints = metadata.constraints
	metadata.foreignByID = buildForeignMetadataDirectory(metadata.constraints)
	declarations := append([]scm.Scmer(nil), base.metadata.declarations.Slice()...)
	declarations[3] = metadata.constraints
	metadata.declarations = scm.NewSlice(declarations)
	return &result, foreignMetadataContainers(&result, nil, rows, declarations)
}

func foreignDropPrograms(child, parent *table, fk foreignKey) []foreignKeyTrigger {
	prefix := "__fk_" + fk.Id + "_"
	update := BeforeUpdate
	if fk.Updatemode != RESTRICT {
		update = AfterUpdate
	}
	return []foreignKeyTrigger{
		{child, TriggerDescription{Name: prefix + "child_insert", Timing: BeforeInsert, IsSystem: true}},
		{child, TriggerDescription{Name: prefix + "child_update", Timing: BeforeUpdate, IsSystem: true}},
		{parent, TriggerDescription{Name: prefix + "parent_delete", Timing: BeforeDelete, IsSystem: true}},
		{parent, TriggerDescription{Name: prefix + "parent_update", Timing: update, IsSystem: true}},
	}
}

func (t *table) reserveForeignDropProgram(transaction *transactionState, expected TriggerDescription) {
	t.mu.Lock()
	defer t.mu.Unlock()
	owner := &t.writeAdmission
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.declaration != transaction || transaction == nil {
		panic("foreign removal lost its native declaration owner")
	}
	if _, exists := owner.triggerNames[expected.Name]; exists {
		panic("foreign removal trigger name already reserved: " + expected.Name)
	}
	found := false
	for _, trigger := range t.Triggers {
		if trigger.Name == expected.Name {
			if !trigger.IsSystem || trigger.Timing != expected.Timing {
				panic("foreign removal predecessor trigger changed")
			}
			found = true
		}
	}
	if !found {
		panic("foreign removal predecessor trigger is missing")
	}
	if owner.triggerNames == nil {
		owner.triggerNames = make(map[string]struct{})
	}
	owner.triggerNames[expected.Name] = struct{}{}
	owner.triggerNamesHighWater = max(owner.triggerNamesHighWater, len(owner.triggerNames))
}

func validateCatalogForeignDrop(db *database, declaration catalogForeignDeclaration, scope catalogForeignEndpointScope) {
	validateCatalogForeign(db, declaration, nil, nil, scope, nil)
	fk := declaration.Foreign
	for _, t := range foreignEndpointTables(db.tables.Get(fk.Tbl1), db.tables.Get(fk.Tbl2)) {
		matched := false
		for _, current := range t.Foreign {
			if current.Id == fk.Id && current.Tbl1 == fk.Tbl1 && current.Tbl2 == fk.Tbl2 &&
				current.Updatemode == fk.Updatemode && current.Deletemode == fk.Deletemode && current.Untrusted == fk.Untrusted &&
				slices.Equal(current.Cols1, fk.Cols1) && slices.Equal(current.Cols2, fk.Cols2) {
				matched = true
			}
		}
		if !matched {
			panic("foreign removal predecessor declaration changed")
		}
	}
}

func (tx *TxContext) validatePublicForeignDrops(db *database) error {
	for _, entry := range tx.publicForeignAmendments() {
		if entry.foreignDrop == nil {
			continue
		}
		if pending := entry.foreignDrop.addition; pending != nil {
			if !sameCatalogForeign(pending.declaration, entry.foreignDrop.declaration) {
				return fmt.Errorf("foreign removal pending recipe changed")
			}
			for _, program := range entry.foreignDrop.programs {
				tx.publicCatalogChanges().validateForeignReservation(tx, program)
			}
			continue
		}
		validateCatalogForeignDrop(db, entry.foreignDrop.declaration, catalogForeignEndpointScope{owner: tx.transactionState})
		for _, program := range entry.foreignDrop.programs {
			t := program.table
			t.mu.Lock()
			owner := &t.writeAdmission
			owner.mu.Lock()
			_, reserved := owner.triggerNames[program.trigger.Name]
			owned := owner.declaration == tx.transactionState
			owner.mu.Unlock()
			found := false
			for _, trigger := range t.Triggers {
				if trigger.Name == program.trigger.Name && trigger.IsSystem && trigger.Timing == program.trigger.Timing {
					found = true
				}
			}
			t.mu.Unlock()
			if !owned || !reserved || !found {
				return fmt.Errorf("foreign removal lost its native predecessor reservation")
			}
		}
	}
	return nil
}

func (db *database) recoverPublicForeignDrops(declarations []catalogForeignDeclaration, scope catalogForeignEndpointScope) {
	for _, declaration := range declarations {
		validateCatalogForeignDrop(db, declaration, scope)
		fk := declaration.Foreign
		child, parent := db.tables.Get(fk.Tbl1), db.tables.Get(fk.Tbl2)
		removeFKTriggers(child, parent, fk)
		for _, t := range foreignEndpointTables(child, parent) {
			t.Foreign = slices.DeleteFunc(t.Foreign, func(current foreignKey) bool {
				return current.Id == fk.Id && current.Tbl1 == fk.Tbl1 && current.Tbl2 == fk.Tbl2
			})
			t.refreshForeignMemoryLocked()
		}
	}
}
