/*
Copyright (C) 2024  Carl-Philip Hänsch

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
import "runtime"
import "runtime/debug"
import "github.com/jtolds/gls"
import "github.com/launix-de/memcp/scm"

// newCachedColumnReader returns a per-goroutine ColumnReader for the given storage.
// For StorageEnum this gives O(1) sequential decode; for others it's a no-op.
// When the storage is OverlayBlob wrapping StorageEnum, unwraps to cache the
// enum directly (blob overlay is only used for large-value storage, not compute inputs).
func newCachedColumnReader(col ColumnStorage) ColumnReader {
	// unwrap OverlayBlob: compute inputs are stored in the base, not the blob layer
	if ob, ok := col.(*OverlayBlob); ok {
		return ob.Base.GetCachedReader()
	}
	return col.GetCachedReader()
}

func (t *table) ComputeColumn(name string, inputCols []string, computor scm.Scmer) {
	for i, c := range t.Columns {
		if c.Name == name {
			// found the column
			t.Columns[i].Computor = computor // set formula so delta storages and rebuild algo know how to recompute
			done := make(chan error, 6)
			shardlist := t.ActiveShards()
			for i, s := range shardlist {
				gls.Go(func(i int, s *storageShard) func() {
					return func() {
						defer func() {
							if r := recover(); r != nil {
								//fmt.Println("panic during compute:", r, string(debug.Stack()))
								done <- scanError{r, string(debug.Stack())}
							}
						}()
						for !s.ComputeColumn(name, inputCols, computor, len(shardlist) == 1) {
							// couldn't compute column because delta is still active
							t.mu.Lock()
							s = s.rebuild(false)
							shardlist[i] = s
							t.mu.Unlock()
							// persist new shard UUID after publishing
							t.schema.save()
						}
						done <- nil
					}
				}(i, s))
			}
			for range shardlist {
				err := <-done // collect finish signal before return
				if err != nil {
					panic(err)
				}
			}
			return
		}
	}
	panic("column " + t.Name + "." + name + " does not exist")
}

func (s *storageShard) ComputeColumn(name string, inputCols []string, computor scm.Scmer, parallel bool) bool {
	fmt.Println("start compute on", s.t.Name, "parallel", parallel)
	if s.deletions.Count() > 0 || len(s.inserts) > 0 {
		return false // can't compute in shards with delta storage
	}
	// We are going to mutate this shard's columns: mark shard as WRITE (not COLD)
	s.srState = WRITE
	// Ensure main_count and input storages are initialized before compute
	s.ensureMainCount(false)
	cols := make([]ColumnStorage, len(inputCols))
	for i, col := range inputCols {
		cols[i] = s.getColumnStorageOrPanic(col)
	}
	vals := make([]scm.Scmer, s.main_count) // build the stretchy value array
	if parallel {
		var done sync.WaitGroup
		done.Add(int(s.main_count))
		progress := make(chan uint32, runtime.NumCPU()/2) // don't go all at once, we don't have enough RAM
		for i := 0; i < runtime.NumCPU()/2; i++ {
			gls.Go(func() { // threadpool with half of the cores
				// allocate private buffers per worker to avoid data races
				colvalues := make([]scm.Scmer, len(cols))
				// per-worker cached readers for O(1) sequential StorageEnum decode
				readers := make([]ColumnReader, len(cols))
				for j, col := range cols {
					readers[j] = newCachedColumnReader(col)
				}
				for i := range progress {
					for j := range readers {
						colvalues[j] = readers[j].GetValue(uint32(i)) // read values from main storage into lambda params
					}
					vals[i] = scm.Apply(computor, colvalues...) // execute computor kernel (but the onoptimized version for non-serial use)
					done.Done()
				}
			})
		}
		// add all items to the queue
		for i := uint32(0); i < s.main_count; i++ {
			progress <- i
		}
		close(progress) // signal workers to exit
		done.Wait()
	} else {
		// allocate a common param buffer to save allocations
		colvalues := make([]scm.Scmer, len(cols))
		fn := scm.OptimizeProcToSerialFunction(computor) // optimize for serial application
		// per-column cached readers for O(1) sequential StorageEnum decode
		readers := make([]ColumnReader, len(cols))
		for j, col := range cols {
			readers[j] = newCachedColumnReader(col)
		}
		for i := uint32(0); i < s.main_count; i++ {
			for j := range readers {
				colvalues[j] = readers[j].GetValue(i) // read values from main storage into lambda params
			}
			vals[i] = fn(colvalues...) // execute computor kernel
		}
	}

	s.mu.Lock() // don't defer because we unlock inbetween
	store := new(StorageSCMER)
	store.values = vals
	s.columns[name] = store
	s.mu.Unlock()
	// TODO: decide whether to rebuild optimized store
	return true
}
