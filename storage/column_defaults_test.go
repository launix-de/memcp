/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: AGPL-3.0-or-later */

package storage

import "fmt"
import "time"
import "unsafe"
import "strings"
import "testing"
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
			tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("pending")}}, nil, scm.NewNil(), false, nil, tx)
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
	c := &column{Name: "captured", InitialValues: map[string]columnInitialValue{"first-generation": {Value: value}}}
	tbl := &table{Columns: []*column{c}}
	tbl.columnNamesSnapshot.Store(&tableColumnNamesSnapshot{declarations: []*column{c}})
	one := tbl.metadataMemory()
	for i := 0; i < 20; i++ {
		c.InitialValues[fmt.Sprintf("generation-%d", i)] = columnInitialValue{Value: value}
	}
	shared := tbl.metadataMemory()
	if shared <= one || shared-one >= uint(len(value.String())) {
		t.Fatal("generation extents must be counted without duplicating their shared scalar payload")
	}
	independent := scm.NewString(strings.Clone(value.String()))
	c.InitialValues["independent-payload"] = columnInitialValue{Value: independent}
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
