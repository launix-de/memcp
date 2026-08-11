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

// BenchmarkScanFixedCosts measures per-scan overhead on an empty Memory table.
//
// Run with:
//
//	go test ./storage/ -bench BenchmarkScanFixed -benchtime=5s -count=3
//
// Three variants let you isolate individual overhead layers:
//
//	_NoSession   – plain scan, no GLS context (goroutine overhead only)
//	_WithGLS     – scan inside gls.SetValues (GLS marks on stack, shallow context)
//	_WithAutocommit – full HTTP handler path: GLS + autocommit transaction
import (
	"context"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func benchScanTable(b *testing.B, name string) *table {
	b.Helper()
	CreateDatabase("bench_scan_fc_"+name, true)
	tbl, _ := CreateTable("bench_scan_fc_"+name, "empty", Memory, true)
	tbl.CreateColumn("id", "INT", nil, nil)
	return tbl
}

// BenchmarkScanFixedCosts_NoSession: scan without any GLS session.
// Establishes the pure goroutine + channel overhead baseline.
func BenchmarkScanFixedCosts_NoSession(b *testing.B) {
	tbl := benchScanTable(b, "nosession")
	trueFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	nilFn := scm.NewNil()
	neutral := scm.NewNil()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tbl.scan(
			nil,
			[]string{"id"}, trueFn,
			[]string{"id"}, trueFn,
			nilFn, neutral, nilFn, false,
		)
	}
}

// BenchmarkScanFixedCosts_WithGLS: scan inside gls.SetValues.
// Measures GLS overhead added on top of the goroutine baseline.
func BenchmarkScanFixedCosts_WithGLS(b *testing.B) {
	tbl := benchScanTable(b, "withgls")
	trueFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	nilFn := scm.NewNil()
	neutral := scm.NewNil()
	session := scm.NewSession()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scm.SetValues(map[string]any{"session": session}, func() {
			tbl.scan(
				nil,
				[]string{"id"}, trueFn,
				[]string{"id"}, trueFn,
				nilFn, neutral, nilFn, false,
			)
		})
	}
}

// BenchmarkScanFixedCosts_WithAutocommit: full HTTP handler path.
// GLS context (SetValues) + autocommit transaction wrapping + scan.
func BenchmarkScanFixedCosts_WithAutocommit(b *testing.B) {
	tbl := benchScanTable(b, "autocommit")
	trueFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	nilFn := scm.NewNil()
	neutral := scm.NewNil()
	session := scm.NewSession()
	sessionFn := session.Func()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scm.SetValues(map[string]any{"session": session}, func() {
			WithAutocommit(sessionFn, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
				return tbl.scan(
					CurrentTx(),
					[]string{"id"}, trueFn,
					[]string{"id"}, trueFn,
					nilFn, neutral, nilFn, false,
				)
			}))
		})
	}
}

// BenchmarkScanFixedCosts_DeepStack: scan called from a deep-ish call stack.
// Simulates GLS stack-scan cost when marks are far below current frame
// (as happens when scan is called from deep Scheme evaluation).
func BenchmarkScanFixedCosts_DeepStack(b *testing.B) {
	tbl := benchScanTable(b, "deepstack")
	trueFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	nilFn := scm.NewNil()
	neutral := scm.NewNil()
	session := scm.NewSession()

	// helper that recurses to a given depth then calls fn
	var recurse func(depth int, fn func())
	recurse = func(depth int, fn func()) {
		if depth == 0 {
			fn()
			return
		}
		recurse(depth-1, fn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scm.SetValues(map[string]any{"session": session}, func() {
			// simulate ~80 extra frames of Scheme evaluation above the scan call
			recurse(80, func() {
				tbl.scan(
					nil,
					[]string{"id"}, trueFn,
					[]string{"id"}, trueFn,
					nilFn, neutral, nilFn, false,
				)
			})
		})
	}
}

func benchmarkUniquePointScan(b *testing.B, name string, currentTx *TxContext) {
	dbName := "bench_scan_point_" + name
	databases.Remove(dbName)
	CreateDatabase(dbName, true)
	tbl, _ := CreateTable(dbName, "items", Memory, true)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("label", "VARCHAR", nil, nil)
	rows := make([][]scm.Scmer, 1024)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewString("value")}
	}
	tbl.Insert([]string{"id", "label"}, rows, nil, scm.NewNil(), false, nil)
	tbl.Unique = append(tbl.Unique, uniqueKey{Id: "PRIMARY", Cols: []string{"id"}})

	condition := scanCondition("id", scm.NewInt(511))
	mapFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] })
	nilFn := scm.NewNil()
	// Warm the lazily built index before measuring regular probe overhead.
	tbl.scan(currentTx, []string{"id"}, condition, []string{"label"}, mapFn, nilFn, scm.NewNil(), nilFn, false)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tbl.scan(currentTx, []string{"id"}, condition, []string{"label"}, mapFn, nilFn, scm.NewNil(), nilFn, false)
	}
}

// BenchmarkScanUniquePoint measures the repeated dimension-table lookup used
// by nested joins after logical decorrelation and join reordering.
func BenchmarkScanUniquePoint(b *testing.B) {
	benchmarkUniquePointScan(b, "read", nil)
}

// BenchmarkScanUniquePointWithTx measures the normal SQL nested-probe path,
// where the transaction already tracks shard write ownership explicitly.
func BenchmarkScanUniquePointWithTx(b *testing.B) {
	benchmarkUniquePointScan(b, "read_tx", NewTxContext(TxCursorStability))
}

func TestOpenMapReducerAllocatesMutationMetadataLazily(t *testing.T) {
	const dbName = "test_scan_mapper_metadata"
	databases.Remove(dbName)
	t.Cleanup(func() { databases.Remove(dbName) })
	CreateDatabase(dbName, true)
	tbl, _ := CreateTable(dbName, "items", Memory, true)
	tbl.CreateColumn("id", "INT", nil, nil)
	shard := tbl.Shards[0]
	mapFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewNil() })

	readMapper := shard.OpenMapReducer([]string{"id"}, mapFn, scm.NewNil(), false, 0, nil, nil)
	defer readMapper.Close()
	if readMapper.isUpdate != nil || readMapper.isIncrement != nil || readMapper.setClosureFn != nil {
		t.Fatal("read mapper allocated mutation-only metadata")
	}

	mutationMapper := shard.OpenMapReducer([]string{"$update"}, mapFn, scm.NewNil(), false, 0, nil, nil)
	defer mutationMapper.Close()
	if len(mutationMapper.isUpdate) != 1 || !mutationMapper.isUpdate[0] || len(mutationMapper.setClosureFn) != 1 {
		t.Fatal("mutation mapper did not initialize mutation metadata")
	}
}

// BenchmarkGLSGetValue measures raw cost of a single GLS GetValue call.
// Establishes per-call overhead for CurrentTx() GLS lookups inside shard goroutines.
func BenchmarkGLSGetValue(b *testing.B) {
	session := scm.NewSession()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scm.SetValues(map[string]any{"session": session}, func() {
			for j := 0; j < 10; j++ {
				scm.GetCurrentTx() // simulates per-shard CurrentTx() cost
			}
		})
	}
}

// BenchmarkNewContext measures the overhead of scm.NewContext (used by HTTP handler).
func BenchmarkNewContext(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scm.NewContext(context.TODO(), func() {})
	}
}

// BenchmarkScanUpdate measures per-row allocation cost when scanning with a
// $increment: pseudo-column. This is the primary target of the tagClosure
// optimization: before tagClosure, each row allocates a new scm.NewFunc closure;
// after tagClosure, NewClosure is used (zero per-row allocation).
//
// Run with:
//
//	go test ./storage/ -bench BenchmarkScanUpdate -benchtime=5s -count=3 -benchmem
func BenchmarkScanUpdate(b *testing.B) {
	CreateDatabase("bench_scan_update", true)
	tbl, _ := CreateTable("bench_scan_update", "su", Memory, true)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("val", "INT", nil, nil)
	// Create computed column schema first, then populate with ComputeColumn
	computor := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return args[0] })
	tbl.CreateColumn("cached_val", "INT", nil, nil)

	// Insert 10k rows
	const N = 10_000
	rows := make([][]scm.Scmer, N)
	for i := 0; i < N; i++ {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewInt(int64(i * 2))}
	}
	tbl.Insert([]string{"id", "val"}, rows, nil, scm.NewNil(), false, nil)

	// Attach the computor to the column after data is loaded
	tbl.ComputeColumn("cached_val", []string{"val"}, computor, nil, scm.NewNil())

	trueFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	nilFn := scm.NewNil()
	neutral := scm.NewNil()
	session := scm.NewSession()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		scm.SetValues(map[string]any{"session": session}, func() {
			tbl.scan(
				nil,
				[]string{"id"}, trueFn,
				[]string{"id", "$increment:cached_val"}, trueFn,
				nilFn, neutral, nilFn, false,
			)
		})
	}
}
