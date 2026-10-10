/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "sync"
import "testing"
import "github.com/launix-de/memcp/scm"

func TestOpaqueMetadataRestartAndStalePublication(t *testing.T) {
	previous := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = previous }()
	Init(scm.Globalenv)
	const name = "opaque_metadata_restart"
	CreateDatabase(name, false)
	defer databases.Remove(name)
	db := GetDatabase(name)
	tbl, _ := CreateTable(name, "items", Safe, false)
	payload := scm.NewSlice([]scm.Scmer{scm.NewString("arbitrary"), scm.NewInt(256)})
	tbl.CreateColumn("value", "ANY", nil, []scm.Scmer{scm.NewString("metadata"), payload})
	db.schemalock.Lock()
	db.metadataValue = payload
	tbl.Metadata = scm.NewInt(42)
	db.saveLockedAndUnlock(schemaSaveFsync)
	restored := newDatabase()
	restored.Name = name
	restored.persistence = db.persistence
	restored.srState = COLD
	restored.ensureLoaded()
	if !scm.Equal(metadataField(restored.schemaMetadata(), "value"), payload) || restored.GetTable("items").Metadata.Int() != 42 || !scm.Equal(restored.GetTable("items").Columns[0].Metadata, payload) {
		t.Fatal("opaque metadata lost during reload")
	}
	current := db.schemaMetadata()
	stale := scm.NewSlice([]scm.Scmer{scm.NewString("revision"), scm.NewInt(int64(metadataField(current, "revision").Int()) - 1), scm.NewString("value"), scm.NewInt(999)})
	requirePanic(t, func() {
		scm.Apply(scm.Globalenv.Vars[scm.Symbol("altercolumn")], NewTableScmer(tbl), scm.NewString("value"), scm.NewString("options"), scm.NewSlice([]scm.Scmer{scm.NewString("default"), scm.NewInt(7), scm.NewString("metadata"), scm.NewInt(999)}), stale)
	})
	if tbl.Columns[0].hasDefault() || !scm.Equal(tbl.Columns[0].Metadata, payload) || !scm.Equal(db.schemaMetadata(), current) {
		t.Fatal("stale publication changed declaration or schema")
	}
	if !db.schemalock.TryLock() {
		t.Fatal("failed publication retained schema lock")
	}
	db.schemalock.Unlock()
	publication := scm.NewSlice([]scm.Scmer{scm.NewString("revision"), scm.NewInt(int64(metadataField(current, "revision").Int())), scm.NewString("value"), scm.NewInt(123)})
	scm.Apply(scm.Globalenv.Vars[scm.Symbol("altercolumn")], NewTableScmer(tbl), scm.NewString("value"), scm.NewString("options"), scm.NewSlice([]scm.Scmer{scm.NewString("default"), scm.NewInt(7), scm.NewString("metadata"), scm.NewInt(88)}), publication)
	if tbl.Columns[0].Default.Int() != 7 || tbl.Columns[0].Metadata.Int() != 88 || metadataField(db.schemaMetadata(), "value").Int() != 123 || !scm.Equal(metadataField(current, "value"), payload) {
		t.Fatal("atomic grouped publication failed")
	}
}

func TestOpaqueMetadataReadersSeeImmutableGenerations(t *testing.T) {
	db := newDatabase()
	db.Name = "opaque_metadata_snapshots"
	db.loadOnce.Do(func() {})
	db.schemalock.Lock()
	db.metadataValue = scm.NewInt(0)
	db.metadataRevision++
	db.schemalock.Unlock()
	retained := db.schemaMetadata()
	var readers sync.WaitGroup
	done := make(chan struct{})
	for i := 0; i < 4; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-done:
					return
				default:
				}
				snapshot := db.schemaMetadata()
				if metadataField(snapshot, "value").Int() != metadataField(snapshot, "revision").Int()-1 {
					t.Error("partial metadata generation")
					return
				}
			}
		}()
	}
	for i := 1; i <= 100; i++ {
		db.schemalock.Lock()
		db.metadataValue = scm.NewInt(int64(i))
		db.metadataRevision++
		db.schemalock.Unlock()
	}
	close(done)
	readers.Wait()
	if metadataField(retained, "value").Int() != 0 || metadataField(retained, "revision").Int() != 1 {
		t.Fatal("retained metadata mutated")
	}
}
