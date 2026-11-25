/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "sort"
import "unsafe"
import "errors"
import "github.com/launix-de/memcp/scm"

// Statement-only and complete original-owned generation inputs support every
// existing DML event. Key/layout amendment inputs retain their earlier bound.
type nativeMutationInputProfile uint8

const (
	nativeDeclarationRowInput nativeMutationInputProfile = iota + 1
	nativeStatementTriggerRowInput
	nativeOwnedCatalogRowInput
	nativeColumnBindingRowInput
	nativeColumnRetirementRowInput
	nativeCheckRetirementRowInput
)

func (views *publicCatalogViews) mutationEventAllowed(t *table, event nativeMutationEvent) bool {
	switch views.checkedWriters[t] {
	case nativeDeclarationRowInput:
		return event == nativeMutationInsert || event == nativeMutationUpdate
	case nativeStatementTriggerRowInput, nativeOwnedCatalogRowInput, nativeColumnBindingRowInput, nativeColumnRetirementRowInput, nativeCheckRetirementRowInput:
		return event == nativeMutationInsert || event == nativeMutationUpdate || event == nativeMutationDelete
	default:
		return false
	}
}

// This is one immutable logical target input of an admitted native producer.
// It owns no table facade, physical index, row identity or independent decision.
// The existing statement lease/drain retains its native target and declarations.
type nativeMutationInput struct {
	// A plain producer's runtime header carries no declaration view or row
	// authority. Native consumers unwrap the identical original executor; only
	// mapper setup consumes its actual event and targets. No wire encoding.
	execution    *TxContext
	targets      []*table
	target       *table
	source       *publicCatalogViews
	owner        *transactionState
	event        nativeMutationEvent
	columns      []*column
	unique       []uniqueKey
	foreign      []foreignKey
	beforeInsert []TriggerDescription
	afterInsert  []TriggerDescription
	beforeUpdate []TriggerDescription
	afterUpdate  []TriggerDescription
	beforeDelete []TriggerDescription
	afterDelete  []TriggerDescription
	memory       *publicCatalogMemory
	bytes        uint64
}

func (*nativeMutationInput) MarshalJSON() ([]byte, error) {
	return nil, errors.New("native mutation execution input is local and cannot be serialized")
}

func (*nativeMutationInput) ClusterObjectType() string { return "local-mutation-input" }

func (*nativeMutationInput) Serialize(*scm.SerializationContext) (scm.Scmer, error) {
	return scm.NewNil(), errors.New("native mutation execution input is local and cannot be serialized")
}

func (input *nativeMutationInput) originalExecution() *TxContext {
	if input == nil || input.execution == nil || input.owner == nil || input.execution.transactionState != input.owner {
		panic("native mutation input lost its original transaction")
	}
	if input.execution.State != TxActive {
		panic("native mutation input original transaction has finished")
	}
	return input.execution
}

func (input *nativeMutationInput) exactUpdateFor(t *table, tx *TxContext) bool {
	if input == nil {
		return false
	}
	if input.event == nativeMutationUpdate {
		for _, target := range input.targets {
			if target == t {
				if input.originalExecution() != tx {
					panic("native mutation input belongs to another executor")
				}
				return true
			}
		}
	}
	return false
}

func producerInputFromScmer(value scm.Scmer) *nativeMutationInput {
	if !value.IsNil() {
		if input, ok := value.Any().(*nativeMutationInput); ok {
			input.originalExecution()
			return input
		}
	}
	return nil
}

// The selected physical source is still owned by its operator here. A callback
// can outlive that scan, so retain this exact generation in the ORIGINAL
// transaction before the operation pin drains. The existing overlay owns the
// transaction pin and releases it at decision; no row effect or WAL is added.
// Call once at a nonempty UPDATE input/queue boundary, never from a row reader.
func (s *storageShard) retainMutationSource(tx *TxContext, input *nativeMutationInput, columns []string) {
	if tx == nil || tx.remoteObject != nil || tx.remoteInvocation || tx.coordinator != nil || tx.relationData != nil ||
		activeReplication(s.t.schema) != nil || sharedClusterSource(s.t) {
		return
	}
	updates := false
	for _, column := range columns {
		updates = updates || column == "$update"
	}
	if !updates {
		return
	}
	exact := input.exactUpdateFor(s.t, tx)
	if operation := tx.mutationOperation; operation != nil {
		operation.mu.Lock()
		effect := operation.effects[s.t]
		exact = effect != nil && effect.event == nativeMutationUpdate && !operation.closed &&
			!operation.producerDone && operation.state == tx.transactionState && operation.invocation == tx
		operation.mu.Unlock()
	}
	if !exact {
		return
	}
	if directory := tx.shards.Load(); directory != nil {
		if _, retained := directory.Load(s); retained {
			return // an actual owned overlay, never a borrowed read-view mask
		}
	}
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.State != TxActive {
		panic("native mutation source original transaction has finished")
	}
	tx.getOrCreateShardTxLocked(s)
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
// Statement programs use the same later-producer admission; other enforcing
// programs outside this bounded input still prohibit writes.
func checkedCatalogWriters(history []catalogDeclarationAmendment, statements, columns, checks map[*table]struct{}, owner *transactionState, leased map[*table]*tableWriteLease) map[*table]nativeMutationInputProfile {
	result, blocked := make(map[*table]nativeMutationInputProfile), make(map[*table]struct{})
	// Rewound heads still have their original reserved slot/declaration lease.
	// The later checked owner binds its actual baseline programs, not a dropped
	// historical recipe. Current endpoint amendments below retain every veto.
	for t := range statements {
		result[t] = nativeStatementTriggerRowInput
	}
	for t := range columns {
		// The retained lease alone selects an unchanged baseline. Actual key,
		// property and layout amendments below impose their narrower profiles.
		result[t] = nativeColumnBindingRowInput
	}
	for t := range checks {
		result[t] = nativeCheckRetirementRowInput
	}
	for t, lease := range leased {
		if lease != nil && lease.owner == &t.writeAdmission && lease.transaction == owner && (catalogForeignEndpointScope{owner: owner}).owns(t) {
			result[t] = nativeOwnedCatalogRowInput
		}
	}
	for _, entry := range history {
		if key := entry.publicKey; key != nil {
			if key.index != nil && key.index.Index.Key == nil {
				// Advisory names do not alter row enforcement. Preserve any actual
				// key/layout veto already selected by the same chronological history.
				if result[key.table] != nativeDeclarationRowInput {
					result[key.table] = nativeColumnBindingRowInput
				}
			} else {
				result[key.table] = nativeDeclarationRowInput
			}
		}
		if drop := entry.foreignDrop; drop != nil {
			for _, t := range drop.endpoints {
				if t != nil {
					result[t] = nativeDeclarationRowInput
				}
			}
		}
		if r := entry.tableRename; r != nil {
			if result[r.table] != nativeDeclarationRowInput {
				result[r.table] = nativeColumnBindingRowInput
			}
		}
		if c := entry.column; c != nil && !c.nameMetadataOnly {
			if c.rename != nil {
				if _, exists := result[c.table]; !exists || result[c.table] != nativeDeclarationRowInput {
					result[c.table] = nativeColumnBindingRowInput
				}
			} else if c.declaration != nil || c.appendDeclaration != nil || c.propertyChange != nil || c.drop != nil {
				result[c.table] = nativeDeclarationRowInput
			} else if _, exists := result[c.table]; !exists {
				result[c.table] = nativeDeclarationRowInput
			}
		}
		if c := entry.column; c != nil && c.appendDeclaration != nil {
			if c.appendDeclaration.Initialization != nil || c.appendDeclaration.RowInitialization != nil || c.appendDeclaration.GeneratedIDs != nil {
				result[c.table] = nativeDeclarationRowInput
			} else {
				blocked[c.table] = struct{}{}
			}
		}
		if c := entry.check; c != nil {
			blocked[c.table] = struct{}{}
		}
		if c := entry.checkDrop; c != nil {
			if c.rowInput {
				result[c.table] = nativeCheckRetirementRowInput
			} else {
				blocked[c.table] = struct{}{}
			}
		}
		if state := entry.constraintState; state != nil {
			for _, view := range state.views {
				blocked[view.table] = struct{}{}
			}
		}
		if tr := entry.statementTrigger; tr != nil {
			if _, found := result[tr.slot.table]; !found {
				result[tr.slot.table] = nativeStatementTriggerRowInput
			}
		}
		if drop := entry.tableDrop; drop != nil {
			blocked[drop.table] = struct{}{}
		}
	}
	for _, addition := range currentForeignAdditions(history) {
		if addition != nil {
			owned := ownedForeignAddition(owner, addition)
			for _, t := range addition.endpoints {
				if t != nil {
					if owned {
						if profile, exists := result[t]; !exists || profile == nativeStatementTriggerRowInput {
							result[t] = nativeOwnedCatalogRowInput
						}
					} else {
						blocked[t] = struct{}{}
					}
				}
			}
		}
	}
	// Retirement has already proven the remaining native descriptor program.
	// The same checked source input owns DELETE as well as INSERT/UPDATE; no
	// raw writer may borrow the changed layout or retired physical spelling.
	for _, entry := range history {
		if c := entry.column; c != nil && c.drop != nil {
			result[c.table] = nativeColumnRetirementRowInput
		}
	}
	for t := range blocked {
		delete(result, t)
	}
	return result
}

// A clean private CREATE owner can acquire public data only through a real
// local checked producer. This selects setup, never grants a raw row writer.
func (tx *TxContext) catalogCreationRowSource(t *table) bool {
	if tx == nil || tx.transactionState == nil || tx.Mode != TxACID || tx.State != TxActive ||
		tx.remoteObject != nil || tx.remoteInvocation || tx.coordinator != nil || tx.relationData != nil ||
		tx.publicCatalogEntered.Load() || t == nil || t.Ephemeral || t.ScopedSchema {
		return false
	}
	identity := tx.catalogIdentity.Load()
	return identity != nil && !identity.lateCreates && identity.Namespace == "" &&
		identity.database == t.schema && identity.Incarnation == t.schema.incarnationID() &&
		GetDatabase(identity.Database) == identity.database && t.schema.tables.Get(t.Name) == t &&
		t.committedAt(t.schema.catalogView().version) && tx.catalogIntents.Load() != nil
}

func (t *table) bindMutationInput(tx *TxContext, source *publicCatalogViews, event nativeMutationEvent) *nativeMutationInput {
	layout := tx.scopedRelationLayout(t)
	if source == nil && layout != nil {
		tx.requireScopedColumnOwner(t)
	} else if source == nil {
		if !tx.catalogCreationRowSource(t) {
			panic("native mutation input lost its original creation source")
		}
		for _, created := range *tx.catalogIntents.Load() {
			if !(catalogForeignEndpointScope{owner: tx.transactionState}).owns(created) {
				panic("native mutation input lost its original private generation")
			}
		}
	} else if tx.privateMutationSource(t) != source || source.tables[t] == nil || !source.mutationEventAllowed(t, event) {
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
	if statement != nil && !statement.finished && (source != nil ||
		statement.ready && statement.database == t.schema && root.localOrigin && root.state == tx.transactionState) {
		lease = statement.tables[t]
	}
	root.mu.Unlock()
	if lease == nil {
		panic("private declaration input requires its admitted statement target")
	}
	owner := &t.writeAdmission
	owner.mu.Lock()
	valid := owner.statement == lease && lease.transaction == tx.transactionState &&
		(source != nil && owner.declaration == tx.transactionState || source == nil && owner.declaration == nil || layout != nil && layout.lease.transaction == tx.transactionState && owner.declaration == tx.transactionState)
	owner.mu.Unlock()
	if !valid {
		panic("private declaration input lost its actual endpoint lease")
	}
	// Compilation is outside all operation/schema/table/UNIQUE locks. Existing
	// producer work keeps the statement lease live until its checked drain.
	var history []catalogDeclarationAmendment
	var selected *tableShowColumnsSnapshot
	var keys []uniqueKey
	if source != nil {
		history, selected, keys = source.history, source.tables[t], uniqueKeysInView(t, source.keys)
	} else if layout != nil {
		tx.requireScopedColumnOwner(t)
		selected, keys = layout.view, layout.keys
	} else {
		// Use the same original-T declaration selector as SHOW/source binding,
		// under its actual statement lease. Never guess from a fresh SQL name.
		selected, keys = t.declarationMetadataForTx(tx), t.Unique
		published := t.showColumnsSnapshot.Load()
		if selected == nil || published == nil || selected.metadata.declarations != published.metadata.declarations ||
			selected.metadata.columnNames != published.metadata.columnNames || selected.metadata.columnNames != t.columnNamesForTx(tx) ||
			!t.visibleIn(tx, t.schema.catalogView().version) {
			panic("native creation source lost its selected declaration cut")
		}
	}
	retired := make(map[nativeTriggerSlot]struct{})
	retiredForeign := make(map[catalogForeignIdentity][]catalogForeignDeclaration)
	for _, entry := range history {
		if drop := entry.checkDrop; drop != nil && drop.table == t && drop.rowInput {
			for _, timing := range []TriggerTiming{BeforeInsert, BeforeUpdate} {
				program := foreignKeyTrigger{table: t, trigger: checkTriggerHeader(drop.declaration.Check, timing)}
				tx.publicCatalogChanges().validateForeignReservation(tx, program)
				retired[nativeTriggerSlot{program.trigger.Name, timing}] = struct{}{}
			}
		}
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
		if retiredForeignRecipe(t.schema, fk, retiredForeign) {
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
	var addedPrograms []TriggerDescription
	active := currentForeignAdditions(history)
	for _, entry := range history {
		addition := entry.foreign
		if addition == nil || active[foreignIdentity(addition.declaration)] != addition {
			continue
		}
		fk := addition.declaration.Foreign
		if fk.Tbl1 != t.Name && fk.Tbl2 != t.Name {
			continue
		}
		foreign = append(foreign, fk)
		for _, program := range addition.programs {
			if program.table == t {
				addedPrograms = append(addedPrograms, program.trigger)
			}
		}
	}
	input := &nativeMutationInput{target: t, source: source, owner: tx.transactionState, event: event,
		columns: selected.metadata.columnNames.declarations, unique: keys, foreign: foreign}
	profile := nativeTriggerProfileFor(tx)
	input.beforeInsert = t.GetTriggers(BeforeInsert, retired, profile, input)
	input.afterInsert = t.GetTriggers(AfterInsert, retired, profile, input)
	input.beforeUpdate = t.GetTriggers(BeforeUpdate, retired, profile, input)
	input.afterUpdate = t.GetTriggers(AfterUpdate, retired, profile, input)
	if event == nativeMutationDelete {
		input.beforeDelete = t.GetTriggers(BeforeDelete, retired, profile, input)
		input.afterDelete = t.GetTriggers(AfterDelete, retired, profile, input)
	}
	for _, program := range addedPrograms {
		switch program.Timing {
		case BeforeInsert:
			input.beforeInsert = append(input.beforeInsert, program)
		case AfterInsert:
			input.afterInsert = append(input.afterInsert, program)
		case BeforeUpdate:
			input.beforeUpdate = append(input.beforeUpdate, program)
		case AfterUpdate:
			input.afterUpdate = append(input.afterUpdate, program)
		case BeforeDelete:
			if event == nativeMutationDelete {
				input.beforeDelete = append(input.beforeDelete, program)
			}
		case AfterDelete:
			if event == nativeMutationDelete {
				input.afterDelete = append(input.afterDelete, program)
			}
		}
	}
	if len(addedPrograms) != 0 {
		for _, programs := range [][]TriggerDescription{input.beforeInsert, input.afterInsert, input.beforeUpdate, input.afterUpdate, input.beforeDelete, input.afterDelete} {
			sort.SliceStable(programs, func(i, j int) bool { return programs[i].Priority < programs[j].Priority })
		}
	}
	if source == nil && layout != nil {
		// The same scoped header selected before program preparation must still
		// belong to this original caller and its actual retained endpoint lease.
		// A nil public source alone never grants scoped declaration authority.
		tx.requireScopedColumnOwner(t)
		owner.mu.Lock()
		valid := tx.scopedRelationLayout(t) == layout && layout.table == t &&
			layout.scope == t.relationLifetime && layout.view == selected && sameUniqueKeys(layout.keys, keys) &&
			layout.lease != nil && layout.lease.owner == owner && layout.lease.transaction == tx.transactionState &&
			layout.lease.declaration && layout.lease.ddlTable.Load() == t &&
			owner.declaration == tx.transactionState && owner.statement == lease && lease.transaction == tx.transactionState
		owner.mu.Unlock()
		if !valid || tx.privateMutationSource(t) != nil || t.declarationMetadataForTx(tx) != selected {
			panic("scoped declaration input changed while preparing its programs")
		}
	} else if tx.privateMutationSource(t) != source || source == nil && (!tx.catalogCreationRowSource(t) ||
		t.declarationMetadataForTx(tx).metadata.declarations != selected.metadata.declarations) {
		panic("private declaration input changed while preparing its programs")
	}
	// Private source programs and new header/vector containers belong to this
	// producer. Column, key and FK-string payloads retain their existing owners.
	input.bytes += uint64(unsafe.Sizeof(*input)) + uint64(cap(input.foreign))*uint64(unsafe.Sizeof(foreignKey{})) +
		uint64(cap(input.beforeInsert)+cap(input.afterInsert)+cap(input.beforeUpdate)+cap(input.afterUpdate)+cap(input.beforeDelete)+cap(input.afterDelete))*uint64(unsafe.Sizeof(TriggerDescription{}))
	if source != nil {
		input.memory = tx.publicCatalogChanges().memory
		input.memory.mutationInputBytes.Add(input.bytes)
	} else {
		t.schema.adjustMetadataMemory(0, uint(input.bytes))
	}
	return input
}

// The actual producer calls this only after its checked frame/work drain.
func (input *nativeMutationInput) release() {
	if input != nil {
		if input.memory != nil {
			input.memory.mutationInputBytes.Add(^(input.bytes - 1))
		} else {
			input.target.schema.adjustMetadataMemory(uint(input.bytes), 0)
		}
	}
}

func (t *table) mutationTriggers(timing TriggerTiming, input *nativeMutationInput, tx *TxContext) []TriggerDescription {
	if input != nil {
		if input.target != t {
			panic("native mutation input belongs to another target")
		}
		switch timing {
		case BeforeInsert:
			return input.beforeInsert
		case AfterInsert:
			return input.afterInsert
		case BeforeUpdate:
			return input.beforeUpdate
		case AfterUpdate:
			return input.afterUpdate
		case BeforeDelete:
			return input.beforeDelete
		case AfterDelete:
			return input.afterDelete
		}
	}
	return t.GetTriggers(timing, nil, nativeTriggerProfileFor(tx), nil)
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
