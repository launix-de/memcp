/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "strings"
import "github.com/launix-de/memcp/scm"

// Mixed addition/retirement has ordered durable authority. The stream contains
// only inert native recipes; no compiled program, table view or second decision.
type catalogForeignAction struct {
	Operation   string                    `json:"operation"`
	Declaration catalogForeignDeclaration `json:"declaration"`
}

func sameCatalogForeign(a, b catalogForeignDeclaration) bool {
	return a.ChildIdentity == b.ChildIdentity && a.ParentIdentity == b.ParentIdentity && sameForeignDeclaration(a.Foreign, b.Foreign)
}

// Native FK identity is the actual pair of relation generations plus its
// folded name. Legacy schemas can contain the same name on different pairs;
// name admission remains a separate database-wide declaration check.
type catalogForeignIdentity struct {
	child      string
	parent     string
	childName  string
	parentName string
	name       string
}

func foreignIdentity(d catalogForeignDeclaration) catalogForeignIdentity {
	return catalogForeignIdentity{d.ChildIdentity, d.ParentIdentity, d.Foreign.Tbl1, d.Foreign.Tbl2, scm.CaseFoldKey(d.Foreign.Id)}
}

func (db *database) storedForeignIdentity(fk foreignKey) catalogForeignIdentity {
	// Descriptive baseline collection does not repair absent legacy identities
	// or endpoints. Actual action admission still validates both generations.
	childID, parentID := "", ""
	if child := db.tables.Get(fk.Tbl1); child != nil {
		childID = child.RuntimeIdentity
	}
	if parent := db.tables.Get(fk.Tbl2); parent != nil {
		parentID = parent.RuntimeIdentity
	}
	return catalogForeignIdentity{childID, parentID, fk.Tbl1, fk.Tbl2, scm.CaseFoldKey(fk.Id)}
}

// One cold producer derives this immutable header view once. A present nil
// entry is a retirement; absence leaves the committed declaration untouched.
func currentForeignAdditions(history []catalogDeclarationAmendment) map[catalogForeignIdentity]*catalogForeignAmendment {
	current := make(map[catalogForeignIdentity]*catalogForeignAmendment)
	for _, entry := range history {
		if add := entry.foreign; add != nil {
			current[foreignIdentity(add.declaration)] = add
		}
		if drop := entry.foreignDrop; drop != nil {
			current[foreignIdentity(drop.declaration)] = nil
		}
	}
	return current
}

// Compare complete retired recipes, not merely a declaration name. Legacy
// endpoints may retain another recipe with the same folded pair/name identity.
func retiredForeignRecipe(db *database, fk foreignKey, retired map[catalogForeignIdentity][]catalogForeignDeclaration) bool {
	for _, previous := range retired[db.storedForeignIdentity(fk)] {
		if sameForeignDeclaration(previous.Foreign, fk) {
			return true
		}
	}
	return false
}

// Callers own native schema publication. Only exact committed predecessors
// named by this history may vacate their declaration names; other kinds retain
// their existing shared namespace owner.
func (db *database) foreignNameConflict(name string, retired []catalogForeignDeclaration) bool {
	for _, t := range db.tables.GetAll() {
		if t.hasNonForeignDeclarationName(name, nil, false) {
			return true
		}
		for _, fk := range t.Foreign {
			if fk.Tbl1 != t.Name || !strings.EqualFold(fk.Id, name) {
				continue
			}
			allowed := false
			for _, old := range retired {
				parent := db.tables.Get(fk.Tbl2)
				if parent != nil && old.ChildIdentity == t.RuntimeIdentity && old.ParentIdentity == parent.RuntimeIdentity && sameForeignDeclaration(old.Foreign, fk) {
					allowed = true
				}
			}
			if !allowed {
				return true
			}
		}
	}
	return false
}

func (tx *TxContext) foreignReplacement(name string, child, parent *table) bool {
	changes := tx.publicCatalogChanges()
	if changes == nil || changes.owner != tx.transactionState {
		return false
	}
	views := changes.views.Load()
	if views == nil {
		return false
	}
	for _, t := range foreignEndpointTables(child, parent) {
		if _, ok := views.blockedWriters[t]; !ok {
			return false
		}
	}
	history := tx.publicForeignAmendments()
	for i := len(history) - 1; i >= 0; i-- {
		if add := history[i].foreign; add != nil && strings.EqualFold(add.declaration.Foreign.Id, name) {
			return false
		}
		if drop := history[i].foreignDrop; drop != nil && strings.EqualFold(drop.declaration.Foreign.Id, name) {
			return true
		}
	}
	return false
}

// Declarations and program reservations remain in native history until its
// actual savepoint/decision owner retires them. Borrowing one does not transfer
// its release obligation to a later amendment.
func foreignHistoryHasProgram(history []catalogDeclarationAmendment, table *table, name string) bool {
	for _, entry := range history {
		if drop := entry.checkDrop; drop != nil && drop.table == table {
			for _, timing := range []TriggerTiming{BeforeInsert, BeforeUpdate} {
				if drop.declaration.Check.triggerName(timing) == name {
					return true
				}
			}
		}
		var programs []foreignKeyTrigger
		if entry.foreign != nil {
			programs = entry.foreign.programs
		}
		if entry.foreignDrop != nil {
			programs = entry.foreignDrop.programs
		}
		for _, program := range programs {
			if program.table == table && program.trigger.Name == name {
				return true
			}
		}
	}
	return false
}

func (changes *publicCatalogChanges) reserveForeignPrograms(tx *TxContext, programs []foreignKeyTrigger) (reserved []foreignKeyTrigger) {
	complete := false
	defer func() {
		if !complete {
			releaseForeignTriggerPrograms(tx.transactionState, reserved)
		}
	}()
	for _, program := range programs {
		if foreignHistoryHasProgram(tx.publicForeignAmendments(), program.table, program.trigger.Name) {
			changes.validateForeignReservation(tx, program)
			continue
		}
		program.table.reserveDeclarationTrigger(tx.transactionState, program.trigger.Name, false)
		reserved = append(reserved, program)
	}
	complete = true
	return
}

func (changes *publicCatalogChanges) validateForeignReservation(tx *TxContext, program foreignKeyTrigger) {
	t := program.table
	t.mu.Lock()
	defer t.mu.Unlock()
	owner := &t.writeAdmission
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.declaration != tx.transactionState || tx.transactionState == nil {
		panic("foreign history lost its actual declaration owner")
	}
	if _, ok := owner.triggerNames[program.trigger.Name]; !ok {
		panic("foreign history lost its native program reservation")
	}
	for _, trigger := range t.Triggers {
		if trigger.Name != program.trigger.Name {
			continue
		}
		matched := false
		for _, entry := range tx.publicForeignAmendments() {
			if drop := entry.checkDrop; drop != nil && drop.table == t && trigger.IsSystem &&
				trigger.checkDefinition == drop.declaration.Check && trigger.Timing == program.trigger.Timing &&
				(trigger.Timing == BeforeInsert || trigger.Timing == BeforeUpdate) &&
				trigger.Name == drop.declaration.Check.triggerName(trigger.Timing) {
				matched = true
			}
			if drop := entry.foreignDrop; drop != nil && drop.addition == nil {
				for _, old := range drop.programs {
					if old.table == t && old.trigger.Name == trigger.Name && old.trigger.Timing == trigger.Timing && trigger.IsSystem {
						matched = true
					}
				}
			}
		}
		if !matched {
			panic("foreign history program conflicts with a published trigger")
		}
	}
}

// The strict stream checker validates order without reading any database. The
// first occurrence still needs actual checkpoint authority, proved below by
// prepare/recovery before any effect. Repeated DROP must match the immediately
// preceding ADD's complete recipe and both immutable endpoint identities.
func validateForeignActionOrder(actions []catalogForeignAction) error {
	seen := make(map[string]bool)
	current := make(map[string]catalogForeignDeclaration)
	added, removed := false, false
	for _, action := range actions {
		name := scm.CaseFoldKey(action.Declaration.Foreign.Id)
		before, exists := current[name]
		switch action.Operation {
		case "add":
			if exists {
				return fmt.Errorf("foreign history adds an existing declaration")
			}
			current[name] = action.Declaration
			added = true
		case "drop":
			if seen[name] && (!exists || !sameCatalogForeign(before, action.Declaration)) {
				return fmt.Errorf("foreign history removal lost its exact predecessor")
			}
			delete(current, name)
			removed = true
		default:
			return fmt.Errorf("invalid native foreign history operation")
		}
		seen[name] = true
	}
	if !added || !removed {
		return fmt.Errorf("foreign history requires addition and retirement")
	}
	return nil
}

// Caller owns schema publication (or the unpublished loader). This computes a
// final native projection, not a different execution order: no intermediate
// recipe becomes visible and recovery below consumes the stream in order.
func nativeForeignActions(actions []catalogForeignAction) []catalogNativeDeclaration {
	result := make([]catalogNativeDeclaration, 0, len(actions))
	for _, action := range actions {
		d := action.Declaration
		result = append(result, catalogNativeDeclaration{Operation: "foreign_" + action.Operation, Foreign: &d})
	}
	return result
}

func (db *database) recoverPublicForeignHistory(actions []catalogForeignAction) {
	db.recoverNativeDeclarations(nativeForeignActions(actions), catalogNativeLegacyV13, catalogForeignEndpointScope{})
}
