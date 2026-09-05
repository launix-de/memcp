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
	"time"

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

func TestCleanDatabaseWaitsForGenerationPublication(t *testing.T) {
	defer setupGCTest(t)()
	CreateDatabase("gcdb", false)
	db := GetDatabase("gcdb")

	db.persistenceLifecycle.RLock()
	done := make(chan struct{})
	go func() {
		CleanDatabase(db)
		close(done)
	}()
	select {
	case <-done:
		db.persistenceLifecycle.RUnlock()
		t.Fatal("cleanup entered while an unpublished generation was active")
	case <-time.After(25 * time.Millisecond):
	}
	db.persistenceLifecycle.RUnlock()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cleanup did not resume after generation publication")
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
		strings.Repeat("A", maxInlineBlobBytes+800),
		strings.Repeat("B", maxInlineBlobBytes+800),
		strings.Repeat("C", maxInlineBlobBytes+800),
	})

	db := GetDatabase("gcdb")
	b, s := CleanDatabase(db)
	if b != 0 || s != 0 {
		t.Errorf("expected 0 blobs, 0 shards deleted; got %d blobs, %d shards", b, s)
	}

	// Data still readable
	count := 0
	tbl.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{}, trueCondition(), []string{"id"},
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { count++; return a[0] }),
		scm.NewNil(), scm.NewNil(), false)
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
		strings.Repeat("X", maxInlineBlobBytes+800),
		strings.Repeat("Y", maxInlineBlobBytes+800),
		strings.Repeat("Z", maxInlineBlobBytes+800),
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
	tbl.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{}, trueCondition(), []string{"id"},
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { count++; return a[0] }),
		scm.NewNil(), scm.NewNil(), false)
	if count != 3 {
		t.Errorf("expected 3 rows after GC, got %d", count)
	}
}

// TestCleanBlobsBackfillsMissingLegacyManifest verifies the upgrade path: a
// legacy active generation is proven from its committed columns before cleanup
// proceeds. Live blobs survive, the new manifest is durable, and only then may
// an orphan be deleted.
func TestCleanBlobsBackfillsMissingLegacyManifest(t *testing.T) {
	defer setupGCTest(t)()

	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)
	insertLongRows(t, tbl, []string{
		strings.Repeat("m", maxInlineBlobBytes+1),
		strings.Repeat("n", maxInlineBlobBytes+1),
		strings.Repeat("o", maxInlineBlobBytes+1),
	})

	shards := tbl.ActiveShards()
	if len(shards) == 0 || shards[0] == nil {
		t.Fatal("expected an active shard")
	}
	tbl.schema.persistence.RemoveColumn(shards[0].uuid.String(), blobManifestColumn)

	orphanHash := "feedfacefeedfacefeedfacefeedface"
	orphanPath := filepath.Join(Basepath, "gcdb", "blob", orphanHash[:2], orphanHash[2:4])
	if err := os.MkdirAll(orphanPath, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orphanPath, orphanHash), []byte("orphan"), 0640); err != nil {
		t.Fatal(err)
	}

	deleted, _ := CleanDatabase(tbl.schema)
	if deleted != 1 {
		t.Fatalf("cleanup deleted %d blobs after legacy backfill, want 1 orphan", deleted)
	}
	if _, err := os.Stat(filepath.Join(orphanPath, orphanHash)); !os.IsNotExist(err) {
		t.Fatalf("orphan survived completed legacy proof: %v", err)
	}
	reader := tbl.schema.persistence.ReadColumn(shards[0].uuid.String(), blobManifestColumn)
	if _, failed := reader.(ErrorReader); failed {
		reader.Close()
		t.Fatal("legacy manifest was not persisted")
	}
	if _, valid := readBlobManifest(reader); !valid {
		t.Fatal("backfilled manifest is invalid")
	}
	if got := len(blobFiles(t, "gcdb")); got != 3 {
		t.Fatalf("live blob count after legacy backfill = %d, want 3", got)
	}
}

func TestStartupCleanLoadsColdSchemaBeforeBlobDeletion(t *testing.T) {
	defer setupGCTest(t)()

	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)
	insertLongRows(t, tbl, []string{
		strings.Repeat("c", maxInlineBlobBytes+1),
		strings.Repeat("d", maxInlineBlobBytes+1),
		strings.Repeat("e", maxInlineBlobBytes+1),
	})
	wantBlobs := len(blobFiles(t, "gcdb"))
	if wantBlobs == 0 {
		t.Fatal("expected external blob fixture")
	}

	for _, shard := range tbl.ActiveShards() {
		tbl.schema.persistence.RemoveColumn(shard.uuid.String(), blobManifestColumn)
	}
	orphanHash := "cabba9ecabba9ecabba9ecabba9ecabb"
	orphanPath := filepath.Join(Basepath, "gcdb", "blob", orphanHash[:2], orphanHash[2:4])
	if err := os.MkdirAll(orphanPath, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orphanPath, orphanHash), []byte("orphan"), 0640); err != nil {
		t.Fatal(err)
	}

	databases.Remove("gcdb")
	LoadDatabases()
	cold := GetDatabase("gcdb")
	if cold == nil || cold.srState != COLD {
		t.Fatal("expected lazily loaded database after catalog discovery")
	}
	deleted, _ := CleanDatabase(cold)
	if deleted != 1 {
		t.Fatalf("startup cleanup deleted %d blobs, want only the legacy orphan", deleted)
	}
	if got := len(blobFiles(t, "gcdb")); got != wantBlobs {
		t.Fatalf("live blob count after startup cleanup = %d, want %d", got, wantBlobs)
	}
}

func TestCleanBlobsRejectsCorruptManifest(t *testing.T) {
	defer setupGCTest(t)()

	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)
	insertLongRows(t, tbl, []string{
		strings.Repeat("p", maxInlineBlobBytes+1),
		strings.Repeat("q", maxInlineBlobBytes+1),
		strings.Repeat("r", maxInlineBlobBytes+1),
	})
	shard := tbl.ActiveShards()[0]
	writer := tbl.schema.persistence.WriteColumn(shard.uuid.String(), blobManifestColumn)
	if _, err := writer.Write([]byte(blobManifestHeader + strings.Repeat("0", 64) + "\n" + strings.Repeat("f", 64) + "\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	orphanHash := "01230123012301230123012301230123"
	orphanPath := filepath.Join(Basepath, "gcdb", "blob", orphanHash[:2], orphanHash[2:4])
	if err := os.MkdirAll(orphanPath, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orphanPath, orphanHash), []byte("orphan"), 0640); err != nil {
		t.Fatal(err)
	}
	deleted, _ := CleanDatabase(tbl.schema)
	if deleted != 0 {
		t.Fatalf("cleanup trusted a corrupt manifest and deleted %d blobs", deleted)
	}
}

func TestRepartitionPublishesBlobManifests(t *testing.T) {
	defer setupGCTest(t)()

	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)
	insertLongRows(t, tbl, []string{
		strings.Repeat("s", maxInlineBlobBytes+1),
		strings.Repeat("t", maxInlineBlobBytes+1),
		strings.Repeat("u", maxInlineBlobBytes+1),
		strings.Repeat("v", maxInlineBlobBytes+1),
	})
	if !tbl.beginManualRepartition() {
		t.Fatal("could not claim repartition")
	}
	tbl.repartition([]shardDimension{tbl.NewShardDimension("id", 2)})
	for _, shard := range tbl.ActiveShards() {
		reader := tbl.schema.persistence.ReadColumn(shard.uuid.String(), blobManifestColumn)
		if _, failed := reader.(ErrorReader); failed {
			reader.Close()
			t.Fatalf("repartitioned shard %s has no blob manifest", shard.uuid)
		}
		reader.Close()
	}
}

func TestBlobManifestFollowsComputedStorageWrappers(t *testing.T) {
	marker := "!0123456789abcdef0123456789abcdef"
	blob := &OverlayBlob{Base: &StorageConst{value: scm.NewString(marker), count: 1}}
	proxy := &StorageComputeProxy{main: blob, compressed: true, count: 1, delta: make(map[uint32]scm.Scmer)}
	references := make(map[string]struct{})
	appendColumnBlobReferences(proxy, references, 1)
	want := "3031323334353637383961626364656630313233343536373839616263646566"
	if _, ok := references[want]; !ok || len(references) != 1 {
		t.Fatalf("nested computed blob references = %v, want only %s", references, want)
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
		strings.Repeat("Q", maxInlineBlobBytes+800),
		strings.Repeat("R", maxInlineBlobBytes+800),
		strings.Repeat("S", maxInlineBlobBytes+800),
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
	tbl.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{}, trueCondition(), []string{"id"},
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { count++; return a[0] }),
		scm.NewNil(), scm.NewNil(), false)
	if count != 2 {
		t.Errorf("expected 2 rows after GC, got %d", count)
	}
}
