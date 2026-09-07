/*
Copyright (C) 2026  Carl-Philip Hänsch

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

import "io"
import "os"
import "bytes"
import "encoding/json"
import "strings"
import "sync"
import "testing"
import "time"

import "github.com/launix-de/memcp/scm"

type recordingMovePersistence struct {
	PersistenceEngine
	events []string
}

func (p *recordingMovePersistence) WriteShardFile(name string) io.WriteCloser {
	p.events = append(p.events, "file:"+name)
	return p.PersistenceEngine.WriteShardFile(name)
}

func (p *recordingMovePersistence) WriteBlob(hash string) io.WriteCloser {
	p.events = append(p.events, "blob:"+hash)
	return p.PersistenceEngine.WriteBlob(hash)
}

func (p *recordingMovePersistence) WriteSchema(schema []byte) {
	p.events = append(p.events, "schema")
	p.PersistenceEngine.WriteSchema(schema)
}

type failingMovePersistence struct {
	PersistenceEngine
}

func (p *failingMovePersistence) WriteShardFile(string) io.WriteCloser {
	panic("injected move failure")
}

type blockingMovePersistence struct {
	PersistenceEngine
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func (p *blockingMovePersistence) WriteShardFile(name string) io.WriteCloser {
	p.once.Do(func() {
		close(p.started)
		<-p.release
	})
	return p.PersistenceEngine.WriteShardFile(name)
}

func writeMoveObject(t *testing.T, writer io.WriteCloser, data []byte) {
	t.Helper()
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}

func readMoveObject(t *testing.T, reader io.ReadCloser) []byte {
	t.Helper()
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestMoveDatabasePublishesSchemaLast(t *testing.T) {
	src := &FileStorage{path: t.TempDir() + "/"}
	dstFiles := &FileStorage{path: t.TempDir() + "/"}
	dst := &recordingMovePersistence{PersistenceEngine: dstFiles}

	writeMoveObject(t, src.WriteShardFile("shard-column"), []byte("column-data"))
	writeMoveObject(t, src.WriteShardFile("00000000-0000-0000-0000-000000000001.log"), []byte("backend-specific-wal"))
	writeMoveObject(t, src.WriteBlob("abcdef"), []byte("blob-data"))
	src.WriteSchema([]byte(`{"name":"move-test"}`))

	MoveDatabase(src, dst)

	if got := readMoveObject(t, dst.ReadShardFile("shard-column")); !bytes.Equal(got, []byte("column-data")) {
		t.Fatalf("copied shard data = %q", got)
	}
	if got := readMoveObject(t, dst.ReadBlob("abcdef")); !bytes.Equal(got, []byte("blob-data")) {
		t.Fatalf("copied blob data = %q", got)
	}
	if got := dst.ReadSchema(); !bytes.Equal(got, []byte(`{"name":"move-test"}`)) {
		t.Fatalf("copied schema = %q", got)
	}
	if len(dst.events) != 3 || dst.events[2] != "schema" {
		t.Fatalf("publication order = %v, want data before schema", dst.events)
	}
}

func TestMoveDatabaseFailureDoesNotPublishSchema(t *testing.T) {
	src := &FileStorage{path: t.TempDir() + "/"}
	dstFiles := &FileStorage{path: t.TempDir() + "/"}
	dst := &failingMovePersistence{PersistenceEngine: dstFiles}

	writeMoveObject(t, src.WriteShardFile("shard-column"), []byte("column-data"))
	src.WriteSchema([]byte(`{"name":"move-test"}`))

	defer func() {
		if recover() == nil {
			t.Fatal("MoveDatabase did not propagate the destination failure")
		}
		if got := dstFiles.ReadSchema(); len(got) != 0 {
			t.Fatalf("destination schema was published after failure: %q", got)
		}
	}()
	MoveDatabase(src, dst)
}

func TestDatabaseBackendConfigForTargetKeepsConnectionAndChangesNamespace(t *testing.T) {
	root := t.TempDir()
	oldBasepath := Basepath
	Basepath = root
	t.Cleanup(func() { Basepath = oldBasepath })
	source := json.RawMessage(`{"backend":"s3","endpoint":"https://objects.example","bucket":"shared","prefix":"source-prefix","access_key_id":"key","secret_access_key":"secret","force_path_style":true}`)
	if err := os.WriteFile(root+"/source.json", source, 0640); err != nil {
		t.Fatal(err)
	}

	var copied map[string]interface{}
	if err := json.Unmarshal(databaseBackendConfigForTarget("source", "target"), &copied); err != nil {
		t.Fatal(err)
	}
	if copied["endpoint"] != "https://objects.example" || copied["bucket"] != "shared" || copied["access_key_id"] != "key" || copied["secret_access_key"] != "secret" {
		t.Fatalf("connection configuration was not copied: %v", copied)
	}
	if copied["prefix"] != "target" {
		t.Fatalf("copied prefix = %v, want target", copied["prefix"])
	}
}

func TestAlterDatabaseStorageMovesDataAndPublishesDescriptor(t *testing.T) {
	root := t.TempDir()
	destinationRoot := t.TempDir()
	databaseName := "alter_storage_move"
	oldBasepath := Basepath
	oldFactory, hadFactory := BackendRegistry["test-filesystem"]
	Basepath = root
	BackendRegistry["test-filesystem"] = func(dbName string, _ json.RawMessage) PersistenceEngine {
		return &FileStorage{path: destinationRoot + "/" + dbName + "/"}
	}
	t.Cleanup(func() {
		db := databases.Remove(databaseName)
		if db != nil {
			db.closeTransactionLog()
		}
		if hadFactory {
			BackendRegistry["test-filesystem"] = oldFactory
		} else {
			delete(BackendRegistry, "test-filesystem")
		}
		Basepath = oldBasepath
	})

	CreateDatabase(databaseName, false)
	table, _ := CreateTable(databaseName, "items", Safe, false)
	table.CreateColumn("id", "INT", nil, nil)
	table.CreateColumn("value", "TEXT", nil, nil)
	largeValue := strings.Repeat("blob-value-", maxInlineBlobBytes)
	table.Insert([]string{"id", "value"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewString("preserved")},
		{scm.NewInt(2), scm.NewString(largeValue)},
		{scm.NewInt(3), scm.NewString(largeValue + "three")},
		{scm.NewInt(4), scm.NewString(largeValue + "four")},
	}, nil, scm.NewNil(), false, nil)

	tx := NewTxContext(TxCursorStability)
	tx.SessionState = &scm.SessionState{ID: 9001}
	target := json.RawMessage(`{"backend":"test-filesystem"}`)
	if !AlterDatabaseStorage(databaseName, target, tx) {
		t.Fatal("ALTER DATABASE storage returned false")
	}

	db := GetDatabase(databaseName)
	if got, want := db.persistence.StorageIdentity(), "filesystem:"+destinationRoot+"/"+databaseName; got != want {
		t.Fatalf("active storage = %q, want %q", got, want)
	}
	if _, err := os.Stat(destinationRoot + "/" + databaseName + "/schema.json"); err != nil {
		t.Fatalf("destination schema was not published: %v", err)
	}
	if _, err := os.Stat(root + "/" + databaseName); !os.IsNotExist(err) {
		t.Fatalf("source storage remains after successful cutover: %v", err)
	}
	config, err := os.ReadFile(root + "/" + databaseName + ".json")
	if err != nil {
		t.Fatalf("backend descriptor was not published: %v", err)
	}
	if !equalDatabaseBackendConfig(config, target) {
		t.Fatalf("backend descriptor = %s, want %s", config, target)
	}

	db.closeTransactionLog()
	databases.Remove(databaseName)
	reloaded := newDatabase()
	reloaded.Name = databaseName
	reloaded.persistence = createPersistenceFromConfig(databaseName, target)
	reloaded.srState = COLD
	databases.Set(reloaded.Name, reloaded)
	reloadedTable := reloaded.GetTable("items")
	if reloadedTable == nil {
		t.Fatal("moved table did not survive backend reload")
	}
	readTx := NewTxContext(TxCursorStability)
	got := reloadedTable.scanLookup(readTx, testLookupAccess([]string{"id"}, []scm.Scmer{scm.NewInt(1)}), "value", true)
	if !scm.Equal(got, scm.NewString("preserved")) {
		t.Fatalf("moved row after reload = %s, want preserved", scm.String(got))
	}
	got = reloadedTable.scanLookup(readTx, testLookupAccess([]string{"id"}, []scm.Scmer{scm.NewInt(2)}), "value", true)
	if !scm.Equal(got, scm.NewString(largeValue)) {
		t.Fatal("moved blob-backed row did not survive backend reload")
	}
}

func TestAlterDatabaseStorageFailureKeepsSourceActive(t *testing.T) {
	root := t.TempDir()
	destinationRoot := t.TempDir()
	databaseName := "alter_storage_failure"
	oldBasepath := Basepath
	oldFactory, hadFactory := BackendRegistry["failing-test-filesystem"]
	Basepath = root
	BackendRegistry["failing-test-filesystem"] = func(dbName string, _ json.RawMessage) PersistenceEngine {
		return &failingMovePersistence{PersistenceEngine: &FileStorage{path: destinationRoot + "/" + dbName + "/"}}
	}
	t.Cleanup(func() {
		db := databases.Remove(databaseName)
		if db != nil {
			db.closeTransactionLog()
		}
		if hadFactory {
			BackendRegistry["failing-test-filesystem"] = oldFactory
		} else {
			delete(BackendRegistry, "failing-test-filesystem")
		}
		Basepath = oldBasepath
	})

	CreateDatabase(databaseName, false)
	table, _ := CreateTable(databaseName, "items", Safe, false)
	table.CreateColumn("id", "INT", nil, nil)
	table.CreateColumn("value", "TEXT", nil, nil)
	table.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewString("source")}}, nil, scm.NewNil(), false, nil)
	sourceIdentity := table.schema.persistence.StorageIdentity()
	tx := NewTxContext(TxCursorStability)
	tx.SessionState = &scm.SessionState{ID: 9002}

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("ALTER DATABASE storage did not propagate copy failure")
			}
		}()
		AlterDatabaseStorage(databaseName, json.RawMessage(`{"backend":"failing-test-filesystem"}`), tx)
	}()

	db := GetDatabase(databaseName)
	if got := db.persistence.StorageIdentity(); got != sourceIdentity {
		t.Fatalf("active storage changed after failed move: got %q want %q", got, sourceIdentity)
	}
	if _, err := os.Stat(root + "/" + databaseName + ".json"); !os.IsNotExist(err) {
		t.Fatalf("backend descriptor was published after failed move: %v", err)
	}
	got := table.scanLookup(NewTxContext(TxCursorStability), testLookupAccess([]string{"id"}, []scm.Scmer{scm.NewInt(1)}), "value", true)
	if !scm.Equal(got, scm.NewString("source")) {
		t.Fatalf("source row after failed move = %s, want source", scm.String(got))
	}
}

func TestAlterDatabaseStorageKeepsReadersOnPublishedGeneration(t *testing.T) {
	root := t.TempDir()
	destinationRoot := t.TempDir()
	databaseName := "alter_storage_online"
	oldBasepath := Basepath
	oldFactory, hadFactory := BackendRegistry["blocking-test-filesystem"]
	started := make(chan struct{})
	release := make(chan struct{})
	Basepath = root
	BackendRegistry["blocking-test-filesystem"] = func(dbName string, _ json.RawMessage) PersistenceEngine {
		return &blockingMovePersistence{
			PersistenceEngine: &FileStorage{path: destinationRoot + "/" + dbName + "/"},
			started:           started,
			release:           release,
		}
	}
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		db := databases.Remove(databaseName)
		if db != nil {
			db.closeTransactionLog()
		}
		if hadFactory {
			BackendRegistry["blocking-test-filesystem"] = oldFactory
		} else {
			delete(BackendRegistry, "blocking-test-filesystem")
		}
		Basepath = oldBasepath
	})

	CreateDatabase(databaseName, false)
	table, _ := CreateTable(databaseName, "items", Safe, false)
	table.CreateColumn("id", "INT", nil, nil)
	table.CreateColumn("value", "TEXT", nil, nil)
	table.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewString("old-generation")}}, nil, scm.NewNil(), false, nil)
	oldTopology := table.activeTopology()
	tx := NewTxContext(TxCursorStability)
	tx.SessionState = &scm.SessionState{ID: 9003}
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		AlterDatabaseStorage(databaseName, json.RawMessage(`{"backend":"blocking-test-filesystem"}`), tx)
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("storage migration did not reach the blocked destination copy")
	}
	if got := table.activeTopology(); got != oldTopology {
		t.Fatal("storage migration published the replacement before destination completion")
	}
	readDone := make(chan scm.Scmer, 1)
	go func() {
		readDone <- table.scanLookup(NewTxContext(TxCursorStability), testLookupAccess([]string{"id"}, []scm.Scmer{scm.NewInt(1)}), "value", true)
	}()
	select {
	case got := <-readDone:
		if !scm.Equal(got, scm.NewString("old-generation")) {
			t.Fatalf("read during migration = %s, want old-generation", scm.String(got))
		}
	case <-time.After(time.Second):
		t.Fatal("reader blocked behind storage migration")
	}

	// The old generation remains the write entry point until publication, while
	// its completed rebuild successor receives the same mutation.
	table.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("mirrored")}}, nil, scm.NewNil(), false, nil)
	if table.maintenanceMu.TryLock() {
		table.maintenanceMu.Unlock()
		t.Fatal("storage migration did not exclude a concurrent rebuild")
	}
	ddlDone := make(chan struct{})
	go func() {
		CreateTable(databaseName, "created_after_move", Memory, false)
		close(ddlDone)
	}()
	select {
	case <-ddlDone:
		t.Fatal("catalog mutation passed an unpublished storage generation")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case recovered := <-done:
		if recovered != nil {
			t.Fatalf("storage migration panicked: %v", recovered)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("storage migration did not finish")
	}
	select {
	case <-ddlDone:
	case <-time.After(5 * time.Second):
		t.Fatal("catalog mutation did not resume after storage migration")
	}
	if !table.maintenanceMu.TryLock() {
		t.Fatal("storage migration did not release the rebuild claim")
	}
	table.maintenanceMu.Unlock()

	db := GetDatabase(databaseName)
	db.closeTransactionLog()
	databases.Remove(databaseName)
	reloaded := newDatabase()
	reloaded.Name = databaseName
	reloaded.persistence = createPersistenceFromConfig(databaseName, json.RawMessage(`{"backend":"blocking-test-filesystem"}`))
	reloaded.srState = COLD
	databases.Set(reloaded.Name, reloaded)
	reloadedTable := reloaded.GetTable("items")
	got := reloadedTable.scanLookup(NewTxContext(TxCursorStability), testLookupAccess([]string{"id"}, []scm.Scmer{scm.NewInt(2)}), "value", true)
	if !scm.Equal(got, scm.NewString("mirrored")) {
		t.Fatalf("concurrent write after reload = %s, want mirrored", scm.String(got))
	}
}

func BenchmarkDatabaseCatalogLookup(b *testing.B) {
	databaseName := "benchmark_database_catalog_lookup"
	db := newDatabase()
	db.Name = databaseName
	databases.Set(db.Name, db)
	b.Cleanup(func() { databases.Remove(databaseName) })
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if GetDatabase(databaseName) != db {
			b.Fatal("database catalog lookup returned a different database")
		}
	}
}

func BenchmarkTableCatalogLookup(b *testing.B) {
	db := newDatabase()
	tableName := "benchmark_table_catalog_lookup"
	tbl := &table{Name: tableName, schema: db}
	db.tables.Set(tableName, tbl)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if db.tables.Get(tableName) != tbl {
			b.Fatal("table catalog lookup returned a different table")
		}
	}
}
