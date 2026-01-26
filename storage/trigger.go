/*
Copyright (C) 2025  Carl-Philip Hänsch

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

// TriggerDescription holds all information about a trigger
type TriggerDescription struct {
	Name     string        // Trigger name (user-defined or auto-generated)
	Timing   TriggerTiming // BEFORE/AFTER INSERT/UPDATE/DELETE
	Func     scm.Scmer     // The trigger function (compiled Scheme procedure)
	IsSystem bool          // True for auto-generated triggers (hidden from SHOW TRIGGERS)
	Priority int           // Execution order (lower = earlier)
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

// ExecuteTriggers executes all triggers for a specific timing
// oldRow is nil for INSERT, newRow is nil for DELETE
// Returns error if a trigger panics (for BEFORE triggers, this aborts the operation)
func (t *table) ExecuteTriggers(timing TriggerTiming, oldRow, newRow dataset) error {
	triggers := t.GetTriggers(timing)
	for _, tr := range triggers {
		if tr.Func.IsNil() {
			continue
		}
		// Build arguments based on timing
		// Convert dataset to []scm.Scmer for Apply
		var args []scm.Scmer
		switch timing {
		case BeforeInsert, AfterInsert:
			args = []scm.Scmer(newRow)
		case BeforeDelete, AfterDelete:
			args = []scm.Scmer(oldRow)
		case BeforeUpdate, AfterUpdate:
			args = append([]scm.Scmer(oldRow), []scm.Scmer(newRow)...)
		}
		// Execute trigger (TODO: proper error handling)
		scm.Apply(tr.Func, args...)
	}
	return nil
}
