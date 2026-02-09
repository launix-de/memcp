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

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/launix-de/memcp/scm"
)

// countBlobFiles counts blob files under db's blob/ directory.
func countBlobFiles(t *testing.T, dbName string) int {
	t.Helper()
	blobDir := filepath.Join(Basepath, dbName, "blob")
	count := 0
	filepath.Walk(blobDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

// queryBlobsTable scans the .blobs table and returns a map of hash -> refcount.
func queryBlobsTable(t *testing.T, db *database) map[string]int {
	t.Helper()
	bt := db.GetTable(".blobs")
	if bt == nil {
		return nil
	}
	result := make(map[string]int)
	bt.scan(
		[]string{},
		scm.NewProcStruct(scm.Proc{
			Params:  scm.NewSlice([]scm.Scmer{}),
			Body:    scm.NewBool(true),
			En:      &scm.Globalenv,
			NumVars: 0,
		}),
		[]string{"hash", "refcount"},
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			h := scm.String(a[0])
			rc := int(scm.ToInt(a[1]))
			result[h] = rc
			return scm.NewNil()
		}),
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] }),
		scm.NewNil(),
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] }),
		false,
	)
	return result
}

// scanCondition builds (lambda (col) (equal?? col val)) for table scans.
func scanCondition(colName string, val scm.Scmer) scm.Scmer {
	return scm.NewProcStruct(scm.Proc{
		Params:  scm.NewSlice([]scm.Scmer{scm.NewSymbol(colName)}),
		Body:    scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewNthLocalVar(0), val}),
		En:      &scm.Globalenv,
		NumVars: 1,
	})
}

// trueCondition builds (lambda () true) for unconditional scans.
func trueCondition() scm.Scmer {
	return scm.NewProcStruct(scm.Proc{
		Params:  scm.NewSlice([]scm.Scmer{}),
		Body:    scm.NewBool(true),
		En:      &scm.Globalenv,
		NumVars: 0,
	})
}

func TestBlobInsertRebuildAndRead(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-blob-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tdb1")

	CreateDatabase("tdb1", false)
	tbl, _ := CreateTable("tdb1", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)

	db := GetDatabase("tdb1")

	// Need >2 long strings to trigger OverlayBlob compression
	longA := strings.Repeat("X", 1000)
	longB := strings.Repeat("Y", 500)
	rows := [][]scm.Scmer{
		{scm.NewInt(1), scm.NewString(longA)},
		{scm.NewInt(2), scm.NewString(longA)}, // duplicate of row 1
		{scm.NewInt(3), scm.NewString(longB)},
		{scm.NewInt(4), scm.NewString(longB)}, // duplicate of row 3
	}
	tbl.Insert([]string{"id", "content"}, rows, nil, scm.NewNil(), false, nil)

	// First rebuild: persists blobs
	Rebuild(true, true)

	// 2 unique blob files (dedup)
	blobDir := filepath.Join(Basepath, "tdb1", "blob")
	filepath.Walk(blobDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			t.Logf("blob file: %s (size %d)", path, info.Size())
		}
		return nil
	})
	if n := countBlobFiles(t, "tdb1"); n != 2 {
		t.Fatalf("after first rebuild: expected 2 blob files, got %d", n)
	}

	// .blobs table should have 2 entries, each refcount=1
	refs := queryBlobsTable(t, db)
	if refs == nil {
		t.Fatal(".blobs table not found after rebuild")
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 entries in .blobs, got %d", len(refs))
	}
	for hash, rc := range refs {
		if rc != 1 {
			t.Errorf("expected refcount 1 for hash %s, got %d", hash, rc)
		}
	}

	// Second rebuild: blobs should survive
	Rebuild(true, true)

	if n := countBlobFiles(t, "tdb1"); n != 2 {
		t.Fatalf("after second rebuild: expected 2 blob files, got %d", n)
	}

	// Verify data still readable after two rebuilds
	for _, tc := range []struct {
		id  int64
		len int
	}{{1, 1000}, {2, 1000}, {3, 500}, {4, 500}} {
		var readLen int
		tbl.scan(
			[]string{"id"}, scanCondition("id", scm.NewInt(tc.id)),
			[]string{"content"},
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
				readLen = len(scm.String(a[0]))
				return scm.NewNil()
			}),
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] }),
			scm.NewNil(),
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] }),
			false,
		)
		if readLen != tc.len {
			t.Errorf("row id=%d: expected content length %d, got %d", tc.id, tc.len, readLen)
		}
	}
}

func TestBlobDeleteRowsAndRebuild(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-blob-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tdb4")

	CreateDatabase("tdb4", false)
	tbl, _ := CreateTable("tdb4", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)

	db := GetDatabase("tdb4")

	// 5 rows: 3 unique long strings, with enough longStrings (>2) to keep
	// OverlayBlob after deletion. Rows 1,4,5 share longA.
	longA := strings.Repeat("A", 1000)
	longB := strings.Repeat("B", 1000)
	longC := strings.Repeat("C", 1000)
	rows := [][]scm.Scmer{
		{scm.NewInt(1), scm.NewString(longA)},
		{scm.NewInt(2), scm.NewString(longB)},
		{scm.NewInt(3), scm.NewString(longC)},
		{scm.NewInt(4), scm.NewString(longA)},
		{scm.NewInt(5), scm.NewString(longA)},
	}
	tbl.Insert([]string{"id", "content"}, rows, nil, scm.NewNil(), false, nil)
	Rebuild(true, true)

	if n := countBlobFiles(t, "tdb4"); n != 3 {
		t.Fatalf("after insert+rebuild: expected 3 blob files, got %d", n)
	}
	refs := queryBlobsTable(t, db)
	if len(refs) != 3 {
		t.Fatalf("after insert+rebuild: expected 3 .blobs entries, got %d", len(refs))
	}

	// Delete rows 2 and 3 (the only rows referencing longB and longC).
	// Rows 1, 4, 5 (longA) remain — 3 longStrings keeps OverlayBlob.
	for _, id := range []int64{2, 3} {
		tbl.scan(
			[]string{"id"}, scanCondition("id", scm.NewInt(id)),
			[]string{"$update"},
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
				scm.Apply(a[0]) // delete
				return scm.NewNil()
			}),
			scm.NewNil(), scm.NewNil(), scm.NewNil(), false,
		)
	}

	// Rebuild: old shard (3 blobs) replaced by new shard (1 blob: longA)
	Rebuild(true, true)

	// Only blob A should remain
	if n := countBlobFiles(t, "tdb4"); n != 1 {
		t.Fatalf("after delete+rebuild: expected 1 blob file, got %d", n)
	}

	refs2 := queryBlobsTable(t, db)
	activeCount := 0
	for hash, rc := range refs2 {
		if rc > 0 {
			activeCount++
			if rc != 1 {
				t.Errorf("expected refcount 1 for remaining hash %s, got %d", hash, rc)
			}
		}
	}
	if activeCount != 1 {
		t.Fatalf("expected 1 active blob ref after delete, got %d", activeCount)
	}

	// Remaining data still readable
	for _, id := range []int64{1, 4, 5} {
		var readLen int
		tbl.scan(
			[]string{"id"}, scanCondition("id", scm.NewInt(id)),
			[]string{"content"},
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
				readLen = len(scm.String(a[0]))
				return scm.NewNil()
			}),
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] }),
			scm.NewNil(),
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] }),
			false,
		)
		if readLen != 1000 {
			t.Fatalf("row id=%d after delete+rebuild: expected content length 1000, got %d", id, readLen)
		}
	}
}

func TestBlobDropTableReleasesBlobs(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-blob-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tdb2")

	CreateDatabase("tdb2", false)
	tbl, _ := CreateTable("tdb2", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)

	db := GetDatabase("tdb2")

	// 3 different long strings to trigger OverlayBlob
	rows := [][]scm.Scmer{
		{scm.NewInt(1), scm.NewString(strings.Repeat("D", 1000))},
		{scm.NewInt(2), scm.NewString(strings.Repeat("E", 1000))},
		{scm.NewInt(3), scm.NewString(strings.Repeat("F", 1000))},
	}
	tbl.Insert([]string{"id", "content"}, rows, nil, scm.NewNil(), false, nil)
	Rebuild(true, true)

	if n := countBlobFiles(t, "tdb2"); n != 3 {
		t.Fatalf("expected 3 blob files before drop, got %d", n)
	}

	refs := queryBlobsTable(t, db)
	if len(refs) != 3 {
		t.Fatalf("expected 3 blob refs before drop, got %d", len(refs))
	}

	// Drop table should release blob refcounts and delete blob files
	DropTable("tdb2", "docs", false)

	if n := countBlobFiles(t, "tdb2"); n != 0 {
		t.Fatalf("expected 0 blob files after drop, got %d", n)
	}

	refsAfter := queryBlobsTable(t, db)
	for hash, rc := range refsAfter {
		if rc > 0 {
			t.Errorf("expected refcount 0 after drop for hash %s, got %d", hash, rc)
		}
	}
}

func TestBlobSharedAcrossTables(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-blob-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tdb3")

	CreateDatabase("tdb3", false)
	tbl1, _ := CreateTable("tdb3", "t1", Safe, false)
	tbl1.CreateColumn("id", "INT", nil, nil)
	tbl1.CreateColumn("content", "TEXT", nil, nil)

	tbl2, _ := CreateTable("tdb3", "t2", Safe, false)
	tbl2.CreateColumn("id", "INT", nil, nil)
	tbl2.CreateColumn("content", "TEXT", nil, nil)

	db := GetDatabase("tdb3")

	// Same long strings in both tables
	shared := strings.Repeat("S", 1000)
	for _, tbl := range []*table{tbl1, tbl2} {
		rows := [][]scm.Scmer{
			{scm.NewInt(1), scm.NewString(shared)},
			{scm.NewInt(2), scm.NewString(strings.Repeat("U", 800))},
			{scm.NewInt(3), scm.NewString(strings.Repeat("V", 600))},
		}
		tbl.Insert([]string{"id", "content"}, rows, nil, scm.NewNil(), false, nil)
	}
	Rebuild(true, true)

	// 3 unique blobs, each referenced by 2 tables -> refcount 2
	refs := queryBlobsTable(t, db)
	if len(refs) != 3 {
		t.Fatalf("expected 3 entries in .blobs, got %d", len(refs))
	}
	for hash, rc := range refs {
		if rc != 2 {
			t.Errorf("expected refcount 2 for shared hash %s, got %d", hash, rc)
		}
	}

	// Drop first table: refcount should decrease to 1
	DropTable("tdb3", "t1", false)

	refs2 := queryBlobsTable(t, db)
	for hash, rc := range refs2 {
		if rc != 1 {
			t.Errorf("after dropping t1: expected refcount 1 for hash %s, got %d", hash, rc)
		}
	}

	// Blobs should still exist on disk
	if n := countBlobFiles(t, "tdb3"); n != 3 {
		t.Fatalf("after dropping t1: expected 3 blob files still, got %d", n)
	}

	// Data in t2 still readable
	var readLen int
	tbl2.scan(
		[]string{"id"}, scanCondition("id", scm.NewInt(1)),
		[]string{"content"},
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			readLen = len(scm.String(a[0]))
			return scm.NewNil()
		}),
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] }),
		scm.NewNil(),
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] }),
		false,
	)
	if readLen != 1000 {
		t.Fatalf("t2 content after t1 drop: expected 1000, got %d", readLen)
	}

	// Drop second table: blobs should be deleted
	DropTable("tdb3", "t2", false)

	if n := countBlobFiles(t, "tdb3"); n != 0 {
		t.Fatalf("after dropping both tables: expected 0 blob files, got %d", n)
	}
	fmt.Println("TestBlobSharedAcrossTables passed")
}

// TestDoubleRebuildPreservesShardFiles is a regression test for a data-loss
// bug where the deferred shard cleanup in database.rebuild() deleted column
// files after a no-change rebuild. When a shard has no inserts/deletes, its
// rebuild() returns a new struct that copies the OLD UUID (shard.go:1047).
// Without UUID comparison, the old shard was collected for cleanup and its
// column files removed -- even though the new shard still references them.
func TestDoubleRebuildPreservesShardFiles(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-rebuild-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tdb5")

	CreateDatabase("tdb5", false)
	tbl, _ := CreateTable("tdb5", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("name", "TEXT", nil, nil)

	db := GetDatabase("tdb5")

	rows := [][]scm.Scmer{
		{scm.NewInt(1), scm.NewString("alpha")},
		{scm.NewInt(2), scm.NewString("beta")},
		{scm.NewInt(3), scm.NewString("gamma")},
	}
	tbl.Insert([]string{"id", "name"}, rows, nil, scm.NewNil(), false, nil)

	// First rebuild: persists column files to disk
	Rebuild(true, true)

	// Verify column files exist after first rebuild
	tbl = db.GetTable("items")
	if tbl == nil {
		t.Fatal("table items not found after first rebuild")
	}
	shardUUID := ""
	if len(tbl.Shards) > 0 && tbl.Shards[0] != nil {
		shardUUID = tbl.Shards[0].uuid.String()
	} else if len(tbl.PShards) > 0 && tbl.PShards[0] != nil {
		shardUUID = tbl.PShards[0].uuid.String()
	}
	if shardUUID == "" {
		t.Fatal("no shard UUID found after first rebuild")
	}
	t.Logf("shard UUID after first rebuild: %s", shardUUID)

	dbDir := filepath.Join(Basepath, "tdb5")
	for _, col := range []string{"id", "name"} {
		colFile := filepath.Join(dbDir, shardUUID+"-"+col)
		if _, err := os.Stat(colFile); os.IsNotExist(err) {
			t.Fatalf("column file %s missing after first rebuild", colFile)
		}
	}

	// Second rebuild with all=false: no changes → shard rebuild copies the same UUID.
	// This is what UnloadDatabases() calls during shutdown.
	Rebuild(false, false)

	// The UUID should remain the same (no changes → no new UUID)
	tbl = db.GetTable("items")
	shardUUID2 := ""
	if len(tbl.Shards) > 0 && tbl.Shards[0] != nil {
		shardUUID2 = tbl.Shards[0].uuid.String()
	} else if len(tbl.PShards) > 0 && tbl.PShards[0] != nil {
		shardUUID2 = tbl.PShards[0].uuid.String()
	}
	t.Logf("shard UUID after second rebuild: %s", shardUUID2)

	if shardUUID != shardUUID2 {
		t.Fatalf("shard UUID changed from %s to %s on no-change rebuild", shardUUID, shardUUID2)
	}

	// Column files must still exist (the bug deleted them here)
	for _, col := range []string{"id", "name"} {
		colFile := filepath.Join(dbDir, shardUUID+"-"+col)
		if _, err := os.Stat(colFile); os.IsNotExist(err) {
			t.Fatalf("REGRESSION: column file %s deleted after second rebuild (no-change shard)", colFile)
		}
	}

	// Data must still be readable
	for _, tc := range []struct {
		id   int64
		name string
	}{{1, "alpha"}, {2, "beta"}, {3, "gamma"}} {
		var readName string
		tbl.scan(
			[]string{"id"}, scanCondition("id", scm.NewInt(tc.id)),
			[]string{"name"},
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
				readName = scm.String(a[0])
				return scm.NewNil()
			}),
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] }),
			scm.NewNil(),
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] }),
			false,
		)
		if readName != tc.name {
			t.Errorf("row id=%d: expected name %q, got %q", tc.id, tc.name, readName)
		}
	}
}
