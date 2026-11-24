/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "maps"
import "sync"
import "context"
import "sync/atomic"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

// Definition admission stops new transactions before a writer waits. Existing
// physical maintenance may still nest a read capability while finishing its
// checkpoint; unlike sync.RWMutex, a queued writer must not block that nesting.
// This lock is used only at transaction/maintenance boundaries, never per row.
type definitionMutex struct {
	mu      sync.Mutex
	readers int
	writer  bool
	changed chan struct{}
}

func (m *definitionMutex) waitLocked() <-chan struct{} {
	if m.changed == nil {
		m.changed = make(chan struct{})
	}
	return m.changed
}

func (m *definitionMutex) notifyLocked() {
	if m.changed != nil {
		close(m.changed)
		m.changed = nil
	}
}

func (m *definitionMutex) RLock() {
	m.mu.Lock()
	for m.writer {
		wait := m.waitLocked()
		m.mu.Unlock()
		<-wait
		m.mu.Lock()
	}
	m.readers++
	m.mu.Unlock()
}

func (m *definitionMutex) TryRLock() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.writer {
		return false
	}
	m.readers++
	return true
}

func (m *definitionMutex) RUnlock() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.readers == 0 {
		panic("definition read released without ownership")
	}
	m.readers--
	if m.readers == 0 {
		m.notifyLocked()
	}
}

func (m *definitionMutex) TryLock() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.writer || m.readers != 0 {
		return false
	}
	m.writer = true
	return true
}

func (m *definitionMutex) LockContext(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for m.writer || m.readers != 0 {
		wait := m.waitLocked()
		m.mu.Unlock()
		select {
		case <-wait:
		case <-ctx.Done():
			m.mu.Lock()
			return ctx.Err()
		}
		m.mu.Lock()
	}
	m.writer = true
	return nil
}

func (m *definitionMutex) Unlock() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.writer {
		panic("definition write released without ownership")
	}
	m.writer = false
	m.notifyLocked()
}

// Definition leases belong to the actual transaction, rather than a parser or
// reusable executor. Getters still read immutable metadata without locks; one
// lease protects the definitions used by a complete transaction.
type schemaLease struct {
	db              *database
	write           bool
	access          WriteAccess
	dirty           bool
	previousPending bool
	reader          atomic.Pointer[mosi.ReadLease]
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
	tx.autoCommit = true
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
		tx.refreshSchemaAuthority((*leases)[db])
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
		db.readAuthority(tx)
		// A queued DDL has not changed the definitions. Readers must still
		// join existing transactions: one session can need another session's
		// read before it releases the final lease awaited by that DDL.
		// definitionMu excludes readers once publication actually owns it;
		// definitionAdmission serializes only competing writer upgrades.
		if writer {
			if !db.definitionMu.TryRLock() {
				panic("schema access conflicts with another schema transaction; retry transaction")
			}
		} else {
			db.definitionMu.RLock()
		}
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
	lease := &schemaLease{db: db, readLocked: true}
	lease.reader.Store(reader)
	next[db] = lease
	scope.leases.Store(&next)
}

// A parked transaction retains its local definition, not MOSI authority over
// physical topology. Renew the shared authority for each executing statement
// and for the decision. Changed incoming definitions must exclude that local
// projection before publication; an incompatible parked transaction aborts.
func (tx *TxContext) refreshSchemaAuthority(lease *schemaLease) {
	if lease.write || tx.coordinator != nil || lease.reader.Load() != nil {
		return
	}
	r := activeReplication(lease.db)
	if r == nil || !validReplicaDatabase(lease.db.Name) {
		return
	}
	tx.schemas.mu.Lock()
	defer tx.schemas.mu.Unlock()
	if lease.reader.Load() != nil {
		return
	}
	for {
		lease.db.readAuthority(tx)
		reader, wait := r.resourceCell(mosiResource{Database: lease.db.Name}).ownstate.TryRetainRead()
		if reader != nil {
			lease.reader.Store(reader)
			return
		}
		if wait != nil {
			ctx := scanQueryContext(tx, context.Background())
			select {
			case <-wait:
			case <-ctx.Done():
				panic(ctx.Err())
			}
		}
	}
}

func (tx *TxContext) releaseStatementSchemaAuthority() {
	tx.schemas.mu.Lock()
	defer tx.schemas.mu.Unlock()
	if leases := tx.schemas.leases.Load(); leases != nil {
		for _, lease := range *leases {
			if !lease.write {
				lease.reader.Swap(nil).Release()
			}
		}
	}
}

func (tx *TxContext) refreshSchemaAuthorities() {
	if leases := tx.schemas.leases.Load(); leases != nil {
		for _, lease := range *leases {
			tx.refreshSchemaAuthority(lease)
		}
	}
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
			lease.reader.Swap(nil).Release()
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
	tx.readSchema(db)
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
	waitable := tx.autoCommit && (previous == nil || len(*previous) <= 1) && tx.shards.Load() == nil && tx.readViews.Load() == nil
	if waitable && lease != nil {
		// A fresh declaration has no earlier row effects. Relinquish its
		// compile-time read before joining the writer queue: another queued
		// declaration must never wait for a reader which is itself upgrading.
		if lease.readLocked {
			db.definitionMu.RUnlock()
			lease.readLocked = false
		}
		lease.reader.Swap(nil).Release()
	}
	// Do not wait for another transaction while retaining row/UNIQUE rights.
	// A conflicting reader makes this schema change retryable as a whole.
	if waitable {
		db.definitionAdmission.Lock()
	} else if !db.definitionAdmission.TryLock() {
		panic("schema change conflicts with pending definition admission; retry transaction")
	}
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
	acquired := db.definitionMu.TryLock()
	var waitError error
	// A fresh autocommit declaration holds no data/generation rights which
	// another reader or maintenance operation might need before it can finish.
	// Wait there; a mixed script or explicit transaction must instead retry.
	if !acquired && waitable {
		waitError = db.definitionMu.LockContext(scanQueryContext(tx, context.Background()))
		acquired = waitError == nil
	}
	if !acquired {
		if lease != nil && !lease.readLocked {
			db.definitionMu.RLock()
			lease.readLocked = true
		}
		db.definitionAdmission.Unlock()
		if waitError != nil {
			panic(waitError)
		}
		panic("schema change conflicts with another transaction; retry transaction")
	}
	db.definitionAdmission.Unlock()
	next = maps.Clone(next)
	write := &schemaLease{db: db, write: true, previousPending: db.schemaExportPending.Load()}
	if lease != nil {
		write.reader.Store(lease.reader.Swap(nil))
	}
	next[db] = write
	scope.leases.Store(&next)
	write.access = db.GetWrite(tx)
	write.reader.Swap(nil).Release()
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
}

// Frontends may own declarations stored in ordinary relations, such as SQL
// views. Their explicit publication epoch joins those row changes in this same
// transaction. Native catalog edits derive their epoch from the actual rows.
func (tx *TxContext) markExternalSchemaDirty(db *database) {
	tx.markSchemaDirty(db)
	var rows catalogRows
	_, rows[catalogDatabases] = catalogShardRecords(tx, db.catalog.live[catalogDatabases], catalogRelationColumns(catalogDatabases), nil)
	if len(rows[catalogDatabases]) != 1 {
		panic("schema revision requires one admitted database declaration")
	}
	rows[catalogDatabases][0][1] = scm.NewString(*db.catalog.revision.Load())
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
	tables := make(map[*database][]*table)
	seen := make(map[*table]bool)
	add := func(t *table) {
		if t != nil && !t.RuntimeOnly && !seen[t] {
			seen[t] = true
			tables[t.schema] = append(tables[t.schema], t)
		}
	}
	tx.schemas.mu.Lock()
	for _, before := range tx.schemas.undo {
		add(before.table)
	}
	tx.schemas.mu.Unlock()
	tx.mu.Lock()
	for shard := range tx.shardTransactions() {
		add(shard.t)
	}
	tx.mu.Unlock()
	for _, db := range tx.schemaWriters() {
		if len(tables[db]) == 0 {
			continue
		}
		func() {
			db.schemalock.Lock()
			defer db.schemalock.Unlock()
			db.checkpointCatalogLocked(tx, tables[db]...)
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
			lease.reader.Swap(nil).Release()
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
		panic("schema change conflicts with table maintenance; retry transaction")
	}
	if tx.schemas.maintenance == nil {
		tx.schemas.maintenance = make(map[*table]bool)
	}
	tx.schemas.maintenance[t] = true
}

func (tx *TxContext) ownsSchemaMaintenance(t *table) bool {
	if tx == nil {
		return false
	}
	tx.schemas.mu.Lock()
	defer tx.schemas.mu.Unlock()
	return tx.schemas.maintenance[t]
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
	tx.claimSchemaMaintenance(t)
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
