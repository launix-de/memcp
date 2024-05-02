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

import "fmt"
import "sync"
import "errors"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

type dataset []scm.Scmer
type column struct {
	Name string
	Typ string
	Typdimensions []int // type dimensions for DECIMAL(10,3) and VARCHAR(5)
	Extrainfo string // TODO: further diversify into NOT NULL, AUTOINCREMENT etc.
	Computor scm.Scmer `json:"-"` // TODO: marshaljson -> serialize
	PartitioningScore int // count this up to increase the chance of partitioning for this column
}
type PersistencyMode uint8
const (
	Safe PersistencyMode = 0
	Sloppy = 1
	Memory = 2
)
type uniqueKey struct {
	Id string
	Cols []string
}
type foreignKey struct {
	Id string
	Tbl1 string
	Cols1 []string
	Tbl2 string
	Cols2 []string
}
type shardDimension struct {
	Column string
	NumPartitions int
	Pivots []scm.Scmer
}
/*
unique keys:
	insert: check all columns of all unique keys (--> index scan!), if it is unique, deny insert
	update: check all columns of all unique keys (--> index scan!), if it is unique, deny deletion, deny insert
	delete: -

foreign keys:
	insert: I am tbl1 -> check all cols1 : do the values exist in tbl2.cols2? if not, deny insert
	update: I am tbl1 -> check all cols1 (new values): do the values exist in tbl2.cols2? if not, deny insert
	update: I am tbl2 -> check all cols2 (old values): do the values exist in tbl1.cols1? if so -> CASCADE an update in tbl1, SET NULL in tbl1 or RESTRICT
	delete: I am tbl2 -> check all cols2 (old values): do the values exist in tbl1.cols1? if so -> CASCADE a delete in tbl1, SET NULL in tbl1 or RESTRICT

*/
type table struct {
	schema *database
	Name string
	Columns []column
	Unique []uniqueKey // unique keys
	Foreign []foreignKey // foreign keys
	PersistencyMode PersistencyMode /* 0 = safe (default), 1 = sloppy, 2 = memory */
	mu sync.Mutex // schema/sharding lock
	uniquelock sync.Mutex // unique insert lock
	Auto_increment uint64 // this dosen't scale over multiple cores, so assign auto_increment ranges to each shard

	// storage
	Shards []*storageShard // unordered shards
	PShards []*storageShard // partitioned shards according to PDimensions
	PDimensions []shardDimension
	// TODO: data structure to per-column value-based shard map
	// every shard has a min-value and a max-value
	// problem: naive approach needs len(Shards)² complexity
	// solution: two shard lists: one sorted by min-value, one sorted by max-value; also every shard remembers its max-index in the min-list and min-index in max list
	// where x > value -> peek value from shard list, iterate shardlist upwards
	// where x = value -> only peek the range
	// TODO: move rows between shards when values get too widespread; triggered on equi-joins
	// the rebalance algorithm is run on a list of shards (mostly the range of shards on a equi-join; goal is to minimize the range to 1)
	// rebalance sorts all values of a column in an index (sharnr+recordid), then moves all items that would change shardid via delete+insert command
}

const max_shardsize = 65536 // dont overload the shards to get a responsive parallel full table scan

/* Implement NonLockingReadMap */
func (t table) GetKey() string {
	return t.Name
}

func (m *PersistencyMode) MarshalJSON() ([]byte, error) {
	if (*m == Memory) {
		return []byte("\"memory\""), nil
	}
	if (*m == Sloppy) {
		return []byte("\"sloppy\""), nil
	}
	if (*m == Safe) {
		return []byte("\"safe\""), nil
	}
	return nil, errors.New("unknown persistency mode")
}

func (m *PersistencyMode) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	if (str == "memory") {
		*m = Memory
		return nil
	}
	if (str == "sloppy") {
		*m = Sloppy
		return nil
	}
	if (str == "safe") {
		*m = Safe
		return nil
	}
	return errors.New("unknown persistency mode: " + str)
}

func (t *table) ShowColumns() scm.Scmer {
	if t == nil {
		return nil
	}
	result := make([]scm.Scmer, len(t.Columns))
	for i, v := range t.Columns {
		result[i] = v.Show()
	}
	return result
}

func (c *column) Show() scm.Scmer {
	dims := make([]scm.Scmer, len(c.Typdimensions))
	for i, v := range c.Typdimensions {
		dims[i] = v
	}
	return []scm.Scmer{"name", c.Name, "type", c.Typ, "dimensions", dims}
}

func (d dataset) Get(key string) scm.Scmer {
	for i := 0; i < len(d); i += 2 {
		if d[i] == key {
			return d[i+1]
		}
	}
	return nil
}

func (t *table) CreateColumn(name string, typ string, typdimensions[] int, extrainfo string) bool {
	t.schema.schemalock.Lock()
	defer t.schema.schemalock.Unlock()

	for _, c := range t.Columns {
		if c.Name == name {
			return false // column already exists
		}
	}
	
	t.Columns = append(t.Columns, column{name, typ, typdimensions, extrainfo, nil, 0})
	for _, s := range t.Shards {
		s.columns[name] = new (StorageSparse)
	}
	t.schema.save()
	return true
}

func (t *table) DropColumn(name string) bool {
	t.schema.schemalock.Lock()
	for i, c := range t.Columns {
		if c.Name == name {
			// found the column
			t.Columns = append(t.Columns[:i], t.Columns[i+1:]...) // remove from slice
			for _, s := range t.Shards {
				delete(s.columns, name)
			}

			t.schema.save()
			t.schema.schemalock.Unlock()
			return true
		}
	}
	t.schema.schemalock.Unlock()
	panic("drop column does not exist: " + t.Name + "." + name)
}

func (t *table) Insert(columns []string, values [][]scm.Scmer, ignoreexists bool, mergeNull bool) int {
	result := 0
	// load balance: if bucket is full, create new one; if bucket is busy (trylock), try another one
	shard := t.Shards[len(t.Shards)-1]
	if shard.Count() >= max_shardsize {
		t.mu.Lock()
		// reload shard after lock to avoid race conditions
		shard = t.Shards[len(t.Shards)-1]
		if shard.Count() >= max_shardsize {
			go func(i int) {
				// rebuild full shards in background
				s := t.Shards[i]
				s.RunOn()
				t.Shards[i] = s.rebuild(false)
				// write new uuids to disk
				t.schema.save()
			}(len(t.Shards)-1)
			shard = NewShard(t)
			fmt.Println("started new shard for table", t.Name)
			t.Shards = append(t.Shards, shard)
		}
		t.mu.Unlock()
	}

	// TODO: check foreign keys (new value of column must be present in referenced table)

	// check unique constraints in a thread safe manner
	if len(t.Unique) > 0 {
		t.uniquelock.Lock()
		uniq_collisions := t.GetUniqueCollisionFor(columns, values, mergeNull)
		// let the whole block fail if one item is unique
		last_i := 0
		for i, err := range uniq_collisions {
			if err != "" {
				if ignoreexists {
					shard.Insert(columns, values[last_i:i]) // insert all chunks from before
					last_i = i+1 // skip this one
				} else {
					t.uniquelock.Unlock()
					panic("Unique key constraint violated in table "+t.Name+": " + err)
				}
			}
		}
		// physically insert
		shard.Insert(columns, values[last_i:])
		t.uniquelock.Unlock()
	} else {
		// physically insert (parallel)
		shard.Insert(columns, values)
	}

	// TODO: Trigger after insert
	return result
}

// TODO: refactor to "has" (cols, dataset); returns the id of the unique key
func (t *table) GetUniqueCollisionFor(columns []string, values [][]scm.Scmer, mergeNull bool) []string {
	// check for duplicates
	result := make([]string, len(values))
	for _, uniq := range t.Unique {
		if len(uniq.Cols) <= 3 {
			// use hashmap
			key := make([]scm.Scmer, len(uniq.Cols))
			keyIdx := make([]int, len(uniq.Cols))
			for i, col := range uniq.Cols {
				for j, col2 := range columns {
					if col == col2 {
						keyIdx[i] = j
					}
				}
			}
			for j, row := range values {
				for i, colidx := range keyIdx {
					key[i] = row[colidx]
				}
				for _, s := range t.Shards {
					uid, present := s.GetRecordidForUnique(uniq.Cols, key)
					if present && !s.deletions.Get(uid) {
						result[j] = uniq.Id
						goto nextrow
					}
				}
				nextrow:
			}
		} else {
			// build scan for unique check
			cols := make([]scm.Scmer, len(uniq.Cols))
			colidx := make([]int, len(uniq.Cols))
			for i, c := range uniq.Cols {
				cols[i] = scm.Symbol(c)
				for j, col := range columns { // find uniq columns
					if c == col {
						colidx[i] = j
					}
				}
			}
			conditionBody := make([]scm.Scmer, len(uniq.Cols) + 1)
			conditionBody[0] = scm.Symbol("and")
			for j, row := range values {
				for i, colidx := range colidx {
					value := row[colidx]
					if !mergeNull && value == nil {
						conditionBody[i + 1] = false // NULL can be there multiple times
					} else {
						conditionBody[i + 1] = []scm.Scmer{scm.Symbol("equal?"), scm.NthLocalVar(i), value}
					}
				}
				condition := scm.Proc {cols, conditionBody, &scm.Globalenv, len(uniq.Cols)}
				if t.scan(uniq.Cols, condition, []string{}, scm.Proc{[]scm.Scmer{}, true, &scm.Globalenv, 0}, func(a ...scm.Scmer) scm.Scmer {return a[0].(bool) || a[1].(bool)}, false, nil) != false {
					result[j] = uniq.Id
				}
			}
		}
	}
	return result
}
