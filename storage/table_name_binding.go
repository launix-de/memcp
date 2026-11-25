/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// V2 extends this REQUIRED checkpoint field, not an optional ignored table
// member. Every prior reader rejects it after index WAL has been checkpointed.
type tableNameBinding struct {
	Version int                       `json:"version"`
	Storage string                    `json:"storage"`
	Visible string                    `json:"visible"`
	Indexes []logicalIndexDeclaration `json:"indexes,omitempty"`
}

func (t *table) persistedName() any {
	if len(t.LogicalIndexes) != 0 {
		return tableNameBinding{2, t.Name, t.rawVisibleName(), t.LogicalIndexes}
	}
	if t.rawVisibleName() == t.Name {
		return t.Name
	}
	return tableNameBinding{Version: 1, Storage: t.Name, Visible: t.rawVisibleName()}
}
func (t *table) UnmarshalJSON(data []byte) error {
	type declaration table
	var encoded struct {
		*declaration
		Name json.RawMessage
	}
	encoded.declaration = (*declaration)(t)
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}
	t.LogicalIndexes = nil
	raw := bytes.TrimSpace(encoded.Name)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil
	}
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &t.Name); err != nil {
			return err
		}
		t.publishedName.Store(nil)
		return nil
	}
	var binding tableNameBinding
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&binding); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	if (binding.Version != 1 && binding.Version != 2) || binding.Storage == "" || binding.Visible == "" || !utf8.ValidString(binding.Storage) || !utf8.ValidString(binding.Visible) {
		return fmt.Errorf("invalid native table name binding")
	}
	if binding.Version == 1 {
		for key := range fields {
			if key == "indexes" {
				return fmt.Errorf("legacy table binding cannot carry indexes")
			}
		}
	} else {
		if len(binding.Indexes) == 0 {
			return fmt.Errorf("required index binding is empty")
		}
		if err := validateIndexDirectory(binding.Indexes, t.Unique); err != nil {
			return err
		}
		for _, index := range binding.Indexes {
			for _, cols := range [][]string{index.Columns, index.Included} {
				for _, name := range cols {
					found := false
					for _, c := range t.Columns {
						if c.Name == name {
							found = true
							break
						}
					}
					if !found {
						return fmt.Errorf("unknown persisted logical index column: %s", name)
					}
				}
			}
		}
		t.LogicalIndexes = binding.Indexes
	}
	t.Name = binding.Storage
	t.publishedName.Store(&binding.Visible)
	return nil
}
