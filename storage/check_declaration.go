/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

// A check declaration is inert language source plus its native dependency
// contract. Programs and validation capabilities are derived, never decoded.
// The same immutable recipe is borrowed by its two native timing headers.
type checkDeclaration struct {
	Id               string
	Language         string
	Source           string
	Columns          []string
	payloadFootprint uint
}

func (definition *checkDeclaration) validate() error {
	if definition == nil || definition.Id == "" || definition.Language == "" || definition.Source == "" ||
		!utf8.ValidString(definition.Id) || !utf8.ValidString(definition.Language) || !utf8.ValidString(definition.Source) {
		return fmt.Errorf("invalid native check declaration")
	}
	seen := make(map[string]bool, len(definition.Columns))
	for _, name := range definition.Columns {
		if name == "" || !utf8.ValidString(name) || seen[name] {
			return fmt.Errorf("invalid native check dependency")
		}
		seen[name] = true
	}
	return nil
}

func (definition *checkDeclaration) measurePayload() {
	definition.payloadFootprint = uint(unsafe.Sizeof(*definition)) + uint(len(definition.Id)+len(definition.Language)+len(definition.Source))
	definition.payloadFootprint += uint(cap(definition.Columns)) * uint(unsafe.Sizeof(""))
	for _, name := range definition.Columns {
		definition.payloadFootprint += uint(len(name))
	}
}

func (definition *checkDeclaration) UnmarshalJSON(data []byte) error {
	type declaration checkDeclaration
	var decoded declaration
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*definition = checkDeclaration(decoded)
	if err := definition.validate(); err != nil {
		return err
	}
	definition.measurePayload()
	return nil
}

// Freeze only the recipe's actual owned strings/vector. Do not retain a caller
// list, procedure environment, table handle, row or session through metadata.
func ownCheckDeclaration(id, language, source string, columns []string) *checkDeclaration {
	definition := &checkDeclaration{Id: strings.Clone(id), Language: strings.Clone(language), Source: strings.Clone(source), Columns: make([]string, len(columns))}
	for i, name := range columns {
		definition.Columns[i] = strings.Clone(name)
	}
	if err := definition.validate(); err != nil {
		panic(err)
	}
	definition.measurePayload()
	return definition
}

func (t *table) refreshCheckMemoryLocked() {
	bytes := uint(cap(t.Checks)) * uint(unsafe.Sizeof((*checkDeclaration)(nil)))
	for _, definition := range t.Checks {
		bytes += definition.payloadFootprint
	}
	t.checkMemoryBytes.Store(uint64(bytes))
	t.hasCheckConstraints.Store(len(t.Checks) != 0)
}

func (t *table) checkColumnDependency(name string) *checkDeclaration {
	for _, definition := range t.Checks {
		if slices.Contains(definition.Columns, name) {
			return definition
		}
	}
	return nil
}

// The caller owns declaration admission or an unpublished loader. Column
// resolution is through the existing immutable native point directory.
func validateCheckColumns(t *table, definition *checkDeclaration) {
	if err := definition.validate(); err != nil {
		panic(err)
	}
	names := t.columnNamesSnapshot.Load()
	if names == nil {
		panic("check declaration requires published column metadata")
	}
	for _, name := range definition.Columns {
		resolved, exists := names.exact[name]
		if !exists || resolved.ordinal >= len(names.declarations) {
			panic("check declaration input does not exist: " + name)
		}
		column := names.declarations[resolved.ordinal]
		if column.IsTemp || isRuntimeComputedColumn(column) {
			panic("check declarations require ordinary stored input columns")
		}
	}
}

// The trusted source compiler receives native semantic data only. Defaults,
// statistics, physical storage and caller capabilities are not compile inputs.
func checkCompilerContext(t *table, definition *checkDeclaration) scm.Scmer {
	validateCheckColumns(t, definition)
	columns := make([]scm.Scmer, len(definition.Columns))
	for i, name := range definition.Columns {
		row := t.ShowColumn(name, false, nil)
		var fields []scm.Scmer
		for _, field := range []string{"Field", "Type", "RawType", "Dimensions", "Null", "Collation", "Computed", "EnumValues", "Properties"} {
			fields = append(fields, scm.NewString(field), foreignMetadataValue(row, field))
		}
		columns[i] = scm.NewSlice(fields)
	}
	var name string
	if published := t.publishedName.Load(); published != nil {
		name = *published
	} else {
		name = t.Name // unpublished constructors own the fallback
	}
	return scm.NewSlice([]scm.Scmer{
		scm.NewString("schema"), scm.NewString(t.schema.Name),
		scm.NewString("table"), scm.NewString(name),
		scm.NewString("name"), scm.NewString(definition.Id),
		scm.NewString("columns"), scm.NewSlice(columns),
		scm.NewString("schema_only"), scm.NewBool(true),
	})
}

func (definition *checkDeclaration) triggerName(timing TriggerTiming) string {
	if timing == BeforeInsert {
		return "__check_" + definition.Id + "_insert"
	}
	return "__check_" + definition.Id + "_update"
}

func checkTriggerHeader(definition *checkDeclaration, timing TriggerTiming) TriggerDescription {
	return TriggerDescription{Name: definition.triggerName(timing), Timing: timing, IsSystem: true,
		checkDefinition: definition, needsFinalInsertValues: timing == BeforeInsert, needsFinalUpdateValues: timing == BeforeUpdate}
}

func compileCheckProgram(t *table, definition *checkDeclaration) scm.Scmer {
	compiler := checkLanguageCompiler(definition.Language)
	value := scm.Apply(compiler, scm.NewString(definition.Source), checkCompilerContext(t, definition)).WithoutSourceInfo()
	predicate, plan := unwrapDeferredTriggerBody(value)
	if predicate.IsNil() && !plan.IsNil() {
		predicate = evaluateTriggerPlan(plan)
	}
	predicate = predicate.WithoutSourceInfo()
	if !predicate.IsProc() || !predicate.Proc().Params.WithoutSourceInfo().IsSlice() || len(predicate.Proc().Params.WithoutSourceInfo().Slice()) != 4 {
		panic("check source compiler must return a four-argument predicate procedure")
	}
	predicate = scm.CloseProcedure(predicate)
	params := []scm.Scmer{scm.NewSymbol("OLD"), scm.NewSymbol("NEW"), scm.NewSymbol("session"), scm.NewSymbol("tx")}
	invoke := scm.NewSlice(append([]scm.Scmer{predicate}, params...))
	body := scm.NewSlice([]scm.Scmer{scm.NewSymbol("__constraint_result"), scm.NewString(definition.Id), invoke})
	program := scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice(params), body})
	return scm.CompileJIT(scm.CloseProcedure(evaluateTriggerPlan(program)), true)
}

func buildCheckTriggers(t *table, definition *checkDeclaration) []foreignKeyTrigger {
	program := compileCheckProgram(t, definition)
	insert, update := checkTriggerHeader(definition, BeforeInsert), checkTriggerHeader(definition, BeforeUpdate)
	insert.Func, update.Func = program, program
	// Both timing headers borrow this SAME procedure. Only the insert header
	// owns its optional initial estimate; future JIT growth is outside the fact.
	insert.initialFKFootprint = &initialFKProgramFootprint{bytes: uint(unsafe.Sizeof(initialFKProgramFootprint{})) +
		uint(len(insert.Name)+len(update.Name)) + scm.ComputeSize(program) - uint(unsafe.Sizeof(scm.Scmer{}))}
	return []foreignKeyTrigger{{table: t, trigger: insert}, {table: t, trigger: update}}
}

// Source compilation occurs outside all table/schema/shard mutation locks.
// Publish both headers only while the exact same immutable recipe remains live.
func (t *table) compileCheckForUse(trigger TriggerDescription) TriggerDescription {
	programs := buildCheckTriggers(t, trigger.checkDefinition)
	t.mu.Lock()
	defer t.mu.Unlock()
	var current *TriggerDescription
	for i := range t.Triggers {
		candidate := &t.Triggers[i]
		if candidate.checkDefinition == trigger.checkDefinition && candidate.Timing == trigger.Timing {
			current = candidate
			break
		}
	}
	if current == nil {
		// A captured immutable operation list can finish with its own private
		// program after a later declaration replacement; do not republish it.
		for _, program := range programs {
			if program.trigger.Timing == trigger.Timing {
				return program.trigger
			}
		}
		panic("check validation lost its native timing")
	}
	if !current.Func.IsNil() {
		return *current
	}
	for i := range t.Triggers {
		candidate := &t.Triggers[i]
		if candidate.checkDefinition != trigger.checkDefinition {
			continue
		}
		for _, program := range programs {
			if candidate.Timing == program.trigger.Timing {
				*candidate = program.trigger
			}
		}
	}
	t.refreshTriggerMemoryLocked()
	return *current
}

// The loader validates all inert collisions before retiring ANY header. It
// never invokes a compiler or sizes a procedure under an older schema lock.
func rebuildCheckTriggersAfterLoad(db *database) {
	for _, t := range db.tables.GetAll() {
		if len(t.Checks) == 0 {
			continue
		}
		seen := make(map[string]bool, len(t.Checks))
		for _, definition := range t.Checks {
			validateCheckColumns(t, definition)
			key := strings.ToLower(definition.Id)
			if seen[key] {
				panic("duplicate native check declaration: " + definition.Id)
			}
			seen[key] = true
			for _, trigger := range t.Triggers {
				if trigger.Name != definition.triggerName(BeforeInsert) && trigger.Name != definition.triggerName(BeforeUpdate) {
					continue
				}
				if !inertCheckTrigger(definition, trigger) {
					panic("check declaration trigger name already exists: " + trigger.Name)
				}
			}
		}
	}
	for _, t := range db.tables.GetAll() {
		if len(t.Checks) == 0 {
			continue
		}
		t.mu.Lock()
		kept := t.Triggers[:0]
		for _, trigger := range t.Triggers {
			retire := false
			for _, definition := range t.Checks {
				retire = retire || inertCheckTrigger(definition, trigger)
			}
			if !retire {
				kept = append(kept, trigger)
			}
		}
		clear(t.Triggers[len(kept):])
		t.Triggers = kept
		for _, definition := range t.Checks {
			t.insertTriggerLocked(checkTriggerHeader(definition, BeforeInsert))
			t.insertTriggerLocked(checkTriggerHeader(definition, BeforeUpdate))
		}
		t.refreshTriggerMemoryLocked()
		t.refreshCheckMemoryLocked()
		t.mu.Unlock()
	}
}

func inertCheckTrigger(definition *checkDeclaration, trigger TriggerDescription) bool {
	return trigger.Name == definition.triggerName(trigger.Timing) &&
		(trigger.Timing == BeforeInsert || trigger.Timing == BeforeUpdate) && trigger.IsSystem &&
		trigger.Func.IsNil() && trigger.FuncPlan.IsNil() && trigger.VectorFunc.IsNil() &&
		trigger.Source == "" && trigger.Language == "" && !trigger.Hidden && !trigger.Async &&
		trigger.Priority == 0 && trigger.Acquire == nil && trigger.Release == nil
}

func initCheckBuiltins(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "createcheck", Fn: func(a ...scm.Scmer) scm.Scmer {
		if len(a) != 7 || !a[1].IsString() || !a[2].IsString() || !a[3].IsString() || !a[4].IsSlice() {
			panic("createcheck requires table, name, source language, source, columns, native transaction and invocation session")
		}
		columns := make([]string, len(a[4].Slice()))
		for i, name := range a[4].Slice() {
			if !name.IsString() {
				panic("check dependencies require string names")
			}
			columns[i] = name.String()
		}
		definition := ownCheckDeclaration(a[1].String(), a[2].String(), a[3].String(), columns)
		checkLanguageCompiler(definition.Language) // reject an ineligible role before catalog effects
		invocation := bindConstraintDefinition(scmerToTxContext(a[5]), a[6])
		defer invocation.finish()
		return scm.NewBool(invocation.binding.stageCatalogCheck(TableFromScmer(a[0]), definition))
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Description: "stages a historically validated native scalar constraint in the existing public declaration transaction",
		Params: []*scm.TypeDescriptor{{Kind: "table", Label: "table"}, {Kind: "string", Label: "name"}, {Kind: "string", Label: "language"}, {Kind: "string", Label: "source"}, {Kind: "list", Label: "columns"}, {Kind: "any", Label: "tx"}, {Kind: "any", Label: "session"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})

	scm.Declare(en, &scm.Declaration{Name: "__constraint_result", Fn: func(a ...scm.Scmer) scm.Scmer {
		if a[1].IsBool() && a[1].Bool() {
			return scm.NewNil()
		}
		if !a[1].IsBool() {
			panic("constraint predicate must return BOOL: " + a[0].String())
		}
		panic("check constraint " + a[0].String() + " failed")
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
		Params: []*scm.TypeDescriptor{{Kind: "string", Label: "name"}, {Kind: "any", Label: "result"}}, Return: &scm.TypeDescriptor{Kind: "nil"}}})
}

// The current native remote transaction view deliberately carries no session.
// Never substitute txSessionScmer's nil-returning fallback for the invocation
// input of a constraint. Until that execution binding has its own remote
// lifetime contract, CHECK mutation fails before defaults/user callbacks/rows.
func requireCheckInvocation(tx *TxContext) {
	if tx == nil || tx.Session.IsNil() {
		panic("check validation requires a bound native invocation session; remote execution is not supported")
	}
}

// This is a fail-closed diagnostic, not authority against concurrent mode
// publication. Operator setup borrows admitStandaloneMutation's actual mode pin;
// direct row callbacks must borrow that SAME operation's native lifetime. The
// callback admission/drain contract must close before this draft is usable.
func (t *table) requireCheckMutation(tx *TxContext, operation string) {
	if !t.hasCheckConstraints.Load() {
		return
	}
	if activeReplication(t.schema) != nil || sharedClusterDatabase(t.schema.Name) {
		panic("CHECK row mutation requires standalone native admission")
	}
	if tx == nil {
		panic("check-constrained " + operation + " requires a native statement transaction")
	}
	requireCheckInvocation(tx)
	if (tx.Mode != TxCursorStability && tx.Mode != TxACID) || tx.mutationOperation == nil {
		panic("check-constrained " + operation + " requires an outer native statement mutation producer")
	}
	tx.requireMutationBinding()
	owner := tx.mutationOperation
	owner.mu.Lock()
	if outer := owner.undoOwner; outer != nil {
		owner.mu.Unlock()
		owner = outer
		owner.mu.Lock()
	}
	defer owner.mu.Unlock()
	statement := owner.statement
	if statement == nil || !statement.ready || !statement.running || statement.finished || statement.database != t.schema || statement.tables[t] == nil {
		panic("check-constrained " + operation + " has no admitted statement undo owner")
	}
}

func (t *table) admitCheckMutation(tx *TxContext, operation string) {
	if !t.hasCheckConstraints.Load() {
		return
	}
	if tx != nil && tx.mutationOperation != nil {
		tx.mutationOperation.admitStatementTable(t)
	}
	t.requireCheckMutation(tx, operation)
	// The prepared outer statement already owns the typed, actual database mode
	// pin. Never create a resource/admission from a row callback under locks.
}

func checkConstraintMetadata(definitions []*checkDeclaration, pending bool) scm.Scmer {
	rows := make([]scm.Scmer, len(definitions))
	for i, definition := range definitions {
		fields := []scm.Scmer{scm.NewString("Id"), scm.NewString(definition.Id),
			scm.NewString("Columns"), showStringSlice(definition.Columns),
			scm.NewString("Language"), scm.NewString(definition.Language),
			scm.NewString("Source"), scm.NewString(definition.Source)}
		if pending {
			fields = append(fields, scm.NewString("Enabled"), scm.NewBool(true), scm.NewString("Trusted"), scm.NewBool(true))
		}
		rows[i] = scm.NewSlice(fields)
	}
	return scm.NewSlice(rows)
}

// Called outside shard.mu, after the same immutable declaration's sanitizers,
// and before the physical same-value skip. Final predicates cannot modify NEW.
// The SQL request's real transaction owns undo of all rows and trigger effects.
func (t *table) validateFinalUpdateRow(declarations []*column, oldRow, newRow dataset, validators []TriggerDescription, tx *TxContext) {
	if tx == nil {
		panic("check-constrained UPDATE requires a native statement transaction")
	}
	requireCheckInvocation(tx)
	columns := make([]string, len(declarations))
	for i, declaration := range declarations {
		columns[i] = declaration.Name
	}
	oldDict, newDict := t.rowToDictWithColumns(oldRow, columns), t.rowToDictWithColumns(newRow, columns)
	for _, validator := range validators {
		if !validator.needsFinalUpdateValues || validator.checkDefinition == nil || validator.Func.IsNil() {
			panic("native final UPDATE validator is unavailable")
		}
		scm.Apply(validator.Func, oldDict, newDict, txSessionScmer(tx), txContextScmer(tx))
	}
}
