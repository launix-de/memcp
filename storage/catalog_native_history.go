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
	Operation string                     `json:"operation"`
	Key       *catalogKeyDeclaration     `json:"key,omitempty"`
	Foreign   *catalogForeignDeclaration `json:"foreign,omitempty"`
}

func (tx *TxContext) publicNativeDeclarations() []catalogNativeDeclaration {
	var result []catalogNativeDeclaration
	history := tx.publicForeignAmendments()
	// Pending state transitions belong to this same amendment history. Freeze
	// each addition's accepted validation fact once, including its matching DROP;
	// recovery never publishes a transient pending state or a second projection.
	trusted := make(map[*catalogForeignAmendment]bool)
	for _, entry := range history {
		if add := entry.foreign; add != nil {
			trusted[add] = !add.declaration.Foreign.Untrusted
		}
		if state := entry.constraintState; state != nil {
			for _, current := range state.current {
				if current.addition != nil {
					trusted[current.addition] = current.trusted
				}
			}
		}
	}
	for _, entry := range history {
		if key := entry.publicKey; key != nil {
			d := key.declaration
			result = append(result, catalogNativeDeclaration{Operation: "key_" + key.operation, Key: &d})
		}
		if add := entry.foreign; add != nil {
			d := add.declaration
			d.Foreign.Untrusted = !trusted[add]
			result = append(result, catalogNativeDeclaration{Operation: "foreign_add", Foreign: &d})
		}
		if drop := entry.foreignDrop; drop != nil {
			d := drop.declaration
			if drop.addition != nil {
				d.Foreign.Untrusted = !trusted[drop.addition]
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
	keys           map[*table][]uniqueKey
	foreign        map[catalogForeignIdentity][]catalogForeignDeclaration
	changedForeign map[catalogForeignIdentity]bool
	foreignAdds    []catalogForeignDeclaration
	foreignDrops   []catalogForeignDeclaration
}

func newCatalogNativeProjection(db *database) *catalogNativeProjection {
	p := &catalogNativeProjection{keys: make(map[*table][]uniqueKey), foreign: make(map[catalogForeignIdentity][]catalogForeignDeclaration), changedForeign: make(map[catalogForeignIdentity]bool)}
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
func (db *database) projectNativeDeclarations(actions []catalogNativeDeclaration, views map[*table]*tableShowColumnsSnapshot, domain catalogNativeReplayDomain) *catalogNativeProjection {
	if domain != catalogNativeCurrent && domain != catalogNativeLegacyV13 {
		panic("invalid native declaration replay domain")
	}
	p := newCatalogNativeProjection(db)
	added := make(map[catalogForeignIdentity]int)
	for index, action := range actions {
		switch action.Operation {
		case "foreign_add", "foreign_drop":
			if action.Foreign == nil || action.Key != nil {
				panic("invalid native foreign action")
			}
			d := *action.Foreign
			validateCatalogForeign(db, d, p.keys, views)
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
		case "key_add", "key_drop":
			if action.Key == nil || action.Foreign != nil {
				panic("invalid native unique action")
			}
			d := *action.Key
			t := validateCatalogKey(db, d, views)
			keys := uniqueKeysInView(t, p.keys)
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

func (db *database) recoverNativeDeclarations(actions []catalogNativeDeclaration, domain catalogNativeReplayDomain) {
	if domain != catalogNativeCurrent && domain != catalogNativeLegacyV13 {
		panic("invalid native declaration replay domain")
	}
	if len(actions) == 0 {
		return
	}
	db.projectNativeDeclarations(actions, nil, domain)
	for _, action := range actions {
		switch action.Operation {
		case "foreign_add":
			db.recoverPublicForeignDeclarations([]catalogForeignDeclaration{*action.Foreign})
		case "foreign_drop":
			db.recoverPublicForeignDrops([]catalogForeignDeclaration{*action.Foreign})
		case "key_add":
			t := validateCatalogKey(db, *action.Key, nil)
			t.Unique = append(append([]uniqueKey(nil), t.Unique...), action.Key.Key)
		case "key_drop":
			t := validateCatalogKey(db, *action.Key, nil)
			t.Unique = slices.DeleteFunc(t.Unique, func(key uniqueKey) bool { return sameUniqueKey(key, action.Key.Key) })
		}
	}
}

func validateNativeDeclarationShape(actions []catalogNativeDeclaration) error {
	if len(actions) == 0 {
		return fmt.Errorf("native declaration history is empty")
	}
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
	for _, action := range actions {
		switch action.Operation {
		case "key_add", "key_drop":
			if action.Key == nil || action.Foreign != nil {
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
			if action.Foreign == nil || action.Key != nil {
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
