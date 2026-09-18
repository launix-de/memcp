// Copyright (C) 2026 MemCP Contributors
// SPDX-License-Identifier: GPL-3.0-or-later
package storage

import (
	"github.com/launix-de/memcp/scm"
	"testing"
)

func setupDeltaPrefixSeek(tb testing.TB) *storageShard {
	tb.Helper()
	name := "test_delta_prefix_seek"
	databases.Remove(name)
	tb.Cleanup(func() { databases.Remove(name) })
	CreateDatabase(name, true)
	tbl, _ := CreateTable(name, "history", Memory, true)
	tbl.CreateColumn("parent", "INT", nil, nil)
	tbl.CreateColumn("version", "INT", nil, nil)
	rows := make([][]scm.Scmer, 0, 16384)
	for parent := 0; parent < 4096; parent++ {
		for version := 0; version < 4; version++ {
			rows = append(rows, []scm.Scmer{scm.NewInt(int64(parent)), scm.NewInt(int64(version))})
		}
	}
	tbl.Insert([]string{"parent", "version"}, rows, nil, scm.NewNil(), false, nil)
	return tbl.Shards[0]
}

func deltaPrefixSeek(shard *storageShard, parent int64, inclusive bool) []uint32 {
	release := shard.GetRead()
	defer release()
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	bounds := runtimeScanAccess(analyzedBoundaries{
		{col: "parent", matcher: EqualMatcher, lower: scm.NewInt(parent), upper: scm.NewInt(parent), lowerInclusive: true, upperInclusive: true},
		{col: "version", matcher: RangeMatcher, upper: scm.NewInt(2), upperInclusive: inclusive},
	})
	var buf [8]uint32
	var rows []uint32
	shard.iterateIndexForce(nil, bounds, len(shard.inserts), buf[:], false, func(batch []uint32) bool { rows = append(rows, batch...); return true })
	return rows
}

func TestDeltaIndexPrefixWithUnboundedSuffix(t *testing.T) {
	shard := setupDeltaPrefixSeek(t)
	for _, parent := range []int64{0, 2048, 4095, 5000} {
		for _, inclusive := range []bool{false, true} {
			rows := deltaPrefixSeek(shard, parent, inclusive)
			want := 2
			if inclusive {
				want = 3
			}
			if parent == 5000 {
				want = 0
			}
			if len(rows) != want {
				t.Fatalf("parent %d inclusive %v got %v", parent, inclusive, rows)
			}
			for i, row := range rows {
				if row != uint32(parent*4+int64(i)) {
					t.Fatalf("unexpected row %d at %d", row, i)
				}
			}
		}
	}
}

func BenchmarkDeltaIndexPrefixWithUnboundedSuffix(b *testing.B) {
	shard := setupDeltaPrefixSeek(b)
	deltaPrefixSeek(shard, 4095, true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := deltaPrefixSeek(shard, 4095, true); len(got) != 3 {
			b.Fatalf("wrong result %v", got)
		}
	}
}
