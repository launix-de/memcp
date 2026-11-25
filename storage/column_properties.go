/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/launix-de/memcp/scm"
)

// columnDeclarationProperties is immutable scalar declaration data. Frontends
// own the interpretation of namespaced keys; storage neither executes values
// nor interprets a dialect's flags. Replacement DDL creates a fresh map.
type columnDeclarationProperties map[string]scm.Scmer

func validateColumnProperty(key string, value scm.Scmer) error {
	if key == "" {
		return fmt.Errorf("column property key must not be empty")
	}
	if !(value.IsNil() || value.IsBool() || value.IsInt() || value.IsFloat() || value.IsString()) {
		return fmt.Errorf("column property values must be scalars")
	}
	if value.IsFloat() && (math.IsNaN(value.Float()) || math.IsInf(value.Float(), 0)) {
		return fmt.Errorf("column property number must be finite")
	}
	return nil
}
func columnPropertiesFromValue(value scm.Scmer) columnDeclarationProperties {
	if value.IsNil() {
		return nil
	}
	fields := mustScmerSlice(value, "column properties")
	if len(fields)%2 != 0 {
		panic("column properties require key/value pairs")
	}
	result := make(columnDeclarationProperties, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		if !fields[i].IsString() {
			panic("column property keys must be strings")
		}
		key := fields[i].String()
		if _, exists := result[key]; exists {
			panic("duplicate column property")
		}
		if err := validateColumnProperty(key, fields[i+1]); err != nil {
			panic(err)
		}
		result[key] = fields[i+1]
	}
	return result
}
func (properties *columnDeclarationProperties) UnmarshalJSON(data []byte) error {
	var values map[string]scm.Scmer
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	for key, value := range values {
		if err := validateColumnProperty(key, value); err != nil {
			return err
		}
	}
	*properties = values
	return nil
}
func (properties columnDeclarationProperties) description() scm.Scmer {
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]scm.Scmer, 0, 2*len(keys))
	for _, key := range keys {
		result = append(result, scm.NewString(key), properties[key])
	}
	return scm.NewSlice(result)
}
