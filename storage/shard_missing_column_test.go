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

func TestGetColumnStorageOrPanicExAddsSchemaColumnWhenLocked(t *testing.T) {
	db := &database{Name: "system"}
	tbl := &table{
		schema:          db,
		Name:            "access",
		PersistencyMode: Memory,
		Columns: []*column{
			{Name: "username"},
			{Name: "database"},
		},
	}
	shard := &storageShard{
		t:            tbl,
		columns:      map[string]ColumnStorage{"username": new(StorageSparse)},
		deltaColumns: make(map[string]int),
		writeOwners:  make(map[uint64]uint32),
	}

	shard.mu.Lock()
	defer shard.mu.Unlock()

	col := shard.getColumnStorageOrPanicEx("database", true)
	if col == nil {
		t.Fatal("expected sparse storage for schema column missing from shard map")
	}
	if _, ok := col.(*StorageSparse); !ok {
		t.Fatalf("expected *StorageSparse, got %T", col)
	}
	if _, ok := shard.columns["database"]; !ok {
		t.Fatal("expected column to be inserted into shard map")
	}
}
