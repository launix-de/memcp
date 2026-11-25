/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestForeignEnforcementSchemaCompatibility(t *testing.T) {
	legacy := []byte(`{"Id":"rule","Tbl1":"child","Cols1":["parent_id"],"Tbl2":"parent","Cols2":["id"],"Updatemode":1,"Deletemode":2}`)
	var enabled foreignKey
	if err := json.Unmarshal(legacy, &enabled); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(enabled)
	if err != nil || !bytes.Equal(encoded, legacy) {
		t.Fatalf("enabled historical schema bytes changed: %s, %v", encoded, err)
	}
	for _, mode := range []foreignKeyMode{RESTRICT, CASCADE, SETNULL} {
		key := enabled
		key.Updatemode, key.Disabled, key.Untrusted = mode, true, true
		encoded, err := json.Marshal(key)
		if err != nil {
			t.Fatal(err)
		}
		var restored foreignKey
		if err := json.Unmarshal(encoded, &restored); err != nil || !sameForeignDeclaration(key, restored) {
			t.Fatalf("disabled schema did not preserve native state: %s, %v", encoded, err)
		}
		// This is the historical numeric field contract, not a second decoder.
		// It must reject even standalone schema imports without a catalog WAL.
		var oldReader struct{ Updatemode foreignKeyMode }
		if err := json.Unmarshal(encoded, &oldReader); err == nil {
			t.Fatalf("old numeric schema reader accepted disabled semantics: %s", encoded)
		}
	}
	for _, malformed := range []string{
		`{"Disabled":true,"Untrusted":true,"Updatemode":0}`,
		`{"Disabled":true,"Untrusted":true,"Updatemode":"disabled-v1/0","Deletemode":3}`,
		`{"Disabled":true,"Untrusted":true}`,
		`{"Disabled":true,"Untrusted":true,"Updatemode":null}`,
		`{"Disabled":true,"Untrusted":true,"Updatemode":"disabled-v1/3"}`,
		`{"Disabled":true,"Untrusted":true,"Updatemode":"disabled-v2/0"}`,
		`{"Disabled":true,"Untrusted":false,"Updatemode":"disabled-v1/0"}`,
		`{"Disabled":false,"Untrusted":true,"Updatemode":"disabled-v1/0"}`,
		`{"Untrusted":true,"Updatemode":"disabled-v1/0"}`,
		`{"Disabled":null,"Untrusted":true,"Updatemode":0}`,
		`{"Disabled":true,"Untrusted":null,"Updatemode":"disabled-v1/0"}`,
	} {
		var key foreignKey
		if err := json.Unmarshal([]byte(malformed), &key); err == nil {
			t.Errorf("accepted malformed enforcement representation: %s", malformed)
		}
	}
}

func TestForeignEnforcementJournalDomains(t *testing.T) {
	const identity = "11111111-1111-4111-8111-111111111111"
	base := catalogDecisionRecord{
		Format: 29, Kind: "prepare", Database: "foreign_state", Incarnation: identity,
		Transaction: identity + "/1", Decision: identity, Installation: identity, Boot: identity,
		Version: 1, Distributed: true,
		NativeDeclarations: []catalogNativeDeclaration{{Operation: "foreign_add", Foreign: &catalogForeignDeclaration{
			ChildIdentity: identity, ParentIdentity: "22222222-2222-4222-8222-222222222222",
			Foreign: foreignKey{Id: "rule", Tbl1: "child", Cols1: []string{"parent_id"}, Tbl2: "parent", Cols2: []string{"id"}, Disabled: true, Untrusted: true},
		}}},
	}
	if restored, err := decodeCatalogDecision(encodeCatalogDecision(&base)); err != nil || restored == nil || !restored.NativeDeclarations[0].Foreign.Foreign.Disabled {
		t.Fatalf("valid V29 disabled decision rejected: %v", err)
	}
	// Preserve established enabled domains while rejecting the new field even
	// when explicitly false. An old version cannot acquire V29 semantics.
	for _, version := range []int{15, 18, 28} {
		record := base
		record.Format = version
		data, err := json.Marshal(&record)
		if err != nil {
			t.Fatal(err)
		}
		prefix := catalogNativeDeclarationJournalPrefix
		if version == 18 {
			prefix = catalogForeignValidationJournalPrefix
		} else if version == 28 {
			prefix = catalogColumnRetirementJournalPrefix
		}
		if _, err := decodeCatalogDecision(prefix + base64.RawURLEncoding.EncodeToString(data)); err == nil {
			t.Errorf("historical V%d accepted disabled enforcement", version)
		}
		data = bytes.ReplaceAll(data, []byte(`"Disabled":true`), []byte(`"Disabled":false`))
		data = bytes.ReplaceAll(data, []byte(`"Updatemode":"disabled-v1/0"`), []byte(`"Updatemode":0`))
		if _, err := decodeCatalogDecision(prefix + base64.RawURLEncoding.EncodeToString(data)); err == nil {
			t.Errorf("historical V%d accepted explicit disabled=false field", version)
		}
	}
	for _, version := range []int{15, 18} {
		record := base
		record.Format = version
		foreign := *base.NativeDeclarations[0].Foreign
		foreign.Foreign.Disabled = false
		foreign.Foreign.Untrusted = version == 18
		record.NativeDeclarations = []catalogNativeDeclaration{{Operation: "foreign_add", Foreign: &foreign}}
		if decoded, err := decodeCatalogDecision(encodeCatalogDecision(&record)); err != nil || decoded == nil {
			t.Fatalf("legacy enabled V%d declaration rejected: %v", version, err)
		}
	}
	valid := encodeCatalogDecision(&base)
	if _, err := decodeCatalogDecision("catalog-v10/v30/" + valid[len(catalogForeignEnforcementJournalPrefix):]); err == nil {
		t.Fatal("unknown required catalog family accepted")
	}
	data, err := json.Marshal(&base)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte(`"format":29`), []byte(`"format":30`), 1)
	if _, err := decodeCatalogDecision(catalogForeignEnforcementJournalPrefix + base64.RawURLEncoding.EncodeToString(data)); err == nil {
		t.Fatal("unknown format inside V29 family accepted")
	}
}
