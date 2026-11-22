/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

// The existing catalog decision owns the exact containing table generation.
// A nil header is a deletion; event programs are regenerated from source.
type catalogStatementTriggerDeclaration struct {
	Table    string              `json:"table"`
	Identity string              `json:"identity"`
	Name     string              `json:"name"`
	Header   *TriggerDescription `json:"header,omitempty"`
}

type catalogStatementTriggerSlot struct {
	table       *table
	identity    string
	tableName   string // immutable physical containing-table key
	baseline    *statementTriggerDefinition
	head        *statementTriggerDefinition
	reservation *statementTriggerDefinition
}

type catalogStatementTriggerAmendment struct {
	slot    *catalogStatementTriggerSlot
	before  *statementTriggerDefinition
	current *statementTriggerDefinition
}

// All constructors/compiler callbacks have returned before this native catalog
// stage. Its existing resource admission orders publication and savepoint Undo.
func (tx *TxContext) stageStatementTrigger(t *table, definition *statementTriggerDefinition, name string, replace, mustExist, ignore bool, expected *statementTriggerDefinition) bool {
	if tx == nil || tx.transactionState == nil {
		panic("statement trigger declaration requires an active native transaction")
	}
	finish := tx.beginCatalogConstruction(catalogStatementTriggerConstruction, t)
	defer func() { finish(); tx.retireUnacceptedPublicCatalogPhase() }()
	if tx.State != TxActive || tx.decisionStarted || tx.coordinator != nil || tx.catalogIntents.Load() != nil {
		panic("statement trigger declaration requires a local public catalog transaction")
	}
	return tx.stagePublicCatalogDeclaration(t.schema, catalogStatementTriggerKind, func(changes *publicCatalogChanges) bool {
		return changes.stageStatementTrigger(tx, t, definition, name, replace, mustExist, ignore, expected)
	})
}

func (changes *publicCatalogChanges) stageStatementTrigger(tx *TxContext, t *table, definition *statementTriggerDefinition, name string, replace, mustExist, ignore bool, expected *statementTriggerDefinition) bool {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	if changes.owner != tx.transactionState || changes.db != t.schema || GetDatabase(changes.db.Name) != changes.db {
		panic("statement trigger declaration lost its native database")
	}
	lease := changes.leases[t]
	newLease := lease == nil
	if newLease {
		var err error
		lease, err = t.writeAdmission.admit(scanQueryContext(tx, context.Background()), tx.transactionState, tableValidatedDeclarationAdmission, false)
		if err != nil {
			panic(err)
		}
	}
	accepted, reserved := false, false
	// A completed earlier producer is not re-executed. Retained raw callbacks
	// cannot bypass the new program input; checked producers bind it afresh.
	restoreInput := func() {}
	defer func() {
		if !accepted {
			restoreInput()
			if reserved {
				t.releaseDeclarationTrigger(tx.transactionState, name)
			}
			if newLease {
				if changes.leases[t] == lease {
					delete(changes.leases, t)
				}
				if changes.anchor == lease {
					changes.anchor = nil
				}
				_ = lease.Close()
			}
		}
	}()
	restoreInput = t.guardCatalogDeclarationInput(tx.transactionState)
	db := changes.db
	access := db.GetWrite(tx)
	defer access.Release()
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	if db.tables.Get(t.Name) != t || t.RuntimeIdentity == "" || !t.committedAt(db.catalogView().version) || t.Ephemeral || sharedClusterDatabase(db.Name) || activeReplication(db) != nil {
		panic("statement trigger declaration requires its current local committed table")
	}
	tx.requireDeclarationEndpoint(t, maintenanceAlter)
	// Source compilation uses the owner-visible name; retained slots and journal
	// endpoints use the separately attested immutable physical key.
	if definition != nil && (definition.compiledTableName != t.visibleNameAt(tx, db.catalogView()) || definition.compiledIdentity != t.RuntimeIdentity) {
		panic("statement trigger target changed during source compilation")
	}
	// Trigger names share the existing database-wide user-trigger namespace.
	// This cold DDL traversal is never performed by a row or statement callback.
	for _, other := range db.tables.GetAll() {
		if other == t {
			continue
		}
		other.mu.Lock()
		collision := false
		for _, trigger := range other.Triggers {
			if trigger.Name == name {
				collision = true
				break
			}
		}
		other.mu.Unlock()
		if collision {
			panic("trigger name belongs to another native table: " + name)
		}
	}
	t.mu.Lock()
	if definition != nil && definition.header.Timing == ReplaceStatement {
		for _, other := range t.visibleTriggerDescriptionsLocked(tx) {
			if other.Name == name || other.Timing != ReplaceStatement {
				continue
			}
			for _, event := range other.StatementEvents {
				for _, requested := range definition.header.StatementEvents {
					if event == requested {
						t.mu.Unlock()
						panic("native statement event already has a replacement producer: " + event)
					}
				}
			}
		}
	}
	var current *statementTriggerDefinition
	for _, trigger := range t.Triggers {
		if trigger.Name != name {
			continue
		}
		if !trigger.Timing.isStatement() || trigger.IsSystem {
			t.mu.Unlock()
			panic("statement trigger cannot replace another native trigger role: " + name)
		}
		current = trigger.statementDefinition
		break
	}
	if current.pending(db.catalogView().version) && current.pendingDecision != tx.pendingCatalogKey() {
		t.mu.Unlock()
		panic("statement trigger name is reserved by another transaction: " + name)
	}
	visible := current.visible(db, tx)
	// A state amendment must not reinstall source captured before a concurrent
	// ALTER. Compare the actual immutable definition under its publication owner.
	if expected != nil && visible != expected {
		t.mu.Unlock()
		panic("statement trigger definition changed during state amendment: " + name)
	}
	if (mustExist || definition == nil) && visible == nil {
		t.mu.Unlock()
		if definition == nil && ignore {
			return false
		}
		panic("statement trigger does not exist: " + name)
	}
	if definition != nil && visible != nil && !replace {
		t.mu.Unlock()
		panic("statement trigger already exists: " + name)
	}
	// A retained descriptive reader is also a real definition lifetime owner.
	func() {
		defer t.mu.Unlock()
		t.requireTriggerDeclarationMutationLocked()
	}()
	slot := changes.statementTriggers[name]
	if slot == nil {
		t.reserveDeclarationTrigger(tx.transactionState, name, true)
		reserved = true
		baseline := current.visible(db, nil)
		reservation := newStatementTriggerDefinition(TriggerDescription{Name: name, Timing: AfterStatement})
		reservation.pendingDecision, reservation.reservation = tx.pendingCatalogKey(), true
		reservation.previousCommitted.Store(baseline)
		slot = &catalogStatementTriggerSlot{table: t, identity: t.RuntimeIdentity, tableName: t.Name, baseline: baseline, reservation: reservation}
		reservation.memory.retain(db)
		if baseline != nil {
			baseline.memory.retain(db)
		}
	}
	if definition == nil {
		definition = newStatementTriggerDefinition(TriggerDescription{Name: name, Timing: AfterStatement})
		definition.removed = true
	}
	definition.pendingDecision = tx.pendingCatalogKey()
	definition.previousCommitted.Store(slot.baseline)
	definition.memory.retain(db) // exact amendment/history payload owner
	amendment := &catalogStatementTriggerAmendment{slot: slot, before: slot.head, current: definition}
	history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{statementTrigger: amendment})
	if changes.statementTriggers == nil {
		changes.statementTriggers = make(map[string]*catalogStatementTriggerSlot)
	}
	changes.statementTriggers[slot.reservation.header.Name] = slot
	changes.statementTriggerHighWater = max(changes.statementTriggerHighWater, len(changes.statementTriggers))
	changes.statementTriggerHistoryBytes += uint(unsafe.Sizeof(*amendment))
	changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	slot.head = definition
	t.mu.Lock()
	t.setStatementTriggerLocked(definition)
	t.mu.Unlock()
	tx.catalogAmendments.Store(&history)
	if newLease {
		changes.leases[t] = lease
		changes.leasesHighWater = max(changes.leasesHighWater, len(changes.leases))
		if changes.anchor == nil {
			changes.refreshMemoryLocked()
			lease.attachRetainedMemory(changes.memory)
			changes.anchor = lease
		}
	}
	changes.publishViews(history, changes.views.Load().tables)
	accepted = true
	return true
}

// Caller owns changes.mu. A retained reservation is an actual resource owned
// by this original transaction, even when savepoint rewind leaves no live head.
// Attest it once before deriving the immutable checked-input directory.
func (changes *publicCatalogChanges) statementInputTables() map[*table]struct{} {
	if len(changes.statementTriggers) == 0 {
		return nil
	}
	inputs := make(map[*table]struct{}, len(changes.statementTriggers))
	for _, slot := range changes.statementTriggers {
		t := slot.table
		lease := changes.leases[t]
		if changes.owner == nil || t.schema != changes.db || GetDatabase(changes.db.Name) != changes.db ||
			changes.db.tables.Get(slot.tableName) != t || t.RuntimeIdentity != slot.identity || lease == nil {
			panic("statement trigger input lost its retained native endpoint")
		}
		owner := &t.writeAdmission
		owner.mu.Lock()
		valid := lease.owner == owner && lease.transaction == changes.owner && lease.declaration && owner.declaration == changes.owner
		owner.mu.Unlock()
		if !valid {
			panic("statement trigger input lost its original declaration lease")
		}
		inputs[t] = struct{}{}
	}
	return inputs
}

// Exact reservation admission precedes this non-failing header publication.
func (t *table) setStatementTriggerLocked(definition *statementTriggerDefinition) {
	name := definition.header.Name
	for i, current := range t.Triggers {
		if current.Name == name {
			t.Triggers[i] = definition.triggerHeader()
			t.refreshTriggerMemoryLocked()
			return
		}
	}
	t.insertTriggerLocked(definition.triggerHeader())
}

func (t *table) removeStatementTriggerHeaderLocked(name string) {
	for i, current := range t.Triggers {
		if current.Name == name {
			copy(t.Triggers[i:], t.Triggers[i+1:])
			clear(t.Triggers[len(t.Triggers)-1:])
			t.Triggers = t.Triggers[:len(t.Triggers)-1]
			t.refreshTriggerMemoryLocked()
			return
		}
	}
}

func (changes *publicCatalogChanges) validateStatementTriggersLocked() error {
	for name, slot := range changes.statementTriggers {
		if changes.db.tables.Get(slot.tableName) != slot.table || slot.table.RuntimeIdentity != slot.identity {
			return fmt.Errorf("statement trigger containing table changed: %s", name)
		}
		expected := slot.head
		if expected == nil {
			expected = slot.reservation
		}
		slot.table.mu.Lock()
		found := false
		for _, header := range slot.table.Triggers {
			if header.Name == name && header.statementDefinition == expected {
				found = true
				break
			}
		}
		slot.table.mu.Unlock()
		if !found || expected.previousCommitted.Load() != slot.baseline {
			return fmt.Errorf("statement trigger reservation changed: %s", name)
		}
	}
	return nil
}

func (tx *TxContext) publicStatementTriggerDeclarations() []catalogStatementTriggerDeclaration {
	changes := tx.publicCatalogChanges()
	if changes == nil {
		return nil
	}
	changes.mu.Lock()
	defer changes.mu.Unlock()
	result := make([]catalogStatementTriggerDeclaration, 0, len(changes.statementTriggers))
	for name, slot := range changes.statementTriggers {
		if slot.head == nil {
			continue
		}
		var header *TriggerDescription
		if !slot.head.removed {
			value := slot.head.triggerHeader()
			header = &value
		}
		result = append(result, catalogStatementTriggerDeclaration{Table: slot.tableName, Identity: slot.identity, Name: name, Header: header})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// Version publication, not the per-table header writes, makes all recipes live.
func (changes *publicCatalogChanges) publishStatementTriggersLocked(version uint64) {
	for _, slot := range changes.statementTriggers {
		if slot.head == nil {
			continue
		}
		if !slot.head.removed {
			slot.head.memory.retain(changes.db)
		}
		slot.head.publicationVersion.Store(version)
		if slot.baseline != nil {
			slot.baseline.memory.release(changes.db)
		}
	}
}

func (changes *publicCatalogChanges) finishStatementTriggerPublicationLocked() {
	for _, slot := range changes.statementTriggers {
		if slot.head != nil && slot.head.removed {
			slot.table.mu.Lock()
			slot.table.removeStatementTriggerHeaderLocked(slot.head.header.Name)
			slot.table.mu.Unlock()
		}
	}
}

func (changes *publicCatalogChanges) restoreStatementTriggerLocked(amendment *catalogStatementTriggerAmendment) {
	slot := amendment.slot
	slot.head = amendment.before
	current := slot.head
	if current == nil {
		current = slot.reservation
	}
	slot.table.mu.Lock()
	slot.table.setStatementTriggerLocked(current)
	slot.table.mu.Unlock()
	amendment.current.memory.release(changes.db)
	changes.statementTriggerHistoryBytes -= uint(unsafe.Sizeof(*amendment))
}

func (changes *publicCatalogChanges) closeStatementTriggersLocked() {
	if len(changes.statementTriggers) == 0 {
		return
	}
	access := changes.db.GetWrite(nil)
	defer access.Release()
	changes.db.schemalock.Lock()
	defer changes.db.schemalock.Unlock()
	committed := changes.owner.State == TxCommitted
	for _, slot := range changes.statementTriggers {
		slot.table.mu.Lock()
		if !committed || slot.head == nil {
			if slot.baseline == nil {
				slot.table.removeStatementTriggerHeaderLocked(slot.reservation.header.Name)
			} else {
				slot.table.setStatementTriggerLocked(slot.baseline)
			}
		} else {
			slot.head.previousCommitted.Store(nil)
		}
		slot.table.mu.Unlock()
		if slot.baseline != nil {
			slot.baseline.memory.release(changes.db)
		}
		slot.reservation.memory.release(changes.db)
	}
	if history := changes.owner.catalogAmendments.Load(); history != nil {
		for _, amendment := range *history {
			if amendment.statementTrigger != nil {
				amendment.statementTrigger.current.memory.release(changes.db)
			}
		}
	}
	changes.statementTriggers = nil
}

// Recovery constructs private, uncompiled headers and validates every target
// before applying any delta. A name alone never authorizes a new incarnation.
func (db *database) recoverStatementTriggerDeclarations(declarations []catalogStatementTriggerDeclaration) {
	for _, declaration := range declarations {
		t := db.tables.Get(declaration.Table)
		if t == nil || t.RuntimeIdentity != declaration.Identity {
			panic("native statement trigger decision lost its containing table identity")
		}
		for _, other := range db.tables.GetAll() {
			for _, header := range other.Triggers {
				if header.Name == declaration.Name && (other != t || !header.Timing.isStatement() || header.IsSystem) {
					panic("native statement trigger decision conflicts with an existing declaration")
				}
			}
		}
	}
	for _, declaration := range declarations {
		t := db.tables.Get(declaration.Table)
		for _, header := range t.Triggers {
			if header.Name == declaration.Name {
				header.statementDefinition.memory.release(db)
				break
			}
		}
		if declaration.Header == nil {
			t.removeStatementTriggerHeaderLocked(declaration.Name)
		} else {
			definition := declaration.Header.statementDefinition
			definition.memory.retain(db)
			t.setStatementTriggerLocked(definition)
		}
	}
}

func createStatementTrigger(t *table, name string, timing TriggerTiming, language, source string, events []string, replace, mustExist bool, tx *TxContext) bool {
	if mustExist && !replace {
		panic("must-exist statement trigger update requires replacement")
	}
	if t == nil || t.schema == nil || tx == nil {
		panic("statement trigger creation requires a native table and transaction")
	}
	t.NativeIdentity(tx)
	tx.requireDeclarationEndpoint(t, maintenanceAlter)
	header := TriggerDescription{Name: name, Timing: timing, Language: language, Source: source, StatementEvents: events}
	if err := validateStatementTrigger(header); err != nil {
		panic(err)
	}
	definition := newStatementTriggerDefinition(header)
	definition.compiledTableName, definition.compiledIdentity = strings.Clone(t.visibleNameAt(tx, t.schema.catalogView())), strings.Clone(t.RuntimeIdentity)
	definition.memory.bytes += uint(len(definition.compiledTableName) + len(definition.compiledIdentity))
	// Validate the complete source recipe before acquiring publication locks.
	// All programs are immutable and carry explicit runtime invocation inputs.
	for _, event := range events {
		definition.callable(t, parseNativeMutationEvent(event))
	}
	return tx.stageStatementTrigger(t, definition, name, replace, mustExist, false, nil)
}

func dropStatementTrigger(db *database, name string, ignore bool, tx *TxContext) bool {
	if db == nil || tx == nil {
		panic("statement trigger removal requires a native database and transaction")
	}
	db.GetRead(tx)
	db.ensureLoaded()
	for _, t := range db.tables.GetAll() {
		t.mu.Lock()
		found := false
		for _, header := range t.Triggers {
			if header.Name == name && header.Timing.isStatement() {
				found = true
				break
			}
		}
		t.mu.Unlock()
		if found {
			return tx.stageStatementTrigger(t, nil, name, true, true, ignore, nil)
		}
	}
	if ignore {
		return false
	}
	panic("statement trigger does not exist: " + name)
}

// State belongs to the same durable, versioned declaration as source and
// events. No definition is dropped, and engine-generated FK/cache programs are
// outside this user statement-trigger API. Nil names selects that exact role.
func setStatementTriggersEnabled(t *table, names []string, enabled bool, tx *TxContext) bool {
	if t == nil || t.schema == nil || tx == nil {
		panic("statement trigger state requires a native table and transaction")
	}
	t.NativeIdentity(tx)
	tx.requireDeclarationEndpoint(t, maintenanceAlter)
	t.mu.Lock()
	headers := t.visibleTriggerDescriptionsLocked(tx)
	t.mu.Unlock()
	selected := make([]TriggerDescription, 0, len(headers))
	if names == nil {
		for _, header := range headers {
			if header.Timing.isStatement() && !header.IsSystem && !header.Hidden {
				selected = append(selected, header)
			}
		}
	} else {
		seen := make(map[string]bool, len(names))
		for _, name := range names {
			if seen[name] {
				continue
			}
			seen[name] = true
			found := false
			for _, header := range headers {
				if header.Name == name && header.Timing.isStatement() && !header.IsSystem && !header.Hidden {
					selected = append(selected, header)
					found = true
					break
				}
			}
			if !found {
				panic("statement trigger does not belong to target table: " + name)
			}
		}
	}
	// Resolve the complete target list before staging its first amendment. The
	// ordinary native statement decision restores all amendments on failure.
	for _, header := range selected {
		previous := header.statementDefinition
		header.Disabled = !enabled
		definition := newStatementTriggerDefinition(header)
		definition.compiledTableName, definition.compiledIdentity = strings.Clone(t.visibleNameAt(tx, t.schema.catalogView())), strings.Clone(t.RuntimeIdentity)
		definition.memory.bytes += uint(len(definition.compiledTableName) + len(definition.compiledIdentity))
		tx.stageStatementTrigger(t, definition, header.Name, true, true, false, previous)
	}
	return true
}

func initStatementTriggerDeclarations(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "statement_trigger_state", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) != 4 || (!a[1].IsNil() && !a[1].IsSlice()) || !a[2].IsBool() {
			panic("statement_trigger_state requires table, names or nil, enabled and transaction")
		}
		var names []string
		if !a[1].IsNil() {
			names = make([]string, len(a[1].Slice()))
			for i, value := range a[1].Slice() {
				if !value.IsString() {
					panic("statement trigger names must be strings")
				}
				names[i] = value.String()
			}
		}
		return scm.NewBool(setStatementTriggersEnabled(TableFromScmer(a[0]), names, a[2].Bool(), scmerToTxContext(a[3])))
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
		Description: "Stages durable native statement-trigger enabled state in its existing catalog decision. Nil names selects all user statement triggers on this table; system and row triggers are never modified.",
		Params:      []*scm.TypeDescriptor{{Kind: "table", Label: "table"}, {Kind: "any", Label: "names"}, {Kind: "bool", Label: "enabled"}, {Kind: "any", Label: "transaction"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "createstatementtrigger", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) != 9 || !a[2].IsString() || !a[5].IsSlice() {
			panic("createstatementtrigger requires table, name, timing, language, source, events, replace, must-exist and transaction")
		}
		var timing TriggerTiming
		switch a[2].String() {
		case "after_statement":
			timing = AfterStatement
		case "replace_statement":
			timing = ReplaceStatement
		default:
			panic("unknown native statement trigger timing")
		}
		events := make([]string, len(a[5].Slice()))
		for i, event := range a[5].Slice() {
			if !event.IsString() {
				panic("statement trigger events must be strings")
			}
			events[i] = event.String()
		}
		return scm.NewBool(createStatementTrigger(TableFromScmer(a[0]), a[1].String(), timing, a[3].String(), a[4].String(), events, a[6].Bool(), a[7].Bool(), scmerToTxContext(a[8])))
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
		Description: "Stages one native statement trigger source recipe and explicit events in the existing catalog decision. Local committed targets only; pending table creation and row mutation in this declaration transaction remain unsupported.",
		Params:      []*scm.TypeDescriptor{{Kind: "table", Label: "table"}, {Kind: "string", Label: "name"}, {Kind: "string", Label: "timing"}, {Kind: "string", Label: "language"}, {Kind: "string", Label: "source"}, {Kind: "list", Label: "events"}, {Kind: "bool", Label: "replace"}, {Kind: "bool", Label: "must_exist"}, {Kind: "any", Label: "transaction"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "dropstatementtrigger", Fn: func(a ...scm.Scmer) scm.Scmer {
		return scm.NewBool(dropStatementTrigger(GetDatabase(a[0].String()), a[1].String(), a[2].Bool(), scmerToTxContext(a[3])))
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
		Description: "Stages native statement trigger removal under its actual containing-table/name owner, with savepoint and catalog decision semantics.",
		Params:      []*scm.TypeDescriptor{{Kind: "string", Label: "database"}, {Kind: "string", Label: "name"}, {Kind: "bool", Label: "if_exists"}, {Kind: "any", Label: "transaction"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
}
