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
import "context"
import "sync/atomic"
import "encoding/json"
import "github.com/google/uuid"
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
	catalog                 atomic.Pointer[catalogSnapshot] `json:"-"`
	incarnation             atomic.Pointer[string]          `json:"-"`
	catalogVersion          uint64                          `json:"-"`
	catalogPublicationMu    sync.Mutex                      `json:"-"`
	metadataMemoryBytes     atomic.Uint64                   // producer-owned native DB metadata; never persisted
	localCatalogPins        atomic.Int64                    // existing native resource lifetime, not persisted
	localCatalogMaintenance atomic.Int64                    // existing native maintenance lifetime, not persisted
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
	// saveMu protects only checkpoint admission, never schema capture or I/O.
	// One owner captures all requests admitted to a batch and persists them
	// before completing that batch. Callers release schema ownership before
	// waiting; concurrent requests retain only one pending batch, not N copies
	// of the complete schema. A batch preserves its own result across retries.
	saveMu      sync.Mutex       `json:"-"`
	savePending *schemaSaveBatch `json:"-"`
	saveDone    chan struct{}    `json:"-"`
	schemaDirty atomic.Bool      `json:"-"`
	blobRefs    *blobRefState    `json:"-"`
	// persistenceLifecycle protects storage generations from rebuild through
	// publication. Rebuild/repartition and short catalog mutations take a read
	// capability; cleanup and backend migration take the exclusive capability.
	// Query and DML paths do not participate.
	persistenceLifecycle sync.RWMutex `json:"-"`
	// transactionLog is the database-wide commit authority for transactional
	// shard WAL entries. Shard logs may contain prepared records after a crash;
	// recovery exposes them only when this log contains their durable commit.
	transactionOnce         sync.Once                         `json:"-"`
	transactionMu           sync.RWMutex                      `json:"-"`
	transactionLogKeyOpened string                            `json:"-"`
	transactionLog          PersistenceLogfile                `json:"-"`
	committedTx             map[string]uint64                 `json:"-"`
	catalogDecisions        map[string]*catalogDecisionRecord `json:"-"`
	catalogProofVersion     atomic.Uint64                     `json:"-"`
	transactionGen          uint64                            `json:"-"`
	// Nontransactional sequence authority shares the database WAL and mutex.
	sequences       map[string]int64 `json:"-"`
	sequenceFailure error            `json:"-"`

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
			_, legacy, handle := db.persistence.ReplayLog(transactionLogName, false)
			for entry := range legacy {
				if err := db.acceptDatabaseWALFrame(entry, committed, 0); err != nil {
					panic(err)
				}
			}
			handle.Close()
		}
		openedKey := db.transactionLogKey()
		_, entries, logfile := db.persistence.ReplayLog(openedKey, false)
		for entry := range entries {
			if err := db.acceptDatabaseWALFrame(entry, committed, 0); err != nil {
				panic(err)
			}
		}
		db.transactionMu.Lock()
		db.committedTx = committed
		db.transactionLog = logfile
		db.transactionLogKeyOpened = openedKey
		db.transactionMu.Unlock()
	})
	// A database can be opened before the native cluster runtime starts. Its
	// pre-runtime handle must never keep writing the shared legacy stream once
	// installations have independent row-commit authority. In particular it may
	// otherwise overwrite a catalog envelope appended by the schema owner.
	db.transactionMu.Lock()
	defer db.transactionMu.Unlock()
	key := db.transactionLogKey()
	if db.transactionLogKeyOpened == key {
		return
	}
	_, entries, logfile := db.persistence.ReplayLog(key, false)
	for entry := range entries {
		if err := db.acceptDatabaseWALFrame(entry, db.committedTx, db.transactionGen); err != nil {
			logfile.Close()
			panic(err)
		}
	}
	old := db.transactionLog
	db.transactionLog = logfile
	db.transactionLogKeyOpened = key
	if old != nil {
		old.Close()
	}
}

func (db *database) transactionCommitted(txID string) bool {
	if txID == "" {
		return true
	}
	db.initializeTransactionLog()
	db.transactionMu.RLock()
	_, committed := db.committedTx[txID]
	var catalogProofs []*catalogDecisionRecord
	if !committed {
		for _, record := range db.catalogDecisions {
			if record.Transaction == txID && record.Incarnation == db.incarnationID() {
				catalogProofs = append(catalogProofs, record)
			}
		}
	}
	db.transactionMu.RUnlock()
	if !committed && len(catalogProofs) != 0 {
		for _, proof := range catalogProofs {
			if readDistributedDecision(db.persistence, proof.authorityID()) {
				return true
			}
		}
		return false
	}
	if !committed {
		if engine := sharedCommitBackend(); engine != nil {
			return readDistributedDecision(engine, txID)
		}
	}
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
		// A checkpoint may require SAFE authority after this decision was
		// recorded with LOGGED durability. An accepted marker also survives
		// an ambiguous prior flush in memory; a SAFE retry must join the
		// actual persistence barrier rather than merely find that marker.
		if durable {
			if err := persistenceCall(func() { db.transactionLog.Flush(true) }); err != nil {
				return markCommitOutcomeUnknown(err)
			}
		}
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
	roots, independent := db.independentRecoveryTopology()
	if !independent {
		return
	}

	db.initializeTransactionLog()
	db.transactionMu.Lock()
	defer db.transactionMu.Unlock()
	if !db.recoveryTopologyStillIndependent(roots) {
		return
	}

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

	// Catalog preparations remain their own native recovery proof. A row-WAL
	// compaction cutoff alone cannot prove the matching schema checkpoint, so
	// preserve these immutable envelopes across this existing log replacement.
	for _, record := range db.catalogDecisions {
		entries = append(entries, LogEntryCommit{txID: record.encoded})
	}
	entries = append(entries, db.sequenceLogEntries()...)

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
	return d.marshalSchema(false)
}

// marshalSchema distinguishes durable declarations from the live schema owned
// by native coherence. Runtime-only relations use the ordinary table catalog.
func (d *database) marshalSchema(live bool) ([]byte, error) {
	type persist struct {
		Name           string                  `json:"name"`
		Incarnation    string                  `json:"incarnation,omitempty"`
		CatalogVersion uint64                  `json:"catalog_version,omitempty"`
		Tables         map[string]*table       `json:"tables"`
		Procedures     []*ProcedureDescription `json:"procedures,omitempty"`
	}
	definitions := make([]*ProcedureDescription, 0)
	if d.procedures != nil {
		for _, stored := range d.procedures.GetAll() {
			if definition := d.visibleProcedure(stored.Name, nil); definition != nil {
				definitions = append(definitions, definition)
			}
		}
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Name < definitions[j].Name })
	// This is a private serialization view, not a concurrently published catalog.
	// ReadMap.Set copies its complete directory for every insertion; rebuilding a
	// filtered ReadMap here makes each schema snapshot quadratic in table count.
	allTables := d.tables.GetAll()
	tables := make(map[string]*table, len(allTables))
	for _, table := range allTables {
		if (live || (!table.Ephemeral && table.committedAt(d.catalogView().version))) && (!sharedClusterDatabase(d.Name) || !table.isEphemeralQueryTable()) {
			tables[table.Name] = table
		}
	}
	return json.Marshal(persist{Name: d.Name, Incarnation: d.incarnationID(), Tables: tables, Procedures: definitions, CatalogVersion: d.catalogView().version})
}

func (d *database) UnmarshalJSON(data []byte) error {
	type persist struct {
		Name           string                                     `json:"name"`
		Incarnation    string                                     `json:"incarnation,omitempty"`
		CatalogVersion uint64                                     `json:"catalog_version,omitempty"`
		Tables         *NonLockingReadMap.ReadMap[string, *table] `json:"tables"`
		Procedures     []*ProcedureDescription                    `json:"procedures,omitempty"`
	}
	var p persist
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	d.Name = p.Name
	if p.Incarnation != "" {
		if _, err := uuid.Parse(p.Incarnation); err != nil {
			return fmt.Errorf("invalid database incarnation")
		}
		d.incarnation.Store(&p.Incarnation)
	}
	d.catalogVersion = p.CatalogVersion
	if d.tables != nil {
		for _, t := range d.tables.GetAll() {
			t.releaseStatementTriggerMemory(d)
		}
	}
	d.tables = p.Tables
	if d.procedures != nil {
		for _, definition := range d.procedures.GetAll() {
			definition.releaseSourceMemory(d)
		}
	}
	d.procedures = NonLockingReadMap.NewReadMap[string, *ProcedureDescription]()
	for _, definition := range p.Procedures {
		if definition == nil || definition.Name == "" || definition.Language == "" || definition.Source == "" || d.procedures.Get(definition.Name) != nil {
			return fmt.Errorf("invalid stored procedure source definition")
		}
		if err := definition.initializeSourceMemory(); err != nil {
			return err
		}
		definition.retainSourceMemory(d)
		d.procedures.Set(definition.Name, definition)
	}
	if d.tables == nil {
		d.tables = NonLockingReadMap.NewReadMap[string, *table]()
	}
	if err := d.validateStoredStatementTriggers(); err != nil {
		return err
	}
	for _, table := range d.tables.GetAll() {
		if err := table.validateSystemCleanupFormats(); err != nil {
			return err
		}
		table.retainStatementTriggerMemory(d)
		table.refreshTriggerMemoryLocked()
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
	if current := d.ReadMap.Get(name); current != nil && current != db &&
		(current.localCatalogPins.Load() != 0 || current.localCatalogMaintenance.Load() != 0) {
		panic("native database has pending declarations or maintenance; retry replacement")
	}
	previous := d.ReadMap.Set(name, db)
	d.publishNames()
	return previous
}

func (d *databaseDirectory) Remove(name string) *database {
	d.namespaceMu.Lock()
	defer d.namespaceMu.Unlock()
	if current := d.ReadMap.Get(name); current != nil && current.localCatalogPins.Load() != 0 {
		panic("native database has pending declarations; retry removal")
	}
	if current := d.ReadMap.Get(name); current != nil && current.localCatalogMaintenance.Load() > 1 {
		panic("native database has concurrent maintenance; retry removal")
	}
	previous := d.ReadMap.Remove(name)
	d.publishNames()
	return previous
}

func (d *databaseDirectory) publishNames() {
	names := make(map[string]string)
	for _, db := range d.GetAll() {
		key := scm.CaseFoldKey(db.Name)
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
	if tbl != nil && tbl.schema != nil && !tbl.committedAt(tbl.schema.catalogView().version) {
		panic("unpublished native table cannot be rebuilt")
	}
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
	requireDatabaseName(dbName)
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
	// Explicit backend configuration owns its physical namespace. Inventory all
	// claims before recovery/publication: otherwise an automatically discovered
	// directory can become a second database with an independent stale schema.
	// Schemas and shards remain lazy; only unpublished backend handles are built.
	entries, err := os.ReadDir(Basepath)
	if err != nil && !os.IsNotExist(err) {
		raisePersistenceFailure("filesystem", Basepath, "database.discover", err)
	}
	configured := make(map[string]bool)
	owners := make(map[string]string)
	selected := make([]*database, 0, len(entries))
	selectDatabase := func(name string, persistence PersistenceEngine) {
		identity := persistence.StorageIdentity()
		if owner, exists := owners[identity]; exists {
			panic(fmt.Sprintf("database storage namespace is claimed by both %s and %s", owner, name))
		}
		owners[identity] = name
		db := newDatabase()
		db.Name = name
		db.persistence = persistence
		db.srState = COLD
		selected = append(selected, db)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || entry.Name() == "settings.json" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		if !validDatabaseName(name) {
			fmt.Println("ignoring invalid database storage component")
			continue
		}
		configPath := Basepath + "/" + entry.Name()
		configData, err := os.ReadFile(configPath)
		if err != nil {
			raisePersistenceFailure("filesystem", name, "database.config.read", err)
		}
		// Other producers may retain ordinary JSON metadata in the root.
		// Only an object declaring a backend claims a database namespace.
		var descriptor struct {
			Backend json.RawMessage `json:"backend"`
		}
		if err := json.Unmarshal(configData, &descriptor); err != nil {
			if _, nonObject := err.(*json.UnmarshalTypeError); nonObject {
				continue
			}
			panic(fmt.Errorf("invalid database storage JSON %s: %w", configPath, err))
		}
		if len(descriptor.Backend) == 0 {
			continue
		}
		configured[name] = true
		persistence := createPersistenceFromConfig(name, json.RawMessage(configData))
		if persistence == nil {
			panic("unknown or invalid database backend configuration: " + configPath)
		}
		selectDatabase(name, persistence)
		fmt.Println("loading database", name, "from backend config", configPath)
	}
	for _, entry := range entries {
		if !entry.IsDir() || configured[entry.Name()] {
			continue
		}
		name := entry.Name()
		if !validDatabaseName(name) {
			fmt.Println("ignoring invalid database storage component")
			continue
		}
		persistence := instrumentPersistence(name, (&FileFactory{Basepath: Basepath}).CreateDatabase(name))
		if owner, exists := owners[persistence.StorageIdentity()]; exists && configured[owner] {
			// This directory is the configured database's actual storage, not
			// another logical database. Never give it a second cleanup owner.
			continue
		}
		selectDatabase(name, persistence)
	}
	for _, db := range selected {
		if !sharedClusterDatabase(db.Name) {
			db.recoverAbandonedBlobWrites()
		}
		databases.Set(db.Name, db)
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

// A batch is mutable only while pending under saveMu. Detaching it fixes the
// capture boundary and durability requirement. failure is published once by
// closing done; a later successful checkpoint cannot change an earlier result.
type schemaSaveBatch struct {
	done    chan struct{}
	durable bool
	failure any
}

func (db *database) commitSchemaSnapshot(durable bool, releaseSchema func()) {
	// Admission happens before releasing the caller's schema owner. Therefore
	// every mutation admitted to this batch precedes its owner's later capture.
	db.saveMu.Lock()
	batch := db.savePending
	if batch == nil {
		batch = &schemaSaveBatch{done: make(chan struct{})}
		db.savePending = batch
	}
	batch.durable = batch.durable || durable
	start := db.saveDone == nil
	if start {
		db.saveDone = make(chan struct{})
	}
	db.saveMu.Unlock()
	releaseSchema()
	if start {
		// The requests waiting on batch.done own this finite task. It exits when
		// no work remains; no goroutine or schema buffer survives an idle queue.
		go db.commitSchemaBatches()
	}
	<-batch.done
	if batch.failure != nil {
		panic(batch.failure)
	}
}

func (db *database) captureSchemaSnapshot() []byte {
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	// Mutable declarations still need this schema read capability during
	// encoding. No queued waiter retains one, and backend I/O happens after it
	// is released. Replacing this final encoding span requires immutable native
	// persistence descriptors, rather than reading live declarations unlocked.
	db.schemaDirty.Store(false)
	snapshot, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		panic(fmt.Errorf("serialize database schema: %w", err))
	}
	return snapshot
}

func (db *database) commitSchemaBatches() {
	for {
		db.saveMu.Lock()
		batch := db.savePending
		db.savePending = nil
		if batch == nil {
			done := db.saveDone
			db.saveDone = nil
			db.saveMu.Unlock()
			close(done)
			return
		}
		db.saveMu.Unlock()

		var failure any
		func() {
			defer func() { failure = recover() }()
			db.writeSchema(db.captureSchemaSnapshot(), batch.durable)
		}()
		if failure == nil {
			close(batch.done)
			continue
		}

		db.schemaDirty.Store(true)
		// All currently admitted callers receive this failed outcome. A fresh
		// request may retry afterwards, but cannot rehabilitate any failed batch.
		db.saveMu.Lock()
		pending, done := db.savePending, db.saveDone
		db.savePending, db.saveDone = nil, nil
		db.saveMu.Unlock()
		batch.failure = failure
		close(batch.done)
		if pending != nil {
			pending.failure = failure
			close(pending.done)
		}
		close(done)
		return
	}
}

func (db *database) save(counterOwner ...*replicaCell) {
	if len(counterOwner) > 1 {
		panic("invalid counter checkpoint ownership")
	}
	if sharedClusterDatabase(db.Name) {
		cluster := activeNodeCluster()
		if cluster == nil {
			return
		}
		cell := cluster.manager.resourceCell(mosiResource{Database: db.Name})
		phase := cell.ownstate.Phase()
		if len(counterOwner) != 0 {
			// The allocator holds this existing directory mutex through the
			// checkpoint. O owns the unchanged declarations and may persist its
			// runtime counter without recalling every S reader. A local M
			// request can already have published MR while waiting for this
			// directory boundary; that request does not transfer ownership.
			// Validate the serialized directory owner, not request admission.
			// Ordinary saves still require M; this cannot authorize DDL.
			cluster.manager.mu.Lock()
			owner := cluster.manager.directory.Records[cluster.manager.resourceKey(mosiResource{Database: db.Name})].Owner
			cluster.manager.mu.Unlock()
			if counterOwner[0] != cell || owner != cluster.self.ID {
				panic("counter checkpoint lost its owner")
			}
		} else if phase != mosi.M {
			return
		}
	}
	if db.srState == COLD {
		// Do not serialize a cold database; keep existing schema.json intact
		return
	}
	db.schemalock.RLock()
	db.commitSchemaSnapshot(true, db.schemalock.RUnlock)
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
	db.commitSchemaSnapshot(mode == schemaSaveFsync, db.schemalock.Unlock)
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
			db.tables = NonLockingReadMap.NewReadMap[string, *table]()
		} else {
			tmp := new(database)
			if err := json.Unmarshal(jsonbytes, tmp); err != nil {
				panic(err)
			}
			db.tables, db.catalogVersion, db.procedures = tmp.tables, tmp.catalogVersion, tmp.procedures
			for _, t := range db.tables.GetAll() {
				t.moveStatementTriggerMemoryOwner(db)
			}
			for _, definition := range db.procedures.GetAll() {
				definition.moveSourceMemoryOwner(db)
			}
			db.incarnation.Store(tmp.incarnation.Load())
		}
		db.recoverCatalogDecisions()
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
		rebuildCheckTriggersAfterLoad(db)
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
	pending := make(map[string]map[string]string)
	for _, table := range db.tables.GetAll() {
		if table.isHiddenFromShowTables() {
			continue
		}
		target := names
		if !table.committedAt(db.catalogVersion) {
			target = pending[table.CatalogDecisionID]
			if target == nil {
				target = make(map[string]string)
				pending[table.CatalogDecisionID] = target
			}
		}
		key := scm.CaseFoldKey(table.Name)
		if _, exists := target[key]; exists {
			target[key] = ""
		} else {
			target[key] = table.Name
		}
	}
	db.catalog.Store(&catalogSnapshot{version: db.catalogVersion, folded: names, pending: pending})
}

// invalidatePersistedPlannerCodeAfterLoad drops executable code derived from a
// physical query plan. Planner-owned CACHE tables contain no durable rows, and
// the current createtable guard supplies their authoritative oninit callback on
// first use. Retaining an older callback across a physical operator ABI change
// would execute stale generated code before that guard can refresh it.
// upgradePersistedTableCalls runs only while loading unpublished executable
// definitions. It changes old native table/key-index call ABIs, never SQL syntax or
// row values. Quoted data stays opaque; the live builtin accepts only the new
// interface. New Proc identities discard any obsolete compiled/optimizer state.
func upgradePersistedTableCalls(value scm.Scmer) scm.Scmer {
	value = upgradePersistedBuiltinCalls(value, "table", func(items []scm.Scmer) (scm.Scmer, bool) {
		switch len(items) {
		case 3:
			return scm.NewSlice([]scm.Scmer{items[0], items[1], scm.NewBool(false), items[2], scm.NewBool(false), scm.NewNil()}), true
		case 5:
			return scm.NewSlice(append(append([]scm.Scmer(nil), items...), scm.NewNil())), true
		}
		return scm.Scmer{}, false
	})
	return upgradePersistedBuiltinCalls(value, "recset_key_index", func(items []scm.Scmer) (scm.Scmer, bool) {
		if len(items) == 4 {
			return scm.NewSlice(append(append([]scm.Scmer(nil), items...), scm.NewString("join"))), true
		}
		return scm.Scmer{}, false
	})
}

// Every migration uses the same lexical-binding and quoted-data boundary. An
// old native call receives the declared canonical ABI exactly once at decode;
// a user binding with the same spelling retains its original meaning.
func upgradePersistedBuiltinCalls(value scm.Scmer, builtinName string, upgrade func([]scm.Scmer) (scm.Scmer, bool)) scm.Scmer {
	bindsBuiltin := func(params scm.Scmer) bool {
		params = params.WithoutSourceInfo()
		if callHeadIs(params, builtinName) {
			return true
		}
		if params.IsSlice() {
			for _, param := range params.Slice() {
				if callHeadIs(param, builtinName) {
					return true
				}
			}
		}
		return false
	}
	var patternBindsBuiltin func(scm.Scmer) bool
	patternBindsBuiltin = func(pattern scm.Scmer) bool {
		pattern = pattern.WithoutSourceInfo()
		if callHeadIs(pattern, builtinName) {
			return true
		}
		if pattern.IsSlice() {
			parts := pattern.Slice()
			if len(parts) > 0 && (callHeadIs(parts[0], "quote") || callHeadIs(parts[0], "symbol") || callHeadIs(parts[0], "ignorecase") || callHeadIs(parts[0], "eval")) {
				return false // literal/reference patterns do not introduce a binding
			}
			for _, part := range parts {
				if patternBindsBuiltin(part) {
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
			if bindsBuiltin(proc.Params) {
				return value, scope
			}
			if proc.En != nil {
				if owner := proc.En.FindRead(scm.Symbol(builtinName)); owner != nil && owner.Vars[scm.Symbol(builtinName)] != scm.Globalenv.Vars[scm.Symbol(builtinName)] {
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
			if !bindsBuiltin(items[1]) {
				body, _ := restore(items[2], scope)
				replace(2, body)
			}
		case len(items) == 3 && (callHeadIs(items[0], "define") || callHeadIs(items[0], "set")):
			declaresBuiltin := callHeadIs(items[1], builtinName)
			initializer := items[2].WithoutSourceInfo()
			recursive := initializer.IsProc() || (initializer.IsSlice() && len(initializer.Slice()) > 0 && callHeadIs(initializer.Slice()[0], "lambda"))
			initialScope := scope
			if declaresBuiltin && recursive {
				initialScope = local
			}
			rewritten, next := restore(items[2], initialScope)
			replace(2, rewritten)
			bound = next
			if declaresBuiltin {
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
					if patternBindsBuiltin(items[i]) {
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
		if callHeadIs(items[0], builtinName) {
			candidate := items
			if replacement != nil {
				candidate = replacement
			}
			if rewritten, changed := upgrade(candidate); changed {
				if scope == conditional {
					panic("cannot upgrade a native call after conditional rebinding; update the stored definition explicitly")
				}
				return rewritten, bound
			}
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
		trigger.initialFKFootprint = nil
		if trigger.Timing.isStatement() {
			continue // a new source recipe has no predecessor executable ABI
		}
		if t.regenerateORCTriggerAfterLoad(trigger) {
			continue
		}
		trigger.Func = upgradePersistedTableCalls(trigger.Func)
		if trigger.Language == "scheme" && trigger.Source != "" {
			definition := scm.Read("stored trigger", trigger.Source, true)
			if restored := upgradePersistedTableCalls(definition); restored != definition {
				trigger.Source = scm.SerializeToString(restored, &scm.Globalenv)
			}
		}
	}
	// Loading has exclusive ownership before table publication; optional finite
	// FK descriptors are regenerated later by the authoritative FK loader.
	t.refreshTriggerMemoryLocked()
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
	for {
		adapter := effectiveSchemaReplication(db)
		if adapter != nil {
			_, release, _ = adapter.writeResource(mosiResource{Database: db.Name}, tx, true)
		}
		db.catalogPublicationMu.Lock()
		nativeRelease := release
		release = func() {
			db.catalogPublicationMu.Unlock()
			if nativeRelease != nil {
				nativeRelease()
			}
		}
		// Mode publishers take this native catalog owner before changing the
		// effective adapter. Retry its acquisition only after releasing BOTH
		// local publication and the previous MOSI capability, outside all locks.
		// A configured-but-starting adapter panics through the owned defer.
		if effectiveSchemaReplication(db) != adapter {
			oldRelease := release
			release = nil
			oldRelease()
			continue
		}
		db.ensureLoaded()
		ready = true
		return WriteAccess{release: release}
	}
}

// helper to fetch a table with lazy db load
func (db *database) GetTable(name string, ignoreCase ...bool) *table {
	return db.GetTableForTx(name, len(ignoreCase) != 0 && ignoreCase[0], nil)
}
func (db *database) getLegacyTempTable(name string, tx *TxContext, version uint64) *table {
	/* Query-temp/keytable names historically used unqualified source aliases in
	the embedded get_column serialization. Newer planner/debug paths may look
	them up with schema-qualified aliases; accept both spellings so planner IR
	can evolve without breaking the physical temp-table namespace. */
	if strings.HasPrefix(name, ".") {
		normalizedName := normalizeTempLookupName(db.Name, name)
		if normalizedName != name {
			if t := db.tables.Get(normalizedName); t != nil && t.visibleIn(tx, version) {
				return t
			}
		}
		for _, t := range db.tables.GetAll() {
			if t.visibleIn(tx, version) && normalizeTempLookupName(db.Name, t.Name) == normalizedName {
				return t
			}
		}
	}
	return nil
}

func (db *database) ShowTables(transaction ...*TxContext) scm.Scmer {
	var tx *TxContext
	if len(transaction) > 0 {
		tx = transaction[0]
	}
	db.ensureLoaded()
	view := db.catalogView()
	tables := db.tables.GetAll()
	result := make([]scm.Scmer, 0, len(tables))
	for _, t := range tables {
		if !t.visibleIn(tx, view.version) || t.isHiddenFromShowTables() {
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
			if !t.committedAt(db.catalogView().version) {
				incomplete.Store(true)
				return
			}
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
						rebuilt := s.rebuild(all)
						s.flushRebuildSuccessor(rebuilt)
						newShardList[i] = rebuilt
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
				count := uint(shard.main_count) + uint(len(shard.deltaRows())) - uint(shard.deletions.Count())
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

			// Completed Source -> Next forwarding keeps both generations current.
			// The builder's recovery WAL was flushed before reaching this point;
			// later mutations and decisions carry their own successor WAL barriers.
			// Keep the old topology authoritative until its successor directory is
			// durable, without retaining row-writer locks during capture or I/O.
			// ddlMu and maintenanceMu already own this definition/rebuild lifetime.
			durablePublication := len(replaced) > 0 && t.PersistencyMode != Memory && t.PersistencyMode != Cache

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
				// Retain every ready Next and its WAL. A failed schema fsync may
				// leave either directory on disk; ordinary Source writes continue
				// forwarding, and a later rebuild can retry the same successor.
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
			t.refreshShowColumnsStatistics()
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
		canonical, exists := (*names)[scm.CaseFoldKey(schema)]
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
	requireDatabaseName(schema)
	db := databases.Get(schema)
	if db != nil {
		if ignoreexists {
			return false
		}
		panic("Database " + schema + " already exists")
	}

	db = newDatabase()
	incarnation := uuid.NewString()
	db.incarnation.Store(&incarnation)
	db.Name = schema
	persistence := FileFactory{Basepath} // TODO: remove this, use parameter instead
	db.persistence = instrumentPersistence(schema, persistence.CreateDatabase(schema))
	if len(db.persistence.ReadSchema()) != 0 {
		panic("destination storage already contains a database schema")
	}
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
	requireDatabaseName(schema)
	backend := options["backend"]
	if len(options) == 0 || (len(options) == 1 && backend == "filesystem") {
		return CreateDatabase(schema, ignoreexists)
	}
	if backend == "" {
		backend = "filesystem"
		options["backend"] = backend
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
	if _, ok := options["prefix"]; !ok && backend != "filesystem" {
		options["prefix"] = schema
	}

	raw := marshalDatabaseBackendOptions(options)

	// Validate the native backend namespace before publishing its config.
	persistence := createPersistenceFromConfig(schema, json.RawMessage(raw))
	if persistence == nil {
		panic("failed to create persistence engine for backend: " + backend)
	}
	if len(persistence.ReadSchema()) != 0 {
		panic("destination storage already contains a database schema")
	}
	writeDatabaseBackendConfig(schema, raw)

	db = newDatabase()
	incarnation := uuid.NewString()
	db.incarnation.Store(&incarnation)
	db.Name = schema
	db.persistence = persistence
	db.tables = NonLockingReadMap.NewReadMap[string, *table]()
	db.srState = WRITE

	last := databases.Set(schema, db)
	if last != nil {
		databases.Set(schema, last)
		removeDatabaseBackendConfig(schema)
		panic("Database " + schema + " already exists")
	}

	db.save()
	return true
}

// Preserve the existing string-valued backend option contract, while rejecting
// non-string roots before the Scheme boundary could coerce them into paths.
func databaseBackendOptions(items []scm.Scmer) map[string]string {
	if len(items)%2 != 0 {
		panic("database backend options must contain key/value pairs")
	}
	options := make(map[string]string, len(items)/2)
	for index := 0; index < len(items); index += 2 {
		key := scm.String(items[index])
		if key == "root" && !items[index+1].IsString() {
			panic("filesystem root must be a string")
		}
		options[key] = scm.String(items[index+1])
	}
	return options
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
	configPath := databaseBackendConfigPath(schema)
	if configPath == "" {
		return json.RawMessage(`{"backend":"filesystem"}`)
	}
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
	configPath := databaseBackendConfigPath(schema)
	if configPath == "" {
		if defaultFilesystemBackendConfig(raw) {
			return
		}
		panic("database backend configuration conflicts with reserved server settings")
	}
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
	if err := os.Rename(tmpName, configPath); err != nil {
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
	requireDatabaseName(schema)
	requireDatabaseName(sourceDB)
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
	// Validate the destination backend before publishing its cloned config.
	persistence := createPersistenceFromConfig(schema, json.RawMessage(raw))
	if persistence == nil {
		panic("failed to create persistence engine from source config")
	}
	if len(persistence.ReadSchema()) != 0 {
		panic("destination storage already contains a database schema")
	}
	writeDatabaseBackendConfig(schema, raw)

	db = newDatabase()
	incarnation := uuid.NewString()
	db.incarnation.Store(&incarnation)
	db.Name = schema
	db.persistence = persistence
	db.tables = NonLockingReadMap.NewReadMap[string, *table]()
	db.srState = WRITE

	last := databases.Set(schema, db)
	if last != nil {
		databases.Set(schema, last)
		removeDatabaseBackendConfig(schema)
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
			shard.flushRebuildSuccessor(generation.newShards[index])
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
	_, input, logfile := engine.ReplayLog(name, false)
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
				entries := replayPersistenceLog(src, shard.uuid.String())
				for _, entry := range entries {
					shard.noteRecoveryEntry(entry)
				}
				shard.logfile = dst.SwapLog(shard.uuid.String(), entries, shard.t.PersistencyMode == Safe)
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
	for _, record := range db.catalogDecisions {
		entries = append(entries, LogEntryCommit{txID: record.encoded})
	}
	entries = append(entries, db.sequenceLogEntries()...)
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
	requireDatabaseName(schema)
	db := GetDatabase(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}
	requireDatabaseMaintenance(schema, maintenanceAlter)
	if SessionStateFromTx(currentTx) == nil {
		panic("ALTER DATABASE storage requires a query session")
	}

	db.ensureLoaded()
	// Acquire the existing operation lifetime before destination construction
	// or blob-catalog initialization can have effects. Its scalar database
	// admission also excludes a local FK constructor before its first endpoint.
	defer db.admitNativeMaintenance()()
	dst := createPersistenceFromConfig(schema, targetConfig)
	if dst == nil {
		panic("unknown or invalid storage backend")
	}
	// The blob catalog must exist before the table set is frozen. Rebuild may
	// update it, and no table may appear while the destination generation is
	// being prepared.
	db.ensureBlobTable()
	defer db.GetWrite(nil).Release()
	db.persistenceLifecycle.Lock()
	defer db.persistenceLifecycle.Unlock()
	if len(registeredClusterShards(schema)) != 0 {
		panic("database has active cluster object references")
	}
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
	requireDatabaseName(schema)
	db := databases.Get(schema)
	if db == nil {
		if ifexists {
			return false
		}
		panic("Database " + schema + " does not exist")
	}
	// Database removal shares the same native publication capability as DDL.
	// A decided catalog generation cannot lose its journal or predecessor files
	// while COMMIT still owns that capability.
	schemaAccess := db.GetWrite(nil)
	defer schemaAccess.Release()
	if databases.Get(schema) != db {
		panic("database was replaced during removal")
	}
	db.persistenceLifecycle.Lock()
	defer db.persistenceLifecycle.Unlock()
	defer db.admitNativeMaintenance()()
	if len(registeredClusterShards(schema)) != 0 {
		panic("database has active cluster object references")
	}
	if r := replication.Load(); r != nil && r.hasTransfers(schema) {
		panic("database has active replication transfers; retry DROP DATABASE")
	}
	requireDatabaseMaintenance(schema, maintenanceDrop)
	func() {
		db.schemalock.Lock()
		defer db.schemalock.Unlock()
		for _, table := range db.tables.GetAll() {
			table.requireUnleasedTriggerDeclarations()
		}
		db = databases.Remove(schema)
	}()
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
	removeDatabaseBackendConfig(schema)
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
	dropTableIdentity(schema, name, ifexists, nil)
}

// An owned relation can only retire its exact native identity. Reusing its name
// after explicit DROP must not grant a stale scope authority over a replacement.
func dropTableIdentity(schema, name string, ifexists bool, expected *table) {
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
	if expected != nil && (t == nil || !t.Ephemeral || t.RuntimeIdentity != expected.RuntimeIdentity) {
		db.schemalock.Unlock()
		return
	}
	if t == nil {
		db.schemalock.Unlock()
		if ifexists {
			return // silentfail
		}
		panic("Table " + schema + "." + name + " does not exist")
	}
	// The actual scoped-generation owner may retire its unpublished MEMORY
	// result after row Undo. Public/name-only DROP retains the existing catalog
	// protection; the identity check above never grants authority to a replacement.
	ownedScopedRetirement := expected != nil && expected.ScopedSchema
	if !ownedScopedRetirement && !tableMaintenanceCapabilities(schema, name).canDrop {
		db.schemalock.Unlock()
		requireTableMaintenance(schema, name, maintenanceDrop)
	}
	releaseAdmission := func() {}
	releaseForeign := func() {}
	dropSaveMode := t.schemaSaveMode()
	func() {
		defer func() {
			if failure := recover(); failure != nil {
				releaseForeign()
				releaseAdmission()
				db.schemalock.Unlock()
				panic(failure)
			}
		}()
		if t.mutationReadOnly && expected == t {
			// The existing relationScope presents the actual identity, not a
			// public DROP-by-name request. It alone may retire result relations.
			lease, err := t.writeAdmission.admit(context.Background(), nil, false, false)
			if err != nil {
				panic(err)
			}
			releaseAdmission = func() { _ = lease.Close() }
		} else {
			releaseAdmission = t.admitNativeMaintenance()
		}
		t.requireUnleasedTriggerDeclarations()
		releaseForeign, dropSaveMode = t.retireDroppedForeignDeclarationsLocked()
	}()
	defer func() {
		releaseForeign()
		releaseAdmission()
	}()
	db.tables.Remove(name)
	t.mu.Lock()
	t.releaseStatementTriggerMemory(db)
	t.mu.Unlock()
	if r := replication.Load(); r != nil && validReplicaDatabase(schema) {
		r.retireTablePayload(t)
	}
	if !t.isHiddenFromShowTables() {
		db.publishTableNamesLocked()
	}
	if name == ".blobs" {
		db.blobRefState().table.Store(nil)
	}
	db.saveLockedAndUnlock(dropSaveMode)
	releaseForeign()
	releaseForeign = func() {}
	releaseAdmission()
	releaseAdmission = func() {}

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
	var releaseAdmission func()
	func() {
		defer func() {
			if failure := recover(); failure != nil {
				db.schemalock.Unlock()
				panic(failure)
			}
		}()
		releaseAdmission = t.admitNativeMaintenance()
	}()
	defer releaseAdmission()
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
			source.refreshTriggerMemoryLocked()
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
	frames := make([]interface{}, 0)
	catalogVersion := db.catalogView().version
	if db.catalogProofVersion.Load() < catalogVersion {
		_, catalogEntries, catalogHandle := db.persistence.ReplayLog(transactionLogName, false)
		for entry := range catalogEntries {
			frames = append(frames, entry)
		}
		catalogHandle.Close()
	}

	for owner := range owners {
		if err := validateClusterURI(owner); err != nil {
			return err
		}
		id := strings.TrimPrefix(owner, "memcp://")
		_, entries, handle := db.persistence.ReplayLog(transactionLogName+"."+id, false)
		for entry := range entries {
			frames = append(frames, entry)
		}
		handle.Close()
	}
	db.initializeTransactionLog()
	db.transactionMu.Lock()
	for _, frame := range frames {
		if err := db.acceptDatabaseWALFrame(frame, db.committedTx, db.transactionGen); err != nil {
			db.transactionMu.Unlock()
			return err
		}
	}
	db.transactionMu.Unlock()
	db.catalogProofVersion.Store(catalogVersion)
	return nil
}
