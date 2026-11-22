/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
	"unicode/utf8"
	"unsafe"
)

// The physical key remains the existing Name. This avoids rewriting any row,
// WAL, index, computed input or file-liveness identity for a logical rename.
func (c *column) displayName() string {
	if c.visibleName != "" {
		return c.visibleName
	}
	return c.Name
}

type columnNameBinding struct {
	Version int    `json:"version"`
	Storage string `json:"storage"`
	Visible string `json:"visible"`
}

// The existing checkpoint Name field changes TYPE for a renamed declaration.
// An older reader expects a string and must reject this object, even after the
// rename WAL has been checkpointed and retired. Unknown extra fields alone
// would allow an older reader to silently open a different physical column.
func (c *column) persistedName() any {
	if c.displayName() == c.Name {
		return c.Name
	}
	return columnNameBinding{1, c.Name, c.displayName()}
}

func (c *column) restoreName(raw json.RawMessage) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil
	}
	c.visibleName = ""
	if raw[0] == '"' {
		return json.Unmarshal(raw, &c.Name)
	}
	var binding columnNameBinding
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&binding); err != nil {
		return err
	}
	if binding.Version != 1 || binding.Storage == "" || binding.Visible == "" || !utf8.ValidString(binding.Storage) || !utf8.ValidString(binding.Visible) {
		return fmt.Errorf("invalid native column name binding")
	}
	c.Name, c.visibleName = binding.Storage, binding.Visible
	return nil
}

// Constraint recipes stay bound to native physical identities. Only the cold
// descriptive result maps them to the selected original-T visible namespace;
// physical vectors remain explicit for subsequent native DDL expectations.
// The schema/amendment publisher supplies the complete selected endpoint
// views. No descriptive getter rebuilds constraints or visits another table.
func (t *table) projectConstraintNames(snapshot *tableShowColumnsSnapshot, views map[*table]*tableShowColumnsSnapshot, namespace *catalogNamespaceView) (*tableShowColumnsSnapshot, uint) {
	constraints := snapshot.nativeConstraints()
	local := snapshot.metadata.columnNames
	bind := func(value scm.Scmer, names *tableColumnNamesSnapshot) (scm.Scmer, bool) {
		if names == nil || !value.IsSlice() {
			return value, false
		}
		var result []scm.Scmer
		for i, name := range value.Slice() {
			entry, ok := names.exact[name.String()]
			if !ok {
				continue
			}
			visible := names.declarations[entry.ordinal].displayName()
			if visible != name.String() {
				if result == nil {
					result = append([]scm.Scmer(nil), value.Slice()...)
				}
				result[i] = scm.NewString(visible)
			}
		}
		if result == nil {
			return value, false
		}
		return scm.NewSlice(result), true
	}
	var projected []scm.Scmer
	var bytes uint
	root := constraints.Slice()
	for i := 0; i+1 < len(root); i += 2 {
		category := root[i].String()
		rows := root[i+1].Slice()
		var changedRows []scm.Scmer
		for j, row := range rows {
			fields := row.Slice()
			var changedFields []scm.Scmer
			var other *tableColumnNamesSnapshot
			if category == "ForeignKeys" {
				name := foreignMetadataValue(row, "OtherTable")
				if name.IsString() {
					if target := t.schema.tables.Get(name.String()); target != nil {
						otherView := views[target]
						if otherView == nil {
							otherView = target.showColumnsSnapshot.Load()
						}
						if otherView != nil {
							other = otherView.metadata.columnNames
						}
						visible := target.rawVisibleName()
						if namespace != nil {
							if own, found := namespace.names[target]; found {
								visible = own
							}
						}
						if visible != name.String() {
							changedFields = append([]scm.Scmer(nil), fields...)
							for k := 0; k+1 < len(fields); k += 2 {
								if fields[k].String() == "OtherTable" {
									changedFields[k+1] = scm.NewString(visible)
								}
							}
							changedFields = append(changedFields, scm.NewString("OtherStorageTable"), name)
						}
					}
				}
			}
			for k := 0; k+1 < len(fields); k += 2 {
				key := fields[k].String()
				var names *tableColumnNamesSnapshot
				storage := ""
				switch key {
				case "Cols":
					names, storage = local, "StorageCols"
				case "Columns":
					names, storage = local, "StorageColumns"
				case "LocalColumns":
					names, storage = local, "LocalStorageColumns"
				case "OtherColumns":
					names, storage = other, "OtherStorageColumns"
				}
				if names == nil {
					continue
				}
				value, changed := bind(fields[k+1], names)
				if !changed {
					continue
				}
				if changedFields == nil {
					changedFields = append([]scm.Scmer(nil), fields...)
				}
				changedFields[k+1] = value
				changedFields = append(changedFields, scm.NewString(storage), fields[k+1])
				bytes += uint(cap(value.Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
			}
			if changedFields != nil {
				if changedRows == nil {
					changedRows = append([]scm.Scmer(nil), rows...)
				}
				changedRows[j] = scm.NewSlice(changedFields)
				bytes += uint(cap(changedFields)) * uint(unsafe.Sizeof(scm.Scmer{}))
			}
		}
		if changedRows != nil {
			if projected == nil {
				projected = append([]scm.Scmer(nil), root...)
			}
			projected[i+1] = scm.NewSlice(changedRows)
			bytes += uint(cap(changedRows)) * uint(unsafe.Sizeof(scm.Scmer{}))
		}
	}
	if projected == nil && snapshot.metadata.constraints == constraints {
		return snapshot, 0
	}
	result, metadata := *snapshot, *snapshot.metadata
	result.metadata = &metadata
	metadata.storageConstraints = constraints
	metadata.constraints = constraints
	if projected != nil {
		metadata.constraints = scm.NewSlice(projected)
		bytes += uint(cap(projected)) * uint(unsafe.Sizeof(scm.Scmer{}))
	}
	metadata.foreignByID = buildForeignMetadataDirectory(metadata.constraints)
	metadata.declarations = buildTableDeclarations(metadata.columnRows, metadata.constraints)
	bytes += uint(unsafe.Sizeof(result)) + uint(unsafe.Sizeof(metadata)) + uint(cap(metadata.declarations.Slice()))*uint(unsafe.Sizeof(scm.Scmer{}))
	return &result, bytes
}

func (snapshot *tableShowColumnsSnapshot) nativeConstraints() scm.Scmer {
	if !snapshot.metadata.storageConstraints.IsNil() {
		return snapshot.metadata.storageConstraints
	}
	return snapshot.metadata.constraints
}

// Constructor/schema owner only. Reusing a visible name never reuses an older
// physical incarnation (including its old WAL cells and retired column files).
func (t *table) allocateColumnStorage(c *column, columns []*column) {
	requested := c.displayName()
	occupied := false
	for _, before := range columns {
		occupied = occupied || before.Name == c.Name
	}
	for name := range t.DroppedColumns {
		occupied = occupied || scm.CaseFoldKey(name) == scm.CaseFoldKey(c.Name)
	}
	if occupied {
		c.Name, c.visibleName = ".column:"+uuid.NewString(), requested
	}
}
