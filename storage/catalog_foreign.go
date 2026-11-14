/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"
import "sync/atomic"
import "sort"
import "sync"
import "slices"
import "context"
import "strings"
import "unsafe"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// A public declaration delta retains stable native endpoint identities. Unlike
// CREATE records it never serializes or replaces a public table's row payload.
type catalogForeignDeclaration struct {
	ChildIdentity  string     `json:"child_identity"`
	ParentIdentity string     `json:"parent_identity"`
	Foreign        foreignKey `json:"foreign"`
}

// The immediate native API compiles on privately captured declaration inputs.
// This legacy non-self producer embeds names/ordered columns/modes only; its
// comparisons and supporting access resolve from the current table at DML time.
// Preserve endpoint identity across compilation without invalidating the program
// for unrelated column/key publication. Staged API8/9 owns stricter declaration
// preconditions separately and never uses this maintenance snapshot.
type foreignMaintenanceSnapshot struct {
	db          *database
	incarnation string
	tables      [2]*table
	names       [2]string
	identities  [2]string
}

func captureForeignMaintenance(child, parent *table) foreignMaintenanceSnapshot {
	db := child.schema
	access := db.GetWrite(nil)
	defer access.Release()
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	snapshot := foreignMaintenanceSnapshot{db: db, incarnation: db.incarnationID(), tables: [2]*table{child, parent}}
	for i, t := range snapshot.tables {
		if GetDatabase(db.Name) != db || t.schema != db || db.tables.Get(t.Name) != t || !t.committedAt(db.catalogView().version) {
			panic("foreign declaration endpoint generation was replaced")
		}
		requireTableMaintenance(db.Name, t.Name, maintenanceAlter)
		snapshot.names[i], snapshot.identities[i] = t.Name, t.RuntimeIdentity
	}
	return snapshot
}

// Caller owns schema publication. Other native maintenance may hold DDL before
// schema, so acquisition here is fail-fast rather than a blocking lock upgrade.
// The names are immutable captured strings, not live fields.
func (snapshot *foreignMaintenanceSnapshot) lockDDL() func() {
	tables := []*table{snapshot.tables[0]}
	if snapshot.tables[1] != snapshot.tables[0] {
		if snapshot.names[0] < snapshot.names[1] {
			tables = append(tables, snapshot.tables[1])
		} else {
			tables = []*table{snapshot.tables[1], snapshot.tables[0]}
		}
	}
	for i, t := range tables {
		if !t.ddlMu.TryLock() {
			for j := i - 1; j >= 0; j-- {
				tables[j].ddlMu.Unlock()
			}
			panic("foreign declaration conflict: endpoint maintenance is active; retry statement")
		}
	}
	return func() {
		for i := len(tables) - 1; i >= 0; i-- {
			tables[i].ddlMu.Unlock()
		}
	}
}

// Caller owns both endpoint DDL mutexes and the same schema publication owner.
func (snapshot *foreignMaintenanceSnapshot) validateLocked() {
	db := snapshot.db
	if GetDatabase(db.Name) != db || db.incarnationID() != snapshot.incarnation {
		panic("foreign declaration database was replaced during compilation")
	}
	for i, t := range snapshot.tables {
		// The same schema owner serializes lease creation with this final
		// legacy FK publication. Cache-system header churn remains separate.
		t.requireUnleasedTriggerDeclarations()
		if t.schema != db || db.tables.Get(snapshot.names[i]) != t || t.Name != snapshot.names[i] ||
			t.RuntimeIdentity != snapshot.identities[i] || !t.committedAt(db.catalogView().version) {
			panic("foreign declaration endpoint changed during compilation")
		}
	}
}

// This state belongs to the existing transaction resource/decision owner.
// The first profile admits declaration-only transactions: row mutations by
// this owner are rejected by ordinary catalog admission before native effects.
type publicCatalogViews struct {
	tables         map[*table]*tableShowColumnsSnapshot
	defaultWriters map[*table]struct{}
}

type publicCatalogChanges struct {
	mu                           sync.Mutex
	db                           *database
	owner                        *transactionState
	leases                       map[*table]*tableWriteLease
	baselines                    map[*table]publicForeignBaseline
	views                        atomic.Pointer[publicCatalogViews]
	memory                       *publicCatalogMemory
	anchor                       *tableWriteLease
	leasesHighWater              int
	baselinesHighWater           int
	viewsHighWater               int
	defaultWritersHighWater      int
	historyBytes                 uint
	dbMemoryBytes                uint // numeric native resource/header charge at the database owner
	sourceBaselines              map[string]*ProcedureDescription
	sourceHeads                  map[string]*ProcedureDescription
	sourceReservations           map[string]*ProcedureDescription
	sourceHighWater              int
	constraintStates             map[string]*catalogConstraintState
	constraintStatesHighWater    int
	sourceHistoryBytes           uint
	statementTriggers            map[string]*catalogStatementTriggerSlot
	statementTriggerHighWater    int
	statementTriggerHistoryBytes uint
	amendmentBytes               uint
	publication                  *publicForeignPublication
	modePin                      *databaseLocalDeclarationLease
}

func (changes *publicCatalogChanges) Close() error {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	if changes.owner == nil {
		return nil
	}
	changes.closeSourcesLocked()
	changes.closeStatementTriggersLocked()
	changes.constraintStates = nil
	changes.views.Store(nil)
	changes.owner.catalogAmendments.Store(nil)
	changes.publication = nil
	changes.memory.snapshot.Store(nil)
	changes.memory.publicationBytes.Store(0)
	changes.memory.transferredProgramBytes.Store(0)
	changes.memory.transferredForeignBytes.Store(0)
	changes.memory.transferredCheckBytes.Store(0)
	changes.memory.transferredDefaultBytes.Store(0)
	changes.memory.baselineWrapperBytes.Store(0)
	changes.db.adjustMetadataMemory(changes.dbMemoryBytes, 0)
	changes.dbMemoryBytes = 0
	for _, lease := range changes.leases {
		_ = lease.Close()
	}
	changes.leases = nil
	changes.baselines = nil
	changes.anchor = nil
	changes.owner.publicCatalog.CompareAndSwap(changes, nil)
	changes.owner = nil
	if changes.modePin != nil {
		_ = changes.modePin.Close()
		changes.modePin = nil
	}
	return nil
}

func (tx *TxContext) publicCatalogChanges() *publicCatalogChanges {
	if tx == nil {
		return nil
	}
	return tx.publicCatalog.Load()
}

func (tx *TxContext) stageCatalogForeign(child, parent *table, fk foreignKey, expected *foreignDeclarationExpectations) bool {
	if tx == nil || tx.transactionState == nil {
		panic("public foreign declaration requires an active native transaction")
	}
	finish := tx.beginCatalogConstruction(true)
	defer finish()
	if tx.State != TxActive || tx.decisionStarted {
		panic("public foreign declaration requires an active native transaction")
	}
	requireTableMaintenance(child.schema.Name, child.Name, maintenanceAlter)
	requireTableMaintenance(parent.schema.Name, parent.Name, maintenanceAlter)
	requireSelfForeignModes(child, parent, fk)
	if child.Ephemeral || parent.Ephemeral || child.isEphemeralQueryTable() || parent.isEphemeralQueryTable() {
		panic("public foreign declaration requires durable native table declarations")
	}
	if child.schema != parent.schema || !child.committedAt(child.schema.catalogView().version) || !parent.committedAt(parent.schema.catalogView().version) {
		panic("public foreign declaration requires two committed endpoints in one database")
	}
	if tx.catalogIntents.Load() != nil {
		panic("public and private catalog declarations cannot be mixed")
	}
	if tx.coordinator != nil || activeReplication(child.schema) != nil {
		panic("public foreign declaration requires distributed table admission")
	}
	return tx.stagePublicCatalogDeclaration(child.schema, func(changes *publicCatalogChanges) bool {
		return changes.stageForeign(tx, child, parent, fk, expected)
	})
}

// The existing construction admission is held by this caller. A complete first
// declaration runs INSIDE the resource producer; a failed constructor retires
// its retained state before the factory wakes any other caller.
func (tx *TxContext) stagePublicCatalogDeclaration(db *database, stage func(*publicCatalogChanges) bool) bool {
	modePin := db.admitLocalCatalog(false)
	transferredPin := false
	defer func() {
		if !transferredPin {
			_ = modePin.Close()
		}
	}()
	tx.beginCatalogCreate(db)
	tx.enterPublicCatalogPhase()
	created := false
	firstResult := false
	state := tx.getOrComputeResource("native-public-declarations", scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		created = true
		changes := &publicCatalogChanges{db: db, owner: tx.transactionState, leases: make(map[*table]*tableWriteLease), baselines: make(map[*table]publicForeignBaseline), memory: &publicCatalogMemory{}, modePin: modePin}
		transferredPin = true
		changes.views.Store(&publicCatalogViews{tables: make(map[*table]*tableShowColumnsSnapshot)})
		tx.publicCatalog.Store(changes)
		accepted := false
		defer func() {
			if !accepted {
				// The stage producer unwinds its mutex and partial reservations
				// first. The resource factory removes this failed entry only
				// after its complete native value has been retired.
				if err := changes.Close(); err != nil {
					panic(fmt.Errorf("public declaration constructor cleanup failed: %w", err))
				}
			}
		}()
		firstResult = stage(changes)
		accepted = true
		return scm.NewAny(changes)
	})).Any().(*publicCatalogChanges)
	if created {
		return firstResult
	}
	return stage(state)
}

// The first call runs inside the native resource constructor; later callers
// reuse precisely the same admitted stage after the existing factory completes.
func (state *publicCatalogChanges) stageForeign(tx *TxContext, child, parent *table, fk foreignKey, expected *foreignDeclarationExpectations) bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	defer state.refreshMemoryLocked()
	endpoints := foreignEndpointTables(child, parent)
	sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].Name < endpoints[j].Name })
	var acquired []*table
	accepted := false
	defer func() {
		if !accepted {
			for _, t := range acquired {
				if state.anchor == state.leases[t] {
					state.anchor = nil
				}
				_ = state.leases[t].Close()
				delete(state.leases, t)
				delete(state.baselines, t)
			}
		}
	}()
	for _, t := range endpoints {
		if state.leases[t] != nil {
			continue
		}
		lease, err := t.writeAdmission.admit(scanQueryContext(tx, context.Background()), tx.transactionState, true, false)
		if err != nil {
			panic(err)
		}
		state.leases[t] = lease
		state.leasesHighWater = max(state.leasesHighWater, len(state.leases))
		acquired = append(acquired, t)
		if state.anchor == nil {
			state.refreshMemoryLocked()
			lease.attachRetainedMemory(state.memory)
			state.anchor = lease
		}
	}
	db := state.db
	declaration := catalogForeignDeclaration{child.RuntimeIdentity, parent.RuntimeIdentity, fk}
	var lookup *foreignKeyLookup
	var observed map[*table]*tableShowColumnsSnapshot
	func() {
		access := db.GetWrite(tx)
		defer access.Release()
		db.schemalock.Lock()
		defer db.schemalock.Unlock()
		state.validateStageLocked(tx, child, parent, declaration, expected)
		observed = make(map[*table]*tableShowColumnsSnapshot, len(endpoints))
		for _, t := range endpoints {
			observed[t] = t.showColumnsSnapshot.Load()
		}
		lookup = newForeignKeyLookup(parent, fk.Cols2)
	}()
	// Both writer leases remain owned through the native transaction decision.
	// Existing rows and generated programs are inspected privately, outside the
	// schema mutex; no trigger body runs and no relational row bag is retained.
	validateExistingForeignRows(tx, child, fk, lookup)
	programs := buildFKTriggers(db, child, parent, fk)
	captureForeignProgramFootprints(programs)
	releasePrograms := reserveForeignTriggerPrograms(tx.transactionState, programs)
	defer func() {
		if !accepted {
			releasePrograms()
		}
	}()
	amendment := &catalogForeignAmendment{declaration: declaration, programs: programs}
	amendment.retainedBytes = foreignAmendmentFootprint(amendment) + uint(unsafe.Sizeof(catalogConstraintState{}))
	views := maps.Clone(state.views.Load().tables)
	for i, t := range endpoints {
		amendment.endpoints[i] = t
		amendment.previous[i] = views[t]
		base := amendment.previous[i]
		if base == nil {
			base = observed[t]
		}
		var bytes uint
		views[t], bytes = appendConstraintMetadata(t, base, []foreignKey{fk}, nil, 0)
		amendment.retainedBytes += bytes
	}
	history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{foreign: amendment})
	access := db.GetWrite(tx)
	defer access.Release()
	db.schemalock.Lock()
	defer db.schemalock.Unlock()
	state.validateStageLocked(tx, child, parent, declaration, expected)
	for t, before := range observed {
		if t.showColumnsSnapshot.Load().metadata.declarations != before.metadata.declarations {
			panic("public declaration conflict: endpoint declaration changed during validation")
		}
	}
	for _, t := range endpoints {
		if _, exists := state.baselines[t]; !exists {
			state.baselines[t] = foreignBaseline(t.showColumnsSnapshot.Load())
		}
	}
	state.baselinesHighWater = max(state.baselinesHighWater, len(state.baselines))
	state.viewsHighWater = max(state.viewsHighWater, len(views))
	state.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	state.amendmentBytes += amendment.retainedBytes
	if state.constraintStates == nil {
		state.constraintStates = make(map[string]*catalogConstraintState)
	}
	state.constraintStates[fk.Id] = &catalogConstraintState{addition: amendment, enabled: true, trusted: true}
	state.constraintStatesHighWater = max(state.constraintStatesHighWater, len(state.constraintStates))
	tx.catalogAmendments.Store(&history)
	state.publishViews(history, views)
	accepted = true
	return true
}

// Caller holds this transaction's amendment mutex and the schema publication
// owner. Repeating the complete proof after validation preserves the same
// declaration semantics through optimistic maintenance and data reads.
func (state *publicCatalogChanges) validateStageLocked(tx *TxContext, child, parent *table, declaration catalogForeignDeclaration, expected *foreignDeclarationExpectations) {
	db := state.db
	if activeReplication(db) != nil {
		panic("public foreign declaration requires distributed table admission")
	}
	identity := tx.catalogIdentity.Load()
	if GetDatabase(db.Name) != db || identity == nil || identity.database != db || identity.Incarnation != db.incarnationID() {
		panic("public declaration lost its native catalog owner")
	}
	for t, before := range state.baselines {
		if db.tables.Get(t.Name) != t || t.showColumnsSnapshot.Load().metadata.declarations != before.declarations {
			panic("public declaration conflict: endpoint declaration changed")
		}
	}
	validateCatalogForeign(db, declaration)
	fk := declaration.Foreign
	if expected != nil {
		expected.validate(child, parent, fk)
	}
	if db.hasDeclarationNameLocked(fk.Id, nil) {
		panic("declaration name already exists: " + fk.Id)
	}
	for _, existing := range tx.publicForeignAmendments() {
		if existing.foreign != nil && strings.EqualFold(existing.foreign.declaration.Foreign.Id, fk.Id) || existing.check != nil && strings.EqualFold(existing.check.declaration.Check.Id, fk.Id) {
			panic("declaration name already exists: " + fk.Id)
		}
	}
	if catalogForeignCycle(db, tx.publicForeignAmendments(), fk.Tbl1, fk.Tbl2) {
		panic("transactional cyclic foreign declarations are not supported")
	}
}

// Read-only validation belongs to the public metadata transaction. Its ordinary
// native scans preserve query cancellation, visibility and reader lifetimes;
// the endpoint admission owner prevents writers through publication/rollback.
func validateExistingForeignRows(tx *TxContext, child *table, fk foreignKey, parent *foreignKeyLookup) {
	validate := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		for _, value := range values[1:] {
			if value.IsNil() {
				return scm.NewBool(true)
			}
		}
		if !parent.exists(tx, values[1:]) {
			panic(fmt.Errorf("foreign key constraint %s failed: existing child row has no parent", fk.Id))
		}
		return scm.NewBool(true)
	})
	child.scan(tx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
		nil, scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
		fk.Cols1, validate, scm.NewBool(true),
		scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), false)
}

func validateCatalogForeign(db *database, declaration catalogForeignDeclaration) {
	fk := declaration.Foreign
	child, parent := db.tables.Get(fk.Tbl1), db.tables.Get(fk.Tbl2)
	if child == nil || parent == nil || child.RuntimeIdentity != declaration.ChildIdentity || parent.RuntimeIdentity != declaration.ParentIdentity ||
		!child.committedAt(db.catalogVersion) || !parent.committedAt(db.catalogVersion) {
		panic("foreign declaration endpoint generation was replaced")
	}
	for _, identity := range []string{declaration.ChildIdentity, declaration.ParentIdentity} {
		if _, err := uuid.Parse(identity); err != nil {
			panic("invalid native foreign endpoint identity")
		}
	}
	if fk.Id == "" || len(fk.Cols1) == 0 || len(fk.Cols1) != len(fk.Cols2) || fk.Updatemode > SETNULL || fk.Deletemode > SETNULL {
		panic("invalid native foreign declaration")
	}
	requireSelfForeignModes(child, parent, fk)
	for endpoint, t := range []*table{child, parent} {
		cols := fk.Cols1
		if endpoint == 1 {
			cols = fk.Cols2
		}
		seen := make(map[string]bool)
		for _, name := range cols {
			if seen[name] {
				panic("duplicate foreign declaration column: " + name)
			}
			seen[name] = true
			if _, ok := t.ResolveColumnName(name, false); !ok {
				panic("unknown foreign declaration column: " + name)
			}
			if child == parent && !foreignStoredComparisonColumn(t.uniqueColumnDeclaration(name)) {
				panic("self foreign declaration requires ordinary stored native columns: " + name)
			}
		}
	}
	for _, key := range parent.Unique {
		if slices.Equal(key.Cols, fk.Cols2) {
			return
		}
	}
	panic("foreign declaration requires an existing native unique target")
}

// An amendment retains only its two changed endpoint views. Program generation
// and the pure durable declaration share this existing native history entry;
// there is no second mutable foreign-key catalog to rebuild on every ADD.
type catalogForeignAmendment struct {
	declaration   catalogForeignDeclaration
	programs      []foreignKeyTrigger
	endpoints     [2]*table
	previous      [2]*tableShowColumnsSnapshot
	retainedBytes uint
}

func (tx *TxContext) publicForeignAmendments() []catalogDeclarationAmendment {
	if tx == nil || tx.publicCatalogChanges() == nil {
		return nil
	}
	if history := tx.catalogAmendments.Load(); history != nil {
		return *history
	}
	return nil
}

func (tx *TxContext) publicDeclarationPosition() int {
	return len(tx.publicForeignAmendments())
}

func (tx *TxContext) restorePublicDeclarations(keep int) {
	changes := tx.publicCatalogChanges()
	if changes == nil {
		return
	}
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	if len(changes.sourceBaselines) != 0 || len(changes.statementTriggers) != 0 {
		access := changes.db.GetWrite(nil)
		defer access.Release()
		changes.db.schemalock.Lock()
		defer changes.db.schemalock.Unlock()
	}
	history := tx.publicForeignAmendments()
	views := maps.Clone(changes.views.Load().tables)
	if keep < 0 || keep > len(history) {
		panic("invalid native public declaration boundary")
	}
	for i := len(history) - 1; i >= keep; i-- {
		if amendment := history[i].columnDefault; amendment != nil {
			changes.amendmentBytes -= amendment.retainedBytes
			if amendment.previous == nil {
				delete(views, amendment.table)
			} else {
				views[amendment.table] = amendment.previous
			}
			continue
		}
		if trigger := history[i].statementTrigger; trigger != nil {
			changes.restoreStatementTriggerLocked(trigger)
			continue
		}
		if source := history[i].source; source != nil {
			changes.restoreSourceLocked(source)
			continue
		}
		if state := history[i].constraintState; state != nil {
			changes.amendmentBytes -= state.retainedBytes
			for _, before := range state.previous {
				changes.constraintStates[before.name()] = before
			}
			for _, before := range state.views {
				views[before.table] = before.previous
			}
			continue
		}
		if amendment := history[i].check; amendment != nil {
			delete(changes.constraintStates, amendment.declaration.Check.Id)
			changes.amendmentBytes -= amendment.retainedBytes
			releaseForeignTriggerPrograms(tx.transactionState, amendment.programs)
			if amendment.previous == nil {
				delete(views, amendment.table)
			} else {
				views[amendment.table] = amendment.previous
			}
			continue
		}
		amendment := history[i].foreign
		if amendment == nil {
			panic("unsupported public declaration history kind")
		}
		delete(changes.constraintStates, amendment.declaration.Foreign.Id)
		changes.amendmentBytes -= amendment.retainedBytes
		releaseForeignTriggerPrograms(tx.transactionState, amendment.programs)
		for j, t := range amendment.endpoints {
			if t == nil {
				continue
			}
			if before := amendment.previous[j]; before != nil {
				views[t] = before
			} else {
				delete(views, t)
			}
		}
	}
	changes.publishViews(history[:keep], views)
	if keep == 0 {
		tx.catalogAmendments.Store(nil)
		changes.historyBytes = 0
	} else {
		remaining := append([]catalogDeclarationAmendment(nil), history[:keep]...)
		tx.catalogAmendments.Store(&remaining)
		changes.historyBytes = uint(unsafe.Sizeof(remaining)) + uint(cap(remaining))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	}
}

func (tx *TxContext) publicForeignDeclarations() []catalogForeignDeclaration {
	declarations := make([]catalogForeignDeclaration, 0)
	for _, amendment := range tx.publicForeignAmendments() {
		if amendment.foreign != nil {
			declarations = append(declarations, amendment.foreign.declaration)
		}
	}
	return declarations
}

func (tx *TxContext) validatePublicDeclarations(db *database) error {
	changes := tx.publicCatalogChanges()
	if changes == nil {
		return nil
	}
	if activeReplication(db) != nil {
		return fmt.Errorf("public foreign declaration requires distributed table admission")
	}
	changes.mu.Lock()
	defer changes.mu.Unlock()
	for t, base := range changes.baselines {
		current := db.tables.Get(t.Name)
		if current != t || current.showColumnsSnapshot.Load().metadata.declarations != base.declarations {
			return fmt.Errorf("public declaration conflict: endpoint declaration changed")
		}
	}
	if err := changes.validateConstraintStatesLocked(); err != nil {
		return err
	}
	if err := changes.validateSourcesLocked(); err != nil {
		return err
	}
	if err := changes.validateStatementTriggersLocked(); err != nil {
		return err
	}
	if err := changes.validateColumnDefaultsLocked(); err != nil {
		return err
	}
	for _, amendment := range tx.publicForeignAmendments() {
		if amendment.foreign == nil {
			continue
		}
		declaration := amendment.foreign.declaration
		validateCatalogForeign(db, declaration)
		for _, program := range amendment.foreign.programs {
			program.table.validateDeclarationTrigger(tx.transactionState, program.trigger.Name)
		}
		if db.hasDeclarationNameLocked(declaration.Foreign.Id, nil) {
			return fmt.Errorf("public declaration conflict: declaration name already exists")
		}
	}
	return tx.validatePublicChecks(db)
}

// The native schema owner prepares all programs and immutable descriptive data
// before the durable decision. Publication executes no trigger body/compiler.
type publicForeignEndpoint struct {
	table                *table
	columns              []*column
	checks               []*checkDeclaration
	foreign              []foreignKey
	snapshot             *tableShowColumnsSnapshot
	baselineWrapperBytes uint
}
type publicForeignPublication struct {
	endpoints           []*publicForeignEndpoint
	programs            []foreignKeyTrigger
	changes             *publicCatalogChanges
	checkPayloadBytes   uint
	foreignPayloadBytes uint
	defaultPayloadBytes uint
}

func appendConstraintMetadata(t *table, base *tableShowColumnsSnapshot, additions []foreignKey, checks []*checkDeclaration, version uint64) (*tableShowColumnsSnapshot, uint) {
	// Private views carry the same immutable column/default directory while
	// changing constraints. Do not retain the planner/statistics owner graph.
	var result tableShowColumnsSnapshot
	var metadata tableShowColumnsSnapshotMetadata
	if version != 0 {
		result, metadata = *base, *base.metadata
	}
	result.metadata = &metadata
	if version == 0 {
		// Keep the same private column/default declaration view while adding
		// constraints; native consumers borrow this directory by point lookup.
		metadata.columnNames = base.metadata.columnNames
		metadata.columnRows = base.metadata.columnRows
	}
	rows := t.buildForeignConstraintMetadata(additions).Slice()
	if version == 0 {
		// These additions passed the staged historical proof. Only the private
		// view describes its transient state: legacy native7 has no historical
		// trust proof, and no persistent disabled-state format is introduced.
		for i, row := range rows {
			fields := append(row.Slice(), scm.NewString("Enabled"), scm.NewBool(true), scm.NewString("Trusted"), scm.NewBool(true))
			rows[i] = scm.NewSlice(fields)
		}
	}
	old := base.metadata.constraints.Slice()
	all := append(append([]scm.Scmer(nil), old[3].Slice()...), rows...)
	constraints := append([]scm.Scmer(nil), old...)
	constraints[3] = scm.NewSlice(all)
	var checkRows []scm.Scmer
	var newCheckRows []scm.Scmer
	if len(checks) != 0 {
		newCheckRows = checkConstraintMetadata(checks, version == 0).Slice()
		if len(old) == 6 {
			checkRows = append([]scm.Scmer(nil), old[5].Slice()...)
		}
		checkRows = append(checkRows, newCheckRows...)
		if len(constraints) == 4 {
			constraints = append(constraints, scm.NewString("Checks"), scm.NewSlice(checkRows))
		} else {
			constraints[5] = scm.NewSlice(checkRows)
		}
	}
	metadata.constraints = scm.NewSlice(constraints)
	metadata.foreignByID = buildForeignMetadataDirectory(metadata.constraints)
	declarations := append([]scm.Scmer(nil), base.metadata.declarations.Slice()...)
	declarations[3] = metadata.constraints
	metadata.declarations = scm.NewSlice(declarations)
	if version != 0 {
		previous, previousMetadata := *base, *base.metadata
		previous.metadata = &previousMetadata
		previousMetadata.catalogVersion = 0
		previousMetadata.catalogPrevious = nil
		metadata.catalogVersion = version
		metadata.catalogPrevious = &previous
	}
	bytes := foreignMetadataContainers(&result, rows, all, declarations)
	bytes += uint(cap(checkRows)) * uint(unsafe.Sizeof(scm.Scmer{}))
	for _, row := range newCheckRows {
		bytes += uint(cap(row.Slice())+cap(row.Slice()[3].Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
	}
	if version != 0 {
		bytes += uint(unsafe.Sizeof(tableShowColumnsSnapshot{})) + uint(unsafe.Sizeof(tableShowColumnsSnapshotMetadata{}))
	}
	return &result, bytes
}

func (db *database) preparePublicConstraintPublication(declarations []catalogForeignDeclaration, checks []catalogCheckDeclaration, defaults []catalogColumnDefaultDeclaration, version uint64, programs map[string][]foreignKeyTrigger, changes *publicCatalogChanges) *publicForeignPublication {
	publication := &publicForeignPublication{changes: changes}
	endpoints := make(map[*table]*publicForeignEndpoint)
	additions := make(map[*table][]foreignKey)
	for _, declaration := range declarations {
		validateCatalogForeign(db, declaration)
		fk := declaration.Foreign
		publication.foreignPayloadBytes += fk.payloadFootprint
		child, parent := db.tables.Get(fk.Tbl1), db.tables.Get(fk.Tbl2)
		publication.programs = append(publication.programs, programs[fk.Id]...)
		for _, t := range foreignEndpointTables(child, parent) {
			endpoint := endpoints[t]
			if endpoint == nil {
				endpoint = &publicForeignEndpoint{table: t, checks: append([]*checkDeclaration(nil), t.Checks...), foreign: append([]foreignKey(nil), t.Foreign...), baselineWrapperBytes: changes.baselines[t].wrapperBytes}
				endpoints[t] = endpoint
				publication.endpoints = append(publication.endpoints, endpoint)
			}
			key := fk
			if t != child {
				key = sharedForeignMirror(key)
			}
			endpoint.foreign = append(endpoint.foreign, key)
			additions[t] = append(additions[t], fk)
		}
	}
	checkAdditions := make(map[*table][]*checkDeclaration)
	for _, declaration := range checks {
		validateCatalogCheck(db, declaration)
		t := db.tables.Get(declaration.Table)
		endpoint := endpoints[t]
		if endpoint == nil {
			endpoint = &publicForeignEndpoint{table: t, checks: append([]*checkDeclaration(nil), t.Checks...), foreign: append([]foreignKey(nil), t.Foreign...), baselineWrapperBytes: changes.baselines[t].wrapperBytes}
			endpoints[t] = endpoint
			publication.endpoints = append(publication.endpoints, endpoint)
		}
		endpoint.checks = append(endpoint.checks, declaration.Check)
		checkAdditions[t] = append(checkAdditions[t], declaration.Check)
		publication.checkPayloadBytes += declaration.Check.payloadFootprint
		publication.programs = append(publication.programs, programs[declaration.Check.Id]...)
	}
	for _, declaration := range defaults {
		t := validateCatalogColumnDefault(db, declaration)
		if endpoints[t] == nil {
			endpoint := &publicForeignEndpoint{table: t, checks: append([]*checkDeclaration(nil), t.Checks...), foreign: append([]foreignKey(nil), t.Foreign...), baselineWrapperBytes: changes.baselines[t].wrapperBytes}
			endpoints[t] = endpoint
			publication.endpoints = append(publication.endpoints, endpoint)
		}
	}
	bytes := uint(unsafe.Sizeof(*publication)) + uint(cap(publication.endpoints))*uint(unsafe.Sizeof((*publicForeignEndpoint)(nil)))
	bytes += uint(cap(publication.programs)) * uint(unsafe.Sizeof(foreignKeyTrigger{}))
	for _, endpoint := range publication.endpoints {
		var snapshotBytes uint
		endpoint.snapshot, snapshotBytes = appendConstraintMetadata(endpoint.table, endpoint.table.showColumnsSnapshot.Load(), additions[endpoint.table], checkAdditions[endpoint.table], version)
		if history := changes.owner.catalogAmendments.Load(); history != nil {
			columns, payloadBytes := publicColumnDefaultColumns(*history, endpoint.table)
			if columns != nil {
				var defaultBytes uint
				endpoint.columns = columns
				endpoint.snapshot, defaultBytes = replaceColumnDefaultMetadata(endpoint.table, endpoint.snapshot, columns)
				snapshotBytes += defaultBytes
				publication.defaultPayloadBytes += payloadBytes
			}
		}
		bytes += uint(unsafe.Sizeof(*endpoint)) + uint(cap(endpoint.foreign))*uint(unsafe.Sizeof(foreignKey{})) + uint(cap(endpoint.checks))*uint(unsafe.Sizeof((*checkDeclaration)(nil))) + snapshotBytes
	}
	// Native decision preparation has drained construction. This retains the
	// actual publication containers until resource Close, not a temporary map.
	changes.publication = publication
	changes.memory.publicationBytes.Store(uint64(bytes))
	return publication
}

func (publication *publicForeignPublication) apply() {
	for _, endpoint := range publication.endpoints {
		if endpoint.columns != nil {
			endpoint.table.Columns = endpoint.columns
			endpoint.table.columnNamesSnapshot.Store(endpoint.snapshot.metadata.columnNames)
		}
		endpoint.table.Foreign = endpoint.foreign
		endpoint.table.Checks = endpoint.checks
		endpoint.table.refreshForeignMemoryLocked()
		endpoint.table.refreshCheckMemoryLocked()
	}
	publication.changes.memory.transferredForeignBytes.Add(uint64(publication.foreignPayloadBytes))
	publication.changes.memory.transferredCheckBytes.Add(uint64(publication.checkPayloadBytes))
	publication.changes.memory.transferredDefaultBytes.Add(uint64(publication.defaultPayloadBytes))
	// Preserve user/runtime cache triggers registered concurrently with native
	// preparation. Only these already-compiled FK programs are added.
	for _, program := range publication.programs {
		program.table.mu.Lock()
		// Every name is reserved through this accepted native decision. Public
		// and system publishers cannot acquire it after prepare; publication
		// performs no newly failing declaration check after durable acceptance.
		program.table.insertTriggerLocked(program.trigger)
		publication.changes.memory.transferredProgramBytes.Add(uint64(constraintProgramBytes(program.trigger)))
		program.table.mu.Unlock()
	}
	for _, endpoint := range publication.endpoints {
		endpoint.table.showColumnsSnapshot.Store(endpoint.snapshot)
		endpoint.table.publishPlannerStatsToken()
	}
}

// Reservations use the same endpoint admission lease as historical validation.
// A failed batch releases its prefix; accepted names live in the amendment
// history until savepoint restoration or transaction resource retirement.
func reserveForeignTriggerPrograms(transaction *transactionState, programs []foreignKeyTrigger) (release func()) {
	var reserved []foreignKeyTrigger
	accepted := false
	defer func() {
		if !accepted {
			releaseForeignTriggerPrograms(transaction, reserved)
		}
	}()
	for _, program := range programs {
		program.table.reserveDeclarationTrigger(transaction, program.trigger.Name, false)
		reserved = append(reserved, program)
	}
	accepted = true
	return func() { releaseForeignTriggerPrograms(transaction, reserved) }
}

func releaseForeignTriggerPrograms(transaction *transactionState, programs []foreignKeyTrigger) {
	for _, program := range programs {
		program.table.releaseDeclarationTrigger(transaction, program.trigger.Name)
	}
}

func (publication *publicForeignPublication) retirePrevious() {
	// This runs only AFTER the durable version is visible. Subsequent retries
	// skip apply at that version, while crash replay uses the durable record.
	// References now owned by the real table must no longer be kept/charged by
	// this private publication. History still retains its own distinct wrappers.
	var baselineBytes uint
	for _, endpoint := range publication.endpoints {
		for {
			current := endpoint.table.showColumnsSnapshot.Load()
			if current.metadata.catalogPrevious == nil {
				break
			}
			replacement, metadata := *current, *current.metadata
			replacement.metadata = &metadata
			metadata.catalogVersion = 0
			metadata.catalogPrevious = nil
			if endpoint.table.showColumnsSnapshot.CompareAndSwap(current, &replacement) {
				break
			}
		}
		endpoint.snapshot = nil
		endpoint.columns = nil
		endpoint.foreign = nil
		endpoint.checks = nil
		baselineBytes += endpoint.baselineWrapperBytes
	}
	clear(publication.programs)
	publication.programs = nil
	bytes := uint(unsafe.Sizeof(*publication)) + uint(cap(publication.endpoints))*uint(unsafe.Sizeof((*publicForeignEndpoint)(nil)))
	bytes += uint(len(publication.endpoints)) * uint(unsafe.Sizeof(publicForeignEndpoint{}))
	publication.changes.memory.publicationBytes.Store(uint64(bytes))
	publication.changes.memory.baselineWrapperBytes.Store(uint64(baselineBytes))
}

// Recovery constructs declarations before ordinary loader initialization. System
// programs and SHOW metadata are rebuilt by that loader after all rows/tables
// have their existing native back-references restored.
func (db *database) recoverPublicForeignDeclarations(declarations []catalogForeignDeclaration) {
	for _, declaration := range declarations {
		validateCatalogForeign(db, declaration)
		fk := declaration.Foreign
		child, parent := db.tables.Get(fk.Tbl1), db.tables.Get(fk.Tbl2)
		child.Foreign = append(child.Foreign, fk)
		child.refreshForeignMemoryLocked()
		if parent != child {
			parent.Foreign = append(parent.Foreign, sharedForeignMirror(fk))
			parent.refreshForeignMemoryLocked()
		}
	}
}

func (t *table) catalogVisibleMetadata() *tableShowColumnsSnapshot {
	snapshot := t.showColumnsSnapshot.Load()
	if snapshot != nil && snapshot.metadata.catalogPrevious != nil && t.schema.catalogView().version < snapshot.metadata.catalogVersion {
		return snapshot.metadata.catalogPrevious
	}
	return snapshot
}

// The transaction resource publishes one immutable endpoint directory. A SHOW
// projection neither waits for builders nor reads mutable transaction state;
// retiring the accepted resource removes the private view atomically.
func (t *table) declarationMetadataForTx(tx *TxContext) *tableShowColumnsSnapshot {
	if tx != nil {
		if changes := tx.publicCatalogChanges(); changes != nil && changes.db == t.schema {
			if views := changes.views.Load(); views != nil {
				if view := views.tables[t]; view != nil {
					return view
				}
			}
		}
	}
	return t.catalogVisibleMetadata()
}

// Existing and staged outbound references form one native dependency graph.
// The bounded first profile does not claim recursive cascade execution.
func catalogForeignCycle(db *database, pending []catalogDeclarationAmendment, child, parent string) bool {
	// The caller has already admitted only RESTRICT/RESTRICT for this case.
	// It is not a recursive cascade edge; ordinary multi-table cycles retain
	// their existing graph rejection below.
	if child == parent {
		return false
	}
	visited := make(map[string]bool)
	var reaches func(string) bool
	reaches = func(name string) bool {
		if name == child {
			return true
		}
		if visited[name] {
			return false
		}
		visited[name] = true
		if t := db.tables.Get(name); t != nil {
			for _, fk := range t.Foreign {
				if fk.Tbl1 == name && reaches(fk.Tbl2) {
					return true
				}
			}
		}
		for _, amendment := range pending {
			if amendment.foreign == nil {
				continue
			}
			declaration := amendment.foreign.declaration
			if declaration.Foreign.Tbl1 == name && reaches(declaration.Foreign.Tbl2) {
				return true
			}
		}
		return false
	}
	return reaches(parent)
}

func (tx *TxContext) publicConstraintPrograms() map[string][]foreignKeyTrigger {
	history := tx.publicForeignAmendments()
	programs := make(map[string][]foreignKeyTrigger, len(history))
	for _, amendment := range history {
		if amendment.foreign != nil {
			programs[amendment.foreign.declaration.Foreign.Id] = amendment.foreign.programs
		}
		if amendment.check != nil {
			programs[amendment.check.declaration.Check.Id] = amendment.check.programs
		}
	}
	return programs
}

// Expected column facts are a bounded native declaration precondition supplied
// by a frontend. All fields are opaque storage declarations, not SQL policies.
type foreignColumnExpectation struct {
	name       string
	typ        string
	dimensions []int64
	collation  string
	nullable   bool
	computed   bool
}
type foreignDeclarationExpectations struct {
	child  []foreignColumnExpectation
	parent []foreignColumnExpectation
}

func decodeForeignExpectations(value scm.Scmer) *foreignDeclarationExpectations {
	if !value.IsSlice() || len(value.Slice()) != 4 {
		panic("foreign declaration options require both expected endpoint declarations")
	}
	result := &foreignDeclarationExpectations{}
	seen := make(map[string]bool, 2)
	options := value.Slice()
	for i := 0; i < len(options); i += 2 {
		if !options[i].IsString() || !options[i+1].IsSlice() {
			panic("invalid foreign declaration options")
		}
		key := options[i].String()
		if (key != "expected_child" && key != "expected_parent") || seen[key] {
			panic("invalid foreign declaration expected endpoint")
		}
		seen[key] = true
		rows := options[i+1].Slice()
		facts := make([]foreignColumnExpectation, len(rows))
		for j, row := range rows {
			if !row.IsSlice() || len(row.Slice()) != 6 {
				panic("foreign expected column requires six declaration fields")
			}
			fields := row.Slice()
			if !fields[0].IsString() || !fields[1].IsString() || !fields[2].IsSlice() || !fields[3].IsString() || !fields[4].IsBool() || !fields[5].IsBool() {
				panic("invalid foreign expected column declaration fields")
			}
			dimensions := make([]int64, len(fields[2].Slice()))
			for k, dimension := range fields[2].Slice() {
				if !dimension.IsInt() {
					panic("foreign expected column dimensions must be integers")
				}
				dimensions[k] = dimension.Int()
			}
			facts[j] = foreignColumnExpectation{fields[0].String(), fields[1].String(), dimensions, fields[3].String(), fields[4].Bool(), fields[5].Bool()}
		}
		if key == "expected_child" {
			result.child = facts
		} else {
			result.parent = facts
		}
	}
	return result
}
func (expected *foreignDeclarationExpectations) validate(child, parent *table, fk foreignKey) {
	for endpoint, t := range []*table{child, parent} {
		facts, names := expected.child, fk.Cols1
		if endpoint == 1 {
			facts, names = expected.parent, fk.Cols2
		}
		if len(facts) != len(names) {
			panic("foreign expected column count does not match its declaration")
		}
		for i, fact := range facts {
			if fact.name != names[i] {
				panic("foreign expected column name does not match its declaration")
			}
			var current *column
			for _, c := range t.Columns {
				if c.Name == fact.name {
					current = c
					break
				}
			}
			if current == nil || current.Typ != fact.typ || current.Collation != fact.collation || current.AllowNull != fact.nullable || isRuntimeComputedColumn(current) != fact.computed || len(current.Typdimensions) != len(fact.dimensions) {
				panic("foreign endpoint declaration changed before staging")
			}
			for j, dimension := range current.Typdimensions {
				if int64(dimension) != fact.dimensions[j] {
					panic("foreign endpoint declaration changed before staging")
				}
			}
		}
	}
}
