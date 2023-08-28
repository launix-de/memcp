/*
Copyright (C) 2023  Carl-Philip Hänsch

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

type database struct {
	Name string `json:"name"`
	path string `json:"-"`
	Tables map[string]*table `json:"tables"`
	schemalock sync.Mutex `json:"-"`
}
var databases map[string]*database = make(map[string]*database)
var databaselock sync.Mutex
var Basepath string = "data"

func LoadDatabases() {
	databaselock.Lock()
	entries, _ := os.ReadDir(Basepath)
	for _, entry := range entries {
		if entry.IsDir() {
			// load database from hdd
			db := new(database)
			db.path = Basepath + "/" + entry.Name() + "/"
			jsonbytes, _ := os.ReadFile(db.path + "schema.json")
			json.Unmarshal(jsonbytes, db) // json import
			databases[db.Name] = db
			// restore back references of the tables
			for _, t := range db.Tables {
				t.schema = db // restore schema reference
				for _, s := range t.Shards {
					s.load(t)
				}
			}
		}
	}
	databaselock.Unlock()
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

func (db *database) ShowTables() []string {
	result := make([]string, len(db.Tables))
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

func CreateDatabase(schema string) {
	databaselock.Lock()
	if _, ok := databases[schema]; ok {
		panic("Database " + schema + " already exists")
	}
	db := new(database)
	db.Name = schema
	db.path = Basepath + "/" + schema + "/" // TODO: alternative save paths
	db.Tables = make(map[string]*table)
	databases[schema] = db
	databaselock.Unlock()
	db.save()
}

func DropDatabase(schema string) {
	databaselock.Lock()
	delete(databases, schema)
	databaselock.Unlock()
	// TODO: remove folder of that database
}

func CreateTable(schema, name string) *table {
	db, ok := databases[schema]
	if !ok {
		panic("Database " + schema + " does not exist")
	}
	db.schemalock.Lock()
	if _, ok := db.Tables[name]; ok {
		panic("Table " + name + " already exists")
	}
	t := new(table)
	t.schema = db
	t.Name = name
	t.Shards = make([]*storageShard, 1)
	t.Shards[0] = NewShard(t)
	db.Tables[t.Name] = t
	db.save()
	db.schemalock.Unlock()
	return t
}

func DropTable(schema, name string) {
	// TODO: remove foreign keys etc.
	db, ok := databases[schema]
	if !ok {
		panic("Database " + schema + " does not exist")
	}
	db.schemalock.Lock()
	delete(db.Tables, name)
	db.save()
	db.schemalock.Unlock()
}

