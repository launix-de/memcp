/*
Copyright (C) 2023  Carl-Philip Hänsch

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
import "bufio"
import "strings"
import "reflect"
import "runtime"
import "encoding/json"
import "encoding/binary"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/NonLockingReadMap"

type storageShard struct {
	t *table
	uuid uuid.UUID // uuid.String()
	// main storage
	main_count uint // size of main storage
	columns map[string]ColumnStorage
	// delta storage
	deltaColumns map[string]int
	inserts [][]scm.Scmer // items added to storage
	deletions NonLockingReadMap.NonBlockingBitMap // items removed from main or inserts (based on main_count + i)
	logfile *os.File // only in safe mode
	mu sync.RWMutex // delta write lock (working on main storage is lock free)
	next *storageShard
	// indexes
	indexes []*StorageIndex
}

func (s *storageShard) Size() uint {
	var result uint = 14*8
	s.mu.RLock()
	for _, c := range s.columns {
		result += c.Size()
	}
	s.mu.RUnlock()
	result += uint(s.deletions.Size()) // approximation of delete map
	result += uint(len(s.inserts) * len(s.deltaColumns)) * 32 // heuristic
	return result
}

func (u *storageShard) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.uuid.String())
}
func (u *storageShard) UnmarshalJSON(data []byte) error {
	u.uuid.UnmarshalText(data)
	u.columns = make(map[string]ColumnStorage)
	u.deltaColumns = make(map[string]int)
	u.deletions.Reset()
	// the rest of the unmarshalling is done in the caller because u.t is nil in the moment
	return nil
}
func (u *storageShard) load(t *table) {
	u.t = t
	// load the columns
	for _, col := range u.t.Columns {
		if t.PersistencyMode == Memory {
			// recreate the shards empty (because in memory-mode we forget all data)
			u.columns[col.Name] = new(StorageSCMER)
		} else {
			// read column from file
			f, err := os.Open(u.t.schema.path + u.uuid.String() + "-" + col.Name)
			if err != nil {
				// file does not exist -> no data available
				u.columns[col.Name] = new(StorageSCMER)
				continue
			}
			var magicbyte uint8 // type of that column
			err = binary.Read(f, binary.LittleEndian, &magicbyte)
			if err != nil {
				// empty storage
				u.columns[col.Name] = new(StorageSCMER)
				continue
			}

			fmt.Println("loading storage "+u.t.schema.path + u.uuid.String() + "-" + col.Name+" of type", magicbyte)

			columnstorage := reflect.New(storages[magicbyte]).Interface().(ColumnStorage)
			u.main_count = columnstorage.Deserialize(f) // read; ownership of f goes to Deserialize, so they will free the handle
			u.columns[col.Name] = columnstorage
		}
	}

	if t.PersistencyMode == Safe {
		f, err := os.OpenFile(u.t.schema.path + u.uuid.String() + ".log", os.O_RDWR|os.O_CREATE, 0750)
		if err != nil {
			panic(err)
		}
		u.logfile = f
		fi, _ := u.logfile.Stat()
		if fi.Size() > 0 {
			fmt.Println("restoring delta storage from logfile " + u.t.schema.path + u.uuid.String() + ".log")
			scanner := bufio.NewScanner(u.logfile)
			for scanner.Scan() {
				b := scanner.Bytes()
				if string(b) == "" {
					// nop
				} else if string(b[0:7]) == "delete " {
					var idx uint
					json.Unmarshal(b[7:], &idx)
					u.deletions.Set(idx, true) // mark deletion
				} else if string(b[0:7]) == "insert " {
					var d dataset
					json.Unmarshal(b[7:], &d)
					u.insertDataset(d)
				} else {
					panic("unknown log sequence: " + string(b))
				}
			}
		}
	}
}

func NewShard(t *table) *storageShard {
	result := new(storageShard)
	result.uuid, _ = uuid.NewRandom()
	result.t = t
	result.columns = make(map[string]ColumnStorage)
	result.deltaColumns = make(map[string]int)
	result.deletions.Reset()
	for _, column := range t.Columns {
		result.columns[column.Name] = new (StorageSCMER)
	}
	if t.PersistencyMode == Safe {
		f, _ := os.Create(result.t.schema.path + result.uuid.String() + ".log")
		result.logfile = f
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
		t.mu.Lock() // write lock

		result := false // result = true when update was possible; false if there was a RESTRICT
		if len(a) > 0 {
			// update statement -> also perform an insert
			// TODO: check if we can do in-place editing in the delta storage (if idx > t.main_count)
			changes := a[0].([]scm.Scmer)
			// build the whole dataset from storage
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
				if idx < t.main_count {
					d2[colidx] = v.GetValue(idx)
				} else {
					d2[colidx] = t.getDelta(int(idx - t.main_count), k)
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
			if result { // only do a write if something changed
				var d dataset
				if t.t.Unique != nil || t.t.PersistencyMode == Safe {
					// we need a dataset object (slow)
					d = make(dataset, 0, 2 * len(t.columns))
					for col, idx := range t.deltaColumns {
						d = append(d, col, d2[idx])
					}
				}

				// unique constraint checking
				if t.t.Unique != nil {
					t.t.uniquelock.Lock()
					t.mu.Unlock() // release write lock, so the scan can be performed
					err := t.t.GetUniqueErrorsFor(d)
					t.mu.Lock() // write lock
					if err != nil {
						t.t.uniquelock.Unlock()
						panic(err)
					}
				}

				t.deletions.Set(idx, true) // mark as deleted
				t.inserts = append(t.inserts, d2)
				if t.t.PersistencyMode == Safe {
					var b strings.Builder
					b.Write([]byte("delete "))
					tmp, _ := json.Marshal(idx)
					b.Write(tmp)
					b.Write([]byte("\ninsert "))
					tmp, _ = json.Marshal(d)
					b.Write(tmp)
					b.Write([]byte("\n"))
					t.logfile.WriteString(b.String())
				}
				if t.t.Unique != nil {
					t.t.uniquelock.Unlock()
				}
				t.mu.Unlock()
				if t.t.PersistencyMode == Safe {
					defer t.logfile.Sync() // write barrier after the lock, so other threads can continue without waiting for the other thread to write
				}
				if withTrigger {
					// TODO: before/after update trigger
				}
			}
		} else {
			// delete
			t.deletions.Set(idx, true) // mark as deleted
			if t.t.PersistencyMode == Safe {
				var b strings.Builder
				b.Write([]byte("delete "))
				tmp, _ := json.Marshal(idx)
				b.Write(tmp)
				b.Write([]byte("\n"))
				t.logfile.WriteString(b.String())
			}
			result = true
			t.mu.Unlock()
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
	return func(idx uint) scm.Scmer {
		if idx < t.main_count {
			return cstorage.GetValue(idx)
		} else {
			return t.getDelta(int(idx - t.main_count), col)
		}
	}
}

func (t *storageShard) Insert(d dataset) {
	t.mu.Lock()
	d2 := t.insertDataset(d)
	if t.t.PersistencyMode == Safe {
		var b strings.Builder
		b.Write([]byte("insert "))
		tmp, _ := json.Marshal(d)
		b.Write(tmp)
		b.Write([]byte("\n"))
		t.logfile.WriteString(b.String())
	}
	for _, index := range t.indexes {
		// add to delta indexes
		index.deltaBtree.ReplaceOrInsert(indexPair{len(t.inserts)-1, d2})
	}
	if t.next != nil {
		// also insert into next storage
		t.next.Insert(d)
	}
	t.mu.Unlock()
	if t.t.PersistencyMode == Safe {
		t.logfile.Sync() // write barrier after the lock, so other threads can continue without waiting for the other thread to write
	}
	// TODO: before/after insert trigger
}

// contract: must only be called inside full write mutex mu.Lock()
func (t *storageShard) insertDataset(d dataset) []scm.Scmer {
	result := make([]scm.Scmer, 0, len(t.deltaColumns))
	for i := 0; i < len(d); i += 2 {
		// copy all dataset entries into packed array
		colidx, ok := t.deltaColumns[scm.String(d[i])]
		if !ok {
			// acquire new column
			colidx := len(t.deltaColumns)
			t.deltaColumns[scm.String(d[i])] = colidx
		}
		for len(result) <= colidx {
			result = append(result, nil)
		}
		result[colidx] = d[i+1]
	}
	t.inserts = append(t.inserts, result)
	return result
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
	if t.t.PersistencyMode == Safe {
		t.logfile.Close()
	}
	for _, col := range t.t.Columns {
		// delete column from file
		os.Remove(t.t.schema.path + t.uuid.String() + "-" + col.Name)
	}
	os.Remove(t.t.schema.path + t.uuid.String() + ".log")
}

// rebuild main storage from main+delta
func (t *storageShard) rebuild() *storageShard {

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
	t.mu.Unlock()

	// now read out deletion list
	t.mu.RLock()
	maxInsertIndex := len(t.inserts)
	// copy-freeze deletions so we don't have to lock anything
	deletions := t.deletions.Copy()
	t.mu.RUnlock()
	// from now on, we can rebuild with no hurry; inserts and update/deletes on the previous shard will propagate to us, too

	if maxInsertIndex > 0 || deletions.Count() > 0 {
		result.uuid, _ = uuid.NewRandom() // new uuid, serialize
		// SetFinalizer to old shard to delete files from disk
		runtime.SetFinalizer(t, func (t *storageShard) {
			t.RemoveFromDisk()
		})

		var b strings.Builder
		b.WriteString("rebuilding shard for table ")
		b.WriteString(t.t.Name)
		b.WriteString("(")

		// prepare delta storage
		result.columns = make(map[string]ColumnStorage)
		result.deltaColumns = make(map[string]int)
		result.deletions.Reset()
		if t.t.PersistencyMode == Safe {
			// safe mode: also write all deltas to disk
			f, err := os.Create(result.t.schema.path + result.uuid.String() + ".log")
			if err != nil {
				panic(err)
			}
			t.logfile = f
		}

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
				f, err := os.Create(result.t.schema.path + result.uuid.String() + "-" + col)
				if err != nil {
					panic(err)
				}
				newcol.Serialize(f) // col takes ownership of f, so they will defer f.Close() at the right time
			}
		}
		b.WriteString(") -> ")
		b.WriteString(fmt.Sprint(result.main_count))
		fmt.Println(b.String())
		rebuildIndexes(t, result)
		result.t.schema.save()
	} else {
		// otherwise: table stays the same
		result.uuid = t.uuid // copy uuid in case nothing changes
		result.columns = t.columns
		result.deltaColumns = t.deltaColumns
		result.main_count = t.main_count
		result.inserts = t.inserts
		result.deletions = deletions
		result.indexes = t.indexes
	}
	return result
}
