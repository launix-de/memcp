/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "sort"
import "slices"
import "unsafe"
import "strings"
import "github.com/launix-de/memcp/scm"

// A declaration changes the value contract of an existing native column, not
// its name, packed delta position, main storage or physical generation.
type storedColumnType struct {
	Type       string `json:"type"`
	Dimensions []int  `json:"dimensions"`
	Nullable   bool   `json:"nullable"`
	Collation  string `json:"collation"`
}

type catalogColumnDeclaration struct {
	Table    string           `json:"table"`
	Identity string           `json:"identity"`
	Column   string           `json:"column"`
	Before   storedColumnType `json:"before"`
	After    storedColumnType `json:"after"`
}

func (declaration *catalogColumnDeclaration) memory() uint {
	if declaration == nil {
		return 0
	}
	return uint(unsafe.Sizeof(*declaration)) + uint(len(declaration.Table)+len(declaration.Identity)+len(declaration.Column)+len(declaration.Before.Type)+len(declaration.Before.Collation)+len(declaration.After.Type)+len(declaration.After.Collation)) +
		uint(cap(declaration.Before.Dimensions)+cap(declaration.After.Dimensions))*uint(unsafe.Sizeof(int(0)))
}

func (t *table) columnDeclarationsForTx(tx *TxContext) []*column {
	if names := t.columnNamesForTx(tx); names != nil {
		return names.declarations
	}
	return t.Columns // private constructor/load owner only
}

// Cold declaration consumers borrow the exact immutable endpoint view already
// published by this transaction's amendment owner. No mutable header is swapped
// to validate keys or FK declarations against a pending column contract.
func (tx *TxContext) publicDeclarationViews() map[*table]*tableShowColumnsSnapshot {
	if changes := tx.publicCatalogChanges(); changes != nil {
		if views := changes.views.Load(); views != nil {
			return views.tables
		}
	}
	return nil
}

func catalogColumnsInView(t *table, views map[*table]*tableShowColumnsSnapshot) []*column {
	if snapshot := views[t]; snapshot != nil && snapshot.metadata.columnNames != nil {
		return snapshot.metadata.columnNames.declarations
	}
	return t.columnDeclarations()
}

func catalogColumnInView(t *table, name string, views map[*table]*tableShowColumnsSnapshot) *column {
	for _, c := range catalogColumnsInView(t, views) {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func storedColumnTypeOf(c *column) storedColumnType {
	return storedColumnType{c.Typ, append([]int(nil), c.Typdimensions...), c.AllowNull, c.Collation}
}

func storedColumnFilterSchema(collation string, columns []*column) uint64 {
	filterSchema := plannerFingerprintString(0x1d26d8e71, collation)
	for _, c := range columns {
		filterSchema = plannerFingerprintString(filterSchema, c.Name)
		filterSchema = plannerFingerprintString(filterSchema, c.Typ)
		filterSchema = plannerFingerprintString(filterSchema, c.Collation)
		for _, dimension := range c.Typdimensions {
			filterSchema = plannerFingerprintMix(filterSchema, uint64(dimension))
		}
		filterSchema = plannerFingerprintMix(filterSchema, uint64(len(c.Typdimensions)))
		for _, label := range c.EnumValues {
			filterSchema = plannerFingerprintString(filterSchema, label)
		}
	}
	return filterSchema
}

func (declaration storedColumnType) equal(other storedColumnType) bool {
	return declaration.Type == other.Type && declaration.Nullable == other.Nullable &&
		declaration.Collation == other.Collation && slices.Equal(declaration.Dimensions, other.Dimensions)
}

func decodeStoredColumnType(value scm.Scmer) storedColumnType {
	if !value.IsSlice() || len(value.Slice()) != 4 {
		panic("column declaration requires type, dimensions, nullability and collation")
	}
	parts := value.Slice()
	if !parts[0].IsString() || !parts[1].IsSlice() || !parts[2].IsBool() || !parts[3].IsString() {
		panic("invalid column declaration shape")
	}
	dimensions := make([]int, len(parts[1].Slice()))
	for i, dimension := range parts[1].Slice() {
		if !dimension.IsInt() || int64(int(dimension.Int())) != dimension.Int() {
			panic("column declaration dimensions must be integers")
		}
		dimensions[i] = int(dimension.Int())
	}
	return storedColumnType{parts[0].String(), dimensions, parts[2].Bool(), parts[3].String()}
}

// Only value-preserving, same-slot transitions are part of this owner. The
// frontend retains its dialect checks; native callers cannot borrow that proof.
// Tightening NULL additionally requires the current transaction's row scan.
func (after storedColumnType) compatibleWith(before storedColumnType) bool {
	if after.Type != before.Type || after.Collation != before.Collation {
		return false
	}
	if slices.Equal(after.Dimensions, before.Dimensions) {
		return true
	}
	compatible := false
	switch strings.ToUpper(after.Type) {
	case "VARCHAR", "NVARCHAR", "VARBINARY":
		if len(after.Dimensions) == 1 && len(before.Dimensions) == 1 {
			oldWidth, width := before.Dimensions[0], after.Dimensions[0]
			compatible = oldWidth > 0 && (width == -1 || width >= oldWidth)
		}
	case "DECIMAL", "NUMERIC":
		if len(after.Dimensions) == 2 && len(before.Dimensions) == 2 {
			compatible = before.Dimensions[0] > 0 && before.Dimensions[1] >= 0 &&
				before.Dimensions[1] <= before.Dimensions[0] && after.Dimensions[0] >= before.Dimensions[0] &&
				after.Dimensions[1] == before.Dimensions[1]
		}
	}
	return compatible
}

func (after storedColumnType) requireCompatible(before storedColumnType) {
	if !after.compatibleWith(before) {
		panic("transactional column declaration requires the same native type, comparison policy and value-preserving dimensions")
	}
}

func (declaration storedColumnType) replace(previous *column) *column {
	replacement := cloneStoredColumnDeclaration(previous)
	// Compatibility retained these exact strings; keep their established
	// native header ownership instead of retaining a request's equal strings.
	replacement.Typdimensions = append([]int(nil), declaration.Dimensions...)
	replacement.AllowNull = declaration.Nullable
	replacement.sanitizer = nil
	replacement.UpdateSanitizer()
	return replacement
}

// Dependency rows and columns are from the same effective native publication,
// including this transaction's pending constraints. A committed-only lookup
// would miss a dependency staged earlier in the same original transaction.
func requireStoredColumnDependencies(t *table, tx *TxContext, name string) {
	snapshot := t.declarationMetadataForTx(tx)
	for _, c := range snapshot.metadata.columnNames.declarations {
		if c.ComputorDefinition != nil && slices.Contains(c.ComputorInputCols, name) {
			panic("column is an input of a declared computation: " + name)
		}
	}
	contains := func(row scm.Scmer, field string) bool {
		columns := foreignMetadataValue(row, field)
		if !columns.IsSlice() {
			panic("column dependency requires a complete native constraint declaration")
		}
		for _, value := range columns.Slice() {
			if value.String() == name {
				return true
			}
		}
		return false
	}
	constraints := snapshot.metadata.constraints.Slice()
	for i := 0; i < len(constraints); i += 2 {
		kind, field := constraints[i].String(), ""
		switch kind {
		case "Unique":
			field = "Cols"
		case "ForeignKeys":
			field = "LocalColumns"
		case "Checks":
			field = "Columns"
		default:
			panic("column declaration requires a supported native constraint directory")
		}
		for _, row := range constraints[i+1].Slice() {
			if contains(row, field) {
				panic("column declaration is required by " + kind + " " + foreignMetadataValue(row, "Id").String() + ": " + name)
			}
		}
	}
}

// The declaration lease is retained before this scan and through publication.
// The ordinary scan owns visibility and reader cleanup, including this same T's
// inserts, replacement rows and deletions. No SQL/default/trigger runs here.
func validateColumnNotNull(tx *TxContext, t *table, name string) {
	validate := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		if values[1].IsNil() {
			panic(declarationValidationViolation("not-null-declaration", name, fmt.Sprintf("column %s.%s.%s cannot be NULL for existing rows", t.schema.Name, t.Name, name)))
		}
		return scm.NewBool(true)
	})
	t.scan(tx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil,
		scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), []string{name}, validate, scm.NewBool(true),
		scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), false)
}

func (tx *TxContext) publicColumnDeclarations() []catalogColumnDeclaration {
	latest := make(map[string]catalogColumnDeclaration)
	for _, entry := range tx.publicForeignAmendments() {
		if amendment := entry.column; amendment != nil && amendment.declaration != nil && !tx.ownsTableDrop(amendment.table) {
			declaration := *amendment.declaration
			key := declaration.Table + "\x00" + declaration.Column
			if previous, found := latest[key]; found {
				// Recovery starts at the published predecessor, not an intermediate
				// private descriptor from the same transaction's amendment suffix.
				declaration.Before = previous.Before
			}
			latest[key] = declaration
		}
	}
	result := make([]catalogColumnDeclaration, 0, len(latest))
	for _, declaration := range latest {
		result = append(result, declaration)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Table == result[j].Table {
			return result[i].Column < result[j].Column
		}
		return result[i].Table < result[j].Table
	})
	return result
}

func validateCatalogColumnDeclaration(db *database, declaration catalogColumnDeclaration) (*table, int) {
	t := db.tables.Get(declaration.Table)
	if t == nil || t.RuntimeIdentity != declaration.Identity || !t.committedAt(db.catalogView().version) {
		panic("column declaration endpoint generation was replaced")
	}
	declaration.After.requireCompatible(declaration.Before)
	for i, current := range t.Columns {
		if current.Name != declaration.Column {
			continue
		}
		if current.IsTemp || current.ComputorDefinition != nil || isRuntimeComputedColumn(current) {
			panic("column declaration requires an ordinary stored column")
		}
		actual := storedColumnTypeOf(current)
		if !actual.equal(declaration.Before) && !actual.equal(declaration.After) {
			panic("column declaration lost its published predecessor")
		}
		return t, i
	}
	panic("column declaration endpoint was removed")
}

func (db *database) recoverColumnDeclarations(declarations []catalogColumnDeclaration) {
	for _, declaration := range declarations {
		t, index := validateCatalogColumnDeclaration(db, declaration)
		columns := append([]*column(nil), t.Columns...)
		columns[index] = declaration.After.replace(columns[index])
		t.Columns = columns
		// The cold loader's metadata consumer can precede its final schema pass,
		// exactly as for recovered CREATE and existing DEFAULT amendments.
		db.initializeRecoveryColumnMetadata(t)
	}
}

func (changes *publicCatalogChanges) validateColumnDeclarationsLocked() error {
	invocation := &TxContext{transactionState: changes.owner}
	for _, declaration := range invocation.publicColumnDeclarations() {
		validateCatalogColumnDeclaration(changes.db, declaration)
	}
	return nil
}
