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
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jtolds/gls"
	"github.com/launix-de/NonLockingReadMap"
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
	mapFn := scm.NewProcStruct(scm.Proc{
		Params:  scm.NewSlice([]scm.Scmer{}),
		Body:    scm.NewInt(1),
		En:      &scm.Globalenv,
		NumVars: 0,
	})
	return scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("group")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("scan"),
			scm.NewString("compconc"),
			scm.NewString("feature"),
			scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("list"),
				scm.NewString("uid"),
				scm.NewString("form"),
				scm.NewString("subid"),
				scm.NewString("k"),
			}),
			filter,
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("list")}),
			mapFn,
			scm.NewSymbol("+"),
			scm.NewInt(0),
			scm.NewNil(),
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
		gls.Go(func(worker int) func() {
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
		}(worker))
	}

	for worker := 0; worker < insertWorkers; worker++ {
		wg.Add(1)
		gls.Go(func(worker int) func() {
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
		}(worker))
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

func TestFilteredComputeColumnReusesSameFilterWithoutRecompute(t *testing.T) {
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
	if got := computeCalls.Load(); got != 2 {
		t.Fatalf("repeated filtered compute recomputed %d values, want cached no-op", got)
	}

	tbl.ComputeColumn("cached", []string{"val"}, computor, []string{"val"}, filterGT1)
	if got := computeCalls.Load(); got != 5 {
		t.Fatalf("changing filtered materialization invoked computor %d times, want 5 total", got)
	}
}

func TestComputeProxySerializeRoundTripPreservesFilteredMaterialization(t *testing.T) {
	filter := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("val")}),
		Body: scm.NewSlice([]scm.Scmer{
			scm.NewSymbol(">"),
			scm.NewNthLocalVar(0),
			scm.NewInt(2),
		}),
		En:      &scm.Globalenv,
		NumVars: 1,
	})
	computor := scm.NewProcStruct(scm.Proc{
		Params:  scm.NewSlice([]scm.Scmer{scm.NewSymbol("val")}),
		Body:    scm.NewNthLocalVar(0),
		En:      &scm.Globalenv,
		NumVars: 1,
	})
	proxy := &StorageComputeProxy{
		delta:      map[uint32]scm.Scmer{2: scm.NewInt(3), 3: scm.NewInt(4)},
		validMask:  NonLockingReadMap.NonBlockingBitMap{},
		computor:   computor,
		inputCols:  []string{"val"},
		filterCols: []string{"val"},
		filter:     filter,
		count:      4,
	}
	proxy.validMask.Set(2, true)
	proxy.validMask.Set(3, true)
	proxy.eagerMask.Set(2, true)
	proxy.eagerMask.Set(3, true)

	var buf bytes.Buffer
	proxy.Serialize(&buf)

	var magic uint8
	if err := binary.Read(&buf, binary.LittleEndian, &magic); err != nil {
		t.Fatal(err)
	}
	if magic != 50 {
		t.Fatalf("magic = %d, want 50", magic)
	}
	var roundTrip StorageComputeProxy
	if got := roundTrip.Deserialize(&buf); got != 4 {
		t.Fatalf("Deserialize count = %d, want 4", got)
	}
	if !stringSlicesEqual(roundTrip.filterCols, []string{"val"}) {
		t.Fatalf("filterCols = %v, want [val]", roundTrip.filterCols)
	}
	if !scmerJSONEqual(roundTrip.filter, filter) {
		t.Fatal("round-trip filter does not match original filtered materialization")
	}
	if !roundTrip.eagerMask.Get(2) || !roundTrip.eagerMask.Get(3) {
		t.Fatal("round-trip eagerMask lost filtered eager rows")
	}
}
