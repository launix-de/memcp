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

import "context"
import "fmt"
import "math"
import "sync"
import "time"
import "errors"
import "unsafe"
import "strconv"
import "strings"
import "unicode"
import "sync/atomic"
import "unicode/utf8"
import "encoding/json"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/go-mysqlstack/sqldb"

// TagTable is the custom Scmer tag for *table pointers.
const TagTable = 101

// NewTableScmer wraps a *table into a Scmer with TagTable.
func NewTableScmer(t *table) scm.Scmer {
	return scm.NewCustom(TagTable, unsafe.Pointer(t))
}

// TableFromScmer extracts a *table from a TagTable Scmer.
func TableFromScmer(s scm.Scmer) *table {
	if s.IsNil() {
		panic("table does not exist")
	}
	return (*table)(s.Custom(TagTable))
}

// String implements serialization for table pointers: (table "schema" "name")
func (t *table) String() string {
	return "(table \"" + t.schema.Name + "\" \"" + t.Name + "\")"
}

type dataset []scm.Scmer

// columnPlannerStatistics is an immutable rebuild-generation snapshot. The
// pointer is published atomically so query compilation never needs shard locks.
type columnPlannerStatistics struct {
	Confidence        float64
	Source            string
	NullCount         uint64
	NullFraction      float64
	AverageValueBytes float64
	MinEstimate       scm.Scmer
	MaxEstimate       scm.Scmer
}

// atomicPlannerRowEstimate persists the last rebuild estimate while retaining
// lock-free access. Its JSON methods use atomic operations too, so concurrent
// schema persistence cannot race query compilation.
type atomicPlannerRowEstimate struct {
	value   atomic.Uint64
	present atomic.Bool
}

func (estimate *atomicPlannerRowEstimate) MarshalJSON() ([]byte, error) {
	return json.Marshal(estimate.value.Load())
}

func (estimate *atomicPlannerRowEstimate) UnmarshalJSON(data []byte) error {
	var value uint64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	estimate.value.Store(value)
	estimate.present.Store(true)
	return nil
}

type column struct {
	Name              string
	Typ               string
	Typdimensions     []int     // type dimensions for DECIMAL(10,3) and VARCHAR(5)
	Computor          scm.Scmer `json:"-"`          // TODO: marshaljson -> serialize
	ComputorInputCols []string  `json:",omitempty"` // input cols for computor (persisted in schema)
	// Repeated planner-issued createcolumn calls must be able to detect that the
	// canonical temp column signature is unchanged and skip schema/trigger work.
	// filter cols are persisted because they are part of the canonical helper
	// definition; the filter expression itself is runtime-only like Computor.
	ComputorFilterCols []string  `json:",omitempty"`
	ComputorFilter     scm.Scmer `json:"-"`
	PartitioningScore  int       // count this up to increase the chance of partitioning for this column
	AutoIncrement      bool
	Default            scm.Scmer
	OnUpdate           scm.Scmer
	AllowNull          bool
	IsTemp             bool // columns with IsTemp may be removed without consequences
	Collation          string
	Comment            string
	sanitizer          func(scm.Scmer) scm.Scmer
	lastAccessed       int64 // atomic; UnixNano timestamp for CacheManager LRU (lock-free via sync/atomic)
	cacheUsers         int64 // atomic; -1 while/after CacheManager eviction, otherwise active trigger users

	// Statistics — updated at rebuild time, O(1) access for query planning.
	// DistinctEstimate is the sum of per-shard DistinctCount() (upper bound).
	// RowEstimate is the table-wide CountEstimate() at last rebuild.
	DistinctEstimate uint64                                  `json:"-"`
	RowEstimate      uint64                                  `json:"-"`
	PlannerStats     atomic.Pointer[columnPlannerStatistics] `json:"-"`

	// ORC fields — non-empty OrcSortCols signals this is an ordered-reduce computed column.
	// The column value is produced by a scan_order pass rather than per-row computation.
	OrcSortCols       []string  `json:",omitempty"` // ORDER BY column names (partition cols first, then order cols)
	OrcSortDirs       []bool    `json:",omitempty"` // false=ASC, true=DESC, one per OrcSortCol
	OrcPartitionCount int       `json:",omitempty"` // number of leading OrcSortCols that form the window partition
	OrcMapCols        []string  `json:",omitempty"` // additional input columns passed to OrcMapFn
	OrcMapFn          scm.Scmer // (lambda ($set mapcols...) ...) — passes data to reduceFn
	OrcReduceFn       scm.Scmer // (lambda (acc mapped) ...) — accumulates and writes via $set
	OrcReduceInit     scm.Scmer // initial accumulator value (neutral element)
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
//	  shutdown or restart. The schema is persisted synchronously. A closed
//	  oninit callback is part of that schema and repopulates the empty data on
//	  the first idempotent createtable guard after restart.
//	  ⚠️  ALTER TABLE … ENGINE=memory on a persisted table PERMANENTLY
//	  DELETES all on-disk files with no possibility of recovery.
//	  Never use for production data that must survive a restart.
//
//	Cache:
//	  Reconstructible RAM-only data managed by the global cache manager. Hidden
//	  query-helper tables use buffered schema creation: creation returns without
//	  schema I/O, and the table definition plus its closed oninit callback joins
//	  the next complete schema save or rebuild. On restart, the first idempotent
//	  createtable guard runs that callback before exposing the empty generation.
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

// tableShardTopology is immutable after publication. Readers load one pointer
// and can inspect the complete authoritative topology without taking t.mu.
type tableShardTopology struct {
	mode       ShardMode
	shards     []*storageShard
	dimensions []shardDimension

	// Operations and transactions drain independently. A rebuild request may be
	// running inside the transaction that touched the retiring generation, so it
	// can wait for operations but must defer physical cleanup until transactions
	// finish after the request returns.
	operations         atomic.Int64
	transactions       atomic.Int64
	retired            atomic.Bool
	operationsDrained  chan struct{}
	drained            chan struct{}
	operationDrainOnce sync.Once
	drainOnce          sync.Once
}

func (topology *tableShardTopology) acquireOperation() bool {
	topology.operations.Add(1)
	if topology.retired.Load() {
		topology.releaseOperation()
		return false
	}
	return true
}

func (topology *tableShardTopology) pinTransaction() {
	topology.transactions.Add(1)
}

func (topology *tableShardTopology) releaseOperation() {
	if topology.operations.Add(-1) == 0 && topology.retired.Load() {
		topology.closeDrains()
	}
}

func (topology *tableShardTopology) releaseTransaction() {
	if topology.transactions.Add(-1) == 0 && topology.retired.Load() {
		topology.closeDrains()
	}
}

func (topology *tableShardTopology) closeDrains() {
	if topology.operations.Load() == 0 {
		topology.operationDrainOnce.Do(func() { close(topology.operationsDrained) })
		if topology.transactions.Load() == 0 {
			topology.drainOnce.Do(func() { close(topology.drained) })
		}
	}
}

func (topology *tableShardTopology) retire() <-chan struct{} {
	topology.retired.Store(true)
	topology.closeDrains()
	return topology.drained
}

type repartitionSourceSet struct {
	shards []*storageShard
	set    map[*storageShard]struct{}
}

type translatedRecid struct {
	pshardIdx int
	newRecid  uint32
	inDelta   bool
}

type pendingSourceDelete struct {
	oldShard *storageShard
	oldRecid uint32
}

type tableStatisticsSnapshot struct {
	rowCount  int64
	sizeBytes int64
}

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
	OnInit          *scm.Scmer           `json:"oninit,omitempty"` // closed callback that repopulates data-empty engines after restart
	cacheUsers      int64                // atomic; -1 while/after CacheManager eviction, otherwise active trigger users
	// LOCK ORDER CONTRACT:
	//   1. db.schemalock
	//   2. t.ddlMu
	//   3. t.mu
	//   4. s.mu
	// Never upgrade while already holding a read lock, and never take a table
	// lock from inside a shard lock. Heavy rebuild/repartition work must
	// snapshot under the narrowest lock and then release it before scanning.
	//
	// t.mu protects table-local storage topology and long-lived maintenance
	// state: ShardMode, Shards/PShards, maintenanceKind and the dual-write
	// flags that coordinate table-level rebuild/repartition.
	mu         sync.Mutex
	uniquelock sync.Mutex // unique insert lock
	// LOCK TABLES: variable-based lock that is cheap for scans to check but
	// expensive to acquire (drains shard readers first via waitTableLock).
	// Software contract:
	//   - tableLockOwner describes the WRITE owner; tableLockState and
	//     tableLockReadOwners describe concurrently granted READ locks.
	//   - tableLockNext/tableLockServe serialize LOCK TABLES acquisition FIFO per table,
	//     so many concurrent waiters (e.g. cron workers) do not stampede on unlock.
	//   - Regular scans/writes only consult waitTableLock; they never participate in
	//     the FIFO queue and are released together once the owner unlocks.
	//   - Mutations wait for tableLockState before taking a shard lock and recheck
	//     after taking it. They never wait for a table lock while holding a shard
	//     lock; cache snapshots hold their table READ lock while entering shards.
	// tableLockState is read from every shard goroutine on
	// every scan; isolate them on their own cache line to prevent false sharing
	// with Auto_increment (written on every INSERT).
	tableLockMu         sync.Mutex                       // guards cond waits + acquisition
	tableLockOnce       sync.Once                        // lazy-inits tableLockCond
	tableLockCond       *sync.Cond                       // broadcast on unlock
	tableLockNext       uint64                           // next FIFO ticket for LOCK TABLES acquisition
	tableLockServe      uint64                           // currently served FIFO ticket
	tableLockOwner      atomic.Pointer[scm.SessionState] // non-nil for a WRITE lock
	tableLockState      atomic.Int64                     // 0 = unlocked, -1 = WRITE, positive = READ count
	tableLockReadOwners map[*scm.SessionState]uint32     // guarded by tableLockMu
	_                   [32]byte                         // separate table locks from insert traffic
	Auto_increment      uint64                           // this dosen't scale over multiple cores, so assign auto_increment ranges to each shard
	Collation           string
	Charset             string
	Comment             string

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

	// creationMu is held while a newly published table runs its synchronous
	// oninit hook. Concurrent if-not-exists calls wait on the same barrier.
	creationMu     sync.Mutex
	creationPanic  any
	onInitComplete bool // guarded by creationMu; deliberately resets after restart

	// cacheInitMu guards the one-time initializer for canonical planner caches.
	// The table object is removed on cache eviction, so initialization state has
	// exactly the same lifetime as the cached relation itself.
	cacheInitMu         sync.Mutex
	cacheInitialized    bool
	cacheInitializerRun *cacheInitializerRun

	// ddlMu is the table-local schema contract:
	//   - Lock(): column/trigger/ORC metadata on this table may change
	//   - RLock(): rebuild/repartition may assume table-local schema stability
	// This keeps long maintenance work local to one table instead of blocking
	// unrelated tables behind the database-global schemalock.
	ddlMu sync.RWMutex

	// showColumnsSnapshot is immutable after publication. SHOW/compiler reads
	// load it without locking; metadata writers replace the complete snapshot.
	showColumnsSnapshot atomic.Pointer[tableShowColumnsSnapshot]
	columnNamesSnapshot atomic.Pointer[tableColumnNamesSnapshot]
	// plannerRowEstimate is deliberately approximate. Rebuild/statistics
	// publication replaces it atomically; query compilation must never load a
	// cold shard or take a shard read lock merely to obtain a row estimate.
	PlannerRowEstimate atomicPlannerRowEstimate `json:"planner_row_estimate"`

	// storage: ShardMode controls which shard set is the read/write target
	ShardMode   ShardMode
	Shards      []*storageShard // unordered shards (used when ShardMode == ShardModeFree)
	PShards     []*storageShard // partitioned shards according to PDimensions (used when ShardMode == ShardModePartition)
	PDimensions []shardDimension
	topology    atomic.Pointer[tableShardTopology]

	// maintenanceMu prevents concurrent rebuild and repartition on this table.
	// Holders: db.rebuild() claims it for rebuild (maintenanceKind=1) and may
	// transition to repartition (maintenanceKind=2) while still holding it.
	// beginManualRepartition() also claims it for direct repartition requests.
	maintenanceMu              sync.Mutex
	maintenanceKind            int          // 0=idle, 1=rebuilding, 2=repartitioning
	overflowRebuilds           atomic.Int32 // background workers; observed without joining the write path
	repartitionDualWriteActive atomic.Bool  // true after Phase B snapshot; dual-write only when true
	repartitionSources         atomic.Pointer[repartitionSourceSet]
	// transactionDrainMu/transactionDrain coordinate the rare repartition
	// publication wait. Transaction start/end stays lock-free unless a waiter is
	// present; transactionDrainWaiters is atomic and the condition is guarded by
	// transactionDrainMu.
	transactionDrainMu      sync.Mutex
	transactionDrain        *sync.Cond
	transactionDrainOnce    sync.Once
	transactionDrainWaiters atomic.Int32

	// repartitionTranslation maps old-shard rows to their new PShard locations.
	// Snapshot rows are published after Phase B. Post-snapshot dual-written rows
	// extend the same map during Phase C so DELETE forwarding stays O(1).
	// Key: pointer to old shard. Value: map from old recid to new location.
	repartitionTranslationMu sync.RWMutex
	repartitionTranslation   map[*storageShard]map[uint32]translatedRecid

	// repartitionPendingDels collects main-storage deletion recids that arrive
	// via DELETE dual-write before Phase D installs main_count. Phase D applies
	// them after shifting delta. Protected by repartitionPendingMu.
	repartitionPendingMu         sync.Mutex
	repartitionPendingDels       []translatedRecid
	repartitionPendingSourceDels []pendingSourceDelete
}

func (t *table) signalTransactionDrain() {
	if t.transactionDrainWaiters.Load() == 0 {
		return
	}
	t.transactionDrainMu.Lock()
	if t.transactionDrain != nil {
		t.transactionDrain.Broadcast()
	}
	t.transactionDrainMu.Unlock()
}

func (t *table) waitForTransactions(shards []*storageShard) {
	hasActive := func() bool {
		for _, shard := range shards {
			if shard != nil && shard.activeTransactions.Load() != 0 {
				return true
			}
		}
		return false
	}
	if !hasActive() {
		return
	}
	t.transactionDrainOnce.Do(func() {
		t.transactionDrainMu.Lock()
		t.transactionDrain = sync.NewCond(&t.transactionDrainMu)
		t.transactionDrainMu.Unlock()
	})
	t.transactionDrainMu.Lock()
	t.transactionDrainWaiters.Add(1)
	for hasActive() {
		t.transactionDrain.Wait()
	}
	t.transactionDrainWaiters.Add(-1)
	t.transactionDrainMu.Unlock()
}

// awaitCreationInitialization is the idempotent createtable barrier for an
// already-published table. Persisted MEMORY/CACHE schemas deliberately reload
// without data, so their closed oninit callback runs again on the first
// createtable guard after every restart.
func (t *table) awaitCreationInitialization() {
	if !t.creationMu.TryLock() {
		t.creationMu.Lock()
	}
	defer t.creationMu.Unlock()
	if t.creationPanic != nil {
		panic(t.creationPanic)
	}
	if t.PersistencyMode != Memory && t.PersistencyMode != Cache {
		t.onInitComplete = true
		return
	}
	if t.onInitComplete {
		return
	}
	if t.OnInit == nil {
		t.onInitComplete = true
		return
	}
	func() {
		defer func() {
			t.creationPanic = recover()
		}()
		scm.Apply(*t.OnInit)
		t.onInitComplete = true
	}()
	if t.creationPanic != nil {
		panic(t.creationPanic)
	}
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

// publishTopologyLocked publishes an immutable authoritative topology. The
// caller holds t.mu or otherwise has exclusive ownership before publication.
func (t *table) publishTopologyLocked() *tableShardTopology {
	mode := t.ShardMode
	shards := t.Shards
	if mode == ShardModePartition {
		shards = t.PShards
	}
	topology := &tableShardTopology{
		mode:              mode,
		shards:            append([]*storageShard(nil), shards...),
		dimensions:        append([]shardDimension(nil), t.PDimensions...),
		operationsDrained: make(chan struct{}),
		drained:           make(chan struct{}),
	}
	for _, shard := range topology.shards {
		if shard != nil {
			shard.generation.Store(topology)
		}
	}
	previous := t.topology.Swap(topology)
	if previous != nil {
		previous.retire()
	}
	return topology
}

func (t *table) pinActiveTopology() *tableShardTopology {
	for {
		topology := t.activeTopology()
		if topology.acquireOperation() {
			if t.topology.Load() == topology {
				return topology
			}
			topology.releaseOperation()
		}
	}
}

func (t *table) activeTopology() *tableShardTopology {
	if topology := t.topology.Load(); topology != nil {
		return topology
	}
	// Tables are unpublished while constructors populate their initial shard.
	// Lazily publishing here also supports zero-value tables in storage tests.
	return t.publishTopologyLocked()
}

// ActiveShards returns an immutable snapshot of the shard set that is currently
// authoritative for reads and writes.
func (t *table) ActiveShards() []*storageShard {
	return t.activeTopology().shards
}

// collectStatistics gathers approximate table statistics without coupling
// shard mutations to table-wide counters. It is intended for maintenance jobs.
func (t *table) collectStatistics() {
	t.mu.Lock()
	shards := append([]*storageShard(nil), t.ActiveShards()...)
	t.mu.Unlock()
	t.collectStatisticsFromShards(shards)
}

func (t *table) collectStatisticsFromShards(shards []*storageShard) {
	stats := &tableStatisticsSnapshot{}
	for _, shard := range shards {
		if shard == nil {
			continue
		}
		shardStats := shard.statsSnapshot()
		stats.rowCount += shardStats.rowCount()
		stats.sizeBytes += int64(shardStats.size)
	}
	t.PlannerRowEstimate.value.Store(uint64(stats.rowCount))
	for {
		current := t.showColumnsSnapshot.Load()
		if current == nil {
			replacement := t.buildShowColumnsSnapshot(uint(stats.rowCount))
			replacement.statistics = stats
			if t.showColumnsSnapshot.CompareAndSwap(nil, replacement) {
				t.columnNamesSnapshot.Store(replacement.columnNames)
				return
			}
			continue
		}
		replacement := t.buildShowColumnsSnapshot(uint(stats.rowCount))
		replacement.statistics = stats
		if t.showColumnsSnapshot.CompareAndSwap(current, replacement) {
			t.columnNamesSnapshot.Store(replacement.columnNames)
			return
		}
	}
}

// collectRebuiltColumnPlannerStatistics reads freshly rebuilt shard generations.
// A generation can already receive forwarded writes, or become live, before
// statistics publication finishes. Lock each shard independently and never
// call this helper while holding table.mu: mutation callbacks may need table
// metadata while they own the shard lock.
func collectRebuiltColumnPlannerStatistics(shards []*storageShard, columnName string) *columnPlannerStatistics {
	stats := &columnPlannerStatistics{
		Confidence:  1,
		Source:      "rebuild",
		MinEstimate: scm.NewNil(),
		MaxEstimate: scm.NewNil(),
	}
	var rowCount uint64
	var valueBytes uint64
	var hasValue bool
	var buf []scm.Scmer
	for _, shard := range shards {
		if shard == nil {
			continue
		}
		func() {
			shard.mu.RLock()
			defer shard.mu.RUnlock()
			columnStorage := shard.columns[columnName]
			if columnStorage == nil {
				return
			}
			if cap(buf) < int(shard.main_count) {
				buf = make([]scm.Scmer, shard.main_count)
			}
			buf = buf[:shard.main_count]
			reader := columnStorage.GetCachedReader()
			reader.GetValueRange(0, shard.main_count, buf, 1)
			for _, value := range buf {
				rowCount++
				if value.IsNil() {
					stats.NullCount++
					continue
				}
				if value.IsString() || value.IsCString() || value.IsBString() {
					valueBytes += uint64(len(value.String()))
				}
				if !hasValue {
					stats.MinEstimate = value
					stats.MaxEstimate = value
					hasValue = true
					continue
				}
				if scm.Less(value, stats.MinEstimate) {
					stats.MinEstimate = value
				}
				if scm.Less(stats.MaxEstimate, value) {
					stats.MaxEstimate = value
				}
			}
		}()
	}
	if rowCount > 0 {
		stats.NullFraction = float64(stats.NullCount) / float64(rowCount)
		stats.AverageValueBytes = float64(valueBytes) / float64(rowCount)
	}
	return stats
}

func (t *table) statistics() tableStatisticsSnapshot {
	if snapshot := t.showColumnsSnapshot.Load(); snapshot != nil {
		if snapshot.statistics != nil {
			return *snapshot.statistics
		}
		return tableStatisticsSnapshot{rowCount: int64(snapshot.rowEstimate)}
	}
	return tableStatisticsSnapshot{}
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
	if t.maintenanceKind == 2 {
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

func (t *table) setRepartitionTranslationMap(m map[*storageShard]map[uint32]translatedRecid) {
	t.repartitionTranslationMu.Lock()
	t.repartitionTranslation = m
	t.repartitionTranslationMu.Unlock()
}

func (t *table) mergeRepartitionTranslations(m map[*storageShard]map[uint32]translatedRecid) {
	if len(m) == 0 {
		return
	}
	t.repartitionTranslationMu.Lock()
	if t.repartitionTranslation == nil {
		t.repartitionTranslation = make(map[*storageShard]map[uint32]translatedRecid, len(m))
	}
	for shard, entries := range m {
		if len(entries) == 0 {
			continue
		}
		dst := t.repartitionTranslation[shard]
		if dst == nil {
			dst = make(map[uint32]translatedRecid, len(entries))
			t.repartitionTranslation[shard] = dst
		}
		for oldRecid, tr := range entries {
			dst[oldRecid] = tr
		}
	}
	t.repartitionTranslationMu.Unlock()
}

func (t *table) recordRepartitionTranslation(oldShard *storageShard, oldRecid uint32, tr translatedRecid) {
	t.repartitionTranslationMu.Lock()
	if t.repartitionTranslation == nil {
		t.repartitionTranslation = make(map[*storageShard]map[uint32]translatedRecid)
	}
	if t.repartitionTranslation[oldShard] == nil {
		t.repartitionTranslation[oldShard] = make(map[uint32]translatedRecid)
	}
	t.repartitionTranslation[oldShard][oldRecid] = tr
	t.repartitionTranslationMu.Unlock()
}

func (t *table) lookupRepartitionTranslation(oldShard *storageShard, oldRecid uint32) (translatedRecid, bool) {
	t.repartitionTranslationMu.RLock()
	defer t.repartitionTranslationMu.RUnlock()
	if t.repartitionTranslation == nil {
		return translatedRecid{}, false
	}
	shardMap := t.repartitionTranslation[oldShard]
	if shardMap == nil {
		return translatedRecid{}, false
	}
	tr, ok := shardMap[oldRecid]
	return tr, ok
}

func (t *table) shiftDeltaRepartitionTranslations(mainCounts []uint32) {
	t.repartitionTranslationMu.Lock()
	for _, shardMap := range t.repartitionTranslation {
		for oldRecid, tr := range shardMap {
			if !tr.inDelta {
				continue
			}
			if tr.pshardIdx >= 0 && tr.pshardIdx < len(mainCounts) {
				tr.newRecid += mainCounts[tr.pshardIdx]
			}
			tr.inDelta = false
			shardMap[oldRecid] = tr
		}
	}
	t.repartitionTranslationMu.Unlock()
}

func (t *table) clearRepartitionTranslation() {
	t.repartitionTranslationMu.Lock()
	t.repartitionTranslation = nil
	t.repartitionTranslationMu.Unlock()
}

func (t *table) appendPendingRepartitionDelete(tr translatedRecid) {
	t.repartitionPendingMu.Lock()
	t.repartitionPendingDels = append(t.repartitionPendingDels, tr)
	t.repartitionPendingMu.Unlock()
}

func (t *table) appendPendingRepartitionSourceDelete(oldShard *storageShard, oldRecid uint32) {
	t.repartitionPendingMu.Lock()
	t.repartitionPendingSourceDels = append(t.repartitionPendingSourceDels, pendingSourceDelete{oldShard: oldShard, oldRecid: oldRecid})
	t.repartitionPendingMu.Unlock()
}

func (t *table) resolvePendingRepartitionSourceDeletes() {
	t.repartitionPendingMu.Lock()
	pending := t.repartitionPendingSourceDels
	t.repartitionPendingSourceDels = nil
	t.repartitionPendingMu.Unlock()
	if len(pending) == 0 {
		return
	}

	unresolved := pending[:0]
	translated := make([]translatedRecid, 0, len(pending))
	for _, src := range pending {
		if tr, ok := t.lookupRepartitionTranslation(src.oldShard, src.oldRecid); ok {
			translated = append(translated, tr)
		} else {
			unresolved = append(unresolved, src)
		}
	}

	t.repartitionPendingMu.Lock()
	if len(translated) > 0 {
		t.repartitionPendingDels = append(t.repartitionPendingDels, translated...)
	}
	if len(unresolved) > 0 {
		t.repartitionPendingSourceDels = append(t.repartitionPendingSourceDels, unresolved...)
	}
	t.repartitionPendingMu.Unlock()
}

// beginManualRepartition serializes direct partitiontable-triggered repartitions.
// Contract:
//   - blocks until any already-running rebuild/repartition on this table finishes
//   - if the table became partitioned while waiting, no new immediate repartition starts
//   - otherwise maintenanceKind is set to 2 under maintenanceMu before returning
func (t *table) beginManualRepartition() bool {
	t.maintenanceMu.Lock()
	t.mu.Lock()
	if t.ShardMode != ShardModeFree {
		t.mu.Unlock()
		t.maintenanceMu.Unlock()
		return false
	}
	t.maintenanceKind = 2
	t.mu.Unlock()
	// maintenanceMu stays locked — released by repartition Phase G
	return true
}

// isEphemeralQueryTable identifies planner-owned query scratch tables
// (keytables, prejoins, scalar helper tables, etc.). These relations are
// dot-prefixed cache-engine tables: durable internal tables such as ".blobs"
// use persisted engines and must not be treated as ephemeral helpers.
func (t *table) isEphemeralQueryTable() bool {
	return strings.HasPrefix(t.Name, ".") && t.PersistencyMode == Cache
}

// schemaSaveMode selects the synchronous persistence guarantee for ordinary
// DDL. Callers explicitly override it for reconstructible temp metadata.
func (t *table) schemaSaveMode() schemaSaveMode {
	return schemaSaveModeForDurability(t.PersistencyMode == Safe)
}

func (t *table) finishSchemaMutationLocked(mode schemaSaveMode) {
	t.schema.saveLockedAndUnlock(mode)
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

// tableLockQueryContext resolves the exact statement generation rather than
// the session's latest query. Autocommit workers carry that generation in the
// transaction, which keeps overlapping persistent HTTP requests independent.
func tableLockQueryContext(ss *scm.SessionState) context.Context {
	if ss == nil {
		return nil
	}
	querySeq := scm.CurrentQuerySeq()
	if tx := CurrentTx(); tx != nil {
		if txSeq := querySeqFromTx(tx); txSeq != 0 {
			querySeq = txSeq
		}
	}
	return ss.QueryContext(querySeq)
}

// waitTableLock blocks until the table lock is compatible with the caller's intent.
// isWrite=true means the caller wants to write (blocked by ANY lock from another session).
// isWrite=false means the caller wants to read (blocked only by WRITE lock from another session).
// SHOW PROCESSLIST derives its waiting state from the active lock-wait count.
// Panics if the owning session tries to write while holding a READ lock (MySQL semantics).
func (t *table) waitTableLock(ss *scm.SessionState, isWrite bool) {
	cond := t.getTableLockCond()
	ctx := tableLockQueryContext(ss)
	if ctx != nil {
		stopWake := context.AfterFunc(ctx, func() {
			t.tableLockMu.Lock()
			cond.Broadcast()
			t.tableLockMu.Unlock()
		})
		defer stopWake()
	}
	if ss != nil {
		ss.BeginLockWait()
		defer ss.EndLockWait()
	}
	var errMsg string
	t.tableLockMu.Lock()
	for {
		if ctx != nil && ctx.Err() != nil {
			errMsg = "query killed"
			break
		}
		owner := t.tableLockOwner.Load()
		state := t.tableLockState.Load()
		if !isWrite {
			if state >= 0 || owner == ss {
				break
			}
		} else if owner == ss {
			break
		} else if t.tableLockReadOwners[ss] != 0 {
			errMsg = "Can't write to table '" + t.Name + "' while it has a READ lock"
			break
		} else if state == 0 {
			break
		}
		cond.Wait()
	}
	t.tableLockMu.Unlock()
	if errMsg != "" {
		panic(errMsg)
	}
}

func (t *table) hasTableLock() bool {
	return t.tableLockState.Load() != 0
}

// unlockTableWrite releases the exclusive table lock and wakes all waiters.
func (t *table) unlockTableWrite() {
	t.tableLockOwner.Store(nil)
	t.tableLockState.Store(0)
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

// CountEstimate returns the last atomically published table-wide estimate.
// It is O(1), non-blocking, and never acquires concurrency rights or causes a
// lazy shard load. Statistics are refreshed by rebuild/collection; staleness is
// preferable to adding locks or storage I/O to the query compiler hot path.
func (t *table) CountEstimate() uint {
	return uint(t.PlannerRowEstimate.value.Load())
}

// initializeLegacyPlannerRowEstimate migrates schemas written before the
// planner_row_estimate field existed. It runs once while the database is still
// loading, touches one persisted column per shard to recover the authoritative
// row count, and keeps every subsequent compiler lookup O(1).
func (t *table) initializeLegacyPlannerRowEstimate() {
	if t.PlannerRowEstimate.present.Load() && t.PlannerRowEstimate.value.Load() > 0 {
		return
	}
	var rows int64
	for _, shard := range t.ActiveShards() {
		if shard == nil {
			continue
		}
		func() {
			release := shard.GetRead()
			defer release()
			rows += shard.statsSnapshot().rowCount()
		}()
	}
	if rows < 0 {
		rows = 0
	}
	t.PlannerRowEstimate.value.Store(uint64(rows))
	t.PlannerRowEstimate.present.Store(true)
}

// adjustPlannerRows advances the approximate cardinality after one DML batch.
// Only the two-entry snapshot root is replaced; the immutable hashed column
// catalog is reused. This keeps maintenance O(1) per batch and avoids forcing
// the next compiler through shard state.
func (t *table) adjustPlannerRows(delta int64) {
	if delta == 0 {
		return
	}
	var rowEstimate uint64
	for {
		oldEstimate := t.PlannerRowEstimate.value.Load()
		if delta > 0 {
			rowEstimate = oldEstimate + uint64(delta)
		} else if decrement := uint64(-delta); decrement < oldEstimate {
			rowEstimate = oldEstimate - decrement
		} else {
			rowEstimate = 0
		}
		if t.PlannerRowEstimate.value.CompareAndSwap(oldEstimate, rowEstimate) {
			break
		}
	}
	for {
		current := t.showColumnsSnapshot.Load()
		if current == nil || current.plannerValue.IsNil() {
			return
		}
		columns, ok := current.plannerValue.FastDict().Get(scm.NewString("columns"))
		if !ok {
			return
		}
		plannerRoot := scm.NewFastDictValue(2)
		plannerRoot.Set(scm.NewString("row_count"), scm.NewInt(int64(rowEstimate)), nil)
		plannerRoot.Set(scm.NewString("columns"), columns, nil)
		replacement := *current
		replacement.plannerValue = scm.NewFastDict(plannerRoot)
		replacement.rowEstimate = uint(rowEstimate)
		if t.showColumnsSnapshot.CompareAndSwap(current, &replacement) {
			return
		}
	}
}

// CountExact returns the current visible row count. It may load shards and is
// reserved for correctness-sensitive execution paths, never query planning.
func (t *table) CountExact() (result uint) {
	for _, shard := range t.ActiveShards() {
		if shard == nil {
			continue
		}
		func() {
			unlock := shard.GetRead()
			defer unlock()
			shard.mu.RLock()
			defer shard.mu.RUnlock()
			result += uint(shard.Count())
		}()
	}
	return result
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

type tableShowColumnsSnapshot struct {
	value        scm.Scmer
	plannerValue scm.Scmer
	rowEstimate  uint
	columns      *tableShowColumnsMetadata
	columnNames  *tableColumnNamesSnapshot
	statistics   *tableStatisticsSnapshot
}

type tableShowColumnsMetadata struct {
	distinctEstimates []uint64
	plannerStatistics []*columnPlannerStatistics
}

type tableColumnNamesSnapshot struct {
	exact  map[string]string
	folded map[string]string
}

func foldIdentifier(name string) string {
	for i := 0; i < len(name); i++ {
		if name[i] >= utf8.RuneSelf {
			var b strings.Builder
			b.Grow(len(name))
			for _, r := range name {
				canonical := r
				for folded := unicode.SimpleFold(r); folded != r; folded = unicode.SimpleFold(folded) {
					if folded < canonical {
						canonical = folded
					}
				}
				b.WriteRune(canonical)
			}
			return b.String()
		}
	}
	return strings.ToLower(name)
}

func (t *table) buildShowColumnsSnapshot(rowEstimate uint) *tableShowColumnsSnapshot {
	result := make([]scm.Scmer, len(t.Columns))
	plannerColumns := scm.NewFastDictValue(len(t.Columns))
	distinctEstimates := make([]uint64, len(t.Columns))
	plannerStatistics := make([]*columnPlannerStatistics, len(t.Columns))
	columnNames := t.buildColumnNamesSnapshot()
	for i, c := range t.Columns {
		keyType := ""
		for _, uk := range t.Unique {
			for _, col := range uk.Cols {
				if col == c.Name {
					if uk.Id == "PRIMARY" {
						keyType = "PRI"
					} else if keyType == "" {
						keyType = "UNI"
					}
				}
			}
		}
		distinctEstimate := atomic.LoadUint64(&c.DistinctEstimate)
		if distinctEstimate == 0 {
			distinctEstimate = uint64(rowEstimate)
		}
		distinctEstimates[i] = distinctEstimate
		plannerStatistics[i] = c.PlannerStats.Load()
		result[i] = c.show(keyType, distinctEstimate, rowEstimate, plannerStatistics[i])
		plannerStatsValue := plannerColumnStatisticsValue(distinctEstimate, plannerStatistics[i], c.Typ)
		plannerColumns.Set(scm.NewString(c.Name), plannerStatsValue, nil)
		if folded := foldIdentifier(c.Name); folded != c.Name {
			plannerColumns.Set(scm.NewString(folded), plannerStatsValue, nil)
		}
	}
	plannerRoot := scm.NewFastDictValue(2)
	plannerRoot.Set(scm.NewString("row_count"), scm.NewInt(int64(rowEstimate)), nil)
	plannerRoot.Set(scm.NewString("columns"), scm.NewFastDict(plannerColumns), nil)
	snapshot := &tableShowColumnsSnapshot{
		value:        scm.NewSlice(result),
		plannerValue: scm.NewFastDict(plannerRoot),
		rowEstimate:  rowEstimate,
		columns: &tableShowColumnsMetadata{
			distinctEstimates: distinctEstimates,
			plannerStatistics: plannerStatistics,
		},
		columnNames: columnNames,
	}
	if current := t.showColumnsSnapshot.Load(); current != nil {
		snapshot.statistics = current.statistics
	}
	return snapshot
}

func (t *table) buildColumnNamesSnapshot() *tableColumnNamesSnapshot {
	exact := make(map[string]string, len(t.Columns))
	folded := make(map[string]string, len(t.Columns))
	for _, c := range t.Columns {
		exact[c.Name] = c.Name
		foldedName := foldIdentifier(c.Name)
		if _, exists := folded[foldedName]; !exists {
			folded[foldedName] = c.Name
		}
	}
	return &tableColumnNamesSnapshot{exact: exact, folded: folded}
}

func (t *table) publishColumnNamesSnapshot() *tableColumnNamesSnapshot {
	snapshot := t.buildColumnNamesSnapshot()
	t.columnNamesSnapshot.Store(snapshot)
	return snapshot
}

func (t *table) publishShowColumnsSnapshot() scm.Scmer {
	snapshot := t.buildShowColumnsSnapshot(t.CountEstimate())
	t.columnNamesSnapshot.Store(snapshot.columnNames)
	t.showColumnsSnapshot.Store(snapshot)
	return snapshot.value
}

func (t *table) invalidateShowColumnsSnapshot() {
	t.publishColumnNamesSnapshot()
	t.showColumnsSnapshot.Store(nil)
}

// ResolveColumnName reads immutable DDL metadata without scanning SHOW rows.
func (t *table) ResolveColumnName(name string, ignoreCase bool) (string, bool) {
	if t == nil {
		return "", false
	}
	snapshot := t.columnNamesSnapshot.Load()
	if snapshot == nil {
		snapshot = t.publishColumnNamesSnapshot()
	}
	if resolved, ok := snapshot.exact[name]; ok {
		return resolved, true
	}
	if ignoreCase {
		resolved, ok := snapshot.folded[foldIdentifier(name)]
		return resolved, ok
	}
	return "", false
}

func (t *table) ShowColumns() scm.Scmer {
	if t == nil {
		return scm.NewNil()
	}
	snapshot := t.showColumnsSnapshot.Load()
	if snapshot != nil {
		return snapshot.value
	}
	return t.publishShowColumnsSnapshot()
}

// PlannerStatistics returns one immutable, atomically published catalog value.
// The compiler can retain this value for its complete planning scope and use
// hashed column lookup without revisiting SHOW metadata.
func (t *table) PlannerStatistics() scm.Scmer {
	if t == nil {
		return scm.NewNil()
	}
	for {
		snapshot := t.showColumnsSnapshot.Load()
		if snapshot != nil {
			return snapshot.plannerValue
		}
		t.publishShowColumnsSnapshot()
	}
}

func plannerColumnStatisticsValue(distinctEstimate uint64, stats *columnPlannerStatistics, rawType string) scm.Scmer {
	known := stats != nil
	confidence := float64(0)
	source := "unknown"
	distinctSource := "fallback_row_count"
	nullFraction := scm.NewNil()
	minEstimate := scm.NewNil()
	maxEstimate := scm.NewNil()
	averageValueBytes := scm.NewNil()
	if stats != nil {
		confidence = stats.Confidence
		source = stats.Source
		distinctSource = "rebuild"
		nullFraction = scm.NewFloat(stats.NullFraction)
		minEstimate = stats.MinEstimate
		maxEstimate = stats.MaxEstimate
		averageValueBytes = scm.NewFloat(stats.AverageValueBytes)
	}
	return scm.NewSlice([]scm.Scmer{
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("known"), scm.NewBool(known)}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("confidence"), scm.NewFloat(confidence)}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("source"), scm.NewSymbol(source)}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("distinct"), scm.NewInt(int64(distinctEstimate))}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("distinct_confidence"), scm.NewFloat(confidence)}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("distinct_source"), scm.NewSymbol(distinctSource)}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("null_fraction"), nullFraction}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("min"), minEstimate}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("max"), maxEstimate}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("raw_type"), scm.NewString(rawType)}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("average_value_bytes"), averageValueBytes}),
	})
}

func (c *column) Show(keyType string, t *table) scm.Scmer {
	return c.show(keyType, uint64(c.distinctEstimateFor(t)), t.CountEstimate(), c.PlannerStats.Load())
}

func (c *column) show(keyType string, distinctEstimate uint64, rowEstimate uint, plannerStats *columnPlannerStatistics) scm.Scmer {
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
	statisticsKnown := plannerStats != nil
	statisticsConfidence := 0.0
	statisticsSource := "unknown"
	nullFraction := scm.NewNil()
	minEstimate := scm.NewNil()
	maxEstimate := scm.NewNil()
	averageValueBytes := scm.NewNil()
	if statisticsKnown {
		statisticsConfidence = plannerStats.Confidence
		statisticsSource = plannerStats.Source
		nullFraction = scm.NewFloat(plannerStats.NullFraction)
		minEstimate = plannerStats.MinEstimate
		maxEstimate = plannerStats.MaxEstimate
		averageValueBytes = scm.NewFloat(plannerStats.AverageValueBytes)
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
		scm.NewString("DistinctEstimate"), scm.NewInt(int64(distinctEstimate)),
		scm.NewString("RowEstimate"), scm.NewInt(int64(rowEstimate)),
		scm.NewString("StatisticsKnown"), scm.NewBool(statisticsKnown),
		scm.NewString("StatisticsConfidence"), scm.NewFloat(statisticsConfidence),
		scm.NewString("StatisticsSource"), scm.NewString(statisticsSource),
		scm.NewString("DistinctEstimateSource"), scm.NewString(func() string {
			if statisticsKnown {
				return "rebuild"
			}
			return "fallback_row_count"
		}()),
		scm.NewString("NullFraction"), nullFraction,
		scm.NewString("MinEstimate"), minEstimate,
		scm.NewString("MaxEstimate"), maxEstimate,
		scm.NewString("AverageValueBytes"), averageValueBytes,
	})
}

// distinctEstimateFor returns the DistinctEstimate for this column.
// Uses cached value from rebuild if available. Falls back to CountEstimate
// (conservative upper bound) when no rebuild statistics exist yet.
// Never acquires shard locks — safe to call during query compilation.
func (c *column) distinctEstimateFor(t *table) uint {
	cached := atomic.LoadUint64(&c.DistinctEstimate)
	if cached > 0 {
		return uint(cached)
	}
	// No rebuild statistics yet — use row count as conservative upper bound.
	// This ensures join_reorder can still compare table sizes even before
	// the first rebuild runs.
	return uint(t.CountEstimate())
}

// truncateStringCharacters applies SQL character limits to UTF-8 characters,
// not to their encoded byte length.
func truncateStringCharacters(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	characters := 0
	for byteOffset := range value {
		if characters == limit {
			return value[:byteOffset]
		}
		characters++
	}
	return value
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
				s := truncateStringCharacters(scm.String(v), maxLen)
				if typ == "CHAR" {
					// Keep CHAR compact while preserving its blank-padded SQL value:
					// trailing pad spaces are not observable when read back.
					s = strings.TrimRight(s, " ")
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
	// Scmer's Go zero value is not Scheme nil. Base columns must be explicitly
	// non-computed immediately after DDL; otherwise rebuild statistics are
	// skipped until a schema reload happens to normalize this field.
	c.Computor = scm.NewNil()
	c.ComputorFilter = scm.NewNil()
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
		case "sortcols", "sortdirs", "partitioncount", "mapcols", "mapfn", "reducefn", "reduceinit":
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
				// A trigger that already snapshotted this column pins it without a
				// mutex. Retry eviction instead of deleting its target underneath it.
				if !col.beginCacheEviction() {
					tbl.schema.schemalock.Unlock()
					return false
				}
				tbl.removeComputeTriggers(colName)
				tbl.removeORCDependencyTriggers(colName)
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
				tbl.invalidateShowColumnsSnapshot()
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
	if cp.IsTemp {
		t.invalidateShowColumnsSnapshot()
	} else {
		t.publishShowColumnsSnapshot()
	}
	mode := t.schemaSaveMode()
	if cp.IsTemp {
		mode = schemaSaveBuffered
	}
	t.finishSchemaMutationLocked(mode)
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

			if c.IsTemp {
				t.invalidateShowColumnsSnapshot()
			} else {
				t.publishShowColumnsSnapshot()
			}
			t.finishSchemaMutationLocked(t.schemaSaveMode())
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

func (t *table) appendFreeShardDurably(topology *tableShardTopology, source *storageShard, incoming uint) (*tableShardTopology, *storageShard) {
	t.maintenanceMu.Lock()
	sourceCount := uint(source.Count())
	t.mu.Lock()
	active := t.activeTopology()
	if active != topology || active.mode != ShardModeFree || len(active.shards) == 0 || active.shards[len(active.shards)-1] != source {
		t.mu.Unlock()
		t.maintenanceMu.Unlock()
		return active, nil
	}
	if sourceCount+incoming <= Settings.ShardSize {
		t.mu.Unlock()
		t.maintenanceMu.Unlock()
		return active, source
	}

	newShard := NewShard(t)
	t.maintenanceKind = 1
	t.Shards = append(t.Shards, newShard)
	newIndex := len(t.Shards) - 1
	sourceIndex := newIndex - 1
	t.mu.Unlock()

	var savePanic any
	func() {
		defer func() { savePanic = recover() }()
		t.schema.save()
	}()
	if savePanic != nil {
		t.mu.Lock()
		if newIndex < len(t.Shards) && t.Shards[newIndex] == newShard {
			t.Shards = t.Shards[:newIndex]
		}
		t.maintenanceKind = 0
		t.mu.Unlock()
		t.maintenanceMu.Unlock()
		discardUnpublishedShard(newShard)
		panic(savePanic)
	}

	t.mu.Lock()
	published := t.publishTopologyLocked()
	t.mu.Unlock()
	if t.PersistencyMode == Cache && !strings.HasPrefix(t.Name, ".") {
		GlobalCache.AddItem(newShard, 0, TypeCacheEntry, cacheShardCleanup, shardLastUsed, nil)
	}
	fmt.Println("started new shard for table", t.Name)
	t.overflowRebuilds.Add(1)
	go t.finishOverflowRebuild(sourceIndex, source)
	return published, newShard
}

// finishOverflowRebuild owns maintenanceMu, which appendFreeShardDurably
// acquired before publishing the appended shard. The replacement UUID becomes
// authoritative only after schema.json durably names it.
func (t *table) finishOverflowRebuild(index int, source *storageShard) {
	defer func() {
		t.mu.Lock()
		if t.maintenanceKind == 1 {
			t.maintenanceKind = 0
		}
		t.mu.Unlock()
		t.maintenanceMu.Unlock()
		t.overflowRebuilds.Add(-1)
		if r := recover(); r != nil {
			fmt.Println("error: shard rebuild failed for", t.schema.Name+".", t.Name, "shard", index, ":", r)
		}
	}()

	rebuilt := source.rebuild(false)
	source.mu.Lock()
	t.mu.Lock()
	if t.ShardMode != ShardModeFree || index >= len(t.Shards) || t.Shards[index] != source {
		t.mu.Unlock()
		source.mu.Unlock()
		return
	}
	t.Shards[index] = rebuilt
	t.mu.Unlock()

	var savePanic any
	func() {
		defer func() { savePanic = recover() }()
		t.schema.save()
	}()
	if savePanic != nil {
		t.mu.Lock()
		if index < len(t.Shards) && t.Shards[index] == rebuilt {
			t.Shards[index] = source
		}
		t.mu.Unlock()
		source.mu.Unlock()
		panic(savePanic)
	}

	t.mu.Lock()
	t.publishTopologyLocked()
	t.mu.Unlock()
	source.mu.Unlock()
	t.collectStatistics()
}

func (t *table) Insert(columns []string, values [][]scm.Scmer, onCollisionCols []string, onCollision scm.Scmer, mergeNull bool, onFirstInsertId func(int64)) int {
	result := 0
	inserted := 0
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

	topology := t.pinActiveTopology()
	defer func() { topology.releaseOperation() }()
	switchTopology := func(next *tableShardTopology) {
		topology.releaseOperation()
		if next != nil && next.acquireOperation() {
			if t.topology.Load() == next {
				topology = next
				return
			}
			next.releaseOperation()
		}
		topology = t.pinActiveTopology()
	}
	if topology.mode == ShardModeFree { // unpartitioned sharding
		// Helper to get or create a shard with capacity for n rows
		getShardWithCapacity := func(n uint) *storageShard {
			for {
				t.mu.Lock()
				active := t.activeTopology()
				if active != topology {
					t.mu.Unlock()
					switchTopology(active)
					if topology.mode != ShardModeFree {
						return nil
					}
					continue
				}
				if len(topology.shards) == 0 {
					t.mu.Unlock()
					return nil
				}
				shard := topology.shards[len(topology.shards)-1]
				if uint(shard.Count())+n <= Settings.ShardSize || t.maintenanceKind != 0 {
					t.mu.Unlock()
					return shard
				}

				t.mu.Unlock()
				published, appended := t.appendFreeShardDurably(topology, shard, n)
				if published != nil {
					switchTopology(published)
				}
				if appended != nil {
					return appended
				}
			}
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
				// Repartition completed after this Insert loaded its starting
				// generation. Reload the published partition topology before routing
				// the remaining rows; the old free generation may already have
				// finished its forwarding drain.
				switchTopology(nil)
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
					inserted += len(chunk)
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
				inserted += len(chunk)
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
		dims := topology.dimensions
		partitionShards := topology.shards
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
					inserted += len(values)
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
				inserted += len(values)
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
			shard := partitionShards[computeShardIndex(dims, shardcols)]
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
	t.adjustPlannerRows(int64(inserted))
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

func (t *table) isRepartitionSource(shard *storageShard) bool {
	sources := t.repartitionSources.Load()
	if sources == nil {
		return false
	}
	_, ok := sources.set[shard]
	return ok
}

func (t *table) dualWriteInsertFromOld(oldShard *storageShard, firstOldRecid uint32, columns []string, values [][]scm.Scmer, currentTx *TxContext) {
	if t.PShards == nil || len(values) == 0 {
		return
	}
	dims := t.PDimensions
	shardcols := make([]scm.Scmer, len(dims))
	translatable := make([]int, len(dims))
	for i, cd := range dims {
		translatable[i] = -1
		for j, col := range columns {
			if cd.Column == col {
				translatable[i] = j
				break
			}
		}
	}

	lastI := 0
	lastPSI := -1
	var lastShard *storageShard
	flush := func(end int) {
		if lastShard == nil || end <= lastI {
			return
		}
		rel := lastShard.GetExclusive()
		firstNewRecid := lastShard.insertReplica(columns, values[lastI:end], false, currentTx)
		rel()
		for i := lastI; i < end; i++ {
			t.recordRepartitionTranslation(oldShard, firstOldRecid+uint32(i), translatedRecid{
				pshardIdx: lastPSI,
				newRecid:  firstNewRecid + uint32(i-lastI),
				inDelta:   true,
			})
		}
	}

	for i := 0; i < len(values); i++ {
		for j, colidx := range translatable {
			if colidx >= 0 && colidx < len(values[i]) {
				shardcols[j] = values[i][colidx]
			} else {
				shardcols[j] = scm.NewNil()
			}
		}
		psi := computeShardIndex(dims, shardcols)
		shard := t.PShards[psi]
		if i > 0 && shard != lastShard {
			flush(i)
			lastI = i
		}
		lastPSI = psi
		lastShard = shard
	}
	flush(len(values))
}

// dualWriteDelete forwards a DELETE to PShards during repartition.
// For rows in the Phase B snapshot (present in repartitionTranslation),
// it uses the translation map to find the exact PShard recid. Post-snapshot
// dual-written rows extend the same translation map as they are inserted.
func (t *table) dualWriteDelete(oldShard *storageShard, oldRecid uint32, currentTx *TxContext) {
	if currentTx != nil {
		currentTx.addRepartitionDelete(t, oldShard, oldRecid)
		return
	}
	if tr, ok := t.lookupRepartitionTranslation(oldShard, oldRecid); ok {
		t.appendPendingRepartitionDelete(tr)
		return
	}
	t.appendPendingRepartitionSourceDelete(oldShard, oldRecid)
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
		// maintenanceKind == 2 still true), in-flight scans on old shards call
		// ProcessUniqueCollision. We must check old Shards because the deletion
		// from the UPDATE is only in the old shard, not yet in PShards.
		if t.maintenanceKind == 2 && t.ShardMode == ShardModePartition && t.Shards != nil {
			shardlist = t.Shards
		}
		allowPruning := false // if we can prune the shardlist
		pruningMap := make([]int, len(uniq.Cols))
		pruningVals := make([]scm.Scmer, len(uniq.Cols))
		if t.ShardMode == ShardModePartition && t.maintenanceKind != 2 {
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

		currentTx := CurrentTx()
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
				uid, present := s.GetRecordidForUnique(uniq.Cols, key, currentTx)
				if present {
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
							params[i] = scm.NewFunc(s.UpdateFunction(uid, true, false, currentTx))
						} else if len(p) > 5 && p[:5] == "$set:" {
							// Match the physical scan mapper's $set pseudo-column. Compute
							// proxies need a point write because they are not row payload;
							// ordinary cache columns retain normal UPDATE semantics.
							cacheColName := p[5:]
							column := s.getColumnStorageOrPanic(cacheColName)
							proxy, computed := column.(*StorageComputeProxy)
							update := s.UpdateFunction(uid, true, false, currentTx)
							params[i] = scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
								if len(args) > 0 {
									if computed {
										proxy.SetValue(uid, args[0])
									} else {
										update(scm.NewSlice([]scm.Scmer{scm.NewString(cacheColName), args[0]}))
									}
								}
								return scm.NewBool(true)
							})
						} else if len(p) >= 4 && p[:4] == "NEW." {
							for j, c := range columns {
								if p[4:] == c {
									params[i] = row[j]
								}
							}
						} else {
							params[i] = s.ColumnReaderTx(currentTx, p)(uid)
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
