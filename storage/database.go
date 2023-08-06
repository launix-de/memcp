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

import "sync"

type database struct {
	Name string `json:"name"`
	path string `json:"-"`
	Tables map[string]*table `json:"tables"`
	schemalock sync.Mutex `json:"-"`
}
var databases map[string]*database = make(map[string]*database)
var databaselock sync.Mutex
var basepath string = "data"

func CreateDatabase(schema string) {
	databaselock.Lock()
	if _, ok := databases[schema]; ok {
		panic("Database " + schema + " already exists")
	}
	db := new(database)
	db.Name = schema
	db.path = basepath + "/" + schema + "/" // TODO: alternative save paths
	db.Tables = make(map[string]*table)
	databases[schema] = db
	databaselock.Unlock()
}

func DropDatabase(schema string) {
	databaselock.Lock()
	delete(databases, schema)
	databaselock.Unlock()
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
	t.Name = name
	t.Shards = make([]*storageShard, 1)
	t.Shards[0] = NewShard(t)
	db.Tables[t.Name] = t
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
	db.schemalock.Unlock()
}

