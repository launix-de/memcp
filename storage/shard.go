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
import "sync"
import "sync/atomic"
import "time"
import "strings"
import "reflect"
import "runtime"
import "encoding/json"
import "encoding/binary"
import "github.com/carli2/hybridsort"
import "github.com/google/uuid"
import "github.com/jtolds/gls"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/go-mysqlstack/sqldb"
import "github.com/launix-de/NonLockingReadMap"

type storageShard struct {
	t    *table
	uuid uuid.UUID // uuid.String()
	// main storage
	main_count uint32 // size of main storage
	// columns, deltaColumns, inserts, deletions and Indexes are shard-local
	// internals guarded by mu. Code outside this file
	// must treat s.mu as the only authority for shard-local state and must not
	// read Go maps here lock-free.
	columns map[string]ColumnStorage
	// delta storage
	deltaColumns map[string]int
	inserts      [][]scm.Scmer                       // items added to storage
	deletions    NonLockingReadMap.NonBlockingBitMap // items removed from main or inserts (based on main_count + i)
	// rollbackProtected keeps transaction-owned tombstones in a rebuild so a
	// later rollback or ACID commit can still change successor visibility.
	// Guarded by mu.
	rollbackProtected NonLockingReadMap.NonBlockingBitMap
	writeOwners       map[uint64]uint32  // goroutine-local write ownership marker
	writeOwnMu        sync.Mutex         // guards writeOwners
	logfile           PersistenceLogfile // only in safe mode
	// mu protects shard-local topology/runtime state:
	//   - columns / deltaColumns / inserts / deletions
	//   - Indexes
	//   - lazy-load attach state and next-shard dual-write publication helpers
	// Reads take RLock snapshots; mutations, log replay, rebuild publish and
	// repartition dual-write state updates take Lock.
	mu         sync.RWMutex
	uniquelock sync.Mutex                   // unique insert lock (only used in the sharded case)
	next       atomic.Pointer[storageShard] // rebuild successor published lock-free to concurrent writers
	// nextReady separates rebuild publication from direct maintenance forwarding.
	// While false, writers only mutate this shard; rebuild catches up from the
	// appended delta rows and pending delete recids before setting it true.
	nextReady          atomic.Bool
	nextPendingDeletes []uint32 // guarded by mu
	// nextTranslation maps old recids on this shard to recids on the current
	// rebuild successor. Snapshot rows are installed during rebuild setup; delta
	// inserts extend the same map as they are mirrored into next.
	nextTranslationMu sync.RWMutex
	nextTranslation   map[uint32]uint32
	// indexes
	Indexes    []*StorageIndex // sorted keys
	indexMutex sync.Mutex

	// lazy-loading/shared-resource state
	srState      SharedState
	lastAccessed uint64 // UnixNano, atomic; updated on GetRead/GetExclusive for LRU eviction

	// Repartition generation tracking. Counters are atomic; generation points to
	// the immutable topology owning this shard and changes only on publication.
	activeScanners     atomic.Int32
	activeTransactions atomic.Int32
	generation         atomic.Pointer[tableShardTopology]

	// guards RemoveFromDisk against double execution (finalizer + explicit cleanup)
	cleanupOnce sync.Once
}

func (s *storageShard) beginTransactionUse() {
	s.activeTransactions.Add(1)
}

func (s *storageShard) endTransactionUse() {
	if s.activeTransactions.Add(-1) != 0 {
		return
	}
	s.t.signalTransactionDrain()
}

func (s *storageShard) loadNext() *storageShard {
	return s.next.Load()
}

func (s *storageShard) storeNext(next *storageShard) {
	s.next.Store(next)
}

func (s *storageShard) clearNext(next *storageShard) {
	s.next.CompareAndSwap(next, nil)
}

// nextForMaintenanceLocked returns a completed rebuild successor. While the
// successor is still being built, deleted recids are buffered on the source
// shard and inserts remain discoverable in its append-only delta suffix.
// Caller must hold s.mu.Lock().
func (s *storageShard) nextForMaintenanceLocked(deletedRecid *uint32) *storageShard {
	next := s.loadNext()
	if next == nil {
		return nil
	}
	if s.nextReady.Load() {
		return next
	}
	if deletedRecid != nil {
		s.nextPendingDeletes = append(s.nextPendingDeletes, *deletedRecid)
	}
	return nil
}

// syncNextVisibilityLocked mirrors the final visibility of one source recid
// when transaction commit/rollback changes it after the rebuild snapshot. A
// A successor already tracked by the same transaction applies its own masks
// and must not be locked recursively here. Caller must hold s.mu.Lock().
func (s *storageShard) syncNextVisibilityLocked(oldRecid uint32, trackedByTx map[*storageShard]struct{}) {
	current := s
	currentRecid := oldRecid
	currentIsSource := true
	for {
		next := current.loadNext()
		if next == nil {
			break
		}
		if _, tracked := trackedByTx[next]; tracked {
			break
		}
		if !current.nextReady.Load() {
			current.nextPendingDeletes = append(current.nextPendingDeletes, currentRecid)
			break
		}
		newRecid, ok := current.translateNextRecid(currentRecid)
		if !ok {
			break
		}
		deleted := current.deletions.Get(uint(currentRecid))
		protected := current.rollbackProtected.Get(uint(currentRecid))
		next.mu.Lock()
		wasDeleted := next.deletions.Get(uint(newRecid))
		next.deletions.Set(uint(newRecid), deleted)
		next.rollbackProtected.Set(uint(newRecid), protected)
		if wasDeleted != deleted {
			next.logVisibilityChangeLocked(newRecid, deleted)
		}
		if !currentIsSource {
			current.mu.Unlock()
		}
		current = next
		currentRecid = newRecid
		currentIsSource = false
	}
	if !currentIsSource {
		current.mu.Unlock()
	}
}

// logVisibilityChangeLocked persists a visibility transition without changing
// row identity. Caller must hold s.mu.Lock().
func (s *storageShard) logVisibilityChangeLocked(recid uint32, deleted bool) {
	if (s.t.PersistencyMode != Safe && s.t.PersistencyMode != Logged) || s.logfile == nil {
		return
	}
	if deleted {
		s.logfile.Write(LogEntryDelete{recid})
		return
	}
	s.logfile.Write(LogEntryUndelete{recid})
}

// catchUpRebuildLocked replays only mutations that happened after the rebuild
// snapshot. Caller holds both s.mu and next.mu. Inserts are recoverable from
// the append-only delta suffix; deletes are the small recid journal populated
// by nextForMaintenanceLocked.
func (s *storageShard) catchUpRebuildLocked(next *storageShard, snapshotInsertCount int) {
	seen := make(map[uint32]struct{}, len(s.nextPendingDeletes))
	syncNext := func(oldRecid uint32) {
		if _, duplicate := seen[oldRecid]; duplicate {
			return
		}
		seen[oldRecid] = struct{}{}
		newRecid, ok := s.translateNextRecid(oldRecid)
		if !ok {
			return
		}
		deleted := s.deletions.Get(uint(oldRecid))
		next.deletions.Set(uint(newRecid), deleted)
		next.rollbackProtected.Set(uint(newRecid), s.rollbackProtected.Get(uint(oldRecid)))
		if deleted && (next.t.PersistencyMode == Safe || next.t.PersistencyMode == Logged) && next.logfile != nil {
			next.logfile.Write(LogEntryDelete{newRecid})
		}
	}

	if snapshotInsertCount < len(s.inserts) {
		columns, rows := s.materializedInsertedRowsLocked(snapshotInsertCount)
		oldStart := s.main_count + uint32(snapshotInsertCount)
		newStart := next.insertReplica(columns, rows, true, nil)
		s.recordNextInsertRange(oldStart, newStart, len(rows))
		for i := range rows {
			syncNext(oldStart + uint32(i))
		}
	}

	for _, oldRecid := range s.nextPendingDeletes {
		syncNext(oldRecid)
	}
	s.nextPendingDeletes = nil
}

func (s *storageShard) setNextTranslation(m map[uint32]uint32) {
	s.nextTranslationMu.Lock()
	s.nextTranslation = m
	s.nextTranslationMu.Unlock()
}

func (s *storageShard) clearNextTranslation() {
	s.nextTranslationMu.Lock()
	s.nextTranslation = nil
	s.nextTranslationMu.Unlock()
}

func (s *storageShard) translateNextRecid(oldRecid uint32) (uint32, bool) {
	s.nextTranslationMu.RLock()
	defer s.nextTranslationMu.RUnlock()
	if s.nextTranslation == nil {
		return 0, false
	}
	newRecid, ok := s.nextTranslation[oldRecid]
	return newRecid, ok
}

func (s *storageShard) recordNextInsertRange(oldStartRecid, newStartRecid uint32, count int) {
	if count == 0 {
		return
	}
	s.nextTranslationMu.Lock()
	if s.nextTranslation == nil {
		s.nextTranslation = make(map[uint32]uint32, count)
	}
	for i := 0; i < count; i++ {
		s.nextTranslation[oldStartRecid+uint32(i)] = newStartRecid + uint32(i)
	}
	s.nextTranslationMu.Unlock()
}

// computeSizeLocked computes the shard's memory footprint without acquiring s.mu.
// Caller must already hold s.mu (read or write).
func (s *storageShard) computeSizeLocked() uint {
	var result uint = 14*8 + 32*8 // heuristic for columns map
	if s.srState != COLD {
		for _, c := range s.columns {
			if c != nil {
				result += c.ComputeSize()
			}
		}
		result += s.deletions.ComputeSize()
		result += scm.ComputeSize(scm.NewAny(s.inserts))
		for _, idx := range s.Indexes {
			result += idx.ComputeSize()
		}
		return result
	}
	result += s.deletions.ComputeSize()
	result += scm.ComputeSize(scm.NewAny(s.inserts))
	return result
}

func (s *storageShard) ComputeSize() uint {
	var result uint = 14*8 + 32*8 // heuristic for columns map
	if s.srState != COLD {
		s.mu.RLock()
		for _, c := range s.columns {
			if c != nil {
				result += c.ComputeSize()
			}
		}
		s.mu.RUnlock()
		result += s.deletions.ComputeSize()
		result += scm.ComputeSize(scm.NewAny(s.inserts))
		for _, idx := range s.Indexes {
			result += idx.ComputeSize()
		}
		return result
	}
	result += s.deletions.ComputeSize()
	result += scm.ComputeSize(scm.NewAny(s.inserts))
	for _, idx := range s.Indexes {
		result += idx.ComputeSize()
	}
	return result
}

type shardStatsSnapshot struct {
	mainCount uint32
	delta     int
	deletions uint
	state     SharedState
	size      uint
}

// statsSnapshotRLocked reads one consistent shard-local statistics snapshot.
// The caller must already hold s.mu.RLock().
func (s *storageShard) statsSnapshotRLocked() shardStatsSnapshot {
	return shardStatsSnapshot{
		mainCount: s.main_count,
		delta:     len(s.inserts),
		deletions: s.deletions.Count(),
		state:     s.srState,
		size:      s.computeSizeLocked(),
	}
}

func (s *storageShard) statsSnapshot() shardStatsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := s.statsSnapshotRLocked()
	return stats
}

func (s shardStatsSnapshot) rowCount() int64 {
	return int64(s.mainCount) + int64(s.delta) - int64(s.deletions)
}

func (u *storageShard) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.uuid.String())
}
func (u *storageShard) UnmarshalJSON(data []byte) error {
	u.uuid.UnmarshalText(data)
	// do not load heavy fields here; delay until first access
	u.columns = make(map[string]ColumnStorage)
	u.deltaColumns = make(map[string]int)
	u.deletions.Reset()
	u.srState = COLD
	// the rest of the unmarshalling is done in the caller because u.t is nil in the moment
	return nil
}
func (u *storageShard) load(t *table) {
	u.t = t
	// mark columns for lazy loading (caller must hold u.mu.Lock)
	for _, col := range u.t.Columns {
		u.columns[col.Name] = nil
	}

	if t.PersistencyMode == Safe || t.PersistencyMode == Logged {
		// Replaying the log mutates inserts/deletions; caller holds u.mu.Lock
		var log chan interface{}
		log, u.logfile = u.t.schema.persistence.ReplayLog(u.uuid.String())
		numEntriesRestored := 0
		for logentry := range log {
			numEntriesRestored++
			switch l := logentry.(type) {
			case LogEntryDelete:
				u.deletions.Set(uint(l.idx), true) // mark deletion
			case LogEntryUndelete:
				u.deletions.Set(uint(l.idx), false)
			case LogEntryInsert:
				u.insertDatasetFromLog(l.cols, l.values)
			case LogEntryInsertHidden:
				firstRecid := u.main_count + uint32(len(u.inserts))
				u.insertDatasetFromLog(l.cols, l.values)
				for i := range l.values {
					u.deletions.Set(uint(firstRecid+uint32(i)), true)
				}
			default:
				panic("unknown log sequence: " + fmt.Sprint(l))
			}
		}
		if numEntriesRestored > 0 {
			fmt.Println("restoring delta storage from database "+u.t.schema.Name+" shard "+u.uuid.String()+":", numEntriesRestored, "entries")
		}
		// Reconstruct Auto_increment counter from replayed delta rows so that
		// cross-connection INSERT sequences never re-use IDs after server restart.
		for _, col := range t.Columns {
			if !col.AutoIncrement {
				continue
			}
			colIdx, ok := u.deltaColumns[col.Name]
			if !ok {
				break
			}
			var maxVal uint64
			for _, row := range u.inserts {
				if colIdx < len(row) && !row[colIdx].IsNil() {
					if v := uint64(scm.ToInt(row[colIdx])); v > maxVal {
						maxVal = v
					}
				}
			}
			if maxVal+1 > t.Auto_increment {
				t.Auto_increment = maxVal + 1
			}
			break // only one AUTO_INCREMENT column per table
		}
	}
}

func (u *storageShard) schemaColumn(colName string) *column {
	for _, c := range u.t.Columns {
		if c.Name == colName {
			return c
		}
	}
	return nil
}

func (u *storageShard) makeComputedColumnProxy(colName string, col *column) ColumnStorage {
	if col == nil || len(col.OrcSortCols) == 0 {
		return nil
	}
	return &StorageComputeProxy{
		delta:     make(map[uint32]scm.Scmer),
		shard:     u,
		colName:   colName,
		count:     u.main_count,
		isOrdered: true,
	}
}

func (u *storageShard) attachColumnRuntime(colName string, columnstorage ColumnStorage) ColumnStorage {
	if blob, ok := columnstorage.(*OverlayBlob); ok {
		blob.SetSchema(u.t.schema)
	}
	col := u.schemaColumn(colName)
	if proxy, ok := columnstorage.(*StorageComputeProxy); ok {
		// StorageComputeProxy persists only durable state. Runtime bindings to
		// the owning shard/column must be restored when the storage is loaded.
		proxy.shard = u
		proxy.colName = colName
		proxy.sessionKeys = extractSessionKeys(proxy.computor)
		if col != nil && len(col.OrcSortCols) > 0 {
			proxy.isOrdered = true
		}
		return proxy
	}
	// ORC columns are a runtime contract, not a best-effort cache. Older or
	// partially rebuilt shards may still have a plain placeholder storage on
	// disk (`StorageSparse`, `const[nil]`, ...). Rehydrate those columns into an
	// ordered proxy so readers/rebuilds never publish the placeholder as a real
	// user-visible value column.
	if proxy := u.makeComputedColumnProxy(colName, col); proxy != nil {
		return proxy
	}
	return columnstorage
}

// Most loaded storages are already runtime-ready. Only blob/computed/ORC-backed
// columns require a re-attach step on access. Keeping plain columns on the
// read-only fast path avoids self-deadlocks when a scan callback recursively
// scans the same shard while already holding u.mu.RLock().
func (u *storageShard) needsRuntimeAttach(colName string, columnstorage ColumnStorage) bool {
	if columnstorage == nil {
		return false
	}
	if _, ok := columnstorage.(*OverlayBlob); ok {
		return true
	}
	if _, ok := columnstorage.(*StorageComputeProxy); ok {
		return true
	}
	col := u.schemaColumn(colName)
	return col != nil && isRuntimeComputedColumn(col)
}

// ensureColumnLoaded loads a single column storage when first accessed.
// If alreadyLocked is true, the caller must hold u.mu.Lock() and no locks
// are taken inside this function. Otherwise, it acquires the appropriate
// locks internally.
func (u *storageShard) ensureColumnLoaded(colName string, alreadyLocked bool) ColumnStorage {
	// Shared critical path which assumes u.mu is held (write).
	loadLocked := func() ColumnStorage {
		cs, present := u.columns[colName]
		if !present {
			panic("Column does not exist: `" + u.t.schema.Name + "`.`" + u.t.Name + "`.`" + colName + "`")
		}
		if cs != nil {
			cs = u.attachColumnRuntime(colName, cs)
			u.columns[colName] = cs
			return cs
		}
		if u.t.PersistencyMode == Memory || u.t.PersistencyMode == Cache {
			if proxy := u.makeComputedColumnProxy(colName, u.schemaColumn(colName)); proxy != nil {
				u.columns[colName] = proxy
			} else {
				u.columns[colName] = new(StorageSparse)
			}
			return u.columns[colName]
		}
		release := acquireLoadSlot()
		defer release()
		f := u.t.schema.persistence.ReadColumn(u.uuid.String(), colName)
		var magicbyte uint8
		if err := binary.Read(f, binary.LittleEndian, &magicbyte); err != nil {
			u.columns[colName] = new(StorageSparse)
			return u.columns[colName]
		}
		fmt.Println("loading storage "+u.t.schema.Name+" shard "+u.uuid.String()+" column "+colName+" of type", magicbyte)
		columnstorage := reflect.New(storages[magicbyte]).Interface().(ColumnStorage)
		cnt := columnstorage.Deserialize(f)
		f.Close()
		if uint32(cnt) > u.main_count {
			u.main_count = uint32(cnt)
		}
		columnstorage = u.attachColumnRuntime(colName, columnstorage)
		u.columns[colName] = columnstorage
		return columnstorage
	}

	if alreadyLocked {
		return loadLocked()
	}

	// Fast path under RLock
	u.mu.RLock()
	cs, present := u.columns[colName]
	u.mu.RUnlock()
	if !present {
		panic("Column does not exist: `" + u.t.schema.Name + "`.`" + u.t.Name + "`.`" + colName + "`")
	}
	if cs != nil {
		if !u.needsRuntimeAttach(colName, cs) {
			return cs
		}
		u.mu.Lock()
		defer u.mu.Unlock()
		cs = u.attachColumnRuntime(colName, u.columns[colName])
		u.columns[colName] = cs
		return cs
	}
	// Acquire write lock and load
	u.mu.Lock()
	defer u.mu.Unlock()
	// Re-check after acquiring lock
	if cs = u.columns[colName]; cs != nil {
		return cs
	}
	return loadLocked()
}

// getColumnStorageRLocked returns the column storage without re-acquiring mu.
// The caller MUST already hold u.mu.RLock(). This avoids the reentrant-RLock
// deadlock that occurs when a concurrent writer is waiting for mu.Lock()
// (Go's write-preferring RWMutex queues new readers behind pending writers).
// Panics if the column is missing or not yet loaded from disk.
func (u *storageShard) getColumnStorageRLocked(colName string) ColumnStorage {
	cs, present := u.columns[colName]
	if !present {
		panic("Column does not exist: `" + u.t.schema.Name + "`.`" + u.t.Name + "`.`" + colName + "`")
	}
	if cs == nil {
		panic("Column not loaded while shard RLocked: `" + u.t.schema.Name + "`.`" + u.t.Name + "`.`" + colName + "`")
	}
	return cs
}

// getColumnStorageOrPanic returns a stable pointer to a column's storage.
// It never reads u.columns without holding the shard lock and loads on demand.
func (u *storageShard) getColumnStorageOrPanic(colName string) ColumnStorage {
	if u.hasWriteOwner() {
		return u.getColumnStorageOrPanicEx(colName, true, nil)
	}
	if tx := CurrentTx(); tx != nil && tx.HasShardWrite(u) {
		return u.getColumnStorageOrPanicEx(colName, true, tx)
	}
	// try under read lock
	u.mu.RLock()
	cs, present := u.columns[colName]
	u.mu.RUnlock()
	if !present {
		// The column may be missing from this shard's map because it was created
		// after the shard (e.g. a PShards shard created during repartition before
		// all columns were installed, or a race between CreateColumn and scan).
		// If the column exists in the table schema, add it as StorageSparse so
		// the scan can proceed without panicking.
		for _, c := range u.t.Columns {
			if c.Name == colName {
				u.mu.Lock()
				if _, present2 := u.columns[colName]; !present2 {
					u.columns[colName] = new(StorageSparse)
				}
				cs2 := u.columns[colName]
				u.mu.Unlock()
				return cs2
			}
		}
		panic("Column does not exist: `" + u.t.schema.Name + "`.`" + u.t.Name + "`.`" + colName + "`")
	}
	if cs != nil {
		return cs
	}
	return u.ensureColumnLoaded(colName, false)
}

func (u *storageShard) getColumnStorageOrPanicEx(colName string, alreadyLocked bool, currentTx *TxContext) ColumnStorage {
	if alreadyLocked {
		cs, present := u.columns[colName]
		if !present {
			// Shards can lag behind table schema changes (for example when a
			// column was added after the shard was created and the old shard was
			// later reloaded from disk). Mirror the fallback from the unlocked
			// path so scans can still proceed under the shard write lock.
			if col := u.schemaColumn(colName); col != nil {
				if proxy := u.makeComputedColumnProxy(colName, col); proxy != nil {
					u.columns[colName] = proxy
				} else {
					u.columns[colName] = new(StorageSparse)
				}
				return u.columns[colName]
			}
			panic("Column does not exist: `" + u.t.schema.Name + "`.`" + u.t.Name + "`.`" + colName + "`")
		}
		if cs != nil {
			if !u.needsRuntimeAttach(colName, cs) {
				return cs
			}
			cs = u.attachColumnRuntime(colName, cs)
			u.columns[colName] = cs
			return cs
		}
		return u.ensureColumnLoaded(colName, true)
	}
	if currentTx == nil {
		currentTx = CurrentTx()
	}
	if currentTx != nil && currentTx.HasShardWrite(u) {
		return u.getColumnStorageOrPanicEx(colName, true, currentTx)
	}
	u.mu.RLock()
	cs, present := u.columns[colName]
	u.mu.RUnlock()
	if !present {
		if col := u.schemaColumn(colName); col != nil {
			u.mu.Lock()
			if _, present2 := u.columns[colName]; !present2 {
				if proxy := u.makeComputedColumnProxy(colName, col); proxy != nil {
					u.columns[colName] = proxy
				} else {
					u.columns[colName] = new(StorageSparse)
				}
			}
			cs2 := u.columns[colName]
			u.mu.Unlock()
			return cs2
		}
		panic("Column does not exist: `" + u.t.schema.Name + "`.`" + u.t.Name + "`.`" + colName + "`")
	}
	if cs != nil {
		if !u.needsRuntimeAttach(colName, cs) {
			return cs
		}
		u.mu.Lock()
		defer u.mu.Unlock()
		cs = u.attachColumnRuntime(colName, u.columns[colName])
		u.columns[colName] = cs
		return cs
	}
	return u.ensureColumnLoaded(colName, false)
}

// ensureMainCount guarantees main_count is initialized by loading one column if needed.
func (u *storageShard) ensureMainCount(alreadyLocked bool) {
	if u.main_count != 0 {
		return
	}
	// Load the first column (if not yet loaded); Deserialize will set main_count.
	if alreadyLocked {
		for _, c := range u.t.Columns {
			cs, ok := u.columns[c.Name]
			if ok && cs == nil {
				u.ensureColumnLoaded(c.Name, true)
				if u.main_count != 0 {
					return
				}
			}
		}
		return
	}
	for _, c := range u.t.Columns {
		u.mu.RLock()
		cs, ok := u.columns[c.Name]
		u.mu.RUnlock()
		if ok && cs == nil {
			u.ensureColumnLoaded(c.Name, false)
			if u.main_count != 0 {
				return
			}
		}
	}
}

// SharedResource impl for shard with lazy load
func (s *storageShard) GetState() SharedState { return s.srState }
func (s *storageShard) GetRead() func() {
	s.ensureLoaded()
	// Ensure main_count is initialized by loading at least one column
	s.ensureMainCount(false)
	if s.srState == COLD {
		s.srState = SHARED
	}
	atomic.StoreUint64(&s.lastAccessed, uint64(time.Now().UnixNano()))
	return func() {}
}
func (s *storageShard) GetExclusive() func() {
	s.ensureLoaded()
	s.srState = WRITE
	atomic.StoreUint64(&s.lastAccessed, uint64(time.Now().UnixNano()))
	return func() {}
}

func (s *storageShard) ensureLoaded() {
	if s.srState != COLD {
		return
	}
	// pre-free memory before loading shard from disk
	GlobalCache.CheckPressure(int64(len(s.t.Columns)) * int64(Settings.ShardSize) * 16)
	// double-check under lock to prevent concurrent map writes in load()
	s.mu.Lock()
	if s.srState != COLD {
		s.mu.Unlock()
		return
	}
	// materialize shard from disk (load expects caller to hold mu.Lock)
	s.load(s.t)
	// memory engine shards stay WRITE to bypass LRU later
	if s.t.PersistencyMode == Memory {
		s.srState = WRITE
	} else {
		s.srState = SHARED
	}
	s.mu.Unlock()
	atomic.StoreUint64(&s.lastAccessed, uint64(time.Now().UnixNano()))
	// register with CacheManager (skip Memory-engine shards and temp tables)
	if s.t.PersistencyMode == Cache && !strings.HasPrefix(s.t.Name, ".") {
		GlobalCache.AddItem(s, int64(s.ComputeSize()), TypeCacheEntry, cacheShardCleanup, shardLastUsed, nil)
	} else if s.t.PersistencyMode != Memory && !strings.HasPrefix(s.t.Name, ".") {
		GlobalCache.AddItem(s, int64(s.ComputeSize()), TypeShard, shardCleanup, shardLastUsed, nil)
	}
}

// shardCleanup is called by the CacheManager when evicting a shard.
// Returns false if the shard lock cannot be acquired (non-blocking).
func shardCleanup(ptr any, freedByType *[numEvictableTypes]int64) bool {
	s := ptr.(*storageShard)
	// Safety: MEMORY-engine shards must NEVER be evicted (data would be lost permanently).
	if s.t != nil && s.t.PersistencyMode == Memory {
		return false
	}
	if !s.mu.TryLock() {
		return false // shard is in use, skip eviction
	}
	// Sloppy/Logged shards with pending deltas must be flushed to disk before eviction.
	// If there are deltas, we can't evict now — a rebuild is needed first.
	if len(s.inserts) > 0 || s.deletions.Count() > 0 {
		s.mu.Unlock()
		return false // has unflushed deltas, skip eviction (rebuild will flush them)
	}
	// remove indexes from CacheManager (recursive free)
	for _, idx := range s.Indexes {
		GlobalCache.removeInternal(idx, freedByType)
		idx.evict(evictFull, 0, freedByType)
	}
	// release column storage (deregister compressed string dicts first)
	for col := range s.columns {
		if str, ok := s.columns[col].(*StorageString); ok && str.compressed {
			GlobalCache.removeInternal(str, freedByType)
		}
		s.columns[col] = nil
	}
	s.srState = COLD
	s.mu.Unlock()
	return true
}

func shardLastUsed(ptr any) time.Time {
	return time.Unix(0, int64(atomic.LoadUint64(&ptr.(*storageShard).lastAccessed)))
}

// cacheShardCleanup is called by the CacheManager when evicting a Cache-engine shard.
// Unlike shardCleanup, it forcibly clears in-flight deltas since there is no disk backing.
func cacheShardCleanup(ptr any, freedByType *[numEvictableTypes]int64) bool {
	s := ptr.(*storageShard)
	if !s.mu.TryLock() {
		return false // shard is in use, retry later
	}
	// remove indexes from CacheManager (recursive free)
	for _, idx := range s.Indexes {
		GlobalCache.removeInternal(idx, freedByType)
		idx.evict(evictFull, 0, freedByType)
	}
	// clear in-memory data (no disk backing to flush to)
	s.inserts = nil
	s.deletions.Reset()
	s.main_count = 0
	for col := range s.columns {
		if str, ok := s.columns[col].(*StorageString); ok && str.compressed {
			GlobalCache.removeInternal(str, freedByType)
		}
		s.columns[col] = nil
	}
	// COLD: on next access ensureLoaded re-initialises as empty and re-registers
	s.srState = COLD
	s.mu.Unlock()
	return true
}

func NewShard(t *table) *storageShard {
	result := new(storageShard)
	result.uuid, _ = uuid.NewRandom()
	result.t = t
	result.columns = make(map[string]ColumnStorage)
	result.deltaColumns = make(map[string]int)
	result.deletions.Reset()
	for _, column := range t.Columns {
		result.columns[column.Name] = new(StorageSparse)
	}
	if t.PersistencyMode == Safe || t.PersistencyMode == Logged {
		result.logfile = result.t.schema.persistence.OpenLog(result.uuid.String())
	}
	// Newly created shards are live/writable, not cold
	result.srState = WRITE
	return result
}

func (t *storageShard) Count() uint32 {
	return t.main_count + uint32(len(t.inserts)) - uint32(t.deletions.Count())
}

func currentGoroutineID() uint64 {
	id, ok := gls.GetGoroutineId()
	if !ok {
		return 0
	}
	return uint64(id)
}

func (t *storageShard) enterWriteOwner() {
	goid := currentGoroutineID()
	if goid == 0 {
		return
	}
	t.writeOwnMu.Lock()
	if t.writeOwners == nil {
		t.writeOwners = make(map[uint64]uint32)
	}
	t.writeOwners[goid]++
	t.writeOwnMu.Unlock()
}

func (t *storageShard) exitWriteOwner() {
	goid := currentGoroutineID()
	if goid == 0 {
		return
	}
	t.writeOwnMu.Lock()
	if d := t.writeOwners[goid]; d <= 1 {
		delete(t.writeOwners, goid)
	} else {
		t.writeOwners[goid] = d - 1
	}
	t.writeOwnMu.Unlock()
}

// SetShardWriteLocked updates the mapper's lock state after an external lock release.
func (m *ShardMapReducer) SetShardWriteLocked(locked bool) {
	m.shardWriteLocked = locked
}

func (t *storageShard) hasWriteOwner() bool {
	goid := currentGoroutineID()
	if goid == 0 {
		return false
	}
	t.writeOwnMu.Lock()
	defer t.writeOwnMu.Unlock()
	return t.writeOwners[goid] > 0
}

// hasWriteOwnerForTx uses the explicit transaction lock state on SQL paths.
// The goroutine owner fallback remains necessary for internal callers that do
// not carry a transaction context.
func (t *storageShard) hasWriteOwnerForTx(currentTx *TxContext) bool {
	if currentTx != nil {
		return currentTx.HasShardWrite(t)
	}
	return t.hasWriteOwner()
}

// rowValueByRecidLocked reads a column value for a recid. Caller must hold t.mu.
func (t *storageShard) rowValueByRecidLocked(recid uint32, col string) scm.Scmer {
	if recid < t.main_count {
		cs := t.getColumnStorageOrPanicEx(col, true, nil)
		return cs.GetValue(recid)
	}
	return t.getDelta(int(recid-t.main_count), col)
}

// resolveVisiblePrimaryRecidLocked maps a stale/deleted recid to the currently
// visible row with the same PRIMARY key. Caller must hold t.mu.
func (t *storageShard) resolveVisiblePrimaryRecidLocked(staleRecid uint32) (uint32, bool) {
	var primaryCols []string
	for _, uk := range t.t.Unique {
		if uk.Id == "PRIMARY" {
			primaryCols = uk.Cols
			break
		}
	}
	if len(primaryCols) == 0 {
		return 0, false
	}

	key := make([]scm.Scmer, len(primaryCols))
	for i, col := range primaryCols {
		key[i] = t.rowValueByRecidLocked(staleRecid, col)
	}

	limit := t.main_count + uint32(len(t.inserts))
	for recid := limit; recid > 0; recid-- {
		candidate := recid - 1
		if candidate == staleRecid || t.deletions.Get(uint(candidate)) {
			continue
		}
		match := true
		for i, col := range primaryCols {
			if !scm.Equal(t.rowValueByRecidLocked(candidate, col), key[i]) {
				match = false
				break
			}
		}
		if match {
			return candidate, true
		}
	}
	return 0, false
}

func (t *storageShard) UpdateFunction(idx uint32, withTrigger bool, alreadyLocked bool, currentTx *TxContext) func(...scm.Scmer) scm.Scmer {
	return t.UpdateFunctionBatch(idx, withTrigger, alreadyLocked, nil, nil, currentTx)
}

// sameUpdateValue keeps Scheme's useful numeric/coercive equality while
// preserving SQL NULL as a distinct stored value. In particular, NULL -> 0 is
// a real row change even though (equal? nil 0) is true in Scheme.
func sameUpdateValue(a, b scm.Scmer) bool {
	if a.IsNil() || b.IsNil() {
		return a.IsNil() && b.IsNil()
	}
	return scm.Equal(a, b)
}

func isRuntimeComputedColumn(colDesc *column) bool {
	return len(colDesc.OrcSortCols) > 0 || !colDesc.Computor.IsNil() || len(colDesc.ComputorInputCols) > 0
}

func (t *storageShard) UpdateFunctionBatch(idx uint32, withTrigger bool, alreadyLocked bool, batch *triggerBatch, deletedRows *uint64, currentTx *TxContext) func(...scm.Scmer) scm.Scmer {
	// returns a callback with which you can delete or update an item
	return func(a ...scm.Scmer) scm.Scmer {
		//fmt.Println("update/delete", a)
		// FK checks are enforced via auto-generated system triggers (see createforeignkey)

		result := false // result = true when update was possible; false if there was a RESTRICT
		targetIdx := idx
		var maintenanceNext *storageShard
		var maintenanceExtraDeletes []uint32
		if len(a) > 0 {
			// update command
			var triggerOldRow, triggerNewRow dataset // for AFTER UPDATE triggers
			var newRecid uint32                      // recid of the newly inserted row (for tx undo)
			var dualWriteCols []string               // captured for dual-write forwarding
			var dualWriteRow [][]scm.Scmer           // captured for dual-write forwarding
			// Build a row in schema-column order from a delta-ordered row buffer.
			schemaRowFromDelta := func(deltaRow dataset) dataset {
				row := make(dataset, len(t.t.Columns))
				for i, colDesc := range t.t.Columns {
					if colidx, ok := t.deltaColumns[colDesc.Name]; ok && colidx < len(deltaRow) {
						row[i] = deltaRow[colidx]
					} else {
						row[i] = scm.NewNil()
					}
				}
				return row
			}
			func() {
				if !alreadyLocked {
					t.mu.Lock()         // write lock
					defer t.mu.Unlock() // write lock
				}

				// For non-ACID updates, callbacks may race on stale recids that are
				// already deleted by a concurrent writer. Follow to the currently
				// visible row with the same PRIMARY key.
				if (currentTx == nil || currentTx.Mode != TxACID) && t.deletions.Get(uint(targetIdx)) {
					followed := false
					for attempt := 0; attempt < 256; attempt++ {
						if followIdx, ok := t.resolveVisiblePrimaryRecidLocked(targetIdx); ok {
							targetIdx = followIdx
							followed = true
							break
						}
						// If we own the lock from caller context we cannot release it
						// here; in this case, resolving failed for this recid.
						if alreadyLocked {
							break
						}
						// Another writer may be between temporary delete and
						// insert publication. Yield and retry.
						if attempt < 255 {
							t.mu.Unlock()
							runtime.Gosched()
							t.mu.Lock()
						}
					}
					if !followed {
						return
					}
				}

				// update statement -> also perform an insert
				// TODO: check if we can do in-place editing in the delta storage (if idx > t.main_count)
				changes := mustScmerSlice(a[0], "update changes")
				uniqueColsSet := make(map[string]bool)
				for _, uk := range t.t.Unique {
					for _, col := range uk.Cols {
						uniqueColsSet[col] = true
					}
				}
				uniqueColsTouched := false
				// Build a complete row using schema columns (not only currently
				// loaded shard columns). Otherwise UPDATEs on a subset of columns
				// can write delta rows that miss PK/other fields.
				d2 := make([]scm.Scmer, 0, len(t.t.Columns))
				for _, colDesc := range t.t.Columns {
					if isRuntimeComputedColumn(colDesc) {
						continue
					}
					k := colDesc.Name
					cs := t.getColumnStorageOrPanicEx(k, true, nil)
					colidx, ok := t.deltaColumns[k]
					if !ok {
						colidx = len(t.deltaColumns)
						t.deltaColumns[k] = colidx
					}
					for len(d2) <= colidx {
						d2 = append(d2, scm.NewNil())
					}
					if targetIdx < t.main_count {
						d2[colidx] = cs.GetValue(targetIdx)
					} else {
						d2[colidx] = t.getDelta(int(targetIdx-t.main_count), k)
					}
				}
				buildPayload := func() ([]string, []scm.Scmer) {
					pCols := make([]string, 0, len(t.t.Columns))
					pRow := make([]scm.Scmer, 0, len(t.t.Columns))
					for _, colDesc := range t.t.Columns {
						if isRuntimeComputedColumn(colDesc) {
							continue
						}
						colName := colDesc.Name
						pos, ok := t.deltaColumns[colName]
						if !ok || pos >= len(d2) {
							continue
						}
						pCols = append(pCols, colName)
						pRow = append(pRow, d2[pos])
					}
					return pCols, pRow
				}
				// now d2 contains the old row values
				oldDeltaRow := append(dataset{}, d2...)
				// copy slice for triggers before modifying (scheme values are immutable, but Go slice is modified)
				if withTrigger && len(t.t.Triggers) > 0 {
					triggerOldRow = schemaRowFromDelta(d2)
				}
				for j := 0; j < len(changes); j += 2 {
					colidx, ok := t.deltaColumns[scm.String(changes[j])]
					if !ok {
						panic("UPDATE on invalid column: " + scm.String(changes[j]))
					}
					newVal := changes[j+1]
					// apply type sanitizer
					for _, colDesc := range t.t.Columns {
						if colDesc.Name == scm.String(changes[j]) && colDesc.sanitizer != nil {
							newVal = colDesc.sanitizer(newVal)
							break
						}
					}
					if !sameUpdateValue(d2[colidx], newVal) {
						d2[colidx] = newVal
						result = true // mark that something has changed
						if uniqueColsSet[scm.String(changes[j])] {
							uniqueColsTouched = true
						}
					}
				}
				// Execute BEFORE UPDATE triggers (can modify d2)
				if withTrigger && triggerOldRow != nil {
					newSchemaRow := schemaRowFromDelta(d2)
					if alreadyLocked {
						func() {
							t.mu.Unlock()
							defer t.mu.Lock()
							newSchemaRow = t.t.ExecuteBeforeUpdateTriggers(triggerOldRow, newSchemaRow)
						}()
					} else {
						newSchemaRow = t.t.ExecuteBeforeUpdateTriggers(triggerOldRow, newSchemaRow)
					}
					// Write trigger-mutated schema values back to delta row layout.
					for i, colDesc := range t.t.Columns {
						if colidx, ok := t.deltaColumns[colDesc.Name]; ok && colidx < len(d2) && i < len(newSchemaRow) {
							d2[colidx] = newSchemaRow[i]
						}
					}
					// BEFORE triggers may change typed/NOT NULL columns; sanitize again.
					for _, colDesc := range t.t.Columns {
						if colDesc.sanitizer == nil {
							continue
						}
						if pos, ok := t.deltaColumns[colDesc.Name]; ok && pos < len(d2) {
							d2[pos] = colDesc.sanitizer(d2[pos])
						}
					}
					// Recheck if anything changed after trigger modifications
					result = false
					for i, v := range d2 {
						if i < len(oldDeltaRow) && !sameUpdateValue(oldDeltaRow[i], v) {
							result = true
							break
						}
					}
					if !result && len(d2) != len(oldDeltaRow) {
						result = true
					}
				}

				if !result { // only do a write if something changed
					return // leave inner func to unlock
				}

				// Only re-run global unique collision scans when at least one
				// unique key column value actually changed.
				uniqueColsChanged := uniqueColsTouched
				if t.t.Unique != nil && len(t.t.Triggers) > 0 {
					// BEFORE UPDATE triggers may touch unique columns even if they
					// are not present in the explicit UPDATE assignment list.
					uniqueColsChanged = false
					for _, uk := range t.t.Unique {
						for _, ucol := range uk.Cols {
							colidx, ok := t.deltaColumns[ucol]
							if ok && colidx < len(oldDeltaRow) && colidx < len(d2) && !sameUpdateValue(oldDeltaRow[colidx], d2[colidx]) {
								uniqueColsChanged = true
								break
							}
						}
						if uniqueColsChanged {
							break
						}
					}
				}

				// save new row for triggers (d2 now contains new values)
				if withTrigger && len(t.t.Triggers) > 0 {
					triggerNewRow = schemaRowFromDelta(d2)
				}

				acidMode := currentTx != nil && currentTx.Mode == TxACID
				uniqueCheckNeeded := t.t.Unique != nil && uniqueColsChanged

				// unique constraint checking
				if uniqueCheckNeeded {
					payloadCols, payloadRow := buildPayload()
					wasDeletedBefore := t.deletions.Get(uint(targetIdx))
					t.deletions.Set(uint(targetIdx), true) // mark as deleted temporarily for unique check
					t.mu.Unlock()                          // release write lock, so the scan can be performed
					t.t.ProcessUniqueCollision(payloadCols, [][]scm.Scmer{payloadRow}, false, func(values [][]scm.Scmer) {
						t.mu.Lock() // start write lock
					}, nil, func(errmsg string, data []scm.Scmer) {
						t.mu.Lock() // start write lock
						if !wasDeletedBefore {
							t.deletions.Set(uint(targetIdx), false) // restore only if we changed visibility here
						}
						panic(sqldb.NewSQLError1(1062, "23000", "Duplicate entry in table %s: %s", t.t.Name, errmsg))
					}, 0)
				} else {
					// Keep old row visible until after we inserted the replacement in
					// non-ACID mode. This avoids transient "row disappears" gaps that
					// make concurrent UPDATE scans miss rows.
					if acidMode {
						t.deletions.Set(uint(targetIdx), true) // staged delete in ACID overlay flow
					}
				}

				payloadCols, payloadRow := buildPayload()
				newRecid = t.main_count + uint32(len(t.inserts))
				t.insertDataset(payloadCols, [][]scm.Scmer{payloadRow}, nil, currentTx)
				if !acidMode && !uniqueCheckNeeded {
					// Atomic visibility switch under shard write lock:
					// make new row hidden first, then delete old, then publish new.
					t.deletions.Set(uint(newRecid), true)
					t.deletions.Set(uint(targetIdx), true)
					t.deletions.Set(uint(newRecid), false)
					// Maintain the non-ACID invariant: at most one visible row per
					// PRIMARY key. Under heavy concurrent UPDATE chains, stale targets
					// can leave older versions visible; collapse them here.
					var primaryCols []string
					for _, uk := range t.t.Unique {
						if uk.Id == "PRIMARY" {
							primaryCols = uk.Cols
							break
						}
					}
					if len(primaryCols) > 0 {
						pkVals := make([]scm.Scmer, len(primaryCols))
						for i, col := range primaryCols {
							if pos, ok := t.deltaColumns[col]; ok && pos < len(d2) {
								pkVals[i] = d2[pos]
							} else {
								pkVals[i] = scm.NewNil()
							}
						}
						limit := t.main_count + uint32(len(t.inserts))
						for recid := uint32(0); recid < limit; recid++ {
							if recid == newRecid || t.deletions.Get(uint(recid)) {
								continue
							}
							match := true
							for i, col := range primaryCols {
								if !scm.Equal(t.rowValueByRecidLocked(recid, col), pkVals[i]) {
									match = false
									break
								}
							}
							if match {
								t.deletions.Set(uint(recid), true)
								if next := t.nextForMaintenanceLocked(&recid); next != nil {
									maintenanceNext = next
									maintenanceExtraDeletes = append(maintenanceExtraDeletes, recid)
								}
							}
						}
					}
				}
				// Capture for dual-write forwarding (cols/d2 are closure-local).
				// Use repartitionDualWriteActive (set after Phase B snapshot)
				// so rows already in the snapshot are not dual-written again.
				if t.t.repartitionDualWriteActive.Load() && t.t.isRepartitionSource(t) {
					dualWriteCols = payloadCols
					dualWriteRow = [][]scm.Scmer{payloadRow}
				}

				if currentTx != nil && currentTx.Mode == TxACID {
					// Check if old row was staged by this tx (in UndeleteMask)
					st := currentTx.getShardTx(t)
					wasStaged := st != nil && st.UndeleteMask.Get(uint(targetIdx))
					if wasStaged {
						// Row was staged by this tx → remove from UndeleteMask.
						// Keep shard.deletions[targetIdx]=true (already globally hidden).
						// Don't add to DeleteMask (not a pre-existing row).
						st.UndeleteMask.Set(uint(targetIdx), false)
					} else {
						// Pre-existing committed row → undo temporary global deletion
						t.deletions.Set(uint(targetIdx), false)
						currentTx.AddToDeleteMask(t, targetIdx)
					}
					// Stage new version: hide globally, add to undelete mask
					t.deletions.Set(uint(newRecid), true)
					t.rollbackProtected.Set(uint(newRecid), true)
					currentTx.AddToUndeleteMask(t, newRecid)
					// Only log the insert (delete applied at commit)
					if (t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged) && t.logfile != nil {
						t.logfile.Write(LogEntryInsertHidden{payloadCols, [][]scm.Scmer{payloadRow}})
					}
				} else {
					// Cursor-stability / no-tx: existing behavior
					if currentTx != nil {
						t.rollbackProtected.Set(uint(targetIdx), true)
					}
					if (t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged) && t.logfile != nil {
						t.logfile.Write(LogEntryDelete{targetIdx})
						t.logfile.Write(LogEntryInsert{payloadCols, [][]scm.Scmer{payloadRow}})
					}
				}
				maintenanceNext = t.nextForMaintenanceLocked(&targetIdx)
			}()
			// Dual-write: forward the new row to the secondary shard set
			if result && dualWriteRow != nil {
				t.t.dualWriteInsertFromOld(t, newRecid, dualWriteCols, dualWriteRow, currentTx)
			}
			// transaction bookkeeping + deferred sync
			// (shard is already registered via OpenMapReducer — no per-row RegisterTouchedShard)
			if result {
				if tx := currentTx; tx != nil {
					switch tx.Mode {
					case TxCursorStability:
						tx.LogDelete(t, targetIdx)
						tx.LogInsert(t, newRecid)
					}
				} else if t.t.PersistencyMode == Safe && t.logfile != nil {
					defer t.logfile.Sync()
				}
			}
			if withTrigger && triggerOldRow != nil {
				if alreadyLocked {
					func() {
						t.mu.Unlock()
						defer t.mu.Lock()
						t.t.ExecuteTriggers(AfterUpdate, triggerOldRow, triggerNewRow)
					}()
				} else {
					t.t.ExecuteTriggers(AfterUpdate, triggerOldRow, triggerNewRow)
				}
			}
		} else {
			// delete
			var triggerDeletedRow dataset // for BEFORE/AFTER DELETE trigger

			// capture row data for triggers before deletion (outside lock for BEFORE trigger)
			if withTrigger && len(t.t.Triggers) > 0 {
				if alreadyLocked {
					triggerDeletedRow = make(dataset, len(t.t.Columns))
					for i, col := range t.t.Columns {
						cs := t.getColumnStorageOrPanicEx(col.Name, true, nil)
						if idx < t.main_count {
							triggerDeletedRow[i] = cs.GetValue(idx)
						} else {
							triggerDeletedRow[i] = t.getDelta(int(idx-t.main_count), col.Name)
						}
					}
				} else {
					t.mu.RLock()
					triggerDeletedRow = make(dataset, len(t.t.Columns))
					for i, col := range t.t.Columns {
						cs := t.getColumnStorageOrPanicEx(col.Name, true, nil)
						if idx < t.main_count {
							triggerDeletedRow[i] = cs.GetValue(idx)
						} else {
							triggerDeletedRow[i] = t.getDelta(int(idx-t.main_count), col.Name)
						}
					}
					t.mu.RUnlock()
				}

				// Execute BEFORE DELETE triggers (can abort delete by returning false)
				beforeDeleteOk := true
				if alreadyLocked {
					func() {
						t.mu.Unlock()
						defer t.mu.Lock()
						beforeDeleteOk = t.t.ExecuteBeforeDeleteTriggers(triggerDeletedRow)
					}()
				} else {
					beforeDeleteOk = t.t.ExecuteBeforeDeleteTriggers(triggerDeletedRow)
				}
				if !beforeDeleteOk {
					return scm.NewBool(false) // trigger aborted delete
				}
			}

			if tx := currentTx; tx != nil && tx.Mode == TxACID {
				// Check if row was staged by this tx
				st2 := tx.getShardTx(t)
				wasStaged := st2 != nil && st2.UndeleteMask.Get(uint(idx))
				if wasStaged {
					// Row was staged by this tx → remove from UndeleteMask.
					// Keep shard.deletions[idx]=true (already globally hidden).
					st2.UndeleteMask.Set(uint(idx), false)
				} else {
					// Pre-existing committed row → add to delete mask
					tx.AddToDeleteMask(t, idx)
				}
				// shard already registered via OpenMapReducer
				if t.nextReady.Load() {
					maintenanceNext = t.loadNext()
				}
				result = true
			} else {
				func() {
					if !alreadyLocked {
						t.mu.Lock()         // write lock
						defer t.mu.Unlock() // write lock
					}

					t.deletions.Set(uint(idx), true) // mark as deleted
					if currentTx != nil {
						t.rollbackProtected.Set(uint(idx), true)
					}
					if (t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged) && t.logfile != nil {
						t.logfile.Write(LogEntryDelete{idx})
					}
					maintenanceNext = t.nextForMaintenanceLocked(&idx)
					result = true
				}()
				// deferred sync (shard already registered via OpenMapReducer)
				if tx != nil {
					tx.LogDelete(t, idx)
				} else if t.t.PersistencyMode == Safe && t.logfile != nil {
					defer t.logfile.Sync()
				}
			}
			if withTrigger && triggerDeletedRow != nil {
				if batch != nil {
					// Batch mode: collect row, trigger fires later via Flush()
					batch.Add(triggerDeletedRow)
				} else if alreadyLocked {
					func() {
						t.mu.Unlock()
						defer t.mu.Lock()
						t.t.ExecuteTriggers(AfterDelete, triggerDeletedRow, nil)
					}()
				} else {
					t.t.ExecuteTriggers(AfterDelete, triggerDeletedRow, nil)
				}
			}
		}
		if result {
			if len(a) == 0 && deletedRows != nil {
				*deletedRows++
			}
			// Dual-write: forward DELETE to PShards during repartition
			if t.t.repartitionDualWriteActive.Load() && t.t.isRepartitionSource(t) {
				t.t.dualWriteDelete(t, idx, currentTx)
			}
			if maintenanceNext != nil {
				// Propagate to the rebuild successor shard via the stable
				// old→new recid translation published by rebuild().
				if len(a) > 0 {
					t.propagateUpdateToNext(maintenanceNext, targetIdx, currentTx, a...)
				} else {
					t.propagateDeleteToNext(maintenanceNext, idx, currentTx)
				}
				for _, deletedRecid := range maintenanceExtraDeletes {
					t.propagateDeleteToNext(maintenanceNext, deletedRecid, nil)
				}
			}
		}
		return scm.NewBool(result) // maybe instead return UpdateFunction for newly inserted item??
	}
}

// propagateUpdateToNext uses the old→new recid translation built by rebuild()
// and extended by mirrored delta inserts.
func (t *storageShard) propagateUpdateToNext(next *storageShard, oldRecid uint32, currentTx *TxContext, a ...scm.Scmer) {
	targetRecid, ok := t.translateNextRecid(oldRecid)
	if !ok {
		// The successor stays write-locked until rebuild() has published the
		// initial translation. Wait for that point once, then retry the map.
		next.mu.Lock()
		next.mu.Unlock()
		targetRecid, ok = t.translateNextRecid(oldRecid)
	}
	if ok {
		next.UpdateFunction(targetRecid, false, false, currentTx)(a...)
	}
}

// propagateDeleteToNext uses the old→new recid translation built by rebuild()
// and extended by mirrored delta inserts. This avoids the CountUntil bug during
// batch deletes without falling back to an O(n) PK scan.
func (t *storageShard) propagateDeleteToNext(next *storageShard, oldRecid uint32, currentTx *TxContext) {
	targetRecid, ok := t.translateNextRecid(oldRecid)
	if !ok {
		next.mu.Lock()
		next.mu.Unlock()
		targetRecid, ok = t.translateNextRecid(oldRecid)
	}
	if !ok {
		return
	}
	next.UpdateFunction(targetRecid, false, false, currentTx)()
}

func (t *storageShard) ColumnReaderTx(tx *TxContext, col string) func(uint32) scm.Scmer {
	cstorage := t.getColumnStorageOrPanic(col)
	reader := newCachedColumnReaderTx(cstorage, tx)
	return func(idx uint32) scm.Scmer {
		if idx < t.main_count {
			return reader.GetValue(idx)
		} else {
			return t.getDelta(int(idx-t.main_count), col)
		}
	}
}

// breakSentinel is the panic value injected by $break pseudo-column closures.
// scan_order.go catches this type to implement early-exit (LIMIT semantics inside ORC).
type breakSentinel struct{}

type mapArgGetter func(uint32, uint32) scm.Scmer

// ShardMapReducer pre-allocates args and applies map+reduce over batches of record IDs.
// Local implementation of the streaming MapReducer pattern (see todos/cluster.md §15.7).
// Stream() partitions recid batches into main/delta runs and dispatches to
// processMainBlock/processDeltaBlock – tight loops suitable for JIT compilation.
// For remote shards, Stream() will be backed by an RPC returning the accumulator per batch.
type ShardMapReducer struct {
	shard           *storageShard
	currentTx       *TxContext
	acidMode        bool
	mainGetters     []mapArgGetter
	deltaGetters    []mapArgGetter
	mainCols        []ColumnStorage        // direct main storage access (nil for $update/$invalidate/$increment cols)
	mainBulkReaders []ColumnReader         // physical map columns gathered once per Stream main-record run
	mainBulkValues  []scm.Scmer            // reusable row-major buffer for every physical map column
	colNames        []string               // column names for delta getDelta access
	isUpdate        []bool                 // true for $update columns
	isInvalidate    []bool                 // true for $invalidate: columns
	invalidateProxy []*StorageComputeProxy // proxy per $invalidate col (nil if not found)
	isIncrement     []bool                 // true for $increment: columns
	incrementProxy  []*StorageComputeProxy // proxy per $increment col (nil if not found)
	isSet           []bool                 // true for $set: columns
	setProxy        []*StorageComputeProxy // proxy per $set col (nil if not found)
	hasSetCol       bool
	isBreak         []bool // true for $break column
	hasBreakCol     bool
	// tagClosure hoisted fn ptrs — allocated once per mapper, reused per row
	setClosureFn   []*func(uint32, ...scm.Scmer) scm.Scmer // per $set col
	incrClosureFn  []*func(uint32, ...scm.Scmer) scm.Scmer // per $increment col
	invClosureFn   []*func(uint32, ...scm.Scmer) scm.Scmer // per $invalidate col
	noopClosureFn  *func(uint32, ...scm.Scmer) scm.Scmer   // shared noop
	breakClosureFn *func(uint32, ...scm.Scmer) scm.Scmer   // shared break
	args           []scm.Scmer                             // pre-allocated args buffer
	reduceArgs     []scm.Scmer                             // pre-allocated binary reducer arguments
	mapFn          func(...scm.Scmer) scm.Scmer
	reduceFn       func(...scm.Scmer) scm.Scmer
	mapScmer       scm.Scmer     // original Scmer for network serialization
	deleteBatch    *triggerBatch // when set, DELETE triggers are batched instead of per-row
	deletedRows    uint64        // applied DELETEs, published once when the mapper flushes
	// Batched side effects: collected during scan, flushed after lock release.
	// $increment calls are aggregated per (proxy, recid) → one update per unique target.
	incrementBatch  map[*StorageComputeProxy]map[uint32]scm.Scmer // proxy → recid → accumulated delta
	invalidateBatch map[*StorageComputeProxy]bool                 // proxies to InvalidateAll after scan
	reduceScmer     scm.Scmer                                     // original Scmer for network serialization
	mainCount       uint32
	hasUpdateCol    bool
	hasIncrementCol bool
	// shardWriteLocked is true when the caller already holds shard.mu (write lock)
	// and registered write ownership before opening this mapper. When true,
	// processMainBlock/processDeltaBlock must NOT try to re-acquire the lock.
	shardWriteLocked bool
}

// MapOne evaluates the mapper for one already-visible record. Ordered scans use
// this outside shard locks for predicates that must run after global ordering.
func (m *ShardMapReducer) MapOne(id uint32) scm.Scmer {
	getters := m.mainGetters
	if id >= m.mainCount {
		getters = m.deltaGetters
	}
	for i, getter := range getters {
		m.args[i] = getter(id, 0)
	}
	return m.mapFn(m.args...)
}

// OpenMapReducer creates a MapReducer for the given columns. Column readers and
// main storage references are built once; the args buffer is pre-allocated.
// mapFn and reduceFn are stored as Scmer for future network serialization;
// OptimizeProcToSerialFunction is called here (TODO: replace with JIT compilation).
func (t *storageShard) OpenMapReducer(cols []string, mapFn scm.Scmer, reduceFn scm.Scmer, alreadyLocked bool, stride int, batchdata []scm.Scmer, currentTx *TxContext) *ShardMapReducer {
	mr := &ShardMapReducer{
		shard:            t,
		currentTx:        currentTx,
		acidMode:         currentTx != nil && currentTx.Mode == TxACID,
		mainGetters:      make([]mapArgGetter, len(cols)),
		deltaGetters:     make([]mapArgGetter, len(cols)),
		mainCols:         make([]ColumnStorage, len(cols)),
		mainBulkReaders:  make([]ColumnReader, len(cols)),
		colNames:         cols,
		args:             make([]scm.Scmer, len(cols)),
		reduceArgs:       make([]scm.Scmer, 2),
		mapFn:            scm.OptimizeProcToSerialFunction(mapFn),
		reduceFn:         scm.OptimizeProcToSerialFunction(reduceFn),
		mapScmer:         mapFn,
		reduceScmer:      reduceFn,
		mainCount:        t.main_count,
		shardWriteLocked: alreadyLocked,
	}
	needsMutationMetadata := false
	for _, col := range cols {
		if col == "$update" || col == "$break" || len(col) >= 4 && col[:4] == "NEW." ||
			len(col) > 12 && col[:12] == "$invalidate:" ||
			len(col) > 11 && col[:11] == "$increment:" ||
			len(col) > 5 && col[:5] == "$set:" {
			needsMutationMetadata = true
			break
		}
	}
	if needsMutationMetadata {
		mr.isUpdate = make([]bool, len(cols))
		mr.isInvalidate = make([]bool, len(cols))
		mr.invalidateProxy = make([]*StorageComputeProxy, len(cols))
		mr.isIncrement = make([]bool, len(cols))
		mr.incrementProxy = make([]*StorageComputeProxy, len(cols))
		mr.isSet = make([]bool, len(cols))
		mr.setProxy = make([]*StorageComputeProxy, len(cols))
		mr.isBreak = make([]bool, len(cols))
		mr.setClosureFn = make([]*func(uint32, ...scm.Scmer) scm.Scmer, len(cols))
		mr.incrClosureFn = make([]*func(uint32, ...scm.Scmer) scm.Scmer, len(cols))
		mr.invClosureFn = make([]*func(uint32, ...scm.Scmer) scm.Scmer, len(cols))
	}
	hasDeleteTriggers := false
	for _, tr := range t.t.Triggers {
		if tr.Timing == AfterDelete {
			hasDeleteTriggers = true
			break
		}
	}
	for i, col := range cols {
		if subidx, ok := parseBatchPseudoColName(col); ok {
			mainSubidx := subidx
			getter := func(id uint32, batchid uint32) scm.Scmer {
				return batchdata[int(batchid)*stride+mainSubidx]
			}
			mr.mainGetters[i] = getter
			mr.deltaGetters[i] = getter
			continue
		}
		if col == "$recset_contains" {
			fnptr := recSetContainsClosure(t)
			getter := func(id uint32, batchid uint32) scm.Scmer {
				return scm.NewClosure(fnptr, id)
			}
			mr.mainGetters[i] = getter
			mr.deltaGetters[i] = getter
			continue
		}
		if col == "$update" {
			mr.isUpdate[i] = true
			mr.hasUpdateCol = true
			if hasDeleteTriggers {
				mr.deleteBatch = t.t.BeginTriggerBatch(AfterDelete, true)
			}
		} else if len(col) >= 4 && col[:4] == "NEW." {
			mr.isUpdate[i] = true // NEW. columns always return nil
			mr.hasUpdateCol = true
		} else if len(col) > 12 && col[:12] == "$invalidate:" {
			mr.isInvalidate[i] = true
			cacheColName := col[12:]
			cs := t.getColumnStorageOrPanicEx(cacheColName, alreadyLocked, currentTx)
			if proxy, ok := cs.(*StorageComputeProxy); ok {
				mr.invalidateProxy[i] = proxy
			}
		} else if len(col) > 11 && col[:11] == "$increment:" {
			mr.isIncrement[i] = true
			mr.hasIncrementCol = true
			cacheColName := col[11:]
			cs := t.getColumnStorageOrPanicEx(cacheColName, alreadyLocked, currentTx)
			if proxy, ok := cs.(*StorageComputeProxy); ok {
				mr.incrementProxy[i] = proxy
			}
		} else if len(col) > 5 && col[:5] == "$set:" {
			mr.isSet[i] = true
			mr.hasSetCol = true
			cacheColName := col[5:]
			cs := t.getColumnStorageOrPanicEx(cacheColName, alreadyLocked, currentTx)
			if proxy, ok := cs.(*StorageComputeProxy); ok {
				mr.setProxy[i] = proxy
			}
		} else if col == "$break" {
			mr.isBreak[i] = true
			mr.hasBreakCol = true
		} else {
			mr.mainCols[i] = t.getColumnStorageOrPanicEx(col, alreadyLocked, currentTx)
		}
	}
	// Pre-allocate tagClosure fn ptrs (hoisted, one per pseudo-col type per column).
	// These are allocated once here so processMainBlock/processDeltaBlock can use
	// NewClosure(ptr, effectiveID) per row without any heap allocation.
	if needsMutationMetadata {
		noopFn := func(id uint32, args ...scm.Scmer) scm.Scmer { return scm.NewBool(true) }
		mr.noopClosureFn = &noopFn
		breakFn := func(id uint32, args ...scm.Scmer) scm.Scmer { panic(breakSentinel{}) }
		mr.breakClosureFn = &breakFn
	}
	for i := range cols {
		if needsMutationMetadata && mr.isSet[i] {
			if proxy := mr.setProxy[i]; proxy != nil {
				fn := func(id uint32, args ...scm.Scmer) scm.Scmer {
					if len(args) > 0 {
						proxy.SetValue(id, args[0])
					}
					return scm.NewBool(true)
				}
				mr.setClosureFn[i] = &fn
			}
		}
		if needsMutationMetadata && mr.isIncrement[i] {
			if proxy := mr.incrementProxy[i]; proxy != nil {
				// Batch increments: aggregate per (proxy, recid) and flush after scan
				if mr.incrementBatch == nil {
					mr.incrementBatch = make(map[*StorageComputeProxy]map[uint32]scm.Scmer)
				}
				if mr.incrementBatch[proxy] == nil {
					mr.incrementBatch[proxy] = make(map[uint32]scm.Scmer)
				}
				proxyBatch := mr.incrementBatch[proxy]
				fn := func(id uint32, args ...scm.Scmer) scm.Scmer {
					if len(args) > 0 {
						if existing, ok := proxyBatch[id]; ok {
							// Aggregate: add deltas
							if existing.IsInt() && args[0].IsInt() {
								proxyBatch[id] = scm.NewInt(existing.Int() + args[0].Int())
							} else if existing.IsFloat() && args[0].IsFloat() {
								proxyBatch[id] = scm.NewFloat(existing.Float() + args[0].Float())
							} else {
								proxyBatch[id] = args[0] // can't aggregate, last wins
							}
						} else {
							proxyBatch[id] = args[0]
						}
					}
					return scm.NewBool(true)
				}
				mr.incrClosureFn[i] = &fn
			}
		}
		if needsMutationMetadata && mr.isInvalidate[i] {
			if proxy := mr.invalidateProxy[i]; proxy != nil {
				// Batch invalidations: mark proxy for InvalidateAll after scan
				if mr.invalidateBatch == nil {
					mr.invalidateBatch = make(map[*StorageComputeProxy]bool)
				}
				fn := func(id uint32, args ...scm.Scmer) scm.Scmer {
					mr.invalidateBatch[proxy] = true
					return scm.NewBool(true)
				}
				mr.invClosureFn[i] = &fn
			}
		}
		if mr.mainGetters[i] != nil {
			continue
		}
		if needsMutationMetadata && mr.isInvalidate[i] {
			if fnptr := mr.invClosureFn[i]; fnptr != nil {
				mr.mainGetters[i] = func(id uint32, batchid uint32) scm.Scmer {
					return scm.NewClosure(fnptr, id)
				}
			} else {
				mr.mainGetters[i] = func(id uint32, batchid uint32) scm.Scmer {
					return scm.NewClosure(mr.noopClosureFn, id)
				}
			}
			mr.deltaGetters[i] = func(id uint32, batchid uint32) scm.Scmer {
				return scm.NewClosure(mr.noopClosureFn, id)
			}
			continue
		}
		if needsMutationMetadata && mr.isIncrement[i] {
			if fnptr := mr.incrClosureFn[i]; fnptr != nil {
				mr.mainGetters[i] = func(id uint32, batchid uint32) scm.Scmer {
					return scm.NewClosure(fnptr, id)
				}
			} else {
				mr.mainGetters[i] = func(id uint32, batchid uint32) scm.Scmer {
					return scm.NewClosure(mr.noopClosureFn, id)
				}
			}
			mr.deltaGetters[i] = func(id uint32, batchid uint32) scm.Scmer {
				return scm.NewClosure(mr.noopClosureFn, id)
			}
			continue
		}
		if needsMutationMetadata && mr.isSet[i] {
			if fnptr := mr.setClosureFn[i]; fnptr != nil {
				getter := func(id uint32, batchid uint32) scm.Scmer {
					return scm.NewClosure(fnptr, id)
				}
				mr.mainGetters[i] = getter
				mr.deltaGetters[i] = getter
			} else {
				getter := func(id uint32, batchid uint32) scm.Scmer {
					return scm.NewClosure(mr.noopClosureFn, id)
				}
				mr.mainGetters[i] = getter
				mr.deltaGetters[i] = getter
			}
			continue
		}
		if needsMutationMetadata && mr.isBreak[i] {
			getter := func(id uint32, batchid uint32) scm.Scmer {
				return scm.NewClosure(mr.breakClosureFn, id)
			}
			mr.mainGetters[i] = getter
			mr.deltaGetters[i] = getter
			continue
		}
		if needsMutationMetadata && mr.isUpdate[i] {
			getter := func(id uint32, batchid uint32) scm.Scmer {
				return scm.NewFunc(mr.shard.UpdateFunctionBatch(id, true, mr.shardWriteLocked, mr.deleteBatch, &mr.deletedRows, mr.currentTx))
			}
			mr.mainGetters[i] = getter
			mr.deltaGetters[i] = getter
			continue
		}
		mainCol := mr.mainCols[i]
		mainReader := newCachedColumnReaderTx(mainCol, mr.currentTx)
		mr.mainBulkReaders[i] = mainReader
		colName := mr.colNames[i]
		mr.mainGetters[i] = func(id uint32, batchid uint32) scm.Scmer {
			return mainReader.GetValue(id)
		}
		if _, isProxy := mainCol.(*StorageComputeProxy); isProxy {
			mr.deltaGetters[i] = func(id uint32, batchid uint32) scm.Scmer {
				return mainReader.GetValue(id)
			}
		} else {
			mr.deltaGetters[i] = func(id uint32, batchid uint32) scm.Scmer {
				return mr.shard.getDelta(int(id-mr.mainCount), colName)
			}
		}
	}
	// Register shard for deferred fsync once, not per row.
	if mr.hasUpdateCol {
		if tx := mr.currentTx; tx != nil {
			tx.RegisterTouchedShard(t)
		}
	}
	return mr
}

func (m *ShardMapReducer) prefetchMainColumns(recids []uint32) {
	width := len(m.mainBulkReaders)
	if width == 0 {
		return
	}
	hasReader := false
	for _, reader := range m.mainBulkReaders {
		if reader != nil {
			hasReader = true
			break
		}
	}
	if !hasReader {
		m.mainBulkValues = m.mainBulkValues[:0]
		return
	}
	needed := len(recids) * width
	if cap(m.mainBulkValues) < needed {
		m.mainBulkValues = make([]scm.Scmer, needed)
	} else {
		m.mainBulkValues = m.mainBulkValues[:needed]
	}
	for i, reader := range m.mainBulkReaders {
		if reader == nil {
			continue
		}
		reader.GetValueMulti(recids, m.mainBulkValues[i:], width)
	}
}

func withTxSession(currentTx *TxContext, fn func() scm.Scmer) scm.Scmer {
	if currentTx == nil || currentTx.Session.IsNil() {
		return fn()
	}
	return scm.WithSession(currentTx.Session, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		return fn()
	}))
}

// Stream applies map+reduce over a batch of record IDs. The recid list is
// partitioned order-preserving into runs of main-storage IDs and delta IDs.
// batchids is optional and, when present, must align 1:1 with recids.
func (m *ShardMapReducer) Stream(acc scm.Scmer, recids []uint32, batchids []uint32) scm.Scmer {
	i := 0
	n := len(recids)
	for i < n {
		j := i + 1
		if recids[i] < m.mainCount {
			for j < n && recids[j] < m.mainCount {
				j++
			}
			if batchids == nil {
				acc = m.processMainBlock(acc, recids[i:j])
			} else {
				acc = m.processMainBlockBatch(acc, recids[i:j], batchids[i:j])
			}
		} else {
			for j < n && recids[j] >= m.mainCount {
				j++
			}
			if batchids == nil {
				acc = m.processDeltaBlock(acc, recids[i:j])
			} else {
				acc = m.processDeltaBlockBatch(acc, recids[i:j], batchids[i:j])
			}
		}
		i = j
	}
	return acc
}

func (m *ShardMapReducer) reduce(acc scm.Scmer, value scm.Scmer) scm.Scmer {
	// A direct variadic call with two values allocates its temporary argument
	// slice on every row because reduceFn may retain it.
	m.reduceArgs[0] = acc
	m.reduceArgs[1] = value
	return m.reduceFn(m.reduceArgs...)
}

// processMainBlock is a tight loop over main-storage records – no branching
// on main vs delta, direct ColumnStorage.GetValue calls. JIT candidate.
func (m *ShardMapReducer) processMainBlock(acc scm.Scmer, recids []uint32) scm.Scmer {
	// Per-row lock: acquire write lock only around each row's mutation,
	// not for the whole batch. This allows nested read scans (e.g. EXISTS
	// inside UPDATE on the same table) to acquire RLock between rows.
	needsPerRowLock := (m.hasUpdateCol || m.hasIncrementCol || m.hasSetCol) && !m.shardWriteLocked
	if !needsPerRowLock && !m.hasUpdateCol && !m.hasIncrementCol && !m.hasSetCol {
		m.prefetchMainColumns(recids)
	}
	bulkWidth := len(m.mainBulkReaders)
	hasBulkValues := bulkWidth > 0 && len(m.mainBulkValues) == len(recids)*bulkWidth
	for rowIndex, id := range recids {
		func() {
			effectiveID := id
			rowLocked := false
			// Acquire write lock per-row for mutation scans
			if needsPerRowLock {
				m.shard.mu.Lock()
				m.shard.enterWriteOwner()
				rowLocked = true
				defer func() {
					if rowLocked {
						m.shard.exitWriteOwner()
						m.shard.mu.Unlock()
					}
				}()
			}
			if m.hasUpdateCol || m.hasIncrementCol || m.hasSetCol {
				if !m.acidMode {
					if m.shard.deletions.Get(uint(effectiveID)) {
						followedID, ok := m.shard.resolveVisiblePrimaryRecidLocked(effectiveID)
						if !ok {
							return
						}
						effectiveID = followedID
					}
				}
			}
			for i, getter := range m.mainGetters {
				if hasBulkValues && m.mainBulkReaders[i] != nil {
					m.args[i] = m.mainBulkValues[rowIndex*bulkWidth+i]
				} else {
					m.args[i] = getter(effectiveID, 0)
				}
			}
			// Release write lock before mapFn: allows nested scans on same shard.
			// $update closures will re-acquire the lock when called.
			if needsPerRowLock {
				m.shard.exitWriteOwner()
				m.shard.mu.Unlock()
				rowLocked = false
			}
			acc = m.reduce(acc, m.mapFn(m.args...))
		}()
	}
	return acc
}

func (m *ShardMapReducer) processMainBlockBatch(acc scm.Scmer, recids []uint32, batchids []uint32) scm.Scmer {
	needsPerRowLock := (m.hasUpdateCol || m.hasIncrementCol || m.hasSetCol) && !m.shardWriteLocked
	if !needsPerRowLock && !m.hasUpdateCol && !m.hasIncrementCol && !m.hasSetCol {
		m.prefetchMainColumns(recids)
	}
	bulkWidth := len(m.mainBulkReaders)
	hasBulkValues := bulkWidth > 0 && len(m.mainBulkValues) == len(recids)*bulkWidth
	for rowidx, id := range recids {
		func() {
			effectiveID := id
			batchid := batchids[rowidx]
			rowLocked := false
			if needsPerRowLock {
				m.shard.mu.Lock()
				m.shard.enterWriteOwner()
				rowLocked = true
				defer func() {
					if rowLocked {
						m.shard.exitWriteOwner()
						m.shard.mu.Unlock()
					}
				}()
			}
			if m.hasUpdateCol || m.hasIncrementCol || m.hasSetCol {
				if !m.acidMode {
					if m.shard.deletions.Get(uint(effectiveID)) {
						followedID, ok := m.shard.resolveVisiblePrimaryRecidLocked(effectiveID)
						if !ok {
							return
						}
						effectiveID = followedID
					}
				}
			}
			for i, getter := range m.mainGetters {
				if hasBulkValues && m.mainBulkReaders[i] != nil {
					m.args[i] = m.mainBulkValues[rowidx*bulkWidth+i]
				} else {
					m.args[i] = getter(effectiveID, batchid)
				}
			}
			if needsPerRowLock {
				m.shard.exitWriteOwner()
				m.shard.mu.Unlock()
				rowLocked = false
			}
			acc = m.reduce(acc, m.mapFn(m.args...))
		}()
	}
	return acc
}

// processDeltaBlock handles delta-storage records via getDelta. JIT candidate.
func (m *ShardMapReducer) processDeltaBlock(acc scm.Scmer, recids []uint32) scm.Scmer {
	// Same hoisting as processMainBlock (see comment there).
	needsPerRowLock := (m.hasUpdateCol || m.hasIncrementCol || m.hasSetCol) && !m.shardWriteLocked
	for _, id := range recids {
		func() {
			effectiveID := id
			rowLocked := false
			if needsPerRowLock {
				m.shard.mu.Lock()
				m.shard.enterWriteOwner()
				rowLocked = true
				defer func() {
					if rowLocked {
						m.shard.exitWriteOwner()
						m.shard.mu.Unlock()
					}
				}()
			}
			if m.hasUpdateCol || m.hasIncrementCol || m.hasSetCol {
				if !m.acidMode {
					if m.shard.deletions.Get(uint(effectiveID)) {
						followedID, ok := m.shard.resolveVisiblePrimaryRecidLocked(effectiveID)
						if !ok {
							return
						}
						effectiveID = followedID
					}
				}
			}
			for i, getter := range m.deltaGetters {
				m.args[i] = getter(effectiveID, 0)
			}
			if needsPerRowLock {
				m.shard.exitWriteOwner()
				m.shard.mu.Unlock()
				rowLocked = false
			}
			acc = m.reduce(acc, m.mapFn(m.args...))
		}()
	}
	return acc
}

func (m *ShardMapReducer) processDeltaBlockBatch(acc scm.Scmer, recids []uint32, batchids []uint32) scm.Scmer {
	needsPerRowLock := (m.hasUpdateCol || m.hasIncrementCol || m.hasSetCol) && !m.shardWriteLocked
	for rowidx, id := range recids {
		func() {
			effectiveID := id
			batchid := batchids[rowidx]
			rowLocked := false
			if needsPerRowLock {
				m.shard.mu.Lock()
				m.shard.enterWriteOwner()
				rowLocked = true
				defer func() {
					if rowLocked {
						m.shard.exitWriteOwner()
						m.shard.mu.Unlock()
					}
				}()
			}
			if m.hasUpdateCol || m.hasIncrementCol || m.hasSetCol {
				if !m.acidMode {
					if m.shard.deletions.Get(uint(effectiveID)) {
						followedID, ok := m.shard.resolveVisiblePrimaryRecidLocked(effectiveID)
						if !ok {
							return
						}
						effectiveID = followedID
					}
				}
			}
			for i, getter := range m.deltaGetters {
				m.args[i] = getter(effectiveID, batchid)
			}
			if needsPerRowLock {
				m.shard.exitWriteOwner()
				m.shard.mu.Unlock()
				rowLocked = false
			}
			acc = m.reduce(acc, m.mapFn(m.args...))
		}()
	}
	return acc
}

// Close releases resources held by the MapReducer. Does NOT flush trigger
// batches — that must happen after all shard locks are released.
// Call FlushTriggerBatch() separately after the scan completes.
func (m *ShardMapReducer) Close() {
}

// FlushSideEffects flushes all batched side effects (triggers, increments,
// invalidations). Must be called AFTER the scan completes and all shard
// locks are released, to avoid deadlocks.
func (m *ShardMapReducer) FlushSideEffects() {
	// 1. Flush batched invalidations (cheapest: just set bits)
	for proxy := range m.invalidateBatch {
		proxy.InvalidateAll()
	}
	m.invalidateBatch = nil

	// 2. Flush batched increments (aggregated per proxy+recid)
	for proxy, batch := range m.incrementBatch {
		for recid, delta := range batch {
			proxy.IncrementalUpdate(recid, delta)
		}
	}
	m.incrementBatch = nil

	// 3. Flush batched DELETE triggers (may trigger cascading scans)
	if m.deleteBatch != nil {
		m.deleteBatch.Flush()
		m.deleteBatch = nil
	}
	if m.deletedRows > 0 {
		m.shard.t.adjustPlannerRows(-int64(m.deletedRows))
		m.deletedRows = 0
	}
}

func (t *storageShard) Insert(columns []string, values [][]scm.Scmer, alreadyLocked bool, onFirstInsertId func(int64), isIgnore bool) uint32 {
	ss := scm.GetCurrentSessionState()
	// Check table-level user lock (LOCK TABLES): writes block under any lock.
	// Always call waitTableLock — it handles other-session blocking and
	// owner-write-under-READ-lock error in one place.
	if t.t.hasTableLock() {
		t.t.waitTableLock(ss, true)
	}
	beforeInsertTriggers := t.t.GetTriggers(BeforeInsert)
	currentTx := CurrentTx()
	if len(beforeInsertTriggers) > 0 {
		preparedColumns := make([][]string, 0, len(values))
		preparedRows := make([][]scm.Scmer, 0, len(values))
		for _, row := range values {
			rowColumns, preparedRow, ok := t.t.executeBeforeInsertTriggerRow(columns, row, isIgnore)
			if !ok {
				continue
			}
			sanitized := t.t.sanitizeInsertRows(rowColumns, [][]scm.Scmer{preparedRow}, isIgnore)
			if len(sanitized) == 0 {
				continue
			}
			preparedColumns = append(preparedColumns, rowColumns)
			preparedRows = append(preparedRows, sanitized[0])
		}
		if len(preparedRows) == 0 {
			return 0
		}
		if !alreadyLocked {
			t.lockForMutation(ss)
			defer t.mu.Unlock()
		}
		firstNewRecid := uint32(0)
		firstInsertId := onFirstInsertId
		for i, row := range preparedRows {
			recid := t.insertPreparedLocked(preparedColumns[i], [][]scm.Scmer{row}, firstInsertId, true, true, currentTx)
			if i == 0 {
				firstNewRecid = recid
			}
			if firstInsertId != nil {
				firstInsertId = nil
			}
		}
		return firstNewRecid
	}

	// Re-apply sanitizers after trigger-free INSERT input preparation.
	values = t.t.sanitizeInsertRows(columns, values, isIgnore)
	if len(values) == 0 {
		return 0 // all rows skipped by sanitizer in INSERT IGNORE mode
	}

	if !alreadyLocked {
		t.lockForMutation(ss)
		defer t.mu.Unlock()
	}
	firstNewRecid := t.insertPreparedLocked(columns, values, onFirstInsertId, true, true, currentTx)
	return firstNewRecid
}

// lockForMutation follows the table-lock-before-shard-lock order without a
// TOCTOU window. A lock that races the first check is visible in the atomic
// recheck after t.mu is acquired. Never wait for that owner while retaining
// t.mu: a cache initializer holding a READ table lock may need this shard to
// finish its snapshot before it can release the table lock.
func (t *storageShard) lockForMutation(ss *scm.SessionState) {
	for {
		if t.t.hasTableLock() {
			t.t.waitTableLock(ss, true)
		}
		t.mu.Lock()
		owner := t.t.tableLockOwner.Load()
		if t.t.tableLockState.Load() == 0 || owner == ss {
			return
		}
		t.mu.Unlock()
	}
}

func (t *storageShard) insertReplica(columns []string, values [][]scm.Scmer, alreadyLocked bool, currentTx *TxContext) uint32 {
	if len(values) == 0 {
		return 0
	}
	if !alreadyLocked {
		t.mu.Lock()
	}
	firstNewInsertIdx := len(t.inserts)
	firstNewRecid := t.insertPreparedLocked(columns, values, nil, false, false, currentTx)
	if next := t.nextForMaintenanceLocked(nil); next != nil {
		// A writer may still hold an older immutable table-topology snapshot
		// after more than one rebuild generation has been published. Replica
		// inserts therefore continue through completed successor generations,
		// without re-running triggers, unique checks, or repartition routing.
		payloadCols, payloadVals := t.materializedInsertedRowsLocked(firstNewInsertIdx)
		firstNextRecid := next.insertReplica(payloadCols, payloadVals, false, currentTx)
		t.recordNextInsertRange(firstNewRecid, firstNextRecid, len(payloadVals))
	}
	if !alreadyLocked {
		t.mu.Unlock()
	}
	return firstNewRecid
}

func (t *storageShard) materializedInsertedRowsLocked(firstNewInsertIdx int) ([]string, [][]scm.Scmer) {
	idx2col := make([]string, len(t.deltaColumns))
	for name, idx := range t.deltaColumns {
		if idx < len(idx2col) {
			idx2col[idx] = name
		}
	}
	newRows := t.inserts[firstNewInsertIdx:]
	logVals := make([][]scm.Scmer, len(newRows))
	for i, row := range newRows {
		rowCopy := make([]scm.Scmer, len(idx2col))
		copy(rowCopy, row)
		logVals[i] = rowCopy
	}
	return idx2col, logVals
}

func (t *storageShard) insertPreparedLocked(columns []string, values [][]scm.Scmer, onFirstInsertId func(int64), fireTriggers bool, propagateMaintenance bool, currentTx *TxContext) uint32 {
	// capture starting row index for undo logging
	firstNewRecid := t.main_count + uint32(len(t.inserts))
	firstNewInsertIdx := len(t.inserts) // for capturing actual rows after insertDataset fills auto-increment
	var triggerInsertRows []dataset
	t.insertDataset(columns, values, onFirstInsertId, currentTx)
	if fireTriggers && len(t.t.Triggers) > 0 {
		newRows := t.inserts[firstNewInsertIdx:]
		triggerInsertRows = make([]dataset, len(newRows))
		for i, deltaRow := range newRows {
			row := make(dataset, len(t.t.Columns))
			for j, colDesc := range t.t.Columns {
				if colidx, ok := t.deltaColumns[colDesc.Name]; ok && colidx < len(deltaRow) {
					row[j] = deltaRow[colidx]
				} else {
					row[j] = scm.NewNil()
				}
			}
			triggerInsertRows[i] = row
		}
	}
	needMaterializedRows := (t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged) && t.logfile != nil
	needMaterializedRows = needMaterializedRows || (propagateMaintenance && (t.loadNext() != nil || t.t.repartitionDualWriteActive.Load()))
	var payloadCols []string
	var payloadVals [][]scm.Scmer
	if needMaterializedRows {
		payloadCols, payloadVals = t.materializedInsertedRowsLocked(firstNewInsertIdx)
	}
	if (t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged) && t.logfile != nil {
		// Log the actual inserted rows (not the original columns/values) so that
		// auto-incremented IDs and column defaults are preserved across restarts.
		if currentTx != nil && currentTx.Mode == TxACID {
			t.logfile.Write(LogEntryInsertHidden{payloadCols, payloadVals})
		} else {
			t.logfile.Write(LogEntryInsert{payloadCols, payloadVals})
		}
	}
	if propagateMaintenance {
		if next := t.nextForMaintenanceLocked(nil); next != nil {
			// also insert into next storage
			firstNextRecid := next.insertReplica(payloadCols, payloadVals, false, currentTx)
			t.recordNextInsertRange(firstNewRecid, firstNextRecid, len(payloadVals))
		}
		if t.t.repartitionDualWriteActive.Load() && t.t.isRepartitionSource(t) {
			t.t.dualWriteInsertFromOld(t, firstNewRecid, payloadCols, payloadVals, currentTx)
		}
	}
	// transaction bookkeeping
	if tx := currentTx; tx != nil {
		switch tx.Mode {
		case TxACID:
			// ACID: hide rows globally, add to undelete mask so this tx can see them
			for i := range values {
				recid := firstNewRecid + uint32(i)
				t.deletions.Set(uint(recid), true)
				t.rollbackProtected.Set(uint(recid), true)
				tx.AddToUndeleteMask(t, recid)
			}
			tx.RegisterTouchedShard(t)
		case TxCursorStability:
			// Cursor-stability: log inserts for undo on rollback
			for i := range values {
				tx.LogInsert(t, firstNewRecid+uint32(i))
			}
			tx.RegisterTouchedShard(t)
		}
	} else if t.t.PersistencyMode == Safe && t.logfile != nil {
		t.logfile.Sync() // write barrier; no tx means immediate sync
	}
	// execute AFTER INSERT triggers outside the shard write lock, matching the
	// AFTER UPDATE/DELETE paths and avoiding lock inversion with computed-column
	// invalidation on shared keytables.
	if fireTriggers && len(t.t.Triggers) > 0 {
		func() {
			t.mu.Unlock()
			defer t.lockForMutation(scm.GetCurrentSessionState())
			for _, row := range triggerInsertRows {
				t.t.ExecuteTriggers(AfterInsert, nil, row)
			}
		}()
	}
	return firstNewRecid
}

// contract: must only be called inside full write mutex mu.Lock()
func (t *storageShard) insertDataset(columns []string, values [][]scm.Scmer, onFirstInsertId func(int64), currentTx *TxContext) {
	colidx := make([]int, len(columns))
	for i, col := range columns {
		// copy all dataset entries into packed array
		var ok bool
		colidx[i], ok = t.deltaColumns[col]
		if !ok {
			// acquire new column
			colidx[i] = len(t.deltaColumns)
			t.deltaColumns[col] = colidx[i]
		}
	}
	var Auto_increment uint64
	var hasAI bool
	var aiColIdx int = -1
	for _, c := range t.t.Columns {
		if c.AutoIncrement {
			hasAI = true
			t.t.mu.Lock() // auto increment with global table lock outside the loop for a batch
			Auto_increment = t.t.Auto_increment
			t.t.Auto_increment = t.t.Auto_increment + uint64(len(values)) // batch reservation of new IDs
			t.t.mu.Unlock()
		}
		if c.AutoIncrement || !c.Default.IsNil() {
			// column with default or auto increment -> also add to deltacolumns
			cidx, ok := t.deltaColumns[c.Name]
			if !ok {
				// add column to delta
				cidx = len(t.deltaColumns)
				t.deltaColumns[c.Name] = cidx
				colidx = append(colidx, cidx)
			}
			if c.AutoIncrement {
				aiColIdx = cidx
			}
		}
	}
	// if requested, notify the first assigned id once per statement
	if hasAI && onFirstInsertId != nil {
		onFirstInsertId(int64(Auto_increment) + 1)
		// do not call again for this shard; table-level wrapper ensures only first shard triggers
		onFirstInsertId = nil
	}

	for _, row := range values {
		newrow := make([]scm.Scmer, len(t.deltaColumns))
		for _, c := range t.t.Columns {
			if c.AutoIncrement {
				// fill auto_increment col (lock-free because the lock is outside the loop)
				cidx := t.deltaColumns[c.Name]
				Auto_increment++ // local increase
				newrow[cidx] = scm.NewInt(int64(Auto_increment))
			} else if !c.Default.IsNil() {
				// fill col with default
				cidx := t.deltaColumns[c.Name]
				newrow[cidx] = c.Default
			}
		}
		recid := uint32(len(t.inserts)) + t.main_count
		for j, colidx := range colidx {
			if j < len(row) {
				newrow[colidx] = row[j]
			}
		}
		t.inserts = append(t.inserts, newrow)

		// also notify indices
		for _, index := range t.Indexes {
			// add to delta indexes
			if len(index.sessionKeys) > 0 {
				index.markVariantsDirty()
				continue
			}
			index.mu.Lock()
			if index.baseState.deltaBtree != nil {
				index.baseState.deltaBtree.ReplaceOrInsert(indexPair{int(recid), newrow})
			}
			index.mu.Unlock()
		}
	}
	// If any row had an explicit AI value exceeding the reserved range, bump the counter.
	// Auto_increment (local) holds the base before reservation; auto-assigned IDs are
	// in [Auto_increment+1 .. Auto_increment+len(values)]. Any stored value beyond that
	// range was explicitly provided by the caller and must advance the counter.
	if hasAI && aiColIdx >= 0 {
		reservedTop := Auto_increment + uint64(len(values))
		var maxExplicit uint64
		for _, row := range t.inserts[len(t.inserts)-len(values):] {
			if aiColIdx < len(row) && !row[aiColIdx].IsNil() {
				if v := uint64(scm.ToInt(row[aiColIdx])); v > reservedTop && v > maxExplicit {
					maxExplicit = v
				}
			}
		}
		if maxExplicit > 0 {
			t.t.mu.Lock()
			if maxExplicit+1 > t.t.Auto_increment {
				t.t.Auto_increment = maxExplicit + 1
			}
			t.t.mu.Unlock()
		}
	}
	// Size tracking happens on rebuild only (computeSizeLocked gives accurate malloc-aware size).
	// For temp keytables we still do a cheap heuristic update here (they are never rebuilt).
	if strings.HasPrefix(t.t.Name, ".") {
		delta := int64(len(values)) * int64(len(t.deltaColumns)) * 16
		GlobalCache.UpdateSize(t.t, delta)
	}
}

// insertDatasetFromLog appends delta rows from a persisted log without applying
// defaults or auto-increment logic. Must only be called while holding t.mu.
func (t *storageShard) insertDatasetFromLog(columns []string, values [][]scm.Scmer) {
	// map provided column names to delta positions, extending deltaColumns if needed
	colidx := make([]int, len(columns))
	for i, col := range columns {
		if idx, ok := t.deltaColumns[col]; ok {
			colidx[i] = idx
		} else {
			idx := len(t.deltaColumns)
			t.deltaColumns[col] = idx
			colidx[i] = idx
		}
	}
	for _, row := range values {
		newrow := make([]scm.Scmer, len(t.deltaColumns))
		recid := uint32(len(t.inserts)) + t.main_count
		for j, pos := range colidx {
			if j < len(row) {
				newrow[pos] = row[j]
			}
		}
		t.inserts = append(t.inserts, newrow)

		// update delta indexes
		for _, index := range t.Indexes {
			if len(index.sessionKeys) > 0 {
				index.markVariantsDirty()
				continue
			}
			index.mu.Lock()
			if index.baseState.deltaBtree != nil {
				index.baseState.deltaBtree.ReplaceOrInsert(indexPair{int(recid), newrow})
			}
			index.mu.Unlock()
		}
	}
}

func (t *storageShard) GetRecordidForUnique(columns []string, values []scm.Scmer, currentTx *TxContext) (result uint32, present bool) {
	// Preload main storages and establish main_count without holding any shard lock
	t.ensureMainCount(false)
	mcols := make([]ColumnStorage, len(columns))
	for i, col := range columns {
		mcols[i] = t.getColumnStorageOrPanic(col)
	}

	// Build equality boundaries for the index lookup
	bounds := make(boundaries, len(columns))
	for i, col := range columns {
		bounds[i] = columnboundaries{col: col, matcher: EqualMatcher, lower: values[i], lowerInclusive: true, upper: values[i], upperInclusive: true}
	}
	lower, upperLast := indexFromBoundaries(bounds)

	// From here on, read under shard read lock for a consistent snapshot of deletions/inserts/deltaColumns
	t.mu.RLock()

	acidMode := currentTx != nil && currentTx.Mode == TxACID

	mainCount := t.main_count

	// Use iterateIndex for O(log n) lookup (builds index lazily if needed)
	// Small buffer for existence check: stop early after first match
	var buf [8]uint32
	t.iterateIndex(currentTx, bounds, lower, upperLast, len(t.inserts), buf[:], 1, nil, func(batch []uint32) bool {
		for _, idx := range batch {
			// Verify all columns match (iterateIndex may return superset for range boundaries)
			matched := true
			if idx < mainCount {
				// Main storage: use ColumnStorage
				for j, v := range values {
					if !scm.Equal(mcols[j].GetValue(idx), v) {
						matched = false
						break
					}
				}
			} else {
				// Delta storage: use getDelta
				for j, v := range values {
					if !scm.Equal(t.getDelta(int(idx-mainCount), columns[j]), v) {
						matched = false
						break
					}
				}
			}
			if !matched {
				continue
			}
			// Check visibility
			if acidMode {
				if currentTx.IsVisible(t, idx) {
					result = idx
					present = true
					return false
				}
			} else if !t.deletions.Get(uint(idx)) {
				result = idx
				present = true
				return false
			}
		}
		return true
	})

	t.mu.RUnlock()
	return
}

type filteredRowEstimate struct {
	rows       int64
	capped     bool
	examined   int64
	population string
	coverage   string
}

func (t *storageShard) EstimateFilteredRows(conditionCols []string, condition scm.Scmer, limit int, currentTx *TxContext) filteredRowEstimate {
	if limit <= 0 {
		limit = 1024
	}
	t.ensureMainCount(false)
	ccols := make([]ColumnStorage, len(conditionCols))
	cReaders := make([]ColumnReader, len(conditionCols))
	cNeedsTxReader := make([]bool, len(conditionCols))
	conditionGetters := make([]mapArgGetter, len(conditionCols))
	bounds := extractBoundaries(conditionCols, condition)
	lower, upperLast := indexFromBoundaries(bounds)
	recsetBoundaryCoversCondition := recSetHooksCoverCondition(bounds, lower, t.t, conditionCols, condition)
	for i, col := range conditionCols {
		if col == "$recset_contains" {
			fnptr := recSetContainsClosure(t)
			if recsetBoundaryCoversCondition {
				fnptr = recSetAlreadyMatchedClosure()
			}
			conditionGetters[i] = func(id uint32, _ uint32) scm.Scmer {
				return scm.NewClosure(fnptr, id)
			}
			continue
		}
		ccols[i] = t.getColumnStorageOrPanic(col)
		cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
		if proxy, ok := ccols[i].(*StorageComputeProxy); ok && proxy.hasSessionVariants() {
			cNeedsTxReader[i] = true
		}
	}

	conditionFn := scm.OptimizeProcToSerialFunction(condition)

	t.mu.RLock()
	defer t.mu.RUnlock()

	acidMode := currentTx != nil && currentTx.Mode == TxACID
	mainCount := t.main_count
	count := int64(0)
	sampled := int64(0)
	capped := false
	indexRestricted := false
	hookCandidates := int64(-1)
	hookUniverse := int64(0)
	cdataset := make([]scm.Scmer, len(conditionCols))

	var buf [256]uint32
	t.iterateIndex(currentTx, bounds, lower, upperLast, len(t.inserts), buf[:], 0, func(index *StorageIndex, active bool) {
		indexRestricted = active && len(lower) > 0
		if active {
			if candidates, universe, ok := index.estimateHookCandidates(currentTx, bounds); ok {
				hookCandidates = int64(candidates) + int64(len(t.inserts))
				hookUniverse = int64(universe) + int64(len(t.inserts))
			}
		}
	}, func(batch []uint32) bool {
		if hookCandidates >= 0 {
			return false
		}
		for _, idx := range batch {
			if acidMode {
				if !currentTx.IsVisible(t, idx) {
					continue
				}
			} else if t.deletions.Get(uint(idx)) {
				continue
			}
			sampled++
			if idx < mainCount {
				for i, c := range ccols {
					if getter := conditionGetters[i]; getter != nil {
						cdataset[i] = getter(idx, 0)
					} else if cNeedsTxReader[i] {
						cdataset[i] = cReaders[i].GetValue(idx)
					} else {
						cdataset[i] = c.GetValue(idx)
					}
				}
			} else {
				for i, col := range conditionCols {
					if getter := conditionGetters[i]; getter != nil {
						cdataset[i] = getter(idx, 0)
					} else if cNeedsTxReader[i] {
						cdataset[i] = cReaders[i].GetValue(idx)
					} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
						cdataset[i] = ccols[i].GetValue(idx)
					} else {
						cdataset[i] = t.getDelta(int(idx-mainCount), col)
					}
				}
			}
			if scm.ToBool(conditionFn(cdataset...)) {
				count++
				if count >= int64(limit) {
					capped = true
					return false
				}
			}
		}
		return true
	})
	if hookCandidates >= 0 {
		return filteredRowEstimate{
			rows:       hookCandidates,
			examined:   hookUniverse,
			population: "index_hook_candidates",
			coverage:   "upper_bound",
		}
	}

	population := "table_rows"
	if indexRestricted {
		population = "index_candidates"
	}
	coverage := "exact"
	if capped {
		if population == "table_rows" {
			coverage = "sampled"
		} else {
			coverage = "lower_bound"
		}
	}
	return filteredRowEstimate{
		rows:       count,
		capped:     capped,
		examined:   sampled,
		population: population,
		coverage:   coverage,
	}
}

func (t *storageShard) getDelta(idx int, col string) scm.Scmer {
	item := t.inserts[idx]
	colidx, ok := t.deltaColumns[col]
	if ok {
		if colidx < len(item) {
			return item[colidx]
		}
	}
	// Computed / ORC columns have no physical delta slot. Their value contract is
	// defined by the column runtime, so delta rows must read through the proxy
	// instead of silently degrading to NULL.
	if cs, ok := t.columns[col]; ok {
		if proxy, ok := cs.(*StorageComputeProxy); ok {
			return proxy.GetValue(t.main_count + uint32(idx))
		}
	}
	return scm.NewNil()
}

func (t *storageShard) RemoveFromDisk() {
	t.cleanupOnce.Do(func() {
		// close logfile
		if t.logfile != nil {
			t.logfile.Close()
		}
		// Release blob refcounts before removing column files.
		// Skip for COLD shards (columns not loaded) -- orphaned blobs will be cleaned by (clean).
		for _, col := range t.t.Columns {
			if cs, ok := t.columns[col.Name]; ok && cs != nil {
				if blob, ok := cs.(*OverlayBlob); ok {
					blob.ReleaseBlobs(uint(t.main_count))
				}
			}
		}
		for _, col := range t.t.Columns {
			t.t.schema.persistence.RemoveColumn(t.uuid.String(), col.Name)
		}
		t.t.schema.persistence.RemoveLog(t.uuid.String())
	})
}

// discardUnpublishedShard removes files belonging to a generation that failed
// before schema.json ever referenced it. It is deliberately separate from
// RemoveFromDisk: published generations may only be retired by the explicit
// lifecycle paths documented in the storage contract.
func discardUnpublishedShard(s *storageShard) {
	if s == nil || s.t == nil || s.t.schema == nil || s.t.schema.persistence == nil {
		return
	}
	if s.logfile != nil {
		s.logfile.Close()
		s.logfile = nil
	}
	if s.uuid == uuid.Nil {
		return
	}
	for _, col := range s.t.Columns {
		s.t.schema.persistence.RemoveColumn(s.uuid.String(), col.Name)
	}
	s.t.schema.persistence.RemoveLog(s.uuid.String())
}

// removePersistence removes on-disk files (columns + logfile) for this shard
// without invalidating the shard itself. Unlike RemoveFromDisk (which uses
// sync.Once and is intended for shard disposal), this method allows the shard
// to continue living in RAM after its persistence has been stripped.
// Caller must hold s.mu.Lock().
//
// ⚠️  DATA SAFETY: This operation is IRREVERSIBLE and results in PERMANENT DATA
// LOSS for the on-disk representation of this shard. It must only be called
// from transitionShardEngine when the caller has explicitly requested
// ENGINE=memory via ALTER TABLE. Never call this from background cleanup,
// eviction, or any path triggered without explicit user intent.
func (s *storageShard) removePersistence() {
	if s.logfile != nil {
		s.logfile.Close()
		s.logfile = nil
	}
	for _, col := range s.t.Columns {
		s.t.schema.persistence.RemoveColumn(s.uuid.String(), col.Name)
	}
	s.t.schema.persistence.RemoveLog(s.uuid.String())
}

// transitionShardEngine handles the per-shard work when ALTER TABLE ENGINE=
// changes the persistency mode. Caller must hold s.mu.Lock() and the table's
// PersistencyMode must already be set to newMode.
//
// ⚠️  DATA SAFETY: The Persisted→Memory transition is IRREVERSIBLE. All
// column files and WAL for this shard are permanently deleted. This path is
// only reached from an explicit ALTER TABLE … ENGINE=memory statement.
// Do NOT add calls to this function from background or cleanup code paths.
//
// Transition safety matrix:
//   - Safe/Logged/Sloppy → Memory : permanent disk deletion (see removePersistence)
//   - Memory → Safe/Logged/Sloppy : safe; in-RAM data is serialised to disk
//   - Safe/Logged → Sloppy        : WAL removed; future writes lose crash safety
//   - Sloppy → Safe/Logged        : WAL opened; future writes become durable
//   - Safe ↔ Logged               : no-op at shard level
func transitionShardEngine(s *storageShard, oldMode, newMode PersistencyMode) {
	oldPersisted := oldMode != Memory && oldMode != Cache
	newPersisted := newMode != Memory && newMode != Cache

	switch {
	case oldPersisted && !newPersisted:
		// Persisted → Memory/Cache: IRREVERSIBLE — permanently delete on-disk files.
		// Only reached via explicit ALTER TABLE ENGINE=memory/cache.
		GlobalCache.Remove(s)
		s.removePersistence()
		if newMode == Cache && !strings.HasPrefix(s.t.Name, ".") {
			s.srState = SHARED
			GlobalCache.AddItem(s, int64(s.computeSizeLocked()), TypeCacheEntry, cacheShardCleanup, shardLastUsed, nil)
		} else {
			s.srState = WRITE
		}

	case !oldPersisted && newPersisted:
		// Memory/Cache → Persisted: materialize columns to disk, register with cache
		// (Cache→Persisted goes through the rebuild path in storage.go; this case
		// handles other non-persisted→persisted transitions that may arise.)
		if oldMode == Cache {
			GlobalCache.Remove(s)
		}
		s.ensureLoaded()
		// write each column to disk
		for colName, cs := range s.columns {
			if cs == nil {
				continue
			}
			f := s.t.schema.persistence.WriteColumn(s.uuid.String(), colName)
			cs.Serialize(f)
			finishColumnWrite(f, newMode == Safe)
		}
		// open logfile for Safe/Logged
		if newMode == Safe || newMode == Logged {
			s.logfile = s.t.schema.persistence.OpenLog(s.uuid.String())
		}
		s.srState = SHARED
		if !strings.HasPrefix(s.t.Name, ".") {
			GlobalCache.AddItem(s, int64(s.computeSizeLocked()), TypeShard, shardCleanup, shardLastUsed, nil)
		}

	case oldMode == Memory && newMode == Cache:
		// Memory → Cache: register with CacheManager as TypeCacheEntry
		s.srState = SHARED
		if !strings.HasPrefix(s.t.Name, ".") {
			GlobalCache.AddItem(s, int64(s.computeSizeLocked()), TypeCacheEntry, cacheShardCleanup, shardLastUsed, nil)
		}

	case oldMode == Cache && newMode == Memory:
		// Cache → Memory: deregister from CacheManager
		GlobalCache.Remove(s)
		s.srState = WRITE

	case oldMode == Sloppy && (newMode == Safe || newMode == Logged):
		// Sloppy → Safe/Logged: open logfile
		s.logfile = s.t.schema.persistence.OpenLog(s.uuid.String())

	case (oldMode == Safe || oldMode == Logged) && newMode == Sloppy:
		// Safe/Logged → Sloppy: close and remove logfile
		if s.logfile != nil {
			s.logfile.Close()
			s.logfile = nil
		}
		s.t.schema.persistence.RemoveLog(s.uuid.String())

	default:
		// Safe ↔ Logged: no-op on shard level (both use logfiles identically)
	}
}

// replayRebuiltRows replays one column's values in rebuiltRowIDs order for
// shard rebuild's scan/build passes. mainRecids/mainMask (precomputed once
// per shard, shared across every column) identify which slots of
// rebuiltRowIDs are sourced from main storage; those are bulk-fetched from
// reader in a single GetValueMulti call instead of one GetValue per row.
// Delta-sourced slots still go through getDelta one row at a time, since
// that reads a plain Go map rather than a ColumnStorage.
func replayRebuiltRows(reader ColumnReader, mainRecids []uint32, mainMask []bool, rebuiltRowIDs []uint32, mainCount uint32, col string, getDelta func(int, string) scm.Scmer, emit func(uint32, scm.Scmer)) {
	buf := make([]scm.Scmer, len(mainRecids))
	if len(mainRecids) > 0 {
		reader.GetValueMulti(mainRecids, buf, 1)
	}
	bufIdx := 0
	for k, id := range rebuiltRowIDs {
		if mainMask[k] {
			emit(uint32(k), buf[bufIdx])
			bufIdx++
		} else {
			emit(uint32(k), getDelta(int(id-mainCount), col))
		}
	}
}

// rebuild main storage from main+delta
func (t *storageShard) rebuild(all bool) *storageShard {
	// An unchanged cold shard already is a complete persistent generation:
	// its UUID references both the column files and any unreplayed WAL. Keep it
	// byte-for-byte instead of loading it merely for a periodic rebuild or
	// shutdown. A forced rebuild must materialize it before continuing.
	var columnSnapshot map[string]ColumnStorage
	var indexSnapshot []*StorageIndex
	var deltaColumnsSnapshot map[string]int
	var insertsSnapshot [][]scm.Scmer
	var mainCount uint32
	var maxInsertIndex int
	var deletions NonLockingReadMap.NonBlockingBitMap
	var rollbackProtected NonLockingReadMap.NonBlockingBitMap
	for {
		t.mu.Lock()
		if t.srState != COLD {
			if next := t.loadNext(); next != nil {
				t.mu.Unlock()
				// lock+unlock the next shard so we don't return too early (sync hazards)
				next.mu.Lock()
				next.mu.Unlock()
				return next // already rebuilding (happens on parallel inserts)
			}

			maxInsertIndex = len(t.inserts)
			deletions = t.deletions.Copy()
			rollbackProtected = t.rollbackProtected.Copy()
			if all || maxInsertIndex > 0 || deletions.Count() > 0 {
				// Materialize cold columns before publishing t.next. The initial
				// snapshot and rebuild publication must become visible atomically;
				// only the bounded final catch-up reacquires the source afterward.
				var nilCols []string
				for col, storage := range t.columns {
					if storage == nil {
						nilCols = append(nilCols, col)
					}
				}
				if len(nilCols) > 0 {
					t.mu.Unlock()
					for _, col := range nilCols {
						t.ensureColumnLoaded(col, false)
					}
					continue
				}
				columnSnapshot = make(map[string]ColumnStorage, len(t.columns))
				for col, storage := range t.columns {
					columnSnapshot[col] = storage
				}
				indexSnapshot = snapshotIndexesForRebuild(t.Indexes)
				deltaColumnsSnapshot = make(map[string]int, len(t.deltaColumns))
				for col, index := range t.deltaColumns {
					deltaColumnsSnapshot[col] = index
				}
				insertsSnapshot = append([][]scm.Scmer(nil), t.inserts[:maxInsertIndex]...)
				mainCount = t.main_count
			}
			break
		}
		if !all {
			t.mu.Unlock()
			return t
		}
		t.mu.Unlock()
		t.ensureLoaded()
	}

	// concurrency! when rebuild is run in background, inserts and deletions into and from old delta storage must be duplicated to the ongoing process
	locked := true
	removedFromCache := false
	defer func() {
		if locked {
			t.mu.Unlock()
		}
	}()
	result := new(storageShard)
	result.t = t.t
	result.srState = WRITE // mark as live so ensureLoaded() won't reset columns
	result.mu.Lock()       // interlock so no one will rebuild the shard twice
	result.enterWriteOwner()
	// Publish only after result.mu is held. A mutator that observes next can
	// therefore buffer on the source shard without touching partial state.
	t.nextPendingDeletes = nil
	t.nextReady.Store(false)
	t.storeNext(result)
	resultLocked := true
	defer func() {
		if resultLocked {
			result.exitWriteOwner()
			result.mu.Unlock()
		}
	}()
	defer func() {
		if r := recover(); r != nil {
			// If rebuild panics, ensure we don't leave a half-built shard reachable via t.next.
			// Otherwise, later rebuild/save cycles may publish a schema referencing a UUID whose
			// column files were never written.
			t.clearNext(result)
			t.clearNextTranslation()
			if result.logfile != nil {
				func() { defer func() { _ = recover() }(); result.logfile.Close() }()
			}
			if result.uuid != uuid.Nil && result.t != nil && result.t.schema != nil && result.t.schema.persistence != nil {
				func() { defer func() { _ = recover() }(); result.t.schema.persistence.RemoveLog(result.uuid.String()) }()
				for _, col := range result.t.Columns {
					func() {
						defer func() { _ = recover() }()
						result.t.schema.persistence.RemoveColumn(result.uuid.String(), col.Name)
					}()
				}
			}
			// Re-register old shard with CacheManager if we deregistered it
			if removedFromCache && t.t != nil {
				if t.t.PersistencyMode == Cache && !strings.HasPrefix(t.t.Name, ".") {
					GlobalCache.AddItem(t, int64(t.ComputeSize()), TypeCacheEntry, cacheShardCleanup, shardLastUsed, nil)
				} else if t.t.PersistencyMode != Memory && !strings.HasPrefix(t.t.Name, ".") {
					GlobalCache.AddItem(t, int64(t.ComputeSize()), TypeShard, shardCleanup, shardLastUsed, nil)
				}
			}
			panic(r)
		}
	}()

	// The expensive build can now run without blocking source mutations. Their
	// delta suffix and delete recids are replayed by the bounded final catch-up.

	if all || maxInsertIndex > 0 || deletions.Count() > 0 {
		result.uuid, _ = uuid.NewRandom() // new uuid, serialize

		var b strings.Builder
		b.WriteString("rebuilding shard for table ")
		b.WriteString(t.t.Name)
		b.WriteString("(")

		// prepare delta storage
		result.columns = make(map[string]ColumnStorage)
		result.deltaColumns = make(map[string]int)
		result.deletions.Reset()
		if result.t.PersistencyMode == Safe || result.t.PersistencyMode == Logged {
			// safe mode: also write all deltas to disk
			result.logfile = result.t.schema.persistence.OpenLog(result.uuid.String())
		}
		t.mu.Unlock() // release lock, from now on, deletions+inserts should work
		locked = false

		// Deregister old shard from CacheManager early to prevent eviction
		// from destroying column data that rebuild reads from.
		GlobalCache.Remove(t)
		removedFromCache = true

		// transfer indexes early so we know which index is Native (physically sorted)
		rebuildIndexes(indexSnapshot, result)

		getDelta := func(idx int, col string) scm.Scmer {
			item := insertsSnapshot[idx]
			if colidx, ok := deltaColumnsSnapshot[col]; ok && colidx < len(item) {
				return item[colidx]
			}
			if proxy, ok := columnSnapshot[col].(*StorageComputeProxy); ok {
				return proxy.GetValue(mainCount + uint32(idx))
			}
			return scm.NewNil()
		}
		keepRecid := func(recid uint32) bool {
			return !deletions.Get(uint(recid)) || rollbackProtected.Get(uint(recid))
		}

		// compute sort permutation for the Native index (if any)
		var sortPerm []uint32
		for _, idx := range result.Indexes {
			if !idx.Native {
				continue
			}
			// check that all index columns exist in old shard
			allFound := true
			for _, col := range idx.Cols {
				if _, ok := columnSnapshot[col]; !ok {
					allFound = false
					break
				}
			}
			if !allFound {
				idx.Native = false
				break
			}
			sortPerm = make([]uint32, 0, int(mainCount)+maxInsertIndex)
			for i := uint32(0); i < mainCount; i++ {
				if keepRecid(i) {
					sortPerm = append(sortPerm, i)
				}
			}
			for i := 0; i < maxInsertIndex; i++ {
				if keepRecid(mainCount + uint32(i)) {
					sortPerm = append(sortPerm, mainCount+uint32(i))
				}
			}
			hybridsort.HybridSort(sortPerm, func(idA, idB uint32) bool {
				for colIdx, colName := range idx.Cols {
					var va, vb scm.Scmer
					if idA < mainCount {
						va = columnSnapshot[colName].GetValue(idA)
					} else {
						va = getDelta(int(idA-mainCount), colName)
					}
					if idB < mainCount {
						vb = columnSnapshot[colName].GetValue(idB)
					} else {
						vb = getDelta(int(idB-mainCount), colName)
					}
					if idx.lessAt(colIdx, va, vb) {
						return true
					}
					if idx.lessAt(colIdx, vb, va) {
						return false
					}
				}
				return false
			})
			break
		}

		rowCap := int(mainCount) + maxInsertIndex
		rebuiltRowIDs := make([]uint32, 0, rowCap)
		if sortPerm != nil {
			rebuiltRowIDs = append(rebuiltRowIDs, sortPerm...)
		} else {
			for idx := uint32(0); idx < mainCount; idx++ {
				if keepRecid(idx) {
					rebuiltRowIDs = append(rebuiltRowIDs, idx)
				}
			}
			for idx := 0; idx < maxInsertIndex; idx++ {
				globalID := mainCount + uint32(idx)
				if keepRecid(globalID) {
					rebuiltRowIDs = append(rebuiltRowIDs, globalID)
				}
			}
		}
		// mainRecids/mainMask precompute, once per rebuilt shard (not per column),
		// which slots of rebuiltRowIDs are sourced from main storage vs delta so
		// every column's scan/build replay can bulk-fetch its main-sourced values
		// in one GetValueMulti call instead of one GetValue per row.
		mainRecids := make([]uint32, 0, len(rebuiltRowIDs))
		mainMask := make([]bool, len(rebuiltRowIDs))
		for k, id := range rebuiltRowIDs {
			if id < mainCount {
				mainMask[k] = true
				mainRecids = append(mainRecids, id)
			}
		}

		nextTranslation := make(map[uint32]uint32, len(rebuiltRowIDs))
		for newRecid, oldRecid := range rebuiltRowIDs {
			nextTranslation[oldRecid] = uint32(newRecid)
		}
		t.setNextTranslation(nextTranslation)
		for oldRecid, newRecid := range nextTranslation {
			if rollbackProtected.Get(uint(oldRecid)) {
				result.rollbackProtected.Set(uint(newRecid), true)
			}
			if !deletions.Get(uint(oldRecid)) {
				continue
			}
			result.deletions.Set(uint(newRecid), true)
			if result.logfile != nil {
				result.logfile.Write(LogEntryDelete{newRecid})
			}
		}

		// copy column data in two phases: scan, build (if delta is non-empty)
		isFirst := true
		for col, c := range columnSnapshot {
			if isFirst {
				isFirst = false
			} else {
				b.WriteString(", ")
			}

			if oldProxy, ok := c.(*StorageComputeProxy); ok {
				newProxy := cloneComputeProxyRows(oldProxy, result, rebuiltRowIDs)
				result.columns[col] = newProxy
				result.main_count = uint32(len(rebuiltRowIDs))
				b.WriteString(col)
				b.WriteString(" ")
				b.WriteString(newProxy.String())
				if t.t.PersistencyMode != Memory && t.t.PersistencyMode != Cache {
					f := result.t.schema.persistence.WriteColumn(result.uuid.String(), col)
					newProxy.Serialize(f)
					finishColumnWrite(f, t.t.PersistencyMode == Safe)
				}
				continue
			}

			var newcol ColumnStorage = new(StorageSCMER) // currently only scmer-storages
			var i uint32
			var reader ColumnReader
			for {
				// scan phase
				reader = c.GetCachedReader() // must NOT use newCachedColumnReader: it strips OverlayBlob
				newcol.prepare()
				replayRebuiltRows(reader, mainRecids, mainMask, rebuiltRowIDs, mainCount, col, getDelta, newcol.scan)
				i = uint32(len(rebuiltRowIDs))
				newcol2 := newcol.proposeCompression(i)
				if newcol2 == nil {
					break // we found the optimal storage format
				} else {
					// redo scan phase with compression
					//fmt.Printf("Compression with %T\n", newcol2)
					newcol = newcol2
				}
			}
			// build phase
			// TODO: when source and target are both OverlayBlob, pass raw
			// compressed blob data through instead of decompressing via
			// GetValue and recompressing in build(). This avoids a full
			// gzip round-trip per blob during rebuild.
			if blob, ok := newcol.(*OverlayBlob); ok {
				blob.schema = result.t.schema
			}
			newcol.init(i)
			reader = c.GetCachedReader() // must NOT use newCachedColumnReader: it strips OverlayBlob
			replayRebuiltRows(reader, mainRecids, mainMask, rebuiltRowIDs, mainCount, col, getDelta, newcol.build)
			newcol.finish()

			// LZ4 string dict compression: if the old column was a StorageString
			// with zero reads in the last rebuild cycle, compress the new column's
			// dictionary so it doesn't occupy RAM until actually needed.
			if oldStr, ok := c.(*StorageString); ok {
				if newStr, ok2 := newcol.(*StorageString); ok2 {
					if oldStr.ReadCount() == 0 {
						newStr.CompressDictionary()
					}
				}
			}

			result.columns[col] = newcol
			result.main_count = i

			// write statistics
			b.WriteString(col) // colname
			b.WriteString(" ")
			b.WriteString(newcol.String()) // storage type (remove *storage.Storage, so it will only say SCMER, Sparse, Int or String)

			// write to disc (only if required)
			if t.t.PersistencyMode != Memory && t.t.PersistencyMode != Cache {
				f := result.t.schema.persistence.WriteColumn(result.uuid.String(), col)
				newcol.Serialize(f) // col takes ownership of f, so they will defer f.Close() at the right time
				finishColumnWrite(f, t.t.PersistencyMode == Safe)
			}
		}
		b.WriteString(") -> ")
		b.WriteString(fmt.Sprint(result.main_count))
		fmt.Println(b.String())

		// Eagerly rebuild indexes with sufficient Savings so the first
		// query after rebuild does not pay a cold-start full-scan penalty.
		for _, idx := range result.Indexes {
			if idx.Savings >= 2.0 && !idx.baseState.active {
				// Verify all required columns exist before building the index.
				// A column may be absent from this shard if it was added after
				// the shard was created (e.g. ALTER TABLE ADD COLUMN).
				allFound := true
				for i, colName := range idx.Cols {
					if len(idx.ColMapFn) > i && !idx.ColMapFn[i].IsNil() {
						// computed column: check that all source columns exist
						for _, mc := range idx.ColMapCols[i] {
							if cs, ok := result.columns[mc]; !ok || cs == nil {
								allFound = false
								break
							}
						}
					} else {
						// raw column: check the column itself exists
						if cs, ok := result.columns[colName]; !ok || cs == nil {
							allFound = false
						}
					}
					if !allFound {
						break
					}
				}
				if allFound {
					getters, sessionKeys := idx.buildGetters(nil)
					idx.syncSessionKeys(sessionKeys)
					idx.buildIndex(idx.stateForTx(nil, true), getters, nil)
					GlobalCache.AddItem(idx, int64(idx.ComputeSize()), TypeIndex, indexCleanup, indexLastUsed, indexGetScore)
				}
			}
		}

		// Do not persist schema from inside shard rebuild; callers
		// publish the new shard pointer and then save atomically at the
		// table/database level to avoid transient, inconsistent schemas.

		// Keep the old WAL until the caller has durably published the new shard
		// list and saved schema.json. Otherwise a crash between rebuild and
		// publish can leave schema.json pointing at the old shard UUID after its
		// WAL was already removed, losing rows that only existed in the old log.

		// The caller retains the old shard until the new schema generation is
		// durably committed, then removes it explicitly. A finalizer must never
		// delete files because the last committed schema may still reference them.

		// Writers never wait for result.mu while the expensive generation is
		// built. Briefly stop source mutations, replay only the delta suffix and
		// buffered deletes, then switch subsequent mutations to direct forwarding.
		t.mu.Lock()
		t.catchUpRebuildLocked(result, maxInsertIndex)
		t.nextReady.Store(true)
		t.mu.Unlock()
	} else {
		// otherwise: table stays the same
		result.uuid = t.uuid // copy uuid in case nothing changes
		result.columns = t.columns
		result.deltaColumns = t.deltaColumns
		result.main_count = t.main_count
		result.inserts = t.inserts
		result.deletions = deletions
		result.Indexes = t.Indexes
		if t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged {
			if t.logfile != nil {
				t.logfile.Close()
			}
			t.t.schema.persistence.RemoveLog(t.uuid.String())
			result.logfile = result.t.schema.persistence.OpenLog(result.uuid.String())
		}
		t.logfile = nil
		nextTranslation := make(map[uint32]uint32, int(result.main_count))
		for recid := uint32(0); recid < result.main_count; recid++ {
			nextTranslation[recid] = recid
		}
		t.setNextTranslation(nextTranslation)
		// Update index parent pointers to reference the new shard
		for _, idx := range result.Indexes {
			idx.t = result
		}
		t.nextReady.Store(true)
		t.mu.Unlock()
		locked = false
		// Deregister old shard — result replaces it
		GlobalCache.Remove(t)
		removedFromCache = true
	}
	// Unlock result before registration (ComputeSize needs RLock)
	result.exitWriteOwner()
	result.mu.Unlock()
	resultLocked = false
	// Register the new shard with CacheManager
	atomic.StoreUint64(&result.lastAccessed, uint64(time.Now().UnixNano()))
	if result.t.PersistencyMode == Cache && !strings.HasPrefix(result.t.Name, ".") {
		GlobalCache.AddItem(result, int64(result.ComputeSize()), TypeCacheEntry, cacheShardCleanup, shardLastUsed, nil)
	} else if result.t.PersistencyMode != Memory && !strings.HasPrefix(result.t.Name, ".") {
		GlobalCache.AddItem(result, int64(result.ComputeSize()), TypeShard, shardCleanup, shardLastUsed, nil)
	}
	return result
}
