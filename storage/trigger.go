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

import "fmt"
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
	AfterDropTable
	AfterDropColumn
	AfterInvalidate // fired when a computed column is invalidated; propagates cache invalidation
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
	case AfterDropTable:
		return "AFTER DROP TABLE"
	case AfterDropColumn:
		return "AFTER DROP COLUMN"
	case AfterInvalidate:
		return "AFTER INVALIDATE"
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
	case AfterDropTable:
		s = "after_drop_table"
	case AfterDropColumn:
		s = "after_drop_column"
	case AfterInvalidate:
		s = "after_invalidate"
	default:
		return nil, errors.New("unknown trigger timing")
	}
	return json.Marshal(s)
}

func (tt *TriggerTiming) UnmarshalJSON(data []byte) error {
	// Try string first (new format)
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
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
		case "after_drop_table":
			*tt = AfterDropTable
		case "after_drop_column":
			*tt = AfterDropColumn
		case "after_invalidate":
			*tt = AfterInvalidate
		default:
			return errors.New("unknown trigger timing: " + s)
		}
		return nil
	}
	// Fall back to numeric (legacy format)
	var n uint8
	if err := json.Unmarshal(data, &n); err != nil {
		return errors.New("trigger timing must be string or number")
	}
	if n > uint8(AfterInvalidate) {
		return fmt.Errorf("unknown trigger timing number: %d", n)
	}
	*tt = TriggerTiming(n)
	return nil
}

// TriggerDescription holds all information about a trigger
type TriggerDescription struct {
	Name      string        `json:"name"`                 // Trigger name (user-defined or auto-generated)
	Timing    TriggerTiming `json:"timing"`               // BEFORE/AFTER INSERT/UPDATE/DELETE
	Func      scm.Scmer     `json:"func"`                 // The trigger function (compiled Scheme procedure)
	SourceSQL string        `json:"source_sql,omitempty"` // Original SQL body text (for SHOW TRIGGERS)
	IsSystem  bool          `json:"is_system,omitempty"`  // True for Go-internal triggers (FK etc.) — not persisted via createtrigger
	Hidden     bool          `json:"hidden,omitempty"`     // True for Scheme-internal triggers — persisted but hidden from SHOW TRIGGERS
	Priority   int           `json:"priority,omitempty"`   // Execution order (lower = earlier)
	Async      bool          `json:"async,omitempty"`      // Run trigger in background goroutine (fire-and-forget, no transaction context)
	VectorFunc scm.Scmer     `json:"-"`                    // Vectorized trigger: (lambda (OLD_batch NEW_batch) ...) for batch execution
}

// GetTriggers returns all triggers for a specific timing
func (t *table) GetTriggers(timing TriggerTiming) []TriggerDescription {
	t.mu.Lock()
	result := make([]TriggerDescription, 0, len(t.Triggers))
	for _, tr := range t.Triggers {
		if tr.Timing == timing {
			result = append(result, tr)
		}
	}
	t.mu.Unlock()
	return result
}

// AddTrigger adds a trigger to the table. Automatically attempts to vectorize
// the trigger for batch execution (DELETE/INSERT patterns on prejoin tables).
func (t *table) AddTrigger(trigger TriggerDescription) {
	// Auto-vectorize: try to produce a batch-aware version of the trigger
	if trigger.VectorFunc.IsNil() && !trigger.Func.IsNil() {
		if vf := VectorizeTrigger(trigger.Func); !vf.IsNil() {
			trigger.VectorFunc = vf
		}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	// Keep trigger list ordered by priority (lower = earlier). For equal
	// priorities preserve registration order by inserting after existing ties.
	insertAt := len(t.Triggers)
	for i, tr := range t.Triggers {
		if tr.Priority > trigger.Priority {
			insertAt = i
			break
		}
	}
	t.Triggers = append(t.Triggers, TriggerDescription{})
	copy(t.Triggers[insertAt+1:], t.Triggers[insertAt:])
	t.Triggers[insertAt] = trigger
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

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (t *table) beforeInsertOutputColumns(dict scm.Scmer, columns []string) []string {
	result := make([]string, 0, len(t.Columns))
	seen := make(map[string]struct{}, len(columns))
	for _, col := range columns {
		result = append(result, col)
		seen[col] = struct{}{}
	}
	if !dict.IsFastDict() {
		return result
	}
	fd := dict.FastDict()
	for _, col := range t.Columns {
		if _, ok := seen[col.Name]; ok {
			continue
		}
		if _, ok := fd.Get(scm.NewString(col.Name)); ok {
			result = append(result, col.Name)
		}
	}
	return result
}

// ExecuteTriggers executes all triggers for a specific timing (AFTER triggers).
// oldRow is nil for INSERT, newRow is nil for DELETE.
// If a transaction is active, a savepoint is created before each trigger;
// on panic the savepoint is rolled back before re-panicking.
// Triggers with Async=true are launched in a background goroutine (fire-and-forget).
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
		case AfterInvalidate:
			// no row data — column-level invalidation propagation
		}
		if tr.Async {
			// Fire-and-forget: run in background goroutine, no transaction context
			trFunc := tr.Func
			trName := tr.Name
			tName := t.Name
			go func() {
				defer func() {
					recover() // async triggers must not crash the process
					_ = trName
					_ = tName
				}()
				scm.Apply(trFunc, oldDict, newDict)
			}()
			continue
		}
		// Execute trigger with savepoint for proper rollback
		func() {
			tx := CurrentTx()
			var sp Savepoint
			hasSavepoint := false
			if tx != nil {
				sp = tx.CreateSavepoint()
				hasSavepoint = true
			}
			defer func() {
				if r := recover(); r != nil {
					if hasSavepoint {
						tx.RollbackToSavepoint(sp)
					}
					panic(fmt.Sprintf("trigger %s (%s) on %s failed: %v", tr.Name, timing, t.Name, r))
				}
			}()
			scm.Apply(tr.Func, oldDict, newDict)
		}()
	}
}

// ExecuteTriggersBatch fires triggers once per trigger with a batch of rows.
// For triggers that have a vectorized form (VectorFunc), the batch is passed
// as a single call. For non-vectorized triggers, falls back to per-row execution.
func (t *table) ExecuteTriggersBatch(timing TriggerTiming, rows []dataset, isOld bool) {
	if len(rows) == 0 {
		return
	}
	if len(rows) == 1 {
		// Single row: use the normal path
		if isOld {
			t.ExecuteTriggers(timing, rows[0], nil)
		} else {
			t.ExecuteTriggers(timing, nil, rows[0])
		}
		return
	}
	triggers := t.GetTriggers(timing)
	for _, tr := range triggers {
		if tr.Func.IsNil() {
			continue
		}
		if tr.Async {
			// Async: fire per-row (no batching for fire-and-forget)
			for _, row := range rows {
				var oldDict, newDict scm.Scmer = scm.NewNil(), scm.NewNil()
				if isOld {
					oldDict = t.rowToDict(row)
				} else {
					newDict = t.rowToDict(row)
				}
				trFunc := tr.Func
				go func() {
					defer func() { recover() }()
					scm.Apply(trFunc, oldDict, newDict)
				}()
			}
			continue
		}
		// Check for vectorized trigger (VectorFunc set)
		if !tr.VectorFunc.IsNil() {
			// Build batch dicts
			dicts := make([]scm.Scmer, len(rows))
			for i, row := range rows {
				dicts[i] = t.rowToDict(row)
			}
			batchList := scm.NewSlice(dicts)
			func() {
				tx := CurrentTx()
				var sp Savepoint
				hasSavepoint := false
				if tx != nil {
					sp = tx.CreateSavepoint()
					hasSavepoint = true
				}
				defer func() {
					if r := recover(); r != nil {
						if hasSavepoint {
							tx.RollbackToSavepoint(sp)
						}
						panic(fmt.Sprintf("vectorized trigger %s (%s) on %s failed: %v", tr.Name, timing, t.Name, r))
					}
				}()
				if isOld {
					scm.Apply(tr.VectorFunc, batchList, scm.NewNil())
				} else {
					scm.Apply(tr.VectorFunc, scm.NewNil(), batchList)
				}
			}()
			continue
		}
		// Fallback: per-row execution
		for _, row := range rows {
			var oldDict, newDict scm.Scmer = scm.NewNil(), scm.NewNil()
			if isOld {
				oldDict = t.rowToDict(row)
			} else {
				newDict = t.rowToDict(row)
			}
			func() {
				tx := CurrentTx()
				var sp Savepoint
				hasSavepoint := false
				if tx != nil {
					sp = tx.CreateSavepoint()
					hasSavepoint = true
				}
				defer func() {
					if r := recover(); r != nil {
						if hasSavepoint {
							tx.RollbackToSavepoint(sp)
						}
						panic(fmt.Sprintf("trigger %s (%s) on %s failed: %v", tr.Name, timing, t.Name, r))
					}
				}()
				scm.Apply(tr.Func, oldDict, newDict)
			}()
		}
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

// ExecuteTableLifecycleTriggers executes AfterDropTable or AfterDropColumn triggers.
// These are non-row-level triggers: OLD and NEW are both nil.
func (t *table) ExecuteTableLifecycleTriggers(timing TriggerTiming) {
	triggers := t.GetTriggers(timing)
	for _, tr := range triggers {
		if tr.Func.IsNil() {
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					// lifecycle triggers are best-effort; log but don't propagate
				}
			}()
			scm.Apply(tr.Func, scm.NewNil(), scm.NewNil())
		}()
	}
}

// ExecuteBeforeInsertTriggers executes BEFORE INSERT triggers and returns modified rows.
// The trigger function can modify NEW values by returning a modified dict, including
// columns not present in the original INSERT.
// When isIgnore is true, rows whose triggers panic are silently skipped
// and any partial transaction effects are rolled back via savepoints.
// When isIgnore is false, trigger panics propagate to the caller.
func (t *table) executeBeforeInsertTriggerRow(columns []string, row dataset, isIgnore bool) ([]string, dataset, bool) {
	triggers := t.GetTriggers(BeforeInsert)
	if len(triggers) == 0 {
		return columns, row, true
	}

	// Build dict using the columns that are being inserted.
	newDict := t.rowToDictWithColumns(row, columns)
	triggerOk := true
	for _, tr := range triggers {
		if tr.Func.IsNil() {
			continue
		}
		if isIgnore {
			// Per-row savepoint + panic recovery for INSERT IGNORE.
			func() {
				tx := CurrentTx()
				var sp Savepoint
				hasSavepoint := false
				if tx != nil {
					sp = tx.CreateSavepoint()
					hasSavepoint = true
				}
				defer func() {
					if r := recover(); r != nil {
						if hasSavepoint {
							tx.RollbackToSavepoint(sp)
						}
						triggerOk = false
					}
				}()
				returned := scm.Apply(tr.Func, scm.NewNil(), newDict)
				if !returned.IsNil() && returned.IsFastDict() {
					newDict = returned
				}
			}()
			if !triggerOk {
				break
			}
		} else {
			// Normal mode: savepoint for proper rollback on propagated panic.
			func() {
				tx := CurrentTx()
				var sp Savepoint
				hasSavepoint := false
				if tx != nil {
					sp = tx.CreateSavepoint()
					hasSavepoint = true
				}
				defer func() {
					if r := recover(); r != nil {
						if hasSavepoint {
							tx.RollbackToSavepoint(sp)
						}
						panic(fmt.Sprintf("trigger %s (BEFORE INSERT) on %s failed: %v", tr.Name, t.Name, r))
					}
				}()
				returned := scm.Apply(tr.Func, scm.NewNil(), newDict)
				if !returned.IsNil() && returned.IsFastDict() {
					newDict = returned
				}
			}()
		}
	}
	if !triggerOk {
		return nil, nil, false
	}
	rowColumns := t.beforeInsertOutputColumns(newDict, columns)
	return rowColumns, t.dictToRow(newDict, rowColumns), true
}

func (t *table) ExecuteBeforeInsertTriggers(columns []string, values [][]scm.Scmer, isIgnore bool) ([]string, [][]scm.Scmer) {
	triggers := t.GetTriggers(BeforeInsert)
	if len(triggers) == 0 {
		return columns, values
	}

	resultColumns := append([]string(nil), columns...)
	rowColumns := make([][]string, 0, len(values))
	result := make([][]scm.Scmer, 0, len(values))
	for _, row := range values {
		newColumns, newRow, ok := t.executeBeforeInsertTriggerRow(columns, row, isIgnore)
		if !ok {
			continue
		}
		if len(result) == 0 {
			resultColumns = append([]string(nil), newColumns...)
		}
		rowColumns = append(rowColumns, newColumns)
		result = append(result, newRow)
	}
	if len(result) == 0 {
		return resultColumns, result
	}
	for _, cols := range rowColumns[1:] {
		if !stringSlicesEqual(resultColumns, cols) {
			allColumns := make([]string, len(t.Columns))
			for i, col := range t.Columns {
				allColumns[i] = col.Name
			}
			for i, row := range result {
				result[i] = t.dictToRow(t.rowToDictWithColumns(row, rowColumns[i]), allColumns)
			}
			return allColumns, result
		}
	}
	return resultColumns, result
}

// ExecuteBeforeUpdateTriggers executes BEFORE UPDATE triggers.
// oldRow: the current row values (all columns in table order)
// newRow: the row with changes applied (all columns in table order)
// Returns the modified newRow. Panics from triggers propagate to the caller.
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
		func() {
			tx := CurrentTx()
			var sp Savepoint
			hasSavepoint := false
			if tx != nil {
				sp = tx.CreateSavepoint()
				hasSavepoint = true
			}
			defer func() {
				if r := recover(); r != nil {
					if hasSavepoint {
						tx.RollbackToSavepoint(sp)
					}
					panic(fmt.Sprintf("trigger %s (BEFORE UPDATE) on %s failed: %v", tr.Name, t.Name, r))
				}
			}()
			returned := scm.Apply(tr.Func, oldDict, newDict)
			if !returned.IsNil() && (returned.IsFastDict() || returned.IsSlice()) {
				newDict = returned
			}
		}()
	}

	// Convert modified dict back to row
	return t.dictToRow(newDict, columns)
}

// ExecuteBeforeDeleteTriggers executes BEFORE DELETE triggers.
// oldRow: the row being deleted (all columns in table order)
// Returns true if delete should proceed, false to abort.
// Panics from triggers propagate to the caller.
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
		var returned scm.Scmer
		func() {
			tx := CurrentTx()
			var sp Savepoint
			hasSavepoint := false
			if tx != nil {
				sp = tx.CreateSavepoint()
				hasSavepoint = true
			}
			defer func() {
				if r := recover(); r != nil {
					if hasSavepoint {
						tx.RollbackToSavepoint(sp)
					}
					panic(fmt.Sprintf("trigger %s (BEFORE DELETE) on %s failed: %v", tr.Name, t.Name, r))
				}
			}()
			returned = scm.Apply(tr.Func, oldDict, scm.NewNil())
		}()
		// If trigger explicitly returns false, abort delete.
		// nil return (side-effect-only triggers) does NOT abort.
		if returned.IsBool() && !scm.ToBool(returned) {
			return false
		}
	}

	return true
}
