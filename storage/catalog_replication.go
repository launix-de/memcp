/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "strings"

// Catalog partitions use the tenant's existing MOSI domain and their native
// UUID. The fixed system_schema table supplies columns; the system backend
// supplies files. Neither storage placement nor Go runtime tables own rights.
const catalogResourcePrefix = ".catalog:"

func isCatalogResource(resource mosiResource) bool {
	return resource.Shard != "" && strings.HasPrefix(resource.Table, catalogResourcePrefix)
}

func (shard *storageShard) authorityDatabase() *database {
	if shard.catalogOwner != "" {
		return databases.Get(shard.catalogOwner)
	}
	return shard.t.schema
}

func (shard *storageShard) replicationResource() mosiResource {
	name, database := shard.t.Name, shard.t.schema.Name
	if shard.catalogOwner != "" {
		name, database = catalogResourcePrefix+name, shard.catalogOwner
	}
	return mosiResource{Database: database, Table: name, Shard: shard.uuid.String()}
}

func (shard *storageShard) replicationManager() *replicationManager {
	if shard.catalogPrivate {
		return nil
	}
	owner := shard.authorityDatabase()
	// Unpublished/synthetic native descriptors have no database MOSI domain.
	if owner == nil || !validReplicaDatabase(owner.Name) {
		return nil
	}
	return activeReplication(owner)
}

func declareCatalogShard(shard *storageShard) {
	shard.catalogPrivate = false
	if cluster := activeNodeCluster(); cluster != nil && cluster.databases[shard.catalogOwner] {
		cluster.manager.declareShard(shard, cluster.self.ID, true)
	} else if r := replication.Load(); r != nil && validReplicaDatabase(shard.catalogOwner) {
		r.declareShard(shard, r.id, true)
	}
}

func catalogResourceRelation(resource mosiResource) (catalogRelationID, bool) {
	if !isCatalogResource(resource) {
		return 0, false
	}
	name := strings.TrimPrefix(resource.Table, catalogResourcePrefix)
	for i, relation := range catalogRelations {
		if relation.name == name {
			return catalogRelationID(i), true
		}
	}
	return 0, false
}

// A root persists UUID topology only. Native cells recover permissions from
// live peers; these aliases never establish permission to read backend bytes.
func (db *database) restoreCatalogRoutes(root *catalogRoot) {
	if !validReplicaDatabase(db.Name) {
		return
	}
	if r := activeReplication(db); r != nil {
		for from, to := range root.Forwards {
			for relation, target := range root.Shards {
				if target == to {
					resource := mosiResource{Database: db.Name, Table: catalogResourcePrefix + relation, Shard: from}
					next := resource
					next.Shard = to
					r.forwardResource(resource, next)
					break
				}
			}
		}
	}
}

func (r *replicationManager) findCatalogShard(resource mosiResource) *storageShard {
	if cell := r.lookupCell(resource); cell != nil {
		if shard := cell.current.Load(); shard != nil && shard.catalogOwner != "" {
			return shard
		}
	}
	relation, valid := catalogResourceRelation(resource)
	if !valid {
		return nil
	}
	db := databases.Get(resource.Database)
	if db == nil {
		return nil
	}
	db.bootstrapCatalog(nil)
	for _, shard := range systemSchemaTables[relation].activeTopology().shards {
		if shard != nil && shard.catalogOwner == db.Name && r.resourceKey(shard.replicationResource()) == r.resourceKey(resource) {
			return shard
		}
	}
	return nil
}

func (r *replicationManager) replaceCatalogShard(resource mosiResource, previous, replacement *storageShard) error {
	relation, valid := catalogResourceRelation(resource)
	db := databases.Get(resource.Database)
	if !valid || db == nil {
		return fmt.Errorf("missing catalog partition for shard publication")
	}
	replacement.catalogOwner = db.Name
	replacement.catalogHome = catalogBackendDatabase()
	replacement.catalogPrivate = false
	cell := r.resourceCell(resource)
	if previous != nil && previous != replacement {
		cell.retainShard(previous)
		if previous.uuid != replacement.uuid {
			r.forwardResource(resource, replacement.replicationResource())
			cell = cell.resolved()
		}
	}
	replacement.coherence.Store(cell)
	cell.current.Store(replacement)
	db.catalog.liveVersion.Add(1)
	// Native topology publishes payload independently of Go declaration objects.
	for _, current := range systemSchemaTables[relation].activeTopology().shards {
		if current != nil && current.catalogOwner == db.Name {
			if current == previous || current.uuid == replacement.uuid || r.resourceKey(current.replicationResource()) == r.resourceKey(resource) {
				var publication [catalogRelationCount]*storageShard
				publication[relation] = replacement
				publishCatalogShards(db.Name, &publication)
			}
			return nil
		}
	}
	var publication [catalogRelationCount]*storageShard
	publication[relation] = replacement
	publishCatalogShards(db.Name, &publication)
	return nil
}
