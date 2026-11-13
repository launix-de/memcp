/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"fmt"
	"sync/atomic"

	"github.com/launix-de/memcp/scm"
)

const (
	relationSchemaUndecided uint32 = iota
	relationSchemaCommitted
	relationSchemaAborted
)

// A lexical relation slot owns native generations, not frontend identifiers.
// One immutable pointer is its complete committed/pending schema view. Physical
// names remain generation-unique, so borrowed table/shard capabilities never
// change identity when a slot is dropped and recreated.
type relationSchemaSlot struct {
	view atomic.Pointer[relationSchemaView]
}
type relationSchemaView struct {
	committed    *table
	pending      *table
	owner        *transactionState
	constructing bool
}
type relationSchemaSlots map[string]*relationSchemaSlot

type relationSchemaChange struct {
	scope       *relationScope
	slot        *relationSchemaSlot
	before      *relationSchemaView
	after       *relationSchemaView
	created     *table
	canceled    bool
	abandoned   bool
	declaration *relationSchemaKeyDeclaration
}

func (view *relationSchemaView) visible(tx *TxContext) *table {
	if view == nil {
		return nil
	}
	if view.owner == nil {
		return view.committed
	}
	switch view.owner.relationSchemaDecision.Load() {
	case relationSchemaCommitted:
		return view.pending
	case relationSchemaAborted:
		return view.committed
	}
	if tx != nil && tx.transactionState == view.owner {
		return view.pending
	}
	return view.committed
}
func (scope *relationScope) schemaSlot(key string) *relationSchemaSlot {
	if slots := scope.schemaSlots.Load(); slots != nil {
		return (*slots)[key]
	}
	return nil
}
func (scope *relationScope) schemaTable(key string, tx *TxContext) *table {
	if !scope.transactionalSchema || key == "" {
		panic("relation schema lookup requires a transactional scope and nonempty slot")
	}
	if scope.schemaClosed.Load() {
		return nil
	}
	slot := scope.schemaSlot(key)
	if slot == nil {
		return nil
	}
	return slot.view.Load().visible(tx)
}

// Declaration construction uses the existing decision admission, independently
// of public catalog phases and row access. No scope lock spans native DDL, row
// operators, callbacks, persistence or a transaction decision.
type relationSchemaRetirement struct{ scope *relationScope }

func (retirement *relationSchemaRetirement) Close() error {
	retirement.scope.mu.Lock()
	closing := retirement.scope.closing
	retirement.scope.mu.Unlock()
	if closing {
		retirement.scope.close()
	}
	return nil
}
func (scope *relationScope) beginSchemaChange(key string, tx *TxContext, creating, ifexists bool) (*relationSchemaChange, func()) {
	if !scope.transactionalSchema || key == "" {
		panic("relation schema operation requires a transactional scope and nonempty slot")
	}
	if tx == nil || tx.State != TxActive || tx.decisionStarted {
		panic("relation schema operation requires an active native transaction")
	}
	if tx.coordinator != nil {
		panic("relation schema declaration requires its native transaction coordinator")
	}
	finish := tx.beginCatalogConstruction(false)
	admitted := false
	defer func() {
		if !admitted {
			finish()
		}
	}()
	tx.getOrComputeResource(fmt.Sprintf("relation-schema-retirement/%p", scope), scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewAny(&relationSchemaRetirement{scope}) }))
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if scope.closing {
		panic("relation scope is closed")
	}
	slot := scope.schemaSlot(key)
	if slot == nil {
		slot = &relationSchemaSlot{}
		slots := make(relationSchemaSlots)
		if old := scope.schemaSlots.Load(); old != nil {
			for key, value := range *old {
				slots[key] = value
			}
		}
		slots[key] = slot
		scope.schemaSlots.Store(&slots)
	}
	before := slot.view.Load()
	if before != nil && before.owner != nil && (before.constructing || before.owner != tx.transactionState) {
		panic("relation schema slot has an unfinished native owner")
	}
	current := before.visible(tx)
	if creating && current != nil {
		if ifexists {
			admitted = true
			return nil, finish
		}
		panic("relation schema slot already exists")
	}
	if !creating && current == nil {
		if ifexists {
			admitted = true
			return nil, finish
		}
		panic("relation schema slot does not exist")
	}
	committed := current
	if before != nil && before.owner != nil {
		committed = before.committed
	}
	after := &relationSchemaView{committed: committed, owner: tx.transactionState, constructing: creating}
	change := &relationSchemaChange{scope: scope, slot: slot, before: before, after: after}
	slot.view.Store(after)
	scope.schemaDecisions++
	tx.mu.Lock()
	tx.relationSchemaChanges = append(tx.relationSchemaChanges, change)
	tx.mu.Unlock()
	admitted = true
	return change, finish
}
func (change *relationSchemaChange) publish(table *table) {
	change.scope.mu.Lock()
	defer change.scope.mu.Unlock()
	change.created = table
	change.after = &relationSchemaView{committed: change.after.committed, pending: table, owner: change.after.owner}
	change.slot.view.Store(change.after)
}
func (change *relationSchemaChange) cancel() {
	change.scope.mu.Lock()
	defer change.scope.mu.Unlock()
	change.slot.view.Store(change.before)
	change.canceled = true
}

// Row undo/commit and all producer leases finish before this owner retires any
// physical generation. A failed retirement remains in the SAME journal for the
// existing decision/cleanup retry; it cannot reverse the row decision.
func (tx *transactionState) finishRelationSchema(keep int, commit bool) {
	tx.mu.Lock()
	if keep < 0 || keep > len(tx.relationSchemaChanges) {
		tx.mu.Unlock()
		panic("invalid native relation schema savepoint")
	}
	changes := append([]*relationSchemaChange(nil), tx.relationSchemaChanges[keep:]...)
	terminal := tx.State != TxActive
	if terminal {
		changes = append(changes, tx.relationSchemaRetirements...)
	}
	tx.mu.Unlock()
	if len(changes) == 0 {
		return
	}
	// Reverse restoration preserves DROP/recreate chains and nested savepoints.
	if !commit {
		for i := len(changes) - 1; i >= 0; i-- {
			change := changes[i]
			if change.declaration != nil {
				if !change.canceled && !change.abandoned {
					tx.restoreScopedRelationKeys(change.declaration)
				}
				continue
			}
			change.scope.mu.Lock()
			if !change.canceled && !change.abandoned {
				change.slot.view.Store(change.before)
			}
			change.scope.mu.Unlock()
		}
	}
	if !terminal {
		// Statement Undo still owns its writer exclusions. Retire these abandoned
		// physical generations after the actual Tx decision, not from inside that
		// same statement's savepoint cleanup. Their SQL/native slot is already gone.
		for _, change := range changes {
			change.abandoned = true
		}
		tx.mu.Lock()
		tx.relationSchemaRetirements = append(tx.relationSchemaRetirements, changes...)
		tx.relationSchemaChanges = tx.relationSchemaChanges[:keep]
		tx.mu.Unlock()
		return
	}
	retire := make(map[*table]bool)
	scopes := make(map[*relationScope]int)
	databases := make(map[*database][]*table)
	publish := make(map[*table]bool)
	for _, change := range changes {
		scopes[change.scope]++
		if change.created != nil {
			retire[change.created] = true
		}
		if !change.abandoned && change.before != nil && change.before.committed != nil {
			retire[change.before.committed] = commit
		}
	}
	if commit {
		for _, change := range changes {
			if change.abandoned || change.declaration != nil {
				continue
			}
			view := change.slot.view.Load()
			if view == nil {
				continue
			}
			current := view.visible(nil)
			if view.owner == tx {
				change.slot.view.Store(&relationSchemaView{committed: current})
			}
			if current != nil {
				retire[current] = false
				if current.ScopedSchema && current.CreatorTxID == tx.durableID() {
					if !publish[current] {
						databases[current.schema] = append(databases[current.schema], current)
						publish[current] = true
					}
				}
			}
		}
	}
	// A metadata publication owns the existing schema/MOSI write capability.
	for db, tables := range databases {
		access := db.GetWrite(nil)
		func() {
			defer access.Release()
			for _, table := range tables {
				current := db.tables.Get(table.Name)
				if current == nil || current.RuntimeIdentity != table.RuntimeIdentity {
					panic("relation schema decision lost its native generation")
				}
				current.CatalogVersion.Store(1)
				current.creator.Store(nil)
				table.CatalogVersion.Store(1)
				table.creator.Store(nil)
			}
			db.save()
		}()
	}
	for table, drop := range retire {
		if !drop {
			continue
		}
		if db := GetDatabase(table.schema.Name); db == table.schema {
			dropTableIdentity(db.Name, table.Name, true, table)
		}
		table.creator.Store(nil)
		if lifetime := table.relationLifetime; lifetime != nil {
			lifetime.mu.Lock()
			delete(lifetime.tables, table)
			lifetime.mu.Unlock()
		}
	}
	tx.mu.Lock()
	if keep == 0 {
		tx.relationSchemaChanges = nil
	} else {
		tx.relationSchemaChanges = tx.relationSchemaChanges[:keep]
	}
	tx.relationSchemaRetirements = nil
	tx.mu.Unlock()
	for scope, count := range scopes {
		scope.mu.Lock()
		scope.schemaDecisions -= count
		scope.mu.Unlock()
	}
}

func initRelationSchema(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "relation_scope_binding", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) != 3 {
			panic("relation_scope_binding requires scope, slot and native transaction")
		}
		scope := relationScopeFromValue(a[0])
		tx := scmerToTxContext(a[2])
		if !a[2].IsNil() && tx == nil {
			panic("relation schema lookup requires a native transaction capability or nil")
		}
		table := scope.schemaTable(a[1].String(), tx)
		if table == nil {
			return scm.NewNil()
		}
		access := table.schema.GetRead(tx)
		defer access.Release()
		table.NativeIdentity(tx)
		return scm.NewSlice([]scm.Scmer{scm.NewString(table.schema.Name), scm.NewString(table.clusterPublishedName())})
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Description: "resolves one native lexical relation slot in its actual transaction schema view", Params: []*scm.TypeDescriptor{{Kind: "any", Label: "scope"}, {Kind: "string", Label: "slot"}, {Kind: "any", Label: "transaction"}}, Return: &scm.TypeDescriptor{Kind: "list"}}})
	scm.Declare(en, &scm.Declaration{Name: "drop_relation_scope_binding", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) != 4 {
			panic("drop_relation_scope_binding requires scope, slot, if_exists and native transaction")
		}
		change, finish := relationScopeFromValue(a[0]).beginSchemaChange(a[1].String(), scmerToTxContext(a[3]), false, a[2].Bool())
		defer finish()
		return scm.NewBool(change != nil)
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Description: "stages removal of one native lexical relation slot in the actual transaction decision", Params: []*scm.TypeDescriptor{{Kind: "any", Label: "scope"}, {Kind: "string", Label: "slot"}, {Kind: "bool", Label: "if_exists"}, {Kind: "any", Label: "transaction"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
}
