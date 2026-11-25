/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "strings"
import "testing"
import "encoding/json"
import "encoding/base64"
import "github.com/google/uuid"

func TestNullColumnInitializationJournalDomains(t *testing.T) {
	const identity = "11111111-1111-4111-8111-111111111111"
	const generation = "22222222-2222-4222-8222-222222222222"
	record := catalogDecisionRecord{Format: 19, Kind: "prepare", Database: "null_prefix_state", Incarnation: identity,
		Transaction: identity + "/1", Decision: identity, Installation: identity, Boot: identity, Version: 1, Distributed: true,
		ColumnAppends: []catalogColumnAppend{{Table: "prefix_rows", Identity: identity, Ordinal: 1, Name: "new_value",
			Type:           storedColumnType{Type: "INT", Nullable: true, Collation: "utf8mb4"},
			Initialization: &catalogColumnInitialization{Extents: map[string]columnInitialExtent{generation: {3, 2}}}}}}
	encoded := encodeCatalogDecision(&record)
	decoded, err := decodeCatalogDecision(encoded)
	if err != nil || decoded == nil || !strings.HasPrefix(encoded, catalogColumnInitializationJournalPrefix) {
		t.Fatalf("existing V19 NULL-prefix domain did not round trip: %v", err)
	}
	c := decoded.ColumnAppends[0].column()
	if !c.AllowNull || c.hasDefault() || c.DefaultName != "" || !c.InitialValue.IsNil() || c.InitialValues[generation] != (columnInitialExtent{3, 2}) {
		t.Fatal("NULL prefix invented a future DEFAULT or lost its actual generation extents")
	}
	// NULL is an explicit durable scalar, not absence of the required V19
	// field. Historical V16 never acquires this input by inference.
	record.Format = 16
	raw, err := json.Marshal(&record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeCatalogDecision(catalogColumnAppendJournalPrefix + base64.RawURLEncoding.EncodeToString(raw)); err == nil {
		t.Fatal("V16 accepted the required initialized prefix field")
	}
	record.ColumnAppends[0].Initialization = nil
	legacyRaw, err := json.Marshal(&record)
	if err != nil {
		t.Fatal(err)
	}
	explicitNull := strings.Replace(string(legacyRaw), `"column_appends":[{`, `"column_appends":[{"initialization":null,`, 1)
	if explicitNull == string(legacyRaw) {
		t.Fatal("explicit-null fixture did not address its append")
	}
	if _, err := decodeCatalogDecision(catalogColumnAppendJournalPrefix + base64.RawURLEncoding.EncodeToString([]byte(explicitNull))); err == nil {
		t.Fatal("V16 accepted an explicit-null initialization field")
	}
	legacy, err := decodeCatalogDecision(encodeCatalogDecision(&record))
	if err != nil || legacy == nil || legacy.ColumnAppends[0].Initialization != nil {
		t.Fatalf("legacy V16 read-only append stopped decoding: %v", err)
	}
}

func TestNullColumnInitializationRetainsImmutableProducerInput(t *testing.T) {
	const generation = "22222222-2222-4222-8222-222222222222"
	declaration := catalogColumnAppend{Name: "new_value", Type: storedColumnType{Type: "INT", Nullable: true, Collation: "utf8mb4"},
		Initialization: &catalogColumnInitialization{Extents: map[string]columnInitialExtent{generation: {3, 2}}}}
	base := &column{Name: "id", Typ: "INT", Collation: "utf8mb4"}
	target := &table{Name: "immutable_prefix"}
	old := target.buildColumnNamesSnapshot([]*column{base}, nil)
	added := declaration.column()
	selected := target.buildColumnNamesSnapshot([]*column{base, added}, old)
	if len(old.declarations) != 1 || len(old.exact) != 1 || old.initialByShard != nil || old.appendedInitialInput != nil {
		t.Fatal("private ADD changed a retained predecessor input")
	}
	if _, found := old.exact["new_value"]; found || selected.exact["new_value"].ordinal != 1 {
		t.Fatal("private ADD changed old binding or lost its new ordinal")
	}
	input := selected.initialByShard[uuid.MustParse(generation)]
	if input == nil || !input.fields["new_value"].main.GetValue(0).IsNil() {
		t.Fatal("selected main input does not expose historical NULL")
	}
	for _, delta := range []int{0, 1} {
		value, exists := initialColumnValue(input, "new_value", delta)
		if !exists || !value.IsNil() {
			t.Fatalf("captured delta %d lost historical NULL", delta)
		}
	}
	if _, exists := initialColumnValue(input, "new_value", 2); exists {
		t.Fatal("NULL prefix replaced a future explicit row outside its captured extent")
	}
	// Durable recipes and selected headers own independent maps; later recipe
	// reconstruction cannot change a retained producer's native input.
	declaration.Initialization.Extents[generation] = columnInitialExtent{99, 99}
	if added.InitialValues[generation] != (columnInitialExtent{3, 2}) || input.fields["new_value"].DeltaRows != 2 {
		t.Fatal("retained prefix aliases the replay recipe")
	}
}
