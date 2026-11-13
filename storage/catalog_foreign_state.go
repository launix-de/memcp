/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"fmt"
	"maps"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

// Only a not-yet-published addition can change state in this profile. Public row
// writes remain forbidden by the existing declaration phase, and preparation
// rejects any final disabled/unvalidated state. No disabled durable format or
// runtime program bypass is implied by these transient declaration facts.
type catalogConstraintState struct {
	addition *catalogForeignAmendment
	check    *catalogCheckAmendment
	enabled  bool
	trusted  bool
}

type catalogConstraintStateView struct {
	table    *table
	previous *tableShowColumnsSnapshot
}

type catalogConstraintStateAmendment struct {
	previous      []*catalogConstraintState
	current       []*catalogConstraintState
	views         []catalogConstraintStateView
	retainedBytes uint
}

func (tx *TxContext) stageConstraintState(child *table, names []string, all, enabled, validate bool) bool {
	if tx == nil || tx.transactionState == nil {
		panic("foreign state requires its active native declaration transaction")
	}
	finish := tx.beginCatalogConstruction(true)
	defer finish()
	if tx.State != TxActive || tx.decisionStarted || validate && !enabled {
		panic("invalid native foreign state request")
	}
	changes := tx.publicCatalogChanges()
	if changes == nil || changes.owner != tx.transactionState || changes.db != child.schema || tx.coordinator != nil {
		panic("foreign state requires an owned pending declaration")
	}
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	db := changes.db
	var targets []*catalogConstraintState
	var lookups []*foreignKeyLookup
	func() {
		access := db.GetWrite(tx)
		defer access.Release()
		db.schemalock.RLock()
		defer db.schemalock.RUnlock()
		requireTableMaintenance(db.Name, child.Name, maintenanceAlter)
		if GetDatabase(db.Name) != db || activeReplication(db) != nil || db.tables.Get(child.Name) != child {
			panic("foreign state lost its native endpoint owner")
		}
		// Resolve ALL against the complete actual child-owned native declaration
		// directory, including committed rows. Incoming parent mirrors are not owned
		// constraints. A committed/unknown target rejects the WHOLE operation.
		if all {
			snapshot := child.declarationMetadataForTx(tx)
			constraints := snapshot.metadata.constraints.Slice()
			if (len(constraints) != 4 && len(constraints) != 6) || constraints[0].String() != "Unique" || constraints[2].String() != "ForeignKeys" || len(constraints) == 6 && constraints[4].String() != "Checks" {
				panic("foreign state ALL requires a complete supported constraint directory")
			}
			for _, row := range constraints[3].Slice() {
				if foreignMetadataValue(row, "Role").String() == "referencing" {
					names = append(names, foreignMetadataValue(row, "Id").String())
				}
			}
			if len(constraints) == 6 {
				for _, row := range constraints[5].Slice() {
					names = append(names, foreignMetadataValue(row, "Id").String())
				}
			}
		}
		seen := make(map[string]bool, len(names))
		for _, name := range names {
			pending := changes.constraintStates[name]
			if seen[name] || pending == nil || !pending.owns(child) {
				panic("foreign state target is not an owned pending declaration: " + name)
			}
			seen[name] = true
			targets = append(targets, pending)
		}
		changes.validateConstraintStateEndpointsLocked(tx)
		for _, target := range targets {
			target.validate(db)
			var lookup *foreignKeyLookup
			if target.addition != nil {
				declaration := target.addition.declaration
				lookup = newForeignKeyLookup(db.tables.Get(declaration.Foreign.Tbl2), declaration.Foreign.Cols2)
			}
			lookups = append(lookups, lookup)
		}
	}()
	// Endpoint declaration leases already exclude DML/maintenance through Close.
	// The existing streaming scan releases its physical reader lock before this
	// nonmutating callback; no SQL query, row bag or secondary executor is used.
	if validate {
		for i, target := range targets {
			if target.addition != nil {
				validateExistingForeignRows(tx, child, target.addition.declaration.Foreign, lookups[i])
			} else {
				validateExistingCheckRows(tx, child, target.check.declaration.Check, target.check.programs[0].trigger.Func)
			}
		}
	}
	if len(targets) == 0 {
		return true
	}
	amendment := &catalogConstraintStateAmendment{}
	updates := make(map[string]*catalogConstraintState, len(targets))
	endpoints := make(map[*table]bool)
	for _, before := range targets {
		current := &catalogConstraintState{addition: before.addition, check: before.check, enabled: enabled, trusted: before.trusted && enabled}
		if !enabled {
			current.trusted = false
		}
		if validate {
			current.trusted = true
		}
		amendment.previous = append(amendment.previous, before)
		amendment.current = append(amendment.current, current)
		updates[before.name()] = current
		for _, table := range before.endpoints() {
			if table != nil {
				endpoints[table] = true
			}
		}
	}
	views := maps.Clone(changes.views.Load().tables)
	amendment.retainedBytes = uint(unsafe.Sizeof(*amendment))
	for table := range endpoints {
		previous := views[table]
		if previous == nil {
			panic("pending foreign state lost its private endpoint view")
		}
		amendment.views = append(amendment.views, catalogConstraintStateView{table: table, previous: previous})
		var bytes uint
		views[table], bytes = replaceConstraintStateMetadata(previous, updates)
		amendment.retainedBytes += bytes
	}
	amendment.retainedBytes += uint(cap(amendment.previous)+cap(amendment.current)) * uint(unsafe.Sizeof((*catalogConstraintState)(nil)))
	amendment.retainedBytes += uint(len(amendment.current)) * uint(unsafe.Sizeof(catalogConstraintState{}))
	amendment.retainedBytes += uint(cap(amendment.views)) * uint(unsafe.Sizeof(catalogConstraintStateView{}))
	history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{constraintState: amendment})
	access := db.GetWrite(tx)
	defer access.Release()
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	changes.validateConstraintStateEndpointsLocked(tx)
	for _, target := range targets {
		target.validate(db)
	}
	// The entire target set and all validations completed before this sole private
	// publication. Construction admission excludes transaction decision/rewind.
	for name, state := range updates {
		changes.constraintStates[name] = state
	}
	changes.amendmentBytes += amendment.retainedBytes
	changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	tx.catalogAmendments.Store(&history)
	changes.publishViews(history, views)
	return true
}

// Caller owns the existing schema publication lock; endpoint declaration leases
// remain held by this same resource across the outside-lock historical scan.
func (changes *publicCatalogChanges) validateConstraintStateEndpointsLocked(tx *TxContext) {
	db := changes.db
	identity := tx.catalogIdentity.Load()
	if GetDatabase(db.Name) != db || identity == nil || identity.database != db || identity.Incarnation != db.incarnationID() || activeReplication(db) != nil {
		panic("foreign state lost its native catalog owner")
	}
	for table, before := range changes.baselines {
		if db.tables.Get(table.Name) != table || table.showColumnsSnapshot.Load().metadata.declarations != before.declarations {
			panic("public declaration conflict: endpoint declaration changed")
		}
	}
}

func replaceConstraintStateMetadata(base *tableShowColumnsSnapshot, updates map[string]*catalogConstraintState) (*tableShowColumnsSnapshot, uint) {
	result := &tableShowColumnsSnapshot{metadata: &tableShowColumnsSnapshotMetadata{}}
	result.metadata.columnNames = base.metadata.columnNames
	result.metadata.columnRows = base.metadata.columnRows
	old := base.metadata.constraints.Slice()
	constraints := append([]scm.Scmer(nil), old...)
	var ownedRows uint
	var foreignRows []scm.Scmer
	var checkRowsBytes uint
	for index := 2; index < len(constraints); index += 2 {
		rows := append([]scm.Scmer(nil), old[index+1].Slice()...)
		for i, row := range rows {
			state := updates[foreignMetadataValue(row, "Id").String()]
			if state == nil {
				continue
			}
			fields := append([]scm.Scmer(nil), row.Slice()...)
			for j := 0; j < len(fields); j += 2 {
				switch fields[j].String() {
				case "Enabled":
					fields[j+1] = scm.NewBool(state.enabled)
				case "Trusted":
					fields[j+1] = scm.NewBool(state.trusted)
				}
			}
			rows[i] = scm.NewSlice(fields)
			ownedRows += uint(cap(fields)) * uint(unsafe.Sizeof(scm.Scmer{}))
		}
		constraints[index+1] = scm.NewSlice(rows)
		if index == 2 {
			foreignRows = rows
		} else {
			checkRowsBytes += uint(cap(rows)) * uint(unsafe.Sizeof(scm.Scmer{}))
		}
	}
	result.metadata.constraints = scm.NewSlice(constraints)
	result.metadata.foreignByID = buildForeignMetadataDirectory(result.metadata.constraints)
	declarations := append([]scm.Scmer(nil), base.metadata.declarations.Slice()...)
	declarations[3] = result.metadata.constraints
	result.metadata.declarations = scm.NewSlice(declarations)
	return result, ownedRows + checkRowsBytes + foreignMetadataContainers(result, nil, foreignRows, declarations)
}

func (changes *publicCatalogChanges) validateConstraintStatesLocked() error {
	for name, state := range changes.constraintStates {
		if !state.enabled || !state.trusted {
			kind := "foreign"
			if state.check != nil {
				kind = "check"
			}
			return fmt.Errorf("pending %s declaration must be enabled and validated before commit: %s", kind, name)
		}
	}
	return nil
}
