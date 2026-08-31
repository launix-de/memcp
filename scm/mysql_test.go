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
	"strings"
	"testing"

	querypb "github.com/launix-de/go-mysqlstack/sqlparser/depends/query"
	"github.com/launix-de/go-mysqlstack/sqlparser/depends/sqltypes"
)

func TestMySQLClientErrorMessageDropsInternalStackAndFitsCommandBuffer(t *testing.T) {
	stack := strings.Repeat("runtime stack frame\n", 1000)
	got := mysqlClientErrorMessage("NthLocalVar(2) out of range (len=0)\n" + stack)
	if got != "NthLocalVar(2) out of range (len=0)" {
		t.Fatalf("unexpected client error %q", got)
	}
	if len(got) > mysqlClientErrorMessageLimit {
		t.Fatalf("client error uses %d bytes, limit is %d", len(got), mysqlClientErrorMessageLimit)
	}
}

func TestMySQLClientErrorMessageBoundsSingleLineErrors(t *testing.T) {
	got := mysqlClientErrorMessage(strings.Repeat("x", mysqlClientErrorMessageLimit+100))
	if len(got) != mysqlClientErrorMessageLimit {
		t.Fatalf("client error uses %d bytes, want %d", len(got), mysqlClientErrorMessageLimit)
	}
}

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

func TestMySQLServerVersionHasClientCompatiblePrefix(t *testing.T) {
	if got := (&MySQLWrapper{}).ServerVersion(); got != "5.7.44-MemCP" {
		t.Fatalf("unexpected MySQL protocol version %q", got)
	}
}

func TestIsSelectQueryHandlesCommentsAndCase(t *testing.T) {
	for _, query := range []string{
		"SELECT 1",
		" select 1",
		"/* client */ SELECT 1",
		"-- client\nSELECT 1",
		"# client\nSeLeCt 1",
	} {
		if !isSelectQuery(query) {
			t.Fatalf("expected SELECT query: %q", query)
		}
	}
	for _, query := range []string{"UPDATE t SET a=1", "/* unfinished", "SELECTED"} {
		if isSelectQuery(query) {
			t.Fatalf("unexpected SELECT query: %q", query)
		}
	}
}
