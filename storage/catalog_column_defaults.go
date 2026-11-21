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

type catalogStoredColumnAmendment struct {
	defaultDeclaration catalogColumnDefaultDeclaration
	declaration        *catalogColumnDeclaration
	propertyRemoval    *catalogColumnPropertyRemoval
	appendDeclaration  *catalogColumnAppend
	table              *table
	columns            []*column
	previous           *tableShowColumnsSnapshot
	columnBytes        uint
	retainedBytes      uint
}

func columnDefaultOperation(operation string) bool {
	switch operation {
	case "default", "default_expression", "add_default", "default_declaration", "drop_default":
		return true
	}
	return false
}

func (tx *TxContext) alterPublicStoredColumn(t *table, name, operation string, value scm.Scmer) scm.Scmer {
	if tx == nil || tx.transactionState == nil {
		panic("default declaration requires an active native transaction")
	}
	if tx.pendingNullColumn(t, name) {
		panic("a pending nullable append cannot be altered before its decision")
	}
	if operation == "drop_default" {
		requireTableMaintenance(t.schema.Name, t.Name, maintenanceAlter)
		t.schema.GetRead(tx).Release()
		if !t.mayHaveDefaultDeclaration(name, tx) {
			return scm.NewBool(false)
		}
	}
	profile := catalogDeclarationConstruction
	if operation == "declaration" {
		profile = catalogValidationConstruction
	}
	finish := tx.beginCatalogConstruction(profile, t)
	defer func() {
		failure := recover()
		finish()
		tx.retireUnacceptedPublicCatalogPhase()
		if failure != nil {
			panic(failure)
		}
	}()
	if tx.State != TxActive || tx.decisionStarted || tx.coordinator != nil || tx.catalogIntents.Load() != nil || activeReplication(t.schema) != nil {
		panic("default declaration requires a local public catalog transaction")
	}
	if (!columnDefaultOperation(operation) && operation != "declaration" && operation != "drop_property") || t.Ephemeral || t.isEphemeralQueryTable() {
		panic("default declaration requires an ordinary public table")
	}
	requireTableMaintenance(t.schema.Name, t.Name, maintenanceAlter)
	result := scm.NewBool(false)
	kind := catalogDefaultKind
	if operation == "declaration" {
		kind = catalogColumnKind
	}
	tx.stagePublicCatalogDeclaration(t.schema, kind, func(changes *publicCatalogChanges) bool {
		result = changes.stageStoredColumn(tx, t, name, operation, value)
		return scm.ToBool(result)
	})
	return result
}

// Both stored-column profiles retain the same endpoint/declaration lease.
// Caller owns changes.mu; an unsuccessful first amendment retires only its lease.
func (changes *publicCatalogChanges) acquireStoredColumnLease(tx *TxContext, t *table, validated bool) func(bool) {
	acquired := changes.leases[t] == nil
	if acquired {
		admission := tableCatalogDeclarationAdmission
		if validated {
			admission = tableValidatedDeclarationAdmission
		}
		lease, err := t.writeAdmission.admit(scanQueryContext(tx, context.Background()), tx.transactionState, admission, false)
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
	lease := changes.leases[t]
	return func(accepted bool) {
		if !accepted && acquired {
			// A failed waiting prefix cannot retire a replacement constructor's lease.
			if changes.leases[t] != lease {
				return
			}
			// A waiting append releases changes.mu. A successful sibling may then
			// publish an amendment borrowing this same lease. The actual accepted
			// view/blocked root owns it now; a failed prefix must not retire it.
			if views := changes.views.Load(); views != nil {
				_, blocked := views.blockedWriters[t]
				if views.tables[t] != nil || blocked {
					return
				}
			}
			if changes.anchor == changes.leases[t] {
				changes.anchor = nil
			}
			_ = changes.leases[t].Close()
			delete(changes.leases, t)
			delete(changes.baselines, t)
		}
	}
}

func (changes *publicCatalogChanges) stageStoredColumn(tx *TxContext, t *table, name, operation string, value scm.Scmer) scm.Scmer {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	accepted := false
	finishLease := changes.acquireStoredColumnLease(tx, t, operation == "declaration")
	defer func() { finishLease(accepted) }()
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
		if names := tx.columnDeclarationSnapshot(t); names != nil {
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
		var declaration *catalogColumnDeclaration
		var propertyRemoval *catalogColumnPropertyRemoval
		result := scm.NewBool(true)
		if operation == "declaration" {
			if !value.IsSlice() || len(value.Slice()) != 2 {
				panic("declaration ALTER requires new and expected declarations")
			}
			after, before := decodeStoredColumnType(value.Slice()[0]), decodeStoredColumnType(value.Slice()[1])
			actual := storedColumnTypeOf(previous)
			if actual.equal(after) {
				return scm.NewBool(true)
			}
			if !actual.equal(before) {
				panic("column declaration changed before ALTER publication")
			}
			after.requireCompatible(before)
			requireStoredColumnDependencies(t, tx, previous.Name)
			// The validated declaration and raw callback setup share the native
			// writer capability owner. The retained guard survives success until
			// this same amendment is rewound or its transaction finishes.
			restore := t.guardCatalogDeclarationInput(tx.transactionState)
			defer func() {
				if !accepted {
					restore()
				}
			}()
			if before.Nullable && !after.Nullable {
				validateColumnNotNull(tx, t, previous.Name)
			}
			replacement = after.replace(previous)
			declaration = &catalogColumnDeclaration{t.Name, t.RuntimeIdentity, previous.Name, before, after}
		} else if operation == "drop_property" {
			if !value.IsString() || value.String() == "" {
				panic("column property removal requires a nonempty key")
			}
			before, found := previous.DeclarationProperties[value.String()]
			if !found {
				return scm.NewBool(false)
			}
			propertyRemoval = &catalogColumnPropertyRemoval{t.Name, t.RuntimeIdentity, previous.Name, value.String(), before}
			result = replacement.Alter(operation, value)
		} else {
			result = scm.NewBool(scm.ToBool(replacement.Alter(operation, value)))
		}
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
		view, bytes := replaceStoredColumnMetadata(t, base, updated)
		amendment := &catalogStoredColumnAmendment{defaultDeclaration: catalogColumnDefaultDeclaration{t.Name, t.RuntimeIdentity, previous.Name, replacement.Default, replacement.DefaultExpression, replacement.DefaultName, replacement.DefaultDeferred}, declaration: declaration, table: t, columns: updated, previous: before}
		amendment.propertyRemoval = propertyRemoval
		amendment.columnBytes = storedColumnBytes(replacement)
		amendment.retainedBytes = uint(unsafe.Sizeof(*amendment)) + uint(cap(updated))*uint(unsafe.Sizeof((*column)(nil))) + amendment.columnBytes + bytes + declaration.memory() + propertyRemoval.memory()
		history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{column: amendment})
		access := db.GetWrite(tx)
		defer access.Release()
		db.schemalock.Lock()
		defer db.schemalock.Unlock()
		changes.validateConstraintStateEndpointsLocked(tx)
		if db.tables.Get(t.Name) != t || t.showColumnsSnapshot.Load().metadata.declarations != observed.metadata.declarations {
			panic("public declaration conflict: default input changed during preparation")
		}
		if replacement.DefaultName != "" && replacement.DefaultName != previous.DefaultName && (db.hasDeclarationNameLocked(replacement.DefaultName, observed.metadata.columnNames.declarations[index]) || tx.pendingUniqueDeclarationName(replacement.DefaultName)) {
			panic("declaration name already exists: " + replacement.DefaultName)
		}
		for _, entry := range tx.publicForeignAmendments() {
			if replacement.DefaultName == "" {
				break
			}
			if entry.foreign != nil && strings.EqualFold(entry.foreign.declaration.Foreign.Id, replacement.DefaultName) || entry.check != nil && strings.EqualFold(entry.check.declaration.Check.Id, replacement.DefaultName) || entry.column != nil && (entry.column.table != t || entry.column.defaultDeclaration.Column != previous.Name) && strings.EqualFold(entry.column.defaultDeclaration.Name, replacement.DefaultName) {
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
		if operation == "declaration" {
			tx.noteStatementMutation(t)
		}
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
// The same amendment publishes SHOW/stored descriptors and native value input.
func (tx *TxContext) columnDeclarationSnapshot(t *table) *tableColumnNamesSnapshot {
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
	t.requireCatalogDataVisibility(tx)
	if names := tx.columnDeclarationSnapshot(t); names != nil {
		return names
	}
	return t.columnNamesSnapshot.Load()
}

// Reuse this transaction's existing exclusive endpoint lease, not a fabricated
// writer capability. Unpublished constraint/trigger programs still prohibit DML.
func (tx *TxContext) publicColumnMutationAllowed(t *table) bool {
	if tx == nil || tx.State != TxActive {
		return false
	}
	if changes := tx.publicCatalogChanges(); changes != nil {
		if views := changes.views.Load(); views != nil {
			_, allowed := views.columnWriters[t]
			return allowed
		}
	}
	return false
}

// Stage/restore own history traversal. One immutable root publishes metadata
// and its exact native writer summary together; read-side operations do neither
// history scans, metadata sizing nor declaration-owner locking.
func (changes *publicCatalogChanges) publishViews(history []catalogDeclarationAmendment, tables map[*table]*tableShowColumnsSnapshot) {
	keys := make(map[*table][]uniqueKey)
	guarded := make(map[*table]bool)
	writers := make(map[*table]struct{})
	blocked := make(map[*table]struct{})
	nullColumns := make(map[*table]map[string]struct{})
	appendedColumns := make(map[*table]map[string]*column)
	dropped := make(map[*table]struct{})
	for _, entry := range history {
		if key := entry.publicKey; key != nil {
			keys[key.table] = key.keys
			guarded[key.table] = true
			blocked[key.table] = struct{}{}
		}
		if amendment := entry.foreign; amendment != nil {
			for _, t := range amendment.endpoints {
				if t != nil {
					blocked[t] = struct{}{}
				}
			}
		}
		if amendment := entry.check; amendment != nil {
			blocked[amendment.table] = struct{}{}
		}
		if amendment := entry.foreignDrop; amendment != nil {
			for _, t := range amendment.endpoints {
				if t != nil {
					blocked[t] = struct{}{}
				}
			}
		}
		if amendment := entry.checkDrop; amendment != nil {
			blocked[amendment.table] = struct{}{}
		}
		if amendment := entry.constraintState; amendment != nil {
			for _, view := range amendment.views {
				blocked[view.table] = struct{}{}
			}
		}
		if amendment := entry.statementTrigger; amendment != nil {
			blocked[amendment.slot.table] = struct{}{}
		}
		if amendment := entry.column; amendment != nil {
			if amendment.appendDeclaration != nil {
				blocked[amendment.table] = struct{}{}
				if appendedColumns[amendment.table] == nil {
					appendedColumns[amendment.table] = make(map[string]*column)
				}
				name := amendment.appendDeclaration.Name
				for _, c := range amendment.columns {
					if c.Name == name {
						appendedColumns[amendment.table][name] = c
						break
					}
				}
				if amendment.appendDeclaration.Initialization == nil {
					if nullColumns[amendment.table] == nil {
						nullColumns[amendment.table] = make(map[string]struct{})
					}
					nullColumns[amendment.table][name] = struct{}{}
				}
			} else {
				writers[amendment.table] = struct{}{}
			}
			if amendment.declaration != nil || amendment.appendDeclaration != nil {
				guarded[amendment.table] = true
			}
		}
		if amendment := entry.tableDrop; amendment != nil {
			dropped[amendment.table] = struct{}{}
			blocked[amendment.table] = struct{}{}
		}
	}
	// A stored-column delta borrows its own lease only when no unpublished
	// enforcement program or retirement changes that exact native endpoint.
	for t := range blocked {
		delete(writers, t)
	}
	checked := checkedCatalogWriters(history)
	changes.checkedWritersHighWater = max(changes.checkedWritersHighWater, len(checked))
	changes.columnWritersHighWater = max(changes.columnWritersHighWater, len(writers))
	changes.blockedWritersHighWater = max(changes.blockedWritersHighWater, len(blocked))
	changes.tableDropsHighWater = max(changes.tableDropsHighWater, len(dropped))
	for t := range changes.leases {
		t.setGuardedCatalogDeclarationInput(changes.owner, guarded[t])
	}
	changes.views.Store(&publicCatalogViews{keys: keys, tables: tables, columnWriters: writers, nullColumns: nullColumns, appendedColumns: appendedColumns, blockedWriters: blocked, dropped: dropped, checkedWriters: checked, history: history})
}

func replaceStoredColumnMetadata(t *table, base *tableShowColumnsSnapshot, columns []*column) (*tableShowColumnsSnapshot, uint) {
	if len(columns) != len(base.metadata.columnNames.declarations) {
		// Preserve earlier private key/default/type descriptions on the prefix;
		// build only the actual newly appended columns beyond that predecessor.
		prefix, prefixBytes := replaceStoredColumnMetadata(t, base, columns[:len(base.metadata.columnNames.declarations)])
		names := t.buildColumnNamesSnapshot(columns)
		result := t.buildShowColumnsSnapshot(base.rowEstimate, names)
		result.metadata.catalogVersion, result.metadata.catalogPrevious = base.metadata.catalogVersion, base.metadata.catalogPrevious
		result.metadata.statistics = base.metadata.statistics
		rows := append([]scm.Scmer(nil), result.metadata.columnRows.Slice()...)
		copy(rows, prefix.metadata.columnRows.Slice())
		result.metadata.columnRows = scm.NewSlice(rows)
		result.value = result.metadata.columnRows
		result.metadata.constraints, result.metadata.foreignByID = base.metadata.constraints, base.metadata.foreignByID
		result.metadata.declarations = buildTableDeclarations(result.metadata.columnRows, result.metadata.constraints)
		bytes := prefixBytes + uint(unsafe.Sizeof(*result)) + uint(unsafe.Sizeof(*result.metadata)) + uint(unsafe.Sizeof(*names)) + names.initialBytes
		bytes += scm.ComputeSize(result.value) + scm.ComputeSize(result.plannerValue) + scm.ComputeSize(result.metadata.declarations)
		bytes += uint(cap(names.declarations))*uint(unsafe.Sizeof((*column)(nil))) + uint(len(names.exact)+len(names.folded))*uint(unsafe.Sizeof(tableColumnName{}))
		return result, bytes
	}
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
		description := column.show("", 0, 0, nil)
		for _, row := range []*scm.Scmer{&rows[i], &declared[i]} {
			fields := append([]scm.Scmer(nil), row.Slice()...)
			for j := 0; j < len(fields); j += 2 {
				switch fields[j].String() {
				case "Type", "RawType", "Dimensions", "Null", "Collation", "Default", "DefaultExpression", "DefaultDeferred", "DefaultName", "Properties":
					fields[j+1] = foreignMetadataValue(description, fields[j].String())
				}
			}
			*row = scm.NewSlice(fields)
			bytes += uint(cap(fields)) * uint(unsafe.Sizeof(scm.Scmer{}))
		}
	}
	result.filterSchema = storedColumnFilterSchema(t.Collation, columns)
	metadata.columnRows = scm.NewSlice(rows)
	result.value = metadata.columnRows
	metadata.constraints, metadata.foreignByID = base.metadata.constraints, base.metadata.foreignByID
	declarations[1], declarations[3] = scm.NewSlice(declared), metadata.constraints
	metadata.declarations = scm.NewSlice(declarations)
	bytes += uint(cap(rows)+cap(declared)+cap(declarations)) * uint(unsafe.Sizeof(scm.Scmer{}))
	return &result, bytes
}

func storedColumnBytes(c *column) uint {
	bytes := uint(unsafe.Sizeof(*c)) + uint(len(c.DefaultExpression)+len(c.DefaultName))
	bytes += uint(cap(c.Typdimensions))*uint(unsafe.Sizeof(int(0))) + uint(cap(c.EnumValues))*uint(unsafe.Sizeof(""))
	bytes += 64 * uint(len(c.DeclarationProperties))
	if c.InitialValues != nil {
		bytes += 64 + 64*uint(len(c.InitialValues)) + scm.ComputeSize(c.InitialValue) - uint(unsafe.Sizeof(scm.Scmer{}))
		for id := range c.InitialValues {
			bytes += uint(len(id))
		}
	}
	return bytes + scm.ComputeSize(c.Default) - uint(unsafe.Sizeof(scm.Scmer{}))
}

func (tx *TxContext) publicColumnDefaultDeclarations() []catalogColumnDefaultDeclaration {
	latest := make(map[string]catalogColumnDefaultDeclaration)
	for _, entry := range tx.publicForeignAmendments() {
		if amendment := entry.column; amendment != nil && amendment.declaration == nil && amendment.propertyRemoval == nil && amendment.appendDeclaration == nil && !tx.ownsTableDrop(amendment.table) {
			latest[amendment.defaultDeclaration.Table+"\x00"+amendment.defaultDeclaration.Column] = amendment.defaultDeclaration
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

type storedColumnPublicationInput struct {
	columns      []*column
	payloadBytes uint
}

// Preparation derives the finite metadata endprojection once. Key-only
// endpoints perform one lookup instead of rescanning all amendment history.
func publicStoredColumnInputs(history []catalogDeclarationAmendment) map[*table]storedColumnPublicationInput {
	result := make(map[*table]storedColumnPublicationInput)
	payloads := make(map[*table]map[string]uint)
	dropped := make(map[*table]bool)
	for _, entry := range history {
		if drop := entry.tableDrop; drop != nil {
			dropped[drop.table] = true
			delete(result, drop.table)
			delete(payloads, drop.table)
		}
		if amendment := entry.column; amendment != nil && !dropped[amendment.table] {
			result[amendment.table] = storedColumnPublicationInput{columns: amendment.columns}
			if payloads[amendment.table] == nil {
				payloads[amendment.table] = make(map[string]uint)
			}
			name := amendment.defaultDeclaration.Column
			if amendment.appendDeclaration != nil {
				name = amendment.appendDeclaration.Name
			}
			payloads[amendment.table][name] = amendment.columnBytes
		}
	}
	for t, input := range result {
		input.payloadBytes = uint(cap(input.columns)) * uint(unsafe.Sizeof((*column)(nil)))
		for _, bytes := range payloads[t] {
			input.payloadBytes += bytes
		}
		result[t] = input
	}
	return result
}

func validateCatalogColumnDefault(db *database, declaration catalogColumnDefaultDeclaration, views ...map[*table]*tableShowColumnsSnapshot) *table {
	t := db.tables.Get(declaration.Table)
	if t == nil || t.RuntimeIdentity != declaration.Identity || !t.committedAt(db.catalogView().version) {
		panic("default declaration endpoint generation was replaced")
	}
	columns := t.Columns
	if len(views) > 0 {
		columns = catalogColumnsInView(t, views[0])
	}
	for _, column := range columns {
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
	latest := make(map[string]*catalogStoredColumnAmendment)
	for _, entry := range *history {
		if amendment := entry.column; amendment != nil && amendment.declaration == nil && amendment.propertyRemoval == nil && amendment.appendDeclaration == nil {
			if changes.db.tables.Get(amendment.defaultDeclaration.Table) != amendment.table || amendment.table.RuntimeIdentity != amendment.defaultDeclaration.Identity {
				return fmt.Errorf("default declaration endpoint generation was replaced")
			}
			latest[amendment.defaultDeclaration.Table+"\x00"+amendment.defaultDeclaration.Column] = amendment
		}
	}
	for _, amendment := range latest {
		declaration := amendment.defaultDeclaration
		for _, column := range amendment.table.columnDeclarations() {
			if column.Name == declaration.Column && declaration.Name != "" && (changes.db.hasDeclarationNameLocked(declaration.Name, column) || (&TxContext{transactionState: changes.owner}).pendingUniqueDeclarationName(declaration.Name)) {
				return fmt.Errorf("public declaration conflict: declaration name already exists: %s", declaration.Name)
			}
		}
	}
	return nil
}
