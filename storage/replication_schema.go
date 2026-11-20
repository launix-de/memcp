/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "bytes"
import "strings"
import "encoding/json"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

// validateReplicaCatalogRows applies the existing client declaration admission
// before an imported native partition becomes visible. JSON-valued catalog
// cells remain inert until validateReplicaSchema has checked them. A client
// must supply the canonical, receiver-restricted declaration: filtering or
// rewriting after publishing a shard would leave unsafe bytes in its root.
func validateReplicaCatalogRows(schema string, relation catalogRelationID, rows [][]scm.Scmer) error {
	r := &catalogRelations[relation]
	fields := r.columns()
	encoder := catalogJSONEncoder{strings: make(map[string]json.RawMessage)}
	ordinalIndex := -1
	for i, field := range r.identity {
		if field.kind == catalogInt {
			ordinalIndex = i + 1
		}
	}
	type ordinalGroup struct{ count, maximum int64 }
	groups := make(map[string]ordinalGroup)
	identities := make(map[string]bool)
	for _, row := range rows {
		if len(row) != len(fields) || !row[0].IsString() || row[0].String() != schema {
			return fmt.Errorf("invalid replicated catalog row identity in %s", r.name)
		}
		for i, field := range fields {
			value := row[i]
			if value.IsNil() {
				if field.key == "" {
					return fmt.Errorf("missing replicated catalog identity in %s", r.name)
				}
				continue
			}
			valid := false
			switch field.kind {
			case catalogText:
				valid = value.IsString()
			case catalogInt:
				valid = value.IsInt()
				if valid && field.key == "" {
					valid = value.Int() >= 0 && value.Int() < int64(len(rows))
				}
			case catalogBool:
				valid = value.IsBool()
			case catalogFloat:
				valid = value.IsFloat() || value.IsInt()
			case catalogList:
				valid = value.IsSlice()
			case catalogJSON:
				valid = value.IsString() && json.Valid([]byte(value.String()))
			}
			if !valid {
				return fmt.Errorf("invalid replicated catalog cell %s.%s", r.name, field.column)
			}
		}
		if ordinalIndex >= 0 {
			key := catalogRowKey(fields[:ordinalIndex+1], row[:ordinalIndex+1])
			if identities[key] {
				return fmt.Errorf("duplicate replicated catalog ordinal in %s", r.name)
			}
			identities[key] = true
			parent := catalogRowKey(fields[:ordinalIndex], row[:ordinalIndex])
			group := groups[parent]
			group.count++
			group.maximum = max(group.maximum, row[ordinalIndex].Int())
			groups[parent] = group
		}
		switch relation {
		case catalogDatabases, catalogKeyColumns, catalogForeignKeyColumns:
			continue // identity/scalar cells only; no runtime declaration codec
		}
		record := encoder.record(relation, row)
		document := catalogDocument{Name: schema, Tables: make(map[string]json.RawMessage)}
		if relation == catalogProcedures {
			document.Procedures = []json.RawMessage{encoder.appendRecord(nil, record)}
		} else {
			name := row[1].String()
			if relation == catalogTables {
				name = catalogTableName(row)
				document.Tables[name] = encoder.appendRecord(nil, record)
			} else {
				child := map[catalogRelationID]string{catalogColumns: "Columns", catalogKeys: "Unique", catalogForeignKeys: "Foreign", catalogTriggers: "Triggers"}[relation]
				table := map[string]json.RawMessage{
					"Name": marshalCatalogValue(name),
					child:  marshalCatalogRecords([]json.RawMessage{encoder.appendRecord(nil, record)}),
				}
				document.Tables[name] = encoder.appendRecord(nil, table)
			}
		}
		encoded, err := json.Marshal(document)
		if err != nil {
			return err
		}
		admitted, err := validateReplicaSchema(encoded)
		if err != nil {
			return err
		}
		normalized := catalogRowsFromDocument(admitted)
		if len(normalized[relation]) != 1 {
			return fmt.Errorf("replicated catalog %s contains a local cache declaration", r.name)
		}
		offset := 1 + len(r.identity)
		for i, field := range r.fields {
			if field.key == "schema_only" && scm.ToBool(row[offset+i]) != scm.ToBool(normalized[relation][0][offset+i]) {
				return fmt.Errorf("replicated catalog %s requires receiver-restricted declarations", r.name)
			}
		}
	}
	for _, group := range groups {
		if group.maximum+1 != group.count {
			return fmt.Errorf("replicated catalog %s has incomplete child ordinals", r.name)
		}
	}
	if relation == catalogDatabases && len(rows) != 1 {
		return fmt.Errorf("replicated catalog lacks its database declaration")
	}
	return nil
}

// validateReplicaSchema admits declarations from clients, never executable
// runtime artifacts. Inspect inert JSON first: Scmer decoding restores callable
// lambdas and custom runtime values. Trusted cluster and local persistence loads
// continue to use their existing codec without this restriction.
func validateReplicaSchema(data []byte) ([]byte, error) {
	// Query caches are local, rebuildable code artifacts rather than replicated
	// declarations. Remove them as inert JSON before admitting any client code.
	var err error
	data, err = replicaSchemaDeclarations(data, true)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	var plain func(any) bool
	plain = func(value any) bool {
		switch value := value.(type) {
		case map[string]any:
			if _, symbol := value["symbol"].(string); symbol {
				return false
			}
			if _, variable := value["var"].(json.Number); variable && len(value) == 1 {
				return false
			}
			if _, custom := value["scmer_custom"].(json.Number); custom && len(value) == 2 {
				return false
			}
			for _, child := range value {
				if !plain(child) {
					return false
				}
			}
		case []any:
			for _, child := range value {
				if !plain(child) {
					return false
				}
			}
		}
		return true
	}
	if !plain(document) {
		return nil, fmt.Errorf("replication slave schema contains Scheme code or runtime values")
	}
	// Decode through the native declarations so field aliases and casing have
	// exactly the same meaning as the catalog loader. Never validate a parallel
	// interpretation of a potentially mixed-case JSON document.
	db := newDatabase()
	if err := json.Unmarshal(data, db); err != nil {
		return nil, err
	}
	for _, table := range db.tables.GetAll() {
		if table == nil {
			return nil, fmt.Errorf("invalid replicated table")
		}
		if initializer := table.OnInit.Load(); initializer != nil && !initializer.IsNil() {
			return nil, fmt.Errorf("replication slave cannot supply oninit code")
		}
		for _, column := range table.Columns {
			// The manifest has its own non-column wire format. It must never
			// double as a user column that bypasses column-format admission.
			if column == nil || column.Name == blobManifestColumn || strings.ContainsAny(column.Name, "/\\\x00") {
				return nil, fmt.Errorf("invalid replicated column name")
			}
		}
		for i := range table.Triggers {
			trigger := &table.Triggers[i]
			// FK callbacks are rebuilt from native foreign-key declarations by
			// ensureLoaded. A client may send the inert declaration, never a body.
			if trigger.IsSystem && strings.HasPrefix(trigger.Name, "__fk_") && trigger.Func.IsNil() && trigger.Language == "" && trigger.Source == "" {
				continue
			}
			if !trigger.Func.IsNil() || !replicationSafeLanguage(trigger.Language) || trigger.Source == "" {
				return nil, fmt.Errorf("replication slave trigger requires an approved source language")
			}
			if trigger.Timing > AfterDelete || trigger.IsSystem || trigger.Hidden || trigger.Acquire != nil {
				return nil, fmt.Errorf("replication slave cannot supply internal or lifecycle triggers")
			}
			trigger.SchemaOnly = true
		}
	}
	for _, procedure := range db.procedures.GetAll() {
		if !replicationSafeLanguage(procedure.Language) {
			return nil, fmt.Errorf("replication slave procedure requires an approved source language")
		}
		procedure.SchemaOnly = true
	}
	// Marshal the validated declarations, including the receiver-owned source
	// restriction. Persisting it keeps lazy compilation safe after a restart.
	return json.Marshal(db)
}

// replicaSchemaDeclarations uses the same case-insensitive field decoding as
// the native catalog, while leaving every runtime value inert. Discarding a
// client-marked cache is safe: none of its supplied bytes are ever installed.
func replicaSchemaDeclarations(data []byte, restricted bool) ([]byte, error) {
	var db struct {
		Name       string                     `json:"name"`
		Tables     map[string]json.RawMessage `json:"tables"`
		Procedures json.RawMessage            `json:"procedures,omitempty"`
	}
	if err := json.Unmarshal(data, &db); err != nil {
		return nil, err
	}
	for name, raw := range db.Tables {
		var metadata struct {
			Name            string
			PersistencyMode PersistencyMode
			Columns         []json.RawMessage
			Triggers        []json.RawMessage
		}
		if err := json.Unmarshal(raw, &metadata); err != nil {
			return nil, err
		}
		if metadata.Name != name {
			return nil, fmt.Errorf("replicated table identity mismatch")
		}
		if strings.HasPrefix(name, ".") && metadata.PersistencyMode == Cache {
			delete(db.Tables, name)
			continue
		}
		var columns []json.RawMessage
		for _, column := range metadata.Columns {
			var declaration struct{ IsTemp bool }
			if err := json.Unmarshal(column, &declaration); err != nil {
				return nil, err
			}
			if !declaration.IsTemp {
				columns = append(columns, column)
			}
		}
		var triggers []json.RawMessage
		for _, trigger := range metadata.Triggers {
			var declaration struct {
				Name           string `json:"name"`
				RequiresTarget bool   `json:"requires_target"`
				Source         string `json:"source"`
			}
			if err := json.Unmarshal(trigger, &declaration); err != nil {
				return nil, err
			}
			if !declaration.RequiresTarget && !restoredTriggerRequiresRuntimeTarget(declaration.Name) {
				if restricted && declaration.Source != "" {
					var err error
					trigger, err = replicaRestrictedSource(trigger)
					if err != nil {
						return nil, err
					}
				}
				triggers = append(triggers, trigger)
			}
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return nil, err
		}
		for key := range fields {
			if strings.EqualFold(key, "Columns") || strings.EqualFold(key, "Triggers") {
				delete(fields, key)
			}
		}
		fields["Columns"], _ = json.Marshal(columns)
		fields["Triggers"], _ = json.Marshal(triggers)
		db.Tables[name], _ = json.Marshal(fields)
	}
	if restricted && len(db.Procedures) != 0 {
		var procedures []json.RawMessage
		if err := json.Unmarshal(db.Procedures, &procedures); err != nil {
			return nil, err
		}
		for i, procedure := range procedures {
			var err error
			procedures[i], err = replicaRestrictedSource(procedure)
			if err != nil {
				return nil, err
			}
		}
		db.Procedures, _ = json.Marshal(procedures)
	}
	return json.Marshal(db)
}

// Source restrictions belong to admitted declarations, not to a rewritten Go
// projection at the receiver. Keep preparation inert even for unsupported local
// languages; their upload is rejected by the ordinary client admission later.
func replicaRestrictedSource(raw json.RawMessage) (json.RawMessage, error) {
	var declaration struct{ Language string }
	if err := json.Unmarshal(raw, &declaration); err != nil {
		return nil, err
	}
	if !replicationSafeLanguage(declaration.Language) {
		return raw, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	for key := range fields {
		if strings.EqualFold(key, "schema_only") {
			delete(fields, key)
		}
	}
	fields["schema_only"] = json.RawMessage("true")
	return json.Marshal(fields)
}

// replicaColumnAdmission checks storage wrappers incrementally, before a client
// column is finalized on disk. Computed-column files contain executable Scheme;
// hiding one below a blob overlay must not bypass the same trust boundary.
// Only a fixed-size header is retained, even for legacy inline blob payloads.
type replicaColumnAdmission struct {
	header []byte
	stage  uint8
	blobs  uint64
	skip   uint64
	depth  int
	done   bool
}

func (check *replicaColumnAdmission) write(data []byte) error {
	for len(data) > 0 && !check.done {
		if check.stage == 3 {
			n := min(uint64(len(data)), check.skip)
			data, check.skip = data[n:], check.skip-n
			if check.skip != 0 {
				continue
			}
			check.blobs--
			check.stage = 2
			if check.blobs == 0 {
				check.stage = 0
			}
			continue
		}
		need := 1
		if check.stage == 1 {
			need = 15
		}
		if check.stage == 2 {
			need = 40
		}
		n := min(len(data), need-len(check.header))
		check.header = append(check.header, data[:n]...)
		data = data[n:]
		if len(check.header) != need {
			continue
		}
		switch check.stage {
		case 0:
			magic := check.header[0]
			if _, known := storages[magic]; !known || magic == 50 {
				return fmt.Errorf("replication slave cannot supply storage format %d", magic)
			}
			if magic == 31 {
				check.depth++
				if check.depth > 16 {
					return fmt.Errorf("excessive replicated column wrappers")
				}
				check.stage = 1
			} else {
				check.done = true
			}
		case 1:
			version := check.header[0]
			if version != 0 && version != 1 && version != '1' {
				return fmt.Errorf("invalid replicated blob overlay version")
			}
			check.blobs = binary.LittleEndian.Uint64(check.header[7:])
			check.stage = 0
			if check.blobs != 0 {
				check.stage = 2
			}
		case 2:
			check.skip = binary.LittleEndian.Uint64(check.header[32:])
			check.stage = 3
		}
		check.header = check.header[:0]
	}
	// A final zero-length inline blob still completes its header transition.
	if check.stage == 3 && check.skip == 0 {
		check.blobs--
		check.stage = 2
		if check.blobs == 0 {
			check.stage = 0
		}
	}
	return nil
}
