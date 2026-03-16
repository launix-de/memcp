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
	"testing"
	"time"

	"github.com/jtolds/gls"
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
