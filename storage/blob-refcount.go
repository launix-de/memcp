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

// ensureBlobTable lazily creates the `.blobs` table inside this database.
func (db *database) ensureBlobTable() *table {
	t := db.GetTable(".blobs")
	if t != nil {
		return t
	}
	fmt.Println("creating table", db.Name+".\".blobs\"")
	t, _ = CreateTable(db.Name, ".blobs", Safe, true)
	if t != nil {
		t.CreateColumn("hash", "TEXT", nil, nil)
		t.CreateColumn("refcount", "INT", nil, nil)
		db.save()
	}
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
func (db *database) IncrBlobRefcount(hash string) {
	db.blobMu.Lock()
	defer db.blobMu.Unlock()
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
	result := t.scan(
		[]string{"hash"}, blobCondition(hashVal),
		[]string{"refcount", "$update"}, callback,
		aggr, scm.NewInt(0), aggr, false,
	)

	if scm.ToInt(result) == 0 {
		t.Insert(
			[]string{"hash", "refcount"},
			[][]scm.Scmer{{hashVal, scm.NewInt(1)}},
			nil, scm.NewNil(), false, nil,
		)
	}
}

// DecrBlobRefcount decrements the reference count for a blob hash in db.`.blobs`.
// If the count reaches 0, the row is deleted and the blob file is removed.
func (db *database) DecrBlobRefcount(hash string) {
	db.blobMu.Lock()
	defer db.blobMu.Unlock()
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
		[]string{"hash"}, blobCondition(hashVal),
		[]string{"refcount", "$update"}, callback,
		aggr, scm.NewInt(0), aggr, false,
	)

	// If row was deleted (RC was <=1), remove the blob file
	if scm.ToInt(result) > 0 && db.persistence != nil {
		db.persistence.DeleteBlob(hash)
	}
}
