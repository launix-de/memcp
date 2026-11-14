/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"
	"unsafe"

	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
)

// ONE logical, immutable source recipe owns at most three derived programs.
// Header copies borrow this node. Catalog history and the public header retain
// the existing sourceDefinitionMemory owner; no invocation/session is stored.
type statementTriggerDefinition struct {
	header             TriggerDescription
	programs           [3]statementTriggerProgram
	memory             sourceDefinitionMemory
	pendingDecision    string
	publicationVersion atomic.Uint64
	previousCommitted  atomic.Pointer[statementTriggerDefinition]
	removed            bool
	reservation        bool
	compiledTableName  string // admission revalidates the compiler's actual target context
	compiledIdentity   string
}

type statementTriggerProgram struct {
	once     sync.Once
	compiled atomic.Pointer[scm.Scmer]
	failure  any
}

func validateStatementTrigger(header TriggerDescription) error {
	if !header.Timing.isStatement() || header.Name == "" || header.Source == "" || header.Language == "" ||
		!utf8.ValidString(header.Name) || !utf8.ValidString(header.Source) || !utf8.ValidString(header.Language) ||
		header.IsSystem || header.Hidden || header.Async || len(header.StatementEvents) == 0 ||
		!header.Func.IsNil() || !header.FuncPlan.IsNil() || !header.VectorFunc.IsNil() || header.Acquire != nil || header.Release != nil {
		return fmt.Errorf("invalid native statement trigger declaration")
	}
	seen := uint8(0)
	for _, name := range header.StatementEvents {
		if header.Timing == ReplaceStatement && name != "delete" {
			return fmt.Errorf("native statement replacement currently supports delete only")
		}
		var event nativeMutationEvent
		switch name {
		case "insert":
			event = nativeMutationInsert
		case "update":
			event = nativeMutationUpdate
		case "delete":
			event = nativeMutationDelete
		default:
			return fmt.Errorf("unknown native statement trigger event: %q", name)
		}
		bit := uint8(1 << event)
		if seen&bit != 0 {
			return fmt.Errorf("duplicate native statement trigger event: %s", name)
		}
		seen |= bit
	}
	return nil
}

func newStatementTriggerDefinition(header TriggerDescription) *statementTriggerDefinition {
	definition := &statementTriggerDefinition{header: header}
	definition.header.Name = strings.Clone(header.Name)
	definition.header.Source = strings.Clone(header.Source)
	definition.header.Language = strings.Clone(header.Language)
	definition.header.StatementEvents = append([]string(nil), header.StatementEvents...)
	for i, event := range definition.header.StatementEvents {
		definition.header.StatementEvents[i] = strings.Clone(event)
	}
	definition.header.Func, definition.header.FuncPlan, definition.header.VectorFunc = scm.NewNil(), scm.NewNil(), scm.NewNil()
	definition.header.statementDefinition = nil
	definition.memory.bytes = uint(unsafe.Sizeof(*definition)) + uint(len(definition.header.Name)+len(definition.header.Source)+len(definition.header.Language)) +
		uint(cap(definition.header.StatementEvents))*uint(unsafe.Sizeof(""))
	for _, event := range definition.header.StatementEvents {
		definition.memory.bytes += uint(len(event))
	}
	return definition
}

func (definition *statementTriggerDefinition) triggerHeader() TriggerDescription {
	header := definition.header
	header.statementDefinition = definition
	return header
}

func (definition *statementTriggerDefinition) pending(version uint64) bool {
	return definition != nil && definition.pendingDecision != "" &&
		(definition.publicationVersion.Load() == 0 || definition.publicationVersion.Load() > version)
}

func (definition *statementTriggerDefinition) visible(db *database, tx *TxContext) *statementTriggerDefinition {
	if definition == nil || definition.pendingDecision == "" {
		return definition
	}
	for {
		view := db.catalogView()
		version := definition.publicationVersion.Load()
		own := tx != nil && tx.State == TxActive && tx.pendingCatalogKey() == definition.pendingDecision
		if version != 0 && version <= view.version || own && !definition.reservation {
			if definition.removed {
				return nil
			}
			return definition
		}
		if previous := definition.previousCommitted.Load(); previous != nil {
			return previous
		}
		if db.catalogView().version == view.version {
			return nil
		}
	}
}

// Caller owns table.mu. Selection is identical for metadata and schema saves;
// neither a private replacement nor a tombstone leaks into a public snapshot.
func (t *table) visibleTriggerDescriptionsLocked(tx *TxContext) []TriggerDescription {
	result := make([]TriggerDescription, 0, len(t.Triggers))
	for _, header := range t.Triggers {
		if header.Timing.isStatement() {
			definition := header.statementDefinition.visible(t.schema, tx)
			if definition == nil {
				continue
			}
			header = definition.triggerHeader()
		}
		result = append(result, header)
	}
	return result
}

func (definition *statementTriggerDefinition) callable(t *table, event nativeMutationEvent) scm.Scmer {
	index := int(event) - 1
	if index < 0 || index >= len(definition.programs) {
		panic("unknown native statement trigger event")
	}
	entry := &definition.programs[index]
	if program := entry.compiled.Load(); program != nil {
		return *program
	}
	language, found := triggerLanguageDefinitionFor(definition.header.Language)
	if !found || !language.statementBody {
		panic("statement trigger requires a registered statement-body source language: " + definition.header.Language)
	}
	entry.once.Do(func() {
		defer func() { entry.failure = recover() }()
		name := definition.compiledTableName
		if name == "" {
			name = t.clusterPublishedName()
		}
		context := triggerCompilerContext(t.schema.Name, name, &definition.header)
		context.FastDict().Set(scm.NewString("timing"), scm.NewString(definition.header.Timing.sourceName()+"_"+event.String()), nil)
		value := scm.Apply(language.compiler, scm.NewString(definition.header.Source), context)
		program, plan := unwrapDeferredTriggerBody(value)
		if program.IsNil() && !plan.IsNil() {
			program = evaluateTriggerPlan(plan)
		}
		program = program.WithoutSourceInfo()
		if !program.IsProc() || !program.Proc().Params.WithoutSourceInfo().IsSlice() || len(program.Proc().Params.WithoutSourceInfo().Slice()) != 4 {
			panic("statement trigger compiler must return a four-argument procedure")
		}
		program = scm.CompileJIT(scm.CloseProcedure(program), true)
		definition.memory.publishInitialCallableMemory(program)
		entry.compiled.Store(&program)
	})
	if entry.failure != nil {
		panic(entry.failure)
	}
	return *entry.compiled.Load()
}

// Writer admission fences definition changes. Copy only immutable recipe roots
// under table.mu, then run the actual compiler outside all table/schema locks.
func (t *table) statementTriggerProgramsFor(event nativeMutationEvent, tx *TxContext, bypassReplacement bool) []nativeStatementTrigger {
	t.mu.Lock()
	headers := t.visibleTriggerDescriptionsLocked(tx)
	t.mu.Unlock()
	programs := make([]nativeStatementTrigger, 0)
	timing := AfterStatement
	if !bypassReplacement {
		for _, header := range headers {
			if header.Timing == ReplaceStatement {
				for _, name := range header.StatementEvents {
					if name == event.String() {
						timing = ReplaceStatement
					}
				}
			}
		}
	}
	for _, header := range headers {
		if header.Timing != timing {
			continue
		}
		for _, name := range header.StatementEvents {
			if name == event.String() {
				programs = append(programs, nativeStatementTrigger{name: header.Name, replacement: timing == ReplaceStatement, program: header.statementDefinition.callable(t, event)})
				break
			}
		}
	}
	return programs
}

func statementTriggerEventNames(events []string) scm.Scmer {
	result := make([]scm.Scmer, len(events))
	for i, event := range events {
		result[i] = scm.NewString(event)
	}
	return scm.NewSlice(result)
}

func (t *table) statementTriggerDeclarations(tx *TxContext) scm.Scmer {
	t.mu.Lock()
	defer t.mu.Unlock()
	rows := make([]scm.Scmer, 0)
	for _, header := range t.visibleTriggerDescriptionsLocked(tx) {
		if header.Timing.isStatement() {
			rows = append(rows, scm.NewSlice([]scm.Scmer{
				scm.NewString("name"), scm.NewString(header.Name),
				scm.NewString("timing"), scm.NewString(header.Timing.sourceName()),
				scm.NewString("events"), statementTriggerEventNames(header.StatementEvents),
				scm.NewString("language"), scm.NewString(header.Language),
				scm.NewString("source"), scm.NewString(header.Source),
			}))
		}
	}
	return scm.NewSlice(rows)
}

// Loading owns a private schema, so validate the new declaration namespace
// before retaining any of its payloads or publishing a table. Legacy row
// headers keep their existing domains; only statement declarations require
// this exact native containing-object identity and unique user name.
func (db *database) validateStoredStatementTriggers() error {
	names := make(map[string]int)
	for _, t := range db.tables.GetAll() {
		for _, header := range t.Triggers {
			names[header.Name]++
		}
	}
	for _, t := range db.tables.GetAll() {
		var replacementEvents uint8
		for _, header := range t.Triggers {
			if !header.Timing.isStatement() {
				continue
			}
			if names[header.Name] != 1 || header.statementDefinition == nil || t.Ephemeral {
				return fmt.Errorf("invalid stored statement trigger namespace")
			}
			if _, err := uuid.Parse(t.RuntimeIdentity); err != nil {
				return fmt.Errorf("invalid stored statement trigger table identity")
			}
			if header.Timing == ReplaceStatement {
				for _, name := range header.StatementEvents {
					bit := uint8(1 << parseNativeMutationEvent(name))
					if replacementEvents&bit != 0 {
						return fmt.Errorf("stored statement event has multiple replacement producers")
					}
					replacementEvents |= bit
				}
			}
		}
	}
	return nil
}

// Publication/load/drop callers own the containing schema or a private table.
// These are numeric source-owner updates, never callable graph traversals.
func (t *table) retainStatementTriggerMemory(db *database) {
	for _, header := range t.Triggers {
		if header.statementDefinition != nil {
			header.statementDefinition.memory.retain(db)
		}
	}
}

func (t *table) releaseStatementTriggerMemory(db *database) {
	for _, header := range t.Triggers {
		if header.statementDefinition != nil {
			header.statementDefinition.memory.release(db)
		}
	}
}

func (t *table) moveStatementTriggerMemoryOwner(db *database) {
	for _, header := range t.Triggers {
		if header.statementDefinition != nil {
			header.statementDefinition.memory.moveOwner(db)
		}
	}
}
