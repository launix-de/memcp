/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "slices"
import "strings"
import "unicode/utf8"
import "unsafe"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// Keys and their FK consumers share one ordered inert declaration program.
// A recipe is not a second catalog, row payload or independently accepted Tx.
type catalogNativeDeclaration struct {
	Index       *catalogIndexDeclaration         `json:"index,omitempty"`
	Operation   string                           `json:"operation"`
	Key         *catalogKeyDeclaration           `json:"key,omitempty"`
	Foreign     *catalogForeignDeclaration       `json:"foreign,omitempty"`
	Property    *catalogColumnPropertyChange     `json:"column_property,omitempty"`
	Rename      *catalogColumnRename             `json:"column_rename,omitempty"`
	TableRename *catalogTableRename              `json:"table_rename,omitempty"`
	Declaration *catalogColumnDeclaration        `json:"column_declaration,omitempty"`
	Drop        *catalogColumnDrop               `json:"column_drop,omitempty"`
	Default     *catalogColumnDefaultDeclaration `json:"column_default,omitempty"`
}

func (tx *TxContext) publicNativeDeclarations(history []catalogDeclarationAmendment) []catalogNativeDeclaration {
	var result []catalogNativeDeclaration
	// Liveness belongs to this supplied immutable program. Constructors also
	// project their prospective suffix before publication; consulting the live
	// view here would retain a rename whose endpoint that suffix retires.
	dropped := make(map[*table]bool)
	for _, entry := range history {
		if entry.tableDrop != nil {
			dropped[entry.tableDrop.table] = true
		}
	}
	properties := tx.hasColumnPropertyClaims(history)
	orderedColumns := tx.validatedDeclarationTables(history)
	retiring := tx.columnRetirementTables(history)
	// Pending state transitions belong to this same amendment history. Freeze
	// each addition's accepted validation fact once, including its matching DROP;
	// recovery never publishes a transient pending state or a second projection.
	trusted := make(map[*catalogForeignAmendment]bool)
	enabled := make(map[*catalogForeignAmendment]bool)
	for _, entry := range history {
		if add := entry.foreign; add != nil {
			trusted[add] = !add.declaration.Foreign.Untrusted
			enabled[add] = !add.declaration.Foreign.Disabled
		}
		if state := entry.constraintState; state != nil {
			for _, current := range state.current {
				if current.addition != nil {
					trusted[current.addition] = current.trusted
					enabled[current.addition] = current.enabled
				}
			}
		}
	}
	for _, entry := range history {
		if a := entry.column; a != nil && a.declaration != nil && orderedColumns[a.table] && !dropped[a.table] {
			result = append(result, catalogNativeDeclaration{Operation: "column_declaration", Declaration: a.declaration})
		}
		if a := entry.column; a != nil && !dropped[a.table] {
			if a.drop != nil {
				result = append(result, catalogNativeDeclaration{Operation: "column_drop", Drop: a.drop})
			} else if retiring[a.table] && a.rename == nil && !a.nameMetadataOnly && a.declaration == nil && a.propertyRemoval == nil && a.propertyChange == nil && a.appendDeclaration == nil {
				d := a.defaultDeclaration
				result = append(result, catalogNativeDeclaration{Operation: "column_default", Default: &d})
			}
		}
		if a := entry.tableRename; a != nil && !dropped[a.table] {
			result = append(result, catalogNativeDeclaration{Operation: "table_rename", TableRename: a.declaration})
		}
		if a := entry.column; a != nil && a.rename != nil && !dropped[a.table] {
			result = append(result, catalogNativeDeclaration{Operation: "column_rename", Rename: a.rename})
		}
		if properties && entry.column != nil && entry.column.propertyChange != nil && !dropped[entry.column.table] {
			operation := "column_property_claim"
			if entry.column.propertyRemoval != nil {
				operation = "column_property_drop"
			}
			result = append(result, catalogNativeDeclaration{Operation: operation, Property: entry.column.propertyChange})
		}
		if key := entry.publicKey; key != nil {
			if key.index != nil {
				result = append(result, catalogNativeDeclaration{Operation: "index_" + key.operation, Index: key.index})
				continue
			}
			d := key.declaration
			result = append(result, catalogNativeDeclaration{Operation: "key_" + key.operation, Key: &d})
		}
		if add := entry.foreign; add != nil {
			d := add.declaration
			d.Foreign.Disabled = !enabled[add]
			d.Foreign.Untrusted = !trusted[add] || d.Foreign.Disabled
			result = append(result, catalogNativeDeclaration{Operation: "foreign_add", Foreign: &d})
		}
		if drop := entry.foreignDrop; drop != nil {
			d := drop.declaration
			if drop.addition != nil {
				d.Foreign.Disabled = !enabled[drop.addition]
				d.Foreign.Untrusted = !trusted[drop.addition] || d.Foreign.Disabled
			}
			result = append(result, catalogNativeDeclaration{Operation: "foreign_drop", Foreign: &d})
		}
	}
	return result
}

func nativeForeignDirectory(declarations []catalogForeignDeclaration) map[catalogForeignIdentity][]catalogForeignDeclaration {
	result := make(map[catalogForeignIdentity][]catalogForeignDeclaration, len(declarations))
	for _, d := range declarations {
		identity := foreignIdentity(d)
		result[identity] = append(result[identity], d)
	}
	return result
}

func effectiveForeignDeclarations(db *database, history []catalogDeclarationAmendment) []catalogForeignDeclaration {
	current := currentForeignAdditions(history)
	var result []catalogForeignDeclaration
	for _, t := range db.tables.GetAll() {
		for _, fk := range t.Foreign {
			if fk.Tbl1 != t.Name {
				continue
			}
			if _, changed := current[db.storedForeignIdentity(fk)]; changed {
				continue
			}
			identity := ""
			if parent := db.tables.Get(fk.Tbl2); parent != nil {
				identity = parent.RuntimeIdentity
			}
			result = append(result, catalogForeignDeclaration{t.RuntimeIdentity, identity, fk})
		}
	}
	for _, add := range current {
		if add != nil {
			result = append(result, add.declaration)
		}
	}
	return result
}

type catalogNativeProjection struct {
	indexes         map[*table][]logicalIndexDeclaration
	keys            map[*table][]uniqueKey
	foreign         map[catalogForeignIdentity][]catalogForeignDeclaration
	changedForeign  map[catalogForeignIdentity]bool
	foreignAdds     []catalogForeignDeclaration
	foreignDrops    []catalogForeignDeclaration
	propertyColumns map[*table][]*column
	tableNames      map[*table]string
}

func newCatalogNativeProjection(db *database) *catalogNativeProjection {
	p := &catalogNativeProjection{indexes: make(map[*table][]logicalIndexDeclaration), tableNames: make(map[*table]string), propertyColumns: make(map[*table][]*column), keys: make(map[*table][]uniqueKey), foreign: make(map[catalogForeignIdentity][]catalogForeignDeclaration), changedForeign: make(map[catalogForeignIdentity]bool)}
	p.foreign = nativeForeignDirectory(effectiveForeignDeclarations(db, nil))
	return p
}

// The V13 reader keeps its established exact-name endpoint retirement cut.
// Current writers additionally require an unambiguous complete predecessor.
// This only selects legacy replay semantics; it is not a persisted authority.
type catalogNativeReplayDomain uint8

const (
	catalogNativeCurrent catalogNativeReplayDomain = iota
	catalogNativeLegacyV13
)

// This is a private metadata interpreter at prepare/recovery. Each action sees
// the exact key and FK declarations left by its predecessor; no public header
// is edited in order to validate a later action.
func (db *database) projectNativeDeclarations(actions []catalogNativeDeclaration, views map[*table]*tableShowColumnsSnapshot, domain catalogNativeReplayDomain, scope catalogForeignEndpointScope) *catalogNativeProjection {
	if domain != catalogNativeCurrent && domain != catalogNativeLegacyV13 {
		panic("invalid native declaration replay domain")
	}
	p := newCatalogNativeProjection(db)
	// Final original-T views already omit a retired column. Native actions
	// must instead validate their own predecessor: seed these endpoints from
	// the published header and advance the same ordered column projection.
	// Other endpoints retain their established view (including prior appends).
	var retirementInputs map[*table][]*column
	for _, action := range actions {
		if d := action.Drop; d != nil {
			t := db.tables.Get(d.Table)
			if t == nil || t.RuntimeIdentity != d.Identity {
				panic("column retirement predecessor generation was replaced")
			}
			p.propertyColumns[t] = t.columnDeclarations()
			if retirementInputs == nil {
				retirementInputs = make(map[*table][]*column)
			}
			retirementInputs[t] = p.propertyColumns[t]
		}
	}
	added := make(map[catalogForeignIdentity]int)
	for index, action := range actions {
		if action.Index != nil && action.Operation != "index_add" && action.Operation != "index_drop" {
			panic("index payload on a different native action")
		}
		if action.TableRename != nil && action.Operation != "table_rename" {
			panic("table rename payload on a different native action")
		}
		if action.Rename != nil && action.Operation != "column_rename" {
			panic("column rename payload on a different native action")
		}
		if action.Declaration != nil && action.Operation != "column_declaration" {
			panic("stored declaration payload on a different native action")
		}
		if action.Drop != nil && action.Operation != "column_drop" || action.Default != nil && action.Operation != "column_default" {
			panic("column retirement payload on another native action")
		}
		switch action.Operation {
		case "index_add", "index_drop":
			if domain != catalogNativeCurrent || action.Index == nil || action.Key != nil || action.Foreign != nil || action.Property != nil || action.Rename != nil || action.TableRename != nil || action.Declaration != nil || action.Drop != nil || action.Default != nil {
				panic("invalid native index action")
			}
			db.projectIndexDeclaration(action.Index, strings.TrimPrefix(action.Operation, "index_"), p, views, scope)
		case "column_drop":
			if action.Drop == nil || action.Default != nil || action.Declaration != nil || action.Rename != nil || action.TableRename != nil || action.Property != nil || action.Key != nil || action.Foreign != nil {
				panic("invalid native column retirement action")
			}
			t := db.tables.Get(action.Drop.Table)
			_, columns := db.projectColumnDrop(action.Drop, p.propertyColumns[t], p)
			p.propertyColumns[t] = columns
			if _, retiring := retirementInputs[t]; retiring {
				retirementInputs[t] = columns
			}
		case "column_default":
			if action.Default == nil || action.Drop != nil || action.Declaration != nil || action.Rename != nil || action.TableRename != nil || action.Property != nil || action.Key != nil || action.Foreign != nil {
				panic("invalid native ordered default action")
			}
			d := action.Default
			t := db.tables.Get(d.Table)
			if t == nil || t.RuntimeIdentity != d.Identity || !t.committedAt(db.catalogView().version) {
				panic("ordered default endpoint generation was replaced")
			}
			columns := p.propertyColumns[t]
			if columns == nil {
				columns = t.columnDeclarations()
			}
			p.propertyColumns[t] = projectOrderedColumnDefault(d, t, columns)
			if _, retiring := retirementInputs[t]; retiring {
				retirementInputs[t] = p.propertyColumns[t]
			}
		case "column_declaration":
			if domain != catalogNativeCurrent || action.Declaration == nil || action.Rename != nil || action.TableRename != nil || action.Property != nil || action.Key != nil || action.Foreign != nil {
				panic("invalid native stored declaration action")
			}
			t := db.tables.Get(action.Declaration.Table)
			_, columns := db.projectStoredDeclaration(action.Declaration, p.propertyColumns[t], p)
			p.propertyColumns[t] = columns
			if _, retiring := retirementInputs[t]; retiring {
				retirementInputs[t] = columns
			}
		case "table_rename":
			if domain != catalogNativeCurrent || action.TableRename == nil || action.Rename != nil || action.Property != nil || action.Key != nil || action.Foreign != nil {
				panic("invalid native table rename action")
			}
			db.projectTableRename(action.TableRename, p, scope)
		case "column_rename":
			if domain != catalogNativeCurrent || action.Rename == nil || action.TableRename != nil || action.Property != nil || action.Key != nil || action.Foreign != nil {
				panic("invalid native column rename action")
			}
			t := db.tables.Get(action.Rename.Table)
			_, columns := db.projectColumnRename(action.Rename, p.propertyColumns[t])
			p.propertyColumns[t] = columns
			if _, retiring := retirementInputs[t]; retiring {
				retirementInputs[t] = columns
			}
		case "foreign_add", "foreign_drop":
			if action.Foreign == nil || action.Key != nil || action.Property != nil {
				panic("invalid native foreign action")
			}
			d := *action.Foreign
			validateCatalogForeign(db, d, p.keys, views, scope, retirementInputs)
			name := foreignIdentity(d)
			current, exists := p.foreign[name]
			if action.Operation == "foreign_drop" {
				if !exists || !slices.ContainsFunc(current, func(previous catalogForeignDeclaration) bool { return sameCatalogForeign(previous, d) }) {
					panic("native foreign retirement predecessor changed")
				}
				if !p.changedForeign[name] {
					p.foreignDrops = append(p.foreignDrops, d)
				}
				if domain == catalogNativeCurrent && slices.ContainsFunc(current, func(previous catalogForeignDeclaration) bool { return !sameCatalogForeign(previous, d) }) {
					panic("foreign removal has ambiguous native predecessors on its endpoint pair")
				}
				remaining := slices.DeleteFunc(append([]catalogForeignDeclaration(nil), current...), func(previous catalogForeignDeclaration) bool {
					if domain == catalogNativeLegacyV13 {
						return previous.Foreign.Id == d.Foreign.Id && previous.Foreign.Tbl1 == d.Foreign.Tbl1 && previous.Foreign.Tbl2 == d.Foreign.Tbl2
					}
					return sameCatalogForeign(previous, d)
				})
				if len(remaining) == 0 {
					delete(p.foreign, name)
				} else {
					p.foreign[name] = remaining
				}
				delete(added, name)
			} else {
				if exists || nativeDeclarationNameConflict(db, d.Foreign.Id, nil, p.keys, p.foreign, true) {
					panic("native foreign addition conflicts with another declaration")
				}
				p.foreign[name] = []catalogForeignDeclaration{d}
				added[name] = index
			}
			p.changedForeign[name] = true
		case "column_property_claim", "column_property_drop":
			if domain != catalogNativeCurrent || action.Property == nil || action.Key != nil || action.Foreign != nil {
				panic("invalid native column property action")
			}
			t := db.tables.Get(action.Property.Table)
			_, columns := db.projectColumnPropertyChange(action.Property, action.Operation, p.propertyColumns[t])
			p.propertyColumns[t] = columns
			if _, retiring := retirementInputs[t]; retiring {
				retirementInputs[t] = columns
			}
		case "key_add", "key_drop":
			if action.Key == nil || action.Foreign != nil || action.Property != nil {
				panic("invalid native unique action")
			}
			d := *action.Key
			t := validateCatalogKey(db, d, views, scope, retirementInputs)
			keys := uniqueKeysInView(t, p.keys)
			if action.Operation == "key_drop" && indexOwnsKey(indexesInView(t, p.indexes), d.Key.Id) {
				panic("standalone index requires native index retirement")
			}
			if action.Operation == "key_add" && logicalIndexPosition(indexesInView(t, p.indexes), d.Key.Id) >= 0 {
				panic("key name belongs to a standalone index")
			}
			if action.Operation == "key_add" {
				if nativeKeyNameConflict(db, t, d.Key, p.keys, p.foreign) {
					panic("native unique addition conflicts with another declaration")
				}
				keys = append(append([]uniqueKey(nil), keys...), d.Key)
			} else {
				found := -1
				for i, key := range keys {
					if sameUniqueKey(key, d.Key) {
						found = i
						break
					}
				}
				if found < 0 {
					panic("native unique retirement predecessor changed")
				}
				keys = append([]uniqueKey(nil), keys...)
				keys = slices.Delete(keys, found, found+1)
				for _, declarations := range p.foreign {
					for _, fk := range declarations {
						if fk.Foreign.Tbl2 != t.Name {
							continue
						}
						supported := false
						for _, key := range keys {
							if slices.Equal(key.Cols, fk.Foreign.Cols2) {
								supported = true
								break
							}
						}
						if !supported {
							panic("unique declaration is required by foreign key " + fk.Foreign.Id + "; drop the foreign key constraint first")
						}
					}
				}
			}
			p.keys[t] = keys
		default:
			panic("invalid native declaration action")
		}
	}
	// Preserve actual final-add order without any suffix/history rescan.
	for index, action := range actions {
		if action.Operation != "foreign_add" {
			continue
		}
		name := foreignIdentity(*action.Foreign)
		if last, ok := added[name]; ok && last == index {
			p.foreignAdds = append(p.foreignAdds, *action.Foreign)
			delete(added, name)
		}
	}
	if len(p.tableNames) > 0 {
		db.validateNameNamespace(p.tableNames, scope)
	}
	return p
}

func nativeKeyNameConflict(db *database, t *table, key uniqueKey, keys map[*table][]uniqueKey, foreign map[catalogForeignIdentity][]catalogForeignDeclaration) bool {
	if nativeDeclarationNameConflict(db, key.Id, t, keys, foreign, false) {
		return true
	}
	return key.ConstraintName != "" && nativeDeclarationNameConflict(db, key.ConstraintName, t, keys, foreign, true)
}

func nativeDeclarationNameConflict(db *database, name string, target *table, keys map[*table][]uniqueKey, foreign map[catalogForeignIdentity][]catalogForeignDeclaration, global bool) bool {
	for _, t := range db.tables.GetAll() {
		if (global || t == target) && t.hasColumnCheckDeclarationName(name, nil) {
			return true
		}
		for _, key := range uniqueKeysInView(t, keys) {
			if t == target && key.matchesName(name) || global && key.ConstraintName != "" && strings.EqualFold(key.ConstraintName, name) {
				return true
			}
		}
	}
	for _, declarations := range foreign {
		for _, fk := range declarations {
			if (global || target != nil && fk.Foreign.Tbl1 == target.Name) && strings.EqualFold(fk.Foreign.Id, name) {
				return true
			}
		}
	}
	return false
}

func (db *database) recoverNativeDeclarations(actions []catalogNativeDeclaration, domain catalogNativeReplayDomain, scope catalogForeignEndpointScope) {
	if domain != catalogNativeCurrent && domain != catalogNativeLegacyV13 {
		panic("invalid native declaration replay domain")
	}
	if len(actions) == 0 {
		return
	}
	db.projectNativeDeclarations(actions, nil, domain, scope)
	for _, action := range actions {
		switch action.Operation {
		case "index_add":
			d := action.Index
			t := db.tables.Get(d.Table)
			t.LogicalIndexes = append(slices.Clone(t.LogicalIndexes), d.Index)
			if d.Index.Key != nil {
				t.Unique = append(slices.Clone(t.Unique), *d.Index.Key)
			}
		case "index_drop":
			d := action.Index
			t := db.tables.Get(d.Table)
			t.LogicalIndexes = slices.DeleteFunc(t.LogicalIndexes, func(i logicalIndexDeclaration) bool { return sameLogicalIndex(i, d.Index) })
			if d.Index.Key != nil {
				t.Unique = slices.DeleteFunc(t.Unique, func(k uniqueKey) bool { return sameUniqueKey(k, *d.Index.Key) })
			}
		case "column_drop":
			d := action.Drop
			t := db.tables.Get(d.Table)
			t.Columns = append(append([]*column(nil), t.Columns[:d.Ordinal]...), t.Columns[d.Ordinal+1:]...)
			if t.DroppedColumns == nil {
				t.DroppedColumns = make(map[string]bool)
			}
			t.DroppedColumns[d.Storage] = true
			db.initializeRecoveryColumnMetadata(t)
		case "column_default":
			t := db.tables.Get(action.Default.Table)
			t.Columns = projectOrderedColumnDefault(action.Default, t, t.Columns)
			db.initializeRecoveryColumnMetadata(t)
		case "column_declaration":
			// Whole-decision projection above has already checked ordered native
			// dependencies; replay only installs its attested immutable header.
			t := db.tables.Get(action.Declaration.Table)
			for index, previous := range t.Columns {
				if previous.Name == action.Declaration.Column {
					columns := append([]*column(nil), t.Columns...)
					if action.Declaration.After.compatibleWith(action.Declaration.Before) {
						columns[index] = action.Declaration.After.replace(previous)
					} else {
						columns[index] = action.Declaration.After.replaceValidated(previous)
					}
					t.Columns = columns
					db.initializeRecoveryColumnMetadata(t)
					break
				}
			}
		case "table_rename":
			t := db.tables.Get(action.TableRename.Storage)
			name := action.TableRename.After
			t.publishedName.Store(&name)
		case "column_rename":
			t, columns := db.projectColumnRename(action.Rename, nil)
			t.Columns = columns
			db.initializeRecoveryColumnMetadata(t)
		case "foreign_add":
			db.recoverPublicForeignDeclarations([]catalogForeignDeclaration{*action.Foreign}, scope)
		case "foreign_drop":
			db.recoverPublicForeignDrops([]catalogForeignDeclaration{*action.Foreign}, scope)
		case "column_property_claim", "column_property_drop":
			t, columns := db.projectColumnPropertyChange(action.Property, action.Operation, nil)
			t.Columns = columns
			db.initializeRecoveryColumnMetadata(t)
		case "key_add":
			t := validateCatalogKey(db, *action.Key, nil, scope, nil)
			t.Unique = append(append([]uniqueKey(nil), t.Unique...), action.Key.Key)
		case "key_drop":
			t := validateCatalogKey(db, *action.Key, nil, scope, nil)
			t.Unique = slices.DeleteFunc(t.Unique, func(key uniqueKey) bool { return sameUniqueKey(key, action.Key.Key) })
		}
	}
}

func validateNativeDeclarationShape(actions []catalogNativeDeclaration) error {
	if len(actions) == 0 {
		return fmt.Errorf("native declaration history is empty")
	}
	indexSeen := make(map[string]bool)
	indexCurrent := make(map[string]*catalogIndexDeclaration)
	keySeen := make(map[string]bool)
	keyCurrent := make(map[string]catalogKeyDeclaration)
	foreignSeen := make(map[catalogForeignIdentity]bool)
	foreignCurrent := make(map[catalogForeignIdentity]catalogForeignDeclaration)
	validName := func(name string) bool { return name != "" && utf8.ValidString(name) }
	validIdentity := func(id string) bool { value, err := uuid.Parse(id); return err == nil && value != uuid.Nil }
	validColumns := func(columns []string) bool {
		if len(columns) == 0 {
			return false
		}
		seen := make(map[string]bool, len(columns))
		for _, name := range columns {
			if !validName(name) || seen[name] {
				return false
			}
			seen[name] = true
		}
		return true
	}
	type propertyState struct {
		declaration *catalogColumnPropertyChange
		present     bool
		value       scm.Scmer
	}
	propertyCurrent := make(map[[3]string]propertyState)
	renameCurrent := make(map[[2]string]*catalogColumnRename)
	tableCurrent := make(map[string]*catalogTableRename)
	retiredColumns := make(map[[2]string]bool)
	orderedTypes := false
	for _, action := range actions {
		orderedTypes = orderedTypes || action.Declaration != nil
	}
	var columnTypes map[[2]string]storedColumnType
	if orderedTypes {
		columnTypes = make(map[[2]string]storedColumnType)
	}
	attestColumn := func(identity, column string, expected storedColumnType) bool {
		if !orderedTypes {
			return true // retain the historical property/name shape domains
		}
		key := [2]string{identity, column}
		if before, found := columnTypes[key]; found && !before.equal(expected) {
			return false
		}
		columnTypes[key] = expected
		return true
	}
	for _, action := range actions {
		if action.Index != nil && action.Operation != "index_add" && action.Operation != "index_drop" {
			return fmt.Errorf("index payload on another native action")
		}
		var identity, physical string
		switch {
		case action.Drop != nil:
			identity, physical = action.Drop.Identity, action.Drop.Storage
		case action.Default != nil:
			identity, physical = action.Default.Identity, action.Default.Column
		case action.Declaration != nil:
			identity, physical = action.Declaration.Identity, action.Declaration.Column
		case action.Rename != nil:
			identity, physical = action.Rename.Identity, action.Rename.Storage
		case action.Property != nil:
			identity, physical = action.Property.Identity, action.Property.Column
		}
		if identity != "" && retiredColumns[[2]string{identity, physical}] {
			return fmt.Errorf("native action refers to a retired stored column")
		}
		if action.Drop != nil {
			retiredColumns[[2]string{identity, physical}] = true
		}

		if action.Drop != nil && action.Operation != "column_drop" || action.Default != nil && action.Operation != "column_default" {
			return fmt.Errorf("column retirement payload on another native action")
		}
		if action.TableRename != nil && action.Operation != "table_rename" {
			return fmt.Errorf("table rename payload on a different native action")
		}
		if action.Rename != nil && action.Operation != "column_rename" {
			return fmt.Errorf("column rename payload on a different native action")
		}
		if action.Declaration != nil && action.Operation != "column_declaration" {
			return fmt.Errorf("stored declaration payload on a different native action")
		}
		switch action.Operation {
		case "index_add", "index_drop":
			d := action.Index
			if d == nil || action.Key != nil || action.Foreign != nil || action.Property != nil || action.Rename != nil || action.TableRename != nil || action.Declaration != nil || action.Drop != nil || action.Default != nil || !validName(d.Table) || !validIdentity(d.Identity) {
				return fmt.Errorf("invalid native index action")
			}
			if err := validateLogicalIndex(d.Index); err != nil {
				return err
			}
			identity := d.Identity + "/" + scm.CaseFoldKey(d.Index.Name)
			prior, present := indexCurrent[identity]
			if action.Operation == "index_add" {
				if indexSeen[identity] && present {
					return fmt.Errorf("repeated native index addition")
				}
				indexCurrent[identity] = d
			} else {
				if indexSeen[identity] && (!present || prior.Table != d.Table || !sameLogicalIndex(prior.Index, d.Index)) {
					return fmt.Errorf("native index retirement lost its exact predecessor")
				}
				delete(indexCurrent, identity)
			}
			indexSeen[identity] = true
		case "column_drop":
			if action.Drop == nil || action.Default != nil || action.Declaration != nil || action.Rename != nil || action.TableRename != nil || action.Property != nil || action.Key != nil || action.Foreign != nil {
				return fmt.Errorf("invalid native column retirement action")
			}
			if err := validateColumnDropShape(action.Drop); err != nil {
				return err
			}
			if !attestColumn(action.Drop.Identity, action.Drop.Storage, action.Drop.Expected) {
				return fmt.Errorf("column retirement lost its ordered declaration predecessor")
			}
		case "column_default":
			if action.Default == nil || action.Drop != nil || action.Declaration != nil || action.Rename != nil || action.TableRename != nil || action.Property != nil || action.Key != nil || action.Foreign != nil {
				return fmt.Errorf("invalid native ordered default action")
			}
			d := action.Default
			if !validName(d.Table) || !validIdentity(d.Identity) || !validName(d.Column) || !utf8.ValidString(d.Name) || !utf8.ValidString(d.Expression) {
				return fmt.Errorf("invalid native ordered default declaration")
			}
			if d.Deferred && (!d.Value.IsNil() || d.Expression == "") {
				return fmt.Errorf("invalid native deferred ordered default")
			}
			if err := validateColumnInitialScalar(d.Value); err != nil {
				return err
			}
		case "column_declaration":
			if action.Declaration == nil || action.Rename != nil || action.TableRename != nil || action.Property != nil || action.Key != nil || action.Foreign != nil {
				return fmt.Errorf("invalid native stored declaration action")
			}
			d := action.Declaration
			if err := validateStoredDeclarationShape(d); err != nil {
				return err
			}
			if !attestColumn(d.Identity, d.Column, d.Before) {
				return fmt.Errorf("native stored declaration lost its ordered predecessor")
			}
			columnTypes[[2]string{d.Identity, d.Column}] = d.After
		case "table_rename":
			if action.TableRename == nil || action.Rename != nil || action.Property != nil || action.Key != nil || action.Foreign != nil {
				return fmt.Errorf("invalid native table rename action")
			}
			d := action.TableRename
			if err := validateTableRenameShape(d); err != nil {
				return err
			}
			if before := tableCurrent[d.Identity]; before != nil && (before.Storage != d.Storage || before.After != d.Before) {
				return fmt.Errorf("native table rename lost its ordered predecessor")
			}
			tableCurrent[d.Identity] = d
		case "column_rename":
			if action.Rename == nil || action.TableRename != nil || action.Property != nil || action.Key != nil || action.Foreign != nil {
				return fmt.Errorf("invalid native column rename action")
			}
			if err := validateColumnRenameShape(action.Rename); err != nil {
				return err
			}
			d := action.Rename
			key := [2]string{d.Identity, d.Storage}
			if !attestColumn(d.Identity, d.Storage, d.Expected) {
				return fmt.Errorf("native column rename lost its ordered type predecessor")
			}
			if before := renameCurrent[key]; before != nil && (before.Table != d.Table || before.After != d.Before || !orderedTypes && !before.Expected.equal(d.Expected)) {
				return fmt.Errorf("native column rename lost its ordered predecessor")
			}
			renameCurrent[key] = d
		case "column_property_claim", "column_property_drop":
			if action.Property == nil || action.Key != nil || action.Foreign != nil {
				return fmt.Errorf("invalid native column property action")
			}
			if err := validateColumnPropertyChangeShape(action.Property); err != nil {
				return err
			}
			d := action.Property
			if !attestColumn(d.Identity, d.Column, d.Expected) {
				return fmt.Errorf("native column property lost its ordered type predecessor")
			}
			value, _ := columnPropertyValue(d.ValueKind, d.Value)
			before, _ := columnPropertyValue(d.BeforeKind, d.Before)
			if action.Operation == "column_property_drop" && (!d.BeforePresent || !sameColumnProperty(value, before)) {
				return fmt.Errorf("column property retirement lost its predecessor value")
			}
			name := [3]string{d.Identity, d.Column, d.Key}
			if previous, seen := propertyCurrent[name]; seen {
				if previous.declaration.Table != d.Table || !orderedTypes && !previous.declaration.Expected.equal(d.Expected) ||
					previous.present != d.BeforePresent || previous.present && !sameColumnProperty(previous.value, before) {
					return fmt.Errorf("native column property action lost its ordered predecessor")
				}
			}
			propertyCurrent[name] = propertyState{d, action.Operation == "column_property_claim", value}
		case "key_add", "key_drop":
			if action.Key == nil || action.Foreign != nil || action.Property != nil {
				return fmt.Errorf("invalid native unique action")
			}
			d := *action.Key
			if !validName(d.Table) || !validIdentity(d.Identity) || !validName(d.Key.Id) || !utf8.ValidString(d.Key.ConstraintName) || !validColumns(d.Key.Cols) {
				return fmt.Errorf("invalid native unique recipe")
			}
			name := d.Identity + "/" + scm.CaseFoldKey(d.Key.Id)
			before, exists := keyCurrent[name]
			if action.Operation == "key_add" {
				if exists {
					return fmt.Errorf("native history adds existing unique declaration")
				}
				keyCurrent[name] = d
			} else {
				if keySeen[name] && (!exists || before.Table != d.Table || before.Identity != d.Identity || !sameUniqueKey(before.Key, d.Key)) {
					return fmt.Errorf("native unique retirement lost exact predecessor")
				}
				delete(keyCurrent, name)
			}
			keySeen[name] = true
		case "foreign_add", "foreign_drop":
			if action.Foreign == nil || action.Key != nil || action.Property != nil {
				return fmt.Errorf("invalid native foreign action")
			}
			d := *action.Foreign
			fk := d.Foreign
			if !validName(fk.Id) || !validName(fk.Tbl1) || !validName(fk.Tbl2) || !validColumns(fk.Cols1) || !validColumns(fk.Cols2) || len(fk.Cols1) != len(fk.Cols2) || fk.Updatemode > SETNULL || fk.Deletemode > SETNULL || !validIdentity(d.ChildIdentity) || !validIdentity(d.ParentIdentity) {
				return fmt.Errorf("invalid native foreign recipe")
			}
			if fk.Tbl1 == fk.Tbl2 && (fk.Updatemode != RESTRICT || fk.Deletemode != RESTRICT || d.ChildIdentity != d.ParentIdentity) {
				return fmt.Errorf("invalid native self foreign recipe")
			}
			name := foreignIdentity(d)
			before, exists := foreignCurrent[name]
			if action.Operation == "foreign_add" {
				if exists {
					return fmt.Errorf("native history adds existing foreign declaration")
				}
				foreignCurrent[name] = d
			} else {
				if foreignSeen[name] && (!exists || !sameCatalogForeign(before, d)) {
					return fmt.Errorf("native foreign retirement lost exact predecessor")
				}
				delete(foreignCurrent, name)
			}
			foreignSeen[name] = true
		default:
			return fmt.Errorf("invalid native declaration operation")
		}
	}
	return nil
}

func uniqueKeyPayloadBytes(key uniqueKey) uint {
	bytes := uint(cap(key.Cols))*uint(unsafe.Sizeof("")) + uint(len(key.Id)+len(key.ConstraintName))
	for _, name := range key.Cols {
		bytes += uint(len(name))
	}
	return bytes
}
