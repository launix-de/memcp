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

import "sync"
import "testing"

import "github.com/google/btree"
import "github.com/launix-de/memcp/scm"

func TestStorageIndexSavingsAccumulateConcurrently(t *testing.T) {
	idx := new(StorageIndex)
	const workers = 8
	const additions = 1000
	var done sync.WaitGroup
	done.Add(workers)
	for range workers {
		go func() {
			defer done.Done()
			for range additions {
				idx.addSavings(nil, 1)
			}
		}()
	}
	done.Wait()
	if got, want := idx.loadSavings(), float64(workers*additions); got != want {
		t.Fatalf("concurrent savings = %v, want %v", got, want)
	}
}

func TestPlannerIndexProbeDoesNotIncreaseIndexSavings(t *testing.T) {
	Init(scm.Globalenv)
	dbname := "test_estimate_no_savings"
	CreateDatabase(dbname, true)
	defer databases.Remove(dbname)

	tbl, _ := CreateTable(dbname, "items", Memory, true)
	tbl.CreateColumn("id", "INT", nil, nil)
	rows := make([][]scm.Scmer, 0, 16)
	for i := 0; i < 16; i++ {
		rows = append(rows, []scm.Scmer{scm.NewInt(int64(i))})
	}
	tbl.Insert([]string{"id"}, rows, nil, scm.NewNil(), false, nil)

	shard := tbl.Shards[0]
	bounds := analyzedBoundaries{{col: "id", matcher: EqualMatcher, lower: scm.NewInt(5), lowerInclusive: true, upper: scm.NewInt(5), upperInclusive: true}}
	var buf [8]uint32
	shard.mu.RLock()
	shard.iterateIndex(nil, runtimeScanAccess(bounds), len(shard.deltaRows()), buf[:], 0, nil, func(batch []uint32) bool {
		return true
	})
	shard.mu.RUnlock()
	if len(shard.Indexes) != 1 {
		t.Fatalf("metadata indexes = %d, want 1", len(shard.Indexes))
	}
	if savings := shard.Indexes[0].loadSavings(); savings != 0 {
		t.Fatalf("estimate must not count as index usage, savings = %v", savings)
	}
	if len(shard.Indexes[0].ColMatchers) != 0 {
		t.Fatalf("sorted index persisted redundant matcher metadata: %#v", shard.Indexes[0].ColMatchers)
	}
	if got := shard.Indexes[0].String(); got != "id" {
		t.Fatalf("sorted index description = %q, want %q", got, "id")
	}
	if shard.Indexes[0].baseState.active {
		t.Fatal("estimate must not materialize a cold auto-index before real usage")
	}

	columns := []string{"id"}
	values := []scm.Scmer{scm.NewInt(5)}
	access := exactScanAccess(newExactScanAccessSchema(columns), values)
	if _, present := shard.GetRecordidForUnique(access, nil, false); !present {
		t.Fatal("warm unique point probe did not find its row")
	}
	if allocations := testing.AllocsPerRun(1000, func() {
		if _, present := shard.GetRecordidForUnique(access, nil, false); !present {
			t.Fatal("unique point probe did not find its row")
		}
	}); allocations != 0 {
		t.Fatalf("warm unique point probe allocated %.2f times per run, want 0", allocations)
	}
}

func TestStorageIndexComputeSizeCountsDeltaBtreeAndHooks(t *testing.T) {
	idx := &StorageIndex{}
	state := &idx.baseState
	state.deltaBtree = btree.NewG[indexPair](8, func(a, b indexPair) bool {
		return a.itemid < b.itemid
	})
	state.deltaBtree.ReplaceOrInsert(indexPair{itemid: 1, data: []scm.Scmer{scm.NewInt(1)}})
	state.deltaBtree.ReplaceOrInsert(indexPair{itemid: 2, data: []scm.Scmer{scm.NewInt(2)}})
	state.indexHookBytes.Store(4096)

	baseOnly := uint(24 * 8)
	if got := idx.ComputeSize(); got <= baseOnly+4096 {
		t.Fatalf("ComputeSize() = %d, want larger than base plus hooks %d", got, baseOnly+4096)
	}
}

// Lazy index declarations are appended by concurrent scans under indexMutex
// while they hold only shared shard ownership. Both cache registration and
// diagnostics must snapshot that directory with its actual publication owner.
func TestShardAccountingConcurrentIndexPublication(t *testing.T) {
	shard := &storageShard{}
	shard.setState(SHARED)
	const count = 512
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		<-start
		for range count {
			index := &StorageIndex{}
			shard.mu.RLock()
			shard.indexMutex.Lock()
			shard.Indexes = append(shard.Indexes, index)
			shard.indexMutex.Unlock()
			shard.mu.RUnlock()
		}
	}()
	close(start)
	for range count {
		shard.exclusiveSize()
		shard.ComputeSize()
	}
	workers.Wait()
	var children uint
	for _, index := range shard.Indexes {
		children += index.ComputeSize()
	}
	if got, want := shard.ComputeSize(), shard.exclusiveSize()+children; got != want {
		t.Fatalf("inclusive shard bytes = %d, want parent plus all published indices %d", got, want)
	}
}

// A delta insertion must not enter callbacks with the shard write lock held.
// The comparator stands in for an ORC reader that scans its own source shard.
func TestComputedIndexDeltaDoesNotInvokeComparator(t *testing.T) {
	index := &StorageIndex{}
	index.baseState.active = true
	index.baseState.deltaBtree = btree.NewG[indexPair](8, func(a, b indexPair) bool {
		t.Fatal("delta maintenance invoked a computed-key comparator")
		return false
	})
	index.baseState.deltaBtree.ReplaceOrInsert(indexPair{itemid: 0})
	if charged := index.insertDelta(1, []scm.Scmer{scm.NewInt(7)}); charged != 0 {
		t.Fatalf("invalidated index charged delta allocation: %d", charged)
	}
	if index.baseState.active || index.baseState.deltaBtree != nil {
		t.Fatal("computed index remained usable after its source delta changed")
	}
}
