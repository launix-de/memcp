/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"encoding/base64"
	"encoding/json"
	"github.com/launix-de/memcp/scm"
	"strings"
	"testing"
)

func TestLogicalIndexRequiredJournalDomain(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	index := logicalIndexDeclaration{Name: "IX_payload", Columns: []string{"id"}, Included: []string{"payload"}, Descending: []bool{true}, Placement: "nonclustered"}
	record := catalogDecisionRecord{Format: 32, Kind: "prepare", Database: "index_state", Incarnation: id, Transaction: id + "/1", Decision: id, Installation: id, Boot: id, Version: 1, Distributed: true, NativeDeclarations: []catalogNativeDeclaration{{Operation: "index_add", Index: &catalogIndexDeclaration{"items", id, index}}}}
	encoded := encodeCatalogDecision(&record)
	decoded, err := decodeCatalogDecision(encoded)
	if err != nil || decoded == nil || !strings.HasPrefix(encoded, catalogIndexDeclarationJournalPrefix) || !sameLogicalIndex(decoded.NativeDeclarations[0].Index.Index, index) {
		t.Fatalf("required index declaration roundtrip: %v", err)
	}
	for _, version := range []int{15, 23, 26, 28, 29, 30, 31} {
		legacy := record
		legacy.Format = version
		if _, err := decodeCatalogDecision(encodeCatalogDecision(&legacy)); err == nil {
			t.Errorf("V%d accepted index authority", version)
		}
	}
	for _, malformed := range []struct {
		name    string
		actions []catalogNativeDeclaration
	}{
		{"duplicate addition", []catalogNativeDeclaration{record.NativeDeclarations[0], record.NativeDeclarations[0]}},
		{"retirement mismatch", []catalogNativeDeclaration{record.NativeDeclarations[0], {Operation: "index_drop", Index: &catalogIndexDeclaration{"items", id, logicalIndexDeclaration{Name: "IX_payload", Columns: []string{"payload"}}}}}},
	} {
		trial := record
		trial.NativeDeclarations = malformed.actions
		if _, err := decodeCatalogDecision(encodeCatalogDecision(&trial)); err == nil {
			t.Errorf("accepted %s", malformed.name)
		}
	}
	rawUnknown, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	rawUnknown = append([]byte(`{"unknown_index_authority":true,`), rawUnknown[1:]...)
	if _, err := decodeCatalogDecision(catalogIndexDeclarationJournalPrefix + base64.RawURLEncoding.EncodeToString(rawUnknown)); err == nil {
		t.Fatal("required V32 reader ignored unknown top-level authority")
	}
	wrong := strings.Replace(encoded, catalogIndexDeclarationJournalPrefix, catalogIdentityAppendJournalPrefix, 1)
	if _, err := decodeCatalogDecision(wrong); err == nil {
		t.Fatal("new index authority accepted under old required family")
	}
	// Even an explicit NULL sidecar remains forbidden in historical formats.
	record.Format = 15
	record.NativeDeclarations = []catalogNativeDeclaration{{Operation: "key_add", Key: &catalogKeyDeclaration{"items", id, uniqueKey{Id: "UQ_old", Cols: []string{"id"}}}}}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	explicit := strings.Replace(string(raw), `"operation":"key_add"`, `"operation":"key_add","index":null`, 1)
	if explicit == string(raw) {
		t.Fatal("fixture failed to inject explicit sidecar")
	}
	if _, err := decodeCatalogDecision(encodeCatalogDecision(&record)); err != nil {
		t.Fatalf("historical key domain changed: %v", err)
	}
	if _, err := decodeCatalogDecision(catalogNativeDeclarationJournalPrefix + base64.RawURLEncoding.EncodeToString([]byte(explicit))); err == nil {
		t.Fatal("historical format accepted NULL index field")
	}
}

func TestLogicalIndexCheckpointRequiresTypedNameEnvelope(t *testing.T) {
	key := uniqueKey{Id: "UX_id", Cols: []string{"id"}, NullsNotDistinct: true}
	tbl := table{Name: "items", Columns: []*column{{Name: "id", Typ: "INT"}}, Unique: []uniqueKey{key}, LogicalIndexes: []logicalIndexDeclaration{{Name: "UX_id", Columns: []string{"id"}, Key: &key}}}
	encoded, err := json.Marshal(&tbl)
	if err != nil {
		t.Fatal(err)
	}
	var restored table
	if err := json.Unmarshal(encoded, &restored); err != nil || len(restored.LogicalIndexes) != 1 || !sameLogicalIndex(tbl.LogicalIndexes[0], restored.LogicalIndexes[0]) {
		t.Fatalf("index checkpoint roundtrip: %v", err)
	}
	// This is the old required-field contract, not an optional unknown member.
	var legacy struct{ Name string }
	if err := json.Unmarshal(encoded, &legacy); err == nil {
		t.Fatal("legacy string-name checkpoint reader accepted required index binding")
	}
	for _, replacement := range []string{`"version":1`, `"version":3`} {
		malformed := strings.Replace(string(encoded), `"version":2`, replacement, 1)
		if err := json.Unmarshal([]byte(malformed), &restored); err == nil {
			t.Fatalf("accepted incompatible Name envelope %s", replacement)
		}
	}
	malformed := strings.Replace(string(encoded), `"columns":["id"]`, `"columns":["missing"]`, 1)
	if err := json.Unmarshal([]byte(malformed), &restored); err == nil {
		t.Fatal("checkpoint accepted missing indexed column")
	}
	tbl.LogicalIndexes = nil
	legacyBytes, err := json.Marshal(&tbl)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(legacyBytes, &restored); err != nil || len(restored.LogicalIndexes) != 0 {
		t.Fatalf("historical missing index metadata stopped loading: %v", err)
	}
}

func TestLogicalIndexOriginalDecisionAndConstraintSafety(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tlogicalindex", 2)
	tx := NewTxContext(TxACID)
	defer func() {
		if tx.State == TxActive {
			tx.Rollback()
		}
	}()
	first := logicalIndexDeclaration{Name: "IX_payload", Columns: []string{"id"}, Included: []string{"payload"}}
	sp := tx.CreateSavepoint()
	if !mutateLogicalIndex(tbl, first, "add", tx) || len(tbl.LogicalIndexes) != 0 || len(tbl.logicalIndexesForTx(tx)) != 1 {
		t.Fatal("index addition leaked or failed original header selection")
	}
	requirePanic(t, func() { tx.dropPublicColumn(tbl, "payload", false, scm.NewNil()) })
	tx.RollbackToSavepoint(sp)
	if len(tbl.logicalIndexesForTx(tx)) != 0 || len(tbl.LogicalIndexes) != 0 {
		t.Fatal("index savepoint did not restore selected header")
	}
	if !mutateLogicalIndex(tbl, first, "add", tx) {
		t.Fatal("index could not be added after rewind")
	}
	if !mutateLogicalIndex(tbl, logicalIndexDeclaration{Name: first.Name}, "drop", tx) {
		t.Fatal("exact standalone declaration did not retire")
	}
	if len(tbl.logicalIndexesForTx(tx)) != 0 {
		t.Fatal("retired declaration remained selected")
	}
	tx.Rollback()
	if len(tbl.LogicalIndexes) != 0 {
		t.Fatal("aborted index declaration became public")
	}
	// An old native key is deliberately not evidence of a standalone SQL index.
	tbl.Unique = []uniqueKey{{Id: "legacy_unknown", Cols: []string{"id"}}}
	tbl.publishShowColumnsSnapshot(true)
	next := NewTxContext(TxACID)
	defer next.Rollback()
	requirePanic(t, func() { mutateLogicalIndex(tbl, logicalIndexDeclaration{Name: "legacy_unknown"}, "drop", next) })
	if len(tbl.Unique) != 1 {
		t.Fatal("DROP INDEX removed unclassified legacy enforcement")
	}
}

func TestLogicalIndexRetirementKeepsForeignSupport(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tlogicalindexfk", 1)
	index := logicalIndexDeclaration{Name: "UX_id", Columns: []string{"id"}, Key: &uniqueKey{Id: "UX_id", Cols: []string{"id"}}}
	tbl.LogicalIndexes = []logicalIndexDeclaration{index}
	tbl.Unique = []uniqueKey{*index.Key}
	tbl.publishShowColumnsSnapshot(true)
	p := newCatalogNativeProjection(tbl.schema)
	p.foreign[catalogForeignIdentity{}] = []catalogForeignDeclaration{{Foreign: foreignKey{Id: "FK_actual", Tbl1: "child", Cols1: []string{"parent"}, Tbl2: tbl.Name, Cols2: []string{"id"}}}}
	d := catalogIndexDeclaration{tbl.Name, tbl.RuntimeIdentity, index}
	requirePanic(t, func() { tbl.schema.projectIndexDeclaration(&d, "drop", p, nil, catalogForeignEndpointScope{}) })
	if len(tbl.Unique) != 1 || len(tbl.LogicalIndexes) != 1 {
		t.Fatal("failed projection edited live enforcement")
	}
	p.foreign = make(map[catalogForeignIdentity][]catalogForeignDeclaration)
	tbl.schema.projectIndexDeclaration(&d, "drop", p, nil, catalogForeignEndpointScope{})
	if len(p.keys[tbl]) != 0 || len(p.indexes[tbl]) != 0 || len(tbl.Unique) != 1 {
		t.Fatal("retirement did not advance only the private existing projector")
	}
}
