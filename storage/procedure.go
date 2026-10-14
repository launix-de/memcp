/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

// ProcedureDescription owns one immutable source definition. Parameters are
// language-owned declaration data; storage neither interprets them nor derives
// types from their values. Only source and declarations survive a restart.
// compiled is a process-local artifact published once, outside catalog locks.
// An invocation owns its session, transaction and result callbacks separately.
type ProcedureDescription struct {
	Name       string    `json:"name"`
	Source     string    `json:"source"`
	Language   string    `json:"language"`
	Parameters scm.Scmer `json:"parameters"`
	compile    sync.Once
	compiled   atomic.Pointer[scm.Scmer]
	compileErr any
}

// ComputeSize includes the retained declaration and its transient callable.
// Catalog diagnostics own this traversal; query and metadata lookup do not.
func (p *ProcedureDescription) ComputeSize() uint {
	size := uint(unsafe.Sizeof(*p)) + uint(len(p.Name)+len(p.Source)+len(p.Language))
	size += scm.ComputeSize(p.Parameters) - uint(unsafe.Sizeof(p.Parameters))
	if fn := p.compiled.Load(); fn != nil {
		size += scm.ComputeSize(*fn)
	}
	return size
}

func (p *ProcedureDescription) callable(schema string) scm.Scmer {
	if fn := p.compiled.Load(); fn != nil {
		return *fn
	}
	compiler, ok := triggerLanguageCompiler(p.Language)
	if !ok {
		panic(fmt.Sprintf("procedure %s.%s requires unregistered language %q", schema, p.Name, p.Language))
	}
	p.compile.Do(func() {
		defer func() { p.compileErr = recover() }()
		context := scm.NewFastDictValue(3)
		context.Set(scm.NewString("schema"), scm.NewString(schema), nil)
		context.Set(scm.NewString("name"), scm.NewString(p.Name), nil)
		context.Set(scm.NewString("parameters"), scm.CloneOptimizerExpression(p.Parameters), nil)
		fn := scm.Apply(compiler, scm.NewString(p.Source), scm.NewFastDict(context))
		if !fn.IsNativeFunc() && !fn.IsProc() && !fn.IsJIT() {
			panic("procedure source compiler must return a callable")
		}
		// The same closure and native compilation boundary as source triggers.
		// Compiler-generated callables must use the ordinary plan/schema guards;
		// they may retain source, never request-owned bindings or physical rows.
		fn = scm.CompileJIT(scm.CloseProcedure(fn), true)
		p.compiled.Store(&fn)
	})
	if p.compileErr != nil {
		panic(p.compileErr)
	}
	return *p.compiled.Load()
}

func CreateProcedure(schema, name, language, source string, parameters scm.Scmer, replace bool) bool {
	db := GetDatabase(schema)
	if db == nil {
		panic("database does not exist: " + schema)
	}
	db.ensureLoaded()
	if name == "" || language == "" || source == "" {
		panic("procedure name, language and source must not be empty")
	}
	definition := &ProcedureDescription{Name: name, Language: language, Source: source,
		Parameters: scm.CloneOptimizerExpression(parameters)}
	// Validate and compile before publication, without catalog ownership. A
	// compiler may inspect the existing schema through ordinary metadata APIs.
	definition.callable(schema)
	db.persistenceLifecycle.RLock()
	defer db.persistenceLifecycle.RUnlock()
	db.schemalock.Lock()
	if db.procedures.Get(name) != nil && !replace {
		db.schemalock.Unlock()
		panic("procedure already exists: " + name)
	}
	db.procedures.Set(name, definition)
	db.saveLockedAndUnlock(schemaSaveFsync)
	return true
}

func DropProcedure(schema, name string, ifexists bool) bool {
	db := GetDatabase(schema)
	if db == nil {
		panic("database does not exist: " + schema)
	}
	db.ensureLoaded()
	db.persistenceLifecycle.RLock()
	defer db.persistenceLifecycle.RUnlock()
	db.schemalock.Lock()
	if db.procedures.Get(name) == nil {
		db.schemalock.Unlock()
		if ifexists {
			return false
		}
		panic("procedure does not exist: " + name)
	}
	db.procedures.Remove(name)
	db.saveLockedAndUnlock(schemaSaveFsync)
	return true
}

func Procedure(schema, name string) scm.Scmer {
	db := GetDatabase(schema)
	if db == nil {
		panic("database does not exist: " + schema)
	}
	db.ensureLoaded()
	definition := db.procedures.Get(name)
	if definition == nil {
		panic("procedure does not exist: " + name)
	}
	return definition.callable(schema)
}

func ShowProcedures(schema string) scm.Scmer {
	db := GetDatabase(schema)
	if db == nil {
		panic("database does not exist: " + schema)
	}
	db.ensureLoaded()
	definitions := db.procedures.GetAll()
	rows := make([]scm.Scmer, 0, len(definitions))
	for _, definition := range definitions {
		rows = append(rows, scm.NewSlice([]scm.Scmer{
			scm.NewString("name"), scm.NewString(definition.Name),
			scm.NewString("language"), scm.NewString(definition.Language),
			scm.NewString("source"), scm.NewString(definition.Source),
			scm.NewString("parameters"), scm.CloneOptimizerExpression(definition.Parameters),
		}))
	}
	sort.Slice(rows, func(i, j int) bool { return scm.String(rows[i].Slice()[1]) < scm.String(rows[j].Slice()[1]) })
	return scm.NewSlice(rows)
}

func initProcedureBuiltins(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "registerprocedurelanguage", Fn: func(a ...scm.Scmer) scm.Scmer {
		// Source triggers and procedures share one language compiler registry.
		registerTriggerLanguage(a[0].String(), a[1])
		return scm.NewBool(true)
	}, Type: &scm.TypeDescriptor{Kind: "func", Description: "registers a source compiler (source context) for stored procedures", HasSideEffects: true,
		Params: []*scm.TypeDescriptor{{Kind: "string", Label: "language"}, {Kind: "func", Label: "compiler"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "createprocedure", Fn: func(a ...scm.Scmer) scm.Scmer {
		return scm.NewBool(CreateProcedure(a[0].String(), a[1].String(), a[2].String(), a[3].String(), a[4], len(a) > 5 && a[5].Bool()))
	}, Type: &scm.TypeDescriptor{Kind: "func", Description: "persists a language-owned procedure source definition", HasSideEffects: true,
		Params: []*scm.TypeDescriptor{{Kind: "string", Label: "schema"}, {Kind: "string", Label: "name"}, {Kind: "string", Label: "language"}, {Kind: "string", Label: "source"}, {Kind: "any", Label: "parameters"}, {Kind: "bool", Label: "replace", Optional: true}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "dropprocedure", Fn: func(a ...scm.Scmer) scm.Scmer {
		return scm.NewBool(DropProcedure(a[0].String(), a[1].String(), len(a) > 2 && a[2].Bool()))
	}, Type: &scm.TypeDescriptor{Kind: "func", Description: "drops a stored procedure definition", HasSideEffects: true,
		Params: []*scm.TypeDescriptor{{Kind: "string", Label: "schema"}, {Kind: "string", Label: "name"}, {Kind: "bool", Label: "if_exists", Optional: true}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "procedure", Fn: func(a ...scm.Scmer) scm.Scmer {
		return Procedure(a[0].String(), a[1].String())
	}, Type: &scm.TypeDescriptor{Kind: "func", Description: "resolves and regenerates a stored source procedure", HasSideEffects: true,
		Params: []*scm.TypeDescriptor{{Kind: "string", Label: "schema"}, {Kind: "string", Label: "name"}}, Return: &scm.TypeDescriptor{Kind: "func"}}})
	scm.Declare(en, &scm.Declaration{Name: "showprocedures", Fn: func(a ...scm.Scmer) scm.Scmer {
		return ShowProcedures(a[0].String())
	}, Type: &scm.TypeDescriptor{Kind: "func", Description: "lists native stored procedure source declarations",
		Params: []*scm.TypeDescriptor{{Kind: "string", Label: "schema"}}, Return: &scm.TypeDescriptor{Kind: "list"}}})
}
