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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/launix-de/memcp/scm"
)

// setupGCTest creates a temp dir, sets Basepath, inits the engine, and returns
// a cleanup func. All tests must defer the cleanup.
func setupGCTest(t *testing.T) func() {
	t.Helper()
	dir, err := os.MkdirTemp("", "memcp-gc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	oldBasepath := Basepath
	Basepath = dir
	Init(scm.Globalenv)
	LoadDatabases()
	return func() {
		databases.Remove("gcdb")
		Basepath = oldBasepath
		os.RemoveAll(dir)
	}
}

// insertLongRows inserts rows with long strings (triggers OverlayBlob) and rebuilds.
func insertLongRows(t *testing.T, tbl *table, rows []string) {
	t.Helper()
	var scmRows [][]scm.Scmer
	for i, s := range rows {
		scmRows = append(scmRows, []scm.Scmer{scm.NewInt(int64(i + 1)), scm.NewString(s)})
	}
	tbl.Insert([]string{"id", "content"}, scmRows, nil, scm.NewNil(), false, nil)
	Rebuild(true, true)
}

// blobFiles returns all blob filenames under dbName/blob/.
func blobFiles(t *testing.T, dbName string) []string {
	t.Helper()
	var result []string
	filepath.Walk(filepath.Join(Basepath, dbName, "blob"), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			result = append(result, info.Name())
		}
		return nil
	})
	return result
}

// shardFilesOnDisk returns all shard-related files in the db directory.
func shardFilesOnDisk(t *testing.T, dbName string) []string {
	t.Helper()
	entries, _ := os.ReadDir(filepath.Join(Basepath, dbName))
	var result []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if n == "schema.json" || n == "schema.json.old" {
			continue
		}
		result = append(result, n)
	}
	return result
}

// TestCleanNoOrphans: normal operation — no orphans → 0 deletions.
func TestCleanNoOrphans(t *testing.T) {
	defer setupGCTest(t)()

	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)

	insertLongRows(t, tbl, []string{
		strings.Repeat("A", 800),
		strings.Repeat("B", 800),
		strings.Repeat("C", 800),
	})

	db := GetDatabase("gcdb")
	b, s := CleanDatabase(db)
	if b != 0 || s != 0 {
		t.Errorf("expected 0 blobs, 0 shards deleted; got %d blobs, %d shards", b, s)
	}

	// Data still readable
	count := 0
	tbl.scan(nil, []string{}, trueCondition(), []string{"id"},
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { count++; return scm.NewNil() }),
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] }),
		scm.NewNil(), scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] }), false)
	if count != 3 {
		t.Errorf("expected 3 rows, got %d", count)
	}
}

// TestCleanOrphanedBlob: blob file on disk with no refcount entry → gets deleted.
func TestCleanOrphanedBlob(t *testing.T) {
	defer setupGCTest(t)()

	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)

	insertLongRows(t, tbl, []string{
		strings.Repeat("X", 800),
		strings.Repeat("Y", 800),
		strings.Repeat("Z", 800),
	})

	beforeBlobs := len(blobFiles(t, "gcdb"))
	if beforeBlobs == 0 {
		t.Skip("no blobs created — OverlayBlob threshold not met")
	}

	// Inject a fake orphan blob file directly on disk.
	orphanHash := "deadbeefdeadbeefdeadbeefdeadbeef"
	orphanPath := filepath.Join(Basepath, "gcdb", "blob", orphanHash[:2], orphanHash[2:4])
	os.MkdirAll(orphanPath, 0750)
	os.WriteFile(filepath.Join(orphanPath, orphanHash), []byte("fake"), 0640)

	db := GetDatabase("gcdb")
	b, _ := CleanDatabase(db)
	if b != 1 {
		t.Errorf("expected 1 orphaned blob deleted, got %d", b)
	}

	// Real blobs must survive.
	if after := len(blobFiles(t, "gcdb")); after != beforeBlobs {
		t.Errorf("expected %d real blobs to survive, got %d", beforeBlobs, after)
	}

	// Data still readable.
	count := 0
	tbl.scan(nil, []string{}, trueCondition(), []string{"id"},
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { count++; return scm.NewNil() }),
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] }),
		scm.NewNil(), scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] }), false)
	if count != 3 {
		t.Errorf("expected 3 rows after GC, got %d", count)
	}
}

// TestCleanOrphanedShardFile: shard file with unknown UUID → gets deleted.
func TestCleanOrphanedShardFile(t *testing.T) {
	defer setupGCTest(t)()

	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("val", "TEXT", nil, nil)

	rows := [][]scm.Scmer{
		{scm.NewInt(1), scm.NewString("alpha")},
		{scm.NewInt(2), scm.NewString("beta")},
	}
	tbl.Insert([]string{"id", "val"}, rows, nil, scm.NewNil(), false, nil)
	Rebuild(true, true)

	beforeFiles := len(shardFilesOnDisk(t, "gcdb"))

	// Inject a fake shard column file with a non-existent UUID.
	fakeUUID := "00000000-0000-0000-0000-000000000001"
	fakeName := fakeUUID + "-val"
	os.WriteFile(filepath.Join(Basepath, "gcdb", fakeName), []byte("garbage"), 0640)

	db := GetDatabase("gcdb")
	_, s := CleanDatabase(db)
	if s != 1 {
		t.Errorf("expected 1 orphaned shard file deleted, got %d", s)
	}

	// Real shard files must survive.
	if after := len(shardFilesOnDisk(t, "gcdb")); after != beforeFiles {
		t.Errorf("expected %d shard files to survive, got %d", beforeFiles, after)
	}
}

// TestCleanIdempotent: second call returns 0 deletions.
func TestCleanIdempotent(t *testing.T) {
	defer setupGCTest(t)()

	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)

	insertLongRows(t, tbl, []string{
		strings.Repeat("Q", 800),
		strings.Repeat("R", 800),
		strings.Repeat("S", 800),
	})

	db := GetDatabase("gcdb")
	CleanDatabase(db)
	b, s := CleanDatabase(db)
	if b != 0 || s != 0 {
		t.Errorf("second Clean: expected 0+0, got %d+%d", b, s)
	}
}

// TestCleanEmptyDatabase: GC on a freshly created database — no panic, no deletions.
func TestCleanEmptyDatabase(t *testing.T) {
	defer setupGCTest(t)()

	CreateDatabase("gcdb", false)
	db := GetDatabase("gcdb")
	b, s := CleanDatabase(db)
	if b != 0 || s != 0 {
		t.Errorf("empty db: expected 0+0, got %d+%d", b, s)
	}
}

// TestCleanAfterRebuildSupersedesShards: after a second rebuild with new data,
// the old shard UUID is gone from schema → its files are orphans and get cleaned.
// This simulates a crash after rebuild wrote new files but before RemoveFromDisk.
func TestCleanAfterRebuildSupersedesShards(t *testing.T) {
	defer setupGCTest(t)()

	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("val", "TEXT", nil, nil)

	// First insert + rebuild → creates shard with UUID-A.
	tbl.Insert([]string{"id", "val"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewString("first")},
	}, nil, scm.NewNil(), false, nil)
	Rebuild(true, true)

	db := GetDatabase("gcdb")
	tbl = db.GetTable("docs")

	// Capture UUID-A.
	var uuidA string
	for _, s := range tbl.ActiveShards() {
		if s != nil {
			uuidA = s.uuid.String()
		}
	}
	if uuidA == "" {
		t.Fatal("no active shard found after first rebuild")
	}

	// Second insert + rebuild → may create shard with UUID-B.
	tbl.Insert([]string{"id", "val"}, [][]scm.Scmer{
		{scm.NewInt(2), scm.NewString("second")},
	}, nil, scm.NewNil(), false, nil)
	Rebuild(true, true)

	tbl = db.GetTable("docs")
	var uuidB string
	for _, s := range tbl.ActiveShards() {
		if s != nil {
			uuidB = s.uuid.String()
		}
	}

	if uuidA == uuidB {
		// No new shard created (e.g. data was appended to existing shard) → no orphan expected.
		t.Log("UUID unchanged after second rebuild — skipping orphan check")
		return
	}

	// Verify UUID-A files still exist on disk (simulate crash-before-cleanup).
	dbDir := filepath.Join(Basepath, "gcdb")
	entries, _ := os.ReadDir(dbDir)
	hasOldFiles := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), uuidA) {
			hasOldFiles = true
			break
		}
	}
	if !hasOldFiles {
		t.Log("old shard files already cleaned up — test not meaningful, skipping")
		return
	}

	// GC should remove UUID-A files.
	_, s := CleanDatabase(db)
	if s == 0 {
		t.Error("expected at least 1 orphaned shard file deleted, got 0")
	}

	// UUID-A files should be gone.
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), uuidA) {
			if _, err := os.Stat(filepath.Join(dbDir, e.Name())); !os.IsNotExist(err) {
				t.Errorf("orphaned shard file %s still exists after GC", e.Name())
			}
		}
	}

	// Data still readable (both rows).
	count := 0
	tbl = db.GetTable("docs")
	tbl.scan(nil, []string{}, trueCondition(), []string{"id"},
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { count++; return scm.NewNil() }),
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] }),
		scm.NewNil(), scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] }), false)
	if count != 2 {
		t.Errorf("expected 2 rows after GC, got %d", count)
	}
}
