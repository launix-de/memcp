/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
	"github.com/launix-de/memcp/storage/mosi"
)

// Definition leases belong to the actual transaction, rather than a parser or
// reusable executor. Getters still read immutable metadata without locks; one
// lease protects the definitions used by a complete transaction.
type schemaLease struct {
	db              *database
	write           bool
	access          WriteAccess
	dirty           bool
	previousPending bool
	reader          *mosi.ReadLease
	readLocked      bool
}

type transactionSchemas struct {
	mu          sync.Mutex
	leases      atomic.Pointer[map[*database]*schemaLease]
	undo        []schemaProjection
	maintenance map[*table]bool
}

// Nil at a native DDL boundary denotes its own autocommit operation. An
// explicit context always retains the caller's original transaction.
func schemaAutocommit(body func(*TxContext)) {
	tx := NewTxContext(TxACID)
	defer func() {
		if tx.State == TxActive && !tx.decisionStarted {
			tx.Rollback()
		}
	}()
	body(tx)
	if err := tx.Commit(); err != nil {
		panic(err)
	}
}

// Native declaration operators share one ownership/undo boundary. Calls from
// a frontend retain its context; context-free administration gets autocommit.
func schemaMutation[T any](tx *TxContext, tables []*table, body func(*TxContext) T) T {
	if tx == nil {
		for _, t := range tables {
			if t != nil && !t.RuntimeOnly {
				var result T
				schemaAutocommit(func(current *TxContext) { result = schemaMutation(current, tables, body) })
				return result
			}
		}
	}
	if tx != nil {
		seen := make(map[*table]bool)
		for _, t := range tables {
			if t != nil && !t.RuntimeOnly && !seen[t] {
				seen[t] = true
				tx.recordSchema(t, false)
			}
		}
	}
	return body(tx)
}

func (tx *TxContext) readSchema(db *database) {
	if tx == nil || tx.State != TxActive || db == nil || db == systemSchemaDB {
		return
	}
	scope := &tx.schemas
	if leases := scope.leases.Load(); leases != nil && (*leases)[db] != nil {
		return
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	previous := scope.leases.Load()
	if previous != nil && (*previous)[db] != nil {
		return
	}
	writer := false
	if previous != nil {
		for _, lease := range *previous {
			writer = writer || lease.write
		}
	}
	var reader *mosi.ReadLease
	for {
		if r := activeReplication(db); writer && r != nil && validReplicaDatabase(db.Name) && !r.resourceCell(mosiResource{Database: db.Name}).ownstate.ReadReady() {
			panic("schema access conflicts with another schema transaction; retry transaction")
		}
		db.GetRead(tx)
		db.definitionAdmission.Lock()
		if writer {
			if !db.definitionMu.TryRLock() {
				db.definitionAdmission.Unlock()
				panic("schema access conflicts with another schema transaction; retry transaction")
			}
		} else {
			db.definitionMu.RLock()
		}
		db.definitionAdmission.Unlock()
		if r := activeReplication(db); tx.coordinator == nil && r != nil && validReplicaDatabase(db.Name) {
			var wait <-chan struct{}
			reader, wait = r.resourceCell(mosiResource{Database: db.Name}).ownstate.TryRetainRead()
			if reader == nil {
				db.definitionMu.RUnlock()
				if writer {
					panic("schema access conflicts with a pending handover; retry transaction")
				}
				if wait != nil {
					ctx := scanQueryContext(tx, context.Background())
					select {
					case <-wait:
					case <-ctx.Done():
						panic(ctx.Err())
					}
				}
				continue
			}
		}
		break
	}
	next := make(map[*database]*schemaLease)
	if previous != nil {
		maps.Copy(next, *previous)
	}
	next[db] = &schemaLease{db: db, reader: reader, readLocked: true}
	scope.leases.Store(&next)
}

func (tx *TxContext) schemaLease(db *database) *schemaLease {
	if tx != nil {
		if leases := tx.schemas.leases.Load(); leases != nil {
			return (*leases)[db]
		}
	}
	return nil
}

func (tx *TxContext) hasSchemaWriter(db *database) bool {
	lease := tx.schemaLease(db)
	return lease != nil && lease.write
}

// Remote workers retain definitions only until their invocation has joined.
// The original coordinator owns the transaction's cluster-wide read proof.
func (tx *TxContext) releaseRemoteSchemaReads() {
	if tx.coordinator == nil {
		return
	}
	tx.schemas.mu.Lock()
	defer tx.schemas.mu.Unlock()
	tx.mu.Lock()
	idle := tx.remoteActive == 0
	tx.mu.Unlock()
	if !idle {
		return
	}
	if leases := tx.schemas.leases.Swap(nil); leases != nil {
		for _, lease := range *leases {
			if lease.readLocked {
				lease.db.definitionMu.RUnlock()
			}
			lease.reader.Release()
		}
	}
}

// The SQL frontend can assign its own object IDs to this stable incarnation.
// Renames and shard rebuilds retain it; replacement tables receive a new one.
// Legacy definitions without an identity explicitly report absence.
func (t *table) NativeIdentity(tx *TxContext) scm.Scmer {
	if t == nil || t.schema == nil {
		return scm.NewNil()
	}
	db := t.schema
	tx.readSchema(db)
	if db != systemSchemaDB && databases.Get(db.Name) != db {
		panic("native relation identity belongs to a replaced database")
	}
	if t.retired.Load() || db.tables.Get(t.clusterPublishedName()) != t {
		panic("native relation identity does not belong to the current definition")
	}
	if t.RuntimeIdentity == "" {
		return scm.NewNil()
	}
	return scm.NewString(t.RuntimeIdentity)
}

func (tx *TxContext) writeSchema(db *database) {
	if tx.State != TxActive || tx.decisionStarted {
		panic("schema mutation requires an active original transaction")
	}
	if tx.coordinator != nil {
		panic("logical schema changes must execute on the original transaction coordinator")
	}
	scope := &tx.schemas
	scope.mu.Lock()
	defer scope.mu.Unlock()
	previous := scope.leases.Load()
	var lease *schemaLease
	if previous != nil {
		lease = (*previous)[db]
	}
	if lease != nil && lease.write {
		return
	}
	// Do not wait for another transaction while retaining row/UNIQUE rights.
	// A conflicting reader makes this schema change retryable as a whole.
	db.definitionAdmission.Lock()
	if lease != nil && lease.readLocked {
		db.definitionMu.RUnlock()
		lease.readLocked = false
	}
	next := make(map[*database]*schemaLease)
	if previous != nil {
		maps.Copy(next, *previous)
	}
	// Keep a retained shared proof until admission succeeds or rollback ends.
	// Another DDL must not validate this transaction's uncommitted CS values.
	if !db.definitionMu.TryLock() {
		if lease != nil && !lease.readLocked {
			db.definitionMu.RLock()
			lease.readLocked = true
		}
		db.definitionAdmission.Unlock()
		panic("schema change conflicts with another transaction; retry transaction")
	}
	db.definitionAdmission.Unlock()
	next = maps.Clone(next)
	write := &schemaLease{db: db, write: true, previousPending: db.schemaExportPending.Load()}
	if lease != nil {
		write.reader = lease.reader
	}
	next[db] = write
	scope.leases.Store(&next)
	write.access = db.GetWrite(tx)
	write.reader.Release()
	write.reader = nil
	if !db.catalog.rootReady.Load() {
		// Migrate only accepted definitions before staging any DDL. Earlier
		// data work keeps its original transaction, snapshot and journals.
		db.schemalock.Lock()
		db.commitSchemaLocked(tx, true, db.schemalock.Unlock)
	}
}

func (tx *TxContext) schemaBoundary() int {
	tx.schemas.mu.Lock()
	defer tx.schemas.mu.Unlock()
	return len(tx.schemas.undo)
}

func (tx *TxContext) markSchemaDirty(db *database) {
	tx.schemas.mu.Lock()
	lease := (*tx.schemas.leases.Load())[db]
	if lease == nil || !lease.write {
		tx.schemas.mu.Unlock()
		panic("schema mutation requires the original transaction's definition lease")
	}
	lease.dirty = true
	tx.schemas.undo = append(tx.schemas.undo, schemaProjection{
		database: db, revisionOnly: true, revision: db.catalog.revision.Load(),
	})
	revision := uuid.NewString()
	db.catalog.revision.Store(&revision)
	db.schemaExportPending.Store(true)
	tx.schemas.mu.Unlock()
	// Definition revisions are catalog data and join the same native decision.
	// Only their immutable scalar projection is used by planner guards.
	var rows catalogRows
	_, rows[catalogDatabases] = catalogShardRecords(tx, db.catalog.live[catalogDatabases], catalogRelationColumns(catalogDatabases), nil)
	if len(rows[catalogDatabases]) != 1 {
		panic("schema revision requires one admitted database declaration")
	}
	rows[catalogDatabases][0][1] = scm.NewString(revision)
	db.editCatalogLocked(tx, &rows, 1<<catalogDatabases, nil)
}

func (tx *TxContext) schemaWriters() []*database {
	tx.schemas.mu.Lock()
	defer tx.schemas.mu.Unlock()
	var result []*database
	if leases := tx.schemas.leases.Load(); leases != nil {
		for db, lease := range *leases {
			if lease.write && lease.dirty {
				result = append(result, db)
			}
		}
	}
	return result
}

func (tx *TxContext) prepareSchemaCommit() {
	tx.prepareSchemaEngineChanges()
	for _, db := range tx.schemaWriters() {
		func() {
			db.schemalock.Lock()
			defer db.schemalock.Unlock()
			db.checkpointCatalogLocked(tx)
		}()
	}
}

func (tx *TxContext) exportCommittedSchemas() error {
	for _, db := range tx.schemaWriters() {
		if err := persistenceCall(func() {
			db.schemalock.Lock()
			db.commitSchemaLocked(tx, true, db.schemalock.Unlock)
		}); err != nil {
			return markCommitOutcomeUnknown(err)
		}
	}
	return nil
}

func (tx *TxContext) releaseSchemas() {
	scope := &tx.schemas
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if leases := scope.leases.Swap(nil); leases != nil {
		for _, lease := range *leases {
			if lease.write && tx.State == TxAborted {
				lease.db.schemaExportPending.Store(lease.previousPending)
			}
			lease.access.Release()
			if lease.write {
				lease.db.definitionMu.Unlock()
			} else if lease.readLocked {
				lease.db.definitionMu.RUnlock()
			}
			lease.reader.Release()
		}
	}
	scope.undo = nil
	for t := range scope.maintenance {
		t.maintenanceMu.Unlock()
	}
	scope.maintenance = nil
}

func (tx *TxContext) claimSchemaMaintenance(t *table) {
	tx.schemas.mu.Lock()
	defer tx.schemas.mu.Unlock()
	if tx.schemas.maintenance[t] {
		return
	}
	if !t.maintenanceMu.TryLock() {
		panic("engine change conflicts with table maintenance; retry transaction")
	}
	if tx.schemas.maintenance == nil {
		tx.schemas.maintenance = make(map[*table]bool)
	}
	tx.schemas.maintenance[t] = true
}

// All statements keep using their original physical transaction journals.
// Build a persistent successor only after the script has finished its DML;
// native commit then forwards the original decision to that successor.
func (tx *TxContext) prepareSchemaEngineChanges() {
	tx.schemas.mu.Lock()
	changes := append([]schemaProjection(nil), tx.schemas.undo...)
	tx.schemas.mu.Unlock()
	seen := make(map[*table]bool)
	for _, before := range changes {
		t := before.table
		if !before.engine || seen[t] || t.retired.Load() || (before.mode == Safe || before.mode == Logged) || (t.PersistencyMode != Safe && t.PersistencyMode != Logged) {
			continue
		}
		seen[t] = true
		if t.schema.tables.Get(t.Name) != t {
			continue // this original transaction also dropped the table
		}
		shards := append([]*storageShard(nil), t.ActiveShards()...)
		for i, shard := range shards {
			shards[i] = shard.rebuild(true)
		}
		t.mu.Lock()
		if t.ShardMode == ShardModePartition {
			t.PShards = shards
		} else {
			t.Shards = shards
		}
		t.publishTopologyLocked()
		t.mu.Unlock()
		t.schema.persistence.SyncNamespace()
	}
}

// Only declaration containers are captured. Data rows and native transaction
// journals are never copied or replaced by schema rollback.
type schemaProjection struct {
	database       *database
	revisionOnly   bool
	revision       *string
	procedureName  string
	procedure      *ProcedureDescription
	table          *table
	created        bool
	dropped        bool
	engine         bool
	name           string
	columns        []*column
	unique         []uniqueKey
	foreign        []foreignKey
	triggers       []TriggerDescription
	retiredColumns map[string]bool
	collation      string
	charset        string
	comment        string
	mode           PersistencyMode
	initializer    *scm.Scmer
	shards         []schemaShardProjection
}

func (tx *TxContext) recordProcedure(db *database, name string) {
	tx.writeSchema(db)
	db.schemalock.RLock()
	before := db.procedures.Get(name)
	db.schemalock.RUnlock()
	tx.schemas.mu.Lock()
	tx.schemas.undo = append(tx.schemas.undo, schemaProjection{database: db, procedureName: name, procedure: before})
	tx.schemas.mu.Unlock()
}

type schemaShardProjection struct {
	shard   *storageShard
	columns map[string]ColumnStorage
	delta   map[string]int
	logfile PersistenceLogfile
}

func (tx *TxContext) recordSchema(t *table, created bool) int {
	if tx == nil || t.RuntimeOnly {
		return -1
	}
	tx.writeSchema(t.schema)
	projection := schemaProjection{table: t, created: created, name: t.Name}
	if !created {
		t.schema.schemalock.Lock()
		defer t.schema.schemalock.Unlock()
		projection.columns = append([]*column(nil), t.Columns...)
		projection.unique = append([]uniqueKey(nil), t.Unique...)
		projection.foreign = append([]foreignKey(nil), t.Foreign...)
		projection.retiredColumns = maps.Clone(t.DroppedColumns)
		projection.collation, projection.charset, projection.comment = t.Collation, t.Charset, t.Comment
		projection.mode, projection.initializer = t.PersistencyMode, t.OnInit.Load()
		t.mu.Lock()
		projection.triggers = append([]TriggerDescription(nil), t.Triggers...)
		t.mu.Unlock()
		for _, shard := range t.ActiveShards() {
			if shard == nil {
				continue
			}
			shard.mu.RLock()
			projection.shards = append(projection.shards, schemaShardProjection{
				shard: shard, columns: maps.Clone(shard.columns), delta: maps.Clone(shard.deltaColumns), logfile: shard.logfile,
			})
			shard.mu.RUnlock()
		}
	}
	tx.schemas.mu.Lock()
	index := len(tx.schemas.undo)
	tx.schemas.undo = append(tx.schemas.undo, projection)
	tx.schemas.mu.Unlock()
	return index
}

func (tx *TxContext) recordEngineChange(t *table) {
	tx.schemas.mu.Lock()
	defer tx.schemas.mu.Unlock()
	for i := len(tx.schemas.undo) - 1; i >= 0; i-- {
		if tx.schemas.undo[i].table == t {
			tx.schemas.undo[i].engine = true
			return
		}
	}
	panic("engine change requires the original schema undo boundary")
}

// Destructive engine transitions follow the accepted catalog/legacy export.
// Until then all predecessor files and WAL handles remain recoverable.
func (tx *TxContext) finishSchemaEngineChanges() {
	tx.schemas.mu.Lock()
	changes := append([]schemaProjection(nil), tx.schemas.undo...)
	tx.schemas.mu.Unlock()
	seen := make(map[*table]bool)
	for _, before := range changes {
		t := before.table
		if !before.engine || seen[t] || t.retired.Load() {
			continue
		}
		seen[t] = true
		mode := t.PersistencyMode
		wasPersisted := before.mode != Memory && before.mode != Cache
		isPersisted := mode != Memory && mode != Cache
		// Promotion already built a native persistent generation before commit.
		if !wasPersisted && isPersisted || mode == before.mode {
			continue
		}
		for _, shard := range t.ActiveShards() {
			shard.mu.Lock()
			func() {
				defer shard.mu.Unlock()
				transitionShardEngine(shard, before.mode, mode)
			}()
		}
	}
}

func (tx *TxContext) recordDrop(index int) {
	tx.schemas.mu.Lock()
	defer tx.schemas.mu.Unlock()
	tx.schemas.undo[index].dropped = true
}

// Both endpoint copies and their generated trigger programs belong to the
// same declaration change. The caller already owns the definition writer.
func (db *database) relatedSchemaTables(name string) []*table {
	var result []*table
	for _, candidate := range db.tables.GetAll() {
		if candidate.Name == name {
			continue
		}
		for _, fk := range candidate.Foreign {
			if fk.Tbl1 == name || fk.Tbl2 == name {
				result = append(result, candidate)
				break
			}
		}
	}
	return result
}

func (tx *TxContext) finishSchemaDrops() {
	tx.schemas.mu.Lock()
	var tables []*table
	seen := make(map[*table]bool)
	for _, change := range tx.schemas.undo {
		if change.dropped && !seen[change.table] {
			seen[change.table] = true
			tables = append(tables, change.table)
		}
	}
	tx.schemas.mu.Unlock()
	for _, t := range tables {
		finishDroppedTable(t)
	}
}

func (tx *TxContext) retireSchemaDrops() {
	tx.schemas.mu.Lock()
	defer tx.schemas.mu.Unlock()
	for _, change := range tx.schemas.undo {
		if change.dropped {
			change.table.retired.Store(true)
		}
	}
}

func (tx *TxContext) restoreSchemas(boundary int) {
	scope := &tx.schemas
	scope.mu.Lock()
	undo := append([]schemaProjection(nil), scope.undo[boundary:]...)
	scope.undo = scope.undo[:boundary]
	scope.mu.Unlock()
	var abandoned []*table
	for i := len(undo) - 1; i >= 0; i-- {
		before := &undo[i]
		if before.database != nil {
			db := before.database
			if before.revisionOnly {
				db.catalog.revision.Store(before.revision)
				continue
			}
			db.schemalock.Lock()
			if before.procedure == nil {
				db.procedures.Remove(before.procedureName)
			} else {
				db.procedures.Set(before.procedureName, before.procedure)
			}
			db.schemalock.Unlock()
			continue
		}
		t, db := before.table, before.table.schema
		db.schemalock.Lock()
		if db.tables.Get(t.Name) == t {
			db.tables.Remove(t.Name)
		}
		if !before.created {
			t.Name = before.name
			t.publishedName.Store(&before.name)
			t.Columns, t.Unique, t.Foreign = before.columns, before.unique, before.foreign
			t.DroppedColumns = before.retiredColumns
			t.Collation, t.Charset, t.Comment = before.collation, before.charset, before.comment
			t.PersistencyMode = before.mode
			t.OnInit.Store(before.initializer)
			t.mu.Lock()
			t.Triggers = before.triggers
			t.mu.Unlock()
			for _, stored := range before.shards {
				stored.shard.mu.Lock()
				if before.engine && before.mode != Safe && before.mode != Logged && stored.logfile != stored.shard.logfile {
					if stored.shard.logfile != nil {
						stored.shard.logfile.Close()
					}
					stored.shard.logfile = stored.logfile
				}
				stored.shard.columns, stored.shard.deltaColumns = stored.columns, stored.delta
				stored.shard.mu.Unlock()
			}
			t.publishShowColumnsSnapshot(true)
			db.tables.Set(t.Name, t)
		} else {
			t.retired.Store(true)
			abandoned = append(abandoned, t)
		}
		db.publishTableNamesLocked()
		db.schemalock.Unlock()
	}
	// All definitions are restored before releasing cache/WAL resources.
	// These UUIDs were never admitted by the transaction; orphan collection
	// can reclaim their files without touching a predecessor's persistence.
	for _, t := range abandoned {
		GlobalCache.Remove(t)
		for _, c := range t.Columns {
			if c.IsTemp {
				GlobalCache.Remove(c)
			}
		}
		for _, shard := range t.ActiveShards() {
			if shard == nil {
				continue
			}
			GlobalCache.Remove(shard)
			shard.mu.Lock()
			if shard.logfile != nil {
				if err := persistenceCall(shard.logfile.Close); err != nil {
					fmt.Println("unpublished schema WAL cleanup:", err)
				}
				shard.logfile = nil
			}
			shard.mu.Unlock()
		}
	}
}
