/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

func callBuiltin(t *testing.T, name string, args ...scm.Scmer) scm.Scmer {
	t.Helper()
	fn, ok := scm.Globalenv.Vars[scm.Symbol(name)]
	if !ok {
		t.Fatalf("builtin %s not found", name)
	}
	return scm.Apply(fn, args...)
}

func TestCreateTableIfNotExistsReturnsFalseWithoutSaving(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-createtable-fast-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tcreatetablefast")

	CreateDatabase("tcreatetablefast", false)
	cols := scm.NewSlice([]scm.Scmer{
		scm.NewSlice([]scm.Scmer{
			scm.NewString("column"),
			scm.NewString("id"),
			scm.NewString("int"),
			scm.NewSlice(nil),
			scm.NewSlice(nil),
		}),
	})
	options := scm.NewSlice([]scm.Scmer{scm.NewString("engine"), scm.NewString("sloppy")})

	first := callBuiltin(t, "createtable",
		scm.NewString("tcreatetablefast"),
		scm.NewString(".hot"),
		cols,
		options,
		scm.NewBool(true),
	)
	if !scm.ToBool(first) {
		t.Fatal("first createtable should report created=true")
	}

	second := callBuiltin(t, "createtable",
		scm.NewString("tcreatetablefast"),
		scm.NewString(".hot"),
		cols,
		options,
		scm.NewBool(true),
	)
	if scm.ToBool(second) {
		t.Fatal("second createtable should report created=false")
	}
}

func TestShardRebuildForwardsConcurrentInsertsViaNext(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-shard-rebuild-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("trebuildnext")

	CreateDatabase("trebuildnext", false)
	tbl, _ := CreateTable("trebuildnext", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("payload", "TEXT", nil, nil)

	initialRows := make([][]scm.Scmer, 0, 20000)
	for i := 0; i < 20000; i++ {
		initialRows = append(initialRows, []scm.Scmer{
			scm.NewInt(int64(i + 1)),
			scm.NewString(fmt.Sprintf("%032x", i+1)),
		})
	}
	tbl.Insert([]string{"id", "payload"}, initialRows, nil, scm.NewNil(), false, nil)

	shard := tbl.Shards[0]
	rebuiltCh := make(chan *storageShard, 1)
	go func() {
		rebuiltCh <- shard.rebuild(true)
	}()

	deadline := time.Now().Add(3 * time.Second)
	for shard.loadNext() == nil {
		if time.Now().After(deadline) {
			t.Fatal("rebuild never published next shard")
		}
		runtime.Gosched()
	}

	extraRows := make([][]scm.Scmer, 0, 2000)
	for i := 0; i < 2000; i++ {
		extraRows = append(extraRows, []scm.Scmer{
			scm.NewInt(int64(20001 + i)),
			scm.NewString(fmt.Sprintf("%032x", 20001+i)),
		})
	}
	shard.Insert([]string{"id", "payload"}, extraRows, false, nil, false)

	rebuilt := <-rebuiltCh
	if rebuilt == nil {
		t.Fatal("rebuild returned nil shard")
	}
	if got, want := rebuilt.Count(), uint32(len(initialRows)+len(extraRows)); got != want {
		t.Fatalf("rebuilt shard count = %d, want %d", got, want)
	}
}

func TestShardRebuildDeletePropagationUsesStableTranslation(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-shard-rebuild-delete-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("trebuilddelete")

	CreateDatabase("trebuilddelete", false)
	tbl, _ := CreateTable("trebuilddelete", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("payload", "TEXT", nil, nil)
	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewString("one")},
		{scm.NewInt(2), scm.NewString("two")},
	}, nil, scm.NewNil(), false, nil)

	shard := tbl.Shards[0]
	rebuilt := shard.rebuild(true)
	if rebuilt == nil {
		t.Fatal("rebuild returned nil shard")
	}

	shard.UpdateFunction(0, false, false)()
	shard.UpdateFunction(1, false, false)()

	rebuilt.mu.RLock()
	firstDeleted := rebuilt.deletions.Get(0)
	secondDeleted := rebuilt.deletions.Get(1)
	rebuilt.mu.RUnlock()
	if !firstDeleted || !secondDeleted {
		t.Fatalf("rebuilt shard deletions = (%v, %v), want both true", firstDeleted, secondDeleted)
	}
}

func TestShardRebuildUpdatePropagationUsesStableTranslation(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-shard-rebuild-update-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("trebuildupdate")

	CreateDatabase("trebuildupdate", false)
	tbl, _ := CreateTable("trebuildupdate", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("payload", "TEXT", nil, nil)
	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewString("one")},
		{scm.NewInt(2), scm.NewString("two")},
	}, nil, scm.NewNil(), false, nil)

	shard := tbl.Shards[0]
	rebuilt := shard.rebuild(true)
	if rebuilt == nil {
		t.Fatal("rebuild returned nil shard")
	}

	shard.UpdateFunction(0, false, false)()
	shard.UpdateFunction(1, false, false)(scm.NewSlice([]scm.Scmer{
		scm.NewString("payload"), scm.NewString("two-updated"),
	}))

	rebuilt.mu.RLock()
	rowOneDeleted := rebuilt.deletions.Get(1)
	rebuilt.mu.RUnlock()
	got := rebuilt.ColumnReader("payload")(2)
	if !rowOneDeleted || got.String() != "two-updated" {
		t.Fatalf("rebuilt update state = (deleted=%v, payload=%v), want (true, two-updated)", rowOneDeleted, got)
	}
}

func TestManualRepartitionForwardsConcurrentInserts(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-manual-repartition-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tmanualrepartition")

	CreateDatabase("tmanualrepartition", false)
	tbl, _ := CreateTable("tmanualrepartition", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("payload", "TEXT", nil, nil)

	initialRows := make([][]scm.Scmer, 0, 20000)
	for i := 0; i < 20000; i++ {
		initialRows = append(initialRows, []scm.Scmer{
			scm.NewInt(int64(i + 1)),
			scm.NewString(fmt.Sprintf("%032x", i+1)),
		})
	}
	tbl.Insert([]string{"id", "payload"}, initialRows, nil, scm.NewNil(), false, nil)

	if !tbl.beginManualRepartition() {
		t.Fatal("manual repartition was not claimed")
	}

	done := make(chan struct{})
	go func() {
		tbl.repartition([]shardDimension{tbl.NewShardDimension("id", 2)})
		close(done)
	}()

	deadline := time.Now().Add(3 * time.Second)
	for {
		tbl.mu.Lock()
		active := tbl.maintenanceKind == 2
		hasPShards := tbl.PShards != nil
		tbl.mu.Unlock()
		if active && hasPShards {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("manual repartition never entered dual-write mode")
		}
		runtime.Gosched()
	}

	extraRows := make([][]scm.Scmer, 0, 2000)
	for i := 0; i < 2000; i++ {
		extraRows = append(extraRows, []scm.Scmer{
			scm.NewInt(int64(20001 + i)),
			scm.NewString(fmt.Sprintf("%032x", 20001+i)),
		})
	}
	tbl.Insert([]string{"id", "payload"}, extraRows, nil, scm.NewNil(), false, nil)
	<-done

	total := uint32(0)
	for _, s := range tbl.ActiveShards() {
		total += s.Count()
	}
	if got, want := total, uint32(len(initialRows)+len(extraRows)); got != want {
		t.Fatalf("manual repartition count = %d, want %d", got, want)
	}
}

func TestManualRepartitionInsertDeleteUsesTranslationMap(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-manual-repartition-delete-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tmanualrepartitiondelete")

	CreateDatabase("tmanualrepartitiondelete", false)
	tbl, _ := CreateTable("tmanualrepartitiondelete", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("payload", "TEXT", nil, nil)

	initialRows := make([][]scm.Scmer, 0, 20000)
	for i := 0; i < 20000; i++ {
		initialRows = append(initialRows, []scm.Scmer{
			scm.NewInt(int64(i + 1)),
			scm.NewString(fmt.Sprintf("%032x", i+1)),
		})
	}
	tbl.Insert([]string{"id", "payload"}, initialRows, nil, scm.NewNil(), false, nil)

	if !tbl.beginManualRepartition() {
		t.Fatal("manual repartition was not claimed")
	}

	done := make(chan struct{})
	go func() {
		tbl.repartition([]shardDimension{tbl.NewShardDimension("id", 2)})
		close(done)
	}()

	deadline := time.Now().Add(3 * time.Second)
	for !tbl.repartitionDualWriteActive.Load() {
		if time.Now().After(deadline) {
			t.Fatal("manual repartition never enabled dual-write")
		}
		runtime.Gosched()
	}

	oldShard := tbl.Shards[len(tbl.Shards)-1]
	oldShard.mu.RLock()
	oldRecid := oldShard.main_count + uint32(len(oldShard.inserts))
	oldShard.mu.RUnlock()

	oldShard.Insert([]string{"id", "payload"}, [][]scm.Scmer{{
		scm.NewInt(30001),
		scm.NewString("transient"),
	}}, false, nil, false)
	oldShard.UpdateFunction(oldRecid, false, false)()

	<-done

	total := uint32(0)
	for _, s := range tbl.ActiveShards() {
		total += s.Count()
	}
	if got, want := total, uint32(len(initialRows)); got != want {
		t.Fatalf("manual repartition count after insert+delete = %d, want %d", got, want)
	}
}

func TestDatabaseRebuildDoesNotForceFreeTableIntoSingleShardPartition(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-db-rebuild-free-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("trebuildfree")

	CreateDatabase("trebuildfree", false)
	db := GetDatabase("trebuildfree")
	if db == nil {
		t.Fatal("database not found")
	}

	tbl, _ := CreateTable("trebuildfree", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("grp", "INT", nil, nil)
	tbl.Insert([]string{"id", "grp"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(1)},
		{scm.NewInt(2), scm.NewInt(2)},
		{scm.NewInt(3), scm.NewInt(3)},
	}, nil, scm.NewNil(), false, nil)

	result := db.rebuild(true, true, false)
	if len(result.errors) > 0 {
		t.Fatalf("rebuild errors: %v", result.errors)
	}
	if tbl.ShardMode != ShardModeFree {
		t.Fatalf("small free table was repartitioned unexpectedly: mode=%v", tbl.ShardMode)
	}
	if tbl.PShards != nil {
		t.Fatal("small free table should not have partition shards after rebuild")
	}
	if len(tbl.Shards) != 1 {
		t.Fatalf("small free table should still have one free shard, got %d", len(tbl.Shards))
	}
	if got := tbl.Shards[0].Count(); got != 3 {
		t.Fatalf("rebuilt free shard count = %d, want 3", got)
	}
}

func TestPartitionTableEmptySpecKeepsFreeShardMode(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-empty-partition-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("temptypartition")

	CreateDatabase("temptypartition", false)
	tbl, _ := CreateTable("temptypartition", "items", Sloppy, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}, {scm.NewInt(2)}}, nil, scm.NewNil(), false, nil)
	origUUID := tbl.Shards[0].uuid

	res := callBuiltin(t, "partitiontable",
		scm.NewString("temptypartition"),
		scm.NewString("items"),
		scm.NewSlice(nil),
	)
	if scm.ToBool(res) {
		t.Fatal("empty partition spec should not claim a repartition")
	}
	if tbl.ShardMode != ShardModeFree {
		t.Fatalf("empty partition spec changed shard mode to %v", tbl.ShardMode)
	}
	if tbl.PShards != nil {
		t.Fatal("empty partition spec must not create partition shards")
	}
	if len(tbl.Shards) != 1 {
		t.Fatalf("empty partition spec changed free shard count to %d", len(tbl.Shards))
	}
	if tbl.Shards[0].uuid != origUUID {
		t.Fatal("empty partition spec unexpectedly rebuilt the free shard")
	}
	if got := tbl.Shards[0].Count(); got != 2 {
		t.Fatalf("free shard count = %d, want 2", got)
	}
}

func TestPartitionTableNestedAssocAppliesRealPartitioning(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-nested-partition-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tnestedpartition")

	CreateDatabase("tnestedpartition", false)
	tbl, _ := CreateTable("tnestedpartition", "items", Sloppy, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.Insert([]string{"id"}, [][]scm.Scmer{
		{scm.NewInt(1)},
		{scm.NewInt(2)},
		{scm.NewInt(3)},
		{scm.NewInt(4)},
	}, nil, scm.NewNil(), false, nil)

	res := callBuiltin(t, "partitiontable",
		scm.NewString("tnestedpartition"),
		scm.NewString("items"),
		scm.NewSlice([]scm.Scmer{
			scm.NewSlice([]scm.Scmer{
				scm.NewString("id"),
				scm.NewSlice([]scm.Scmer{scm.NewInt(2)}),
			}),
		}),
	)
	if !scm.ToBool(res) {
		t.Fatal("nested assoc partition spec should trigger repartition")
	}
	if tbl.ShardMode != ShardModePartition {
		t.Fatalf("nested assoc partition spec did not switch shard mode: %v", tbl.ShardMode)
	}
	if len(tbl.PDimensions) != 1 || tbl.PDimensions[0].Column != "id" {
		t.Fatalf("unexpected partition schema: %+v", tbl.PDimensions)
	}
	if len(tbl.PShards) != 2 {
		t.Fatalf("expected 2 partition shards, got %d", len(tbl.PShards))
	}
	total := uint32(0)
	for _, s := range tbl.PShards {
		total += s.Count()
	}
	if total != 4 {
		t.Fatalf("partitioned row count = %d, want 4", total)
	}
}

func TestPartitionTableSinglePartitionSpecIsNoop(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-single-partition-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tsinglepartition")

	CreateDatabase("tsinglepartition", false)
	tbl, _ := CreateTable("tsinglepartition", "items", Sloppy, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}, {scm.NewInt(2)}}, nil, scm.NewNil(), false, nil)
	origUUID := tbl.Shards[0].uuid

	res := callBuiltin(t, "partitiontable",
		scm.NewString("tsinglepartition"),
		scm.NewString("items"),
		scm.NewSlice([]scm.Scmer{
			scm.NewSlice([]scm.Scmer{
				scm.NewString("id"),
				scm.NewSlice(nil),
			}),
		}),
	)
	if scm.ToBool(res) {
		t.Fatal("single-partition spec should not trigger repartition")
	}
	if tbl.ShardMode != ShardModeFree {
		t.Fatalf("single-partition spec changed shard mode to %v", tbl.ShardMode)
	}
	if tbl.PShards != nil {
		t.Fatal("single-partition spec must not create partition shards")
	}
	if tbl.Shards[0].uuid != origUUID {
		t.Fatal("single-partition spec unexpectedly rebuilt the shard")
	}
}

func TestDatabaseRebuildWaitsForTableDDL(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-db-rebuild-ddl-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("trebuildddl")

	CreateDatabase("trebuildddl", false)
	db := GetDatabase("trebuildddl")
	if db == nil {
		t.Fatal("database not found")
	}

	tbl, _ := CreateTable("trebuildddl", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)

	tbl.ddlMu.Lock()
	done := make(chan rebuildDatabaseResult, 1)
	go func() {
		done <- db.rebuild(true, false, false)
	}()

	select {
	case <-done:
		t.Fatal("global rebuild ignored the table-local DDL lock")
	case <-time.After(150 * time.Millisecond):
	}

	tbl.ddlMu.Unlock()
	result := <-done
	if len(result.errors) > 0 {
		t.Fatalf("rebuild errors: %v", result.errors)
	}
}

func TestCreateColumnWaitsForTableRebuildLock(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-create-column-ddl-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tcreatecolumnddl")

	CreateDatabase("tcreatecolumnddl", false)
	tbl, _ := CreateTable("tcreatecolumnddl", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)

	tbl.ddlMu.RLock()
	done := make(chan bool, 1)
	go func() {
		done <- tbl.CreateColumn("payload", "TEXT", nil, nil)
	}()

	select {
	case <-done:
		t.Fatal("CreateColumn bypassed the table-local rebuild/read lock")
	case <-time.After(150 * time.Millisecond):
	}

	tbl.ddlMu.RUnlock()
	if ok := <-done; !ok {
		t.Fatal("CreateColumn failed after rebuild/read lock was released")
	}
}

func TestShardRebuildPreservesComputeProxyColumns(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-rebuild-proxy-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("trebuildproxy")

	CreateDatabase("trebuildproxy", false)
	tbl, _ := CreateTable("trebuildproxy", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("running", "INT", nil, nil)
	tbl.Insert([]string{"id"}, [][]scm.Scmer{
		{scm.NewInt(1)},
		{scm.NewInt(2)},
	}, nil, scm.NewNil(), false, nil)

	shard := tbl.Shards[0]
	shard.mu.Lock()
	proxy := &StorageComputeProxy{
		delta:     make(map[uint32]scm.Scmer),
		shard:     shard,
		colName:   "running",
		count:     shard.main_count,
		isOrdered: true,
	}
	proxy.delta[0] = scm.NewInt(100)
	proxy.validMask.Set(0, true)
	shard.columns["running"] = proxy
	shard.mu.Unlock()

	rebuilt := shard.rebuild(true)
	rebuilt.mu.RLock()
	rebuiltCol := rebuilt.columns["running"]
	rebuilt.mu.RUnlock()

	rebuiltProxy, ok := rebuiltCol.(*StorageComputeProxy)
	if !ok {
		t.Fatalf("rebuild materialized compute proxy into %T", rebuiltCol)
	}
	if !rebuiltProxy.isOrdered {
		t.Fatal("rebuild lost ordered-compute proxy flag")
	}
	if !rebuiltProxy.validMask.Get(0) {
		t.Fatal("rebuild lost cached valid row in compute proxy")
	}
	rebuiltProxy.mu.RLock()
	got := rebuiltProxy.delta[0]
	rebuiltProxy.mu.RUnlock()
	if got.Int() != 100 {
		t.Fatalf("rebuilt proxy cached value = %v, want 100", got)
	}
	if rebuiltProxy.validMask.Get(1) {
		t.Fatal("rebuild should keep invalid rows lazy instead of materializing them")
	}
}

func TestEnsureColumnLoadedRestoresComputeProxyRuntimeBindings(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-load-proxy-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tloadproxy")

	CreateDatabase("tloadproxy", false)
	tbl, _ := CreateTable("tloadproxy", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("running", "INT", nil, nil)
	tbl.Insert([]string{"id"}, [][]scm.Scmer{
		{scm.NewInt(1)},
		{scm.NewInt(2)},
	}, nil, scm.NewNil(), false, nil)

	shard := tbl.Shards[0]
	shard.mu.Lock()
	proxy := &StorageComputeProxy{
		delta:     make(map[uint32]scm.Scmer),
		shard:     shard,
		colName:   "running",
		count:     shard.main_count,
		isOrdered: true,
	}
	proxy.delta[0] = scm.NewInt(123)
	proxy.validMask.Set(0, true)
	shard.columns["running"] = proxy
	shard.mu.Unlock()

	f := tbl.schema.persistence.WriteColumn(shard.uuid.String(), "running")
	proxy.Serialize(f)
	f.Close()

	shard.mu.Lock()
	shard.columns["running"] = nil
	shard.mu.Unlock()

	loadedCol := shard.ensureColumnLoaded("running", false)
	loadedProxy, ok := loadedCol.(*StorageComputeProxy)
	if !ok {
		t.Fatalf("loaded column is %T, want *StorageComputeProxy", loadedCol)
	}
	if loadedProxy.shard != shard {
		t.Fatal("ensureColumnLoaded did not restore proxy shard binding")
	}
	if loadedProxy.colName != "running" {
		t.Fatalf("ensureColumnLoaded restored proxy colName=%q, want %q", loadedProxy.colName, "running")
	}
	if !loadedProxy.isOrdered {
		t.Fatal("ensureColumnLoaded lost ordered-proxy flag")
	}
	if got := loadedProxy.GetValue(0).Int(); got != 123 {
		t.Fatalf("loaded proxy cached value = %d, want 123", got)
	}
}

func TestEnsureColumnLoadedRehydratesOrderedProxyFromSchemaPlaceholder(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-load-orc-placeholder-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tloadorcplaceholder")

	CreateDatabase("tloadorcplaceholder", false)
	tbl, _ := CreateTable("tloadorcplaceholder", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("day", "INT", nil, nil)
	tbl.CreateColumn("running", "INT", nil, nil)
	tbl.Insert([]string{"id", "day"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(10)},
		{scm.NewInt(2), scm.NewInt(20)},
	}, nil, scm.NewNil(), false, nil)

	tbl.ddlMu.Lock()
	tbl.computeOrderedColumnDDLLocked(
		"running",
		[]string{"day"},
		[]bool{false},
		0,
		[]string{"id"},
		scm.NewNil(),
		scm.NewNil(),
		scm.NewNil(),
	)
	tbl.ddlMu.Unlock()

	shard := tbl.Shards[0]
	shard.mu.Lock()
	shard.columns["running"] = new(StorageSparse)
	shard.mu.Unlock()

	f := tbl.schema.persistence.WriteColumn(shard.uuid.String(), "running")
	(&StorageSparse{}).Serialize(f)
	f.Close()

	shard.mu.Lock()
	shard.columns["running"] = nil
	shard.mu.Unlock()

	loadedCol := shard.ensureColumnLoaded("running", false)
	loadedProxy, ok := loadedCol.(*StorageComputeProxy)
	if !ok {
		t.Fatalf("loaded column is %T, want *StorageComputeProxy", loadedCol)
	}
	if loadedProxy.shard != shard {
		t.Fatal("ensureColumnLoaded did not restore placeholder proxy shard binding")
	}
	if loadedProxy.colName != "running" {
		t.Fatalf("ensureColumnLoaded restored placeholder proxy colName=%q, want %q", loadedProxy.colName, "running")
	}
	if !loadedProxy.isOrdered {
		t.Fatal("ensureColumnLoaded lost ordered-proxy contract for placeholder-backed ORC column")
	}
	if loadedProxy.validMask.Count() != 0 {
		t.Fatal("placeholder-backed ORC proxy must stay invalid until foreground recompute")
	}
}

func TestEphemeralQueryShardLoadIgnoresPersistedHelperContents(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-ephemeral-helper-load-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tephemeralhelper")

	CreateDatabase("tephemeralhelper", false)
	tbl, _ := CreateTable("tephemeralhelper", ".helper", Cache, false)
	tbl.CreateColumn("grp", "TEXT", nil, nil)
	tbl.CreateColumn("sumv", "INT", nil, nil)

	orig := tbl.Shards[0]
	grp := &StorageSCMER{values: []scm.Scmer{scm.NewString("A"), scm.NewString("B")}}
	f := tbl.schema.persistence.WriteColumn(orig.uuid.String(), "grp")
	grp.Serialize(f)
	f.Close()

	proxy := &StorageComputeProxy{
		delta:    map[uint32]scm.Scmer{0: scm.NewInt(100), 1: scm.NewInt(200)},
		count:    2,
		computor: scm.NewSymbol("+"),
	}
	proxy.validMask.Set(0, true)
	proxy.validMask.Set(1, true)
	f = tbl.schema.persistence.WriteColumn(orig.uuid.String(), "sumv")
	proxy.Serialize(f)
	f.Close()

	reloaded := &storageShard{
		uuid:         orig.uuid,
		columns:      make(map[string]ColumnStorage),
		deltaColumns: make(map[string]int),
	}
	reloaded.load(tbl)

	if reloaded.main_count != 0 {
		t.Fatalf("ephemeral helper load restored main_count=%d, want 0", reloaded.main_count)
	}
	if reloaded.columns["grp"] != nil {
		t.Fatalf("ephemeral helper grp column should stay unloaded on reload, got %T", reloaded.columns["grp"])
	}
	if reloaded.columns["sumv"] != nil {
		t.Fatalf("ephemeral helper compute column should stay unloaded on reload, got %T", reloaded.columns["sumv"])
	}
	if _, ok := reloaded.ensureColumnLoaded("grp", false).(*StorageSparse); !ok {
		t.Fatalf("ephemeral helper grp lazy load returned %T, want *StorageSparse", reloaded.columns["grp"])
	}
	if _, ok := reloaded.ensureColumnLoaded("sumv", false).(*StorageSparse); !ok {
		t.Fatalf("ephemeral helper compute lazy load returned %T, want *StorageSparse", reloaded.columns["sumv"])
	}
	if got := reloaded.Count(); got != 0 {
		t.Fatalf("ephemeral helper row count = %d, want 0", got)
	}
}

func TestCreateColumnBuiltinUpgradesExistingColumnToORC(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-createcolumn-orc-upgrade-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tcreatecolumnorc")

	CreateDatabase("tcreatecolumnorc", false)
	tbl, _ := CreateTable("tcreatecolumnorc", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("day", "INT", nil, nil)
	tbl.CreateColumn("amount", "INT", nil, nil)
	tbl.CreateColumn("running", "INT", nil, nil)
	tbl.Insert([]string{"id", "day", "amount"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(10), scm.NewInt(100)},
		{scm.NewInt(2), scm.NewInt(20), scm.NewInt(200)},
	}, nil, scm.NewNil(), false, nil)

	mapFn := scm.Eval(scm.Read("test", "(lambda ($set v) (list $set v))"), &scm.Globalenv)
	reduceFn := scm.Eval(scm.Read("test", "(lambda (acc mapped) (begin (define new_acc (+ acc (cadr mapped))) ((car mapped) new_acc) new_acc))"), &scm.Globalenv)
	options := scm.NewSlice([]scm.Scmer{
		scm.NewString("sortcols"), scm.NewSlice([]scm.Scmer{scm.NewString("day")}),
		scm.NewString("sortdirs"), scm.NewSlice([]scm.Scmer{scm.NewBool(false)}),
		scm.NewString("partitioncount"), scm.NewInt(0),
		scm.NewString("mapcols"), scm.NewSlice([]scm.Scmer{scm.NewString("amount")}),
		scm.NewString("mapfn"), mapFn,
		scm.NewString("reducefn"), reduceFn,
		scm.NewString("reduceinit"), scm.NewInt(0),
	})
	createcolumn := scm.Globalenv.Vars[scm.Symbol("createcolumn")]
	result := scm.Apply(
		createcolumn,
		scm.NewString("tcreatecolumnorc"),
		scm.NewString("items"),
		scm.NewString("running"),
		scm.NewString("INT"),
		scm.NewSlice(nil),
		options,
	)
	if !result.Bool() {
		t.Fatal("createcolumn should report success when upgrading an existing column to ORC")
	}

	shard := tbl.Shards[0]
	shard.mu.RLock()
	col := shard.columns["running"]
	shard.mu.RUnlock()
	proxy, ok := col.(*StorageComputeProxy)
	if !ok {
		t.Fatalf("running column is %T, want *StorageComputeProxy", col)
	}
	if !proxy.isOrdered {
		t.Fatal("createcolumn upgrade did not mark proxy as ordered")
	}
	if got := proxy.GetValue(0).Int(); got != 100 {
		t.Fatalf("running[0] = %d, want 100", got)
	}
	if got := proxy.GetValue(1).Int(); got != 300 {
		t.Fatalf("running[1] = %d, want 300", got)
	}
	if got := shard.getDelta(0, "running").Int(); got != 100 {
		t.Fatalf("delta running[0] = %d, want 100", got)
	}
	if got := shard.getDelta(1, "running").Int(); got != 300 {
		t.Fatalf("delta running[1] = %d, want 300", got)
	}
}

func TestShardRebuildWaitsForOrderedProxySnapshot(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-rebuild-orc-snapshot-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("trebuildorc")

	CreateDatabase("trebuildorc", false)
	tbl, _ := CreateTable("trebuildorc", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("running", "INT", nil, nil)
	tbl.Insert([]string{"id"}, [][]scm.Scmer{
		{scm.NewInt(1)},
		{scm.NewInt(2)},
	}, nil, scm.NewNil(), false, nil)

	shard := tbl.Shards[0]
	shard.mu.Lock()
	proxy := &StorageComputeProxy{
		delta:     make(map[uint32]scm.Scmer),
		shard:     shard,
		colName:   "running",
		count:     shard.main_count,
		isOrdered: true,
	}
	shard.columns["running"] = proxy
	shard.mu.Unlock()

	tbl.orcMu.Lock()
	done := make(chan *storageShard, 1)
	go func() {
		done <- shard.rebuild(true)
	}()

	select {
	case <-done:
		t.Fatal("rebuild cloned ordered proxy before ORC snapshot was released")
	case <-time.After(50 * time.Millisecond):
	}

	proxy.mu.Lock()
	proxy.delta[0] = scm.NewInt(100)
	proxy.delta[1] = scm.NewInt(300)
	proxy.validMask.Set(0, true)
	proxy.validMask.Set(1, true)
	proxy.mu.Unlock()
	tbl.orcMu.Unlock()

	rebuilt := <-done
	rebuilt.mu.RLock()
	rebuiltCol := rebuilt.columns["running"]
	rebuilt.mu.RUnlock()
	rebuiltProxy, ok := rebuiltCol.(*StorageComputeProxy)
	if !ok {
		t.Fatalf("rebuilt column is %T, want *StorageComputeProxy", rebuiltCol)
	}
	if !rebuiltProxy.validMask.Get(0) || !rebuiltProxy.validMask.Get(1) {
		t.Fatal("rebuild did not snapshot ordered proxy values published before orcMu release")
	}
	if got := rebuiltProxy.delta[0].Int(); got != 100 {
		t.Fatalf("rebuilt ordered proxy value[0] = %d, want 100", got)
	}
	if got := rebuiltProxy.delta[1].Int(); got != 300 {
		t.Fatalf("rebuilt ordered proxy value[1] = %d, want 300", got)
	}
}

func TestAppendComputeProxyRowsSkipsUncachedDeltaRecids(t *testing.T) {
	proxy := &StorageComputeProxy{
		delta: make(map[uint32]scm.Scmer),
		count: 2,
		main: &StorageSCMER{
			values: []scm.Scmer{scm.NewString("a"), scm.NewString("b")},
		},
	}
	proxy.validMask.Set(0, true)
	proxy.validMask.Set(1, true)

	newProxy := &StorageComputeProxy{delta: make(map[uint32]scm.Scmer), count: 3}
	newIdx := appendComputeProxyRows(newProxy, proxy, []uint32{0, 1, 2}, 0)
	if newIdx != 3 {
		t.Fatalf("appendComputeProxyRows returned %d, want 3", newIdx)
	}
	if !newProxy.validMask.Get(0) || !newProxy.validMask.Get(1) {
		t.Fatal("valid main rows were not ported")
	}
	if newProxy.validMask.Get(2) {
		t.Fatal("uncached forwarded delta row must stay invalid after port")
	}
	if !scm.Equal(newProxy.delta[0], scm.NewString("a")) {
		t.Fatalf("row 0 = %v, want %v", newProxy.delta[0], scm.NewString("a"))
	}
	if !scm.Equal(newProxy.delta[1], scm.NewString("b")) {
		t.Fatalf("row 1 = %v, want %v", newProxy.delta[1], scm.NewString("b"))
	}
}

func TestComputeProxyGetValueUsesDeltaBeyondMainCount(t *testing.T) {
	proxy := &StorageComputeProxy{
		delta:      map[uint32]scm.Scmer{2: scm.NewString("delta")},
		count:      2,
		compressed: true,
		main: &StorageSCMER{
			values: []scm.Scmer{scm.NewString("a"), scm.NewString("b")},
		},
	}
	if !scm.Equal(proxy.GetValue(2), scm.NewString("delta")) {
		t.Fatalf("proxy.GetValue(2) = %v, want %v", proxy.GetValue(2), scm.NewString("delta"))
	}
}

func TestComputeProxyGetCachedReaderTxUsesSessionVariants(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-compute-reader-session-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tcomputesession")

	CreateDatabase("tcomputesession", false)
	tbl, _ := CreateTable("tcomputesession", "items", Safe, false)
	tbl.CreateColumn("base", "INT", nil, nil)
	tbl.Insert([]string{"base"}, [][]scm.Scmer{{scm.NewInt(5)}}, nil, scm.NewNil(), false, nil)

	formulaExpr := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("+"),
		scm.NewSymbol("base"),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("session"),
			scm.NewString("k"),
		}),
	})
	params := scm.NewSlice([]scm.Scmer{scm.NewSymbol("base")})
	inputCols, computor := buildComputedFn(formulaExpr, params, &scm.Globalenv, []string{"base"})
	if computor.IsNil() {
		t.Fatal("buildComputedFn returned nil computor")
	}

	shard := tbl.Shards[0]
	shard.mu.Lock()
	proxy := &StorageComputeProxy{
		delta:       make(map[uint32]scm.Scmer),
		computor:    computor,
		inputCols:   inputCols,
		shard:       shard,
		colName:     ".temp",
		count:       shard.main_count,
		sessionKeys: extractSessionKeys(formulaExpr),
	}
	shard.columns[".temp"] = proxy
	shard.mu.Unlock()

	makeTx := func(k int64) *TxContext {
		sess := scm.NewSession()
		tx := NewTxContext(TxCursorStability)
		tx.Session = sess
		scm.Apply(sess, scm.NewString("__memcp_tx"), scm.NewAny(tx))
		scm.Apply(sess, scm.NewString("k"), scm.NewInt(k))
		return tx
	}

	tx1 := makeTx(1)
	tx2 := makeTx(10)

	reader1 := proxy.GetCachedReaderTx(tx1)
	reader2 := proxy.GetCachedReaderTx(tx2)

	if got := reader1.GetValue(0).Int(); got != 6 {
		t.Fatalf("reader1.GetValue(0) = %d, want 6", got)
	}
	if got := reader2.GetValue(0).Int(); got != 15 {
		t.Fatalf("reader2.GetValue(0) = %d, want 15", got)
	}
	if got := reader1.GetValue(0).Int(); got != 6 {
		t.Fatalf("reader1.GetValue(0) second read = %d, want 6", got)
	}
	if len(proxy.variants) != 2 {
		t.Fatalf("proxy variants = %d, want 2", len(proxy.variants))
	}
}

func TestInvalidateORCHitsShadowRebuildShards(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-orc-shadow-invalidate-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("torcshadow")

	CreateDatabase("torcshadow", false)
	tbl, _ := CreateTable("torcshadow", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("day", "INT", nil, nil)
	tbl.CreateColumn("running", "INT", nil, nil)
	tbl.Insert([]string{"id", "day"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(10)},
		{scm.NewInt(2), scm.NewInt(20)},
	}, nil, scm.NewNil(), false, nil)
	for i := range tbl.Columns {
		if tbl.Columns[i].Name == "running" {
			tbl.Columns[i].OrcSortCols = []string{"day"}
			tbl.Columns[i].OrcSortDirs = []bool{false}
			break
		}
	}

	makeDayCol := func(vals ...int64) *StorageSCMER {
		col := new(StorageSCMER)
		col.init(uint32(len(vals)))
		for i, v := range vals {
			col.build(uint32(i), scm.NewInt(v))
		}
		col.finish()
		return col
	}
	makeProxy := func(sh *storageShard) *StorageComputeProxy {
		proxy := &StorageComputeProxy{
			delta:     make(map[uint32]scm.Scmer),
			shard:     sh,
			colName:   "running",
			count:     2,
			isOrdered: true,
		}
		proxy.delta[0] = scm.NewInt(100)
		proxy.delta[1] = scm.NewInt(300)
		proxy.validMask.Set(0, true)
		proxy.validMask.Set(1, true)
		return proxy
	}

	base := tbl.Shards[0]
	base.mu.Lock()
	base.main_count = 2
	base.columns["day"] = makeDayCol(10, 20)
	base.columns["running"] = makeProxy(base)
	base.mu.Unlock()

	shadow := NewShard(tbl)
	shadow.mu.Lock()
	shadow.main_count = 2
	shadow.columns["day"] = makeDayCol(10, 20)
	shadow.columns["running"] = makeProxy(shadow)
	shadow.mu.Unlock()
	base.storeNext(shadow)

	tbl.invalidateORCFromSortKey("running", []scm.Scmer{scm.NewInt(5)})

	base.mu.RLock()
	baseProxy := base.columns["running"].(*StorageComputeProxy)
	base.mu.RUnlock()
	shadow.mu.RLock()
	shadowProxy := shadow.columns["running"].(*StorageComputeProxy)
	shadow.mu.RUnlock()

	if baseProxy.validMask.Get(0) || baseProxy.validMask.Get(1) {
		t.Fatal("active shard ORC proxy stayed valid after invalidateORC")
	}
	if shadowProxy.validMask.Get(0) || shadowProxy.validMask.Get(1) {
		t.Fatal("shadow rebuild shard ORC proxy stayed valid after invalidateORC")
	}
}
