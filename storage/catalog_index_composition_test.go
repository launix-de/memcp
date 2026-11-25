/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"encoding/json"
	"github.com/launix-de/memcp/scm"
	"strings"
	"testing"
)

// This candidate-only domain composes the existing original-transaction append profile;
// its supporting index is never a substitute for native namespace authority.
func TestLogicalIndexV32PreservesJointNullableAppendProfile(t *testing.T) {
	const authority = "11111111-1111-4111-8111-111111111111"
	const target = "22222222-2222-4222-8222-222222222222"
	const retired = "33333333-3333-4333-8333-333333333333"
	const indexed = "44444444-4444-4444-8444-444444444444"
	const support = "55555555-5555-4555-8555-555555555555"
	makeRecord := func() catalogDecisionRecord {
		return catalogDecisionRecord{Format: 32, Kind: "prepare", Database: "index_append_state", Incarnation: authority,
			Transaction: authority + "/1", Decision: authority, Installation: authority, Boot: authority, Version: 1, Distributed: true,
			ColumnAppends: []catalogColumnAppend{{Table: "append_rows", Identity: target, Ordinal: 1, Name: "added",
				Type: storedColumnType{Type: "INT", Nullable: true, Collation: "utf8mb4"}, Initialization: &catalogColumnInitialization{Default: scm.NewNil(), Value: scm.NewNil(), Extents: map[string]columnInitialExtent{}}}},
			TableDrops: []catalogTableDropDeclaration{{Name: "retired_rows", Identity: retired}},
			NativeDeclarations: []catalogNativeDeclaration{
				{Operation: "table_rename", TableRename: &catalogTableRename{Storage: "namespace_support", Identity: support, Before: "namespace_support", After: "renamed_support"}},
				{Operation: "index_add", Index: &catalogIndexDeclaration{Table: "index_rows", Identity: indexed, Index: logicalIndexDeclaration{Name: "IX_current", Columns: []string{"id"}}}},
			}}
	}
	// Validate the independent payload owners before asking the outer reader
	// about composition. A fixture shape error must not impersonate the composition gate.
	assertIndependentShapes := func(t *testing.T, record catalogDecisionRecord) {
		t.Helper()
		raw, err := json.Marshal(record)
		if err != nil {
			t.Fatalf("fixture cannot encode: %v", err)
		}
		var decoded catalogDecisionRecord
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("fixture cannot decode its native fields: %v", err)
		}
		for _, created := range decoded.Tables {
			if created == nil || created.Name == "" || created.RuntimeIdentity == "" || created.CreatorTxID != decoded.Transaction || created.CreatorIncarnation != decoded.Incarnation || created.CreatorNamespace != decoded.Namespace || created.CatalogDecisionID != decoded.Decision || created.CatalogVersion.Load() != decoded.Version {
				t.Fatal("invalid constructor fixture authority after native field roundtrip")
			}
		}
		if err := validateColumnAppendShape(decoded.ColumnAppends); err != nil {
			t.Fatalf("invalid append fixture before composition: %v", err)
		}
		if len(decoded.NativeDeclarations) > 0 {
			if err := validateNativeDeclarationShape(decoded.NativeDeclarations); err != nil {
				t.Fatalf("invalid native action fixture before composition: %v", err)
			}
		}
		if err := validateAllocatorResetDeclarations(decoded.AllocatorResets); err != nil {
			t.Fatalf("invalid reset fixture before composition: %v", err)
		}
		if err := decoded.validateIdentityAppendProfile(); err != nil {
			t.Fatalf("invalid identity fixture before composition: %v", err)
		}
	}
	accepted := makeRecord()
	assertIndependentShapes(t, accepted)
	if _, err := decodeCatalogDecision(encodeCatalogDecision(&accepted)); err != nil {
		t.Fatalf("V32 rejected the existing disjoint name-owned program: %v", err)
	}
	// A genuine constructor is an independent existing namespace proof, without
	// requiring a fabricated name action or a different transaction.
	constructed := makeRecord()
	created := &table{Name: "replacement_rows", RuntimeIdentity: support, CreatorTxID: constructed.Transaction,
		CreatorIncarnation: constructed.Incarnation, CreatorNamespace: constructed.Namespace, CatalogDecisionID: constructed.Decision,
		PersistencyMode: Safe, Collation: "utf8mb4", Charset: "utf8mb4", Columns: []*column{{Name: "id", Typ: "INT", Collation: "utf8mb4"}}}
	created.CatalogVersion.Store(constructed.Version)
	created.Columns[0].UpdateSanitizer()
	created.publishShowColumnsSnapshot(true)
	constructed.Tables = []*table{created}
	constructed.NativeDeclarations = constructed.NativeDeclarations[1:]
	constructed.NativeDeclarations[0].Index.Table = created.Name
	constructed.NativeDeclarations[0].Index.Identity = support
	assertIndependentShapes(t, constructed)
	if _, err := decodeCatalogDecision(encodeCatalogDecision(&constructed)); err != nil {
		t.Fatalf("V32 rejected the existing constructor-owned replacement program: %v", err)
	}
	for _, tc := range []struct {
		name   string
		want   string
		change func(*catalogDecisionRecord)
	}{
		{"index_is_not_namespace_authority", "existing owned namespace decision", func(r *catalogDecisionRecord) { r.NativeDeclarations = r.NativeDeclarations[1:] }},
		{"same_physical_endpoint", "retirement of its endpoint", func(r *catalogDecisionRecord) { r.TableDrops[0].Name = r.ColumnAppends[0].Table }},
		{"same_uuid_other_spelling", "retirement of its endpoint", func(r *catalogDecisionRecord) { r.TableDrops[0].Identity = r.ColumnAppends[0].Identity }},
		{"not_nullable", "ordinary nullable prefix append", func(r *catalogDecisionRecord) {
			r.ColumnAppends[0].Type.Nullable = false
			r.ColumnAppends[0].Initialization.Value = scm.NewInt(0)
		}},
		{"missing_prefix_recipe", "ordinary nullable prefix append", func(r *catalogDecisionRecord) { r.ColumnAppends[0].Initialization = nil }},
		{"realized_rows", "ordinary nullable prefix append", func(r *catalogDecisionRecord) {
			d := &r.ColumnAppends[0]
			d.Initialization = nil
			d.RowInitialization = &catalogColumnDefaultDeclaration{Table: d.Table, Identity: d.Identity, Column: d.Name, Value: scm.NewNil()}
		}},
		{"generated_rows", "ordinary nullable prefix append", func(r *catalogDecisionRecord) {
			d := &r.ColumnAppends[0]
			d.Type.Nullable = false
			d.Initialization = nil
			d.GeneratedIDs = &columnIDGeneration{1, 1}
			r.AllocatorResets = []catalogAllocatorResetDeclaration{{Table: d.Table, Column: d.Name, State: tableAllocatorState{Version: 2, Table: d.Identity, StartKnown: true, Start: 1, IssuedKnown: true, High: 0, Epoch: "66666666-6666-4666-8666-666666666666", Current: 1, Next: 1, HasNext: true}}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := makeRecord()
			tc.change(&record)
			assertIndependentShapes(t, record)
			if _, err := decodeCatalogDecision(encodeCatalogDecision(&record)); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("V32 did not preserve the exact existing composition refusal %q: %v", tc.want, err)
			}
		})
	}
	// The additional restriction belongs only to the new required family. An
	// already valid V23 namespace program keeps its exact historical reader.
	legacy := makeRecord()
	legacy.Format = 23
	legacy.NativeDeclarations = legacy.NativeDeclarations[:1]
	assertIndependentShapes(t, legacy)
	if _, err := decodeCatalogDecision(encodeCatalogDecision(&legacy)); err != nil {
		t.Fatalf("existing V23 disjoint namespace reader changed: %v", err)
	}
	// V32 index-only decisions need no joint profile when no append/drop pair
	// exists. This must not become a blanket namespace prerequisite for indexes.
	ordinary := makeRecord()
	ordinary.ColumnAppends = nil
	ordinary.TableDrops = nil
	ordinary.NativeDeclarations = ordinary.NativeDeclarations[1:]
	assertIndependentShapes(t, ordinary)
	if _, err := decodeCatalogDecision(encodeCatalogDecision(&ordinary)); err != nil {
		t.Fatalf("ordinary V32 index authority was overrestricted: %v", err)
	}
}
