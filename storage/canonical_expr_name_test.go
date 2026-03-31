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

import (
	"testing"

	"github.com/launix-de/memcp/scm"
)

func TestBuildCanonicalAliasMapAcceptsFlatAssocList(t *testing.T) {
	aliasMap := buildCanonicalAliasMap(scm.NewSlice([]scm.Scmer{
		scm.NewString("Ticket"), scm.NewString("test.ticket"),
		scm.NewString("User"), scm.NewString("system.user"),
	}))

	if got := aliasMap["ticket"]; got != "test.ticket" {
		t.Fatalf("aliasMap[ticket] = %q, want %q", got, "test.ticket")
	}
	if got := aliasMap["user"]; got != "system.user" {
		t.Fatalf("aliasMap[user] = %q, want %q", got, "system.user")
	}
}

func TestBuildCanonicalAliasMapAcceptsFastDict(t *testing.T) {
	fd := scm.NewFastDictValue(2)
	fd.Set(scm.NewString("Ticket"), scm.NewString("test.ticket"), nil)
	fd.Set(scm.NewString("User"), scm.NewString("system.user"), nil)

	aliasMap := buildCanonicalAliasMap(scm.NewFastDict(fd))

	if got := aliasMap["ticket"]; got != "test.ticket" {
		t.Fatalf("aliasMap[ticket] = %q, want %q", got, "test.ticket")
	}
	if got := aliasMap["user"]; got != "system.user" {
		t.Fatalf("aliasMap[user] = %q, want %q", got, "system.user")
	}
}

func TestBuildCanonicalAliasMapAcceptsNestedPairList(t *testing.T) {
	aliasMap := buildCanonicalAliasMap(scm.NewSlice([]scm.Scmer{
		scm.NewSlice([]scm.Scmer{scm.NewString("Ticket"), scm.NewString("test.ticket")}),
		scm.NewSlice([]scm.Scmer{scm.NewString("User"), scm.NewString("system.user")}),
	}))

	if got := aliasMap["ticket"]; got != "test.ticket" {
		t.Fatalf("aliasMap[ticket] = %q, want %q", got, "test.ticket")
	}
	if got := aliasMap["user"]; got != "system.user" {
		t.Fatalf("aliasMap[user] = %q, want %q", got, "system.user")
	}
}

func TestCanonicalizeScmerToStringUsesAssocAliasMap(t *testing.T) {
	aliasMap := buildCanonicalAliasMap(scm.NewSlice([]scm.Scmer{
		scm.NewString("Ticket"), scm.NewString("test.ticket"),
	}))
	expr := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("get_column"),
		scm.NewString("Ticket"),
		scm.NewBool(false),
		scm.NewString("ID"),
		scm.NewBool(false),
	})

	got := canonicalizeScmerToString(expr, nil, nil, aliasMap)
	want := "(get_column \"test.ticket\" false \"ID\" false)"
	if got != want {
		t.Fatalf("canonicalizeScmerToString() = %q, want %q", got, want)
	}
}
