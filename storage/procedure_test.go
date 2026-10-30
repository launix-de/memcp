/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"encoding/json"
	"math"
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
	if got := len(ShowProcedures(schema, "p").Slice()); got != 1 {
		t.Fatalf("named source definition lookup returned %d rows", got)
	}
	if got := len(ShowProcedures(schema, "absent").Slice()); got != 0 {
		t.Fatalf("absent source definition lookup returned %d rows", got)
	}
	for name, operation := range map[string]func(){
		"duplicate":        func() { CreateProcedure(schema, "p", "procedure-catalog-test", "duplicate", scm.NewNil(), false) },
		"unknown-language": func() { CreateProcedure(schema, "unknown", "absent-language", "source", scm.NewNil(), false) },
		"missing-name":     func() { CreateProcedure(schema, "", "procedure-catalog-test", "source", scm.NewNil(), false) },
		"missing-source":   func() { CreateProcedure(schema, "empty", "procedure-catalog-test", "", scm.NewNil(), false) },
		"missing-drop":     func() { DropProcedure(schema, "absent", false) },
		"nonportable-parameters": func() {
			CreateProcedure(schema, "invalid-data", "procedure-catalog-test", "source",
				scm.NewSlice([]scm.Scmer{scm.NewAny(NewTxContext(TxCursorStability))}), false)
		},
		"nonfinite-parameters": func() {
			CreateProcedure(schema, "invalid-number", "procedure-catalog-test", "source",
				scm.NewFloat(math.NaN()), false)
		},
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
	if got := len(ShowProcedures(schema, "").Slice()); got != 1 {
		t.Fatalf("failed DDL published %d procedures", got)
	}
	oldCallable := Procedure(schema, "p", false)
	original := GetDatabase(schema)
	restored := newDatabase()
	restored.Name, restored.persistence, restored.srState = schema, original.persistence, COLD
	restored.ensureLoaded()
	if entry := restored.procedures.Get("p"); entry == nil || entry.Source != "first" || entry.compiled.Load() != nil {
		t.Fatal("schema checkpoint lost authoritative source or retained runtime code")
	}
	CreateProcedure(schema, "p", "procedure-catalog-test", "replacement", scm.NewSlice(nil), true)
	if got := scm.Apply(Procedure(schema, "p", false)).String(); got != "replacement" {
		t.Fatalf("replacement lookup = %q", got)
	}
	if got := scm.Apply(oldCallable).String(); got != "first" {
		t.Fatalf("replacement mutated an already-owned invocation: %q", got)
	}
	DropProcedure(schema, "p", false)
	if !ShowProcedures(schema, "").IsSlice() || len(ShowProcedures(schema, "").Slice()) != 0 {
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
	if got := scm.Apply(Procedure(schema, "existing", false)).String(); got != "existing-source" {
		t.Fatalf("cold procedure lookup = %q", got)
	}
	checkTable()
	cold()
	if got := len(ShowProcedures(schema, "").Slice()); got != 1 {
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
	if got := scm.Apply(Procedure(schema, "added", false)).String(); got != "acknowledged-source" {
		t.Fatalf("acknowledged CREATE did not survive restart: %q", got)
	}
	if got := scm.Apply(Procedure(schema, "existing", false)).String(); got != "existing-source" {
		t.Fatalf("CREATE lost existing definition: %q", got)
	}
	checkTable()
	cold()
	DropProcedure(schema, "added", false)
	checkTable()
	cold()
	if got := len(ShowProcedures(schema, "").Slice()); got != 1 {
		t.Fatalf("cold DROP did not survive restart: %d definitions", got)
	}
	checkTable()
}

func TestProcedureParametersFreezePortableDataAndExactIntegers(t *testing.T) {
	original := scm.NewSlice([]scm.Scmer{scm.NewString("largest"), scm.NewInt(9223372036854775807),
		scm.NewSlice([]scm.Scmer{scm.NewString("nested"), scm.NewBool(true)})})
	frozen := freezeProcedureParameters(original)
	original.Slice()[1] = scm.NewInt(0)
	original.Slice()[2].Slice()[1] = scm.NewBool(false)
	if got := frozen.Slice()[1].Int(); got != 9223372036854775807 {
		t.Fatalf("frozen declaration lost exact integer carrier: %d", got)
	}
	if !frozen.Slice()[2].Slice()[1].Bool() {
		t.Fatal("caller mutation changed frozen declaration")
	}
	declarations := scm.NewFastDictValue(1)
	declarations.Set(scm.NewString("limit"), scm.NewInt(9223372036854775807), nil)
	frozenMap := freezeProcedureParameters(scm.NewFastDict(declarations))
	declarations.Set(scm.NewString("limit"), scm.NewInt(0), nil)
	if got := frozenMap.Slice()[1].Int(); got != 9223372036854775807 {
		t.Fatalf("dictionary freeze lost integer: %d", got)
	}
	for _, invalid := range []scm.Scmer{scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewNil() }),
		scm.NewAny(NewTxContext(TxCursorStability)), scm.NewFloat(math.NaN()), scm.NewFloat(math.Inf(1))} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("nonportable procedure declaration was accepted")
				}
			}()
			freezeProcedureParameters(scm.NewSlice([]scm.Scmer{invalid}))
		}()
	}
}

func TestProcedureCompilationDoesNotPublishIntoReplacedDatabase(t *testing.T) {
	previousBasepath := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = previousBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	const schema = "procedure-replacement-test"
	CreateDatabase(schema, false)
	defer databases.Remove(schema)
	started, resume := make(chan struct{}), make(chan struct{})
	registerTriggerLanguage("procedure-replacement-test", scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		close(started)
		<-resume
		return scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewInt(7) })
	}))
	completed := make(chan any, 1)
	go func() {
		defer func() { completed <- recover() }()
		CreateProcedure(schema, "retired", "procedure-replacement-test", "source", scm.NewNil(), false)
	}()
	<-started
	DropDatabase(schema, false)
	CreateDatabase(schema, false)
	close(resume)
	if failure := <-completed; failure == nil {
		t.Fatal("source compilation acknowledged publication into a replaced database")
	}
	if len(ShowProcedures(schema, "").Slice()) != 0 {
		t.Fatal("retired definition entered the replacement catalog")
	}
}

func TestProcedureDeclarationSnapshotSurvivesConcurrentReplacement(t *testing.T) {
	previousBasepath := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = previousBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	const schema = "procedure-snapshot-test"
	CreateDatabase(schema, false)
	defer databases.Remove(schema)
	registerTriggerLanguage("procedure-snapshot-test", scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		source := a[0]
		return scm.NewFunc(func(...scm.Scmer) scm.Scmer { return source })
	}))
	CreateProcedure(schema, "p", "procedure-snapshot-test", "first",
		scm.NewSlice([]scm.Scmer{scm.NewString("first-contract")}), false)
	snapshot := Procedure(schema, "p", true).Slice()
	callable, declaration := snapshot[1], snapshot[3].Slice()
	if declaration[1].String() != "p" || declaration[3].String() != "procedure-snapshot-test" ||
		declaration[5].String() != "first" || declaration[7].Slice()[0].String() != "first-contract" {
		t.Fatal("snapshot did not retain its complete immutable declaration")
	}
	declaration[7].Slice()[0] = scm.NewString("caller-mutation")
	if got := ShowProcedures(schema, "p").Slice()[0].Slice()[7].Slice()[0].String(); got != "first-contract" {
		t.Fatal("snapshot parameters borrowed mutable catalog data")
	}
	replaced := make(chan any, 1)
	go func() {
		defer func() { replaced <- recover() }()
		CreateProcedure(schema, "p", "procedure-snapshot-test", "second",
			scm.NewSlice([]scm.Scmer{scm.NewString("second-contract")}), true)
	}()
	if failure := <-replaced; failure != nil {
		t.Fatalf("concurrent replacement failed: %v", failure)
	}
	if got := scm.Apply(callable).String(); got != "first" {
		t.Fatalf("binding first declaration invoked a different source: %q", got)
	}
	latest := Procedure(schema, "p", true).Slice()
	if got := scm.Apply(latest[1]).String(); got != "second" || latest[3].Slice()[7].Slice()[0].String() != "second-contract" {
		t.Fatal("new lookup did not observe the complete replacement definition")
	}
	DropProcedure(schema, "p", false)
	if got := scm.Apply(callable).String(); got != "first" {
		t.Fatal("dropping a catalog name invalidated its retained invocation")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("snapshot lookup acknowledged a missing procedure")
		}
	}()
	Procedure(schema, "p", true)
}
