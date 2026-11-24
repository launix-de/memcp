/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "sort"
import "slices"
import "unsafe"
import "strings"
import "encoding/json"
import "unicode/utf8"
import "github.com/google/uuid"
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

func columnDeclarationOperation(operation string) bool {
	return operation == "declaration" || operation == "validated_declaration"
}

// This profile changes an ordinary stored descriptor, never its physical slot
// or values. The frontend decides which declaration transitions its language
// permits. Native admission separately proves exact scalar/carrier identity.
func (declaration storedColumnType) replaceValidated(previous *column) *column {
	if declaration.Type == "" || declaration.Collation != previous.Collation {
		panic("validated declaration requires a type and unchanged comparison policy")
	}
	for _, dimension := range declaration.Dimensions {
		if dimension < -1 {
			panic("invalid validated declaration dimension")
		}
	}
	replacement := cloneStoredColumnDeclaration(previous)
	replacement.Typ = declaration.Type
	replacement.Typdimensions = append([]int(nil), declaration.Dimensions...)
	replacement.AllowNull = declaration.Nullable
	replacement.sanitizer = nil
	// UpdateSanitizer also resets the previous declaration's recovery hook.
	replacement.UpdateSanitizer()
	return replacement
}

func sameStoredScalar(before, after scm.Scmer) bool {
	if before.GetTag() != after.GetTag() {
		return false
	}
	if before.IsString() || before.IsBufferString() {
		return scm.String(before) == scm.String(after)
	}
	if before.IsDate() {
		return before.Int() == after.Int() && before.DateNanosecond() == after.DateNanosecond() && before.DateZone() == after.DateZone()
	}
	beforePointer, beforeBits := before.RawWords()
	afterPointer, afterBits := after.RawWords()
	return beforePointer == afterPointer && beforeBits == afterBits
}

// A caller may attest the exact input recipe it inspected before requesting a
// declaration. Compare under the existing endpoint lease and amendment mutex,
// not against the caller's earlier metadata read. This transient witness is
// neither a SQL DEFAULT policy nor part of the accepted journal format.
func requireExpectedStoredColumnInput(previous *column, expected scm.Scmer) {
	if !expected.IsSlice() || len(expected.Slice()) != 4 {
		panic("column input recipe requires value, expression, name and deferred flag")
	}
	parts := expected.Slice()
	if !parts[1].IsString() || !parts[2].IsString() || !parts[3].IsBool() {
		panic("invalid expected column input recipe")
	}
	if !sameStoredScalar(previous.Default, parts[0]) || previous.DefaultExpression != parts[1].String() ||
		previous.DefaultName != parts[2].String() || previous.DefaultDeferred != parts[3].Bool() {
		panic(declarationValidationViolation("value-preserving-declaration", previous.Name, "column declaration lost its expected input recipe"))
	}
}

func requireStoredScalarIdentity(target *column, value scm.Scmer) {
	converted := value
	if target.sanitizer != nil {
		converted = target.sanitizer(value)
	}
	if !sameStoredScalar(value, converted) {
		panic(declarationValidationViolation("value-preserving-declaration", target.Name, "column declaration would convert an existing native value"))
	}
	// Modern persistent scalar frames preserve the exact native value. A new
	// descriptor's legacy recovery adapter must retain that same value too.
	if target.recoverPersisted != nil && !value.IsNil() && !sameStoredScalar(value, target.recoverPersisted(value)) {
		panic("column declaration changes its native recovery carrier")
	}
}

// Constant descriptor payloads still use the established schema JSON boundary.
// Prove its target reader, rather than assuming a runtime sanitizer proves the
// persisted representation. Opaque deferred recipes remain frontend-owned.
func requireStoredDeclarationConstants(previous, target *column, projection scm.Scmer) {
	check := func(value scm.Scmer) {
		if value.IsNil() {
			return
		}
		requireStoredValueProjection(projection, target.Name, value)
		requireStoredScalarIdentity(target, value)
		encoded, err := json.Marshal(value)
		var recovered scm.Scmer
		if err != nil || json.Unmarshal(encoded, &recovered) != nil {
			panic("column declaration has an unsupported persisted constant")
		}
		if !sameStoredScalar(value, recovered) {
			panic("validated declaration requires a lossless native schema constant")
		}
		if target.recoverPersisted != nil && !sameStoredScalar(value, target.recoverPersisted(recovered)) {
			panic("column declaration would change a persisted constant carrier")
		}
	}
	if !previous.DefaultDeferred && previous.DefaultExpression == "" {
		check(previous.Default)
	}
	if len(previous.InitialValues) != 0 {
		check(previous.InitialValue)
	}
}

// A language may supply a transient scalar-domain proof, never a conversion.
// Its closed procedure is borrowed only during preparation, outside mutation
// locks, and is not retained by the accepted native descriptor or journal.
func requireStoredValueProjection(projection scm.Scmer, name string, value scm.Scmer) {
	if !projection.IsNil() && !sameStoredScalar(value, scm.Apply(projection, value)) {
		panic(declarationValidationViolation("value-preserving-declaration", name, "column value projection would convert an existing native value"))
	}
}

func closeStoredValueProjection(value scm.Scmer) scm.Scmer {
	value = value.WithoutSourceInfo()
	if !value.IsProc() {
		panic("column value projection requires a one-argument procedure")
	}
	parameters := value.Proc().Params.WithoutSourceInfo()
	if !parameters.IsSlice() || len(parameters.Slice()) != 1 {
		panic("column value projection requires a one-argument procedure")
	}
	return scm.CompileJIT(scm.CloseProcedure(value), true)
}

func validateStoredColumnValues(tx *TxContext, t *table, previous, target *column, projection scm.Scmer) {
	requireStoredDeclarationConstants(previous, target, projection)
	validate := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		requireStoredValueProjection(projection, target.Name, values[1])
		requireStoredScalarIdentity(target, values[1])
		return scm.NewBool(true)
	})
	t.scan(tx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil,
		scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), []string{previous.Name}, validate, scm.NewBool(true),
		scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), false)
}

// A validated endpoint's descriptors form one ordered native program. Keep
// every intermediate declaration so recovery proves A->B->C, never a weaker
// collapsed A->C claim, and sees the surrounding key/FK/name actions in order.
func (tx *TxContext) validatedDeclarationTables(history []catalogDeclarationAmendment) map[*table]bool {
	var result map[*table]bool
	for _, entry := range history {
		if a := entry.column; a != nil && a.validatedDeclaration && !tx.ownsTableDrop(a.table) {
			if result == nil {
				result = make(map[*table]bool)
			}
			result[a.table] = true
		}
	}
	return result
}

func (tx *TxContext) hasValidatedColumnDeclaration(target *table, name string) bool {
	for _, entry := range tx.publicForeignAmendments() {
		if a := entry.column; a != nil && a.validatedDeclaration && a.table == target && a.declaration.Column == name {
			return true
		}
	}
	return false
}

func (record *catalogDecisionRecord) hasValidatedDeclarations() bool {
	for _, action := range record.NativeDeclarations {
		if action.Declaration != nil {
			return true
		}
	}
	return false
}

func (record *catalogDecisionRecord) validateValidatedDeclarationProfile() error {
	endpoints := make(map[string]bool)
	for _, action := range record.NativeDeclarations {
		if action.Declaration != nil {
			endpoints[action.Declaration.Table] = true
		}
	}
	for _, declaration := range record.ColumnDeclarations {
		if endpoints[declaration.Table] {
			return fmt.Errorf("ordered column declarations cannot also use collapsed endpoint contracts")
		}
	}
	return nil
}

func validateStoredDeclarationShape(declaration *catalogColumnDeclaration) error {
	if declaration == nil || declaration.Table == "" || declaration.Column == "" ||
		!utf8.ValidString(declaration.Table) || !utf8.ValidString(declaration.Column) {
		return fmt.Errorf("invalid native stored declaration action")
	}
	if identity, err := uuid.Parse(declaration.Identity); err != nil || identity == uuid.Nil {
		return fmt.Errorf("invalid native stored declaration identity")
	}
	for _, shape := range []storedColumnType{declaration.Before, declaration.After} {
		if shape.Type == "" || !utf8.ValidString(shape.Type) || !utf8.ValidString(shape.Collation) {
			return fmt.Errorf("invalid native stored declaration contract")
		}
		for _, dimension := range shape.Dimensions {
			if dimension < -1 {
				return fmt.Errorf("invalid native stored declaration dimension")
			}
		}
	}
	if declaration.Before.Collation != declaration.After.Collation {
		return fmt.Errorf("validated stored declaration changes native comparison policy")
	}
	return nil
}

// This is the same ordered descriptor projection used by native names and
// properties. Value proof belongs to preparation; replay validates the complete
// accepted descriptor chain before changing any header.
func (db *database) projectStoredDeclaration(d *catalogColumnDeclaration, columns []*column, projection *catalogNativeProjection) (*table, []*column) {
	if err := validateStoredDeclarationShape(d); err != nil {
		panic(err)
	}
	t := db.tables.Get(d.Table)
	if t == nil || t.RuntimeIdentity != d.Identity || !t.committedAt(db.catalogView().version) {
		panic("stored declaration endpoint generation was replaced")
	}
	if columns == nil {
		columns = t.columnDeclarations()
	}
	for index, previous := range columns {
		if previous.Name != d.Column {
			continue
		}
		if !storedColumnTypeOf(previous).equal(d.Before) || previous.IsTemp || previous.ComputorDefinition != nil || isRuntimeComputedColumn(previous) {
			panic("stored declaration lost its exact predecessor")
		}
		for _, c := range columns {
			if c.ComputorDefinition != nil && slices.Contains(c.ComputorInputCols, d.Column) {
				panic("stored declaration is required by a native computation")
			}
		}
		for _, key := range uniqueKeysInView(t, projection.keys) {
			if slices.Contains(key.Cols, d.Column) {
				panic("stored declaration is required by a native key")
			}
		}
		for _, group := range projection.foreign {
			for _, foreign := range group {
				if foreign.Foreign.Tbl1 == t.Name && slices.Contains(foreign.Foreign.Cols1, d.Column) ||
					foreign.Foreign.Tbl2 == t.Name && slices.Contains(foreign.Foreign.Cols2, d.Column) {
					panic("stored declaration is required by a native foreign key")
				}
			}
		}
		if t.checkColumnDependency(d.Column) != nil {
			panic("validated declaration requires an unchanged native check directory")
		}
		var replacement *column
		if d.After.compatibleWith(d.Before) {
			replacement = d.After.replace(previous)
		} else {
			replacement = d.After.replaceValidated(previous)
			requireStoredDeclarationConstants(previous, replacement, scm.NewNil())
		}
		updated := append([]*column(nil), columns...)
		updated[index] = replacement
		return t, updated
	}
	panic("stored declaration physical column was removed")
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
	constraints := snapshot.nativeConstraints().Slice()
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
	history := tx.publicForeignAmendments()
	ordered := tx.validatedDeclarationTables(history)
	for _, entry := range history {
		if amendment := entry.column; amendment != nil && amendment.declaration != nil && !tx.ownsTableDrop(amendment.table) {
			if ordered[amendment.table] {
				continue
			}
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
