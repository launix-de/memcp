/*
Copyright (C) 2025, 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import "errors"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

// TriggerTiming defines when a trigger fires
type TriggerTiming uint8

const (
	BeforeInsert TriggerTiming = iota
	AfterInsert
	BeforeUpdate
	AfterUpdate
	BeforeDelete
	AfterDelete
)

func (tt TriggerTiming) String() string {
	switch tt {
	case BeforeInsert:
		return "BEFORE INSERT"
	case AfterInsert:
		return "AFTER INSERT"
	case BeforeUpdate:
		return "BEFORE UPDATE"
	case AfterUpdate:
		return "AFTER UPDATE"
	case BeforeDelete:
		return "BEFORE DELETE"
	case AfterDelete:
		return "AFTER DELETE"
	default:
		return "UNKNOWN"
	}
}

func (tt TriggerTiming) MarshalJSON() ([]byte, error) {
	var s string
	switch tt {
	case BeforeInsert:
		s = "before_insert"
	case AfterInsert:
		s = "after_insert"
	case BeforeUpdate:
		s = "before_update"
	case AfterUpdate:
		s = "after_update"
	case BeforeDelete:
		s = "before_delete"
	case AfterDelete:
		s = "after_delete"
	default:
		return nil, errors.New("unknown trigger timing")
	}
	return json.Marshal(s)
}

func (tt *TriggerTiming) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "before_insert":
		*tt = BeforeInsert
	case "after_insert":
		*tt = AfterInsert
	case "before_update":
		*tt = BeforeUpdate
	case "after_update":
		*tt = AfterUpdate
	case "before_delete":
		*tt = BeforeDelete
	case "after_delete":
		*tt = AfterDelete
	default:
		return errors.New("unknown trigger timing: " + s)
	}
	return nil
}

// TriggerDescription holds all information about a trigger
type TriggerDescription struct {
	Name      string        `json:"name"`                 // Trigger name (user-defined or auto-generated)
	Timing    TriggerTiming `json:"timing"`               // BEFORE/AFTER INSERT/UPDATE/DELETE
	Func      scm.Scmer     `json:"func"`                 // The trigger function (compiled Scheme procedure)
	SourceSQL string        `json:"source_sql,omitempty"` // Original SQL body text (for SHOW TRIGGERS)
	IsSystem  bool          `json:"is_system,omitempty"`  // True for auto-generated triggers (hidden from SHOW TRIGGERS)
	Priority  int           `json:"priority,omitempty"`   // Execution order (lower = earlier)
}

// GetTriggers returns all triggers for a specific timing
func (t *table) GetTriggers(timing TriggerTiming) []TriggerDescription {
	result := make([]TriggerDescription, 0)
	for _, tr := range t.Triggers {
		if tr.Timing == timing {
			result = append(result, tr)
		}
	}
	return result
}

// AddTrigger adds a trigger to the table
func (t *table) AddTrigger(trigger TriggerDescription) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Triggers = append(t.Triggers, trigger)
}

// RemoveTrigger removes a trigger by name
func (t *table) RemoveTrigger(name string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, tr := range t.Triggers {
		if tr.Name == name {
			t.Triggers = append(t.Triggers[:i], t.Triggers[i+1:]...)
			return true
		}
	}
	return false
}

// rowToDict converts a dataset to a dict with column names
func (t *table) rowToDict(row dataset) scm.Scmer {
	if row == nil {
		return scm.NewNil()
	}
	fd := scm.NewFastDictValue(len(t.Columns))
	for i, col := range t.Columns {
		if i < len(row) {
			fd.Set(scm.NewString(col.Name), row[i], nil)
		}
	}
	return scm.NewFastDict(fd)
}

// dictToRow converts a dict back to a dataset using column order
func (t *table) dictToRow(dict scm.Scmer, columns []string) dataset {
	if dict.IsNil() {
		return nil
	}
	row := make(dataset, len(columns))
	if dict.IsFastDict() {
		fd := dict.FastDict()
		for i, col := range columns {
			if v, ok := fd.Get(scm.NewString(col)); ok {
				row[i] = v
			} else {
				row[i] = scm.NewNil()
			}
		}
	}
	return row
}

// ExecuteTriggers executes all triggers for a specific timing (AFTER triggers)
// oldRow is nil for INSERT, newRow is nil for DELETE
func (t *table) ExecuteTriggers(timing TriggerTiming, oldRow, newRow dataset) {
	triggers := t.GetTriggers(timing)
	for _, tr := range triggers {
		if tr.Func.IsNil() {
			continue
		}
		// Build arguments: pass OLD and NEW as dicts with column names
		var oldDict, newDict scm.Scmer = scm.NewNil(), scm.NewNil()
		switch timing {
		case BeforeInsert, AfterInsert:
			newDict = t.rowToDict(newRow)
		case BeforeDelete, AfterDelete:
			oldDict = t.rowToDict(oldRow)
		case BeforeUpdate, AfterUpdate:
			oldDict = t.rowToDict(oldRow)
			newDict = t.rowToDict(newRow)
		}
		// Execute trigger with (OLD NEW) arguments
		// OLD and NEW are dicts: {"col1": val1, "col2": val2, ...}
		scm.Apply(tr.Func, oldDict, newDict)
	}
}

// rowToDictWithColumns converts a dataset to a dict using explicit column names
func (t *table) rowToDictWithColumns(row dataset, columns []string) scm.Scmer {
	if row == nil {
		return scm.NewNil()
	}
	fd := scm.NewFastDictValue(len(columns))
	for i, col := range columns {
		if i < len(row) {
			fd.Set(scm.NewString(col), row[i], nil)
		}
	}
	return scm.NewFastDict(fd)
}

// ExecuteBeforeInsertTriggers executes BEFORE INSERT triggers and returns modified rows
// The trigger function can modify NEW values by returning a modified dict
func (t *table) ExecuteBeforeInsertTriggers(columns []string, values [][]scm.Scmer) [][]scm.Scmer {
	triggers := t.GetTriggers(BeforeInsert)
	if len(triggers) == 0 {
		return values
	}

	result := make([][]scm.Scmer, len(values))
	for i, row := range values {
		// Build dict using the columns that are being inserted
		newDict := t.rowToDictWithColumns(row, columns)
		// Execute all BEFORE INSERT triggers, each can modify NEW
		for _, tr := range triggers {
			if tr.Func.IsNil() {
				continue
			}
			// Trigger receives (OLD NEW), OLD is nil for INSERT
			returned := scm.Apply(tr.Func, scm.NewNil(), newDict)
			// If trigger returns a dict, use it as the new NEW
			if !returned.IsNil() && returned.IsFastDict() {
				newDict = returned
			}
		}
		// Convert modified dict back to row using same columns
		result[i] = t.dictToRow(newDict, columns)
	}
	return result
}

// ExecuteBeforeUpdateTriggers executes BEFORE UPDATE triggers
// oldRow: the current row values (all columns in table order)
// newRow: the row with changes applied (all columns in table order)
// Returns the modified newRow
func (t *table) ExecuteBeforeUpdateTriggers(oldRow, newRow dataset) dataset {
	triggers := t.GetTriggers(BeforeUpdate)
	if len(triggers) == 0 {
		return newRow
	}

	// Build column names from table
	columns := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		columns[i] = col.Name
	}

	oldDict := t.rowToDictWithColumns(oldRow, columns)
	newDict := t.rowToDictWithColumns(newRow, columns)

	// Execute all BEFORE UPDATE triggers
	for _, tr := range triggers {
		if tr.Func.IsNil() {
			continue
		}
		// Trigger receives (OLD NEW)
		returned := scm.Apply(tr.Func, oldDict, newDict)
		// If trigger returns a dict, use it as the new NEW
		if !returned.IsNil() && returned.IsFastDict() {
			newDict = returned
		}
	}

	// Convert modified dict back to row
	return t.dictToRow(newDict, columns)
}

// ExecuteBeforeDeleteTriggers executes BEFORE DELETE triggers
// oldRow: the row being deleted (all columns in table order)
// Returns true if delete should proceed, false to abort
func (t *table) ExecuteBeforeDeleteTriggers(oldRow dataset) bool {
	triggers := t.GetTriggers(BeforeDelete)
	if len(triggers) == 0 {
		return true
	}

	// Build column names from table
	columns := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		columns[i] = col.Name
	}

	oldDict := t.rowToDictWithColumns(oldRow, columns)

	// Execute all BEFORE DELETE triggers
	for _, tr := range triggers {
		if tr.Func.IsNil() {
			continue
		}
		// Trigger receives (OLD nil) for DELETE
		returned := scm.Apply(tr.Func, oldDict, scm.NewNil())
		// If trigger returns false/nil, abort delete
		if returned.IsNil() || (returned.IsBool() && !scm.ToBool(returned)) {
			return false
		}
	}

	return true
}
