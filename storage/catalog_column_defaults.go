/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"
import "sort"
import "context"
import "strings"
import "unsafe"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"

// A default delta changes declaration data only. Row payloads, sanitizers,
// historical ADD extents and compiled frontend expressions are not serialized.
type catalogColumnDefaultDeclaration struct {
	Table      string    `json:"table"`
	Identity   string    `json:"identity"`
	Column     string    `json:"column"`
	Value      scm.Scmer `json:"value"`
	Expression string    `json:"expression"`
	Name       string    `json:"name"`
	Deferred   bool      `json:"deferred"`
}

type catalogColumnDefaultAmendment struct {
	declaration   catalogColumnDefaultDeclaration
	table         *table
	columns       []*column
	previous      *tableShowColumnsSnapshot
	columnBytes   uint
	retainedBytes uint
}

func columnDefaultOperation(operation string) bool {
	switch operation {
	case "default", "default_expression", "add_default", "default_declaration", "drop_default":
		return true
	}
	return false
}

func (tx *TxContext) alterPublicColumnDefault(t *table, name, operation string, value scm.Scmer) scm.Scmer {
	if tx == nil || tx.transactionState == nil {
		panic("default declaration requires an active native transaction")
	}
	finish := tx.beginCatalogConstruction(false)
	defer finish()
	if tx.State != TxActive || tx.decisionStarted || tx.coordinator != nil || tx.catalogIntents.Load() != nil || activeReplication(t.schema) != nil {
		panic("default declaration requires a local public catalog transaction")
	}
	if !columnDefaultOperation(operation) || t.Ephemeral || t.isEphemeralQueryTable() {
		panic("default declaration requires an ordinary public table")
	}
	requireTableMaintenance(t.schema.Name, t.Name, maintenanceAlter)
	result := scm.NewBool(false)
	tx.stagePublicCatalogDeclaration(t.schema, func(changes *publicCatalogChanges) bool {
		result = changes.stageColumnDefault(tx, t, name, operation, value)
		return scm.ToBool(result)
	})
	return result
}

func (changes *publicCatalogChanges) stageColumnDefault(tx *TxContext, t *table, name, operation string, value scm.Scmer) scm.Scmer {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	accepted := false
	acquired := changes.leases[t] == nil
	if acquired {
		lease, err := t.writeAdmission.admit(scanQueryContext(tx, context.Background()), tx.transactionState, true, false)
		if err != nil {
			panic(err)
		}
		changes.leases[t] = lease
		changes.leasesHighWater = max(changes.leasesHighWater, len(changes.leases))
		if changes.anchor == nil {
			changes.refreshMemoryLocked()
			lease.attachRetainedMemory(changes.memory)
			changes.anchor = lease
		}
	}
	defer func() {
		if !accepted && acquired {
			if changes.anchor == changes.leases[t] {
				changes.anchor = nil
			}
			_ = changes.leases[t].Close()
			delete(changes.leases, t)
			delete(changes.baselines, t)
		}
	}()
	db := changes.db
	var columns []*column
	var observed *tableShowColumnsSnapshot
	func() {
		access := db.GetWrite(tx)
		defer access.Release()
		db.schemalock.RLock()
		defer db.schemalock.RUnlock()
		changes.validateConstraintStateEndpointsLocked(tx)
		if t.schema != db || db.tables.Get(t.Name) != t || !t.committedAt(db.catalogView().version) {
			panic("default declaration endpoint generation was replaced")
		}
		observed = t.showColumnsSnapshot.Load()
		columns = observed.metadata.columnNames.declarations
		if names := tx.columnDefaultSnapshot(t); names != nil {
			columns = names.declarations
		}
	}()
	for index, previous := range columns {
		if operation == "drop_default" {
			if previous.DefaultName == "" || !strings.EqualFold(previous.DefaultName, name) {
				continue
			}
		} else if previous.Name != name {
			continue
		}
		t.requireDeclaredColumnAlter(previous, operation, value)
		if previous.IsTemp || isRuntimeComputedColumn(previous) || atomic.LoadInt64(&previous.cacheUsers) != 0 {
			panic("default declaration requires an ordinary stored column")
		}
		replacement := cloneStoredColumnDeclaration(previous)
		result := scm.NewBool(scm.ToBool(replacement.Alter(operation, value)))
		if operation == "add_default" || operation == "default_declaration" {
			result = scm.NewBool(true)
		}
		updated := append([]*column(nil), columns...)
		updated[index] = replacement
		views := maps.Clone(changes.views.Load().tables)
		before := views[t]
		base := before
		if base == nil {
			base = observed
		}
		view, bytes := replaceColumnDefaultMetadata(t, base, updated)
		amendment := &catalogColumnDefaultAmendment{declaration: catalogColumnDefaultDeclaration{t.Name, t.RuntimeIdentity, previous.Name, replacement.Default, replacement.DefaultExpression, replacement.DefaultName, replacement.DefaultDeferred}, table: t, columns: updated, previous: before}
		amendment.columnBytes = storedDefaultColumnBytes(replacement)
		amendment.retainedBytes = uint(unsafe.Sizeof(*amendment)) + uint(cap(updated))*uint(unsafe.Sizeof((*column)(nil))) + amendment.columnBytes + bytes
		history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{columnDefault: amendment})
		access := db.GetWrite(tx)
		defer access.Release()
		db.schemalock.Lock()
		defer db.schemalock.Unlock()
		changes.validateConstraintStateEndpointsLocked(tx)
		if db.tables.Get(t.Name) != t || t.showColumnsSnapshot.Load().metadata.declarations != observed.metadata.declarations {
			panic("public declaration conflict: default input changed during preparation")
		}
		if replacement.DefaultName != "" && replacement.DefaultName != previous.DefaultName && db.hasDeclarationNameLocked(replacement.DefaultName, observed.metadata.columnNames.declarations[index]) {
			panic("declaration name already exists: " + replacement.DefaultName)
		}
		for _, entry := range tx.publicForeignAmendments() {
			if replacement.DefaultName == "" {
				break
			}
			if entry.foreign != nil && strings.EqualFold(entry.foreign.declaration.Foreign.Id, replacement.DefaultName) || entry.check != nil && strings.EqualFold(entry.check.declaration.Check.Id, replacement.DefaultName) || entry.columnDefault != nil && (entry.columnDefault.table != t || entry.columnDefault.declaration.Column != previous.Name) && strings.EqualFold(entry.columnDefault.declaration.Name, replacement.DefaultName) {
				panic("declaration name already exists: " + replacement.DefaultName)
			}
		}
		if _, exists := changes.baselines[t]; !exists {
			changes.baselines[t] = foreignBaseline(observed)
		}
		changes.baselinesHighWater = max(changes.baselinesHighWater, len(changes.baselines))
		views[t] = view
		changes.viewsHighWater = max(changes.viewsHighWater, len(views))
		changes.amendmentBytes += amendment.retainedBytes
		changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
		tx.catalogAmendments.Store(&history)
		changes.publishViews(history, views)
		accepted = true
		return result
	}
	if operation != "drop_default" {
		panic("column " + db.Name + "." + t.Name + "." + name + " does not exist")
	}
	return scm.NewBool(false)
}

// A caller borrows this immutable directory once per native INSERT/batch.
// The same amendment publishes SHOW/default descriptors and native value input.
func (tx *TxContext) columnDefaultSnapshot(t *table) *tableColumnNamesSnapshot {
	if tx != nil {
		if changes := tx.publicCatalogChanges(); changes != nil {
			if views := changes.views.Load(); views != nil {
				if view := views.tables[t]; view != nil {
					return view.metadata.columnNames
				}
			}
		}
	}
	return nil
}

func (t *table) columnNamesForTx(tx *TxContext) *tableColumnNamesSnapshot {
	if names := tx.columnDefaultSnapshot(t); names != nil {
		return names
	}
	return t.columnNamesSnapshot.Load()
}

// Reuse this transaction's existing exclusive endpoint lease, not a fabricated
// writer capability. Unpublished constraint/trigger programs still prohibit DML.
func (tx *TxContext) publicDefaultMutationAllowed(t *table) bool {
	if tx == nil || tx.State != TxActive {
		return false
	}
	if changes := tx.publicCatalogChanges(); changes != nil {
		if views := changes.views.Load(); views != nil {
			_, allowed := views.defaultWriters[t]
			return allowed
		}
	}
	return false
}

// Stage/restore own history traversal. One immutable root publishes metadata
// and its exact native writer summary together; read-side operations do neither
// history scans, metadata sizing nor declaration-owner locking.
func (changes *publicCatalogChanges) publishViews(history []catalogDeclarationAmendment, tables map[*table]*tableShowColumnsSnapshot) {
	var writers map[*table]struct{}
	for _, entry := range history {
		if entry.foreign != nil || entry.check != nil || entry.constraintState != nil || entry.statementTrigger != nil {
			writers = nil
			break
		}
		if amendment := entry.columnDefault; amendment != nil {
			if writers == nil {
				writers = make(map[*table]struct{})
			}
			writers[amendment.table] = struct{}{}
		}
	}
	changes.defaultWritersHighWater = max(changes.defaultWritersHighWater, len(writers))
	changes.views.Store(&publicCatalogViews{tables: tables, defaultWriters: writers})
}

func replaceColumnDefaultMetadata(t *table, base *tableShowColumnsSnapshot, columns []*column) (*tableShowColumnsSnapshot, uint) {
	render := base
	if render.metadata.columnNames == nil {
		render = t.showColumnsSnapshot.Load()
	}
	result, metadata, names := *render, *render.metadata, *render.metadata.columnNames
	result.metadata = &metadata
	names.declarations = append([]*column(nil), columns...)
	metadata.columnNames = &names
	rows := append([]scm.Scmer(nil), render.metadata.columnRows.Slice()...)
	declarations := append([]scm.Scmer(nil), render.metadata.declarations.Slice()...)
	declared := append([]scm.Scmer(nil), declarations[1].Slice()...)
	bytes := uint(unsafe.Sizeof(result)) + uint(unsafe.Sizeof(metadata)) + uint(unsafe.Sizeof(names)) + uint(cap(names.declarations))*uint(unsafe.Sizeof((*column)(nil)))
	for i, column := range columns {
		if column == render.metadata.columnNames.declarations[i] {
			continue
		}
		for _, row := range []*scm.Scmer{&rows[i], &declared[i]} {
			fields := append([]scm.Scmer(nil), row.Slice()...)
			for j := 0; j < len(fields); j += 2 {
				switch fields[j].String() {
				case "Default":
					fields[j+1] = column.Default
				case "DefaultExpression":
					fields[j+1] = scm.NewString(column.DefaultExpression)
				case "DefaultDeferred":
					fields[j+1] = scm.NewBool(column.DefaultDeferred)
				case "DefaultName":
					fields[j+1] = scm.NewString(column.DefaultName)
				}
			}
			*row = scm.NewSlice(fields)
			bytes += uint(cap(fields)) * uint(unsafe.Sizeof(scm.Scmer{}))
		}
	}
	metadata.columnRows = scm.NewSlice(rows)
	result.value = metadata.columnRows
	metadata.constraints, metadata.foreignByID = base.metadata.constraints, base.metadata.foreignByID
	declarations[1], declarations[3] = scm.NewSlice(declared), metadata.constraints
	metadata.declarations = scm.NewSlice(declarations)
	bytes += uint(cap(rows)+cap(declared)+cap(declarations)) * uint(unsafe.Sizeof(scm.Scmer{}))
	return &result, bytes
}

func storedDefaultColumnBytes(c *column) uint {
	bytes := uint(unsafe.Sizeof(*c)) + uint(len(c.DefaultExpression)+len(c.DefaultName))
	bytes += uint(cap(c.Typdimensions))*uint(unsafe.Sizeof(int(0))) + uint(cap(c.EnumValues))*uint(unsafe.Sizeof(""))
	bytes += 64 * uint(len(c.DeclarationProperties))
	return bytes + scm.ComputeSize(c.Default) - uint(unsafe.Sizeof(scm.Scmer{}))
}

func (tx *TxContext) publicColumnDefaultDeclarations() []catalogColumnDefaultDeclaration {
	latest := make(map[string]catalogColumnDefaultDeclaration)
	for _, entry := range tx.publicForeignAmendments() {
		if amendment := entry.columnDefault; amendment != nil {
			latest[amendment.declaration.Table+"\x00"+amendment.declaration.Column] = amendment.declaration
		}
	}
	result := make([]catalogColumnDefaultDeclaration, 0, len(latest))
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

func publicColumnDefaultColumns(history []catalogDeclarationAmendment, t *table) ([]*column, uint) {
	var columns []*column
	payloads := make(map[string]uint)
	for _, entry := range history {
		if amendment := entry.columnDefault; amendment != nil && amendment.table == t {
			columns = amendment.columns
			payloads[amendment.declaration.Column] = amendment.columnBytes
		}
	}
	bytes := uint(cap(columns)) * uint(unsafe.Sizeof((*column)(nil)))
	for _, payload := range payloads {
		bytes += payload
	}
	return columns, bytes
}

func validateCatalogColumnDefault(db *database, declaration catalogColumnDefaultDeclaration) *table {
	t := db.tables.Get(declaration.Table)
	if t == nil || t.RuntimeIdentity != declaration.Identity || !t.committedAt(db.catalogView().version) {
		panic("default declaration endpoint generation was replaced")
	}
	for _, column := range t.Columns {
		if column.Name == declaration.Column {
			if column.IsTemp || column.ComputorDefinition != nil || isRuntimeComputedColumn(column) {
				panic("default declaration requires an ordinary stored column")
			}
			return t
		}
	}
	panic("default declaration column was removed")
}

func (db *database) recoverColumnDefaultDeclarations(declarations []catalogColumnDefaultDeclaration) {
	for _, declaration := range declarations {
		t := validateCatalogColumnDefault(db, declaration)
		for i, previous := range t.Columns {
			if previous.Name != declaration.Column {
				continue
			}
			if declaration.Name != "" && db.hasDeclarationNameLocked(declaration.Name, previous) {
				panic("default recovery declaration name already exists: " + declaration.Name)
			}
			replacement := cloneStoredColumnDeclaration(previous)
			replacement.Default, replacement.DefaultExpression, replacement.DefaultName, replacement.DefaultDeferred = declaration.Value, declaration.Expression, declaration.Name, declaration.Deferred
			columns := append([]*column(nil), t.Columns...)
			columns[i] = replacement
			t.Columns = columns
			break
		}
	}
}

func (changes *publicCatalogChanges) validateColumnDefaultsLocked() error {
	history := changes.owner.catalogAmendments.Load()
	if history == nil {
		return nil
	}
	latest := make(map[string]*catalogColumnDefaultAmendment)
	for _, entry := range *history {
		if amendment := entry.columnDefault; amendment != nil {
			if changes.db.tables.Get(amendment.declaration.Table) != amendment.table || amendment.table.RuntimeIdentity != amendment.declaration.Identity {
				return fmt.Errorf("default declaration endpoint generation was replaced")
			}
			latest[amendment.declaration.Table+"\x00"+amendment.declaration.Column] = amendment
		}
	}
	for _, amendment := range latest {
		declaration := amendment.declaration
		for _, column := range amendment.table.columnDeclarations() {
			if column.Name == declaration.Column && declaration.Name != "" && changes.db.hasDeclarationNameLocked(declaration.Name, column) {
				return fmt.Errorf("public declaration conflict: declaration name already exists: %s", declaration.Name)
			}
		}
	}
	return nil
}
