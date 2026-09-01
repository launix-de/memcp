/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
import "github.com/launix-de/memcp/scm"

func blobRefStripe(hash string) int {
	value := byte(0)
	for i := 0; i < len(hash) && i < 2; i++ {
		value <<= 4
		switch c := hash[i]; {
		case c >= '0' && c <= '9':
			value |= c - '0'
		case c >= 'a' && c <= 'f':
			value |= c - 'a' + 10
		case c >= 'A' && c <= 'F':
			value |= c - 'A' + 10
		default:
			value ^= c
		}
	}
	return int(value) & 63
}

func (db *database) lockBlobRef(hash string) func() {
	lock := &db.blobRefState().locks[blobRefStripe(hash)].mu
	lock.Lock()
	return lock.Unlock
}

func blobTableHasColumns(t *table) bool {
	hash, refcount := false, false
	for _, col := range t.Columns {
		hash = hash || col.Name == "hash"
		refcount = refcount || col.Name == "refcount"
	}
	return hash && refcount
}

// ensureBlobTable lazily creates the `.blobs` table inside this database.
func (db *database) ensureBlobTable() *table {
	state := db.blobRefState()
	t := state.table.Load()
	if t != nil && blobTableHasColumns(t) {
		return t
	}

	state.tableMu.Lock()
	defer state.tableMu.Unlock()
	if t = state.table.Load(); t != nil && blobTableHasColumns(t) {
		return t
	}
	db.ensureLoaded()
	if t = db.tables.Get(".blobs"); t != nil {
		// Repair schemas created by older versions that published the table
		// before both columns had been added.
		t.CreateColumn("hash", "TEXT", nil, nil)
		t.CreateColumn("refcount", "INT", nil, nil)
		state.table.Store(t)
		return t
	}

	fmt.Println("creating table", db.Name+".\".blobs\"")
	db.schemalock.Lock()
	if t = db.tables.Get(".blobs"); t != nil {
		db.schemalock.Unlock()
		t.CreateColumn("hash", "TEXT", nil, nil)
		t.CreateColumn("refcount", "INT", nil, nil)
		state.table.Store(t)
		return t
	}
	t = db.newTable(".blobs", Safe)
	t.createColumnLocked("hash", "TEXT", nil, nil)
	t.createColumnLocked("refcount", "INT", nil, nil)
	db.tables.Set(t)
	db.saveLockedAndUnlock(schemaSaveFsync)
	state.table.Store(t)
	registerCreatedTable(t)
	return t
}

// blobCondition builds a Scheme lambda (lambda (hash) (equal?? hash val))
// that the boundary analyzer can introspect for index hints.
func blobCondition(hashVal scm.Scmer) scm.Scmer {
	return scm.NewProcStruct(scm.Proc{
		Params:  scm.NewSlice([]scm.Scmer{scm.NewSymbol("hash")}),
		Body:    scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewNthLocalVar(0), hashVal}),
		En:      &scm.Globalenv,
		NumVars: 1,
	})
}

// sumProc builds (lambda (a b) (+ a b)) for aggregation.
func sumProc() scm.Scmer {
	return scm.NewProcStruct(scm.Proc{
		Params:  scm.NewSlice([]scm.Scmer{scm.NewSymbol("a"), scm.NewSymbol("b")}),
		Body:    scm.NewSlice([]scm.Scmer{scm.NewSymbol("+"), scm.NewNthLocalVar(0), scm.NewNthLocalVar(1)}),
		En:      &scm.Globalenv,
		NumVars: 2,
	})
}

// IncrBlobRefcount increments the reference count for a blob hash in db.`.blobs`.
// If no row exists yet, it inserts one with refcount=1.
// Same-hash operations are serialized across the scan and insert/update;
// independent hashes retain the concurrency of separate lock stripes.
func (db *database) IncrBlobRefcount(hash string) {
	defer db.lockBlobRef(hash)()
	state := db.blobRefState()
	t := db.ensureBlobTable()
	if t == nil {
		return
	}

	hashVal := scm.NewString(hash)

	// callback: (lambda (refcount $update) (if ($update (list "refcount" (+ refcount 1))) 1 0))
	callback := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("refcount"), scm.NewSymbol("$update")}),
		Body: scm.NewSlice([]scm.Scmer{scm.NewSymbol("if"),
			scm.NewSlice([]scm.Scmer{
				scm.NewNthLocalVar(1),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"),
					scm.NewString("refcount"),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("+"), scm.NewNthLocalVar(0), scm.NewInt(1)}),
				}),
			}),
			scm.NewInt(1),
			scm.NewInt(0),
		}),
		En:      &scm.Globalenv,
		NumVars: 2,
	})

	aggr := sumProc()
	incrementExisting := func() bool {
		result := t.scan(
			nil,
			[]string{"hash"}, blobCondition(hashVal),
			[]string{"refcount", "$update"}, callback,
			aggr, scm.NewInt(0), aggr, false,
		)
		return scm.ToInt(result) > 0
	}
	state.rows.RLock()
	found := incrementExisting()
	state.rows.RUnlock()
	if found {
		return
	}

	state.rows.Lock()
	defer state.rows.Unlock()
	if incrementExisting() {
		return
	}
	t.Insert(
		[]string{"hash", "refcount"},
		[][]scm.Scmer{{hashVal, scm.NewInt(1)}},
		nil, scm.NewNil(), false, nil,
	)
}

// DecrBlobRefcount decrements the reference count for a blob hash in db.`.blobs`.
// If the count reaches 0, the row is deleted and the blob file is removed.
func (db *database) DecrBlobRefcount(hash string) {
	defer db.lockBlobRef(hash)()
	state := db.blobRefState()
	state.rows.Lock()
	defer state.rows.Unlock()
	t := db.ensureBlobTable()
	if t == nil {
		return
	}

	hashVal := scm.NewString(hash)

	// callback: (lambda (refcount $update)
	//   (if (<= refcount 1)
	//     (if ($update) 1 0)
	//     (if ($update (list "refcount" (- refcount 1))) 0 0)))
	callback := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("refcount"), scm.NewSymbol("$update")}),
		Body: scm.NewSlice([]scm.Scmer{scm.NewSymbol("if"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("<="), scm.NewNthLocalVar(0), scm.NewInt(1)}),
			// then: delete row, return 1
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("if"),
				scm.NewSlice([]scm.Scmer{scm.NewNthLocalVar(1)}),
				scm.NewInt(1), scm.NewInt(0),
			}),
			// else: decrement, return 0
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("if"),
				scm.NewSlice([]scm.Scmer{
					scm.NewNthLocalVar(1),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"),
						scm.NewString("refcount"),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("-"), scm.NewNthLocalVar(0), scm.NewInt(1)}),
					}),
				}),
				scm.NewInt(0), scm.NewInt(0),
			}),
		}),
		En:      &scm.Globalenv,
		NumVars: 2,
	})

	aggr := sumProc()
	result := t.scan(
		nil,
		[]string{"hash"}, blobCondition(hashVal),
		[]string{"refcount", "$update"}, callback,
		aggr, scm.NewInt(0), aggr, false,
	)

	// If row was deleted (RC was <=1), remove the blob file
	if scm.ToInt(result) > 0 && db.persistence != nil {
		db.persistence.DeleteBlob(hash)
	}
}
