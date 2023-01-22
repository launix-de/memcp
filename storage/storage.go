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

import "time"
import "fmt"
import "sync"
import "runtime"
import "strings"
import "github.com/launix-de/cpdb/scm"

type ColumnStorage interface {
	getValue(uint) scm.Scmer // read function
	String() string // self-description
	// buildup functions 1) prepare 2) scan, 3) proposeCompression(), if != nil repeat at 1, 4) init, 5) build; all values are passed through twice
	// analyze
	prepare()
	scan(uint, scm.Scmer)
	proposeCompression() ColumnStorage

	// store
	init(uint)
	build(uint, scm.Scmer)
	finish()
}

type dataset map[string]scm.Scmer
type table struct {
	name string
	// main storage
	main_count uint // size of main storage
	columns map[string]ColumnStorage
	// delta storage
	inserts []dataset // items added to storage
	deletions map[uint]struct{} // items removed from main or inserts (based on main_count + i)
	mu sync.Mutex // delta write lock
	// indexes
	indexes []*StorageIndex
}

func (t *table) Insert(d dataset) {
	t.mu.Lock()
	t.inserts = append(t.inserts, d) // append to delta storage
	// check columns for some to add
	for c, _ := range d {
		if _, ok := t.columns[c]; !ok {
			// create column with dummy storage for next rebuild
			t.columns[c] = new(StorageSCMER)
		}
	}
	t.mu.Unlock()
}

// rebuild main storage from main+delta
func (t *table) rebuild() *table {
	// TODO: concurrency! when rebuild is run in background, inserts and deletions into and from old delta storage must be duplicated to the ongoing process
	result := new(table)
	result.name = t.name
	if len(t.inserts) > 0 || len(t.deletions) > 0 {
		var b strings.Builder
		b.WriteString("rebuilding table ")
		b.WriteString(t.name)
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
				for idx, item := range t.inserts {
					// check for deletion
					if _, ok := t.deletions[t.main_count + uint(idx)]; ok {
						continue
					}
					// scan
					newcol.scan(i, item[col])
					i++
				}
				newcol2 := newcol.proposeCompression()
				if newcol2 == nil {
					break // we found the optimal storage format
				} else {
					// redo scan phase with compression
					//fmt.Printf("Compression with %T\n", newcol2)
					newcol = newcol2
				}
			}
			b.WriteString(col) // colname
			b.WriteString(" ")
			b.WriteString(newcol.String()) // storage type (remove *storage.Storage, so it will only say SCMER, Sparse, Int or String)
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
				newcol.build(i, item[col])
				i++
			}
			newcol.finish()
			result.columns[col] = newcol
			result.main_count = i
		}
		b.WriteString(") -> ")
		b.WriteString(fmt.Sprint(result.main_count))
		fmt.Println(b.String())
		rebuildIndexes(t, result)
	} else {
		// otherwise: table stays the same
		result.columns = t.columns
		result.main_count = t.main_count
		result.inserts = t.inserts
		result.deletions = t.deletions
		result.indexes = t.indexes
	}
	return result
}

func (t *table) scan(condition scm.Scmer, callback scm.Scmer) string {
	start := time.Now() // time measurement

	cargs := condition.(scm.Proc).Params.([]scm.Scmer) // list of arguments condition
	margs := callback.(scm.Proc).Params.([]scm.Scmer) // list of arguments map
	cdataset := make([]scm.Scmer, len(cargs))
	mdataset := make([]scm.Scmer, len(margs))

	// TODO: implement a proxy for scripts that routes the scan between nodes first

	// main storage
	ccols := make([]ColumnStorage, len(cargs))
	mcols := make([]ColumnStorage, len(margs))
	for i, k := range cargs { // iterate over columns
		ccols[i] = t.columns[string(k.(scm.Symbol))] // find storage
	}
	for i, k := range margs { // iterate over columns
		mcols[i] = t.columns[string(k.(scm.Symbol))] // find storage
	}
	// iterate over items (indexed)
	for idx := range t.iterateIndex(condition) {
		if _, ok := t.deletions[idx]; ok {
			continue // item is on delete list
		}
		// check condition
		for i, k := range ccols { // iterate over columns
			cdataset[i] = k.getValue(idx)
		}
		if (!scm.ToBool(scm.Apply(condition, cdataset))) {
			continue // condition did not match
		}

		// call map function
		for i, k := range mcols { // iterate over columns
			mdataset[i] = k.getValue(idx)
		}
		scm.Apply(callback, mdataset) // TODO: output monad
	}

	// delta storage (unindexed)
	for idx, item := range t.inserts { // iterate over table
		if _, ok := t.deletions[t.main_count + uint(idx)]; ok {
			continue // item is in delete list
		}
		// prepare&call condition function
		for i, k := range cargs { // iterate over columns
			cdataset[i] = item[string(k.(scm.Symbol))] // fill value
		}
		// check condition
		if (!scm.ToBool(scm.Apply(condition, cdataset))) {
			continue // condition did not match
		}

		// prepare&call map function
		for i, k := range margs { // iterate over columns
			mdataset[i] = item[string(k.(scm.Symbol))] // fill value
		}
		scm.Apply(callback, mdataset) // TODO: output monad
	}
	return fmt.Sprint(time.Since(start))
}

var tables map[string]*table = make(map[string]*table)

func Init(en scm.Env) {
	en.Vars["scan"] = func (a ...scm.Scmer) scm.Scmer {
		// params: table, condition, map, reduce, reduceSeed
		t := tables[a[0].(string)]
		return t.scan(a[1], a[2])
	}
	en.Vars["stat"] = func (a ...scm.Scmer) scm.Scmer {
		return PrintMemUsage()
	}
	en.Vars["rebuild"] = func (a ...scm.Scmer) scm.Scmer {
		start := time.Now()

		for k, t := range tables {
			// todo: parallel??
			tables[k] = t.rebuild()
		}

		return fmt.Sprint(time.Since(start))
	}
	en.Vars["loadJSON"] = func (a ...scm.Scmer) scm.Scmer {
		start := time.Now()

		LoadJSON(a[0].(string))

		return fmt.Sprint(time.Since(start))
	}
}

func PrintMemUsage() string {
	runtime.GC()
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        // For info on each, see: https://golang.org/pkg/runtime/#MemStats
	var b strings.Builder
        b.WriteString(fmt.Sprintf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\tNUMA nodes = %v", bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys), m.NumGC))
	return b.String()
}

func bToMb(b uint64) uint64 {
    return b / 1024 / 1024
}
