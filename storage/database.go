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
import "sort"
import "encoding/json"
import "github.com/lrita/numa"
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

func Rebuild(all bool) string {
	start := time.Now()
	dbs := databases.GetAll()
	for _, db := range dbs {
		db.rebuild(all)
		db.save()
	}
	return fmt.Sprint(time.Since(start))
}

func LoadDatabases() {
	// this happens before any init, so no read/write action is performed on any data yet
	var done sync.WaitGroup
	entries, _ := os.ReadDir(Basepath)
	for _, entry := range entries {
		if entry.IsDir() {
			// load database from hdd
			db := new(database)
			db.path = Basepath + "/" + entry.Name() + "/"
			jsonbytes, _ := os.ReadFile(db.path + "schema.json")
			if len(jsonbytes) == 0 {
				// try to load backup (in case of failure while save)
				jsonbytes, _ = os.ReadFile(db.path + "schema.json.old")
			}
			if len(jsonbytes) == 0 {
				fmt.Println("Warning: database " + entry.Name() + " is empty")
			} else {
				json.Unmarshal(jsonbytes, db) // json import
				// restore back references of the tables
				for _, t := range db.Tables.GetAll() {
					t.schema = db // restore schema reference
					done.Add(len(t.Shards))
					for _, s := range t.Shards {
						go func(t *table, s *storageShard) {
							s.load(t) // this captures current node id of shard
							done.Done()
						}(t, s)
					}
				}
				databases.Set(db)
			}
		}
	}
	// wait for all loading go routines to finish
	done.Wait()
}

func (db *database) save() {
	os.MkdirAll(db.path, 0750)
	if stat, err := os.Stat(db.path + "schema.json"); err == nil && stat.Size() > 0 {
		// rescue a copy of schema.json in case the schema is not serializable
		os.Rename(db.path + "schema.json", db.path + "schema.json.old")
	}
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

func (db *database) rebuild(all bool) {
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
			var sdone sync.WaitGroup
			maincount := uint(0)
			sdone.Add(len(shardlist))
			for i, s := range shardlist {
				maincount += s.main_count + uint(len(s.inserts)) // estimate size of that table
				go func(shardlist []*storageShard, i int, s *storageShard) {
					// reshuffle numa awareness, so memory can reorganize during rebuild
					numa.RunOnNode(-1)
					s.RunOn()
					shardlist[i] = s.rebuild(all)
					sdone.Done()
				}(shardlist, i, s)
			}
			sdone.Wait()

			// reevaluate partitioning schema
			var shardCandidates []shardDimension
			for _, c := range t.Columns {
				if c.PartitioningScore > 0 {
					shardCandidates = append(shardCandidates, shardDimension{c.Name, c.PartitioningScore, nil})
				}
			}
			if len(shardCandidates) > 0 {
				// sort for highest ranking column
				sort.Slice(shardCandidates, func (i, j int) bool { // Less
					return shardCandidates[i].NumPartitions > shardCandidates[j].NumPartitions
				})
				sf := 0.01 // scale factor
				desiredNumberOfShards := maincount / 30000 + 1 // keep some extra room
				for iter := 2; iter < 30; iter++ { // find perfect scale factor such that we get the best number of shards
					deviation := 1
					for _, sc := range shardCandidates {
						deviation *= int(float64(sc.NumPartitions) * sf)
					}
					deviation -= int(desiredNumberOfShards)
					if deviation < 0 {
						// too few shards: increase sf
						sf = sf * (1.0+1.0/float64(iter))
					} else {
						// too much shards: decrease sf
						sf = sf * (1.0-1.0/float64(iter))
					}
				}
				for i, sc := range shardCandidates {
					shardCandidates[i] = t.NewShardDimension(sc.Column, int(float64(sc.NumPartitions) * sf))
				}
				// remove partitions
				for len(shardCandidates) > 0 && shardCandidates[len(shardCandidates)-1].NumPartitions <= 1 {
					shardCandidates = shardCandidates[:len(shardCandidates)-1]
				}
				// check if we should change partitioning schema already
				shouldChange := false
				if len(shardCandidates) != len(t.PDimensions) {
					shouldChange = true
				} else {
					totalShards1 := 1
					totalShards2 := 1
					for i, sc := range shardCandidates {
						if sc.Column != t.PDimensions[i].Column {
							shouldChange = true
						} else {
							totalShards1 *= sc.NumPartitions
							totalShards2 *= t.PDimensions[i].NumPartitions
						}
					}
					// deviation of >50% of shardsize
					if 2 * totalShards1 > 3 * totalShards2 || 2 * totalShards2 > 3 * totalShards1 {
						shouldChange = true
					}
				}

				// rebuild sharding schema
				if len(shardCandidates) > 0 && shouldChange {
					fmt.Println("repartitioning", t.Name, "by", shardCandidates)
					// TODO
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
}

