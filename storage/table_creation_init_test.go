// Copyright (C) 2026 MemCP Contributors
// SPDX-License-Identifier: GPL-3.0-or-later
package storage

import (
	"os"
	"testing"

	"github.com/launix-de/memcp/scm"
)

// TestAwaitCreationInitializationRetriesAfterQueryKilled guards against a
// transient per-query cancellation ("query killed", e.g. from a client
// timeout while OnInit happens to be running) getting permanently cached as
// creationPanic. Before the fix, any later caller -- for any query, from any
// session, long after the killed query is gone -- would replay that same
// stale "query killed" panic forever, breaking the table until restart.
func TestAwaitCreationInitializationRetriesAfterQueryKilled(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-table-creation-retry-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tcreationretry")

	CreateDatabase("tcreationretry", false)
	tbl, _ := CreateTable("tcreationretry", "items", Memory, false)

	calls := 0
	onInit := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		calls++
		if calls == 1 {
			panic("query killed")
		}
		return scm.NewBool(true)
	})
	tbl.OnInit.Store(&onInit)

	func() {
		defer func() {
			if r := recover(); r == nil || r != "query killed" {
				t.Fatalf("first call: expected panic %q, got %v", "query killed", r)
			}
		}()
		tbl.awaitCreationInitialization(scm.NewNil())
	}()

	if tbl.creationPanic != nil {
		t.Fatalf("a transient query-killed panic must not be cached as creationPanic, got %v", tbl.creationPanic)
	}
	if tbl.onInitComplete.Load() {
		t.Fatal("onInitComplete must stay false after a killed OnInit attempt")
	}

	// A later, unrelated call must retry OnInit from scratch rather than
	// replaying the stale panic against an unrelated query/session.
	tbl.awaitCreationInitialization(scm.NewNil())
	if !tbl.onInitComplete.Load() {
		t.Fatal("retry after query-killed should have completed OnInit")
	}
	if calls != 2 {
		t.Fatalf("OnInit should have run twice (killed once, then retried), ran %d times", calls)
	}
}

// TestAwaitCreationInitializationCachesGenuineFailure documents the
// intentional counterpart: a deterministic OnInit failure (not a transient
// cancellation) must still be cached and replayed, so a table with a truly
// broken OnInit fails fast on every later caller instead of silently
// re-running (and re-failing) expensive setup work each time.
func TestAwaitCreationInitializationCachesGenuineFailure(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-table-creation-genuine-failure-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tcreationfailure")

	CreateDatabase("tcreationfailure", false)
	tbl, _ := CreateTable("tcreationfailure", "items", Memory, false)

	calls := 0
	onInit := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		calls++
		panic("boom: genuinely broken onInit")
	})
	tbl.OnInit.Store(&onInit)

	for i := 0; i < 2; i++ {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("call %d: expected a panic", i)
				}
			}()
			tbl.awaitCreationInitialization(scm.NewNil())
		}()
	}

	if tbl.creationPanic == nil {
		t.Fatal("a genuine deterministic OnInit failure must be cached as creationPanic")
	}
	if calls != 1 {
		t.Fatalf("OnInit should only run once for a cached genuine failure, ran %d times", calls)
	}
}

func TestCreateTableAmbiguousNameReleasesSchemaOwnership(t *testing.T) {
	for _, ifNotExists := range []bool{false, true} {
		name := "ordinary"
		if ifNotExists {
			name = "if_not_exists"
		}
		t.Run(name, func(t *testing.T) {
			base, _ := createDurabilityTestTable(t, "tcreateambiguous", 0)
			db := base.schema
			first, _ := CreateTable(db.Name, "Foo", Memory, false)
			second, _ := CreateTable(db.Name, "foo", Memory, false)
			create := scm.Globalenv.Vars[scm.Symbol("createtable")]
			var failure any
			func() {
				defer func() { failure = recover() }()
				scm.Apply(create, scm.NewString(db.Name), scm.NewString("FOO"),
					scm.NewSlice(nil), scm.NewSlice([]scm.Scmer{
						scm.NewString("name_ignorecase"), scm.NewBool(true),
					}), scm.NewBool(ifNotExists))
			}()
			// Inspect ownership directly so a regression fails without hanging
			// another DDL operation or the test fixture's cleanup.
			if !db.schemalock.TryLock() {
				db.schemalock.Unlock()
				t.Fatal("ambiguous CREATE retained database schema ownership")
			}
			db.schemalock.Unlock()
			if failure != "ambiguous table name: FOO" {
				t.Fatalf("CREATE failure = %v, want the original ambiguity error", failure)
			}
			if db.tables.Get("Foo") != first || db.tables.Get("foo") != second || db.tables.Get("FOO") != nil {
				t.Fatal("rejected CREATE changed the namespace")
			}
			if !scm.Apply(create, scm.NewString(db.Name), scm.NewString("unrelated"),
				scm.NewSlice(nil), scm.NewSlice(nil)).Bool() {
				t.Fatal("unrelated CREATE did not complete after the rejected declaration")
			}
		})
	}
}
