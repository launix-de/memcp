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
		result += scm.ComputeSize(s.inserts)
		for _, idx := range s.Indexes {
			result += idx.ComputeSize()
		}
		// TODO: hashmaps for unique
		return result
	}
	result += s.deletions.ComputeSize()
	result += scm.ComputeSize(s.inserts)
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
		var log chan interface{}
		log, u.logfile = u.t.schema.persistence.ReplayLog(u.uuid.String())
		numEntriesRestored := 0
		for logentry := range log {
			numEntriesRestored++
			switch l := logentry.(type) {
			case LogEntryDelete:
				u.deletions.Set(l.idx, true) // mark deletion
			case LogEntryInsert:
				u.insertDataset(l.cols, l.values, nil)
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
func (u *storageShard) ensureColumnLoaded(colName string) ColumnStorage {
	if u.columns[colName] != nil {
		return u.columns[colName]
	}
	if u.t.PersistencyMode == Memory {
		u.columns[colName] = new(StorageSparse)
		return u.columns[colName]
	}
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
	if cnt > u.main_count {
		u.main_count = cnt
	}
	u.columns[colName] = columnstorage
	return columnstorage
}

// ensureMainCount guarantees main_count is initialized by loading one column if needed.
func (u *storageShard) ensureMainCount() {
	if u.main_count != 0 {
		return
	}
	if u.t == nil {
		return
	}
	// try to load columns in table order until we get a non-zero main_count
	for _, c := range u.t.Columns {
		if _, ok := u.columns[c.Name]; ok {
			if u.columns[c.Name] == nil {
				u.ensureColumnLoaded(c.Name)
				if u.main_count != 0 {
					return
				}
			}
		}
	}
}

// SharedResource impl for shard with lazy load
func (s *storageShard) GetState() SharedState { return s.srState }
func (s *storageShard) GetRead() func() {
	s.ensureLoaded()
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
	if s.t != nil && s.t.PersistencyMode == Memory {
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
			func() {
				t.mu.Lock()         // write lock
				defer t.mu.Unlock() // write lock

				// update statement -> also perform an insert
				// TODO: check if we can do in-place editing in the delta storage (if idx > t.main_count)
				changes := a[0].([]scm.Scmer)
				// build the whole dataset from storage
				cols := make([]string, len(t.columns))
				d2 := make([]scm.Scmer, 0, len(t.columns))
				for k, v := range t.columns {
					colidx, ok := t.deltaColumns[k]
					if !ok {
						colidx = len(t.deltaColumns)
						t.deltaColumns[k] = colidx
					}
					for len(d2) <= colidx {
						d2 = append(d2, nil)
					}
					cols[colidx] = k
					if idx < t.main_count {
						d2[colidx] = v.GetValue(idx)
					} else {
						d2[colidx] = t.getDelta(int(idx-t.main_count), k)
					}
				}
				// now d2 contains the old col (TODO: preserve OLD and NEW for triggers or bind them to trigger variables)
				for j := 0; j < len(changes); j += 2 {
					colidx, ok := t.deltaColumns[scm.String(changes[j])]
					if !ok {
						panic("UPDATE on invalid column: " + scm.String(changes[j]))
					}
					if d2[colidx] != changes[j+1] {
						d2[colidx] = changes[j+1]
						result = true // mark that something has changed
					}
				}
				if !result { // only do a write if something changed
					return // leave inner func to unlock
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
			if withTrigger {
				// TODO: before/after update trigger
			}
		} else {
			// delete
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
			if withTrigger {
				// TODO: before/after delete trigger
			}
		}
		if result && t.next != nil {
			// also change in next storage
			// idx translation (subtract the amount of deletions from that idx)
			idx2 := idx - t.deletions.CountUntil(idx)
			t.next.UpdateFunction(idx2, false)(a...) // propagate to succeeding shard
		}
		return result // maybe instead return UpdateFunction for newly inserted item??
	}
}

func (t *storageShard) ColumnReader(col string) func(uint) scm.Scmer {
	cstorage, ok := t.columns[col]
	if !ok {
		panic("Column does not exist: `" + t.t.schema.Name + "`.`" + t.t.Name + "`.`" + col + "`")
	}
	if cstorage == nil {
		cstorage = t.ensureColumnLoaded(col)
	}
	return func(idx uint) scm.Scmer {
		if idx < t.main_count {
			return cstorage.GetValue(idx)
		} else {
			return t.getDelta(int(idx-t.main_count), col)
		}
	}
}

func (t *storageShard) Insert(columns []string, values [][]scm.Scmer, alreadyLocked bool, onFirstInsertId func(int64)) {
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
	// TODO: before/after insert trigger
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
		if c.AutoIncrement || c.Default != nil {
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
				newrow[cidx] = int64(Auto_increment)
			} else if c.Default != nil {
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

func (t *storageShard) GetRecordidForUnique(columns []string, values []scm.Scmer) (result uint, present bool) {
	t.mu.RLock()
	/* TODO: this all does not work since a StorageInt stores int64 while the user might have provided string or float64 with the same content; also hashmaps eat up too much space */
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

	// TODO: optimization potential -> canonical form is string or NULL
	mcols := make([]ColumnStorage, len(columns))
	dcols := make([]int, len(columns))
	for i, col := range columns {
		mcols[i] = t.columns[col]
		dcols[i] = t.deltaColumns[col]
	}
	for i := uint(0); i < t.main_count; i++ {
		for j, v := range values {
			if !scm.Equal(mcols[j].GetValue(i), v) {
				goto skipnextmain
			}
		}
		result = i
		present = true
		goto found

	skipnextmain:
	}
	for i := uint(0); i < uint(len(t.inserts)); i++ {
		for j, v := range values {
			if !scm.Equal(t.inserts[i][dcols[j]], v) {
				goto skipnextdelta
			}
		}
		result = i + t.main_count
		present = true
		goto found

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
		} else {
			return nil
		}
	} else {
		return nil
	}
}

func (t *storageShard) RemoveFromDisk() {
	// close logfile
	if t.logfile != nil {
		t.logfile.Close()
	}
	for _, col := range t.t.Columns {
		// delete column from file
		t.t.schema.persistence.RemoveColumn(t.uuid.String(), col.Name)
	}
	t.t.schema.persistence.RemoveLog(t.uuid.String())
}

// rebuild main storage from main+delta
func (t *storageShard) rebuild(all bool) *storageShard {

	// concurrency! when rebuild is run in background, inserts and deletions into and from old delta storage must be duplicated to the ongoing process
	t.mu.Lock()
	if t.next != nil {
		t.mu.Unlock()
		// lock+unlock the next shard so we don't return too early (sync hazards)
		t.next.mu.Lock()
		t.next.mu.Unlock()
		return t.next // already rebuilding (happens on parallel inserts)
		// possible problem: this call may return the t.next shard faster than the competing rebuild() call that actually rebuilds; maybe use a additional lock on t.next??
	}
	result := new(storageShard)
	result.t = t.t
	t.next = result
	result.mu.Lock() // interlock so no one will rebuild the shard twice
	defer result.mu.Unlock()

	// now read out deletion list
	maxInsertIndex := len(t.inserts)
	// copy-freeze deletions so we don't have to lock anything
	deletions := t.deletions.Copy()
	// from now on, we can rebuild with no hurry; inserts and update/deletes on the previous shard will propagate to us, too

	if all || maxInsertIndex > 0 || deletions.Count() > 0 {
		result.uuid, _ = uuid.NewRandom() // new uuid, serialize
		// SetFinalizer to old shard to delete files from disk
		runtime.SetFinalizer(t, func(t *storageShard) {
			t.RemoveFromDisk()
		})

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
		result.t.schema.save()

		if t.t.PersistencyMode == Safe || t.t.PersistencyMode == Logged {
			// remove old log file
			// TODO: this should be in sync with setting the new pointer
			logfile := t.logfile
			t.logfile = nil
			logfile.Close()
			t.t.schema.persistence.RemoveLog(t.uuid.String())
		}
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
	}
	return result
}
