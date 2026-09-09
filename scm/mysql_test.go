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
	"time"

	querypb "github.com/launix-de/go-mysqlstack/sqlparser/depends/query"
)

func TestMySQLInitializationGate(t *testing.T) {
	BeginMySQLInitialization()
	ready := waitForMySQLInitialization()
	select {
	case <-ready:
		t.Fatal("MySQL initialization gate opened before bootstrap completed")
	default:
	}

	CompleteMySQLInitialization()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("MySQL initialization gate stayed closed after bootstrap completed")
	}
}

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
	var fields []*querypb.Field
	colmap := map[string]int{}

	row, unknown := prepareMySQLResultRow(&fields, colmap, []Scmer{
		NewString("x"), NewInt(1),
		NewString("x"), NewString("EUR"),
	}, nil, false)

	if unknown {
		t.Fatal("first row unexpectedly reported an unknown column")
	}
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	if fields[0].Type != querypb.Type_VARCHAR {
		t.Fatalf("expected varchar metadata, got %v", fields[0].Type)
	}
	if fields[0].Charset != 45 {
		t.Fatalf("expected utf8mb4 charset, got %d", fields[0].Charset)
	}
	if got := row[0].String(); got != "EUR" {
		t.Fatalf("expected last duplicate value, got %q", got)
	}
}

func TestPrepareMySQLResultRowPadsMissingAndRejectsNewPublishedColumns(t *testing.T) {
	fields := []*querypb.Field{{Name: "a", Type: querypb.Type_INT64}, {Name: "b", Type: querypb.Type_VARCHAR}}
	colmap := map[string]int{"a": 0, "b": 1}
	row := []Scmer{NewInt(1), NewString("old")}

	row, unknown := prepareMySQLResultRow(&fields, colmap, []Scmer{
		NewString("b"), NewString("new"),
		NewString("c"), NewString("ignored"),
	}, row, true)

	if !unknown {
		t.Fatal("new column after publishing fields was not reported")
	}
	if !row[0].IsNil() {
		t.Fatalf("missing column was not padded with NULL: %v", row[0])
	}
	if got := row[1].String(); got != "new" {
		t.Fatalf("known column got %q, want new", got)
	}
}

func TestPrepareMySQLResultRowUsesCompilerTypeAcrossNullRows(t *testing.T) {
	fields, colmap, row := prepareMySQLResultFields([]Scmer{
		NewSlice([]Scmer{NewString("value"), NewString("BIGINT"), NewString("bin")}),
	})
	for _, value := range []Scmer{NewNil(), NewFloat(42)} {
		row, _ = prepareMySQLResultRow(&fields, colmap, []Scmer{NewString("value"), value}, row, true)
		if fields[0].Type != querypb.Type_INT64 {
			t.Fatalf("declared integer metadata changed: %v", fields[0].Type)
		}
	}
}

func TestSQLResultFieldPackingUsesDeclaredType(t *testing.T) {
	for _, test := range []struct {
		typ         string
		input, want Scmer
	}{
		{"BIGINT", NewFloat(42), NewInt(42)},
		{"VARCHAR", NewInt(42), NewString("42")},
		{"DECIMAL", NewFloat(12.5), NewString("12.5")},
		{"DOUBLE", NewInt(42), NewFloat(42)},
		{"BIGINT", NewNil(), NewNil()},
		{"VARBINARY", NewString("a\x00b"), NewString("a\x00b")},
	} {
		got := (SQLResultField{Type: test.typ}).Value(test.input)
		if got.GetTag() != test.want.GetTag() || got.String() != test.want.String() {
			t.Fatalf("%s packed %v as %v, want %v", test.typ, test.input, got, test.want)
		}
	}
}

func BenchmarkPrepareMySQLResultRow10PublishedColumns(b *testing.B) {
	fields := make([]*querypb.Field, 10)
	colmap := make(map[string]int, 10)
	item := make([]Scmer, 0, 20)
	row := make([]Scmer, 10)
	for i := 0; i < 10; i++ {
		name := string(rune('a' + i))
		fields[i] = &querypb.Field{Name: name, Type: querypb.Type_INT64}
		colmap[name] = i
		item = append(item, NewString(name), NewInt(int64(i)))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		row, _ = prepareMySQLResultRow(&fields, colmap, item, row, true)
	}
}

func TestMySQLServerVersionHasClientCompatiblePrefix(t *testing.T) {
	if got := (&MySQLWrapper{}).ServerVersion(); got != "5.7.44-MemCP" {
		t.Fatalf("unexpected MySQL protocol version %q", got)
	}
}

func TestPrepareMySQLResultFieldsPreservesCompilerOrderAndDuplicates(t *testing.T) {
	fields, colmap, row := prepareMySQLResultFields([]Scmer{
		NewSlice([]Scmer{NewString("id"), NewString("INT"), NewString("bin")}),
		NewSlice([]Scmer{NewString("value"), NewString("VARCHAR"), NewString("utf8mb4_unicode_520_ci")}),
		NewSlice([]Scmer{NewString("value"), NewString("DOUBLE"), NewString("bin")}),
	})
	if len(fields) != 3 || len(row) != 3 {
		t.Fatalf("prepared %d fields and %d row slots, want 3 each", len(fields), len(row))
	}
	if fields[0].Name != "id" || fields[1].Name != "value" || fields[2].Name != "value" {
		t.Fatalf("compiler field order was not preserved: %+v", fields)
	}
	if fields[0].Type != querypb.Type_INT64 || fields[1].Type != querypb.Type_VARCHAR || fields[2].Type != querypb.Type_FLOAT64 || fields[1].Charset != 246 {
		t.Fatalf("compiler types/collation not preserved: %+v", fields)
	}
	if colmap["id"] != 0 || colmap["value"] != 2 {
		t.Fatalf("unexpected fallback column map: %+v", colmap)
	}
}
