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
	db.tables = NonLockingReadMap.New[table, string]()
	db.tables.Set(&table{schema: db, Name: "user", PersistencyMode: Safe})
	databases.Set(db)
	t.Cleanup(func() {
		databases.Remove("system")
		if oldSystem != nil {
			databases.Set(oldSystem)
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
