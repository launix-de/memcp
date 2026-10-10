/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: AGPL-3.0-or-later */

package storage

import "fmt"
import "time"
import "unsafe"
import "strings"
import "testing"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

func assertInitializedColumn(t *testing.T, tbl *table, column string, value scm.Scmer, rows int) {
	t.Helper()
	count := 0
	for _, shard := range tbl.ActiveShards() {
		release := shard.GetRead(nil)
		read := shard.ColumnReaderTx(nil, column, false)
		shard.mu.RLock()
		for i := uint32(0); i < shard.main_count+uint32(len(shard.inserts)); i++ {
			if !shard.deletions.Get(uint(i)) {
				if actual := read(i); !scm.Equal(actual, value) {
					shard.mu.RUnlock()
					release()
					t.Fatalf("row %d column %s = %v, want %v", i, column, actual, value)
				}
				count++
			}
		}
		shard.mu.RUnlock()
		release()
	}
	if count != rows {
		t.Fatalf("initialized visible rows = %d, want %d", count, rows)
	}
}

func TestAddedDefaultOverlappingInsertAndRebuild(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "taddeddefaultconcurrent", 50)
	if result := RebuildTable(tbl, true, false); strings.Contains(result, "errors:") {
		t.Fatal(result)
	}
	started := make(chan struct{})
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		close(started)
		for i := int64(51); i <= 100; i++ {
			tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(i), scm.NewString("concurrent")}}, nil, scm.NewNil(), false, nil)
		}
	}()
	<-started
	tbl.CreateColumn("flag", "INT", nil, []scm.Scmer{scm.NewString("null"), scm.NewBool(false), scm.NewString("default"), scm.NewInt(7)})
	if failure := <-done; failure != nil {
		t.Fatalf("overlapping INSERT failed: %v", failure)
	}
	assertInitializedColumn(t, tbl, "flag", scm.NewInt(7), 100)
	if result := RebuildTable(tbl, true, false); strings.Contains(result, "errors:") {
		t.Fatal(result)
	}
	assertInitializedColumn(t, reloadTableFromPersistence(t, "taddeddefaultconcurrent", persistence), "flag", scm.NewInt(7), 100)
}

func TestAddedDefaultOverlappingGeneratedIDInsert(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "taddeddefaultid", 0)
	tbl.Columns[0].AutoIncrement = true
	tbl.publishShowColumnsSnapshot()
	started := make(chan struct{})
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		close(started)
		for i := 0; i < 100; i++ {
			tbl.Insert([]string{"payload"}, [][]scm.Scmer{{scm.NewString("generated")}}, nil, scm.NewNil(), false, nil)
		}
	}()
	<-started
	tbl.CreateColumn("flag", "INT", nil, []scm.Scmer{scm.NewString("default"), scm.NewInt(2)})
	select {
	case failure := <-done:
		if failure != nil {
			t.Fatalf("generated-ID INSERT failed: %v", failure)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ADD deadlocked with generated-ID batch reservation")
	}
	assertInitializedColumn(t, tbl, "flag", scm.NewInt(2), 100)
}

func TestAddedDefaultOverlappingOverflowAppend(t *testing.T) {
	oldSize := Settings.ShardSize
	Settings.ShardSize = 2
	t.Cleanup(func() { Settings.ShardSize = oldSize })
	tbl, persistence := createDurabilityTestTable(t, "taddeddefaultoverflow", 2)
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		for i := int64(3); i <= 20; i++ {
			tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(i), scm.NewString("overflow")}}, nil, scm.NewNil(), false, nil)
		}
	}()
	tbl.CreateColumn("flag", "INT", nil, []scm.Scmer{scm.NewString("default"), scm.NewInt(9)})
	select {
	case failure := <-done:
		if failure != nil {
			t.Fatalf("overflow INSERT failed: %v", failure)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ADD deadlocked with overflow publication")
	}
	deadline := time.Now().Add(5 * time.Second)
	for tbl.overflowRebuilds.Load() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("overflow rebuild did not finish")
		}
		time.Sleep(time.Millisecond)
	}
	assertInitializedColumn(t, tbl, "flag", scm.NewInt(9), 20)
	assertInitializedColumn(t, reloadTableFromPersistence(t, "taddeddefaultoverflow", persistence), "flag", scm.NewInt(9), 20)
}

func TestAddedRequiredColumnRechecksInFlightInsert(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "taddedrequiredinflight", 0)
	entered, resume := make(chan struct{}), make(chan struct{})
	tbl.AddTrigger(TriggerDescription{Name: "hold_insert", Timing: BeforeInsert,
		Func: scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
			close(entered)
			<-resume
			return args[1]
		})})
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewString("old-schema")}}, nil, scm.NewNil(), false, nil)
	}()
	<-entered
	tbl.CreateColumn("required", "INT", nil, []scm.Scmer{scm.NewString("null"), scm.NewBool(false)})
	close(resume)
	select {
	case failure := <-done:
		if failure == nil {
			t.Fatal("old-schema INSERT stored an omitted NOT NULL column")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("in-flight INSERT did not finish")
	}
	tbl.RemoveTrigger("hold_insert")
	if got := tbl.Count(); got != 0 {
		t.Fatalf("rejected INSERT published %d rows", got)
	}
	tbl.Insert([]string{"id", "payload", "required"}, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("new-schema"), scm.NewInt(4)}}, nil, scm.NewNil(), false, nil)
	assertInitializedColumn(t, reloadTableFromPersistence(t, "taddedrequiredinflight", persistence), "required", scm.NewInt(4), 1)
}

func TestAddedDefaultPreservesTransactionCommitAndRollback(t *testing.T) {
	for _, commit := range []bool{false, true} {
		t.Run(fmt.Sprintf("commit=%t", commit), func(t *testing.T) {
			tbl, persistence := createDurabilityTestTable(t, "taddeddefaulttx", 1)
			tx := NewTxContext(TxACID)
			tx.Session = scm.NewSession()
			scm.Apply(tx.Session, scm.NewString("__memcp_tx"), scm.NewAny(tx))
			tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("pending")}}, nil, scm.NewNil(), false, nil, InsertOptions{Tx: tx})
			tbl.CreateColumn("flag", "INT", nil, []scm.Scmer{scm.NewString("null"), scm.NewBool(false), scm.NewString("default"), scm.NewInt(3)})
			want := 1
			if commit {
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
				want++
			} else {
				tx.Rollback()
			}
			assertInitializedColumn(t, tbl, "flag", scm.NewInt(3), want)
			assertInitializedColumn(t, reloadTableFromPersistence(t, "taddeddefaulttx", persistence), "flag", scm.NewInt(3), want)
		})
	}
}

func TestAddedDefaultNullableAdapterPolicy(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "taddeddefaultpolicy", 1)
	tbl.CreateColumn("optional", "INT", nil, []scm.Scmer{scm.NewString("default"), scm.NewInt(4), scm.NewString("fill_existing"), scm.NewBool(false)})
	assertInitializedColumn(t, tbl, "optional", scm.NewNil(), 1)
	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("new")}}, nil, scm.NewNil(), false, nil)
	reloaded := reloadTableFromPersistence(t, "taddeddefaultpolicy", persistence)
	shard := reloaded.ActiveShards()[0]
	read := shard.ColumnReaderTx(nil, "optional", false)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	if !read(0).IsNil() || read(1).Int() != 4 {
		t.Fatal("nullable ADD policy changed old NULL or future INSERT default")
	}
}

func TestAddedDefaultWaitsForExistingRebuildPublication(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "taddeddefaultrebuild", 50)
	blocking := &blockingColumnWritePersistence{
		PersistenceEngine: persistence,
		entered:           make(chan struct{}),
		release:           make(chan struct{}),
	}
	tbl.schema.persistence = blocking
	rebuilt := make(chan string, 1)
	go func() { rebuilt <- RebuildTable(tbl, true, false) }()
	<-blocking.entered
	added := make(chan any, 1)
	go func() {
		defer func() { added <- recover() }()
		tbl.CreateColumn("flag", "INT", nil, []scm.Scmer{scm.NewString("default"), scm.NewInt(8)})
	}()
	select {
	case failure := <-added:
		close(blocking.release)
		<-rebuilt
		t.Fatalf("ADD bypassed in-progress rebuild publication: %v", failure)
	case <-time.After(20 * time.Millisecond):
	}
	close(blocking.release)
	if result := <-rebuilt; strings.Contains(result, "errors:") {
		t.Fatal(result)
	}
	select {
	case failure := <-added:
		if failure != nil {
			t.Fatalf("ADD failed after rebuild: %v", failure)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ADD did not resume after rebuild publication")
	}
	assertInitializedColumn(t, tbl, "flag", scm.NewInt(8), 50)
	assertInitializedColumn(t, reloadTableFromPersistence(t, "taddeddefaultrebuild", persistence), "flag", scm.NewInt(8), 50)
}

func TestRejectedAddedDefaultLeavesSchemaAndRowsUnchanged(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "taddeddefaultreject", 2)
	for _, options := range [][]scm.Scmer{
		{scm.NewString("null"), scm.NewBool(false)},
		{scm.NewString("unique"), scm.NewBool(true), scm.NewString("default"), scm.NewInt(1)},
		{scm.NewString("default_expression"), scm.NewString("RANDOM")},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid ADD succeeded")
				}
			}()
			tbl.CreateColumn("rejected", "INT", nil, options)
		}()
		if len(tbl.Columns) != 2 || len(tbl.Unique) != 0 {
			t.Fatal("rejected ADD changed columns or constraints")
		}
	}
	tbl.CreateColumn("flag", "INT", nil, []scm.Scmer{scm.NewString("default"), scm.NewInt(2)})
	assertInitializedColumn(t, reloadTableFromPersistence(t, "taddeddefaultreject", persistence), "flag", scm.NewInt(2), 2)
}

func TestAddedDefaultMetadataAccountsSharedPayloadAndDirectory(t *testing.T) {
	value := scm.NewString(strings.Repeat("payload", 10000))
	c := &column{Name: "captured", InitialValue: value, InitialValues: map[string]columnInitialExtent{"first-generation": {}}}
	tbl := &table{Columns: []*column{c}}
	tbl.columnNamesSnapshot.Store(&tableColumnNamesSnapshot{declarations: []*column{c}})
	one := tbl.metadataMemory()
	for i := 0; i < 20; i++ {
		c.InitialValues[fmt.Sprintf("generation-%d", i)] = columnInitialExtent{}
	}
	shared := tbl.metadataMemory()
	if shared <= one || shared-one >= uint(len(value.String())) {
		t.Fatal("generation extents must be counted without duplicating their shared scalar payload")
	}
	independent := scm.NewString(strings.Clone(value.String()))
	tbl.Columns = append(tbl.Columns, &column{Name: "independent", InitialValue: independent,
		InitialValues: map[string]columnInitialExtent{"independent-generation": {}}})
	if got := tbl.metadataMemory(); got-shared < uint(len(value.String())) {
		t.Fatal("an independent retained scalar allocation was omitted")
	}
	beforeDirectory := tbl.metadataMemory()
	directory := make([]*column, 1, 1024)
	directory[0] = c
	tbl.columnNamesSnapshot.Store(&tableColumnNamesSnapshot{declarations: directory})
	if got := tbl.metadataMemory(); got-beforeDirectory < 1023*uint(unsafe.Sizeof((*column)(nil))) {
		t.Fatal("the published declaration pointer array was omitted")
	}
}

func TestAddedDefaultSchemaDropsRetiredRecoveryExtents(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "taddeddefaultmetadata", 2)
	second := NewShard(tbl)
	func() {
		release := second.GetExclusive()
		defer release()
		second.Insert([]string{"id", "payload"}, [][]scm.Scmer{
			{scm.NewInt(3), scm.NewString("third")},
			{scm.NewInt(4), scm.NewString("fourth")},
		}, false, nil, nil, false, nil)
	}()
	tbl.mu.Lock()
	tbl.Shards = append(tbl.Shards, second)
	tbl.publishTopologyLocked()
	tbl.mu.Unlock()
	tbl.schema.save()
	value := strings.Repeat("captured-default-payload", 100)
	tbl.CreateColumn("captured", "VARCHAR", nil, []scm.Scmer{scm.NewString("default"), scm.NewString(value)})
	encoded, err := json.Marshal(tbl)
	if err != nil {
		t.Fatal(err)
	}
	// One current INSERT default and one captured ADD value, regardless of
	// shard count; generation records contain only row extents.
	if count := strings.Count(string(encoded), value); count != 2 {
		t.Fatalf("serialized payload count = %d, want 2", count)
	}
	if result := RebuildTable(tbl, true, false); strings.Contains(result, "errors:") {
		t.Fatal(result)
	}
	encoded, err = json.Marshal(tbl)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "initial_values") || strings.Contains(string(encoded), "initial_value") {
		t.Fatal("checkpoint retains recovery metadata for retired shard generations")
	}
	assertInitializedColumn(t, reloadTableFromPersistence(t, "taddeddefaultmetadata", persistence), "captured", scm.NewString(value), 4)
}

func TestAddedNullableFreshColumnKeepsSparseColdShard(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "taddednullablecold", 2)
	if result := RebuildTable(tbl, true, false); strings.Contains(result, "errors:") {
		t.Fatal(result)
	}
	db := newDatabase()
	db.Name = "taddednullablecold"
	db.persistence = persistence
	db.srState = COLD
	db.ensureLoaded()
	reloaded := db.GetTable("items")
	shard := reloaded.ActiveShards()[0]
	if shard.state() != COLD {
		t.Fatal("fixture is not cold")
	}
	reloaded.CreateColumn("fresh", "INT", nil, nil)
	if shard.state() != COLD || shard.plannerMainRows.Load() != 0 || shard.plannerDeltaRows.Load() != 0 {
		t.Fatal("fresh nullable ADD loaded old rows")
	}
	if _, sparse := shard.getColumnStorageOrPanic("fresh", false, nil).(*StorageSparse); !sparse {
		t.Fatal("fresh nullable ADD did not use existing sparse storage")
	}
	assertInitializedColumn(t, reloaded, "fresh", scm.NewNil(), 2)
	assertInitializedColumn(t, reloadTableFromPersistence(t, "taddednullablecold", persistence), "fresh", scm.NewNil(), 2)
}

func TestAddedNullableFreshColumnKeepsExistingDeltaRows(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "taddednullabledelta", 2)
	shard := tbl.ActiveShards()[0]
	row := func() *scm.Scmer {
		release := shard.GetRead(nil)
		defer release()
		shard.mu.RLock()
		defer shard.mu.RUnlock()
		return &shard.inserts[0][0]
	}()
	tbl.CreateColumn("fresh", "INT", nil, nil)
	func() {
		release := shard.GetRead(nil)
		defer release()
		shard.mu.RLock()
		defer shard.mu.RUnlock()
		if &shard.inserts[0][0] != row {
			t.Fatal("fresh nullable ADD copied existing delta rows")
		}
		if _, exists := shard.deltaColumns["fresh"]; exists {
			t.Fatal("fresh nullable ADD widened existing delta rows")
		}
	}()
	assertInitializedColumn(t, tbl, "fresh", scm.NewNil(), 2)
}

func TestAddedNullableRetiredNameDoesNotRestoreOldValues(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "taddednullablereused", 2)
	tbl.CreateColumn("reused", "INT", nil, []scm.Scmer{scm.NewString("default"), scm.NewInt(8)})
	if result := RebuildTable(tbl, true, false); strings.Contains(result, "errors:") {
		t.Fatal(result)
	}
	tbl.DropColumn("reused")
	reloaded := reloadTableFromPersistence(t, "taddednullablereused", persistence)
	reloaded.CreateColumn("reused", "INT", nil, nil)
	assertInitializedColumn(t, reloaded, "reused", scm.NewNil(), 2)
	assertInitializedColumn(t, reloadTableFromPersistence(t, "taddednullablereused", persistence), "reused", scm.NewNil(), 2)
}

func TestAddedNullableUntrackedSchemaKeepsRecoveryExtents(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "taddednullableuntracked", 2)
	// Older schema files contain no complete dropped-name history.
	tbl.DroppedColumns = nil
	tbl.CreateColumn("fresh", "INT", nil, nil)
	if len(tbl.Columns[len(tbl.Columns)-1].InitialValues) == 0 {
		t.Fatal("untracked schema skipped conservative ADD recovery")
	}
}

func TestAddedDefaultExpressionSurvivesNilConstantOption(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "taddeddefaultmixedoptions", 2)
	tbl.CreateColumn("created", "DATETIME", nil, []scm.Scmer{
		scm.NewString("default_expression"), scm.NewString("CURRENT_TIMESTAMP"),
		scm.NewString("default"), scm.NewNil(),
	})
	shard := tbl.ActiveShards()[0]
	release := shard.GetRead(nil)
	defer release()
	read := shard.ColumnReaderTx(nil, "created", false)
	value := func() scm.Scmer {
		shard.mu.RLock()
		defer shard.mu.RUnlock()
		return read(0)
	}()
	if value.IsNil() {
		t.Fatal("nil constant option hid the existing expression default")
	}
	assertInitializedColumn(t, tbl, "created", value, 2)
}
