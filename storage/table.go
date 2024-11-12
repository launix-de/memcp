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
import "strings"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

type dataset []scm.Scmer
type column struct {
	Name string
	Typ string
	Typdimensions []int // type dimensions for DECIMAL(10,3) and VARCHAR(5)
	Computor scm.Scmer `json:"-"` // TODO: marshaljson -> serialize
	PartitioningScore int // count this up to increase the chance of partitioning for this column
	AutoIncrement bool
	Default scm.Scmer
	AllowNull bool // TODO: respect this
	IsTemp bool // columns with IsTemp may be removed without consequences
	Collation string
	Comment string
	// TODO: LRU statistics for computed columns
}
type PersistencyMode uint8
const (
	Safe PersistencyMode = 0
	Logged = 1
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
	Collation string
	Charset string

	// storage: if both arrays Shards and PShards are present, Shards is the single point of truth
	Shards []*storageShard // unordered shards; as long as this value is not nil, use shards instead of pshards
	PShards []*storageShard // partitioned shards according to PDimensions
	PDimensions []shardDimension
	// TODO: move rows from Shards to PShards according to PDimensions
}

func (t *table) Count() (result uint) {
	shards := t.Shards
	if shards == nil {
		shards = t.PShards
	}
	for _, s := range shards {
		result += s.Count()
	}
	return
}

/* Implement NonLockingReadMap */
func (t table) GetKey() string {
	return t.Name
}

// increases PartitioningScore for a set of columns
func (t *table) AddPartitioningScore(cols []string) {
	// we don't sync because we want to be fast; we ignore write-after-write hazards
	for i, c := range t.Columns {
		for _, col := range cols {
			if col == c.Name {
				t.Columns[i].PartitioningScore = c.PartitioningScore + 1
			}
		}
	}
}

func (m *PersistencyMode) MarshalJSON() ([]byte, error) {
	if (*m == Memory) {
		return []byte("\"memory\""), nil
	}
	if (*m == Sloppy) {
		return []byte("\"sloppy\""), nil
	}
	if (*m == Logged) {
		return []byte("\"logged\""), nil
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
	if (str == "logged") {
		*m = Logged
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
	typ := c.Typ
	if len(dims) > 0 {
		typ += "("
		for i, v := range c.Typdimensions {
			if i > 0 {
				typ = typ + ","
			}
			typ = typ + scm.String(v)
		}
		typ += ")"
	}
	extra := ""
	if c.AutoIncrement {
		extra = "auto_increment"
	}
	return []scm.Scmer{"Field", c.Name, "Type", typ, "Collation", c.Collation, "RawType", c.Typ, "Dimensions", dims, "Null", c.AllowNull, "Default", c.Default, "Extra", extra, "Privileges", "select,insert,update,references", "Comment", c.Comment}
}

func (c *column) Alter(key string, val scm.Scmer) scm.Scmer {
	switch (key) {
		case "default":
			c.Default = val
			return c.Default
		case "null":
			c.AllowNull = scm.ToBool(val)
			return c.AllowNull
		case "temp":
			c.IsTemp = scm.ToBool(val)
			return c.IsTemp
		case "collation":
			c.Collation = scm.String(val)
			return c.Collation
		case "comment":
			c.Comment = scm.String(val)
			return c.Comment
		default:
			panic("unimplemented alter column operation: " + key)
	}
}

func (d dataset) Get(key string) (scm.Scmer, bool) {
	for i := 0; i < len(d); i += 2 {
		if d[i] == key {
			return d[i+1], true
		}
	}
	return nil, false
}

func (d dataset) GetI(key string) (scm.Scmer, bool) { // case insensitive
	for i := 0; i < len(d); i += 2 {
		if strings.EqualFold(scm.String(d[i]), key) {
			return d[i+1], true
		}
	}
	return nil, false
}

func (t *table) CreateColumn(name string, typ string, typdimensions[] int, extrainfo []scm.Scmer) bool {
	// one early out without schemalock (especially for computed columns)
	for _, c := range t.Columns {
		if c.Name == name {
			return false // column already exists
		}
	}

	t.schema.schemalock.Lock()
	defer t.schema.schemalock.Unlock()

	for _, c := range t.Columns {
		if c.Name == name {
			return false // column already exists
		}
	}
	
	var c column
	c.Name = name
	c.Typ = typ
	c.Typdimensions = typdimensions
	c.Collation = "utf8mb4"
	c.AllowNull = true
	for i := 0; i < len(extrainfo); i += 2 {
		if extrainfo[i] == "primary" {
			// append unique key
			t.Unique = append(t.Unique, uniqueKey{"PRIMARY", []string{name}})
		} else if extrainfo[i] == "unique" {
			// append unique key
			t.Unique = append(t.Unique, uniqueKey{name, []string{name}})
		} else if extrainfo[i] == "auto_increment" {
			c.AutoIncrement = scm.ToBool(extrainfo[i+1])
		} else if extrainfo[i] == "null" {
			c.AllowNull = scm.ToBool(extrainfo[i+1])
		} else if extrainfo[i] == "default" {
			c.Default = extrainfo[i+1]
		} else if extrainfo[i] == "comment" {
			c.Comment = scm.String(extrainfo[i+1])
		} else if extrainfo[i] == "collate" {
			c.Collation = scm.String(extrainfo[i+1])
		} else if extrainfo[i] == "temp" {
			c.IsTemp = scm.ToBool(extrainfo[i+1])
		} else {
			panic("unknown column attribute: " + scm.String(extrainfo[i]))
		}
	}
	t.Columns = append(t.Columns, c)
	for _, s := range t.Shards {
		s.columns[name] = new (StorageSparse)
	}
	for _, s := range t.PShards {
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
			for _, s := range t.PShards {
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

func (t *table) Insert(columns []string, values [][]scm.Scmer, onCollisionCols []string, onCollision scm.Scmer, mergeNull bool) int {
	result := 0
	// TODO: check foreign keys (new value of column must be present in referenced table)

	if t.Shards != nil { // unpartitioned sharding
		shard := t.Shards[len(t.Shards)-1]
		// load balance: if bucket is full, create new one; if bucket is busy (trylock), try another one
		if shard.Count() >= Settings.ShardSize {
			t.mu.Lock()
			// reload shard after lock to avoid race conditions
			shard = t.Shards[len(t.Shards)-1]
			if shard.Count() >= Settings.ShardSize {
				go func(i int) {
					// rebuild full shards in background
					s := t.Shards[i]
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

		// check unique constraints in a thread safe manner
		if len(t.Unique) > 0 {
			t.ProcessUniqueCollision(columns, values, mergeNull, func (values [][]scm.Scmer) {
				// physically insert
				shard.Insert(columns, values, false)
				result += len(values)
			}, onCollisionCols, func (errmsg string, data []scm.Scmer) {
				if onCollision != nil {
					scm.Apply(onCollision, data...)
				} else {
					panic("Unique key constraint violated in table "+t.Name+": " + errmsg)
				}
			}, 0)
		} else {
			// physically insert (parallel)
			shard.Insert(columns, values, false)
			result += len(values)
		}
	} else {
		// partitions
		// TODO: check which shards are involved; a sharding dimension column must be present in ALL unique keys, otherwise we cannot prune
		dims := t.PDimensions
		shardcols := make([]scm.Scmer, len(dims))
		translatable := make([]int, len(dims))
		for i, cd := range dims {
			for j, col := range columns {
				if cd.Column == col {
					translatable[i] = j
				}
			}
		}

		checkUniqueForShard := func(s *storageShard, values [][]scm.Scmer) {
			// check unique constraints in a thread safe manner
			if len(t.Unique) > 0 {
				// this function will do the locking for us
				t.ProcessUniqueCollision(columns, values, mergeNull, func (values [][]scm.Scmer) {
					// physically insert
					s.Insert(columns, values, false)
					result += len(values)
				}, onCollisionCols, func (errmsg string, data []scm.Scmer) {
					if onCollision != nil {
						scm.Apply(onCollision, data...)
					} else {
						panic("Unique key constraint violated in table "+t.Name+": " + errmsg)
					}
				}, 0)
			} else {
				// physically insert (parallel)
				s.Insert(columns, values, false)
				result += len(values)
			}
		}

		last_i := 0
		var last_shard *storageShard = nil
		for i := 0; i < len(values); i++ {
			for j, colidx := range translatable {
				if colidx < len(values[i]) {
					shardcols[j] = values[i][colidx]
				} else {
					shardcols[j] = nil
				}
			}
			shard := t.PShards[computeShardIndex(dims, shardcols)]
			if i > 0 && shard != last_shard {
				checkUniqueForShard(last_shard, values[last_i:i]) // shard has changed: bulk insert all items that belong to this shard
				last_i = i
			}
			last_shard = shard
		}
		if last_i < len(values) { // bulk insert the rest
			checkUniqueForShard(last_shard, values[last_i:])
		}
	}

	// TODO: Trigger after insert
	return result
}

/*
	checks a number of datasets for unique collisions.
	For each block of datasets that pass, success is called.
	For each single unique collision that fails, failure is called.
*/
func (t *table) ProcessUniqueCollision(columns []string, values [][]scm.Scmer, mergeNull bool, success func([][]scm.Scmer), onCollisionCols []string, failure func(string, []scm.Scmer), idx int) {
	// check for duplicates
	if idx >= len(t.Unique) {
		success(values) // we finally made it, these values have passed all unique checks
		return
	}
	uniq := t.Unique[idx]
	t.AddPartitioningScore(uniq.Cols) // increases partitioning score, so partitioning is improved
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

		shardlist := t.Shards
		allowPruning := false // if we can prune the shardlist
		pruningMap := make([]int, len(uniq.Cols))
		pruningVals := make([]scm.Scmer, len(uniq.Cols))
		if shardlist == nil {
			// partitioning
			allowPruning = true
			shardlist = t.PShards
			for i, col := range uniq.Cols {
				hasPruningCol := false
				for j, dim := range t.PDimensions {
					if dim.Column == col {
						hasPruningCol = true // we found the uniq column in our partitioning schema
						pruningMap[j] = i
					}
				}
				if !hasPruningCol {
					allowPruning = false // all unique columns must be present in the partitioning schema, otherwise a unique collision might hide in pruned shards
				}
			}
		}
		// TODO: only shard-local lock if allowPruning

		var lock *sync.Mutex
		lock = &t.uniquelock
		if (!allowPruning || len(t.Unique) > 1) && idx == 0 {
			lock.Lock()
		}

		last_j := 0
		for j, row := range values {
			for i, colidx := range keyIdx {
				key[i] = row[colidx]
			}
			shardlist2 := shardlist
			if allowPruning {
				for j, xidx := range pruningMap {
					pruningVals[j] = row[keyIdx[xidx]]
				}
				// only one shard to visit for unique check
				shardlist2 = []*storageShard{shardlist[computeShardIndex(t.PDimensions, pruningVals)]}
				if len(t.Unique) == 1 {
					lock = &shardlist2[0].uniquelock
					lock.Lock()
				}
			}
			for _, s := range shardlist2 {
				uid, present := s.GetRecordidForUnique(uniq.Cols, key)
				if present && !s.deletions.Get(uid) {
					// found a unique collision
					if j != last_j {
						t.ProcessUniqueCollision(columns, values[last_j:j], mergeNull, success, onCollisionCols, failure, idx + 1) // flush
					}
					last_j = j+1
					lock.Unlock()
					params := make([]scm.Scmer, len(onCollisionCols))
					for i, p := range onCollisionCols {
						if p == "$update" {
							params[i] = s.UpdateFunction(uid, true)
						} else if len(p) >= 4 && p[:4] == "NEW." {
							for j, c := range columns {
								if p[4:] == c {
									params[i] = row[j]
								}
							}
						} else {
							params[i] = s.ColumnReader(p)(uid)
						}
					}
					failure(uniq.Id, params) // notify about failure
					lock.Lock()
					goto nextrow
				}
			}
			nextrow:
			if allowPruning {
				if len(t.Unique) == 1 {
					lock.Unlock()
				}
			}
		}
		if len(values) != last_j {
			t.ProcessUniqueCollision(columns, values[last_j:], mergeNull, success, onCollisionCols, failure, idx + 1) // flush the rest
		}
		if (!allowPruning || len(t.Unique) > 1) && idx == 0 {
			lock.Unlock() // TODO: instead of uniquelock, in case of sharding, use a shard local lock
		}
	} else {
		if idx == 0 {
			t.uniquelock.Unlock() // TODO: instead of uniquelock, in case of sharding, use a shard local lock
		}
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
		last_j := 0
		for j, row := range values {
			for i, colidx := range colidx {
				value := row[colidx]
				if !mergeNull && value == nil {
					conditionBody[i + 1] = false // NULL can be there multiple times
				} else {
					conditionBody[i + 1] = []scm.Scmer{scm.Symbol("equal??"), scm.NthLocalVar(i), value}
				}
			}
			condition := scm.Proc {cols, conditionBody, &scm.Globalenv, len(uniq.Cols)}
			updatefn := t.scan(uniq.Cols, condition, onCollisionCols, func (args ...scm.Scmer) scm.Scmer {
				t.uniquelock.Unlock()
				for i, p := range onCollisionCols {
					if len(p) >= 4 && p[:4] == "NEW." {
						for j, c := range columns {
							if p[4:] == c {
								args[i] = row[j]
							}
						}
					}
				}
				failure(uniq.Id, args) // call collision function
				t.uniquelock.Lock()
				return true // feedback that there was a collision
			}, func(a ...scm.Scmer) scm.Scmer {return a[1]}, nil, nil, false)
			if updatefn != nil {
				// found a unique collision: flush the successing items and skip this one
				if j != last_j {
					t.ProcessUniqueCollision(columns, values[last_j:j], mergeNull, success, onCollisionCols, failure, idx + 1) // flush
				}
				last_j = j+1
			}
		}
		if len(values) != last_j {
			t.ProcessUniqueCollision(columns, values[last_j:], mergeNull, success, onCollisionCols, failure, idx + 1) // flush the rest
		}
		if idx == 0 {
			t.uniquelock.Unlock()
		}
	}
}
