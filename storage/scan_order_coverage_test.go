/*
Copyright (C) 2026  Carl-Philip Haensch

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
	"testing"

	"github.com/launix-de/memcp/scm"
)

func TestExtendBoundariesRejectsPartiallyCoveredOrder(t *testing.T) {
	lookupSort := buildProc([]string{"foreign_id", "$tx"}, scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("nested_lookup"),
		scm.NewSymbol("foreign_id"),
		scm.NewSymbol("$tx"),
	}))
	original := boundaries{{
		col: "tenant_id", matcher: EqualMatcher,
		lower: scm.NewInt(7), lowerInclusive: true,
		upper: scm.NewInt(7), upperInclusive: true,
	}}

	got, covered := extendBoundariesWithSortCols(original,
		[]scm.Scmer{lookupSort, scm.NewString("id")}, nil)

	if covered {
		t.Fatal("ORDER BY must not enable early LIMIT when its first key is not index-covered")
	}
	if len(got) != len(original) {
		t.Fatalf("partial ORDER BY coverage changed boundaries: got %d, want %d", len(got), len(original))
	}
}

func TestCoveredOrderedLimitBrakesInsideIndexBatch(t *testing.T) {
	const rows = 4096
	const limit = 20
	tbl := setupScanParallelTestTable(t, "tcoveredorderedlimit")
	tbl.CreateColumn("bucket", "VARCHAR(16)", nil, nil)
	tbl.CreateColumn("rank", "INT", nil, nil)
	values := make([][]scm.Scmer, rows)
	for i := range values {
		values[i] = []scm.Scmer{
			scm.NewInt(int64(i)),
			scm.NewString("target"),
			scm.NewInt(int64(i)),
		}
	}
	tbl.Insert([]string{"id", "bucket", "rank"}, values, nil, scm.NewNil(), false, nil)
	RebuildTable(tbl, true, false)

	equalSymbol := scm.Symbol("equal?")
	equalFn := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewBool(scm.Equal(values[0], values[1]))
	})
	condition := buildProc([]string{"bucket"}, scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal?"),
		scm.NewSymbol("bucket"),
		scm.NewString("target"),
	}))
	condition.Proc().En.Vars[equalSymbol] = equalFn
	_, order := integerOrder(false)
	bounds := extractBoundaries([]string{"bucket"}, condition)
	var covered bool
	bounds, covered = extendBoundariesWithSortCols(bounds,
		[]scm.Scmer{scm.NewString("rank")}, []func(...scm.Scmer) scm.Scmer{order})
	if !covered {
		t.Fatal("bucket/rank index does not cover ORDER BY")
	}
	lower, upper := indexFromBoundaries(bounds)
	shard := tbl.ActiveShards()[0]
	shard.mu.RLock()
	var buildBuf [8]uint32
	shard.iterateIndexForce(nil, scanAccess{suffix: bounds}, lower, upper, len(shard.inserts), buildBuf[:], false,
		func([]uint32) bool { return false })
	shard.mu.RUnlock()
	run := func(acceptCols []string, accept scm.Scmer) *shardqueue {
		return shard.scan_order(scanAccess{suffix: bounds, filterCovered: true}, lower, upper,
			[]string{"bucket"}, condition, acceptCols, accept,
			[]scm.Scmer{scm.NewString("rank")}, []func(...scm.Scmer) scm.Scmer{order},
			0, 0, limit, nil, nil, nil)
	}
	queue := run(nil, scm.NewNil())
	if queue.candidateCount != limit {
		t.Fatalf("covered ordered LIMIT examined %d candidates, want %d",
			queue.candidateCount, limit)
	}

	// An additional acceptance predicate can still reject index matches, so it
	// must retain the ordinary scan batch instead of claiming full coverage.
	acceptAll := buildProc([]string{"bucket"}, scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("equal?"),
		scm.NewSymbol("bucket"),
		scm.NewString("target"),
	}))
	acceptAll.Proc().En.Vars[equalSymbol] = equalFn
	queue = run([]string{"bucket"}, acceptAll)
	if queue.candidateCount <= limit {
		t.Fatalf("acceptance predicate incorrectly used covered LIMIT batch: %d candidates",
			queue.candidateCount)
	}

	// Visibility is checked after index iteration. A short covered batch may be
	// refilled, but must still return LIMIT live rows after deleted entries.
	shard.mu.Lock()
	for recid := uint(0); recid < 5; recid++ {
		shard.deletions.Set(recid, true)
	}
	shard.mu.Unlock()
	queue = run(nil, scm.NewNil())
	if len(queue.items) != limit {
		t.Fatalf("covered ordered LIMIT returned %d live rows after deletions, want %d",
			len(queue.items), limit)
	}
	if queue.candidateCount <= limit {
		t.Fatalf("covered ordered LIMIT did not refill deleted candidates: examined %d",
			queue.candidateCount)
	}
}
