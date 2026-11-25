/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"
import "sync"
import "unsafe"
import "context"
import "sync/atomic"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// DROP records authority over an existing logical identity, never a replacement
// table with the same name. Payload retirement uses the native topology pins.
type catalogTableDropDeclaration struct {
	Name     string `json:"name"`
	Identity string `json:"identity"`
}

const tableDropLifecycleResourceKey = "native-table-drop-lifecycle"

// A transaction borrows this EXISTING database lifecycle capability only once.
// Repeated RLock while CLEAN waits would deadlock Go's writer-preferred RWMutex.
type catalogTableDropLifecycle struct {
	db          *database
	once        sync.Once
	transferred atomic.Bool
	bytes       uint
}

func (lifecycle *catalogTableDropLifecycle) release() {
	lifecycle.once.Do(func() {
		lifecycle.db.persistenceLifecycle.RUnlock()
		lifecycle.db.adjustMetadataMemory(lifecycle.bytes, 0)
	})
}
func (lifecycle *catalogTableDropLifecycle) Close() error {
	if !lifecycle.transferred.Load() {
		lifecycle.release()
	}
	return nil
}
func (tx *TxContext) tableDropLifecycle(db *database) *catalogTableDropLifecycle {
	value := tx.getOrComputeResource(tableDropLifecycleResourceKey, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		db.persistenceLifecycle.RLock()
		owner := &catalogTableDropLifecycle{db: db}
		owner.bytes = uint(unsafe.Sizeof(*owner)) + uint(unsafe.Sizeof(transactionResource{})) + 128
		db.adjustMetadataMemory(0, owner.bytes)
		return scm.NewAny(owner)
	})).Any().(*catalogTableDropLifecycle)
	if value.db != db {
		panic("table removal requires its original database")
	}
	return value
}

type catalogTableDropLease struct {
	table       *table
	ddlOnce     sync.Once
	lifecycle   *catalogTableDropLifecycle
	borrowedDDL bool
}

func (lease *catalogTableDropLease) releaseDDL() {
	lease.ddlOnce.Do(func() {
		if !lease.borrowedDDL {
			lease.table.ddlMu.Unlock()
		}
	})
}
func (lease *catalogTableDropLease) Close() {
	lease.releaseDDL()
}

type catalogTableDropAmendment struct {
	declaration catalogTableDropDeclaration
	table       *table
	lease       *catalogTableDropLease
	writer      bool // this amendment, rather than an earlier delta, acquired it
	bytes       uint
	retired     []replacedShardGeneration
	published   bool
}

func (tx *TxContext) hasTableDrops() bool {
	if changes := tx.publicCatalogChanges(); changes != nil {
		if views := changes.views.Load(); views != nil {
			return len(views.dropped) != 0
		}
	}
	return false
}

func (tx *TxContext) ownsTableDrop(t *table) bool {
	if changes := tx.publicCatalogChanges(); changes != nil {
		if views := changes.views.Load(); views != nil {
			_, found := views.dropped[t]
			return found
		}
	}
	return false
}

func (t *table) droppedAt(version uint64) bool {
	dropped := t.dropVersion.Load()
	return dropped != 0 && dropped <= version
}

// One input/metadata setup check; existing row readers retain their topology.
func (t *table) requireCatalogDataVisibility(tx *TxContext) {
	if tx.ownsTableDrop(t) || t.dropVersion.Load() != 0 && t.schema != nil && t.droppedAt(t.schema.catalogView().version) {
		panic("table is outside its native catalog lifetime: " + t.Name)
	}
}

func (tx *TxContext) dropCatalogTable(db *database, name string, ifexists bool) bool {
	if tx == nil || tx.transactionState == nil || tx.State != TxActive || tx.decisionStarted {
		panic("table removal requires an active native transaction")
	}
	finish := tx.beginCatalogConstruction(catalogDeclarationConstruction, nil)
	defer func() { finish(); tx.retireUnacceptedPublicCatalogPhase() }()
	t := db.GetTableForTx(name, false, tx)
	if t == nil {
		if ifexists {
			return false
		}
		panic("Table " + db.Name + "." + name + " does not exist")
	}
	prospective := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{
		tableDrop: &catalogTableDropAmendment{table: t, declaration: catalogTableDropDeclaration{t.Name, t.RuntimeIdentity}}})
	if err := tx.validateColumnAppendTableProgram(prospective, tx.catalogIntentTables()); err != nil {
		panic(err)
	}
	requireTableMaintenance(db.Name, name, maintenanceDrop)
	if t.Ephemeral || t.ScopedSchema || tx.ownsPendingTable(t) ||
		tx.coordinator != nil || sharedClusterDatabase(db.Name) || activeReplication(db) != nil {
		panic("transactional table removal requires a local committed public table")
	}
	if _, err := uuid.Parse(t.RuntimeIdentity); err != nil {
		panic("transactional table removal requires a durable native table identity")
	}
	// Match rebuild's existing lifecycle -> DDL order. Never wait for a builder
	// while retaining catalog publication/schema locks or another DDL owner.
	lifecycle := tx.tableDropLifecycle(db)
	borrowedDDL := tx.retainedColumnDDL(t)
	if !borrowedDDL && !t.ddlMu.TryLock() {
		panic("table removal conflicts with active native maintenance; retry transaction")
	}
	lease := &catalogTableDropLease{table: t, lifecycle: lifecycle, borrowedDDL: borrowedDDL}
	accepted := false
	defer func() {
		if !accepted {
			lease.Close()
		}
	}()
	result := tx.stagePublicCatalogDeclaration(db, catalogTableDropKind, nil, func(changes *publicCatalogChanges) bool {
		return changes.stageTableDrop(tx, t, lease)
	})
	accepted = result
	return result
}

func (changes *publicCatalogChanges) stageTableDrop(tx *TxContext, t *table, lease *catalogTableDropLease) bool {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	// Re-attest under the amendment owner: an append constructor can publish
	// after the caller's cold check while waiting for its existing DDL lease.
	prospective := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{
		tableDrop: &catalogTableDropAmendment{table: t, declaration: catalogTableDropDeclaration{t.Name, t.RuntimeIdentity}}})
	if err := tx.validateColumnAppendTableProgram(prospective, tx.catalogIntentTables()); err != nil {
		panic(err)
	}
	if tx.hasSurvivingColumnPropertiesExcept(t) {
		panic("table removal cannot join surviving column property amendments")
	}
	acquired := changes.leases[t] == nil
	accepted := false
	if acquired {
		// Reuse the validated constructor upgrade: only this original T may
		// retain an earlier framed writer; escaped callbacks still reject.
		writer, err := t.writeAdmission.admit(scanQueryContext(tx, context.Background()), tx.transactionState, tableValidatedDeclarationAdmission, false)
		if err != nil {
			panic(err)
		}
		changes.leases[t] = writer
		changes.leasesHighWater = max(changes.leasesHighWater, len(changes.leases))
	}
	defer func() {
		if !accepted && acquired {
			_ = changes.leases[t].Close()
			delete(changes.leases, t)
		}
	}()
	db := changes.db
	access := db.GetWrite(tx)
	defer access.Release()
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	if GetDatabase(db.Name) != db || t.schema != db || db.tables.Get(t.Name) != t ||
		!t.committedAt(db.catalogView().version) || tx.ownsTableDrop(t) {
		panic("table removal endpoint was replaced")
	}
	// Endpoint-changing FK programs need a joint declaration operation. This
	// profile never silently removes a surviving table's enforcement rights.
	if tx.tableHasSurvivingForeign(t) {
		panic("transactional table removal requires retired foreign key endpoints")
	}
	for _, entry := range tx.publicForeignAmendments() {
		if (tx.catalogIntents.Load() == nil && !tx.hasNameAmendments()) && (entry.publicKey != nil || entry.foreignDrop != nil || entry.foreign != nil || entry.check != nil || entry.checkDrop != nil || entry.constraintState != nil || entry.statementTrigger != nil) {
			panic("table removal cannot join enforcing declaration amendments")
		}
	}
	if t.topology.Load() == nil {
		panic("table removal requires its published native topology")
	}
	amendment := &catalogTableDropAmendment{declaration: catalogTableDropDeclaration{t.Name, t.RuntimeIdentity}, table: t, lease: lease, writer: acquired}
	amendment.bytes = uint(unsafe.Sizeof(*amendment)) + uint(unsafe.Sizeof(*lease))
	changes.tableDropBytes += amendment.bytes
	history := append(append([]catalogDeclarationAmendment(nil), tx.publicForeignAmendments()...), catalogDeclarationAmendment{tableDrop: amendment})
	changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	tx.catalogAmendments.Store(&history)
	changes.publishViews(history, maps.Clone(changes.views.Load().tables))
	accepted = true
	return true
}

func (changes *publicCatalogChanges) restoreTableDropLocked(amendment *catalogTableDropAmendment) {
	changes.tableDropBytes -= amendment.bytes
	amendment.lease.Close()
	amendment.lease = nil
	if amendment.writer {
		_ = changes.leases[amendment.table].Close()
		delete(changes.leases, amendment.table)
	}
}

func (tx *TxContext) publicTableDrops() []catalogTableDropDeclaration {
	var result []catalogTableDropDeclaration
	for _, entry := range tx.publicForeignAmendments() {
		if entry.tableDrop != nil {
			result = append(result, entry.tableDrop.declaration)
		}
	}
	return result
}

func (changes *publicCatalogChanges) validateTableDropsLocked() error {
	if changes.owner == nil {
		return fmt.Errorf("table removal lost its transaction owner")
	}
	history := changes.owner.catalogAmendments.Load()
	if history == nil {
		return nil
	}
	for _, entry := range *history {
		if drop := entry.tableDrop; drop != nil {
			t := drop.table
			if changes.db.tables.Get(drop.declaration.Name) != t || t.RuntimeIdentity != drop.declaration.Identity ||
				!t.committedAt(changes.db.catalogView().version) || (&TxContext{transactionState: changes.owner}).tableHasSurvivingForeign(t) || drop.lease == nil || changes.leases[t] == nil {
				return fmt.Errorf("table removal endpoint declaration changed")
			}
		}
	}
	return nil
}

// Caller owns the same native decision and schema publication. Per-table cuts
// become authoritative together at the subsequent catalog snapshot Store.
func (tx *TxContext) publishTableDropCutsLocked(version uint64) {
	for _, entry := range tx.publicForeignAmendments() {
		if drop := entry.tableDrop; drop != nil {
			drop.table.dropVersion.Store(version)
		}
	}
}

func (tx *TxContext) detachTableDropsLocked(version uint64) {
	for _, entry := range tx.publicForeignAmendments() {
		if drop := entry.tableDrop; drop != nil && !drop.published {
			t := drop.table
			if t.dropVersion.Load() != version || t.schema.tables.Get(drop.declaration.Name) != t {
				panic("table removal lost its published native identity")
			}
			t.schema.tables.Remove(drop.declaration.Name)
			terminal := &tableShardTopology{dropped: true}
			previous := t.topology.Swap(terminal)
			if previous != nil {
				drained := previous.retire()
				for _, shard := range previous.shards {
					if shard != nil {
						drop.retired = append(drop.retired, replacedShardGeneration{shard, drained, previous.retirementDependencies})
					}
				}
			}
			bytes := uint(cap(drop.retired)) * uint(unsafe.Sizeof(replacedShardGeneration{}))
			drop.bytes += bytes
			tx.publicCatalogChanges().tableDropBytes += bytes
			drop.published = true
			drop.lease.releaseDDL()
		}
	}
	if changes := tx.publicCatalogChanges(); changes != nil {
		changes.refreshMemoryLocked()
	}
}

// A committed checkpoint no longer references these payloads. Transfer the
// EXISTING lifecycle read capabilities until every pinned input/Tx has drained.
// All callbacks run after catalog, DDL, and persistence capabilities are gone.
func (tx *TxContext) finishTableDrops() {
	var drops []*catalogTableDropAmendment
	for _, entry := range tx.publicForeignAmendments() {
		if drop := entry.tableDrop; drop != nil && drop.lease != nil {
			drops = append(drops, drop)
		}
	}
	if len(drops) == 0 {
		return
	}
	lifecycle := drops[0].lease.lifecycle
	leases := make([]*catalogTableDropLease, len(drops))
	var transferred uint
	for i, drop := range drops {
		leases[i], drop.lease = drop.lease, nil
		transferred += drop.bytes
	}
	// Transfer only NEW retirement containers, never borrowed table payload.
	changes := tx.publicCatalogChanges()
	changes.tableDropBytes -= transferred
	changes.refreshMemoryLocked()
	bytes := transferred + uint(cap(drops)+cap(leases))*uint(unsafe.Sizeof((*table)(nil)))
	lifecycle.bytes += bytes
	lifecycle.db.adjustMetadataMemory(0, bytes)
	lifecycle.transferred.Store(true)
	cleanup := func() {
		defer lifecycle.release()
		defer func() {
			for _, lease := range leases {
				lease.Close()
			}
		}()
		// Drain ALL removed endpoints before any cleanup can fail and release
		// their shared CLEAN exclusion in the deferred unwinder.
		for _, drop := range drops {
			for _, generation := range drop.retired {
				generation.wait()
			}
		}
		for _, drop := range drops {
			for _, failure := range cleanupReplacedShardGenerations(drop.retired, "catalog DROP") {
				fmt.Println("native table retirement:", failure)
			}
			drop.table.mu.Lock()
			drop.table.releaseStatementTriggerMemory(drop.table.schema)
			drop.table.mu.Unlock()
			GlobalCache.Remove(drop.table)
			for _, column := range drop.table.Columns {
				if column.IsTemp {
					GlobalCache.Remove(column)
				}
			}
			for _, generation := range drop.retired {
				GlobalCache.Remove(generation.shard)
			}
		}
		for _, lease := range leases {
			lease.Close()
		}
		lifecycle.release()
		for _, drop := range drops {
			drop.table.ExecuteTableLifecycleTriggers(AfterDropTable, nil)
		}
	}
	ready := true
	for _, drop := range drops {
		for _, generation := range drop.retired {
			ready = ready && generation.ready()
		}
	}
	if ready {
		cleanup()
	} else {
		go func() {
			defer func() {
				if failure := recover(); failure != nil {
					fmt.Println("native table retirement callback:", failure)
				}
			}()
			cleanup()
		}()
	}
}

// Recovery runs before this database is published. Accepted logical DROP is
// replayed by identity; post-checkpoint orphan payloads remain explicit CLEAN's
// established responsibility, just like interrupted rebuild retirement.
func (db *database) recoverTableDrops(drops []catalogTableDropDeclaration) {
	for _, drop := range drops {
		t := db.tables.Get(drop.Name)
		if t == nil {
			panic("native table removal lacks its predecessor identity")
		}
		if t.RuntimeIdentity != drop.Identity {
			panic("native table removal conflicts with its predecessor identity")
		}
		db.tables.Remove(drop.Name)
	}
}
