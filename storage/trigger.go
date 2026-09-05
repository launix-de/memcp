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
)

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
	default:
		panic("unknown trigger timing")
	}
}

func (tt TriggerTiming) MarshalJSON() ([]byte, error) {
	if tt > AfterCreateTable {
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
	if n > uint8(AfterCreateTable) {
		return fmt.Errorf("unknown trigger timing number: %d", n)
	}
	*tt = TriggerTiming(n)
	return nil
}

// TriggerDescription holds all information about a trigger
type TriggerDescription struct {
	Name       string                `json:"name"`                // Trigger name (user-defined or auto-generated)
	Timing     TriggerTiming         `json:"timing"`              // BEFORE/AFTER INSERT/UPDATE/DELETE
	Func       scm.Scmer             `json:"func"`                // The compiled trigger procedure; omitted when source is authoritative
	FuncPlan   scm.Scmer             `json:"-"`                   // Unevaluated lambda AST, compiled lazily on first use
	Source     string                `json:"source,omitempty"`    // Authoritative source code for persistent language-defined triggers
	Language   string                `json:"language,omitempty"`  // Name resolved through the process-local trigger compiler registry
	IsSystem   bool                  `json:"is_system,omitempty"` // True for Go-internal triggers (FK etc.) — not persisted via createtrigger
	Hidden     bool                  `json:"hidden,omitempty"`    // True for Scheme-internal triggers — persisted but hidden from SHOW TRIGGERS
	Priority   int                   `json:"priority,omitempty"`  // Execution order (lower = earlier)
	Async      bool                  `json:"async,omitempty"`     // Run trigger in background goroutine (fire-and-forget, no transaction context)
	VectorFunc scm.Scmer             `json:"-"`                   // Vectorized trigger: (lambda (OLD_batch NEW_batch) ...) for batch execution
	Acquire    func(*TxContext) bool `json:"-"`                   // Optional lock-free pin for an ephemeral trigger target
	Release    func()                `json:"-"`                   // Releases a successful Acquire
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
	LegacySourceSQL string        `json:"source_sql,omitempty"`
	IsSystem        bool          `json:"is_system,omitempty"`
	Hidden          bool          `json:"hidden,omitempty"`
	Priority        int           `json:"priority,omitempty"`
	Async           bool          `json:"async,omitempty"`
	RequiresTarget  bool          `json:"requires_target,omitempty"`
}

func (tr TriggerDescription) MarshalJSON() ([]byte, error) {
	persist := persistedTriggerDescription{
		Name:           tr.Name,
		Timing:         tr.Timing,
		Source:         tr.Source,
		Language:       tr.Language,
		IsSystem:       tr.IsSystem,
		Hidden:         tr.Hidden,
		Priority:       tr.Priority,
		Async:          tr.Async,
		RequiresTarget: tr.Acquire != nil,
	}
	// Source plus language is the durable definition. Compiled Scheme is a
	// process-local artifact and is persisted only for legacy/internal triggers
	// without source. FK callbacks are generated from FK metadata instead.
	regeneratedFK := tr.IsSystem && strings.HasPrefix(tr.Name, "__fk_")
	if tr.Language == "" && !regeneratedFK {
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
	tr.Name = persist.Name
	tr.Timing = persist.Timing
	tr.Source = persist.Source
	tr.Language = persist.Language
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
	tr.Async = persist.Async
	tr.FuncPlan = scm.NewNil()
	tr.VectorFunc = scm.NewNil()
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
	return nil
}

func unavailableTriggerTarget(_ *TxContext) bool { return false }

func restoredTriggerRequiresRuntimeTarget(name string) bool {
	// Older schema files predate requires_target. These are all current
	// internal trigger families whose bodies address an evictable cache target.
	return strings.HasPrefix(name, ".kt_cleanup:") ||
		strings.HasPrefix(name, ".cache:") ||
		strings.HasPrefix(name, ".orcdep:")
}

func triggerScmerMissing(v scm.Scmer) bool {
	return v.IsNil()
}

var triggerLanguageCompilers = struct {
	sync.RWMutex
	byName map[string]scm.Scmer
}{byName: make(map[string]scm.Scmer)}

func registerTriggerLanguage(language string, compiler scm.Scmer) {
	if language == "" {
		panic("trigger language must not be empty")
	}
	if compiler.IsNil() {
		panic("trigger language compiler must not be empty")
	}
	triggerLanguageCompilers.Lock()
	triggerLanguageCompilers.byName[language] = compiler
	triggerLanguageCompilers.Unlock()
}

func triggerLanguageCompiler(language string) (scm.Scmer, bool) {
	triggerLanguageCompilers.RLock()
	compiler, ok := triggerLanguageCompilers.byName[language]
	triggerLanguageCompilers.RUnlock()
	return compiler, ok
}

func triggerCompilerContext(schemaName, tableName string, trigger *TriggerDescription) scm.Scmer {
	context := scm.NewFastDictValue(5)
	context.Set(scm.NewString("schema"), scm.NewString(schemaName), nil)
	context.Set(scm.NewString("table"), scm.NewString(tableName), nil)
	context.Set(scm.NewString("name"), scm.NewString(trigger.Name), nil)
	context.Set(scm.NewString("timing"), scm.NewString(trigger.Timing.sourceName()), nil)
	context.Set(scm.NewString("hidden"), scm.NewBool(trigger.Hidden), nil)
	return scm.NewFastDict(context)
}

func loadPersistedTriggerPlan(schemaName, tableName string, trigger *TriggerDescription) {
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
	// native compilation. Compile the outer trigger afterwards: nested physical
	// callbacks retain their own scan specialization and compilation boundary.
	// Apply then dispatches directly to the native trigger entry point, while
	// unsupported bodies and non-JIT builds keep the interpreter fallback.
	if (trigger.IsSystem || trigger.Hidden || trigger.Language == "") && trigger.VectorFunc.IsNil() {
		if vf := VectorizeTrigger(trigger.Func); !vf.IsNil() {
			trigger.VectorFunc = vf
		}
	}
	trigger.Func = scm.CompileJIT(scm.CloseProcedure(trigger.Func), false)
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
		triggerIndex int
		resultIndex  int
	}
	lazy := make([]lazyTrigger, 0)
	for i, tr := range t.Triggers {
		if tr.Timing != timing {
			continue
		}
		result = append(result, tr)
		if triggerScmerMissing(tr.Func) && (!triggerScmerMissing(tr.FuncPlan) || tr.Language != "") {
			lazy = append(lazy, lazyTrigger{triggerIndex: i, resultIndex: len(result) - 1})
		}
	}
	t.mu.Unlock()
	for _, item := range lazy {
		tr := result[item.resultIndex]
		compileTriggerForUse(t.schema.Name, t.Name, &tr)
		result[item.resultIndex] = tr

		t.mu.Lock()
		if item.triggerIndex < len(t.Triggers) {
			current := &t.Triggers[item.triggerIndex]
			if current.Name == tr.Name && current.Timing == tr.Timing && triggerScmerMissing(current.Func) {
				current.Func = tr.Func
				current.FuncPlan = tr.FuncPlan
				current.VectorFunc = tr.VectorFunc
			}
		}
		t.mu.Unlock()
	}
	return result
}

// AddTrigger adds a trigger to the table. Automatically attempts to vectorize
// the trigger for batch execution (DELETE/INSERT patterns on prejoin tables).
func (t *table) AddTrigger(trigger TriggerDescription) {
	// Auto-vectorize: try to produce a batch-aware version of the trigger
	// Contract: only internal/system triggers participate in automatic
	// vectorization. User-facing SQL triggers may have very large bodies;
	// walking their full AST during CREATE TRIGGER would make schema setup
	// disproportionately expensive without improving correctness.
	finalizeTriggerCompilation(&trigger)
	t.mu.Lock()
	defer t.mu.Unlock()
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
}

// RemoveTrigger removes a trigger by name
func (t *table) RemoveTrigger(name string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, tr := range t.Triggers {
		if tr.Name == name {
			t.Triggers = append(t.Triggers[:i], t.Triggers[i+1:]...)
			return true
		}
	}
	return false
}

// dropTrigger follows the same table-DDL-before-schema lock order as trigger
// creation. Tables may disappear between the catalog snapshot and locking, so
// each candidate is revalidated under schemalock before it is modified.
func (db *database) dropTrigger(name string) bool {
	db.schemalock.RLock()
	tables := db.tables.GetAll()
	db.schemalock.RUnlock()

	for _, t := range tables {
		t.ddlMu.Lock()
		db.schemalock.Lock()
		if db.tables.Get(t.Name) != t {
			db.schemalock.Unlock()
			t.ddlMu.Unlock()
			continue
		}
		if !tableMaintenanceCapabilities(db.Name, t.Name).canAlter {
			t.mu.Lock()
			found := false
			for _, trigger := range t.Triggers {
				if trigger.Name == name {
					found = true
					break
				}
			}
			t.mu.Unlock()
			if found {
				db.schemalock.Unlock()
				t.ddlMu.Unlock()
				requireTableMaintenance(db.Name, t.Name, maintenanceAlter)
			}
		}
		if t.RemoveTrigger(name) {
			db.saveLockedAndUnlock(t.schemaSaveMode())
			t.ddlMu.Unlock()
			return true
		}
		db.schemalock.Unlock()
		t.ddlMu.Unlock()
	}
	return false
}

// SetTriggerTarget refreshes the runtime-only cache target pin on an
// idempotently reused system trigger, including triggers restored from JSON.
func (t *table) SetTriggerTarget(name string, acquire func(*TxContext) bool, release func()) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := range t.Triggers {
		if t.Triggers[i].Name == name {
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
			if tx != nil {
				sp = tx.CreateSavepoint()
				hasSavepoint = true
			}
			defer func() {
				if r := recover(); r != nil {
					if hasSavepoint {
						tx.RollbackToSavepoint(sp)
					}
					panic(fmt.Sprintf("trigger %s (%s) on %s failed: %v", tr.Name, timing, t.Name, r))
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
			if !tr.VectorFunc.IsNil() {
				// Build columnar dict-of-lists: {"col1": [v1,v2,...], "col2": [v1,v2,...]}
				colBatch := t.rowsToColumnar(rows)
				func() {
					var sp Savepoint
					hasSavepoint := false
					if tx != nil {
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
					if tx != nil {
						sp = tx.CreateSavepoint()
						hasSavepoint = true
					}
					defer func() {
						if r := recover(); r != nil {
							if hasSavepoint {
								tx.RollbackToSavepoint(sp)
							}
							panic(fmt.Sprintf("trigger %s (%s) on %s failed: %v", tr.Name, timing, t.Name, r))
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

// ExecuteBeforeInsertTriggers executes BEFORE INSERT triggers and returns modified rows.
// The trigger function can modify NEW values by returning a modified dict, including
// columns not present in the original INSERT.
// When isIgnore is true, rows whose triggers panic are silently skipped
// and any partial transaction effects are rolled back via savepoints.
// When isIgnore is false, trigger panics propagate to the caller.
func (t *table) executeBeforeInsertTriggerRow(columns []string, row dataset, isIgnore bool, tx *TxContext) ([]string, dataset, bool) {
	triggers := t.GetTriggers(BeforeInsert)
	if len(triggers) == 0 {
		return columns, row, true
	}

	// Build dict using the columns that are being inserted.
	newDict := t.rowToDictWithColumns(row, columns)
	triggerOk := true
	for _, tr := range triggers {
		if tr.Func.IsNil() {
			continue
		}
		if isIgnore {
			// Per-row savepoint + panic recovery for INSERT IGNORE.
			func() {
				var sp Savepoint
				hasSavepoint := false
				if tx != nil {
					sp = tx.CreateSavepoint()
					hasSavepoint = true
				}
				defer func() {
					if r := recover(); r != nil {
						if hasSavepoint {
							tx.RollbackToSavepoint(sp)
						}
						triggerOk = false
					}
				}()
				returned := scm.Apply(tr.Func, scm.NewNil(), newDict, txSessionScmer(tx), txContextScmer(tx))
				if !returned.IsNil() && returned.IsFastDict() {
					newDict = returned
				}
			}()
			if !triggerOk {
				break
			}
		} else {
			// Normal mode: savepoint for proper rollback on propagated panic.
			func() {
				var sp Savepoint
				hasSavepoint := false
				if tx != nil {
					sp = tx.CreateSavepoint()
					hasSavepoint = true
				}
				defer func() {
					if r := recover(); r != nil {
						if hasSavepoint {
							tx.RollbackToSavepoint(sp)
						}
						panic(fmt.Sprintf("trigger %s (BEFORE INSERT) on %s failed: %v", tr.Name, t.Name, r))
					}
				}()
				returned := scm.Apply(tr.Func, scm.NewNil(), newDict, txSessionScmer(tx), txContextScmer(tx))
				if !returned.IsNil() && returned.IsFastDict() {
					newDict = returned
				}
			}()
		}
	}
	if !triggerOk {
		return nil, nil, false
	}
	rowColumns := t.beforeInsertOutputColumns(newDict, columns)
	return rowColumns, t.dictToRow(newDict, rowColumns), true
}

func (t *table) ExecuteBeforeInsertTriggers(columns []string, values [][]scm.Scmer, isIgnore bool, tx *TxContext) ([]string, [][]scm.Scmer) {
	triggers := t.GetTriggers(BeforeInsert)
	if len(triggers) == 0 {
		return columns, values
	}

	resultColumns := append([]string(nil), columns...)
	rowColumns := make([][]string, 0, len(values))
	result := make([][]scm.Scmer, 0, len(values))
	for _, row := range values {
		newColumns, newRow, ok := t.executeBeforeInsertTriggerRow(columns, row, isIgnore, tx)
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

// ExecuteBeforeUpdateTriggers executes BEFORE UPDATE triggers.
// oldRow: the current row values (all columns in table order)
// newRow: the row with changes applied (all columns in table order)
// Returns the modified newRow. Panics from triggers propagate to the caller.
func (t *table) ExecuteBeforeUpdateTriggers(oldRow, newRow dataset, tx *TxContext) dataset {
	triggers := t.GetTriggers(BeforeUpdate)
	if len(triggers) == 0 {
		return newRow
	}

	// Build column names from table
	columns := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		columns[i] = col.Name
	}

	oldDict := t.rowToDictWithColumns(oldRow, columns)
	newDict := t.rowToDictWithColumns(newRow, columns)

	// Execute all BEFORE UPDATE triggers
	for _, tr := range triggers {
		if tr.Func.IsNil() {
			continue
		}
		func() {
			var sp Savepoint
			hasSavepoint := false
			if tx != nil {
				sp = tx.CreateSavepoint()
				hasSavepoint = true
			}
			defer func() {
				if r := recover(); r != nil {
					if hasSavepoint {
						tx.RollbackToSavepoint(sp)
					}
					panic(fmt.Sprintf("trigger %s (BEFORE UPDATE) on %s failed: %v", tr.Name, t.Name, r))
				}
			}()
			returned := scm.Apply(tr.Func, oldDict, newDict, txSessionScmer(tx), txContextScmer(tx))
			if !returned.IsNil() && (returned.IsFastDict() || returned.IsSlice()) {
				newDict = returned
			}
		}()
	}

	// Convert modified dict back to row
	return t.dictToRow(newDict, columns)
}

// ExecuteBeforeDeleteTriggers executes BEFORE DELETE triggers.
// oldRow: the row being deleted (all columns in table order)
// Returns true if delete should proceed, false to abort.
// Panics from triggers propagate to the caller.
func (t *table) ExecuteBeforeDeleteTriggers(oldRow dataset, tx *TxContext) bool {
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
			if tx != nil {
				sp = tx.CreateSavepoint()
				hasSavepoint = true
			}
			defer func() {
				if r := recover(); r != nil {
					if hasSavepoint {
						tx.RollbackToSavepoint(sp)
					}
					panic(fmt.Sprintf("trigger %s (BEFORE DELETE) on %s failed: %v", tr.Name, t.Name, r))
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
