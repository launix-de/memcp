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
	"os"
	"runtime"
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
		boundary   columnboundaries
		wantShards []int
	}{
		{
			name: "point between late pivots",
			boundary: columnboundaries{
				col: "id", matcher: EqualMatcher,
				lower: scm.NewInt(35), lowerInclusive: true,
				upper: scm.NewInt(35), upperInclusive: true,
			},
			wantShards: []int{3},
		},
		{
			name: "point on pivot",
			boundary: columnboundaries{
				col: "id", matcher: EqualMatcher,
				lower: scm.NewInt(20), lowerInclusive: true,
				upper: scm.NewInt(20), upperInclusive: true,
			},
			wantShards: []int{1},
		},
		{
			name: "closed range",
			boundary: columnboundaries{
				col: "id", matcher: RangeMatcher,
				lower: scm.NewInt(15), lowerInclusive: true,
				upper: scm.NewInt(35), upperInclusive: true,
			},
			wantShards: []int{1, 2, 3},
		},
		{
			name: "exclusive lower pivot",
			boundary: columnboundaries{
				col: "id", matcher: RangeMatcher,
				lower: scm.NewInt(20), lowerInclusive: false,
				upper: scm.NewNil(), upperInclusive: false,
			},
			wantShards: []int{2, 3, 4},
		},
		{
			name: "exclusive upper pivot remains conservative",
			boundary: columnboundaries{
				col: "id", matcher: RangeMatcher,
				lower: scm.NewNil(), lowerInclusive: false,
				upper: scm.NewInt(20), upperInclusive: false,
			},
			wantShards: []int{0, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectRelevantShards(schema, []columnboundaries{tt.boundary}, shards)
			if len(got) != len(tt.wantShards) {
				t.Fatalf("collectRelevantShards returned %d shards, want %d", len(got), len(tt.wantShards))
			}
			for i, wantIndex := range tt.wantShards {
				if got[i] != shards[wantIndex] {
					t.Fatalf("collectRelevantShards shard %d = %p, want shard[%d] %p", i, got[i], wantIndex, shards[wantIndex])
				}
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

func TestIterateShardsParallelMarksFreeSingleShardSolo(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscanparfree")

	calls := 0
	sawSolo := false
	done := tbl.iterateShardsParallel(nil, nil, func(s *storageShard, solo bool) {
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

func TestIterateShardsParallelMarksPartitionSingleShardSolo(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscanparpartsolo")
	tbl.ShardMode = ShardModePartition
	tbl.PDimensions = []shardDimension{{
		Column:        "id",
		NumPartitions: 2,
		Pivots:        []scm.Scmer{scm.NewInt(10)},
	}}
	tbl.PShards = []*storageShard{NewShard(tbl), NewShard(tbl)}

	calls := 0
	sawSolo := false
	done := tbl.iterateShardsParallel(nil, []columnboundaries{{
		col:            "id",
		matcher:        EqualMatcher,
		lower:          scm.NewInt(15),
		lowerInclusive: true,
		upper:          scm.NewInt(15),
		upperInclusive: true,
	}}, func(s *storageShard, solo bool) {
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

	var calls atomic.Int32
	var sawSolo atomic.Bool
	done := tbl.iterateShardsParallel(nil, nil, func(s *storageShard, solo bool) {
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

func TestIterateShardsParallelAutocommitUsesExplicitContext(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscanparexplicit")
	tbl.ShardMode = ShardModePartition
	tbl.PDimensions = []shardDimension{{
		Column:        "id",
		NumPartitions: 2,
		Pivots:        []scm.Scmer{scm.NewInt(10)},
	}}
	tbl.PShards = []*storageShard{NewShard(tbl), NewShard(tbl)}
	tx := NewTxContext(TxCursorStability)
	tx.autoCommit = true
	tx.Session = scm.NewSession()
	scm.Apply(tx.Session, scm.NewString("worker-session-test"), scm.NewBool(true))

	var calls atomic.Int32
	var inheritedContext atomic.Bool
	var missingSession atomic.Bool
	scm.SetValues(map[string]any{"scan-worker-test": true}, func() {
		done := tbl.iterateShardsParallel(tx, nil, func(s *storageShard, solo bool) {
			calls.Add(1)
			if _, ok := scm.GetGLSValue("scan-worker-test"); ok {
				inheritedContext.Store(true)
			}
			session := scm.Context(scm.NewString("session"))
			if !scm.Apply(session, scm.NewString("worker-session-test")).Bool() {
				missingSession.Store(true)
			}
		})
		if done == nil {
			t.Fatal("iterateShardsParallel autocommit multi-shard scan did not return done channel")
		}
		<-done
	})

	if calls.Load() != 2 {
		t.Fatalf("iterateShardsParallel autocommit calls = %d, want 2", calls.Load())
	}
	if inheritedContext.Load() {
		t.Fatal("autocommit shard worker inherited GLS despite explicit transaction context")
	}
	if missingSession.Load() {
		t.Fatal("autocommit shard worker did not install the transaction session")
	}
}

func TestShardWriteOwnershipUsesExplicitTransactionState(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscanpartxowner")
	shard := tbl.Shards[0]
	tx := NewTxContext(TxCursorStability)

	if shard.hasWriteOwnerForTx(tx) {
		t.Fatal("fresh transaction unexpectedly owns shard write lock")
	}
	tx.EnterShardWrite(shard)
	if !shard.hasWriteOwnerForTx(tx) {
		t.Fatal("transaction write ownership was not detected")
	}
	tx.ExitShardWrite(shard)
	if shard.hasWriteOwnerForTx(tx) {
		t.Fatal("released transaction write ownership remained visible")
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
		srState:     WRITE,
	}
	tbl := &table{schema: &database{Name: "table-statistics-test"}, Shards: []*storageShard{shard}}
	shard.t = tbl
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
			srState:      WRITE,
			deltaColumns: make(map[string]int),
		}
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
