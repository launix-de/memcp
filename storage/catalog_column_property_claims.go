/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"
import "unsafe"
import "unicode/utf8"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// V20 property actions belong to the existing ordered native declaration
// interpreter. V17 keeps its original independent removal-only reader.
type catalogColumnPropertyChange struct {
	Table         string           `json:"table"`
	Identity      string           `json:"identity"`
	Column        string           `json:"column"`
	Key           string           `json:"key"`
	Expected      storedColumnType `json:"expected"`
	ValueKind     string           `json:"value_kind"`
	Value         scm.Scmer        `json:"value"`
	BeforePresent bool             `json:"before_present"`
	BeforeKind    string           `json:"before_kind,omitempty"`
	Before        scm.Scmer        `json:"before"`
}

func columnPropertyChange(t *table, previous *column, key string, value scm.Scmer) *catalogColumnPropertyChange {
	before, present := previous.DeclarationProperties[key]
	d := &catalogColumnPropertyChange{Table: t.Name, Identity: t.RuntimeIdentity, Column: previous.Name,
		Key: key, Expected: storedColumnTypeOf(previous), ValueKind: columnPropertyValueKind(value), Value: value,
		BeforePresent: present, Before: before}
	if present {
		d.BeforeKind = columnPropertyValueKind(before)
	}
	return d
}

func (d *catalogColumnPropertyChange) memory() uint {
	if d == nil {
		return 0
	}
	return uint(unsafe.Sizeof(*d)) + uint(len(d.Table)+len(d.Identity)+len(d.Column)+len(d.Key)+len(d.Expected.Type)+len(d.Expected.Collation)) +
		uint(cap(d.Expected.Dimensions))*uint(unsafe.Sizeof(int(0))) +
		scm.ComputeSize(d.Value) + scm.ComputeSize(d.Before) - 2*uint(unsafe.Sizeof(scm.Scmer{}))
}

func validateColumnPropertyChangeShape(d *catalogColumnPropertyChange) error {
	if d == nil || d.Table == "" || d.Column == "" || d.Key == "" || !utf8.ValidString(d.Table) || !utf8.ValidString(d.Column) || !utf8.ValidString(d.Key) {
		return fmt.Errorf("invalid native column property action")
	}
	if id, err := uuid.Parse(d.Identity); err != nil || id == uuid.Nil {
		return fmt.Errorf("invalid native column property identity")
	}
	if d.Expected.Type == "" || !utf8.ValidString(d.Expected.Type) || !utf8.ValidString(d.Expected.Collation) {
		return fmt.Errorf("invalid native column property declaration")
	}
	for _, dimension := range d.Expected.Dimensions {
		if dimension < -1 {
			return fmt.Errorf("invalid native column property dimension")
		}
	}
	value, err := columnPropertyValue(d.ValueKind, d.Value)
	if err != nil {
		return err
	}
	if err = validateColumnProperty(d.Key, value); err != nil {
		return err
	}
	if d.BeforePresent {
		before, err := columnPropertyValue(d.BeforeKind, d.Before)
		if err != nil {
			return err
		}
		return validateColumnProperty(d.Key, before)
	}
	if d.BeforeKind != "" || !d.Before.IsNil() {
		return fmt.Errorf("absent native column property has a predecessor value")
	}
	return nil
}

func (tx *TxContext) hasColumnPropertyClaims() bool {
	for _, entry := range tx.publicForeignAmendments() {
		if a := entry.column; a != nil && a.propertyChange != nil && a.propertyRemoval == nil && !tx.ownsTableDrop(a.table) {
			return true
		}
	}
	return false
}

// A whole-table retirement supersedes its own property changes. Surviving
// property actions on another endpoint cannot enter the existing retirement
// journal domain; reject that composition before publishing either new delta.
func (tx *TxContext) hasSurvivingColumnPropertiesExcept(retiring *table) bool {
	for _, entry := range tx.publicForeignAmendments() {
		if a := entry.column; a != nil && a.table != retiring && !tx.ownsTableDrop(a.table) &&
			(a.propertyChange != nil || a.propertyRemoval != nil) {
			return true
		}
	}
	return false
}

func (tx *TxContext) requireColumnPropertyRetirementProfile() {
	if tx.hasTableDrops() {
		panic("column properties cannot join an unrelated table retirement decision")
	}
}

// This initial property profile does not manufacture ordering proofs for a
// changed physical/type layout. DEFAULT recipes are independent attributes and
// remain composable through the same immutable stored-column history.
func (tx *TxContext) requireColumnPropertyLayout(t *table, claiming bool) {
	for _, entry := range tx.publicForeignAmendments() {
		a := entry.column
		if a == nil || a.table != t {
			continue
		}
		if claiming && (a.declaration != nil || a.appendDeclaration != nil) ||
			!claiming && a.propertyChange != nil && a.propertyRemoval == nil {
			panic("column property claims with changed column layouts require ordered layout admission")
		}
	}
}

func (db *database) projectColumnPropertyChange(d *catalogColumnPropertyChange, operation string, columns []*column) (*table, []*column) {
	if err := validateColumnPropertyChangeShape(d); err != nil {
		panic(err)
	}
	t := db.tables.Get(d.Table)
	if t == nil || t.RuntimeIdentity != d.Identity || !t.committedAt(db.catalogView().version) {
		panic("column property endpoint generation was replaced")
	}
	if columns == nil {
		columns = t.Columns
	}
	for i, previous := range columns {
		if previous.Name != d.Column {
			continue
		}
		if !storedColumnTypeOf(previous).equal(d.Expected) {
			panic("column property action lost its expected declaration")
		}
		before, present := previous.DeclarationProperties[d.Key]
		if present != d.BeforePresent {
			panic("column property action lost its predecessor")
		}
		if present {
			expected, err := columnPropertyValue(d.BeforeKind, d.Before)
			if err != nil || !sameColumnProperty(before, expected) {
				panic("column property action lost its predecessor value")
			}
		}
		value, _ := columnPropertyValue(d.ValueKind, d.Value)
		var replacement *column
		switch operation {
		case "column_property_claim":
			replacement = t.claimColumnProperty(columns, i, columnPropertyClaim{d.Key, value, d.Expected})
		case "column_property_drop":
			if !present || !sameColumnProperty(before, value) {
				panic("column property retirement lost its predecessor")
			}
			replacement = cloneStoredColumnDeclaration(previous)
			replacement.DeclarationProperties = maps.Clone(previous.DeclarationProperties)
			delete(replacement.DeclarationProperties, d.Key)
		default:
			panic("invalid native column property operation")
		}
		updated := append([]*column(nil), columns...)
		updated[i] = replacement
		return t, updated
	}
	panic("column property endpoint column was removed")
}

func (record *catalogDecisionRecord) hasColumnPropertyClaimActions() bool {
	for _, action := range record.NativeDeclarations {
		if action.Operation == "column_property_claim" {
			return true
		}
	}
	return false
}

func (record *catalogDecisionRecord) validateColumnPropertyProfile() error {
	if record.Format != 20 && record.Format != 22 && record.Format != 23 {
		return nil
	}
	claimed := make(map[string]bool)
	for _, action := range record.NativeDeclarations {
		if action.Property != nil {
			claimed[action.Property.Table] = true
		}
	}
	for _, declaration := range record.ColumnDeclarations {
		if claimed[declaration.Table] {
			return fmt.Errorf("column property claims require an unchanged endpoint column layout")
		}
	}
	for _, append := range record.ColumnAppends {
		if claimed[append.Table] {
			return fmt.Errorf("column property claims require an unchanged endpoint column layout")
		}
	}
	return nil
}
