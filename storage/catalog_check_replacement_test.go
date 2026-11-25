/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestCheckReplacementJournalDomains(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	makeRecord := func() catalogDecisionRecord {
		return catalogDecisionRecord{Format: 30, Kind: "prepare", Database: "check_state", Incarnation: id,
			Transaction: id + "/1", Decision: id, Installation: id, Boot: id, Version: 1, Distributed: true,
			CheckDrops: []catalogCheckDeclaration{{Table: "check_rows", Identity: id,
				Check: &checkDeclaration{Id: "CK_value", Language: "tsql-check", Source: "v >= 0", Columns: []string{"v"}}}},
			Checks: []catalogCheckDeclaration{{Table: "check_rows", Identity: id,
				Check: &checkDeclaration{Id: "CK_value", Language: "tsql-check", Source: "v > 0", Columns: []string{"v"}}}},
		}
	}
	record := makeRecord()
	decoded, err := decodeCatalogDecision(encodeCatalogDecision(&record))
	if err != nil || decoded == nil || decoded.CheckDrops[0].Check.Source != "v >= 0" || decoded.Checks[0].Check.Source != "v > 0" {
		t.Fatalf("V30 did not retain exact old/new CHECK recipes: %v", err)
	}
	for _, version := range []int{8, 11, 15, 26, 28, 29} {
		record := makeRecord()
		record.Format = version
		if _, err := decodeCatalogDecision(encodeCatalogDecision(&record)); err == nil {
			t.Errorf("historical V%d acquired duplicate CHECK identity semantics", version)
		}
	}
	for _, tc := range []struct {
		name string
		edit func(*catalogDecisionRecord)
	}{
		{"endpoint", func(r *catalogDecisionRecord) { r.Checks[0].Table = "different_rows" }},
		{"generation", func(r *catalogDecisionRecord) { r.Checks[0].Identity = "22222222-2222-4222-8222-222222222222" }},
		{"native_identity", func(r *catalogDecisionRecord) { r.Checks[0].Check.Id = "ck_value" }},
		{"repeated_retirement", func(r *catalogDecisionRecord) { r.CheckDrops = append(r.CheckDrops, r.CheckDrops[0]) }},
		{"repeated_addition", func(r *catalogDecisionRecord) { r.Checks = append(r.Checks, r.Checks[0]) }},
		{"missing_pair", func(r *catalogDecisionRecord) { r.CheckDrops = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := makeRecord()
			tc.edit(&record)
			if _, err := decodeCatalogDecision(encodeCatalogDecision(&record)); err == nil {
				t.Fatal("malformed CHECK replacement accepted")
			}
		})
	}
	// Existing column/enforcement payloads keep their own invariants when the
	// CHECK pair selects the newer required family. No field is ignored.
	record = makeRecord()
	record.ColumnDeclarations = []catalogColumnDeclaration{{Table: "typed_rows", Identity: id, Column: "code",
		Before: storedColumnType{Type: "VARCHAR", Dimensions: []int{2}, Nullable: true},
		After:  storedColumnType{Type: "VARCHAR", Dimensions: []int{4}, Nullable: true}}}
	record.NativeDeclarations = []catalogNativeDeclaration{{Operation: "foreign_add", Foreign: &catalogForeignDeclaration{
		ChildIdentity: id, ParentIdentity: "22222222-2222-4222-8222-222222222222",
		Foreign: foreignKey{Id: "FK_disabled", Tbl1: "child_rows", Cols1: []string{"parent_id"}, Tbl2: "parent_rows", Cols2: []string{"id"}, Disabled: true, Untrusted: true},
	}}}
	if decoded, err := decodeCatalogDecision(encodeCatalogDecision(&record)); err != nil || decoded == nil || !decoded.NativeDeclarations[0].Foreign.Foreign.Disabled || len(decoded.ColumnDeclarations) != 1 {
		t.Fatalf("mixed V30 column/enforcement decision rejected: %v", err)
	}
	data, err := json.Marshal(&record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeCatalogDecision("catalog-v10/v31/" + base64.RawURLEncoding.EncodeToString(data)); err == nil {
		t.Fatal("unknown required CHECK family accepted")
	}
	unknown := strings.Replace(string(data), `"format":30`, `"format":31`, 1)
	if _, err := decodeCatalogDecision(catalogCheckReplacementJournalPrefix + base64.RawURLEncoding.EncodeToString([]byte(unknown))); err == nil {
		t.Fatal("unknown version in CHECK family accepted")
	}
}
