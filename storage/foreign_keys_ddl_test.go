/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "testing"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

func strictForeignKeyFixture(t *testing.T) (*database, *table, *table, *TxContext) {
	t.Helper()
	oldBasepath := Basepath
	Basepath = t.TempDir()
	Init(scm.Globalenv)
	schema := "strict_foreign_key_test"
	CreateDatabase(schema, false)
	db := GetDatabase(schema)
	child, _ := CreateTable(schema, "child", Memory, false)
	parent, _ := CreateTable(schema, "parent", Memory, false)
	for _, tbl := range []*table{child, parent} {
		tbl.CreateColumn("id", "INT", nil, nil)
	}
	parent.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	tx := NewTxContext(TxCursorStability)
	tx.SessionState = &scm.SessionState{}
	t.Cleanup(func() { databases.Remove(schema); Basepath = oldBasepath })
	return db, child, parent, tx
}
func assertStrictForeignKeyUnlocked(t *testing.T, db *database, tables ...*table) {
	t.Helper()
	if !db.schemalock.TryLock() {
		t.Fatal("foreign key failure retained schema lock")
	}
	db.schemalock.Unlock()
	for _, tbl := range tables {
		if !tbl.ddlMu.TryLock() {
			t.Fatal("foreign key failure retained table DDL lock")
		}
		tbl.ddlMu.Unlock()
		if tbl.hasTableLock() {
			t.Fatal("foreign key failure retained table WRITE lock")
		}
		if len(tbl.Foreign) != 0 || len(tbl.Triggers) != 0 {
			t.Fatal("foreign key failure published metadata or triggers")
		}
	}
}
func strictForeignKeyFixtureKey() foreignKey {
	return foreignKey{Id: "FK_test", Cols1: []string{"id"}, Cols2: []string{"id"}}
}

func TestStrictForeignKeyRejectsUnknownMainEstimate(t *testing.T) {
	db, child, parent, tx := strictForeignKeyFixture(t)
	child.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
	RebuildTable(child, true, false)
	child.PlannerRowEstimate.value.Store(0)
	child.PlannerRowEstimate.present.Store(false)
	if child.CountEstimate() != 0 || child.Count() != 1 {
		t.Fatal("fixture lacks a main row with unknown estimate")
	}
	requirePanic(t, func() { createStrictForeignKey(child, parent, strictForeignKeyFixtureKey(), tx, scm.NewNil()) })
	assertStrictForeignKeyUnlocked(t, db, child, parent)
}
func TestStrictForeignKeyRejectsPreparedWriter(t *testing.T) {
	db, child, parent, tx := strictForeignKeyFixture(t)
	parent.beginContributionMutation()
	requirePanic(t, func() { createStrictForeignKey(child, parent, strictForeignKeyFixtureKey(), tx, scm.NewNil()) })
	parent.endContributionMutation()
	assertStrictForeignKeyUnlocked(t, db, child, parent)
	if !createStrictForeignKey(child, parent, strictForeignKeyFixtureKey(), tx, scm.NewNil()) {
		t.Fatal("DDL failed after writer drained")
	}
	if !dropStrictForeignKey(child, "FK_test", tx, scm.NewNil()) {
		t.Fatal("exact dependency handle drop failed")
	}
	assertStrictForeignKeyUnlocked(t, db, child, parent)
}
func TestStrictForeignKeyRejectsPendingTransaction(t *testing.T) {
	db, child, parent, tx := strictForeignKeyFixture(t)
	child.Shards[0].beginTransactionUse()
	requirePanic(t, func() { createStrictForeignKey(child, parent, strictForeignKeyFixtureKey(), tx, scm.NewNil()) })
	child.Shards[0].endTransactionUse()
	assertStrictForeignKeyUnlocked(t, db, child, parent)
}
func TestStrictForeignKeyValidationPanicReleasesLocks(t *testing.T) {
	db, child, parent, tx := strictForeignKeyFixture(t)
	bad := strictForeignKeyFixtureKey()
	bad.Cols2 = []string{"missing"}
	requirePanic(t, func() { createStrictForeignKey(child, parent, bad, tx, scm.NewNil()) })
	assertStrictForeignKeyUnlocked(t, db, child, parent)
	parent.Unique = nil
	requirePanic(t, func() { createStrictForeignKey(child, parent, strictForeignKeyFixtureKey(), tx, scm.NewNil()) })
	assertStrictForeignKeyUnlocked(t, db, child, parent)
}
func TestStrictForeignKeyBusyMetadataReleasesLocks(t *testing.T) {
	db, child, parent, tx := strictForeignKeyFixture(t)
	parent.ddlMu.Lock()
	requirePanic(t, func() { createStrictForeignKey(child, parent, strictForeignKeyFixtureKey(), tx, scm.NewNil()) })
	parent.ddlMu.Unlock()
	assertStrictForeignKeyUnlocked(t, db, child, parent)
}
func TestStrictForeignKeyPrivateCreateValidationIsAtomic(t *testing.T) {
	db, _, parent, tx := strictForeignKeyFixture(t)
	private := &table{Name: "private_child", schema: db, Columns: []*column{{Name: "id", Typ: "INT", AllowNull: true, Collation: parent.Columns[0].Collation}}}
	first := strictForeignKeyFixtureKey()
	first.Tbl1 = private.Name
	first.Tbl2 = parent.Name
	second := first
	second.Id = "FK_bad"
	second.Cols1 = []string{"id"}
	second.Cols2 = []string{"missing"}
	private.Foreign = []foreignKey{first, second}
	parents, unlock := strictCreateForeignKeyParents(db, private, tx)
	db.schemalock.Lock()
	if err := prepareStrictCreateForeignKeys(db, private, parents); err == nil {
		t.Fatal("bad second constraint was accepted")
	}
	db.schemalock.Unlock()
	unlock()
	if db.tables.Get(private.Name) != nil || len(parent.Foreign) != 0 || len(parent.Triggers) != 0 {
		t.Fatal("invalid CREATE partially published a parent or child")
	}
	assertStrictForeignKeyUnlocked(t, db, parent)
}

func TestStrictForeignKeyPreparedInsertCannotMissPublication(t *testing.T) {
	db, child, parent, tx := strictForeignKeyFixture(t)
	entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	child.AddTrigger(TriggerDescription{Name: "pause_insert", Timing: BeforeInsert, Func: scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { close(entered); <-resume; return args[1] })})
	go func() {
		defer close(done)
		child.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
	}()
	<-entered
	requirePanic(t, func() { createStrictForeignKey(child, parent, strictForeignKeyFixtureKey(), tx, scm.NewNil()) })
	close(resume)
	<-done
	child.RemoveTrigger("pause_insert")
	assertStrictForeignKeyUnlocked(t, db, child, parent)
	if child.Count() != 1 {
		t.Fatal("rejected DDL damaged the in-flight insert")
	}
}

func TestStrictForeignKeyRenameSafety(t *testing.T) {
	db, child, parent, tx := strictForeignKeyFixture(t)
	if !createStrictForeignKey(child, parent, strictForeignKeyFixtureKey(), tx, scm.NewNil()) {
		t.Fatal("fixture key missing")
	}
	childSnapshot, parentSnapshot := showMetadataTable(db, child.Name), showMetadataTable(db, parent.Name)
	for _, tbl := range []*table{child, parent} {
		original := tbl.Name
		requirePanic(t, func() { RenameTable(db.Name, original, "renamed") })
		if tbl.Name != original || db.tables.Get(original) != tbl || db.tables.Get("renamed") != nil {
			t.Fatal("rejected rename changed table identity")
		}
		if !db.schemalock.TryLock() {
			t.Fatal("rejected rename retained schema lock")
		}
		db.schemalock.Unlock()
	}
	if !scm.Equal(childSnapshot, showMetadataTable(db, child.Name)) || !scm.Equal(parentSnapshot, showMetadataTable(db, parent.Name)) || len(child.Foreign) != 1 || len(parent.Foreign) != 1 || len(child.Triggers) != 2 || len(parent.Triggers) != 2 {
		t.Fatal("rejected rename changed constraints or metadata snapshots")
	}
	if !dropStrictForeignKey(child, "FK_test", tx, scm.NewNil()) {
		t.Fatal("drop after rejected rename failed")
	}
	RenameTable(db.Name, child.Name, "renamed")
	if child.Name != "renamed" || db.tables.Get("renamed") != child || !showMetadataTable(db, "child").IsNil() || showMetadataTable(db, "renamed").IsNil() {
		t.Fatal("rename after constraint drop failed")
	}
}

func TestStrictForeignKeyEndpointDropsAreSafe(t *testing.T) {
	db, child, parent, tx := strictForeignKeyFixture(t)
	createStrictForeignKey(child, parent, strictForeignKeyFixtureKey(), tx, scm.NewNil())
	if !child.Foreign[0].Strict || !parent.Foreign[0].Strict {
		t.Fatal("strict safety contract missing on endpoint")
	}
	childSnapshot, parentSnapshot := showMetadataTable(db, child.Name), showMetadataTable(db, parent.Name)
	for _, tbl := range []*table{child, parent} {
		requirePanic(t, func() { DropTable(db.Name, tbl.Name, false) })
		requirePanic(t, func() { tbl.DropColumn("id") })
		if !db.schemalock.TryLock() {
			t.Fatal("rejected drop retained schema lock")
		}
		db.schemalock.Unlock()
		if !tbl.ddlMu.TryLock() {
			t.Fatal("rejected drop retained DDL lock")
		}
		tbl.ddlMu.Unlock()
		if db.tables.Get(tbl.Name) != tbl || len(tbl.Columns) != 1 || len(tbl.Foreign) != 1 {
			t.Fatal("rejected drop changed endpoint")
		}
	}
	requirePanic(t, func() {
		scm.Apply(scm.Globalenv.Vars[scm.Symbol("dropkey")], NewTableScmer(parent), scm.NewString("PRIMARY"))
	})
	if !scm.Equal(childSnapshot, showMetadataTable(db, child.Name)) || !scm.Equal(parentSnapshot, showMetadataTable(db, parent.Name)) || len(parent.Unique) != 1 {
		t.Fatal("rejected drops changed catalog or candidate key")
	}
	if !dropStrictForeignKey(child, "FK_test", tx, scm.NewNil()) {
		t.Fatal("constraint removal failed")
	}
	if !child.DropColumn("id") {
		t.Fatal("column removal failed after DROP constraint")
	}
	DropTable(db.Name, child.Name, false)
	DropTable(db.Name, parent.Name, false)
}

func TestStrictForeignKeySafetyFlagJSONCompatibility(t *testing.T) {
	key := strictForeignKeyFixtureKey()
	key.Strict = true
	encoded, err := json.Marshal(key)
	if err != nil {
		t.Fatal(err)
	}
	var restored foreignKey
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if !restored.Strict {
		t.Fatal("strict constraint safety flag lost during serialization")
	}
	var legacy foreignKey
	if err := json.Unmarshal([]byte(`{"Id":"legacy","Tbl1":"child","Cols1":["id"],"Tbl2":"parent","Cols2":["id"]}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.Strict {
		t.Fatal("legacy constraints changed their safety profile")
	}
}

// Exercise installed triggers on both delta and indexed main generations.
// NULL parent keys have no references; zero belongs to a distinct parent.
func TestForeignKeyNullableParentUpdatePreservesUnrelatedZero(t *testing.T) {
	for _, mode := range []foreignKeyMode{CASCADE, SETNULL, RESTRICT} {
		for _, rebuilt := range []bool{false, true} {
			t.Run(fmt.Sprintf("mode_%d/rebuilt_%t", mode, rebuilt), func(t *testing.T) {
				_, child, parent, tx := strictForeignKeyFixture(t)
				child.CreateColumn("p", "INT", nil, nil)
				parent.Unique = []uniqueKey{{Id: "UQ_parent", Cols: []string{"id"}}}
				parent.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewNil()}, {scm.NewInt(0)}}, nil, scm.NewNil(), false, nil)
				fk := strictForeignKeyFixtureKey()
				fk.Cols1 = []string{"p"}
				fk.Updatemode = mode
				fk.Deletemode = CASCADE
				createStrictForeignKey(child, parent, fk, tx, scm.NewNil())
				child.Insert([]string{"id", "p"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewInt(0)}, {scm.NewInt(2), scm.NewNil()}}, nil, scm.NewNil(), false, nil)
				if rebuilt {
					RebuildTable(parent, true, false)
					RebuildTable(child, true, false)
				}
				if !fkExistenceCheck(nil, child, []string{"p"}, []scm.Scmer{scm.NewInt(0)}) || fkExistenceCheck(nil, child, []string{"p"}, []scm.Scmer{scm.NewNil()}) {
					t.Fatal("reference probe confused NULL and zero")
				}
				updateParent := func(old, next scm.Scmer) {
					parent.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{"id"},
						scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
							return scm.NewBool(a[0].IsNil() == old.IsNil() && (old.IsNil() || scm.Equal(a[0], old)))
						}), []string{"id", "$update"}, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
							scm.Apply(a[2], scm.NewSlice([]scm.Scmer{scm.NewString("id"), next}))
							return a[0]
						}), scm.NewNil(), scm.NewNil(), false)
				}
				readChildren := func() map[int64]scm.Scmer {
					got := map[int64]scm.Scmer{}
					child.scan(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil,
						scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), []string{"id", "p"},
						scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { got[a[1].Int()] = a[2]; return a[0] }), scm.NewNil(), scm.NewNil(), false)
					return got
				}
				assertUnchanged := func() {
					t.Helper()
					got := readChildren()
					if len(got) != 2 || got[1].IsNil() || got[1].Int() != 0 || !got[2].IsNil() {
						t.Fatalf("NULL parent update modified unrelated references: %v", got)
					}
				}
				updateParent(scm.NewNil(), scm.NewInt(1))
				assertUnchanged()
				if mode == RESTRICT {
					requirePanic(t, func() { updateParent(scm.NewInt(0), scm.NewNil()) })
					assertUnchanged()
					if !fkExistenceCheck(nil, parent, []string{"id"}, []scm.Scmer{scm.NewInt(0)}) {
						t.Fatal("rejected parent update changed the parent row")
					}
				} else {
					updateParent(scm.NewInt(0), scm.NewNil())
					got := readChildren()
					if len(got) != 2 || !got[1].IsNil() || !got[2].IsNil() {
						t.Fatalf("zero to NULL did not execute update policy: %v", got)
					}
					updateParent(scm.NewNil(), scm.NewInt(0))
					got = readChildren()
					if len(got) != 2 || !got[1].IsNil() || !got[2].IsNil() {
						t.Fatalf("NULL to zero modified exempt children: %v", got)
					}
				}
			})
		}
	}
}
