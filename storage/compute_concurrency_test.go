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
			scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("list"),
				scm.NewString("uid"),
				scm.NewString("form"),
				scm.NewString("subid"),
				scm.NewString("k"),
			}),
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

	tbl.ComputeColumn("cached", []string{"val"}, computor, []string{"val"}, filterGT2)
	if got := computeCalls.Load(); got != 4 {
		t.Fatalf("repeated filtered compute invoked computor %d times, want 4 total", got)
	}

	tbl.ComputeColumn("cached", []string{"val"}, computor, []string{"val"}, filterGT1)
	if got := computeCalls.Load(); got != 7 {
		t.Fatalf("changing filtered materialization invoked computor %d times, want 7 total", got)
	}
}
