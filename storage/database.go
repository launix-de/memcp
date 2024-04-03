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
import "sync"
import "encoding/json"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/NonLockingReadMap"

type database struct {
	Name string `json:"name"`
	path string `json:"-"`
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

func LoadDatabases() {
	// this happens before any init, so no read/write action is performed on any data yet
	done := make(chan bool, 200)
	entries, _ := os.ReadDir(Basepath)
	for _, entry := range entries {
		if entry.IsDir() {
			// load database from hdd
			db := new(database)
			db.path = Basepath + "/" + entry.Name() + "/"
			jsonbytes, _ := os.ReadFile(db.path + "schema.json")
			json.Unmarshal(jsonbytes, db) // json import
			// restore back references of the tables
			for _, t := range db.Tables.GetAll() {
				t.schema = db // restore schema reference
				for _, s := range t.Shards {
					go func(t *table, s *storageShard) {
						s.load(t)
						done <- true
					}(t, s)
				}
			}
			databases.Set(db)
		}
	}
	// wait for all loading go routines to finish
	for _, entry := range entries {
		if entry.IsDir() {
			db := databases.Get(entry.Name())
			for _, t := range db.Tables.GetAll() {
				for range t.Shards {
					<-done // wait for shard
				}
			}
		}
	}
}

func (db *database) save() {
	os.MkdirAll(db.path, 0750)
	f, err := os.Create(db.path + "schema.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	jsonbytes, _ := json.MarshalIndent(db, "", "  ")
	f.Write(jsonbytes)
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

func (db *database) rebuild() {
	for _, t := range db.Tables.GetAll() {
		t.mu.Lock() // table lock
		for i, s := range t.Shards {
			// TODO: go + chan done
			t.Shards[i] = s.rebuild()
		}
		t.mu.Unlock() // TODO: do this after chan done??
	}
}

func GetDatabase(schema string) *database {
	return databases.Get(schema)
}

func CreateDatabase(schema string) {
	db := databases.Get(schema)
	if db != nil {
		panic("Database " + schema + " already exists")
	}

	db = new(database)
	db.Name = schema
	db.path = Basepath + "/" + schema + "/" // TODO: alternative save paths
	db.Tables = NonLockingReadMap.New[table, string]()

	last := databases.Set(db)
	if last != nil {
		// two concurrent CREATE
		databases.Set(last)
		panic("Database " + schema + " already exists")
	}

	db.save()
}

func DropDatabase(schema string) {
	db := databases.Remove(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}

	// remove remains of the folder structure
	os.RemoveAll(db.path)
}

func CreateTable(schema, name string, pm PersistencyMode, ifnotexists bool) *table {
	db := GetDatabase(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	t := db.Tables.Get(name)
	if t != nil {
		if ifnotexists {
			return t // return the table found
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
	return t
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
}

