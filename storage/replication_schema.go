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

// validateReplicaSchema admits declarations from clients, never executable
// runtime artifacts. Inspect inert JSON first: Scmer decoding restores callable
// lambdas and custom runtime values. Trusted cluster and local persistence loads
// continue to use their existing codec without this restriction.
func validateReplicaSchema(data []byte) ([]byte, error) {
	// Query caches are local, rebuildable code artifacts rather than replicated
	// declarations. Remove them as inert JSON before admitting any client code.
	var err error
	data, err = replicaSchemaDeclarations(data)
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
func replicaSchemaDeclarations(data []byte) ([]byte, error) {
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
			}
			if err := json.Unmarshal(trigger, &declaration); err != nil {
				return nil, err
			}
			if !declaration.RequiresTarget && !restoredTriggerRequiresRuntimeTarget(declaration.Name) {
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
	return json.Marshal(db)
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
