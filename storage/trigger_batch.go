/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

import "sync"

// triggerBatch collects rows for deferred batch trigger execution.
// Used by DML operations to avoid firing triggers per-row.
type triggerBatch struct {
	mu     sync.Mutex
	timing TriggerTiming
	rows   []dataset
	table  *table
	isOld  bool // true for DELETE (rows are OLD), false for INSERT (rows are NEW)
}

// BeginTriggerBatch starts collecting trigger rows for the given timing.
// Returns a batch handle. Call Flush() when the DML operation is complete.
func (t *table) BeginTriggerBatch(timing TriggerTiming, isOld bool) *triggerBatch {
	return &triggerBatch{
		timing: timing,
		table:  t,
		isOld:  isOld,
	}
}

// Add appends a row to the batch. Thread-safe.
func (b *triggerBatch) Add(row dataset) {
	b.mu.Lock()
	b.rows = append(b.rows, row)
	b.mu.Unlock()
}

// Flush fires all collected triggers as a batch.
// If vectorized triggers exist, they receive the entire batch at once.
// Otherwise falls back to per-row execution.
func (b *triggerBatch) Flush() {
	b.mu.Lock()
	rows := b.rows
	b.rows = nil
	b.mu.Unlock()

	if len(rows) == 0 {
		return
	}
	b.table.ExecuteTriggersBatch(b.timing, rows, b.isOld)
}

// Len returns the current batch size.
func (b *triggerBatch) Len() int {
	b.mu.Lock()
	n := len(b.rows)
	b.mu.Unlock()
	return n
}
