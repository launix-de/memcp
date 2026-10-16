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

import "testing"

import "github.com/launix-de/memcp/scm"

import "github.com/launix-de/NonLockingReadMap"

func requirePanic(t *testing.T, action func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("operation did not panic")
		}
	}()
	action()
}

func TestInternalDatabaseProtection(t *testing.T) {
	for _, schema := range []string{"system", "SYSTEM", "system_statistic"} {
		if databaseMaintenanceCapabilities(schema).canDrop {
			t.Errorf("databaseMaintenanceCapabilities(%q).canDrop = true, want false", schema)
		}
	}
	if !databaseMaintenanceCapabilities("application").canDrop {
		t.Fatal("application database must remain droppable")
	}
}

func TestInternalTableProtection(t *testing.T) {
	for _, test := range []struct {
		schema string
		table  string
	}{
		{schema: "system", table: "user"},
		{schema: "system_statistic", table: "logs"},
		{schema: "application", table: ".blobs"},
	} {
		if tableMaintenanceCapabilities(test.schema, test.table).canDrop {
			t.Errorf("tableMaintenanceCapabilities(%q, %q).canDrop = true, want false", test.schema, test.table)
		}
	}
	if !tableMaintenanceCapabilities("application", "logs").canDrop {
		t.Fatal("ordinary application tables must remain droppable")
	}
}

func TestMaintenanceCapabilityMatrix(t *testing.T) {
	catalog := tableMaintenanceCapabilities("system", "user")
	if catalog.canDrop || catalog.canTruncate || catalog.canRename || catalog.canAlter || catalog.canChangeEngine {
		t.Fatalf("system catalog has destructive capability: %+v", catalog)
	}

	telemetry := tableMaintenanceCapabilities("system_statistic", "logs")
	if telemetry.canDrop || !telemetry.canTruncate || telemetry.canRename || telemetry.canAlter || telemetry.canChangeEngine {
		t.Fatalf("telemetry capability matrix is invalid: %+v", telemetry)
	}

	user := tableMaintenanceCapabilities("application", "logs")
	if !user.canDrop || !user.canTruncate || !user.canRename || !user.canAlter || !user.canChangeEngine {
		t.Fatalf("user table capability matrix is unexpectedly restricted: %+v", user)
	}
}

func TestDropRefusesInternalObjectsBeforeMutation(t *testing.T) {
	oldSystem := databases.Get("system")
	db := newDatabase()
	db.Name = "system"
	db.tables = NonLockingReadMap.NewReadMap[string, *table]()
	db.tables.Set("user", &table{schema: db, Name: "user", PersistencyMode: Safe})
	databases.Set(db.Name, db)
	t.Cleanup(func() {
		databases.Remove("system")
		if oldSystem != nil {
			databases.Set(oldSystem.Name, oldSystem)
		}
	})

	requirePanic(t, func() { DropTable("system", "user", false) })
	if db.tables.Get("user") == nil {
		t.Fatal("protected table was removed")
	}
	requirePanic(t, func() { RenameTable("system", "user", "renamed_user") })
	if db.tables.Get("user") == nil || db.tables.Get("renamed_user") != nil {
		t.Fatal("protected table was renamed")
	}
	requirePanic(t, func() { db.tables.Get("user").DropColumn("admin") })

	requirePanic(t, func() { DropDatabase("system", false) })
	if databases.Get("system") != db {
		t.Fatal("protected database was removed")
	}
}

func TestPublishedFoldedTableNames(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tfoldednames", 0)
	db := tbl.schema
	before := db.foldedTableNames.Load()
	created, _ := CreateTable(db.Name, "MixedName", Safe, false)
	if db.GetTable("mIXEDnAME", true) != created {
		t.Fatal("shared table lookup did not resolve its folded spelling")
	}
	if db.GetTable("mixedname", false) != nil {
		t.Fatal("folded lookup changed exact-case namespace semantics")
	}
	if _, exists := (*before)["mixedname"]; exists {
		t.Fatal("namespace publication mutated a retained snapshot")
	}
	RenameTable(db.Name, "MixedName", "Renamed")
	if db.GetTable("MIXEDNAME", true) != nil {
		t.Fatal("rename retained an obsolete name")
	}
	if db.GetTable("renamed", true) != created || *created.publishedName.Load() != "Renamed" {
		t.Fatal("rename did not publish its new spelling")
	}
	CreateTable(db.Name, "RENAMED", Safe, false)
	requirePanic(t, func() { db.GetTable("Renamed", true) })
	if db.GetTable("Renamed", false) != created {
		t.Fatal("ambiguous folded names affected an exact-case caller")
	}
	DropTable(db.Name, "RENAMED", false)
	if db.GetTable("RENAMED", true) != created {
		t.Fatal("DROP did not remove the folded-name ambiguity")
	}
	DropTable(db.Name, "Renamed", false)
	if db.GetTable("renamed", true) != nil {
		t.Fatal("DROP retained a removed table")
	}
	unicodeTable, _ := CreateTable(db.Name, "Key", Safe, false)
	if db.GetTable("KEY", true) != unicodeTable {
		t.Fatal("folded namespace does not match the existing Unicode identifier policy")
	}
}

func TestPublishedDatabaseIdentifierPolicy(t *testing.T) {
	first, second := newDatabase(), newDatabase()
	first.Name, second.Name = "CaseCatalog", "CASECATALOG"
	databases.Set(first.Name, first)
	defer databases.Remove(first.Name)
	if GetDatabase("casecatalog", true) != first || GetDatabase("casecatalog", false) != nil {
		t.Fatal("database comparison policy was not respected")
	}
	before := databases.folded.Load()
	databases.Set(second.Name, second)
	defer databases.Remove(second.Name)
	requirePanic(t, func() { GetDatabase("CaseCatalog", true) })
	if GetDatabase("CaseCatalog", false) != first {
		t.Fatal("folded ambiguity changed the exact catalog")
	}
	databases.Remove(second.Name)
	if GetDatabase("CASECATALOG", true) != first {
		t.Fatal("database removal did not remove folded ambiguity")
	}
	if (*before)["casecatalog"] != first.Name {
		t.Fatal("database publication mutated an existing snapshot")
	}
}

func TestPersistedTableCallABIUpgradePreservesData(t *testing.T) {
	legacy := scm.NewSlice([]scm.Scmer{scm.NewSymbol("table"),
		scm.NewString("memcp-tests"), scm.NewString("items")})
	quoted := scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), legacy})
	if upgradePersistedTableCalls(quoted) != quoted {
		t.Fatal("ABI upgrade rewrote a quoted data value")
	}
	original := scm.NewProcStruct(scm.Proc{Params: scm.NewSlice(nil), Body: legacy, En: &scm.Globalenv})
	restored := upgradePersistedTableCalls(original)
	body := restored.Proc().Body.Slice()
	if len(body) != 5 || body[2].Bool() || body[4].Bool() || scm.String(body[3]) != "items" {
		t.Fatal("stored executable table reference did not migrate to explicit policies")
	}
	if len(original.Proc().Body.Slice()) != 3 {
		t.Fatal("ABI upgrade mutated the source definition")
	}
	if upgradePersistedTableCalls(restored) != restored {
		t.Fatal("ABI upgrade did not preserve a current definition")
	}
	shadowed := scm.NewProcStruct(scm.Proc{Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("table")}),
		Body: legacy, En: &scm.Globalenv})
	if upgradePersistedTableCalls(shadowed) != shadowed {
		t.Fatal("ABI upgrade rewrote an invocation of a lexical parameter")
	}

	definition := &table{Name: "items", OnInit: &original,
		Triggers: []TriggerDescription{{Language: "scheme", Source: "(lambda () (table \"memcp-tests\" \"items\"))"}}}
	invalidatePersistedPlannerCodeAfterLoad(definition)
	if len(definition.OnInit.Proc().Body.Slice()) != 5 {
		t.Fatal("load did not upgrade the existing initialization callback")
	}
	source := scm.Read("upgraded trigger", definition.Triggers[0].Source).WithoutSourceInfo().Slice()
	if len(source[2].WithoutSourceInfo().Slice()) != 5 {
		t.Fatal("load did not upgrade the authoritative Scheme trigger source")
	}
}
