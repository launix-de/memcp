/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "errors"

const lateCreateResourceKey = "native-private-catalog-database"

// Private declarations can join the original local ACID owner, not adopt an
// already prepared, distributed, public-declaration or independently visible
// decision. The complete participant proof is checked at entry and prepare;
// later declarations retain O(1) admission to the same database and owner.
// Caller holds tx.mu. The existing database lease pins local mode separately.
func (tx *TxContext) validateLateCreateOwnerLocked(db *database, participants bool) error {
	if tx.publicCatalogChanges() != nil {
		// A public phase already owns this original local decision. Preserve its
		// participant proof and writer restrictions while adding private names.
		return tx.validateLatePublicOwnerLocked(db, participants)
	}
	if tx.Mode != TxACID || tx.State != TxActive || tx.decisionStarted || tx.prepared != nil ||
		tx.catalogCommit != nil || tx.coordinator != nil || tx.clusterNumber.Load() != 0 || tx.clusterID != "" ||
		len(tx.participants) != 0 || len(tx.endpoints) != 0 || tx.readViews.Load() != nil ||
		tx.remoteActive != 0 || tx.mutationOperations != 0 || tx.checkedMutation != nil || tx.relationDataOrigins != 0 ||
		tx.publicCatalogEntered.Load() || tx.publicCatalogChanges() != nil ||
		db.incarnationID() == "" || GetDatabase(db.Name) != db ||
		sharedClusterDatabase(db.Name) || activeReplication(db) != nil {
		return errors.New("late private declarations require an unprepared local ACID transaction")
	}
	if !participants {
		return nil
	}
	for shard := range tx.shardTransactions() {
		if shard.t.schema != db && !(shard.t.Ephemeral && shard.t.PersistencyMode == Memory) {
			return errors.New("late private declarations require one public data database")
		}
	}
	foreign := false
	tx.recoveryDecisionDatabases.Range(func(key, _ any) bool {
		foreign = foreign || key.(*database) != db
		return !foreign
	})
	if foreign {
		return errors.New("late private declarations require one recovery decision database")
	}
	return nil
}

// Earlier stored targets retain ordinary native row leases. Only this owner's
// actual new generations can use the separate private declaration boundary.
func (tx *TxContext) lateCreateRowAllowed(t *table) bool {
	identity := tx.catalogIdentity.Load()
	return identity != nil && identity.lateCreates && identity.database == t.schema &&
		identity.Incarnation == t.schema.incarnationID() &&
		(t.committedAt(t.schema.catalogView().version) || tx.ownsPendingTable(t))
}

// Enter once per native data input, before even empty/range/window shortcuts.
// The established resource gate orders this with declaration construction;
// only the first input seals the owned generation. Ordinary and already sealed
// inputs take no new lock or shared synchronization write, including row loops.
// Metadata description does not call this data boundary.
func (t *table) enterPendingCatalogData(tx *TxContext, construction *tableWriteLease) {
	if tx == nil || tx.transactionState == nil || t.catalogDataEntered.Load() || t.creator.Load() != tx.transactionState {
		return
	}
	identity := tx.catalogIdentity.Load()
	if identity == nil || !tx.ownsPendingTable(t) {
		return
	}
	scope := tx.resourceScope()
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if scope.closed || scope.deciding || scope.constructing != 0 && construction == nil || tx.catalogIdentity.Load() != identity {
		panic("private catalog generation is not open for data access")
	}
	if construction != nil {
		owner := &t.writeAdmission
		owner.mu.Lock()
		valid := scope.constructing != 0 && construction.owner == owner && construction.transaction == tx.transactionState && construction.declaration &&
			owner.declaration == tx.transactionState && owner.declarationGuardedInput &&
			(catalogForeignEndpointScope{owner: tx.transactionState}).owns(t)
		owner.mu.Unlock()
		if !valid {
			panic("catalog validation lost its exact construction-owned input")
		}
	}
	t.catalogDataEntered.Store(true)
	tx.rowAccessed.Store(true)
	scope.publicRowsEntered = true
}

func (tx *TxContext) catalogDeclarationDataEntered(t *table) bool {
	if tx.ownsPendingTable(t) {
		return t.catalogDataEntered.Load()
	}
	return tx.rowAccessed.Load()
}

// Removing the last private generation can leave earlier data journals alive.
// Retire only the unaccepted phase and its local-mode pin. Original WAL T,
// transaction state, row leases and the monotone enforcement boundary survive.
func (tx *TxContext) retireEmptyLateCreatePhase() {
	identity := tx.catalogIdentity.Load()
	if identity == nil || !identity.lateCreates || tx.hasCatalogIntents() {
		return
	}
	scope := tx.resourceScope()
	scope.mu.Lock()
	tx.mu.Lock()
	if tx.State != TxActive {
		tx.mu.Unlock()
		scope.mu.Unlock()
		return
	}
	if tx.decisionStarted || tx.prepared != nil || tx.catalogCommit != nil || scope.closed ||
		!scope.deciding || scope.constructing != 0 {
		tx.mu.Unlock()
		scope.mu.Unlock()
		panic("private catalog retirement requires its unaccepted savepoint owner")
	}
	entry := scope.entries[lateCreateResourceKey]
	if entry == nil {
		tx.mu.Unlock()
		scope.mu.Unlock()
		panic("private catalog phase lost its database lease")
	}
	lease, valid := entry.value.Any().(*databaseLocalDeclarationLease)
	if !valid || lease.db != identity.database {
		tx.mu.Unlock()
		scope.mu.Unlock()
		panic("private catalog phase lost its database owner")
	}
	tx.mu.Unlock()
	scope.mu.Unlock()
	if err := lease.Close(); err != nil {
		panic(err)
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.catalogIdentity.Load() != identity || tx.decisionStarted || tx.prepared != nil ||
		tx.catalogCommit != nil || !scope.deciding || tx.hasCatalogIntents() ||
		scope.entries[lateCreateResourceKey] != entry {
		panic("private catalog phase changed during native retirement")
	}
	delete(scope.entries, lateCreateResourceKey)
	tx.catalogIdentity.Store(nil)
}

// A constructor's schema owners and construction count have already unwound.
// A failed or no-op first CREATE must not change a later ordinary data commit
// to an empty catalog authority that has no durable declaration proof.
func (tx *TxContext) retireUnacceptedLateCreatePhase() {
	identity := tx.catalogIdentity.Load()
	if identity == nil || !identity.lateCreates || tx.hasCatalogIntents() {
		return
	}
	tx.beginResourceDecision()
	defer tx.finishResourceDecision()
	tx.retireEmptyLateCreatePhase()
}
