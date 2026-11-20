/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "unsafe"

// This is one immutable logical target input of an admitted native producer.
// It owns no table facade, physical index, row identity or independent decision.
// The existing statement lease/drain retains its native target and declarations.
type nativeMutationInput struct {
	target       *table
	owner        *transactionState
	columns      []*column
	unique       []uniqueKey
	foreign      []foreignKey
	beforeUpdate []TriggerDescription
	afterUpdate  []TriggerDescription
	memory       *publicCatalogMemory
	bytes        uint64
}

type nativeTriggerSlot struct {
	name   string
	timing TriggerTiming
}

// This is only a setup candidate, never a row or callback capability. The
// caller must acquire a real checked statement and bind the complete input.
func (tx *TxContext) privateMutationSource(t *table) *publicCatalogViews {
	if tx == nil || tx.transactionState == nil || tx.Mode != TxACID || tx.State != TxActive {
		return nil
	}
	if changes := tx.publicCatalogChanges(); changes != nil && changes.owner == tx.transactionState && changes.db == t.schema {
		if view := changes.views.Load(); view != nil {
			// Ordinary stored-column-only inputs already retain their existing
			// native INSERT/UPDATE owner. This alternative is needed only where
			// key/FK retirement otherwise blocks that actual endpoint's writers.
			if _, blocked := view.blockedWriters[t]; !blocked {
				return nil
			}
			if _, allowed := view.checkedWriters[t]; allowed {
				return view
			}
		}
	}
	return nil
}

func (tx *TxContext) mutationInputFor(t *table) *nativeMutationInput {
	if tx != nil && tx.mutationOperation != nil && tx.declarationInput != nil &&
		tx.declarationInput.target == t && tx.declarationInput.owner == tx.transactionState {
		return tx.declarationInput
	}
	return nil
}

// Source/history and descriptors come from the same immutable publication.
// New enforcing programs outside the bounded retirement/key/column profile
// continue to prohibit writes; they are not silently treated as committed.
func checkedCatalogWriters(history []catalogDeclarationAmendment) map[*table]struct{} {
	result, blocked := make(map[*table]struct{}), make(map[*table]struct{})
	for _, entry := range history {
		if key := entry.publicKey; key != nil {
			result[key.table] = struct{}{}
		}
		if drop := entry.foreignDrop; drop != nil {
			for _, t := range drop.endpoints {
				if t != nil {
					result[t] = struct{}{}
				}
			}
		}
		if c := entry.column; c != nil && c.declaration != nil {
			result[c.table] = struct{}{}
		}
		if c := entry.column; c != nil && c.appendDeclaration != nil {
			blocked[c.table] = struct{}{}
		}
		if c := entry.check; c != nil {
			blocked[c.table] = struct{}{}
		}
		if c := entry.checkDrop; c != nil {
			blocked[c.table] = struct{}{}
		}
		if state := entry.constraintState; state != nil {
			for _, view := range state.views {
				blocked[view.table] = struct{}{}
			}
		}
		if tr := entry.statementTrigger; tr != nil {
			blocked[tr.slot.table] = struct{}{}
		}
		if drop := entry.tableDrop; drop != nil {
			blocked[drop.table] = struct{}{}
		}
	}
	for _, addition := range currentForeignAdditions(history) {
		if addition != nil {
			for _, t := range addition.endpoints {
				if t != nil {
					blocked[t] = struct{}{}
				}
			}
		}
	}
	for t := range blocked {
		delete(result, t)
	}
	return result
}

func (t *table) bindMutationInput(tx *TxContext, source *publicCatalogViews) *nativeMutationInput {
	if source == nil || tx.privateMutationSource(t) != source || source.tables[t] == nil {
		panic("native mutation input lost its private declaration source")
	}
	operation := tx.mutationOperation
	operation.mu.Lock()
	root := operation
	if operation.undoOwner != nil {
		root = operation.undoOwner
	}
	operation.mu.Unlock()
	root.mu.Lock()
	statement := root.statement
	var lease *statementTableWriteLease
	if statement != nil && !statement.finished {
		lease = statement.tables[t]
	}
	root.mu.Unlock()
	if lease == nil {
		panic("private declaration input requires its admitted statement target")
	}
	owner := &t.writeAdmission
	owner.mu.Lock()
	valid := owner.statement == lease && owner.declaration == tx.transactionState && lease.transaction == tx.transactionState
	owner.mu.Unlock()
	if !valid {
		panic("private declaration input lost its actual endpoint lease")
	}
	// Compilation is outside all operation/schema/table/UNIQUE locks. Existing
	// producer work keeps the statement lease live until its checked drain.
	retired := make(map[nativeTriggerSlot]struct{})
	retiredForeign := make(map[catalogForeignIdentity][]catalogForeignDeclaration)
	for _, entry := range source.history {
		if drop := entry.foreignDrop; drop != nil {
			identity := foreignIdentity(drop.declaration)
			retiredForeign[identity] = append(retiredForeign[identity], drop.declaration)
			for _, program := range drop.programs {
				if program.table == t {
					retired[nativeTriggerSlot{program.trigger.Name, program.trigger.Timing}] = struct{}{}
				}
			}
		}
	}
	var foreign []foreignKey
	// A legacy program-name collision must not suppress another retained FK.
	// Selection is owned by exact native recipes, not a textual prefix filter.
	for _, fk := range t.Foreign {
		identity := t.schema.storedForeignIdentity(fk)
		removed := false
		for _, old := range retiredForeign[identity] {
			if sameForeignDeclaration(old.Foreign, fk) {
				removed = true
				break
			}
		}
		if removed {
			continue
		}
		foreign = append(foreign, fk)
		child, parent := t.schema.tables.Get(fk.Tbl1), t.schema.tables.Get(fk.Tbl2)
		if child == nil || parent == nil {
			panic("native FK input has a missing endpoint")
		}
		for _, program := range foreignDropPrograms(child, parent, fk) {
			if program.table == t {
				if _, collision := retired[nativeTriggerSlot{program.trigger.Name, program.trigger.Timing}]; collision {
					panic("retired FK program is shared by another native declaration")
				}
			}
		}
	}
	input := &nativeMutationInput{target: t, owner: tx.transactionState,
		columns: source.tables[t].metadata.columnNames.declarations, unique: uniqueKeysInView(t, source.keys), foreign: foreign}
	input.beforeUpdate = t.GetTriggers(BeforeUpdate, retired)
	input.afterUpdate = t.GetTriggers(AfterUpdate, retired)
	if tx.privateMutationSource(t) != source {
		panic("private declaration input changed while preparing its programs")
	}
	// Only the new header/vector containers belong to this producer. Column,
	// key, FK-string and compiled-program payloads retain their existing owners.
	input.bytes = uint64(unsafe.Sizeof(*input)) + uint64(cap(input.foreign))*uint64(unsafe.Sizeof(foreignKey{})) +
		uint64(cap(input.beforeUpdate)+cap(input.afterUpdate))*uint64(unsafe.Sizeof(TriggerDescription{}))
	input.memory = tx.publicCatalogChanges().memory
	input.memory.mutationInputBytes.Add(input.bytes)
	return input
}

// The actual producer calls this only after its checked frame/work drain.
func (input *nativeMutationInput) release() {
	if input != nil {
		input.memory.mutationInputBytes.Add(^(input.bytes - 1))
	}
}

func (t *table) mutationTriggers(timing TriggerTiming, input *nativeMutationInput) []TriggerDescription {
	if input != nil {
		if input.target != t {
			panic("native mutation input belongs to another target")
		}
		switch timing {
		case BeforeUpdate:
			return input.beforeUpdate
		case AfterUpdate:
			return input.afterUpdate
		}
	}
	return t.GetTriggers(timing, nil)
}

// All callers bind the input during setup. Row callbacks retain this immutable
// pointer; they never reread a transaction's amendment/history directory.
func mutationUniqueKeys(t *table, input *nativeMutationInput) []uniqueKey {
	if input != nil {
		return input.unique
	}
	return t.Unique
}

func mutationColumns(t *table, tx *TxContext, input *nativeMutationInput) []*column {
	if input != nil {
		return input.columns
	}
	return t.columnDeclarationsForTx(tx)
}

func mutationForeignKeys(t *table, tx *TxContext) []foreignKey {
	if input := tx.mutationInputFor(t); input != nil {
		return input.foreign
	}
	return t.Foreign
}

func (t *table) requireUnusedStatementDeclarations() {
	owner := &t.writeAdmission
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.statement != nil {
		panic("native declaration input is in use by an admitted statement")
	}
}
