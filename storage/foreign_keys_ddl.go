/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "sort"
import "slices"
import "github.com/launix-de/memcp/scm"

// Strict FK DDL uses existing table-lock publication and mutation guards. It
// never adds synchronization to a scan or mutation loop. Pending transactions
// and preprocessed writes must finish before their trigger contract can change.
func lockStrictForeignKeyTables(tables []*table, tx *TxContext) func() {
	ss := SessionStateFromTx(tx)
	if ss == nil {
		panic("validated foreign key DDL requires a query session")
	}
	tables = append([]*table(nil), tables...)
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })
	var unlocks []func()
	completed := false
	defer func() {
		if !completed {
			for i := len(unlocks) - 1; i >= 0; i-- {
				unlocks[i]()
			}
		}
	}()
	var previous *table
	for _, tbl := range tables {
		if tbl == previous {
			continue
		}
		previous = tbl
		unlocks = append(unlocks, acquireTableLock(tbl.schema.Name, tbl.Name, true, false, ss, querySeqFromTx(tx)))
		if tbl.contributionWriters.Load() != 0 {
			panic("foreign key DDL cannot overlap an active mutation batch")
		}
		for _, shard := range tbl.ActiveShards() {
			if shard != nil && shard.activeTransactions.Load() != 0 {
				panic("foreign key DDL cannot overlap an active table transaction")
			}
		}
	}
	completed = true
	return func() {
		for i := len(unlocks) - 1; i >= 0; i-- {
			unlocks[i]()
		}
	}
}

// Schema membership is locked by the caller. TryLock also avoids waiting for
// legacy DDL which may still acquire its table barrier before schemalock.
func tryStrictForeignKeyDDL(tables []*table) (func(), error) {
	tables = append([]*table(nil), tables...)
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })
	var locked []*table
	unlock := func() {
		for i := len(locked) - 1; i >= 0; i-- {
			locked[i].ddlMu.Unlock()
		}
	}
	for _, tbl := range tables {
		if len(locked) > 0 && locked[len(locked)-1] == tbl {
			continue
		}
		if !tbl.ddlMu.TryLock() {
			unlock()
			return nil, fmt.Errorf("foreign key DDL requires idle table metadata")
		}
		locked = append(locked, tbl)
	}
	return unlock, nil
}

func strictForeignKeyColumn(tbl *table, name string) *column {
	for _, col := range tbl.Columns {
		if col.Name == name {
			return col
		}
	}
	return nil
}

// Validate engine dependencies against exact column names already resolved by
// the frontend. Type/namespace policies are checked against the caller's schema
// revision; this mechanism checks real key handles and enforcement safety.
func validateStrictForeignKey(db *database, child, parent *table, fk *foreignKey) error {
	if parent == nil || child == parent {
		return fmt.Errorf("foreign keys require an existing distinct parent table")
	}
	if child.schema != parent.schema {
		return fmt.Errorf("cross-database foreign keys are unsupported")
	}
	if len(fk.Cols1) == 0 || len(fk.Cols1) != len(fk.Cols2) {
		return fmt.Errorf("foreign key column counts must match")
	}
	if fk.Id == "" {
		return fmt.Errorf("foreign dependency requires a nonempty handle")
	}
	for _, existing := range child.Foreign {
		if existing.Tbl1 == child.Name && existing.Id == fk.Id {
			return fmt.Errorf("foreign dependency handle already exists")
		}
	}
	seenChild, seenParent := map[string]bool{}, map[string]bool{}
	for i := range fk.Cols1 {
		left, right := strictForeignKeyColumn(child, fk.Cols1[i]), strictForeignKeyColumn(parent, fk.Cols2[i])
		if left == nil || right == nil {
			return fmt.Errorf("unknown foreign key column")
		}
		if seenChild[left.Name] || seenParent[right.Name] {
			return fmt.Errorf("duplicate foreign key column")
		}
		seenChild[left.Name], seenParent[right.Name] = true, true
		if left.IsTemp || right.IsTemp {
			return fmt.Errorf("temporary columns cannot be foreign dependencies")
		}
		if (fk.Updatemode == SETNULL || fk.Deletemode == SETNULL) && !left.AllowNull {
			return fmt.Errorf("SET NULL foreign key columns must be nullable")
		}
		fk.Cols1[i], fk.Cols2[i] = left.Name, right.Name
	}
	found := false
	for _, key := range parent.Unique {
		if slices.Equal(key.Cols, fk.Cols2) {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("foreign key target must match an ordered primary or unique key")
	}
	// Cascading cycles need a separate trigger/transaction proof. Reject them
	// instead of publishing metadata that cannot terminate reliably.
	if fk.Updatemode != RESTRICT || fk.Deletemode != RESTRICT {
		var visit func(*table, map[*table]bool) bool
		visit = func(tbl *table, seen map[*table]bool) bool {
			if tbl == child {
				return true
			}
			if tbl == nil || seen[tbl] {
				return false
			}
			seen[tbl] = true
			for _, existing := range tbl.Foreign {
				if existing.Tbl1 == tbl.Name && visit(db.tables.Get(existing.Tbl2), seen) {
					return true
				}
			}
			return false
		}
		if visit(parent, map[*table]bool{}) {
			return fmt.Errorf("cyclic cascading foreign keys are unsupported")
		}
	}
	fk.Tbl1, fk.Tbl2 = child.Name, parent.Name
	return nil
}

func strictForeignKeyChildEmpty(child *table, tx *TxContext) bool {
	child.ddlMu.RLock()
	defer child.ddlMu.RUnlock()
	topology := child.pinActiveTopology()
	defer topology.releaseOperation()
	for _, shard := range topology.shards {
		if shard == nil {
			continue
		}
		empty := func() bool {
			release := shard.GetRead(tx)
			defer release()
			shard.mu.RLock()
			defer shard.mu.RUnlock()
			return shard.main_count == 0 && len(shard.inserts) == 0 && shard.activeTransactions.Load() == 0
		}()
		if !empty {
			return false
		}
	}
	return true
}

func createStrictForeignKey(child, parent *table, fk foreignKey, tx *TxContext, publication scm.Scmer) bool {
	if child.schema != parent.schema {
		panic("cross-database foreign keys are unsupported")
	}
	if child == parent {
		panic("self-referencing foreign keys are unsupported")
	}
	requireTableMaintenance(child.schema.Name, child.Name, maintenanceAlter)
	requireTableMaintenance(parent.schema.Name, parent.Name, maintenanceAlter)
	fk.Strict = true
	db := child.schema
	unlockTables := lockStrictForeignKeyTables([]*table{child, parent}, tx)
	defer unlockTables()
	if !strictForeignKeyChildEmpty(child, tx) {
		panic("adding foreign keys to populated tables is unsupported")
	}
	db.schemalock.Lock()
	unlockDDL, err := tryStrictForeignKeyDDL([]*table{child, parent})
	if err != nil {
		db.schemalock.Unlock()
		panic(err)
	}
	defer unlockDDL()
	if err := db.checkMetadataPublicationLocked(publication); err != nil {
		db.schemalock.Unlock()
		panic(err)
	}
	if db.tables.Get(child.Name) != child || db.tables.Get(parent.Name) != parent {
		db.schemalock.Unlock()
		panic("foreign key DDL received a stale table handle")
	}
	if err := validateStrictForeignKey(db, child, parent, &fk); err != nil {
		db.schemalock.Unlock()
		panic(err)
	}
	programs := compileStrictForeignKeyTriggers(db, child, []*table{parent}, []foreignKey{fk})[0]
	child.Foreign = append(child.Foreign, fk)
	parent.Foreign = append(parent.Foreign, fk)
	publishStrictForeignKeyTriggers(programs)
	db.applyMetadataPublicationLocked(publication)
	db.saveLockedAndUnlock(schemaSaveModeForDurability(child.PersistencyMode == Safe || parent.PersistencyMode == Safe))
	return true
}

// Strict CREATE resolves and locks only already published parents. The child
// remains private until every reference has passed validation.
func strictCreateForeignKeyParents(db *database, child *table, tx *TxContext) ([]*table, func()) {
	var parents []*table
	for _, fk := range child.Foreign {
		name := fk.Tbl2
		parent := db.tables.Get(name)
		if parent == nil {
			panic("foreign key parent table does not exist")
		}
		if parent == nil || child.Name == name {
			panic("self references and forward-declared foreign keys are unsupported")
		}
		requireTableMaintenance(db.Name, parent.Name, maintenanceAlter)
		parents = append(parents, parent)
	}
	if len(parents) == 0 {
		return parents, func() {}
	}
	return parents, lockStrictForeignKeyTables(parents, tx)
}

func prepareStrictCreateForeignKeys(db *database, child *table, parents []*table) error {
	seen := map[string]bool{}
	for i := range child.Foreign {
		fk, parent := &child.Foreign[i], parents[i]
		if db.tables.Get(parent.Name) != parent {
			return fmt.Errorf("foreign key parent changed during CREATE TABLE")
		}
		name := fk.Id
		if seen[name] {
			return fmt.Errorf("duplicate foreign dependency handle")
		}
		seen[name] = true
		if err := validateStrictForeignKey(db, child, parent, fk); err != nil {
			return err
		}
	}
	return nil
}

// Compile every generated program before changing any table or trigger metadata.
func compileStrictForeignKeyTriggersForPair(db *database, child, parent *table, fk foreignKey) []foreignKeyTriggerProgram {
	programs := foreignKeyTriggerPrograms(db, child, parent, fk)
	for i := range programs {
		finalizeTriggerCompilation(&programs[i].trigger)
	}
	return programs
}
func compileStrictForeignKeyTriggers(db *database, child *table, parents []*table, keys []foreignKey) [][]foreignKeyTriggerProgram {
	defer func() {
		if failed := recover(); failed != nil {
			db.schemalock.Unlock()
			panic(failed)
		}
	}()
	programs := make([][]foreignKeyTriggerProgram, len(parents))
	for i, fk := range keys {
		programs[i] = compileStrictForeignKeyTriggersForPair(db, child, parents[i], fk)
	}
	return programs
}
func publishStrictForeignKeyTriggers(programs []foreignKeyTriggerProgram) {
	for _, program := range programs {
		program.table.mu.Lock()
		program.table.addTriggerLocked(program.trigger)
		program.table.mu.Unlock()
	}
}

func dropStrictForeignKey(child *table, name string, tx *TxContext, publication scm.Scmer) bool {
	db := child.schema
	requireTableMaintenance(db.Name, child.Name, maintenanceAlter)
	db.schemalock.Lock()
	var found foreignKey
	exists := false
	for _, fk := range child.Foreign {
		if fk.Tbl1 == child.Name && fk.Id == name {
			found, exists = fk, true
			break
		}
	}
	parent := db.tables.Get(found.Tbl2)
	db.schemalock.Unlock()
	if !exists {
		return false
	}
	if parent == nil || parent == child {
		panic("unsupported foreign key parent")
	}
	requireTableMaintenance(db.Name, parent.Name, maintenanceAlter)
	unlockTables := lockStrictForeignKeyTables([]*table{child, parent}, tx)
	defer unlockTables()
	db.schemalock.Lock()
	unlockDDL, err := tryStrictForeignKeyDDL([]*table{child, parent})
	if err != nil {
		db.schemalock.Unlock()
		panic(err)
	}
	defer unlockDDL()
	if err := db.checkMetadataPublicationLocked(publication); err != nil {
		db.schemalock.Unlock()
		panic(err)
	}
	if db.tables.Get(child.Name) != child || db.tables.Get(parent.Name) != parent {
		db.schemalock.Unlock()
		panic("foreign key DDL received a stale table handle")
	}
	exists = false
	for _, fk := range child.Foreign {
		if fk.Tbl1 == found.Tbl1 && fk.Tbl2 == found.Tbl2 && fk.Id == found.Id {
			exists = true
			break
		}
	}
	if !exists {
		db.schemalock.Unlock()
		return false
	}
	removeFKTriggers(child, parent, found)
	for _, tbl := range []*table{child, parent} {
		kept := make([]foreignKey, 0, len(tbl.Foreign))
		for _, fk := range tbl.Foreign {
			if fk.Tbl1 != found.Tbl1 || fk.Tbl2 != found.Tbl2 || fk.Id != found.Id {
				kept = append(kept, fk)
			}
		}
		tbl.Foreign = kept
	}
	db.applyMetadataPublicationLocked(publication)
	db.saveLockedAndUnlock(schemaSaveModeForDurability(child.PersistencyMode == Safe || parent.PersistencyMode == Safe))
	return true
}

// A referenced candidate key may be removed only if another real ordered
// candidate remains. The caller holds schemalock and the table DDL barrier.
func strictForeignKeyNeedsUnique(tbl *table, index int) bool {
	columns := tbl.Unique[index].Cols
	for _, fk := range tbl.Foreign {
		if !fk.Strict || fk.Tbl2 != tbl.Name || !slices.Equal(fk.Cols2, columns) {
			continue
		}
		replacement := false
		for i, key := range tbl.Unique {
			if i != index && slices.Equal(key.Cols, columns) {
				replacement = true
				break
			}
		}
		if !replacement {
			return true
		}
	}
	return false
}
