/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

// The native FK helpers execute once per parent ROW. They are not statement
// producers for a child's statement handlers. Until that aggregate producer is
// owned, reject the affected shape before ANY outer row/default/callback effect.
// This transient walk follows only actual native Foreign declarations; it never
// inspects user callback code or persists a second dependency graph.
func (operation *nativeMutationOperation) rejectStatementCascades(targets []*table, event nativeMutationEvent) {
	type action struct {
		target *table
		event  nativeMutationEvent
	}
	pending := make([]action, 0, len(targets))
	for _, target := range targets {
		pending = append(pending, action{target, event})
	}
	visited := make(map[action]bool)
	for len(pending) != 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if visited[current] {
			continue
		}
		visited[current] = true
		if current.target.cascadingForeignMutationEvents.Load()&(1<<current.event) == 0 {
			continue
		}
		// The actual statement lease excludes semantic maintenance and FK
		// publication on this endpoint. No schema/shard/UNIQUE lock is held.
		operation.admitStatementTable(current.target)
		for _, key := range current.target.Foreign {
			if key.Tbl2 != current.target.Name {
				continue
			}
			mode, childEvent := RESTRICT, nativeMutationUpdate
			switch current.event {
			case nativeMutationUpdate:
				mode = key.Updatemode
			case nativeMutationDelete:
				mode = key.Deletemode
				if mode == CASCADE {
					childEvent = nativeMutationDelete
				}
			}
			if mode == RESTRICT {
				continue
			}
			child := current.target.schema.tables.Get(key.Tbl1)
			operation.admitStatementTable(child)
			// Inspect descriptive headers only, under their existing owner.
			// The same statement lease prevents a later handler publication.
			func() {
				child.mu.Lock()
				defer child.mu.Unlock()
				for _, header := range child.visibleTriggerDescriptionsLocked(operation.invocation) {
					if !header.Timing.isStatement() {
						continue
					}
					for _, declared := range header.StatementEvents {
						if declared == childEvent.String() {
							panic("native cascading action on a statement-triggered child is not supported")
						}
					}
				}
			}()
			pending = append(pending, action{child, childEvent})
		}
	}
}
