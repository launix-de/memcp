/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"
import "sort"
import "unsafe"
import "strings"
import "unicode/utf8"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// V16 appends only a fresh, nullable ordinary declaration. There is no private
// packed slot or stored value: old generations already represent its NULLs.
// A preceding or subsequent write to the same endpoint is not in this profile.
type catalogColumnAppend struct {
	Table          string                       `json:"table"`
	Identity       string                       `json:"identity"`
	Ordinal        int                          `json:"ordinal"`
	Name           string                       `json:"name"`
	Type           storedColumnType             `json:"type"`
	EnumValues     []string                     `json:"enum_values,omitempty"`
	Properties     columnDeclarationProperties  `json:"properties,omitempty"`
	Comment        string                       `json:"comment,omitempty"`
	Initialization *catalogColumnInitialization `json:"initialization,omitempty"`
}

func (d catalogColumnAppend) column() *column {
	c := &column{Name: d.Name, Typ: d.Type.Type, Typdimensions: append([]int(nil), d.Type.Dimensions...), AllowNull: d.Type.Nullable, Collation: d.Type.Collation, EnumValues: append([]string(nil), d.EnumValues...), DeclarationProperties: maps.Clone(d.Properties), Comment: d.Comment}
	if d.Initialization != nil {
		c.Default, c.DefaultName = d.Initialization.Default, d.Initialization.DefaultName
		c.InitialValue, c.InitialValues = d.Initialization.Value, maps.Clone(d.Initialization.Extents)
	}
	c.UpdateSanitizer()
	if d.Initialization != nil && c.sanitizer != nil {
		c.Default = c.sanitizer(c.Default)
		c.InitialValue = c.sanitizer(c.InitialValue)
	}
	return c
}

// Fresh declarations own these scalar strings; same-slot clones normally
// borrow them. This finite payload walk never traverses executable values.
func columnAppendStrings(name, typ, collation, comment string, labels []string, properties columnDeclarationProperties) uint {
	bytes := uint(len(name) + len(typ) + len(collation) + len(comment))
	for _, label := range labels {
		bytes += uint(len(label))
	}
	for key, value := range properties {
		bytes += uint(len(key)) + scm.ComputeSize(value) - uint(unsafe.Sizeof(scm.Scmer{}))
	}
	return bytes
}

func (d *catalogColumnAppend) containers() uint {
	bytes := uint(unsafe.Sizeof(*d)) + uint(cap(d.Type.Dimensions))*uint(unsafe.Sizeof(int(0))) + uint(cap(d.EnumValues))*uint(unsafe.Sizeof(""))
	if d.Properties != nil {
		bytes += 64 + 64*uint(len(d.Properties))
	}
	return bytes
}

func (d *catalogColumnAppend) memory() uint {
	if d == nil {
		return 0
	}
	// A decoded durable envelope owns independent scalar payloads as well.
	return d.containers() + uint(len(d.Table)+len(d.Identity)) + columnAppendStrings(d.Name, d.Type.Type, d.Type.Collation, d.Comment, d.EnumValues, d.Properties) + d.Initialization.memory()
}

func (tx *TxContext) appendPublicColumn(t *table, name, typ string, dimensions []int, options []scm.Scmer) bool {
	finish := tx.beginCatalogConstruction(catalogValidationConstruction, t)
	defer func() { finish(); tx.retireUnacceptedPublicCatalogPhase() }()
	if tx.State != TxActive || tx.decisionStarted || tx.coordinator != nil || tx.remoteObject != nil || tx.remoteInvocation || tx.catalogIntents.Load() != nil || tx.hasTableDrops() || t.Ephemeral || t.ScopedSchema || t.PersistencyMode == Cache || !localCatalogRowDatabase(t.schema) {
		panic("nullable append requires a local committed native table")
	}
	for i := 0; i+1 < len(options); i += 2 {
		switch scm.String(options[i]) {
		case "null", "fill_existing", "name_ignorecase", "collate", "properties", "comment", "enum_values", "default", "default_name":
		default:
			panic("transactional nullable append requires an ordinary column without default or computation")
		}
	}
	definition := decodeColumnDefinition(name, typ, dimensions, options)
	c := definition.column
	if name == "" || c.IsTemp || c.AutoIncrement || len(definition.unique) != 0 || (!c.AllowNull && !definition.hasDefault) {
		panic("transactional append requires a fresh nullable column without a default")
	}
	if definition.hasDefault {
		if err := validateColumnInitialScalar(c.Default); err != nil {
			panic(err)
		}
		if c.sanitizer != nil {
			c.Default = c.sanitizer(c.Default)
		}
		if err := validateColumnInitialScalar(c.Default); err != nil {
			panic(err)
		}
	}
	requireTableMaintenance(t.schema.Name, t.Name, maintenanceAlter)
	return tx.stagePublicCatalogDeclaration(t.schema, catalogColumnKind, func(changes *publicCatalogChanges) bool {
		return changes.stageColumnAppend(tx, t, c, definition.hasDefault, definition.fillExisting)
	})
}

func (changes *publicCatalogChanges) stageColumnAppend(tx *TxContext, t *table, c *column, initialized, fillExisting bool) bool {
	changes.mu.Lock()
	defer changes.mu.Unlock()
	defer changes.refreshMemoryLocked()
	accepted := false
	finishLease := changes.acquireStoredColumnLease(tx, t, true)
	defer func() { finishLease(accepted) }()
	restore := t.guardCatalogDeclarationInput(tx.transactionState)
	defer func() {
		if !accepted {
			restore()
			// Another constructor may have published history while DDL was waiting.
			// Recompute from that actual history rather than restoring a stale guard.
			changes.publishViews(tx.publicForeignAmendments(), changes.views.Load().tables)
		}
	}()
	if !initialized {
		tx.requireNoColumnAppendRows(t)
	}
	// The validated lease rejects old raw callbacks and closes new writer input
	// before waiting for a rebuild's existing DDL owner. No schema, decision,
	// shard or amendment mutex is retained during the wait. Construction still
	// excludes rewind/decision, and every actual row entry checks that owner.
	lease := changes.leases[t]
	changes.mu.Unlock()
	if initialized {
		lease.retainDDL(t, tx.transactionState)
	} else if lease.ddlTable.Load() == nil {
		t.ddlMu.Lock()
	}
	changes.mu.Lock()
	if changes.leases[t] != lease {
		panic("append constructor lost its native declaration lease while waiting")
	}
	if !initialized && lease.ddlTable.Load() == nil {
		defer t.ddlMu.Unlock()
	}
	// Metadata/generations and retained guards can change during maintenance or
	// another same-T constructor. Re-attest before publishing the private input.
	t.guardCatalogDeclarationInput(tx.transactionState)
	if !initialized {
		tx.requireNoColumnAppendRows(t)
	}
	var initialization *catalogColumnInitialization
	if initialized {
		initialization = captureColumnInitialization(tx, t, c, fillExisting)
	}
	access := changes.db.GetWrite(tx)
	defer access.Release()
	changes.db.schemalock.Lock()
	defer changes.db.schemalock.Unlock()
	changes.validateConstraintStateEndpointsLocked(tx)
	if t.schema != changes.db || changes.db.tables.Get(t.Name) != t || !t.committedAt(changes.db.catalogView().version) || t.DroppedColumns == nil {
		panic("nullable append requires the current public generation with complete retirement history")
	}
	if _, err := uuid.Parse(t.RuntimeIdentity); err != nil {
		panic("nullable append requires native generation identity")
	}
	for name := range t.DroppedColumns {
		if strings.EqualFold(name, c.Name) {
			panic("nullable append cannot reuse a retired column name")
		}
	}
	observed := t.showColumnsSnapshot.Load()
	base := changes.views.Load().tables[t]
	if base == nil {
		base = observed
	}
	columns := base.metadata.columnNames.declarations
	for _, previous := range columns {
		if strings.EqualFold(previous.Name, c.Name) {
			return false
		}
	}
	if c.DefaultName != "" {
		if changes.db.hasDeclarationNameLocked(c.DefaultName, nil) || tx.pendingUniqueDeclarationName(c.DefaultName) {
			panic("declaration name already exists: " + c.DefaultName)
		}
		for _, view := range changes.views.Load().tables {
			for _, previous := range view.metadata.columnNames.declarations {
				if strings.EqualFold(previous.DefaultName, c.DefaultName) {
					panic("declaration name already exists: " + c.DefaultName)
				}
			}
		}
	}
	d := &catalogColumnAppend{Table: t.Name, Identity: t.RuntimeIdentity, Ordinal: len(columns), Name: c.Name, Type: storedColumnTypeOf(c), EnumValues: append([]string(nil), c.EnumValues...), Properties: maps.Clone(c.DeclarationProperties), Comment: c.Comment, Initialization: initialization}
	columns = append(append([]*column(nil), columns...), c)
	view, bytes := replaceStoredColumnMetadata(t, base, columns)
	amendment := &catalogStoredColumnAmendment{appendDeclaration: d, table: t, columns: columns, previous: changes.views.Load().tables[t], columnBytes: storedColumnBytes(c) + columnAppendStrings(c.Name, c.Typ, c.Collation, c.Comment, c.EnumValues, c.DeclarationProperties)}
	amendment.retainedBytes = uint(unsafe.Sizeof(*amendment)) + uint(cap(columns))*uint(unsafe.Sizeof((*column)(nil))) + amendment.columnBytes + bytes + d.containers() + d.Initialization.containers() + 128
	history := appendCatalogDeclarationHistory(tx.publicForeignAmendments(), catalogDeclarationAmendment{column: amendment})
	views := maps.Clone(changes.views.Load().tables)
	views[t] = view
	if _, exists := changes.baselines[t]; !exists {
		changes.baselines[t] = foreignBaseline(observed)
	}
	changes.baselinesHighWater = max(changes.baselinesHighWater, len(changes.baselines))
	changes.viewsHighWater = max(changes.viewsHighWater, len(views))
	changes.amendmentBytes += amendment.retainedBytes
	changes.historyBytes = uint(unsafe.Sizeof(history)) + uint(cap(history))*uint(unsafe.Sizeof(catalogDeclarationAmendment{}))
	tx.noteStatementMutation(t)
	tx.catalogAmendments.Store(&history)
	changes.publishViews(history, views)
	accepted = true
	return true
}

// Construction has closed mutation admission. Inspect the actual original-T
// journals, including retained/forwarded generations, not today's shard list.
func (tx *TxContext) requireNoColumnAppendRows(t *table) {
	tx.mu.Lock()
	shards := tx.shardTransactions()
	for _, action := range tx.repartitionDeletes {
		if action.table == t {
			tx.mu.Unlock()
			panic("nullable append follows row effects on its endpoint")
		}
	}
	tx.mu.Unlock()
	seen := make(map[*storageShardTransaction]bool)
	var inspect func(*storageShardTransaction)
	inspect = func(st *storageShardTransaction) {
		if st == nil || seen[st] {
			return
		}
		seen[st] = true
		if st.clusterShard == nil {
			panic("nullable append requires complete native row journal identity")
		}
		if st.clusterShard.t != t {
			return
		}
		st.mu.Lock()
		dirty := len(st.InsertRecids) != 0 || len(st.DeletedRecids) != 0 || len(st.DeleteRecids) != 0 || len(st.UndeleteRecids) != 0 || st.remoteObject != nil
		st.mu.Unlock()
		if dirty {
			panic("nullable append follows row effects on its endpoint")
		}
		inspect(st.endpoint)
		inspect(st.forwarded.Load())
	}
	for shard, st := range shards {
		if shard.t == t {
			inspect(st)
		}
	}
}

func (tx *TxContext) publicColumnAppends() []catalogColumnAppend {
	var result []catalogColumnAppend
	for _, entry := range tx.publicForeignAmendments() {
		if c := entry.column; c != nil && c.appendDeclaration != nil {
			result = append(result, *c.appendDeclaration)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Table == result[j].Table {
			return result[i].Ordinal < result[j].Ordinal
		}
		return result[i].Table < result[j].Table
	})
	return result
}

// This is an invocation input, never a public sparse-column/index installation.
func (tx *TxContext) pendingNullColumns(t *table) map[string]struct{} {
	if tx == nil || tx.transactionState == nil || tx.State != TxActive {
		return nil
	}
	if changes := tx.publicCatalogChanges(); changes != nil {
		if views := changes.views.Load(); views != nil {
			return views.nullColumns[t]
		}
	}
	return nil
}

func (tx *TxContext) pendingNullColumn(t *table, name string) bool {
	_, found := tx.pendingNullColumns(t)[name]
	return found
}

func validateCatalogColumnAppend(db *database, d catalogColumnAppend, columns []*column) *table {
	t := db.tables.Get(d.Table)
	if t == nil || t.RuntimeIdentity != d.Identity || !t.committedAt(db.catalogView().version) || t.DroppedColumns == nil || d.Ordinal != len(columns) {
		panic("nullable append lost its exact fresh predecessor")
	}
	for name := range t.DroppedColumns {
		if strings.EqualFold(name, d.Name) {
			panic("nullable append name was retired")
		}
	}
	for _, c := range columns {
		if strings.EqualFold(c.Name, d.Name) {
			panic("nullable append name already exists")
		}
	}
	return t
}

func (changes *publicCatalogChanges) validateColumnAppendsLocked() error {
	tx := &TxContext{transactionState: changes.owner}
	columns := make(map[*table][]*column)
	for _, d := range tx.publicColumnAppends() {
		t := changes.db.tables.Get(d.Table)
		if t == nil {
			return fmt.Errorf("nullable append endpoint was removed")
		}
		current := columns[t]
		if current == nil {
			current = t.columnDeclarations()
		}
		validateCatalogColumnAppend(changes.db, d, current)
		columns[t] = append(append([]*column(nil), current...), d.column())
	}
	return nil
}

func (db *database) recoverColumnAppends(declarations []catalogColumnAppend) {
	for _, d := range declarations {
		t := db.tables.Get(d.Table)
		if t == nil {
			panic("nullable append endpoint was removed")
		}
		validateCatalogColumnAppend(db, d, t.Columns)
		t.Columns = append(append([]*column(nil), t.Columns...), d.column())
		db.initializeRecoveryColumnMetadata(t)
	}
}

// New writers carry one ordered append sequence per exact table generation.
// Earlier domains and their readers never acquire absent-predecessor semantics.
func validateColumnAppendShape(appends []catalogColumnAppend) error {
	seen := make(map[string]bool)
	last := make(map[string]int)
	identities := make(map[string]string)
	for _, d := range appends {
		key := d.Table + "\x00" + scm.CaseFoldKey(d.Name)
		if d.Table == "" || d.Name == "" || d.Ordinal < 0 || seen[key] || d.Type.Type == "" || (!d.Type.Nullable && d.Initialization == nil) || d.Type.Collation == "" || !utf8.ValidString(d.Table) || !utf8.ValidString(d.Name) || !utf8.ValidString(d.Type.Type) || !utf8.ValidString(d.Type.Collation) || !utf8.ValidString(d.Comment) {
			return fmt.Errorf("invalid native nullable append")
		}
		if identity, err := uuid.Parse(d.Identity); err != nil || identity == uuid.Nil {
			return fmt.Errorf("invalid native nullable append identity")
		}
		if previous, found := last[d.Table]; found && (d.Ordinal != previous+1 || identities[d.Table] != d.Identity) {
			return fmt.Errorf("nullable append lost its ordered predecessor")
		}
		if d.Initialization != nil {
			if err := validateColumnInitialization(d); err != nil {
				return err
			}
		}
		seen[key], last[d.Table], identities[d.Table] = true, d.Ordinal, d.Identity
	}
	return nil
}
