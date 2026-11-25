/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "testing"
import "strings"

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
	tx.catalogAmendments.Store(&history)
	intents := []*table{support}
	tx.catalogIntents.Store(&intents)
	var failure any
	func() { defer func() { failure = recover() }(); tx.discardCatalogIntents(0) }()
	if failure == nil || !strings.Contains(fmt.Sprint(failure), "existing owned namespace decision") {
		t.Fatalf("partial constructor cancel did not refuse its unsupported surviving program: %v", failure)
	}
	if tx.catalogIntents.Load() != &intents || tx.catalogAmendments.Load() != &history {
		t.Fatal("failed constructor cancellation changed the selected original program")
	}
}
