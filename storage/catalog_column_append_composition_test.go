/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "context"
import "testing"
import "strings"
import "github.com/launix-de/memcp/scm"

func TestColumnAppendRetirementUsesSurvivingJointProgram(t *testing.T) {
	const targetID = "11111111-1111-4111-8111-111111111111"
	const droppedID = "22222222-2222-4222-8222-222222222222"
	const supportingID = "33333333-3333-4333-8333-333333333333"
	target := &table{Name: "existing", RuntimeIdentity: targetID}
	dropped := &table{Name: "retired", RuntimeIdentity: droppedID}
	support := &table{Name: "constructor", RuntimeIdentity: supportingID}
	appended := &catalogColumnAppend{Table: target.Name, Identity: targetID, Name: "added",
		Type: storedColumnType{Type: "INT", Nullable: true}, Initialization: &catalogColumnInitialization{Extents: map[string]columnInitialExtent{}}}
	history := []catalogDeclarationAmendment{
		{tableDrop: &catalogTableDropAmendment{table: dropped, declaration: catalogTableDropDeclaration{dropped.Name, droppedID}}},
		{column: &catalogStoredColumnAmendment{table: target, appendDeclaration: appended}},
	}
	tx := &TxContext{transactionState: &transactionState{State: TxActive}}
	if err := tx.validateColumnAppendTableProgram(history, nil); err == nil {
		t.Fatal("bare DROP+append acquired a new journal domain")
	}
	if err := tx.validateColumnAppendTableProgram(history, []*table{support}); err != nil {
		t.Fatalf("existing owned V23 program rejected: %v", err)
	}
	same := appendCatalogDeclarationHistory(history, catalogDeclarationAmendment{tableDrop: &catalogTableDropAmendment{
		table: target, declaration: catalogTableDropDeclaration{target.Name, targetID}}})
	if err := tx.validateColumnAppendTableProgram(same, []*table{support}); err == nil {
		t.Fatal("same native endpoint acquired append/drop semantics")
	}
	alias := appendCatalogDeclarationHistory(history, catalogDeclarationAmendment{tableDrop: &catalogTableDropAmendment{
		table: target, declaration: catalogTableDropDeclaration{"other_name", targetID}}})
	if err := tx.validateColumnAppendTableProgram(alias, []*table{support}); err == nil {
		t.Fatal("another physical spelling hid the same retiring native UUID")
	}
	for _, generated := range []bool{false, true} {
		if generated {
			appended.GeneratedIDs = &columnIDGeneration{1, 1}
		} else {
			appended.RowInitialization = &catalogColumnDefaultDeclaration{}
		}
		if err := tx.validateColumnAppendTableProgram(history, []*table{support}); err == nil {
			t.Fatal("ordinary prefix composition widened realized/generated append authority")
		}
		appended.GeneratedIDs, appended.RowInitialization = nil, nil
	}
	// The sole supporting name action is genuine, but disappears if its native
	// endpoint is retired by the prospective suffix. Projection must read the
	// supplied program rather than the transaction's older published views.
	renamed := appendCatalogDeclarationHistory(history, catalogDeclarationAmendment{tableRename: &catalogTableRenameAmendment{
		table: support, declaration: &catalogTableRename{Storage: support.Name, Identity: supportingID, Before: support.Name, After: "renamed"}}})
	if err := tx.validateColumnAppendTableProgram(renamed, nil); err != nil {
		t.Fatalf("existing name V23 program rejected: %v", err)
	}
	withoutName := appendCatalogDeclarationHistory(renamed, catalogDeclarationAmendment{tableDrop: &catalogTableDropAmendment{
		table: support, declaration: catalogTableDropDeclaration{support.Name, supportingID}}})
	if err := tx.validateColumnAppendTableProgram(withoutName, nil); err == nil {
		t.Fatal("retiring the sole name proof retained a fictitious V23 profile")
	}
	// Exercise the actual cancellation entry: it must refuse before touching
	// constructor identity/storage, even though the currently selected program
	// still has that constructor. Normal savepoints pass their retained program.
	owner, changes, constructor, publicTarget, retired := cancelledCatalogInputFixture(t)
	selected := []catalogDeclarationAmendment{
		{tableDrop: &catalogTableDropAmendment{table: retired, declaration: catalogTableDropDeclaration{retired.Name, retired.RuntimeIdentity}}},
		{column: &catalogStoredColumnAmendment{table: publicTarget, appendDeclaration: &catalogColumnAppend{
			Table: publicTarget.Name, Identity: publicTarget.RuntimeIdentity, Name: "added", Type: storedColumnType{Type: "INT", Nullable: true},
			Initialization: &catalogColumnInitialization{Extents: map[string]columnInitialExtent{}}}}},
	}
	owner.catalogAmendments.Store(&selected)
	intents, views, lease := owner.catalogIntents.Load(), changes.views.Load(), changes.leases[constructor]
	var failure any
	func() { defer func() { failure = recover() }(); owner.discardCatalogIntents(0) }()
	if failure == nil || !strings.Contains(fmt.Sprint(failure), "existing owned namespace decision") {
		t.Fatalf("partial constructor cancel did not refuse its unsupported surviving program: %v", failure)
	}
	if owner.catalogIntents.Load() != intents || owner.catalogAmendments.Load() != &selected || changes.views.Load() != views ||
		changes.leases[constructor] != lease || lease.owner == nil || changes.db.tables.Get(constructor.Name) != constructor {
		t.Fatal("failed constructor cancellation changed the selected original program or registered native resources")
	}
}

// The native constructor must not turn a rejected prospective program into an
// apparent row/declaration effect. The language's existing failure policy owns
// the resulting untouched error; prior original-T rename/drop/append stays live.
func TestRejectedSupportingNameDropLeavesOriginalStatementUntouched(t *testing.T) {
	tx, changes, target, support, dropped := cancelledCatalogInputFixture(t)
	changes.db.loadOnce.Do(func() {}) // this finite native-owner fixture has no persistence load
	tx.catalogIntents.Store(nil)
	history := []catalogDeclarationAmendment{
		{tableRename: &catalogTableRenameAmendment{table: support, declaration: &catalogTableRename{
			Storage: support.Name, Identity: support.RuntimeIdentity, Before: support.Name, After: "renamed_support"}}},
		{tableDrop: &catalogTableDropAmendment{table: dropped, declaration: catalogTableDropDeclaration{dropped.Name, dropped.RuntimeIdentity}}},
		{column: &catalogStoredColumnAmendment{table: target, appendDeclaration: &catalogColumnAppend{
			Table: target.Name, Identity: target.RuntimeIdentity, Name: "added", Type: storedColumnType{Type: "INT", Nullable: true},
			Initialization: &catalogColumnInitialization{Extents: map[string]columnInitialExtent{}}}}},
	}
	tx.catalogAmendments.Store(&history)
	views := &publicCatalogViews{tables: changes.views.Load().tables,
		names: nativeNamespaceProjection(history), dropped: map[*table]struct{}{dropped: {}}}
	changes.views.Store(views)
	anchor, leases := changes.anchor, len(changes.leases)
	tx.failureTracking.Store(true)
	defer tx.failureTracking.Store(false)
	var failure any
	func() {
		defer func() { failure = recover() }()
		tx.dropCatalogTable(changes.db, "renamed_support", false)
	}()
	if failure == nil || !strings.Contains(fmt.Sprint(failure), "existing owned namespace decision") {
		t.Fatalf("supporting-name DROP did not refuse the unsupported candidate: %v", failure)
	}
	conflict, semantic := failure.(*catalogDeclarationProfileConflict)
	if !semantic || conflict.diagnostic.Slice()[1].String() != "owned-namespace" {
		t.Fatalf("actual profile refusal lost its finite native diagnostic: %T", failure)
	}
	if completion := tx.failureCompletion(tx.ID, tx.statementGeneration.Load(), failure); completion != "untouched" {
		t.Fatalf("rejected admission reported %s and would abort the original transaction", completion)
	}
	if tx.State != TxActive || tx.catalogAmendments.Load() != &history || changes.views.Load() != views ||
		changes.anchor != anchor || len(changes.leases) != leases || tx.resources.constructing != 0 ||
		changes.db.GetTableForTx("renamed_support", false, tx) != support ||
		changes.db.tables.Get(dropped.Name) != dropped {
		t.Fatal("refusing the prospective DROP changed the original program, generation or retained leases")
	}
	// Contrast with an actual accepted mutation note: the same native policy
	// owner must still refuse to preserve an unrestored declaration effect.
	tx.noteStatementMutation(support)
	if completion := tx.failureCompletion(tx.ID, tx.statementGeneration.Load(), failure); completion != "unrestored" {
		t.Fatalf("actual declaration effect reported %s", completion)
	}
}

func TestUnpublishedTableDropRefusesThroughItsActualDeclarationOwner(t *testing.T) {
	tx, changes, constructor, _, _ := cancelledCatalogInputFixture(t)
	changes.db.loadOnce.Do(func() {})
	changes.db.publishTableNamesLocked(nil)
	before, history, anchor := changes.views.Load(), tx.catalogAmendments.Load(), changes.anchor
	tx.failureTracking.Store(true)
	defer tx.failureTracking.Store(false)
	refuse := func() any {
		var failure any
		func() {
			defer func() { failure = recover() }()
			tx.dropCatalogTable(changes.db, constructor.Name, false)
		}()
		return failure
	}
	failure := refuse()
	conflict, semantic := failure.(*catalogDeclarationProfileConflict)
	if !semantic || conflict.diagnostic.Slice()[1].String() != "unpublished-table-drop" ||
		fmt.Sprint(failure) != "transactional table removal requires a local committed public table" {
		t.Fatalf("original private generation did not reach its exact unsupported-profile refusal: %v", failure)
	}
	if tx.failureCompletion(tx.ID, tx.statementGeneration.Load(), failure) != "untouched" ||
		tx.State != TxActive || changes.views.Load() != before || tx.catalogAmendments.Load() != history ||
		changes.anchor != anchor || changes.db.GetTableForTx(constructor.Name, false, tx) != constructor {
		t.Fatal("unsupported private DROP changed the original constructor or public prefix")
	}
	constructor.creator.Store(&transactionState{State: TxActive})
	foreignFailure := refuse()
	constructor.creator.Store(tx.transactionState)
	if _, semantic := foreignFailure.(*catalogDeclarationProfileConflict); foreignFailure == nil || semantic {
		t.Fatalf("foreign constructor bypassed actual owner attestation: %v", foreignFailure)
	}
}

func TestDeclarationProfileDiagnosticGrantsNoNativeRecoveryAuthority(t *testing.T) {
	failure := newCatalogDeclarationProfileConflict("owned-namespace", "profile refused")
	// The shared projection intentionally unwraps Scmer to its ordinary native
	// value. Check complete semantic data, not a wrapper it does not preserve.
	projected := scm.FromAny(nativeFailureOriginalValue(failure))
	if nativeFailurePolicyExcluded(failure) || !projected.IsSlice() || len(projected.Slice()) != 3 ||
		projected.Slice()[0].String() != "catalog-declaration-conflict" ||
		projected.Slice()[1].String() != "owned-namespace" || projected.Slice()[2].String() != "profile refused" {
		t.Fatal("finite semantic refusal lost its ordinary native diagnostic projection")
	}
	unknown := fmt.Errorf("opaque native failure")
	if nativeFailureOriginalValue(unknown) != unknown {
		t.Fatal("unknown native failure was converted into a semantic admission fact")
	}
	for _, excluded := range []error{context.Canceled, context.DeadlineExceeded, &PersistenceFailure{}, &transactionResourceCleanupError{state: TxAborted}} {
		if !nativeFailurePolicyExcluded(excluded) {
			t.Fatalf("failure policy admitted excluded %T", excluded)
		}
	}
}
