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
import "sort"
import "sync/atomic"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

type database struct {
	Name string `json:"name"`
	path string `json:"-"`
	Tables map[string]*table `json:"tables"`
	schemalock sync.RWMutex `json:"-"` // TODO: rw-locks for schemalock
}
// TODO: replace databases map everytime something changes, so we don't run into read-while-write
// e.g. a table of databases
var databases atomic.Pointer[[]*database]
var Basepath string = "data"

func LoadDatabases() {
	// this happens before any init, so no read/write action is performed on any data yet
	dbs := databases.Load()
	if dbs == nil {
		dbs = new([]*database)
	}
	entries, _ := os.ReadDir(Basepath)
	for _, entry := range entries {
		if entry.IsDir() {
			// load database from hdd
			db := new(database)
			db.path = Basepath + "/" + entry.Name() + "/"
			jsonbytes, _ := os.ReadFile(db.path + "schema.json")
			json.Unmarshal(jsonbytes, db) // json import
			*dbs = append(*dbs, db)
			// restore back references of the tables
			for _, t := range db.Tables {
				t.schema = db // restore schema reference
				for _, s := range t.Shards {
					s.load(t)
				}
			}
		}
	}
	sort.Slice(*dbs, func (i, j int) bool {
		return (*dbs)[i].Name < (*dbs)[j].Name;
	})
	databases.Store(dbs)
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
	result := make([]scm.Scmer, len(db.Tables))
	i := 0
	for k, _ := range db.Tables {
		result[i] = k
		i = i + 1
	}
	return result
}

func (db *database) rebuild() {
	for _, t := range db.Tables {
		t.mu.Lock() // table lock
		for i, s := range t.Shards {
			// TODO: go + chan done
			t.Shards[i] = s.rebuild()
		}
		t.mu.Unlock() // TODO: do this after chan done??
	}
}

func GetDatabase(schema string) *database {
	dbs := databases.Load() // atomically work on the current database list
	var lower int = 0
	var upper int = len(*dbs)
	for {
		if lower == upper {
			return nil // database does not exist
		}
		pivot := (lower + upper) / 2
		db := (*dbs)[pivot]
		if schema == db.Name {
			return db // found database
		} else if schema < db.Name {
			upper = pivot
		} else {
			lower = pivot + 1
		}
	}
}

func CreateDatabase(schema string) {
	start:
	old_dbs := databases.Load()
	dbs := new([]*database)
	*dbs = *old_dbs
	// duplicate check
	var lower int = 0
	var upper int = len(*dbs)
	for {
		if lower == upper {
			// database does not exist -> create
			break
		}
		pivot := (lower + upper) / 2
		db := (*dbs)[pivot]
		if schema == db.Name {
			panic("Database " + schema + " already exists")
		} else if schema < db.Name {
			upper = pivot
		} else {
			lower = pivot + 1
		}
	}

	db := new(database)
	db.Name = schema
	db.path = Basepath + "/" + schema + "/" // TODO: alternative save paths
	db.Tables = make(map[string]*table)
	*dbs = make([]*database, len(*old_dbs) + 1)
	*dbs = append(*dbs, (*old_dbs)...)
	*dbs = append(*dbs, db)
	sort.Slice(*dbs, func (i, j int) bool {
		return (*dbs)[i].Name < (*dbs)[j].Name;
	})
	if databases.CompareAndSwap(old_dbs, dbs) {
		db.save()
	} else {
		goto start
	}
}

func DropDatabase(schema string) {
	start:
	old_dbs := databases.Load()
	dbs := new([]*database)
	*dbs = *old_dbs
	// duplicate check
	var lower int = 0
	var upper int = len(*dbs)
	var pivot int
	var db *database
	for {
		if lower == upper {
			panic("Database " + schema + " does not exist")
		}
		pivot = (lower + upper) / 2
		db = (*dbs)[pivot]
		if schema == db.Name {
			break // found, ok
		} else if schema < db.Name {
			upper = pivot
		} else {
			lower = pivot + 1
		}
	}

	(*dbs)[pivot] = (*dbs)[len(*dbs)-1]
	*dbs = (*dbs)[0:len(*dbs)-1] // remove one element
	sort.Slice(*dbs, func (i, j int) bool {
		return (*dbs)[i].Name < (*dbs)[j].Name;
	})
	if databases.CompareAndSwap(old_dbs, dbs) {
		os.RemoveAll(db.path)
		db.save()
	} else {
		goto start
	}
}

func CreateTable(schema, name string, pm PersistencyMode) *table {
	db := GetDatabase(schema)
	if db == nil {
		panic("Database " + schema + " does not exist")
	}
	db.schemalock.Lock()
	if _, ok := db.Tables[name]; ok {
		panic("Table " + name + " already exists")
	}
	t := new(table)
	t.schema = db
	t.Name = name
	t.PersistencyMode = pm
	t.Shards = make([]*storageShard, 1)
	t.Shards[0] = NewShard(t)
	db.Tables[t.Name] = t
	db.save()
	db.schemalock.Unlock()
	return t
}

func DropTable(schema, name string) {
	db := GetDatabase(schema)
	if db != nil {
		panic("Database " + schema + " does not exist")
	}
	t, ok := db.Tables[name]
	if !ok {
		panic("Table " + schema + "." + name + " does not exist")
	}
	db.schemalock.Lock()
	delete(db.Tables, name)
	db.save()
	db.schemalock.Unlock()

	// delete shard files from disk
	for _, s := range t.Shards {
		s.RemoveFromDisk()
	}
}

