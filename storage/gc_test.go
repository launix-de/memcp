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
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	marker := "!b0123456789abcdef0123456789abcdef"
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

func TestBlobManifestUpgradesIncompleteV1(t *testing.T) {
	defer setupGCTest(t)()
	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)
	values := []string{blobBangPayload(), strings.Repeat("y", 5300), strings.Repeat("z", 5300)}
	insertLongRows(t, tbl, values)
	shard := tbl.ActiveShards()[0]
	// An old column and an internally valid but incomplete v1 manifest.
	// Neither their format nor their contents are generated by the new writer.
	raw := make([]string, len(values))
	for i, value := range values {
		hash := sha256.Sum256([]byte(value))
		raw[i] = "!" + string(hash[:])
	}
	writer := tbl.schema.persistence.WriteColumn(shard.uuid.String(), "content")
	writer.Write(legacyBlobFixture(0, raw, nil))
	finishColumnWrite(writer, true)
	tbl.schema.persistence.RemoveColumn(shard.uuid.String(), blobManifestColumn)
	writer = tbl.schema.persistence.WriteColumn(shard.uuid.String(), ".blobrefs-v1")
	fmt.Fprintf(writer, "memcp-blob-references-v1\n%x\n", sha256.Sum256(nil))
	finishColumnWrite(writer, true)
	databases.Remove("gcdb")
	LoadDatabases()
	db := GetDatabase("gcdb")
	if deleted, _ := CleanDatabase(db); deleted != 0 {
		t.Fatalf("upgrade deleted %d live blobs", deleted)
	}
	if len(blobFiles(t, "gcdb")) != len(values) {
		t.Fatal("legacy blob payload lost")
	}
	refs, valid := readBlobManifest(db.persistence.ReadColumn(shard.uuid.String(), blobManifestColumn))
	if !valid || len(refs) != len(values) {
		t.Fatalf("v2 ownership incomplete: %v", refs)
	}
	for _, value := range values {
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
		if _, ok := refs[hash]; !ok {
			t.Errorf("missing hash %s", hash)
		}
	}
}

func TestBlobManifestSurvivesUnchangedColdRebuild(t *testing.T) {
	defer setupGCTest(t)()
	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)
	insertLongRows(t, tbl, []string{blobBangPayload(), strings.Repeat("y", 5300), strings.Repeat("z", 5300)})
	shardID := tbl.ActiveShards()[0].uuid.String()
	path := filepath.Join(Basepath, "gcdb", shardID+"-"+blobManifestColumn)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	databases.Remove("gcdb")
	LoadDatabases()
	db := GetDatabase("gcdb")
	cold := db.GetTable("docs")
	RebuildTable(cold, false, false)
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("unchanged cold rebuild overwrote ownership with a partial column map")
	}
	if deleted, _ := CleanDatabase(db); deleted != 0 {
		t.Fatalf("cold rebuild lost %d live blobs", deleted)
	}
}

// A crash may leave the operational count below the number of committed owners.
// Releasing one owner must never remove another owner's payload.
func TestBlobUndercountRetainsCommittedOwners(t *testing.T) {
	for _, scenario := range []string{"shared", "restart", "rebuild", "failed-publication"} {
		t.Run(scenario, func(t *testing.T) {
			defer setupGCTest(t)()
			CreateDatabase("gcdb", false)
			payloads := []string{strings.Repeat("a", maxInlineBlobBytes+1), strings.Repeat("b", maxInlineBlobBytes+1), strings.Repeat("c", maxInlineBlobBytes+1)}
			for _, name := range []string{"first", "second"} {
				tbl, _ := CreateTable("gcdb", name, Safe, false)
				tbl.CreateColumn("id", "INT", nil, nil)
				tbl.CreateColumn("content", "TEXT", nil, nil)
				insertLongRows(t, tbl, payloads)
			}
			db := GetDatabase("gcdb")
			for hash, count := range queryBlobsTable(t, db) {
				if count != 2 {
					t.Fatalf("fixture count = %d, want 2", count)
				}
				db.DecrBlobRefcount(hash)
			}
			if scenario == "restart" {
				Rebuild(true, true)
				databases.Remove("gcdb")
				LoadDatabases()
				db = GetDatabase("gcdb")
				db.ensureLoaded()
			}
			if scenario == "rebuild" {
				Rebuild(true, true)
			}
			if scenario == "failed-publication" {
				persistence := db.persistence
				failing := &failSchemaWritePersistence{PersistenceEngine: persistence, failAt: 1}
				db.persistence = failing
				tbl := db.tables.Get("second")
				tbl.Insert([]string{"id", "content"}, [][]scm.Scmer{{scm.NewInt(4), scm.NewString(payloads[0])}}, nil, scm.NewNil(), false, nil)
				result := Rebuild(true, true)
				db.persistence = persistence
				if !strings.Contains(result, "schema publication failure") {
					t.Fatalf("missing injected failure: %s", result)
				}
			}
			DropTable("gcdb", "first", false)
			if got := len(blobFiles(t, "gcdb")); got != 3 {
				t.Fatalf("decrement deleted committed blobs: got %d, want 3", got)
			}
			if deleted, _ := CleanDatabase(db); deleted != 0 {
				t.Fatalf("cleanup deleted %d owned blobs", deleted)
			}
			references, complete := activeBlobReferences(db, true)
			if !complete || len(references) != 3 {
				t.Fatalf("references = %v, complete = %v", references, complete)
			}
			for _, payload := range payloads {
				hash := fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))
				reader := db.persistence.ReadBlob(hash)
				value, ok := gunzipReader(reader)
				reader.Close()
				if !ok || value.String() != payload {
					t.Fatalf("surviving payload %s unreadable", hash)
				}
			}
			DropTable("gcdb", "second", false)
			if got := len(blobFiles(t, "gcdb")); got != 3 {
				t.Fatalf("drop deleted blobs before ownership check: %d", got)
			}
			if deleted, _ := CleanDatabase(db); deleted != 3 {
				t.Fatalf("cleanup deleted %d orphans, want 3", deleted)
			}
		})
	}
}

// Audits must work even on backends whose ReadBlob eagerly loads the payload.
type inventoryMetadataOnlyPersistence struct {
	PersistenceEngine
	failList bool
}

func (p *inventoryMetadataOnlyPersistence) ReadBlob(string) io.ReadCloser {
	panic("inventory fetched payload")
}
func (p *inventoryMetadataOnlyPersistence) WriteColumn(string, string) io.WriteCloser {
	panic("inventory wrote column metadata")
}
func (p *inventoryMetadataOnlyPersistence) DeleteBlob(string) { panic("inventory deleted blob") }
func (p *inventoryMetadataOnlyPersistence) WalkBlobs(fn func(string)) {
	if p.failList {
		panic("injected inventory listing failure")
	}
	p.PersistenceEngine.WalkBlobs(fn)
}

func TestBlobInventoryListsMissingReferencesWithoutFetchingPayloads(t *testing.T) {
	defer setupGCTest(t)()
	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "docs", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)
	insertLongRows(t, tbl, []string{strings.Repeat("x", maxInlineBlobBytes+1), strings.Repeat("y", maxInlineBlobBytes+1), strings.Repeat("z", maxInlineBlobBytes+1)})
	db := GetDatabase("gcdb")
	references, complete := activeBlobReferences(db, true)
	if !complete || len(references) != 3 {
		t.Fatal("incomplete fixture")
	}
	backend := db.persistence
	db.persistence = &inventoryMetadataOnlyPersistence{PersistenceEngine: backend}
	report := AuditBlobInventory(db)
	if report.Referenced != 3 || report.Listed != 3 || len(report.Missing) != 0 || report.Unreferenced != 0 {
		t.Fatalf("healthy inventory: %+v", report)
	}
	if deleted, _ := CleanDatabase(db); deleted != 0 {
		t.Fatalf("cleanup deleted %d live blobs", deleted)
	}
	// Missing legacy manifests are discovered without publishing a backfill.
	for _, shard := range tbl.ActiveShards() {
		backend.RemoveColumn(shard.uuid.String(), blobManifestColumn)
	}
	legacy := AuditBlobInventory(db)
	if legacy.Referenced != 3 || len(legacy.Missing) != 0 {
		t.Fatalf("legacy inventory: %+v", legacy)
	}
	for _, shard := range tbl.ActiveShards() {
		reader := backend.ReadColumn(shard.uuid.String(), blobManifestColumn)
		reader.Close()
		if missing, ok := reader.(ErrorReader); !ok || !missing.Missing() {
			t.Fatal("audit published legacy manifest")
		}
	}
	for hash := range references {
		backend.DeleteBlob(hash)
	}
	orphan := strings.Repeat("ab", 32)
	writer := db.persistence.WriteBlob(orphan)
	if _, err := writer.Write([]byte("orphan")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	report = AuditBlobInventory(db)
	if report.Referenced != 3 || report.Listed != 1 || len(report.Missing) != 3 || report.Unreferenced != 1 {
		t.Fatalf("damaged inventory: %+v", report)
	}
	for _, hash := range report.Missing {
		if _, expected := references[hash]; !expected {
			t.Fatalf("unexpected missing hash %s", hash)
		}
	}
	if len(blobFiles(t, "gcdb")) != 1 {
		t.Fatal("audit deleted the orphan")
	}
	db.persistence = backend
	if deleted, _ := CleanDatabase(db); deleted != 1 {
		t.Fatalf("cleanup retained an unowned blob because another file is missing: deleted %d", deleted)
	}
	if got := len(blobFiles(t, "gcdb")); got != 0 {
		t.Fatalf("orphan not collected: %d files", got)
	}
}

func TestBlobInventoryRejectsIncompleteSnapshotsAndReleasesLock(t *testing.T) {
	for _, failure := range []string{"manifest", "listing"} {
		t.Run(failure, func(t *testing.T) {
			defer setupGCTest(t)()
			CreateDatabase("gcdb", false)
			tbl, _ := CreateTable("gcdb", "docs", Safe, false)
			tbl.CreateColumn("id", "INT", nil, nil)
			tbl.CreateColumn("content", "TEXT", nil, nil)
			insertLongRows(t, tbl, []string{strings.Repeat("a", maxInlineBlobBytes+1), strings.Repeat("b", maxInlineBlobBytes+1), strings.Repeat("c", maxInlineBlobBytes+1)})
			db := GetDatabase("gcdb")
			if failure == "manifest" {
				writer := db.persistence.WriteColumn(tbl.ActiveShards()[0].uuid.String(), blobManifestColumn)
				if _, err := writer.Write([]byte("corrupt")); err != nil {
					t.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
			}
			db.persistence = &inventoryMetadataOnlyPersistence{PersistenceEngine: db.persistence, failList: failure == "listing"}
			var caught any
			func() { defer func() { caught = recover() }(); AuditBlobInventory(db) }()
			expected := "incomplete generation references"
			if failure == "listing" {
				expected = "injected inventory listing failure"
			}
			if !strings.Contains(fmt.Sprint(caught), expected) {
				t.Fatalf("unexpected error: %v", caught)
			}
			if !db.persistenceLifecycle.TryLock() {
				t.Fatal("audit retained lifecycle lock after failure")
			}
			db.persistenceLifecycle.Unlock()
			if len(blobFiles(t, "gcdb")) != 3 {
				t.Fatal("failed audit changed files")
			}
		})
	}
}

func TestBlobInventoryWaitsForGenerationPublication(t *testing.T) {
	defer setupGCTest(t)()
	CreateDatabase("gcdb", false)
	db := GetDatabase("gcdb")
	db.persistenceLifecycle.RLock()
	done := make(chan BlobInventory, 1)
	go func() { done <- AuditBlobInventory(db) }()
	select {
	case <-done:
		db.persistenceLifecycle.RUnlock()
		t.Fatal("audit entered an unpublished generation")
	case <-time.After(25 * time.Millisecond):
	}
	db.persistenceLifecycle.RUnlock()
	select {
	case report := <-done:
		if report.Referenced != 0 || report.Listed != 0 {
			t.Fatalf("empty inventory: %+v", report)
		}
	case <-time.After(time.Second):
		t.Fatal("audit did not resume after publication")
	}
}

// pausingBlobPersistence pauses the first WriteBlob write right after the
// blob file is durable on disk, so a test can deterministically interleave a
// concurrent CleanDatabase call inside the build->publish window instead of
// relying on goroutine-scheduling luck.
type pausingBlobPersistence struct {
	PersistenceEngine
	once    sync.Once
	written chan struct{}
	resume  chan struct{}
}

func newPausingBlobPersistence(engine PersistenceEngine) *pausingBlobPersistence {
	return &pausingBlobPersistence{PersistenceEngine: engine, written: make(chan struct{}), resume: make(chan struct{})}
}

func (p *pausingBlobPersistence) WriteBlob(hash string) io.WriteCloser {
	return &pausingBlobWriter{WriteCloser: p.PersistenceEngine.WriteBlob(hash), p: p}
}

type pausingBlobWriter struct {
	io.WriteCloser
	p *pausingBlobPersistence
}

func (w *pausingBlobWriter) Close() error {
	err := w.WriteCloser.Close()
	w.p.once.Do(func() {
		close(w.p.written)
		<-w.p.resume
	})
	return err
}

// readContentColumn scans the "content" column, recovering a panic instead of
// crashing the test process so a missing-blob corruption surfaces as a
// regular test failure.
func readContentColumn(tbl *table) (values []string, panicked any) {
	defer func() { panicked = recover() }()
	tbl.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{}, trueCondition(), []string{"content"},
		// mapReduce receives (accumulator, content); a[0] is the fold
		// accumulator (unused here), a[1] is the requested column value.
		scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { values = append(values, a[1].String()); return a[0] }),
		scm.NewNil(), scm.NewNil(), false)
	return
}

// TestOverflowRebuildBlobSurvivesConcurrentClean reproduces the file-upload
// 500 seen in production: a shard-overflow rebuild (the automatic background
// compaction triggered when ordinary INSERTs, e.g. uploaded file content,
// push a shard past Settings.ShardSize) writes a new blob file to disk and
// only later publishes the rebuilt generation into the table's active shard
// list. Until that publish, activeBlobReferences cannot see the new blob
// (the still-active old shard never referenced it), so a CleanDatabase sweep
// racing this window treats the freshly-written blob as an orphan and
// deletes it out from under the in-flight rebuild.
func TestOverflowRebuildBlobSurvivesConcurrentClean(t *testing.T) {
	defer setupGCTest(t)()
	CreateDatabase("gcdb", false)
	db := GetDatabase("gcdb")
	pausing := newPausingBlobPersistence(db.persistence)
	db.persistence = pausing

	tbl, _ := CreateTable("gcdb", "uploads", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("content", "TEXT", nil, nil)

	// A single distinct value would be optimized to a constant column and
	// never touch OverlayBlob at all; use distinct values like the existing
	// blob-producing fixtures elsewhere in this file (e.g. TestCleanNoOrphans).
	uploaded := strings.Repeat("U", maxInlineBlobBytes+800)
	tbl.Insert([]string{"id", "content"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewString(uploaded)},
		{scm.NewInt(2), scm.NewString(strings.Repeat("V", maxInlineBlobBytes+800))},
		{scm.NewInt(3), scm.NewString(strings.Repeat("W", maxInlineBlobBytes+800))},
	}, nil, scm.NewNil(), false, nil)

	topology := tbl.pinActiveTopology()
	source := topology.shards[len(topology.shards)-1]

	rebuildDone := make(chan struct{})
	go func() {
		defer close(rebuildDone)
		// Force the overflow branch regardless of the configured ShardSize:
		// this is exactly what an oversized upload triggers organically.
		published, _ := tbl.appendFreeShardDurably(topology, source, Settings.ShardSize+1)
		if published != nil {
			topology.releaseOperation()
		}
	}()

	select {
	case <-pausing.written:
		// blob file is now durable on disk; the rebuilt generation is not
		// yet published into t.Shards.
	case <-time.After(5 * time.Second):
		t.Fatal("overflow rebuild never wrote the blob")
	}

	cleanDone := make(chan struct{})
	go func() {
		CleanDatabase(db)
		close(cleanDone)
	}()

	select {
	case <-cleanDone:
		t.Fatal("CleanDatabase ran to completion while an overflow rebuild had an unpublished blob in flight")
	case <-time.After(50 * time.Millisecond):
	}

	close(pausing.resume)

	select {
	case <-rebuildDone:
	case <-time.After(5 * time.Second):
		t.Fatal("overflow rebuild did not finish after being released")
	}
	select {
	case <-cleanDone:
	case <-time.After(5 * time.Second):
		t.Fatal("CleanDatabase did not resume after the overflow rebuild published its generation")
	}

	values, panicked := readContentColumn(tbl)
	if panicked != nil {
		t.Fatalf("reading the uploaded rows panicked (a blob was deleted from under the in-flight rebuild): %v", panicked)
	}
	if len(values) != 3 {
		t.Fatalf("uploaded rows corrupted by a concurrent GC race: got %d rows, want 3", len(values))
	}
	found := false
	for _, v := range values {
		if v == uploaded {
			found = true
		}
	}
	if !found {
		for i, v := range values {
			t.Logf("row %d: len=%d prefix=%q", i, len(v), v[:min(20, len(v))])
		}
		t.Fatalf("uploaded row content corrupted by a concurrent GC race")
	}
}
