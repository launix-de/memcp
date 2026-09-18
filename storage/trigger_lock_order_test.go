/*
Copyright (C) 2023-2026 Launix GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

package storage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/launix-de/NonLockingReadMap"
	"github.com/launix-de/memcp/scm"
)

func TestForeignKeyTriggerPersistenceOmitsRegenerableProc(t *testing.T) {
	trigger := TriggerDescription{
		Name:     "__fk_example_child_insert",
		Timing:   BeforeInsert,
		Func:     scm.NewBool(true),
		IsSystem: true,
	}
	encoded, err := json.Marshal(trigger)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, persisted := fields["func"]; persisted {
		t.Fatalf("regenerable FK trigger persisted compiled Scheme code: %s", encoded)
	}
}

func TestNonRegenerableInternalTriggerPersistenceKeepsProc(t *testing.T) {
	trigger := TriggerDescription{
		Name:     ".internal_example",
		Timing:   BeforeInsert,
		Func:     scm.NewBool(true),
		IsSystem: true,
	}
	encoded, err := json.Marshal(trigger)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, persisted := fields["func"]; !persisted {
		t.Fatalf("non-regenerable internal trigger lost compiled Scheme code: %s", encoded)
	}
}

func TestDropTableLifecycleTriggerMayDropAnotherTable(t *testing.T) {
	dir := t.TempDir()
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	const databaseName = "tdroplifecycle"
	defer databases.Remove(databaseName)

	CreateDatabase(databaseName, false)
	source, _ := CreateTable(databaseName, "source", Memory, false)
	CreateTable(databaseName, "derived", Memory, false)
	source.AddTrigger(TriggerDescription{
		Name:     ".drop-derived",
		Timing:   AfterDropTable,
		IsSystem: true,
		Func: buildFKProc(scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("droptable"),
			scm.NewString(databaseName),
			scm.NewString("derived"),
			scm.NewBool(true),
		})),
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		DropTable(databaseName, "source", false)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("DropTable deadlocked while an AfterDropTable trigger dropped a related table")
	}
	if GetDatabase(databaseName).GetTable("derived") != nil {
		t.Fatal("AfterDropTable trigger did not drop the related table")
	}
}

func TestDropTriggerDoesNotWaitForUnrelatedRebuild(t *testing.T) {
	defer setupGCTest(t)()
	CreateDatabase("gcdb", false)
	db := GetDatabase("gcdb")
	CreateTable("gcdb", "first", Memory, false)
	CreateTable("gcdb", "second", Memory, false)
	tables := db.tables.GetAll()
	unrelated, owner := tables[0], tables[len(tables)-1]
	owner.AddTrigger(TriggerDescription{Name: "owned_cleanup", Timing: AfterInsert})
	// A rebuild holds this read lock while waiting for CacheManager.Remove.
	// Cleanup of a trigger on another table must not take its DDL write lock.
	unrelated.ddlMu.RLock()
	done := make(chan bool, 1)
	go func() { done <- db.dropTrigger("owned_cleanup") }()
	select {
	case removed := <-done:
		unrelated.ddlMu.RUnlock()
		if !removed {
			t.Fatal("target trigger was not removed")
		}
	case <-time.After(time.Second):
		unrelated.ddlMu.RUnlock()
		<-done // release the old implementation before failing the test
		t.Fatal("trigger cleanup waited for an unrelated rebuild DDL lock")
	}
}

func TestLanguageTriggerPersistenceKeepsOnlySourceDefinition(t *testing.T) {
	trigger := TriggerDescription{
		Name:     "source_defined",
		Timing:   BeforeInsert,
		Func:     scm.NewBool(true),
		Source:   "trigger source",
		Language: "example",
	}
	encoded, err := json.Marshal(trigger)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, persisted := fields["func"]; persisted {
		t.Fatalf("source-defined trigger persisted compiled Scheme code: %s", encoded)
	}
	if string(fields["source"]) != `"trigger source"` || string(fields["language"]) != `"example"` {
		t.Fatalf("source-defined trigger lost its generic definition: %s", encoded)
	}
}

func TestLegacySQLTriggerSourceImportsIntoLanguageDefinition(t *testing.T) {
	encoded := []byte(`{"name":"legacy","timing":"before_insert","source_sql":"SET NEW.value = 1"}`)
	var trigger TriggerDescription
	if err := json.Unmarshal(encoded, &trigger); err != nil {
		t.Fatal(err)
	}
	if trigger.Source != "SET NEW.value = 1" || trigger.Language != "sql" {
		t.Fatalf("legacy SQL source imported as source=%q language=%q", trigger.Source, trigger.Language)
	}
	upgraded, err := json.Marshal(trigger)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(upgraded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, legacy := fields["source_sql"]; legacy {
		t.Fatalf("upgraded trigger retained legacy source_sql field: %s", upgraded)
	}
	if string(fields["language"]) != `"sql"` {
		t.Fatalf("upgraded trigger did not persist SQL language: %s", upgraded)
	}
}

func TestPersistedKeytableTriggerWaitsForRuntimeTarget(t *testing.T) {
	original := TriggerDescription{
		Name:     ".kt_cleanup:.grp:query:test|items|AFTER DELETE",
		Timing:   AfterDelete,
		IsSystem: true,
		Acquire:  func(*TxContext) bool { return true },
	}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var restored TriggerDescription
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.acquireTarget(nil) {
		t.Fatal("persisted keytable trigger ran before its ephemeral target was rebound")
	}

	tbl := &table{Triggers: []TriggerDescription{restored}}
	if !tbl.SetTriggerTarget(original.Name, func(*TxContext) bool { return true }, func() {}) {
		t.Fatal("restored keytable trigger was not reusable")
	}
	if !tbl.Triggers[0].acquireTarget(nil) {
		t.Fatal("rebound keytable trigger did not acquire its runtime target")
	}
}

func TestLegacyPersistedCacheTriggerWaitsForRuntimeTarget(t *testing.T) {
	for _, name := range []string{
		".cache:.grp:query:test:agg|scan0|items|AFTER DELETE",
		".prejoin:abc|items|after_delete",
	} {
		encoded, err := json.Marshal(map[string]any{
			"name": name, "timing": "after_delete", "is_system": true,
		})
		if err != nil {
			t.Fatal(err)
		}
		var restored TriggerDescription
		if err := json.Unmarshal(encoded, &restored); err != nil {
			t.Fatal(err)
		}
		if restored.acquireTarget(nil) {
			t.Fatalf("persisted cache trigger %s ran before its ephemeral target was rebound", name)
		}
	}
}

func TestDropTriggerDoesNotHoldSchemaLockWhileWaitingForTableDDL(t *testing.T) {
	db := &database{Name: "trigger-lock-order", srState: COLD}
	db.tables = NonLockingReadMap.NewReadMap[string, *table]()
	table := &table{Name: "items", schema: db}
	table.Triggers = []TriggerDescription{{Name: "items_after_drop"}}
	db.tables.Set(table.Name, table)

	table.ddlMu.Lock()
	dropped := make(chan bool, 1)
	go func() { dropped <- db.dropTrigger("items_after_drop") }()

	// Give dropTrigger time to take its catalog snapshot and wait on ddlMu.
	time.Sleep(20 * time.Millisecond)
	schemaAvailable := make(chan struct{}, 1)
	go func() {
		db.schemalock.Lock()
		db.schemalock.Unlock()
		schemaAvailable <- struct{}{}
	}()

	select {
	case <-schemaAvailable:
	case <-time.After(2 * time.Second):
		table.ddlMu.Unlock()
		t.Fatal("dropTrigger held schemalock while waiting for table ddlMu")
	}

	table.ddlMu.Unlock()
	select {
	case ok := <-dropped:
		if !ok {
			t.Fatal("trigger was not removed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("dropTrigger did not finish after ddlMu was released")
	}
}

func TestReadTableLockPublicationDoesNotWaitForShardReaders(t *testing.T) {
	shard := &storageShard{}
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	acquired := make(chan func(), 1)
	go func() {
		acquired <- lockTablePublicationShards([]*storageShard{shard}, false, false)
	}()

	select {
	case unlock := <-acquired:
		unlock()
	case <-time.After(2 * time.Second):
		t.Fatal("READ table lock publication waited for an existing shard reader")
	}
}

func TestSnapshotReadTableLockPublicationDoesNotDeadlockBehindQueuedWriter(t *testing.T) {
	shard := &storageShard{}
	shard.mu.RLock()

	writerAcquired := make(chan struct{})
	releaseWriter := make(chan struct{})
	go func() {
		shard.mu.Lock()
		close(writerAcquired)
		<-releaseWriter
		shard.mu.Unlock()
	}()
	// Give the writer time to queue. Go's RWMutex blocks later readers once a
	// writer is waiting, which reproduces the cache-in-a-reader lock cycle.
	time.Sleep(20 * time.Millisecond)

	readPublished := make(chan func(), 1)
	go func() {
		readPublished <- lockTablePublicationShards([]*storageShard{shard}, false, true)
	}()

	var unlockPublication func()
	blocked := false
	select {
	case unlockPublication = <-readPublished:
	case <-time.After(100 * time.Millisecond):
		blocked = true
	}
	shard.mu.RUnlock()
	<-writerAcquired
	close(releaseWriter)
	if blocked {
		unlockPublication = <-readPublished
	}
	unlockPublication()
	if blocked {
		t.Fatal("READ table lock publication deadlocked behind a queued shard writer")
	}
}

func TestWriteTableLockPublicationWaitsForShardReaders(t *testing.T) {
	shard := &storageShard{}
	shard.mu.RLock()

	acquired := make(chan func(), 1)
	go func() {
		acquired <- lockTablePublicationShards([]*storageShard{shard}, true, false)
	}()

	select {
	case unlock := <-acquired:
		unlock()
		shard.mu.RUnlock()
		t.Fatal("WRITE table lock publication passed an existing shard reader")
	case <-time.After(20 * time.Millisecond):
	}

	shard.mu.RUnlock()
	select {
	case unlock := <-acquired:
		unlock()
	case <-time.After(2 * time.Second):
		t.Fatal("WRITE table lock publication did not continue after reader release")
	}
}

func TestTriggerUnlockTemporarilyWithdrawsWriteOwnership(t *testing.T) {
	shard := &storageShard{}
	tx := NewTxContext(TxCursorStability)
	shard.mu.Lock()
	tx.EnterShardWrite(shard)

	txOwnerVisible := false
	nestedAcquired := false
	shard.runWithWriteLockReleased(tx, func() {
		txOwnerVisible = tx.HasShardWrite(shard)
		acquired := make(chan struct{}, 1)
		go func() {
			shard.mu.Lock()
			shard.mu.Unlock()
			acquired <- struct{}{}
		}()
		select {
		case <-acquired:
			nestedAcquired = true
		case <-time.After(2 * time.Second):
		}
	})

	if txOwnerVisible {
		t.Fatal("trigger callback observed stale shard write ownership")
	}
	if !nestedAcquired {
		t.Fatal("nested trigger query could not acquire the released shard lock")
	}
	if !tx.HasShardWrite(shard) {
		t.Fatal("write ownership was not restored after trigger callback")
	}
	tx.ExitShardWrite(shard)
	shard.mu.Unlock()
}

func TestForeignKeyExistenceIndexedComposite(t *testing.T) {
	Init(scm.Globalenv)
	const name = "fk_indexed_probe"
	CreateDatabase(name, true)
	defer databases.Remove(name)
	parent, _ := CreateTable(name, "parent", Memory, true)
	parent.CreateColumn("a", "INT", nil, nil)
	parent.CreateColumn("b", "INT", nil, nil)
	var rows [][]scm.Scmer
	for i := 0; i < 1000; i++ {
		rows = append(rows, []scm.Scmer{scm.NewInt(int64(i)), scm.NewInt(int64(i % 7))})
	}
	parent.Insert([]string{"a", "b"}, rows, nil, scm.NewNil(), false, nil)
	for _, test := range []struct {
		values []scm.Scmer
		want   bool
	}{
		{[]scm.Scmer{scm.NewInt(999), scm.NewInt(5)}, true},
		{[]scm.Scmer{scm.NewInt(999), scm.NewInt(4)}, false},
		{[]scm.Scmer{scm.NewInt(1001), scm.NewInt(0)}, false},
		{[]scm.Scmer{scm.NewInt(1001), scm.NewNil()}, true},
	} {
		if got := fkExistenceCheck(nil, parent, []string{"a", "b"}, test.values); got != test.want {
			t.Fatalf("probe %v: got %v, want %v", test.values, got, test.want)
		}
	}
	shard := parent.Shards[0]
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	if len(shard.Indexes) == 0 {
		t.Fatal("foreign-key probes did not expose their equality bounds to the index engine")
	}
}

func BenchmarkForeignKeyExistenceProbe(b *testing.B) {
	Init(scm.Globalenv)
	const name = "fk_probe_benchmark"
	CreateDatabase(name, true)
	defer databases.Remove(name)
	parent, _ := CreateTable(name, "parent", Memory, true)
	parent.CreateColumn("id", "INT", nil, nil)
	rows := make([][]scm.Scmer, 10000)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i))}
	}
	parent.Insert([]string{"id"}, rows, nil, scm.NewNil(), false, nil)
	parent.schema.rebuild(true, false, false, parent)
	cols := []string{"id"}
	vals := []scm.Scmer{scm.NewInt(9999)}
	for i := 0; i < 10; i++ {
		if !fkExistenceCheck(nil, parent, cols, vals) {
			b.Fatal("existing parent not found")
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !fkExistenceCheck(nil, parent, cols, vals) {
			b.Fatal("existing parent not found")
		}
	}
}

func TestBeforeUpdateReleasesLocallyAcquiredShardLock(t *testing.T) {
	for _, alreadyLocked := range []bool{false, true} {
		name := "local_lock"
		if alreadyLocked {
			name = "caller_lock"
		}
		t.Run(name, func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "tbeforeupdatelock")
			tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
			shard := tbl.ActiveShards()[0]
			called, unlocked := false, false
			tbl.AddTrigger(TriggerDescription{
				Name: "check_update_lock", Timing: BeforeUpdate,
				Func: scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
					called = true
					unlocked = shard.mu.TryLock()
					if unlocked {
						shard.mu.Unlock()
					}
					return args[1]
				}),
			})
			release := shard.GetRead()
			defer release()
			if alreadyLocked {
				shard.mu.Lock()
				defer shard.mu.Unlock()
			}
			updated := shard.UpdateFunction(0, true, alreadyLocked, nil)(
				scm.NewSlice([]scm.Scmer{scm.NewString("id"), scm.NewInt(2)}))
			if !updated.Bool() || !called || !unlocked {
				t.Fatalf("updated=%v, trigger called=%v, shard unlocked=%v", updated.Bool(), called, unlocked)
			}
		})
	}
}

func TestBeforeUpdateForwardsTriggerValuesAfterRebuildCompletion(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tbeforeupdateforward")
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
	shard := tbl.ActiveShards()[0]
	var rebuilt *storageShard
	tbl.AddTrigger(TriggerDescription{
		Name: "rebuild_during_update", Timing: BeforeUpdate,
		Func: scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
			// Complete the successor while the source UPDATE is inside its
			// unlocked trigger. Forwarding must use NEW after trigger rewriting.
			rebuilt = shard.rebuild(true)
			newRow := scm.NewFastDictValue(1)
			newRow.Set(scm.NewString("id"), scm.NewInt(3), nil)
			return scm.NewFastDict(newRow)
		}),
	})
	release := shard.GetRead()
	defer release()
	updated := shard.UpdateFunction(0, true, false, nil)(
		scm.NewSlice([]scm.Scmer{scm.NewString("id"), scm.NewInt(2)}))
	if !updated.Bool() || rebuilt == nil {
		t.Fatal("update did not finish across rebuild publication")
	}
	releaseNext := rebuilt.GetRead()
	defer releaseNext()
	rebuilt.mu.RLock()
	defer rebuilt.mu.RUnlock()
	var values []int
	for id := uint32(0); id < rebuilt.main_count+uint32(len(rebuilt.inserts)); id++ {
		if !rebuilt.deletions.Get(uint(id)) {
			values = append(values, scm.ToInt(rebuilt.rowValueByRecidLocked(id, "id")))
		}
	}
	if len(values) != 1 || values[0] != 3 {
		t.Fatalf("successor values = %v, want trigger-rewritten [3]", values)
	}
}
