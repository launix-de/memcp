/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"
import "sort"
import "unsafe"
import "unicode/utf8"
import "encoding/json"
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

func (tx *TxContext) hasColumnPropertyClaims(history []catalogDeclarationAmendment) bool {
	for _, entry := range history {
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
func (tx *TxContext) requireColumnPropertyLayout(t *table) {
	for _, entry := range tx.publicForeignAmendments() {
		a := entry.column
		if a == nil || a.table != t {
			continue
		}
		if a.propertyChange != nil && a.propertyRemoval == nil {
			panic("column property claims with changed column layouts require ordered layout admission")
		}
	}
}

type catalogColumnPropertyDecision struct {
	native   []catalogNativeDeclaration
	removals []catalogColumnPropertyRemoval
}

// The amendment owner (or its drained decision) owns this finite projection.
// Raw history remains the validation, private-view and savepoint authority.
// Only complete restoration cycles on actual append endpoints can disappear
// from a durable decision; no old record or reader acquires new semantics.
func (changes *publicCatalogChanges) projectColumnPropertyDecision(tx *TxContext, history []catalogDeclarationAmendment) catalogColumnPropertyDecision {
	result := catalogColumnPropertyDecision{tx.publicNativeDeclarations(history), tx.publicColumnPropertyRemovals(history)}
	if changes == nil || !tx.hasColumnPropertyClaims(history) {
		return result
	}
	if changes.owner != tx.transactionState {
		panic("column property projection lost its original declaration owner")
	}
	byTable := make(map[*table][]int)
	for i, entry := range history {
		if a := entry.column; a != nil && !tx.ownsTableDrop(a.table) {
			byTable[a.table] = append(byTable[a.table], i)
		}
	}
	elided := make(map[*catalogColumnPropertyChange]bool)
	for t, positions := range byTable {
		through := changes.restoredAppendProperties(tx, t, history, positions)
		for _, i := range positions {
			if i <= through && history[i].column.propertyChange != nil {
				elided[history[i].column.propertyChange] = true
			}
		}
	}
	if len(elided) == 0 {
		return result // preserve every existing writer profile byte-for-byte
	}
	native := make([]catalogNativeDeclaration, 0, len(result.native))
	claims := false
	for _, action := range result.native {
		if action.Property != nil && elided[action.Property] {
			continue
		}
		claims = claims || action.Operation == "column_property_claim"
		native = append(native, action)
	}
	result.native, result.removals = native, nil
	if claims {
		return result // V20 keeps its sole ordered property stream
	}
	// A restored prefix can leave genuine later removals. With no surviving
	// claim those belong to the existing V17 stream, not an invalid V15 action.
	result.native = make([]catalogNativeDeclaration, 0, len(native))
	for _, action := range native {
		if action.Operation == "column_property_drop" {
			d := action.Property
			encoded, err := json.Marshal(d.Before)
			var decoded scm.Scmer
			if err != nil || json.Unmarshal(encoded, &decoded) != nil || columnPropertyValueKind(decoded) != d.BeforeKind || !sameColumnProperty(decoded, d.Before) {
				panic("restored property survivor has no lossless removal-only encoding")
			}
			result.removals = append(result.removals, catalogColumnPropertyRemoval{d.Table, d.Identity, d.Column, d.Key, d.Before})
		} else {
			result.native = append(result.native, action)
		}
	}
	sort.Slice(result.removals, func(i, j int) bool {
		a, b := result.removals[i], result.removals[j]
		if a.Table != b.Table {
			return a.Table < b.Table
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return a.Key < b.Key
	})
	return result
}

func (changes *publicCatalogChanges) restoredAppendProperties(tx *TxContext, t *table, history []catalogDeclarationAmendment, positions []int) int {
	firstAppend, hasClaim := -1, false
	keys := make(map[string]bool)
	for _, i := range positions {
		a := history[i].column
		if a.declaration != nil {
			return -1 // no retype, nullable or collation predecessor reinterpretation
		}
		if a.appendDeclaration != nil && firstAppend < 0 {
			firstAppend = i
		}
		if d := a.propertyChange; d != nil {
			keys[d.Key] = true
			hasClaim = hasClaim || a.propertyRemoval == nil
		}
	}
	if firstAppend < 0 || !hasClaim {
		return -1 // in particular, never turn a property-only decision empty
	}
	lease := changes.leases[t]
	if lease == nil || lease.owner != &t.writeAdmission || lease.transaction != tx.transactionState || !lease.declaration || lease.ddlTable.Load() != t ||
		t.schema != changes.db || changes.db.tables.Get(t.Name) != t || !t.committedAt(changes.db.catalogView().version) {
		return -1
	}
	observed := t.showColumnsSnapshot.Load()
	if base, ok := changes.baselines[t]; !ok || base.declarations != observed.metadata.declarations {
		panic("restored column property lost its published predecessor")
	}
	baseline := observed.metadata.columnNames.declarations
	indices := make(map[string]int, len(baseline))
	for i, c := range baseline {
		indices[c.Name] = i
	}
	nextOrdinal := len(baseline)
	for _, i := range positions {
		a := history[i].column
		if len(a.columns) < len(baseline) {
			return -1
		}
		for j, before := range baseline {
			if a.columns[j].Name != before.Name || !storedColumnTypeOf(a.columns[j]).equal(storedColumnTypeOf(before)) {
				return -1
			}
		}
		if d := a.appendDeclaration; d != nil {
			if d.Ordinal != nextOrdinal {
				return -1
			}
			nextOrdinal++
			for key := range d.Properties {
				if keys[key] {
					return -1 // an appended declaration cannot change claim uniqueness
				}
			}
		}
		if d := a.propertyChange; d != nil {
			j, exists := indices[d.Column]
			if !exists || d.Identity != t.RuntimeIdentity || !d.Expected.equal(storedColumnTypeOf(baseline[j])) {
				return -1
			}
		}
	}
	current, through := baseline, -1
	retired := make(map[[2]string]bool)
	for _, i := range positions {
		a := history[i].column
		d := a.propertyChange
		if d == nil {
			continue
		}
		before, present := baseline[indices[d.Column]].DeclarationProperties[d.Key]
		key := [2]string{d.Column, d.Key}
		operation := "column_property_drop"
		if a.propertyRemoval == nil {
			operation = "column_property_claim"
			if !present || !retired[key] || d.BeforePresent || columnPropertyValueKind(before) != d.ValueKind || !sameColumnProperty(before, d.Value) {
				return -1 // an arbitrary new claim is not a restoration
			}
			retired[key] = false
		} else {
			retired[key] = present && columnPropertyValueKind(before) == d.BeforeKind && sameColumnProperty(before, d.Before)
		}
		// Reuse the actual property interpreter: validate every raw predecessor
		// and genuine uniqueness claim before it may be omitted from the wire.
		_, current = changes.db.projectColumnPropertyChange(d, operation, current)
		if a.propertyRemoval == nil && i > firstAppend {
			equal := true
			for j, c := range baseline {
				same := func(a, b scm.Scmer) bool {
					return columnPropertyValueKind(a) == columnPropertyValueKind(b) && sameColumnProperty(a, b)
				}
				equal = equal && maps.EqualFunc(c.DeclarationProperties, current[j].DeclarationProperties, same) &&
					maps.EqualFunc(c.DeclarationProperties, a.columns[j].DeclarationProperties, same)
			}
			if equal {
				through = i
			}
		}
	}
	return through
}

// Reject an unrepresentable mixed survivor profile before its amendment is
// published. These are the unchanged old record validators, not a new domain.
func (changes *publicCatalogChanges) requireColumnPropertyDecision(tx *TxContext, history []catalogDeclarationAmendment) {
	projection := changes.projectColumnPropertyDecision(tx, history)
	record := catalogDecisionRecord{Format: 20, NativeDeclarations: projection.native}
	for _, entry := range history {
		if a := entry.column; a != nil {
			if a.appendDeclaration != nil {
				record.ColumnAppends = append(record.ColumnAppends, *a.appendDeclaration)
			}
			if a.declaration != nil {
				record.ColumnDeclarations = append(record.ColumnDeclarations, *a.declaration)
			}
		}
	}
	if record.hasColumnPropertyClaimActions() {
		if err := record.validateColumnPropertyProfile(); err != nil {
			panic(err)
		}
	} else if err := validateColumnPropertyRemovalShape(projection.removals); err != nil {
		panic(err)
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
	if record.Format != 20 && record.Format != 21 && record.Format != 22 && record.Format != 23 && record.Format != 24 && record.Format != 25 && record.Format != 26 && record.Format != 28 && record.Format != 29 && record.Format != 30 {
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
	for _, action := range record.NativeDeclarations {
		if action.Declaration != nil && claimed[action.Declaration.Table] {
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
