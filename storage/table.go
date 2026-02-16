/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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
import "math"
import "sync"
import "errors"
import "strings"
import "strconv"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

type dataset []scm.Scmer
type column struct {
	Name              string
	Typ               string
	Typdimensions     []int     // type dimensions for DECIMAL(10,3) and VARCHAR(5)
	Computor          scm.Scmer `json:"-"` // TODO: marshaljson -> serialize
	PartitioningScore int       // count this up to increase the chance of partitioning for this column
	AutoIncrement     bool
	Default           scm.Scmer
	OnUpdate          scm.Scmer
	AllowNull         bool
	IsTemp            bool // columns with IsTemp may be removed without consequences
	Collation         string
	Comment           string
	sanitizer         func(scm.Scmer) scm.Scmer
	// TODO: LRU statistics for computed columns
}
type PersistencyMode uint8

const (
	Safe   PersistencyMode = 0
	Logged                 = 1
	Sloppy                 = 3
	Memory                 = 2
)

type uniqueKey struct {
	Id   string
	Cols []string
}
type foreignKeyMode uint8

const (
	RESTRICT foreignKeyMode = 0
	CASCADE                 = 1
	SETNULL                 = 2
)

type foreignKey struct {
	Id         string
	Tbl1       string
	Cols1      []string
	Tbl2       string
	Cols2      []string
	Updatemode foreignKeyMode
	Deletemode foreignKeyMode
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
	schema          *database
	Name            string
	Columns         []column
	Unique          []uniqueKey          // unique keys
	Foreign         []foreignKey         // foreign keys
	Triggers        []TriggerDescription // triggers on this table
	PersistencyMode PersistencyMode      /* 0 = safe (default), 1 = sloppy, 2 = memory */
	mu              sync.Mutex           // schema/sharding lock
	uniquelock      sync.Mutex           // unique insert lock
	Auto_increment  uint64               // this dosen't scale over multiple cores, so assign auto_increment ranges to each shard
	Collation       string
	Charset         string
	Comment         string

	// storage: if both arrays Shards and PShards are present, Shards is the single point of truth
	Shards      []*storageShard // unordered shards; as long as this value is not nil, use shards instead of pshards
	PShards     []*storageShard // partitioned shards according to PDimensions
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

// CountEstimate returns a quick estimate of the number of items by taking the
// first shard's count and multiplying it by the number of shards. This avoids
// iterating all shards and can be used as an inputCount estimate for planning.
func (t *table) CountEstimate() (result uint) {
	shards := t.Shards
	if shards == nil {
		shards = t.PShards
	}
	if len(shards) == 0 {
		return 0
	}
	// Ensure shard is loaded and main_count initialized
	unlock := shards[0].GetRead()
	defer unlock()
	c := shards[0].Count()
	return c * uint(len(shards))
}

/* Implement NonLockingReadMap */
func (t table) GetKey() string {
	return t.Name
}

func (t table) ComputeSize() uint {
	var size uint = 10*8 + 32*uint(len(t.Columns))
	for _, s := range t.Shards {
		size += s.ComputeSize()
	}
	for _, s := range t.PShards {
		size += s.ComputeSize()
	}
	return size
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
	if *m == Memory {
		return []byte("\"memory\""), nil
	}
	if *m == Sloppy {
		return []byte("\"sloppy\""), nil
	}
	if *m == Logged {
		return []byte("\"logged\""), nil
	}
	if *m == Safe {
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
	if str == "memory" {
		*m = Memory
		return nil
	}
	if str == "sloppy" {
		*m = Sloppy
		return nil
	}
	if str == "logged" {
		*m = Logged
		return nil
	}
	if str == "safe" {
		*m = Safe
		return nil
	}
	return errors.New("unknown persistency mode: " + str)
}

func getForeignKeyMode(val scm.Scmer) foreignKeyMode {
	if val.IsNil() {
		return RESTRICT
	}
	switch scm.String(val) {
	case "restrict":
		return RESTRICT
	case "cascade":
		return CASCADE
	case "set null":
		return SETNULL
	default:
		panic("unknown update mode: " + scm.String(val))
	}
}

func (t *table) ShowColumns() scm.Scmer {
	if t == nil {
		return scm.NewNil()
	}
	result := make([]scm.Scmer, len(t.Columns))
	for i, v := range t.Columns {
		result[i] = v.Show()
	}
	return scm.NewSlice(result)
}

func (c *column) Show() scm.Scmer {
	dims := make([]scm.Scmer, len(c.Typdimensions))
	for i, v := range c.Typdimensions {
		dims[i] = scm.NewInt(int64(v))
	}
	typ := c.Typ
	if len(c.Typdimensions) > 0 {
		var b strings.Builder
		b.WriteString(c.Typ)
		b.WriteByte('(')
		for i, v := range c.Typdimensions {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.Itoa(v))
		}
		b.WriteByte(')')
		typ = b.String()
	}
	extra := ""
	if c.AutoIncrement {
		extra = "auto_increment"
	}
	return scm.NewSlice([]scm.Scmer{
		scm.NewString("Field"), scm.NewString(c.Name),
		scm.NewString("Type"), scm.NewString(typ),
		scm.NewString("Collation"), scm.NewString(c.Collation),
		scm.NewString("RawType"), scm.NewString(c.Typ),
		scm.NewString("Dimensions"), scm.NewSlice(dims),
		scm.NewString("Null"), scm.NewBool(c.AllowNull),
		scm.NewString("Default"), c.Default,
		scm.NewString("Extra"), scm.NewString(extra),
		scm.NewString("Privileges"), scm.NewString("select,insert,update,references"),
		scm.NewString("Comment"), scm.NewString(c.Comment),
	})
}

func (c *column) UpdateSanitizer() {
	typ := strings.ToUpper(c.Typ)
	allowNull := c.AllowNull
	name := c.Name
	var inner func(scm.Scmer) scm.Scmer
	switch typ {
	case "INT", "INTEGER", "BIGINT", "SMALLINT", "MEDIUMINT", "TINYINT":
		inner = func(v scm.Scmer) scm.Scmer {
			tag := v.GetTag()
			if tag == scm.TagString || tag == scm.TagSymbol {
				// try numeric string
				if _, err := strconv.ParseInt(v.String(), 10, 64); err != nil {
					if _, err2 := strconv.ParseFloat(v.String(), 64); err2 != nil {
						panic("cannot convert string to INT for column " + name + ": " + v.String())
					}
				}
			}
			return scm.NewInt(int64(scm.ToInt(v)))
		}
	case "FLOAT", "DOUBLE", "REAL":
		inner = func(v scm.Scmer) scm.Scmer {
			tag := v.GetTag()
			if tag == scm.TagString || tag == scm.TagSymbol {
				if _, err := strconv.ParseFloat(v.String(), 64); err != nil {
					panic("cannot convert string to FLOAT for column " + name + ": " + v.String())
				}
			}
			return scm.NewFloat(v.Float())
		}
	case "DECIMAL", "NUMERIC":
		dims := c.Typdimensions
		inner = func(v scm.Scmer) scm.Scmer {
			tag := v.GetTag()
			if tag == scm.TagString || tag == scm.TagSymbol {
				if _, err := strconv.ParseFloat(v.String(), 64); err != nil {
					panic("cannot convert string to DECIMAL for column " + name + ": " + v.String())
				}
			}
			f := v.Float()
			if len(dims) >= 2 && dims[1] == 0 {
				// DECIMAL(n,0) → round to integer
				if f >= 0 {
					return scm.NewInt(int64(f + 0.5))
				}
				return scm.NewInt(int64(f - 0.5))
			}
			if len(dims) >= 2 && dims[1] > 0 {
				// DECIMAL(n,s) → round to s decimal places
				mult := math.Pow(10, float64(dims[1]))
				return scm.NewFloat(math.Round(f*mult) / mult)
			}
			return scm.NewFloat(f)
		}
	case "DATE", "DATETIME", "TIMESTAMP":
		inner = func(v scm.Scmer) scm.Scmer {
			if v.GetTag() == scm.TagDate {
				return v
			}
			if v.IsInt() {
				return scm.NewDate(v.Int())
			}
			if v.IsFloat() {
				return scm.NewDate(int64(v.Float()))
			}
			if v.IsString() {
				if ts, ok := scm.ParseDateString(v.String()); ok {
					return scm.NewDate(ts)
				}
				panic("cannot parse date string for column " + name + ": " + v.String())
			}
			return scm.NewDate(v.Int())
		}
	case "VARCHAR", "CHAR":
		dims := c.Typdimensions
		if len(dims) >= 1 && dims[0] > 0 {
			maxLen := dims[0]
			inner = func(v scm.Scmer) scm.Scmer {
				s := scm.String(v)
				if len(s) > maxLen {
					s = s[:maxLen]
				}
				return scm.NewString(s)
			}
		}
	}
	// wrap with NOT NULL check
	if !allowNull && inner != nil {
		base := inner
		c.sanitizer = func(v scm.Scmer) scm.Scmer {
			if v.IsNil() {
				panic("column " + name + " cannot be NULL")
			}
			return base(v)
		}
	} else if !allowNull {
		c.sanitizer = func(v scm.Scmer) scm.Scmer {
			if v.IsNil() {
				panic("column " + name + " cannot be NULL")
			}
			return v
		}
	} else if inner != nil {
		c.sanitizer = func(v scm.Scmer) scm.Scmer {
			if v.IsNil() {
				return v
			}
			return inner(v)
		}
	}
}

func (c *column) Alter(key string, val scm.Scmer) scm.Scmer {
	switch key {
	case "type":
		c.Typ = scm.String(val)
		c.UpdateSanitizer()
		return scm.NewString(c.Typ)
	case "dimensions":
		// expect val to be a list of numbers
		if val.IsNil() {
			c.Typdimensions = nil
			return scm.NewNil()
		}
		if l, ok := scmerSlice(val); ok {
			dims := make([]int, len(l))
			for i, v := range l {
				dims[i] = scm.ToInt(v)
			}
			c.Typdimensions = dims
			c.UpdateSanitizer()
			return scm.NewSlice(l)
		}
		panic("invalid dimensions value for alter column")
	case "default":
		c.Default = val
		return c.Default
	case "null":
		c.AllowNull = scm.ToBool(val)
		c.UpdateSanitizer()
		return scm.NewBool(c.AllowNull)
	case "temp":
		c.IsTemp = scm.ToBool(val)
		return scm.NewBool(c.IsTemp)
	case "collation":
		c.Collation = scm.String(val)
		return scm.NewString(c.Collation)
	case "comment":
		c.Comment = scm.String(val)
		return scm.NewString(c.Comment)
	default:
		panic("unimplemented alter column operation: " + key)
	}
}

func (d dataset) Get(key string) (scm.Scmer, bool) {
	for i := 0; i < len(d); i += 2 {
		if scm.String(d[i]) == key {
			return d[i+1], true
		}
	}
	return scm.NewNil(), false
}

func (d dataset) GetI(key string) (scm.Scmer, bool) { // case insensitive
	for i := 0; i < len(d); i += 2 {
		if strings.EqualFold(scm.String(d[i]), key) {
			return d[i+1], true
		}
	}
	return scm.NewNil(), false
}

func (t *table) CreateColumn(name string, typ string, typdimensions []int, extrainfo []scm.Scmer) bool {
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
		key := scm.String(extrainfo[i])
		switch key {
		case "primary":
			// append unique key
			t.Unique = append(t.Unique, uniqueKey{"PRIMARY", []string{name}})
		case "unique":
			// append unique key
			t.Unique = append(t.Unique, uniqueKey{name, []string{name}})
		case "auto_increment":
			c.AutoIncrement = scm.ToBool(extrainfo[i+1])
		case "null":
			c.AllowNull = scm.ToBool(extrainfo[i+1])
		case "default":
			c.Default = extrainfo[i+1]
		case "update":
			c.OnUpdate = extrainfo[i+1]
		case "comment":
			c.Comment = scm.String(extrainfo[i+1])
		case "collate":
			c.Collation = scm.String(extrainfo[i+1])
		case "temp":
			c.IsTemp = scm.ToBool(extrainfo[i+1])
		default:
			panic("unknown column attribute: " + key)
		}
	}
	c.UpdateSanitizer()
	t.Columns = append(t.Columns, c)
	for _, s := range t.Shards {
		// mutate shard column map under shard lock to avoid races with readers
		s.mu.Lock()
		s.columns[name] = new(StorageSparse)
		s.mu.Unlock()
	}
	for _, s := range t.PShards {
		// mutate shard column map under shard lock to avoid races with readers
		s.mu.Lock()
		s.columns[name] = new(StorageSparse)
		s.mu.Unlock()
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
				s.mu.Lock()
				delete(s.columns, name)
				s.mu.Unlock()
			}
			for _, s := range t.PShards {
				s.mu.Lock()
				delete(s.columns, name)
				s.mu.Unlock()
			}

			t.schema.save()
			t.schema.schemalock.Unlock()
			return true
		}
	}
	t.schema.schemalock.Unlock()
	panic("drop column does not exist: " + t.Name + "." + name)
}

func (t *table) Insert(columns []string, values [][]scm.Scmer, onCollisionCols []string, onCollision scm.Scmer, mergeNull bool, onFirstInsertId func(int64)) int {
	result := 0
	isIgnore := !onCollision.IsNil() // INSERT IGNORE or ON DUPLICATE KEY UPDATE
	// FK checks are enforced via auto-generated system triggers (see createforeignkey)

	// check NOT NULL for omitted columns (not skippable by IGNORE)
	for _, colDesc := range t.Columns {
		if !colDesc.AllowNull && !colDesc.Default.IsNil() {
			continue // has a default value
		}
		if !colDesc.AllowNull && !colDesc.AutoIncrement {
			found := false
			for _, col := range columns {
				if col == colDesc.Name {
					found = true
					break
				}
			}
			if !found {
				panic("column " + colDesc.Name + " cannot be NULL")
			}
		}
	}

	// sanitize values (per-row recovery for INSERT IGNORE)
	if isIgnore {
		filtered := values[:0]
		for _, row := range values {
			ok := true
			func() {
				defer func() {
					if r := recover(); r != nil {
						ok = false
					}
				}()
				for i, col := range columns {
					for _, colDesc := range t.Columns {
						if col == colDesc.Name && colDesc.sanitizer != nil {
							if i < len(row) {
								row[i] = colDesc.sanitizer(row[i])
							}
						}
					}
				}
			}()
			if ok {
				filtered = append(filtered, row)
			}
		}
		values = filtered
		if len(values) == 0 {
			return 0
		}
	} else {
		for i, col := range columns {
			for _, colDesc := range t.Columns {
				if col == colDesc.Name && colDesc.sanitizer != nil {
					for _, row := range values {
						if i < len(row) {
							row[i] = colDesc.sanitizer(row[i])
						}
					}
				}
			}
		}
	}

	if t.Shards != nil { // unpartitioned sharding
		// Helper to get or create a shard with capacity for n rows
		getShardWithCapacity := func(n uint) *storageShard {
			t.mu.Lock()
			shard := t.Shards[len(t.Shards)-1]
			if shard.Count()+n > Settings.ShardSize {
				// Current shard would overflow, create new one
				go func(i int) {
					defer func() {
						if r := recover(); r != nil {
							fmt.Println("error: shard rebuild failed for", t.schema.Name+".", t.Name, "shard", i, ":", r)
						}
					}()
					s := t.Shards[i]
					t.Shards[i] = s.rebuild(false)
					t.schema.save()
				}(len(t.Shards) - 1)
				shard = NewShard(t)
				fmt.Println("started new shard for table", t.Name)
				t.Shards = append(t.Shards, shard)
			}
			t.mu.Unlock()
			return shard
		}

		// For bulk inserts larger than ShardSize, split into chunks
		chunkSize := int(Settings.ShardSize)
		for start := 0; start < len(values); start += chunkSize {
			end := start + chunkSize
			if end > len(values) {
				end = len(values)
			}
			chunk := values[start:end]

			shard := getShardWithCapacity(uint(len(chunk)))
			release := shard.GetExclusive()

			// check unique constraints in a thread safe manner
			if len(t.Unique) > 0 {
				t.ProcessUniqueCollision(columns, chunk, mergeNull, func(chunk [][]scm.Scmer) {
					shard.Insert(columns, chunk, false, onFirstInsertId, isIgnore)
					result += len(chunk)
				}, onCollisionCols, func(errmsg string, data []scm.Scmer) {
					if !onCollision.IsNil() {
						// Evaluate onCollision and add to affected rows per MySQL semantics
						// - inserted rows already counted above
						// - on duplicate: count 2 if changed, 1 if no-op
						ret := scm.Apply(onCollision, data...)
						switch {
						case ret.IsBool():
							if ret.Bool() {
								result += 2
							} else {
								result++
							}
						case ret.IsInt():
							result += int(ret.Int())
						case ret.IsFloat():
							result += int(ret.Float())
						default:
							// Fallback: consider as one affected row
							result++
						}
					} else {
						panic("Unique key constraint violated in table " + t.Name + ": " + errmsg)
					}
				}, 0)
			} else {
				// physically insert (no unique constraints)
				shard.Insert(columns, chunk, false, onFirstInsertId, isIgnore)
				result += len(chunk)
			}
			release()
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
			// ensure shard is loaded and writable for inserts
			rel := s.GetExclusive()
			defer rel()
			// check unique constraints in a thread safe manner
			if len(t.Unique) > 0 {
				// this function will do the locking for us
				t.ProcessUniqueCollision(columns, values, mergeNull, func(values [][]scm.Scmer) {
					// physically insert
					s.Insert(columns, values, false, onFirstInsertId, isIgnore)
					result += len(values)
				}, onCollisionCols, func(errmsg string, data []scm.Scmer) {
					if !onCollision.IsNil() {
						// Evaluate onCollision and add to affected rows per MySQL semantics
						ret := scm.Apply(onCollision, data...)
						switch {
						case ret.IsBool():
							if ret.Bool() {
								result += 2
							} else {
								result++
							}
						case ret.IsInt():
							result += int(ret.Int())
						case ret.IsFloat():
							result += int(ret.Float())
						default:
							result++
						}
					} else {
						panic("Unique key constraint violated in table " + t.Name + ": " + errmsg)
					}
				}, 0)
			} else {
				// physically insert (parallel)
				s.Insert(columns, values, false, onFirstInsertId, isIgnore)
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
					shardcols[j] = scm.NewNil()
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
			for j, dim := range t.PDimensions {
				hasPruningCol := false
				for i, col := range uniq.Cols {
					if dim.Column == col {
						hasPruningCol = true // we found the uniq column in our partitioning schema
						pruningMap[j] = i
					}
				}
				if !hasPruningCol {
					// a column different from the unique key is part of our partitioning schema -> we cannot prune (TODO: array pruning)
					allowPruning = false // all unique columns must be present in the partitioning schema, otherwise a unique collision might hide in pruned shards
				}
			}
		}
		// TODO: only shard-local lock if allowPruning

		var lock *sync.Mutex
		lock = &t.uniquelock
		uniquelockHeld := false
		if (!allowPruning || len(t.Unique) > 1) && idx == 0 {
			lock.Lock()
			uniquelockHeld = true
			defer func() {
				if r := recover(); r != nil {
					if uniquelockHeld {
						lock.Unlock()
					}
					panic(r) // re-panic after releasing lock
				}
			}()
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
				shardlist2 = []*storageShard{shardlist[computeShardIndex(t.PDimensions, pruningVals)]} // (TODO: array pruning)
				if len(t.Unique) == 1 {
					lock = &shardlist2[0].uniquelock
					lock.Lock()
				}
			}
			for _, s := range shardlist2 {
				// ensure shard is loaded for read during unique check
				r := s.GetRead()
				uid, present := s.GetRecordidForUnique(uniq.Cols, key)
				isVisible := false
				if present {
					if currentTx := CurrentTx(); currentTx != nil && currentTx.Mode == TxACID {
						isVisible = currentTx.IsVisible(s, uid)
					} else {
						isVisible = !s.deletions.Get(uid)
					}
				}
				if isVisible {
					// found a unique collision
					if j != last_j {
						t.ProcessUniqueCollision(columns, values[last_j:j], mergeNull, success, onCollisionCols, failure, idx+1) // flush
					}
					last_j = j + 1
					lock.Unlock()
					uniquelockHeld = false
					params := make([]scm.Scmer, len(onCollisionCols))
					for i, p := range onCollisionCols {
						if p == "$update" {
							params[i] = scm.NewFunc(s.UpdateFunction(uid, true))
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
					func() {
						defer func() {
							if r := recover(); r != nil {
								// Re-lock before re-panicking so the outer
								// defer at idx==0 can safely release it.
								lock.Lock()
								uniquelockHeld = true
								panic(r)
							}
						}()
						failure(uniq.Id, params) // notify about failure
					}()
					lock.Lock()
					uniquelockHeld = true
					r()
					goto nextrow
				}
				r()
			}
		nextrow:
			if allowPruning {
				if len(t.Unique) == 1 {
					lock.Unlock()
				}
			}
		}
		if len(values) != last_j {
			t.ProcessUniqueCollision(columns, values[last_j:], mergeNull, success, onCollisionCols, failure, idx+1) // flush the rest
		}
		if (!allowPruning || len(t.Unique) > 1) && idx == 0 {
			lock.Unlock()
		}
	} else {
		if idx == 0 {
			t.uniquelock.Unlock() // TODO: instead of uniquelock, in case of sharding, use a shard local lock
		}
		// build scan for unique check
		cols := make([]scm.Scmer, len(uniq.Cols))
		colidx := make([]int, len(uniq.Cols))
		for i, c := range uniq.Cols {
			cols[i] = scm.NewSymbol(c)
			for j, col := range columns { // find uniq columns
				if c == col {
					colidx[i] = j
				}
			}
		}
		conditionBody := make([]scm.Scmer, len(uniq.Cols)+1)
		conditionBody[0] = scm.NewSymbol("and")
		last_j := 0
		for j, row := range values {
			for i, colidx := range colidx {
				value := row[colidx]
				if !mergeNull && value.IsNil() {
					conditionBody[i+1] = scm.NewBool(false) // NULL can be there multiple times
				} else {
					conditionBody[i+1] = scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewAny(scm.NthLocalVar(i)), value})
				}
			}
			condition := scm.Proc{Params: scm.NewSlice(cols), Body: scm.NewSlice(conditionBody), En: &scm.Globalenv, NumVars: len(uniq.Cols)}
			updatefn := t.scan(
				uniq.Cols,
				scm.NewAny(condition),
				onCollisionCols,
				scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
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
					return scm.NewBool(true) // feedback that there was a collision
				}),
				scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] }),
				scm.NewNil(),
				scm.NewNil(),
				false,
			)
			if !updatefn.IsNil() {
				// found a unique collision: flush the successing items and skip this one
				if j != last_j {
					t.ProcessUniqueCollision(columns, values[last_j:j], mergeNull, success, onCollisionCols, failure, idx+1) // flush
				}
				last_j = j + 1
			}
		}
		if len(values) != last_j {
			t.ProcessUniqueCollision(columns, values[last_j:], mergeNull, success, onCollisionCols, failure, idx+1) // flush the rest
		}
		if idx == 0 {
			t.uniquelock.Unlock()
		}
	}
}
