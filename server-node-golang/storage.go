package main

import "os"
import "bufio"
import "time"
import "fmt"
import "runtime"
import "encoding/json"

type ColumnStorage interface {
	getValue(uint) scmer // read function
	// buildup functions 1) prepare 2) scan, 3) proposeCompression(), if != nil repeat at 1, 4) init, 5) build; all values are passed through twice
	// analyze
	prepare()
	scan(uint, scmer)
	proposeCompression() ColumnStorage

	// store
	init(uint)
	build(uint, scmer)
	finish()
}

// todo: enhance table datatype
type dataset map[string]scmer
type table struct {
	name string
	// main storage
	main_count uint // size of main storage
	columns map[string]ColumnStorage
	// delta storage
	inserts []dataset // items added to storage
	deletions map[uint]struct{} // items removed from main or inserts (based on main_count + i)
	// indexes
	indexes []StorageIndex
}

func (t *table) insert(d dataset) {
	t.inserts = append(t.inserts, d) // append to delta storage
	// check columns for some to add
	for c, _ := range d {
		if _, ok := t.columns[c]; !ok {
			// create column with dummy storage for next rebuild
			t.columns[c] = new(StorageSCMER)
		}
	}
}

// rebuild main storage from main+delta
func (t *table) rebuild() *table {
	result := new(table)
	result.name = t.name
	if len(t.inserts) > 0 || len(t.deletions) > 0 {
		fmt.Println("rebuilding table", t.name)
		result.columns = make(map[string]ColumnStorage)
		result.deletions = make(map[uint]struct{})
		// copy column data in two phases: scan, build (if delta is non-empty)
		for col, c := range t.columns {
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
	} else {
		// otherwise: table stays the same
		result.columns = t.columns
		result.main_count = t.main_count
		result.inserts = t.inserts
		result.deletions = t.deletions
	}
	return result
}

func (t *table) scan(condition scmer, callback scmer) string {
	start := time.Now() // time measurement

	cargs := condition.(proc).params.([]scmer) // list of arguments condition
	margs := callback.(proc).params.([]scmer) // list of arguments map
	cdataset := make([]scmer, len(cargs))
	mdataset := make([]scmer, len(margs))

	// TODO: analyze condition and find indexes

	// main storage
	ccols := make([]ColumnStorage, len(cargs))
	mcols := make([]ColumnStorage, len(margs))
	for i, k := range cargs { // iterate over columns
		ccols[i] = t.columns[string(k.(symbol))] // find storage
	}
	for i, k := range margs { // iterate over columns
		mcols[i] = t.columns[string(k.(symbol))] // find storage
	}
	// iterate over items
	for idx := uint(0); idx < t.main_count; idx++ {
		if _, ok := t.deletions[idx]; ok {
			continue // item is on delete list
		}
		// check condition
		for i, k := range ccols { // iterate over columns
			cdataset[i] = k.getValue(idx)
		}
		if (!toBool(apply(condition, cdataset))) {
			continue // condition did not match
		}

		// call map function
		for i, k := range mcols { // iterate over columns
			mdataset[i] = k.getValue(idx)
		}
		apply(callback, mdataset) // TODO: output monad
	}

	// delta storage
	for idx, item := range t.inserts { // iterate over table
		if _, ok := t.deletions[t.main_count + uint(idx)]; ok {
			continue // item is in delete list
		}
		// prepare&call condition function
		for i, k := range cargs { // iterate over columns
			cdataset[i] = item[string(k.(symbol))] // fill value
		}
		// check condition
		if (!toBool(apply(condition, cdataset))) {
			continue // condition did not match
		}

		// prepare&call map function
		for i, k := range margs { // iterate over columns
			mdataset[i] = item[string(k.(symbol))] // fill value
		}
		apply(callback, mdataset) // TODO: output monad
	}
	return fmt.Sprint(time.Since(start))
}

var tables map[string]*table = make(map[string]*table)

func loadStorageFrom(filename string) {
	f, _ := os.Open(filename)
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	var t *table
	for scanner.Scan() {
		s := scanner.Text()
		if s == "" {
			// ignore
		} else if s[0:7] == "#table " {
			var ok bool
			t, ok = tables[s[7:]]
			if !ok {
				// new table
				t = new(table)
				t.name = s[7:]
				tables[t.name] = t
				t.columns = make(map[string]ColumnStorage)
				t.deletions = make(map[uint]struct{})
			}
		} else if s[0] == '#' {
			// comment
		} else {
			if t == nil {
				panic("no table set")
			} else {
				var x dataset
				json.Unmarshal([]byte(s), &x) // parse JSON
				t.insert(x) // put into table
			}
		}
	}
}

func initStorageEngine(en env) {
	// example: (scan "PLZ" (lambda () 1) (lambda (PLZ Ort) (print PLZ " - " Ort)))
	// example: (scan "PLZ" (lambda (Ort) (equal? Ort "Neugersdorf")) (lambda (PLZ Ort) (print PLZ " - " Ort)))
	// example: (scan "PLZ" (lambda (Ort) (equal? Ort "Dresden")) (lambda (PLZ Ort) (print PLZ " - " Ort)))
	// example: (scan "manufacturer" (lambda () 1) (lambda (ID) (print ID)))
	// example: (scan "referrer" (lambda () 1) (lambda (ID) (print ID)))
	en.vars["scan"] = func (a ...scmer) scmer {
		// params: table, condition, map, reduce, reduceSeed
		t := tables[a[0].(string)]
		return t.scan(a[1], a[2])
	}
	en.vars["stat"] = func (a ...scmer) scmer {
		return PrintMemUsage()
	}
	en.vars["rebuild"] = func (a ...scmer) scmer {
		start := time.Now()

		for k, t := range tables {
			// todo: parallel??
			tables[k] = t.rebuild()
		}

		return fmt.Sprint(time.Since(start))
	}
}

func PrintMemUsage() string {
	runtime.GC()
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        // For info on each, see: https://golang.org/pkg/runtime/#MemStats
        return fmt.Sprintf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v", bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys), m.NumGC)
}

func bToMb(b uint64) uint64 {
    return b / 1024 / 1024
}
