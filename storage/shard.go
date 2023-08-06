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
import "strings"
import "reflect"
import "encoding/json"
import "encoding/binary"
import "github.com/google/uuid"

type storageShard struct {
	t *table
	uuid uuid.UUID // uuid.String()
	// main storage
	main_count uint // size of main storage
	columns map[string]ColumnStorage
	// delta storage
	inserts []dataset // items added to storage
	deletions map[uint]struct{} // items removed from main or inserts (based on main_count + i)
	mu sync.Mutex // delta write lock
	next *storageShard
	// indexes
	indexes []*StorageIndex
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
	//fmt.Println("TODO: load "+u.t.Name+" from "+u.uuid.String())
	for _, col := range u.t.Columns {
		// read column from file
		f, err := os.Open(u.t.schema.path + u.uuid.String() + "-" + col.Name)
		if err != nil {
			panic(err) // this is fatal, but we might be more graceful if some data is missing??
		}
		var magicbyte uint8 // type of that column
		binary.Read(f, binary.LittleEndian, &magicbyte)
		columnstorage := reflect.New(storages[magicbyte]).Interface().(ColumnStorage)
		// TODO: remove StorageSCMER typecast as soon as all storage types have a serialization option
		u.main_count = columnstorage.(*StorageSCMER).Deserialize(f) // read; ownership of f goes to Deserialize, so they will free the handle
		u.columns[col.Name] = columnstorage
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

func (t *storageShard) Insert(d dataset) {
	t.mu.Lock()
	t.inserts = append(t.inserts, d) // append to delta storage
	if t.next != nil {
		// also insert into next storage
		t.next.Insert(d)
	}
	t.mu.Unlock()
}

/* TODO: Delete; Delete must also lock next OR translate delete indexes */

// rebuild main storage from main+delta
func (t *storageShard) rebuild() *storageShard {
	// concurrency! when rebuild is run in background, inserts and deletions into and from old delta storage must be duplicated to the ongoing process
	t.mu.Lock()
	result := new(storageShard)
	result.t = t.t
	t.next = result
	maxInsertIndex := len(t.inserts)
	t.mu.Unlock()
	// from now on, we can rebuild with no hurry
	if maxInsertIndex > 0 || len(t.deletions) > 0 {
		result.uuid, _ = uuid.NewRandom() // new uuid, serialize
		// TODO: SetFinalizer to old shard to delete files from disk
		var b strings.Builder
		b.WriteString("rebuilding shard for table ")
		b.WriteString(t.t.Name)
		b.WriteString("(")
		result.columns = make(map[string]ColumnStorage)
		result.deletions = make(map[uint]struct{})
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
					if _, ok := t.deletions[idx]; ok {
						continue
					}
					// scan
					newcol.scan(i, c.getValue(idx))
					i++
				}
				// scan delta
				for idx := 0; idx < maxInsertIndex; idx++ {
					// check for deletion
					if _, ok := t.deletions[t.main_count + uint(idx)]; ok {
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
				if _, ok := t.deletions[idx]; ok {
					continue
				}
				// build
				newcol.build(i, c.getValue(idx))
				i++
			}
			// build delta
			for idx, item := range t.inserts {
				// check for deletion
				if _, ok := t.deletions[t.main_count + uint(idx)]; ok {
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

			// write to disc
			go func() {
				for colname, col := range result.columns {
					f, err := os.Create(result.t.schema.path + result.uuid.String() + "-" + colname)
					if err != nil {
						panic(err)
					}
					// TODO: don't cast; use the interface
					col.(*StorageSCMER).Serialize(f) // col takes ownership of f, so they will defer f.Close() at the right time
				}
			}()
		}
		b.WriteString(") -> ")
		b.WriteString(fmt.Sprint(result.main_count))
		fmt.Println(b.String())
		rebuildIndexes(t, result)
	} else {
		// otherwise: table stays the same
		result.uuid = t.uuid // copy uuid in case nothing changes
		result.columns = t.columns
		result.main_count = t.main_count
		result.inserts = t.inserts
		result.deletions = t.deletions
		result.indexes = t.indexes
	}
	return result
}
