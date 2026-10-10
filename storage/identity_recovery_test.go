/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func identityRecoveryFixture(t *testing.T, schema, typ string, dimensions []int) *table {
	t.Helper()
	oldBasepath := Basepath
	Basepath = t.TempDir()
	t.Cleanup(func() { Basepath = oldBasepath })
	Init(scm.Globalenv)
	CreateDatabase(schema, false)
	t.Cleanup(func() { databases.Remove(schema) })
	tbl, _ := CreateTable(schema, "items", Safe, false)
	attrs := []scm.Scmer{scm.NewString("auto_increment"), scm.NewBool(true), scm.NewString("allocator_max"), scm.NewInt(9223372036854775807), scm.NewString("null"), scm.NewBool(false)}
	if typ == "ANY" {
		scm.RegisterExactPrimitives()
		attrs = append(attrs,
			scm.NewString("allocator_value"), scm.EvalAll("allocator reader", `(lambda (value) (coefficient_to_integer value 0 false))`, &scm.Globalenv),
			scm.NewString("allocator_encode"), scm.EvalAll("allocator writer", `(lambda (value) (integer_to_coefficient value))`, &scm.Globalenv))
	}
	tbl.CreateColumn("id", typ, dimensions, attrs)
	tbl.CreateColumn("bucket", "INT", nil, nil)
	if !createTableKey(tbl, "PRIMARY", []string{"id"}, nil, scm.NewNil(), false) {
		t.Fatal("fixture primary key was not created")
	}
	tbl.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(-1)}, {scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
	if !tbl.beginManualRepartition() {
		t.Fatal("fixture could not start partitioning")
	}
	tbl.repartition([]shardDimension{{Column: "bucket", Pivots: []scm.Scmer{scm.NewInt(0)}, NumPartitions: 2}})
	if len(tbl.ActiveShards()) != 2 {
		t.Fatal("fixture must have two independently cold partitions")
	}
	return tbl
}

func coldIdentityTable(t *testing.T, source *table) *table {
	t.Helper()
	for _, shard := range source.ActiveShards() {
		func() {
			release := shard.GetRead(nil)
			defer release()
			shard.mu.Lock()
			defer shard.mu.Unlock()
			if shard.logfile != nil {
				shard.logfile.Close() // Retain real WAL without rebuilding its deltas.
			}
		}()
	}
	db := newDatabase()
	db.Name, db.persistence, db.srState = source.schema.Name, source.schema.persistence, COLD
	db.ensureLoaded()
	// Do not use reloadTableFromPersistence: it warms every shard before INSERT.
	restored := db.GetTable("items")
	for _, shard := range restored.ActiveShards() {
		if shard.state() != COLD {
			t.Fatal("restored fixture warmed a shard before the first insert")
		}
	}
	return restored
}

func identityRecoveryValues(t *testing.T, tbl *table) map[string]bool {
	t.Helper()
	ids := make(map[string]bool)
	for _, shard := range tbl.ActiveShards() {
		func() {
			release := shard.GetRead(nil)
			defer release()
			reader := shard.ColumnReaderTx(nil, "id", false)
			shard.mu.RLock()
			defer shard.mu.RUnlock()
			for recid := uint32(0); recid < shard.main_count+uint32(len(shard.inserts)); recid++ {
				if shard.deletions.Get(uint(recid)) {
					continue
				}
				value := reader(recid)
				if tbl.Columns[0].AllocatorValue != nil {
					value = scm.Apply(*tbl.Columns[0].AllocatorValue, value)
				}
				id := value.String()
				if ids[id] {
					t.Fatalf("identity allocation violated the restored primary key: %s", id)
				}
				ids[id] = true
			}
		}()
	}
	return ids
}

func TestIdentityRecoveryReplaysOtherColdPartitionBeforeAllocation(t *testing.T) {
	tbl := identityRecoveryFixture(t, "identity_cold_partition", "BIGINT", nil)
	tbl.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(-1)}}, nil, scm.NewNil(), false, nil)
	restored := coldIdentityTable(t, tbl)
	var allocated int64
	restored.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false,
		func(first, last int64) { allocated = last })
	if allocated != 4 {
		t.Fatalf("first high-partition insert reused an unseen low-partition WAL identity: got %d, want 4", allocated)
	}
	if ids := identityRecoveryValues(t, restored); len(ids) != 4 || !ids["3"] || !ids["4"] {
		t.Fatal("restored primary key or rows were lost", ids)
	}
}

func TestIdentityRecoveryIncludesExplicitAndRolledBackWALValues(t *testing.T) {
	for _, typ := range []string{"BIGINT", "ANY"} {
		t.Run(typ, func(t *testing.T) {
			var dimensions []int

			tbl := identityRecoveryFixture(t, "identity_cold_explicit_"+typ, typ, dimensions)
			cast := tbl.Columns[0].sanitizer
			if typ == "ANY" {
				cast = func(value scm.Scmer) scm.Scmer { return scm.CoefficientEncode(value) }
			}
			tbl.Insert([]string{"id", "bucket"}, [][]scm.Scmer{
				{cast(scm.NewString("9007199254740993")), scm.NewInt(-1)},
				{cast(scm.NewString("-5")), scm.NewInt(-1)},
			}, nil, scm.NewNil(), false, nil)
			tbl.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(-1)}}, nil, scm.NewNil(), false, nil)
			tx := NewTxContext(TxACID)
			var rolled int64
			tbl.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(-1)}}, nil, scm.NewNil(), false,
				func(first, last int64) { rolled = last }, InsertOptions{Tx: tx})
			tx.Rollback()
			if rolled != 9007199254740995 {
				t.Fatal("fixture lost the consumed rollback identity", rolled)
			}
			restored := coldIdentityTable(t, tbl)
			var allocated int64
			restored.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false,
				func(first, last int64) { allocated = last })
			if allocated != rolled+1 {
				t.Fatal("unseen explicit or rollback identity was reused or rounded", allocated, rolled)
			}
			ids := identityRecoveryValues(t, restored)
			if len(ids) != 6 || !ids["-5"] || !ids["9007199254740993"] || ids["9007199254740995"] {
				t.Fatal("recovered rows changed visibility or exact identity values", ids)
			}
		})
	}
}

func TestIdentityRecoveryConcurrentFirstInsertsInDifferentPartitions(t *testing.T) {
	tbl := identityRecoveryFixture(t, "identity_cold_concurrent", "BIGINT", nil)
	tbl.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(-1)}}, nil, scm.NewNil(), false, nil)
	restored := coldIdentityTable(t, tbl)
	start := make(chan struct{})
	allocated := make(chan int64, 2)
	var writers sync.WaitGroup
	for _, bucket := range []int64{-1, 1} {
		writers.Add(1)
		go func(bucket int64) {
			defer writers.Done()
			<-start
			restored.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(bucket)}}, nil, scm.NewNil(), false,
				func(first, last int64) { allocated <- last })
		}(bucket)
	}
	close(start)
	writers.Wait()
	first, second := <-allocated, <-allocated
	if !((first == 4 && second == 5) || (first == 5 && second == 4)) {
		t.Fatal("concurrent first inserts did not reserve distinct recovered ranges", first, second)
	}
	if ids := identityRecoveryValues(t, restored); len(ids) != 5 {
		t.Fatal("concurrent restored inserts lost or reused identities", ids)
	}
}

func TestIdentityRecoveryReadDoesNotCompleteOtherPartition(t *testing.T) {
	tbl := identityRecoveryFixture(t, "identity_cold_read_first", "BIGINT", nil)
	tbl.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(-1)}}, nil, scm.NewNil(), false, nil)
	restored := coldIdentityTable(t, tbl)
	release := restored.ActiveShards()[1].GetRead(nil)
	release()
	if restored.ActiveShards()[0].state() != COLD || restored.identityRecoveryState.Load() != 0 {
		t.Fatal("a single-partition read incorrectly completed identity recovery")
	}
	var allocated int64
	restored.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false,
		func(first, last int64) { allocated = last })
	if allocated != 4 {
		t.Fatal("a read-before-insert bypassed the other WAL", allocated)
	}
}

func TestIdentityRecoveryRejectsRecoveredAllocatorMaximum(t *testing.T) {
	tbl := identityRecoveryFixture(t, "identity_cold_maximum", "ANY", nil)
	tbl.Insert([]string{"id", "bucket"}, [][]scm.Scmer{{
		scm.IntegerToCoefficient(scm.NewInt(9223372036854775807)), scm.NewInt(-1),
	}}, nil, scm.NewNil(), false, nil)
	restored := coldIdentityTable(t, tbl)
	callback := false
	requirePanic(t, func() {
		restored.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false,
			func(first, last int64) { callback = true })
	})
	if callback || atomic.LoadUint64(&restored.Auto_increment) != 9223372036854775807 || len(identityRecoveryValues(t, restored)) != 3 {
		t.Fatal("exhausted recovered allocator published a counter, callback or row")
	}
}

type failedIdentityReplay struct {
	PersistenceEngine
	calls atomic.Int32
}

func (p *failedIdentityReplay) ReplayLog(string) (map[string]struct{}, chan interface{}, PersistenceLogfile) {
	p.calls.Add(1)
	panic("identity recovery replay failure")
}

func TestIdentityRecoveryFailureKeepsAllocationBlockedAndReleasesLocks(t *testing.T) {
	tbl := identityRecoveryFixture(t, "identity_cold_replay_failure", "BIGINT", nil)
	restored := coldIdentityTable(t, tbl)
	failure := &failedIdentityReplay{PersistenceEngine: restored.schema.persistence}
	restored.schema.persistence = failure
	for attempt := 0; attempt < 2; attempt++ {
		requirePanic(t, func() {
			restored.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
		})
	}
	if failure.calls.Load() != 1 || restored.identityRecoveryState.Load() != 2 || atomic.LoadUint64(&restored.Auto_increment) != 2 {
		t.Fatal("failed recovery allowed retry, completion or allocation")
	}
	if restored.activeTopology().operations.Load() != 0 {
		t.Fatal("failed recovery retained a topology operation")
	}
	for _, shard := range restored.ActiveShards() {
		if !shard.mu.TryLock() {
			t.Fatal("failed recovery retained a shard lock")
		}
		shard.mu.Unlock()
	}
}

func TestIdentityRecoveryLockedShardCannotBypassRecovery(t *testing.T) {
	tbl := identityRecoveryFixture(t, "identity_cold_locked_insert", "BIGINT", nil)
	tbl.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(-1)}}, nil, scm.NewNil(), false, nil)
	restored := coldIdentityTable(t, tbl)
	shard := restored.ActiveShards()[1]
	func() {
		release := shard.GetRead(nil)
		defer release()
		shard.mu.Lock()
		defer shard.mu.Unlock()
		requirePanic(t, func() {
			shard.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(1)}}, true, nil, nil, false, nil)
		})
	}()
	if atomic.LoadUint64(&restored.Auto_increment) != 2 {
		t.Fatal("already-locked shard reserved an unrecovered range")
	}
	restored.prepareIdentityRecovery()
	func() {
		release := shard.GetRead(nil)
		defer release()
		shard.mu.Lock()
		defer shard.mu.Unlock()
		shard.Insert([]string{"bucket"}, [][]scm.Scmer{{scm.NewInt(1)}}, true, nil, nil, false, nil)
	}()
	if ids := identityRecoveryValues(t, restored); len(ids) != 4 || !ids["4"] {
		t.Fatal("prepared already-locked insertion did not use its recovered range", ids)
	}
}
