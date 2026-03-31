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
	"testing"

	"github.com/launix-de/memcp/scm"
)

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
	tbl.iterateShardsParallel(nil, func(s *storageShard, solo bool) {
		calls++
		sawSolo = solo
	})

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
	tbl.iterateShardsParallel([]columnboundaries{{
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

	calls := 0
	sawSolo := false
	tbl.iterateShardsParallel(nil, func(s *storageShard, solo bool) {
		calls++
		sawSolo = sawSolo || solo
	})

	if calls != 2 {
		t.Fatalf("iterateShardsParallel partition multi shard calls = %d, want 2", calls)
	}
	if sawSolo {
		t.Fatal("iterateShardsParallel partition multi shard incorrectly marked callback as solo")
	}
}
