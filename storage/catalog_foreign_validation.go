/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

// Only declaration recipes are visited, never table data, programs or shards.
// Presence fences old journal domains; the semantic bit selects the new writer.
func (record *catalogDecisionRecord) foreignValidationFacts() (present, untrusted bool) {
	record.visitForeignDeclarations(func(key foreignKey) {
		present = present || key.validationStatePresent
		untrusted = untrusted || key.Untrusted
	})
	return
}

func (record *catalogDecisionRecord) foreignEnforcementFacts() (present, disabled bool) {
	record.visitForeignDeclarations(func(key foreignKey) {
		present = present || key.enforcementStatePresent
		disabled = disabled || key.Disabled
	})
	return
}

// The same finite recipe directory is used by both strict version fences.
func (record *catalogDecisionRecord) visitForeignDeclarations(observe func(foreignKey)) {
	for _, table := range record.Tables {
		if table == nil {
			continue // the existing table-shape validator owns malformed tables
		}
		for _, key := range table.Foreign {
			observe(key)
		}
	}
	for _, declarations := range [][]catalogForeignDeclaration{record.ForeignKeys, record.ForeignDrops} {
		for _, declaration := range declarations {
			observe(declaration.Foreign)
		}
	}
	for _, action := range record.ForeignHistory {
		observe(action.Declaration.Foreign)
	}
	for _, action := range record.NativeDeclarations {
		if action.Foreign != nil {
			observe(action.Foreign.Foreign)
		}
	}
}
