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
import "testing"

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
	databases.Set(reloaded)
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
