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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
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
		tbl.ComputeColumn("derived", []string{"base"}, computor, nil, scm.NewNil())
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
	tbl.ComputeColumn("derived", []string{"base"}, computor, nil, scm.NewNil())
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
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("table"), scm.NewString("compconc"), scm.NewString("feature")}),
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
	keytable.ComputeColumn("counted", []string{"1"}, computor, nil, scm.NewNil())

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
					keytable.ComputeColumn("counted", []string{"1"}, computor, nil, scm.NewNil())
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

	tbl.ComputeColumn("cached", []string{"val"}, computor, []string{"val"}, filterGT2)
	if got := computeCalls.Load(); got != 2 {
		t.Fatalf("first filtered compute invoked computor %d times, want 2", got)
	}
	shard := tbl.Shards[0]
	shard.mu.RLock()
	canonicalProxy := shard.columns["cached"]
	shard.mu.RUnlock()

	tbl.ComputeColumn("cached", []string{"val"}, computor, []string{"val"}, filterGT2)
	if got := computeCalls.Load(); got != 4 {
		t.Fatalf("repeated filtered compute invoked computor %d times, want 4 total", got)
	}
	shard.mu.RLock()
	repeatedProxy := shard.columns["cached"]
	shard.mu.RUnlock()
	if repeatedProxy != canonicalProxy {
		t.Fatal("repeated filtered compute replaced the canonical compute proxy")
	}

	tbl.ComputeColumn("cached", []string{"val"}, computor, []string{"val"}, filterGT1)
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

	tbl.ComputeColumn("cached", []string{"val"}, computor, nil, scm.NewNil())
	if got := computeCalls.Load(); got != 3 {
		t.Fatalf("first unfiltered compute invoked computor %d times, want 3", got)
	}

	// Planner-generated cache plans issue the same createcolumn on every use.
	// A complete generation is an O(1) no-op; proving that fact must not walk
	// every delta record or invoke the computor again.
	tbl.ComputeColumn("cached", []string{"val"}, computor, nil, scm.NewNil())
	if got := computeCalls.Load(); got != 3 {
		t.Fatalf("repeated unfiltered compute invoked computor %d times, want 3 total", got)
	}

	tbl.Insert([]string{"id", "val"}, [][]scm.Scmer{
		{scm.NewInt(4), scm.NewInt(40)},
	}, nil, scm.NewNil(), false, nil)
	tbl.ComputeColumn("cached", []string{"val"}, computor, nil, scm.NewNil())
	if got := computeCalls.Load(); got != 4 {
		t.Fatalf("post-insert repair invoked computor %d times, want 4 total", got)
	}

	reader := tbl.ActiveShards()[0].ColumnReaderTx(nil, "cached", false)
	if got := reader(3).Int(); got != 80 {
		t.Fatalf("repaired appended computed value = %d, want 80", got)
	}
}

// TestOrderedComputeInvalidationReadsDeltaUnderShardLock races ordinary
// concurrent INSERTs (which grow shard.inserts/deltaColumns under s.mu.Lock,
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

	// Writer: keeps extending shard.inserts/deltaColumns, exactly like an
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
	// concurrently with the writer's mutation of shard.inserts/deltaColumns
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
