/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "github.com/launix-de/memcp/scm"

// Native FK helpers run per parent row, not as child statement producers. This
// cold walk follows only potentially written edges before exposing the caller.
// A nonempty UPDATE descriptor is enforced at its physical assignment sink;
// ordinary BEFORE programs and indirectly computed keys widen that footprint.
func (operation *nativeMutationOperation) rejectStatementCascades(targets []*table, descriptor nativeMutationDescriptor) {
	type action struct {
		target *table
		event  nativeMutationEvent
	}
	type work struct {
		action
		columns nativeMutationAssignments // folded names; nil means unknown/all
	}
	footprint := func(columns []string) nativeMutationAssignments {
		if len(columns) == 0 {
			return nil
		}
		result := make(nativeMutationAssignments, len(columns))
		for _, column := range columns {
			result[scm.CaseFoldKey(column)] = struct{}{}
		}
		return result
	}
	pending := make([]work, 0, len(targets))
	for _, target := range targets {
		var columns nativeMutationAssignments
		if descriptor.event == nativeMutationUpdate {
			columns = footprint(descriptor.columns)
		}
		pending = append(pending, work{action{target, descriptor.event}, columns})
	}
	visited := make(map[action]nativeMutationAssignments)
	profile := nativeTriggerProfileFor(operation.invocation)
	for len(pending) != 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if current.target.cascadingForeignMutationEvents.Load()&(1<<current.event) == 0 {
			continue
		}
		operation.admitStatementTable(current.target)
		if current.event == nativeMutationUpdate && current.columns != nil {
			var before []TriggerDescription
			if input := operation.invocation.mutationInputFor(current.target); input != nil {
				before = input.beforeUpdate
			} else {
				current.target.mu.Lock()
				before = append(before, current.target.visibleTriggerDescriptionsLocked(operation.invocation)...)
				current.target.mu.Unlock()
			}
			for _, header := range before {
				// Ordinary row execution does not use Disabled. Do not infer a
				// stronger exclusion than its real selector; lazy headers stay broad.
				if header.Timing == BeforeUpdate && !header.needsFinalUpdateValues &&
					!(profile == nativeInternalTriggers && isUserTrigger(header)) {
					current.columns = nil
					break
				}
			}
		}
		previous, seen := visited[current.action]
		if seen {
			if previous == nil {
				continue
			}
			if current.columns != nil {
				widened := false
				for column := range current.columns {
					if _, exists := previous[column]; !exists {
						previous[column] = struct{}{}
						widened = true
					}
				}
				if !widened {
					continue
				}
				current.columns = previous
			}
		}
		visited[current.action] = current.columns
		columnNames := current.target.columnNamesForTx(operation.invocation)
		for _, key := range mutationForeignKeys(current.target, operation.invocation) {
			if key.Tbl2 != current.target.Name {
				continue
			}
			mode, childEvent := RESTRICT, nativeMutationUpdate
			switch current.event {
			case nativeMutationUpdate:
				mode = key.Updatemode
				if current.columns != nil {
					mayChange := false
					for _, column := range key.Cols2 {
						name := scm.CaseFoldKey(column)
						_, assigned := current.columns[name]
						if assigned || columnNames == nil {
							mayChange = true
							break
						}
						declared, exists := columnNames.folded[name]
						if !exists || declared.computed || isRuntimeComputedColumn(columnNames.declarations[declared.ordinal]) {
							mayChange = true
							break
						}
					}
					if !mayChange {
						continue
					}
				}
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
			func() {
				child.mu.Lock()
				defer child.mu.Unlock()
				for _, header := range child.visibleTriggerDescriptionsLocked(operation.invocation) {
					if !header.Timing.isStatement() || header.Disabled ||
						profile == nativeInternalTriggers && isUserTrigger(header) {
						continue
					}
					for _, declared := range header.StatementEvents {
						if declared == childEvent.String() {
							panic("native cascading action on a statement-triggered child is not supported")
						}
					}
				}
			}()
			var childColumns nativeMutationAssignments
			if childEvent == nativeMutationUpdate {
				childColumns = footprint(key.Cols1)
			}
			pending = append(pending, work{action{child, childEvent}, childColumns})
		}
	}
}
