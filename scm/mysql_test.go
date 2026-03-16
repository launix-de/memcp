/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

package scm

import (
	"testing"

	querypb "github.com/launix-de/go-mysqlstack/sqlparser/depends/query"
	"github.com/launix-de/go-mysqlstack/sqlparser/depends/sqltypes"
)

func TestAppendMySQLResultRowDuplicateAliasUsesLastValueType(t *testing.T) {
	result := sqltypes.Result{}
	colmap := map[string]int{}

	row := appendMySQLResultRow(&result, colmap, []Scmer{
		NewString("x"), NewInt(1),
		NewString("x"), NewString("EUR"),
	})

	if len(result.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(result.Fields))
	}
	if result.Fields[0].Type != querypb.Type_VARCHAR {
		t.Fatalf("expected varchar metadata, got %v", result.Fields[0].Type)
	}
	if result.Fields[0].Charset != 45 {
		t.Fatalf("expected utf8mb4 charset, got %d", result.Fields[0].Charset)
	}
	if got := row[0].ToString(); got != "EUR" {
		t.Fatalf("expected last duplicate value, got %q", got)
	}
}
