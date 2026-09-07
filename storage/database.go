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
import "encoding/json"
import "reflect"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/NonLockingReadMap"

type database struct {
	Name        string                                     `json:"name"`
	persistence PersistenceEngine                          `json:"-"`
	tables      *NonLockingReadMap.ReadMap[string, *table] `json:"-"`
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
	// persistenceLifecycle prevents cleanup from inspecting generation-private
	// files between their write and schema publication. Rebuild/repartition take
	// a read capability; cleanup takes the exclusive capability. Query and DML
	// paths do not participate.
	persistenceLifecycle sync.RWMutex `json:"-"`
	// storageMoveMu serializes backend migration with rare catalog membership
	// changes. Query/DML paths and idempotent planner setup never enter it.
	storageMoveMu sync.Mutex `json:"-"`
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
	return &database{
		tables:      NonLockingReadMap.NewReadMap[string, *table](),
		blobRefs:    new(blobRefState),
		committedTx: make(map[string]uint64),
	}
}

const transactionLogName = ".transactions"

func (db *database) initializeTransactionLog() {
	db.transactionOnce.Do(func() {
		committed := make(map[string]uint64)
		_, entries, logfile := db.persistence.ReplayLog(transactionLogName)
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
	replacement := db.persistence.SwapLog(transactionLogName, entries, true)
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
		Name   string                                     `json:"name"`
		Tables *NonLockingReadMap.ReadMap[string, *table] `json:"tables"`
	}
	return json.Marshal(persist{Name: d.Name, Tables: d.tables})
}

func (d *database) UnmarshalJSON(data []byte) error {
	type persist struct {
		Name   string                                     `json:"name"`
		Tables *NonLockingReadMap.ReadMap[string, *table] `json:"tables"`
	}
	var p persist
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	d.Name = p.Name
	d.tables = p.Tables
	if d.tables == nil {
		d.tables = NonLockingReadMap.NewReadMap[string, *table]()
	}
	return nil
}

var databases = NonLockingReadMap.NewReadMap[string, *database]()
var Basepath string = "data"

func (d *database) ComputeSize() uint {
	var sz uint = 16 * 8 // heuristic
	for _, t := range d.tables.GetAll() {
		sz += t.ComputeSize()
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
		errs = append(errs, rebuildDatabaseAndCompact(db, all, repartition, includeEphemeral)...)
	}
	duration := fmt.Sprint(time.Since(start))
	if len(errs) == 0 {
		return duration
	}
	return duration + " errors: " + strings.Join(errs, " | ")
}

func rebuildDatabaseAndCompact(db *database, all bool, repartition bool, includeEphemeral bool) []string {
	db.persistenceLifecycle.RLock()
	defer db.persistenceLifecycle.RUnlock()
	var transactionCutoff uint64
	var hasTransactions bool
	if err := func() (err error) {
		defer func() { err = recoverAsError("transaction log snapshot failed for database " + db.Name) }()
		transactionCutoff, hasTransactions = db.transactionCompactionSnapshot()
		return nil
	}(); err != nil {
		return []string{err.Error()}
	}
	result := db.rebuildWithLifecycle(all, repartition, includeEphemeral)
	if len(result.errors) > 0 {
		return result.errors
	}
	if err := func() (err error) {
		defer func() { err = recoverAsError("save failed for database " + db.Name) }()
		db.save()
		return nil
	}(); err != nil {
		return []string{err.Error()}
	}
	// db.save() committed all replacement UUIDs. A save failure returns above
	// and deliberately retains every old generation.
	errs := cleanupReplacedShardGenerations(result.replaced, "database "+db.Name)
	if len(errs) != 0 || !result.complete || !hasTransactions {
		return errs
	}
	if err := func() (err error) {
		defer func() { err = recoverAsError("transaction log compaction failed for database " + db.Name) }()
		db.compactTransactionLog(transactionCutoff)
		return nil
	}(); err != nil {
		errs = append(errs, err.Error())
	}
	return errs
}

func UnloadDatabases() {
	fmt.Println("table compression done in ", rebuildDatabases(false, false, true))
	for _, db := range databases.GetAll() {
		db.closeTransactionLog()
	}
	data, _ := json.Marshal(Settings)
	if settings, err := os.OpenFile(Basepath+"/settings.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640); err == nil {
		defer settings.Close()
		settings.Write(data)
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

func (db *database) commitSchemaSnapshot(jsonbytes []byte, durable bool) {
	// Serialize publications without holding schemalock. Coalescing may skip an
	// intermediate in-memory snapshot, but every backend call still publishes a
	// complete generation and every waiter observes its success or panic.
	db.saveMu.Lock()
	cond := db.getSaveCond()
	db.saveRequested++
	seq := db.saveRequested
	db.savePending = jsonbytes
	db.savePendingSync = db.savePendingSync || durable
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
	if db.srState == COLD {
		// Do not serialize a cold database; keep existing schema.json intact
		return
	}
	db.schemalock.RLock()
	jsonbytes, _ := json.MarshalIndent(db, "", "  ")
	// Buffered mutations after this snapshot must leave the catalog dirty.
	// They need the exclusive schema lock and therefore cannot race this store.
	db.schemaDirty.Store(false)
	db.schemalock.RUnlock()
	func() {
		defer func() {
			if r := recover(); r != nil {
				db.schemaDirty.Store(true)
				panic(r)
			}
		}()
		db.commitSchemaSnapshot(jsonbytes, true)
	}()
	// shards are written while rebuild
}

// saveLockedAndUnlock publishes according to mode and always releases
// schemalock. Buffered temp metadata is immediately visible in memory; a later
// synchronous schema save or rebuild includes it in its complete snapshot.
func (db *database) saveLockedAndUnlock(mode schemaSaveMode) {
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
	db.schemalock.Unlock()
	defer func() {
		if r := recover(); r != nil {
			db.schemaDirty.Store(true)
			panic(r)
		}
	}()
	db.commitSchemaSnapshot(jsonbytes, mode == schemaSaveFsync)
}

// ensureLoaded loads schema.json into the database struct exactly once.
func (db *database) ensureLoaded() {
	db.loadOnce.Do(func() {
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
			t.publishShowColumnsSnapshot()
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

// invalidatePersistedPlannerCodeAfterLoad drops executable code derived from a
// physical query plan. Planner-owned CACHE tables contain no durable rows, and
// the current createtable guard supplies their authoritative oninit callback on
// first use. Retaining an older callback across a physical operator ABI change
// would execute stale generated code before that guard can refresh it.
func invalidatePersistedPlannerCodeAfterLoad(t *table) {
	if t.isEphemeralQueryTable() {
		t.OnInit = nil
	}
}

// SharedResource impl for database
func (db *database) GetState() SharedState { return db.srState }
func (db *database) GetRead() func()       { db.ensureLoaded(); return func() {} }
func (db *database) GetExclusive() func()  { db.ensureLoaded(); db.srState = WRITE; return func() {} }

// helper to fetch a table with lazy db load
func (db *database) GetTable(name string) *table {
	db.ensureLoaded()
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
				cold := shard.srState == COLD
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
			t.publishShowColumnsSnapshot()
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
			if repartition && !hasColdShard {
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
						desiredShards := int(1 + (2*maincount)/Settings.ShardSize)
						minShards := 2 * runtime.NumCPU()
						if desiredShards < minShards && maincount > Settings.ShardSize {
							desiredShards = minShards
						}
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

func GetDatabase(schema string) *database {
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
	table       *table
	oldTopology *tableShardTopology
	oldShards   []*storageShard
	newShards   []*storageShard
	partitioned bool
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
		table:       t,
		oldTopology: topology,
		oldShards:   oldShards,
		newShards:   make([]*storageShard, len(oldShards)),
		partitioned: partitioned,
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
	replacement := dst.SwapLog(transactionLogName, entries, true)
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
		generation.table.mu.Unlock()
	}

	snapshot, err := json.MarshalIndent(db, "", "  ")
	for index := len(generations) - 1; index >= 0; index-- {
		generation := generations[index]
		generation.table.mu.Lock()
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
	db.storageMoveMu.Lock()
	defer db.storageMoveMu.Unlock()
	if GetDatabase(schema) != db {
		panic("Database " + schema + " was dropped while waiting for storage migration")
	}
	if equalDatabaseBackendConfig(databaseBackendConfig(schema), targetConfig) {
		return true
	}
	// Create internal schema before taking the exclusive schema-generation lock.
	db.ensureBlobTable()
	src := db.persistence
	sameStorage := src.StorageIdentity() == dst.StorageIdentity()
	if !sameStorage && len(dst.ReadSchema()) != 0 {
		panic("destination storage already contains a database schema")
	}
	db.persistenceLifecycle.Lock()
	defer db.persistenceLifecycle.Unlock()

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
	db.storageMoveMu.Lock()
	defer db.storageMoveMu.Unlock()
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
	db.storageMoveMu.Lock()
	defer db.storageMoveMu.Unlock()
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
	return t, true
}

// newTable builds a complete table object without publishing it in db.tables.
// Callers may add internal columns before publication while holding schemalock.
func (db *database) newTable(name string, pm PersistencyMode) *table {
	t := new(table)
	t.schema = db
	t.Name = name
	t.PersistencyMode = pm
	t.ShardMode = ShardModeFree
	t.lastAccessed = uint64(time.Now().UnixNano())
	t.Shards = make([]*storageShard, 1)
	t.Shards[0] = NewShard(t)
	t.publishTopologyLocked()
	t.Auto_increment = 0
	t.publishShowColumnsSnapshot()
	return t
}

func registerCreatedTable(t *table) {
	// register temp keytable with CacheManager AFTER releasing schemalock
	// to avoid deadlock: AddItem → run() → evict → keytableCleanup → TryLock(schemalock)
	if t.isEphemeralQueryTable() {
		schemaName := t.schema.Name
		GlobalCache.AddItem(t, int64(t.ComputeSize()), TypeTempKeytable, func(ptr any, freedByType *[numEvictableTypes]int64) bool {
			return keytableCleanup(ptr.(*table), schemaName, freedByType)
		}, keytableLastUsed, nil)
	} else if t.PersistencyMode == Cache {
		// Register the initial shard so eviction can reach it before the first rebuild.
		GlobalCache.AddItem(t.Shards[0], int64(t.Shards[0].ComputeSize()), TypeCacheEntry, cacheShardCleanup, shardLastUsed, nil)
	} else if t.PersistencyMode != Memory {
		// The initial writable generation owns WAL/delta memory before its first
		// rebuild just as much as a rebuilt shard does.
		GlobalCache.AddItem(t.Shards[0], int64(t.Shards[0].ComputeSize()), TypeShard, shardCleanup, shardLastUsed, nil)
	}
}

func DropTable(schema, name string, ifexists bool) {
	db := GetDatabase(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}
	db.storageMoveMu.Lock()
	defer db.storageMoveMu.Unlock()
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
	if name == ".blobs" {
		db.blobRefState().table.Store(nil)
	}
	db.saveLockedAndUnlock(t.schemaSaveMode())
	// fire AfterDropTable triggers after releasing schemalock (avoids deadlock on cascading drops)
	t.ExecuteTableLifecycleTriggers(AfterDropTable, nil)

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
}

func RenameTable(schema, oldname, newname string) {
	db := GetDatabase(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}
	db.storageMoveMu.Lock()
	defer db.storageMoveMu.Unlock()
	db.ensureLoaded()
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
	db.tables.Set(newname, t)
	db.saveLockedAndUnlock(t.schemaSaveMode())
}

// keytableCleanup is called by the CacheManager when evicting a temp keytable.
// MUST NOT call public GlobalCache.Remove (deadlock: we're inside the CacheManager goroutine).
// MUST NOT use Lock on schemalock (deadlock: CreateTable holds schemalock → AddItem → evict → here).
// Returns false if the schemalock is busy (item pushed back for later retry).
func keytableCleanup(tbl *table, schemaName string, freedByType *[numEvictableTypes]int64) bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("error: keytableCleanup panic for", schemaName+"."+tbl.Name, ":", r)
		}
	}()
	// drop the table directly (bypass DropTable to avoid deadlock on opChan)
	db := GetDatabase(schemaName)
	if db != nil {
		if !db.storageMoveMu.TryLock() {
			return false // a storage generation is retaining this catalog member
		}
		defer db.storageMoveMu.Unlock()
		if !db.schemalock.TryLock() {
			return false // schemalock is held (e.g. by CreateTable); retry later
		}
		if db.tables.Get(tbl.Name) != tbl {
			db.schemalock.Unlock()
			return true
		}
		if !tbl.beginCacheEviction() {
			db.schemalock.Unlock()
			return false
		}
		db.tables.Remove(tbl.Name)
		db.saveLockedAndUnlock(tbl.schemaSaveMode())
	} else if !tbl.beginCacheEviction() {
		return false
	}
	// The table's self-cleanup hooks remove exactly the source-table triggers
	// installed for its computed columns. Trigger target pins above make this
	// safe even when a writer snapshotted a trigger concurrently.
	tbl.ExecuteTableLifecycleTriggers(AfterDropTable, nil)
	// remove all shard+index+temp column registrations for this table (recursive)
	for _, c := range tbl.Columns {
		if c.IsTemp {
			GlobalCache.removeInternal(c, freedByType)
		}
	}
	for _, s := range tbl.Shards {
		GlobalCache.removeInternal(s, freedByType)
		for _, idx := range s.Indexes {
			GlobalCache.removeInternal(idx, freedByType)
			idx.evict(evictFull, 0, freedByType)
		}
	}
	for _, s := range tbl.PShards {
		GlobalCache.removeInternal(s, freedByType)
		for _, idx := range s.Indexes {
			GlobalCache.removeInternal(idx, freedByType)
			idx.evict(evictFull, 0, freedByType)
		}
	}
	for _, s := range tbl.Shards {
		s.RemoveFromDisk()
	}
	for _, s := range tbl.PShards {
		s.RemoveFromDisk()
	}
	return true
}

func keytableLastUsed(ptr any) time.Time {
	tbl := ptr.(*table)
	return time.Unix(0, int64(atomic.LoadUint64(&tbl.lastAccessed)))
}
