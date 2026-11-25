/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "maps"
import "sort"
import "unsafe"
import "context"
import "strings"
import "github.com/launix-de/memcp/scm"

// A committed state change uses the existing exact DROP/ADD history. These
// private constructor inputs never become another catalog or durable format.
// Pending members retain their original state-amendment and savepoint owner.
type catalogForeignStateTarget struct {
	pending *catalogConstraintState
	before  catalogForeignDeclaration
	parent  *table
	state   *catalogConstraintState
	drop    *catalogForeignDropAmendment
	add     *catalogForeignAmendment
}

// Caller owns changes.mu and native schema access. Only child-owned entries
// participate in ALL; incoming parent mirrors never grant constraint ownership.
func constraintStateNames(snapshot *tableShowColumnsSnapshot, names []string, all bool) []string {
	if !all {
		return names
	}
	constraints := snapshot.nativeConstraints().Slice()
	if (len(constraints) != 4 && len(constraints) != 6) || constraints[0].String() != "Unique" || constraints[2].String() != "ForeignKeys" || len(constraints) == 6 && constraints[4].String() != "Checks" {
		panic("foreign state ALL requires a complete supported constraint directory")
	}
	var result []string
	for _, row := range constraints[3].Slice() {
		if foreignMetadataValue(row, "Role").String() == "referencing" {
			result = append(result, foreignMetadataValue(row, "Id").String())
		}
	}
	if len(constraints) == 6 {
		for _, row := range constraints[5].Slice() {
			result = append(result, foreignMetadataValue(row, "Id").String())
		}
	}
	return result
}

func (changes *publicCatalogChanges) constraintStateHasCommittedLocked(tx *TxContext, child *table, names []string, all bool) bool {
	access := changes.db.GetRead(tx)
	defer access.Release()
	changes.db.schemalock.RLock()
	defer changes.db.schemalock.RUnlock()
	for _, name := range constraintStateNames(child.declarationMetadataForTx(tx), names, all) {
		if state := changes.constraintStates[name]; state == nil || !state.owns(child) {
			return true
		}
	}
	return false
}

// A whole named/ALL cohort is prepared under this one native constructor.
// No DROP prefix, replacement, state view or program is published before all
// targets, historical rows, endpoint generations and reservations pass.
func (changes *publicCatalogChanges) stageCommittedForeignState(tx *TxContext, child *table, names []string, all, enabled, validate bool) bool {
	tx.mu.Lock()
	mode := tx.Mode
	tx.mu.Unlock()
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	db := changes.db
	if mode != TxACID || !localCatalogRowDatabase(db) || tx.hasTableDrops() || tx.coordinator != nil || tx.remoteObject != nil || tx.remoteInvocation || child.Ephemeral || child.ScopedSchema || child.PersistencyMode == Cache {
		panic("committed foreign state requires a local original ACID declaration owner")
	}
	var targets []*catalogForeignStateTarget
	endpoints := make(map[*table]bool)
	func() {
		access := db.GetRead(tx)
		defer access.Release()
		db.schemalock.RLock()
		defer db.schemalock.RUnlock()
		tx.requireDeclarationEndpoint(child, maintenanceAlter)
		if db.tables.Get(child.Name) != child || !child.committedAt(db.catalogView().version) {
			panic("committed foreign state lost its child generation")
		}
		snapshot := child.declarationMetadataForTx(tx)
		seen := make(map[string]bool)
		for _, name := range constraintStateNames(snapshot, names, all) {
			folded := scm.CaseFoldKey(name)
			if seen[folded] {
				panic("duplicate native constraint state target: " + name)
			}
			seen[folded] = true
			target := &catalogForeignStateTarget{pending: changes.constraintStates[name]}
			if target.pending != nil && !target.pending.owns(child) {
				panic("constraint state name belongs to another native endpoint")
			}
			if target.pending != nil && target.pending.owns(child) {
				target.pending.validate(db, tx.publicUniqueKeys(), tx.publicDeclarationViews(), catalogForeignEndpointScope{owner: tx.transactionState})
				for _, t := range target.pending.endpoints() {
					if t != nil {
						endpoints[t] = true
					}
				}
			} else {
				target.pending = nil
				for _, fk := range child.Foreign {
					if fk.Tbl1 != child.Name || !strings.EqualFold(fk.Id, name) || !foreignDropMetadataContains(child, snapshot, fk) {
						continue
					}
					parent := db.tables.Get(fk.Tbl2)
					if parent == nil || parent.Ephemeral || parent.ScopedSchema || parent.PersistencyMode == Cache || !parent.committedAt(db.catalogView().version) {
						panic("committed foreign state requires its two public endpoint generations")
					}
					declaration := catalogForeignDeclaration{child.RuntimeIdentity, parent.RuntimeIdentity, fk}
					if target.parent != nil && !sameCatalogForeign(target.before, declaration) {
						panic("committed foreign state has ambiguous native predecessors")
					}
					tx.requireDeclarationEndpoint(parent, maintenanceAlter)
					validateCatalogForeignDrop(db, declaration, catalogForeignEndpointScope{owner: tx.transactionState})
					target.before, target.parent = declaration, parent
				}
				if target.parent != nil && checkMetadataContains(snapshot, name) {
					panic("committed foreign state has an ambiguous declaration kind")
				}
				if target.parent == nil {
					if checkMetadataContains(snapshot, name) {
						panic("committed CHECK state requires an owned pending declaration")
					}
					panic("constraint state target is not an actual child-owned declaration: " + name)
				}
				endpoints[child], endpoints[target.parent] = true, true
			}
			targets = append(targets, target)
		}
	}()
	if len(targets) == 0 {
		return true
	}
	ordered := make([]*table, 0, len(endpoints))
	for t := range endpoints {
		ordered = append(ordered, t)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	var acquired []*table
	var restore []func()
	var reserved []foreignKeyTrigger
	accepted := false
	defer func() {
		if !accepted {
			releaseForeignTriggerPrograms(tx.transactionState, reserved)
			for i := len(restore) - 1; i >= 0; i-- {
				restore[i]()
			}
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
	for _, t := range ordered {
		if changes.leases[t] == nil {
			lease, err := t.writeAdmission.admit(scanQueryContext(tx, context.Background()), tx.transactionState, tableValidatedDeclarationAdmission, false)
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
		restore = append(restore, t.guardCatalogDeclarationInput(tx.transactionState))
	}
	observed := make(map[*table]*tableShowColumnsSnapshot, len(ordered))
	func() {
		access := db.GetWrite(tx)
		defer access.Release()
		db.schemalock.RLock()
		defer db.schemalock.RUnlock()
		changes.validateConstraintStateEndpointsLocked(tx)
		for _, t := range ordered {
			observed[t] = t.showColumnsSnapshot.Load()
		}
		for _, target := range targets {
			if target.pending == nil {
				validateCatalogForeignDrop(db, target.before, catalogForeignEndpointScope{owner: tx.transactionState})
			}
		}
	}()
	for _, target := range targets {
		if target.pending != nil {
			before := target.pending
			target.state = &catalogConstraintState{addition: before.addition, check: before.check, enabled: enabled, trusted: enabled && (before.trusted || validate)}
		} else {
			fk := target.before.Foreign
			fk.Id = strings.Clone(fk.Id)
			fk.Cols1, fk.Cols2 = append([]string(nil), fk.Cols1...), append([]string(nil), fk.Cols2...)
			fk.Disabled, fk.Untrusted = !enabled, !enabled || !validate && fk.Untrusted
			fk = ownForeignPayload(fk)
			target.add = &catalogForeignAmendment{declaration: catalogForeignDeclaration{child.RuntimeIdentity, target.parent.RuntimeIdentity, fk}, rowInput: true}
			target.state = &catalogConstraintState{addition: target.add, enabled: enabled, trusted: !fk.Untrusted}
		}
		if validate {
			if target.state.addition != nil {
				fk := target.state.addition.declaration.Foreign
				parent := db.tables.Get(fk.Tbl2)
				child.enterPendingCatalogData(tx, changes.leases[child])
				validateExistingForeignRows(tx, child, fk, newForeignKeyLookup(parent, fk.Cols2), changes.leases[parent])
			} else {
				validateExistingCheckRows(tx, child, target.state.check.declaration.Check, target.state.check.programs[0].trigger.Func)
			}
		}
		if target.add != nil {
			// Like an initially enabled pending addition, retain the prospective
			// programs even when this private state is disabled. A later CHECK in
			// this same T may enable them; the accepted state alone selects which
			// programs publish. Nothing is installed here.
			program := target.add.declaration.Foreign
			program.Disabled = false
			target.add.programs = buildFKTriggers(db, child, target.parent, program)
			captureForeignProgramFootprints(target.add.programs)
			target.drop = &catalogForeignDropAmendment{declaration: target.before, programs: foreignDropPrograms(child, target.parent, target.before.Foreign)}
			for _, old := range target.drop.programs {
				old.table.reserveForeignDropProgram(tx.transactionState, old.trigger)
				target.drop.reservations = append(target.drop.reservations, old)
				reserved = append(reserved, old)
			}
			for _, current := range target.add.programs {
				borrowed := false
				for _, old := range target.drop.programs {
					if current.table == old.table && current.trigger.Name == old.trigger.Name && current.trigger.Timing == old.trigger.Timing && current.trigger.IsSystem && old.trigger.IsSystem {
						borrowed = true
						break
					}
				}
				if !borrowed {
					current.table.reserveDeclarationTrigger(tx.transactionState, current.trigger.Name, false)
					target.add.reservations = append(target.add.reservations, current)
					reserved = append(reserved, current)
				}
			}
		}
	}
	views := maps.Clone(changes.views.Load().tables)
	history := append([]catalogDeclarationAmendment(nil), tx.publicForeignAmendments()...)
	var bytes uint
	for _, target := range targets {
		if target.add == nil {
			continue
		}
		drop, add := target.drop, target.add
		drop.retainedBytes = uint(unsafe.Sizeof(*drop)) + uint(cap(drop.programs)+cap(drop.reservations))*uint(unsafe.Sizeof(foreignKeyTrigger{}))
		for _, program := range drop.programs {
			drop.retainedBytes += uint(len(program.trigger.Name))
		}
		add.retainedBytes = foreignAmendmentFootprint(add) + uint(unsafe.Sizeof(catalogConstraintState{}))
		for i, t := range foreignEndpointTables(child, target.parent) {
			drop.endpoints[i], add.endpoints[i] = t, t
			drop.previous[i] = views[t]
			base := views[t]
			if base == nil {
				base = observed[t]
			}
			var n uint
			removed, n := removeForeignDropMetadata(t, base, target.before.Foreign, false)
			drop.retainedBytes += n
			add.previous[i] = removed
			views[t], n = appendConstraintMetadata(t, removed, []foreignKey{add.declaration.Foreign}, nil, 0)
			add.retainedBytes += n
		}
		drop.retiredBytes = foreignDropRetiredBytes(drop.endpoints[:], drop.declaration.Foreign)
		bytes += drop.retainedBytes + add.retainedBytes
		history = append(history, catalogDeclarationAmendment{foreignDrop: drop}, catalogDeclarationAmendment{foreign: add})
	}
	pending := &catalogConstraintStateAmendment{}
	updates := make(map[string]*catalogConstraintState)
	pendingEndpoints := make(map[*table]bool)
	for _, target := range targets {
		updates[target.state.name()] = target.state
		if target.pending == nil {
			continue
		}
		pending.previous, pending.current = append(pending.previous, target.pending), append(pending.current, target.state)
		for _, t := range target.pending.endpoints() {
			if t != nil {
				pendingEndpoints[t] = true
			}
		}
	}
	if len(pending.current) != 0 {
		pending.retainedBytes = uint(unsafe.Sizeof(*pending)) + uint(cap(pending.previous)+cap(pending.current))*uint(unsafe.Sizeof((*catalogConstraintState)(nil))) + uint(len(pending.current))*uint(unsafe.Sizeof(catalogConstraintState{}))
		pendingUpdates := make(map[string]*catalogConstraintState)
		for _, state := range pending.current {
			pendingUpdates[state.name()] = state
		}
		for t := range pendingEndpoints {
			before := views[t]
			if before == nil {
				panic("pending foreign state lost its private endpoint view")
			}
			pending.views = append(pending.views, catalogConstraintStateView{table: t, previous: before})
			var n uint
			views[t], n = replaceConstraintStateMetadata(before, pendingUpdates)
			pending.retainedBytes += n
		}
		pending.retainedBytes += uint(cap(pending.views)) * uint(unsafe.Sizeof(catalogConstraintStateView{}))
		bytes += pending.retainedBytes
		history = append(history, catalogDeclarationAmendment{constraintState: pending})
	}
	access := db.GetWrite(tx)
	defer access.Release()
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	changes.validateConstraintStateEndpointsLocked(tx)
	for t, before := range observed {
		if db.tables.Get(t.Name) != t || t.showColumnsSnapshot.Load().metadata.declarations != before.metadata.declarations {
			panic("committed foreign state input changed during validation")
		}
	}
	for _, target := range targets {
		if target.pending == nil {
			validateCatalogForeignDrop(db, target.before, catalogForeignEndpointScope{owner: tx.transactionState})
		} else {
			target.pending.validate(db, tx.publicUniqueKeys(), tx.publicDeclarationViews(), catalogForeignEndpointScope{owner: tx.transactionState})
		}
	}
	// Reuse the accepted/recovery interpreter now, before this statement can
	// publish: earlier key/column/table edits retain their original chronology
	// and exact predecessor checks rather than a second state-only validator.
	db.projectNativeDeclarations(tx.publicNativeDeclarations(history), views, catalogNativeCurrent, catalogForeignEndpointScope{owner: tx.transactionState})
	// Existing construction admission excludes decision/rewind; all native
	// proofs precede this sole immutable publication and its retained charges.
	tx.noteStatementMutation(child)
	for t, before := range observed {
		if _, exists := changes.baselines[t]; !exists {
			changes.baselines[t] = foreignBaseline(before)
		}
	}
	if changes.constraintStates == nil {
		changes.constraintStates = make(map[string]*catalogConstraintState)
	}
	for name, state := range updates {
		changes.constraintStates[name] = state
	}
	changes.baselinesHighWater = max(changes.baselinesHighWater, len(changes.baselines))
	changes.viewsHighWater = max(changes.viewsHighWater, len(views))
	changes.constraintStatesHighWater = max(changes.constraintStatesHighWater, len(changes.constraintStates))
	changes.amendmentBytes += bytes
	changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	tx.catalogAmendments.Store(&history)
	changes.publishViews(history, views)
	accepted = true
	return true
}

// Existing DROP and state replacement retain the same actual predecessor
// allocation facts, including both endpoint metadata copies.
func foreignDropRetiredBytes(endpoints []*table, fk foreignKey) uint {
	var bytes uint
	for _, t := range endpoints {
		if t == nil {
			continue
		}
		for _, current := range t.Foreign {
			if current.Id == fk.Id && current.Tbl1 == fk.Tbl1 && current.Tbl2 == fk.Tbl2 {
				bytes += current.payloadFootprint
			}
		}
		for _, row := range t.showColumnsSnapshot.Load().metadata.constraints.Slice()[3].Slice() {
			if foreignMetadataValue(row, "Id").String() == fk.Id && foreignMetadataValue(row, "OtherTable").String() == foreignDropOther(t, fk) {
				bytes += uint(cap(row.Slice())+cap(foreignMetadataValue(row, "LocalColumns").Slice())+cap(foreignMetadataValue(row, "OtherColumns").Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
			}
		}
	}
	return bytes
}
