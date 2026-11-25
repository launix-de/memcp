/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
	"slices"
	"strings"
	"unicode/utf8"
	"unsafe"
)

// Named SQL declarations are independent of physical autoindexes. Only this
// positive link proves ownership of a standalone enforcing key; old keys do not.
type logicalIndexDeclaration struct {
	Name       string     `json:"name"`
	Columns    []string   `json:"columns"`
	Included   []string   `json:"included,omitempty"`
	Descending []bool     `json:"descending,omitempty"`
	Placement  string     `json:"placement,omitempty"`
	Key        *uniqueKey `json:"key,omitempty"`
}
type catalogIndexDeclaration struct {
	Table    string                  `json:"table"`
	Identity string                  `json:"identity"`
	Index    logicalIndexDeclaration `json:"index"`
}

func (d logicalIndexDeclaration) memory() uint {
	n := uint(len(d.Name)+len(d.Placement)) + uint(cap(d.Columns)+cap(d.Included))*uint(unsafe.Sizeof("")) + uint(cap(d.Descending))
	for _, s := range d.Columns {
		n += uint(len(s))
	}
	for _, s := range d.Included {
		n += uint(len(s))
	}
	if d.Key != nil {
		n += uint(unsafe.Sizeof(uniqueKey{})) + uint(len(d.Key.Id))
	}
	return n
}
func (d *catalogIndexDeclaration) memory() uint {
	if d == nil {
		return 0
	}
	return uint(unsafe.Sizeof(*d)) + uint(len(d.Table)+len(d.Identity)) + d.Index.memory()
}
func cloneLogicalIndex(d logicalIndexDeclaration) logicalIndexDeclaration {
	d.Columns = slices.Clone(d.Columns)
	d.Included = slices.Clone(d.Included)
	d.Descending = slices.Clone(d.Descending)
	if d.Key != nil {
		k := *d.Key
		k.Cols = d.Columns
		d.Key = &k
	}
	return d
}
func sameLogicalIndex(a, b logicalIndexDeclaration) bool {
	return a.Name == b.Name && a.Placement == b.Placement && slices.Equal(a.Columns, b.Columns) && slices.Equal(a.Included, b.Included) && slices.Equal(a.Descending, b.Descending) && ((a.Key == nil && b.Key == nil) || (a.Key != nil && b.Key != nil && sameUniqueKey(*a.Key, *b.Key)))
}
func logicalIndexPosition(indexes []logicalIndexDeclaration, name string) int {
	for i, d := range indexes {
		if strings.EqualFold(d.Name, name) {
			return i
		}
	}
	return -1
}
func indexOwnsKey(indexes []logicalIndexDeclaration, name string) bool {
	for _, d := range indexes {
		if d.Key != nil && d.Key.matchesName(name) {
			return true
		}
	}
	return false
}
func validateLogicalIndex(d logicalIndexDeclaration) error {
	if d.Name == "" || !utf8.ValidString(d.Name) || strings.EqualFold(d.Name, "PRIMARY") || len(d.Columns) == 0 {
		return fmt.Errorf("invalid standalone index name or columns")
	}
	if len(d.Descending) != 0 && len(d.Descending) != len(d.Columns) {
		return fmt.Errorf("index direction count differs from columns")
	}
	if d.Placement != "" && d.Placement != "clustered" && d.Placement != "nonclustered" {
		return fmt.Errorf("unsupported logical index placement")
	}
	seen := map[string]bool{}
	for _, cols := range [][]string{d.Columns, d.Included} {
		for _, c := range cols {
			if c == "" || !utf8.ValidString(c) || seen[c] {
				return fmt.Errorf("invalid or repeated index column")
			}
			seen[c] = true
		}
	}
	if d.Key != nil && (d.Key.Id != d.Name || d.Key.ConstraintName != "" || !slices.Equal(d.Key.Cols, d.Columns)) {
		return fmt.Errorf("standalone index lost its exact enforcing key")
	}
	return nil
}
func validateIndexDirectory(indexes []logicalIndexDeclaration, keys []uniqueKey) error {
	seen := map[string]bool{}
	for _, d := range indexes {
		if err := validateLogicalIndex(d); err != nil {
			return err
		}
		name := scm.CaseFoldKey(d.Name)
		if seen[name] {
			return fmt.Errorf("duplicate logical index name")
		}
		seen[name] = true
		linked := d.Key == nil
		for _, k := range keys {
			if k.matchesName(d.Name) {
				if d.Key == nil || !sameUniqueKey(k, *d.Key) {
					return fmt.Errorf("index conflicts with a native constraint")
				}
				linked = true
			}
		}
		if !linked {
			return fmt.Errorf("standalone index enforcing key missing")
		}
	}
	return nil
}
func indexesInView(t *table, indexes map[*table][]logicalIndexDeclaration) []logicalIndexDeclaration {
	if v, ok := indexes[t]; ok {
		return v
	}
	return t.LogicalIndexes
}
func (t *table) logicalIndexesForTx(tx *TxContext) []logicalIndexDeclaration {
	if s := t.declarationMetadataForTx(tx); s != nil {
		return s.metadata.logicalIndexes
	}
	return nil
}

// Index and optional enforcing-key effects advance the EXISTING ordered native
// interpreter together. No intermediate key-only publication is possible.
func (db *database) projectIndexDeclaration(d *catalogIndexDeclaration, op string, p *catalogNativeProjection, views map[*table]*tableShowColumnsSnapshot, scope catalogForeignEndpointScope) *table {
	if d == nil {
		panic("missing native index declaration")
	}
	if err := validateLogicalIndex(d.Index); err != nil {
		panic(err)
	}
	id, err := uuid.Parse(d.Identity)
	if err != nil || id == uuid.Nil {
		panic("invalid native index endpoint identity")
	}
	t := db.tables.Get(d.Table)
	if t == nil || t.RuntimeIdentity != d.Identity || (!t.committedAt(db.catalogView().version) && !scope.owns(t) && !(scope.owner != nil && t.ScopedSchema && t.relationLifetime != nil && t.relationLifetime.owns(t))) {
		panic("index endpoint generation replaced")
	}
	indexes := indexesInView(t, p.indexes)
	keys := uniqueKeysInView(t, p.keys)
	at := logicalIndexPosition(indexes, d.Index.Name)
	switch op {
	case "add":
		if at >= 0 {
			panic("native index addition already exists")
		}
		for _, k := range keys {
			if k.matchesName(d.Index.Name) {
				panic("index name belongs to native constraint")
			}
		}
		for _, c := range append(slices.Clone(d.Index.Columns), d.Index.Included...) {
			if catalogColumnInView(t, c, views, p.propertyColumns) == nil {
				panic("unknown index column: " + c)
			}
		}
		if d.Index.Key != nil {
			key := catalogKeyDeclaration{t.Name, t.RuntimeIdentity, *d.Index.Key}
			if scope.owner != nil && t.ScopedSchema && t.relationLifetime != nil && t.relationLifetime.owns(t) {
				t.requireStoredUniqueKey(*d.Index.Key)
			} else {
				validateCatalogKey(db, key, views, scope, p.propertyColumns)
			}
			if nativeKeyNameConflict(db, t, *d.Index.Key, p.keys, p.foreign) {
				panic("index key conflicts with native declaration")
			}
			keys = append(slices.Clone(keys), *d.Index.Key)
		}
		indexes = append(slices.Clone(indexes), d.Index)
	case "drop":
		if at < 0 || !sameLogicalIndex(indexes[at], d.Index) {
			panic("native index retirement predecessor changed")
		}
		if d.Index.Key != nil {
			ki := slices.IndexFunc(keys, func(k uniqueKey) bool { return sameUniqueKey(k, *d.Index.Key) })
			if ki < 0 {
				panic("index enforcing key predecessor changed")
			}
			keys = slices.Delete(slices.Clone(keys), ki, ki+1)
			requireProjectedForeignKeys(t, keys, p.foreign)
		}
		indexes = slices.Delete(slices.Clone(indexes), at, at+1)
	default:
		panic("invalid native index operation")
	}
	p.keys[t], p.indexes[t] = keys, indexes
	return t
}
func requireProjectedForeignKeys(t *table, keys []uniqueKey, foreign map[catalogForeignIdentity][]catalogForeignDeclaration) {
	for _, group := range foreign {
		for _, d := range group {
			if d.Foreign.Tbl2 == t.Name && !slices.ContainsFunc(keys, func(k uniqueKey) bool { return slices.Equal(k.Cols, d.Foreign.Cols2) }) {
				panic("unique declaration is required by foreign key " + d.Foreign.Id)
			}
		}
	}
}
func (tx *TxContext) indexProjection(t *table) *catalogNativeProjection {
	p := newCatalogNativeProjection(t.schema)
	for table, keys := range tx.publicUniqueKeys() {
		p.keys[table] = keys
	}
	p.foreign = nativeForeignDirectory(effectiveForeignDeclarations(t.schema, tx.publicForeignAmendments()))
	if c := tx.publicCatalogChanges(); c != nil {
		if v := c.views.Load(); v != nil {
			for table, s := range v.tables {
				p.indexes[table] = s.metadata.logicalIndexes
			}
		}
	}
	return p
}

func logicalIndexRows(indexes []logicalIndexDeclaration, names *tableColumnNamesSnapshot) scm.Scmer {
	rows := make([]scm.Scmer, len(indexes))
	display := func(cols []string) scm.Scmer {
		v := make([]scm.Scmer, len(cols))
		for i, c := range cols {
			name := c
			if names != nil {
				if e, ok := names.exact[c]; ok {
					name = names.declarations[e.ordinal].displayName()
				}
			}
			v[i] = scm.NewString(name)
		}
		return scm.NewSlice(v)
	}
	for i, d := range indexes {
		dirs := make([]scm.Scmer, len(d.Descending))
		for j, b := range d.Descending {
			dirs[j] = scm.NewBool(b)
		}
		kind := "nonunique"
		if d.Key != nil {
			kind = "unique"
		}
		rows[i] = scm.NewSlice([]scm.Scmer{scm.NewString("Id"), scm.NewString(d.Name), scm.NewString("Kind"), scm.NewString(kind), scm.NewString("Columns"), display(d.Columns), scm.NewString("Included"), display(d.Included), scm.NewString("StorageColumns"), showStringSlice(d.Columns), scm.NewString("Descending"), scm.NewSlice(dirs), scm.NewString("RequestedPlacement"), scm.NewString(d.Placement)})
	}
	return scm.NewSlice(rows)
}
func replaceLogicalIndexMetadata(base *tableShowColumnsSnapshot, indexes []logicalIndexDeclaration) (*tableShowColumnsSnapshot, uint) {
	result, metadata := *base, *base.metadata
	result.metadata = &metadata
	metadata.logicalIndexes = indexes
	metadata.indexRows = logicalIndexRows(indexes, metadata.columnNames)
	constraints := slices.Clone(base.nativeConstraints().Slice())
	uniques := slices.Clone(constraints[1].Slice())
	for i, row := range uniques {
		fields := row.Slice()
		id := foreignMetadataValue(row, "Id").String()
		next := make([]scm.Scmer, 0, len(fields)+2)
		for j := 0; j < len(fields); j += 2 {
			if fields[j].String() != "standalone_index" {
				next = append(next, fields[j], fields[j+1])
			}
		}
		if indexOwnsKey(indexes, id) {
			next = append(next, scm.NewString("standalone_index"), scm.NewBool(true))
		}
		uniques[i] = scm.NewSlice(next)
	}
	constraints[1] = scm.NewSlice(uniques)
	metadata.constraints = scm.NewSlice(constraints)
	metadata.storageConstraints = metadata.constraints
	decl := slices.Clone(base.metadata.declarations.Slice())
	decl[3] = metadata.constraints
	metadata.declarations = scm.NewSlice(decl)
	bytes := uint(unsafe.Sizeof(result)) + uint(unsafe.Sizeof(metadata)) + uint(cap(constraints)+cap(uniques)+cap(decl)+cap(metadata.indexRows.Slice()))*uint(unsafe.Sizeof(scm.Scmer{}))
	for _, row := range metadata.indexRows.Slice() {
		bytes += uint(cap(row.Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
		for j := 1; j < len(row.Slice()); j += 2 {
			if v := row.Slice()[j]; v.IsSlice() {
				bytes += uint(cap(v.Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
			}
		}
	}
	for _, row := range uniques {
		bytes += uint(cap(row.Slice())) * uint(unsafe.Sizeof(scm.Scmer{}))
	}
	return &result, bytes
}
func mutateLogicalIndex(t *table, index logicalIndexDeclaration, op string, tx *TxContext) bool {
	if tx == nil || tx.State != TxActive || tx.decisionStarted {
		panic("index declaration requires active original native transaction")
	}
	tx.requireDeclarationEndpoint(t, maintenanceAlter)
	if op == "drop" {
		current := t.logicalIndexesForTx(tx)
		at := logicalIndexPosition(current, index.Name)
		if at < 0 {
			for _, key := range uniqueKeysInView(t, tx.publicUniqueKeys()) {
				if key.matchesName(index.Name) {
					panic("DROP INDEX cannot remove constraint or unclassified legacy key")
				}
			}
			return false
		}
		index = current[at]
	} else {
		if err := validateLogicalIndex(index); err != nil {
			panic(err)
		}
		index = cloneLogicalIndex(index)
	}
	if t.ScopedSchema {
		return tx.mutateScopedLogicalIndex(t, index, op)
	}
	if tx.ownsPendingTable(t) && !tx.catalogDeclarationDataEntered(t) {
		return tx.amendCatalogDeclaration(t, func(current *table) bool {
			if op == "add" && logicalIndexPosition(current.LogicalIndexes, index.Name) >= 0 {
				return false
			}
			p := newCatalogNativeProjection(current.schema)
			d := catalogIndexDeclaration{current.Name, current.RuntimeIdentity, index}
			current.schema.projectIndexDeclaration(&d, op, p, nil, catalogForeignEndpointScope{owner: tx.transactionState})
			current.Unique, current.LogicalIndexes = p.keys[current], p.indexes[current]
			return true
		})
	}
	return tx.stagePublicKey(t, op, uniqueKey{Id: index.Name}, &index)
}
func (tx *TxContext) mutateScopedLogicalIndex(t *table, index logicalIndexDeclaration, op string) bool {
	return tx.mutateScopedIndexOrKey(t, &index, uniqueKey{}, op)
}

func (tx *TxContext) mutateScopedIndexOrKey(t *table, index *logicalIndexDeclaration, key uniqueKey, op string) bool {
	scope := tx.requireScopedColumnOwner(t)
	finish := tx.beginCatalogConstruction(catalogValidationConstruction, nil, t)
	defer finish()
	leave := scope.beginUse()
	defer leave()
	tx.requireDeclarationEndpoint(t, maintenanceAlter)
	lease := tx.admitScopedRelationLayout(t, scope)
	before := tx.scopedRelationLayout(t)
	if index != nil && op == "add" && logicalIndexPosition(before.view.metadata.logicalIndexes, index.Name) >= 0 {
		return false
	}
	if op == "add" {
		if index != nil {
			if index.Key != nil {
				validateExistingUniqueRows(tx, t, *index.Key)
			}
		} else {
			validateExistingUniqueRows(tx, t, key)
		}
	}
	access := t.schema.GetWrite(tx)
	defer access.Release()
	t.schema.schemalock.Lock()
	defer t.schema.schemalock.Unlock()
	tx.requireScopedColumnOwner(t)
	p := tx.indexProjection(t)
	p.keys[t] = before.keys
	p.indexes[t] = before.view.metadata.logicalIndexes
	p.propertyColumns[t] = before.view.metadata.columnNames.declarations
	if index != nil {
		d := catalogIndexDeclaration{t.Name, t.RuntimeIdentity, *index}
		t.schema.projectIndexDeclaration(&d, op, p, nil, catalogForeignEndpointScope{owner: tx.transactionState})
	} else {
		if op != "add" {
			panic("unsupported scoped key action")
		}
		if logicalIndexPosition(p.indexes[t], key.Id) >= 0 {
			panic("key name belongs to standalone index")
		}
		for _, existing := range p.keys[t] {
			if existing.matchesName(key.Id) || key.ConstraintName != "" && existing.matchesName(key.ConstraintName) {
				return false
			}
		}
		t.requireStoredUniqueKey(key)
		for _, c := range key.Cols {
			if catalogColumnInView(t, c, nil, p.propertyColumns) == nil {
				panic("unknown key column: " + c)
			}
		}
		if nativeKeyNameConflict(t.schema, t, key, p.keys, p.foreign) {
			panic("key name conflicts with native declaration")
		}
		validateCatalogKey(t.schema, catalogKeyDeclaration{t.Name, t.RuntimeIdentity, key}, nil, catalogForeignEndpointScope{owner: tx.transactionState}, p.propertyColumns)
		key.Cols = slices.Clone(key.Cols)
		p.keys[t] = append(slices.Clone(p.keys[t]), key)
	}
	view, _ := replaceUniqueMetadata(before.view, p.keys[t], false)
	view, _ = replaceLogicalIndexMetadata(view, p.indexes[t])
	after := *before
	after.view = view
	after.keys = p.keys[t]
	after.lease = lease
	t.writeAdmission.mu.Lock()
	defer t.writeAdmission.mu.Unlock()
	if t.writeAdmission.declaration != tx.transactionState || t.writeAdmission.statement != nil || t.writeAdmission.writers[tx.transactionState].unframedCallbacks {
		panic("scoped index input is in use by a native producer")
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if scope.dataInputs[tx.transactionState] != 0 {
		panic("scoped index input is in use by original caller")
	}
	tx.noteStatementMutation(t)
	scope.schemaDecisions++
	tx.mu.Lock()
	tx.relationSchemaChanges = append(tx.relationSchemaChanges, &relationSchemaChange{scope: scope, layout: &relationSchemaLayoutChange{before, &after}})
	tx.storeScopedRelationLayout(&after)
	tx.mu.Unlock()
	return true
}
func sameLogicalIndexes(a, b []logicalIndexDeclaration) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !sameLogicalIndex(a[i], b[i]) {
			return false
		}
	}
	return true
}
func sameUniqueKeys(a, b []uniqueKey) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !sameUniqueKey(a[i], b[i]) {
			return false
		}
	}
	return true
}

func initLogicalIndexes(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "create_index", Fn: func(a ...scm.Scmer) scm.Scmer {
		t := TableFromScmer(a[0])
		d := logicalIndexDeclaration{Name: a[1].String(), Columns: scmerSliceToStrings(mustScmerSlice(a[2], "index columns"))}
		unique := scm.ToBool(a[3])
		tx := scmerToTxContext(a[4])
		nulls := false
		if len(a) > 5 && !a[5].IsNil() {
			options := a[5].Slice()
			for i := 0; i < len(options); i += 2 {
				if i+1 >= len(options) {
					panic("invalid index options")
				}
				switch options[i].String() {
				case "included":
					d.Included = scmerSliceToStrings(mustScmerSlice(options[i+1], "included columns"))
				case "descending":
					for _, v := range mustScmerSlice(options[i+1], "directions") {
						d.Descending = append(d.Descending, scm.ToBool(v))
					}
				case "placement":
					d.Placement = options[i+1].String()
				case "nulls_not_distinct":
					nulls = scm.ToBool(options[i+1])
				default:
					panic("unknown logical index option")
				}
			}
		}
		if unique {
			d.Key = &uniqueKey{Id: d.Name, Cols: d.Columns, NullsNotDistinct: nulls}
		}
		return scm.NewBool(mutateLogicalIndex(t, d, "add", tx))
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Params: []*scm.TypeDescriptor{{Kind: "table", Label: "table"}, {Kind: "string", Label: "name"}, {Kind: "list", Label: "columns"}, {Kind: "bool", Label: "unique"}, {Kind: "any", Label: "tx"}, {Kind: "list", Label: "options", Optional: true}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "drop_index", Fn: func(a ...scm.Scmer) scm.Scmer {
		t := TableFromScmer(a[0])
		name := a[1].String()
		ok := mutateLogicalIndex(t, logicalIndexDeclaration{Name: name}, "drop", scmerToTxContext(a[2]))
		if !ok && (len(a) < 4 || !scm.ToBool(a[3])) {
			panic("index does not exist: " + name)
		}
		return scm.NewBool(ok)
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Params: []*scm.TypeDescriptor{{Kind: "table", Label: "table"}, {Kind: "string", Label: "name"}, {Kind: "any", Label: "tx"}, {Kind: "bool", Label: "if_exists", Optional: true}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
}

func (record *catalogDecisionRecord) hasLogicalIndexes() bool {
	for _, a := range record.NativeDeclarations {
		if a.Index != nil {
			return true
		}
	}
	for _, t := range record.Tables {
		if len(t.LogicalIndexes) != 0 {
			return true
		}
	}
	return false
}
