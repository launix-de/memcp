/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "sort"
import "unsafe"
import "unicode/utf8"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// Property removal changes immutable declaration data only. The frontend owns
// the meaning of the key; neither its spelling nor its value is executable.
type catalogColumnPropertyRemoval struct {
	Table    string    `json:"table"`
	Identity string    `json:"identity"`
	Column   string    `json:"column"`
	Key      string    `json:"key"`
	Before   scm.Scmer `json:"before"`
}

func (declaration *catalogColumnPropertyRemoval) memory() uint {
	if declaration == nil {
		return 0
	}
	return uint(unsafe.Sizeof(*declaration)) + uint(len(declaration.Table)+len(declaration.Identity)+len(declaration.Column)+len(declaration.Key)) + scm.ComputeSize(declaration.Before) - uint(unsafe.Sizeof(scm.Scmer{}))
}

func (tx *TxContext) publicColumnPropertyRemovals(history []catalogDeclarationAmendment) []catalogColumnPropertyRemoval {
	if tx.hasColumnPropertyClaims(history) {
		return nil // V20 has one ordered property authority inside NativeDeclarations.
	}
	var result []catalogColumnPropertyRemoval
	for _, entry := range history {
		if amendment := entry.column; amendment != nil && amendment.propertyRemoval != nil && !tx.ownsTableDrop(amendment.table) {
			result = append(result, *amendment.propertyRemoval)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Table != result[j].Table {
			return result[i].Table < result[j].Table
		}
		if result[i].Column != result[j].Column {
			return result[i].Column < result[j].Column
		}
		return result[i].Key < result[j].Key
	})
	return result
}

func validateColumnPropertyRemovalShape(declarations []catalogColumnPropertyRemoval) error {
	seen := make(map[string]bool)
	for _, declaration := range declarations {
		key := declaration.Table + "\x00" + declaration.Column + "\x00" + declaration.Key
		if declaration.Table == "" || declaration.Column == "" || !utf8.ValidString(declaration.Table) || !utf8.ValidString(declaration.Column) || !utf8.ValidString(declaration.Key) || seen[key] {
			return fmt.Errorf("invalid native column property removal")
		}
		if id, err := uuid.Parse(declaration.Identity); err != nil || id == uuid.Nil {
			return fmt.Errorf("invalid native column property endpoint identity")
		}
		if err := validateColumnProperty(declaration.Key, declaration.Before); err != nil {
			return err
		}
		seen[key] = true
	}
	return nil
}

func validateCatalogColumnPropertyRemoval(db *database, declaration catalogColumnPropertyRemoval, recovering bool) (*table, int) {
	t := db.tables.Get(declaration.Table)
	if t == nil || t.RuntimeIdentity != declaration.Identity || !t.committedAt(db.catalogView().version) {
		panic("column property endpoint generation was replaced")
	}
	for i, current := range t.Columns {
		if current.Name != declaration.Column {
			continue
		}
		if current.IsTemp || current.ComputorDefinition != nil || isRuntimeComputedColumn(current) {
			panic("column property removal requires an ordinary stored column")
		}
		value, found := current.DeclarationProperties[declaration.Key]
		if !found && recovering {
			return t, i // an already applied durable decision
		}
		if !found || value.GetTag() != declaration.Before.GetTag() || !scm.Equal(value, declaration.Before) {
			panic("column property removal lost its published predecessor")
		}
		return t, i
	}
	panic("column property endpoint was removed")
}

func (db *database) recoverColumnPropertyRemovals(declarations []catalogColumnPropertyRemoval) {
	for _, declaration := range declarations {
		t, index := validateCatalogColumnPropertyRemoval(db, declaration, true)
		columns := append([]*column(nil), t.Columns...)
		columns[index] = cloneStoredColumnDeclaration(columns[index])
		columns[index].Alter("drop_property", scm.NewString(declaration.Key))
		t.Columns = columns
		db.initializeRecoveryColumnMetadata(t)
	}
}

func (changes *publicCatalogChanges) validateColumnPropertyRemovalsLocked() error {
	tx := &TxContext{transactionState: changes.owner}
	for _, declaration := range tx.publicColumnPropertyRemovals(tx.publicForeignAmendments()) {
		validateCatalogColumnPropertyRemoval(changes.db, declaration, false)
	}
	return nil
}
