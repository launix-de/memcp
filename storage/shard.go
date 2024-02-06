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

type storageShard struct {
	t *table
	uuid uuid.UUID // uuid.String()
	// main storage
	main_count uint // size of main storage
	columns map[string]ColumnStorage
	// delta storage
	inserts []dataset // items added to storage
	deletions map[uint]struct{} // items removed from main or inserts (based on main_count + i)
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
	result += uint(len(s.deletions)) * 8 // approximation of delete map
	result += uint(len(s.inserts)) * 128 // heuristic
	return result
}

func (u *storageShard) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.uuid.String())
}
func (u *storageShard) UnmarshalJSON(data []byte) error {
	u.uuid.UnmarshalText(data)
	u.columns = make(map[string]ColumnStorage)
	u.deletions = make(map[uint]struct{})
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
				if string(b[0:7]) == "delete " {
					var idx uint
					json.Unmarshal(b[7:], &idx)
					u.deletions[idx] = struct{}{} // mark deletion
				} else if string(b[0:7]) == "insert " {
					var d dataset
					json.Unmarshal(b[7:], &d)
					u.inserts = append(u.inserts, d) // insert
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
	result.deletions = make(map[uint]struct{})
	for _, column := range t.Columns {
		result.columns[column.Name] = new (StorageSCMER)
	}
	return result
}

func (t *storageShard) Count() uint {
	return uint(int(t.main_count) + len(t.inserts) - len(t.deletions))
}

func (t *storageShard) UpdateFunction(idx uint, withTrigger bool) func(...scm.Scmer) scm.Scmer {
	// returns a callback with which you can delete or update an item
	return func(a ...scm.Scmer) scm.Scmer {
		//fmt.Println("update/delete", a)
		t.mu.Lock() // write lock
		result := false
		if len(a) > 0 {
			// update statement -> also perform an insert
			changes := a[0].([]scm.Scmer)
			// build the whole dataset from storage
			d := make(dataset, 2 * len(t.columns))
			i := 0
			for k, v := range t.columns {
				d[i] = k
				if idx < t.main_count {
					d[i+1] = v.GetValue(idx)
				} else {
					d[i+1] = t.inserts[idx - t.main_count].Get(k)
				}
				for j := 0; j < len(changes); j += 2 {
					if k == changes[j] {
						if d[i+1] != changes[j+1] {
							d[i+1] = changes[j+1]
							result = true // something changed, so return true
						}
						goto skip_set
					}
				}
				skip_set:
				i += 2
			}
			if result { // only do a write if something changed
				t.deletions[idx] = struct{}{} // mark as deleted
				t.inserts = append(t.inserts, d) // append to delta storage
				if t.t.PersistencyMode == Safe {
					t.logfile.Write([]byte("delete "))
					tmp, _ := json.Marshal(idx)
					t.logfile.Write(tmp)
					t.logfile.Write([]byte("\ninsert "))
					tmp, _ = json.Marshal(d)
					t.logfile.Write(tmp)
					t.logfile.Write([]byte("\n"))
				}
				if withTrigger {
					// TODO: before/after update trigger
				}
			}
		} else {
			// delete
			t.deletions[idx] = struct{}{} // mark as deleted
			if t.t.PersistencyMode == Safe {
				t.logfile.Write([]byte("delete "))
				tmp, _ := json.Marshal(idx)
				t.logfile.Write(tmp)
				t.logfile.Write([]byte("\n"))
			}
			result = true
			if withTrigger {
				// TODO: before/after delete trigger
			}
		}
		if result && t.next != nil {
			// also change in next storage
			// idx translation (subtract the amount of deletions from that idx)
			idx2 := idx
			for k, _ := range t.deletions {
				if k < idx {
					idx2--
				}
			}
			t.next.UpdateFunction(idx2, false)(a...) // propagate to succeeding shard
		}
		t.mu.Unlock()
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
			item := t.inserts[idx - t.main_count]
			for i := 0; i < len(item); i += 2 {
				if item[i] == col {
					return item[i+1]
				}
			}
			return nil
		}
	}
}

func (t *storageShard) Insert(d dataset) {
	t.mu.Lock()
	t.inserts = append(t.inserts, d) // append to delta storage
	if t.t.PersistencyMode == Safe {
		t.logfile.Write([]byte("insert "))
		tmp, _ := json.Marshal(d)
		t.logfile.Write(tmp)
		t.logfile.Write([]byte("\n"))
	}
	if t.next != nil {
		// also insert into next storage
		t.next.Insert(d)
	}
	t.mu.Unlock()
	// TODO: before/after insert trigger
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
	t.mu.RLock()
	result := new(storageShard)
	result.t = t.t
	t.next = result
	maxInsertIndex := len(t.inserts)
	// copy-freeze deletions so we don't have to lock anything
	deletions := make(map[uint]struct{})
	for k, v := range t.deletions {
		deletions[k] = v
	}
	t.mu.RUnlock()
	// from now on, we can rebuild with no hurry; inserts and update/deletes on the previous shard will propagate to us, too

	if maxInsertIndex > 0 || len(deletions) > 0 {
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
		result.deletions = make(map[uint]struct{})
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
					if _, ok := deletions[idx]; ok {
						continue
					}
					// scan
					newcol.scan(i, c.GetValue(idx))
					i++
				}
				// scan delta
				for idx := 0; idx < maxInsertIndex; idx++ {
					// check for deletion
					if _, ok := deletions[t.main_count + uint(idx)]; ok {
						continue
					}
					// scan
					newcol.scan(i, t.inserts[idx].Get(col))
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
				if _, ok := deletions[idx]; ok {
					continue
				}
				// build
				newcol.build(i, c.GetValue(idx))
				i++
			}
			// build delta
			for idx, item := range t.inserts {
				// check for deletion
				if _, ok := deletions[t.main_count + uint(idx)]; ok {
					continue
				}
				// build
				newcol.build(i, item.Get(col))
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
		result.main_count = t.main_count
		result.inserts = t.inserts
		result.deletions = deletions
		result.indexes = t.indexes
	}
	return result
}
