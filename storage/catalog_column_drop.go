/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"
import "slices"
import "unsafe"
import "unicode/utf8"
import "github.com/google/uuid"

// Retirement removes a declaration, not its historical physical storage. The
// original snapshot/undo and readers retain those values until native rebuild
// and generation reclamation no longer reference them.
type catalogColumnDrop struct {
	Table    string           `json:"table"`
	Identity string           `json:"identity"`
	Storage  string           `json:"storage"`
	Visible  string           `json:"visible"`
	Ordinal  int              `json:"ordinal"`
	Expected storedColumnType `json:"expected"`
}

func (d *catalogColumnDrop) memory() uint {
	if d == nil {
		return 0
	}
	return uint(unsafe.Sizeof(*d)) + uint(len(d.Table)+len(d.Identity)+len(d.Storage)+len(d.Visible)+len(d.Expected.Type)+len(d.Expected.Collation)) + uint(cap(d.Expected.Dimensions))*uint(unsafe.Sizeof(int(0)))
}

func validateColumnDropShape(d *catalogColumnDrop) error {
	if d == nil || d.Table == "" || d.Storage == "" || d.Visible == "" || d.Ordinal < 0 || !utf8.ValidString(d.Table) || !utf8.ValidString(d.Storage) || !utf8.ValidString(d.Visible) || d.Expected.Type == "" || !utf8.ValidString(d.Expected.Type) || !utf8.ValidString(d.Expected.Collation) {
		return fmt.Errorf("invalid native stored column retirement")
	}
	if id, err := uuid.Parse(d.Identity); err != nil || id == uuid.Nil {
		return fmt.Errorf("invalid native column retirement identity")
	}
	for _, n := range d.Expected.Dimensions {
		if n < -1 {
			return fmt.Errorf("invalid native column retirement dimension")
		}
	}
	return nil
}

func (tx *TxContext) columnRetirementTables(history []catalogDeclarationAmendment) map[*table]bool {
	result := make(map[*table]bool)
	for _, entry := range history {
		if a := entry.column; a != nil && a.drop != nil && !tx.ownsTableDrop(a.table) {
			result[a.table] = true
		}
	}
	return result
}

func (record *catalogDecisionRecord) hasColumnRetirements() bool {
	for _, a := range record.NativeDeclarations {
		if a.Drop != nil {
			return true
		}
	}
	return false
}

// Defaults on a retired endpoint are part of the same ordered program. A
// preceding DROP DEFAULT must be visible to DROP COLUMN during prepare/replay.
func projectOrderedColumnDefault(d *catalogColumnDefaultDeclaration, t *table, columns []*column) []*column {
	for i, c := range columns {
		if c.Name != d.Column {
			continue
		}
		if c.IsTemp || c.ComputorDefinition != nil || isRuntimeComputedColumn(c) {
			panic("ordered default requires an ordinary stored column")
		}
		replacement := cloneStoredColumnDeclaration(c)
		replacement.Default, replacement.DefaultExpression, replacement.DefaultName, replacement.DefaultDeferred = d.Value, d.Expression, d.Name, d.Deferred
		updated := append([]*column(nil), columns...)
		updated[i] = replacement
		return updated
	}
	panic("ordered default physical column was removed")
}

func (db *database) projectColumnDrop(d *catalogColumnDrop, columns []*column, projection *catalogNativeProjection) (*table, []*column) {
	if err := validateColumnDropShape(d); err != nil {
		panic(err)
	}
	t := db.tables.Get(d.Table)
	if t == nil || t.RuntimeIdentity != d.Identity || !t.committedAt(db.catalogView().version) {
		panic("column retirement endpoint generation was replaced")
	}
	if columns == nil {
		columns = t.columnDeclarations()
	}
	if d.Ordinal >= len(columns) {
		panic("column retirement lost its physical ordinal")
	}
	c := columns[d.Ordinal]
	if c.Name != d.Storage || c.displayName() != d.Visible || !storedColumnTypeOf(c).equal(d.Expected) {
		panic("column retirement lost its exact declaration predecessor")
	}
	if c.IsTemp || c.AutoIncrement || c.ComputorDefinition != nil || isRuntimeComputedColumn(c) || !c.OnUpdate.IsNil() || len(c.DeclarationProperties) != 0 {
		panic("column retirement requires an ordinary stored column")
	}
	if !c.Default.IsNil() || c.DefaultName != "" || c.DefaultExpression != "" || c.DefaultDeferred {
		panic("column retirement requires its default declaration to be removed")
	}
	if len(columns) == 1 {
		panic("column retirement cannot remove the final stored column")
	}
	for _, input := range columns {
		if slices.Contains(input.ComputorInputCols, d.Storage) {
			panic("column retirement is required by a native computation")
		}
	}
	for _, key := range uniqueKeysInView(t, projection.keys) {
		if slices.Contains(key.Cols, d.Storage) {
			panic("column retirement is required by a native key")
		}
	}
	for _, group := range projection.foreign {
		for _, fk := range group {
			if fk.Foreign.Tbl1 == t.Name && slices.Contains(fk.Foreign.Cols1, d.Storage) || fk.Foreign.Tbl2 == t.Name && slices.Contains(fk.Foreign.Cols2, d.Storage) {
				panic("column retirement is required by a native foreign key")
			}
		}
	}
	if t.checkColumnDependency(d.Storage) != nil {
		panic("column retirement requires an unchanged native check directory")
	}
	updated := append([]*column(nil), columns[:d.Ordinal]...)
	updated = append(updated, columns[d.Ordinal+1:]...)
	return t, updated
}

func (tx *TxContext) dropPublicColumn(t *table, name string, ifExists bool) bool {
	if tx == nil || tx.transactionState == nil || tx.State != TxActive || tx.coordinator != nil || tx.remoteObject != nil || tx.remoteInvocation || t == nil || activeReplication(t.schema) != nil || sharedClusterDatabase(t.schema.Name) || t.Ephemeral || t.ScopedSchema || t.PersistencyMode == Cache {
		panic("column retirement requires an original local native transaction")
	}
	finish := tx.beginCatalogConstruction(catalogValidationConstruction, nil, t)
	defer func() {
		failure := recover()
		finish()
		tx.retireUnacceptedPublicCatalogPhase()
		if failure != nil {
			panic(failure)
		}
	}()
	requireTableMaintenance(t.schema.Name, t.Name, maintenanceAlter)
	return tx.stagePublicCatalogDeclaration(t.schema, catalogColumnKind, nil, func(changes *publicCatalogChanges) bool { return changes.stageColumnDrop(tx, t, name, ifExists) })
}

func (changes *publicCatalogChanges) stageColumnDrop(tx *TxContext, t *table, name string, ifExists bool) bool {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	if tx.hasTableDrops() {
		panic("column retirement cannot join table retirement in this profile")
	}
	accepted := false
	finishLease := changes.acquireStoredColumnLease(tx, t, true)
	defer func() { finishLease(accepted) }()
	restore := t.guardCatalogDeclarationInput(tx.transactionState)
	defer func() {
		if !accepted {
			restore()
			changes.publishViews(tx.publicForeignAmendments(), changes.views.Load().tables)
		}
	}()
	lease := changes.leases[t]
	changes.mu.Unlock()
	func() { defer changes.mu.Lock(); lease.retainDDL(t, tx.transactionState) }()
	if changes.leases[t] != lease {
		panic("column retirement lost its declaration lease")
	}
	access := changes.db.GetWrite(tx)
	defer access.Release()
	changes.db.schemalock.Lock()
	defer changes.db.schemalock.Unlock()
	changes.validateConstraintStateEndpointsLocked(tx)
	if changes.db.tables.Get(t.Name) != t || !t.committedAt(changes.db.catalogView().version) || t.DroppedColumns == nil {
		panic("column retirement requires a committed stored generation")
	}
	observed := t.showColumnsSnapshot.Load()
	base := changes.views.Load().tables[t]
	if base == nil {
		base = observed
	}
	for _, entry := range tx.publicForeignAmendments() {
		if a := entry.column; a != nil && a.table == t && a.appendDeclaration != nil {
			panic("column retirement cannot join append on the same endpoint")
		}
	}
	t.mu.Lock()
	lifecycle := false
	for _, tr := range t.Triggers {
		if tr.Timing == AfterDropColumn {
			lifecycle = true
		}
	}
	t.mu.Unlock()
	if lifecycle {
		panic("transactional column retirement requires lifecycle consumers to be removed")
	}
	ordinal := -1
	for i, c := range base.metadata.columnNames.declarations {
		if c.Name == name {
			ordinal = i
			break
		}
	}
	if ordinal < 0 {
		if ifExists {
			return false
		}
		panic("drop column does not exist: " + t.Name + "." + name)
	}
	c := base.metadata.columnNames.declarations[ordinal]
	d := &catalogColumnDrop{t.Name, t.RuntimeIdentity, c.Name, c.displayName(), ordinal, storedColumnTypeOf(c)}
	// Reuse the same chronological native dependency interpreter as prepare and
	// recovery, including previous default/key/FK/type removals and changes.
	history := tx.publicForeignAmendments()
	projection := newCatalogNativeProjection(changes.db)
	if actions := tx.publicNativeDeclarations(history); len(actions) != 0 {
		projection = changes.db.projectNativeDeclarations(actions, tx.publicDeclarationViews(), catalogNativeCurrent, catalogForeignEndpointScope{owner: tx.transactionState})
	}
	_, columns := changes.db.projectColumnDrop(d, base.metadata.columnNames.declarations, projection)
	view, bytes := replaceStoredColumnMetadata(t, base, columns, observed.metadata.columnNames)
	a := &catalogStoredColumnAmendment{drop: d, table: t, columns: columns, previous: changes.views.Load().tables[t]}
	a.retainedBytes = uint(unsafe.Sizeof(*a)) + uint(cap(columns))*uint(unsafe.Sizeof((*column)(nil))) + bytes + d.memory()
	history = appendCatalogDeclarationHistory(history, catalogDeclarationAmendment{column: a})
	views := maps.Clone(changes.views.Load().tables)
	views[t] = view
	t.mu.Lock()
	func() {
		defer t.mu.Unlock()
		for _, tr := range t.Triggers {
			if tr.Timing == AfterDropColumn {
				panic("column retirement acquired a lifecycle consumer; retry declaration")
			}
		}
		if _, exists := changes.baselines[t]; !exists {
			changes.baselines[t] = foreignBaseline(observed)
		}
		changes.baselinesHighWater = max(changes.baselinesHighWater, len(changes.baselines))
		changes.viewsHighWater = max(changes.viewsHighWater, len(views))
		changes.amendmentBytes += a.retainedBytes
		changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
		tx.noteStatementMutation(t)
		tx.catalogAmendments.Store(&history)
	}()
	changes.publishViews(history, views)
	accepted = true
	return true
}

// V28 extends the ordered declaration interpreter only. No old writer/read
// domain acquires either new action, even if its payload is explicitly null.
func (record *catalogDecisionRecord) validateColumnRetirementProfile() error {
	retiring := make(map[string]bool)
	for _, action := range record.NativeDeclarations {
		if action.Drop != nil {
			retiring[action.Drop.Table] = true
		}
	}
	for _, action := range record.NativeDeclarations {
		if action.Default != nil && !retiring[action.Default.Table] {
			return fmt.Errorf("ordered default has no column retirement endpoint")
		}
	}
	for _, d := range record.ColumnDefaults {
		if retiring[d.Table] {
			return fmt.Errorf("retired endpoint defaults require chronological native actions")
		}
	}
	for _, d := range record.ColumnAppends {
		if retiring[d.Table] {
			return fmt.Errorf("column retirement cannot join append on the same endpoint")
		}
	}
	for _, d := range record.ColumnPropertyRemovals {
		if retiring[d.Table] {
			return fmt.Errorf("column retirement properties require chronological native actions")
		}
	}
	return nil
}

func (record *catalogDecisionRecord) activeOrderedColumnDefaults() []catalogColumnDefaultDeclaration {
	active := make(map[[2]string]catalogColumnDefaultDeclaration)
	for _, action := range record.NativeDeclarations {
		if d := action.Default; d != nil {
			active[[2]string{d.Identity, d.Column}] = *d
		}
		if d := action.Drop; d != nil {
			delete(active, [2]string{d.Identity, d.Storage})
		}
	}
	result := make([]catalogColumnDefaultDeclaration, 0, len(active))
	for _, d := range active {
		result = append(result, d)
	}
	return result
}

func columnRetirementMapBytes(retired map[string]bool) uint {
	var bytes uint
	for name := range retired {
		bytes += 64 + uint(len(name))
	}
	return bytes
}
