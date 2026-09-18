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
	tbl.OnInit = &onInit

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
	if tbl.onInitComplete {
		t.Fatal("onInitComplete must stay false after a killed OnInit attempt")
	}

	// A later, unrelated call must retry OnInit from scratch rather than
	// replaying the stale panic against an unrelated query/session.
	tbl.awaitCreationInitialization(scm.NewNil())
	if !tbl.onInitComplete {
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
	tbl.OnInit = &onInit

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
