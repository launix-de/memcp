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
	"testing"

	"github.com/launix-de/memcp/scm"
)

const (
	batchAcceptBenchmarkRows     = 1_000_000
	batchAcceptBenchmarkACLRows  = 65_536
	batchAcceptBenchmarkPageSize = 72
)

type batchAcceptBenchmarkFixture struct {
	documents   *table
	files       *table
	sortCols    []scm.Scmer
	sortDirs    []func(...scm.Scmer) scm.Scmer
	mapReduceFn scm.Scmer
}

func insertBatchAcceptBenchmarkRows(tbl *table, columns []string, count int, makeRow func(int) []scm.Scmer) {
	const chunkSize = 16_384
	for start := 0; start < count; start += chunkSize {
		end := start + chunkSize
		if end > count {
			end = count
		}
		rows := make([][]scm.Scmer, end-start)
		for i := start; i < end; i++ {
			rows[i-start] = makeRow(i)
		}
		tbl.Insert(columns, rows, nil, scm.NewNil(), false, nil)
	}
}

func newBatchAcceptBenchmarkFixture(b *testing.B) *batchAcceptBenchmarkFixture {
	b.Helper()
	oldShardSize := Settings.ShardSize
	Settings.ShardSize = 2 * batchAcceptBenchmarkRows
	defer func() { Settings.ShardSize = oldShardSize }()
	database := fmt.Sprintf("bench_batch_accept_%p", b)
	databases.Remove(database)
	b.Cleanup(func() { databases.Remove(database) })
	CreateDatabase(database, true)

	documents, _ := CreateTable(database, "documents", Memory, true)
	documents.CreateColumn("id", "INT", nil, nil)
	documents.CreateColumn("bucket", "INT", nil, nil)
	documents.CreateColumn("file_id", "INT", nil, nil)
	insertBatchAcceptBenchmarkRows(documents, []string{"id", "bucket", "file_id"}, batchAcceptBenchmarkRows, func(i int) []scm.Scmer {
		return []scm.Scmer{
			scm.NewInt(int64(i)),
			scm.NewInt(int64(i % 100)),
			scm.NewInt(int64(i % batchAcceptBenchmarkACLRows)),
		}
	})
	RebuildTable(documents, true, false)

	files, _ := CreateTable(database, "files", Memory, true)
	files.CreateColumn("id", "INT", nil, nil)
	files.CreateColumn("bucket", "INT", nil, nil)
	insertBatchAcceptBenchmarkRows(files, []string{"id", "bucket"}, batchAcceptBenchmarkACLRows, func(i int) []scm.Scmer {
		return []scm.Scmer{scm.NewInt(int64(i)), scm.NewInt(int64(i % 100))}
	})
	RebuildTable(files, true, false)

	_, ascending := integerOrder(false)
	fixture := &batchAcceptBenchmarkFixture{
		documents:   documents,
		files:       files,
		sortCols:    []scm.Scmer{scm.NewString("id")},
		sortDirs:    []func(...scm.Scmer) scm.Scmer{ascending},
		mapReduceFn: scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[1] }),
	}

	// Warm the adaptive unique id order index outside the timed sections.
	// Four small ordered windows exceed the index savings threshold without
	// retaining a million-record benchmark result.
	identity := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })
	for i := 0; i < 4; i++ {
		scanOrderBatchAccept(nil, scanOrderTableSpec{table: documents}, identity,
			fixture.sortCols, fixture.sortDirs, 0, 0, batchAcceptBenchmarkPageSize,
			[]string{"id"}, fixture.mapReduceFn, scm.NewNil(), false, scm.NewNil())
	}
	return fixture
}

func (f *batchAcceptBenchmarkFixture) scanOrder(postOrderCols []string, postOrderFilter scm.Scmer) scm.Scmer {
	return f.documents.scan_order(nil,
		nil, scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
		f.sortCols, f.sortDirs, 0, 0, batchAcceptBenchmarkPageSize,
		[]string{"id"}, f.mapReduceFn, scm.NewNil(), false, scm.NewNil(),
		postOrderCols, postOrderFilter, scm.NewNil(), nil)
}

func (f *batchAcceptBenchmarkFixture) batchAccept(batchFilter scm.Scmer) scm.Scmer {
	return scanOrderBatchAccept(nil, scanOrderTableSpec{table: f.documents}, batchFilter,
		f.sortCols, f.sortDirs, 0, 0, batchAcceptBenchmarkPageSize,
		[]string{"id"}, f.mapReduceFn, scm.NewNil(), false, scm.NewNil())
}

func benchmarkBatchAcceptPair(b *testing.B, scan func() scm.Scmer, batch func() scm.Scmer) {
	b.Helper()
	want := scan()
	if got := batch(); scm.String(got) != scm.String(want) {
		b.Fatalf("batch result %v differs from scan_order result %v", got, want)
	}
	for _, implementation := range []struct {
		name string
		run  func() scm.Scmer
	}{
		{name: "scan_order", run: scan},
		{name: "batch_accept", run: batch},
	} {
		b.Run(implementation.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkBatchAcceptResult = implementation.run()
			}
		})
	}
}

var benchmarkBatchAcceptResult scm.Scmer

// BenchmarkScanOrderBatchAcceptMillion compares the existing row-at-a-time
// post-order filter with adaptive RecSet batches over one million driver rows.
// ACL cases deliberately model the expensive shape this operator targets:
// document -> file projection, ACL membership, then file -> document projection.
func BenchmarkScanOrderBatchAcceptMillion(b *testing.B) {
	fixture := newBatchAcceptBenchmarkFixture(b)
	for _, percent := range []int{50, 1} {
		threshold := int64(percent)
		rowFilter := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			return scm.NewBool(values[0].Int() < threshold)
		})
		batchFilter := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			batch := RecSetFromScmer(values[0])
			return NewRecSetScmer(batch.filterToRecSet(nil, []string{"bucket"}, rowFilter))
		})
		b.Run(fmt.Sprintf("simple_%02dpct", percent), func(b *testing.B) {
			benchmarkBatchAcceptPair(b,
				func() scm.Scmer { return fixture.scanOrder([]string{"bucket"}, rowFilter) },
				func() scm.Scmer { return fixture.batchAccept(batchFilter) })
		})
	}

	for _, percent := range []int{50, 1} {
		threshold := int64(percent)
		allowedCondition := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			return scm.NewBool(values[0].Int() < threshold)
		})
		allowedFiles := fixture.files.scanRecSet(nil, []string{"bucket"}, allowedCondition, scm.NewNil(), nil)
		rowACLFilter := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			fileID := values[0].Int()
			return scm.NewBool(allowedFiles.scanExists(nil, []string{"id"}, scm.NewFunc(func(candidate ...scm.Scmer) scm.Scmer {
				return scm.NewBool(candidate[0].Int() == fileID)
			}), scm.NewNil(), nil))
		})
		batchACLFilter := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			batch := RecSetFromScmer(values[0])
			candidateFiles := batch.projectJoin(nil, []string{"file_id"}, fixture.files, []string{"id"})
			acceptedFiles := recSetIntersect(nil, []*recSet{candidateFiles, allowedFiles})
			projectedDocuments := acceptedFiles.projectJoin(nil, []string{"id"}, fixture.documents, []string{"file_id"})
			return NewRecSetScmer(recSetIntersect(nil, []*recSet{batch, projectedDocuments}))
		})
		b.Run(fmt.Sprintf("acl_projection_%02dpct", percent), func(b *testing.B) {
			benchmarkBatchAcceptPair(b,
				func() scm.Scmer { return fixture.scanOrder([]string{"file_id"}, rowACLFilter) },
				func() scm.Scmer { return fixture.batchAccept(batchACLFilter) })
		})
	}
}
