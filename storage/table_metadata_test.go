/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "sync"
import "testing"
import "github.com/launix-de/memcp/scm"

func TestShowMetadataImmutablePublications(t *testing.T) {
	db := newDatabase()
	db.Name = "keys"
	db.srState = SHARED
	db.loadOnce.Do(func() {})
	parent := &table{Name: "Parent", Unique: []uniqueKey{{Id: "PRIMARY", Cols: []string{"second", "first"}}}}
	child := &table{Name: "Child", Unique: []uniqueKey{{Id: "UQ_name", Cols: []string{"name"}}}, Foreign: []foreignKey{{"FK_parent", "Child", []string{"p_second", "p_first"}, "Parent", []string{"second", "first"}, CASCADE, SETNULL, false}}}
	parent.schema, child.schema = db, db
	db.tables.Set(parent.Name, parent)
	db.tables.Set(child.Name, child)
	db.schemalock.Lock()
	db.metadataRevision++
	db.schemalock.Unlock()
	old := showMetadataTable(db, "Child")
	if scm.String(metadataField(old, "Name")) != "Child" {
		t.Fatal("metadata name")
	}
	relationships := metadataField(old, "ForeignKeys").Slice()
	if len(relationships) != 1 || metadataField(relationships[0], "LocalColumns").Slice()[1].String() != "p_first" {
		t.Fatal("composite foreign key order", relationships)
	}
	if metadataField(showMetadataTable(db, "Parent"), "Unique").Slice()[0].Slice()[3].Slice()[0].String() != "second" {
		t.Fatal("composite primary key order")
	}
	db.schemalock.Lock()
	child.Unique[0].Cols[0] = "renamed"
	child.Foreign[0].Cols1[0] = "changed"
	db.metadataRevision++
	db.schemalock.Unlock()
	if metadataField(metadataField(old, "Unique").Slice()[0], "Cols").Slice()[0].String() != "name" || metadataField(relationships[0], "LocalColumns").Slice()[0].String() != "p_second" {
		t.Fatal("old snapshot mutated")
	}
	if !showMetadataTable(db, "absent").IsNil() {
		t.Fatal("invented absent table")
	}
}

func TestShowMetadataConcurrentDDLPublication(t *testing.T) {
	db := newDatabase()
	db.Name = "keys"
	db.srState = SHARED
	db.loadOnce.Do(func() {})
	db.schemalock.Lock()
	db.schemalock.Unlock()
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
					for _, value := range showMetadataTable(db, "").Slice() {
						if metadataField(value, "Name").String() != "Live" {
							t.Error("partial metadata publication")
							return
						}
					}
				}
			}
		}()
	}
	for i := 0; i < 40; i++ {
		db.schemalock.Lock()
		db.tables.Set("Live", &table{schema: db, Name: "Live", Unique: []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}})
		db.metadataRevision++
		db.tables.Remove("Live")
		db.metadataRevision++
		db.schemalock.Unlock()
	}
	close(done)
	readers.Wait()
}

// Exercise the normal SHOW metadata builder, retaining the returned values
// across DDL just as a frontend retains its bound schema information.
func showMetadataTable(db *database, name string) scm.Scmer {
	db.ensureLoaded()
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	if name != "" {
		table := db.tables.Get(name)
		if table == nil {
			return scm.NewNil()
		}
		return showBuildMeta(db, table, true)
	}
	values := make([]scm.Scmer, 0)
	for _, table := range db.tables.GetAll() {
		values = append(values, showBuildMeta(db, table, true))
	}
	return scm.NewSlice(values)
}

func metadataField(value scm.Scmer, name string) scm.Scmer {
	values := value.Slice()
	for i := 0; i+1 < len(values); i += 2 {
		if values[i].String() == name {
			return values[i+1]
		}
	}
	return scm.NewNil()
}

func TestShowMetadataRestartAndBufferedDDL(t *testing.T) {
	oldBasepath := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	const name = "sql_key_metadata_restart"
	CreateDatabase(name, false)
	defer databases.Remove(name)
	db := GetDatabase(name)
	parent, _ := CreateTable(name, "Parent", Safe, false)
	parent.CreateColumn("Id", "INT", nil, []scm.Scmer{scm.NewString("primary"), scm.NewBool(true)})
	child, _ := CreateTable(name, "Child", Safe, false)
	child.CreateColumn("ParentId", "INT", nil, nil)
	created := scm.Apply(scm.Globalenv.Vars[scm.Symbol("createforeignkey")], NewTableScmer(child), scm.NewString("FK_parent"), scm.NewSlice([]scm.Scmer{scm.NewString("ParentId")}), NewTableScmer(parent), scm.NewSlice([]scm.Scmer{scm.NewString("Id")}), scm.NewString("cascade"), scm.NewString("restrict"))
	if !created.Bool() {
		t.Fatal("fixture foreign key was not created")
	}
	retained := scm.Apply(scm.Globalenv.Vars[scm.Symbol("show")], NewTableScmer(child), scm.NewBool(true))
	columns := metadataField(retained, "Columns").Slice()
	if len(columns) != 1 || metadataField(columns[0], "Field").String() != "ParentId" || &columns[0] != &child.ShowColumns().Slice()[0] {
		t.Fatal("SHOW did not reuse the published column definitions")
	}
	restored := newDatabase()
	restored.Name = name
	restored.persistence = db.persistence
	restored.srState = COLD
	restored.ensureLoaded()
	foreign := metadataField(showMetadataTable(restored, "Child"), "ForeignKeys").Slice()
	if len(foreign) != 1 || metadataField(foreign[0], "Id").String() != "FK_parent" || metadataField(foreign[0], "UpdateMode").String() != "cascade" {
		t.Fatal("restart lost foreign definitions", foreign)
	}
	db.schemalock.Lock()
	child.Foreign = nil
	db.saveLockedAndUnlock(schemaSaveBuffered)
	if len(metadataField(showMetadataTable(db, "Child"), "ForeignKeys").Slice()) != 0 || len(metadataField(retained, "ForeignKeys").Slice()) != 1 {
		t.Fatal("buffered mutation did not publish or mutated retained catalog")
	}
}
