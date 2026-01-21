/*
Copyright (C) 2023, 2024, 2026  Carl-Philip Hänsch

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

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/launix-de/memcp/scm"
)

var Basepath string = "data"

type persistedSchema struct {
	Name   string            `json:"name"`
	Tables map[string]*table `json:"tables"`
}

var schemaLocksMu sync.Mutex
var schemaLocks = map[string]*sync.Mutex{}

func schemaLock(schema string) *sync.Mutex {
	schemaLocksMu.Lock()
	defer schemaLocksMu.Unlock()
	if m := schemaLocks[schema]; m != nil {
		return m
	}
	m := new(sync.Mutex)
	schemaLocks[schema] = m
	return m
}

func persistenceForSchema(schema string) PersistenceEngine {
	return &FileStorage{Basepath + "/" + schema + "/"}
}

func schemaDir(schema string) string { return filepath.Join(Basepath, schema) }

func schemaExists(schema string) bool {
	_, err := os.Stat(schemaDir(schema))
	return err == nil
}

func listTablesInSchema(schema string) []string {
	ps := readSchema(schema)
	out := make([]string, 0, len(ps.Tables))
	for name := range ps.Tables {
		out = append(out, name)
	}
	return out
}

func GetTable(schema string, name string) *table {
	if Tables != nil {
		if t := lookupTableHandle(schema, name); t != nil {
			t.Schema = schema
			t.Name = name
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
			return t
		}
	}
	mu := schemaLock(schema)
	mu.Lock()
	defer mu.Unlock()
	ps := readSchema(schema)
	t := ps.Tables[name]
	if t == nil {
		return nil
	}
	t.Schema = schema
	t.Name = name
	ensureTableID(t)
	registerTable(t)
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
	addToTablesCatalog(schema, name, t)
	return t
}

func readSchema(schema string) *persistedSchema {
	p := persistenceForSchema(schema)
	b := p.ReadSchema()
	if len(b) == 0 {
		return &persistedSchema{Name: schema, Tables: map[string]*table{}}
	}
	tmp := new(persistedSchema)
	if err := json.Unmarshal(b, tmp); err != nil {
		panic(err)
	}
	if tmp.Name == "" {
		tmp.Name = schema
	}
	if tmp.Tables == nil {
		tmp.Tables = map[string]*table{}
	}
	return tmp
}

func writeSchema(schema string, ps *persistedSchema) {
	p := persistenceForSchema(schema)
	b, _ := json.MarshalIndent(ps, "", "  ")
	p.WriteSchema(b)
}

func listSchemasOnDisk() []string {
	entries, _ := os.ReadDir(Basepath)
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

func LoadDatabases() {
	// settings + optional estimator
	if settings, err := os.Open(Basepath + "/settings.json"); err == nil {
		defer settings.Close()
		stat, _ := settings.Stat()
		data := make([]byte, stat.Size())
		if _, err := settings.Read(data); err == nil {
			_ = json.Unmarshal(data, &Settings)
		}
	}
	InitSettings()
	if Settings.AIEstimator {
		StartGlobalEstimator()
	}

	initTablesCatalog()

	// load tables from disk into registry and catalog
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
			addToTablesCatalog(schema, name, t)
		}
	}

	ensureSystemStatistic()

	// Refresh Scheme global var in case Storage.Init already ran.
	if Tables != nil {
		scm.Globalenv.Vars[scm.Symbol("Tables")] = scm.NewTableRef(Tables.ID)
	}
}

func UnloadDatabases() {
	fmt.Println("table compression done in ", Rebuild(false, false))
	data, _ := json.Marshal(Settings)
	if settings, err := os.OpenFile(Basepath+"/settings.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640); err == nil {
		defer settings.Close()
		_, _ = settings.Write(data)
	}
	StopGlobalEstimator()
}

func Rebuild(all bool, repartition bool) string {
	start := time.Now()
	// Best-effort: rebuild all known tables by scanning the catalog registry.
	tableRegistryMu.RLock()
	allTables := make([]*table, 0, len(tableByID))
	for _, t := range tableByID {
		if t != nil && t != Tables {
			allTables = append(allTables, t)
		}
	}
	tableRegistryMu.RUnlock()

	var done sync.WaitGroup
	done.Add(len(allTables))
	for _, t := range allTables {
		go func(t *table) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("error: rebuild/save failed for table", t.Schema+".", t.Name, ":", r)
					func() { defer func() { _ = recover() }(); t.mu.Unlock() }()
				}
				done.Done()
			}()
			rebuildTable(t, all, repartition)
		}(t)
	}
	done.Wait()
	return fmt.Sprint(time.Since(start))
}

func rebuildTable(t *table, all bool, repartition bool) {
	if t == nil {
		return
	}
	t.mu.Lock()

	origShardList := t.Shards
	targetIsP := false
	if origShardList == nil {
		origShardList = t.PShards
		targetIsP = true
	}
	newShardList := make([]*storageShard, len(origShardList))
	hasColdShard := false

	getShardCount := func(s *storageShard) uint {
		if s == nil {
			return 0
		}
		s.mu.RLock()
		c := s.main_count + uint(len(s.inserts)) - uint(s.deletions.Count())
		s.mu.RUnlock()
		return c
	}
	maincount := uint(0)

	var sdone sync.WaitGroup
	sdone.Add(len(origShardList))
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
						fmt.Println("error: shard rebuild failed for", t.Schema+".", t.Name, "shard", i, ":", r)
					}
					sdone.Done()
				}()
				newShardList[i] = s.rebuild(all)
			}(i, s)
		}
	} else {
		jobs := make(chan job, workers)
		for w := 0; w < workers; w++ {
			go func() {
				for j := range jobs {
					func(j job) {
						defer func() {
							if r := recover(); r != nil {
								fmt.Println("error: shard rebuild failed for", t.Schema+".", t.Name, "shard", j.i, ":", r)
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

	if targetIsP {
		for i := range newShardList {
			origShardList[i] = newShardList[i]
		}
		t.PShards = origShardList
	} else {
		for i := range newShardList {
			origShardList[i] = newShardList[i]
		}
		t.Shards = origShardList
	}

	if repartition && !hasColdShard {
		shardCandidates, shouldChange := t.proposerepartition(maincount)
		if shouldChange || (t.PShards != nil && t.Shards != nil) {
			t.repartition(shardCandidates)
		}
	}

	t.mu.Unlock()
	saveTableMetadata(t)
}

func CreateDatabase(schema string, ignoreexists bool) bool {
	dir := schemaDir(schema)
	if _, err := os.Stat(dir); err == nil {
		if ignoreexists {
			return false
		}
		panic("Database " + schema + " already exists")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		panic(err)
	}
	// create empty schema.json
	mu := schemaLock(schema)
	mu.Lock()
	writeSchema(schema, &persistedSchema{Name: schema, Tables: map[string]*table{}})
	mu.Unlock()
	return true
}

func DropDatabase(schema string, ifexists bool) bool {
	dir := schemaDir(schema)
	if _, err := os.Stat(dir); err != nil {
		if ifexists {
			return false
		}
		panic("Database " + schema + " does not exist")
	}
	if err := os.RemoveAll(dir); err != nil {
		panic(err)
	}
	// remove tables from registry/catalog
	// (best-effort: scan Tables catalog for this schema and unregister)
	if Tables != nil {
		// Note: removeFromTablesCatalog removes a single row; do a quick rebuild instead.
		rebuildTablesCatalogFromDatabases()
	}
	return true
}

func saveTableMetadata(t *table) {
	saveTableMetadataEx(t, false)
}

func saveTableMetadataLocked(t *table) {
	saveTableMetadataEx(t, true)
}

func saveTableMetadataEx(t *table, alreadyLocked bool) {
	if t == nil || t.Schema == "" || t.Name == "" {
		return
	}
	mu := schemaLock(t.Schema)
	if !alreadyLocked {
		mu.Lock()
		defer mu.Unlock()
	}
	ps := readSchema(t.Schema)
	if ps.Tables == nil {
		ps.Tables = map[string]*table{}
	}
	ps.Tables[t.Name] = t
	writeSchema(t.Schema, ps)
}

func CreateTable(schema, name string, pm PersistencyMode, ifnotexists bool) (*table, bool) {
	// Ensure schema exists on disk
	_ = CreateDatabase(schema, true)

	mu := schemaLock(schema)
	mu.Lock()
	defer mu.Unlock()
	ps := readSchema(schema)
	if ps.Tables == nil {
		ps.Tables = map[string]*table{}
	}
	if existing := ps.Tables[name]; existing != nil {
		if ifnotexists {
			existing.Schema = schema
			existing.Name = name
			ensureTableID(existing)
			registerTable(existing)
			return existing, false
		}
		panic("Table " + name + " already exists")
	}
	t := new(table)
	t.Schema = schema
	t.Name = name
	t.PersistencyMode = pm
	t.Shards = make([]*storageShard, 1)
	t.Shards[0] = NewShard(t)
	t.Auto_increment = 1
	ensureTableID(t)
	ps.Tables[name] = t
	writeSchema(schema, ps)

	registerTable(t)
	addToTablesCatalog(schema, name, t)
	return t, true
}

func DropTable(schema, name string, ifexists bool) {
	mu := schemaLock(schema)
	mu.Lock()
	ps := readSchema(schema)
	t := ps.Tables[name]
	if t == nil {
		mu.Unlock()
		if ifexists {
			return
		}
		panic("Table " + schema + "." + name + " does not exist")
	}
	t.Schema = schema
	t.Name = name
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
	delete(ps.Tables, name)
	writeSchema(schema, ps)
	mu.Unlock()

	removeFromTablesCatalog(schema, name)
	unregisterTable(t)

	// delete shard files from disk
	for _, s := range t.Shards {
		if s != nil {
			s.RemoveFromDisk()
		}
	}
	for _, s := range t.PShards {
		if s != nil {
			s.RemoveFromDisk()
		}
	}
}
