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
import "encoding/json"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/NonLockingReadMap"

type database struct {
	Name string `json:"name"`
	persistence PersistenceEngine `json:"-"`
	Tables NonLockingReadMap.NonLockingReadMap[table, string] `json:"tables"`
	schemalock sync.RWMutex `json:"-"` // TODO: rw-locks for schemalock
}
// TODO: replace databases map everytime something changes, so we don't run into read-while-write
// e.g. a table of databases
var databases NonLockingReadMap.NonLockingReadMap[database, string] = NonLockingReadMap.New[database, string]()
var Basepath string = "data"

/* implement NonLockingReadMap */
func (d database) GetKey() string {
	return d.Name
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
	if settings, err := os.OpenFile(Basepath + "/settings.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640); err == nil {
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

	// load dbs
	var done sync.WaitGroup
	entries, _ := os.ReadDir(Basepath)
	for _, entry := range entries {
		if entry.IsDir() {
			// load database from hdd
			db := new(database)
			db.persistence = &FileStorage{Basepath + "/" + entry.Name() + "/"}
			jsonbytes := db.persistence.ReadSchema()
			if len(jsonbytes) == 0 {
				fmt.Println("Warning: database " + entry.Name() + " is empty")
			} else {
				json.Unmarshal(jsonbytes, db) // json import
				// restore back references of the tables
				for _, t := range db.Tables.GetAll() {
					t.schema = db // restore schema reference
					func (t *table) {
						t.iterateShards(nil, func (s *storageShard) {
							s.load(t)
						})
					}(t)
				}
				databases.Set(db)
			}
		} else {
			// TODO: read .json files for S3 tables
		}
	}
	// wait for all loading go routines to finish
	done.Wait()
}

func (db *database) save() {
	jsonbytes, _ := json.MarshalIndent(db, "", "  ")
	db.persistence.WriteSchema(jsonbytes)
	// shards are written while rebuild
}

func (db *database) ShowTables() scm.Scmer {
	tables := db.Tables.GetAll()
	result := make([]scm.Scmer, len(tables))
	i := 0
	for _, t := range tables {
		result[i] = t.Name
		i = i + 1
	}
	return result
}

func (db *database) rebuild(all bool, repartition bool) {
	var done sync.WaitGroup
	dbs := db.Tables.GetAll()
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
			for i, s := range shardlist {
				maincount += s.main_count + uint(len(s.inserts)) - uint(s.deletions.Count()) // estimate size of that table
				go func(shardlist []*storageShard, i int, s *storageShard) {
					shardlist[i] = s.rebuild(all)
					sdone.Done()
				}(shardlist, i, s)
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

func CreateDatabase(schema string, ignoreexists bool/*, persistence PersistenceFactory*/) bool {
	db := databases.Get(schema)
	if db != nil {
		if ignoreexists {
			return false
		}
		panic("Database " + schema + " already exists")
	}

	db = new(database)
	db.Name = schema
	persistence := FileFactory{Basepath}// TODO: remove this, use parameter instead
	db.persistence = persistence.CreateDatabase(schema)
	db.Tables = NonLockingReadMap.New[table, string]()

	last := databases.Set(db)
	if last != nil {
		// two concurrent CREATE
		databases.Set(last)
		panic("Database " + schema + " already exists")
	}

	db.save()
	return true
}

func DropDatabase(schema string) {
	db := databases.Remove(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}

	// remove remains of the folder structure
	db.persistence.Remove()
}

func CreateTable(schema, name string, pm PersistencyMode, ifnotexists bool) (*table, bool) {
	db := GetDatabase(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	t := db.Tables.Get(name)
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
	t2 := db.Tables.Set(t)
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
	db.schemalock.Lock()
	t := db.Tables.Get(name)
	if t == nil {
		db.schemalock.Unlock()
		if ifexists {
			return // silentfail
		}
		panic("Table " + schema + "." + name + " does not exist")
	}
	db.Tables.Remove(name)
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

