/*
Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "sync"
import "github.com/launix-de/memcp/scm"

type systemTriggerRemoval struct{ schema, name string }

// The predecessor public producer persisted its FK helpers as hidden,
// non-system symbolic procedures. Recognize that finite old ABI only under an
// actual Foreign declaration; a name or builtin reference alone grants nothing.
// This reader does not execute, optimize or rewrite the retained procedure.
func legacyForeignTrigger(db *database, owner *table, fk foreignKey, trigger TriggerDescription) bool {
	if trigger.IsSystem || !trigger.Hidden || trigger.Priority != 0 || trigger.Async || trigger.SchemaOnly ||
		trigger.Source != "" || trigger.Language != "" || trigger.Acquire != nil || trigger.Release != nil || trigger.cacheTarget != nil ||
		!trigger.FuncPlan.IsNil() || !trigger.VectorFunc.IsNil() || fk.Updatemode > SETNULL || fk.Deletemode > SETNULL ||
		fk.Id == "" || len(fk.Cols1) == 0 || len(fk.Cols1) != len(fk.Cols2) {
		return false
	}
	value := trigger.Func.WithoutSourceInfo()
	if !value.IsProc() {
		return false
	}
	proc := value.Proc()
	params := proc.Params.WithoutSourceInfo()
	if (proc.En != nil && proc.En != &scm.Globalenv) || proc.NumberedOnly || (proc.NumVars != 0 && proc.NumVars != 4) ||
		!params.IsSlice() || len(params.Slice()) != 4 || proc.JITCode != 0 || proc.Compiled != nil {
		return false
	}
	for i, name := range []string{"OLD", "NEW", "session", "tx"} {
		if !params.Slice()[i].WithoutSourceInfo().SymbolEquals(name) {
			return false
		}
	}
	columns := fkQuotedList
	call := func(name string, arguments ...scm.Scmer) scm.Scmer {
		return scm.NewSlice(append([]scm.Scmer{scm.NewSymbol(name)}, arguments...))
	}
	prefix := "__fk_" + fk.Id + "_"
	var expected scm.Scmer
	switch {
	case owner.Name == fk.Tbl1 && ((trigger.Name == prefix+"child_insert" && trigger.Timing == BeforeInsert) ||
		(trigger.Name == prefix+"child_update" && trigger.Timing == BeforeUpdate)):
		expected = call("begin", call("__fk_check_ref", scm.NewString(db.Name), scm.NewString(fk.Tbl2),
			columns(fk.Cols2), fkValListExpr("NEW", fk.Cols1), scm.NewString(fk.Id)), scm.NewSymbol("NEW"))
	case owner.Name == fk.Tbl2 && trigger.Name == prefix+"parent_delete" && trigger.Timing == BeforeDelete:
		expected = call("__fk_on_parent_delete", scm.NewString(db.Name), scm.NewString(fk.Tbl1),
			columns(fk.Cols1), fkValListExpr("OLD", fk.Cols2), scm.NewString(fk.Id), scm.NewString(legacyForeignMode(fk.Deletemode)))
	case owner.Name == fk.Tbl2 && trigger.Name == prefix+"parent_update":
		timing := BeforeUpdate
		if fk.Updatemode != RESTRICT {
			timing = AfterUpdate
		}
		if trigger.Timing != timing {
			return false
		}
		expected = call("__fk_on_parent_update", scm.NewString(db.Name), scm.NewString(fk.Tbl1),
			columns(fk.Cols1), fkValListExpr("OLD", fk.Cols2), fkValListExpr("NEW", fk.Cols2),
			scm.NewString(fk.Id), scm.NewString(legacyForeignMode(fk.Updatemode)))
	default:
		return false
	}
	return matchLegacyForeignBody(proc.Body, expected)
}

// A role belongs to an actual native declaration and its exact endpoint, never
// to arbitrary trigger names which merely share a generated-looking prefix.
func foreignTriggerRole(owner *table, fk foreignKey, name string) bool {
	prefix := "__fk_" + fk.Id + "_"
	return (owner.Name == fk.Tbl1 && (name == prefix+"child_insert" || name == prefix+"child_update")) ||
		(owner.Name == fk.Tbl2 && (name == prefix+"parent_delete" || name == prefix+"parent_update"))
}

// Loader-only: all endpoint collisions are proved before any derived program
// is retired. Unknown user bodies remain untouched and block regeneration.
func (t *table) validateLoadedForeignTriggers(db *database, fk foreignKey) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, trigger := range t.Triggers {
		if foreignTriggerRole(t, fk, trigger.Name) && !trigger.IsSystem && !legacyForeignTrigger(db, t, fk, trigger) {
			panic("foreign declaration trigger name already exists: " + trigger.Name)
		}
	}
}

func (t *table) retireLoadedForeignTriggers(db *database, fk foreignKey) {
	t.mu.Lock()
	defer t.mu.Unlock()
	kept := t.Triggers[:0]
	for _, trigger := range t.Triggers {
		if foreignTriggerRole(t, fk, trigger.Name) && (trigger.IsSystem || legacyForeignTrigger(db, t, fk, trigger)) {
			continue
		}
		kept = append(kept, trigger)
	}
	clear(t.Triggers[len(kept):])
	t.Triggers = kept
	t.refreshTriggerMemoryLocked()
}

func legacyForeignMode(mode foreignKeyMode) string {
	switch mode {
	case CASCADE:
		return "CASCADE"
	case SETNULL:
		return "SETNULL"
	default:
		return "RESTRICT"
	}
}

// The expected finite grammar contains only literal strings, symbols and exact
// lists constructed above. Traverse that grammar, never arbitrary executable
// forms or callable values; extra expressions and numbered/captured values fail.
func matchLegacyForeignBody(actual, expected scm.Scmer) bool {
	actual, expected = actual.WithoutSourceInfo(), expected.WithoutSourceInfo()
	// Historical column arguments used both quoted literal lists and list
	// constructors. Admit only the exact strings from the native declaration;
	// this is a finite data spelling, never evaluation of retained syntax.
	if expected.IsSlice() && len(expected.Slice()) == 2 && expected.Slice()[0].SymbolEquals("quote") &&
		expected.Slice()[1].IsSlice() && actual.IsSlice() && len(actual.Slice()) > 0 && actual.Slice()[0].WithoutSourceInfo().SymbolEquals("list") {
		return matchLegacyForeignBody(scm.NewSlice(actual.Slice()[1:]), expected.Slice()[1])
	}
	if expected.IsString() {
		return actual.IsString() && actual.String() == expected.String()
	}
	if expected.IsSymbol() {
		return actual.IsSymbol() && actual.Symbol() == expected.Symbol()
	}
	if !expected.IsSlice() || !actual.IsSlice() || len(actual.Slice()) != len(expected.Slice()) {
		return false
	}
	for i, value := range expected.Slice() {
		if !matchLegacyForeignBody(actual.Slice()[i], value) {
			return false
		}
	}
	return true
}

// Canonical names identify a native column's generated recipe only after its
// actual system/timing owner is proved. A user name never grants this authority.
func (t *table) dependencyCleanupColumn(trigger TriggerDescription) (*column, bool) {
	if !trigger.IsSystem || trigger.Timing != AfterDropTable {
		return nil, false
	}
	for _, col := range t.Columns {
		ordered := trigger.Name == ".orcdep_cleanup:"+t.Name+":"+col.Name
		if ordered || trigger.Name == ".self_cleanup:"+t.Name+":"+col.Name {
			return col, ordered
		}
	}
	return nil, false
}

// Validate unpublished storage metadata, before a malformed historical program
// can be reached by an irreversible lifecycle operation. Value recipes remain
// authoritative when present; runtime-only caches admit only the finite format.
func (t *table) validateSystemCleanupFormats() error {
	for _, trigger := range t.Triggers {
		col, ordered := t.dependencyCleanupColumn(trigger)
		if col == nil {
			continue
		}
		definition := col.Computor
		if ordered {
			definition = col.OrcMapReduceFn
		}
		if !definition.IsNil() {
			continue
		}
		if trigger.Language != "" || trigger.Source != "" {
			return fmt.Errorf("native dependency cleanup cannot use a user trigger language")
		}
		if _, recognized := readSystemTriggerRemovals(trigger.Func); !recognized {
			return fmt.Errorf("unsupported persisted native dependency cleanup format")
		}
	}
	return nil
}

func compileSystemTriggerRemovals(removals []systemTriggerRemoval) scm.Scmer {
	calls := []scm.Scmer{scm.NewSymbol("begin")}
	for _, removal := range removals {
		calls = append(calls, scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("remove_system_trigger"), scm.NewString(removal.schema), scm.NewString(removal.name),
		}))
	}
	return buildFKProc(scm.NewSlice(calls))
}

// Read the two finite native cleanup formats. Some historical runtime caches
// intentionally persisted no value recipe, so their closed cleanup procedure
// is the only remaining ownership record. This reader never executes, rewrites
// or recursively walks arbitrary code/data. Its caller proves the actual
// IsSystem flag and canonical column cleanup owner before admitting this format.
func readSystemTriggerRemovals(value scm.Scmer) ([]systemTriggerRemoval, bool) {
	value = value.WithoutSourceInfo()
	if !value.IsProc() {
		return nil, false
	}
	proc := value.Proc()
	if proc.En != nil && proc.En != &scm.Globalenv {
		return nil, false
	}
	params := proc.Params.WithoutSourceInfo()
	if !params.IsSlice() || len(params.Slice()) != 4 || proc.NumberedOnly || (proc.NumVars != 0 && proc.NumVars != 4) {
		return nil, false
	}
	for i, name := range []string{"OLD", "NEW", "session", "tx"} {
		if !params.Slice()[i].WithoutSourceInfo().SymbolEquals(name) {
			return nil, false
		}
	}
	body := proc.Body.WithoutSourceInfo()
	if !body.IsSlice() || len(body.Slice()) == 0 || !callHeadIs(body.Slice()[0], "begin") {
		return nil, false
	}
	removals := make([]systemTriggerRemoval, 0, len(body.Slice())-1)
	format := ""
	for _, raw := range body.Slice()[1:] {
		call := raw.WithoutSourceInfo()
		if !call.IsSlice() {
			return nil, false
		}
		parts := call.Slice()
		kind := ""
		if len(parts) == 4 && callHeadIs(parts[0], "droptrigger") && parts[3].IsBool() && parts[3].Bool() {
			kind = "legacy-public-drop"
		} else if len(parts) == 3 && callHeadIs(parts[0], "remove_system_trigger") {
			kind = "native-system-remove"
		}
		if kind == "" || (format != "" && format != kind) || !parts[1].IsString() || !parts[2].IsString() {
			return nil, false
		}
		format = kind
		removals = append(removals, systemTriggerRemoval{parts[1].String(), parts[2].String()})
	}
	return removals, true
}

// A declaration consumer retains definitions through the existing transaction
// resource owner. No mutex is held while callbacks, rows or cache DDL execute.
type triggerDeclarationLease struct {
	table *table
	rows  scm.Scmer
	once  sync.Once
}

func (lease *triggerDeclarationLease) Close() error {
	lease.once.Do(func() {
		lease.table.mu.Lock()
		lease.table.triggerDeclarationReaders--
		lease.table.mu.Unlock()
	})
	return nil
}

// Callers hold t.mu. User-definition changes conflict before effects rather
// than waiting: a trigger callback may itself request DDL on the same table.
func (t *table) requireTriggerDeclarationMutationLocked() {
	if t.triggerDeclarationReaders != 0 {
		panic("native trigger declarations are in use; retry after the transaction finishes")
	}
}

func (t *table) requireUnleasedTriggerDeclarations() {
	if t.triggerDeclarationsInUse() {
		panic("native trigger declarations are in use; retry after the transaction finishes")
	}
}

func (t *table) triggerDeclarationsInUse() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.triggerDeclarationReaders != 0
}

// retainTriggerDeclarations returns metadata, never executable programs. The
// explicit transaction owns one lease per table; reader and ordinary DML paths
// do not register. A local counter cannot represent distributed authority.
func (t *table) retainTriggerDeclarations(tx *TxContext) scm.Scmer {
	if tx == nil || tx.transactionState == nil {
		panic("trigger declaration retention requires a native transaction")
	}
	key := fmt.Sprintf("native-trigger-declarations:%p", t)
	value := tx.getOrComputeResource(key, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		return scm.NewAny(t.borrowTriggerDeclarations(tx))
	}))
	return value.Any().(*triggerDeclarationLease).rows
}

// Ordinary producer admission borrows this SAME declaration counter only until
// its producer/drain/undo returns. It does not install a transaction resource
// that could block the caller's later, independent DDL statements.
func (t *table) borrowTriggerDeclarations(tx *TxContext) *triggerDeclarationLease {
	if tx.State != TxActive {
		panic("trigger declaration retention requires an active native transaction")
	}
	db := t.schema
	if db == nil || activeReplication(db) != nil || sharedClusterDatabase(db.Name) {
		panic("distributed trigger declaration retention is not supported")
	}
	if t.isEphemeralQueryTable() {
		panic("planner-owned cache trigger declarations cannot be retained")
	}
	db.persistenceLifecycle.RLock()
	defer db.persistenceLifecycle.RUnlock()
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	if db.tables.Get(t.clusterPublishedName()) != t {
		panic("trigger declarations belong to a retired native relation")
	}
	// Reuse the native generation and visibility proof; legacy IDs remain
	// absent instead of being allocated by a descriptive operation.
	t.NativeIdentity(tx)
	t.mu.Lock()
	defer t.mu.Unlock()
	rows := make([]scm.Scmer, 0, len(t.Triggers))
	for _, trigger := range t.visibleTriggerDescriptionsLocked(tx) {
		if trigger.IsSystem {
			continue
		}
		rows = append(rows, scm.NewSlice([]scm.Scmer{
			scm.NewString("name"), scm.NewString(trigger.Name),
			scm.NewString("timing"), scm.NewString(trigger.Timing.sourceName()),
			scm.NewString("events"), statementTriggerEventNames(trigger.StatementEvents),
			scm.NewString("hidden"), scm.NewBool(trigger.Hidden),
		}))
	}
	t.triggerDeclarationReaders++
	return &triggerDeclarationLease{table: t, rows: scm.NewSlice(rows)}
}

func initTriggerDeclarations(en *scm.Env) {
	initStatementTriggerDeclarations(en)
	scm.Declare(en, &scm.Declaration{
		Name: "remove_system_trigger",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			db := GetDatabase(scm.String(a[0]))
			return scm.NewBool(db != nil && db.removeSystemTrigger(scm.String(a[1])))
		},
		Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
			Description: "Removes only an engine-generated system trigger through its native declaration owner. Used by trusted generated dependency cleanup; ordinary droptrigger never removes system definitions.",
			Params:      []*scm.TypeDescriptor{{Kind: "string", Label: "schema"}, {Kind: "string", Label: "name"}},
			Return:      &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(en, &scm.Declaration{
		Name: "retain_table_trigger_declarations",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			return TableFromScmer(a[0]).retainTriggerDeclarations(scmerToTxContext(a[1]))
		},
		Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
			Description: "Retains the current non-system trigger definitions until the native transaction finishes and returns descriptive rows, including hidden definitions. Conflicting definition changes fail before effects. Local tables only; native row/system-trigger execution is unchanged.",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "any", Label: "transaction"},
			},
			Return: &scm.TypeDescriptor{Kind: "list"},
		},
	})
}
