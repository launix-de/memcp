/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/launix-de/memcp/scm"
)

type failingSequencePersistence struct{ PersistenceEngine }

func (failingSequencePersistence) WriteSchemaWithMode([]byte, bool) {
	panic("sequence reservation persistence failure")
}

func sequenceTestDatabase(t *testing.T) *database {
	t.Helper()
	db := newDatabase()
	db.Name = "sequence-test"
	db.persistence = &FileStorage{path: t.TempDir() + "/"}
	db.srState = SHARED
	db.sequenceEnabled.Store(true)
	db.prepareSequence()
	return db
}

func TestSequenceDurableReservation(t *testing.T) {
	db := sequenceTestDatabase(t)
	first := db.allocateSequence(2)
	if first != 1 {
		t.Fatalf("first range starts at %d", first)
	}
	encoded := []byte(sequenceToken(first).String())
	if len(encoded) != 8 || binary.BigEndian.Uint64(encoded) != first {
		t.Fatalf("invalid binary sequence %x", encoded)
	}
	// Restore from the actual schema file with the normal loader, not from an
	// in-memory copy. A crash/restart may skip numbers but must never reuse them.
	restored := newDatabase()
	restored.persistence = db.persistence
	restored.Name = db.Name
	restored.srState = COLD
	restored.ensureLoaded()
	t.Cleanup(func() { restored.transactionLog.Close() })
	restored.sequenceEnabled.Store(true)
	restored.prepareSequence()
	next := restored.allocateSequence(1)
	if next <= first+1 || next != sequenceReservation+1 {
		t.Fatalf("recovery reused range: %d after %d", next, first)
	}
}

func TestSequenceConcurrentAllocations(t *testing.T) {
	db := sequenceTestDatabase(t)
	const writers, perWriter = 16, 1000
	values := make(chan uint64, writers*perWriter)
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perWriter; j++ {
				values <- db.allocateSequence(1)
			}
		}()
	}
	wg.Wait()
	close(values)
	seen := make(map[uint64]bool)
	for value := range values {
		if value == 0 || seen[value] {
			t.Fatalf("duplicate sequence value %d", value)
		}
		seen[value] = true
	}
	if len(seen) != writers*perWriter {
		t.Fatalf("lost sequence values: %d", len(seen))
	}
}

func TestSequenceFailedReservationDoesNotPublish(t *testing.T) {
	db := sequenceTestDatabase(t)
	db.sequenceNext.Store(sequenceReservation - 1)
	db.persistence = failingSequencePersistence{db.persistence}
	var failed bool
	func() {
		defer func() { failed = recover() != nil }()
		db.prepareSequence()
	}()
	if !failed || db.sequenceLimit.Load() != sequenceReservation {
		t.Fatal("failed schema fsync published an unreserved sequence interval")
	}
	if got := db.allocateSequence(1); got != sequenceReservation {
		t.Fatalf("last previously reserved value lost: %d", got)
	}
	failed = false
	func() {
		defer func() { failed = recover() != nil }()
		db.allocateSequence(1)
	}()
	if !failed {
		t.Fatal("DML consumed an unpersisted sequence")
	}
}

func TestSequenceBinaryWALRoundTrip(t *testing.T) {
	for _, value := range []uint64{0x80, 0xff, 0xff0000ff, ^uint64(0)} {
		original := sequenceToken(value)
		var encoded bytes.Buffer
		encodeFileInsert(&encoded, "insert ", "", []string{"rv"}, [][]scm.Scmer{{original}})
		cols, rows := decodeFileInsertLog(bytes.TrimSpace(bytes.TrimPrefix(encoded.Bytes(), []byte("insert "))))
		if len(cols) != 1 || len(rows) != 1 || rows[0][0].String() != original.String() {
			t.Fatalf("binary sequence %x changed through WAL", value)
		}
	}
}

// A failed load callback must remain a failure after sync.Once has finished;
// neither the first request nor a later concurrent request may use its partial schema.
func TestSchemaInitializerFailureBlocksEveryTableAccess(t *testing.T) {
	db := sequenceTestDatabase(t)
	db.Name = "initializer_failure"
	tbl := db.newTable("items", Safe)
	tbl.createColumnLocked("value", "INT", nil, nil, false)
	db.tables.Set(tbl.Name, tbl)
	db.save()
	original := append([]byte(nil), db.persistence.ReadSchema()...)
	var called atomic.Int64
	name := t.Name()
	schemaInitializers.Lock()
	schemaInitializers.byName[name] = scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		if a[0].Slice()[1].String() != db.Name {
			return scm.NewSlice(nil)
		}
		called.Add(1)
		return scm.NewSlice([]scm.Scmer{
			scm.NewSlice([]scm.Scmer{scm.NewString("default"), scm.NewString("value"), scm.NewInt(7)}),
			scm.NewSlice([]scm.Scmer{scm.NewString("trigger"), scm.NewString("invalid"), scm.NewString("before_insert"), scm.NewString("source"), scm.NewString("unregistered-initializer-language"), scm.NewNil(), scm.NewBool(false)}),
		})
	})
	schemaInitializers.Unlock()
	defer func() {
		schemaInitializers.Lock()
		delete(schemaInitializers.byName, name)
		schemaInitializers.Unlock()
	}()
	restored := newDatabase()
	restored.Name, restored.persistence, restored.srState = db.Name, db.persistence, COLD
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); requirePanic(t, func() { restored.GetTable("items") }) }()
	}
	wg.Wait()
	if called.Load() != 1 || restored.loadFailure == nil {
		t.Fatal("failed initializer did not retain its one-time failure")
	}
	if !bytes.Equal(original, db.persistence.ReadSchema()) {
		t.Fatal("failed initializer published partial schema to disk")
	}
	if restored.transactionLog != nil {
		restored.transactionLog.Close()
	}
}

func TestSequenceExhaustionKeepsLoadedSchemaReadable(t *testing.T) {
	db := sequenceTestDatabase(t)
	db.Name = "sequence_exhausted"
	tbl := db.newTable("items", Safe)
	tbl.createColumnLocked("rv", "BINARY", []int{8}, []scm.Scmer{scm.NewString("default"), sequenceToken(0)}, false)
	tbl.Triggers = []TriggerDescription{
		{Name: "insert", Timing: BeforeInsert, OwnerColumn: "rv", Source: "rv", Language: "exhaustion-test"},
		{Name: "update", Timing: BeforeUpdate, OwnerColumn: "rv", Source: "rv", Language: "exhaustion-test"},
	}
	db.tables.Set(tbl.Name, tbl)
	db.SequenceHighWater = math.MaxUint64
	db.save()
	restored := newDatabase()
	restored.Name, restored.persistence, restored.srState = db.Name, db.persistence, COLD
	if restored.GetTable("items") == nil {
		t.Fatal("sequence exhaustion hid readable schema")
	}
	requirePanic(t, func() { restored.allocateSequence(1) })
	requirePanic(t, func() { restored.prepareSequence() })
	if restored.GetTable("items") == nil {
		t.Fatal("failed exhausted write hid readable schema")
	}
	if restored.transactionLog != nil {
		restored.transactionLog.Close()
	}
}

func TestPrivateCreateTriggerCompilationFailsBeforePublication(t *testing.T) {
	oldBase := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = oldBase }()
	Init(scm.Globalenv)
	schema := "private_trigger_failure"
	CreateDatabase(schema, false)
	defer databases.Remove(schema)
	db := GetDatabase(schema)
	definitions := scm.NewSlice([]scm.Scmer{
		scm.NewSlice([]scm.Scmer{scm.NewString("column"), scm.NewString("id"), scm.NewString("INT"), scm.NewSlice(nil), scm.NewSlice(nil)}),
		scm.NewSlice([]scm.Scmer{scm.NewString("trigger"), scm.NewString("bad"), scm.NewString("before_insert"), scm.NewString("source"), scm.NewString("unregistered-create-language"), scm.NewNil(), scm.NewBool(false)}),
	})
	requirePanic(t, func() {
		scm.Apply(scm.Globalenv.Vars[scm.Symbol("createtable")], scm.NewString(schema), scm.NewString("items"), definitions, scm.NewSlice(nil))
	})
	if db.GetTable("items") != nil {
		t.Fatal("failed trigger compilation published its table")
	}
	if !db.schemalock.TryLock() {
		t.Fatal("failed trigger compilation retained schema lock")
	}
	db.schemalock.Unlock()
	if db.SequenceHighWater != 0 {
		t.Fatal("ordinary failed CREATE performed sequence initialization")
	}
}

func TestColumnOwnedTriggerDropPersistsOnlyItsOwnRemoval(t *testing.T) {
	db := sequenceTestDatabase(t)
	db.Name = "column_owned_trigger"
	tbl := db.newTable("items", Safe)
	for _, name := range []string{"value", "keep"} {
		tbl.createColumnLocked(name, "INT", nil, nil, false)
	}
	language := t.Name()
	registerTriggerLanguage(language, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		return scm.NewSlice([]scm.Scmer{scm.NewSymbol("deferred_trigger"), scm.Read("owned trigger", `(lambda (old new session tx) new)`)})
	}))
	defer func() {
		triggerLanguageCompilers.Lock()
		delete(triggerLanguageCompilers.byName, language)
		triggerLanguageCompilers.Unlock()
	}()
	for _, owner := range []string{"value", "keep"} {
		applyPrivateTableDefinition(tbl, []scm.Scmer{scm.NewString("trigger"), scm.NewString(owner), scm.NewString("before_update"), scm.NewString("return-new"), scm.NewString(language), scm.NewNil(), scm.NewBool(false), scm.NewInt(1), scm.NewString(owner)})
	}
	requirePanic(t, func() {
		applyPrivateTableDefinition(tbl, []scm.Scmer{scm.NewString("trigger"), scm.NewString("bad-owner"), scm.NewString("before_update"), scm.NewString("return-new"), scm.NewString(language), scm.NewNil(), scm.NewBool(false), scm.NewInt(1), scm.NewString("absent")})
	})
	if len(tbl.Triggers) != 2 {
		t.Fatal("invalid owner mutated private trigger definitions")
	}
	db.tables.Set(tbl.Name, tbl)
	db.save()
	restored := newDatabase()
	restored.Name, restored.persistence, restored.srState = db.Name, db.persistence, COLD
	table := restored.GetTable("items")
	if len(table.Triggers) != 2 || table.Triggers[0].OwnerColumn != "value" || table.Triggers[1].OwnerColumn != "keep" {
		t.Fatal("trigger owner metadata did not survive actual restart")
	}
	if !table.dropColumnForMigration("value") {
		t.Fatal("explicit column drop failed")
	}
	if len(table.Columns) != 1 || table.Columns[0].Name != "keep" || len(table.Triggers) != 1 || table.Triggers[0].OwnerColumn != "keep" {
		t.Fatal("drop changed an unrelated column or owned trigger")
	}
	var snapshot database
	if err := json.Unmarshal(restored.persistence.ReadSchema(), &snapshot); err != nil {
		t.Fatal(err)
	}
	saved := snapshot.tables.Get("items")
	if len(saved.Triggers) != 1 || saved.Triggers[0].OwnerColumn != "keep" || len(saved.Columns) != 1 {
		t.Fatal("drop did not persist the same column/owned-trigger mutation")
	}
	if restored.transactionLog != nil {
		restored.transactionLog.Close()
	}
}
