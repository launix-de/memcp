/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func retainedOwnedStatementSlot(t *testing.T, changes *publicCatalogChanges, target *table, name string) *catalogStatementTriggerSlot {
	t.Helper()
	reservation := newStatementTriggerDefinition(TriggerDescription{Name: name, Timing: AfterStatement})
	reservation.pendingDecision, reservation.reservation = changes.owner.pendingCatalogKey(), true
	target.reserveDeclarationTrigger(changes.owner, name, true)
	target.mu.Lock()
	target.setStatementTriggerLocked(reservation)
	target.mu.Unlock()
	reservation.memory.retain(changes.db)
	slot := &catalogStatementTriggerSlot{table: target, tableName: target.Name, identity: target.RuntimeIdentity, reservation: reservation}
	if changes.statementTriggers == nil {
		changes.statementTriggers = make(map[string]*catalogStatementTriggerSlot)
	}
	changes.statementTriggers[name] = slot
	t.Cleanup(func() {
		if changes.statementTriggers[name] == slot {
			target.releaseDeclarationTrigger(changes.owner, name)
			changes.closeStatementTriggerSlotLocked(slot, false)
			delete(changes.statementTriggers, name)
		}
	})
	return slot
}

func TestCancelledOwnedStatementSlotReleasesExactNameAndSourceBeforeGeneration(t *testing.T) {
	tx, changes, cancelled, surviving, dropped := cancelledCatalogInputFixture(t)
	removed := retainedOwnedStatementSlot(t, changes, cancelled, "cancelled_statement")
	kept := retainedOwnedStatementSlot(t, changes, surviving, "surviving_statement")
	lease, keptLease, dropLease := changes.leases[cancelled], changes.leases[surviving], changes.leases[dropped]
	validateCancelledCatalogIntents(tx, changes.db, []*table{cancelled}, tx.publicForeignAmendments())
	changes.retireCancelledCatalogInputs(tx, []*table{cancelled})
	if changes.statementTriggers[removed.reservation.header.Name] != nil || len(cancelled.Triggers) != 0 ||
		removed.reservation.memory.references != 0 || removed.reservation.memory.owner != nil ||
		lease.owner != nil || cancelled.writeAdmission.triggerNames != nil {
		t.Fatal("canceled original generation retained its trigger header, reservation, memory or lease")
	}
	if changes.statementTriggers[kept.reservation.header.Name] != kept || kept.reservation.memory.references != 1 ||
		changes.leases[surviving] != keptLease || changes.leases[dropped] != dropLease || changes.db.tables.Get(cancelled.Name) != cancelled {
		t.Fatal("private slot retirement changed a surviving endpoint or removed storage prematurely")
	}
	changes.db.tables.Remove(cancelled.Name)
	changes.publishViews(tx.publicForeignAmendments(), changes.views.Load().tables)
	if _, ok := changes.statementInputTables()[surviving]; !ok {
		t.Fatal("surviving headless statement input lost its original owner")
	}
}

func TestCancelledOwnedStatementSlotRefusesLiveHeadOrWrongReservationBeforeEffects(t *testing.T) {
	tx, changes, cancelled, _, _ := cancelledCatalogInputFixture(t)
	slot := retainedOwnedStatementSlot(t, changes, cancelled, "pending_statement")
	before, lease := changes.views.Load(), changes.leases[cancelled]
	assertUnchanged := func() {
		t.Helper()
		if changes.statementTriggers[slot.reservation.header.Name] != slot || changes.leases[cancelled] != lease ||
			lease.owner == nil || changes.views.Load() != before || slot.reservation.memory.references != 1 ||
			len(cancelled.Triggers) != 1 {
			t.Fatal("refused slot retirement changed native declaration state")
		}
	}
	slot.head = newStatementTriggerDefinition(TriggerDescription{Name: slot.reservation.header.Name, Timing: AfterStatement})
	requirePanic(t, func() { changes.retireCancelledCatalogInputs(tx, []*table{cancelled}) })
	slot.head = nil
	assertUnchanged()
	identity := slot.identity
	slot.identity = "another-native-generation"
	requirePanic(t, func() { changes.retireCancelledCatalogInputs(tx, []*table{cancelled}) })
	slot.identity = identity
	assertUnchanged()
	decision := slot.reservation.pendingDecision
	slot.reservation.pendingDecision = "another-original-decision"
	requirePanic(t, func() { changes.retireCancelledCatalogInputs(tx, []*table{cancelled}) })
	slot.reservation.pendingDecision = decision
	assertUnchanged()
}

func TestOwnedStatementTriggerUsesExistingJournalDomainAndSingleRecipe(t *testing.T) {
	const authority = "11111111-1111-4111-8111-111111111111"
	const identity = "22222222-2222-4222-8222-222222222222"
	db := newDatabase()
	db.Name = "owned_statement_journal"
	target := &table{schema: db, Name: "owned_statement_target", RuntimeIdentity: identity,
		CreatorTxID: authority + "/1", CatalogDecisionID: authority, CreatorIncarnation: authority,
		PersistencyMode: Safe}
	target.CatalogVersion.Store(1)
	target.schemaTopology.Store(&tableSchemaTopology{})
	target.initializeNewAllocator(1)
	header := TriggerDescription{Name: "owned_statement_recipe", Timing: AfterStatement,
		Language: "native-owned-source-test", Source: "retained source", StatementEvents: []string{"update"}}
	pending := newStatementTriggerDefinition(header)
	pending.pendingDecision = authority
	target.Triggers = []TriggerDescription{pending.triggerHeader()}
	record := &catalogDecisionRecord{Format: 22, Kind: "prepare", Database: db.Name, Incarnation: authority,
		Transaction: authority + "/1", Decision: authority, Installation: authority, Boot: authority, Version: 1, Distributed: true,
		Tables: []*table{target}, StatementTriggers: []catalogStatementTriggerDeclaration{{Table: target.Name, Identity: identity, Name: header.Name, Header: &header}}}
	encoded := encodeCatalogDecision(record)
	decoded, err := decodeCatalogDecision(encoded)
	if err != nil || decoded == nil || !strings.HasPrefix(encoded, catalogOwnedDeclarationJournalPrefix) ||
		len(decoded.Tables) != 1 || len(decoded.Tables[0].Triggers) != 0 || len(decoded.StatementTriggers) != 1 {
		t.Fatalf("existing owned domain duplicated/lost the private recipe: %v", err)
	}
	// Replay's existing owner installs all accepted constructors before source
	// deltas. This applies the exact decoded UUID recipe, not the creator's Proc.
	recovered := decoded.Tables[0]
	recovered.schema = db
	db.tables.Set(recovered.Name, recovered)
	db.recoverStatementTriggerDeclarations(decoded.StatementTriggers)
	if len(recovered.Triggers) != 1 || recovered.Triggers[0].Source != header.Source ||
		recovered.Triggers[0].statementDefinition.memory.references != 1 {
		t.Fatal("existing replay failed to install the single accepted source header")
	}
	defer recovered.releaseStatementTriggerMemory(db)
	db.catalogVersion = 1
	checkpoint, err := json.Marshal(recovered)
	if err != nil {
		t.Fatal(err)
	}
	var restored table
	if err := json.Unmarshal(checkpoint, &restored); err != nil || len(restored.Triggers) != 1 || restored.Triggers[0].Source != header.Source {
		t.Fatalf("existing checkpoint header lost accepted constructor trigger: %v", err)
	}
	for _, version := range []int{5, 7} {
		record.Format = version
		if _, err := decodeCatalogDecision(encodeCatalogDecision(record)); err == nil {
			t.Fatalf("historical public trigger format %d acquired private constructor authority", version)
		}
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(encoded, catalogOwnedDeclarationJournalPrefix))
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	fields["unknown_owned_trigger_authority"] = json.RawMessage("true")
	payload, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeCatalogDecision(catalogOwnedDeclarationJournalPrefix + base64.RawURLEncoding.EncodeToString(payload)); err == nil {
		t.Fatal("existing strict owned domain accepted an unknown authority field")
	}
}

func TestOwnedStatementInputFreezesThePrivateConstructorHeader(t *testing.T) {
	tx, changes, constructor, _, _ := cancelledCatalogInputFixture(t)
	slot := retainedOwnedStatementSlot(t, changes, constructor, "frozen_statement")
	history := []catalogDeclarationAmendment{{statementTrigger: &catalogStatementTriggerAmendment{slot: slot}}}
	tx.catalogAmendments.Store(&history)
	before := constructor.showColumnsSnapshot.Load()
	called := false
	var failure any
	func() {
		defer func() { failure = recover() }()
		tx.amendCatalogDeclaration(constructor, func(*table) bool { called = true; return true })
	}()
	if failure == nil || !strings.Contains(fmt.Sprint(failure), "private declaration is already bound") || called ||
		constructor.showColumnsSnapshot.Load() != before || tx.catalogAmendments.Load() != &history {
		t.Fatalf("compiled original constructor input did not freeze the existing amendment boundary: %v", failure)
	}
}

func TestOwnedStatementInputDoesNotBroadenTableRetirementInEitherOrder(t *testing.T) {
	tx, changes, constructor, public, dropped := cancelledCatalogInputFixture(t)
	tx.failureTracking.Store(true)
	defer tx.failureTracking.Store(false)
	refused := func(body func()) {
		t.Helper()
		var failure any
		func() { defer func() { failure = recover() }(); body() }()
		conflict, semantic := failure.(*catalogDeclarationProfileConflict)
		if !semantic || conflict.diagnostic.Slice()[1].String() != "table-drop-statement-trigger" ||
			tx.failureCompletion(tx.ID, tx.statementGeneration.Load(), failure) != "untouched" {
			t.Fatalf("unsupported enforcing/retirement composition lost its untouched semantic refusal: %v", failure)
		}
	}
	history := []catalogDeclarationAmendment{{statementTrigger: &catalogStatementTriggerAmendment{
		slot: &catalogStatementTriggerSlot{table: constructor}}}}
	tx.catalogAmendments.Store(&history)
	before := changes.views.Load()
	refused(func() { changes.stageTableDrop(tx, public, nil) })
	if tx.catalogAmendments.Load() != &history || changes.views.Load() != before {
		t.Fatal("reverse retirement refusal changed the admitted original program")
	}
	changes.views.Store(&publicCatalogViews{tables: before.tables, dropped: map[*table]struct{}{dropped: {}}})
	refused(func() { tx.beginCatalogDeclaration(changes.db, catalogStatementTriggerKind, nil) })
}
