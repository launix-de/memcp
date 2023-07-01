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

import "fmt"
import "sync"
import "github.com/launix-de/memcp/scm"

type dataset []scm.Scmer
type column struct {
	name string
	typ string
	typdimensions []int // type dimensions for DECIMAL(10,3) and VARCHAR(5)
	extrainfo string // TODO: further diversify into NOT NULL, AUTOINCREMENT etc.
}
type table struct {
	// schema
	name string
	columns []column
	mu sync.Mutex // schema lock

	// storage
	shards []*storageShard
}

const max_shardsize = 65536 // dont overload the shards to get a responsive parallel full table scan

// TODO: schemas, databases
var tables map[string]map[string]*table = make(map[string]map[string]*table)

func (d dataset) Get(key string) scm.Scmer {
	for i := 0; i < len(d); i += 2 {
		if d[i] == key {
			return d[i+1]
		}
	}
	return nil
}

func CreateDatabase(schema string) {
	if _, ok := tables[schema]; ok {
		panic("Database " + schema + " already exists")
	}
	tables[schema] = make(map[string]*table)
}

func DropDatabase(schema string) {
	delete(tables, schema)
}

func CreateTable(schema, name string) *table {
	if _, ok := tables[schema][name]; ok {
		panic("Table " + name + " already exists")
	}
	t := new(table)
	t.name = name
	t.shards = make([]*storageShard, 1)
	t.shards[0] = NewShard(t)
	tables[schema][t.name] = t
	return t
}

func DropTable(schema, name string) {
	// TODO: remove foreign keys etc.
	delete(tables[schema], name)
}

func (t *table) CreateColumn(name string, typ string, typdimensions[] int, extrainfo string) {
	t.mu.Lock()
	t.columns = append(t.columns, column{name, typ, typdimensions, extrainfo})
	for i := range t.shards {
		t.shards[i].columns[name] = new (StorageSCMER)
	}
	t.mu.Unlock()
}

func (t *table) Insert(d dataset) {
	// load balance: if bucket is full, create new one
	shard := t.shards[len(t.shards)-1]
	if shard.Count() >= max_shardsize {
		t.mu.Lock()
		// reload shard after lock to avoid race conditions
		shard = t.shards[len(t.shards)-1]
		if shard.Count() >= max_shardsize {
			go func(i int) {
				// rebuild full shards in background
				t.shards[i] = t.shards[i].rebuild()
			}(len(t.shards)-1)
			shard = NewShard(t)
			fmt.Println("started new shard for table", t.name)
			t.shards = append(t.shards, shard)
		}
		t.mu.Unlock()
	}
	// physically insert
	shard.Insert(d)
}
