/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import "os"
import "fmt"
import "sort"
import "sync"
import "time"
import "runtime"
import "strings"
import "reflect"
import "sync/atomic"
import "encoding/json"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/NonLockingReadMap"
import "github.com/launix-de/memcp/storage/mosi"

type database struct {
	Name        string                                     `json:"name"`
	persistence PersistenceEngine                          `json:"-"`
	tables      *NonLockingReadMap.ReadMap[string, *table] `json:"-"`
	// Name-only metadata, rebuilt under schemalock at namespace publication.
	// Readers never retain tables or traverse the catalog. An empty spelling
	// records an ambiguous folded name; exact-case frontends are unaffected.
	foldedTableNames atomic.Pointer[map[string]string] `json:"-"`
	// Immutable source definitions use the same lock-free catalog as tables.
	// DDL publishes under schemalock; compiled code is transient per definition.
	procedures *NonLockingReadMap.ReadMap[string, *ProcedureDescription] `json:"-"`
	// loadOnce is the database-wide lazy-load barrier. The MySQL and HTTP
	// listeners may issue the first queries concurrently; none may observe a
	// partially decoded table catalog.
	loadOnce sync.Once `json:"-"`
	// schemalock protects only database-local schema membership and schema.json
	// snapshots. It must never cover long rebuild/repartition/blob work. The
	// lock order continues with table.ddlMu -> table.mu -> shard.mu.
	schemalock sync.RWMutex `json:"-"`
	// saveMu/ saveCond serialize schema.json commits per database.
	// Software contract:
	//   - every caller still waits until its schema mutation is persisted
	//   - concurrent callers may coalesce into one later snapshot write
	//     instead of forcing N full schema.json rewrites in sequence
	//   - durable=true (Safe engine metadata) upgrades the coalesced write to
	//     a fully synced commit; non-durable callers may piggyback on it
	saveMu          sync.Mutex    `json:"-"`
	saveCondOnce    sync.Once     `json:"-"`
	saveCond        *sync.Cond    `json:"-"`
	saveRequested   uint64        `json:"-"`
	saveCompleted   uint64        `json:"-"`
	saveInFlight    bool          `json:"-"`
	savePending     []byte        `json:"-"`
	savePendingSync bool          `json:"-"`
	savePanic       any           `json:"-"`
	schemaDirty     atomic.Bool   `json:"-"`
	blobRefs        *blobRefState `json:"-"`
	// persistenceLifecycle protects storage generations from rebuild through
	// publication. Rebuild/repartition and short catalog mutations take a read
	// capability; cleanup and backend migration take the exclusive capability.
	// Query and DML paths do not participate.
	persistenceLifecycle sync.RWMutex `json:"-"`
	// transactionLog is the database-wide commit authority for transactional
	// shard WAL entries. Shard logs may contain prepared records after a crash;
	// recovery exposes them only when this log contains their durable commit.
	transactionOnce sync.Once          `json:"-"`
	transactionMu   sync.RWMutex       `json:"-"`
	transactionLog  PersistenceLogfile `json:"-"`
	committedTx     map[string]uint64  `json:"-"`
	transactionGen  uint64             `json:"-"`

	// lazy-loading/shared-resource state (not serialized)
	srState SharedState `json:"-"`
}

// blobRefLock occupies one cache line so unrelated hash stripes do not
// invalidate each other's lock state under parallel blob rebuilds.
type blobRefLock struct {
	mu sync.Mutex
	_  [56]byte
}

type blobRefState struct {
	tableMu sync.Mutex
	rows    sync.RWMutex
	locks   [64]blobRefLock
	table   atomic.Pointer[table]
}

func (db *database) blobRefState() *blobRefState {
	return db.blobRefs
}

func newDatabase() *database {
	db := &database{
		tables:      NonLockingReadMap.NewReadMap[string, *table](),
		procedures:  NonLockingReadMap.NewReadMap[string, *ProcedureDescription](),
		blobRefs:    new(blobRefState),
		committedTx: make(map[string]uint64),
	}
	return db
}

const transactionLogName = ".transactions"

// Cluster nodes own independent native commit streams in the shared backend.
// Warm handover rebuilds drain native transactions. Cold references can still
// replay a durable WAL written by an earlier owner, so acquisition merges the
// known installations' native commit authority before loading that generation.
// Reboot retains the installation UUID and its ordinary commit stream.
// This is native engine persistence, never a MOSI journal or backend override.
func (db *database) transactionLogKey() string {
	if !sharedClusterDatabase(db.Name) {
		return transactionLogName
	}
	id, err := localNodeIdentity()
	if err != nil {
		panic(err)
	}
	return transactionLogName + "." + id
}

func (db *database) initializeTransactionLog() {
	db.transactionOnce.Do(func() {
		committed := make(map[string]uint64)
		if sharedClusterDatabase(db.Name) {
			// A standalone database may enter the cluster with a valid native
			// WAL. Preserve its existing commit authority without rewriting it.
			_, legacy, handle := db.persistence.ReplayLog(transactionLogName)
			for entry := range legacy {
				if commit, ok := entry.(LogEntryCommit); ok && commit.txID != "" {
					committed[commit.txID] = 0
				}
			}
			handle.Close()
		}
		_, entries, logfile := db.persistence.ReplayLog(db.transactionLogKey())
		for entry := range entries {
			commit, ok := entry.(LogEntryCommit)
			if !ok || commit.txID == "" {
				panic("invalid entry in transaction commit log")
			}
			committed[commit.txID] = 0
		}
		db.transactionMu.Lock()
		db.committedTx = committed
		db.transactionLog = logfile
		db.transactionMu.Unlock()
	})
}

func (db *database) transactionCommitted(txID string) bool {
	if txID == "" {
		return true
	}
	db.initializeTransactionLog()
	db.transactionMu.RLock()
	_, committed := db.committedTx[txID]
	db.transactionMu.RUnlock()
	return committed
}

func (db *database) commitTransaction(txID string, durable bool) error {
	if txID == "" {
		return nil
	}
	db.initializeTransactionLog()
	db.transactionMu.Lock()
	defer db.transactionMu.Unlock()
	if _, exists := db.committedTx[txID]; exists {
		return nil
	}
	if err := persistenceCall(func() { db.transactionLog.Write(LogEntryCommit{txID: txID}) }); err != nil {
		return err
	}
	if err := persistenceCall(func() { db.transactionLog.Flush(durable) }); err != nil {
		// The authority frame was accepted into the logfile. A failed local
		// durability barrier or remote publication cannot tell us whether it
		// will survive a crash, so memory must keep treating it as committed.
		db.committedTx[txID] = db.transactionGen
		return markCommitOutcomeUnknown(err)
	}
	db.committedTx[txID] = db.transactionGen
	return nil
}

func (db *database) transactionCompactionSnapshot() (uint64, bool) {
	db.initializeTransactionLog()
	db.transactionMu.Lock()
	cutoff := db.transactionGen
	hasTransactions := len(db.committedTx) != 0
	db.transactionGen++
	db.transactionMu.Unlock()
	return cutoff, hasTransactions
}

func (db *database) compactTransactionLog(cutoff uint64) {
	// A shared generation may still replay this installation's committed WAL
	// on another node. Retain native commit authority until distributed WAL
	// retirement proves that no historical generation needs these entries.
	if sharedClusterDatabase(db.Name) {
		return
	}

	db.initializeTransactionLog()
	db.transactionMu.Lock()
	defer db.transactionMu.Unlock()

	// Usually only the small set committed while the rebuild was running is
	// retained. Do not reserve capacity for the much larger obsolete set.
	var retainedIDs []string
	for txID, generation := range db.committedTx {
		if generation > cutoff {
			retainedIDs = append(retainedIDs, txID)
		}
	}
	sort.Strings(retainedIDs)
	entries := make([]interface{}, len(retainedIDs))
	retained := make(map[string]uint64, len(retainedIDs))
	for i, txID := range retainedIDs {
		entries[i] = LogEntryCommit{txID: txID}
		retained[txID] = db.committedTx[txID]
	}

	// Committers share transactionMu, so nobody can append between this
	// barrier and publishing the replacement log.
	db.transactionLog.Flush(true)
	replacement := db.persistence.SwapLog(db.transactionLogKey(), entries, true)
	old := db.transactionLog
	db.transactionLog = replacement
	db.committedTx = retained
	old.Close()
}

func (db *database) closeTransactionLog() {
	db.transactionMu.Lock()
	defer db.transactionMu.Unlock()
	if db.transactionLog != nil {
		db.transactionLog.Close()
		db.transactionLog = nil
	}
}

type rebuildDatabaseResult struct {
	// replaced remains reachable until the caller has committed schema.json.
	// On any build or publication error these shards must stay on disk because
	// the last committed schema may still reference them.
	replaced []replacedShardGeneration
	errors   []string
	complete bool
}

type replacedShardGeneration struct {
	shard   *storageShard
	drained <-chan struct{}
}

func cleanupReplacedShardGenerations(replaced []replacedShardGeneration, context string) []string {
	cleanup := func() []string {
		var errors []string
		for _, generation := range replaced {
			<-generation.drained
			if err := func() (err error) {
				defer func() {
					err = recoverAsError("cleanup failed for " + context + " shard " + generation.shard.uuid.String())
				}()
				generation.shard.RemoveFromDisk()
				return nil
			}(); err != nil {
				errors = append(errors, err.Error())
			}
		}
		return errors
	}

	for _, generation := range replaced {
		select {
		case <-generation.drained:
		default:
			go func() {
				for _, err := range cleanup() {
					fmt.Println("error:", err)
				}
			}()
			return nil
		}
	}
	return cleanup()
}

type schemaWriteOptions interface {
	// This has the same atomic-generation contract as WriteSchema. durable also
	// requires data and namespace metadata to be stable before returning.
	WriteSchemaWithMode(schema []byte, durable bool)
}

type schemaSaveMode uint8

const (
	schemaSaveFsync schemaSaveMode = iota
	schemaSaveNoFsync
	schemaSaveBuffered
)

func schemaSaveModeForDurability(durable bool) schemaSaveMode {
	if durable {
		return schemaSaveFsync
	}
	return schemaSaveNoFsync
}

func normalizeTempLookupName(dbName string, name string) string {
	if !strings.HasPrefix(name, ".") {
		return name
	}
	replacer := strings.NewReplacer(
		`"`+dbName+`.`, `"`,
		`\"`+dbName+`.`, `\"`,
	)
	return replacer.Replace(name)
}

// Custom JSON to persist private tables field
func (d *database) MarshalJSON() ([]byte, error) {
	type persist struct {
		Name       string                                     `json:"name"`
		Tables     *NonLockingReadMap.ReadMap[string, *table] `json:"tables"`
		Procedures []*ProcedureDescription                    `json:"procedures,omitempty"`
	}
	definitions := make([]*ProcedureDescription, 0)
	if d.procedures != nil {
		for _, definition := range d.procedures.GetAll() {
			definitions = append(definitions, definition)
		}
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Name < definitions[j].Name })
	tables := d.tables
	if sharedClusterDatabase(d.Name) {
		tables = NonLockingReadMap.NewReadMap[string, *table]()
		for _, table := range d.tables.GetAll() {
			if !table.isEphemeralQueryTable() {
				tables.Set(table.Name, table)
			}
		}
	}
	return json.Marshal(persist{Name: d.Name, Tables: tables, Procedures: definitions})
}

func (d *database) UnmarshalJSON(data []byte) error {
	type persist struct {
		Name       string                                     `json:"name"`
		Tables     *NonLockingReadMap.ReadMap[string, *table] `json:"tables"`
		Procedures []*ProcedureDescription                    `json:"procedures,omitempty"`
	}
	var p persist
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	d.Name = p.Name
	d.tables = p.Tables
	d.procedures = NonLockingReadMap.NewReadMap[string, *ProcedureDescription]()
	for _, definition := range p.Procedures {
		if definition == nil || definition.Name == "" || definition.Language == "" || definition.Source == "" || d.procedures.Get(definition.Name) != nil {
			return fmt.Errorf("invalid stored procedure source definition")
		}
		d.procedures.Set(definition.Name, definition)
	}
	if d.tables == nil {
		d.tables = NonLockingReadMap.NewReadMap[string, *table]()
	}
	return nil
}

// Database namespace writes publish the folded secondary index with the
// existing exact catalog. Only DDL takes namespaceMu; all lookups are reads of
// immutable publications. Empty spellings denote an ambiguous folded name.
type databaseDirectory struct {
	*NonLockingReadMap.ReadMap[string, *database]
	namespaceMu sync.Mutex
	folded      atomic.Pointer[map[string]string]
}

func (d *databaseDirectory) Set(name string, db *database) *database {
	d.namespaceMu.Lock()
	defer d.namespaceMu.Unlock()
	previous := d.ReadMap.Set(name, db)
	d.publishNames()
	return previous
}

func (d *databaseDirectory) Remove(name string) *database {
	d.namespaceMu.Lock()
	defer d.namespaceMu.Unlock()
	previous := d.ReadMap.Remove(name)
	d.publishNames()
	return previous
}

func (d *databaseDirectory) publishNames() {
	names := make(map[string]string)
	for _, db := range d.GetAll() {
		key := foldIdentifier(db.Name)
		if _, exists := names[key]; exists {
			names[key] = ""
		} else {
			names[key] = db.Name
		}
	}
	d.folded.Store(&names)
}

var databases = &databaseDirectory{ReadMap: NonLockingReadMap.NewReadMap[string, *database]()}
var Basepath string = "data"

func (d *database) ComputeSize() uint {
	var sz uint = 16 * 8 // heuristic
	for _, t := range d.tables.GetAll() {
		sz += t.ComputeSize()
	}
	if d.procedures != nil {
		for _, definition := range d.procedures.GetAll() {
			sz += definition.ComputeSize()
		}
	}
	return sz
}

func recoverAsError(context string) (err error) {
	if r := recover(); r != nil {
		err = fmt.Errorf("%s: %v", context, r)
	}
	return err
}

func Rebuild(all bool, repartition bool) string {
	return rebuildDatabases(all, repartition, false)
}

// RebuildTable compacts only the table referenced by tbl. It uses the same
// publication, schema-save and deferred old-shard cleanup path as a global
// rebuild without touching unrelated tables or databases.
func RebuildTable(tbl *table, all bool, repartition bool) string {
	start := time.Now()
	if tbl == nil || tbl.schema == nil || tbl.schema.tables.Get(tbl.Name) != tbl {
		panic("cannot rebuild a stale table handle")
	}

	tbl.schema.persistenceLifecycle.RLock()
	defer tbl.schema.persistenceLifecycle.RUnlock()
	result := tbl.schema.rebuildWithLifecycle(all, repartition, true, tbl)
	if r := replication.Load(); r != nil && !r.master && validReplicaDatabase(tbl.schema.Name) {
		defer func() { go replicationCheckpoint(tbl.schema.Name, "") }()
	}
	errs := append([]string(nil), result.errors...)
	if len(errs) == 0 {
		if err := func() (err error) {
			defer func() { err = recoverAsError("save failed for database " + tbl.schema.Name) }()
			tbl.schema.save()
			return nil
		}(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) == 0 {
		// tbl.schema.save() committed all replacement UUIDs. Only this success
		// path may remove files belonging to the previous generation.
		errs = append(errs, cleanupReplacedShardGenerations(result.replaced, "table "+tbl.schema.Name+"."+tbl.Name)...)
	}

	duration := fmt.Sprint(time.Since(start))
	if len(errs) == 0 {
		return duration
	}
	return duration + " errors: " + strings.Join(errs, " | ")
}

func rebuildDatabases(all bool, repartition bool, includeEphemeral bool) string {
	start := time.Now()
	dbs := databases.GetAll()
	var errs []string
	for _, db := range dbs {
		failures, _ := rebuildDatabaseAndCompact(db, all, repartition, includeEphemeral)
		errs = append(errs, failures...)
	}
	duration := fmt.Sprint(time.Since(start))
	if len(errs) == 0 {
		return duration
	}
	return duration + " errors: " + strings.Join(errs, " | ")
}

// complete distinguishes a full checkpoint from successful best-effort
// maintenance which skipped a busy or cold table. Replication needs the former.
func rebuildDatabaseAndCompact(db *database, all bool, repartition bool, includeEphemeral bool) ([]string, bool) {
	db.persistenceLifecycle.RLock()
	defer db.persistenceLifecycle.RUnlock()
	if replication.Load() != nil && GetDatabase(db.Name) != db {
		return []string{"database generation was replaced"}, false
	}
	var transactionCutoff uint64
	var hasTransactions bool
	if err := func() (err error) {
		defer func() { err = recoverAsError("transaction log snapshot failed for database " + db.Name) }()
		transactionCutoff, hasTransactions = db.transactionCompactionSnapshot()
		return nil
	}(); err != nil {
		return []string{err.Error()}, false
	}
	result := db.rebuildWithLifecycle(all, repartition, includeEphemeral)
	if r := replication.Load(); r != nil && !r.master && validReplicaDatabase(db.Name) {
		defer func() { go replicationCheckpoint(db.Name, "") }()
	}
	if len(result.errors) > 0 {
		return result.errors, false
	}
	if err := func() (err error) {
		defer func() { err = recoverAsError("save failed for database " + db.Name) }()
		db.save()
		return nil
	}(); err != nil {
		return []string{err.Error()}, false
	}
	// db.save() committed all replacement UUIDs. A save failure returns above
	// and deliberately retains every old generation.
	errs := cleanupReplacedShardGenerations(result.replaced, "database "+db.Name)
	if len(errs) != 0 || !result.complete || !hasTransactions {
		return errs, result.complete && len(errs) == 0
	}
	if err := func() (err error) {
		defer func() { err = recoverAsError("transaction log compaction failed for database " + db.Name) }()
		db.compactTransactionLog(transactionCutoff)
		return nil
	}(); err != nil {
		errs = append(errs, err.Error())
	}
	return errs, result.complete && len(errs) == 0
}

func UnloadDatabases() {
	fmt.Println("table compression done in ", rebuildDatabases(false, false, true))
	for _, db := range databases.GetAll() {
		db.closeTransactionLog()
	}
	if err := saveSettings(); err != nil {
		fmt.Println("saving local settings:", err)
	}
	GlobalCache.Stop()
}

// createPersistenceFromConfig creates a PersistenceEngine by looking up the
// "backend" field in the JSON config and dispatching to the registered factory.
func createPersistenceFromConfig(dbName string, raw json.RawMessage) PersistenceEngine {
	var header struct {
		Backend string `json:"backend"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return nil
	}
	factory, ok := BackendRegistry[header.Backend]
	if !ok {
		return nil
	}
	return instrumentPersistence(dbName, factory(dbName, raw))
}

func LoadDatabases() {
	// this happens before any init, so no read/write action is performed on any data yet
	// read settings file
	if settings, err := os.Open(Basepath + "/settings.json"); err == nil {
		defer settings.Close()
		stat, _ := settings.Stat()
		data := make([]byte, stat.Size())
		if _, err := settings.Read(data); err == nil {
			json.Unmarshal(data, &Settings)
		}
	}
	InitSettings()
	// enumerate dbs; do not load schemas/shards yet (lazy-load on demand)
	entries, _ := os.ReadDir(Basepath)
	configured := make(map[string]bool)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") && entry.Name() != "settings.json" {
			configured[strings.TrimSuffix(entry.Name(), ".json")] = true
		}
	}
	for _, entry := range entries {
		if entry.IsDir() && !configured[entry.Name()] {
			db := newDatabase()
			db.Name = entry.Name()
			db.persistence = instrumentPersistence(entry.Name(), &FileStorage{path: Basepath + "/" + entry.Name() + "/"})
			db.srState = COLD
			if !sharedClusterDatabase(db.Name) {
				db.recoverAbandonedBlobWrites()
			}
			databases.Set(db.Name, db)
		} else if strings.HasSuffix(entry.Name(), ".json") && entry.Name() != "settings.json" {
			// Backend configuration file (e.g., Ceph, S3)
			dbName := strings.TrimSuffix(entry.Name(), ".json")
			configPath := Basepath + "/" + entry.Name()
			configData, err := os.ReadFile(configPath)
			if err != nil {
				fmt.Println("error: failed to read backend config", configPath, ":", err)
				continue
			}
			persistence := createPersistenceFromConfig(dbName, json.RawMessage(configData))
			if persistence == nil {
				fmt.Println("error: unknown or invalid backend in", configPath)
				continue
			}
			fmt.Println("loading database", dbName, "from backend config", configPath)
			db := newDatabase()
			db.Name = dbName
			db.persistence = persistence
			db.srState = COLD
			if !sharedClusterDatabase(db.Name) {
				db.recoverAbandonedBlobWrites()
			}
			databases.Set(db.Name, db)
		}
	}

	// Ensure system_statistic.scans exists for scan logging
	ensureSystemStatistic()
}

func (db *database) writeSchema(jsonbytes []byte, durable bool) {
	if writer, ok := db.persistence.(schemaWriteOptions); ok {
		writer.WriteSchemaWithMode(jsonbytes, durable)
		return
	}
	db.persistence.WriteSchema(jsonbytes)
}

func (db *database) getSaveCond() *sync.Cond {
	db.saveCondOnce.Do(func() {
		db.saveCond = sync.NewCond(&db.saveMu)
	})
	return db.saveCond
}

func (db *database) commitSchemaSnapshot(jsonbytes []byte, durable bool, releaseSchema func()) {
	// The caller still owns schemalock from snapshot capture. Register its
	// queue position before releasing that ownership: a later DDL snapshot
	// must never be followed on disk by an older rebuild snapshot. Waiting
	// and backend I/O continue without schemalock, as before.
	db.saveMu.Lock()
	cond := db.getSaveCond()
	db.saveRequested++
	seq := db.saveRequested
	db.savePending = jsonbytes
	db.savePendingSync = db.savePendingSync || durable
	releaseSchema()
	if db.savePanic != nil && !db.saveInFlight {
		db.savePanic = nil
	}
	if db.saveInFlight {
		for db.saveCompleted < seq && db.savePanic == nil {
			cond.Wait()
		}
		panicVal := db.savePanic
		db.saveMu.Unlock()
		if panicVal != nil {
			panic(panicVal)
		}
		return
	}
	db.saveInFlight = true
	db.savePanic = nil
	for {
		snapshot := db.savePending
		writeSync := db.savePendingSync
		targetSeq := db.saveRequested
		db.savePending = nil
		db.savePendingSync = false
		db.saveMu.Unlock()

		var panicVal any
		func() {
			defer func() {
				panicVal = recover()
			}()
			db.writeSchema(snapshot, writeSync)
		}()

		db.saveMu.Lock()
		if panicVal != nil {
			db.savePanic = panicVal
			db.saveInFlight = false
			cond.Broadcast()
			db.saveMu.Unlock()
			panic(panicVal)
		}
		db.saveCompleted = targetSeq
		cond.Broadcast()
		if db.savePending == nil {
			db.saveInFlight = false
			db.saveMu.Unlock()
			return
		}
	}
}

func (db *database) save() {
	if sharedClusterDatabase(db.Name) {
		cluster := activeNodeCluster()
		if cluster == nil || cluster.manager.resourceCell(mosiResource{Database: db.Name}).ownstate.Phase() != mosi.M {
			return
		}
	}
	if db.srState == COLD {
		// Do not serialize a cold database; keep existing schema.json intact
		return
	}
	db.schemalock.RLock()
	jsonbytes, _ := json.MarshalIndent(db, "", "  ")
	// Buffered mutations after this snapshot must leave the catalog dirty.
	// They need the exclusive schema lock and therefore cannot race this store.
	db.schemaDirty.Store(false)
	func() {
		defer func() {
			if r := recover(); r != nil {
				db.schemaDirty.Store(true)
				panic(r)
			}
		}()
		db.commitSchemaSnapshot(jsonbytes, true, db.schemalock.RUnlock)
	}()
	// shards are written while rebuild
}

// saveLockedAndUnlock publishes according to mode and always releases
// schemalock. Buffered temp metadata is immediately visible in memory; a later
// synchronous schema save or rebuild includes it in its complete snapshot.
func (db *database) saveLockedAndUnlock(mode schemaSaveMode) {
	if sharedClusterDatabase(db.Name) {
		cluster := activeNodeCluster()
		if cluster == nil || cluster.manager.resourceCell(mosiResource{Database: db.Name}).ownstate.Phase() != mosi.M {
			db.schemalock.Unlock()
			return
		}
	}
	if db.srState == COLD {
		db.schemalock.Unlock()
		return
	}
	if mode == schemaSaveBuffered {
		db.schemaDirty.Store(true)
		db.schemalock.Unlock()
		return
	}
	jsonbytes, _ := json.MarshalIndent(db, "", "  ")
	// Clear while the snapshot is protected. A later buffered mutation takes
	// schemalock exclusively and sets dirty again after this generation.
	db.schemaDirty.Store(false)
	defer func() {
		if r := recover(); r != nil {
			db.schemaDirty.Store(true)
			panic(r)
		}
	}()
	db.commitSchemaSnapshot(jsonbytes, mode == schemaSaveFsync, db.schemalock.Unlock)
}

// ensureLoaded loads schema.json into the database struct exactly once.
func (db *database) ensureLoaded() {
	db.loadOnce.Do(func() {
		defer db.publishTableNamesLocked()
		db.initializeTransactionLog()
		if db.srState != COLD {
			return
		}
		jsonbytes := db.persistence.ReadSchema()
		if len(jsonbytes) == 0 {
			// fresh/empty database
			db.tables = NonLockingReadMap.NewReadMap[string, *table]()
			db.srState = SHARED
			return
		}
		tmp := new(database)
		if err := json.Unmarshal(jsonbytes, tmp); err != nil {
			panic(err)
		}
		db.tables = tmp.tables
		db.procedures = tmp.procedures
		// restore back-references; do not touch on-disk columns yet
		for _, t := range db.tables.GetAll() {
			t.schema = db
			invalidatePersistedPlannerCodeAfterLoad(t)
			if t.Name == ".blobs" {
				db.blobRefState().table.Store(t)
			}
			for _, col := range t.Columns {
				col.UpdateSanitizer()
			}
			// attach table pointer to existing shard stubs without loading them
			if t.Shards != nil {
				for _, s := range t.Shards {
					if s != nil {
						s.t = t
					}
				}
			}
			if t.PShards != nil {
				for _, s := range t.PShards {
					if s != nil {
						s.t = t
					}
				}
			}
			// Derive ShardMode from shard presence for backward compatibility
			// with schemas that don't yet have ShardMode persisted.
			if t.PShards != nil && t.Shards == nil {
				t.ShardMode = ShardModePartition
			} else {
				t.ShardMode = ShardModeFree
			}
			t.publishTopologyLocked()
			t.initializeLegacyPlannerRowEstimate()
			t.publishShowColumnsSnapshot(true)
			t.restoreKeyFrequencies()
			t.restoreFilterFeedback()
		}
		// FK declarations are authoritative, while their system triggers are
		// generated code. Rebuild that code at the persistence boundary so schema
		// files written by older binaries cannot retain an obsolete builtin ABI.
		rebuildFKTriggersAfterLoad(db)
		db.srState = SHARED
		// Dot-prefixed cache tables are planner-owned and persisted only so a
		// warm query cache can survive restart. Re-register their table-level
		// owner after loading; durable internal tables such as .blobs remain
		// ordinary shard-owned storage.
		for _, table := range db.tables.GetAll() {
			if table.isEphemeralQueryTable() {
				registerCreatedTable(table)
			}
		}
	})
}

// publishTableNamesLocked owns schema membership (or an unpublished database).
// Names are immutable scalar copies, not pointers to mutable table descriptors.
func (db *database) publishTableNamesLocked() {
	names := make(map[string]string)
	for _, t := range db.tables.GetAll() {
		if t.isHiddenFromShowTables() {
			continue
		}
		folded := foldIdentifier(t.Name)
		if _, exists := names[folded]; exists {
			names[folded] = ""
		} else {
			names[folded] = t.Name
		}
	}
	db.foldedTableNames.Store(&names)
}

// invalidatePersistedPlannerCodeAfterLoad drops executable code derived from a
// physical query plan. Planner-owned CACHE tables contain no durable rows, and
// the current createtable guard supplies their authoritative oninit callback on
// first use. Retaining an older callback across a physical operator ABI change
// would execute stale generated code before that guard can refresh it.
// upgradePersistedTableCalls runs only while loading unpublished executable
// definitions. It changes the old native table-call ABI, never SQL syntax or
// row values. Quoted data stays opaque; the live builtin accepts only the new
// interface. New Proc identities discard any obsolete compiled/optimizer state.
func upgradePersistedTableCalls(value scm.Scmer) scm.Scmer {
	bindsTable := func(params scm.Scmer) bool {
		params = params.WithoutSourceInfo()
		if callHeadIs(params, "table") {
			return true
		}
		if params.IsSlice() {
			for _, param := range params.Slice() {
				if callHeadIs(param, "table") {
					return true
				}
			}
		}
		return false
	}
	var patternBindsTable func(scm.Scmer) bool
	patternBindsTable = func(pattern scm.Scmer) bool {
		pattern = pattern.WithoutSourceInfo()
		if callHeadIs(pattern, "table") {
			return true
		}
		if pattern.IsSlice() {
			parts := pattern.Slice()
			if len(parts) > 0 && (callHeadIs(parts[0], "quote") || callHeadIs(parts[0], "symbol") || callHeadIs(parts[0], "ignorecase") || callHeadIs(parts[0], "eval")) {
				return false // literal/reference patterns do not introduce a binding
			}
			for _, part := range parts {
				if patternBindsTable(part) {
					return true
				}
			}
		}
		return false
	}
	// Scope state belongs only to this cold definition transformation.
	// BEGIN owns a scope; !BEGIN and argument evaluation share their caller's.
	// DEFINE and SET bind only after their initializer, while a lambda initializer
	// can recursively refer to the binding established before its invocation.
	const (
		native = iota
		local
		conditional
	)
	join := func(a, b int) int {
		if a == b {
			return a
		}
		return conditional
	}
	var restore func(scm.Scmer, int) (scm.Scmer, int)
	restore = func(value scm.Scmer, scope int) (scm.Scmer, int) {
		if scope == local {
			return value, local
		}
		if value.IsSourceInfo() {
			inner := value.WithoutSourceInfo()
			if rewritten, bound := restore(inner, scope); rewritten != inner {
				return rewritten, bound
			} else {
				return value, bound
			}
		}
		if value.IsProc() {
			proc := value.Proc()
			if bindsTable(proc.Params) {
				return value, scope
			}
			if proc.En != nil {
				if owner := proc.En.FindRead(scm.Symbol("table")); owner != nil && owner.Vars[scm.Symbol("table")] != scm.Globalenv.Vars[scm.Symbol("table")] {
					return value, scope // retained lexical capture, not the builtin
				}
			}
			body, _ := restore(proc.Body, scope)
			if body == proc.Body {
				return value, scope
			}
			return scm.NewProcStruct(scm.Proc{Params: proc.Params, Body: body, En: proc.En,
				NumVars: proc.NumVars, NumberedOnly: proc.NumberedOnly}), scope
		}
		if !value.IsSlice() || len(value.Slice()) == 0 {
			return value, scope
		}
		items := value.Slice()
		if callHeadIs(items[0], "quote") {
			return value, scope
		}
		var replacement []scm.Scmer
		replace := func(i int, rewritten scm.Scmer) {
			if rewritten != items[i] {
				if replacement == nil {
					replacement = append([]scm.Scmer(nil), items...)
				}
				replacement[i] = rewritten
			}
		}
		bound := scope
		switch {
		case len(items) == 3 && callHeadIs(items[0], "lambda"):
			if !bindsTable(items[1]) {
				body, _ := restore(items[2], scope)
				replace(2, body)
			}
		case len(items) == 3 && (callHeadIs(items[0], "define") || callHeadIs(items[0], "set")):
			declaresTable := callHeadIs(items[1], "table")
			initializer := items[2].WithoutSourceInfo()
			recursive := initializer.IsProc() || (initializer.IsSlice() && len(initializer.Slice()) > 0 && callHeadIs(initializer.Slice()[0], "lambda"))
			initialScope := scope
			if declaresTable && recursive {
				initialScope = local
			}
			rewritten, next := restore(items[2], initialScope)
			replace(2, rewritten)
			bound = next
			if declaresTable {
				bound = local
			}
		case callHeadIs(items[0], "if"):
			// Alternatives do not execute in sequence. Retain their possible
			// outgoing bindings, and never guess across a conditional rebinding.
			branchScopes := make([]int, 0, len(items)/2+1)
			for i := 1; i < len(items); i += 2 {
				rewritten, next := restore(items[i], bound)
				replace(i, rewritten)
				bound = next
				if i+1 < len(items) {
					rewritten, next = restore(items[i+1], bound)
					replace(i+1, rewritten)
					branchScopes = append(branchScopes, next)
				}
			}
			for _, branch := range branchScopes {
				bound = join(bound, branch)
			}
		case callHeadIs(items[0], "match") || callHeadIs(items[0], "match_mut"):
			if len(items) > 1 {
				rewritten, next := restore(items[1], bound)
				replace(1, rewritten)
				bound = next
			}
			var restorePattern func(scm.Scmer) scm.Scmer
			restorePattern = func(value scm.Scmer) scm.Scmer {
				pattern := value.WithoutSourceInfo()
				if !pattern.IsSlice() || len(pattern.Slice()) == 0 {
					return value
				}
				parts := pattern.Slice()
				if callHeadIs(parts[0], "eval") {
					rewritten, _ := restore(value, bound)
					return rewritten
				}
				if callHeadIs(parts[0], "quote") || callHeadIs(parts[0], "symbol") || callHeadIs(parts[0], "ignorecase") {
					return value
				}
				var rewritten []scm.Scmer
				for i, part := range parts {
					if next := restorePattern(part); next != part {
						if rewritten == nil {
							rewritten = append([]scm.Scmer(nil), parts...)
						}
						rewritten[i] = next
					}
				}
				if rewritten != nil {
					return scm.NewSlice(rewritten)
				}
				return value
			}
			for i := 2; i < len(items); i += 2 {
				body, branchScope := i, bound
				if i+1 < len(items) {
					body = i + 1
					replace(i, restorePattern(items[i]))
					if patternBindsTable(items[i]) {
						branchScope = local
					}
				}
				rewritten, _ := restore(items[body], branchScope)
				replace(body, rewritten)
			}
		default:
			for i, item := range items {
				rewritten, next := restore(item, bound)
				replace(i, rewritten)
				bound = next
			}
			if callHeadIs(items[0], "begin") || callHeadIs(items[0], "begin_mut") {
				bound = scope // declarations in this nested scope do not escape
			}
		}
		if len(items) == 3 && callHeadIs(items[0], "table") {
			if scope == conditional {
				panic("cannot upgrade a table call after conditional rebinding; update the stored definition explicitly")
			}
			if replacement != nil {
				items = replacement
			}
			return scm.NewSlice([]scm.Scmer{items[0], items[1], scm.NewBool(false), items[2], scm.NewBool(false)}), bound
		}
		if replacement != nil {
			return scm.NewSlice(replacement), bound
		}
		return value, bound
	}
	restored, _ := restore(value, native)
	return restored
}

func invalidatePersistedPlannerCodeAfterLoad(t *table) {
	if initializer := t.OnInit.Load(); initializer != nil {
		restored := upgradePersistedTableCalls(*initializer)
		t.OnInit.Store(&restored)
	}
	for _, col := range t.Columns {
		col.OnUpdate = upgradePersistedTableCalls(col.OnUpdate)
		col.OrcFilterFn = upgradePersistedTableCalls(col.OrcFilterFn)
		col.OrcMapReduceFn = upgradePersistedTableCalls(col.OrcMapReduceFn)
	}
	for i := range t.Triggers {
		trigger := &t.Triggers[i]
		trigger.Func = upgradePersistedTableCalls(trigger.Func)
		if trigger.Language == "scheme" && trigger.Source != "" {
			definition := scm.Read("stored trigger", trigger.Source)
			if restored := upgradePersistedTableCalls(definition); restored != definition {
				trigger.Source = scm.SerializeToString(restored, &scm.Globalenv)
			}
		}
	}
	if t.isEphemeralQueryTable() {
		t.OnInit.Store(nil)
	}
}

// SharedResource impl for database
func (db *database) GetState() SharedState { return db.srState }
func (db *database) GetRead(tx *TxContext) ReadAccess {
	if r := activeReplication(db); r != nil && validReplicaDatabase(db.Name) {
		resource := mosiResource{Database: db.Name}
		// An authorized schema read only samples its local permission. User
		// proofs and SessionState context lookups belong to cold acquisition;
		// evaluating them here would synchronize even the granted read path.
		if !r.resourceCell(resource).ownstate.Phase().Readable() {
			r.obtain(resource, mosi.S, replicationUser(tx), tx)
		}
	}
	db.ensureLoaded()
	return ReadAccess{}
}
func (db *database) GetWrite(tx *TxContext) WriteAccess {
	var release func()
	ready := false
	defer func() {
		if !ready && release != nil {
			release()
		}
	}()
	if r := activeReplication(db); r != nil && validReplicaDatabase(db.Name) {
		_, release = r.writeResource(mosiResource{Database: db.Name}, tx)
	}
	db.ensureLoaded()
	ready = true
	return WriteAccess{release: release}
}

// helper to fetch a table with lazy db load
func (db *database) GetTable(name string, ignoreCase ...bool) *table {
	db.ensureLoaded()
	if len(ignoreCase) != 0 && ignoreCase[0] {
		names := db.foldedTableNames.Load()
		if names == nil {
			return nil
		}
		canonical, exists := (*names)[foldIdentifier(name)]
		if !exists {
			return nil
		}
		if canonical == "" {
			panic("ambiguous table name: " + name)
		}
		name = canonical
	}
	if t := db.tables.Get(name); t != nil {
		return t
	}
	/* Query-temp/keytable names historically used unqualified source aliases in
	the embedded get_column serialization. Newer planner/debug paths may look
	them up with schema-qualified aliases; accept both spellings so planner IR
	can evolve without breaking the physical temp-table namespace. */
	if strings.HasPrefix(name, ".") {
		normalizedName := normalizeTempLookupName(db.Name, name)
		if normalizedName != name {
			if t := db.tables.Get(normalizedName); t != nil {
				return t
			}
		}
		for _, t := range db.tables.GetAll() {
			if normalizeTempLookupName(db.Name, t.Name) == normalizedName {
				return t
			}
		}
	}
	return nil
}

func (db *database) ShowTables() scm.Scmer {
	db.ensureLoaded()
	tables := db.tables.GetAll()
	result := make([]scm.Scmer, 0, len(tables))
	for _, t := range tables {
		if t.isHiddenFromShowTables() {
			continue
		}
		result = append(result, scm.NewString(t.Name))
	}
	return scm.NewSlice(result)
}

func (db *database) rebuild(all bool, repartition bool, includeEphemeral bool, only ...*table) rebuildDatabaseResult {
	db.persistenceLifecycle.RLock()
	defer db.persistenceLifecycle.RUnlock()
	return db.rebuildWithLifecycle(all, repartition, includeEphemeral, only...)
}

// rebuildWithLifecycle requires a persistenceLifecycle read capability. The
// public rebuild paths retain it through schema publication, so cleanup cannot
// observe generation-private files in the build→publish interval.
func (db *database) rebuildWithLifecycle(all bool, repartition bool, includeEphemeral bool, only ...*table) rebuildDatabaseResult {
	if sharedClusterDatabase(db.Name) {
		r := activeReplication(db)
		if r.resourceCell(mosiResource{Database: db.Name}).ownstate.Phase() != mosi.M {
			return rebuildDatabaseResult{}
		}
	}

	if replication.Load() != nil && GetDatabase(db.Name) != db {
		return rebuildDatabaseResult{}
	}
	if db.srState == COLD {
		// do nothing for cold databases; avoid loading during rebuild
		return rebuildDatabaseResult{}
	}
	// Blob-producing table rebuilds run in parallel. Publish their shared
	// refcount table before workers start so no worker mutates the catalog.
	db.ensureBlobTable()
	var done sync.WaitGroup
	// Collect pre-rebuild shards that were superseded. Their cleanup
	// (RemoveFromDisk → ReleaseBlobs → DecrBlobRefcount) must run after
	// ALL table rebuilds finish to avoid deadlocks with concurrent .blobs
	// repartition holding shard read-locks.
	var replacedMu sync.Mutex
	var allReplaced []replacedShardGeneration
	var errMu sync.Mutex
	var rebuildErrors []string
	var incomplete atomic.Bool
	dbs := db.tables.GetAll()
	if len(only) > 0 {
		dbs = []*table{only[0]}
	}
	if !includeEphemeral {
		onlineTables := make([]*table, 0, len(dbs))
		for _, t := range dbs {
			if t.Name != ".blobs" {
				onlineTables = append(onlineTables, t)
			}
		}
		dbs = onlineTables
	}
	// Keep the shared blob catalog last. It waits for every blob-producing table;
	// putting it before an admission-blocked producer could otherwise reserve the
	// maintenance budget while waiting for work that has not been started yet.
	orderedTables := make([]*table, 0, len(dbs))
	var blobTable *table
	for _, t := range dbs {
		if t.Name == ".blobs" {
			blobTable = t
		} else {
			orderedTables = append(orderedTables, t)
		}
	}
	if blobTable != nil {
		orderedTables = append(orderedTables, blobTable)
	}
	dbs = orderedTables
	var blobWriters sync.WaitGroup
	for _, t := range dbs {
		if t.Name != ".blobs" {
			blobWriters.Add(1)
		}
	}
	done.Add(len(dbs))
	for _, t := range dbs {
		tableBytes := estimateTableMaintenanceBytes(t)
		tableLease := GlobalMaintenanceRAMBudget.Acquire(tableBytes)
		go func(t *table, tableBytes int64, tableLease *RAMLease) {
			defer tableLease.Release()
			if t.Name == ".blobs" {
				// Rebuilding other tables may update blob refcounts. Rebuild the
				// shared catalog only after those writes have completed.
				blobWriters.Wait()
			} else {
				defer blobWriters.Done()
			}
			tableLocked := false
			rebuildClaimed := false
			t.ddlMu.RLock()
			defer func() {
				t.ddlMu.RUnlock()
				if r := recover(); r != nil {
					errmsg := fmt.Sprintf("rebuild failed for table %s.%s: %v", db.Name, t.Name, r)
					scm.PrintError(errmsg)
					errMu.Lock()
					rebuildErrors = append(rebuildErrors, errmsg)
					errMu.Unlock()
					// best-effort unlock if still locked
					if tableLocked {
						func() { defer func() { _ = recover() }(); t.mu.Unlock() }()
					}
				}
				if rebuildClaimed {
					func() {
						defer func() { _ = recover() }()
						t.mu.Lock()
						t.maintenanceKind = 0
						t.mu.Unlock()
						t.maintenanceMu.Unlock()
					}()
				}
				done.Done()
			}()
			if t.isEphemeralQueryTable() && !includeEphemeral {
				return
			}
			if sharedClusterSource(t) {
				// Background maintenance must not collect distributed data or
				// rewrite a table only partly held here. Explicit MOSI handover
				// still rebuilds its individual target through prepareResource.
				r := activeReplication(db)
				for _, shard := range t.activeTopology().shards {
					if shard != nil && r.shardCell(shard).ownstate.Phase() != mosi.M {
						incomplete.Store(true)
						return
					}
				}
			}
			if !t.maintenanceMu.TryLock() {
				incomplete.Store(true)
				return // another rebuild/repartition is in progress
			}
			t.mu.Lock() // table lock
			tableLocked = true
			t.maintenanceKind = 1
			rebuildClaimed = true
			// TODO: check LRU statistics and remove unused computed columns

			// Snapshot the active shard list, then release t.mu while the
			// expensive shard rebuild runs. Writers must not be blocked on
			// the whole sdone.Wait() phase.
			targetIsP := t.ShardMode == ShardModePartition
			origTopology := t.activeTopology()
			origShardList := append([]*storageShard(nil), origTopology.shards...)
			t.mu.Unlock()
			tableLocked = false
			newShardList := make([]*storageShard, len(origShardList))
			// These facts are derived from the completed shard generations below.
			hasColdShard := false
			maincount := uint(0)
			var sdone sync.WaitGroup
			var shardErrMu sync.Mutex
			var shardErrors []string
			sdone.Add(len(origShardList))
			// Acquire both CPU and RAM before creating the goroutine. This avoids a
			// pool of blocked worker goroutines when memory is the tighter limit.
			workers := runtime.NumCPU()
			if workers < 1 {
				workers = 1
			}
			workerRights := make(chan struct{}, workers)
			tableRows := int64(t.CountEstimate())
			for i, s := range origShardList {
				workerRights <- struct{}{}
				shardLease := tableLease.Acquire(estimateShardMaintenanceBytes(tableBytes, estimatedShardRows(s), tableRows))
				go func(i int, s *storageShard, shardLease *RAMLease) {
					defer func() { <-workerRights }()
					defer shardLease.Release()
					defer func() {
						if r := recover(); r != nil {
							errmsg := fmt.Sprintf("shard rebuild failed for %s.%s shard %d: %v", db.Name, t.Name, i, r)
							fmt.Println("error:", errmsg)
							shardErrMu.Lock()
							shardErrors = append(shardErrors, errmsg)
							shardErrMu.Unlock()
						}
						sdone.Done()
					}()
					if s != nil {
						if r := activeReplication(db); r != nil && validReplicaDatabase(db.Name) && !t.isEphemeralQueryTable() {
							cell := r.shardCell(s)
							handover := cell.ownstate.TryHandover(cell.writesDrained)
							if handover == nil {
								newShardList[i] = s
								incomplete.Store(true)
								return
							}
							defer handover.Release()
							phase := cell.ownstate.Phase()
							if phase != mosi.M && phase != mosi.O {
								newShardList[i] = s
								incomplete.Store(true)
								return
							}
						}
						newShardList[i] = s.rebuild(all)
					}
				}(i, s, shardLease)
			}
			sdone.Wait()
			if len(shardErrors) > 0 {
				errMu.Lock()
				rebuildErrors = append(rebuildErrors, shardErrors...)
				errMu.Unlock()
				t.mu.Lock()
				t.maintenanceKind = 0
				t.mu.Unlock()
				t.maintenanceMu.Unlock()
				rebuildClaimed = false
				return
			}
			// Recompute both facts after the workers finish. A forced rebuild can
			// turn an initially cold shard into an authoritative warm generation;
			// conversely, an unchanged shard can become cold after the snapshot.
			// Repartitioning must use only the completed generations, never the
			// pre-rebuild zero counters of cold shards.
			hasColdShard = false
			maincount = 0
			for _, shard := range newShardList {
				if shard == nil {
					continue
				}
				shard.mu.RLock()
				cold := shard.state() == COLD
				count := uint(shard.main_count) + uint(len(shard.inserts)) - uint(shard.deletions.Count())
				shard.mu.RUnlock()
				maincount += count
				if cold {
					hasColdShard = true
				}
			}
			// A non-forced rebuild deliberately retains cold generations without
			// replaying their WAL. Their coordinator references are therefore
			// unknown and the database transaction log must remain intact.
			if hasColdShard {
				incomplete.Store(true)
			}

			// Collect pre-rebuild shards that were replaced so we can clean them up.
			var replaced []*storageShard
			if targetIsP {
				for i, oldShard := range origShardList {
					if oldShard != nil && oldShard != newShardList[i] && oldShard.uuid != newShardList[i].uuid {
						replaced = append(replaced, oldShard)
					}
				}
			} else {
				for i, oldShard := range origShardList {
					if oldShard != nil && oldShard != newShardList[i] && oldShard.uuid != newShardList[i].uuid {
						replaced = append(replaced, oldShard)
					}
				}
			}

			// A persistent generation is named by schema.json. Keep the old
			// topology authoritative and briefly drain its writers while the new
			// UUID list is committed. Ephemeral tables have no schema commit
			// boundary; waiting for their transaction here can self-deadlock cache
			// initialization, which invokes RebuildTable inside that transaction.
			durablePublication := len(replaced) > 0 && t.PersistencyMode != Memory && t.PersistencyMode != Cache
			if durablePublication {
				for _, shard := range origShardList {
					if shard != nil {
						shard.mu.Lock()
					}
				}
			}

			t.mu.Lock()
			tableLocked = true
			if targetIsP {
				t.PShards = newShardList
			} else {
				t.Shards = newShardList
			}
			previousSchemaTopology := t.publishSchemaTopology(origTopology.mode, newShardList, origTopology.dimensions)
			t.mu.Unlock()
			tableLocked = false

			var publicationPanic any
			if durablePublication {
				func() {
					defer func() { publicationPanic = recover() }()
					db.save()
				}()
			}
			if publicationPanic != nil {
				t.mu.Lock()
				if targetIsP {
					t.PShards = origShardList
				} else {
					t.Shards = origShardList
				}
				t.schemaTopology.Store(previousSchemaTopology)
				t.maintenanceKind = 0
				t.mu.Unlock()
				if durablePublication {
					for i := len(origShardList) - 1; i >= 0; i-- {
						if origShardList[i] != nil {
							origShardList[i].mu.Unlock()
						}
					}
				}
				errMu.Lock()
				rebuildErrors = append(rebuildErrors, fmt.Sprintf("schema publication failed for %s.%s: %v", db.Name, t.Name, publicationPanic))
				errMu.Unlock()
				t.maintenanceMu.Unlock()
				rebuildClaimed = false
				return
			}

			t.mu.Lock()
			tableLocked = true
			t.publishTopologyLocked()
			if durablePublication {
				for i := len(origShardList) - 1; i >= 0; i-- {
					if origShardList[i] != nil {
						origShardList[i].mu.Unlock()
					}
				}
			}
			// The published generations can now receive ordinary mutations. Release
			// table.mu before taking their shard locks for statistics: mutation
			// callbacks may already own a shard lock and need table metadata.
			t.mu.Unlock()
			tableLocked = false

			if !hasColdShard {
				// Update per-column statistics only when every rebuilt shard has
				// authoritative in-memory counters and column statistics.
				t.collectStatisticsFromShards(newShardList)
				rowEst := uint64(t.CountEstimate())
				for ci := range t.Columns {
					var distinctSum uint64
					colName := t.Columns[ci].Name
					for _, shard := range newShardList {
						if shard == nil {
							continue
						}
						// columns are populated during rebuild - access directly
						if cs, ok := shard.columns[colName]; ok && cs != nil {
							distinctSum += uint64(cs.DistinctCount())
						}
					}
					atomic.StoreUint64(&t.Columns[ci].DistinctEstimate, distinctSum)
					atomic.StoreUint64(&t.Columns[ci].RowEstimate, rowEst)
					// Planner statistics are properties of persisted base columns. Reading
					// hidden/cache or computed storages can initialize maintenance state and
					// must never be part of a statistics-only rebuild pass.
					if !strings.HasPrefix(t.Name, ".") && !t.Columns[ci].IsTemp && t.Columns[ci].Computor.IsNil() && len(t.Columns[ci].OrcSortCols) == 0 {
						t.Columns[ci].PlannerStats.Store(
							collectRebuiltColumnPlannerStatistics(newShardList, colName))
					} else {
						t.Columns[ci].PlannerStats.Store(nil)
					}
				}
			}
			t.publishShowColumnsSnapshot(false)
			t.mu.Lock()
			tableLocked = true

			// Collect replaced shards for deferred cleanup (see comment above).
			if len(replaced) > 0 {
				replacedMu.Lock()
				for _, shard := range replaced {
					allReplaced = append(allReplaced, replacedShardGeneration{
						shard:   shard,
						drained: origTopology.drained,
					})
				}
				replacedMu.Unlock()
			}

			// Decide on repartition while holding t.mu, but execute it
			// OUTSIDE the table lock so concurrent inserts can proceed
			// and the dual-write mechanism works correctly.
			//
			// Contract:
			//   - manual/queryplan partitiontable drives explicit repartitioning
			//   - db.rebuild() only repartitions when proposerepartition says the
			//     physical layout should actually change
			//   - a plain rebuild must not silently convert every free table into
			//     "partitioned with one shard", because that touches unrelated
			//     tables during shared rebuild tests without any layout benefit
			//
			// Since we already hold maintenanceMu from the rebuild, just
			// transition to maintenanceKind=2 for repartition (no re-lock needed).
			var shardCandidates []shardDimension
			doRepart := false
			if repartition && !hasColdShard && replication.Load() == nil {
				var shouldChange bool
				shardCandidates, shouldChange = t.proposerepartition(maincount)
				if shouldChange {
					if len(shardCandidates) > 0 {
						doRepart = true
					} else {
						// No explicit partition dimensions means repartition() would
						// fall back to generic parallel sharding. Only do that when
						// it would actually create more than one shard; otherwise a
						// small free table would be pointlessly converted into
						// "partitioned with one shard" during a global rebuild.
						desiredShards := repartitionShardTarget(maincount, true)
						doRepart = desiredShards > 1
					}
				}
			}
			if doRepart {
				t.maintenanceKind = 2 // transition rebuild→repartition, still holding maintenanceMu
			} else {
				t.maintenanceKind = 0
			}

			t.mu.Unlock()
			tableLocked = false
			// Publication retired origTopology. Wait without polling until every
			// scan/write task that selected it has left; the authoritative topology
			// and its writers are already live while this cleanup barrier drains.
			select {
			case <-origTopology.operationsDrained:
			case <-time.After(repartitionDrainTimeout):
				panic(fmt.Sprintf("rebuild %s timed out while waiting for repartition readiness", t.Name))
			}

			if doRepart {
				// maintenanceMu stays locked; repartition generation retirement
				// releases it after the last old-topology user.
				rebuildClaimed = false
				t.repartitionDDLReadLocked(shardCandidates, tableLease)
			} else {
				// No repartition — release maintenanceMu now
				t.maintenanceMu.Unlock()
				rebuildClaimed = false
			}
		}(t, tableBytes, tableLease)
	}
	done.Wait()

	// Return replaced shards to the caller (Rebuild) so it can delete their
	// on-disk files AFTER db.save() has written the new schema.json.
	// Deleting old column files before saving the schema creates a window
	// where a crash/kill leaves schema.json pointing at already-deleted UUIDs.
	// Failed schema publication intentionally retains the old generation: it is
	// still referenced by the last committed schema and must remain recoverable.
	return rebuildDatabaseResult{
		replaced: allReplaced,
		errors:   rebuildErrors,
		complete: !incomplete.Load() && len(rebuildErrors) == 0,
	}
}

func GetDatabase(schema string, ignoreCase ...bool) *database {
	if len(ignoreCase) != 0 && ignoreCase[0] {
		names := databases.folded.Load()
		if names == nil {
			return nil
		}
		canonical, exists := (*names)[foldIdentifier(schema)]
		if !exists {
			return nil
		}
		if canonical == "" {
			panic("ambiguous database name: " + schema)
		}
		schema = canonical
	}
	return databases.Get(schema)
}

func CreateDatabase(schema string, ignoreexists bool /*, persistence PersistenceFactory*/) bool {
	db := databases.Get(schema)
	if db != nil {
		if ignoreexists {
			return false
		}
		panic("Database " + schema + " already exists")
	}

	db = newDatabase()
	db.Name = schema
	persistence := FileFactory{Basepath} // TODO: remove this, use parameter instead
	db.persistence = instrumentPersistence(schema, persistence.CreateDatabase(schema))
	db.tables = NonLockingReadMap.NewReadMap[string, *table]()
	// Newly created database is live for writes
	db.srState = WRITE

	last := databases.Set(schema, db)
	if last != nil {
		// two concurrent CREATE
		databases.Set(schema, last)
		panic("Database " + schema + " already exists")
	}

	db.save()
	return true
}

func CreateDatabaseWithBackend(schema string, ignoreexists bool, options map[string]string) bool {
	backend := options["backend"]
	if backend == "" || backend == "filesystem" {
		return CreateDatabase(schema, ignoreexists)
	}

	db := databases.Get(schema)
	if db != nil {
		if ignoreexists {
			return false
		}
		panic("Database " + schema + " already exists")
	}

	// Validate backend exists in registry
	if _, ok := BackendRegistry[backend]; !ok {
		panic("Unknown storage backend: " + backend)
	}
	if _, ok := options["prefix"]; !ok {
		options["prefix"] = schema
	}

	raw := marshalDatabaseBackendOptions(options)

	// Write config file
	configPath := Basepath + "/" + schema + ".json"
	if err := os.WriteFile(configPath, raw, 0640); err != nil {
		panic("failed to write backend config: " + err.Error())
	}

	// Create persistence engine from config
	persistence := createPersistenceFromConfig(schema, json.RawMessage(raw))
	if persistence == nil {
		os.Remove(configPath)
		panic("failed to create persistence engine for backend: " + backend)
	}

	db = newDatabase()
	db.Name = schema
	db.persistence = persistence
	db.tables = NonLockingReadMap.NewReadMap[string, *table]()
	db.srState = WRITE

	last := databases.Set(schema, db)
	if last != nil {
		databases.Set(schema, last)
		os.Remove(configPath)
		panic("Database " + schema + " already exists")
	}

	db.save()
	return true
}

func marshalDatabaseBackendOptions(options map[string]string) json.RawMessage {
	configMap := make(map[string]interface{}, len(options))
	for key, value := range options {
		if key == "force_path_style" {
			configMap[key] = value == "true" || value == "1" || value == "TRUE"
			continue
		}
		configMap[key] = value
	}
	raw, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		panic("failed to marshal backend config: " + err.Error())
	}
	return raw
}

func databaseBackendConfig(schema string) json.RawMessage {
	configPath := Basepath + "/" + schema + ".json"
	raw, err := os.ReadFile(configPath)
	if err == nil {
		return raw
	}
	if !os.IsNotExist(err) {
		panic("failed to read database backend config: " + err.Error())
	}
	return json.RawMessage(`{"backend":"filesystem"}`)
}

func databaseBackendConfigForTarget(source string, target string) json.RawMessage {
	var config map[string]interface{}
	if err := json.Unmarshal(databaseBackendConfig(source), &config); err != nil {
		panic("failed to parse source database backend config: " + err.Error())
	}
	if backend, _ := config["backend"].(string); backend != "filesystem" {
		config["prefix"] = target
	}
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		panic("failed to copy source database backend config: " + err.Error())
	}
	return raw
}

func equalDatabaseBackendConfig(left, right json.RawMessage) bool {
	var leftConfig interface{}
	var rightConfig interface{}
	if json.Unmarshal(left, &leftConfig) != nil || json.Unmarshal(right, &rightConfig) != nil {
		return false
	}
	return reflect.DeepEqual(leftConfig, rightConfig)
}

func writeDatabaseBackendConfig(schema string, raw json.RawMessage) {
	if err := os.MkdirAll(Basepath, 0750); err != nil {
		raisePersistenceFailure("filesystem", Basepath, "backend.config.write", err)
	}
	tmp, err := os.CreateTemp(Basepath, "."+schema+".json.tmp-")
	if err != nil {
		raisePersistenceFailure("filesystem", Basepath, "backend.config.write", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0640); err != nil {
		_ = tmp.Close()
		raisePersistenceFailure("filesystem", Basepath, "backend.config.write", err)
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		raisePersistenceFailure("filesystem", Basepath, "backend.config.write", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		raisePersistenceFailure("filesystem", Basepath, "backend.config.write", err)
	}
	if err := tmp.Close(); err != nil {
		raisePersistenceFailure("filesystem", Basepath, "backend.config.write", err)
	}
	if err := os.Rename(tmpName, Basepath+"/"+schema+".json"); err != nil {
		raisePersistenceFailure("filesystem", Basepath, "backend.config.write", err)
	}
	dir, err := os.Open(Basepath)
	if err != nil {
		reportPersistenceAmbiguousFailure("filesystem", Basepath, "backend.config.sync", err)
		return
	}
	if err := dir.Sync(); err != nil {
		_ = dir.Close()
		reportPersistenceAmbiguousFailure("filesystem", Basepath, "backend.config.sync", err)
		return
	}
	if err := dir.Close(); err != nil {
		reportPersistenceAmbiguousFailure("filesystem", Basepath, "backend.config.close", err)
	}
}

func CreateDatabaseFrom(schema string, ignoreexists bool, sourceDB string) bool {
	db := databases.Get(schema)
	if db != nil {
		if ignoreexists {
			return false
		}
		panic("Database " + schema + " already exists")
	}

	if databases.Get(sourceDB) == nil {
		panic("Source database " + sourceDB + " does not exist")
	}
	raw := databaseBackendConfigForTarget(sourceDB, schema)
	writeDatabaseBackendConfig(schema, raw)

	// Create persistence engine
	persistence := createPersistenceFromConfig(schema, json.RawMessage(raw))
	if persistence == nil {
		os.Remove(Basepath + "/" + schema + ".json")
		panic("failed to create persistence engine from source config")
	}

	db = newDatabase()
	db.Name = schema
	db.persistence = persistence
	db.tables = NonLockingReadMap.NewReadMap[string, *table]()
	db.srState = WRITE

	last := databases.Set(schema, db)
	if last != nil {
		databases.Set(schema, last)
		os.Remove(Basepath + "/" + schema + ".json")
		panic("Database " + schema + " already exists")
	}

	db.save()
	return true
}

type storageMoveGeneration struct {
	table             *table
	oldTopology       *tableShardTopology
	oldSchemaTopology *tableSchemaTopology
	oldShards         []*storageShard
	newShards         []*storageShard
	partitioned       bool
}

func prepareStorageMoveGeneration(t *table) (generation storageMoveGeneration) {
	t.maintenanceMu.Lock()
	t.mu.Lock()
	if t.maintenanceKind != 0 {
		t.mu.Unlock()
		t.maintenanceMu.Unlock()
		panic("storage migration raced table maintenance for " + t.schema.Name + "." + t.Name)
	}
	t.maintenanceKind = 1
	topology := t.activeTopology()
	oldShards := append([]*storageShard(nil), topology.shards...)
	partitioned := topology.mode == ShardModePartition
	t.mu.Unlock()

	generation = storageMoveGeneration{
		table:             t,
		oldTopology:       topology,
		oldSchemaTopology: t.schemaTopology.Load(),
		oldShards:         oldShards,
		newShards:         make([]*storageShard, len(oldShards)),
		partitioned:       partitioned,
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			abortStorageMoveGenerations([]storageMoveGeneration{generation})
			panic(recovered)
		}
	}()
	for index, shard := range oldShards {
		if shard != nil {
			generation.newShards[index] = shard.rebuild(true)
		}
	}
	return generation
}

func abortStorageMoveGenerations(generations []storageMoveGeneration) {
	for _, generation := range generations {
		for index, shard := range generation.oldShards {
			if shard == nil || generation.newShards[index] == nil {
				continue
			}
			shard.clearNext(generation.newShards[index])
			GlobalCache.Remove(generation.newShards[index])
			discardUnpublishedShard(generation.newShards[index])
		}
		generation.table.mu.Lock()
		generation.table.schemaTopology.Store(generation.oldSchemaTopology)
		if generation.partitioned {
			generation.table.PShards = generation.oldShards
		} else {
			generation.table.Shards = generation.oldShards
		}
		generation.table.maintenanceKind = 0
		generation.table.mu.Unlock()
		generation.table.maintenanceMu.Unlock()
	}
}

func replayPersistenceLog(engine PersistenceEngine, name string) []interface{} {
	_, input, logfile := engine.ReplayLog(name)
	entries := make([]interface{}, 0)
	for entry := range input {
		entries = append(entries, entry)
	}
	logfile.Close()
	return entries
}

func movePreparedShardLogs(src, dst PersistenceEngine, generations []storageMoveGeneration) {
	for _, generation := range generations {
		for _, shard := range generation.newShards {
			if shard == nil {
				continue
			}
			shard.mu.Lock()
			if shard.logfile != nil {
				shard.logfile.Flush(shard.t.PersistencyMode == Safe)
				shard.logfile.Close()
				shard.logfile = dst.SwapLog(shard.uuid.String(), replayPersistenceLog(src, shard.uuid.String()), shard.t.PersistencyMode == Safe)
			}
			shard.mu.Unlock()
		}
	}
}

func moveTransactionLog(db *database, dst PersistenceEngine) PersistenceLogfile {
	db.transactionMu.Lock()
	defer db.transactionMu.Unlock()
	if db.transactionLog == nil {
		return nil
	}
	ids := make([]string, 0, len(db.committedTx))
	for id := range db.committedTx {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	entries := make([]interface{}, len(ids))
	for index, id := range ids {
		entries[index] = LogEntryCommit{txID: id}
	}
	replacement := dst.SwapLog(db.transactionLogKey(), entries, true)
	old := db.transactionLog
	db.transactionLog = replacement
	return old
}

func publishStorageMoveGenerations(db *database, dst PersistenceEngine, generations []storageMoveGeneration, targetConfig json.RawMessage) {
	// Build the destination catalog from the private successors without making
	// them visible through the tables' atomic topology pointers. The direct
	// shard fields are only serialization inputs; mutations use activeTopology.
	for _, generation := range generations {
		generation.table.mu.Lock()
		if generation.partitioned {
			generation.table.PShards = generation.newShards
		} else {
			generation.table.Shards = generation.newShards
		}
		generation.table.publishSchemaTopology(
			generation.oldTopology.mode,
			generation.newShards,
			generation.oldTopology.dimensions,
		)
		generation.table.mu.Unlock()
	}

	snapshot, err := json.MarshalIndent(db, "", "  ")
	for index := len(generations) - 1; index >= 0; index-- {
		generation := generations[index]
		generation.table.mu.Lock()
		generation.table.schemaTopology.Store(generation.oldSchemaTopology)
		if generation.partitioned {
			generation.table.PShards = generation.oldShards
		} else {
			generation.table.Shards = generation.oldShards
		}
		generation.table.mu.Unlock()
	}
	if err != nil {
		panic(err)
	}
	if writer, ok := dst.(schemaWriteOptions); ok {
		writer.WriteSchemaWithMode(snapshot, true)
	} else {
		dst.WriteSchema(snapshot)
	}
	writeDatabaseBackendConfig(db.Name, targetConfig)
	db.persistence = dst

	for _, generation := range generations {
		generation.table.mu.Lock()
		if generation.partitioned {
			generation.table.PShards = generation.newShards
		} else {
			generation.table.Shards = generation.newShards
		}
		generation.table.publishTopologyLocked()
		generation.table.maintenanceKind = 0
		generation.table.mu.Unlock()
		generation.table.maintenanceMu.Unlock()
	}
}

func retireSourceStorage(src PersistenceEngine, generations []storageMoveGeneration, oldTransactionLog PersistenceLogfile) {
	cleanup := func() {
		for _, generation := range generations {
			<-generation.oldTopology.drained
			for _, shard := range generation.oldShards {
				if shard != nil && shard.logfile != nil {
					func() { defer func() { _ = recover() }(); shard.logfile.Close() }()
				}
			}
		}
		if oldTransactionLog != nil {
			func() { defer func() { _ = recover() }(); oldTransactionLog.Close() }()
		}
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					fmt.Println("error: old database storage cleanup failed after cutover:", recovered)
				}
			}()
			src.Remove()
		}()
	}
	for _, generation := range generations {
		select {
		case <-generation.oldTopology.drained:
		default:
			go cleanup()
			return
		}
	}
	cleanup()
}

// AlterDatabaseStorage builds private shard successors while readers remain on
// the published generation. Source mutations are mirrored by the existing
// rebuild chain. Only schema writers are excluded; publication swaps every
// table topology after the complete destination schema is durable.
func AlterDatabaseStorage(schema string, targetConfig json.RawMessage, currentTx *TxContext) bool {
	db := GetDatabase(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}
	requireDatabaseMaintenance(schema, maintenanceAlter)
	dst := createPersistenceFromConfig(schema, targetConfig)
	if dst == nil {
		panic("unknown or invalid storage backend")
	}
	if SessionStateFromTx(currentTx) == nil {
		panic("ALTER DATABASE storage requires a query session")
	}

	db.ensureLoaded()
	// The blob catalog must exist before the table set is frozen. Rebuild may
	// update it, and no table may appear while the destination generation is
	// being prepared.
	db.ensureBlobTable()
	defer db.GetWrite(nil).Release()
	db.persistenceLifecycle.Lock()
	defer db.persistenceLifecycle.Unlock()
	if r := replication.Load(); r != nil && r.hasTransfers(schema) {
		panic("database has active replication transfers; retry ALTER DATABASE storage")
	}
	if GetDatabase(schema) != db {
		panic("Database " + schema + " was dropped while waiting for storage migration")
	}
	if equalDatabaseBackendConfig(databaseBackendConfig(schema), targetConfig) {
		return true
	}
	src := db.persistence
	sameStorage := src.StorageIdentity() == dst.StorageIdentity()
	if !sameStorage && len(dst.ReadSchema()) != 0 {
		panic("destination storage already contains a database schema")
	}
	// A same-namespace change only replaces connection/configuration metadata.
	if sameStorage {
		db.schemalock.Lock()
		defer db.schemalock.Unlock()
		writeDatabaseBackendConfig(schema, targetConfig)
		db.persistence = dst
		return true
	}

	generations := make([]storageMoveGeneration, 0)
	var oldTransactionLog PersistenceLogfile
	published := false
	tables := db.tables.GetAll()
	sort.Slice(tables, func(i, j int) bool {
		if tables[i].Name == ".blobs" {
			return false
		}
		if tables[j].Name == ".blobs" {
			return true
		}
		return tables[i].Name < tables[j].Name
	})
	for _, table := range tables {
		table.ddlMu.RLock()
	}
	defer func() {
		if !published {
			if oldTransactionLog != nil {
				db.transactionMu.Lock()
				failedTargetLog := db.transactionLog
				db.transactionLog = oldTransactionLog
				db.transactionMu.Unlock()
				if failedTargetLog != nil {
					func() { defer func() { _ = recover() }(); failedTargetLog.Close() }()
				}
			}
			abortStorageMoveGenerations(generations)
			func() { defer func() { _ = recover() }(); dst.Remove() }()
		}
		for index := len(tables) - 1; index >= 0; index-- {
			tables[index].ddlMu.RUnlock()
		}
	}()
	for _, table := range tables {
		generations = append(generations, prepareStorageMoveGeneration(table))
	}

	// Columns and blobs are immutable for the prepared successors. Writes keep
	// entering their WAL/delta through the source generation's rebuild link.
	copyDatabaseObjects(src, dst)
	movePreparedShardLogs(src, dst, generations)
	oldTransactionLog = moveTransactionLog(db, dst)
	func() {
		db.schemalock.Lock()
		defer db.schemalock.Unlock()
		publishStorageMoveGenerations(db, dst, generations, targetConfig)
	}()
	published = true
	retireSourceStorage(src, generations, oldTransactionLog)
	return true
}

func DatabaseBackendName(schema string) string {
	db := databases.Get(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}
	return db.persistence.BackendName()
}

func DropDatabase(schema string, ifexists bool) bool {
	db := databases.Get(schema)
	if db == nil {
		if ifexists {
			return false
		}
		panic("Database " + schema + " does not exist")
	}
	db.persistenceLifecycle.Lock()
	defer db.persistenceLifecycle.Unlock()
	if r := replication.Load(); r != nil && r.hasTransfers(schema) {
		panic("database has active replication transfers; retry DROP DATABASE")
	}
	requireDatabaseMaintenance(schema, maintenanceDrop)
	db = databases.Remove(schema)
	if db == nil {
		if ifexists {
			return false
		}
		panic("Database " + schema + " does not exist")
	}

	// clean up shards/indexes/temp columns from GlobalCache
	db.ensureLoaded()
	for _, t := range db.tables.GetAll() {
		GlobalCache.Remove(t) // temp keytable
		for _, c := range t.Columns {
			if c.IsTemp {
				GlobalCache.Remove(c)
			}
		}
		for _, s := range t.Shards {
			GlobalCache.Remove(s)
		}
		for _, s := range t.PShards {
			if s != nil {
				GlobalCache.Remove(s)
			}
		}
	}

	// remove remains of the folder structure
	db.closeTransactionLog()
	db.persistence.Remove()
	// also remove backend config file if it exists
	os.Remove(Basepath + "/" + schema + ".json")
	return true
}

func CreateTable(schema, name string, pm PersistencyMode, ifnotexists bool) (*table, bool) {
	db := GetDatabase(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}
	if ifnotexists {
		if existing := db.tables.Get(name); existing != nil {
			atomic.StoreUint64(&existing.lastAccessed, uint64(time.Now().UnixNano()))
			return existing, false
		}
	}
	if !(pm == Cache && strings.HasPrefix(name, ".")) {
		defer db.GetWrite(nil).Release()
	}
	db.persistenceLifecycle.RLock()
	defer db.persistenceLifecycle.RUnlock()
	db.ensureLoaded()
	db.schemalock.Lock()
	t, created := db.createTableLocked(name, pm, ifnotexists)
	if !created {
		db.schemalock.Unlock()
		return t, false
	}
	mode := t.schemaSaveMode()
	if t.isEphemeralQueryTable() {
		mode = schemaSaveBuffered
	}
	db.saveLockedAndUnlock(mode)
	registerCreatedTable(t)
	return t, true
}

// createTableLocked mutates db.tables while db.schemalock is held but does
// not persist schema.json yet. Callers must follow up with saveLockedAndUnlock.
func (db *database) createTableLocked(name string, pm PersistencyMode, ifnotexists bool) (*table, bool) {
	t := db.tables.Get(name)
	if t != nil {
		if ifnotexists {
			atomic.StoreUint64(&t.lastAccessed, uint64(time.Now().UnixNano()))
			return t, false
		}
		panic("Table " + name + " already exists")
	}
	t = db.newTable(name, pm)
	if existing := db.tables.Set(name, t); existing != nil {
		panic("Table " + name + " already exists")
	}
	if !t.isHiddenFromShowTables() {
		db.publishTableNamesLocked()
	}
	return t, true
}

// newTable builds a complete table object without publishing it in db.tables.
// Callers may add internal columns before publication while holding schemalock.
func (db *database) newTable(name string, pm PersistencyMode) *table {
	t := new(table)
	t.DroppedColumns = make(map[string]bool)
	t.schema = db
	t.Name = name
	t.PersistencyMode = pm
	t.ShardMode = ShardModeFree
	t.lastAccessed = uint64(time.Now().UnixNano())
	t.Shards = make([]*storageShard, 1)
	t.Shards[0] = NewShard(t)
	t.publishTopologyLocked()
	t.Auto_increment = 0
	t.publishShowColumnsSnapshot(true)
	return t
}

func registerCreatedTable(t *table) {
	// register temp keytable with CacheManager AFTER releasing schemalock
	// to avoid deadlock: AddItem → run() → evict → keytableCleanup → TryLock(schemalock)
	if t.isEphemeralQueryTable() {
		GlobalCache.AddItem(t, int64(t.exclusiveSize()), TypeTempKeytable, func(ptr any, freedByType *[numEvictableTypes]int64) bool {
			return keytableCleanup(ptr.(*table), freedByType)
		}, keytableLastUsed, nil)
	} else if t.PersistencyMode == Cache {
		// Register the initial shard so eviction can reach it before the first rebuild.
		GlobalCache.AddItem(t.Shards[0], int64(t.Shards[0].exclusiveSize()), TypeCacheEntry, cacheShardCleanup, shardLastUsed, nil)
	} else if t.PersistencyMode != Memory {
		// The initial writable generation owns WAL/delta memory before its first
		// rebuild just as much as a rebuilt shard does.
		GlobalCache.AddItem(t.Shards[0], int64(t.Shards[0].exclusiveSize()), TypeShard, shardCleanup, shardLastUsed, nil)
	}
}

func DropTable(schema, name string, ifexists bool) {
	db := GetDatabase(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}
	schemaAccess := db.GetWrite(nil)
	defer func() { schemaAccess.Release() }()
	db.persistenceLifecycle.RLock()
	persistenceLifecycleLocked := true
	defer func() {
		if persistenceLifecycleLocked {
			db.persistenceLifecycle.RUnlock()
		}
	}()
	db.ensureLoaded()
	db.schemalock.Lock()
	t := db.tables.Get(name)
	if t == nil {
		db.schemalock.Unlock()
		if ifexists {
			return // silentfail
		}
		panic("Table " + schema + "." + name + " does not exist")
	}
	if !tableMaintenanceCapabilities(schema, name).canDrop {
		db.schemalock.Unlock()
		requireTableMaintenance(schema, name, maintenanceDrop)
	}
	db.tables.Remove(name)
	if !t.isHiddenFromShowTables() {
		db.publishTableNamesLocked()
	}
	if name == ".blobs" {
		db.blobRefState().table.Store(nil)
	}
	db.saveLockedAndUnlock(t.schemaSaveMode())

	// deregister temp keytable from CacheManager (no-op if not registered or already evicted)
	// Must be AFTER schemalock.Unlock to avoid deadlock: Remove → run() → evict → keytableCleanup → TryLock
	GlobalCache.Remove(t)
	// deregister temp columns from CacheManager
	for _, c := range t.Columns {
		if c.IsTemp {
			GlobalCache.Remove(c)
		}
	}
	// deregister shards and delete from disk
	for _, s := range t.Shards {
		GlobalCache.Remove(s)
		s.RemoveFromDisk()
	}
	for _, s := range t.PShards {
		if s != nil {
			GlobalCache.Remove(s)
			s.RemoveFromDisk()
		}
	}
	db.persistenceLifecycle.RUnlock()
	persistenceLifecycleLocked = false
	// Fire AfterDropTable triggers after releasing catalog and persistence
	// lifecycle locks. Cascading drops may perform DDL in this database again.
	schemaAccess.Release()
	schemaAccess = WriteAccess{}
	t.ExecuteTableLifecycleTriggers(AfterDropTable, nil)
}

func RenameTable(schema, oldname, newname string) {
	db := GetDatabase(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}
	defer db.GetWrite(nil).Release()
	db.persistenceLifecycle.RLock()
	defer db.persistenceLifecycle.RUnlock()
	db.ensureLoaded()
	if r := replication.Load(); r != nil && validReplicaDatabase(schema) {
		if target := db.tables.Get(oldname); target != nil {
			for _, shard := range target.ActiveShards() {
				if shard != nil {
					r.obtain(*r.shardCell(shard).resource.Load(), mosi.M, "", nil)
				}
			}
		}
	}
	db.schemalock.Lock()
	t := db.tables.Get(oldname)
	if t == nil {
		db.schemalock.Unlock()
		panic("Table " + schema + "." + oldname + " does not exist")
	}
	if !tableMaintenanceCapabilities(schema, oldname).canRename {
		db.schemalock.Unlock()
		requireTableMaintenance(schema, oldname, maintenanceRename)
	}
	if !tableMaintenanceCapabilities(schema, newname).canRename {
		db.schemalock.Unlock()
		requireTableMaintenance(schema, newname, maintenanceRename)
	}
	if db.tables.Get(newname) != nil {
		db.schemalock.Unlock()
		panic("Table " + schema + "." + newname + " already exists")
	}
	db.tables.Remove(oldname)
	t.Name = newname
	t.publishedName.Store(&newname)
	db.tables.Set(newname, t)
	db.publishTableNamesLocked()
	db.saveLockedAndUnlock(t.schemaSaveMode())
}

// keytableCleanup retires a planner-owned in-memory cache on the manager
// goroutine. It performs only nonblocking metadata operations: no DDL, schema
// I/O, disk deletion, Scheme execution, or public CacheManager calls.
func keytableCleanup(tbl *table, freedByType *[numEvictableTypes]int64) bool {
	if !tbl.isEphemeralQueryTable() {
		return false
	}
	// Registration publishes immutable reverse edges, so cleanup need not visit
	// unrelated tables or execute name-based self-cleanup triggers. Rebinding a
	// same-named cache is safe: only triggers carrying this exact target retire.
	sources := []*table{tbl}
	head := tbl.cacheTriggerSources.Load()
	for edge := head; edge != nil; edge = edge.next {
		sources = append(sources, edge.table)
	}
	var schemas []*database
	var locked []*table
	defer func() {
		for i := len(locked) - 1; i >= 0; i-- {
			locked[i].mu.Unlock()
		}
		for i := len(schemas) - 1; i >= 0; i-- {
			schemas[i].schemalock.Unlock()
		}
	}()
	// Schema serialization reads trigger metadata under schemalock. Acquire the
	// existing locks without waiting, before changing anything; a busy source
	// leaves the cache and all dependencies intact for the next eviction pass.
	for _, source := range sources {
		db := source.schema
		seen := false
		for _, held := range schemas {
			if held == db {
				seen = true
				break
			}
		}
		if !seen {
			if !db.schemalock.TryLock() {
				return false
			}
			schemas = append(schemas, db)
		}
	}
	for _, source := range sources {
		if !source.mu.TryLock() {
			return false
		}
		locked = append(locked, source)
	}
	// Evict leaves first. Removing an intermediate cache would disconnect the
	// invalidation path of its live consumers. Their own eviction removes the
	// reverse edges and makes this table eligible without cascading DDL.
	for _, trigger := range tbl.Triggers {
		if target := trigger.cacheTarget; target != nil && target != tbl && atomic.LoadInt64(&target.cacheUsers) >= 0 {
			return false
		}
	}
	if !tbl.beginCacheEviction() {
		return false
	}
	// A registrant may have published another source before its last cache-use
	// pin drained. Retry with the complete snapshot instead of missing an edge.
	if tbl.cacheTriggerSources.Load() != head {
		atomic.StoreInt64(&tbl.cacheUsers, 0)
		return false
	}
	if !tbl.evictShardChildren(freedByType) {
		atomic.StoreInt64(&tbl.cacheUsers, 0)
		return false
	}
	for _, source := range sources[1:] {
		kept := source.Triggers[:0]
		for _, trigger := range source.Triggers {
			if trigger.cacheTarget != tbl {
				kept = append(kept, trigger)
			}
		}
		if len(kept) != len(source.Triggers) {
			clear(source.Triggers[len(kept):])
			source.Triggers = kept
			// These are reconstructible runtime dependencies, never user DDL.
			source.schema.schemaDirty.Store(true)
		}
	}
	tbl.cacheTriggerSources.Store(nil)
	db := tbl.schema
	if db.tables.Get(tbl.Name) == tbl {
		db.tables.Remove(tbl.Name)
		db.schemaDirty.Store(true)
	}
	for _, c := range tbl.Columns {
		if c.IsTemp {
			GlobalCache.removeInternal(c, freedByType)
		}
	}
	for _, shard := range tbl.ActiveShards() {
		if shard != nil {
			GlobalCache.removeInternal(shard, freedByType)
		}
	}
	return true
}

func keytableLastUsed(ptr any) time.Time {
	tbl := ptr.(*table)
	return time.Unix(0, int64(atomic.LoadUint64(&tbl.lastAccessed)))
}

func (t *table) evictShardChildren(freedByType *[numEvictableTypes]int64) bool {
	for _, shard := range t.ActiveShards() {
		if shard == nil {
			continue
		}
		if !shard.mu.TryLock() {
			return false
		}
		ok := shard.evictChildrenLocked(freedByType)
		shard.mu.Unlock()
		if !ok {
			return false
		}
	}
	return true
}

func (db *database) importClusterCommits(owner string) error {
	if err := validateClusterURI(owner); err != nil {
		return err
	}
	// A cold generation can pass through several owners without loading or
	// rebuilding its WAL. The immediate sender's own commit stream alone is
	// therefore insufficient. Membership retains unreachable installations;
	// these ordinary backend logs remain readable without contacting the node.
	owners := map[string]bool{owner: true}
	if cluster := activeNodeCluster(); cluster != nil {
		if view := cluster.view.Load(); view != nil {
			for _, member := range view.members {
				owners[member.ID] = true
			}
		}
	}
	commits := make([]string, 0)
	for owner := range owners {
		if err := validateClusterURI(owner); err != nil {
			return err
		}
		id := strings.TrimPrefix(owner, "memcp://")
		_, entries, handle := db.persistence.ReplayLog(transactionLogName + "." + id)
		for entry := range entries {
			commit, ok := entry.(LogEntryCommit)
			if !ok || commit.txID == "" {
				handle.Close()
				return fmt.Errorf("invalid native commit log")
			}
			commits = append(commits, commit.txID)
		}
		handle.Close()
	}
	db.initializeTransactionLog()
	db.transactionMu.Lock()
	for _, txID := range commits {
		if _, exists := db.committedTx[txID]; !exists {
			db.committedTx[txID] = db.transactionGen
		}
	}
	db.transactionMu.Unlock()
	return nil
}
