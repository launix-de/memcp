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
import "sync"
import "sync/atomic"
import "time"
import "runtime"
import "strings"
import "encoding/json"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/NonLockingReadMap"

type database struct {
	Name        string                                             `json:"name"`
	persistence PersistenceEngine                                  `json:"-"`
	tables      NonLockingReadMap.NonLockingReadMap[table, string] `json:"-"`
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
	saveMu          sync.Mutex `json:"-"`
	saveCondOnce    sync.Once  `json:"-"`
	saveCond        *sync.Cond `json:"-"`
	saveRequested   uint64     `json:"-"`
	saveCompleted   uint64     `json:"-"`
	saveInFlight    bool       `json:"-"`
	savePending     []byte     `json:"-"`
	savePendingSync bool       `json:"-"`
	savePanic       any        `json:"-"`
	blobMu          sync.Mutex `json:"-"` // serializes IncrBlobRefcount/DecrBlobRefcount

	// lazy-loading/shared-resource state (not serialized)
	srState SharedState `json:"-"`
}

type rebuildDatabaseResult struct {
	replaced []*storageShard
	errors   []string
}

type schemaWriteOptions interface {
	WriteSchemaWithMode(schema []byte, durable bool)
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
		Name   string                                             `json:"name"`
		Tables NonLockingReadMap.NonLockingReadMap[table, string] `json:"tables"`
	}
	return json.Marshal(persist{Name: d.Name, Tables: d.tables})
}

func (d *database) UnmarshalJSON(data []byte) error {
	type persist struct {
		Name   string                                             `json:"name"`
		Tables NonLockingReadMap.NonLockingReadMap[table, string] `json:"tables"`
	}
	var p persist
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	d.Name = p.Name
	d.tables = p.Tables
	return nil
}

// TODO: replace databases map everytime something changes, so we don't run into read-while-write
// e.g. a table of databases
var databases NonLockingReadMap.NonLockingReadMap[database, string] = NonLockingReadMap.New[database, string]()
var Basepath string = "data"

/* implement NonLockingReadMap */
func (d database) GetKey() string {
	return d.Name
}

func (d database) ComputeSize() uint {
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

func rebuildDatabases(all bool, repartition bool, includeEphemeral bool) string {
	start := time.Now()
	dbs := databases.GetAll()
	var errs []string
	for _, db := range dbs {
		func(db *database) {
			result := db.rebuild(all, repartition, includeEphemeral)
			if len(result.errors) > 0 {
				errs = append(errs, result.errors...)
				return
			}
			if err := func() (err error) {
				defer func() { err = recoverAsError("save failed for database " + db.Name) }()
				db.save()
				return nil
			}(); err != nil {
				errs = append(errs, err.Error())
				return
			}
			for _, s := range result.replaced {
				if err := func() (err error) {
					defer func() { err = recoverAsError("cleanup failed for database " + db.Name + " shard " + s.uuid.String()) }()
					s.RemoveFromDisk()
					return nil
				}(); err != nil {
					errs = append(errs, err.Error())
				}
			}
		}(db)
	}
	duration := fmt.Sprint(time.Since(start))
	if len(errs) == 0 {
		return duration
	}
	return duration + " errors: " + strings.Join(errs, " | ")
}

func UnloadDatabases() {
	fmt.Println("table compression done in ", rebuildDatabases(false, false, true))
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
	return factory(dbName, raw)
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
	for _, entry := range entries {
		if entry.IsDir() {
			db := new(database)
			db.Name = entry.Name()
			db.persistence = &FileStorage{path: Basepath + "/" + entry.Name() + "/"}
			db.srState = COLD
			databases.Set(db)
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
			db := new(database)
			db.Name = dbName
			db.persistence = persistence
			db.srState = COLD
			databases.Set(db)
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
	db.schemalock.RUnlock()
	db.commitSchemaSnapshot(jsonbytes, true)
	// shards are written while rebuild
}

// saveLockedAndUnlock snapshots the schema while schemalock is held, then
// releases schemalock before performing the synchronous schema write.
// Callers must already hold db.schemalock.Lock().
func (db *database) saveLockedAndUnlock() {
	db.saveLockedWithDurabilityAndUnlock(true)
}

func (db *database) saveLockedWithDurabilityAndUnlock(durable bool) {
	if db.srState == COLD {
		db.schemalock.Unlock()
		return
	}
	jsonbytes, _ := json.MarshalIndent(db, "", "  ")
	db.schemalock.Unlock()
	db.commitSchemaSnapshot(jsonbytes, durable)
}

// ensureLoaded loads schema.json into the database struct exactly once.
func (db *database) ensureLoaded() {
	if db.srState != COLD {
		return
	}
	jsonbytes := db.persistence.ReadSchema()
	if len(jsonbytes) == 0 {
		// fresh/empty database
		db.tables = NonLockingReadMap.New[table, string]()
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
	}
	// FK enforcement triggers are serializable Procs and persist with the table JSON.
	// No re-installation needed on load.
	db.srState = SHARED
}

func triggerTimingSQL(timing TriggerTiming) string {
	switch timing {
	case BeforeInsert:
		return "BEFORE INSERT"
	case AfterInsert:
		return "AFTER INSERT"
	case BeforeUpdate:
		return "BEFORE UPDATE"
	case AfterUpdate:
		return "AFTER UPDATE"
	case BeforeDelete:
		return "BEFORE DELETE"
	case AfterDelete:
		return "AFTER DELETE"
	case AfterDropTable:
		return "AFTER DROP TABLE"
	case AfterDropColumn:
		return "AFTER DROP COLUMN"
	case AfterInvalidate:
		return "AFTER INVALIDATE"
	default:
		panic("unknown trigger timing")
	}
}

func loadPersistedTriggerPlan(schemaName, tableName string, trigger *TriggerDescription) {
	if !triggerScmerMissing(trigger.Func) || !triggerScmerMissing(trigger.FuncPlan) || trigger.SourceSQL == "" {
		return
	}
	parseSQL := scm.Globalenv.Vars[scm.Symbol("parse_sql")]
	allow := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	sql := fmt.Sprintf("CREATE TRIGGER `%s` %s ON `%s` FOR EACH ROW %s", trigger.Name, triggerTimingSQL(trigger.Timing), tableName, trigger.SourceSQL)
	plan := scm.Apply(parseSQL, scm.NewString(schemaName), scm.NewString(sql), allow)
	if !plan.IsSlice() {
		panic("persisted trigger parse did not return an AST")
	}
	items := plan.Slice()
	if len(items) < 7 {
		panic("persisted trigger AST is incomplete")
	}
	trigger.Func, trigger.FuncPlan = unwrapDeferredTriggerBody(items[6])
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

func (db *database) rebuild(all bool, repartition bool, includeEphemeral bool) rebuildDatabaseResult {
	if db.srState == COLD {
		// do nothing for cold databases; avoid loading during rebuild
		return rebuildDatabaseResult{}
	}
	var done sync.WaitGroup
	// Collect pre-rebuild shards that were superseded. Their cleanup
	// (RemoveFromDisk → ReleaseBlobs → DecrBlobRefcount) must run after
	// ALL table rebuilds finish to avoid deadlocks with concurrent .blobs
	// repartition holding shard read-locks.
	var replacedMu sync.Mutex
	var allReplaced []*storageShard
	var errMu sync.Mutex
	var rebuildErrors []string
	dbs := db.tables.GetAll()
	done.Add(len(dbs))
	for _, t := range dbs {
		go func(t *table) {
			tableLocked := false
			rebuildClaimed := false
			t.ddlMu.RLock()
			defer func() {
				t.ddlMu.RUnlock()
				if r := recover(); r != nil {
					errmsg := fmt.Sprintf("rebuild failed for table %s.%s: %v", db.Name, t.Name, r)
					fmt.Println("error:", errmsg)
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
			origShardList := append([]*storageShard(nil), t.ActiveShards()...)
			t.mu.Unlock()
			tableLocked = false
			newShardList := make([]*storageShard, len(origShardList))
			// Track if any shard is still COLD to avoid triggering repartition logic
			hasColdShard := false
			// Count items using shard read locks to avoid races
			getShardCount := func(s *storageShard) uint {
				if s == nil {
					return 0
				}
				s.mu.RLock()
				c := uint(s.main_count) + uint(len(s.inserts)) - uint(s.deletions.Count())
				s.mu.RUnlock()
				return c
			}
			maincount := uint(0)
			var sdone sync.WaitGroup
			var shardErrMu sync.Mutex
			var shardErrors []string
			sdone.Add(len(origShardList))
			// throttle concurrent shard rebuilds by CPU count
			workers := runtime.NumCPU()
			if workers < 1 {
				workers = 1
			}
			type job struct {
				i int
				s *storageShard
			}
			if len(origShardList) <= workers {
				for i, s := range origShardList {
					if s != nil && s.GetState() == COLD {
						hasColdShard = true
					}
					maincount += getShardCount(s)
					go func(i int, s *storageShard) {
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
					}(i, s)
				}
			} else {
				jobs := make(chan job, workers)
				// launch workers
				for w := 0; w < workers; w++ {
					go func() {
						for j := range jobs {
							func(j job) {
								defer func() {
									if r := recover(); r != nil {
										errmsg := fmt.Sprintf("shard rebuild failed for %s.%s shard %d: %v", db.Name, t.Name, j.i, r)
										fmt.Println("error:", errmsg)
										shardErrMu.Lock()
										shardErrors = append(shardErrors, errmsg)
										shardErrMu.Unlock()
									}
									sdone.Done()
								}()
								if j.s != nil {
									newShardList[j.i] = j.s.rebuild(all)
								}
							}(j)
						}
					}()
				}
				for i, s := range origShardList {
					if s != nil && s.GetState() == COLD {
						hasColdShard = true
					}
					maincount += getShardCount(s)
					jobs <- job{i: i, s: s}
				}
				close(jobs)
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

			t.mu.Lock()
			tableLocked = true

			// Collect pre-rebuild shards that were replaced so we can clean them up.
			var replaced []*storageShard
			// Publish the new shard list only after completion.
			if targetIsP {
				for i, oldShard := range origShardList {
					if oldShard != nil && oldShard != newShardList[i] && oldShard.uuid != newShardList[i].uuid {
						replaced = append(replaced, oldShard)
					}
				}
				t.PShards = newShardList
			} else {
				for i, oldShard := range origShardList {
					if oldShard != nil && oldShard != newShardList[i] && oldShard.uuid != newShardList[i].uuid {
						replaced = append(replaced, oldShard)
					}
				}
				t.Shards = newShardList
			}

			// Collect replaced shards for deferred cleanup (see comment above).
			if len(replaced) > 0 {
				replacedMu.Lock()
				allReplaced = append(allReplaced, replaced...)
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

			if doRepart {
				// maintenanceMu stays locked; repartition Phase G will unlock it
				rebuildClaimed = false
				t.repartitionDDLReadLocked(shardCandidates)
			} else {
				// No repartition — release maintenanceMu now
				t.maintenanceMu.Unlock()
				rebuildClaimed = false
			}
		}(t)
	}
	done.Wait()

	// Return replaced shards to the caller (Rebuild) so it can delete their
	// on-disk files AFTER db.save() has written the new schema.json.
	// Deleting old column files before saving the schema creates a window
	// where a crash/kill leaves schema.json pointing at already-deleted UUIDs.
	// The finalizer set in shard.rebuild() provides a safety net: if the
	// caller never calls RemoveFromDisk (e.g. on panic), GC will clean up.
	return rebuildDatabaseResult{replaced: allReplaced, errors: rebuildErrors}
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

	db = new(database)
	db.Name = schema
	persistence := FileFactory{Basepath} // TODO: remove this, use parameter instead
	db.persistence = persistence.CreateDatabase(schema)
	db.tables = NonLockingReadMap.New[table, string]()
	// Newly created database is live for writes
	db.srState = WRITE

	last := databases.Set(db)
	if last != nil {
		// two concurrent CREATE
		databases.Set(last)
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

	// Default prefix to schema name
	if _, ok := options["prefix"]; !ok {
		options["prefix"] = schema
	}

	// Convert force_path_style string to bool for JSON
	forcePathStyle := false
	if fps, ok := options["force_path_style"]; ok {
		forcePathStyle = fps == "true" || fps == "1" || fps == "TRUE"
		delete(options, "force_path_style")
	}

	// Build JSON config
	configMap := make(map[string]interface{})
	for k, v := range options {
		configMap[k] = v
	}
	if forcePathStyle {
		configMap["force_path_style"] = true
	}

	raw, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		panic("failed to marshal backend config: " + err.Error())
	}

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

	db = new(database)
	db.Name = schema
	db.persistence = persistence
	db.tables = NonLockingReadMap.New[table, string]()
	db.srState = WRITE

	last := databases.Set(db)
	if last != nil {
		databases.Set(last)
		os.Remove(configPath)
		panic("Database " + schema + " already exists")
	}

	db.save()
	return true
}

func CreateDatabaseFrom(schema string, ignoreexists bool, sourceDB string) bool {
	db := databases.Get(schema)
	if db != nil {
		if ignoreexists {
			return false
		}
		panic("Database " + schema + " already exists")
	}

	// Read source config
	configPath := Basepath + "/" + sourceDB + ".json"
	configData, err := os.ReadFile(configPath)
	if err != nil {
		panic("Source database " + sourceDB + " has no backend config (filesystem databases cannot be copied)")
	}

	// Parse and update prefix
	var configMap map[string]interface{}
	if err := json.Unmarshal(configData, &configMap); err != nil {
		panic("failed to parse source config: " + err.Error())
	}
	configMap["prefix"] = schema

	raw, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		panic("failed to marshal backend config: " + err.Error())
	}

	// Write new config file
	newConfigPath := Basepath + "/" + schema + ".json"
	if err := os.WriteFile(newConfigPath, raw, 0640); err != nil {
		panic("failed to write backend config: " + err.Error())
	}

	// Create persistence engine
	persistence := createPersistenceFromConfig(schema, json.RawMessage(raw))
	if persistence == nil {
		os.Remove(newConfigPath)
		panic("failed to create persistence engine from source config")
	}

	db = new(database)
	db.Name = schema
	db.persistence = persistence
	db.tables = NonLockingReadMap.New[table, string]()
	db.srState = WRITE

	last := databases.Set(db)
	if last != nil {
		databases.Set(last)
		os.Remove(newConfigPath)
		panic("Database " + schema + " already exists")
	}

	db.save()
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
	db := databases.Remove(schema)
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
	db.ensureLoaded()
	db.schemalock.Lock()
	t, created := db.createTableLocked(name, pm, ifnotexists)
	if !created {
		db.schemalock.Unlock()
		return t, false
	}
	db.saveLockedWithDurabilityAndUnlock(t.PersistencyMode == Safe)
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
	t = new(table)
	t.schema = db
	t.Name = name
	t.PersistencyMode = pm
	t.ShardMode = ShardModeFree
	t.lastAccessed = uint64(time.Now().UnixNano())
	t.Shards = make([]*storageShard, 1)
	t.Shards[0] = NewShard(t)
	t.Auto_increment = 1
	if existing := db.tables.Set(t); existing != nil {
		panic("Table " + name + " already exists")
	}
	return t, true
}

func registerCreatedTable(t *table) {
	// register temp keytable with CacheManager AFTER releasing schemalock
	// to avoid deadlock: AddItem → run() → evict → keytableCleanup → TryLock(schemalock)
	if strings.HasPrefix(t.Name, ".") {
		schemaName := t.schema.Name
		GlobalCache.AddItem(t, int64(t.ComputeSize()), TypeTempKeytable, func(ptr any, freedByType *[numEvictableTypes]int64) bool {
			return keytableCleanup(ptr.(*table), schemaName, freedByType)
		}, keytableLastUsed, nil)
	} else if t.PersistencyMode == Cache {
		// Register the initial shard so eviction can reach it before the first rebuild.
		GlobalCache.AddItem(t.Shards[0], int64(t.Shards[0].ComputeSize()), TypeCacheEntry, cacheShardCleanup, shardLastUsed, nil)
	}
}

func DropTable(schema, name string, ifexists bool) {
	db := GetDatabase(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}
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
	db.tables.Remove(name)
	db.saveLockedWithDurabilityAndUnlock(t.PersistencyMode == Safe)
	// fire AfterDropTable triggers after releasing schemalock (avoids deadlock on cascading drops)
	t.ExecuteTableLifecycleTriggers(AfterDropTable)

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
	db.ensureLoaded()
	db.schemalock.Lock()
	t := db.tables.Get(oldname)
	if t == nil {
		db.schemalock.Unlock()
		panic("Table " + schema + "." + oldname + " does not exist")
	}
	if db.tables.Get(newname) != nil {
		db.schemalock.Unlock()
		panic("Table " + schema + "." + newname + " already exists")
	}
	db.tables.Remove(oldname)
	t.Name = newname
	db.tables.Set(t)
	db.saveLockedWithDurabilityAndUnlock(t.PersistencyMode == Safe)
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
		if !db.schemalock.TryLock() {
			return false // schemalock is held (e.g. by CreateTable); retry later
		}
		db.tables.Remove(tbl.Name) // no-op if already removed by DropTable
		db.saveLockedWithDurabilityAndUnlock(tbl.PersistencyMode == Safe)
	}
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
		}
	}
	for _, s := range tbl.PShards {
		GlobalCache.removeInternal(s, freedByType)
		for _, idx := range s.Indexes {
			GlobalCache.removeInternal(idx, freedByType)
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
