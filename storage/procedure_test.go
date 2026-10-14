/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func TestProcedureSourceSurvivesSchemaRoundtripWithoutCode(t *testing.T) {
	var compilations atomic.Int64
	registerTriggerLanguage("procedure-roundtrip-test", scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		compilations.Add(1)
		source := a[0].String()
		return scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			return scm.NewString(source + ":" + values[0].String())
		})
	}))
	db := newDatabase()
	db.Name = "procedure-roundtrip"
	definition := &ProcedureDescription{Name: "calculate", Source: "source-only", Language: "procedure-roundtrip-test",
		Parameters: scm.NewSlice([]scm.Scmer{scm.NewString("input"), scm.NewInt(7)})}
	db.procedures.Set(definition.Name, definition)
	if got := scm.Apply(definition.callable(db.Name), scm.NewString("first")).String(); got != "source-only:first" {
		t.Fatalf("first invocation = %q", got)
	}
	encoded, err := json.Marshal(db)
	if err != nil {
		t.Fatal(err)
	}
	var fields struct {
		Procedures []map[string]json.RawMessage `json:"procedures"`
	}
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields.Procedures) != 1 || len(fields.Procedures[0]) != 4 {
		t.Fatalf("schema persisted runtime procedure fields: %s", encoded)
	}
	var restored database
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	entry := restored.procedures.Get("calculate")
	if entry == nil || entry.compiled.Load() != nil {
		t.Fatal("schema load retained executable code instead of source")
	}
	if got := scm.Apply(entry.callable(restored.Name), scm.NewString("second")).String(); got != "source-only:second" {
		t.Fatalf("regenerated invocation = %q", got)
	}
	if compilations.Load() != 2 {
		t.Fatalf("expected one compilation per process-local definition, got %d", compilations.Load())
	}
}

func TestProcedureCompilationPublishesOnceUnderConcurrency(t *testing.T) {
	var compilations atomic.Int64
	registerTriggerLanguage("procedure-concurrent-test", scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		compilations.Add(1)
		return scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })
	}))
	definition := &ProcedureDescription{Name: "identity", Source: "source", Language: "procedure-concurrent-test"}
	var callers sync.WaitGroup
	for i := 0; i < 24; i++ {
		callers.Add(1)
		go func(value int64) {
			defer callers.Done()
			if got := scm.Apply(definition.callable("concurrent"), scm.NewInt(value)).Int(); got != value {
				t.Errorf("callable captured another invocation: got %d, want %d", got, value)
			}
		}(int64(i))
	}
	callers.Wait()
	if compilations.Load() != 1 {
		t.Fatalf("concurrent source compiled %d times", compilations.Load())
	}
}

func TestProcedureSchemaSupportsOlderCatalogAndRejectsDuplicateDefinitions(t *testing.T) {
	var db database
	if err := json.Unmarshal([]byte(`{"name":"old","tables":{}}`), &db); err != nil {
		t.Fatal(err)
	}
	if db.procedures == nil || len(db.procedures.GetAll()) != 0 {
		t.Fatal("old schema did not restore an empty procedure directory")
	}
	invalid := `{"name":"invalid","tables":{},"procedures":[{"name":"p","source":"body","language":"example"},{"name":"p","source":"other","language":"example"}]}`
	if err := json.Unmarshal([]byte(invalid), &db); err == nil {
		t.Fatal("duplicate stored procedure definition was accepted")
	}
}

func TestProcedureCatalogDDLCheckpointReloadAndReplace(t *testing.T) {
	previousBasepath := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = previousBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	const schema = "procedure-catalog-test"
	CreateDatabase(schema, false)
	defer databases.Remove(schema)
	registerTriggerLanguage("procedure-catalog-test", scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		source := a[0]
		return scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return source })
	}))
	CreateProcedure(schema, "p", "procedure-catalog-test", "first", scm.NewSlice(nil), false)
	for name, operation := range map[string]func(){
		"duplicate":        func() { CreateProcedure(schema, "p", "procedure-catalog-test", "duplicate", scm.NewNil(), false) },
		"unknown-language": func() { CreateProcedure(schema, "unknown", "absent-language", "source", scm.NewNil(), false) },
		"missing-name":     func() { CreateProcedure(schema, "", "procedure-catalog-test", "source", scm.NewNil(), false) },
		"missing-source":   func() { CreateProcedure(schema, "empty", "procedure-catalog-test", "", scm.NewNil(), false) },
		"missing-drop":     func() { DropProcedure(schema, "absent", false) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid procedure operation did not fail")
				}
			}()
			operation()
		})
	}
	if got := len(ShowProcedures(schema).Slice()); got != 1 {
		t.Fatalf("failed DDL published %d procedures", got)
	}
	oldCallable := Procedure(schema, "p")
	original := GetDatabase(schema)
	restored := newDatabase()
	restored.Name, restored.persistence, restored.srState = schema, original.persistence, COLD
	restored.ensureLoaded()
	if entry := restored.procedures.Get("p"); entry == nil || entry.Source != "first" || entry.compiled.Load() != nil {
		t.Fatal("schema checkpoint lost authoritative source or retained runtime code")
	}
	CreateProcedure(schema, "p", "procedure-catalog-test", "replacement", scm.NewSlice(nil), true)
	if got := scm.Apply(Procedure(schema, "p")).String(); got != "replacement" {
		t.Fatalf("replacement lookup = %q", got)
	}
	if got := scm.Apply(oldCallable).String(); got != "first" {
		t.Fatalf("replacement mutated an already-owned invocation: %q", got)
	}
	DropProcedure(schema, "p", false)
	if !ShowProcedures(schema).IsSlice() || len(ShowProcedures(schema).Slice()) != 0 {
		t.Fatal("dropped procedure is still listed")
	}
	if DropProcedure(schema, "p", true) {
		t.Fatal("DROP IF EXISTS reported a removed definition twice")
	}
}

func TestProcedureCompilerUsesOrdinarySchemeClosureAndJITBoundary(t *testing.T) {
	registerTriggerLanguage("procedure-expression-test", scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		return scm.Eval(scm.Read("stored procedure source", a[0].String()), &scm.Globalenv)
	}))
	definition := &ProcedureDescription{Name: "add", Source: "(lambda (value) (+ value 7))", Language: "procedure-expression-test"}
	callable := definition.callable("generic")
	if got := scm.Apply(callable, scm.NewInt(5)).Int(); got != 12 {
		t.Fatalf("ordinary compiled source callable = %d, want 12", got)
	}
}

func TestProcedurePublicAPIsLoadColdCatalogBeforeAccess(t *testing.T) {
	previousBasepath := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = previousBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	const schema = "procedure-cold-api-test"
	CreateDatabase(schema, false)
	defer databases.Remove(schema)
	CreateTable(schema, "preserved", Safe, false)
	// A constant source compiler deliberately performs no catalog lookup.
	// Public procedure APIs themselves must load a restored database.
	registerTriggerLanguage("procedure-cold-api-test", scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		source := a[0]
		return scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return source })
	}))
	CreateProcedure(schema, "existing", "procedure-cold-api-test", "existing-source", scm.NewNil(), false)
	backend := GetDatabase(schema).persistence
	cold := func() {
		db := newDatabase()
		db.Name, db.persistence, db.srState = schema, backend, COLD
		databases.Set(schema, db)
	}
	checkTable := func() {
		if GetDatabase(schema).tables.Get("preserved") == nil {
			t.Fatal("procedure catalog operation lost an existing table")
		}
	}
	cold()
	if got := scm.Apply(Procedure(schema, "existing")).String(); got != "existing-source" {
		t.Fatalf("cold procedure lookup = %q", got)
	}
	checkTable()
	cold()
	if got := len(ShowProcedures(schema).Slice()); got != 1 {
		t.Fatalf("cold procedure listing has %d definitions", got)
	}
	checkTable()
	cold()
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("cold duplicate CREATE was accepted")
			}
		}()
		CreateProcedure(schema, "existing", "procedure-cold-api-test", "wrong", scm.NewNil(), false)
	}()
	checkTable()
	cold()
	CreateProcedure(schema, "added", "procedure-cold-api-test", "acknowledged-source", scm.NewNil(), false)
	checkTable()
	cold()
	if got := scm.Apply(Procedure(schema, "added")).String(); got != "acknowledged-source" {
		t.Fatalf("acknowledged CREATE did not survive restart: %q", got)
	}
	if got := scm.Apply(Procedure(schema, "existing")).String(); got != "existing-source" {
		t.Fatalf("CREATE lost existing definition: %q", got)
	}
	checkTable()
	cold()
	DropProcedure(schema, "added", false)
	checkTable()
	cold()
	if got := len(ShowProcedures(schema).Slice()); got != 1 {
		t.Fatalf("cold DROP did not survive restart: %d definitions", got)
	}
	checkTable()
}
