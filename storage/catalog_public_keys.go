/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"
import "strings"
import "slices"
import "unsafe"
import "context"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

type catalogKeyDeclaration struct {
	Table    string    `json:"table"`
	Identity string    `json:"identity"`
	Key      uniqueKey `json:"key"`
}

type catalogPublicKeyAmendment struct {
	operation     string
	index         *catalogIndexDeclaration
	indexes       []logicalIndexDeclaration
	declaration   catalogKeyDeclaration
	table         *table
	keys          []uniqueKey
	previous      *tableShowColumnsSnapshot
	retainedBytes uint
}

func (tx *TxContext) publicUniqueKeys() map[*table][]uniqueKey {
	if changes := tx.publicCatalogChanges(); changes != nil {
		if views := changes.views.Load(); views != nil {
			return views.keys
		}
	}
	return nil
}

func uniqueKeysInView(t *table, keys map[*table][]uniqueKey) []uniqueKey {
	if effective, ok := keys[t]; ok {
		return effective
	}
	return t.Unique
}

func sameUniqueKey(a, b uniqueKey) bool {
	return a.Id == b.Id && a.ConstraintName == b.ConstraintName && a.NullsNotDistinct == b.NullsNotDistinct && slices.Equal(a.Cols, b.Cols)
}

func validateCatalogKey(db *database, declaration catalogKeyDeclaration, views map[*table]*tableShowColumnsSnapshot, scope catalogForeignEndpointScope, projected map[*table][]*column) *table {
	t := db.tables.Get(declaration.Table)
	if t == nil || t.RuntimeIdentity != declaration.Identity || (!t.committedAt(db.catalogView().version) && !scope.owns(t)) {
		panic("unique declaration endpoint generation was replaced")
	}
	if _, err := uuid.Parse(declaration.Identity); err != nil {
		panic("invalid native unique endpoint identity")
	}
	key := declaration.Key
	if key.Id == "" || len(key.Cols) == 0 {
		panic("unique declaration requires a name and columns")
	}
	seen := make(map[string]bool, len(key.Cols))
	for _, name := range key.Cols {
		if name == "" || seen[name] {
			panic("invalid unique declaration column")
		}
		seen[name] = true
		column := catalogColumnInView(t, name, views, projected)
		if column == nil {
			panic("unknown unique declaration column: " + name)
		}
		if column.ComputorDefinition != nil {
			panic("unique keys on declared computed columns are not supported")
		}
	}
	return t
}

func (tx *TxContext) stagePublicKey(t *table, operation string, key uniqueKey, index *logicalIndexDeclaration) bool {
	if tx == nil || tx.transactionState == nil || tx.State != TxActive || tx.decisionStarted {
		panic("unique declaration requires an active native transaction")
	}
	// A committed or already-owned positive name is an ordinary create miss;
	// avoid allocating an otherwise empty catalog authority for that no-op.
	if index == nil && operation == "add" && (t.mayHaveUniqueDeclaration(key.Id, tx) || key.ConstraintName != "" && t.mayHaveUniqueDeclaration(key.ConstraintName, tx)) {
		return false
	}
	finish := tx.beginCatalogConstruction(catalogValidationConstruction, nil, t)
	defer func() { finish(); tx.retireUnacceptedPublicCatalogPhase() }()
	if tx.replicationActive || tx.replicationWriting || tx.relationData != nil || tx.coordinator != nil || tx.clusterNumber.Load() != 0 || tx.remoteObject != nil || tx.remoteInvocation || index == nil && (tx.catalogIntents.Load() != nil || tx.hasTableDrops()) && !(catalogForeignEndpointScope{owner: tx.transactionState}).owns(t) || t.Ephemeral || t.ScopedSchema || t.PersistencyMode == Cache {
		panic("public unique declaration requires a local committed native table")
	}
	if index != nil {
		tx.requireLogicalIndexEndpoint(t)
	}
	return tx.stagePublicCatalogDeclaration(t.schema, catalogPublicKeyKind, nil, func(changes *publicCatalogChanges) bool { return changes.stageKey(tx, t, operation, key, index) })
}

// Only typed index actions select required V32. Their endpoint may be public
// or belong to the SAME original constructor program; raw keys keep the old
// transaction-wide veto. The existing common entry still attests prepared,
// remote and native participant state under its actual declaration admission.
func (tx *TxContext) requireLogicalIndexEndpoint(t *table) {
	if t == nil || t.schema == nil || !localCatalogRowDatabase(t.schema) ||
		t.schema.tables.Get(t.Name) != t || t.RuntimeIdentity == "" || tx.ownsTableDrop(t) {
		panic("logical index requires an actual local endpoint outside pending table retirement")
	}
	tx.requireDeclarationEndpoint(t, maintenanceAlter)
	if !t.committedAt(t.schema.catalogView().version) && !(catalogForeignEndpointScope{owner: tx.transactionState}).owns(t) {
		panic("logical index requires its original native endpoint owner")
	}
	if tx.catalogIntents.Load() != nil || tx.hasTableDrops() {
		identity := tx.catalogIdentity.Load()
		if tx.Mode != TxACID || tx.State != TxActive || tx.decisionStarted || tx.prepared != nil || tx.catalogCommit != nil || tx.rollbackOnly.Load() ||
			identity == nil || identity.Namespace != "" || identity.database != t.schema || identity.Database != t.schema.Name || identity.Incarnation != t.schema.incarnationID() {
			panic("joint logical index requires the same unprepared local ACID catalog decision")
		}
	}
}

func (changes *publicCatalogChanges) stageKey(tx *TxContext, t *table, operation string, key uniqueKey, index *logicalIndexDeclaration) bool {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	accepted := false
	acquired := changes.leases[t] == nil
	if acquired {
		lease, err := t.writeAdmission.admit(scanQueryContext(tx, context.Background()), tx.transactionState, tableValidatedDeclarationAdmission, false)
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
	restoreInput := t.guardCatalogDeclarationInput(tx.transactionState)
	defer func() {
		if !accepted {
			restoreInput()
		}
	}()
	db := changes.db
	var declaration catalogKeyDeclaration
	var observed *tableShowColumnsSnapshot
	var keys []uniqueKey
	var indexes []logicalIndexDeclaration
	var logical *catalogIndexDeclaration
	func() {
		access := db.GetWrite(tx)
		defer access.Release()
		db.schemalock.RLock()
		defer db.schemalock.RUnlock()
		changes.validateConstraintStateEndpointsLocked(tx)
		if index != nil {
			tx.requireLogicalIndexEndpoint(t)
		} else {
			tx.requireDeclarationEndpoint(t, maintenanceAlter)
		}
		if t.schema != db || db.tables.Get(t.Name) != t || (!t.committedAt(db.catalogView().version) && !(catalogForeignEndpointScope{owner: tx.transactionState}).owns(t)) {
			panic("unique declaration endpoint generation was replaced")
		}
		current := uniqueKeysInView(t, tx.publicUniqueKeys())
		if index != nil {
			p := tx.indexProjection(t)
			if operation == "add" && logicalIndexPosition(indexesInView(t, p.indexes), index.Name) >= 0 {
				return
			}
			d := catalogIndexDeclaration{t.Name, t.RuntimeIdentity, *index}
			db.projectIndexDeclaration(&d, operation, p, tx.publicDeclarationViews(), catalogForeignEndpointScope{owner: tx.transactionState})
			keys, indexes, logical = p.keys[t], p.indexes[t], &d
			if index.Key != nil {
				key = *index.Key
			} else {
				key = uniqueKey{Id: index.Name, Cols: index.Columns}
			}
			declaration = catalogKeyDeclaration{t.Name, t.RuntimeIdentity, key}
		} else {
			if operation == "drop" && indexOwnsKey(t.logicalIndexesForTx(tx), key.Id) {
				panic("standalone index requires DROP INDEX")
			}
			if operation == "add" && logicalIndexPosition(t.logicalIndexesForTx(tx), key.Id) >= 0 {
				panic("key name belongs to a standalone index")
			}
			switch operation {
			case "add":
				for _, existing := range current {
					if existing.matchesName(key.Id) || key.ConstraintName != "" && existing.matchesName(key.ConstraintName) {
						return
					}
				}
				key.Cols = append([]string(nil), key.Cols...)
				declaration = catalogKeyDeclaration{t.Name, t.RuntimeIdentity, key}
				validateCatalogKey(db, declaration, tx.publicDeclarationViews(), catalogForeignEndpointScope{owner: tx.transactionState}, nil)
				if keyNameConflict(db, t, key, tx.publicUniqueKeys(), tx.publicForeignAmendments()) {
					panic("declaration name already exists: " + key.Id)
				}
				keys = append(append([]uniqueKey(nil), current...), key)
			case "drop":
				found := -1
				for i, existing := range current {
					if existing.matchesName(key.Id) {
						found = i
						key = existing
						break
					}
				}
				if found < 0 {
					return
				}
				declaration = catalogKeyDeclaration{t.Name, t.RuntimeIdentity, key}
				validateCatalogKey(db, declaration, tx.publicDeclarationViews(), catalogForeignEndpointScope{owner: tx.transactionState}, nil)
				keys = append([]uniqueKey(nil), current...)
				keys = slices.Delete(keys, found, found+1)
				if dependency := foreignKeyRequiresUnique(t, keys, tx.publicForeignAmendments()); dependency != "" {
					panic("unique declaration is required by foreign key " + dependency + ": " + key.Id + "; drop the foreign key constraint first")
				}
			default:
				panic("invalid public unique declaration operation")
			}
		}
		observed = t.showColumnsSnapshot.Load()
	}()
	if observed == nil {
		return false
	}
	// All data callbacks and index preparation run outside schema/table/shard
	// locks. The native declaration lease retains the complete validation cut.
	if operation == "add" && (index == nil || index.Key != nil) {
		t.enterPendingCatalogData(tx, changes.leases[t])
		validateExistingUniqueRows(tx, t, key)
	}
	views := maps.Clone(changes.views.Load().tables)
	previous := views[t]
	base := previous
	if base == nil {
		base = observed
	}
	view, bytes := replaceUniqueMetadata(base, keys, false)
	if logical != nil {
		var owned uint
		view, owned = replaceLogicalIndexMetadata(view, indexes)
		bytes += owned
	}
	amendment := &catalogPublicKeyAmendment{index: logical, indexes: indexes, operation: operation, declaration: declaration, table: t, keys: keys, previous: previous}
	amendment.retainedBytes = uint(cap(indexes))*uint(unsafe.Sizeof(logicalIndexDeclaration{})) + uint(unsafe.Sizeof(*amendment)) + uint(cap(keys))*uint(unsafe.Sizeof(uniqueKey{})) + bytes
	if logical != nil && operation == "add" {
		amendment.retainedBytes += logical.memory()
	}
	if operation == "add" {
		amendment.retainedBytes += uniqueKeyPayloadBytes(key)
	}
	access := db.GetWrite(tx)
	defer access.Release()
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	changes.validateConstraintStateEndpointsLocked(tx)
	if t.showColumnsSnapshot.Load().metadata.declarations != observed.metadata.declarations {
		panic("public unique declaration changed during validation")
	}
	if _, exists := changes.baselines[t]; !exists {
		changes.baselines[t] = foreignBaseline(observed)
	}
	changes.baselinesHighWater = max(changes.baselinesHighWater, len(changes.baselines))
	views[t] = view
	changes.viewsHighWater = max(changes.viewsHighWater, len(views))
	history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{publicKey: amendment})
	changes.amendmentBytes += amendment.retainedBytes
	changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	tx.noteStatementMutation(t)
	tx.catalogAmendments.Store(&history)
	changes.publishViews(history, views)
	accepted = true
	return true
}

// Native exact comparisons provide the same collation/NULL contracts as
// ordinary key lookup. Only bounded tuples and scalar counts are retained.
func validateExistingUniqueRows(tx *TxContext, t *table, key uniqueKey) {
	schema := newExactScanAccessSchema(key.Cols)
	if key.NullsNotDistinct {
		for i, name := range key.Cols {
			schema[scanAccessSchemaHeaderSize+i] = newScanBoundarySpec(name, EqualMatcher, i, i, true, true, "", true, -1, nil, nil, "", false)
		}
	}
	access := scm.NewSlice(schema)
	truth := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	one := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return scm.NewInt(values[0].Int() + 1) })
	count := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return scm.NewInt(values[0].Int() + values[1].Int()) })
	validate := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		if !key.NullsNotDistinct {
			for _, value := range values[1:] {
				if value.IsNil() {
					return scm.NewBool(true)
				}
			}
		}
		matches := t.scan(tx, access, values[1:], nil, truth, nil, one, scm.NewInt(0), count, false)
		if matches.Int() > 1 {
			panic(declarationValidationViolation("unique-declaration", key.Id, fmt.Sprintf("Duplicate entry in table %s prevents unique key %s", t.Name, key.Id)))
		}
		return scm.NewBool(true)
	})
	t.scan(tx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, truth, key.Cols, validate, scm.NewBool(true), truth, false)
}

func foreignKeyRequiresUnique(t *table, keys []uniqueKey, history []catalogDeclarationAmendment) string {
	pending := currentForeignAdditions(history)
	var foreign []foreignKey
	for _, fk := range t.Foreign {
		if _, changed := pending[t.schema.storedForeignIdentity(fk)]; !changed && fk.Tbl2 == t.Name {
			foreign = append(foreign, fk)
		}
	}
	for _, addition := range pending {
		if addition != nil && addition.declaration.Foreign.Tbl2 == t.Name {
			foreign = append(foreign, addition.declaration.Foreign)
		}
	}
	for _, fk := range foreign {
		found := false
		for _, candidate := range keys {
			if slices.Equal(candidate.Cols, fk.Cols2) {
				found = true
				break
			}
		}
		if !found {
			return fk.Id
		}
	}
	return ""
}

func keyNameConflict(db *database, t *table, key uniqueKey, keys map[*table][]uniqueKey, history []catalogDeclarationAmendment) bool {
	foreign := nativeForeignDirectory(effectiveForeignDeclarations(db, history))
	if nativeKeyNameConflict(db, t, key, keys, foreign) {
		return true
	}
	latestDefaults := make(map[string]*catalogStoredColumnAmendment)
	for _, entry := range history {
		if c := entry.check; c != nil && ((c.table == t && strings.EqualFold(c.declaration.Check.Id, key.Id)) || key.ConstraintName != "" && strings.EqualFold(c.declaration.Check.Id, key.ConstraintName)) {
			return true
		}
		if d := entry.column; d != nil && d.declaration == nil && d.propertyRemoval == nil && d.appendDeclaration == nil {
			latestDefaults[d.defaultDeclaration.Table+"/"+d.defaultDeclaration.Column] = d
		}
	}
	for _, d := range latestDefaults {
		if d.defaultDeclaration.Name != "" && ((d.table == t && strings.EqualFold(d.defaultDeclaration.Name, key.Id)) || key.ConstraintName != "" && strings.EqualFold(d.defaultDeclaration.Name, key.ConstraintName)) {
			return true
		}
	}
	return false
}

func replaceUniqueMetadata(base *tableShowColumnsSnapshot, keys []uniqueKey, published bool) (*tableShowColumnsSnapshot, uint) {
	var result tableShowColumnsSnapshot
	var metadata tableShowColumnsSnapshotMetadata
	if published {
		result, metadata = *base, *base.metadata
	} else {
		metadata.columnNames, metadata.columnRows = base.metadata.columnNames, base.metadata.columnRows
		metadata.logicalIndexes, metadata.indexRows = base.metadata.logicalIndexes, base.metadata.indexRows
	}
	result.metadata = &metadata
	constraints := append([]scm.Scmer(nil), base.nativeConstraints().Slice()...)
	rows := uniqueConstraintMetadata(keys)
	for i, key := range keys {
		if indexOwnsKey(base.metadata.logicalIndexes, key.Id) {
			rows[i] = scm.NewSlice(append(rows[i].Slice(), scm.NewString("standalone_index"), scm.NewBool(true)))
		}
	}
	constraints[1] = scm.NewSlice(rows)
	metadata.constraints = scm.NewSlice(constraints)
	metadata.storageConstraints = metadata.constraints
	metadata.foreignByID = base.metadata.foreignByID
	declarations := append([]scm.Scmer(nil), base.metadata.declarations.Slice()...)
	keyTypes := make(map[string]string)
	for _, key := range keys {
		for _, column := range key.Cols {
			if key.Id == "PRIMARY" {
				keyTypes[column] = "PRI"
			} else if keyTypes[column] == "" {
				keyTypes[column] = "UNI"
			}
		}
	}
	var columnBytes uint
	rewriteColumns := func(old scm.Scmer) scm.Scmer {
		rows := append([]scm.Scmer(nil), old.Slice()...)
		columnBytes += uint(cap(rows)) * uint(unsafe.Sizeof(scm.Scmer{}))
		for i, row := range rows {
			kind := keyTypes[foreignMetadataValue(row, "Field").String()]
			if foreignMetadataValue(row, "Key").String() == kind {
				continue
			}
			fields := append([]scm.Scmer(nil), row.Slice()...)
			for j := 0; j < len(fields); j += 2 {
				if fields[j].String() == "Key" {
					fields[j+1] = scm.NewString(kind)
				}
			}
			rows[i] = scm.NewSlice(fields)
			columnBytes += uint(cap(fields)) * uint(unsafe.Sizeof(scm.Scmer{}))
		}
		return scm.NewSlice(rows)
	}
	metadata.columnRows = rewriteColumns(base.metadata.columnRows)
	result.value = metadata.columnRows
	declarations[1] = rewriteColumns(base.metadata.declarations.Slice()[1])
	declarations[3] = metadata.constraints
	metadata.declarations = scm.NewSlice(declarations)
	bytes := columnBytes + uint(unsafe.Sizeof(result)) + uint(unsafe.Sizeof(metadata)) + uint(cap(constraints)+cap(declarations)+cap(rows))*uint(unsafe.Sizeof(scm.Scmer{}))
	for _, row := range rows {
		bytes += uint(cap(row.Slice())+cap(foreignMetadataValue(row, "Cols").Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
	}
	return &result, bytes
}

// Metadata-only accounting at the original declaration/publication owners.
// Payload transfer follows actual final additions; canceled additions remain
// owned solely by immutable history until its normal savepoint/Close.
func lastPublicKeyAmendments(history []catalogDeclarationAmendment) map[*table]map[string]*catalogPublicKeyAmendment {
	last := make(map[*table]map[string]*catalogPublicKeyAmendment)
	for _, entry := range history {
		if a := entry.publicKey; a != nil {
			if last[a.table] == nil {
				last[a.table] = make(map[string]*catalogPublicKeyAmendment)
			}
			last[a.table][scm.CaseFoldKey(a.declaration.Key.Id)] = a
		}
	}
	return last
}
func publicUniqueTransfer(last map[string]*catalogPublicKeyAmendment, keys []uniqueKey) uint {
	var bytes uint
	for _, key := range keys {
		if a := last[scm.CaseFoldKey(key.Id)]; a != nil && a.operation == "add" && sameUniqueKey(a.declaration.Key, key) {
			bytes += uniqueKeyPayloadBytes(key)
		}
	}
	return bytes
}
func retiredUniquePayloads(before, after []uniqueKey) uint {
	current := make(map[string]uniqueKey, len(after))
	for _, key := range after {
		current[scm.CaseFoldKey(key.Id)] = key
	}
	var bytes uint
	for _, key := range before {
		next, exists := current[scm.CaseFoldKey(key.Id)]
		if !exists || !sameUniqueKey(key, next) {
			bytes += uniqueKeyPayloadBytes(key)
		}
	}
	return bytes
}

// Inspect only the native catalog's known container layout, not its payloads,
// planner graph, procedures, or table data. Identity prevents duplicate charges
// when column renders and declaration rows share the same immutable vector.
func keyMetadataContainers(columns, uniques scm.Scmer, containers map[scm.Scmer]uint) {
	add := func(value scm.Scmer) {
		if value.IsSlice() {
			containers[value] = uint(cap(value.Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
		}
	}
	add(columns)
	if columns.IsSlice() {
		for _, row := range columns.Slice() {
			add(row)
		}
	}
	add(uniques)
	if uniques.IsSlice() {
		for _, row := range uniques.Slice() {
			add(row)
			add(foreignMetadataValue(row, "Cols"))
		}
	}
}
func keySnapshotContainers(snapshot *tableShowColumnsSnapshot, containers map[scm.Scmer]uint) {
	if snapshot == nil || snapshot.metadata == nil {
		return
	}
	keyMetadataContainers(snapshot.metadata.columnRows, scm.NewNil(), containers)
	if rows := snapshot.metadata.indexRows; rows.IsSlice() {
		containers[rows] = uint(cap(rows.Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
		for _, row := range rows.Slice() {
			containers[row] = uint(cap(row.Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
			for j := 1; j < len(row.Slice()); j += 2 {
				v := row.Slice()[j]
				if v.IsSlice() {
					containers[v] = uint(cap(v.Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
				}
			}
		}
	}
	if declarations := snapshot.metadata.declarations; declarations.IsSlice() {
		fields := declarations.Slice()
		keyMetadataContainers(fields[1], fields[3].Slice()[1], containers)
	}
}

// The old public containers were table-owned before the single version cut.
// Charge only those now retained by the resource and absent from the new table
// snapshot. New history containers already have their producer's own charge.
// One cold publication pass visits the actual finite metadata history; there
// is no whole-Scheme sizing, row traversal, or per-endpoint history rescan.
func (publication *publicForeignPublication) retainKeyBaselineContainers() {
	type retainedMetadata struct {
		original, live, retained map[scm.Scmer]uint
		endpoint                 *publicForeignEndpoint
	}
	owners := make(map[*table]*retainedMetadata)
	for _, endpoint := range publication.endpoints {
		if !endpoint.keyChanged {
			continue
		}
		state := &retainedMetadata{original: make(map[scm.Scmer]uint), live: make(map[scm.Scmer]uint), retained: make(map[scm.Scmer]uint), endpoint: endpoint}
		keySnapshotContainers(endpoint.table.showColumnsSnapshot.Load(), state.original)
		keySnapshotContainers(endpoint.snapshot, state.live)
		baseline := publication.changes.baselines[endpoint.table].declarations.Slice()
		keyMetadataContainers(baseline[1], baseline[3].Slice()[1], state.retained)
		owners[endpoint.table] = state
	}
	retain := func(t *table, snapshot *tableShowColumnsSnapshot) {
		if state := owners[t]; state != nil {
			keySnapshotContainers(snapshot, state.retained)
		}
	}
	if views := publication.changes.views.Load(); views != nil {
		for t, snapshot := range views.tables {
			retain(t, snapshot)
		}
	}
	if history := publication.changes.owner.catalogAmendments.Load(); history != nil {
		for _, entry := range *history {
			if a := entry.publicKey; a != nil {
				retain(a.table, a.previous)
			}
			if a := entry.column; a != nil {
				retain(a.table, a.previous)
			}
			if a := entry.check; a != nil {
				retain(a.table, a.previous)
			}
			if a := entry.checkDrop; a != nil {
				retain(a.table, a.previous)
			}
			if a := entry.constraintState; a != nil {
				for _, view := range a.views {
					retain(view.table, view.previous)
				}
			}
			if a := entry.foreign; a != nil {
				for i, t := range a.endpoints {
					retain(t, a.previous[i])
				}
			}
			if a := entry.foreignDrop; a != nil {
				for i, t := range a.endpoints {
					retain(t, a.previous[i])
				}
			}
		}
	}
	for _, state := range owners {
		for value, bytes := range state.original {
			if _, held := state.retained[value]; held {
				if _, live := state.live[value]; !live {
					state.endpoint.keyBaselineBytes += bytes
				}
			}
		}
	}
}

// New declaration families consume the same immutable pending key headers.
// Native index IDs remain table-local; explicit constraint names are global.
func (tx *TxContext) pendingUniqueDeclarationName(name string) bool {
	if name == "" {
		return false
	}
	for _, keys := range tx.publicUniqueKeys() {
		for _, key := range keys {
			if key.ConstraintName != "" && strings.EqualFold(key.ConstraintName, name) {
				return true
			}
		}
	}
	return false
}
