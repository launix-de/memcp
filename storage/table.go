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
import "sync/atomic"
import "time"
import "errors"
import "strings"
import "strconv"
import "encoding/json"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/go-mysqlstack/sqldb"

type dataset []scm.Scmer
type column struct {
	Name              string
	Typ               string
	Typdimensions     []int     // type dimensions for DECIMAL(10,3) and VARCHAR(5)
	Computor          scm.Scmer `json:"-"`          // TODO: marshaljson -> serialize
	ComputorInputCols []string  `json:",omitempty"` // input cols for computor (persisted in schema)
	PartitioningScore int       // count this up to increase the chance of partitioning for this column
	AutoIncrement     bool
	Default           scm.Scmer
	OnUpdate          scm.Scmer
	AllowNull         bool
	IsTemp            bool // columns with IsTemp may be removed without consequences
	Collation         string
	Comment           string
	sanitizer         func(scm.Scmer) scm.Scmer
	lastAccessed      int64 // atomic; UnixNano timestamp for CacheManager LRU (lock-free via sync/atomic)

	// ORC fields — non-empty OrcSortCols signals this is an ordered-reduce computed column.
	// The column value is produced by a scan_order pass rather than per-row computation.
	OrcSortCols   []string  `json:",omitempty"` // ORDER BY column names (partition cols first, then order cols)
	OrcSortDirs   []bool    `json:",omitempty"` // false=ASC, true=DESC, one per OrcSortCol
	OrcMapCols    []string  `json:",omitempty"` // additional input columns passed to OrcMapFn
	OrcMapFn      scm.Scmer // (lambda ($set mapcols...) ...) — passes data to reduceFn
	OrcReduceFn   scm.Scmer // (lambda (acc mapped) ...) — accumulates and writes via $set
	OrcReduceInit scm.Scmer // initial accumulator value (neutral element)
}

// OrcFirstSortCol returns the first sort column name.
func (c *column) OrcFirstSortCol() string {
	return c.OrcSortCols[0]
}

// OrcFirstSortDesc returns true if the first sort direction is DESC.
func (c *column) OrcFirstSortDesc() bool {
	return len(c.OrcSortDirs) > 0 && c.OrcSortDirs[0]
}

// PersistencyMode controls the durability and persistence behaviour of a table.
//
// DATA SAFETY CONTRACT — each mode's guarantees and risks:
//
//	Safe (default):
//	  Full durability including power-outage protection. Every committed write
//	  is recorded in a WAL and the log is fsync'd to disk at transaction end.
//	  On the next startup the WAL is replayed; no committed write is ever lost,
//	  even if power is cut mid-write.
//	  Use for all production tables that must survive crashes AND power outages.
//
//	Logged:
//	  Process-crash durability. The WAL is written but NOT fsync'd — the OS
//	  page cache may buffer the write. Data is safe against a process crash or
//	  clean shutdown, but a sudden power loss before the OS flushes the buffer
//	  can lose the last uncommitted WAL tail.
//	  Use when crash safety matters but the extra fsync latency of Safe is
//	  unacceptable and hardware power protection (UPS, battery-backed RAID) is
//	  provided externally.
//
//	Sloppy:
//	  Data is stored on disk as compressed columnar files, but there is NO
//	  write-ahead log. In-memory deltas (inserts/deletes since the last
//	  rebuild/flush) are LOST on unclean shutdown. Only the data that has
//	  been flushed via rebuild() to the main columnar storage is durable.
//	  Use only for data that can be reconstructed or where some loss is
//	  acceptable (e.g. caches, staging tables).
//
//	Memory:
//	  Non-persistent. ALL data is held in RAM only and is LOST on any
//	  shutdown or restart. This is intentional and by design.
//	  ⚠️  ALTER TABLE … ENGINE=memory on a persisted table PERMANENTLY
//	  DELETES all on-disk files with no possibility of recovery.
//	  Never use for production data that must survive a restart.
//
// ENGINE TRANSITION DATA SAFETY:
//   - persisted (Safe/Logged/Sloppy) → Memory:  IRREVERSIBLE disk deletion.
//     All column files and logs are removed immediately. Ensure data is
//     backed up or no longer needed before issuing ALTER TABLE ENGINE=memory.
//   - Safe/Logged → Sloppy:  WAL is closed and deleted. Future writes lose
//     crash/power-outage safety going forward.
//   - Memory → persisted:  Safe — current in-RAM data is serialised to disk.
//   - Sloppy → Safe:  WAL opened with fsync; future writes are fully durable.
//   - Sloppy → Logged:  WAL opened without fsync; future writes survive crashes.
//
// CLEANUP RULES (must not be violated):
//   - shardCleanup() (LRU eviction) MUST NEVER delete persistent data.
//     It only releases in-memory representations; disk files remain intact.
//   - RemoveFromDisk() is ONLY called from DropTable (explicit user DDL) or
//     from transitionShardEngine when moving to Memory (explicit ALTER TABLE).
//   - Trigger callbacks registered via AfterDropTable MUST NOT delete data
//     in unrelated tables, except through explicitly declared CASCADE foreign
//     key policies.
type PersistencyMode uint8

const (
	Safe   PersistencyMode = 0
	Logged                 = 1
	Sloppy                 = 3
	Memory                 = 2
	Cache  PersistencyMode = 4
)

// parsePersistencyMode converts an engine name string to a PersistencyMode.
func parsePersistencyMode(engine string) PersistencyMode {
	switch engine {
	case "memory":
		return Memory
	case "cache":
		return Cache
	case "sloppy":
		return Sloppy
	case "logged":
		return Logged
	case "safe":
		return Safe
	default:
		panic("unknown engine: " + engine)
	}
}

type ShardMode int

const (
	ShardModeFree      ShardMode = 0 // use Shards (unpartitioned)
	ShardModePartition ShardMode = 1 // use PShards (partitioned)
)

func (m *ShardMode) MarshalJSON() ([]byte, error) {
	if *m == ShardModePartition {
		return []byte("\"partition\""), nil
	}
	return []byte("\"freeshard\""), nil
}

func (m *ShardMode) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	if str == "partition" {
		*m = ShardModePartition
		return nil
	}
	if str == "freeshard" {
		*m = ShardModeFree
		return nil
	}
	return errors.New("unknown shard mode: " + str)
}

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
	Columns         []*column
	Unique          []uniqueKey          // unique keys
	Foreign         []foreignKey         // foreign keys
	Triggers        []TriggerDescription // triggers on this table
	PersistencyMode PersistencyMode      /* 0 = safe (default), 1 = sloppy, 2 = memory */
	mu              sync.Mutex           // schema/sharding lock
	uniquelock      sync.Mutex           // unique insert lock
	// LOCK TABLES: variable-based lock that is cheap for scans to check but
	// expensive to acquire (drains shard readers first via waitTableLock).
	// Software contract:
	//   - tableLockOwner/tableLockWrite describe the currently granted user lock.
	//   - tableLockNext/tableLockServe serialize LOCK TABLES acquisition FIFO per table,
	//     so many concurrent waiters (e.g. cron workers) do not stampede on unlock.
	//   - Regular scans/writes only consult waitTableLock; they never participate in
	//     the FIFO queue and are released together once the owner unlocks.
	// tableLockOwner and tableLockWrite are read from every shard goroutine on
	// every scan; isolate them on their own cache line to prevent false sharing
	// with Auto_increment (written on every INSERT).
	tableLockMu    sync.Mutex                       // guards cond waits + acquisition
	tableLockOnce  sync.Once                        // lazy-inits tableLockCond
	tableLockCond  *sync.Cond                       // broadcast on unlock
	tableLockNext  uint64                           // next FIFO ticket for LOCK TABLES acquisition
	tableLockServe uint64                           // currently served FIFO ticket
	tableLockOwner atomic.Pointer[scm.SessionState] // nil = no lock; points to owning *SessionState
	tableLockWrite atomic.Bool                      // true = WRITE lock, false = READ lock
	_              [39]byte                         // pad to cache-line boundary
	Auto_increment uint64                           // this dosen't scale over multiple cores, so assign auto_increment ranges to each shard
	Collation      string
	Charset        string
	Comment        string

	// index column frequency: used to sort equality columns by frequency
	// so that the most-queried columns come first, maximizing prefix overlap.
	colFreq   map[string]int64
	colFreqMu sync.Mutex
	// mutationMu serializes concurrent mutation scan statements (e.g. UPDATE with
	// $update callbacks) on this table. Ownership is tracked per goroutine to
	// allow reentrant scans within the same call stack.
	mutationMu     sync.Mutex
	mutationOwnMu  sync.Mutex
	mutationOwners map[uint64]uint32

	// orcMu serializes ORC recomputes: only one full scan_order pass at a time per table.
	orcMu          sync.Mutex
	orcRecomputing int32 // atomic: >0 means an ORC recompute is in progress (skip re-entry in GetValue)

	lastAccessed uint64 // atomic; UnixNano timestamp for CacheManager LRU of TempKeytable

	// ddlMu is the table-local schema contract:
	//   - Lock(): column/trigger/ORC metadata on this table may change
	//   - RLock(): rebuild/repartition may assume table-local schema stability
	// This keeps long maintenance work local to one table instead of blocking
	// unrelated tables behind the database-global schemalock.
	ddlMu sync.RWMutex

	// storage: ShardMode controls which shard set is the read/write target
	ShardMode   ShardMode
	rebuilding  bool
	shardModeMu sync.RWMutex    // protects ShardMode reads in iterateShards vs Phase F flip
	Shards      []*storageShard // unordered shards (used when ShardMode == ShardModeFree)
	PShards     []*storageShard // partitioned shards according to PDimensions (used when ShardMode == ShardModePartition)
	PDimensions []shardDimension

	// repartitionActive is the table-local contract for live repartition:
	// while true, concurrent DML must dual-write into both shard sets.
	// It is protected by t.mu and is claimed before a repartition starts,
	// not only by background rebuilds but also by direct/manual partitiontable calls.
	repartitionActive bool
	repartitionOnce   sync.Once
	repartitionCond   *sync.Cond
}

func (t *table) enterMutationOwner() {
	goid := currentGoroutineID()
	if goid == 0 {
		return
	}
	t.mutationOwnMu.Lock()
	if t.mutationOwners == nil {
		t.mutationOwners = make(map[uint64]uint32)
	}
	t.mutationOwners[goid]++
	t.mutationOwnMu.Unlock()
}

func (t *table) exitMutationOwner() {
	goid := currentGoroutineID()
	if goid == 0 {
		return
	}
	t.mutationOwnMu.Lock()
	if d := t.mutationOwners[goid]; d <= 1 {
		delete(t.mutationOwners, goid)
	} else {
		t.mutationOwners[goid] = d - 1
	}
	t.mutationOwnMu.Unlock()
}

func (t *table) hasMutationOwner() bool {
	goid := currentGoroutineID()
	if goid == 0 {
		return false
	}
	t.mutationOwnMu.Lock()
	defer t.mutationOwnMu.Unlock()
	return t.mutationOwners[goid] > 0
}

// bumpColFreq increments the query frequency counter for a column.
func (t *table) bumpColFreq(col string) {
	t.colFreqMu.Lock()
	if t.colFreq == nil {
		t.colFreq = make(map[string]int64)
	}
	t.colFreq[col]++
	t.colFreqMu.Unlock()
}

// getColFreq returns the query frequency counter for a column.
func (t *table) getColFreq(col string) int64 {
	t.colFreqMu.Lock()
	defer t.colFreqMu.Unlock()
	if t.colFreq == nil {
		return 0
	}
	return t.colFreq[col]
}

// ActiveShards returns the shard set that is currently authoritative for reads/writes.
func (t *table) ActiveShards() []*storageShard {
	if t.ShardMode == ShardModePartition {
		return t.PShards
	}
	return t.Shards
}

// maintenanceShards returns every shard set that must observe derived-column
// maintenance immediately:
//   - the currently authoritative shard set
//   - any staging shard set of an in-progress repartition
//   - per-shard rebuild successors reachable via shard.next
//
// Contract:
// mutations may dual-write rows into these shadow shards before they become
// authoritative. Invalidation of ORC/computed proxies must therefore touch the
// same maintenance set, otherwise a later publish can surface stale caches.
func (t *table) maintenanceShards() []*storageShard {
	active := t.ActiveShards()
	seen := make(map[*storageShard]struct{}, len(active)*2)
	result := make([]*storageShard, 0, len(active)*2)
	appendShard := func(s *storageShard) {
		if s == nil {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		result = append(result, s)
	}
	for _, s := range active {
		appendShard(s)
		appendShard(s.loadNext())
	}
	if t.repartitionActive {
		if t.ShardMode == ShardModePartition {
			for _, s := range t.Shards {
				appendShard(s)
			}
		} else {
			for _, s := range t.PShards {
				appendShard(s)
			}
		}
	}
	return result
}

func (t *table) getRepartitionCond() *sync.Cond {
	t.repartitionOnce.Do(func() {
		t.repartitionCond = sync.NewCond(&t.mu)
	})
	return t.repartitionCond
}

// beginManualRepartition serializes direct partitiontable-triggered repartitions.
// Contract:
//   - callers wait until any already-running repartition on this table finishes
//   - if the table became partitioned while waiting, no new immediate repartition starts
//   - otherwise repartitionActive is claimed under t.mu before returning
func (t *table) beginManualRepartition() bool {
	cond := t.getRepartitionCond()
	t.mu.Lock()
	for t.repartitionActive {
		cond.Wait()
	}
	if t.ShardMode != ShardModeFree {
		t.mu.Unlock()
		return false
	}
	t.repartitionActive = true
	t.mu.Unlock()
	return true
}

// isEphemeralQueryTable identifies query-local scratch tables (keytables,
// prejoins, scalar helper tables, etc.). These are dot-prefixed and created
// with non-durable engines so they can be recreated from the query plan after
// eviction. Global rebuild/repartition must skip them; otherwise a background
// "(rebuild)" can rebuild a live query's scratch tables and interfere with the
// in-flight scan.
//
// Contract:
//   - durable internal tables such as ".blobs" are NOT ephemeral and must
//     continue to participate in rebuild/persistence.
//   - query-local temp tables are dot-prefixed AND use a non-durable engine.
func (t *table) isEphemeralQueryTable() bool {
	if !strings.HasPrefix(t.Name, ".") {
		return false
	}
	switch t.PersistencyMode {
	case Sloppy, Memory, Cache:
		return true
	default:
		return false
	}
}

// schemaWriteDurable reports whether schema.json must be flushed with Sync for
// DDL on this table. Safe tables keep the full durability contract; other
// engines still write schema.json synchronously, but without fsync.
func (t *table) schemaWriteDurable() bool {
	return t.PersistencyMode == Safe
}

func (t *table) finishSchemaMutationLocked() {
	t.schema.saveLockedWithDurabilityAndUnlock(t.schemaWriteDurable())
}

// isHiddenFromShowTables implements the SQL metadata contract for internal
// helper tables. Dot-prefixed tables are planner/storage internals and must not
// leak through SHOW TABLES / INFORMATION_SCHEMA.TABLES listings, otherwise a
// metadata query ends up materializing live keytables/prejoins/materialized
// helper relations from unrelated requests.
func (t *table) isHiddenFromShowTables() bool {
	return strings.HasPrefix(t.Name, ".")
}

func (t *table) getTableLockCond() *sync.Cond {
	t.tableLockOnce.Do(func() {
		t.tableLockCond = sync.NewCond(&t.tableLockMu)
	})
	return t.tableLockCond
}

// waitTableLock blocks until the table lock is compatible with the caller's intent.
// isWrite=true means the caller wants to write (blocked by ANY lock from another session).
// isWrite=false means the caller wants to read (blocked only by WRITE lock from another session).
// Sets State to "Waiting for table lock" while blocking.
// Panics if the owning session tries to write while holding a READ lock (MySQL semantics).
func (t *table) waitTableLock(ss *scm.SessionState, isWrite bool) {
	cond := t.getTableLockCond()
	if ss != nil {
		ss.SetState("Waiting for table lock")
	}
	var errMsg string
	t.tableLockMu.Lock()
	for {
		owner := t.tableLockOwner.Load()
		if owner == nil {
			break
		}
		if owner == ss {
			if !isWrite || t.tableLockWrite.Load() {
				break // owner can always read; owner can write under WRITE lock
			}
			// Owner trying to write while holding a READ lock: MySQL returns an error.
			errMsg = "Can't write to table '" + t.Name + "' while it has a READ lock"
			break
		}
		if !isWrite && !t.tableLockWrite.Load() {
			break // READ lock doesn't block reads from other sessions
		}
		cond.Wait()
	}
	t.tableLockMu.Unlock()
	if ss != nil {
		ss.SetState("")
	}
	if errMsg != "" {
		panic(errMsg)
	}
}

// unlockTable releases the table lock and wakes all waiters.
func (t *table) unlockTable() {
	t.tableLockOwner.Store(nil)
	cond := t.getTableLockCond()
	t.tableLockMu.Lock()
	t.tableLockServe++
	cond.Broadcast()
	t.tableLockMu.Unlock()
}

func (t *table) Count() (result uint) {
	for _, s := range t.ActiveShards() {
		result += uint(s.Count())
	}
	return
}

// CountEstimate returns a quick estimate of the number of items by taking the
// first shard's count and multiplying it by the number of shards. This avoids
// iterating all shards and can be used as an inputCount estimate for planning.
func (t *table) CountEstimate() (result uint) {
	shards := t.ActiveShards()
	if len(shards) == 0 {
		return 0
	}
	// Ensure shard is loaded and main_count initialized
	unlock := shards[0].GetRead()
	defer unlock()
	c := shards[0].Count()
	return uint(c) * uint(len(shards))
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
	for _, c := range t.Columns {
		for _, col := range cols {
			if col == c.Name {
				c.PartitioningScore++
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
	if *m == Cache {
		return []byte("\"cache\""), nil
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
	if str == "cache" {
		*m = Cache
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
		// Determine Key type for this column
		keyType := ""
		for _, uk := range t.Unique {
			for _, col := range uk.Cols {
				if col == v.Name {
					if uk.Id == "PRIMARY" {
						keyType = "PRI"
					} else if keyType == "" {
						keyType = "UNI"
					}
				}
			}
		}
		result[i] = v.Show(keyType)
	}
	return scm.NewSlice(result)
}

func (c *column) Show(keyType string) scm.Scmer {
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
		scm.NewString("Key"), scm.NewString(keyType),
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
				s := v.String()
				// MySQL-compatible: parse leading integer part of string (e.g. "2026-03-11" -> 2026)
				i := 0
				if i < len(s) && (s[i] == '-' || s[i] == '+') {
					i++
				}
				for i < len(s) && s[i] >= '0' && s[i] <= '9' {
					i++
				}
				if i == 0 || (i == 1 && (s[0] == '-' || s[0] == '+')) {
					return scm.NewInt(0)
				}
				if n, err := strconv.ParseInt(s[:i], 10, 64); err == nil {
					return scm.NewInt(n)
				}
				return scm.NewInt(0)
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

// createColumnLocked mutates table metadata while the database schemalock is
// held. It does not persist schema.json yet; callers must follow up with a
// single saveLockedAndUnlock once the whole DDL mutation is complete.
func (t *table) createColumnLocked(name string, typ string, typdimensions []int, extrainfo []scm.Scmer) (*column, bool) {
	for _, c := range t.Columns {
		if c.Name == name {
			return nil, false // column already exists
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
		case "filtercols", "filter":
			// handled by createcolumn builtin, not a column property
		case "sortcols", "sortdirs", "mapcols", "mapfn", "reducefn", "reduceinit":
			// ORC params handled by createcolumn builtin after CreateColumn
		default:
			panic("unknown column attribute: " + key)
		}
	}
	c.UpdateSanitizer()
	cp := &c
	t.Columns = append(t.Columns, cp)
	for _, s := range t.Shards {
		if s == nil {
			continue
		}
		// mutate shard column map under shard lock to avoid races with readers
		s.mu.Lock()
		s.columns[name] = new(StorageSparse)
		s.mu.Unlock()
	}
	for _, s := range t.PShards {
		if s == nil {
			continue
		}
		// mutate shard column map under shard lock to avoid races with readers
		s.mu.Lock()
		s.columns[name] = new(StorageSparse)
		s.mu.Unlock()
	}
	return cp, true
}

func (t *table) registerTempColumn(cp *column) {
	// register temp column with CacheManager AFTER releasing schemalock
	// to avoid deadlock: AddItem → run() → evict → cleanup → TryLock(schemalock)
	tbl := t
	colName := cp.Name
	GlobalCache.AddItem(cp, 0, TypeTempColumn, func(ptr any, freedByType *[numEvictableTypes]int64) bool {
		// We're inside the CacheManager goroutine. MUST NOT call GlobalCache.Remove.
		// Use TryLock to avoid blocking the CacheManager if schema is locked.
		if !tbl.schema.schemalock.TryLock() {
			return false // busy, retry later
		}
		for i, col := range tbl.Columns {
			if col.Name == colName {
				tbl.Columns = append(tbl.Columns[:i], tbl.Columns[i+1:]...)
				for _, s := range tbl.Shards {
					s.mu.Lock()
					delete(s.columns, colName)
					s.mu.Unlock()
				}
				for _, s := range tbl.PShards {
					s.mu.Lock()
					delete(s.columns, colName)
					s.mu.Unlock()
				}
				tbl.schema.schemalock.Unlock()
				return true
			}
		}
		tbl.schema.schemalock.Unlock()
		return true
	}, tempColumnLastUsed, nil)
}

func (t *table) createColumnDDLLocked(name string, typ string, typdimensions []int, extrainfo []scm.Scmer) bool {
	// one early out without schemalock (especially for computed columns)
	for _, c := range t.Columns {
		if c.Name == name {
			return false // column already exists
		}
	}

	t.schema.schemalock.Lock()
	cp, ok := t.createColumnLocked(name, typ, typdimensions, extrainfo)
	if !ok {
		t.schema.schemalock.Unlock()
		return false
	}
	t.finishSchemaMutationLocked()
	if cp.IsTemp {
		t.registerTempColumn(cp)
	}
	return true
}

func (t *table) CreateColumn(name string, typ string, typdimensions []int, extrainfo []scm.Scmer) bool {
	t.ddlMu.Lock()
	defer t.ddlMu.Unlock()
	return t.createColumnDDLLocked(name, typ, typdimensions, extrainfo)
}

func (t *table) dropColumnDDLLocked(name string) bool {
	t.schema.schemalock.Lock()
	var removedCol *column
	for i, c := range t.Columns {
		if c.Name == name {
			removedCol = c
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
			// remove cache invalidation triggers from source tables
			t.removeComputeTriggers(name)

			t.finishSchemaMutationLocked()
			// Fire lifecycle hooks after unlock so dependents (e.g. prejoin caches)
			// can invalidate without lock-ordering cycles.
			t.ExecuteTableLifecycleTriggers(AfterDropColumn)
			// deregister temp column AFTER releasing schemalock
			if removedCol.IsTemp {
				GlobalCache.Remove(removedCol)
			}
			return true
		}
	}
	t.schema.schemalock.Unlock()
	panic("drop column does not exist: " + t.Name + "." + name)
}

func (t *table) DropColumn(name string) bool {
	t.ddlMu.Lock()
	defer t.ddlMu.Unlock()
	return t.dropColumnDDLLocked(name)
}

func (t *table) DropColumnIfExists(name string) bool {
	t.schema.schemalock.Lock()
	for _, c := range t.Columns {
		if c.Name == name {
			t.schema.schemalock.Unlock()
			return t.DropColumn(name)
		}
	}
	t.schema.schemalock.Unlock()
	return false
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
	values = t.sanitizeInsertRows(columns, values, isIgnore)
	if len(values) == 0 {
		return 0
	}

	if t.ShardMode == ShardModeFree { // unpartitioned sharding
		// Helper to get or create a shard with capacity for n rows
		getShardWithCapacity := func(n uint) *storageShard {
			t.mu.Lock()
			if t.Shards == nil {
				// Repartition completed while we waited for the lock.
				t.mu.Unlock()
				return nil
			}
			shard := t.Shards[len(t.Shards)-1]
			if uint(shard.Count())+n > Settings.ShardSize {
				// Current shard would overflow, create new one
				go func(i int, s *storageShard) {
					defer func() {
						if r := recover(); r != nil {
							fmt.Println("error: shard rebuild failed for", t.schema.Name+".", t.Name, "shard", i, ":", r)
						}
					}()
					rebuilt := s.rebuild(false)
					t.mu.Lock()
					t.Shards[i] = rebuilt
					t.mu.Unlock()
					t.schema.save()
				}(len(t.Shards)-1, shard)
				shard = NewShard(t)
				fmt.Println("started new shard for table", t.Name)
				t.Shards = append(t.Shards, shard)
				if t.PersistencyMode == Cache && !strings.HasPrefix(t.Name, ".") {
					GlobalCache.AddItem(shard, 0, TypeCacheEntry, cacheShardCleanup, shardLastUsed, nil)
				}
			}
			t.mu.Unlock()
			return shard
		}

		// For bulk inserts larger than ShardSize, split into chunks
		chunkSize := int(Settings.ShardSize)
		repartitioned := false
		for start := 0; start < len(values); start += chunkSize {
			end := start + chunkSize
			if end > len(values) {
				end = len(values)
			}
			chunk := values[start:end]

			shard := getShardWithCapacity(uint(len(chunk)))
			if shard == nil {
				// Repartition completed: re-insert remaining values via partition path
				values = values[start:]
				repartitioned = true
				break
			}
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
						panic(sqldb.NewSQLError1(1062, "23000", "Duplicate entry in table %s: %s", t.Name, errmsg))
					}
				}, 0)
			} else {
				// physically insert (no unique constraints)
				shard.Insert(columns, chunk, false, onFirstInsertId, isIgnore)
				result += len(chunk)
			}
			release()
		}
		if !repartitioned {
			goto insertDone
		}
		// fall through to partition path with remaining values
	}
	{
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
						panic(sqldb.NewSQLError1(1062, "23000", "Duplicate entry in table %s: %s", t.Name, errmsg))
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

insertDone:
	// Dual-write: forward inserts to the secondary shard set during repartition.
	// The secondary insert bypasses unique checks (already handled above),
	// triggers, and auto-increment (already assigned in primary insert).
	if t.repartitionActive {
		t.dualWriteInsert(columns, values)
	}

	return result
}

func (t *table) sanitizeInsertRows(columns []string, values [][]scm.Scmer, isIgnore bool) [][]scm.Scmer {
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
		return filtered
	}

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
	return values
}

// dualWriteInsert routes rows into the secondary shard set (the one not selected
// by ShardMode) during an active repartition. Called only when repartitionActive
// is true. Rows are inserted with alreadyLocked=false and no unique/trigger processing.
func (t *table) dualWriteInsert(columns []string, values [][]scm.Scmer) {
	if t.ShardMode == ShardModeFree {
		// Primary is Shards, secondary is PShards
		if t.PShards == nil {
			return
		}
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
		last_i := 0
		var last_shard *storageShard
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
				rel := last_shard.GetExclusive()
				last_shard.Insert(columns, values[last_i:i], false, nil, false)
				rel()
				last_i = i
			}
			last_shard = shard
		}
		if last_i < len(values) && last_shard != nil {
			rel := last_shard.GetExclusive()
			last_shard.Insert(columns, values[last_i:], false, nil, false)
			rel()
		}
	} else {
		// Primary is PShards, secondary is Shards
		if t.Shards == nil {
			return
		}
		// Route to the last shard in the free shard list
		t.mu.Lock()
		shard := t.Shards[len(t.Shards)-1]
		t.mu.Unlock()
		rel := shard.GetExclusive()
		shard.Insert(columns, values, false, nil, false)
		rel()
	}
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
	{
		key := make([]scm.Scmer, len(uniq.Cols))
		keyIdx := make([]int, len(uniq.Cols))
		skipConstraint := false // true if a key col is auto-assigned (auto-increment/default) and not in columns
		for i, col := range uniq.Cols {
			found := false
			for j, col2 := range columns {
				if col == col2 {
					keyIdx[i] = j
					found = true
				}
			}
			if !found {
				// Column not provided by the caller — check if it's auto-assigned.
				// If so, the auto-increment/default mechanism guarantees a unique value,
				// so there is no point checking (and no safe value to check against).
				for _, tc := range t.Columns {
					if tc.Name == col && (tc.AutoIncrement || !tc.Default.IsNil()) {
						skipConstraint = true
						break
					}
				}
			}
		}
		if skipConstraint {
			success(values)
			return
		}

		shardlist := t.ActiveShards()
		// During repartition drain (ShardMode already flipped to Partition but
		// repartitionActive still true), in-flight scans on old shards call
		// ProcessUniqueCollision. We must check old Shards because the deletion
		// from the UPDATE is only in the old shard, not yet in PShards.
		if t.repartitionActive && t.ShardMode == ShardModePartition && t.Shards != nil {
			shardlist = t.Shards
		}
		allowPruning := false // if we can prune the shardlist
		pruningMap := make([]int, len(uniq.Cols))
		pruningVals := make([]scm.Scmer, len(uniq.Cols))
		if t.ShardMode == ShardModePartition && !t.repartitionActive {
			// partitioning
			allowPruning = true
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
		// Always register panic recovery so both outer (t.uniquelock) and inner
		// (shard.uniquelock) lock releases are handled on panic.
		defer func() {
			if r := recover(); r != nil {
				if uniquelockHeld {
					lock.Unlock()
				}
				panic(r) // re-panic after releasing lock
			}
		}()
		if (!allowPruning || len(t.Unique) > 1) && idx == 0 {
			lock.Lock()
			uniquelockHeld = true
		}

		last_j := 0
		for j, row := range values {
			shardlist2 := shardlist
			skipUniqueCheck := false
			for i, colidx := range keyIdx {
				key[i] = row[colidx]
				if !mergeNull && key[i].IsNil() {
					skipUniqueCheck = true
				}
			}
			if skipUniqueCheck {
				goto nextrow
			}
			if allowPruning {
				for j, xidx := range pruningMap {
					pruningVals[j] = row[keyIdx[xidx]]
				}
				// only one shard to visit for unique check
				shardlist2 = []*storageShard{shardlist[computeShardIndex(t.PDimensions, pruningVals)]} // (TODO: array pruning)
				if len(t.Unique) == 1 {
					lock = &shardlist2[0].uniquelock
					lock.Lock()
					uniquelockHeld = true
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
						isVisible = !s.deletions.Get(uint(uid))
					}
				}
				if isVisible {
					// found a unique collision
					if j != last_j {
						// If the inner check panics (unique violation in a later constraint),
						// it will have released our lock via its own defer chain. Clear
						// uniquelockHeld so our outer defer does not double-unlock.
						var flushPanic interface{}
						func() {
							defer func() { flushPanic = recover() }()
							t.ProcessUniqueCollision(columns, values[last_j:j], mergeNull, success, onCollisionCols, failure, idx+1) // flush
						}()
						if flushPanic != nil {
							// Only deeper unique-check levels (idx+1 < len(t.Unique)) can
							// have released our lock. The success callback level does not.
							if idx+1 < len(t.Unique) {
								uniquelockHeld = false
							}
							panic(flushPanic)
						}
					}
					last_j = j + 1
					lock.Unlock()
					uniquelockHeld = false
					params := make([]scm.Scmer, len(onCollisionCols))
					for i, p := range onCollisionCols {
						if p == "$update" {
							params[i] = scm.NewFunc(s.UpdateFunction(uid, true, false))
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
				if len(t.Unique) == 1 && !skipUniqueCheck {
					uniquelockHeld = false
					lock.Unlock()
				}
			}
		}
		if len(values) != last_j {
			// Same as above: clear uniquelockHeld if inner call releases the lock via panic.
			var flushPanic interface{}
			func() {
				defer func() { flushPanic = recover() }()
				t.ProcessUniqueCollision(columns, values[last_j:], mergeNull, success, onCollisionCols, failure, idx+1) // flush the rest
			}()
			if flushPanic != nil {
				// Same rationale as above: only inner unique-check levels may have
				// unlocked our lock before panicking.
				if idx+1 < len(t.Unique) {
					uniquelockHeld = false
				}
				panic(flushPanic)
			}
		}
		if (!allowPruning || len(t.Unique) > 1) && idx == 0 {
			lock.Unlock()
		}
	}
}

func tempColumnLastUsed(ptr any) time.Time {
	c := ptr.(*column)
	ts := atomic.LoadInt64(&c.lastAccessed)
	if ts == 0 {
		return time.Time{} // never accessed
	}
	return time.Unix(0, ts)
}
