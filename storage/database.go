/*
Copyright (C) 2023, 2024  Carl-Philip Hänsch

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
import "encoding/json"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/NonLockingReadMap"

type database struct {
	Name        string                                             `json:"name"`
	persistence PersistenceEngine                                  `json:"-"`
	tables      NonLockingReadMap.NonLockingReadMap[table, string] `json:"-"`
	schemalock  sync.RWMutex                                       `json:"-"` // TODO: rw-locks for schemalock

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
		db.rebuild(all, repartition)
		db.save()
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
			db.persistence = &FileStorage{Basepath + "/" + entry.Name() + "/"}
			db.srState = COLD
			databases.Set(db)
		} else {
			// TODO: read .json files for S3 tables
		}
	}
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
	}
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
		result[i] = t.Name
		i = i + 1
	}
	return result
}

func (db *database) rebuild(all bool, repartition bool) {
	if db.srState == COLD {
		// do nothing for cold databases; avoid loading during rebuild
		return
	}
	var done sync.WaitGroup
	dbs := db.tables.GetAll()
	done.Add(len(dbs))
	for _, t := range dbs {
		go func(t *table) {
			t.mu.Lock() // table lock
			// TODO: check LRU statistics and remove unused computed columns

			// rebuild shards
			shardlist := t.Shards // if Shards AND PShards are present, Shards is the single point of truth
			if shardlist == nil {
				shardlist = t.PShards
			}
			maincount := uint(0)
			var sdone sync.WaitGroup
			sdone.Add(len(shardlist))
			// throttle concurrent shard rebuilds by CPU count
			workers := runtime.NumCPU()
			if workers < 1 {
				workers = 1
			}
			type job struct {
				i int
				s *storageShard
			}
			if len(shardlist) <= workers {
				for i, s := range shardlist {
					maincount += s.main_count + uint(len(s.inserts)) - uint(s.deletions.Count())
					go func(i int, s *storageShard) {
						shardlist[i] = s.rebuild(all)
						sdone.Done()
					}(i, s)
				}
			} else {
				jobs := make(chan job, workers)
				// launch workers
				for w := 0; w < workers; w++ {
					go func() {
						for j := range jobs {
							shardlist[j.i] = j.s.rebuild(all)
							sdone.Done()
						}
					}()
				}
				for i, s := range shardlist {
					maincount += s.main_count + uint(len(s.inserts)) - uint(s.deletions.Count())
					jobs <- job{i: i, s: s}
				}
				close(jobs)
			}
			sdone.Wait()

			// check if we should do the repartitioning
			if repartition {
				shardCandidates, shouldChange := t.proposerepartition(maincount)
				if shouldChange || (t.PShards != nil && t.Shards != nil) {
					t.repartition(shardCandidates) // perform the repartitioning
				}
			}

			t.mu.Unlock()
			done.Done()
		}(t)
	}
	done.Wait()
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
	db.tables.Remove(name)
	db.save()
	db.schemalock.Unlock()

	// delete shard files from disk
	for _, s := range t.Shards {
		s.RemoveFromDisk()
	}
	for _, s := range t.PShards {
		s.RemoveFromDisk()
	}
}
