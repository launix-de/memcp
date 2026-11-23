/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"
import "context"
import "unicode/utf8"
import "github.com/launix-de/memcp/scm"

// This capability belongs to the existing statement Undo/drain owner and one
// exact declaration construction. It is never exported to Scheme, installed on
// an ambient executor, or registered as an UPDATE effect. Only its internal
// parallel scan sink can borrow physical mutation admission during construction.
type nativeColumnInitializer struct {
	exclusiveProperties []string
	rowBased            bool
	generate            bool
	operation           *nativeMutationOperation
	target              *table
	column              *column
	generator           scm.Scmer
	lease               *tableWriteLease
	before              *tableColumnNamesSnapshot
	after               *tableColumnNamesSnapshot
}

// Caller holds the original state mutex. All compared pointers are immutable
// for this producer; release follows complete worker/reader drain.
func (initializer *nativeColumnInitializer) owns(tx *TxContext, db *database) bool {
	return initializer != nil && initializer.operation != nil && tx != nil &&
		initializer.operation.invocation == tx && initializer.operation.state == tx.transactionState &&
		initializer.target.schema == db && tx.Mode == TxACID && tx.State == TxActive &&
		tx.checkedMutation == initializer.operation && tx.mutationOperations == 1 &&
		initializer.operation.columnInitializer == initializer
}

type nativeColumnInitializationDefinition struct {
	column       *column
	hasDefault   bool
	fillExisting bool
	generator    scm.Scmer
	exclusive    []string
}

// Decode the finite declaration batch before creating its statement owner.
// This is declaration data, not SQL syntax or a relation-sized row collection.
func decodeNativeColumnInitialization(definitions scm.Scmer) []nativeColumnInitializationDefinition {
	result := make([]nativeColumnInitializationDefinition, 0)
	for _, value := range mustScmerSlice(definitions, "column declarations") {
		fields := mustScmerSlice(value, "column declaration")
		if len(fields) != 5 || !fields[0].IsString() || !fields[1].IsString() {
			panic("column initialization requires name, type, dimensions, options and initializer")
		}
		dimensions := make([]int, len(mustScmerSlice(fields[2], "dimensions")))
		for i, n := range fields[2].Slice() {
			if !n.IsInt() {
				panic("column dimension must be integer")
			}
			dimensions[i] = int(n.Int())
		}
		options := mustScmerSlice(fields[3], "column options")
		if len(options)%2 != 0 {
			panic("column attributes must be key/value pairs")
		}
		generator := fields[4]
		if !generator.IsNil() && !(generator.IsProc() || generator.IsNativeFunc() || generator.IsJIT()) {
			panic("row initializer requires an invocation callable")
		}
		columnOptions := make([]scm.Scmer, 0, len(options))
		var exclusive []string
		for i := 0; i < len(options); i += 2 {
			key := scm.String(options[i])
			if key == "exclusive_properties" {
				exclusive = scmerSliceToStrings(mustScmerSlice(options[i+1], "exclusive properties"))
				continue
			}
			switch key {
			case "null", "fill_existing", "name_ignorecase", "collate", "properties", "comment", "enum_values", "default", "default_name":
			case "default_expression", "default_deferred":
				if generator.IsNil() {
					panic("historical row initializer is required by this default recipe")
				}
			default:
				panic("column initialization requires ordinary stored declarations")
			}
			columnOptions = append(columnOptions, options[i], options[i+1])
		}
		d := decodeColumnDefinition(fields[0].String(), fields[1].String(), dimensions, columnOptions)
		c := d.column
		if c.Name == "" || c.IsTemp || c.AutoIncrement || len(d.unique) != 0 {
			panic("column initialization requires a fresh ordinary declaration")
		}
		if generator.IsNil() {
			// Exclusive metadata can accompany the existing realized NULL history.
			// Constant-default extents use a different lifetime and cannot carry it.
			if (len(exclusive) != 0 && d.hasDefault) || (!c.AllowNull && !d.hasDefault) {
				panic("column initialization requires a complete ordinary declaration")
			}
			if d.hasDefault && c.sanitizer != nil {
				c.Default = c.sanitizer(c.Default)
			}
		} else if c.DefaultExpression == "" || !c.Default.IsNil() {
			panic("row initializer requires an inert future default recipe")
		}
		if err := validateColumnInitialScalar(c.Default); err != nil {
			panic(err)
		}
		result = append(result, nativeColumnInitializationDefinition{c, d.hasDefault, d.fillExisting, generator, exclusive})
	}
	if len(result) == 0 {
		panic("column initialization requires a nonempty declaration batch")
	}
	return result
}

func (tx *TxContext) appendInitializedColumns(t *table, definitions []nativeColumnInitializationDefinition) bool {
	if tx == nil || tx.transactionState == nil || tx.Mode != TxACID || tx.State != TxActive || tx.mutationOperation != nil || tx.remoteObject != nil || tx.remoteInvocation || tx.coordinator != nil || tx.hasTableDrops() || tx.relationData != nil || t.Ephemeral || t.ScopedSchema || t.PersistencyMode == Cache || !localCatalogRowDatabase(t.schema) {
		panic("row-initialized append requires its original local ACID transaction")
	}
	if len(tx.publicColumnPropertyRemovals(tx.publicForeignAmendments())) != 0 {
		panic("row-initialized append cannot share unordered property retirement")
	}
	requireTableMaintenance(t.schema.Name, t.Name, maintenanceAlter)
	operation, err := newNativeMutationBinding(tx, tx.Session)
	if err != nil {
		panic(err)
	}
	defer func() {
		operation.mu.Lock()
		retired := operation.state == nil
		operation.mu.Unlock()
		if retired {
			tx.retireUnacceptedPublicCatalogPhase()
		}
	}()
	initializer := &nativeColumnInitializer{operation: operation, target: t}
	operation.columnInitializer = initializer
	result := operation.runProducer(func(binding *TxContext) scm.Scmer {
		// Undo precedes the FIRST column, row, generator, or private input in the
		// statement. No DML effect/transition/trigger program is registered here.
		operation.beginStatement(t.schema)
		operation.prepareStatement()
		finish := binding.beginCatalogConstruction(catalogValidationConstruction, initializer, t)
		defer finish()
		for _, d := range definitions {
			// Each preceding column's workers have drained before this cold setup.
			initializer.column, initializer.generator = d.column, d.generator
			initializer.rowBased, initializer.generate = !d.generator.IsNil() || !d.hasDefault, !d.generator.IsNil() && d.fillExisting
			initializer.exclusiveProperties = d.exclusive
			result := binding.stagePublicCatalogDeclaration(t.schema, catalogColumnKind, initializer, func(changes *publicCatalogChanges) bool {
				return changes.stageColumnAppend(binding, t, d.column, d.hasDefault, d.fillExisting, initializer)
			})
			if !result {
				panic(newColumnDeclarationConflict(t, d.column.Name))
			}
		}
		return scm.NewBool(true)
	})
	return result.Bool()
}

func initCatalogColumnInitialization(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "createcolumns", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) != 3 {
			panic("createcolumns expects a native table, declaration batch and original transaction")
		}
		tx := scmerToTxContext(a[2])
		if !a[2].IsNil() && tx == nil {
			panic("createcolumns requires a native transaction")
		}
		return scm.NewBool(tx.appendInitializedColumns(TableFromScmer(a[0]), decodeNativeColumnInitialization(a[1])))
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Description: "Atomically appends ordinary stored declarations using the original local ACID statement Undo. Optional zero-argument initializers run once per historical visible row outside owner locks; realized values use ordinary native row journals without DML events.", Params: []*scm.TypeDescriptor{{Kind: "table", Label: "table"}, {Kind: "list", Label: "definitions"}, {Kind: "any", Label: "tx"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
}

// This is the existing getWrite admission seam, with a single precise caller
// instead of an ordinary DML grant. Normal producers still pass nil and retain
// every constructing/statement/enforcement check.
func (initializer *nativeColumnInitializer) admitShard(s *storageShard, tx *TxContext) {
	scope := tx.resourceScope()
	scope.mu.Lock()
	tx.mu.Lock()
	valid := initializer.owns(tx, s.t.schema) && s.t == initializer.target &&
		!scope.closed && !scope.deciding && scope.constructing != 0 &&
		initializer.before != nil && initializer.after != nil
	tx.mu.Unlock()
	scope.mu.Unlock()
	if !valid {
		panic("physical column initializer is outside its construction")
	}
	lease := initializer.lease
	owner := &s.t.writeAdmission
	owner.mu.Lock()
	valid = lease != nil && lease.owner == owner && lease.transaction == tx.transactionState &&
		lease.declaration && lease.ddlTable.Load() == s.t && owner.declaration == tx.transactionState && owner.declarationGuardedInput
	owner.mu.Unlock()
	if !valid {
		panic("physical column initializer lost its actual declaration lease")
	}
	// Once at this actual native scan-input acquisition, before any row value.
	// Existing original-T overlay owns generation/WAL lifetime through decision.
	tx.mu.Lock()
	tx.getOrCreateShardTxLocked(s)
	tx.mu.Unlock()
}

func (initializer *nativeColumnInitializer) run() {
	t, tx := initializer.target, initializer.operation.invocation
	t.beginContributionMutation()
	defer t.endContributionMutation()
	// The existing table scan captures finite per-shard input highwaters,
	// respects original-T masks, schedules parallel shard workers and drains all
	// workers before returning a failure. No relation-sized Scheme row list.
	access := scanAccess{runtime: &scanAccessRuntime{initializer: initializer}}
	t.scanWithBatchFrom(tx, nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, access, nil, scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
		[]string{"$record_ref"}, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			if err := scanQueryContext(tx, context.Background()).Err(); err != nil {
				panic(err)
			}
			ref := recordRefFromScmer(values[1])
			// Scalar evaluation is outside physical/schema/amendment locks and
			// happens once for this exact historically visible source row.
			value := scm.NewNil()
			if initializer.generate {
				value = scm.Apply(initializer.generator)
			}
			if initializer.column.sanitizer != nil {
				value = initializer.column.sanitizer(value)
			}
			if err := validateColumnInitialScalar(value); err != nil {
				panic(err)
			}
			initializer.replace(ref, value)
			return scm.NewInt(values[0].Int() + 1)
		}), scm.NewInt(0), scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return scm.NewInt(values[0].Int() + values[1].Int()) }), false, 0, nil)
}

func (initializer *nativeColumnInitializer) replace(ref recMapTarget, value scm.Scmer) {
	tx := initializer.operation.invocation
	s := ref.shard
	if s == nil || s.t != initializer.target {
		panic("column initializer source generation changed")
	}
	initial := initializer.before.initialByShard[s.uuid]
	s.lockForMutation(tx, nil)
	defer s.mu.Unlock()
	locked := s.lockReplacementSuccessors(ref.recid, tx)
	defer unlockReplacementSuccessors(locked)
	columns := make([]string, 0, len(initializer.after.declarations))
	row := make([]scm.Scmer, 0, len(initializer.after.declarations))
	for _, c := range initializer.before.declarations {
		if isRuntimeComputedColumn(c) {
			continue
		}
		columns = append(columns, c.Name)
		row = append(row, s.rowValueByRecidLocked(ref.recid, c.Name, tx, initial))
	}
	columns, row = append(columns, initializer.column.Name), append(row, value)
	// Original-T masks and realized complete row payloads use the existing
	// physical journal/forwarding owner. No UPDATE/default/trigger execution.
	var barriers []replacementWALBarrier
	s.replacePreparedRowLocked(ref.recid, columns, row, tx, false, false, &barriers, locked, initializer.after)
}

func validateColumnRowInitialization(d catalogColumnAppend) error {
	r := d.RowInitialization
	if r == nil {
		if len(d.ExclusiveProperties) != 0 {
			return fmt.Errorf("exclusive append properties require realized row initialization")
		}
		return nil
	}
	if d.Initialization != nil || r.Table != d.Table || r.Identity != d.Identity || r.Column != d.Name || !r.Value.IsNil() || !utf8.ValidString(r.Expression) || !utf8.ValidString(r.Name) {
		return fmt.Errorf("invalid realized-row column initialization")
	}
	if err := validateColumnInitialScalar(r.Value); err != nil {
		return err
	}
	if r.Expression == "" && (!d.Type.Nullable || r.Name != "" || r.Deferred) {
		return fmt.Errorf("materialized NULL declaration has no future default recipe")
	}
	if r.Deferred && !r.Value.IsNil() {
		return fmt.Errorf("deferred column default cannot carry a scalar")
	}
	seen := make(map[string]bool, len(d.ExclusiveProperties))
	for _, key := range d.ExclusiveProperties {
		if key == "" || !utf8.ValidString(key) || seen[key] {
			return fmt.Errorf("invalid exclusive append property")
		}
		if _, exists := d.Properties[key]; !exists {
			return fmt.Errorf("exclusive append property has no declared value")
		}
		seen[key] = true
	}
	return nil
}

// The frontend supplies property keys/values and declaration eligibility. This
// native claim is table-scoped and atomic with the exact append predecessor.
func validateAppendedPropertyClaims(t *table, columns []*column, c *column, keys []string) *column {
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		value, present := c.DeclarationProperties[key]
		if key == "" || !utf8.ValidString(key) || seen[key] || !present {
			panic("invalid exclusive append property")
		}
		seen[key] = true
		bare := cloneStoredColumnDeclaration(c)
		bare.DeclarationProperties = maps.Clone(c.DeclarationProperties)
		delete(bare.DeclarationProperties, key)
		input := append(append([]*column(nil), columns...), bare)
		c = t.claimColumnProperty(input, len(columns), columnPropertyClaim{key, value, storedColumnTypeOf(c)})
	}
	return c
}

// A real failed native name admission is semantic data. SQL error numbers and
// statement policy remain with the frontend; evaluator/I/O failures retain
// their own opaque cause. Allocate one identity-stable diagnostic per failure.
type columnDeclarationConflict struct {
	message    string
	diagnostic scm.Scmer
}

func (failure *columnDeclarationConflict) Error() string        { return failure.message }
func (failure *columnDeclarationConflict) OriginalFailure() any { return failure.diagnostic }
func newColumnDeclarationConflict(t *table, name string) *columnDeclarationConflict {
	return &columnDeclarationConflict{message: fmt.Sprintf("column %s already exists in %s", name, t.Name), diagnostic: scm.NewSlice([]scm.Scmer{scm.NewString("column-declaration-conflict"), scm.NewString("name"), scm.NewString(t.Name), scm.NewString(name)})}
}

// Rewind can remove all target columns while an unrelated prefix retains this
// original phase. The existing retained DDL lease still owns that real input;
// close/reacquire would lose its generation lifetime. Derive baseline candidates
// once from actual lease ownership, then normal history vetoes remain dominant.
func (changes *publicCatalogChanges) retainedColumnInputTables() map[*table]struct{} {
	var result map[*table]struct{}
	for t, lease := range changes.leases {
		if lease == nil || lease.ddlTable.Load() != t {
			continue
		}
		if changes.owner == nil || t.schema != changes.db || GetDatabase(changes.db.Name) != changes.db || changes.db.tables.Get(t.Name) != t || t.RuntimeIdentity == "" {
			panic("retained column input lost its native generation")
		}
		owner := &t.writeAdmission
		owner.mu.Lock()
		valid := lease.owner == owner && lease.transaction == changes.owner && lease.declaration && owner.declaration == changes.owner
		owner.mu.Unlock()
		if !valid {
			panic("retained column input lost its original declaration lease")
		}
		if result == nil {
			result = make(map[*table]struct{})
		}
		result[t] = struct{}{}
	}
	return result
}

func (tx *TxContext) hasRowInitializedColumns() bool {
	for _, entry := range tx.publicForeignAmendments() {
		if a := entry.column; a != nil && a.appendDeclaration != nil && a.appendDeclaration.RowInitialization != nil && !tx.ownsTableDrop(a.table) {
			return true
		}
	}
	return false
}

// New declarations use an explicit realized-row domain, optionally composed
// with accepted tables/name actions; standalone V21 remains a read-only domain.
func (record *catalogDecisionRecord) hasRowInitializedAppends() bool {
	for _, d := range record.ColumnAppends {
		if d.RowInitialization != nil {
			return true
		}
	}
	return false
}
