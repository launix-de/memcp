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
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
	"github.com/launix-de/memcp/storage/mosi"
)

func setupComputeConcurrencyTest(t *testing.T) func() {
	t.Helper()
	dir, err := os.MkdirTemp("", "memcp-compute-concurrency-*")
	if err != nil {
		t.Fatal(err)
	}
	oldBasepath := Basepath
	Basepath = dir
	Init(scm.Globalenv)
	LoadDatabases()
	return func() {
		databases.Remove("compconc")
		Basepath = oldBasepath
		_ = os.RemoveAll(dir)
	}
}

func TestComputedColumnRejectsImplicitExecutionContext(t *testing.T) {
	for _, symbol := range []string{"tx", "session", "__memcp_tx"} {
		computor := scm.NewProcStruct(scm.Proc{
			Params: scm.NewSlice(nil),
			Body:   scm.NewSymbol(symbol),
			En:     &scm.Globalenv,
		})
		if !hasImplicitComputeContext(computor) {
			t.Fatalf("computed-column dependency on %s was accepted", symbol)
		}
	}

	stateless := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("value")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("+"), scm.NewSymbol("value"), scm.NewInt(1),
		}),
		En: &scm.Globalenv,
	})
	if hasImplicitComputeContext(stateless) {
		t.Fatal("ordinary computed-column parameter was treated as request context")
	}
	explicitSession := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("session")}),
		Body:   scm.NewSlice([]scm.Scmer{scm.NewSymbol("session"), scm.NewString("key")}),
		En:     &scm.Globalenv,
	})
	if hasImplicitComputeContext(explicitSession) {
		t.Fatal("explicit computed-column session parameter was treated as a closure dependency")
	}
}

func TestComputeColumnRejectsRequestContextBeforePublishing(t *testing.T) {
	cleanup := setupComputeConcurrencyTest(t)
	defer cleanup()

	CreateDatabase("compconc", false)
	tbl, _ := CreateTable("compconc", "stateless", Memory, false)
	tbl.CreateColumn("base", "INT", nil, nil)
	tbl.CreateColumn("derived", "INT", nil, nil)
	computor := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("base")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("session"), scm.NewString("request-value"),
		}),
		En: &scm.Globalenv,
	})

	var failure any
	func() {
		defer func() { failure = recover() }()
		tbl.ComputeColumn(nil, "derived", []string{"base"}, computor, nil, scm.NewNil())
	}()
	if failure == nil {
		t.Fatal("computed column accepted an implicit session dependency")
	}
	shard := tbl.Shards[0]
	shard.mu.RLock()
	_, published := shard.columns["derived"].(*StorageComputeProxy)
	shard.mu.RUnlock()
	if published {
		t.Fatal("invalid computed-column callback was published before validation")
	}
}

func BenchmarkComputedColumnRepair(b *testing.B) {
	oldBasepath := Basepath
	Basepath = b.TempDir()
	defer func() { Basepath = oldBasepath }()
	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("bench_stateless_compute")

	CreateDatabase("bench_stateless_compute", false)
	tbl, _ := CreateTable("bench_stateless_compute", "items", Memory, false)
	tbl.CreateColumn("base", "INT", nil, nil)
	tbl.CreateColumn("derived", "INT", nil, nil)
	tbl.Insert([]string{"base"}, [][]scm.Scmer{{scm.NewInt(41)}}, nil, scm.NewNil(), false, nil)
	if result := GetDatabase("bench_stateless_compute").rebuild(true, false, true); len(result.errors) > 0 {
		b.Fatalf("rebuild errors: %v", result.errors)
	}
	computor := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("base")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("+"), scm.NewSymbol("base"), scm.NewInt(1),
		}),
		En: &scm.Globalenv,
	})
	tbl.ComputeColumn(nil, "derived", []string{"base"}, computor, nil, scm.NewNil())
	proxy := tbl.Shards[0].columns["derived"].(*StorageComputeProxy)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proxy.Invalidate(0)
	}
}

func countCollapsedComputor() scm.Scmer {
	filterColumns := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("list"),
		scm.NewString("uid"),
		scm.NewString("form"),
		scm.NewString("subid"),
		scm.NewString("k"),
	})
	filter := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("uid"),
			scm.NewSymbol("form"),
			scm.NewSymbol("subid"),
			scm.NewSymbol("k"),
		}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("and"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewNthLocalVar(0), scm.NewNil()}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewNthLocalVar(1), scm.NewString("wf:userconfig:edit")}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewNthLocalVar(2), scm.NewString("Offers")}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal??"), scm.NewNthLocalVar(3), scm.NewString("collapsed")}),
		}),
		En:      &scm.Globalenv,
		NumVars: 4,
	})
	accessSchema, accessBindings, _ := compileScanAccess(filterColumns, filter)
	accessValues := scm.NewSlice(append([]scm.Scmer{scm.NewSymbol("list")}, accessBindings...))
	mapReduceFn := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("acc")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("+"), scm.NewSymbol("acc"), scm.NewInt(1),
		}),
		En:      &scm.Globalenv,
		NumVars: 1,
	})
	return scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("group")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("scan"),
			scm.NewNil(),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("table"), scm.NewString("compconc"), scm.NewBool(false), scm.NewString("feature"), scm.NewBool(false)}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), accessSchema}),
			accessValues,
			filterColumns,
			filter,
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("list")}),
			mapReduceFn,
			scm.NewInt(0),
			scm.NewSymbol("+"),
			scm.NewBool(false),
		}),
		En:      &scm.Globalenv,
		NumVars: 1,
	})
}

func TestGlobalAggregateComputeAndInsertDoNotDeadlock(t *testing.T) {
	defer setupComputeConcurrencyTest(t)()

	CreateDatabase("compconc", false)
	src, _ := CreateTable("compconc", "feature", Memory, false)
	src.CreateColumn("uid", "INT", nil, nil)
	src.CreateColumn("form", "TEXT", nil, nil)
	src.CreateColumn("subid", "TEXT", nil, nil)
	src.CreateColumn("k", "TEXT", nil, nil)
	src.CreateColumn("value", "TEXT", nil, nil)

	keytable, _ := CreateTable("compconc", ".feature:(1)", Memory, true)
	keytable.CreateColumn("1", "ANY", nil, nil)
	keytable.CreateColumn("counted", "ANY", nil, nil)
	keytable.Insert([]string{"1"}, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)

	computor := countCollapsedComputor()
	keytable.ComputeColumn(nil, "counted", []string{"1"}, computor, nil, scm.NewNil())

	row := []scm.Scmer{
		scm.NewNil(),
		scm.NewString("wf:userconfig:edit"),
		scm.NewString("Offers"),
		scm.NewString("collapsed"),
		scm.NewString("0"),
	}

	const computeWorkers = 4
	const insertWorkers = 4
	const iterations = 25

	errCh := make(chan error, computeWorkers+insertWorkers)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for worker := 0; worker < computeWorkers; worker++ {
		wg.Add(1)
		go func(worker int) func() {
			return func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						errCh <- fmt.Errorf("compute worker %d panic: %v", worker, r)
					}
				}()
				<-start
				for iter := 0; iter < iterations; iter++ {
					keytable.ComputeColumn(nil, "counted", []string{"1"}, computor, nil, scm.NewNil())
				}
			}
		}(worker)()
	}

	for worker := 0; worker < insertWorkers; worker++ {
		wg.Add(1)
		go func(worker int) func() {
			return func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						errCh <- fmt.Errorf("insert worker %d panic: %v", worker, r)
					}
				}()
				<-start
				for iter := 0; iter < iterations; iter++ {
					src.Insert([]string{"uid", "form", "subid", "k", "value"}, [][]scm.Scmer{row}, nil, scm.NewNil(), false, nil)
				}
			}
		}(worker)()
	}

	close(start)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errCh:
		t.Fatal(err)
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent global aggregate recompute and insert timed out")
	}

	select {
	case err := <-errCh:
		t.Fatal(err)
	default:
	}
}

func TestInsertSanitizationDoesNotMutateCallerRows(t *testing.T) {
	defer setupComputeConcurrencyTest(t)()

	CreateDatabase("compconc", false)
	tbl, _ := CreateTable("compconc", "caller_rows", Memory, false)
	tbl.CreateColumn("value", "INT", nil, nil)
	row := []scm.Scmer{scm.NewString("7")}
	values := [][]scm.Scmer{row}

	tbl.Insert([]string{"value"}, values, nil, scm.NewNil(), false, nil)

	if !row[0].IsString() || row[0].String() != "7" {
		t.Fatalf("insert rewrote caller-owned row to %v", row[0])
	}
	if len(values) != 1 || len(values[0]) != 1 || !values[0][0].IsString() {
		t.Fatalf("insert rewrote caller-owned values slice: %v", values)
	}
}

func TestComputeColumnRejectsColdRemoteShardBeforeMaterialization(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tcomputeremote", 4)
	RebuildTable(tbl, true, false)
	shard := tbl.ActiveShards()[0]
	var freed [numEvictableTypes]int64
	if !shardCleanup(shard, &freed) || shard.state() != COLD {
		t.Fatal("fixture did not become cold")
	}
	setTestScanTarget(shard, func(mosiResource, shardScanInvocation) scanResult {
		t.Fatal("computed-column preparation unexpectedly dispatched a scan")
		return scanResult{}
	})
	var calls int
	var failure any
	func() {
		defer func() { failure = recover() }()
		shard.ComputeColumn(nil, "cached", []string{"id"}, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			calls++
			return a[0]
		}), nil, scm.NewNil())
	}()
	delegation, ok := failure.(shardReadDelegation)
	if !ok || delegation.target != shard.scanTarget.Load() {
		t.Fatalf("preparation bypassed acquisition: panic = %v", failure)
	}
	if calls != 0 || shard.state() != COLD || shard.cacheReaders.Load() != 0 {
		t.Fatalf("rejected preparation touched payload: calls=%d state=%v readers=%d", calls, shard.state(), shard.cacheReaders.Load())
	}
}

func TestComputeColumnRetainsPayloadThroughCallbackAndPanic(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tcomputeread", 4)
	RebuildTable(tbl, true, false)
	shard := tbl.ActiveShards()[0]
	r := &replicationManager{cells: make(map[string]*replicaCell), directory: replicaDirectory{Records: make(map[string]replicaRecord)}}
	cell := r.shardCell(shard)
	cell.ownstate.Initialize(mosi.S)
	previous := replication.Swap(r)
	defer replication.Store(previous)
	for _, fail := range []bool{false, true} {
		name := fmt.Sprintf("cached_%v", fail)
		calls := 0
		var failure any
		func() {
			defer func() { failure = recover() }()
			shard.ComputeColumn(nil, name, []string{"id"}, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
				calls++
				if shard.cacheReaders.Load() == 0 {
					t.Error("computed callback has no acquired payload capability")
				}
				if cell.ownstate.Phase() != mosi.S {
					t.Error("derived cache fill changed native read permission")
				}
				if fail {
					panic("compute callback failed")
				}
				return scm.NewInt(a[0].Int() * 2)
			}), nil, scm.NewNil())
		}()
		if fail && failure != "compute callback failed" || !fail && failure != nil {
			t.Fatalf("callback failure changed: %v", failure)
		}
		if calls == 0 || shard.cacheReaders.Load() != 0 {
			t.Fatalf("preparation calls=%d leaked readers=%d", calls, shard.cacheReaders.Load())
		}
		if !fail {
			reader := shard.ColumnReaderTx(nil, name, false, ReadAuto)
			if got := reader.GetValue(0).Int(); got != 2 {
				t.Errorf("computed value = %d, want 2", got)
			}
			ReleaseColumnReader(reader)
		}
	}
}

func TestComputeColumnCancellationKeepsSharedAcquisition(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tcomputecancel", 1)
	tbl.CreateColumn("cached", "INT", nil, nil)
	cluster := &nodeCluster{self: clusterMember{ID: "memcp://" + uuid.NewString()}, ctx: context.Background()}
	cluster.initializeManager()
	owner := "memcp://" + uuid.NewString()
	peer := &mosiPeer{done: make(chan struct{}), outgoing: make(chan mosiMessage, 1)}
	cluster.view.Store(&clusterView{links: []*clusterLink{{member: clusterMember{ID: owner}, peer: peer}}})
	shard := tbl.ActiveShards()[0]
	resource := mosiResource{Database: tbl.schema.Name, Table: tbl.Name, Shard: shard.uuid.String()}
	cell := cluster.manager.shardCell(shard)
	cluster.manager.updateRecord(resource, func(record *replicaRecord) bool {
		record.Owner, record.Epoch = owner, 1
		return true
	})
	request, _ := cell.ownstate.Request(mosi.S)
	previous := replication.Swap(cluster.manager)
	defer replication.Store(previous)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ss := &scm.SessionState{}
	seq := ss.BeginQuery("Test", "computed-column acquisition")
	ss.SetQueryContext(seq, ctx)
	tx := NewTxContext(TxCursorStability)
	tx.beginQuery(ss, seq, "computed-column acquisition")
	defer tx.endQuery(seq)
	var failure any
	func() {
		defer func() { failure = recover() }()
		tbl.ComputeColumn(tx, "cached", []string{"id"}, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
			t.Error("cancelled preparation ran a computed callback")
			return scm.NewNil()
		}), nil, scm.NewNil())
	}()
	if err, ok := failure.(scanError); !ok || err.r != context.Canceled {
		t.Fatalf("preparation lost query cancellation: %v", failure)
	}
	if cell.ownstate.Phase() != mosi.SR || len(peer.outgoing) != 0 || shard.cacheReaders.Load() != 0 {
		t.Fatal("cancelled preparation changed the shared flight or retained payload")
	}
	select {
	case <-request.Flight().Done():
		t.Fatal("one cancelled preparation completed the shared acquisition")
	default:
	}
	if _, err := cell.ownstate.Grant(request.Flight().ID(), mosi.S, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if cluster.manager.readShard(shard, nil) != shard {
		t.Fatal("remaining readers could not use the completed shared acquisition")
	}
}

func TestComputeColumnWaitsForStartedShardsAfterFailure(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tcomputedrain", 4)
	tbl.CreateColumn("cached", "INT", nil, nil)
	RebuildTable(tbl, true, false)
	if !tbl.beginManualRepartition() {
		t.Fatal("manual repartition was not claimed")
	}
	tbl.repartition([]shardDimension{{Column: "id", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(2)}}})
	if len(tbl.ActiveShards()) != 2 {
		t.Fatal("fixture did not produce two shards")
	}
	slowStarted := make(chan struct{})
	failed := make(chan struct{})
	release := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	defer unblock()
	start := sync.OnceFunc(func() { close(slowStarted) })
	fail := sync.OnceFunc(func() { close(failed) })
	finished := make(chan any, 1)
	go func() {
		defer func() { finished <- recover() }()
		tbl.ComputeColumn(nil, "cached", []string{"id"}, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			if a[0].Int() <= 2 {
				<-slowStarted
				fail()
				panic("compute shard failed")
			}
			start()
			<-release
			return a[0]
		}), nil, scm.NewNil())
	}()
	select {
	case <-failed:
	case failure := <-finished:
		t.Fatalf("preparation stopped before shard jobs synchronized: %v", failure)
	case <-time.After(5 * time.Second):
		t.Fatal("shard jobs did not start")
	}
	var failure any
	returned := false
	select {
	case failure = <-finished:
		returned = true
		t.Error("preparation returned while another shard callback was still running")
	case <-time.After(100 * time.Millisecond):
	}
	unblock()
	if !returned {
		select {
		case failure = <-finished:
		case <-time.After(5 * time.Second):
			t.Fatal("preparation did not finish after remaining shard completed")
		}
	}
	if err, ok := failure.(scanError); !ok || err.r != "compute shard failed" {
		t.Fatalf("lost original shard failure: %v", failure)
	}
}

func TestFilteredComputeColumnConservativelyRecomputesRepeatedFilter(t *testing.T) {
	defer setupComputeConcurrencyTest(t)()

	CreateDatabase("compconc", false)
	tbl, _ := CreateTable("compconc", "filtered", Memory, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("val", "INT", nil, nil)
	tbl.CreateColumn("cached", "INT", nil, nil)
	tbl.Insert([]string{"id", "val"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(1)},
		{scm.NewInt(2), scm.NewInt(2)},
		{scm.NewInt(3), scm.NewInt(3)},
		{scm.NewInt(4), scm.NewInt(4)},
	}, nil, scm.NewNil(), false, nil)

	var computeCalls atomic.Int64
	computor := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		computeCalls.Add(1)
		return a[0]
	})
	filterGT2 := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("val")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol(">"),
			scm.NewNthLocalVar(0),
			scm.NewInt(2),
		}),
		En:      &scm.Globalenv,
		NumVars: 1,
	})
	filterGT1 := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("val")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol(">"),
			scm.NewNthLocalVar(0),
			scm.NewInt(1),
		}),
		En:      &scm.Globalenv,
		NumVars: 1,
	})

	tbl.ComputeColumn(nil, "cached", []string{"val"}, computor, []string{"val"}, filterGT2)
	if got := computeCalls.Load(); got != 2 {
		t.Fatalf("first filtered compute invoked computor %d times, want 2", got)
	}
	shard := tbl.Shards[0]
	shard.mu.RLock()
	canonicalProxy := shard.columns["cached"]
	shard.mu.RUnlock()

	tbl.ComputeColumn(nil, "cached", []string{"val"}, computor, []string{"val"}, filterGT2)
	if got := computeCalls.Load(); got != 4 {
		t.Fatalf("repeated filtered compute invoked computor %d times, want 4 total", got)
	}
	shard.mu.RLock()
	repeatedProxy := shard.columns["cached"]
	shard.mu.RUnlock()
	if repeatedProxy != canonicalProxy {
		t.Fatal("repeated filtered compute replaced the canonical compute proxy")
	}

	tbl.ComputeColumn(nil, "cached", []string{"val"}, computor, []string{"val"}, filterGT1)
	if got := computeCalls.Load(); got != 7 {
		t.Fatalf("changing filtered materialization invoked computor %d times, want 7 total", got)
	}
	shard.mu.RLock()
	changedFilterProxy := shard.columns["cached"]
	shard.mu.RUnlock()
	if changedFilterProxy != canonicalProxy {
		t.Fatal("changed filter replaced the canonical compute proxy")
	}
}

func TestUnfilteredComputeColumnReusesCompletePreparationUntilMutation(t *testing.T) {
	defer setupComputeConcurrencyTest(t)()

	CreateDatabase("compconc", false)
	tbl, _ := CreateTable("compconc", "unfiltered", Memory, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("val", "INT", nil, nil)
	tbl.CreateColumn("cached", "INT", nil, nil)
	tbl.Insert([]string{"id", "val"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(10)},
		{scm.NewInt(2), scm.NewInt(20)},
		{scm.NewInt(3), scm.NewInt(30)},
	}, nil, scm.NewNil(), false, nil)

	var computeCalls atomic.Int64
	computor := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		computeCalls.Add(1)
		return scm.NewInt(a[0].Int() * 2)
	})

	tbl.ComputeColumn(nil, "cached", []string{"val"}, computor, nil, scm.NewNil())
	if got := computeCalls.Load(); got != 3 {
		t.Fatalf("first unfiltered compute invoked computor %d times, want 3", got)
	}

	// Planner-generated cache plans issue the same createcolumn on every use.
	// A complete generation is an O(1) no-op; proving that fact must not walk
	// every delta record or invoke the computor again.
	tbl.ComputeColumn(nil, "cached", []string{"val"}, computor, nil, scm.NewNil())
	if got := computeCalls.Load(); got != 3 {
		t.Fatalf("repeated unfiltered compute invoked computor %d times, want 3 total", got)
	}

	tbl.Insert([]string{"id", "val"}, [][]scm.Scmer{
		{scm.NewInt(4), scm.NewInt(40)},
	}, nil, scm.NewNil(), false, nil)
	tbl.ComputeColumn(nil, "cached", []string{"val"}, computor, nil, scm.NewNil())
	if got := computeCalls.Load(); got != 4 {
		t.Fatalf("post-insert repair invoked computor %d times, want 4 total", got)
	}

	reader := tbl.ActiveShards()[0].ColumnReaderTx(nil, "cached", false, ReadAuto).GetValue
	if got := reader(3).Int(); got != 80 {
		t.Fatalf("repaired appended computed value = %d, want 80", got)
	}
}

// TestOrderedComputeInvalidationReadsDeltaUnderShardLock races ordinary
// concurrent INSERTs (which grow shard.deltaRows()/deltaColumns under s.mu.Lock,
// each also firing its own AFTER-INSERT invalidateorc trigger) against an
// explicit, concurrent invalidateORCFromSortKey call — the same entry point
// the invalidateorc SQL builtin uses, and thus something two overlapping
// real INSERTs can legitimately trigger concurrently on the same shard.
// invalidateORCFromSortKey previously read requestShard.columns/getDelta
// after releasing s.mu.RLock, racing the concurrent INSERT's writes to the
// same maps/slice. Run with -race to reproduce; it must also converge on the
// correct running sum, since a lost or corrupted read would otherwise
// silently produce a wrong aggregate instead of a detectable panic.
func TestOrderedComputeInvalidationReadsDeltaUnderShardLock(t *testing.T) {
	defer setupComputeConcurrencyTest(t)()

	CreateDatabase("compconc", false)
	tbl, _ := CreateTable("compconc", "orcrace", Memory, false)
	tbl.CreateColumn("grp", "INT", nil, nil)
	tbl.CreateColumn("amount", "INT", nil, nil)
	tbl.CreateColumn("running", "INT", nil, nil)
	tbl.Insert([]string{"grp", "amount"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(10)},
	}, nil, scm.NewNil(), false, nil)

	mapReduceFn := scm.Eval(scm.Read("test", "(lambda (acc $set v) (begin (define new_acc (+ acc v)) ($set new_acc) new_acc))"), &scm.Globalenv)
	options := scm.NewSlice([]scm.Scmer{
		scm.NewString("sortcols"), scm.NewSlice([]scm.Scmer{scm.NewString("grp")}),
		scm.NewString("sortdirs"), scm.NewSlice([]scm.Scmer{scm.NewBool(false)}),
		scm.NewString("partitioncount"), scm.NewInt(1),
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
		t.Fatal("createcolumn should upgrade running to a partitioned ORC column")
	}

	shard := tbl.Shards[0]
	const rounds = 300
	var wg sync.WaitGroup
	wg.Add(2)

	// Writer: keeps extending shard.deltaRows()/deltaColumns, exactly like an
	// ordinary concurrent session issuing INSERTs on the same shard.
	go func() {
		defer wg.Done()
		for i := 0; i < rounds; i++ {
			tbl.Insert([]string{"grp", "amount"}, [][]scm.Scmer{
				{scm.NewInt(1), scm.NewInt(1)},
			}, nil, scm.NewNil(), false, nil)
		}
	}()

	// Reader: repeatedly drives invalidateORCFromSortKey directly (the same
	// entry point the invalidateorc SQL builtin uses from an AFTER-INSERT
	// trigger), so the exact getDelta call fixed in compute.go runs
	// concurrently with the writer's mutation of shard.deltaRows()/deltaColumns
	// above, and concurrently with the writer's own trigger-fired calls to
	// the same function.
	go func() {
		defer wg.Done()
		for i := 0; i < rounds; i++ {
			tbl.invalidateORCFromSortKey("running", []scm.Scmer{scm.NewInt(1)})
		}
	}()

	wg.Wait()

	shard.mu.RLock()
	proxy, ok := shard.columns["running"].(*StorageComputeProxy)
	shard.mu.RUnlock()
	if !ok {
		t.Fatal("running column lost its ORC proxy after concurrent invalidation")
	}
	// running is a cumulative sum within the single "grp" partition, so only
	// the last inserted row's value reflects every prior row's amount (row 0
	// is always just its own amount, by definition of a running sum).
	lastIdx := uint32(rounds)
	want := int64(10 + rounds)
	if got := proxy.GetValue(lastIdx).Int(); got != want {
		t.Fatalf("running[%d] after %d concurrent inserts = %d, want %d", lastIdx, rounds, got, want)
	}
}

// Compute-cache publication is not a stored-row mutation. Each mapper variant
// must retain the selected physical identity and fill its cache without trying
// to upgrade a caller's shard read lock, even with a writer already queued.
func TestComputeSetDoesNotUpgradeShardReadOwnership(t *testing.T) {
	for _, main := range []bool{true, false} {
		for _, batch := range []bool{false, true} {
			t.Run(fmt.Sprintf("main=%t/batch=%t", main, batch), func(t *testing.T) {
				shard := &storageShard{t: &table{PersistencyMode: Memory}, main_count: 1,
					columns: make(map[string]ColumnStorage)}
				shard.publishDeltaRows([][]scm.Scmer{{}})
				shard.setState(WRITE)
				proxy := &StorageComputeProxy{shard: shard, colName: "cached", isOrdered: true,
					count: 1, delta: make(map[uint32]scm.Scmer)}
				shard.columns["cached"] = proxy
				callback := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
					scm.Apply(args[1], scm.NewInt(42))
					return args[0]
				})
				mapper := shard.OpenMapReducer([]string{"$set:cached"}, callback, false, 0, nil, nil)
				defer mapper.Close()
				id := uint32(0)
				if !main {
					id = 1
				}
				// A cached value belongs to its captured generation, including an
				// identity hidden by a later DELETE. Filling it must not revive or
				// forward that stored row. Scans own visibility filtering.
				shard.deletions.Set(uint(id), true)
				shard.mu.RLock()
				writerDone := make(chan struct{})
				go func() {
					shard.mu.Lock()
					shard.mu.Unlock()
					close(writerDone)
				}()
				deadline := time.Now().Add(time.Second)
				for shard.mu.TryRLock() {
					shard.mu.RUnlock()
					if time.Now().After(deadline) {
						shard.mu.RUnlock()
						<-writerDone
						t.Fatal("writer did not queue")
					}
					runtime.Gosched()
				}
				done := make(chan struct{})
				go func() {
					if batch {
						mapper.Stream(scm.NewNil(), []uint32{id}, []uint32{0})
					} else {
						mapper.Stream(scm.NewNil(), []uint32{id}, nil)
					}
					close(done)
				}()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Error("compute-cache fill waited for the shard lock")
				}
				shard.mu.RUnlock()
				<-writerDone
				<-done
				if got := proxy.storedORCValue(id); !got.IsInt() || got.Int() != 42 {
					t.Fatalf("cache slot %d = %v, want 42", id, got)
				}
				if !shard.deletions.Get(uint(id)) {
					t.Fatal("cache fill changed stored-row visibility")
				}
			})
		}
	}
}
