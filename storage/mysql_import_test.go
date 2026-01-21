/*
Copyright (C) 2023, 2024  Carl-Philip Hänsch

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

import "database/sql"
import "testing"
import "time"

import "github.com/launix-de/memcp/scm"

func TestMySQLMapType(t *testing.T) {
	typ, dims := mysqlMapType("decimal", "decimal(10,2)", sql.NullInt64{}, sql.NullInt64{Int64: 10, Valid: true}, sql.NullInt64{Int64: 2, Valid: true})
	if typ != "decimal" || len(dims) != 2 || dims[0] != 10 || dims[1] != 2 {
		t.Fatalf("unexpected mapping: %q %v", typ, dims)
	}

	typ, dims = mysqlMapType("varchar", "varchar(255)", sql.NullInt64{Int64: 255, Valid: true}, sql.NullInt64{}, sql.NullInt64{})
	if typ != "varchar" || len(dims) != 1 || dims[0] != 255 {
		t.Fatalf("unexpected mapping: %q %v", typ, dims)
	}
}

func TestMySQLToScmer(t *testing.T) {
	if !mysqlToScmer(nil).IsNil() {
		t.Fatal("nil should map to scm nil")
	}
	if got := mysqlToScmer(int64(7)); scm.ToInt(got) != 7 {
		t.Fatalf("int64: got %v", got)
	}
	now := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if got := scm.String(mysqlToScmer(now)); got != "2020-01-02 03:04:05" {
		t.Fatalf("time: got %q", got)
	}
}
