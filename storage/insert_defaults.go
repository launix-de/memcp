/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "github.com/launix-de/memcp/scm"

// InsertOptions binds invocation-owned values without changing a transaction.
// CalculatorContext is borrowed immutable data, never persisted or retained.
type InsertOptions struct {
	Tx                *TxContext
	CalculatorContext *scm.Scmer
}

// Materialize only callable recipes before constraints, routing and WAL build
// their row images. Literal and clock defaults retain the native insertion
// path. Never alter a prepared caller's column or row slices.
func (t *table) assignInsertDefaults(columns []string, rows [][]scm.Scmer, context scm.Scmer) ([]string, [][]scm.Scmer) {
	var omitted []*column
	for _, col := range t.columnDeclarations() {
		if col.DefaultCalculator == nil {
			continue
		}
		present := false
		for _, name := range columns {
			if name == col.Name {
				present = true
				break
			}
		}
		if !present {
			omitted = append(omitted, col)
		}
	}
	if len(omitted) == 0 || len(rows) == 0 {
		return columns, rows
	}
	newColumns := append([]string(nil), columns...)
	for _, col := range omitted {
		newColumns = append(newColumns, col.Name)
	}
	// Prepared callback state belongs to this serial materialization invocation,
	// never to a published declaration or parallel shard worker.
	var calculators []scm.SerialProc
	for i, col := range omitted {
		if col.DefaultCalculator != nil {
			if calculators == nil {
				calculators = make([]scm.SerialProc, len(omitted))
			}
			calculators[i] = scm.PrepareSerialProc(*col.DefaultCalculator)
		}
	}
	var calculatorArgs []scm.Scmer
	if !context.IsNil() {
		calculatorArgs = []scm.Scmer{context}
	}
	newRows := make([][]scm.Scmer, len(rows))
	for i, row := range rows {
		if len(row) != len(columns) {
			panic("INSERT column/value count mismatch")
		}
		newRows[i] = append([]scm.Scmer(nil), row...)
		for j, col := range omitted {
			value := col.Default
			if col.DefaultCalculator != nil {
				value = calculators[j].Call(calculatorArgs)
			}
			newRows[i] = append(newRows[i], value)
		}
	}
	return newColumns, newRows
}
