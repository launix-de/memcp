/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"unsafe"

	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
)

type catalogCheckDeclaration struct {
	Table    string            `json:"table"`
	Identity string            `json:"identity"`
	Check    *checkDeclaration `json:"check"`
}

type catalogCheckAmendment struct {
	declaration   catalogCheckDeclaration
	table         *table
	previous      *tableShowColumnsSnapshot
	programs      []foreignKeyTrigger
	retainedBytes uint
}

func (tx *TxContext) stageCatalogCheck(t *table, definition *checkDeclaration) bool {
	if tx == nil || tx.transactionState == nil {
		panic("check declaration requires an active native transaction")
	}
	requireCheckInvocation(tx)
	finish := tx.beginCatalogConstruction(true)
	defer finish()
	if tx.State != TxActive || tx.decisionStarted || tx.coordinator != nil || tx.catalogIntents.Load() != nil {
		panic("check declaration requires a local public metadata transaction")
	}
	return tx.stagePublicCatalogDeclaration(t.schema, func(changes *publicCatalogChanges) bool {
		return changes.stageCheck(tx, t, definition)
	})
}

func (changes *publicCatalogChanges) stageCheck(tx *TxContext, t *table, definition *checkDeclaration) bool {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	accepted := false
	acquired := changes.leases[t] == nil
	if acquired {
		lease, err := t.writeAdmission.admit(scanQueryContext(tx, context.Background()), tx.transactionState, true, false)
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
	var declaration catalogCheckDeclaration
	var observed *tableShowColumnsSnapshot
	func() {
		access := db.GetWrite(tx)
		defer access.Release()
		db.schemalock.RLock()
		defer db.schemalock.RUnlock()
		declaration = catalogCheckDeclaration{Table: t.Name, Identity: t.RuntimeIdentity, Check: definition}
		changes.validateCheckStageLocked(tx, t, declaration)
		observed = t.showColumnsSnapshot.Load()
	}()
	// The exclusive native declaration lease holds the actual dependency
	// contract stable. Compile/scan/measure privately, outside schema/shard locks.
	programs := buildCheckTriggers(t, definition)
	validateExistingCheckRows(tx, t, definition, programs[0].trigger.Func)
	release := reserveForeignTriggerPrograms(tx.transactionState, programs)
	defer func() {
		if !accepted {
			release()
		}
	}()
	views := maps.Clone(changes.views.Load().tables)
	previous := views[t]
	base := previous
	if base == nil {
		base = observed
	}
	view, bytes := appendConstraintMetadata(t, base, nil, []*checkDeclaration{definition}, 0)
	amendment := &catalogCheckAmendment{declaration: declaration, table: t, previous: previous, programs: programs}
	amendment.retainedBytes = uint(unsafe.Sizeof(*amendment)) + definition.payloadFootprint + bytes + uint(unsafe.Sizeof(catalogConstraintState{}))
	amendment.retainedBytes += uint(cap(programs)) * uint(unsafe.Sizeof(foreignKeyTrigger{}))
	for _, program := range programs {
		amendment.retainedBytes += constraintProgramBytes(program.trigger)
	}
	history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{check: amendment})
	access := db.GetWrite(tx)
	defer access.Release()
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	changes.validateCheckStageLocked(tx, t, declaration)
	if t.showColumnsSnapshot.Load().metadata.declarations != observed.metadata.declarations {
		panic("public declaration conflict: check input changed during validation")
	}
	if _, exists := changes.baselines[t]; !exists {
		changes.baselines[t] = foreignBaseline(observed)
	}
	changes.baselinesHighWater = max(changes.baselinesHighWater, len(changes.baselines))
	views[t] = view
	changes.viewsHighWater = max(changes.viewsHighWater, len(views))
	if changes.constraintStates == nil {
		changes.constraintStates = make(map[string]*catalogConstraintState)
	}
	changes.constraintStates[definition.Id] = &catalogConstraintState{check: amendment, enabled: true, trusted: true}
	changes.constraintStatesHighWater = max(changes.constraintStatesHighWater, len(changes.constraintStates))
	changes.amendmentBytes += amendment.retainedBytes
	changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	tx.catalogAmendments.Store(&history)
	changes.publishViews(history, views)
	accepted = true
	return true
}

// Caller owns schema publication and this transaction's amendment producer.
func (changes *publicCatalogChanges) validateCheckStageLocked(tx *TxContext, t *table, declaration catalogCheckDeclaration) {
	changes.validateConstraintStateEndpointsLocked(tx)
	validateCatalogCheck(changes.db, declaration)
	requireTableMaintenance(changes.db.Name, declaration.Table, maintenanceAlter)
	if changes.db.hasDeclarationNameLocked(declaration.Check.Id, nil) {
		panic("declaration name already exists: " + declaration.Check.Id)
	}
	for _, entry := range tx.publicForeignAmendments() {
		if entry.foreign != nil && strings.EqualFold(entry.foreign.declaration.Foreign.Id, declaration.Check.Id) ||
			entry.check != nil && strings.EqualFold(entry.check.declaration.Check.Id, declaration.Check.Id) {
			panic("declaration name already exists: " + declaration.Check.Id)
		}
	}
}

func validateCatalogCheck(db *database, declaration catalogCheckDeclaration) {
	if _, err := uuid.Parse(declaration.Identity); err != nil {
		panic("invalid check endpoint identity")
	}
	t := db.tables.Get(declaration.Table)
	if t == nil || t.RuntimeIdentity != declaration.Identity || !t.committedAt(db.catalogView().version) {
		panic("check declaration endpoint generation was replaced")
	}
	validateCheckColumns(t, declaration.Check)
}

func validateExistingCheckRows(tx *TxContext, t *table, definition *checkDeclaration, predicate scm.Scmer) {
	requireCheckInvocation(tx)
	validate := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		row := t.rowToDictWithColumns(dataset(values[1:]), definition.Columns)
		scm.Apply(predicate, scm.NewNil(), row, txSessionScmer(tx), txContextScmer(tx))
		return scm.NewBool(true)
	})
	t.scan(tx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil,
		scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), definition.Columns, validate, scm.NewBool(true),
		scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), false)
}

func (tx *TxContext) publicCheckDeclarations() []catalogCheckDeclaration {
	var result []catalogCheckDeclaration
	for _, entry := range tx.publicForeignAmendments() {
		if entry.check != nil {
			result = append(result, entry.check.declaration)
		}
	}
	return result
}

// The decoded durable envelope and actual live catalog own distinct allocations.
func (db *database) recoverPublicCheckDeclarations(declarations []catalogCheckDeclaration) {
	for _, declaration := range declarations {
		validateCatalogCheck(db, declaration)
		if db.hasDeclarationNameLocked(declaration.Check.Id, nil) {
			panic("check recovery declaration name already exists")
		}
		t := db.tables.Get(declaration.Table)
		definition := declaration.Check
		t.Checks = append(t.Checks, ownCheckDeclaration(definition.Id, definition.Language, definition.Source, definition.Columns))
		t.refreshCheckMemoryLocked()
	}
}

func constraintProgramBytes(trigger TriggerDescription) uint {
	if trigger.initialFKFootprint == nil {
		return 0
	}
	return trigger.initialFKFootprint.bytes
}

func (state *catalogConstraintState) name() string {
	if state.addition != nil {
		return state.addition.declaration.Foreign.Id
	}
	return state.check.declaration.Check.Id
}

func (state *catalogConstraintState) owns(t *table) bool {
	if state.addition != nil {
		return state.addition.declaration.ChildIdentity == t.RuntimeIdentity && state.addition.declaration.Foreign.Tbl1 == t.Name
	}
	return state.check != nil && state.check.table == t && state.check.declaration.Identity == t.RuntimeIdentity
}

func (state *catalogConstraintState) endpoints() []*table {
	if state.addition != nil {
		return state.addition.endpoints[:]
	}
	return []*table{state.check.table}
}

func (state *catalogConstraintState) validate(db *database) {
	if state.addition != nil {
		validateCatalogForeign(db, state.addition.declaration)
	} else {
		validateCatalogCheck(db, state.check.declaration)
	}
}

func (tx *TxContext) validatePublicChecks(db *database) error {
	for _, entry := range tx.publicForeignAmendments() {
		if entry.check == nil {
			continue
		}
		validateCatalogCheck(db, entry.check.declaration)
		if db.hasDeclarationNameLocked(entry.check.declaration.Check.Id, nil) {
			return fmt.Errorf("public declaration conflict: check name already exists")
		}
		for _, program := range entry.check.programs {
			program.table.validateDeclarationTrigger(tx.transactionState, program.trigger.Name)
		}
	}
	return nil
}
