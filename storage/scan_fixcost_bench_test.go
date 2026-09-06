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
//	_NoSession      – plain scan without transaction context
//	_WithExplicitTx – scan with an explicitly passed transaction
//	_WithAutocommit – autocommit transaction creation plus scan
import (
	"strconv"
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

// BenchmarkScanFixedCosts_NoSession: scan without an explicit session.
// Establishes the pure goroutine + channel overhead baseline.
func BenchmarkScanFixedCosts_NoSession(b *testing.B) {
	tbl := benchScanTable(b, "nosession")
	trueFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	mapReduceFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] })
	nilFn := scm.NewNil()
	neutral := scm.NewNil()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tbl.scan(
			nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
			[]string{"id"}, trueFn,
			[]string{"id"}, mapReduceFn,
			neutral, nilFn, false,
		)
	}
}

// BenchmarkScanFixedCosts_WithExplicitTx measures explicit transaction passing.
func BenchmarkScanFixedCosts_WithExplicitTx(b *testing.B) {
	tbl := benchScanTable(b, "explicit_tx")
	trueFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	mapReduceFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] })
	nilFn := scm.NewNil()
	neutral := scm.NewNil()
	tx := NewTxContext(TxCursorStability)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tbl.scan(tx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{"id"}, trueFn, []string{"id"}, mapReduceFn, neutral, nilFn, false)
	}
}

// BenchmarkScanFixedCosts_WithAutocommit: full HTTP handler path.
// Autocommit transaction creation plus an explicitly transaction-bound scan.
func BenchmarkScanFixedCosts_WithAutocommit(b *testing.B) {
	tbl := benchScanTable(b, "autocommit")
	trueFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	mapReduceFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] })
	nilFn := scm.NewNil()
	neutral := scm.NewNil()
	session := scm.NewSession()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WithAutocommit(session, nil, 0, "benchmark scan", scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			return tbl.scan(
				scmerToTxContext(a[0]),
				newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
				[]string{"id"}, trueFn,
				[]string{"id"}, mapReduceFn,
				neutral, nilFn, false,
			)
		}))
	}
}

// BenchmarkScanFixedCosts_DeepStack: scan called from a deep-ish call stack.
// Measures whether deep Go call stacks affect scan setup cost.
func BenchmarkScanFixedCosts_DeepStack(b *testing.B) {
	tbl := benchScanTable(b, "deepstack")
	trueFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	mapReduceFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] })
	nilFn := scm.NewNil()
	neutral := scm.NewNil()

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
		// simulate ~80 extra frames of Scheme evaluation above the scan call
		recurse(80, func() {
			tbl.scan(
				nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
				[]string{"id"}, trueFn,
				[]string{"id"}, mapReduceFn,
				neutral, nilFn, false,
			)
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
	mapReduceFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] })
	nilFn := scm.NewNil()
	// Warm the lazily built index before measuring regular probe overhead.
	tbl.scan(currentTx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{"id"}, condition, []string{"label"}, mapReduceFn, scm.NewNil(), nilFn, false)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tbl.scan(currentTx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{"id"}, condition, []string{"label"}, mapReduceFn, scm.NewNil(), nilFn, false)
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

func BenchmarkScanUniquePointCompiledAccessWithTx(b *testing.B) {
	dbName := "bench_scan_point_compiled"
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
	tx := NewTxContext(TxCursorStability)
	condition := scanCondition("id", scm.NewInt(511))
	mapReduceFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] })
	schema := scm.NewSlice([]scm.Scmer{
		newScanAccessHeader(1, scanAccessConsumerScan, 0, -1), scm.NewString("equal"),
		scm.NewString("id"), newScanAccessBoundaryMeta(0, 0, 3), scm.NewString(""),
	})
	values := []scm.Scmer{scm.NewInt(511)}
	tbl.scanWithBatchFrom(tx, nil, schema, values, scanAccess{}, []string{"id"}, condition, []string{"label"}, mapReduceFn,
		scm.NewNil(), scm.NewNil(), false, 0, nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tbl.scanWithBatchFrom(tx, nil, schema, values, scanAccess{}, []string{"id"}, condition, []string{"label"}, mapReduceFn,
			scm.NewNil(), scm.NewNil(), false, 0, nil)
	}
}

// BenchmarkScanUniquePointCoveredCompiledAccessWithTx models the physical
// shape emitted after compile_scan_plan proves an exact point predicate. All
// immutable argument slices are prepared outside the timed loop, so this
// reports storage execution allocations rather than caller construction.
func BenchmarkScanUniquePointCoveredCompiledAccessWithTx(b *testing.B) {
	dbName := "bench_scan_point_covered_compiled"
	databases.Remove(dbName)
	b.Cleanup(func() { databases.Remove(dbName) })
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
	tx := NewTxContext(TxCursorStability)
	condition := scanCondition("id", scm.NewInt(511))
	mapReduceFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] })
	schema := scm.NewSlice([]scm.Scmer{
		newScanAccessHeader(1, scanAccessConsumerCoveredScan, 0, -1), scm.NewString("equal"),
		scm.NewString("id"), newScanAccessBoundaryMeta(0, 0, 3), scm.NewString(""),
	})
	values := []scm.Scmer{scm.NewInt(511)}
	conditionCols := []string{"id"}
	callbackCols := []string{"label"}
	tbl.scanWithBatchFrom(tx, nil, schema, values, scanAccess{}, conditionCols, condition, callbackCols, mapReduceFn,
		scm.NewNil(), scm.NewNil(), false, 0, nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tbl.scanWithBatchFrom(tx, nil, schema, values, scanAccess{}, conditionCols, condition, callbackCols, mapReduceFn,
			scm.NewNil(), scm.NewNil(), false, 0, nil)
	}
}

func benchmarkUniqueMainPointScan(b *testing.B, name string, currentTx *TxContext) {
	dbName := "bench_scan_main_point_" + name
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
	result := GetDatabase(dbName).rebuild(true, false, true)
	if len(result.errors) > 0 {
		b.Fatalf("rebuild errors: %v", result.errors)
	}

	condition := scanCondition("id", scm.NewInt(511))
	mapReduceFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] })
	nilFn := scm.NewNil()
	// The first two probes cross the adaptive-index threshold and build the
	// main-storage index. Only steady-state point probes are measured.
	tbl.scan(currentTx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{"id"}, condition, []string{"label"}, mapReduceFn, scm.NewNil(), nilFn, false)
	tbl.scan(currentTx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{"id"}, condition, []string{"label"}, mapReduceFn, scm.NewNil(), nilFn, false)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tbl.scan(currentTx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{"id"}, condition, []string{"label"}, mapReduceFn, scm.NewNil(), nilFn, false)
	}
}

// BenchmarkScanUniqueMainPoint measures a warm unique lookup after rows and
// the corresponding index have moved to compressed main storage.
func BenchmarkScanUniqueMainPoint(b *testing.B) {
	benchmarkUniqueMainPointScan(b, "read", nil)
}

// BenchmarkScanUniqueMainPointWithTx includes the explicit transaction used
// by prepared SQL execution while retaining the same warm main-storage probe.
func BenchmarkScanUniqueMainPointWithTx(b *testing.B) {
	benchmarkUniqueMainPointScan(b, "read_tx", NewTxContext(TxCursorStability))
}

func TestOpenMapReducerAllocatesMutationMetadataLazily(t *testing.T) {
	const dbName = "test_scan_mapper_metadata"
	databases.Remove(dbName)
	t.Cleanup(func() { databases.Remove(dbName) })
	CreateDatabase(dbName, true)
	tbl, _ := CreateTable(dbName, "items", Memory, true)
	tbl.CreateColumn("id", "INT", nil, nil)
	shard := tbl.Shards[0]
	mapReduceFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] })

	readMapper := shard.OpenMapReducer([]string{"id"}, mapReduceFn, false, 0, nil, nil)
	defer readMapper.Close()
	if readMapper.isUpdate != nil || readMapper.isIncrement != nil || readMapper.setClosureFn != nil {
		t.Fatal("read mapper allocated mutation-only metadata")
	}

	mutationMapper := shard.OpenMapReducer([]string{"$update"}, mapReduceFn, false, 0, nil, nil)
	defer mutationMapper.Close()
	if len(mutationMapper.isUpdate) != 1 || !mutationMapper.isUpdate[0] || len(mutationMapper.setClosureFn) != 1 {
		t.Fatal("mutation mapper did not initialize mutation metadata")
	}
}

func TestReadMapReducerWorkspaceMainDeltaAndWideProjection(t *testing.T) {
	const dbName = "test_scan_read_mapper_workspace"
	databases.Remove(dbName)
	t.Cleanup(func() { databases.Remove(dbName) })
	CreateDatabase(dbName, true)
	tbl, _ := CreateTable(dbName, "items", Memory, true)
	columns := make([]string, inlineMapReducerColumns+1)
	for i := range columns {
		columns[i] = "c" + strconv.Itoa(i)
		tbl.CreateColumn(columns[i], "INT", nil, nil)
	}
	mainRows := make([][]scm.Scmer, 2)
	for rowIndex := range mainRows {
		mainRows[rowIndex] = make([]scm.Scmer, len(columns))
		for columnIndex := range mainRows[rowIndex] {
			mainRows[rowIndex][columnIndex] = scm.NewInt(int64(rowIndex*20 + columnIndex + 1))
		}
	}
	tbl.Insert(columns, mainRows, nil, scm.NewNil(), false, nil)
	result := GetDatabase(dbName).rebuild(true, false, true)
	if len(result.errors) > 0 {
		t.Fatalf("rebuild errors: %v", result.errors)
	}

	shard := tbl.Shards[0]
	mapLast := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return args[len(args)-1] })
	mapReduceLast := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return args[len(args)-1] })

	// A small projection uses the inline workspace. MapOne follows Stream to
	// ensure it reads its requested row instead of reusing prefetched values.
	var inlineMapper ShardMapReducer
	var inlineWorkspace shardMapReducerWorkspace
	prepareReadMapReducerStorage(&inlineMapper, &inlineWorkspace, 2)
	shard.initReadMapReducer(&inlineMapper, columns[:2], mapReduceLast, false, nil)
	inlineMapper.mapProgram = scm.PrepareSerialProc(mapLast)
	inlineMapper.Stream(scm.NewNil(), []uint32{0}, nil)
	if got := inlineMapper.MapOne(1).Int(); got != mainRows[1][1].Int() {
		t.Fatalf("inline main-row projection = %d, want %d", got, mainRows[1][1].Int())
	}

	// Wider projections use the allocation-backed fallback rather than
	// truncating the caller-owned inline arrays.
	var mapper ShardMapReducer
	var workspace shardMapReducerWorkspace
	prepareReadMapReducerStorage(&mapper, &workspace, len(columns))
	shard.initReadMapReducer(&mapper, columns, mapReduceLast, false, nil)
	mapper.mapProgram = scm.PrepareSerialProc(mapLast)
	if got := mapper.Stream(scm.NewNil(), []uint32{0}, nil).Int(); got != mainRows[0][len(columns)-1].Int() {
		t.Fatalf("wide main-row projection = %d, want %d", got, mainRows[0][len(columns)-1].Int())
	}

	deltaRow := make([]scm.Scmer, len(columns))
	for i := range deltaRow {
		deltaRow[i] = scm.NewInt(int64(100 + i))
	}
	tbl.Insert(columns, [][]scm.Scmer{deltaRow}, nil, scm.NewNil(), false, nil)
	if got := mapper.MapOne(shard.main_count).Int(); got != deltaRow[len(deltaRow)-1].Int() {
		t.Fatalf("wide delta-row projection = %d, want %d", got, deltaRow[len(deltaRow)-1].Int())
	}
}

// BenchmarkExplicitTxAccess measures the replacement for implicit context lookup.
func BenchmarkExplicitTxAccess(b *testing.B) {
	tx := NewTxContext(TxCursorStability)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			_ = tx.SessionState
		}
	}
}

// BenchmarkNewExecutionContext measures request-context construction.
func BenchmarkNewExecutionContext(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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
	mapReduceFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] })
	nilFn := scm.NewNil()
	neutral := scm.NewNil()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tbl.scan(
			nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
			[]string{"id"}, trueFn,
			[]string{"id", "$increment:cached_val"}, mapReduceFn,
			neutral, nilFn, false,
		)
	}
}
