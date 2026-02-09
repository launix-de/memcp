/*
Copyright (C) 2023-2024  Carl-Philip Hänsch

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
import "strings"
import "reflect"
import "runtime"
import "encoding/json"
import "encoding/binary"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/NonLockingReadMap"

type storageShard struct {
	t    *table
	uuid uuid.UUID // uuid.String()
	// main storage
	main_count uint // size of main storage
	columns    map[string]ColumnStorage
	// delta storage
	deltaColumns map[string]int
	inserts      [][]scm.Scmer                       // items added to storage
	deletions    NonLockingReadMap.NonBlockingBitMap // items removed from main or inserts (based on main_count + i)
	logfile      PersistenceLogfile                  // only in safe mode
	mu           sync.RWMutex                        // delta write lock (working on main storage is lock free)
	uniquelock   sync.Mutex                          // unique insert lock (only used in the sharded case)
	next         *storageShard                       // TODO: also make a next-partition-schema
	// indexes
	Indexes    []*StorageIndex // sorted keys
	indexMutex sync.Mutex
	hashmaps1  map[[1]string]map[[1]scm.Scmer]uint // hashmaps for single columned unique keys
	hashmaps2  map[[2]string]map[[2]scm.Scmer]uint // hashmaps for single columned unique keys
	hashmaps3  map[[3]string]map[[3]scm.Scmer]uint // hashmaps for single columned unique keys

	// lazy-loading/shared-resource state
	srState SharedState
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
	u.hashmaps1 = make(map[[1]string]map[[1]scm.Scmer]uint)
	u.hashmaps2 = make(map[[2]string]map[[2]scm.Scmer]uint)
	u.hashmaps3 = make(map[[3]string]map[[3]scm.Scmer]uint)
	u.deletions.Reset()
	u.srState = COLD
	// the rest of the unmarshalling is done in the caller because u.t is nil in the moment
	return nil
}
func (u *storageShard) load(t *table) {
	u.t = t
	// mark columns for lazy loading
	for _, col := range u.t.Columns {
		u.columns[col.Name] = nil
	}

	if t.PersistencyMode == Safe || t.PersistencyMode == Logged {
		// Replaying the log mutates inserts/deletions; use shard write lock
		u.mu.Lock()
		defer u.mu.Unlock()
		var log chan interface{}
		log, u.logfile = u.t.schema.persistence.ReplayLog(u.uuid.String())
		numEntriesRestored := 0
		for logentry := range log {
			numEntriesRestored++
			switch l := logentry.(type) {
			case LogEntryDelete:
				u.deletions.Set(l.idx, true) // mark deletion
			case LogEntryInsert:
				u.insertDatasetFromLog(l.cols, l.values)
			default:
				panic("unknown log sequence: " + fmt.Sprint(l))
			}
		}
		if numEntriesRestored > 0 {
			fmt.Println("restoring delta storage from database "+u.t.schema.Name+" shard "+u.uuid.String()+":", numEntriesRestored, "entries")
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
		if u.t.PersistencyMode == Memory {
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
		if cnt > u.main_count {
			u.main_count = cnt
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

// getColumnStorageOrPanic returns a stable pointer to a column's storage.
// It never reads u.columns without holding the shard lock and loads on demand.
func (u *storageShard) getColumnStorageOrPanic(colName string) ColumnStorage {
	// try under read lock
	u.mu.RLock()
	cs, present := u.columns[colName]
	u.mu.RUnlock()
	if !present {
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
			panic("Column does not exist: `" + u.t.schema.Name + "`.`" + u.t.Name + "`.`" + colName + "`")
		}
		if cs != nil {
			return cs
		}
		return u.ensureColumnLoaded(colName, true)
	}
	return u.getColumnStorageOrPanic(colName)
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
	return func() {}
}
func (s *storageShard) GetExclusive() func() {
	s.ensureLoaded()
	s.srState = WRITE
	return func() {}
}

func (s *storageShard) ensureLoaded() {
	if s.srState != COLD {
		return
	}
	// materialize shard from disk
	s.load(s.t)
	// memory engine shards stay WRITE to bypass LRU later
	if s.t.PersistencyMode == Memory {
		s.srState = WRITE
	} else {
		s.srState = SHARED
	}
}

func NewShard(t *table) *storageShard {
	result := new(storageShard)
	result.uuid, _ = uuid.NewRandom()
	result.t = t
	result.columns = make(map[string]ColumnStorage)
	result.deltaColumns = make(map[string]int)
	result.hashmaps1 = make(map[[1]string]map[[1]scm.Scmer]uint)
	result.hashmaps2 = make(map[[2]string]map[[2]scm.Scmer]uint)
	result.hashmaps3 = make(map[[3]string]map[[3]scm.Scmer]uint)
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

func (t *storageShard) Count() uint {
	return t.main_count + uint(len(t.inserts)) - t.deletions.Count()
}

func (t *storageShard) UpdateFunction(idx uint, withTrigger bool) func(...scm.Scmer) scm.Scmer {
	// returns a callback with which you can delete or update an item
	return func(a ...scm.Scmer) scm.Scmer {
		//fmt.Println("update/delete", a)
		// TODO: check foreign keys (new value of column must be present in referenced table)
		// TODO: check foreign key removal (old value is referenced in another table)

		result := false // result = true when update was possible; false if there was a RESTRICT
		if len(a) > 0 {
			// update command
			var triggerOldRow, triggerNewRow dataset // for AFTER UPDATE triggers
			func() {
				t.mu.Lock()         // write lock
				defer t.mu.Unlock() // write lock

				// update statement -> also perform an insert
				// TODO: check if we can do in-place editing in the delta storage (if idx > t.main_count)
				changes := mustScmerSlice(a[0], "update changes")
				// build the whole dataset from storage
				cols := make([]string, len(t.columns))
				d2 := make([]scm.Scmer, 0, len(t.columns))
				for k := range t.columns {
					// Access to t.columns is protected by t.mu and storages may be
					// lazily loaded; ensure we obtain a non-nil storage pointer.
					cs := t.getColumnStorageOrPanicEx(k, true)
					colidx, ok := t.deltaColumns[k]
					if !ok {
						colidx = len(t.deltaColumns)
						t.deltaColumns[k] = colidx
					}
					for len(d2) <= colidx {
						d2 = append(d2, scm.NewNil())
					}
					cols[colidx] = k
					if idx < t.main_count {
						d2[colidx] = cs.GetValue(idx)
					} else {
						d2[colidx] = t.getDelta(int(idx-t.main_count), k)
					}
				}
				// now d2 contains the old row values
				// copy slice for triggers before modifying (scheme values are immutable, but Go slice is modified)
				if withTrigger && len(t.t.Triggers) > 0 {
					triggerOldRow = append(dataset{}, d2...)
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
					}
				}

				// Execute BEFORE UPDATE triggers (can modify d2)
				if withTrigger && triggerOldRow != nil {
					d2 = t.t.ExecuteBeforeUpdateTriggers(triggerOldRow, d2)
					// Recheck if anything changed after trigger modifications
					result = false
					for i, v := range d2 {
						if !scm.Equal(triggerOldRow[i], v) {
							result = true
							break
						}
					}
				}

				if !result { // only do a write if something changed
					return // leave inner func to unlock
				}

				// save new row for triggers (d2 now contains new values)
				if withTrigger && len(t.t.Triggers) > 0 {
					triggerNewRow = d2 // no copy needed, d2 won't be modified after this
				}

				// unique constraint checking
				if t.t.Unique != nil {
					t.deletions.Set(idx, true) // mark as deleted
					t.mu.Unlock()              // release write lock, so the scan can be performed
					t.t.ProcessUniqueCollision(cols, [][]scm.Scmer{d2}, false, func(values [][]scm.Scmer) {
						t.mu.Lock() // start write lock
					}, nil, func(errmsg string, data []scm.Scmer) {
						t.mu.Lock()                 // start write lock
						t.deletions.Set(idx, false) // mark as undeleted
						panic("Unique key constraint violated in table " + t.t.Name + ": " + errmsg)
					}, 0)
				} else {
					t.deletions.Set(idx, true) // mark as deleted
				}

				t.insertDataset(cols, [][]scm.Scmer{d2}, nil)
				if t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged {
					t.logfile.Write(LogEntryDelete{idx})
					t.logfile.Write(LogEntryInsert{cols, [][]scm.Scmer{d2}})
				}
			}()
			if t.t.PersistencyMode == Safe {
				defer t.logfile.Sync() // write barrier after the lock, so other threads can continue without waiting for the other thread to write
			}
			if withTrigger && triggerOldRow != nil {
				t.t.ExecuteTriggers(AfterUpdate, triggerOldRow, triggerNewRow)
			}
		} else {
			// delete
			var triggerDeletedRow dataset // for BEFORE/AFTER DELETE trigger

			// capture row data for triggers before deletion (outside lock for BEFORE trigger)
			if withTrigger && len(t.t.Triggers) > 0 {
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

				// Execute BEFORE DELETE triggers (can abort delete by returning false)
				if !t.t.ExecuteBeforeDeleteTriggers(triggerDeletedRow) {
					return scm.NewBool(false) // trigger aborted delete
				}
			}

			func() {
				t.mu.Lock()         // write lock
				defer t.mu.Unlock() // write lock

				t.deletions.Set(idx, true) // mark as deleted
				if t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged {
					t.logfile.Write(LogEntryDelete{idx})
				}
				result = true
			}()
			if t.t.PersistencyMode == Safe {
				defer t.logfile.Sync() // write barrier after the lock, so other threads can continue without waiting for the other thread to write
			}
			if withTrigger && triggerDeletedRow != nil {
				t.t.ExecuteTriggers(AfterDelete, triggerDeletedRow, nil)
			}
		}
		if result && t.next != nil {
			// also change in next storage
			// idx translation (subtract the amount of deletions from that idx)
			idx2 := idx - t.deletions.CountUntil(idx)
			t.next.UpdateFunction(idx2, false)(a...) // propagate to succeeding shard
		}
		return scm.NewBool(result) // maybe instead return UpdateFunction for newly inserted item??
	}
}

func (t *storageShard) ColumnReader(col string) func(uint) scm.Scmer {
	cstorage := t.getColumnStorageOrPanic(col)
	return func(idx uint) scm.Scmer {
		if idx < t.main_count {
			return cstorage.GetValue(idx)
		} else {
			return t.getDelta(int(idx-t.main_count), col)
		}
	}
}

func (t *storageShard) Insert(columns []string, values [][]scm.Scmer, alreadyLocked bool, onFirstInsertId func(int64)) {
	// Execute BEFORE INSERT triggers (can modify values)
	if len(t.t.Triggers) > 0 {
		values = t.t.ExecuteBeforeInsertTriggers(columns, values)
	}

	if !alreadyLocked {
		t.mu.Lock()
	}
	t.insertDataset(columns, values, onFirstInsertId)
	if t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged {
		t.logfile.Write(LogEntryInsert{columns, values})
	}
	if t.next != nil {
		// also insert into next storage
		t.next.Insert(columns, values, false, nil)
	}
	if !alreadyLocked {
		t.mu.Unlock()
	}
	if t.t.PersistencyMode == Safe {
		t.logfile.Sync() // write barrier after the lock, so other threads can continue without waiting for the other thread to write
	}
	// execute AFTER INSERT triggers
	if len(t.t.Triggers) > 0 {
		for _, row := range values {
			t.t.ExecuteTriggers(AfterInsert, nil, row)
		}
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
		recid := uint(len(t.inserts)) + t.main_count
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
			if index.deltaBtree != nil {
				index.deltaBtree.ReplaceOrInsert(indexPair{int(recid), newrow})
			}
		}
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
		recid := uint(len(t.inserts)) + t.main_count
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
			if index.deltaBtree != nil {
				index.deltaBtree.ReplaceOrInsert(indexPair{int(recid), newrow})
			}
		}
	}
}

func (t *storageShard) GetRecordidForUnique(columns []string, values []scm.Scmer) (result uint, present bool) {
	/* TODO: this all does not work since a StorageInt stores int64 while the user might have provided string or float64 with the same content; also hashmaps eat up too much space */
	// Preload main storages and establish main_count without holding any shard lock
	t.ensureMainCount(false)
	mcols := make([]ColumnStorage, len(columns))
	for i, col := range columns {
		mcols[i] = t.getColumnStorageOrPanic(col)
	}

	// From here on, read under shard read lock for a consistent snapshot of deletions/inserts/deltaColumns
	t.mu.RLock()
	/*
		if len(columns) == 1 {
			columns_ := (*[1]string)(columns)
			values_ := (*[1]scm.Scmer)(values)
			hm, ok := t.hashmaps1[*columns_]
			if !ok {
				// no hashmap entry? create the hashmap
				t.mu.RUnlock()
				t.mu.Lock()
				hm := make(map[[1]scm.Scmer]uint)
				col := []ColumnStorage{
					t.columns[columns[0]],
				}
				for i := uint(0); i < t.main_count; i++ {
					hm[[1]scm.Scmer{
						col[0].GetValue(i),
					}] = i
				}
				dcolids := []int{
					t.deltaColumns[columns[0]],
				}
				for i := uint(0); i < uint(len(t.inserts)); i++ {
					hm[[1]scm.Scmer{
						t.inserts[i][dcolids[0]],
					}] = i + t.main_count
				}
				t.hashmaps1[*columns_] = hm
				t.mu.Unlock()
				return t.GetRecordidForUnique(columns, values) // retry
			}
			result, present = hm[*values_] // read recid from hashmap
		} else
		if len(columns) == 2 {
			columns_ := (*[2]string)(columns)
			values_ := (*[2]scm.Scmer)(values)
			hm, ok := t.hashmaps2[*columns_]
			if !ok {
				// no hashmap entry? create the hashmap
				t.mu.RUnlock()
				t.mu.Lock()
				hm := make(map[[2]scm.Scmer]uint)
				col := []ColumnStorage{
					t.columns[columns[0]],
					t.columns[columns[1]],
				}
				for i := uint(0); i < t.main_count; i++ {
					hm[[2]scm.Scmer{
						col[0].GetValue(i),
						col[1].GetValue(i),
					}] = i
				}
				dcolids := []int{
					t.deltaColumns[columns[0]],
					t.deltaColumns[columns[1]],
				}
				for i := uint(0); i < uint(len(t.inserts)); i++ {
					hm[[2]scm.Scmer{
						t.inserts[i][dcolids[0]],
						t.inserts[i][dcolids[1]],
					}] = i + t.main_count
				}
				t.hashmaps2[*columns_] = hm
				t.mu.Unlock()
				return t.GetRecordidForUnique(columns, values) // retry
			}
			result, present = hm[*values_] // read recid from hashmap
		} else
		if len(columns) == 3 {
			columns_ := (*[3]string)(columns)
			values_ := (*[3]scm.Scmer)(values)
			hm, ok := t.hashmaps3[*columns_]
			if !ok {
				// no hashmap entry? create the hashmap
				t.mu.RUnlock()
				t.mu.Lock()
				hm := make(map[[3]scm.Scmer]uint)
				col := []ColumnStorage{
					t.columns[columns[0]],
					t.columns[columns[1]],
					t.columns[columns[2]],
				}
				for i := uint(0); i < t.main_count; i++ {
					hm[[3]scm.Scmer{
						col[0].GetValue(i),
						col[1].GetValue(i),
						col[2].GetValue(i),
					}] = i
				}
				dcolids := []int{
					t.deltaColumns[columns[0]],
					t.deltaColumns[columns[1]],
					t.deltaColumns[columns[2]],
				}
				for i := uint(0); i < uint(len(t.inserts)); i++ {
					hm[[3]scm.Scmer{
						t.inserts[i][dcolids[0]],
						t.inserts[i][dcolids[1]],
						t.inserts[i][dcolids[2]],
					}] = i + t.main_count
				}
				t.hashmaps3[*columns_] = hm
				t.mu.Unlock()
				return t.GetRecordidForUnique(columns, values) // retry
			}
			result, present = hm[*values_] // read recid from hashmap
		} else
	*/

	// Build delta column index mapping under lock to match current inserts layout
	dcols := make([]int, len(columns))
	dcolPresent := make([]bool, len(columns))
	for i, col := range columns {
		if idx, ok := t.deltaColumns[col]; ok {
			dcols[i] = idx
			dcolPresent[i] = true
		} else {
			dcols[i] = -1
			dcolPresent[i] = false
		}
	}
	var recid uint
	for i := uint(0); i < t.main_count; i++ {
		for j, v := range values {
			if !scm.Equal(mcols[j].GetValue(i), v) {
				goto skipnextmain
			}
		}
		// prefer non-deleted main rows; if deleted, keep searching
		if !t.deletions.Get(i) {
			result = i
			present = true
			goto found
		}

	skipnextmain:
	}
	for i := uint(0); i < uint(len(t.inserts)); i++ {
		item := t.inserts[i]
		for j, v := range values {
			if dcolPresent[j] {
				idx := dcols[j]
				got := scm.NewNil()
				if idx >= 0 && idx < len(item) {
					got = item[idx]
				}
				if !scm.Equal(got, v) {
					goto skipnextdelta
				}
			} else {
				if !scm.Equal(scm.NewNil(), v) {
					goto skipnextdelta
				}
			}
		}
		// prefer non-deleted delta rows
		recid = i + t.main_count
		if !t.deletions.Get(recid) {
			result = recid
			present = true
			goto found
		}

	skipnextdelta:
	}
	// nothing found
	present = false

found:
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
	// close logfile
	if t.logfile != nil {
		t.logfile.Close()
	}
	// Release blob refcounts before removing column files.
	// Skip for COLD shards (columns not loaded) -- orphaned blobs will be cleaned by (clean).
	for _, col := range t.t.Columns {
		if cs, ok := t.columns[col.Name]; ok && cs != nil {
			if blob, ok := cs.(*OverlayBlob); ok {
				blob.ReleaseBlobs(t.main_count)
			}
		}
	}
	for _, col := range t.t.Columns {
		t.t.schema.persistence.RemoveColumn(t.uuid.String(), col.Name)
	}
	t.t.schema.persistence.RemoveLog(t.uuid.String())
}

// rebuild main storage from main+delta
func (t *storageShard) rebuild(all bool) *storageShard {

	// concurrency! when rebuild is run in background, inserts and deletions into and from old delta storage must be duplicated to the ongoing process
	t.mu.Lock()
	locked := true
	defer func() {
		if locked {
			t.mu.Unlock()
		}
	}()
	if t.next != nil {
		t.mu.Unlock()
		locked = false
		// lock+unlock the next shard so we don't return too early (sync hazards)
		t.next.mu.Lock()
		t.next.mu.Unlock()
		return t.next // already rebuilding (happens on parallel inserts)
		// possible problem: this call may return the t.next shard faster than the competing rebuild() call that actually rebuilds; maybe use a additional lock on t.next??
	}
	result := new(storageShard)
	result.t = t.t
	result.srState = WRITE // mark as live so ensureLoaded() won't reset columns
	t.next = result
	result.mu.Lock() // interlock so no one will rebuild the shard twice
	defer result.mu.Unlock()
	defer func() {
		if r := recover(); r != nil {
			// If rebuild panics, ensure we don't leave a half-built shard reachable via t.next.
			// Otherwise, later rebuild/save cycles may publish a schema referencing a UUID whose
			// column files were never written.
			t.mu.Lock()
			if t.next == result {
				t.next = nil
			}
			t.mu.Unlock()
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
		result.hashmaps1 = make(map[[1]string]map[[1]scm.Scmer]uint)
		result.hashmaps2 = make(map[[2]string]map[[2]scm.Scmer]uint)
		result.hashmaps3 = make(map[[3]string]map[[3]scm.Scmer]uint)
		result.deletions.Reset()
		if result.t.PersistencyMode == Safe || result.t.PersistencyMode == Logged {
			// safe mode: also write all deltas to disk
			result.logfile = result.t.schema.persistence.OpenLog(result.uuid.String())
		}
		t.mu.Unlock() // release lock, from now on, deletions+inserts should work
		locked = false

		// copy column data in two phases: scan, build (if delta is non-empty)
		isFirst := true
		for col, c := range t.columns {
			if isFirst {
				isFirst = false
			} else {
				b.WriteString(", ")
			}
			var newcol ColumnStorage = new(StorageSCMER) // currently only scmer-storages
			var i uint
			for {
				// scan phase
				i = 0
				newcol.prepare()
				// scan main
				for idx := uint(0); idx < t.main_count; idx++ {
					// check for deletion
					if deletions.Get(idx) {
						continue
					}
					// scan
					newcol.scan(i, c.GetValue(idx))
					i++
				}
				// scan delta
				for idx := 0; idx < maxInsertIndex; idx++ {
					// check for deletion
					if deletions.Get(t.main_count + uint(idx)) {
						continue
					}
					// scan
					newcol.scan(i, t.getDelta(idx, col))
					i++
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
			if blob, ok := newcol.(*OverlayBlob); ok {
				blob.schema = result.t.schema
			}
			newcol.init(i)
			i = 0
			// build main
			for idx := uint(0); idx < t.main_count; idx++ {
				// check for deletion
				if deletions.Get(idx) {
					continue
				}
				// build
				newcol.build(i, c.GetValue(idx))
				i++
			}
			// build delta
			for idx := 0; idx < maxInsertIndex; idx++ {
				// check for deletion
				if deletions.Get(t.main_count + uint(idx)) {
					continue
				}
				// build
				newcol.build(i, t.getDelta(idx, col))
				i++
			}
			newcol.finish()
			result.columns[col] = newcol
			result.main_count = i

			// write statistics
			b.WriteString(col) // colname
			b.WriteString(" ")
			b.WriteString(newcol.String()) // storage type (remove *storage.Storage, so it will only say SCMER, Sparse, Int or String)

			// write to disc (only if required)
			if t.t.PersistencyMode != Memory {
				f := result.t.schema.persistence.WriteColumn(result.uuid.String(), col)
				newcol.Serialize(f) // col takes ownership of f, so they will defer f.Close() at the right time
				f.Close()
			}
		}
		b.WriteString(") -> ")
		b.WriteString(fmt.Sprint(result.main_count))
		fmt.Println(b.String())
		rebuildIndexes(t, result)
		// Do not persist schema from inside shard rebuild; callers
		// publish the new shard pointer and then save atomically at the
		// table/database level to avoid transient, inconsistent schemas.

		if t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged {
			// remove old log file
			// TODO: this should be in sync with setting the new pointer
			logfile := t.logfile
			t.logfile = nil
			logfile.Close()
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
		t.mu.Unlock()
		locked = false
	}
	return result
}
