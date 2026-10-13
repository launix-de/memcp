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
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

func TestCollectRelevantShardsUsesPartitionBounds(t *testing.T) {
	shards := make([]*storageShard, 5)
	for i := range shards {
		shards[i] = &storageShard{}
	}
	schema := []shardDimension{{
		Column:        "id",
		NumPartitions: len(shards),
		Pivots: []scm.Scmer{
			scm.NewInt(10),
			scm.NewInt(20),
			scm.NewInt(30),
			scm.NewInt(40),
		},
	}}

	tests := []struct {
		name       string
		boundary   analyzedBoundary
		wantShards []int
	}{
		{
			name: "point between late pivots",
			boundary: analyzedBoundary{
				col: "id", matcher: EqualMatcher,
				lower: scm.NewInt(35), lowerInclusive: true,
				upper: scm.NewInt(35), upperInclusive: true,
			},
			wantShards: []int{3},
		},
		{
			name: "point on pivot",
			boundary: analyzedBoundary{
				col: "id", matcher: EqualMatcher,
				lower: scm.NewInt(20), lowerInclusive: true,
				upper: scm.NewInt(20), upperInclusive: true,
			},
			wantShards: []int{1},
		},
		{
			name: "closed range",
			boundary: analyzedBoundary{
				col: "id", matcher: RangeMatcher,
				lower: scm.NewInt(15), lowerInclusive: true,
				upper: scm.NewInt(35), upperInclusive: true,
			},
			wantShards: []int{1, 2, 3},
		},
		{
			name: "exclusive lower pivot",
			boundary: analyzedBoundary{
				col: "id", matcher: RangeMatcher,
				lower: scm.NewInt(20), lowerInclusive: false,
				upper: scm.NewNil(), upperInclusive: false,
			},
			wantShards: []int{2, 3, 4},
		},
		{
			name: "exclusive upper pivot remains conservative",
			boundary: analyzedBoundary{
				col: "id", matcher: RangeMatcher,
				lower: scm.NewNil(), lowerInclusive: false,
				upper: scm.NewInt(20), upperInclusive: false,
			},
			wantShards: []int{0, 1},
		},
		{
			name: "unordered IN binding is not a scalar partition interval",
			boundary: analyzedBoundary{
				col: "id", matcher: InMatcher,
				lower: scm.NewSlice([]scm.Scmer{scm.NewInt(35), scm.NewInt(5)}), lowerInclusive: true,
				upper: scm.NewSlice([]scm.Scmer{scm.NewInt(35), scm.NewInt(5)}), upperInclusive: true,
			},
			wantShards: []int{0, 1, 2, 3, 4},
		},
		{
			name: "LIKE binding is not a scalar partition interval",
			boundary: analyzedBoundary{
				col: "id", matcher: LikeMatcher,
				lower: scm.NewString("%3%"), upper: scm.NewString("%3%"),
			},
			wantShards: []int{0, 1, 2, 3, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectRelevantShards(schema, runtimeScanAccess(analyzedBoundaries{tt.boundary}), shards)
			if len(got) != len(tt.wantShards) {
				t.Fatalf("collectRelevantShards returned %d shards, want %d", len(got), len(tt.wantShards))
			}
			for i, wantIndex := range tt.wantShards {
				if got[i] != shards[wantIndex] {
					t.Fatalf("collectRelevantShards shard %d = %p, want shard[%d] %p", i, got[i], wantIndex, shards[wantIndex])
				}
			}
			one, single := singleRelevantShard(schema, runtimeScanAccess(analyzedBoundaries{tt.boundary}), shards)
			if single != (len(tt.wantShards) == 1) || (single && one != shards[tt.wantShards[0]]) {
				t.Fatalf("singleRelevantShard returned %p single=%v for expected shards %v", one, single, tt.wantShards)
			}
		})
	}
}

func setupScanParallelTestTable(t *testing.T, dbName string) *table {
	t.Helper()

	dir, err := os.MkdirTemp("", "memcp-scan-parallel-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	oldBasepath := Basepath
	Basepath = dir
	t.Cleanup(func() {
		Basepath = oldBasepath
	})

	Init(scm.Globalenv)
	LoadDatabases()
	t.Cleanup(func() {
		databases.Remove(dbName)
	})

	CreateDatabase(dbName, false)
	tbl, _ := CreateTable(dbName, "items", Memory, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	return tbl
}

func TestUniqueBatchRejectsRepeatedKeys(t *testing.T) {
	for _, ignore := range []bool{false, true} {
		t.Run(fmt.Sprintf("ignore-%v", ignore), func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "tuniquebatchduplicates")
			tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
			tx := NewTxContext(TxCursorStability)
			onCollision := scm.NewNil()
			if ignore {
				onCollision = scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewInt(0) })
			}
			var caught any
			func() {
				defer func() { caught = recover() }()
				tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(7)}, {scm.NewInt(8)}, {scm.NewInt(7)}}, nil, onCollision, false, nil, tx)
			}()
			if ignore {
				if caught != nil {
					t.Fatalf("INSERT IGNORE panicked: %v", caught)
				}
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
				if got := tbl.CountExact(); got != 2 {
					t.Fatalf("INSERT IGNORE published %d rows, want 2", got)
				}
			} else {
				tx.Rollback()
				if caught == nil || !strings.Contains(fmt.Sprint(caught), "Duplicate entry") {
					t.Fatalf("expected duplicate error, got %v", caught)
				}
				if got := tbl.CountExact(); got != 0 {
					t.Fatalf("failed statement retained %d rows", got)
				}
			}
		})
	}
}

func TestUniqueBatchRechecksRowsRejectedByAnotherKey(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tuniquebatchlaterkey")
	tbl.CreateColumn("val", "INT", nil, nil)
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}, {Id: "val", Cols: []string{"val"}}}
	cols := []string{"id", "val"}
	tbl.Insert(cols, [][]scm.Scmer{{scm.NewInt(10), scm.NewInt(99)}}, nil, scm.NewNil(), false, nil)
	ignore := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewInt(0) })
	got := tbl.Insert(cols, [][]scm.Scmer{{scm.NewInt(1), scm.NewInt(99)}, {scm.NewInt(1), scm.NewInt(100)}}, nil, ignore, false, nil)
	if got != 1 || tbl.CountExact() != 2 {
		t.Fatalf("later key rejection incorrectly reserved first key: inserted=%d count=%d", got, tbl.CountExact())
	}
}

func TestUniqueBatchCoercingAndNullKeys(t *testing.T) {
	cases := []struct {
		name             string
		first, second    scm.Scmer
		nullsNotDistinct bool
		want             int
	}{
		{"numeric-prefix", scm.NewString("12x"), scm.NewInt(12), false, 1},
		{"int-float", scm.NewInt(12), scm.NewFloat(12), false, 1},
		{"signed-zero", scm.NewFloat(0), scm.NewFloat(math.Copysign(0, -1)), false, 1},
		{"large-distinct-ints", scm.NewInt(9007199254740992), scm.NewInt(9007199254740993), false, 2},
		{"rounded-int-float", scm.NewInt(9007199254740993), scm.NewFloat(9007199254740992), false, 1},
		{"null-distinct", scm.NewNil(), scm.NewNil(), false, 2},
		{"null-not-distinct", scm.NewNil(), scm.NewNil(), true, 1},
		{"null-false-coercion", scm.NewBool(false), scm.NewNil(), true, 1},
		{"null-empty-text", scm.NewString(""), scm.NewNil(), true, 1},
		{"null-false-text", scm.NewString("false"), scm.NewNil(), true, 1},
		{"null-uppercase-false-text", scm.NewString("FALSE"), scm.NewNil(), true, 1},
		{"null-zero-text", scm.NewString("0"), scm.NewNil(), true, 1},
		{"null-spaced-zero-text", scm.NewString(" 0 "), scm.NewNil(), true, 1},
		{"null-negative-zero-text", scm.NewString("-0.0"), scm.NewNil(), true, 1},
		{"null-zero-exponent-text", scm.NewString("0e3"), scm.NewNil(), true, 1},
		{"distinct-false-text", scm.NewString("false"), scm.NewString("0"), true, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "tuniquebatchcoercion")
			tbl.CreateColumn("value", "TEXT", nil, nil)
			tbl.Unique = []uniqueKey{{Id: "value", Cols: []string{"value"}, NullsNotDistinct: tc.nullsNotDistinct}}
			cols := []string{"id", "value"}
			rows := [][]scm.Scmer{{scm.NewInt(1), tc.first}, {scm.NewInt(2), tc.second}}
			accepted, rejected := 0, 0
			tbl.ProcessUniqueCollision(cols, rows, false, func(rows [][]scm.Scmer) {
				tbl.ActiveShards()[0].Insert(cols, rows, false, nil, nil, false, nil)
				accepted += len(rows)
			}, nil, func(string, []scm.Scmer) { rejected++ }, 0, nil)
			if accepted != tc.want || rejected != 2-tc.want || tbl.CountExact() != uint(tc.want) {
				t.Fatalf("accepted=%d rejected=%d visible=%d, want %d", accepted, rejected, tbl.CountExact(), tc.want)
			}
		})
	}
}

func TestGeneratedUniqueKeySkipReleasesOuterLockOnPanic(t *testing.T) {
	for _, generatedFirst := range []bool{true, false} {
		name := "generated_first"
		if !generatedFirst {
			name = "generated_last"
		}
		t.Run(name, func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "tuniqueskiplock")
			tbl.CreateColumn("name", "TEXT", nil, nil)
			tbl.mu.Lock()
			tbl.Columns[0].AutoIncrement = true
			keys := []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}, {Id: "name", Cols: []string{"name"}}}
			if !generatedFirst {
				keys[0], keys[1] = keys[1], keys[0]
			}
			tbl.Unique = keys
			tbl.mu.Unlock()
			var caught any
			func() {
				defer func() { caught = recover() }()
				tbl.ProcessUniqueCollision([]string{"name"}, [][]scm.Scmer{{scm.NewString("new")}}, false,
					func([][]scm.Scmer) { panic("insert callback failed") }, nil, nil, 0, nil)
			}()
			if caught != "insert callback failed" {
				t.Fatalf("expected callback panic, got %v", caught)
			}
			assertUniqueWriteLocksReleased(t, tbl)
		})
	}
}

func TestMultipleUniqueKeysReleaseOuterLockOnCallbackPanic(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tmultipleuniquepanic")
	tbl.CreateColumn("name", "TEXT", nil, nil)
	tbl.mu.Lock()
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}, {Id: "name", Cols: []string{"name"}}}
	tbl.mu.Unlock()

	var caught any
	func() {
		defer func() { caught = recover() }()
		tbl.ProcessUniqueCollision([]string{"id", "name"},
			[][]scm.Scmer{{scm.NewInt(1), scm.NewString("new")}}, false,
			func([][]scm.Scmer) { panic("insert callback failed") }, nil, nil, 0, nil)
	}()
	if caught != "insert callback failed" {
		t.Fatalf("expected callback panic, got %v", caught)
	}
	for _, shard := range tbl.ActiveShards() {
		if got := shard.cacheReaders.Load(); got != 0 {
			t.Fatalf("insert callback panic retained %d cache read leases", got)
		}
	}
	assertUniqueWriteLocksReleased(t, tbl)
}

func TestMultipleUniqueKeysReleaseOuterLockOnCollisionPanic(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tmultipleuniquecollisionpanic")
	tbl.CreateColumn("name", "TEXT", nil, nil)
	tbl.mu.Lock()
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}, {Id: "name", Cols: []string{"name"}}}
	tbl.mu.Unlock()
	tbl.Insert([]string{"id", "name"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewString("taken")}}, nil, scm.NewNil(), false, nil)

	var caught any
	func() {
		defer func() { caught = recover() }()
		tbl.ProcessUniqueCollision([]string{"id", "name"},
			[][]scm.Scmer{{scm.NewInt(2), scm.NewString("taken")}}, false,
			func([][]scm.Scmer) { t.Fatal("duplicate row reached success callback") }, nil,
			func(string, []scm.Scmer) { panic("collision callback failed") }, 0, nil)
	}()
	if caught != "collision callback failed" {
		t.Fatalf("expected collision callback panic, got %v", caught)
	}
	for _, shard := range tbl.ActiveShards() {
		if got := shard.cacheReaders.Load(); got != 0 {
			t.Fatalf("collision callback panic retained %d cache read leases", got)
		}
	}
	assertUniqueWriteLocksReleased(t, tbl)
}

func TestIterateShardsParallelMarksFreeSingleShardSolo(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscanparfree")

	calls := 0
	sawSolo := false
	done := tbl.iterateShardsParallel(nil, scanAccess{}, nil, func(s *storageShard, solo bool, acquisitionFailure any) {
		if acquisitionFailure != nil {
			panic(acquisitionFailure)
		}
		calls++
		sawSolo = solo
	})
	if done != nil {
		t.Fatal("iterateShardsParallel free single shard unexpectedly returned async done channel")
	}

	if calls != 1 {
		t.Fatalf("iterateShardsParallel free single shard calls = %d, want 1", calls)
	}
	if !sawSolo {
		t.Fatal("iterateShardsParallel free single shard did not mark callback as solo")
	}
}

func TestIterateShardsParallelReleasesFreeShardRegistrationOnPanic(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscanparpanic")
	shard := tbl.Shards[0]
	topology := tbl.activeTopology()

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("iterateShardsParallel callback did not panic")
			}
		}()
		tbl.iterateShardsParallel(nil, scanAccess{}, nil, func(_ *storageShard, _ bool, acquisitionFailure any) {
			if acquisitionFailure != nil {
				panic(acquisitionFailure)
			}
			panic("forced scan failure")
		})
	}()

	if got := shard.activeScanners.Load(); got != 0 {
		t.Fatalf("active scanner registrations after panic = %d, want 0", got)
	}
	if got := shard.cacheReaders.Load(); got != 0 {
		t.Fatalf("cache read leases after panic = %d, want 0", got)
	}
	if got := topology.operations.Load(); got != 0 {
		t.Fatalf("topology user registrations after panic = %d, want 0", got)
	}
}

func TestRepartitionPublishesWhileOldGenerationReaderRuns(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscanparrepartitionreader")
	rows := make([][]scm.Scmer, 128)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i))}
	}
	tbl.Insert([]string{"id"}, rows, nil, scm.NewNil(), false, nil)
	dimensions := []shardDimension{tbl.NewShardDimension("id", 2)}
	oldTopology := tbl.activeTopology()

	readerEntered := make(chan struct{})
	releaseReader := make(chan struct{})
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		tbl.iterateShardsParallel(nil, scanAccess{}, nil, func(_ *storageShard, _ bool, acquisitionFailure any) {
			if acquisitionFailure != nil {
				panic(acquisitionFailure)
			}
			close(readerEntered)
			<-releaseReader
		})
	}()
	<-readerEntered
	t.Cleanup(func() {
		select {
		case <-releaseReader:
		default:
			close(releaseReader)
		}
		<-readerDone
	})

	if !tbl.beginManualRepartition() {
		t.Fatal("manual repartition was not claimed")
	}
	repartitionDone := make(chan struct{})
	go func() {
		tbl.repartition(dimensions)
		close(repartitionDone)
	}()

	select {
	case <-repartitionDone:
	case <-time.After(5 * time.Second):
		t.Fatal("repartition waited for a reader pinned to the old generation")
	}
	if topology := tbl.activeTopology(); topology == oldTopology || topology.mode != ShardModePartition {
		t.Fatal("repartition did not publish the new partition generation")
	}
	select {
	case <-oldTopology.drained:
		t.Fatal("old generation drained before its reader returned")
	default:
	}

	insertDone := make(chan struct{})
	go func() {
		tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(1000)}}, nil, scm.NewNil(), false, nil)
		close(insertDone)
	}()
	select {
	case <-insertDone:
	case <-time.After(time.Second):
		t.Fatal("writer on the published generation was blocked by an old reader")
	}

	close(releaseReader)
	<-readerDone
	select {
	case <-oldTopology.drained:
	case <-time.After(time.Second):
		t.Fatal("old generation did not drain after its last reader returned")
	}
	deadline := time.Now().Add(time.Second)
	for {
		tbl.mu.Lock()
		idle := tbl.maintenanceKind == 0
		tbl.mu.Unlock()
		if idle {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("repartition retirement did not release maintenance ownership")
		}
		runtime.Gosched()
	}
	if got := tbl.Count(); got != uint(len(rows)+1) {
		t.Fatalf("row count after asynchronous generation retirement = %d, want %d", got, len(rows)+1)
	}
}

func TestOverflowRebuildPublishesShardTopologySafely(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscanpartopology")
	oldShardSize := Settings.ShardSize
	Settings.ShardSize = 8
	t.Cleanup(func() {
		Settings.ShardSize = oldShardSize
	})

	rows := make([][]scm.Scmer, 8)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i))}
	}
	tbl.Insert([]string{"id"}, rows, nil, scm.NewNil(), false, nil)
	oldShard := tbl.Shards[0]

	started := make(chan struct{})
	stop := make(chan struct{})
	var readers sync.WaitGroup
	readers.Add(1)
	go func() {
		defer readers.Done()
		close(started)
		for {
			select {
			case <-stop:
				return
			default:
				shards := tbl.ActiveShards()
				if len(shards) > 0 {
					_ = shards[0]
				}
			}
		}
	}()
	<-started

	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(8)}}, nil, scm.NewNil(), false, nil)

	deadline := time.Now().Add(3 * time.Second)
	for oldShard.loadNext() == nil {
		if time.Now().After(deadline) {
			close(stop)
			readers.Wait()
			t.Fatal("overflow rebuild did not publish a successor")
		}
		runtime.Gosched()
	}
	next := oldShard.loadNext()
	next.mu.Lock()
	next.mu.Unlock()
	for tbl.overflowRebuilds.Load() != 0 {
		if time.Now().After(deadline) {
			close(stop)
			readers.Wait()
			t.Fatal("overflow rebuild did not finish")
		}
		runtime.Gosched()
	}
	close(stop)
	readers.Wait()

	if got := tbl.Count(); got != 9 {
		t.Fatalf("row count after overflow rebuild = %d, want 9", got)
	}
}

func TestIterateShardsParallelMarksPartitionSingleShardSolo(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscanparpartsolo")
	tbl.ShardMode = ShardModePartition
	tbl.PDimensions = []shardDimension{{
		Column:        "id",
		NumPartitions: 2,
		Pivots:        []scm.Scmer{scm.NewInt(10)},
	}}
	tbl.PShards = []*storageShard{NewShard(tbl), NewShard(tbl)}
	tbl.publishTopologyLocked()

	calls := 0
	sawSolo := false
	done := tbl.iterateShardsParallel(nil, runtimeScanAccess(analyzedBoundaries{{
		col:            "id",
		matcher:        EqualMatcher,
		lower:          scm.NewInt(15),
		lowerInclusive: true,
		upper:          scm.NewInt(15),
		upperInclusive: true,
	}}), nil, func(s *storageShard, solo bool, acquisitionFailure any) {
		if acquisitionFailure != nil {
			panic(acquisitionFailure)
		}
		calls++
		sawSolo = solo
	})
	if done != nil {
		t.Fatal("iterateShardsParallel partition single shard unexpectedly returned async done channel")
	}

	if calls != 1 {
		t.Fatalf("iterateShardsParallel partition single shard calls = %d, want 1", calls)
	}
	if !sawSolo {
		t.Fatal("iterateShardsParallel partition single shard did not mark callback as solo")
	}
}

func TestIterateShardsParallelMarksPartitionMultiShardNonSolo(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscanparpartmulti")
	tbl.ShardMode = ShardModePartition
	tbl.PDimensions = []shardDimension{{
		Column:        "id",
		NumPartitions: 2,
		Pivots:        []scm.Scmer{scm.NewInt(10)},
	}}
	tbl.PShards = []*storageShard{NewShard(tbl), NewShard(tbl)}
	tbl.publishTopologyLocked()

	var calls atomic.Int32
	var sawSolo atomic.Bool
	done := tbl.iterateShardsParallel(nil, scanAccess{}, nil, func(s *storageShard, solo bool, acquisitionFailure any) {
		if acquisitionFailure != nil {
			panic(acquisitionFailure)
		}
		calls.Add(1)
		if solo {
			sawSolo.Store(true)
		}
	})
	if done == nil {
		t.Fatal("iterateShardsParallel partition multi shard did not return async done channel")
	}
	<-done

	if calls.Load() != 2 {
		t.Fatalf("iterateShardsParallel partition multi shard calls = %d, want 2", calls.Load())
	}
	if sawSolo.Load() {
		t.Fatal("iterateShardsParallel partition multi shard incorrectly marked callback as solo")
	}
}

func TestIterateShardsParallelMarksSynchronousMultiShardNonSolo(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscanparsyncmulti")
	tbl.ShardMode = ShardModePartition
	tbl.PDimensions = []shardDimension{{
		Column:        "id",
		NumPartitions: 2,
		Pivots:        []scm.Scmer{scm.NewInt(10)},
	}}
	tbl.PShards = []*storageShard{NewShard(tbl), NewShard(tbl)}
	tbl.publishTopologyLocked()
	tx := NewTxContext(TxCursorStability)
	tx.fanoutLimit.Store(1)

	calls := 0
	sawSolo := false
	done := tbl.iterateShardsParallel(tx, scanAccess{}, nil, func(_ *storageShard, solo bool, acquisitionFailure any) {
		if acquisitionFailure != nil {
			panic(acquisitionFailure)
		}
		calls++
		sawSolo = sawSolo || solo
	})
	if done != nil {
		t.Fatal("synchronous multi-shard scan unexpectedly returned a done channel")
	}
	if calls != 2 {
		t.Fatalf("synchronous multi-shard calls = %d, want 2", calls)
	}
	if sawSolo {
		t.Fatal("synchronous multi-shard callback was incorrectly marked as solo")
	}
}

func TestScanResultCollectorDoesNotInventEmptyShardResult(t *testing.T) {
	var collector scanResultCollector
	collector.finish(nil)
	if result, ok := collector.next(); ok {
		t.Fatalf("empty collector returned an invented result: %#v", result)
	}

	want := scanResult{outCount: 1}
	collector.send(true, want)
	if result, ok := collector.next(); !ok || result.outCount != want.outCount {
		t.Fatalf("solo collector result = (%#v, %v), want (%#v, true)", result, ok, want)
	}
	if result, ok := collector.next(); ok {
		t.Fatalf("solo collector returned a second result: %#v", result)
	}
}

func TestScanResultCollectorUsesPinnedDomainAfterTopologyGrowth(t *testing.T) {
	previousProcs := runtime.GOMAXPROCS(4)
	defer runtime.GOMAXPROCS(previousProcs)
	for _, workerLimit := range []int32{1, 4} {
		t.Run(fmt.Sprintf("workers_%d", workerLimit), func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "tscanresultgrowth")
			var collector scanResultCollector
			if got := len(tbl.activeTopology().shards); got != 1 {
				t.Fatalf("initial shards = %d, want 1", got)
			}
			// Reproduce INSERT publishing writer lanes after the collector exists
			// but before the scan registers its authoritative shard domain.
			tbl.mu.Lock()
			tbl.Shards = append(tbl.Shards, NewShard(tbl), NewShard(tbl))
			topology := tbl.publishTopologyLocked()
			tbl.mu.Unlock()
			tx := NewTxContext(TxCursorStability)
			tx.fanoutLimit.Store(workerLimit)
			finished := make(chan struct{})
			go func() {
				done := tbl.iterateShardsParallel(tx, scanAccess{}, collector.prepare, func(_ *storageShard, solo bool, failure any) {
					collector.send(solo, scanResult{outCount: 1, err: scanError{r: failure}})
				})
				collector.finish(done)
				close(finished)
			}()
			select {
			case <-finished:
			case <-time.After(5 * time.Second):
				t.Fatal("fanout completion blocked before result collection")
			}
			if capacity := cap(collector.parallel); capacity != 3 {
				t.Fatalf("result capacity = %d, want registered domain 3", capacity)
			}
			results := 0
			for result, ok := collector.next(); ok; result, ok = collector.next() {
				if result.err.r != nil {
					t.Fatalf("shard acquisition failed: %v", result.err.r)
				}
				results += int(result.outCount)
			}
			if results != 3 {
				t.Fatalf("collected %d shard results, want 3", results)
			}
			if got := topology.operations.Load(); got != 0 {
				t.Fatalf("completed fanout retained %d topology operations", got)
			}
			for _, shard := range topology.shards {
				if shard.activeScanners.Load() != 0 || shard.cacheReaders.Load() != 0 {
					t.Fatal("completed fanout retained shard ownership")
				}
			}
		})
	}
}

func TestIterateShardsParallelPreparationPanicReleasesDomain(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscanpreparepanic")
	tbl.mu.Lock()
	tbl.Shards = append(tbl.Shards, NewShard(tbl))
	topology := tbl.publishTopologyLocked()
	tbl.mu.Unlock()
	var failure any
	func() {
		defer func() { failure = recover() }()
		tbl.iterateShardsParallel(nil, scanAccess{}, func(shards int) {
			if shards != 2 {
				t.Fatalf("preparation domain = %d, want 2", shards)
			}
			panic("result preparation failed")
		}, func(*storageShard, bool, any) {
			t.Fatal("shard ran after result preparation failed")
		})
	}()
	if failure != "result preparation failed" {
		t.Fatalf("preparation failure = %v", failure)
	}
	if got := topology.operations.Load(); got != 0 {
		t.Fatalf("failed preparation retained %d topology operations", got)
	}
	for _, shard := range topology.shards {
		if shard.activeScanners.Load() != 0 || shard.cacheReaders.Load() != 0 {
			t.Fatal("failed result preparation retained shard ownership")
		}
	}
}

func TestIterateShardsParallelAutocommitUsesExplicitContext(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscanparexplicit")
	tbl.ShardMode = ShardModePartition
	tbl.PDimensions = []shardDimension{{
		Column:        "id",
		NumPartitions: 2,
		Pivots:        []scm.Scmer{scm.NewInt(10)},
	}}
	tbl.PShards = []*storageShard{NewShard(tbl), NewShard(tbl)}
	tbl.publishTopologyLocked()
	tx := NewTxContext(TxCursorStability)
	tx.autoCommit = true
	tx.Session = scm.NewSession()
	tx.SessionState = &scm.SessionState{ID: 91}
	tx.querySeq.Store(37)
	scm.Apply(tx.Session, scm.NewString("worker-session-test"), scm.NewBool(true))

	var calls atomic.Int32
	var missingSession atomic.Bool
	var missingSessionState atomic.Bool
	var missingQuerySeq atomic.Bool
	done := tbl.iterateShardsParallel(tx, scanAccess{}, nil, func(s *storageShard, solo bool, acquisitionFailure any) {
		if acquisitionFailure != nil {
			panic(acquisitionFailure)
		}
		calls.Add(1)
		if !scm.Apply(tx.Session, scm.NewString("worker-session-test")).Bool() {
			missingSession.Store(true)
		}
		if SessionStateFromTx(tx) != tx.SessionState {
			missingSessionState.Store(true)
		}
		if querySeqFromTx(tx) != 37 {
			missingQuerySeq.Store(true)
		}
	})
	if done == nil {
		t.Fatal("iterateShardsParallel autocommit multi-shard scan did not return done channel")
	}
	<-done

	if calls.Load() != 2 {
		t.Fatalf("iterateShardsParallel autocommit calls = %d, want 2", calls.Load())
	}
	if missingSession.Load() {
		t.Fatal("autocommit shard worker did not install the transaction session")
	}
	if missingSessionState.Load() {
		t.Fatal("autocommit shard worker did not install the transaction session state")
	}
	if missingQuerySeq.Load() {
		t.Fatal("autocommit shard worker did not install the statement query sequence")
	}
}

func TestShowStatsLockedDoesNotReenterWithQueuedWriter(t *testing.T) {
	shard := &storageShard{
		main_count:  1,
		columns:     map[string]ColumnStorage{"value": &StorageConst{value: scm.NewInt(7), count: 1}},
		writeOwners: make(map[uint64]uint32),
	}
	readerReady := make(chan struct{})
	readStats := make(chan shardStatsSnapshot, 1)
	releaseReader := make(chan struct{})
	go func() {
		shard.mu.RLock()
		defer shard.mu.RUnlock()
		close(readerReady)
		<-releaseReader
		readStats <- shard.statsSnapshotRLocked()
	}()
	<-readerReady

	writerDone := make(chan struct{})
	go func() {
		shard.mu.Lock()
		defer shard.mu.Unlock()
		close(writerDone)
	}()

	deadline := time.Now().Add(time.Second)
	for shard.mu.TryRLock() {
		shard.mu.RUnlock()
		if time.Now().After(deadline) {
			close(releaseReader)
			t.Fatal("writer did not queue for the shard lock")
		}
		runtime.Gosched()
	}
	close(releaseReader)

	select {
	case stats := <-readStats:
		if stats.rowCount() != 1 || stats.size == 0 {
			t.Fatalf("locked statistics = %v, want one row and a nonzero size", stats)
		}
	case <-time.After(time.Second):
		t.Fatal("locked SHOW statistics re-entered the shard read lock")
	}
	select {
	case <-writerDone:
	case <-time.After(time.Second):
		t.Fatal("queued writer did not acquire the released shard lock")
	}
}

func TestTableStatisticsReadsPublishedSnapshotWithoutShardLock(t *testing.T) {
	shard := &storageShard{
		main_count:  1,
		columns:     map[string]ColumnStorage{"value": &StorageConst{value: scm.NewInt(7), count: 1}},
		writeOwners: make(map[uint64]uint32),
	}
	shard.setState(WRITE)
	tbl := &table{schema: &database{Name: "table-statistics-test"}, Shards: []*storageShard{shard}}
	shard.t = tbl
	tbl.PlannerRowEstimate.value.Store(1)
	tbl.publishShowColumnsSnapshot()
	if stats := tbl.statistics(); stats.rowCount != 1 || stats.sizeBytes != 0 {
		t.Fatalf("startup statistics = %+v, want row estimate 1 before size collection", stats)
	}
	tbl.collectStatistics()

	shard.mu.Lock()
	shard.main_count = 2
	readStats := make(chan tableStatisticsSnapshot, 1)
	go func() {
		readStats <- tbl.statistics()
	}()

	select {
	case stats := <-readStats:
		if stats.rowCount != 1 || stats.sizeBytes == 0 {
			t.Fatalf("published statistics = %+v, want the previously collected one-row snapshot", stats)
		}
	case <-time.After(time.Second):
		shard.mu.Unlock()
		t.Fatal("reading published table statistics waited for the shard lock")
	}
	shard.mu.Unlock()

	tbl.ShowColumns()
	if stats := tbl.statistics(); stats.rowCount != 1 {
		t.Fatalf("SHOW COLUMNS replaced collected row count with %d, want 1", stats.rowCount)
	}

	tbl.collectStatistics()
	if stats := tbl.statistics(); stats.rowCount != 2 {
		t.Fatalf("refreshed row count = %d, want 2", stats.rowCount)
	}
}

var tableStatisticsBenchmarkSink tableStatisticsSnapshot

func TestTableShowColumnsSnapshotFitsCacheLine(t *testing.T) {
	if size := unsafe.Sizeof(tableShowColumnsSnapshot{}); size > 64 {
		t.Fatalf("table metadata snapshot size = %d bytes, want at most one cache line", size)
	}
}

func benchmarkStatisticsTable() *table {
	tbl := &table{schema: &database{Name: "table-statistics-benchmark"}}
	for i := 0; i < 8; i++ {
		shard := &storageShard{
			t:            tbl,
			main_count:   1000,
			columns:      make(map[string]ColumnStorage, 4),
			writeOwners:  make(map[uint64]uint32),
			deltaColumns: make(map[string]int),
		}
		shard.setState(WRITE)
		for col := 0; col < 4; col++ {
			shard.columns[string(rune('a'+col))] = &StorageConst{value: scm.NewInt(int64(col)), count: 1000}
		}
		tbl.Shards = append(tbl.Shards, shard)
	}
	tbl.collectStatistics()
	return tbl
}

func BenchmarkTableStatisticsOnDemandShardScan(b *testing.B) {
	tbl := benchmarkStatisticsTable()
	b.ReportAllocs()
	for b.Loop() {
		stats := tableStatisticsSnapshot{}
		for _, shard := range tbl.Shards {
			shard.mu.RLock()
			stats.rowCount += int64(shard.main_count) + int64(len(shard.inserts)) - int64(shard.deletions.Count())
			stats.sizeBytes += int64(shard.ComputeSize())
			shard.mu.RUnlock()
		}
		tableStatisticsBenchmarkSink = stats
	}
}

func BenchmarkTableStatisticsPublishedRead(b *testing.B) {
	tbl := benchmarkStatisticsTable()
	b.ReportAllocs()
	for b.Loop() {
		tableStatisticsBenchmarkSink = tbl.statistics()
	}
}

func TestUniqueChecksWithMorePartitionDimensionsThanKeyColumns(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tunique2dimensions")
	tbl.CreateColumn("name", "TEXT", nil, nil)
	tbl.mu.Lock()
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}, {Id: "name", Cols: []string{"name"}}}
	tbl.ShardMode = ShardModePartition
	tbl.PDimensions = []shardDimension{
		{Column: "id", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(10)}},
		{Column: "name", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewString("m")}},
	}
	tbl.PShards = []*storageShard{NewShard(tbl), NewShard(tbl), NewShard(tbl), NewShard(tbl)}
	tbl.publishTopologyLocked()
	tbl.mu.Unlock()
	accepted := 0
	tbl.ProcessUniqueCollision([]string{"id", "name"}, [][]scm.Scmer{{scm.NewInt(15), scm.NewString("new")}}, false,
		func(rows [][]scm.Scmer) { accepted += len(rows) }, nil,
		func(key string, _ []scm.Scmer) { t.Fatalf("unexpected collision on %s", key) }, 0, nil)
	if accepted != 1 {
		t.Fatalf("accepted %d rows, want 1", accepted)
	}
}

func TestUniqueChecksRejectCollisionsOutsideInsertPartition(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tuniquecrosspartition")
	tbl.CreateColumn("name", "TEXT", nil, nil)
	tbl.mu.Lock()
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}, {Id: "name", Cols: []string{"name"}}}
	tbl.ShardMode = ShardModePartition
	tbl.PDimensions = []shardDimension{
		{Column: "id", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(10)}},
		{Column: "name", NumPartitions: 2, Pivots: []scm.Scmer{scm.NewString("m")}},
	}
	tbl.PShards = []*storageShard{NewShard(tbl), NewShard(tbl), NewShard(tbl), NewShard(tbl)}
	tbl.publishTopologyLocked()
	tbl.mu.Unlock()
	columns := []string{"id", "name"}
	tbl.Insert(columns, [][]scm.Scmer{{scm.NewInt(5), scm.NewString("alpha")}}, nil, scm.NewNil(), false, nil)
	for _, tc := range []struct {
		key string
		row []scm.Scmer
	}{
		{"PRIMARY", []scm.Scmer{scm.NewInt(5), scm.NewString("zulu")}},
		{"name", []scm.Scmer{scm.NewInt(15), scm.NewString("alpha")}},
	} {
		t.Run(tc.key, func(t *testing.T) {
			collision := ""
			tbl.ProcessUniqueCollision(columns, [][]scm.Scmer{tc.row}, false,
				func([][]scm.Scmer) { t.Error("accepted a duplicate from another partition") }, nil,
				func(key string, _ []scm.Scmer) { collision = key }, 0, nil)
			if collision != tc.key {
				t.Fatalf("collision = %q, want %q", collision, tc.key)
			}
		})
	}
}

func TestUniqueNullPolicyConcurrentInsert(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tnullkeyconcurrent")
	tbl.CreateColumn("value", "INT", nil, []scm.Scmer{
		scm.NewString("unique"), scm.NewBool(true),
		scm.NewString("nulls_not_distinct"), scm.NewBool(true),
	})
	var successes atomic.Int64
	var workers sync.WaitGroup
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func(id int) {
			defer workers.Done()
			defer func() {
				if caught := recover(); caught != nil && !strings.Contains(fmt.Sprint(caught), "Duplicate entry") {
					t.Errorf("unexpected insert failure: %v", caught)
				}
			}()
			tbl.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(int64(id)), scm.NewNil()}}, nil, scm.NewNil(), false, nil)
			successes.Add(1)
		}(i)
	}
	workers.Wait()
	if got := successes.Load(); got != 1 {
		t.Fatalf("concurrent equal-NULL inserts succeeded %d times, want 1", got)
	}
}

func TestReserveIDsConcurrentBlocks(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "treserveidblocks")
	reserve := scm.Globalenv.Vars["reserve_ids"]
	const workers, blocks, count = 8, 32, 7
	ranges := make(chan scm.Scmer, workers*blocks)
	var writers sync.WaitGroup
	for w := 0; w < workers; w++ {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for i := 0; i < blocks; i++ {
				ranges <- scm.Apply(reserve, NewTableScmer(tbl), scm.NewInt(count))
			}
		}()
	}
	writers.Wait()
	close(ranges)
	seen := make(map[int64]bool)
	for block := range ranges {
		ends := block.Slice()
		if len(ends) != 2 || ends[1].Int()-ends[0].Int()+1 != count {
			t.Fatalf("invalid inclusive block: %v", block)
		}
		for id := ends[0].Int(); id <= ends[1].Int(); id++ {
			if seen[id] {
				t.Fatalf("overlapping reserved ID %d", id)
			}
			seen[id] = true
		}
	}
	if len(seen) != workers*blocks*count || atomic.LoadUint64(&tbl.Auto_increment) != workers*blocks*count {
		t.Fatal("concurrent blocks lost IDs")
	}
}

func TestReserveIDsRejectsInvalidAndExhaustedBlocks(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "treserveidlimits")
	reserve := scm.Globalenv.Vars["reserve_ids"]
	for _, count := range []scm.Scmer{scm.NewInt(-1), scm.NewFloat(1.5), scm.NewString("2")} {
		requirePanic(t, func() { scm.Apply(reserve, NewTableScmer(tbl), count) })
	}
	if atomic.LoadUint64(&tbl.Auto_increment) != 0 {
		t.Fatal("invalid request changed the counter")
	}
	atomic.StoreUint64(&tbl.Auto_increment, math.MaxInt64-1)
	requirePanic(t, func() { scm.Apply(reserve, NewTableScmer(tbl), scm.NewInt(2)) })
	if atomic.LoadUint64(&tbl.Auto_increment) != math.MaxInt64-1 {
		t.Fatal("overflow request changed the counter")
	}
	block := scm.Apply(reserve, NewTableScmer(tbl), scm.NewInt(1)).Slice()
	if block[0].Int() != math.MaxInt64 || block[1].Int() != math.MaxInt64 {
		t.Fatalf("last valid native block = %v", block)
	}
	requirePanic(t, func() { scm.Apply(reserve, NewTableScmer(tbl), scm.NewInt(1)) })
	if zero := scm.Apply(reserve, NewTableScmer(tbl), scm.NewInt(0)); len(zero.Slice()) != 0 {
		t.Fatalf("zero request returned %v", zero)
	}
	if atomic.LoadUint64(&tbl.Auto_increment) != math.MaxInt64 {
		t.Fatal("zero or rejected block changed the exhausted counter")
	}
	// Legacy signed explicit IDs can leave a high unsigned recovery counter.
	// Loading/advancing that state without generating values stays supported.
	if base := tbl.reserveAutoIncrement(math.MaxUint64, 0); base != math.MaxUint64 {
		t.Fatalf("legacy recovery counter changed: %d", base)
	}
	requirePanic(t, func() { tbl.reserveAutoIncrement(0, 1) })
}

func TestReserveIDsExplicitInsertAndCallback(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "treserveidinsert")
	tbl.Columns[0].AutoIncrement = true
	reserve := scm.Globalenv.Vars["reserve_ids"]
	block := scm.Apply(reserve, NewTableScmer(tbl), scm.NewInt(3)).Slice()
	if block[0].Int() != 1 || block[1].Int() != 3 {
		t.Fatalf("first block = %v", block)
	}
	var callbackID int64
	tbl.Insert([]string{"id"}, [][]scm.Scmer{{scm.NewInt(3)}, {scm.NewInt(1)}}, nil, scm.NewNil(), false, func(id int64) { callbackID = id })
	if atomic.LoadUint64(&tbl.Auto_increment) != 3 || callbackID != 0 {
		t.Fatal("explicit reserved values allocated again or called the generated-ID callback")
	}
	tbl.Insert(nil, [][]scm.Scmer{{}, {}}, nil, scm.NewNil(), false, func(id int64) { callbackID = id })
	if callbackID != 4 || atomic.LoadUint64(&tbl.Auto_increment) != 5 || tbl.CountExact() != 4 {
		t.Fatal("implicit INSERT lost its first-ID callback or reused reserved values")
	}
	atomic.StoreUint64(&tbl.Auto_increment, math.MaxInt64)
	callbackID = 0
	requirePanic(t, func() {
		tbl.Insert(nil, [][]scm.Scmer{{}}, nil, scm.NewNil(), false, func(id int64) { callbackID = id })
	})
	if callbackID != 0 || tbl.CountExact() != 4 {
		t.Fatal("exhausted implicit INSERT published a row or called its callback")
	}
}

func TestReserveIDsRestartContract(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "treserveidrestart", 0)
	tbl.Columns[0].AutoIncrement = true
	tbl.publishShowColumnsSnapshot()
	tbl.schema.save()
	reserve := scm.Globalenv.Vars["reserve_ids"]
	block := scm.Apply(reserve, NewTableScmer(tbl), scm.NewInt(3)).Slice()
	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{block[1], scm.NewString("durable reserved ID")}}, nil, scm.NewNil(), false, nil)
	reloaded := reloadTableFromPersistence(t, "treserveidrestart", persistence)
	if reloaded.CountExact() != 1 || atomic.LoadUint64(&reloaded.Auto_increment) != 3 {
		t.Fatal("persisted reserved ID did not recover through existing WAL")
	}
	unused := scm.Apply(reserve, NewTableScmer(reloaded), scm.NewInt(2)).Slice()
	if unused[0].Int() != 4 || unused[1].Int() != 5 {
		t.Fatalf("unexpected unused block %v", unused)
	}
	crashed := reloadTableFromPersistence(t, "treserveidrestart", persistence)
	if atomic.LoadUint64(&crashed.Auto_increment) != 3 {
		t.Fatal("pure reservation unexpectedly changed persistent state")
	}
	crashed.schema.save()
	scm.Apply(reserve, NewTableScmer(crashed), scm.NewInt(2))
	crashed.schema.save()
	checkpointed := reloadTableFromPersistence(t, "treserveidrestart", persistence)
	if atomic.LoadUint64(&checkpointed.Auto_increment) != 5 {
		t.Fatal("existing schema checkpoint did not retain the reserved counter")
	}
}

// Native INSERT and callers reserving their own blocks must compete for the
// same counter. Each callback reports its own batch, never a later global value.
func TestReserveIDsConcurrentWithNativeInsert(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "treserveidmixed")
	tbl.Columns[0].AutoIncrement = true
	reserve := scm.Globalenv.Vars["reserve_ids"]
	const workers, batches = 8, 20
	blocks := make(chan [2]int64, workers*batches)
	var writers sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		writers.Add(1)
		go func(worker int) {
			defer writers.Done()
			for batch := 0; batch < batches; batch++ {
				if worker%2 == 0 {
					block := scm.Apply(reserve, NewTableScmer(tbl), scm.NewInt(7)).Slice()
					blocks <- [2]int64{block[0].Int(), block[1].Int()}
				} else {
					tbl.Insert(nil, [][]scm.Scmer{{}, {}, {}}, nil, scm.NewNil(), false, func(first int64) {
						blocks <- [2]int64{first, first + 2}
					})
				}
			}
		}(worker)
	}
	writers.Wait()
	close(blocks)
	seen := make(map[int64]bool)
	for block := range blocks {
		for id := block[0]; id <= block[1]; id++ {
			if seen[id] {
				t.Fatalf("native INSERT and reservation overlap at %d", id)
			}
			seen[id] = true
		}
	}
	const expected = workers / 2 * batches * (7 + 3)
	if len(seen) != expected || atomic.LoadUint64(&tbl.Auto_increment) != expected {
		t.Fatal("mixed native batches lost or consumed IDs twice")
	}
	if tbl.CountExact() != workers/2*batches*3 {
		t.Fatal("reservation unexpectedly published rows or INSERT lost rows")
	}
}

func TestReserveIDsPersistedPrefixRecovery(t *testing.T) {
	tbl, persistence := createDurabilityTestTable(t, "treserveidprefix", 0)
	tbl.Columns[0].AutoIncrement = true
	tbl.publishShowColumnsSnapshot()
	tbl.schema.save()
	reserve := scm.Globalenv.Vars["reserve_ids"]
	block := scm.Apply(reserve, NewTableScmer(tbl), scm.NewInt(100)).Slice()
	tbl.Insert([]string{"id", "payload"}, [][]scm.Scmer{{block[0], scm.NewString("only first ID persisted")}}, nil, scm.NewNil(), false, nil)
	reloaded := reloadTableFromPersistence(t, "treserveidprefix", persistence)
	if atomic.LoadUint64(&reloaded.Auto_increment) != 1 {
		t.Fatal("a persisted prefix made its unlogged unused suffix durable")
	}
	next := scm.Apply(reserve, NewTableScmer(reloaded), scm.NewInt(1)).Slice()
	if next[0].Int() != 2 || next[1].Int() != 2 {
		t.Fatalf("unlogged suffix was not recovered from persisted IDs: %v", next)
	}
}

func assertUniqueWriteLocksReleased(t *testing.T, tbl *table) {
	t.Helper()
	locks := tbl.uniqueWrites.Load()
	if locks == nil {
		return
	}
	for i := range locks {
		if !locks[i].TryLock() {
			t.Fatalf("unique stripe %d retained after callback panic", i)
		}
		locks[i].Unlock()
	}
}

func TestUniqueOverlappingACIDInserts(t *testing.T) {
	for _, mode := range []string{"free", "pruned", "global"} {
		t.Run(mode, func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "tuniqueacidoverlap")
			tbl.CreateColumn("route", "INT", nil, nil)
			tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
			if mode != "free" {
				dimension := "id"
				if mode == "global" {
					dimension = "route"
				}
				tbl.mu.Lock()
				tbl.ShardMode = ShardModePartition
				tbl.PDimensions = []shardDimension{{Column: dimension, NumPartitions: 2, Pivots: []scm.Scmer{scm.NewInt(3)}}}
				tbl.PShards = []*storageShard{NewShard(tbl), NewShard(tbl)}
				tbl.publishTopologyLocked()
				tbl.mu.Unlock()
			}
			const writers = 8
			staged := make(chan struct{}, writers)
			commit := make(chan struct{})
			var committed atomic.Int64
			var workers sync.WaitGroup
			for writer := range writers {
				workers.Add(1)
				go func() {
					defer workers.Done()
					tx := NewTxContext(TxACID)
					var caught any
					func() {
						defer func() { caught = recover() }()
						tbl.Insert([]string{"id", "route"}, [][]scm.Scmer{{scm.NewInt(7), scm.NewInt(int64(writer))}}, nil, scm.NewNil(), false, nil, tx)
					}()
					staged <- struct{}{}
					<-commit
					if caught != nil {
						tx.Rollback()
						if !strings.Contains(fmt.Sprint(caught), "Concurrent UNIQUE reservation") && !strings.Contains(fmt.Sprint(caught), "Duplicate entry") {
							t.Errorf("unexpected insert error: %v", caught)
						}
						return
					}
					if err := tx.Commit(); err != nil {
						tx.Rollback()
						return
					}
					committed.Add(1)
				}()
			}
			for range writers {
				<-staged
			}
			close(commit)
			workers.Wait()
			if got := committed.Load(); got != 1 {
				t.Fatalf("%d transactions committed the same key, want 1", got)
			}
			if got := tbl.CountExact(); got != 1 {
				t.Fatalf("published %d rows for one unique key", got)
			}

		})
	}
}

func TestUniqueReservationLifecycle(t *testing.T) {
	for _, action := range []string{"rollback", "savepoint", "unstage", "commit", "rebuild"} {
		t.Run(action, func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "tuniquereservation")
			tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
			tx := NewTxContext(TxACID)
			defer func() {
				if tx.State == TxActive {
					tx.Rollback()
				}
			}()
			sp := tx.CreateSavepoint()
			cols := []string{"id"}
			rows := [][]scm.Scmer{{scm.NewInt(7)}}
			tbl.Insert(cols, rows, nil, scm.NewNil(), false, nil, tx)
			shard := tbl.ActiveShards()[0]
			access := exactScanAccess(newExactScanAccessSchema(cols), rows[0])
			if _, present := shard.GetRecordidForUnique(access, nil, false); present {
				t.Fatal("ordinary key lookup exposed an uncommitted row")
			}
			var callbacks int
			collision := scm.NewFunc(func(...scm.Scmer) scm.Scmer {
				callbacks++
				return scm.NewBool(true)
			})
			var caught any
			func() {
				defer func() { caught = recover() }()
				tbl.Insert(cols, rows, nil, collision, false, nil)
			}()
			if caught == nil || !strings.Contains(fmt.Sprint(caught), "Concurrent UNIQUE reservation") || callbacks != 0 {
				t.Fatalf("pending key: error=%v callbacks=%d", caught, callbacks)
			}
			if !shard.mu.TryLock() {
				t.Fatal("reservation conflict leaked shard lock")
			}
			shard.mu.Unlock()
			switch action {
			case "rollback":
				tx.Rollback()
			case "savepoint":
				tx.RollbackToSavepoint(sp)
			case "unstage":
				shard.mu.Lock()
				unstaged := tx.UnstageRow(shard, shard.main_count)
				shard.mu.Unlock()
				if !unstaged {
					t.Fatal("could not unstage inserted row")
				}
				tbl.Insert(cols, rows, nil, scm.NewNil(), false, nil, tx)
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
			case "commit":
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
			case "rebuild":
				Rebuild(true, true)
				caught = nil
				func() {
					defer func() { caught = recover() }()
					tbl.Insert(cols, rows, nil, scm.NewNil(), false, nil)
				}()
				if caught == nil || !strings.Contains(fmt.Sprint(caught), "Concurrent UNIQUE reservation") {
					t.Fatalf("rebuild lost pending key: %v", caught)
				}
				tx.Rollback()
			}
			if action == "commit" || action == "unstage" {
				caught = nil
				func() {
					defer func() { caught = recover() }()
					tbl.Insert(cols, rows, nil, scm.NewNil(), false, nil)
				}()
				if caught == nil || !strings.Contains(fmt.Sprint(caught), "Duplicate entry") {
					t.Fatalf("committed key: %v", caught)
				}
			} else {
				if got := tbl.Insert(cols, rows, nil, scm.NewNil(), false, nil); got != 1 {
					t.Fatalf("released key accepted %d rows", got)
				}
			}
			if got := tbl.CountExact(); got != 1 {
				t.Fatalf("reservation lifecycle left %d visible rows", got)
			}
		})
	}
}

func TestUniqueReservationCommitConflictReleasesStagedKey(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tuniquereservationcommitconflict")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	cols := []string{"id"}
	tbl.Insert(cols, [][]scm.Scmer{{scm.NewInt(1)}}, nil, scm.NewNil(), false, nil)
	shard := tbl.ActiveShards()[0]
	first, second := NewTxContext(TxACID), NewTxContext(TxACID)
	defer func() {
		for _, tx := range []*TxContext{first, second} {
			if tx.State == TxActive {
				tx.Rollback()
			}
		}
	}()
	if !scm.ToBool(shard.UpdateFunction(0, false, false, first)(scm.NewSlice([]scm.Scmer{scm.NewString("id"), scm.NewInt(2)}))) {
		t.Fatal("first replacement was not staged")
	}
	if !scm.ToBool(shard.UpdateFunction(0, false, false, second)(scm.NewSlice([]scm.Scmer{scm.NewString("id"), scm.NewInt(3)}))) {
		t.Fatal("second replacement was not staged")
	}
	if err := first.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := second.Commit(); err == nil || !strings.Contains(err.Error(), "already deleted") {
		t.Fatalf("expected write-write commit conflict, got %v", err)
	}
	if second.State != TxAborted {
		t.Fatalf("conflicting transaction state = %v, want aborted", second.State)
	}
	if got := tbl.Insert(cols, [][]scm.Scmer{{scm.NewInt(3)}}, nil, scm.NewNil(), false, nil); got != 1 {
		t.Fatalf("aborted replacement retained its key: inserted %d", got)
	}
	if got := tbl.CountExact(); got != 2 {
		t.Fatalf("commit conflict left %d visible rows, want 2", got)
	}
}

func TestUniqueReservationOwnKeyReplacementCycle(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tuniquereservationreplacementcycle")
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	cols := []string{"id"}
	tbl.Insert(cols, [][]scm.Scmer{{scm.NewInt(7)}}, nil, scm.NewNil(), false, nil)
	shard := tbl.ActiveShards()[0]
	tx := NewTxContext(TxACID)
	defer func() {
		if tx.State == TxActive {
			tx.Rollback()
		}
	}()
	if !scm.ToBool(shard.UpdateFunction(0, false, false, tx)(scm.NewSlice([]scm.Scmer{scm.NewString("id"), scm.NewInt(8)}))) {
		t.Fatal("first replacement was not staged")
	}
	access := exactScanAccess(newExactScanAccessSchema(cols), []scm.Scmer{scm.NewInt(8)})
	recid, present := shard.GetRecordidForUnique(access, tx, false)
	if !present {
		t.Fatal("own replacement is not visible")
	}
	if !scm.ToBool(shard.UpdateFunction(recid, false, false, tx)(scm.NewSlice([]scm.Scmer{scm.NewString("id"), scm.NewInt(7)}))) {
		t.Fatal("replacement did not reuse its own hidden old key")
	}
	if _, present := shard.GetRecordidForUnique(access, tx, false); present {
		t.Fatal("superseded own replacement remained visible")
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if got := tbl.Insert(cols, [][]scm.Scmer{{scm.NewInt(8)}}, nil, scm.NewNil(), false, nil); got != 1 {
		t.Fatalf("superseded replacement retained its key: inserted %d", got)
	}
	if got := tbl.CountExact(); got != 2 {
		t.Fatalf("replacement cycle left %d visible rows, want 2", got)
	}
}
