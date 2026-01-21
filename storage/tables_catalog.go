/*
Copyright (C) 2026  Carl-Philip Hänsch

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
import "sync/atomic"
import "time"

import "github.com/launix-de/memcp/scm"

// Tables is the global root-catalog. It stores (schema, table, handle).
// handle is a Scheme value that encodes the table UUID.
var Tables *table

var tableRegistryMu sync.RWMutex
var tableByID = map[string]*table{}
var tableIDCounter uint64 = uint64(time.Now().UnixNano())

func registerTable(t *table) {
	if t == nil || t.ID == "" {
		return
	}
	tableRegistryMu.Lock()
	tableByID[t.ID] = t
	tableRegistryMu.Unlock()
}

func unregisterTable(t *table) {
	if t == nil || t.ID == "" {
		return
	}
	tableRegistryMu.Lock()
	if cur := tableByID[t.ID]; cur == t {
		delete(tableByID, t.ID)
	}
	tableRegistryMu.Unlock()
}

func resolveTableHandle(v scm.Scmer) *table {
	if !v.IsTableRef() {
		return nil
	}
	id := v.TableRef()
	if id == "" {
		return nil
	}
	tableRegistryMu.RLock()
	t := tableByID[id]
	tableRegistryMu.RUnlock()
	return t
}

func ensureTableID(t *table) {
	if t != nil && t.ID == "" {
		ctr := atomic.AddUint64(&tableIDCounter, 1)
		id := newUUID()
		// Mix counter into the UUID to avoid collisions across restarts.
		b := id
		b[0] ^= byte(ctr)
		b[1] ^= byte(ctr >> 8)
		b[2] ^= byte(ctr >> 16)
		b[3] ^= byte(ctr >> 24)
		t.ID = b.String()
	}
}

func initTablesCatalog() {
	if Tables != nil {
		return
	}
	Tables = new(table)
	Tables.Schema = "system"
	Tables.Name = "tables"
	Tables.PersistencyMode = Memory
	Tables.Shards = make([]*storageShard, 1)
	Tables.Shards[0] = NewShard(Tables)
	Tables.Auto_increment = 1
	ensureTableID(Tables)
	registerTable(Tables)

	// Minimal schema: schema TEXT, table TEXT, handle SCMER
	Tables.Columns = []column{
		{Name: "schema", Typ: "TEXT", AllowNull: false},
		{Name: "table", Typ: "TEXT", AllowNull: false},
		{Name: "handle", Typ: "SCMER", AllowNull: false},
	}
	for i := range Tables.Columns {
		Tables.Columns[i].UpdateSanitizer()
	}
	Tables.Unique = []uniqueKey{
		{Id: "PRIMARY", Cols: []string{"schema", "table"}},
	}
	// Ensure shard columns exist for the schema we just assigned.
	for _, c := range Tables.Columns {
		Tables.Shards[0].mu.Lock()
		Tables.Shards[0].columns[c.Name] = new(StorageSparse)
		Tables.Shards[0].mu.Unlock()
	}
}

func rebuildTablesCatalogFromDatabases() {
	if Tables == nil {
		return
	}
	// best-effort: keep existing in-memory catalog but rebuild registry
	Tables.Shards[0].mu.Lock()
	Tables.Shards[0].inserts = nil
	Tables.Shards[0].deletions.Reset()
	Tables.Shards[0].mu.Unlock()

	for _, schema := range listSchemasOnDisk() {
		ps := readSchema(schema)
		for name, t := range ps.Tables {
			if t == nil {
				continue
			}
			t.Schema = schema
			t.Name = name
			ensureTableID(t)
			registerTable(t)
			addToTablesCatalog(schema, name, t)
		}
	}
}

func addToTablesCatalog(schema string, name string, t *table) {
	if Tables == nil || t == nil {
		return
	}
	row := []scm.Scmer{
		scm.NewString(schema),
		scm.NewString(name),
		scm.NewTableRef(t.ID),
	}
	Tables.Insert([]string{"schema", "table", "handle"}, [][]scm.Scmer{row}, nil, scm.NewNil(), false, nil)
}

func removeFromTablesCatalog(schema string, name string) {
	if Tables == nil {
		return
	}
	shard := Tables.Shards[0]
	shard.ensureLoaded()
	// Preload storages outside lock to avoid lazy-load under lock.
	schemaCol := shard.getColumnStorageOrPanic("schema")
	tableCol := shard.getColumnStorageOrPanic("table")
	shard.mu.Lock()
	defer shard.mu.Unlock()
	// scan main
	for i := uint(0); i < shard.main_count; i++ {
		if shard.deletions.Get(i) {
			continue
		}
		if scm.String(schemaCol.GetValue(i)) == schema && scm.String(tableCol.GetValue(i)) == name {
			shard.deletions.Set(i, true)
			return
		}
	}
	// scan delta
	for i := 0; i < len(shard.inserts); i++ {
		recid := shard.main_count + uint(i)
		if shard.deletions.Get(recid) {
			continue
		}
		ds := shard.inserts[i]
		// schema/table are guaranteed present in delta for our inserts
		schemaIdx := shard.deltaColumns["schema"]
		tableIdx := shard.deltaColumns["table"]
		if scm.String(ds[schemaIdx]) == schema && scm.String(ds[tableIdx]) == name {
			shard.deletions.Set(recid, true)
			return
		}
	}
}

func lookupTableHandle(schema string, name string) *table {
	if Tables == nil {
		return nil
	}
	shard := Tables.Shards[0]
	shard.ensureLoaded()
	// Preload storages outside lock to avoid lazy-load under lock.
	schemaCol := shard.getColumnStorageOrPanic("schema")
	tableCol := shard.getColumnStorageOrPanic("table")
	handleCol := shard.getColumnStorageOrPanic("handle")

	shard.mu.RLock()
	defer shard.mu.RUnlock()
	// main scan
	for i := uint(0); i < shard.main_count; i++ {
		if shard.deletions.Get(i) {
			continue
		}
		if scm.String(schemaCol.GetValue(i)) == schema && scm.String(tableCol.GetValue(i)) == name {
			return resolveTableHandle(handleCol.GetValue(i))
		}
	}
	// delta scan
	schemaIdx := shard.deltaColumns["schema"]
	tableIdx := shard.deltaColumns["table"]
	handleIdx := shard.deltaColumns["handle"]
	for i := 0; i < len(shard.inserts); i++ {
		recid := shard.main_count + uint(i)
		if shard.deletions.Get(recid) {
			continue
		}
		ds := shard.inserts[i]
		if scm.String(ds[schemaIdx]) == schema && scm.String(ds[tableIdx]) == name {
			return resolveTableHandle(ds[handleIdx])
		}
	}
	return nil
}
