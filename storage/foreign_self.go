/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "slices"
import "unsafe"
import "github.com/launix-de/memcp/scm"

// Endpoint ownership is physical, while the child and parent column contracts
// remain distinct even when both roles belong to the same native table.
func foreignEndpointTables(child, parent *table) []*table {
	if child == parent {
		return []*table{child}
	}
	return []*table{child, parent}
}

func requireSelfForeignModes(child, parent *table, fk foreignKey) {
	if child == parent && (fk.Updatemode != RESTRICT || fk.Deletemode != RESTRICT) {
		panic("self-referencing foreign declarations require RESTRICT for both actions")
	}
}

func sameForeignDeclaration(a, b foreignKey) bool {
	return a.Id == b.Id && a.Tbl1 == b.Tbl1 && a.Tbl2 == b.Tbl2 &&
		slices.Equal(a.Cols1, b.Cols1) && slices.Equal(a.Cols2, b.Cols2) &&
		a.Updatemode == b.Updatemode && a.Deletemode == b.Deletemode
}

func foreignMetadataValue(row scm.Scmer, name string) scm.Scmer {
	fields := row.Slice()
	for i := 0; i+1 < len(fields); i += 2 {
		if fields[i].IsString() && fields[i].String() == name {
			return fields[i+1]
		}
	}
	return scm.NewNil()
}

func selfForeignColumns(value scm.Scmer) []string {
	if !value.IsSlice() || len(value.Slice()) == 0 {
		panic("invalid self foreign declaration columns")
	}
	columns := make([]string, len(value.Slice()))
	for i, name := range value.Slice() {
		if !name.IsString() {
			panic("invalid self foreign declaration column")
		}
		columns[i] = name.String()
	}
	return columns
}

// The directory borrows rows/ID strings from the SAME immutable constraints
// publication. Conflicting IDs remain ambiguous (nil), never arbitrary wins.
type tableForeignMetadata struct {
	byID     map[string]scm.Scmer
	capacity uint // allocation hint (not Go bucket capacity), including colliding IDs
}

func buildForeignMetadataDirectory(constraints scm.Scmer) *tableForeignMetadata {
	rows := constraints.Slice()[3].Slice()
	if len(rows) == 0 {
		return nil
	}
	directory := &tableForeignMetadata{byID: make(map[string]scm.Scmer, len(rows)), capacity: uint(len(rows))}
	for _, row := range rows {
		id := foreignMetadataValue(row, "Id").String()
		if _, exists := directory.byID[id]; exists {
			directory.byID[id] = scm.NewNil()
		} else {
			directory.byID[id] = row
		}
	}
	return directory
}

func (directory *tableForeignMetadata) metadataMemory() uint {
	// Reuse the native map estimate, including base/minimum bucket allocation.
	// The retained allocation hint includes colliding IDs; it is not an exact
	// bucket capacity. Row/ID payloads are borrowed at the constraint owner.
	return uint(unsafe.Sizeof(*directory)) + publicCatalogMapFootprint(int(directory.capacity))
}

func foreignStoredComparisonColumn(column *column) bool {
	return column != nil && !column.IsTemp && column.ComputorDefinition == nil && !isRuntimeComputedColumn(column)
}

// Schema-entry/loader eligibility only, never a row callback walk. Recognized
// old self records outside this profile remain readable with mutation guards.
func supportsSelfForeign(t *table, fk foreignKey) bool {
	if fk.Updatemode != RESTRICT || fk.Deletemode != RESTRICT || fk.Id == "" || len(fk.Cols1) == 0 || len(fk.Cols1) != len(fk.Cols2) {
		return false
	}
	for _, columns := range [][]string{fk.Cols1, fk.Cols2} {
		seen := make(map[string]bool, len(columns))
		for _, name := range columns {
			if seen[name] || !foreignStoredComparisonColumn(t.uniqueColumnDeclaration(name)) {
				return false
			}
			seen[name] = true
		}
	}
	for _, row := range t.Constraints().Slice()[1].Slice() {
		if slices.Equal(selfForeignColumns(foreignMetadataValue(row, "Cols")), fk.Cols2) {
			return true
		}
	}
	return false
}

// Point lookup from one immutable catalog publication; no schema/UNIQUE scan,
// lazy metadata work or lock in the row callback. Admission plus endpoint/key
// dependency guards preserve the declared comparison contract.
func requireSelfForeign(tx *TxContext, t *table, id string, childColumns, parentColumns []string) {
	snapshot := t.catalogVisibleMetadata()
	if snapshot == nil || snapshot.metadata.foreignByID == nil {
		panic("self foreign recipe requires its native declaration")
	}
	row := snapshot.metadata.foreignByID.byID[id]
	if row.IsNil() {
		panic("self foreign recipe requires its unambiguous native declaration")
	}
	name := t.clusterPublishedName()
	other := foreignMetadataValue(row, "OtherTable").String()
	if t.schema.GetTableForTx(name, false, tx) != t || t.schema.GetTableForTx(other, false, tx) != t ||
		foreignMetadataValue(row, "Role").String() != "referencing" ||
		!slices.Equal(selfForeignColumns(foreignMetadataValue(row, "LocalColumns")), childColumns) ||
		!slices.Equal(selfForeignColumns(foreignMetadataValue(row, "OtherColumns")), parentColumns) ||
		foreignMetadataValue(row, "UpdateMode").String() != "restrict" ||
		foreignMetadataValue(row, "DeleteMode").String() != "restrict" {
		panic("self foreign recipe does not match its native declaration")
	}
}

func foreignTupleMatches(declarations []*column, a, b []scm.Scmer) bool {
	if len(a) != len(declarations) || len(b) != len(declarations) {
		panic("self foreign tuple does not match its native unique key")
	}
	for i, declaration := range declarations {
		// NULL child components are admitted by the caller. A NULL parent key
		// never supplies a candidate for an actual non-NULL child reference.
		if a[i].IsNil() || b[i].IsNil() || !uniqueColumnValuesEqual(declaration, a[i], b[i]) {
			return false
		}
	}
	return true
}

// The optional OLD identity is the referenced native UNIQUE tuple, never a
// guessed recid. Keep the existing scan/access/visibility kernel and add only
// its complete residual exclusion. Input columns are bounded declaration data.
func (lookup *foreignKeyLookup) existsExceptUnique(tx *TxContext, values []scm.Scmer, identityColumns []string, oldIdentity []scm.Scmer) bool {
	if oldIdentity == nil {
		return lookup.exists(tx, values)
	}
	identity := newForeignKeyLookup(lookup.table, identityColumns)
	if len(oldIdentity) != len(identity.declarations) {
		panic("self foreign OLD identity does not match its native unique key")
	}
	key, condition := lookup.predicate(values)
	if key == nil {
		return false
	}
	columns := append([]string(nil), lookup.columns...)
	positions := make([]int, len(identityColumns))
	for i, name := range identityColumns {
		position := slices.Index(columns, name)
		if position == -1 {
			position = len(columns)
			columns = append(columns, name)
		}
		positions[i] = position
	}
	matchReference := condition.Func()
	predicate := scm.NewFunc(func(row ...scm.Scmer) scm.Scmer {
		if !scm.ToBool(matchReference(row[:len(lookup.columns)]...)) {
			return scm.NewBool(false)
		}
		for i, position := range positions {
			if row[position].IsNil() || oldIdentity[i].IsNil() ||
				!uniqueColumnValuesEqual(identity.declarations[i], row[position], oldIdentity[i]) {
				return scm.NewBool(true)
			}
		}
		return scm.NewBool(false)
	})
	return lookup.table.scanExists(tx, lookup.access, key, columns, predicate)
}

// The old immediate producer appended a self declaration to both endpoints.
// Plan exact normalization before touching any loaded callbacks or metadata;
// conflicting same-ID data is never repaired by choosing an arbitrary copy.
func loadedSelfForeignDeclarations(db *database) map[*table][]foreignKey {
	result := make(map[*table][]foreignKey)
	for _, t := range db.tables.GetAll() {
		self := make(map[string]foreignKey)
		for _, fk := range t.Foreign {
			if fk.Tbl1 == t.Name && fk.Tbl2 == t.Name {
				if previous, exists := self[fk.Id]; exists && !sameForeignDeclaration(previous, fk) {
					panic("conflicting loaded self foreign declarations: " + fk.Id)
				}
				self[fk.Id] = fk
			}
		}
		if len(self) == 0 {
			continue
		}
		seen := make(map[string]bool)
		foreign := make([]foreignKey, 0, len(t.Foreign))
		for _, fk := range t.Foreign {
			if expected, isSelf := self[fk.Id]; isSelf {
				if !sameForeignDeclaration(expected, fk) {
					panic("conflicting loaded self foreign declarations: " + fk.Id)
				}
				if seen[fk.Id] {
					continue
				}
				seen[fk.Id] = true
			}
			foreign = append(foreign, fk)
		}
		result[t] = foreign
	}
	return result
}
