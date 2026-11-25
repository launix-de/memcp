/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

// This finite cold validation scope is evidence from the existing original
// transaction or accepted recovery record. It grants no row/maintenance rights.
// A public predecessor remains the default; only exact registered generations
// belonging to the same owner or accepted record may still be unpublished.
type catalogForeignEndpointScope struct {
	owner     *transactionState
	recovered map[*table]struct{}
	retired   map[*table]struct{}
}

func (scope catalogForeignEndpointScope) owns(t *table) bool {
	if t == nil || t.schema == nil || t.schema.tables.Get(t.Name) != t || t.RuntimeIdentity == "" {
		return false
	}
	if scope.owner != nil && GetDatabase(t.schema.Name) == t.schema && t.creator.Load() == scope.owner && scope.owner.ownsPendingTable(t) {
		return true
	}
	_, accepted := scope.recovered[t]
	return accepted
}

func (scope catalogForeignEndpointScope) retires(t *table) bool {
	if scope.owner != nil && (&TxContext{transactionState: scope.owner}).ownsTableDrop(t) {
		return true
	}
	_, accepted := scope.retired[t]
	return accepted
}

// Caller has decoded and accepted the complete decision, registered these exact
// generations, and retained the predecessor catalog version until publication.
func recoveredForeignEndpointScope(db *database, record *catalogDecisionRecord) catalogForeignEndpointScope {
	if record.Format != 22 && record.Format != 23 && record.Format != 24 && record.Format != 25 && record.Format != 26 && record.Format != 28 && record.Format != 29 && record.Format != 30 && record.Format != 31 && record.Format != 32 {
		return catalogForeignEndpointScope{}
	}
	if record.Database != db.Name || record.Incarnation != db.incarnationID() || record.Version != db.catalogVersion+1 {
		panic("owned foreign recovery lost its accepted catalog predecessor")
	}
	scope := catalogForeignEndpointScope{retired: make(map[*table]struct{}), recovered: make(map[*table]struct{}, len(record.Tables))}
	for _, t := range record.Tables {
		if db.tables.Get(t.Name) != t || t.CreatorTxID != record.Transaction || t.CatalogDecisionID != record.Decision || t.CreatorIncarnation != record.Incarnation || t.CreatorNamespace != record.Namespace || t.CatalogVersion.Load() != record.Version {
			panic("owned foreign recovery lost its accepted generation")
		}
		scope.recovered[t] = struct{}{}
	}
	if record.Format == 23 || record.Format == 26 || record.Format == 28 || record.Format == 29 || (record.Format == 30 || (record.Format == 31 || record.Format == 32)) {
		for _, d := range record.TableDrops {
			t := db.tables.Get(d.Name)
			if t == nil || t.RuntimeIdentity != d.Identity || !t.committedAt(db.catalogView().version) {
				panic("retired namespace recovery lost its accepted predecessor")
			}
			scope.retired[t] = struct{}{}
		}
	}
	return scope
}

func (tx *TxContext) requireDeclarationEndpoint(t *table, operation maintenanceOperation) {
	if t == nil || t.schema == nil || t.schema.tables.Get(t.Name) != t {
		panic("catalog declaration endpoint generation was replaced")
	}
	if t.ScopedSchema {
		// A lexical MEMORY generation may be unpublished or belong to a later
		// transaction in the same connection. Its actual slot/session owner,
		// rather than public-name visibility or constructor identity, attests it.
		tx.requireScopedRelationOwner(t)
	} else if t.committedAt(t.schema.catalogView().version) {
		requireTableMaintenance(t.schema.Name, t.Name, operation)
		return
	} else if !(catalogForeignEndpointScope{owner: tx.transactionState}).owns(t) {
		panic("catalog declaration requires its original native endpoint owner")
	}
	if !declaredTableMaintenanceCapabilities(t.schema.Name, t.Name).allows(operation) {
		panic("catalog declaration requires its original native endpoint owner")
	}
}

func ownedForeignAddition(owner *transactionState, addition *catalogForeignAmendment) bool {
	if owner == nil || addition == nil {
		return false
	}
	for _, t := range addition.endpoints {
		if (catalogForeignEndpointScope{owner: owner}).owns(t) {
			return true
		}
	}
	return false
}
