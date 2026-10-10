/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "testing"
import "github.com/launix-de/memcp/scm"

func TestMetadataExactNamesRestartAndDDL(t *testing.T) {
	previous := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = previous }()
	Init(scm.Globalenv)
	const name = "metadata_names"
	CreateDatabase(name, false)
	defer databases.Remove(name)
	db := GetDatabase(name)
	first, _ := CreateTable(name, "Item", Safe, false)
	second, _ := CreateTable(name, "item", Safe, false)
	first.CreateColumn("Id", "INT", nil, nil)
	if !first.CreateColumn("ID", "INT", nil, nil) {
		t.Fatal("exact column spelling lost")
	}
	if first == second || db.GetTable("Item") != first || db.GetTable("item") != second || !showMetadataTable(db, "ITEM").IsNil() {
		t.Fatal("engine added frontend name policy")
	}
	restored := newDatabase()
	restored.Name = name
	restored.persistence = db.persistence
	restored.srState = COLD
	restored.ensureLoaded()
	if restored.GetTable("Item") == nil || restored.GetTable("item") == nil || !showMetadataTable(restored, "ITEM").IsNil() {
		t.Fatal("restart lost exact names")
	}
	RenameTable(name, "Item", "Renamed")
	if !showMetadataTable(db, "Item").IsNil() || showMetadataTable(db, "Renamed").IsNil() || showMetadataTable(db, "item").IsNil() {
		t.Fatal("rename published incorrect exact names")
	}
	DropTable(name, "Renamed", false)
	if !showMetadataTable(db, "Renamed").IsNil() || showMetadataTable(db, "item").IsNil() {
		t.Fatal("drop affected case-distinct table")
	}
}
