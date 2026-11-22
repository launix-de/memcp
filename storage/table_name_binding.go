/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

func (t *table) persistedName() any {
	if t.rawVisibleName() == t.Name {
		return t.Name
	}
	return columnNameBinding{1, t.Name, t.rawVisibleName()}
}

// Like a column binding, a renamed table changes the TYPE of the existing
// checkpoint Name field. Old binaries fail closed after WAL retirement.
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
	raw := bytes.TrimSpace(encoded.Name)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil
	}
	if raw[0] == '"' {
		return json.Unmarshal(raw, &t.Name)
	}
	var binding columnNameBinding
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&binding); err != nil {
		return err
	}
	if binding.Version != 1 || binding.Storage == "" || binding.Visible == "" || !utf8.ValidString(binding.Storage) || !utf8.ValidString(binding.Visible) {
		return fmt.Errorf("invalid native table name binding")
	}
	t.Name = binding.Storage
	t.publishedName.Store(&binding.Visible)
	return nil
}
