/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "errors"

const publicDeclarationResourceKey = "native-public-declarations"

// Caller holds the existing resource-scope mutex. The first public factory
// owns its unfinished entry until complete stage acceptance or cleanup; a
// recursive declaration may neither wait on it nor retire that outer owner.
func (scope *transactionResources) publicDeclarationPendingLocked() bool {
	if entry := scope.entries[publicDeclarationResourceKey]; entry != nil {
		select {
		case <-entry.done:
		default:
			return true
		}
	}
	return false
}

// Non-enforcing public deltas join an existing local ACID owner. Earlier WAL still
// names that owner's original T; catalog prepare binds T to the final decision.
// No prepared/remote authority, isolation conversion or database migration can
// be adopted here. Caller holds tx.mu at both identity-admission boundaries.
func (tx *TxContext) validateLatePublicOwnerLocked(db *database, participants bool, initializer *nativeColumnInitializer) error {
	if tx.Mode != TxACID || tx.State != TxActive || tx.decisionStarted || tx.prepared != nil ||
		tx.catalogCommit != nil || tx.coordinator != nil || tx.clusterNumber.Load() != 0 || tx.clusterID != "" ||
		len(tx.participants) != 0 || len(tx.endpoints) != 0 || tx.readViews.Load() != nil ||
		tx.remoteActive != 0 || (!initializer.owns(tx, db) && (tx.mutationOperations != 0 || tx.checkedMutation != nil)) || tx.relationDataOrigins != 0 ||
		db.incarnationID() == "" || GetDatabase(db.Name) != db ||
		sharedClusterDatabase(db.Name) || activeReplication(db) != nil {
		return errors.New("late public declarations require an unprepared local ACID transaction")
	}
	if intents := tx.catalogIntents.Load(); intents != nil {
		for _, t := range *intents {
			if t.schema != db || !(catalogForeignEndpointScope{owner: tx.transactionState}).owns(t) {
				return errors.New("joint declarations lost their original private generation")
			}
		}
	}
	if !participants {
		return nil
	}
	for shard := range tx.shardTransactions() {
		// Native scoped RAM participants keep their original undo/lifetime. Only
		// durable public data requires this database's final recovery proof.
		if !(shard.t.Ephemeral && shard.t.PersistencyMode == Memory) && !localCatalogRowDatabase(shard.t.schema) {
			return errors.New("late public declarations require local public data databases")
		}
	}
	foreign := false
	tx.recoveryDecisionDatabases.Range(func(key, _ any) bool {
		foreign = foreign || !localCatalogRowDatabase(key.(*database))
		return !foreign
	})
	if foreign {
		return errors.New("late public declarations require local recovery decision databases")
	}
	return nil
}

// Ordinary local DML on unamended endpoints and borrowing a default endpoint's declaration
// lease are separate rights. Callable source edits borrow no table lease;
// unamended tables retain their ordinary native row admission.
func (tx *TxContext) latePublicRowAllowed(t *table) bool {
	identity := tx.catalogIdentity.Load()
	if identity == nil || identity.lateCreates && !tx.publicCatalogEntered.Load() || !localCatalogRowDatabase(t.schema) || !localCatalogRowDatabase(identity.database) ||
		tx.Mode != TxACID || tx.State != TxActive || tx.coordinator != nil || tx.clusterNumber.Load() != 0 ||
		tx.clusterID != "" || len(tx.participants) != 0 {
		return false
	}
	changes := tx.publicCatalogChanges()
	if changes == nil {
		// A failed first public factory can leave original private intents and
		// the monotone phase marker. It published no public row restriction.
		return (catalogForeignEndpointScope{owner: tx.transactionState}).owns(t) || identity.lateCreates && tx.lateCreateRowAllowed(t)
	}
	views := changes.views.Load()
	if views == nil || identity.Incarnation != identity.database.incarnationID() {
		return false
	}
	_, blocked := views.blockedWriters[t]
	return !blocked
}

func localCatalogRowDatabase(db *database) bool {
	return db != nil && GetDatabase(db.Name) == db && db.incarnationID() != "" &&
		!sharedClusterDatabase(db.Name) && activeReplication(db) == nil
}

// Savepoint restoration can remove every public delta, including a clean
// first declaration. Retire only this unaccepted public phase: the original WAL T, snapshot,
// row leases and monotone enforcement boundary survive. An accepted/prepared
// decision is never switched back to another authority.
func (tx *TxContext) retireEmptyPublicCatalogPhase() {
	identity := tx.catalogIdentity.Load()
	if identity == nil || !tx.publicCatalogEntered.Load() || tx.hasCatalogIntents() {
		return
	}
	scope := tx.resourceScope()
	scope.mu.Lock()
	tx.mu.Lock()
	if tx.State != TxActive {
		tx.mu.Unlock()
		scope.mu.Unlock()
		return // whole-transaction cleanup owns terminal retirement
	}
	if tx.decisionStarted || tx.prepared != nil || tx.catalogCommit != nil || scope.closed ||
		!scope.deciding || scope.constructing != 0 {
		tx.mu.Unlock()
		scope.mu.Unlock()
		panic("public catalog phase retirement requires its unaccepted savepoint owner")
	}
	entry := scope.entries[publicDeclarationResourceKey]
	changes := tx.publicCatalogChanges()
	if entry != nil {
		value, valid := entry.value.Any().(*publicCatalogChanges)
		if !valid || value != changes {
			tx.mu.Unlock()
			scope.mu.Unlock()
			panic("public catalog phase lost its native resource")
		}
	}
	tx.mu.Unlock()
	scope.mu.Unlock()
	if changes != nil {
		if err := changes.Close(); err != nil {
			panic(err)
		}
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.catalogIdentity.Load() != identity || tx.decisionStarted || tx.prepared != nil ||
		tx.catalogCommit != nil || !scope.deciding || tx.hasCatalogIntents() ||
		scope.entries[publicDeclarationResourceKey] != entry {
		panic("public catalog phase changed during native retirement")
	}
	delete(scope.entries, publicDeclarationResourceKey)
	// A mixed late CREATE still has its existing database-mode resource. The
	// immediately following private phase retirement closes that original pin.
	if !identity.lateCreates {
		tx.catalogIdentity.Store(nil)
	}
}

// A failed or no-op first declaration has no accepted delta. Its factory has
// already unwound; acquire the ordinary decision exclusion before retiring only
// that phase. Prior row effects remain owned and can still be rolled back.
func (tx *TxContext) retireUnacceptedPublicCatalogPhase() {
	identity := tx.catalogIdentity.Load()
	if identity == nil || identity.lateCreates || !tx.publicCatalogEntered.Load() || tx.hasCatalogIntents() {
		return
	}
	scope := tx.resourceScope()
	scope.mu.Lock()
	pending := scope.publicDeclarationPendingLocked()
	scope.mu.Unlock()
	if pending {
		return // The original unfinished factory owns cleanup, including failure.
	}
	tx.beginResourceDecision()
	defer tx.finishResourceDecision()
	tx.retireEmptyPublicCatalogPhase()
}
