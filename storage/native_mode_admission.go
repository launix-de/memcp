/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "sort"
import "sync"
import "maps"

// Match exactly the adapter that native schema writes actually use. System
// bootstrap is intentionally excluded by the existing native database policy.
func effectiveSchemaReplication(db *database) *replicationManager {
	if !validReplicaDatabase(db.Name) {
		return nil
	}
	return activeReplication(db)
}

// This is a lifetime pin in the existing database owner, not another catalog
// or transaction registry. Resource Close releases it only after endpoint gates.
// Namespace/schema admission publishes all increments; final decrement is
// lock-free and can only remove an already completed native owner.
type databaseLocalDeclarationLease struct {
	db          *database
	maintenance bool
	once        sync.Once
}

func (lease *databaseLocalDeclarationLease) Close() error {
	lease.once.Do(func() {
		db := lease.db
		var remaining int64
		if lease.maintenance {
			remaining = db.localCatalogMaintenance.Add(-1)
		} else {
			remaining = db.localCatalogPins.Add(-1)
		}
		if remaining < 0 {
			panic("native database declaration pin lost its owner")
		}
		lease.db = nil
	})
	return nil
}

// Match DROP's schema-before-namespace order. Ordinary schema contention must
// finish before taking namespaceMu, rather than reject an authorized incoming
// schema packet. Mode publication takes namespaceMu but only tries schema locks.
func (db *database) admitLocalCatalog(maintenance bool) *databaseLocalDeclarationLease {
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	databases.namespaceMu.Lock()
	defer databases.namespaceMu.Unlock()
	if databases.ReadMap.Get(db.Name) != db {
		panic("native database declaration owner was replaced")
	}
	if maintenance {
		if db.localCatalogPins.Load() != 0 {
			panic("native database has pending declarations; retry maintenance")
		}
		db.localCatalogMaintenance.Add(1)
	} else {
		if replicationStarting.Load() || replication.Load() != nil || sharedClusterDatabase(db.Name) {
			panic("public foreign declaration requires distributed table admission")
		}
		if db.localCatalogMaintenance.Load() != 0 {
			panic("native database declaration conflicts with active maintenance; retry statement")
		}
		db.localCatalogPins.Add(1)
	}
	return &databaseLocalDeclarationLease{db: db, maintenance: maintenance}
}

// Caller owns namespaceMu. This visits only database identities and numeric
// owner facts: no cold schema loads, table scans, callbacks or network grants.
// All schema acquisition is nonblocking; every failed prefix is unwound before
// any mode guard, configuration or externally reachable role is published.
func acquireModePublicationLocked(names map[string]bool) (func(), error) {
	dbs := databases.ReadMap.GetAll()
	sort.Slice(dbs, func(i, j int) bool { return dbs[i].Name < dbs[j].Name })
	var locked []*database
	release := func() {
		for i := len(locked) - 1; i >= 0; i-- {
			locked[i].schemalock.Unlock()
			locked[i].catalogPublicationMu.Unlock()
		}
	}
	accepted := false
	defer func() {
		if !accepted {
			release()
		}
	}()
	for _, db := range dbs {
		if names != nil && !names[db.Name] {
			continue
		}
		if !db.catalogPublicationMu.TryLock() {
			return nil, fmt.Errorf("native mode activation conflicts with catalog publication; retry startup")
		}
		if !db.schemalock.TryLock() {
			db.catalogPublicationMu.Unlock()
			return nil, fmt.Errorf("native mode activation conflicts with schema maintenance; retry startup")
		}
		locked = append(locked, db)
		if db.localCatalogPins.Load() != 0 || db.localCatalogMaintenance.Load() != 0 {
			return nil, fmt.Errorf("native mode activation conflicts with pending declarations or maintenance; retry after the transaction finishes")
		}
	}
	accepted = true
	return release, nil
}

// Configured mode is fail-closed even if later private initialization fails.
// No automatic abort restores local authority after a role might be reachable.
func prepareReplicationMode() error {
	databases.namespaceMu.Lock()
	defer databases.namespaceMu.Unlock()
	release, err := acquireModePublicationLocked(nil)
	if err != nil {
		return err
	}
	defer release()
	replicationStarting.Store(true)
	return nil
}

func beginReplicationStart() (func(), error) {
	databases.namespaceMu.Lock()
	defer databases.namespaceMu.Unlock()
	if replication.Load() != nil || replicationConstructing {
		return nil, fmt.Errorf("replication is already initialized or initializing")
	}
	release, err := acquireModePublicationLocked(nil)
	if err != nil {
		return nil, err
	}
	defer release()
	replicationConstructing = true
	replicationStarting.Store(true)
	return func() {
		databases.namespaceMu.Lock()
		replicationConstructing = false
		databases.namespaceMu.Unlock()
	}, nil
}

// The scan role and optional cluster manager share one native service lifetime.
// A scalar claim prevents duplicate private constructors without holding locks
// across compilation, I/O, callbacks or transport registry publication.
func beginNodeServiceStart() (func(), error) {
	databases.namespaceMu.Lock()
	defer databases.namespaceMu.Unlock()
	if nodeScans.Load() != nil || nodeServiceConstructing {
		return nil, fmt.Errorf("node protocol is already initialized or initializing")
	}
	nodeServiceConstructing = true
	return func() {
		databases.namespaceMu.Lock()
		nodeServiceConstructing = false
		databases.namespaceMu.Unlock()
	}, nil
}

func publishSharedClusterMounts(mounted map[string]bool) error {
	databases.namespaceMu.Lock()
	defer databases.namespaceMu.Unlock()
	if previous := sharedClusterMounts.Load(); previous != nil && !maps.Equal(*previous, mounted) {
		return fmt.Errorf("shared database mounts are already configured")
	}
	release, err := acquireModePublicationLocked(mounted)
	if err != nil {
		return err
	}
	defer release()
	sharedClusterMounts.Store(&mounted)
	return nil
}
