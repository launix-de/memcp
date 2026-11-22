/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"maps"
	"sort"
	"unsafe"
)

// The actual DDL producer binds every neighboring description before touching
// metadata. Its endpoint leases freeze the FK dependency set; a changed public
// predecessor during admission is rejected rather than publishing a partial set.
func (changes *publicCatalogChanges) acquireNameLeases(tx *TxContext, t *table) ([]*table, func(bool)) {
	observed := t.showColumnsSnapshot.Load()
	current := changes.views.Load().tables[t]
	if current == nil {
		current = observed
	}
	endpoints := map[*table]bool{t: true}
	for _, row := range current.nativeConstraints().Slice()[3].Slice() {
		name := foreignMetadataValue(row, "OtherTable").String()
		other := changes.db.tables.Get(name)
		if other == nil {
			panic("name publication lost its foreign endpoint")
		}
		endpoints[other] = true
	}
	tables := make([]*table, 0, len(endpoints))
	for endpoint := range endpoints {
		tables = append(tables, endpoint)
	}
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })
	var finishes []func(bool)
	complete := false
	defer func() {
		if !complete {
			for i := len(finishes) - 1; i >= 0; i-- {
				finishes[i](false)
			}
		}
	}()
	for _, endpoint := range tables {
		finishes = append(finishes, changes.acquireStoredColumnLease(tx, endpoint, true))
	}
	if t.showColumnsSnapshot.Load().metadata.declarations != observed.metadata.declarations {
		panic("name dependency declaration changed during admission")
	}
	complete = true
	return tables, func(accepted bool) {
		for i := len(finishes) - 1; i >= 0; i-- {
			finishes[i](accepted)
		}
	}
}

// Metadata-only neighbor carriers share the same ordered amendment/rewind
// owner. They introduce no constraint recipe or synthetic row producer.
func (changes *publicCatalogChanges) nameNeighborHistory(tx *TxContext, t *table, endpoints []*table, history []catalogDeclarationAmendment, views map[*table]*tableShowColumnsSnapshot) ([]catalogDeclarationAmendment, map[*table]*tableShowColumnsSnapshot) {
	for _, endpoint := range endpoints {
		if endpoint == t {
			continue
		}
		if endpoint.schema != changes.db || changes.db.tables.Get(endpoint.Name) != endpoint || !endpoint.visibleIn(tx, changes.db.catalogView().version) {
			panic("name publication foreign generation was replaced")
		}
		observed := endpoint.showColumnsSnapshot.Load()
		base := views[endpoint]
		if base == nil {
			base = observed
		}
		a := &catalogStoredColumnAmendment{nameMetadataOnly: true, table: endpoint, columns: base.metadata.columnNames.declarations, previous: changes.views.Load().tables[endpoint]}
		a.retainedBytes = uint(unsafe.Sizeof(*a))
		changes.amendmentBytes += a.retainedBytes
		history = appendCatalogDeclarationHistory(history, catalogDeclarationAmendment{column: a})
		views[endpoint] = base
		if _, exists := changes.baselines[endpoint]; !exists {
			changes.baselines[endpoint] = foreignBaseline(observed)
		}
	}
	return history, views
}

// Cold publication only. A SHOW/declaration getter remains a direct immutable
// snapshot read; there is no late schema traversal or row-path name mapping.
func (changes *publicCatalogChanges) projectNameViews(tables map[*table]*tableShowColumnsSnapshot, names *catalogNamespaceView) map[*table]*tableShowColumnsSnapshot {
	var result map[*table]*tableShowColumnsSnapshot
	for t, snapshot := range tables {
		projected, bytes := t.projectConstraintNames(snapshot, tables, names)
		if projected != snapshot {
			if result == nil {
				result = maps.Clone(tables)
			}
			result[t] = projected
			changes.nameProjectionBytes += bytes
		}
	}
	if result != nil {
		return result
	}
	return tables
}

func (publication *publicForeignPublication) projectNameMetadata() uint {
	var result uint
	views := make(map[*table]*tableShowColumnsSnapshot, len(publication.endpoints))
	for _, endpoint := range publication.endpoints {
		views[endpoint.table] = endpoint.snapshot
	}
	names := nativeNamespaceProjection((&TxContext{transactionState: publication.changes.owner}).publicForeignAmendments())
	for _, endpoint := range publication.endpoints {
		snapshot, bytes := endpoint.table.projectConstraintNames(endpoint.snapshot, views, names)
		endpoint.snapshot = snapshot
		result += bytes
	}
	return result
}

// The unpublished loader has restored every endpoint before this final pass.
// Retain that complete input set while projecting all public descriptions;
// getters never repair names or read a second endpoint on demand.
func (db *database) publishLoadedNameMetadata() {
	tables := db.tables.GetAll()
	views := make(map[*table]*tableShowColumnsSnapshot, len(tables))
	for _, t := range tables {
		views[t] = t.showColumnsSnapshot.Load()
	}
	for _, t := range tables {
		snapshot, _ := t.projectConstraintNames(views[t], views, nil)
		t.showColumnsSnapshot.Store(snapshot)
		t.publishPlannerStatsToken()
	}
}
