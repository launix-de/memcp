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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

type failColumnWritePersistence struct {
	PersistenceEngine
	failNext atomic.Bool
}

func (p *failColumnWritePersistence) WriteColumn(shard string, column string) io.WriteCloser {
	if p.failNext.CompareAndSwap(true, false) {
		panic("injected repartition column write failure")
	}
	return p.PersistenceEngine.WriteColumn(shard, column)
}

type failSchemaWritePersistence struct {
	PersistenceEngine
	calls  atomic.Int32
	failAt int32
}

func (p *failSchemaWritePersistence) WriteSchema(schema []byte) {
	if p.calls.Add(1) == p.failAt {
		panic("injected schema publication failure")
	}
	p.PersistenceEngine.WriteSchema(schema)
}

type countingSchemaWritePersistence struct {
	PersistenceEngine
	calls atomic.Int32
}

func (p *countingSchemaWritePersistence) WriteSchema(schema []byte) {
	p.calls.Add(1)
	p.PersistenceEngine.WriteSchema(schema)
}

type blockingSchemaWritePersistence struct {
	PersistenceEngine
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

type blockingSchemaReadPersistence struct {
	PersistenceEngine
	entered chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (p *blockingSchemaReadPersistence) ReadSchema() []byte {
	p.calls.Add(1)
	p.entered <- struct{}{}
	<-p.release
	return p.PersistenceEngine.ReadSchema()
}

func (p *blockingSchemaWritePersistence) WriteSchema(schema []byte) {
	p.once.Do(func() {
		close(p.entered)
		<-p.release
	})
	p.PersistenceEngine.WriteSchema(schema)
}

type blockingColumnWritePersistence struct {
	PersistenceEngine
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (p *blockingColumnWritePersistence) WriteColumn(shard string, column string) io.WriteCloser {
	p.once.Do(func() {
		close(p.entered)
		<-p.release
	})
	return p.PersistenceEngine.WriteColumn(shard, column)
}

func reloadTableFromPersistence(t *testing.T, name string, persistence PersistenceEngine) *table {
	t.Helper()
	db := newDatabase()
	db.Name = name
	db.persistence = persistence
	db.srState = COLD
	db.ensureLoaded()
	tbl := db.GetTable("items")
	if tbl == nil {
		t.Fatalf("reloaded database %s has no items table", name)
	}
	for _, shard := range tbl.ActiveShards() {
		release := shard.GetRead()
		release()
	}
	return tbl
}

func TestConcurrentFirstDatabaseReadersShareOneSchemaLoad(t *testing.T) {
	_, persistence := createDurabilityTestTable(t, "tconcurrentloadsource", 1)
	blocking := &blockingSchemaReadPersistence{
		PersistenceEngine: persistence,
		entered:           make(chan struct{}, 2),
		release:           make(chan struct{}),
	}
	db := newDatabase()
	db.Name = "tconcurrentload"
	db.persistence = blocking
	db.srState = COLD

	loaded := make(chan *table, 2)
	go func() { loaded <- db.GetTable("items") }()
	<-blocking.entered
	go func() { loaded <- db.GetTable("items") }()

	select {
	case <-blocking.entered:
		close(blocking.release)
		t.Fatal("a concurrent reader started a second schema load")
	case <-time.After(50 * time.Millisecond):
	}
	close(blocking.release)
	for range 2 {
		select {
		case tbl := <-loaded:
			if tbl == nil {
				t.Fatal("concurrent reader did not observe the loaded table catalog")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("concurrent reader did not continue after schema load")
		}
	}
	if calls := blocking.calls.Load(); calls != 1 {
		t.Fatalf("schema loaded %d times, want 1", calls)
	}
}

func waitForRepartitionDualWrite(t *testing.T, tbl *table) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !tbl.repartitionDualWriteActive.Load() {
		if time.Now().After(deadline) {
			t.Fatal("repartition never enabled dual-write")
		}
		runtime.Gosched()
	}
}

func createDurabilityTestTable(t *testing.T, databaseName string, rows int) (*table, PersistenceEngine) {
	t.Helper()
	dir, err := os.MkdirTemp("", "memcp-durability-regression-*")
	if err != nil {
		t.Fatal(err)
	}
	oldBasepath := Basepath
	Basepath = dir
	t.Cleanup(func() {
		databases.Remove(databaseName)
		Basepath = oldBasepath
		os.RemoveAll(dir)
	})

	Init(scm.Globalenv)
	LoadDatabases()
	CreateDatabase(databaseName, false)
	tbl, _ := CreateTable(databaseName, "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("payload", "TEXT", nil, nil)
	values := make([][]scm.Scmer, rows)
	for i := range values {
		values[i] = []scm.Scmer{scm.NewInt(int64(i + 1)), scm.NewString(fmt.Sprintf("row-%08d", i+1))}
	}
	tbl.Insert([]string{"id", "payload"}, values, nil, scm.NewNil(), false, nil)
	return tbl, tbl.schema.persistence
}

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

func TestSchemaReloadInvalidatesPlannerCacheOnInit(t *testing.T) {
	stale := scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		panic("stale planner callback must not run")
	})
	plannerCache := &table{Name: ".grp:items:old", PersistencyMode: Cache, OnInit: &stale}
	invalidatePersistedPlannerCodeAfterLoad(plannerCache)
	if plannerCache.OnInit != nil {
		t.Fatal("schema reload retained planner-generated cache oninit")
	}

	for _, durable := range []*table{
		{Name: "application_cache", PersistencyMode: Cache, OnInit: &stale},
		{Name: ".internal", PersistencyMode: Safe, OnInit: &stale},
		{Name: ".memory_helper", PersistencyMode: Memory, OnInit: &stale},
	} {
		invalidatePersistedPlannerCodeAfterLoad(durable)
		if durable.OnInit == nil {
			t.Fatalf("schema reload invalidated non-planner table %s", durable.Name)
		}
	}
}

func TestCreateTableRefreshesInvalidatedPlannerCacheOnInit(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-planner-oninit-refresh-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tplanneroninitrefresh")
	CreateDatabase("tplanneroninitrefresh", false)

	cols := scm.NewSlice([]scm.Scmer{
		scm.NewSlice([]scm.Scmer{
			scm.NewString("column"), scm.NewString("id"), scm.NewString("int"),
			scm.NewSlice(nil), scm.NewSlice(nil),
		}),
	})
	var staleCalls atomic.Int64
	stale := scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		staleCalls.Add(1)
		return scm.NewNil()
	})
	options := scm.NewSlice([]scm.Scmer{
		scm.NewString("engine"), scm.NewString("cache"),
		scm.NewString("oninit"), stale,
	})
	callBuiltin(t, "createtable",
		scm.NewString("tplanneroninitrefresh"), scm.NewString(".grp:items:old"),
		cols, options, scm.NewBool(true), scm.NewNil())
	if staleCalls.Load() != 1 {
		t.Fatalf("initial planner callback ran %d times, want 1", staleCalls.Load())
	}

	plannerCache := GetDatabase("tplanneroninitrefresh").GetTable(".grp:items:old")
	plannerCache.onInitComplete = false // restart-only state is deliberately not persisted
	invalidatePersistedPlannerCodeAfterLoad(plannerCache)
	var currentCalls atomic.Int64
	current := scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		currentCalls.Add(1)
		return scm.NewNil()
	})
	options = scm.NewSlice([]scm.Scmer{
		scm.NewString("engine"), scm.NewString("cache"),
		scm.NewString("oninit"), current,
	})
	created := callBuiltin(t, "createtable",
		scm.NewString("tplanneroninitrefresh"), scm.NewString(".grp:items:old"),
		cols, options, scm.NewBool(true), scm.NewNil())
	if scm.ToBool(created) {
		t.Fatal("refreshing the planner callback recreated the cache table")
	}
	if staleCalls.Load() != 1 || currentCalls.Load() != 1 {
		t.Fatalf("refresh ran stale=%d current=%d callbacks, want 1 and 1", staleCalls.Load(), currentCalls.Load())
	}
}

func TestRegisteredCreateTableTriggerRunsOnCreate(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-createtable-trigger-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	oldRegs := Settings.CreateTableTriggers
	Settings.CreateTableTriggers = nil
	defer func() { Settings.CreateTableTriggers = oldRegs }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tcreatetabletrigger")

	CreateDatabase("tcreatetabletrigger", false)
	body := buildFKProc(scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("insert"),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("table"),
			scm.NewString("tcreatetabletrigger"),
			scm.NewString(".hook"),
		}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString("id")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("list"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewInt(1)}),
		}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("list")}),
		scm.NewProcStruct(scm.Proc{
			Params: scm.NewSlice([]scm.Scmer{}),
			Body:   scm.NewBool(true),
			En:     &scm.Globalenv,
		}),
		scm.NewBool(true),
	}))
	callBuiltin(t, "createcreatetabletrigger",
		scm.NewString("tcreatetabletrigger"),
		scm.NewString(".hook"),
		scm.NewString("seed"),
		scm.NewString(""),
		scm.NewString(""),
		body,
		scm.NewBool(false),
	)
	if _, err := json.Marshal(Settings); err != nil {
		t.Fatalf("settings with registered create-table trigger must stay serializable: %v", err)
	}

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
	created := callBuiltin(t, "createtable",
		scm.NewString("tcreatetabletrigger"),
		scm.NewString(".hook"),
		cols,
		options,
		scm.NewBool(true),
	)
	if !scm.ToBool(created) {
		t.Fatal("createtable should report created=true")
	}

	tbl := GetDatabase("tcreatetabletrigger").GetTable(".hook")
	if tbl == nil {
		t.Fatal("expected created table")
	}
	if got := tbl.Count(); got != 1 {
		t.Fatalf("create-table trigger should have inserted one row, got %d", got)
	}

	created = callBuiltin(t, "createtable",
		scm.NewString("tcreatetabletrigger"),
		scm.NewString(".hook"),
		cols,
		options,
		scm.NewBool(true),
	)
	if scm.ToBool(created) {
		t.Fatal("second createtable should report created=false")
	}
	if got := tbl.Count(); got != 1 {
		t.Fatalf("existing table must not re-fire create-table trigger, got %d rows", got)
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
	shard.Insert([]string{"id", "payload"}, extraRows, false, nil, false, nil)

	rebuilt := <-rebuiltCh
	if rebuilt == nil {
		t.Fatal("rebuild returned nil shard")
	}
	if got, want := rebuilt.Count(), uint32(len(initialRows)+len(extraRows)); got != want {
		t.Fatalf("rebuilt shard count = %d, want %d", got, want)
	}
}

func TestShardRebuildForwardsInsertAcrossSuccessorChain(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "trebuildchain")
	source := tbl.Shards[0]
	firstSuccessor := NewShard(tbl)
	latestSuccessor := NewShard(tbl)

	source.storeNext(firstSuccessor)
	source.nextReady.Store(true)
	firstSuccessor.storeNext(latestSuccessor)
	firstSuccessor.nextReady.Store(true)

	source.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(42)}}, false, nil, false, nil)

	if got := source.Count(); got != 1 {
		t.Fatalf("source count = %d, want 1", got)
	}
	if got := firstSuccessor.Count(); got != 1 {
		t.Fatalf("first successor count = %d, want 1", got)
	}
	if got := latestSuccessor.Count(); got != 1 {
		t.Fatalf("latest successor count = %d, want 1", got)
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

	shard.UpdateFunction(0, false, false, nil)()
	shard.UpdateFunction(1, false, false, nil)()

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

	shard.UpdateFunction(0, false, false, nil)()
	shard.UpdateFunction(1, false, false, nil)(scm.NewSlice([]scm.Scmer{
		scm.NewString("payload"), scm.NewString("two-updated"),
	}))

	rebuilt.mu.RLock()
	rowOneDeleted := rebuilt.deletions.Get(1)
	rebuilt.mu.RUnlock()
	got := rebuilt.ColumnReaderTx(nil, "payload")(2)
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
	}}, false, nil, false, nil)
	oldShard.UpdateFunction(oldRecid, false, false, nil)()

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
		NewTableScmer(tbl),
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
		NewTableScmer(tbl),
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
		NewTableScmer(tbl),
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
		nil,
		scm.NewNil(),
		[]string{"id"},
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

	mapReduceFn := scm.Eval(scm.Read("test", "(lambda (acc $set v) (begin (define new_acc (+ acc v)) ($set new_acc) new_acc))"), &scm.Globalenv)
	options := scm.NewSlice([]scm.Scmer{
		scm.NewString("sortcols"), scm.NewSlice([]scm.Scmer{scm.NewString("day")}),
		scm.NewString("sortdirs"), scm.NewSlice([]scm.Scmer{scm.NewBool(false)}),
		scm.NewString("partitioncount"), scm.NewInt(0),
		scm.NewString("mapcols"), scm.NewSlice([]scm.Scmer{scm.NewString("amount")}),
		scm.NewString("mapreducefn"), mapReduceFn,
		scm.NewString("reduceinit"), scm.NewInt(0),
	})
	createcolumn := scm.Globalenv.Vars[scm.Symbol("createcolumn")]
	result := scm.Apply(
		createcolumn,
		NewTableScmer(tbl),
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

func TestMutationScanRepairsInvalidOrderedComputeColumnBeforeTakingShardLock(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-scan-orc-repair-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("tscanorcrepair")

	CreateDatabase("tscanorcrepair", false)
	tbl, _ := CreateTable("tscanorcrepair", "items", Memory, false)
	tbl.CreateColumn("day", "INT", nil, nil)
	tbl.CreateColumn("amount", "INT", nil, nil)
	tbl.CreateColumn("running", "INT", nil, nil)
	tbl.Insert([]string{"day", "amount"}, [][]scm.Scmer{
		{scm.NewInt(10), scm.NewInt(100)},
		{scm.NewInt(20), scm.NewInt(200)},
	}, nil, scm.NewNil(), false, nil)

	mapReduceFn := scm.Eval(scm.Read("test", "(lambda (acc $set v) (begin (define new_acc (+ acc v)) ($set new_acc) new_acc))"), &scm.Globalenv)
	options := scm.NewSlice([]scm.Scmer{
		scm.NewString("sortcols"), scm.NewSlice([]scm.Scmer{scm.NewString("day")}),
		scm.NewString("sortdirs"), scm.NewSlice([]scm.Scmer{scm.NewBool(false)}),
		scm.NewString("partitioncount"), scm.NewInt(0),
		scm.NewString("mapcols"), scm.NewSlice([]scm.Scmer{scm.NewString("amount")}),
		scm.NewString("mapreducefn"), mapReduceFn,
		scm.NewString("reduceinit"), scm.NewInt(0),
	})
	createcolumn := scm.Globalenv.Vars[scm.Symbol("createcolumn")]
	if !scm.Apply(
		createcolumn,
		NewTableScmer(tbl),
		scm.NewString("running"),
		scm.NewString("INT"),
		scm.NewSlice(nil),
		options,
	).Bool() {
		t.Fatal("createcolumn should upgrade running to an ordered compute column")
	}

	shard := tbl.Shards[0]
	shard.mu.RLock()
	proxy := shard.columns["running"].(*StorageComputeProxy)
	shard.mu.RUnlock()
	if got := proxy.GetValue(1).Int(); got != 300 {
		t.Fatalf("initial running value = %d, want 300", got)
	}
	proxy.validMask.Set(0, false)
	proxy.validMask.Set(1, false)

	done := make(chan []int64, 1)
	go func() {
		values := make([]int64, 0, 2)
		tbl.scan(
			nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
			[]string{},
			trueCondition(),
			[]string{"$update", "running"},
			scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
				values = append(values, args[2].Int())
				return args[0]
			}),
			scm.NewNil(),
			scm.NewNil(),
			false,
		)
		done <- values
	}()

	select {
	case values := <-done:
		if len(values) != 2 || values[0] != 100 || values[1] != 300 {
			t.Fatalf("scan returned ordered values %v, want [100 300]", values)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("mutation scan deadlocked while repairing an ordered compute column under the shard lock")
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

func TestRepartitionBuildFailureDoesNotPublishPartialGeneration(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "trepartitionbuildfailure", 128)
	failing := &failColumnWritePersistence{PersistenceEngine: persistence}
	failing.failNext.Store(true)
	tbl.schema.persistence = failing

	if !tbl.beginManualRepartition() {
		t.Fatal("manual repartition was not claimed")
	}
	var repartitionPanic any
	func() {
		defer func() { repartitionPanic = recover() }()
		tbl.repartition([]shardDimension{tbl.NewShardDimension("id", 2)})
	}()
	if !strings.Contains(fmt.Sprint(repartitionPanic), "injected repartition column write failure") {
		t.Fatalf("repartition panic %q does not report the build failure", repartitionPanic)
	}

	if topology := tbl.activeTopology(); topology.mode != ShardModeFree {
		t.Fatalf("failed repartition published mode %v with %d shards; old free generation must remain authoritative", topology.mode, len(topology.shards))
	}
}

func TestRepartitionConcurrentDeleteSurvivesReload(t *testing.T) {
	const rows = 256
	tbl, persistence := createDurabilityTestTable(t, "trepartitiondeletereload", rows)
	blocking := &blockingColumnWritePersistence{
		PersistenceEngine: persistence,
		entered:           make(chan struct{}),
		release:           make(chan struct{}),
	}
	tbl.schema.persistence = blocking

	if !tbl.beginManualRepartition() {
		t.Fatal("manual repartition was not claimed")
	}
	done := make(chan struct{})
	go func() {
		tbl.repartition([]shardDimension{tbl.NewShardDimension("id", 2)})
		close(done)
	}()
	<-blocking.entered
	waitForRepartitionDualWrite(t, tbl)

	oldShard := tbl.repartitionSources.Load().shards[0]
	if !scm.ToBool(oldShard.UpdateFunction(0, false, false, nil)()) {
		t.Fatal("concurrent delete did not change the source row")
	}
	close(blocking.release)
	<-done

	if got := tbl.Count(); got != rows-1 {
		t.Fatalf("live repartition count after delete = %d, want %d", got, rows-1)
	}
	reloaded := reloadTableFromPersistence(t, "trepartitiondeletereload", persistence)
	if got := reloaded.Count(); got != rows-1 {
		t.Fatalf("reloaded repartition count after delete = %d, want %d; forwarded delete was not durable", got, rows-1)
	}
}

func TestRepartitionPublishedSchemaReloadsNewGeneration(t *testing.T) {
	const rows = 32
	tbl, persistence := createDurabilityTestTable(t, "trepartitionreload", rows)
	if !tbl.beginManualRepartition() {
		t.Fatal("manual repartition was not claimed")
	}
	tbl.repartition([]shardDimension{tbl.NewShardDimension("id", 2)})
	if got := tbl.Count(); got != rows {
		t.Fatalf("live repartition count = %d, want %d", got, rows)
	}

	reloaded := reloadTableFromPersistence(t, "trepartitionreload", persistence)
	if got := reloaded.Count(); got != rows {
		t.Fatalf("reloaded repartition count = %d, want %d; schema references the retired generation", got, rows)
	}
}

func TestRepartitionRolledBackACIDDeleteKeepsRow(t *testing.T) {
	const rows = 256
	tbl, persistence := createDurabilityTestTable(t, "trepartitionacidrollback", rows)
	blocking := &blockingColumnWritePersistence{
		PersistenceEngine: persistence,
		entered:           make(chan struct{}),
		release:           make(chan struct{}),
	}
	tbl.schema.persistence = blocking

	if !tbl.beginManualRepartition() {
		t.Fatal("manual repartition was not claimed")
	}
	done := make(chan struct{})
	go func() {
		tbl.repartition([]shardDimension{tbl.NewShardDimension("id", 2)})
		close(done)
	}()
	<-blocking.entered
	waitForRepartitionDualWrite(t, tbl)

	oldShard := tbl.repartitionSources.Load().shards[0]
	tx := NewTxContext(TxACID)
	if !scm.ToBool(oldShard.UpdateFunction(0, false, false, tx)()) {
		t.Fatal("transactional delete was not staged")
	}
	tx.Rollback()
	close(blocking.release)
	<-done

	if got := tbl.Count(); got != rows {
		t.Fatalf("repartition count after rolled-back ACID delete = %d, want %d", got, rows)
	}
}

func TestRepartitionPostFlipUpdateDoesNotDuplicateRow(t *testing.T) {
	const rows = 256
	tbl, persistence := createDurabilityTestTable(t, "trepartitionpostflipupdate", rows)
	blocking := &blockingSchemaWritePersistence{
		PersistenceEngine: persistence,
		entered:           make(chan struct{}),
		release:           make(chan struct{}),
	}
	tbl.schema.persistence = blocking

	if !tbl.beginManualRepartition() {
		t.Fatal("manual repartition was not claimed")
	}
	done := make(chan struct{})
	go func() {
		tbl.repartition([]shardDimension{tbl.NewShardDimension("id", 2)})
		close(done)
	}()
	defer func() {
		select {
		case <-blocking.release:
		default:
			close(blocking.release)
		}
	}()
	<-blocking.entered

	var target *storageShard
	for _, shard := range tbl.ActiveShards() {
		if shard.Count() > 0 {
			target = shard
			break
		}
	}
	if target == nil {
		t.Fatal("published partition topology has no rows")
	}
	updateDone := make(chan bool, 1)
	go func() {
		changes := scm.NewSlice([]scm.Scmer{scm.NewString("payload"), scm.NewString("updated-after-flip")})
		updateDone <- scm.ToBool(target.UpdateFunction(0, false, false, nil)(changes))
	}()
	select {
	case <-updateDone:
		t.Fatal("update reached a generation before schema publication committed")
	case <-time.After(20 * time.Millisecond):
	}
	close(blocking.release)
	<-done
	if !<-updateDone {
		t.Fatal("post-publication update did not change the row")
	}

	if got := tbl.Count(); got != rows {
		t.Fatalf("repartition count after post-flip update = %d, want %d", got, rows)
	}
}

func TestFailedRebuildSchemaPublicationKeepsLaterWritesRecoverable(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "trebuildsavefailure", 1)
	tbl.schema.ensureBlobTable()
	failing := &failSchemaWritePersistence{PersistenceEngine: persistence, failAt: 1}
	tbl.schema.persistence = failing

	result := RebuildTable(tbl, true, false)
	if !strings.Contains(result, "schema publication failure") {
		t.Fatalf("rebuild result %q does not report injected schema failure", result)
	}
	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("after-failed-save")}}, nil, scm.NewNil(), false, nil)
	if got := tbl.Count(); got != 2 {
		t.Fatalf("live count after failed schema save = %d, want 2", got)
	}

	reloaded := reloadTableFromPersistence(t, "trebuildsavefailure", persistence)
	if got := reloaded.Count(); got != 2 {
		t.Fatalf("reloaded count after failed rebuild schema save = %d, want 2; write reached only the unpublished generation", got)
	}
}

func TestRebuildInsideActiveTransactionDoesNotWaitForItself(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "trebuildselftransaction", 1)
	oldTopology := tbl.activeTopology()
	tx := NewTxContext(TxCursorStability)
	tx.Session = scm.NewSession()
	scm.Apply(tx.Session, scm.NewString("__memcp_tx"), scm.NewAny(tx))
	resultCh := make(chan string, 1)
	panicCh := make(chan any, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()
		withTxSession(tx, func() scm.Scmer {
			tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(2), scm.NewString("same-request")}}, nil, scm.NewNil(), false, nil, tx)
			resultCh <- RebuildTable(tbl, true, false)
			return scm.NewNil()
		})
	}()

	select {
	case r := <-panicCh:
		t.Fatalf("rebuild in active transaction panicked: %v", r)
	case result := <-resultCh:
		if strings.Contains(result, "errors:") {
			t.Fatalf("rebuild in active transaction returned %q", result)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("rebuild waited for the transaction owned by its own request")
	}
	select {
	case <-oldTopology.operationsDrained:
	case <-time.After(time.Second):
		t.Fatal("retired rebuild generation still has active operations")
	}
	select {
	case <-oldTopology.drained:
		t.Fatal("retired rebuild generation drained before its transaction completed")
	default:
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-oldTopology.drained:
	case <-time.After(time.Second):
		t.Fatal("retired rebuild generation did not drain after transaction commit")
	}
	if got := tbl.Count(); got != 2 {
		t.Fatalf("row count after transactional rebuild = %d, want 2", got)
	}
}

func TestEphemeralCacheColumnDDLBuffersSchemaPublicationUntilRebuild(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-cache-schema-batching-*")
	if err != nil {
		t.Fatal(err)
	}
	oldBasepath := Basepath
	Basepath = dir
	t.Cleanup(func() {
		databases.Remove("tcachecolumnbatch")
		Basepath = oldBasepath
		os.RemoveAll(dir)
	})

	Init(scm.Globalenv)
	LoadDatabases()
	CreateDatabase("tcachecolumnbatch", false)
	db := GetDatabase("tcachecolumnbatch")
	tbl, created := CreateTable("tcachecolumnbatch", ".grp:query:test", Cache, false)
	if !created {
		t.Fatal("ephemeral cache table was not created")
	}
	db.ensureBlobTable()
	counting := &countingSchemaWritePersistence{PersistenceEngine: db.persistence}
	db.persistence = counting

	for i := 0; i < 32; i++ {
		if !tbl.CreateColumn(fmt.Sprintf("agg_%d", i), "int", nil, nil) {
			t.Fatalf("cache column %d was not created", i)
		}
	}
	if got := counting.calls.Load(); got != 0 {
		t.Fatalf("ephemeral cache column DDL published schema %d times, want 0 before rebuild", got)
	}
	if !db.schemaDirty.Load() {
		t.Fatal("buffered ephemeral cache columns did not mark the schema dirty")
	}
	if got := len(tbl.Columns); got != 32 {
		t.Fatalf("live cache table has %d columns, want 32 before schema publication", got)
	}

	if result := RebuildTable(tbl, true, false); strings.Contains(result, "errors:") {
		t.Fatalf("cache table rebuild failed: %s", result)
	}
	if got := counting.calls.Load(); got != 1 {
		t.Fatalf("rebuild published schema %d times, want one deduplicated snapshot", got)
	}
	if db.schemaDirty.Load() {
		t.Fatal("rebuild left the buffered schema dirty")
	}
}

func TestOverflowRebuildSchemaFailureKeepsWritesRecoverable(t *testing.T) {
	oldShardSize := Settings.ShardSize
	Settings.ShardSize = 2
	t.Cleanup(func() { Settings.ShardSize = oldShardSize })
	tbl, persistence := createDurabilityTestTable(t, "toverflowsavefailure", 2)
	failing := &failSchemaWritePersistence{PersistenceEngine: persistence, failAt: 2}
	tbl.schema.persistence = failing

	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(3), scm.NewString("starts-overflow-rebuild")}}, nil, scm.NewNil(), false, nil)
	deadline := time.Now().Add(5 * time.Second)
	for tbl.overflowRebuilds.Load() > 0 {
		if time.Now().After(deadline) {
			t.Fatal("overflow rebuild did not finish")
		}
		runtime.Gosched()
	}
	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(4), scm.NewString("after-failed-save")}}, nil, scm.NewNil(), false, nil)
	if got := tbl.Count(); got != 4 {
		t.Fatalf("live count after failed overflow schema save = %d, want 4", got)
	}

	reloaded := reloadTableFromPersistence(t, "toverflowsavefailure", persistence)
	if got := reloaded.Count(); got != 4 {
		t.Fatalf("reloaded count after failed overflow schema save = %d, want 4; writes reached only unpublished shards", got)
	}
}

func TestACIDRolledBackInsertDoesNotReplayAfterRestart(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "tacidrollbackreload", 0)
	session := scm.NewSession()
	tx := NewTxContext(TxACID)
	tx.Session = session
	scm.Apply(session, scm.NewString("__memcp_tx"), scm.NewAny(tx))
	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewString("rolled-back")}}, nil, scm.NewNil(), false, nil, tx)
	tx.Rollback()
	if got := tbl.Count(); got != 0 {
		t.Fatalf("live count after ACID rollback = %d, want 0", got)
	}

	reloaded := reloadTableFromPersistence(t, "tacidrollbackreload", persistence)
	if got := reloaded.Count(); got != 0 {
		t.Fatalf("reloaded count after ACID rollback = %d, want 0; uncommitted insert was replayed", got)
	}
}

func TestCursorRollbackOfMainDeleteSurvivesRestart(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "tcursorrollbackreload", 1)
	if result := RebuildTable(tbl, true, false); strings.Contains(result, "errors:") {
		t.Fatalf("preparing main storage failed: %s", result)
	}
	shard := tbl.ActiveShards()[0]
	if shard.main_count != 1 {
		t.Fatalf("rebuilt main count = %d, want 1", shard.main_count)
	}
	tx := NewTxContext(TxCursorStability)
	if !scm.ToBool(shard.UpdateFunction(0, false, false, tx)()) {
		t.Fatal("cursor-stability delete did not change the row")
	}
	tx.Rollback()
	if got := tbl.Count(); got != 1 {
		t.Fatalf("live count after cursor rollback = %d, want 1", got)
	}

	reloaded := reloadTableFromPersistence(t, "tcursorrollbackreload", persistence)
	if got := reloaded.Count(); got != 1 {
		t.Fatalf("reloaded count after cursor rollback = %d, want 1; rollback was not represented in WAL", got)
	}
}
