/*
Copyright (C) 2025, 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/launix-de/memcp/scm"
)

// TriggerTiming defines when a trigger fires
type TriggerTiming uint8

// Preserve typed native failures through trigger diagnostics. A string panic
// remains a string; an error retains its cause and exactly the same visible text.
func triggerExecutionFailure(name string, timing TriggerTiming, tableName string, failure any) any {
	if cause, ok := failure.(error); ok {
		return fmt.Errorf("trigger %s (%s) on %s failed: %w", name, timing, tableName, cause)
	}
	return fmt.Sprintf("trigger %s (%s) on %s failed: %v", name, timing, tableName, failure)
}

const (
	BeforeInsert TriggerTiming = iota
	AfterInsert
	BeforeUpdate
	AfterUpdate
	BeforeDelete
	AfterDelete
	AfterDropTable
	AfterDropColumn
	AfterInvalidate // fired when a computed column is invalidated; propagates cache invalidation
	AfterCreateTable
	AfterStatement   // one declaration, with its explicit native statement event mask
	ReplaceStatement // owns candidate transitions in place of the native row operation
)

func (tt TriggerTiming) isStatement() bool {
	return tt == AfterStatement || tt == ReplaceStatement
}

func (tt TriggerTiming) String() string {
	switch tt {
	case BeforeInsert:
		return "BEFORE INSERT"
	case AfterInsert:
		return "AFTER INSERT"
	case BeforeUpdate:
		return "BEFORE UPDATE"
	case AfterUpdate:
		return "AFTER UPDATE"
	case BeforeDelete:
		return "BEFORE DELETE"
	case AfterDelete:
		return "AFTER DELETE"
	case AfterDropTable:
		return "AFTER DROP TABLE"
	case AfterDropColumn:
		return "AFTER DROP COLUMN"
	case AfterInvalidate:
		return "AFTER INVALIDATE"
	case AfterCreateTable:
		return "AFTER CREATE TABLE"
	case AfterStatement:
		return "AFTER STATEMENT"
	case ReplaceStatement:
		return "REPLACE STATEMENT"
	default:
		return "UNKNOWN"
	}
}

func (tt TriggerTiming) sourceName() string {
	switch tt {
	case BeforeInsert:
		return "before_insert"
	case AfterInsert:
		return "after_insert"
	case BeforeUpdate:
		return "before_update"
	case AfterUpdate:
		return "after_update"
	case BeforeDelete:
		return "before_delete"
	case AfterDelete:
		return "after_delete"
	case AfterDropTable:
		return "after_drop_table"
	case AfterDropColumn:
		return "after_drop_column"
	case AfterInvalidate:
		return "after_invalidate"
	case AfterCreateTable:
		return "after_create_table"
	case AfterStatement:
		return "after_statement"
	case ReplaceStatement:
		return "replace_statement"
	default:
		panic("unknown trigger timing")
	}
}

func (tt TriggerTiming) MarshalJSON() ([]byte, error) {
	if tt > ReplaceStatement {
		return nil, errors.New("unknown trigger timing")
	}
	return json.Marshal(tt.sourceName())
}

func (tt *TriggerTiming) UnmarshalJSON(data []byte) error {
	// Try string first (new format)
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		switch s {
		case "before_insert":
			*tt = BeforeInsert
		case "after_insert":
			*tt = AfterInsert
		case "before_update":
			*tt = BeforeUpdate
		case "after_update":
			*tt = AfterUpdate
		case "before_delete":
			*tt = BeforeDelete
		case "after_delete":
			*tt = AfterDelete
		case "after_drop_table":
			*tt = AfterDropTable
		case "after_drop_column":
			*tt = AfterDropColumn
		case "after_invalidate":
			*tt = AfterInvalidate
		case "after_create_table":
			*tt = AfterCreateTable
		case "after_statement":
			*tt = AfterStatement
		case "replace_statement":
			*tt = ReplaceStatement
		default:
			return errors.New("unknown trigger timing: " + s)
		}
		return nil
	}
	// Fall back to numeric (legacy format)
	var n uint8
	if err := json.Unmarshal(data, &n); err != nil {
		return errors.New("trigger timing must be string or number")
	}
	if n > uint8(ReplaceStatement) {
		return fmt.Errorf("unknown trigger timing number: %d", n)
	}
	*tt = TriggerTiming(n)
	return nil
}

// TriggerDescription holds all information about a trigger
type TriggerDescription struct {
	Name            string                `json:"name"`                       // Trigger name (user-defined or auto-generated)
	Timing          TriggerTiming         `json:"timing"`                     // BEFORE/AFTER INSERT/UPDATE/DELETE
	Func            scm.Scmer             `json:"func"`                       // The compiled trigger procedure; omitted when source is authoritative
	FuncPlan        scm.Scmer             `json:"-"`                          // Unevaluated lambda AST, compiled lazily on first use
	Source          string                `json:"source,omitempty"`           // Authoritative source code for persistent language-defined triggers
	SchemaOnly      bool                  `json:"schema_only,omitempty"`      // Imported source may access only its owning database
	Language        string                `json:"language,omitempty"`         // Name resolved through the process-local trigger compiler registry
	StatementEvents []string              `json:"statement_events,omitempty"` // One logical declaration, not one header per event
	IsSystem        bool                  `json:"is_system,omitempty"`        // True for Go-internal triggers (FK etc.) — not persisted via createtrigger
	Hidden          bool                  `json:"hidden,omitempty"`           // True for Scheme-internal triggers — persisted but hidden from SHOW TRIGGERS
	Priority        int                   `json:"priority,omitempty"`         // Execution order (lower = earlier)
	Async           bool                  `json:"async,omitempty"`            // Run trigger in background goroutine (fire-and-forget, no transaction context)
	VectorFunc      scm.Scmer             `json:"-"`                          // Vectorized trigger: (lambda (OLD_batch NEW_batch) ...) for batch execution
	Acquire         func(*TxContext) bool `json:"-"`                          // Optional lock-free pin for an ephemeral trigger target
	Release         func()                `json:"-"`                          // Releases a successful Acquire
	// Guarded by table.mu after publication. Generated dependency code restored
	// from a previous binary must be regenerated before its target is rebound.
	cacheTarget        *table // runtime identity, protected by the source table mutex
	needsRegeneration  bool
	initialFKFootprint *initialFKProgramFootprint // optional finite producer estimate, never persisted
	// Authoritative native producers alone may require final INSERT values.
	// This validation capability is regenerated, never a durable/user option.
	needsFinalInsertValues bool `json:"-"`
	needsFinalUpdateValues bool `json:"-"`
	checkDefinition        *checkDeclaration
	statementDefinition    *statementTriggerDefinition
}

// cacheTriggerSource is an immutable reverse edge for a runtime cache target.
// Trigger registration already runs outside row loops. Publishing an edge must
// not lock the target while the source table mutex is held.
type cacheTriggerSource struct {
	table *table
	next  *cacheTriggerSource
}

func (t *table) rememberCacheTriggerSource(source *table) {
	if t == nil || !t.isEphemeralQueryTable() || source == t {
		return
	}
	for {
		head := t.cacheTriggerSources.Load()
		for edge := head; edge != nil; edge = edge.next {
			if edge.table == source {
				return
			}
		}
		if t.cacheTriggerSources.CompareAndSwap(head, &cacheTriggerSource{source, head}) {
			return
		}
	}
}

func acquireCacheUse(users *int64) bool {
	for {
		current := atomic.LoadInt64(users)
		if current < 0 {
			return false
		}
		if atomic.CompareAndSwapInt64(users, current, current+1) {
			return true
		}
	}
}

func beginCacheEviction(users *int64) bool {
	return atomic.CompareAndSwapInt64(users, 0, -1)
}

func (t *table) acquireCacheUse() bool                       { return acquireCacheUse(&t.cacheUsers) }
func (t *table) acquireCacheUseForTrigger(_ *TxContext) bool { return t.acquireCacheUse() }
func (t *table) releaseCacheUse()                            { atomic.AddInt64(&t.cacheUsers, -1) }
func (t *table) beginCacheEviction() bool {
	return beginCacheEviction(&t.cacheUsers)
}
func (c *column) acquireCacheUse() bool { return acquireCacheUse(&c.cacheUsers) }
func (c *column) releaseCacheUse()      { atomic.AddInt64(&c.cacheUsers, -1) }
func (c *column) beginCacheEviction() bool {
	return beginCacheEviction(&c.cacheUsers)
}

func (t *table) acquireColumnCacheUse(c *column) bool {
	if !t.acquireCacheUse() {
		return false
	}
	if !c.acquireCacheUse() {
		t.releaseCacheUse()
		return false
	}
	return true
}

func (t *table) releaseColumnCacheUse(c *column) {
	c.releaseCacheUse()
	t.releaseCacheUse()
}

func (tr TriggerDescription) acquireTarget(tx *TxContext) bool {
	return tr.Acquire == nil || tr.Acquire(tx)
}

func (tr TriggerDescription) releaseTarget() {
	if tr.Release != nil {
		tr.Release()
	}
}

type persistedTriggerDescription struct {
	Name            string        `json:"name"`
	Timing          TriggerTiming `json:"timing"`
	Func            *scm.Scmer    `json:"func,omitempty"`
	Source          string        `json:"source,omitempty"`
	Language        string        `json:"language,omitempty"`
	StatementEvents []string      `json:"statement_events,omitempty"`
	SchemaOnly      bool          `json:"schema_only,omitempty"`
	LegacySourceSQL string        `json:"source_sql,omitempty"`
	IsSystem        bool          `json:"is_system,omitempty"`
	Hidden          bool          `json:"hidden,omitempty"`
	Priority        int           `json:"priority,omitempty"`
	Async           bool          `json:"async,omitempty"`
	RequiresTarget  bool          `json:"requires_target,omitempty"`
}

func (tr TriggerDescription) MarshalJSON() ([]byte, error) {
	persist := persistedTriggerDescription{
		Name:            tr.Name,
		Timing:          tr.Timing,
		Source:          tr.Source,
		Language:        tr.Language,
		StatementEvents: tr.StatementEvents,
		SchemaOnly:      tr.SchemaOnly,
		IsSystem:        tr.IsSystem,
		Hidden:          tr.Hidden,
		Priority:        tr.Priority,
		Async:           tr.Async,
		RequiresTarget:  tr.Acquire != nil,
	}
	// Source plus language is the durable definition. Compiled Scheme is a
	// process-local artifact and is persisted only for legacy/internal triggers
	// without source. FK callbacks are generated from FK metadata instead.
	regeneratedFK := tr.IsSystem && strings.HasPrefix(tr.Name, "__fk_")
	if tr.Language == "" && !regeneratedFK && tr.checkDefinition == nil {
		fn := tr.Func
		persist.Func = &fn
	}
	return json.Marshal(persist)
}

func (tr *TriggerDescription) UnmarshalJSON(data []byte) error {
	var persist persistedTriggerDescription
	if err := json.Unmarshal(data, &persist); err != nil {
		return err
	}
	tr.initialFKFootprint = nil
	tr.Name = persist.Name
	tr.Timing = persist.Timing
	tr.Source = persist.Source
	tr.Language = persist.Language
	tr.StatementEvents = persist.StatementEvents
	tr.SchemaOnly = persist.SchemaOnly
	// Schemas written before trigger-language registration stored SQL source in
	// source_sql. Import it into the generic representation; new saves emit only
	// source and language.
	if tr.Source == "" && persist.LegacySourceSQL != "" {
		tr.Source = persist.LegacySourceSQL
		tr.Language = "sql"
	}
	tr.IsSystem = persist.IsSystem
	tr.Hidden = persist.Hidden
	tr.Priority = persist.Priority
	tr.needsRegeneration = strings.HasPrefix(tr.Name, ".cache:") || strings.HasPrefix(tr.Name, ".orcdep:")
	tr.Async = persist.Async
	tr.FuncPlan = scm.NewNil()
	tr.VectorFunc = scm.NewNil()
	tr.needsFinalInsertValues = false
	tr.needsFinalUpdateValues = false
	tr.checkDefinition = nil
	tr.statementDefinition = nil
	// Cache-maintenance triggers carry a process-local target pin. The function
	// cannot be serialized, and its table or column may no longer exist after a
	// restart. Keep the trigger inert until cache preparation finds/recreates the
	// canonical target and rebinds it with SetTriggerTarget.
	if persist.RequiresTarget || restoredTriggerRequiresRuntimeTarget(tr.Name) {
		tr.Acquire = unavailableTriggerTarget
	}
	if persist.Func != nil {
		tr.Func = *persist.Func
	} else {
		tr.Func = scm.NewNil()
	}
	if tr.Timing.isStatement() {
		if err := validateStatementTrigger(*tr); err != nil {
			return err
		}
		*tr = newStatementTriggerDefinition(*tr).triggerHeader()
	} else if len(tr.StatementEvents) != 0 {
		return errors.New("row trigger cannot declare statement events")
	}
	return nil
}

func unavailableTriggerTarget(_ *TxContext) bool { return false }

func restoredTriggerRequiresRuntimeTarget(name string) bool {
	// Older schema files predate requires_target. These are all current
	// internal trigger families whose bodies address an evictable cache target.
	return strings.HasPrefix(name, ".kt_cleanup:") ||
		strings.HasPrefix(name, ".cache:") ||
		strings.HasPrefix(name, ".orcdep:") ||
		strings.HasPrefix(name, ".prejoin:")
}

func triggerScmerMissing(v scm.Scmer) bool {
	return v.IsNil()
}

// Source roles belong to the trusted compiler registration, never persisted
// schema/user options. A row predicate is a closed, row-local scalar routine;
// its four invocation inputs and BOOL/NIL result do not permit trigger DML.
type triggerLanguageDefinition struct {
	compiler        scm.Scmer
	replicationSafe bool
	rowPredicate    bool
	statementBody   bool
}

var triggerLanguageCompilers = struct {
	sync.RWMutex
	byName map[string]triggerLanguageDefinition
}{byName: make(map[string]triggerLanguageDefinition)}

func registerTriggerLanguage(language string, compiler scm.Scmer, roles ...bool) {
	if language == "" {
		panic("trigger language must not be empty")
	}
	if compiler.IsNil() {
		panic("trigger language compiler must not be empty")
	}
	definition := triggerLanguageDefinition{compiler: compiler,
		replicationSafe: len(roles) > 0 && roles[0], rowPredicate: len(roles) > 1 && roles[1], statementBody: len(roles) > 2 && roles[2]}
	triggerLanguageCompilers.Lock()
	triggerLanguageCompilers.byName[language] = definition
	triggerLanguageCompilers.Unlock()
}

func triggerLanguageDefinitionFor(language string) (triggerLanguageDefinition, bool) {
	triggerLanguageCompilers.RLock()
	definition, ok := triggerLanguageCompilers.byName[language]
	triggerLanguageCompilers.RUnlock()
	return definition, ok
}

// Only trusted startup code may opt a compiler into slave uploads. Such
// compilers must honor schema_only on imported definitions.
func replicationSafeLanguage(language string) bool {
	definition, _ := triggerLanguageDefinitionFor(language)
	return definition.replicationSafe
}

func triggerLanguageCompiler(language string) (scm.Scmer, bool) {
	definition, ok := triggerLanguageDefinitionFor(language)
	return definition.compiler, ok
}

func checkLanguageCompiler(language string) scm.Scmer {
	// Role and callable are copied from ONE registry publication. A concurrent
	// administrative replacement cannot pair a prior role with a different body.
	definition, ok := triggerLanguageDefinitionFor(language)
	if !ok || !definition.rowPredicate {
		panic("check requires a registered row-predicate source language: " + language)
	}
	return definition.compiler
}

func triggerCompilerContext(schemaName, tableName string, trigger *TriggerDescription) scm.Scmer {
	context := scm.NewFastDictValue(6)
	context.Set(scm.NewString("schema"), scm.NewString(schemaName), nil)
	context.Set(scm.NewString("table"), scm.NewString(tableName), nil)
	context.Set(scm.NewString("name"), scm.NewString(trigger.Name), nil)
	context.Set(scm.NewString("timing"), scm.NewString(trigger.Timing.sourceName()), nil)
	context.Set(scm.NewString("hidden"), scm.NewBool(trigger.Hidden), nil)
	context.Set(scm.NewString("schema_only"), scm.NewBool(trigger.SchemaOnly), nil)
	return scm.NewFastDict(context)
}

func loadPersistedTriggerPlan(schemaName, tableName string, trigger *TriggerDescription) {
	if trigger.Timing.isStatement() {
		return // event-specific programs belong to the immutable statement definition
	}
	if !triggerScmerMissing(trigger.Func) || !triggerScmerMissing(trigger.FuncPlan) || trigger.Language == "" {
		return
	}
	compiler, ok := triggerLanguageCompiler(trigger.Language)
	if !ok {
		panic(fmt.Sprintf("trigger %s.%s:%s requires unregistered language %q", schemaName, tableName, trigger.Name, trigger.Language))
	}
	compiled := scm.Apply(compiler, scm.NewString(trigger.Source), triggerCompilerContext(schemaName, tableName, trigger))
	trigger.Func, trigger.FuncPlan = unwrapDeferredTriggerBody(compiled)
	if triggerScmerMissing(trigger.Func) && triggerScmerMissing(trigger.FuncPlan) {
		panic(fmt.Sprintf("trigger language %q returned an empty trigger for %s.%s:%s", trigger.Language, schemaName, tableName, trigger.Name))
	}
}

func unwrapDeferredTriggerBody(body scm.Scmer) (scm.Scmer, scm.Scmer) {
	if !body.IsSlice() {
		return body, scm.NewNil()
	}
	items := body.Slice()
	if len(items) == 2 && scm.String(items[0]) == "quote" {
		return unwrapDeferredTriggerBody(items[1])
	}
	if len(items) != 2 || scm.String(items[0]) != "deferred_trigger" {
		return body, scm.NewNil()
	}
	return scm.NewNil(), items[1]
}

func finalizeTriggerCompilation(trigger *TriggerDescription) {
	if triggerScmerMissing(trigger.Func) {
		return
	}
	// Vectorization inspects the retained Scheme body, so it must run before
	// native compilation. Recursive compilation is intentional: maintenance
	// triggers spend their time in filter/map/reduce callbacks, while the outer
	// procedure is only orchestration. Apply dispatches directly to native code;
	// unsupported bodies and non-JIT builds retain the interpreter fallback.
	if (trigger.IsSystem || trigger.Hidden || trigger.Language == "") && trigger.VectorFunc.IsNil() {
		if vf := VectorizeTrigger(trigger.Func); !vf.IsNil() {
			trigger.VectorFunc = vf
		}
	}
	trigger.Func = scm.CompileJIT(scm.CloseProcedure(trigger.Func), true)
}

func evaluateTriggerPlan(plan scm.Scmer) scm.Scmer {
	// JIT consumes optimizer-normalized procedures. This is the same phase order
	// used for cached SQL query plans and is required for numbered mutable locals
	// in trigger sequences such as SET followed by a conditional SET. Optimize
	// rewrites AST containers in place, while CREATE TRIGGER plans are cacheable
	// and may install the same source repeatedly. Transfer a private copy so one
	// trigger's local-variable numbering never corrupts the cached plan.
	ownedPlan := scm.CloneOptimizerExpression(plan)
	return scm.Eval(scm.Optimize(ownedPlan, &scm.Globalenv, nil), &scm.Globalenv)
}

func compileTriggerForUse(schemaName, tableName string, trigger *TriggerDescription) {
	if !triggerScmerMissing(trigger.Func) {
		finalizeTriggerCompilation(trigger)
		return
	}
	if !triggerScmerMissing(trigger.FuncPlan) {
		trigger.Func = evaluateTriggerPlan(trigger.FuncPlan)
		trigger.FuncPlan = scm.NewNil()
		finalizeTriggerCompilation(trigger)
		return
	}
	loadPersistedTriggerPlan(schemaName, tableName, trigger)
	if !triggerScmerMissing(trigger.FuncPlan) {
		trigger.Func = evaluateTriggerPlan(trigger.FuncPlan)
		trigger.FuncPlan = scm.NewNil()
	}
	finalizeTriggerCompilation(trigger)
}

// GetTriggers returns all triggers for a specific timing
func (t *table) GetTriggers(timing TriggerTiming) []TriggerDescription {
	t.mu.Lock()
	result := make([]TriggerDescription, 0, len(t.Triggers))
	type lazyTrigger struct {
		triggerIndex   int
		resultIndex    int
		cleanupColumn  *column
		orderedCleanup bool
	}
	lazy := make([]lazyTrigger, 0)
	for i, tr := range t.Triggers {
		if tr.Timing != timing {
			continue
		}
		result = append(result, tr)
		if col, ordered := t.dependencyCleanupColumn(tr); col != nil {
			lazy = append(lazy, lazyTrigger{triggerIndex: i, resultIndex: len(result) - 1,
				cleanupColumn: col, orderedCleanup: ordered})
			continue
		}
		if triggerScmerMissing(tr.Func) && (tr.checkDefinition != nil || !triggerScmerMissing(tr.FuncPlan) || tr.Language != "") {
			lazy = append(lazy, lazyTrigger{triggerIndex: i, resultIndex: len(result) - 1})
		}
	}
	t.mu.Unlock()
	for _, item := range lazy {
		tr := result[item.resultIndex]
		if tr.checkDefinition != nil {
			result[item.resultIndex] = t.compileCheckForUse(tr)
			continue
		}
		if item.cleanupColumn != nil {
			definition := item.cleanupColumn.Computor
			if item.orderedCleanup {
				definition = item.cleanupColumn.OrcMapReduceFn
			}
			if !definition.IsNil() {
				refs := extractScanJoinInfo(definition)
				if !item.orderedCleanup {
					for i := 0; i+1 < len(item.cleanupColumn.ComputorDependencies); i += 2 {
						refs = append(refs, scanJoinInfo{schema: item.cleanupColumn.ComputorDependencies[i],
							table: item.cleanupColumn.ComputorDependencies[i+1], unknownReads: true})
					}
				}
				tr.Func = t.dependencyCleanupProgram(item.cleanupColumn.Name, refs, item.orderedCleanup)
			} else {
				removals, recognized := readSystemTriggerRemovals(tr.Func)
				if !recognized {
					panic("unsupported persisted native dependency cleanup format")
				}
				tr.Func = compileSystemTriggerRemovals(removals)
			}
			tr.FuncPlan, tr.VectorFunc = scm.NewNil(), scm.NewNil()
		} else {
			compileTriggerForUse(t.schema.Name, t.Name, &tr)
		}
		tr.initialFKFootprint = nil
		result[item.resultIndex] = tr

		t.mu.Lock()
		if item.triggerIndex < len(t.Triggers) {
			current := &t.Triggers[item.triggerIndex]
			if current.Name == tr.Name && current.Timing == tr.Timing && triggerScmerMissing(current.Func) {
				current.Func = tr.Func
				current.FuncPlan = tr.FuncPlan
				current.VectorFunc = tr.VectorFunc
				// A lazy replacement is not the finite FK producer described by
				// an old optional descriptor. Ordinary callback sizes stay unknown.
				current.initialFKFootprint = nil
				t.refreshTriggerMemoryLocked()
			}
		}
		t.mu.Unlock()
	}
	return result
}

// AddTrigger adds a trigger to the table. Automatically attempts to vectorize
// the trigger for batch execution (DELETE/INSERT patterns on prejoin tables).
func (t *table) AddTrigger(trigger TriggerDescription) {
	if trigger.Timing.isStatement() {
		panic("statement trigger definitions require their native catalog owner")
	}
	// Auto-vectorize: try to produce a batch-aware version of the trigger
	// Contract: only internal/system triggers participate in automatic
	// vectorization. User-facing SQL triggers may have very large bodies;
	// walking their full AST during CREATE TRIGGER would make schema setup
	// disproportionately expensive without improving correctness.
	finalizeTriggerCompilation(&trigger)
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, old := range t.Triggers {
		if old.Name == trigger.Name && old.IsSystem != trigger.IsSystem {
			panic("user and system trigger definitions cannot replace one another")
		}
	}
	t.addTriggerLocked(trigger)
}

// replaceTrigger publishes one already-compiled definition atomically. A
// conflicting reader must not observe removal of the preceding definition.
func (t *table) replaceTrigger(trigger TriggerDescription) {
	t.mu.Lock()
	defer t.mu.Unlock()
	userDefinition := !trigger.IsSystem
	for _, old := range t.Triggers {
		if old.Name == trigger.Name && (old.Timing.isStatement() || trigger.Timing.isStatement()) {
			panic("statement trigger definitions require their native catalog owner")
		}
		if old.Name == trigger.Name && old.IsSystem && !trigger.IsSystem {
			panic("system trigger definitions are owned by the storage engine")
		}
		if old.Name == trigger.Name && !old.IsSystem {
			userDefinition = true
		}
	}
	release := t.admitTriggerMutationLocked(trigger.Name, !userDefinition)
	defer release()
	for i, old := range t.Triggers {
		if old.Name == trigger.Name {
			copy(t.Triggers[i:], t.Triggers[i+1:])
			clear(t.Triggers[len(t.Triggers)-1:])
			t.Triggers = t.Triggers[:len(t.Triggers)-1]
			break
		}
	}
	t.insertTriggerLocked(trigger)
}

// addTriggerLocked inserts a compiled trigger while the caller holds t.mu.
func (t *table) addTriggerLocked(trigger TriggerDescription) {
	release := t.admitTriggerMutationLocked(trigger.Name, trigger.IsSystem)
	defer release()
	t.insertTriggerLocked(trigger)
}

// insertTriggerLocked is the non-failing list publication primitive. Ordinary
// publishers admit the mutation first; an accepted FK decision already owns
// its exact names through the native declaration lease.
func (t *table) insertTriggerLocked(trigger TriggerDescription) {
	trigger.cacheTarget.rememberCacheTriggerSource(t)
	// Keep trigger list ordered by priority (lower = earlier). For equal
	// priorities preserve registration order by inserting after existing ties.
	insertAt := len(t.Triggers)
	for i, tr := range t.Triggers {
		if tr.Priority > trigger.Priority {
			insertAt = i
			break
		}
	}
	t.Triggers = append(t.Triggers, TriggerDescription{})
	copy(t.Triggers[insertAt+1:], t.Triggers[insertAt:])
	t.Triggers[insertAt] = trigger
	t.refreshTriggerMemoryLocked()
}

// RemoveTrigger removes a trigger by name
func (t *table) RemoveTrigger(name string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.removeTriggerLocked(name, nil)
}

// A nil kind is trusted native cleanup. Public and generated-code entrypoints
// select their actual declaration owner while holding the same table mutex.
func (t *table) removeTriggerLocked(name string, system *bool) bool {
	for i, tr := range t.Triggers {
		if tr.Name == name {
			if tr.Timing.isStatement() {
				panic("statement trigger definitions require their native catalog owner")
			}
			if system != nil && tr.IsSystem != *system {
				if !*system {
					panic("system trigger definitions are owned by the storage engine")
				}
				return false
			}
			release := t.admitTriggerMutationLocked(name, tr.IsSystem)
			defer release()
			copy(t.Triggers[i:], t.Triggers[i+1:])
			clear(t.Triggers[len(t.Triggers)-1:])
			t.Triggers = t.Triggers[:len(t.Triggers)-1]
			t.refreshTriggerMemoryLocked()
			return true
		}
	}
	return false
}

// dropTrigger follows the same table-DDL-before-schema lock order as trigger
// creation. Tables may disappear between the catalog snapshot and locking, so
// each candidate is revalidated under schemalock before it is modified.
func (db *database) dropTrigger(name string) bool {
	return db.removeTriggerDefinition(name, false)
}

func (db *database) removeSystemTrigger(name string) bool {
	return db.removeTriggerDefinition(name, true)
}

func (db *database) removeTriggerDefinition(name string, system bool) bool {
	db.schemalock.RLock()
	tables := db.tables.GetAll()
	db.schemalock.RUnlock()

	for _, t := range tables {
		// Explicit DROP TRIGGER must not wait for unrelated table maintenance.
		// Cache eviction uses runtime dependency identities instead of this DDL
		// path, so no manager callback can wait on the owner's rebuild lock.
		// Revalidate below under the usual DDL/schema locks; this snapshot only
		// excludes non-owners and does not authorize removing a stale trigger.
		t.mu.Lock()
		ownsTrigger := false
		for _, tr := range t.Triggers {
			if tr.Name == name {
				ownsTrigger = true
				break
			}
		}
		t.mu.Unlock()
		if !ownsTrigger {
			continue
		}
		removed := func() bool {
			t.ddlMu.Lock()
			defer t.ddlMu.Unlock()
			db.schemalock.Lock()
			locked := true
			defer func() {
				if locked {
					db.schemalock.Unlock()
				}
			}()
			if db.tables.Get(t.Name) != t {
				return false
			}
			if !system && !tableMaintenanceCapabilities(db.Name, t.Name).canAlter {
				requireTableMaintenance(db.Name, t.Name, maintenanceAlter)
			}
			removed := func() bool {
				t.mu.Lock()
				defer t.mu.Unlock()
				return t.removeTriggerLocked(name, &system)
			}()
			if !removed {
				return false
			}
			// The persistence owner releases schemalock even on failure.
			locked = false
			db.saveLockedAndUnlock(t.schemaSaveMode())
			return true
		}()
		if removed {
			return true
		}
	}
	return false
}

// SetTriggerTarget refreshes only a runtime cache pin, including for hidden
// native callbacks. It does not replace a trigger's definition or grant system
// declaration ownership. Generated-definition callers validate that separately.
func (t *table) SetTriggerTarget(name string, target *table, acquire func(*TxContext) bool, release func()) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := range t.Triggers {
		if t.Triggers[i].Name == name {
			target.rememberCacheTriggerSource(t)
			t.Triggers[i].cacheTarget = target
			t.Triggers[i].Acquire = acquire
			t.Triggers[i].Release = release
			return true
		}
	}
	return false
}

// rowToDict converts a dataset to a dict with column names
func (t *table) rowToDict(row dataset) scm.Scmer {
	if row == nil {
		return scm.NewNil()
	}
	fd := scm.NewFastDictValue(len(t.Columns))
	for i, col := range t.Columns {
		if i < len(row) {
			fd.Set(scm.NewString(col.Name), row[i], nil)
		}
	}
	return scm.NewFastDict(fd)
}

// dictToRow converts a dict back to a dataset using column order
func (t *table) dictToRow(dict scm.Scmer, columns []string) dataset {
	if dict.IsNil() {
		return nil
	}
	row := make(dataset, len(columns))
	if dict.IsFastDict() {
		fd := dict.FastDict()
		for i, col := range columns {
			if v, ok := fd.Get(scm.NewString(col)); ok {
				row[i] = v
			} else {
				row[i] = scm.NewNil()
			}
		}
	}
	return row
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (t *table) beforeInsertOutputColumns(dict scm.Scmer, columns []string) []string {
	result := make([]string, 0, len(t.Columns))
	seen := make(map[string]struct{}, len(columns))
	for _, col := range columns {
		result = append(result, col)
		seen[col] = struct{}{}
	}
	if !dict.IsFastDict() {
		return result
	}
	fd := dict.FastDict()
	for _, col := range t.Columns {
		if _, ok := seen[col.Name]; ok {
			continue
		}
		if _, ok := fd.Get(scm.NewString(col.Name)); ok {
			result = append(result, col.Name)
		}
	}
	return result
}

// ExecuteTriggers executes all triggers for a specific timing (AFTER triggers).
// oldRow is nil for INSERT, newRow is nil for DELETE.
// If a transaction is active, a savepoint is created before each trigger;
// on panic the savepoint is rolled back before re-panicking.
// Triggers with Async=true are launched in a background goroutine (fire-and-forget).
func (t *table) ExecuteTriggers(timing TriggerTiming, oldRow, newRow dataset, tx *TxContext) {
	triggers := t.GetTriggers(timing)
	if len(triggers) != 0 {
		tx = nativeRowTriggerInvocation(tx)
	}
	session := txSessionScmer(tx)
	for _, tr := range triggers {
		if tr.Func.IsNil() {
			continue
		}
		// Build arguments: pass OLD and NEW as dicts with column names
		var oldDict, newDict scm.Scmer = scm.NewNil(), scm.NewNil()
		switch timing {
		case BeforeInsert, AfterInsert:
			newDict = t.rowToDict(newRow)
		case BeforeDelete, AfterDelete:
			oldDict = t.rowToDict(oldRow)
		case BeforeUpdate, AfterUpdate:
			oldDict = t.rowToDict(oldRow)
			newDict = t.rowToDict(newRow)
		case AfterInvalidate:
			// no row data — column-level invalidation propagation
		}
		if tr.Async && tx.hasStatementUndo() {
			panic("asynchronous triggers cannot participate in checked statement undo")
		}
		if tr.Async {
			// Fire-and-forget: run in background goroutine, no transaction context
			trFunc := tr.Func
			trName := tr.Name
			tName := t.Name
			go func() {
				defer func() {
					recover() // async triggers must not crash the process
					_ = trName
					_ = tName
				}()
				if !tr.acquireTarget(nil) {
					return
				}
				defer tr.releaseTarget()
				scm.Apply(trFunc, oldDict, newDict, session, scm.NewNil())
			}()
			continue
		}
		if !tr.acquireTarget(tx) {
			continue
		}
		// Execute trigger with savepoint for proper rollback
		func() {
			defer tr.releaseTarget()
			var sp Savepoint
			hasSavepoint := false
			if tx != nil && !tx.hasStatementUndo() {
				sp = tx.CreateSavepoint()
				hasSavepoint = true
			}
			defer func() {
				if r := recover(); r != nil {
					if hasSavepoint {
						tx.RollbackToSavepoint(sp)
					}
					panic(triggerExecutionFailure(tr.Name, timing, t.Name, r))
				}
			}()
			scm.Apply(tr.Func, oldDict, newDict, session, txContextScmer(tx))
		}()
	}
}

// ExecuteTriggersBatch fires triggers once per trigger with a batch of rows.
// For triggers that have a vectorized form (VectorFunc), the batch is passed
// as a single call. For non-vectorized triggers, falls back to per-row execution.
func (t *table) ExecuteTriggersBatch(timing TriggerTiming, rows []dataset, isOld bool, tx *TxContext) {
	tx = nativeRowTriggerInvocation(tx)
	if len(rows) == 0 {
		return
	}
	session := txSessionScmer(tx)
	if len(rows) == 1 {
		// Single row: use the normal path
		if isOld {
			t.ExecuteTriggers(timing, rows[0], nil, tx)
		} else {
			t.ExecuteTriggers(timing, nil, rows[0], tx)
		}
		return
	}
	triggers := t.GetTriggers(timing)
	for _, tr := range triggers {
		if tr.Func.IsNil() {
			continue
		}
		if tr.Async && tx.hasStatementUndo() {
			panic("asynchronous triggers cannot participate in checked statement undo")
		}
		if tr.Async {
			// Each queued invocation acquires its own target pin. Holding one only
			// while launching goroutines would let eviction race their execution.
			for _, row := range rows {
				var oldDict, newDict scm.Scmer = scm.NewNil(), scm.NewNil()
				if isOld {
					oldDict = t.rowToDict(row)
				} else {
					newDict = t.rowToDict(row)
				}
				tr := tr
				go func() {
					defer func() { recover() }()
					if !tr.acquireTarget(nil) {
						return
					}
					defer tr.releaseTarget()
					scm.Apply(tr.Func, oldDict, newDict, session, scm.NewNil())
				}()
			}
			continue
		}
		if !tr.acquireTarget(tx) {
			continue
		}
		func() {
			defer tr.releaseTarget()
			// Check for vectorized trigger (VectorFunc set)
			if !tr.VectorFunc.IsNil() && !tx.hasStatementUndo() {
				// Build columnar dict-of-lists: {"col1": [v1,v2,...], "col2": [v1,v2,...]}
				colBatch := t.rowsToColumnar(rows)
				func() {
					var sp Savepoint
					hasSavepoint := false
					if tx != nil && !tx.hasStatementUndo() {
						sp = tx.CreateSavepoint()
						hasSavepoint = true
					}
					defer func() {
						if r := recover(); r != nil {
							if hasSavepoint {
								tx.RollbackToSavepoint(sp)
							}
							// Vectorization failed: fall back to per-row
							for _, row := range rows {
								var oldDict, newDict scm.Scmer = scm.NewNil(), scm.NewNil()
								if isOld {
									oldDict = t.rowToDict(row)
								} else {
									newDict = t.rowToDict(row)
								}
								scm.Apply(tr.Func, oldDict, newDict, session, txContextScmer(tx))
							}
						}
					}()
					if isOld {
						scm.Apply(tr.VectorFunc, colBatch, scm.NewNil(), session, txContextScmer(tx))
					} else {
						scm.Apply(tr.VectorFunc, scm.NewNil(), colBatch, session, txContextScmer(tx))
					}
				}()
				return
			}
			// Fallback: per-row execution
			for _, row := range rows {
				var oldDict, newDict scm.Scmer = scm.NewNil(), scm.NewNil()
				if isOld {
					oldDict = t.rowToDict(row)
				} else {
					newDict = t.rowToDict(row)
				}
				func() {
					var sp Savepoint
					hasSavepoint := false
					if tx != nil && !tx.hasStatementUndo() {
						sp = tx.CreateSavepoint()
						hasSavepoint = true
					}
					defer func() {
						if r := recover(); r != nil {
							if hasSavepoint {
								tx.RollbackToSavepoint(sp)
							}
							panic(triggerExecutionFailure(tr.Name, timing, t.Name, r))
						}
					}()
					scm.Apply(tr.Func, oldDict, newDict, session, txContextScmer(tx))
				}()
			}
		}()
	}
}

// rowsToColumnar converts a batch of rows into columnar dict-of-lists format.
// Result: FastDict{"col1": [v1,v2,...], "col2": [v1,v2,...], ...}
// This is the optimal format for vectorized trigger bodies: get_assoc returns
// the entire column as a list, enabling batch operations like has?.
func (t *table) rowsToColumnar(rows []dataset) scm.Scmer {
	if len(rows) == 0 {
		return scm.NewNil()
	}
	cols := t.Columns
	// Build flat assoc: (col1 [v1,v2,...] col2 [v1,v2,...] ...)
	result := make([]scm.Scmer, 0, len(cols)*2)
	for ci, col := range cols {
		vals := make([]scm.Scmer, len(rows))
		for i, row := range rows {
			if row != nil && ci < len(row) {
				vals[i] = row[ci]
			} else {
				vals[i] = scm.NewNil()
			}
		}
		result = append(result, scm.NewString(col.Name), scm.NewSlice(vals))
	}
	return scm.NewSlice(result)
}

// rowToDictWithColumns converts a dataset to a dict using explicit column names
func (t *table) rowToDictWithColumns(row dataset, columns []string) scm.Scmer {
	if row == nil {
		return scm.NewNil()
	}
	fd := scm.NewFastDictValue(len(columns))
	for i, col := range columns {
		if i < len(row) {
			fd.Set(scm.NewString(col), row[i], nil)
		}
	}
	return scm.NewFastDict(fd)
}

// ExecuteTableLifecycleTriggers executes non-row-level table lifecycle triggers.
// These are non-row-level triggers: OLD and NEW are both nil.
func (t *table) ExecuteTableLifecycleTriggers(timing TriggerTiming, tx *TxContext) {
	triggers := t.GetTriggers(timing)
	session := txSessionScmer(tx)
	for _, tr := range triggers {
		if tr.Func.IsNil() {
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					// lifecycle triggers are best-effort; log but don't propagate
				}
			}()
			scm.Apply(tr.Func, scm.NewNil(), scm.NewNil(), session, txContextScmer(tx))
		}()
	}
}

func executeRegisteredCreateTableTriggers(t *table, tx *TxContext) {
	registrations := getCreateTableTriggers(t.schema.Name, t.Name)
	if len(registrations) == 0 {
		return
	}
	session := txSessionScmer(tx)
	for _, reg := range registrations {
		trigger := reg.triggerDescription()
		compileTriggerForUse(t.schema.Name, t.Name, &trigger)
		if triggerScmerMissing(trigger.Func) {
			continue
		}
		scm.Apply(trigger.Func, scm.NewNil(), scm.NewNil(), session, txContextScmer(tx))
	}
}

// splitBeforeInsertTriggers consumes one published list. Final validators keep
// their producer's native capability; public callbacks remain in the early phase.
func splitBeforeInsertTriggers(triggers []TriggerDescription) (early, final []TriggerDescription) {
	for _, trigger := range triggers {
		if trigger.needsFinalInsertValues {
			final = append(final, trigger)
		} else {
			early = append(early, trigger)
		}
	}
	return
}

// Retain the existing per-callback savepoint and IGNORE recovery boundary. It
// does not rewind user callbacks that completed before this invocation.
func (t *table) invokeBeforeInsertTrigger(trigger TriggerDescription, newDict scm.Scmer, isIgnore bool, tx *TxContext) (returned scm.Scmer, ok bool) {
	ok = true
	var sp Savepoint
	hasSavepoint := tx != nil && !tx.hasStatementUndo()
	if hasSavepoint {
		sp = tx.CreateSavepoint()
	}
	defer func() {
		if failure := recover(); failure != nil {
			if hasSavepoint {
				tx.RollbackToSavepoint(sp)
			}
			if isIgnore {
				ok = false
				return
			}
			panic(triggerExecutionFailure(trigger.Name, BeforeInsert, t.Name, failure))
		}
	}()
	returned = scm.Apply(trigger.Func, scm.NewNil(), newDict, txSessionScmer(tx), txContextScmer(tx))
	return
}

// Ordinary BEFORE callbacks may modify NEW, before sanitization/ID allocation.
// The caller supplies the early subset of one captured native trigger list.
func (t *table) executeBeforeInsertTriggerRow(columns []string, row dataset, triggers []TriggerDescription, isIgnore bool, tx *TxContext) ([]string, dataset, bool) {
	tx = nativeRowTriggerInvocation(tx)
	if len(triggers) == 0 {
		return columns, row, true
	}
	newDict := t.rowToDictWithColumns(row, columns)
	triggerOk := true
	for _, trigger := range triggers {
		if trigger.Func.IsNil() {
			continue
		}
		returned, ok := t.invokeBeforeInsertTrigger(trigger, newDict, isIgnore, tx)
		if !ok {
			triggerOk = false
			break
		}
		if !returned.IsNil() && returned.IsFastDict() {
			newDict = returned
		}
	}
	if !triggerOk {
		return nil, nil, false
	}
	rowColumns := t.beforeInsertOutputColumns(newDict, columns)
	return rowColumns, t.dictToRow(newDict, rowColumns), true
}

func (t *table) ExecuteBeforeInsertTriggers(columns []string, values [][]scm.Scmer, triggers []TriggerDescription, isIgnore bool, tx *TxContext) ([]string, [][]scm.Scmer) {
	if len(triggers) == 0 {
		return columns, values
	}

	resultColumns := append([]string(nil), columns...)
	rowColumns := make([][]string, 0, len(values))
	result := make([][]scm.Scmer, 0, len(values))
	for _, row := range values {
		newColumns, newRow, ok := t.executeBeforeInsertTriggerRow(columns, row, triggers, isIgnore, tx)
		if !ok {
			continue
		}
		if len(result) == 0 {
			resultColumns = append([]string(nil), newColumns...)
		}
		rowColumns = append(rowColumns, newColumns)
		result = append(result, newRow)
	}
	if len(result) == 0 {
		return resultColumns, result
	}
	for _, cols := range rowColumns[1:] {
		if !stringSlicesEqual(resultColumns, cols) {
			allColumns := make([]string, len(t.Columns))
			for i, col := range t.Columns {
				allColumns[i] = col.Name
			}
			for i, row := range result {
				result[i] = t.dictToRow(t.rowToDictWithColumns(row, rowColumns[i]), allColumns)
			}
			return allColumns, result
		}
	}
	return resultColumns, result
}

// validateFinalInsertRows invokes only the native producer's finite scalar
// validators after sanitization and outer ID preparation, before routing/locks.
// Return values cannot modify the prepared row or add unsanitized output cells.
func (t *table) validateFinalInsertRows(columns []string, values [][]scm.Scmer, validators []TriggerDescription, isIgnore bool, tx *TxContext) [][]scm.Scmer {
	if len(validators) == 0 {
		return values
	}
	accepted := make([][]scm.Scmer, 0, len(values))
	for _, row := range values {
		newDict := t.rowToDictWithColumns(row, columns)
		valid := true
		for _, validator := range validators {
			if !validator.needsFinalInsertValues || validator.Func.IsNil() {
				panic("native final INSERT validator is unavailable")
			}
			if _, ok := t.invokeBeforeInsertTrigger(validator, newDict, isIgnore, tx); !ok {
				valid = false
				break
			}
		}
		if valid {
			accepted = append(accepted, row)
		}
	}
	return accepted
}

// ExecuteBeforeUpdateTriggers executes BEFORE UPDATE triggers.
// oldRow: the current row values (all columns in table order)
// newRow: the row with changes applied (all columns in table order)
// Returns the modified newRow. Panics from triggers propagate to the caller.
func (t *table) ExecuteBeforeUpdateTriggers(oldRow, newRow dataset, tx *TxContext) (dataset, []TriggerDescription) {
	tx = nativeRowTriggerInvocation(tx)
	triggers := t.GetTriggers(BeforeUpdate)
	if len(triggers) == 0 {
		return newRow, nil
	}
	var final []TriggerDescription

	// Build column names from table
	columns := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		columns[i] = col.Name
	}

	oldDict := t.rowToDictWithColumns(oldRow, columns)
	newDict := t.rowToDictWithColumns(newRow, columns)

	// Execute ordinary BEFORE callbacks; final predicates cannot modify NEW.
	for _, tr := range triggers {
		if tr.needsFinalUpdateValues {
			final = append(final, tr)
			continue
		}
		if tr.Func.IsNil() {
			continue
		}
		func() {
			var sp Savepoint
			hasSavepoint := false
			if tx != nil && !tx.hasStatementUndo() {
				sp = tx.CreateSavepoint()
				hasSavepoint = true
			}
			defer func() {
				if r := recover(); r != nil {
					if hasSavepoint {
						tx.RollbackToSavepoint(sp)
					}
					panic(triggerExecutionFailure(tr.Name, BeforeUpdate, t.Name, r))
				}
			}()
			returned := scm.Apply(tr.Func, oldDict, newDict, txSessionScmer(tx), txContextScmer(tx))
			if !returned.IsNil() && (returned.IsFastDict() || returned.IsSlice()) {
				newDict = returned
			}
		}()
	}

	// Convert modified dict back to row
	return t.dictToRow(newDict, columns), final
}

// ExecuteBeforeDeleteTriggers executes BEFORE DELETE triggers.
// oldRow: the row being deleted (all columns in table order)
// Returns true if delete should proceed, false to abort.
// Panics from triggers propagate to the caller.
func (t *table) ExecuteBeforeDeleteTriggers(oldRow dataset, tx *TxContext) bool {
	tx = nativeRowTriggerInvocation(tx)
	triggers := t.GetTriggers(BeforeDelete)
	if len(triggers) == 0 {
		return true
	}

	// Build column names from table
	columns := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		columns[i] = col.Name
	}

	oldDict := t.rowToDictWithColumns(oldRow, columns)

	// Execute all BEFORE DELETE triggers
	for _, tr := range triggers {
		if tr.Func.IsNil() {
			continue
		}
		var returned scm.Scmer
		func() {
			var sp Savepoint
			hasSavepoint := false
			if tx != nil && !tx.hasStatementUndo() {
				sp = tx.CreateSavepoint()
				hasSavepoint = true
			}
			defer func() {
				if r := recover(); r != nil {
					if hasSavepoint {
						tx.RollbackToSavepoint(sp)
					}
					panic(triggerExecutionFailure(tr.Name, BeforeDelete, t.Name, r))
				}
			}()
			returned = scm.Apply(tr.Func, oldDict, scm.NewNil(), txSessionScmer(tx), txContextScmer(tx))
		}()
		// If trigger explicitly returns false, abort delete.
		// nil return (side-effect-only triggers) does NOT abort.
		if returned.IsBool() && !scm.ToBool(returned) {
			return false
		}
	}

	return true
}
