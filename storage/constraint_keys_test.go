/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "sync"
import "testing"
import "bytes"
import "github.com/launix-de/memcp/scm"

func keyProjectionFixture(t *testing.T, name string, unique bool) *table {
	t.Helper()
	old := Basepath
	Basepath = t.TempDir()
	t.Cleanup(func() { Basepath = old })
	Init(scm.Globalenv)
	CreateDatabase(name, false)
	t.Cleanup(func() { databases.Remove(name) })
	tbl, _ := CreateTable(name, "items", Safe, false)
	recipe := scm.EvalAll("key recipe", `(lambda (value) (substr value 0 4))`, &scm.Globalenv)
	attrs := []scm.Scmer{scm.NewString("key_projection"), recipe}
	if unique {
		attrs = append(attrs, scm.NewString("unique"), scm.NewBool(true))
	}
	if !tbl.CreateColumn("key", "ANY", nil, attrs) {
		t.Fatal("key declaration failed")
	}
	tbl.CreateColumn("value", "INT", nil, nil)
	return tbl
}

func TestConstraintProjectionUniqueMainDeltaAndRecovery(t *testing.T) {
	tbl := keyProjectionFixture(t, "constraint_projection_recovery", true)
	insert := func(key string) {
		tbl.Insert([]string{"key", "value"}, [][]scm.Scmer{{scm.NewString(key), scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
	}
	insert("aaaa:original")
	requirePanic(t, func() { insert("aaaa:different") })
	RebuildTable(tbl, true, false)
	requirePanic(t, func() { insert("aaaa:rebuilt") })
	insert("bbbb:retained")
	before := tbl.Count()
	restored := reloadTableFromPersistence(t, tbl.schema.Name, tbl.schema.persistence)
	requirePanic(t, func() {
		restored.Insert([]string{"key"}, [][]scm.Scmer{{scm.NewString("aaaa:recovered")}}, nil, scm.NewNil(), false, nil)
	})
	if restored.Count() != before {
		t.Fatal("rejected canonical duplicate changed rows")
	}
	for _, shard := range restored.ActiveShards() {
		release := shard.GetRead(nil)
		reader := shard.ColumnReaderTx(nil, "key", false)
		shard.mu.RLock()
		for i := uint32(0); i < shard.main_count+uint32(len(shard.inserts)); i++ {
			value := reader(i).String()
			if value != "aaaa:original" && value != "bbbb:retained" {
				t.Fatal("canonical key replaced stored payload", value)
			}
		}
		shard.mu.RUnlock()
		release()
	}
}

func TestConstraintProjectionConcurrentUniqueWrites(t *testing.T) {
	tbl := keyProjectionFixture(t, "constraint_projection_concurrent", true)
	var writers sync.WaitGroup
	successes := make(chan bool, 2)
	for _, value := range []string{"same:first", "same:second"} {
		writers.Add(1)
		go func(value string) {
			defer writers.Done()
			succeeded := false
			func() {
				defer func() { recover() }()
				tbl.Insert([]string{"key"}, [][]scm.Scmer{{scm.NewString(value)}}, nil, scm.NewNil(), false, nil)
				succeeded = true
			}()
			successes <- succeeded
		}(value)
	}
	writers.Wait()
	close(successes)
	count := 0
	for ok := range successes {
		if ok {
			count++
		}
	}
	if count != 1 || tbl.Count() != 1 {
		t.Fatal("canonical unique writes were not atomic", count, tbl.Count())
	}
}

func TestConstraintProjectionValidatesExistingUniqueRows(t *testing.T) {
	tbl := keyProjectionFixture(t, "constraint_projection_existing", false)
	tbl.Insert([]string{"key"}, [][]scm.Scmer{{scm.NewString("same:first")}, {scm.NewString("same:second")}}, nil, scm.NewNil(), false, nil)
	requirePanic(t, func() { createTableKey(tbl, "handle", []string{"key"}, nil, scm.NewNil(), false) })
	if len(tbl.Unique) != 0 || tbl.Count() != 2 {
		t.Fatal("failed key publication changed schema or rows")
	}
}

func TestConstraintProjectionReferenceAndCascade(t *testing.T) {
	parent := keyProjectionFixture(t, "constraint_projection_dependency", true)
	child, _ := CreateTable(parent.schema.Name, "child", Safe, false)
	recipe := *parent.Columns[0].KeyProjection
	child.CreateColumn("key", "ANY", nil, []scm.Scmer{scm.NewString("key_projection"), recipe})
	child.Insert([]string{"key"}, [][]scm.Scmer{{scm.NewString("same:child")}, {scm.NewString("keep:child")}}, nil, scm.NewNil(), false, nil)
	parent.Insert([]string{"key"}, [][]scm.Scmer{{scm.NewString("same:parent")}}, nil, scm.NewNil(), false, nil)
	if !fkExistenceCheck(nil, parent, []string{"key"}, []scm.Scmer{scm.NewString("same:other")}) || fkExistenceCheck(nil, parent, []string{"key"}, []scm.Scmer{scm.NewString("none:other")}) {
		t.Fatal("reference did not use canonical key")
	}
	fkCascadeDelete(nil, child, []string{"key"}, []scm.Scmer{scm.NewString("same:parent")})
	if child.Count() != 1 || !fkExistenceCheck(nil, child, []string{"key"}, []scm.Scmer{scm.NewString("keep:other")}) {
		t.Fatal("cascade did not target canonical key")
	}
}

func TestConstraintEmptyKeyEqualityIsExplicit(t *testing.T) {
	for _, equal := range []bool{false, true} {
		t.Run(map[bool]string{false: "distinct", true: "equal"}[equal], func(t *testing.T) {
			old := Basepath
			Basepath = t.TempDir()
			defer func() { Basepath = old }()
			Init(scm.Globalenv)
			CreateDatabase("empty_keys", false)
			defer databases.Remove("empty_keys")
			tbl, _ := CreateTable("empty_keys", "items", Safe, false)
			tbl.CreateColumn("key", "ANY", nil, []scm.Scmer{scm.NewString("unique"), scm.NewBool(true), scm.NewString("key_nulls_equal"), scm.NewBool(equal)})
			insert := func(value scm.Scmer) {
				tbl.Insert([]string{"key"}, [][]scm.Scmer{{value}}, nil, scm.NewNil(), false, nil)
			}
			insert(scm.NewNil())
			insert(scm.NewInt(0))
			if equal {
				requirePanic(t, func() { insert(scm.NewNil()) })
			} else {
				insert(scm.NewNil())
			}
			RebuildTable(tbl, true, false)
			if equal {
				requirePanic(t, func() { insert(scm.NewNil()) })
			} else {
				insert(scm.NewNil())
			}
			restored := reloadTableFromPersistence(t, tbl.schema.Name, tbl.schema.persistence)
			if restored.Unique[0].NullsEqual != equal {
				t.Fatal("key empty-value policy lost on restart")
			}
			if equal {
				requirePanic(t, func() { restored.Insert([]string{"key"}, [][]scm.Scmer{{scm.NewNil()}}, nil, scm.NewNil(), false, nil) })
			}
		})
	}
}

func TestConstraintEmptyKeyValidationBeforePublication(t *testing.T) {
	old := Basepath
	Basepath = t.TempDir()
	defer func() { Basepath = old }()
	Init(scm.Globalenv)
	CreateDatabase("empty_key_ddl", false)
	defer databases.Remove("empty_key_ddl")
	tbl, _ := CreateTable("empty_key_ddl", "items", Safe, false)
	tbl.CreateColumn("key", "ANY", nil, nil)
	tbl.Insert([]string{"key"}, [][]scm.Scmer{{scm.NewNil()}, {scm.NewNil()}}, nil, scm.NewNil(), false, nil)
	requirePanic(t, func() { createTableKey(tbl, "equal", []string{"key"}, nil, scm.NewNil(), true) })
	if len(tbl.Unique) != 0 || tbl.Count() != 2 {
		t.Fatal("invalid equal-empty key published")
	}
	if !createTableKey(tbl, "distinct", []string{"key"}, nil, scm.NewNil(), false) {
		t.Fatal("ordinary distinct-empty key rejected")
	}
}

func TestSchemaInitializerUniqueKeyValidatesPrivateRows(t *testing.T) {
	for _, duplicates := range []bool{false, true} {
		t.Run(map[bool]string{false: "valid", true: "duplicates"}[duplicates], func(t *testing.T) {
			tbl := keyProjectionFixture(t, "private_key_initializer", false)
			second := "bbbb:second"
			if duplicates {
				second = "aaaa:second"
			}
			tbl.Insert([]string{"key"}, [][]scm.Scmer{{scm.NewString("aaaa:first")}, {scm.NewString(second)}}, nil, scm.NewNil(), false, nil)
			before := append([]byte(nil), tbl.schema.persistence.ReadSchema()...)
			definition := scm.NewSlice([]scm.Scmer{scm.NewString("unique"), scm.NewString("handle"), scm.NewSlice([]scm.Scmer{scm.NewString("key")}), scm.NewBool(false)})
			name := t.Name()
			schemaInitializers.Lock()
			schemaInitializers.byName[name] = scm.NewFunc(func(context ...scm.Scmer) scm.Scmer {
				if context[0].Slice()[1].String() != tbl.schema.Name || context[0].Slice()[3].String() != tbl.Name {
					return scm.NewSlice(nil)
				}
				return scm.NewSlice([]scm.Scmer{definition})
			})
			schemaInitializers.Unlock()
			defer func() {
				schemaInitializers.Lock()
				delete(schemaInitializers.byName, name)
				schemaInitializers.Unlock()
			}()
			if duplicates {
				requirePanic(t, func() { reloadTableFromPersistence(t, tbl.schema.Name, tbl.schema.persistence) })
				if !bytes.Equal(before, tbl.schema.persistence.ReadSchema()) {
					t.Fatal("rejected private key published partial schema")
				}
				return
			}
			restored := reloadTableFromPersistence(t, tbl.schema.Name, tbl.schema.persistence)
			if len(restored.Unique) != 1 || restored.Unique[0].Id != "handle" {
				t.Fatal("private key was not published")
			}
			applyPrivateTableDefinition(restored, definition.Slice())
			if len(restored.Unique) != 1 {
				t.Fatal("idempotent private key duplicated metadata")
			}
			mismatch := append([]scm.Scmer(nil), definition.Slice()...)
			mismatch[3] = scm.NewBool(true)
			requirePanic(t, func() { applyPrivateTableDefinition(restored, mismatch) })
			requirePanic(t, func() {
				restored.Insert([]string{"key"}, [][]scm.Scmer{{scm.NewString("aaaa:rejected")}}, nil, scm.NewNil(), false, nil)
			})
		})
	}
}
