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
import "sort"
import "sync"
import "sync/atomic"
import "time"
import "strings"
import "reflect"
import "runtime"
import "encoding/json"
import "encoding/binary"
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
	columns    map[string]ColumnStorage
	// delta storage
	deltaColumns map[string]int
	inserts      [][]scm.Scmer                       // items added to storage
	deletions    NonLockingReadMap.NonBlockingBitMap // items removed from main or inserts (based on main_count + i)
	writeOwners  map[uint64]uint32                   // goroutine-local write ownership marker
	writeOwnMu   sync.Mutex                          // guards writeOwners
	logfile      PersistenceLogfile                  // only in safe mode
	mu           sync.RWMutex                        // delta write lock (working on main storage is lock free)
	uniquelock   sync.Mutex                          // unique insert lock (only used in the sharded case)
	next         atomic.Pointer[storageShard]        // rebuild successor published lock-free to concurrent writers
	// indexes
	Indexes    []*StorageIndex // sorted keys
	indexMutex sync.Mutex
	hashmaps1  map[[1]string]map[[1]scm.Scmer]uint32 // hashmaps for single columned unique keys
	hashmaps2  map[[2]string]map[[2]scm.Scmer]uint32 // hashmaps for single columned unique keys
	hashmaps3  map[[3]string]map[[3]scm.Scmer]uint32 // hashmaps for single columned unique keys

	// lazy-loading/shared-resource state
	srState      SharedState
	lastAccessed uint64 // UnixNano, atomic; updated on GetRead/GetExclusive for LRU eviction

	// repartition drain tracking: counts in-flight scans on this shard
	activeScanners atomic.Int32

	// guards RemoveFromDisk against double execution (finalizer + explicit cleanup)
	cleanupOnce sync.Once
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
		// TODO: hashmaps for unique
		return result
	}
	result += s.deletions.ComputeSize()
	result += scm.ComputeSize(scm.NewAny(s.inserts))
	for _, idx := range s.Indexes {
		result += idx.ComputeSize()
	}
	// TODO: hashmaps for unique
	return result
}

func (u *storageShard) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.uuid.String())
}
func (u *storageShard) UnmarshalJSON(data []byte) error {
	u.uuid.UnmarshalText(data)
	// do not load heavy fields here; delay until first access
	u.columns = make(map[string]ColumnStorage)
	u.deltaColumns = make(map[string]int)
	u.hashmaps1 = make(map[[1]string]map[[1]scm.Scmer]uint32)
	u.hashmaps2 = make(map[[2]string]map[[2]scm.Scmer]uint32)
	u.hashmaps3 = make(map[[3]string]map[[3]scm.Scmer]uint32)
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
			case LogEntryInsert:
				u.insertDatasetFromLog(l.cols, l.values)
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
			return cs
		}
		if u.t.PersistencyMode == Memory || u.t.PersistencyMode == Cache {
			u.columns[colName] = new(StorageSparse)
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
		if blob, ok := columnstorage.(*OverlayBlob); ok {
			blob.SetSchema(u.t.schema)
		}
		if proxy, ok := columnstorage.(*StorageComputeProxy); ok {
			proxy.shard = u
		}
		if uint32(cnt) > u.main_count {
			u.main_count = uint32(cnt)
		}
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
		return u.getColumnStorageOrPanicEx(colName, true)
	}
	if tx := CurrentTx(); tx != nil && tx.HasShardWrite(u) {
		return u.getColumnStorageOrPanicEx(colName, true)
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

func (u *storageShard) getColumnStorageOrPanicEx(colName string, alreadyLocked bool) ColumnStorage {
	if alreadyLocked {
		cs, present := u.columns[colName]
		if !present {
			// Shards can lag behind table schema changes (for example when a
			// column was added after the shard was created and the old shard was
			// later reloaded from disk). Mirror the fallback from the unlocked
			// path so scans can still proceed under the shard write lock.
			for _, c := range u.t.Columns {
				if c.Name == colName {
					u.columns[colName] = new(StorageSparse)
					return u.columns[colName]
				}
			}
			panic("Column does not exist: `" + u.t.schema.Name + "`.`" + u.t.Name + "`.`" + colName + "`")
		}
		if cs != nil {
			return cs
		}
		return u.ensureColumnLoaded(colName, true)
	}
	// alreadyLocked=false: caller guarantees ownsWrite=false — skip hasWriteOwner().
	if tx := CurrentTx(); tx != nil && tx.HasShardWrite(u) {
		return u.getColumnStorageOrPanicEx(colName, true)
	}
	u.mu.RLock()
	cs, present := u.columns[colName]
	u.mu.RUnlock()
	if !present {
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
		idx.active = false
		idx.mainIndexes = StorageInt{}
		idx.deltaBtree = nil
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
		idx.active = false
		idx.mainIndexes = StorageInt{}
		idx.deltaBtree = nil
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
	result.hashmaps1 = make(map[[1]string]map[[1]scm.Scmer]uint32)
	result.hashmaps2 = make(map[[2]string]map[[2]scm.Scmer]uint32)
	result.hashmaps3 = make(map[[3]string]map[[3]scm.Scmer]uint32)
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

func (t *storageShard) hasWriteOwner() bool {
	goid := currentGoroutineID()
	if goid == 0 {
		return false
	}
	t.writeOwnMu.Lock()
	defer t.writeOwnMu.Unlock()
	return t.writeOwners[goid] > 0
}

// rowValueByRecidLocked reads a column value for a recid. Caller must hold t.mu.
func (t *storageShard) rowValueByRecidLocked(recid uint32, col string) scm.Scmer {
	if recid < t.main_count {
		cs := t.getColumnStorageOrPanicEx(col, true)
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

func (t *storageShard) UpdateFunction(idx uint32, withTrigger bool, alreadyLocked bool) func(...scm.Scmer) scm.Scmer {
	return t.UpdateFunctionBatch(idx, withTrigger, alreadyLocked, nil)
}

func (t *storageShard) UpdateFunctionBatch(idx uint32, withTrigger bool, alreadyLocked bool, batch *triggerBatch) func(...scm.Scmer) scm.Scmer {
	// returns a callback with which you can delete or update an item
	return func(a ...scm.Scmer) scm.Scmer {
		//fmt.Println("update/delete", a)
		// FK checks are enforced via auto-generated system triggers (see createforeignkey)

		result := false // result = true when update was possible; false if there was a RESTRICT
		targetIdx := idx
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
				if (func() bool {
					tx := CurrentTx()
					return tx == nil || tx.Mode != TxACID
				})() && t.deletions.Get(uint(targetIdx)) {
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
					k := colDesc.Name
					cs := t.getColumnStorageOrPanicEx(k, true)
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
						colName := colDesc.Name
						pos, ok := t.deltaColumns[colName]
						if !ok || pos >= len(d2) {
							pCols = append(pCols, colName)
							pRow = append(pRow, scm.NewNil())
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
					if !scm.Equal(d2[colidx], newVal) {
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
						if i < len(oldDeltaRow) && !scm.Equal(oldDeltaRow[i], v) {
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
							if ok && colidx < len(oldDeltaRow) && colidx < len(d2) && !scm.Equal(oldDeltaRow[colidx], d2[colidx]) {
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

				currentTx := CurrentTx()
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
				t.insertDataset(payloadCols, [][]scm.Scmer{payloadRow}, nil)
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
							}
						}
					}
				}
				// Capture for dual-write forwarding (cols/d2 are closure-local)
				if t.t.repartitionActive {
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
					currentTx.AddToUndeleteMask(t, newRecid)
					// Only log the insert (delete applied at commit)
					if (t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged) && t.logfile != nil {
						t.logfile.Write(LogEntryInsert{payloadCols, [][]scm.Scmer{payloadRow}})
					}
				} else {
					// Cursor-stability / no-tx: existing behavior
					if (t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged) && t.logfile != nil {
						t.logfile.Write(LogEntryDelete{targetIdx})
						t.logfile.Write(LogEntryInsert{payloadCols, [][]scm.Scmer{payloadRow}})
					}
				}
			}()
			// Dual-write: forward the new row to the secondary shard set
			if result && dualWriteRow != nil {
				t.t.dualWriteInsert(dualWriteCols, dualWriteRow)
			}
			// transaction bookkeeping + deferred sync
			// (shard is already registered via OpenMapReducer — no per-row RegisterTouchedShard)
			if result {
				if tx := CurrentTx(); tx != nil {
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
						cs := t.getColumnStorageOrPanicEx(col.Name, true)
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
						cs := t.getColumnStorageOrPanicEx(col.Name, true)
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

			if tx := CurrentTx(); tx != nil && tx.Mode == TxACID {
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
				result = true
			} else {
				func() {
					if !alreadyLocked {
						t.mu.Lock()         // write lock
						defer t.mu.Unlock() // write lock
					}

					t.deletions.Set(uint(idx), true) // mark as deleted
					if (t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged) && t.logfile != nil {
						t.logfile.Write(LogEntryDelete{idx})
					}
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
			next := t.loadNext()
			if next != nil {
				// also change in next storage
				// idx translation (subtract the amount of deletions from that idx)
				idx2 := targetIdx - uint32(t.deletions.CountUntil(uint(targetIdx)))
				next.UpdateFunction(idx2, false, false)(a...) // propagate to succeeding shard
			}
		}
		return scm.NewBool(result) // maybe instead return UpdateFunction for newly inserted item??
	}
}

func (t *storageShard) ColumnReader(col string) func(uint32) scm.Scmer {
	cstorage := t.getColumnStorageOrPanic(col)
	return func(idx uint32) scm.Scmer {
		if idx < t.main_count {
			return cstorage.GetValue(idx)
		} else {
			return t.getDelta(int(idx-t.main_count), col)
		}
	}
}

// breakSentinel is the panic value injected by $break pseudo-column closures.
// scan_order.go catches this type to implement early-exit (LIMIT semantics inside ORC).
type breakSentinel struct{}

// ShardMapReducer pre-allocates args and applies map+reduce over batches of record IDs.
// Local implementation of the streaming MapReducer pattern (see todos/cluster.md §15.7).
// Stream() partitions recid batches into main/delta runs and dispatches to
// processMainBlock/processDeltaBlock – tight loops suitable for JIT compilation.
// For remote shards, Stream() will be backed by an RPC returning the accumulator per batch.
type ShardMapReducer struct {
	shard           *storageShard
	mainCols        []ColumnStorage        // direct main storage access (nil for $update/$invalidate/$increment cols)
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
	setClosureFn    []*func(uint32, ...scm.Scmer) scm.Scmer // per $set col
	incrClosureFn   []*func(uint32, ...scm.Scmer) scm.Scmer // per $increment col
	invClosureFn    []*func(uint32, ...scm.Scmer) scm.Scmer // per $invalidate col
	noopClosureFn   *func(uint32, ...scm.Scmer) scm.Scmer   // shared noop
	breakClosureFn  *func(uint32, ...scm.Scmer) scm.Scmer   // shared break
	args            []scm.Scmer                             // pre-allocated args buffer
	mapFn           func(...scm.Scmer) scm.Scmer
	reduceFn        func(...scm.Scmer) scm.Scmer
	mapScmer        scm.Scmer // original Scmer for network serialization
	deleteBatch     *triggerBatch // when set, DELETE triggers are batched instead of per-row
	reduceScmer     scm.Scmer // original Scmer for network serialization
	mainCount       uint32
	hasUpdateCol    bool
	hasIncrementCol bool
	// shardWriteLocked is true when the caller already holds shard.mu (write lock)
	// and registered write ownership before opening this mapper. When true,
	// processMainBlock/processDeltaBlock must NOT try to re-acquire the lock.
	shardWriteLocked bool
}

// OpenMapReducer creates a MapReducer for the given columns. Column readers and
// main storage references are built once; the args buffer is pre-allocated.
// mapFn and reduceFn are stored as Scmer for future network serialization;
// OptimizeProcToSerialFunction is called here (TODO: replace with JIT compilation).
func (t *storageShard) OpenMapReducer(cols []string, mapFn scm.Scmer, reduceFn scm.Scmer) *ShardMapReducer {
	return t.openMapReducerEx(cols, mapFn, reduceFn, false)
}

// openMapReducerEx is the implementation of OpenMapReducer.
// alreadyLocked=true skips the hasWriteOwner() check per column (caller already holds
// the write lock or has confirmed ownsWrite=false via skipShardReadLock).
func (t *storageShard) openMapReducerEx(cols []string, mapFn scm.Scmer, reduceFn scm.Scmer, alreadyLocked bool) *ShardMapReducer {
	mr := &ShardMapReducer{
		shard:            t,
		mainCols:         make([]ColumnStorage, len(cols)),
		colNames:         cols,
		isUpdate:         make([]bool, len(cols)),
		isInvalidate:     make([]bool, len(cols)),
		invalidateProxy:  make([]*StorageComputeProxy, len(cols)),
		isIncrement:      make([]bool, len(cols)),
		incrementProxy:   make([]*StorageComputeProxy, len(cols)),
		isSet:            make([]bool, len(cols)),
		setProxy:         make([]*StorageComputeProxy, len(cols)),
		isBreak:          make([]bool, len(cols)),
		setClosureFn:     make([]*func(uint32, ...scm.Scmer) scm.Scmer, len(cols)),
		incrClosureFn:    make([]*func(uint32, ...scm.Scmer) scm.Scmer, len(cols)),
		invClosureFn:     make([]*func(uint32, ...scm.Scmer) scm.Scmer, len(cols)),
		args:             make([]scm.Scmer, len(cols)),
		mapFn:            scm.OptimizeProcToSerialFunction(mapFn),
		reduceFn:         scm.OptimizeProcToSerialFunction(reduceFn),
		mapScmer:         mapFn,
		reduceScmer:      reduceFn,
		mainCount:        t.main_count,
		shardWriteLocked: alreadyLocked,
	}
	hasDeleteTriggers := false
	for _, tr := range t.t.Triggers {
		if tr.Timing == AfterDelete {
			hasDeleteTriggers = true
			break
		}
	}
	for i, col := range cols {
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
			cs := t.getColumnStorageOrPanicEx(cacheColName, alreadyLocked)
			if proxy, ok := cs.(*StorageComputeProxy); ok {
				mr.invalidateProxy[i] = proxy
			}
		} else if len(col) > 11 && col[:11] == "$increment:" {
			mr.isIncrement[i] = true
			mr.hasIncrementCol = true
			cacheColName := col[11:]
			cs := t.getColumnStorageOrPanicEx(cacheColName, alreadyLocked)
			if proxy, ok := cs.(*StorageComputeProxy); ok {
				mr.incrementProxy[i] = proxy
			}
		} else if len(col) > 5 && col[:5] == "$set:" {
			mr.isSet[i] = true
			mr.hasSetCol = true
			cacheColName := col[5:]
			cs := t.getColumnStorageOrPanicEx(cacheColName, alreadyLocked)
			if proxy, ok := cs.(*StorageComputeProxy); ok {
				mr.setProxy[i] = proxy
			}
		} else if col == "$break" {
			mr.isBreak[i] = true
			mr.hasBreakCol = true
		} else {
			mr.mainCols[i] = t.getColumnStorageOrPanicEx(col, alreadyLocked)
		}
	}
	// Pre-allocate tagClosure fn ptrs (hoisted, one per pseudo-col type per column).
	// These are allocated once here so processMainBlock/processDeltaBlock can use
	// NewClosure(ptr, effectiveID) per row without any heap allocation.
	noopFn := func(id uint32, args ...scm.Scmer) scm.Scmer { return scm.NewBool(true) }
	mr.noopClosureFn = &noopFn
	breakFn := func(id uint32, args ...scm.Scmer) scm.Scmer { panic(breakSentinel{}) }
	mr.breakClosureFn = &breakFn
	for i := range cols {
		if mr.isSet[i] {
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
		if mr.isIncrement[i] {
			if proxy := mr.incrementProxy[i]; proxy != nil {
				fn := func(id uint32, args ...scm.Scmer) scm.Scmer {
					if len(args) > 0 {
						proxy.IncrementalUpdate(id, args[0])
					}
					return scm.NewBool(true)
				}
				mr.incrClosureFn[i] = &fn
			}
		}
		if mr.isInvalidate[i] {
			if proxy := mr.invalidateProxy[i]; proxy != nil {
				fn := func(id uint32, args ...scm.Scmer) scm.Scmer {
					proxy.Invalidate(id)
					return scm.NewBool(true)
				}
				mr.invClosureFn[i] = &fn
			}
		}
	}
	// Register shard for deferred fsync once, not per row.
	if mr.hasUpdateCol {
		if tx := CurrentTx(); tx != nil {
			tx.RegisterTouchedShard(t)
		}
	}
	return mr
}

// Stream applies map+reduce over a batch of record IDs. The recid list is
// partitioned order-preserving into runs of main-storage IDs and delta IDs.
// Callers can pass subslices (e.g. items[0:1]) for zero-allocation single-item batches.
func (m *ShardMapReducer) Stream(acc scm.Scmer, recids []uint32) scm.Scmer {
	i := 0
	n := len(recids)
	for i < n {
		j := i + 1
		if recids[i] < m.mainCount {
			for j < n && recids[j] < m.mainCount {
				j++
			}
			acc = m.processMainBlock(acc, recids[i:j])
		} else {
			for j < n && recids[j] >= m.mainCount {
				j++
			}
			acc = m.processDeltaBlock(acc, recids[i:j])
		}
		i = j
	}
	return acc
}

// processMainBlock is a tight loop over main-storage records – no branching
// on main vs delta, direct ColumnStorage.GetValue calls. JIT candidate.
func (m *ShardMapReducer) processMainBlock(acc scm.Scmer, recids []uint32) scm.Scmer {
	// Hoist write-lock acquisition out of the per-row loop.
	// shardWriteLocked is set by openMapReducerEx when the caller (scan.go) already
	// holds shard.mu; in that case we must NOT re-acquire it. For the rare case
	// where a mapper is opened without the shard being locked (e.g. NEW. columns
	// outside a mutation scan), acquire once for the whole batch instead of per-row.
	if (m.hasUpdateCol || m.hasIncrementCol || m.hasSetCol) && !m.shardWriteLocked {
		currentTx := CurrentTx()
		m.shard.mu.Lock()
		defer m.shard.mu.Unlock()
		m.shard.enterWriteOwner()
		defer m.shard.exitWriteOwner()
		if currentTx != nil {
			currentTx.EnterShardWrite(m.shard)
			defer currentTx.ExitShardWrite(m.shard)
		}
	}
	for _, id := range recids {
		func() {
			effectiveID := id
			if m.hasUpdateCol || m.hasIncrementCol || m.hasSetCol {
				if tx := CurrentTx(); tx == nil || tx.Mode != TxACID {
					if m.shard.deletions.Get(uint(effectiveID)) {
						followedID, ok := m.shard.resolveVisiblePrimaryRecidLocked(effectiveID)
						if !ok {
							return
						}
						effectiveID = followedID
					}
				}
			}
			for i, col := range m.mainCols {
				if m.isInvalidate[i] {
					if fnptr := m.invClosureFn[i]; fnptr != nil {
						m.args[i] = scm.NewClosure(fnptr, effectiveID)
					} else {
						m.args[i] = scm.NewClosure(m.noopClosureFn, effectiveID)
					}
				} else if m.isIncrement[i] {
					if fnptr := m.incrClosureFn[i]; fnptr != nil {
						m.args[i] = scm.NewClosure(fnptr, effectiveID)
					} else {
						m.args[i] = scm.NewClosure(m.noopClosureFn, effectiveID)
					}
				} else if m.isSet[i] {
					if fnptr := m.setClosureFn[i]; fnptr != nil {
						m.args[i] = scm.NewClosure(fnptr, effectiveID)
					} else {
						m.args[i] = scm.NewClosure(m.noopClosureFn, effectiveID)
					}
				} else if m.isBreak[i] {
					m.args[i] = scm.NewClosure(m.breakClosureFn, effectiveID)
				} else if m.isUpdate[i] {
					m.args[i] = scm.NewFunc(m.shard.UpdateFunctionBatch(effectiveID, true, m.hasUpdateCol, m.deleteBatch))
				} else {
					if effectiveID < m.mainCount {
						m.args[i] = col.GetValue(effectiveID)
					} else {
						m.args[i] = m.shard.getDelta(int(effectiveID-m.mainCount), m.colNames[i])
					}
				}
			}
			acc = m.reduceFn(acc, m.mapFn(m.args...))
		}()
	}
	return acc
}

// processDeltaBlock handles delta-storage records via getDelta. JIT candidate.
func (m *ShardMapReducer) processDeltaBlock(acc scm.Scmer, recids []uint32) scm.Scmer {
	// Same hoisting as processMainBlock (see comment there).
	if (m.hasUpdateCol || m.hasIncrementCol || m.hasSetCol) && !m.shardWriteLocked {
		currentTx := CurrentTx()
		m.shard.mu.Lock()
		defer m.shard.mu.Unlock()
		m.shard.enterWriteOwner()
		defer m.shard.exitWriteOwner()
		if currentTx != nil {
			currentTx.EnterShardWrite(m.shard)
			defer currentTx.ExitShardWrite(m.shard)
		}
	}
	for _, id := range recids {
		func() {
			effectiveID := id
			if m.hasUpdateCol || m.hasIncrementCol || m.hasSetCol {
				if tx := CurrentTx(); tx == nil || tx.Mode != TxACID {
					if m.shard.deletions.Get(uint(effectiveID)) {
						followedID, ok := m.shard.resolveVisiblePrimaryRecidLocked(effectiveID)
						if !ok {
							return
						}
						effectiveID = followedID
					}
				}
			}
			for i, col := range m.colNames {
				if m.isSet[i] {
					// $set: works on delta rows too — proxy stores values
					// in its delta map keyed by effectiveID (any row index).
					if fnptr := m.setClosureFn[i]; fnptr != nil {
						m.args[i] = scm.NewClosure(fnptr, effectiveID)
					} else {
						m.args[i] = scm.NewClosure(m.noopClosureFn, effectiveID)
					}
				} else if m.isInvalidate[i] || m.isIncrement[i] {
					// invalidate/increment on delta rows outside proxy range — no-op
					m.args[i] = scm.NewClosure(m.noopClosureFn, effectiveID)
				} else if m.isBreak[i] {
					m.args[i] = scm.NewClosure(m.breakClosureFn, effectiveID)
				} else if m.isUpdate[i] {
					m.args[i] = scm.NewFunc(m.shard.UpdateFunctionBatch(effectiveID, true, m.hasUpdateCol, m.deleteBatch))
				} else if len(col) >= 4 && col[:4] == "NEW." {
					m.args[i] = scm.NewNil()
				} else {
					if effectiveID < m.mainCount {
						m.args[i] = m.mainCols[i].GetValue(effectiveID)
					} else if _, isProxy := m.mainCols[i].(*StorageComputeProxy); isProxy {
						// Proxy columns (computed/ORC): read from proxy even for delta rows.
						// The proxy stores values in its delta map keyed by row index.
						m.args[i] = m.mainCols[i].GetValue(effectiveID)
					} else {
						m.args[i] = m.shard.getDelta(int(effectiveID-m.mainCount), col)
					}
				}
			}
			acc = m.reduceFn(acc, m.mapFn(m.args...))
		}()
	}
	return acc
}

// Close releases resources held by the MapReducer. Does NOT flush trigger
// batches — that must happen after all shard locks are released.
// Call FlushTriggerBatch() separately after the scan completes.
func (m *ShardMapReducer) Close() {
}

// FlushTriggerBatch flushes any pending trigger batches. Must be called
// AFTER the scan completes and all shard locks are released, to avoid
// deadlocks when trigger handlers scan other tables.
func (m *ShardMapReducer) FlushTriggerBatch() {
	if m.deleteBatch != nil {
		m.deleteBatch.Flush()
		m.deleteBatch = nil
	}
}

func (t *storageShard) Insert(columns []string, values [][]scm.Scmer, alreadyLocked bool, onFirstInsertId func(int64), isIgnore bool) {
	// Check table-level user lock (LOCK TABLES): writes block under any lock.
	// Always call waitTableLock — it handles other-session blocking and
	// owner-write-under-READ-lock error in one place.
	if t.t.tableLockOwner.Load() != nil {
		t.t.waitTableLock(scm.GetCurrentSessionState(), true)
	}
	beforeInsertTriggers := t.t.GetTriggers(BeforeInsert)
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
			return
		}
		if !alreadyLocked {
			t.mu.Lock()
		}
		firstInsertId := onFirstInsertId
		for i, row := range preparedRows {
			t.insertPreparedLocked(preparedColumns[i], [][]scm.Scmer{row}, firstInsertId)
			if firstInsertId != nil {
				firstInsertId = nil
			}
		}
		if !alreadyLocked {
			t.mu.Unlock()
		}
		return
	}

	// Re-apply sanitizers after trigger-free INSERT input preparation.
	values = t.t.sanitizeInsertRows(columns, values, isIgnore)
	if len(values) == 0 {
		return // all rows skipped by sanitizer in INSERT IGNORE mode
	}

	if !alreadyLocked {
		t.mu.Lock()
	}
	t.insertPreparedLocked(columns, values, onFirstInsertId)
	if !alreadyLocked {
		t.mu.Unlock()
	}
}

func (t *storageShard) insertPreparedLocked(columns []string, values [][]scm.Scmer, onFirstInsertId func(int64)) {
	// capture starting row index for undo logging
	firstNewRecid := t.main_count + uint32(len(t.inserts))
	firstNewInsertIdx := len(t.inserts) // for capturing actual rows after insertDataset fills auto-increment
	var triggerInsertRows []dataset
	t.insertDataset(columns, values, onFirstInsertId)
	if len(t.t.Triggers) > 0 {
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
	if (t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged) && t.logfile != nil {
		// Log the actual inserted rows (not the original columns/values) so that
		// auto-incremented IDs and column defaults are preserved across restarts.
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
		t.logfile.Write(LogEntryInsert{idx2col, logVals})
	}
	if next := t.loadNext(); next != nil {
		// also insert into next storage
		next.Insert(columns, values, false, nil, false)
	}
	// transaction bookkeeping
	if tx := CurrentTx(); tx != nil {
		switch tx.Mode {
		case TxACID:
			// ACID: hide rows globally, add to undelete mask so this tx can see them
			for i := range values {
				recid := firstNewRecid + uint32(i)
				t.deletions.Set(uint(recid), true)
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
	if len(t.t.Triggers) > 0 {
		func() {
			t.mu.Unlock()
			defer t.mu.Lock()
			for _, row := range triggerInsertRows {
				t.t.ExecuteTriggers(AfterInsert, nil, row)
			}
		}()
	}
}

// contract: must only be called inside full write mutex mu.Lock()
func (t *storageShard) insertDataset(columns []string, values [][]scm.Scmer, onFirstInsertId func(int64)) {
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

		// notify all hashmaps (what if col is not present in newrow??)
		for k, v := range t.hashmaps1 {
			v[[1]scm.Scmer{
				newrow[t.deltaColumns[k[0]]],
			}] = recid
		}
		for k, v := range t.hashmaps2 {
			v[[2]scm.Scmer{
				newrow[t.deltaColumns[k[0]]],
				newrow[t.deltaColumns[k[1]]],
			}] = recid
		}
		for k, v := range t.hashmaps3 {
			v[[3]scm.Scmer{
				newrow[t.deltaColumns[k[0]]],
				newrow[t.deltaColumns[k[1]]],
				newrow[t.deltaColumns[k[2]]],
			}] = recid
		}

		// also notify indices
		for _, index := range t.Indexes {
			// add to delta indexes
			index.mu.Lock()
			if index.deltaBtree != nil {
				index.deltaBtree.ReplaceOrInsert(indexPair{int(recid), newrow})
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

		// notify temporary unique hashmaps
		for k, v := range t.hashmaps1 {
			v[[1]scm.Scmer{newrow[t.deltaColumns[k[0]]]}] = recid
		}
		for k, v := range t.hashmaps2 {
			v[[2]scm.Scmer{newrow[t.deltaColumns[k[0]]], newrow[t.deltaColumns[k[1]]]}] = recid
		}
		for k, v := range t.hashmaps3 {
			v[[3]scm.Scmer{newrow[t.deltaColumns[k[0]]], newrow[t.deltaColumns[k[1]]], newrow[t.deltaColumns[k[2]]]}] = recid
		}

		// update delta indexes
		for _, index := range t.Indexes {
			index.mu.Lock()
			if index.deltaBtree != nil {
				index.deltaBtree.ReplaceOrInsert(indexPair{int(recid), newrow})
			}
			index.mu.Unlock()
		}
	}
}

func (t *storageShard) GetRecordidForUnique(columns []string, values []scm.Scmer) (result uint32, present bool) {
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

	currentTx := CurrentTx()
	acidMode := currentTx != nil && currentTx.Mode == TxACID

	mainCount := t.main_count

	// Use iterateIndex for O(log n) lookup (builds index lazily if needed)
	// Small buffer for existence check: stop early after first match
	var buf [8]uint32
	t.iterateIndex(bounds, lower, upperLast, len(t.inserts), buf[:], func(batch []uint32) bool {
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

func (t *storageShard) getDelta(idx int, col string) scm.Scmer {
	item := t.inserts[idx]
	colidx, ok := t.deltaColumns[col]
	if ok {
		if colidx < len(item) {
			return item[colidx]
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
			f.Close()
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

// rebuild main storage from main+delta
func (t *storageShard) rebuild(all bool) *storageShard {

	// concurrency! when rebuild is run in background, inserts and deletions into and from old delta storage must be duplicated to the ongoing process
	t.mu.Lock()
	locked := true
	removedFromCache := false
	defer func() {
		if locked {
			t.mu.Unlock()
		}
	}()
	if next := t.loadNext(); next != nil {
		t.mu.Unlock()
		locked = false
		// lock+unlock the next shard so we don't return too early (sync hazards)
		next.mu.Lock()
		next.mu.Unlock()
		return next // already rebuilding (happens on parallel inserts)
		// possible problem: this call may return the t.next shard faster than the competing rebuild() call that actually rebuilds; maybe use a additional lock on t.next??
	}
	result := new(storageShard)
	result.t = t.t
	result.srState = WRITE // mark as live so ensureLoaded() won't reset columns
	t.storeNext(result)
	result.mu.Lock() // interlock so no one will rebuild the shard twice
	resultLocked := true
	defer func() {
		if resultLocked {
			result.mu.Unlock()
		}
	}()
	defer func() {
		if r := recover(); r != nil {
			// If rebuild panics, ensure we don't leave a half-built shard reachable via t.next.
			// Otherwise, later rebuild/save cycles may publish a schema referencing a UUID whose
			// column files were never written.
			t.clearNext(result)
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

	// now read out deletion list
	maxInsertIndex := len(t.inserts)
	// copy-freeze deletions so we don't have to lock anything
	deletions := t.deletions.Copy()
	// from now on, we can rebuild with no hurry; inserts and update/deletes on the previous shard will propagate to us, too

	if all || maxInsertIndex > 0 || deletions.Count() > 0 {
		result.uuid, _ = uuid.NewRandom() // new uuid, serialize

		var b strings.Builder
		b.WriteString("rebuilding shard for table ")
		b.WriteString(t.t.Name)
		b.WriteString("(")

		// prepare delta storage
		result.columns = make(map[string]ColumnStorage)
		result.deltaColumns = make(map[string]int)
		result.hashmaps1 = make(map[[1]string]map[[1]scm.Scmer]uint32)
		result.hashmaps2 = make(map[[2]string]map[[2]scm.Scmer]uint32)
		result.hashmaps3 = make(map[[3]string]map[[3]scm.Scmer]uint32)
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

		// Ensure all columns are loaded from disk. shardCleanup sets column
		// values to nil (while keeping the key) to free memory under pressure.
		// rebuild must read all columns, so materialize any nil entries now
		// while t.mu is not held (ensureColumnLoaded acquires it internally).
		for col, c := range t.columns {
			if c == nil {
				t.ensureColumnLoaded(col, false)
			}
		}

		// transfer indexes early so we know which index is Native (physically sorted)
		rebuildIndexes(t, result)

		// compute sort permutation for the Native index (if any)
		var sortPerm []uint32
		for _, idx := range result.Indexes {
			if !idx.Native {
				continue
			}
			// check that all index columns exist in old shard
			allFound := true
			for _, col := range idx.Cols {
				if _, ok := t.columns[col]; !ok {
					allFound = false
					break
				}
			}
			if !allFound {
				idx.Native = false
				break
			}
			sortPerm = make([]uint32, 0, int(t.main_count)+maxInsertIndex)
			for i := uint32(0); i < t.main_count; i++ {
				if !deletions.Get(uint(i)) {
					sortPerm = append(sortPerm, i)
				}
			}
			for i := 0; i < maxInsertIndex; i++ {
				if !deletions.Get(uint(t.main_count) + uint(i)) {
					sortPerm = append(sortPerm, t.main_count+uint32(i))
				}
			}
			idxCols := idx.Cols
			sort.Slice(sortPerm, func(a, b int) bool {
				for _, colName := range idxCols {
					var va, vb scm.Scmer
					idA, idB := sortPerm[a], sortPerm[b]
					if idA < t.main_count {
						va = t.columns[colName].GetValue(idA)
					} else {
						va = t.getDelta(int(idA-t.main_count), colName)
					}
					if idB < t.main_count {
						vb = t.columns[colName].GetValue(idB)
					} else {
						vb = t.getDelta(int(idB-t.main_count), colName)
					}
					if scm.Less(va, vb) {
						return true
					}
					if scm.Less(vb, va) {
						return false
					}
				}
				return false
			})
			break
		}

		// Snapshot column keys under lock to avoid concurrent map iteration + write.
		// After unlock, new columns may be added but won't be seen by this rebuild.
		t.mu.RLock()
		columnSnapshot := make(map[string]ColumnStorage, len(t.columns))
		for k, v := range t.columns {
			columnSnapshot[k] = v
		}
		t.mu.RUnlock()

		// copy column data in two phases: scan, build (if delta is non-empty)
		isFirst := true
		for col, c := range columnSnapshot {
			if isFirst {
				isFirst = false
			} else {
				b.WriteString(", ")
			}
			var newcol ColumnStorage = new(StorageSCMER) // currently only scmer-storages
			var i uint32
			var reader ColumnReader
			for {
				// scan phase
				i = 0
				reader = c.GetCachedReader() // must NOT use newCachedColumnReader: it strips OverlayBlob
				newcol.prepare()
				if sortPerm != nil {
					for _, globalID := range sortPerm {
						if globalID < t.main_count {
							newcol.scan(i, reader.GetValue(globalID))
						} else {
							newcol.scan(i, t.getDelta(int(globalID-t.main_count), col))
						}
						i++
					}
				} else {
					// scan main
					for idx := uint32(0); idx < t.main_count; idx++ {
						if deletions.Get(uint(idx)) {
							continue
						}
						newcol.scan(i, reader.GetValue(idx))
						i++
					}
					// scan delta
					for idx := 0; idx < maxInsertIndex; idx++ {
						if deletions.Get(uint(t.main_count + uint32(idx))) {
							continue
						}
						newcol.scan(i, t.getDelta(idx, col))
						i++
					}
				}
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
			i = 0
			reader = c.GetCachedReader() // must NOT use newCachedColumnReader: it strips OverlayBlob
			if sortPerm != nil {
				for _, globalID := range sortPerm {
					if globalID < t.main_count {
						newcol.build(i, reader.GetValue(globalID))
					} else {
						newcol.build(i, t.getDelta(int(globalID-t.main_count), col))
					}
					i++
				}
			} else {
				// build main
				for idx := uint32(0); idx < t.main_count; idx++ {
					if deletions.Get(uint(idx)) {
						continue
					}
					newcol.build(i, reader.GetValue(idx))
					i++
				}
				// build delta
				for idx := 0; idx < maxInsertIndex; idx++ {
					if deletions.Get(uint(t.main_count + uint32(idx))) {
						continue
					}
					newcol.build(i, t.getDelta(idx, col))
					i++
				}
			}
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
				f.Close()
			}
		}
		b.WriteString(") -> ")
		b.WriteString(fmt.Sprint(result.main_count))
		fmt.Println(b.String())

		// Eagerly rebuild indexes with sufficient Savings so the first
		// query after rebuild does not pay a cold-start full-scan penalty.
		for _, idx := range result.Indexes {
			if idx.Savings >= 2.0 && !idx.active {
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
					idx.buildIndex(idx.buildGetters())
					GlobalCache.AddItem(idx, int64(idx.ComputeSize()), TypeIndex, indexCleanup, indexLastUsed, indexGetScore)
				}
			}
		}

		// Do not persist schema from inside shard rebuild; callers
		// publish the new shard pointer and then save atomically at the
		// table/database level to avoid transient, inconsistent schemas.

		if t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged {
			// remove old log file
			// TODO: this should be in sync with setting the new pointer
			logfile := t.logfile
			t.logfile = nil
			if logfile != nil {
				logfile.Close()
			}
			t.t.schema.persistence.RemoveLog(t.uuid.String())
		}

		// Only after a successful rebuild, schedule old shard files for deletion.
		runtime.SetFinalizer(t, func(t *storageShard) {
			t.RemoveFromDisk()
		})
	} else {
		// otherwise: table stays the same
		result.uuid = t.uuid // copy uuid in case nothing changes
		result.columns = t.columns
		result.deltaColumns = t.deltaColumns
		result.main_count = t.main_count
		result.inserts = t.inserts
		result.deletions = deletions
		result.Indexes = t.Indexes
		result.hashmaps1 = t.hashmaps1
		result.hashmaps2 = t.hashmaps2
		result.hashmaps3 = t.hashmaps3
		if t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged {
			if t.logfile != nil {
				t.logfile.Close()
			}
			t.t.schema.persistence.RemoveLog(t.uuid.String())
			result.logfile = result.t.schema.persistence.OpenLog(result.uuid.String())
		}
		t.logfile = nil
		// Update index parent pointers to reference the new shard
		for _, idx := range result.Indexes {
			idx.t = result
		}
		t.mu.Unlock()
		locked = false
		// Deregister old shard — result replaces it
		GlobalCache.Remove(t)
		removedFromCache = true
	}
	// Unlock result before registration (ComputeSize needs RLock)
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
