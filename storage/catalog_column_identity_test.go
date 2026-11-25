/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "bytes"
import "strings"
import "testing"
import "encoding/json"
import "encoding/base64"

func TestIdentityAppendRequiredJournalAndHeader(t *testing.T) {
	const identity = "11111111-1111-4111-8111-111111111111"
	const epoch = "22222222-2222-4222-8222-222222222222"
	makeRecord := func() catalogDecisionRecord {
		return catalogDecisionRecord{Format: 31, Kind: "prepare", Database: "identity_state", Incarnation: identity,
			Transaction: identity + "/1", Decision: identity, Installation: identity, Boot: identity, Version: 1, Distributed: true,
			ColumnAppends: []catalogColumnAppend{{Table: "identity_rows", Identity: identity, Ordinal: 1, Name: "id",
				Type: storedColumnType{Type: "INT", Collation: "utf8mb4"}, GeneratedIDs: &columnIDGeneration{1, 5}}},
			AllocatorResets: []catalogAllocatorResetDeclaration{{Table: "identity_rows", Column: "id", State: tableAllocatorState{
				Version: 2, Table: identity, StartKnown: true, Start: 1, IssuedKnown: true, High: 0,
				Epoch: epoch, Current: 5, Next: 5, HasNext: true}}}}
	}
	record := makeRecord()
	decoded, err := decodeCatalogDecision(encodeCatalogDecision(&record))
	if err != nil || decoded == nil || !decoded.ColumnAppends[0].column().AutoIncrement || decoded.ColumnAppends[0].column().generatedIDs.Start != 5 {
		t.Fatalf("required identity append lost native generation: %v", err)
	}
	if decoded.AllocatorResets[0].State.Start != 1 || decoded.AllocatorResets[0].State.Issued {
		t.Fatal("column seed replaced actual table allocator provenance")
	}
	for _, version := range []int{16, 19, 24, 25, 26, 28, 29, 30} {
		record := makeRecord()
		record.Format = version
		if _, err := decodeCatalogDecision(encodeCatalogDecision(&record)); err == nil {
			t.Errorf("historical V%d accepted generated-ID append", version)
		}
	}
	legacyAppend := makeRecord()
	legacyAppend.Format, legacyAppend.AllocatorResets = 16, nil
	legacyAppend.ColumnAppends[0].GeneratedIDs = nil
	legacyAppend.ColumnAppends[0].Type.Nullable = true
	if _, err := decodeCatalogDecision(encodeCatalogDecision(&legacyAppend)); err != nil {
		t.Fatalf("ordinary legacy append stopped decoding: %v", err)
	}
	raw, _ := json.Marshal(&legacyAppend)
	withNull := strings.Replace(string(raw), `"column_appends":[{`, `"column_appends":[{"generated_ids":null,`, 1)
	if withNull == string(raw) {
		t.Fatal("explicit-null fixture did not address its append")
	}
	if _, err := decodeCatalogDecision(catalogColumnAppendJournalPrefix + base64.RawURLEncoding.EncodeToString([]byte(withNull))); err == nil {
		t.Fatal("legacy append accepted an explicit generated-ID null field")
	}
	for _, tc := range []struct {
		name string
		edit func(*catalogDecisionRecord)
	}{
		{"missing_epoch", func(r *catalogDecisionRecord) { r.AllocatorResets = nil }},
		{"wrong_column", func(r *catalogDecisionRecord) { r.AllocatorResets[0].Column = "different" }},
		{"wrong_generation", func(r *catalogDecisionRecord) { r.AllocatorResets[0].State.Table = epoch }},
		{"nullable", func(r *catalogDecisionRecord) { r.ColumnAppends[0].Type.Nullable = true }},
		{"unknown_origin", func(r *catalogDecisionRecord) { r.ColumnAppends[0].GeneratedIDs.Version = 2 }},
		{"negative_origin", func(r *catalogDecisionRecord) { r.ColumnAppends[0].GeneratedIDs.Start = -1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := makeRecord()
			tc.edit(&record)
			if _, err := decodeCatalogDecision(encodeCatalogDecision(&record)); err == nil {
				t.Fatal("invalid generated-ID authority accepted")
			}
		})
	}
	// A checkpoint retains the required declaration discriminator after V31 WAL
	// retires. An old name-binding reader rejects the new required field.
	c := decoded.ColumnAppends[0].column()
	header, err := json.Marshal(struct {
		Name          any
		Typ           string
		AutoIncrement bool
	}{c.persistedName(), "INT", true})
	if err != nil {
		t.Fatal(err)
	}
	var restored column
	if err := json.Unmarshal(header, &restored); err != nil || restored.generatedIDs == nil || restored.generatedIDs.Start != 5 || !restored.AutoIncrement {
		t.Fatalf("checkpoint lost generated identity: %v", err)
	}
	var legacy struct {
		Version int    `json:"version"`
		Storage string `json:"storage"`
		Visible string `json:"visible"`
	}
	encodedName, _ := json.Marshal(c.persistedName())
	oldReader := json.NewDecoder(bytes.NewReader(encodedName))
	oldReader.DisallowUnknownFields()
	if err := oldReader.Decode(&legacy); err == nil {
		t.Fatal("legacy name binding silently ignored configured identity origin")
	}
	unknown := strings.Replace(string(header), `"version":2`, `"version":3`, 1)
	if err := json.Unmarshal([]byte(unknown), new(column)); err == nil {
		t.Fatal("unknown required checkpoint generation accepted")
	}
}

func TestIdentityAppendPredecessorSelection(t *testing.T) {
	const identity = "33333333-3333-4333-8333-333333333333"
	unused := tableAllocatorState{Version: 1, Table: identity, StartKnown: true, Start: 1, IssuedKnown: true}
	for _, tc := range []struct {
		name                    string
		state                   tableAllocatorState
		stale, existing, reject bool
	}{
		{name: "unused", state: unused},
		{name: "unknown", state: tableAllocatorState{Version: 1, Table: identity}, reject: true},
		{name: "issued", state: tableAllocatorState{Version: 1, Table: identity, StartKnown: true, Start: 1, IssuedKnown: true, Issued: true, High: 3, HighAllocated: true}, reject: true},
		{name: "stale", state: unused, stale: true, reject: true},
		{name: "existing_identity", state: unused, existing: true, reject: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := &table{schema: &database{Name: "identity_predecessor_unit"}, RuntimeIdentity: identity}
			target.allocator.Store(&tc.state)
			before := target.allocator.Load()
			token := tableAllocatorToken(tc.state)
			if tc.stale {
				token = "stale"
			}
			initializer := &nativeColumnInitializer{operation: &nativeMutationOperation{}, target: target, column: &column{Name: "generated"}, ids: &nativeColumnIDInitialization{token: token, start: 7, maximum: 100}}
			var columns []*column
			if tc.existing {
				columns = []*column{{Name: "already", AutoIncrement: true}}
			}
			var failure any
			func() { defer func() { failure = recover() }(); initializer.prepareIDBranch(columns) }()
			if (failure != nil) != tc.reject {
				t.Fatalf("predecessor decision rejected=%v, want %v: %v", failure != nil, tc.reject, failure)
			}
			if target.allocator.Load() != before {
				t.Fatal("private identity selection changed the public allocator")
			}
			if tc.reject {
				if initializer.allocator != nil || initializer.column.generatedIDs != nil {
					t.Fatal("rejected predecessor acquired a header or allocator branch")
				}
				return
			}
			selected := initializer.allocator.state.Load()
			if selected.Start != 1 || selected.Issued || selected.Current != 7 || selected.Next != 7 || !selected.HasNext || initializer.column.generatedIDs.Start != 7 {
				t.Fatal("private branch replaced actual provenance with the column seed")
			}
			reservation := initializer.allocator.reserve(tableIDExplicit{}, 3)
			selected = initializer.allocator.state.Load()
			if reservation.First != 7 || reservation.Last != 9 || selected.Current != 9 || selected.Next != 10 || !selected.Issued || selected.Start != 1 {
				t.Fatal("historical range did not use the selected typed native cursor")
			}
		})
	}
}
