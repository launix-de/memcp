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
	schemalock  sync.RWMutex                                       `json:"-"` // TODO: rw-locks for schemalock
	blobMu      sync.Mutex                                         `json:"-"` // serializes IncrBlobRefcount/DecrBlobRefcount

	// lazy-loading/shared-resource state (not serialized)
	srState SharedState `json:"-"`
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

func Rebuild(all bool, repartition bool) string {
	start := time.Now()
	dbs := databases.GetAll()
	for _, db := range dbs {
		func(db *database) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("error: rebuild/save failed for database", db.Name, ":", r)
				}
			}()
			db.rebuild(all, repartition)
			db.save()
		}(db)
	}
	return fmt.Sprint(time.Since(start))
}

func UnloadDatabases() {
	fmt.Println("table compression done in ", Rebuild(false, false))
	data, _ := json.Marshal(Settings)
	if settings, err := os.OpenFile(Basepath+"/settings.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640); err == nil {
		defer settings.Close()
		settings.Write(data)
	}
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

func (db *database) save() {
	if db.srState == COLD {
		// Do not serialize a cold database; keep existing schema.json intact
		return
	}
	jsonbytes, _ := json.MarshalIndent(db, "", "  ")
	db.persistence.WriteSchema(jsonbytes)
	// shards are written while rebuild
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

// SharedResource impl for database
func (db *database) GetState() SharedState { return db.srState }
func (db *database) GetRead() func()       { db.ensureLoaded(); return func() {} }
func (db *database) GetExclusive() func()  { db.ensureLoaded(); db.srState = WRITE; return func() {} }

// helper to fetch a table with lazy db load
func (db *database) GetTable(name string) *table {
	db.ensureLoaded()
	return db.tables.Get(name)
}

func (db *database) ShowTables() scm.Scmer {
	db.ensureLoaded()
	tables := db.tables.GetAll()
	result := make([]scm.Scmer, len(tables))
	i := 0
	for _, t := range tables {
		result[i] = scm.NewString(t.Name)
		i = i + 1
	}
	return scm.NewSlice(result)
}

func (db *database) rebuild(all bool, repartition bool) {
	if db.srState == COLD {
		// do nothing for cold databases; avoid loading during rebuild
		return
	}
	var done sync.WaitGroup
	// Collect pre-rebuild shards that were superseded. Their cleanup
	// (RemoveFromDisk → ReleaseBlobs → DecrBlobRefcount) must run after
	// ALL table rebuilds finish to avoid deadlocks with concurrent .blobs
	// repartition holding shard read-locks.
	var replacedMu sync.Mutex
	var allReplaced []*storageShard
	dbs := db.tables.GetAll()
	done.Add(len(dbs))
	for _, t := range dbs {
		go func(t *table) {
			tableLocked := false
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("error: rebuild failed for table", db.Name+".", t.Name, ":", r)
					// best-effort unlock if still locked
					if tableLocked {
						func() { defer func() { _ = recover() }(); t.mu.Unlock() }()
					}
				}
				done.Done()
			}()
			t.mu.Lock() // table lock
			tableLocked = true
			// TODO: check LRU statistics and remove unused computed columns

			// rebuild shards without mutating the live shard list; swap when complete
			targetIsP := t.ShardMode == ShardModePartition
			origShardList := t.ActiveShards()
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
								fmt.Println("error: shard rebuild failed for", db.Name+".", t.Name, "shard", i, ":", r, "\n", r)
							}
							sdone.Done()
						}()
						newShardList[i] = s.rebuild(all)
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
										fmt.Println("error: shard rebuild failed for", db.Name+".", t.Name, "shard", j.i, ":", r)
									}
									sdone.Done()
								}()
								newShardList[j.i] = j.s.rebuild(all)
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

			// Collect pre-rebuild shards that were replaced so we can clean them up.
			var replaced []*storageShard
			// Publish the new shard list only after completion.
			// To avoid slice header races for concurrent readers, update in place.
			if targetIsP {
				for i := range newShardList {
					if origShardList[i] != nil && origShardList[i] != newShardList[i] && origShardList[i].uuid != newShardList[i].uuid {
						replaced = append(replaced, origShardList[i])
					}
					origShardList[i] = newShardList[i]
				}
				t.PShards = origShardList
			} else {
				for i := range newShardList {
					if origShardList[i] != nil && origShardList[i] != newShardList[i] && origShardList[i].uuid != newShardList[i].uuid {
						replaced = append(replaced, origShardList[i])
					}
					origShardList[i] = newShardList[i]
				}
				t.Shards = origShardList
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
			var shardCandidates []shardDimension
			doRepart := false
			if repartition && !hasColdShard {
				var shouldChange bool
				shardCandidates, shouldChange = t.proposerepartition(maincount)
				doRepart = shouldChange || (t.ShardMode == ShardModeFree && t.Shards != nil)
			}

			t.mu.Unlock()
			tableLocked = false

			if doRepart {
				t.repartition(shardCandidates)
			}
		}(t)
	}
	done.Wait()

	// All table rebuilds (including .blobs) are complete. Safe to clean up
	// replaced pre-rebuild shards without risk of deadlocking with repartition.
	for _, s := range allReplaced {
		GlobalCache.Remove(s)
		s.RemoveFromDisk()
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

func DropDatabase(schema string, ifexists bool) bool {
	db := databases.Remove(schema)
	if db == nil {
		if ifexists {
			return false
		}
		panic("Database " + schema + " does not exist")
	}

	// remove remains of the folder structure
	db.persistence.Remove()
	return true
}

func CreateTable(schema, name string, pm PersistencyMode, ifnotexists bool) (*table, bool) {
	db := GetDatabase(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}
	db.ensureLoaded()
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	t := db.tables.Get(name)
	if t != nil {
		if ifnotexists {
			return t, false // return the table found
		}
		panic("Table " + name + " already exists")
	}
	t = new(table)
	t.schema = db
	t.Name = name
	t.PersistencyMode = pm
	t.ShardMode = ShardModeFree
	t.Shards = make([]*storageShard, 1)
	t.Shards[0] = NewShard(t)
	t.Auto_increment = 1
	t2 := db.tables.Set(t)
	if t2 != nil {
		// concurrent create
		panic("Table " + name + " already exists")
	} else {
		db.save()
	}
	// register temp keytable with CacheManager (`.`-prefix = temp)
	if strings.HasPrefix(name, ".") {
		schemaName := schema
		GlobalCache.AddItem(t, 0, TypeTempKeytable, func(ptr any, freedByType *[numEvictableTypes]int64) {
			keytableCleanup(ptr.(*table), schemaName, freedByType)
		}, keytableLastUsed, nil)
	}
	return t, true
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
	// deregister temp keytable from CacheManager
	GlobalCache.Remove(t)
	db.tables.Remove(name)
	db.save()
	db.schemalock.Unlock()

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
	defer db.schemalock.Unlock()
	t := db.tables.Get(oldname)
	if t == nil {
		panic("Table " + schema + "." + oldname + " does not exist")
	}
	if db.tables.Get(newname) != nil {
		panic("Table " + schema + "." + newname + " already exists")
	}
	db.tables.Remove(oldname)
	t.Name = newname
	db.tables.Set(t)
	db.save()
}

// keytableCleanup is called by the CacheManager when evicting a temp keytable.
// MUST NOT call public GlobalCache.Remove (deadlock: we're inside the CacheManager goroutine).
func keytableCleanup(tbl *table, schemaName string, freedByType *[numEvictableTypes]int64) {
	// remove all shard+index registrations for this table (recursive)
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
	// drop the table directly (bypass DropTable to avoid deadlock on opChan)
	db := GetDatabase(schemaName)
	if db != nil {
		db.schemalock.Lock()
		db.tables.Remove(tbl.Name)
		db.save()
		db.schemalock.Unlock()
		for _, s := range tbl.Shards {
			s.RemoveFromDisk()
		}
		for _, s := range tbl.PShards {
			s.RemoveFromDisk()
		}
	}
}

func keytableLastUsed(ptr any) time.Time {
	tbl := ptr.(*table)
	// use the newest shard lastAccessed as proxy
	var latest time.Time
	for _, s := range tbl.Shards {
		if s.lastAccessed.After(latest) {
			latest = s.lastAccessed
		}
	}
	for _, s := range tbl.PShards {
		if s.lastAccessed.After(latest) {
			latest = s.lastAccessed
		}
	}
	return latest
}
