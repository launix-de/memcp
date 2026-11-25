/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"fmt"
	"testing"
)

func TestCursorStableOwnedKeyRetainsPreRowBoundary(t *testing.T) {
	tx, _, constructor, _, _ := cancelledCatalogInputFixture(t)
	constructor.catalogDataEntered.Store(true)
	before := constructor.showColumnsSnapshot.Load()
	var cause any
	func() {
		defer func() { cause = recover() }()
		tx.createCatalogKey(constructor, uniqueKey{Id: "late_key", Cols: []string{"id"}})
	}()
	if fmt.Sprint(cause) != "transactional declaration amendments must precede native row access" || constructor.showColumnsSnapshot.Load() != before {
		t.Fatalf("cursor-stable row owner changed its historical key boundary: %v", cause)
	}
}

func TestRawPublicKeyJointEndpointRequiresActualOriginalConstructors(t *testing.T) {
	tx, changes, constructor, public, dropped := cancelledCatalogInputFixture(t)
	tx.Mode = TxACID
	identity := *tx.catalogIdentity.Load()
	identity.Namespace = ""
	tx.catalogIdentity.Store(&identity)
	constructor.CreatorNamespace = ""
	public.CatalogVersion.Store(1)
	dropped.CatalogVersion.Store(1)
	changes.db.catalogVersion = 1
	// Select a coherent no-DROP original program through the same summary
	// owner as savepoint restoration. Keep its real constructors and leases.
	dropHistory := append([]catalogDeclarationAmendment(nil), tx.publicForeignAmendments()...)
	withoutDrop := []catalogDeclarationAmendment{}
	tx.catalogAmendments.Store(&withoutDrop)
	changes.publishViews(withoutDrop, changes.views.Load().tables)
	if tx.rowAccessed.Load() {
		t.Fatal("protected-input control unexpectedly entered a native row phase")
	}
	tx.requireKeyEndpoint(public, nil)
	before := changes.views.Load()
	refused := func() {
		t.Helper()
		requirePanic(t, func() { tx.requireKeyEndpoint(public, nil) })
		if changes.leases[constructor] == nil || changes.leases[public] == nil || changes.views.Load() != before {
			t.Fatal("refused key endpoint changed existing original resources")
		}
	}
	owner := changes.owner
	changes.owner = &transactionState{}
	refused()
	changes.owner = owner
	database := changes.db
	changes.db = newDatabase()
	refused()
	changes.db = database
	tx.Mode = TxCursorStability
	refused()
	tx.Mode = TxACID
	creator := constructor.creator.Load()
	constructor.creator.Store(&transactionState{})
	refused()
	constructor.creator.Store(creator)
	tx.rollbackOnly.Store(true)
	refused()
	tx.rollbackOnly.Store(false)
	// No raw public-key/table-DROP profile is introduced by constructor support.
	tx.catalogAmendments.Store(&dropHistory)
	changes.publishViews(dropHistory, changes.views.Load().tables)
	before = changes.views.Load()
	if !tx.ownsTableDrop(dropped) || len(tx.publicForeignAmendments()) != 1 {
		t.Fatal("fixture did not select its actual retained table-DROP prefix")
	}
	refused()
}

func TestRawPublicKeyRejectsCleanPromotedConstructorUntilActualRowInput(t *testing.T) {
	tx, changes, constructor, public, _ := cancelledCatalogInputFixture(t)
	tx.Mode = TxACID // clean catalog admission promotes a tx_begin CS binding
	identity := *tx.catalogIdentity.Load()
	identity.Namespace = ""
	tx.catalogIdentity.Store(&identity)
	constructor.CreatorNamespace = ""
	public.CatalogVersion.Store(1)
	changes.db.catalogVersion = 1
	db := changes.db
	// Retire the fixture's public resource through its actual close owner.
	// The registered original constructor and its identity remain live.
	if err := changes.Close(); err != nil {
		t.Fatal(err)
	}
	if tx.publicCatalogChanges() != nil || tx.rowAccessed.Load() || !tx.ownsPendingTable(constructor) {
		t.Fatal("clean promoted constructor fixture retained public/row authority")
	}
	tx.publicCatalogEntered.Store(true)
	before, history, intents := public.showColumnsSnapshot.Load(), tx.catalogAmendments.Load(), tx.catalogIntents.Load()
	var cause any
	func() {
		defer func() { cause = recover() }()
		tx.requireKeyEndpoint(public, nil)
	}()
	if fmt.Sprint(cause) != "public unique declaration requires a local committed native table" ||
		public.showColumnsSnapshot.Load() != before || tx.catalogAmendments.Load() != history || tx.catalogIntents.Load() != intents ||
		db.tables.Get(constructor.Name) != constructor || tx.State != TxActive {
		t.Fatalf("clean constructor-only promotion gained public key authority or changed original state: %v", cause)
	}
	// Only the existing native data admission publishes the row capability;
	// no SQL flag, extra owner, caller spelling or fake row count is involved.
	constructor.enterPendingCatalogData(tx, nil)
	if !tx.rowAccessed.Load() || !constructor.catalogDataEntered.Load() {
		t.Fatal("actual original constructor data admission did not publish its input")
	}
	tx.requireKeyEndpoint(public, nil)
}
